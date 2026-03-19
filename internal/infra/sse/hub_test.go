package sse

import (
	"context"
	"testing"
	"time"

	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestHubFansOutEventsToMatchingSubscribers(t *testing.T) {
	bus := eventinfra.NewChannelBus()
	hub := NewHub(bus, nil)
	defer func() {
		if err := hub.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	}()

	topic := provider.MustParseTopic("runtime.events")
	firstCtx, firstCancel := context.WithCancel(context.Background())
	defer firstCancel()
	secondCtx, secondCancel := context.WithCancel(context.Background())
	defer secondCancel()

	first, err := hub.Register(firstCtx, topic)
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	second, err := hub.Register(secondCtx, topic)
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	message, err := provider.NewJSONEvent(
		topic,
		provider.MustParseEventType("runtime.started"),
		map[string]string{"mode": "all-in-one"},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}
	if err := bus.Publish(context.Background(), message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	assertEventReceived(t, first, message)
	assertEventReceived(t, second, message)
}

func TestHubUnregistersConnectionsOnContextCancel(t *testing.T) {
	bus := eventinfra.NewChannelBus()
	hub := NewHub(bus, nil)
	defer func() {
		if err := hub.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := hub.Register(ctx, provider.MustParseTopic("runtime.events"))
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if got := hub.ActiveConnections(); got != 1 {
		t.Fatalf("expected 1 active connection, got %d", got)
	}

	cancel()

	select {
	case _, ok := <-stream:
		if ok {
			t.Fatal("expected stream to close after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream close")
	}

	if got := hub.ActiveConnections(); got != 0 {
		t.Fatalf("expected 0 active connections, got %d", got)
	}
	if len(hub.topicStreams) != 0 {
		t.Fatalf("expected topic streams to be cleaned up, got %d", len(hub.topicStreams))
	}
}

func assertEventReceived(t *testing.T, stream <-chan provider.Event, want provider.Event) {
	t.Helper()

	select {
	case got := <-stream:
		if got.Topic != want.Topic {
			t.Fatalf("expected topic %q, got %q", want.Topic, got.Topic)
		}
		if got.Type != want.Type {
			t.Fatalf("expected type %q, got %q", want.Type, got.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE hub delivery")
	}
}
