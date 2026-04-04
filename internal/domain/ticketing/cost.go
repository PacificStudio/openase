package ticketing

import (
	"fmt"
	"math"

	"github.com/BetterAndBetterII/openase/internal/domain/pricing"
)

type UsageCostSource string

const (
	UsageCostSourceActual    UsageCostSource = "actual"
	UsageCostSourceEstimated UsageCostSource = "estimated"
	CostRecordedEventType    string          = "ticket.cost_recorded"
)

type RawUsageDelta struct {
	InputTokens                *int64
	OutputTokens               *int64
	CachedInputTokens          *int64
	CacheCreationInputTokens   *int64
	CacheCreationInputTokens5m *int64
	CacheCreationInputTokens1h *int64
	PromptTokens               *int64
	CandidateTokens            *int64
	ToolTokens                 *int64
	ReasoningTokens            *int64
	CacheStorageTokens         *int64
	CacheStorageHours          *float64
	ModelContextWindow         *int64
	CostUSD                    *float64
}

type UsageDelta struct {
	InputTokens                int64
	OutputTokens               int64
	CachedInputTokens          int64
	CacheCreationInputTokens   int64
	CacheCreationInputTokens5m int64
	CacheCreationInputTokens1h int64
	PromptTokens               int64
	CandidateTokens            int64
	ToolTokens                 int64
	ReasoningTokens            int64
	CacheStorageTokens         int64
	CacheStorageHours          float64
	ModelContextWindow         int64
	ExplicitCostUSD            *float64
}

type ResolvedUsageCost struct {
	AmountUSD float64
	Source    UsageCostSource
}

func ParseRawUsageDelta(raw RawUsageDelta) (UsageDelta, error) {
	inputTokens, err := parseNonNegativeInt64("input_tokens", raw.InputTokens)
	if err != nil {
		return UsageDelta{}, err
	}

	outputTokens, err := parseNonNegativeInt64("output_tokens", raw.OutputTokens)
	if err != nil {
		return UsageDelta{}, err
	}
	cachedInputTokens, err := parseNonNegativeInt64("cached_input_tokens", raw.CachedInputTokens)
	if err != nil {
		return UsageDelta{}, err
	}
	cacheCreationInputTokens, err := parseNonNegativeInt64("cache_creation_input_tokens", raw.CacheCreationInputTokens)
	if err != nil {
		return UsageDelta{}, err
	}
	cacheCreationInputTokens5m, err := parseNonNegativeInt64("cache_creation_input_tokens_5m", raw.CacheCreationInputTokens5m)
	if err != nil {
		return UsageDelta{}, err
	}
	cacheCreationInputTokens1h, err := parseNonNegativeInt64("cache_creation_input_tokens_1h", raw.CacheCreationInputTokens1h)
	if err != nil {
		return UsageDelta{}, err
	}
	promptTokens, err := parseNonNegativeInt64("prompt_tokens", raw.PromptTokens)
	if err != nil {
		return UsageDelta{}, err
	}
	candidateTokens, err := parseNonNegativeInt64("candidate_tokens", raw.CandidateTokens)
	if err != nil {
		return UsageDelta{}, err
	}
	toolTokens, err := parseNonNegativeInt64("tool_tokens", raw.ToolTokens)
	if err != nil {
		return UsageDelta{}, err
	}
	reasoningTokens, err := parseNonNegativeInt64("reasoning_tokens", raw.ReasoningTokens)
	if err != nil {
		return UsageDelta{}, err
	}
	cacheStorageTokens, err := parseNonNegativeInt64("cache_storage_tokens", raw.CacheStorageTokens)
	if err != nil {
		return UsageDelta{}, err
	}
	cacheStorageHours, err := parseNonNegativeFloat64Value("cache_storage_hours", raw.CacheStorageHours)
	if err != nil {
		return UsageDelta{}, err
	}
	modelContextWindow, err := parseNonNegativeInt64("model_context_window", raw.ModelContextWindow)
	if err != nil {
		return UsageDelta{}, err
	}

	explicitCostUSD, err := parseOptionalNonNegativeFloat64("cost_usd", raw.CostUSD)
	if err != nil {
		return UsageDelta{}, err
	}

	if inputTokens == 0 &&
		outputTokens == 0 &&
		cachedInputTokens == 0 &&
		cacheCreationInputTokens == 0 &&
		cacheCreationInputTokens5m == 0 &&
		cacheCreationInputTokens1h == 0 &&
		cacheStorageTokens == 0 &&
		explicitCostUSD == nil {
		return UsageDelta{}, fmt.Errorf("usage delta must include billable tokens, cache usage, or cost_usd")
	}

	return UsageDelta{
		InputTokens:                inputTokens,
		OutputTokens:               outputTokens,
		CachedInputTokens:          cachedInputTokens,
		CacheCreationInputTokens:   cacheCreationInputTokens,
		CacheCreationInputTokens5m: cacheCreationInputTokens5m,
		CacheCreationInputTokens1h: cacheCreationInputTokens1h,
		PromptTokens:               promptTokens,
		CandidateTokens:            candidateTokens,
		ToolTokens:                 toolTokens,
		ReasoningTokens:            reasoningTokens,
		CacheStorageTokens:         cacheStorageTokens,
		CacheStorageHours:          cacheStorageHours,
		ModelContextWindow:         modelContextWindow,
		ExplicitCostUSD:            explicitCostUSD,
	}, nil
}

func (u UsageDelta) TotalTokens() int64 {
	return u.InputTokens + u.OutputTokens
}

func (u UsageDelta) ResolveCost(pricingConfig pricing.ProviderModelPricingConfig) (ResolvedUsageCost, error) {
	normalizedPricing, err := pricing.NormalizeProviderModelPricingConfig(pricingConfig)
	if err != nil {
		return ResolvedUsageCost{}, err
	}
	if u.ExplicitCostUSD != nil {
		return ResolvedUsageCost{
			AmountUSD: *u.ExplicitCostUSD,
			Source:    UsageCostSourceActual,
		}, nil
	}

	rates := normalizedPricing.SelectRates(u.promptTokenCount())
	cacheWrite5mTokens, cacheWrite1hTokens := u.cacheWriteTokens(normalizedPricing.DefaultCacheWriteWindow)
	cachedReadTokens := u.CachedInputTokens
	standardInputTokens := u.InputTokens - cachedReadTokens - cacheWrite5mTokens - cacheWrite1hTokens
	if standardInputTokens < 0 {
		standardInputTokens = 0
	}

	amountUSD := float64(standardInputTokens)*rates.InputPerToken +
		float64(cachedReadTokens)*fallbackRate(rates.CachedInputReadPerToken, rates.InputPerToken) +
		float64(cacheWrite5mTokens)*fallbackRate(rates.CacheWriteFiveMinutesPerToken, rates.InputPerToken) +
		float64(cacheWrite1hTokens)*fallbackRate(rates.CacheWriteOneHourPerToken, rates.InputPerToken) +
		float64(u.OutputTokens)*rates.OutputPerToken
	if u.CacheStorageTokens > 0 && u.CacheStorageHours > 0 && rates.CacheStoragePerTokenHour > 0 {
		amountUSD += float64(u.CacheStorageTokens) * u.CacheStorageHours * rates.CacheStoragePerTokenHour
	}

	return ResolvedUsageCost{
		AmountUSD: amountUSD,
		Source:    UsageCostSourceEstimated,
	}, nil
}

func (u UsageDelta) ResolveCostUSD(pricingConfig pricing.ProviderModelPricingConfig) (float64, error) {
	resolved, err := u.ResolveCost(pricingConfig)
	if err != nil {
		return 0, err
	}

	return resolved.AmountUSD, nil
}

func RoundUSD(value float64) float64 {
	return math.Round(value*100) / 100
}

func (s UsageCostSource) String() string {
	return string(s)
}

func parseNonNegativeInt64(field string, value *int64) (int64, error) {
	if value == nil {
		return 0, nil
	}
	if *value < 0 {
		return 0, fmt.Errorf("%s must be greater than or equal to zero", field)
	}
	return *value, nil
}

func parseOptionalNonNegativeFloat64(field string, value *float64) (*float64, error) {
	if value == nil {
		return nil, nil
	}
	if *value < 0 {
		return nil, fmt.Errorf("%s must be greater than or equal to zero", field)
	}

	cloned := *value
	return &cloned, nil
}

func parseNonNegativeFloat64Value(field string, value *float64) (float64, error) {
	parsed, err := parseOptionalNonNegativeFloat64(field, value)
	if err != nil {
		return 0, err
	}
	if parsed == nil {
		return 0, nil
	}
	return *parsed, nil
}

func (u UsageDelta) promptTokenCount() int64 {
	if u.PromptTokens > 0 {
		return u.PromptTokens
	}
	return u.InputTokens
}

func (u UsageDelta) cacheWriteTokens(
	defaultWindow pricing.CacheWriteWindow,
) (int64, int64) {
	fiveMinutes := u.CacheCreationInputTokens5m
	oneHour := u.CacheCreationInputTokens1h
	if u.CacheCreationInputTokens > 0 && fiveMinutes == 0 && oneHour == 0 {
		if defaultWindow == pricing.CacheWriteWindowOneHour {
			oneHour = u.CacheCreationInputTokens
		} else {
			fiveMinutes = u.CacheCreationInputTokens
		}
	}
	return fiveMinutes, oneHour
}

func fallbackRate(primary float64, fallback float64) float64 {
	if primary > 0 {
		return primary
	}
	return fallback
}
