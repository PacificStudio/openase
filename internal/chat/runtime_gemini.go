package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

type GeminiRuntime struct {
	processManager provider.AgentCLIProcessManager
	mu             sync.Mutex
	sessions       map[SessionID]*geminiRuntimeSession
}

type geminiRuntimeSession struct {
	turnsUsed int
	history   []geminiTranscriptEntry
	cancel    context.CancelFunc
}

type geminiTranscriptEntry struct {
	role    string
	content string
}

func NewGeminiRuntime(processManager provider.AgentCLIProcessManager) *GeminiRuntime {
	if processManager == nil {
		return nil
	}

	return &GeminiRuntime{processManager: processManager}
}

func (r *GeminiRuntime) Supports(providerItem catalogdomain.AgentProvider) bool {
	return r != nil &&
		r.processManager != nil &&
		providerItem.AdapterType == catalogdomain.AgentProviderAdapterTypeGeminiCLI
}

func (r *GeminiRuntime) StartTurn(ctx context.Context, input RuntimeTurnInput) (TurnStream, error) {
	if !r.Supports(input.Provider) {
		return TurnStream{}, fmt.Errorf("%w: %s", ErrProviderUnsupported, input.Provider.AdapterType)
	}

	command, err := provider.ParseAgentCLICommand(input.Provider.CliCommand)
	if err != nil {
		return TurnStream{}, err
	}

	state := r.session(input.SessionID)
	prompt := r.buildPrompt(state, input.SystemPrompt, input.Message)

	var workingDirectory *provider.AbsolutePath
	if input.WorkingDirectory != "" {
		workingDirectory = &input.WorkingDirectory
	}

	processSpec, err := provider.NewAgentCLIProcessSpec(
		command,
		buildGeminiArgs(input.Provider.CliArgs, input.Provider.ModelName, prompt),
		workingDirectory,
		provider.AuthConfigEnvironment(input.Provider.AuthConfig),
	)
	if err != nil {
		return TurnStream{}, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	process, err := r.processManager.Start(runCtx, processSpec)
	if err != nil {
		cancel()
		return TurnStream{}, fmt.Errorf("start gemini chat turn: %w", err)
	}
	r.setCancel(input.SessionID, cancel)

	events := make(chan StreamEvent, 16)
	go r.collectTurn(runCtx, input.SessionID, input.Message, input.MaxTurns, state, process, events)

	return TurnStream{Events: events}, nil
}

func (r *GeminiRuntime) CloseSession(sessionID SessionID) bool {
	r.mu.Lock()
	state := r.sessions[sessionID]
	if state != nil {
		delete(r.sessions, sessionID)
	}
	r.mu.Unlock()

	if state == nil {
		return false
	}

	if state.cancel != nil {
		state.cancel()
	}

	return true
}

func (r *GeminiRuntime) session(sessionID SessionID) *geminiRuntimeSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.sessions == nil {
		r.sessions = make(map[SessionID]*geminiRuntimeSession)
	}

	state := r.sessions[sessionID]
	if state == nil {
		state = &geminiRuntimeSession{}
		r.sessions[sessionID] = state
	}

	return state
}

func (r *GeminiRuntime) setCancel(sessionID SessionID, cancel context.CancelFunc) {
	r.mu.Lock()
	if r.sessions != nil {
		if state := r.sessions[sessionID]; state != nil {
			state.cancel = cancel
		}
	}
	r.mu.Unlock()
}

func (r *GeminiRuntime) clearCancel(sessionID SessionID) {
	r.mu.Lock()
	if r.sessions != nil {
		if state := r.sessions[sessionID]; state != nil {
			state.cancel = nil
		}
	}
	r.mu.Unlock()
}

func (r *GeminiRuntime) buildPrompt(
	state *geminiRuntimeSession,
	systemPrompt string,
	message string,
) string {
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(systemPrompt))
	if state != nil && len(state.history) > 0 {
		sb.WriteString("\n\n## Previous conversation\n")
		for _, entry := range state.history {
			_, _ = fmt.Fprintf(&sb, "%s: %s\n\n", entry.role, entry.content)
		}
	}
	sb.WriteString("## User request\n")
	sb.WriteString(strings.TrimSpace(message))
	return sb.String()
}

func (r *GeminiRuntime) collectTurn(
	ctx context.Context,
	sessionID SessionID,
	message string,
	maxTurns int,
	state *geminiRuntimeSession,
	process provider.AgentCLIProcess,
	events chan<- StreamEvent,
) {
	defer close(events)
	defer r.clearCancel(sessionID)

	stdout := process.Stdout()
	stderr := process.Stderr()

	var stdoutBytes []byte
	var stderrBytes []byte
	var stdoutErr error
	var stderrErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		stdoutBytes, stdoutErr = io.ReadAll(stdout)
	}()
	go func() {
		defer wg.Done()
		stderrBytes, stderrErr = io.ReadAll(stderr)
	}()

	waitErr := process.Wait()
	wg.Wait()

	if stdoutErr != nil {
		events <- StreamEvent{Event: "error", Payload: errorPayload{Message: stdoutErr.Error()}}
		return
	}
	if stderrErr != nil {
		events <- StreamEvent{Event: "error", Payload: errorPayload{Message: stderrErr.Error()}}
		return
	}
	if ctx.Err() != nil {
		return
	}
	if waitErr != nil {
		messageText := strings.TrimSpace(string(stderrBytes))
		if messageText == "" {
			messageText = waitErr.Error()
		}
		events <- StreamEvent{Event: "error", Payload: errorPayload{Message: messageText}}
		return
	}

	var payload struct {
		Response string         `json:"response"`
		Error    map[string]any `json:"error"`
	}
	if err := json.Unmarshal(stdoutBytes, &payload); err != nil {
		events <- StreamEvent{Event: "error", Payload: errorPayload{Message: fmt.Sprintf("parse gemini response: %v", err)}}
		return
	}
	if len(payload.Error) > 0 {
		events <- StreamEvent{
			Event:   "error",
			Payload: errorPayload{Message: strings.TrimSpace(string(stdoutBytes))},
		}
		return
	}

	responseText := strings.TrimSpace(payload.Response)
	if responseText != "" {
		events <- StreamEvent{
			Event:   "message",
			Payload: textPayload{Type: "text", Content: responseText},
		}
	}

	r.mu.Lock()
	if current := r.sessions[sessionID]; current != nil {
		current.history = append(current.history,
			geminiTranscriptEntry{role: "User", content: strings.TrimSpace(message)},
			geminiTranscriptEntry{role: "Assistant", content: responseText},
		)
		current.turnsUsed += 1
		state = current
	}
	r.mu.Unlock()

	turnsUsed := 0
	if state != nil {
		turnsUsed = state.turnsUsed
	}
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
}

func buildGeminiArgs(cliArgs []string, modelName string, prompt string) []string {
	args := append([]string(nil), cliArgs...)
	if strings.TrimSpace(modelName) != "" && !hasGeminiModelFlag(args) {
		args = append(args, "-m", modelName)
	}
	args = append(args, "-p", prompt, "--output-format", "json")
	return args
}

func hasGeminiModelFlag(args []string) bool {
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

func stopProcess(process provider.AgentCLIProcess) {
	if process == nil {
		return
	}
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer closeCancel()
	_ = process.Stop(closeCtx)
}
