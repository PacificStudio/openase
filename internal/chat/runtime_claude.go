package chat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

type claudeRuntimeToolCallState struct {
	Tool    string
	Command string
}

type claudeRuntimeStreamState struct {
	toolCalls map[string]claudeRuntimeToolCallState
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

	return r.startSessionTurn(ctx, input)
}

func (r *ClaudeRuntime) RespondInterrupt(
	ctx context.Context,
	input RuntimeInterruptResponseInput,
) (TurnStream, error) {
	if !r.Supports(input.Provider) {
		return TurnStream{}, fmt.Errorf("%w: %s", ErrProviderUnsupported, input.Provider.AdapterType)
	}

	message, err := buildClaudeInterruptResponsePrompt(input)
	if err != nil {
		return TurnStream{}, err
	}

	return r.startSessionTurn(ctx, RuntimeTurnInput{
		SessionID:              input.SessionID,
		Provider:               input.Provider,
		Message:                message,
		WorkingDirectory:       input.WorkingDirectory,
		Environment:            append([]string(nil), input.Environment...),
		ResumeProviderThreadID: input.ResumeProviderThreadID,
		ResumeProviderTurnID:   input.ResumeProviderTurnID,
		PersistentConversation: input.PersistentConversation,
	})
}

func (r *ClaudeRuntime) startSessionTurn(ctx context.Context, input RuntimeTurnInput) (TurnStream, error) {
	if err := ctx.Err(); err != nil {
		return TurnStream{}, err
	}

	sessionSpec, err := r.buildSessionSpec(input)
	if err != nil {
		return TurnStream{}, err
	}

	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
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
		buildClaudeArgs(input.Provider.CliArgs, input.Provider.ModelName, input.Provider.PermissionProfile),
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

func buildClaudeArgs(
	cliArgs []string,
	modelName string,
	profile catalogdomain.AgentProviderPermissionProfile,
) []string {
	args := buildBaseArgs(cliArgs, modelName)
	if normalizeRuntimePermissionProfile(profile) == catalogdomain.AgentProviderPermissionProfileUnrestricted &&
		!hasClaudePermissionBypassArg(args) {
		args = append(args, "--permission-mode", "bypassPermissions")
	}
	return args
}

func hasClaudePermissionBypassArg(args []string) bool {
	for index := 0; index < len(args); index++ {
		if args[index] == "--dangerously-skip-permissions" {
			return true
		}
		if args[index] == "--permission-mode" && index+1 < len(args) &&
			strings.EqualFold(strings.TrimSpace(args[index+1]), "bypassPermissions") {
			return true
		}
		if strings.EqualFold(strings.TrimSpace(args[index]), "--permission-mode=bypassPermissions") {
			return true
		}
	}
	return false
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
	emittedInterrupts := map[string]struct{}{}
	streamState := &claudeRuntimeStreamState{}

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

			for _, item := range mapClaudeEvent(sessionID, maxTurns, event, streamState) {
				if item.Event == "interrupt_requested" {
					payload, ok := item.Payload.(RuntimeInterruptEvent)
					if !ok || strings.TrimSpace(payload.RequestID) == "" {
						continue
					}
					if _, duplicate := emittedInterrupts[payload.RequestID]; duplicate {
						continue
					}
					emittedInterrupts[payload.RequestID] = struct{}{}
				}
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

func mapClaudeEvent(
	sessionID SessionID,
	maxTurns int,
	event provider.ClaudeCodeEvent,
	state *claudeRuntimeStreamState,
) []StreamEvent {
	switch event.Kind {
	case provider.ClaudeCodeEventKindSystem, provider.ClaudeCodeEventKindStream:
		if payload, ok := parseClaudeSessionStatePayload(event); ok {
			events := []StreamEvent{{Event: "session_state", Payload: payload}}
			if interrupt, ok := parseClaudeInterruptEvent(event, payload); ok {
				events = append(events, StreamEvent{Event: "interrupt_requested", Payload: interrupt})
			}
			return events
		}
		return nil
	case provider.ClaudeCodeEventKindAssistant:
		return mapClaudeAssistantEvent(event, state)
	case provider.ClaudeCodeEventKindUser:
		return mapClaudeUserEvent(event, state)
	case provider.ClaudeCodeEventKindTaskStart:
		return []StreamEvent{newTaskMessageEvent(chatMessageTypeTaskStarted, decodeRawJSON(event.Raw))}
	case provider.ClaudeCodeEventKindTaskProgress:
		return []StreamEvent{newTaskMessageEvent(chatMessageTypeTaskProgress, decodeRawJSON(event.Raw))}
	case provider.ClaudeCodeEventKindTaskNotice:
		return []StreamEvent{newTaskMessageEvent(chatMessageTypeTaskNotification, decodeRawJSON(event.Raw))}
	case provider.ClaudeCodeEventKindRateLimit:
		if event.RateLimitInfo == nil {
			return nil
		}
		return []StreamEvent{{
			Event: "rate_limit_updated",
			Payload: runtimeRateLimitPayload{
				RateLimit:  event.RateLimitInfo,
				ObservedAt: time.Now().UTC(),
			},
		}}
	case provider.ClaudeCodeEventKindUnknown:
		payload := map[string]any{"type": event.UnknownType}
		if data := decodeRawJSON(event.Raw); data != nil {
			payload["raw"] = data
		}
		return []StreamEvent{{Event: "message", Payload: payload}}
	case provider.ClaudeCodeEventKindResult:
		if isClaudeResultError(event) {
			message, _ := provider.ClaudeCodeTurnFailure(event)
			return []StreamEvent{{Event: "error", Payload: errorPayload{Message: message}}}
		}
		costUSD := cloneCostUSD(event.TotalCostUSD)
		if event.UsageInfo != nil {
			costUSD = cloneCostUSD(event.UsageInfo.CostUSD)
		}
		events := make([]StreamEvent, 0, 2)
		if payload := runtimeTokenUsagePayloadFromCLIUsage(event.UsageInfo, costUSD); payload != nil {
			events = append(events, StreamEvent{Event: "token_usage_updated", Payload: *payload})
		}
		events = append(events, StreamEvent{
			Event: "done",
			Payload: donePayload{
				SessionID:      sessionID.String(),
				CostUSD:        costUSD,
				TurnsUsed:      event.NumTurns,
				TurnsRemaining: remainingTurns(maxTurns, event.NumTurns),
			},
		})
		return events
	default:
		return nil
	}
}

func mapClaudeAssistantEvent(
	event provider.ClaudeCodeEvent,
	state *claudeRuntimeStreamState,
) []StreamEvent {
	blocks, ok := provider.ParseClaudeCodeMessageBlocks(event.Message)
	if !ok || len(blocks) == 0 {
		return nil
	}

	threadID := provider.ClaudeCodeEventSessionID(event)
	turnID := provider.ClaudeCodeEventTurnID(event)
	texts := make([]string, 0, len(blocks))
	events := make([]StreamEvent, 0, len(blocks)+1)
	for _, block := range blocks {
		switch block.Kind {
		case provider.ClaudeCodeContentBlockKindText:
			text := strings.TrimSpace(block.Text)
			if text != "" {
				texts = append(texts, text)
			}
		case provider.ClaudeCodeContentBlockKindToolUse,
			provider.ClaudeCodeContentBlockKindServerToolUse,
			provider.ClaudeCodeContentBlockKindMCPToolUse:
			callID := strings.TrimSpace(block.ID)
			if callID == "" {
				callID = provider.ClaudeCodeEventUUID(event)
			}
			if state != nil {
				state.rememberToolCall(callID, block.Name, block.Input)
			}
			events = append(events, newTaskMessageEvent(chatMessageTypeTaskNotification, map[string]any{
				"tool":       firstNonEmptyString(strings.TrimSpace(block.Name), string(block.Kind)),
				"arguments":  cloneAnyMap(block.Input),
				"call_id":    callID,
				"provider":   "claude",
				"thread_id":  threadID,
				"turn_id":    turnID,
				"item_id":    provider.ClaudeCodeEventUUID(event),
				"session_id": provider.ClaudeCodeEventSessionID(event),
			}))
		}
	}

	text := strings.TrimSpace(strings.Join(texts, "\n\n"))
	if text != "" {
		return append(normalizeAssistantText(text), events...)
	}
	return events
}

func mapClaudeUserEvent(
	event provider.ClaudeCodeEvent,
	state *claudeRuntimeStreamState,
) []StreamEvent {
	blocks, ok := provider.ParseClaudeCodeMessageBlocks(event.Message)
	if !ok || len(blocks) == 0 {
		return []StreamEvent{newTaskMessageEvent(chatMessageTypeTaskProgress, provider.ClaudeCodeRawPayload(event))}
	}

	threadID := provider.ClaudeCodeEventSessionID(event)
	turnID := provider.ClaudeCodeEventTurnID(event)
	events := make([]StreamEvent, 0, len(blocks)+1)
	for _, block := range blocks {
		if block.Kind != provider.ClaudeCodeContentBlockKindToolResult {
			continue
		}

		toolUseID := strings.TrimSpace(block.ToolUseID)
		text := strings.TrimSpace(provider.ExtractClaudeCodeToolResultText(block.Content))
		callState, hasCallState := claudeRuntimeToolCallState{}, false
		if state != nil {
			callState, hasCallState = state.toolCall(toolUseID)
		}
		if hasCallState && strings.TrimSpace(callState.Command) != "" && text != "" {
			events = append(events, newTaskMessageEvent(chatMessageTypeTaskProgress, map[string]any{
				"stream":      "command",
				"command":     callState.Command,
				"text":        text,
				"snapshot":    true,
				"tool_use_id": toolUseID,
				"tool":        callState.Tool,
				"provider":    "claude",
				"thread_id":   threadID,
				"turn_id":     turnID,
				"is_error":    block.IsError,
			}))
			if looksLikeUnifiedDiff(text) && strings.TrimSpace(turnID) != "" {
				events = append(events, StreamEvent{
					Event: "diff_updated",
					Payload: runtimeDiffUpdatedPayload{
						ThreadID: threadID,
						TurnID:   turnID,
						Diff:     strings.TrimSpace(text),
					},
				})
			}
			continue
		}

		raw := provider.ClaudeCodeRawPayload(event)
		if toolUseID != "" {
			raw["tool_use_id"] = toolUseID
		}
		if hasCallState {
			raw["tool"] = callState.Tool
			if callState.Command != "" {
				raw["command"] = callState.Command
			}
		}
		if text != "" {
			raw["text"] = text
		}
		if block.IsError {
			raw["is_error"] = true
		}
		events = append(events, newTaskMessageEvent(chatMessageTypeTaskProgress, raw))
	}

	if len(events) == 0 {
		return []StreamEvent{newTaskMessageEvent(chatMessageTypeTaskProgress, provider.ClaudeCodeRawPayload(event))}
	}
	return events
}

func isClaudeResultError(event provider.ClaudeCodeEvent) bool {
	if event.IsError {
		return true
	}
	subtype := strings.TrimSpace(event.Subtype)
	return subtype == "error" || strings.HasPrefix(subtype, "error_")
}

func (s *claudeRuntimeStreamState) rememberToolCall(callID string, toolName string, input map[string]any) {
	if strings.TrimSpace(callID) == "" {
		return
	}
	if s.toolCalls == nil {
		s.toolCalls = map[string]claudeRuntimeToolCallState{}
	}
	semantics := provider.DeriveClaudeCodeToolUseSemantics(toolName, input)
	s.toolCalls[callID] = claudeRuntimeToolCallState{
		Tool:    strings.TrimSpace(toolName),
		Command: strings.TrimSpace(semantics.Command),
	}
}

func (s *claudeRuntimeStreamState) toolCall(callID string) (claudeRuntimeToolCallState, bool) {
	if s == nil || strings.TrimSpace(callID) == "" || len(s.toolCalls) == 0 {
		return claudeRuntimeToolCallState{}, false
	}
	state, ok := s.toolCalls[callID]
	return state, ok
}

func looksLikeUnifiedDiff(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.Contains(trimmed, "diff --git ") && strings.Contains(trimmed, "\n@@ ")
}

func parseClaudeInterruptEvent(
	event provider.ClaudeCodeEvent,
	state runtimeSessionStatePayload,
) (RuntimeInterruptEvent, bool) {
	if strings.TrimSpace(state.Status) != "requires_action" {
		return RuntimeInterruptEvent{}, false
	}

	stateObject := decodeClaudeSessionStateObject(event)
	request := asMap(stateObject["requires_action"])
	if request == nil {
		request = stateObject
	}

	requestID := firstNonEmptyString(
		readStringMapKey(request, "request_id"),
		readStringMapKey(request, "requestId"),
		readStringMapKey(request, "id"),
		readStringMapKey(stateObject, "request_id"),
		readStringMapKey(stateObject, "requestId"),
		readStringMapKey(stateObject, "id"),
	)
	if requestID == "" {
		requestID = "claude-requires-action-" + hashClaudeInterruptPayload(request)
	}

	kind := mapClaudeInterruptKind(firstNonEmptyString(
		readStringMapKey(request, "kind"),
		readStringMapKey(request, "type"),
		readStringMapKey(stateObject, "kind"),
		readStringMapKey(stateObject, "type"),
	))

	payload := cloneAnyMap(request)
	if len(payload) == 0 {
		payload = cloneAnyMap(state.Raw)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	if strings.TrimSpace(state.Detail) != "" {
		payload["detail"] = state.Detail
	}
	payload["session_state"] = strings.TrimSpace(state.Status)

	return RuntimeInterruptEvent{
		RequestID: requestID,
		Kind:      kind,
		Options:   parseClaudeInterruptOptions(request),
		Payload:   payload,
	}, true
}

func mapClaudeInterruptKind(raw string) string {
	switch strings.TrimSpace(raw) {
	case "command_execution", "command_execution_approval", "command_approval", "approval":
		return "command_execution"
	case "file_change", "file_change_approval":
		return "file_change"
	default:
		return "user_input"
	}
}

func parseClaudeInterruptOptions(record map[string]any) []RuntimeInterruptDecision {
	if record == nil {
		return nil
	}
	raw, ok := record["options"]
	if !ok {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	decisions := make([]RuntimeInterruptDecision, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case map[string]any:
			id := firstNonEmptyString(
				readStringMapKey(typed, "id"),
				readStringMapKey(typed, "value"),
				readStringMapKey(typed, "key"),
			)
			label := firstNonEmptyString(
				readStringMapKey(typed, "label"),
				readStringMapKey(typed, "title"),
				id,
			)
			if id != "" {
				decisions = append(decisions, RuntimeInterruptDecision{ID: id, Label: label})
			}
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				decisions = append(decisions, RuntimeInterruptDecision{ID: trimmed, Label: trimmed})
			}
		}
	}
	return decisions
}

func hashClaudeInterruptPayload(payload map[string]any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "unknown"
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:8])
}

func buildClaudeInterruptResponsePrompt(input RuntimeInterruptResponseInput) (string, error) {
	decision := strings.TrimSpace(input.Decision)
	if decision == "" && len(input.Answer) == 0 {
		return "", fmt.Errorf("claude interrupt response must include a decision or answer")
	}

	responsePayload := map[string]any{
		"request_id": input.RequestID,
		"kind":       input.Kind,
	}
	if decision != "" {
		responsePayload["decision"] = decision
	}
	if len(input.Answer) > 0 {
		responsePayload["answer"] = cloneAnyMap(input.Answer)
	}
	if nested := cloneAnyMap(input.Payload); len(nested) > 0 {
		responsePayload["pending_action"] = nested
	}

	encoded, err := json.MarshalIndent(responsePayload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode claude interrupt response: %w", err)
	}

	return strings.TrimSpace(strings.Join([]string{
		"OpenASE interrupt response.",
		"Treat this as the user's response to the currently pending Claude requires_action state.",
		"Resume the interrupted session, apply the response below, and continue without restating prior conversation history.",
		string(encoded),
	}, "\n\n")), nil
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
