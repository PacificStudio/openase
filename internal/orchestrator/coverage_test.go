package orchestrator

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entagentstepevent "github.com/BetterAndBetterII/openase/ent/agentstepevent"
	entagenttraceevent "github.com/BetterAndBetterII/openase/ent/agenttraceevent"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
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

func TestNormalizeAgentStepSummaryPreservesUTF8Boundaries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		input        string
		wantExact    bool
		wantEllipsis bool
	}{
		{
			name:         "short ascii untouched",
			input:        "reviewing coverage report",
			wantExact:    true,
			wantEllipsis: false,
		},
		{
			name:         "long chinese stays valid",
			input:        strings.Repeat("上下文已经足够，准备进入产出阶段。", 16),
			wantExact:    false,
			wantEllipsis: true,
		},
		{
			name:         "mixed utf8 stays valid",
			input:        strings.Repeat("Plan 已确认，继续写 PRD。", 18),
			wantExact:    false,
			wantEllipsis: true,
		},
		{
			name:         "invalid utf8 input is repaired",
			input:        "上下文" + string([]byte{0xe7, 0x8c}) + "继续推进",
			wantExact:    false,
			wantEllipsis: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeAgentStepSummary(tc.input)
			if got == "" {
				t.Fatal("normalizeAgentStepSummary() returned empty string")
			}
			if !utf8.ValidString(got) {
				t.Fatalf("normalizeAgentStepSummary() returned invalid UTF-8: %q", got)
			}
			if len(got) > 240 {
				t.Fatalf("normalizeAgentStepSummary() length = %d, want <= 240", len(got))
			}
			if len(strings.TrimSpace(tc.input)) <= 240 && tc.wantExact {
				if got != strings.TrimSpace(tc.input) {
					t.Fatalf("normalizeAgentStepSummary() = %q, want %q", got, strings.TrimSpace(tc.input))
				}
				return
			}
			if tc.wantEllipsis && !strings.HasSuffix(got, "...") {
				t.Fatalf("normalizeAgentStepSummary() = %q, want ellipsis suffix", got)
			}
			if !tc.wantEllipsis && strings.HasSuffix(got, "...") {
				t.Fatalf("normalizeAgentStepSummary() = %q, want no ellipsis suffix", got)
			}
		})
	}
}

func TestAgentTokenUsageFromCLIUsagePreservesProviderReportedCost(t *testing.T) {
	costUSD := 0.012345
	usage := agentTokenUsageFromCLIUsage("thread-1", "turn-1", &provider.CLIUsage{
		CostUSD: &costUSD,
		Total: provider.CLIUsageTokens{
			InputTokens:  120,
			OutputTokens: 35,
			TotalTokens:  155,
		},
		Delta: provider.CLIUsageTokens{
			InputTokens:  20,
			OutputTokens: 5,
			TotalTokens:  25,
		},
	})
	if usage == nil {
		t.Fatal("agentTokenUsageFromCLIUsage() = nil, want usage event")
	}
	if usage.CostUSD == nil || *usage.CostUSD != costUSD {
		t.Fatalf("agentTokenUsageFromCLIUsage() cost = %#v, want %.6f", usage.CostUSD, costUSD)
	}

	*usage.CostUSD = 0.5
	if costUSD != 0.012345 {
		t.Fatalf("agentTokenUsageFromCLIUsage() should clone provider cost, got %.6f", costUSD)
	}
}

func TestRuntimeLauncherWorkspaceHelperCoverage(t *testing.T) {
	t.Run("build local openase environment", func(t *testing.T) {
		env, err := buildLocalOpenASEEnvironment()
		if err != nil {
			t.Fatalf("buildLocalOpenASEEnvironment() error = %v", err)
		}
		if len(env) != 1 || !strings.HasPrefix(env[0], "OPENASE_REAL_BIN=") {
			t.Fatalf("buildLocalOpenASEEnvironment() = %+v", env)
		}
		if strings.TrimSpace(strings.TrimPrefix(env[0], "OPENASE_REAL_BIN=")) == "" {
			t.Fatalf("expected executable path in environment, got %+v", env)
		}
	})

	t.Run("build github token environment", func(t *testing.T) {
		env := buildGitHubTokenEnvironment(nil, "ghu_test")
		if len(env) != 6 {
			t.Fatalf("buildGitHubTokenEnvironment() len = %d, want 6: %+v", len(env), env)
		}
		if env[0] != "GH_TOKEN=ghu_test" {
			t.Fatalf("buildGitHubTokenEnvironment() GH_TOKEN = %q", env[0])
		}
		if env[1] != "GIT_CONFIG_COUNT=2" {
			t.Fatalf("buildGitHubTokenEnvironment() GIT_CONFIG_COUNT = %q", env[1])
		}
		if env[2] != "GIT_CONFIG_KEY_0=http.https://github.com/.extraheader" {
			t.Fatalf("buildGitHubTokenEnvironment() key0 = %q", env[2])
		}
		if !strings.HasPrefix(env[3], "GIT_CONFIG_VALUE_0=AUTHORIZATION: basic ") {
			t.Fatalf("buildGitHubTokenEnvironment() value0 = %q", env[3])
		}
		if env[4] != "GIT_CONFIG_KEY_1=credential.helper" || env[5] != "GIT_CONFIG_VALUE_1=" {
			t.Fatalf("buildGitHubTokenEnvironment() helper entries = %+v", env[4:])
		}
	})

	t.Run("build github token environment appends to existing git config", func(t *testing.T) {
		env := buildGitHubTokenEnvironment([]string{"GIT_CONFIG_COUNT=1"}, "ghu_test")
		if env[1] != "GIT_CONFIG_COUNT=3" {
			t.Fatalf("buildGitHubTokenEnvironment(existing) GIT_CONFIG_COUNT = %q", env[1])
		}
		if env[2] != "GIT_CONFIG_KEY_1=http.https://github.com/.extraheader" {
			t.Fatalf("buildGitHubTokenEnvironment(existing) key1 = %q", env[2])
		}
		if env[4] != "GIT_CONFIG_KEY_2=credential.helper" {
			t.Fatalf("buildGitHubTokenEnvironment(existing) key2 = %q", env[4])
		}
	})

	t.Run("working directory selection", func(t *testing.T) {
		primaryRepoID := uuid.New()
		secondaryRepoID := uuid.New()
		launchContext := runtimeLaunchContext{
			projectRepos: []*ent.ProjectRepo{
				{ID: primaryRepoID, Name: "backend", WorkspaceDirname: "repos/backend"},
				{ID: secondaryRepoID, Name: "frontend", WorkspaceDirname: "repos/frontend"},
			},
			ticketScopes: []*ent.TicketRepoScope{
				{RepoID: secondaryRepoID},
			},
		}

		workspace := workspaceinfra.Workspace{
			Path: "/tmp/workspaces/ASE-278",
			Repos: []workspaceinfra.PreparedRepo{
				{Name: "backend", WorkspaceDirname: "repos/backend", Path: "/tmp/workspaces/ASE-278/repos/backend"},
				{Name: "frontend", WorkspaceDirname: "repos/frontend", Path: "/tmp/workspaces/ASE-278/repos/frontend"},
			},
		}

		selectedRepos, err := selectLaunchContextProjectRepos(launchContext.projectRepos, launchContext.ticketScopes)
		if err != nil || len(selectedRepos) != 1 || selectedRepos[0].ID != secondaryRepoID {
			t.Fatalf("selectLaunchContextProjectRepos(explicit scope) = %+v, %v", selectedRepos, err)
		}
		if got := resolveAgentWorkingDirectory(launchContext, workspace); got != "/tmp/workspaces/ASE-278" {
			t.Fatalf("resolveAgentWorkingDirectory(multi repo) = %q", got)
		}

		singleRepoContext := runtimeLaunchContext{
			projectRepos: []*ent.ProjectRepo{
				{ID: primaryRepoID, Name: "backend", WorkspaceDirname: "repos/backend"},
			},
		}
		selectedRepos, err = selectLaunchContextProjectRepos(singleRepoContext.projectRepos, nil)
		if err != nil || len(selectedRepos) != 1 || selectedRepos[0].ID != primaryRepoID {
			t.Fatalf("selectLaunchContextProjectRepos(single repo) = %+v, %v", selectedRepos, err)
		}
		if got := resolveAgentWorkingDirectory(singleRepoContext, workspaceinfra.Workspace{
			Path:  "/tmp/workspaces/ASE-278",
			Repos: []workspaceinfra.PreparedRepo{{Name: "backend", WorkspaceDirname: "repos/backend", Path: "/tmp/workspaces/ASE-278/repos/backend"}},
		}); got != "/tmp/workspaces/ASE-278/repos/backend" {
			t.Fatalf("resolveAgentWorkingDirectory(single repo) = %q", got)
		}
		if _, err := selectLaunchContextProjectRepos([]*ent.ProjectRepo{
			{ID: primaryRepoID, Name: "backend"},
			{ID: secondaryRepoID, Name: "frontend"},
		}, nil); !errors.Is(err, errExplicitRepoScopeRequired) {
			t.Fatalf("selectLaunchContextProjectRepos(multi repo without scope) error = %v", err)
		}
		if got := resolveAgentWorkingDirectory(runtimeLaunchContext{}, workspaceinfra.Workspace{Path: "/tmp/workspaces/ASE-278"}); got != "/tmp/workspaces/ASE-278" {
			t.Fatalf("resolveAgentWorkingDirectory(workspace root) = %q", got)
		}
		if got := projectRepoWorkspaceDirname(&ent.ProjectRepo{Name: "backend"}); got != "backend" {
			t.Fatalf("projectRepoWorkspaceDirname(default) = %q", got)
		}
		if got := projectRepoWorkspaceDirname(nil); got != "" {
			t.Fatalf("projectRepoWorkspaceDirname(nil) = %q", got)
		}
	})
}

func TestSchedulerProviderAvailabilityHelpers(t *testing.T) {
	if got := schedulerOptionalString("  "); got != nil {
		t.Fatalf("schedulerOptionalString(blank) = %+v", got)
	}
	if got := schedulerOptionalString("ready"); got == nil || *got != "ready" {
		t.Fatalf("schedulerOptionalString(value) = %+v", got)
	}

	for _, testCase := range []struct {
		state catalogdomain.AgentProviderAvailabilityState
		want  string
	}{
		{state: catalogdomain.AgentProviderAvailabilityStateStale, want: skipReasonProviderStale},
		{state: catalogdomain.AgentProviderAvailabilityStateAvailable, want: ""},
		{state: catalogdomain.AgentProviderAvailabilityStateUnknown, want: skipReasonProviderUnknown},
		{state: catalogdomain.AgentProviderAvailabilityStateUnavailable, want: skipReasonProviderUnavailable},
	} {
		if got := skipReasonForProviderAvailability(testCase.state); got != testCase.want {
			t.Fatalf("skipReasonForProviderAvailability(%q) = %q, want %q", testCase.state, got, testCase.want)
		}
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
	remoteMachineID := uuid.New()

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
			ID:               repoID,
			Name:             "backend",
			RepositoryURL:    "https://github.com/acme/backend.git",
			DefaultBranch:    "main",
			WorkspaceDirname: "services/backend",
		}},
		ticketScopes: []*ent.TicketRepoScope{{
			RepoID:     repoID,
			BranchName: "agent/ASE-77",
		}},
	}

	remoteRoot := "/srv/openase/workspaces"
	remoteMachine := catalogdomain.Machine{
		ID:            remoteMachineID,
		Name:          "builder",
		Host:          "10.0.0.12",
		WorkspaceRoot: &remoteRoot,
	}

	request, plans, err := buildWorkspaceRequest(launchContext, remoteMachine, true)
	if err != nil {
		t.Fatalf("buildWorkspaceRequest() error = %v", err)
	}
	if request.WorkspaceRoot != remoteRoot || request.OrganizationSlug != "acme" || request.ProjectSlug != "payments" || request.TicketIdentifier != "ASE-77" {
		t.Fatalf("buildWorkspaceRequest() = %+v", request)
	}
	if len(request.Repos) != 1 || request.Repos[0].BranchName != "agent/ASE-77" {
		t.Fatalf("buildWorkspaceRequest().Repos = %+v", request.Repos)
	}
	if len(plans) != 1 || plans[0].Input.RepositoryURL != "https://github.com/acme/backend.git" {
		t.Fatalf("buildWorkspaceRequest().Plans = %+v", plans)
	}

	launchContext.agent.Name = "codex-real-01"
	launchContext.ticketScopes[0].BranchName = "agent/codex-01/ASE-77"
	request, _, err = buildWorkspaceRequest(launchContext, remoteMachine, true)
	if err != nil {
		t.Fatalf("buildWorkspaceRequest(legacy scoped branch) error = %v", err)
	}
	if len(request.Repos) != 1 || request.Repos[0].BranchName != "agent/codex-01/ASE-77" {
		t.Fatalf("buildWorkspaceRequest(legacy scoped branch).Repos = %+v", request.Repos)
	}

	workspacePath, err := buildWorkspacePath(launchContext, remoteMachine, true)
	if err != nil {
		t.Fatalf("buildWorkspacePath() error = %v", err)
	}
	if workspacePath != filepath.Join(remoteRoot, "acme", "payments", "ASE-77") {
		t.Fatalf("buildWorkspacePath() = %q", workspacePath)
	}

	if _, _, err := buildWorkspaceRequest(runtimeLaunchContext{project: &ent.Project{}}, remoteMachine, true); err == nil || err.Error() == "" {
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
	if got := composeRuntimeTurnPrompt("workflow prompt", "continuation prompt"); got != "workflow prompt\n\ncontinuation prompt" {
		t.Fatalf("composeRuntimeTurnPrompt() = %q", got)
	}
	if got := composeRuntimeTurnPrompt(" workflow prompt ", " "); got != "workflow prompt" {
		t.Fatalf("composeRuntimeTurnPrompt(base-only) = %q", got)
	}
	if got := composeRuntimeTurnPrompt(" ", " continuation prompt "); got != "continuation prompt" {
		t.Fatalf("composeRuntimeTurnPrompt(continuation-only) = %q", got)
	}
	if isCleanTurnSessionClose(nil) {
		t.Fatal("isCleanTurnSessionClose(nil) expected false")
	}
	if !isCleanTurnSessionClose(&turnSessionClosedError{turnID: "turn-clean"}) {
		t.Fatal("isCleanTurnSessionClose(clean close) expected true")
	}
	if isCleanTurnSessionClose(&turnSessionClosedError{turnID: "turn-failed", cause: errors.New("boom")}) {
		t.Fatal("isCleanTurnSessionClose(with cause) expected false")
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

	if got := agentTraceKindForOutput(nil); got != catalogdomain.AgentTraceKindAssistantDelta {
		t.Fatalf("agentTraceKindForOutput(nil) = %q", got)
	}
	if got := agentTraceKindForOutput(&agentOutputEvent{Stream: "command"}); got != catalogdomain.AgentTraceKindCommandDelta {
		t.Fatalf("agentTraceKindForOutput(command delta) = %q", got)
	}
	if got := agentTraceKindForOutput(&agentOutputEvent{Stream: "command", Snapshot: true}); got != catalogdomain.AgentTraceKindCommandSnapshot {
		t.Fatalf("agentTraceKindForOutput(command snapshot) = %q", got)
	}
	if got := agentTraceKindForOutput(&agentOutputEvent{Stream: "assistant", Snapshot: true}); got != catalogdomain.AgentTraceKindAssistantSnapshot {
		t.Fatalf("agentTraceKindForOutput(assistant snapshot) = %q", got)
	}

	if stepStatus, stepSummary, ok := agentStepFromOutput(nil, "ignored"); ok || stepStatus != "" || stepSummary != "" {
		t.Fatalf("agentStepFromOutput(nil) = %q, %q, %t", stepStatus, stepSummary, ok)
	}
	if stepStatus, stepSummary, ok := agentStepFromOutput(&agentOutputEvent{Phase: " planning "}, " line one \nline two "); !ok || stepStatus != "planning" || stepSummary != "line one " {
		t.Fatalf("agentStepFromOutput(phase) = %q, %q, %t", stepStatus, stepSummary, ok)
	}
	if stepStatus, _, ok := agentStepFromOutput(&agentOutputEvent{Stream: "command"}, "run tests"); !ok || stepStatus != "running_command" {
		t.Fatalf("agentStepFromOutput(command) = %q, %t", stepStatus, ok)
	}
	if _, _, ok := agentStepFromOutput(&agentOutputEvent{Stream: "assistant"}, "reply"); ok {
		t.Fatal("agentStepFromOutput(assistant) expected false")
	}
	if _, _, ok := agentStepFromOutput(&agentOutputEvent{Stream: "unknown"}, "noop"); ok {
		t.Fatal("agentStepFromOutput(unknown) expected false")
	}

	if got := summarizeAgentStepText(" \n "); got != "" {
		t.Fatalf("summarizeAgentStepText(blank) = %q", got)
	}
	longLine := strings.Repeat("x", 141)
	if got := summarizeAgentStepText(longLine + "\nsecond"); got != strings.Repeat("x", 140)+"..." {
		t.Fatalf("summarizeAgentStepText(long) = %q", got)
	}
	longChineseLine := "仓库内容很少，当前只有 README.md；接下来我会从 OpenASE 读取工单详情、评论和工作流，再基于这些信息先落一版 workpad。"
	gotChinese := summarizeAgentStepText(longChineseLine)
	if gotChinese == "" {
		t.Fatal("summarizeAgentStepText(long Chinese) returned empty string")
	}
	if !utf8.ValidString(gotChinese) {
		t.Fatalf("summarizeAgentStepText(long Chinese) returned invalid UTF-8: %q", gotChinese)
	}
	if !strings.HasSuffix(gotChinese, "...") {
		t.Fatalf("summarizeAgentStepText(long Chinese) = %q, want ellipsis suffix", gotChinese)
	}
	if len(gotChinese) > 143 {
		t.Fatalf("summarizeAgentStepText(long Chinese) length = %d, want <= 143", len(gotChinese))
	}
	invalidLine := "进度更新" + string([]byte{0xe7, 0x8c}) + "继续"
	gotInvalid := summarizeAgentStepText(invalidLine)
	if gotInvalid == "" {
		t.Fatal("summarizeAgentStepText(invalid UTF-8) returned empty string")
	}
	if !utf8.ValidString(gotInvalid) {
		t.Fatalf("summarizeAgentStepText(invalid UTF-8) returned invalid UTF-8: %q", gotInvalid)
	}
	if strings.ContainsRune(gotInvalid, utf8.RuneError) {
		t.Fatalf("summarizeAgentStepText(invalid UTF-8) = %q, want invalid bytes removed", gotInvalid)
	}
	if got := toolCallStepSummary(" "); got != "Running provider tool call." {
		t.Fatalf("toolCallStepSummary(blank) = %q", got)
	}
	if got := toolCallStepSummary("shell"); got != `Running provider tool "shell".` {
		t.Fatalf("toolCallStepSummary(tool) = %q", got)
	}

	launcher := &RuntimeLauncher{now: func() time.Time { return time.Date(2026, 3, 27, 15, 0, 0, 0, time.UTC) }}
	if err := (*RuntimeLauncher)(nil).recordAgentOutput(context.Background(), uuid.New(), uuid.New(), uuid.New(), runID, entagentprovider.AdapterTypeCodexAppServer, nil); err != nil {
		t.Fatalf("recordAgentOutput(nil launcher) error = %v", err)
	}
	if err := launcher.recordAgentOutput(context.Background(), uuid.New(), uuid.New(), uuid.New(), runID, entagentprovider.AdapterTypeCodexAppServer, nil); err != nil {
		t.Fatalf("recordAgentOutput(nil output) error = %v", err)
	}
	if err := launcher.recordAgentOutput(context.Background(), uuid.New(), uuid.New(), uuid.New(), runID, entagentprovider.AdapterTypeCodexAppServer, &agentOutputEvent{Text: "   "}); err != nil {
		t.Fatalf("recordAgentOutput(blank text) error = %v", err)
	}
	if err := launcher.recordAgentOutput(context.Background(), uuid.New(), uuid.New(), uuid.New(), runID, entagentprovider.AdapterTypeCodexAppServer, &agentOutputEvent{
		Text:     " stderr line ",
		Stream:   "command",
		ItemID:   " item-1 ",
		TurnID:   " turn-1 ",
		Phase:    " running_command ",
		Snapshot: true,
	}); err == nil || !strings.Contains(err.Error(), "record agent output for run") || !strings.Contains(err.Error(), "agent trace event requires a client") {
		t.Fatalf("recordAgentOutput(no client) error = %v", err)
	}
	if err := (*RuntimeLauncher)(nil).recordAgentToolCall(context.Background(), uuid.New(), uuid.New(), uuid.New(), runID, entagentprovider.AdapterTypeCodexAppServer, nil); err != nil {
		t.Fatalf("recordAgentToolCall(nil launcher) error = %v", err)
	}
	if err := launcher.recordAgentToolCall(context.Background(), uuid.New(), uuid.New(), uuid.New(), runID, entagentprovider.AdapterTypeCodexAppServer, nil); err != nil {
		t.Fatalf("recordAgentToolCall(nil request) error = %v", err)
	}
	if err := launcher.recordAgentToolCall(context.Background(), uuid.New(), uuid.New(), uuid.New(), runID, entagentprovider.AdapterTypeCodexAppServer, &agentToolCallRequest{
		Tool:     " shell ",
		CallID:   " call-1 ",
		TurnID:   " turn-1 ",
		ThreadID: " thread-1 ",
	}); err == nil || !strings.Contains(err.Error(), "agent trace event requires a client") {
		t.Fatalf("recordAgentToolCall(no client) error = %v", err)
	}
	if err := (*RuntimeLauncher)(nil).recordAgentStep(context.Background(), uuid.New(), uuid.New(), uuid.New(), runID, "responding", "summary", nil); err != nil {
		t.Fatalf("recordAgentStep(nil launcher) error = %v", err)
	}
	if err := launcher.recordAgentStep(context.Background(), uuid.New(), uuid.New(), uuid.New(), runID, "responding", "summary", nil); err == nil || !strings.Contains(err.Error(), "agent step event requires a client") {
		t.Fatalf("recordAgentStep(no client) error = %v", err)
	}

	highWater := &tokenUsageHighWater{}
	if err := (*RuntimeLauncher)(nil).recordTokenUsage(context.Background(), uuid.New(), uuid.New(), uuid.New(), nil, highWater); err != nil {
		t.Fatalf("recordTokenUsage(nil launcher) error = %v", err)
	}
	launcher.tickets = ticketservice.NewService(nil)
	if err := launcher.recordTokenUsage(context.Background(), uuid.New(), uuid.New(), uuid.New(), &agentTokenUsageEvent{
		TotalInputTokens:  5,
		TotalOutputTokens: 3,
	}, highWater); err == nil || !strings.Contains(err.Error(), "record token usage for ticket") {
		t.Fatalf("recordTokenUsage(service error) error = %v", err)
	}
	if highWater.inputTokens != 5 || highWater.outputTokens != 3 {
		t.Fatalf("recordTokenUsage() highWater = %+v", highWater)
	}
	if err := launcher.recordTokenUsage(context.Background(), uuid.New(), uuid.New(), uuid.New(), &agentTokenUsageEvent{
		TotalInputTokens:  4,
		TotalOutputTokens: 2,
	}, highWater); err != nil {
		t.Fatalf("recordTokenUsage(no delta) error = %v", err)
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

	blankTrace, err := publishAgentTraceEvent(ctx, client, nil, agentTraceEventInput{
		ProjectID:   fixture.projectID,
		AgentID:     agentItem.ID,
		TicketID:    ticketItem.ID,
		AgentRunID:  currentRun.ID,
		Provider:    "codex",
		Kind:        catalogdomain.AgentTraceKindCommandDelta,
		Stream:      "stdout",
		Text:        "   ",
		EventType:   agentOutputType,
		PublishedAt: currentStartedAt.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("publishAgentTraceEvent(blank) error = %v", err)
	}
	if blankTrace.Text != "" {
		t.Fatalf("publishAgentTraceEvent(blank) = %+v", blankTrace)
	}
	if _, err := publishAgentTraceEvent(ctx, nil, nil, agentTraceEventInput{
		ProjectID:   fixture.projectID,
		AgentID:     agentItem.ID,
		TicketID:    ticketItem.ID,
		AgentRunID:  currentRun.ID,
		Provider:    "codex",
		Kind:        catalogdomain.AgentTraceKindCommandDelta,
		Stream:      "stdout",
		Text:        " line ",
		EventType:   agentOutputType,
		PublishedAt: time.Now(),
	}); err == nil || !strings.Contains(err.Error(), "agent trace event requires a client") {
		t.Fatalf("publishAgentTraceEvent(nil client) error = %v", err)
	}

	bus := eventinfra.NewChannelBus()
	defer func() {
		if err := bus.Close(); err != nil {
			t.Fatalf("bus.Close() error = %v", err)
		}
	}()
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := bus.Subscribe(streamCtx, agentTraceTopic)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	lifecycleStream, err := bus.Subscribe(streamCtx, agentLifecycleTopic)
	if err != nil {
		t.Fatalf("Subscribe(agent lifecycle) error = %v", err)
	}
	activityStream, err := bus.Subscribe(streamCtx, activityLifecycleTopic)
	if err != nil {
		t.Fatalf("Subscribe(activity lifecycle) error = %v", err)
	}
	stepStream, err := bus.Subscribe(streamCtx, agentStepTopic)
	if err != nil {
		t.Fatalf("Subscribe(agent step) error = %v", err)
	}

	publishedAt := time.Date(2026, time.March, 27, 12, 9, 0, 0, time.UTC)
	if _, err := publishAgentTraceEvent(ctx, client, bus, agentTraceEventInput{
		ProjectID:   fixture.projectID,
		AgentID:     agentItem.ID,
		TicketID:    ticketItem.ID,
		AgentRunID:  currentRun.ID,
		Provider:    "codex",
		Kind:        catalogdomain.AgentTraceKindCommandDelta,
		Stream:      "stdout",
		Text:        " stdout line ",
		Payload:     map[string]any{"stream": "stdout"},
		EventType:   agentOutputType,
		PublishedAt: publishedAt,
	}); err != nil {
		t.Fatalf("publishAgentTraceEvent() error = %v", err)
	}

	select {
	case event := <-stream:
		if event.Topic != agentTraceTopic || event.Type != agentOutputType {
			t.Fatalf("published event = %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for activity lifecycle event")
	}

	traceItems, err := client.AgentTraceEvent.Query().
		Where(entagenttraceevent.AgentIDEQ(agentItem.ID)).
		Order(ent.Desc(entagenttraceevent.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		t.Fatalf("query agent trace events: %v", err)
	}
	if len(traceItems) == 0 || traceItems[0].Text != "stdout line" || traceItems[0].Stream != "stdout" {
		t.Fatalf("agent trace events = %+v", traceItems)
	}
	if streamValue, ok := traceItems[0].Payload["stream"]; !ok || streamValue != "stdout" {
		t.Fatalf("trace payload = %+v", traceItems[0].Payload)
	}

	lifecyclePublishedAt := publishedAt.Add(30 * time.Second)
	if err := publishAgentLifecycleEvent(ctx, client, bus, agentReadyType, state, "agent ready", map[string]any{
		"status": "running",
		"phase":  "executing",
	}, lifecyclePublishedAt); err != nil {
		t.Fatalf("publishAgentLifecycleEvent() error = %v", err)
	}
	select {
	case event := <-lifecycleStream:
		if event.Topic != agentLifecycleTopic || event.Type != agentReadyType {
			t.Fatalf("agent lifecycle event = %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for agent lifecycle event")
	}
	select {
	case event := <-activityStream:
		if event.Topic != activityLifecycleTopic || event.Type != agentReadyType {
			t.Fatalf("activity lifecycle event = %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for activity lifecycle event")
	}
	if err := publishAgentLifecycleEvent(ctx, nil, nil, agentReadyType, state, "agent ready without persistence", map[string]any{"status": "running"}, lifecyclePublishedAt); err != nil {
		t.Fatalf("publishAgentLifecycleEvent(nil client, nil events) error = %v", err)
	}
	if err := publishAgentLifecycleEvent(ctx, client, bus, agentHeartbeatType, state, "agent heartbeat", map[string]any{"status": "running"}, lifecyclePublishedAt.Add(time.Second)); err != nil {
		t.Fatalf("publishAgentLifecycleEvent(heartbeat) error = %v", err)
	}
	select {
	case event := <-lifecycleStream:
		if event.Topic != agentLifecycleTopic || event.Type != agentHeartbeatType {
			t.Fatalf("agent heartbeat lifecycle event = %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for agent heartbeat lifecycle event")
	}
	if err := publishAgentLifecycleEvent(ctx, client, nil, agentReadyType, agentLifecycleState{}, "missing agent", nil, lifecyclePublishedAt); err == nil || !strings.Contains(err.Error(), "agent lifecycle event requires an agent") {
		t.Fatalf("publishAgentLifecycleEvent(missing agent) error = %v", err)
	}

	lifecycleActivities, err := client.ActivityEvent.Query().
		Where(entactivityevent.EventTypeEQ(agentReadyType.String())).
		All(ctx)
	if err != nil {
		t.Fatalf("query agent lifecycle activities: %v", err)
	}
	if len(lifecycleActivities) == 0 || lifecycleActivities[0].Message != "agent ready" || lifecycleActivities[0].Metadata["phase"] != "executing" {
		t.Fatalf("agent lifecycle activities = %+v", lifecycleActivities)
	}
	heartbeatActivities, err := client.ActivityEvent.Query().
		Where(entactivityevent.EventTypeEQ(agentHeartbeatType.String())).
		All(ctx)
	if err != nil {
		t.Fatalf("query agent heartbeat activities: %v", err)
	}
	if len(heartbeatActivities) != 0 {
		t.Fatalf("expected heartbeat lifecycle events to stay out of activity catalog, got %+v", heartbeatActivities)
	}

	launcher := &RuntimeLauncher{
		client: client,
		now:    func() time.Time { return publishedAt.Add(time.Minute) },
	}
	if err := launcher.recordAgentToolCall(ctx, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, entagentprovider.AdapterTypeCodexAppServer, &agentToolCallRequest{
		Tool:     " shell ",
		CallID:   " call-1 ",
		TurnID:   " turn-1 ",
		ThreadID: " thread-1 ",
	}); err != nil {
		t.Fatalf("recordAgentToolCall() error = %v", err)
	}
	if err := launcher.recordAgentOutput(ctx, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, entagentprovider.AdapterTypeCodexAppServer, &agentOutputEvent{
		Stream: "assistant",
		Text:   "assistant response",
		Phase:  "responding",
		TurnID: "turn-1",
	}); err != nil {
		t.Fatalf("recordAgentOutput() error = %v", err)
	}
	if err := launcher.recordAgentTaskStatus(ctx, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, entagentprovider.AdapterTypeClaudeCodeCli, &agentTaskStatusEvent{
		ThreadID:   "claude-session-1",
		TurnID:     "turn-claude-1",
		ItemID:     "tool-use-1",
		StatusType: catalogdomain.AgentTraceKindTaskProgress,
		Text:       "command",
		Payload: map[string]any{
			"stream":   "command",
			"command":  "pwd",
			"text":     "/repo\n",
			"snapshot": true,
		},
	}); err != nil {
		t.Fatalf("recordAgentTaskStatus(task_progress) error = %v", err)
	}
	if err := launcher.recordAgentTaskStatus(ctx, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, entagentprovider.AdapterTypeClaudeCodeCli, &agentTaskStatusEvent{
		ThreadID:   "claude-session-1",
		StatusType: catalogdomain.AgentTraceKindSessionState,
		Text:       "Status: active",
		Payload: map[string]any{
			"status":       "active",
			"detail":       "Running",
			"active_flags": []string{"running"},
		},
	}); err != nil {
		t.Fatalf("recordAgentTaskStatus(session_state) error = %v", err)
	}
	if err := launcher.recordAgentTaskStatus(ctx, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, entagentprovider.AdapterTypeClaudeCodeCli, &agentTaskStatusEvent{
		ThreadID:   "claude-session-1",
		StatusType: catalogdomain.AgentTraceKindError,
		Text:       "Claude Code reported an empty error result.",
		Payload: map[string]any{
			"type":     "result",
			"subtype":  "error",
			"is_error": true,
		},
	}); err != nil {
		t.Fatalf("recordAgentTaskStatus(error) error = %v", err)
	}
	if err := launcher.recordAgentApprovalRequest(ctx, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, entagentprovider.AdapterTypeCodexAppServer, &agentApprovalRequest{
		RequestID: "approval-1",
		ThreadID:  "thread-1",
		TurnID:    "turn-1",
		Kind:      "command_execution",
		Options: []agentApprovalOption{
			{ID: "approve_once", Label: "Approve once", RawDecision: "approve_once"},
		},
		Payload: map[string]any{"command": "make check"},
	}); err != nil {
		t.Fatalf("recordAgentApprovalRequest() error = %v", err)
	}
	if err := launcher.recordAgentUserInputRequest(ctx, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, entagentprovider.AdapterTypeCodexAppServer, &agentUserInputRequest{
		RequestID: "input-1",
		ThreadID:  "thread-1",
		TurnID:    "turn-1",
		Payload: map[string]any{
			"questions": []any{
				map[string]any{"id": "answer", "question": "Need confirmation?"},
			},
		},
	}); err != nil {
		t.Fatalf("recordAgentUserInputRequest() error = %v", err)
	}

	toolTrace, err := client.AgentTraceEvent.Query().
		Where(
			entagenttraceevent.AgentRunIDEQ(currentRun.ID),
			entagenttraceevent.KindEQ(catalogdomain.AgentTraceKindToolCallStarted),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("query tool-call trace event: %v", err)
	}
	if toolTrace.Stream != "tool" || toolTrace.Text != "shell" || toolTrace.Payload["call_id"] != "call-1" || toolTrace.Payload["turn_id"] != "turn-1" || toolTrace.Payload["thread_id"] != "thread-1" {
		t.Fatalf("tool call trace = %+v", toolTrace)
	}
	approvalTrace, err := client.AgentTraceEvent.Query().
		Where(
			entagenttraceevent.AgentRunIDEQ(currentRun.ID),
			entagenttraceevent.KindEQ(catalogdomain.AgentTraceKindApprovalRequested),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("query approval trace event: %v", err)
	}
	if approvalTrace.Stream != "interrupt" || approvalTrace.Payload["request_id"] != "approval-1" || approvalTrace.Payload["kind"] != "command_execution" {
		t.Fatalf("approval trace = %+v", approvalTrace)
	}
	userInputTrace, err := client.AgentTraceEvent.Query().
		Where(
			entagenttraceevent.AgentRunIDEQ(currentRun.ID),
			entagenttraceevent.KindEQ(catalogdomain.AgentTraceKindUserInputRequested),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("query user input trace event: %v", err)
	}
	if userInputTrace.Stream != "interrupt" || userInputTrace.Payload["request_id"] != "input-1" {
		t.Fatalf("user input trace = %+v", userInputTrace)
	}
	taskProgressTrace, err := client.AgentTraceEvent.Query().
		Where(
			entagenttraceevent.AgentRunIDEQ(currentRun.ID),
			entagenttraceevent.KindEQ(catalogdomain.AgentTraceKindTaskProgress),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("query task progress trace event: %v", err)
	}
	if taskProgressTrace.Stream != "task" || taskProgressTrace.Payload["item_id"] != "tool-use-1" || taskProgressTrace.Payload["command"] != "pwd" {
		t.Fatalf("task progress trace = %+v", taskProgressTrace)
	}
	sessionStateTrace, err := client.AgentTraceEvent.Query().
		Where(
			entagenttraceevent.AgentRunIDEQ(currentRun.ID),
			entagenttraceevent.KindEQ(catalogdomain.AgentTraceKindSessionState),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("query session state trace event: %v", err)
	}
	if sessionStateTrace.Payload["status"] != "active" || sessionStateTrace.Payload["detail"] != "Running" {
		t.Fatalf("session state trace = %+v", sessionStateTrace)
	}
	errorTrace, err := client.AgentTraceEvent.Query().
		Where(
			entagenttraceevent.AgentRunIDEQ(currentRun.ID),
			entagenttraceevent.KindEQ(catalogdomain.AgentTraceKindError),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("query error trace event: %v", err)
	}
	if errorTrace.Payload["subtype"] != "error" || errorTrace.Text != "Claude Code reported an empty error result." {
		t.Fatalf("error trace = %+v", errorTrace)
	}

	stepItems, err := client.AgentStepEvent.Query().
		Where(entagentstepevent.AgentRunIDEQ(currentRun.ID)).
		Order(ent.Asc(entagentstepevent.FieldCreatedAt), ent.Asc(entagentstepevent.FieldID)).
		All(ctx)
	if err != nil {
		t.Fatalf("query agent step events: %v", err)
	}
	if len(stepItems) < 2 {
		t.Fatalf("agent step events = %+v", stepItems)
	}
	var toolStepItem *ent.AgentStepEvent
	var responseStepItem *ent.AgentStepEvent
	for _, item := range stepItems {
		if item.StepStatus == "running_tool" && item.Summary == `Running provider tool "shell".` {
			toolStepItem = item
		}
		if item.StepStatus == "responding" && item.Summary == "assistant response" {
			responseStepItem = item
		}
	}
	if toolStepItem == nil || toolStepItem.SourceTraceEventID == nil || *toolStepItem.SourceTraceEventID != toolTrace.ID {
		t.Fatalf("tool step event not found in %+v", stepItems)
	}
	if responseStepItem == nil {
		t.Fatalf("output step event not found in %+v", stepItems)
	}
	if !containsStepEvent(stepItems, "awaiting_command_approval", `Waiting for command approval to run "make check".`) {
		t.Fatalf("approval step event not found in %+v", stepItems)
	}
	if !containsStepEvent(stepItems, "awaiting_input", "Waiting for user input: Need confirmation?") {
		t.Fatalf("user input step event not found in %+v", stepItems)
	}
	beforeDuplicateCount := len(stepItems)
	if err := publishAgentStepEvent(ctx, client, nil, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, "responding", "assistant response", nil, publishedAt.Add(2*time.Minute)); err != nil {
		t.Fatalf("publishAgentStepEvent(same status/summary, different source) error = %v", err)
	}
	if err := publishAgentStepEvent(ctx, client, nil, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, "responding", "assistant response", nil, publishedAt.Add(2*time.Minute+500*time.Millisecond)); err != nil {
		t.Fatalf("publishAgentStepEvent(true duplicate) error = %v", err)
	}
	if err := publishAgentStepEvent(ctx, client, nil, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, "responding", "assistant follow-up", nil, publishedAt.Add(2*time.Minute+time.Second)); err != nil {
		t.Fatalf("publishAgentStepEvent(updated summary) error = %v", err)
	}
	if err := publishAgentStepEvent(ctx, client, nil, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, "   ", "ignored", nil, publishedAt.Add(3*time.Minute)); err != nil {
		t.Fatalf("publishAgentStepEvent(blank status) error = %v", err)
	}
	stepCountAfterDuplicate, err := client.AgentStepEvent.Query().
		Where(entagentstepevent.AgentRunIDEQ(currentRun.ID)).
		Count(ctx)
	if err != nil {
		t.Fatalf("count agent step events after duplicate status: %v", err)
	}
	if stepCountAfterDuplicate != beforeDuplicateCount+2 {
		t.Fatalf("agent step event count after duplicate status = %d, want %d", stepCountAfterDuplicate, beforeDuplicateCount+2)
	}
	stepPublishedAt := publishedAt.Add(4 * time.Minute)
	if err := publishAgentStepEvent(ctx, client, bus, fixture.projectID, agentItem.ID, ticketItem.ID, currentRun.ID, "reviewing", "reviewing coverage report", nil, stepPublishedAt); err != nil {
		t.Fatalf("publishAgentStepEvent(with events) error = %v", err)
	}
	select {
	case event := <-stepStream:
		if event.Topic != agentStepTopic || event.Type != agentStepType {
			t.Fatalf("agent step stream event = %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for agent step event")
	}

	runAfter, err := client.AgentRun.Get(ctx, currentRun.ID)
	if err != nil {
		t.Fatalf("reload current run after tool/output events: %v", err)
	}
	if runAfter.CurrentStepStatus == nil || *runAfter.CurrentStepStatus != "reviewing" || runAfter.CurrentStepSummary == nil || *runAfter.CurrentStepSummary != "reviewing coverage report" {
		t.Fatalf("run step snapshot after tool/output events = %+v", runAfter)
	}
}

func TestPublishAgentStepEventAcceptsLongChineseSummary(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	agentItem := fixture.createAgent(ctx, t, "codex-utf8", 2)
	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetAgentID(agentItem.ID).
		SetName("UTF-8 Workflow").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/utf8.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-UTF8").
		SetTitle("UTF-8 step summary").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityMedium).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	runItem := mustCreateCurrentRun(
		ctx,
		t,
		client,
		agentItem,
		workflowItem.ID,
		ticketItem.ID,
		entagentrun.StatusExecuting,
		time.Date(2026, time.April, 2, 17, 14, 0, 0, time.UTC),
	)

	longChineseSummary := strings.Repeat("上下文已经足够，准备进入产出阶段。先回写工作台，确保计划、进展和验证都集中在同一条评论里。", 8)
	publishedAt := time.Date(2026, time.April, 2, 17, 14, 48, 0, time.UTC)
	if err := publishAgentStepEvent(
		ctx,
		client,
		nil,
		fixture.projectID,
		agentItem.ID,
		ticketItem.ID,
		runItem.ID,
		"commentary",
		longChineseSummary,
		nil,
		publishedAt,
	); err != nil {
		t.Fatalf("publishAgentStepEvent(long Chinese summary) error = %v", err)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run after Chinese summary: %v", err)
	}
	if runAfter.CurrentStepSummary == nil {
		t.Fatal("run current_step_summary = nil, want value")
	}
	if !utf8.ValidString(*runAfter.CurrentStepSummary) {
		t.Fatalf("run current_step_summary is invalid UTF-8: %q", *runAfter.CurrentStepSummary)
	}
	if len(*runAfter.CurrentStepSummary) > 240 {
		t.Fatalf("run current_step_summary length = %d, want <= 240", len(*runAfter.CurrentStepSummary))
	}
	if !strings.HasSuffix(*runAfter.CurrentStepSummary, "...") {
		t.Fatalf("run current_step_summary = %q, want ellipsis suffix", *runAfter.CurrentStepSummary)
	}

	stepItem, err := client.AgentStepEvent.Query().
		Where(entagentstepevent.AgentRunIDEQ(runItem.ID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("load persisted agent step event: %v", err)
	}
	if !utf8.ValidString(stepItem.Summary) {
		t.Fatalf("persisted step summary is invalid UTF-8: %q", stepItem.Summary)
	}
	if stepItem.Summary != *runAfter.CurrentStepSummary {
		t.Fatalf("persisted step summary = %q, want %q", stepItem.Summary, *runAfter.CurrentStepSummary)
	}
}

func TestRecordAgentOutputPersistsValidUTF8StepSummaryFromChinesePhaseText(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	agentItem := fixture.createAgent(ctx, t, "codex-output-utf8", 2)
	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetAgentID(agentItem.ID).
		SetName("Output UTF-8 Workflow").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/output-utf8.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-OUTPUT-UTF8").
		SetTitle("Persist UTF-8 output summary").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityMedium).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	runStartedAt := time.Date(2026, time.April, 3, 10, 0, 0, 0, time.UTC)
	runItem := mustCreateCurrentRun(
		ctx,
		t,
		client,
		agentItem,
		workflowItem.ID,
		ticketItem.ID,
		entagentrun.StatusExecuting,
		runStartedAt,
	)

	launcher := &RuntimeLauncher{
		client: client,
		now:    func() time.Time { return runStartedAt.Add(30 * time.Second) },
	}
	longChineseText := "仓库内容很少，当前只有 README.md；接下来我会从 OpenASE 读取工单详情、评论和工作流，再基于这些信息先落一版 workpad。"
	expectedSummary := summarizeAgentStepText(longChineseText)
	if expectedSummary == "" {
		t.Fatal("expectedSummary = empty, want value")
	}
	if !utf8.ValidString(expectedSummary) {
		t.Fatalf("expectedSummary is invalid UTF-8: %q", expectedSummary)
	}

	if err := launcher.recordAgentOutput(ctx, fixture.projectID, agentItem.ID, ticketItem.ID, runItem.ID, entagentprovider.AdapterTypeCodexAppServer, &agentOutputEvent{
		Stream: "assistant",
		Text:   longChineseText,
		Phase:  "commentary",
		ItemID: "item-utf8",
		TurnID: "turn-utf8",
	}); err != nil {
		t.Fatalf("recordAgentOutput(long Chinese phase text) error = %v", err)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run after long Chinese output: %v", err)
	}
	if runAfter.CurrentStepStatus == nil || *runAfter.CurrentStepStatus != "commentary" {
		t.Fatalf("run current_step_status = %+v, want commentary", runAfter.CurrentStepStatus)
	}
	if runAfter.CurrentStepSummary == nil {
		t.Fatal("run current_step_summary = nil, want value")
	}
	if *runAfter.CurrentStepSummary != expectedSummary {
		t.Fatalf("run current_step_summary = %q, want %q", *runAfter.CurrentStepSummary, expectedSummary)
	}
	if !utf8.ValidString(*runAfter.CurrentStepSummary) {
		t.Fatalf("run current_step_summary is invalid UTF-8: %q", *runAfter.CurrentStepSummary)
	}

	stepItems, err := client.AgentStepEvent.Query().
		Where(entagentstepevent.AgentRunIDEQ(runItem.ID)).
		All(ctx)
	if err != nil {
		t.Fatalf("load persisted step events: %v", err)
	}
	if len(stepItems) != 1 {
		t.Fatalf("persisted step events = %d, want 1", len(stepItems))
	}
	if stepItems[0].StepStatus != "commentary" {
		t.Fatalf("persisted step status = %q, want commentary", stepItems[0].StepStatus)
	}
	if stepItems[0].Summary != expectedSummary {
		t.Fatalf("persisted step summary = %q, want %q", stepItems[0].Summary, expectedSummary)
	}
	if !utf8.ValidString(stepItems[0].Summary) {
		t.Fatalf("persisted step summary is invalid UTF-8: %q", stepItems[0].Summary)
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

func containsStepEvent(items []*ent.AgentStepEvent, status string, summary string) bool {
	for _, item := range items {
		if item.StepStatus == status && item.Summary == summary {
			return true
		}
	}
	return false
}
