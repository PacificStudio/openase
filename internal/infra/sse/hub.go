package sse

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

const connectionBufferSize = 32

var sseHubComponent = logging.DeclareComponent("sse-hub")

type Hub struct {
	events provider.EventProvider
	logger *slog.Logger

	baseCtx    context.Context
	baseCancel context.CancelFunc

	mu               sync.RWMutex
	closed           bool
	nextConnectionID int
	connections      map[int]connection
	topicMembers     map[provider.Topic]map[int]chan provider.Event
	topicStreams     map[provider.Topic]topicStream
}

type connection struct {
	topics []provider.Topic
	out    chan provider.Event
}

type topicStream struct {
	cancel context.CancelFunc
}

func NewHub(events provider.EventProvider, logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}

	baseCtx, baseCancel := context.WithCancel(context.Background())

	return &Hub{
		events:       events,
		logger:       logging.WithComponent(logger, sseHubComponent),
		baseCtx:      baseCtx,
		baseCancel:   baseCancel,
		connections:  make(map[int]connection),
		topicMembers: make(map[provider.Topic]map[int]chan provider.Event),
		topicStreams: make(map[provider.Topic]topicStream),
	}
}

func (h *Hub) Register(ctx context.Context, topics ...provider.Topic) (<-chan provider.Event, error) {
	uniqueTopics, err := dedupeTopics(topics)
	if err != nil {
		return nil, err
	}

	out := make(chan provider.Event, connectionBufferSize)

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, fmt.Errorf("sse hub is closed")
	}

	connectionID := h.nextConnectionID
	h.nextConnectionID++
	h.connections[connectionID] = connection{
		topics: uniqueTopics,
		out:    out,
	}

	topicsToSubscribe := make([]provider.Topic, 0, len(uniqueTopics))
	for _, topic := range uniqueTopics {
		if h.topicMembers[topic] == nil {
			h.topicMembers[topic] = make(map[int]chan provider.Event)
		}
		h.topicMembers[topic][connectionID] = out
		if _, exists := h.topicStreams[topic]; !exists {
			topicsToSubscribe = append(topicsToSubscribe, topic)
		}
	}

	for _, topic := range topicsToSubscribe {
		if err := h.startTopicStreamLocked(topic); err != nil {
			streamsToCancel := h.removeConnectionLocked(connectionID)
			h.mu.Unlock()
			for _, stream := range streamsToCancel {
				stream.cancel()
			}
			close(out)
			return nil, err
		}
	}
	h.mu.Unlock()

	go func() {
		<-ctx.Done()
		h.unregister(connectionID)
	}()

	return out, nil
}

func (h *Hub) ActiveConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.connections)
}

func (h *Hub) Close() error {
	h.baseCancel()

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true

	connections := h.connections
	h.connections = make(map[int]connection)

	streams := h.topicStreams
	h.topicMembers = make(map[provider.Topic]map[int]chan provider.Event)
	h.topicStreams = make(map[provider.Topic]topicStream)
	h.mu.Unlock()

	for _, stream := range streams {
		stream.cancel()
	}
	for _, conn := range connections {
		close(conn.out)
	}

	return nil
}

func (h *Hub) startTopicStreamLocked(topic provider.Topic) error {
	if h.events == nil {
		return fmt.Errorf("sse hub requires an event provider")
	}

	subscribeCtx, cancel := context.WithCancel(h.baseCtx)
	stream, err := h.events.Subscribe(subscribeCtx, topic)
	if err != nil {
		cancel()
		return fmt.Errorf("subscribe topic %q: %w", topic, err)
	}

	h.topicStreams[topic] = topicStream{cancel: cancel}

	go h.runTopicStream(topic, stream)

	return nil
}

func (h *Hub) runTopicStream(topic provider.Topic, stream <-chan provider.Event) {
	for event := range stream {
		h.broadcast(topic, event)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}

	if len(h.topicMembers[topic]) == 0 {
		delete(h.topicStreams, topic)
		return
	}

	delete(h.topicStreams, topic)
	h.logger.Warn("topic stream stopped while subscribers remain", "topic", topic.String())
}

func (h *Hub) broadcast(topic provider.Topic, event provider.Event) {
	h.mu.RLock()
	members := h.topicMembers[topic]
	targets := make([]chan provider.Event, 0, len(members))
	for _, out := range members {
		targets = append(targets, out)
	}
	h.mu.RUnlock()

	for _, out := range targets {
		select {
		case out <- event:
		default:
			h.logger.Warn("dropping sse event for slow subscriber", "topic", topic.String(), "type", event.Type.String())
		}
	}
}

func (h *Hub) unregister(connectionID int) {
	h.mu.Lock()
	conn, ok := h.connections[connectionID]
	if !ok {
		h.mu.Unlock()
		return
	}

	streamsToCancel := h.removeConnectionLocked(connectionID)
	h.mu.Unlock()

	for _, stream := range streamsToCancel {
		stream.cancel()
	}
	close(conn.out)
}

func (h *Hub) removeConnectionLocked(connectionID int) []topicStream {
	conn, ok := h.connections[connectionID]
	if !ok {
		return nil
	}

	delete(h.connections, connectionID)

	streamsToCancel := make([]topicStream, 0, len(conn.topics))
	for _, topic := range conn.topics {
		members := h.topicMembers[topic]
		delete(members, connectionID)
		if len(members) != 0 {
			continue
		}

		delete(h.topicMembers, topic)
		if stream, exists := h.topicStreams[topic]; exists {
			delete(h.topicStreams, topic)
			streamsToCancel = append(streamsToCancel, stream)
		}
	}

	return streamsToCancel
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
