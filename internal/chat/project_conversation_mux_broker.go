package chat

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type projectConversationMuxWatchKey struct {
	ProjectID uuid.UUID
	UserID    UserID
}

type ProjectConversationMuxEvent struct {
	Event          string
	ConversationID uuid.UUID
	Payload        any
	SentAt         time.Time
}

type projectConversationMuxBroker struct {
	mu          sync.Mutex
	watchers    map[projectConversationMuxWatchKey]map[int]chan ProjectConversationMuxEvent
	nextWatcher int
}

func newProjectConversationMuxBroker() *projectConversationMuxBroker {
	return &projectConversationMuxBroker{
		watchers: map[projectConversationMuxWatchKey]map[int]chan ProjectConversationMuxEvent{},
	}
}

func (b *projectConversationMuxBroker) Watch(
	key projectConversationMuxWatchKey,
	initial []ProjectConversationMuxEvent,
) (<-chan ProjectConversationMuxEvent, func()) {
	events := make(chan ProjectConversationMuxEvent, projectConversationStreamBufferSize)

	b.mu.Lock()
	if b.watchers[key] == nil {
		b.watchers[key] = map[int]chan ProjectConversationMuxEvent{}
	}
	watcherID := b.nextWatcher
	b.nextWatcher++
	b.watchers[key][watcherID] = events
	b.mu.Unlock()

	for _, item := range initial {
		events <- item
	}

	var once sync.Once
	return events, func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()

			if watchers := b.watchers[key]; watchers != nil {
				delete(watchers, watcherID)
				if len(watchers) == 0 {
					delete(b.watchers, key)
				}
			}
			close(events)
		})
	}
}

func (b *projectConversationMuxBroker) Broadcast(
	key projectConversationMuxWatchKey,
	event ProjectConversationMuxEvent,
) {
	if b == nil {
		return
	}
	if event.SentAt.IsZero() {
		event.SentAt = time.Now().UTC()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, watcher := range b.watchers[key] {
		if isGuaranteedProjectConversationEvent(event.Event) {
			enqueueGuaranteedProjectConversationMuxEvent(watcher, event)
			continue
		}
		select {
		case watcher <- event:
		default:
		}
	}
}

func enqueueGuaranteedProjectConversationMuxEvent(
	watcher chan ProjectConversationMuxEvent,
	event ProjectConversationMuxEvent,
) {
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
