package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

type SessionID string

func ParseSessionID(raw string) (SessionID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("chat session id must not be empty")
	}

	return SessionID(trimmed), nil
}

func (s SessionID) String() string {
	return string(s)
}

type RuntimeTurnInput struct {
	SessionID              SessionID
	ProjectID              uuid.UUID
	TicketID               *uuid.UUID
	WorkflowID             *uuid.UUID
	AgentID                *uuid.UUID
	Provider               catalogdomain.AgentProvider
	Message                string
	SystemPrompt           string
	WorkingDirectory       provider.AbsolutePath
	Environment            []string
	ResumeProviderThreadID string
	ResumeProviderTurnID   string
	MaxTurns               int
	MaxBudgetUSD           float64
	PersistentConversation bool
}

type RuntimeSessionAnchor struct {
	ProviderThreadID          string
	LastTurnID                string
	ProviderThreadStatus      string
	ProviderThreadActiveFlags []string
	ProviderAnchorID          string
	ProviderAnchorKind        string
	ProviderTurnSupported     bool
}

type RuntimeInterruptResponseInput struct {
	SessionID              SessionID
	ProjectID              uuid.UUID
	TicketID               *uuid.UUID
	WorkflowID             *uuid.UUID
	AgentID                *uuid.UUID
	Provider               catalogdomain.AgentProvider
	RequestID              string
	Kind                   string
	Decision               string
	Answer                 map[string]any
	Payload                map[string]any
	WorkingDirectory       provider.AbsolutePath
	Environment            []string
	ResumeProviderThreadID string
	ResumeProviderTurnID   string
	PersistentConversation bool
}

func reasoningEffortValue(value *catalogdomain.AgentProviderReasoningEffort) string {
	if value == nil {
		return ""
	}
	return value.String()
}

type runtimeSessionStatePayload struct {
	Status      string         `json:"status"`
	ActiveFlags []string       `json:"active_flags,omitempty"`
	Detail      string         `json:"detail,omitempty"`
	Raw         map[string]any `json:"raw,omitempty"`
}

type runtimeTokenUsagePayload struct {
	TotalInputTokens         int64    `json:"total_input_tokens"`
	TotalOutputTokens        int64    `json:"total_output_tokens"`
	TotalCachedInputTokens   int64    `json:"total_cached_input_tokens"`
	TotalCacheCreationTokens int64    `json:"total_cache_creation_tokens"`
	TotalReasoningTokens     int64    `json:"total_reasoning_tokens"`
	TotalPromptTokens        int64    `json:"total_prompt_tokens"`
	TotalCandidateTokens     int64    `json:"total_candidate_tokens"`
	TotalToolTokens          int64    `json:"total_tool_tokens"`
	TotalTokens              int64    `json:"total_tokens"`
	CostUSD                  *float64 `json:"cost_usd,omitempty"`
	ModelContextWindow       *int64   `json:"model_context_window,omitempty"`
}

type runtimeRateLimitPayload struct {
	RateLimit  *provider.CLIRateLimit `json:"-"`
	ObservedAt time.Time              `json:"observed_at"`
}

func runtimeTokenUsagePayloadFromCLIUsage(usage *provider.CLIUsage, fallbackCostUSD *float64) *runtimeTokenUsagePayload {
	if usage == nil || !usage.HasTokenTotals() {
		if fallbackCostUSD == nil {
			return nil
		}
		return &runtimeTokenUsagePayload{CostUSD: cloneCostUSD(fallbackCostUSD)}
	}

	costUSD := cloneCostUSD(usage.CostUSD)
	if costUSD == nil {
		costUSD = cloneCostUSD(fallbackCostUSD)
	}

	return &runtimeTokenUsagePayload{
		TotalInputTokens:         usage.Total.InputTokens,
		TotalOutputTokens:        usage.Total.OutputTokens,
		TotalCachedInputTokens:   usage.Total.CachedInputTokens,
		TotalCacheCreationTokens: usage.Total.CacheCreationInputTokens,
		TotalReasoningTokens:     usage.Total.ReasoningTokens,
		TotalPromptTokens:        usage.Total.PromptTokens,
		TotalCandidateTokens:     usage.Total.CandidateTokens,
		TotalToolTokens:          usage.Total.ToolTokens,
		TotalTokens:              usage.Total.TotalTokens,
		CostUSD:                  costUSD,
		ModelContextWindow:       cloneInt64Pointer(usage.ModelContextWindow),
	}
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func remainingTurns(maxTurns int, turnsUsed int) *int {
	if maxTurns <= 0 {
		return nil
	}

	remaining := 0
	if maxTurns > turnsUsed {
		remaining = maxTurns - turnsUsed
	}

	return &remaining
}

type Runtime interface {
	Supports(catalogdomain.AgentProvider) bool
	StartTurn(context.Context, RuntimeTurnInput) (TurnStream, error)
	CloseSession(SessionID) bool
}

func NewRuntime(runtimes ...Runtime) Runtime {
	filtered := make([]Runtime, 0, len(runtimes))
	for _, runtime := range runtimes {
		if runtime != nil {
			filtered = append(filtered, runtime)
		}
	}

	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return &compositeRuntime{runtimes: filtered}
	}
}

type compositeRuntime struct {
	runtimes []Runtime
}

func (r *compositeRuntime) Supports(providerItem catalogdomain.AgentProvider) bool {
	if r == nil {
		return false
	}

	for _, runtime := range r.runtimes {
		if runtime.Supports(providerItem) {
			return true
		}
	}

	return false
}

func (r *compositeRuntime) StartTurn(ctx context.Context, input RuntimeTurnInput) (TurnStream, error) {
	if r == nil {
		return TurnStream{}, ErrUnavailable
	}

	for _, runtime := range r.runtimes {
		if !runtime.Supports(input.Provider) {
			continue
		}
		return runtime.StartTurn(ctx, input)
	}

	return TurnStream{}, fmt.Errorf("%w: %s", ErrProviderUnsupported, input.Provider.AdapterType)
}

func (r *compositeRuntime) CloseSession(sessionID SessionID) bool {
	if r == nil {
		return false
	}

	closed := false
	for _, runtime := range r.runtimes {
		if runtime.CloseSession(sessionID) {
			closed = true
		}
	}

	return closed
}
