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
	if third.Type != agentEventTypeRateLimitUpdated || third.RateLimit == nil || third.RateLimit.Gemini == nil {
		t.Fatalf("unexpected third event: %+v", third)
	}
	fourth := requireAgentEvent(t, session.Events())
	if fourth.Type != agentEventTypeTurnCompleted || fourth.Turn == nil || fourth.Turn.Status != "completed" || fourth.Turn.TurnID != turn.TurnID {
		t.Fatalf("unexpected fourth event: %+v", fourth)
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
	if _, err := io.WriteString(process.stdoutWrite, `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":1775037600,"rateLimitType":"five_hour","isUsingOverage":false}}`+"\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(process.stdoutWrite, `{"type":"result","subtype":"success","session_id":"claude-session-1","result":"done","num_turns":1}`+"\n"); err != nil {
		return err
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
