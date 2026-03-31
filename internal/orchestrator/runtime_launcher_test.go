package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entagentstepevent "github.com/BetterAndBetterII/openase/ent/agentstepevent"
	entagenttraceevent "github.com/BetterAndBetterII/openase/ent/agenttraceevent"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entprojectrepomirror "github.com/BetterAndBetterII/openase/ent/projectrepomirror"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/httpapi"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	projectrepomirrorsvc "github.com/BetterAndBetterII/openase/internal/projectrepomirror"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
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
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
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
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})
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
	launcher.ConfigurePlatformEnvironment("http://127.0.0.1:19836/api/v1/platform", agentplatform.NewService(client))
	launcher.now = func() time.Time {
		return now
	}
	currentExecutable, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve current executable: %v", err)
	}
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run launcher tick: %v", err)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusReady {
		t.Fatalf("expected ready run, got %+v", runAfter)
	}
	if runAfter.SessionID != "thread-runtime-1" {
		t.Fatalf("expected thread-runtime-1 session id, got %q", runAfter.SessionID)
	}
	if runAfter.RuntimeStartedAt == nil || !runAfter.RuntimeStartedAt.UTC().Equal(now) {
		t.Fatalf("expected runtime_started_at %s, got %+v", now.Format(time.RFC3339), runAfter.RuntimeStartedAt)
	}
	if runAfter.LastHeartbeatAt == nil || !runAfter.LastHeartbeatAt.UTC().Equal(now) {
		t.Fatalf("expected last_heartbeat_at %s, got %+v", now.Format(time.RFC3339), runAfter.LastHeartbeatAt)
	}
	if runAfter.LastError != "" {
		t.Fatalf("expected empty last_error, got %q", runAfter.LastError)
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
	expectedLocalWorkspaceRoot, err := workspaceinfra.LocalWorkspaceRoot()
	if err != nil {
		t.Fatalf("resolve local workspace root: %v", err)
	}
	if !strings.Contains(manager.capturedThreadStart().DeveloperInstructions, "Current local root="+expectedLocalWorkspaceRoot) {
		t.Fatalf("expected rendered current machine in developer instructions, got %q", manager.capturedThreadStart().DeveloperInstructions)
	}
	if !strings.Contains(manager.capturedThreadStart().DeveloperInstructions, "storage=openase@10.0.1.20|") {
		t.Fatalf("expected whitelisted machine in developer instructions, got %q", manager.capturedThreadStart().DeveloperInstructions)
	}
	if strings.Contains(manager.capturedThreadStart().DeveloperInstructions, "dev-01=") {
		t.Fatalf("expected non-whitelisted machine to stay out of developer instructions, got %q", manager.capturedThreadStart().DeveloperInstructions)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".openase", "skills", "openase-platform", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("expected built-in platform skill to stay out of primary repo authority paths, stat err=%v", err)
	}
	workspacePath, err := workspaceinfra.TicketWorkspacePath(
		expectedLocalWorkspaceRoot,
		"better-and-better",
		"openase",
		ticketItem.Identifier,
	)
	if err != nil {
		t.Fatalf("resolve workspace path: %v", err)
	}
	repoWorkspacePath := filepath.Join(
		workspacePath,
		"repo-"+strings.ReplaceAll(fixture.projectID.String(), "-", "")[:8],
	)
	if manager.capturedProcessSpec().WorkingDirectory == nil || manager.capturedProcessSpec().WorkingDirectory.String() != repoWorkspacePath {
		t.Fatalf("expected process working directory %s, got %+v", repoWorkspacePath, manager.capturedProcessSpec().WorkingDirectory)
	}
	if _, err := os.Stat(filepath.Join(repoWorkspacePath, ".codex", "skills", "openase-platform", "SKILL.md")); err != nil {
		t.Fatalf("expected platform skill in codex workspace: %v", err)
	}
	// #nosec G304 -- test reads a fixture from the temp workspace path created above.
	workspaceSkillContent, err := os.ReadFile(filepath.Join(repoWorkspacePath, ".codex", "skills", "openase-platform", "SKILL.md"))
	if err != nil {
		t.Fatalf("read codex workspace platform skill: %v", err)
	}
	if !strings.HasPrefix(string(workspaceSkillContent), "---\nname: ") {
		t.Fatalf("expected codex workspace platform skill to include frontmatter, got %q", string(workspaceSkillContent))
	}
	if _, err := os.Stat(filepath.Join(repoWorkspacePath, ".openase", "bin", "openase")); err != nil {
		t.Fatalf("expected openase wrapper in codex workspace: %v", err)
	}
	processEnvironment := manager.capturedProcessSpec().Environment
	if !containsEnvironmentPrefix(processEnvironment, "OPENASE_API_URL=http://127.0.0.1:19836/api/v1/platform") {
		t.Fatalf("expected OPENASE_API_URL in process environment, got %+v", processEnvironment)
	}
	if !containsEnvironmentPrefix(processEnvironment, "OPENASE_PROJECT_ID="+fixture.projectID.String()) {
		t.Fatalf("expected OPENASE_PROJECT_ID in process environment, got %+v", processEnvironment)
	}
	if !containsEnvironmentPrefix(processEnvironment, "OPENASE_TICKET_ID="+ticketItem.ID.String()) {
		t.Fatalf("expected OPENASE_TICKET_ID in process environment, got %+v", processEnvironment)
	}
	if !containsEnvironmentPrefix(processEnvironment, "OPENASE_AGENT_TOKEN=ase_agent_") {
		t.Fatalf("expected OPENASE_AGENT_TOKEN in process environment, got %+v", processEnvironment)
	}
	if !containsEnvironmentPrefix(processEnvironment, "OPENASE_REAL_BIN="+currentExecutable) {
		t.Fatalf("expected OPENASE_REAL_BIN in process environment, got %+v", processEnvironment)
	}
	repoWorkspace, err := client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.AgentRunIDEQ(runItem.ID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("load ticket repo workspace: %v", err)
	}
	if repoWorkspace.State != entticketrepoworkspace.StateReady {
		t.Fatalf("expected ready ticket repo workspace, got %+v", repoWorkspace)
	}
	if repoWorkspace.MirrorID == uuid.Nil || repoWorkspace.PreparedAt == nil {
		t.Fatalf("expected persisted ticket repo workspace metadata, got %+v", repoWorkspace)
	}
}

func TestRuntimeLauncherRunTickLaunchesConcurrentRunsForSameAgent(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 1, 0, 0, time.UTC)

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	if err := os.WriteFile(harnessPath, []byte(`---
workflow:
  role: coding
---

	Parallel runtime launch test
	`), 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-parallel-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	ticketOne, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-401A").
		SetTitle("Launch first run").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create first ticket: %v", err)
	}
	ticketTwo, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-401B").
		SetTitle("Launch second run").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create second ticket: %v", err)
	}

	runOne := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketOne.ID, entagentrun.StatusLaunching, time.Time{})
	runTwo := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketTwo.ID, entagentrun.StatusLaunching, time.Time{})
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

	runOneAfter, err := client.AgentRun.Get(ctx, runOne.ID)
	if err != nil {
		t.Fatalf("reload first run: %v", err)
	}
	runTwoAfter, err := client.AgentRun.Get(ctx, runTwo.ID)
	if err != nil {
		t.Fatalf("reload second run: %v", err)
	}
	if runOneAfter.Status != entagentrun.StatusReady || runTwoAfter.Status != entagentrun.StatusReady {
		t.Fatalf("expected both runs ready, got first=%+v second=%+v", runOneAfter, runTwoAfter)
	}
	if launcher.loadSession(runOne.ID) == nil || launcher.loadSession(runTwo.ID) == nil {
		t.Fatal("expected concurrent runs to keep separate cached sessions")
	}

	readyEvents, err := client.ActivityEvent.Query().
		Where(
			entactivityevent.AgentIDEQ(agentItem.ID),
			entactivityevent.EventTypeEQ(agentReadyType.String()),
		).
		All(ctx)
	if err != nil {
		t.Fatalf("list ready activity events: %v", err)
	}
	if len(readyEvents) != 2 {
		t.Fatalf("expected 2 ready activity events, got %+v", readyEvents)
	}

	readyRuns := map[string]bool{}
	for _, event := range readyEvents {
		runID, ok := event.Metadata["run_id"].(string)
		if !ok || strings.TrimSpace(runID) == "" {
			t.Fatalf("expected ready event metadata to include run_id, got %+v", event.Metadata)
		}
		readyRuns[runID] = true
	}
	if !readyRuns[runOne.ID.String()] || !readyRuns[runTwo.ID.String()] {
		t.Fatalf("expected ready activity metadata for runs %s and %s, got %+v", runOne.ID, runTwo.ID, readyRuns)
	}
}

func TestRuntimeLauncherRunTickContinuesWhenLifecyclePublishBlocks(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 2, 0, 0, time.UTC)

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

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	if err := os.WriteFile(harnessPath, []byte(`---
workflow:
  role: coding
---

Blocked lifecycle publish regression test.
`), 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-401C").
		SetTitle("Blocked lifecycle publish").
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
		SetName("codex-blocked-events-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	manager := &runtimeFakeProcessManager{}
	events := newRuntimeBlockingEventProvider(agentLaunchingType, agentReadyType, agentHeartbeatType)
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), events, manager, nil, workflowSvc)
	launcher.now = func() time.Time { return now }
	launcher.eventTimeout = 20 * time.Millisecond
	t.Cleanup(func() {
		events.Release()
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run launcher tick: %v", err)
	}

	waitForRuntimeCondition(t, time.Second, func() bool {
		runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
		return err == nil && runAfter.Status == entagentrun.StatusReady && runAfter.SessionID == "thread-runtime-1"
	})

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusReady || runAfter.SessionID != "thread-runtime-1" {
		t.Fatalf("expected ready run after blocked publish, got %+v", runAfter)
	}
	waitForRuntimeCondition(t, time.Second, func() bool {
		activityCount, err := client.ActivityEvent.Query().
			Where(entactivityevent.AgentIDEQ(agentItem.ID)).
			Count(ctx)
		return err == nil && activityCount >= 2
	})

	activityTypes, err := client.ActivityEvent.Query().
		Where(entactivityevent.AgentIDEQ(agentItem.ID)).
		Order(ent.Asc(entactivityevent.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		t.Fatalf("list activity events after blocked publish: %v", err)
	}
	if len(activityTypes) != 2 {
		t.Fatalf("expected only canonical lifecycle activity events to persist, got %+v", activityTypes)
	}
	for _, item := range activityTypes {
		if item.EventType == agentHeartbeatType.String() {
			t.Fatalf("expected heartbeat lifecycle event to stay out of activity persistence, got %+v", activityTypes)
		}
	}
}

func TestRuntimeLauncherStartRuntimeSessionSupportsGeminiProvider(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 31, 11, 0, 0, 0, time.UTC)

	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetName("Gemini").
		SetAdapterType(entagentprovider.AdapterTypeGeminiCli).
		SetCliCommand("gemini").
		SetModelName("gemini-2.5-pro").
		Save(ctx); err != nil {
		t.Fatalf("update provider to gemini: %v", err)
	}

	process := newRuntimeRunnerFakeProcess()
	manager := &geminiAdapterTestProcessManager{process: process}
	workflowItem, workflowSvc, ticketItem, agentItem, runItem, launcher := newRuntimeExecutionFixture(
		ctx,
		t,
		client,
		fixture,
		now,
		"Gemini Coding",
		"ASE-402",
		"Align Gemini orchestrator runtime",
		"gemini-runner-01",
		"Implement the ticket using Gemini.",
		manager,
	)
	_ = workflowItem
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- runGeminiRuntimeProtocol(process)
	}()

	session, err := launcher.startRuntimeSession(ctx, runtimeAssignment{
		ticket: ticketItem,
		agent:  agentItem,
		run:    runItem,
	})
	if err != nil {
		t.Fatalf("startRuntimeSession() error = %v", err)
	}

	turn, err := session.SendPrompt(context.Background(), "Implement the fix.")
	if err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
	if turn.TurnID == "" {
		t.Fatal("SendPrompt() turn id = empty, want Gemini turn id")
	}

	first := requireAgentEvent(t, session.Events())
	if first.Type != agentEventTypeOutputProduced || first.Output == nil || first.Output.Text != "Implemented the shared contract." {
		t.Fatalf("unexpected first event: %+v", first)
	}
	second := requireAgentEvent(t, session.Events())
	if second.Type != agentEventTypeTurnCompleted || second.Turn == nil || second.Turn.TurnID != turn.TurnID {
		t.Fatalf("unexpected second event: %+v", second)
	}

	if manager.capturedSpec.Command != provider.MustParseAgentCLICommand("gemini") {
		t.Fatalf("process command = %q, want gemini", manager.capturedSpec.Command)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("fake gemini process returned error: %v", err)
	}
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

func TestRuntimeLauncherRunTickDoesNotStarveLaterLaunchesWhenFirstStartBlocks(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 3, 0, 0, time.UTC)

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	if err := os.WriteFile(harnessPath, []byte(`---
workflow:
  role: coding
---

Launch starvation regression test.
`), 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-blocked-start-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	ticketOne, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-401D").
		SetTitle("First blocked launch").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create first ticket: %v", err)
	}
	ticketTwo, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-401E").
		SetTitle("Second launch should continue").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create second ticket: %v", err)
	}
	runOne := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketOne.ID, entagentrun.StatusLaunching, time.Time{})
	runTwo := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketTwo.ID, entagentrun.StatusLaunching, time.Time{})
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

	baseManager := &runtimeFakeProcessManager{}
	manager := newRuntimeSequencedProcessManager(baseManager)
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, manager, nil, workflowSvc)
	launcher.now = func() time.Time { return now }
	t.Cleanup(func() {
		manager.ReleaseFirst()
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- launcher.RunTick(ctx)
	}()

	select {
	case <-manager.SecondStarted():
	case <-time.After(2 * time.Second):
		manager.ReleaseFirst()
		t.Fatal("timed out waiting for second launch to start while the first launch was blocked")
	}

	manager.ReleaseFirst()
	if err := <-runErrCh; err != nil {
		t.Fatalf("run launcher tick: %v", err)
	}

	waitForRuntimeCondition(t, time.Second, func() bool {
		runOneAfter, errOne := client.AgentRun.Get(ctx, runOne.ID)
		runTwoAfter, errTwo := client.AgentRun.Get(ctx, runTwo.ID)
		return errOne == nil && errTwo == nil &&
			runOneAfter.Status == entagentrun.StatusReady &&
			runTwoAfter.Status == entagentrun.StatusReady
	})
}

func TestRuntimeLauncherRunTickLaunchTimeoutMarksRunErrored(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

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

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	if err := os.WriteFile(harnessPath, []byte(`---
workflow:
  role: coding
---

Blocked launch should time out cleanly.
`), 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-330C").
		SetTitle("Time out blocked launch").
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
		SetName("codex-timeout-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})
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

	manager := &runtimeBlockingStartProcessManager{
		releaseFirstStart: make(chan struct{}),
		firstStartSeen:    make(chan struct{}, 1),
		laterStartSeen:    make(chan struct{}, 1),
	}
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, manager, nil, workflowSvc)
	launcher.launchTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		manager.release()
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run launcher tick: %v", err)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusErrored {
		t.Fatalf("expected errored run after launch timeout, got %+v", runAfter)
	}
	if !strings.Contains(runAfter.LastError, "timed out after 50ms") {
		t.Fatalf("expected launch timeout in last error, got %q", runAfter.LastError)
	}
}

func TestRuntimeLauncherCloseClearsTicketCurrentRunOnGracefulShutdown(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

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
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-402").
		SetTitle("Release graceful shutdown claim").
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
		SetName("codex-close-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, nil, nil, nil)
	launcher.storeSession(runItem.ID, nil)

	if err := launcher.Close(ctx); err != nil {
		t.Fatalf("close launcher: %v", err)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after close: %v", err)
	}
	if ticketAfter.StatusID != fixture.statusIDs["Todo"] {
		t.Fatalf("expected graceful shutdown to keep status Todo, got %+v", ticketAfter.StatusID)
	}
	if ticketAfter.CurrentRunID != nil {
		t.Fatalf("expected graceful shutdown to clear current run, got %+v", ticketAfter.CurrentRunID)
	}
	if got := statusActiveRuns(ctx, t, client, fixture.projectID, "Todo"); got != 0 {
		t.Fatalf("expected graceful shutdown to drop todo status occupancy to 0, got %d", got)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run after close: %v", err)
	}
	if runAfter.Status != entagentrun.StatusTerminated {
		t.Fatalf("expected graceful shutdown to terminate run, got %+v", runAfter)
	}

	terminatedEvent := waitForAgentLifecycleEvent(t, stream, agentTerminatedType)
	payload := decodeLifecycleEnvelope(t, terminatedEvent.Payload)
	if payload.Agent.ID != agentItem.ID.String() {
		t.Fatalf("unexpected terminated event payload: %+v", payload.Agent)
	}
	if payload.Agent.CurrentRunID != nil {
		t.Fatalf("expected terminated event to publish cleared current_run_id, got %+v", payload.Agent.CurrentRunID)
	}
}

func TestRuntimeLauncherFinishResolvedExecutionReleasesStageOccupancy(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 5, 0, 0, time.UTC)

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

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-finish-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-402B").
		SetTitle("Finish runtime execution").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	if got := statusActiveRuns(ctx, t, client, fixture.projectID, "Todo"); got != 1 {
		t.Fatalf("expected active todo status occupancy before finish, got %d", got)
	}

	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).SetStatusID(fixture.statusIDs["Done"]).Save(ctx); err != nil {
		t.Fatalf("mark ticket done: %v", err)
	}
	retryAt := now.Add(15 * time.Minute)
	seedRetryToken := uuid.NewString()
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetAttemptCount(3).
		SetConsecutiveErrors(2).
		SetNextRetryAt(retryAt).
		SetRetryPaused(true).
		SetPauseReason(ticketing.PauseReasonBudgetExhausted.String()).
		SetRetryToken(seedRetryToken).
		Save(ctx); err != nil {
		t.Fatalf("seed retry baseline before finish: %v", err)
	}
	resolvedTicket, err := client.Ticket.Query().
		Where(entticket.IDEQ(ticketItem.ID)).
		WithCurrentRun().
		WithWorkflow().
		WithStatus().
		Only(ctx)
	if err != nil {
		t.Fatalf("reload resolved ticket: %v", err)
	}

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil, nil)
	launcher.now = func() time.Time {
		return now
	}
	if err := launcher.finishResolvedExecution(ctx, runItem.ID, agentItem.ID, resolvedTicket); err != nil {
		t.Fatalf("finish resolved execution: %v", err)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID != nil {
		t.Fatalf("expected finish to clear current run, got %+v", ticketAfter.CurrentRunID)
	}
	if ticketAfter.CompletedAt == nil {
		t.Fatalf("expected finish to stamp completed_at, got %+v", ticketAfter)
	}
	if ticketAfter.AttemptCount != 3 || ticketAfter.ConsecutiveErrors != 0 || ticketAfter.NextRetryAt != nil || ticketAfter.RetryPaused || ticketAfter.PauseReason != "" {
		t.Fatalf("expected finish to normalize retry baseline, got %+v", ticketAfter)
	}
	if ticketAfter.RetryToken == "" || ticketAfter.RetryToken == seedRetryToken {
		t.Fatalf("expected finish to rotate retry token, got %q", ticketAfter.RetryToken)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusCompleted {
		t.Fatalf("expected completed run after finish, got %+v", runAfter)
	}
	if got := statusActiveRuns(ctx, t, client, fixture.projectID, "Todo"); got != 0 {
		t.Fatalf("expected finish to drop todo status occupancy to 0, got %d", got)
	}
}

func TestRuntimeLauncherReleaseExecutionOwnershipPreservesTicketStatusAndCompletion(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 35, 0, 0, time.UTC)

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

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-release-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-402C").
		SetTitle("Release runtime ownership").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	if got := statusActiveRuns(ctx, t, client, fixture.projectID, "Todo"); got != 1 {
		t.Fatalf("expected active todo status occupancy before release, got %d", got)
	}

	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).SetStatusID(fixture.statusIDs["In Review"]).Save(ctx); err != nil {
		t.Fatalf("mark ticket in review: %v", err)
	}
	reloadedTicket, err := client.Ticket.Query().
		Where(entticket.IDEQ(ticketItem.ID)).
		WithCurrentRun().
		Only(ctx)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil, nil)
	launcher.now = func() time.Time { return now }
	if err := launcher.releaseExecutionOwnership(ctx, runItem.ID, agentItem.ID, reloadedTicket); err != nil {
		t.Fatalf("release execution ownership: %v", err)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.StatusID != fixture.statusIDs["In Review"] {
		t.Fatalf("expected status to remain In Review %s, got %s", fixture.statusIDs["In Review"], ticketAfter.StatusID)
	}
	if ticketAfter.CurrentRunID != nil {
		t.Fatalf("expected release to clear current run, got %+v", ticketAfter.CurrentRunID)
	}
	if ticketAfter.CompletedAt != nil {
		t.Fatalf("expected release to preserve nil completed_at, got %+v", ticketAfter.CompletedAt)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusTerminated {
		t.Fatalf("expected terminated run after release, got %+v", runAfter)
	}
	if got := statusActiveRuns(ctx, t, client, fixture.projectID, "Todo"); got != 0 {
		t.Fatalf("expected release to drop todo status occupancy to 0, got %d", got)
	}
}

func TestRuntimeLauncherFinishResolvedExecutionAutoAppliesSingleFinishStatus(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 22, 10, 0, 0, 0, time.UTC)

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
	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflowItem.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-91").
		SetTitle("Auto finish").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	resolvedTicket, err := client.Ticket.Query().
		Where(entticket.IDEQ(ticketItem.ID)).
		WithCurrentRun().
		WithWorkflow(func(query *ent.WorkflowQuery) {
			query.WithFinishStatuses()
		}).
		Only(ctx)
	if err != nil {
		t.Fatalf("reload resolved ticket: %v", err)
	}

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil, nil)
	launcher.now = func() time.Time { return now }
	if err := launcher.finishResolvedExecution(ctx, runItem.ID, agentItem.ID, resolvedTicket); err != nil {
		t.Fatalf("finish resolved execution: %v", err)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.StatusID != fixture.statusIDs["Done"] {
		t.Fatalf("expected auto finish status %s, got %s", fixture.statusIDs["Done"], ticketAfter.StatusID)
	}
	if ticketAfter.CompletedAt == nil {
		t.Fatalf("expected completed_at after auto finish, got %+v", ticketAfter)
	}
}

func TestRuntimeLauncherFinishResolvedExecutionRequiresExplicitFinishChoiceWhenMultipleAllowed(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 22, 10, 30, 0, 0, time.UTC)

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"], fixture.statusIDs["In Review"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflowItem.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-92").
		SetTitle("Need explicit finish").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	resolvedTicket, err := client.Ticket.Query().
		Where(entticket.IDEQ(ticketItem.ID)).
		WithCurrentRun().
		WithWorkflow(func(query *ent.WorkflowQuery) {
			query.WithFinishStatuses()
		}).
		Only(ctx)
	if err != nil {
		t.Fatalf("reload resolved ticket: %v", err)
	}

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil, nil)
	launcher.now = func() time.Time { return now }
	if err := launcher.finishResolvedExecution(ctx, runItem.ID, agentItem.ID, resolvedTicket); err == nil {
		t.Fatalf("expected missing explicit finish selection to fail")
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.StatusID != fixture.statusIDs["Todo"] || ticketAfter.CompletedAt != nil || ticketAfter.CurrentRunID == nil {
		t.Fatalf("expected ticket to remain unresolved without explicit finish selection, got %+v", ticketAfter)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusExecuting {
		t.Fatalf("expected direct finish helper error to leave run untouched, got %+v", runAfter)
	}
}

func TestRuntimeLauncherRunTickDropsCachedSessionWhenAgentLeavesRunningState(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

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

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
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
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})
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
	runAfterLaunch, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload launched run: %v", err)
	}
	if runAfterLaunch.Status != entagentrun.StatusReady || runAfterLaunch.SessionID != "thread-runtime-1" {
		t.Fatalf("expected ready run after launch, got %+v", runAfterLaunch)
	}

	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		ClearCurrentRunID().
		Save(ctx); err != nil {
		t.Fatalf("clear ticket runtime assignment in db: %v", err)
	}

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("reconcile non-running runtime: %v", err)
	}

	if launcher.loadSession(runItem.ID) != nil {
		t.Fatal("expected non-running agent session to be removed from cache")
	}
}

func TestRuntimeLauncherRunTickExecutesTurnsRecordsUsageAndSchedulesContinuation(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC)

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

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	if err := os.WriteFile(harnessPath, []byte(`---
workflow:
  role: coding
---

Implement the ticket using the current workspace.
`), 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-410").
		SetTitle("Execute Codex turns").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetStallCount(2).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	originalRetryToken := ticketItem.RetryToken

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-runner-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	manager := &runtimeFakeProcessManager{
		turnInputDelta:  5,
		turnOutputDelta: 2,
		executionDone:   make(chan struct{}),
	}
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, manager, nil, workflowSvc)
	launcher.now = func() time.Time {
		return now
	}
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("launch runtime session: %v", err)
	}
	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("start ready execution: %v", err)
	}

	select {
	case <-manager.executionDone:
	case <-time.After(5 * time.Second):
		agentSnapshot, _ := client.Agent.Get(ctx, agentItem.ID)
		ticketSnapshot, _ := client.Ticket.Get(ctx, ticketItem.ID)
		t.Fatalf(
			"timed out waiting for execution continuation scheduling: turns=%d agent=%+v ticket=%+v",
			manager.capturedTurnCount(),
			agentSnapshot,
			ticketSnapshot,
		)
	}
	waitForRuntimeCondition(t, 5*time.Second, func() bool {
		ticketSnapshot, err := client.Ticket.Get(ctx, ticketItem.ID)
		if err != nil {
			return false
		}
		runSnapshot, err := client.AgentRun.Get(ctx, runItem.ID)
		if err != nil {
			return false
		}
		return ticketSnapshot.CurrentRunID == nil &&
			ticketSnapshot.NextRetryAt != nil &&
			runSnapshot.Status == entagentrun.StatusTerminated
	})

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.NextRetryAt == nil || !ticketAfter.NextRetryAt.UTC().Equal(now.Add(continuationRetryDelay)) {
		t.Fatalf("expected next retry at %s, got %+v", now.Add(continuationRetryDelay), ticketAfter.NextRetryAt)
	}
	if ticketAfter.RetryToken == "" || ticketAfter.RetryToken == originalRetryToken {
		t.Fatalf("expected continuation scheduling to rotate retry token, got %q", ticketAfter.RetryToken)
	}
	if ticketAfter.StallCount != 0 {
		t.Fatalf("expected continuation to reset stall count, got %d", ticketAfter.StallCount)
	}
	if ticketAfter.CostTokensInput != int64(defaultRuntimeMaxTurns)*manager.turnInputDelta {
		t.Fatalf("expected input tokens %d, got %d", int64(defaultRuntimeMaxTurns)*manager.turnInputDelta, ticketAfter.CostTokensInput)
	}
	if ticketAfter.CostTokensOutput != int64(defaultRuntimeMaxTurns)*manager.turnOutputDelta {
		t.Fatalf("expected output tokens %d, got %d", int64(defaultRuntimeMaxTurns)*manager.turnOutputDelta, ticketAfter.CostTokensOutput)
	}
	if ticketAfter.StartedAt == nil || !ticketAfter.StartedAt.UTC().Equal(now) {
		t.Fatalf("expected started_at %s, got %+v", now.Format(time.RFC3339), ticketAfter.StartedAt)
	}

	if ticketAfter.CurrentRunID != nil {
		t.Fatalf("expected continuation scheduling to clear current run, got %+v", ticketAfter.CurrentRunID)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusTerminated {
		t.Fatalf("expected terminated run after continuation scheduling, got %+v", runAfter)
	}
	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.TotalTokensUsed != int64(defaultRuntimeMaxTurns)*(manager.turnInputDelta+manager.turnOutputDelta) {
		t.Fatalf("expected total tokens %d, got %d", int64(defaultRuntimeMaxTurns)*(manager.turnInputDelta+manager.turnOutputDelta), agentAfter.TotalTokensUsed)
	}
	if manager.capturedTurnCount() != defaultRuntimeMaxTurns {
		t.Fatalf("expected %d turns, got %d", defaultRuntimeMaxTurns, manager.capturedTurnCount())
	}
	outputCount, err := client.ActivityEvent.Query().
		Where(
			entactivityevent.AgentIDEQ(agentItem.ID),
			entactivityevent.EventTypeEQ(catalogdomain.AgentOutputEventType),
		).
		Count(ctx)
	if err != nil {
		t.Fatalf("count agent output activity events: %v", err)
	}
	if outputCount != 0 {
		t.Fatalf("expected token-only execution to persist no agent output events, got %d", outputCount)
	}
}

func TestRuntimeLauncherExposesAgentOutputViaHTTPAndSSEDuringExecution(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 23, 10, 30, 0, 0, time.UTC)

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

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	if err := os.WriteFile(harnessPath, []byte(`---
workflow:
  role: coding
---

Emit visible runtime output.
`), 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-412").
		SetTitle("Expose runtime output").
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
		SetName("codex-output-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	manager := &runtimeFakeProcessManager{
		agentMessageDelta:  "Thinking through the fix.",
		commandOutputDelta: "go test ./...\n",
		releaseTurn:        make(chan struct{}),
	}
	bus := eventinfra.NewChannelBus()
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, manager, nil, workflowSvc)
	launcher.now = func() time.Time {
		return now
	}
	t.Cleanup(func() {
		select {
		case <-manager.releaseTurn:
		default:
			close(manager.releaseTurn)
		}
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	catalogSvc := catalogservice.New(catalogrepo.NewEntRepository(client), nil, nil)
	server := httpapi.NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		nil,
		nil,
		nil,
		catalogSvc,
		nil,
	)
	testServer := httptest.NewServer(server.Handler())
	t.Cleanup(testServer.Close)

	response, cancel := openRuntimeSSERequest(t, testServer.URL+"/api/v1/projects/"+fixture.projectID.String()+"/agents/"+agentItem.ID.String()+"/output/stream")
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close agent output stream response body: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("launch runtime session: %v", err)
	}
	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("start ready execution: %v", err)
	}

	waitForRuntimeCondition(t, 5*time.Second, func() bool {
		runSnapshot, err := client.AgentRun.Get(ctx, runItem.ID)
		if err != nil {
			return false
		}
		outputCount, err := client.AgentTraceEvent.Query().
			Where(
				entagenttraceevent.AgentID(agentItem.ID),
				entagenttraceevent.KindIn(catalogdomain.AgentTraceOutputKinds()...),
			).
			Count(ctx)
		if err != nil {
			return false
		}
		stepCount, err := client.AgentStepEvent.Query().
			Where(entagentstepevent.AgentID(agentItem.ID)).
			Count(ctx)
		if err != nil {
			return false
		}
		return runSnapshot.Status == entagentrun.StatusExecuting &&
			outputCount >= 2 &&
			runSnapshot.CurrentStepStatus != nil &&
			stepCount >= 1
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/projects/"+fixture.projectID.String()+"/agents/"+agentItem.ID.String()+"/output?ticket_id="+ticketItem.ID.String(),
		nil,
	)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected output list 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Thinking through the fix.") || !strings.Contains(rec.Body.String(), "go test ./...") {
		t.Fatalf("expected executing runtime output in list API, got %s", rec.Body.String())
	}

	body := readRuntimeSSEBody(t, response, cancel)
	if !strings.Contains(body, "\"topic\":\"agent.output.events\"") {
		t.Fatalf("expected dedicated agent output topic, got %q", body)
	}
	if !strings.Contains(body, "Thinking through the fix.") {
		t.Fatalf("expected streamed agent output delta, got %q", body)
	}

	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetStatusID(fixture.statusIDs["In Review"]).
		Save(ctx); err != nil {
		t.Fatalf("mark ticket in review: %v", err)
	}
	close(manager.releaseTurn)

	waitForRuntimeCondition(t, 5*time.Second, func() bool {
		runSnapshot, err := client.AgentRun.Get(ctx, runItem.ID)
		if err != nil {
			return false
		}
		ticketSnapshot, err := client.Ticket.Get(ctx, ticketItem.ID)
		if err != nil {
			return false
		}
		return ticketSnapshot.CurrentRunID == nil &&
			ticketSnapshot.StatusID == fixture.statusIDs["In Review"] &&
			runSnapshot.Status == entagentrun.StatusTerminated
	})

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after status handoff: %v", err)
	}
	if ticketAfter.StatusID != fixture.statusIDs["In Review"] {
		t.Fatalf("expected runner to preserve In Review status %s, got %s", fixture.statusIDs["In Review"], ticketAfter.StatusID)
	}
	if ticketAfter.CompletedAt != nil {
		t.Fatalf("expected normal turn exit to keep completed_at unset, got %+v", ticketAfter.CompletedAt)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run after status handoff: %v", err)
	}
	if runAfter.Status != entagentrun.StatusTerminated {
		t.Fatalf("expected terminated run after leaving pickup, got %+v", runAfter)
	}
}

func TestRuntimeLauncherRunTickStopsRuntimeWhenTicketBecomesTerminal(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 24, 14, 0, 0, 0, time.UTC)

	manager := &runtimeFakeProcessManager{releaseTurn: make(chan struct{})}
	workflowItem, workflowSvc, ticketItem, agentItem, runItem, launcher := newRuntimeExecutionFixture(
		ctx,
		t,
		client,
		fixture,
		now,
		"Terminal reconciliation",
		"ASE-413",
		"Stop when ticket becomes terminal",
		"codex-terminal-01",
		"Keep working until the orchestrator stops the session.",
		manager,
	)
	_ = workflowItem
	defer func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	}()
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("launch runtime session: %v", err)
	}
	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("start ready execution: %v", err)
	}

	waitForRuntimeExecuting(ctx, t, client, ticketItem.ID, runItem.ID)

	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetStatusID(fixture.statusIDs["Done"]).
		SetCompletedAt(now.Add(time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("mark ticket terminal: %v", err)
	}

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("reconcile terminal ticket: %v", err)
	}

	waitForRuntimeCondition(t, 5*time.Second, func() bool {
		ticketSnapshot, err := client.Ticket.Get(ctx, ticketItem.ID)
		if err != nil {
			return false
		}
		runSnapshot, err := client.AgentRun.Get(ctx, runItem.ID)
		if err != nil {
			return false
		}
		return ticketSnapshot.CurrentRunID == nil &&
			ticketSnapshot.StatusID == fixture.statusIDs["Done"] &&
			runSnapshot.Status == entagentrun.StatusTerminated
	})

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.StatusID != fixture.statusIDs["Done"] || ticketAfter.CompletedAt == nil {
		t.Fatalf("expected terminal ticket state to be preserved, got %+v", ticketAfter)
	}
	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("expected agent control state active after terminal reconciliation, got %+v", agentAfter)
	}
}

func TestRuntimeLauncherRunTickStopsRuntimeWhenWorkflowRoutingChanges(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 24, 14, 30, 0, 0, time.UTC)

	manager := &runtimeFakeProcessManager{releaseTurn: make(chan struct{})}
	workflowItem, workflowSvc, ticketItem, agentItem, runItem, launcher := newRuntimeExecutionFixture(
		ctx,
		t,
		client,
		fixture,
		now,
		"Routing reconciliation",
		"ASE-414",
		"Stop when workflow routing changes",
		"codex-routing-01",
		"Keep working until workflow routing changes.",
		manager,
	)
	defer func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	}()
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	otherWorkflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Alternate Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/alternate.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create alternate workflow: %v", err)
	}

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("launch runtime session: %v", err)
	}
	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("start ready execution: %v", err)
	}

	waitForRuntimeExecuting(ctx, t, client, ticketItem.ID, runItem.ID)

	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetWorkflowID(otherWorkflow.ID).
		Save(ctx); err != nil {
		t.Fatalf("change ticket workflow: %v", err)
	}

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("reconcile workflow drift: %v", err)
	}

	waitForRuntimeCondition(t, 5*time.Second, func() bool {
		ticketSnapshot, err := client.Ticket.Get(ctx, ticketItem.ID)
		if err != nil {
			return false
		}
		runSnapshot, err := client.AgentRun.Get(ctx, runItem.ID)
		if err != nil {
			return false
		}
		return ticketSnapshot.CurrentRunID == nil &&
			ticketSnapshot.WorkflowID != nil &&
			*ticketSnapshot.WorkflowID == otherWorkflow.ID &&
			runSnapshot.Status == entagentrun.StatusTerminated
	})

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.StatusID != fixture.statusIDs["Todo"] {
		t.Fatalf("expected ticket status to stay active after workflow drift, got %+v", ticketAfter)
	}
	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("expected agent control state active after routing reconciliation, got %+v", agentAfter)
	}
	_ = workflowItem
}

func TestRuntimeLauncherRunTickSchedulesContinuationAfterCleanSessionExit(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 24, 15, 0, 0, 0, time.UTC)

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Runtime fact reconciliation").
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
		SetIdentifier("ASE-415").
		SetTitle("Continue after clean session exit").
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
		SetName("codex-runtime-fact-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, nil, nil)
	launcher.now = func() time.Time {
		return now
	}
	runtimeState := NewRuntimeStateStore()
	launcher.ConfigureRuntimeState(runtimeState)
	runtimeState.markReady(runItem.ID, agentItem.ID, ticketItem.ID, workflowItem.ID, "thread-runtime-1", now)
	runtimeState.recordTurnStart(runItem.ID, 1, now)
	runtimeState.recordRuntimeFact(runItem.ID, runtimeFactSessionExited, now, "")

	if err := launcher.reconcileRuntimeFacts(ctx); err != nil {
		t.Fatalf("reconcile runtime fact: %v", err)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.NextRetryAt == nil || !ticketAfter.NextRetryAt.UTC().Equal(now.Add(continuationRetryDelay)) {
		t.Fatalf("expected continuation retry at %s, got %+v", now.Add(continuationRetryDelay), ticketAfter.NextRetryAt)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusTerminated {
		t.Fatalf("expected terminated run after runtime fact continuation, got %+v", runAfter)
	}
}

func TestRuntimeLauncherRunTickStallsByLastCodexTimestamp(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 24, 15, 30, 0, 0, time.UTC)

	manager := &runtimeFakeProcessManager{releaseTurn: make(chan struct{})}
	workflowItem, workflowSvc, ticketItem, _, runItem, launcher := newRuntimeExecutionFixture(
		ctx,
		t,
		client,
		fixture,
		now,
		"Stall reconciliation",
		"ASE-416",
		"Stall by codex event timestamp",
		"codex-stall-01",
		"Block without producing new codex events.",
		manager,
	)
	t.Cleanup(func() {
		select {
		case <-manager.releaseTurn:
		default:
			close(manager.releaseTurn)
		}
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	if _, err := client.Workflow.UpdateOneID(workflowItem.ID).
		SetStallTimeoutMinutes(1).
		Save(ctx); err != nil {
		t.Fatalf("update workflow stall timeout: %v", err)
	}

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("launch runtime session: %v", err)
	}
	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("start ready execution: %v", err)
	}

	waitForRuntimeCondition(t, 5*time.Second, func() bool {
		runSnapshot, err := client.AgentRun.Get(ctx, runItem.ID)
		return err == nil && runSnapshot.Status == entagentrun.StatusExecuting
	})

	launcher.now = func() time.Time {
		return now.Add(2 * time.Minute)
	}

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("reconcile stalled runtime: %v", err)
	}

	waitForRuntimeCondition(t, 5*time.Second, func() bool {
		ticketSnapshot, err := client.Ticket.Get(ctx, ticketItem.ID)
		if err != nil {
			return false
		}
		runSnapshot, err := client.AgentRun.Get(ctx, runItem.ID)
		if err != nil {
			return false
		}
		return ticketSnapshot.CurrentRunID == nil &&
			ticketSnapshot.NextRetryAt != nil &&
			(runSnapshot.Status == entagentrun.StatusErrored ||
				runSnapshot.Status == entagentrun.StatusTerminated)
	})

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.StallCount != 1 {
		t.Fatalf("expected stall count 1 after runtime reconciliation, got %+v", ticketAfter)
	}
}

func TestRuntimeLauncherRunTickMarksRetryOnTurnFailure(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 23, 11, 0, 0, 0, time.UTC)

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

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	if err := os.WriteFile(harnessPath, []byte(`---
workflow:
  role: coding
---

Handle a failing runtime turn.
`), 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-411").
		SetTitle("Retry failed turn").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	originalRetryToken := ticketItem.RetryToken

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-runner-fail-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	manager := &runtimeFakeProcessManager{
		turnInputDelta:  5,
		turnOutputDelta: 2,
		failTurn:        1,
	}
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, manager, nil, workflowSvc)
	launcher.now = func() time.Time {
		return now
	}
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("launch runtime session: %v", err)
	}
	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("start ready execution: %v", err)
	}

	waitForRuntimeCondition(t, 5*time.Second, func() bool {
		ticketSnapshot, err := client.Ticket.Get(ctx, ticketItem.ID)
		if err != nil {
			return false
		}
		runSnapshot, err := client.AgentRun.Get(ctx, runItem.ID)
		if err != nil {
			return false
		}
		return ticketSnapshot.CurrentRunID == nil &&
			ticketSnapshot.NextRetryAt != nil &&
			ticketSnapshot.AttemptCount == 1 &&
			ticketSnapshot.ConsecutiveErrors == 1 &&
			runSnapshot.Status == entagentrun.StatusErrored
	})

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.AttemptCount != 1 || ticketAfter.ConsecutiveErrors != 1 {
		t.Fatalf("expected retry counters to increment once, got %+v", ticketAfter)
	}
	if ticketAfter.NextRetryAt == nil || !ticketAfter.NextRetryAt.UTC().Equal(now.Add(10*time.Second)) {
		t.Fatalf("expected next retry at %s, got %+v", now.Add(10*time.Second), ticketAfter.NextRetryAt)
	}
	if ticketAfter.RetryToken == "" || ticketAfter.RetryToken == originalRetryToken {
		t.Fatalf("expected failed turn retry to rotate retry token, got %q", ticketAfter.RetryToken)
	}

	if ticketAfter.CurrentRunID != nil {
		t.Fatalf("expected retry path to clear current run, got %+v", ticketAfter.CurrentRunID)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusErrored {
		t.Fatalf("expected errored run after failed turn retry, got %+v", runAfter)
	}
	if runAfter.LastError == "" {
		t.Fatalf("expected retry release to preserve run error, got %+v", runAfter)
	}
	if manager.capturedTurnCount() != 1 {
		t.Fatalf("expected one failed turn, got %d", manager.capturedTurnCount())
	}
}

func TestRuntimeLauncherRunsTicketHooksAcrossSuccessfulLifecycle(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)
	identifier := "ASE-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetHooks(map[string]any{
			"ticket_hooks": map[string]any{
				"on_claim": []any{
					map[string]any{"cmd": `printf 'claim\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
				"on_start": []any{
					map[string]any{"cmd": `printf 'start\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
				"on_complete": []any{
					map[string]any{"cmd": `printf 'complete\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
				"on_done": []any{
					map[string]any{"cmd": `printf 'done\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
			},
		}).
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	if err := os.WriteFile(harnessPath, []byte(`---
workflow:
  role: coding
---

Exercise successful ticket hook lifecycle.
`), 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier(identifier).
		SetTitle("Run success hooks").
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
		SetName("codex-hooks-success-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	manager := &runtimeFakeProcessManager{}
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, manager, nil, workflowSvc)
	launcher.now = func() time.Time { return now }
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	assignment, err := launcher.loadAssignmentByRun(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("load assignment: %v", err)
	}
	session, err := launcher.startRuntimeSession(ctx, assignment)
	if err != nil {
		t.Fatalf("start runtime session: %v", err)
	}
	launcher.storeSession(runItem.ID, session)

	repoWorkspace, err := client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.AgentRunIDEQ(runItem.ID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("load ticket repo workspace: %v", err)
	}
	logPath := filepath.Join(repoWorkspace.WorkspaceRoot, "hook.log")
	//nolint:gosec // Test controls the temporary workspace path under t.TempDir-backed fixtures.
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read hook log after start: %v", err)
	}
	if string(raw) != "claim\nstart\n" {
		t.Fatalf("unexpected start hook log %q", string(raw))
	}

	if err := launcher.tickets.RunLifecycleHook(ctx, ticketservice.RunLifecycleHookInput{
		TicketID: ticketItem.ID,
		RunID:    runItem.ID,
		HookName: infrahook.TicketHookOnComplete,
		Blocking: true,
	}); err != nil {
		t.Fatalf("run ticket on_complete hooks: %v", err)
	}

	resolvedTicket, err := client.Ticket.Query().
		Where(entticket.IDEQ(ticketItem.ID)).
		WithCurrentRun().
		WithWorkflow(func(query *ent.WorkflowQuery) {
			query.WithFinishStatuses()
		}).
		Only(ctx)
	if err != nil {
		t.Fatalf("reload resolved ticket: %v", err)
	}
	if err := launcher.finishResolvedExecution(ctx, runItem.ID, agentItem.ID, resolvedTicket); err != nil {
		t.Fatalf("finish resolved execution: %v", err)
	}

	ticketSnapshot, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload finished ticket: %v", err)
	}
	runSnapshot, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload finished run: %v", err)
	}
	if ticketSnapshot.CurrentRunID != nil {
		t.Fatalf("expected current run to be cleared, got %+v", ticketSnapshot.CurrentRunID)
	}
	if runSnapshot.Status != entagentrun.StatusCompleted {
		t.Fatalf("expected completed run, got %+v", runSnapshot)
	}

	//nolint:gosec // Test controls the temporary workspace path under t.TempDir-backed fixtures.
	raw, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read hook log: %v", err)
	}
	if string(raw) != "claim\nstart\ncomplete\ndone\n" {
		t.Fatalf("unexpected success hook log %q", string(raw))
	}
}

func TestRuntimeLauncherRunsErrorHookWhenLaunchHookBlocks(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 31, 9, 30, 0, 0, time.UTC)
	identifier := "ASE-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetHooks(map[string]any{
			"ticket_hooks": map[string]any{
				"on_claim": []any{
					map[string]any{"cmd": `printf 'claim\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
				"on_start": []any{
					map[string]any{
						"cmd":        `printf 'start\n' >> "$OPENASE_WORKSPACE/hook.log"; exit 9`,
						"on_failure": "block",
					},
				},
				"on_error": []any{
					map[string]any{
						"cmd":        `printf 'error\n' >> "$OPENASE_WORKSPACE/hook.log"`,
						"on_failure": "ignore",
					},
				},
			},
		}).
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	if err := os.WriteFile(harnessPath, []byte(`---
workflow:
  role: coding
---

Exercise failing ticket hook lifecycle.
`), 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	commitRuntimeLauncherRepo(t, repoRoot)
	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier(identifier).
		SetTitle("Run failing hooks").
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
		SetName("codex-hooks-failure-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, nil, workflowSvc)
	launcher.now = func() time.Time { return now }
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("launch runtime session: %v", err)
	}

	waitForRuntimeCondition(t, 5*time.Second, func() bool {
		ticketSnapshot, err := client.Ticket.Get(ctx, ticketItem.ID)
		if err != nil {
			return false
		}
		runSnapshot, err := client.AgentRun.Get(ctx, runItem.ID)
		if err != nil {
			return false
		}
		return ticketSnapshot.CurrentRunID == nil &&
			ticketSnapshot.NextRetryAt != nil &&
			runSnapshot.Status == entagentrun.StatusErrored
	})

	repoWorkspace, err := client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.AgentRunIDEQ(runItem.ID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("load ticket repo workspace: %v", err)
	}
	logPath := filepath.Join(repoWorkspace.WorkspaceRoot, "hook.log")
	//nolint:gosec // Test controls the temporary workspace path under t.TempDir-backed fixtures.
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read hook log: %v", err)
	}
	if string(raw) != "claim\nstart\nerror\n" {
		t.Fatalf("unexpected failure hook log %q", string(raw))
	}
}

func TestRuntimeLauncherRunTickPreparesRemoteWorkspaceAndLaunchesOverSSH(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

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
		SetIdentifier("ASE-401").
		SetTitle("Launch Codex on remote machine").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
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
		SetWorkspaceDirname("backend").
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
	remoteMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(fixture.orgID),
			entmachine.NameEQ("gpu-01"),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load remote machine: %v", err)
	}
	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(remoteMachine.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider machine: %v", err)
	}
	if _, err := client.ProjectRepoMirror.Create().
		SetProjectRepoID(repoItem.ID).
		SetMachineID(remoteMachine.ID).
		SetLocalPath("/srv/openase/mirrors/backend").
		SetState(entprojectrepomirror.StateReady).
		SetHeadCommit("abc123remote").
		Save(ctx); err != nil {
		t.Fatalf("create ready remote mirror: %v", err)
	}

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

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

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusReady {
		t.Fatalf("expected ready run, got %+v", runAfter)
	}
	if runAfter.SessionID != "thread-runtime-1" {
		t.Fatalf("expected thread-runtime-1 session id, got %q", runAfter.SessionID)
	}
	if !strings.Contains(prepareSession.command, "git clone --branch 'main' --single-branch '/srv/openase/mirrors/backend' '/srv/openase/workspaces/better-and-better/openase/ASE-401/backend'") {
		t.Fatalf("expected remote workspace clone command, got %q", prepareSession.command)
	}
	if !strings.Contains(processSession.startedCommand, "cd '/srv/openase/workspaces/better-and-better/openase/ASE-401/backend'") {
		t.Fatalf("expected remote process to cd into the single scoped repo workspace, got %q", processSession.startedCommand)
	}
	if !strings.Contains(processSession.startedCommand, "'/usr/local/bin/codex'") {
		t.Fatalf("expected machine agent cli path in remote command, got %q", processSession.startedCommand)
	}
	repoWorkspace, err := client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.AgentRunIDEQ(runItem.ID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("load remote ticket repo workspace: %v", err)
	}
	if repoWorkspace.State != entticketrepoworkspace.StateReady || repoWorkspace.HeadCommit != "abc123remote" {
		t.Fatalf("unexpected remote ticket repo workspace %+v", repoWorkspace)
	}
}

func TestRuntimeLauncherRunTickSyncsStaleRemoteMirrorBeforePreparingWorkspace(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

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
		SetIdentifier("ASE-401A").
		SetTitle("Sync stale remote mirror before launch").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
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
		SetWorkspaceDirname("backend").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}
	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(repoItem.ID).
		SetBranchName("agent/codex-01/ASE-401A").
		SetPrStatus("none").
		SetCiStatus("pending").
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
	remoteMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(fixture.orgID),
			entmachine.NameEQ("gpu-01"),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load remote machine: %v", err)
	}
	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(remoteMachine.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider machine: %v", err)
	}
	if _, err := client.ProjectRepoMirror.Create().
		SetProjectRepoID(repoItem.ID).
		SetMachineID(remoteMachine.ID).
		SetLocalPath("/srv/openase/mirrors/backend").
		SetState(entprojectrepomirror.StateStale).
		SetHeadCommit("abc123stale").
		Save(ctx); err != nil {
		t.Fatalf("create stale remote mirror: %v", err)
	}

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	syncSession := &runtimeSSHCommandSession{output: []byte("def456fresh\n")}
	prepareSession := &runtimeSSHPrepareSession{}
	processSession := newRuntimeSSHProcessSession()
	sshPool := sshinfra.NewPool("/tmp/openase",
		sshinfra.WithDialer(&runtimeSSHDialer{client: &runtimeSSHClient{sessions: []sshinfra.Session{syncSession, prepareSession, processSession}}}),
		sshinfra.WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
	)

	mirrorService := projectrepomirrorsvc.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mirrorService.ConfigureSSHPool(sshPool)
	mirrorService.ConfigureGitHubCredentials(stubTokenResolver{
		projectID: fixture.projectID,
		resolved: githubauthdomain.ResolvedCredential{
			Scope: githubauthdomain.ScopeProject,
			Token: "runtime-launcher-test-token",
		},
	})

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, sshPool, nil)
	launcher.ConfigureMirrorService(mirrorService)
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run launcher tick: %v", err)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusReady {
		t.Fatalf("expected ready run, got %+v", runAfter)
	}
	if !strings.Contains(syncSession.command, "/srv/openase/mirrors/backend") || !strings.Contains(syncSession.command, "fetch origin") {
		t.Fatalf("expected remote mirror sync command, got %q", syncSession.command)
	}
	if !strings.Contains(prepareSession.command, "git clone --branch 'main' --single-branch '/srv/openase/mirrors/backend' '/srv/openase/workspaces/better-and-better/openase/ASE-401A/backend'") {
		t.Fatalf("expected remote workspace clone command, got %q", prepareSession.command)
	}

	mirrorAfter, err := client.ProjectRepoMirror.Query().
		Where(
			entprojectrepomirror.ProjectRepoIDEQ(repoItem.ID),
			entprojectrepomirror.MachineIDEQ(remoteMachine.ID),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("reload project repo mirror: %v", err)
	}
	if mirrorAfter.State != entprojectrepomirror.StateReady || mirrorAfter.HeadCommit != "def456fresh" || mirrorAfter.LastSyncedAt == nil {
		t.Fatalf("expected synced remote mirror, got %+v", mirrorAfter)
	}
}

func TestRuntimeLauncherRunTickFailsWhenRemoteCodexEnvironmentIsNotReady(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

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
		SetIdentifier("ASE-402").
		SetTitle("Launch Codex on remote machine without auth").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
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
	remoteMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(fixture.orgID),
			entmachine.NameEQ("gpu-02"),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load remote machine: %v", err)
	}
	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(remoteMachine.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider machine: %v", err)
	}

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-02").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, nil, nil)
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run launcher tick: %v", err)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusErrored {
		t.Fatalf("expected errored run, got %+v", runAfter)
	}
	if !strings.Contains(runAfter.LastError, "codex cli is not logged in") {
		t.Fatalf("expected codex auth failure in last error, got %q", runAfter.LastError)
	}
	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID != nil {
		t.Fatalf("expected launch failure to clear current run, got %+v", ticketAfter.CurrentRunID)
	}
	if ticketAfter.NextRetryAt == nil {
		t.Fatalf("expected launch failure to schedule retry, got %+v", ticketAfter)
	}
}

func TestRuntimeLauncherRunTickMarksTicketRepoWorkspaceFailedWhenRemoteSSHPoolIsMissing(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)

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
		SetIdentifier("ASE-404").
		SetTitle("Remote launch without ssh pool").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	projectRepo, err := client.ProjectRepo.Create().
		SetProjectID(fixture.projectID).
		SetName("openase").
		SetRepositoryURL("https://github.com/GrandCX/openase.git").
		SetDefaultBranch("main").
		SetWorkspaceDirname("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}

	sshUser := "openase"
	sshKeyPath := "keys/gpu-03.pem"
	workspaceRoot := "/srv/openase/workspaces"
	if _, err := client.Machine.Create().
		SetOrganizationID(fixture.orgID).
		SetName("gpu-03").
		SetHost("10.0.1.12").
		SetPort(22).
		SetSSHUser(sshUser).
		SetSSHKeyPath(sshKeyPath).
		SetWorkspaceRoot(workspaceRoot).
		SetStatus(entmachine.StatusOnline).
		Save(ctx); err != nil {
		t.Fatalf("create machine: %v", err)
	}
	remoteMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(fixture.orgID),
			entmachine.NameEQ("gpu-03"),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load remote machine: %v", err)
	}
	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(remoteMachine.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider machine: %v", err)
	}
	if _, err := client.ProjectRepoMirror.Create().
		SetProjectRepoID(projectRepo.ID).
		SetMachineID(remoteMachine.ID).
		SetLocalPath("/srv/openase/mirrors/openase").
		SetState(entprojectrepomirror.StateReady).
		SetHeadCommit("abc123remote").
		Save(ctx); err != nil {
		t.Fatalf("create ready remote mirror: %v", err)
	}

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-03").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, nil, nil)
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run launcher tick: %v", err)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusErrored {
		t.Fatalf("expected errored run, got %+v", runAfter)
	}

	repoWorkspace, err := client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.AgentRunIDEQ(runItem.ID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("load ticket repo workspace: %v", err)
	}
	if repoWorkspace.State != entticketrepoworkspace.StateFailed {
		t.Fatalf("expected failed repo workspace, got %+v", repoWorkspace)
	}
	if !strings.Contains(repoWorkspace.LastError, "ssh pool unavailable for remote machine gpu-03") {
		t.Fatalf("expected ssh pool failure in last_error, got %+v", repoWorkspace)
	}
}

func TestRuntimeLauncherRunTickSkipsMachineCodexPreflightForNonCodexCommand(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Fake app server").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

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
		SetWorkflowID(workflowItem.ID).
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
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, nil, nil)
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run launcher tick: %v", err)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusReady {
		t.Fatalf("expected ready run, got %+v", runAfter)
	}
	if runAfter.SessionID != "thread-runtime-1" {
		t.Fatalf("expected thread-runtime-1 session id, got %q", runAfter.SessionID)
	}
	if runAfter.LastError != "" {
		t.Fatalf("expected empty last error, got %q", runAfter.LastError)
	}
}

func TestRuntimeLauncherRunTickSkipsMachineCodexPreflightWhenAPIKeyIsConfigured(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("API key launch").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

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
		SetName("codex-api-key-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

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

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusReady {
		t.Fatalf("expected ready run, got %+v", runAfter)
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

func TestRuntimeLauncherRunTickTransitionsPauseRequestedAgentToPaused(t *testing.T) {
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
		SetName("Pause runtime").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-405").
		SetTitle("Pause Codex runtime").
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
		SetName("codex-pause-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create claimed agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	manager := &runtimeFakeProcessManager{}
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, manager, nil, nil)
	launcher.now = func() time.Time {
		return now
	}
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run initial launcher tick: %v", err)
	}
	waitForAgentLifecycleEvent(t, stream, agentReadyType)

	if _, err := client.Agent.UpdateOneID(agentItem.ID).
		SetRuntimeControlState(entagent.RuntimeControlStatePauseRequested).
		Save(ctx); err != nil {
		t.Fatalf("request pause: %v", err)
	}

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run pause launcher tick: %v", err)
	}

	pausedEvent := waitForAgentLifecycleEvent(t, stream, agentPausedType)
	payload := decodeLifecycleEnvelope(t, pausedEvent.Payload)
	if payload.Agent.ID != agentItem.ID.String() || payload.Agent.RuntimeControlState != "paused" {
		t.Fatalf("unexpected paused event payload: %+v", payload.Agent)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.RuntimeControlState != entagent.RuntimeControlStatePaused {
		t.Fatalf("expected paused control state, got %s", agentAfter.RuntimeControlState)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusTerminated {
		t.Fatalf("expected terminated run after pause, got %s", runAfter.Status)
	}
	if runAfter.SessionID != "" || runAfter.RuntimeStartedAt != nil || runAfter.LastHeartbeatAt != nil {
		t.Fatalf("expected runtime state to be cleared after pause, got %+v", runAfter)
	}
}

func newRuntimeExecutionFixture(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	fixture projectFixture,
	now time.Time,
	workflowName string,
	ticketIdentifier string,
	ticketTitle string,
	agentName string,
	harnessBody string,
	manager provider.AgentCLIProcessManager,
) (*ent.Workflow, *workflowservice.Service, *ent.Ticket, *ent.Agent, *ent.AgentRun, *RuntimeLauncher) {
	t.Helper()

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName(workflowName).
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	repoRoot := t.TempDir()
	initRuntimeLauncherRepo(t, repoRoot)
	createRuntimeLauncherPrimaryRepo(ctx, t, client, fixture.projectID, repoRoot)
	harnessPath := filepath.Join(repoRoot, ".openase", "harnesses", "coding.md")
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		t.Fatalf("create harness dir: %v", err)
	}
	harnessContent := []byte("---\nworkflow:\n  role: coding\n---\n\n" + harnessBody + "\n")
	if err := os.WriteFile(harnessPath, harnessContent, 0o600); err != nil {
		t.Fatalf("write harness file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".openase", "harnesses", "alternate.md"), harnessContent, 0o600); err != nil {
		t.Fatalf("write alternate harness file: %v", err)
	}
	commitRuntimeLauncherRepo(t, repoRoot)

	workflowSvc := newRuntimeLauncherWorkflowService(t, client, repoRoot, fixture.projectID)

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier(ticketIdentifier).
		SetTitle(ticketTitle).
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
		SetName(agentName).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, manager, nil, workflowSvc)
	launcher.now = func() time.Time {
		return now
	}
	if fakeManager, ok := manager.(*runtimeFakeProcessManager); ok && fakeManager.releaseTurn != nil {
		t.Cleanup(func() {
			select {
			case <-fakeManager.releaseTurn:
			default:
				close(fakeManager.releaseTurn)
			}
		})
	}

	return workflowItem, workflowSvc, ticketItem, agentItem, runItem, launcher
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

func waitForRuntimeCondition(t *testing.T, timeout time.Duration, predicate func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("timed out waiting for runtime condition")
}

func waitForRuntimeExecuting(ctx context.Context, t *testing.T, client *ent.Client, ticketID uuid.UUID, runID uuid.UUID) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		runSnapshot, err := client.AgentRun.Get(ctx, runID)
		if err == nil && runSnapshot.Status == entagentrun.StatusExecuting {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	runSnapshot, _ := client.AgentRun.Get(ctx, runID)
	ticketSnapshot, _ := client.Ticket.Get(ctx, ticketID)
	t.Fatalf("timed out waiting for executing run: run=%+v ticket=%+v", runSnapshot, ticketSnapshot)
}

func containsEnvironmentPrefix(environment []string, want string) bool {
	for _, item := range environment {
		if strings.HasPrefix(item, want) {
			return true
		}
	}

	return false
}

func createRuntimeLauncherPrimaryRepo(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	projectID uuid.UUID,
	repoRoot string,
) {
	t.Helper()

	repoName := "repo-" + strings.ReplaceAll(projectID.String(), "-", "")[:8]
	projectRepo, err := client.ProjectRepo.Create().
		SetProjectID(projectID).
		SetName(repoName).
		SetRepositoryURL(fmt.Sprintf("https://github.com/acme/%s.git", repoName)).
		SetDefaultBranch("main").
		SetWorkspaceDirname(repoName).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}

	project, err := client.Project.Get(ctx, projectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	localMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(project.OrganizationID),
			entmachine.NameEQ(catalogdomain.LocalMachineName),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load local machine: %v", err)
	}
	if _, err := client.ProjectRepoMirror.Create().
		SetProjectRepoID(projectRepo.ID).
		SetMachineID(localMachine.ID).
		SetLocalPath(repoRoot).
		SetState(entprojectrepomirror.StateReady).
		Save(ctx); err != nil {
		t.Fatalf("create ready project repo mirror: %v", err)
	}
}

func newRuntimeLauncherWorkflowService(
	t *testing.T,
	client *ent.Client,
	repoRoot string,
	projectID uuid.UUID,
) *workflowservice.Service {
	t.Helper()

	service, err := workflowservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}

	projectStorageRoot := filepath.Join(repoRoot, ".openase-projects", projectID.String())

	copyRuntimeLauncherDir(t, filepath.Join(repoRoot, ".openase"), filepath.Join(projectStorageRoot, ".openase"))
	defaultHarnessPath := filepath.Join(projectStorageRoot, ".openase", "harnesses", "coding.md")
	if _, statErr := os.Stat(defaultHarnessPath); os.IsNotExist(statErr) {
		if err := os.MkdirAll(filepath.Dir(defaultHarnessPath), 0o750); err != nil {
			t.Fatalf("create default harness dir: %v", err)
		}
		if err := os.WriteFile(defaultHarnessPath, []byte("---\nworkflow:\n  role: coding\n---\n\n# Coding\n"), 0o600); err != nil {
			t.Fatalf("write default harness: %v", err)
		}
	} else if statErr != nil {
		t.Fatalf("stat default harness: %v", statErr)
	}

	if _, statErr := os.Stat(filepath.Join(projectStorageRoot, ".git")); os.IsNotExist(statErr) {
		initRuntimeLauncherRepo(t, projectStorageRoot)
		commitRuntimeLauncherRepo(t, projectStorageRoot)
	} else if statErr != nil {
		t.Fatalf("stat project storage git dir: %v", statErr)
	} else {
		runRuntimeLauncherGit(t, projectStorageRoot, "add", ".")
		runRuntimeLauncherGit(t, projectStorageRoot, "commit", "--allow-empty", "-m", "Sync project storage")
	}

	t.Cleanup(func() {
		if err := service.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	return service
}

func copyRuntimeLauncherDir(t *testing.T, sourceRoot string, targetRoot string) {
	t.Helper()

	info, err := os.Stat(sourceRoot)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("stat source dir %s: %v", sourceRoot, err)
	}
	if !info.IsDir() {
		return
	}
	if err := filepath.Walk(sourceRoot, func(path string, entry os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetRoot, relativePath)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o750)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0o600)
	}); err != nil {
		t.Fatalf("copy %s to %s: %v", sourceRoot, targetRoot, err)
	}
}

func initRuntimeLauncherRepo(t *testing.T, repoRoot string) {
	t.Helper()

	runRuntimeLauncherGit(t, repoRoot, "init", "-b", "main")
	runRuntimeLauncherGit(t, repoRoot, "config", "user.name", "Codex")
	runRuntimeLauncherGit(t, repoRoot, "config", "user.email", "codex@openai.com")
}

func commitRuntimeLauncherRepo(t *testing.T, repoRoot string) {
	t.Helper()

	runRuntimeLauncherGit(t, repoRoot, "add", ".")
	runRuntimeLauncherGit(t, repoRoot, "commit", "-m", "Seed harness")
}

func runRuntimeLauncherGit(t *testing.T, repoRoot string, args ...string) {
	t.Helper()

	//nolint:gosec // Test helper intentionally invokes local git with controlled arguments to seed repos.
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run git %v in %s: %v\n%s", args, repoRoot, err, string(output))
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
	capturedTurns      []runtimeTurnStartParams
	turnCount          int
	turnInputDelta     int64
	turnOutputDelta    int64
	agentMessageDelta  string
	commandOutputDelta string
	agentMessageFinal  string
	commandOutputFinal string
	failTurn           int
	releaseTurn        chan struct{}
	executionDone      chan struct{}
	closeTurnSession   bool
	sessionExitErr     error
}

type runtimeBlockingEventProvider struct {
	blockTypes map[provider.EventType]struct{}
	release    chan struct{}
}

func newRuntimeBlockingEventProvider(blockTypes ...provider.EventType) *runtimeBlockingEventProvider {
	set := make(map[provider.EventType]struct{}, len(blockTypes))
	for _, eventType := range blockTypes {
		set[eventType] = struct{}{}
	}
	return &runtimeBlockingEventProvider{
		blockTypes: set,
		release:    make(chan struct{}),
	}
}

func (p *runtimeBlockingEventProvider) Publish(ctx context.Context, event provider.Event) error {
	if p != nil {
		if _, ok := p.blockTypes[event.Type]; ok {
			select {
			case <-p.release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func (p *runtimeBlockingEventProvider) Subscribe(context.Context, ...provider.Topic) (<-chan provider.Event, error) {
	stream := make(chan provider.Event)
	close(stream)
	return stream, nil
}

func (p *runtimeBlockingEventProvider) Close() error {
	p.Release()
	return nil
}

func (p *runtimeBlockingEventProvider) Release() {
	if p == nil {
		return
	}
	select {
	case <-p.release:
	default:
		close(p.release)
	}
}

type runtimeSequencedProcessManager struct {
	delegate      *runtimeFakeProcessManager
	firstRelease  chan struct{}
	secondStarted chan struct{}
	mu            sync.Mutex
	starts        int
}

func newRuntimeSequencedProcessManager(delegate *runtimeFakeProcessManager) *runtimeSequencedProcessManager {
	return &runtimeSequencedProcessManager{
		delegate:      delegate,
		firstRelease:  make(chan struct{}),
		secondStarted: make(chan struct{}),
	}
}

func (m *runtimeSequencedProcessManager) Start(ctx context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	m.mu.Lock()
	m.starts++
	callNumber := m.starts
	m.mu.Unlock()

	switch callNumber {
	case 1:
		select {
		case <-m.firstRelease:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case 2:
		select {
		case <-m.secondStarted:
		default:
			close(m.secondStarted)
		}
	}

	return m.delegate.Start(ctx, spec)
}

func (m *runtimeSequencedProcessManager) ReleaseFirst() {
	if m == nil {
		return
	}
	select {
	case <-m.firstRelease:
	default:
		close(m.firstRelease)
	}
}

func (m *runtimeSequencedProcessManager) SecondStarted() <-chan struct{} {
	if m == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return m.secondStarted
}

type runtimeBlockingStartProcessManager struct {
	runtimeFakeProcessManager
	startMu           sync.Mutex
	startCount        int
	releaseFirstStart chan struct{}
	firstStartSeen    chan struct{}
	laterStartSeen    chan struct{}
	releaseOnce       sync.Once
}

func (m *runtimeBlockingStartProcessManager) Start(ctx context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	m.startMu.Lock()
	m.startCount++
	startCount := m.startCount
	m.startMu.Unlock()

	if startCount == 1 {
		signalRuntimeStart(m.firstStartSeen)
		<-m.releaseFirstStart
	} else {
		signalRuntimeStart(m.laterStartSeen)
	}

	return m.runtimeFakeProcessManager.Start(ctx, spec)
}

func (m *runtimeBlockingStartProcessManager) release() {
	m.releaseOnce.Do(func() {
		close(m.releaseFirstStart)
	})
}

func signalRuntimeStart(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
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

func (m *runtimeFakeProcessManager) appendTurn(params runtimeTurnStartParams) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.capturedTurns = append(m.capturedTurns, params)
	m.turnCount++
	return m.turnCount
}

func (m *runtimeFakeProcessManager) capturedTurnCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.turnCount
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

func (p *runtimeFakeProcess) finish(err error) {
	_ = p.stdinWrite.Close()
	_ = p.stdoutWrite.Close()
	_ = p.stderrWrite.Close()
	select {
	case p.done <- err:
	default:
	}
}

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

	if manager != nil && (manager.turnInputDelta > 0 ||
		manager.turnOutputDelta > 0 ||
		strings.TrimSpace(manager.agentMessageDelta) != "" ||
		strings.TrimSpace(manager.commandOutputDelta) != "" ||
		strings.TrimSpace(manager.agentMessageFinal) != "" ||
		strings.TrimSpace(manager.commandOutputFinal) != "" ||
		manager.failTurn > 0 ||
		manager.releaseTurn != nil ||
		manager.executionDone != nil ||
		manager.closeTurnSession) {
		for {
			turnStart, err := readRuntimeMessage(decoder)
			if err != nil {
				select {
				case <-process.stopCh:
					return nil
				default:
					return err
				}
			}
			if turnStart.Method != "turn/start" {
				return fmt.Errorf("expected turn/start, got %s", turnStart.Method)
			}

			var turnParams runtimeTurnStartParams
			if err := json.Unmarshal(turnStart.Params, &turnParams); err != nil {
				return fmt.Errorf("decode turn/start params: %w", err)
			}
			turnNumber := manager.appendTurn(turnParams)
			turnID := fmt.Sprintf("turn-runtime-%d", turnNumber)

			if err := encoder.Encode(runtimeJSONRPCMessage{
				JSONRPC: "2.0",
				ID:      turnStart.ID,
				Result: mustMarshalRuntimeJSON(map[string]any{
					"turn": map[string]any{"id": turnID},
				}),
			}); err != nil {
				return err
			}

			if manager.turnInputDelta > 0 || manager.turnOutputDelta > 0 {
				if err := encoder.Encode(runtimeJSONRPCMessage{
					JSONRPC: "2.0",
					Method:  "thread/tokenUsage/updated",
					Params: mustMarshalRuntimeJSON(map[string]any{
						"threadId": "thread-runtime-1",
						"turnId":   turnID,
						"tokenUsage": map[string]any{
							"total": map[string]any{
								"inputTokens":  int64(turnNumber) * manager.turnInputDelta,
								"outputTokens": int64(turnNumber) * manager.turnOutputDelta,
								"totalTokens":  int64(turnNumber) * (manager.turnInputDelta + manager.turnOutputDelta),
							},
							"last": map[string]any{
								"inputTokens":  manager.turnInputDelta,
								"outputTokens": manager.turnOutputDelta,
								"totalTokens":  manager.turnInputDelta + manager.turnOutputDelta,
							},
						},
					}),
				}); err != nil {
					return err
				}
			}

			if strings.TrimSpace(manager.agentMessageDelta) != "" {
				if err := encoder.Encode(runtimeJSONRPCMessage{
					JSONRPC: "2.0",
					Method:  "item/agentMessage/delta",
					Params: mustMarshalRuntimeJSON(map[string]any{
						"threadId": "thread-runtime-1",
						"turnId":   turnID,
						"itemId":   fmt.Sprintf("agent-message-%d", turnNumber),
						"delta":    manager.agentMessageDelta,
					}),
				}); err != nil {
					return err
				}
			}
			if strings.TrimSpace(manager.commandOutputDelta) != "" {
				if err := encoder.Encode(runtimeJSONRPCMessage{
					JSONRPC: "2.0",
					Method:  "item/commandExecution/outputDelta",
					Params: mustMarshalRuntimeJSON(map[string]any{
						"threadId": "thread-runtime-1",
						"turnId":   turnID,
						"itemId":   fmt.Sprintf("command-output-%d", turnNumber),
						"delta":    manager.commandOutputDelta,
					}),
				}); err != nil {
					return err
				}
			}
			if strings.TrimSpace(manager.agentMessageFinal) != "" {
				if err := encoder.Encode(runtimeJSONRPCMessage{
					JSONRPC: "2.0",
					Method:  "item/completed",
					Params: mustMarshalRuntimeJSON(map[string]any{
						"threadId": "thread-runtime-1",
						"turnId":   turnID,
						"item": map[string]any{
							"id":    fmt.Sprintf("agent-message-%d", turnNumber),
							"type":  "agentMessage",
							"text":  manager.agentMessageFinal,
							"phase": "commentary",
						},
					}),
				}); err != nil {
					return err
				}
			}
			if strings.TrimSpace(manager.commandOutputFinal) != "" {
				if err := encoder.Encode(runtimeJSONRPCMessage{
					JSONRPC: "2.0",
					Method:  "item/completed",
					Params: mustMarshalRuntimeJSON(map[string]any{
						"threadId": "thread-runtime-1",
						"turnId":   turnID,
						"item": map[string]any{
							"id":               fmt.Sprintf("command-output-%d", turnNumber),
							"type":             "commandExecution",
							"aggregatedOutput": manager.commandOutputFinal,
						},
					}),
				}); err != nil {
					return err
				}
			}
			if manager.releaseTurn != nil {
				<-manager.releaseTurn
			}
			if manager.closeTurnSession {
				process.finish(manager.sessionExitErr)
				return nil
			}

			if manager.failTurn > 0 && turnNumber == manager.failTurn {
				if err := encoder.Encode(runtimeJSONRPCMessage{
					JSONRPC: "2.0",
					Method:  "error",
					Params: mustMarshalRuntimeJSON(map[string]any{
						"threadId": "thread-runtime-1",
						"turnId":   turnID,
						"error": map[string]any{
							"message": "simulated turn failure",
						},
					}),
				}); err != nil {
					return err
				}
			} else {
				if err := encoder.Encode(runtimeJSONRPCMessage{
					JSONRPC: "2.0",
					Method:  "turn/completed",
					Params: mustMarshalRuntimeJSON(map[string]any{
						"threadId": "thread-runtime-1",
						"turn": map[string]any{
							"id":     turnID,
							"status": "completed",
						},
					}),
				}); err != nil {
					return err
				}
			}

			if manager.executionDone != nil && turnNumber >= defaultRuntimeMaxTurns {
				close(manager.executionDone)
				<-process.stopCh
				return nil
			}
		}
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

type runtimeTurnStartParams struct {
	ThreadID string `json:"threadId,omitempty"`
	CWD      string `json:"cwd,omitempty"`
	Title    string `json:"title,omitempty"`
	Input    []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"input,omitempty"`
}

func readRuntimeMessage(decoder *json.Decoder) (runtimeJSONRPCMessage, error) {
	var message runtimeJSONRPCMessage
	if err := decoder.Decode(&message); err != nil {
		return runtimeJSONRPCMessage{}, err
	}
	return message, nil
}

func openRuntimeSSERequest(t *testing.T, url string) (*http.Response, context.CancelFunc) {
	t.Helper()

	requestCtx, cancel := context.WithCancel(context.Background())
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, url, nil)
	if err != nil {
		cancel()
		t.Fatalf("new SSE request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		cancel()
		t.Fatalf("open SSE request: %v", err)
	}

	return response, cancel
}

func readRuntimeSSEBody(t *testing.T, response *http.Response, cancel context.CancelFunc) string {
	t.Helper()

	bodyCh := make(chan string, 1)
	go func() {
		bytes, _ := io.ReadAll(response.Body)
		bodyCh <- string(bytes)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case body := <-bodyCh:
		return body
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE response body")
		return ""
	}
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

type runtimeSSHCommandSession struct {
	command string
	output  []byte
	err     error
}

func (s *runtimeSSHCommandSession) CombinedOutput(cmd string) ([]byte, error) {
	s.command = cmd
	return s.output, s.err
}

func (s *runtimeSSHCommandSession) StdinPipe() (io.WriteCloser, error) {
	return nil, fmt.Errorf("not supported")
}

func (s *runtimeSSHCommandSession) StdoutPipe() (io.Reader, error) { return strings.NewReader(""), nil }

func (s *runtimeSSHCommandSession) StderrPipe() (io.Reader, error) { return strings.NewReader(""), nil }

func (s *runtimeSSHCommandSession) Start(string) error { return fmt.Errorf("not supported") }

func (s *runtimeSSHCommandSession) Signal(string) error { return nil }

func (s *runtimeSSHCommandSession) Wait() error { return nil }

func (s *runtimeSSHCommandSession) Close() error { return nil }

type runtimeSSHProcessSession struct {
	stdinRead  *io.PipeReader
	stdinWrite *io.PipeWriter

	stdoutRead  *io.PipeReader
	stdoutWrite *io.PipeWriter

	stderrRead  *io.PipeReader
	stderrWrite *io.PipeWriter

	done    chan error
	closeCh chan struct{}
	once    sync.Once

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
		closeCh:     make(chan struct{}),
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
		if err := runRuntimeSSHHandshake(s); err != nil {
			s.done <- err
			return
		}
		<-s.closeCh
		s.done <- nil
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
	s.once.Do(func() {
		close(s.closeCh)
		_ = s.stdinRead.Close()
		_ = s.stdinWrite.Close()
		_ = s.stdoutRead.Close()
		_ = s.stdoutWrite.Close()
		_ = s.stderrRead.Close()
		_ = s.stderrWrite.Close()
	})
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
