package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

type SessionID string

func ParseSessionID(raw string) (SessionID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("chat session id must not be empty")
	}

	return SessionID(trimmed), nil
}

func (s SessionID) String() string {
	return string(s)
}

type RuntimeTurnInput struct {
	SessionID        SessionID
	Provider         catalogdomain.AgentProvider
	Message          string
	SystemPrompt     string
	WorkingDirectory provider.AbsolutePath
	MaxTurns         int
	MaxBudgetUSD     float64
}

type Runtime interface {
	Supports(catalogdomain.AgentProvider) bool
	StartTurn(context.Context, RuntimeTurnInput) (TurnStream, error)
	CloseSession(SessionID) bool
}

func NewRuntime(runtimes ...Runtime) Runtime {
	filtered := make([]Runtime, 0, len(runtimes))
	for _, runtime := range runtimes {
		if runtime != nil {
			filtered = append(filtered, runtime)
		}
	}

	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return &compositeRuntime{runtimes: filtered}
	}
}

type compositeRuntime struct {
	runtimes []Runtime
}

func (r *compositeRuntime) Supports(providerItem catalogdomain.AgentProvider) bool {
	if r == nil {
		return false
	}

	for _, runtime := range r.runtimes {
		if runtime.Supports(providerItem) {
			return true
		}
	}

	return false
}

func (r *compositeRuntime) StartTurn(ctx context.Context, input RuntimeTurnInput) (TurnStream, error) {
	if r == nil {
		return TurnStream{}, ErrUnavailable
	}

	for _, runtime := range r.runtimes {
		if !runtime.Supports(input.Provider) {
			continue
		}
		return runtime.StartTurn(ctx, input)
	}

	return TurnStream{}, fmt.Errorf("%w: %s", ErrProviderUnsupported, input.Provider.AdapterType)
}

func (r *compositeRuntime) CloseSession(sessionID SessionID) bool {
	if r == nil {
		return false
	}

	closed := false
	for _, runtime := range r.runtimes {
		if runtime.CloseSession(sessionID) {
			closed = true
		}
	}

	return closed
}

type sessionProviderRegistry struct {
	mu       sync.Mutex
	provider map[SessionID]uuid.UUID
}

func (r *sessionProviderRegistry) Register(sessionID SessionID, providerID uuid.UUID) {
	if sessionID == "" || providerID == uuid.Nil {
		return
	}

	r.mu.Lock()
	if r.provider == nil {
		r.provider = make(map[SessionID]uuid.UUID)
	}
	r.provider[sessionID] = providerID
	r.mu.Unlock()
}

func (r *sessionProviderRegistry) Resolve(sessionID SessionID) (uuid.UUID, bool) {
	if sessionID == "" {
		return uuid.UUID{}, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.provider == nil {
		return uuid.UUID{}, false
	}

	providerID, ok := r.provider[sessionID]
	return providerID, ok
}

func (r *sessionProviderRegistry) Delete(sessionID SessionID) {
	if sessionID == "" {
		return
	}

	r.mu.Lock()
	if r.provider != nil {
		delete(r.provider, sessionID)
	}
	r.mu.Unlock()
}
