package httpapi

import (
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

	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

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
		SetResources(map[string]any{"transport": "local", "last_success": true}).
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

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
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

	ticketItem, err := ticketservice.NewService(client).Create(ctx, ticketservice.CreateInput{
		ProjectID:   project.ID,
		Title:       "Implement ephemeral chat",
		Description: "Explain why the last hook failed and propose smaller follow-up tickets.",
		Priority:    "medium",
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
		SetEventType("agent.output").
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
	if _, err := catalogSvc.CreateAgentProvider(ctx, providerInput); err != nil {
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
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
		nil,
		catalogSvc,
		nil,
		WithChatService(chatservice.NewService(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			adapter,
			catalogSvc,
			ticketservice.NewService(client),
			staticWorkflowReader{},
			"",
		)),
	)

	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	requestBody := mustMarshalJSON(t, map[string]any{
		"message":    "为什么失败了？",
		"source":     "ticket_detail",
		"session_id": "sess-prev",
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
	if !strings.Contains(textBody, "The ticket failed because") {
		t.Fatalf("expected assistant message in stream, got %q", textBody)
	}
	if !strings.Contains(textBody, "event: done\n") || !strings.Contains(textBody, "\"session_id\":\"sess-ephemeral-1\"") {
		t.Fatalf("expected done event with session id, got %q", textBody)
	}

	if adapter.lastSpec.ResumeSessionID == nil || adapter.lastSpec.ResumeSessionID.String() != "sess-prev" {
		t.Fatalf("expected resume session id sess-prev, got %+v", adapter.lastSpec.ResumeSessionID)
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
	if !strings.Contains(adapter.lastSpec.AppendSystemPrompt, "\"type\":\"action_proposal\"") {
		t.Fatalf("expected action proposal instructions in system prompt, got %q", adapter.lastSpec.AppendSystemPrompt)
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
		{name: "provider", err: chatservice.ErrProviderNotFound, wantStatus: http.StatusConflict, wantCode: "CHAT_PROVIDER_NOT_CONFIGURED"},
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

type fakeClaudeAdapter struct {
	session  *fakeClaudeSession
	lastSpec provider.ClaudeCodeSessionSpec
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

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return body
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
