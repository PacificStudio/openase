package pricing

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PricingSourceKind string

const (
	PricingSourceKindCustom   PricingSourceKind = "custom"
	PricingSourceKindOfficial PricingSourceKind = "official"
)

func (k PricingSourceKind) IsValid() bool {
	return k == PricingSourceKindCustom || k == PricingSourceKindOfficial
}

type PricingMode string

const (
	PricingModeFlat   PricingMode = "flat"
	PricingModeTiered PricingMode = "tiered"
	PricingModeRouted PricingMode = "routed"
)

func (m PricingMode) IsValid() bool {
	return m == PricingModeFlat || m == PricingModeTiered || m == PricingModeRouted
}

type CacheWriteWindow string

const (
	CacheWriteWindowFiveMinutes CacheWriteWindow = "5m"
	CacheWriteWindowOneHour     CacheWriteWindow = "1h"
)

func (w CacheWriteWindow) IsValid() bool {
	return w == "" || w == CacheWriteWindowFiveMinutes || w == CacheWriteWindowOneHour
}

type ProviderModelPricingRates struct {
	InputPerToken                 float64 `json:"input_per_token,omitempty"`
	OutputPerToken                float64 `json:"output_per_token,omitempty"`
	CachedInputReadPerToken       float64 `json:"cached_input_read_per_token,omitempty"`
	CacheWriteFiveMinutesPerToken float64 `json:"cache_write_5m_per_token,omitempty"`
	CacheWriteOneHourPerToken     float64 `json:"cache_write_1h_per_token,omitempty"`
	CacheStoragePerTokenHour      float64 `json:"cache_storage_per_token_hour,omitempty"`
}

type ProviderModelPricingTier struct {
	Label           string                    `json:"label,omitempty"`
	MaxPromptTokens int64                     `json:"max_prompt_tokens,omitempty"`
	Rates           ProviderModelPricingRates `json:"rates"`
}

type ProviderModelPricingConfig struct {
	Version                 string                     `json:"version,omitempty"`
	SourceKind              PricingSourceKind          `json:"source_kind,omitempty"`
	PricingMode             PricingMode                `json:"pricing_mode,omitempty"`
	Provider                string                     `json:"provider,omitempty"`
	ModelID                 string                     `json:"model_id,omitempty"`
	SourceURL               string                     `json:"source_url,omitempty"`
	SourceVerifiedAt        string                     `json:"source_verified_at,omitempty"`
	DefaultCacheWriteWindow CacheWriteWindow           `json:"default_cache_write_window,omitempty"`
	Notes                   []string                   `json:"notes,omitempty"`
	Rates                   ProviderModelPricingRates  `json:"rates,omitempty"`
	Tiers                   []ProviderModelPricingTier `json:"tiers,omitempty"`
}

func CustomFlatPricingConfig(inputPerToken float64, outputPerToken float64) ProviderModelPricingConfig {
	return ProviderModelPricingConfig{
		SourceKind:  PricingSourceKindCustom,
		PricingMode: PricingModeFlat,
		Rates: ProviderModelPricingRates{
			InputPerToken:  inputPerToken,
			OutputPerToken: outputPerToken,
		},
	}
}

func ParseRawProviderModelPricingConfig(
	raw map[string]any,
	fallbackInputPerToken float64,
	fallbackOutputPerToken float64,
) (ProviderModelPricingConfig, error) {
	if len(raw) == 0 {
		return CustomFlatPricingConfig(fallbackInputPerToken, fallbackOutputPerToken), nil
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return ProviderModelPricingConfig{}, fmt.Errorf("marshal pricing_config: %w", err)
	}

	var config ProviderModelPricingConfig
	if err := json.Unmarshal(payload, &config); err != nil {
		return ProviderModelPricingConfig{}, fmt.Errorf("decode pricing_config: %w", err)
	}

	return NormalizeProviderModelPricingConfig(config)
}

func NormalizeProviderModelPricingConfig(
	config ProviderModelPricingConfig,
) (ProviderModelPricingConfig, error) {
	config.Version = strings.TrimSpace(config.Version)
	config.Provider = strings.TrimSpace(config.Provider)
	config.ModelID = strings.TrimSpace(config.ModelID)
	config.SourceURL = strings.TrimSpace(config.SourceURL)
	config.SourceVerifiedAt = strings.TrimSpace(config.SourceVerifiedAt)
	config.Notes = cloneNonEmptyStrings(config.Notes)

	if config.SourceKind == "" {
		config.SourceKind = PricingSourceKindCustom
	}
	if !config.SourceKind.IsValid() {
		return ProviderModelPricingConfig{}, fmt.Errorf("pricing_config.source_kind must be custom or official")
	}

	if config.PricingMode == "" {
		config.PricingMode = PricingModeFlat
	}
	if !config.PricingMode.IsValid() {
		return ProviderModelPricingConfig{}, fmt.Errorf("pricing_config.pricing_mode must be flat, tiered, or routed")
	}
	if !config.DefaultCacheWriteWindow.IsValid() {
		return ProviderModelPricingConfig{}, fmt.Errorf("pricing_config.default_cache_write_window must be 5m or 1h")
	}

	if err := validatePricingRates("pricing_config.rates", config.Rates); err != nil {
		return ProviderModelPricingConfig{}, err
	}

	if len(config.Tiers) > 0 {
		var previousMax int64
		for index, tier := range config.Tiers {
			tier.Label = strings.TrimSpace(tier.Label)
			if tier.MaxPromptTokens < 0 {
				return ProviderModelPricingConfig{}, fmt.Errorf("pricing_config.tiers[%d].max_prompt_tokens must be greater than or equal to zero", index)
			}
			if tier.MaxPromptTokens > 0 && tier.MaxPromptTokens <= previousMax {
				return ProviderModelPricingConfig{}, fmt.Errorf("pricing_config.tiers must be sorted by max_prompt_tokens")
			}
			if tier.MaxPromptTokens > 0 {
				previousMax = tier.MaxPromptTokens
			}
			if err := validatePricingRates(fmt.Sprintf("pricing_config.tiers[%d].rates", index), tier.Rates); err != nil {
				return ProviderModelPricingConfig{}, err
			}
			config.Tiers[index] = tier
		}
		if config.PricingMode == PricingModeFlat {
			config.PricingMode = PricingModeTiered
		}
	}

	return config, nil
}

func (c ProviderModelPricingConfig) Clone() ProviderModelPricingConfig {
	cloned := c
	cloned.Notes = append([]string(nil), c.Notes...)
	if len(c.Tiers) > 0 {
		cloned.Tiers = append([]ProviderModelPricingTier(nil), c.Tiers...)
	}
	return cloned
}

func (c ProviderModelPricingConfig) ToMap() map[string]any {
	if c.IsZero() {
		return map[string]any{}
	}

	payload, err := json.Marshal(c)
	if err != nil {
		return map[string]any{}
	}

	var decoded map[string]any
	_ = json.Unmarshal(payload, &decoded)
	return decoded
}

func (c ProviderModelPricingConfig) IsZero() bool {
	return c.SourceKind == "" &&
		c.PricingMode == "" &&
		c.Provider == "" &&
		c.ModelID == "" &&
		c.SourceURL == "" &&
		c.SourceVerifiedAt == "" &&
		c.DefaultCacheWriteWindow == "" &&
		len(c.Notes) == 0 &&
		len(c.Tiers) == 0 &&
		c.Rates == (ProviderModelPricingRates{})
}

func (c ProviderModelPricingConfig) SummaryInputPerToken() float64 {
	if len(c.Tiers) > 0 {
		return c.Tiers[0].Rates.InputPerToken
	}
	return c.Rates.InputPerToken
}

func (c ProviderModelPricingConfig) SummaryOutputPerToken() float64 {
	if len(c.Tiers) > 0 {
		return c.Tiers[0].Rates.OutputPerToken
	}
	return c.Rates.OutputPerToken
}

func (c ProviderModelPricingConfig) HasAnyPricing() bool {
	if c.SummaryInputPerToken() > 0 || c.SummaryOutputPerToken() > 0 {
		return true
	}

	rates := c.Rates
	if rates.CachedInputReadPerToken > 0 ||
		rates.CacheWriteFiveMinutesPerToken > 0 ||
		rates.CacheWriteOneHourPerToken > 0 ||
		rates.CacheStoragePerTokenHour > 0 {
		return true
	}

	for _, tier := range c.Tiers {
		if tier.Rates.InputPerToken > 0 ||
			tier.Rates.OutputPerToken > 0 ||
			tier.Rates.CachedInputReadPerToken > 0 ||
			tier.Rates.CacheWriteFiveMinutesPerToken > 0 ||
			tier.Rates.CacheWriteOneHourPerToken > 0 ||
			tier.Rates.CacheStoragePerTokenHour > 0 {
			return true
		}
	}

	return false
}

func (c ProviderModelPricingConfig) SelectRates(promptTokens int64) ProviderModelPricingRates {
	if len(c.Tiers) == 0 {
		return c.Rates
	}
	if promptTokens <= 0 {
		return c.Tiers[0].Rates
	}
	for _, tier := range c.Tiers {
		if tier.MaxPromptTokens == 0 || promptTokens <= tier.MaxPromptTokens {
			return tier.Rates
		}
	}
	return c.Tiers[len(c.Tiers)-1].Rates
}

func (c ProviderModelPricingConfig) CacheWriteRate(window CacheWriteWindow, promptTokens int64) float64 {
	rates := c.SelectRates(promptTokens)
	switch window {
	case CacheWriteWindowOneHour:
		if rates.CacheWriteOneHourPerToken > 0 {
			return rates.CacheWriteOneHourPerToken
		}
		return rates.CacheWriteFiveMinutesPerToken
	default:
		if rates.CacheWriteFiveMinutesPerToken > 0 {
			return rates.CacheWriteFiveMinutesPerToken
		}
		return rates.CacheWriteOneHourPerToken
	}
}

func validatePricingRates(field string, rates ProviderModelPricingRates) error {
	values := map[string]float64{
		"input_per_token":              rates.InputPerToken,
		"output_per_token":             rates.OutputPerToken,
		"cached_input_read_per_token":  rates.CachedInputReadPerToken,
		"cache_write_5m_per_token":     rates.CacheWriteFiveMinutesPerToken,
		"cache_write_1h_per_token":     rates.CacheWriteOneHourPerToken,
		"cache_storage_per_token_hour": rates.CacheStoragePerTokenHour,
	}
	for name, value := range values {
		if value < 0 {
			return fmt.Errorf("%s.%s must be greater than or equal to zero", field, name)
		}
	}
	return nil
}

func cloneNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	cloned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			cloned = append(cloned, trimmed)
		}
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}
