package orchestrator

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func TestTicketWorkspaceResetServiceResetTicketWorkspaceCleansExistingWorkspace(t *testing.T) {
	client := testPostgres.NewIsolatedEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().SetName("Acme").SetSlug("acme").Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	todoID := uuid.UUID{}
	for _, status := range statuses {
		if status.Name == "Todo" {
			todoID = status.ID
			break
		}
	}
	if todoID == uuid.Nil {
		t.Fatalf("todo status not found")
	}
	doneID := todoID
	for _, status := range statuses {
		if status.Name == "Done" {
			doneID = status.ID
			break
		}
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local-devbox").
		SetHost(catalogdomain.LocalMachineHost).
		SetPort(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(providerItem.ID).
		SetName("coder").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("coding-workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-13").
		SetTitle("Reuse workspace on rerun").
		SetStatusID(todoID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem, err := client.AgentRun.Create().
		SetTicketID(ticketItem.ID).
		SetWorkflowID(workflowItem.ID).
		SetAgentID(agentItem.ID).
		SetProviderID(providerItem.ID).
		SetStatus("completed").
		Save(ctx)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	workspaceRoot := filepath.Join(t.TempDir(), "workspace-root")
	repoPath := filepath.Join(workspaceRoot, "repo")
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		t.Fatalf("mkdir repo path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "DIRTY.txt"), []byte("dirty\n"), 0o600); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}
	repoItem, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("openase").
		SetRepositoryURL("https://example.com/openase.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}

	workspaceRecord, err := client.TicketRepoWorkspace.Create().
		SetTicketID(ticketItem.ID).
		SetAgentRunID(runItem.ID).
		SetRepoID(repoItem.ID).
		SetWorkspaceRoot(workspaceRoot).
		SetRepoPath(repoPath).
		SetBranchName("scratch").
		SetState(entticketrepoworkspace.StateReady).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket repo workspace: %v", err)
	}

	service := NewTicketWorkspaceResetService(
		client,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
	)

	if err := service.ResetTicketWorkspace(ctx, ticketItem.ID); err != nil {
		t.Fatalf("ResetTicketWorkspace() error = %v", err)
	}

	if _, err := os.Stat(workspaceRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected workspace root to be removed, got err=%v", err)
	}

	workspaceAfter, err := client.TicketRepoWorkspace.Get(ctx, workspaceRecord.ID)
	if err != nil {
		t.Fatalf("reload ticket repo workspace: %v", err)
	}
	if workspaceAfter.State != entticketrepoworkspace.StateCleaned || workspaceAfter.CleanedAt == nil {
		t.Fatalf("expected workspace cleaned, got %+v", workspaceAfter)
	}
}

func TestTicketWorkspaceResetServiceResetTicketWorkspaceRejectsActiveRun(t *testing.T) {
	client := testPostgres.NewIsolatedEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().SetName("Acme").SetSlug("acme").Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	todoID := uuid.UUID{}
	for _, status := range statuses {
		if status.Name == "Todo" {
			todoID = status.ID
			break
		}
	}
	if todoID == uuid.Nil {
		t.Fatalf("todo status not found")
	}
	doneID := todoID
	for _, status := range statuses {
		if status.Name == "Done" {
			doneID = status.ID
			break
		}
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-14").
		SetTitle("Preserve dirty worktree").
		SetStatusID(todoID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local-devbox").
		SetHost(catalogdomain.LocalMachineHost).
		SetPort(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(providerItem.ID).
		SetName("coder").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("coding-workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	runItem, err := client.AgentRun.Create().
		SetTicketID(ticketItem.ID).
		SetWorkflowID(workflowItem.ID).
		SetAgentID(agentItem.ID).
		SetProviderID(providerItem.ID).
		SetStatus("executing").
		Save(ctx)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).SetCurrentRunID(runItem.ID).Save(ctx); err != nil {
		t.Fatalf("attach current run: %v", err)
	}

	service := NewTicketWorkspaceResetService(
		client,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
	)

	err = service.ResetTicketWorkspace(ctx, ticketItem.ID)
	var conflictErr TicketWorkspaceResetConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("ResetTicketWorkspace() error = %v, want conflict error", err)
	}
}
