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
	if usage.ExplicitCostUSD == nil && !providerHasUsagePricing(providerItem) {
		return nil
	}

	costUSD, err := usage.ResolveCostUSD(ticketing.ModelPricing{
		CostPerInputToken:  providerItem.CostPerInputToken,
		CostPerOutputToken: providerItem.CostPerOutputToken,
	})
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

	return resolveUsageCostUSD(providerItem, raw)
}

func providerHasUsagePricing(providerItem catalogdomain.AgentProvider) bool {
	return providerItem.CostPerInputToken > 0 || providerItem.CostPerOutputToken > 0
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
