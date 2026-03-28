package chat

import (
	"context"
	"io"
	"strings"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
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

func TestGeminiRuntimeStartTurnPromotesActionProposalJSON(t *testing.T) {
	manager := &fakeAgentCLIProcessManager{
		process: &fakeAgentCLIProcess{
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
	if done.SessionID != "session-gemini-1" || done.TurnsUsed != 1 || done.TurnsRemaining != DefaultMaxTurns-1 {
		t.Fatalf("unexpected done payload: %#v", done)
	}

	if manager.startSpec.Command != provider.MustParseAgentCLICommand("gemini") {
		t.Fatalf("process command = %q, want gemini", manager.startSpec.Command)
	}
	if joined := strings.Join(manager.startSpec.Args, " "); !strings.Contains(joined, "--output-format json") {
		t.Fatalf("process args = %v, want json output mode", manager.startSpec.Args)
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
	stdout  string
	stderr  string
	waitErr error
}

func (p *fakeAgentCLIProcess) PID() int { return 4242 }

func (p *fakeAgentCLIProcess) Stdin() io.WriteCloser { return nopWriteCloser{} }

func (p *fakeAgentCLIProcess) Stdout() io.ReadCloser {
	return io.NopCloser(strings.NewReader(p.stdout))
}

func (p *fakeAgentCLIProcess) Stderr() io.ReadCloser {
	return io.NopCloser(strings.NewReader(p.stderr))
}

func (p *fakeAgentCLIProcess) Wait() error { return p.waitErr }

func (p *fakeAgentCLIProcess) Stop(context.Context) error { return nil }

type nopWriteCloser struct{}

func (nopWriteCloser) Write(data []byte) (int, error) { return len(data), nil }

func (nopWriteCloser) Close() error { return nil }
