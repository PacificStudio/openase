package chat

import (
	"sync"

	"github.com/google/uuid"
)

type projectConversationStreamBroker struct {
	mu          sync.Mutex
	watchers    map[uuid.UUID]map[int]chan StreamEvent
	nextWatcher int
}

const projectConversationStreamBufferSize = 2048

func newProjectConversationStreamBroker() *projectConversationStreamBroker {
	return &projectConversationStreamBroker{
		watchers: map[uuid.UUID]map[int]chan StreamEvent{},
	}
}

func (b *projectConversationStreamBroker) Watch(
	conversationID uuid.UUID,
	initial StreamEvent,
) (<-chan StreamEvent, func()) {
	events := make(chan StreamEvent, projectConversationStreamBufferSize)

	b.mu.Lock()
	if b.watchers[conversationID] == nil {
		b.watchers[conversationID] = map[int]chan StreamEvent{}
	}
	watcherID := b.nextWatcher
	b.nextWatcher++
	b.watchers[conversationID][watcherID] = events
	b.mu.Unlock()

	events <- initial

	var once sync.Once
	return events, func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()

			if watchers := b.watchers[conversationID]; watchers != nil {
				delete(watchers, watcherID)
				if len(watchers) == 0 {
					delete(b.watchers, conversationID)
				}
			}
			close(events)
		})
	}
}

func (b *projectConversationStreamBroker) Broadcast(conversationID uuid.UUID, event StreamEvent) {
	if b == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, watcher := range b.watchers[conversationID] {
		if isGuaranteedProjectConversationEvent(event.Event) {
			enqueueGuaranteedProjectConversationEvent(watcher, event)
			continue
		}
		select {
		case watcher <- event:
		default:
		}
	}
}

func isGuaranteedProjectConversationEvent(event string) bool {
	switch event {
	case "turn_done", "error", "interrupt_requested":
		return true
	default:
		return false
	}
}

func enqueueGuaranteedProjectConversationEvent(watcher chan StreamEvent, event StreamEvent) {
	for {
		select {
		case watcher <- event:
			return
		default:
		}

		select {
		case <-watcher:
		default:
		}
	}
}
