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
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
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
	localMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(fixture.orgID),
			entmachine.NameEQ(catalogdomain.LocalMachineName),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load local machine: %v", err)
	}
	if _, err := client.Machine.UpdateOneID(localMachine.ID).
		SetWorkspaceRoot(localWorkspaceRoot).
		Save(ctx); err != nil {
		t.Fatalf("update local machine workspace root: %v", err)
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
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, manager, nil, workflowSvc)
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

func TestRuntimeLauncherRunTickStopsSessionWhenAgentLeavesRunningState(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

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

Runtime reconcile test
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
		SetIdentifier("ASE-402").
		SetTitle("Pause Codex").
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
		SetName("codex-02").
		SetStatus(entagent.StatusClaimed).
		SetCurrentTicketID(ticketItem.ID).
		SetRuntimePhase(entagent.RuntimePhaseNone).
		SetWorkspacePath("/tmp/openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	localMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(fixture.orgID),
			entmachine.NameEQ(catalogdomain.LocalMachineName),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load local machine: %v", err)
	}
	if _, err := client.Machine.UpdateOneID(localMachine.ID).
		SetWorkspaceRoot("/srv/openase/workspaces").
		Save(ctx); err != nil {
		t.Fatalf("update local machine workspace root: %v", err)
	}

	manager := &runtimeFakeProcessManager{}
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, manager, nil, workflowSvc)
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("launch initial runtime: %v", err)
	}
	agentAfterLaunch, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload launched agent: %v", err)
	}
	if agentAfterLaunch.Status != entagent.StatusRunning || agentAfterLaunch.RuntimePhase != entagent.RuntimePhaseReady {
		t.Fatalf("expected running ready agent after launch, got %+v", agentAfterLaunch)
	}

	if _, err := client.Agent.UpdateOneID(agentItem.ID).
		SetStatus(entagent.StatusPaused).
		SetRuntimePhase(entagent.RuntimePhaseNone).
		ClearSessionID().
		ClearRuntimeStartedAt().
		ClearLastHeartbeatAt().
		Save(ctx); err != nil {
		t.Fatalf("pause agent runtime in db: %v", err)
	}

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("reconcile paused runtime: %v", err)
	}

	if len(launcher.sessionsSnapshot()) != 0 {
		t.Fatalf("expected paused agent session to be removed, got %d sessions", len(launcher.sessionsSnapshot()))
	}
	process := manager.capturedProcess()
	if process == nil {
		t.Fatal("expected captured runtime process")
	}
	select {
	case <-process.stopCh:
	default:
		t.Fatal("expected paused agent session to be stopped")
	}
}

func TestRuntimeLauncherRunTickPreparesRemoteWorkspaceAndLaunchesOverSSH(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

	if _, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-401").
		SetTitle("Launch Codex on remote machine").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	repoItem, err := client.ProjectRepo.Create().
		SetProjectID(fixture.projectID).
		SetName("backend").
		SetRepositoryURL("git@github.com:acme/backend.git").
		SetDefaultBranch("main").
		SetClonePath("backend").
		SetIsPrimary(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}
	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(repoItem.ID).
		SetBranchName("agent/codex-01/ASE-401").
		SetPrStatus("none").
		SetCiStatus("pending").
		SetIsPrimaryScope(true).
		Save(ctx); err != nil {
		t.Fatalf("create ticket repo scope: %v", err)
	}

	sshUser := "openase"
	sshKeyPath := "keys/gpu-01.pem"
	workspaceRoot := "/srv/openase/workspaces"
	agentCLIPath := "/usr/local/bin/codex"
	if _, err := client.Machine.Create().
		SetOrganizationID(fixture.orgID).
		SetName("gpu-01").
		SetHost("10.0.1.10").
		SetPort(22).
		SetSSHUser(sshUser).
		SetSSHKeyPath(sshKeyPath).
		SetWorkspaceRoot(workspaceRoot).
		SetAgentCliPath(agentCLIPath).
		SetStatus(entmachine.StatusOnline).
		Save(ctx); err != nil {
		t.Fatalf("create machine: %v", err)
	}

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-01").
		SetStatus(entagent.StatusClaimed).
		SetCurrentTicketID(ticketItem.ID).
		SetRuntimePhase(entagent.RuntimePhaseNone).
		SetWorkspacePath("/srv/openase/workspaces/ASE-401").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}

	prepareSession := &runtimeSSHPrepareSession{}
	processSession := newRuntimeSSHProcessSession()
	sshPool := sshinfra.NewPool("/tmp/openase",
		sshinfra.WithDialer(&runtimeSSHDialer{client: &runtimeSSHClient{sessions: []sshinfra.Session{prepareSession, processSession}}}),
		sshinfra.WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
	)

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, sshPool, nil)
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
	if agentAfter.SessionID != "thread-runtime-1" {
		t.Fatalf("expected thread-runtime-1 session id, got %q", agentAfter.SessionID)
	}
	if !strings.Contains(prepareSession.command, "git clone --branch 'main' --single-branch 'git@github.com:acme/backend.git' '/srv/openase/workspaces/ASE-401/backend'") {
		t.Fatalf("expected remote workspace clone command, got %q", prepareSession.command)
	}
	if !strings.Contains(processSession.startedCommand, "cd '/srv/openase/workspaces/ASE-401'") {
		t.Fatalf("expected remote process to cd into workspace, got %q", processSession.startedCommand)
	}
	if !strings.Contains(processSession.startedCommand, "'/usr/local/bin/codex'") {
		t.Fatalf("expected machine agent cli path in remote command, got %q", processSession.startedCommand)
	}
}

func TestRuntimeLauncherRunTickFailsWhenRemoteCodexEnvironmentIsNotReady(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

	if _, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-402").
		SetTitle("Launch Codex on remote machine without auth").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	sshUser := "openase"
	sshKeyPath := "keys/gpu-02.pem"
	workspaceRoot := "/srv/openase/workspaces"
	if _, err := client.Machine.Create().
		SetOrganizationID(fixture.orgID).
		SetName("gpu-02").
		SetHost("10.0.1.11").
		SetPort(22).
		SetSSHUser(sshUser).
		SetSSHKeyPath(sshKeyPath).
		SetWorkspaceRoot(workspaceRoot).
		SetStatus(entmachine.StatusOnline).
		SetResources(map[string]any{
			"monitor": map[string]any{
				"l4": map[string]any{
					"codex": map[string]any{
						"installed":   true,
						"auth_status": "not_logged_in",
						"ready":       false,
					},
				},
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("create machine: %v", err)
	}

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-02").
		SetStatus(entagent.StatusClaimed).
		SetCurrentTicketID(ticketItem.ID).
		SetRuntimePhase(entagent.RuntimePhaseNone).
		SetWorkspacePath("/srv/openase/workspaces/ASE-402").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, nil, nil)
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
	if agentAfter.Status != entagent.StatusFailed || agentAfter.RuntimePhase != entagent.RuntimePhaseFailed {
		t.Fatalf("expected failed runtime state, got %+v", agentAfter)
	}
	if !strings.Contains(agentAfter.LastError, "codex cli is not logged in") {
		t.Fatalf("expected codex auth failure in last error, got %q", agentAfter.LastError)
	}
}

func TestRuntimeLauncherRunTickSkipsMachineCodexPreflightForNonCodexCommand(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetCliCommand("python3").
		Save(ctx); err != nil {
		t.Fatalf("update provider command: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-403").
		SetTitle("Launch fake Codex app server").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	localMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(fixture.orgID),
			entmachine.NameEQ(catalogdomain.LocalMachineName),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load local machine: %v", err)
	}
	if _, err := client.Machine.UpdateOneID(localMachine.ID).
		SetResources(map[string]any{
			"monitor": map[string]any{
				"l4": map[string]any{
					"codex": map[string]any{
						"installed":   true,
						"auth_status": "not_logged_in",
						"ready":       false,
					},
				},
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("update local machine resources: %v", err)
	}

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-fake-01").
		SetStatus(entagent.StatusClaimed).
		SetCurrentTicketID(ticketItem.ID).
		SetRuntimePhase(entagent.RuntimePhaseNone).
		SetWorkspacePath("/tmp/openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, nil, nil)
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
	if agentAfter.Status != entagent.StatusRunning || agentAfter.RuntimePhase != entagent.RuntimePhaseReady {
		t.Fatalf("expected ready runtime state, got %+v", agentAfter)
	}
	if agentAfter.SessionID != "thread-runtime-1" {
		t.Fatalf("expected thread-runtime-1 session id, got %q", agentAfter.SessionID)
	}
	if agentAfter.LastError != "" {
		t.Fatalf("expected empty last error, got %q", agentAfter.LastError)
	}
}

func TestRuntimeLauncherRunTickSkipsMachineCodexPreflightWhenAPIKeyIsConfigured(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetAuthConfig(map[string]any{"openai_api_key": "sk-test-runtime"}).
		Save(ctx); err != nil {
		t.Fatalf("update provider auth config: %v", err)
	}

	localMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(fixture.orgID),
			entmachine.NameEQ(catalogdomain.LocalMachineName),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load local machine: %v", err)
	}
	if _, err := client.Machine.UpdateOneID(localMachine.ID).
		SetResources(map[string]any{
			"monitor": map[string]any{
				"l4": map[string]any{
					"codex": map[string]any{
						"installed":   true,
						"auth_status": "not_logged_in",
						"auth_mode":   "login",
						"ready":       false,
					},
				},
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("update local machine resources: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-404").
		SetTitle("Launch Codex with API key auth").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-api-key-01").
		SetStatus(entagent.StatusClaimed).
		SetCurrentTicketID(ticketItem.ID).
		SetRuntimePhase(entagent.RuntimePhaseNone).
		SetWorkspacePath("/tmp/openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}

	manager := &runtimeFakeProcessManager{}
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, manager, nil, nil)
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
	if agentAfter.Status != entagent.StatusRunning || agentAfter.RuntimePhase != entagent.RuntimePhaseReady {
		t.Fatalf("expected ready runtime state, got %+v", agentAfter)
	}

	processSpec := manager.capturedProcessSpec()
	if value, ok := provider.LookupEnvironmentValue(processSpec.Environment, "OPENAI_API_KEY"); !ok || value != "sk-test-runtime" {
		t.Fatalf("expected OPENAI_API_KEY to be injected into runtime environment, got %+v", processSpec.Environment)
	}
}

func TestRequiresMachineCodexReady(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		command string
		env     []string
		want    bool
	}{
		{name: "bare codex", command: "codex", want: true},
		{name: "absolute codex path", command: "/usr/local/bin/codex", want: true},
		{name: "quoted codex path with args", command: `"/Applications/Codex/codex" --version`, want: true},
		{name: "windows codex path", command: `C:\Program Files\Codex\codex.exe`, want: true},
		{name: "codex with api key", command: "codex", env: []string{"OPENAI_API_KEY=sk-test"}, want: false},
		{name: "python", command: "python3", want: false},
		{name: "fake app server wrapper", command: "/usr/bin/python3", want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := requiresMachineCodexReady(provider.MustParseAgentCLICommand(tc.command), tc.env)
			if got != tc.want {
				t.Fatalf("requiresMachineCodexReady(%q, %+v) = %v, want %v", tc.command, tc.env, got, tc.want)
			}
		})
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
	capturedSpec       provider.AgentCLIProcessSpec
	lastProcess        *runtimeFakeProcess
}

func (m *runtimeFakeProcessManager) Start(_ context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	process := newRuntimeFakeProcess()
	if m != nil {
		m.mu.Lock()
		m.capturedSpec = spec
		m.lastProcess = process
		m.mu.Unlock()
	}
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

func (m *runtimeFakeProcessManager) capturedProcessSpec() provider.AgentCLIProcessSpec {
	m.mu.Lock()
	defer m.mu.Unlock()
	return provider.AgentCLIProcessSpec{
		Command:          m.capturedSpec.Command,
		Args:             append([]string(nil), m.capturedSpec.Args...),
		WorkingDirectory: m.capturedSpec.WorkingDirectory,
		Environment:      append([]string(nil), m.capturedSpec.Environment...),
	}
}

func (m *runtimeFakeProcessManager) capturedProcess() *runtimeFakeProcess {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastProcess
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

type runtimeSSHDialer struct {
	client sshinfra.Client
}

func (d *runtimeSSHDialer) DialContext(context.Context, sshinfra.DialConfig) (sshinfra.Client, error) {
	return d.client, nil
}

type runtimeSSHClient struct {
	sessions   []sshinfra.Session
	sessionIdx int
}

func (c *runtimeSSHClient) NewSession() (sshinfra.Session, error) {
	if c.sessionIdx >= len(c.sessions) {
		return nil, fmt.Errorf("unexpected ssh session request %d", c.sessionIdx)
	}
	session := c.sessions[c.sessionIdx]
	c.sessionIdx++
	return session, nil
}

func (c *runtimeSSHClient) SendRequest(string, bool, []byte) (bool, []byte, error) {
	return true, nil, nil
}

func (c *runtimeSSHClient) Close() error {
	return nil
}

type runtimeSSHPrepareSession struct {
	command string
}

func (s *runtimeSSHPrepareSession) CombinedOutput(cmd string) ([]byte, error) {
	s.command = cmd
	return nil, nil
}

func (s *runtimeSSHPrepareSession) StdinPipe() (io.WriteCloser, error) {
	return nil, fmt.Errorf("not supported")
}

func (s *runtimeSSHPrepareSession) StdoutPipe() (io.Reader, error) { return strings.NewReader(""), nil }

func (s *runtimeSSHPrepareSession) StderrPipe() (io.Reader, error) { return strings.NewReader(""), nil }

func (s *runtimeSSHPrepareSession) Start(string) error { return fmt.Errorf("not supported") }

func (s *runtimeSSHPrepareSession) Signal(string) error { return nil }

func (s *runtimeSSHPrepareSession) Wait() error { return nil }

func (s *runtimeSSHPrepareSession) Close() error { return nil }

type runtimeSSHProcessSession struct {
	stdinRead  *io.PipeReader
	stdinWrite *io.PipeWriter

	stdoutRead  *io.PipeReader
	stdoutWrite *io.PipeWriter

	stderrRead  *io.PipeReader
	stderrWrite *io.PipeWriter

	done chan error

	startedCommand string
}

func newRuntimeSSHProcessSession() *runtimeSSHProcessSession {
	stdinRead, stdinWrite := io.Pipe()
	stdoutRead, stdoutWrite := io.Pipe()
	stderrRead, stderrWrite := io.Pipe()
	return &runtimeSSHProcessSession{
		stdinRead:   stdinRead,
		stdinWrite:  stdinWrite,
		stdoutRead:  stdoutRead,
		stdoutWrite: stdoutWrite,
		stderrRead:  stderrRead,
		stderrWrite: stderrWrite,
		done:        make(chan error, 1),
	}
}

func (s *runtimeSSHProcessSession) CombinedOutput(string) ([]byte, error) {
	return nil, fmt.Errorf("not supported")
}

func (s *runtimeSSHProcessSession) StdinPipe() (io.WriteCloser, error) { return s.stdinWrite, nil }

func (s *runtimeSSHProcessSession) StdoutPipe() (io.Reader, error) { return s.stdoutRead, nil }

func (s *runtimeSSHProcessSession) StderrPipe() (io.Reader, error) { return s.stderrRead, nil }

func (s *runtimeSSHProcessSession) Start(cmd string) error {
	s.startedCommand = cmd
	go func() {
		s.done <- runRuntimeSSHHandshake(s)
	}()
	return nil
}

func (s *runtimeSSHProcessSession) Signal(string) error {
	return s.Close()
}

func (s *runtimeSSHProcessSession) Wait() error {
	return <-s.done
}

func (s *runtimeSSHProcessSession) Close() error {
	_ = s.stdinRead.Close()
	_ = s.stdinWrite.Close()
	_ = s.stdoutRead.Close()
	_ = s.stdoutWrite.Close()
	_ = s.stderrRead.Close()
	_ = s.stderrWrite.Close()
	return nil
}

func runRuntimeSSHHandshake(session *runtimeSSHProcessSession) error {
	decoder := json.NewDecoder(session.stdinRead)
	encoder := json.NewEncoder(session.stdoutWrite)

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
	if err := encoder.Encode(runtimeJSONRPCMessage{
		JSONRPC: "2.0",
		ID:      threadStart.ID,
		Result: mustMarshalRuntimeJSON(map[string]any{
			"thread": map[string]any{"id": "thread-runtime-1"},
		}),
	}); err != nil {
		return err
	}

	return nil
}
