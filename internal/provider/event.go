package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var eventTokenPattern = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*$`)

// Topic names an event stream channel.
type Topic string

// ParseTopic validates a raw event topic token.
func ParseTopic(raw string) (Topic, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("topic must not be empty")
	}
	if len(trimmed) > 128 {
		return "", fmt.Errorf("topic %q exceeds 128 characters", trimmed)
	}
	if !eventTokenPattern.MatchString(trimmed) {
		return "", fmt.Errorf("topic %q must match %s", trimmed, eventTokenPattern.String())
	}

	return Topic(trimmed), nil
}

// MustParseTopic parses a topic and panics on invalid input.
func MustParseTopic(raw string) Topic {
	topic, err := ParseTopic(raw)
	if err != nil {
		panic(err)
	}

	return topic
}

func (t Topic) String() string {
	return string(t)
}

// EventType names a specific event kind within a topic.
type EventType string

// ParseEventType validates a raw event type token.
func ParseEventType(raw string) (EventType, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("event type must not be empty")
	}
	if len(trimmed) > 128 {
		return "", fmt.Errorf("event type %q exceeds 128 characters", trimmed)
	}
	if !eventTokenPattern.MatchString(trimmed) {
		return "", fmt.Errorf("event type %q must match %s", trimmed, eventTokenPattern.String())
	}

	return EventType(trimmed), nil
}

// MustParseEventType parses an event type and panics on invalid input.
func MustParseEventType(raw string) EventType {
	eventType, err := ParseEventType(raw)
	if err != nil {
		panic(err)
	}

	return eventType
}

func (t EventType) String() string {
	return string(t)
}

// Event is the envelope published through the event provider abstraction.
type Event struct {
	Topic       Topic           `json:"topic"`
	Type        EventType       `json:"type"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	PublishedAt time.Time       `json:"published_at"`
}

// NewEvent constructs a validated event envelope.
func NewEvent(topic Topic, eventType EventType, payload json.RawMessage, publishedAt time.Time) (Event, error) {
	if topic == "" {
		return Event{}, fmt.Errorf("topic must not be empty")
	}
	if eventType == "" {
		return Event{}, fmt.Errorf("event type must not be empty")
	}
	if publishedAt.IsZero() {
		return Event{}, fmt.Errorf("published_at must not be zero")
	}

	return Event{
		Topic:       topic,
		Type:        eventType,
		Payload:     payload,
		PublishedAt: publishedAt.UTC(),
	}, nil
}

// NewJSONEvent marshals a JSON payload and constructs a validated event envelope.
func NewJSONEvent(topic Topic, eventType EventType, payload any, publishedAt time.Time) (Event, error) {
	var encoded json.RawMessage
	if payload != nil {
		bytes, err := json.Marshal(payload)
		if err != nil {
			return Event{}, fmt.Errorf("marshal event payload: %w", err)
		}
		encoded = bytes
	}

	return NewEvent(topic, eventType, encoded, publishedAt)
}

// EventProvider publishes and subscribes runtime events.
type EventProvider interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(ctx context.Context, topics ...Topic) (<-chan Event, error)
	Close() error
}
