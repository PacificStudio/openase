package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

var (
	ticketEventsTopic     = provider.MustParseTopic("ticket.events")
	ticketStatusEventType = provider.MustParseEventType("ticket.status_changed")
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

			projectID, contextMap, err := buildRuleContext(event)
			if err != nil {
				e.logger.Warn(
					"notification event ignored",
					"operation", "build_notification_rule_context",
					"topic", event.Topic.String(),
					"type", event.Type.String(),
					"payload_bytes", len(event.Payload),
					"published_at", event.PublishedAt.UTC().Format(time.RFC3339),
					"error", err,
				)
				continue
			}

			rules, err := e.service.MatchingRules(ctx, projectID, domain.RuleEventType(event.Type.String()))
			if err != nil {
				e.logger.Warn(
					"notification rule lookup failed",
					"project_id", projectID.String(),
					"type", event.Type.String(),
					"error", err,
				)
				continue
			}

			for _, rule := range rules {
				if !rule.Matches(contextMap) {
					continue
				}
				message, err := rule.RenderMessage(contextMap)
				if err != nil {
					e.logger.Warn(
						"notification rule render failed",
						"rule_id", rule.ID.String(),
						"project_id", projectID.String(),
						"type", event.Type.String(),
						"error", err,
					)
					continue
				}
				if err := e.service.SendRule(ctx, rule, message); err != nil {
					e.logger.Warn(
						"notification dispatch failed",
						"operation", "send_notification_rule",
						"rule_id", rule.ID.String(),
						"project_id", projectID.String(),
						"channel_id", rule.Channel.ID.String(),
						"channel_name", rule.Channel.Name,
						"channel_type", rule.Channel.Type.String(),
						"type", event.Type.String(),
						"error", err,
					)
				}
			}
		}
	}
}

func buildRuleContext(event provider.Event) (uuid.UUID, map[string]any, error) {
	if event.Topic != ticketEventsTopic {
		return uuid.UUID{}, nil, fmt.Errorf("unsupported topic %s", event.Topic)
	}

	payload := map[string]any{}
	if len(event.Payload) > 0 {
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return uuid.UUID{}, nil, fmt.Errorf("decode ticket event payload: %w", err)
		}
	}

	projectIDValue, ok := payload["project_id"].(string)
	if !ok {
		return uuid.UUID{}, nil, fmt.Errorf("ticket event project_id is missing")
	}
	projectID, err := uuid.Parse(projectIDValue)
	if err != nil {
		return uuid.UUID{}, nil, fmt.Errorf("ticket event project_id is invalid: %w", err)
	}

	contextMap := map[string]any{
		"event": map[string]any{
			"topic":        event.Topic.String(),
			"type":         event.Type.String(),
			"published_at": event.PublishedAt.UTC().Format(time.RFC3339),
		},
		"event_type":   event.Type.String(),
		"project_id":   projectIDValue,
		"published_at": event.PublishedAt.UTC().Format(time.RFC3339),
		"payload":      payload,
	}
	for key, value := range payload {
		if _, exists := contextMap[key]; !exists {
			contextMap[key] = value
		}
	}
	if ticket, ok := payload["ticket"].(map[string]any); ok {
		contextMap["ticket"] = ticket
		for key, value := range ticket {
			if _, exists := contextMap[key]; !exists {
				contextMap[key] = value
			}
		}
		if event.Type == ticketStatusEventType {
			if statusName, ok := ticket["status_name"]; ok {
				contextMap["new_status"] = statusName
			}
		}
	}

	return projectID, contextMap, nil
}
