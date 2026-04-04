package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestMachineMonitorRunTickCollectsSingleLocalMachine(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	orgID := createMachineMonitorOrg(ctx, t, client)

	if _, err := client.Machine.Create().
		SetOrganizationID(orgID).
		SetName(domain.LocalMachineName).
		SetHost(domain.LocalMachineHost).
		SetPort(22).
		SetStatus(entmachine.StatusOnline).
		SetResources(map[string]any{}).
		Save(ctx); err != nil {
		t.Fatalf("create local machine: %v", err)
	}

	now := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	collector := &fakeMachineMonitorCollector{now: func() time.Time { return now }}
	monitor := NewMachineMonitor(client, slog.New(slog.NewTextHandler(io.Discard, nil)), collector)
	monitor.now = func() time.Time { return now }

	report, err := monitor.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.MachinesScanned != 1 || report.L1Checks != 1 || report.L2Checks != 1 || report.L3Checks != 1 || report.L4Checks != 1 || report.L5Checks != 1 {
		t.Fatalf("expected local machine checks to run, got %+v", report)
	}
	if collector.reachabilityCalls != 1 || collector.systemCalls != 1 || collector.gpuCalls != 1 || collector.agentEnvCalls != 1 || collector.fullAuditCalls != 1 {
		t.Fatalf("expected collector to run for local machine, got %+v", collector)
	}
}

func TestMachineMonitorRunTickMarksRemoteMachineOfflineAfterThreeReachabilityFailures(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	orgID := createMachineMonitorOrg(ctx, t, client)

	sshUser := "openase"
	sshKeyPath := "keys/gpu-01.pem"
	machineItem, err := client.Machine.Create().
		SetOrganizationID(orgID).
		SetName("gpu-01").
		SetHost("10.0.1.10").
		SetPort(22).
		SetSSHUser(sshUser).
		SetSSHKeyPath(sshKeyPath).
		SetStatus(entmachine.StatusOnline).
		SetResources(map[string]any{}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create remote machine: %v", err)
	}

	tickTime := time.Date(2026, 3, 20, 15, 0, 0, 0, time.UTC)
	collector := &fakeMachineMonitorCollector{
		now:               func() time.Time { return tickTime },
		reachabilityError: errors.New("dial machine gpu-01: i/o timeout"),
	}
	monitor := NewMachineMonitor(client, slog.New(slog.NewTextHandler(io.Discard, nil)), collector)
	monitor.now = func() time.Time { return tickTime }

	for attempt := 1; attempt <= 3; attempt++ {
		report, err := monitor.RunTick(ctx)
		if err != nil {
			t.Fatalf("run tick %d: %v", attempt, err)
		}
		if report.L1Checks != 1 {
			t.Fatalf("expected one L1 check on attempt %d, got %+v", attempt, report)
		}
		tickTime = tickTime.Add(16 * time.Second)
	}

	machineAfter, err := client.Machine.Get(ctx, machineItem.ID)
	if err != nil {
		t.Fatalf("reload machine: %v", err)
	}
	if machineAfter.Status != entmachine.StatusOffline {
		t.Fatalf("expected machine to be offline, got %+v", machineAfter)
	}
	monitorMap := machineAfter.Resources["monitor"].(map[string]any)
	l1 := monitorMap["l1"].(map[string]any)
	if l1["consecutive_failures"] != float64(3) {
		t.Fatalf("expected 3 consecutive failures, got %+v", l1)
	}
	if machineAfter.LastHeartbeatAt == nil {
		t.Fatalf("expected heartbeat to be stamped after failures, got %+v", machineAfter)
	}
}

func TestMachineMonitorRunTickCollectsL2AndL3Snapshots(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	orgID := createMachineMonitorOrg(ctx, t, client)

	sshUser := "openase"
	sshKeyPath := "keys/gpu-02.pem"
	machineItem, err := client.Machine.Create().
		SetOrganizationID(orgID).
		SetName("gpu-02").
		SetHost("10.0.1.11").
		SetPort(22).
		SetSSHUser(sshUser).
		SetSSHKeyPath(sshKeyPath).
		SetStatus(entmachine.StatusOnline).
		SetResources(map[string]any{}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create gpu machine: %v", err)
	}

	now := time.Date(2026, 3, 20, 16, 0, 0, 0, time.UTC)
	collector := &fakeMachineMonitorCollector{
		now: func() time.Time { return now },
		systemResources: domain.MachineSystemResources{
			CollectedAt:            now,
			CPUCores:               32,
			CPUUsagePercent:        45.2,
			MemoryTotalGB:          256,
			MemoryUsedGB:           120,
			MemoryAvailableGB:      136,
			MemoryAvailablePercent: 53.13,
			DiskTotalGB:            2000,
			DiskAvailableGB:        1200,
			DiskAvailablePercent:   60,
		},
		gpuResources: domain.MachineGPUResources{
			CollectedAt: now,
			Available:   true,
			GPUs: []domain.MachineGPU{
				{Index: 1, Name: "A100-80G", MemoryTotalGB: 80, MemoryUsedGB: 80, UtilizationPercent: 100},
				{Index: 0, Name: "A100-80G", MemoryTotalGB: 80, MemoryUsedGB: 80, UtilizationPercent: 97},
			},
		},
	}
	monitor := NewMachineMonitor(client, slog.New(slog.NewTextHandler(io.Discard, nil)), collector)
	monitor.now = func() time.Time { return now }

	report, err := monitor.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.L1Checks != 1 || report.L2Checks != 1 || report.L3Checks != 1 {
		t.Fatalf("unexpected monitor report: %+v", report)
	}

	machineAfter, err := client.Machine.Get(ctx, machineItem.ID)
	if err != nil {
		t.Fatalf("reload machine: %v", err)
	}
	if machineAfter.Status != entmachine.StatusOnline {
		t.Fatalf("expected machine to stay online, got %+v", machineAfter)
	}
	if machineAfter.Resources["cpu_cores"] != float64(32) {
		t.Fatalf("expected cpu snapshot, got %+v", machineAfter.Resources)
	}
	if machineAfter.Resources["gpu_dispatchable"] != false {
		t.Fatalf("expected saturated gpus to block gpu dispatch, got %+v", machineAfter.Resources)
	}
	gpuItems, ok := machineAfter.Resources["gpu"].([]interface{})
	if !ok {
		t.Fatalf("expected gpu slice in resources, got %+v", machineAfter.Resources["gpu"])
	}
	if len(gpuItems) != 2 {
		t.Fatalf("expected 2 gpu snapshots, got %+v", gpuItems)
	}
	firstGPU := gpuItems[0].(map[string]any)
	if firstGPU["index"] != float64(0) {
		t.Fatalf("expected gpu snapshots to be index-sorted, got %+v", gpuItems)
	}
}

func TestMachineMonitorRunTickMarksNoGPUMachineUndispatchable(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	orgID := createMachineMonitorOrg(ctx, t, client)

	sshUser := "openase"
	sshKeyPath := "keys/gpu-03.pem"
	machineItem, err := client.Machine.Create().
		SetOrganizationID(orgID).
		SetName("gpu-03").
		SetHost("10.0.1.12").
		SetPort(22).
		SetSSHUser(sshUser).
		SetSSHKeyPath(sshKeyPath).
		SetStatus(entmachine.StatusOnline).
		SetResources(map[string]any{
			"gpu_dispatchable": true,
			"gpu":              []map[string]any{{"index": 0}},
		}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create gpu machine: %v", err)
	}

	now := time.Date(2026, 3, 20, 16, 5, 0, 0, time.UTC)
	collector := &fakeMachineMonitorCollector{
		now: func() time.Time { return now },
		systemResources: domain.MachineSystemResources{
			CollectedAt:            now,
			CPUCores:               32,
			CPUUsagePercent:        12.5,
			MemoryTotalGB:          256,
			MemoryUsedGB:           64,
			MemoryAvailableGB:      192,
			MemoryAvailablePercent: 75,
			DiskTotalGB:            2000,
			DiskAvailableGB:        1500,
			DiskAvailablePercent:   75,
		},
		gpuResources: domain.MachineGPUResources{
			CollectedAt: now,
			Available:   false,
		},
	}
	monitor := NewMachineMonitor(client, slog.New(slog.NewTextHandler(io.Discard, nil)), collector)
	monitor.now = func() time.Time { return now }

	report, err := monitor.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.L3Checks != 1 {
		t.Fatalf("expected one L3 check, got %+v", report)
	}

	machineAfter, err := client.Machine.Get(ctx, machineItem.ID)
	if err != nil {
		t.Fatalf("reload machine: %v", err)
	}
	if machineAfter.Resources["gpu_dispatchable"] != false {
		t.Fatalf("expected unavailable gpu machine to be undispatchable, got %+v", machineAfter.Resources)
	}
	monitorMap := machineAfter.Resources["monitor"].(map[string]any)
	l3 := monitorMap["l3"].(map[string]any)
	if l3["gpu_dispatchable"] != false {
		t.Fatalf("expected l3 monitor state to record gpu_dispatchable=false, got %+v", l3)
	}
	if gpuItems, ok := machineAfter.Resources["gpu"].([]interface{}); !ok || len(gpuItems) != 0 {
		t.Fatalf("expected empty gpu inventory, got %+v", machineAfter.Resources["gpu"])
	}
}

func TestMachineMonitorRunTickCapturesL4AndL5WithoutChangingMachineStatus(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	orgID := createMachineMonitorOrg(ctx, t, client)

	sshUser := "openase"
	sshKeyPath := "keys/gpu-04.pem"
	machineItem, err := client.Machine.Create().
		SetOrganizationID(orgID).
		SetName("builder-01").
		SetHost("10.0.1.13").
		SetPort(22).
		SetSSHUser(sshUser).
		SetSSHKeyPath(sshKeyPath).
		SetStatus(entmachine.StatusOnline).
		SetResources(map[string]any{
			"monitor": map[string]any{
				"l1": map[string]any{"checked_at": "2026-03-20T17:59:50Z"},
				"l2": map[string]any{"checked_at": "2026-03-20T17:59:30Z"},
				"l4": map[string]any{"checked_at": "2026-03-20T17:20:00Z"},
				"l5": map[string]any{"checked_at": "2026-03-20T11:00:00Z"},
			},
		}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create remote machine: %v", err)
	}

	now := time.Date(2026, 3, 20, 18, 0, 0, 0, time.UTC)
	collector := &fakeMachineMonitorCollector{
		now: func() time.Time { return now },
		agentEnvironment: domain.MachineAgentEnvironment{
			CollectedAt:  now,
			Dispatchable: true,
			CLIs: []domain.MachineAgentCLI{
				{Name: "claude_code", Installed: false, AuthStatus: domain.MachineAgentAuthStatusUnknown, AuthMode: domain.MachineAgentAuthModeUnknown},
				{Name: "codex", Installed: true, Version: "0.0.1", AuthStatus: domain.MachineAgentAuthStatusLoggedIn, AuthMode: domain.MachineAgentAuthModeLogin, Ready: true},
				{Name: "gemini", Installed: true, Version: "1.2.3", AuthStatus: domain.MachineAgentAuthStatusUnknown, AuthMode: domain.MachineAgentAuthModeUnknown, Ready: true},
			},
		},
		fullAudit: domain.MachineFullAudit{
			CollectedAt: now,
			Git: domain.MachineGitAudit{
				Installed: true,
				UserName:  "OpenASE",
				UserEmail: "openase@example.com",
			},
			GitHubCLI: domain.MachineGitHubCLIAudit{
				Installed:  true,
				AuthStatus: domain.MachineAgentAuthStatusNotLoggedIn,
			},
			Network: domain.MachineNetworkAudit{
				GitHubReachable: true,
				PyPIReachable:   false,
				NPMReachable:    true,
			},
		},
	}
	monitor := NewMachineMonitor(client, slog.New(slog.NewTextHandler(io.Discard, nil)), collector)
	monitor.now = func() time.Time { return now }

	report, err := monitor.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.L4Checks != 1 || report.L5Checks != 1 {
		t.Fatalf("expected one L4 and one L5 check, got %+v", report)
	}

	machineAfter, err := client.Machine.Get(ctx, machineItem.ID)
	if err != nil {
		t.Fatalf("reload machine: %v", err)
	}
	if machineAfter.Status != entmachine.StatusOnline {
		t.Fatalf("expected L4/L5 snapshots to keep machine online, got %+v", machineAfter)
	}
	if machineAfter.Resources["agent_dispatchable"] != true {
		t.Fatalf("expected agent dispatchability summary, got %+v", machineAfter.Resources)
	}

	monitorMap := machineAfter.Resources["monitor"].(map[string]any)
	l4 := monitorMap["l4"].(map[string]any)
	codex := l4["codex"].(map[string]any)
	if codex["installed"] != true || codex["auth_status"] != "logged_in" || codex["auth_mode"] != "login" || codex["ready"] != true {
		t.Fatalf("expected codex l4 snapshot, got %+v", codex)
	}

	fullAudit := machineAfter.Resources["full_audit"].(map[string]any)
	ghCLI := fullAudit["gh_cli"].(map[string]any)
	if ghCLI["auth_status"] != "not_logged_in" {
		t.Fatalf("expected gh cli audit summary, got %+v", ghCLI)
	}
	network := fullAudit["network"].(map[string]any)
	if network["pypi_reachable"] != false {
		t.Fatalf("expected pypi reachability=false in full audit, got %+v", network)
	}
}

func TestMachineMonitorRunTickMarksMachineDegradedWhenL4Fails(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	orgID := createMachineMonitorOrg(ctx, t, client)

	sshUser := "openase"
	sshKeyPath := "keys/gpu-05.pem"
	machineItem, err := client.Machine.Create().
		SetOrganizationID(orgID).
		SetName("builder-02").
		SetHost("10.0.1.14").
		SetPort(22).
		SetSSHUser(sshUser).
		SetSSHKeyPath(sshKeyPath).
		SetStatus(entmachine.StatusOnline).
		SetResources(map[string]any{
			"monitor": map[string]any{
				"l1": map[string]any{"checked_at": "2026-03-20T18:29:50Z"},
				"l2": map[string]any{"checked_at": "2026-03-20T18:29:30Z"},
			},
		}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create remote machine: %v", err)
	}

	now := time.Date(2026, 3, 20, 18, 30, 0, 0, time.UTC)
	collector := &fakeMachineMonitorCollector{
		now:           func() time.Time { return now },
		agentEnvError: errors.New("codex auth probe failed"),
	}
	monitor := NewMachineMonitor(client, slog.New(slog.NewTextHandler(io.Discard, nil)), collector)
	monitor.now = func() time.Time { return now }

	report, err := monitor.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.L4Checks != 1 || report.DegradedMachines != 1 {
		t.Fatalf("expected one degraded machine after L4 failure, got %+v", report)
	}

	machineAfter, err := client.Machine.Get(ctx, machineItem.ID)
	if err != nil {
		t.Fatalf("reload machine: %v", err)
	}
	if machineAfter.Status != entmachine.StatusDegraded {
		t.Fatalf("expected machine degraded after L4 failure, got %+v", machineAfter)
	}
	l4 := machineAfter.Resources["monitor"].(map[string]any)["l4"].(map[string]any)
	if l4["error"] != "codex auth probe failed" {
		t.Fatalf("expected l4 error to be recorded, got %+v", l4)
	}
}

func TestMachineMonitorRunTickPublishesMachineAndProviderRuntimeEvents(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	orgID := createMachineMonitorOrg(ctx, t, client)

	sshUser := "openase"
	workspaceRoot := "/srv/openase/workspace"
	agentCLIPath := "/usr/local/bin/codex"
	now := time.Date(2026, 3, 20, 19, 0, 0, 0, time.UTC)

	machineItem, err := client.Machine.Create().
		SetOrganizationID(orgID).
		SetName("builder-03").
		SetHost("10.0.1.15").
		SetPort(22).
		SetSSHUser(sshUser).
		SetWorkspaceRoot(workspaceRoot).
		SetAgentCliPath(agentCLIPath).
		SetStatus(entmachine.StatusDegraded).
		SetResources(map[string]any{
			"monitor": map[string]any{
				"l1": map[string]any{"checked_at": "2026-03-20T18:59:40Z"},
				"l2": map[string]any{"checked_at": "2026-03-20T18:58:00Z"},
				"l4": map[string]any{
					"checked_at": "2026-03-20T17:00:00Z",
					"codex": map[string]any{
						"installed":   true,
						"auth_status": "logged_in",
						"auth_mode":   "login",
						"ready":       true,
					},
				},
			},
		}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}

	if _, err := client.AgentProvider.Create().
		SetOrganizationID(orgID).
		SetMachineID(machineItem.ID).
		SetName("OpenAI Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetCliArgs([]string{"app-server", "--listen", "stdio://"}).
		SetAuthConfig(map[string]any{}).
		SetModelName("gpt-5.1").
		Save(ctx); err != nil {
		t.Fatalf("create provider: %v", err)
	}

	bus := eventinfra.NewChannelBus()
	stream, err := bus.Subscribe(ctx, machineEventsTopic, providerEventsTopic)
	if err != nil {
		t.Fatalf("subscribe runtime events: %v", err)
	}

	collector := &fakeMachineMonitorCollector{
		now: func() time.Time { return now },
		systemResources: domain.MachineSystemResources{
			CollectedAt:            now,
			CPUCores:               16,
			CPUUsagePercent:        22.5,
			MemoryTotalGB:          64,
			MemoryUsedGB:           24,
			MemoryAvailableGB:      40,
			MemoryAvailablePercent: 62.5,
			DiskTotalGB:            512,
			DiskAvailableGB:        320,
			DiskAvailablePercent:   62.5,
		},
		agentEnvironment: domain.MachineAgentEnvironment{
			CollectedAt:  now,
			Dispatchable: true,
			CLIs: []domain.MachineAgentCLI{
				{Name: "codex", Installed: true, Version: "0.118.0", AuthStatus: domain.MachineAgentAuthStatusLoggedIn, AuthMode: domain.MachineAgentAuthModeLogin, Ready: true},
			},
		},
		fullAudit: domain.MachineFullAudit{
			CollectedAt: now,
		},
	}
	monitor := NewMachineMonitor(client, slog.New(slog.NewTextHandler(io.Discard, nil)), collector)
	monitor.ConfigureEvents(bus)
	monitor.now = func() time.Time { return now }

	if _, err := monitor.RunTick(ctx); err != nil {
		t.Fatalf("run tick: %v", err)
	}

	events := collectMachineMonitorEvents(t, stream, 3)
	if len(events) != 3 {
		t.Fatalf("expected 3 runtime events, got %+v", events)
	}
	if events[0].Topic != machineEventsTopic || events[0].Type != machineOnlineEventType {
		t.Fatalf("expected first event machine.online, got %+v", events[0])
	}
	if events[1].Topic != machineEventsTopic || events[1].Type != machineResourcesUpdatedEventType {
		t.Fatalf("expected second event machine.resources_updated, got %+v", events[1])
	}
	if events[2].Topic != providerEventsTopic || events[2].Type != providerAvailableEventType {
		t.Fatalf("expected third event provider.available, got %+v", events[2])
	}

	var machinePayload struct {
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal(events[0].Payload, &machinePayload); err != nil {
		t.Fatalf("decode machine event payload: %v", err)
	}
	if machinePayload.OrganizationID != orgID.String() {
		t.Fatalf("expected machine event org %s, got %+v", orgID, machinePayload)
	}

	var providerPayload struct {
		OrganizationID string `json:"organization_id"`
		Provider       struct {
			AvailabilityState string `json:"availability_state"`
			Available         bool   `json:"available"`
		} `json:"provider"`
	}
	if err := json.Unmarshal(events[2].Payload, &providerPayload); err != nil {
		t.Fatalf("decode provider event payload: %v", err)
	}
	if providerPayload.OrganizationID != orgID.String() || providerPayload.Provider.AvailabilityState != domain.AgentProviderAvailabilityStateAvailable.String() || !providerPayload.Provider.Available {
		t.Fatalf("unexpected provider payload: %+v", providerPayload)
	}
}

func createMachineMonitorOrg(ctx context.Context, t *testing.T, client *ent.Client) uuid.UUID {
	t.Helper()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-machine-monitor").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	return org.ID
}

type fakeMachineMonitorCollector struct {
	now               func() time.Time
	reachabilityError error
	systemError       error
	gpuError          error
	agentEnvError     error
	fullAuditError    error
	systemResources   domain.MachineSystemResources
	gpuResources      domain.MachineGPUResources
	agentEnvironment  domain.MachineAgentEnvironment
	fullAudit         domain.MachineFullAudit
	reachabilityCalls int
	systemCalls       int
	gpuCalls          int
	agentEnvCalls     int
	fullAuditCalls    int
	lastMachine       domain.Machine
}

func (f *fakeMachineMonitorCollector) CollectReachability(_ context.Context, machine domain.Machine) (domain.MachineReachability, error) {
	f.lastMachine = machine
	f.reachabilityCalls++
	checkedAt := time.Now().UTC()
	if f.now != nil {
		checkedAt = f.now().UTC()
	}
	reachability := domain.MachineReachability{
		CheckedAt: checkedAt,
		Transport: "ssh",
		Reachable: f.reachabilityError == nil,
	}
	if f.reachabilityError != nil {
		reachability.FailureCause = f.reachabilityError.Error()
	}
	return reachability, f.reachabilityError
}

func TestMachineMonitorRunTickKeepsReverseWebsocketMachineOnlineWithoutSSHCollectors(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	orgID := createMachineMonitorOrg(ctx, t, client)

	machineItem, err := client.Machine.Create().
		SetOrganizationID(orgID).
		SetName("reverse-01").
		SetHost("reverse-01.example.com").
		SetConnectionMode(entmachine.ConnectionModeWsReverse).
		SetDaemonRegistered(true).
		SetDaemonSessionID("session-1").
		SetDaemonSessionState(entmachine.DaemonSessionStateConnected).
		SetStatus(entmachine.StatusOnline).
		SetResources(map[string]any{}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create reverse websocket machine: %v", err)
	}

	now := time.Date(2026, 4, 4, 15, 0, 0, 0, time.UTC)
	collector := &fakeMachineMonitorCollector{now: func() time.Time { return now }}
	monitor := NewMachineMonitor(client, slog.New(slog.NewTextHandler(io.Discard, nil)), collector)
	monitor.now = func() time.Time { return now }

	report, err := monitor.RunTick(ctx)
	if err != nil {
		t.Fatalf("RunTick returned error: %v", err)
	}
	if report.L1Checks != 1 || report.L2Checks != 0 || report.L3Checks != 0 || report.L4Checks != 0 || report.L5Checks != 0 {
		t.Fatalf("expected only L1 check for reverse websocket machine, got %+v", report)
	}
	if collector.systemCalls != 0 || collector.gpuCalls != 0 || collector.agentEnvCalls != 0 || collector.fullAuditCalls != 0 {
		t.Fatalf("expected websocket machine to skip SSH-only collectors, got %+v", collector)
	}
	if collector.lastMachine.ConnectionMode != domain.MachineConnectionModeWSReverse || !collector.lastMachine.DaemonStatus.Registered {
		t.Fatalf("expected collector to receive websocket connection metadata, got %+v", collector.lastMachine)
	}

	machineAfter, err := client.Machine.Get(ctx, machineItem.ID)
	if err != nil {
		t.Fatalf("reload machine: %v", err)
	}
	if machineAfter.Status != entmachine.StatusOnline {
		t.Fatalf("expected reverse websocket machine to stay online, got %+v", machineAfter)
	}
}

func (f *fakeMachineMonitorCollector) CollectSystemResources(context.Context, domain.Machine) (domain.MachineSystemResources, error) {
	f.systemCalls++
	if f.systemError != nil {
		return domain.MachineSystemResources{}, f.systemError
	}
	return f.systemResources, nil
}

func (f *fakeMachineMonitorCollector) CollectGPUResources(context.Context, domain.Machine) (domain.MachineGPUResources, error) {
	f.gpuCalls++
	if f.gpuError != nil {
		return domain.MachineGPUResources{}, f.gpuError
	}
	return f.gpuResources, nil
}

func (f *fakeMachineMonitorCollector) CollectAgentEnvironment(context.Context, domain.Machine) (domain.MachineAgentEnvironment, error) {
	f.agentEnvCalls++
	if f.agentEnvError != nil {
		return domain.MachineAgentEnvironment{}, f.agentEnvError
	}
	return f.agentEnvironment, nil
}

func (f *fakeMachineMonitorCollector) CollectFullAudit(context.Context, domain.Machine) (domain.MachineFullAudit, error) {
	f.fullAuditCalls++
	if f.fullAuditError != nil {
		return domain.MachineFullAudit{}, f.fullAuditError
	}
	return f.fullAudit, nil
}

func collectMachineMonitorEvents(t *testing.T, stream <-chan provider.Event, want int) []provider.Event {
	t.Helper()

	events := make([]provider.Event, 0, want)
	timeout := time.After(2 * time.Second)
	for len(events) < want {
		select {
		case event, ok := <-stream:
			if !ok {
				t.Fatalf("machine monitor event stream closed after %d events", len(events))
			}
			events = append(events, event)
		case <-timeout:
			t.Fatalf("timed out waiting for %d machine monitor events; got %+v", want, events)
		}
	}

	return events
}
