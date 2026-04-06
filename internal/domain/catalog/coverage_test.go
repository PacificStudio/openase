package catalog

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCatalogActivityAndOutputParsers(t *testing.T) {
	projectID := uuid.New()
	agentID := uuid.New()
	ticketID := uuid.New()

	parsed, err := ParseListActivityEvents(projectID, ActivityEventListInput{
		AgentID:  " " + agentID.String() + " ",
		TicketID: ticketID.String(),
		Limit:    "25",
	})
	if err != nil {
		t.Fatalf("ParseListActivityEvents() error = %v", err)
	}
	if parsed.ProjectID != projectID || parsed.AgentID == nil || *parsed.AgentID != agentID || parsed.TicketID == nil || *parsed.TicketID != ticketID || parsed.Limit != 25 {
		t.Fatalf("ParseListActivityEvents() = %+v", parsed)
	}

	output, err := ParseListAgentOutput(projectID, agentID, AgentOutputListInput{})
	if err != nil {
		t.Fatalf("ParseListAgentOutput() error = %v", err)
	}
	if output.Limit != DefaultActivityEventLimit || output.TicketID != nil {
		t.Fatalf("ParseListAgentOutput() = %+v", output)
	}

	if got := AgentOutputMetadataStream(map[string]any{"stream": " stderr "}); got != "stderr" {
		t.Fatalf("AgentOutputMetadataStream() = %q, want stderr", got)
	}
	if got := AgentOutputMetadataStream(map[string]any{"stream": ""}); got != "runtime" {
		t.Fatalf("AgentOutputMetadataStream() blank = %q, want runtime", got)
	}
	if got := AgentOutputMetadataStream(map[string]any{}); got != "runtime" {
		t.Fatalf("AgentOutputMetadataStream() missing = %q, want runtime", got)
	}

	if _, err := parseOptionalUUIDText("agent_id", "not-a-uuid"); err == nil {
		t.Fatal("parseOptionalUUIDText() expected UUID validation error")
	}
	if _, err := parseActivityEventLimit("bad"); err == nil {
		t.Fatal("parseActivityEventLimit() expected integer validation error")
	}
	if _, err := parseActivityEventLimit("0"); err == nil {
		t.Fatal("parseActivityEventLimit() expected positive validation error")
	}
	if _, err := parseActivityEventLimit("999"); err == nil {
		t.Fatal("parseActivityEventLimit() expected max validation error")
	}
	if _, err := ParseListActivityEvents(projectID, ActivityEventListInput{AgentID: "bad"}); err == nil {
		t.Fatal("ParseListActivityEvents() expected agent_id validation error")
	}
	if _, err := ParseListActivityEvents(projectID, ActivityEventListInput{TicketID: "bad"}); err == nil {
		t.Fatal("ParseListActivityEvents() expected ticket_id validation error")
	}
	if _, err := ParseListActivityEvents(projectID, ActivityEventListInput{Limit: "bad"}); err == nil {
		t.Fatal("ParseListActivityEvents() expected limit validation error")
	}
	if _, err := ParseListAgentOutput(projectID, agentID, AgentOutputListInput{TicketID: "bad"}); err == nil {
		t.Fatal("ParseListAgentOutput() expected ticket_id validation error")
	}
	if _, err := ParseListAgentOutput(projectID, agentID, AgentOutputListInput{Limit: "bad"}); err == nil {
		t.Fatal("ParseListAgentOutput() expected limit validation error")
	}

	stepInput, err := ParseListAgentSteps(projectID, agentID, AgentEventListInput{
		TicketID: ticketID.String(),
		Limit:    "17",
	})
	if err != nil {
		t.Fatalf("ParseListAgentSteps() error = %v", err)
	}
	if stepInput.ProjectID != projectID || stepInput.AgentID != agentID || stepInput.TicketID == nil || *stepInput.TicketID != ticketID || stepInput.Limit != 17 {
		t.Fatalf("ParseListAgentSteps() = %+v", stepInput)
	}
	if _, err := ParseListAgentSteps(projectID, agentID, AgentEventListInput{TicketID: "bad"}); err == nil {
		t.Fatal("ParseListAgentSteps() expected ticket_id validation error")
	}
	if _, err := ParseListAgentSteps(projectID, agentID, AgentEventListInput{Limit: "bad"}); err == nil {
		t.Fatal("ParseListAgentSteps() expected limit validation error")
	}
	if got := AgentTraceOutputKinds(); !reflect.DeepEqual(got, []string{
		AgentTraceKindAssistantDelta,
		AgentTraceKindAssistantSnapshot,
		AgentTraceKindCommandDelta,
		AgentTraceKindCommandSnapshot,
	}) {
		t.Fatalf("AgentTraceOutputKinds() = %+v", got)
	}
}

func TestCatalogAgentParsersAndRuntimeHelpers(t *testing.T) {
	organizationID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()
	machineID := uuid.New()
	runID := uuid.New()
	ticketID := uuid.New()
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)

	modelTemperature := 0.25
	modelMaxTokens := 8192
	maxParallelRuns := 3
	costPerInput := 0.001
	costPerOutput := 0.002
	createProvider, err := ParseCreateAgentProvider(organizationID, AgentProviderInput{
		MachineID:          machineID.String(),
		Name:               " Codex ",
		AdapterType:        " codex-app-server ",
		CliCommand:         " codex ",
		CliArgs:            []string{" app-server ", " --stdio "},
		AuthConfig:         map[string]any{"token": "secret"},
		ModelName:          " gpt-5.4 ",
		ModelTemperature:   &modelTemperature,
		ModelMaxTokens:     &modelMaxTokens,
		MaxParallelRuns:    &maxParallelRuns,
		CostPerInputToken:  &costPerInput,
		CostPerOutputToken: &costPerOutput,
	})
	if err != nil {
		t.Fatalf("ParseCreateAgentProvider() error = %v", err)
	}
	if createProvider.MachineID != machineID || createProvider.Name != "Codex" || createProvider.AdapterType != AgentProviderAdapterTypeCodexAppServer {
		t.Fatalf("ParseCreateAgentProvider() = %+v", createProvider)
	}
	if createProvider.MaxParallelRuns != maxParallelRuns {
		t.Fatalf("ParseCreateAgentProvider() max_parallel_runs = %d, want %d", createProvider.MaxParallelRuns, maxParallelRuns)
	}
	createProvider.AuthConfig["token"] = "changed"
	if got := cloneAnyMap(map[string]any{"k": "v"})["k"]; got != "v" {
		t.Fatalf("cloneAnyMap() = %v, want v", got)
	}
	if got := cloneAnyMap(nil); len(got) != 0 {
		t.Fatalf("cloneAnyMap(nil) = %v, want empty map", got)
	}

	updateProvider, err := ParseUpdateAgentProvider(uuid.New(), organizationID, AgentProviderInput{
		MachineID:   machineID.String(),
		Name:        "Gemini",
		AdapterType: "gemini-cli",
		ModelName:   "gemini-2.5-pro",
	})
	if err != nil {
		t.Fatalf("ParseUpdateAgentProvider() error = %v", err)
	}
	if updateProvider.ModelMaxTokens != DefaultAgentProviderModelMaxTokens || updateProvider.MaxParallelRuns != DefaultAgentProviderMaxParallelRuns || updateProvider.CostPerInputToken != DefaultAgentProviderCostPerInputToken || updateProvider.CliArgs != nil {
		t.Fatalf("ParseUpdateAgentProvider() defaults = %+v", updateProvider)
	}

	createAgent, err := ParseCreateAgent(projectID, AgentInput{
		ProviderID: providerID.String(),
		Name:       " Worker ",
	})
	if err != nil {
		t.Fatalf("ParseCreateAgent() error = %v", err)
	}
	if createAgent.ProjectID != projectID || createAgent.ProviderID != providerID || createAgent.RuntimeControlState != DefaultAgentRuntimeControlState {
		t.Fatalf("ParseCreateAgent() = %+v", createAgent)
	}

	updateAgent, err := ParseUpdateAgent(uuid.New(), projectID, AgentInput{
		ProviderID: providerID.String(),
		Name:       " Reviewer ",
	})
	if err != nil {
		t.Fatalf("ParseUpdateAgent() error = %v", err)
	}
	if updateAgent.ProjectID != projectID || updateAgent.ProviderID != providerID || updateAgent.Name != "Reviewer" {
		t.Fatalf("ParseUpdateAgent() = %+v", updateAgent)
	}
	if _, err := ParseUpdateAgent(uuid.New(), projectID, AgentInput{Name: "Reviewer"}); err == nil {
		t.Fatal("ParseUpdateAgent() expected provider validation error")
	}
	if _, err := ParseUpdateAgent(uuid.New(), projectID, AgentInput{ProviderID: providerID.String(), Name: " "}); err == nil {
		t.Fatal("ParseUpdateAgent() expected name validation error")
	}

	runtime := BuildAgentRuntimeSummary([]AgentRun{{
		ID:               runID,
		TicketID:         ticketID,
		Status:           AgentRunStatusExecuting,
		SessionID:        "sess-1",
		RuntimeStartedAt: &now,
		LastError:        "boom",
		LastHeartbeatAt:  &now,
	}}, AgentRuntimeControlStateActive)
	if runtime == nil || runtime.Status != AgentStatusRunning || runtime.RuntimePhase != AgentRuntimePhaseExecuting || runtime.CurrentRunID == nil || *runtime.CurrentRunID != runID {
		t.Fatalf("BuildAgentRuntimeSummary() executing = %+v", runtime)
	}
	if runtime.RuntimeStartedAt == &now || runtime.LastHeartbeatAt == &now {
		t.Fatal("BuildAgentRuntimeSummary() did not clone time pointers")
	}
	if got := BuildAgentRuntimeSummary([]AgentRun{{Status: AgentRunStatusTerminated}}, AgentRuntimeControlStatePaused); got.Status != AgentStatusPaused {
		t.Fatalf("BuildAgentRuntimeSummary() paused terminated status = %q, want %q", got.Status, AgentStatusPaused)
	}
	if got := BuildAgentRuntimeSummary([]AgentRun{{Status: AgentRunStatusReady}}, AgentRuntimeControlStateActive); got.RuntimePhase != AgentRuntimePhaseReady {
		t.Fatalf("BuildAgentRuntimeSummary() ready phase = %q, want %q", got.RuntimePhase, AgentRuntimePhaseReady)
	}
	if got := BuildAgentRuntimeSummary([]AgentRun{{Status: AgentRunStatusLaunching}}, AgentRuntimeControlStateActive); got.Status != AgentStatusClaimed {
		t.Fatalf("BuildAgentRuntimeSummary() launching status = %q, want claimed", got.Status)
	}
	if got := BuildAgentRuntimeSummary([]AgentRun{{Status: AgentRunStatusErrored}}, AgentRuntimeControlStateActive); got.Status != AgentStatusFailed {
		t.Fatalf("BuildAgentRuntimeSummary() errored status = %q, want failed", got.Status)
	}
	if got := BuildAgentRuntimeSummary([]AgentRun{{Status: AgentRunStatusInterrupted}}, AgentRuntimeControlStateActive); got.Status != AgentStatusInterrupted {
		t.Fatalf("BuildAgentRuntimeSummary() interrupted status = %q, want interrupted", got.Status)
	}
	if got := BuildAgentRuntimeSummary([]AgentRun{{Status: AgentRunStatusCompleted}}, AgentRuntimeControlStateActive); got.Status != DefaultAgentStatus {
		t.Fatalf("BuildAgentRuntimeSummary() completed status = %q, want idle", got.Status)
	}
	if got := BuildAgentRuntimeSummary([]AgentRun{{
		ID:                 runID,
		TicketID:           ticketID,
		Status:             AgentRunStatus("mystery"),
		CurrentStepStatus:  stringPtr(" executing "),
		CurrentStepSummary: stringPtr(" summary "),
	}}, AgentRuntimeControlStateActive); got.Status != DefaultAgentStatus || got.RuntimePhase != DefaultAgentRuntimePhase || got.CurrentStepStatus == nil || *got.CurrentStepStatus != "executing" || got.CurrentStepSummary == nil || *got.CurrentStepSummary != "summary" {
		t.Fatalf("BuildAgentRuntimeSummary() unknown status = %+v", got)
	}
	if BuildAgentRuntimeSummary(nil, AgentRuntimeControlStateActive) != nil {
		t.Fatal("BuildAgentRuntimeSummary(nil) expected nil")
	}
	multiRuntime := BuildAgentRuntimeSummary([]AgentRun{
		{
			ID:                   uuid.New(),
			TicketID:             uuid.New(),
			Status:               AgentRunStatusReady,
			LastError:            "older",
			RuntimeStartedAt:     timePtr(now.Add(-3 * time.Minute)),
			LastHeartbeatAt:      timePtr(now.Add(-2 * time.Minute)),
			CurrentStepStatus:    stringPtr("running"),
			CurrentStepSummary:   stringPtr("older"),
			CurrentStepChangedAt: timePtr(now.Add(-2 * time.Minute)),
			CreatedAt:            now.Add(-2 * time.Minute),
		},
		{
			ID:               uuid.New(),
			TicketID:         uuid.New(),
			Status:           AgentRunStatusExecuting,
			LastError:        "newer",
			RuntimeStartedAt: timePtr(now.Add(-time.Minute)),
			LastHeartbeatAt:  timePtr(now),
			CreatedAt:        now.Add(-time.Minute),
		},
	}, AgentRuntimeControlStateActive)
	if multiRuntime == nil || multiRuntime.ActiveRunCount != 2 || multiRuntime.CurrentRunID != nil || multiRuntime.CurrentTicketID != nil || multiRuntime.LastError != "newer" || multiRuntime.RuntimeStartedAt == nil || !multiRuntime.RuntimeStartedAt.Equal(now.Add(-time.Minute)) || multiRuntime.CurrentStepStatus != nil || multiRuntime.CurrentStepSummary != nil || multiRuntime.CurrentStepChangedAt != nil {
		t.Fatalf("BuildAgentRuntimeSummary(multi) = %+v", multiRuntime)
	}
	mergedRuntime := BuildAgentRuntimeSummary([]AgentRun{
		{
			ID:               uuid.New(),
			TicketID:         uuid.New(),
			Status:           AgentRunStatusExecuting,
			RuntimeStartedAt: timePtr(now.Add(-5 * time.Minute)),
			LastHeartbeatAt:  timePtr(now.Add(-5 * time.Minute)),
			CreatedAt:        now.Add(-5 * time.Minute),
		},
		{
			ID:               uuid.New(),
			TicketID:         uuid.New(),
			Status:           AgentRunStatusReady,
			RuntimeStartedAt: timePtr(now),
			LastHeartbeatAt:  timePtr(now),
			LastError:        "fresh error",
			CreatedAt:        now,
		},
	}, AgentRuntimeControlStateActive)
	if mergedRuntime == nil || mergedRuntime.RuntimeStartedAt == nil || !mergedRuntime.RuntimeStartedAt.Equal(now) || mergedRuntime.LastHeartbeatAt == nil || !mergedRuntime.LastHeartbeatAt.Equal(now) || mergedRuntime.LastError != "fresh error" {
		t.Fatalf("BuildAgentRuntimeSummary(merged timestamps) = %+v", mergedRuntime)
	}
	if !preferAgentRuntimeRepresentative(
		AgentRun{Status: AgentRunStatusExecuting, CreatedAt: now},
		AgentRun{Status: AgentRunStatusReady, CreatedAt: now.Add(time.Minute)},
	) {
		t.Fatal("preferAgentRuntimeRepresentative(priority) expected true")
	}
	if !preferAgentRuntimeRepresentative(
		AgentRun{Status: AgentRunStatusReady, CreatedAt: now.Add(time.Minute)},
		AgentRun{Status: AgentRunStatusReady, CreatedAt: now},
	) {
		t.Fatal("preferAgentRuntimeRepresentative(recency) expected true")
	}
	if preferAgentRuntimeRepresentative(
		AgentRun{Status: AgentRunStatusReady, CreatedAt: now},
		AgentRun{Status: AgentRunStatusReady, CreatedAt: now},
	) {
		t.Fatal("preferAgentRuntimeRepresentative(equal) expected false")
	}
	if got := agentRuntimeRepresentativePriority(AgentRunStatusCompleted); got != 6 {
		t.Fatalf("agentRuntimeRepresentativePriority(completed) = %d, want 6", got)
	}
	if got := agentRuntimeRepresentativePriority(AgentRunStatusReady); got != 1 {
		t.Fatalf("agentRuntimeRepresentativePriority(ready) = %d, want 1", got)
	}
	if got := agentRuntimeRepresentativePriority(AgentRunStatusLaunching); got != 2 {
		t.Fatalf("agentRuntimeRepresentativePriority(launching) = %d, want 2", got)
	}
	if got := agentRuntimeRepresentativePriority(AgentRunStatusErrored); got != 3 {
		t.Fatalf("agentRuntimeRepresentativePriority(errored) = %d, want 3", got)
	}
	if got := agentRuntimeRepresentativePriority(AgentRunStatusInterrupted); got != 4 {
		t.Fatalf("agentRuntimeRepresentativePriority(interrupted) = %d, want 4", got)
	}
	if got := agentRuntimeRepresentativePriority(AgentRunStatusTerminated); got != 5 {
		t.Fatalf("agentRuntimeRepresentativePriority(terminated) = %d, want 5", got)
	}
	if got := BuildAgentRuntimeSummary([]AgentRun{{Status: AgentRunStatusTerminated}}, AgentRuntimeControlStateActive); got.Status != AgentStatusTerminated {
		t.Fatalf("BuildAgentRuntimeSummary(terminated active) = %q, want terminated", got.Status)
	}
	if !preferAgentRuntimeError(AgentRun{LastError: "boom", LastHeartbeatAt: timePtr(now)}, "", nil) {
		t.Fatal("preferAgentRuntimeError(empty current) expected true")
	}
	if preferAgentRuntimeError(AgentRun{LastError: " ", LastHeartbeatAt: timePtr(now)}, "current", timePtr(now.Add(-time.Minute))) {
		t.Fatal("preferAgentRuntimeError(blank candidate) expected false")
	}
	if !preferAgentRuntimeError(AgentRun{LastError: "boom", LastHeartbeatAt: timePtr(now)}, "current", timePtr(now.Add(-time.Minute))) {
		t.Fatal("preferAgentRuntimeError(more recent) expected true")
	}
	if !moreRecentTime(timePtr(now), nil) {
		t.Fatal("moreRecentTime(nil current) expected true")
	}
	if moreRecentTime(nil, timePtr(now)) {
		t.Fatal("moreRecentTime(nil candidate) expected false")
	}
	if cloneTimePointer(nil) != nil {
		t.Fatal("cloneTimePointer(nil) expected nil")
	}

	if _, err := parseRequiredUUID("provider_id", ""); err == nil {
		t.Fatal("parseRequiredUUID() expected empty validation error")
	}
	if _, err := parseRequiredUUID("provider_id", "bad"); err == nil {
		t.Fatal("parseRequiredUUID() expected UUID validation error")
	}
	if _, err := parseAgentProviderAdapterType("bogus"); err == nil {
		t.Fatal("parseAgentProviderAdapterType() expected validation error")
	}
	if _, err := parseAgentProviderPermissionProfile("bogus"); err == nil {
		t.Fatal("parseAgentProviderPermissionProfile() expected validation error")
	}
	if got, err := parseAgentProviderPermissionProfile(""); err != nil || got != DefaultAgentProviderPermissionProfile {
		t.Fatalf("parseAgentProviderPermissionProfile(\"\") = %q, %v; want %q, nil", got, err, DefaultAgentProviderPermissionProfile)
	}
	if got, err := parseAgentProviderPermissionProfile(" STANDARD "); err != nil || got != AgentProviderPermissionProfileStandard {
		t.Fatalf("parseAgentProviderPermissionProfile(\" STANDARD \") = %q, %v; want %q, nil", got, err, AgentProviderPermissionProfileStandard)
	}
	if !AgentProviderPermissionProfile(AgentProviderPermissionProfileUnrestricted).IsValid() {
		t.Fatal("AgentProviderPermissionProfileUnrestricted.IsValid() expected true")
	}
	if got := AgentProviderPermissionProfile(AgentProviderPermissionProfileStandard).String(); got != "standard" {
		t.Fatalf("AgentProviderPermissionProfileStandard.String() = %q; want %q", got, "standard")
	}
	if _, err := parseStringList("cli_args", []string{"ok", " "}); err == nil {
		t.Fatal("parseStringList() expected empty item validation error")
	}
	if got, err := parseStringList("cli_args", nil); err != nil || got != nil {
		t.Fatalf("parseStringList(nil) = %v, %v; want nil, nil", got, err)
	}
	if _, err := parsePositiveInt("model_max_tokens", intPtr(0), 1); err == nil {
		t.Fatal("parsePositiveInt() expected validation error")
	}
	if got, err := parsePositiveInt("model_max_tokens", nil, 123); err != nil || got != 123 {
		t.Fatalf("parsePositiveInt(nil) = %d, %v; want 123, nil", got, err)
	}
	if _, err := parseNonNegativeFloat("model_temperature", floatPtr(-1), 0); err == nil {
		t.Fatal("parseNonNegativeFloat() expected validation error")
	}
	if got, err := parseNonNegativeFloat("model_temperature", nil, 0.5); err != nil || got != 0.5 {
		t.Fatalf("parseNonNegativeFloat(nil) = %v, %v; want 0.5, nil", got, err)
	}

	invalidProviderInputs := []AgentProviderInput{
		{Name: "Codex", AdapterType: "codex-app-server", ModelName: "gpt-5.4"},
		{MachineID: machineID.String(), Name: " ", AdapterType: "codex-app-server", ModelName: "gpt-5.4"},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "bad", ModelName: "gpt-5.4"},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", PermissionProfile: "bad", ModelName: "gpt-5.4"},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", CliArgs: []string{" "}, ModelName: "gpt-5.4"},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", ModelName: " "},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", ModelName: "gpt-5.4", ModelTemperature: floatPtr(-1)},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", ModelName: "gpt-5.4", ModelMaxTokens: intPtr(0)},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", ModelName: "gpt-5.4", MaxParallelRuns: intPtr(-1)},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", ModelName: "gpt-5.4", CostPerInputToken: floatPtr(-1)},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", ModelName: "gpt-5.4", CostPerOutputToken: floatPtr(-1)},
	}
	for _, raw := range invalidProviderInputs {
		if _, err := ParseCreateAgentProvider(organizationID, raw); err == nil {
			t.Fatalf("ParseCreateAgentProvider(%+v) expected validation error", raw)
		}
	}
	if _, err := ParseUpdateAgentProvider(uuid.New(), organizationID, AgentProviderInput{Name: "bad"}); err == nil {
		t.Fatal("ParseUpdateAgentProvider() expected validation error")
	}
	if _, err := ParseCreateAgent(projectID, AgentInput{Name: "Worker"}); err == nil {
		t.Fatal("ParseCreateAgent() expected provider validation error")
	}
	if _, err := ParseCreateAgent(projectID, AgentInput{ProviderID: providerID.String(), Name: " "}); err == nil {
		t.Fatal("ParseCreateAgent() expected name validation error")
	}
}

func TestCatalogProviderAvailabilityHelpers(t *testing.T) {
	checkedAt := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	localCLIPath := "/usr/local/bin/codex"
	workspaceRoot := "/srv/openase"
	base := AgentProvider{
		ID:                  uuid.New(),
		Name:                "OpenAI Codex",
		AdapterType:         AgentProviderAdapterTypeCodexAppServer,
		CliCommand:          "codex",
		MachineHost:         LocalMachineHost,
		MachineStatus:       MachineStatusOnline,
		MachineAgentCLIPath: &localCLIPath,
		MachineResources: map[string]any{
			"monitor": map[string]any{
				"l4": map[string]any{
					"checked_at": checkedAt.Format(time.RFC3339),
					"codex": map[string]any{
						"installed":   true,
						"auth_status": string(MachineAgentAuthStatusLoggedIn),
						"auth_mode":   string(MachineAgentAuthModeLogin),
						"ready":       true,
					},
				},
			},
		},
	}

	derived := DeriveAgentProviderAvailability(base, checkedAt)
	if !derived.Available || derived.AvailabilityState != AgentProviderAvailabilityStateAvailable || derived.AvailabilityCheckedAt == nil || derived.AvailabilityReason != nil {
		t.Fatalf("DeriveAgentProviderAvailability() = %+v", derived)
	}

	cases := []struct {
		name   string
		item   AgentProvider
		now    time.Time
		state  AgentProviderAvailabilityState
		reason string
	}{
		{name: "offline", item: AgentProvider{MachineStatus: MachineStatusOffline}, state: AgentProviderAvailabilityStateUnavailable, reason: providerReasonMachineOffline},
		{name: "degraded", item: AgentProvider{MachineStatus: MachineStatusDegraded}, state: AgentProviderAvailabilityStateUnavailable, reason: providerReasonMachineDegraded},
		{name: "maintenance", item: AgentProvider{MachineStatus: MachineStatusMaintenance}, state: AgentProviderAvailabilityStateUnavailable, reason: providerReasonMachineMaintenance},
		{name: "missing-snapshot", item: AgentProvider{MachineStatus: MachineStatusOnline}, state: AgentProviderAvailabilityStateUnknown, reason: providerReasonL4SnapshotMissing},
		{name: "stale", item: base, now: checkedAt.Add(ProviderAvailabilityStaleAfter + time.Minute), state: AgentProviderAvailabilityStateStale, reason: providerReasonStaleL4Snapshot},
		{name: "unsupported-adapter", item: AgentProvider{AdapterType: AgentProviderAdapterTypeCustom, MachineStatus: MachineStatusOnline, MachineResources: base.MachineResources, CliCommand: "custom"}, now: checkedAt, state: AgentProviderAvailabilityStateUnavailable, reason: providerReasonUnsupportedAdapter},
		{name: "cli-missing", item: withProviderCLIValue(base, "installed", false), now: checkedAt, state: AgentProviderAvailabilityStateUnavailable, reason: providerReasonCLIMissing},
		{name: "not-logged-in", item: withProviderCLIValue(base, "auth_status", string(MachineAgentAuthStatusUnknown)), now: checkedAt, state: AgentProviderAvailabilityStateUnavailable, reason: providerReasonNotLoggedIn},
		{name: "not-ready", item: withProviderCLIValue(base, "ready", false), now: checkedAt, state: AgentProviderAvailabilityStateUnavailable, reason: providerReasonNotReady},
		{name: "config-incomplete", item: AgentProvider{AdapterType: AgentProviderAdapterTypeCodexAppServer, MachineStatus: MachineStatusOnline, MachineHost: "10.0.0.10", MachineResources: base.MachineResources, CliCommand: "codex"}, now: checkedAt, state: AgentProviderAvailabilityStateUnavailable, reason: providerReasonConfigIncomplete},
		{name: "remote-ready", item: AgentProvider{AdapterType: AgentProviderAdapterTypeCodexAppServer, MachineStatus: MachineStatusOnline, MachineHost: "10.0.0.10", MachineWorkspaceRoot: &workspaceRoot, MachineResources: base.MachineResources, CliCommand: "codex"}, now: checkedAt, state: AgentProviderAvailabilityStateAvailable},
		{name: "api-key-ready", item: withProviderCLIValue(withProviderCLIValue(base, "auth_mode", string(MachineAgentAuthModeAPIKey)), "auth_status", string(MachineAgentAuthStatusUnknown)), now: checkedAt, state: AgentProviderAvailabilityStateAvailable},
	}
	for _, tc := range cases {
		state, coveredAt, reason := ResolveAgentProviderAvailability(tc.item, tc.now)
		if state != tc.state {
			t.Fatalf("%s state = %q, want %q", tc.name, state, tc.state)
		}
		if tc.reason == "" {
			if reason != nil {
				t.Fatalf("%s reason = %v, want nil", tc.name, reason)
			}
		} else if reason == nil || *reason != tc.reason {
			t.Fatalf("%s reason = %v, want %q", tc.name, reason, tc.reason)
		}
		if tc.state == AgentProviderAvailabilityStateAvailable || tc.state == AgentProviderAvailabilityStateStale {
			if coveredAt == nil || !coveredAt.Equal(checkedAt) {
				t.Fatalf("%s checkedAt = %v, want %v", tc.name, coveredAt, checkedAt)
			}
		}
	}
	nowBase := time.Now().UTC()
	zeroNowItem := AgentProvider{
		AdapterType:   AgentProviderAdapterTypeCodexAppServer,
		CliCommand:    "codex",
		MachineHost:   LocalMachineHost,
		MachineStatus: MachineStatusOnline,
		MachineResources: map[string]any{
			"monitor": map[string]any{
				"l4": map[string]any{
					"checked_at": nowBase.Format(time.RFC3339),
					"codex": map[string]any{
						"installed":   true,
						"auth_status": string(MachineAgentAuthStatusLoggedIn),
						"auth_mode":   string(MachineAgentAuthModeLogin),
						"ready":       true,
					},
				},
			},
		},
	}
	if state, _, reason := ResolveAgentProviderAvailability(zeroNowItem, time.Time{}); state != AgentProviderAvailabilityStateAvailable || reason != nil {
		t.Fatalf("ResolveAgentProviderAvailability(zero now) = %q, %v", state, reason)
	}

	l4Snapshot, coveredAt, ok := providerL4Snapshot(base.MachineResources)
	if !ok || coveredAt == nil || l4Snapshot["checked_at"] == nil {
		t.Fatalf("providerL4Snapshot() = %+v, %v, %t", l4Snapshot, coveredAt, ok)
	}
	if _, _, ok := providerL4Snapshot(map[string]any{"monitor": "bad"}); ok {
		t.Fatal("providerL4Snapshot(invalid) expected false")
	}
	if _, _, ok := providerL4Snapshot(map[string]any{}); ok {
		t.Fatal("providerL4Snapshot(missing monitor) expected false")
	}
	if _, _, ok := providerL4Snapshot(map[string]any{"monitor": map[string]any{}}); ok {
		t.Fatal("providerL4Snapshot(missing l4) expected false")
	}
	if _, _, ok := providerL4Snapshot(map[string]any{"monitor": map[string]any{"l4": map[string]any{"checked_at": 123}}}); ok {
		t.Fatal("providerL4Snapshot(non-string checked_at) expected false")
	}
	if _, _, ok := providerL4Snapshot(map[string]any{"monitor": map[string]any{"l4": map[string]any{"checked_at": ""}}}); ok {
		t.Fatal("providerL4Snapshot(blank checked_at) expected false")
	}
	if _, _, ok := providerL4Snapshot(map[string]any{"monitor": map[string]any{"l4": map[string]any{"checked_at": "not-a-time"}}}); ok {
		t.Fatal("providerL4Snapshot(invalid checked_at) expected false")
	}
	if cliSnapshot, ok := providerCLISnapshot(AgentProviderAdapterTypeCodexAppServer, l4Snapshot); !ok || cliSnapshot["installed"] != true {
		t.Fatalf("providerCLISnapshot(codex) = %+v, %t", cliSnapshot, ok)
	}
	if cliSnapshot, ok := providerCLISnapshot(AgentProviderAdapterTypeClaudeCodeCLI, map[string]any{"claude_code": map[string]any{"installed": true}}); !ok || cliSnapshot["installed"] != true {
		t.Fatalf("providerCLISnapshot(claude) = %+v, %t", cliSnapshot, ok)
	}
	if cliSnapshot, ok := providerCLISnapshot(AgentProviderAdapterTypeGeminiCLI, map[string]any{"gemini": map[string]any{"installed": true}}); !ok || cliSnapshot["installed"] != true {
		t.Fatalf("providerCLISnapshot(gemini) = %+v, %t", cliSnapshot, ok)
	}
	if _, ok := providerCLISnapshot(AgentProviderAdapterTypeCustom, l4Snapshot); ok {
		t.Fatal("providerCLISnapshot(custom) expected false")
	}
	if nested, ok := providerNestedMap(map[string]any{"monitor": map[string]any{"l4": "ok"}}, "monitor"); !ok || nested["l4"] != "ok" {
		t.Fatalf("providerNestedMap() = %+v, %t", nested, ok)
	}
	if _, ok := providerNestedMap(map[string]any{"monitor": "bad"}, "monitor"); ok {
		t.Fatal("providerNestedMap(invalid) expected false")
	}
	if !providerAuthReady(MachineAgentAuthStatusUnknown, MachineAgentAuthModeAPIKey) {
		t.Fatal("providerAuthReady(api key) expected true")
	}
	if providerAuthReady(MachineAgentAuthStatusUnknown, MachineAgentAuthModeLogin) {
		t.Fatal("providerAuthReady(login unknown) expected false")
	}
	if !providerLaunchConfigComplete(base) {
		t.Fatal("providerLaunchConfigComplete(local) expected true")
	}
	if providerLaunchConfigComplete(AgentProvider{MachineHost: LocalMachineHost}) {
		t.Fatal("providerLaunchConfigComplete(empty local command) expected false")
	}
	if !providerLaunchConfigComplete(AgentProvider{MachineHost: "10.0.0.10", CliCommand: "codex", MachineWorkspaceRoot: &workspaceRoot}) {
		t.Fatal("providerLaunchConfigComplete(remote) expected true")
	}
	if providerLaunchConfigComplete(AgentProvider{MachineHost: "10.0.0.10", CliCommand: "codex"}) {
		t.Fatal("providerLaunchConfigComplete(remote missing workspace) expected false")
	}
	if got := stringValue(" value "); got != " value " {
		t.Fatalf("stringValue() = %q", got)
	}
	if got := stringValue(123); got != "" {
		t.Fatalf("stringValue(non-string) = %q", got)
	}
	if got := availabilityReasonPointer("reason"); got == nil || *got != "reason" {
		t.Fatalf("availabilityReasonPointer() = %+v", got)
	}
	if AgentProviderAvailabilityStateAvailable.String() != "available" || !AgentProviderAvailabilityStateAvailable.IsValid() {
		t.Fatal("AgentProviderAvailabilityStateAvailable helpers failed")
	}
	if AgentProviderAvailabilityState("bogus").IsValid() {
		t.Fatal("AgentProviderAvailabilityState(invalid) expected false")
	}
}

func withProviderCLIValue(item AgentProvider, key string, value any) AgentProvider {
	cloned := cloneAnyMap(item.MachineResources)
	monitor := cloneAnyMap(cloned["monitor"].(map[string]any))
	l4 := cloneAnyMap(monitor["l4"].(map[string]any))
	codex := cloneAnyMap(l4["codex"].(map[string]any))
	codex[key] = value
	l4["codex"] = codex
	monitor["l4"] = l4
	cloned["monitor"] = monitor
	item.MachineResources = cloned
	return item
}

func TestCatalogEntityParsersAndHelpers(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	repoID := uuid.New()
	ticketID := uuid.New()
	defaultProviderID := uuid.New()
	accessibleA := uuid.New()
	accessibleB := uuid.New()
	maxConcurrent := 7
	clonePath := " services/repo "
	prURL := " https://github.com/PacificStudio/openase/pull/1 "
	branchName := " feat/openase-278-coverage "

	createOrg, err := ParseCreateOrganization(OrganizationInput{
		Name:                   " OpenASE ",
		Slug:                   " OpenASE-Main ",
		DefaultAgentProviderID: stringPtr(defaultProviderID.String()),
	})
	if err != nil {
		t.Fatalf("ParseCreateOrganization() error = %v", err)
	}
	if createOrg.Name != "OpenASE" || createOrg.Slug != "openase-main" || createOrg.DefaultAgentProviderID == nil || *createOrg.DefaultAgentProviderID != defaultProviderID {
		t.Fatalf("ParseCreateOrganization() = %+v", createOrg)
	}
	updateOrg, err := ParseUpdateOrganization(uuid.New(), OrganizationInput{Name: "Org", Slug: "org"})
	if err != nil {
		t.Fatalf("ParseUpdateOrganization() error = %v", err)
	}
	if updateOrg.Name != "Org" || updateOrg.Slug != "org" {
		t.Fatalf("ParseUpdateOrganization() = %+v", updateOrg)
	}
	if _, err := ParseUpdateOrganization(uuid.New(), OrganizationInput{Name: " ", Slug: "bad"}); err == nil {
		t.Fatal("ParseUpdateOrganization() expected validation error")
	}
	if _, err := ParseCreateOrganization(OrganizationInput{Name: "OpenASE", Slug: "bad slug"}); err == nil {
		t.Fatal("ParseCreateOrganization() expected slug validation error")
	}
	if _, err := ParseCreateOrganization(OrganizationInput{Name: "OpenASE", Slug: "openase", DefaultAgentProviderID: stringPtr("bad")}); err == nil {
		t.Fatal("ParseCreateOrganization() expected default provider validation error")
	}

	createProject, err := ParseCreateProject(orgID, ProjectInput{
		Name:                   " Coverage Rollout ",
		Slug:                   " Coverage-Rollout ",
		Description:            " Raise backend coverage ",
		Status:                 ProjectStatusInProgress.String(),
		DefaultAgentProviderID: stringPtr(defaultProviderID.String()),
		AccessibleMachineIDs:   []string{accessibleA.String(), accessibleA.String(), accessibleB.String()},
		MaxConcurrentAgents:    &maxConcurrent,
		AgentRunSummaryPrompt:  stringPtr(" Summarize the run outcome. "),
	})
	if err != nil {
		t.Fatalf("ParseCreateProject() error = %v", err)
	}
	if createProject.Description != "Raise backend coverage" || createProject.Status != ProjectStatusInProgress || len(createProject.AccessibleMachineIDs) != 2 || createProject.AgentRunSummaryPrompt != "Summarize the run outcome." {
		t.Fatalf("ParseCreateProject() = %+v", createProject)
	}
	updateProject, err := ParseUpdateProject(uuid.New(), orgID, ProjectInput{Name: "P", Slug: "p"})
	if err != nil {
		t.Fatalf("ParseUpdateProject() error = %v", err)
	}
	if updateProject.Status != DefaultProjectStatus || updateProject.MaxConcurrentAgents != DefaultProjectMaxConcurrentAgents {
		t.Fatalf("ParseUpdateProject() defaults = %+v", updateProject)
	}
	updateProject, err = ParseUpdateProject(uuid.New(), orgID, ProjectInput{Name: "Project", Slug: "project", Status: ProjectStatusCompleted.String()})
	if err != nil {
		t.Fatalf("ParseUpdateProject(success) error = %v", err)
	}
	if updateProject.Status != ProjectStatusCompleted {
		t.Fatalf("ParseUpdateProject(success) = %+v", updateProject)
	}
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: " ", Slug: "p"}); err == nil {
		t.Fatal("ParseCreateProject() expected name validation error")
	}
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: "P", Slug: "bad slug"}); err == nil {
		t.Fatal("ParseCreateProject() expected slug validation error")
	}
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: "P", Slug: "p", DefaultAgentProviderID: stringPtr("bad")}); err == nil {
		t.Fatal("ParseCreateProject() expected agent provider validation error")
	}
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: "P", Slug: "p", AccessibleMachineIDs: []string{"bad"}}); err == nil {
		t.Fatal("ParseCreateProject() expected accessible machine validation error")
	}
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: "P", Slug: "p", Status: "bad"}); err == nil {
		t.Fatal("ParseCreateProject() expected status validation error")
	}
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: "P", Slug: "p", Status: "planning"}); err == nil {
		t.Fatal("ParseCreateProject() expected legacy status validation error")
	}
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: "P", Slug: "p", Status: " Planned "}); err == nil {
		t.Fatal("ParseCreateProject() expected whitespace status validation error")
	}
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: "P", Slug: "p", Status: "planned"}); err == nil {
		t.Fatal("ParseCreateProject() expected lowercase status validation error")
	}
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: "P", Slug: "p", MaxConcurrentAgents: intPtr(-1)}); err == nil {
		t.Fatal("ParseCreateProject() expected max_concurrent_agents validation error")
	}

	createRepo, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{
		Name:             " OpenASE ",
		RepositoryURL:    " https://github.com/PacificStudio/openase ",
		DefaultBranch:    " trunk ",
		WorkspaceDirname: &clonePath,
		Labels:           []string{" backend ", "backend", " coverage "},
	})
	if err != nil {
		t.Fatalf("ParseCreateProjectRepo() error = %v", err)
	}
	if createRepo.RepositoryURL != "https://github.com/pacificstudio/openase.git" || createRepo.DefaultBranch != "trunk" || createRepo.WorkspaceDirname != "services/repo" || len(createRepo.Labels) != 2 {
		t.Fatalf("ParseCreateProjectRepo() = %+v", createRepo)
	}
	createRepo, err = ParseCreateProjectRepo(projectID, ProjectRepoInput{
		Name:             "Backend",
		RepositoryURL:    "https://github.com/PacificStudio/openase",
		WorkspaceDirname: stringPtr("   "),
	})
	if err != nil {
		t.Fatalf("ParseCreateProjectRepo(blank workspace dirname) error = %v", err)
	}
	if createRepo.RepositoryURL != "https://github.com/pacificstudio/openase.git" || createRepo.WorkspaceDirname != "Backend" {
		t.Fatalf("ParseCreateProjectRepo(blank workspace dirname) = %+v", createRepo)
	}
	updateRepo, err := ParseUpdateProjectRepo(uuid.New(), projectID, ProjectRepoInput{
		Name:          "OpenASE",
		RepositoryURL: "ssh://repo",
	})
	if err != nil {
		t.Fatalf("ParseUpdateProjectRepo() error = %v", err)
	}
	if updateRepo.DefaultBranch != "main" {
		t.Fatalf("ParseUpdateProjectRepo() defaults = %+v", updateRepo)
	}
	updateRepo, err = ParseUpdateProjectRepo(uuid.New(), projectID, ProjectRepoInput{Name: "Repo", RepositoryURL: "https://github.com/PacificStudio/openase", Labels: []string{"alpha"}})
	if err != nil {
		t.Fatalf("ParseUpdateProjectRepo(success) error = %v", err)
	}
	if len(updateRepo.Labels) != 1 || updateRepo.Labels[0] != "alpha" {
		t.Fatalf("ParseUpdateProjectRepo(success) = %+v", updateRepo)
	}
	if updateRepo.RepositoryURL != "https://github.com/pacificstudio/openase.git" {
		t.Fatalf("ParseUpdateProjectRepo(success) repository_url = %q", updateRepo.RepositoryURL)
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: " ", RepositoryURL: "https://github.com"}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected name validation error")
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: "repo", RepositoryURL: " "}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected repository_url validation error")
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: "repo", RepositoryURL: "https://github.com", WorkspaceDirname: stringPtr("/abs")}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected absolute workspace_dirname validation error")
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: "repo", RepositoryURL: "https://github.com", WorkspaceDirname: stringPtr(`nested\\repo`)}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected backslash workspace_dirname validation error")
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: "repo", RepositoryURL: "https://github.com", WorkspaceDirname: stringPtr("../repo")}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected parent traversal workspace_dirname validation error")
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: "repo", RepositoryURL: "https://github.com", WorkspaceDirname: stringPtr("./")}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected empty workspace_dirname validation error")
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: "repo", RepositoryURL: "https://github.com", WorkspaceDirname: stringPtr("repo dir")}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected spaced workspace_dirname validation error")
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: "repo", RepositoryURL: "https://github.com", Labels: []string{""}}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected labels validation error")
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: "repo", RepositoryURL: "git@github.com:PacificStudio/openase.git"}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected SSH GitHub URL validation error")
	}
	if _, err := ParseUpdateProjectRepo(uuid.New(), projectID, ProjectRepoInput{Name: "Repo", RepositoryURL: "ssh://git@github.com/PacificStudio/openase.git"}); err == nil {
		t.Fatal("ParseUpdateProjectRepo() expected SSH GitHub URL validation error")
	}
	if _, err := ParseUpdateProjectRepo(uuid.New(), projectID, ProjectRepoInput{Name: " "}); err == nil {
		t.Fatal("ParseUpdateProjectRepo() expected validation error")
	}
	if _, err := ParseUpdateProject(uuid.New(), orgID, ProjectInput{Name: " ", Slug: "project"}); err == nil {
		t.Fatal("ParseUpdateProject() expected validation error")
	}
	if got := AgentRunCompletionSummaryStatusPending.String(); got != "pending" {
		t.Fatalf("AgentRunCompletionSummaryStatusPending.String() = %q, want pending", got)
	}
	for _, tc := range []struct {
		name   string
		status AgentRunCompletionSummaryStatus
		want   bool
	}{
		{name: "pending", status: AgentRunCompletionSummaryStatusPending, want: true},
		{name: "completed", status: AgentRunCompletionSummaryStatusCompleted, want: true},
		{name: "failed", status: AgentRunCompletionSummaryStatusFailed, want: true},
		{name: "invalid", status: AgentRunCompletionSummaryStatus("unknown"), want: false},
	} {
		if got := tc.status.IsValid(); got != tc.want {
			t.Fatalf("%s status validity = %t, want %t", tc.name, got, tc.want)
		}
	}

	createScope, err := ParseCreateTicketRepoScope(projectID, ticketID, TicketRepoScopeInput{
		RepoID:         repoID.String(),
		BranchName:     &branchName,
		PullRequestURL: &prURL,
	})
	if err != nil {
		t.Fatalf("ParseCreateTicketRepoScope() error = %v", err)
	}
	if createScope.BranchName == nil || *createScope.BranchName != "feat/openase-278-coverage" || createScope.PullRequestURL == nil || *createScope.PullRequestURL != "https://github.com/PacificStudio/openase/pull/1" {
		t.Fatalf("ParseCreateTicketRepoScope() = %+v", createScope)
	}
	updateScope, err := ParseUpdateTicketRepoScope(uuid.New(), projectID, ticketID, TicketRepoScopeInput{RepoID: repoID.String()})
	if err != nil {
		t.Fatalf("ParseUpdateTicketRepoScope() error = %v", err)
	}
	if updateScope.BranchName != nil || updateScope.PullRequestURL != nil || updateScope.BranchNameSet || updateScope.PullRequestSet {
		t.Fatalf("ParseUpdateTicketRepoScope() defaults = %+v", updateScope)
	}
	updateScope, err = ParseUpdateTicketRepoScope(uuid.New(), projectID, ticketID, TicketRepoScopeInput{RepoID: repoID.String(), BranchName: stringPtr("feature/demo")})
	if err != nil {
		t.Fatalf("ParseUpdateTicketRepoScope(success) error = %v", err)
	}
	if updateScope.BranchName == nil || *updateScope.BranchName != "feature/demo" || !updateScope.BranchNameSet {
		t.Fatalf("ParseUpdateTicketRepoScope(success) = %+v", updateScope)
	}
	if _, err := ParseCreateTicketRepoScope(projectID, ticketID, TicketRepoScopeInput{RepoID: "bad"}); err == nil {
		t.Fatal("ParseCreateTicketRepoScope() expected repo_id validation error")
	}
	if _, err := ParseUpdateTicketRepoScope(uuid.New(), projectID, ticketID, TicketRepoScopeInput{RepoID: "bad"}); err == nil {
		t.Fatal("ParseUpdateTicketRepoScope() expected validation error")
	}

	if _, err := parseName("name", " "); err == nil {
		t.Fatal("parseName() expected validation error")
	}
	if _, err := parseTrimmedRequired("name", " "); err == nil {
		t.Fatal("parseTrimmedRequired() expected validation error")
	}
	if got := parseDefaultBranch(""); got != "main" {
		t.Fatalf("parseDefaultBranch() = %q, want main", got)
	}
	if got := parseOptionalText(stringPtr(" ")); got != nil {
		t.Fatalf("parseOptionalText(blank) = %v, want nil", got)
	}
	if got := parseOptionalText(stringPtr(" /tmp ")); got == nil || *got != "/tmp" {
		t.Fatalf("parseOptionalText() = %v, want /tmp", got)
	}
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	if got, err := parseMachineWorkspaceRoot(nil, "gpu-01"); err != nil || got != nil {
		t.Fatalf("parseMachineWorkspaceRoot(nil) = %v, %v", got, err)
	}
	if got, err := parseMachineWorkspaceRoot(stringPtr("~/workspace"), LocalMachineHost); err != nil || got == nil || *got != filepath.Join(homeDir, "workspace") {
		t.Fatalf("parseMachineWorkspaceRoot(local tilde) = %v, %v", got, err)
	}
	if got, err := parseMachineWorkspaceRoot(stringPtr("/srv/openase/../workspace"), "gpu-01"); err != nil || got == nil || *got != "/srv/workspace" {
		t.Fatalf("parseMachineWorkspaceRoot(remote abs) = %v, %v", got, err)
	}
	if _, err := parseMachineWorkspaceRoot(stringPtr("relative/workspace"), "gpu-01"); err == nil {
		t.Fatal("parseMachineWorkspaceRoot(relative) expected validation error")
	}
	originalHomeDir := machineUserHomeDir
	machineUserHomeDir = func() (string, error) {
		return "", errors.New("boom")
	}
	t.Cleanup(func() {
		machineUserHomeDir = originalHomeDir
	})
	if _, err := parseMachineWorkspaceRoot(stringPtr("~/workspace"), LocalMachineHost); err == nil || !strings.Contains(err.Error(), "resolve local workspace_root") {
		t.Fatalf("parseMachineWorkspaceRoot(home error) = %v, want wrapped home resolution error", err)
	}

	detectionCases := []struct {
		name         string
		detectedOS   MachineDetectedOS
		detectedArch MachineDetectedArch
		status       MachineDetectionStatus
		wantContains string
	}{
		{"ok-both", MachineDetectedOSLinux, MachineDetectedArchAMD64, MachineDetectionStatusOK, "Detected amd64 on Linux."},
		{"ok-os-only", MachineDetectedOSDarwin, MachineDetectedArchUnknown, MachineDetectionStatusOK, "Detected macOS"},
		{"ok-arch-only", MachineDetectedOSUnknown, MachineDetectedArchARM64, MachineDetectionStatusOK, "Detected arm64"},
		{"ok-none", MachineDetectedOSUnknown, MachineDetectedArchUnknown, MachineDetectionStatusOK, "completed"},
		{"degraded-os", MachineDetectedOSLinux, MachineDetectedArchUnknown, MachineDetectionStatusDegraded, "Detected Linux"},
		{"degraded-arch", MachineDetectedOSUnknown, MachineDetectedArchARM64, MachineDetectionStatusDegraded, "Detected arm64"},
		{"degraded-none", MachineDetectedOSUnknown, MachineDetectedArchUnknown, MachineDetectionStatusDegraded, "could not reliably confirm"},
		{"pending", MachineDetectedOSUnknown, MachineDetectedArchUnknown, MachineDetectionStatusPending, "has not run yet"},
		{"unknown", MachineDetectedOSUnknown, MachineDetectedArchUnknown, MachineDetectionStatusUnknown, "unknown"},
	}
	for _, tc := range detectionCases {
		if got := MachineDetectionMessage(tc.detectedOS, tc.detectedArch, tc.status); !strings.Contains(got, tc.wantContains) {
			t.Fatalf("MachineDetectionMessage(%s) = %q, want substring %q", tc.name, got, tc.wantContains)
		}
	}
	if _, err := parseLabels([]string{"ok", ""}); err == nil {
		t.Fatal("parseLabels() expected validation error")
	}
	if _, err := parseSlug("bad slug"); err == nil {
		t.Fatal("parseSlug() expected validation error")
	}
	if _, err := parseOptionalUUID("provider_id", stringPtr("bad")); err == nil {
		t.Fatal("parseOptionalUUID() expected validation error")
	}
	if got, err := parseOptionalUUID("provider_id", stringPtr(" ")); err != nil || got != nil {
		t.Fatalf("parseOptionalUUID(blank) = %v, %v; want nil, nil", got, err)
	}
	if _, err := parseUUIDList("machine_ids", []string{"", accessibleA.String()}); err == nil {
		t.Fatal("parseUUIDList() expected empty validation error")
	}
	if _, err := parseUUIDList("machine_ids", []string{"bad"}); err == nil {
		t.Fatal("parseUUIDList() expected UUID validation error")
	}
	if _, err := parseProjectStatus("invalid"); err == nil {
		t.Fatal("parseProjectStatus() expected validation error")
	}
	if got, err := parseProjectStatus(ProjectStatusBacklog.String()); err != nil || got != ProjectStatusBacklog {
		t.Fatalf("parseProjectStatus(canonical) = %q, %v", got, err)
	}
	if _, err := parseProjectStatus(" In Progress "); err == nil {
		t.Fatal("parseProjectStatus() expected exact-match validation error")
	}
	if got, err := parseMaxConcurrentAgents(intPtr(0)); err != nil || got != 0 {
		t.Fatalf("parseMaxConcurrentAgents(0) = (%d, %v), want (0, nil)", got, err)
	}
	if _, err := parseMaxConcurrentAgents(intPtr(-1)); err == nil {
		t.Fatal("parseMaxConcurrentAgents() expected validation error")
	}
}

func TestCatalogMachineParsers(t *testing.T) {
	orgID := uuid.New()
	port := 2222
	sshUser := " codex "
	sshKeyPath := " /home/codex/.ssh/id_ed25519 "
	workspaceRoot := " /srv/openase "
	agentCLIPath := " /usr/local/bin/codex "
	endpoint := " wss://builder.example.com/openase "

	createMachine, err := ParseCreateMachine(orgID, MachineInput{
		Name:               " Builder 01 ",
		Host:               " 10.0.1.8 ",
		Port:               &port,
		AdvertisedEndpoint: &endpoint,
		SSHUser:            &sshUser,
		SSHKeyPath:         &sshKeyPath,
		Description:        " Primary builder ",
		Labels:             []string{" linux ", "linux", " gpu "},
		Status:             " online ",
		WorkspaceRoot:      &workspaceRoot,
		AgentCLIPath:       &agentCLIPath,
		EnvVars:            []string{"OPENASE_ENV=prod", " OPENASE_ENV=prod ", "LOG_LEVEL=debug"},
	})
	if err != nil {
		t.Fatalf("ParseCreateMachine() error = %v", err)
	}
	if createMachine.Port != 2222 || len(createMachine.Labels) != 2 || len(createMachine.EnvVars) != 2 {
		t.Fatalf("ParseCreateMachine() = %+v", createMachine)
	}
	if createMachine.ConnectionMode != MachineConnectionModeWSListener {
		t.Fatalf("ParseCreateMachine() connection mode = %q, want ws_listener", createMachine.ConnectionMode)
	}
	if got := len(createMachine.TransportCapabilities); got != 4 {
		t.Fatalf("ParseCreateMachine() default transport capabilities = %d, want 4", got)
	}

	updateMachine, err := ParseUpdateMachine(uuid.New(), orgID, MachineInput{
		Name:   "local",
		Host:   "local",
		Status: "",
	})
	if err != nil {
		t.Fatalf("ParseUpdateMachine() error = %v", err)
	}
	if updateMachine.Status != MachineStatusOnline || updateMachine.Port != 22 {
		t.Fatalf("ParseUpdateMachine() defaults = %+v", updateMachine)
	}
	if updateMachine.ConnectionMode != MachineConnectionModeLocal {
		t.Fatalf("ParseUpdateMachine() connection mode = %q, want local", updateMachine.ConnectionMode)
	}

	registered := true
	lastRegisteredAt := "2026-04-04T10:15:00Z"
	currentSessionID := " ws-session-01 "
	endpoint = " wss://machines.example.com/connect "
	tokenID := " machine-token-01 "
	websocketMachine, err := ParseCreateMachine(orgID, MachineInput{
		Name:               " listener-01 ",
		Host:               " listener.example.com ",
		ReachabilityMode:   " direct_connect ",
		ExecutionMode:      " websocket ",
		AdvertisedEndpoint: &endpoint,
		DaemonStatus: MachineDaemonStatusInput{
			Registered:       &registered,
			LastRegisteredAt: &lastRegisteredAt,
			CurrentSessionID: &currentSessionID,
			SessionState:     " connected ",
		},
		DetectedOS:      " linux ",
		DetectedArch:    " arm64 ",
		DetectionStatus: " ok ",
		ChannelCredential: &MachineChannelCredentialInput{
			Kind:    " token ",
			TokenID: &tokenID,
		},
	})
	if err != nil {
		t.Fatalf("ParseCreateMachine(ws_listener) error = %v", err)
	}
	if websocketMachine.ConnectionMode != MachineConnectionModeWSListener {
		t.Fatalf("ParseCreateMachine(ws_listener) mode = %q", websocketMachine.ConnectionMode)
	}
	if websocketMachine.AdvertisedEndpoint == nil || *websocketMachine.AdvertisedEndpoint != "wss://machines.example.com/connect" {
		t.Fatalf("ParseCreateMachine(ws_listener) endpoint = %v", websocketMachine.AdvertisedEndpoint)
	}
	if websocketMachine.DetectedOS != MachineDetectedOSLinux {
		t.Fatalf("ParseCreateMachine(ws_listener) detected os = %q", websocketMachine.DetectedOS)
	}
	if websocketMachine.DetectedArch != MachineDetectedArchARM64 {
		t.Fatalf("ParseCreateMachine(ws_listener) detected arch = %q", websocketMachine.DetectedArch)
	}
	if websocketMachine.DetectionStatus != MachineDetectionStatusOK {
		t.Fatalf("ParseCreateMachine(ws_listener) detection status = %q", websocketMachine.DetectionStatus)
	}
	if websocketMachine.DaemonStatus.CurrentSessionID == nil || *websocketMachine.DaemonStatus.CurrentSessionID != "ws-session-01" {
		t.Fatalf("ParseCreateMachine(ws_listener) daemon status = %+v", websocketMachine.DaemonStatus)
	}
	if websocketMachine.ChannelCredential.Kind != MachineChannelCredentialKindToken ||
		websocketMachine.ChannelCredential.TokenID == nil ||
		*websocketMachine.ChannelCredential.TokenID != "machine-token-01" {
		t.Fatalf("ParseCreateMachine(ws_listener) channel credential = %+v", websocketMachine.ChannelCredential)
	}
	if got := len(websocketMachine.TransportCapabilities); got != 4 {
		t.Fatalf("ParseCreateMachine(ws_listener) transport capabilities = %d, want 4", got)
	}

	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "local"}); err == nil {
		t.Fatal("ParseCreateMachine() expected local host/name mismatch error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "local", Host: "remote"}); err == nil {
		t.Fatal("ParseCreateMachine() expected local name/host mismatch error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: " ", Host: "10.0.0.1"}); err == nil {
		t.Fatal("ParseCreateMachine() expected name validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: " "}); err == nil {
		t.Fatal("ParseCreateMachine() expected host validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1", Port: intPtr(70000), AdvertisedEndpoint: &endpoint}); err == nil {
		t.Fatal("ParseCreateMachine() expected port validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1", AdvertisedEndpoint: &endpoint, Labels: []string{""}}); err == nil {
		t.Fatal("ParseCreateMachine() expected labels validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1", AdvertisedEndpoint: &endpoint, Status: "bad"}); err == nil {
		t.Fatal("ParseCreateMachine() expected status validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1", AdvertisedEndpoint: &endpoint, EnvVars: []string{"NOPE"}}); err == nil {
		t.Fatal("ParseCreateMachine() expected env_vars validation error")
	}
	relativeWorkspaceRoot := "relative/workspace"
	if _, err := ParseCreateMachine(orgID, MachineInput{
		Name:               "remote",
		Host:               "10.0.0.1",
		AdvertisedEndpoint: &endpoint,
		WorkspaceRoot:      &relativeWorkspaceRoot,
	}); err == nil {
		t.Fatal("ParseCreateMachine() expected workspace_root validation error")
	}
	if _, err := ParseUpdateMachine(uuid.New(), orgID, MachineInput{Name: "remote", Host: "10.0.0.1"}); err == nil {
		t.Fatal("ParseUpdateMachine() expected validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1", ExecutionMode: "ssh_compat"}); err == nil {
		t.Fatal("ParseCreateMachine() expected execution_mode validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{
		Name:             "listener",
		Host:             "listener.example.com",
		ReachabilityMode: "direct_connect",
		ExecutionMode:    "websocket",
	}); err == nil {
		t.Fatal("ParseCreateMachine() expected advertised_endpoint validation error")
	}
	badEndpoint := "https://machines.example.com/connect"
	if _, err := ParseCreateMachine(orgID, MachineInput{
		Name:               "listener",
		Host:               "listener.example.com",
		ReachabilityMode:   "direct_connect",
		ExecutionMode:      "websocket",
		AdvertisedEndpoint: &badEndpoint,
	}); err == nil {
		t.Fatal("ParseCreateMachine() expected advertised_endpoint scheme validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "listener", Host: "listener.example.com", ReachabilityMode: "reverse_connect", ExecutionMode: "websocket", DetectedOS: "windows"}); err == nil {
		t.Fatal("ParseCreateMachine() expected detected_os validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "listener", Host: "listener.example.com", ReachabilityMode: "reverse_connect", ExecutionMode: "websocket", DetectedArch: "386"}); err == nil {
		t.Fatal("ParseCreateMachine() expected detected_arch validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "listener", Host: "listener.example.com", ReachabilityMode: "reverse_connect", ExecutionMode: "websocket", DetectionStatus: "bad"}); err == nil {
		t.Fatal("ParseCreateMachine() expected detection_status validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{
		Name:             "listener",
		Host:             "listener.example.com",
		ReachabilityMode: "reverse_connect",
		ExecutionMode:    "websocket",
		DaemonStatus:     MachineDaemonStatusInput{SessionState: "broken"},
	}); err == nil {
		t.Fatal("ParseCreateMachine() expected daemon_status.session_state validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{
		Name:             "listener",
		Host:             "listener.example.com",
		ReachabilityMode: "reverse_connect",
		ExecutionMode:    "websocket",
		ChannelCredential: &MachineChannelCredentialInput{
			Kind: "token",
		},
	}); err == nil {
		t.Fatal("ParseCreateMachine() expected channel credential token validation error")
	}
	if _, err := parseMachineName(""); err == nil {
		t.Fatal("parseMachineName() expected validation error")
	}
	if got, err := parseMachineHost("example.com"); err != nil || got != "example.com" {
		t.Fatalf("parseMachineHost() = %q, %v; want example.com, nil", got, err)
	}
	if _, err := parseMachineHost("has space"); err == nil {
		t.Fatal("parseMachineHost() expected space validation error")
	}
	if _, err := parseMachineHost(" "); err == nil {
		t.Fatal("parseMachineHost() expected empty validation error")
	}
	if _, err := parseMachinePort(intPtr(70000)); err == nil {
		t.Fatal("parseMachinePort() expected range validation error")
	}
	if got, err := parseMachineStatus("", false); err != nil || got != MachineStatusMaintenance {
		t.Fatalf("parseMachineStatus(remote default) = %q, %v; want maintenance, nil", got, err)
	}
	if _, err := parseMachineStatus("bad", true); err == nil {
		t.Fatal("parseMachineStatus() expected validation error")
	}
	if _, err := parseMachineEnvVars([]string{"", "KEY=VALUE"}); err == nil {
		t.Fatal("parseMachineEnvVars() expected empty validation error")
	}
	if _, err := parseMachineEnvVars([]string{"NOPE"}); err == nil {
		t.Fatal("parseMachineEnvVars() expected format validation error")
	}
	if _, err := parseMachineEnvVars([]string{" =value"}); err == nil {
		t.Fatal("parseMachineEnvVars() expected key validation error")
	}
}

func TestCatalogMachineTransportHelpers(t *testing.T) {
	if MachineConnectionModeWSReverse.String() != "ws_reverse" {
		t.Fatalf("MachineConnectionMode.String() = %q", MachineConnectionModeWSReverse.String())
	}
	if MachineTransportCapabilityProcessStreaming.String() != "process_streaming" {
		t.Fatalf("MachineTransportCapability.String() = %q", MachineTransportCapabilityProcessStreaming.String())
	}
	if MachineDetectedOSLinux.String() != "linux" {
		t.Fatalf("MachineDetectedOS.String() = %q", MachineDetectedOSLinux.String())
	}
	if MachineDetectedArchAMD64.String() != "amd64" {
		t.Fatalf("MachineDetectedArch.String() = %q", MachineDetectedArchAMD64.String())
	}
	if MachineDetectionStatusPending.String() != "pending" {
		t.Fatalf("MachineDetectionStatus.String() = %q", MachineDetectionStatusPending.String())
	}
	if MachineChannelCredentialKindCertificate.String() != "certificate" {
		t.Fatalf("MachineChannelCredentialKind.String() = %q", MachineChannelCredentialKindCertificate.String())
	}
	if MachineTransportSessionStateDisconnected.String() != "disconnected" {
		t.Fatalf("MachineTransportSessionState.String() = %q", MachineTransportSessionStateDisconnected.String())
	}
	if MachineTransportCapability("bad").IsValid() {
		t.Fatal("MachineTransportCapability.IsValid() expected false")
	}
	if MachineChannelCredentialKind("bad").IsValid() {
		t.Fatal("MachineChannelCredentialKind.IsValid() expected false")
	}
	if got := defaultMachineTransportCapabilities(MachineConnectionMode("bad")); got != nil {
		t.Fatalf("defaultMachineTransportCapabilities(invalid) = %+v, want nil", got)
	}

	mode, err := ParseStoredMachineConnectionMode("", "10.0.0.9")
	if err != nil || mode != MachineConnectionModeWSListener {
		t.Fatalf("ParseStoredMachineConnectionMode(remote default) = %q, %v", mode, err)
	}
	if mode, err = ParseStoredMachineConnectionMode("", LocalMachineHost); err != nil || mode != MachineConnectionModeLocal {
		t.Fatalf("ParseStoredMachineConnectionMode(local default) = %q, %v", mode, err)
	}
	capabilities, err := ParseStoredMachineTransportCapabilities(nil, MachineConnectionModeWSListener)
	if err != nil || len(capabilities) != 4 {
		t.Fatalf("ParseStoredMachineTransportCapabilities(default) = %+v, %v", capabilities, err)
	}
	if _, err := ParseStoredMachineTransportCapabilities([]string{"bad"}, MachineConnectionModeWSListener); err == nil {
		t.Fatal("ParseStoredMachineTransportCapabilities() expected validation error")
	}
	if detectedOS, err := ParseStoredMachineDetectedOS("linux"); err != nil || detectedOS != MachineDetectedOSLinux {
		t.Fatalf("ParseStoredMachineDetectedOS() = %q, %v", detectedOS, err)
	}
	if detectedArch, err := ParseStoredMachineDetectedArch("amd64"); err != nil || detectedArch != MachineDetectedArchAMD64 {
		t.Fatalf("ParseStoredMachineDetectedArch() = %q, %v", detectedArch, err)
	}
	if detectionStatus, err := ParseStoredMachineDetectionStatus("degraded"); err != nil || detectionStatus != MachineDetectionStatusDegraded {
		t.Fatalf("ParseStoredMachineDetectionStatus() = %q, %v", detectionStatus, err)
	}
	if credentialKind, err := ParseStoredMachineChannelCredentialKind("certificate"); err != nil || credentialKind != MachineChannelCredentialKindCertificate {
		t.Fatalf("ParseStoredMachineChannelCredentialKind() = %q, %v", credentialKind, err)
	}
	if credentialKind, err := ParseStoredMachineChannelCredentialKind(""); err != nil || credentialKind != MachineChannelCredentialKindNone {
		t.Fatalf("ParseStoredMachineChannelCredentialKind(empty) = %q, %v", credentialKind, err)
	}
	if _, err := ParseStoredMachineChannelCredentialKind("broken"); err == nil {
		t.Fatal("ParseStoredMachineChannelCredentialKind() expected validation error")
	}
	if sessionState, err := ParseStoredMachineSessionState("connected"); err != nil || sessionState != MachineTransportSessionStateConnected {
		t.Fatalf("ParseStoredMachineSessionState() = %q, %v", sessionState, err)
	}
	if sessionState, err := ParseStoredMachineSessionState(""); err != nil || sessionState != MachineTransportSessionStateUnknown {
		t.Fatalf("ParseStoredMachineSessionState(empty) = %q, %v", sessionState, err)
	}
	if _, err := ParseStoredMachineSessionState("broken"); err == nil {
		t.Fatal("ParseStoredMachineSessionState() expected validation error")
	}

	validEndpoint := "ws://listener.example.com/daemon"
	if endpoint, err := parseMachineAdvertisedEndpoint(&validEndpoint, MachineConnectionModeWSListener); err != nil || endpoint == nil || *endpoint != validEndpoint {
		t.Fatalf("parseMachineAdvertisedEndpoint(valid) = %v, %v", endpoint, err)
	}
	badURL := "://bad"
	if _, err := parseMachineAdvertisedEndpoint(&badURL, MachineConnectionModeWSListener); err == nil {
		t.Fatal("parseMachineAdvertisedEndpoint() expected URL validation error")
	}
	noHostEndpoint := "ws:///missing-host"
	if _, err := parseMachineAdvertisedEndpoint(&noHostEndpoint, MachineConnectionModeWSListener); err == nil {
		t.Fatal("parseMachineAdvertisedEndpoint() expected host validation error")
	}

	lastRegisteredAt := "2026-04-04T12:00:00Z"
	sessionID := "session-1"
	registered := true
	daemonStatus, err := parseMachineDaemonStatus(MachineDaemonStatusInput{
		Registered:       &registered,
		LastRegisteredAt: &lastRegisteredAt,
		CurrentSessionID: &sessionID,
		SessionState:     "disconnected",
	})
	if err != nil || daemonStatus.SessionState != MachineTransportSessionStateDisconnected {
		t.Fatalf("parseMachineDaemonStatus() = %+v, %v", daemonStatus, err)
	}
	badTime := "not-a-time"
	if _, err := parseMachineDaemonStatus(MachineDaemonStatusInput{LastRegisteredAt: &badTime}); err == nil {
		t.Fatal("parseMachineDaemonStatus() expected timestamp validation error")
	}
	if _, err := parseMachineDaemonStatus(MachineDaemonStatusInput{SessionState: "broken"}); err == nil {
		t.Fatal("parseMachineDaemonStatus() expected session_state validation error")
	}

	tokenID := "machine-token"
	certificateID := "machine-cert"
	if credential, err := parseMachineChannelCredential(nil); err != nil || credential.Kind != MachineChannelCredentialKindNone {
		t.Fatalf("parseMachineChannelCredential(nil) = %+v, %v", credential, err)
	}
	if credential, err := parseMachineChannelCredential(&MachineChannelCredentialInput{Kind: "none", TokenID: &tokenID, CertificateID: &certificateID}); err != nil || credential.Kind != MachineChannelCredentialKindNone || credential.TokenID != nil || credential.CertificateID != nil {
		t.Fatalf("parseMachineChannelCredential(none) = %+v, %v", credential, err)
	}
	if credential, err := parseMachineChannelCredential(&MachineChannelCredentialInput{Kind: "   "}); err != nil || credential.Kind != MachineChannelCredentialKindNone || credential.TokenID != nil || credential.CertificateID != nil {
		t.Fatalf("parseMachineChannelCredential(blank) = %+v, %v", credential, err)
	}
	if credential, err := parseMachineChannelCredential(&MachineChannelCredentialInput{Kind: "token", TokenID: &tokenID, CertificateID: &certificateID}); err != nil || credential.Kind != MachineChannelCredentialKindToken || credential.CertificateID != nil {
		t.Fatalf("parseMachineChannelCredential(token) = %+v, %v", credential, err)
	}
	if credential, err := parseMachineChannelCredential(&MachineChannelCredentialInput{Kind: "certificate", TokenID: &tokenID, CertificateID: &certificateID}); err != nil || credential.Kind != MachineChannelCredentialKindCertificate || credential.TokenID != nil {
		t.Fatalf("parseMachineChannelCredential(certificate) = %+v, %v", credential, err)
	}
	if _, err := parseMachineChannelCredential(&MachineChannelCredentialInput{Kind: "token"}); err == nil {
		t.Fatal("parseMachineChannelCredential(token missing id) expected validation error")
	}
	if _, err := parseMachineChannelCredential(&MachineChannelCredentialInput{Kind: "certificate"}); err == nil {
		t.Fatal("parseMachineChannelCredential(certificate missing id) expected validation error")
	}
	if _, err := parseMachineChannelCredential(&MachineChannelCredentialInput{Kind: "broken"}); err == nil {
		t.Fatal("parseMachineChannelCredential() expected kind validation error")
	}

	statusTime := time.Date(2026, 4, 4, 12, 30, 0, 0, time.UTC)
	clonedDaemonStatus := cloneMachineDaemonStatus(MachineDaemonStatus{
		Registered:       true,
		LastRegisteredAt: &statusTime,
		CurrentSessionID: &sessionID,
		SessionState:     MachineTransportSessionStateConnected,
	})
	if clonedDaemonStatus.LastRegisteredAt == nil || clonedDaemonStatus.LastRegisteredAt == &statusTime || clonedDaemonStatus.CurrentSessionID == nil || clonedDaemonStatus.CurrentSessionID == &sessionID {
		t.Fatalf("cloneMachineDaemonStatus() = %+v", clonedDaemonStatus)
	}
	clonedCredential := cloneMachineChannelCredential(MachineChannelCredential{
		Kind:          MachineChannelCredentialKindToken,
		TokenID:       &tokenID,
		CertificateID: &certificateID,
	})
	if clonedCredential.TokenID == nil || clonedCredential.TokenID == &tokenID || clonedCredential.CertificateID == nil || clonedCredential.CertificateID == &certificateID {
		t.Fatalf("cloneMachineChannelCredential() = %+v", clonedCredential)
	}

	sshUser := "openase"
	sshKeyPath := "/tmp/id_ed25519"
	if _, err := ParseCreateMachine(uuid.New(), MachineInput{
		Name:             "local",
		Host:             "local",
		ReachabilityMode: "direct_connect",
		ExecutionMode:    "websocket",
		SSHUser:          &sshUser,
		SSHKeyPath:       &sshKeyPath,
	}); err == nil {
		t.Fatal("ParseCreateMachine() expected local host connection_mode validation error")
	}
	if _, err := ParseCreateMachine(uuid.New(), MachineInput{
		Name:             "remote",
		Host:             "10.0.0.10",
		ReachabilityMode: "local",
		ExecutionMode:    "local_process",
	}); err == nil {
		t.Fatal("ParseCreateMachine() expected remote local-mode validation error")
	}
	if createMachine, err := ParseCreateMachine(uuid.New(), MachineInput{
		Name:   "local",
		Host:   "local",
		Status: "online",
	}); err != nil || createMachine.ConnectionMode != MachineConnectionModeLocal {
		t.Fatalf("ParseCreateMachine(local default) = %+v, %v", createMachine, err)
	}
}

func TestCatalogMachineMonitorParsers(t *testing.T) {
	collectedAt := time.Date(2026, 3, 27, 11, 0, 0, 0, time.UTC)
	systemRaw := strings.Join([]string{
		"cpu_cores=8",
		"cpu_usage_percent=65.432",
		"memory_total_kb=8388608",
		"memory_available_kb=2097152",
		"disk_total_kb=20971520",
		"disk_available_kb=10485760",
	}, "\n")
	system, err := ParseMachineSystemResources(systemRaw, collectedAt)
	if err != nil {
		t.Fatalf("ParseMachineSystemResources() error = %v", err)
	}
	if system.CPUCores != 8 || system.MemoryTotalGB != 8 || system.MemoryAvailablePercent != 25 || system.DiskAvailablePercent != 50 {
		t.Fatalf("ParseMachineSystemResources() = %+v", system)
	}
	if _, err := ParseMachineSystemResources(strings.Join([]string{
		"cpu_cores=bad",
		"cpu_usage_percent=65.4",
		"memory_total_kb=8388608",
		"memory_available_kb=2097152",
		"disk_total_kb=20971520",
		"disk_available_kb=10485760",
	}, "\n"), collectedAt); err == nil {
		t.Fatal("ParseMachineSystemResources() expected cpu_cores validation error")
	}
	if _, err := ParseMachineSystemResources(strings.Join([]string{
		"cpu_cores=8",
		"cpu_usage_percent=bad",
		"memory_total_kb=8388608",
		"memory_available_kb=2097152",
		"disk_total_kb=20971520",
		"disk_available_kb=10485760",
	}, "\n"), collectedAt); err == nil {
		t.Fatal("ParseMachineSystemResources() expected cpu usage validation error")
	}
	if _, err := ParseMachineSystemResources(strings.Join([]string{
		"cpu_cores=8",
		"cpu_usage_percent=65.4",
		"memory_total_kb=bad",
		"memory_available_kb=2097152",
		"disk_total_kb=20971520",
		"disk_available_kb=10485760",
	}, "\n"), collectedAt); err == nil {
		t.Fatal("ParseMachineSystemResources() expected memory_total validation error")
	}
	if _, err := ParseMachineSystemResources(strings.Join([]string{
		"cpu_cores=8",
		"cpu_usage_percent=65.4",
		"memory_total_kb=8388608",
		"memory_available_kb=bad",
		"disk_total_kb=20971520",
		"disk_available_kb=10485760",
	}, "\n"), collectedAt); err == nil {
		t.Fatal("ParseMachineSystemResources() expected memory_available validation error")
	}
	if _, err := ParseMachineSystemResources(strings.Join([]string{
		"cpu_cores=8",
		"cpu_usage_percent=65.4",
		"memory_total_kb=8388608",
		"memory_available_kb=2097152",
		"disk_total_kb=bad",
		"disk_available_kb=10485760",
	}, "\n"), collectedAt); err == nil {
		t.Fatal("ParseMachineSystemResources() expected disk_total validation error")
	}
	if _, err := ParseMachineSystemResources(strings.Join([]string{
		"cpu_cores=8",
		"cpu_usage_percent=65.4",
		"memory_total_kb=8388608",
		"memory_available_kb=2097152",
		"disk_total_kb=20971520",
		"disk_available_kb=bad",
	}, "\n"), collectedAt); err == nil {
		t.Fatal("ParseMachineSystemResources() expected disk_available validation error")
	}
	if _, err := ParseMachineSystemResources("bad-line", collectedAt); err == nil {
		t.Fatal("ParseMachineSystemResources() expected metric line validation error")
	}

	gpus, err := ParseMachineGPUResources("0,Tesla T4,16384,8192,50.432\n1,L4,24576,4096,12.34", collectedAt)
	if err != nil {
		t.Fatalf("ParseMachineGPUResources() error = %v", err)
	}
	if !gpus.Available || len(gpus.GPUs) != 2 || gpus.GPUs[0].MemoryTotalGB != 16 || gpus.GPUs[1].MemoryUsedGB != 4 {
		t.Fatalf("ParseMachineGPUResources() = %+v", gpus)
	}
	if blankGPU, err := ParseMachineGPUResources("   ", collectedAt); err != nil || blankGPU.Available {
		t.Fatalf("ParseMachineGPUResources(blank) = %+v, %v", blankGPU, err)
	}
	noGPU, err := ParseMachineGPUResources(" no_gpu ", collectedAt)
	if err != nil || noGPU.Available || noGPU.GPUs != nil {
		t.Fatalf("ParseMachineGPUResources(no_gpu) = %+v, %v", noGPU, err)
	}
	if _, err := ParseMachineGPUResources("0,bad,row", collectedAt); err == nil {
		t.Fatal("ParseMachineGPUResources() expected column validation error")
	}
	if _, err := ParseMachineGPUResources("\"unterminated", collectedAt); err == nil {
		t.Fatal("ParseMachineGPUResources() expected csv parse validation error")
	}
	if _, err := ParseMachineGPUResources("bad,Tesla,1,1,1", collectedAt); err == nil {
		t.Fatal("ParseMachineGPUResources() expected gpu index validation error")
	}
	if _, err := ParseMachineGPUResources("0,Tesla,bad,1,1", collectedAt); err == nil {
		t.Fatal("ParseMachineGPUResources() expected gpu memory total validation error")
	}
	if _, err := ParseMachineGPUResources("0,Tesla,1,bad,1", collectedAt); err == nil {
		t.Fatal("ParseMachineGPUResources() expected gpu memory used validation error")
	}
	if _, err := ParseMachineGPUResources("0,Tesla,1,1,bad", collectedAt); err == nil {
		t.Fatal("ParseMachineGPUResources() expected gpu utilization validation error")
	}

	if _, err := ParseMachineAgentEnvironment("claude_code\tfalse\tunknown\ncodex\ttrue\t1.0\tnot_logged_in\tapi_key\ngemini\ttrue\t1.1\tlogged_in\tlogin", collectedAt); err == nil {
		t.Fatal("ParseMachineAgentEnvironment() expected column count validation error")
	}
	if _, err := ParseMachineAgentEnvironment("", collectedAt); err == nil {
		t.Fatal("ParseMachineAgentEnvironment() expected empty payload validation error")
	}
	if _, err := ParseMachineAgentEnvironment("codex\ttrue\t1.0\tlogged_in\ngemini\ttrue\t1.1\tlogged_in", collectedAt); err == nil {
		t.Fatal("ParseMachineAgentEnvironment() expected missing claude_code validation error")
	}
	if _, err := ParseMachineAgentEnvironment("claude_code\ttrue\t1.0\tlogged_in\ngemini\ttrue\t1.1\tlogged_in", collectedAt); err == nil {
		t.Fatal("ParseMachineAgentEnvironment() expected missing codex validation error")
	}
	if _, err := ParseMachineAgentEnvironment("claude_code\tfalse\t\tunknown\ncodex\ttrue\t1.0\tlogged_in\ngemini\ttrue\t1.1\tlogged_in", collectedAt); err != nil {
		t.Fatalf("ParseMachineAgentEnvironment(4-col) error = %v", err)
	}
	if _, err := ParseMachineAgentEnvironment("\"\"\tfalse\t\tunknown\tunknown\ncodex\ttrue\t1.0\tnot_logged_in\tapi_key\ngemini\ttrue\t1.1\tlogged_in\tlogin", collectedAt); err == nil {
		t.Fatal("ParseMachineAgentEnvironment() expected missing name validation error")
	}
	if _, err := ParseMachineAgentEnvironment("claude_code\tbad\t\tunknown\tunknown\ncodex\ttrue\t1.0\tnot_logged_in\tapi_key\ngemini\ttrue\t1.1\tlogged_in\tlogin", collectedAt); err == nil {
		t.Fatal("ParseMachineAgentEnvironment() expected installed bool validation error")
	}
	if _, err := ParseMachineAgentEnvironment("claude_code\tfalse\t\tunknown\tunknown\ncodex\ttrue\t1.0\tnot_logged_in\tapi_key\ncodex\ttrue\t1.1\tlogged_in\tlogin", collectedAt); err == nil {
		t.Fatal("ParseMachineAgentEnvironment() expected duplicate cli validation error")
	}
	if _, err := ParseMachineAgentEnvironment("claude_code\tfalse\t\tbad\tunknown\ncodex\ttrue\t1.0\tnot_logged_in\tapi_key\ngemini\ttrue\t1.1\tlogged_in\tlogin", collectedAt); err == nil {
		t.Fatal("ParseMachineAgentEnvironment() expected auth status validation error")
	}
	if _, err := ParseMachineAgentEnvironment("claude_code\tfalse\t\tunknown\tunknown\ncodex\ttrue\t1.0\tnot_logged_in\tbad\ngemini\ttrue\t1.1\tlogged_in\tlogin", collectedAt); err == nil {
		t.Fatal("ParseMachineAgentEnvironment() expected auth mode validation error")
	}
	if _, err := ParseMachineAgentEnvironment("claude_code\tfalse\t\tunknown\tunknown\ncodex\ttrue\t1.0\tnot_logged_in\tapi_key", collectedAt); err == nil {
		t.Fatal("ParseMachineAgentEnvironment() expected missing gemini validation error")
	}

	if _, err := ParseMachineFullAudit("git\ttrue\tname\temail\nnetwork\ttrue\tfalse\ttrue", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected missing gh_cli validation error")
	}
	if _, err := ParseMachineFullAudit("", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected empty payload validation error")
	}
	if _, err := ParseMachineFullAudit("git\ttrue\tname\temail\n\ngh_cli\ttrue\tlogged_in\nnetwork\ttrue\tfalse\ttrue", collectedAt); err != nil {
		t.Fatalf("ParseMachineFullAudit(blank line) error = %v", err)
	}
	if _, err := ParseMachineFullAudit("gh_cli\ttrue\tlogged_in\nnetwork\ttrue\tfalse\ttrue", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected missing git entry validation error")
	}
	if _, err := ParseMachineFullAudit("git\ttrue\tname\ngh_cli\ttrue\tlogged_in\nnetwork\ttrue\tfalse\ttrue", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected git row column validation error")
	}
	if _, err := ParseMachineFullAudit("git\tmaybe\tname\temail\ngh_cli\ttrue\tlogged_in\nnetwork\ttrue\tfalse\ttrue", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected git bool validation error")
	}
	if _, err := ParseMachineFullAudit("git\ttrue\tname\temail\ngh_cli\ttrue\nnetwork\ttrue\tfalse\ttrue", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected gh row column validation error")
	}
	if _, err := ParseMachineFullAudit("git\ttrue\tname\temail\ngh_cli\tmaybe\tlogged_in\nnetwork\ttrue\tfalse\ttrue", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected gh bool validation error")
	}
	if _, err := ParseMachineFullAudit("git\ttrue\tname\temail\ngh_cli\ttrue\tbad\nnetwork\ttrue\tfalse\ttrue", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected gh auth validation error")
	}
	if _, err := ParseMachineFullAudit("git\ttrue\tname\temail\ngh_cli\ttrue\tlogged_in\nnetwork\ttrue\tfalse", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected network row column validation error")
	}
	if _, err := ParseMachineFullAudit("git\ttrue\tname\temail\ngh_cli\ttrue\tlogged_in\nnetwork\tbad\tfalse\ttrue", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected github reachability validation error")
	}
	if _, err := ParseMachineFullAudit("git\ttrue\tname\temail\ngh_cli\ttrue\tlogged_in\nnetwork\ttrue\tbad\ttrue", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected network bool validation error")
	}
	if _, err := ParseMachineFullAudit("git\ttrue\tname\temail\ngh_cli\ttrue\tlogged_in\nnetwork\ttrue\tfalse\tbad", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected npm reachability validation error")
	}
	if _, err := ParseMachineFullAudit("unknown\ttrue", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected unknown row validation error")
	}
	if _, err := ParseMachineFullAudit("git\ttrue\tname\temail\ngh_cli\ttrue\tlogged_in", collectedAt); err == nil {
		t.Fatal("ParseMachineFullAudit() expected missing network entry validation error")
	}

	metricValues, err := parseMachineMetricLines("cpu_cores=4\nmemory_total_kb=1024")
	if err != nil {
		t.Fatalf("parseMachineMetricLines() error = %v", err)
	}
	if got, err := parseMetricInt(metricValues, "cpu_cores"); err != nil || got != 4 {
		t.Fatalf("parseMetricInt() = %d, %v; want 4, nil", got, err)
	}
	if got, err := parseMetricFloat(metricValues, "memory_total_kb"); err != nil || got != 1024 {
		t.Fatalf("parseMetricFloat() = %v, %v; want 1024, nil", got, err)
	}
	if _, err := parseMachineMetricLines("bad-line"); err == nil {
		t.Fatal("parseMachineMetricLines() expected format validation error")
	}
	if values, err := parseMachineMetricLines("cpu_cores=4\n \nmemory_total_kb=1"); err != nil || values["cpu_cores"] != "4" || values["memory_total_kb"] != "1" {
		t.Fatalf("parseMachineMetricLines(blank lines) = %+v, %v", values, err)
	}
	if _, err := parseMachineTabularRecords(""); err == nil {
		t.Fatal("parseMachineTabularRecords() expected empty validation error")
	}
	if _, err := parseMachineTabularRecords("\"unterminated"); err == nil {
		t.Fatal("parseMachineTabularRecords() expected csv parse validation error")
	}
	if _, err := parseMetricInt(metricValues, "missing"); err == nil {
		t.Fatal("parseMetricInt() expected missing key validation error")
	}
	if _, err := parseMetricInt(map[string]string{"cpu_cores": "bad"}, "cpu_cores"); err == nil {
		t.Fatal("parseMetricInt() expected parse validation error")
	}
	if _, err := parseMetricFloat(metricValues, "missing"); err == nil {
		t.Fatal("parseMetricFloat() expected missing key validation error")
	}
	if _, err := parseMetricFloat(map[string]string{"memory_total_kb": "bad"}, "memory_total_kb"); err == nil {
		t.Fatal("parseMetricFloat() expected parse validation error")
	}
	if _, err := parseMachineAgentAuthStatus("bad"); err == nil {
		t.Fatal("parseMachineAgentAuthStatus() expected validation error")
	}
	if _, err := parseMachineAgentAuthMode("bad"); err == nil {
		t.Fatal("parseMachineAgentAuthMode() expected validation error")
	}
	if got := kilobytesToGigabytes(1048576); got != 1 {
		t.Fatalf("kilobytesToGigabytes() = %v, want 1", got)
	}
	if got := percentage(1, 0); got != 0 {
		t.Fatalf("percentage() zero total = %v, want 0", got)
	}
	if got := roundTwoDecimals(1.235); got != 1.24 {
		t.Fatalf("roundTwoDecimals() = %v, want 1.24", got)
	}
}

func TestCatalogEnvironmentProvisioningHelpers(t *testing.T) {
	plan := MachineEnvironmentProvisioningPlan{}
	plan.appendIssue(MachineEnvironmentProvisioningIssue{
		Code:      "git_missing",
		SkillName: stringPointer(EnvironmentProvisionerSkillSetupGit),
	})
	plan.appendIssue(MachineEnvironmentProvisioningIssue{
		Code:      "git_missing_duplicate",
		SkillName: stringPointer(EnvironmentProvisionerSkillSetupGit),
	})
	if len(plan.RequiredSkills) != 1 {
		t.Fatalf("appendIssue() required skills = %+v, want one unique skill", plan.RequiredSkills)
	}
	plan.appendNote("same")
	plan.appendNote("same")
	if len(plan.Notes) != 1 {
		t.Fatalf("appendNote() notes = %+v, want one unique note", plan.Notes)
	}

	environment, err := parseStoredAgentEnvironment(map[string]any{
		"agent_environment": map[string]any{
			"claude_code": map[string]any{"installed": false, "auth_status": "unknown"},
			"codex":       map[string]any{"installed": true, "auth_status": "logged_in"},
		},
	})
	if err != nil {
		t.Fatalf("parseStoredAgentEnvironment() error = %v", err)
	}
	if environment.Codex.AuthStatus != MachineAgentAuthStatusLoggedIn {
		t.Fatalf("parseStoredAgentEnvironment() = %+v", environment)
	}
	if _, err := parseStoredAgentEnvironment(map[string]any{}); err == nil {
		t.Fatal("parseStoredAgentEnvironment() expected missing snapshot validation error")
	}
	if _, err := parseStoredCLI(map[string]any{"installed": true, "auth_status": "bad"}); err == nil {
		t.Fatal("parseStoredCLI() expected auth validation error")
	}

	audit, err := parseStoredFullAudit(map[string]any{
		"full_audit": map[string]any{
			"git":     map[string]any{"installed": true, "user_name": "OpenASE", "user_email": "ops@example.com"},
			"gh_cli":  map[string]any{"installed": true, "auth_status": "logged_in"},
			"network": map[string]any{"github_reachable": true, "pypi_reachable": false, "npm_reachable": true},
		},
	})
	if err != nil {
		t.Fatalf("parseStoredFullAudit() error = %v", err)
	}
	if !audit.Network.GitHubReachable || audit.Network.PyPIReachable {
		t.Fatalf("parseStoredFullAudit() = %+v", audit)
	}
	if _, err := parseStoredGitAudit(map[string]any{"installed": "bad"}); err == nil {
		t.Fatal("parseStoredGitAudit() expected bool validation error")
	}
	if _, err := parseStoredGitHubCLIAudit(map[string]any{"installed": true}); err == nil {
		t.Fatal("parseStoredGitHubCLIAudit() expected missing auth status validation error")
	}
	if _, err := parseStoredNetworkAudit(map[string]any{"github_reachable": true}); err == nil {
		t.Fatal("parseStoredNetworkAudit() expected missing field validation error")
	}

	resources := map[string]any{
		"monitor": map[string]any{
			"l1": map[string]any{"reachable": true},
		},
		"last_success": false,
	}
	if !machineIsReachable(resources) {
		t.Fatal("machineIsReachable() expected nested reachability to win")
	}
	if got, ok := nestedObject(map[string]any{"obj": map[string]any{"x": 1}}, "obj"); !ok || got["x"].(int) != 1 {
		t.Fatalf("nestedObject() = %v, %t", got, ok)
	}
	if got, ok := boolField(map[string]any{"reachable": true}, "reachable"); !ok || !got {
		t.Fatalf("boolField() = %v, %t", got, ok)
	}
	if got, ok := stringField(map[string]any{"name": "codex"}, "name"); !ok || got != "codex" {
		t.Fatalf("stringField() = %q, %t", got, ok)
	}
	if got := stringFieldOrEmpty(map[string]any{}, "name"); got != "" {
		t.Fatalf("stringFieldOrEmpty() = %q, want empty", got)
	}
	if got := stringPointer("codex"); got == nil || *got != "codex" {
		t.Fatalf("stringPointer() = %v, want codex", got)
	}
	if got, ok := stringField(map[string]any{"name": 1}, "name"); ok || got != "" {
		t.Fatalf("stringField(non-string) = %q, %t; want empty, false", got, ok)
	}

	description := buildProvisioningTicketDescription(Machine{Name: "builder", Host: "10.0.0.1", Status: MachineStatusOnline}, MachineEnvironmentProvisioningPlan{
		Runnable:       true,
		Issues:         []MachineEnvironmentProvisioningIssue{{Title: "Install Codex", Detail: "Missing", SkillName: stringPointer(EnvironmentProvisionerSkillInstallCodex)}},
		Notes:          []string{"Network unstable"},
		RequiredSkills: []string{EnvironmentProvisionerSkillInstallCodex},
	})
	if !strings.Contains(description, "Install Codex via `install-codex`") || !strings.Contains(description, "Required skills:") {
		t.Fatalf("buildProvisioningTicketDescription() = %q", description)
	}

	templates := BuiltinAgentProviderTemplates()
	if len(templates) != 3 || templates[1].AdapterType != AgentProviderAdapterTypeCodexAppServer || !reflect.DeepEqual(templates[1].CliArgs, []string{"app-server", "--listen", "stdio://"}) {
		t.Fatalf("BuiltinAgentProviderTemplates() = %+v", templates)
	}

	auditPlan := MachineEnvironmentProvisioningPlan{}
	auditPlan.appendAuditIssues(storedFullAudit{
		Git:    storedGitAudit{Installed: false},
		GitHub: storedGitHubCLIAudit{Installed: false},
	})
	if len(auditPlan.Issues) != 2 {
		t.Fatalf("appendAuditIssues() missing install branches = %+v", auditPlan.Issues)
	}
	plan.appendIssue(MachineEnvironmentProvisioningIssue{Code: "note-only"})
	if len(plan.RequiredSkills) != 1 {
		t.Fatalf("appendIssue(nil skill) mutated required skills: %+v", plan.RequiredSkills)
	}
	if _, err := parseStoredAgentEnvironment(map[string]any{"agent_environment": map[string]any{"claude_code": map[string]any{"installed": false, "auth_status": "unknown"}}}); err == nil {
		t.Fatal("parseStoredAgentEnvironment() expected missing codex validation error")
	}
	if _, err := parseStoredAgentEnvironment(map[string]any{"agent_environment": map[string]any{"codex": map[string]any{"installed": true, "auth_status": "logged_in"}}}); err == nil {
		t.Fatal("parseStoredAgentEnvironment() expected missing claude validation error")
	}
	if _, err := parseStoredAgentEnvironment(map[string]any{"agent_environment": map[string]any{"claude_code": map[string]any{"installed": true}, "codex": map[string]any{"installed": true, "auth_status": "logged_in"}}}); err == nil {
		t.Fatal("parseStoredAgentEnvironment() expected claude parse validation error")
	}
	if _, err := parseStoredAgentEnvironment(map[string]any{"agent_environment": map[string]any{"claude_code": map[string]any{"installed": true, "auth_status": "logged_in"}, "codex": map[string]any{"installed": true}}}); err == nil {
		t.Fatal("parseStoredAgentEnvironment() expected codex parse validation error")
	}
	if _, err := parseStoredCLI(map[string]any{"auth_status": "unknown"}); err == nil {
		t.Fatal("parseStoredCLI() expected missing installed validation error")
	}
	if _, err := parseStoredCLI(map[string]any{"installed": true}); err == nil {
		t.Fatal("parseStoredCLI() expected missing auth_status validation error")
	}
	if _, err := parseStoredFullAudit(map[string]any{"full_audit": map[string]any{}}); err == nil {
		t.Fatal("parseStoredFullAudit() expected missing nested objects validation error")
	}
	if _, err := parseStoredFullAudit(map[string]any{}); err == nil {
		t.Fatal("parseStoredFullAudit() expected missing full_audit validation error")
	}
	if _, err := parseStoredFullAudit(map[string]any{"full_audit": map[string]any{
		"git":     map[string]any{"installed": true, "user_name": "OpenASE", "user_email": "ops@example.com"},
		"network": map[string]any{"github_reachable": true, "pypi_reachable": true, "npm_reachable": true},
	}}); err == nil {
		t.Fatal("parseStoredFullAudit() expected missing gh_cli validation error")
	}
	if _, err := parseStoredFullAudit(map[string]any{"full_audit": map[string]any{
		"git":    map[string]any{"installed": true, "user_name": "OpenASE", "user_email": "ops@example.com"},
		"gh_cli": map[string]any{"installed": true, "auth_status": "logged_in"},
	}}); err == nil {
		t.Fatal("parseStoredFullAudit() expected missing network validation error")
	}
	if _, err := parseStoredFullAudit(map[string]any{"full_audit": map[string]any{
		"git":     map[string]any{"installed": "bad"},
		"gh_cli":  map[string]any{"installed": true, "auth_status": "logged_in"},
		"network": map[string]any{"github_reachable": true, "pypi_reachable": true, "npm_reachable": true},
	}}); err == nil {
		t.Fatal("parseStoredFullAudit() expected git parse validation error")
	}
	if _, err := parseStoredFullAudit(map[string]any{"full_audit": map[string]any{
		"git":     map[string]any{"installed": true, "user_name": "OpenASE", "user_email": "ops@example.com"},
		"gh_cli":  map[string]any{"installed": true},
		"network": map[string]any{"github_reachable": true, "pypi_reachable": true, "npm_reachable": true},
	}}); err == nil {
		t.Fatal("parseStoredFullAudit() expected gh parse validation error")
	}
	if _, err := parseStoredFullAudit(map[string]any{"full_audit": map[string]any{
		"git":     map[string]any{"installed": true, "user_name": "OpenASE", "user_email": "ops@example.com"},
		"gh_cli":  map[string]any{"installed": true, "auth_status": "logged_in"},
		"network": map[string]any{"github_reachable": true, "pypi_reachable": true},
	}}); err == nil {
		t.Fatal("parseStoredFullAudit() expected network parse validation error")
	}
	if _, err := parseStoredGitHubCLIAudit(map[string]any{"auth_status": "unknown"}); err == nil {
		t.Fatal("parseStoredGitHubCLIAudit() expected missing installed validation error")
	}
	if _, err := parseStoredGitHubCLIAudit(map[string]any{"installed": true}); err == nil {
		t.Fatal("parseStoredGitHubCLIAudit() expected missing auth_status validation error")
	}
	if _, err := parseStoredGitHubCLIAudit(map[string]any{"installed": true, "auth_status": "bad"}); err == nil {
		t.Fatal("parseStoredGitHubCLIAudit() expected auth_status validation error")
	}
	if _, err := parseStoredNetworkAudit(map[string]any{"github_reachable": "bad", "pypi_reachable": true, "npm_reachable": true}); err == nil {
		t.Fatal("parseStoredNetworkAudit() expected bool validation error")
	}
	if _, err := parseStoredNetworkAudit(map[string]any{"github_reachable": true, "npm_reachable": true}); err == nil {
		t.Fatal("parseStoredNetworkAudit() expected missing pypi validation error")
	}
	if _, err := parseStoredNetworkAudit(map[string]any{"github_reachable": true, "pypi_reachable": true}); err == nil {
		t.Fatal("parseStoredNetworkAudit() expected missing npm validation error")
	}

	healthyPlan := PlanMachineEnvironmentProvisioning(Machine{
		ID:     uuid.New(),
		Name:   "builder-healthy",
		Host:   "10.0.0.9",
		Status: MachineStatusOnline,
		Resources: map[string]any{
			"monitor": map[string]any{"l1": map[string]any{"reachable": true}},
			"agent_environment": map[string]any{
				"claude_code": map[string]any{"installed": true, "auth_status": "logged_in"},
				"codex":       map[string]any{"installed": true, "auth_status": "logged_in"},
			},
			"full_audit": map[string]any{
				"git":     map[string]any{"installed": true, "user_name": "OpenASE", "user_email": "ops@example.com"},
				"gh_cli":  map[string]any{"installed": true, "auth_status": "logged_in"},
				"network": map[string]any{"github_reachable": true, "pypi_reachable": true, "npm_reachable": true},
			},
		},
	})
	if healthyPlan.Needed || !healthyPlan.Available || healthyPlan.Runnable {
		t.Fatalf("PlanMachineEnvironmentProvisioning(healthy) = %+v", healthyPlan)
	}
	localPlan := PlanMachineEnvironmentProvisioning(Machine{
		ID:     uuid.New(),
		Name:   "local",
		Host:   LocalMachineHost,
		Status: MachineStatusOnline,
		Resources: map[string]any{
			"agent_environment": map[string]any{
				"claude_code": map[string]any{"installed": false, "auth_status": "unknown"},
				"codex":       map[string]any{"installed": false, "auth_status": "unknown"},
			},
		},
	})
	if localPlan.Runnable {
		t.Fatalf("PlanMachineEnvironmentProvisioning(local) = %+v, want unrunnable", localPlan)
	}
}

func TestCatalogRuntimeControlAndEnumHelpers(t *testing.T) {
	activeRunID := uuid.New()
	activeTicketID := uuid.New()
	activeAgent := Agent{
		RuntimeControlState: AgentRuntimeControlStateActive,
		Runtime: &AgentRuntime{
			CurrentRunID:    &activeRunID,
			CurrentTicketID: &activeTicketID,
			Status:          AgentStatusRunning,
		},
	}
	state, err := ResolvePauseRuntimeControlState(activeAgent)
	if err != nil || state != AgentRuntimeControlStatePauseRequested {
		t.Fatalf("ResolvePauseRuntimeControlState() = %q, %v; want pause_requested, nil", state, err)
	}
	state, err = ResolveInterruptRuntimeControlState(activeAgent)
	if err != nil || state != AgentRuntimeControlStateInterruptRequested {
		t.Fatalf("ResolveInterruptRuntimeControlState() = %q, %v; want interrupt_requested, nil", state, err)
	}
	if _, err := ResolvePauseRuntimeControlState(Agent{}); err == nil {
		t.Fatal("ResolvePauseRuntimeControlState() expected missing run validation error")
	}
	if _, err := ResolveInterruptRuntimeControlState(Agent{}); err == nil {
		t.Fatal("ResolveInterruptRuntimeControlState() expected missing run validation error")
	}
	if _, err := ResolvePauseRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStatePauseRequested, Runtime: activeAgent.Runtime}); err == nil {
		t.Fatal("ResolvePauseRuntimeControlState() expected in-progress validation error")
	}
	if _, err := ResolvePauseRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStateInterruptRequested, Runtime: activeAgent.Runtime}); err == nil {
		t.Fatal("ResolvePauseRuntimeControlState() expected interrupt-in-progress validation error")
	}
	if _, err := ResolveInterruptRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStateInterruptRequested, Runtime: activeAgent.Runtime}); err == nil {
		t.Fatal("ResolveInterruptRuntimeControlState() expected in-progress validation error")
	}
	if _, err := ResolvePauseRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStatePaused, Runtime: activeAgent.Runtime}); err == nil {
		t.Fatal("ResolvePauseRuntimeControlState() expected paused validation error")
	}
	if _, err := ResolveInterruptRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStatePauseRequested, Runtime: activeAgent.Runtime}); err == nil {
		t.Fatal("ResolveInterruptRuntimeControlState() expected pause-in-progress validation error")
	}
	if _, err := ResolveInterruptRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStatePaused, Runtime: activeAgent.Runtime}); err == nil {
		t.Fatal("ResolveInterruptRuntimeControlState() expected paused validation error")
	}
	if _, err := ResolvePauseRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStateActive, Runtime: &AgentRuntime{CurrentRunID: &activeRunID, CurrentTicketID: &activeTicketID, Status: AgentStatusIdle}}); err == nil {
		t.Fatal("ResolvePauseRuntimeControlState() expected status validation error")
	}
	if _, err := ResolveInterruptRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStateActive, Runtime: &AgentRuntime{CurrentRunID: &activeRunID, CurrentTicketID: &activeTicketID, Status: AgentStatusIdle}}); err == nil {
		t.Fatal("ResolveInterruptRuntimeControlState() expected status validation error")
	}
	if _, err := ResolveInterruptRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStateRetired, Runtime: activeAgent.Runtime}); err == nil {
		t.Fatal("ResolveInterruptRuntimeControlState() expected retired validation error")
	}

	pausedAgent := Agent{
		RuntimeControlState: AgentRuntimeControlStatePaused,
		Runtime: &AgentRuntime{
			CurrentRunID:    &activeRunID,
			CurrentTicketID: &activeTicketID,
			Status:          AgentStatusClaimed,
		},
	}
	state, err = ResolveResumeRuntimeControlState(pausedAgent)
	if err != nil || state != AgentRuntimeControlStateActive {
		t.Fatalf("ResolveResumeRuntimeControlState() = %q, %v; want active, nil", state, err)
	}
	if _, err := ResolveResumeRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStateActive, Runtime: pausedAgent.Runtime}); err == nil {
		t.Fatal("ResolveResumeRuntimeControlState() expected already active validation error")
	}
	if _, err := ResolveResumeRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStatePauseRequested, Runtime: pausedAgent.Runtime}); err == nil {
		t.Fatal("ResolveResumeRuntimeControlState() expected still pausing validation error")
	}
	if _, err := ResolveResumeRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStateInterruptRequested, Runtime: pausedAgent.Runtime}); err == nil {
		t.Fatal("ResolveResumeRuntimeControlState() expected still interrupting validation error")
	}
	if _, err := ResolveResumeRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStatePaused, Runtime: &AgentRuntime{CurrentRunID: &activeRunID, CurrentTicketID: &activeTicketID, Status: AgentStatusIdle}}); err == nil {
		t.Fatal("ResolveResumeRuntimeControlState() expected status validation error")
	}
	if _, err := ResolveResumeRuntimeControlState(Agent{}); err == nil {
		t.Fatal("ResolveResumeRuntimeControlState() expected missing run validation error")
	}

	validityChecks := []struct {
		name      string
		stringer  interface{ String() string }
		isValid   func() bool
		wantValue string
	}{
		{"org", OrganizationStatusActive, OrganizationStatusActive.IsValid, "active"},
		{"project", ProjectStatusCanceled, ProjectStatusCanceled.IsValid, "Canceled"},
		{"machine", MachineStatusOffline, MachineStatusOffline.IsValid, "offline"},
		{"provider_capability", AgentProviderCapabilityStateUnsupported, AgentProviderCapabilityStateUnsupported.IsValid, "unsupported"},
		{"adapter", AgentProviderAdapterTypeCustom, AgentProviderAdapterTypeCustom.IsValid, "custom"},
		{"agent_status", AgentStatusInterrupted, AgentStatusInterrupted.IsValid, "interrupted"},
		{"runtime_phase", AgentRuntimePhaseFailed, AgentRuntimePhaseFailed.IsValid, "failed"},
		{"run_status", AgentRunStatusInterrupted, AgentRunStatusInterrupted.IsValid, "interrupted"},
		{"runtime_control", AgentRuntimeControlStateInterruptRequested, AgentRuntimeControlStateInterruptRequested.IsValid, "interrupt_requested"},
	}
	for _, check := range validityChecks {
		if !check.isValid() || check.stringer.String() != check.wantValue {
			t.Fatalf("%s validity/string mismatch", check.name)
		}
	}
	if OrganizationStatus("bad").IsValid() || ProjectStatus("bad").IsValid() || MachineStatus("bad").IsValid() ||
		AgentProviderCapabilityState("bad").IsValid() || AgentProviderAdapterType("bad").IsValid() || AgentStatus("bad").IsValid() || AgentRuntimePhase("bad").IsValid() ||
		AgentRunStatus("bad").IsValid() || AgentRuntimeControlState("bad").IsValid() {
		t.Fatal("expected invalid enum values to be rejected")
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}
