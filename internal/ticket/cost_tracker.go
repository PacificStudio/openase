package ticket

import (
	"context"
	"fmt"

	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func (s *Service) RecordUsage(
	ctx context.Context,
	input RecordUsageInput,
	metrics provider.MetricsProvider,
) (RecordUsageResult, error) {
	if s == nil || s.repo == nil {
		return RecordUsageResult{}, ErrUnavailable
	}
	if input.AgentID == uuid.Nil {
		return RecordUsageResult{}, fmt.Errorf("agent_id must be a valid UUID")
	}
	if input.TicketID == uuid.Nil {
		return RecordUsageResult{}, fmt.Errorf("ticket_id must be a valid UUID")
	}
	if input.RunID != nil && *input.RunID == uuid.Nil {
		return RecordUsageResult{}, fmt.Errorf("run_id must be a valid UUID")
	}
	if metrics == nil {
		metrics = provider.NewNoopMetricsProvider()
	}

	usageDelta, err := ticketing.ParseRawUsageDelta(input.Usage)
	if err != nil {
		return RecordUsageResult{}, err
	}

	persisted, err := s.repo.RecordUsage(ctx, input, usageDelta)
	if err != nil {
		return RecordUsageResult{}, err
	}

	recordTokenUsageMetrics(metrics, persisted.MetricsAgent, usageDelta)
	recordCostUsageMetrics(metrics, persisted.MetricsAgent, persisted.ProjectID, persisted.Result.Applied.CostUSD)

	return persisted.Result, nil
}

func recordTokenUsageMetrics(metrics provider.MetricsProvider, agent UsageMetricsAgent, usage ticketing.UsageDelta) {
	if metrics == nil || agent.ProviderName == "" || agent.ModelName == "" {
		return
	}

	baseTags := provider.Tags{
		"provider": agent.ProviderName,
		"model":    agent.ModelName,
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

func recordCostUsageMetrics(metrics provider.MetricsProvider, agent UsageMetricsAgent, projectID uuid.UUID, costUSD float64) {
	if metrics == nil || costUSD <= 0 || agent.ProviderName == "" || agent.ModelName == "" {
		return
	}

	metrics.Counter("openase.agent.cost_usd_total", provider.Tags{
		"provider": agent.ProviderName,
		"model":    agent.ModelName,
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
