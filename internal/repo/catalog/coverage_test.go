package catalog

import (
	"errors"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func TestCatalogEnumMappings(t *testing.T) {
	t.Parallel()

	if got := toEntOrganizationStatus(domain.OrganizationStatusActive); got != entorganization.StatusActive {
		t.Fatalf("toEntOrganizationStatus() = %q", got)
	}
	if got := toDomainOrganizationStatus(entorganization.StatusArchived); got != domain.OrganizationStatusArchived {
		t.Fatalf("toDomainOrganizationStatus() = %q", got)
	}
	if got := toEntProjectStatus(domain.ProjectStatusCanceled); got != "Canceled" {
		t.Fatalf("toEntProjectStatus() = %q", got)
	}
	if got := toDomainProjectStatus("In Progress"); got != domain.ProjectStatusInProgress {
		t.Fatalf("toDomainProjectStatus() = %q", got)
	}
	if got := toEntMachineStatus(domain.MachineStatusOnline); got != entmachine.StatusOnline {
		t.Fatalf("toEntMachineStatus() = %q", got)
	}
	if got := toDomainMachineStatus(entmachine.StatusMaintenance); got != domain.MachineStatusMaintenance {
		t.Fatalf("toDomainMachineStatus() = %q", got)
	}
	if got := toEntAgentProviderAdapterType(domain.AgentProviderAdapterTypeClaudeCodeCLI); got != entagentprovider.AdapterTypeClaudeCodeCli {
		t.Fatalf("toEntAgentProviderAdapterType() = %q", got)
	}
	if got := toDomainAgentProviderAdapterType(entagentprovider.AdapterTypeCodexAppServer); got != domain.AgentProviderAdapterTypeCodexAppServer {
		t.Fatalf("toDomainAgentProviderAdapterType() = %q", got)
	}
	if got := toEntAgentRuntimeControlState(domain.AgentRuntimeControlStatePaused); got != entagent.RuntimeControlStatePaused {
		t.Fatalf("toEntAgentRuntimeControlState() = %q", got)
	}
	if got := toDomainAgentRuntimeControlState(entagent.RuntimeControlStateActive); got != domain.AgentRuntimeControlStateActive {
		t.Fatalf("toDomainAgentRuntimeControlState() = %q", got)
	}
	if got := toDomainAgentRunStatus(entagentrun.StatusCompleted); got != domain.AgentRunStatusCompleted {
		t.Fatalf("toDomainAgentRunStatus() = %q", got)
	}
}

func TestCatalogMappingHelpers(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projectID := uuid.New()
	repoID := uuid.New()
	ticketID := uuid.New()
	machineID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	runID := uuid.New()
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	defaultAgentProviderID := uuid.New()
	workflowID := uuid.New()
	stageWorkspaceDirname := "openase"
	pullRequestURL := "https://github.com/PacificStudio/openase/pull/278"
	sshUser := "codex"
	workspaceRoot := "/workspace/openase"
	agentCLIPath := "/usr/bin/codex"
	runtimeStartedAt := now
	lastHeartbeatAt := now.Add(2 * time.Minute)

	org := &ent.Organization{
		ID:                     orgID,
		Name:                   "Better And Better",
		Slug:                   "better-and-better",
		Status:                 entorganization.StatusActive,
		DefaultAgentProviderID: &defaultAgentProviderID,
	}
	project := &ent.Project{
		ID:                     projectID,
		OrganizationID:         orgID,
		Name:                   "OpenASE",
		Slug:                   "openase",
		Description:            "automation",
		Status:                 "In Progress",
		DefaultAgentProviderID: &defaultAgentProviderID,
		AccessibleMachineIds:   []uuid.UUID{machineID},
		MaxConcurrentAgents:    4,
	}
	projectRepo := &ent.ProjectRepo{
		ID:               repoID,
		ProjectID:        projectID,
		Name:             "openase",
		RepositoryURL:    "https://github.com/PacificStudio/openase.git",
		DefaultBranch:    "main",
		WorkspaceDirname: stageWorkspaceDirname,
		Labels:           []string{"backend", "automation"},
	}
	ticketRepoScope := &ent.TicketRepoScope{
		ID:             uuid.New(),
		TicketID:       ticketID,
		RepoID:         repoID,
		BranchName:     "fix/openase-278-coverage",
		PullRequestURL: pullRequestURL,
	}
	machine := &ent.Machine{
		ID:              machineID,
		OrganizationID:  orgID,
		Name:            "worker-1",
		Host:            "worker-1.internal",
		Port:            22,
		SSHUser:         sshUser,
		SSHKeyPath:      "/keys/id_ed25519",
		Description:     "runner",
		Labels:          []string{"linux", "arm64"},
		Status:          entmachine.StatusOnline,
		WorkspaceRoot:   workspaceRoot,
		AgentCliPath:    agentCLIPath,
		EnvVars:         []string{"OPENASE=1"},
		LastHeartbeatAt: &lastHeartbeatAt,
		Resources:       map[string]any{"cpu": "8"},
	}
	agentProvider := &ent.AgentProvider{
		ID:                 providerID,
		OrganizationID:     orgID,
		MachineID:          machineID,
		Name:               "Codex",
		AdapterType:        entagentprovider.AdapterTypeCodexAppServer,
		CliCommand:         "codex",
		CliArgs:            []string{"run"},
		AuthConfig:         map[string]any{"token": "secret"},
		ModelName:          "gpt-5.4",
		ModelTemperature:   0.2,
		ModelMaxTokens:     64000,
		CostPerInputToken:  0.1,
		CostPerOutputToken: 0.2,
		Edges: ent.AgentProviderEdges{
			Machine: machine,
		},
	}
	run := &ent.AgentRun{
		ID:                       runID,
		AgentID:                  agentID,
		WorkflowID:               workflowID,
		TicketID:                 ticketID,
		ProviderID:               providerID,
		Status:                   entagentrun.StatusExecuting,
		SessionID:                "session-1",
		RuntimeStartedAt:         &runtimeStartedAt,
		LastError:                "",
		LastHeartbeatAt:          &lastHeartbeatAt,
		InputTokens:              120,
		OutputTokens:             30,
		CachedInputTokens:        15,
		CacheCreationInputTokens: 9,
		ReasoningTokens:          6,
		PromptTokens:             90,
		CandidateTokens:          24,
		ToolTokens:               12,
		TotalTokens:              150,
		CreatedAt:                now,
	}
	agent := &ent.Agent{
		ID:                    agentID,
		ProviderID:            providerID,
		ProjectID:             projectID,
		Name:                  "Planner",
		RuntimeControlState:   entagent.RuntimeControlStateActive,
		TotalTokensUsed:       1234,
		TotalTicketsCompleted: 7,
	}
	activityAgentID := agentID
	activity := &ent.ActivityEvent{
		ID:        uuid.New(),
		ProjectID: projectID,
		TicketID:  &ticketID,
		AgentID:   &activityAgentID,
		EventType: domain.AgentOutputEventType,
		Message:   "build finished",
		Metadata:  map[string]any{"stream": "stdout"},
		CreatedAt: now,
	}

	mappedOrg := mapOrganization(org)
	if mappedOrg.DefaultAgentProviderID == nil || *mappedOrg.DefaultAgentProviderID != defaultAgentProviderID {
		t.Fatalf("mapOrganization() = %+v", mappedOrg)
	}
	if mapped := mapOrganizations([]*ent.Organization{org}); len(mapped) != 1 || mapped[0].ID != orgID {
		t.Fatalf("mapOrganizations() = %+v", mapped)
	}

	mappedProject := mapProject(project)
	if len(mappedProject.AccessibleMachineIDs) != 1 || mappedProject.AccessibleMachineIDs[0] != machineID {
		t.Fatalf("mapProject() = %+v", mappedProject)
	}
	if mapped := mapProjects([]*ent.Project{project}); len(mapped) != 1 || mapped[0].ID != projectID {
		t.Fatalf("mapProjects() = %+v", mapped)
	}

	mappedProjectRepo := mapProjectRepo(projectRepo)
	if mappedProjectRepo.WorkspaceDirname != stageWorkspaceDirname || len(mappedProjectRepo.Labels) != 2 {
		t.Fatalf("mapProjectRepo() = %+v", mappedProjectRepo)
	}
	if mapped := mapProjectRepos([]*ent.ProjectRepo{projectRepo}); len(mapped) != 1 || mapped[0].ID != repoID {
		t.Fatalf("mapProjectRepos() = %+v", mapped)
	}

	mappedScope := mapTicketRepoScope(ticketRepoScope)
	if mappedScope.PullRequestURL == nil || *mappedScope.PullRequestURL != pullRequestURL {
		t.Fatalf("mapTicketRepoScope() = %+v", mappedScope)
	}
	if mapped := mapTicketRepoScopes([]*ent.TicketRepoScope{ticketRepoScope}); len(mapped) != 1 || mapped[0].ID != ticketRepoScope.ID {
		t.Fatalf("mapTicketRepoScopes() = %+v", mapped)
	}

	mappedMachine := mapMachine(machine)
	if mappedMachine.SSHUser == nil || *mappedMachine.SSHUser != sshUser || mappedMachine.LastHeartbeatAt == nil || mappedMachine.LastHeartbeatAt.Location() != time.UTC {
		t.Fatalf("mapMachine() = %+v", mappedMachine)
	}
	if mapped := mapMachines([]*ent.Machine{machine}); len(mapped) != 1 || mapped[0].ID != machineID {
		t.Fatalf("mapMachines() = %+v", mapped)
	}

	mappedProvider := mapAgentProvider(agentProvider)
	if mappedProvider.MachineName != machine.Name || mappedProvider.MachineWorkspaceRoot == nil || *mappedProvider.MachineWorkspaceRoot != workspaceRoot {
		t.Fatalf("mapAgentProvider() = %+v", mappedProvider)
	}
	if mapped := mapAgentProviders([]*ent.AgentProvider{agentProvider}); len(mapped) != 1 || mapped[0].ID != providerID {
		t.Fatalf("mapAgentProviders() = %+v", mapped)
	}

	snapshot := agentCurrentRunSummary{runs: []*ent.AgentRun{run}}
	mappedAgent := mapAgent(agent, snapshot)
	if mappedAgent.Runtime == nil || mappedAgent.Runtime.CurrentRunID == nil || *mappedAgent.Runtime.CurrentRunID != runID {
		t.Fatalf("mapAgent() = %+v", mappedAgent)
	}
	if mapped := mapAgents([]*ent.Agent{agent}, map[uuid.UUID]agentCurrentRunSummary{agentID: snapshot}); len(mapped) != 1 || mapped[0].ID != agentID {
		t.Fatalf("mapAgents() = %+v", mapped)
	}
	mappedRun := mapAgentRun(run)
	if mappedRun.RuntimeStartedAt == nil || mappedRun.RuntimeStartedAt.Location() != time.UTC || mappedRun.CreatedAt.Location() != time.UTC {
		t.Fatalf("mapAgentRun() = %+v", mappedRun)
	}
	if mappedRun.CacheCreationInputTokens != 9 ||
		mappedRun.PromptTokens != 90 ||
		mappedRun.CandidateTokens != 24 ||
		mappedRun.ToolTokens != 12 ||
		mappedRun.TotalTokens != 150 {
		t.Fatalf("mapAgentRun() usage fields = %+v", mappedRun)
	}
	if mapped := mapAgentRuns([]*ent.AgentRun{run}); len(mapped) != 1 || mapped[0].ID != runID {
		t.Fatalf("mapAgentRuns() = %+v", mapped)
	}
	if mapped := mapAgentRunList([]*ent.AgentRun{run}); len(mapped) != 1 || mapped[0].ID != runID {
		t.Fatalf("mapAgentRunList() = %+v", mapped)
	}
	if mapped := mapAgentRunList([]*ent.AgentRun{nil}); len(mapped) != 0 {
		t.Fatalf("mapAgentRunList(nil entry) = %+v, want empty", mapped)
	}

	mappedActivity := mapActivityEvent(activity)
	if mappedActivity.AgentID == nil || *mappedActivity.AgentID != activityAgentID || mappedActivity.CreatedAt.Location() != time.UTC {
		t.Fatalf("mapActivityEvent() = %+v", mappedActivity)
	}
	if mapped := mapActivityEvents([]*ent.ActivityEvent{activity}); len(mapped) != 1 || mapped[0].ID != activity.ID {
		t.Fatalf("mapActivityEvents() = %+v", mapped)
	}
	traceEvent := &ent.AgentTraceEvent{
		ID:         activity.ID,
		ProjectID:  projectID,
		TicketID:   ticketID,
		AgentID:    activityAgentID,
		AgentRunID: runID,
		Kind:       domain.AgentTraceKindCommandDelta,
		Stream:     "stdout",
		Text:       "output",
		CreatedAt:  now,
	}
	mappedOutput := mapAgentOutputEntry(traceEvent)
	if mappedOutput.AgentID != activityAgentID || mappedOutput.Stream != "stdout" || mappedOutput.CreatedAt.Location() != time.UTC {
		t.Fatalf("mapAgentOutputEntry(trace) = %+v", mappedOutput)
	}
	if mapped := mapAgentOutputEntries([]*ent.AgentTraceEvent{traceEvent}); len(mapped) != 1 || mapped[0].ID != activity.ID {
		t.Fatalf("mapAgentOutputEntries() = %+v", mapped)
	}

	if got := cloneAnyMap(map[string]any{"a": 1}); got["a"] != 1 {
		t.Fatalf("cloneAnyMap() = %+v", got)
	}
	if got := cloneAnyMap(nil); len(got) != 0 {
		t.Fatalf("cloneAnyMap(nil) = %+v", got)
	}
	if got := cloneTimePointer(&runtimeStartedAt); got == nil || got.Location() != time.UTC {
		t.Fatalf("cloneTimePointer() = %+v", got)
	}
	if got := cloneTimePointer(nil); got != nil {
		t.Fatalf("cloneTimePointer(nil) = %+v, want nil", got)
	}
	if got := cloneActivityCreatedAt(now); got.Location() != time.UTC {
		t.Fatalf("cloneActivityCreatedAt() = %+v", got)
	}
	if got := optionalString(" value "); got == nil || *got != " value " {
		t.Fatalf("optionalString() = %+v", got)
	}
	if got := optionalString(""); got != nil {
		t.Fatalf("optionalString(empty) = %+v, want nil", got)
	}
}

func TestCatalogErrorMappingHelpers(t *testing.T) {
	t.Parallel()

	plain := errors.New("plain")
	if got := mapReadError("read", plain); got.Error() != "read: plain" {
		t.Fatalf("mapReadError() = %v", got)
	}
	if got := mapWriteError("write", plain); got.Error() != "write: plain" {
		t.Fatalf("mapWriteError() = %v", got)
	}
	if got := NewEntRepository(&ent.Client{}); got == nil || got.client == nil {
		t.Fatalf("NewEntRepository() = %+v", got)
	}
	if predicates := activeOrganizationPredicates(uuid.New()); len(predicates) != 2 {
		t.Fatalf("activeOrganizationPredicates() len = %d, want 2", len(predicates))
	}
}

var _ = entprojectrepo.FieldID
