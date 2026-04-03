package provider

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type CLIRateLimitProvider string

const (
	CLIRateLimitProviderClaudeCode CLIRateLimitProvider = "claude_code"
	CLIRateLimitProviderCodex      CLIRateLimitProvider = "codex"
	CLIRateLimitProviderGemini     CLIRateLimitProvider = "gemini"
)

type CLIRateLimit struct {
	Provider   CLIRateLimitProvider `json:"provider"`
	ClaudeCode *ClaudeCodeRateLimit `json:"claude_code,omitempty"`
	Codex      *CodexRateLimit      `json:"codex,omitempty"`
	Gemini     *GeminiRateLimit     `json:"gemini,omitempty"`
	Raw        map[string]any       `json:"raw,omitempty"`
}

type ClaudeCodeRateLimit struct {
	Status                string     `json:"status,omitempty"`
	RateLimitType         string     `json:"rate_limit_type,omitempty"`
	ResetsAt              *time.Time `json:"resets_at,omitempty"`
	Utilization           *float64   `json:"utilization,omitempty"`
	SurpassedThreshold    *float64   `json:"surpassed_threshold,omitempty"`
	OverageStatus         string     `json:"overage_status,omitempty"`
	OverageDisabledReason string     `json:"overage_disabled_reason,omitempty"`
	IsUsingOverage        *bool      `json:"is_using_overage,omitempty"`
}

type CodexRateLimit struct {
	LimitID   string                `json:"limit_id,omitempty"`
	LimitName string                `json:"limit_name,omitempty"`
	Primary   *CodexRateLimitWindow `json:"primary,omitempty"`
	Secondary *CodexRateLimitWindow `json:"secondary,omitempty"`
	PlanType  string                `json:"plan_type,omitempty"`
}

type CodexRateLimitWindow struct {
	UsedPercent   *float64   `json:"used_percent,omitempty"`
	WindowMinutes int64      `json:"window_minutes,omitempty"`
	ResetsAt      *time.Time `json:"resets_at,omitempty"`
}

type GeminiRateLimit struct {
	AuthType  string                  `json:"auth_type,omitempty"`
	Remaining *int64                  `json:"remaining,omitempty"`
	Limit     *int64                  `json:"limit,omitempty"`
	ResetTime *time.Time              `json:"reset_time,omitempty"`
	Buckets   []GeminiRateLimitBucket `json:"buckets,omitempty"`
}

type GeminiRateLimitBucket struct {
	ModelID           string     `json:"model_id,omitempty"`
	TokenType         string     `json:"token_type,omitempty"`
	RemainingAmount   string     `json:"remaining_amount,omitempty"`
	RemainingFraction *float64   `json:"remaining_fraction,omitempty"`
	ResetTime         *time.Time `json:"reset_time,omitempty"`
}

func ParseClaudeCodeRateLimit(raw json.RawMessage) (*CLIRateLimit, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var payload struct {
		Status                string   `json:"status"`
		ResetsAt              *int64   `json:"resetsAt"`
		RateLimitType         string   `json:"rateLimitType"`
		Utilization           *float64 `json:"utilization"`
		SurpassedThreshold    *float64 `json:"surpassedThreshold"`
		OverageStatus         string   `json:"overageStatus"`
		OverageDisabledReason string   `json:"overageDisabledReason"`
		IsUsingOverage        *bool    `json:"isUsingOverage"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse claude code rate limit: %w", err)
	}

	rawMap, err := decodeRateLimitRawMap(raw)
	if err != nil {
		return nil, fmt.Errorf("parse claude code rate limit raw payload: %w", err)
	}

	rateLimit := &CLIRateLimit{
		Provider: CLIRateLimitProviderClaudeCode,
		ClaudeCode: &ClaudeCodeRateLimit{
			Status:                strings.TrimSpace(payload.Status),
			RateLimitType:         strings.TrimSpace(payload.RateLimitType),
			ResetsAt:              unixSecondsPointerToTime(payload.ResetsAt),
			Utilization:           cloneRateLimitFloatPointer(payload.Utilization),
			SurpassedThreshold:    cloneRateLimitFloatPointer(payload.SurpassedThreshold),
			OverageStatus:         strings.TrimSpace(payload.OverageStatus),
			OverageDisabledReason: strings.TrimSpace(payload.OverageDisabledReason),
			IsUsingOverage:        cloneRateLimitBoolPointer(payload.IsUsingOverage),
		},
		Raw: rawMap,
	}
	if rateLimit.ClaudeCode.Status == "" &&
		rateLimit.ClaudeCode.RateLimitType == "" &&
		rateLimit.ClaudeCode.ResetsAt == nil &&
		rateLimit.ClaudeCode.Utilization == nil &&
		rateLimit.ClaudeCode.SurpassedThreshold == nil &&
		rateLimit.ClaudeCode.OverageStatus == "" &&
		rateLimit.ClaudeCode.OverageDisabledReason == "" &&
		rateLimit.ClaudeCode.IsUsingOverage == nil &&
		len(rateLimit.Raw) == 0 {
		return nil, nil
	}
	if len(rateLimit.Raw) == 0 {
		rateLimit.Raw = nil
	}

	return rateLimit, nil
}

func ParseCodexRateLimit(raw json.RawMessage) (*CLIRateLimit, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var payload struct {
		LimitID     string `json:"limit_id"`
		LimitIDV2   string `json:"limitId"`
		LimitName   string `json:"limit_name"`
		LimitNameV2 string `json:"limitName"`
		Primary     struct {
			UsedPercent        *float64 `json:"used_percent"`
			UsedPercentV2      *float64 `json:"usedPercent"`
			WindowMinutes      int64    `json:"window_minutes"`
			WindowDurationMins int64    `json:"windowDurationMins"`
			ResetsAt           *int64   `json:"resets_at"`
			ResetsAtV2         *int64   `json:"resetsAt"`
		} `json:"primary"`
		Secondary struct {
			UsedPercent        *float64 `json:"used_percent"`
			UsedPercentV2      *float64 `json:"usedPercent"`
			WindowMinutes      int64    `json:"window_minutes"`
			WindowDurationMins int64    `json:"windowDurationMins"`
			ResetsAt           *int64   `json:"resets_at"`
			ResetsAtV2         *int64   `json:"resetsAt"`
		} `json:"secondary"`
		PlanType   string `json:"plan_type"`
		PlanTypeV2 string `json:"planType"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse codex rate limit: %w", err)
	}

	rawMap, err := decodeRateLimitRawMap(raw)
	if err != nil {
		return nil, fmt.Errorf("parse codex rate limit raw payload: %w", err)
	}

	rateLimit := &CLIRateLimit{
		Provider: CLIRateLimitProviderCodex,
		Codex: &CodexRateLimit{
			LimitID:   strings.TrimSpace(firstNonEmptyString(payload.LimitID, payload.LimitIDV2)),
			LimitName: strings.TrimSpace(firstNonEmptyString(payload.LimitName, payload.LimitNameV2)),
			Primary:   newCodexRateLimitWindow(payload.Primary.UsedPercent, payload.Primary.UsedPercentV2, payload.Primary.WindowMinutes, payload.Primary.WindowDurationMins, payload.Primary.ResetsAt, payload.Primary.ResetsAtV2),
			Secondary: newCodexRateLimitWindow(payload.Secondary.UsedPercent, payload.Secondary.UsedPercentV2, payload.Secondary.WindowMinutes, payload.Secondary.WindowDurationMins, payload.Secondary.ResetsAt, payload.Secondary.ResetsAtV2),
			PlanType:  strings.TrimSpace(firstNonEmptyString(payload.PlanType, payload.PlanTypeV2)),
		},
		Raw: rawMap,
	}
	if rateLimit.Codex.LimitID == "" &&
		rateLimit.Codex.LimitName == "" &&
		rateLimit.Codex.Primary == nil &&
		rateLimit.Codex.Secondary == nil &&
		rateLimit.Codex.PlanType == "" &&
		len(rateLimit.Raw) == 0 {
		return nil, nil
	}
	if len(rateLimit.Raw) == 0 {
		rateLimit.Raw = nil
	}

	return rateLimit, nil
}

func newCodexRateLimitWindow(
	usedPercent *float64,
	usedPercentV2 *float64,
	windowMinutes int64,
	windowDurationMins int64,
	resetsAt *int64,
	resetsAtV2 *int64,
) *CodexRateLimitWindow {
	window := &CodexRateLimitWindow{
		UsedPercent:   firstRateLimitFloatPointer(usedPercent, usedPercentV2),
		WindowMinutes: firstNonZeroInt64(windowMinutes, windowDurationMins),
		ResetsAt:      firstUnixSecondsPointerToTime(resetsAt, resetsAtV2),
	}
	if window.UsedPercent == nil && window.WindowMinutes == 0 && window.ResetsAt == nil {
		return nil
	}

	return window
}

func ParseGeminiCLIRateLimit(raw json.RawMessage) (*CLIRateLimit, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var payload struct {
		AuthType  string `json:"authType"`
		Remaining *int64 `json:"remaining"`
		Limit     *int64 `json:"limit"`
		ResetTime string `json:"resetTime"`
		Buckets   []struct {
			ModelID           string   `json:"modelId"`
			TokenType         string   `json:"tokenType"`
			RemainingAmount   string   `json:"remainingAmount"`
			RemainingFraction *float64 `json:"remainingFraction"`
			ResetTime         string   `json:"resetTime"`
		} `json:"buckets"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse gemini rate limit: %w", err)
	}

	rawMap, err := decodeRateLimitRawMap(raw)
	if err != nil {
		return nil, fmt.Errorf("parse gemini rate limit raw payload: %w", err)
	}

	rateLimit := &CLIRateLimit{
		Provider: CLIRateLimitProviderGemini,
		Gemini: &GeminiRateLimit{
			AuthType:  strings.TrimSpace(payload.AuthType),
			Remaining: cloneRateLimitInt64Pointer(payload.Remaining),
			Limit:     cloneRateLimitInt64Pointer(payload.Limit),
			ResetTime: parseRateLimitTime(payload.ResetTime),
		},
		Raw: rawMap,
	}
	if len(payload.Buckets) > 0 {
		rateLimit.Gemini.Buckets = make([]GeminiRateLimitBucket, 0, len(payload.Buckets))
		for _, bucket := range payload.Buckets {
			rateLimit.Gemini.Buckets = append(rateLimit.Gemini.Buckets, GeminiRateLimitBucket{
				ModelID:           strings.TrimSpace(bucket.ModelID),
				TokenType:         strings.TrimSpace(bucket.TokenType),
				RemainingAmount:   strings.TrimSpace(bucket.RemainingAmount),
				RemainingFraction: cloneRateLimitFloatPointer(bucket.RemainingFraction),
				ResetTime:         parseRateLimitTime(bucket.ResetTime),
			})
		}
	}
	if rateLimit.Gemini.AuthType == "" &&
		rateLimit.Gemini.Remaining == nil &&
		rateLimit.Gemini.Limit == nil &&
		rateLimit.Gemini.ResetTime == nil &&
		len(rateLimit.Gemini.Buckets) == 0 &&
		len(rateLimit.Raw) == 0 {
		return nil, nil
	}
	if len(rateLimit.Raw) == 0 {
		rateLimit.Raw = nil
	}

	return rateLimit, nil
}

func decodeRateLimitRawMap(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	rawMap := map[string]any{}
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return nil, err
	}
	if len(rawMap) == 0 {
		return nil, nil
	}

	return rawMap, nil
}

func unixSecondsPointerToTime(value *int64) *time.Time {
	if value == nil || *value <= 0 {
		return nil
	}

	parsed := time.Unix(*value, 0).UTC()
	return &parsed
}

func firstUnixSecondsPointerToTime(values ...*int64) *time.Time {
	for _, value := range values {
		if parsed := unixSecondsPointerToTime(value); parsed != nil {
			return parsed
		}
	}

	return nil
}

func parseRateLimitTime(raw string) *time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}

func cloneRateLimitBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneRateLimitInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneRateLimitFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}

	return 0
}

func firstRateLimitFloatPointer(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return cloneRateLimitFloatPointer(value)
		}
	}

	return nil
}
