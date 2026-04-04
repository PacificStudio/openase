package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
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
		}, true
	case codex.EventTypeRateLimitUpdated:
		if event.RateLimit == nil {
			return agentEvent{}, false
		}
		return agentEvent{
			Type:      agentEventTypeRateLimitUpdated,
			RateLimit: event.RateLimit,
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
		}, true
	default:
		return agentEvent{}, false
	}
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
