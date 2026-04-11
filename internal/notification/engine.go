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
	ticketEventsTopic   = provider.MustParseTopic("ticket.events")
	agentEventsTopic    = provider.MustParseTopic("agent.events")
	activityEventsTopic = provider.MustParseTopic("activity.events")

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

	stream, err := e.events.Subscribe(ctx, supportedNotificationTopics()...)
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
	eventType, err := domain.ParseRuleEventType(event.Type.String())
	if err != nil {
		return uuid.UUID{}, nil, fmt.Errorf("unsupported event type %s", event.Type)
	}
	if !notificationEventTopicAllowed(eventType, event.Topic) {
		return uuid.UUID{}, nil, fmt.Errorf("event type %s is not wired for topic %s", eventType, event.Topic)
	}

	switch event.Topic {
	case ticketEventsTopic:
		return buildTicketRuleContext(event)
	case agentEventsTopic:
		return buildAgentRuleContext(event)
	case activityEventsTopic:
		return buildActivityRuleContext(event)
	default:
		return uuid.UUID{}, nil, fmt.Errorf("unsupported topic %s", event.Topic)
	}
}

func supportedNotificationTopics() []provider.Topic {
	seen := map[string]struct{}{}
	for _, item := range domain.SupportedRuleEventContracts() {
		seen[item.Topic] = struct{}{}
	}
	topics := make([]provider.Topic, 0, len(seen))
	for _, topic := range []provider.Topic{ticketEventsTopic, agentEventsTopic, activityEventsTopic} {
		if _, ok := seen[topic.String()]; ok {
			topics = append(topics, topic)
			delete(seen, topic.String())
		}
	}
	for topic := range seen {
		topics = append(topics, provider.MustParseTopic(topic))
	}
	return topics
}

func notificationEventTopicAllowed(eventType domain.RuleEventType, topic provider.Topic) bool {
	expectedTopic, ok := domain.RuleEventTopic(eventType)
	return ok && topic.String() == expectedTopic
}

func buildTicketRuleContext(event provider.Event) (uuid.UUID, map[string]any, error) {
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

func buildAgentRuleContext(event provider.Event) (uuid.UUID, map[string]any, error) {
	payload := map[string]any{}
	if len(event.Payload) > 0 {
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return uuid.UUID{}, nil, fmt.Errorf("decode agent event payload: %w", err)
		}
	}

	agentPayload, ok := payload["agent"].(map[string]any)
	if !ok {
		return uuid.UUID{}, nil, fmt.Errorf("agent event payload.agent is missing")
	}

	projectIDValue, ok := agentPayload["project_id"].(string)
	if !ok {
		return uuid.UUID{}, nil, fmt.Errorf("agent event project_id is missing")
	}
	projectID, err := uuid.Parse(projectIDValue)
	if err != nil {
		return uuid.UUID{}, nil, fmt.Errorf("agent event project_id is invalid: %w", err)
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
		"agent":        agentPayload,
	}
	for key, value := range agentPayload {
		if _, exists := contextMap[key]; !exists {
			contextMap[key] = value
		}
	}
	if currentTicketID, ok := agentPayload["current_ticket_id"]; ok {
		contextMap["ticket_id"] = currentTicketID
	}

	return projectID, contextMap, nil
}

func buildActivityRuleContext(event provider.Event) (uuid.UUID, map[string]any, error) {
	payload := map[string]any{}
	if len(event.Payload) > 0 {
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return uuid.UUID{}, nil, fmt.Errorf("decode activity event payload: %w", err)
		}
	}

	activityPayload, ok := payload["event"].(map[string]any)
	if !ok {
		return uuid.UUID{}, nil, fmt.Errorf("activity event payload.event is missing")
	}

	projectIDValue, ok := activityPayload["project_id"].(string)
	if !ok {
		return uuid.UUID{}, nil, fmt.Errorf("activity event project_id is missing")
	}
	projectID, err := uuid.Parse(projectIDValue)
	if err != nil {
		return uuid.UUID{}, nil, fmt.Errorf("activity event project_id is invalid: %w", err)
	}

	contextMap := map[string]any{
		"event": map[string]any{
			"topic":        event.Topic.String(),
			"type":         event.Type.String(),
			"published_at": event.PublishedAt.UTC().Format(time.RFC3339),
			"message":      stringMapValue(activityPayload, "message"),
		},
		"event_type":   event.Type.String(),
		"project_id":   projectIDValue,
		"published_at": event.PublishedAt.UTC().Format(time.RFC3339),
		"payload":      payload,
		"activity":     activityPayload,
		"message":      stringMapValue(activityPayload, "message"),
	}
	for key, value := range activityPayload {
		if _, exists := contextMap[key]; !exists {
			contextMap[key] = value
		}
	}

	metadata, ok := activityPayload["metadata"].(map[string]any)
	if ok {
		contextMap["metadata"] = metadata
		for key, value := range metadata {
			if _, exists := contextMap[key]; !exists {
				contextMap[key] = value
			}
		}
	}

	return projectID, contextMap, nil
}

func stringMapValue(values map[string]any, key string) string {
	raw, ok := values[key]
	if !ok {
		return ""
	}
	text, _ := raw.(string)
	return text
}
