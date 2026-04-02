package chat

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	codexadapter "github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestMapCodexAssistantOutputPromotesActionProposalFromSnapshot(t *testing.T) {
	items := make(map[string]*codexAssistantItemState)

	events := mapCodexAssistantOutput(&codexadapter.OutputEvent{
		ItemID: "item-1",
		Stream: "assistant",
		Text:   "```json\n{\"type\":\"action_proposal\",",
	}, items)
	if len(events) != 0 {
		t.Fatalf("first assistant delta should be buffered, got %+v", events)
	}

	events = mapCodexAssistantOutput(&codexadapter.OutputEvent{
		ItemID:   "item-1",
		Stream:   "assistant",
		Text:     "\"summary\":\"Create child ticket\",\"actions\":[{\"method\":\"POST\",\"path\":\"/api/v1/projects/p/tickets\"}]}\n```",
		Snapshot: true,
	}, items)
	if len(events) != 1 {
		t.Fatalf("snapshot should emit one normalized event, got %d", len(events))
	}

	payload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload = %#v, want action proposal payload map", events[0].Payload)
	}
	if payload["type"] != chatMessageTypeActionProposal || payload["summary"] != "Create child ticket" {
		t.Fatalf("unexpected action proposal payload: %#v", payload)
	}
}

func TestMapCodexAssistantOutputPrefersSnapshotTextForJSONResponses(t *testing.T) {
	items := make(map[string]*codexAssistantItemState)

	events := mapCodexAssistantOutput(&codexadapter.OutputEvent{
		ItemID: "item-1",
		Stream: "assistant",
		Text:   "{\"type\":\"action_proposal\",\"summary\":\"Create child ticket\",\"actions\":[{\"method\":\"POST\",\"path\":\"/api/v1/projects/p/tickets\"}]}",
	}, items)
	if len(events) != 0 {
		t.Fatalf("first assistant delta should be buffered, got %+v", events)
	}

	events = mapCodexAssistantOutput(&codexadapter.OutputEvent{
		ItemID:   "item-1",
		Stream:   "assistant",
		Text:     "{\"type\":\"action_proposal\",\"summary\":\"Create child ticket\",\"actions\":[{\"method\":\"POST\",\"path\":\"/api/v1/projects/p/tickets\"}]}",
		Snapshot: true,
	}, items)
	if len(events) != 1 {
		t.Fatalf("snapshot should emit one normalized event, got %d", len(events))
	}

	payload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("payload = %#v, want action proposal payload map", events[0].Payload)
	}
	if payload["type"] != chatMessageTypeActionProposal || payload["summary"] != "Create child ticket" {
		t.Fatalf("unexpected action proposal payload: %#v", payload)
	}
}

func TestMapCodexAssistantOutputPromotesDiffFromSnapshot(t *testing.T) {
	items := make(map[string]*codexAssistantItemState)

	events := mapCodexAssistantOutput(&codexadapter.OutputEvent{
		ItemID: "item-1",
		Stream: "assistant",
		Text:   "```json\n{\"type\":\"diff\",",
	}, items)
	if len(events) != 0 {
		t.Fatalf("first assistant delta should be buffered, got %+v", events)
	}

	events = mapCodexAssistantOutput(&codexadapter.OutputEvent{
		ItemID:   "item-1",
		Stream:   "assistant",
		Text:     "\"file\":\"harness content\",\"hunks\":[{\"old_start\":1,\"old_lines\":1,\"new_start\":1,\"new_lines\":2,\"lines\":[{\"op\":\"context\",\"text\":\"---\"},{\"op\":\"add\",\"text\":\"new line\"}]}]}\n```",
		Snapshot: true,
	}, items)
	if len(events) != 1 {
		t.Fatalf("snapshot should emit one normalized event, got %d", len(events))
	}

	payload, ok := events[0].Payload.(diffPayload)
	if !ok {
		t.Fatalf("payload = %#v, want diff payload", events[0].Payload)
	}
	if payload.Type != chatMessageTypeDiff || payload.File != "harness content" {
		t.Fatalf("unexpected diff payload: %#v", payload)
	}
}

func TestGeminiRuntimeStartTurnPromotesActionProposalJSON(t *testing.T) {
	stdin := &trackingWriteCloser{}
	manager := &fakeAgentCLIProcessManager{
		process: &fakeAgentCLIProcess{
			stdin:  stdin,
			stdout: "{\"response\":\"```json\\n{\\\"type\\\":\\\"action_proposal\\\",\\\"summary\\\":\\\"Create 2 tickets\\\",\\\"actions\\\":[{\\\"method\\\":\\\"POST\\\",\\\"path\\\":\\\"/api/v1/projects/p/tickets\\\"}]}\\n```\"}",
		},
	}
	runtime := NewGeminiRuntime(manager)

	stream, err := runtime.StartTurn(context.Background(), RuntimeTurnInput{
		SessionID:        SessionID("session-gemini-1"),
		Message:          "Split this into two tickets",
		SystemPrompt:     "You are OpenASE.",
		MaxTurns:         DefaultMaxTurns,
		WorkingDirectory: provider.MustParseAbsolutePath("/tmp/openase"),
		Provider: catalogdomain.AgentProvider{
			AdapterType: catalogdomain.AgentProviderAdapterTypeGeminiCLI,
			CliCommand:  "gemini",
		},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}

	events := collectStreamEvents(stream.Events)
	if len(events) != 2 {
		t.Fatalf("stream event count = %d, want 2: %+v", len(events), events)
	}

	payload, ok := events[0].Payload.(map[string]any)
	if events[0].Event != "message" || !ok {
		t.Fatalf("first event = %+v, want normalized message", events[0])
	}
	if payload["type"] != chatMessageTypeActionProposal || payload["summary"] != "Create 2 tickets" {
		t.Fatalf("unexpected action proposal payload: %#v", payload)
	}

	done, ok := events[1].Payload.(donePayload)
	if events[1].Event != "done" || !ok {
		t.Fatalf("second event = %+v, want done payload", events[1])
	}
	if done.SessionID != "session-gemini-1" || done.TurnsUsed != 1 || done.TurnsRemaining == nil || *done.TurnsRemaining != DefaultMaxTurns-1 {
		t.Fatalf("unexpected done payload: %#v", done)
	}
	if done.CostUSD != nil {
		t.Fatalf("done cost = %#v, want nil spend-unavailable payload", done.CostUSD)
	}

	if manager.startSpec.Command != provider.MustParseAgentCLICommand("gemini") {
		t.Fatalf("process command = %q, want gemini", manager.startSpec.Command)
	}
	if joined := strings.Join(manager.startSpec.Args, " "); !strings.Contains(joined, "--output-format json") {
		t.Fatalf("process args = %v, want json output mode", manager.startSpec.Args)
	}
	if !stdin.closed {
		t.Fatal("expected gemini stdin to be closed after start")
	}
}

func TestGeminiRuntimeStartTurnPromotesDiffJSON(t *testing.T) {
	stdin := &trackingWriteCloser{}
	manager := &fakeAgentCLIProcessManager{
		process: &fakeAgentCLIProcess{
			stdin:  stdin,
			stdout: "{\"response\":\"```json\\n{\\\"type\\\":\\\"diff\\\",\\\"file\\\":\\\"harness content\\\",\\\"hunks\\\":[{\\\"old_start\\\":1,\\\"old_lines\\\":1,\\\"new_start\\\":1,\\\"new_lines\\\":2,\\\"lines\\\":[{\\\"op\\\":\\\"context\\\",\\\"text\\\":\\\"---\\\"},{\\\"op\\\":\\\"add\\\",\\\"text\\\":\\\"new line\\\"}]}]}\\n```\"}",
		},
	}
	runtime := NewGeminiRuntime(manager)

	stream, err := runtime.StartTurn(context.Background(), RuntimeTurnInput{
		SessionID:        SessionID("session-gemini-diff-1"),
		Message:          "Tighten this harness",
		SystemPrompt:     "You are OpenASE.",
		MaxTurns:         DefaultMaxTurns,
		WorkingDirectory: provider.MustParseAbsolutePath("/tmp/openase"),
		Provider: catalogdomain.AgentProvider{
			AdapterType: catalogdomain.AgentProviderAdapterTypeGeminiCLI,
			CliCommand:  "gemini",
		},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}

	events := collectStreamEvents(stream.Events)
	if len(events) != 2 {
		t.Fatalf("stream event count = %d, want 2: %+v", len(events), events)
	}

	payload, ok := events[0].Payload.(diffPayload)
	if events[0].Event != "message" || !ok {
		t.Fatalf("first event = %+v, want normalized diff message", events[0])
	}
	if payload.Type != chatMessageTypeDiff || payload.File != "harness content" {
		t.Fatalf("unexpected diff payload: %#v", payload)
	}
}

func TestResolveUsageCostUSDUsesConfiguredProviderPricing(t *testing.T) {
	costUSD := resolveUsageCostUSD(
		catalogdomain.AgentProvider{
			CostPerInputToken:  0.001,
			CostPerOutputToken: 0.002,
		},
		ticketing.RawUsageDelta{
			InputTokens:  int64Pointer(120),
			OutputTokens: int64Pointer(35),
		},
	)
	if costUSD == nil {
		t.Fatal("resolveUsageCostUSD() = nil, want computed cost")
	}
	if *costUSD != 0.19 {
		t.Fatalf("resolveUsageCostUSD() = %.2f, want 0.19", *costUSD)
	}
}

func TestResolveUsageCostUSDReturnsNilWithoutConfiguredPricing(t *testing.T) {
	costUSD := resolveUsageCostUSD(
		catalogdomain.AgentProvider{},
		ticketing.RawUsageDelta{
			InputTokens:  int64Pointer(120),
			OutputTokens: int64Pointer(35),
		},
	)
	if costUSD != nil {
		t.Fatalf("resolveUsageCostUSD() = %.2f, want nil", *costUSD)
	}
}

func TestMapClaudeEventDoneIncludesProviderReportedCost(t *testing.T) {
	costUSD := 0.37
	events := mapClaudeEvent(SessionID("session-claude-1"), DefaultMaxTurns, provider.ClaudeCodeEvent{
		Kind:         provider.ClaudeCodeEventKindResult,
		NumTurns:     2,
		TotalCostUSD: &costUSD,
	})
	if len(events) != 1 {
		t.Fatalf("mapClaudeEvent() len = %d, want 1", len(events))
	}

	done, ok := events[0].Payload.(donePayload)
	if !ok {
		t.Fatalf("payload = %#v, want done payload", events[0].Payload)
	}
	if done.CostUSD == nil || *done.CostUSD != costUSD {
		t.Fatalf("done cost = %#v, want %.2f", done.CostUSD, costUSD)
	}
}

func TestMapClaudeEventPromotesSessionStateChanges(t *testing.T) {
	events := mapClaudeEvent(SessionID("session-claude-1"), DefaultMaxTurns, provider.ClaudeCodeEvent{
		Kind:    provider.ClaudeCodeEventKindStream,
		Subtype: "session_state_changed",
		Event: mustMarshalJSON(t, map[string]any{
			"state":        "requires_action",
			"active_flags": []string{"requires_action"},
			"detail":       "approval required",
		}),
	})
	if len(events) != 2 {
		t.Fatalf("mapClaudeEvent() len = %d, want 2", len(events))
	}
	if events[0].Event != "session_state" {
		t.Fatalf("event kind = %q, want session_state", events[0].Event)
	}
	payload, ok := events[0].Payload.(runtimeSessionStatePayload)
	if !ok {
		t.Fatalf("payload = %#v, want runtimeSessionStatePayload", events[0].Payload)
	}
	if payload.Status != "requires_action" || payload.Detail != "approval required" {
		t.Fatalf("unexpected session state payload: %#v", payload)
	}
	if len(payload.ActiveFlags) != 1 || payload.ActiveFlags[0] != "requires_action" {
		t.Fatalf("unexpected session state flags: %#v", payload.ActiveFlags)
	}
}

func TestMapClaudeEventPromotesRequiresActionInterrupts(t *testing.T) {
	events := mapClaudeEvent(SessionID("session-claude-1"), DefaultMaxTurns, provider.ClaudeCodeEvent{
		Kind:    provider.ClaudeCodeEventKindStream,
		Subtype: "session_state_changed",
		Event: mustMarshalJSON(t, map[string]any{
			"state": "requires_action",
			"requires_action": map[string]any{
				"request_id": "claude-req-1",
				"type":       "approval",
				"detail":     "command approval required",
				"options": []map[string]any{
					{"id": "approve_once", "label": "Approve once"},
					{"id": "deny", "label": "Deny"},
				},
			},
		}),
	})
	if len(events) != 2 {
		t.Fatalf("mapClaudeEvent() len = %d, want 2", len(events))
	}
	interrupt, ok := events[1].Payload.(RuntimeInterruptEvent)
	if events[1].Event != "interrupt_requested" || !ok {
		t.Fatalf("second event = %+v, want interrupt_requested payload", events[1])
	}
	if interrupt.RequestID != "claude-req-1" || interrupt.Kind != "command_execution" {
		t.Fatalf("unexpected interrupt payload: %#v", interrupt)
	}
	if len(interrupt.Options) != 2 || interrupt.Options[0].ID != "approve_once" {
		t.Fatalf("unexpected interrupt options: %#v", interrupt.Options)
	}
	if interrupt.Payload["session_state"] != "requires_action" {
		t.Fatalf("unexpected interrupt payload map: %#v", interrupt.Payload)
	}
}

func TestClaudeRuntimeStartTurnUsesPersistentResumeSessionIDAndEmitsSessionAnchor(t *testing.T) {
	stdin := &trackingWriteCloser{}
	manager := &fakeAgentCLIProcessManager{
		process: &fakeAgentCLIProcess{
			stdin: stdin,
			stdout: strings.Join([]string{
				`{"type":"system","subtype":"init","data":{"session_id":"claude-session-42"}}`,
				`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"OK"}]}}`,
				`{"type":"result","subtype":"success","session_id":"claude-session-42","num_turns":1}`,
			}, "\n"),
		},
	}
	runtime := NewClaudeRuntime(newClaudeAdapterForManager(manager))

	stream, err := runtime.StartTurn(context.Background(), RuntimeTurnInput{
		SessionID:              SessionID("session-claude-runtime"),
		Message:                "Resume this project conversation",
		SystemPrompt:           "You are OpenASE.",
		ResumeProviderThreadID: "claude-session-existing",
		PersistentConversation: true,
		Provider: catalogdomain.AgentProvider{
			AdapterType: catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
			CliCommand:  "claude",
		},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}

	if joined := strings.Join(manager.startSpec.Args, " "); !strings.Contains(joined, "--resume claude-session-existing") {
		t.Fatalf("process args = %v, want durable --resume anchor", manager.startSpec.Args)
	}
	if !strings.Contains(strings.Join(manager.startSpec.Environment, "\n"), claudeCodeResumeInterruptedTurnEnv+"=1") {
		t.Fatalf("process environment = %v, want %s=1", manager.startSpec.Environment, claudeCodeResumeInterruptedTurnEnv)
	}

	events := collectStreamEvents(stream.Events)
	if len(events) != 3 {
		t.Fatalf("stream event count = %d, want 3: %+v", len(events), events)
	}

	anchor, ok := events[0].Payload.(RuntimeSessionAnchor)
	if events[0].Event != "session_anchor" || !ok {
		t.Fatalf("first event = %+v, want session anchor payload", events[0])
	}
	if anchor.ProviderThreadID != "claude-session-42" {
		t.Fatalf("anchor provider thread id = %q, want claude-session-42", anchor.ProviderThreadID)
	}

	done, ok := events[2].Payload.(donePayload)
	if events[2].Event != "done" || !ok {
		t.Fatalf("last event = %+v, want done payload", events[2])
	}
	if done.SessionID != "session-claude-runtime" || done.TurnsUsed != 1 {
		t.Fatalf("unexpected done payload: %#v", done)
	}

	resolved := runtime.SessionAnchor(SessionID("session-claude-runtime"))
	if resolved.ProviderThreadID != "claude-session-42" {
		t.Fatalf("resolved session anchor = %+v, want provider thread claude-session-42", resolved)
	}
}

func TestClaudeRuntimeRespondInterruptResumesSessionAndStreamsContinuation(t *testing.T) {
	stdin := &trackingWriteCloser{}
	manager := &fakeAgentCLIProcessManager{
		process: &fakeAgentCLIProcess{
			stdin: stdin,
			stdout: strings.Join([]string{
				`{"type":"system","subtype":"init","data":{"session_id":"claude-session-55"}}`,
				`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Continuing after approval"}]}}`,
				`{"type":"result","subtype":"success","session_id":"claude-session-55","num_turns":3}`,
			}, "\n"),
		},
	}
	runtime := NewClaudeRuntime(newClaudeAdapterForManager(manager))

	stream, err := runtime.RespondInterrupt(context.Background(), RuntimeInterruptResponseInput{
		SessionID:              SessionID("session-claude-runtime"),
		Provider:               catalogdomain.AgentProvider{AdapterType: catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI, CliCommand: "claude"},
		RequestID:              "claude-req-55",
		Kind:                   "command_execution",
		Decision:               "approve_once",
		Payload:                map[string]any{"provider": "claude", "payload": map[string]any{"detail": "approval required"}},
		ResumeProviderThreadID: "claude-session-55",
		PersistentConversation: true,
	})
	if err != nil {
		t.Fatalf("RespondInterrupt() error = %v", err)
	}

	if joined := strings.Join(manager.startSpec.Args, " "); !strings.Contains(joined, "--resume claude-session-55") {
		t.Fatalf("process args = %v, want durable --resume anchor", manager.startSpec.Args)
	}
	if !strings.Contains(strings.Join(manager.startSpec.Environment, "\n"), claudeCodeResumeInterruptedTurnEnv+"=1") {
		t.Fatalf("process environment = %v, want %s=1", manager.startSpec.Environment, claudeCodeResumeInterruptedTurnEnv)
	}
	if !strings.Contains(stdin.String(), "approve_once") || !strings.Contains(stdin.String(), "claude-req-55") {
		t.Fatalf("stdin payload = %q, want interrupt response prompt", stdin.String())
	}

	events := collectStreamEvents(stream.Events)
	if len(events) != 3 {
		t.Fatalf("stream event count = %d, want 3: %+v", len(events), events)
	}
	if events[1].Event != "message" {
		t.Fatalf("second event = %+v, want assistant message", events[1])
	}
	done, ok := events[2].Payload.(donePayload)
	if events[2].Event != "done" || !ok {
		t.Fatalf("last event = %+v, want done payload", events[2])
	}
	if done.SessionID != "session-claude-runtime" || done.TurnsUsed != 3 {
		t.Fatalf("unexpected done payload: %#v", done)
	}
}

func TestGeminiRuntimeCloseSessionStopsProcess(t *testing.T) {
	process := &fakeAgentCLIProcess{
		stdin:         &trackingWriteCloser{},
		stdout:        "{\"response\":\"\"}",
		waitUntilStop: true,
		stopped:       make(chan struct{}),
		stopCalled:    make(chan struct{}),
	}
	manager := &fakeAgentCLIProcessManager{process: process}
	runtime := NewGeminiRuntime(manager)

	stream, err := runtime.StartTurn(context.Background(), RuntimeTurnInput{
		SessionID:    SessionID("session-gemini-stop"),
		Message:      "Stop this turn",
		SystemPrompt: "You are OpenASE.",
		Provider: catalogdomain.AgentProvider{
			AdapterType: catalogdomain.AgentProviderAdapterTypeGeminiCLI,
			CliCommand:  "gemini",
		},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}

	if closed := runtime.CloseSession(SessionID("session-gemini-stop")); !closed {
		t.Fatal("CloseSession() = false, want true")
	}

	select {
	case <-process.stopCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected CloseSession to stop the running gemini process")
	}

	select {
	case _, ok := <-stream.Events:
		if ok {
			for range stream.Events {
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected event stream to close after session shutdown")
	}
}

func TestGeminiRuntimeDoneIncludesProviderPricedUsageCost(t *testing.T) {
	stdin := &trackingWriteCloser{}
	manager := &fakeAgentCLIProcessManager{
		process: &fakeAgentCLIProcess{
			stdin: stdin,
			stdout: `{
				"response":"OK",
				"stats":{
					"models":{
						"gemini-2.5-pro":{
							"api":{"totalRequests":1,"totalErrors":0,"totalLatencyMs":900},
							"tokens":{"input":120,"prompt":120,"candidates":35,"total":155,"cached":0,"thoughts":0,"tool":0}
						}
					}
				}
			}`,
		},
	}
	runtime := NewGeminiRuntime(manager)

	stream, err := runtime.StartTurn(context.Background(), RuntimeTurnInput{
		SessionID:        SessionID("session-gemini-cost"),
		Message:          "Reply with exactly OK.",
		SystemPrompt:     "You are OpenASE.",
		MaxTurns:         DefaultMaxTurns,
		WorkingDirectory: provider.MustParseAbsolutePath("/tmp/openase"),
		Provider: catalogdomain.AgentProvider{
			AdapterType:        catalogdomain.AgentProviderAdapterTypeGeminiCLI,
			CliCommand:         "gemini",
			CostPerInputToken:  0.001,
			CostPerOutputToken: 0.002,
		},
	})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}

	events := collectStreamEvents(stream.Events)
	if len(events) != 2 {
		t.Fatalf("stream event count = %d, want 2: %+v", len(events), events)
	}

	done, ok := events[1].Payload.(donePayload)
	if events[1].Event != "done" || !ok {
		t.Fatalf("last event = %+v, want done payload", events[1])
	}
	if done.CostUSD == nil || *done.CostUSD != 0.19 {
		t.Fatalf("done cost = %#v, want 0.19", done.CostUSD)
	}
}

type fakeAgentCLIProcessManager struct {
	process   provider.AgentCLIProcess
	startSpec provider.AgentCLIProcessSpec
}

func (m *fakeAgentCLIProcessManager) Start(_ context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	m.startSpec = spec
	return m.process, nil
}

type fakeAgentCLIProcess struct {
	stdin         io.WriteCloser
	stdout        string
	stderr        string
	waitErr       error
	waitUntilStop bool
	stopped       chan struct{}
	stopCalled    chan struct{}
}

func (p *fakeAgentCLIProcess) PID() int { return 4242 }

func (p *fakeAgentCLIProcess) Stdin() io.WriteCloser {
	if p.stdin != nil {
		return p.stdin
	}
	return nopWriteCloser{}
}

func (p *fakeAgentCLIProcess) Stdout() io.ReadCloser {
	return io.NopCloser(strings.NewReader(p.stdout))
}

func (p *fakeAgentCLIProcess) Stderr() io.ReadCloser {
	return io.NopCloser(strings.NewReader(p.stderr))
}

func (p *fakeAgentCLIProcess) Wait() error {
	if p.waitUntilStop {
		if p.stopped == nil {
			p.stopped = make(chan struct{})
		}
		<-p.stopped
	}
	return p.waitErr
}

func (p *fakeAgentCLIProcess) Stop(context.Context) error {
	if p.stopCalled != nil {
		select {
		case <-p.stopCalled:
		default:
			close(p.stopCalled)
		}
	}
	if p.stopped != nil {
		select {
		case <-p.stopped:
		default:
			close(p.stopped)
		}
	}
	return nil
}

type trackingWriteCloser struct {
	closed bool
	writes strings.Builder
}

func (w *trackingWriteCloser) Write(data []byte) (int, error) {
	_, _ = w.writes.Write(data)
	return len(data), nil
}

func (w *trackingWriteCloser) Close() error {
	w.closed = true
	return nil
}

func (w *trackingWriteCloser) String() string {
	return w.writes.String()
}

type nopWriteCloser struct{}

func (nopWriteCloser) Write(data []byte) (int, error) { return len(data), nil }

func (nopWriteCloser) Close() error { return nil }
