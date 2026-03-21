package catalog

import (
	"context"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
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

func (r *EntRepository) ListAgentOutput(ctx context.Context, input domain.ListAgentOutput) ([]domain.AgentOutputEntry, error) {
	exists, err := r.client.Agent.Query().
		Where(entagent.ID(input.AgentID), entagent.ProjectID(input.ProjectID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check agent before listing output: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	predicates := []predicate.ActivityEvent{
		entactivityevent.ProjectID(input.ProjectID),
		entactivityevent.AgentID(input.AgentID),
		entactivityevent.EventType(domain.AgentOutputEventType),
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
		return nil, fmt.Errorf("list agent output: %w", err)
	}

	return mapAgentOutputEntries(items), nil
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

func mapAgentOutputEntries(items []*ent.ActivityEvent) []domain.AgentOutputEntry {
	entries := make([]domain.AgentOutputEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, mapAgentOutputEntry(item))
	}

	return entries
}

func mapAgentOutputEntry(item *ent.ActivityEvent) domain.AgentOutputEntry {
	agentID := uuid.Nil
	if item.AgentID != nil {
		agentID = *item.AgentID
	}

	return domain.AgentOutputEntry{
		ID:        item.ID,
		ProjectID: item.ProjectID,
		AgentID:   agentID,
		TicketID:  item.TicketID,
		Stream:    domain.AgentOutputMetadataStream(item.Metadata),
		Output:    item.Message,
		CreatedAt: cloneActivityCreatedAt(item.CreatedAt),
	}
}

func cloneActivityCreatedAt(value time.Time) time.Time {
	return value.UTC()
}
