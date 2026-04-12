package ticket

import (
	"context"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
)

func (r *UsageRepository) RecordUsage(
	ctx context.Context,
	input RecordUsageInput,
	usageDelta ticketing.UsageDelta,
) (PersistedUsageResult, error) {
	if r.client == nil {
		return PersistedUsageResult{}, errUnavailable
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return PersistedUsageResult{}, fmt.Errorf("start ticket usage tx: %w", err)
	}
	defer rollback(tx)

	ticketItem, err := tx.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		return PersistedUsageResult{}, mapTicketReadError("get ticket for usage", err)
	}

	agentItem, err := tx.Agent.Query().
		Where(entagent.IDEQ(input.AgentID)).
		WithProvider().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return PersistedUsageResult{}, fmt.Errorf("agent %s not found", input.AgentID)
		}
		return PersistedUsageResult{}, fmt.Errorf("get agent for usage: %w", err)
	}
	if agentItem.ProjectID != ticketItem.ProjectID {
		return PersistedUsageResult{}, fmt.Errorf("agent %s does not belong to ticket project %s", agentItem.ID, ticketItem.ProjectID)
	}
	if agentItem.Edges.Provider == nil {
		return PersistedUsageResult{}, fmt.Errorf("agent provider must be loaded for usage accounting")
	}

	pricingConfig := catalogdomain.ResolveAgentProviderPricingConfig(
		catalogdomain.AgentProviderAdapterType(agentItem.Edges.Provider.AdapterType),
		agentItem.Edges.Provider.ModelName,
		agentItem.Edges.Provider.CostPerInputToken,
		agentItem.Edges.Provider.CostPerOutputToken,
		agentItem.Edges.Provider.PricingConfig,
	)
	resolvedCost, err := usageDelta.ResolveCost(pricingConfig)
	if err != nil {
		return PersistedUsageResult{}, err
	}

	nextCostAmount := ticketItem.CostAmount + resolvedCost.AmountUSD
	update := tx.Ticket.UpdateOneID(ticketItem.ID).
		AddCostTokensInput(usageDelta.InputTokens).
		AddCostTokensOutput(usageDelta.OutputTokens).
		AddCostAmount(resolvedCost.AmountUSD)

	if ticketing.ShouldPauseForBudget(nextCostAmount, ticketItem.BudgetUsd) &&
		(!ticketItem.RetryPaused || ticketItem.PauseReason == "" || ticketItem.PauseReason == ticketing.PauseReasonBudgetExhausted.String()) {
		update.SetRetryPaused(true).
			SetPauseReason(ticketing.PauseReasonBudgetExhausted.String())
	}

	if _, err := update.Save(ctx); err != nil {
		return PersistedUsageResult{}, mapTicketWriteError("update ticket usage", err)
	}

	if usageDelta.TotalTokens() > 0 {
		if _, err := tx.Agent.UpdateOneID(agentItem.ID).
			AddTotalTokensUsed(usageDelta.TotalTokens()).
			Save(ctx); err != nil {
			return PersistedUsageResult{}, fmt.Errorf("update agent usage counters: %w", err)
		}
	}

	var agentRunID string
	if input.RunID != nil {
		agentRunID = input.RunID.String()
		runItem, err := tx.AgentRun.Get(ctx, *input.RunID)
		if err != nil {
			if ent.IsNotFound(err) {
				return PersistedUsageResult{}, fmt.Errorf("agent run %s not found", *input.RunID)
			}
			return PersistedUsageResult{}, fmt.Errorf("get agent run for usage: %w", err)
		}
		if runItem.AgentID != agentItem.ID {
			return PersistedUsageResult{}, fmt.Errorf("agent run %s does not belong to agent %s", runItem.ID, agentItem.ID)
		}
		if runItem.TicketID != ticketItem.ID {
			return PersistedUsageResult{}, fmt.Errorf("agent run %s does not belong to ticket %s", runItem.ID, ticketItem.ID)
		}

		if _, err := tx.AgentRun.UpdateOneID(runItem.ID).
			AddInputTokens(usageDelta.InputTokens).
			AddOutputTokens(usageDelta.OutputTokens).
			AddCachedInputTokens(usageDelta.CachedInputTokens).
			AddCacheCreationInputTokens(usageDelta.CacheCreationInputTokens).
			AddReasoningTokens(usageDelta.ReasoningTokens).
			AddPromptTokens(usageDelta.PromptTokens).
			AddCandidateTokens(usageDelta.CandidateTokens).
			AddToolTokens(usageDelta.ToolTokens).
			AddTotalTokens(usageDelta.TotalTokens()).
			Save(ctx); err != nil {
			return PersistedUsageResult{}, fmt.Errorf("update agent run usage counters: %w", err)
		}
	}

	if _, err := tx.ActivityEvent.Create().
		SetProjectID(ticketItem.ProjectID).
		SetTicketID(ticketItem.ID).
		SetAgentID(agentItem.ID).
		SetEventType(ticketing.CostRecordedEventType).
		SetMessage("").
		SetMetadata(map[string]any{
			"input_tokens":  usageDelta.InputTokens,
			"output_tokens": usageDelta.OutputTokens,
			"total_tokens":  usageDelta.TotalTokens(),
			"cost_usd":      resolvedCost.AmountUSD,
			"cost_source":   resolvedCost.Source.String(),
			"agent_run_id":  agentRunID,
		}).
		SetCreatedAt(timeNowUTC()).
		Save(ctx); err != nil {
		return PersistedUsageResult{}, fmt.Errorf("create ticket cost event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return PersistedUsageResult{}, fmt.Errorf("commit ticket usage tx: %w", err)
	}

	ticketAfter, err := NewQueryRepository(r.client).Get(ctx, ticketItem.ID)
	if err != nil {
		return PersistedUsageResult{}, err
	}

	return PersistedUsageResult{
		Result: RecordUsageResult{
			Ticket: ticketAfter,
			Applied: AppliedUsage{
				InputTokens:  usageDelta.InputTokens,
				OutputTokens: usageDelta.OutputTokens,
				CostUSD:      resolvedCost.AmountUSD,
				CostSource:   resolvedCost.Source.String(),
			},
			BudgetExceeded: ticketing.ShouldPauseForBudget(ticketAfter.CostAmount, ticketAfter.BudgetUSD),
		},
		MetricsAgent: UsageMetricsAgent{
			ProviderName: agentItem.Edges.Provider.Name,
			ModelName:    agentItem.Edges.Provider.ModelName,
		},
		ProjectID: ticketItem.ProjectID,
	}, nil
}
