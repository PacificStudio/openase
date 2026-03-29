package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketreposcope "github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

const (
	defaultLaunchTimeout           = 30 * time.Second
	defaultLaunchCleanupTimeout    = 5 * time.Second
	defaultLifecyclePublishTimeout = 2 * time.Second
)

type RuntimeLauncher struct {
	client         *ent.Client
	logger         *slog.Logger
	events         provider.EventProvider
	processManager provider.AgentCLIProcessManager
	sshPool        *sshinfra.Pool
	workflow       *workflowservice.Service
	agentPlatform  runtimeAgentPlatform
	platformAPIURL string
	githubAuth     githubauthservice.TokenResolver
	now            func() time.Time
	launchTimeout  time.Duration
	eventTimeout   time.Duration

	sessionsMu sync.Mutex
	sessions   map[uuid.UUID]*codex.Session

	launchesMu sync.Mutex
	launches   map[uuid.UUID]struct{}

	executionsMu sync.Mutex
	executions   map[uuid.UUID]struct{}

	tickets *ticketservice.Service
}

type runtimeAgentPlatform interface {
	IssueToken(ctx context.Context, input agentplatform.IssueInput) (agentplatform.IssuedToken, error)
}

type runtimeAssignment struct {
	ticket *ent.Ticket
	agent  *ent.Agent
	run    *ent.AgentRun
}

func NewRuntimeLauncher(
	client *ent.Client,
	logger *slog.Logger,
	events provider.EventProvider,
	processManager provider.AgentCLIProcessManager,
	sshPool *sshinfra.Pool,
	workflow *workflowservice.Service,
) *RuntimeLauncher {
	if logger == nil {
		logger = slog.Default()
	}

	return &RuntimeLauncher{
		client:         client,
		logger:         logger.With("component", "runtime-launcher"),
		events:         events,
		processManager: processManager,
		sshPool:        sshPool,
		workflow:       workflow,
		now:            time.Now,
		launchTimeout:  defaultLaunchTimeout,
		eventTimeout:   defaultLifecyclePublishTimeout,
		sessions:       map[uuid.UUID]*codex.Session{},
		launches:       map[uuid.UUID]struct{}{},
		executions:     map[uuid.UUID]struct{}{},
		tickets:        ticketservice.NewService(client),
	}
}

func (l *RuntimeLauncher) ConfigurePlatformEnvironment(apiURL string, agentPlatform runtimeAgentPlatform) {
	if l == nil {
		return
	}

	l.platformAPIURL = strings.TrimSpace(apiURL)
	l.agentPlatform = agentPlatform
}

func (l *RuntimeLauncher) ConfigureGitHubCredentials(resolver githubauthservice.TokenResolver) {
	if l == nil {
		return
	}
	l.githubAuth = resolver
}

func (l *RuntimeLauncher) RunTick(ctx context.Context) error {
	if l == nil || l.client == nil {
		return fmt.Errorf("runtime launcher unavailable")
	}
	if l.processManager == nil {
		return fmt.Errorf("runtime launcher process manager unavailable")
	}

	if err := l.reconcilePauseRequests(ctx); err != nil {
		return err
	}
	if err := l.refreshHeartbeats(ctx); err != nil {
		return err
	}
	if err := l.startReadyExecutions(ctx); err != nil {
		return err
	}

	assignments, err := l.listAssignments(ctx,
		entticket.CurrentRunIDNotNil(),
		entticket.HasCurrentRunWith(
			entagentrun.HasAgentWith(
				entagent.RuntimeControlStateEQ(entagent.RuntimeControlStateActive),
			),
			entagentrun.StatusIn(entagentrun.StatusLaunching, entagentrun.StatusTerminated),
		),
	)
	if err != nil {
		return fmt.Errorf("list current runs awaiting launch: %w", err)
	}

	var launchWG sync.WaitGroup
	for _, assignment := range assignments {
		if assignment.run == nil || !l.beginLaunch(assignment.run.ID) {
			continue
		}
		launchWG.Add(1)
		go func(assignment runtimeAssignment) {
			defer launchWG.Done()
			l.runLaunch(ctx, assignment)
		}(assignment)
	}
	launchWG.Wait()

	return nil
}

func (l *RuntimeLauncher) Close(ctx context.Context) error {
	if l == nil {
		return nil
	}

	sessions := l.drainSessions()
	for runID, session := range sessions {
		if session != nil {
			stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			_ = session.Stop(stopCtx)
			cancel()
		}

		now := l.now().UTC()
		assignment, err := l.loadAssignmentByRun(ctx, runID)
		if err != nil {
			l.logger.Warn("load run assignment during close", "run_id", runID, "error", err)
			continue
		}
		if assignment.run != nil {
			tx, err := l.client.Tx(ctx)
			if err != nil {
				l.logger.Warn("start graceful shutdown tx", "agent_id", assignment.agent.ID, "run_id", assignment.run.ID, "error", err)
				continue
			}

			if _, err := clearRuntimeState(
				tx.AgentRun.Update().
					Where(
						entagentrun.IDEQ(assignment.run.ID),
						entagentrun.StatusIn(entagentrun.StatusLaunching, entagentrun.StatusReady, entagentrun.StatusExecuting),
					).
					SetStatus(entagentrun.StatusTerminated),
			).Save(ctx); err != nil {
				rollback(tx)
				l.logger.Warn("mark agent run terminated", "agent_id", assignment.agent.ID, "run_id", assignment.run.ID, "error", err)
				continue
			}
			if assignment.ticket != nil && assignment.ticket.CurrentRunID != nil {
				if _, err := tx.Ticket.Update().
					Where(
						entticket.IDEQ(assignment.ticket.ID),
						entticket.CurrentRunIDEQ(assignment.run.ID),
					).
					ClearCurrentRunID().
					Save(ctx); err != nil {
					rollback(tx)
					l.logger.Warn("clear ticket current run during close", "agent_id", assignment.agent.ID, "ticket_id", assignment.ticket.ID, "run_id", assignment.run.ID, "error", err)
					continue
				}
			}
			if err := tx.Commit(); err != nil {
				l.logger.Warn("commit graceful shutdown release", "agent_id", assignment.agent.ID, "run_id", assignment.run.ID, "error", err)
				continue
			}
		}

		if assignment.agent == nil {
			continue
		}
		agentState, err := loadAgentLifecycleState(ctx, l.client, assignment.agent.ID, &runID)
		if err != nil {
			l.logger.Warn("reload terminated agent", "agent_id", assignment.agent.ID, "run_id", runID, "error", err)
			continue
		}
		l.publishLifecycleEvent(
			ctx,
			agentTerminatedType,
			agentState,
			lifecycleMessage(agentTerminatedType, agentState.agent.Name),
			runtimeEventMetadataForState(agentState),
			now,
		)
	}

	return nil
}

func (l *RuntimeLauncher) startLaunch(ctx context.Context, assignment runtimeAssignment) {
	if l == nil || assignment.run == nil {
		return
	}
	if !l.beginLaunch(assignment.run.ID) {
		return
	}

	//nolint:gosec // Launching runs must not block later assignments in the orchestrator tick.
	go l.runLaunch(ctx, assignment)
}

func (l *RuntimeLauncher) runLaunch(ctx context.Context, assignment runtimeAssignment) {
	defer l.finishLaunch(assignment.run.ID)

	launchCtx, cancel := l.launchContext(ctx, l.launchTimeout)
	defer cancel()

	err := l.launchAgent(launchCtx, assignment)
	if err == nil {
		return
	}

	l.logger.Error("launch current run", "agent_id", assignment.agent.ID, "run_id", assignment.run.ID, "error", err)
	if assignment.agent == nil || assignment.run == nil || assignment.ticket == nil {
		return
	}

	failureCtx, failureCancel := l.launchContext(ctx, defaultLaunchCleanupTimeout)
	defer failureCancel()
	if markErr := l.markLaunchFailed(failureCtx, assignment.agent.ID, assignment.ticket.ID, assignment.run.ID, err); markErr != nil {
		l.logger.Error("mark launch failed", "agent_id", assignment.agent.ID, "ticket_id", assignment.ticket.ID, "run_id", assignment.run.ID, "error", markErr)
	}
}

func (l *RuntimeLauncher) launchAgent(ctx context.Context, assignment runtimeAssignment) error {
	if assignment.agent == nil || assignment.run == nil || assignment.ticket == nil {
		return nil
	}

	now := l.now().UTC()
	launchingCount, err := l.client.AgentRun.Update().
		Where(
			entagentrun.IDEQ(assignment.run.ID),
			entagentrun.StatusIn(entagentrun.StatusLaunching, entagentrun.StatusTerminated),
		).
		SetStatus(entagentrun.StatusLaunching).
		SetLastError("").
		ClearSessionID().
		ClearRuntimeStartedAt().
		ClearLastHeartbeatAt().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark run %s launching: %w", assignment.run.ID, err)
	}
	if launchingCount == 0 {
		return nil
	}

	launchingAgent, err := loadAgentLifecycleState(ctx, l.client, assignment.agent.ID, &assignment.run.ID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentLaunchingType,
		launchingAgent,
		lifecycleMessage(agentLaunchingType, launchingAgent.agent.Name),
		runtimeEventMetadataForState(launchingAgent),
		now,
	)

	session, launchErr := l.startCodexSession(ctx, assignment)
	if launchErr != nil {
		return launchErr
	}

	l.storeSession(assignment.run.ID, session)

	readyAt := l.now().UTC()
	readyCount, err := l.client.AgentRun.Update().
		Where(
			entagentrun.IDEQ(assignment.run.ID),
			entagentrun.StatusEQ(entagentrun.StatusLaunching),
		).
		SetSessionID(session.ThreadID()).
		SetStatus(entagentrun.StatusReady).
		SetRuntimeStartedAt(readyAt).
		SetLastHeartbeatAt(readyAt).
		SetLastError("").
		Save(ctx)
	if err != nil {
		l.deleteSession(assignment.run.ID)
		stopSession(context.Background(), session)
		return fmt.Errorf("mark run %s ready: %w", assignment.run.ID, err)
	}
	if readyCount == 0 {
		l.deleteSession(assignment.run.ID)
		stopSession(context.Background(), session)
		return nil
	}

	readyAgent, err := loadAgentLifecycleState(ctx, l.client, assignment.agent.ID, &assignment.run.ID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentReadyType,
		readyAgent,
		lifecycleMessage(agentReadyType, readyAgent.agent.Name),
		runtimeEventMetadataForState(readyAgent),
		readyAt,
	)
	l.publishLifecycleEvent(
		ctx,
		agentHeartbeatType,
		readyAgent,
		lifecycleMessage(agentHeartbeatType, readyAgent.agent.Name),
		runtimeEventMetadataForState(readyAgent),
		readyAt,
	)

	return nil
}

func (l *RuntimeLauncher) markLaunchFailed(ctx context.Context, agentID uuid.UUID, ticketID uuid.UUID, runID uuid.UUID, launchErr error) error {
	now := l.now().UTC()
	count, err := l.client.AgentRun.Update().
		Where(
			entagentrun.IDEQ(runID),
			entagentrun.StatusEQ(entagentrun.StatusLaunching),
		).
		SetStatus(entagentrun.StatusErrored).
		SetLastError(strings.TrimSpace(launchErr.Error())).
		ClearSessionID().
		ClearRuntimeStartedAt().
		ClearLastHeartbeatAt().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark run %s failed: %w", runID, err)
	}
	if count == 0 {
		return nil
	}

	retrySvc := NewRetryService(l.client, l.logger)
	retrySvc.now = l.now
	if _, err := retrySvc.MarkAttemptFailed(ctx, ticketID); err != nil {
		return fmt.Errorf("release failed launch claim for ticket %s: %w", ticketID, err)
	}

	failedAgent, err := loadAgentLifecycleState(ctx, l.client, agentID, &runID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentFailedType,
		failedAgent,
		lifecycleMessage(agentFailedType, failedAgent.agent.Name),
		runtimeEventMetadataForState(failedAgent),
		now,
	)
	return nil
}

func (l *RuntimeLauncher) reconcilePauseRequests(ctx context.Context) error {
	pausedAssignments, err := l.listAssignments(ctx,
		entticket.CurrentRunIDNotNil(),
		entticket.HasCurrentRunWith(
			entagentrun.HasAgentWith(
				entagent.RuntimeControlStateEQ(entagent.RuntimeControlStatePauseRequested),
			),
		),
	)
	if err != nil {
		return fmt.Errorf("list agents pending pause: %w", err)
	}

	for _, assignment := range pausedAssignments {
		if err := l.pauseAgent(ctx, assignment); err != nil {
			return err
		}
	}

	return nil
}

func (l *RuntimeLauncher) refreshHeartbeats(ctx context.Context) error {
	l.sessionsMu.Lock()
	runIDs := make([]uuid.UUID, 0, len(l.sessions))
	for runID := range l.sessions {
		runIDs = append(runIDs, runID)
	}
	l.sessionsMu.Unlock()

	if len(runIDs) == 0 {
		return nil
	}

	now := l.now().UTC()
	for _, runID := range runIDs {
		assignment, err := l.loadAssignmentByRun(ctx, runID)
		if err != nil {
			stopSession(context.Background(), l.loadSession(runID))
			l.deleteSession(runID)
			l.finishExecution(runID)
			continue
		}
		if assignment.agent == nil || assignment.run == nil || assignment.ticket == nil ||
			assignment.agent.RuntimeControlState != entagent.RuntimeControlStateActive ||
			(assignment.run.Status != entagentrun.StatusReady && assignment.run.Status != entagentrun.StatusExecuting) {
			stopSession(context.Background(), l.loadSession(runID))
			l.deleteSession(runID)
			l.finishExecution(runID)
			continue
		}
	}

	_ = now
	return nil
}

func (l *RuntimeLauncher) pauseAgent(ctx context.Context, assignment runtimeAssignment) error {
	if assignment.agent == nil || assignment.run == nil {
		return nil
	}

	session := l.loadSession(assignment.run.ID)
	if session != nil {
		stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		stopErr := session.Stop(stopCtx)
		cancel()
		if stopErr != nil {
			return fmt.Errorf("stop runtime session for run %s: %w", assignment.run.ID, stopErr)
		}
		l.deleteSession(assignment.run.ID)
	}

	pausedAt := l.now().UTC()
	pausedCount, err := clearRuntimeState(
		l.client.AgentRun.Update().
			Where(
				entagentrun.IDEQ(assignment.run.ID),
				entagentrun.StatusIn(entagentrun.StatusLaunching, entagentrun.StatusReady, entagentrun.StatusExecuting),
			).
			SetStatus(entagentrun.StatusTerminated),
	).Save(ctx)
	if err != nil {
		return fmt.Errorf("mark agent %s paused: %w", assignment.agent.ID, err)
	}
	if pausedCount == 0 {
		return nil
	}

	if _, err := l.client.Agent.UpdateOneID(assignment.agent.ID).
		SetRuntimeControlState(entagent.RuntimeControlStatePaused).
		Save(ctx); err != nil {
		return fmt.Errorf("mark agent %s control paused: %w", assignment.agent.ID, err)
	}

	pausedAgent, err := loadAgentLifecycleState(ctx, l.client, assignment.agent.ID, &assignment.run.ID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentPausedType,
		pausedAgent,
		lifecycleMessage(agentPausedType, pausedAgent.agent.Name),
		runtimeEventMetadataForState(pausedAgent),
		pausedAt,
	)
	return nil
}

func (l *RuntimeLauncher) startCodexSession(ctx context.Context, assignment runtimeAssignment) (*codex.Session, error) {
	launchContext, err := l.loadLaunchContext(ctx, assignment.agent.ID, assignment.ticket.ID)
	if err != nil {
		return nil, err
	}
	if launchContext.agent.Edges.Provider.AdapterType != entagentprovider.AdapterTypeCodexAppServer {
		return nil, fmt.Errorf("unsupported adapter type %s", launchContext.agent.Edges.Provider.AdapterType)
	}

	machine, remote, err := l.resolveLaunchMachine(ctx, launchContext)
	if err != nil {
		return nil, err
	}

	commandString := launchContext.agent.Edges.Provider.CliCommand
	if machine.AgentCLIPath != nil {
		commandString = *machine.AgentCLIPath
	}

	command, err := provider.ParseAgentCLICommand(commandString)
	if err != nil {
		return nil, fmt.Errorf("parse agent cli command: %w", err)
	}
	environment := buildAgentCLIEnvironment(machine.EnvVars, launchContext.agent.Edges.Provider.AuthConfig)
	platformEnvironment, err := l.buildAgentPlatformEnvironment(ctx, launchContext)
	if err != nil {
		return nil, err
	}
	environment = append(environment, platformEnvironment...)
	if !remote {
		launcherEnvironment, err := buildLocalOpenASEEnvironment()
		if err != nil {
			return nil, err
		}
		environment = append(environment, launcherEnvironment...)
	}
	if requiresMachineCodexReady(command, environment) {
		if ready, reason, ok := machineCodexReady(machine.Resources); ok && !ready {
			return nil, fmt.Errorf("machine %s codex environment not ready: %s", machine.Name, reason)
		}
	}

	resolvedGitHubToken, err := l.resolveProjectGitHubToken(ctx, launchContext)
	if err != nil {
		return nil, err
	}

	workspaceRequest, err := buildWorkspaceRequest(launchContext, machine, remote, resolvedGitHubToken)
	if err != nil {
		return nil, err
	}

	var workspaceItem workspaceinfra.Workspace
	if remote {
		if l.sshPool == nil {
			return nil, fmt.Errorf("ssh pool unavailable for remote machine %s", machine.Name)
		}
		workspaceItem, err = workspaceinfra.NewRemoteManager(l.sshPool).Prepare(ctx, machine, workspaceRequest)
		if err != nil {
			return nil, err
		}
	} else {
		workspaceItem, err = workspaceinfra.NewManager().Prepare(ctx, workspaceRequest)
		if err != nil {
			return nil, err
		}
	}

	workingDirectoryValue := resolveAgentWorkingDirectory(launchContext, workspaceItem)
	if !remote && l.workflow != nil {
		if _, err := l.workflow.RefreshSkills(ctx, workflowservice.RefreshSkillsInput{
			ProjectID:     launchContext.project.ID,
			WorkspaceRoot: workingDirectoryValue,
			AdapterType:   string(launchContext.agent.Edges.Provider.AdapterType),
		}); err != nil {
			return nil, fmt.Errorf("prepare local codex workspace skills: %w", err)
		}
	}
	workingDirectory, err := provider.ParseAbsolutePath(workingDirectoryValue)
	if err != nil {
		return nil, fmt.Errorf("parse agent workspace path: %w", err)
	}
	developerInstructions, err := l.buildDeveloperInstructions(
		ctx,
		launchContext,
		machine,
		workingDirectory.String(),
	)
	if err != nil {
		return nil, err
	}

	processManager := l.processManager
	if remote {
		processManager = sshinfra.NewProcessManager(l.sshPool, machine)
	}

	processSpec, err := provider.NewAgentCLIProcessSpec(
		command,
		launchContext.agent.Edges.Provider.CliArgs,
		&workingDirectory,
		environment,
	)
	if err != nil {
		return nil, fmt.Errorf("build codex process spec: %w", err)
	}

	adapter, err := codex.NewAdapter(codex.AdapterOptions{ProcessManager: processManager})
	if err != nil {
		return nil, fmt.Errorf("construct codex adapter: %w", err)
	}

	session, err := adapter.Start(ctx, codex.StartRequest{
		Process: processSpec,
		Initialize: codex.InitializeParams{
			ClientName:    "openase",
			ClientVersion: "dev",
			ClientTitle:   "OpenASE",
		},
		Thread: codex.ThreadStartParams{
			WorkingDirectory:       workingDirectory.String(),
			Model:                  launchContext.agent.Edges.Provider.ModelName,
			ServiceName:            "openase",
			DeveloperInstructions:  developerInstructions,
			ApprovalPolicy:         "never",
			Sandbox:                "danger-full-access",
			PersistExtendedHistory: true,
		},
		Turn: codex.TurnConfig{
			WorkingDirectory: workingDirectory.String(),
			Title:            fmt.Sprintf("%s: %s", launchContext.ticket.Identifier, launchContext.ticket.Title),
			ApprovalPolicy:   "never",
			SandboxPolicy: map[string]any{
				"type":          "dangerFullAccess",
				"networkAccess": true,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (l *RuntimeLauncher) buildAgentPlatformEnvironment(ctx context.Context, launchContext runtimeLaunchContext) ([]string, error) {
	if l == nil || l.agentPlatform == nil {
		return nil, nil
	}
	if launchContext.agent == nil || launchContext.project == nil || launchContext.ticket == nil {
		return nil, fmt.Errorf("runtime launch context is incomplete for platform environment")
	}

	issued, err := l.agentPlatform.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   launchContext.agent.ID,
		ProjectID: launchContext.project.ID,
		TicketID:  launchContext.ticket.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("issue agent platform token: %w", err)
	}

	return agentplatform.BuildEnvironment(
		l.platformAPIURL,
		issued.Token,
		launchContext.project.ID,
		launchContext.ticket.ID,
	), nil
}

func buildLocalOpenASEEnvironment() ([]string, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve local openase executable: %w", err)
	}
	if strings.TrimSpace(executable) == "" {
		return nil, fmt.Errorf("resolve local openase executable: empty path")
	}

	return []string{"OPENASE_REAL_BIN=" + executable}, nil
}

func (l *RuntimeLauncher) buildDeveloperInstructions(
	ctx context.Context,
	launchContext runtimeLaunchContext,
	machine catalogdomain.Machine,
	workspace string,
) (string, error) {
	if l == nil || l.workflow == nil || launchContext.ticket == nil || launchContext.ticket.WorkflowID == nil {
		return "", nil
	}
	if launchContext.agent == nil || launchContext.project == nil {
		return "", fmt.Errorf("runtime launch context is incomplete for harness injection")
	}

	document, err := l.workflow.GetHarness(ctx, *launchContext.ticket.WorkflowID)
	if err != nil {
		return "", fmt.Errorf("load workflow harness for agent launch: %w", err)
	}

	currentMachine, accessibleMachines, err := l.loadMachineAccess(ctx, launchContext.project, machine, workspace)
	if err != nil {
		return "", fmt.Errorf("load project machine access for harness injection: %w", err)
	}

	data, err := l.workflow.BuildHarnessTemplateData(ctx, workflowservice.BuildHarnessTemplateDataInput{
		WorkflowID:         *launchContext.ticket.WorkflowID,
		TicketID:           launchContext.ticket.ID,
		AgentID:            &launchContext.agent.ID,
		Workspace:          strings.TrimSpace(workspace),
		Timestamp:          l.now().UTC(),
		Machine:            currentMachine,
		AccessibleMachines: accessibleMachines,
	})
	if err != nil {
		return "", fmt.Errorf("build workflow harness context for agent launch: %w", err)
	}

	rendered, err := workflowservice.RenderHarnessBody(document.Content, data)
	if err != nil {
		return "", fmt.Errorf("render workflow harness for agent launch: %w", err)
	}

	return strings.TrimSpace(rendered), nil
}

type runtimeLaunchContext struct {
	agent        *ent.Agent
	project      *ent.Project
	ticket       *ent.Ticket
	projectRepos []*ent.ProjectRepo
	ticketScopes []*ent.TicketRepoScope
}

func (l *RuntimeLauncher) loadLaunchContext(ctx context.Context, agentID uuid.UUID, ticketID uuid.UUID) (runtimeLaunchContext, error) {
	if agentID == uuid.Nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent id must not be empty")
	}
	if ticketID == uuid.Nil {
		return runtimeLaunchContext{}, fmt.Errorf("ticket id must not be empty")
	}

	loadedAgent, err := l.client.Agent.Query().
		Where(entagent.IDEQ(agentID)).
		WithProvider().
		WithProject(func(query *ent.ProjectQuery) {
			query.WithOrganization()
			query.WithRepos(func(repoQuery *ent.ProjectRepoQuery) {
				repoQuery.Order(entprojectrepo.ByName())
			})
		}).
		Only(ctx)
	if err != nil {
		return runtimeLaunchContext{}, fmt.Errorf("load runtime launch context for agent %s: %w", agentID, err)
	}
	if loadedAgent.Edges.Provider == nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent provider must be loaded")
	}
	if loadedAgent.Edges.Project == nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent project must be loaded")
	}
	if loadedAgent.Edges.Project.Edges.Organization == nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent project organization must be loaded")
	}

	ticketItem, err := l.client.Ticket.Query().
		Where(entticket.IDEQ(ticketID)).
		WithRepoScopes(func(scopeQuery *ent.TicketRepoScopeQuery) {
			scopeQuery.Order(
				entticketreposcope.ByIsPrimaryScope(),
				entticketreposcope.ByRepoID(),
			)
		}).
		Only(ctx)
	if err != nil {
		return runtimeLaunchContext{}, fmt.Errorf("load runtime launch ticket %s: %w", ticketID, err)
	}

	return runtimeLaunchContext{
		agent:        loadedAgent,
		project:      loadedAgent.Edges.Project,
		ticket:       ticketItem,
		projectRepos: loadedAgent.Edges.Project.Edges.Repos,
		ticketScopes: ticketItem.Edges.RepoScopes,
	}, nil
}

func (l *RuntimeLauncher) listAssignments(ctx context.Context, predicates ...predicate.Ticket) ([]runtimeAssignment, error) {
	items, err := l.client.Ticket.Query().
		Where(predicates...).
		WithCurrentRun(func(query *ent.AgentRunQuery) {
			query.WithAgent()
		}).
		Order(ent.Asc(entticket.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	assignments := make([]runtimeAssignment, 0, len(items))
	for _, ticketItem := range items {
		if ticketItem.Edges.CurrentRun == nil || ticketItem.Edges.CurrentRun.Edges.Agent == nil {
			continue
		}
		assignments = append(assignments, runtimeAssignment{
			ticket: ticketItem,
			agent:  ticketItem.Edges.CurrentRun.Edges.Agent,
			run:    ticketItem.Edges.CurrentRun,
		})
	}
	return assignments, nil
}

func (l *RuntimeLauncher) loadAssignmentByRun(ctx context.Context, runID uuid.UUID) (runtimeAssignment, error) {
	assignments, err := l.listAssignments(ctx,
		entticket.CurrentRunIDEQ(runID),
	)
	if err != nil {
		return runtimeAssignment{}, err
	}
	if len(assignments) == 0 {
		return runtimeAssignment{}, nil
	}
	return assignments[0], nil
}

func (l *RuntimeLauncher) resolveLaunchMachine(ctx context.Context, launchContext runtimeLaunchContext) (catalogdomain.Machine, bool, error) {
	machines, err := l.client.Machine.Query().
		Where(entmachine.OrganizationID(launchContext.project.OrganizationID)).
		Order(entmachine.ByName()).
		All(ctx)
	if err != nil {
		return catalogdomain.Machine{}, false, fmt.Errorf("list machines for runtime launch: %w", err)
	}

	providerItem := launchContext.agent.Edges.Provider
	if providerItem == nil {
		return catalogdomain.Machine{}, false, fmt.Errorf("agent provider must be loaded")
	}

	for _, machineItem := range machines {
		if machineItem.ID == providerItem.MachineID {
			mapped := mapRuntimeMachine(machineItem)
			return mapped, mapped.Host != catalogdomain.LocalMachineHost, nil
		}
	}

	return catalogdomain.Machine{}, false, fmt.Errorf("provider %s bound machine %s not found", providerItem.ID, providerItem.MachineID)
}

func buildWorkspaceRequest(
	launchContext runtimeLaunchContext,
	machine catalogdomain.Machine,
	remote bool,
	githubToken string,
) (workspaceinfra.SetupRequest, error) {
	if launchContext.project == nil || launchContext.project.Edges.Organization == nil {
		return workspaceinfra.SetupRequest{}, fmt.Errorf("project organization must be loaded for ticket workspace derivation")
	}

	workspaceRoot, err := resolveWorkspaceRoot(machine, remote)
	if err != nil {
		return workspaceinfra.SetupRequest{}, err
	}

	repoInputs := buildWorkspaceRepoInputs(launchContext.projectRepos, launchContext.ticketScopes, githubToken)
	request, err := workspaceinfra.ParseSetupRequest(workspaceinfra.SetupInput{
		WorkspaceRoot:    workspaceRoot,
		OrganizationSlug: launchContext.project.Edges.Organization.Slug,
		ProjectSlug:      launchContext.project.Slug,
		AgentName:        launchContext.agent.Name,
		TicketIdentifier: launchContext.ticket.Identifier,
		Repos:            repoInputs,
	})
	if err != nil {
		return workspaceinfra.SetupRequest{}, fmt.Errorf("build ticket workspace request: %w", err)
	}

	return request, nil
}

func resolveWorkspaceRoot(machine catalogdomain.Machine, remote bool) (string, error) {
	if remote {
		if machine.WorkspaceRoot == nil {
			return "", fmt.Errorf("machine %s is missing workspace_root", machine.Name)
		}
		return strings.TrimSpace(*machine.WorkspaceRoot), nil
	}

	root, err := workspaceinfra.LocalWorkspaceRoot()
	if err != nil {
		return "", fmt.Errorf("resolve local workspace root: %w", err)
	}
	return root, nil
}

func buildWorkspacePath(launchContext runtimeLaunchContext, machine catalogdomain.Machine, remote bool) (string, error) {
	request, err := buildWorkspaceRequest(launchContext, machine, remote, "")
	if err != nil {
		return "", err
	}

	workspacePath, err := workspaceinfra.TicketWorkspacePath(
		request.WorkspaceRoot,
		request.OrganizationSlug,
		request.ProjectSlug,
		request.TicketIdentifier,
	)
	if err != nil {
		return "", fmt.Errorf("derive ticket workspace path: %w", err)
	}

	return workspacePath, nil
}

func buildWorkspaceRepoInputs(
	projectRepos []*ent.ProjectRepo,
	ticketScopes []*ent.TicketRepoScope,
	githubToken string,
) []workspaceinfra.RepoInput {
	scopeByRepoID := make(map[uuid.UUID]*ent.TicketRepoScope, len(ticketScopes))
	for _, scope := range ticketScopes {
		scopeByRepoID[scope.RepoID] = scope
	}

	selectedRepos := projectRepos
	if len(scopeByRepoID) > 0 {
		selectedRepos = make([]*ent.ProjectRepo, 0, len(scopeByRepoID))
		for _, repo := range projectRepos {
			if _, ok := scopeByRepoID[repo.ID]; ok {
				selectedRepos = append(selectedRepos, repo)
			}
		}
	}

	inputs := make([]workspaceinfra.RepoInput, 0, len(selectedRepos))
	for _, repo := range selectedRepos {
		input := workspaceinfra.RepoInput{
			Name:          repo.Name,
			RepositoryURL: repo.RepositoryURL,
			DefaultBranch: repo.DefaultBranch,
		}
		if githubToken != "" {
			if _, ok := githubauthdomain.ParseGitHubRepositoryURL(repo.RepositoryURL); ok {
				token := githubToken
				input.GitHubToken = &token
			}
		}
		if clonePath := strings.TrimSpace(repo.ClonePath); clonePath != "" {
			input.ClonePath = &clonePath
		}
		if scope, ok := scopeByRepoID[repo.ID]; ok {
			branchName := scope.BranchName
			input.BranchName = &branchName
		}
		inputs = append(inputs, input)
	}

	return inputs
}

func (l *RuntimeLauncher) resolveProjectGitHubToken(ctx context.Context, launchContext runtimeLaunchContext) (string, error) {
	if l == nil || l.githubAuth == nil || launchContext.project == nil {
		return "", nil
	}

	resolved, err := l.githubAuth.ResolveProjectCredential(ctx, launchContext.project.ID)
	if err != nil {
		return "", fmt.Errorf("resolve GitHub outbound credential for project %s: %w", launchContext.project.ID, err)
	}
	return resolved.Token, nil
}

func resolveAgentWorkingDirectory(launchContext runtimeLaunchContext, workspaceItem workspaceinfra.Workspace) string {
	if len(workspaceItem.Repos) == 0 {
		return workspaceItem.Path
	}

	if primaryPath, ok := primaryPreparedRepoPath(launchContext, workspaceItem.Repos); ok {
		return primaryPath
	}

	if len(workspaceItem.Repos) == 1 {
		return workspaceItem.Repos[0].Path
	}

	return workspaceItem.Path
}

func primaryPreparedRepoPath(
	launchContext runtimeLaunchContext,
	repos []workspaceinfra.PreparedRepo,
) (string, bool) {
	primaryClonePath := primaryWorkspaceClonePath(launchContext)
	if primaryClonePath == "" {
		return "", false
	}

	for _, repo := range repos {
		if repo.ClonePath == primaryClonePath {
			return repo.Path, true
		}
	}

	return "", false
}

func primaryWorkspaceClonePath(launchContext runtimeLaunchContext) string {
	projectReposByID := make(map[uuid.UUID]*ent.ProjectRepo, len(launchContext.projectRepos))
	for _, repo := range launchContext.projectRepos {
		projectReposByID[repo.ID] = repo
	}

	for _, scope := range launchContext.ticketScopes {
		if !scope.IsPrimaryScope {
			continue
		}
		if repo := projectReposByID[scope.RepoID]; repo != nil {
			return projectRepoClonePath(repo)
		}
	}

	for _, repo := range launchContext.projectRepos {
		if repo.IsPrimary {
			return projectRepoClonePath(repo)
		}
	}

	return ""
}

func projectRepoClonePath(repo *ent.ProjectRepo) string {
	if repo == nil {
		return ""
	}
	if clonePath := strings.TrimSpace(repo.ClonePath); clonePath != "" {
		return clonePath
	}

	return repo.Name
}

func mapRuntimeMachine(item *ent.Machine) catalogdomain.Machine {
	return catalogdomain.Machine{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		Name:           item.Name,
		Host:           item.Host,
		Port:           item.Port,
		SSHUser:        optionalRuntimeString(item.SSHUser),
		SSHKeyPath:     optionalRuntimeString(item.SSHKeyPath),
		Description:    item.Description,
		Labels:         append([]string(nil), item.Labels...),
		Status:         catalogdomain.MachineStatus(item.Status),
		WorkspaceRoot:  optionalRuntimeString(item.WorkspaceRoot),
		AgentCLIPath:   optionalRuntimeString(item.AgentCliPath),
		EnvVars:        append([]string(nil), item.EnvVars...),
		Resources:      cloneResourceMap(item.Resources),
	}
}

func optionalRuntimeString(raw string) *string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	value := raw
	return &value
}

func machineCodexReady(resources map[string]any) (bool, string, bool) {
	monitor, ok := nestedMap(resources, "monitor")
	if !ok {
		return false, "", false
	}
	levelMap, ok := nestedMap(monitor, "l4")
	if !ok {
		return false, "", false
	}
	codexMap, ok := nestedMap(levelMap, "codex")
	if !ok {
		return false, "", false
	}

	installed := anyToBool(codexMap["installed"])
	authStatus := strings.TrimSpace(fmt.Sprint(codexMap["auth_status"]))
	if rawReady, exists := codexMap["ready"]; exists {
		if anyToBool(rawReady) {
			return true, "", true
		}
	} else if installed && !strings.EqualFold(authStatus, "not_logged_in") {
		return true, "", true
	}

	if !installed {
		return false, "codex cli is not installed", true
	}

	if strings.EqualFold(authStatus, "not_logged_in") {
		return false, "codex cli is not logged in", true
	}

	return false, "codex cli is not ready", true
}

func buildAgentCLIEnvironment(machineEnv []string, authConfig map[string]any) []string {
	environment := append([]string(nil), machineEnv...)
	return append(environment, provider.AuthConfigEnvironment(authConfig)...)
}

func requiresMachineCodexReady(command provider.AgentCLICommand, environment []string) bool {
	if value, ok := provider.LookupEnvironmentValue(environment, "OPENAI_API_KEY"); ok && strings.TrimSpace(value) != "" {
		return false
	}

	executable := agentCLIExecutable(command)
	if executable == "" {
		return false
	}

	base := path.Base(strings.ReplaceAll(executable, "\\", "/"))
	return strings.EqualFold(base, "codex") || strings.EqualFold(base, "codex.exe")
}

func agentCLIExecutable(command provider.AgentCLICommand) string {
	trimmed := strings.TrimSpace(command.String())
	if trimmed == "" {
		return ""
	}

	if isCodexExecutablePath(trimmed) {
		return strings.Trim(trimmed, `"'`)
	}

	token := firstCommandToken(trimmed)
	if token == "" {
		return ""
	}
	return strings.Trim(token, `"'`)
}

func firstCommandToken(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if quote := trimmed[0]; quote == '"' || quote == '\'' {
		for index := 1; index < len(trimmed); index++ {
			if trimmed[index] == quote {
				return trimmed[1:index]
			}
		}
		return strings.Trim(trimmed, `"'`)
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func isCodexExecutablePath(raw string) bool {
	base := path.Base(strings.ReplaceAll(strings.Trim(raw, `"'`), "\\", "/"))
	return strings.EqualFold(base, "codex") || strings.EqualFold(base, "codex.exe")
}

func (l *RuntimeLauncher) loadMachineAccess(
	ctx context.Context,
	projectItem *ent.Project,
	currentMachine catalogdomain.Machine,
	workspace string,
) (workflowservice.HarnessMachineData, []workflowservice.HarnessAccessibleMachineData, error) {
	if projectItem == nil {
		return workflowservice.HarnessMachineData{}, nil, fmt.Errorf("project must not be nil")
	}

	accessibleMachines, err := l.resolveAccessibleMachines(
		ctx,
		projectItem.OrganizationID,
		projectItem.AccessibleMachineIds,
		currentMachine,
	)
	if err != nil {
		return workflowservice.HarnessMachineData{}, nil, err
	}

	return mapHarnessMachine(currentMachine, workspace), accessibleMachines, nil
}

func mapHarnessMachine(machine catalogdomain.Machine, workspace string) workflowservice.HarnessMachineData {
	root := workspaceRoot(machine, workspace)

	return workflowservice.HarnessMachineData{
		Name:          machine.Name,
		Host:          machine.Host,
		Description:   machine.Description,
		Labels:        append([]string(nil), machine.Labels...),
		WorkspaceRoot: root,
	}
}

func (l *RuntimeLauncher) resolveAccessibleMachines(
	ctx context.Context,
	organizationID uuid.UUID,
	machineIDs []uuid.UUID,
	currentMachine catalogdomain.Machine,
) ([]workflowservice.HarnessAccessibleMachineData, error) {
	if len(machineIDs) == 0 {
		return nil, nil
	}

	items, err := l.client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(organizationID),
			entmachine.IDIn(machineIDs...),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query accessible machines: %w", err)
	}

	byID := make(map[uuid.UUID]*ent.Machine, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}

	accessible := make([]workflowservice.HarnessAccessibleMachineData, 0, len(machineIDs))
	for _, machineID := range machineIDs {
		item, ok := byID[machineID]
		if !ok {
			return nil, fmt.Errorf("project accessible machine %s not found", machineID)
		}
		if currentMachine.ID != uuid.Nil && currentMachine.ID == item.ID {
			continue
		}
		if strings.TrimSpace(currentMachine.Host) != "" && currentMachine.Host == item.Host {
			continue
		}
		if item.Host == catalogdomain.LocalMachineHost {
			continue
		}
		accessible = append(accessible, workflowservice.HarnessAccessibleMachineData{
			Name:        item.Name,
			Host:        item.Host,
			Description: item.Description,
			Labels:      append([]string(nil), item.Labels...),
			SSHUser:     strings.TrimSpace(item.SSHUser),
		})
	}

	slices.SortFunc(accessible, func(left, right workflowservice.HarnessAccessibleMachineData) int {
		return strings.Compare(left.Name, right.Name)
	})

	return accessible, nil
}

func workspaceRoot(machine catalogdomain.Machine, workspace string) string {
	if strings.TrimSpace(machine.Host) == "" || machine.Host == catalogdomain.LocalMachineHost {
		if root, err := workspaceinfra.LocalWorkspaceRoot(); err == nil {
			return root
		}
	}
	if machine.WorkspaceRoot != nil && strings.TrimSpace(*machine.WorkspaceRoot) != "" {
		return strings.TrimSpace(*machine.WorkspaceRoot)
	}
	trimmed := strings.TrimSpace(workspace)
	if trimmed == "" {
		return ""
	}
	parent := filepath.Clean(filepath.Dir(trimmed))
	return filepath.Clean(filepath.Dir(filepath.Dir(parent)))
}

func (l *RuntimeLauncher) storeSession(runID uuid.UUID, session *codex.Session) {
	l.sessionsMu.Lock()
	defer l.sessionsMu.Unlock()
	l.sessions[runID] = session
}

func (l *RuntimeLauncher) loadSession(runID uuid.UUID) *codex.Session {
	l.sessionsMu.Lock()
	defer l.sessionsMu.Unlock()
	return l.sessions[runID]
}

func (l *RuntimeLauncher) deleteSession(runID uuid.UUID) {
	l.sessionsMu.Lock()
	defer l.sessionsMu.Unlock()
	delete(l.sessions, runID)
}

func (l *RuntimeLauncher) drainSessions() map[uuid.UUID]*codex.Session {
	l.sessionsMu.Lock()
	defer l.sessionsMu.Unlock()

	drained := make(map[uuid.UUID]*codex.Session, len(l.sessions))
	for runID, session := range l.sessions {
		drained[runID] = session
	}
	l.sessions = map[uuid.UUID]*codex.Session{}
	return drained
}

func (l *RuntimeLauncher) beginExecution(runID uuid.UUID) bool {
	l.executionsMu.Lock()
	defer l.executionsMu.Unlock()
	if _, exists := l.executions[runID]; exists {
		return false
	}
	l.executions[runID] = struct{}{}
	return true
}

func (l *RuntimeLauncher) finishExecution(runID uuid.UUID) {
	l.executionsMu.Lock()
	defer l.executionsMu.Unlock()
	delete(l.executions, runID)
}

func (l *RuntimeLauncher) beginLaunch(runID uuid.UUID) bool {
	l.launchesMu.Lock()
	defer l.launchesMu.Unlock()
	if _, exists := l.launches[runID]; exists {
		return false
	}
	l.launches[runID] = struct{}{}
	return true
}

func (l *RuntimeLauncher) finishLaunch(runID uuid.UUID) {
	l.launchesMu.Lock()
	defer l.launchesMu.Unlock()
	delete(l.launches, runID)
}

func (l *RuntimeLauncher) launchContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	if timeout <= 0 {
		return base, func() {}
	}
	return context.WithTimeout(base, timeout)
}

func (l *RuntimeLauncher) publishLifecycleEvent(
	ctx context.Context,
	eventType provider.EventType,
	state agentLifecycleState,
	message string,
	metadata map[string]any,
	publishedAt time.Time,
) {
	if l == nil || state.agent == nil {
		return
	}

	publishCtx, cancel := l.launchContext(ctx, l.eventTimeout)
	defer cancel()
	if err := publishAgentLifecycleEvent(
		publishCtx,
		l.client,
		l.events,
		eventType,
		state,
		message,
		metadata,
		publishedAt,
	); err != nil {
		l.logger.Warn(
			"publish agent lifecycle",
			"event_type", eventType.String(),
			"agent_id", state.agent.ID,
			"run_id", uuidPointerToString(lifecycleRunUUID(state)),
			"error", err,
		)
	}
}

func lifecycleRunUUID(state agentLifecycleState) *uuid.UUID {
	if state.run == nil {
		return nil
	}
	value := state.run.ID
	return &value
}

func stopSession(ctx context.Context, session *codex.Session) {
	if session == nil {
		return
	}
	_ = session.Stop(ctx)
}
