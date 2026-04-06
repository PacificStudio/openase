package chatconversation

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Source string

const (
	SourceProjectSidebar Source = "project_sidebar"
)

type ConversationStatus string

const (
	ConversationStatusActive      ConversationStatus = "active"
	ConversationStatusInterrupted ConversationStatus = "interrupted"
	ConversationStatusClosed      ConversationStatus = "closed"
)

type TurnStatus string

const (
	TurnStatusPending     TurnStatus = "pending"
	TurnStatusRunning     TurnStatus = "running"
	TurnStatusInterrupted TurnStatus = "interrupted"
	TurnStatusCompleted   TurnStatus = "completed"
	TurnStatusFailed      TurnStatus = "failed"
	TurnStatusTerminated  TurnStatus = "terminated"
)

type InterruptStatus string

const (
	InterruptStatusPending  InterruptStatus = "pending"
	InterruptStatusResolved InterruptStatus = "resolved"
)

type EntryKind string

const (
	EntryKindUserMessage         EntryKind = "user_message"
	EntryKindAssistantTextDelta  EntryKind = "assistant_text_delta"
	EntryKindDiff                EntryKind = "diff"
	EntryKindInterrupt           EntryKind = "interrupt"
	EntryKindInterruptResolution EntryKind = "interrupt_resolution"
	EntryKindSystem              EntryKind = "system"
)

type InterruptKind string

const (
	InterruptKindCommandExecutionApproval InterruptKind = "command_execution_approval"
	InterruptKindFileChangeApproval       InterruptKind = "file_change_approval"
	InterruptKindUserInput                InterruptKind = "user_input"
)

type Conversation struct {
	ID                        uuid.UUID
	ProjectID                 uuid.UUID
	UserID                    string
	Source                    Source
	ProviderID                uuid.UUID
	Status                    ConversationStatus
	Title                     ConversationTitle
	ProviderThreadID          *string
	LastTurnID                *string
	ProviderThreadStatus      *string
	ProviderThreadActiveFlags []string
	RollingSummary            string
	LastActivityAt            time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type ConversationAnchors struct {
	ProviderThreadID          *string
	LastTurnID                *string
	ProviderThreadStatus      *string
	ProviderThreadActiveFlags *[]string
	RollingSummary            string
}

type Turn struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	TurnIndex      int
	ProviderTurnID *string
	Status         TurnStatus
	StartedAt      time.Time
	CompletedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Entry struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	TurnID         *uuid.UUID
	Seq            int
	Kind           EntryKind
	Payload        map[string]any
	CreatedAt      time.Time
}

type PendingInterrupt struct {
	ID                uuid.UUID
	ConversationID    uuid.UUID
	TurnID            uuid.UUID
	ProviderRequestID string
	Kind              InterruptKind
	Payload           map[string]any
	Status            InterruptStatus
	Decision          *string
	Response          map[string]any
	ResolvedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CreateConversation struct {
	ProjectID  uuid.UUID
	UserID     string
	Source     Source
	ProviderID uuid.UUID
}

type ListConversationsFilter struct {
	ProjectID  uuid.UUID
	UserID     string
	Source     *Source
	ProviderID *uuid.UUID
}

type InterruptResponse struct {
	Decision *string
	Answer   map[string]any
}

func ParseSource(raw string) (Source, error) {
	switch Source(strings.TrimSpace(raw)) {
	case SourceProjectSidebar:
		return SourceProjectSidebar, nil
	default:
		return "", fmt.Errorf("source must be %q", SourceProjectSidebar)
	}
}
