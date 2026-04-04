package chat

import (
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

func resolveUsageCostUSD(providerItem catalogdomain.AgentProvider, raw ticketing.RawUsageDelta) *float64 {
	usage, err := ticketing.ParseRawUsageDelta(raw)
	if err != nil {
		return nil
	}
	pricingConfig := catalogdomain.ResolveAgentProviderPricingConfig(
		providerItem.AdapterType,
		providerItem.ModelName,
		providerItem.CostPerInputToken,
		providerItem.CostPerOutputToken,
		providerItem.PricingConfig.ToMap(),
	)
	if usage.ExplicitCostUSD == nil && !pricingConfig.HasAnyPricing() {
		return nil
	}

	costUSD, err := usage.ResolveCostUSD(pricingConfig)
	if err != nil {
		return nil
	}

	return cloneCostUSD(&costUSD)
}

func resolveCLIUsageCostUSD(providerItem catalogdomain.AgentProvider, usage *provider.CLIUsage) *float64 {
	if usage == nil {
		return nil
	}

	raw := ticketing.RawUsageDelta{
		CostUSD: cloneCostUSD(usage.CostUSD),
	}
	if usage.Total.InputTokens > 0 {
		raw.InputTokens = int64Pointer(usage.Total.InputTokens)
	}
	if usage.Total.OutputTokens > 0 {
		raw.OutputTokens = int64Pointer(usage.Total.OutputTokens)
	}
	if usage.Total.CachedInputTokens > 0 {
		raw.CachedInputTokens = int64Pointer(usage.Total.CachedInputTokens)
	}
	if usage.Total.CacheCreationInputTokens > 0 {
		raw.CacheCreationInputTokens = int64Pointer(usage.Total.CacheCreationInputTokens)
	}
	if usage.Total.PromptTokens > 0 {
		raw.PromptTokens = int64Pointer(usage.Total.PromptTokens)
	}
	if usage.Total.CandidateTokens > 0 {
		raw.CandidateTokens = int64Pointer(usage.Total.CandidateTokens)
	}
	if usage.Total.ToolTokens > 0 {
		raw.ToolTokens = int64Pointer(usage.Total.ToolTokens)
	}
	if usage.Total.ReasoningTokens > 0 {
		raw.ReasoningTokens = int64Pointer(usage.Total.ReasoningTokens)
	}
	if usage.ModelContextWindow != nil {
		raw.ModelContextWindow = int64Pointer(*usage.ModelContextWindow)
	}

	return resolveUsageCostUSD(providerItem, raw)
}

func cloneCostUSD(costUSD *float64) *float64 {
	if costUSD == nil {
		return nil
	}

	cloned := *costUSD
	return &cloned
}

func int64Pointer(value int64) *int64 {
	cloned := value
	return &cloned
}
