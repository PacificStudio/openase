package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

type CLIUsageProvider string

const (
	CLIUsageProviderCodex      CLIUsageProvider = "codex"
	CLIUsageProviderClaudeCode CLIUsageProvider = "claude_code"
	CLIUsageProviderGemini     CLIUsageProvider = "gemini"
)

type CLIUsageSource string

const (
	CLIUsageSourceCodexAppServerThreadUpdate CLIUsageSource = "codex_app_server_thread_update"
	CLIUsageSourceClaudeCodeResult           CLIUsageSource = "claude_code_result"
	CLIUsageSourceGeminiJSONStats            CLIUsageSource = "gemini_json_stats"
	CLIUsageSourceGeminiStreamJSONResult     CLIUsageSource = "gemini_stream_json_result"
)

type CLIUsage struct {
	Provider           CLIUsageProvider `json:"provider"`
	Source             CLIUsageSource   `json:"source"`
	Model              string           `json:"model,omitempty"`
	CostUSD            *float64         `json:"cost_usd,omitempty"`
	ModelContextWindow *int64           `json:"model_context_window,omitempty"`
	Total              CLIUsageTokens   `json:"total"`
	Delta              CLIUsageTokens   `json:"delta,omitempty"`
	Raw                json.RawMessage  `json:"raw,omitempty"`
	Claude             *ClaudeCodeUsage `json:"claude,omitempty"`
	Gemini             *GeminiCLIUsage  `json:"gemini,omitempty"`
}

func (u *CLIUsage) HasTokenTotals() bool {
	return u != nil && (!u.Total.isZero() || !u.Delta.isZero())
}

type CLIUsageTokens struct {
	InputTokens              int64 `json:"input_tokens,omitempty"`
	OutputTokens             int64 `json:"output_tokens,omitempty"`
	TotalTokens              int64 `json:"total_tokens,omitempty"`
	CachedInputTokens        int64 `json:"cached_input_tokens,omitempty"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,omitempty"`
	ReasoningTokens          int64 `json:"reasoning_tokens,omitempty"`
	PromptTokens             int64 `json:"prompt_tokens,omitempty"`
	CandidateTokens          int64 `json:"candidate_tokens,omitempty"`
	ToolTokens               int64 `json:"tool_tokens,omitempty"`
}

func (t CLIUsageTokens) isZero() bool {
	return t.InputTokens == 0 &&
		t.OutputTokens == 0 &&
		t.TotalTokens == 0 &&
		t.CachedInputTokens == 0 &&
		t.CacheCreationInputTokens == 0 &&
		t.ReasoningTokens == 0 &&
		t.PromptTokens == 0 &&
		t.CandidateTokens == 0 &&
		t.ToolTokens == 0
}

func (t CLIUsageTokens) add(other CLIUsageTokens) CLIUsageTokens {
	return CLIUsageTokens{
		InputTokens:              t.InputTokens + other.InputTokens,
		OutputTokens:             t.OutputTokens + other.OutputTokens,
		TotalTokens:              t.TotalTokens + other.TotalTokens,
		CachedInputTokens:        t.CachedInputTokens + other.CachedInputTokens,
		CacheCreationInputTokens: t.CacheCreationInputTokens + other.CacheCreationInputTokens,
		ReasoningTokens:          t.ReasoningTokens + other.ReasoningTokens,
		PromptTokens:             t.PromptTokens + other.PromptTokens,
		CandidateTokens:          t.CandidateTokens + other.CandidateTokens,
		ToolTokens:               t.ToolTokens + other.ToolTokens,
	}
}

type ClaudeCodeUsage struct {
	ServiceTier  string `json:"service_tier,omitempty"`
	InferenceGeo string `json:"inference_geo,omitempty"`
}

type GeminiCLIUsage struct {
	DurationMS int64                          `json:"duration_ms,omitempty"`
	ToolCalls  int64                          `json:"tool_calls,omitempty"`
	Models     map[string]GeminiCLIModelUsage `json:"models,omitempty"`
}

type GeminiCLIModelUsage struct {
	API    GeminiCLIAPIUsage             `json:"api"`
	Tokens CLIUsageTokens                `json:"tokens"`
	Roles  map[string]GeminiCLIRoleUsage `json:"roles,omitempty"`
}

type GeminiCLIRoleUsage struct {
	API    GeminiCLIAPIUsage `json:"api"`
	Tokens CLIUsageTokens    `json:"tokens"`
}

type GeminiCLIAPIUsage struct {
	TotalRequests  int64 `json:"total_requests,omitempty"`
	TotalErrors    int64 `json:"total_errors,omitempty"`
	TotalLatencyMS int64 `json:"total_latency_ms,omitempty"`
}

type rawGeminiAPIUsage struct {
	TotalRequests  int64 `json:"totalRequests"`
	TotalErrors    int64 `json:"totalErrors"`
	TotalLatencyMS int64 `json:"totalLatencyMs"`
}

type rawGeminiTokens struct {
	Input      int64 `json:"input"`
	Prompt     int64 `json:"prompt"`
	Candidates int64 `json:"candidates"`
	Total      int64 `json:"total"`
	Cached     int64 `json:"cached"`
	Thoughts   int64 `json:"thoughts"`
	Tool       int64 `json:"tool"`
}

type rawGeminiRoleUsage struct {
	API    rawGeminiAPIUsage `json:"api"`
	Tokens rawGeminiTokens   `json:"tokens"`
}

type rawGeminiModelUsage struct {
	API    rawGeminiAPIUsage             `json:"api"`
	Tokens rawGeminiTokens               `json:"tokens"`
	Roles  map[string]rawGeminiRoleUsage `json:"roles"`
}

type rawGeminiStreamModelUsage struct {
	TotalTokens  int64 `json:"total_tokens"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	Cached       int64 `json:"cached"`
	Input        int64 `json:"input"`
}

type rawGeminiStreamStats struct {
	TotalTokens  int64                                `json:"total_tokens"`
	InputTokens  int64                                `json:"input_tokens"`
	OutputTokens int64                                `json:"output_tokens"`
	Cached       int64                                `json:"cached"`
	Input        int64                                `json:"input"`
	DurationMS   int64                                `json:"duration_ms"`
	ToolCalls    int64                                `json:"tool_calls"`
	Models       map[string]rawGeminiStreamModelUsage `json:"models"`
}

func NewCodexCLIUsage(total CLIUsageTokens, delta CLIUsageTokens, modelContextWindow *int64) *CLIUsage {
	if total.isZero() && delta.isZero() && modelContextWindow == nil {
		return nil
	}

	return &CLIUsage{
		Provider:           CLIUsageProviderCodex,
		Source:             CLIUsageSourceCodexAppServerThreadUpdate,
		Total:              total,
		Delta:              delta,
		ModelContextWindow: cloneUsageInt64Pointer(modelContextWindow),
	}
}

func ParseClaudeCodeUsage(raw json.RawMessage, model string, totalCostUSD *float64) (*CLIUsage, error) {
	if len(raw) == 0 && totalCostUSD == nil {
		return nil, nil
	}

	var payload struct {
		InputTokens              int64  `json:"input_tokens"`
		OutputTokens             int64  `json:"output_tokens"`
		TotalTokens              int64  `json:"total_tokens"`
		CachedInputTokens        int64  `json:"cached_input_tokens"`
		CacheReadInputTokens     int64  `json:"cache_read_input_tokens"`
		CacheCreationInputTokens int64  `json:"cache_creation_input_tokens"`
		ServiceTier              string `json:"service_tier"`
		InferenceGeo             string `json:"inference_geo"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("parse claude code usage: %w", err)
		}
	}

	cachedInputTokens := payload.CachedInputTokens
	if payload.CacheReadInputTokens > 0 {
		cachedInputTokens = payload.CacheReadInputTokens
	}
	totalTokens := payload.TotalTokens
	if totalTokens == 0 && (payload.InputTokens > 0 || payload.OutputTokens > 0) {
		totalTokens = payload.InputTokens + payload.OutputTokens
	}

	usage := &CLIUsage{
		Provider: CLIUsageProviderClaudeCode,
		Source:   CLIUsageSourceClaudeCodeResult,
		Model:    strings.TrimSpace(model),
		CostUSD:  cloneUsageFloatPointer(totalCostUSD),
		Raw:      cloneUsageRawJSON(raw),
		Total: CLIUsageTokens{
			InputTokens:              payload.InputTokens,
			OutputTokens:             payload.OutputTokens,
			TotalTokens:              totalTokens,
			CachedInputTokens:        cachedInputTokens,
			CacheCreationInputTokens: payload.CacheCreationInputTokens,
		},
		Claude: &ClaudeCodeUsage{
			ServiceTier:  strings.TrimSpace(payload.ServiceTier),
			InferenceGeo: strings.TrimSpace(payload.InferenceGeo),
		},
	}
	if usage.Total.isZero() && usage.CostUSD == nil {
		return nil, nil
	}
	if usage.Claude.ServiceTier == "" && usage.Claude.InferenceGeo == "" {
		usage.Claude = nil
	}

	return usage, nil
}

func ParseGeminiCLIUsage(raw json.RawMessage) (*CLIUsage, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var payload struct {
		Stats struct {
			Models map[string]rawGeminiModelUsage `json:"models"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse gemini usage: %w", err)
	}
	if len(payload.Stats.Models) == 0 {
		return nil, nil
	}

	usage := &CLIUsage{
		Provider: CLIUsageProviderGemini,
		Source:   CLIUsageSourceGeminiJSONStats,
		Raw:      cloneUsageRawJSON(raw),
		Gemini: &GeminiCLIUsage{
			Models: make(map[string]GeminiCLIModelUsage, len(payload.Stats.Models)),
		},
	}

	if len(payload.Stats.Models) == 1 {
		for modelName := range payload.Stats.Models {
			usage.Model = strings.TrimSpace(modelName)
		}
	}

	for modelName, modelUsage := range payload.Stats.Models {
		tokens := geminiUsageTokens(modelUsage.Tokens)
		usage.Total = usage.Total.add(tokens)

		detail := GeminiCLIModelUsage{
			API: GeminiCLIAPIUsage{
				TotalRequests:  modelUsage.API.TotalRequests,
				TotalErrors:    modelUsage.API.TotalErrors,
				TotalLatencyMS: modelUsage.API.TotalLatencyMS,
			},
			Tokens: tokens,
		}
		if len(modelUsage.Roles) > 0 {
			detail.Roles = make(map[string]GeminiCLIRoleUsage, len(modelUsage.Roles))
			for roleName, roleUsage := range modelUsage.Roles {
				detail.Roles[roleName] = GeminiCLIRoleUsage{
					API: GeminiCLIAPIUsage{
						TotalRequests:  roleUsage.API.TotalRequests,
						TotalErrors:    roleUsage.API.TotalErrors,
						TotalLatencyMS: roleUsage.API.TotalLatencyMS,
					},
					Tokens: geminiUsageTokens(roleUsage.Tokens),
				}
			}
		}
		usage.Gemini.Models[strings.TrimSpace(modelName)] = detail
	}

	return usage, nil
}

func ParseGeminiCLIStreamUsage(raw json.RawMessage) (*CLIUsage, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var payload struct {
		Stats rawGeminiStreamStats `json:"stats"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse gemini stream usage: %w", err)
	}
	if len(payload.Stats.Models) == 0 && payload.Stats.TotalTokens == 0 && payload.Stats.InputTokens == 0 && payload.Stats.OutputTokens == 0 {
		return nil, nil
	}

	usage := &CLIUsage{
		Provider: CLIUsageProviderGemini,
		Source:   CLIUsageSourceGeminiStreamJSONResult,
		Raw:      cloneUsageRawJSON(raw),
		Total: CLIUsageTokens{
			InputTokens:       payload.Stats.InputTokens,
			OutputTokens:      payload.Stats.OutputTokens,
			TotalTokens:       payload.Stats.TotalTokens,
			CachedInputTokens: payload.Stats.Cached,
			PromptTokens:      payload.Stats.Input,
		},
		Gemini: &GeminiCLIUsage{
			DurationMS: payload.Stats.DurationMS,
			ToolCalls:  payload.Stats.ToolCalls,
			Models:     make(map[string]GeminiCLIModelUsage, len(payload.Stats.Models)),
		},
	}
	if usage.Total.TotalTokens == 0 && (usage.Total.InputTokens > 0 || usage.Total.OutputTokens > 0) {
		usage.Total.TotalTokens = usage.Total.InputTokens + usage.Total.OutputTokens
	}
	if len(payload.Stats.Models) == 1 {
		for modelName := range payload.Stats.Models {
			usage.Model = strings.TrimSpace(modelName)
		}
	}
	for modelName, modelUsage := range payload.Stats.Models {
		tokens := CLIUsageTokens{
			InputTokens:       modelUsage.InputTokens,
			OutputTokens:      modelUsage.OutputTokens,
			TotalTokens:       modelUsage.TotalTokens,
			CachedInputTokens: modelUsage.Cached,
			PromptTokens:      modelUsage.Input,
		}
		if tokens.TotalTokens == 0 && (tokens.InputTokens > 0 || tokens.OutputTokens > 0) {
			tokens.TotalTokens = tokens.InputTokens + tokens.OutputTokens
		}
		usage.Gemini.Models[strings.TrimSpace(modelName)] = GeminiCLIModelUsage{
			Tokens: tokens,
		}
	}

	return usage, nil
}

func geminiUsageTokens(raw rawGeminiTokens) CLIUsageTokens {
	outputTokens := raw.Total - raw.Input
	if outputTokens < 0 {
		outputTokens = 0
	}
	if outputTokens == 0 && (raw.Candidates > 0 || raw.Thoughts > 0 || raw.Tool > 0) {
		outputTokens = raw.Candidates + raw.Thoughts + raw.Tool
	}
	totalTokens := raw.Total
	if totalTokens == 0 && (raw.Input > 0 || outputTokens > 0) {
		totalTokens = raw.Input + outputTokens
	}

	return CLIUsageTokens{
		InputTokens:       raw.Input,
		OutputTokens:      outputTokens,
		TotalTokens:       totalTokens,
		CachedInputTokens: raw.Cached,
		ReasoningTokens:   raw.Thoughts,
		PromptTokens:      raw.Prompt,
		CandidateTokens:   raw.Candidates,
		ToolTokens:        raw.Tool,
	}
}

func cloneUsageFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneUsageInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneUsageRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}

	return append(json.RawMessage(nil), raw...)
}
