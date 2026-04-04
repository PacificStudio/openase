package catalog

import (
	"testing"

	"github.com/BetterAndBetterII/openase/internal/domain/pricing"
)

func TestBuiltinAgentProviderPricingConfigReturnsClone(t *testing.T) {
	t.Parallel()

	config, ok := BuiltinAgentProviderPricingConfig(AgentProviderAdapterTypeCodexAppServer, " gpt-5.4 ")
	if !ok {
		t.Fatal("expected builtin codex pricing config")
	}
	if config.SourceKind != pricing.PricingSourceKindOfficial || config.Provider != "openai" {
		t.Fatalf("unexpected builtin pricing config: %+v", config)
	}

	config.Notes[0] = "changed"
	fresh, ok := BuiltinAgentProviderPricingConfig(AgentProviderAdapterTypeCodexAppServer, "gpt-5.4")
	if !ok {
		t.Fatal("expected fresh builtin codex pricing config")
	}
	if fresh.Notes[0] == "changed" {
		t.Fatalf("expected builtin pricing lookup to return clone, got %+v", fresh)
	}

	if _, ok := BuiltinAgentProviderPricingConfig(AgentProviderAdapterType("unknown"), "gpt-5.4"); ok {
		t.Fatal("expected unknown adapter pricing lookup to fail")
	}
	if _, ok := BuiltinAgentProviderPricingConfig(AgentProviderAdapterTypeCodexAppServer, "unknown"); ok {
		t.Fatal("expected unknown model pricing lookup to fail")
	}
}

func TestResolveAgentProviderPricingConfig(t *testing.T) {
	t.Parallel()

	rawCustom := map[string]any{
		"source_kind":  "custom",
		"pricing_mode": "flat",
		"rates": map[string]any{
			"input_per_token":  0.123,
			"output_per_token": 0.456,
		},
	}
	custom := ResolveAgentProviderPricingConfig(
		AgentProviderAdapterTypeCodexAppServer,
		"gpt-5.4",
		0,
		0,
		rawCustom,
	)
	if custom.Rates.InputPerToken != 0.123 || custom.Rates.OutputPerToken != 0.456 {
		t.Fatalf("expected raw custom pricing config to win, got %+v", custom)
	}

	builtin := ResolveAgentProviderPricingConfig(
		AgentProviderAdapterTypeCodexAppServer,
		"gpt-5.4",
		0,
		0,
		map[string]any{"rates": map[string]any{"input_per_token": -1}},
	)
	if builtin.SourceKind != pricing.PricingSourceKindOfficial || builtin.ModelID != "gpt-5.4" {
		t.Fatalf("expected builtin fallback pricing config, got %+v", builtin)
	}

	manual := ResolveAgentProviderPricingConfig(
		AgentProviderAdapterTypeGeminiCLI,
		"auto-gemini-2.5",
		0.001,
		0.002,
		nil,
	)
	if manual.SourceKind != pricing.PricingSourceKindCustom || manual.Rates.InputPerToken != 0.001 || manual.Rates.OutputPerToken != 0.002 {
		t.Fatalf("expected manual fallback pricing config, got %+v", manual)
	}

	manualAfterInvalidRaw := ResolveAgentProviderPricingConfig(
		AgentProviderAdapterTypeGeminiCLI,
		"auto-gemini-2.5",
		0.003,
		0.004,
		map[string]any{"rates": map[string]any{"input_per_token": -1}},
	)
	if manualAfterInvalidRaw.SourceKind != pricing.PricingSourceKindCustom ||
		manualAfterInvalidRaw.Rates.InputPerToken != 0.003 ||
		manualAfterInvalidRaw.Rates.OutputPerToken != 0.004 {
		t.Fatalf("expected invalid raw pricing to fall back to manual flat pricing, got %+v", manualAfterInvalidRaw)
	}
}

func TestDeriveAgentProviderPricing(t *testing.T) {
	t.Parallel()

	derived := DeriveAgentProviderPricing(AgentProvider{
		AdapterType: AgentProviderAdapterTypeCodexAppServer,
		ModelName:   "gpt-5.4",
	})
	if derived.PricingConfig.SourceKind != pricing.PricingSourceKindOfficial {
		t.Fatalf("expected official derived pricing config, got %+v", derived.PricingConfig)
	}
	if derived.CostPerInputToken != derived.PricingConfig.SummaryInputPerToken() ||
		derived.CostPerOutputToken != derived.PricingConfig.SummaryOutputPerToken() {
		t.Fatalf("expected derived summary rates to mirror pricing config, got %+v", derived)
	}
}
