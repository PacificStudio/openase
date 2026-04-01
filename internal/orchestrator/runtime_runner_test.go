package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestRuntimeLauncherConsumeTurnIncludesSessionExitCause(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	adapter, err := codex.NewAdapter(codex.AdapterOptions{ProcessManager: &runtimeRunnerFakeProcessManager{process: process}})
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

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
				Result:  mustMarshalJSON(map[string]any{"turn": map[string]any{"id": "turn-eof", "status": "inProgress"}}),
			}); err != nil {
				return err
			}

			if _, err := io.WriteString(process.stderrWrite, "fatal: app-server crashed"); err != nil {
				return err
			}
			process.finish(errors.New("exit status 2"))
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

	session, err := adapter.Start(context.Background(), codex.StartRequest{
		Process: processSpec,
		Thread: codex.ThreadStartParams{
			WorkingDirectory: "/tmp/openase",
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	agentSession := newCodexAgentSession(session)

	turn, err := session.SendPrompt(context.Background(), "Keep working.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}

	logHandler := &runtimeRunnerLogHandler{}
	launcher := &RuntimeLauncher{logger: slog.New(logHandler)}
	highWater := tokenUsageHighWater{}
	err = launcher.consumeTurn(
		context.Background(),
		uuid.Nil,
		uuid.Nil,
		uuid.Nil,
		uuid.Nil,
		entagentprovider.AdapterTypeCodexAppServer,
		uuid.Nil,
		agentSession,
		turn.TurnID,
		&highWater,
	)
	if err == nil {
		t.Fatal("consumeTurn() returned nil")
	}
	message := err.Error()
	if !strings.Contains(message, "agent session closed before turn turn-eof completed") {
		t.Fatalf("consumeTurn() error = %q", message)
	}
	if !strings.Contains(message, "codex app server exited: exit status 2: fatal: app-server crashed") {
		t.Fatalf("consumeTurn() error missing exit cause: %q", message)
	}

	record := logHandler.lastRecord()
	if record.Message != "agent session closed before turn completed" {
		t.Fatalf("unexpected log message: %+v", record)
	}
	if record.Level != slog.LevelError {
		t.Fatalf("unexpected log level: %+v", record)
	}
	if got := record.Attrs["provider_pid"]; got != int64(4242) {
		t.Fatalf("unexpected provider_pid attr: %+v", record.Attrs)
	}
	if got := record.Attrs["provider_session_id"]; got != "thread-1" {
		t.Fatalf("unexpected provider_session_id attr: %+v", record.Attrs)
	}
	if got := record.Attrs["provider_stderr"]; got != "fatal: app-server crashed" {
		t.Fatalf("unexpected provider_stderr attr: %+v", record.Attrs)
	}
	if got := record.Attrs["provider_session_error"]; got != "codex app server exited: exit status 2: fatal: app-server crashed" {
		t.Fatalf("unexpected provider_session_error attr: %+v", record.Attrs)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

func TestRuntimeLauncherConsumeTurnReturnsCleanSessionCloseWithoutExitCause(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	adapter, err := codex.NewAdapter(codex.AdapterOptions{ProcessManager: &runtimeRunnerFakeProcessManager{process: process}})
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

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
				Result:  mustMarshalJSON(map[string]any{"turn": map[string]any{"id": "turn-clean-close", "status": "inProgress"}}),
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

	session, err := adapter.Start(context.Background(), codex.StartRequest{
		Process: processSpec,
		Thread: codex.ThreadStartParams{
			WorkingDirectory: "/tmp/openase",
		},
	})
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	agentSession := newCodexAgentSession(session)

	turn, err := session.SendPrompt(context.Background(), "Keep working.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}

	launcher := &RuntimeLauncher{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	highWater := tokenUsageHighWater{}
	err = launcher.consumeTurn(
		context.Background(),
		uuid.Nil,
		uuid.Nil,
		uuid.Nil,
		uuid.Nil,
		entagentprovider.AdapterTypeCodexAppServer,
		uuid.Nil,
		agentSession,
		turn.TurnID,
		&highWater,
	)
	if err == nil {
		t.Fatal("consumeTurn() returned nil")
	}
	if !isCleanTurnSessionClose(err) {
		t.Fatalf("consumeTurn() error = %v, want clean session close", err)
	}

	var closedErr *turnSessionClosedError
	if !errors.As(err, &closedErr) || closedErr == nil || closedErr.cause != nil {
		t.Fatalf("consumeTurn() error = %#v, want clean turnSessionClosedError", err)
	}
	if message := err.Error(); !strings.Contains(message, "agent session closed before turn turn-clean-close completed") {
		t.Fatalf("consumeTurn() error = %q", message)
	}

	if err := <-serverDone; err != nil {
		t.Fatalf("fake server returned error: %v", err)
	}
}

type runtimeRunnerFakeProcessManager struct {
	process *runtimeRunnerFakeProcess
}

func (m *runtimeRunnerFakeProcessManager) Start(context.Context, provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	return m.process, nil
}

type runtimeRunnerFakeProcess struct {
	stdinRead  *io.PipeReader
	stdinWrite *io.PipeWriter

	stdoutRead  *io.PipeReader
	stdoutWrite *io.PipeWriter

	stderrRead  *io.PipeReader
	stderrWrite *io.PipeWriter

	done chan struct{}

	stopOnce sync.Once

	waitMu  sync.Mutex
	waitErr error
}

func newRuntimeRunnerFakeProcess() *runtimeRunnerFakeProcess {
	stdinRead, stdinWrite := io.Pipe()
	stdoutRead, stdoutWrite := io.Pipe()
	stderrRead, stderrWrite := io.Pipe()

	return &runtimeRunnerFakeProcess{
		stdinRead:   stdinRead,
		stdinWrite:  stdinWrite,
		stdoutRead:  stdoutRead,
		stdoutWrite: stdoutWrite,
		stderrRead:  stderrRead,
		stderrWrite: stderrWrite,
		done:        make(chan struct{}),
	}
}

func (p *runtimeRunnerFakeProcess) PID() int              { return 4242 }
func (p *runtimeRunnerFakeProcess) Stdin() io.WriteCloser { return p.stdinWrite }
func (p *runtimeRunnerFakeProcess) Stdout() io.ReadCloser { return p.stdoutRead }
func (p *runtimeRunnerFakeProcess) Stderr() io.ReadCloser { return p.stderrRead }
func (p *runtimeRunnerFakeProcess) Wait() error {
	<-p.done
	p.waitMu.Lock()
	defer p.waitMu.Unlock()
	return p.waitErr
}
func (p *runtimeRunnerFakeProcess) Stop(context.Context) error {
	p.finish(nil)
	return nil
}

func (p *runtimeRunnerFakeProcess) finish(err error) {
	p.stopOnce.Do(func() {
		_ = p.stdinWrite.Close()
		_ = p.stdoutWrite.Close()
		_ = p.stderrWrite.Close()
		p.waitMu.Lock()
		p.waitErr = err
		p.waitMu.Unlock()
		close(p.done)
	})
}

type runtimeRunnerJSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func runRuntimeRunnerProtocolServer(
	process *runtimeRunnerFakeProcess,
	handler func(decoder *json.Decoder, encoder *json.Encoder) error,
) error {
	defer process.finish(nil)

	decoder := json.NewDecoder(process.stdinRead)
	encoder := json.NewEncoder(process.stdoutWrite)

	return handler(decoder, encoder)
}

func runtimeRunnerCompleteHandshake(decoder *json.Decoder, encoder *json.Encoder) error {
	initialize, err := runtimeRunnerReadMessage(decoder)
	if err != nil {
		return err
	}
	if initialize.Method != "initialize" {
		return errors.New("expected initialize request")
	}
	if err := encoder.Encode(runtimeRunnerJSONRPCMessage{
		JSONRPC: "2.0",
		ID:      initialize.ID,
		Result:  mustMarshalJSON(map[string]any{"userAgent": "codex-cli/1.0.0", "platformFamily": "unix", "platformOs": "linux"}),
	}); err != nil {
		return err
	}

	initialized, err := runtimeRunnerReadMessage(decoder)
	if err != nil {
		return err
	}
	if initialized.Method != "initialized" {
		return errors.New("expected initialized notification")
	}

	threadStart, err := runtimeRunnerReadMessage(decoder)
	if err != nil {
		return err
	}
	if threadStart.Method != "thread/start" {
		return errors.New("expected thread/start request")
	}

	return encoder.Encode(runtimeRunnerJSONRPCMessage{
		JSONRPC: "2.0",
		ID:      threadStart.ID,
		Result:  mustMarshalJSON(map[string]any{"thread": map[string]any{"id": "thread-1"}}),
	})
}

func runtimeRunnerReadMessage(decoder *json.Decoder) (runtimeRunnerJSONRPCMessage, error) {
	var message runtimeRunnerJSONRPCMessage
	if err := decoder.Decode(&message); err != nil {
		return runtimeRunnerJSONRPCMessage{}, err
	}

	return message, nil
}

func mustMarshalJSON(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func TestRuntimeRunnerFakeProcessDoesNotHang(t *testing.T) {
	process := newRuntimeRunnerFakeProcess()
	process.finish(nil)

	done := make(chan struct{})
	go func() {
		_ = process.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fake process wait")
	}
}

type runtimeRunnerCapturedRecord struct {
	Level   slog.Level
	Message string
	Attrs   map[string]any
}

type runtimeRunnerLogHandler struct {
	mu      sync.Mutex
	records []runtimeRunnerCapturedRecord
}

func (h *runtimeRunnerLogHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *runtimeRunnerLogHandler) Handle(_ context.Context, record slog.Record) error {
	captured := runtimeRunnerCapturedRecord{
		Level:   record.Level,
		Message: record.Message,
		Attrs:   map[string]any{},
	}
	record.Attrs(func(attr slog.Attr) bool {
		captured.Attrs[attr.Key] = attr.Value.Any()
		return true
	})

	h.mu.Lock()
	h.records = append(h.records, captured)
	h.mu.Unlock()
	return nil
}

func (h *runtimeRunnerLogHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *runtimeRunnerLogHandler) WithGroup(string) slog.Handler      { return h }

func (h *runtimeRunnerLogHandler) lastRecord() runtimeRunnerCapturedRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.records) == 0 {
		return runtimeRunnerCapturedRecord{}
	}
	return h.records[len(h.records)-1]
}
