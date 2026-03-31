package ticket

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/google/uuid"
)

type ticketHookSSHPool interface {
	Get(ctx context.Context, machine catalogdomain.Machine) (sshinfra.Client, error)
}

type ticketHookAgentPlatform interface {
	IssueToken(ctx context.Context, input agentplatform.IssueInput) (agentplatform.IssuedToken, error)
}

type RunLifecycleHookInput struct {
	TicketID   uuid.UUID
	RunID      uuid.UUID
	HookName   infrahook.TicketHookName
	WorkflowID *uuid.UUID
	Blocking   bool
}

func (s *Service) ConfigureSSHPool(pool ticketHookSSHPool) {
	if s == nil {
		return
	}
	s.sshPool = pool
}

func (s *Service) ConfigurePlatformEnvironment(apiURL string, agentPlatform ticketHookAgentPlatform) {
	if s == nil {
		return
	}
	s.platformAPIURL = strings.TrimSpace(apiURL)
	s.agentPlatform = agentPlatform
}

func (s *Service) RunLifecycleHook(ctx context.Context, input RunLifecycleHookInput) error {
	if s == nil || s.client == nil {
		return ErrUnavailable
	}
	if input.TicketID == uuid.Nil {
		return fmt.Errorf("ticket hook ticket id must not be empty")
	}
	if input.RunID == uuid.Nil {
		return fmt.Errorf("ticket hook run id must not be empty")
	}

	runtime, err := s.loadHookRuntime(ctx, input)
	if err != nil {
		return err
	}
	if len(runtime.definitions) == 0 {
		return nil
	}

	results, err := runtime.executor.RunAll(ctx, input.HookName, runtime.definitions, runtime.env)
	s.logHookResults(input.HookName, runtime.ticketID, input.RunID, results, err)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) RunLifecycleHookBestEffort(ctx context.Context, input RunLifecycleHookInput) {
	if err := s.RunLifecycleHook(ctx, input); err != nil {
		logger := slog.Default()
		if s != nil && s.logger != nil {
			logger = s.logger
		}
		logger.Warn(
			"ticket lifecycle hook failed",
			"hook_name", input.HookName,
			"ticket_id", input.TicketID,
			"run_id", input.RunID,
			"error", err,
		)
	}
}

type loadedTicketHookRuntime struct {
	ticketID    uuid.UUID
	definitions []infrahook.Definition
	executor    infrahook.Executor
	env         infrahook.Env
}

func (s *Service) loadHookRuntime(ctx context.Context, input RunLifecycleHookInput) (loadedTicketHookRuntime, error) {
	runItem, err := s.client.AgentRun.Query().
		Where(entagentrun.IDEQ(input.RunID)).
		WithAgent(func(query *ent.AgentQuery) {
			query.WithProvider()
		}).
		Only(ctx)
	if err != nil {
		return loadedTicketHookRuntime{}, fmt.Errorf("load ticket hook run %s: %w", input.RunID, err)
	}
	if runItem.Edges.Agent == nil {
		return loadedTicketHookRuntime{}, fmt.Errorf("ticket hook run %s is missing agent", input.RunID)
	}
	if runItem.Edges.Agent.Edges.Provider == nil {
		return loadedTicketHookRuntime{}, fmt.Errorf("ticket hook run %s agent is missing provider", input.RunID)
	}

	ticketItem, err := s.client.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		return loadedTicketHookRuntime{}, s.mapTicketReadError("load ticket for lifecycle hook", err)
	}

	workflowID := ticketItem.WorkflowID
	if input.WorkflowID != nil {
		workflowID = input.WorkflowID
	}
	if workflowID == nil {
		return loadedTicketHookRuntime{ticketID: ticketItem.ID}, nil
	}

	workflowItem, err := s.client.Workflow.Query().
		Where(entworkflow.IDEQ(*workflowID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return loadedTicketHookRuntime{ticketID: ticketItem.ID}, nil
		}
		return loadedTicketHookRuntime{}, fmt.Errorf("load workflow %s for lifecycle hook: %w", *workflowID, err)
	}

	parsedHooks, err := infrahook.ParseTicketHooks(workflowItem.Hooks)
	if err != nil {
		return loadedTicketHookRuntime{}, fmt.Errorf("parse ticket hooks for workflow %s: %w", workflowItem.ID, err)
	}
	definitions := selectTicketHookDefinitions(parsedHooks, input.HookName)
	if len(definitions) == 0 {
		return loadedTicketHookRuntime{ticketID: ticketItem.ID}, nil
	}

	workspaces, err := s.client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.AgentRunIDEQ(input.RunID)).
		Order(ent.Asc(entticketrepoworkspace.FieldRepoPath)).
		WithRepo(func(query *ent.ProjectRepoQuery) {
			query.Order(entprojectrepo.ByName())
		}).
		All(ctx)
	if err != nil {
		return loadedTicketHookRuntime{}, fmt.Errorf("load ticket repo workspaces for run %s: %w", input.RunID, err)
	}
	if len(workspaces) == 0 {
		return loadedTicketHookRuntime{}, fmt.Errorf("ticket hook workspace is unavailable for run %s", input.RunID)
	}

	machineItem, err := s.client.Machine.Get(ctx, runItem.Edges.Agent.Edges.Provider.MachineID)
	if err != nil {
		return loadedTicketHookRuntime{}, fmt.Errorf("load machine for ticket hook run %s: %w", input.RunID, err)
	}
	machine := mapTicketHookMachine(machineItem)
	remote := machine.Host != catalogdomain.LocalMachineHost

	repos := make([]infrahook.Repo, 0, len(workspaces))
	for _, workspace := range workspaces {
		repoName := strings.TrimSpace(workspace.RepoPath)
		if workspace.Edges.Repo != nil && strings.TrimSpace(workspace.Edges.Repo.Name) != "" {
			repoName = strings.TrimSpace(workspace.Edges.Repo.Name)
		}
		repos = append(repos, infrahook.Repo{
			Name: repoName,
			Path: strings.TrimSpace(workspace.RepoPath),
		})
	}

	env := infrahook.Env{
		TicketID:         ticketItem.ID,
		ProjectID:        ticketItem.ProjectID,
		TicketIdentifier: ticketItem.Identifier,
		Workspace:        strings.TrimSpace(workspaces[0].WorkspaceRoot),
		Repos:            repos,
		AgentName:        runItem.Edges.Agent.Name,
		WorkflowType:     string(workflowItem.Type),
		Attempt:          ticketItem.AttemptCount + 1,
		APIURL:           s.platformAPIURL,
	}
	if s.agentPlatform != nil {
		issued, issueErr := s.agentPlatform.IssueToken(ctx, agentplatform.IssueInput{
			AgentID:   runItem.AgentID,
			ProjectID: ticketItem.ProjectID,
			TicketID:  ticketItem.ID,
		})
		if issueErr != nil {
			return loadedTicketHookRuntime{}, fmt.Errorf("issue ticket hook agent token: %w", issueErr)
		}
		env.AgentToken = issued.Token
	}

	executor, err := s.ticketHookExecutor(machine, remote)
	if err != nil {
		return loadedTicketHookRuntime{}, err
	}

	return loadedTicketHookRuntime{
		ticketID:    ticketItem.ID,
		definitions: definitions,
		executor:    executor,
		env:         env,
	}, nil
}

func (s *Service) ticketHookExecutor(machine catalogdomain.Machine, remote bool) (infrahook.Executor, error) {
	if !remote {
		return infrahook.NewShellExecutor(), nil
	}
	if s.sshPool == nil {
		return nil, fmt.Errorf("ticket hook ssh pool unavailable for machine %s", machine.Name)
	}
	return infrahook.NewRemoteShellExecutor(s.sshPool, machine), nil
}

func (s *Service) logHookResults(
	hookName infrahook.TicketHookName,
	ticketID uuid.UUID,
	runID uuid.UUID,
	results []infrahook.Result,
	runErr error,
) {
	logger := slog.Default()
	if s != nil && s.logger != nil {
		logger = s.logger
	}

	for _, result := range results {
		attrs := []any{
			"hook_name", hookName,
			"ticket_id", ticketID,
			"run_id", runID,
			"command", result.Command,
			"policy", result.Policy,
			"outcome", result.Outcome,
			"duration", result.Duration,
			"workdir", result.WorkingDirectory,
		}
		if result.ExitCode != nil {
			attrs = append(attrs, "exit_code", *result.ExitCode)
		}
		if strings.TrimSpace(result.Stdout) != "" {
			attrs = append(attrs, "stdout", result.Stdout)
		}
		if strings.TrimSpace(result.Stderr) != "" {
			attrs = append(attrs, "stderr", result.Stderr)
		}
		if strings.TrimSpace(result.Error) != "" {
			attrs = append(attrs, "error", result.Error)
		}

		switch result.Outcome {
		case infrahook.OutcomePass:
			logger.Info("ticket lifecycle hook succeeded", attrs...)
		default:
			logger.Warn("ticket lifecycle hook finished with error", attrs...)
		}
	}
	if runErr != nil && len(results) == 0 {
		logger.Warn(
			"ticket lifecycle hook failed before command execution",
			"hook_name", hookName,
			"ticket_id", ticketID,
			"run_id", runID,
			"error", runErr,
		)
	}
}

func selectTicketHookDefinitions(hooks infrahook.TicketHooks, hookName infrahook.TicketHookName) []infrahook.Definition {
	switch hookName {
	case infrahook.TicketHookOnClaim:
		return hooks.OnClaim
	case infrahook.TicketHookOnStart:
		return hooks.OnStart
	case infrahook.TicketHookOnComplete:
		return hooks.OnComplete
	case infrahook.TicketHookOnDone:
		return hooks.OnDone
	case infrahook.TicketHookOnError:
		return hooks.OnError
	case infrahook.TicketHookOnCancel:
		return hooks.OnCancel
	default:
		return nil
	}
}

func mapTicketHookMachine(item *ent.Machine) catalogdomain.Machine {
	if item == nil {
		return catalogdomain.Machine{}
	}

	return catalogdomain.Machine{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		Name:           item.Name,
		Host:           item.Host,
		Port:           item.Port,
		SSHUser:        cloneOptionalText(item.SSHUser),
		SSHKeyPath:     cloneOptionalText(item.SSHKeyPath),
		Status:         catalogdomain.MachineStatus(item.Status),
		WorkspaceRoot:  cloneOptionalText(item.WorkspaceRoot),
		MirrorRoot:     cloneOptionalText(item.MirrorRoot),
		AgentCLIPath:   cloneOptionalText(item.AgentCliPath),
		EnvVars:        slices.Clone(item.EnvVars),
		Resources:      cloneMap(item.Resources),
	}
}

func cloneOptionalText(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
