package chatconversation

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const CostRecordedEventType string = "project_conversation.cost_recorded"

type PrincipalStatus string

const (
	PrincipalStatusActive PrincipalStatus = "active"
	PrincipalStatusClosed PrincipalStatus = "closed"
)

type RuntimeState string

const (
	RuntimeStateInactive    RuntimeState = "inactive"
	RuntimeStateReady       RuntimeState = "ready"
	RuntimeStateExecuting   RuntimeState = "executing"
	RuntimeStateInterrupted RuntimeState = "interrupted"
)

type RunStatus string

const (
	RunStatusLaunching   RunStatus = "launching"
	RunStatusExecuting   RunStatus = "executing"
	RunStatusInterrupted RunStatus = "interrupted"
	RunStatusCompleted   RunStatus = "completed"
	RunStatusFailed      RunStatus = "failed"
	RunStatusTerminated  RunStatus = "terminated"
)

type ProjectConversationPrincipal struct {
	ID                   uuid.UUID
	ConversationID       uuid.UUID
	ProjectID            uuid.UUID
	ProviderID           uuid.UUID
	Name                 string
	Status               PrincipalStatus
	RuntimeState         RuntimeState
	CurrentSessionID     *string
	CurrentWorkspacePath *string
	CurrentRunID         *uuid.UUID
	LastHeartbeatAt      *time.Time
	CurrentStepStatus    *string
	CurrentStepSummary   *string
	CurrentStepChangedAt *time.Time
	ClosedAt             *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ProjectConversationRun struct {
	ID                   uuid.UUID
	PrincipalID          uuid.UUID
	ConversationID       uuid.UUID
	ProjectID            uuid.UUID
	ProviderID           uuid.UUID
	TurnID               *uuid.UUID
	Status               RunStatus
	SessionID            *string
	WorkspacePath        *string
	ProviderThreadID     *string
	ProviderTurnID       *string
	RuntimeStartedAt     *time.Time
	TerminalAt           *time.Time
	LastError            *string
	LastHeartbeatAt      *time.Time
	CostAmount           float64
	InputTokens          int64
	OutputTokens         int64
	CachedInputTokens    int64
	CacheCreationTokens  int64
	ReasoningTokens      int64
	PromptTokens         int64
	CandidateTokens      int64
	ToolTokens           int64
	TotalTokens          int64
	CurrentStepStatus    *string
	CurrentStepSummary   *string
	CurrentStepChangedAt *time.Time
	CreatedAt            time.Time
}

type ProjectConversationTraceEvent struct {
	ID             uuid.UUID
	PrincipalID    uuid.UUID
	RunID          uuid.UUID
	ConversationID uuid.UUID
	ProjectID      uuid.UUID
	Sequence       int64
	Provider       string
	Kind           string
	Stream         string
	Text           *string
	Payload        map[string]any
	CreatedAt      time.Time
}

type ProjectConversationStepEvent struct {
	ID                 uuid.UUID
	PrincipalID        uuid.UUID
	RunID              uuid.UUID
	ConversationID     uuid.UUID
	ProjectID          uuid.UUID
	StepStatus         string
	Summary            *string
	SourceTraceEventID *uuid.UUID
	CreatedAt          time.Time
}

type EnsurePrincipalInput struct {
	ConversationID uuid.UUID
	ProjectID      uuid.UUID
	ProviderID     uuid.UUID
	Name           string
}

type UpdatePrincipalRuntimeInput struct {
	PrincipalID          uuid.UUID
	RuntimeState         RuntimeState
	CurrentSessionID     *string
	CurrentWorkspacePath *string
	CurrentRunID         *uuid.UUID
	LastHeartbeatAt      *time.Time
	CurrentStepStatus    *string
	CurrentStepSummary   *string
	CurrentStepChangedAt *time.Time
}

type ClosePrincipalInput struct {
	PrincipalID uuid.UUID
}

type CreateRunInput struct {
	RunID                uuid.UUID
	PrincipalID          uuid.UUID
	ConversationID       uuid.UUID
	ProjectID            uuid.UUID
	ProviderID           uuid.UUID
	TurnID               *uuid.UUID
	Status               RunStatus
	SessionID            *string
	WorkspacePath        *string
	ProviderThreadID     *string
	ProviderTurnID       *string
	RuntimeStartedAt     *time.Time
	LastHeartbeatAt      *time.Time
	CurrentStepStatus    *string
	CurrentStepSummary   *string
	CurrentStepChangedAt *time.Time
}

type UpdateRunInput struct {
	RunID                uuid.UUID
	Status               *RunStatus
	ProviderThreadID     *string
	ProviderTurnID       *string
	TerminalAt           *time.Time
	LastError            *string
	LastHeartbeatAt      *time.Time
	CostAmount           *float64
	CurrentStepStatus    *string
	CurrentStepSummary   *string
	CurrentStepChangedAt *time.Time
}

type RunUsageSnapshot struct {
	InputTokens         int64
	OutputTokens        int64
	CachedInputTokens   int64
	CacheCreationTokens int64
	ReasoningTokens     int64
	PromptTokens        int64
	CandidateTokens     int64
	ToolTokens          int64
	TotalTokens         int64
	CostAmount          *float64
	ModelContextWindow  *int64
}

type RecordRunUsageInput struct {
	RunID      uuid.UUID
	ProjectID  uuid.UUID
	ProviderID uuid.UUID
	RecordedAt time.Time
	Totals     RunUsageSnapshot
	Delta      RunUsageSnapshot
}

type UpdateProviderRateLimitInput struct {
	ProjectID        uuid.UUID
	ProviderID       uuid.UUID
	ObservedAt       time.Time
	RateLimitPayload map[string]any
}

type AppendTraceEventInput struct {
	RunID          uuid.UUID
	PrincipalID    uuid.UUID
	ConversationID uuid.UUID
	ProjectID      uuid.UUID
	Provider       string
	Kind           string
	Stream         string
	Text           *string
	Payload        map[string]any
}

type AppendStepEventInput struct {
	RunID              uuid.UUID
	PrincipalID        uuid.UUID
	ConversationID     uuid.UUID
	ProjectID          uuid.UUID
	StepStatus         string
	Summary            *string
	SourceTraceEventID *uuid.UUID
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
