package ticketing

import (
	"fmt"
	"math"
)

type RawUsageDelta struct {
	InputTokens  *int64
	OutputTokens *int64
	CostUSD      *float64
}

type UsageDelta struct {
	InputTokens     int64
	OutputTokens    int64
	ExplicitCostUSD *float64
}

type ModelPricing struct {
	CostPerInputToken  float64
	CostPerOutputToken float64
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

	explicitCostUSD, err := parseOptionalNonNegativeFloat64("cost_usd", raw.CostUSD)
	if err != nil {
		return UsageDelta{}, err
	}

	if inputTokens == 0 && outputTokens == 0 && explicitCostUSD == nil {
		return UsageDelta{}, fmt.Errorf("usage delta must include input_tokens, output_tokens, or cost_usd")
	}

	return UsageDelta{
		InputTokens:     inputTokens,
		OutputTokens:    outputTokens,
		ExplicitCostUSD: explicitCostUSD,
	}, nil
}

func (u UsageDelta) TotalTokens() int64 {
	return u.InputTokens + u.OutputTokens
}

func (u UsageDelta) ResolveCostUSD(pricing ModelPricing) (float64, error) {
	if pricing.CostPerInputToken < 0 {
		return 0, fmt.Errorf("cost_per_input_token must be greater than or equal to zero")
	}
	if pricing.CostPerOutputToken < 0 {
		return 0, fmt.Errorf("cost_per_output_token must be greater than or equal to zero")
	}
	if u.ExplicitCostUSD != nil {
		return RoundUSD(*u.ExplicitCostUSD), nil
	}

	costUSD := float64(u.InputTokens)*pricing.CostPerInputToken +
		float64(u.OutputTokens)*pricing.CostPerOutputToken
	return RoundUSD(costUSD), nil
}

func RoundUSD(value float64) float64 {
	return math.Round(value*100) / 100
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

	rounded := RoundUSD(*value)
	return &rounded, nil
}
