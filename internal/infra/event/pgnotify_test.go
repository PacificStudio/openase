package event

import (
	"context"
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

func TestNewPGNotifyBusDefaultsLoggerAndRejectsConnectionFailures(t *testing.T) {
	bus, err := NewPGNotifyBus(" postgres://localhost/test ", nil)
	if err != nil {
		t.Fatalf("NewPGNotifyBus returned error: %v", err)
	}
	if bus.logger == nil {
		t.Fatal("expected default logger")
	}
	if bus.dsn != "postgres://localhost/test" {
		t.Fatalf("dsn=%q, want trimmed dsn", bus.dsn)
	}

	invalidBus, err := NewPGNotifyBus("postgres://127.0.0.1:1/openase?sslmode=disable", nil)
	if err != nil {
		t.Fatalf("NewPGNotifyBus returned error: %v", err)
	}
	if _, err := invalidBus.publisherConn(context.Background()); err == nil || !strings.Contains(err.Error(), "connect pgnotify publisher") {
		t.Fatalf("expected publisher connect error, got %v", err)
	}
	if _, err := invalidBus.Subscribe(context.Background(), provider.MustParseTopic("runtime.events")); err == nil || !strings.Contains(err.Error(), "connect pgnotify subscriber") {
		t.Fatalf("expected subscriber connect error, got %v", err)
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

func TestPGNotifyDecodeRejectsMalformedPayloads(t *testing.T) {
	if _, err := decodePGNotifyEvent("{"); err == nil || !strings.Contains(err.Error(), "unmarshal pgnotify event") {
		t.Fatalf("expected unmarshal error, got %v", err)
	}
	if _, err := decodePGNotifyEvent(`{"topic":"","type":"runtime.started","payload":null,"published_at":"2026-03-27T20:00:00Z"}`); err == nil || !strings.Contains(err.Error(), "topic must not be empty") {
		t.Fatalf("expected provider validation error, got %v", err)
	}
}

func TestPGChannelNameFitsPostgresLimit(t *testing.T) {
	topic, err := provider.ParseTopic(strings.Repeat("a", 128))
	if err != nil {
		t.Fatalf("ParseTopic returned error: %v", err)
	}

	channel := pgChannelName(topic)
	if len(channel) != maxPGChannelNameBytes {
		t.Fatalf("expected channel name length %d, got %d (%q)", maxPGChannelNameBytes, len(channel), channel)
	}
	if !strings.HasPrefix(channel, pgChannelPrefix) {
		t.Fatalf("expected channel prefix %q, got %q", pgChannelPrefix, channel)
	}
	if channel != pgChannelName(topic) {
		t.Fatalf("expected deterministic channel mapping for %q", topic)
	}
}

func TestPGNotifyBusPublishesRuntimeEventsWithLengthSafeChannelNames(t *testing.T) {
	dsn := startEmbeddedPostgres(t)
	bus, err := NewPGNotifyBus(dsn, nil)
	if err != nil {
		t.Fatalf("NewPGNotifyBus returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := bus.Close(); err != nil {
			t.Errorf("Close returned error: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := provider.MustParseTopic("runtime.events")
	subscriber, err := bus.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	message, err := provider.NewJSONEvent(
		topic,
		provider.MustParseEventType("runtime.started"),
		map[string]string{"mode": "orchestrate"},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}
	if err := bus.Publish(ctx, message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	select {
	case received := <-subscriber:
		if received.Topic != message.Topic {
			t.Fatalf("expected topic %q, got %q", message.Topic, received.Topic)
		}
		if received.Type != message.Type {
			t.Fatalf("expected type %q, got %q", message.Type, received.Type)
		}
		if string(received.Payload) != string(message.Payload) {
			t.Fatalf("expected payload %s, got %s", string(message.Payload), string(received.Payload))
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for pgnotify delivery: %v", ctx.Err())
	}
}

func TestPGNotifyBusCloseIsIdempotent(t *testing.T) {
	bus, err := NewPGNotifyBus("postgres://localhost/openase", nil)
	if err != nil {
		t.Fatalf("NewPGNotifyBus returned error: %v", err)
	}
	if err := bus.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := bus.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
}

func startEmbeddedPostgres(t *testing.T) string {
	t.Helper()

	return testPostgres.NewIsolatedDatabase(t).DSN
}
