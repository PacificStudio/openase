package catalog

import (
	"context"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entagentstepevent "github.com/BetterAndBetterII/openase/ent/agentstepevent"
	entagenttraceevent "github.com/BetterAndBetterII/openase/ent/agenttraceevent"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatconversationdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
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
		entactivityevent.EventTypeNEQ(domain.AgentOutputEventType),
		entactivityevent.EventTypeNEQ(ticketing.CostRecordedEventType),
		entactivityevent.EventTypeNEQ(chatconversationdomain.CostRecordedEventType),
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

	predicates := []predicate.AgentTraceEvent{
		entagenttraceevent.ProjectID(input.ProjectID),
		entagenttraceevent.AgentID(input.AgentID),
		entagenttraceevent.KindIn(domain.AgentTraceOutputKinds()...),
	}
	if input.TicketID != nil {
		predicates = append(predicates, entagenttraceevent.TicketID(*input.TicketID))
	}

	items, err := r.client.AgentTraceEvent.Query().
		Where(predicates...).
		Order(entagenttraceevent.ByCreatedAt(entsql.OrderDesc()), entagenttraceevent.ByID(entsql.OrderDesc())).
		Limit(input.Limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agent output: %w", err)
	}

	return mapAgentOutputEntries(items), nil
}

func (r *EntRepository) ListAgentSteps(ctx context.Context, input domain.ListAgentSteps) ([]domain.AgentStepEntry, error) {
	exists, err := r.client.Agent.Query().
		Where(entagent.ID(input.AgentID), entagent.ProjectID(input.ProjectID)).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check agent before listing steps: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	predicates := []predicate.AgentStepEvent{
		entagentstepevent.ProjectID(input.ProjectID),
		entagentstepevent.AgentID(input.AgentID),
	}
	if input.TicketID != nil {
		predicates = append(predicates, entagentstepevent.TicketID(*input.TicketID))
	}

	items, err := r.client.AgentStepEvent.Query().
		Where(predicates...).
		Order(entagentstepevent.ByCreatedAt(entsql.OrderDesc()), entagentstepevent.ByID(entsql.OrderDesc())).
		Limit(input.Limit).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agent steps: %w", err)
	}

	return mapAgentStepEntries(items), nil
}

func (r *EntRepository) ListAgentRunTraceEntries(ctx context.Context, input domain.ListAgentRunTraceEntries) ([]domain.AgentTraceEntry, error) {
	exists, err := r.client.AgentRun.Query().
		Where(
			entagentrun.ID(input.AgentRunID),
			entagentrun.HasTicketWith(entticket.ProjectID(input.ProjectID)),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check run before listing trace entries: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	query := r.client.AgentTraceEvent.Query().
		Where(
			entagenttraceevent.ProjectID(input.ProjectID),
			entagenttraceevent.AgentRunID(input.AgentRunID),
		).
		Order(entagenttraceevent.BySequence(entsql.OrderAsc()), entagenttraceevent.ByID(entsql.OrderAsc()))
	if input.Limit > 0 {
		query = query.Limit(input.Limit)
	}

	items, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list run trace entries: %w", err)
	}

	return mapAgentTraceEntries(items), nil
}

func (r *EntRepository) ListAgentRunStepEntries(ctx context.Context, input domain.ListAgentRunStepEntries) ([]domain.AgentStepEntry, error) {
	exists, err := r.client.AgentRun.Query().
		Where(
			entagentrun.ID(input.AgentRunID),
			entagentrun.HasTicketWith(entticket.ProjectID(input.ProjectID)),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check run before listing step entries: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	query := r.client.AgentStepEvent.Query().
		Where(
			entagentstepevent.ProjectID(input.ProjectID),
			entagentstepevent.AgentRunID(input.AgentRunID),
		).
		Order(entagentstepevent.ByCreatedAt(entsql.OrderAsc()), entagentstepevent.ByID(entsql.OrderAsc()))
	if input.Limit > 0 {
		query = query.Limit(input.Limit)
	}

	items, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list run step entries: %w", err)
	}

	return mapAgentStepEntries(items), nil
}

func mapActivityEvents(items []*ent.ActivityEvent) []domain.ActivityEvent {
	events := make([]domain.ActivityEvent, 0, len(items))
	for _, item := range items {
		events = append(events, mapActivityEvent(item))
	}

	return events
}

func mapActivityEvent(item *ent.ActivityEvent) domain.ActivityEvent {
	eventType, unknownRaw := activityevent.ParseStoredType(item.EventType, nil)
	return domain.ActivityEvent{
		ID:                  item.ID,
		ProjectID:           item.ProjectID,
		TicketID:            item.TicketID,
		AgentID:             item.AgentID,
		EventType:           eventType,
		UnknownEventTypeRaw: unknownRaw,
		Message:             item.Message,
		Metadata:            cloneAnyMap(item.Metadata),
		CreatedAt:           cloneActivityCreatedAt(item.CreatedAt),
	}
}

func mapAgentOutputEntries(items []*ent.AgentTraceEvent) []domain.AgentOutputEntry {
	entries := make([]domain.AgentOutputEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, mapAgentOutputEntry(item))
	}

	return entries
}

func mapAgentTraceEntries(items []*ent.AgentTraceEvent) []domain.AgentTraceEntry {
	entries := make([]domain.AgentTraceEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, mapAgentTraceEntry(item))
	}

	return entries
}

func mapAgentTraceEntry(item *ent.AgentTraceEvent) domain.AgentTraceEntry {
	return domain.AgentTraceEntry{
		ID:         item.ID,
		ProjectID:  item.ProjectID,
		AgentID:    item.AgentID,
		TicketID:   uuidPointer(item.TicketID),
		AgentRunID: item.AgentRunID,
		Sequence:   item.Sequence,
		Provider:   item.Provider,
		Kind:       item.Kind,
		Stream:     item.Stream,
		Output:     item.Text,
		Payload:    cloneAnyMap(item.Payload),
		CreatedAt:  cloneActivityCreatedAt(item.CreatedAt),
	}
}

func mapAgentOutputEntry(item *ent.AgentTraceEvent) domain.AgentOutputEntry {
	return domain.AgentOutputEntry{
		ID:         item.ID,
		ProjectID:  item.ProjectID,
		AgentID:    item.AgentID,
		TicketID:   uuidPointer(item.TicketID),
		AgentRunID: item.AgentRunID,
		Stream:     item.Stream,
		Output:     item.Text,
		CreatedAt:  cloneActivityCreatedAt(item.CreatedAt),
	}
}

func mapAgentStepEntries(items []*ent.AgentStepEvent) []domain.AgentStepEntry {
	entries := make([]domain.AgentStepEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, mapAgentStepEntry(item))
	}

	return entries
}

func mapAgentStepEntry(item *ent.AgentStepEvent) domain.AgentStepEntry {
	return domain.AgentStepEntry{
		ID:                 item.ID,
		ProjectID:          item.ProjectID,
		AgentID:            item.AgentID,
		TicketID:           uuidPointer(item.TicketID),
		AgentRunID:         item.AgentRunID,
		StepStatus:         item.StepStatus,
		Summary:            item.Summary,
		SourceTraceEventID: item.SourceTraceEventID,
		CreatedAt:          cloneActivityCreatedAt(item.CreatedAt),
	}
}

func uuidPointer(value uuid.UUID) *uuid.UUID {
	if value == uuid.Nil {
		return nil
	}

	cloned := value
	return &cloned
}

func cloneActivityCreatedAt(value time.Time) time.Time {
	return value.UTC()
}
