package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/orchestrator"
)

func TestHandleResetTicketWorkspaceDeleteFailureReturnsActionableError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can delete protected dirs; run as non-root")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40024},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
		WithTicketWorkspaceResetter(
			orchestrator.NewTicketWorkspaceResetService(
				client,
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				nil,
			),
		),
	)

	ctx := context.Background()
	org, err := client.Organization.Create().SetName("Acme").SetSlug("acme").Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().SetOrganizationID(org.ID).SetName("OpenASE").SetSlug("openase").Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
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
		SetIdentifier("ASE-499").
		SetTitle("Reset blocked").
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
		SetStatus(entagentrun.StatusCompleted).
		Save(ctx)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	root := filepath.Join(home, ".openase", "workspace")
	workspaceRoot := filepath.Join(root, "acme", "openase", "ASE-499")
	protected := filepath.Join(workspaceRoot, "openase", "web", "node_modules")
	if err := os.MkdirAll(protected, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(protected, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(protected, 0o755) })

	repoItem, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("openase").
		SetRepositoryURL("https://example.com/openase.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	workspaceItem, err := client.TicketRepoWorkspace.Create().
		SetTicketID(ticketItem.ID).
		SetAgentRunID(runItem.ID).
		SetRepoID(repoItem.ID).
		SetWorkspaceRoot(workspaceRoot).
		SetRepoPath(filepath.Join(workspaceRoot, "openase")).
		SetBranchName("scratch").
		SetState(entticketrepoworkspace.StateReady).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workspace row: %v", err)
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/workspace/reset", ticketItem.ID),
		"",
	)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "WORKSPACE_RESET_DELETE_FAILED") {
		t.Fatalf("expected delete failed conflict, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "errSymlink") || strings.Contains(body, "PANIC=Error") {
		t.Fatalf("response leaked internal error: %s", body)
	}

	workspaceAfter, err := client.TicketRepoWorkspace.Get(ctx, workspaceItem.ID)
	if err != nil {
		t.Fatalf("reload workspace: %v", err)
	}
	if workspaceAfter.State != entticketrepoworkspace.StateFailed {
		t.Fatalf("expected failed state, got %+v", workspaceAfter)
	}
	if strings.TrimSpace(workspaceAfter.LastError) == "" {
		t.Fatal("expected persisted LastError")
	}
	if strings.Contains(workspaceAfter.LastError, "errSymlink") {
		t.Fatalf("LastError leaked internal error: %q", workspaceAfter.LastError)
	}
}