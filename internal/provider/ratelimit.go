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
)

type CLIRateLimit struct {
	Provider   CLIRateLimitProvider `json:"provider"`
	ClaudeCode *ClaudeCodeRateLimit `json:"claude_code,omitempty"`
	Raw        map[string]any       `json:"raw,omitempty"`
}

type ClaudeCodeRateLimit struct {
	Status                string     `json:"status,omitempty"`
	RateLimitType         string     `json:"rate_limit_type,omitempty"`
	ResetsAt              *time.Time `json:"resets_at,omitempty"`
	OverageStatus         string     `json:"overage_status,omitempty"`
	OverageDisabledReason string     `json:"overage_disabled_reason,omitempty"`
	IsUsingOverage        *bool      `json:"is_using_overage,omitempty"`
}

func ParseClaudeCodeRateLimit(raw json.RawMessage) (*CLIRateLimit, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var payload struct {
		Status                string `json:"status"`
		ResetsAt              *int64 `json:"resetsAt"`
		RateLimitType         string `json:"rateLimitType"`
		OverageStatus         string `json:"overageStatus"`
		OverageDisabledReason string `json:"overageDisabledReason"`
		IsUsingOverage        *bool  `json:"isUsingOverage"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse claude code rate limit: %w", err)
	}

	rawMap := map[string]any{}
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return nil, fmt.Errorf("parse claude code rate limit raw payload: %w", err)
	}

	var resetsAt *time.Time
	if payload.ResetsAt != nil && *payload.ResetsAt > 0 {
		value := time.Unix(*payload.ResetsAt, 0).UTC()
		resetsAt = &value
	}

	rateLimit := &CLIRateLimit{
		Provider: CLIRateLimitProviderClaudeCode,
		ClaudeCode: &ClaudeCodeRateLimit{
			Status:                strings.TrimSpace(payload.Status),
			RateLimitType:         strings.TrimSpace(payload.RateLimitType),
			ResetsAt:              resetsAt,
			OverageStatus:         strings.TrimSpace(payload.OverageStatus),
			OverageDisabledReason: strings.TrimSpace(payload.OverageDisabledReason),
			IsUsingOverage:        cloneRateLimitBoolPointer(payload.IsUsingOverage),
		},
		Raw: rawMap,
	}
	if rateLimit.ClaudeCode.Status == "" &&
		rateLimit.ClaudeCode.RateLimitType == "" &&
		rateLimit.ClaudeCode.ResetsAt == nil &&
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

func cloneRateLimitBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}
