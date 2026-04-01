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

	event := requireAgentEvent(t, session.Events())
	if event.Type != agentEventTypeTurnCompleted || event.Turn == nil || event.Turn.TurnID != "turn-contract" {
		t.Fatalf("unexpected event: %+v", event)
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
	second := requireAgentEvent(t, session.Events())
	if second.Type != agentEventTypeTurnCompleted || second.Turn == nil || second.Turn.Status != "completed" {
		t.Fatalf("unexpected second event: %+v", second)
	}

	sessionID, ok := session.SessionID()
	if !ok || sessionID != "claude-session-1" {
		t.Fatalf("SessionID() = %q, %t", sessionID, ok)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestDefaultAgentAdapterRegistryRegistersGeminiRuntimeContract(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	manager := &geminiAdapterTestProcessManager{process: process}

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
	if second.Type != agentEventTypeTurnCompleted || second.Turn == nil || second.Turn.Status != "completed" || second.Turn.TurnID != turn.TurnID {
		t.Fatalf("unexpected second event: %+v", second)
	}

	if manager.capturedSpec.Command != provider.MustParseAgentCLICommand("gemini") {
		t.Fatalf("process command = %q, want gemini", manager.capturedSpec.Command)
	}
	joinedArgs := strings.Join(manager.capturedSpec.Args, " ")
	if !strings.Contains(joinedArgs, "-m gemini-2.5-pro") {
		t.Fatalf("process args = %v, want model flag", manager.capturedSpec.Args)
	}
	if !strings.Contains(joinedArgs, "--output-format json") {
		t.Fatalf("process args = %v, want json output mode", manager.capturedSpec.Args)
	}
	if !strings.Contains(joinedArgs, "--approval-mode=yolo") {
		t.Fatalf("process args = %v, want yolo approval mode", manager.capturedSpec.Args)
	}
	if !strings.Contains(joinedArgs, "-p") {
		t.Fatalf("process args = %v, want prompt flag", manager.capturedSpec.Args)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
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

func runClaudeRuntimeProtocol(process *runtimeRunnerFakeProcess) error {
	reader := bufio.NewReader(process.stdinRead)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.Contains(line, `"type":"user"`) {
		return errors.New("expected Claude user input frame")
	}
	if _, err := io.WriteString(process.stdoutWrite, `{"type":"assistant","session_id":"claude-session-1","message":{"content":[{"type":"text","text":"Implemented the shared contract."}]}}`+"\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(process.stdoutWrite, `{"type":"result","subtype":"success","session_id":"claude-session-1","result":"done","num_turns":1}`+"\n"); err != nil {
		return err
	}
	process.finish(nil)
	return nil
}

type geminiAdapterTestProcessManager struct {
	process      provider.AgentCLIProcess
	capturedSpec provider.AgentCLIProcessSpec
}

func (m *geminiAdapterTestProcessManager) Start(_ context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	m.capturedSpec = spec
	return m.process, nil
}

func runGeminiRuntimeProtocol(process *runtimeRunnerFakeProcess) error {
	if _, err := io.ReadAll(process.stdinRead); err != nil {
		return err
	}
	if _, err := io.WriteString(process.stdoutWrite, `{"response":"Implemented the shared contract."}`); err != nil {
		return err
	}
	process.finish(nil)
	return nil
}
