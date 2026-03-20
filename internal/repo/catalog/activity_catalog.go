package catalog

import (
	"context"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func (r *EntRepository) ListActivityEvents(ctx context.Context, input domain.ListActivityEvents) ([]domain.ActivityEvent, error) {
	exists, err := r.client.Project.Query().
		Where(entproject.ID(input.ProjectID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check project before listing activity events: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	predicates := []predicate.ActivityEvent{
		entactivityevent.ProjectID(input.ProjectID),
	}
	if input.AgentID != nil {
		predicates = append(predicates, entactivityevent.AgentID(*input.AgentID))
	}
	if input.TicketID != nil {
		predicates = append(predicates, entactivityevent.TicketID(*input.TicketID))
	}

	items, err := r.client.ActivityEvent.Query().
		Where(predicates...).
		Order(entactivityevent.ByCreatedAt(entsql.OrderDesc()), entactivityevent.ByID(entsql.OrderDesc())).
		Limit(input.Limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list activity events: %w", err)
	}

	return mapActivityEvents(items), nil
}

func mapActivityEvents(items []*ent.ActivityEvent) []domain.ActivityEvent {
	events := make([]domain.ActivityEvent, 0, len(items))
	for _, item := range items {
		events = append(events, mapActivityEvent(item))
	}

	return events
}

func mapActivityEvent(item *ent.ActivityEvent) domain.ActivityEvent {
	return domain.ActivityEvent{
		ID:        item.ID,
		ProjectID: item.ProjectID,
		TicketID:  item.TicketID,
		AgentID:   item.AgentID,
		EventType: item.EventType,
		Message:   item.Message,
		Metadata:  cloneAnyMap(item.Metadata),
		CreatedAt: cloneActivityCreatedAt(item.CreatedAt),
	}
}

func cloneActivityCreatedAt(value time.Time) time.Time {
	return value.UTC()
}
