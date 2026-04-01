package pricing

import (
	"math"
	"strings"
	"testing"
)

func TestPricingSourceKindIsValid(t *testing.T) {
	t.Parallel()

	if !PricingSourceKindCustom.IsValid() || !PricingSourceKindOfficial.IsValid() {
		t.Fatalf("expected known pricing source kinds to be valid")
	}
	if PricingSourceKind("nope").IsValid() {
		t.Fatalf("expected unknown pricing source kind to be invalid")
	}
}

func TestPricingModeIsValid(t *testing.T) {
	t.Parallel()

	if !PricingModeFlat.IsValid() || !PricingModeTiered.IsValid() || !PricingModeRouted.IsValid() {
		t.Fatalf("expected known pricing modes to be valid")
	}
	if PricingMode("nope").IsValid() {
		t.Fatalf("expected unknown pricing mode to be invalid")
	}
}

func TestCacheWriteWindowIsValid(t *testing.T) {
	t.Parallel()

	if !CacheWriteWindow("").IsValid() ||
		!CacheWriteWindowFiveMinutes.IsValid() ||
		!CacheWriteWindowOneHour.IsValid() {
		t.Fatalf("expected known cache write windows to be valid")
	}
	if CacheWriteWindow("24h").IsValid() {
		t.Fatalf("expected unknown cache write window to be invalid")
	}
}

func TestCustomFlatPricingConfig(t *testing.T) {
	t.Parallel()

	config := CustomFlatPricingConfig(1.25, 9.5)
	if config.SourceKind != PricingSourceKindCustom {
		t.Fatalf("expected custom source kind, got %q", config.SourceKind)
	}
	if config.PricingMode != PricingModeFlat {
		t.Fatalf("expected flat pricing mode, got %q", config.PricingMode)
	}
	if config.Rates.InputPerToken != 1.25 || config.Rates.OutputPerToken != 9.5 {
		t.Fatalf("unexpected flat pricing config: %+v", config.Rates)
	}
}

func TestParseRawProviderModelPricingConfigUsesFallbackForEmptyMap(t *testing.T) {
	t.Parallel()

	config, err := ParseRawProviderModelPricingConfig(map[string]any{}, 0.1, 0.2)
	if err != nil {
		t.Fatalf("expected fallback config, got error: %v", err)
	}
	if config.SummaryInputPerToken() != 0.1 || config.SummaryOutputPerToken() != 0.2 {
		t.Fatalf("unexpected fallback config: %+v", config)
	}
}

func TestParseRawProviderModelPricingConfigRejectsMarshalAndDecodeErrors(t *testing.T) {
	t.Parallel()

	_, err := ParseRawProviderModelPricingConfig(map[string]any{"bad": make(chan int)}, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "marshal pricing_config") {
		t.Fatalf("expected marshal error, got %v", err)
	}

	_, err = ParseRawProviderModelPricingConfig(map[string]any{"rates": map[string]any{"input_per_token": "oops"}}, 0, 0)
	if err == nil || !strings.Contains(err.Error(), "decode pricing_config") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestParseAndNormalizeProviderModelPricingConfig(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"version":                    " v1 ",
		"provider":                   " openai ",
		"model_id":                   " gpt-5.4 ",
		"source_url":                 " https://example.com ",
		"source_verified_at":         " 2026-04-01 ",
		"default_cache_write_window": "5m",
		"notes":                      []any{" official ", "", " cached "},
		"tiers": []any{
			map[string]any{
				"label":             " short ",
				"max_prompt_tokens": float64(200000),
				"rates": map[string]any{
					"input_per_token":  1.0,
					"output_per_token": 2.0,
				},
			},
			map[string]any{
				"label": " long ",
				"rates": map[string]any{
					"input_per_token":          3.0,
					"output_per_token":         4.0,
					"cache_write_1h_per_token": 5.0,
				},
			},
		},
	}

	config, err := ParseRawProviderModelPricingConfig(raw, 0, 0)
	if err != nil {
		t.Fatalf("expected normalized config, got error: %v", err)
	}
	if config.Version != "v1" || config.Provider != "openai" || config.ModelID != "gpt-5.4" {
		t.Fatalf("expected trimmed metadata, got %+v", config)
	}
	if config.SourceKind != PricingSourceKindCustom {
		t.Fatalf("expected default custom source kind, got %q", config.SourceKind)
	}
	if config.PricingMode != PricingModeTiered {
		t.Fatalf("expected flat-with-tiers to normalize to tiered, got %q", config.PricingMode)
	}
	if len(config.Notes) != 2 || config.Notes[0] != "official" || config.Notes[1] != "cached" {
		t.Fatalf("expected trimmed non-empty notes, got %#v", config.Notes)
	}
	if config.Tiers[0].Label != "short" || config.Tiers[1].Label != "long" {
		t.Fatalf("expected trimmed tier labels, got %#v", config.Tiers)
	}
}

func TestNormalizeProviderModelPricingConfigRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		config ProviderModelPricingConfig
		want   string
	}{
		{
			name:   "invalid source kind",
			config: ProviderModelPricingConfig{SourceKind: PricingSourceKind("bad")},
			want:   "pricing_config.source_kind must be custom or official",
		},
		{
			name:   "invalid pricing mode",
			config: ProviderModelPricingConfig{PricingMode: PricingMode("bad")},
			want:   "pricing_config.pricing_mode must be flat, tiered, or routed",
		},
		{
			name:   "invalid cache window",
			config: ProviderModelPricingConfig{DefaultCacheWriteWindow: CacheWriteWindow("bad")},
			want:   "pricing_config.default_cache_write_window must be 5m or 1h",
		},
		{
			name: "negative base rate",
			config: ProviderModelPricingConfig{
				Rates: ProviderModelPricingRates{InputPerToken: -1},
			},
			want: "pricing_config.rates.input_per_token must be greater than or equal to zero",
		},
		{
			name: "negative tier prompt tokens",
			config: ProviderModelPricingConfig{
				Tiers: []ProviderModelPricingTier{{MaxPromptTokens: -1}},
			},
			want: "pricing_config.tiers[0].max_prompt_tokens must be greater than or equal to zero",
		},
		{
			name: "unsorted tiers",
			config: ProviderModelPricingConfig{
				Tiers: []ProviderModelPricingTier{
					{MaxPromptTokens: 10},
					{MaxPromptTokens: 5},
				},
			},
			want: "pricing_config.tiers must be sorted by max_prompt_tokens",
		},
		{
			name: "negative tier rate",
			config: ProviderModelPricingConfig{
				Tiers: []ProviderModelPricingTier{
					{
						MaxPromptTokens: 10,
						Rates: ProviderModelPricingRates{
							CacheStoragePerTokenHour: -1,
						},
					},
				},
			},
			want: "pricing_config.tiers[0].rates.cache_storage_per_token_hour must be greater than or equal to zero",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := NormalizeProviderModelPricingConfig(tc.config)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestProviderModelPricingConfigCloneIsZeroAndToMap(t *testing.T) {
	t.Parallel()

	var zero ProviderModelPricingConfig
	if !zero.IsZero() {
		t.Fatalf("expected zero config to be zero")
	}
	if len(zero.ToMap()) != 0 {
		t.Fatalf("expected zero config map to be empty")
	}

	config := ProviderModelPricingConfig{
		SourceKind:  PricingSourceKindOfficial,
		PricingMode: PricingModeTiered,
		Notes:       []string{"official"},
		Tiers: []ProviderModelPricingTier{
			{Label: "short", Rates: ProviderModelPricingRates{InputPerToken: 1}},
		},
	}
	cloned := config.Clone()
	cloned.Notes[0] = "changed"
	cloned.Tiers[0].Label = "changed"
	if config.Notes[0] != "official" || config.Tiers[0].Label != "short" {
		t.Fatalf("expected clone to avoid mutating original: %+v", config)
	}
	if len(config.ToMap()) == 0 {
		t.Fatalf("expected non-zero config to serialize to map")
	}

	nonSerializable := ProviderModelPricingConfig{
		SourceKind: PricingSourceKindOfficial,
		Rates: ProviderModelPricingRates{
			InputPerToken: math.NaN(),
		},
	}
	if len(nonSerializable.ToMap()) != 0 {
		t.Fatalf("expected NaN pricing to fail marshal and return empty map")
	}
}

func TestProviderModelPricingConfigPricingHelpers(t *testing.T) {
	t.Parallel()

	config := ProviderModelPricingConfig{
		Rates: ProviderModelPricingRates{
			InputPerToken:                 1,
			OutputPerToken:                2,
			CachedInputReadPerToken:       3,
			CacheWriteFiveMinutesPerToken: 4,
			CacheWriteOneHourPerToken:     5,
			CacheStoragePerTokenHour:      6,
		},
	}
	if !config.HasAnyPricing() {
		t.Fatalf("expected base rates to count as pricing")
	}
	if got := config.SelectRates(10); got.InputPerToken != 1 || got.OutputPerToken != 2 {
		t.Fatalf("expected base rates, got %+v", got)
	}
	if config.CacheWriteRate(CacheWriteWindowOneHour, 10) != 5 {
		t.Fatalf("expected one hour cache write rate")
	}
	if config.CacheWriteRate(CacheWriteWindow(""), 10) != 4 {
		t.Fatalf("expected default cache write rate")
	}

	cacheOnly := ProviderModelPricingConfig{
		Rates: ProviderModelPricingRates{CachedInputReadPerToken: 3},
	}
	if !cacheOnly.HasAnyPricing() {
		t.Fatalf("expected cache-only base pricing to count as pricing")
	}

	tiered := ProviderModelPricingConfig{
		Tiers: []ProviderModelPricingTier{
			{MaxPromptTokens: 100, Rates: ProviderModelPricingRates{InputPerToken: 7, OutputPerToken: 8, CacheWriteFiveMinutesPerToken: 9}},
			{Rates: ProviderModelPricingRates{InputPerToken: 10, OutputPerToken: 11, CacheWriteOneHourPerToken: 12}},
		},
	}
	if tiered.SummaryInputPerToken() != 7 || tiered.SummaryOutputPerToken() != 8 {
		t.Fatalf("expected tier summary rates")
	}
	if !tiered.HasAnyPricing() {
		t.Fatalf("expected tier rates to count as pricing")
	}
	if got := tiered.SelectRates(0); got.InputPerToken != 7 {
		t.Fatalf("expected first tier for non-positive prompt tokens, got %+v", got)
	}
	if got := tiered.SelectRates(50); got.InputPerToken != 7 {
		t.Fatalf("expected first matching tier, got %+v", got)
	}
	if got := tiered.SelectRates(500); got.InputPerToken != 10 {
		t.Fatalf("expected fallback to last tier, got %+v", got)
	}
	if tiered.CacheWriteRate(CacheWriteWindowOneHour, 50) != 9 {
		t.Fatalf("expected one hour fallback to five minute rate")
	}
	if tiered.CacheWriteRate(CacheWriteWindow(""), 500) != 12 {
		t.Fatalf("expected default fallback to one hour rate")
	}

	openEnded := ProviderModelPricingConfig{
		Tiers: []ProviderModelPricingTier{
			{MaxPromptTokens: 100, Rates: ProviderModelPricingRates{InputPerToken: 1}},
			{MaxPromptTokens: 0, Rates: ProviderModelPricingRates{CachedInputReadPerToken: 2}},
		},
	}
	if !openEnded.HasAnyPricing() {
		t.Fatalf("expected cache-only tier pricing to count as pricing")
	}
	if got := openEnded.SelectRates(101); got.CachedInputReadPerToken != 2 {
		t.Fatalf("expected open-ended tier rates, got %+v", got)
	}

	bounded := ProviderModelPricingConfig{
		Tiers: []ProviderModelPricingTier{
			{MaxPromptTokens: 100, Rates: ProviderModelPricingRates{InputPerToken: 1}},
			{MaxPromptTokens: 200, Rates: ProviderModelPricingRates{OutputPerToken: 2}},
		},
	}
	if got := bounded.SelectRates(150); got.OutputPerToken != 2 {
		t.Fatalf("expected bounded second tier rates, got %+v", got)
	}
	if got := bounded.SelectRates(250); got.OutputPerToken != 2 {
		t.Fatalf("expected bounded tiers to fall back to last rates, got %+v", got)
	}
}

func TestProviderModelPricingConfigHasAnyPricingFalseWhenEmpty(t *testing.T) {
	t.Parallel()

	if (ProviderModelPricingConfig{}).HasAnyPricing() {
		t.Fatalf("expected empty pricing config to have no pricing")
	}

	zeroTiers := ProviderModelPricingConfig{
		Tiers: []ProviderModelPricingTier{
			{MaxPromptTokens: 100},
			{},
		},
	}
	if zeroTiers.HasAnyPricing() {
		t.Fatalf("expected zero-value tiers to have no pricing")
	}

	tierOnly := ProviderModelPricingConfig{
		Tiers: []ProviderModelPricingTier{
			{MaxPromptTokens: 100},
			{Rates: ProviderModelPricingRates{CacheStoragePerTokenHour: 1}},
		},
	}
	if !tierOnly.HasAnyPricing() {
		t.Fatalf("expected non-summary tier pricing to count as pricing")
	}
}

func TestNormalizeProviderModelPricingConfigDropsBlankNotes(t *testing.T) {
	t.Parallel()

	config, err := NormalizeProviderModelPricingConfig(ProviderModelPricingConfig{
		Notes: []string{" ", ""},
	})
	if err != nil {
		t.Fatalf("expected blank-note config to normalize, got %v", err)
	}
	if config.Notes != nil {
		t.Fatalf("expected blank notes to normalize to nil, got %#v", config.Notes)
	}
}
