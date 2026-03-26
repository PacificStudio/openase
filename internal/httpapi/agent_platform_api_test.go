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
