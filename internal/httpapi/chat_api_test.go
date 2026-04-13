package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

func addAIPrincipalCookie(req *http.Request, principal chatservice.UserID) {
	req.AddCookie(&http.Cookie{
		Name:  aiPrincipalCookieName,
		Value: principal.String(),
		Path:  "/",
	})
}

func testBrowserSessionAIPrincipal() chatservice.UserID {
	return chatservice.UserID(aiPrincipalCookiePrefix + uuid.NewString())
}

func TestCurrentRequestAIPrincipalUsesHumanPrincipalInOIDCMode(t *testing.T) {
	server := &Server{auth: config.AuthConfig{Mode: config.AuthModeOIDC}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	userID := uuid.MustParse("8db7261e-e16d-458e-8926-cd01550686a5")
	setHumanPrincipal(ctx, humanauthdomain.AuthenticatedPrincipal{
		User: humanauthdomain.User{ID: userID},
	})

	got, err := server.currentRequestAIPrincipal(ctx)
	if err != nil {
		t.Fatalf("currentRequestAIPrincipal() error = %v", err)
	}
	if got != chatservice.UserID("user:"+userID.String()) {
		t.Fatalf("currentRequestAIPrincipal() = %q, want %q", got, "user:"+userID.String())
	}
}

func TestCurrentRequestAIPrincipalRequiresHumanSessionInOIDCMode(t *testing.T) {
	server := &Server{auth: config.AuthConfig{Mode: config.AuthModeOIDC}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	_, err := server.currentRequestAIPrincipal(ctx)
	if !errors.Is(err, humanauthservice.ErrUnauthorized) {
		t.Fatalf("currentRequestAIPrincipal() error = %v, want %v", err, humanauthservice.ErrUnauthorized)
	}
}

func TestCurrentRequestAIPrincipalUsesServerDefinedCookieWhenAuthDisabled(t *testing.T) {
	server := &Server{auth: config.AuthConfig{Mode: config.AuthModeDisabled}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	want := testBrowserSessionAIPrincipal()
	addAIPrincipalCookie(req, want)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	got, err := server.currentRequestAIPrincipal(ctx)
	if err != nil {
		t.Fatalf("currentRequestAIPrincipal() error = %v", err)
	}
	if got != want {
		t.Fatalf("currentRequestAIPrincipal() = %q, want %q", got, want)
	}
}

func TestCurrentRequestAIPrincipalIssuesServerDefinedCookieWhenAuthDisabled(t *testing.T) {
	server := &Server{auth: config.AuthConfig{Mode: config.AuthModeDisabled}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	got, err := server.currentRequestAIPrincipal(ctx)
	if err != nil {
		t.Fatalf("currentRequestAIPrincipal() error = %v", err)
	}
	if !strings.HasPrefix(got.String(), aiPrincipalCookiePrefix) {
		t.Fatalf("currentRequestAIPrincipal() = %q, want prefix %q", got, aiPrincipalCookiePrefix)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one ai principal cookie, got %d", len(cookies))
	}
	if cookies[0].Name != aiPrincipalCookieName || cookies[0].Value != got.String() {
		t.Fatalf("ai principal cookie = %#v, want name %q value %q", cookies[0], aiPrincipalCookieName, got)
	}
}

func TestCurrentProjectConversationUserIDUsesHumanPrincipalInOIDCMode(t *testing.T) {
	server := &Server{auth: config.AuthConfig{Mode: config.AuthModeOIDC}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	userID := uuid.MustParse("8db7261e-e16d-458e-8926-cd01550686a5")
	setHumanPrincipal(ctx, humanauthdomain.AuthenticatedPrincipal{
		User: humanauthdomain.User{ID: userID},
	})

	got, err := server.currentProjectConversationUserID(ctx)
	if err != nil {
		t.Fatalf("currentProjectConversationUserID() error = %v", err)
	}
	if got != chatservice.UserID("user:"+userID.String()) {
		t.Fatalf("currentProjectConversationUserID() = %q, want %q", got, "user:"+userID.String())
	}
}

func TestCurrentProjectConversationUserIDRejectsMissingHumanSessionInOIDCMode(t *testing.T) {
	server := &Server{auth: config.AuthConfig{Mode: config.AuthModeOIDC}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	_, err := server.currentProjectConversationUserID(ctx)
	if !errors.Is(err, humanauthservice.ErrUnauthorized) {
		t.Fatalf("currentProjectConversationUserID() error = %v, want %v", err, humanauthservice.ErrUnauthorized)
	}
}

func TestCurrentProjectConversationUserIDUsesStableLocalPrincipalWhenAuthDisabled(t *testing.T) {
	server := &Server{auth: config.AuthConfig{Mode: config.AuthModeDisabled}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	got, err := server.currentProjectConversationUserID(ctx)
	if err != nil {
		t.Fatalf("currentProjectConversationUserID() error = %v", err)
	}
	if got != chatservice.LocalProjectConversationUserID {
		t.Fatalf("currentProjectConversationUserID() = %q, want %q", got, chatservice.LocalProjectConversationUserID)
	}
}
func TestProjectConversationRoutesRequireHumanPrincipalInOIDCMode(t *testing.T) {
	projectConversationService := chatservice.NewProjectConversationService(nil, nil, nil, nil, nil, nil, nil)
	terminalService := chatservice.NewConversationTerminalService(nil, projectConversationService)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithHumanAuthConfig(config.AuthConfig{Mode: config.AuthModeOIDC}),
		WithHumanAuthService(nil, &humanauthservice.Authorizer{}),
		WithProjectConversationService(projectConversationService),
		WithConversationTerminalService(terminalService),
	)

	conversationID := uuid.NewString()
	interruptID := uuid.NewString()

	for _, tc := range []struct {
		name   string
		method string
		target string
		body   string
	}{
		{
			name:   "stream conversation",
			method: http.MethodGet,
			target: "/api/v1/chat/conversations/" + conversationID + "/stream",
		},
		{
			name:   "get conversation",
			method: http.MethodGet,
			target: "/api/v1/chat/conversations/" + conversationID,
		},
		{
			name:   "list entries",
			method: http.MethodGet,
			target: "/api/v1/chat/conversations/" + conversationID + "/entries",
		},
		{
			name:   "workspace metadata",
			method: http.MethodGet,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace",
		},
		{
			name:   "workspace sync",
			method: http.MethodPost,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/sync",
		},
		{
			name:   "workspace tree",
			method: http.MethodGet,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/tree?repo_path=openase",
		},
		{
			name:   "workspace search",
			method: http.MethodGet,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/search?repo_path=openase&q=readme",
		},
		{
			name:   "workspace file",
			method: http.MethodGet,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/file?repo_path=openase&path=README.md",
		},
		{
			name:   "workspace file save",
			method: http.MethodPut,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/file",
			body:   `{"repo_path":"openase","path":"README.md","base_revision":"rev-1","content":"hello\n","encoding":"utf-8","line_ending":"lf"}`,
		},
		{
			name:   "workspace file patch",
			method: http.MethodGet,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/file-patch?repo_path=openase&path=README.md",
		},
		{
			name:   "workspace diff",
			method: http.MethodGet,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace-diff",
		},
		{
			name:   "workspace git stage",
			method: http.MethodPost,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/git-stage",
			body:   `{"repo_path":"openase","path":"README.md"}`,
		},
		{
			name:   "workspace git stage all",
			method: http.MethodPost,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/git-stage-all",
			body:   `{"repo_path":"openase"}`,
		},
		{
			name:   "workspace git unstage",
			method: http.MethodPost,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/git-unstage",
			body:   `{"repo_path":"openase","path":"README.md"}`,
		},
		{
			name:   "workspace git commit",
			method: http.MethodPost,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/git-commit",
			body:   `{"repo_path":"openase","message":"feat: test"}`,
		},
		{
			name:   "workspace git discard",
			method: http.MethodPost,
			target: "/api/v1/chat/conversations/" + conversationID + "/workspace/git-discard",
			body:   `{"repo_path":"openase","path":"README.md"}`,
		},
		{
			name:   "create terminal session",
			method: http.MethodPost,
			target: "/api/v1/chat/conversations/" + conversationID + "/terminal-sessions",
			body:   `{"mode":"shell"}`,
		},
		{
			name:   "attach terminal session",
			method: http.MethodGet,
			target: "/api/v1/chat/conversations/" + conversationID + "/terminal-sessions/" + uuid.NewString() + "/attach?attach_token=token-1",
		},
		{
			name:   "start turn",
			method: http.MethodPost,
			target: "/api/v1/chat/conversations/" + conversationID + "/turns",
			body:   `{"message":"continue"}`,
		},
		{
			name:   "interrupt turn",
			method: http.MethodPost,
			target: "/api/v1/chat/conversations/" + conversationID + "/interrupt-turn",
		},
		{
			name:   "delete conversation",
			method: http.MethodDelete,
			target: "/api/v1/chat/conversations/" + conversationID,
		},
		{
			name:   "close runtime",
			method: http.MethodDelete,
			target: "/api/v1/chat/conversations/" + conversationID + "/runtime",
		},
		{
			name:   "respond interrupt",
			method: http.MethodPost,
			target: "/api/v1/chat/conversations/" + conversationID + "/interrupts/" + interruptID + "/respond",
			body:   `{"decision":"approve"}`,
		},
		{
			name:   "project mux stream",
			method: http.MethodGet,
			target: "/api/v1/chat/projects/" + uuid.NewString() + "/conversations/stream",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := performJSONRequest(t, server, tc.method, tc.target, tc.body)
			if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "HUMAN_SESSION_REQUIRED") {
				t.Fatalf("expected 401 HUMAN_SESSION_REQUIRED, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestProjectConversationWorkspaceGitStageRouteRequiresConversationUpdatePermissionInOIDCMode(t *testing.T) {
	t.Parallel()

	fixture := newHumanAuthFixture(t)
	routeFixture := setupProjectConversationTerminalRouteFixture(t, fixture.client)

	unauthorizedToken, unauthorizedCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:   "viewer@example.com",
		displayName: "Viewer",
	})

	unauthorizedRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/chat/conversations/"+routeFixture.conversation.ID.String()+"/workspace/git-stage",
		`{"repo_path":"backend","path":"src/main.go"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + unauthorizedToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": unauthorizedCSRF,
			"User-Agent":     "ConversationGitAuthTest/1.0",
		},
	)
	assertAPIErrorResponse(t, unauthorizedRec, http.StatusForbidden, "AUTHORIZATION_DENIED", "required permission is missing")

	projectItem, err := fixture.client.Project.Get(context.Background(), routeFixture.conversation.ProjectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	authorizedToken, authorizedCSRF := fixture.createSession(t, humanFixtureSessionInput{
		userEmail:      "project-admin@example.com",
		displayName:    "Project Admin",
		orgID:          projectItem.OrganizationID,
		projectID:      routeFixture.conversation.ProjectID,
		projectRoleKey: "project_admin",
	})

	authorizedRec := fixture.requestJSON(
		t,
		http.MethodPost,
		"/api/v1/chat/conversations/"+routeFixture.conversation.ID.String()+"/workspace/git-stage",
		`{"repo_path":"backend","path":"src/main.go"}`,
		map[string]string{
			"Cookie":         humanSessionCookieName + "=" + authorizedToken,
			"Origin":         "http://example.com",
			"X-OpenASE-CSRF": authorizedCSRF,
			"User-Agent":     "ConversationGitAuthTest/1.0",
		},
	)
	if authorizedRec.Code == http.StatusForbidden {
		t.Fatalf("expected authorized request to pass authorization, got 403: %s", authorizedRec.Body.String())
	}
}

func TestMapProjectConversationResponseIncludesProviderAnchors(t *testing.T) {
	threadID := "thread-1"
	lastTurnID := "turn-9"
	threadStatus := "active"
	response := (&Server{}).mapProjectConversationResponse(context.Background(), chatdomain.Conversation{
		ID:                        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		ProjectID:                 uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
		UserID:                    "user:conversation",
		Source:                    chatdomain.SourceProjectSidebar,
		ProviderID:                uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
		Status:                    chatdomain.ConversationStatusInterrupted,
		ProviderThreadID:          &threadID,
		LastTurnID:                &lastTurnID,
		ProviderThreadStatus:      &threadStatus,
		ProviderThreadActiveFlags: []string{"waitingOnApproval"},
		RollingSummary:            "summary",
		LastActivityAt:            time.Unix(1, 0).UTC(),
		CreatedAt:                 time.Unix(2, 0).UTC(),
		UpdatedAt:                 time.Unix(3, 0).UTC(),
	})

	if response["provider_thread_id"] != "thread-1" || response["last_turn_id"] != "turn-9" || response["provider_thread_status"] != "active" {
		t.Fatalf("unexpected provider anchors: %+v", response)
	}
	flags, ok := response["provider_thread_active_flags"].([]string)
	if !ok || len(flags) != 1 || flags[0] != "waitingOnApproval" {
		t.Fatalf("unexpected provider active flags: %+v", response["provider_thread_active_flags"])
	}
}

func TestChatRouteStreamsTicketDetailContext(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName(catalogdomain.LocalMachineName).
		SetHost(catalogdomain.LocalMachineHost).
		SetPort(22).
		SetDescription("Control-plane local execution host.").
		SetStatus("online").
		SetResources(map[string]any{
			"transport": "local",
			"monitor": map[string]any{
				"l4": map[string]any{
					"checked_at": time.Now().UTC().Format(time.RFC3339),
					"claude_code": map[string]any{
						"installed":   true,
						"auth_status": string(catalogdomain.MachineAgentAuthStatusLoggedIn),
						"auth_mode":   string(catalogdomain.MachineAgentAuthModeLogin),
						"ready":       true,
					},
				},
			},
		}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
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
		AddPickupStatusIDs(backlogID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	priority := ticketservice.PriorityMedium

	ticketItem, err := newTicketService(client).Create(ctx, ticketservice.CreateInput{
		ProjectID:   project.ID,
		Title:       "Implement ephemeral chat",
		Description: "Explain why the last hook failed and propose smaller follow-up tickets.",
		Priority:    &priority,
		Type:        "feature",
		WorkflowID:  &workflowItem.ID,
		CreatedBy:   "user:codex",
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetEventType("pr.opened").
		SetMessage("Collected failing test output").
		SetMetadata(map[string]any{"stream": "stdout"}).
		Save(ctx); err != nil {
		t.Fatalf("create activity event: %v", err)
	}
	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetEventType("hook.failed").
		SetMessage("go test ./... failed in auth package").
		SetMetadata(map[string]any{"hook_name": "ticket.on_complete"}).
		Save(ctx); err != nil {
		t.Fatalf("create hook event: %v", err)
	}

	catalogSvc := catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil)
	providerInput, err := catalogdomain.ParseCreateAgentProvider(org.ID, catalogdomain.AgentProviderInput{
		MachineID:   localMachine.ID.String(),
		Name:        "Claude Code",
		AdapterType: "claude-code-cli",
		CliCommand:  "claude",
		AuthConfig:  map[string]any{"anthropic_api_key": "test-key"},
		ModelName:   "claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatalf("parse provider input: %v", err)
	}
	providerItem, err := catalogSvc.CreateAgentProvider(ctx, providerInput)
	if err != nil {
		t.Fatalf("create agent provider: %v", err)
	}

	adapter := &fakeClaudeAdapter{
		session: newFakeClaudeSession(
			[]provider.ClaudeCodeEvent{
				{
					Kind:      provider.ClaudeCodeEventKindSystem,
					SessionID: "sess-ephemeral-1",
				},
				{
					Kind:    provider.ClaudeCodeEventKindAssistant,
					Message: mustMarshalJSON(t, map[string]any{"role": "assistant", "content": []map[string]any{{"type": "text", "text": "The ticket failed because `go test ./...` is red."}}}),
				},
				{
					Kind:         provider.ClaudeCodeEventKindResult,
					SessionID:    "sess-ephemeral-1",
					NumTurns:     1,
					TotalCostUSD: testFloatPointer(0.03),
				},
			},
			nil,
		),
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalogSvc,
		nil,
		WithChatService(chatservice.NewService(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			chatservice.NewClaudeRuntime(adapter),
			catalogSvc,
			newTicketService(client),
			staticWorkflowReader{},
			nil,
			"",
		)),
	)

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	requestBody := mustMarshalJSON(t, map[string]any{
		"message":     "Why did this fail?",
		"source":      "ticket_detail",
		"provider_id": providerItem.ID.String(),
		"context": map[string]any{
			"project_id": project.ID.String(),
			"ticket_id":  ticketItem.ID.String(),
		},
	})
	req, err := http.NewRequest(http.MethodPost, testServer.URL+"/api/v1/chat", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	textBody := string(body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, textBody)
	}
	if contentType := resp.Header.Get(echo.HeaderContentType); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}
	if !strings.Contains(textBody, "event: message\n") {
		t.Fatalf("expected message event in stream, got %q", textBody)
	}
	if !strings.Contains(textBody, "event: session\n") ||
		!strings.Contains(textBody, "\"session_id\":\"") ||
		!strings.Contains(textBody, "\"provider_resume_supported\":false") ||
		!strings.Contains(textBody, "\"resume_scope\":\"process_local\"") {
		t.Fatalf("expected session event with session id, got %q", textBody)
	}
	if !strings.Contains(textBody, "The ticket failed because") {
		t.Fatalf("expected assistant message in stream, got %q", textBody)
	}
	if !strings.Contains(textBody, "event: done\n") || !strings.Contains(textBody, "\"session_id\":\"") {
		t.Fatalf("expected done event with session id, got %q", textBody)
	}

	if adapter.lastSpec.ResumeSessionID != nil {
		t.Fatalf("expected first turn not to resume an existing session, got %+v", adapter.lastSpec.ResumeSessionID)
	}
	if adapter.lastSpec.MaxTurns == nil || *adapter.lastSpec.MaxTurns != chatservice.DefaultMaxTurns {
		t.Fatalf("expected max turns %d, got %+v", chatservice.DefaultMaxTurns, adapter.lastSpec.MaxTurns)
	}
	if adapter.lastSpec.MaxBudgetUSD == nil || *adapter.lastSpec.MaxBudgetUSD != chatservice.DefaultMaxBudgetUSD {
		t.Fatalf("expected max budget %.2f, got %+v", chatservice.DefaultMaxBudgetUSD, adapter.lastSpec.MaxBudgetUSD)
	}
	if !adapter.lastSpec.IncludePartialMessages {
		t.Fatalf("expected IncludePartialMessages to be true")
	}
	if !strings.Contains(adapter.lastSpec.AppendSystemPrompt, ticketItem.Identifier) {
		t.Fatalf("expected system prompt to include ticket identifier, got %q", adapter.lastSpec.AppendSystemPrompt)
	}
	if !strings.Contains(adapter.lastSpec.AppendSystemPrompt, "go test ./... failed in auth package") {
		t.Fatalf("expected hook history in system prompt, got %q", adapter.lastSpec.AppendSystemPrompt)
	}
	if !strings.Contains(adapter.lastSpec.AppendSystemPrompt, "Do not claim that you have already performed platform write operations.") ||
		!strings.Contains(adapter.lastSpec.AppendSystemPrompt, "Do not output proposal JSON.") {
		t.Fatalf("expected direct-execution instructions in system prompt, got %q", adapter.lastSpec.AppendSystemPrompt)
	}
	if !slicesContain(adapter.lastSpec.Environment, "ANTHROPIC_API_KEY=test-key") {
		t.Fatalf("expected auth config env injection, got %v", adapter.lastSpec.Environment)
	}
	if len(adapter.session.sent) != 1 || adapter.session.sent[0].Prompt != "Why did this fail?" {
		t.Fatalf("expected sent prompt to round-trip, got %+v", adapter.session.sent)
	}
}

func TestChatDeleteRouteAndErrorMappings(t *testing.T) {
	serverWithoutChat := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	unavailableRec := performJSONRequest(t, serverWithoutChat, http.MethodDelete, "/api/v1/chat/sess-1", "")
	if unavailableRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected unavailable delete route 503, got %d: %s", unavailableRec.Code, unavailableRec.Body.String())
	}

	chatSvc := chatservice.NewService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		nil,
		nil,
		nil,
		nil,
		"",
	)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithChatService(chatSvc),
	)

	invalidRec := performJSONRequest(t, server, http.MethodDelete, "/api/v1/chat/%20%20", "")
	if invalidRec.Code != http.StatusBadRequest || !strings.Contains(invalidRec.Body.String(), "INVALID_SESSION_ID") {
		t.Fatalf("expected invalid session response, got %d: %s", invalidRec.Code, invalidRec.Body.String())
	}

	validRec := performJSONRequest(t, server, http.MethodDelete, "/api/v1/chat/sess-1", "")
	if validRec.Code != http.StatusNoContent {
		t.Fatalf("expected successful delete 204, got %d: %s", validRec.Code, validRec.Body.String())
	}

	for _, tc := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unavailable", err: chatservice.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "SERVICE_UNAVAILABLE"},
		{name: "source", err: chatservice.ErrSourceUnsupported, wantStatus: http.StatusBadRequest, wantCode: "INVALID_CHAT_SOURCE"},
		{name: "session provider mismatch", err: chatservice.ErrSessionProviderMismatch, wantStatus: http.StatusConflict, wantCode: "CHAT_SESSION_PROVIDER_MISMATCH"},
		{name: "session turn limit", err: chatservice.ErrSessionTurnLimitReached, wantStatus: http.StatusConflict, wantCode: "CHAT_SESSION_LIMIT_REACHED"},
		{name: "session budget limit", err: chatservice.ErrSessionBudgetExceeded, wantStatus: http.StatusConflict, wantCode: "CHAT_SESSION_LIMIT_REACHED"},
		{name: "provider", err: chatservice.ErrProviderNotFound, wantStatus: http.StatusConflict, wantCode: "CHAT_PROVIDER_NOT_CONFIGURED"},
		{name: "provider unavailable", err: chatservice.ErrProviderUnavailable, wantStatus: http.StatusConflict, wantCode: "CHAT_PROVIDER_UNAVAILABLE"},
		{name: "provider unsupported", err: chatservice.ErrProviderUnsupported, wantStatus: http.StatusConflict, wantCode: "CHAT_PROVIDER_UNSUPPORTED"},
		{name: "session missing", err: chatservice.ErrSessionNotFound, wantStatus: http.StatusNotFound, wantCode: "CHAT_SESSION_NOT_FOUND"},
		{name: "ticket", err: ticketservice.ErrTicketNotFound, wantStatus: http.StatusNotFound, wantCode: "CHAT_CONTEXT_NOT_FOUND"},
		{name: "workflow", err: workflowservice.ErrWorkflowNotFound, wantStatus: http.StatusNotFound, wantCode: "CHAT_CONTEXT_NOT_FOUND"},
		{name: "catalog", err: catalogservice.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "CHAT_CONTEXT_NOT_FOUND"},
		{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			if err := writeChatError(ctx, tc.err); err != nil {
				t.Fatalf("writeChatError() error = %v", err)
			}
			if rec.Code != tc.wantStatus || !strings.Contains(rec.Body.String(), tc.wantCode) {
				t.Fatalf("writeChatError(%s) = %d %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestChatRouteLogsStructuredStartFailures(t *testing.T) {
	projectID := uuid.New()
	providerID := uuid.New()
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	chatSvc := chatservice.NewService(
		logger,
		chatRuntimeStub{
			startErr: errors.New("provider bootstrap exploded"),
		},
		chatCatalogStub{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: uuid.New(),
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:          providerID,
					Name:        "Codex",
					AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
					Available:   true,
				},
			},
		},
		chatTicketStub{},
		chatWorkflowStub{},
		nil,
		"",
	)

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithChatService(chatSvc),
	)

	body := mustMarshalJSON(t, map[string]any{
		"message":     "Summarize the current project status for me.",
		"source":      "project_sidebar",
		"provider_id": providerID.String(),
		"context": map[string]any{
			"project_id": projectID.String(),
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	principal := testBrowserSessionAIPrincipal()
	addAIPrincipalCookie(req, principal)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}

	logOutput := logBuffer.String()
	for _, want := range []string{
		"chat start failed",
		"provider bootstrap exploded",
		"chat_source=project_sidebar",
		"chat_project_id=" + projectID.String(),
		"chat_provider_id=" + providerID.String(),
		"chat_session_id=",
		"chat_user_id=" + principal.String(),
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("expected log output to contain %q, got %q", want, logOutput)
		}
	}
}

func TestWriteProjectConversationErrorMappings(t *testing.T) {
	for _, tc := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "turn active", err: chatservice.ErrConversationTurnActive, wantStatus: http.StatusConflict, wantCode: "PROJECT_CONVERSATION_TURN_ALREADY_ACTIVE"},
		{name: "workspace dirty", err: chatdomain.ErrWorkspaceDirty, wantStatus: http.StatusConflict, wantCode: "PROJECT_CONVERSATION_WORKSPACE_DIRTY"},
		{name: "workspace delete failed", err: chatdomain.ErrWorkspaceDeleteFailed, wantStatus: http.StatusConflict, wantCode: "PROJECT_CONVERSATION_WORKSPACE_DELETE_FAILED"},
		{name: "workspace path conflict", err: chatdomain.ErrWorkspacePathConflict, wantStatus: http.StatusConflict, wantCode: "PROJECT_CONVERSATION_WORKSPACE_PATH_CONFLICT"},
		{name: "generic conflict", err: chatservice.ErrConversationConflict, wantStatus: http.StatusConflict, wantCode: "CHAT_CONVERSATION_CONFLICT"},
		{name: "missing conversation", err: chatservice.ErrConversationNotFound, wantStatus: http.StatusNotFound, wantCode: "CHAT_CONVERSATION_NOT_FOUND"},
		{name: "runtime missing", err: chatservice.ErrConversationRuntimeAbsent, wantStatus: http.StatusConflict, wantCode: "CHAT_CONVERSATION_RUNTIME_UNAVAILABLE"},
		{name: "workspace unavailable", err: chatservice.ErrProjectConversationWorkspaceUnavailable, wantStatus: http.StatusConflict, wantCode: "PROJECT_CONVERSATION_WORKSPACE_UNAVAILABLE"},
		{name: "workspace path invalid", err: chatservice.ErrProjectConversationWorkspacePathInvalid, wantStatus: http.StatusBadRequest, wantCode: "PROJECT_CONVERSATION_WORKSPACE_PATH_INVALID"},
		{name: "workspace exists", err: chatservice.ErrProjectConversationWorkspaceEntryExists, wantStatus: http.StatusConflict, wantCode: "PROJECT_CONVERSATION_WORKSPACE_FILE_EXISTS"},
		{name: "workspace missing", err: chatservice.ErrProjectConversationWorkspaceEntryNotFound, wantStatus: http.StatusNotFound, wantCode: "PROJECT_CONVERSATION_WORKSPACE_NOT_FOUND"},
		{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			if err := writeProjectConversationError(ctx, tc.err); err != nil {
				t.Fatalf("writeProjectConversationError() error = %v", err)
			}
			if rec.Code != tc.wantStatus || !strings.Contains(rec.Body.String(), tc.wantCode) {
				t.Fatalf("writeProjectConversationError(%s) = %d %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestProjectConversationStreamRouteKeepsParallelConnectionsIsolated(t *testing.T) {
	client := openTestEntClient(t)
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
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	principal := testBrowserSessionAIPrincipal()

	repoStore := chatrepo.NewEntRepository(client)
	firstConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     principal.String(),
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	secondConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     principal.String(),
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create second conversation: %v", err)
	}

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repoStore,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		config.ServerConfig{Port: 40023, WriteTimeout: time.Second},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	streamCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	firstReq, err := http.NewRequestWithContext(
		streamCtx,
		http.MethodGet,
		testServer.URL+"/api/v1/chat/conversations/"+firstConversation.ID.String()+"/stream",
		nil,
	)
	if err != nil {
		t.Fatalf("new first stream request: %v", err)
	}
	secondReq, err := http.NewRequestWithContext(
		streamCtx,
		http.MethodGet,
		testServer.URL+"/api/v1/chat/conversations/"+secondConversation.ID.String()+"/stream",
		nil,
	)
	if err != nil {
		t.Fatalf("new second stream request: %v", err)
	}
	addAIPrincipalCookie(firstReq, principal)
	addAIPrincipalCookie(secondReq, principal)

	firstResp, err := http.DefaultClient.Do(firstReq)
	if err != nil {
		t.Fatalf("open first stream: %v", err)
	}
	defer func() {
		_ = firstResp.Body.Close()
	}()
	secondResp, err := http.DefaultClient.Do(secondReq)
	if err != nil {
		t.Fatalf("open second stream: %v", err)
	}
	defer func() {
		_ = secondResp.Body.Close()
	}()

	firstReader := bufio.NewReader(firstResp.Body)
	secondReader := bufio.NewReader(secondResp.Body)

	firstSession := readProjectConversationSSEFrame(t, firstReader)
	if firstSession.Event != "session" || !strings.Contains(firstSession.Data, firstConversation.ID.String()) {
		t.Fatalf("first session frame = %+v, want conversation %s", firstSession, firstConversation.ID)
	}
	secondSession := readProjectConversationSSEFrame(t, secondReader)
	if secondSession.Event != "session" || !strings.Contains(secondSession.Data, secondConversation.ID.String()) {
		t.Fatalf("second session frame = %+v, want conversation %s", secondSession, secondConversation.ID)
	}

	if _, err := projectConversationService.AppendSystemEntry(
		ctx,
		chatservice.LocalProjectConversationUserID,
		firstConversation.ID,
		nil,
		testTaskNotificationPayload("conversation-1"),
	); err != nil {
		t.Fatalf("append first action result: %v", err)
	}

	firstMessage := readProjectConversationSSEFrame(t, firstReader)
	if firstMessage.Event != "message" ||
		!strings.Contains(firstMessage.Data, "\"type\":\"task_notification\"") ||
		!strings.Contains(firstMessage.Data, "\"marker\":\"conversation-1\"") {
		t.Fatalf("first message frame = %+v", firstMessage)
	}

	if _, err := projectConversationService.AppendSystemEntry(
		ctx,
		chatservice.LocalProjectConversationUserID,
		secondConversation.ID,
		nil,
		testTaskNotificationPayload("conversation-2"),
	); err != nil {
		t.Fatalf("append second action result: %v", err)
	}

	secondMessage := readProjectConversationSSEFrame(t, secondReader)
	if secondMessage.Event != "message" ||
		!strings.Contains(secondMessage.Data, "\"type\":\"task_notification\"") ||
		!strings.Contains(secondMessage.Data, "\"marker\":\"conversation-2\"") {
		t.Fatalf("second message frame = %+v", secondMessage)
	}
}

func TestProjectConversationRoutesReturnStableTitleAndBackfillLegacyConversations(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	principal := testBrowserSessionAIPrincipal()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-chat-routes").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-chat-routes").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	repoStore := chatrepo.NewEntRepository(client)
	legacyConversation, err := client.ChatConversation.Create().
		SetProjectID(project.ID).
		SetUserID(principal.String()).
		SetSource(string(chatdomain.SourceProjectSidebar)).
		SetProviderID(uuid.New()).
		SetStatus(string(chatdomain.ConversationStatusActive)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create legacy conversation: %v", err)
	}
	legacyTurn, err := client.ChatTurn.Create().
		SetConversationID(legacyConversation.ID).
		SetTurnIndex(1).
		SetStatus(string(chatdomain.TurnStatusCompleted)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create legacy turn: %v", err)
	}
	if _, err := client.ChatEntry.Create().
		SetConversationID(legacyConversation.ID).
		SetTurnID(legacyTurn.ID).
		SetSeq(0).
		SetKind(string(chatdomain.EntryKindUserMessage)).
		SetPayloadJSON(map[string]any{
			"role":    "user",
			"content": "固定这个对话标题。后面的 summary 只保留摘要语义。",
		}).
		Save(ctx); err != nil {
		t.Fatalf("create legacy entry: %v", err)
	}

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repoStore,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		config.ServerConfig{Port: 40023, WriteTimeout: time.Second},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	listReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/chat/conversations?project_id="+project.ID.String(),
		nil,
	)
	addAIPrincipalCookie(listReq, principal)
	listRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), `"title":"固定这个对话标题。"`) {
		t.Fatalf("expected list response to include stable title, got %s", listRec.Body.String())
	}

	getReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/chat/conversations/"+legacyConversation.ID.String(),
		nil,
	)
	addAIPrincipalCookie(getReq, principal)
	getRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), `"title":"固定这个对话标题。"`) {
		t.Fatalf("expected get response to include stable title, got %s", getRec.Body.String())
	}

	reloadedConversation, err := client.ChatConversation.Get(ctx, legacyConversation.ID)
	if err != nil {
		t.Fatalf("reload legacy conversation: %v", err)
	}
	if got, want := reloadedConversation.Title, "固定这个对话标题。"; got != want {
		t.Fatalf("persisted title = %q, want %q", got, want)
	}
}

func TestProjectConversationMuxStreamRouteMultiplexesConversationsWithinOneProject(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-mux").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Mux").
		SetSlug("openase-mux").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	principal := testBrowserSessionAIPrincipal()

	repoStore := chatrepo.NewEntRepository(client)
	firstConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     principal.String(),
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	secondConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     principal.String(),
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create second conversation: %v", err)
	}

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repoStore,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		config.ServerConfig{Port: 40023, WriteTimeout: time.Second},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	streamCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		streamCtx,
		http.MethodGet,
		testServer.URL+"/api/v1/chat/projects/"+project.ID.String()+"/conversations/stream",
		nil,
	)
	if err != nil {
		t.Fatalf("new mux stream request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open mux stream: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	reader := bufio.NewReader(resp.Body)
	firstSession := readProjectConversationSSEFrame(t, reader)
	secondSession := readProjectConversationSSEFrame(t, reader)
	if firstSession.Event != "session" || secondSession.Event != "session" {
		t.Fatalf("expected initial mux session frames, got %+v and %+v", firstSession, secondSession)
	}
	if !strings.Contains(firstSession.Data, "\"conversation_id\":\""+firstConversation.ID.String()+"\"") &&
		!strings.Contains(secondSession.Data, "\"conversation_id\":\""+firstConversation.ID.String()+"\"") {
		t.Fatalf("expected one initial mux frame for first conversation, got %+v and %+v", firstSession, secondSession)
	}
	if !strings.Contains(firstSession.Data, "\"conversation_id\":\""+secondConversation.ID.String()+"\"") &&
		!strings.Contains(secondSession.Data, "\"conversation_id\":\""+secondConversation.ID.String()+"\"") {
		t.Fatalf("expected one initial mux frame for second conversation, got %+v and %+v", firstSession, secondSession)
	}

	if _, err := projectConversationService.AppendSystemEntry(
		ctx,
		chatservice.LocalProjectConversationUserID,
		firstConversation.ID,
		nil,
		testTaskNotificationPayload("conversation-1"),
	); err != nil {
		t.Fatalf("append first action result: %v", err)
	}
	firstMessage := readProjectConversationSSEFrame(t, reader)
	if firstMessage.Event != "message" ||
		!strings.Contains(firstMessage.Data, "\"conversation_id\":\""+firstConversation.ID.String()+"\"") ||
		!strings.Contains(firstMessage.Data, "\"marker\":\"conversation-1\"") {
		t.Fatalf("first mux message frame = %+v", firstMessage)
	}

	if _, err := projectConversationService.AppendSystemEntry(
		ctx,
		chatservice.LocalProjectConversationUserID,
		secondConversation.ID,
		nil,
		testTaskNotificationPayload("conversation-2"),
	); err != nil {
		t.Fatalf("append second action result: %v", err)
	}
	secondMessage := readProjectConversationSSEFrame(t, reader)
	if secondMessage.Event != "message" ||
		!strings.Contains(secondMessage.Data, "\"conversation_id\":\""+secondConversation.ID.String()+"\"") ||
		!strings.Contains(secondMessage.Data, "\"marker\":\"conversation-2\"") {
		t.Fatalf("second mux message frame = %+v", secondMessage)
	}
}

func TestProjectConversationTerminalSessionCreateRoute(t *testing.T) {
	client := openTestEntClient(t)
	fixture := setupProjectConversationTerminalRouteFixture(t, client)

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		chatrepo.NewEntRepository(client),
		fixture.catalog,
		nil,
		nil,
		nil,
		nil,
	)
	terminalService := chatservice.NewConversationTerminalService(nil, projectConversationService)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		fixture.catalog,
		nil,
		WithProjectConversationService(projectConversationService),
		WithConversationTerminalService(terminalService),
	)

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/terminal-sessions",
		`{"mode":"shell","repo_path":"backend","cwd_path":"src","cols":90,"rows":33}`,
	)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	wantCWD := filepath.Join(fixture.repoPath, "src")
	if !strings.Contains(rec.Body.String(), `"cwd":"`+wantCWD+`"`) || !strings.Contains(rec.Body.String(), `"mode":"shell"`) || !strings.Contains(rec.Body.String(), `"attach_token":"`) {
		t.Fatalf("unexpected create terminal response: %s", rec.Body.String())
	}
}

func TestProjectConversationTerminalSessionCreateRouteRejectsInvalidMode(t *testing.T) {
	client := openTestEntClient(t)
	fixture := setupProjectConversationTerminalRouteFixture(t, client)

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		chatrepo.NewEntRepository(client),
		fixture.catalog,
		nil,
		nil,
		nil,
		nil,
	)
	terminalService := chatservice.NewConversationTerminalService(nil, projectConversationService)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		fixture.catalog,
		nil,
		WithProjectConversationService(projectConversationService),
		WithConversationTerminalService(terminalService),
	)

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/terminal-sessions",
		`{"mode":"tmux"}`,
	)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "mode must be") {
		t.Fatalf("expected invalid mode rejection, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectConversationWorkspaceGitStageAndCommitRoutes(t *testing.T) {
	client := openTestEntClient(t)
	fixture := setupProjectConversationTerminalRouteFixture(t, client)

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		chatrepo.NewEntRepository(client),
		fixture.catalog,
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		fixture.catalog,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	if err := os.WriteFile(filepath.Join(fixture.repoPath, "src", "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatalf("write dirty workspace file: %v", err)
	}

	stageRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace/git-stage",
		`{"repo_path":"backend","path":"src/main.go"}`,
	)
	if stageRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from stage, got %d: %s", stageRec.Code, stageRec.Body.String())
	}

	diffRec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace-diff",
		"",
	)
	if diffRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from diff, got %d: %s", diffRec.Code, diffRec.Body.String())
	}
	if !strings.Contains(diffRec.Body.String(), `"path":"src/main.go"`) ||
		!strings.Contains(diffRec.Body.String(), `"staged":true`) ||
		!strings.Contains(diffRec.Body.String(), `"unstaged":false`) {
		t.Fatalf("expected staged diff metadata in body, got %s", diffRec.Body.String())
	}

	commitRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace/git-commit",
		`{"repo_path":"backend","message":"feat: commit staged route test"}`,
	)
	if commitRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from commit, got %d: %s", commitRec.Code, commitRec.Body.String())
	}
	if !strings.Contains(commitRec.Body.String(), `feat: commit staged route test`) {
		t.Fatalf("expected commit output in body, got %s", commitRec.Body.String())
	}

	logOutput := runConversationGitCommand(t, "", "git", "-C", fixture.repoPath, "log", "-1", "--format=%s")
	if strings.TrimSpace(logOutput) != "feat: commit staged route test" {
		t.Fatalf("head subject = %q", strings.TrimSpace(logOutput))
	}
}

func TestProjectConversationWorkspaceGitStageAllAndUnstageRoutes(t *testing.T) {
	client := openTestEntClient(t)
	fixture := setupProjectConversationTerminalRouteFixture(t, client)

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		chatrepo.NewEntRepository(client),
		fixture.catalog,
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		fixture.catalog,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	if err := os.WriteFile(filepath.Join(fixture.repoPath, "src", "main.go"), []byte("package main\n\nfunc changed() {}\n"), 0o600); err != nil {
		t.Fatalf("write modified file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixture.repoPath, "src", "extra.go"), []byte("package main\n\nconst extra = true\n"), 0o600); err != nil {
		t.Fatalf("write new file: %v", err)
	}

	stageAllRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace/git-stage-all",
		`{"repo_path":"backend"}`,
	)
	if stageAllRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from stage all, got %d: %s", stageAllRec.Code, stageAllRec.Body.String())
	}

	diffRec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace-diff",
		"",
	)
	if diffRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from diff, got %d: %s", diffRec.Code, diffRec.Body.String())
	}
	if !strings.Contains(diffRec.Body.String(), `"path":"src/main.go"`) ||
		!strings.Contains(diffRec.Body.String(), `"path":"src/extra.go"`) ||
		!strings.Contains(diffRec.Body.String(), `"staged":true`) {
		t.Fatalf("expected stage-all metadata in body, got %s", diffRec.Body.String())
	}

	unstageRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace/git-unstage",
		`{"repo_path":"backend","path":"src/main.go"}`,
	)
	if unstageRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from unstage, got %d: %s", unstageRec.Code, unstageRec.Body.String())
	}

	diffRec = performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace-diff",
		"",
	)
	if diffRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from diff after unstage, got %d: %s", diffRec.Code, diffRec.Body.String())
	}
	if !strings.Contains(diffRec.Body.String(), `"path":"src/main.go"`) ||
		!strings.Contains(diffRec.Body.String(), `"unstaged":true`) ||
		!strings.Contains(diffRec.Body.String(), `"path":"src/extra.go"`) {
		t.Fatalf("expected mixed staged/unstaged metadata in body, got %s", diffRec.Body.String())
	}

	unstageAllRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace/git-unstage",
		`{"repo_path":"backend"}`,
	)
	if unstageAllRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from unstage all, got %d: %s", unstageAllRec.Code, unstageAllRec.Body.String())
	}

	statusOutput := runConversationGitCommand(t, "", "git", "-C", fixture.repoPath, "status", "--porcelain=v1")
	if strings.Contains(statusOutput, "M  src/main.go") || strings.Contains(statusOutput, "A  src/extra.go") {
		t.Fatalf("expected staged changes to be cleared, got %q", statusOutput)
	}
	if !strings.Contains(statusOutput, " M src/main.go") || !strings.Contains(statusOutput, "?? src/extra.go") {
		t.Fatalf("expected unstaged status after unstage all, got %q", statusOutput)
	}
}

func TestProjectConversationWorkspaceGitDiscardRouteRestoresFile(t *testing.T) {
	client := openTestEntClient(t)
	fixture := setupProjectConversationTerminalRouteFixture(t, client)

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		chatrepo.NewEntRepository(client),
		fixture.catalog,
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		fixture.catalog,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	targetPath := filepath.Join(fixture.repoPath, "src", "main.go")
	if err := os.WriteFile(targetPath, []byte("package main\n\nfunc changed() {}\n"), 0o600); err != nil {
		t.Fatalf("write dirty workspace file: %v", err)
	}

	stageRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace/git-stage",
		`{"repo_path":"backend","path":"src/main.go"}`,
	)
	if stageRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from stage, got %d: %s", stageRec.Code, stageRec.Body.String())
	}

	discardRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace/git-discard",
		`{"repo_path":"backend","path":"src/main.go"}`,
	)
	if discardRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from discard, got %d: %s", discardRec.Code, discardRec.Body.String())
	}

	diffRec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/workspace-diff",
		"",
	)
	if diffRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from diff, got %d: %s", diffRec.Code, diffRec.Body.String())
	}
	if strings.Contains(diffRec.Body.String(), `"path":"src/main.go"`) {
		t.Fatalf("expected discarded file to disappear from diff, got %s", diffRec.Body.String())
	}

	//nolint:gosec // Test reads a fixture-controlled temp repo file.
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(content) != "package main\n" {
		t.Fatalf("restored file content = %q", string(content))
	}
}

func TestProjectConversationTerminalAttachWebsocketFlow(t *testing.T) {
	client := openTestEntClient(t)
	fixture := setupProjectConversationTerminalRouteFixture(t, client)

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		chatrepo.NewEntRepository(client),
		fixture.catalog,
		nil,
		nil,
		nil,
		nil,
	)
	terminalService := chatservice.NewConversationTerminalService(nil, projectConversationService)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		fixture.catalog,
		nil,
		WithProjectConversationService(projectConversationService),
		WithConversationTerminalService(terminalService),
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	createReq, err := http.NewRequest(
		http.MethodPost,
		testServer.URL+"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/terminal-sessions",
		strings.NewReader(`{"mode":"shell","repo_path":"backend","cwd_path":"src"}`),
	)
	if err != nil {
		t.Fatalf("new create request: %v", err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create terminal session request: %v", err)
	}
	defer func() { _ = createResp.Body.Close() }()
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("expected 201, got %d: %s", createResp.StatusCode, string(body))
	}
	var createPayload struct {
		TerminalSession struct {
			ID          string `json:"id"`
			CWD         string `json:"cwd"`
			WSPath      string `json:"ws_path"`
			AttachToken string `json:"attach_token"`
		} `json:"terminal_session"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	wantCWD := filepath.Join(fixture.repoPath, "src")
	if createPayload.TerminalSession.CWD != wantCWD {
		t.Fatalf("cwd = %q, want %q", createPayload.TerminalSession.CWD, wantCWD)
	}

	wsURL := projectConversationWebsocketURL(
		testServer.URL + createPayload.TerminalSession.WSPath + "?attach_token=" + url.QueryEscape(createPayload.TerminalSession.AttachToken),
	)
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		body := ""
		if response != nil && response.Body != nil {
			raw, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			body = string(raw)
		}
		t.Fatalf("dial websocket %s: %v body=%s", wsURL, err, body)
	}
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	defer func() { _ = conn.Close() }()

	if frame := readConversationTerminalWebsocketFrame(t, conn); frame["type"] != "ready" {
		t.Fatalf("ready frame = %+v", frame)
	}
	if err := conn.WriteJSON(map[string]any{"type": "resize", "cols": 100, "rows": 40}); err != nil {
		t.Fatalf("write resize frame: %v", err)
	}
	if err := conn.WriteJSON(map[string]any{"type": "input", "data": "cHJpbnRmICdXU19PS1xuJwo="}); err != nil {
		t.Fatalf("write input frame: %v", err)
	}

	output := collectConversationTerminalOutput(t, conn, func(text string) bool {
		return strings.Contains(text, "WS_OK")
	})
	if !strings.Contains(output, "WS_OK") {
		t.Fatalf("unexpected terminal output %q", output)
	}

	if err := conn.WriteJSON(map[string]any{"type": "close"}); err != nil {
		t.Fatalf("write close frame: %v", err)
	}
	exit := awaitConversationTerminalExitFrame(t, conn)
	if exit["type"] != "exit" {
		t.Fatalf("exit frame = %+v", exit)
	}
	sessionID := uuid.MustParse(createPayload.TerminalSession.ID)
	awaitConversationTerminalHTTPCleanup(t, terminalService, fixture.conversation.ID, sessionID)
}

func TestProjectConversationTerminalAttachWebsocketReconnectsExistingSession(t *testing.T) {
	client := openTestEntClient(t)
	fixture := setupProjectConversationTerminalRouteFixture(t, client)

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		chatrepo.NewEntRepository(client),
		fixture.catalog,
		nil,
		nil,
		nil,
		nil,
	)
	terminalService := chatservice.NewConversationTerminalService(nil, projectConversationService)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		fixture.catalog,
		nil,
		WithProjectConversationService(projectConversationService),
		WithConversationTerminalService(terminalService),
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	createReq, err := http.NewRequest(
		http.MethodPost,
		testServer.URL+"/api/v1/chat/conversations/"+fixture.conversation.ID.String()+"/terminal-sessions",
		strings.NewReader(`{"mode":"shell","repo_path":"backend","cwd_path":"src"}`),
	)
	if err != nil {
		t.Fatalf("new create request: %v", err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create terminal session request: %v", err)
	}
	defer func() { _ = createResp.Body.Close() }()
	if createResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("expected 201, got %d: %s", createResp.StatusCode, string(body))
	}
	var createPayload struct {
		TerminalSession struct {
			ID          string `json:"id"`
			WSPath      string `json:"ws_path"`
			AttachToken string `json:"attach_token"`
		} `json:"terminal_session"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createPayload); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	wsURL := projectConversationWebsocketURL(
		testServer.URL + createPayload.TerminalSession.WSPath + "?attach_token=" + url.QueryEscape(createPayload.TerminalSession.AttachToken),
	)
	firstConn, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		body := ""
		if response != nil && response.Body != nil {
			raw, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			body = string(raw)
		}
		t.Fatalf("dial first websocket %s: %v body=%s", wsURL, err, body)
	}
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}

	if frame := readConversationTerminalWebsocketFrame(t, firstConn); frame["type"] != "ready" {
		t.Fatalf("first ready frame = %+v", frame)
	}
	if err := firstConn.WriteJSON(map[string]any{
		"type": "input",
		"data": base64.StdEncoding.EncodeToString([]byte("export ASE_REATTACH_MARKER=reattached\n")),
	}); err != nil {
		t.Fatalf("write first input frame: %v", err)
	}
	if err := firstConn.Close(); err != nil {
		t.Fatalf("close first websocket: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	secondConn, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		body := ""
		if response != nil && response.Body != nil {
			raw, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			body = string(raw)
		}
		t.Fatalf("dial second websocket %s: %v body=%s", wsURL, err, body)
	}
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	defer func() { _ = secondConn.Close() }()

	if frame := readConversationTerminalWebsocketFrame(t, secondConn); frame["type"] != "ready" {
		t.Fatalf("second ready frame = %+v", frame)
	}
	if err := secondConn.WriteJSON(map[string]any{
		"type": "input",
		"data": base64.StdEncoding.EncodeToString([]byte("printf '%s\\n' \"$ASE_REATTACH_MARKER\"\n")),
	}); err != nil {
		t.Fatalf("write second input frame: %v", err)
	}
	output := collectConversationTerminalOutput(t, secondConn, func(text string) bool {
		return strings.Contains(text, "reattached")
	})
	if !strings.Contains(output, "reattached") {
		t.Fatalf("unexpected reconnect output %q", output)
	}

	if err := secondConn.WriteJSON(map[string]any{"type": "close"}); err != nil {
		t.Fatalf("write close frame: %v", err)
	}
	exit := awaitConversationTerminalExitFrame(t, secondConn)
	if exit["type"] != "exit" {
		t.Fatalf("exit frame = %+v", exit)
	}
	sessionID := uuid.MustParse(createPayload.TerminalSession.ID)
	awaitConversationTerminalHTTPCleanup(t, terminalService, fixture.conversation.ID, sessionID)
}

func TestProjectConversationTerminalAttachRejectsInvalidToken(t *testing.T) {
	client := openTestEntClient(t)
	fixture := setupProjectConversationTerminalRouteFixture(t, client)

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		chatrepo.NewEntRepository(client),
		fixture.catalog,
		nil,
		nil,
		nil,
		nil,
	)
	terminalService := chatservice.NewConversationTerminalService(nil, projectConversationService)
	input, err := chatdomain.ParseOpenTerminalSessionInput(chatdomain.OpenTerminalSessionRawInput{Mode: "shell"})
	if err != nil {
		t.Fatalf("ParseOpenTerminalSessionInput() error = %v", err)
	}
	session, err := terminalService.CreateSession(context.Background(), chatservice.LocalProjectConversationUserID, fixture.conversation.ID, input)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		fixture.catalog,
		nil,
		WithProjectConversationService(projectConversationService),
		WithConversationTerminalService(terminalService),
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	wsURL := projectConversationWebsocketURL(testServer.URL + session.WSPath + "?attach_token=wrong-token")
	_, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected invalid attach token dial to fail")
	}
	if response != nil && response.Body != nil {
		defer func() { _ = response.Body.Close() }()
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for invalid token, got response=%v err=%v", response, err)
	}
}

func TestProjectConversationMuxStreamRouteStreamsPeriodicKeepalives(t *testing.T) {
	originalInterval := chatSSEKeepaliveInterval
	chatSSEKeepaliveInterval = 5 * time.Millisecond
	defer func() {
		chatSSEKeepaliveInterval = originalInterval
	}()

	client := openTestEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-mux-keepalive").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Mux Keepalive").
		SetSlug("openase-mux-keepalive").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	principal := testBrowserSessionAIPrincipal()

	repoStore := chatrepo.NewEntRepository(client)
	if _, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     principal.String(),
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	}); err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repoStore,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		config.ServerConfig{Port: 40023, WriteTimeout: time.Second},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	// Keep the request open long enough for the SSE handshake and repeated keepalives under load.
	streamCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		streamCtx,
		http.MethodGet,
		testServer.URL+"/api/v1/chat/projects/"+project.ID.String()+"/conversations/stream",
		nil,
	)
	if err != nil {
		t.Fatalf("new mux stream request: %v", err)
	}
	addAIPrincipalCookie(req, principal)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open mux stream: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	bodyCh := make(chan []byte, 1)
	errCh := make(chan error, 1)
	go func() {
		var body bytes.Buffer
		keepalives := 0
		for keepalives < 2 {
			line, err := reader.ReadString('\n')
			if err != nil {
				errCh <- err
				return
			}
			body.WriteString(line)
			if line == ": keepalive\n" {
				separator, err := reader.ReadString('\n')
				if err != nil {
					errCh <- err
					return
				}
				body.WriteString(separator)
				if separator == "\n" {
					keepalives++
				}
			}
		}
		bodyCh <- body.Bytes()
	}()

	select {
	case body := <-bodyCh:
		if got := strings.Count(string(body), ": keepalive\n\n"); got < 2 {
			t.Fatalf("expected at least two keepalive comments, got %d in %q", got, string(body))
		}
	case err := <-errCh:
		t.Fatalf("read mux stream body: %v", err)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for mux stream keepalives")
	}
}

func TestProjectConversationDeleteRouteDeletesConversationWhenAuthDisabled(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-delete-route").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	workspaceRoot := t.TempDir()
	machineItem, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName(catalogdomain.LocalMachineName).
		SetHost(catalogdomain.LocalMachineHost).
		SetPort(22).
		SetWorkspaceRoot(workspaceRoot).
		SetDescription("Local delete host.").
		SetStatus("online").
		SetResources(map[string]any{"transport": "local"}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	projectItem, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Delete Route").
		SetSlug("openase-delete-route").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machineItem.ID).
		SetName("Gemini").
		SetAdapterType("gemini-cli").
		SetCliCommand("gemini").
		SetModelName("gemini-2.5-pro").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	repoStore := chatrepo.NewEntRepository(client)
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  projectItem.ID,
		UserID:     chatservice.LocalProjectConversationUserID.String(),
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerItem.ID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repoStore,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	rec := performJSONRequest(
		t,
		server,
		http.MethodDelete,
		"/api/v1/chat/conversations/"+conversation.ID.String(),
		"",
	)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := repoStore.GetConversation(ctx, conversation.ID); !errors.Is(err, chatrepo.ErrNotFound) {
		t.Fatalf("expected deleted conversation, got %v", err)
	}
}

func TestProjectConversationListRouteUsesStableLocalPrincipalWhenAuthDisabled(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-local-principal").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Local Principal").
		SetSlug("openase-local-principal").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	repoStore := chatrepo.NewEntRepository(client)
	_, err = repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "browser-user-a",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	_, err = repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "browser-user-b",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create second conversation: %v", err)
	}

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repoStore,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	rec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/chat/conversations?project_id="+project.ID.String(),
		"",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := strings.Count(rec.Body.String(), `"user_id":"`+chatservice.LocalProjectConversationUserID.String()+`"`); got != 2 {
		t.Fatalf("expected both conversations to normalize to the stable local principal, got body %s", rec.Body.String())
	}
}

func TestProjectConversationStreamRouteUsesStableLocalPrincipalWhenAuthDisabled(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-local-stream").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Local Stream").
		SetSlug("openase-local-stream").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	repoStore := chatrepo.NewEntRepository(client)
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "browser-user-a",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repoStore,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		config.ServerConfig{Port: 40023, WriteTimeout: time.Second},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	streamCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		streamCtx,
		http.MethodGet,
		testServer.URL+"/api/v1/chat/conversations/"+conversation.ID.String()+"/stream",
		nil,
	)
	if err != nil {
		t.Fatalf("new stream request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	frame := readProjectConversationSSEFrame(t, bufio.NewReader(resp.Body))
	if frame.Event != "session" || !strings.Contains(frame.Data, conversation.ID.String()) {
		t.Fatalf("expected initial session frame for the legacy conversation, got %+v", frame)
	}
}

type projectConversationTerminalRouteFixture struct {
	catalog      catalogservice.Service
	conversation chatdomain.Conversation
	repoPath     string
}

func setupProjectConversationTerminalRouteFixture(t *testing.T, client *ent.Client) projectConversationTerminalRouteFixture {
	t.Helper()

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-terminal").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	workspaceRoot := t.TempDir()
	machineItem, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName(catalogdomain.LocalMachineName).
		SetHost(catalogdomain.LocalMachineHost).
		SetPort(22).
		SetWorkspaceRoot(workspaceRoot).
		SetDescription("Local terminal host.").
		SetStatus("online").
		SetResources(map[string]any{"transport": "local"}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	projectItem, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Terminal").
		SetSlug("openase-terminal").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machineItem.ID).
		SetName("Codex").
		SetAdapterType("codex-app-server").
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if _, err := client.ProjectRepo.Create().
		SetProjectID(projectItem.ID).
		SetName("backend").
		SetRepositoryURL("file:///tmp/backend.git").
		SetDefaultBranch("main").
		Save(ctx); err != nil {
		t.Fatalf("create project repo: %v", err)
	}
	repoStore := chatrepo.NewEntRepository(client)
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  projectItem.ID,
		UserID:     chatservice.LocalProjectConversationUserID.String(),
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: providerItem.ID,
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	workspacePath, err := workspaceinfra.TicketWorkspacePath(
		workspaceRoot,
		org.ID.String(),
		projectItem.Slug,
		"conv-"+conversation.ID.String(),
	)
	if err != nil {
		t.Fatalf("workspace path: %v", err)
	}
	repoPath := workspaceinfra.RepoPath(workspacePath, "", "backend")
	if err := os.MkdirAll(filepath.Join(repoPath, "src"), 0o750); err != nil {
		t.Fatalf("mkdir repo path: %v", err)
	}
	repository, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("git init repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "src", "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("write repo seed file: %v", err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("load repo worktree: %v", err)
	}
	if _, err := worktree.Add("src/main.go"); err != nil {
		t.Fatalf("git add repo seed file: %v", err)
	}
	if _, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@openai.com",
			When:  time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC),
		},
	}); err != nil {
		t.Fatalf("git commit repo seed file: %v", err)
	}

	return projectConversationTerminalRouteFixture{
		catalog:      catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		conversation: conversation,
		repoPath:     repoPath,
	}
}

func runConversationGitCommand(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...) // #nosec G204 -- test helper executes fixed commands.
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, string(output))
	}
	return string(output)
}

func readConversationTerminalWebsocketFrame(t *testing.T, conn *websocket.Conn) map[string]string {
	t.Helper()
	frame, err := readConversationTerminalWebsocketFrameWithTimeout(conn, 30*time.Second)
	if err != nil {
		t.Fatalf("read websocket frame: %v", err)
	}
	return frame
}

func readConversationTerminalWebsocketFrameWithTimeout(
	conn *websocket.Conn,
	timeout time.Duration,
) (map[string]string, error) {
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("set read deadline: %w", err)
	}
	var payload map[string]any
	if err := conn.ReadJSON(&payload); err != nil {
		return nil, err
	}
	frame := map[string]string{}
	for key, value := range payload {
		frame[key] = fmt.Sprint(value)
	}
	return frame, nil
}

func collectConversationTerminalOutput(
	t *testing.T,
	conn *websocket.Conn,
	done func(string) bool,
) string {
	t.Helper()
	var builder strings.Builder
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		frame, err := readConversationTerminalWebsocketFrameWithTimeout(conn, time.Until(deadline))
		if err != nil {
			t.Fatalf("read websocket frame: %v", err)
		}
		switch frame["type"] {
		case "output":
			decoded, err := base64.StdEncoding.DecodeString(frame["data"])
			if err != nil {
				t.Fatalf("decode terminal output: %v", err)
			}
			builder.Write(decoded)
			if done(builder.String()) {
				return builder.String()
			}
		case "error":
			t.Fatalf("unexpected terminal error frame: %+v", frame)
		}
	}
	t.Fatalf("timed out collecting terminal output: %q", builder.String())
	return ""
}

func awaitConversationTerminalExitFrame(t *testing.T, conn *websocket.Conn) map[string]string {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		frame, err := readConversationTerminalWebsocketFrameWithTimeout(conn, time.Until(deadline))
		if err != nil {
			t.Fatalf("read websocket frame: %v", err)
		}
		switch frame["type"] {
		case "exit":
			return frame
		case "output":
			continue
		case "error":
			t.Fatalf("unexpected terminal error frame: %+v", frame)
		default:
			t.Fatalf("unexpected terminal frame while waiting for exit: %+v", frame)
		}
	}
	t.Fatal("timed out waiting for terminal exit frame")
	return nil
}

func awaitConversationTerminalHTTPCleanup(
	t *testing.T,
	service *chatservice.ConversationTerminalService,
	conversationID uuid.UUID,
	sessionID uuid.UUID,
) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := service.CloseSession(conversationID, sessionID); errors.Is(err, chatservice.ErrConversationTerminalSessionNotFound) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("expected terminal session cleanup")
}

func projectConversationWebsocketURL(raw string) string {
	switch {
	case strings.HasPrefix(raw, "https://"):
		return "wss://" + strings.TrimPrefix(raw, "https://")
	case strings.HasPrefix(raw, "http://"):
		return "ws://" + strings.TrimPrefix(raw, "http://")
	default:
		return raw
	}
}

func TestChatRouteStreamsPeriodicKeepalives(t *testing.T) {
	originalInterval := chatSSEKeepaliveInterval
	chatSSEKeepaliveInterval = 5 * time.Millisecond
	defer func() {
		chatSSEKeepaliveInterval = originalInterval
	}()

	projectID := uuid.New()
	providerID := uuid.New()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	chatSvc := chatservice.NewService(
		logger,
		streamingChatRuntimeStub{
			stream: func() <-chan chatservice.StreamEvent {
				events := make(chan chatservice.StreamEvent)
				go func() {
					time.Sleep(12 * time.Millisecond)
					events <- chatservice.StreamEvent{
						Event: "done",
						Payload: map[string]any{
							"session_id": "session-keepalive-1",
							"turns_used": 1,
						},
					}
					close(events)
				}()
				return events
			},
		},
		chatCatalogStub{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: uuid.New(),
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:          providerID,
					Name:        "Codex",
					AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
					Available:   true,
				},
			},
		},
		chatTicketStub{},
		chatWorkflowStub{},
		nil,
		"",
	)

	server := NewServer(
		config.ServerConfig{Port: 40023, WriteTimeout: time.Second},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithChatService(chatSvc),
	)

	body := mustMarshalJSON(t, map[string]any{
		"message": "keep streaming",
		"source":  "project_sidebar",
		"context": map[string]any{
			"project_id": projectID.String(),
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	addAIPrincipalCookie(req, testBrowserSessionAIPrincipal())
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := strings.Count(rec.Body.String(), ": keepalive\n\n"); got < 2 {
		t.Fatalf("expected at least two keepalive comments, got %d in %q", got, rec.Body.String())
	}
}

func TestChatRouteLogsUnexpectedStreamTermination(t *testing.T) {
	projectID := uuid.New()
	providerID := uuid.New()
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
	chatSvc := chatservice.NewService(
		logger,
		streamingChatRuntimeStub{
			stream: func() <-chan chatservice.StreamEvent {
				events := make(chan chatservice.StreamEvent, 1)
				events <- chatservice.StreamEvent{
					Event: "message",
					Payload: map[string]any{
						"type":    "text",
						"content": "partial reply",
					},
				}
				close(events)
				return events
			},
		},
		chatCatalogStub{
			project: catalogdomain.Project{
				ID:             projectID,
				OrganizationID: uuid.New(),
				Name:           "OpenASE",
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:          providerID,
					Name:        "Codex",
					AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
					Available:   true,
				},
			},
		},
		chatTicketStub{},
		chatWorkflowStub{},
		nil,
		"",
	)

	server := NewServer(
		config.ServerConfig{Port: 40023, WriteTimeout: time.Second},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithChatService(chatSvc),
	)

	body := mustMarshalJSON(t, map[string]any{
		"message":     "edit this harness",
		"source":      "project_sidebar",
		"provider_id": providerID.String(),
		"context": map[string]any{
			"project_id": projectID.String(),
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	principal := testBrowserSessionAIPrincipal()
	addAIPrincipalCookie(req, principal)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	logOutput := logBuffer.String()
	for _, want := range []string{
		"chat stream terminated before completion",
		"chat_source=project_sidebar",
		"chat_project_id=" + projectID.String(),
		"chat_provider_id=" + providerID.String(),
		"chat_user_id=" + principal.String(),
		"last_event=message",
		"terminal_event_seen=false",
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("expected log output to contain %q, got %q", want, logOutput)
		}
	}
}

type fakeClaudeAdapter struct {
	session  *fakeClaudeSession
	lastSpec provider.ClaudeCodeSessionSpec
}

type streamingChatRuntimeStub struct {
	stream func() <-chan chatservice.StreamEvent
}

type chatRuntimeStub struct {
	startErr error
}

func (s streamingChatRuntimeStub) Supports(catalogdomain.AgentProvider) bool {
	return true
}

func (s streamingChatRuntimeStub) StartTurn(
	context.Context,
	chatservice.RuntimeTurnInput,
) (chatservice.TurnStream, error) {
	return chatservice.TurnStream{Events: s.stream()}, nil
}

func (s streamingChatRuntimeStub) CloseSession(chatservice.SessionID) bool {
	return false
}

func (s chatRuntimeStub) Supports(catalogdomain.AgentProvider) bool {
	return true
}

func (s chatRuntimeStub) StartTurn(context.Context, chatservice.RuntimeTurnInput) (chatservice.TurnStream, error) {
	return chatservice.TurnStream{}, s.startErr
}

func (s chatRuntimeStub) CloseSession(chatservice.SessionID) bool {
	return false
}

type chatCatalogStub struct {
	project   catalogdomain.Project
	providers []catalogdomain.AgentProvider
}

func (s chatCatalogStub) GetProject(context.Context, uuid.UUID) (catalogdomain.Project, error) {
	return s.project, nil
}

func (s chatCatalogStub) ListActivityEvents(context.Context, catalogdomain.ListActivityEvents) (catalogdomain.ActivityEventPage, error) {
	return catalogdomain.ActivityEventPage{}, nil
}

func (s chatCatalogStub) ListProjectRepos(context.Context, uuid.UUID) ([]catalogdomain.ProjectRepo, error) {
	return nil, nil
}

func (s chatCatalogStub) ListTicketRepoScopes(context.Context, uuid.UUID, uuid.UUID) ([]catalogdomain.TicketRepoScope, error) {
	return nil, nil
}

func (s chatCatalogStub) ListAgentProviders(context.Context, uuid.UUID) ([]catalogdomain.AgentProvider, error) {
	return s.providers, nil
}

func (s chatCatalogStub) GetAgentProvider(context.Context, uuid.UUID) (catalogdomain.AgentProvider, error) {
	if len(s.providers) == 0 {
		return catalogdomain.AgentProvider{}, errors.New("provider not found")
	}
	return s.providers[0], nil
}

type chatTicketStub struct{}

func (chatTicketStub) Get(context.Context, uuid.UUID) (ticketservice.Ticket, error) {
	return ticketservice.Ticket{}, errors.New("not implemented")
}

func (chatTicketStub) List(context.Context, ticketservice.ListInput) ([]ticketservice.Ticket, error) {
	return nil, nil
}

type chatWorkflowStub struct{}

func (chatWorkflowStub) Get(context.Context, uuid.UUID) (workflowservice.WorkflowDetail, error) {
	return workflowservice.WorkflowDetail{}, errors.New("not implemented")
}

func (chatWorkflowStub) List(context.Context, uuid.UUID) ([]workflowservice.Workflow, error) {
	return nil, nil
}

func (chatWorkflowStub) GetSkill(context.Context, uuid.UUID) (workflowservice.SkillDetail, error) {
	return workflowservice.SkillDetail{}, errors.New("not implemented")
}

func (f *fakeClaudeAdapter) Start(_ context.Context, spec provider.ClaudeCodeSessionSpec) (provider.ClaudeCodeSession, error) {
	f.lastSpec = spec
	return f.session, nil
}

type fakeClaudeSession struct {
	sessionID provider.ClaudeCodeSessionID
	events    chan provider.ClaudeCodeEvent
	errors    chan error
	sent      []provider.ClaudeCodeTurnInput
}

func newFakeClaudeSession(events []provider.ClaudeCodeEvent, errs []error) *fakeClaudeSession {
	session := &fakeClaudeSession{
		events: make(chan provider.ClaudeCodeEvent, len(events)),
		errors: make(chan error, len(errs)),
	}
	for _, event := range events {
		session.events <- event
		if strings.TrimSpace(event.SessionID) != "" && session.sessionID == "" {
			session.sessionID = provider.MustParseClaudeCodeSessionID(event.SessionID)
		}
	}
	for _, err := range errs {
		session.errors <- err
	}
	close(session.events)
	close(session.errors)
	return session
}

func (s *fakeClaudeSession) SessionID() (provider.ClaudeCodeSessionID, bool) {
	if s.sessionID == "" {
		return "", false
	}
	return s.sessionID, true
}

func (s *fakeClaudeSession) Events() <-chan provider.ClaudeCodeEvent { return s.events }

func (s *fakeClaudeSession) Errors() <-chan error { return s.errors }

func (s *fakeClaudeSession) Send(_ context.Context, input provider.ClaudeCodeTurnInput) error {
	s.sent = append(s.sent, input)
	return nil
}

func (s *fakeClaudeSession) Close(_ context.Context) error { return nil }

type staticWorkflowReader struct{}

func (staticWorkflowReader) Get(context.Context, uuid.UUID) (workflowservice.WorkflowDetail, error) {
	return workflowservice.WorkflowDetail{}, workflowservice.ErrWorkflowNotFound
}

func (staticWorkflowReader) List(context.Context, uuid.UUID) ([]workflowservice.Workflow, error) {
	return nil, nil
}

func (staticWorkflowReader) GetSkill(context.Context, uuid.UUID) (workflowservice.SkillDetail, error) {
	return workflowservice.SkillDetail{}, workflowservice.ErrSkillNotFound
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return body
}

type projectConversationSSEFrame struct {
	Event string
	Data  string
}

func readProjectConversationSSEFrame(t *testing.T, reader *bufio.Reader) projectConversationSSEFrame {
	t.Helper()

	for {
		frame := projectConversationSSEFrame{}
		dataLines := make([]string, 0, 1)

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("read sse line: %v", err)
			}
			line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
			if line == "" {
				break
			}
			if strings.HasPrefix(line, ":") {
				continue
			}
			switch {
			case strings.HasPrefix(line, "event:"):
				frame.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}

		if frame.Event == "" && len(dataLines) == 0 {
			continue
		}
		frame.Data = strings.Join(dataLines, "\n")
		return frame
	}
}

func testTaskNotificationPayload(marker string) map[string]any {
	return map[string]any{
		"type": "task_notification",
		"raw": map[string]any{
			"marker": marker,
		},
	}
}

func testFloatPointer(value float64) *float64 {
	return &value
}

func slicesContain(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
