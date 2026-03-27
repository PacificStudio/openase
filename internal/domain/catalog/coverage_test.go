package catalog

import (
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
		CostPerInputToken:  &costPerInput,
		CostPerOutputToken: &costPerOutput,
	})
	if err != nil {
		t.Fatalf("ParseCreateAgentProvider() error = %v", err)
	}
	if createProvider.MachineID != machineID || createProvider.Name != "Codex" || createProvider.AdapterType != AgentProviderAdapterTypeCodexAppServer {
		t.Fatalf("ParseCreateAgentProvider() = %+v", createProvider)
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
	if updateProvider.ModelMaxTokens != DefaultAgentProviderModelMaxTokens || updateProvider.CostPerInputToken != DefaultAgentProviderCostPerInputToken || updateProvider.CliArgs != nil {
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

	runtime := BuildAgentRuntime(&AgentRun{
		ID:               runID,
		TicketID:         ticketID,
		Status:           AgentRunStatusExecuting,
		SessionID:        "sess-1",
		RuntimeStartedAt: &now,
		LastError:        "boom",
		LastHeartbeatAt:  &now,
	}, AgentRuntimeControlStateActive)
	if runtime == nil || runtime.Status != AgentStatusRunning || runtime.RuntimePhase != AgentRuntimePhaseExecuting || runtime.CurrentRunID == nil || *runtime.CurrentRunID != runID {
		t.Fatalf("BuildAgentRuntime() executing = %+v", runtime)
	}
	if runtime.RuntimeStartedAt == &now || runtime.LastHeartbeatAt == &now {
		t.Fatal("BuildAgentRuntime() did not clone time pointers")
	}
	if got := BuildAgentRuntime(&AgentRun{Status: AgentRunStatusTerminated}, AgentRuntimeControlStatePaused); got.Status != AgentStatusPaused {
		t.Fatalf("BuildAgentRuntime() paused terminated status = %q, want %q", got.Status, AgentStatusPaused)
	}
	if got := BuildAgentRuntime(&AgentRun{Status: AgentRunStatusReady}, AgentRuntimeControlStateActive); got.RuntimePhase != AgentRuntimePhaseReady {
		t.Fatalf("BuildAgentRuntime() ready phase = %q, want %q", got.RuntimePhase, AgentRuntimePhaseReady)
	}
	if got := BuildAgentRuntime(&AgentRun{Status: AgentRunStatusLaunching}, AgentRuntimeControlStateActive); got.Status != AgentStatusClaimed {
		t.Fatalf("BuildAgentRuntime() launching status = %q, want claimed", got.Status)
	}
	if got := BuildAgentRuntime(&AgentRun{Status: AgentRunStatusErrored}, AgentRuntimeControlStateActive); got.Status != AgentStatusFailed {
		t.Fatalf("BuildAgentRuntime() errored status = %q, want failed", got.Status)
	}
	if got := BuildAgentRuntime(&AgentRun{Status: AgentRunStatusCompleted}, AgentRuntimeControlStateActive); got.Status != DefaultAgentStatus {
		t.Fatalf("BuildAgentRuntime() completed status = %q, want idle", got.Status)
	}
	if BuildAgentRuntime(nil, AgentRuntimeControlStateActive) != nil {
		t.Fatal("BuildAgentRuntime(nil) expected nil")
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
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", CliArgs: []string{" "}, ModelName: "gpt-5.4"},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", ModelName: " "},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", ModelName: "gpt-5.4", ModelTemperature: floatPtr(-1)},
		{MachineID: machineID.String(), Name: "Codex", AdapterType: "codex-app-server", ModelName: "gpt-5.4", ModelMaxTokens: intPtr(0)},
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

func TestCatalogEntityParsersAndHelpers(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	repoID := uuid.New()
	ticketID := uuid.New()
	workflowID := uuid.New()
	defaultProviderID := uuid.New()
	accessibleA := uuid.New()
	accessibleB := uuid.New()
	maxConcurrent := 7
	clonePath := " /srv/repo "
	prURL := " https://github.com/GrandCX/openase/pull/1 "
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
		Status:                 " active ",
		DefaultWorkflowID:      stringPtr(workflowID.String()),
		DefaultAgentProviderID: stringPtr(defaultProviderID.String()),
		AccessibleMachineIDs:   []string{accessibleA.String(), accessibleA.String(), accessibleB.String()},
		MaxConcurrentAgents:    &maxConcurrent,
	})
	if err != nil {
		t.Fatalf("ParseCreateProject() error = %v", err)
	}
	if createProject.Description != "Raise backend coverage" || createProject.Status != ProjectStatusActive || len(createProject.AccessibleMachineIDs) != 2 {
		t.Fatalf("ParseCreateProject() = %+v", createProject)
	}
	updateProject, err := ParseUpdateProject(uuid.New(), orgID, ProjectInput{Name: "P", Slug: "p"})
	if err != nil {
		t.Fatalf("ParseUpdateProject() error = %v", err)
	}
	if updateProject.Status != DefaultProjectStatus || updateProject.MaxConcurrentAgents != DefaultProjectMaxConcurrentAgents {
		t.Fatalf("ParseUpdateProject() defaults = %+v", updateProject)
	}
	updateProject, err = ParseUpdateProject(uuid.New(), orgID, ProjectInput{Name: "Project", Slug: "project", Status: ProjectStatusActive.String()})
	if err != nil {
		t.Fatalf("ParseUpdateProject(success) error = %v", err)
	}
	if updateProject.Status != ProjectStatusActive {
		t.Fatalf("ParseUpdateProject(success) = %+v", updateProject)
	}
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: "P", Slug: "p", DefaultWorkflowID: stringPtr("bad")}); err == nil {
		t.Fatal("ParseCreateProject() expected workflow validation error")
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
	if _, err := ParseCreateProject(orgID, ProjectInput{Name: "P", Slug: "p", MaxConcurrentAgents: intPtr(0)}); err == nil {
		t.Fatal("ParseCreateProject() expected max_concurrent_agents validation error")
	}

	createRepo, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{
		Name:          " OpenASE ",
		RepositoryURL: " https://github.com/GrandCX/openase ",
		DefaultBranch: " trunk ",
		ClonePath:     &clonePath,
		IsPrimary:     boolPtr(true),
		Labels:        []string{" backend ", "backend", " coverage "},
	})
	if err != nil {
		t.Fatalf("ParseCreateProjectRepo() error = %v", err)
	}
	if createRepo.DefaultBranch != "trunk" || createRepo.ClonePath == nil || *createRepo.ClonePath != "/srv/repo" || len(createRepo.Labels) != 2 {
		t.Fatalf("ParseCreateProjectRepo() = %+v", createRepo)
	}
	updateRepo, err := ParseUpdateProjectRepo(uuid.New(), projectID, ProjectRepoInput{
		Name:          "OpenASE",
		RepositoryURL: "ssh://repo",
	})
	if err != nil {
		t.Fatalf("ParseUpdateProjectRepo() error = %v", err)
	}
	if updateRepo.DefaultBranch != "main" || updateRepo.IsPrimary {
		t.Fatalf("ParseUpdateProjectRepo() defaults = %+v", updateRepo)
	}
	updateRepo, err = ParseUpdateProjectRepo(uuid.New(), projectID, ProjectRepoInput{Name: "Repo", RepositoryURL: "https://github.com/GrandCX/openase", IsPrimary: boolPtr(true)})
	if err != nil {
		t.Fatalf("ParseUpdateProjectRepo(success) error = %v", err)
	}
	if !updateRepo.IsPrimary {
		t.Fatalf("ParseUpdateProjectRepo(success) = %+v", updateRepo)
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: " ", RepositoryURL: "https://github.com"}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected name validation error")
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: "repo", RepositoryURL: " "}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected repository_url validation error")
	}
	if _, err := ParseCreateProjectRepo(projectID, ProjectRepoInput{Name: "repo", RepositoryURL: "https://github.com", Labels: []string{""}}); err == nil {
		t.Fatal("ParseCreateProjectRepo() expected labels validation error")
	}
	if _, err := ParseUpdateProjectRepo(uuid.New(), projectID, ProjectRepoInput{Name: " "}); err == nil {
		t.Fatal("ParseUpdateProjectRepo() expected validation error")
	}
	if _, err := ParseUpdateProject(uuid.New(), orgID, ProjectInput{Name: " ", Slug: "project"}); err == nil {
		t.Fatal("ParseUpdateProject() expected validation error")
	}

	createScope, err := ParseCreateTicketRepoScope(projectID, ticketID, TicketRepoScopeInput{
		RepoID:         repoID.String(),
		BranchName:     &branchName,
		PullRequestURL: &prURL,
		PrStatus:       " open ",
		CiStatus:       " passing ",
		IsPrimaryScope: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("ParseCreateTicketRepoScope() error = %v", err)
	}
	if createScope.BranchName == nil || *createScope.BranchName != "feat/openase-278-coverage" || createScope.PullRequestURL == nil || *createScope.PullRequestURL != "https://github.com/GrandCX/openase/pull/1" {
		t.Fatalf("ParseCreateTicketRepoScope() = %+v", createScope)
	}
	updateScope, err := ParseUpdateTicketRepoScope(uuid.New(), projectID, ticketID, TicketRepoScopeInput{RepoID: repoID.String()})
	if err != nil {
		t.Fatalf("ParseUpdateTicketRepoScope() error = %v", err)
	}
	if updateScope.PrStatus != DefaultTicketRepoScopePRStatus || updateScope.CiStatus != DefaultTicketRepoScopeCIStatus || updateScope.IsPrimaryScope {
		t.Fatalf("ParseUpdateTicketRepoScope() defaults = %+v", updateScope)
	}
	updateScope, err = ParseUpdateTicketRepoScope(uuid.New(), projectID, ticketID, TicketRepoScopeInput{RepoID: repoID.String(), IsPrimaryScope: boolPtr(true)})
	if err != nil {
		t.Fatalf("ParseUpdateTicketRepoScope(success) error = %v", err)
	}
	if !updateScope.IsPrimaryScope {
		t.Fatalf("ParseUpdateTicketRepoScope(success) = %+v", updateScope)
	}
	if _, err := ParseCreateTicketRepoScope(projectID, ticketID, TicketRepoScopeInput{RepoID: "bad"}); err == nil {
		t.Fatal("ParseCreateTicketRepoScope() expected repo_id validation error")
	}
	if _, err := ParseCreateTicketRepoScope(projectID, ticketID, TicketRepoScopeInput{RepoID: repoID.String(), PrStatus: "bad"}); err == nil {
		t.Fatal("ParseCreateTicketRepoScope() expected pr_status validation error")
	}
	if _, err := ParseCreateTicketRepoScope(projectID, ticketID, TicketRepoScopeInput{RepoID: repoID.String(), CiStatus: "bad"}); err == nil {
		t.Fatal("ParseCreateTicketRepoScope() expected ci_status validation error")
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
	if _, err := parseMaxConcurrentAgents(intPtr(0)); err == nil {
		t.Fatal("parseMaxConcurrentAgents() expected validation error")
	}
	if _, err := parseTicketRepoScopePrStatus("invalid"); err == nil {
		t.Fatal("parseTicketRepoScopePrStatus() expected validation error")
	}
	if _, err := parseTicketRepoScopeCiStatus("invalid"); err == nil {
		t.Fatal("parseTicketRepoScopeCiStatus() expected validation error")
	}
}

func TestCatalogMachineParsers(t *testing.T) {
	orgID := uuid.New()
	port := 2222
	sshUser := " codex "
	sshKeyPath := " /home/codex/.ssh/id_ed25519 "
	workspaceRoot := " /srv/openase "
	agentCLIPath := " /usr/local/bin/codex "

	createMachine, err := ParseCreateMachine(orgID, MachineInput{
		Name:          " Builder 01 ",
		Host:          " 10.0.1.8 ",
		Port:          &port,
		SSHUser:       &sshUser,
		SSHKeyPath:    &sshKeyPath,
		Description:   " Primary builder ",
		Labels:        []string{" linux ", "linux", " gpu "},
		Status:        " online ",
		WorkspaceRoot: &workspaceRoot,
		AgentCLIPath:  &agentCLIPath,
		EnvVars:       []string{"OPENASE_ENV=prod", " OPENASE_ENV=prod ", "LOG_LEVEL=debug"},
	})
	if err != nil {
		t.Fatalf("ParseCreateMachine() error = %v", err)
	}
	if createMachine.Port != 2222 || len(createMachine.Labels) != 2 || len(createMachine.EnvVars) != 2 {
		t.Fatalf("ParseCreateMachine() = %+v", createMachine)
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

	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "local"}); err == nil {
		t.Fatal("ParseCreateMachine() expected local host/name mismatch error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "local", Host: "remote"}); err == nil {
		t.Fatal("ParseCreateMachine() expected local name/host mismatch error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1"}); err == nil {
		t.Fatal("ParseCreateMachine() expected remote ssh validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1", SSHUser: &sshUser}); err == nil {
		t.Fatal("ParseCreateMachine() expected ssh_key_path validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: " ", Host: "10.0.0.1"}); err == nil {
		t.Fatal("ParseCreateMachine() expected name validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: " "}); err == nil {
		t.Fatal("ParseCreateMachine() expected host validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1", Port: intPtr(70000), SSHUser: &sshUser, SSHKeyPath: &sshKeyPath}); err == nil {
		t.Fatal("ParseCreateMachine() expected port validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1", SSHUser: &sshUser, SSHKeyPath: &sshKeyPath, Labels: []string{""}}); err == nil {
		t.Fatal("ParseCreateMachine() expected labels validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1", SSHUser: &sshUser, SSHKeyPath: &sshKeyPath, Status: "bad"}); err == nil {
		t.Fatal("ParseCreateMachine() expected status validation error")
	}
	if _, err := ParseCreateMachine(orgID, MachineInput{Name: "remote", Host: "10.0.0.1", SSHUser: &sshUser, SSHKeyPath: &sshKeyPath, EnvVars: []string{"NOPE"}}); err == nil {
		t.Fatal("ParseCreateMachine() expected env_vars validation error")
	}
	if _, err := ParseUpdateMachine(uuid.New(), orgID, MachineInput{Name: "remote", Host: "10.0.0.1"}); err == nil {
		t.Fatal("ParseUpdateMachine() expected validation error")
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
	if _, err := ResolvePauseRuntimeControlState(Agent{}); err == nil {
		t.Fatal("ResolvePauseRuntimeControlState() expected missing run validation error")
	}
	if _, err := ResolvePauseRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStatePauseRequested, Runtime: activeAgent.Runtime}); err == nil {
		t.Fatal("ResolvePauseRuntimeControlState() expected in-progress validation error")
	}
	if _, err := ResolvePauseRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStatePaused, Runtime: activeAgent.Runtime}); err == nil {
		t.Fatal("ResolvePauseRuntimeControlState() expected paused validation error")
	}
	if _, err := ResolvePauseRuntimeControlState(Agent{RuntimeControlState: AgentRuntimeControlStateActive, Runtime: &AgentRuntime{CurrentRunID: &activeRunID, CurrentTicketID: &activeTicketID, Status: AgentStatusIdle}}); err == nil {
		t.Fatal("ResolvePauseRuntimeControlState() expected status validation error")
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
		{"org", OrganizationStatusActive, func() bool { return OrganizationStatusActive.IsValid() }, "active"},
		{"project", ProjectStatusPaused, func() bool { return ProjectStatusPaused.IsValid() }, "paused"},
		{"repo_pr", TicketRepoScopePRStatusMerged, func() bool { return TicketRepoScopePRStatusMerged.IsValid() }, "merged"},
		{"repo_ci", TicketRepoScopeCIStatusPassing, func() bool { return TicketRepoScopeCIStatusPassing.IsValid() }, "passing"},
		{"machine", MachineStatusOffline, func() bool { return MachineStatusOffline.IsValid() }, "offline"},
		{"adapter", AgentProviderAdapterTypeCustom, func() bool { return AgentProviderAdapterTypeCustom.IsValid() }, "custom"},
		{"agent_status", AgentStatusPaused, func() bool { return AgentStatusPaused.IsValid() }, "paused"},
		{"runtime_phase", AgentRuntimePhaseFailed, func() bool { return AgentRuntimePhaseFailed.IsValid() }, "failed"},
		{"run_status", AgentRunStatusCompleted, func() bool { return AgentRunStatusCompleted.IsValid() }, "completed"},
		{"runtime_control", AgentRuntimeControlStatePaused, func() bool { return AgentRuntimeControlStatePaused.IsValid() }, "paused"},
	}
	for _, check := range validityChecks {
		if !check.isValid() || check.stringer.String() != check.wantValue {
			t.Fatalf("%s validity/string mismatch", check.name)
		}
	}
	if OrganizationStatus("bad").IsValid() || ProjectStatus("bad").IsValid() || TicketRepoScopePRStatus("bad").IsValid() ||
		TicketRepoScopeCIStatus("bad").IsValid() || MachineStatus("bad").IsValid() || AgentProviderAdapterType("bad").IsValid() ||
		AgentStatus("bad").IsValid() || AgentRuntimePhase("bad").IsValid() || AgentRunStatus("bad").IsValid() ||
		AgentRuntimeControlState("bad").IsValid() {
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
