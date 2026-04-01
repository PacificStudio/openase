package ticketing

import (
	"math"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/domain/pricing"
)

func TestParseRawUsageDeltaRejectsEmptyPayload(t *testing.T) {
	if _, err := ParseRawUsageDelta(RawUsageDelta{}); err == nil {
		t.Fatal("expected empty usage payload to fail")
	}
}

func TestUsageDeltaResolveCostUsesExplicitCostWhenProvided(t *testing.T) {
	inputTokens := int64(120)
	outputTokens := int64(45)
	explicitCost := 0.037

	usage, err := ParseRawUsageDelta(RawUsageDelta{
		InputTokens:  &inputTokens,
		OutputTokens: &outputTokens,
		CostUSD:      &explicitCost,
	})
	if err != nil {
		t.Fatalf("ParseRawUsageDelta returned error: %v", err)
	}

	resolved, err := usage.ResolveCost(pricing.CustomFlatPricingConfig(99, 99))
	if err != nil {
		t.Fatalf("ResolveCost returned error: %v", err)
	}
	if resolved.AmountUSD != 0.037 {
		t.Fatalf("ResolveCost amount = %.6f, want 0.037000", resolved.AmountUSD)
	}
	if resolved.Source != UsageCostSourceActual {
		t.Fatalf("ResolveCost source = %q, want %q", resolved.Source, UsageCostSourceActual)
	}
}

func TestUsageDeltaResolveCostComputesFlatProviderPricing(t *testing.T) {
	inputTokens := int64(120)
	outputTokens := int64(45)

	usage, err := ParseRawUsageDelta(RawUsageDelta{
		InputTokens:  &inputTokens,
		OutputTokens: &outputTokens,
	})
	if err != nil {
		t.Fatalf("ParseRawUsageDelta returned error: %v", err)
	}

	resolved, err := usage.ResolveCost(pricing.CustomFlatPricingConfig(0.001, 0.002))
	if err != nil {
		t.Fatalf("ResolveCost returned error: %v", err)
	}
	if resolved.AmountUSD != 0.21 {
		t.Fatalf("ResolveCost amount = %.6f, want 0.210000", resolved.AmountUSD)
	}
	if resolved.Source != UsageCostSourceEstimated {
		t.Fatalf("ResolveCost source = %q, want %q", resolved.Source, UsageCostSourceEstimated)
	}
	if usage.TotalTokens() != 165 {
		t.Fatalf("TotalTokens() = %d, want 165", usage.TotalTokens())
	}
}

func TestUsageDeltaResolveCostUsesCachedInputPricing(t *testing.T) {
	inputTokens := int64(100)
	outputTokens := int64(20)
	cachedInputTokens := int64(40)

	usage, err := ParseRawUsageDelta(RawUsageDelta{
		InputTokens:       &inputTokens,
		OutputTokens:      &outputTokens,
		CachedInputTokens: &cachedInputTokens,
	})
	if err != nil {
		t.Fatalf("ParseRawUsageDelta returned error: %v", err)
	}

	pricingConfig := pricing.ProviderModelPricingConfig{
		SourceKind:  pricing.PricingSourceKindOfficial,
		PricingMode: pricing.PricingModeFlat,
		Rates: pricing.ProviderModelPricingRates{
			InputPerToken:           0.001,
			OutputPerToken:          0.002,
			CachedInputReadPerToken: 0.0001,
		},
	}

	resolved, err := usage.ResolveCost(pricingConfig)
	if err != nil {
		t.Fatalf("ResolveCost returned error: %v", err)
	}
	if math.Abs(resolved.AmountUSD-0.104) > 0.0000001 {
		t.Fatalf("ResolveCost amount = %.6f, want 0.104000", resolved.AmountUSD)
	}
}

func TestUsageDeltaResolveCostUsesAnthropicCacheWriteWindows(t *testing.T) {
	inputTokens := int64(120)
	outputTokens := int64(10)
	cacheWrite5mTokens := int64(20)
	cacheWrite1hTokens := int64(30)

	usage, err := ParseRawUsageDelta(RawUsageDelta{
		InputTokens:                &inputTokens,
		OutputTokens:               &outputTokens,
		CacheCreationInputTokens5m: &cacheWrite5mTokens,
		CacheCreationInputTokens1h: &cacheWrite1hTokens,
	})
	if err != nil {
		t.Fatalf("ParseRawUsageDelta returned error: %v", err)
	}

	pricingConfig := pricing.ProviderModelPricingConfig{
		SourceKind:              pricing.PricingSourceKindOfficial,
		PricingMode:             pricing.PricingModeFlat,
		DefaultCacheWriteWindow: pricing.CacheWriteWindowFiveMinutes,
		Rates: pricing.ProviderModelPricingRates{
			InputPerToken:                 0.003,
			OutputPerToken:                0.015,
			CacheWriteFiveMinutesPerToken: 0.00375,
			CacheWriteOneHourPerToken:     0.006,
		},
	}

	resolved, err := usage.ResolveCost(pricingConfig)
	if err != nil {
		t.Fatalf("ResolveCost returned error: %v", err)
	}
	if resolved.AmountUSD != 0.615 {
		t.Fatalf("ResolveCost amount = %.6f, want 0.615000", resolved.AmountUSD)
	}
}

func TestUsageDeltaResolveCostUsesGeminiTieredCaching(t *testing.T) {
	inputTokens := int64(250_000)
	outputTokens := int64(20_000)
	promptTokens := int64(250_000)
	cachedInputTokens := int64(100_000)
	cacheStorageTokens := int64(100_000)
	cacheStorageHours := 2.0

	usage, err := ParseRawUsageDelta(RawUsageDelta{
		InputTokens:        &inputTokens,
		OutputTokens:       &outputTokens,
		PromptTokens:       &promptTokens,
		CachedInputTokens:  &cachedInputTokens,
		CacheStorageTokens: &cacheStorageTokens,
		CacheStorageHours:  &cacheStorageHours,
	})
	if err != nil {
		t.Fatalf("ParseRawUsageDelta returned error: %v", err)
	}

	pricingConfig := pricing.ProviderModelPricingConfig{
		SourceKind:  pricing.PricingSourceKindOfficial,
		PricingMode: pricing.PricingModeTiered,
		Tiers: []pricing.ProviderModelPricingTier{
			{
				Label:           "<=200k prompt tokens",
				MaxPromptTokens: 200_000,
				Rates: pricing.ProviderModelPricingRates{
					InputPerToken:            0.00125,
					OutputPerToken:           0.01,
					CachedInputReadPerToken:  0.000125,
					CacheStoragePerTokenHour: 0.0000045,
				},
			},
			{
				Label: ">200k prompt tokens",
				Rates: pricing.ProviderModelPricingRates{
					InputPerToken:            0.0025,
					OutputPerToken:           0.015,
					CachedInputReadPerToken:  0.00025,
					CacheStoragePerTokenHour: 0.0000045,
				},
			},
		},
	}

	resolved, err := usage.ResolveCost(pricingConfig)
	if err != nil {
		t.Fatalf("ResolveCost returned error: %v", err)
	}
	want := 700.9
	if resolved.AmountUSD != want {
		t.Fatalf("ResolveCost amount = %.6f, want %.6f", resolved.AmountUSD, want)
	}
}

func TestUsageDeltaResolveCostDoesNotRoundSmallDeltas(t *testing.T) {
	inputTokens := int64(1)

	usage, err := ParseRawUsageDelta(RawUsageDelta{
		InputTokens: &inputTokens,
	})
	if err != nil {
		t.Fatalf("ParseRawUsageDelta returned error: %v", err)
	}

	resolved, err := usage.ResolveCost(pricing.CustomFlatPricingConfig(0.000003, 0))
	if err != nil {
		t.Fatalf("ResolveCost returned error: %v", err)
	}
	if resolved.AmountUSD != 0.000003 {
		t.Fatalf("ResolveCost amount = %.6f, want 0.000003", resolved.AmountUSD)
	}
}

func TestUsageDeltaResolveCostUSDReturnsAmountOnly(t *testing.T) {
	inputTokens := int64(2)

	usage, err := ParseRawUsageDelta(RawUsageDelta{
		InputTokens: &inputTokens,
	})
	if err != nil {
		t.Fatalf("ParseRawUsageDelta returned error: %v", err)
	}

	costUSD, err := usage.ResolveCostUSD(pricing.CustomFlatPricingConfig(0.5, 0))
	if err != nil {
		t.Fatalf("ResolveCostUSD returned error: %v", err)
	}
	if costUSD != 1 {
		t.Fatalf("ResolveCostUSD = %.2f, want 1.00", costUSD)
	}
}

func TestUsageDeltaValidationHelpers(t *testing.T) {
	negativeTokens := int64(-1)
	if _, err := ParseRawUsageDelta(RawUsageDelta{InputTokens: &negativeTokens}); err == nil {
		t.Fatal("ParseRawUsageDelta() expected negative token validation error")
	}
	if _, err := ParseRawUsageDelta(RawUsageDelta{OutputTokens: &negativeTokens}); err == nil {
		t.Fatal("ParseRawUsageDelta() expected negative output token validation error")
	}

	negativeCost := -0.1
	if _, err := ParseRawUsageDelta(RawUsageDelta{CostUSD: &negativeCost}); err == nil {
		t.Fatal("ParseRawUsageDelta() expected negative cost validation error")
	}

	if _, err := parseNonNegativeInt64("input_tokens", &negativeTokens); err == nil {
		t.Fatal("parseNonNegativeInt64() expected validation error")
	}
	if got, err := parseNonNegativeInt64("input_tokens", nil); err != nil || got != 0 {
		t.Fatalf("parseNonNegativeInt64(nil) = %d, %v; want 0, nil", got, err)
	}

	if _, err := parseOptionalNonNegativeFloat64("cost_usd", &negativeCost); err == nil {
		t.Fatal("parseOptionalNonNegativeFloat64() expected validation error")
	}
	if got, err := parseOptionalNonNegativeFloat64("cost_usd", nil); err != nil || got != nil {
		t.Fatalf("parseOptionalNonNegativeFloat64(nil) = %v, %v; want nil, nil", got, err)
	}

	explicitCost := 0.123
	usage, err := ParseRawUsageDelta(RawUsageDelta{CostUSD: &explicitCost})
	if err != nil {
		t.Fatalf("ParseRawUsageDelta() explicit cost error = %v", err)
	}
	invalidPricing := pricing.ProviderModelPricingConfig{
		SourceKind:  pricing.PricingSourceKindCustom,
		PricingMode: pricing.PricingModeFlat,
		Rates: pricing.ProviderModelPricingRates{
			InputPerToken: -1,
		},
	}
	if resolved, err := usage.ResolveCostUSD(invalidPricing); err == nil {
		t.Fatalf("ResolveCostUSD() expected input pricing validation error, got %.2f", resolved)
	}

	outputOnly := int64(42)
	if parsed, err := ParseRawUsageDelta(RawUsageDelta{OutputTokens: &outputOnly}); err != nil || parsed.OutputTokens != 42 {
		t.Fatalf("ParseRawUsageDelta(output only) = %+v, %v", parsed, err)
	}
	inputOnly := int64(21)
	if parsed, err := ParseRawUsageDelta(RawUsageDelta{InputTokens: &inputOnly}); err != nil || parsed.InputTokens != 21 {
		t.Fatalf("ParseRawUsageDelta(input only) = %+v, %v", parsed, err)
	}

	unroundedCost := 0.000004
	if got, err := parseOptionalNonNegativeFloat64("cost_usd", &unroundedCost); err != nil || got == nil || *got != unroundedCost {
		t.Fatalf("parseOptionalNonNegativeFloat64(unrounded) = %v, %v; want %.6f, nil", got, err, unroundedCost)
	}
}

func TestUsageCostHelpers(t *testing.T) {
	if rounded := RoundUSD(1.235); rounded != 1.24 {
		t.Fatalf("RoundUSD(1.235) = %.2f, want 1.24", rounded)
	}
	if UsageCostSourceActual.String() != "actual" {
		t.Fatalf("UsageCostSourceActual.String() = %q, want actual", UsageCostSourceActual.String())
	}
}
