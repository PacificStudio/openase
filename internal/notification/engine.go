package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

var (
	ticketEventsTopic      = provider.MustParseTopic("ticket.events")
	ticketCreatedEventType = provider.MustParseEventType("ticket.created")
	ticketUpdatedEventType = provider.MustParseEventType("ticket.updated")
	ticketStatusEventType  = provider.MustParseEventType("ticket.status_changed")
)

// Engine subscribes to runtime events and sends best-effort notifications.
type Engine struct {
	logger  *slog.Logger
	events  provider.EventProvider
	service *Service
}

// NewEngine constructs a notification engine.
func NewEngine(service *Service, events provider.EventProvider, logger *slog.Logger) *Engine {
	resolvedLogger := logger
	if resolvedLogger == nil {
		resolvedLogger = slog.Default()
	}

	return &Engine{
		logger:  resolvedLogger.With("component", "notification-engine"),
		events:  events,
		service: service,
	}
}

// Start subscribes to supported event topics and runs the fan-out loop in a goroutine.
func (e *Engine) Start(ctx context.Context) error {
	if e.service == nil || e.events == nil {
		return nil
	}

	stream, err := e.events.Subscribe(ctx, ticketEventsTopic)
	if err != nil {
		return fmt.Errorf("subscribe notification engine: %w", err)
	}

	go e.run(ctx, stream)
	return nil
}

func (e *Engine) run(ctx context.Context, stream <-chan provider.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-stream:
			if !ok {
				return
			}

			projectID, message, err := renderMessage(event)
			if err != nil {
				e.logger.Warn(
					"notification event ignored",
					"topic", event.Topic.String(),
					"type", event.Type.String(),
					"error", err,
				)
				continue
			}

			if err := e.service.SendToProjectChannels(ctx, projectID, message); err != nil {
				e.logger.Warn(
					"notification dispatch failed",
					"project_id", projectID.String(),
					"type", event.Type.String(),
					"error", err,
				)
			}
		}
	}
}

type ticketEventPayload struct {
	ProjectID string           `json:"project_id"`
	Ticket    ticketEventModel `json:"ticket"`
}

type ticketEventModel struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	StatusName string `json:"status_name"`
	Priority   string `json:"priority"`
	Type       string `json:"type"`
}

func renderMessage(event provider.Event) (uuid.UUID, domain.Message, error) {
	if event.Topic != ticketEventsTopic {
		return uuid.UUID{}, domain.Message{}, fmt.Errorf("unsupported topic %s", event.Topic)
	}

	var payload ticketEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return uuid.UUID{}, domain.Message{}, fmt.Errorf("decode ticket event payload: %w", err)
	}

	projectID, err := uuid.Parse(strings.TrimSpace(payload.ProjectID))
	if err != nil {
		return uuid.UUID{}, domain.Message{}, fmt.Errorf("ticket event project_id is invalid: %w", err)
	}

	title := payload.Ticket.Title
	if title == "" {
		title = payload.Ticket.Identifier
	}

	message := domain.Message{
		Level: "info",
		Metadata: map[string]string{
			"event_type":  event.Type.String(),
			"project_id":  payload.ProjectID,
			"ticket_id":   payload.Ticket.ID,
			"identifier":  payload.Ticket.Identifier,
			"status_name": payload.Ticket.StatusName,
			"priority":    payload.Ticket.Priority,
			"ticket_type": payload.Ticket.Type,
		},
	}

	switch event.Type {
	case ticketCreatedEventType:
		message.Title = fmt.Sprintf("Ticket created: %s", payload.Ticket.Identifier)
		message.Body = fmt.Sprintf("%s\nStatus: %s\nPriority: %s", title, payload.Ticket.StatusName, payload.Ticket.Priority)
	case ticketStatusEventType:
		message.Title = fmt.Sprintf("Ticket status changed: %s", payload.Ticket.Identifier)
		message.Body = fmt.Sprintf("%s\nNew status: %s", title, payload.Ticket.StatusName)
	case ticketUpdatedEventType:
		message.Title = fmt.Sprintf("Ticket updated: %s", payload.Ticket.Identifier)
		message.Body = fmt.Sprintf("%s\nStatus: %s", title, payload.Ticket.StatusName)
	default:
		message.Title = fmt.Sprintf("Ticket event: %s", payload.Ticket.Identifier)
		message.Body = fmt.Sprintf("%s\nEvent: %s", title, event.Type.String())
	}

	return projectID, message, nil
}
