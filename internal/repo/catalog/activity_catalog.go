package catalog

import (
	"context"
	"fmt"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentactivityinstance "github.com/BetterAndBetterII/openase/ent/agentactivityinstance"
	entagentrawevent "github.com/BetterAndBetterII/openase/ent/agentrawevent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entagentstepevent "github.com/BetterAndBetterII/openase/ent/agentstepevent"
	entagenttraceevent "github.com/BetterAndBetterII/openase/ent/agenttraceevent"
	entagenttranscriptentry "github.com/BetterAndBetterII/openase/ent/agenttranscriptentry"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatconversationdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

func (r *EntRepository) ListActivityEvents(ctx context.Context, input domain.ListActivityEvents) (domain.ActivityEventPage, error) {
	exists, err := r.client.Project.Query().
		Where(entproject.ID(input.ProjectID)).
		Exist(ctx)
	if err != nil {
		return domain.ActivityEventPage{}, fmt.Errorf("check project before listing activity events: %w", err)
	}
	if !exists {
		return domain.ActivityEventPage{}, ErrNotFound
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
	if input.Before != nil {
		predicates = append(predicates, entactivityevent.Or(
			entactivityevent.CreatedAtLT(input.Before.CreatedAt),
			entactivityevent.And(
				entactivityevent.CreatedAtEQ(input.Before.CreatedAt),
				entactivityevent.IDLT(input.Before.ID),
			),
		))
	}

	items, err := r.client.ActivityEvent.Query().
		Where(predicates...).
		Order(entactivityevent.ByCreatedAt(entsql.OrderDesc()), entactivityevent.ByID(entsql.OrderDesc())).
		Limit(input.Limit + 1).
		All(ctx)
	if err != nil {
		return domain.ActivityEventPage{}, fmt.Errorf("list activity events: %w", err)
	}

	hasMore := len(items) > input.Limit
	if hasMore {
		items = items[:input.Limit]
	}
	events := mapActivityEvents(items)

	page := domain.ActivityEventPage{
		Events:  events,
		HasMore: hasMore,
	}
	if hasMore && len(events) > 0 {
		page.NextCursor = domain.ActivityEventCursorFor(events[len(events)-1]).String()
	}

	return page, nil
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

func (r *EntRepository) ListAgentRunRawEvents(ctx context.Context, input domain.ListAgentRunRawEvents) (domain.AgentRunRawEventPage, error) {
	exists, err := r.client.AgentRun.Query().
		Where(
			entagentrun.ID(input.AgentRunID),
			entagentrun.HasTicketWith(entticket.ProjectID(input.ProjectID)),
		).
		Exist(ctx)
	if err != nil {
		return domain.AgentRunRawEventPage{}, fmt.Errorf("check run before listing raw events: %w", err)
	}
	if !exists {
		return domain.AgentRunRawEventPage{}, ErrNotFound
	}

	items, err := r.client.AgentRawEvent.Query().
		Where(
			entagentrawevent.ProjectID(input.ProjectID),
			entagentrawevent.AgentRunID(input.AgentRunID),
		).
		Order(entagentrawevent.ByOccurredAt(entsql.OrderAsc()), entagentrawevent.ByID(entsql.OrderAsc())).
		All(ctx)
	if err != nil {
		return domain.AgentRunRawEventPage{}, fmt.Errorf("list run raw events: %w", err)
	}

	return buildAgentRawEventPage(items, input), nil
}

func (r *EntRepository) ListAgentRunActivities(ctx context.Context, input domain.ListAgentRunActivities) ([]domain.AgentActivityInstance, error) {
	exists, err := r.client.AgentRun.Query().
		Where(
			entagentrun.ID(input.AgentRunID),
			entagentrun.HasTicketWith(entticket.ProjectID(input.ProjectID)),
		).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check run before listing activities: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	predicates := []predicate.AgentActivityInstance{
		entagentactivityinstance.ProjectID(input.ProjectID),
		entagentactivityinstance.AgentRunID(input.AgentRunID),
	}
	if input.Status != "" {
		predicates = append(predicates, entagentactivityinstance.Status(strings.TrimSpace(input.Status)))
	}

	items, err := r.client.AgentActivityInstance.Query().
		Where(predicates...).
		Order(entagentactivityinstance.ByUpdatedAt(entsql.OrderDesc()), entagentactivityinstance.ByID(entsql.OrderDesc())).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list run activities: %w", err)
	}

	return mapAgentActivityInstances(items), nil
}

func (r *EntRepository) ListAgentRunTranscriptEntries(
	ctx context.Context,
	input domain.ListAgentRunTranscriptEntries,
) (domain.AgentRunTranscriptEntryPage, error) {
	exists, err := r.client.AgentRun.Query().
		Where(
			entagentrun.ID(input.AgentRunID),
			entagentrun.HasTicketWith(entticket.ProjectID(input.ProjectID)),
		).
		Exist(ctx)
	if err != nil {
		return domain.AgentRunTranscriptEntryPage{}, fmt.Errorf("check run before listing transcript entries: %w", err)
	}
	if !exists {
		return domain.AgentRunTranscriptEntryPage{}, ErrNotFound
	}

	items, err := r.client.AgentTranscriptEntry.Query().
		Where(
			entagenttranscriptentry.ProjectID(input.ProjectID),
			entagenttranscriptentry.AgentRunID(input.AgentRunID),
		).
		Order(entagenttranscriptentry.ByCreatedAt(entsql.OrderAsc()), entagenttranscriptentry.ByID(entsql.OrderAsc())).
		All(ctx)
	if err != nil {
		return domain.AgentRunTranscriptEntryPage{}, fmt.Errorf("list run transcript entries: %w", err)
	}

	return buildAgentTranscriptEntryPage(items, input), nil
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

func mapAgentRawEvents(items []*ent.AgentRawEvent) []domain.AgentRawEventEntry {
	entries := make([]domain.AgentRawEventEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, mapAgentRawEvent(item))
	}
	return entries
}

func mapAgentRawEvent(item *ent.AgentRawEvent) domain.AgentRawEventEntry {
	return domain.AgentRawEventEntry{
		ID:                   item.ID,
		ProjectID:            item.ProjectID,
		AgentID:              item.AgentID,
		TicketID:             optionalUUIDPointer(item.TicketID),
		AgentRunID:           item.AgentRunID,
		Provider:             item.Provider,
		ProviderEventKind:    item.ProviderEventKind,
		ProviderEventSubtype: item.ProviderEventSubtype,
		ProviderEventID:      cloneOptionalString(item.ProviderEventID),
		ThreadID:             cloneOptionalString(item.ThreadID),
		TurnID:               cloneOptionalString(item.TurnID),
		ActivityHintID:       cloneOptionalString(item.ActivityHintID),
		OccurredAt:           cloneActivityCreatedAt(item.OccurredAt),
		Payload:              cloneAnyMap(item.Payload),
		TextExcerpt:          item.TextExcerpt,
	}
}

func mapAgentActivityInstances(items []*ent.AgentActivityInstance) []domain.AgentActivityInstance {
	entries := make([]domain.AgentActivityInstance, 0, len(items))
	for _, item := range items {
		entries = append(entries, mapAgentActivityInstance(item))
	}
	return entries
}

func mapAgentActivityInstance(item *ent.AgentActivityInstance) domain.AgentActivityInstance {
	return domain.AgentActivityInstance{
		ID:                 item.ID,
		ProjectID:          item.ProjectID,
		AgentID:            item.AgentID,
		TicketID:           optionalUUIDPointer(item.TicketID),
		AgentRunID:         item.AgentRunID,
		Provider:           item.Provider,
		ActivityKind:       item.ActivityKind,
		ActivityID:         item.ActivityID,
		IDSource:           item.IDSource,
		IdentityConfidence: item.IdentityConfidence,
		ParentActivityID:   cloneOptionalString(item.ParentActivityID),
		ThreadID:           cloneOptionalString(item.ThreadID),
		TurnID:             cloneOptionalString(item.TurnID),
		Command:            cloneOptionalString(item.Command),
		ToolName:           cloneOptionalString(item.ToolName),
		Title:              cloneOptionalString(item.Title),
		Status:             item.Status,
		LiveText:           cloneOptionalString(item.LiveText),
		FinalText:          cloneOptionalString(item.FinalText),
		LiveTextBytes:      item.LiveTextBytes,
		FinalTextBytes:     item.FinalTextBytes,
		Metadata:           cloneAnyMap(item.Metadata),
		StartedAt:          cloneOptionalTime(item.StartedAt),
		UpdatedAt:          cloneActivityCreatedAt(item.UpdatedAt),
		CompletedAt:        cloneOptionalTime(item.CompletedAt),
	}
}

func mapAgentTranscriptEntries(items []*ent.AgentTranscriptEntry) []domain.AgentTranscriptEntry {
	entries := make([]domain.AgentTranscriptEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, mapAgentTranscriptEntry(item))
	}
	return entries
}

func mapAgentTranscriptEntry(item *ent.AgentTranscriptEntry) domain.AgentTranscriptEntry {
	return domain.AgentTranscriptEntry{
		ID:           item.ID,
		ProjectID:    item.ProjectID,
		AgentID:      item.AgentID,
		TicketID:     optionalUUIDPointer(item.TicketID),
		AgentRunID:   item.AgentRunID,
		Provider:     item.Provider,
		EntryKey:     item.EntryKey,
		EntryKind:    item.EntryKind,
		ActivityKind: cloneOptionalString(item.ActivityKind),
		ActivityID:   cloneOptionalString(item.ActivityID),
		Title:        cloneOptionalString(item.Title),
		Summary:      cloneOptionalString(item.Summary),
		BodyText:     cloneOptionalString(item.BodyText),
		Command:      cloneOptionalString(item.Command),
		ToolName:     cloneOptionalString(item.ToolName),
		Metadata:     cloneAnyMap(item.Metadata),
		CreatedAt:    cloneActivityCreatedAt(item.CreatedAt),
	}
}

func buildAgentRawEventPage(items []*ent.AgentRawEvent, input domain.ListAgentRunRawEvents) domain.AgentRunRawEventPage {
	mapped := mapAgentRawEvents(items)
	start, end := resolveAgentRawEventWindow(mapped, input.Before, input.After, input.Limit)
	pageEntries := mapped[start:end]
	page := domain.AgentRunRawEventPage{
		Entries:          pageEntries,
		HasOlder:         start > 0,
		HiddenOlderCount: start,
		HasNewer:         end < len(mapped),
		HiddenNewerCount: len(mapped) - end,
	}
	if len(pageEntries) > 0 {
		page.OldestCursor = domain.AgentRunEventCursorForRawEvent(pageEntries[0]).String()
		page.NewestCursor = domain.AgentRunEventCursorForRawEvent(pageEntries[len(pageEntries)-1]).String()
	}
	return page
}

func buildAgentTranscriptEntryPage(items []*ent.AgentTranscriptEntry, input domain.ListAgentRunTranscriptEntries) domain.AgentRunTranscriptEntryPage {
	mapped := mapAgentTranscriptEntries(items)
	start, end := resolveAgentTranscriptEntryWindow(mapped, input.Before, input.After, input.Limit)
	pageEntries := mapped[start:end]
	page := domain.AgentRunTranscriptEntryPage{
		Entries:          pageEntries,
		HasOlder:         start > 0,
		HiddenOlderCount: start,
		HasNewer:         end < len(mapped),
		HiddenNewerCount: len(mapped) - end,
	}
	if len(pageEntries) > 0 {
		page.OldestCursor = domain.AgentRunEventCursorForTranscriptEntry(pageEntries[0]).String()
		page.NewestCursor = domain.AgentRunEventCursorForTranscriptEntry(pageEntries[len(pageEntries)-1]).String()
	}
	return page
}

func resolveAgentRawEventWindow(
	items []domain.AgentRawEventEntry,
	before *domain.AgentRunEventCursor,
	after *domain.AgentRunEventCursor,
	limit int,
) (start int, end int) {
	if len(items) == 0 {
		return 0, 0
	}
	switch {
	case after != nil:
		start = len(items)
		for index, item := range items {
			if domain.CompareAgentRunEventCursor(domain.AgentRunEventCursorForRawEvent(item), *after) > 0 {
				start = index
				break
			}
		}
		end = min(start+limit, len(items))
	case before != nil:
		end = 0
		for index, item := range items {
			if domain.CompareAgentRunEventCursor(domain.AgentRunEventCursorForRawEvent(item), *before) >= 0 {
				end = index
				break
			}
			end = index + 1
		}
		start = max(0, end-limit)
	default:
		end = len(items)
		start = max(0, end-limit)
	}
	return start, end
}

func resolveAgentTranscriptEntryWindow(
	items []domain.AgentTranscriptEntry,
	before *domain.AgentRunEventCursor,
	after *domain.AgentRunEventCursor,
	limit int,
) (start int, end int) {
	if len(items) == 0 {
		return 0, 0
	}
	switch {
	case after != nil:
		start = len(items)
		for index, item := range items {
			if domain.CompareAgentRunEventCursor(domain.AgentRunEventCursorForTranscriptEntry(item), *after) > 0 {
				start = index
				break
			}
		}
		end = min(start+limit, len(items))
	case before != nil:
		end = 0
		for index, item := range items {
			if domain.CompareAgentRunEventCursor(domain.AgentRunEventCursorForTranscriptEntry(item), *before) >= 0 {
				end = index
				break
			}
			end = index + 1
		}
		start = max(0, end-limit)
	default:
		end = len(items)
		start = max(0, end-limit)
	}
	return start, end
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

func optionalUUIDPointer(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := strings.TrimSpace(*value)
	return &cloned
}

func cloneOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}
