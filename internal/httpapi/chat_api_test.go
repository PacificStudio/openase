package httpapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestCurrentRequestChatUserIDUsesHumanPrincipalInOIDCMode(t *testing.T) {
	server := &Server{auth: config.AuthConfig{Mode: config.AuthModeOIDC}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(chatUserHeader, "browser-user-1")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	userID := uuid.MustParse("8db7261e-e16d-458e-8926-cd01550686a5")
	setHumanPrincipal(ctx, humanauthdomain.AuthenticatedPrincipal{
		User: humanauthdomain.User{ID: userID},
	})

	got, err := server.currentRequestChatUserID(ctx)
	if err != nil {
		t.Fatalf("currentRequestChatUserID() error = %v", err)
	}
	if got != chatservice.UserID("user:"+userID.String()) {
		t.Fatalf("currentRequestChatUserID() = %q, want %q", got, "user:"+userID.String())
	}
}

func TestCurrentRequestChatUserIDRejectsHeaderFallbackInOIDCMode(t *testing.T) {
	server := &Server{auth: config.AuthConfig{Mode: config.AuthModeOIDC}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(chatUserHeader, "browser-user-1")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	_, err := server.currentRequestChatUserID(ctx)
	if !errors.Is(err, humanauthservice.ErrUnauthorized) {
		t.Fatalf("currentRequestChatUserID() error = %v, want %v", err, humanauthservice.ErrUnauthorized)
	}
}

func TestCurrentRequestChatUserIDAllowsHeaderFallbackWhenAuthDisabled(t *testing.T) {
	server := &Server{auth: config.AuthConfig{Mode: config.AuthModeDisabled}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(chatUserHeader, "browser-user-1")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	got, err := server.currentRequestChatUserID(ctx)
	if err != nil {
		t.Fatalf("currentRequestChatUserID() error = %v", err)
	}
	if got != "browser-user-1" {
		t.Fatalf("currentRequestChatUserID() = %q, want %q", got, "browser-user-1")
	}
}

func TestPrepareProjectConversationActionBodyInjectsExplicitAuditActor(t *testing.T) {
	conversationID := uuid.MustParse("57cdcb4e-6e4c-4474-839a-4daa5abdd8d2")
	executedBy := projectConversationConfirmedActionActor(chatservice.UserID("user:browser-user"), conversationID)

	tests := []struct {
		name      string
		method    string
		path      string
		body      map[string]any
		fieldName string
	}{
		{
			name:      "ticket create",
			method:    http.MethodPost,
			path:      "/api/v1/projects/" + uuid.NewString() + "/tickets",
			body:      map[string]any{"title": "Follow up"},
			fieldName: "created_by",
		},
		{
			name:      "ticket comment patch",
			method:    http.MethodPatch,
			path:      "/api/v1/tickets/" + uuid.NewString() + "/comments/" + uuid.NewString(),
			body:      map[string]any{"body": "Updated after confirmation"},
			fieldName: "edited_by",
		},
		{
			name:      "workflow harness update",
			method:    http.MethodPut,
			path:      "/api/v1/workflows/" + uuid.NewString() + "/harness",
			body:      map[string]any{"content": "new harness"},
			fieldName: "edited_by",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prepared, err := prepareProjectConversationActionBody(tc.method, tc.path, tc.body, executedBy)
			if err != nil {
				t.Fatalf("prepareProjectConversationActionBody() error = %v", err)
			}
			if got := prepared[tc.fieldName]; got != executedBy {
				t.Fatalf("prepareProjectConversationActionBody() %s = %#v, want %q", tc.fieldName, got, executedBy)
			}
		})
	}
}

func TestPrepareProjectConversationActionBodyRejectsImplicitAuditRoutes(t *testing.T) {
	executedBy := projectConversationConfirmedActionActor(chatservice.UserID("user:browser-user"), uuid.New())

	_, err := prepareProjectConversationActionBody(
		http.MethodPost,
		"/api/v1/projects/"+uuid.NewString()+"/repos",
		map[string]any{"name": "repo"},
		executedBy,
	)
	if err == nil {
		t.Fatal("expected unsupported path to be rejected")
	}
	if !strings.Contains(err.Error(), "audit actor is not explicit") {
		t.Fatalf("unexpected error = %v", err)
	}
}

func TestProjectConversationRoutesRequireHumanPrincipalInOIDCMode(t *testing.T) {
	projectConversationService := chatservice.NewProjectConversationService(nil, nil, nil, nil, nil, nil, nil)
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := performJSONRequest(t, server, tc.method, tc.target, tc.body)
			if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "HUMAN_SESSION_REQUIRED") {
				t.Fatalf("expected 401 HUMAN_SESSION_REQUIRED, got %d: %s", rec.Code, rec.Body.String())
			}
		})
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
		"message":     "为什么失败了？",
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
	if !strings.Contains(adapter.lastSpec.AppendSystemPrompt, "不要输出 `action_proposal` 或 `platform_command_proposal`") {
		t.Fatalf("expected direct-execution instructions in system prompt, got %q", adapter.lastSpec.AppendSystemPrompt)
	}
	if !slicesContain(adapter.lastSpec.Environment, "ANTHROPIC_API_KEY=test-key") {
		t.Fatalf("expected auth config env injection, got %v", adapter.lastSpec.Environment)
	}
	if len(adapter.session.sent) != 1 || adapter.session.sent[0].Prompt != "为什么失败了？" {
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
		"message":     "帮我总结一下项目状态",
		"source":      "project_sidebar",
		"provider_id": providerID.String(),
		"context": map[string]any{
			"project_id": projectID.String(),
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(chatUserHeader, "browser-user-1")
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
		"chat_user_id=browser-user-1",
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
		{name: "generic conflict", err: chatservice.ErrConversationConflict, wantStatus: http.StatusConflict, wantCode: "CHAT_CONVERSATION_CONFLICT"},
		{name: "missing conversation", err: chatservice.ErrConversationNotFound, wantStatus: http.StatusNotFound, wantCode: "CHAT_CONVERSATION_NOT_FOUND"},
		{name: "runtime missing", err: chatservice.ErrConversationRuntimeAbsent, wantStatus: http.StatusConflict, wantCode: "CHAT_CONVERSATION_RUNTIME_UNAVAILABLE"},
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

	repoStore := chatrepo.NewEntRepository(client)
	firstConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	secondConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
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

	if _, err := projectConversationService.AppendActionExecutionResult(
		ctx,
		chatservice.UserID("user:conversation"),
		firstConversation.ID,
		nil,
		map[string]any{"marker": "conversation-1"},
	); err != nil {
		t.Fatalf("append first action result: %v", err)
	}

	firstMessage := readProjectConversationSSEFrame(t, firstReader)
	if firstMessage.Event != "message" ||
		!strings.Contains(firstMessage.Data, "\"type\":\"action_result\"") ||
		!strings.Contains(firstMessage.Data, "\"marker\":\"conversation-1\"") {
		t.Fatalf("first message frame = %+v", firstMessage)
	}

	if _, err := projectConversationService.AppendActionExecutionResult(
		ctx,
		chatservice.UserID("user:conversation"),
		secondConversation.ID,
		nil,
		map[string]any{"marker": "conversation-2"},
	); err != nil {
		t.Fatalf("append second action result: %v", err)
	}

	secondMessage := readProjectConversationSSEFrame(t, secondReader)
	if secondMessage.Event != "message" ||
		!strings.Contains(secondMessage.Data, "\"type\":\"action_result\"") ||
		!strings.Contains(secondMessage.Data, "\"marker\":\"conversation-2\"") {
		t.Fatalf("second message frame = %+v", secondMessage)
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

	repoStore := chatrepo.NewEntRepository(client)
	firstConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create first conversation: %v", err)
	}
	secondConversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:conversation",
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
	req.Header.Set(chatUserHeader, "user:conversation")

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

	if _, err := projectConversationService.AppendActionExecutionResult(
		ctx,
		chatservice.UserID("user:conversation"),
		firstConversation.ID,
		nil,
		map[string]any{"marker": "conversation-1"},
	); err != nil {
		t.Fatalf("append first action result: %v", err)
	}
	firstMessage := readProjectConversationSSEFrame(t, reader)
	if firstMessage.Event != "message" ||
		!strings.Contains(firstMessage.Data, "\"conversation_id\":\""+firstConversation.ID.String()+"\"") ||
		!strings.Contains(firstMessage.Data, "\"marker\":\"conversation-1\"") {
		t.Fatalf("first mux message frame = %+v", firstMessage)
	}

	if _, err := projectConversationService.AppendActionExecutionResult(
		ctx,
		chatservice.UserID("user:conversation"),
		secondConversation.ID,
		nil,
		map[string]any{"marker": "conversation-2"},
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
	req.Header.Set(chatUserHeader, "browser-user-keepalive")
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
	req.Header.Set(chatUserHeader, "browser-user-stream")
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
		"chat_user_id=browser-user-stream",
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

func (s chatCatalogStub) ListActivityEvents(context.Context, catalogdomain.ListActivityEvents) ([]catalogdomain.ActivityEvent, error) {
	return nil, nil
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
