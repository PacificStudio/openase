package orchestrator

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketreposcope "github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	ticketingdomain "github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

const (
	defaultLaunchTimeout            = 30 * time.Second
	defaultLaunchCleanupTimeout     = 5 * time.Second
	defaultLifecyclePublishTimeout  = 2 * time.Second
	defaultCompletionSummaryTimeout = 45 * time.Second
)

var errExplicitRepoScopeRequired = errors.New("explicit repo scope required for multi-repo project")

type RuntimeLauncher struct {
	client         *ent.Client
	logger         *slog.Logger
	events         provider.EventProvider
	processManager provider.AgentCLIProcessManager
	sshPool        *sshinfra.Pool
	transports     *machinetransport.Resolver
	workflow       *workflowservice.Service
	agentPlatform  runtimeAgentPlatform
	platformAPIURL string
	githubAuth     githubauthservice.TokenResolver
	metrics        provider.MetricsProvider
	now            func() time.Time
	launchTimeout  time.Duration
	eventTimeout   time.Duration

	sessions            *runtimeSessionRegistry
	launches            *runtimeRunTracker
	executions          *runtimeRunTracker
	adapters            *agentAdapterRegistry
	runtime             *RuntimeStateStore
	workspaces          *runtimeWorkspaceProvisioner
	completionSummaries *runtimeCompletionSummaryCoordinator

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

type runtimePlatformAccess struct {
	environment []string
	contract    string
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

	launcher := &RuntimeLauncher{
		client:         client,
		logger:         logger.With("component", "runtime-launcher"),
		events:         events,
		processManager: processManager,
		sshPool:        sshPool,
		transports:     machinetransport.NewResolver(processManager, sshPool),
		workflow:       workflow,
		metrics:        provider.NewNoopMetricsProvider(),
		now:            time.Now,
		launchTimeout:  defaultLaunchTimeout,
		eventTimeout:   defaultLifecyclePublishTimeout,
		sessions:       newRuntimeSessionRegistry(),
		launches:       newRuntimeRunTracker(),
		executions:     newRuntimeRunTracker(),
		adapters:       newDefaultAgentAdapterRegistry(),
		runtime:        NewRuntimeStateStore(),
		tickets:        ticketservice.NewService(ticketrepo.NewEntRepository(client)),
	}
	launcher.tickets.ConfigureSSHPool(sshPool)
	launcher.tickets.ConfigureTransportResolver(launcher.transports)
	launcher.workspaces = newRuntimeWorkspaceProvisioner(client, launcher.logger, sshPool, launcher.now)
	launcher.completionSummaries = newRuntimeCompletionSummaryCoordinator(
		client,
		launcher.logger,
		events,
		launcher.adapters,
		processManager,
		sshPool,
		workflow,
		launcher.now,
		defaultCompletionSummaryTimeout,
	)
	return launcher
}

func (l *RuntimeLauncher) ConfigureRuntimeState(store *RuntimeStateStore) {
	if l == nil || store == nil {
		return
	}
	l.runtime = store
}

func (l *RuntimeLauncher) ConfigurePlatformEnvironment(apiURL string, agentPlatform runtimeAgentPlatform) {
	if l == nil {
		return
	}

	l.platformAPIURL = strings.TrimSpace(apiURL)
	l.agentPlatform = agentPlatform
	l.tickets.ConfigurePlatformEnvironment(apiURL, agentPlatform)
}

func (l *RuntimeLauncher) ConfigureGitHubCredentials(resolver githubauthservice.TokenResolver) {
	if l == nil {
		return
	}
	l.githubAuth = resolver
	if l.workspaces != nil {
		l.workspaces.githubAuth = resolver
	}
}

func (l *RuntimeLauncher) ConfigureMetrics(metrics provider.MetricsProvider) {
	if l == nil || metrics == nil {
		return
	}
	l.metrics = metrics
}

func (l *RuntimeLauncher) ConfigureReverseRuntimeRelay(relay *machinetransport.ReverseRuntimeRelayRegistry) {
	if l == nil || relay == nil || l.transports == nil {
		return
	}
	l.transports.WithReverseRuntimeRelay(relay)
	if l.tickets != nil {
		l.tickets.ConfigureTransportResolver(l.transports)
	}
}

func (l *RuntimeLauncher) RunTick(ctx context.Context) error {
	if l == nil || l.client == nil {
		return fmt.Errorf("runtime launcher unavailable")
	}
	if l.processManager == nil {
		return fmt.Errorf("runtime launcher process manager unavailable")
	}

	if err := l.reconcileInterruptRequests(ctx); err != nil {
		return err
	}
	if err := l.reconcilePauseRequests(ctx); err != nil {
		return err
	}
	if err := l.reconcileRuntimeFacts(ctx); err != nil {
		return err
	}
	if err := l.reconcileTrackerState(ctx); err != nil {
		return err
	}
	if err := l.reconcileStalledRuntime(ctx); err != nil {
		return err
	}
	if err := l.refreshHeartbeats(ctx); err != nil {
		return err
	}
	if err := l.reconcileRunCompletionSummaries(ctx); err != nil {
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

	executions := l.executionRunIDs()
	sessions := l.drainSessions()
	for runID, session := range sessions {
		stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if session != nil {
			_ = session.Stop(stopCtx)
		}
		if err := l.waitForExecutionStop(stopCtx, runID); err != nil {
			l.logger.Warn("wait for execution shutdown", "run_id", runID, "error", err)
		}
		cancel()

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
					SetStatus(entagentrun.StatusTerminated).
					SetTerminalAt(now),
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
			if err := catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, assignment.run.ID, now); err != nil {
				l.logger.Warn("materialize graceful shutdown run usage", "run_id", assignment.run.ID, "error", err)
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
		if assignment.run != nil {
			l.prepareRunCompletionSummaryBestEffort(ctx, assignment.run.ID)
			l.scheduleRunCompletionSummary(assignment.run.ID)
		}
	}

	for _, runID := range executions {
		if _, hadSession := sessions[runID]; hadSession {
			continue
		}
		stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if err := l.waitForExecutionStop(stopCtx, runID); err != nil {
			l.logger.Warn("wait for execution shutdown", "run_id", runID, "error", err)
		}
		cancel()
	}

	return nil
}

func (l *RuntimeLauncher) runLaunch(ctx context.Context, assignment runtimeAssignment) {
	defer l.finishLaunch(assignment.run.ID)

	err := l.launchAgent(ctx, assignment)
	if err == nil {
		return
	}

	logAttrs := []any{
		"agent_id", assignment.agent.ID,
		"run_id", assignment.run.ID,
		"ticket_id", assignment.ticket.ID,
		"error", err,
	}
	if details := runtimeLaunchFailureDetails(err); details != nil {
		if details.stage != "" {
			logAttrs = append(logAttrs, "failure_stage", string(details.stage))
		}
		if details.machineID != uuid.Nil {
			logAttrs = append(logAttrs, "machine_id", details.machineID.String())
		}
		if strings.TrimSpace(details.transportMode) != "" {
			logAttrs = append(logAttrs, "transport_mode", details.transportMode)
		}
		if strings.TrimSpace(details.workspaceRoot) != "" {
			logAttrs = append(logAttrs, "workspace_root", details.workspaceRoot)
		}
	}
	l.logger.Error("launch current run", logAttrs...)
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

	session, launchErr := l.startRuntimeSessionWithTimeout(ctx, assignment)
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
		SetStatus(entagentrun.StatusReady).
		SetRuntimeStartedAt(readyAt).
		SetLastHeartbeatAt(readyAt).
		SetLastError("").
		Save(ctx)
	if err == nil {
		if sessionID, ok := session.SessionID(); ok {
			readyCount, err = l.client.AgentRun.Update().
				Where(
					entagentrun.IDEQ(assignment.run.ID),
					entagentrun.StatusEQ(entagentrun.StatusReady),
				).
				SetSessionID(sessionID).
				Save(ctx)
		}
	}
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
	sessionID, _ := session.SessionID()
	l.runtime.markReady(
		assignment.run.ID,
		assignment.agent.ID,
		assignment.ticket.ID,
		assignment.run.WorkflowID,
		sessionID,
		readyAt,
	)

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

func (l *RuntimeLauncher) startRuntimeSessionWithTimeout(ctx context.Context, assignment runtimeAssignment) (agentSession, error) {
	timeout := l.launchTimeout
	if timeout <= 0 {
		return l.startRuntimeSession(ctx, assignment)
	}

	launchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	type launchResult struct {
		session agentSession
		err     error
	}

	resultCh := make(chan launchResult)
	//nolint:gosec // launch timeout cleanup needs a detached stop context to reclaim late sessions safely.
	go func() {
		session, err := l.startRuntimeSession(launchCtx, assignment)
		select {
		case resultCh <- launchResult{session: session, err: err}:
		case <-launchCtx.Done():
			stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer stopCancel()
			stopSession(stopCtx, session)
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-resultCh:
		return result.session, result.err
	case <-timer.C:
		cancel()
		return nil, fmt.Errorf("start runtime session timed out after %s", timeout)
	case <-ctx.Done():
		cancel()
		return nil, ctx.Err()
	}
}

func (l *RuntimeLauncher) markLaunchFailed(ctx context.Context, agentID uuid.UUID, ticketID uuid.UUID, runID uuid.UUID, launchErr error) error {
	now := l.now().UTC()
	count, err := l.client.AgentRun.Update().
		Where(
			entagentrun.IDEQ(runID),
			entagentrun.StatusEQ(entagentrun.StatusLaunching),
		).
		SetStatus(entagentrun.StatusErrored).
		SetTerminalAt(now).
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
	l.recordLaunchFailureMetric(launchErr)
	if err := catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, runID, now); err != nil {
		return err
	}
	l.tickets.RunLifecycleHookBestEffort(ctx, ticketservice.RunLifecycleHookInput{
		TicketID: ticketID,
		RunID:    runID,
		HookName: infrahook.TicketHookOnError,
	})

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
		mergeRuntimeFailureMetadata(runtimeEventMetadataForState(failedAgent), launchErr),
		now,
	)
	l.prepareRunCompletionSummaryBestEffort(ctx, runID)
	l.scheduleRunCompletionSummary(runID)
	return nil
}

func (l *RuntimeLauncher) recordLaunchFailureMetric(launchErr error) {
	if l == nil || l.metrics == nil {
		return
	}
	stage := string(runtimeLaunchStageProcessStart)
	transportMode := ""
	if details := runtimeLaunchFailureDetails(launchErr); details != nil {
		if details.stage != "" {
			stage = string(details.stage)
		}
		transportMode = strings.TrimSpace(details.transportMode)
	}
	l.metrics.Counter("openase.runtime.launch_failures_total", provider.Tags{
		"failure_stage":  stage,
		"transport_mode": transportMode,
	}).Add(1)
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

func (l *RuntimeLauncher) reconcileInterruptRequests(ctx context.Context) error {
	assignments, err := l.listAssignments(ctx,
		entticket.CurrentRunIDNotNil(),
		entticket.HasCurrentRunWith(
			entagentrun.HasAgentWith(
				entagent.RuntimeControlStateEQ(entagent.RuntimeControlStateInterruptRequested),
			),
			entagentrun.StatusIn(entagentrun.StatusLaunching, entagentrun.StatusReady, entagentrun.StatusExecuting),
		),
	)
	if err != nil {
		return fmt.Errorf("list interrupt requests: %w", err)
	}

	for _, assignment := range assignments {
		if err := l.interruptAgent(ctx, assignment); err != nil {
			return err
		}
	}
	return nil
}

func (l *RuntimeLauncher) refreshHeartbeats(ctx context.Context) error {
	runIDs := l.sessionRunIDs()

	if len(runIDs) == 0 {
		return nil
	}

	now := l.now().UTC()
	for _, runID := range runIDs {
		assignment, err := l.loadAssignmentByRun(ctx, runID)
		if err != nil {
			stopSession(context.Background(), l.loadSession(runID))
			l.deleteSession(runID)
			l.runtime.delete(runID)
			continue
		}
		if assignment.agent == nil || assignment.run == nil || assignment.ticket == nil ||
			assignment.agent.RuntimeControlState != entagent.RuntimeControlStateActive ||
			(assignment.run.Status != entagentrun.StatusReady && assignment.run.Status != entagentrun.StatusExecuting) {
			stopSession(context.Background(), l.loadSession(runID))
			l.deleteSession(runID)
			l.runtime.delete(runID)
			continue
		}
	}

	_ = now
	return nil
}

func (l *RuntimeLauncher) reconcileTrackerState(ctx context.Context) error {
	for _, snapshot := range l.runtime.snapshots() {
		if snapshot.PendingRuntimeFact != nil {
			continue
		}

		ticket, err := l.reloadExecutionTicket(ctx, snapshot.TicketID)
		if err != nil {
			return fmt.Errorf("reload ticket %s during tracker reconciliation: %w", snapshot.TicketID, err)
		}

		switch classifyRuntimeTicket(ticket, snapshot.RunID, snapshot.WorkflowID) {
		case runtimeTicketActive:
			continue
		default:
			if err := l.releaseExecutionOwnership(ctx, snapshot.RunID, snapshot.AgentID, ticket); err != nil {
				return fmt.Errorf("release run %s during tracker reconciliation: %w", snapshot.RunID, err)
			}
		}
	}

	return nil
}

func (l *RuntimeLauncher) reconcileRuntimeFacts(ctx context.Context) error {
	for _, snapshot := range l.runtime.snapshots() {
		if snapshot.PendingRuntimeFact == nil || l.executionActive(snapshot.RunID) {
			continue
		}

		ticket, err := l.reloadExecutionTicket(ctx, snapshot.TicketID)
		if err != nil {
			return fmt.Errorf("reload ticket %s during runtime fact reconciliation: %w", snapshot.TicketID, err)
		}

		if snapshot.PendingRuntimeFact.Kind != runtimeFactSessionExited {
			continue
		}

		suppressFailure, err := l.shouldSuppressExecutionFailure(ctx, snapshot.RunID, snapshot.TicketID)
		if err != nil {
			return fmt.Errorf("check runtime fact failure suppression for run %s: %w", snapshot.RunID, err)
		}
		if suppressFailure {
			stopSession(context.Background(), l.loadSession(snapshot.RunID))
			l.deleteSession(snapshot.RunID)
			l.runtime.delete(snapshot.RunID)
			continue
		}

		disposition := classifyRuntimeTicket(ticket, snapshot.RunID, snapshot.WorkflowID)
		if disposition == runtimeTicketActive {
			if trimmed := strings.TrimSpace(snapshot.PendingRuntimeFact.Message); trimmed != "" {
				l.handleExecutionFailure(ctx, snapshot.RunID, snapshot.AgentID, snapshot.TicketID, fmt.Errorf("%s", trimmed))
				continue
			}
			if err := l.scheduleContinuation(ctx, snapshot.RunID, snapshot.AgentID, snapshot.TicketID); err != nil {
				return fmt.Errorf("schedule continuation for run %s after subprocess exit: %w", snapshot.RunID, err)
			}
			continue
		}
		if err := l.releaseExecutionOwnership(ctx, snapshot.RunID, snapshot.AgentID, ticket); err != nil {
			return fmt.Errorf("release run %s after runtime fact reconciliation: %w", snapshot.RunID, err)
		}
	}

	return nil
}

func (l *RuntimeLauncher) reconcileStalledRuntime(ctx context.Context) error {
	now := l.now().UTC()
	for _, snapshot := range l.runtime.snapshots() {
		if snapshot.PendingRuntimeFact != nil {
			continue
		}
		if l.loadSession(snapshot.RunID) == nil {
			continue
		}

		ticket, err := l.reloadExecutionTicket(ctx, snapshot.TicketID)
		if err != nil {
			return fmt.Errorf("reload ticket %s during stall reconciliation: %w", snapshot.TicketID, err)
		}
		timeout := stallTimeoutForWorkflow(ticket.Edges.Workflow)
		lastCodexAt := snapshot.StartedAt
		if !snapshot.LastCodexTimestamp.IsZero() {
			lastCodexAt = snapshot.LastCodexTimestamp
		}
		if age := now.Sub(lastCodexAt); age <= timeout {
			continue
		}

		stopSession(context.Background(), l.loadSession(snapshot.RunID))
		l.deleteSession(snapshot.RunID)
		l.runtime.delete(snapshot.RunID)

		_, _, err = releaseStalledClaim(
			ctx,
			l.client,
			ticket.ProjectID,
			snapshot.TicketID,
			snapshot.RunID,
			snapshot.AgentID,
			ticket.AttemptCount,
			ticket.ConsecutiveErrors,
			ticket.StallCount,
			now,
			"runtime_launcher",
			"runtime stalled based on last codex event timestamp",
			l.events,
		)
		if err != nil {
			return fmt.Errorf("release stalled runtime claim for run %s: %w", snapshot.RunID, err)
		}
	}

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
			SetStatus(entagentrun.StatusTerminated).
			SetTerminalAt(pausedAt),
	).Save(ctx)
	if err != nil {
		return fmt.Errorf("mark agent %s paused: %w", assignment.agent.ID, err)
	}
	if pausedCount == 0 {
		return nil
	}
	if err := catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, assignment.run.ID, pausedAt); err != nil {
		return err
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

func (l *RuntimeLauncher) interruptAgent(ctx context.Context, assignment runtimeAssignment) error {
	if assignment.agent == nil || assignment.run == nil || assignment.ticket == nil {
		return nil
	}

	session := l.loadSession(assignment.run.ID)
	if session != nil {
		stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		stopErr := session.Stop(stopCtx)
		cancel()
		if stopErr != nil {
			return fmt.Errorf("interrupt runtime session for run %s: %w", assignment.run.ID, stopErr)
		}
		l.deleteSession(assignment.run.ID)
	}
	l.runtime.delete(assignment.run.ID)

	interruptedAt := l.now().UTC()
	tx, err := l.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start interrupt tx for run %s: %w", assignment.run.ID, err)
	}
	defer rollback(tx)

	runCount, err := clearRuntimeState(
		tx.AgentRun.Update().
			Where(
				entagentrun.IDEQ(assignment.run.ID),
				entagentrun.StatusIn(entagentrun.StatusLaunching, entagentrun.StatusReady, entagentrun.StatusExecuting),
			).
			SetStatus(entagentrun.StatusInterrupted).
			SetTerminalAt(interruptedAt),
	).Save(ctx)
	if err != nil {
		return fmt.Errorf("mark agent %s interrupted: %w", assignment.agent.ID, err)
	}

	ticketUpdate := ticketrepo.ResetRetryBaseline(tx.Ticket.UpdateOneID(assignment.ticket.ID), assignment.ticket)
	if assignment.ticket.CurrentRunID != nil && *assignment.ticket.CurrentRunID == assignment.run.ID {
		ticketUpdate.ClearCurrentRunID()
	}
	ticketUpdate.
		SetRetryPaused(true).
		SetPauseReason(ticketingdomain.PauseReasonUserInterrupted.String())
	ticketUpdate.ClearNextRetryAt()
	if _, err := ticketUpdate.Save(ctx); err != nil {
		return fmt.Errorf("pause ticket %s after interrupt: %w", assignment.ticket.ID, err)
	}

	if _, err := tx.Agent.UpdateOneID(assignment.agent.ID).
		SetRuntimeControlState(entagent.RuntimeControlStateActive).
		Save(ctx); err != nil {
		return fmt.Errorf("reset agent %s control after interrupt: %w", assignment.agent.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit interrupt tx for run %s: %w", assignment.run.ID, err)
	}
	if runCount == 0 {
		return nil
	}
	if err := catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, assignment.run.ID, interruptedAt); err != nil {
		return err
	}
	if err := l.recordAgentStep(
		ctx,
		assignment.agent.ProjectID,
		assignment.agent.ID,
		assignment.ticket.ID,
		assignment.run.ID,
		"interrupted",
		"Interrupted by operator request.",
		nil,
	); err != nil {
		return fmt.Errorf("record interrupt step for run %s: %w", assignment.run.ID, err)
	}

	interruptedAgent, err := loadAgentLifecycleState(ctx, l.client, assignment.agent.ID, &assignment.run.ID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentInterruptedType,
		interruptedAgent,
		lifecycleMessage(agentInterruptedType, interruptedAgent.agent.Name),
		runtimeEventMetadataForState(interruptedAgent),
		interruptedAt,
	)
	l.prepareRunCompletionSummaryBestEffort(ctx, assignment.run.ID)
	l.scheduleRunCompletionSummary(assignment.run.ID)
	return nil
}

func (l *RuntimeLauncher) startRuntimeSession(ctx context.Context, assignment runtimeAssignment) (agentSession, error) {
	launchContext, err := l.loadLaunchContext(ctx, assignment.agent.ID, assignment.ticket.ID)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(catalogdomain.Machine{}, "", runtimeLaunchStageContext, err)
	}

	machine, remote, err := l.resolveLaunchMachine(ctx, launchContext)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(catalogdomain.Machine{}, "", runtimeLaunchStageResolveMachine, err)
	}

	session, err := l.startRuntimeSessionOnMachine(ctx, assignment, launchContext, machine, remote)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (l *RuntimeLauncher) startRuntimeSessionOnMachine(
	ctx context.Context,
	assignment runtimeAssignment,
	launchContext runtimeLaunchContext,
	machine catalogdomain.Machine,
	remote bool,
) (agentSession, error) {
	workspaceRoot := ""
	if remote && machine.WorkspaceRoot != nil {
		workspaceRoot = strings.TrimSpace(*machine.WorkspaceRoot)
	}
	if remote && machine.ConnectionMode == catalogdomain.MachineConnectionModeSSH {
		return nil, wrapRuntimeLaunchFailure(
			machine,
			workspaceRoot,
			runtimeLaunchStageTransportResolve,
			fmt.Errorf("ssh runtime execution is no longer supported for machine %s; migrate the machine to websocket execution and use SSH only for bootstrap or diagnostics", machine.Name),
		)
	}

	commandString := launchContext.agent.Edges.Provider.CliCommand
	if machine.AgentCLIPath != nil {
		commandString = *machine.AgentCLIPath
	}

	command, err := provider.ParseAgentCLICommand(commandString)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageProcessStart, fmt.Errorf("parse agent cli command: %w", err))
	}
	environment := buildAgentCLIEnvironment(machine.EnvVars, launchContext.agent.Edges.Provider.AuthConfig)
	platformAccess, err := l.buildAgentPlatformAccess(ctx, launchContext)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageContext, err)
	}
	environment = append(environment, platformAccess.environment...)
	githubEnvironment, err := l.buildGitHubOutboundEnvironment(ctx, launchContext.project.ID, environment)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageContext, err)
	}
	environment = append(environment, githubEnvironment...)
	if !remote {
		launcherEnvironment, err := buildLocalOpenASEEnvironment()
		if err != nil {
			return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageContext, err)
		}
		environment = append(environment, launcherEnvironment...)
	}
	if requiresMachineCodexReady(command, environment) {
		if ready, reason, ok := machineCodexReady(machine.Resources); ok && !ready {
			return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageAgentCLIPreflight, fmt.Errorf("machine %s codex environment not ready: %s", machine.Name, reason))
		}
	}

	workspaceItem, err := l.prepareTicketWorkspace(ctx, assignment.run.ID, launchContext, machine, remote)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, classifyRuntimeLaunchWorkspaceStage(err), err)
	}
	if err := l.tickets.RunLifecycleHook(ctx, ticketservice.RunLifecycleHookInput{
		TicketID: assignment.ticket.ID,
		RunID:    assignment.run.ID,
		HookName: infrahook.TicketHookOnClaim,
		Blocking: true,
	}); err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceItem.Path, runtimeLaunchStageHookOnClaim, fmt.Errorf("run ticket on_claim hooks: %w", err))
	}

	workingDirectoryValue := resolveAgentWorkingDirectory(launchContext, workspaceItem)
	var runtimeSnapshot workflowservice.RuntimeSnapshot
	if l.workflow != nil {
		if launchContext.ticket.WorkflowID != nil {
			runtimeSnapshot, err = l.materializeRuntimeSnapshot(
				ctx,
				assignment.run.ID,
				*launchContext.ticket.WorkflowID,
				machine,
				workingDirectoryValue,
				string(launchContext.agent.Edges.Provider.AdapterType),
				remote,
			)
			if err != nil {
				return nil, wrapRuntimeLaunchFailure(machine, workspaceItem.Path, runtimeLaunchStageRuntimeSnapshot, fmt.Errorf("materialize runtime snapshot: %w", err))
			}
		}
	}
	if err := l.runRemoteRuntimePreflight(ctx, machine, remote, workingDirectoryValue, command.String(), environment); err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectoryValue, classifyRuntimeLaunchPreflightStage(err), err)
	}
	workingDirectory, err := provider.ParseAbsolutePath(workingDirectoryValue)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectoryValue, runtimeLaunchStageWorkspaceRoot, fmt.Errorf("parse agent workspace path: %w", err))
	}
	developerInstructions, err := l.buildDeveloperInstructions(
		ctx,
		launchContext,
		machine,
		workingDirectory.String(),
		runtimeSnapshot,
		platformAccess.contract,
	)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageBuildInstructions, err)
	}
	if err := l.tickets.RunLifecycleHook(ctx, ticketservice.RunLifecycleHookInput{
		TicketID: assignment.ticket.ID,
		RunID:    assignment.run.ID,
		HookName: infrahook.TicketHookOnStart,
		Blocking: true,
	}); err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageHookOnStart, fmt.Errorf("run ticket on_start hooks: %w", err))
	}

	processManager := l.processManager
	if l.transports != nil {
		transport, transportErr := l.transports.Resolve(machine)
		if transportErr != nil {
			return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageTransportResolve, transportErr)
		}
		processManager = machinetransport.NewProcessManager(transport, machine)
	}

	processSpec, err := provider.NewAgentCLIProcessSpec(
		command,
		launchContext.agent.Edges.Provider.CliArgs,
		&workingDirectory,
		environment,
	)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageProcessStart, fmt.Errorf("build agent process spec: %w", err))
	}

	adapter, err := l.adapters.adapterFor(launchContext.agent.Edges.Provider.AdapterType)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageProcessStart, err)
	}

	session, err := adapter.Start(ctx, agentSessionStartSpec{
		Process:               processSpec,
		ProcessManager:        processManager,
		WorkingDirectory:      workingDirectory.String(),
		Model:                 launchContext.agent.Edges.Provider.ModelName,
		PermissionProfile:     catalogdomain.AgentProviderPermissionProfile(launchContext.agent.Edges.Provider.PermissionProfile),
		DeveloperInstructions: developerInstructions,
		TurnTitle:             fmt.Sprintf("%s: %s", launchContext.ticket.Identifier, launchContext.ticket.Title),
	})
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageProcessStart, err)
	}
	return session, nil
}

func (l *RuntimeLauncher) buildAgentPlatformAccess(ctx context.Context, launchContext runtimeLaunchContext) (runtimePlatformAccess, error) {
	if l == nil || l.agentPlatform == nil {
		return runtimePlatformAccess{}, nil
	}
	if launchContext.agent == nil || launchContext.project == nil || launchContext.ticket == nil {
		return runtimePlatformAccess{}, fmt.Errorf("runtime launch context is incomplete for platform environment")
	}

	scopeWhitelist := agentplatform.ScopeWhitelist{}
	if launchContext.ticket.WorkflowID != nil {
		workflowItem, err := l.client.Workflow.Get(ctx, *launchContext.ticket.WorkflowID)
		if err != nil {
			return runtimePlatformAccess{}, fmt.Errorf("load workflow %s for agent platform token: %w", *launchContext.ticket.WorkflowID, err)
		}
		scopeWhitelist = agentplatform.ScopeWhitelist{
			Configured: len(workflowItem.PlatformAccessAllowed) > 0,
			Scopes:     append([]string(nil), workflowItem.PlatformAccessAllowed...),
		}
	}

	issued, err := l.agentPlatform.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:        launchContext.agent.ID,
		ProjectID:      launchContext.project.ID,
		TicketID:       launchContext.ticket.ID,
		ScopeWhitelist: scopeWhitelist,
	})
	if err != nil {
		return runtimePlatformAccess{}, fmt.Errorf("issue agent platform token: %w", err)
	}
	contractScopes := issued.Scopes
	if len(contractScopes) == 0 {
		contractScopes = agentplatform.DefaultScopesForPrincipalKind(agentplatform.PrincipalKindTicketAgent)
	}

	contractInput := agentplatform.RuntimeContractInput{
		PrincipalKind: agentplatform.PrincipalKindTicketAgent,
		ProjectID:     launchContext.project.ID,
		TicketID:      launchContext.ticket.ID,
		APIURL:        l.platformAPIURL,
		Token:         issued.Token,
		Scopes:        contractScopes,
	}
	return runtimePlatformAccess{
		environment: agentplatform.BuildRuntimeEnvironment(contractInput),
		contract:    agentplatform.BuildCapabilityContract(contractInput),
	}, nil
}

func (l *RuntimeLauncher) ticketRuntimePlatformContract(
	launchContext runtimeLaunchContext,
	scopes []string,
) string {
	if launchContext.project == nil || launchContext.ticket == nil {
		return ""
	}
	return agentplatform.BuildCapabilityContract(agentplatform.RuntimeContractInput{
		PrincipalKind: agentplatform.PrincipalKindTicketAgent,
		ProjectID:     launchContext.project.ID,
		TicketID:      launchContext.ticket.ID,
		APIURL:        l.platformAPIURL,
		Token:         "<runtime-injected>",
		Scopes:        scopes,
	})
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

func (l *RuntimeLauncher) runRemoteRuntimePreflight(
	ctx context.Context,
	machine catalogdomain.Machine,
	remote bool,
	workingDirectory string,
	command string,
	environment []string,
) error {
	if !remote || l == nil || l.transports == nil {
		return nil
	}

	resolved, err := l.transports.ResolveRuntime(machine)
	if err != nil {
		return err
	}
	if resolved.Execution.Runtime == nil ||
		!resolved.Execution.Runtime.SupportsAll(
			catalogdomain.MachineTransportCapabilityWorkspacePrepare,
			catalogdomain.MachineTransportCapabilityArtifactSync,
			catalogdomain.MachineTransportCapabilityProcessStreaming,
		) ||
		resolved.CommandSessionExecutor() == nil {
		return nil
	}

	return machinetransport.RunRemoteRuntimePreflight(ctx, resolved.CommandSessionExecutor(), machine, machinetransport.RuntimePreflightSpec{
		WorkingDirectory: workingDirectory,
		AgentCommand:     command,
		Environment:      environment,
	})
}

func (l *RuntimeLauncher) buildGitHubOutboundEnvironment(
	ctx context.Context,
	projectID uuid.UUID,
	baseEnvironment []string,
) ([]string, error) {
	if l == nil || l.githubAuth == nil || projectID == uuid.Nil {
		return nil, nil
	}

	resolved, err := l.githubAuth.ResolveProjectCredential(ctx, projectID)
	if err != nil {
		if errors.Is(err, githubauthservice.ErrCredentialNotConfigured) {
			return nil, nil
		}
		return nil, fmt.Errorf("resolve project GitHub credential for agent environment: %w", err)
	}

	token := strings.TrimSpace(resolved.Token)
	if token == "" {
		return nil, nil
	}

	return buildGitHubTokenEnvironment(baseEnvironment, token), nil
}

func buildGitHubTokenEnvironment(baseEnvironment []string, token string) []string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return nil
	}

	environment := make([]string, 0, 6)
	environment = append(environment, "GH_TOKEN="+trimmed)
	authHeader := "AUTHORIZATION: basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+trimmed))

	existingConfigCount := 0
	if rawCount, ok := provider.LookupEnvironmentValue(baseEnvironment, "GIT_CONFIG_COUNT"); ok {
		if parsedCount, err := strconv.Atoi(strings.TrimSpace(rawCount)); err == nil && parsedCount >= 0 {
			existingConfigCount = parsedCount
		}
	}

	environment = append(
		environment,
		"GIT_CONFIG_COUNT="+strconv.Itoa(existingConfigCount+2),
		"GIT_CONFIG_KEY_"+strconv.Itoa(existingConfigCount)+"=http.https://github.com/.extraheader",
		"GIT_CONFIG_VALUE_"+strconv.Itoa(existingConfigCount)+"="+authHeader,
		"GIT_CONFIG_KEY_"+strconv.Itoa(existingConfigCount+1)+"=credential.helper",
		"GIT_CONFIG_VALUE_"+strconv.Itoa(existingConfigCount+1)+"=",
	)
	return environment
}

func (l *RuntimeLauncher) refreshRemoteWorkspaceSkills(
	ctx context.Context,
	projectID uuid.UUID,
	workflowID *uuid.UUID,
	machine catalogdomain.Machine,
	workspaceRoot string,
	adapterType string,
) error {
	if l == nil || l.workflow == nil {
		return nil
	}
	if l.transports == nil {
		return fmt.Errorf("machine transport resolver unavailable for remote machine %s", machine.Name)
	}

	skillNames, err := l.resolveLaunchSkillNames(ctx, projectID, workflowID)
	if err != nil {
		return err
	}
	target, err := workflowservice.ResolveSkillTargetForRuntime(workspaceRoot, adapterType)
	if err != nil {
		return err
	}

	resolved, err := l.transports.ResolveRuntime(machine)
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

	command := buildRemoteRefreshSkillsCommand(workspaceRoot, target.SkillsDir, skillNames)
	if output, err := session.CombinedOutput(command); err != nil {
		return fmt.Errorf("refresh remote skills: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (l *RuntimeLauncher) resolveLaunchSkillNames(
	ctx context.Context,
	projectID uuid.UUID,
	workflowID *uuid.UUID,
) ([]string, error) {
	items, err := l.workflow.ListSkills(ctx, projectID)
	if err != nil {
		return nil, err
	}

	selected := make([]string, 0, len(items))
	for _, item := range items {
		if !item.IsEnabled {
			continue
		}
		if workflowID == nil {
			selected = append(selected, item.Name)
			continue
		}
		if item.Name == "openase-platform" {
			selected = append(selected, item.Name)
			continue
		}
		for _, binding := range item.BoundWorkflows {
			if binding.ID == *workflowID {
				selected = append(selected, item.Name)
				break
			}
		}
	}
	sort.Strings(selected)
	return selected, nil
}

func buildRemoteRefreshSkillsCommand(workspaceRoot string, skillsDir string, skillNames []string) string {
	lines := make([]string, 0, 3+len(skillNames))
	lines = append(lines,
		"set -eu",
		"rm -rf "+sshinfra.ShellQuote(skillsDir),
		"mkdir -p "+sshinfra.ShellQuote(skillsDir),
	)

	for _, skillName := range skillNames {
		src := filepath.Join(workspaceRoot, ".openase", "skills", skillName)
		dst := filepath.Join(skillsDir, skillName)
		lines = append(lines,
			"if [ -d "+sshinfra.ShellQuote(src)+" ]; then cp -R "+sshinfra.ShellQuote(src)+" "+sshinfra.ShellQuote(dst)+"; fi",
		)
	}

	return strings.Join(lines, "\n")
}

func (l *RuntimeLauncher) buildDeveloperInstructions(
	ctx context.Context,
	launchContext runtimeLaunchContext,
	machine catalogdomain.Machine,
	workspace string,
	runtimeSnapshot workflowservice.RuntimeSnapshot,
	platformContract string,
) (string, error) {
	if l == nil || l.workflow == nil || launchContext.ticket == nil || launchContext.ticket.WorkflowID == nil {
		return "", nil
	}
	if launchContext.agent == nil || launchContext.project == nil {
		return "", fmt.Errorf("runtime launch context is incomplete for harness injection")
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

	rendered, err := workflowservice.RenderHarnessBody(runtimeSnapshot.Workflow.Content, data)
	if err != nil {
		return "", fmt.Errorf("render workflow harness for agent launch: %w", err)
	}

	rendered = strings.TrimSpace(rendered)
	platformContract = strings.TrimSpace(platformContract)
	if platformContract == "" {
		return rendered, nil
	}
	if rendered == "" {
		return platformContract, nil
	}
	return rendered + "\n\n" + platformContract, nil
}

type runtimeLaunchContext struct {
	agent        *ent.Agent
	project      *ent.Project
	ticket       *ent.Ticket
	projectRepos []*ent.ProjectRepo
	ticketScopes []*ent.TicketRepoScope
}

type repoWorkspacePlan struct {
	RepoID       uuid.UUID
	RepoName     string
	WorkspaceDir string
	BranchName   string
	HeadCommit   string
	Input        workspaceinfra.RepoInput
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
			scopeQuery.Order(entticketreposcope.ByRepoID())
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
) (workspaceinfra.SetupRequest, []repoWorkspacePlan, error) {
	if launchContext.project == nil || launchContext.project.Edges.Organization == nil {
		return workspaceinfra.SetupRequest{}, nil, fmt.Errorf("project organization must be loaded for ticket workspace derivation")
	}

	workspaceRoot, err := resolveWorkspaceRoot(machine, remote)
	if err != nil {
		return workspaceinfra.SetupRequest{}, nil, err
	}

	repoPlans, err := buildWorkspaceRepoPlans(
		launchContext.ticket.Identifier,
		launchContext.projectRepos,
		launchContext.ticketScopes,
	)
	if err != nil {
		return workspaceinfra.SetupRequest{}, nil, err
	}

	repoInputs := make([]workspaceinfra.RepoInput, 0, len(repoPlans))
	for _, plan := range repoPlans {
		repoInputs = append(repoInputs, plan.Input)
	}
	request, err := workspaceinfra.ParseSetupRequest(workspaceinfra.SetupInput{
		WorkspaceRoot:    workspaceRoot,
		OrganizationSlug: launchContext.project.Edges.Organization.Slug,
		ProjectSlug:      launchContext.project.Slug,
		AgentName:        launchContext.agent.Name,
		TicketIdentifier: launchContext.ticket.Identifier,
		Repos:            repoInputs,
	})
	if err != nil {
		return workspaceinfra.SetupRequest{}, nil, fmt.Errorf("build ticket workspace request: %w", err)
	}

	return request, repoPlans, nil
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
	request, _, err := buildWorkspaceRequest(launchContext, machine, remote)
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

func buildWorkspaceRepoPlans(
	ticketIdentifier string,
	projectRepos []*ent.ProjectRepo,
	ticketScopes []*ent.TicketRepoScope,
) ([]repoWorkspacePlan, error) {
	selectedRepos, err := selectLaunchContextProjectRepos(projectRepos, ticketScopes)
	if err != nil {
		return nil, err
	}
	scopeByRepoID := make(map[uuid.UUID]*ent.TicketRepoScope, len(ticketScopes))
	for _, scope := range ticketScopes {
		scopeByRepoID[scope.RepoID] = scope
	}

	plans := make([]repoWorkspacePlan, 0, len(selectedRepos))
	for _, repo := range selectedRepos {
		workspaceDirname := resolvedWorkspaceDirname(repo)
		effectiveBranchName := ticketingdomain.DefaultRepoWorkBranch(ticketIdentifier)
		input := workspaceinfra.RepoInput{
			Name:          repo.Name,
			RepositoryURL: strings.TrimSpace(repo.RepositoryURL),
			DefaultBranch: repo.DefaultBranch,
		}
		if workspaceDirname != strings.TrimSpace(repo.Name) {
			input.WorkspaceDirname = &workspaceDirname
		}
		if scope, ok := scopeByRepoID[repo.ID]; ok {
			branchOverride := ticketingdomain.NormalizeRepoWorkBranchOverride(scope.BranchName)
			effectiveBranchName = ticketingdomain.ResolveRepoWorkBranch(ticketIdentifier, scope.BranchName)
			if branchOverride != "" {
				input.BranchName = &branchOverride
			}
		}
		plans = append(plans, repoWorkspacePlan{
			RepoID:       repo.ID,
			RepoName:     repo.Name,
			WorkspaceDir: workspaceDirname,
			BranchName:   effectiveBranchName,
			Input:        input,
		})
	}

	return plans, nil
}

func repoPlansWithPreparedHeads(
	repoPlans []repoWorkspacePlan,
	preparedRepos []workspaceinfra.PreparedRepo,
) []repoWorkspacePlan {
	if len(repoPlans) == 0 || len(preparedRepos) == 0 {
		return repoPlans
	}

	headByDir := make(map[string]string, len(preparedRepos))
	for _, repo := range preparedRepos {
		headByDir[repo.WorkspaceDirname] = strings.TrimSpace(repo.HeadCommit)
	}

	updated := append([]repoWorkspacePlan(nil), repoPlans...)
	for index := range updated {
		if headCommit := headByDir[updated[index].WorkspaceDir]; headCommit != "" {
			updated[index].HeadCommit = headCommit
		}
	}
	return updated
}

func selectLaunchContextProjectRepos(
	projectRepos []*ent.ProjectRepo,
	ticketScopes []*ent.TicketRepoScope,
) ([]*ent.ProjectRepo, error) {
	if len(ticketScopes) == 0 {
		switch len(projectRepos) {
		case 0:
			return nil, nil
		case 1:
			return projectRepos, nil
		default:
			return nil, errExplicitRepoScopeRequired
		}
	}

	scopeByRepoID := make(map[uuid.UUID]struct{}, len(ticketScopes))
	for _, scope := range ticketScopes {
		scopeByRepoID[scope.RepoID] = struct{}{}
	}

	selectedRepos := make([]*ent.ProjectRepo, 0, len(scopeByRepoID))
	for _, repo := range projectRepos {
		if repo == nil {
			continue
		}
		if _, ok := scopeByRepoID[repo.ID]; ok {
			selectedRepos = append(selectedRepos, repo)
		}
	}
	return selectedRepos, nil
}

func resolvedWorkspaceDirname(repo *ent.ProjectRepo) string {
	if repo == nil {
		return ""
	}
	if workspaceDirname := strings.TrimSpace(repo.WorkspaceDirname); workspaceDirname != "" {
		return workspaceDirname
	}
	return strings.TrimSpace(repo.Name)
}

func resolveAgentWorkingDirectory(_ runtimeLaunchContext, workspaceItem workspaceinfra.Workspace) string {
	if len(workspaceItem.Repos) == 0 {
		return workspaceItem.Path
	}

	if len(workspaceItem.Repos) == 1 {
		return workspaceItem.Repos[0].Path
	}

	return workspaceItem.Path
}

func projectRepoWorkspaceDirname(repo *ent.ProjectRepo) string {
	if repo == nil {
		return ""
	}
	if workspaceDirname := strings.TrimSpace(repo.WorkspaceDirname); workspaceDirname != "" {
		return workspaceDirname
	}

	return repo.Name
}

func mapRuntimeMachine(item *ent.Machine) catalogdomain.Machine {
	return catalogdomain.Machine{
		ID:                 item.ID,
		OrganizationID:     item.OrganizationID,
		Name:               item.Name,
		Host:               item.Host,
		Port:               item.Port,
		SSHUser:            optionalRuntimeString(item.SSHUser),
		SSHKeyPath:         optionalRuntimeString(item.SSHKeyPath),
		Description:        item.Description,
		Labels:             append([]string(nil), item.Labels...),
		Status:             catalogdomain.MachineStatus(item.Status),
		ConnectionMode:     mapStoredRuntimeConnectionMode(item),
		AdvertisedEndpoint: optionalRuntimeString(item.AdvertisedEndpoint),
		WorkspaceRoot:      optionalRuntimeString(item.WorkspaceRoot),
		AgentCLIPath:       optionalRuntimeString(item.AgentCliPath),
		EnvVars:            append([]string(nil), item.EnvVars...),
		Resources:          cloneResourceMap(item.Resources),
		DaemonStatus: catalogdomain.MachineDaemonStatus{
			Registered:       item.DaemonRegistered,
			LastRegisteredAt: cloneRuntimeTime(item.DaemonLastRegisteredAt),
			CurrentSessionID: optionalRuntimeString(item.DaemonSessionID),
			SessionState:     catalogdomain.MachineTransportSessionState(item.DaemonSessionState),
		},
		ChannelCredential: catalogdomain.MachineChannelCredential{
			Kind:          catalogdomain.MachineChannelCredentialKind(item.ChannelCredentialKind),
			TokenID:       optionalRuntimeString(item.ChannelTokenID),
			CertificateID: optionalRuntimeString(item.ChannelCertificateID),
		},
	}
}

func mapStoredRuntimeConnectionMode(item *ent.Machine) catalogdomain.MachineConnectionMode {
	if item == nil {
		return ""
	}
	if item.Host == catalogdomain.LocalMachineHost || item.Name == catalogdomain.LocalMachineName {
		return catalogdomain.MachineConnectionModeLocal
	}
	mode, err := catalogdomain.ParseStoredMachineConnectionMode(string(item.ConnectionMode), item.Host)
	if err != nil {
		return catalogdomain.MachineConnectionModeWSListener
	}
	return mode
}

func optionalRuntimeString(raw string) *string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	value := raw
	return &value
}

func cloneRuntimeTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
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
		Resources:     cloneResourceMap(machine.Resources),
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
			Resources:   cloneResourceMap(item.Resources),
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

func (l *RuntimeLauncher) waitForExecutionStop(ctx context.Context, runID uuid.UUID) error {
	if l == nil {
		return nil
	}
	if !l.executionActive(runID) {
		return nil
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for l.executionActive(runID) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}

	return nil
}

func (l *RuntimeLauncher) launchContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	if timeout <= 0 {
		return base, func() {}
	}
	//nolint:gosec // Cancel ownership is intentionally transferred to callers of launchContext.
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

func stopSession(ctx context.Context, session agentSession) {
	if session == nil {
		return
	}
	_ = session.Stop(ctx)
}
