package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

type sessionStore interface {
	Save(sessionID SessionID, state sessionState) error
	LoadForUser(userID UserID, sessionID SessionID) (sessionState, bool, error)
	ResolveUserSession(userID UserID) (SessionID, bool, error)
	Delete(sessionID SessionID) error
}

type fileSessionStore struct {
	mu   sync.Mutex
	path string
}

type fileSessionStorePayload struct {
	Sessions map[string]fileSessionState `json:"sessions"`
	ByUser   map[string]string           `json:"by_user"`
}

type fileSessionState struct {
	UserID                    string   `json:"user_id"`
	ProviderID                string   `json:"provider_id"`
	MaxTurns                  int      `json:"max_turns"`
	MaxBudgetUSD              float64  `json:"max_budget_usd"`
	TurnsUsed                 int      `json:"turns_used"`
	CostUSD                   float64  `json:"cost_usd"`
	HasCostUSD                bool     `json:"has_cost_usd"`
	Released                  bool     `json:"released"`
	ExhaustedMessage          string   `json:"exhausted_message,omitempty"`
	ResumeProviderThreadID    string   `json:"resume_provider_thread_id,omitempty"`
	ProviderThreadStatus      string   `json:"provider_thread_status,omitempty"`
	ProviderThreadActiveFlags []string `json:"provider_thread_active_flags,omitempty"`
}

func newFileSessionStore(path string) *fileSessionStore {
	if path == "" {
		return nil
	}
	return &fileSessionStore{path: path}
}

func (s *fileSessionStore) Save(sessionID SessionID, state sessionState) error {
	if s == nil || sessionID == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readLocked()
	if err != nil {
		return err
	}
	if payload.Sessions == nil {
		payload.Sessions = make(map[string]fileSessionState)
	}
	if payload.ByUser == nil {
		payload.ByUser = make(map[string]string)
	}

	if previousSessionID, ok := payload.ByUser[state.UserID.String()]; ok && previousSessionID != sessionID.String() {
		delete(payload.Sessions, previousSessionID)
	}
	payload.ByUser[state.UserID.String()] = sessionID.String()
	payload.Sessions[sessionID.String()] = fileSessionState{
		UserID:                    state.UserID.String(),
		ProviderID:                state.ProviderID.String(),
		MaxTurns:                  state.MaxTurns,
		MaxBudgetUSD:              state.MaxBudgetUSD,
		TurnsUsed:                 state.TurnsUsed,
		CostUSD:                   state.CostUSD,
		HasCostUSD:                state.HasCostUSD,
		Released:                  state.Released,
		ExhaustedMessage:          state.ExhaustedMessage,
		ResumeProviderThreadID:    state.ResumeProviderThreadID,
		ProviderThreadStatus:      state.ProviderThreadStatus,
		ProviderThreadActiveFlags: append([]string(nil), state.ProviderThreadActiveFlags...),
	}
	return s.writeLocked(payload)
}

func (s *fileSessionStore) LoadForUser(userID UserID, sessionID SessionID) (sessionState, bool, error) {
	if s == nil || userID == "" || sessionID == "" {
		return sessionState{}, false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readLocked()
	if err != nil {
		return sessionState{}, false, err
	}
	raw, ok := payload.Sessions[sessionID.String()]
	if !ok || raw.UserID != userID.String() {
		return sessionState{}, false, nil
	}
	state, err := raw.toSessionState()
	if err != nil {
		return sessionState{}, false, err
	}
	return state, true, nil
}

func (s *fileSessionStore) ResolveUserSession(userID UserID) (SessionID, bool, error) {
	if s == nil || userID == "" {
		return "", false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readLocked()
	if err != nil {
		return "", false, err
	}
	rawSessionID, ok := payload.ByUser[userID.String()]
	if !ok {
		return "", false, nil
	}
	sessionID, err := ParseSessionID(rawSessionID)
	if err != nil {
		return "", false, fmt.Errorf("parse stored chat session id: %w", err)
	}
	return sessionID, true, nil
}

func (s *fileSessionStore) Delete(sessionID SessionID) error {
	if s == nil || sessionID == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readLocked()
	if err != nil {
		return err
	}
	raw, ok := payload.Sessions[sessionID.String()]
	if !ok {
		return nil
	}
	delete(payload.Sessions, sessionID.String())
	if payload.ByUser[raw.UserID] == sessionID.String() {
		delete(payload.ByUser, raw.UserID)
	}
	return s.writeLocked(payload)
}

func (s *fileSessionStore) readLocked() (fileSessionStorePayload, error) {
	if s == nil || s.path == "" {
		return fileSessionStorePayload{}, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fileSessionStorePayload{
				Sessions: map[string]fileSessionState{},
				ByUser:   map[string]string{},
			}, nil
		}
		return fileSessionStorePayload{}, fmt.Errorf("read chat session store: %w", err)
	}
	if len(data) == 0 {
		return fileSessionStorePayload{
			Sessions: map[string]fileSessionState{},
			ByUser:   map[string]string{},
		}, nil
	}

	var payload fileSessionStorePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fileSessionStorePayload{}, fmt.Errorf("decode chat session store: %w", err)
	}
	if payload.Sessions == nil {
		payload.Sessions = map[string]fileSessionState{}
	}
	if payload.ByUser == nil {
		payload.ByUser = map[string]string{}
	}
	return payload, nil
}

func (s *fileSessionStore) writeLocked(payload fileSessionStorePayload) error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create chat session store directory: %w", err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode chat session store: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write chat session store temp file: %w", err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace chat session store: %w", err)
	}
	return nil
}

func (s fileSessionState) toSessionState() (sessionState, error) {
	providerID, err := uuid.Parse(s.ProviderID)
	if err != nil {
		return sessionState{}, fmt.Errorf("parse stored provider id: %w", err)
	}
	userID, err := ParseUserID(s.UserID)
	if err != nil {
		return sessionState{}, fmt.Errorf("parse stored user id: %w", err)
	}
	return sessionState{
		UserID:                    userID,
		ProviderID:                providerID,
		MaxTurns:                  s.MaxTurns,
		MaxBudgetUSD:              s.MaxBudgetUSD,
		TurnsUsed:                 s.TurnsUsed,
		CostUSD:                   s.CostUSD,
		HasCostUSD:                s.HasCostUSD,
		Released:                  s.Released,
		ExhaustedMessage:          s.ExhaustedMessage,
		ResumeProviderThreadID:    s.ResumeProviderThreadID,
		ProviderThreadStatus:      s.ProviderThreadStatus,
		ProviderThreadActiveFlags: append([]string(nil), s.ProviderThreadActiveFlags...),
	}, nil
}
