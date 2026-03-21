package catalog

import (
	"time"

	"github.com/google/uuid"
)

const AgentOutputEventType = "agent.output"

type AgentOutputEntry struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	AgentID   uuid.UUID
	TicketID  *uuid.UUID
	Stream    string
	Output    string
	CreatedAt time.Time
}

type AgentOutputListInput struct {
	TicketID string
	Limit    string
}

type ListAgentOutput struct {
	ProjectID uuid.UUID
	AgentID   uuid.UUID
	TicketID  *uuid.UUID
	Limit     int
}

func ParseListAgentOutput(projectID uuid.UUID, agentID uuid.UUID, raw AgentOutputListInput) (ListAgentOutput, error) {
	ticketID, err := parseOptionalUUIDText("ticket_id", raw.TicketID)
	if err != nil {
		return ListAgentOutput{}, err
	}

	limit, err := parseActivityEventLimit(raw.Limit)
	if err != nil {
		return ListAgentOutput{}, err
	}

	return ListAgentOutput{
		ProjectID: projectID,
		AgentID:   agentID,
		TicketID:  ticketID,
		Limit:     limit,
	}, nil
}
