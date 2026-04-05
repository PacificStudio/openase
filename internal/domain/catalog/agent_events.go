package catalog

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const AgentStepEventType = "agent.step"

const (
	DefaultAgentRunTranscriptPageLimit = 200
	MaxAgentRunTranscriptPageLimit     = 500
	AgentRunTranscriptKindStep         = "step"
	AgentRunTranscriptKindTrace        = "trace"

	AgentTraceKindAssistantDelta     = "assistant_delta"
	AgentTraceKindAssistantSnapshot  = "assistant_snapshot"
	AgentTraceKindCommandDelta       = "command_output_delta"
	AgentTraceKindCommandSnapshot    = "command_output_snapshot"
	AgentTraceKindToolCallStarted    = "tool_call_started"
	AgentTraceKindApprovalRequested  = "approval_requested"
	AgentTraceKindUserInputRequested = "user_input_requested"
	AgentTraceKindThreadStatus       = "thread_status"
	AgentTraceKindTaskStarted        = "task_started"
	AgentTraceKindTaskProgress       = "task_progress"
	AgentTraceKindTaskNotification   = "task_notification"
	AgentTraceKindSessionState       = "session_state"
	AgentTraceKindError              = "error"
	AgentTraceKindTurnDiffUpdated    = "turn_diff_updated"
	AgentTraceKindReasoningUpdated   = "reasoning_updated"
)

type AgentTraceEntry struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	AgentID    uuid.UUID
	TicketID   *uuid.UUID
	AgentRunID uuid.UUID
	Sequence   int64
	Provider   string
	Kind       string
	Stream     string
	Output     string
	Payload    map[string]any
	CreatedAt  time.Time
}

type AgentStepEntry struct {
	ID                 uuid.UUID
	ProjectID          uuid.UUID
	AgentID            uuid.UUID
	TicketID           *uuid.UUID
	AgentRunID         uuid.UUID
	StepStatus         string
	Summary            string
	SourceTraceEventID *uuid.UUID
	CreatedAt          time.Time
}

func AgentTraceOutputKinds() []string {
	return []string{
		AgentTraceKindAssistantDelta,
		AgentTraceKindAssistantSnapshot,
		AgentTraceKindCommandDelta,
		AgentTraceKindCommandSnapshot,
	}
}

type AgentEventListInput struct {
	TicketID string
	Limit    string
}

type ListAgentSteps struct {
	ProjectID uuid.UUID
	AgentID   uuid.UUID
	TicketID  *uuid.UUID
	Limit     int
}

type ListAgentRunTraceEntries struct {
	ProjectID  uuid.UUID
	AgentRunID uuid.UUID
	Limit      int
}

type ListAgentRunStepEntries struct {
	ProjectID  uuid.UUID
	AgentRunID uuid.UUID
	Limit      int
}

type AgentRunTranscriptPageInput struct {
	Limit  string
	Before string
	After  string
}

type AgentRunTranscriptCursor struct {
	CreatedAt time.Time
	Kind      string
	Order     int64
	ID        uuid.UUID
}

type ListAgentRunTranscriptPage struct {
	ProjectID  uuid.UUID
	AgentRunID uuid.UUID
	Limit      int
	Before     *AgentRunTranscriptCursor
	After      *AgentRunTranscriptCursor
}

type AgentRunTranscriptItem struct {
	Kind       string
	Cursor     string
	TraceEntry *AgentTraceEntry
	StepEntry  *AgentStepEntry
}

type AgentRunTranscriptPage struct {
	Items            []AgentRunTranscriptItem
	HasOlder         bool
	HiddenOlderCount int
	HasNewer         bool
	HiddenNewerCount int
	OldestCursor     string
	NewestCursor     string
}

func ParseListAgentSteps(projectID uuid.UUID, agentID uuid.UUID, raw AgentEventListInput) (ListAgentSteps, error) {
	ticketID, err := parseOptionalUUIDText("ticket_id", raw.TicketID)
	if err != nil {
		return ListAgentSteps{}, err
	}

	limit, err := parseActivityEventLimit(raw.Limit)
	if err != nil {
		return ListAgentSteps{}, err
	}

	return ListAgentSteps{
		ProjectID: projectID,
		AgentID:   agentID,
		TicketID:  ticketID,
		Limit:     limit,
	}, nil
}

func ParseListAgentRunTranscriptPage(
	projectID uuid.UUID,
	agentRunID uuid.UUID,
	raw AgentRunTranscriptPageInput,
) (ListAgentRunTranscriptPage, error) {
	limit, err := parseAgentRunTranscriptPageLimit(raw.Limit)
	if err != nil {
		return ListAgentRunTranscriptPage{}, err
	}
	before, err := parseOptionalAgentRunTranscriptCursor("before", raw.Before)
	if err != nil {
		return ListAgentRunTranscriptPage{}, err
	}
	after, err := parseOptionalAgentRunTranscriptCursor("after", raw.After)
	if err != nil {
		return ListAgentRunTranscriptPage{}, err
	}
	if before != nil && after != nil {
		return ListAgentRunTranscriptPage{}, fmt.Errorf("before and after cannot be combined")
	}

	return ListAgentRunTranscriptPage{
		ProjectID:  projectID,
		AgentRunID: agentRunID,
		Limit:      limit,
		Before:     before,
		After:      after,
	}, nil
}

func AgentRunTranscriptCursorForTrace(entry AgentTraceEntry) AgentRunTranscriptCursor {
	return AgentRunTranscriptCursor{
		CreatedAt: entry.CreatedAt.UTC(),
		Kind:      AgentRunTranscriptKindTrace,
		Order:     entry.Sequence,
		ID:        entry.ID,
	}
}

func AgentRunTranscriptCursorForStep(entry AgentStepEntry) AgentRunTranscriptCursor {
	return AgentRunTranscriptCursor{
		CreatedAt: entry.CreatedAt.UTC(),
		Kind:      AgentRunTranscriptKindStep,
		Order:     0,
		ID:        entry.ID,
	}
}

func ParseAgentRunTranscriptCursor(raw string) (AgentRunTranscriptCursor, error) {
	parts := strings.Split(strings.TrimSpace(raw), "|")
	if len(parts) != 4 {
		return AgentRunTranscriptCursor{}, fmt.Errorf("cursor must be in timestamp|kind|order|id format")
	}

	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return AgentRunTranscriptCursor{}, fmt.Errorf("cursor timestamp must be RFC3339")
	}
	if parts[1] != AgentRunTranscriptKindStep && parts[1] != AgentRunTranscriptKindTrace {
		return AgentRunTranscriptCursor{}, fmt.Errorf("cursor kind must be step or trace")
	}
	order, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return AgentRunTranscriptCursor{}, fmt.Errorf("cursor order must be an integer")
	}
	id, err := uuid.Parse(parts[3])
	if err != nil {
		return AgentRunTranscriptCursor{}, fmt.Errorf("cursor id must be a valid UUID")
	}

	return AgentRunTranscriptCursor{
		CreatedAt: createdAt.UTC(),
		Kind:      parts[1],
		Order:     order,
		ID:        id,
	}, nil
}

func (c AgentRunTranscriptCursor) String() string {
	return fmt.Sprintf("%s|%s|%d|%s", c.CreatedAt.UTC().Format(time.RFC3339Nano), c.Kind, c.Order, c.ID.String())
}

func CompareAgentRunTranscriptCursor(left AgentRunTranscriptCursor, right AgentRunTranscriptCursor) int {
	if left.CreatedAt.Before(right.CreatedAt) {
		return -1
	}
	if left.CreatedAt.After(right.CreatedAt) {
		return 1
	}

	leftRank := agentRunTranscriptKindRank(left.Kind)
	rightRank := agentRunTranscriptKindRank(right.Kind)
	if leftRank != rightRank {
		if leftRank < rightRank {
			return -1
		}
		return 1
	}

	if left.Order != right.Order {
		if left.Order < right.Order {
			return -1
		}
		return 1
	}

	return strings.Compare(left.ID.String(), right.ID.String())
}

func parseAgentRunTranscriptPageLimit(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultAgentRunTranscriptPageLimit, nil
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("limit must be a valid integer")
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("limit must be greater than zero")
	}
	if parsed > MaxAgentRunTranscriptPageLimit {
		return 0, fmt.Errorf("limit must be less than or equal to %d", MaxAgentRunTranscriptPageLimit)
	}

	return parsed, nil
}

func parseOptionalAgentRunTranscriptCursor(fieldName string, raw string) (*AgentRunTranscriptCursor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	cursor, err := ParseAgentRunTranscriptCursor(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s %w", fieldName, err)
	}

	return &cursor, nil
}

func agentRunTranscriptKindRank(kind string) int {
	switch kind {
	case AgentRunTranscriptKindStep:
		return 0
	case AgentRunTranscriptKindTrace:
		return 1
	default:
		return 2
	}
}
