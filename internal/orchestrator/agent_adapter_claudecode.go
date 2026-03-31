package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/BetterAndBetterII/openase/internal/infra/adapter/claudecode"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

type claudeCodeAgentAdapter struct{}

func (claudeCodeAgentAdapter) Start(ctx context.Context, spec agentSessionStartSpec) (agentSession, error) {
	if spec.ProcessManager == nil {
		return nil, fmt.Errorf("claude code process manager must not be nil")
	}

	baseArgs := append([]string(nil), spec.Process.Args...)
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

type claudeCodeAgentSession struct {
	session provider.ClaudeCodeSession
	events  chan agentEvent

	sessionMu sync.RWMutex
	sessionID string

	doneMu  sync.RWMutex
	doneErr error
}

func newClaudeCodeAgentSession(session provider.ClaudeCodeSession) *claudeCodeAgentSession {
	wrapped := &claudeCodeAgentSession{
		session: session,
		events:  make(chan agentEvent, 64),
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
			for _, mapped := range mapClaudeCodeAgentEvents(event) {
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

func mapClaudeCodeAgentEvents(event provider.ClaudeCodeEvent) []agentEvent {
	switch event.Kind {
	case provider.ClaudeCodeEventKindAssistant:
		text := strings.TrimSpace(strings.Join(extractClaudeAssistantTextBlocks(event.Message), "\n\n"))
		if text == "" {
			return nil
		}
		return []agentEvent{{
			Type: agentEventTypeOutputProduced,
			Output: &agentOutputEvent{
				Stream:   "assistant",
				Text:     text,
				Snapshot: true,
			},
		}}
	case provider.ClaudeCodeEventKindTaskStart:
		return []agentEvent{{
			Type: agentEventTypeTurnStarted,
			Turn: &agentTurnEvent{Status: "started"},
		}}
	case provider.ClaudeCodeEventKindTaskProgress:
		text := strings.TrimSpace(string(event.Raw))
		if text == "" {
			return nil
		}
		return []agentEvent{{
			Type: agentEventTypeOutputProduced,
			Output: &agentOutputEvent{
				Stream: "task",
				Text:   text,
			},
		}}
	case provider.ClaudeCodeEventKindResult:
		events := make([]agentEvent, 0, 2)
		if usage := parseClaudeUsage(event.Usage); usage != nil {
			events = append(events, agentEvent{
				Type:       agentEventTypeTokenUsageUpdated,
				TokenUsage: usage,
			})
		}
		completed := agentEvent{
			Type: agentEventTypeTurnCompleted,
			Turn: &agentTurnEvent{Status: "completed"},
		}
		if event.IsError {
			completed.Type = agentEventTypeTurnFailed
			completed.Turn.Error = &agentTurnError{Message: strings.TrimSpace(event.Result)}
		}
		events = append(events, completed)
		return events
	default:
		return nil
	}
}

func extractClaudeAssistantTextBlocks(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}

	var message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil
	}

	items := make([]string, 0, len(message.Content))
	for _, block := range message.Content {
		if block.Type != "text" {
			continue
		}
		text := strings.TrimSpace(block.Text)
		if text == "" {
			continue
		}
		items = append(items, text)
	}
	return items
}

func parseClaudeUsage(raw json.RawMessage) *agentTokenUsageEvent {
	if len(raw) == 0 {
		return nil
	}

	var usage struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
		TotalTokens  int64 `json:"total_tokens"`
	}
	if err := json.Unmarshal(raw, &usage); err != nil {
		return nil
	}
	if usage.InputTokens == 0 && usage.OutputTokens == 0 && usage.TotalTokens == 0 {
		return nil
	}
	return &agentTokenUsageEvent{
		TotalInputTokens:  usage.InputTokens,
		TotalOutputTokens: usage.OutputTokens,
		TotalTokens:       usage.TotalTokens,
	}
}
