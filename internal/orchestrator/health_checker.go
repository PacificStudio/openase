package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

const (
	defaultStallTimeout        = 5 * time.Minute
	stalledRetryDelay          = time.Second
	stalledRetryPauseThreshold = 20
	stalledRetryPauseEventType = activityevent.TypeTicketRetryPaused
)

// HealthCheckReport summarizes the orchestrator health snapshot.
type HealthCheckReport struct {
	ClaimsChecked  int `json:"claims_checked"`
	StalledClaims  int `json:"stalled_claims"`
	AgentsReleased int `json:"agents_released"`
}

// HealthChecker inspects orchestrator state and reports unhealthy agents or tickets.
type HealthChecker struct {
	client  *ent.Client
	logger  *slog.Logger
	now     func() time.Time
	runtime *RuntimeStateStore
	events  provider.EventProvider
}

type claimHealthState struct {
	stalled       bool
	reason        string
	timeout       time.Duration
	lastHeartbeat *time.Time
	age           time.Duration
}

// NewHealthChecker constructs a health checker for the orchestrator runtime.
func NewHealthChecker(client *ent.Client, logger *slog.Logger) *HealthChecker {
	if logger == nil {
		logger = slog.Default()
	}

	return &HealthChecker{
		client:  client,
		logger:  logger.With("component", "health-checker"),
		now:     time.Now,
		runtime: NewRuntimeStateStore(),
	}
}

func (h *HealthChecker) ConfigureRuntimeState(store *RuntimeStateStore) {
	if h == nil || store == nil {
		return
	}
	h.runtime = store
}

func (h *HealthChecker) ConfigureEvents(events provider.EventProvider) {
	if h == nil {
		return
	}
	h.events = events
}

// Run evaluates the current orchestrator health.
func (h *HealthChecker) Run(ctx context.Context) (HealthCheckReport, error) {
	report := HealthCheckReport{}
	if h == nil || h.client == nil {
		return report, fmt.Errorf("health checker unavailable")
	}

	now := h.now().UTC()
	tickets, err := h.client.Ticket.Query().
		Where(
			entticket.CurrentRunIDNotNil(),
			entticket.HasCurrentRunWith(
				entagentrun.HasAgentWith(
					entagent.RuntimeControlStateEQ(entagent.RuntimeControlStateActive),
				),
				entagentrun.StatusIn(
					entagentrun.StatusLaunching,
					entagentrun.StatusReady,
					entagentrun.StatusExecuting,
					entagentrun.StatusErrored,
				),
			),
		).
		WithCurrentRun().
		WithWorkflow().
		Order(ent.Asc(entticket.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return report, fmt.Errorf("list claimed tickets: %w", err)
	}

	for _, ticket := range tickets {
		report.ClaimsChecked++
		if ticket.CurrentRunID != nil {
			if _, managed := h.runtime.load(*ticket.CurrentRunID); managed {
				continue
			}
		}

		state := evaluateClaimHealth(ticket, now)
		if !state.stalled {
			continue
		}

		ticketReleased, agentReleased, err := h.releaseStalledClaim(ctx, ticket, now)
		if err != nil {
			return report, fmt.Errorf("release stalled ticket %s: %w", ticket.ID, err)
		}
		if !ticketReleased {
			continue
		}

		report.StalledClaims++
		if agentReleased {
			report.AgentsReleased++
		}
		nextStallCount := ticket.StallCount + 1
		retryPaused := nextStallCount >= stalledRetryPauseThreshold

		attrs := []any{
			"ticket_id", ticket.ID,
			"agent_id", ticket.Edges.CurrentRun.AgentID,
			"reason", state.reason,
			"stall_count", nextStallCount,
			"timeout", state.timeout.String(),
			"retry_paused", retryPaused,
			"agent_released", agentReleased,
		}
		if retryPaused {
			attrs = append(attrs, "pause_reason", ticketing.PauseReasonRepeatedStalls.String())
		}
		if state.lastHeartbeat != nil {
			attrs = append(
				attrs,
				"last_heartbeat_at", state.lastHeartbeat.Format(time.RFC3339),
				"heartbeat_age", state.age.String(),
			)
		}
		h.logger.Warn("stalled claim released", attrs...)
	}

	return report, nil
}

func evaluateClaimHealth(ticket *ent.Ticket, now time.Time) claimHealthState {
	if ticket == nil {
		return claimHealthState{}
	}

	timeout := stallTimeoutForWorkflow(ticket.Edges.Workflow)
	run := ticket.Edges.CurrentRun
	if run == nil {
		return claimHealthState{
			stalled: true,
			reason:  "missing_run",
			timeout: timeout,
		}
	}
	if run.LastHeartbeatAt == nil {
		reference := run.CreatedAt.UTC()
		if run.RuntimeStartedAt != nil {
			reference = run.RuntimeStartedAt.UTC()
		}
		age := now.Sub(reference)
		if age < 0 {
			age = 0
		}
		if age <= timeout {
			return claimHealthState{}
		}
		return claimHealthState{
			stalled: true,
			reason:  "missing_heartbeat",
			timeout: timeout,
			age:     age,
		}
	}

	lastHeartbeat := run.LastHeartbeatAt.UTC()
	age := now.Sub(lastHeartbeat)
	if age < 0 {
		age = 0
	}
	if age <= timeout {
		return claimHealthState{}
	}

	return claimHealthState{
		stalled:       true,
		reason:        "stalled",
		timeout:       timeout,
		lastHeartbeat: &lastHeartbeat,
		age:           age,
	}
}

func stallTimeoutForWorkflow(workflow *ent.Workflow) time.Duration {
	if workflow == nil || workflow.StallTimeoutMinutes <= 0 {
		return defaultStallTimeout
	}

	return time.Duration(workflow.StallTimeoutMinutes) * time.Minute
}

func (h *HealthChecker) releaseStalledClaim(
	ctx context.Context,
	ticket *ent.Ticket,
	now time.Time,
) (bool, bool, error) {
	if ticket == nil || ticket.CurrentRunID == nil || ticket.Edges.CurrentRun == nil {
		return false, false, nil
	}

	return releaseStalledClaim(
		ctx,
		h.client,
		ticket.ProjectID,
		ticket.ID,
		*ticket.CurrentRunID,
		ticket.Edges.CurrentRun.AgentID,
		ticket.AttemptCount,
		ticket.ConsecutiveErrors,
		ticket.StallCount,
		now,
		"health_checker",
		"runtime stalled or heartbeat missing",
		h.events,
	)
}
