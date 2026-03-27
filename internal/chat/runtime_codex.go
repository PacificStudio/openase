package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	codexadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

type CodexRuntime struct {
	adapter  *codexadapter.Adapter
	mu       sync.Mutex
	sessions map[SessionID]*codexRuntimeSession
}

type codexRuntimeSession struct {
	session   *codexadapter.Session
	turnsUsed int
	running   bool
}

func NewCodexRuntime(adapter *codexadapter.Adapter) *CodexRuntime {
	if adapter == nil {
		return nil
	}

	return &CodexRuntime{adapter: adapter}
}

func (r *CodexRuntime) Supports(providerItem catalogdomain.AgentProvider) bool {
	return r != nil &&
		r.adapter != nil &&
		providerItem.AdapterType == catalogdomain.AgentProviderAdapterTypeCodexAppServer
}

func (r *CodexRuntime) StartTurn(ctx context.Context, input RuntimeTurnInput) (TurnStream, error) {
	if !r.Supports(input.Provider) {
		return TurnStream{}, fmt.Errorf("%w: %s", ErrProviderUnsupported, input.Provider.AdapterType)
	}

	state, err := r.ensureSession(ctx, input)
	if err != nil {
		return TurnStream{}, err
	}

	r.mu.Lock()
	if state.running {
		r.mu.Unlock()
		return TurnStream{}, fmt.Errorf("chat session %s already has a running turn", input.SessionID)
	}
	state.running = true
	r.mu.Unlock()

	turn, err := state.session.SendPrompt(ctx, input.Message)
	if err != nil {
		r.mu.Lock()
		state.running = false
		r.mu.Unlock()
		return TurnStream{}, fmt.Errorf("start codex chat turn: %w", err)
	}

	events := make(chan StreamEvent, 64)
	go r.bridgeTurn(input.SessionID, input.MaxTurns, turn.TurnID, state, events)

	return TurnStream{Events: events}, nil
}

func (r *CodexRuntime) CloseSession(sessionID SessionID) bool {
	r.mu.Lock()
	state := r.sessions[sessionID]
	if state != nil {
		delete(r.sessions, sessionID)
	}
	r.mu.Unlock()

	if state == nil || state.session == nil {
		return false
	}

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()
	_ = state.session.Stop(closeCtx)
	return true
}

func (r *CodexRuntime) ensureSession(ctx context.Context, input RuntimeTurnInput) (*codexRuntimeSession, error) {
	r.mu.Lock()
	if r.sessions == nil {
		r.sessions = make(map[SessionID]*codexRuntimeSession)
	}
	state := r.sessions[input.SessionID]
	r.mu.Unlock()
	if state != nil {
		return state, nil
	}

	command, err := provider.ParseAgentCLICommand(input.Provider.CliCommand)
	if err != nil {
		return nil, err
	}

	var workingDirectory *provider.AbsolutePath
	if input.WorkingDirectory != "" {
		workingDirectory = &input.WorkingDirectory
	}

	processSpec, err := provider.NewAgentCLIProcessSpec(
		command,
		buildCodexArgs(input.Provider.CliArgs),
		workingDirectory,
		provider.AuthConfigEnvironment(input.Provider.AuthConfig),
	)
	if err != nil {
		return nil, err
	}

	session, err := r.adapter.Start(ctx, codexadapter.StartRequest{
		Process: processSpec,
		Initialize: codexadapter.InitializeParams{
			ClientName:    "openase",
			ClientVersion: "dev",
			ClientTitle:   "OpenASE",
		},
		Thread: codexadapter.ThreadStartParams{
			WorkingDirectory:       input.WorkingDirectory.String(),
			Model:                  input.Provider.ModelName,
			ServiceName:            "openase",
			DeveloperInstructions:  input.SystemPrompt,
			ApprovalPolicy:         "never",
			Sandbox:                "danger-full-access",
			Ephemeral:              boolPointer(true),
			PersistExtendedHistory: true,
		},
		Turn: codexadapter.TurnConfig{
			WorkingDirectory: input.WorkingDirectory.String(),
			Title:            "OpenASE Ephemeral Chat",
			ApprovalPolicy:   "never",
			SandboxPolicy: map[string]any{
				"type":          "dangerFullAccess",
				"networkAccess": true,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("start codex chat session: %w", err)
	}

	state = &codexRuntimeSession{session: session}

	r.mu.Lock()
	existing := r.sessions[input.SessionID]
	if existing == nil {
		r.sessions[input.SessionID] = state
		r.mu.Unlock()
		return state, nil
	}
	r.mu.Unlock()

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()
	_ = session.Stop(closeCtx)
	return existing, nil
}

func (r *CodexRuntime) bridgeTurn(
	sessionID SessionID,
	maxTurns int,
	turnID string,
	state *codexRuntimeSession,
	events chan<- StreamEvent,
) {
	defer close(events)
	defer func() {
		r.mu.Lock()
		state.running = false
		r.mu.Unlock()
	}()

	for event := range state.session.Events() {
		switch event.Type {
		case codexadapter.EventTypeOutputProduced:
			if event.Output == nil || event.Output.TurnID != turnID || event.Output.Text == "" {
				continue
			}
			if event.Output.Stream == "assistant" {
				events <- StreamEvent{
					Event:   "message",
					Payload: textPayload{Type: "text", Content: event.Output.Text},
				}
				continue
			}
			events <- StreamEvent{
				Event: "message",
				Payload: map[string]any{
					"type": "task_progress",
					"raw": map[string]any{
						"stream":   event.Output.Stream,
						"text":     event.Output.Text,
						"phase":    event.Output.Phase,
						"snapshot": event.Output.Snapshot,
					},
				},
			}
		case codexadapter.EventTypeToolCallRequested:
			if event.ToolCall == nil || event.ToolCall.TurnID != turnID {
				continue
			}
			events <- StreamEvent{
				Event: "message",
				Payload: map[string]any{
					"type": "task_notification",
					"raw": map[string]any{
						"tool":      event.ToolCall.Tool,
						"arguments": decodeRawJSON(event.ToolCall.Arguments),
					},
				},
			}
		case codexadapter.EventTypeTurnCompleted:
			if event.Turn == nil || event.Turn.TurnID != turnID {
				continue
			}

			r.mu.Lock()
			state.turnsUsed++
			turnsUsed := state.turnsUsed
			r.mu.Unlock()

			turnsRemaining := 0
			if maxTurns > turnsUsed {
				turnsRemaining = maxTurns - turnsUsed
			}
			events <- StreamEvent{
				Event: "done",
				Payload: donePayload{
					SessionID:      sessionID.String(),
					TurnsUsed:      turnsUsed,
					TurnsRemaining: turnsRemaining,
				},
			}
			return
		case codexadapter.EventTypeTurnFailed:
			if event.Turn == nil || event.Turn.TurnID != turnID {
				continue
			}
			message := "codex chat turn failed"
			if event.Turn.Error != nil && event.Turn.Error.Message != "" {
				message = event.Turn.Error.Message
			}
			events <- StreamEvent{
				Event:   "error",
				Payload: errorPayload{Message: message},
			}
			return
		}
	}

	events <- StreamEvent{
		Event:   "error",
		Payload: errorPayload{Message: "codex chat session ended unexpectedly"},
	}
}

func boolPointer(value bool) *bool {
	return &value
}
