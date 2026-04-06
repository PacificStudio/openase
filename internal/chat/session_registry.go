package chat

import (
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type UserID string

const AnonymousUserID UserID = "anonymous"

// LocalProjectConversationUserID is the stable non-OIDC owner used for
// persistent project conversations when browser login is disabled.
const LocalProjectConversationUserID UserID = "local-user:default"

func ParseUserID(raw string) (UserID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("chat user id must not be empty")
	}

	return UserID(trimmed), nil
}

func (u UserID) String() string {
	return string(u)
}

type sessionState struct {
	UserID                    UserID
	ProviderID                uuid.UUID
	MaxTurns                  int
	MaxBudgetUSD              float64
	TurnsUsed                 int
	CostUSD                   float64
	HasCostUSD                bool
	Released                  bool
	ExhaustedMessage          string
	ResumeProviderThreadID    string
	ProviderThreadStatus      string
	ProviderThreadActiveFlags []string
}

type sessionRegistry struct {
	mu        sync.Mutex
	bySession map[SessionID]sessionState
	byUser    map[UserID]SessionID
}

func (r *sessionRegistry) Register(
	userID UserID,
	sessionID SessionID,
	providerID uuid.UUID,
	maxTurns int,
	maxBudgetUSD float64,
) {
	if sessionID == "" || providerID == uuid.Nil || userID == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.bySession == nil {
		r.bySession = make(map[SessionID]sessionState)
	}
	if r.byUser == nil {
		r.byUser = make(map[UserID]SessionID)
	}

	if previousSessionID, ok := r.byUser[userID]; ok && previousSessionID != sessionID {
		delete(r.bySession, previousSessionID)
	}

	r.bySession[sessionID] = sessionState{
		UserID:       userID,
		ProviderID:   providerID,
		MaxTurns:     maxTurns,
		MaxBudgetUSD: maxBudgetUSD,
	}
	r.byUser[userID] = sessionID
}

func (r *sessionRegistry) Remember(sessionID SessionID, state sessionState) {
	if sessionID == "" || state.UserID == "" || state.ProviderID == uuid.Nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.bySession == nil {
		r.bySession = make(map[SessionID]sessionState)
	}
	if r.byUser == nil {
		r.byUser = make(map[UserID]SessionID)
	}

	if previousSessionID, ok := r.byUser[state.UserID]; ok && previousSessionID != sessionID {
		delete(r.bySession, previousSessionID)
	}

	cloned := state
	cloned.ProviderThreadActiveFlags = append([]string(nil), state.ProviderThreadActiveFlags...)
	r.bySession[sessionID] = cloned
	r.byUser[state.UserID] = sessionID
}

func (r *sessionRegistry) Resolve(sessionID SessionID) (sessionState, bool) {
	if sessionID == "" {
		return sessionState{}, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.bySession == nil {
		return sessionState{}, false
	}

	state, ok := r.bySession[sessionID]
	return state, ok
}

func (r *sessionRegistry) ResolveForUser(userID UserID, sessionID SessionID) (sessionState, bool) {
	state, ok := r.Resolve(sessionID)
	if !ok || state.UserID != userID {
		return sessionState{}, false
	}

	return state, true
}

func (r *sessionRegistry) ResolveUserSession(userID UserID) (SessionID, bool) {
	if userID == "" {
		return "", false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.byUser == nil {
		return "", false
	}

	sessionID, ok := r.byUser[userID]
	return sessionID, ok
}

func (r *sessionRegistry) MarkUsage(sessionID SessionID, turnsUsed int, costUSD *float64) (sessionState, bool) {
	if sessionID == "" {
		return sessionState{}, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.bySession == nil {
		return sessionState{}, false
	}

	state, ok := r.bySession[sessionID]
	if !ok {
		return sessionState{}, false
	}

	state.TurnsUsed = turnsUsed
	if costUSD != nil {
		state.CostUSD = *costUSD
		state.HasCostUSD = true
	}
	r.bySession[sessionID] = state
	return state, true
}

func (r *sessionRegistry) MarkReleased(sessionID SessionID, exhaustedMessage string) (sessionState, bool) {
	if sessionID == "" {
		return sessionState{}, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.bySession == nil {
		return sessionState{}, false
	}

	state, ok := r.bySession[sessionID]
	if !ok {
		return sessionState{}, false
	}

	state.Released = true
	state.ExhaustedMessage = strings.TrimSpace(exhaustedMessage)
	r.bySession[sessionID] = state
	return state, true
}

func (r *sessionRegistry) UpdateProviderAnchor(
	sessionID SessionID,
	providerThreadID string,
	providerThreadStatus string,
	providerThreadActiveFlags []string,
) (sessionState, bool) {
	if sessionID == "" {
		return sessionState{}, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.bySession == nil {
		return sessionState{}, false
	}

	state, ok := r.bySession[sessionID]
	if !ok {
		return sessionState{}, false
	}

	if trimmed := strings.TrimSpace(providerThreadID); trimmed != "" {
		state.ResumeProviderThreadID = trimmed
	}
	if trimmed := strings.TrimSpace(providerThreadStatus); trimmed != "" {
		state.ProviderThreadStatus = trimmed
	}
	if providerThreadActiveFlags != nil {
		state.ProviderThreadActiveFlags = append([]string(nil), providerThreadActiveFlags...)
	}
	r.bySession[sessionID] = state
	return state, true
}

func (r *sessionRegistry) Delete(sessionID SessionID) (sessionState, bool) {
	if sessionID == "" {
		return sessionState{}, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.bySession == nil {
		return sessionState{}, false
	}

	state, ok := r.bySession[sessionID]
	if !ok {
		return sessionState{}, false
	}

	delete(r.bySession, sessionID)
	if r.byUser != nil {
		if currentSessionID, current := r.byUser[state.UserID]; current && currentSessionID == sessionID {
			delete(r.byUser, state.UserID)
		}
	}

	return state, true
}

type userLockRegistry struct {
	mu    sync.Mutex
	locks map[UserID]*userLockState
}

type userLockState struct {
	mu   sync.Mutex
	refs int
}

func (r *userLockRegistry) Lock(userID UserID) func() {
	r.mu.Lock()
	if r.locks == nil {
		r.locks = make(map[UserID]*userLockState)
	}

	state := r.locks[userID]
	if state == nil {
		state = &userLockState{}
		r.locks[userID] = state
	}
	state.refs++
	r.mu.Unlock()

	state.mu.Lock()
	return func() {
		state.mu.Unlock()

		r.mu.Lock()
		defer r.mu.Unlock()

		state.refs--
		if state.refs == 0 {
			delete(r.locks, userID)
		}
	}
}
