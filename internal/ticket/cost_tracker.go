package ticket

import (
	"context"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

type RecordUsageInput struct {
	AgentID  uuid.UUID
	TicketID uuid.UUID
	Usage    ticketing.RawUsageDelta
}

type AppliedUsage struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
}

type RecordUsageResult struct {
	Ticket         Ticket       `json:"ticket"`
	Applied        AppliedUsage `json:"applied"`
	BudgetExceeded bool         `json:"budget_exceeded"`
}

func (s *Service) RecordUsage(
	ctx context.Context,
	input RecordUsageInput,
	metrics provider.MetricsProvider,
) (RecordUsageResult, error) {
	if s.client == nil {
		return RecordUsageResult{}, ErrUnavailable
	}
	if input.AgentID == uuid.Nil {
		return RecordUsageResult{}, fmt.Errorf("agent_id must be a valid UUID")
	}
	if input.TicketID == uuid.Nil {
		return RecordUsageResult{}, fmt.Errorf("ticket_id must be a valid UUID")
	}
	if metrics == nil {
		metrics = provider.NewNoopMetricsProvider()
	}

	usageDelta, err := ticketing.ParseRawUsageDelta(input.Usage)
	if err != nil {
		return RecordUsageResult{}, err
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return RecordUsageResult{}, fmt.Errorf("start ticket usage tx: %w", err)
	}
	defer rollback(tx)

	ticketItem, err := tx.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		return RecordUsageResult{}, s.mapTicketReadError("get ticket for usage", err)
	}

	agentItem, err := tx.Agent.Query().
		Where(entagent.IDEQ(input.AgentID)).
		WithProvider().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return RecordUsageResult{}, fmt.Errorf("get agent for usage: %w", ErrTicketNotFound)
		}
		return RecordUsageResult{}, fmt.Errorf("get agent for usage: %w", err)
	}
	if agentItem.ProjectID != ticketItem.ProjectID {
		return RecordUsageResult{}, fmt.Errorf("agent %s does not belong to ticket project %s", agentItem.ID, ticketItem.ProjectID)
	}
	if agentItem.Edges.Provider == nil {
		return RecordUsageResult{}, fmt.Errorf("agent provider must be loaded for usage accounting")
	}

	costUSD, err := usageDelta.ResolveCostUSD(ticketing.ModelPricing{
		CostPerInputToken:  agentItem.Edges.Provider.CostPerInputToken,
		CostPerOutputToken: agentItem.Edges.Provider.CostPerOutputToken,
	})
	if err != nil {
		return RecordUsageResult{}, err
	}

	nextCostAmount := ticketing.RoundUSD(ticketItem.CostAmount + costUSD)
	update := tx.Ticket.UpdateOneID(ticketItem.ID).
		AddCostTokensInput(usageDelta.InputTokens).
		AddCostTokensOutput(usageDelta.OutputTokens).
		AddCostAmount(costUSD)

	if ticketing.ShouldPauseForBudget(nextCostAmount, ticketItem.BudgetUsd) &&
		(!ticketItem.RetryPaused || ticketItem.PauseReason == "" || ticketItem.PauseReason == ticketing.PauseReasonBudgetExhausted.String()) {
		update.SetRetryPaused(true).
			SetPauseReason(ticketing.PauseReasonBudgetExhausted.String())
	}

	if _, err := update.Save(ctx); err != nil {
		return RecordUsageResult{}, s.mapTicketWriteError("update ticket usage", err)
	}

	if usageDelta.TotalTokens() > 0 {
		if _, err := tx.Agent.UpdateOneID(agentItem.ID).
			AddTotalTokensUsed(usageDelta.TotalTokens()).
			Save(ctx); err != nil {
			return RecordUsageResult{}, fmt.Errorf("update agent usage counters: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return RecordUsageResult{}, fmt.Errorf("commit ticket usage tx: %w", err)
	}

	recordTokenUsageMetrics(metrics, agentItem, usageDelta)
	recordCostUsageMetrics(metrics, agentItem, ticketItem.ProjectID, costUSD)

	ticketAfter, err := s.Get(ctx, ticketItem.ID)
	if err != nil {
		return RecordUsageResult{}, err
	}

	return RecordUsageResult{
		Ticket: ticketAfter,
		Applied: AppliedUsage{
			InputTokens:  usageDelta.InputTokens,
			OutputTokens: usageDelta.OutputTokens,
			CostUSD:      costUSD,
		},
		BudgetExceeded: ticketing.ShouldPauseForBudget(ticketAfter.CostAmount, ticketAfter.BudgetUSD),
	}, nil
}

func recordTokenUsageMetrics(metrics provider.MetricsProvider, agentItem *ent.Agent, usage ticketing.UsageDelta) {
	if metrics == nil || agentItem == nil || agentItem.Edges.Provider == nil {
		return
	}

	baseTags := provider.Tags{
		"provider": agentItem.Edges.Provider.Name,
		"model":    agentItem.Edges.Provider.ModelName,
	}
	if usage.InputTokens > 0 {
		metrics.Counter("openase.agent.tokens_used_total", mergeTags(baseTags, provider.Tags{
			"direction": "input",
		})).Add(float64(usage.InputTokens))
	}
	if usage.OutputTokens > 0 {
		metrics.Counter("openase.agent.tokens_used_total", mergeTags(baseTags, provider.Tags{
			"direction": "output",
		})).Add(float64(usage.OutputTokens))
	}
}

func recordCostUsageMetrics(metrics provider.MetricsProvider, agentItem *ent.Agent, projectID uuid.UUID, costUSD float64) {
	if metrics == nil || costUSD <= 0 || agentItem == nil || agentItem.Edges.Provider == nil {
		return
	}

	metrics.Counter("openase.agent.cost_usd_total", provider.Tags{
		"provider": agentItem.Edges.Provider.Name,
		"model":    agentItem.Edges.Provider.ModelName,
		"project":  projectID.String(),
	}).Add(costUSD)
}

func mergeTags(base provider.Tags, extra provider.Tags) provider.Tags {
	merged := make(provider.Tags, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}
