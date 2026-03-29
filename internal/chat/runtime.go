package chat

import (
	"context"
	"fmt"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
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

func remainingTurns(maxTurns int, turnsUsed int) *int {
	if maxTurns <= 0 {
		return nil
	}

	remaining := 0
	if maxTurns > turnsUsed {
		remaining = maxTurns - turnsUsed
	}

	return &remaining
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
