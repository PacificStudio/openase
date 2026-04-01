package orchestrator

import (
	"context"
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
	// #nosec G101 -- runtime event identifier, not a credential.
	agentEventTypeTokenUsageUpdated agentEventType = "token_usage_updated"
	agentEventTypeRateLimitUpdated  agentEventType = "rate_limit_updated"
	agentEventTypeOutputProduced    agentEventType = "output_produced"
	agentEventTypeTurnStarted       agentEventType = "turn_started"
	agentEventTypeTurnCompleted     agentEventType = "turn_completed"
	agentEventTypeTurnFailed        agentEventType = "turn_failed"
)

type agentEvent struct {
	Type       agentEventType
	ToolCall   *agentToolCallRequest
	Approval   *agentApprovalRequest
	UserInput  *agentUserInputRequest
	TokenUsage *agentTokenUsageEvent
	RateLimit  *provider.CLIRateLimit
	ObservedAt *time.Time
	Output     *agentOutputEvent
	Turn       *agentTurnEvent
}

type agentToolCallRequest struct {
	RequestID string
	ThreadID  string
	TurnID    string
	CallID    string
	Tool      string
}

type agentApprovalRequest struct {
	RequestID string
	Kind      string
}

type agentUserInputRequest struct {
	RequestID string
}

type agentTokenUsageEvent struct {
	ThreadID           string
	TurnID             string
	TotalInputTokens   int64
	TotalOutputTokens  int64
	LastInputTokens    int64
	LastOutputTokens   int64
	TotalTokens        int64
	LastTokens         int64
	CostUSD            *float64
	ModelContextWindow *int64
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
		ThreadID:           threadID,
		TurnID:             turnID,
		TotalInputTokens:   usage.Total.InputTokens,
		TotalOutputTokens:  usage.Total.OutputTokens,
		LastInputTokens:    usage.Delta.InputTokens,
		LastOutputTokens:   usage.Delta.OutputTokens,
		TotalTokens:        usage.Total.TotalTokens,
		LastTokens:         usage.Delta.TotalTokens,
		CostUSD:            cloneCostUSD(usage.CostUSD),
		ModelContextWindow: modelContextWindow,
	}
}

func cloneCostUSD(costUSD *float64) *float64 {
	if costUSD == nil {
		return nil
	}

	cloned := *costUSD
	return &cloned
}

type agentOutputEvent struct {
	ThreadID string
	TurnID   string
	ItemID   string
	Stream   string
	Text     string
	Phase    string
	Snapshot bool
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
