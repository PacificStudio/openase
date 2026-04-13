package ticket

import (
	"context"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entticketreposcope "github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	domain "github.com/BetterAndBetterII/openase/internal/domain/ticket"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()
	return testPostgres.NewIsolatedEntClient(t)
}

func newTicketStatusService(client *ent.Client) *ticketstatus.Service {
	return ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client))
}

func some[T any](value T) Optional[T] {
	return Optional[T]{Set: true, Value: value}
}

func createRepoTicketTestProject(ctx context.Context, t *testing.T, client *ent.Client) (uuid.UUID, []ticketstatus.Status) {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-repo-ticket").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-repo-ticket").
		SetDescription("ticket repo tests").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("ResetToDefaultTemplate() error = %v", err)
	}
	return project.ID, statuses
}

func createProjectRepo(ctx context.Context, t *testing.T, client *ent.Client, projectID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	repoItem, err := client.ProjectRepo.Create().
		SetProjectID(projectID).
		SetName(name).
		SetRepositoryURL("https://github.com/pacificstudio/" + name + ".git").
		SetDefaultBranch("main").
		SetWorkspaceDirname(name).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}
	return repoItem.ID
}

func findStatusIDByName(t *testing.T, statuses []ticketstatus.Status, name string) uuid.UUID {
	t.Helper()
	for _, status := range statuses {
		if status.Name == name {
			return status.ID
		}
	}
	t.Fatalf("status %q not found in %+v", name, statuses)
	return uuid.Nil
}

func createTicketForRepoTest(ctx context.Context, t *testing.T, repo *CommandRepository, projectID uuid.UUID, statusID uuid.UUID, title string) Ticket {
	t.Helper()
	ticketItem, err := repo.Create(ctx, CreateInput{
		ProjectID: projectID,
		Title:     title,
		StatusID:  &statusID,
		CreatedBy: "codex",
		Type:      domain.TypeRefactor,
	})
	if err != nil {
		t.Fatalf("Create(%q) error = %v", title, err)
	}
	return ticketItem
}

func TestCommandAndQueryRepositoriesCreateUpdateListGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	projectID, statuses := createRepoTicketTestProject(ctx, t, client)
	createProjectRepo(ctx, t, client, projectID, "openase")

	todoID := findStatusIDByName(t, statuses, "Todo")
	inProgressID := findStatusIDByName(t, statuses, "In Progress")

	commandRepo := NewCommandRepository(client)
	queryRepo := NewQueryRepository(client)

	created, err := commandRepo.Create(ctx, CreateInput{
		ProjectID:   projectID,
		Title:       "Split ticket repo",
		Description: "narrow repo boundaries",
		StatusID:    &todoID,
		CreatedBy:   "codex",
		BudgetUSD:   12.5,
		ExternalRef: "ASE-183",
		RepoScopes:  nil,
		Type:        domain.TypeRefactor,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.StatusID != todoID || created.Title != "Split ticket repo" {
		t.Fatalf("Create() = %+v", created)
	}
	repoScopeCount, err := client.TicketRepoScope.Query().Where(entticketreposcope.TicketIDEQ(created.ID)).Count(ctx)
	if err != nil {
		t.Fatalf("count repo scopes: %v", err)
	}
	if repoScopeCount != 1 {
		t.Fatalf("repo scope count = %d, want 1", repoScopeCount)
	}

	updated, err := commandRepo.Update(ctx, UpdateInput{
		TicketID:    created.ID,
		Title:       some("Split ticket repository"),
		Description: some("split command/query/comment/runtime repos"),
		StatusID:    some(inProgressID),
		BudgetUSD:   some(25.0),
		ExternalRef: some("ASE-183-refined"),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Ticket.Title != "Split ticket repository" || updated.Ticket.StatusID != inProgressID || updated.Ticket.BudgetUSD != 25 {
		t.Fatalf("Update() = %+v", updated.Ticket)
	}

	listed, err := queryRepo.List(ctx, ListInput{ProjectID: projectID})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("List() = %+v", listed)
	}

	got, err := queryRepo.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Title != updated.Ticket.Title || got.ExternalRef != "ASE-183-refined" || got.BudgetUSD != 25 {
		t.Fatalf("Get() = %+v", got)
	}
}

func TestCommentRepositoryTracksRevisions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	projectID, statuses := createRepoTicketTestProject(ctx, t, client)
	todoID := findStatusIDByName(t, statuses, "Todo")

	commandRepo := NewCommandRepository(client)
	commentRepo := NewCommentRepository(client)
	ticketItem := createTicketForRepoTest(ctx, t, commandRepo, projectID, todoID, "Comment target")

	created, err := commentRepo.AddComment(ctx, AddCommentInput{
		TicketID:  ticketItem.ID,
		Body:      "First draft",
		CreatedBy: "codex",
	})
	if err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}

	updated, err := commentRepo.UpdateComment(ctx, UpdateCommentInput{
		TicketID:   ticketItem.ID,
		CommentID:  created.ID,
		Body:       "Second draft",
		EditedBy:   "reviewer",
		EditReason: "tighten wording",
	})
	if err != nil {
		t.Fatalf("UpdateComment() error = %v", err)
	}
	if updated.EditCount != 1 || updated.LastEditedBy == nil || *updated.LastEditedBy != "reviewer" {
		t.Fatalf("UpdateComment() = %+v", updated)
	}

	comments, err := commentRepo.ListComments(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("ListComments() error = %v", err)
	}
	if len(comments) != 1 || comments[0].BodyMarkdown != "Second draft" {
		t.Fatalf("ListComments() = %+v", comments)
	}

	revisions, err := commentRepo.ListCommentRevisions(ctx, ticketItem.ID, created.ID)
	if err != nil {
		t.Fatalf("ListCommentRevisions() error = %v", err)
	}
	if len(revisions) != 2 || revisions[0].BodyMarkdown != "First draft" || revisions[1].EditReason == nil || *revisions[1].EditReason != "tighten wording" {
		t.Fatalf("ListCommentRevisions() = %+v", revisions)
	}

	deleted, err := commentRepo.RemoveComment(ctx, ticketItem.ID, created.ID)
	if err != nil {
		t.Fatalf("RemoveComment() error = %v", err)
	}
	if deleted.DeletedCommentID != created.ID {
		t.Fatalf("RemoveComment() = %+v", deleted)
	}

	commentsAfterDelete, err := commentRepo.ListComments(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("ListComments(after delete) error = %v", err)
	}
	if len(commentsAfterDelete) != 1 || !commentsAfterDelete[0].IsDeleted {
		t.Fatalf("ListComments(after delete) = %+v", commentsAfterDelete)
	}
}

func TestLinkRepositoryDependenciesAndExternalLinks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	projectID, statuses := createRepoTicketTestProject(ctx, t, client)
	todoID := findStatusIDByName(t, statuses, "Todo")

	commandRepo := NewCommandRepository(client)
	queryRepo := NewQueryRepository(client)
	linkRepo := NewLinkRepository(client)
	parent := createTicketForRepoTest(ctx, t, commandRepo, projectID, todoID, "Parent")
	child := createTicketForRepoTest(ctx, t, commandRepo, projectID, todoID, "Child")

	dependency, err := linkRepo.AddDependency(ctx, AddDependencyInput{
		TicketID:       child.ID,
		TargetTicketID: parent.ID,
		Type:           DependencyTypeBlocks,
	})
	if err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}
	if dependency.Target.ID != parent.ID || dependency.Type != DependencyTypeBlocks {
		t.Fatalf("AddDependency() = %+v", dependency)
	}

	firstLink, err := linkRepo.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   child.ID,
		LinkType:   domain.ExternalLinkTypeGithubPR,
		URL:        "https://github.com/pacificstudio/openase/pull/183",
		ExternalID: "183",
		Title:      "ASE-183 PR",
		Status:     "open",
	})
	if err != nil {
		t.Fatalf("AddExternalLink(first) error = %v", err)
	}
	secondLink, err := linkRepo.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   child.ID,
		LinkType:   domain.ExternalLinkTypeGithubIssue,
		URL:        "https://github.com/pacificstudio/openase/issues/183",
		ExternalID: "issue-183",
		Title:      "ASE-183 issue",
		Status:     "open",
	})
	if err != nil {
		t.Fatalf("AddExternalLink(second) error = %v", err)
	}

	got, err := queryRepo.Get(ctx, child.ID)
	if err != nil {
		t.Fatalf("Get() after link/dependency error = %v", err)
	}
	if len(got.Dependencies) != 1 || len(got.ExternalLinks) != 2 || got.ExternalRef != "183" {
		t.Fatalf("Get() after link/dependency = %+v", got)
	}

	removedDependency, err := linkRepo.RemoveDependency(ctx, child.ID, dependency.ID)
	if err != nil {
		t.Fatalf("RemoveDependency() error = %v", err)
	}
	if removedDependency.DeletedDependencyID != dependency.ID {
		t.Fatalf("RemoveDependency() = %+v", removedDependency)
	}

	removedLink, err := linkRepo.RemoveExternalLink(ctx, child.ID, firstLink.ID)
	if err != nil {
		t.Fatalf("RemoveExternalLink(first) error = %v", err)
	}
	if removedLink.DeletedExternalLinkID != firstLink.ID {
		t.Fatalf("RemoveExternalLink(first) = %+v", removedLink)
	}

	gotAfterFirstDelete, err := queryRepo.Get(ctx, child.ID)
	if err != nil {
		t.Fatalf("Get() after first external link delete error = %v", err)
	}
	if len(gotAfterFirstDelete.ExternalLinks) != 1 || gotAfterFirstDelete.ExternalRef != secondLink.ExternalID || len(gotAfterFirstDelete.Dependencies) != 0 {
		t.Fatalf("Get() after first external link delete = %+v", gotAfterFirstDelete)
	}

	if _, err := linkRepo.RemoveExternalLink(ctx, child.ID, secondLink.ID); err != nil {
		t.Fatalf("RemoveExternalLink(second) error = %v", err)
	}
	gotAfterAllDeletes, err := queryRepo.Get(ctx, child.ID)
	if err != nil {
		t.Fatalf("Get() after all external link deletes error = %v", err)
	}
	if len(gotAfterAllDeletes.ExternalLinks) != 0 || gotAfterAllDeletes.ExternalRef != "" {
		t.Fatalf("Get() after all external link deletes = %+v", gotAfterAllDeletes)
	}
}

func TestRuntimeRepositoryPickupDiagnosisBlockedDependency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	projectID, statuses := createRepoTicketTestProject(ctx, t, client)
	todoID := findStatusIDByName(t, statuses, "Todo")

	commandRepo := NewCommandRepository(client)
	linkRepo := NewLinkRepository(client)
	runtimeRepo := NewRuntimeRepository(client)
	blocker := createTicketForRepoTest(ctx, t, commandRepo, projectID, todoID, "Blocked by me")
	target := createTicketForRepoTest(ctx, t, commandRepo, projectID, todoID, "Wait on blocker")

	if _, err := linkRepo.AddDependency(ctx, AddDependencyInput{
		TicketID:       blocker.ID,
		TargetTicketID: target.ID,
		Type:           DependencyTypeBlocks,
	}); err != nil {
		t.Fatalf("AddDependency() error = %v", err)
	}

	diagnosis, err := runtimeRepo.GetPickupDiagnosis(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetPickupDiagnosis() error = %v", err)
	}
	if diagnosis.PrimaryReasonCode != domain.PickupDiagnosisReasonBlockedDependency || diagnosis.State != domain.PickupDiagnosisStateBlocked {
		t.Fatalf("GetPickupDiagnosis() = %+v", diagnosis)
	}
	if len(diagnosis.BlockedBy) != 1 || diagnosis.BlockedBy[0].ID != blocker.ID {
		t.Fatalf("GetPickupDiagnosis().BlockedBy = %+v", diagnosis.BlockedBy)
	}
	if len(diagnosis.Reasons) == 0 || diagnosis.Reasons[0].Code != domain.PickupDiagnosisReasonBlockedDependency {
		t.Fatalf("GetPickupDiagnosis().Reasons = %+v", diagnosis.Reasons)
	}
}

func TestRuntimeRepositoryLifecycleHookRuntimeWithoutWorkflow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := openTestEntClient(t)
	projectID, statuses := createRepoTicketTestProject(ctx, t, client)
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	commandRepo := NewCommandRepository(client)
	runtimeRepo := NewRuntimeRepository(client)
	ticketItem := createTicketForRepoTest(ctx, t, commandRepo, projectID, todoID, "Hook runtime target")

	orgID, err := client.Project.Get(ctx, projectID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(orgID.OrganizationID).
		SetName("local-machine").
		SetHost("127.0.0.1").
		SetPort(22).
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(orgID.OrganizationID).
		SetMachineID(machine.ID).
		SetName("Codex").
		SetAdapterType("codex-app-server").
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agent, err := client.Agent.Create().
		SetProjectID(projectID).
		SetProviderID(provider.ID).
		SetName("Developer").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	workflowItem, err := client.Workflow.Create().
		SetProjectID(projectID).
		SetAgentID(agent.ID).
		SetName("Fullstack Developer").
		SetType("Fullstack Developer").
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	runID := uuid.New()
	if _, err := client.AgentRun.Create().
		SetID(runID).
		SetAgentID(agent.ID).
		SetWorkflowID(workflowItem.ID).
		SetProviderID(provider.ID).
		SetTicketID(ticketItem.ID).
		SetStatus("executing").
		SetCreatedAt(time.Now().UTC()).
		Save(ctx); err != nil {
		t.Fatalf("create run: %v", err)
	}

	data, err := runtimeRepo.LoadLifecycleHookRuntimeData(ctx, ticketItem.ID, runID, nil)
	if err != nil {
		t.Fatalf("LoadLifecycleHookRuntimeData() error = %v", err)
	}
	if data.TicketID != ticketItem.ID {
		t.Fatalf("LoadLifecycleHookRuntimeData() = %+v", data)
	}
	if data.ProjectID != uuid.Nil || data.AgentID != uuid.Nil || data.WorkflowType != "" || len(data.Workspaces) != 0 {
		t.Fatalf("LoadLifecycleHookRuntimeData() unexpected workflow/workspaces = %+v", data)
	}
}
