package ticketstatus

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/google/uuid"
)

func TestTicketStatusServiceStatusCRUDResetAndRebind(t *testing.T) {
	t.Parallel()

	client := openTicketStatusTestEntClient(t)
	ctx := context.Background()
	projectID := seedTicketStatusProject(ctx, t, client)
	service := NewService(client)

	resetStatuses, err := service.ResetToDefaultTemplate(ctx, projectID)
	if err != nil {
		t.Fatalf("ResetToDefaultTemplate() error = %v", err)
	}
	if len(resetStatuses) != 6 {
		t.Fatalf("ResetToDefaultTemplate() statuses = %d, want 6", len(resetStatuses))
	}

	listResult, err := service.List(ctx, projectID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listResult.Stages) != 4 || len(listResult.Statuses) != 6 || len(listResult.StageGroups) != 4 {
		t.Fatalf("List() = %+v", listResult)
	}
	if got := statusNames(listResult.Statuses); got != "Backlog,Todo,In Progress,In Review,Done,Cancelled" {
		t.Fatalf("status order = %q", got)
	}

	createdStatus, err := service.Create(ctx, CreateInput{
		ProjectID:   projectID,
		Name:        "QA",
		Color:       "#FF00AA",
		Description: "quality gate",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if createdStatus.Name != "QA" {
		t.Fatalf("Create() = %+v", createdStatus)
	}

	updatedStatus, err := service.Update(ctx, UpdateInput{
		StatusID:    createdStatus.ID,
		Name:        Some("Ready for QA"),
		Color:       Some("#00AAFF"),
		Icon:        Some("shield-check"),
		Position:    Some(9),
		IsDefault:   Some(true),
		Description: Some("review before merge"),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updatedStatus.Name != "Ready for QA" || !updatedStatus.IsDefault {
		t.Fatalf("Update() = %+v", updatedStatus)
	}

	workflowWithDeletedStatus, err := client.Workflow.Create().
		SetProjectID(projectID).
		SetName("qa-workflow").
		SetType("test").
		SetHarnessPath("roles/qa.md").
		AddPickupStatusIDs(updatedStatus.ID).
		AddFinishStatusIDs(updatedStatus.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow for delete rebind: %v", err)
	}
	ticketWithDeletedStatus, err := client.Ticket.Create().
		SetProjectID(projectID).
		SetIdentifier("ASE-5").
		SetTitle("qa gate").
		SetStatusID(updatedStatus.ID).
		SetWorkflowID(workflowWithDeletedStatus.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket for delete rebind: %v", err)
	}

	deleteResult, err := service.Delete(ctx, updatedStatus.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleteResult.DeletedStatusID != updatedStatus.ID {
		t.Fatalf("Delete() = %+v", deleteResult)
	}

	ticketAfterDelete, err := client.Ticket.Get(ctx, ticketWithDeletedStatus.ID)
	if err != nil {
		t.Fatalf("load ticket after delete: %v", err)
	}
	if ticketAfterDelete.StatusID != deleteResult.ReplacementStatusID {
		t.Fatalf("ticket status = %s, want %s", ticketAfterDelete.StatusID, deleteResult.ReplacementStatusID)
	}
	workflowAfterDelete, err := client.Workflow.Query().
		Where(entworkflow.IDEQ(workflowWithDeletedStatus.ID)).
		WithPickupStatuses().
		WithFinishStatuses().
		Only(ctx)
	if err != nil {
		t.Fatalf("load workflow after delete: %v", err)
	}
	if len(workflowAfterDelete.Edges.PickupStatuses) != 1 || workflowAfterDelete.Edges.PickupStatuses[0].ID != deleteResult.ReplacementStatusID {
		t.Fatalf("pickup statuses after delete = %+v", workflowAfterDelete.Edges.PickupStatuses)
	}
	if len(workflowAfterDelete.Edges.FinishStatuses) != 1 || workflowAfterDelete.Edges.FinishStatuses[0].ID != deleteResult.ReplacementStatusID {
		t.Fatalf("finish statuses after delete = %+v", workflowAfterDelete.Edges.FinishStatuses)
	}

	extraStatus, err := service.Create(ctx, CreateInput{
		ProjectID: projectID,
		Name:      "Research",
		Color:     "#111111",
		Position:  Some(12),
	})
	if err != nil {
		t.Fatalf("create extra status: %v", err)
	}
	workflowForReset, err := client.Workflow.Create().
		SetProjectID(projectID).
		SetName("research-workflow").
		SetType("custom").
		SetHarnessPath("roles/research.md").
		AddPickupStatusIDs(extraStatus.ID).
		AddFinishStatusIDs(extraStatus.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow for reset rebind: %v", err)
	}
	ticketForReset, err := client.Ticket.Create().
		SetProjectID(projectID).
		SetIdentifier("ASE-6").
		SetTitle("research").
		SetStatusID(extraStatus.ID).
		SetWorkflowID(workflowForReset.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket for reset rebind: %v", err)
	}

	resetAgainStatuses, err := service.ResetToDefaultTemplate(ctx, projectID)
	if err != nil {
		t.Fatalf("second ResetToDefaultTemplate() error = %v", err)
	}
	if len(resetAgainStatuses) != 6 {
		t.Fatalf("second ResetToDefaultTemplate() statuses = %d, want 6", len(resetAgainStatuses))
	}
	for _, status := range resetAgainStatuses {
		if status.Name == "Research" {
			t.Fatalf("reset should remove Research, got %+v", resetAgainStatuses)
		}
	}

	listAfterReset, err := service.List(ctx, projectID)
	if err != nil {
		t.Fatalf("List() after reset error = %v", err)
	}
	backlogID := findStatusIDByName(t, listAfterReset.Statuses, "Backlog")
	todoID := findStatusIDByName(t, listAfterReset.Statuses, "Todo")
	doneID := findStatusIDByName(t, listAfterReset.Statuses, "Done")
	if got := statusNames(listAfterReset.Statuses); got != "Backlog,Todo,In Progress,In Review,Done,Cancelled" {
		t.Fatalf("status order after reset = %q", got)
	}

	ticketAfterReset, err := client.Ticket.Get(ctx, ticketForReset.ID)
	if err != nil {
		t.Fatalf("load ticket after reset: %v", err)
	}
	if ticketAfterReset.StatusID != backlogID {
		t.Fatalf("ticket reset status = %s, want backlog %s", ticketAfterReset.StatusID, backlogID)
	}
	workflowAfterReset, err := client.Workflow.Query().
		Where(entworkflow.IDEQ(workflowForReset.ID)).
		WithPickupStatuses().
		WithFinishStatuses().
		Only(ctx)
	if err != nil {
		t.Fatalf("load workflow after reset: %v", err)
	}
	if len(workflowAfterReset.Edges.PickupStatuses) != 1 || workflowAfterReset.Edges.PickupStatuses[0].ID != todoID {
		t.Fatalf("pickup statuses after reset = %+v", workflowAfterReset.Edges.PickupStatuses)
	}
	if len(workflowAfterReset.Edges.FinishStatuses) != 1 || workflowAfterReset.Edges.FinishStatuses[0].ID != doneID {
		t.Fatalf("finish statuses after reset = %+v", workflowAfterReset.Edges.FinishStatuses)
	}
}

func TestTicketStatusServiceStageCRUDAndSnapshots(t *testing.T) {
	t.Parallel()

	client := openTicketStatusTestEntClient(t)
	ctx := context.Background()
	projectID := seedTicketStatusProject(ctx, t, client)
	service := NewService(client)

	stage, err := service.CreateStage(ctx, CreateStageInput{
		ProjectID:     projectID,
		Key:           "qa",
		Name:          "QA",
		Position:      Some(2),
		MaxActiveRuns: ptrInt(1),
		Description:   "quality gate",
	})
	if err != nil {
		t.Fatalf("CreateStage() error = %v", err)
	}
	if stage.Key != "qa" || stage.MaxActiveRuns == nil || *stage.MaxActiveRuns != 1 {
		t.Fatalf("CreateStage() = %+v", stage)
	}

	status, err := service.Create(ctx, CreateInput{
		ProjectID: projectID,
		StageID:   &stage.ID,
		Name:      "QA Ready",
		Color:     "#FF00AA",
		IsDefault: true,
	})
	if err != nil {
		t.Fatalf("Create() stage status error = %v", err)
	}
	if status.StageID == nil || *status.StageID != stage.ID {
		t.Fatalf("Create() stage status = %+v", status)
	}

	seedTicketStatusActiveRun(ctx, t, client, projectID, status.ID)

	listResult, err := service.List(ctx, projectID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listResult.Stages) != 1 || listResult.Stages[0].ActiveRuns != 1 {
		t.Fatalf("List() stages = %+v", listResult.Stages)
	}
	if len(listResult.StageGroups) != 1 || listResult.StageGroups[0].Stage == nil || listResult.StageGroups[0].Stage.ActiveRuns != 1 {
		t.Fatalf("List() groups = %+v", listResult.StageGroups)
	}

	stages, err := service.ListStages(ctx, projectID)
	if err != nil {
		t.Fatalf("ListStages() error = %v", err)
	}
	if len(stages) != 1 || stages[0].ActiveRuns != 1 {
		t.Fatalf("ListStages() = %+v", stages)
	}

	projectSnapshots, err := ListProjectStageRuntimeSnapshots(ctx, client, projectID)
	if err != nil {
		t.Fatalf("ListProjectStageRuntimeSnapshots() error = %v", err)
	}
	if len(projectSnapshots) != 1 || projectSnapshots[0].ActiveRuns != 1 {
		t.Fatalf("project snapshots = %+v", projectSnapshots)
	}
	allSnapshots, err := ListStageRuntimeSnapshots(ctx, client)
	if err != nil {
		t.Fatalf("ListStageRuntimeSnapshots() error = %v", err)
	}
	if len(allSnapshots) != 1 || allSnapshots[0].ActiveRuns != 1 {
		t.Fatalf("all snapshots = %+v", allSnapshots)
	}

	updatedStage, err := service.UpdateStage(ctx, UpdateStageInput{
		StageID:       stage.ID,
		Name:          Some("QA Gate"),
		Position:      Some(5),
		MaxActiveRuns: Some[*int](nil),
		Description:   Some("merge gate"),
	})
	if err != nil {
		t.Fatalf("UpdateStage() error = %v", err)
	}
	if updatedStage.Name != "QA Gate" || updatedStage.Position != 5 || updatedStage.MaxActiveRuns != nil {
		t.Fatalf("UpdateStage() = %+v", updatedStage)
	}

	deleteResult, err := service.DeleteStage(ctx, stage.ID)
	if err != nil {
		t.Fatalf("DeleteStage() error = %v", err)
	}
	if deleteResult.DeletedStageID != stage.ID || deleteResult.DetachedStatuses != 1 {
		t.Fatalf("DeleteStage() = %+v", deleteResult)
	}

	statusAfterDelete, err := client.TicketStatus.Get(ctx, status.ID)
	if err != nil {
		t.Fatalf("load status after stage delete: %v", err)
	}
	if statusAfterDelete.StageID != nil {
		t.Fatalf("status after stage delete = %+v", statusAfterDelete)
	}

	listAfterDelete, err := service.List(ctx, projectID)
	if err != nil {
		t.Fatalf("List() after delete error = %v", err)
	}
	if len(listAfterDelete.Stages) != 0 {
		t.Fatalf("stages after delete = %+v", listAfterDelete.Stages)
	}
	if len(listAfterDelete.StageGroups) != 1 || listAfterDelete.StageGroups[0].Stage != nil || len(listAfterDelete.StageGroups[0].Statuses) != 1 {
		t.Fatalf("groups after delete = %+v", listAfterDelete.StageGroups)
	}
}

func TestTicketStatusServiceBackfillDefaultStages(t *testing.T) {
	t.Parallel()

	client := openTicketStatusTestEntClient(t)
	ctx := context.Background()
	projectID := seedTicketStatusProject(ctx, t, client)
	service := NewService(client)

	for index, item := range defaultStatusTemplate {
		builder := client.TicketStatus.Create().
			SetProjectID(projectID).
			SetName(item.Name).
			SetColor(item.Color).
			SetPosition(index).
			SetIsDefault(item.IsDefault)
		if item.Icon != "" {
			builder.SetIcon(item.Icon)
		}
		if item.Description != "" {
			builder.SetDescription(item.Description)
		}
		if _, err := builder.Save(ctx); err != nil {
			t.Fatalf("seed legacy status %q: %v", item.Name, err)
		}
	}

	if err := service.BackfillDefaultStages(ctx); err != nil {
		t.Fatalf("BackfillDefaultStages() error = %v", err)
	}
	stages, err := service.ListStages(ctx, projectID)
	if err != nil {
		t.Fatalf("ListStages() after backfill error = %v", err)
	}
	if len(stages) != 4 {
		t.Fatalf("stages after backfill = %+v", stages)
	}
	statuses, err := client.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectID)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		t.Fatalf("load statuses after backfill: %v", err)
	}
	for _, status := range statuses {
		if status.StageID == nil {
			t.Fatalf("status %q missing stage after backfill", status.Name)
		}
	}

	if err := service.BackfillDefaultStages(ctx); err != nil {
		t.Fatalf("BackfillDefaultStages() second call error = %v", err)
	}
}

func TestTicketStatusServiceErrorPaths(t *testing.T) {
	t.Parallel()

	client := openTicketStatusTestEntClient(t)
	ctx := context.Background()
	projectID := seedTicketStatusProject(ctx, t, client)
	service := NewService(client)

	primaryStatus, err := service.Create(ctx, CreateInput{
		ProjectID: projectID,
		Name:      "Solo",
		Color:     "#111111",
	})
	if err != nil {
		t.Fatalf("Create(primary status) error = %v", err)
	}
	if !primaryStatus.IsDefault {
		t.Fatalf("Create(primary status) = %+v, want default", primaryStatus)
	}

	if _, err := service.Update(ctx, UpdateInput{
		StatusID:  primaryStatus.ID,
		IsDefault: Some(false),
	}); !errors.Is(err, ErrDefaultStatusRequired) {
		t.Fatalf("Update(clear only default) error = %v, want %v", err, ErrDefaultStatusRequired)
	}
	if _, err := service.Delete(ctx, primaryStatus.ID); !errors.Is(err, ErrCannotDeleteLastStatus) {
		t.Fatalf("Delete(last status) error = %v, want %v", err, ErrCannotDeleteLastStatus)
	}

	stage, err := service.CreateStage(ctx, CreateStageInput{
		ProjectID: projectID,
		Key:       "qa",
		Name:      "QA",
	})
	if err != nil {
		t.Fatalf("CreateStage() error = %v", err)
	}
	if _, err := service.CreateStage(ctx, CreateStageInput{
		ProjectID: projectID,
		Key:       "qa",
		Name:      "QA Duplicate",
	}); !errors.Is(err, ErrDuplicateStageKey) {
		t.Fatalf("CreateStage(duplicate) error = %v, want %v", err, ErrDuplicateStageKey)
	}

	secondStatus, err := service.Create(ctx, CreateInput{
		ProjectID: projectID,
		StageID:   &stage.ID,
		Name:      "Shared",
		Color:     "#222222",
	})
	if err != nil {
		t.Fatalf("Create(second status) error = %v", err)
	}
	if _, err := service.Create(ctx, CreateInput{
		ProjectID: projectID,
		Name:      "Shared",
		Color:     "#333333",
	}); !errors.Is(err, ErrDuplicateStatusName) {
		t.Fatalf("Create(duplicate status) error = %v, want %v", err, ErrDuplicateStatusName)
	}

	project, err := client.Project.Get(ctx, projectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	otherProject, err := client.Project.Create().
		SetOrganizationID(project.OrganizationID).
		SetName("OpenASE Secondary").
		SetSlug(strings.ToLower("openase-secondary-" + uuid.NewString()[:8])).
		Save(ctx)
	if err != nil {
		t.Fatalf("create second project: %v", err)
	}
	foreignStage, err := service.CreateStage(ctx, CreateStageInput{
		ProjectID: otherProject.ID,
		Key:       "foreign",
		Name:      "Foreign",
	})
	if err != nil {
		t.Fatalf("CreateStage(foreign) error = %v", err)
	}
	if _, err := service.Update(ctx, UpdateInput{
		StatusID: secondStatus.ID,
		StageID:  Some(&foreignStage.ID),
	}); !errors.Is(err, ErrStageNotFound) {
		t.Fatalf("Update(foreign stage) error = %v, want %v", err, ErrStageNotFound)
	}

	updatedStatus, err := service.Update(ctx, UpdateInput{
		StatusID:    secondStatus.ID,
		StageID:     Some[*uuid.UUID](nil),
		Icon:        Some(""),
		Description: Some(""),
		Position:    Some(7),
	})
	if err != nil {
		t.Fatalf("Update(clear stage fields) error = %v", err)
	}
	if updatedStatus.StageID != nil || updatedStatus.Icon != "" || updatedStatus.Description != "" || updatedStatus.Position != 7 {
		t.Fatalf("Update(clear stage fields) = %+v", updatedStatus)
	}

	if _, err := service.DeleteStage(ctx, uuid.New()); !errors.Is(err, ErrStageNotFound) {
		t.Fatalf("DeleteStage(missing) error = %v, want %v", err, ErrStageNotFound)
	}
	if _, err := service.UpdateStage(ctx, UpdateStageInput{StageID: uuid.New(), Name: Some("missing")}); !errors.Is(err, ErrStageNotFound) {
		t.Fatalf("UpdateStage(missing) error = %v, want %v", err, ErrStageNotFound)
	}
	if _, err := service.Update(ctx, UpdateInput{StatusID: uuid.New(), Name: Some("missing")}); !errors.Is(err, ErrStatusNotFound) {
		t.Fatalf("Update(missing) error = %v, want %v", err, ErrStatusNotFound)
	}
	if _, err := service.Delete(ctx, uuid.New()); !errors.Is(err, ErrStatusNotFound) {
		t.Fatalf("Delete(missing) error = %v, want %v", err, ErrStatusNotFound)
	}
}

func TestTicketStatusServiceMissingProjectAndEmptyBackfillPaths(t *testing.T) {
	t.Parallel()

	client := openTicketStatusTestEntClient(t)
	ctx := context.Background()
	service := NewService(client)
	missingProjectID := uuid.New()

	if _, err := service.List(ctx, missingProjectID); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("List(missing project) error = %v, want %v", err, ErrProjectNotFound)
	}
	if _, err := service.ListStages(ctx, missingProjectID); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("ListStages(missing project) error = %v, want %v", err, ErrProjectNotFound)
	}
	if _, err := service.CreateStage(ctx, CreateStageInput{
		ProjectID: missingProjectID,
		Key:       "qa",
		Name:      "QA",
	}); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("CreateStage(missing project) error = %v, want %v", err, ErrProjectNotFound)
	}
	if _, err := service.Create(ctx, CreateInput{
		ProjectID: missingProjectID,
		Name:      "Todo",
		Color:     "#111111",
	}); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("Create(missing project) error = %v, want %v", err, ErrProjectNotFound)
	}
	if _, err := service.ResetToDefaultTemplate(ctx, missingProjectID); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("ResetToDefaultTemplate(missing project) error = %v, want %v", err, ErrProjectNotFound)
	}
	if err := service.BackfillDefaultStages(ctx); err != nil {
		t.Fatalf("BackfillDefaultStages(empty) error = %v", err)
	}
}

func openTicketStatusTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}

func seedTicketStatusProject(ctx context.Context, t *testing.T, client *ent.Client) uuid.UUID {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug(strings.ToLower("openase-" + uuid.NewString()[:8])).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return project.ID
}

func seedTicketStatusActiveRun(ctx context.Context, t *testing.T, client *ent.Client, projectID uuid.UUID, statusID uuid.UUID) {
	t.Helper()

	project, err := client.Project.Get(ctx, projectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(project.OrganizationID).
		SetName("worker-" + uuid.NewString()[:8]).
		SetHost("worker.internal").
		SetPort(22).
		SetDescription("runner").
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(project.OrganizationID).
		SetMachineID(machine.ID).
		SetName("Provider").
		SetAdapterType(entagentprovider.AdapterTypeCustom).
		SetCliCommand("codex").
		SetCliArgs([]string{"run"}).
		SetAuthConfig(map[string]any{}).
		SetModelName("gpt-5.4").
		SetModelTemperature(0).
		SetModelMaxTokens(8192).
		SetCostPerInputToken(0).
		SetCostPerOutputToken(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(projectID).
		SetProviderID(providerItem.ID).
		SetName("Planner").
		SetRuntimeControlState(entagent.RuntimeControlStateActive).
		SetTotalTokensUsed(0).
		SetTotalTicketsCompleted(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	workflowItem, err := client.Workflow.Create().
		SetProjectID(projectID).
		SetName("QA Workflow").
		SetType("custom").
		SetHarnessPath("roles/qa.md").
		SetAgentID(agentItem.ID).
		AddPickupStatusIDs(statusID).
		AddFinishStatusIDs(statusID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(projectID).
		SetIdentifier("ASE-99").
		SetTitle("qa gate").
		SetStatusID(statusID).
		SetWorkflowID(workflowItem.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem, err := client.AgentRun.Create().
		SetAgentID(agentItem.ID).
		SetWorkflowID(workflowItem.ID).
		SetTicketID(ticketItem.ID).
		SetProviderID(providerItem.ID).
		SetStatus(entagentrun.StatusExecuting).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).SetCurrentRunID(runItem.ID).Save(ctx); err != nil {
		t.Fatalf("set ticket current run: %v", err)
	}
}

func statusNames(items []Status) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return strings.Join(names, ",")
}

func findStatusIDByName(t *testing.T, items []Status, want string) uuid.UUID {
	t.Helper()

	for _, item := range items {
		if item.Name == want {
			return item.ID
		}
	}
	t.Fatalf("status %q not found in %+v", want, items)
	return uuid.Nil
}

func ptrInt(value int) *int {
	return &value
}
