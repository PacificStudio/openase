package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	"github.com/BetterAndBetterII/openase/internal/config"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	"github.com/BetterAndBetterII/openase/internal/provider"
	agentplatformrepo "github.com/BetterAndBetterII/openase/internal/repo/agentplatform"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	scheduledjobrepo "github.com/BetterAndBetterII/openase/internal/repo/scheduledjob"
	workflowrepo "github.com/BetterAndBetterII/openase/internal/repo/workflow"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestAgentPlatformTicketRoutesRespectScopesAndBoundaries(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, doneTicketID := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))
	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), t.TempDir())
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		workflowSvc,
	)

	issued, err := platformService.IssueToken(ctx, agentplatform.IssueInput{
		AgentID:   agentID,
		ProjectID: projectID,
		TicketID:  currentTicketID,
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, projectID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	workflowItem, err := client.Workflow.Create().
		SetProjectID(projectID).
		SetAgentID(agentID).
		SetName("Todo App Coding Workflow").
		SetType("coding").
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
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

	workflowResp := struct {
		Workflows []workflowResponse `json:"workflows"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/projects/%s/workflows", projectID),
		nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&workflowResp,
	)
	if len(workflowResp.Workflows) != 1 || workflowResp.Workflows[0].ID != workflowItem.ID.String() {
		t.Fatalf("unexpected workflows payload: %+v", workflowResp.Workflows)
	}

	currentTicket, err := client.Ticket.Get(ctx, currentTicketID)
	if err != nil {
		t.Fatalf("load current ticket: %v", err)
	}

	getTicketResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
		nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&getTicketResp,
	)
	if getTicketResp.Ticket.ID != currentTicketID.String() {
		t.Fatalf("unexpected get ticket payload by uuid: %+v", getTicketResp.Ticket)
	}

	invalidIdentifierRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicket.Identifier),
		"",
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
	)
	if invalidIdentifierRec.Code != http.StatusBadRequest || !strings.Contains(invalidIdentifierRec.Body.String(), "INVALID_TICKET_ID") {
		t.Fatalf("expected identifier lookup to return INVALID_TICKET_ID, got %d: %s", invalidIdentifierRec.Code, invalidIdentifierRec.Body.String())
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
			"external_ref": "PacificStudio/openase#37",
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
			"body": "## Workpad\n\nProgress\n- inspected current code",
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

	invalidCommentIdentifierRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/tickets/%s/comments", currentTicket.Identifier),
		"",
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
	)
	if invalidCommentIdentifierRec.Code != http.StatusBadRequest || !strings.Contains(invalidCommentIdentifierRec.Body.String(), "INVALID_TICKET_ID") {
		t.Fatalf("expected comment identifier lookup to return INVALID_TICKET_ID, got %d: %s", invalidCommentIdentifierRec.Code, invalidCommentIdentifierRec.Body.String())
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
			"body": "## Workpad\n\nValidation\n- npm test",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&updateCommentResp,
	)
	if updateCommentResp.Comment.Body != "## Workpad\n\nValidation\n- npm test" || updateCommentResp.Comment.LastEditedBy == nil || *updateCommentResp.Comment.LastEditedBy != "agent:coding-01" {
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
	projectScopedForbiddenRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/projects/%s/tickets/%s", projectID, currentTicketID),
		`{"description":"should fail"}`,
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token, echo.HeaderContentType: echo.MIMEApplicationJSON},
	)
	if projectScopedForbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("expected project-scoped ticket update without tickets.update to return 403, got %d: %s", projectScopedForbiddenRec.Code, projectScopedForbiddenRec.Body.String())
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

func TestAgentPlatformTicketMutationsPublishRefreshEvents(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)
	entStatuses, err := client.TicketStatus.Query().All(ctx)
	if err != nil {
		t.Fatalf("list ticket statuses: %v", err)
	}
	var todoID uuid.UUID
	var inProgressID uuid.UUID
	for _, status := range entStatuses {
		switch status.Name {
		case "Todo":
			todoID = status.ID
		case "In Progress":
			inProgressID = status.ID
		}
	}
	if todoID == uuid.Nil || inProgressID == uuid.Nil {
		t.Fatalf("expected Todo and In Progress statuses, got %+v", entStatuses)
	}
	parentTicket, err := client.Ticket.Create().
		SetProjectID(projectID).
		SetIdentifier("ASE-99").
		SetTitle("Parent via agent create").
		SetStatusID(todoID).
		SetCreatedBy("user:codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create parent ticket: %v", err)
	}

	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))
	bus := eventinfra.NewChannelBus()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		newTicketService(client),
		newTicketStatusService(client),
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
	headers := map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token}
	ticketStream := subscribeTopicEvents(t, bus, ticketEventsTopic)

	createResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/projects/%s/tickets", projectID),
		map[string]any{
			"title":            "Agent child ticket",
			"parent_ticket_id": parentTicket.ID.String(),
		},
		headers,
		http.StatusCreated,
		&createResp,
	)
	assertStringSet(
		t,
		readTicketEventTicketIDs(t, ticketStream, 2),
		createResp.Ticket.ID,
		parentTicket.ID.String(),
	)

	activityStream := subscribeTopicEvents(t, bus, activityStreamTopic)

	createCommentResp := struct {
		Comment ticketCommentResponse `json:"comment"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/tickets/%s/comments", currentTicketID),
		map[string]any{"body": "agent comment"},
		headers,
		http.StatusCreated,
		&createCommentResp,
	)
	commentCreateActivity := readEvents(t, activityStream, 2)
	if commentCreateActivity[0].Type != provider.MustParseEventType(activityevent.TypeTicketCommentCreated.String()) ||
		commentCreateActivity[1].Type != provider.MustParseEventType(activityevent.TypeTicketUpdated.String()) {
		t.Fatalf("unexpected agent comment create activity types: %+v", commentCreateActivity)
	}
	assertStringSet(t, readTicketEventTicketIDs(t, ticketStream, 1), currentTicketID.String())

	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/tickets/%s/comments/%s", currentTicketID, createCommentResp.Comment.ID),
		map[string]any{"body": "agent comment updated"},
		headers,
		http.StatusOK,
		nil,
	)
	commentUpdateActivity := readEvents(t, activityStream, 2)
	if commentUpdateActivity[0].Type != provider.MustParseEventType(activityevent.TypeTicketCommentEdited.String()) ||
		commentUpdateActivity[1].Type != provider.MustParseEventType(activityevent.TypeTicketUpdated.String()) {
		t.Fatalf("unexpected agent comment update activity types: %+v", commentUpdateActivity)
	}
	assertStringSet(t, readTicketEventTicketIDs(t, ticketStream, 1), currentTicketID.String())

	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
		map[string]any{"status_id": inProgressID.String()},
		headers,
		http.StatusOK,
		nil,
	)
	statusEvent := readEvents(t, ticketStream, 1)[0]
	if statusEvent.Type != provider.MustParseEventType(activityevent.TypeTicketStatusChanged.String()) {
		t.Fatalf("unexpected agent status change event type: %+v", statusEvent)
	}
}

func TestAgentPlatformPrivilegedRoutesRequireExplicitScopes(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
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
	activityItems, err := client.ActivityEvent.Query().All(ctx)
	if err != nil {
		t.Fatalf("query activity after agent project patch: %v", err)
	}
	if len(activityItems) != 1 || activityItems[0].EventType != activityevent.TypeProjectUpdated.String() {
		t.Fatalf("expected agent project patch to emit project.updated activity, got %+v", activityItems)
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

func TestAgentPlatformProjectConversationTokenRejectsTicketOnlyRoutes(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))

	conversationID := uuid.New()
	if _, err := client.ChatConversation.Create().
		SetID(conversationID).
		SetProjectID(projectID).
		SetUserID("browser-user").
		SetSource("project_sidebar").
		SetProviderID(uuid.New()).
		SetStatus("active").
		Save(ctx); err != nil {
		t.Fatalf("create chat conversation: %v", err)
	}
	if _, err := client.ProjectConversationPrincipal.Create().
		SetID(conversationID).
		SetConversationID(conversationID).
		SetProjectID(projectID).
		SetProviderID(uuid.New()).
		SetName("project-conversation:" + conversationID.String()).
		Save(ctx); err != nil {
		t.Fatalf("create project conversation principal: %v", err)
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	issued, err := platformService.IssueToken(ctx, agentplatform.IssueInput{
		PrincipalKind:  agentplatform.PrincipalKindProjectConversation,
		PrincipalID:    conversationID,
		ProjectID:      projectID,
		ConversationID: conversationID,
	})
	if err != nil {
		t.Fatalf("IssueToken(project conversation) returned error: %v", err)
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
			"title":       "Conversation-created follow-up",
			"description": "Created from project conversation token",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusCreated,
		&createResp,
	)
	if createResp.Ticket.CreatedBy != "project-conversation:"+conversationID.String() {
		t.Fatalf("unexpected created_by for project conversation token: %+v", createResp.Ticket)
	}

	updateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/projects/%s/tickets/%s", projectID, currentTicketID),
		map[string]any{
			"status_name": "In Progress",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
		http.StatusOK,
		&updateResp,
	)
	if updateResp.Ticket.StatusName != "In Progress" || updateResp.Ticket.CreatedBy != "project-conversation:"+conversationID.String() {
		t.Fatalf("unexpected project-scoped update payload: %+v", updateResp.Ticket)
	}

	forbiddenRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID),
		"",
		map[string]string{echo.HeaderAuthorization: "Bearer " + issued.Token},
	)
	if forbiddenRec.Code != http.StatusForbidden || !strings.Contains(forbiddenRec.Body.String(), "AGENT_PRINCIPAL_KIND_FORBIDDEN") {
		t.Fatalf("expected ticket-only route to reject project conversation principal, got %d: %s", forbiddenRec.Code, forbiddenRec.Body.String())
	}

	_ = agentID
}

func TestAgentPlatformProjectUpdateRoutesRespectScopesAndBoundaries(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))
	bus := eventinfra.NewChannelBus()
	projectUpdateSvc := projectupdateservice.NewService(
		client,
		activitysvc.NewEmitter(activitysvc.EntRecorder{Client: client}, bus),
	)

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		newTicketService(client),
		newTicketStatusService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
		WithProjectUpdateService(projectUpdateSvc),
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
			string(agentplatform.ScopeProjectsUpdate),
			string(agentplatform.ScopeTicketsList),
			string(agentplatform.ScopeTicketsUpdateSelf),
		},
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	createThreadResp := struct {
		Thread projectUpdateThreadResponse `json:"thread"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/projects/%s/updates", projectID),
		map[string]any{
			"status": "on_track",
			"body":   "Agent is monitoring cross-ticket GPU utilization.",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + privilegedToken.Token},
		http.StatusCreated,
		&createThreadResp,
	)
	if createThreadResp.Thread.CreatedBy != "agent:coding-01" || createThreadResp.Thread.Status != "on_track" {
		t.Fatalf("unexpected created thread payload: %+v", createThreadResp.Thread)
	}
	if createThreadResp.Thread.Title != "Agent is monitoring cross-ticket GPU utilization." {
		t.Fatalf("unexpected derived thread title payload: %+v", createThreadResp.Thread)
	}

	listResp := struct {
		Threads []projectUpdateThreadResponse `json:"threads"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/projects/%s/updates", projectID),
		nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + defaultToken.Token},
		http.StatusOK,
		&listResp,
	)
	if len(listResp.Threads) != 1 || listResp.Threads[0].ID != createThreadResp.Thread.ID {
		t.Fatalf("unexpected list threads payload: %+v", listResp.Threads)
	}

	updateThreadResp := struct {
		Thread projectUpdateThreadResponse `json:"thread"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/platform/projects/%s/updates/%s", projectID, createThreadResp.Thread.ID),
		map[string]any{
			"status":      "at_risk",
			"body":        "One queue is backing up on the A100 pool.",
			"edit_reason": "refined after another measurement window",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + privilegedToken.Token},
		http.StatusOK,
		&updateThreadResp,
	)
	if updateThreadResp.Thread.LastEditedBy == nil || *updateThreadResp.Thread.LastEditedBy != "agent:coding-01" || updateThreadResp.Thread.Status != "at_risk" {
		t.Fatalf("unexpected updated thread payload: %+v", updateThreadResp.Thread)
	}
	if updateThreadResp.Thread.Title != "One queue is backing up on the A100 pool." {
		t.Fatalf("unexpected updated derived thread title payload: %+v", updateThreadResp.Thread)
	}

	createCommentResp := struct {
		Comment projectUpdateCommentResponse `json:"comment"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/projects/%s/updates/%s/comments", projectID, createThreadResp.Thread.ID),
		map[string]any{
			"body": "Watching another 30 minutes before escalating.",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + privilegedToken.Token},
		http.StatusCreated,
		&createCommentResp,
	)
	if createCommentResp.Comment.CreatedBy != "agent:coding-01" {
		t.Fatalf("unexpected created comment payload: %+v", createCommentResp.Comment)
	}

	updateCommentResp := struct {
		Comment projectUpdateCommentResponse `json:"comment"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf(
			"/api/v1/platform/projects/%s/updates/%s/comments/%s",
			projectID,
			createThreadResp.Thread.ID,
			createCommentResp.Comment.ID,
		),
		map[string]any{
			"body":        "Watching another 30 minutes before escalating to the infra owner.",
			"edit_reason": "added escalation target",
		},
		map[string]string{echo.HeaderAuthorization: "Bearer " + privilegedToken.Token},
		http.StatusOK,
		&updateCommentResp,
	)
	if updateCommentResp.Comment.LastEditedBy == nil || *updateCommentResp.Comment.LastEditedBy != "agent:coding-01" {
		t.Fatalf("unexpected updated comment payload: %+v", updateCommentResp.Comment)
	}

	deleteCommentResp := struct {
		DeletedCommentID string `json:"deleted_comment_id"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf(
			"/api/v1/platform/projects/%s/updates/%s/comments/%s",
			projectID,
			createThreadResp.Thread.ID,
			createCommentResp.Comment.ID,
		),
		nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + privilegedToken.Token},
		http.StatusOK,
		&deleteCommentResp,
	)
	if deleteCommentResp.DeletedCommentID != createCommentResp.Comment.ID {
		t.Fatalf("unexpected delete comment payload: %+v", deleteCommentResp)
	}

	deleteThreadResp := struct {
		DeletedThreadID string `json:"deleted_thread_id"`
	}{}
	executeJSONWithHeaders(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/platform/projects/%s/updates/%s", projectID, createThreadResp.Thread.ID),
		nil,
		map[string]string{echo.HeaderAuthorization: "Bearer " + privilegedToken.Token},
		http.StatusOK,
		&deleteThreadResp,
	)
	if deleteThreadResp.DeletedThreadID != createThreadResp.Thread.ID {
		t.Fatalf("unexpected delete thread payload: %+v", deleteThreadResp)
	}

	forbiddenCreateRec := performJSONRequestWithHeaders(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/platform/projects/%s/updates", projectID),
		`{"status":"on_track","body":"no scope"}`,
		map[string]string{echo.HeaderAuthorization: "Bearer " + defaultToken.Token, echo.HeaderContentType: echo.MIMEApplicationJSON},
	)
	if forbiddenCreateRec.Code != http.StatusForbidden {
		t.Fatalf("expected update create without scope to return 403, got %d: %s", forbiddenCreateRec.Code, forbiddenCreateRec.Body.String())
	}
}

func TestAgentPlatformHarnessWhitelistConstrainsTokenScopes(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, currentTicketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)
	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	access, err := workflowservice.ParsePlatformAccess("tickets.list")
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
	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
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
	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		platformService,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, projectID)
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
	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
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
			string(agentplatform.ScopeTicketsUpdate),
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
		{name: "project update invalid project", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/platform/projects/%s/tickets/%s", "not-a-uuid", currentTicketID), body: `{"description":"nope"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "project update invalid ticket", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/platform/projects/%s/tickets/%s", projectID, "not-a-uuid"), body: `{"description":"nope"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_TICKET_ID"},
		{name: "update invalid ticket", method: http.MethodPatch, target: "/api/v1/platform/tickets/not-a-uuid", body: `{"description":"nope"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_TICKET_ID"},
		{name: "update invalid status", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/platform/tickets/%s", currentTicketID), body: `{"status_id":"not-a-uuid"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "project update invalid status", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/platform/projects/%s/tickets/%s", projectID, currentTicketID), body: `{"status_id":"not-a-uuid"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "report usage invalid ticket", method: http.MethodPost, target: "/api/v1/platform/tickets/not-a-uuid/usage", body: `{"input_tokens":1}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_TICKET_ID"},
		{name: "report usage invalid request", method: http.MethodPost, target: fmt.Sprintf("/api/v1/platform/tickets/%s/usage", currentTicketID), body: `{}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "update project invalid project", method: http.MethodPatch, target: "/api/v1/platform/projects/not-a-uuid", body: `{"description":"x"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "update project missing fields", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/platform/projects/%s", projectID), body: `{}`, wantStatus: http.StatusBadRequest, wantBody: "at least one project field must be provided"},
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

	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
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

type agentPlatformExpandedFixture struct {
	client               *ent.Client
	server               *Server
	platformService      *agentplatform.Service
	projectID            uuid.UUID
	agentID              uuid.UUID
	ticketID             uuid.UUID
	providerID           uuid.UUID
	mainWorkflowID       uuid.UUID
	deleteWorkflowID     uuid.UUID
	repoReadID           uuid.UUID
	repoScopeCreateID    uuid.UUID
	repoDeleteID         uuid.UUID
	ticketRepoScopeID    uuid.UUID
	ticketRepoDeleteID   uuid.UUID
	scheduledJobID       uuid.UUID
	scheduledJobDeleteID uuid.UUID
	skillMainID          uuid.UUID
	skillDeleteID        uuid.UUID
	statusUpdateID       uuid.UUID
	statusDeleteID       uuid.UUID
}

func TestAgentPlatformExpandedProjectRoutesRequireExplicitScopes(t *testing.T) {
	fixture := newAgentPlatformExpandedFixture(t)

	for _, tc := range []struct {
		name       string
		scope      agentplatform.Scope
		method     string
		path       string
		body       any
		wantStatus int
		wantBody   string
	}{
		{name: "activity.read", scope: agentplatform.ScopeActivityRead, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/projects/%s/activity", fixture.projectID), wantStatus: http.StatusOK, wantBody: `"events":[`},
		{name: "statuses.list", scope: agentplatform.ScopeStatusesList, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/projects/%s/statuses", fixture.projectID), wantStatus: http.StatusOK, wantBody: `"statuses":[`},
		{name: "statuses.create", scope: agentplatform.ScopeStatusesCreate, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/projects/%s/statuses", fixture.projectID), body: map[string]any{"name": "Platform QA", "stage": "started", "color": "#22AA66"}, wantStatus: http.StatusCreated, wantBody: `"name":"Platform QA"`},
		{name: "statuses.update", scope: agentplatform.ScopeStatusesUpdate, method: http.MethodPatch, path: fmt.Sprintf("/api/v1/platform/statuses/%s", fixture.statusUpdateID), body: map[string]any{"name": "Platform Updated", "color": "#3366FF"}, wantStatus: http.StatusOK, wantBody: `"name":"Platform Updated"`},
		{name: "workflows.list", scope: agentplatform.ScopeWorkflowsList, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/projects/%s/workflows", fixture.projectID), wantStatus: http.StatusOK, wantBody: `"workflows":[`},
		{name: "workflows.read", scope: agentplatform.ScopeWorkflowsRead, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/workflows/%s", fixture.mainWorkflowID), wantStatus: http.StatusOK, wantBody: fixture.mainWorkflowID.String()},
		{name: "workflows.create", scope: agentplatform.ScopeWorkflowsCreate, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/projects/%s/workflows", fixture.projectID), body: map[string]any{"agent_id": fixture.agentID.String(), "name": "Platform Workflow Create", "type": "coding", "pickup_status_ids": []string{fixture.statusUpdateID.String()}, "finish_status_ids": []string{fixture.statusDeleteID.String()}, "harness_content": "# Platform Create\n", "is_active": false}, wantStatus: http.StatusCreated, wantBody: `"name":"Platform Workflow Create"`},
		{name: "workflows.update", scope: agentplatform.ScopeWorkflowsUpdate, method: http.MethodPatch, path: fmt.Sprintf("/api/v1/platform/workflows/%s", fixture.mainWorkflowID), body: map[string]any{"name": "Platform Workflow Updated"}, wantStatus: http.StatusOK, wantBody: `"name":"Platform Workflow Updated"`},
		{name: "workflows.delete", scope: agentplatform.ScopeWorkflowsDelete, method: http.MethodDelete, path: fmt.Sprintf("/api/v1/platform/workflows/%s", fixture.deleteWorkflowID), wantStatus: http.StatusOK, wantBody: fixture.deleteWorkflowID.String()},
		{name: "workflows.harness.read", scope: agentplatform.ScopeWorkflowsHarnessRead, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/workflows/%s/harness", fixture.mainWorkflowID), wantStatus: http.StatusOK, wantBody: `"workflow_id":"` + fixture.mainWorkflowID.String() + `"`},
		{name: "workflows.harness.history.read", scope: agentplatform.ScopeWorkflowsHarnessHistoryRead, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/workflows/%s/harness/history", fixture.mainWorkflowID), wantStatus: http.StatusOK, wantBody: `"history":[`},
		{name: "workflows.harness.update", scope: agentplatform.ScopeWorkflowsHarnessUpdate, method: http.MethodPut, path: fmt.Sprintf("/api/v1/platform/workflows/%s/harness", fixture.mainWorkflowID), body: map[string]any{"content": "# Platform Harness Update\n"}, wantStatus: http.StatusOK, wantBody: `"version":`},
		{name: "workflows.harness.validate", scope: agentplatform.ScopeWorkflowsHarnessValidate, method: http.MethodPost, path: "/api/v1/platform/harness/validate", body: map[string]any{"content": "# Validate\n"}, wantStatus: http.StatusOK, wantBody: `"valid":true`},
		{name: "workflows.harness.variables.read", scope: agentplatform.ScopeWorkflowsHarnessVariablesRead, method: http.MethodGet, path: "/api/v1/platform/harness/variables", wantStatus: http.StatusOK, wantBody: `"groups":[`},
		{name: "statuses.delete", scope: agentplatform.ScopeStatusesDelete, method: http.MethodDelete, path: fmt.Sprintf("/api/v1/platform/statuses/%s", fixture.statusDeleteID), wantStatus: http.StatusOK, wantBody: fixture.statusDeleteID.String()},
		{name: "statuses.reset", scope: agentplatform.ScopeStatusesReset, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/projects/%s/statuses/reset", fixture.projectID), wantStatus: http.StatusOK, wantBody: `"statuses":[`},
		{name: "repos.read", scope: agentplatform.ScopeReposRead, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/projects/%s/repos", fixture.projectID), wantStatus: http.StatusOK, wantBody: `"repos":[`},
		{name: "repos.update", scope: agentplatform.ScopeReposUpdate, method: http.MethodPatch, path: fmt.Sprintf("/api/v1/platform/projects/%s/repos/%s", fixture.projectID, fixture.repoReadID), body: map[string]any{"name": "platform-primary-updated", "repository_url": "https://github.com/acme/platform-primary-updated.git", "default_branch": "main"}, wantStatus: http.StatusOK, wantBody: `"name":"platform-primary-updated"`},
		{name: "repos.delete", scope: agentplatform.ScopeReposDelete, method: http.MethodDelete, path: fmt.Sprintf("/api/v1/platform/projects/%s/repos/%s", fixture.projectID, fixture.repoDeleteID), wantStatus: http.StatusOK, wantBody: fixture.repoDeleteID.String()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPlatformScopeRoute(t, fixture, tc.scope, tc.method, tc.path, tc.body, tc.wantStatus, tc.wantBody)
		})
	}
}

func TestAgentPlatformExpandedTicketRepoScopeRoutesRequireExplicitScopes(t *testing.T) {
	fixture := newAgentPlatformExpandedFixture(t)

	for _, tc := range []struct {
		name       string
		scope      agentplatform.Scope
		method     string
		path       string
		body       any
		wantStatus int
		wantBody   string
	}{
		{name: "ticket_repo_scopes.list", scope: agentplatform.ScopeTicketRepoScopesList, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/projects/%s/tickets/%s/repo-scopes", fixture.projectID, fixture.ticketID), wantStatus: http.StatusOK, wantBody: `"repo_scopes":[`},
		{name: "ticket_repo_scopes.create", scope: agentplatform.ScopeTicketRepoScopesCreate, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/projects/%s/tickets/%s/repo-scopes", fixture.projectID, fixture.ticketID), body: map[string]any{"repo_id": fixture.repoScopeCreateID.String(), "branch_name": "feature/platform-create"}, wantStatus: http.StatusCreated, wantBody: `"branch_name":"feature/platform-create"`},
		{name: "ticket_repo_scopes.update", scope: agentplatform.ScopeTicketRepoScopesUpdate, method: http.MethodPatch, path: fmt.Sprintf("/api/v1/platform/projects/%s/tickets/%s/repo-scopes/%s", fixture.projectID, fixture.ticketID, fixture.ticketRepoScopeID), body: map[string]any{"branch_name": "feature/platform-update"}, wantStatus: http.StatusOK, wantBody: `"branch_name":"feature/platform-update"`},
		{name: "ticket_repo_scopes.delete", scope: agentplatform.ScopeTicketRepoScopesDelete, method: http.MethodDelete, path: fmt.Sprintf("/api/v1/platform/projects/%s/tickets/%s/repo-scopes/%s", fixture.projectID, fixture.ticketID, fixture.ticketRepoDeleteID), wantStatus: http.StatusOK, wantBody: fixture.ticketRepoDeleteID.String()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPlatformScopeRoute(t, fixture, tc.scope, tc.method, tc.path, tc.body, tc.wantStatus, tc.wantBody)
		})
	}
}

func TestAgentPlatformProjectConversationTokenCanListTicketRepoScopes(t *testing.T) {
	fixture := newAgentPlatformExpandedFixture(t)
	ctx := context.Background()
	conversationID := uuid.New()

	if _, err := fixture.client.ChatConversation.Create().
		SetID(conversationID).
		SetProjectID(fixture.projectID).
		SetUserID("browser-user").
		SetSource("project_sidebar").
		SetProviderID(fixture.providerID).
		SetStatus("active").
		Save(ctx); err != nil {
		t.Fatalf("create chat conversation: %v", err)
	}
	if _, err := fixture.client.ProjectConversationPrincipal.Create().
		SetID(conversationID).
		SetConversationID(conversationID).
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("project-conversation:" + conversationID.String()).
		Save(ctx); err != nil {
		t.Fatalf("create project conversation principal: %v", err)
	}

	issued, err := fixture.platformService.IssueToken(ctx, agentplatform.IssueInput{
		PrincipalKind:  agentplatform.PrincipalKindProjectConversation,
		PrincipalID:    conversationID,
		ProjectID:      fixture.projectID,
		ConversationID: conversationID,
		Scopes:         []string{string(agentplatform.ScopeTicketRepoScopesList)},
	})
	if err != nil {
		t.Fatalf("IssueToken(project conversation) returned error: %v", err)
	}

	rec := performPlatformRequest(
		t,
		fixture.server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/platform/projects/%s/tickets/%s/repo-scopes", fixture.projectID, fixture.ticketID),
		nil,
		issued.Token,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected project conversation repo scope list to return 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"repo_scopes":[`) {
		t.Fatalf("expected repo scope list response, got %s", rec.Body.String())
	}
}

func TestAgentPlatformExpandedScheduledJobRoutesRequireExplicitScopes(t *testing.T) {
	fixture := newAgentPlatformExpandedFixture(t)
	if _, err := fixture.server.catalog.DeleteTicketRepoScope(context.Background(), fixture.projectID, fixture.ticketID, fixture.ticketRepoScopeID); err != nil {
		t.Fatalf("delete ticket repo scope %s: %v", fixture.ticketRepoScopeID, err)
	}
	if _, err := fixture.server.catalog.DeleteTicketRepoScope(context.Background(), fixture.projectID, fixture.ticketID, fixture.ticketRepoDeleteID); err != nil {
		t.Fatalf("delete ticket repo delete scope %s: %v", fixture.ticketRepoDeleteID, err)
	}
	repos, err := fixture.server.catalog.ListProjectRepos(context.Background(), fixture.projectID)
	if err != nil {
		t.Fatalf("list project repos: %v", err)
	}
	for index, repo := range repos {
		if index == 0 {
			continue
		}
		if _, err := fixture.server.catalog.DeleteProjectRepo(context.Background(), fixture.projectID, repo.ID); err != nil {
			t.Fatalf("delete extra repo %s: %v", repo.ID, err)
		}
	}

	for _, tc := range []struct {
		name       string
		scope      agentplatform.Scope
		method     string
		path       string
		body       any
		wantStatus int
		wantBody   string
	}{
		{name: "scheduled_jobs.list", scope: agentplatform.ScopeScheduledJobsList, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/projects/%s/scheduled-jobs", fixture.projectID), wantStatus: http.StatusOK, wantBody: `"scheduled_jobs":[`},
		{name: "scheduled_jobs.create", scope: agentplatform.ScopeScheduledJobsCreate, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/projects/%s/scheduled-jobs", fixture.projectID), body: map[string]any{"name": "platform-create-job", "cron_expression": "0 7 * * 2", "ticket_template": map[string]any{"title": "Platform create job", "description": "create via platform scope", "status": "Backlog", "priority": "medium", "type": "feature"}}, wantStatus: http.StatusCreated, wantBody: `"name":"platform-create-job"`},
		{name: "scheduled_jobs.update", scope: agentplatform.ScopeScheduledJobsUpdate, method: http.MethodPatch, path: fmt.Sprintf("/api/v1/platform/scheduled-jobs/%s", fixture.scheduledJobID), body: map[string]any{"name": "platform-main-job-updated"}, wantStatus: http.StatusOK, wantBody: `"name":"platform-main-job-updated"`},
		{name: "scheduled_jobs.trigger", scope: agentplatform.ScopeScheduledJobsTrigger, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/scheduled-jobs/%s/trigger", fixture.scheduledJobID), wantStatus: http.StatusOK, wantBody: `"ticket":{`},
		{name: "scheduled_jobs.delete", scope: agentplatform.ScopeScheduledJobsDelete, method: http.MethodDelete, path: fmt.Sprintf("/api/v1/platform/scheduled-jobs/%s", fixture.scheduledJobDeleteID), wantStatus: http.StatusOK, wantBody: fixture.scheduledJobDeleteID.String()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPlatformScopeRoute(t, fixture, tc.scope, tc.method, tc.path, tc.body, tc.wantStatus, tc.wantBody)
		})
	}
}

func TestAgentPlatformExpandedSkillRoutesRequireExplicitScopes(t *testing.T) {
	fixture := newAgentPlatformExpandedFixture(t)
	importedEntry := base64.StdEncoding.EncodeToString([]byte("---\nname: platform-imported\ndescription: platform imported bundle\n---\n\n# Imported\n"))

	for _, tc := range []struct {
		name       string
		scope      agentplatform.Scope
		method     string
		path       string
		body       any
		wantStatus int
		wantBody   string
	}{
		{name: "skills.list", scope: agentplatform.ScopeSkillsList, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/projects/%s/skills", fixture.projectID), wantStatus: http.StatusOK, wantBody: `"skills":[`},
		{name: "skills.read", scope: agentplatform.ScopeSkillsRead, method: http.MethodGet, path: fmt.Sprintf("/api/v1/platform/skills/%s", fixture.skillMainID), wantStatus: http.StatusOK, wantBody: fixture.skillMainID.String()},
		{name: "skills.create", scope: agentplatform.ScopeSkillsCreate, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/projects/%s/skills", fixture.projectID), body: map[string]any{"name": "platform-created", "content": "# Platform Created\n", "description": "created via platform"}, wantStatus: http.StatusCreated, wantBody: `"name":"platform-created"`},
		{name: "skills.import", scope: agentplatform.ScopeSkillsImport, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/projects/%s/skills/import", fixture.projectID), body: map[string]any{"name": "platform-imported", "files": []map[string]any{{"path": "SKILL.md", "content_base64": importedEntry, "media_type": "text/markdown; charset=utf-8", "is_executable": false}}}, wantStatus: http.StatusCreated, wantBody: `"name":"platform-imported"`},
		{name: "skills.refresh", scope: agentplatform.ScopeSkillsRefresh, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/projects/%s/skills/refresh", fixture.projectID), body: map[string]any{"workspace_root": t.TempDir(), "adapter_type": "claude-code-cli"}, wantStatus: http.StatusOK, wantBody: `"injected_skills":[`},
		{name: "skills.update", scope: agentplatform.ScopeSkillsUpdate, method: http.MethodPut, path: fmt.Sprintf("/api/v1/platform/skills/%s", fixture.skillMainID), body: map[string]any{"content": "# Platform Updated Skill\n", "description": "updated via platform"}, wantStatus: http.StatusOK, wantBody: `updated via platform`},
		{name: "skills.disable", scope: agentplatform.ScopeSkillsDisable, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/skills/%s/disable", fixture.skillMainID), wantStatus: http.StatusOK, wantBody: `"is_enabled":false`},
		{name: "skills.enable", scope: agentplatform.ScopeSkillsEnable, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/skills/%s/enable", fixture.skillMainID), wantStatus: http.StatusOK, wantBody: `"is_enabled":true`},
		{name: "skills.bind", scope: agentplatform.ScopeSkillsBind, method: http.MethodPost, path: fmt.Sprintf("/api/v1/platform/skills/%s/bind", fixture.skillMainID), body: map[string]any{"workflow_ids": []string{fixture.mainWorkflowID.String()}}, wantStatus: http.StatusOK, wantBody: `"bound_workflows":[`},
		{name: "skills.delete", scope: agentplatform.ScopeSkillsDelete, method: http.MethodDelete, path: fmt.Sprintf("/api/v1/platform/skills/%s", fixture.skillDeleteID), wantStatus: http.StatusOK, wantBody: fixture.skillDeleteID.String()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPlatformScopeRoute(t, fixture, tc.scope, tc.method, tc.path, tc.body, tc.wantStatus, tc.wantBody)
		})
	}
}

func newAgentPlatformExpandedFixture(t *testing.T) *agentPlatformExpandedFixture {
	t.Helper()

	client := openTestEntClient(t)
	ctx := context.Background()
	projectID, agentID, ticketID, _ := seedAgentPlatformHTTPFixture(ctx, t, client)

	agentItem, err := client.Agent.Get(ctx, agentID)
	if err != nil {
		t.Fatalf("load agent: %v", err)
	}
	providerItem, err := client.AgentProvider.Get(ctx, agentItem.ProviderID)
	if err != nil {
		t.Fatalf("load provider: %v", err)
	}
	repoRoot := createTestGitRepo(t)
	createPrimaryProjectRepo(ctx, t, client, projectID, repoRoot)
	attachPrimaryProjectRepoCheckout(ctx, t, client, projectID, providerItem.MachineID, repoRoot)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	statusList, err := newTicketStatusService(client).List(ctx, projectID)
	if err != nil {
		t.Fatalf("list statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statusList.Statuses, "Todo")
	doneID := findStatusIDByName(t, statusList.Statuses, "Done")
	inProgressID := findStatusIDByName(t, statusList.Statuses, "In Progress")

	mainWorkflow, err := workflowSvc.Create(ctx, workflowservice.CreateInput{
		ProjectID:           projectID,
		AgentID:             agentID,
		Name:                "Platform Main Workflow",
		Type:                "coding",
		HarnessContent:      "# Main Workflow\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      60,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     workflowservice.MustStatusBindingSet(todoID),
		FinishStatusIDs:     workflowservice.MustStatusBindingSet(doneID),
	})
	if err != nil {
		t.Fatalf("create main workflow: %v", err)
	}
	deleteWorkflow, err := workflowSvc.Create(ctx, workflowservice.CreateInput{
		ProjectID:           projectID,
		AgentID:             agentID,
		Name:                "Platform Delete Workflow",
		Type:                "coding",
		HarnessContent:      "# Delete Workflow\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      60,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     workflowservice.MustStatusBindingSet(inProgressID),
		FinishStatusIDs:     workflowservice.MustStatusBindingSet(doneID),
	})
	if err != nil {
		t.Fatalf("create delete workflow: %v", err)
	}

	catalogSvc := catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil)
	projectRepos, err := catalogSvc.ListProjectRepos(ctx, projectID)
	if err != nil || len(projectRepos) == 0 {
		t.Fatalf("list initial repos: %v repos=%d", err, len(projectRepos))
	}
	repoReadID := projectRepos[0].ID
	repoScopeCreate, err := catalogSvc.CreateProjectRepo(ctx, catalogdomain.CreateProjectRepo{ProjectID: projectID, Name: "platform-scope-create", RepositoryURL: "https://github.com/acme/platform-scope-create.git", DefaultBranch: "main", WorkspaceDirname: "platform-scope-create"})
	if err != nil {
		t.Fatalf("create repo scope create repo: %v", err)
	}
	repoScopeUpdate, err := catalogSvc.CreateProjectRepo(ctx, catalogdomain.CreateProjectRepo{ProjectID: projectID, Name: "platform-scope-update", RepositoryURL: "https://github.com/acme/platform-scope-update.git", DefaultBranch: "main", WorkspaceDirname: "platform-scope-update"})
	if err != nil {
		t.Fatalf("create repo scope update repo: %v", err)
	}
	repoScopeDelete, err := catalogSvc.CreateProjectRepo(ctx, catalogdomain.CreateProjectRepo{ProjectID: projectID, Name: "platform-scope-delete", RepositoryURL: "https://github.com/acme/platform-scope-delete.git", DefaultBranch: "main", WorkspaceDirname: "platform-scope-delete"})
	if err != nil {
		t.Fatalf("create repo scope delete repo: %v", err)
	}
	repoDelete, err := catalogSvc.CreateProjectRepo(ctx, catalogdomain.CreateProjectRepo{ProjectID: projectID, Name: "platform-delete-repo", RepositoryURL: "https://github.com/acme/platform-delete-repo.git", DefaultBranch: "main", WorkspaceDirname: "platform-delete-repo"})
	if err != nil {
		t.Fatalf("create repo delete repo: %v", err)
	}

	ticketRepoScope, err := catalogSvc.CreateTicketRepoScope(ctx, catalogdomain.CreateTicketRepoScope{ProjectID: projectID, TicketID: ticketID, RepoID: repoScopeUpdate.ID, BranchName: stringPtr("feature/existing-scope")})
	if err != nil {
		t.Fatalf("create ticket repo scope: %v", err)
	}
	ticketRepoDelete, err := catalogSvc.CreateTicketRepoScope(ctx, catalogdomain.CreateTicketRepoScope{ProjectID: projectID, TicketID: ticketID, RepoID: repoScopeDelete.ID, BranchName: stringPtr("feature/delete-scope")})
	if err != nil {
		t.Fatalf("create ticket repo delete scope: %v", err)
	}

	statusUpdate, err := client.TicketStatus.Create().SetProjectID(projectID).SetName("Platform Scope Update").SetStage("started").SetColor("#1144AA").SetPosition(20).Save(ctx)
	if err != nil {
		t.Fatalf("create status update seed: %v", err)
	}
	statusDelete, err := client.TicketStatus.Create().SetProjectID(projectID).SetName("Platform Scope Delete").SetStage("started").SetColor("#AA4411").SetPosition(21).Save(ctx)
	if err != nil {
		t.Fatalf("create status delete seed: %v", err)
	}

	skillMain, err := workflowSvc.CreateSkill(ctx, workflowservice.CreateSkillInput{ProjectID: projectID, Name: "platform-main-skill", Content: "# Main Skill\n", Description: "main skill", CreatedBy: "user:platform"})
	if err != nil {
		t.Fatalf("create main skill: %v", err)
	}
	skillDelete, err := workflowSvc.CreateSkill(ctx, workflowservice.CreateSkillInput{ProjectID: projectID, Name: "platform-delete-skill", Content: "# Delete Skill\n", Description: "delete skill", CreatedBy: "user:platform"})
	if err != nil {
		t.Fatalf("create delete skill: %v", err)
	}

	ticketSvc := newTicketService(client)
	if _, err := ticketSvc.RecordActivityEvent(ctx, ticketservice.RecordActivityEventInput{ProjectID: projectID, TicketID: &ticketID, AgentID: &agentID, EventType: "agent.ready", Message: "platform scope fixture ready"}); err != nil {
		t.Fatalf("record activity event: %v", err)
	}

	scheduledJobSvc := scheduledjobservice.NewService(scheduledjobrepo.NewEntRepository(client), ticketSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	scheduledJobSvc.SetNowFunc(func() time.Time {
		return time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)
	})
	scheduledJobMain, err := scheduledJobSvc.Create(ctx, scheduledjobservice.CreateInput{ProjectID: projectID, Name: "platform-main-job", CronExpression: "0 9 * * 1", TicketTemplate: scheduledjobservice.TicketTemplate{Title: "Platform main job", Description: "main scheduled job", Status: "Backlog", Priority: ticketservice.PriorityMedium, Type: ticketservice.TypeFeature, CreatedBy: "system:platform"}, IsEnabled: true})
	if err != nil {
		t.Fatalf("create main scheduled job: %v", err)
	}
	scheduledJobDelete, err := scheduledJobSvc.Create(ctx, scheduledjobservice.CreateInput{ProjectID: projectID, Name: "platform-delete-job", CronExpression: "0 10 * * 2", TicketTemplate: scheduledjobservice.TicketTemplate{Title: "Platform delete job", Description: "delete scheduled job", Status: "Backlog", Priority: ticketservice.PriorityMedium, Type: ticketservice.TypeFeature, CreatedBy: "system:platform"}, IsEnabled: true})
	if err != nil {
		t.Fatalf("create delete scheduled job: %v", err)
	}

	platformService := agentplatform.NewService(agentplatformrepo.NewEntRepository(client))
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketSvc,
		newTicketStatusService(client),
		platformService,
		catalogSvc,
		workflowSvc,
		WithScheduledJobService(scheduledJobSvc),
	)

	return &agentPlatformExpandedFixture{
		client:               client,
		server:               server,
		platformService:      platformService,
		projectID:            projectID,
		agentID:              agentID,
		ticketID:             ticketID,
		providerID:           providerItem.ID,
		mainWorkflowID:       mainWorkflow.ID,
		deleteWorkflowID:     deleteWorkflow.ID,
		repoReadID:           repoReadID,
		repoScopeCreateID:    repoScopeCreate.ID,
		repoDeleteID:         repoDelete.ID,
		ticketRepoScopeID:    ticketRepoScope.ID,
		ticketRepoDeleteID:   ticketRepoDelete.ID,
		scheduledJobID:       scheduledJobMain.ID,
		scheduledJobDeleteID: scheduledJobDelete.ID,
		skillMainID:          skillMain.ID,
		skillDeleteID:        skillDelete.ID,
		statusUpdateID:       statusUpdate.ID,
		statusDeleteID:       statusDelete.ID,
	}
}

func assertPlatformScopeRoute(
	t *testing.T,
	fixture *agentPlatformExpandedFixture,
	scope agentplatform.Scope,
	method string,
	path string,
	body any,
	wantStatus int,
	wantBody string,
) {
	t.Helper()

	forbiddenToken := fixture.issueToken(t, agentplatform.ScopeTicketsCreate)
	forbiddenRec := performPlatformRequest(t, fixture.server, method, path, body, forbiddenToken)
	if forbiddenRec.Code != http.StatusForbidden {
		t.Fatalf("expected %s %s without %s to return 403, got %d: %s", method, path, scope, forbiddenRec.Code, forbiddenRec.Body.String())
	}

	scopedToken := fixture.issueToken(t, scope)
	rec := performPlatformRequest(t, fixture.server, method, path, body, scopedToken)
	if rec.Code != wantStatus {
		t.Fatalf("expected %s %s with %s to return %d, got %d: %s", method, path, scope, wantStatus, rec.Code, rec.Body.String())
	}
	if wantBody != "" && !strings.Contains(rec.Body.String(), wantBody) {
		t.Fatalf("expected %s %s response to contain %q, got %s", method, path, wantBody, rec.Body.String())
	}
}

func performPlatformRequest(
	t *testing.T,
	server *Server,
	method string,
	target string,
	body any,
	token string,
) *httptest.ResponseRecorder {
	t.Helper()

	var payload string
	switch value := body.(type) {
	case nil:
		payload = ""
	case string:
		payload = value
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		payload = string(encoded)
	}

	headers := map[string]string{echo.HeaderAuthorization: "Bearer " + token}
	if payload != "" {
		headers[echo.HeaderContentType] = echo.MIMEApplicationJSON
	}
	return performJSONRequestWithHeaders(t, server, method, target, payload, headers)
}

func (f *agentPlatformExpandedFixture) issueToken(t *testing.T, scopes ...agentplatform.Scope) string {
	t.Helper()

	scopeStrings := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scopeStrings = append(scopeStrings, string(scope))
	}
	issued, err := f.platformService.IssueToken(context.Background(), agentplatform.IssueInput{
		AgentID:   f.agentID,
		ProjectID: f.projectID,
		TicketID:  f.ticketID,
		Scopes:    scopeStrings,
	})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	return issued.Token
}

func stringPtr(value string) *string {
	return &value
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
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
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
