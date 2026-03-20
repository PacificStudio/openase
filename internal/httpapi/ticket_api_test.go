package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
)

func TestTicketRoutesCRUDAndDependencies(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
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
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statusSvc := ticketstatus.NewService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")
	doneID := findStatusIDByName(t, statuses, "Done")

	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("coding-workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		SetPickupStatusID(backlogID).
		SetFinishStatusID(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	parentCreateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{
			"title":       "Implement ticket API",
			"description": "cover create/list/detail/update",
			"priority":    "high",
			"type":        "epic",
			"workflow_id": workflowItem.ID.String(),
			"created_by":  "user:gary",
			"budget_usd":  3.5,
		},
		http.StatusCreated,
		&parentCreateResp,
	)
	if parentCreateResp.Ticket.Identifier != "ASE-1" {
		t.Fatalf("expected first identifier ASE-1, got %+v", parentCreateResp.Ticket)
	}
	if parentCreateResp.Ticket.StatusName != "Backlog" || parentCreateResp.Ticket.CreatedBy != "user:gary" {
		t.Fatalf("unexpected parent create response: %+v", parentCreateResp.Ticket)
	}

	childCreateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{
			"title":            "Implement dependency routes",
			"description":      "child ticket",
			"parent_ticket_id": parentCreateResp.Ticket.ID,
			"type":             "feature",
		},
		http.StatusCreated,
		&childCreateResp,
	)
	if childCreateResp.Ticket.Identifier != "ASE-2" || childCreateResp.Ticket.CreatedBy != "user:api" {
		t.Fatalf("unexpected child create response: %+v", childCreateResp.Ticket)
	}
	if childCreateResp.Ticket.Parent == nil || childCreateResp.Ticket.Parent.ID != parentCreateResp.Ticket.ID {
		t.Fatalf("expected child to point at parent, got %+v", childCreateResp.Ticket)
	}
	if len(childCreateResp.Ticket.Dependencies) != 1 || childCreateResp.Ticket.Dependencies[0].Type != "sub_issue" {
		t.Fatalf("expected child create to add sub_issue dependency, got %+v", childCreateResp.Ticket.Dependencies)
	}

	listResp := struct {
		Tickets []ticketResponse `json:"tickets"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/tickets?priority=high", project.ID),
		nil,
		http.StatusOK,
		&listResp,
	)
	if len(listResp.Tickets) != 1 || listResp.Tickets[0].ID != parentCreateResp.Ticket.ID {
		t.Fatalf("expected priority filter to return only parent, got %+v", listResp.Tickets)
	}

	parentDetailResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", parentCreateResp.Ticket.ID),
		nil,
		http.StatusOK,
		&parentDetailResp,
	)
	if len(parentDetailResp.Ticket.Children) != 1 || parentDetailResp.Ticket.Children[0].ID != childCreateResp.Ticket.ID {
		t.Fatalf("expected parent detail to expose child, got %+v", parentDetailResp.Ticket)
	}

	childUpdateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", childCreateResp.Ticket.ID),
		map[string]any{
			"title":            "Implement dependency HTTP routes",
			"priority":         "low",
			"status_id":        doneID.String(),
			"external_ref":     "BetterAndBetterII/openase#6",
			"budget_usd":       1.25,
			"parent_ticket_id": "",
		},
		http.StatusOK,
		&childUpdateResp,
	)
	if childUpdateResp.Ticket.Parent != nil || len(childUpdateResp.Ticket.Dependencies) != 0 {
		t.Fatalf("expected patch to clear sub_issue link, got %+v", childUpdateResp.Ticket)
	}
	if childUpdateResp.Ticket.StatusID != doneID.String() || childUpdateResp.Ticket.ExternalRef != "BetterAndBetterII/openase#6" {
		t.Fatalf("unexpected child patch response: %+v", childUpdateResp.Ticket)
	}

	peerCreateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{"title": "Peer ticket"},
		http.StatusCreated,
		&peerCreateResp,
	)

	blockDependencyResp := struct {
		Dependency ticketDependencyResponse `json:"dependency"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/dependencies", parentCreateResp.Ticket.ID),
		map[string]any{
			"target_ticket_id": peerCreateResp.Ticket.ID,
			"type":             "blocks",
		},
		http.StatusCreated,
		&blockDependencyResp,
	)
	if blockDependencyResp.Dependency.Type != "blocks" || blockDependencyResp.Dependency.Target.ID != peerCreateResp.Ticket.ID {
		t.Fatalf("unexpected blocks dependency response: %+v", blockDependencyResp.Dependency)
	}

	parentAfterBlocksResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", parentCreateResp.Ticket.ID),
		nil,
		http.StatusOK,
		&parentAfterBlocksResp,
	)
	if len(parentAfterBlocksResp.Ticket.Dependencies) != 1 || parentAfterBlocksResp.Ticket.Dependencies[0].Type != "blocks" {
		t.Fatalf("expected parent detail to expose blocks dependency, got %+v", parentAfterBlocksResp.Ticket.Dependencies)
	}

	subIssueDependencyResp := struct {
		Dependency ticketDependencyResponse `json:"dependency"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/dependencies", peerCreateResp.Ticket.ID),
		map[string]any{
			"target_ticket_id": parentCreateResp.Ticket.ID,
			"type":             "sub_issue",
		},
		http.StatusCreated,
		&subIssueDependencyResp,
	)
	if subIssueDependencyResp.Dependency.Type != "sub_issue" {
		t.Fatalf("unexpected sub_issue dependency response: %+v", subIssueDependencyResp.Dependency)
	}

	peerAfterSubIssueResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", peerCreateResp.Ticket.ID),
		nil,
		http.StatusOK,
		&peerAfterSubIssueResp,
	)
	if peerAfterSubIssueResp.Ticket.Parent == nil || peerAfterSubIssueResp.Ticket.Parent.ID != parentCreateResp.Ticket.ID {
		t.Fatalf("expected peer parent to be synced from sub_issue dependency, got %+v", peerAfterSubIssueResp.Ticket)
	}

	deleteResp := ticketservice.DeleteDependencyResult{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/tickets/%s/dependencies/%s", peerCreateResp.Ticket.ID, subIssueDependencyResp.Dependency.ID),
		nil,
		http.StatusOK,
		&deleteResp,
	)
	if deleteResp.DeletedDependencyID.String() != subIssueDependencyResp.Dependency.ID {
		t.Fatalf("unexpected dependency delete response: %+v", deleteResp)
	}

	peerAfterDeleteResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", peerCreateResp.Ticket.ID),
		nil,
		http.StatusOK,
		&peerAfterDeleteResp,
	)
	if peerAfterDeleteResp.Ticket.Parent != nil {
		t.Fatalf("expected sub_issue delete to clear parent, got %+v", peerAfterDeleteResp.Ticket)
	}
}

func TestTicketDetailRouteIncludesRepoScopesAndTicketActivity(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40026},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver()),
		nil,
	)

	ctx := context.Background()
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
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-9").
		SetTitle("Build ticket detail page").
		SetDescription("Expose PR status, activity, and hook history in one place.").
		SetStatusID(backlogID).
		SetCreatedBy("user:codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	backendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("backend").
		SetRepositoryURL("https://github.com/acme/backend.git").
		SetDefaultBranch("main").
		SetIsPrimary(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create backend repo: %v", err)
	}
	frontendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("frontend").
		SetRepositoryURL("https://github.com/acme/frontend.git").
		SetDefaultBranch("develop").
		SetIsPrimary(false).
		Save(ctx)
	if err != nil {
		t.Fatalf("create frontend repo: %v", err)
	}

	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(frontendRepo.ID).
		SetBranchName("agent/codex/ASE-9").
		SetPullRequestURL("https://github.com/acme/frontend/pull/9").
		SetPrStatus("open").
		SetCiStatus("pending").
		SetIsPrimaryScope(true).
		Save(ctx); err != nil {
		t.Fatalf("create frontend repo scope: %v", err)
	}
	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(backendRepo.ID).
		SetBranchName("main").
		SetPrStatus("approved").
		SetCiStatus("passing").
		SetIsPrimaryScope(false).
		Save(ctx); err != nil {
		t.Fatalf("create backend repo scope: %v", err)
	}

	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetEventType("agent.output").
		SetMessage("Opened frontend PR #9").
		SetMetadata(map[string]any{"stream": "stdout"}).
		Save(ctx); err != nil {
		t.Fatalf("create activity event: %v", err)
	}
	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetEventType("hook.failed").
		SetMessage("on_complete failed for run-tests.sh").
		SetMetadata(map[string]any{"hook_name": "on_complete", "command": "run-tests.sh"}).
		Save(ctx); err != nil {
		t.Fatalf("create hook event: %v", err)
	}

	var payload struct {
		Ticket      ticketResponse                  `json:"ticket"`
		RepoScopes  []ticketRepoScopeDetailResponse `json:"repo_scopes"`
		Activity    []activityEventResponse         `json:"activity"`
		HookHistory []activityEventResponse         `json:"hook_history"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/tickets/%s/detail", project.ID, ticketItem.ID),
		nil,
		http.StatusOK,
		&payload,
	)

	if payload.Ticket.ID != ticketItem.ID.String() || payload.Ticket.Identifier != "ASE-9" {
		t.Fatalf("unexpected ticket payload: %+v", payload.Ticket)
	}
	if len(payload.RepoScopes) != 2 || payload.RepoScopes[0].Repo == nil || payload.RepoScopes[0].Repo.Name != "frontend" {
		t.Fatalf("expected repo scopes with repo metadata, got %+v", payload.RepoScopes)
	}
	if payload.RepoScopes[0].PullRequestURL == nil || *payload.RepoScopes[0].PullRequestURL != "https://github.com/acme/frontend/pull/9" {
		t.Fatalf("expected frontend pull request URL, got %+v", payload.RepoScopes[0])
	}
	if len(payload.Activity) != 2 {
		t.Fatalf("expected two ticket activity events, got %+v", payload.Activity)
	}
	if len(payload.HookHistory) != 1 || payload.HookHistory[0].EventType != "hook.failed" {
		t.Fatalf("expected hook history to filter hook-tagged events, got %+v", payload.HookHistory)
	}
}

func TestTicketRouteStatusChangeClearsAssignmentAndReleasesAgent(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40024},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
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
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")

	assignedAgent, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(provider.ID).
		SetName("coding-01").
		SetStatus(entagent.StatusClaimed).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Implement pickup/finish state transitions").
		SetStatusID(todoID).
		SetAssignedAgentID(assignedAgent.ID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := client.Agent.UpdateOneID(assignedAgent.ID).
		SetCurrentTicketID(ticketItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("claim agent for ticket: %v", err)
	}

	titleOnlyResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		map[string]any{"title": "Implement ticket pickup/finish transitions"},
		http.StatusOK,
		&titleOnlyResp,
	)

	ticketAfterTitleOnly, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after title update: %v", err)
	}
	if ticketAfterTitleOnly.AssignedAgentID == nil || *ticketAfterTitleOnly.AssignedAgentID != assignedAgent.ID {
		t.Fatalf("expected non-status update to keep assignment, got %+v", ticketAfterTitleOnly.AssignedAgentID)
	}
	agentAfterTitleOnly, err := client.Agent.Get(ctx, assignedAgent.ID)
	if err != nil {
		t.Fatalf("reload agent after title update: %v", err)
	}
	if agentAfterTitleOnly.Status != entagent.StatusClaimed || agentAfterTitleOnly.CurrentTicketID == nil || *agentAfterTitleOnly.CurrentTicketID != ticketItem.ID {
		t.Fatalf("expected non-status update to keep agent claim, got %+v", agentAfterTitleOnly)
	}

	statusResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		map[string]any{"status_id": doneID.String()},
		http.StatusOK,
		&statusResp,
	)

	ticketAfterStatusChange, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after status update: %v", err)
	}
	if ticketAfterStatusChange.StatusID != doneID {
		t.Fatalf("expected ticket status %s, got %s", doneID, ticketAfterStatusChange.StatusID)
	}
	if ticketAfterStatusChange.AssignedAgentID != nil {
		t.Fatalf("expected status update to clear assignment, got %+v", ticketAfterStatusChange.AssignedAgentID)
	}

	agentAfterStatusChange, err := client.Agent.Get(ctx, assignedAgent.ID)
	if err != nil {
		t.Fatalf("reload agent after status update: %v", err)
	}
	if agentAfterStatusChange.Status != entagent.StatusIdle || agentAfterStatusChange.CurrentTicketID != nil {
		t.Fatalf("expected status update to release agent, got %+v", agentAfterStatusChange)
	}
}

func TestTicketRoutesPublishSSEEvents(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40025},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		nil,
		nil,
		nil,
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	ctx := context.Background()
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
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	doneID := findStatusIDByName(t, statuses, "Done")

	createResponse, cancelCreate := openSSERequest(t, testServer.URL+fmt.Sprintf("/api/v1/projects/%s/tickets/stream", project.ID))
	t.Cleanup(func() {
		if err := createResponse.Body.Close(); err != nil {
			t.Errorf("close create response body: %v", err)
		}
	})
	createPayload := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{
			"title":       "Implement board realtime updates",
			"description": "publish ticket.created when a ticket is added",
		},
		http.StatusCreated,
		&createPayload,
	)
	createBody := readSSEBody(t, createResponse, cancelCreate)
	if !strings.Contains(createBody, "event: ticket.created\n") {
		t.Fatalf("expected ticket.created frame, got %q", createBody)
	}
	if !strings.Contains(createBody, createPayload.Ticket.Identifier) {
		t.Fatalf("expected created ticket identifier in SSE payload, got %q", createBody)
	}

	updateResponse, cancelUpdate := openSSERequest(t, testServer.URL+fmt.Sprintf("/api/v1/projects/%s/tickets/stream", project.ID))
	t.Cleanup(func() {
		if err := updateResponse.Body.Close(); err != nil {
			t.Errorf("close update response body: %v", err)
		}
	})
	updatePayload := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", createPayload.Ticket.ID),
		map[string]any{"status_id": doneID.String()},
		http.StatusOK,
		&updatePayload,
	)
	updateBody := readSSEBody(t, updateResponse, cancelUpdate)
	if !strings.Contains(updateBody, "event: ticket.status_changed\n") {
		t.Fatalf("expected ticket.status_changed frame, got %q", updateBody)
	}
	if !strings.Contains(updateBody, doneID.String()) {
		t.Fatalf("expected updated status id in SSE payload, got %q", updateBody)
	}
}

func TestTicketBudgetUpdatesSyncBudgetExhaustedPauseState(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
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
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Adjust retry budget").
		SetStatusID(todoID).
		SetBudgetUsd(5).
		SetCostAmount(5).
		SetRetryPaused(true).
		SetPauseReason(ticketing.PauseReasonBudgetExhausted.String()).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	increaseResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		map[string]any{"budget_usd": 8.0},
		http.StatusOK,
		&increaseResp,
	)

	if increaseResp.Ticket.BudgetUSD != 8 || increaseResp.Ticket.CostAmount != 5 {
		t.Fatalf("unexpected ticket budget fields after increase: %+v", increaseResp.Ticket)
	}
	if increaseResp.Ticket.RetryPaused || increaseResp.Ticket.PauseReason != "" {
		t.Fatalf("expected budget increase to resume retry, got %+v", increaseResp.Ticket)
	}

	ticketAfterIncrease, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after increase: %v", err)
	}
	if ticketAfterIncrease.RetryPaused || ticketAfterIncrease.PauseReason != "" {
		t.Fatalf("expected budget increase to clear budget pause, got %+v", ticketAfterIncrease)
	}

	decreaseResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		map[string]any{"budget_usd": 4.0},
		http.StatusOK,
		&decreaseResp,
	)

	if !decreaseResp.Ticket.RetryPaused || decreaseResp.Ticket.PauseReason != ticketing.PauseReasonBudgetExhausted.String() {
		t.Fatalf("expected lowered budget to pause retry again, got %+v", decreaseResp.Ticket)
	}

	ticketAfterDecrease, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after decrease: %v", err)
	}
	if !ticketAfterDecrease.RetryPaused || ticketAfterDecrease.PauseReason != ticketing.PauseReasonBudgetExhausted.String() {
		t.Fatalf("expected lowered budget to persist budget pause, got %+v", ticketAfterDecrease)
	}
}
