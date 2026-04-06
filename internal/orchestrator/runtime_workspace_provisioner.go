package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	"github.com/google/uuid"
)

type runtimeWorkspaceProvisioner struct {
	client     *ent.Client
	logger     *slog.Logger
	sshPool    *sshinfra.Pool
	transports *machinetransport.Resolver
	githubAuth githubauthservice.TokenResolver
	now        func() time.Time
}

func newRuntimeWorkspaceProvisioner(
	client *ent.Client,
	logger *slog.Logger,
	sshPool *sshinfra.Pool,
	now func() time.Time,
) *runtimeWorkspaceProvisioner {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return &runtimeWorkspaceProvisioner{
		client:     client,
		logger:     logger.With("component", "runtime-workspace-provisioner"),
		sshPool:    sshPool,
		transports: machinetransport.NewResolver(nil, sshPool),
		now:        now,
	}
}

func (p *runtimeWorkspaceProvisioner) prepareTicketWorkspace(
	ctx context.Context,
	runID uuid.UUID,
	launchContext runtimeLaunchContext,
	machine catalogdomain.Machine,
	remote bool,
) (workspaceinfra.Workspace, error) {
	request, repoPlans, err := buildWorkspaceRequest(launchContext, machine, remote)
	if err != nil {
		return workspaceinfra.Workspace{}, err
	}
	request, err = p.applyGitHubWorkspaceAuth(ctx, launchContext.project.ID, request)
	if err != nil {
		return workspaceinfra.Workspace{}, err
	}
	if err := p.ensureTicketRepoWorkspaceRecords(ctx, runID, launchContext.ticket.ID, request, repoPlans); err != nil {
		return workspaceinfra.Workspace{}, err
	}
	if err := p.setTicketRepoWorkspaceState(ctx, runID, repoPlans, entticketrepoworkspace.StateMaterializing, ""); err != nil {
		return workspaceinfra.Workspace{}, err
	}

	var workspaceItem workspaceinfra.Workspace
	resolved, transportErr := p.resolveRuntimeTransport(machine)
	if transportErr != nil {
		err = transportErr
	} else {
		workspaceExecutor := resolved.WorkspaceExecutor()
		if workspaceExecutor == nil {
			err = fmt.Errorf("%w: workspace preparation unavailable for machine %s", machinetransport.ErrTransportUnavailable, machine.Name)
		} else {
			workspaceItem, err = workspaceExecutor.PrepareWorkspace(ctx, machine, request)
		}
	}
	if err != nil {
		if updateErr := p.setTicketRepoWorkspaceState(ctx, runID, repoPlans, entticketrepoworkspace.StateFailed, err.Error()); updateErr != nil {
			return workspaceinfra.Workspace{}, fmt.Errorf("prepare workspace failed: %w (ticket repo workspace state update failed: %v)", err, updateErr)
		}
		return workspaceinfra.Workspace{}, err
	}
	if err := p.materializeWorkspaceInstructionHub(
		ctx,
		machine,
		resolved,
		workspaceItem,
		string(launchContext.agent.Edges.Provider.AdapterType),
		remote,
	); err != nil {
		if updateErr := p.setTicketRepoWorkspaceState(ctx, runID, repoPlans, entticketrepoworkspace.StateFailed, err.Error()); updateErr != nil {
			return workspaceinfra.Workspace{}, fmt.Errorf("materialize workspace instruction hub failed: %w (ticket repo workspace state update failed: %v)", err, updateErr)
		}
		return workspaceinfra.Workspace{}, err
	}
	repoPlans = repoPlansWithPreparedHeads(repoPlans, workspaceItem.Repos)
	if err := p.markTicketRepoWorkspacesReady(ctx, runID, repoPlans); err != nil {
		return workspaceinfra.Workspace{}, err
	}

	return workspaceItem, nil
}

func (p *runtimeWorkspaceProvisioner) applyGitHubWorkspaceAuth(
	ctx context.Context,
	projectID uuid.UUID,
	request workspaceinfra.SetupRequest,
) (workspaceinfra.SetupRequest, error) {
	return githubauthservice.ApplyWorkspaceAuth(ctx, p.githubAuth, projectID, request)
}

func (p *runtimeWorkspaceProvisioner) ensureTicketRepoWorkspaceRecords(
	ctx context.Context,
	runID uuid.UUID,
	ticketID uuid.UUID,
	request workspaceinfra.SetupRequest,
	repoPlans []repoWorkspacePlan,
) error {
	if p == nil || p.client == nil || len(repoPlans) == 0 {
		return nil
	}

	workspaceRoot, err := workspaceinfra.TicketWorkspacePath(
		request.WorkspaceRoot,
		request.OrganizationSlug,
		request.ProjectSlug,
		request.TicketIdentifier,
	)
	if err != nil {
		return fmt.Errorf("derive ticket workspace root for runtime state: %w", err)
	}

	for _, plan := range repoPlans {
		repoPath := workspaceinfra.RepoPath(workspaceRoot, plan.WorkspaceDir, plan.RepoName)
		existing, err := p.client.TicketRepoWorkspace.Query().
			Where(
				entticketrepoworkspace.AgentRunIDEQ(runID),
				entticketrepoworkspace.RepoIDEQ(plan.RepoID),
			).
			Only(ctx)
		switch {
		case ent.IsNotFound(err):
			create := p.client.TicketRepoWorkspace.Create().
				SetTicketID(ticketID).
				SetAgentRunID(runID).
				SetRepoID(plan.RepoID).
				SetWorkspaceRoot(workspaceRoot).
				SetRepoPath(repoPath).
				SetBranchName(plan.BranchName).
				SetState(entticketrepoworkspace.StatePlanned)
			if plan.HeadCommit != "" {
				create.SetHeadCommit(plan.HeadCommit)
			}
			if _, err := create.Save(ctx); err != nil {
				return fmt.Errorf("create ticket repo workspace for repo %s: %w", plan.RepoName, err)
			}
		case err != nil:
			return fmt.Errorf("load ticket repo workspace for repo %s: %w", plan.RepoName, err)
		default:
			update := p.client.TicketRepoWorkspace.UpdateOneID(existing.ID).
				SetWorkspaceRoot(workspaceRoot).
				SetRepoPath(repoPath).
				SetBranchName(plan.BranchName).
				SetState(entticketrepoworkspace.StatePlanned).
				ClearLastError().
				ClearPreparedAt().
				ClearCleanedAt()
			if plan.HeadCommit != "" {
				update.SetHeadCommit(plan.HeadCommit)
			} else {
				update.ClearHeadCommit()
			}
			if _, err := update.Save(ctx); err != nil {
				return fmt.Errorf("reset ticket repo workspace for repo %s: %w", plan.RepoName, err)
			}
		}
	}

	return nil
}

func (p *runtimeWorkspaceProvisioner) setTicketRepoWorkspaceState(
	ctx context.Context,
	runID uuid.UUID,
	repoPlans []repoWorkspacePlan,
	state entticketrepoworkspace.State,
	lastError string,
) error {
	if p == nil || len(repoPlans) == 0 {
		return nil
	}

	trimmedError := strings.TrimSpace(lastError)
	for _, plan := range repoPlans {
		update := p.client.TicketRepoWorkspace.Update().
			Where(
				entticketrepoworkspace.AgentRunIDEQ(runID),
				entticketrepoworkspace.RepoIDEQ(plan.RepoID),
			).
			SetState(state)
		if trimmedError != "" {
			update.SetLastError(trimmedError)
		} else {
			update.ClearLastError()
		}
		if _, err := update.Save(ctx); err != nil {
			return fmt.Errorf("update ticket repo workspace %s state to %s: %w", plan.RepoName, state, err)
		}
	}

	return nil
}

func (p *runtimeWorkspaceProvisioner) markTicketRepoWorkspacesReady(
	ctx context.Context,
	runID uuid.UUID,
	repoPlans []repoWorkspacePlan,
) error {
	if p == nil || len(repoPlans) == 0 {
		return nil
	}

	preparedAt := time.Now().UTC()
	for _, plan := range repoPlans {
		update := p.client.TicketRepoWorkspace.Update().
			Where(
				entticketrepoworkspace.AgentRunIDEQ(runID),
				entticketrepoworkspace.RepoIDEQ(plan.RepoID),
			).
			SetState(entticketrepoworkspace.StateReady).
			SetPreparedAt(preparedAt).
			ClearLastError()
		if plan.HeadCommit != "" {
			update.SetHeadCommit(plan.HeadCommit)
		}
		if _, err := update.Save(ctx); err != nil {
			return fmt.Errorf("mark ticket repo workspace %s ready: %w", plan.RepoName, err)
		}
	}

	return nil
}

func (p *runtimeWorkspaceProvisioner) cleanupRunWorkspacesBestEffort(ctx context.Context, runID uuid.UUID, reason string) {
	if p == nil || runID == uuid.Nil {
		return
	}
	if err := p.cleanupRunWorkspaces(ctx, runID); err != nil {
		p.logger.Warn("cleanup ticket workspaces", "run_id", runID, "reason", strings.TrimSpace(reason), "error", err)
	}
}

func (p *runtimeWorkspaceProvisioner) cleanupRunWorkspaces(ctx context.Context, runID uuid.UUID) error {
	if p == nil || p.client == nil || runID == uuid.Nil {
		return nil
	}

	machine, remote, workspaceRoot, err := p.resolveRunWorkspaceCleanupTarget(ctx, runID)
	if err != nil {
		return err
	}

	workspaces, err := p.client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.AgentRunIDEQ(runID)).
		Order(ent.Asc(entticketrepoworkspace.FieldRepoPath)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("load ticket repo workspaces for cleanup: %w", err)
	}
	trackCleanupState := false
	if len(workspaces) > 0 {
		for _, item := range workspaces {
			if trimmed := strings.TrimSpace(item.WorkspaceRoot); trimmed != workspaceRoot {
				_ = p.markRunWorkspaceCleanupFailed(ctx, runID, fmt.Errorf("run %s spans multiple workspace roots (%s, %s)", runID, workspaceRoot, trimmed))
				return fmt.Errorf("run %s spans multiple workspace roots (%s, %s)", runID, workspaceRoot, trimmed)
			}
			if item.PreparedAt != nil || item.State == entticketrepoworkspace.StateReady {
				trackCleanupState = true
			}
		}
		if trackCleanupState {
			if _, err := p.client.TicketRepoWorkspace.Update().
				Where(entticketrepoworkspace.AgentRunIDEQ(runID)).
				SetState(entticketrepoworkspace.StateCleaning).
				ClearLastError().
				Save(ctx); err != nil {
				return fmt.Errorf("mark ticket repo workspaces cleaning: %w", err)
			}
		}
	}

	if err := p.removeWorkspaceRoot(ctx, machine, remote, workspaceRoot); err != nil {
		_ = p.markRunWorkspaceCleanupFailed(ctx, runID, err)
		return err
	}

	if len(workspaces) == 0 || !trackCleanupState {
		return nil
	}

	cleanedAt := p.now().UTC()
	if _, err := p.client.TicketRepoWorkspace.Update().
		Where(entticketrepoworkspace.AgentRunIDEQ(runID)).
		SetState(entticketrepoworkspace.StateCleaned).
		SetCleanedAt(cleanedAt).
		ClearLastError().
		Save(ctx); err != nil {
		return fmt.Errorf("mark ticket repo workspaces cleaned: %w", err)
	}
	return nil
}

func (p *runtimeWorkspaceProvisioner) resolveRunWorkspaceCleanupTarget(ctx context.Context, runID uuid.UUID) (catalogdomain.Machine, bool, string, error) {
	if p == nil || p.client == nil {
		return catalogdomain.Machine{}, false, "", fmt.Errorf("runtime workspace provisioner unavailable")
	}
	runItem, err := p.client.AgentRun.Query().
		Where(entagentrun.IDEQ(runID)).
		WithAgent(func(query *ent.AgentQuery) {
			query.WithProject(func(projectQuery *ent.ProjectQuery) {
				projectQuery.WithOrganization()
			})
		}).
		WithProvider().
		WithTicket().
		Only(ctx)
	if err != nil {
		return catalogdomain.Machine{}, false, "", fmt.Errorf("load run %s for workspace cleanup: %w", runID, err)
	}
	if runItem.Edges.Agent == nil || runItem.Edges.Agent.Edges.Project == nil || runItem.Edges.Agent.Edges.Project.Edges.Organization == nil {
		return catalogdomain.Machine{}, false, "", fmt.Errorf("run %s missing project context for workspace cleanup", runID)
	}
	if runItem.Edges.Provider == nil {
		return catalogdomain.Machine{}, false, "", fmt.Errorf("run %s missing provider for workspace cleanup", runID)
	}
	if runItem.Edges.Ticket == nil {
		return catalogdomain.Machine{}, false, "", fmt.Errorf("run %s missing ticket for workspace cleanup", runID)
	}

	machineItem, err := p.client.Machine.Get(ctx, runItem.Edges.Provider.MachineID)
	if err != nil {
		return catalogdomain.Machine{}, false, "", fmt.Errorf("load machine %s for workspace cleanup: %w", runItem.Edges.Provider.MachineID, err)
	}
	machine := mapRuntimeMachine(machineItem)
	remote := machine.Host != catalogdomain.LocalMachineHost

	workspaceRootBase, err := resolveWorkspaceRoot(machine, remote)
	if err != nil {
		return catalogdomain.Machine{}, false, "", err
	}
	workspaceRoot, err := workspaceinfra.TicketWorkspacePath(
		workspaceRootBase,
		runItem.Edges.Agent.Edges.Project.Edges.Organization.Slug,
		runItem.Edges.Agent.Edges.Project.Slug,
		runItem.Edges.Ticket.Identifier,
	)
	if err != nil {
		return catalogdomain.Machine{}, false, "", fmt.Errorf("derive workspace path for cleanup: %w", err)
	}
	return machine, remote, workspaceRoot, nil
}

func (p *runtimeWorkspaceProvisioner) markRunWorkspaceCleanupFailed(ctx context.Context, runID uuid.UUID, cleanupErr error) error {
	if p == nil || p.client == nil || runID == uuid.Nil || cleanupErr == nil {
		return nil
	}
	update := p.client.TicketRepoWorkspace.Update().
		Where(entticketrepoworkspace.AgentRunIDEQ(runID)).
		SetState(entticketrepoworkspace.StateFailed)

	existingError, err := p.client.TicketRepoWorkspace.Query().
		Where(
			entticketrepoworkspace.AgentRunIDEQ(runID),
			entticketrepoworkspace.LastErrorNEQ(""),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("inspect ticket repo workspace cleanup failure state: %w", err)
	}
	if !existingError {
		update.SetLastError(strings.TrimSpace(cleanupErr.Error()))
	}

	_, err = update.Save(ctx)
	if err != nil {
		return fmt.Errorf("record ticket repo workspace cleanup failure: %w", err)
	}
	return nil
}

func (p *runtimeWorkspaceProvisioner) removeWorkspaceRoot(ctx context.Context, machine catalogdomain.Machine, remote bool, workspaceRoot string) error {
	trimmedRoot := strings.TrimSpace(workspaceRoot)
	if trimmedRoot == "" {
		return fmt.Errorf("workspace root must not be empty")
	}

	if !remote {
		if err := os.RemoveAll(trimmedRoot); err != nil {
			return fmt.Errorf("remove local workspace %s: %w", trimmedRoot, err)
		}
		return nil
	}
	resolved, err := p.resolveRuntimeTransport(machine)
	if err != nil {
		return err
	}

	commandSessionExecutor := resolved.CommandSessionExecutor()
	if commandSessionExecutor == nil {
		return fmt.Errorf("%w: remote command session unavailable for machine %s", machinetransport.ErrTransportUnavailable, machine.Name)
	}
	session, err := commandSessionExecutor.OpenCommandSession(ctx, machine)
	if err != nil {
		return fmt.Errorf("open remote command session for machine %s: %w", machine.Name, err)
	}
	defer func() {
		_ = session.Close()
	}()

	command := "set -eu\nrm -rf " + sshinfra.ShellQuote(trimmedRoot)
	if output, err := session.CombinedOutput(command); err != nil {
		return fmt.Errorf("remove remote workspace %s: %w: %s", trimmedRoot, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (p *runtimeWorkspaceProvisioner) resolveRuntimeTransport(machine catalogdomain.Machine) (machinetransport.ResolvedTransport, error) {
	if p == nil || p.transports == nil {
		return machinetransport.ResolvedTransport{}, fmt.Errorf("machine transport resolver unavailable for machine %s", machine.Name)
	}
	return p.transports.ResolveRuntime(machine)
}
