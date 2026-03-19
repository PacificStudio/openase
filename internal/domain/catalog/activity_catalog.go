package catalog

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultActivityEventLimit = 40
	MaxActivityEventLimit     = 200
)

type ActivityEvent struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	TicketID  *uuid.UUID
	AgentID   *uuid.UUID
	EventType string
	Message   string
	Metadata  map[string]any
	CreatedAt time.Time
}

type ActivityEventListInput struct {
	AgentID string
	Limit   string
}

type ListActivityEvents struct {
	ProjectID uuid.UUID
	AgentID   *uuid.UUID
	Limit     int
}

func ParseListActivityEvents(projectID uuid.UUID, raw ActivityEventListInput) (ListActivityEvents, error) {
	agentID, err := parseOptionalUUIDText("agent_id", raw.AgentID)
	if err != nil {
		return ListActivityEvents{}, err
	}

	limit, err := parseActivityEventLimit(raw.Limit)
	if err != nil {
		return ListActivityEvents{}, err
	}

	return ListActivityEvents{
		ProjectID: projectID,
		AgentID:   agentID,
		Limit:     limit,
	}, nil
}

func parseOptionalUUIDText(fieldName string, raw string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid UUID", fieldName)
	}

	return &parsed, nil
}

func parseActivityEventLimit(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultActivityEventLimit, nil
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("limit must be a valid integer")
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("limit must be greater than zero")
	}
	if parsed > MaxActivityEventLimit {
		return 0, fmt.Errorf("limit must be less than or equal to %d", MaxActivityEventLimit)
	}

	return parsed, nil
}
