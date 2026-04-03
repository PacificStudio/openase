package orchestrator

import (
	"sync"

	"github.com/google/uuid"
)

type runtimeSessionRegistry struct {
	mu       sync.Mutex
	sessions map[uuid.UUID]agentSession
}

func newRuntimeSessionRegistry() *runtimeSessionRegistry {
	return &runtimeSessionRegistry{sessions: map[uuid.UUID]agentSession{}}
}

func (r *runtimeSessionRegistry) store(runID uuid.UUID, session agentSession) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[runID] = session
}

func (r *runtimeSessionRegistry) load(runID uuid.UUID) agentSession {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessions[runID]
}

func (r *runtimeSessionRegistry) delete(runID uuid.UUID) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, runID)
}

func (r *runtimeSessionRegistry) drain() map[uuid.UUID]agentSession {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	drained := make(map[uuid.UUID]agentSession, len(r.sessions))
	for runID, session := range r.sessions {
		drained[runID] = session
	}
	r.sessions = map[uuid.UUID]agentSession{}
	return drained
}

func (r *runtimeSessionRegistry) runIDs() []uuid.UUID {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	runIDs := make([]uuid.UUID, 0, len(r.sessions))
	for runID := range r.sessions {
		runIDs = append(runIDs, runID)
	}
	return runIDs
}

type runtimeRunTracker struct {
	mu     sync.Mutex
	runIDs map[uuid.UUID]struct{}
}

func newRuntimeRunTracker() *runtimeRunTracker {
	return &runtimeRunTracker{runIDs: map[uuid.UUID]struct{}{}}
}

func (t *runtimeRunTracker) begin(runID uuid.UUID) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.runIDs[runID]; exists {
		return false
	}
	t.runIDs[runID] = struct{}{}
	return true
}

func (t *runtimeRunTracker) finish(runID uuid.UUID) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.runIDs, runID)
}

func (t *runtimeRunTracker) active(runID uuid.UUID) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	_, active := t.runIDs[runID]
	return active
}

func (t *runtimeRunTracker) list() []uuid.UUID {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	runIDs := make([]uuid.UUID, 0, len(t.runIDs))
	for runID := range t.runIDs {
		runIDs = append(runIDs, runID)
	}
	return runIDs
}
