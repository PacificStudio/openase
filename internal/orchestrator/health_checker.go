package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
)

const (
	defaultStallTimeout = 5 * time.Minute
	stalledRetryDelay   = time.Second
)

type HealthCheckReport struct {
	ClaimsChecked  int `json:"claims_checked"`
	StalledClaims  int `json:"stalled_claims"`
	AgentsReleased int `json:"agents_released"`
}

type HealthChecker struct {
	client *ent.Client
	logger *slog.Logger
	now    func() time.Time
}

type claimHealthState struct {
	stalled       bool
	reason        string
	timeout       time.Duration
	lastHeartbeat *time.Time
	age           time.Duration
}

func NewHealthChecker(client *ent.Client, logger *slog.Logger) *HealthChecker {
	if logger == nil {
		logger = slog.Default()
	}

	return &HealthChecker{
		client: client,
		logger: logger.With("component", "health-checker"),
		now:    time.Now,
	}
}

func (h *HealthChecker) Run(ctx context.Context) (HealthCheckReport, error) {
	report := HealthCheckReport{}
	if h == nil || h.client == nil {
		return report, fmt.Errorf("health checker unavailable")
	}

	now := h.now().UTC()
	tickets, err := h.client.Ticket.Query().
		Where(
			entticket.AssignedAgentIDNotNil(),
			entticket.HasAssignedAgentWith(
				entagent.StatusIn(entagent.StatusClaimed, entagent.StatusRunning),
			),
		).
		WithAssignedAgent().
		WithWorkflow().
		Order(ent.Asc(entticket.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return report, fmt.Errorf("list claimed tickets: %w", err)
	}

	for _, ticket := range tickets {
		report.ClaimsChecked++

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

		attrs := []any{
			"ticket_id", ticket.ID,
			"agent_id", ticket.AssignedAgentID,
			"reason", state.reason,
			"timeout", state.timeout.String(),
			"agent_released", agentReleased,
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
	agent := ticket.Edges.AssignedAgent
	if agent == nil || agent.LastHeartbeatAt == nil {
		return claimHealthState{
			stalled: true,
			reason:  "missing_heartbeat",
			timeout: timeout,
		}
	}

	lastHeartbeat := agent.LastHeartbeatAt.UTC()
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
	if ticket == nil || ticket.AssignedAgentID == nil {
		return false, false, nil
	}

	agentID := *ticket.AssignedAgentID
	tx, err := h.client.Tx(ctx)
	if err != nil {
		return false, false, fmt.Errorf("start health check tx: %w", err)
	}
	defer rollback(tx)

	retryAt := now.Add(stalledRetryDelay)
	releasedTickets, err := tx.Ticket.Update().
		Where(
			entticket.IDEQ(ticket.ID),
			entticket.AssignedAgentIDEQ(agentID),
		).
		ClearAssignedAgentID().
		SetNextRetryAt(retryAt).
		SetRetryPaused(false).
		AddStallCount(1).
		Save(ctx)
	if err != nil {
		return false, false, fmt.Errorf("release stalled ticket: %w", err)
	}
	if releasedTickets == 0 {
		return false, false, nil
	}

	releasedAgents, err := tx.Agent.Update().
		Where(
			entagent.IDEQ(agentID),
			entagent.CurrentTicketIDEQ(ticket.ID),
			entagent.StatusIn(entagent.StatusClaimed, entagent.StatusRunning),
		).
		ClearCurrentTicketID().
		SetStatus(entagent.StatusIdle).
		Save(ctx)
	if err != nil {
		return false, false, fmt.Errorf("release stalled agent: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, false, fmt.Errorf("commit stalled release tx: %w", err)
	}

	return true, releasedAgents > 0, nil
}
