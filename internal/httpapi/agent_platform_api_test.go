package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestAgentPlatformTicketRoutesRespectScopesAndBoundaries(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, doneTicketID := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(client)

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	issued, err := platformService.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  currentTicketID,
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	listResp := struct {
		Tickets []ticketResponse `json:"tickets"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/projects/%s/tickets", projectID),
		nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&listResp,
	)
	if len(listResp.Tickets) != 2 {
		t.Fatalf("expected two tickets in list, got %+v", listResp.Tickets)
	}

	createResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/projects/%s/tickets", projectID),
		map[string]any{
			"title":       "Agent-created follow-up",
			"description": "split out integration coverage",
			"priority":    "high",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusCreated,
		&createResp,
	)
	if createResp.Ticket.CreatedBy != "agent:coding-01" {
		t.Fatalf("expected agent created_by marker, got %+v", createResp.Ticket)
	}

	updateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
		map[string]any{
			"description":  "captured follow-up implementation notes",
			"external_ref": "BetterAndBetterII/openase#37",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&updateResp,
	)
	if updateResp.Ticket.Description != "captured follow-up implementation notes" || updateResp.Ticket.CreatedBy != "agent:coding-01" {
		t.Fatalf("unexpected updated ticket payload: %+v", updateResp.Ticket)
	}

	createCommentResp := struct {
		Comment ticketCommentResponse `json:"comment"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/tickets/%s/comments", currentTicketID),
		map[string]any{
			"body": "## Codex Workpad\n\nProgress\n- inspected current code",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusCreated,
		&createCommentResp,
	)
	if createCommentResp.Comment.CreatedBy != "agent:coding-01" {
		t.Fatalf("unexpected created comment payload: %+v", createCommentResp.Comment)
	}

	listCommentsResp := struct {
		Comments []ticketCommentResponse `json:"comments"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/tickets/%s/comments", currentTicketID),
		nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&listCommentsResp,
	)
	if len(listCommentsResp.Comments) != 1 || listCommentsResp.Comments[0].ID != createCommentResp.Comment.ID {
		t.Fatalf("unexpected listed comments payload: %+v", listCommentsResp.Comments)
	}

	updateCommentResp := struct {
		Comment ticketCommentResponse `json:"comment"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/tickets/%s/comments/%s", currentTicketID, createCommentResp.Comment.ID),
		map[string]any{
			"body": "## Codex Workpad\n\nValidation\n- npm test",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&updateCommentResp,
	)
	if updateCommentResp.Comment.Body != "## Codex Workpad\n\nValidation\n- npm test" || updateCommentResp.Comment.LastEditedBy == nil || *updateCommentResp.Comment.LastEditedBy != "agent:coding-01" {
		t.Fatalf("unexpected updated comment payload: %+v", updateCommentResp.Comment)
	}

	statusUpdateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
		map[string]any{
			"status_name": "In Progress",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&statusUpdateResp,
	)
	if statusUpdateResp.Ticket.StatusName != "In Progress" || statusUpdateResp.Ticket.CreatedBy != "agent:coding-01" {
		t.Fatalf("unexpected status update payload: %+v", statusUpdateResp.Ticket)
	}

	usageResp := struct {
		Ticket         ticketResponse             `json:"ticket"`
		Applied        ticketservice.AppliedUsage `json:"applied"`
		BudgetExceeded bool                       `json:"budget_exceeded"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/tickets/%s/usage", currentTicketID),
		map[string]any{
			"input_tokens":  120,
			"output_tokens": 45,
			"cost_usd":      0.21,
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&usageResp,
	)
	if usageResp.Applied.InputTokens != 120 || usageResp.Applied.OutputTokens != 45 || usageResp.BudgetExceeded {
		t.Fatalf("unexpected usage response: %+v", usageResp)
	}
	if usageResp.Ticket.CostTokensInput != 120 || usageResp.Ticket.CostTokensOutput != 45 || usageResp.Ticket.CostAmount != 0.21 {
		t.Fatalf("unexpected ticket usage totals: %+v", usageResp.Ticket)
	}

	forbiddenRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/tickets/%s", doneTicketID),
		`{"description":"should fail"}`,
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token, echo.HeaderContentType: echo.MIMEApplicationJSON},
	)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("expected updating another ticket to return 403, got %d: %s", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	forbiddenCommentRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/tickets/%s/comments", doneTicketID),
		`{"body":"should fail"}`,
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token, echo.HeaderContentType: echo.MIMEApplicationJSON},
	)
	if forbiddenCommentRec.Code != http.StatusForbidden {
		t.Fatalf("expected commenting on another ticket to return 403, got %d: %s", forbiddenCommentRec.Code, forbiddenCommentRec.Body.String())
	}
}

func TestAgentPlatformPrivilegedRoutesRequireExplicitScopes(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(client)

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	defaultToken, err := platformService.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  currentTicketID,
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	privilegedToken, err := platformService.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  currentTicketID,
		Scopes: []string{
			string(agentplatform.ScopeProjectsAddRepo),
			string(agentplatform.ScopeProjectsUpdate),
			string(agentplatform.ScopeTicketsCreate),
			string(agentplatform.ScopeTicketsList),
			string(agentplatform.ScopeTicketsUpdateSelf),
		},
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	forbiddenProjectRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/projects/%s", projectID),
		`{"description":"agent wants to update project"}`,
		map[string]string{echo.HeaderAuthorization: "Bearer " + defaultToken.Token, echo.HeaderContentType: echo.MIMEApplicationJSON},
	)
	if forbiddenProjectRec.Code != http.StatusForbidden {
		t.Fatalf("expected project patch without scope to return 403, got %d: %s", forbiddenProjectRec.Code, forbiddenProjectRec.Body.String())
	}

	projectResp := struct {
		Project projectResponse `json:"project"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/projects/%s", projectID),
		map[string]any{"description": "Updated by agent platform API"},
		map[string]string{echo.HeaderAuthorization: "Bearer " + privilegedToken.Token},
		http.StatusOK,
		&projectResp,
	)
	if projectResp.Project.Description != "Updated by agent platform API" {
		t.Fatalf("unexpected project patch payload: %+v", projectResp.Project)
	}

	forbiddenRepoRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/projects/%s/repos", projectID),
		`{"name":"worker-tools","repository_url":"https://github.com/acme/worker-tools.git"}`,
		map[string]string{echo.HeaderAuthorization: "Bearer " + defaultToken.Token, echo.HeaderContentType: echo.MIMEApplicationJSON},
	)
	if forbiddenRepoRec.Code != http.StatusForbidden {
		t.Fatalf("expected repo create without scope to return 403, got %d: %s", forbiddenRepoRec.Code, forbiddenRepoRec.Body.String())
	}

	repoResp := struct {
		Repo projectRepoResponse `json:"repo"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/projects/%s/repos", projectID),
		map[string]any{
			"name":           "worker-tools",
			"repository_url": "https://github.com/acme/worker-tools.git",
			"default_branch": "main",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + privilegedToken.Token},
		http.StatusCreated,
		&repoResp,
	)
	if repoResp.Repo.Name != "worker-tools" {
		t.Fatalf("unexpected repo create payload: %+v", repoResp.Repo)
	}
}

func TestAgentPlatformHarnessWhitelistConstrainsTokenScopes(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(client)

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	access, err := workflowservice.ParsePlatformAccess(`---
platform_access:
  allowed:
    - "tickets.list"
---
# Dispatcher`)
	if err != nil {
		t.Fatalf("ParsePlatformAccess returned error: %v", err)
	}

	issued, err := platformService.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  currentTicketID,
		ScopeWhitelist: agentplatform.ScopeWhitelist{
			Configured: access.Configured,
			Scopes:     access.Allowed,
		},
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	listResp := struct {
		Tickets []ticketResponse `json:"tickets"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/projects/%s/tickets", projectID),
		nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&listResp,
	)
	if len(listResp.Tickets) != 2 {
		t.Fatalf("expected two tickets in list, got %+v", listResp.Tickets)
	}

	forbiddenCreateRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/projects/%s/tickets", projectID),
		`{"title":"should fail"}`,
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token, echo.HeaderContentType: echo.MIMEApplicationJSON},
	)
	if forbiddenCreateRec.Code != http.StatusForbidden {
		t.Fatalf("expected ticket create without whitelisted scope to return 403, got %d: %s", forbiddenCreateRec.Code, forbiddenCreateRec.Body.String())
	}

	forbiddenUpdateRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
		`{"description":"should fail"}`,
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token, echo.HeaderContentType: echo.MIMEApplicationJSON},
	)
	if forbiddenUpdateRec.Code != http.StatusForbidden {
		t.Fatalf("expected ticket update without whitelisted scope to return 403, got %d: %s", forbiddenUpdateRec.Code, forbiddenUpdateRec.Body.String())
	}
}

func TestAgentPlatformRejectsMissingOrCrossProjectToken(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(client)

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	missingAuthRec := performJSONRequest(t, server, http.MethodGet, fmt.Sprintf("/api/v1/platform/projects/%s/tickets", projectID), "")
	if missingAuthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing auth to return 401, got %d: %s", missingAuthRec.Code, missingAuthRec.Body.String())
	}

	issued, err := platformService.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  currentTicketID,
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	otherProjectRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/projects/%s/tickets", uuid.New()),
		"",
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
	)
	if otherProjectRec.Code != http.StatusForbidden {
		t.Fatalf("expected cross-project access to return 403, got %d: %s", otherProjectRec.Code, otherProjectRec.Body.String())
	}
}

func TestAgentPlatformTicketUpdateAllowsProjectStatuses(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(client)

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, projectID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	reviewID := findStatusIDByName(t, statuses, "In Review")
	inProgressID := findStatusIDByName(t, statuses, "In Progress")

	workflowItem, err := client.Workflow.Create().
		SetProjectID(projectID).
		SetAgentID(agentID).
		SetName("Coding").
		SetType("coding").
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID, reviewID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(currentTicketID).
		SetWorkflowID(workflowItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind current ticket workflow: %v", err)
	}

	issued, err := platformService.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  currentTicketID,
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	forbiddenRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
		fmt.Sprintf(`{"status_id":"%s"}`, uuid.NewString()),
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token, echo.HeaderContentType: echo.MIMEApplicationJSON},
	)
	if forbiddenRec.Code != http.StatusBadRequest {
		t.Fatalf("expected unknown project status to return 400, got %d: %s", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	updateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
		map[string]any{"status_id": inProgressID.String()},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&updateResp,
	)
	if updateResp.Ticket.StatusID != inProgressID.String() {
		t.Fatalf("expected allowed in-project intermediate status %s, got %+v", inProgressID, updateResp.Ticket)
	}

	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
		map[string]any{"status_id": reviewID.String()},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&updateResp,
	)
	if updateResp.Ticket.StatusID != reviewID.String() {
		t.Fatalf("expected allowed review status %s, got %+v", reviewID, updateResp.Ticket)
	}
}

func TestAgentPlatformRouteErrorMappingsAndInvalidPayloads(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(client)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	issued, err := platformService.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  currentTicketID,
		Scopes: []string{
			string(agentplatform.ScopeProjectsAddRepo),
			string(agentplatform.ScopeProjectsUpdate),
			string(agentplatform.ScopeTicketsCreate),
			string(agentplatform.ScopeTicketsList),
			string(agentplatform.ScopeTicketsReportUsage),
			string(agentplatform.ScopeTicketsUpdateSelf),
		},
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	headers := map[string]string{
		echo.HeaderAuthorization: "Bearer " + issued.Token,
		echo.HeaderContentType:   echo.MIMEApplicationJSON,
	}

	for _, testCase := range []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "list invalid project", method: http.MethodGet, target: "/api/v1/platform/projects/not-a-uuid/tickets", wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "create invalid request", method: http.MethodPost, target: fmt.Sprintf("/api/v1/platform/projects/%s/tickets", projectID), body: `{"title":"   "}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "update invalid ticket", method: http.MethodPatch, target: "/api/v1/platform/tickets/not-a-uuid", body: `{"description":"nope"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_TICKET_ID"},
		{name: "update invalid status", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID), body: `{"status_id":"not-a-uuid"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "report usage invalid ticket", method: http.MethodPost, target: "/api/v1/platform/tickets/not-a-uuid/usage", body: `{"input_tokens":1}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_TICKET_ID"},
		{name: "report usage invalid request", method: http.MethodPost, target: fmt.Sprintf("/api/v1/platform/tickets/%s/usage", currentTicketID), body: `{}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "update project invalid project", method: http.MethodPatch, target: "/api/v1/platform/projects/not-a-uuid", body: `{"description":"x"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "update project missing description", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/platform/projects/%s", projectID), body: `{}`, wantStatus: http.StatusBadRequest, wantBody: "description is required"},
		{name: "create repo invalid project", method: http.MethodPost, target: "/api/v1/platform/projects/not-a-uuid/repos", body: `{"name":"worker-tools","repository_url":"https://github.com/acme/worker-tools.git"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "create repo invalid request", method: http.MethodPost, target: fmt.Sprintf("/api/v1/platform/projects/%s/repos", projectID), body: `{"name":"","repository_url":"bad"}`, wantStatus: http.StatusBadRequest, wantBody: "name must not be empty"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequestWithHeaders(t, server, testCase.method, testCase.target, testCase.body, headers)
			if rec.Code != testCase.wantStatus || !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("%s %s = %d %s, want %d containing %q", testCase.method, testCase.target, rec.Code, rec.Body.String(), testCase.wantStatus, testCase.wantBody)
			}
		})
	}
}

func TestAgentPlatformForbiddenBoundaryPaths(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, doneTicketID := seedAgentPlatformHTTPFixture(ctx, t, client)
	projectItem, err := client.Project.Get(ctx, projectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}

	otherProject, err := client.Project.Create().
		SetOrganizationID(projectItem.OrganizationID).
		SetName("Other Project").
		SetSlug("other-project").
		Save(ctx)
	if err != nil {
		t.Fatalf("create other project: %v", err)
	}

	platformService := agentplatform.NewService(client)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	issued, err := platformService.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  currentTicketID,
		Scopes: []string{
			string(agentplatform.ScopeProjectsAddRepo),
			string(agentplatform.ScopeProjectsUpdate),
			string(agentplatform.ScopeTicketsCreate),
			string(agentplatform.ScopeTicketsList),
			string(agentplatform.ScopeTicketsReportUsage),
			string(agentplatform.ScopeTicketsUpdateSelf),
		},
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	headers := map[string]string{
		echo.HeaderAuthorization: "Bearer " + issued.Token,
		echo.HeaderContentType:   echo.MIMEApplicationJSON,
	}

	for _, testCase := range []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "list another project",
			method:     http.MethodGet,
			target:     fmt.Sprintf("/api/v1/platform/projects/%s/tickets", otherProject.ID),
			wantStatus: http.StatusForbidden,
			wantBody:   "AGENT_PROJECT_FORBIDDEN",
		},
		{
			name:       "create ticket in another project",
			method:     http.MethodPost,
			target:     fmt.Sprintf("/api/v1/platform/projects/%s/tickets", otherProject.ID),
			body:       `{"title":"forbidden"}`,
			wantStatus: http.StatusForbidden,
			wantBody:   "AGENT_PROJECT_FORBIDDEN",
		},
		{
			name:       "update another project",
			method:     http.MethodPatch,
			target:     fmt.Sprintf("/api/v1/platform/projects/%s", otherProject.ID),
			body:       `{"description":"forbidden"}`,
			wantStatus: http.StatusForbidden,
			wantBody:   "AGENT_PROJECT_FORBIDDEN",
		},
		{
			name:       "create repo in another project",
			method:     http.MethodPost,
			target:     fmt.Sprintf("/api/v1/platform/projects/%s/repos", otherProject.ID),
			body:       `{"name":"worker-tools","repository_url":"https://github.com/acme/worker-tools.git"}`,
			wantStatus: http.StatusForbidden,
			wantBody:   "AGENT_PROJECT_FORBIDDEN",
		},
		{
			name:       "update another ticket",
			method:     http.MethodPatch,
			target:     fmt.Sprintf("/api/v1/platform/tickets/%s", doneTicketID),
			body:       `{"description":"forbidden"}`,
			wantStatus: http.StatusForbidden,
			wantBody:   "AGENT_TICKET_FORBIDDEN",
		},
		{
			name:       "report usage another ticket",
			method:     http.MethodPost,
			target:     fmt.Sprintf("/api/v1/platform/tickets/%s/usage", doneTicketID),
			body:       `{"input_tokens":1}`,
			wantStatus: http.StatusForbidden,
			wantBody:   "AGENT_TICKET_FORBIDDEN",
		},
		{
			name:       "update conflicting status fields",
			method:     http.MethodPatch,
			target:     fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
			body:       fmt.Sprintf(`{"status_id":"%s","status_name":"Done"}`, uuid.NewString()),
			wantStatus: http.StatusBadRequest,
			wantBody:   "status_id and status_name cannot be provided together",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequestWithHeaders(t, server, testCase.method, testCase.target, testCase.body, headers)
			if rec.Code != testCase.wantStatus || !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("%s %s = %d %s, want %d containing %q", testCase.method, testCase.target, rec.Code, rec.Body.String(), testCase.wantStatus, testCase.wantBody)
			}
		})
	}

	if _, err := client.Ticket.UpdateOneID(currentTicketID).SetProjectID(otherProject.ID).Save(ctx); err != nil {
		t.Fatalf("move current ticket to other project: %v", err)
	}

	for _, testCase := range []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "update own ticket after project mismatch",
			method:     http.MethodPatch,
			target:     fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
			body:       `{"description":"forbidden"}`,
			wantStatus: http.StatusForbidden,
			wantBody:   "AGENT_PROJECT_FORBIDDEN",
		},
		{
			name:       "report usage after project mismatch",
			method:     http.MethodPost,
			target:     fmt.Sprintf("/api/v1/platform/tickets/%s/usage", currentTicketID),
			body:       `{"input_tokens":1}`,
			wantStatus: http.StatusForbidden,
			wantBody:   "AGENT_PROJECT_FORBIDDEN",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequestWithHeaders(t, server, testCase.method, testCase.target, testCase.body, headers)
			if rec.Code != testCase.wantStatus || !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("%s %s = %d %s, want %d containing %q", testCase.method, testCase.target, rec.Code, rec.Body.String(), testCase.wantStatus, testCase.wantBody)
			}
		})
	}
}

func seedAgentPlatformHTTPFixture(ctx context.Context, t *testing.T, client *ent.Client) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
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
		SetMachineID(localMachine.ID).
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
		t.Fatalf("reset statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")

	currentTicket, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-37").
		SetTitle("Agent platform ticket").
		SetStatusID(todoID).
		SetCreatedBy("user:codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create current ticket: %v", err)
	}
	doneTicket, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-38").
		SetTitle("Another ticket").
		SetStatusID(doneID).
		SetCreatedBy("user:codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create other ticket: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(provider.ID).
		SetName("coding-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	return project.ID, agentItem.ID, currentTicket.ID, doneTicket.ID
}

func executeJSONWithHeaders(t *testing.T, server *Server, method string, target string, body any, headers map[string]string, wantStatus int, out any) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("expected %s %s to return %d, got %d with body %s", method, target, wantStatus, rec.Code, rec.Body.String())
	}
	if out == nil {
		return
	}
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

func performJSONRequestWithHeaders(t *testing.T, server *Server, method string, target string, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	return rec
}
