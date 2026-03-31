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
}

func (w *trackingWriteCloser) Write(data []byte) (int, error) { return len(data), nil }

func (w *trackingWriteCloser) Close() error {
	w.closed = true
	return nil
}

type nopWriteCloser struct{}

func (nopWriteCloser) Write(data []byte) (int, error) { return len(data), nil }

func (nopWriteCloser) Close() error { return nil }

func int64Pointer(value int64) *int64 {
	return &value
}
