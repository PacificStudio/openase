package chat

import (
	"bufio"
	"context"
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

type geminiPendingToolCall struct {
	useEvent geminiCLIToolUseEvent
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
		buildGeminiArgs(input.Provider.CliArgs, input.Provider.ModelName, input.Provider.PermissionProfile, prompt),
		workingDirectory,
		append(provider.AuthConfigEnvironment(input.Provider.AuthConfig), input.Environment...),
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
	if stdin := process.Stdin(); stdin != nil {
		if err := stdin.Close(); err != nil {
			cancel()
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer stopCancel()
			_ = process.Stop(stopCtx)
			return TurnStream{}, fmt.Errorf("close gemini stdin: %w", err)
		}
	}
	r.setCancel(input.SessionID, cancel)

	events := make(chan StreamEvent, 16)
	go r.collectTurn(runCtx, input.SessionID, input.Provider, input.Message, input.MaxTurns, state, process, events)

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
	providerItem catalogdomain.AgentProvider,
	message string,
	maxTurns int,
	state *geminiRuntimeSession,
	process provider.AgentCLIProcess,
	events chan<- StreamEvent,
) {
	defer close(events)
	defer r.clearCancel(sessionID)

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

	var stderrBytes []byte
	var stderrErr error
	var stderrWG sync.WaitGroup
	stderrWG.Add(1)
	go func() {
		defer stderrWG.Done()
		stderrBytes, stderrErr = io.ReadAll(stderr)
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	pendingTools := make(map[string]geminiPendingToolCall)
	var assistantText strings.Builder
	var providerSessionID string
	var usageInfo *provider.CLIUsage
	var resultEvent *geminiCLIResultEvent
	var resultRaw []byte

	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		rawLine := []byte(line)
		protocolEvent, err := parseGeminiCLIStreamEvent(rawLine)
		if err != nil {
			events <- StreamEvent{Event: "error", Payload: errorPayload{Message: err.Error()}}
			return
		}

		switch typed := protocolEvent.(type) {
		case geminiCLIInitEvent:
			providerSessionID = strings.TrimSpace(typed.SessionID)
			if providerSessionID == "" {
				continue
			}
			events <- StreamEvent{
				Event: "session_anchor",
				Payload: RuntimeSessionAnchor{
					ProviderThreadID:      providerSessionID,
					ProviderAnchorID:      providerSessionID,
					ProviderAnchorKind:    "session",
					ProviderTurnSupported: false,
				},
			}
		case geminiCLIMessageEvent:
			if strings.TrimSpace(typed.Role) != "assistant" {
				continue
			}
			content := typed.Content
			if strings.TrimSpace(content) == "" {
				continue
			}
			assistantText.WriteString(content)
			for _, item := range normalizeAssistantText(content) {
				events <- item
			}
		case geminiCLIToolUseEvent:
			if strings.TrimSpace(typed.ToolID) != "" {
				pendingTools[typed.ToolID] = geminiPendingToolCall{useEvent: typed}
			}
			events <- newTaskMessageEvent(chatMessageTypeTaskNotification, map[string]any{
				"provider":  "gemini_cli",
				"tool":      geminiCLIToolDisplayName(typed.ToolName),
				"tool_id":   strings.TrimSpace(typed.ToolID),
				"semantic":  string(deriveGeminiCLIToolSemantic(typed.ToolName)),
				"arguments": cloneAnyMap(typed.Parameters),
			})
		case geminiCLIToolResultEvent:
			toolUse, ok := pendingTools[strings.TrimSpace(typed.ToolID)]
			if ok {
				delete(pendingTools, strings.TrimSpace(typed.ToolID))
			}
			messageText := geminiCLIToolResultMessage(typed)
			semantic := deriveGeminiCLIToolSemantic(toolUse.useEvent.ToolName)
			if semantic == geminiCLIToolSemanticCommand && strings.TrimSpace(messageText) != "" {
				events <- newTaskMessageEvent(chatMessageTypeTaskProgress, map[string]any{
					"provider": "gemini_cli",
					"stream":   "command",
					"command":  geminiCLIToolCommand(toolUse.useEvent),
					"text":     messageText,
					"snapshot": true,
					"tool":     geminiCLIToolDisplayName(toolUse.useEvent.ToolName),
					"tool_id":  strings.TrimSpace(typed.ToolID),
					"status":   strings.TrimSpace(typed.Status),
				})
				continue
			}

			raw := map[string]any{
				"provider": "gemini_cli",
				"tool_id":  strings.TrimSpace(typed.ToolID),
				"status":   strings.TrimSpace(typed.Status),
				"message":  messageText,
			}
			if ok {
				raw["tool"] = geminiCLIToolDisplayName(toolUse.useEvent.ToolName)
				raw["semantic"] = string(semantic)
				raw["arguments"] = cloneAnyMap(toolUse.useEvent.Parameters)
			}
			if typed.Error != nil {
				raw["error"] = map[string]any{
					"type":    strings.TrimSpace(typed.Error.Type),
					"message": strings.TrimSpace(typed.Error.Message),
				}
			}
			events <- newTaskMessageEvent(chatMessageTypeTaskProgress, raw)
		case geminiCLIErrorEvent:
			events <- newTaskMessageEvent(chatMessageTypeTaskNotification, map[string]any{
				"provider": "gemini_cli",
				"event":    "error",
				"severity": strings.TrimSpace(typed.Severity),
				"status":   strings.TrimSpace(typed.Severity),
				"message":  strings.TrimSpace(typed.Message),
			})
		case geminiCLIResultEvent:
			copied := typed
			resultEvent = &copied
			resultRaw = append(resultRaw[:0], rawLine...)
			parsedUsage, err := provider.ParseGeminiCLIStreamUsage(resultRaw)
			if err != nil {
				events <- StreamEvent{Event: "error", Payload: errorPayload{Message: fmt.Sprintf("parse gemini stream usage: %v", err)}}
				return
			}
			usageInfo = parsedUsage
		}
	}
	if err := scanner.Err(); err != nil {
		events <- StreamEvent{Event: "error", Payload: errorPayload{Message: fmt.Sprintf("read gemini stream-json stdout: %v", err)}}
		return
	}

	waitErr := process.Wait()
	stderrWG.Wait()

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
	if resultEvent == nil {
		events <- StreamEvent{Event: "error", Payload: errorPayload{Message: "gemini stream-json ended without a result event"}}
		return
	}
	if strings.TrimSpace(resultEvent.Status) != "success" {
		messageText := ""
		if resultEvent.Error != nil {
			messageText = strings.TrimSpace(resultEvent.Error.Message)
			if messageText == "" {
				messageText = strings.TrimSpace(resultEvent.Error.Type)
			}
		}
		if messageText == "" {
			messageText = strings.TrimSpace(string(stderrBytes))
		}
		if messageText == "" {
			messageText = "gemini chat turn failed"
		}
		events <- StreamEvent{Event: "error", Payload: errorPayload{Message: messageText}}
		return
	}

	r.mu.Lock()
	if current := r.sessions[sessionID]; current != nil {
		responseText := strings.TrimSpace(assistantText.String())
		current.history = append(current.history,
			geminiTranscriptEntry{role: "User", content: strings.TrimSpace(message)},
		)
		if responseText != "" {
			current.history = append(current.history, geminiTranscriptEntry{role: "Assistant", content: responseText})
		}
		current.turnsUsed++
		state = current
	}
	r.mu.Unlock()

	turnsUsed := 0
	if state != nil {
		turnsUsed = state.turnsUsed
	}
	costUSD := resolveCLIUsageCostUSD(providerItem, usageInfo)
	if payload := runtimeTokenUsagePayloadFromCLIUsage(usageInfo, costUSD); payload != nil {
		events <- StreamEvent{Event: "token_usage_updated", Payload: *payload}
	}
	events <- StreamEvent{
		Event: "done",
		Payload: donePayload{
			SessionID:      sessionID.String(),
			CostUSD:        costUSD,
			TurnsUsed:      turnsUsed,
			TurnsRemaining: remainingTurns(maxTurns, turnsUsed),
		},
	}
}

func buildGeminiArgs(
	cliArgs []string,
	modelName string,
	profile catalogdomain.AgentProviderPermissionProfile,
	prompt string,
) []string {
	args := append([]string(nil), cliArgs...)
	if strings.TrimSpace(modelName) != "" && !hasGeminiModelFlag(args) {
		args = append(args, "-m", modelName)
	}
	if normalizeRuntimePermissionProfile(profile) == catalogdomain.AgentProviderPermissionProfileUnrestricted &&
		!hasGeminiApprovalModeYoloArg(args) {
		args = append(args, "--approval-mode=yolo")
	}
	args = append(args, "-p", prompt, "--output-format", "stream-json")
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

func hasGeminiApprovalModeYoloArg(args []string) bool {
	for index := 0; index < len(args); index++ {
		switch {
		case args[index] == "--approval-mode" && index+1 < len(args) &&
			strings.EqualFold(strings.TrimSpace(args[index+1]), "yolo"):
			return true
		case strings.EqualFold(strings.TrimSpace(args[index]), "--approval-mode=yolo"):
			return true
		case args[index] == "-y", args[index] == "--yolo":
			return true
		}
	}
	return false
}
