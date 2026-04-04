package sse

import (
	"context"
	"errors"
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

func TestHubCoversValidationAndCloseBranches(t *testing.T) {
	hub := NewHub(nil, nil)

	if _, err := hub.Register(context.Background()); err == nil || err.Error() != "at least one topic is required" {
		t.Fatalf("expected missing topics error, got %v", err)
	}
	if _, err := hub.Register(context.Background(), provider.Topic("")); err == nil || err.Error() != "topic must not be empty" {
		t.Fatalf("expected empty topic error, got %v", err)
	}
	if _, err := hub.Register(context.Background(), provider.MustParseTopic("runtime.events")); err == nil || err.Error() != "sse hub requires an event provider" {
		t.Fatalf("expected missing provider error, got %v", err)
	}

	bus := eventinfra.NewChannelBus()
	closedHub := NewHub(bus, nil)
	if err := closedHub.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := closedHub.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
	if _, err := closedHub.Register(context.Background(), provider.MustParseTopic("runtime.events")); err == nil || err.Error() != "sse hub is closed" {
		t.Fatalf("expected closed hub error, got %v", err)
	}
}

func TestHubHandlesProviderFailureAndTopicCleanup(t *testing.T) {
	topic := provider.MustParseTopic("runtime.events")
	fake := &fakeEventProvider{
		subscribeErr: errors.New("boom"),
	}
	hub := NewHub(fake, nil)

	if _, err := hub.Register(context.Background(), topic); err == nil || err.Error() != `subscribe topic "runtime.events": boom` {
		t.Fatalf("expected subscribe failure, got %v", err)
	}
	if hub.ActiveConnections() != 0 {
		t.Fatalf("expected no active connections, got %d", hub.ActiveConnections())
	}

	streamHub := NewHub(fake, nil)
	streamHub.topicMembers[topic] = map[int]chan provider.Event{1: make(chan provider.Event, 1)}
	streamHub.topicStreams[topic] = topicStream{cancel: func() {}}
	streamHub.runTopicStream(topic, closedEventStream())
	if _, exists := streamHub.topicStreams[topic]; exists {
		t.Fatal("expected topic stream to be removed after stream end")
	}
}

func TestHubBroadcastDropsSlowSubscribersAndUnregisterNoops(t *testing.T) {
	bus := eventinfra.NewChannelBus()
	hub := NewHub(bus, nil)
	defer func() {
		if err := hub.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	}()

	topic := provider.MustParseTopic("runtime.events")
	slow := make(chan provider.Event)
	fast := make(chan provider.Event, 1)
	hub.topicMembers[topic] = map[int]chan provider.Event{
		1: slow,
		2: fast,
	}

	event, err := provider.NewJSONEvent(topic, provider.MustParseEventType("runtime.started"), nil, time.Now())
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}
	hub.broadcast(topic, event)

	select {
	case got := <-fast:
		if got.Type != event.Type {
			t.Fatalf("expected event type %q, got %q", event.Type, got.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for fast subscriber event")
	}

	hub.unregister(404)
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

type fakeEventProvider struct {
	subscribeErr error
}

func (f *fakeEventProvider) Publish(context.Context, provider.Event) error {
	return nil
}

func (f *fakeEventProvider) Subscribe(context.Context, ...provider.Topic) (<-chan provider.Event, error) {
	if f.subscribeErr != nil {
		return nil, f.subscribeErr
	}
	return closedEventStream(), nil
}

func (f *fakeEventProvider) Close() error {
	return nil
}

func closedEventStream() <-chan provider.Event {
	stream := make(chan provider.Event)
	close(stream)
	return stream
}
