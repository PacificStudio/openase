package orchestrator

import (
	"context"
	"fmt"
	"strings"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
)

func (s runtimeRecoverySlice) reconcilePauseRequests(ctx context.Context) error {
	l := s.launcher
	pausedAssignments, err := l.selectionSlice().listAssignments(ctx,
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
		if err := l.processSlice().pauseAgent(ctx, assignment); err != nil {
			return err
		}
	}

	return nil
}

func (s runtimeRecoverySlice) reconcileInterruptRequests(ctx context.Context) error {
	l := s.launcher
	assignments, err := l.selectionSlice().listAssignments(ctx,
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
		if err := l.processSlice().interruptAgent(ctx, assignment); err != nil {
			return err
		}
	}
	return nil
}

func (s runtimeRecoverySlice) refreshHeartbeats(ctx context.Context) error {
	l := s.launcher
	runIDs := l.sessionRunIDs()
	if len(runIDs) == 0 {
		return nil
	}

	for _, runID := range runIDs {
		assignment, err := l.selectionSlice().loadAssignmentByRun(ctx, runID)
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

	return nil
}

func (s runtimeRecoverySlice) reconcileTrackerState(ctx context.Context) error {
	l := s.launcher
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

func (s runtimeRecoverySlice) reconcileRuntimeFacts(ctx context.Context) error {
	l := s.launcher
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

func (s runtimeRecoverySlice) reconcileStalledRuntime(ctx context.Context) error {
	l := s.launcher
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
