package orchestrator

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestOrchestratorUtilityCoverage(t *testing.T) {
	agentID := uuid.New()
	ticketID := uuid.New()
	runID := uuid.New()
	startedAt := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	heartbeatAt := startedAt.Add(2 * time.Minute)

	agentItem := &ent.Agent{
		ID:                  agentID,
		RuntimeControlState: entagent.RuntimeControlStateActive,
	}
	runItem := &ent.AgentRun{
		ID:               runID,
		TicketID:         ticketID,
		Status:           entagentrun.StatusExecuting,
		SessionID:        "sess-1",
		LastError:        "boom",
		RuntimeStartedAt: &startedAt,
		LastHeartbeatAt:  &heartbeatAt,
	}

	metadata := runtimeEventMetadataForState(agentLifecycleState{
		agent:        agentItem,
		run:          runItem,
		runIsCurrent: true,
	})
	if metadata["status"] != "running" || metadata["runtime_phase"] != "executing" || metadata["session_id"] != "sess-1" {
		t.Fatalf("runtimeEventMetadataForState() = %+v", metadata)
	}
	if metadata["run_id"] != runID.String() || metadata["current_run_id"] != runID.String() || metadata["ticket_id"] != ticketID.String() {
		t.Fatalf("runtimeEventMetadataForState() missing run identifiers: %+v", metadata)
	}

	if got := runtimeEventMetadata(agentItem); got["status"] != "idle" || got["runtime_phase"] != "none" {
		t.Fatalf("runtimeEventMetadata() = %+v", got)
	}

	pausedState := agentLifecycleState{
		agent: &ent.Agent{
			ID:                  agentID,
			RuntimeControlState: entagent.RuntimeControlStatePaused,
		},
		run: &ent.AgentRun{Status: entagentrun.StatusLaunching},
	}
	if status := lifecycleAgentStatus(pausedState); status != "paused" {
		t.Fatalf("lifecycleAgentStatus(paused) = %q", status)
	}
	if phase := lifecycleAgentRuntimePhase(agentLifecycleState{agent: agentItem, run: &ent.AgentRun{Status: entagentrun.StatusErrored}}); phase != "failed" {
		t.Fatalf("lifecycleAgentRuntimePhase(errored) = %q", phase)
	}
	if phase := lifecycleAgentRuntimePhase(agentLifecycleState{agent: agentItem, run: &ent.AgentRun{Status: entagentrun.StatusTerminated}}); phase != "none" {
		t.Fatalf("lifecycleAgentRuntimePhase(terminated) = %q", phase)
	}

	if !machineHasAllLabels([]string{"gpu", "linux", "a100"}, []string{"gpu", "a100"}) {
		t.Fatal("machineHasAllLabels() expected true")
	}
	if machineHasAllLabels([]string{"gpu"}, []string{"gpu", "a100"}) {
		t.Fatal("machineHasAllLabels() expected false")
	}

	tickets := []*ent.Ticket{
		{Identifier: "ASE-3", Priority: entticket.PriorityHigh, CreatedAt: time.Date(2026, 3, 27, 12, 3, 0, 0, time.UTC)},
		{Identifier: "ASE-1", Priority: entticket.PriorityUrgent, CreatedAt: time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC)},
		{Identifier: "ASE-2", Priority: entticket.PriorityHigh, CreatedAt: time.Date(2026, 3, 27, 12, 1, 0, 0, time.UTC)},
		{Identifier: "ASE-4", Priority: entticket.PriorityLow, CreatedAt: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
		{Identifier: "ASE-5", Priority: "custom", CreatedAt: time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)},
	}
	sortTicketsByPriorityAndAge(tickets)
	if tickets[0].Identifier != "ASE-1" || tickets[1].Identifier != "ASE-2" || tickets[2].Identifier != "ASE-3" || tickets[3].Identifier != "ASE-4" || tickets[4].Identifier != "ASE-5" {
		t.Fatalf("sortTicketsByPriorityAndAge() = %+v", tickets)
	}

	if priorityRank(entticket.PriorityUrgent) != 0 || priorityRank(entticket.PriorityHigh) != 1 || priorityRank(entticket.PriorityMedium) != 2 || priorityRank(entticket.PriorityLow) != 3 || priorityRank("custom") != 4 {
		t.Fatalf("priorityRank() produced unexpected ordering")
	}
}

func TestOrchestratorHelperCoverage(t *testing.T) {
	now := time.Date(2026, 3, 27, 12, 30, 0, 0, time.FixedZone("UTC+1", 60*60))
	runID := uuid.New()
	machineID := uuid.New()

	if !machineHasLowDisk(map[string]any{"disk_available_gb": 4}) {
		t.Fatal("machineHasLowDisk(int) expected true")
	}
	if !machineHasLowDisk(map[string]any{"disk_available_gb": float32(4.5)}) {
		t.Fatal("machineHasLowDisk(float32) expected true")
	}
	if machineHasLowDisk(map[string]any{"disk_available_gb": "9"}) || machineHasLowDisk(map[string]any{}) {
		t.Fatal("machineHasLowDisk() expected false for unsupported or missing values")
	}

	resources := map[string]any{}
	level := ensureMonitorLevel(resources, "l2")
	level["ok"] = true
	if nested, ok := nestedMap(resources, "monitor"); !ok || nested["l2"] == nil {
		t.Fatalf("ensureMonitorLevel()/nestedMap() = %+v", resources)
	}
	if _, ok := nestedMap(map[string]any{"monitor": "bad"}, "monitor"); ok {
		t.Fatal("nestedMap() expected false for non-map values")
	}

	original := map[string]any{
		"nested": map[string]any{"count": int64(3)},
		"list":   []any{map[string]any{"enabled": true}, []any{"x"}},
		"maps":   []map[string]any{{"name": "builder"}},
	}
	cloned := cloneResourceMap(original)
	cloned["nested"].(map[string]any)["count"] = 99
	cloned["list"].([]any)[0].(map[string]any)["enabled"] = false
	cloned["maps"].([]map[string]any)[0]["name"] = "worker"
	if original["nested"].(map[string]any)["count"] != int64(3) || original["list"].([]any)[0].(map[string]any)["enabled"] != true || original["maps"].([]map[string]any)[0]["name"] != "builder" {
		t.Fatalf("cloneResourceMap()/cloneAnyValue() mutated source: %+v", original)
	}

	if compareAnyInt(int64(4), float64(2)) <= 0 || compareAnyInt("bad", 0) != 0 {
		t.Fatal("compareAnyInt() produced unexpected ordering")
	}
	if anyToInt(float64(7.8)) != 7 || anyToInt("bad") != 0 {
		t.Fatal("anyToInt() produced unexpected conversion")
	}
	if !anyToBool(" TRUE ") || anyToBool("nope") {
		t.Fatal("anyToBool() produced unexpected conversion")
	}

	if formatted := timePointerToRFC3339(&now); formatted == nil || *formatted != now.UTC().Format(time.RFC3339) {
		t.Fatalf("timePointerToRFC3339() = %+v", formatted)
	}
	if timePointerToRFC3339(nil) != nil {
		t.Fatal("timePointerToRFC3339(nil) expected nil")
	}
	if formatted := uuidPointerToString(&runID); formatted == nil || *formatted != runID.String() {
		t.Fatalf("uuidPointerToString() = %+v", formatted)
	}
	if uuidPointerToString(nil) != nil {
		t.Fatal("uuidPointerToString(nil) expected nil")
	}

	clonedLifecycle := cloneLifecycleMetadata(map[string]any{"status": "running"})
	clonedLifecycle["status"] = "paused"
	if empty := cloneLifecycleMetadata(nil); len(empty) != 0 {
		t.Fatalf("cloneLifecycleMetadata(nil) = %+v", empty)
	}
	if clonedLifecycle["status"] != "paused" {
		t.Fatalf("cloneLifecycleMetadata() = %+v", clonedLifecycle)
	}

	if msg := lifecycleMessage(agentClaimedType, "codex"); msg == "" {
		t.Fatal("lifecycleMessage(claimed) expected content")
	}
	if msg := lifecycleMessage(agentFailedType, "codex"); msg == "" {
		t.Fatal("lifecycleMessage(failed) expected content")
	}

	agentState := agentLifecycleState{
		agent: &ent.Agent{
			ID:                  uuid.New(),
			RuntimeControlState: entagent.RuntimeControlStateActive,
		},
		run: &ent.AgentRun{
			ID:     runID,
			Status: entagentrun.StatusExecuting,
		},
	}
	if metadata := schedulerRuntimeEventMetadata(agentState, nil); metadata["target_machine_id"] != nil {
		t.Fatalf("schedulerRuntimeEventMetadata(nil machine) = %+v", metadata)
	}
	machine := &ent.Machine{ID: machineID, Name: "builder"}
	metadata := schedulerRuntimeEventMetadata(agentState, machine)
	if metadata["target_machine_id"] != machineID.String() || metadata["target_machine_name"] != "builder" {
		t.Fatalf("schedulerRuntimeEventMetadata(machine) = %+v", metadata)
	}

	localRoot, err := workspaceinfra.LocalWorkspaceRoot()
	if err != nil {
		t.Fatalf("LocalWorkspaceRoot() error = %v", err)
	}
	if got := workspaceRoot(catalogdomain.Machine{Host: catalogdomain.LocalMachineHost}, "/srv/openase/workspaces/org/project/repo"); got != localRoot {
		t.Fatalf("workspaceRoot(local) = %q, want %q", got, localRoot)
	}
	remoteRoot := "/srv/remote/workspaces"
	if got := workspaceRoot(catalogdomain.Machine{Host: "10.0.0.10", WorkspaceRoot: &remoteRoot}, "/ignored/path"); got != remoteRoot {
		t.Fatalf("workspaceRoot(remote explicit) = %q, want %q", got, remoteRoot)
	}
	if got := workspaceRoot(catalogdomain.Machine{Host: "10.0.0.11"}, "/srv/openase/workspaces/org/project/repo"); got != "/srv/openase/workspaces" {
		t.Fatalf("workspaceRoot(derived) = %q", got)
	}
}

func TestOrchestratorGuardAndConstructorCoverage(t *testing.T) {
	ctx := context.Background()

	scheduler := NewScheduler(nil, nil, nil)
	if scheduler == nil || scheduler.logger == nil || scheduler.now == nil || scheduler.scheduledJobs == nil {
		t.Fatalf("NewScheduler() = %+v", scheduler)
	}
	if report, err := scheduler.RunTick(ctx); err == nil || err.Error() != "scheduler unavailable" || len(report.TicketsSkipped) != 0 {
		t.Fatalf("Scheduler.RunTick(unavailable) = %+v, %v", report, err)
	}

	retryService := NewRetryService(nil, nil)
	if retryService == nil || retryService.logger == nil || retryService.now == nil {
		t.Fatalf("NewRetryService() = %+v", retryService)
	}
	if _, err := retryService.MarkAttemptFailed(ctx, uuid.New()); err == nil || err.Error() != "retry service unavailable" {
		t.Fatalf("RetryService.MarkAttemptFailed(unavailable) error = %v", err)
	}
	if err := releaseCurrentRunClaim(ctx, nil, nil); err != nil {
		t.Fatalf("releaseCurrentRunClaim(nil) error = %v", err)
	}
	if err := releaseCurrentRunClaim(ctx, nil, &ent.Ticket{CurrentRunID: nil}); err != nil {
		t.Fatalf("releaseCurrentRunClaim(no current run) error = %v", err)
	}

	launcher := NewRuntimeLauncher(nil, nil, nil, nil, nil, nil)
	if launcher == nil || launcher.logger == nil || launcher.now == nil || launcher.sessions == nil || launcher.executions == nil || launcher.tickets == nil {
		t.Fatalf("NewRuntimeLauncher() = %+v", launcher)
	}
	if err := launcher.RunTick(ctx); err == nil || err.Error() != "runtime launcher unavailable" {
		t.Fatalf("RuntimeLauncher.RunTick(unavailable) error = %v", err)
	}
	if err := launcher.Close(ctx); err != nil {
		t.Fatalf("RuntimeLauncher.Close(empty) error = %v", err)
	}

	launcherNoProcessManager := &RuntimeLauncher{
		client: &ent.Client{},
		logger: launcher.logger,
		now:    time.Now,
	}
	if err := launcherNoProcessManager.RunTick(ctx); err == nil || err.Error() != "runtime launcher process manager unavailable" {
		t.Fatalf("RuntimeLauncher.RunTick(no process manager) error = %v", err)
	}

	monitor := NewMachineMonitor(nil, nil, nil)
	if monitor == nil || monitor.logger == nil || monitor.now == nil {
		t.Fatalf("NewMachineMonitor() = %+v", monitor)
	}
	if report, err := monitor.RunTick(ctx); err == nil || err.Error() != "machine monitor unavailable" {
		t.Fatalf("MachineMonitor.RunTick(unavailable) = %+v, %v", report, err)
	}
	monitor.client = &ent.Client{}
	if report, err := monitor.RunTick(ctx); err == nil || err.Error() != "machine monitor collector unavailable" {
		t.Fatalf("MachineMonitor.RunTick(no collector) = %+v, %v", report, err)
	}

	syncer := NewConnectorSyncer(nil, nil, nil, nil)
	if syncer == nil || syncer.logger == nil || syncer.now == nil {
		t.Fatalf("NewConnectorSyncer() = %+v", syncer)
	}
	if _, err := syncer.SyncAll(ctx); err == nil || err.Error() != "connector syncer unavailable" {
		t.Fatalf("ConnectorSyncer.SyncAll(unavailable) error = %v", err)
	}
	if _, err := syncer.SyncConnector(ctx, uuid.New()); err == nil || err.Error() != "connector syncer unavailable" {
		t.Fatalf("ConnectorSyncer.SyncConnector(unavailable) error = %v", err)
	}
	if _, err := syncer.HandleWebhook(ctx, uuid.New(), nil, nil); err == nil || err.Error() != "connector syncer unavailable" {
		t.Fatalf("ConnectorSyncer.HandleWebhook(unavailable) error = %v", err)
	}
	if err := syncer.SyncBack(ctx, SyncBackRequest{ConnectorID: uuid.New()}); err == nil || err.Error() != "connector syncer unavailable" {
		t.Fatalf("ConnectorSyncer.SyncBack(unavailable) error = %v", err)
	}

	if err := (*RuntimeLauncher)(nil).Close(ctx); err != nil {
		t.Fatalf("RuntimeLauncher.Close(nil) error = %v", err)
	}
	if err := (*RuntimeLauncher)(nil).startReadyExecutions(ctx); err != nil {
		t.Fatalf("RuntimeLauncher.startReadyExecutions(nil) error = %v", err)
	}
	launcher.runReadyExecution(ctx, uuid.New())
}

func TestRuntimeLauncherWorkspaceAndCommandHelpers(t *testing.T) {
	orgID := uuid.New()
	projectID := uuid.New()
	repoID := uuid.New()
	ticketID := uuid.New()

	project := &ent.Project{
		ID:   projectID,
		Slug: "payments",
		Edges: ent.ProjectEdges{
			Organization: &ent.Organization{ID: orgID, Slug: "acme", Name: "Acme"},
		},
	}
	launchContext := runtimeLaunchContext{
		agent: &ent.Agent{
			ID:   uuid.New(),
			Name: "codex-01",
		},
		project: project,
		ticket: &ent.Ticket{
			ID:         ticketID,
			Identifier: "ASE-77",
		},
		projectRepos: []*ent.ProjectRepo{{
			ID:            repoID,
			Name:          "backend",
			RepositoryURL: "https://github.com/acme/backend.git",
			DefaultBranch: "main",
			ClonePath:     "services/backend",
		}},
		ticketScopes: []*ent.TicketRepoScope{{
			RepoID:     repoID,
			BranchName: "agent/codex-01/ASE-77",
		}},
	}

	remoteRoot := "/srv/openase/workspaces"
	remoteMachine := catalogdomain.Machine{
		ID:            uuid.New(),
		Name:          "builder",
		Host:          "10.0.0.12",
		WorkspaceRoot: &remoteRoot,
	}

	request, err := buildWorkspaceRequest(launchContext, remoteMachine, true)
	if err != nil {
		t.Fatalf("buildWorkspaceRequest() error = %v", err)
	}
	if request.WorkspaceRoot != remoteRoot || request.OrganizationSlug != "acme" || request.ProjectSlug != "payments" || request.TicketIdentifier != "ASE-77" {
		t.Fatalf("buildWorkspaceRequest() = %+v", request)
	}
	if len(request.Repos) != 1 || request.Repos[0].BranchName != "agent/codex-01/ASE-77" {
		t.Fatalf("buildWorkspaceRequest().Repos = %+v", request.Repos)
	}

	workspacePath, err := buildWorkspacePath(launchContext, remoteMachine, true)
	if err != nil {
		t.Fatalf("buildWorkspacePath() error = %v", err)
	}
	if workspacePath != filepath.Join(remoteRoot, "acme", "payments", "ASE-77") {
		t.Fatalf("buildWorkspacePath() = %q", workspacePath)
	}

	if _, err := buildWorkspaceRequest(runtimeLaunchContext{project: &ent.Project{}}, remoteMachine, true); err == nil || err.Error() == "" {
		t.Fatalf("buildWorkspaceRequest(missing org) error = %v", err)
	}
	if _, err := resolveWorkspaceRoot(catalogdomain.Machine{Name: "builder", Host: "10.0.0.12"}, true); err == nil || err.Error() == "" {
		t.Fatalf("resolveWorkspaceRoot(missing remote root) error = %v", err)
	}

	if ready, reason, known := machineCodexReady(map[string]any{
		"monitor": map[string]any{
			"l4": map[string]any{
				"codex": map[string]any{"installed": true, "auth_status": "logged_in", "ready": true},
			},
		},
	}); !ready || reason != "" || !known {
		t.Fatalf("machineCodexReady(ready) = (%v, %q, %v)", ready, reason, known)
	}
	if ready, reason, known := machineCodexReady(map[string]any{
		"monitor": map[string]any{
			"l4": map[string]any{
				"codex": map[string]any{"installed": false},
			},
		},
	}); ready || reason != "codex cli is not installed" || !known {
		t.Fatalf("machineCodexReady(not installed) = (%v, %q, %v)", ready, reason, known)
	}
	if ready, reason, known := machineCodexReady(map[string]any{}); ready || reason != "" || known {
		t.Fatalf("machineCodexReady(unknown) = (%v, %q, %v)", ready, reason, known)
	}

	codexCommand := provider.MustParseAgentCLICommand(`"/usr/local/bin/codex" --stdio`)
	if !requiresMachineCodexReady(codexCommand, nil) {
		t.Fatal("requiresMachineCodexReady(codex) expected true")
	}
	if requiresMachineCodexReady(codexCommand, []string{"OPENAI_API_KEY=sk-test"}) {
		t.Fatal("requiresMachineCodexReady(api key) expected false")
	}
	if got := agentCLIExecutable(codexCommand); got != "/usr/local/bin/codex" {
		t.Fatalf("agentCLIExecutable() = %q", got)
	}
	if got := agentCLIExecutable(provider.AgentCLICommand("")); got != "" {
		t.Fatalf("agentCLIExecutable(blank) = %q", got)
	}
	if got := firstCommandToken(`"C:\Program Files\Codex\codex.exe" --serve`); got != `C:\Program Files\Codex\codex.exe` {
		t.Fatalf("firstCommandToken(quoted) = %q", got)
	}
	if !isCodexExecutablePath(`"C:\Program Files\Codex\codex.exe"`) || isCodexExecutablePath("/usr/bin/claude") {
		t.Fatal("isCodexExecutablePath() mismatch")
	}

	if got := mapHarnessMachine(remoteMachine, filepath.Join(remoteRoot, "acme", "payments", "ASE-77")); got.WorkspaceRoot != remoteRoot {
		t.Fatalf("mapHarnessMachine() = %+v", got)
	}
}

func TestRuntimeRunnerHelperCoverage(t *testing.T) {
	runID := uuid.New()
	pickupStatusID := uuid.New()
	finishStatusID := uuid.New()
	otherFinishStatusID := uuid.New()
	ticketID := uuid.New()

	prompt := buildContinuationPrompt(&ent.Ticket{ID: ticketID, Identifier: "ASE-500", Title: "Continue work"}, 3, 10, " timeout ")
	if !containsAll(prompt, "continuation turn #3 of 10", "ASE-500", "Continue work", "timeout") {
		t.Fatalf("buildContinuationPrompt() = %q", prompt)
	}
	if prompt := buildContinuationPrompt(nil, 2, 10, ""); containsAll(prompt, "Address the latest blocker") {
		t.Fatalf("buildContinuationPrompt(nil) should omit blocker line: %q", prompt)
	}

	if shouldContinueExecution(nil, runID) {
		t.Fatal("shouldContinueExecution(nil) expected false")
	}
	activeRun := &ent.AgentRun{ID: runID}
	workflow := &ent.Workflow{
		Edges: ent.WorkflowEdges{
			PickupStatuses: []*ent.TicketStatus{{ID: pickupStatusID}},
			FinishStatuses: []*ent.TicketStatus{{ID: finishStatusID}},
		},
	}
	ticket := &ent.Ticket{
		ID:           ticketID,
		StatusID:     pickupStatusID,
		WorkflowID:   &ticketID,
		CurrentRunID: &runID,
		Edges: ent.TicketEdges{
			CurrentRun: activeRun,
			Workflow:   workflow,
		},
	}
	if !shouldContinueExecution(ticket, runID) {
		t.Fatalf("shouldContinueExecution(active pickup) = false")
	}
	ticket.RetryPaused = true
	if shouldContinueExecution(ticket, runID) {
		t.Fatal("shouldContinueExecution(retry paused) expected false")
	}
	ticket.RetryPaused = false
	ticket.StatusID = finishStatusID
	if shouldContinueExecution(ticket, runID) {
		t.Fatal("shouldContinueExecution(non-pickup status) expected false")
	}

	if _, err := resolveWorkflowFinishStatus(nil); err == nil {
		t.Fatal("resolveWorkflowFinishStatus(nil) expected error")
	}
	if _, err := resolveWorkflowFinishStatus(&ent.Ticket{ID: ticketID}); err == nil {
		t.Fatal("resolveWorkflowFinishStatus(no workflow) expected error")
	}
	if _, err := resolveWorkflowFinishStatus(&ent.Ticket{
		ID:         ticketID,
		WorkflowID: &ticketID,
		Edges:      ent.TicketEdges{Workflow: &ent.Workflow{ID: ticketID}},
	}); err == nil {
		t.Fatal("resolveWorkflowFinishStatus(no finish statuses) expected error")
	}

	singleFinish, err := resolveWorkflowFinishStatus(&ent.Ticket{
		ID:         ticketID,
		StatusID:   pickupStatusID,
		WorkflowID: &ticketID,
		Edges: ent.TicketEdges{
			Workflow: &ent.Workflow{
				ID: ticketID,
				Edges: ent.WorkflowEdges{
					FinishStatuses: []*ent.TicketStatus{{ID: finishStatusID}},
				},
			},
		},
	})
	if err != nil || singleFinish != finishStatusID {
		t.Fatalf("resolveWorkflowFinishStatus(single) = %s, %v", singleFinish, err)
	}

	multiFinish, err := resolveWorkflowFinishStatus(&ent.Ticket{
		ID:         ticketID,
		StatusID:   otherFinishStatusID,
		WorkflowID: &ticketID,
		Edges: ent.TicketEdges{
			Workflow: &ent.Workflow{
				ID: ticketID,
				Edges: ent.WorkflowEdges{
					FinishStatuses: []*ent.TicketStatus{{ID: finishStatusID}, {ID: otherFinishStatusID}},
				},
			},
		},
	})
	if err != nil || multiFinish != otherFinishStatusID {
		t.Fatalf("resolveWorkflowFinishStatus(multi current) = %s, %v", multiFinish, err)
	}
	if _, err := resolveWorkflowFinishStatus(&ent.Ticket{
		ID:         ticketID,
		StatusID:   pickupStatusID,
		WorkflowID: &ticketID,
		Edges: ent.TicketEdges{
			Workflow: &ent.Workflow{
				ID: ticketID,
				Edges: ent.WorkflowEdges{
					FinishStatuses: []*ent.TicketStatus{{ID: finishStatusID}, {ID: otherFinishStatusID}},
				},
			},
		},
	}); err == nil {
		t.Fatal("resolveWorkflowFinishStatus(multi non-member) expected error")
	}

	if maxInt64(2, 5) != 5 || maxInt64(8, 3) != 8 {
		t.Fatal("maxInt64() produced unexpected result")
	}
	if ptr := int64Pointer(7); ptr == nil || *ptr != 7 {
		t.Fatalf("int64Pointer() = %+v", ptr)
	}
}

func TestRuntimeLifecycleEventAndStateCoverage(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	agentItem := fixture.createAgent(ctx, t, "codex-lifecycle", 4)
	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetAgentID(agentItem.ID).
		SetName("Lifecycle Workflow").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/lifecycle.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-700").
		SetTitle("Lifecycle coverage").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityMedium).
		SetCreatedBy("user:test").
		SetCreatedAt(time.Date(2026, time.March, 27, 12, 0, 0, 0, time.UTC)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	currentStartedAt := time.Date(2026, time.March, 27, 12, 5, 0, 0, time.UTC)
	currentHeartbeatAt := currentStartedAt.Add(2 * time.Minute)
	currentRun := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusExecuting, currentHeartbeatAt)
	currentRun, err = client.AgentRun.UpdateOneID(currentRun.ID).
		SetSessionID("sess-current").
		SetRuntimeStartedAt(currentStartedAt).
		Save(ctx)
	if err != nil {
		t.Fatalf("update current run timing: %v", err)
	}

	preferredRun, err := client.AgentRun.Create().
		SetAgentID(agentItem.ID).
		SetWorkflowID(workflowItem.ID).
		SetTicketID(ticketItem.ID).
		SetProviderID(agentItem.ProviderID).
		SetStatus(entagentrun.StatusLaunching).
		SetSessionID("sess-preferred").
		Save(ctx)
	if err != nil {
		t.Fatalf("create preferred run: %v", err)
	}

	state, err := loadAgentLifecycleState(ctx, client, agentItem.ID, nil)
	if err != nil {
		t.Fatalf("loadAgentLifecycleState(current) error = %v", err)
	}
	if state.run == nil || state.run.ID != currentRun.ID || !state.runIsCurrent {
		t.Fatalf("loadAgentLifecycleState(current) = %+v", state)
	}

	preferredState, err := loadAgentLifecycleState(ctx, client, agentItem.ID, &preferredRun.ID)
	if err != nil {
		t.Fatalf("loadAgentLifecycleState(preferred) error = %v", err)
	}
	if preferredState.run == nil || preferredState.run.ID != preferredRun.ID || preferredState.runIsCurrent {
		t.Fatalf("loadAgentLifecycleState(preferred) = %+v", preferredState)
	}

	missingPreferredID := uuid.New()
	fallbackState, err := loadAgentLifecycleState(ctx, client, agentItem.ID, &missingPreferredID)
	if err != nil {
		t.Fatalf("loadAgentLifecycleState(fallback current) error = %v", err)
	}
	if fallbackState.run == nil || fallbackState.run.ID != currentRun.ID || !fallbackState.runIsCurrent {
		t.Fatalf("loadAgentLifecycleState(fallback current) = %+v", fallbackState)
	}

	snapshot := mapAgentLifecycleSnapshot(state)
	if snapshot.RuntimeStartedAt == nil || *snapshot.RuntimeStartedAt != currentStartedAt.Format(time.RFC3339) {
		t.Fatalf("RuntimeStartedAt = %+v", snapshot.RuntimeStartedAt)
	}
	if snapshot.LastHeartbeatAt == nil || *snapshot.LastHeartbeatAt != currentHeartbeatAt.Format(time.RFC3339) {
		t.Fatalf("LastHeartbeatAt = %+v", snapshot.LastHeartbeatAt)
	}

	if timeout := stallTimeoutForWorkflow(&ent.Workflow{StallTimeoutMinutes: 9}); timeout != 9*time.Minute {
		t.Fatalf("stallTimeoutForWorkflow(custom) = %v", timeout)
	}
	if timeout := stallTimeoutForWorkflow(nil); timeout != defaultStallTimeout {
		t.Fatalf("stallTimeoutForWorkflow(default) = %v", timeout)
	}

	if err := publishAgentOutputEvent(ctx, nil, nil, fixture.projectID, agentItem.ID, ticketItem.ID, "   ", map[string]any{"stream": "stdout"}, time.Now()); err != nil {
		t.Fatalf("publishAgentOutputEvent(blank) error = %v", err)
	}
	if err := publishAgentOutputEvent(ctx, nil, nil, fixture.projectID, agentItem.ID, ticketItem.ID, " line ", map[string]any{"stream": "stdout"}, time.Now()); err == nil || !strings.Contains(err.Error(), "agent output event requires a client") {
		t.Fatalf("publishAgentOutputEvent(nil client) error = %v", err)
	}

	bus := eventinfra.NewChannelBus()
	defer func() {
		if err := bus.Close(); err != nil {
			t.Fatalf("bus.Close() error = %v", err)
		}
	}()
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := bus.Subscribe(streamCtx, activityLifecycleTopic)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	publishedAt := time.Date(2026, time.March, 27, 12, 9, 0, 0, time.UTC)
	if err := publishAgentOutputEvent(ctx, client, bus, fixture.projectID, agentItem.ID, ticketItem.ID, " stdout line ", map[string]any{"stream": "stdout"}, publishedAt); err != nil {
		t.Fatalf("publishAgentOutputEvent() error = %v", err)
	}

	select {
	case event := <-stream:
		if event.Topic != activityLifecycleTopic || event.Type != agentOutputType {
			t.Fatalf("published event = %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for activity lifecycle event")
	}

	activityItems, err := client.ActivityEvent.Query().
		Where(entactivityevent.AgentIDEQ(agentItem.ID)).
		Order(ent.Desc(entactivityevent.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		t.Fatalf("query activity events: %v", err)
	}
	if len(activityItems) == 0 || activityItems[0].Message != "stdout line" {
		t.Fatalf("activity events = %+v", activityItems)
	}
	if streamValue, ok := activityItems[0].Metadata["stream"]; !ok || streamValue != "stdout" {
		t.Fatalf("activity metadata = %+v", activityItems[0].Metadata)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
