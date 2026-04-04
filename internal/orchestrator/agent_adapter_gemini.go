package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

type geminiAgentAdapter struct{}

func (geminiAgentAdapter) Start(_ context.Context, spec agentSessionStartSpec) (agentSession, error) {
	if spec.ProcessManager == nil {
		return nil, fmt.Errorf("gemini process manager must not be nil")
	}

	return newGeminiAgentSession(spec), nil
}

func (geminiAgentAdapter) Resume(_ context.Context, _ agentSessionResumeSpec) (agentSession, error) {
	return nil, unsupportedAgentAdapter{
		adapterType: entagentprovider.AdapterTypeGeminiCli,
		reason:      "Gemini CLI sessions cannot be resumed across orchestrator processes because conversation history is kept in memory",
	}.unsupportedError("resume")
}

type geminiAgentSession struct {
	process               provider.AgentCLIProcessSpec
	processManager        provider.AgentCLIProcessManager
	model                 string
	permissionProfile     catalogdomain.AgentProviderPermissionProfile
	developerInstructions string

	events chan agentEvent

	mu         sync.Mutex
	sessionID  string
	turnCount  int
	history    []geminiTranscriptEntry
	activeStop context.CancelFunc
	lastPID    int
	lastStderr string
	stopped    bool

	turnWG    sync.WaitGroup
	closeOnce sync.Once
}

type geminiTranscriptEntry struct {
	role    string
	content string
}

func newGeminiAgentSession(spec agentSessionStartSpec) *geminiAgentSession {
	return &geminiAgentSession{
		process:               spec.Process,
		processManager:        spec.ProcessManager,
		model:                 spec.Model,
		permissionProfile:     spec.PermissionProfile,
		developerInstructions: spec.DeveloperInstructions,
		sessionID:             "gemini-session-" + uuid.NewString(),
		events:                make(chan agentEvent, 64),
	}
}

func (s *geminiAgentSession) SessionID() (string, bool) {
	if s == nil {
		return "", false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionID, s.sessionID != ""
}

func (s *geminiAgentSession) Events() <-chan agentEvent {
	if s == nil {
		return nil
	}

	return s.events
}

func (s *geminiAgentSession) SendPrompt(ctx context.Context, prompt string) (agentTurnStartResult, error) {
	if s == nil {
		return agentTurnStartResult{}, fmt.Errorf("gemini session must not be nil")
	}

	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return agentTurnStartResult{}, fmt.Errorf("gemini session is stopped")
	}
	if s.activeStop != nil {
		s.mu.Unlock()
		return agentTurnStartResult{}, fmt.Errorf("gemini turn is already in progress")
	}

	s.turnCount++
	turnID := fmt.Sprintf("%s-turn-%d", s.sessionID, s.turnCount)
	promptText := s.buildPromptLocked(prompt)
	processArgs := buildGeminiTurnArgs(s.process.Args, s.model, promptText)
	if permissionArgs := buildGeminiPermissionArgs(s.permissionProfile); len(permissionArgs) > 0 {
		processArgs = append(processArgs, permissionArgs...)
	}
	processSpec, err := provider.NewAgentCLIProcessSpec(
		s.process.Command,
		processArgs,
		s.process.WorkingDirectory,
		s.process.Environment,
	)
	if err != nil {
		s.mu.Unlock()
		return agentTurnStartResult{}, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	process, err := s.processManager.Start(runCtx, processSpec)
	if err != nil {
		cancel()
		s.mu.Unlock()
		return agentTurnStartResult{}, fmt.Errorf("start gemini turn: %w", err)
	}
	if stdin := process.Stdin(); stdin != nil {
		if err := stdin.Close(); err != nil {
			cancel()
			s.mu.Unlock()
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer stopCancel()
			_ = process.Stop(stopCtx)
			return agentTurnStartResult{}, fmt.Errorf("close gemini stdin: %w", err)
		}
	}

	s.activeStop = cancel
	s.lastPID = process.PID()
	s.turnWG.Add(1)
	s.mu.Unlock()

	go s.collectTurn(runCtx, turnID, prompt, process)

	return agentTurnStartResult{TurnID: turnID}, nil
}

func (s *geminiAgentSession) Stop(_ context.Context) error {
	if s == nil {
		return fmt.Errorf("gemini session must not be nil")
	}

	s.mu.Lock()
	s.stopped = true
	cancel := s.activeStop
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	s.turnWG.Wait()
	s.closeEvents()
	return nil
}

func (s *geminiAgentSession) Err() error {
	return nil
}

func (s *geminiAgentSession) Diagnostic() agentSessionDiagnostic {
	if s == nil {
		return agentSessionDiagnostic{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return agentSessionDiagnostic{
		PID:       s.lastPID,
		SessionID: s.sessionID,
		Stderr:    s.lastStderr,
	}
}

func (s *geminiAgentSession) buildPromptLocked(prompt string) string {
	var builder strings.Builder
	if instructions := strings.TrimSpace(s.developerInstructions); instructions != "" {
		builder.WriteString(instructions)
		builder.WriteString("\n\n")
	}
	if len(s.history) > 0 {
		builder.WriteString("## Previous conversation\n")
		for _, entry := range s.history {
			_, _ = fmt.Fprintf(&builder, "%s: %s\n\n", entry.role, entry.content)
		}
	}
	builder.WriteString("## User request\n")
	builder.WriteString(strings.TrimSpace(prompt))
	return strings.TrimSpace(builder.String())
}

func (s *geminiAgentSession) collectTurn(
	ctx context.Context,
	turnID string,
	prompt string,
	process provider.AgentCLIProcess,
) {
	defer s.turnWG.Done()
	defer s.clearActiveTurn()

	stopDone := make(chan struct{})
	defer close(stopDone)
	go func() {
		select {
		case <-ctx.Done():
			stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer stopCancel()
			_ = process.Stop(stopCtx)
		case <-stopDone:
		}
	}()

	stdout := process.Stdout()
	stderr := process.Stderr()

	var stdoutBytes []byte
	var stderrBytes []byte
	var stdoutErr error
	var stderrErr error
	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		stdoutBytes, stdoutErr = io.ReadAll(stdout)
	}()
	go func() {
		defer readers.Done()
		stderrBytes, stderrErr = io.ReadAll(stderr)
	}()

	waitErr := process.Wait()
	readers.Wait()

	s.setLastStderr(string(stderrBytes))
	if ctx.Err() != nil {
		return
	}
	if stdoutErr != nil {
		s.emitTurnFailed(turnID, stdoutErr.Error())
		return
	}
	if stderrErr != nil {
		s.emitTurnFailed(turnID, stderrErr.Error())
		return
	}
	if waitErr != nil {
		messageText := strings.TrimSpace(string(stderrBytes))
		if messageText == "" {
			messageText = waitErr.Error()
		}
		s.emitTurnFailed(turnID, messageText)
		return
	}

	var payload struct {
		Response string         `json:"response"`
		Error    map[string]any `json:"error"`
	}
	if err := json.Unmarshal(stdoutBytes, &payload); err != nil {
		s.emitTurnFailed(turnID, fmt.Sprintf("parse gemini response: %v", err))
		return
	}
	if len(payload.Error) > 0 {
		messageText := strings.TrimSpace(string(stdoutBytes))
		if messageText == "" {
			messageText = "Gemini CLI returned an error payload"
		}
		s.emitTurnFailed(turnID, messageText)
		return
	}

	responseText := strings.TrimSpace(payload.Response)
	if responseText != "" {
		s.emit(agentEvent{
			Type: agentEventTypeOutputProduced,
			Output: &agentOutputEvent{
				ThreadID: s.sessionID,
				TurnID:   turnID,
				ItemID:   turnID,
				Stream:   "assistant",
				Text:     responseText,
				Snapshot: true,
			},
		})
	}

	usageInfo, err := provider.ParseGeminiCLIUsage(stdoutBytes)
	if err != nil {
		s.emitTurnFailed(turnID, fmt.Sprintf("parse gemini usage: %v", err))
		return
	}
	if usage := agentTokenUsageFromCLIUsage(s.sessionID, turnID, usageInfo); usage != nil {
		s.emit(agentEvent{
			Type:       agentEventTypeTokenUsageUpdated,
			TokenUsage: usage,
		})
	}
	s.emitGeminiRateLimit()

	s.appendHistory(prompt, responseText)
	s.emit(agentEvent{
		Type: agentEventTypeTurnCompleted,
		Turn: &agentTurnEvent{
			ThreadID: s.sessionID,
			TurnID:   turnID,
			Status:   "completed",
		},
	})
}

func (s *geminiAgentSession) appendHistory(prompt string, responseText string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.history = append(s.history,
		geminiTranscriptEntry{role: "User", content: strings.TrimSpace(prompt)},
		geminiTranscriptEntry{role: "Assistant", content: responseText},
	)
}

func (s *geminiAgentSession) emitTurnFailed(turnID string, message string) {
	s.emit(agentEvent{
		Type: agentEventTypeTurnFailed,
		Turn: &agentTurnEvent{
			ThreadID: s.sessionID,
			TurnID:   turnID,
			Status:   "failed",
			Error: &agentTurnError{
				Message: strings.TrimSpace(message),
			},
		},
	})
}

func (s *geminiAgentSession) emit(event agentEvent) {
	if s == nil {
		return
	}

	s.mu.Lock()
	stopped := s.stopped
	s.mu.Unlock()
	if stopped {
		return
	}

	s.events <- event
}

func (s *geminiAgentSession) emitGeminiRateLimit() {
	if s == nil || s.processManager == nil || s.process.Command == "" {
		return
	}

	probeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rateLimit, observedAt, err := provider.ProbeGeminiCLIRateLimit(
		probeCtx,
		s.processManager,
		s.process.Command,
		s.process.WorkingDirectory,
		s.process.Environment,
		s.model,
	)
	if err != nil || rateLimit == nil {
		return
	}

	s.emit(agentEvent{
		Type:       agentEventTypeRateLimitUpdated,
		RateLimit:  rateLimit,
		ObservedAt: observedAt,
	})
}

func (s *geminiAgentSession) clearActiveTurn() {
	s.mu.Lock()
	s.activeStop = nil
	s.lastPID = 0
	s.mu.Unlock()
}

func (s *geminiAgentSession) setLastStderr(stderr string) {
	s.mu.Lock()
	s.lastStderr = strings.TrimSpace(stderr)
	s.mu.Unlock()
}

func (s *geminiAgentSession) closeEvents() {
	s.closeOnce.Do(func() {
		close(s.events)
	})
}

func buildGeminiTurnArgs(baseArgs []string, model string, prompt string) []string {
	args := append([]string(nil), baseArgs...)
	if trimmed := strings.TrimSpace(model); trimmed != "" && !hasGeminiModelArg(args) {
		args = append(args, "-m", trimmed)
	}
	args = append(args, "-p", prompt, "--output-format", "json")
	return args
}

func buildGeminiPermissionArgs(profile catalogdomain.AgentProviderPermissionProfile) []string {
	profile = normalizeAgentPermissionProfile(profile)
	if profile != catalogdomain.AgentProviderPermissionProfileUnrestricted {
		return nil
	}
	return []string{"--approval-mode=yolo"}
}

func hasGeminiModelArg(args []string) bool {
	for index, arg := range args {
		switch {
		case arg == "-m" && index+1 < len(args):
			return true
		case arg == "--model" && index+1 < len(args):
			return true
		case strings.HasPrefix(arg, "--model="):
			return true
		}
	}

	return false
}
