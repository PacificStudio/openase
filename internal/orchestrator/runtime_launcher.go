package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticketreposcope "github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

type RuntimeLauncher struct {
	client         *ent.Client
	logger         *slog.Logger
	events         provider.EventProvider
	processManager provider.AgentCLIProcessManager
	sshPool        *sshinfra.Pool
	workflow       *workflowservice.Service
	now            func() time.Time

	sessionsMu sync.Mutex
	sessions   map[uuid.UUID]*codex.Session

	executionsMu sync.Mutex
	executions   map[uuid.UUID]struct{}

	tickets *ticketservice.Service
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
		sessions:       map[uuid.UUID]*codex.Session{},
		executions:     map[uuid.UUID]struct{}{},
		tickets:        ticketservice.NewService(client),
	}
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

	claimedAgents, err := l.client.Agent.Query().
		Where(
			entagent.StatusEQ(entagent.StatusClaimed),
			entagent.RuntimePhaseEQ(entagent.RuntimePhaseNone),
			entagent.RuntimeControlStateEQ(entagent.RuntimeControlStateActive),
			entagent.CurrentTicketIDNotNil(),
		).
		WithProvider().
		Order(ent.Asc(entagent.FieldName)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list claimed agents awaiting launch: %w", err)
	}

	for _, agentItem := range claimedAgents {
		if err := l.launchAgent(ctx, agentItem); err != nil {
			l.logger.Error("launch claimed agent", "agent_id", agentItem.ID, "error", err)
		}
	}

	return nil
}

func (l *RuntimeLauncher) Close(ctx context.Context) error {
	if l == nil {
		return nil
	}

	sessions := l.drainSessions()
	for agentID, session := range sessions {
		if session != nil {
			stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			_ = session.Stop(stopCtx)
			cancel()
		}

		now := l.now().UTC()
		if _, err := clearRuntimeState(
			l.client.Agent.Update().
				Where(
					entagent.IDEQ(agentID),
					entagent.StatusIn(entagent.StatusClaimed, entagent.StatusRunning),
				).
				SetStatus(entagent.StatusTerminated),
		).Save(ctx); err != nil {
			l.logger.Warn("mark agent terminated", "agent_id", agentID, "error", err)
			continue
		}

		agentItem, err := loadAgentLifecycleState(ctx, l.client, agentID)
		if err != nil {
			l.logger.Warn("reload terminated agent", "agent_id", agentID, "error", err)
			continue
		}
		if err := publishAgentLifecycleEvent(
			ctx,
			l.client,
			l.events,
			agentTerminatedType,
			agentItem,
			lifecycleMessage(agentTerminatedType, agentItem.Name),
			runtimeEventMetadata(agentItem),
			now,
		); err != nil {
			l.logger.Warn("publish terminated lifecycle", "agent_id", agentID, "error", err)
		}
	}

	return nil
}

func (l *RuntimeLauncher) launchAgent(ctx context.Context, agentItem *ent.Agent) error {
	if agentItem == nil || agentItem.CurrentTicketID == nil {
		return nil
	}

	now := l.now().UTC()
	launchingCount, err := l.client.Agent.Update().
		Where(
			entagent.IDEQ(agentItem.ID),
			entagent.StatusEQ(entagent.StatusClaimed),
			entagent.RuntimePhaseEQ(entagent.RuntimePhaseNone),
			entagent.CurrentTicketIDEQ(*agentItem.CurrentTicketID),
		).
		SetRuntimePhase(entagent.RuntimePhaseLaunching).
		SetLastError("").
		ClearSessionID().
		ClearRuntimeStartedAt().
		ClearLastHeartbeatAt().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark agent %s launching: %w", agentItem.ID, err)
	}
	if launchingCount == 0 {
		return nil
	}

	launchingAgent, err := loadAgentLifecycleState(ctx, l.client, agentItem.ID)
	if err != nil {
		return err
	}
	if err := publishAgentLifecycleEvent(
		ctx,
		l.client,
		l.events,
		agentLaunchingType,
		launchingAgent,
		lifecycleMessage(agentLaunchingType, launchingAgent.Name),
		runtimeEventMetadata(launchingAgent),
		now,
	); err != nil {
		return err
	}

	session, launchErr := l.startCodexSession(ctx, agentItem)
	if launchErr != nil {
		return l.markLaunchFailed(ctx, agentItem.ID, launchErr)
	}

	l.storeSession(agentItem.ID, session)

	readyAt := l.now().UTC()
	readyCount, err := l.client.Agent.Update().
		Where(
			entagent.IDEQ(agentItem.ID),
			entagent.StatusEQ(entagent.StatusClaimed),
			entagent.RuntimePhaseEQ(entagent.RuntimePhaseLaunching),
		).
		SetStatus(entagent.StatusRunning).
		SetSessionID(session.ThreadID()).
		SetRuntimePhase(entagent.RuntimePhaseReady).
		SetRuntimeStartedAt(readyAt).
		SetLastHeartbeatAt(readyAt).
		SetLastError("").
		Save(ctx)
	if err != nil {
		l.deleteSession(agentItem.ID)
		stopSession(context.Background(), session)
		return fmt.Errorf("mark agent %s ready: %w", agentItem.ID, err)
	}
	if readyCount == 0 {
		l.deleteSession(agentItem.ID)
		stopSession(context.Background(), session)
		return nil
	}

	readyAgent, err := loadAgentLifecycleState(ctx, l.client, agentItem.ID)
	if err != nil {
		return err
	}
	if err := publishAgentLifecycleEvent(
		ctx,
		l.client,
		l.events,
		agentReadyType,
		readyAgent,
		lifecycleMessage(agentReadyType, readyAgent.Name),
		runtimeEventMetadata(readyAgent),
		readyAt,
	); err != nil {
		return err
	}
	if err := publishAgentLifecycleEvent(
		ctx,
		l.client,
		l.events,
		agentHeartbeatType,
		readyAgent,
		lifecycleMessage(agentHeartbeatType, readyAgent.Name),
		runtimeEventMetadata(readyAgent),
		readyAt,
	); err != nil {
		return err
	}

	return nil
}

func (l *RuntimeLauncher) markLaunchFailed(ctx context.Context, agentID uuid.UUID, launchErr error) error {
	now := l.now().UTC()
	count, err := l.client.Agent.Update().
		Where(
			entagent.IDEQ(agentID),
			entagent.StatusEQ(entagent.StatusClaimed),
			entagent.RuntimePhaseEQ(entagent.RuntimePhaseLaunching),
		).
		SetStatus(entagent.StatusFailed).
		SetRuntimePhase(entagent.RuntimePhaseFailed).
		SetLastError(strings.TrimSpace(launchErr.Error())).
		ClearSessionID().
		ClearRuntimeStartedAt().
		ClearLastHeartbeatAt().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark agent %s failed: %w", agentID, err)
	}
	if count == 0 {
		return nil
	}

	failedAgent, err := loadAgentLifecycleState(ctx, l.client, agentID)
	if err != nil {
		return err
	}
	return publishAgentLifecycleEvent(
		ctx,
		l.client,
		l.events,
		agentFailedType,
		failedAgent,
		lifecycleMessage(agentFailedType, failedAgent.Name),
		runtimeEventMetadata(failedAgent),
		now,
	)
}

func (l *RuntimeLauncher) reconcilePauseRequests(ctx context.Context) error {
	pausedAgents, err := l.client.Agent.Query().
		Where(
			entagent.RuntimeControlStateEQ(entagent.RuntimeControlStatePauseRequested),
			entagent.StatusIn(entagent.StatusClaimed, entagent.StatusRunning),
			entagent.CurrentTicketIDNotNil(),
		).
		Order(ent.Asc(entagent.FieldName)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list agents pending pause: %w", err)
	}

	for _, agentItem := range pausedAgents {
		if err := l.pauseAgent(ctx, agentItem); err != nil {
			return err
		}
	}

	return nil
}

func (l *RuntimeLauncher) refreshHeartbeats(ctx context.Context) error {
	l.sessionsMu.Lock()
	sessionIDs := make([]uuid.UUID, 0, len(l.sessions))
	for agentID := range l.sessions {
		sessionIDs = append(sessionIDs, agentID)
	}
	l.sessionsMu.Unlock()

	if len(sessionIDs) == 0 {
		return nil
	}

	now := l.now().UTC()
	for _, agentID := range sessionIDs {
		agentItem, err := l.client.Agent.Get(ctx, agentID)
		if err != nil {
			stopSession(context.Background(), l.loadSession(agentID))
			l.deleteSession(agentID)
			l.finishExecution(agentID)
			continue
		}
		if agentItem.Status != entagent.StatusRunning || (agentItem.RuntimePhase != entagent.RuntimePhaseReady && agentItem.RuntimePhase != entagent.RuntimePhaseExecuting) || agentItem.CurrentTicketID == nil {
			stopSession(context.Background(), l.loadSession(agentID))
			l.deleteSession(agentID)
			l.finishExecution(agentID)
			continue
		}
	}

	_ = now
	return nil
}

func (l *RuntimeLauncher) pauseAgent(ctx context.Context, agentItem *ent.Agent) error {
	if agentItem == nil {
		return nil
	}

	session := l.loadSession(agentItem.ID)
	if session != nil {
		stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		stopErr := session.Stop(stopCtx)
		cancel()
		if stopErr != nil {
			return fmt.Errorf("stop runtime session for agent %s: %w", agentItem.ID, stopErr)
		}
		l.deleteSession(agentItem.ID)
	}

	pausedAt := l.now().UTC()
	pausedCount, err := clearRuntimeState(
		l.client.Agent.Update().
			Where(
				entagent.IDEQ(agentItem.ID),
				entagent.RuntimeControlStateEQ(entagent.RuntimeControlStatePauseRequested),
				entagent.StatusIn(entagent.StatusClaimed, entagent.StatusRunning),
				entagent.CurrentTicketIDNotNil(),
			).
			SetStatus(entagent.StatusClaimed).
			SetRuntimeControlState(entagent.RuntimeControlStatePaused),
	).Save(ctx)
	if err != nil {
		return fmt.Errorf("mark agent %s paused: %w", agentItem.ID, err)
	}
	if pausedCount == 0 {
		return nil
	}

	pausedAgent, err := loadAgentLifecycleState(ctx, l.client, agentItem.ID)
	if err != nil {
		return err
	}
	return publishAgentLifecycleEvent(
		ctx,
		l.client,
		l.events,
		agentPausedType,
		pausedAgent,
		lifecycleMessage(agentPausedType, pausedAgent.Name),
		runtimeEventMetadata(pausedAgent),
		pausedAt,
	)
}

func (l *RuntimeLauncher) startCodexSession(ctx context.Context, agentItem *ent.Agent) (*codex.Session, error) {
	launchContext, err := l.loadLaunchContext(ctx, agentItem)
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
	if requiresMachineCodexReady(command, environment) {
		if ready, reason, ok := machineCodexReady(machine.Resources); ok && !ready {
			return nil, fmt.Errorf("machine %s codex environment not ready: %s", machine.Name, reason)
		}
	}

	workingDirectoryValue := strings.TrimSpace(launchContext.agent.WorkspacePath)
	if remote {
		if l.sshPool == nil {
			return nil, fmt.Errorf("ssh pool unavailable for remote machine %s", machine.Name)
		}
		workspaceRequest, err := buildRemoteWorkspaceRequest(launchContext, machine)
		if err != nil {
			return nil, err
		}
		workspaceItem, err := workspaceinfra.NewRemoteManager(l.sshPool).Prepare(ctx, machine, workspaceRequest)
		if err != nil {
			return nil, err
		}
		workingDirectoryValue = workspaceItem.Path
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

	return adapter.Start(ctx, codex.StartRequest{
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

func (l *RuntimeLauncher) loadLaunchContext(ctx context.Context, agentItem *ent.Agent) (runtimeLaunchContext, error) {
	if agentItem == nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent must not be nil")
	}

	loaded, err := l.client.Agent.Query().
		Where(entagent.IDEQ(agentItem.ID)).
		WithProvider().
		WithProject(func(query *ent.ProjectQuery) {
			query.WithRepos(func(repoQuery *ent.ProjectRepoQuery) {
				repoQuery.Order(entprojectrepo.ByName())
			})
		}).
		WithCurrentTicket(func(query *ent.TicketQuery) {
			query.WithRepoScopes(func(scopeQuery *ent.TicketRepoScopeQuery) {
				scopeQuery.Order(
					entticketreposcope.ByIsPrimaryScope(),
					entticketreposcope.ByRepoID(),
				)
			})
		}).
		Only(ctx)
	if err != nil {
		return runtimeLaunchContext{}, fmt.Errorf("load runtime launch context for agent %s: %w", agentItem.ID, err)
	}
	if loaded.Edges.Provider == nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent provider must be loaded")
	}
	if loaded.Edges.Project == nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent project must be loaded")
	}
	if loaded.Edges.CurrentTicket == nil {
		return runtimeLaunchContext{}, fmt.Errorf("agent current ticket must be loaded")
	}

	return runtimeLaunchContext{
		agent:        loaded,
		project:      loaded.Edges.Project,
		ticket:       loaded.Edges.CurrentTicket,
		projectRepos: loaded.Edges.Project.Edges.Repos,
		ticketScopes: loaded.Edges.CurrentTicket.Edges.RepoScopes,
	}, nil
}

func (l *RuntimeLauncher) resolveLaunchMachine(ctx context.Context, launchContext runtimeLaunchContext) (catalogdomain.Machine, bool, error) {
	machines, err := l.client.Machine.Query().
		Where(entmachine.OrganizationID(launchContext.project.OrganizationID)).
		Order(entmachine.ByName()).
		All(ctx)
	if err != nil {
		return catalogdomain.Machine{}, false, fmt.Errorf("list machines for runtime launch: %w", err)
	}

	workspacePath := strings.TrimSpace(launchContext.agent.WorkspacePath)
	var matched *ent.Machine
	for _, machineItem := range machines {
		if machineItem.Host == catalogdomain.LocalMachineHost || strings.TrimSpace(machineItem.WorkspaceRoot) == "" {
			continue
		}
		if pathWithinRoot(workspacePath, machineItem.WorkspaceRoot) {
			if matched != nil {
				return catalogdomain.Machine{}, false, fmt.Errorf("workspace path %q matches multiple remote machines", workspacePath)
			}
			matched = machineItem
		}
	}
	if matched != nil {
		return mapRuntimeMachine(matched), true, nil
	}

	for _, machineItem := range machines {
		if machineItem.Host == catalogdomain.LocalMachineHost {
			return mapRuntimeMachine(machineItem), false, nil
		}
	}

	return catalogdomain.Machine{
		Name: catalogdomain.LocalMachineName,
		Host: catalogdomain.LocalMachineHost,
	}, false, nil
}

func buildRemoteWorkspaceRequest(launchContext runtimeLaunchContext, machine catalogdomain.Machine) (workspaceinfra.SetupRequest, error) {
	if machine.WorkspaceRoot == nil {
		return workspaceinfra.SetupRequest{}, fmt.Errorf("machine %s is missing workspace_root", machine.Name)
	}
	if len(launchContext.projectRepos) == 0 {
		return workspaceinfra.SetupRequest{}, fmt.Errorf("project %s has no repos configured for remote workspace", launchContext.project.ID)
	}

	repoInputs := buildWorkspaceRepoInputs(launchContext.projectRepos, launchContext.ticketScopes)
	request, err := workspaceinfra.ParseSetupRequest(workspaceinfra.SetupInput{
		WorkspaceRoot:    *machine.WorkspaceRoot,
		AgentName:        launchContext.agent.Name,
		TicketIdentifier: launchContext.ticket.Identifier,
		Repos:            repoInputs,
	})
	if err != nil {
		return workspaceinfra.SetupRequest{}, fmt.Errorf("build remote workspace request: %w", err)
	}

	if current := strings.TrimSpace(launchContext.agent.WorkspacePath); current != "" {
		expected := filepath.Join(request.WorkspaceRoot, request.TicketIdentifier)
		if filepath.Clean(current) != expected {
			return workspaceinfra.SetupRequest{}, fmt.Errorf("agent workspace path %q does not match remote workspace %q", current, expected)
		}
	}

	return request, nil
}

func buildWorkspaceRepoInputs(projectRepos []*ent.ProjectRepo, ticketScopes []*ent.TicketRepoScope) []workspaceinfra.RepoInput {
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

func pathWithinRoot(path string, root string) bool {
	trimmedPath := strings.TrimSpace(path)
	trimmedRoot := strings.TrimSpace(root)
	if trimmedPath == "" || trimmedRoot == "" {
		return false
	}

	cleanPath := filepath.Clean(trimmedPath)
	cleanRoot := filepath.Clean(trimmedRoot)
	if cleanPath == cleanRoot {
		return true
	}

	relative, err := filepath.Rel(cleanRoot, cleanPath)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
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
	root := workspaceRoot("", workspace)
	if machine.WorkspaceRoot != nil {
		root = workspaceRoot(*machine.WorkspaceRoot, workspace)
	}

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

func workspaceRoot(configured string, workspace string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	trimmed := strings.TrimSpace(workspace)
	if trimmed == "" {
		return ""
	}
	return filepath.Clean(filepath.Dir(trimmed))
}

func (l *RuntimeLauncher) storeSession(agentID uuid.UUID, session *codex.Session) {
	l.sessionsMu.Lock()
	defer l.sessionsMu.Unlock()
	l.sessions[agentID] = session
}

func (l *RuntimeLauncher) loadSession(agentID uuid.UUID) *codex.Session {
	l.sessionsMu.Lock()
	defer l.sessionsMu.Unlock()
	return l.sessions[agentID]
}

func (l *RuntimeLauncher) deleteSession(agentID uuid.UUID) {
	l.sessionsMu.Lock()
	defer l.sessionsMu.Unlock()
	delete(l.sessions, agentID)
}

func (l *RuntimeLauncher) drainSessions() map[uuid.UUID]*codex.Session {
	l.sessionsMu.Lock()
	defer l.sessionsMu.Unlock()

	drained := make(map[uuid.UUID]*codex.Session, len(l.sessions))
	for agentID, session := range l.sessions {
		drained[agentID] = session
	}
	l.sessions = map[uuid.UUID]*codex.Session{}
	return drained
}

func (l *RuntimeLauncher) beginExecution(agentID uuid.UUID) bool {
	l.executionsMu.Lock()
	defer l.executionsMu.Unlock()
	if _, exists := l.executions[agentID]; exists {
		return false
	}
	l.executions[agentID] = struct{}{}
	return true
}

func (l *RuntimeLauncher) finishExecution(agentID uuid.UUID) {
	l.executionsMu.Lock()
	defer l.executionsMu.Unlock()
	delete(l.executions, agentID)
}

func stopSession(ctx context.Context, session *codex.Session) {
	if session == nil {
		return
	}
	_ = session.Stop(ctx)
}
