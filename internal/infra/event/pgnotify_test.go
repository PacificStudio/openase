package event

import (
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestNewPGNotifyBusRequiresDSN(t *testing.T) {
	if _, err := NewPGNotifyBus("   ", nil); err == nil {
		t.Fatal("expected missing dsn error")
	}
}

func TestPGNotifyEncodeDecodeRoundTrip(t *testing.T) {
	message, err := provider.NewJSONEvent(
		provider.MustParseTopic("ticket.events"),
		provider.MustParseEventType("ticket.updated"),
		map[string]any{"ticket_id": "ASE-21", "status": "in_progress"},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}

	encoded, err := encodePGNotifyEvent(message)
	if err != nil {
		t.Fatalf("encodePGNotifyEvent returned error: %v", err)
	}

	decoded, err := decodePGNotifyEvent(encoded)
	if err != nil {
		t.Fatalf("decodePGNotifyEvent returned error: %v", err)
	}

	if decoded.Topic != message.Topic {
		t.Fatalf("expected topic %q, got %q", message.Topic, decoded.Topic)
	}
	if decoded.Type != message.Type {
		t.Fatalf("expected type %q, got %q", message.Type, decoded.Type)
	}
	if string(decoded.Payload) != string(message.Payload) {
		t.Fatalf("expected payload %s, got %s", string(message.Payload), string(decoded.Payload))
	}
}

func TestPGNotifyEncodeRejectsOversizedPayload(t *testing.T) {
	message, err := provider.NewJSONEvent(
		provider.MustParseTopic("ticket.events"),
		provider.MustParseEventType("ticket.updated"),
		map[string]string{"blob": strings.Repeat("x", maxPGNotifyPayloadBytes)},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}

	if _, err := encodePGNotifyEvent(message); err == nil {
		t.Fatal("expected oversize payload error")
	}
}
