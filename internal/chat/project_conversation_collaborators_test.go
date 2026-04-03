package chat

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProjectConversationStreamBrokerCleansUpWatcher(t *testing.T) {
	t.Parallel()

	broker := newProjectConversationStreamBroker()
	conversationID := uuid.New()

	events, cleanup := broker.Watch(conversationID, StreamEvent{
		Event:   "session",
		Payload: map[string]any{"conversation_id": conversationID.String()},
	})
	if event := requireProjectConversationStreamEvent(t, events); event.Event != "session" {
		t.Fatalf("initial event = %q, want session", event.Event)
	}

	cleanup()
	broker.Broadcast(conversationID, StreamEvent{Event: "message"})
	requireClosedProjectConversationStream(t, events)
}

func TestProjectConversationStreamBrokerDropsBlockedWatcherOnly(t *testing.T) {
	t.Parallel()

	broker := newProjectConversationStreamBroker()
	conversationID := uuid.New()
	otherConversationID := uuid.New()

	blocked, cleanupBlocked := broker.Watch(conversationID, StreamEvent{Event: "session"})
	defer cleanupBlocked()
	active, cleanupActive := broker.Watch(conversationID, StreamEvent{Event: "session"})
	defer cleanupActive()
	other, cleanupOther := broker.Watch(otherConversationID, StreamEvent{Event: "session"})
	defer cleanupOther()

	requireProjectConversationStreamEvent(t, blocked)
	requireProjectConversationStreamEvent(t, active)
	requireProjectConversationStreamEvent(t, other)

	for index := range projectConversationStreamBufferSize {
		broker.Broadcast(conversationID, StreamEvent{
			Event: "message",
			Payload: map[string]any{
				"index": index,
			},
		})
		if event := requireProjectConversationStreamEvent(t, active); event.Event != "message" {
			t.Fatalf("active event = %q, want message", event.Event)
		}
	}

	broker.Broadcast(conversationID, StreamEvent{Event: "message"})
	if event := requireProjectConversationStreamEvent(t, active); event.Event != "message" {
		t.Fatalf("active event after blocked buffer = %q, want message", event.Event)
	}
	for range projectConversationStreamBufferSize {
		requireProjectConversationStreamEvent(t, blocked)
	}
	requireNoProjectConversationStreamEvent(t, blocked)

	broker.Broadcast(otherConversationID, StreamEvent{Event: "message"})
	if event := requireProjectConversationStreamEvent(t, other); event.Event != "message" {
		t.Fatalf("other event = %q, want message", event.Event)
	}
}

func TestProjectConversationStreamBrokerGuaranteedEventDisplacesBufferedNoise(t *testing.T) {
	t.Parallel()

	broker := newProjectConversationStreamBroker()
	conversationID := uuid.New()

	blocked, cleanupBlocked := broker.Watch(conversationID, StreamEvent{Event: "session"})
	defer cleanupBlocked()

	requireProjectConversationStreamEvent(t, blocked)

	for index := range projectConversationStreamBufferSize {
		broker.Broadcast(conversationID, StreamEvent{
			Event: "message",
			Payload: map[string]any{
				"index": index,
			},
		})
	}

	broker.Broadcast(conversationID, StreamEvent{
		Event: "turn_done",
		Payload: map[string]any{
			"conversation_id": conversationID.String(),
			"turn_id":         uuid.New().String(),
		},
	})

	seenTurnDone := false
	for range projectConversationStreamBufferSize {
		if event := requireProjectConversationStreamEvent(t, blocked); event.Event == "turn_done" {
			seenTurnDone = true
		}
	}

	if !seenTurnDone {
		t.Fatal("expected buffered watcher to receive turn_done event")
	}
}

func TestProjectConversationRuntimeManagerCloseRemovesRegistryAndClosesSession(t *testing.T) {
	t.Parallel()

	manager := newProjectConversationRuntimeManager(nil, nil, nil, nil, nil, nil)
	conversationID := uuid.New()
	runtime := &fakeRuntime{closeResult: true}

	manager.live[conversationID] = &liveProjectConversation{runtime: runtime}
	live, ok := manager.Close(conversationID)
	if !ok || live == nil {
		t.Fatal("expected live runtime to be removed")
	}
	if len(runtime.closeCalls) != 1 || runtime.closeCalls[0] != SessionID(conversationID.String()) {
		t.Fatalf("close calls = %+v, want %s", runtime.closeCalls, conversationID)
	}
	if _, exists := manager.live[conversationID]; exists {
		t.Fatal("expected runtime registry entry to be removed")
	}
}

func requireClosedProjectConversationStream(t *testing.T, events <-chan StreamEvent) {
	t.Helper()

	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("expected stream to be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for stream close")
	}
}
