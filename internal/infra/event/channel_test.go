package event

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestChannelBusPublishAndSubscribe(t *testing.T) {
	bus := NewChannelBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topic := provider.MustParseTopic("runtime.events")
	eventType := provider.MustParseEventType("orchestrator.tick")
	subscriber, err := bus.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	message, err := provider.NewJSONEvent(topic, eventType, map[string]string{"status": "ok"}, time.Now())
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}

	if err := bus.Publish(context.Background(), message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	select {
	case received := <-subscriber:
		if received.Topic != topic {
			t.Fatalf("expected topic %q, got %q", topic, received.Topic)
		}
		if received.Type != eventType {
			t.Fatalf("expected event type %q, got %q", eventType, received.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel bus delivery")
	}
}

func TestChannelBusDeduplicatesTopicsPerSubscriber(t *testing.T) {
	bus := NewChannelBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topic := provider.MustParseTopic("runtime.events")
	subscriber, err := bus.Subscribe(ctx, topic, topic)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	message, err := provider.NewJSONEvent(topic, provider.MustParseEventType("runtime.started"), nil, time.Now())
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}
	if err := bus.Publish(context.Background(), message); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	select {
	case <-subscriber:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first event")
	}

	select {
	case extra := <-subscriber:
		t.Fatalf("expected one delivery, got extra event %+v", extra)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestChannelBusCloseStopsNewSubscriptions(t *testing.T) {
	bus := NewChannelBus()
	if err := bus.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	_, err := bus.Subscribe(context.Background(), provider.MustParseTopic("runtime.events"))
	if err == nil {
		t.Fatal("expected subscribe to fail after close")
	}

	if !errors.Is(err, context.Canceled) && err.Error() != "channel bus is closed" {
		t.Fatalf("expected closed bus error, got %v", err)
	}
}

func TestChannelBusCoversValidationAndClosePaths(t *testing.T) {
	bus := NewChannelBus()

	if _, err := bus.Subscribe(context.Background()); err == nil || err.Error() != "at least one topic is required" {
		t.Fatalf("expected missing topics error, got %v", err)
	}
	if _, err := bus.Subscribe(context.Background(), provider.Topic("")); err == nil || err.Error() != "topic must not be empty" {
		t.Fatalf("expected empty topic error, got %v", err)
	}
	if err := bus.Publish(context.Background(), provider.Event{}); err == nil || err.Error() != "event topic must not be empty" {
		t.Fatalf("expected empty event topic error, got %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	topic := provider.MustParseTopic("ticket.events")
	subscriber, err := bus.Subscribe(ctx, topic)
	if err != nil {
		t.Fatalf("Subscribe returned error: %v", err)
	}

	cancel()

	select {
	case _, ok := <-subscriber:
		if ok {
			t.Fatal("expected subscriber channel to close after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subscriber close")
	}

	if err := bus.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := bus.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}

	message, err := provider.NewJSONEvent(topic, provider.MustParseEventType("ticket.updated"), nil, time.Now())
	if err != nil {
		t.Fatalf("NewJSONEvent returned error: %v", err)
	}
	if err := bus.Publish(context.Background(), message); err == nil || err.Error() != "channel bus is closed" {
		t.Fatalf("expected closed bus publish error, got %v", err)
	}

	bus.removeSubscriber(999)
}
