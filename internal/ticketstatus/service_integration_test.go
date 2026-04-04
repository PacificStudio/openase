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
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	"github.com/google/uuid"
)

func TestTicketStatusServiceStatusCRUDResetAndRebind(t *testing.T) {
	t.Parallel()

	client := openTicketStatusTestEntClient(t)
	ctx := context.Background()
	projectID := seedTicketStatusProject(ctx, t, client)
	repo := ticketstatusrepo.NewEntRepository(client)
	service := NewService(repo)

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
	if got := statusNames(listResult.Statuses); got != "Backlog,Todo,In Progress,In Review,Done,Cancelled" {
		t.Fatalf("status order = %q", got)
	}

	createdStatus, err := service.Create(ctx, CreateInput{
		ProjectID:     projectID,
		Name:          "QA",
		Color:         "#FF00AA",
		MaxActiveRuns: ptrInt(1),
		Description:   "quality gate",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if createdStatus.Name != "QA" || createdStatus.MaxActiveRuns == nil || *createdStatus.MaxActiveRuns != 1 {
		t.Fatalf("Create() = %+v", createdStatus)
	}

	updatedStatus, err := service.Update(ctx, UpdateInput{
		StatusID:      createdStatus.ID,
		Name:          Some("Ready for QA"),
		Color:         Some("#00AAFF"),
		Icon:          Some("shield-check"),
		Position:      Some(9),
		MaxActiveRuns: Some[*int](nil),
		IsDefault:     Some(true),
		Description:   Some("review before merge"),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updatedStatus.Name != "Ready for QA" || !updatedStatus.IsDefault || updatedStatus.MaxActiveRuns != nil {
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

func TestTicketStatusServiceRuntimeSnapshots(t *testing.T) {
	t.Parallel()

	client := openTicketStatusTestEntClient(t)
	ctx := context.Background()
	projectID := seedTicketStatusProject(ctx, t, client)
	repo := ticketstatusrepo.NewEntRepository(client)
	service := NewService(repo)

	status, err := service.Create(ctx, CreateInput{
		ProjectID:     projectID,
		Name:          "QA Ready",
		Color:         "#FF00AA",
		MaxActiveRuns: ptrInt(1),
		IsDefault:     true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	seedTicketStatusActiveRun(ctx, t, client, projectID, status.ID)

	listResult, err := service.List(ctx, projectID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listResult.Statuses) != 1 || listResult.Statuses[0].ActiveRuns != 1 {
		t.Fatalf("List() statuses = %+v", listResult.Statuses)
	}

	projectSnapshots, err := ListProjectStatusRuntimeSnapshots(ctx, repo, projectID)
	if err != nil {
		t.Fatalf("ListProjectStatusRuntimeSnapshots() error = %v", err)
	}
	if len(projectSnapshots) != 1 || projectSnapshots[0].ActiveRuns != 1 {
		t.Fatalf("project snapshots = %+v", projectSnapshots)
	}

	allSnapshots, err := ListStatusRuntimeSnapshots(ctx, repo)
	if err != nil {
		t.Fatalf("ListStatusRuntimeSnapshots() error = %v", err)
	}
	if len(allSnapshots) != 1 || allSnapshots[0].ActiveRuns != 1 {
		t.Fatalf("all snapshots = %+v", allSnapshots)
	}
}

func TestTicketStatusServiceErrorPaths(t *testing.T) {
	t.Parallel()

	client := openTicketStatusTestEntClient(t)
	ctx := context.Background()
	projectID := seedTicketStatusProject(ctx, t, client)
	service := NewService(ticketstatusrepo.NewEntRepository(client))

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

	secondStatus, err := service.Create(ctx, CreateInput{
		ProjectID: projectID,
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
	if _, err := service.Create(ctx, CreateInput{
		ProjectID: projectID,
		Name:      "shared",
		Color:     "#444444",
	}); !errors.Is(err, ErrDuplicateStatusName) {
		t.Fatalf("Create(case-insensitive duplicate status) error = %v, want %v", err, ErrDuplicateStatusName)
	}

	updatedStatus, err := service.Update(ctx, UpdateInput{
		StatusID:      secondStatus.ID,
		MaxActiveRuns: Some[*int](nil),
		Icon:          Some(""),
		Description:   Some(""),
		Position:      Some(7),
	})
	if err != nil {
		t.Fatalf("Update(clear optional fields) error = %v", err)
	}
	if updatedStatus.MaxActiveRuns != nil || updatedStatus.Icon != "" || updatedStatus.Description != "" || updatedStatus.Position != 7 {
		t.Fatalf("Update(clear optional fields) = %+v", updatedStatus)
	}

	if _, err := service.Update(ctx, UpdateInput{StatusID: uuid.New(), Name: Some("missing")}); !errors.Is(err, ErrStatusNotFound) {
		t.Fatalf("Update(missing) error = %v, want %v", err, ErrStatusNotFound)
	}
	if _, err := service.Delete(ctx, uuid.New()); !errors.Is(err, ErrStatusNotFound) {
		t.Fatalf("Delete(missing) error = %v, want %v", err, ErrStatusNotFound)
	}

	casefoldA, err := client.TicketStatus.Create().
		SetProjectID(projectID).
		SetName("CaseFold").
		SetStage("unstarted").
		SetColor("#555555").
		SetPosition(10).
		Save(ctx)
	if err != nil {
		t.Fatalf("Create(casefold status A) error = %v", err)
	}
	if _, err := client.TicketStatus.Create().
		SetProjectID(projectID).
		SetName("casefold").
		SetStage("unstarted").
		SetColor("#666666").
		SetPosition(11).
		Save(ctx); err != nil {
		t.Fatalf("Create(casefold status B) error = %v", err)
	}
	if _, err := service.ResolveStatusIDByName(ctx, projectID, "CASEFOLD"); !errors.Is(err, ErrDuplicateStatusName) {
		t.Fatalf("ResolveStatusIDByName(ambiguous casefold) error = %v, want %v", err, ErrDuplicateStatusName)
	}
	if err := client.TicketStatus.DeleteOneID(casefoldA.ID).Exec(ctx); err != nil {
		t.Fatalf("Delete(casefold status A) error = %v", err)
	}
}

func TestTicketStatusServiceMissingProjectPaths(t *testing.T) {
	t.Parallel()

	client := openTicketStatusTestEntClient(t)
	ctx := context.Background()
	service := NewService(ticketstatusrepo.NewEntRepository(client))
	missingProjectID := uuid.New()

	if _, err := service.List(ctx, missingProjectID); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("List(missing project) error = %v, want %v", err, ErrProjectNotFound)
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
}

func openTicketStatusTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}

func seedTicketStatusProject(ctx context.Context, t *testing.T, client *ent.Client) uuid.UUID {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug(strings.ToLower("better-and-better-" + uuid.NewString()[:8])).
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
		SetName("worker").
		SetHost("127.0.0.1").
		SetSSHUser("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(project.OrganizationID).
		SetName("codex").
		SetMachineID(machine.ID).
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		SetMaxParallelRuns(3).
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(projectID).
		SetProviderID(provider.ID).
		SetName("coding-01").
		SetRuntimeControlState(entagent.RuntimeControlStateActive).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	workflowItem, err := client.Workflow.Create().
		SetProjectID(projectID).
		SetAgentID(agentItem.ID).
		SetName("Coding").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(statusID).
		AddFinishStatusIDs(statusID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(projectID).
		SetIdentifier("ASE-100").
		SetTitle("runtime snapshot").
		SetStatusID(statusID).
		SetWorkflowID(workflowItem.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	run, err := client.AgentRun.Create().
		SetWorkflowID(workflowItem.ID).
		SetAgentID(agentItem.ID).
		SetProviderID(provider.ID).
		SetTicketID(ticketItem.ID).
		SetStatus(entagentrun.StatusExecuting).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).SetCurrentRunID(run.ID).Save(ctx); err != nil {
		t.Fatalf("attach current run: %v", err)
	}
}

func findStatusIDByName(t *testing.T, statuses []Status, name string) uuid.UUID {
	t.Helper()

	for _, status := range statuses {
		if status.Name == name {
			return status.ID
		}
	}
	t.Fatalf("status %q not found in %+v", name, statuses)
	return uuid.UUID{}
}

func statusNames(statuses []Status) string {
	names := make([]string, 0, len(statuses))
	for _, status := range statuses {
		names = append(names, status.Name)
	}
	return strings.Join(names, ",")
}

func ptrInt(value int) *int {
	return &value
}
