package orchestrator

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestCodexAgentAdapterSatisfiesRuntimeContract(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	manager := &runtimeRunnerFakeProcessManager{process: process}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runRuntimeRunnerProtocolServer(process, func(decoder *json.Decoder, encoder *json.Encoder) error {
			if err := runtimeRunnerCompleteHandshake(decoder, encoder); err != nil {
				return err
			}

			turnStart, err := runtimeRunnerReadMessage(decoder)
			if err != nil {
				return err
			}
			if turnStart.Method != "turn/start" {
				return errors.New("expected turn/start request")
			}
			if err := encoder.Encode(runtimeRunnerJSONRPCMessage{
				JSONRPC: "2.0",
				ID:      turnStart.ID,
				Result:  mustMarshalJSON(map[string]any{"turn": map[string]any{"id": "turn-contract", "status": "inProgress"}}),
			}); err != nil {
				return err
			}
			if err := encoder.Encode(runtimeRunnerJSONRPCMessage{
				JSONRPC: "2.0",
				Method:  "account/rateLimits/updated",
				Params: mustMarshalJSON(map[string]any{
					"rateLimits": map[string]any{
						"limitId": "codex",
						"primary": map[string]any{
							"usedPercent":        15,
							"windowDurationMins": 300,
							"resetsAt":           1775050232,
						},
						"planType": "pro",
					},
				}),
			}); err != nil {
				return err
			}
			if err := encoder.Encode(runtimeRunnerJSONRPCMessage{
				JSONRPC: "2.0",
				Method:  "turn/completed",
				Params:  mustMarshalJSON(map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-contract", "status": "completed"}}),
			}); err != nil {
				return err
			}
			process.finish(nil)
			return nil
		})
	}()

	processSpec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("codex"),
		[]string{"app-server", "--listen", "stdio://"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	session, err := codexAgentAdapter{}.Start(context.Background(), agentSessionStartSpec{
		Process:          processSpec,
		ProcessManager:   manager,
		WorkingDirectory: "/tmp/openase",
		TurnTitle:        "ASE-377",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	sessionID, ok := session.SessionID()
	if !ok || sessionID != "thread-1" {
		t.Fatalf("SessionID() = %q, %t", sessionID, ok)
	}

	turn, err := session.SendPrompt(context.Background(), "Keep working.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID != "turn-contract" {
		t.Fatalf("SendPrompt() turn id = %q", turn.TurnID)
	}

	first := requireAgentEvent(t, session.Events())
	if first.Type != agentEventTypeRateLimitUpdated || first.RateLimit == nil || first.RateLimit.Codex == nil {
		t.Fatalf("unexpected first event: %+v", first)
	}
	if first.ObservedAt == nil {
		t.Fatalf("expected observedAt on rate limit event, got %+v", first)
	}

	second := requireAgentEvent(t, session.Events())
	if second.Type != agentEventTypeTurnCompleted || second.Turn == nil || second.Turn.TurnID != "turn-contract" {
		t.Fatalf("unexpected second event: %+v", second)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func TestClaudeCodeAgentAdapterSatisfiesRuntimeContract(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	manager := &runtimeRunnerFakeProcessManager{process: process}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runClaudeRuntimeProtocol(process)
	}()

	processSpec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("claude"),
		[]string{"--verbose"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	session, err := claudeCodeAgentAdapter{}.Start(context.Background(), agentSessionStartSpec{
		Process:               processSpec,
		ProcessManager:        manager,
		DeveloperInstructions: "Follow the harness.",
		ReasoningEffort:       reasoningEffortPointer(catalogdomain.AgentProviderReasoningEffortHigh),
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	turn, err := session.SendPrompt(context.Background(), "Implement the fix.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID != "" {
		t.Fatalf("SendPrompt() turn id = %q, want empty Claude turn id", turn.TurnID)
	}

	first := requireAgentEvent(t, session.Events())
	if first.Type != agentEventTypeOutputProduced || first.Output == nil || first.Output.Text != "Implemented the shared contract." {
		t.Fatalf("unexpected first event: %+v", first)
	}
	if first.Output.ItemID != "assistant-1" {
		t.Fatalf("unexpected first output item id: %+v", first.Output)
	}
	second := requireAgentEvent(t, session.Events())
	if second.Type != agentEventTypeRateLimitUpdated || second.RateLimit == nil || second.RateLimit.ClaudeCode == nil {
		t.Fatalf("unexpected second event: %+v", second)
	}
	third := requireAgentEvent(t, session.Events())
	if third.Type != agentEventTypeTurnCompleted || third.Turn == nil || third.Turn.Status != "completed" {
		t.Fatalf("unexpected third event: %+v", third)
	}

	sessionID, ok := session.SessionID()
	if !ok || sessionID != "claude-session-1" {
		t.Fatalf("SessionID() = %q, %t", sessionID, ok)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestClaudeCodeAgentAdapterPreservesTaskAndSessionEventsAndFallbackFailureDetails(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	manager := &runtimeRunnerFakeProcessManager{process: process}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runClaudeTaskRuntimeProtocol(process)
	}()

	processSpec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("claude"),
		[]string{"--verbose"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	session, err := claudeCodeAgentAdapter{}.Start(context.Background(), agentSessionStartSpec{
		Process:        processSpec,
		ProcessManager: manager,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if _, err := session.SendPrompt(context.Background(), "Implement the fix."); err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}

	first := requireAgentEvent(t, session.Events())
	if first.Type != agentEventTypeTaskStatus || first.TaskStatus == nil || first.TaskStatus.StatusType != "task_started" {
		t.Fatalf("unexpected first event: %+v", first)
	}
	if first.TaskStatus.TurnID != "turn-claude-1" || first.TaskStatus.Payload["turn_id"] != "turn-claude-1" {
		t.Fatalf("unexpected task_started payload: %+v", first.TaskStatus)
	}

	second := requireAgentEvent(t, session.Events())
	if second.Type != agentEventTypeTurnStarted || second.Turn == nil || second.Turn.TurnID != "turn-claude-1" {
		t.Fatalf("unexpected second event: %+v", second)
	}

	third := requireAgentEvent(t, session.Events())
	if third.Type != agentEventTypeTaskStatus || third.TaskStatus == nil || third.TaskStatus.StatusType != "task_progress" {
		t.Fatalf("unexpected third event: %+v", third)
	}
	if third.TaskStatus.ItemID != "tool-use-1" {
		t.Fatalf("unexpected task_progress item id: %+v", third.TaskStatus)
	}
	if third.TaskStatus.Payload["stream"] != "command" || third.TaskStatus.Payload["command"] != "pwd" {
		t.Fatalf("unexpected task_progress payload: %+v", third.TaskStatus.Payload)
	}

	fourth := requireAgentEvent(t, session.Events())
	if fourth.Type != agentEventTypeTaskStatus || fourth.TaskStatus == nil || fourth.TaskStatus.StatusType != "session_state" {
		t.Fatalf("unexpected fourth event: %+v", fourth)
	}
	if fourth.TaskStatus.Payload["status"] != "active" {
		t.Fatalf("unexpected session_state payload: %+v", fourth.TaskStatus.Payload)
	}

	fifth := requireAgentEvent(t, session.Events())
	if fifth.Type != agentEventTypeTaskStatus || fifth.TaskStatus == nil || fifth.TaskStatus.StatusType != "error" {
		t.Fatalf("unexpected fifth event: %+v", fifth)
	}

	sixth := requireAgentEvent(t, session.Events())
	if sixth.Type != agentEventTypeTurnFailed || sixth.Turn == nil || sixth.Turn.Error == nil {
		t.Fatalf("unexpected sixth event: %+v", sixth)
	}
	if !strings.Contains(sixth.Turn.Error.Message, "empty error result") {
		t.Fatalf("unexpected turn failure message: %+v", sixth.Turn.Error)
	}
	if !strings.Contains(sixth.Turn.Error.AdditionalDetails, `"subtype":"error"`) {
		t.Fatalf("unexpected turn failure details: %+v", sixth.Turn.Error)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestClaudeCodeAgentAdapterSynthesizesStableAssistantItemIDsWithoutProviderUUIDs(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	manager := &runtimeRunnerFakeProcessManager{process: process}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runClaudeAssistantSnapshotProtocol(process)
	}()

	processSpec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("claude"),
		[]string{"--verbose"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	session, err := claudeCodeAgentAdapter{}.Start(context.Background(), agentSessionStartSpec{
		Process:        processSpec,
		ProcessManager: manager,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if _, err := session.SendPrompt(context.Background(), "Implement the fix."); err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}

	first := requireAgentEvent(t, session.Events())
	second := requireAgentEvent(t, session.Events())
	third := requireAgentEvent(t, session.Events())

	if first.Output == nil || second.Output == nil || third.Output == nil {
		t.Fatalf("expected assistant outputs, got %+v %+v %+v", first, second, third)
	}
	if first.Output.ItemID == "" || second.Output.ItemID == "" || third.Output.ItemID == "" {
		t.Fatalf("expected synthesized assistant item ids, got %+v %+v %+v", first.Output, second.Output, third.Output)
	}
	if first.Output.ItemID != second.Output.ItemID {
		t.Fatalf("expected extending snapshots to keep the same item id, got %+v %+v", first.Output, second.Output)
	}
	if third.Output.ItemID == second.Output.ItemID {
		t.Fatalf("expected a new assistant item id for a non-extending snapshot, got %+v %+v", second.Output, third.Output)
	}

	completed := requireAgentEvent(t, session.Events())
	if completed.Type != agentEventTypeTurnCompleted {
		t.Fatalf("unexpected completion event: %+v", completed)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestClaudeCodeAgentAdapterSynthesizesStableTaskProgressItemIDsWithoutProviderIDs(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	manager := &runtimeRunnerFakeProcessManager{process: process}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runClaudeTaskProgressSnapshotProtocol(process)
	}()

	processSpec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("claude"),
		[]string{"--verbose"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	session, err := claudeCodeAgentAdapter{}.Start(context.Background(), agentSessionStartSpec{
		Process:        processSpec,
		ProcessManager: manager,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if _, err := session.SendPrompt(context.Background(), "Implement the fix."); err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}

	first := requireAgentEvent(t, session.Events())
	second := requireAgentEvent(t, session.Events())
	third := requireAgentEvent(t, session.Events())

	if first.TaskStatus == nil || second.TaskStatus == nil || third.TaskStatus == nil {
		t.Fatalf("expected task status events, got %+v %+v %+v", first, second, third)
	}
	if first.TaskStatus.ItemID == "" || second.TaskStatus.ItemID == "" || third.TaskStatus.ItemID == "" {
		t.Fatalf("expected synthesized task item ids, got %+v %+v %+v", first.TaskStatus, second.TaskStatus, third.TaskStatus)
	}
	if first.TaskStatus.ItemID != second.TaskStatus.ItemID {
		t.Fatalf("expected extending command snapshots to keep the same item id, got %+v %+v", first.TaskStatus, second.TaskStatus)
	}
	if third.TaskStatus.ItemID == second.TaskStatus.ItemID {
		t.Fatalf("expected a new task item id for a new command snapshot, got %+v %+v", second.TaskStatus, third.TaskStatus)
	}

	completed := requireAgentEvent(t, session.Events())
	if completed.Type != agentEventTypeTurnCompleted {
		t.Fatalf("unexpected completion event: %+v", completed)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestClaudeCodeAgentAdapterMapsAssistantToolUseAndToolResults(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	manager := &runtimeRunnerFakeProcessManager{process: process}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runClaudeToolUseRuntimeProtocol(process)
	}()

	processSpec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("claude"),
		[]string{"--verbose"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	session, err := claudeCodeAgentAdapter{}.Start(context.Background(), agentSessionStartSpec{
		Process:        processSpec,
		ProcessManager: manager,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if _, err := session.SendPrompt(context.Background(), "Implement the fix."); err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}

	first := requireAgentEvent(t, session.Events())
	if first.Type != agentEventTypeOutputProduced || first.Output == nil || first.Output.Stream != "assistant" {
		t.Fatalf("unexpected first event: %+v", first)
	}
	if first.Output.Text != "Let me inspect the repository first." {
		t.Fatalf("unexpected first assistant output: %+v", first.Output)
	}

	second := requireAgentEvent(t, session.Events())
	if second.Type != agentEventTypeToolCallRequested || second.ToolCall == nil {
		t.Fatalf("unexpected second event: %+v", second)
	}
	if second.ToolCall.CallID != "toolu_01" || second.ToolCall.Tool != "functions.exec_command" {
		t.Fatalf("unexpected tool call payload: %+v", second.ToolCall)
	}
	if string(second.ToolCall.Arguments) != `{"cmd":"git status --short"}` {
		t.Fatalf("unexpected tool call arguments: %s", second.ToolCall.Arguments)
	}

	third := requireAgentEvent(t, session.Events())
	if third.Type != agentEventTypeOutputProduced || third.Output == nil {
		t.Fatalf("unexpected third event: %+v", third)
	}
	if third.Output.Stream != "command" || third.Output.Command != "git status --short" {
		t.Fatalf("unexpected command output metadata: %+v", third.Output)
	}
	if third.Output.ItemID != "toolu_01" || third.Output.Text != "M README.md" {
		t.Fatalf("unexpected command output payload: %+v", third.Output)
	}

	completed := requireAgentEvent(t, session.Events())
	if completed.Type != agentEventTypeTurnCompleted {
		t.Fatalf("unexpected completion event: %+v", completed)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestClaudeCodeAgentAdapterExtractsUnifiedDiffFromToolResults(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	manager := &runtimeRunnerFakeProcessManager{process: process}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runClaudeToolResultDiffRuntimeProtocol(process)
	}()

	processSpec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("claude"),
		[]string{"--verbose"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	session, err := claudeCodeAgentAdapter{}.Start(context.Background(), agentSessionStartSpec{
		Process:        processSpec,
		ProcessManager: manager,
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if _, err := session.SendPrompt(context.Background(), "Implement the fix."); err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}

	toolCall := requireAgentEvent(t, session.Events())
	if toolCall.Type != agentEventTypeToolCallRequested || toolCall.ToolCall == nil {
		t.Fatalf("unexpected tool call event: %+v", toolCall)
	}

	output := requireAgentEvent(t, session.Events())
	if output.Type != agentEventTypeOutputProduced || output.Output == nil {
		t.Fatalf("unexpected output event: %+v", output)
	}
	if output.Output.Stream != "command" || output.Output.Command != "git diff -- README.md" {
		t.Fatalf("unexpected command output metadata: %+v", output.Output)
	}

	diff := requireAgentEvent(t, session.Events())
	if diff.Type != agentEventTypeTurnDiffUpdated || diff.Diff == nil {
		t.Fatalf("unexpected diff event: %+v", diff)
	}
	if !strings.Contains(diff.Diff.Diff, "diff --git a/README.md b/README.md") {
		t.Fatalf("unexpected diff payload: %+v", diff.Diff)
	}

	completed := requireAgentEvent(t, session.Events())
	if completed.Type != agentEventTypeTurnCompleted {
		t.Fatalf("unexpected completion event: %+v", completed)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestDefaultAgentAdapterRegistryRegistersGeminiRuntimeContract(t *testing.T) {
	ensureGeminiCLIProbePath(t)

	process := newRuntimeRunnerFakeProcess()
	probeProcess := &stubGeminiProbeProcess{
		stdout: io.NopCloser(strings.NewReader(`{"authType":"oauth-personal","remaining":3,"limit":10,"resetTime":"2026-04-02T10:02:55Z","buckets":[{"modelId":"gemini-2.5-pro","tokenType":"REQUESTS","remainingFraction":0.3,"resetTime":"2026-04-02T10:02:55Z"}]}`)),
		stderr: io.NopCloser(strings.NewReader("")),
	}
	manager := &geminiAdapterTestProcessManager{processes: []provider.AgentCLIProcess{process, probeProcess}}

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runGeminiRuntimeProtocol(process)
	}()

	registry := newDefaultAgentAdapterRegistry()
	adapter, err := registry.adapterFor(entagentprovider.AdapterTypeGeminiCli)
	if err != nil {
		t.Fatalf("adapterFor(gemini) returned error: %v", err)
	}
	if _, ok := adapter.(geminiAgentAdapter); !ok {
		t.Fatalf("adapterFor(gemini) = %T, want geminiAgentAdapter", adapter)
	}

	processSpec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("gemini"),
		[]string{"--sandbox", "danger-full-access"},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewAgentCLIProcessSpec returned error: %v", err)
	}

	session, err := adapter.Start(context.Background(), agentSessionStartSpec{
		Process:        processSpec,
		ProcessManager: manager,
		Model:          "gemini-2.5-pro",
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	turn, err := session.SendPrompt(context.Background(), "Implement the fix.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID == "" {
		t.Fatal("SendPrompt() turn id = empty, want generated Gemini turn id")
	}

	first := requireAgentEvent(t, session.Events())
	if first.Type != agentEventTypeOutputProduced || first.Output == nil || first.Output.Text != "Implemented the shared contract." {
		t.Fatalf("unexpected first event: %+v", first)
	}
	if first.Output.Stream != "assistant" || !first.Output.Snapshot || first.Output.TurnID != turn.TurnID {
		t.Fatalf("unexpected first output metadata: %+v", first.Output)
	}

	second := requireAgentEvent(t, session.Events())
	if second.Type != agentEventTypeTokenUsageUpdated || second.TokenUsage == nil {
		t.Fatalf("unexpected second event: %+v", second)
	}
	if second.TokenUsage.TotalInputTokens != 120 || second.TokenUsage.TotalOutputTokens != 35 || second.TokenUsage.TotalTokens != 155 {
		t.Fatalf("unexpected Gemini token usage event: %+v", second.TokenUsage)
	}

	third := requireAgentEvent(t, session.Events())
	if third.Type == agentEventTypeRateLimitUpdated {
		if third.RateLimit == nil || third.RateLimit.Gemini == nil {
			t.Fatalf("unexpected Gemini rate limit event: %+v", third)
		}
		third = requireAgentEvent(t, session.Events())
	}
	if third.Type != agentEventTypeTurnCompleted || third.Turn == nil || third.Turn.Status != "completed" || third.Turn.TurnID != turn.TurnID {
		t.Fatalf("unexpected turn completion event: %+v", third)
	}

	if len(manager.capturedSpecs) < 2 {
		t.Fatalf("captured specs = %d, want at least 2", len(manager.capturedSpecs))
	}
	turnSpec := manager.capturedSpecs[0]
	if turnSpec.Command != provider.MustParseAgentCLICommand("gemini") {
		t.Fatalf("turn command = %q, want gemini", turnSpec.Command)
	}
	joinedArgs := strings.Join(turnSpec.Args, " ")
	if !strings.Contains(joinedArgs, "-m gemini-2.5-pro") {
		t.Fatalf("turn args = %v, want model flag", turnSpec.Args)
	}
	if !strings.Contains(joinedArgs, "--output-format json") {
		t.Fatalf("turn args = %v, want json output mode", turnSpec.Args)
	}
	if !strings.Contains(joinedArgs, "--approval-mode=yolo") {
		t.Fatalf("turn args = %v, want yolo approval mode", turnSpec.Args)
	}
	if !strings.Contains(joinedArgs, "-p") {
		t.Fatalf("turn args = %v, want prompt flag", turnSpec.Args)
	}
	probeSpec := manager.capturedSpecs[1]
	if probeSpec.Command != provider.MustParseAgentCLICommand("node") {
		t.Fatalf("probe command = %q, want node", probeSpec.Command)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func TestMapCodexAgentEventPreservesExtendedTraceFields(t *testing.T) {
	commandOutput, ok := mapCodexAgentEvent(codex.Event{
		Type: codex.EventTypeOutputProduced,
		Output: &codex.OutputEvent{
			ThreadID: "thread-1",
			TurnID:   "turn-1",
			ItemID:   "item-1",
			Stream:   "command",
			Command:  "pnpm vitest run",
			Text:     "PASS src/app.test.ts",
			Phase:    "running_command",
			Snapshot: true,
		},
	})
	if !ok || commandOutput.Output == nil {
		t.Fatalf("mapCodexAgentEvent(output) = %+v, %t", commandOutput, ok)
	}
	if commandOutput.Output.Command != "pnpm vitest run" {
		t.Fatalf("output command = %q, want pnpm vitest run", commandOutput.Output.Command)
	}

	toolCall, ok := mapCodexAgentEvent(codex.Event{
		Type: codex.EventTypeToolCallRequested,
		ToolCall: &codex.ToolCallRequest{
			ThreadID:  "thread-1",
			TurnID:    "turn-1",
			CallID:    "call-1",
			Tool:      "functions.exec_command",
			Arguments: json.RawMessage(`{"cmd":"pnpm vitest run","workdir":"/repo"}`),
		},
	})
	if !ok || toolCall.ToolCall == nil {
		t.Fatalf("mapCodexAgentEvent(tool call) = %+v, %t", toolCall, ok)
	}
	if string(toolCall.ToolCall.Arguments) != `{"cmd":"pnpm vitest run","workdir":"/repo"}` {
		t.Fatalf("tool call arguments = %s", toolCall.ToolCall.Arguments)
	}

	threadStatus, ok := mapCodexAgentEvent(codex.Event{
		Type: codex.EventTypeThreadStatus,
		ThreadStatus: &codex.ThreadStatusEvent{
			ThreadID:    "thread-1",
			Status:      "active",
			ActiveFlags: []string{"waitingOnUserInput"},
		},
	})
	if !ok || threadStatus.Thread == nil {
		t.Fatalf("mapCodexAgentEvent(thread status) = %+v, %t", threadStatus, ok)
	}
	if threadStatus.Thread.Status != "active" || len(threadStatus.Thread.ActiveFlags) != 1 {
		t.Fatalf("thread status payload = %+v", threadStatus.Thread)
	}

	diffUpdate, ok := mapCodexAgentEvent(codex.Event{
		Type: codex.EventTypeTurnDiffUpdated,
		Diff: &codex.TurnDiffEvent{
			ThreadID: "thread-1",
			TurnID:   "turn-1",
			Diff:     "diff --git a/app.ts b/app.ts\n@@ -1 +1 @@\n-old\n+new",
		},
	})
	if !ok || diffUpdate.Diff == nil || !strings.Contains(diffUpdate.Diff.Diff, "diff --git") {
		t.Fatalf("mapCodexAgentEvent(diff) = %+v, %t", diffUpdate, ok)
	}

	reasoningUpdate, ok := mapCodexAgentEvent(codex.Event{
		Type: codex.EventTypeReasoningUpdated,
		Reasoning: &codex.ReasoningEvent{
			ThreadID:     "thread-1",
			TurnID:       "turn-1",
			ItemID:       "reasoning-1",
			Kind:         codex.ReasoningKindText,
			Delta:        "Inspecting run transcript rendering.",
			ContentIndex: intPointer(2),
		},
	})
	if !ok || reasoningUpdate.Reasoning == nil {
		t.Fatalf("mapCodexAgentEvent(reasoning) = %+v, %t", reasoningUpdate, ok)
	}
	if reasoningUpdate.Reasoning.Delta != "Inspecting run transcript rendering." ||
		reasoningUpdate.Reasoning.ContentIndex == nil ||
		*reasoningUpdate.Reasoning.ContentIndex != 2 {
		t.Fatalf("reasoning payload = %+v", reasoningUpdate.Reasoning)
	}
}

func TestAgentPermissionProfileHelpers(t *testing.T) {
	if got := codexApprovalPolicy(catalogdomain.AgentProviderPermissionProfileStandard); got != "on-request" {
		t.Fatalf("codexApprovalPolicy(standard) = %q", got)
	}
	if got := codexSandboxMode(catalogdomain.AgentProviderPermissionProfileStandard); got != "workspace-write" {
		t.Fatalf("codexSandboxMode(standard) = %q", got)
	}
	if got := codexSandboxPolicy(catalogdomain.AgentProviderPermissionProfileStandard)["type"]; got != "workspaceWrite" {
		t.Fatalf("codexSandboxPolicy(standard) = %+v", codexSandboxPolicy(catalogdomain.AgentProviderPermissionProfileStandard))
	}
	if got := codexApprovalPolicy(""); got != "never" {
		t.Fatalf("codexApprovalPolicy(default) = %q", got)
	}
	if !hasClaudePermissionBypassArg([]string{"--permission-mode=bypassPermissions"}) {
		t.Fatal("hasClaudePermissionBypassArg() expected true")
	}
	if hasClaudePermissionBypassArg([]string{"--permission-mode", "default"}) {
		t.Fatal("hasClaudePermissionBypassArg() expected false")
	}
	if !hasClaudeEffortArg([]string{"--effort=high"}) {
		t.Fatal("hasClaudeEffortArg() expected true")
	}
	if hasClaudeEffortArg([]string{"--verbose"}) {
		t.Fatal("hasClaudeEffortArg() expected false")
	}
	if got := buildGeminiPermissionArgs(catalogdomain.AgentProviderPermissionProfileStandard); got != nil {
		t.Fatalf("buildGeminiPermissionArgs(standard) = %v", got)
	}
	if got := buildGeminiPermissionArgs(""); len(got) != 1 || got[0] != "--approval-mode=yolo" {
		t.Fatalf("buildGeminiPermissionArgs(default) = %v", got)
	}
}

func requireAgentEvent(t *testing.T, events <-chan agentEvent) agentEvent {
	t.Helper()

	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("agent event stream closed before next event")
		}
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for agent event")
		return agentEvent{}
	}
}

func intPointer(value int) *int {
	return &value
}

func reasoningEffortPointer(
	value catalogdomain.AgentProviderReasoningEffort,
) *catalogdomain.AgentProviderReasoningEffort {
	return &value
}

func runClaudeRuntimeProtocol(process *runtimeRunnerFakeProcess) error {
	reader := bufio.NewReader(process.stdinRead)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.Contains(line, `"type":"user"`) {
		return errors.New("expected Claude user input frame")
	}
	if _, err := io.WriteString(process.stdoutWrite, `{"type":"assistant","session_id":"claude-session-1","uuid":"assistant-1","message":{"content":[{"type":"text","text":"Implemented the shared contract."}]}}`+"\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(process.stdoutWrite, `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":1775037600,"rateLimitType":"five_hour","isUsingOverage":false}}`+"\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(process.stdoutWrite, `{"type":"result","subtype":"success","session_id":"claude-session-1","result":"done","num_turns":1}`+"\n"); err != nil {
		return err
	}
	process.finish(nil)
	return nil
}

func runClaudeTaskRuntimeProtocol(process *runtimeRunnerFakeProcess) error {
	reader := bufio.NewReader(process.stdinRead)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.Contains(line, `"type":"user"`) {
		return errors.New("expected Claude user input frame")
	}
	frames := []string{
		`{"type":"task_started","session_id":"claude-session-2","turn_id":"turn-claude-1","status":"in_progress","message":"Planning work"}`,
		`{"type":"task_progress","session_id":"claude-session-2","parent_tool_use_id":"tool-use-1","stream":"command","command":"pwd","text":"/repo\n","snapshot":true}`,
		`{"type":"system","subtype":"session_state_changed","session_id":"claude-session-2","event":{"state":"active","detail":"Running","active_flags":["running"]}}`,
		`{"type":"result","subtype":"error","session_id":"claude-session-2","is_error":true,"num_turns":1}`,
	}
	for _, frame := range frames {
		if _, err := io.WriteString(process.stdoutWrite, frame+"\n"); err != nil {
			return err
		}
	}
	process.finish(nil)
	return nil
}

func runClaudeAssistantSnapshotProtocol(process *runtimeRunnerFakeProcess) error {
	reader := bufio.NewReader(process.stdinRead)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.Contains(line, `"type":"user"`) {
		return errors.New("expected Claude user input frame")
	}
	frames := []string{
		`{"type":"assistant","session_id":"claude-session-3","message":{"content":[{"type":"text","text":"Draft PRD"}]}}`,
		`{"type":"assistant","session_id":"claude-session-3","message":{"content":[{"type":"text","text":"Draft PRD with more detail"}]}}`,
		`{"type":"assistant","session_id":"claude-session-3","message":{"content":[{"type":"text","text":"Now updating the ticket"}]}}`,
		`{"type":"result","subtype":"success","session_id":"claude-session-3","num_turns":1}`,
	}
	for _, frame := range frames {
		if _, err := io.WriteString(process.stdoutWrite, frame+"\n"); err != nil {
			return err
		}
	}
	process.finish(nil)
	return nil
}

func runClaudeTaskProgressSnapshotProtocol(process *runtimeRunnerFakeProcess) error {
	reader := bufio.NewReader(process.stdinRead)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.Contains(line, `"type":"user"`) {
		return errors.New("expected Claude user input frame")
	}
	frames := []string{
		`{"type":"task_progress","session_id":"claude-session-4","stream":"command","command":"pwd","text":"/repo","snapshot":true}`,
		`{"type":"task_progress","session_id":"claude-session-4","stream":"command","command":"pwd","text":"/repo\n/home","snapshot":true}`,
		`{"type":"task_progress","session_id":"claude-session-4","stream":"command","command":"git status","text":"On branch main","snapshot":true}`,
		`{"type":"result","subtype":"success","session_id":"claude-session-4","num_turns":1}`,
	}
	for _, frame := range frames {
		if _, err := io.WriteString(process.stdoutWrite, frame+"\n"); err != nil {
			return err
		}
	}
	process.finish(nil)
	return nil
}

func runClaudeToolUseRuntimeProtocol(process *runtimeRunnerFakeProcess) error {
	reader := bufio.NewReader(process.stdinRead)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.Contains(line, `"type":"user"`) {
		return errors.New("expected Claude user input frame")
	}
	frames := []string{
		`{"type":"assistant","session_id":"claude-session-5","message":{"content":[{"type":"text","text":"Let me inspect the repository first."},{"type":"tool_use","id":"toolu_01","name":"functions.exec_command","input":{"cmd":"git status --short"}}]}}`,
		`{"type":"user","session_id":"claude-session-5","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_01","content":"M README.md"}]}}`,
		`{"type":"result","subtype":"success","session_id":"claude-session-5","num_turns":1}`,
	}
	for _, frame := range frames {
		if _, err := io.WriteString(process.stdoutWrite, frame+"\n"); err != nil {
			return err
		}
	}
	process.finish(nil)
	return nil
}

func runClaudeToolResultDiffRuntimeProtocol(process *runtimeRunnerFakeProcess) error {
	reader := bufio.NewReader(process.stdinRead)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.Contains(line, `"type":"user"`) {
		return errors.New("expected Claude user input frame")
	}
	frames := []string{
		`{"type":"assistant","session_id":"claude-session-6","message":{"content":[{"type":"tool_use","id":"toolu_diff","name":"functions.exec_command","input":{"cmd":"git diff -- README.md"}}]}}`,
		"{\"type\":\"user\",\"session_id\":\"claude-session-6\",\"message\":{\"content\":[{\"type\":\"tool_result\",\"tool_use_id\":\"toolu_diff\",\"content\":\"diff --git a/README.md b/README.md\\n@@ -1 +1 @@\\n-old\\n+new\"}]}}",
		`{"type":"result","subtype":"success","session_id":"claude-session-6","num_turns":1}`,
	}
	for _, frame := range frames {
		if _, err := io.WriteString(process.stdoutWrite, frame+"\n"); err != nil {
			return err
		}
	}
	process.finish(nil)
	return nil
}

type geminiAdapterTestProcessManager struct {
	processes     []provider.AgentCLIProcess
	capturedSpecs []provider.AgentCLIProcessSpec
}

func (m *geminiAdapterTestProcessManager) Start(_ context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	m.capturedSpecs = append(m.capturedSpecs, spec)
	if len(m.processes) == 0 {
		return nil, errors.New("no fake process configured")
	}
	process := m.processes[0]
	m.processes = m.processes[1:]
	return process, nil
}

func runGeminiRuntimeProtocol(process *runtimeRunnerFakeProcess) error {
	if _, err := io.ReadAll(process.stdinRead); err != nil {
		return err
	}
	if _, err := io.WriteString(process.stdoutWrite, `{
		"response":"Implemented the shared contract.",
		"stats":{
			"models":{
				"gemini-2.5-pro":{
					"api":{"totalRequests":1,"totalErrors":0,"totalLatencyMs":1200},
					"tokens":{"input":120,"prompt":120,"candidates":35,"total":155,"cached":0,"thoughts":0,"tool":0}
				}
			}
		}
	}`); err != nil {
		return err
	}
	process.finish(nil)
	return nil
}

type stubGeminiProbeProcess struct {
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (p *stubGeminiProbeProcess) PID() int              { return 8181 }
func (p *stubGeminiProbeProcess) Stdin() io.WriteCloser { return nopWriteCloser{} }
func (p *stubGeminiProbeProcess) Stdout() io.ReadCloser { return p.stdout }
func (p *stubGeminiProbeProcess) Stderr() io.ReadCloser { return p.stderr }
func (p *stubGeminiProbeProcess) Wait() error           { return nil }
func (p *stubGeminiProbeProcess) Stop(context.Context) error {
	return nil
}

type nopWriteCloser struct{}

func (nopWriteCloser) Write(data []byte) (int, error) { return len(data), nil }
func (nopWriteCloser) Close() error                   { return nil }
