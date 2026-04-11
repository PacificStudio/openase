package catalog

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultAgentRunEventPageLimit = 100
	MaxAgentRunEventPageLimit     = 500

	AgentTranscriptEntryKindAssistantMessage = "assistant_message"
	AgentTranscriptEntryKindCommandStarted   = "command_started"
	AgentTranscriptEntryKindCommandCompleted = "command_completed"
	AgentTranscriptEntryKindCommandFailed    = "command_failed"
	AgentTranscriptEntryKindToolCallStarted  = "tool_call_started"
	AgentTranscriptEntryKindToolCallFinished = "tool_call_finished"
	AgentTranscriptEntryKindApprovalRequest  = "approval_requested"
	AgentTranscriptEntryKindDiff             = "turn_diff"
	AgentTranscriptEntryKindError            = "error"
	AgentTranscriptEntryKindResult           = "result"

	AgentActivityKindAssistantMessage = "assistant_message"
	AgentActivityKindCommandExecution = "command_execution"
	AgentActivityKindToolCall         = "tool_call"
	AgentActivityKindFileChange       = "file_change"
	AgentActivityKindReasoning        = "reasoning"
	AgentActivityKindApproval         = "approval"
	AgentActivityKindTurn             = "turn"

	AgentActivityIDSourceCodexItemID           = "codex_item_id"
	AgentActivityIDSourceClaudeToolUseID       = "claude_tool_use_id"
	AgentActivityIDSourceClaudeTaskID          = "claude_task_id"
	AgentActivityIDSourceClaudeParentToolUseID = "claude_parent_tool_use_id"
	AgentActivityIDSourceEventUUID             = "event_uuid"
	AgentActivityIDSourceSynthetic             = "synthetic"
	AgentActivityIDSourceCodexCallID           = "codex_call_id"

	AgentActivityIdentityConfidenceHigh   = "high"
	AgentActivityIdentityConfidenceMedium = "medium"
	AgentActivityIdentityConfidenceLow    = "low"
)

type AgentRawEventEntry struct {
	ID                   uuid.UUID
	ProjectID            uuid.UUID
	AgentID              uuid.UUID
	TicketID             *uuid.UUID
	AgentRunID           uuid.UUID
	Provider             string
	ProviderEventKind    string
	ProviderEventSubtype string
	ProviderEventID      *string
	ThreadID             *string
	TurnID               *string
	ActivityHintID       *string
	OccurredAt           time.Time
	Payload              map[string]any
	TextExcerpt          string
}

type AgentActivityInstance struct {
	ID                 uuid.UUID
	ProjectID          uuid.UUID
	AgentID            uuid.UUID
	TicketID           *uuid.UUID
	AgentRunID         uuid.UUID
	Provider           string
	ActivityKind       string
	ActivityID         string
	IDSource           string
	IdentityConfidence string
	ParentActivityID   *string
	ThreadID           *string
	TurnID             *string
	Command            *string
	ToolName           *string
	Title              *string
	Status             string
	LiveText           *string
	FinalText          *string
	LiveTextBytes      int
	FinalTextBytes     int
	Metadata           map[string]any
	StartedAt          *time.Time
	UpdatedAt          time.Time
	CompletedAt        *time.Time
}

type AgentTranscriptEntry struct {
	ID           uuid.UUID
	ProjectID    uuid.UUID
	AgentID      uuid.UUID
	TicketID     *uuid.UUID
	AgentRunID   uuid.UUID
	Provider     string
	EntryKey     string
	EntryKind    string
	ActivityKind *string
	ActivityID   *string
	Title        *string
	Summary      *string
	BodyText     *string
	Command      *string
	ToolName     *string
	Metadata     map[string]any
	CreatedAt    time.Time
}

type AgentRunEventPageInput struct {
	Limit  string
	Before string
	After  string
}

type AgentRunEventCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type ListAgentRunRawEvents struct {
	ProjectID  uuid.UUID
	AgentRunID uuid.UUID
	Limit      int
	Before     *AgentRunEventCursor
	After      *AgentRunEventCursor
}

type ListAgentRunActivities struct {
	ProjectID  uuid.UUID
	AgentRunID uuid.UUID
	Status     string
}

type ListAgentRunTranscriptEntries struct {
	ProjectID  uuid.UUID
	AgentRunID uuid.UUID
	Limit      int
	Before     *AgentRunEventCursor
	After      *AgentRunEventCursor
}

type AgentRunRawEventPage struct {
	Entries          []AgentRawEventEntry
	HasOlder         bool
	HiddenOlderCount int
	HasNewer         bool
	HiddenNewerCount int
	OldestCursor     string
	NewestCursor     string
}

type AgentRunTranscriptEntryPage struct {
	Entries          []AgentTranscriptEntry
	HasOlder         bool
	HiddenOlderCount int
	HasNewer         bool
	HiddenNewerCount int
	OldestCursor     string
	NewestCursor     string
}

func ParseListAgentRunRawEvents(
	projectID uuid.UUID,
	agentRunID uuid.UUID,
	raw AgentRunEventPageInput,
) (ListAgentRunRawEvents, error) {
	page, err := parseAgentRunEventPage(raw)
	if err != nil {
		return ListAgentRunRawEvents{}, err
	}
	return ListAgentRunRawEvents{
		ProjectID:  projectID,
		AgentRunID: agentRunID,
		Limit:      page.Limit,
		Before:     page.Before,
		After:      page.After,
	}, nil
}

func ParseListAgentRunTranscriptEntries(
	projectID uuid.UUID,
	agentRunID uuid.UUID,
	raw AgentRunEventPageInput,
) (ListAgentRunTranscriptEntries, error) {
	page, err := parseAgentRunEventPage(raw)
	if err != nil {
		return ListAgentRunTranscriptEntries{}, err
	}
	return ListAgentRunTranscriptEntries{
		ProjectID:  projectID,
		AgentRunID: agentRunID,
		Limit:      page.Limit,
		Before:     page.Before,
		After:      page.After,
	}, nil
}

func ParseListAgentRunActivities(
	projectID uuid.UUID,
	agentRunID uuid.UUID,
	status string,
) (ListAgentRunActivities, error) {
	return ListAgentRunActivities{
		ProjectID:  projectID,
		AgentRunID: agentRunID,
		Status:     strings.TrimSpace(status),
	}, nil
}

func ParseAgentRunEventCursor(raw string) (AgentRunEventCursor, error) {
	parts := strings.Split(strings.TrimSpace(raw), "|")
	if len(parts) != 2 {
		return AgentRunEventCursor{}, fmt.Errorf("cursor must be in timestamp|id format")
	}

	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return AgentRunEventCursor{}, fmt.Errorf("cursor timestamp must be RFC3339")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return AgentRunEventCursor{}, fmt.Errorf("cursor id must be a valid UUID")
	}

	return AgentRunEventCursor{
		CreatedAt: createdAt.UTC(),
		ID:        id,
	}, nil
}

func (c AgentRunEventCursor) String() string {
	return c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID.String()
}

func CompareAgentRunEventCursor(left AgentRunEventCursor, right AgentRunEventCursor) int {
	if left.CreatedAt.Before(right.CreatedAt) {
		return -1
	}
	if left.CreatedAt.After(right.CreatedAt) {
		return 1
	}
	return strings.Compare(left.ID.String(), right.ID.String())
}

func AgentRunEventCursorForRawEvent(entry AgentRawEventEntry) AgentRunEventCursor {
	return AgentRunEventCursor{CreatedAt: entry.OccurredAt.UTC(), ID: entry.ID}
}

func AgentRunEventCursorForTranscriptEntry(entry AgentTranscriptEntry) AgentRunEventCursor {
	return AgentRunEventCursor{CreatedAt: entry.CreatedAt.UTC(), ID: entry.ID}
}

type parsedAgentRunEventPage struct {
	Limit  int
	Before *AgentRunEventCursor
	After  *AgentRunEventCursor
}

func parseAgentRunEventPage(raw AgentRunEventPageInput) (parsedAgentRunEventPage, error) {
	limit, err := parseAgentRunEventPageLimit(raw.Limit)
	if err != nil {
		return parsedAgentRunEventPage{}, err
	}
	before, err := parseOptionalAgentRunEventCursor("before", raw.Before)
	if err != nil {
		return parsedAgentRunEventPage{}, err
	}
	after, err := parseOptionalAgentRunEventCursor("after", raw.After)
	if err != nil {
		return parsedAgentRunEventPage{}, err
	}
	if before != nil && after != nil {
		return parsedAgentRunEventPage{}, fmt.Errorf("before and after cannot be combined")
	}
	return parsedAgentRunEventPage{
		Limit:  limit,
		Before: before,
		After:  after,
	}, nil
}

func parseAgentRunEventPageLimit(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultAgentRunEventPageLimit, nil
	}
	limit, err := strconv.Atoi(trimmed)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("limit must be a positive integer")
	}
	if limit > MaxAgentRunEventPageLimit {
		return 0, fmt.Errorf("limit must be <= %d", MaxAgentRunEventPageLimit)
	}
	return limit, nil
}

func parseOptionalAgentRunEventCursor(fieldName string, raw string) (*AgentRunEventCursor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	cursor, err := ParseAgentRunEventCursor(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s %w", fieldName, err)
	}
	return &cursor, nil
}
