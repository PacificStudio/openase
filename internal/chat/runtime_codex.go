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
	adapter        *codexadapter.Adapter
	secretResolver RuntimeEnvironmentResolver
	mu             sync.Mutex
	sessions       map[SessionID]*codexRuntimeSession
}

type codexRuntimeSession struct {
	session            *codexadapter.Session
	costUSD            *float64
	turnsUsed          int
	running            bool
	interruptRequested bool
	lastTurnID         string
	threadStatus       string
	threadActiveFlags  []string
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

type runtimeThreadStatusPayload struct {
	ThreadID    string   `json:"thread_id"`
	Status      string   `json:"status"`
	ActiveFlags []string `json:"active_flags,omitempty"`
}

type runtimeThreadCompactedPayload struct {
	ThreadID string `json:"thread_id"`
	TurnID   string `json:"turn_id"`
}

type runtimePlanStepPayload struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

type runtimePlanUpdatedPayload struct {
	ThreadID    string                   `json:"thread_id"`
	TurnID      string                   `json:"turn_id"`
	Explanation *string                  `json:"explanation,omitempty"`
	Plan        []runtimePlanStepPayload `json:"plan"`
}

type runtimeDiffUpdatedPayload struct {
	ThreadID string `json:"thread_id"`
	TurnID   string `json:"turn_id"`
	Diff     string `json:"diff"`
}

type runtimeReasoningUpdatedPayload struct {
	ThreadID     string `json:"thread_id"`
	TurnID       string `json:"turn_id"`
	ItemID       string `json:"item_id"`
	Kind         string `json:"kind"`
	Delta        string `json:"delta,omitempty"`
	SummaryIndex *int   `json:"summary_index,omitempty"`
	ContentIndex *int   `json:"content_index,omitempty"`
}

func NewCodexRuntime(adapter *codexadapter.Adapter) *CodexRuntime {
	if adapter == nil {
		return nil
	}

	return &CodexRuntime{adapter: adapter}
}

func (r *CodexRuntime) ConfigureSecretResolver(resolver RuntimeEnvironmentResolver) {
	if r == nil {
		return
	}
	r.secretResolver = resolver
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

func (r *CodexRuntime) EnsureSession(ctx context.Context, input RuntimeTurnInput) error {
	if !r.Supports(input.Provider) {
		return fmt.Errorf("%w: %s", ErrProviderUnsupported, input.Provider.AdapterType)
	}
	_, err := r.ensureSession(ctx, input)
	return err
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

func (r *CodexRuntime) InterruptTurn(
	ctx context.Context,
	sessionID SessionID,
) (RuntimeSessionAnchor, error) {
	r.mu.Lock()
	state := r.sessions[sessionID]
	if state == nil || state.session == nil || !state.running {
		r.mu.Unlock()
		return RuntimeSessionAnchor{}, fmt.Errorf("codex chat session %s is not running", sessionID)
	}
	anchor := RuntimeSessionAnchor{
		ProviderThreadID:          strings.TrimSpace(state.session.ThreadID()),
		LastTurnID:                strings.TrimSpace(state.lastTurnID),
		ProviderThreadStatus:      strings.TrimSpace(state.threadStatus),
		ProviderThreadActiveFlags: append([]string(nil), state.threadActiveFlags...),
		ProviderAnchorID:          strings.TrimSpace(state.session.ThreadID()),
		ProviderAnchorKind:        "thread",
		ProviderTurnSupported:     true,
	}
	state.interruptRequested = true
	delete(r.sessions, sessionID)
	session := state.session
	r.mu.Unlock()

	if ctx == nil {
		return RuntimeSessionAnchor{}, fmt.Errorf("context must not be nil")
	}
	stopCtx := ctx
	cancel := func() {}
	if _, ok := ctx.Deadline(); !ok {
		// Keep stop requests bounded even when the caller uses a long-lived request context.
		stopCtx, cancel = context.WithTimeout(ctx, 2*time.Second)
	}
	defer cancel()
	if err := session.Stop(stopCtx); err != nil {
		return RuntimeSessionAnchor{}, err
	}
	return anchor, nil
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

	environment, err := resolveRuntimeEnvironment(ctx, r.secretResolver, input)
	if err != nil {
		return nil, err
	}

	processSpec, err := provider.NewAgentCLIProcessSpec(
		command,
		buildCodexArgs(input.Provider.CliArgs),
		workingDirectory,
		environment,
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
			ReasoningEffort:        reasoningEffortValue(input.Provider.ReasoningEffort),
			ServiceName:            "openase",
			DeveloperInstructions:  input.SystemPrompt,
			ApprovalPolicy:         codexApprovalPolicy(input.Provider.PermissionProfile, input.PersistentConversation),
			Sandbox:                codexSandboxMode(input.Provider.PermissionProfile, input.PersistentConversation),
			Ephemeral:              boolPointer(!input.PersistentConversation),
			PersistExtendedHistory: true,
		},
		Turn: codexadapter.TurnConfig{
			WorkingDirectory: input.WorkingDirectory.String(),
			Title:            codexTurnTitle(input.PersistentConversation),
			ApprovalPolicy:   codexApprovalPolicy(input.Provider.PermissionProfile, input.PersistentConversation),
			SandboxPolicy:    codexSandboxPolicy(input.Provider.PermissionProfile, input.PersistentConversation),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("start codex chat session: %w", err)
	}

	state = &codexRuntimeSession{
		session:    session,
		lastTurnID: strings.TrimSpace(input.ResumeProviderTurnID),
	}
	if threadStatus := session.ThreadStatus(); threadStatus != nil {
		state.threadStatus = strings.TrimSpace(threadStatus.Status)
		state.threadActiveFlags = append([]string(nil), threadStatus.ActiveFlags...)
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
		state.interruptRequested = false
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
				"command":  event.Output.Command,
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
			payload := runtimeTokenUsagePayloadFromCLIUsage(usageInfo, costUSD)
			if payload == nil {
				continue
			}

			r.mu.Lock()
			state.costUSD = cloneCostUSD(payload.CostUSD)
			r.mu.Unlock()
			events <- StreamEvent{Event: "token_usage_updated", Payload: *payload}
		case codexadapter.EventTypeRateLimitUpdated:
			if event.RateLimit == nil {
				continue
			}
			events <- StreamEvent{
				Event: "rate_limit_updated",
				Payload: runtimeRateLimitPayload{
					RateLimit:  event.RateLimit,
					ObservedAt: time.Now().UTC(),
				},
			}
		case codexadapter.EventTypeThreadStarted:
			if event.Thread == nil {
				continue
			}
			r.mu.Lock()
			state.threadStatus = strings.TrimSpace(event.Thread.Status)
			state.threadActiveFlags = append([]string(nil), event.Thread.ActiveFlags...)
			r.mu.Unlock()
			events <- StreamEvent{
				Event: "thread_status",
				Payload: runtimeThreadStatusPayload{
					ThreadID:    event.Thread.ThreadID,
					Status:      event.Thread.Status,
					ActiveFlags: append([]string(nil), event.Thread.ActiveFlags...),
				},
			}
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
		case codexadapter.EventTypeTurnInterrupted:
			if event.Turn == nil || event.Turn.TurnID != turnID {
				continue
			}
			r.mu.Lock()
			interruptedByUser := state.interruptRequested
			r.mu.Unlock()
			if interruptedByUser {
				return
			}
			message := "codex chat turn interrupted"
			if event.Turn.Error != nil && event.Turn.Error.Message != "" {
				message = event.Turn.Error.Message
			}
			events <- StreamEvent{
				Event:   "interrupted",
				Payload: errorPayload{Message: message},
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
		case codexadapter.EventTypeThreadStatus:
			if event.ThreadStatus == nil {
				continue
			}
			r.mu.Lock()
			state.threadStatus = strings.TrimSpace(event.ThreadStatus.Status)
			state.threadActiveFlags = append([]string(nil), event.ThreadStatus.ActiveFlags...)
			r.mu.Unlock()
			events <- StreamEvent{
				Event: "thread_status",
				Payload: runtimeThreadStatusPayload{
					ThreadID:    event.ThreadStatus.ThreadID,
					Status:      event.ThreadStatus.Status,
					ActiveFlags: append([]string(nil), event.ThreadStatus.ActiveFlags...),
				},
			}
		case codexadapter.EventTypeThreadCompacted:
			if event.Compaction == nil || event.Compaction.TurnID != turnID {
				continue
			}
			events <- StreamEvent{
				Event: "thread_compacted",
				Payload: runtimeThreadCompactedPayload{
					ThreadID: event.Compaction.ThreadID,
					TurnID:   event.Compaction.TurnID,
				},
			}
		case codexadapter.EventTypeTurnPlanUpdated:
			if event.Plan == nil || event.Plan.TurnID != turnID {
				continue
			}
			plan := make([]runtimePlanStepPayload, 0, len(event.Plan.Plan))
			for _, item := range event.Plan.Plan {
				plan = append(plan, runtimePlanStepPayload{
					Step:   item.Step,
					Status: item.Status,
				})
			}
			events <- StreamEvent{
				Event: "plan_updated",
				Payload: runtimePlanUpdatedPayload{
					ThreadID:    event.Plan.ThreadID,
					TurnID:      event.Plan.TurnID,
					Explanation: event.Plan.Explanation,
					Plan:        plan,
				},
			}
		case codexadapter.EventTypeTurnDiffUpdated:
			if event.Diff == nil || event.Diff.TurnID != turnID {
				continue
			}
			events <- StreamEvent{
				Event: "diff_updated",
				Payload: runtimeDiffUpdatedPayload{
					ThreadID: event.Diff.ThreadID,
					TurnID:   event.Diff.TurnID,
					Diff:     event.Diff.Diff,
				},
			}
		case codexadapter.EventTypeReasoningUpdated:
			if event.Reasoning == nil || event.Reasoning.TurnID != turnID {
				continue
			}
			events <- StreamEvent{
				Event: "reasoning_updated",
				Payload: runtimeReasoningUpdatedPayload{
					ThreadID:     event.Reasoning.ThreadID,
					TurnID:       event.Reasoning.TurnID,
					ItemID:       event.Reasoning.ItemID,
					Kind:         string(event.Reasoning.Kind),
					Delta:        event.Reasoning.Delta,
					SummaryIndex: event.Reasoning.SummaryIndex,
					ContentIndex: event.Reasoning.ContentIndex,
				},
			}
		}
	}

	r.mu.Lock()
	interruptedByUser := state.interruptRequested
	r.mu.Unlock()
	if interruptedByUser {
		return
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
	emittedText := state.text.String()
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
			return normalizeAssistantSnapshotSupplement(combined, emittedText)
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

func normalizeAssistantSnapshotSupplement(snapshotText string, emittedText string) []StreamEvent {
	leadingText, payload, ok := extractStructuredAssistantPayload(snapshotText)
	if !ok {
		return nil
	}

	emittedComparable := comparableAssistantText(emittedText)
	leadingComparable := comparableAssistantText(leadingText)
	if emittedComparable == "" || leadingComparable == "" {
		return []StreamEvent{{Event: "message", Payload: payload}}
	}
	if leadingComparable == emittedComparable ||
		strings.HasPrefix(leadingComparable, emittedComparable) ||
		strings.HasPrefix(emittedComparable, leadingComparable) {
		return []StreamEvent{{Event: "message", Payload: payload}}
	}

	return []StreamEvent{{Event: "message", Payload: payload}}
}

func comparableAssistantText(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
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

func codexApprovalPolicy(
	profile catalogdomain.AgentProviderPermissionProfile,
	persistent bool,
) any {
	if !persistent {
		return "never"
	}
	profile = normalizeCodexPermissionProfile(profile)
	if profile == catalogdomain.AgentProviderPermissionProfileStandard {
		return "on-request"
	}
	return "never"
}

func codexSandboxMode(
	profile catalogdomain.AgentProviderPermissionProfile,
	persistent bool,
) string {
	if !persistent {
		return "danger-full-access"
	}
	profile = normalizeCodexPermissionProfile(profile)
	if profile == catalogdomain.AgentProviderPermissionProfileStandard {
		return "workspace-write"
	}
	return "danger-full-access"
}

func codexSandboxPolicy(
	profile catalogdomain.AgentProviderPermissionProfile,
	persistent bool,
) map[string]any {
	if !persistent {
		return map[string]any{
			"type":          "dangerFullAccess",
			"networkAccess": true,
		}
	}
	profile = normalizeCodexPermissionProfile(profile)
	if profile == catalogdomain.AgentProviderPermissionProfileStandard {
		return map[string]any{
			"type": "workspaceWrite",
		}
	}
	return map[string]any{
		"type":          "dangerFullAccess",
		"networkAccess": true,
	}
}

func normalizeCodexPermissionProfile(
	profile catalogdomain.AgentProviderPermissionProfile,
) catalogdomain.AgentProviderPermissionProfile {
	if !profile.IsValid() {
		return catalogdomain.DefaultAgentProviderPermissionProfile
	}
	return profile
}

func codexTurnTitle(persistent bool) string {
	if persistent {
		return "OpenASE Project Conversation"
	}
	return "OpenASE Ephemeral Chat"
}

func (r *CodexRuntime) RespondInterrupt(
	ctx context.Context,
	input RuntimeInterruptResponseInput,
) (TurnStream, error) {
	r.mu.Lock()
	state := r.sessions[input.SessionID]
	startBridge := false
	turnID := ""
	if state != nil && !state.running {
		turnID = strings.TrimSpace(state.lastTurnID)
		if turnID != "" {
			state.running = true
			startBridge = true
		}
	}
	r.mu.Unlock()
	if state == nil || state.session == nil {
		return TurnStream{}, fmt.Errorf("codex chat session %s not found", input.SessionID)
	}
	if startBridge && turnID == "" {
		return TurnStream{}, fmt.Errorf("codex chat session %s is missing the interrupted turn id", input.SessionID)
	}

	parsedRequestID, err := codexadapter.ParseRequestIDString(input.RequestID)
	if err != nil {
		if startBridge {
			r.mu.Lock()
			state.running = false
			r.mu.Unlock()
		}
		return TurnStream{}, err
	}

	switch input.Kind {
	case "command_execution", "file_change":
		err = state.session.RespondApproval(ctx, codexadapter.ApprovalRequest{
			RequestID: parsedRequestID,
			Kind:      codexadapter.ApprovalRequestKind(input.Kind),
		}, input.Decision)
	case "user_input":
		err = state.session.RespondUserInput(ctx, codexadapter.UserInputRequest{
			RequestID: parsedRequestID,
		}, input.Answer)
	default:
		err = fmt.Errorf("unsupported interrupt kind %q", input.Kind)
	}
	if err != nil {
		if startBridge {
			r.mu.Lock()
			state.running = false
			r.mu.Unlock()
		}
		return TurnStream{}, err
	}
	if !startBridge {
		return TurnStream{}, nil
	}

	events := make(chan StreamEvent, 64)
	go r.bridgeTurn(input.SessionID, input.Provider, 0, turnID, state, events)
	return TurnStream{Events: events}, nil
}

func (r *CodexRuntime) SessionAnchor(sessionID SessionID) RuntimeSessionAnchor {
	r.mu.Lock()
	defer r.mu.Unlock()

	state := r.sessions[sessionID]
	if state == nil || state.session == nil {
		return RuntimeSessionAnchor{}
	}
	return RuntimeSessionAnchor{
		ProviderThreadID:          strings.TrimSpace(state.session.ThreadID()),
		LastTurnID:                strings.TrimSpace(state.lastTurnID),
		ProviderThreadStatus:      strings.TrimSpace(state.threadStatus),
		ProviderThreadActiveFlags: append([]string(nil), state.threadActiveFlags...),
		ProviderAnchorID:          strings.TrimSpace(state.session.ThreadID()),
		ProviderAnchorKind:        "thread",
		ProviderTurnSupported:     true,
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
