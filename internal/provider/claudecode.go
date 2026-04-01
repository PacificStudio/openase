package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ClaudeCodeSessionID identifies a Claude Code session.
type ClaudeCodeSessionID string

// ParseClaudeCodeSessionID validates a raw Claude Code session identifier.
func ParseClaudeCodeSessionID(raw string) (ClaudeCodeSessionID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("claude code session id must not be empty")
	}

	return ClaudeCodeSessionID(trimmed), nil
}

// MustParseClaudeCodeSessionID parses a session ID and panics on invalid input.
func MustParseClaudeCodeSessionID(raw string) ClaudeCodeSessionID {
	sessionID, err := ParseClaudeCodeSessionID(raw)
	if err != nil {
		panic(err)
	}

	return sessionID
}

func (s ClaudeCodeSessionID) String() string {
	return string(s)
}

// ClaudeCodeEventKind classifies one streamed Claude Code event frame.
type ClaudeCodeEventKind string

const (
	// Claude Code event kinds mirror the upstream streaming protocol.
	ClaudeCodeEventKindUnknown      ClaudeCodeEventKind = "unknown"
	ClaudeCodeEventKindSystem       ClaudeCodeEventKind = "system"
	ClaudeCodeEventKindAssistant    ClaudeCodeEventKind = "assistant"
	ClaudeCodeEventKindUser         ClaudeCodeEventKind = "user"
	ClaudeCodeEventKindResult       ClaudeCodeEventKind = "result"
	ClaudeCodeEventKindRateLimit    ClaudeCodeEventKind = "rate_limit_event"
	ClaudeCodeEventKindStream       ClaudeCodeEventKind = "stream_event"
	ClaudeCodeEventKindTaskStart    ClaudeCodeEventKind = "task_started"
	ClaudeCodeEventKindTaskProgress ClaudeCodeEventKind = "task_progress"
	ClaudeCodeEventKindTaskNotice   ClaudeCodeEventKind = "task_notification"
)

// ClaudeCodeEvent is the normalized event envelope emitted by the adapter.
type ClaudeCodeEvent struct {
	Kind            ClaudeCodeEventKind
	Raw             json.RawMessage
	UnknownType     string
	Subtype         string
	SessionID       string
	ParentToolUseID string
	Message         json.RawMessage
	Data            json.RawMessage
	Result          string
	Model           string
	Usage           json.RawMessage
	UsageInfo       *CLIUsage
	RateLimit       json.RawMessage
	RateLimitInfo   *CLIRateLimit
	Event           json.RawMessage
	UUID            string
	IsError         bool
	NumTurns        int
	DurationMS      int
	DurationAPIMS   int
	TotalCostUSD    *float64
}

// ClaudeCodeTurnInput is a single prompt turn sent to Claude Code.
type ClaudeCodeTurnInput struct {
	Prompt string
}

// NewClaudeCodeTurnInput validates and constructs a Claude Code turn input.
func NewClaudeCodeTurnInput(prompt string) (ClaudeCodeTurnInput, error) {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return ClaudeCodeTurnInput{}, fmt.Errorf("claude code turn prompt must not be empty")
	}

	return ClaudeCodeTurnInput{Prompt: trimmed}, nil
}

// ClaudeCodeSessionSpec defines how to start or resume a Claude Code session.
type ClaudeCodeSessionSpec struct {
	Command                AgentCLICommand
	BaseArgs               []string
	WorkingDirectory       *AbsolutePath
	Environment            []string
	AllowedTools           []string
	AppendSystemPrompt     string
	MaxTurns               *int
	MaxBudgetUSD           *float64
	ResumeSessionID        *ClaudeCodeSessionID
	IncludePartialMessages bool
}

// NewClaudeCodeSessionSpec validates and constructs a Claude Code session spec.
func NewClaudeCodeSessionSpec(
	command AgentCLICommand,
	baseArgs []string,
	workingDirectory *AbsolutePath,
	environment []string,
	allowedTools []string,
	appendSystemPrompt string,
	maxTurns *int,
	maxBudgetUSD *float64,
	resumeSessionID *ClaudeCodeSessionID,
	includePartialMessages bool,
) (ClaudeCodeSessionSpec, error) {
	if command == "" {
		return ClaudeCodeSessionSpec{}, fmt.Errorf("claude code command must not be empty")
	}
	if workingDirectory != nil && *workingDirectory == "" {
		return ClaudeCodeSessionSpec{}, fmt.Errorf("working directory must not be empty when provided")
	}

	normalizedEnvironment := make([]string, 0, len(environment))
	for _, entry := range environment {
		if err := validateProcessEnvironmentEntry(entry); err != nil {
			return ClaudeCodeSessionSpec{}, err
		}
		normalizedEnvironment = append(normalizedEnvironment, entry)
	}

	normalizedAllowedTools := make([]string, 0, len(allowedTools))
	for _, tool := range allowedTools {
		trimmed := strings.TrimSpace(tool)
		if trimmed == "" {
			return ClaudeCodeSessionSpec{}, fmt.Errorf("allowed tools must not contain empty entries")
		}
		normalizedAllowedTools = append(normalizedAllowedTools, trimmed)
	}

	if maxTurns != nil && *maxTurns <= 0 {
		return ClaudeCodeSessionSpec{}, fmt.Errorf("max turns must be positive when provided")
	}
	if maxBudgetUSD != nil && *maxBudgetUSD <= 0 {
		return ClaudeCodeSessionSpec{}, fmt.Errorf("max budget usd must be positive when provided")
	}
	if resumeSessionID != nil && *resumeSessionID == "" {
		return ClaudeCodeSessionSpec{}, fmt.Errorf("resume session id must not be empty when provided")
	}

	return ClaudeCodeSessionSpec{
		Command:                command,
		BaseArgs:               append([]string(nil), baseArgs...),
		WorkingDirectory:       workingDirectory,
		Environment:            normalizedEnvironment,
		AllowedTools:           normalizedAllowedTools,
		AppendSystemPrompt:     strings.TrimSpace(appendSystemPrompt),
		MaxTurns:               cloneIntPointer(maxTurns),
		MaxBudgetUSD:           cloneFloatPointer(maxBudgetUSD),
		ResumeSessionID:        cloneClaudeCodeSessionIDPointer(resumeSessionID),
		IncludePartialMessages: includePartialMessages,
	}, nil
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func cloneClaudeCodeSessionIDPointer(value *ClaudeCodeSessionID) *ClaudeCodeSessionID {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

// ClaudeCodeSession represents a live Claude Code session.
type ClaudeCodeSession interface {
	SessionID() (ClaudeCodeSessionID, bool)
	Events() <-chan ClaudeCodeEvent
	Errors() <-chan error
	Send(context.Context, ClaudeCodeTurnInput) error
	Close(context.Context) error
}

// ClaudeCodeAdapter starts Claude Code sessions behind a provider abstraction.
type ClaudeCodeAdapter interface {
	Start(context.Context, ClaudeCodeSessionSpec) (ClaudeCodeSession, error)
}
