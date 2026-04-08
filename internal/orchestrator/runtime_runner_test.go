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

	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

type runtimeRunnerFailingSession struct{}

func (runtimeRunnerFailingSession) SessionID() (string, bool) { return "session-executing", true }
func (runtimeRunnerFailingSession) Events() <-chan agentEvent { return nil }
func (runtimeRunnerFailingSession) SendPrompt(context.Context, string) (agentTurnStartResult, error) {
	return agentTurnStartResult{}, errors.New("session unavailable")
}
func (runtimeRunnerFailingSession) Stop(context.Context) error { return nil }
func (runtimeRunnerFailingSession) Err() error                 { return nil }
func (runtimeRunnerFailingSession) Diagnostic() agentSessionDiagnostic {
	return agentSessionDiagnostic{}
}

type runtimeRunnerClosedSession struct {
	err        error
	diagnostic agentSessionDiagnostic
}

func (s runtimeRunnerClosedSession) SessionID() (string, bool) {
	if strings.TrimSpace(s.diagnostic.SessionID) == "" {
		return "", false
	}
	return s.diagnostic.SessionID, true
}

func (runtimeRunnerClosedSession) Events() <-chan agentEvent {
	stream := make(chan agentEvent)
	close(stream)
	return stream
}

func (runtimeRunnerClosedSession) SendPrompt(context.Context, string) (agentTurnStartResult, error) {
	return agentTurnStartResult{}, nil
}

func (runtimeRunnerClosedSession) Stop(context.Context) error { return nil }

func (s runtimeRunnerClosedSession) Err() error { return s.err }

func (s runtimeRunnerClosedSession) Diagnostic() agentSessionDiagnostic {
	return s.diagnostic
}

func TestRuntimeLauncherConsumeTurnIncludesSessionExitCause(t *testing.T) {
	logHandler := &runtimeRunnerLogHandler{}
	launcher := &RuntimeLauncher{logger: slog.New(logHandler)}
	highWater := tokenUsageHighWater{}
	sessionErr := errors.New("codex app server exited: exit status 2: fatal: app-server crashed")
	err := launcher.consumeTurn(
		context.Background(),
		uuid.Nil,
		uuid.Nil,
		uuid.Nil,
		uuid.Nil,
		entagentprovider.AdapterTypeCodexAppServer,
		uuid.Nil,
		runtimeRunnerClosedSession{
			err: sessionErr,
			diagnostic: agentSessionDiagnostic{
				PID:       4242,
				SessionID: "thread-1",
				Error:     sessionErr.Error(),
				Stderr:    "fatal: app-server crashed",
			},
		},
		"turn-eof",
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
}

func TestRuntimeLauncherRecordProviderRateLimitEmitsCanonicalActivity(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	fixture.createAgent(ctx, t, "codex-rate-limit-01", 0)

	bus := eventinfra.NewChannelBus()
	defer func() {
		if err := bus.Close(); err != nil {
			t.Fatalf("bus.Close() error = %v", err)
		}
	}()
	stream, err := bus.Subscribe(ctx, activityLifecycleTopic)
	if err != nil {
		t.Fatalf("Subscribe(activity lifecycle) error = %v", err)
	}

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, nil, nil, nil)
	observedAt := time.Date(2026, time.April, 1, 18, 55, 0, 0, time.UTC)
	usedPercent := 42.5
	resetAt := observedAt.Add(15 * time.Minute)
	rateLimit := &provider.CLIRateLimit{
		Provider: provider.CLIRateLimitProviderCodex,
		Codex: &provider.CodexRateLimit{
			LimitID:   "codex-limit",
			LimitName: "Codex Pro",
			PlanType:  "pro",
			Primary: &provider.CodexRateLimitWindow{
				UsedPercent:   &usedPercent,
				WindowMinutes: 60,
				ResetsAt:      &resetAt,
			},
		},
		Raw: map[string]any{"limit_id": "codex-limit"},
	}

	if err := launcher.recordProviderRateLimit(ctx, fixture.providerID, rateLimit, observedAt); err != nil {
		t.Fatalf("recordProviderRateLimit() error = %v", err)
	}

	updatedProvider, err := client.AgentProvider.Get(ctx, fixture.providerID)
	if err != nil {
		t.Fatalf("reload provider: %v", err)
	}
	if updatedProvider.CliRateLimitUpdatedAt == nil || !updatedProvider.CliRateLimitUpdatedAt.UTC().Equal(observedAt) {
		t.Fatalf("expected cli_rate_limit_updated_at %s, got %+v", observedAt.Format(time.RFC3339), updatedProvider.CliRateLimitUpdatedAt)
	}
	if updatedProvider.CliRateLimit["provider"] != string(provider.CLIRateLimitProviderCodex) {
		t.Fatalf("persisted provider rate limit = %+v", updatedProvider.CliRateLimit)
	}

	activities, err := client.ActivityEvent.Query().
		Where(
			entactivityevent.ProjectIDEQ(fixture.projectID),
			entactivityevent.EventTypeEQ(activityevent.TypeProviderRateLimitUpdated.String()),
		).
		All(ctx)
	if err != nil {
		t.Fatalf("query provider rate limit activities: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected one provider rate limit activity, got %+v", activities)
	}
	if activities[0].Metadata["provider_id"] != fixture.providerID.String() {
		t.Fatalf("provider rate limit activity metadata = %+v", activities[0].Metadata)
	}

	select {
	case event := <-stream:
		if event.Topic != activityLifecycleTopic || event.Type != provider.MustParseEventType(activityevent.TypeProviderRateLimitUpdated.String()) {
			t.Fatalf("activity event = %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for provider rate limit activity event")
	}

	nextObservedAt := observedAt.Add(5 * time.Minute)
	if err := launcher.recordProviderRateLimit(ctx, fixture.providerID, rateLimit, nextObservedAt); err != nil {
		t.Fatalf("recordProviderRateLimit(second) error = %v", err)
	}

	activities, err = client.ActivityEvent.Query().
		Where(
			entactivityevent.ProjectIDEQ(fixture.projectID),
			entactivityevent.EventTypeEQ(activityevent.TypeProviderRateLimitUpdated.String()),
		).
		All(ctx)
	if err != nil {
		t.Fatalf("query provider rate limit activities after duplicate payload: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected duplicate payload to avoid extra activity rows, got %+v", activities)
	}

	updatedProvider, err = client.AgentProvider.Get(ctx, fixture.providerID)
	if err != nil {
		t.Fatalf("reload provider after duplicate payload: %v", err)
	}
	if updatedProvider.CliRateLimitUpdatedAt == nil || !updatedProvider.CliRateLimitUpdatedAt.UTC().Equal(nextObservedAt) {
		t.Fatalf("expected latest cli_rate_limit_updated_at %s, got %+v", nextObservedAt.Format(time.RFC3339), updatedProvider.CliRateLimitUpdatedAt)
	}

	select {
	case event := <-stream:
		t.Fatalf("unexpected duplicate activity stream event: %+v", event)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestRuntimeLauncherStartReadyExecutionsPublishesExecutingLifecycle(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	executingAt := time.Date(2026, time.April, 3, 15, 4, 5, 0, time.UTC)

	bus := eventinfra.NewChannelBus()
	defer func() {
		if err := bus.Close(); err != nil {
			t.Fatalf("bus.Close() error = %v", err)
		}
	}()
	stream, err := bus.Subscribe(ctx, agentLifecycleTopic)
	if err != nil {
		t.Fatalf("Subscribe(agent lifecycle) error = %v", err)
	}

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-902").
		SetTitle("Publish executing lifecycle").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority("high").
		SetType("feature").
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "executing-01", 0)
	runItem := mustCreateCurrentRun(
		ctx,
		t,
		client,
		agentItem,
		workflowItem.ID,
		ticketItem.ID,
		entagentrun.StatusReady,
		executingAt.Add(-15*time.Second),
	)

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, nil, nil, nil)
	launcher.now = func() time.Time { return executingAt }
	launcher.storeSession(runItem.ID, runtimeRunnerFailingSession{})

	if err := launcher.startReadyExecutions(ctx); err != nil {
		t.Fatalf("startReadyExecutions() error = %v", err)
	}

	executingEvent := waitForAgentLifecycleEvent(t, stream, agentExecutingType)
	payload := decodeLifecycleEnvelope(t, executingEvent.Payload)
	if payload.Agent.ID != agentItem.ID.String() {
		t.Fatalf("executing payload agent id = %q, want %q", payload.Agent.ID, agentItem.ID.String())
	}
	if payload.Agent.RuntimePhase != "executing" {
		t.Fatalf("executing payload runtime phase = %q, want executing", payload.Agent.RuntimePhase)
	}
	if payload.Agent.CurrentTicketID == nil || *payload.Agent.CurrentTicketID != ticketItem.ID.String() {
		t.Fatalf("executing payload current ticket = %+v, want %s", payload.Agent.CurrentTicketID, ticketItem.ID)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusExecuting {
		t.Fatalf("run status = %s, want executing", runAfter.Status)
	}

	activities, err := client.ActivityEvent.Query().
		Where(
			entactivityevent.AgentIDEQ(agentItem.ID),
			entactivityevent.EventTypeEQ(activityevent.TypeAgentExecuting.String()),
		).
		All(ctx)
	if err != nil {
		t.Fatalf("query executing activities: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected one executing activity, got %+v", activities)
	}
	if activities[0].TicketID == nil || *activities[0].TicketID != ticketItem.ID {
		t.Fatalf("executing activity ticket_id = %+v, want %s", activities[0].TicketID, ticketItem.ID)
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

func TestAgentOutputAccumulatorAggregatesAssistantFragmentsUntilSentenceBoundary(t *testing.T) {
	accumulator := agentOutputAccumulator{}

	first := accumulator.push(&agentOutputEvent{
		Stream: "assistant",
		ItemID: "assistant-1",
		TurnID: "turn-1",
		Text:   "Current ",
	})
	if len(first) != 0 {
		t.Fatalf("first push should stay buffered, got %+v", first)
	}

	second := accumulator.push(&agentOutputEvent{
		Stream: "assistant",
		ItemID: "assistant-1",
		TurnID: "turn-1",
		Text:   "ticket created.",
	})
	if len(second) != 1 {
		t.Fatalf("expected sentence boundary flush, got %+v", second)
	}
	if second[0].Text != "Current ticket created." {
		t.Fatalf("unexpected merged assistant text: %+v", second[0])
	}
	if second[0].Snapshot {
		t.Fatalf("expected merged delta output, got snapshot: %+v", second[0])
	}
}

func TestAgentOutputAccumulatorPrefersSnapshotBeforeBufferedDeltaFlush(t *testing.T) {
	accumulator := agentOutputAccumulator{}

	initial := accumulator.push(&agentOutputEvent{
		Stream:  "command",
		ItemID:  "command-1",
		TurnID:  "turn-1",
		Command: "pnpm vitest run",
		Text:    "PASS src/app.test.ts",
	})
	if len(initial) != 0 {
		t.Fatalf("initial delta should stay buffered, got %+v", initial)
	}

	flushed := accumulator.push(&agentOutputEvent{
		Stream:   "command",
		ItemID:   "command-1",
		TurnID:   "turn-1",
		Command:  "pnpm vitest run",
		Text:     "PASS src/app.test.ts\nDone in 1.2s\n",
		Snapshot: true,
	})
	if len(flushed) != 1 {
		t.Fatalf("expected snapshot flush, got %+v", flushed)
	}
	if !flushed[0].Snapshot {
		t.Fatalf("expected snapshot output, got %+v", flushed[0])
	}
	if flushed[0].Text != "PASS src/app.test.ts\nDone in 1.2s\n" {
		t.Fatalf("unexpected snapshot text: %+v", flushed[0])
	}
}

func TestOutputForPersistencePromotesAggregatedOutputsToSnapshots(t *testing.T) {
	persisted := outputForPersistence(&agentOutputEvent{
		Stream:  "assistant",
		ItemID:  "assistant-1",
		TurnID:  "turn-1",
		Text:    "Complete sentence.",
		Command: "",
	})
	if persisted == nil || !persisted.Snapshot {
		t.Fatalf("expected persisted assistant output snapshot, got %+v", persisted)
	}

	commandPersisted := outputForPersistence(&agentOutputEvent{
		Stream:   "command",
		ItemID:   "command-1",
		TurnID:   "turn-1",
		Command:  "pnpm vitest run",
		Text:     "PASS src/app.test.ts\n",
		Snapshot: false,
	})
	if commandPersisted == nil || !commandPersisted.Snapshot {
		t.Fatalf("expected persisted command output snapshot, got %+v", commandPersisted)
	}

	taskPersisted := outputForPersistence(&agentOutputEvent{
		Stream: "task",
		Text:   "{\"status\":\"planning\"}",
	})
	if taskPersisted != nil {
		t.Fatalf("expected non-semantic task output to be dropped, got %+v", taskPersisted)
	}
}

func TestAgentOutputPersistenceFingerprintSkipsItemlessOutputs(t *testing.T) {
	if key, value := agentOutputPersistenceFingerprint(&agentOutputEvent{Stream: "assistant", Text: "hello"}); key != "" || value != "" {
		t.Fatalf("expected empty fingerprint for itemless output, got %q %q", key, value)
	}

	key, value := agentOutputPersistenceFingerprint(&agentOutputEvent{
		Stream:   "command",
		ItemID:   "command-1",
		TurnID:   "turn-1",
		Command:  "pnpm vitest run",
		Phase:    "running_command",
		Snapshot: true,
		Text:     "PASS\n",
	})
	if key == "" || value == "" {
		t.Fatalf("expected non-empty fingerprint, got %q %q", key, value)
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
