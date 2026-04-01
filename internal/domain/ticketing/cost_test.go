package ticketing

import "testing"

func TestParseRawUsageDeltaRejectsEmptyPayload(t *testing.T) {
	if _, err := ParseRawUsageDelta(RawUsageDelta{}); err == nil {
		t.Fatal("expected empty usage payload to fail")
	}
}

func TestUsageDeltaResolveCostUSDUsesExplicitCostWhenProvided(t *testing.T) {
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

	resolved, err := usage.ResolveCost(ModelPricing{
		CostPerInputToken:  99,
		CostPerOutputToken: 99,
	})
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

func TestUsageDeltaResolveCostUSDComputesProviderPricing(t *testing.T) {
	inputTokens := int64(120)
	outputTokens := int64(45)

	usage, err := ParseRawUsageDelta(RawUsageDelta{
		InputTokens:  &inputTokens,
		OutputTokens: &outputTokens,
	})
	if err != nil {
		t.Fatalf("ParseRawUsageDelta returned error: %v", err)
	}

	resolved, err := usage.ResolveCost(ModelPricing{
		CostPerInputToken:  0.001,
		CostPerOutputToken: 0.002,
	})
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

func TestUsageDeltaResolveCostDoesNotRoundSmallDeltas(t *testing.T) {
	inputTokens := int64(1)

	usage, err := ParseRawUsageDelta(RawUsageDelta{
		InputTokens: &inputTokens,
	})
	if err != nil {
		t.Fatalf("ParseRawUsageDelta returned error: %v", err)
	}

	resolved, err := usage.ResolveCost(ModelPricing{
		CostPerInputToken:  0.000003,
		CostPerOutputToken: 0,
	})
	if err != nil {
		t.Fatalf("ResolveCost returned error: %v", err)
	}
	if resolved.AmountUSD != 0.000003 {
		t.Fatalf("ResolveCost amount = %.6f, want 0.000003", resolved.AmountUSD)
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
	if resolved, err := usage.ResolveCostUSD(ModelPricing{CostPerInputToken: -1}); err == nil {
		t.Fatalf("ResolveCostUSD() expected input pricing validation error, got %.2f", resolved)
	}
	if _, err := usage.ResolveCostUSD(ModelPricing{CostPerOutputToken: -1}); err == nil {
		t.Fatal("ResolveCostUSD() expected output pricing validation error")
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
