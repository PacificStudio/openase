package ticket

import (
	"context"
	"fmt"
	"strings"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func (r *ActivityRepository) RecordActivityEvent(
	ctx context.Context,
	input RecordActivityEventInput,
) (catalogdomain.ActivityEvent, error) {
	if r == nil || r.client == nil {
		return catalogdomain.ActivityEvent{}, errUnavailable
	}
	if input.ProjectID == uuid.Nil {
		return catalogdomain.ActivityEvent{}, fmt.Errorf("activity event project id must not be empty")
	}

	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	builder := r.client.ActivityEvent.Create().
		SetProjectID(input.ProjectID).
		SetEventType(input.EventType.String()).
		SetMessage(strings.TrimSpace(input.Message)).
		SetMetadata(cloneAnyMap(input.Metadata)).
		SetCreatedAt(createdAt.UTC())
	if input.TicketID != nil {
		builder.SetTicketID(*input.TicketID)
	}
	if input.AgentID != nil {
		builder.SetAgentID(*input.AgentID)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return catalogdomain.ActivityEvent{}, fmt.Errorf("record activity event: %w", err)
	}

	return catalogdomain.ActivityEvent{
		ID:        item.ID,
		ProjectID: item.ProjectID,
		TicketID:  item.TicketID,
		AgentID:   item.AgentID,
		EventType: input.EventType,
		Message:   item.Message,
		Metadata:  cloneAnyMap(item.Metadata),
		CreatedAt: item.CreatedAt.UTC(),
	}, nil
}
