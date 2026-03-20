package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func TestRuntimeLauncherRunTickTransitionsClaimedAgentToReady(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 0, 0, 0, time.UTC)

	bus := eventinfra.NewChannelBus()
	stream, err := bus.Subscribe(ctx, agentLifecycleTopic)
	if err != nil {
		t.Fatalf("subscribe agent lifecycle stream: %v", err)
	}

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("create git marker: %v", err)
	}
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	if err := os.WriteFile(harnessPath, []byte(`---
workflow:
  role: coding
---

Current {{ machine.name }} root={{ machine.workspace_root }}
Access {% for machine in accessible_machines %}{{ machine.name }}={{ machine.ssh_user }}@{{ machine.host }}|{% endfor %}
`), 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	workflowSvc, err := workflowservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-401").
		SetTitle("Launch Codex").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-01").
		SetStatus(entagent.StatusClaimed).
		SetCurrentTicketID(ticketItem.ID).
		SetRuntimePhase(entagent.RuntimePhaseNone).
		SetWorkspacePath("/tmp/openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	localWorkspaceRoot := "/srv/openase/workspaces"
	if _, err := client.Machine.Create().
		SetOrganizationID(fixture.orgID).
		SetName("local").
		SetHost("local").
		SetStatus(entmachine.StatusOnline).
		SetWorkspaceRoot(localWorkspaceRoot).
		Save(ctx); err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	sshUser := "openase"
	storageMachine, err := client.Machine.Create().
		SetOrganizationID(fixture.orgID).
		SetName("storage").
		SetHost("10.0.1.20").
		SetSSHUser(sshUser).
		SetSSHKeyPath("keys/storage.pem").
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create storage machine: %v", err)
	}
	if _, err := client.Machine.Create().
		SetOrganizationID(fixture.orgID).
		SetName("dev-01").
		SetHost("10.0.1.30").
		SetSSHUser(sshUser).
		SetSSHKeyPath("keys/dev-01.pem").
		SetStatus(entmachine.StatusOnline).
		Save(ctx); err != nil {
		t.Fatalf("create non-whitelisted machine: %v", err)
	}
	if _, err := client.Project.UpdateOneID(fixture.projectID).
		SetAccessibleMachineIds([]uuid.UUID{storageMachine.ID}).
		Save(ctx); err != nil {
		t.Fatalf("update project accessible machines: %v", err)
	}

	manager := &runtimeFakeProcessManager{}
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, manager, workflowSvc)
	launcher.now = func() time.Time {
		return now
	}
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run launcher tick: %v", err)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.Status != entagent.StatusRunning {
		t.Fatalf("expected running status, got %s", agentAfter.Status)
	}
	if agentAfter.RuntimePhase != entagent.RuntimePhaseReady {
		t.Fatalf("expected runtime phase ready, got %s", agentAfter.RuntimePhase)
	}
	if agentAfter.SessionID != "thread-runtime-1" {
		t.Fatalf("expected thread-runtime-1 session id, got %q", agentAfter.SessionID)
	}
	if agentAfter.RuntimeStartedAt == nil || !agentAfter.RuntimeStartedAt.UTC().Equal(now) {
		t.Fatalf("expected runtime_started_at %s, got %+v", now.Format(time.RFC3339), agentAfter.RuntimeStartedAt)
	}
	if agentAfter.LastHeartbeatAt == nil || !agentAfter.LastHeartbeatAt.UTC().Equal(now) {
		t.Fatalf("expected last_heartbeat_at %s, got %+v", now.Format(time.RFC3339), agentAfter.LastHeartbeatAt)
	}
	if agentAfter.LastError != "" {
		t.Fatalf("expected empty last_error, got %q", agentAfter.LastError)
	}

	readyEvent := waitForAgentLifecycleEvent(t, stream, agentReadyType)
	payload := decodeLifecycleEnvelope(t, readyEvent.Payload)
	if payload.Agent.ID != agentItem.ID.String() || payload.Agent.RuntimePhase != "ready" {
		t.Fatalf("unexpected ready event payload: %+v", payload.Agent)
	}

	activityItems, err := client.ActivityEvent.Query().
		Where(entactivityevent.AgentIDEQ(agentItem.ID)).
		All(ctx)
	if err != nil {
		t.Fatalf("list activity events: %v", err)
	}
	if len(activityItems) == 0 {
		t.Fatal("expected runtime lifecycle activity events to be persisted")
	}
	if !strings.Contains(manager.capturedThreadStart().DeveloperInstructions, "Current local root=/srv/openase/workspaces") {
		t.Fatalf("expected rendered current machine in developer instructions, got %q", manager.capturedThreadStart().DeveloperInstructions)
	}
	if !strings.Contains(manager.capturedThreadStart().DeveloperInstructions, "storage=openase@10.0.1.20|") {
		t.Fatalf("expected whitelisted machine in developer instructions, got %q", manager.capturedThreadStart().DeveloperInstructions)
	}
	if strings.Contains(manager.capturedThreadStart().DeveloperInstructions, "dev-01=") {
		t.Fatalf("expected non-whitelisted machine to stay out of developer instructions, got %q", manager.capturedThreadStart().DeveloperInstructions)
	}
}

func waitForAgentLifecycleEvent(t *testing.T, stream <-chan provider.Event, want provider.EventType) provider.Event {
	t.Helper()

	timeout := time.After(2 * time.Second)
	for {
		select {
		case event := <-stream:
			if event.Type == want {
				return event
			}
		case <-timeout:
			t.Fatalf("timed out waiting for %s", want)
			return provider.Event{}
		}
	}
}

func decodeLifecycleEnvelope(t *testing.T, payload json.RawMessage) agentLifecycleEnvelope {
	t.Helper()

	var decoded agentLifecycleEnvelope
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode lifecycle payload: %v", err)
	}
	return decoded
}

type runtimeFakeProcessManager struct {
	mu                 sync.Mutex
	capturedThreadData runtimeThreadStartParams
}

func (m *runtimeFakeProcessManager) Start(_ context.Context, _ provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	process := newRuntimeFakeProcess()
	go func() {
		_ = runRuntimeFakeHandshake(process, m)
	}()
	return process, nil
}

func (m *runtimeFakeProcessManager) setThreadStart(params runtimeThreadStartParams) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.capturedThreadData = params
}

func (m *runtimeFakeProcessManager) capturedThreadStart() runtimeThreadStartParams {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.capturedThreadData
}

type runtimeFakeProcess struct {
	stdinRead  *io.PipeReader
	stdinWrite *io.PipeWriter

	stdoutRead  *io.PipeReader
	stdoutWrite *io.PipeWriter

	stderrRead  *io.PipeReader
	stderrWrite *io.PipeWriter

	done   chan error
	stopCh chan struct{}
}

func newRuntimeFakeProcess() *runtimeFakeProcess {
	stdinRead, stdinWrite := io.Pipe()
	stdoutRead, stdoutWrite := io.Pipe()
	stderrRead, stderrWrite := io.Pipe()

	return &runtimeFakeProcess{
		stdinRead:   stdinRead,
		stdinWrite:  stdinWrite,
		stdoutRead:  stdoutRead,
		stdoutWrite: stdoutWrite,
		stderrRead:  stderrRead,
		stderrWrite: stderrWrite,
		done:        make(chan error, 1),
		stopCh:      make(chan struct{}),
	}
}

func (p *runtimeFakeProcess) PID() int              { return 4242 }
func (p *runtimeFakeProcess) Stdin() io.WriteCloser { return p.stdinWrite }
func (p *runtimeFakeProcess) Stdout() io.ReadCloser { return p.stdoutRead }
func (p *runtimeFakeProcess) Stderr() io.ReadCloser { return p.stderrRead }
func (p *runtimeFakeProcess) Wait() error           { return <-p.done }

func (p *runtimeFakeProcess) Stop(context.Context) error {
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
	_ = p.stdinRead.Close()
	_ = p.stdinWrite.Close()
	_ = p.stdoutRead.Close()
	_ = p.stdoutWrite.Close()
	_ = p.stderrRead.Close()
	_ = p.stderrWrite.Close()

	select {
	case p.done <- nil:
	default:
	}
	return nil
}

func runRuntimeFakeHandshake(process *runtimeFakeProcess, manager *runtimeFakeProcessManager) error {
	decoder := json.NewDecoder(process.stdinRead)
	encoder := json.NewEncoder(process.stdoutWrite)

	initialize, err := readRuntimeMessage(decoder)
	if err != nil {
		return err
	}
	if initialize.Method != "initialize" {
		return fmt.Errorf("expected initialize, got %s", initialize.Method)
	}
	if err := encoder.Encode(runtimeJSONRPCMessage{
		JSONRPC: "2.0",
		ID:      initialize.ID,
		Result: mustMarshalRuntimeJSON(map[string]any{
			"userAgent":      "codex-cli/test",
			"platformFamily": "unix",
			"platformOs":     "linux",
		}),
	}); err != nil {
		return err
	}

	initialized, err := readRuntimeMessage(decoder)
	if err != nil {
		return err
	}
	if initialized.Method != "initialized" {
		return fmt.Errorf("expected initialized, got %s", initialized.Method)
	}

	threadStart, err := readRuntimeMessage(decoder)
	if err != nil {
		return err
	}
	if threadStart.Method != "thread/start" {
		return fmt.Errorf("expected thread/start, got %s", threadStart.Method)
	}
	var threadStartParams runtimeThreadStartParams
	if err := json.Unmarshal(threadStart.Params, &threadStartParams); err != nil {
		return fmt.Errorf("decode thread/start params: %w", err)
	}
	if manager != nil {
		manager.setThreadStart(threadStartParams)
	}
	if err := encoder.Encode(runtimeJSONRPCMessage{
		JSONRPC: "2.0",
		ID:      threadStart.ID,
		Result: mustMarshalRuntimeJSON(map[string]any{
			"thread": map[string]any{"id": "thread-runtime-1"},
		}),
	}); err != nil {
		return err
	}

	<-process.stopCh
	return nil
}

type runtimeJSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type runtimeThreadStartParams struct {
	CWD                   string `json:"cwd,omitempty"`
	DeveloperInstructions string `json:"developerInstructions,omitempty"`
}

func readRuntimeMessage(decoder *json.Decoder) (runtimeJSONRPCMessage, error) {
	var message runtimeJSONRPCMessage
	if err := decoder.Decode(&message); err != nil {
		return runtimeJSONRPCMessage{}, err
	}
	return message, nil
}

func mustMarshalRuntimeJSON(value any) json.RawMessage {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return payload
}
