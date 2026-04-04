package catalog

import (
	"time"

	"github.com/google/uuid"
)

const AgentStepEventType = "agent.step"

const (
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
