package machinechannel

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type SessionCloser interface {
	Close(reason string) error
}

type RegisteredSession struct {
	MachineID       uuid.UUID
	SessionID       string
	ConnectedAt     time.Time
	LastHeartbeatAt time.Time
	Replaced        bool
	closer          SessionCloser
}

type ExpiredSession struct {
	MachineID      uuid.UUID
	SessionID      string
	DisconnectedAt time.Time
}

type SessionRegistry struct {
	mu       sync.Mutex
	timeout  time.Duration
	sessions map[string]RegisteredSession
	machines map[uuid.UUID]string
}

func NewSessionRegistry(timeout time.Duration) *SessionRegistry {
	if timeout <= 0 {
		timeout = DefaultHeartbeatTimeout
	}
	return &SessionRegistry{
		timeout:  timeout,
		sessions: map[string]RegisteredSession{},
		machines: map[uuid.UUID]string{},
	}
}

func (r *SessionRegistry) Register(machineID uuid.UUID, sessionID string, connectedAt time.Time, closer SessionCloser) (RegisteredSession, *RegisteredSession) {
	r.mu.Lock()

	session := RegisteredSession{
		MachineID:       machineID,
		SessionID:       sessionID,
		ConnectedAt:     connectedAt.UTC(),
		LastHeartbeatAt: connectedAt.UTC(),
		closer:          closer,
	}
	var replaced *RegisteredSession
	if existingID, ok := r.machines[machineID]; ok {
		if existing, exists := r.sessions[existingID]; exists {
			copyExisting := existing
			replaced = &copyExisting
			session.Replaced = true
		}
		delete(r.sessions, existingID)
	}
	r.sessions[sessionID] = session
	r.machines[machineID] = sessionID
	r.mu.Unlock()

	if replaced != nil && replaced.closer != nil {
		_ = replaced.closer.Close("replaced by newer reverse websocket session")
	}
	return session, replaced
}

func (r *SessionRegistry) Heartbeat(sessionID string, heartbeatAt time.Time) (RegisteredSession, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return RegisteredSession{}, false
	}
	session.LastHeartbeatAt = heartbeatAt.UTC()
	r.sessions[sessionID] = session
	return session, true
}

func (r *SessionRegistry) Remove(sessionID string) (RegisteredSession, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return RegisteredSession{}, false
	}
	delete(r.sessions, sessionID)
	if current, exists := r.machines[session.MachineID]; exists && current == sessionID {
		delete(r.machines, session.MachineID)
	}
	return session, true
}

func (r *SessionRegistry) Snapshot(machineID uuid.UUID) (RegisteredSession, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sessionID, ok := r.machines[machineID]
	if !ok {
		return RegisteredSession{}, false
	}
	session, exists := r.sessions[sessionID]
	return session, exists
}

func (r *SessionRegistry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.sessions)
}

func (r *SessionRegistry) Expire(now time.Time) []ExpiredSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := now.UTC().Add(-r.timeout)
	expired := make([]ExpiredSession, 0)
	for sessionID, session := range r.sessions {
		if !session.LastHeartbeatAt.Before(cutoff) {
			continue
		}
		delete(r.sessions, sessionID)
		if current, exists := r.machines[session.MachineID]; exists && current == sessionID {
			delete(r.machines, session.MachineID)
		}
		expired = append(expired, ExpiredSession{
			MachineID:      session.MachineID,
			SessionID:      sessionID,
			DisconnectedAt: now.UTC(),
		})
	}
	return expired
}

func (r *SessionRegistry) CloseAll(reason string) []RegisteredSession {
	r.mu.Lock()
	sessions := make([]RegisteredSession, 0, len(r.sessions))
	for _, session := range r.sessions {
		sessions = append(sessions, session)
	}
	r.sessions = map[string]RegisteredSession{}
	r.machines = map[uuid.UUID]string{}
	r.mu.Unlock()

	for _, session := range sessions {
		if session.closer != nil {
			_ = session.closer.Close(reason)
		}
	}

	return sessions
}
