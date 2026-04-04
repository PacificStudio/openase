package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entagenttraceevent "github.com/BetterAndBetterII/openase/ent/agenttraceevent"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func TestEntRepositoryMachineAgentProviderAndActivityLifecycle(t *testing.T) {
	client := openRepoCatalogTestEntClient(t)
	ctx := context.Background()
	repo := NewEntRepository(client)

	org, err := repo.CreateOrganization(ctx, domain.CreateOrganization{
		Name: "Better And Better",
		Slug: "better-and-better",
	})
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}
	if _, err := repo.ListMachines(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListMachines(missing org) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.ListAgentProviders(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListAgentProviders(missing org) error = %v, want %v", err, ErrNotFound)
	}

	initialMachines, err := repo.ListMachines(ctx, org.ID)
	if err != nil {
		t.Fatalf("ListMachines() initial error = %v", err)
	}
	if len(initialMachines) != 1 || initialMachines[0].Name != domain.LocalMachineName {
		t.Fatalf("ListMachines() initial = %+v", initialMachines)
	}
	localMachineID := initialMachines[0].ID

	remoteMachine, err := repo.CreateMachine(ctx, domain.CreateMachine{
		OrganizationID: org.ID,
		Name:           "builder",
		Host:           "10.0.0.10",
		Port:           2222,
		SSHUser:        strPtr("openase"),
		SSHKeyPath:     strPtr("/tmp/id_ed25519"),
		Description:    "builder host",
		Labels:         []string{"linux", "amd64"},
		Status:         domain.MachineStatusMaintenance,
		WorkspaceRoot:  strPtr("/srv/openase"),
		AgentCLIPath:   strPtr("/usr/local/bin/codex"),
		EnvVars:        []string{"OPENASE_ENV=dev"},
	})
	if err != nil {
		t.Fatalf("CreateMachine() error = %v", err)
	}
	if remoteMachine.Name != "builder" || remoteMachine.SSHUser == nil || *remoteMachine.SSHUser != "openase" {
		t.Fatalf("CreateMachine() = %+v", remoteMachine)
	}
	gotMachine, err := repo.GetMachine(ctx, remoteMachine.ID)
	if err != nil {
		t.Fatalf("GetMachine() error = %v", err)
	}
	if gotMachine.Host != "10.0.0.10" || gotMachine.Port != 2222 {
		t.Fatalf("GetMachine() = %+v", gotMachine)
	}
	if _, err := repo.GetMachine(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetMachine(missing) error = %v, want %v", err, ErrNotFound)
	}

	updatedMachine, err := repo.UpdateMachine(ctx, domain.UpdateMachine{
		ID:             remoteMachine.ID,
		OrganizationID: org.ID,
		Name:           "builder-2",
		Host:           "10.0.0.11",
		Port:           22,
		SSHUser:        strPtr("runner"),
		SSHKeyPath:     strPtr("/tmp/id_rsa"),
		Description:    "updated builder host",
		Labels:         []string{"linux"},
		Status:         domain.MachineStatusOnline,
		WorkspaceRoot:  strPtr("/work/openase"),
		AgentCLIPath:   strPtr("/opt/codex"),
		EnvVars:        []string{"OPENASE_ENV=ci"},
	})
	if err != nil {
		t.Fatalf("UpdateMachine() error = %v", err)
	}
	if updatedMachine.Name != "builder-2" || updatedMachine.Host != "10.0.0.11" || updatedMachine.Status != domain.MachineStatusOnline {
		t.Fatalf("UpdateMachine() = %+v", updatedMachine)
	}
	if _, err := repo.UpdateMachine(ctx, domain.UpdateMachine{
		ID:             uuid.New(),
		OrganizationID: org.ID,
		Name:           "missing",
		Host:           "10.0.0.99",
		Port:           22,
		Description:    "missing machine",
		Status:         domain.MachineStatusOffline,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateMachine(missing) error = %v, want %v", err, ErrNotFound)
	}

	probeTime := time.Date(2026, 3, 27, 14, 0, 0, 0, time.UTC)
	if err := repo.RecordMachineProbe(ctx, domain.RecordMachineProbe{
		ID:              remoteMachine.ID,
		Status:          domain.MachineStatusDegraded,
		LastHeartbeatAt: probeTime,
		Resources: map[string]any{
			"cpu":  "80%",
			"disk": "60%",
		},
	}); err != nil {
		t.Fatalf("RecordMachineProbe() error = %v", err)
	}
	probedMachine, err := repo.GetMachine(ctx, remoteMachine.ID)
	if err != nil {
		t.Fatalf("GetMachine() after probe error = %v", err)
	}
	if probedMachine.Status != domain.MachineStatusDegraded || probedMachine.LastHeartbeatAt == nil || !probedMachine.LastHeartbeatAt.Equal(probeTime) {
		t.Fatalf("GetMachine() after probe = %+v", probedMachine)
	}
	if err := repo.RecordMachineProbe(ctx, domain.RecordMachineProbe{
		ID:              uuid.New(),
		Status:          domain.MachineStatusOffline,
		LastHeartbeatAt: probeTime,
		Resources:       map[string]any{"cpu": "0%"},
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("RecordMachineProbe(missing) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.DeleteMachine(ctx, localMachineID); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("DeleteMachine(local) error = %v, want %v", err, ErrInvalidInput)
	}

	removableMachine, err := repo.CreateMachine(ctx, domain.CreateMachine{
		OrganizationID: org.ID,
		Name:           "scratch",
		Host:           "10.0.0.12",
		Port:           22,
		SSHUser:        strPtr("openase"),
		SSHKeyPath:     strPtr("/tmp/id_remove"),
		Description:    "removable host",
		Status:         domain.MachineStatusMaintenance,
	})
	if err != nil {
		t.Fatalf("CreateMachine() removable error = %v", err)
	}
	deletedMachine, err := repo.DeleteMachine(ctx, removableMachine.ID)
	if err != nil {
		t.Fatalf("DeleteMachine() error = %v", err)
	}
	if deletedMachine.ID != removableMachine.ID {
		t.Fatalf("DeleteMachine() = %+v", deletedMachine)
	}
	if _, err := repo.DeleteMachine(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteMachine(missing) error = %v, want %v", err, ErrNotFound)
	}

	agentProvider, err := repo.CreateAgentProvider(ctx, domain.CreateAgentProvider{
		OrganizationID:     org.ID,
		MachineID:          remoteMachine.ID,
		Name:               "Codex Primary",
		AdapterType:        domain.AgentProviderAdapterTypeCodexAppServer,
		CliCommand:         "codex",
		CliArgs:            []string{"app-server", "--listen", "stdio://"},
		AuthConfig:         map[string]any{"token_env": openAIAPIKeyEnv},
		ModelName:          "gpt-5.4",
		ModelTemperature:   0.1,
		ModelMaxTokens:     20000,
		CostPerInputToken:  0.01,
		CostPerOutputToken: 0.02,
	})
	if err != nil {
		t.Fatalf("CreateAgentProvider() error = %v", err)
	}
	if agentProvider.Name != "Codex Primary" || agentProvider.MachineName != "builder-2" {
		t.Fatalf("CreateAgentProvider() = %+v", agentProvider)
	}
	if _, err := repo.CreateAgentProvider(ctx, domain.CreateAgentProvider{
		OrganizationID:     uuid.New(),
		MachineID:          remoteMachine.ID,
		Name:               "Missing Org Provider",
		AdapterType:        domain.AgentProviderAdapterTypeCodexAppServer,
		CliCommand:         "codex",
		AuthConfig:         map[string]any{},
		ModelName:          "gpt-5.4",
		CostPerInputToken:  0,
		CostPerOutputToken: 0,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("CreateAgentProvider(missing org) error = %v, want %v", err, ErrNotFound)
	}

	gotProvider, err := repo.GetAgentProvider(ctx, agentProvider.ID)
	if err != nil {
		t.Fatalf("GetAgentProvider() error = %v", err)
	}
	if gotProvider.ModelName != "gpt-5.4" || gotProvider.MachineHost != "10.0.0.11" {
		t.Fatalf("GetAgentProvider() = %+v", gotProvider)
	}
	if _, err := repo.GetAgentProvider(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetAgentProvider(missing) error = %v, want %v", err, ErrNotFound)
	}

	updatedProvider, err := repo.UpdateAgentProvider(ctx, domain.UpdateAgentProvider{
		ID:                 agentProvider.ID,
		OrganizationID:     org.ID,
		MachineID:          remoteMachine.ID,
		Name:               "Codex Updated",
		AdapterType:        domain.AgentProviderAdapterTypeCodexAppServer,
		CliCommand:         "codex",
		CliArgs:            []string{"app-server"},
		AuthConfig:         map[string]any{"token_env": openAIAPIKeyEnv},
		ModelName:          "gpt-5.4-mini",
		ModelTemperature:   0.2,
		ModelMaxTokens:     16000,
		CostPerInputToken:  0.005,
		CostPerOutputToken: 0.006,
	})
	if err != nil {
		t.Fatalf("UpdateAgentProvider() error = %v", err)
	}
	if updatedProvider.Name != "Codex Updated" || updatedProvider.ModelName != "gpt-5.4-mini" {
		t.Fatalf("UpdateAgentProvider() = %+v", updatedProvider)
	}
	if _, err := repo.UpdateAgentProvider(ctx, domain.UpdateAgentProvider{
		ID:                 agentProvider.ID,
		OrganizationID:     org.ID,
		MachineID:          uuid.New(),
		Name:               "Missing Machine Provider",
		AdapterType:        domain.AgentProviderAdapterTypeCodexAppServer,
		CliCommand:         "codex",
		AuthConfig:         map[string]any{},
		ModelName:          "gpt-5.4",
		CostPerInputToken:  0,
		CostPerOutputToken: 0,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateAgentProvider(missing machine) error = %v, want %v", err, ErrNotFound)
	}

	providers, err := repo.ListAgentProviders(ctx, org.ID)
	if err != nil {
		t.Fatalf("ListAgentProviders() error = %v", err)
	}
	if len(providers) < 2 {
		t.Fatalf("ListAgentProviders() = %+v, want builtin providers plus created provider", providers)
	}

	project, err := repo.CreateProject(ctx, domain.CreateProject{
		OrganizationID:         org.ID,
		Name:                   "OpenASE",
		Slug:                   "openase",
		Status:                 domain.ProjectStatusInProgress,
		DefaultAgentProviderID: &agentProvider.ID,
		AccessibleMachineIDs:   []uuid.UUID{localMachineID, remoteMachine.ID},
		MaxConcurrentAgents:    3,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if _, err := repo.ListAgents(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListAgents(missing project) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.ListAgentRuns(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListAgentRuns(missing project) error = %v, want %v", err, ErrNotFound)
	}

	orgB, err := repo.CreateOrganization(ctx, domain.CreateOrganization{
		Name: "GrandCX",
		Slug: "grandcx-agent-catalog",
	})
	if err != nil {
		t.Fatalf("CreateOrganization() orgB error = %v", err)
	}
	machinesB, err := repo.ListMachines(ctx, orgB.ID)
	if err != nil {
		t.Fatalf("ListMachines() orgB error = %v", err)
	}
	providerOtherOrg, err := repo.CreateAgentProvider(ctx, domain.CreateAgentProvider{
		OrganizationID:     orgB.ID,
		MachineID:          machinesB[0].ID,
		Name:               "Cross Org",
		AdapterType:        domain.AgentProviderAdapterTypeCodexAppServer,
		CliCommand:         "codex",
		AuthConfig:         map[string]any{},
		ModelName:          "gpt-5.4-mini",
		CostPerInputToken:  0,
		CostPerOutputToken: 0,
	})
	if err != nil {
		t.Fatalf("CreateAgentProvider() orgB error = %v", err)
	}

	createdAgent, err := repo.CreateAgent(ctx, domain.CreateAgent{
		ProjectID:             project.ID,
		ProviderID:            agentProvider.ID,
		Name:                  "codex-01",
		RuntimeControlState:   domain.AgentRuntimeControlStateActive,
		TotalTokensUsed:       12,
		TotalTicketsCompleted: 1,
	})
	if err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}
	if createdAgent.Name != "codex-01" || createdAgent.RuntimeControlState != domain.AgentRuntimeControlStateActive {
		t.Fatalf("CreateAgent() = %+v", createdAgent)
	}
	if _, err := repo.CreateAgent(ctx, domain.CreateAgent{
		ProjectID:             project.ID,
		ProviderID:            agentProvider.ID,
		Name:                  "codex-01",
		RuntimeControlState:   domain.AgentRuntimeControlStateActive,
		TotalTokensUsed:       0,
		TotalTicketsCompleted: 0,
	}); !errors.Is(err, domain.ErrAgentNameConflict) {
		t.Fatalf("CreateAgent(duplicate name) error = %v, want %v", err, domain.ErrAgentNameConflict)
	}
	if _, err := repo.CreateAgent(ctx, domain.CreateAgent{
		ProjectID:             project.ID,
		ProviderID:            uuid.New(),
		Name:                  "missing-provider",
		RuntimeControlState:   domain.AgentRuntimeControlStateActive,
		TotalTokensUsed:       0,
		TotalTicketsCompleted: 0,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("CreateAgent(missing provider) error = %v, want %v", err, ErrNotFound)
	}
	if _, err := repo.CreateAgent(ctx, domain.CreateAgent{
		ProjectID:             project.ID,
		ProviderID:            providerOtherOrg.ID,
		Name:                  "cross-org-provider",
		RuntimeControlState:   domain.AgentRuntimeControlStateActive,
		TotalTokensUsed:       0,
		TotalTicketsCompleted: 0,
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateAgent(cross org provider) error = %v, want %v", err, ErrInvalidInput)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("ResetToDefaultTemplate() error = %v", err)
	}
	todoID := findRepoCatalogStatusIDByName(t, statuses, "Todo")
	doneID := findRepoCatalogStatusIDByName(t, statuses, "Done")

	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetAgentID(createdAgent.ID).
		SetName("Coding Workflow").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-278").
		SetTitle("Finish backend coverage rollout").
		SetStatusID(todoID).
		SetWorkflowID(workflowItem.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	otherTicketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-279").
		SetTitle("Secondary output sink").
		SetStatusID(todoID).
		SetWorkflowID(workflowItem.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create secondary ticket: %v", err)
	}
	runCreatedAt := time.Date(2026, 3, 27, 14, 30, 0, 0, time.UTC)
	lastHeartbeatAt := runCreatedAt.Add(5 * time.Minute)
	runtimeStartedAt := runCreatedAt.Add(30 * time.Second)
	agentRun, err := client.AgentRun.Create().
		SetAgentID(createdAgent.ID).
		SetWorkflowID(workflowItem.ID).
		SetTicketID(ticketItem.ID).
		SetProviderID(agentProvider.ID).
		SetStatus(entagentrun.StatusExecuting).
		SetSessionID("session-278").
		SetRuntimeStartedAt(runtimeStartedAt).
		SetLastHeartbeatAt(lastHeartbeatAt).
		SetCreatedAt(runCreatedAt).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).SetCurrentRunID(agentRun.ID).Save(ctx); err != nil {
		t.Fatalf("bind current run: %v", err)
	}

	gotAgent, err := repo.GetAgent(ctx, createdAgent.ID)
	if err != nil {
		t.Fatalf("GetAgent() error = %v", err)
	}
	if gotAgent.Runtime == nil || gotAgent.Runtime.CurrentRunID == nil || *gotAgent.Runtime.CurrentRunID != agentRun.ID || gotAgent.Runtime.SessionID != "session-278" {
		t.Fatalf("GetAgent() = %+v", gotAgent)
	}
	if _, err := repo.GetAgent(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetAgent(missing) error = %v, want %v", err, ErrNotFound)
	}

	agents, err := repo.ListAgents(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListAgents() error = %v", err)
	}
	if len(agents) != 1 || agents[0].Runtime == nil || agents[0].Runtime.CurrentRunID == nil || *agents[0].Runtime.CurrentRunID != agentRun.ID {
		t.Fatalf("ListAgents() = %+v", agents)
	}

	updatedAgent, err := repo.UpdateAgentRuntimeControlState(ctx, domain.UpdateAgentRuntimeControlState{
		ID:                  createdAgent.ID,
		RuntimeControlState: domain.AgentRuntimeControlStatePauseRequested,
	})
	if err != nil {
		t.Fatalf("UpdateAgentRuntimeControlState() error = %v", err)
	}
	if updatedAgent.RuntimeControlState != domain.AgentRuntimeControlStatePauseRequested {
		t.Fatalf("UpdateAgentRuntimeControlState() = %+v", updatedAgent)
	}
	if _, err := repo.UpdateAgentRuntimeControlState(ctx, domain.UpdateAgentRuntimeControlState{
		ID:                  uuid.New(),
		RuntimeControlState: domain.AgentRuntimeControlStatePaused,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateAgentRuntimeControlState(missing) error = %v, want %v", err, ErrNotFound)
	}

	agentRuns, err := repo.ListAgentRuns(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListAgentRuns() error = %v", err)
	}
	if len(agentRuns) != 1 || agentRuns[0].ID != agentRun.ID {
		t.Fatalf("ListAgentRuns() = %+v", agentRuns)
	}

	gotRun, err := repo.GetAgentRun(ctx, agentRun.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if gotRun.SessionID != "session-278" || gotRun.RuntimeStartedAt == nil || !gotRun.RuntimeStartedAt.Equal(runtimeStartedAt) {
		t.Fatalf("GetAgentRun() = %+v", gotRun)
	}
	if _, err := repo.GetAgentRun(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetAgentRun(missing) error = %v, want %v", err, ErrNotFound)
	}

	eventOneCreatedAt := time.Date(2026, 3, 27, 14, 35, 0, 0, time.UTC)
	eventTwoCreatedAt := eventOneCreatedAt.Add(time.Minute)
	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetAgentID(createdAgent.ID).
		SetEventType("ticket.updated").
		SetMessage("ticket updated").
		SetMetadata(map[string]any{"field": "status"}).
		SetCreatedAt(eventOneCreatedAt).
		Save(ctx); err != nil {
		t.Fatalf("create activity event: %v", err)
	}
	traceEventOne, err := client.AgentTraceEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetAgentID(createdAgent.ID).
		SetAgentRunID(agentRun.ID).
		SetSequence(1).
		SetProvider("codex").
		SetKind(domain.AgentTraceKindCommandDelta).
		SetStream("stdout").
		SetText("stdout line").
		SetCreatedAt(eventTwoCreatedAt).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticketed agent trace event: %v", err)
	}
	traceEventTwo, err := client.AgentTraceEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(otherTicketItem.ID).
		SetAgentID(createdAgent.ID).
		SetAgentRunID(agentRun.ID).
		SetSequence(2).
		SetProvider("codex").
		SetKind(domain.AgentTraceKindCommandDelta).
		SetStream("stderr").
		SetText("stderr line").
		SetCreatedAt(eventTwoCreatedAt.Add(time.Minute)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create secondary ticket agent trace event: %v", err)
	}
	if _, err := repo.ListActivityEvents(ctx, domain.ListActivityEvents{
		ProjectID: uuid.New(),
		Limit:     10,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListActivityEvents(missing project) error = %v, want %v", err, ErrNotFound)
	}

	activityEvents, err := repo.ListActivityEvents(ctx, domain.ListActivityEvents{
		ProjectID: project.ID,
		AgentID:   &createdAgent.ID,
		TicketID:  &ticketItem.ID,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListActivityEvents() error = %v", err)
	}
	if len(activityEvents) != 1 || activityEvents[0].EventType != "ticket.updated" || activityEvents[0].Message != "ticket updated" {
		t.Fatalf("ListActivityEvents() = %+v", activityEvents)
	}
	projectWideActivity, err := repo.ListActivityEvents(ctx, domain.ListActivityEvents{
		ProjectID: project.ID,
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("ListActivityEvents() project-wide error = %v", err)
	}
	if len(projectWideActivity) != 1 || projectWideActivity[0].Message != "ticket updated" {
		t.Fatalf("ListActivityEvents() project-wide = %+v", projectWideActivity)
	}

	agentOutput, err := repo.ListAgentOutput(ctx, domain.ListAgentOutput{
		ProjectID: project.ID,
		AgentID:   createdAgent.ID,
		TicketID:  &ticketItem.ID,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListAgentOutput() error = %v", err)
	}
	if len(agentOutput) != 1 || agentOutput[0].Output != "stdout line" || agentOutput[0].Stream != "stdout" {
		t.Fatalf("ListAgentOutput() = %+v", agentOutput)
	}
	agentOutputAll, err := repo.ListAgentOutput(ctx, domain.ListAgentOutput{
		ProjectID: project.ID,
		AgentID:   createdAgent.ID,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListAgentOutput() without ticket filter error = %v", err)
	}
	if len(agentOutputAll) != 2 || agentOutputAll[0].Stream != "stderr" || agentOutputAll[1].Stream != "stdout" {
		t.Fatalf("ListAgentOutput() without ticket filter = %+v", agentOutputAll)
	}
	if _, err := repo.ListAgentOutput(ctx, domain.ListAgentOutput{
		ProjectID: project.ID,
		AgentID:   uuid.New(),
		Limit:     10,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListAgentOutput(missing agent) error = %v, want %v", err, ErrNotFound)
	}

	if _, err := client.AgentStepEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetAgentID(createdAgent.ID).
		SetAgentRunID(agentRun.ID).
		SetStepStatus("running").
		SetSummary("apply coverage fixes").
		SetSourceTraceEventID(traceEventOne.ID).
		SetCreatedAt(eventTwoCreatedAt.Add(2 * time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("create ticketed agent step event: %v", err)
	}
	if _, err := client.AgentStepEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(otherTicketItem.ID).
		SetAgentID(createdAgent.ID).
		SetAgentRunID(agentRun.ID).
		SetStepStatus("completed").
		SetSummary("secondary ticket complete").
		SetSourceTraceEventID(traceEventTwo.ID).
		SetCreatedAt(eventTwoCreatedAt.Add(3 * time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("create secondary ticket agent step event: %v", err)
	}

	agentSteps, err := repo.ListAgentSteps(ctx, domain.ListAgentSteps{
		ProjectID: project.ID,
		AgentID:   createdAgent.ID,
		TicketID:  &ticketItem.ID,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListAgentSteps() error = %v", err)
	}
	if len(agentSteps) != 1 || agentSteps[0].Summary != "apply coverage fixes" || agentSteps[0].SourceTraceEventID == nil || *agentSteps[0].SourceTraceEventID != traceEventOne.ID {
		t.Fatalf("ListAgentSteps() = %+v", agentSteps)
	}

	agentStepsAll, err := repo.ListAgentSteps(ctx, domain.ListAgentSteps{
		ProjectID: project.ID,
		AgentID:   createdAgent.ID,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListAgentSteps() without ticket filter error = %v", err)
	}
	if len(agentStepsAll) != 2 || agentStepsAll[0].Summary != "secondary ticket complete" || agentStepsAll[1].Summary != "apply coverage fixes" {
		t.Fatalf("ListAgentSteps() without ticket filter = %+v", agentStepsAll)
	}
	if _, err := repo.ListAgentSteps(ctx, domain.ListAgentSteps{
		ProjectID: project.ID,
		AgentID:   uuid.New(),
		Limit:     10,
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListAgentSteps(missing agent) error = %v, want %v", err, ErrNotFound)
	}

	if _, err := repo.DeleteAgent(ctx, createdAgent.ID); !errors.Is(err, domain.ErrAgentInUseConflict) {
		t.Fatalf("DeleteAgent() with active runs error = %v, want %v", err, domain.ErrAgentInUseConflict)
	}
	if _, err := client.AgentRun.UpdateOneID(agentRun.ID).SetTerminalAt(time.Now().UTC()).Save(ctx); err != nil {
		t.Fatalf("terminalize agent run: %v", err)
	}
	if _, err := repo.DeleteAgent(ctx, createdAgent.ID); !errors.Is(err, domain.ErrAgentInUseConflict) {
		t.Fatalf("DeleteAgent() with historical runs error = %v, want %v", err, domain.ErrAgentInUseConflict)
	}
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).ClearCurrentRunID().Save(ctx); err != nil {
		t.Fatalf("clear current run: %v", err)
	}
	if _, err := client.AgentStepEvent.Delete().Exec(ctx); err != nil {
		t.Fatalf("delete agent step events: %v", err)
	}
	if _, err := client.AgentTraceEvent.Delete().Where(entagenttraceevent.AgentRunID(agentRun.ID)).Exec(ctx); err != nil {
		t.Fatalf("delete agent trace events: %v", err)
	}
	if err := client.AgentRun.DeleteOneID(agentRun.ID).Exec(ctx); err != nil {
		t.Fatalf("delete agent run: %v", err)
	}

	deletedAgent, err := repo.DeleteAgent(ctx, createdAgent.ID)
	if err != nil {
		t.Fatalf("DeleteAgent() error = %v", err)
	}
	if deletedAgent.ID != createdAgent.ID {
		t.Fatalf("DeleteAgent() = %+v", deletedAgent)
	}
	if _, err := repo.DeleteAgent(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteAgent(missing) error = %v, want %v", err, ErrNotFound)
	}
}

const openAIAPIKeyEnv = "OPENAI_" + "API_KEY"
