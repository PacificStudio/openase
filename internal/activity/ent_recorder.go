package activity

import (
	"context"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type EntRecorder struct {
	Client *ent.Client
}

func (r EntRecorder) RecordActivityEvent(ctx context.Context, input RecordInput) (catalogdomain.ActivityEvent, error) {
	if r.Client == nil {
		return catalogdomain.ActivityEvent{}, fmt.Errorf("activity ent recorder unavailable")
	}
	if input.ProjectID == uuid.Nil {
		return catalogdomain.ActivityEvent{}, fmt.Errorf("activity event project id must not be empty")
	}

	builder := r.Client.ActivityEvent.Create().
		SetProjectID(input.ProjectID).
		SetEventType(input.EventType.String()).
		SetMessage(strings.TrimSpace(input.Message)).
		SetMetadata(cloneMetadata(input.Metadata)).
		SetCreatedAt(input.CreatedAt.UTC())
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
		Metadata:  cloneMetadata(item.Metadata),
		CreatedAt: item.CreatedAt.UTC(),
	}, nil
}
