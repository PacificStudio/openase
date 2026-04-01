package chat

import (
	"context"
	"fmt"
	"strings"
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
	session    *codexadapter.Session
	costUSD    *float64
	turnsUsed  int
	running    bool
	lastTurnID string
}

type codexAssistantItemState struct {
	text              strings.Builder
	emittedText       bool
	waitingOnSnapshot bool
}

type RuntimeInterruptEvent struct {
	RequestID string                     `json:"request_id"`
	Kind      string                     `json:"kind"`
	Options   []RuntimeInterruptDecision `json:"options,omitempty"`
	Payload   map[string]any             `json:"payload,omitempty"`
}

type RuntimeInterruptDecision struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type RuntimeSessionAnchor struct {
	ProviderThreadID string
	LastTurnID       string
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
	go r.bridgeTurn(input.SessionID, input.Provider, input.MaxTurns, turn.TurnID, state, events)

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
		append(provider.AuthConfigEnvironment(input.Provider.AuthConfig), input.Environment...),
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
			ResumeThreadID:         strings.TrimSpace(input.ResumeProviderThreadID),
			WorkingDirectory:       input.WorkingDirectory.String(),
			Model:                  input.Provider.ModelName,
			ServiceName:            "openase",
			DeveloperInstructions:  input.SystemPrompt,
			ApprovalPolicy:         codexApprovalPolicy(input.PersistentConversation),
			Sandbox:                "danger-full-access",
			Ephemeral:              boolPointer(!input.PersistentConversation),
			PersistExtendedHistory: true,
		},
		Turn: codexadapter.TurnConfig{
			WorkingDirectory: input.WorkingDirectory.String(),
			Title:            codexTurnTitle(input.PersistentConversation),
			ApprovalPolicy:   codexApprovalPolicy(input.PersistentConversation),
			SandboxPolicy: map[string]any{
				"type":          "dangerFullAccess",
				"networkAccess": true,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("start codex chat session: %w", err)
	}

	state = &codexRuntimeSession{
		session:    session,
		lastTurnID: strings.TrimSpace(input.ResumeProviderTurnID),
	}

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
	providerItem catalogdomain.AgentProvider,
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

	assistantItems := make(map[string]*codexAssistantItemState)

	for event := range state.session.Events() {
		switch event.Type {
		case codexadapter.EventTypeTurnStarted:
			if event.Turn == nil || event.Turn.TurnID != turnID {
				continue
			}
			r.mu.Lock()
			state.lastTurnID = event.Turn.TurnID
			r.mu.Unlock()
			events <- newTaskMessageEvent(chatMessageTypeTaskStarted, map[string]any{
				"thread_id": event.Turn.ThreadID,
				"turn_id":   event.Turn.TurnID,
				"status":    event.Turn.Status,
			})
		case codexadapter.EventTypeOutputProduced:
			if event.Output == nil || event.Output.TurnID != turnID || event.Output.Text == "" {
				continue
			}
			if event.Output.Stream == "assistant" {
				for _, item := range mapCodexAssistantOutput(event.Output, assistantItems) {
					events <- item
				}
				continue
			}
			events <- newTaskMessageEvent(chatMessageTypeTaskProgress, map[string]any{
				"stream":   event.Output.Stream,
				"text":     event.Output.Text,
				"phase":    event.Output.Phase,
				"snapshot": event.Output.Snapshot,
			})
		case codexadapter.EventTypeToolCallRequested:
			if event.ToolCall == nil || event.ToolCall.TurnID != turnID {
				continue
			}
			events <- newTaskMessageEvent(chatMessageTypeTaskNotification, map[string]any{
				"tool":      event.ToolCall.Tool,
				"arguments": decodeRawJSON(event.ToolCall.Arguments),
			})
		case codexadapter.EventTypeApprovalRequested:
			if event.Approval == nil || event.Approval.TurnID != turnID {
				continue
			}
			events <- StreamEvent{
				Event: "interrupt_requested",
				Payload: RuntimeInterruptEvent{
					RequestID: event.Approval.RequestID.String(),
					Kind:      string(event.Approval.Kind),
					Options:   mapRuntimeInterruptOptions(event.Approval.Options),
					Payload:   cloneAnyMap(event.Approval.Payload),
				},
			}
		case codexadapter.EventTypeUserInputRequested:
			if event.UserInput == nil || event.UserInput.TurnID != turnID {
				continue
			}
			events <- StreamEvent{
				Event: "interrupt_requested",
				Payload: RuntimeInterruptEvent{
					RequestID: event.UserInput.RequestID.String(),
					Kind:      "user_input",
					Payload:   cloneAnyMap(event.UserInput.Payload),
				},
			}
		case codexadapter.EventTypeTokenUsageUpdated:
			if event.TokenUsage == nil || event.TokenUsage.TurnID != turnID {
				continue
			}

			usageInfo := provider.NewCodexCLIUsage(
				provider.CLIUsageTokens{
					InputTokens:       event.TokenUsage.TotalInputTokens,
					OutputTokens:      event.TokenUsage.TotalOutputTokens,
					TotalTokens:       event.TokenUsage.TotalTokens,
					CachedInputTokens: event.TokenUsage.TotalCachedInputTokens,
					ReasoningTokens:   event.TokenUsage.TotalReasoningTokens,
				},
				provider.CLIUsageTokens{
					InputTokens:       event.TokenUsage.LastInputTokens,
					OutputTokens:      event.TokenUsage.LastOutputTokens,
					TotalTokens:       event.TokenUsage.LastTokens,
					CachedInputTokens: event.TokenUsage.LastCachedInputTokens,
					ReasoningTokens:   event.TokenUsage.LastReasoningTokens,
				},
				event.TokenUsage.ModelContextWindow,
			)
			costUSD := resolveCLIUsageCostUSD(providerItem, usageInfo)
			if costUSD == nil {
				continue
			}

			r.mu.Lock()
			state.costUSD = costUSD
			r.mu.Unlock()
		case codexadapter.EventTypeTurnCompleted:
			if event.Turn == nil || event.Turn.TurnID != turnID {
				continue
			}

			r.mu.Lock()
			state.turnsUsed++
			turnsUsed := state.turnsUsed
			costUSD := cloneCostUSD(state.costUSD)
			r.mu.Unlock()

			events <- StreamEvent{
				Event: "done",
				Payload: donePayload{
					SessionID:      sessionID.String(),
					CostUSD:        costUSD,
					TurnsUsed:      turnsUsed,
					TurnsRemaining: remainingTurns(maxTurns, turnsUsed),
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

func mapCodexAssistantOutput(
	output *codexadapter.OutputEvent,
	items map[string]*codexAssistantItemState,
) []StreamEvent {
	if output == nil || strings.TrimSpace(output.Text) == "" {
		return nil
	}

	itemID := strings.TrimSpace(output.ItemID)
	if itemID == "" {
		return normalizeAssistantText(output.Text)
	}

	state := items[itemID]
	if state == nil {
		state = &codexAssistantItemState{}
		items[itemID] = state
	}
	combined := state.append(output.Text, output.Snapshot)

	if state.waitingOnSnapshot {
		if !output.Snapshot {
			return nil
		}
		delete(items, itemID)
		return normalizeAssistantText(combined)
	}

	if !state.emittedText && shouldDelayCodexAssistantEmission(combined) {
		state.waitingOnSnapshot = true
		if !output.Snapshot {
			return nil
		}
		delete(items, itemID)
		return normalizeAssistantText(combined)
	}

	if output.Snapshot {
		delete(items, itemID)
		if state.emittedText {
			return nil
		}
		return normalizeAssistantText(combined)
	}

	state.emittedText = true
	return []StreamEvent{newTextMessageEvent(output.Text)}
}

func shouldDelayCodexAssistantEmission(text string) bool {
	trimmed := strings.TrimLeft(text, " \t\r\n")
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "```")
}

func (s *codexAssistantItemState) append(text string, snapshot bool) string {
	if snapshot {
		existing := s.text.String()
		switch {
		case existing == "":
			return text
		case strings.HasPrefix(text, existing):
			return text
		default:
			return existing + text
		}
	}

	s.text.WriteString(text)
	return s.text.String()
}

func boolPointer(value bool) *bool {
	return &value
}

func codexApprovalPolicy(persistent bool) any {
	if persistent {
		return nil
	}
	return "never"
}

func codexTurnTitle(persistent bool) string {
	if persistent {
		return "OpenASE Project Conversation"
	}
	return "OpenASE Ephemeral Chat"
}

func (r *CodexRuntime) RespondInterrupt(
	ctx context.Context,
	sessionID SessionID,
	requestID string,
	kind string,
	decision string,
	answer map[string]any,
) error {
	r.mu.Lock()
	state := r.sessions[sessionID]
	r.mu.Unlock()
	if state == nil || state.session == nil {
		return fmt.Errorf("codex chat session %s not found", sessionID)
	}

	parsedRequestID, err := codexadapter.ParseRequestIDString(requestID)
	if err != nil {
		return err
	}

	switch kind {
	case "command_execution", "file_change":
		return state.session.RespondApproval(ctx, codexadapter.ApprovalRequest{
			RequestID: parsedRequestID,
			Kind:      codexadapter.ApprovalRequestKind(kind),
		}, decision)
	case "user_input":
		return state.session.RespondUserInput(ctx, codexadapter.UserInputRequest{
			RequestID: parsedRequestID,
		}, answer)
	default:
		return fmt.Errorf("unsupported interrupt kind %q", kind)
	}
}

func (r *CodexRuntime) SessionAnchor(sessionID SessionID) RuntimeSessionAnchor {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.sessions[sessionID]
	if state == nil || state.session == nil {
		return RuntimeSessionAnchor{}
	}
	return RuntimeSessionAnchor{
		ProviderThreadID: strings.TrimSpace(state.session.ThreadID()),
		LastTurnID:       strings.TrimSpace(state.lastTurnID),
	}
}

func mapRuntimeInterruptOptions(options []codexadapter.ApprovalOption) []RuntimeInterruptDecision {
	decisions := make([]RuntimeInterruptDecision, 0, len(options))
	for _, option := range options {
		decisions = append(decisions, RuntimeInterruptDecision{
			ID:    option.ID,
			Label: option.Label,
		})
	}
	return decisions
}

func cloneAnyMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	copied := make(map[string]any, len(value))
	for key, item := range value {
		copied[key] = item
	}
	return copied
}
