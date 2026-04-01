package catalog

import (
	"strings"

	"github.com/BetterAndBetterII/openase/internal/domain/pricing"
)

const builtinPricingVerifiedAt = "2026-04-01"

var builtinAgentProviderPricingConfigs = map[AgentProviderAdapterType]map[string]pricing.ProviderModelPricingConfig{
	AgentProviderAdapterTypeCodexAppServer: {
		"gpt-5.4":            openAIPricingConfig("gpt-5.4", 2.50, 15.00, 0.25),
		"gpt-5.4-mini":       openAIPricingConfig("gpt-5.4-mini", 0.75, 4.50, 0.075),
		"gpt-5.3-codex":      openAIPricingConfig("gpt-5.3-codex", 1.75, 14.00, 0.175),
		"gpt-5.2-codex":      openAIPricingConfig("gpt-5.2-codex", 1.75, 14.00, 0.175),
		"gpt-5.2":            openAIPricingConfig("gpt-5.2", 1.75, 14.00, 0.175),
		"gpt-5.1-codex-max":  openAIPricingConfig("gpt-5.1-codex-max", 1.25, 10.00, 0.125),
		"gpt-5.1-codex-mini": openAIPricingConfig("gpt-5.1-codex-mini", 0.25, 2.00, 0.025),
	},
	AgentProviderAdapterTypeClaudeCodeCLI: {
		"claude-opus-4-6": anthropicPricingConfig(
			"claude-opus-4-6",
			"Claude Opus 4.6 pricing is mapped from Anthropic Claude API list pricing.",
			5.00,
			25.00,
		),
		"claude-sonnet-4-6": anthropicPricingConfig(
			"claude-sonnet-4-6",
			"Claude Sonnet 4.6 pricing is mapped from Anthropic Claude API list pricing.",
			3.00,
			15.00,
		),
		"claude-haiku-4-5": anthropicPricingConfig(
			"claude-haiku-4-5",
			"Claude Haiku 4.5 pricing is mapped from Anthropic Claude API list pricing.",
			1.00,
			5.00,
		),
	},
	AgentProviderAdapterTypeGeminiCLI: {
		"auto-gemini-2.5": routedGeminiPricingConfig(
			"auto-gemini-2.5",
			"Auto-routed Gemini 2.5 requests can resolve to different billable models at runtime.",
		),
		"gemini-2.5-pro": geminiTieredPricingConfig(
			"gemini-2.5-pro",
			[]pricing.ProviderModelPricingTier{
				{
					Label:           "<=200k prompt tokens",
					MaxPromptTokens: 200_000,
					Rates: pricing.ProviderModelPricingRates{
						InputPerToken:            usdPerMillion(1.25),
						OutputPerToken:           usdPerMillion(10.00),
						CachedInputReadPerToken:  usdPerMillion(0.125),
						CacheStoragePerTokenHour: usdPerMillion(4.50),
					},
				},
				{
					Label: " >200k prompt tokens",
					Rates: pricing.ProviderModelPricingRates{
						InputPerToken:            usdPerMillion(2.50),
						OutputPerToken:           usdPerMillion(15.00),
						CachedInputReadPerToken:  usdPerMillion(0.25),
						CacheStoragePerTokenHour: usdPerMillion(4.50),
					},
				},
			},
			"Standard tier pricing from Gemini Developer API pricing.",
		),
		"gemini-2.5-flash": geminiFlatPricingConfig(
			"gemini-2.5-flash",
			0.30,
			2.50,
			0.03,
			1.00,
		),
		"gemini-2.5-flash-lite": geminiFlatPricingConfig(
			"gemini-2.5-flash-lite",
			0.10,
			0.40,
			0.01,
			1.00,
		),
		"auto-gemini-3": routedGeminiPricingConfig(
			"auto-gemini-3",
			"Auto-routed Gemini 3 requests can resolve to different preview billable models at runtime.",
		),
	},
}

func BuiltinAgentProviderPricingConfig(
	adapterType AgentProviderAdapterType,
	modelID string,
) (pricing.ProviderModelPricingConfig, bool) {
	models := builtinAgentProviderPricingConfigs[adapterType]
	if len(models) == 0 {
		return pricing.ProviderModelPricingConfig{}, false
	}

	config, ok := models[strings.TrimSpace(modelID)]
	if !ok {
		return pricing.ProviderModelPricingConfig{}, false
	}

	return config.Clone(), true
}

func ResolveAgentProviderPricingConfig(
	adapterType AgentProviderAdapterType,
	modelID string,
	costPerInputToken float64,
	costPerOutputToken float64,
	raw map[string]any,
) pricing.ProviderModelPricingConfig {
	config, err := pricing.ParseRawProviderModelPricingConfig(raw, costPerInputToken, costPerOutputToken)
	if err == nil && !config.IsZero() && config.HasAnyPricing() {
		return config
	}

	if costPerInputToken == 0 && costPerOutputToken == 0 {
		if builtin, ok := BuiltinAgentProviderPricingConfig(adapterType, modelID); ok {
			return builtin
		}
	}

	return pricing.CustomFlatPricingConfig(costPerInputToken, costPerOutputToken)
}

func DeriveAgentProviderPricing(item AgentProvider) AgentProvider {
	config := ResolveAgentProviderPricingConfig(
		item.AdapterType,
		item.ModelName,
		item.CostPerInputToken,
		item.CostPerOutputToken,
		item.PricingConfig.ToMap(),
	)
	item.PricingConfig = config
	item.CostPerInputToken = config.SummaryInputPerToken()
	item.CostPerOutputToken = config.SummaryOutputPerToken()
	return item
}

func openAIPricingConfig(
	modelID string,
	inputUSDPerMillion float64,
	outputUSDPerMillion float64,
	cachedInputUSDPerMillion float64,
) pricing.ProviderModelPricingConfig {
	return pricing.ProviderModelPricingConfig{
		Version:          builtinPricingVerifiedAt,
		SourceKind:       pricing.PricingSourceKindOfficial,
		PricingMode:      pricing.PricingModeFlat,
		Provider:         "openai",
		ModelID:          modelID,
		SourceURL:        "https://platform.openai.com/docs/pricing/",
		SourceVerifiedAt: builtinPricingVerifiedAt,
		Notes: []string{
			"Official OpenAI API pricing defaults.",
		},
		Rates: pricing.ProviderModelPricingRates{
			InputPerToken:           usdPerMillion(inputUSDPerMillion),
			OutputPerToken:          usdPerMillion(outputUSDPerMillion),
			CachedInputReadPerToken: usdPerMillion(cachedInputUSDPerMillion),
		},
	}
}

func anthropicPricingConfig(
	modelID string,
	mappingNote string,
	inputUSDPerMillion float64,
	outputUSDPerMillion float64,
) pricing.ProviderModelPricingConfig {
	return pricing.ProviderModelPricingConfig{
		Version:                 builtinPricingVerifiedAt,
		SourceKind:              pricing.PricingSourceKindOfficial,
		PricingMode:             pricing.PricingModeFlat,
		Provider:                "anthropic",
		ModelID:                 modelID,
		SourceURL:               "https://platform.claude.com/docs/en/about-claude/pricing",
		SourceVerifiedAt:        builtinPricingVerifiedAt,
		DefaultCacheWriteWindow: pricing.CacheWriteWindowFiveMinutes,
		Notes: []string{
			mappingNote,
			"Anthropic prompt caching cache hits are priced at 10% of base input, 5-minute writes at 1.25x, and 1-hour writes at 2x.",
		},
		Rates: pricing.ProviderModelPricingRates{
			InputPerToken:                 usdPerMillion(inputUSDPerMillion),
			OutputPerToken:                usdPerMillion(outputUSDPerMillion),
			CachedInputReadPerToken:       usdPerMillion(inputUSDPerMillion * 0.10),
			CacheWriteFiveMinutesPerToken: usdPerMillion(inputUSDPerMillion * 1.25),
			CacheWriteOneHourPerToken:     usdPerMillion(inputUSDPerMillion * 2.00),
		},
	}
}

func geminiFlatPricingConfig(
	modelID string,
	inputUSDPerMillion float64,
	outputUSDPerMillion float64,
	cachedInputUSDPerMillion float64,
	cacheStorageUSDPerMillionHour float64,
) pricing.ProviderModelPricingConfig {
	return pricing.ProviderModelPricingConfig{
		Version:          builtinPricingVerifiedAt,
		SourceKind:       pricing.PricingSourceKindOfficial,
		PricingMode:      pricing.PricingModeFlat,
		Provider:         "google",
		ModelID:          modelID,
		SourceURL:        "https://ai.google.dev/gemini-api/docs/pricing",
		SourceVerifiedAt: builtinPricingVerifiedAt,
		Notes: []string{
			"Standard Gemini Developer API pricing for text, image, and video requests.",
		},
		Rates: pricing.ProviderModelPricingRates{
			InputPerToken:            usdPerMillion(inputUSDPerMillion),
			OutputPerToken:           usdPerMillion(outputUSDPerMillion),
			CachedInputReadPerToken:  usdPerMillion(cachedInputUSDPerMillion),
			CacheStoragePerTokenHour: usdPerMillion(cacheStorageUSDPerMillionHour),
		},
	}
}

func geminiTieredPricingConfig(
	modelID string,
	tiers []pricing.ProviderModelPricingTier,
	note string,
) pricing.ProviderModelPricingConfig {
	return pricing.ProviderModelPricingConfig{
		Version:          builtinPricingVerifiedAt,
		SourceKind:       pricing.PricingSourceKindOfficial,
		PricingMode:      pricing.PricingModeTiered,
		Provider:         "google",
		ModelID:          modelID,
		SourceURL:        "https://ai.google.dev/gemini-api/docs/pricing",
		SourceVerifiedAt: builtinPricingVerifiedAt,
		Notes: []string{
			note,
		},
		Tiers: append([]pricing.ProviderModelPricingTier(nil), tiers...),
	}
}

func routedGeminiPricingConfig(modelID string, note string) pricing.ProviderModelPricingConfig {
	return pricing.ProviderModelPricingConfig{
		Version:          builtinPricingVerifiedAt,
		SourceKind:       pricing.PricingSourceKindOfficial,
		PricingMode:      pricing.PricingModeRouted,
		Provider:         "google",
		ModelID:          modelID,
		SourceURL:        "https://ai.google.dev/gemini-api/docs/pricing",
		SourceVerifiedAt: builtinPricingVerifiedAt,
		Notes: []string{
			note,
		},
	}
}

func usdPerMillion(value float64) float64 {
	return value / 1_000_000
}
