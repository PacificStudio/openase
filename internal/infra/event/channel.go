package event

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

var _ = logging.DeclareComponent("event-channel-bus")

// ChannelBus is an in-process pub/sub implementation for runtime events.
type ChannelBus struct {
	mu                  sync.RWMutex
	closed              bool
	nextSubscriberID    int
	topicSubscribers    map[provider.Topic]map[int]chan provider.Event
	activeSubscriptions map[int]channelSubscription
}

type channelSubscription struct {
	topics []provider.Topic
	out    chan provider.Event
}

// NewChannelBus constructs an in-process event bus backed by Go channels.
func NewChannelBus() *ChannelBus {
	return &ChannelBus{
		topicSubscribers:    make(map[provider.Topic]map[int]chan provider.Event),
		activeSubscriptions: make(map[int]channelSubscription),
	}
}

// Publish fan-outs an event to subscribers of the event topic.
func (b *ChannelBus) Publish(ctx context.Context, event provider.Event) error {
	if event.Topic == "" {
		return errors.New("event topic must not be empty")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return errors.New("channel bus is closed")
	}

	for _, subscriber := range b.topicSubscribers[event.Topic] {
		select {
		case subscriber <- event:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// Subscribe registers a subscriber for one or more event topics.
func (b *ChannelBus) Subscribe(ctx context.Context, topics ...provider.Topic) (<-chan provider.Event, error) {
	uniqueTopics, err := dedupeTopics(topics)
	if err != nil {
		return nil, err
	}

	out := make(chan provider.Event, 16)

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil, errors.New("channel bus is closed")
	}

	subscriberID := b.nextSubscriberID
	b.nextSubscriberID++
	for _, topic := range uniqueTopics {
		if b.topicSubscribers[topic] == nil {
			b.topicSubscribers[topic] = make(map[int]chan provider.Event)
		}
		b.topicSubscribers[topic][subscriberID] = out
	}
	b.activeSubscriptions[subscriberID] = channelSubscription{
		topics: uniqueTopics,
		out:    out,
	}
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.removeSubscriber(subscriberID)
	}()

	return out, nil
}

// Close shuts down the bus and closes all active subscriptions.
func (b *ChannelBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true

	subscriptions := b.activeSubscriptions
	b.activeSubscriptions = make(map[int]channelSubscription)
	b.topicSubscribers = make(map[provider.Topic]map[int]chan provider.Event)
	b.mu.Unlock()

	for _, subscription := range subscriptions {
		close(subscription.out)
	}

	return nil
}

func (b *ChannelBus) removeSubscriber(subscriberID int) {
	b.mu.Lock()
	subscription, ok := b.activeSubscriptions[subscriberID]
	if !ok {
		b.mu.Unlock()
		return
	}

	delete(b.activeSubscriptions, subscriberID)
	for _, topic := range subscription.topics {
		delete(b.topicSubscribers[topic], subscriberID)
		if len(b.topicSubscribers[topic]) == 0 {
			delete(b.topicSubscribers, topic)
		}
	}
	b.mu.Unlock()

	close(subscription.out)
}

func dedupeTopics(topics []provider.Topic) ([]provider.Topic, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("at least one topic is required")
	}

	seen := make(map[provider.Topic]struct{}, len(topics))
	uniqueTopics := make([]provider.Topic, 0, len(topics))
	for _, topic := range topics {
		if topic == "" {
			return nil, fmt.Errorf("topic must not be empty")
		}
		if _, exists := seen[topic]; exists {
			continue
		}
		seen[topic] = struct{}{}
		uniqueTopics = append(uniqueTopics, topic)
	}

	return uniqueTopics, nil
}
