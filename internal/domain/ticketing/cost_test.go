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

	costUSD, err := usage.ResolveCostUSD(ModelPricing{
		CostPerInputToken:  99,
		CostPerOutputToken: 99,
	})
	if err != nil {
		t.Fatalf("ResolveCostUSD returned error: %v", err)
	}
	if costUSD != 0.04 {
		t.Fatalf("ResolveCostUSD = %.2f, want 0.04", costUSD)
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

	costUSD, err := usage.ResolveCostUSD(ModelPricing{
		CostPerInputToken:  0.001,
		CostPerOutputToken: 0.002,
	})
	if err != nil {
		t.Fatalf("ResolveCostUSD returned error: %v", err)
	}
	if costUSD != 0.21 {
		t.Fatalf("ResolveCostUSD = %.2f, want 0.21", costUSD)
	}
}
