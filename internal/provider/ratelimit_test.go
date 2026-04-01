package provider

import (
	"testing"
	"time"
)

func TestParseClaudeCodeRateLimit(t *testing.T) {
	rateLimit, err := ParseClaudeCodeRateLimit([]byte(`{
		"status":"allowed",
		"resetsAt":1775037600,
		"rateLimitType":"five_hour",
		"overageStatus":"rejected",
		"overageDisabledReason":"org_level_disabled",
		"isUsingOverage":false
	}`))
	if err != nil {
		t.Fatalf("ParseClaudeCodeRateLimit returned error: %v", err)
	}
	if rateLimit == nil {
		t.Fatal("ParseClaudeCodeRateLimit returned nil")
	}
	if rateLimit.Provider != CLIRateLimitProviderClaudeCode {
		t.Fatalf("provider = %q", rateLimit.Provider)
	}
	if rateLimit.ClaudeCode == nil {
		t.Fatal("ClaudeCode payload = nil")
	}
	if rateLimit.ClaudeCode.Status != "allowed" || rateLimit.ClaudeCode.RateLimitType != "five_hour" {
		t.Fatalf("ClaudeCode payload = %+v", rateLimit.ClaudeCode)
	}
	if rateLimit.ClaudeCode.ResetsAt == nil || !rateLimit.ClaudeCode.ResetsAt.Equal(time.Unix(1775037600, 0).UTC()) {
		t.Fatalf("resets_at = %+v", rateLimit.ClaudeCode.ResetsAt)
	}
	if rateLimit.ClaudeCode.IsUsingOverage == nil || *rateLimit.ClaudeCode.IsUsingOverage {
		t.Fatalf("is_using_overage = %+v", rateLimit.ClaudeCode.IsUsingOverage)
	}
	if rateLimit.Raw["status"] != "allowed" {
		t.Fatalf("raw payload = %+v", rateLimit.Raw)
	}
}

func TestParseClaudeCodeRateLimitEmptyPayload(t *testing.T) {
	rateLimit, err := ParseClaudeCodeRateLimit(nil)
	if err != nil {
		t.Fatalf("ParseClaudeCodeRateLimit(nil) returned error: %v", err)
	}
	if rateLimit != nil {
		t.Fatalf("ParseClaudeCodeRateLimit(nil) = %+v, want nil", rateLimit)
	}
}
