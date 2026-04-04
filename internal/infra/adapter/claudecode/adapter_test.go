package claudecode

import (
	"bytes"
	"context"
	"errors"
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
		"--verbose",
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

func TestAdapterStartDoesNotDuplicateVerboseFlag(t *testing.T) {
	manager := &fakeProcessManager{
		process: newFakeProcess("", ""),
	}
	adapter := NewAdapter(manager)

	spec, err := provider.NewClaudeCodeSessionSpec(
		provider.MustParseAgentCLICommand("claude"),
		[]string{"--verbose", "--model", "claude-sonnet-4-6"},
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
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = session.Close(closeCtx)
	})

	verboseCount := 0
	for _, arg := range manager.lastSpec.Args {
		if arg == "--verbose" {
			verboseCount++
		}
	}
	if verboseCount != 1 {
		t.Fatalf("expected exactly one --verbose flag, got %d in %#v", verboseCount, manager.lastSpec.Args)
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
		`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":1775037600,"rateLimitType":"five_hour","isUsingOverage":false}}`,
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

	if len(events) != 4 {
		t.Fatalf("expected 4 parsed events, got %d", len(events))
	}
	if events[0].Kind != provider.ClaudeCodeEventKindSystem || events[0].SessionID != "sess-1" {
		t.Fatalf("unexpected system event: %+v", events[0])
	}
	if events[1].Kind != provider.ClaudeCodeEventKindAssistant || events[1].Model != "claude-sonnet-4-6" {
		t.Fatalf("unexpected assistant event: %+v", events[1])
	}
	if events[2].Kind != provider.ClaudeCodeEventKindRateLimit || events[2].RateLimitInfo == nil || events[2].RateLimitInfo.ClaudeCode == nil {
		t.Fatalf("unexpected rate limit event: %+v", events[2])
	}
	if events[3].Kind != provider.ClaudeCodeEventKindResult || events[3].Result != "done" || events[3].NumTurns != 1 {
		t.Fatalf("unexpected result event: %+v", events[3])
	}
	if len(errors) != 1 || !strings.Contains(errors[0].Error(), "parse claude code ndjson event") {
		t.Fatalf("expected one parse error, got %#v", errors)
	}

	sessionID, ok := session.SessionID()
	if !ok || sessionID != provider.MustParseClaudeCodeSessionID("sess-1") {
		t.Fatalf("unexpected session id: %q ok=%v", sessionID, ok)
	}
}

func TestAdapterAndSessionValidationBranches(t *testing.T) {
	adapter := NewAdapter(nil)
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

	var nilCtx context.Context

	if _, err := adapter.Start(nilCtx, spec); err == nil || err.Error() != "context must not be nil" {
		t.Fatalf("expected nil context error, got %v", err)
	}
	if _, err := adapter.Start(context.Background(), spec); err == nil || err.Error() != "claude code process manager must not be nil" {
		t.Fatalf("expected missing manager error, got %v", err)
	}

	startErrAdapter := NewAdapter(&fakeProcessManager{startErr: errors.New("start failed")})
	if _, err := startErrAdapter.Start(context.Background(), spec); err == nil || err.Error() != "start failed" {
		t.Fatalf("expected start error, got %v", err)
	}

	session := &session{
		process: newFakeProcess("", ""),
		events:  make(chan provider.ClaudeCodeEvent, 1),
		errors:  make(chan error, 4),
		done:    make(chan struct{}),
	}
	input := provider.ClaudeCodeTurnInput{Prompt: "hello"}

	if err := session.Send(nilCtx, input); err == nil || err.Error() != "context must not be nil" {
		t.Fatalf("expected nil send context error, got %v", err)
	}
	if err := session.Send(context.Background(), provider.ClaudeCodeTurnInput{}); err == nil || err.Error() != "claude code turn prompt must not be empty" {
		t.Fatalf("expected empty prompt error, got %v", err)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := session.Send(canceledCtx, input); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context error, got %v", err)
	}

	close(session.done)
	if err := session.Send(context.Background(), input); err == nil || err.Error() != "claude code session already closed" {
		t.Fatalf("expected closed session error, got %v", err)
	}
	if err := session.Close(nilCtx); err == nil || err.Error() != "context must not be nil" {
		t.Fatalf("expected nil close context error, got %v", err)
	}
}

func TestSessionReaderAndHelperCoverage(t *testing.T) {
	process := newFakeProcess("", "")
	sess := &session{
		process: process,
		events:  make(chan provider.ClaudeCodeEvent, 8),
		errors:  make(chan error, 8),
		done:    make(chan struct{}),
	}

	sess.readStdout(errorReadCloser{})
	if err := <-sess.errors; err == nil || !strings.Contains(err.Error(), "read claude code stdout") {
		t.Fatalf("expected stdout read error, got %v", err)
	}

	sess.readStderr(io.NopCloser(strings.NewReader("stderr line\n")))
	if err := <-sess.errors; err == nil || !strings.Contains(err.Error(), "claude code stderr: stderr line") {
		t.Fatalf("expected stderr line error, got %v", err)
	}

	sess.readStderr(errorReadCloser{})
	if err := <-sess.errors; err == nil || !strings.Contains(err.Error(), "read claude code stderr") {
		t.Fatalf("expected stderr read error, got %v", err)
	}

	if kind := mapEventKind("task_notification"); kind != provider.ClaudeCodeEventKindTaskNotice {
		t.Fatalf("mapEventKind(task_notification)=%q", kind)
	}
	if kind := mapEventKind("rate_limit_event"); kind != provider.ClaudeCodeEventKindRateLimit {
		t.Fatalf("mapEventKind(rate_limit_event)=%q", kind)
	}
	if kind := mapEventKind("unknown-type"); kind != provider.ClaudeCodeEventKindUnknown {
		t.Fatalf("mapEventKind(unknown-type)=%q", kind)
	}

	if _, err := parseStreamEvent([]byte(`{"type":"   "}`)); err == nil || !strings.Contains(err.Error(), "missing type") {
		t.Fatalf("expected missing type error, got %v", err)
	}

	unknownEvent, err := parseStreamEvent([]byte(`{"type":"custom","session_id":" invalid ","message":{"hello":"world"},"usage":{"tokens":1},"event":{"step":"a"},"uuid":" u-1 "}`))
	if err != nil {
		t.Fatalf("parseStreamEvent returned error: %v", err)
	}
	if unknownEvent.Kind != provider.ClaudeCodeEventKindUnknown || unknownEvent.UnknownType != "custom" {
		t.Fatalf("unexpected unknown event: %+v", unknownEvent)
	}
	if unknownEvent.SessionID != "invalid" || unknownEvent.UUID != "u-1" {
		t.Fatalf("expected trimmed fields, got %+v", unknownEvent)
	}

	value := 3.25
	cloned := cloneOptionalFloat(&value)
	if cloned == nil || *cloned != value || cloned == &value {
		t.Fatalf("cloneOptionalFloat() = %v", cloned)
	}
	if cloneOptionalFloat(nil) != nil {
		t.Fatal("expected nil float clone to stay nil")
	}
	if cloneRawJSON(nil) != nil {
		t.Fatal("expected nil raw json clone to stay nil")
	}
	clonedJSON := cloneRawJSON(bytes.Repeat([]byte("a"), 2))
	if string(clonedJSON) != "aa" {
		t.Fatalf("unexpected raw json clone: %q", string(clonedJSON))
	}

	payload, err := encodeTurnInput(provider.ClaudeCodeTurnInput{Prompt: " prompt "})
	if err != nil {
		t.Fatalf("encodeTurnInput returned error: %v", err)
	}
	if !strings.HasSuffix(string(payload), "\n") || !strings.Contains(string(payload), `"text":" prompt "`) {
		t.Fatalf("unexpected encoded payload: %s", string(payload))
	}

	sess.setSessionID("   ")
	if _, ok := sess.SessionID(); ok {
		t.Fatal("expected invalid session id to be ignored")
	}
	sess.setSessionID("sess-valid")
	sess.setSessionID("sess-next")
	sessionID, ok := sess.SessionID()
	if !ok || sessionID != provider.MustParseClaudeCodeSessionID("sess-valid") {
		t.Fatalf("unexpected session id: %q ok=%v", sessionID, ok)
	}

	close(sess.done)
	sess.pushError(errors.New("ignored"))
	select {
	case err := <-sess.errors:
		t.Fatalf("expected pushError to skip closed session, got %v", err)
	default:
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
	startErr error
}

func (m *fakeProcessManager) Start(_ context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	m.lastSpec = spec
	if m.startErr != nil {
		return nil, m.startErr
	}
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

type errorReadCloser struct{}

func (errorReadCloser) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func (errorReadCloser) Close() error {
	return nil
}
