package provider

import (
	"encoding/json"
	"testing"
)

func TestParseClaudeCodeUsage(t *testing.T) {
	costUSD := 0.0527345
	usage, err := ParseClaudeCodeUsage(json.RawMessage(`{
		"input_tokens": 3,
		"cache_creation_input_tokens": 7530,
		"cache_read_input_tokens": 11114,
		"output_tokens": 4,
		"service_tier": "standard",
		"inference_geo": "not_available"
	}`), "claude-opus-4-6", &costUSD)
	if err != nil {
		t.Fatalf("ParseClaudeCodeUsage() error = %v", err)
	}
	if usage == nil {
		t.Fatal("ParseClaudeCodeUsage() = nil, want usage")
	}
	if usage.Provider != CLIUsageProviderClaudeCode || usage.Source != CLIUsageSourceClaudeCodeResult {
		t.Fatalf("unexpected usage routing: %+v", usage)
	}
	if usage.Model != "claude-opus-4-6" {
		t.Fatalf("usage.Model = %q, want claude-opus-4-6", usage.Model)
	}
	if usage.CostUSD == nil || *usage.CostUSD != costUSD {
		t.Fatalf("usage.CostUSD = %#v, want %.7f", usage.CostUSD, costUSD)
	}
	if usage.Total.InputTokens != 3 || usage.Total.OutputTokens != 4 || usage.Total.TotalTokens != 7 {
		t.Fatalf("unexpected total token summary: %+v", usage.Total)
	}
	if usage.Total.CachedInputTokens != 11114 || usage.Total.CacheCreationInputTokens != 7530 {
		t.Fatalf("unexpected cache token summary: %+v", usage.Total)
	}
	if usage.Claude == nil || usage.Claude.ServiceTier != "standard" || usage.Claude.InferenceGeo != "not_available" {
		t.Fatalf("unexpected Claude usage details: %+v", usage.Claude)
	}
}

func TestParseGeminiCLIUsage(t *testing.T) {
	usage, err := ParseGeminiCLIUsage(json.RawMessage(`{
		"session_id": "session-1",
		"response": "OK",
		"stats": {
			"models": {
				"gemini-3.1-pro-preview": {
					"api": {
						"totalRequests": 1,
						"totalErrors": 0,
						"totalLatencyMs": 8157
					},
					"tokens": {
						"input": 8286,
						"prompt": 8286,
						"candidates": 1,
						"total": 8355,
						"cached": 0,
						"thoughts": 68,
						"tool": 0
					},
					"roles": {
						"main": {
							"totalRequests": 1,
							"totalErrors": 0,
							"totalLatencyMs": 8157,
							"tokens": {
								"input": 8286,
								"prompt": 8286,
								"candidates": 1,
								"total": 8355,
								"cached": 0,
								"thoughts": 68,
								"tool": 0
							}
						}
					}
				}
			}
		}
	}`))
	if err != nil {
		t.Fatalf("ParseGeminiCLIUsage() error = %v", err)
	}
	if usage == nil {
		t.Fatal("ParseGeminiCLIUsage() = nil, want usage")
	}
	if usage.Provider != CLIUsageProviderGemini || usage.Source != CLIUsageSourceGeminiJSONStats {
		t.Fatalf("unexpected usage routing: %+v", usage)
	}
	if usage.Model != "gemini-3.1-pro-preview" {
		t.Fatalf("usage.Model = %q, want gemini-3.1-pro-preview", usage.Model)
	}
	if usage.Total.InputTokens != 8286 || usage.Total.OutputTokens != 69 || usage.Total.TotalTokens != 8355 {
		t.Fatalf("unexpected total token summary: %+v", usage.Total)
	}
	if usage.Total.ReasoningTokens != 68 || usage.Total.CandidateTokens != 1 || usage.Total.PromptTokens != 8286 {
		t.Fatalf("unexpected Gemini token detail summary: %+v", usage.Total)
	}
	if usage.Gemini == nil || len(usage.Gemini.Models) != 1 {
		t.Fatalf("unexpected Gemini usage details: %+v", usage.Gemini)
	}
	modelUsage, ok := usage.Gemini.Models["gemini-3.1-pro-preview"]
	if !ok {
		t.Fatalf("expected model usage entry, got %+v", usage.Gemini.Models)
	}
	if modelUsage.API.TotalRequests != 1 || modelUsage.API.TotalLatencyMS != 8157 {
		t.Fatalf("unexpected Gemini model API summary: %+v", modelUsage.API)
	}
	if roleUsage, ok := modelUsage.Roles["main"]; !ok || roleUsage.Tokens.OutputTokens != 69 {
		t.Fatalf("unexpected Gemini role usage: %+v", modelUsage.Roles)
	}
}

func TestParseGeminiCLIStreamUsage(t *testing.T) {
	usage, err := ParseGeminiCLIStreamUsage(json.RawMessage(`{
		"type":"result",
		"status":"success",
		"stats":{
			"total_tokens":155,
			"input_tokens":120,
			"output_tokens":35,
			"cached":5,
			"input":115,
			"duration_ms":900,
			"tool_calls":2,
			"models":{
				"gemini-2.5-pro":{
					"total_tokens":155,
					"input_tokens":120,
					"output_tokens":35,
					"cached":5,
					"input":115
				}
			}
		}
	}`))
	if err != nil {
		t.Fatalf("ParseGeminiCLIStreamUsage() error = %v", err)
	}
	if usage == nil {
		t.Fatal("ParseGeminiCLIStreamUsage() = nil, want usage")
	}
	if usage.Provider != CLIUsageProviderGemini || usage.Source != CLIUsageSourceGeminiStreamJSONResult {
		t.Fatalf("unexpected usage routing: %+v", usage)
	}
	if usage.Model != "gemini-2.5-pro" {
		t.Fatalf("usage.Model = %q, want gemini-2.5-pro", usage.Model)
	}
	if usage.Total.InputTokens != 120 || usage.Total.OutputTokens != 35 || usage.Total.TotalTokens != 155 {
		t.Fatalf("unexpected stream token summary: %+v", usage.Total)
	}
	if usage.Total.CachedInputTokens != 5 || usage.Total.PromptTokens != 115 {
		t.Fatalf("unexpected cached/prompt token summary: %+v", usage.Total)
	}
	if usage.Gemini == nil || usage.Gemini.DurationMS != 900 || usage.Gemini.ToolCalls != 2 {
		t.Fatalf("unexpected Gemini stream usage details: %+v", usage.Gemini)
	}
	modelUsage, ok := usage.Gemini.Models["gemini-2.5-pro"]
	if !ok || modelUsage.Tokens.OutputTokens != 35 {
		t.Fatalf("unexpected Gemini stream model usage: %+v", usage.Gemini.Models)
	}
}

func TestNewCodexCLIUsage(t *testing.T) {
	contextWindow := int64(200000)
	usage := NewCodexCLIUsage(
		CLIUsageTokens{
			InputTokens:       120,
			OutputTokens:      35,
			TotalTokens:       155,
			CachedInputTokens: 12,
			ReasoningTokens:   5,
		},
		CLIUsageTokens{
			InputTokens:  20,
			OutputTokens: 7,
			TotalTokens:  27,
		},
		&contextWindow,
	)
	if usage == nil {
		t.Fatal("NewCodexCLIUsage() = nil, want usage")
	}
	if usage.Provider != CLIUsageProviderCodex || usage.Source != CLIUsageSourceCodexAppServerThreadUpdate {
		t.Fatalf("unexpected usage routing: %+v", usage)
	}
	if usage.ModelContextWindow == nil || *usage.ModelContextWindow != contextWindow {
		t.Fatalf("usage.ModelContextWindow = %#v, want %d", usage.ModelContextWindow, contextWindow)
	}
	if usage.Total.CachedInputTokens != 12 || usage.Total.ReasoningTokens != 5 || usage.Delta.TotalTokens != 27 {
		t.Fatalf("unexpected codex usage summary: %+v", usage)
	}
}
