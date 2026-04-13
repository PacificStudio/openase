package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
)

type codexAgentAdapter struct{}

func (codexAgentAdapter) Start(ctx context.Context, spec agentSessionStartSpec) (agentSession, error) {
	if spec.ProcessManager == nil {
		return nil, fmt.Errorf("codex process manager must not be nil")
	}

	adapter, err := codex.NewAdapter(codex.AdapterOptions{ProcessManager: spec.ProcessManager})
	if err != nil {
		return nil, fmt.Errorf("construct codex adapter: %w", err)
	}

	session, err := adapter.Start(ctx, codex.StartRequest{
		Process: spec.Process,
		Initialize: codex.InitializeParams{
			ClientName:    "openase",
			ClientVersion: "dev",
			ClientTitle:   "OpenASE",
		},
		Thread: codex.ThreadStartParams{
			WorkingDirectory:       spec.WorkingDirectory,
			Model:                  spec.Model,
			ReasoningEffort:        reasoningEffortValue(spec.ReasoningEffort),
			ServiceName:            "openase",
			DeveloperInstructions:  spec.DeveloperInstructions,
			ApprovalPolicy:         codexApprovalPolicy(spec.PermissionProfile),
			Sandbox:                codexSandboxMode(spec.PermissionProfile),
			PersistExtendedHistory: true,
		},
		Turn: codex.TurnConfig{
			WorkingDirectory: spec.WorkingDirectory,
			Title:            spec.TurnTitle,
			ApprovalPolicy:   codexApprovalPolicy(spec.PermissionProfile),
			SandboxPolicy:    codexSandboxPolicy(spec.PermissionProfile),
		},
	})
	if err != nil {
		return nil, err
	}

	return newCodexAgentSession(session), nil
}

func (codexAgentAdapter) Resume(_ context.Context, spec agentSessionResumeSpec) (agentSession, error) {
	return nil, unsupportedAgentAdapter{reason: "Codex app-server sessions cannot be resumed across orchestrator processes"}.unsupportedError("resume")
}

func codexApprovalPolicy(profile catalogdomain.AgentProviderPermissionProfile) string {
	profile = normalizeAgentPermissionProfile(profile)
	if profile == catalogdomain.AgentProviderPermissionProfileStandard {
		return "on-request"
	}
	return "never"
}

func codexSandboxMode(profile catalogdomain.AgentProviderPermissionProfile) string {
	profile = normalizeAgentPermissionProfile(profile)
	if profile == catalogdomain.AgentProviderPermissionProfileStandard {
		return "workspace-write"
	}
	return "danger-full-access"
}

func codexSandboxPolicy(profile catalogdomain.AgentProviderPermissionProfile) map[string]any {
	profile = normalizeAgentPermissionProfile(profile)
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

func normalizeAgentPermissionProfile(
	profile catalogdomain.AgentProviderPermissionProfile,
) catalogdomain.AgentProviderPermissionProfile {
	if !profile.IsValid() {
		return catalogdomain.DefaultAgentProviderPermissionProfile
	}
	return profile
}

type codexAgentSession struct {
	session *codex.Session
	events  chan agentEvent
}

func newCodexAgentSession(session *codex.Session) *codexAgentSession {
	wrapped := &codexAgentSession{
		session: session,
		events:  make(chan agentEvent, 32),
	}
	go wrapped.bridge()
	return wrapped
}

func (s *codexAgentSession) SessionID() (string, bool) {
	if s == nil || s.session == nil {
		return "", false
	}
	threadID := s.session.ThreadID()
	return threadID, threadID != ""
}

func (s *codexAgentSession) Events() <-chan agentEvent {
	if s == nil {
		return nil
	}
	return s.events
}

func (s *codexAgentSession) SendPrompt(ctx context.Context, prompt string) (agentTurnStartResult, error) {
	if s == nil || s.session == nil {
		return agentTurnStartResult{}, fmt.Errorf("codex session must not be nil")
	}
	result, err := s.session.SendPrompt(ctx, prompt)
	if err != nil {
		return agentTurnStartResult{}, err
	}
	return agentTurnStartResult{TurnID: result.TurnID}, nil
}

func (s *codexAgentSession) Stop(ctx context.Context) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("codex session must not be nil")
	}
	return s.session.Stop(ctx)
}

func (s *codexAgentSession) Err() error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Err()
}

func (s *codexAgentSession) Diagnostic() agentSessionDiagnostic {
	if s == nil || s.session == nil {
		return agentSessionDiagnostic{}
	}
	diagnostic := s.session.Diagnostic()
	return agentSessionDiagnostic{
		PID:       diagnostic.PID,
		SessionID: diagnostic.ThreadID,
		Error:     diagnostic.Error,
		Stderr:    diagnostic.Stderr,
	}
}

func (s *codexAgentSession) bridge() {
	if s == nil || s.session == nil {
		return
	}
	defer close(s.events)
	for event := range s.session.Events() {
		mapped, ok := mapCodexAgentEvent(event)
		if !ok {
			continue
		}
		if event.Type == codex.EventTypeRateLimitUpdated {
			observedAt := time.Now().UTC()
			mapped.ObservedAt = &observedAt
		}
		s.events <- mapped
	}
}

func mapCodexAgentEvent(event codex.Event) (agentEvent, bool) {
	switch event.Type {
	case codex.EventTypeToolCallRequested:
		if event.ToolCall == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type: agentEventTypeToolCallRequested,
			ToolCall: &agentToolCallRequest{
				RequestID: event.ToolCall.RequestID.String(),
				ThreadID:  event.ToolCall.ThreadID,
				TurnID:    event.ToolCall.TurnID,
				CallID:    event.ToolCall.CallID,
				Tool:      event.ToolCall.Tool,
				Arguments: append(json.RawMessage(nil), event.ToolCall.Arguments...),
			},
			Raw: rawCodexProviderEvent(event),
		}, true
	case codex.EventTypeApprovalRequested:
		if event.Approval == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type: agentEventTypeApprovalRequested,
			Approval: &agentApprovalRequest{
				RequestID: event.Approval.RequestID.String(),
				ThreadID:  event.Approval.ThreadID,
				TurnID:    event.Approval.TurnID,
				Kind:      string(event.Approval.Kind),
				Options:   mapCodexApprovalOptions(event.Approval.Options),
				Payload:   cloneCodexPayload(event.Approval.Payload),
			},
			Raw: rawCodexProviderEvent(event),
		}, true
	case codex.EventTypeUserInputRequested:
		if event.UserInput == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type: agentEventTypeUserInputRequested,
			UserInput: &agentUserInputRequest{
				RequestID: event.UserInput.RequestID.String(),
				ThreadID:  event.UserInput.ThreadID,
				TurnID:    event.UserInput.TurnID,
				Payload:   cloneCodexPayload(event.UserInput.Payload),
			},
			Raw: rawCodexProviderEvent(event),
		}, true
	case codex.EventTypeItemStarted:
		if event.Item == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type: agentEventTypeItemStarted,
			Item: &agentItemStartedEvent{
				ThreadID: event.Item.ThreadID,
				TurnID:   event.Item.TurnID,
				ItemID:   event.Item.ItemID,
				ItemType: event.Item.ItemType,
				Phase:    event.Item.Phase,
				Command:  event.Item.Command,
				Text:     event.Item.Text,
			},
			Raw: rawCodexProviderEvent(event),
		}, true
	case codex.EventTypeTokenUsageUpdated:
		if event.TokenUsage == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type: agentEventTypeTokenUsageUpdated,
			TokenUsage: &agentTokenUsageEvent{
				ThreadID:           event.TokenUsage.ThreadID,
				TurnID:             event.TokenUsage.TurnID,
				TotalInputTokens:   event.TokenUsage.TotalInputTokens,
				TotalOutputTokens:  event.TokenUsage.TotalOutputTokens,
				LastInputTokens:    event.TokenUsage.LastInputTokens,
				LastOutputTokens:   event.TokenUsage.LastOutputTokens,
				TotalTokens:        event.TokenUsage.TotalTokens,
				LastTokens:         event.TokenUsage.LastTokens,
				ModelContextWindow: event.TokenUsage.ModelContextWindow,
			},
			Raw: rawCodexProviderEvent(event),
		}, true
	case codex.EventTypeRateLimitUpdated:
		if event.RateLimit == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type:      agentEventTypeRateLimitUpdated,
			RateLimit: event.RateLimit,
			Raw:       rawCodexProviderEvent(event),
		}, true
	case codex.EventTypeOutputProduced:
		if event.Output == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type: agentEventTypeOutputProduced,
			Output: &agentOutputEvent{
				ThreadID: event.Output.ThreadID,
				TurnID:   event.Output.TurnID,
				ItemID:   event.Output.ItemID,
				Stream:   event.Output.Stream,
				Command:  event.Output.Command,
				Text:     event.Output.Text,
				Phase:    event.Output.Phase,
				Snapshot: event.Output.Snapshot,
			},
			Raw: rawCodexProviderEvent(event),
		}, true
	case codex.EventTypeThreadStatus:
		if event.ThreadStatus == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type: agentEventTypeThreadStatus,
			Thread: &agentThreadStatusEvent{
				ThreadID:    event.ThreadStatus.ThreadID,
				Status:      event.ThreadStatus.Status,
				ActiveFlags: append([]string(nil), event.ThreadStatus.ActiveFlags...),
			},
			Raw: rawCodexProviderEvent(event),
		}, true
	case codex.EventTypeTurnDiffUpdated:
		if event.Diff == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type: agentEventTypeTurnDiffUpdated,
			Diff: &agentTurnDiffEvent{
				ThreadID: event.Diff.ThreadID,
				TurnID:   event.Diff.TurnID,
				Diff:     event.Diff.Diff,
			},
			Raw: rawCodexProviderEvent(event),
		}, true
	case codex.EventTypeReasoningUpdated:
		if event.Reasoning == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type: agentEventTypeReasoningUpdated,
			Reasoning: &agentReasoningEvent{
				ThreadID:     event.Reasoning.ThreadID,
				TurnID:       event.Reasoning.TurnID,
				ItemID:       event.Reasoning.ItemID,
				Kind:         string(event.Reasoning.Kind),
				Delta:        event.Reasoning.Delta,
				SummaryIndex: cloneIntPointer(event.Reasoning.SummaryIndex),
				ContentIndex: cloneIntPointer(event.Reasoning.ContentIndex),
			},
			Raw: rawCodexProviderEvent(event),
		}, true
	case codex.EventTypeTurnStarted, codex.EventTypeTurnCompleted, codex.EventTypeTurnFailed:
		if event.Turn == nil {
			return agentEvent{}, false
		}
		mappedType := agentEventTypeTurnStarted
		switch event.Type {
		case codex.EventTypeTurnCompleted:
			mappedType = agentEventTypeTurnCompleted
		case codex.EventTypeTurnFailed:
			mappedType = agentEventTypeTurnFailed
		}
		return agentEvent{
			Type: mappedType,
			Turn: &agentTurnEvent{
				ThreadID: event.Turn.ThreadID,
				TurnID:   event.Turn.TurnID,
				Status:   event.Turn.Status,
				Error:    mapCodexTurnError(event.Turn.Error),
			},
			Raw: rawCodexProviderEvent(event),
		}, true
	default:
		return agentEvent{}, false
	}
}

func rawCodexProviderEvent(event codex.Event) *agentRawProviderEvent {
	method, payload, threadID, turnID, activityHintID, eventID, text := codexRawEventEnvelope(event)
	if method == "" {
		return nil
	}
	dedupKey := firstCodexNonEmpty(eventID, threadID+"|"+turnID+"|"+method+"|"+activityHintID)
	if strings.TrimSpace(dedupKey) == "" {
		return nil
	}
	kind, subtype := splitCodexMethod(method)
	return &agentRawProviderEvent{
		DedupKey:             dedupKey,
		ProviderEventKind:    kind,
		ProviderEventSubtype: subtype,
		ProviderEventID:      eventID,
		ThreadID:             threadID,
		TurnID:               turnID,
		ActivityHintID:       activityHintID,
		Payload:              payload,
		TextExcerpt:          text,
	}
}

func codexRawEventEnvelope(event codex.Event) (method string, payload map[string]any, threadID string, turnID string, activityHintID string, eventID string, text string) {
	switch event.Type {
	case codex.EventTypeItemStarted:
		if event.Item == nil {
			return "", nil, "", "", "", "", ""
		}
		payload = map[string]any{
			"thread_id": event.Item.ThreadID,
			"turn_id":   event.Item.TurnID,
			"item_id":   event.Item.ItemID,
			"item_type": event.Item.ItemType,
			"phase":     event.Item.Phase,
			"command":   event.Item.Command,
			"text":      event.Item.Text,
		}
		return "item/started", payload, event.Item.ThreadID, event.Item.TurnID, event.Item.ItemID, event.Item.ItemID, firstCodexNonEmpty(event.Item.Command, event.Item.Text)
	case codex.EventTypeOutputProduced:
		if event.Output == nil {
			return "", nil, "", "", "", "", ""
		}
		payload = map[string]any{
			"thread_id": event.Output.ThreadID,
			"turn_id":   event.Output.TurnID,
			"item_id":   event.Output.ItemID,
			"stream":    event.Output.Stream,
			"command":   event.Output.Command,
			"text":      event.Output.Text,
			"phase":     event.Output.Phase,
			"snapshot":  event.Output.Snapshot,
		}
		method = "item/agentMessage/delta"
		if strings.TrimSpace(event.Output.Stream) == "command" {
			method = "item/commandExecution/outputDelta"
		}
		if event.Output.Snapshot {
			method = "item/completed"
		}
		return method, payload, event.Output.ThreadID, event.Output.TurnID, event.Output.ItemID, event.Output.ItemID, firstCodexNonEmpty(event.Output.Command, event.Output.Text)
	case codex.EventTypeToolCallRequested:
		if event.ToolCall == nil {
			return "", nil, "", "", "", "", ""
		}
		payload = map[string]any{
			"thread_id": event.ToolCall.ThreadID,
			"turn_id":   event.ToolCall.TurnID,
			"call_id":   event.ToolCall.CallID,
			"tool":      event.ToolCall.Tool,
			"arguments": decodeRawJSON(event.ToolCall.Arguments),
		}
		return "item/tool/call", payload, event.ToolCall.ThreadID, event.ToolCall.TurnID, event.ToolCall.CallID, event.ToolCall.CallID, event.ToolCall.Tool
	case codex.EventTypeApprovalRequested:
		if event.Approval == nil {
			return "", nil, "", "", "", "", ""
		}
		method = "item/commandExecution/requestApproval"
		approvalKind := strings.TrimSpace(string(event.Approval.Kind))
		if approvalKind == "file_change" {
			method = "item/fileChange/requestApproval"
		}
		payload = map[string]any{
			"thread_id":  event.Approval.ThreadID,
			"turn_id":    event.Approval.TurnID,
			"request_id": event.Approval.RequestID,
			"kind":       approvalKind,
			"payload":    cloneCodexPayload(event.Approval.Payload),
		}
		requestID := event.Approval.RequestID.String()
		return method, payload, event.Approval.ThreadID, event.Approval.TurnID, requestID, requestID, approvalKind
	case codex.EventTypeUserInputRequested:
		if event.UserInput == nil {
			return "", nil, "", "", "", "", ""
		}
		payload = map[string]any{
			"thread_id":  event.UserInput.ThreadID,
			"turn_id":    event.UserInput.TurnID,
			"request_id": event.UserInput.RequestID,
			"payload":    cloneCodexPayload(event.UserInput.Payload),
		}
		requestID := event.UserInput.RequestID.String()
		return "item/tool/requestUserInput", payload, event.UserInput.ThreadID, event.UserInput.TurnID, requestID, requestID, ""
	case codex.EventTypeThreadStatus:
		if event.ThreadStatus == nil {
			return "", nil, "", "", "", "", ""
		}
		payload = map[string]any{
			"thread_id":    event.ThreadStatus.ThreadID,
			"status":       event.ThreadStatus.Status,
			"active_flags": append([]string(nil), event.ThreadStatus.ActiveFlags...),
		}
		return "thread/status/changed", payload, event.ThreadStatus.ThreadID, "", "", event.ThreadStatus.ThreadID + ":" + event.ThreadStatus.Status, event.ThreadStatus.Status
	case codex.EventTypeTurnDiffUpdated:
		if event.Diff == nil {
			return "", nil, "", "", "", "", ""
		}
		payload = map[string]any{
			"thread_id": event.Diff.ThreadID,
			"turn_id":   event.Diff.TurnID,
			"diff":      event.Diff.Diff,
		}
		return "turn/diff/updated", payload, event.Diff.ThreadID, event.Diff.TurnID, "", event.Diff.ThreadID + ":" + event.Diff.TurnID + ":diff", event.Diff.Diff
	case codex.EventTypeReasoningUpdated:
		if event.Reasoning == nil {
			return "", nil, "", "", "", "", ""
		}
		method = "item/reasoning/textDelta"
		switch event.Reasoning.Kind {
		case codex.ReasoningKindSummaryPart:
			method = "item/reasoning/summaryPartAdded"
		case codex.ReasoningKindSummaryText:
			method = "item/reasoning/summaryTextDelta"
		}
		payload = map[string]any{
			"thread_id":     event.Reasoning.ThreadID,
			"turn_id":       event.Reasoning.TurnID,
			"item_id":       event.Reasoning.ItemID,
			"kind":          event.Reasoning.Kind,
			"delta":         event.Reasoning.Delta,
			"summary_index": event.Reasoning.SummaryIndex,
			"content_index": event.Reasoning.ContentIndex,
		}
		return method, payload, event.Reasoning.ThreadID, event.Reasoning.TurnID, event.Reasoning.ItemID, event.Reasoning.ItemID, event.Reasoning.Delta
	case codex.EventTypeTokenUsageUpdated:
		if event.TokenUsage == nil {
			return "", nil, "", "", "", "", ""
		}
		payload = map[string]any{
			"thread_id":    event.TokenUsage.ThreadID,
			"turn_id":      event.TokenUsage.TurnID,
			"total_tokens": event.TokenUsage.TotalTokens,
			"last_tokens":  event.TokenUsage.LastTokens,
		}
		return "thread/tokenUsage/updated", payload, event.TokenUsage.ThreadID, event.TokenUsage.TurnID, "", event.TokenUsage.ThreadID + ":" + event.TokenUsage.TurnID + ":usage", ""
	case codex.EventTypeTurnStarted, codex.EventTypeTurnCompleted, codex.EventTypeTurnFailed:
		if event.Turn == nil {
			return "", nil, "", "", "", "", ""
		}
		method = "turn/started"
		switch event.Type {
		case codex.EventTypeTurnCompleted:
			method = "turn/completed"
		case codex.EventTypeTurnFailed:
			method = "turn/failed"
		}
		payload = map[string]any{
			"thread_id": event.Turn.ThreadID,
			"turn_id":   event.Turn.TurnID,
			"status":    event.Turn.Status,
		}
		if event.Turn.Error != nil {
			payload["error"] = map[string]any{
				"message":            event.Turn.Error.Message,
				"additional_details": event.Turn.Error.AdditionalDetails,
			}
		}
		errorMessage := ""
		if event.Turn.Error != nil {
			errorMessage = event.Turn.Error.Message
		}
		return method, payload, event.Turn.ThreadID, event.Turn.TurnID, "", event.Turn.ThreadID + ":" + event.Turn.TurnID + ":" + method, firstCodexNonEmpty(event.Turn.Status, errorMessage)
	default:
		return "", nil, "", "", "", "", ""
	}
}

func splitCodexMethod(method string) (string, string) {
	trimmed := strings.TrimSpace(method)
	if trimmed == "" {
		return "", ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], "/")
}

func firstCodexNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func mapCodexTurnError(err *codex.TurnError) *agentTurnError {
	if err == nil {
		return nil
	}
	return &agentTurnError{
		Message:           err.Message,
		AdditionalDetails: err.AdditionalDetails,
	}
}

func mapCodexApprovalOptions(options []codex.ApprovalOption) []agentApprovalOption {
	if len(options) == 0 {
		return nil
	}

	mapped := make([]agentApprovalOption, 0, len(options))
	for _, option := range options {
		mapped = append(mapped, agentApprovalOption{
			ID:          option.ID,
			Label:       option.Label,
			RawDecision: option.RawDecision,
		})
	}
	return mapped
}

func cloneCodexPayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
