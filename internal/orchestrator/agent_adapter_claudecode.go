package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/claudecode"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

type claudeCodeAgentAdapter struct{}

func (claudeCodeAgentAdapter) Start(ctx context.Context, spec agentSessionStartSpec) (agentSession, error) {
	if spec.ProcessManager == nil {
		return nil, fmt.Errorf("claude code process manager must not be nil")
	}

	baseArgs := append([]string(nil), spec.Process.Args...)
	if normalizeAgentPermissionProfile(spec.PermissionProfile) == catalogdomain.AgentProviderPermissionProfileUnrestricted && !hasClaudePermissionBypassArg(baseArgs) {
		baseArgs = append(baseArgs, "--permission-mode", "bypassPermissions")
	}
	if trimmed := strings.TrimSpace(spec.Model); trimmed != "" && !hasClaudeModelArg(baseArgs) {
		baseArgs = append(baseArgs, "--model", trimmed)
	}

	sessionSpec, err := provider.NewClaudeCodeSessionSpec(
		spec.Process.Command,
		baseArgs,
		spec.Process.WorkingDirectory,
		spec.Process.Environment,
		nil,
		spec.DeveloperInstructions,
		nil,
		nil,
		nil,
		true,
	)
	if err != nil {
		return nil, err
	}

	session, err := claudecode.NewAdapter(spec.ProcessManager).Start(ctx, sessionSpec)
	if err != nil {
		return nil, err
	}
	return newClaudeCodeAgentSession(session), nil
}

func (claudeCodeAgentAdapter) Resume(ctx context.Context, spec agentSessionResumeSpec) (agentSession, error) {
	if spec.StartSpec.ProcessManager == nil {
		return nil, fmt.Errorf("claude code process manager must not be nil")
	}
	resumeID, err := provider.ParseClaudeCodeSessionID(spec.SessionID)
	if err != nil {
		return nil, err
	}

	baseArgs := append([]string(nil), spec.StartSpec.Process.Args...)
	if normalizeAgentPermissionProfile(spec.StartSpec.PermissionProfile) == catalogdomain.AgentProviderPermissionProfileUnrestricted && !hasClaudePermissionBypassArg(baseArgs) {
		baseArgs = append(baseArgs, "--permission-mode", "bypassPermissions")
	}
	if trimmed := strings.TrimSpace(spec.StartSpec.Model); trimmed != "" && !hasClaudeModelArg(baseArgs) {
		baseArgs = append(baseArgs, "--model", trimmed)
	}

	sessionSpec, err := provider.NewClaudeCodeSessionSpec(
		spec.StartSpec.Process.Command,
		baseArgs,
		spec.StartSpec.Process.WorkingDirectory,
		spec.StartSpec.Process.Environment,
		nil,
		spec.StartSpec.DeveloperInstructions,
		nil,
		nil,
		&resumeID,
		true,
	)
	if err != nil {
		return nil, err
	}

	session, err := claudecode.NewAdapter(spec.StartSpec.ProcessManager).Start(ctx, sessionSpec)
	if err != nil {
		return nil, err
	}
	return newClaudeCodeAgentSession(session), nil
}

func hasClaudeModelArg(args []string) bool {
	for index := 0; index < len(args); index++ {
		if args[index] == "--model" {
			return true
		}
		if strings.HasPrefix(args[index], "--model=") {
			return true
		}
	}
	return false
}

func hasClaudePermissionBypassArg(args []string) bool {
	for index := 0; index < len(args); index++ {
		if args[index] == "--dangerously-skip-permissions" {
			return true
		}
		if args[index] == "--permission-mode" && index+1 < len(args) && strings.EqualFold(strings.TrimSpace(args[index+1]), "bypassPermissions") {
			return true
		}
		if strings.EqualFold(strings.TrimSpace(args[index]), "--permission-mode=bypassPermissions") {
			return true
		}
	}
	return false
}

type claudeCodeAgentSession struct {
	session provider.ClaudeCodeSession
	events  chan agentEvent

	sessionMu sync.RWMutex
	sessionID string

	doneMu  sync.RWMutex
	doneErr error

	assistantItemID   string
	assistantText     string
	assistantSequence int

	taskItemID   string
	taskText     string
	taskCommand  string
	taskSequence int

	toolCalls     map[string]claudeToolCallState
	toolSequence  int
	rawEventCount int
}

type claudeToolCallState struct {
	Tool    string
	Command string
}

func newClaudeCodeAgentSession(session provider.ClaudeCodeSession) *claudeCodeAgentSession {
	wrapped := &claudeCodeAgentSession{
		session:   session,
		events:    make(chan agentEvent, 64),
		toolCalls: map[string]claudeToolCallState{},
	}
	if sessionID, ok := session.SessionID(); ok {
		wrapped.sessionID = sessionID.String()
	}
	go wrapped.bridge()
	return wrapped
}

func (s *claudeCodeAgentSession) SessionID() (string, bool) {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	return s.sessionID, s.sessionID != ""
}

func (s *claudeCodeAgentSession) Events() <-chan agentEvent {
	if s == nil {
		return nil
	}
	return s.events
}

func (s *claudeCodeAgentSession) SendPrompt(ctx context.Context, prompt string) (agentTurnStartResult, error) {
	if s == nil || s.session == nil {
		return agentTurnStartResult{}, fmt.Errorf("claude code session must not be nil")
	}
	turnInput, err := provider.NewClaudeCodeTurnInput(prompt)
	if err != nil {
		return agentTurnStartResult{}, err
	}
	if err := s.session.Send(ctx, turnInput); err != nil {
		return agentTurnStartResult{}, err
	}
	return agentTurnStartResult{}, nil
}

func (s *claudeCodeAgentSession) Stop(ctx context.Context) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("claude code session must not be nil")
	}
	return s.session.Close(ctx)
}

func (s *claudeCodeAgentSession) Err() error {
	s.doneMu.RLock()
	defer s.doneMu.RUnlock()
	return s.doneErr
}

func (s *claudeCodeAgentSession) Diagnostic() agentSessionDiagnostic {
	diagnostic := agentSessionDiagnostic{}
	if sessionID, ok := s.SessionID(); ok {
		diagnostic.SessionID = sessionID
	}
	if err := s.Err(); err != nil {
		diagnostic.Error = err.Error()
	}
	return diagnostic
}

func (s *claudeCodeAgentSession) bridge() {
	if s == nil || s.session == nil {
		return
	}
	defer close(s.events)

	eventCh := s.session.Events()
	errorCh := s.session.Errors()
	for eventCh != nil || errorCh != nil {
		select {
		case err, ok := <-errorCh:
			if !ok {
				errorCh = nil
				continue
			}
			s.setDoneErr(err)
		case event, ok := <-eventCh:
			if !ok {
				eventCh = nil
				continue
			}
			if trimmed := strings.TrimSpace(event.SessionID); trimmed != "" {
				s.setSessionID(trimmed)
			}
			for _, mapped := range s.mapEvent(event) {
				s.events <- mapped
			}
		}
	}
}

func (s *claudeCodeAgentSession) setSessionID(sessionID string) {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return
	}
	s.sessionMu.Lock()
	if s.sessionID == "" {
		s.sessionID = trimmed
	}
	s.sessionMu.Unlock()
}

func (s *claudeCodeAgentSession) setDoneErr(err error) {
	if err == nil {
		return
	}
	s.doneMu.Lock()
	s.doneErr = err
	s.doneMu.Unlock()
}

func (s *claudeCodeAgentSession) mapEvent(event provider.ClaudeCodeEvent) []agentEvent {
	switch event.Kind {
	case provider.ClaudeCodeEventKindAssistant:
		return s.mapAssistantEvent(event)
	case provider.ClaudeCodeEventKindUser:
		return s.mapUserEvent(event)
	case provider.ClaudeCodeEventKindTaskStart:
		parsed, ok := parseClaudeProtocolTaskStarted(event)
		if !ok {
			return s.rawTaskStatusEvent(event, "task_started", "Claude task started")
		}
		payload := claudeTaskStartedTracePayload(parsed)
		threadID := parsed.Envelope.SessionID
		turnID := parsed.Envelope.TurnID
		itemID := s.taskItemIDFor(event, nil)
		if itemID != "" {
			payload["item_id"] = itemID
		}
		return []agentEvent{
			{
				Type: agentEventTypeTaskStatus,
				TaskStatus: &agentTaskStatusEvent{
					ThreadID:   threadID,
					TurnID:     turnID,
					ItemID:     itemID,
					StatusType: catalogdomain.AgentTraceKindTaskStarted,
					Text:       claudeProtocolTaskStatusText(payload),
					Payload:    payload,
				},
			},
			{
				Type: agentEventTypeTurnStarted,
				Turn: &agentTurnEvent{
					ThreadID: threadID,
					TurnID:   turnID,
					Status:   "started",
				},
			},
		}
	case provider.ClaudeCodeEventKindTaskProgress:
		parsed, ok := parseClaudeProtocolTaskProgress(event)
		if !ok {
			return s.rawTaskStatusEvent(event, "task_progress", "Claude task progress")
		}
		payload := claudeTaskProgressTracePayload(parsed)
		itemID := s.taskItemIDFor(event, &parsed)
		if itemID != "" {
			payload["item_id"] = itemID
		}
		events := []agentEvent{{
			Type: agentEventTypeTaskStatus,
			TaskStatus: &agentTaskStatusEvent{
				ThreadID:   parsed.Envelope.SessionID,
				TurnID:     parsed.Envelope.TurnID,
				ItemID:     itemID,
				StatusType: catalogdomain.AgentTraceKindTaskProgress,
				Text:       claudeProtocolTaskStatusText(payload),
				Payload:    payload,
			},
		}}
		if diff := claudeTurnDiffEvent(
			parsed.Envelope.SessionID,
			parsed.Envelope.TurnID,
			parsed.Text,
		); diff != nil {
			events = append(events, agentEvent{
				Type: agentEventTypeTurnDiffUpdated,
				Diff: diff,
			})
		}
		return events
	case provider.ClaudeCodeEventKindTaskNotice:
		parsed, ok := parseClaudeProtocolTaskNotification(event)
		if !ok {
			return s.rawTaskStatusEvent(event, "task_notification", "Claude task notification")
		}
		payload := claudeTaskNotificationTracePayload(parsed)
		itemID := s.taskItemIDFor(event, nil)
		if itemID != "" {
			payload["item_id"] = itemID
		}
		return []agentEvent{{
			Type: agentEventTypeTaskStatus,
			TaskStatus: &agentTaskStatusEvent{
				ThreadID:   parsed.Envelope.SessionID,
				TurnID:     parsed.Envelope.TurnID,
				ItemID:     itemID,
				StatusType: catalogdomain.AgentTraceKindTaskNotification,
				Text:       claudeProtocolTaskStatusText(payload),
				Payload:    payload,
			},
		}}
	case provider.ClaudeCodeEventKindSystem, provider.ClaudeCodeEventKindStream:
		if parsed, ok := parseClaudeProtocolSessionStateChanged(event); ok {
			payload := claudeSessionStateTracePayload(parsed)
			return []agentEvent{{
				Type: agentEventTypeTaskStatus,
				TaskStatus: &agentTaskStatusEvent{
					ThreadID:   parsed.Envelope.SessionID,
					TurnID:     parsed.Envelope.TurnID,
					ItemID:     firstClaudeNonEmptyString(parsed.Envelope.EventUUID, s.syntheticRawItemID()),
					StatusType: catalogdomain.AgentTraceKindSessionState,
					Text:       claudeProtocolTaskStatusText(payload),
					Payload:    payload,
				},
			}}
		}
		return s.rawTaskStatusEvent(event, claudeRawStatusType(event), claudeRawStatusText(event))
	case provider.ClaudeCodeEventKindResult:
		events := make([]agentEvent, 0, 2)
		if usage := agentTokenUsageFromCLIUsage("", "", event.UsageInfo); usage != nil {
			events = append(events, agentEvent{
				Type:       agentEventTypeTokenUsageUpdated,
				TokenUsage: usage,
			})
		}
		completed := agentEvent{
			Type: agentEventTypeTurnCompleted,
			Turn: &agentTurnEvent{
				ThreadID: claudeProtocolEnvelopeFromEvent(event).SessionID,
				TurnID:   claudeProtocolEnvelopeFromEvent(event).TurnID,
				Status:   "completed",
			},
		}
		if event.IsError {
			message, additionalDetails := claudeTurnFailure(event)
			completed.Type = agentEventTypeTurnFailed
			completed.Turn.Status = "failed"
			completed.Turn.Error = &agentTurnError{
				Message:           message,
				AdditionalDetails: additionalDetails,
			}
			events = append([]agentEvent{{
				Type: agentEventTypeTaskStatus,
				TaskStatus: &agentTaskStatusEvent{
					ThreadID:   claudeProtocolEnvelopeFromEvent(event).SessionID,
					TurnID:     claudeProtocolEnvelopeFromEvent(event).TurnID,
					ItemID:     firstClaudeNonEmptyString(claudeProtocolEnvelopeFromEvent(event).EventUUID, s.syntheticRawItemID()),
					StatusType: catalogdomain.AgentTraceKindError,
					Text:       message,
					Payload:    claudeRawPayload(event),
				},
			}}, events...)
		}
		events = append(events, completed)
		return events
	case provider.ClaudeCodeEventKindRateLimit:
		if event.RateLimitInfo == nil {
			return nil
		}
		return []agentEvent{{
			Type:      agentEventTypeRateLimitUpdated,
			RateLimit: event.RateLimitInfo,
		}}
	default:
		return s.rawTaskStatusEvent(event, "unknown_event", "Claude unknown event")
	}
}

func (s *claudeCodeAgentSession) mapAssistantEvent(event provider.ClaudeCodeEvent) []agentEvent {
	parsed, ok := parseClaudeProtocolAssistantMessage(event)
	if !ok || len(parsed.Blocks) == 0 {
		return nil
	}

	threadID := parsed.Envelope.SessionID
	turnID := parsed.Envelope.TurnID
	events := make([]agentEvent, 0, len(parsed.Blocks)+1)
	texts := make([]string, 0, len(parsed.Blocks))
	for _, block := range parsed.Blocks {
		switch block.Kind {
		case claudeProtocolContentBlockText:
			text := strings.TrimSpace(block.Text)
			if text != "" {
				texts = append(texts, text)
			}
		case claudeProtocolContentBlockToolUse, claudeProtocolContentBlockServerToolUse, claudeProtocolContentBlockMCPToolUse:
			toolName := strings.TrimSpace(block.Name)
			if toolName == "" {
				toolName = string(block.Kind)
			}
			callID := s.toolCallIDFor(event, block)
			rawArguments := mustMarshalJSON(cloneClaudeMap(block.Input))
			events = append(events, agentEvent{
				Type: agentEventTypeToolCallRequested,
				ToolCall: &agentToolCallRequest{
					ThreadID:  threadID,
					TurnID:    turnID,
					CallID:    callID,
					Tool:      toolName,
					Arguments: rawArguments,
				},
			})
			s.rememberToolCall(callID, toolName, block.Input)
		}
	}

	text := strings.TrimSpace(strings.Join(texts, "\n\n"))
	if text != "" {
		itemID := s.assistantItemIDFor(event, text)
		events = append([]agentEvent{{
			Type: agentEventTypeOutputProduced,
			Output: &agentOutputEvent{
				ThreadID: threadID,
				TurnID:   turnID,
				ItemID:   itemID,
				Stream:   "assistant",
				Text:     text,
				Snapshot: true,
			},
		}}, events...)
	}

	if len(events) == 0 {
		return s.rawTaskStatusEvent(event, "assistant_event", "Claude assistant event")
	}
	return events
}

func (s *claudeCodeAgentSession) mapUserEvent(event provider.ClaudeCodeEvent) []agentEvent {
	parsed, ok := parseClaudeProtocolUserMessage(event)
	if !ok || len(parsed.Blocks) == 0 {
		return s.rawTaskStatusEvent(event, "user_event", "Claude user event")
	}

	threadID := parsed.Envelope.SessionID
	turnID := parsed.Envelope.TurnID
	events := make([]agentEvent, 0, len(parsed.Blocks))
	for _, block := range parsed.Blocks {
		if block.Kind != claudeProtocolContentBlockToolResult {
			continue
		}
		toolUseID := strings.TrimSpace(block.ToolUseID)
		text := strings.TrimSpace(extractClaudeToolResultText(block.Content))
		callState, ok := s.toolCallState(toolUseID)
		if ok && callState.Command != "" && text != "" {
			events = append(events, agentEvent{
				Type: agentEventTypeOutputProduced,
				Output: &agentOutputEvent{
					ThreadID: threadID,
					TurnID:   turnID,
					ItemID: firstClaudeNonEmptyString(toolUseID, s.taskItemIDFor(event, &claudeProtocolTaskProgress{
						Stream:  "command",
						Command: callState.Command,
						Text:    text,
					})),
					Stream:   "command",
					Command:  callState.Command,
					Text:     text,
					Snapshot: true,
				},
			})
			if diff := claudeTurnDiffEvent(threadID, turnID, text); diff != nil {
				events = append(events, agentEvent{
					Type: agentEventTypeTurnDiffUpdated,
					Diff: diff,
				})
			}
			continue
		}

		payload := claudeRawPayload(event)
		if toolUseID != "" {
			payload["tool_use_id"] = toolUseID
		}
		if ok {
			payload["tool"] = callState.Tool
			if callState.Command != "" {
				payload["command"] = callState.Command
			}
		}
		if text != "" {
			payload["text"] = text
		}
		if block.IsError {
			payload["is_error"] = true
		}
		events = append(events, agentEvent{
			Type: agentEventTypeTaskStatus,
			TaskStatus: &agentTaskStatusEvent{
				ThreadID:   threadID,
				TurnID:     turnID,
				ItemID:     firstClaudeNonEmptyString(toolUseID, claudeEventUUID(event)),
				StatusType: catalogdomain.AgentTraceKindTaskProgress,
				Text:       firstClaudeNonEmptyString(text, callState.Tool, "Claude tool result"),
				Payload:    payload,
			},
		})
	}

	if len(events) == 0 {
		return s.rawTaskStatusEvent(event, "user_event", "Claude user event")
	}
	return events
}

func (s *claudeCodeAgentSession) assistantItemIDFor(event provider.ClaudeCodeEvent, text string) string {
	if rawID := strings.TrimSpace(event.UUID); rawID != "" {
		s.assistantItemID = rawID
		s.assistantText = text
		return rawID
	}

	if s.assistantItemID != "" && claudeAssistantSnapshotContinues(s.assistantText, text) {
		s.assistantText = text
		return s.assistantItemID
	}

	s.assistantSequence++
	s.assistantItemID = fmt.Sprintf("claude-assistant-%d", s.assistantSequence)
	s.assistantText = text
	return s.assistantItemID
}

func (s *claudeCodeAgentSession) toolCallIDFor(event provider.ClaudeCodeEvent, block claudeProtocolContentBlock) string {
	if trimmed := firstClaudeNonEmptyString(block.ID, claudeReadString(block.Input, "id")); trimmed != "" {
		return trimmed
	}
	if rawID := claudeEventUUID(event); rawID != "" {
		return rawID
	}
	s.toolSequence++
	return fmt.Sprintf("claude-tool-%d", s.toolSequence)
}

func (s *claudeCodeAgentSession) rememberToolCall(callID string, toolName string, input map[string]any) {
	if strings.TrimSpace(callID) == "" {
		return
	}
	if s.toolCalls == nil {
		s.toolCalls = map[string]claudeToolCallState{}
	}
	s.toolCalls[callID] = claudeToolCallState{
		Tool:    strings.TrimSpace(toolName),
		Command: deriveClaudeToolUseSemantics(toolName, input).Command,
	}
}

func (s *claudeCodeAgentSession) toolCallState(callID string) (claudeToolCallState, bool) {
	if strings.TrimSpace(callID) == "" || len(s.toolCalls) == 0 {
		return claudeToolCallState{}, false
	}
	state, ok := s.toolCalls[callID]
	return state, ok
}

func (s *claudeCodeAgentSession) rawTaskStatusEvent(event provider.ClaudeCodeEvent, statusType string, text string) []agentEvent {
	envelope := claudeProtocolEnvelopeFromEvent(event)
	payload := envelope.RawPayload
	if len(payload) == 0 {
		return nil
	}
	itemID := envelope.EventUUID
	if itemID == "" {
		itemID = s.syntheticRawItemID()
	}
	return []agentEvent{{
		Type: agentEventTypeTaskStatus,
		TaskStatus: &agentTaskStatusEvent{
			ThreadID:   envelope.SessionID,
			TurnID:     envelope.TurnID,
			ItemID:     itemID,
			StatusType: strings.TrimSpace(statusType),
			Text:       strings.TrimSpace(text),
			Payload:    payload,
		},
	}}
}

func (s *claudeCodeAgentSession) taskItemIDFor(event provider.ClaudeCodeEvent, progress *claudeProtocolTaskProgress) string {
	if rawID := claudeEventUUID(event); rawID != "" {
		return rawID
	}
	if progress == nil {
		return ""
	}
	if progress.ToolUseID != "" {
		return progress.ToolUseID
	}
	if progress.Stream != "command" {
		return ""
	}
	text := progress.Text
	if text == "" {
		return ""
	}
	command := progress.Command
	if s.taskItemID != "" && command == s.taskCommand && claudeAssistantSnapshotContinues(s.taskText, text) {
		s.taskText = text
		return s.taskItemID
	}

	s.taskSequence++
	s.taskItemID = fmt.Sprintf("claude-task-%d", s.taskSequence)
	s.taskText = text
	s.taskCommand = command
	return s.taskItemID
}

func (s *claudeCodeAgentSession) syntheticRawItemID() string {
	s.rawEventCount++
	return fmt.Sprintf("claude-raw-%d", s.rawEventCount)
}

func claudeAssistantSnapshotContinues(previous string, next string) bool {
	trimmedPrevious := strings.TrimSpace(previous)
	trimmedNext := strings.TrimSpace(next)
	if trimmedPrevious == "" || trimmedNext == "" {
		return false
	}
	return strings.HasPrefix(trimmedNext, trimmedPrevious)
}

func claudeRawStatusType(event provider.ClaudeCodeEvent) string {
	switch event.Kind {
	case provider.ClaudeCodeEventKindSystem:
		return "system_event"
	case provider.ClaudeCodeEventKindStream:
		return "stream_event"
	case provider.ClaudeCodeEventKindUser:
		return "user_event"
	default:
		return "raw_event"
	}
}

func claudeRawStatusText(event provider.ClaudeCodeEvent) string {
	switch event.Kind {
	case provider.ClaudeCodeEventKindSystem:
		return "Claude system event"
	case provider.ClaudeCodeEventKindStream:
		return "Claude stream event"
	case provider.ClaudeCodeEventKindUser:
		return "Claude user event"
	default:
		return "Claude raw event"
	}
}

func extractClaudeToolResultText(content any) string {
	switch typed := content.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			record := claudeMap(item)
			if record == nil {
				if text, ok := item.(string); ok {
					if trimmed := strings.TrimSpace(text); trimmed != "" {
						items = append(items, trimmed)
					}
				}
				continue
			}
			if strings.TrimSpace(claudeReadString(record, "type")) == "text" {
				if trimmed := strings.TrimSpace(claudeReadString(record, "text")); trimmed != "" {
					items = append(items, trimmed)
				}
			}
		}
		return strings.TrimSpace(strings.Join(items, "\n\n"))
	case map[string]any:
		if strings.TrimSpace(claudeReadString(typed, "type")) == "text" {
			return strings.TrimSpace(claudeReadString(typed, "text"))
		}
	}
	return ""
}

func claudeTurnDiffEvent(threadID string, turnID string, text string) *agentTurnDiffEvent {
	diffText := strings.TrimSpace(text)
	if !looksLikeUnifiedDiff(diffText) {
		return nil
	}
	return &agentTurnDiffEvent{
		ThreadID: threadID,
		TurnID:   turnID,
		Diff:     diffText,
	}
}

func looksLikeUnifiedDiff(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.Contains(trimmed, "diff --git ") && strings.Contains(trimmed, "\n@@ ")
}
