package activity

import (
	"context"
	"fmt"
	"time"

	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

var activityStreamTopic = provider.MustParseTopic("activity.events")

type RecordInput struct {
	ProjectID uuid.UUID
	TicketID  *uuid.UUID
	AgentID   *uuid.UUID
	EventType activityevent.Type
	Message   string
	Metadata  map[string]any
	CreatedAt time.Time
}

type Recorder interface {
	RecordActivityEvent(ctx context.Context, input RecordInput) (catalogdomain.ActivityEvent, error)
}

type RecordFunc func(context.Context, RecordInput) (catalogdomain.ActivityEvent, error)

func (fn RecordFunc) RecordActivityEvent(ctx context.Context, input RecordInput) (catalogdomain.ActivityEvent, error) {
	return fn(ctx, input)
}

type Emitter struct {
	recorder Recorder
	events   provider.EventProvider
}

func NewEmitter(recorder Recorder, events provider.EventProvider) *Emitter {
	return &Emitter{recorder: recorder, events: events}
}

func (e *Emitter) Emit(ctx context.Context, input RecordInput) (*catalogdomain.ActivityEvent, error) {
	if e == nil {
		return nil, nil
	}
	if _, err := activityevent.ParseRawType(input.EventType.String()); err != nil {
		return nil, fmt.Errorf("parse activity event type: %w", err)
	}

	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	input.CreatedAt = createdAt.UTC()
	input.Metadata = cloneMetadata(input.Metadata)

	var item *catalogdomain.ActivityEvent
	if e.recorder != nil {
		recorded, err := e.recorder.RecordActivityEvent(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("record activity event: %w", err)
		}
		item = &recorded
	}
	if e.events == nil {
		return item, nil
	}

	providerType, err := provider.ParseEventType(input.EventType.String())
	if err != nil {
		return nil, fmt.Errorf("parse provider event type: %w", err)
	}

	payload := map[string]any{
		"event": mapActivityEventPayload(input, item),
	}
	event, err := provider.NewJSONEvent(activityStreamTopic, providerType, payload, createdAt)
	if err != nil {
		return nil, fmt.Errorf("build activity stream event: %w", err)
	}
	if err := e.events.Publish(ctx, event); err != nil {
		return nil, fmt.Errorf("publish activity stream event: %w", err)
	}

	return item, nil
}

func cloneMetadata(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}
	return cloned
}

func mapActivityEventPayload(input RecordInput, item *catalogdomain.ActivityEvent) map[string]any {
	if item != nil {
		return map[string]any{
			"id":         item.ID.String(),
			"project_id": item.ProjectID.String(),
			"ticket_id":  uuidPointerString(item.TicketID),
			"agent_id":   uuidPointerString(item.AgentID),
			"event_type": item.EventType.String(),
			"message":    item.Message,
			"metadata":   cloneMetadata(item.Metadata),
			"created_at": item.CreatedAt.UTC().Format(time.RFC3339),
		}
	}

	return map[string]any{
		"id":         "",
		"project_id": input.ProjectID.String(),
		"ticket_id":  uuidPointerString(input.TicketID),
		"agent_id":   uuidPointerString(input.AgentID),
		"event_type": input.EventType.String(),
		"message":    input.Message,
		"metadata":   cloneMetadata(input.Metadata),
		"created_at": input.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func uuidPointerString(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}
	formatted := value.String()
	return &formatted
}
