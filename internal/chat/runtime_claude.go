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

const claudeCodeResumeInterruptedTurnEnv = "CLAUDE_CODE_RESUME_INTERRUPTED_TURN"

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
	var maxBudgetUSDPointer *float64
	if maxBudgetUSD > 0 {
		maxBudgetUSDPointer = &maxBudgetUSD
	}
	environment := append(provider.AuthConfigEnvironment(input.Provider.AuthConfig), input.Environment...)
	if input.PersistentConversation && !hasProcessEnvironmentKey(environment, claudeCodeResumeInterruptedTurnEnv) {
		environment = append(environment, claudeCodeResumeInterruptedTurnEnv+"=1")
	}
	resumeSessionID := r.resolveResumeSessionID(input)

	return provider.NewClaudeCodeSessionSpec(
		command,
		buildBaseArgs(input.Provider.CliArgs, input.Provider.ModelName),
		workingDirectory,
		environment,
		nil,
		input.SystemPrompt,
		maxTurnsPointer,
		maxBudgetUSDPointer,
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
		r.nativeSessions.RegisterSessionID(sessionID, nativeSessionID)
		events <- newClaudeSessionAnchorEvent(nativeSessionID)
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
					r.nativeSessions.RegisterSessionID(sessionID, parsed)
					events <- newClaudeSessionAnchorEvent(parsed)
				}
			}

			for _, item := range mapClaudeEvent(sessionID, maxTurns, event) {
				if item.Event == "session_state" {
					if payload, ok := item.Payload.(runtimeSessionStatePayload); ok {
						r.nativeSessions.UpdateState(sessionID, payload.Status, payload.ActiveFlags)
					}
				}
				events <- item
			}
			if event.Kind == provider.ClaudeCodeEventKindResult {
				return
			}
		}
	}
}

func (r *ClaudeRuntime) SessionAnchor(sessionID SessionID) RuntimeSessionAnchor {
	if r == nil {
		return RuntimeSessionAnchor{}
	}

	state, ok := r.nativeSessions.Resolve(sessionID)
	if !ok || state.NativeSessionID == "" {
		return RuntimeSessionAnchor{}
	}

	return RuntimeSessionAnchor{
		ProviderThreadID:          state.NativeSessionID.String(),
		ProviderThreadStatus:      strings.TrimSpace(state.Status),
		ProviderThreadActiveFlags: append([]string(nil), state.ActiveFlags...),
		ProviderAnchorID:          state.NativeSessionID.String(),
		ProviderAnchorKind:        "session",
		ProviderTurnSupported:     false,
	}
}

func (r *ClaudeRuntime) resolveResumeSessionID(input RuntimeTurnInput) *provider.ClaudeCodeSessionID {
	if parsed, err := provider.ParseClaudeCodeSessionID(input.ResumeProviderThreadID); err == nil {
		return &parsed
	}
	state, ok := r.nativeSessions.Resolve(input.SessionID)
	if !ok || state.NativeSessionID == "" {
		return nil
	}
	cloned := state.NativeSessionID
	return &cloned
}

func hasProcessEnvironmentKey(entries []string, key string) bool {
	want := strings.TrimSpace(key)
	if want == "" {
		return false
	}

	for _, entry := range entries {
		name, _, found := strings.Cut(entry, "=")
		if found && strings.TrimSpace(name) == want {
			return true
		}
	}
	return false
}

func newClaudeSessionAnchorEvent(sessionID provider.ClaudeCodeSessionID) StreamEvent {
	return StreamEvent{
		Event: "session_anchor",
		Payload: RuntimeSessionAnchor{
			ProviderThreadID:      sessionID.String(),
			ProviderAnchorID:      sessionID.String(),
			ProviderAnchorKind:    "session",
			ProviderTurnSupported: false,
		},
	}
}

func mapClaudeEvent(sessionID SessionID, maxTurns int, event provider.ClaudeCodeEvent) []StreamEvent {
	switch event.Kind {
	case provider.ClaudeCodeEventKindSystem, provider.ClaudeCodeEventKindStream:
		if payload, ok := parseClaudeSessionStatePayload(event); ok {
			return []StreamEvent{{Event: "session_state", Payload: payload}}
		}
		return nil
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

func parseClaudeSessionStatePayload(event provider.ClaudeCodeEvent) (runtimeSessionStatePayload, bool) {
	subtype := strings.TrimSpace(event.Subtype)
	if subtype == "" {
		return runtimeSessionStatePayload{}, false
	}

	stateObject := decodeClaudeSessionStateObject(event)
	status := firstNonEmptyString(
		readStringMapKey(stateObject, "state"),
		readStringMapKey(stateObject, "session_state"),
		readStringMapKey(stateObject, "status"),
	)
	if status == "" && subtype == "requires_action" {
		status = "requires_action"
	}
	if status == "" {
		return runtimeSessionStatePayload{}, false
	}

	activeFlags := readStringSliceMapKey(stateObject, "active_flags")
	if len(activeFlags) == 0 {
		activeFlags = readStringSliceMapKey(stateObject, "activeFlags")
	}
	if len(activeFlags) == 0 && status == "requires_action" {
		activeFlags = []string{"requires_action"}
	}

	detail := firstNonEmptyString(
		readStringMapKey(stateObject, "detail"),
		readStringMapKey(stateObject, "message"),
		readStringMapKey(stateObject, "reason"),
		readStringMapKey(asMap(stateObject["requires_action"]), "type"),
	)
	if detail == "" && subtype == "requires_action" {
		detail = "Additional action is required before the session can continue."
	}

	return runtimeSessionStatePayload{
		Status:      status,
		ActiveFlags: append([]string(nil), activeFlags...),
		Detail:      detail,
		Raw:         cloneAnyMap(stateObject),
	}, true
}

func decodeClaudeSessionStateObject(event provider.ClaudeCodeEvent) map[string]any {
	if decoded := asMap(decodeRawJSON(event.Event)); decoded != nil {
		return decoded
	}
	if decoded := asMap(decodeRawJSON(event.Data)); decoded != nil {
		return decoded
	}
	raw := asMap(decodeRawJSON(event.Raw))
	if raw == nil {
		return map[string]any{}
	}
	if nested := asMap(raw["event"]); nested != nil {
		return nested
	}
	if nested := asMap(raw["data"]); nested != nil {
		return nested
	}
	return raw
}

func asMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func readStringMapKey(record map[string]any, key string) string {
	if record == nil {
		return ""
	}
	value, ok := record[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func readStringSliceMapKey(record map[string]any, key string) []string {
	if record == nil {
		return nil
	}
	raw, ok := record[key]
	if !ok {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		if typed, ok := raw.([]string); ok {
			return append([]string(nil), typed...)
		}
		return nil
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			continue
		}
		trimmed := strings.TrimSpace(text)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
	sessions map[SessionID]claudeRuntimeSessionState
}

type claudeRuntimeSessionState struct {
	NativeSessionID provider.ClaudeCodeSessionID
	Status          string
	ActiveFlags     []string
}

func (r *claudeSessionRegistry) RegisterSessionID(sessionID SessionID, native provider.ClaudeCodeSessionID) {
	if sessionID == "" || native == "" {
		return
	}

	r.mu.Lock()
	if r.sessions == nil {
		r.sessions = make(map[SessionID]claudeRuntimeSessionState)
	}
	state := r.sessions[sessionID]
	state.NativeSessionID = native
	r.sessions[sessionID] = state
	r.mu.Unlock()
}

func (r *claudeSessionRegistry) UpdateState(sessionID SessionID, status string, activeFlags []string) {
	if sessionID == "" {
		return
	}

	r.mu.Lock()
	if r.sessions == nil {
		r.sessions = make(map[SessionID]claudeRuntimeSessionState)
	}
	state := r.sessions[sessionID]
	state.Status = strings.TrimSpace(status)
	state.ActiveFlags = append([]string(nil), activeFlags...)
	r.sessions[sessionID] = state
	r.mu.Unlock()
}

func (r *claudeSessionRegistry) Resolve(sessionID SessionID) (claudeRuntimeSessionState, bool) {
	if sessionID == "" {
		return claudeRuntimeSessionState{}, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	state, ok := r.sessions[sessionID]
	if !ok {
		return claudeRuntimeSessionState{}, false
	}

	cloned := state
	cloned.ActiveFlags = append([]string(nil), state.ActiveFlags...)
	return cloned, true
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
