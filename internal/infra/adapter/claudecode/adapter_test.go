package claudecode

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestAdapterStartBuildsStreamJSONCommandAndResume(t *testing.T) {
	manager := &fakeProcessManager{
		process: newFakeProcess("", ""),
	}
	adapter := NewAdapter(manager)
	maxTurns := 20
	maxBudgetUSD := 5.0
	resumeID := provider.MustParseClaudeCodeSessionID("sess-123")

	spec, err := provider.NewClaudeCodeSessionSpec(
		provider.MustParseAgentCLICommand("claude"),
		[]string{"--model", "claude-sonnet-4-6"},
		nil,
		[]string{"ANTHROPIC_API_KEY=test"},
		[]string{"Bash", "Read", "Edit"},
		"Follow the workflow harness.",
		&maxTurns,
		&maxBudgetUSD,
		&resumeID,
		true,
	)
	if err != nil {
		t.Fatalf("NewClaudeCodeSessionSpec returned error: %v", err)
	}

	session, err := adapter.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = session.Close(closeCtx)
	})

	if manager.lastSpec.Command != provider.MustParseAgentCLICommand("claude") {
		t.Fatalf("expected command claude, got %q", manager.lastSpec.Command)
	}

	expectedArgs := []string{
		"--model", "claude-sonnet-4-6",
		"-p",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--include-partial-messages",
		"--resume", "sess-123",
		"--allowedTools", "Bash,Read,Edit",
		"--max-turns", "20",
		"--max-budget-usd", "5",
		"--append-system-prompt", "Follow the workflow harness.",
	}
	if strings.Join(manager.lastSpec.Args, "\n") != strings.Join(expectedArgs, "\n") {
		t.Fatalf("unexpected args:\nwant: %#v\ngot:  %#v", expectedArgs, manager.lastSpec.Args)
	}
	if len(manager.lastSpec.Environment) != 1 || manager.lastSpec.Environment[0] != "ANTHROPIC_API_KEY=test" {
		t.Fatalf("unexpected environment: %#v", manager.lastSpec.Environment)
	}
}

func TestSessionSendEncodesStreamJSONUserTurns(t *testing.T) {
	process := newFakeProcess("", "")
	manager := &fakeProcessManager{process: process}
	adapter := NewAdapter(manager)

	spec, err := provider.NewClaudeCodeSessionSpec(
		provider.MustParseAgentCLICommand("claude"),
		nil,
		nil,
		nil,
		nil,
		"",
		nil,
		nil,
		nil,
		false,
	)
	if err != nil {
		t.Fatalf("NewClaudeCodeSessionSpec returned error: %v", err)
	}

	session, err := adapter.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	firstTurn, err := provider.NewClaudeCodeTurnInput("Investigate failing tests")
	if err != nil {
		t.Fatalf("NewClaudeCodeTurnInput returned error: %v", err)
	}
	secondTurn, err := provider.NewClaudeCodeTurnInput("Apply the narrow fix")
	if err != nil {
		t.Fatalf("NewClaudeCodeTurnInput returned error: %v", err)
	}

	if err := session.Send(context.Background(), firstTurn); err != nil {
		t.Fatalf("first Send returned error: %v", err)
	}
	if err := session.Send(context.Background(), secondTurn); err != nil {
		t.Fatalf("second Send returned error: %v", err)
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := session.Close(closeCtx); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	expected := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Investigate failing tests"}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"Apply the narrow fix"}]}}`,
		"",
	}, "\n")
	if process.stdin.String() != expected {
		t.Fatalf("unexpected stdin payload:\nwant: %s\ngot:  %s", expected, process.stdin.String())
	}
}

func TestSessionParsesNDJSONAndTracksSessionID(t *testing.T) {
	stdout := strings.Join([]string{
		`not-json`,
		`{"type":"system","subtype":"init","data":{"session_id":"sess-1"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]},"model":"claude-sonnet-4-6"}`,
		`{"type":"result","subtype":"success","session_id":"sess-1","result":"done","num_turns":1}`,
		"",
	}, "\n")
	manager := &fakeProcessManager{
		process: newCompletedFakeProcess(stdout, ""),
	}
	adapter := NewAdapter(manager)

	spec, err := provider.NewClaudeCodeSessionSpec(
		provider.MustParseAgentCLICommand("claude"),
		nil,
		nil,
		nil,
		nil,
		"",
		nil,
		nil,
		nil,
		false,
	)
	if err != nil {
		t.Fatalf("NewClaudeCodeSessionSpec returned error: %v", err)
	}

	session, err := adapter.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	events := collectEvents(t, session.Events())
	errors := collectErrors(t, session.Errors())

	if len(events) != 3 {
		t.Fatalf("expected 3 parsed events, got %d", len(events))
	}
	if events[0].Kind != provider.ClaudeCodeEventKindSystem || events[0].SessionID != "sess-1" {
		t.Fatalf("unexpected system event: %+v", events[0])
	}
	if events[1].Kind != provider.ClaudeCodeEventKindAssistant || events[1].Model != "claude-sonnet-4-6" {
		t.Fatalf("unexpected assistant event: %+v", events[1])
	}
	if events[2].Kind != provider.ClaudeCodeEventKindResult || events[2].Result != "done" || events[2].NumTurns != 1 {
		t.Fatalf("unexpected result event: %+v", events[2])
	}
	if len(errors) != 1 || !strings.Contains(errors[0].Error(), "parse claude code ndjson event") {
		t.Fatalf("expected one parse error, got %#v", errors)
	}

	sessionID, ok := session.SessionID()
	if !ok || sessionID != provider.MustParseClaudeCodeSessionID("sess-1") {
		t.Fatalf("unexpected session id: %q ok=%v", sessionID, ok)
	}
}

func collectEvents(t *testing.T, events <-chan provider.ClaudeCodeEvent) []provider.ClaudeCodeEvent {
	t.Helper()

	deadline := time.After(2 * time.Second)
	collected := make([]provider.ClaudeCodeEvent, 0)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return collected
			}
			collected = append(collected, event)
		case <-deadline:
			t.Fatal("timed out waiting for event stream to close")
		}
	}
}

func collectErrors(t *testing.T, errors <-chan error) []error {
	t.Helper()

	deadline := time.After(2 * time.Second)
	collected := make([]error, 0)
	for {
		select {
		case err, ok := <-errors:
			if !ok {
				return collected
			}
			collected = append(collected, err)
		case <-deadline:
			t.Fatal("timed out waiting for error stream to close")
		}
	}
}

type fakeProcessManager struct {
	process  provider.AgentCLIProcess
	lastSpec provider.AgentCLIProcessSpec
}

func (m *fakeProcessManager) Start(_ context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	m.lastSpec = spec
	return m.process, nil
}

type fakeProcess struct {
	stdin  *bufferWriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	waitOnce sync.Once
	waitCh   chan error
}

func newFakeProcess(stdout string, stderr string) *fakeProcess {
	return &fakeProcess{
		stdin:  &bufferWriteCloser{},
		stdout: io.NopCloser(strings.NewReader(stdout)),
		stderr: io.NopCloser(strings.NewReader(stderr)),
		waitCh: make(chan error),
	}
}

func newCompletedFakeProcess(stdout string, stderr string) *fakeProcess {
	process := newFakeProcess(stdout, stderr)
	close(process.waitCh)
	return process
}

func (p *fakeProcess) PID() int {
	return 42
}

func (p *fakeProcess) Stdin() io.WriteCloser {
	return p.stdin
}

func (p *fakeProcess) Stdout() io.ReadCloser {
	return p.stdout
}

func (p *fakeProcess) Stderr() io.ReadCloser {
	return p.stderr
}

func (p *fakeProcess) Wait() error {
	err, ok := <-p.waitCh
	if !ok {
		return nil
	}

	return err
}

func (p *fakeProcess) Stop(context.Context) error {
	p.waitOnce.Do(func() {
		close(p.waitCh)
	})
	return nil
}

type bufferWriteCloser struct {
	mu     sync.Mutex
	buffer strings.Builder
	closed bool
}

func (b *bufferWriteCloser) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, io.ErrClosedPipe
	}

	return b.buffer.Write(p)
}

func (b *bufferWriteCloser) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	return nil
}

func (b *bufferWriteCloser) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buffer.String()
}
