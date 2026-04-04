package orchestrator

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

type runtimeFactKind string

const (
	runtimeFactSessionExited runtimeFactKind = "session_exited"
)

type runtimeFactSnapshot struct {
	Kind       runtimeFactKind
	ObservedAt time.Time
	Message    string
}

type runtimeRunSnapshot struct {
	RunID              uuid.UUID
	AgentID            uuid.UUID
	TicketID           uuid.UUID
	WorkflowID         uuid.UUID
	StartedAt          time.Time
	LastCodexTimestamp time.Time
	LastCodexEvent     string
	SessionID          string
	TurnCount          int
	PendingRuntimeFact *runtimeFactSnapshot
}

type RuntimeStateStore struct {
	mu   sync.Mutex
	runs map[uuid.UUID]runtimeRunSnapshot
}

func NewRuntimeStateStore() *RuntimeStateStore {
	return &RuntimeStateStore{runs: map[uuid.UUID]runtimeRunSnapshot{}}
}

func (s *RuntimeStateStore) markReady(runID uuid.UUID, agentID uuid.UUID, ticketID uuid.UUID, workflowID uuid.UUID, sessionID string, startedAt time.Time) {
	if s == nil {
		return
	}
	startedAt = startedAt.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[runID] = runtimeRunSnapshot{
		RunID:              runID,
		AgentID:            agentID,
		TicketID:           ticketID,
		WorkflowID:         workflowID,
		StartedAt:          startedAt,
		LastCodexTimestamp: startedAt,
		LastCodexEvent:     string(codex.EventTypeTurnStarted),
		SessionID:          strings.TrimSpace(sessionID),
	}
}

func (s *RuntimeStateStore) recordTurnStart(runID uuid.UUID, turnCount int, observedAt time.Time) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.runs[runID]
	if !ok {
		return
	}
	snapshot.TurnCount = turnCount
	snapshot.LastCodexTimestamp = observedAt.UTC()
	snapshot.LastCodexEvent = string(codex.EventTypeTurnStarted)
	snapshot.PendingRuntimeFact = nil
	s.runs[runID] = snapshot
}

func (s *RuntimeStateStore) recordCodexEvent(runID uuid.UUID, eventType string, observedAt time.Time) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.runs[runID]
	if !ok {
		return
	}
	snapshot.LastCodexTimestamp = observedAt.UTC()
	snapshot.LastCodexEvent = strings.TrimSpace(eventType)
	s.runs[runID] = snapshot
}

func (s *RuntimeStateStore) recordRuntimeFact(runID uuid.UUID, kind runtimeFactKind, observedAt time.Time, message string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.runs[runID]
	if !ok {
		return
	}
	snapshot.PendingRuntimeFact = &runtimeFactSnapshot{
		Kind:       kind,
		ObservedAt: observedAt.UTC(),
		Message:    strings.TrimSpace(message),
	}
	s.runs[runID] = snapshot
}

func (s *RuntimeStateStore) delete(runID uuid.UUID) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.runs, runID)
}

func (s *RuntimeStateStore) load(runID uuid.UUID) (runtimeRunSnapshot, bool) {
	if s == nil {
		return runtimeRunSnapshot{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.runs[runID]
	return snapshot, ok
}

func (s *RuntimeStateStore) snapshots() []runtimeRunSnapshot {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshots := make([]runtimeRunSnapshot, 0, len(s.runs))
	for _, snapshot := range s.runs {
		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}

type runtimeTicketDisposition string

const (
	runtimeTicketActive        runtimeTicketDisposition = "active"
	runtimeTicketTerminal      runtimeTicketDisposition = "terminal"
	runtimeTicketWorkflowDrift runtimeTicketDisposition = "workflow_drift"
	runtimeTicketInactive      runtimeTicketDisposition = "inactive"
	runtimeTicketLostOwnership runtimeTicketDisposition = "lost_ownership"
)

func classifyRuntimeTicket(ticket *ent.Ticket, runID uuid.UUID, runWorkflowID uuid.UUID) runtimeTicketDisposition {
	if ticket == nil || ticket.CurrentRunID == nil || *ticket.CurrentRunID != runID {
		return runtimeTicketLostOwnership
	}
	if ticket.WorkflowID == nil || *ticket.WorkflowID != runWorkflowID {
		return runtimeTicketWorkflowDrift
	}
	if ticket.Archived {
		return runtimeTicketInactive
	}
	if ticket.Edges.Workflow == nil {
		return runtimeTicketInactive
	}

	finishStatusIDs := ticketStatusIDs(ticket.Edges.Workflow.Edges.FinishStatuses)
	if ticket.CompletedAt != nil || slices.Contains(finishStatusIDs, ticket.StatusID) {
		return runtimeTicketTerminal
	}

	pickupStatusIDs := ticketStatusIDs(ticket.Edges.Workflow.Edges.PickupStatuses)
	if !ticket.RetryPaused && slices.Contains(pickupStatusIDs, ticket.StatusID) {
		return runtimeTicketActive
	}

	return runtimeTicketInactive
}

func releaseStalledClaim(
	ctx context.Context,
	client *ent.Client,
	projectID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	agentID uuid.UUID,
	attemptCount int,
	consecutiveErrors int,
	stallCount int,
	now time.Time,
	source string,
	reason string,
	events provider.EventProvider,
) (bool, bool, error) {
	if client == nil {
		return false, false, fmt.Errorf("release stalled claim: client unavailable")
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		return false, false, fmt.Errorf("start health check tx: %w", err)
	}
	defer rollback(tx)

	nextStallCount := stallCount + 1
	nextAttemptCount := attemptCount + 1
	nextConsecutiveErrors := consecutiveErrors + 1
	retryPaused := nextStallCount >= stalledRetryPauseThreshold
	releasedTickets, err := tx.Ticket.Update().
		Where(
			entticket.IDEQ(ticketID),
			entticket.CurrentRunIDNotNil(),
			entticket.CurrentRunIDEQ(runID),
		).
		ClearCurrentRunID().
		SetStallCount(nextStallCount).
		SetRetryToken(ticketservice.NewRetryToken()).
		Save(ctx)
	if err != nil {
		return false, false, fmt.Errorf("release stalled ticket: %w", err)
	}
	if releasedTickets == 0 {
		return false, false, nil
	}

	ticketUpdate := tx.Ticket.UpdateOneID(ticketID)
	ticketUpdate.SetAttemptCount(nextAttemptCount).
		SetConsecutiveErrors(nextConsecutiveErrors)
	if retryPaused {
		ticketUpdate.ClearNextRetryAt().
			SetRetryPaused(true).
			SetPauseReason(ticketing.PauseReasonRepeatedStalls.String())
	} else {
		ticketUpdate.SetNextRetryAt(now.UTC().Add(stalledRetryDelay)).
			SetRetryPaused(false).
			ClearPauseReason()
	}
	if _, err := ticketUpdate.Save(ctx); err != nil {
		return false, false, fmt.Errorf("update stalled retry policy: %w", err)
	}

	runLastError := strings.TrimSpace(reason)
	if retryPaused {
		runLastError = fmt.Sprintf(
			"%s; ticket retries paused after %d consecutive stalls",
			runLastError,
			nextStallCount,
		)
	}
	releasedRuns, err := tx.AgentRun.Update().
		Where(
			entagentrun.IDEQ(runID),
			entagentrun.StatusIn(
				entagentrun.StatusLaunching,
				entagentrun.StatusReady,
				entagentrun.StatusExecuting,
				entagentrun.StatusErrored,
			),
		).
		SetStatus(entagentrun.StatusErrored).
		SetTerminalAt(now.UTC()).
		SetLastError(runLastError).
		ClearSessionID().
		ClearRuntimeStartedAt().
		ClearLastHeartbeatAt().
		Save(ctx)
	if err != nil {
		return false, false, fmt.Errorf("release stalled run: %w", err)
	}

	releasedAgents, err := tx.Agent.Update().
		Where(
			entagent.IDEQ(agentID),
			entagent.RuntimeControlStateEQ(entagent.RuntimeControlStateActive),
		).
		SetRuntimeControlState(entagent.RuntimeControlStateActive).
		Save(ctx)
	if err != nil {
		return false, false, fmt.Errorf("release stalled agent: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, false, fmt.Errorf("commit stalled release tx: %w", err)
	}
	if retryPaused {
		if _, err := activitysvc.NewEmitter(activitysvc.EntRecorder{Client: client}, events).Emit(ctx, activitysvc.RecordInput{
			ProjectID: projectID,
			TicketID:  &ticketID,
			EventType: stalledRetryPauseEventType,
			Message: fmt.Sprintf(
				"Paused ticket retries after %d consecutive orchestrator stalls; human intervention is required before retrying.",
				nextStallCount,
			),
			Metadata: map[string]any{
				"pause_reason": ticketing.PauseReasonRepeatedStalls.String(),
				"stall_count":  nextStallCount,
				"threshold":    stalledRetryPauseThreshold,
				"source":       strings.TrimSpace(source),
			},
			CreatedAt: now.UTC(),
		}); err != nil {
			return false, false, fmt.Errorf("emit stalled retry pause activity: %w", err)
		}
	}

	return true, releasedAgents > 0 || releasedRuns > 0, nil
}
