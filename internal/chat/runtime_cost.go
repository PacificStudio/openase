package chat

import (
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
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
