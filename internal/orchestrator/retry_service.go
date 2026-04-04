package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	"github.com/google/uuid"
)

// RetryResult reports the state change produced by a retry operation.
type RetryResult struct {
	TicketID          uuid.UUID             `json:"ticket_id"`
	AttemptCount      int                   `json:"attempt_count"`
	ConsecutiveErrors int                   `json:"consecutive_errors"`
	NextRetryAt       time.Time             `json:"next_retry_at"`
	RetryPaused       bool                  `json:"retry_paused"`
	PauseReason       ticketing.PauseReason `json:"pause_reason"`
	ReleasedAgentID   *uuid.UUID            `json:"released_agent_id,omitempty"`
}

// RetryService manages ticket retry bookkeeping after failed attempts.
type RetryService struct {
	client *ent.Client
	logger *slog.Logger
	now    func() time.Time
}

// NewRetryService constructs a retry service for orchestrator failures.
func NewRetryService(client *ent.Client, logger *slog.Logger) *RetryService {
	if logger == nil {
		logger = slog.Default()
	}

	return &RetryService{
		client: client,
		logger: logger.With("component", "retry-service"),
		now:    time.Now,
	}
}

// MarkAttemptFailed records a failed attempt and computes the next retry state.
func (s *RetryService) MarkAttemptFailed(ctx context.Context, ticketID uuid.UUID) (RetryResult, error) {
	if s == nil || s.client == nil {
		return RetryResult{}, fmt.Errorf("retry service unavailable")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return RetryResult{}, fmt.Errorf("start retry tx: %w", err)
	}
	defer rollback(tx)

	current, err := tx.Ticket.Get(ctx, ticketID)
	if err != nil {
		return RetryResult{}, fmt.Errorf("load ticket %s for retry: %w", ticketID, err)
	}

	releasedAgentID := (*uuid.UUID)(nil)
	if current.CurrentRunID != nil {
		runItem, err := tx.AgentRun.Get(ctx, *current.CurrentRunID)
		if err != nil {
			return RetryResult{}, fmt.Errorf("load current run %s for retry: %w", *current.CurrentRunID, err)
		}
		releasedAgentID = &runItem.AgentID
	}
	nextAttemptCount := current.AttemptCount + 1
	nextConsecutiveErrors := current.ConsecutiveErrors + 1
	nextRetryAt := s.now().UTC().Add(ticketing.ComputeRetryBackoff(nextConsecutiveErrors))
	pauseReason := ticketing.PauseReason("")
	if ticketing.ShouldPauseForBudget(current.CostAmount, current.BudgetUsd) {
		pauseReason = ticketing.PauseReasonBudgetExhausted
	}

	update := ticketrepo.ScheduleRetryOne(
		tx.Ticket.UpdateOneID(current.ID).
			ClearCurrentRunID().
			SetAttemptCount(nextAttemptCount).
			SetConsecutiveErrors(nextConsecutiveErrors).
			SetStallCount(0),
		nextRetryAt,
		pauseReason.String(),
	)

	if _, err := update.Save(ctx); err != nil {
		return RetryResult{}, fmt.Errorf("update ticket %s retry state: %w", ticketID, err)
	}

	if err := releaseCurrentRunClaim(ctx, tx, current); err != nil {
		return RetryResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return RetryResult{}, fmt.Errorf("commit retry tx: %w", err)
	}

	s.logger.Info(
		"ticket retry scheduled",
		"operation", "schedule_retry",
		"ticket_id", current.ID,
		"project_id", current.ProjectID,
		"workflow_id", current.WorkflowID,
		"current_run_id", current.CurrentRunID,
		"attempt_count", nextAttemptCount,
		"consecutive_errors", nextConsecutiveErrors,
		"backoff_seconds", int(nextRetryAt.Sub(s.now().UTC()).Seconds()),
		"next_retry_at", nextRetryAt.Format(time.RFC3339),
		"retry_paused", pauseReason != "",
		"pause_reason", pauseReason.String(),
		"released_agent_id", releasedAgentID,
		"cost_amount", current.CostAmount,
		"budget_usd", current.BudgetUsd,
	)

	return RetryResult{
		TicketID:          current.ID,
		AttemptCount:      nextAttemptCount,
		ConsecutiveErrors: nextConsecutiveErrors,
		NextRetryAt:       nextRetryAt,
		RetryPaused:       pauseReason != "",
		PauseReason:       pauseReason,
		ReleasedAgentID:   releasedAgentID,
	}, nil
}

func releaseCurrentRunClaim(ctx context.Context, tx *ent.Tx, ticketItem *ent.Ticket) error {
	if ticketItem == nil {
		return nil
	}

	if ticketItem.CurrentRunID == nil {
		return nil
	}

	runItem, err := tx.AgentRun.Get(ctx, *ticketItem.CurrentRunID)
	if err != nil {
		return fmt.Errorf("load failed current run: %w", err)
	}

	if _, err := tx.AgentRun.UpdateOneID(runItem.ID).
		SetStatus(entagentrun.StatusErrored).
		SetTerminalAt(time.Now().UTC()).
		ClearSessionID().
		ClearRuntimeStartedAt().
		ClearLastHeartbeatAt().
		Save(ctx); err != nil {
		return fmt.Errorf("finalize failed agent run: %w", err)
	}

	if _, err := tx.Agent.UpdateOneID(runItem.AgentID).
		SetRuntimeControlState(entagent.RuntimeControlStateActive).
		Save(ctx); err != nil {
		return fmt.Errorf("reset current run agent runtime control state: %w", err)
	}

	return nil
}
