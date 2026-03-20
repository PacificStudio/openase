package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

const runtimeHeartbeatInterval = 30 * time.Second

type RuntimeLauncher struct {
	client         *ent.Client
	logger         *slog.Logger
	events         provider.EventProvider
	processManager provider.AgentCLIProcessManager
	now            func() time.Time

	sessionsMu sync.Mutex
	sessions   map[uuid.UUID]*codex.Session
}

func NewRuntimeLauncher(
	client *ent.Client,
	logger *slog.Logger,
	events provider.EventProvider,
	processManager provider.AgentCLIProcessManager,
) *RuntimeLauncher {
	if logger == nil {
		logger = slog.Default()
	}

	return &RuntimeLauncher{
		client:         client,
		logger:         logger.With("component", "runtime-launcher"),
		events:         events,
		processManager: processManager,
		now:            time.Now,
		sessions:       map[uuid.UUID]*codex.Session{},
	}
}

func (l *RuntimeLauncher) RunTick(ctx context.Context) error {
	if l == nil || l.client == nil {
		return fmt.Errorf("runtime launcher unavailable")
	}
	if l.processManager == nil {
		return fmt.Errorf("runtime launcher process manager unavailable")
	}

	if err := l.refreshHeartbeats(ctx); err != nil {
		return err
	}

	claimedAgents, err := l.client.Agent.Query().
		Where(
			entagent.StatusEQ(entagent.StatusClaimed),
			entagent.RuntimePhaseEQ(entagent.RuntimePhaseNone),
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
			l.deleteSession(agentID)
			continue
		}
		if agentItem.Status != entagent.StatusRunning || agentItem.RuntimePhase != entagent.RuntimePhaseReady {
			l.deleteSession(agentID)
			continue
		}
		if agentItem.LastHeartbeatAt != nil && now.Sub(agentItem.LastHeartbeatAt.UTC()) < runtimeHeartbeatInterval {
			continue
		}

		if _, err := l.client.Agent.Update().
			Where(
				entagent.IDEQ(agentID),
				entagent.StatusEQ(entagent.StatusRunning),
				entagent.RuntimePhaseEQ(entagent.RuntimePhaseReady),
			).
			SetLastHeartbeatAt(now).
			Save(ctx); err != nil {
			return fmt.Errorf("refresh heartbeat for agent %s: %w", agentID, err)
		}

		refreshedAgent, err := loadAgentLifecycleState(ctx, l.client, agentID)
		if err != nil {
			return err
		}
		if err := publishAgentLifecycleEvent(
			ctx,
			l.client,
			l.events,
			agentHeartbeatType,
			refreshedAgent,
			lifecycleMessage(agentHeartbeatType, refreshedAgent.Name),
			runtimeEventMetadata(refreshedAgent),
			now,
		); err != nil {
			return err
		}
	}

	return nil
}

func (l *RuntimeLauncher) startCodexSession(ctx context.Context, agentItem *ent.Agent) (*codex.Session, error) {
	if agentItem == nil {
		return nil, fmt.Errorf("agent must not be nil")
	}
	if agentItem.Edges.Provider == nil {
		return nil, fmt.Errorf("agent provider must be loaded")
	}
	if agentItem.Edges.Provider.AdapterType != entagentprovider.AdapterTypeCodexAppServer {
		return nil, fmt.Errorf("unsupported adapter type %s", agentItem.Edges.Provider.AdapterType)
	}

	command, err := provider.ParseAgentCLICommand(agentItem.Edges.Provider.CliCommand)
	if err != nil {
		return nil, fmt.Errorf("parse agent cli command: %w", err)
	}
	workingDirectory, err := provider.ParseAbsolutePath(agentItem.WorkspacePath)
	if err != nil {
		return nil, fmt.Errorf("parse agent workspace path: %w", err)
	}

	processSpec, err := provider.NewAgentCLIProcessSpec(
		command,
		agentItem.Edges.Provider.CliArgs,
		&workingDirectory,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build codex process spec: %w", err)
	}

	adapter, err := codex.NewAdapter(codex.AdapterOptions{ProcessManager: l.processManager})
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
			WorkingDirectory: workingDirectory.String(),
			Model:            agentItem.Edges.Provider.ModelName,
			ServiceName:      "openase",
		},
	})
}

func (l *RuntimeLauncher) storeSession(agentID uuid.UUID, session *codex.Session) {
	l.sessionsMu.Lock()
	defer l.sessionsMu.Unlock()
	l.sessions[agentID] = session
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

func stopSession(ctx context.Context, session *codex.Session) {
	if session == nil {
		return
	}
	_ = session.Stop(ctx)
}
