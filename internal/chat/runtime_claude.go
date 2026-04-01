package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

type ClaudeRuntime struct {
	adapter        provider.ClaudeCodeAdapter
	activeSessions runtimeCancelRegistry
	nativeSessions claudeSessionRegistry
}

func NewClaudeRuntime(adapter provider.ClaudeCodeAdapter) *ClaudeRuntime {
	if adapter == nil {
		return nil
	}

	return &ClaudeRuntime{adapter: adapter}
}

func (r *ClaudeRuntime) Supports(providerItem catalogdomain.AgentProvider) bool {
	return r != nil &&
		r.adapter != nil &&
		providerItem.AdapterType == catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI
}

func (r *ClaudeRuntime) StartTurn(ctx context.Context, input RuntimeTurnInput) (TurnStream, error) {
	if !r.Supports(input.Provider) {
		return TurnStream{}, fmt.Errorf("%w: %s", ErrProviderUnsupported, input.Provider.AdapterType)
	}

	sessionSpec, err := r.buildSessionSpec(input)
	if err != nil {
		return TurnStream{}, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	session, err := r.adapter.Start(runCtx, sessionSpec)
	if err != nil {
		cancel()
		return TurnStream{}, fmt.Errorf("start claude code chat turn: %w", err)
	}

	turnInput, err := provider.NewClaudeCodeTurnInput(input.Message)
	if err != nil {
		cancel()
		return TurnStream{}, err
	}
	if err := session.Send(runCtx, turnInput); err != nil {
		cancel()
		return TurnStream{}, fmt.Errorf("send chat turn input: %w", err)
	}

	events := make(chan StreamEvent, 64)
	go r.bridgeSession(runCtx, cancel, input.SessionID, input.MaxTurns, session, events)

	return TurnStream{Events: events}, nil
}

func (r *ClaudeRuntime) CloseSession(sessionID SessionID) bool {
	r.nativeSessions.Delete(sessionID)
	return r.activeSessions.Close(sessionID)
}

func (r *ClaudeRuntime) buildSessionSpec(input RuntimeTurnInput) (provider.ClaudeCodeSessionSpec, error) {
	command, err := provider.ParseAgentCLICommand(input.Provider.CliCommand)
	if err != nil {
		return provider.ClaudeCodeSessionSpec{}, err
	}

	var workingDirectory *provider.AbsolutePath
	if input.WorkingDirectory != "" {
		workingDirectory = &input.WorkingDirectory
	}

	maxTurns := input.MaxTurns
	var maxTurnsPointer *int
	if maxTurns > 0 {
		maxTurnsPointer = &maxTurns
	}
	maxBudgetUSD := input.MaxBudgetUSD
	resumeSessionID := r.nativeSessions.Resolve(input.SessionID)

	return provider.NewClaudeCodeSessionSpec(
		command,
		buildBaseArgs(input.Provider.CliArgs, input.Provider.ModelName),
		workingDirectory,
		append(provider.AuthConfigEnvironment(input.Provider.AuthConfig), input.Environment...),
		nil,
		input.SystemPrompt,
		maxTurnsPointer,
		&maxBudgetUSD,
		resumeSessionID,
		true,
	)
}

func (r *ClaudeRuntime) bridgeSession(
	ctx context.Context,
	cancel context.CancelFunc,
	sessionID SessionID,
	maxTurns int,
	session provider.ClaudeCodeSession,
	events chan<- StreamEvent,
) {
	defer close(events)
	defer cancel()
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer closeCancel()
		_ = session.Close(closeCtx)
	}()

	r.activeSessions.Register(sessionID, cancel)
	defer r.activeSessions.Delete(sessionID)

	nativeSessionID, hasNativeSessionID := session.SessionID()
	if hasNativeSessionID {
		r.nativeSessions.Register(sessionID, nativeSessionID)
	}

	eventCh := session.Events()
	errorCh := session.Errors()

	for eventCh != nil || errorCh != nil {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-errorCh:
			if !ok {
				errorCh = nil
				continue
			}
			events <- StreamEvent{
				Event:   "error",
				Payload: errorPayload{Message: err.Error()},
			}
		case event, ok := <-eventCh:
			if !ok {
				eventCh = nil
				continue
			}

			if !hasNativeSessionID && event.SessionID != "" {
				parsed, err := provider.ParseClaudeCodeSessionID(event.SessionID)
				if err == nil {
					hasNativeSessionID = true
					r.nativeSessions.Register(sessionID, parsed)
				}
			}

			for _, item := range mapClaudeEvent(sessionID, maxTurns, event) {
				events <- item
			}
			if event.Kind == provider.ClaudeCodeEventKindResult {
				return
			}
		}
	}
}

func mapClaudeEvent(sessionID SessionID, maxTurns int, event provider.ClaudeCodeEvent) []StreamEvent {
	switch event.Kind {
	case provider.ClaudeCodeEventKindAssistant:
		texts := extractAssistantTextBlocks(event.Message)
		if len(texts) == 0 {
			return nil
		}

		return normalizeAssistantText(strings.Join(texts, "\n\n"))
	case provider.ClaudeCodeEventKindTaskStart:
		return []StreamEvent{newTaskMessageEvent(chatMessageTypeTaskStarted, decodeRawJSON(event.Raw))}
	case provider.ClaudeCodeEventKindTaskProgress:
		return []StreamEvent{newTaskMessageEvent(chatMessageTypeTaskProgress, decodeRawJSON(event.Raw))}
	case provider.ClaudeCodeEventKindTaskNotice:
		return []StreamEvent{newTaskMessageEvent(chatMessageTypeTaskNotification, decodeRawJSON(event.Raw))}
	case provider.ClaudeCodeEventKindUnknown:
		payload := map[string]any{"type": event.UnknownType}
		if data := decodeRawJSON(event.Raw); data != nil {
			payload["raw"] = data
		}
		return []StreamEvent{{Event: "message", Payload: payload}}
	case provider.ClaudeCodeEventKindResult:
		costUSD := cloneCostUSD(event.TotalCostUSD)
		if event.UsageInfo != nil {
			costUSD = cloneCostUSD(event.UsageInfo.CostUSD)
		}
		return []StreamEvent{{
			Event: "done",
			Payload: donePayload{
				SessionID:      sessionID.String(),
				CostUSD:        costUSD,
				TurnsUsed:      event.NumTurns,
				TurnsRemaining: remainingTurns(maxTurns, event.NumTurns),
			},
		}}
	default:
		return nil
	}
}

type runtimeCancelRegistry struct {
	mu       sync.Mutex
	sessions map[SessionID]context.CancelFunc
}

func (r *runtimeCancelRegistry) Register(sessionID SessionID, cancel context.CancelFunc) {
	if sessionID == "" || cancel == nil {
		return
	}

	r.mu.Lock()
	if r.sessions == nil {
		r.sessions = make(map[SessionID]context.CancelFunc)
	}
	previous := r.sessions[sessionID]
	r.sessions[sessionID] = cancel
	r.mu.Unlock()

	if previous != nil {
		previous()
	}
}

func (r *runtimeCancelRegistry) Delete(sessionID SessionID) {
	if sessionID == "" {
		return
	}

	r.mu.Lock()
	if r.sessions != nil {
		delete(r.sessions, sessionID)
	}
	r.mu.Unlock()
}

func (r *runtimeCancelRegistry) Close(sessionID SessionID) bool {
	if sessionID == "" {
		return false
	}

	r.mu.Lock()
	cancel := r.sessions[sessionID]
	if cancel != nil {
		delete(r.sessions, sessionID)
	}
	r.mu.Unlock()

	if cancel == nil {
		return false
	}

	cancel()
	return true
}

type claudeSessionRegistry struct {
	mu       sync.Mutex
	sessions map[SessionID]provider.ClaudeCodeSessionID
}

func (r *claudeSessionRegistry) Register(sessionID SessionID, native provider.ClaudeCodeSessionID) {
	if sessionID == "" || native == "" {
		return
	}

	r.mu.Lock()
	if r.sessions == nil {
		r.sessions = make(map[SessionID]provider.ClaudeCodeSessionID)
	}
	r.sessions[sessionID] = native
	r.mu.Unlock()
}

func (r *claudeSessionRegistry) Resolve(sessionID SessionID) *provider.ClaudeCodeSessionID {
	if sessionID == "" {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	native, ok := r.sessions[sessionID]
	if !ok {
		return nil
	}

	cloned := native
	return &cloned
}

func (r *claudeSessionRegistry) Delete(sessionID SessionID) {
	if sessionID == "" {
		return
	}

	r.mu.Lock()
	if r.sessions != nil {
		delete(r.sessions, sessionID)
	}
	r.mu.Unlock()
}
