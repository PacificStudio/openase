package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

type agentAdapter interface {
	Start(context.Context, agentSessionStartSpec) (agentSession, error)
	Resume(context.Context, agentSessionResumeSpec) (agentSession, error)
}

type agentSession interface {
	SessionID() (string, bool)
	Events() <-chan agentEvent
	SendPrompt(context.Context, string) (agentTurnStartResult, error)
	Stop(context.Context) error
	Err() error
	Diagnostic() agentSessionDiagnostic
}

type agentSessionStartSpec struct {
	Process               provider.AgentCLIProcessSpec
	ProcessManager        provider.AgentCLIProcessManager
	WorkingDirectory      string
	Model                 string
	ReasoningEffort       *catalogdomain.AgentProviderReasoningEffort
	PermissionProfile     catalogdomain.AgentProviderPermissionProfile
	DeveloperInstructions string
	TurnTitle             string
}

type agentSessionResumeSpec struct {
	StartSpec agentSessionStartSpec
	SessionID string
}

type agentTurnStartResult struct {
	TurnID string
}

func reasoningEffortValue(value *catalogdomain.AgentProviderReasoningEffort) string {
	if value == nil {
		return ""
	}
	return value.String()
}

type agentSessionDiagnostic struct {
	PID       int
	SessionID string
	Error     string
	Stderr    string
}

type agentEventType string

const (
	agentEventTypeToolCallRequested  agentEventType = "tool_call_requested"
	agentEventTypeApprovalRequested  agentEventType = "approval_requested"
	agentEventTypeUserInputRequested agentEventType = "user_input_requested"
	agentEventTypeItemStarted        agentEventType = "item_started"
	// #nosec G101 -- runtime event identifier, not a credential.
	agentEventTypeTokenUsageUpdated agentEventType = "token_usage_updated"
	agentEventTypeRateLimitUpdated  agentEventType = "rate_limit_updated"
	agentEventTypeOutputProduced    agentEventType = "output_produced"
	agentEventTypeTaskStatus        agentEventType = "task_status"
	agentEventTypeThreadStatus      agentEventType = "thread_status"
	agentEventTypeTurnDiffUpdated   agentEventType = "turn_diff_updated"
	agentEventTypeReasoningUpdated  agentEventType = "reasoning_updated"
	agentEventTypeTurnStarted       agentEventType = "turn_started"
	agentEventTypeTurnCompleted     agentEventType = "turn_completed"
	agentEventTypeTurnFailed        agentEventType = "turn_failed"
)

type agentEvent struct {
	Type       agentEventType
	ToolCall   *agentToolCallRequest
	Approval   *agentApprovalRequest
	UserInput  *agentUserInputRequest
	Item       *agentItemStartedEvent
	TokenUsage *agentTokenUsageEvent
	RateLimit  *provider.CLIRateLimit
	ObservedAt *time.Time
	Output     *agentOutputEvent
	TaskStatus *agentTaskStatusEvent
	Thread     *agentThreadStatusEvent
	Diff       *agentTurnDiffEvent
	Reasoning  *agentReasoningEvent
	Turn       *agentTurnEvent
	Raw        *agentRawProviderEvent
}

type agentItemStartedEvent struct {
	ThreadID string
	TurnID   string
	ItemID   string
	ItemType string
	Phase    string
	Command  string
	Text     string
}

type agentRawProviderEvent struct {
	DedupKey             string
	ProviderEventKind    string
	ProviderEventSubtype string
	ProviderEventID      string
	ThreadID             string
	TurnID               string
	ActivityHintID       string
	Payload              map[string]any
	TextExcerpt          string
}

type agentTaskStatusEvent struct {
	ThreadID   string
	TurnID     string
	ItemID     string
	StatusType string
	Text       string
	Payload    map[string]any
}

type agentToolCallRequest struct {
	RequestID string
	ThreadID  string
	TurnID    string
	CallID    string
	Tool      string
	Arguments json.RawMessage
}

type agentApprovalRequest struct {
	RequestID string
	ThreadID  string
	TurnID    string
	Kind      string
	Options   []agentApprovalOption
	Payload   map[string]any
}

type agentUserInputRequest struct {
	RequestID string
	ThreadID  string
	TurnID    string
	Payload   map[string]any
}

type agentApprovalOption struct {
	ID          string
	Label       string
	RawDecision string
}

type agentTokenUsageEvent struct {
	ThreadID                      string
	TurnID                        string
	TotalInputTokens              int64
	TotalOutputTokens             int64
	TotalCachedInputTokens        int64
	TotalCacheCreationInputTokens int64
	TotalReasoningTokens          int64
	TotalPromptTokens             int64
	TotalCandidateTokens          int64
	TotalToolTokens               int64
	LastInputTokens               int64
	LastOutputTokens              int64
	LastCachedInputTokens         int64
	LastCacheCreationInputTokens  int64
	LastReasoningTokens           int64
	LastPromptTokens              int64
	LastCandidateTokens           int64
	LastToolTokens                int64
	TotalTokens                   int64
	LastTokens                    int64
	CostUSD                       *float64
	ModelContextWindow            *int64
}

func agentTokenUsageFromCLIUsage(threadID string, turnID string, usage *provider.CLIUsage) *agentTokenUsageEvent {
	if usage == nil || !usage.HasTokenTotals() {
		return nil
	}

	var modelContextWindow *int64
	if usage.ModelContextWindow != nil {
		cloned := *usage.ModelContextWindow
		modelContextWindow = &cloned
	}

	return &agentTokenUsageEvent{
		ThreadID:                      threadID,
		TurnID:                        turnID,
		TotalInputTokens:              usage.Total.InputTokens,
		TotalOutputTokens:             usage.Total.OutputTokens,
		TotalCachedInputTokens:        usage.Total.CachedInputTokens,
		TotalCacheCreationInputTokens: usage.Total.CacheCreationInputTokens,
		TotalReasoningTokens:          usage.Total.ReasoningTokens,
		TotalPromptTokens:             usage.Total.PromptTokens,
		TotalCandidateTokens:          usage.Total.CandidateTokens,
		TotalToolTokens:               usage.Total.ToolTokens,
		LastInputTokens:               usage.Delta.InputTokens,
		LastOutputTokens:              usage.Delta.OutputTokens,
		LastCachedInputTokens:         usage.Delta.CachedInputTokens,
		LastCacheCreationInputTokens:  usage.Delta.CacheCreationInputTokens,
		LastReasoningTokens:           usage.Delta.ReasoningTokens,
		LastPromptTokens:              usage.Delta.PromptTokens,
		LastCandidateTokens:           usage.Delta.CandidateTokens,
		LastToolTokens:                usage.Delta.ToolTokens,
		TotalTokens:                   usage.Total.TotalTokens,
		LastTokens:                    usage.Delta.TotalTokens,
		CostUSD:                       cloneCostUSD(usage.CostUSD),
		ModelContextWindow:            modelContextWindow,
	}
}

func cloneCostUSD(costUSD *float64) *float64 {
	if costUSD == nil {
		return nil
	}

	cloned := *costUSD
	return &cloned
}

func mustMarshalJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return payload
}

type agentOutputEvent struct {
	ThreadID string
	TurnID   string
	ItemID   string
	Stream   string
	Command  string
	Text     string
	Phase    string
	Snapshot bool
}

type agentThreadStatusEvent struct {
	ThreadID    string
	Status      string
	ActiveFlags []string
}

type agentTurnDiffEvent struct {
	ThreadID string
	TurnID   string
	Diff     string
}

type agentReasoningEvent struct {
	ThreadID     string
	TurnID       string
	ItemID       string
	Kind         string
	Delta        string
	SummaryIndex *int
	ContentIndex *int
}

type agentTurnEvent struct {
	ThreadID string
	TurnID   string
	Status   string
	Error    *agentTurnError
}

type agentTurnError struct {
	Message           string
	AdditionalDetails string
}

type agentAdapterRegistry struct {
	adapters map[entagentprovider.AdapterType]agentAdapter
}

func newDefaultAgentAdapterRegistry() *agentAdapterRegistry {
	return &agentAdapterRegistry{
		adapters: map[entagentprovider.AdapterType]agentAdapter{
			entagentprovider.AdapterTypeCodexAppServer: codexAgentAdapter{},
			entagentprovider.AdapterTypeClaudeCodeCli:  claudeCodeAgentAdapter{},
			entagentprovider.AdapterTypeGeminiCli:      geminiAgentAdapter{},
		},
	}
}

func (r *agentAdapterRegistry) adapterFor(adapterType entagentprovider.AdapterType) (agentAdapter, error) {
	if r == nil || len(r.adapters) == 0 {
		return nil, fmt.Errorf("agent adapter registry is empty")
	}
	adapter, ok := r.adapters[adapterType]
	if !ok || adapter == nil {
		return nil, fmt.Errorf("no orchestrator agent adapter registered for %s", adapterType)
	}
	return adapter, nil
}

type unsupportedAgentAdapter struct {
	adapterType entagentprovider.AdapterType
	reason      string
}

func (a unsupportedAgentAdapter) Start(_ context.Context, _ agentSessionStartSpec) (agentSession, error) {
	return nil, a.unsupportedError("start")
}

func (a unsupportedAgentAdapter) Resume(_ context.Context, _ agentSessionResumeSpec) (agentSession, error) {
	return nil, a.unsupportedError("resume")
}

func (a unsupportedAgentAdapter) unsupportedError(operation string) error {
	adapterType := strings.TrimSpace(string(a.adapterType))
	if adapterType == "" {
		adapterType = "unknown"
	}
	reason := strings.TrimSpace(a.reason)
	if reason == "" {
		reason = "this provider is not available in the orchestrator runtime"
	}
	return fmt.Errorf("%s %s adapter: %s", operation, adapterType, reason)
}

func runtimeProviderName(adapterType entagentprovider.AdapterType) string {
	switch adapterType {
	case entagentprovider.AdapterTypeClaudeCodeCli:
		return "claude"
	case entagentprovider.AdapterTypeGeminiCli:
		return "gemini"
	default:
		return "codex"
	}
}
