package catalog

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	"github.com/google/uuid"
)

const (
	DefaultActivityEventLimit = 40
	MaxActivityEventLimit     = 200
)

type ActivityEvent struct {
	ID                  uuid.UUID
	ProjectID           uuid.UUID
	TicketID            *uuid.UUID
	AgentID             *uuid.UUID
	EventType           activityevent.Type
	UnknownEventTypeRaw string
	Message             string
	Metadata            map[string]any
	CreatedAt           time.Time
}

type ActivityEventListInput struct {
	AgentID  string
	TicketID string
	Limit    string
	Before   string
}

type ActivityEventCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type ActivityEventPage struct {
	Events     []ActivityEvent
	NextCursor string
	HasMore    bool
}

type ListActivityEvents struct {
	ProjectID uuid.UUID
	AgentID   *uuid.UUID
	TicketID  *uuid.UUID
	Limit     int
	Before    *ActivityEventCursor
}

func ParseListActivityEvents(projectID uuid.UUID, raw ActivityEventListInput) (ListActivityEvents, error) {
	agentID, err := parseOptionalUUIDText("agent_id", raw.AgentID)
	if err != nil {
		return ListActivityEvents{}, err
	}
	ticketID, err := parseOptionalUUIDText("ticket_id", raw.TicketID)
	if err != nil {
		return ListActivityEvents{}, err
	}

	limit, err := parseActivityEventLimit(raw.Limit)
	if err != nil {
		return ListActivityEvents{}, err
	}
	before, err := parseOptionalActivityEventCursor("before", raw.Before)
	if err != nil {
		return ListActivityEvents{}, err
	}

	return ListActivityEvents{
		ProjectID: projectID,
		AgentID:   agentID,
		TicketID:  ticketID,
		Limit:     limit,
		Before:    before,
	}, nil
}

func ActivityEventCursorFor(item ActivityEvent) ActivityEventCursor {
	return ActivityEventCursor{
		CreatedAt: item.CreatedAt.UTC(),
		ID:        item.ID,
	}
}

func ParseActivityEventCursor(raw string) (ActivityEventCursor, error) {
	parts := strings.Split(strings.TrimSpace(raw), "|")
	if len(parts) != 2 {
		return ActivityEventCursor{}, fmt.Errorf("cursor must be in timestamp|id format")
	}

	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return ActivityEventCursor{}, fmt.Errorf("cursor timestamp must be RFC3339")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return ActivityEventCursor{}, fmt.Errorf("cursor id must be a valid UUID")
	}

	return ActivityEventCursor{
		CreatedAt: createdAt.UTC(),
		ID:        id,
	}, nil
}

func (c ActivityEventCursor) String() string {
	return fmt.Sprintf("%s|%s", c.CreatedAt.UTC().Format(time.RFC3339Nano), c.ID.String())
}

func CompareActivityEventCursor(left ActivityEventCursor, right ActivityEventCursor) int {
	if left.CreatedAt.Before(right.CreatedAt) {
		return -1
	}
	if left.CreatedAt.After(right.CreatedAt) {
		return 1
	}

	return strings.Compare(left.ID.String(), right.ID.String())
}

func parseOptionalActivityEventCursor(fieldName string, raw string) (*ActivityEventCursor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	cursor, err := ParseActivityEventCursor(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s %s", fieldName, err.Error())
	}

	return &cursor, nil
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
