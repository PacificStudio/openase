package httpapi

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	"github.com/BetterAndBetterII/openase/internal/builtin"
	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	workflowrepo "github.com/BetterAndBetterII/openase/internal/repo/workflow"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestSkillRoutesErrorMappingsAndInvalidPayloads(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := NewServer(
		config.ServerConfig{Port: 40029},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		nil,
		newTicketStatusService(client),
		nil,
		nil,
		workflowSvc,
	)
	unavailableServer := NewServer(
		config.ServerConfig{Port: 40030},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		nil,
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	rec := performJSONRequest(t, unavailableServer, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/skills", uuid.New()), "")
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "SERVICE_UNAVAILABLE") {
		t.Fatalf("list skills unavailable = %d %s", rec.Code, rec.Body.String())
	}

	for _, testCase := range []struct {
		name       string
		server     *Server
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "list invalid project", server: server, method: http.MethodGet, target: "/api/v1/projects/not-a-uuid/skills", wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "refresh invalid project", server: server, method: http.MethodPost, target: "/api/v1/projects/not-a-uuid/skills/refresh", body: `{"workspace_root":"/tmp/ws","adapter_type":"claude-code-cli"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "refresh invalid payload", server: server, method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/skills/refresh", uuid.New()), body: `{"workspace_root":"   ","adapter_type":"claude-code-cli"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "bind invalid workflow id", server: server, method: http.MethodPost, target: "/api/v1/workflows/not-a-uuid/skills/bind", body: `{"skills":["commit"]}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_WORKFLOW_ID"},
		{name: "bind invalid payload", server: server, method: http.MethodPost, target: fmt.Sprintf("/api/v1/workflows/%s/skills/bind", uuid.New()), body: `{"skills":[]}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "bind missing workflow", server: server, method: http.MethodPost, target: fmt.Sprintf("/api/v1/workflows/%s/skills/bind", uuid.New()), body: `{"skills":["commit"]}`, wantStatus: http.StatusNotFound, wantBody: "WORKFLOW_NOT_FOUND"},
		{name: "unbind invalid workflow id", server: server, method: http.MethodPost, target: "/api/v1/workflows/not-a-uuid/skills/unbind", body: `{"skills":["commit"]}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_WORKFLOW_ID"},
		{name: "unbind invalid payload", server: server, method: http.MethodPost, target: fmt.Sprintf("/api/v1/workflows/%s/skills/unbind", uuid.New()), body: `{"skills":[]}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "unbind missing workflow", server: server, method: http.MethodPost, target: fmt.Sprintf("/api/v1/workflows/%s/skills/unbind", uuid.New()), body: `{"skills":["commit"]}`, wantStatus: http.StatusNotFound, wantBody: "WORKFLOW_NOT_FOUND"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequest(t, testCase.server, testCase.method, testCase.target, testCase.body)
			if rec.Code != testCase.wantStatus || !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("%s %s = %d %s, want %d containing %q", testCase.method, testCase.target, rec.Code, rec.Body.String(), testCase.wantStatus, testCase.wantBody)
			}
		})
	}
}

func TestWriteSkillRefinementError(t *testing.T) {
	for _, tc := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unavailable", err: chatservice.ErrSkillRefinementUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "SERVICE_UNAVAILABLE"},
		{name: "provider", err: chatservice.ErrProviderNotFound, wantStatus: http.StatusConflict, wantCode: "CHAT_PROVIDER_NOT_CONFIGURED"},
		{name: "provider unavailable", err: chatservice.ErrProviderUnavailable, wantStatus: http.StatusConflict, wantCode: "CHAT_PROVIDER_UNAVAILABLE"},
		{name: "provider unsupported", err: chatservice.ErrProviderUnsupported, wantStatus: http.StatusConflict, wantCode: "CHAT_PROVIDER_UNSUPPORTED"},
		{name: "skill", err: workflowservice.ErrSkillNotFound, wantStatus: http.StatusNotFound, wantCode: "CHAT_CONTEXT_NOT_FOUND"},
		{name: "catalog", err: catalogservice.ErrNotFound, wantStatus: http.StatusNotFound, wantCode: "CHAT_CONTEXT_NOT_FOUND"},
		{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			if err := writeSkillRefinementError(ctx, tc.err); err != nil {
				t.Fatalf("writeSkillRefinementError() error = %v", err)
			}
			if rec.Code != tc.wantStatus || !strings.Contains(rec.Body.String(), tc.wantCode) {
				t.Fatalf("writeSkillRefinementError(%s) = %d %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandleStartSkillRefinementRejectsHeaderFallbackInOIDCMode(t *testing.T) {
	skillID := uuid.New()
	projectID := uuid.New()
	providerID := uuid.New()

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
		WithSkillRefinementService(&chatservice.SkillRefinementService{}),
	)

	body := fmt.Sprintf(`{
		"project_id":"%s",
		"message":"Tighten the skill.",
		"provider_id":"%s",
		"files":[
			{"path":"SKILL.md","content_base64":"%s","media_type":"text/markdown; charset=utf-8","is_executable":false}
		]
	}`,
		projectID,
		providerID,
		base64.StdEncoding.EncodeToString([]byte("---\nname: deploy-openase\n---\n")),
	)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/skills/%s/refinement-runs", skillID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(chatUserHeader, "user:spoofed-header")
	rec := httptest.NewRecorder()
	ctx := server.echo.NewContext(req, rec)
	ctx.SetPath("/api/v1/skills/:skillId/refinement-runs")
	ctx.SetParamNames("skillId")
	ctx.SetParamValues(skillID.String())

	if err := server.handleStartSkillRefinement(ctx); err != nil {
		t.Fatalf("handleStartSkillRefinement() error = %v", err)
	}
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "HUMAN_SESSION_REQUIRED") {
		t.Fatalf("handleStartSkillRefinement() = %d %s", rec.Code, rec.Body.String())
	}
}

func TestSkillRoutesRefreshBindAndUnbind(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		newTicketStatusService(client),
		nil,
		nil,
		workflowSvc,
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
	createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)
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
	attachPrimaryProjectRepoCheckout(ctx, t, client, project.ID, localMachine.ID, repoRoot)

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
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
	agent, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("codex-coding").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	createdWorkflow, err := workflowSvc.Create(ctx, workflowservice.CreateInput{
		ProjectID:           project.ID,
		AgentID:             agent.ID,
		Name:                "Coding Workflow",
		Type:                "coding",
		HarnessContent:      "# Coding\n",
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
		t.Fatalf("create workflow: %v", err)
	}

	bindResp := struct {
		Harness harnessResponse `json:"harness"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/workflows/%s/skills/bind", createdWorkflow.ID),
		map[string]any{"skills": []string{"review-code", "commit"}},
		http.StatusOK,
		&bindResp,
	)
	if bindResp.Harness.Version != 2 {
		t.Fatalf("expected bind response to advance workflow version, got %+v", bindResp.Harness)
	}
	if bindResp.Harness.Content != "# Coding\n" {
		t.Fatalf("expected bind to preserve pure harness body, got %q", bindResp.Harness.Content)
	}

	listResp := struct {
		Skills []skillResponse `json:"skills"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/skills", project.ID),
		nil,
		http.StatusOK,
		&listResp,
	)
	if len(listResp.Skills) != len(builtin.Skills()) {
		t.Fatalf("expected %d skills, got %+v", len(builtin.Skills()), listResp.Skills)
	}
	reviewSkill := findSkillResponse(t, listResp.Skills, "review-code")
	if len(reviewSkill.BoundWorkflows) != 1 || reviewSkill.BoundWorkflows[0].Name != "Coding Workflow" {
		t.Fatalf("expected review-code to bind to Coding Workflow, got %+v", reviewSkill)
	}
	if reviewSkill.CurrentVersion != 1 {
		t.Fatalf("expected review-code current version to be published as v1, got %+v", reviewSkill)
	}
	if !reviewSkill.IsBuiltin {
		t.Fatalf("expected review-code to be marked as built-in, got %+v", reviewSkill)
	}
	if reviewSkill.Description == "" {
		t.Fatalf("expected review-code to expose a description, got %+v", reviewSkill)
	}
	platformSkill := findSkillResponse(t, listResp.Skills, "openase-platform")
	if !platformSkill.IsBuiltin {
		t.Fatalf("expected openase-platform to be marked as built-in, got %+v", platformSkill)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".openase", "skills", "openase-platform", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("expected built-in platform skill to stay out of repo authority paths, stat err=%v", err)
	}

	detailResp := skillDetailResponse{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/skills/%s", reviewSkill.ID),
		nil,
		http.StatusOK,
		&detailResp,
	)
	if detailResp.Skill.CurrentVersion != 1 || len(detailResp.History) != 1 || detailResp.History[0].Version != 1 {
		t.Fatalf("expected skill detail to expose current published version and history, got %+v", detailResp)
	}

	historyResp := skillHistoryResponse{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/skills/%s/history", reviewSkill.ID),
		nil,
		http.StatusOK,
		&historyResp,
	)
	if len(historyResp.History) != 1 || historyResp.History[0].Version != 1 {
		t.Fatalf("expected skill history route to expose published versions, got %+v", historyResp)
	}

	workspaceRoot := t.TempDir()
	refreshResp := skillSyncResponse{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/skills/refresh", project.ID),
		map[string]any{
			"workspace_root": workspaceRoot,
			"adapter_type":   "claude-code-cli",
		},
		http.StatusOK,
		&refreshResp,
	)
	if len(refreshResp.InjectedSkills) != len(builtin.Skills()) {
		t.Fatalf("expected %d injected skills, got %+v", len(builtin.Skills()), refreshResp)
	}
	if !containsSkillName(refreshResp.InjectedSkills, "openase-platform") {
		t.Fatalf("expected openase-platform to be injected, got %+v", refreshResp.InjectedSkills)
	}
	//nolint:gosec // test reads a file from a controlled temp workspace
	refreshedSkill, err := os.ReadFile(filepath.Join(workspaceRoot, ".claude", "skills", "review-code", "SKILL.md"))
	if err != nil {
		t.Fatalf("read refreshed skill: %v", err)
	}
	if !strings.HasPrefix(string(refreshedSkill), "---\nname: ") {
		t.Fatalf("expected refreshed skill to include frontmatter, got %q", string(refreshedSkill))
	}
	refreshedScriptPath := filepath.Join(workspaceRoot, ".claude", "skills", "openase-platform", "scripts", "upsert_workpad.sh")
	refreshedScriptInfo, err := os.Stat(refreshedScriptPath)
	if err != nil {
		t.Fatalf("expected refreshed openase-platform script: %v", err)
	}
	if refreshedScriptInfo.Mode()&0o111 == 0 {
		t.Fatalf("expected refreshed openase-platform script to be executable, mode=%v", refreshedScriptInfo.Mode())
	}
	if _, err := os.Stat(filepath.Join(workspaceRoot, ".openase", "bin", "openase")); err != nil {
		t.Fatalf("expected openase wrapper in refreshed workspace: %v", err)
	}

	unbindResp := struct {
		Harness harnessResponse `json:"harness"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/workflows/%s/skills/unbind", createdWorkflow.ID),
		map[string]any{"skills": []string{"review-code", "commit"}},
		http.StatusOK,
		&unbindResp,
	)
	if unbindResp.Harness.Version != 3 {
		t.Fatalf("expected unbind response to advance workflow version, got %+v", unbindResp.Harness)
	}
	if unbindResp.Harness.Content != "# Coding\n" {
		t.Fatalf("expected unbind to preserve pure harness body, got %q", unbindResp.Harness.Content)
	}

	listAfterResp := struct {
		Skills []skillResponse `json:"skills"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/skills", project.ID),
		nil,
		http.StatusOK,
		&listAfterResp,
	)
	if len(listAfterResp.Skills) != len(builtin.Skills()) {
		t.Fatalf("expected %d skills after deprecated harvest attempt, got %+v", len(builtin.Skills()), listAfterResp.Skills)
	}
	for _, item := range listAfterResp.Skills {
		if len(item.BoundWorkflows) != 0 {
			t.Fatalf("expected %s to have no bound workflows, got %+v", item.Name, item.BoundWorkflows)
		}
	}
	for _, item := range listAfterResp.Skills {
		if item.Name == "deploy-docker" {
			t.Fatalf("expected deprecated harvest path to avoid creating deploy-docker, got %+v", item)
		}
	}
}

func TestSkillRefinementRouteStreamsRichRuntimeEvents(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	providerID := uuid.New()
	skillID := uuid.New()

	providerItem := catalogdomain.AgentProvider{
		ID:                providerID,
		OrganizationID:    orgID,
		Name:              "Codex",
		AdapterType:       catalogdomain.AgentProviderAdapterTypeCodexAppServer,
		AvailabilityState: catalogdomain.AgentProviderAvailabilityStateAvailable,
		Available:         true,
		ModelName:         "gpt-5.4",
	}

	runtime := &httpSkillRefinementRuntime{
		events: []chatservice.StreamEvent{
			{
				Event: "session_anchor",
				Payload: chatservice.RuntimeSessionAnchor{
					ProviderThreadID:      "thread-http",
					LastTurnID:            "turn-http",
					ProviderAnchorID:      "thread-http",
					ProviderAnchorKind:    "thread",
					ProviderTurnSupported: true,
				},
			},
			{
				Event:   "thread_status",
				Payload: mapRuntimeThreadStatusPayload("thread-http", "active", []string{"running"}),
			},
			{
				Event: "message",
				Payload: map[string]any{
					"type": "task_started",
					"raw":  map[string]any{"thread_id": "thread-http", "turn_id": "turn-http"},
				},
			},
			{
				Event: "message",
				Payload: map[string]any{
					"type": "task_progress",
					"raw":  map[string]any{"text": "bash -n scripts/check.sh\n./scripts/check.sh\nverified"},
				},
			},
			{
				Event:   "interrupt_requested",
				Payload: mapRuntimeInterruptEvent("req-http", "command_execution", map[string]any{"command": "git status"}),
			},
			{
				Event: "session_state",
				Payload: map[string]any{
					"status":       "active",
					"active_flags": []string{"running"},
					"detail":       "Verification running",
				},
			},
			{
				Event:   "plan_updated",
				Payload: mapRuntimePlanUpdatedPayload("thread-http", "turn-http"),
			},
			{
				Event:   "diff_updated",
				Payload: mapRuntimeDiffUpdatedPayload("thread-http", "turn-http", "diff --git a/SKILL.md b/SKILL.md"),
			},
			{
				Event:   "reasoning_updated",
				Payload: mapRuntimeReasoningUpdatedPayload("thread-http", "turn-http", "item-1"),
			},
			{
				Event:   "message",
				Payload: map[string]any{"type": "text", "content": `{"type":"skill_refinement_result","status":"verified","summary":"Bundle verified from HTTP SSE","verification_notes":"Verification command succeeded"}`},
			},
		},
		anchor: chatservice.RuntimeSessionAnchor{
			ProviderThreadID:      "thread-http",
			LastTurnID:            "turn-http",
			ProviderAnchorID:      "thread-http",
			ProviderAnchorKind:    "thread",
			ProviderTurnSupported: true,
		},
	}

	service := chatservice.NewSkillRefinementService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		runtime,
		httpSkillRefinementCatalog{
			project: catalogdomain.Project{
				ID:                     projectID,
				OrganizationID:         orgID,
				Name:                   "OpenASE",
				DefaultAgentProviderID: &providerID,
			},
			providers: []catalogdomain.AgentProvider{providerItem},
		},
		httpSkillRefinementWorkflow{
			skill: workflowservice.SkillDetail{
				Skill: workflowservice.Skill{
					ID:             skillID,
					Name:           "deploy-openase",
					CurrentVersion: 1,
					IsEnabled:      true,
				},
			},
		},
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
		WithSkillRefinementService(service),
	)

	requestBody := fmt.Sprintf(`{
		"project_id":"%s",
		"message":"Tighten the skill and keep the event stream detailed.",
		"provider_id":"%s",
		"files":[
			{"path":"SKILL.md","content_base64":"%s","media_type":"text/markdown; charset=utf-8","is_executable":false},
			{"path":"scripts/check.sh","content_base64":"%s","media_type":"text/x-shellscript; charset=utf-8","is_executable":true}
		]
	}`,
		projectID,
		providerID,
		base64.StdEncoding.EncodeToString([]byte("---\nname: deploy-openase\ndescription: Safely redeploy OpenASE\n---\n\n# Deploy\n")),
		base64.StdEncoding.EncodeToString([]byte("#!/usr/bin/env bash\necho verified\n")),
	)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/skills/%s/refinement-runs", skillID), strings.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(chatUserHeader, "user:skill-http")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get(echo.HeaderContentType); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}

	body := rec.Body.String()
	for _, expected := range []string{
		"event: session",
		"event: status",
		"event: session_anchor",
		"event: thread_status",
		"event: message",
		"event: interrupt_requested",
		"event: session_state",
		"event: plan_updated",
		"event: diff_updated",
		"event: reasoning_updated",
		"event: result",
		`"provider_thread_id":"thread-http"`,
		`"provider_turn_id":"turn-http"`,
		`"type":"task_progress"`,
		`"request_id":"req-http"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected SSE body to contain %q, got %q", expected, body)
		}
	}
}

type httpSkillRefinementRuntime struct {
	events []chatservice.StreamEvent
	anchor chatservice.RuntimeSessionAnchor
}

func (r *httpSkillRefinementRuntime) Supports(catalogdomain.AgentProvider) bool {
	return true
}

func (r *httpSkillRefinementRuntime) StartTurn(context.Context, chatservice.RuntimeTurnInput) (chatservice.TurnStream, error) {
	events := make(chan chatservice.StreamEvent, len(r.events))
	for _, event := range r.events {
		events <- event
	}
	close(events)
	return chatservice.TurnStream{Events: events}, nil
}

func (r *httpSkillRefinementRuntime) CloseSession(chatservice.SessionID) bool {
	return true
}

func (r *httpSkillRefinementRuntime) SessionAnchor(chatservice.SessionID) chatservice.RuntimeSessionAnchor {
	return r.anchor
}

type httpSkillRefinementCatalog struct {
	project   catalogdomain.Project
	providers []catalogdomain.AgentProvider
}

func (c httpSkillRefinementCatalog) GetProject(context.Context, uuid.UUID) (catalogdomain.Project, error) {
	return c.project, nil
}

func (httpSkillRefinementCatalog) ListActivityEvents(context.Context, catalogdomain.ListActivityEvents) ([]catalogdomain.ActivityEvent, error) {
	return nil, nil
}

func (httpSkillRefinementCatalog) ListProjectRepos(context.Context, uuid.UUID) ([]catalogdomain.ProjectRepo, error) {
	return nil, nil
}

func (httpSkillRefinementCatalog) ListTicketRepoScopes(context.Context, uuid.UUID, uuid.UUID) ([]catalogdomain.TicketRepoScope, error) {
	return nil, nil
}

func (c httpSkillRefinementCatalog) ListAgentProviders(context.Context, uuid.UUID) ([]catalogdomain.AgentProvider, error) {
	return c.providers, nil
}

func (c httpSkillRefinementCatalog) GetAgentProvider(context.Context, uuid.UUID) (catalogdomain.AgentProvider, error) {
	if len(c.providers) == 0 {
		return catalogdomain.AgentProvider{}, fmt.Errorf("provider not found")
	}
	return c.providers[0], nil
}

type httpSkillRefinementWorkflow struct {
	skill workflowservice.SkillDetail
}

func (httpSkillRefinementWorkflow) Get(context.Context, uuid.UUID) (workflowservice.WorkflowDetail, error) {
	return workflowservice.WorkflowDetail{}, nil
}

func (httpSkillRefinementWorkflow) List(context.Context, uuid.UUID) ([]workflowservice.Workflow, error) {
	return nil, nil
}

func (w httpSkillRefinementWorkflow) GetSkill(context.Context, uuid.UUID) (workflowservice.SkillDetail, error) {
	return w.skill, nil
}

func mapRuntimeThreadStatusPayload(threadID string, status string, activeFlags []string) map[string]any {
	return map[string]any{
		"thread_id":    threadID,
		"status":       status,
		"active_flags": activeFlags,
	}
}

func mapRuntimeInterruptEvent(requestID string, kind string, payload map[string]any) map[string]any {
	return map[string]any{
		"request_id": requestID,
		"kind":       kind,
		"payload":    payload,
	}
}

func mapRuntimePlanUpdatedPayload(threadID string, turnID string) map[string]any {
	return map[string]any{
		"thread_id": threadID,
		"turn_id":   turnID,
		"plan": []map[string]any{
			{"step": "Inspect", "status": "completed"},
			{"step": "Verify", "status": "completed"},
		},
	}
}

func mapRuntimeDiffUpdatedPayload(threadID string, turnID string, diff string) map[string]any {
	return map[string]any{
		"thread_id": threadID,
		"turn_id":   turnID,
		"diff":      diff,
	}
}

func mapRuntimeReasoningUpdatedPayload(threadID string, turnID string, itemID string) map[string]any {
	return map[string]any{
		"thread_id": threadID,
		"turn_id":   turnID,
		"item_id":   itemID,
		"kind":      "summary_text_delta",
		"delta":     "Reasoning",
	}
}

func TestSkillRoutesImportBundleAndExposeFiles(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40024},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		newTicketStatusService(client),
		nil,
		nil,
		workflowSvc,
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
	createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)

	importResp := skillDetailResponse{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/skills/import", project.ID),
		map[string]any{
			"name":       "deploy-openase",
			"created_by": "user:cli",
			"files": []map[string]any{
				{
					"path":           "SKILL.md",
					"content_base64": base64.StdEncoding.EncodeToString([]byte("---\nname: \"deploy-openase\"\ndescription: \"Safely redeploy OpenASE\"\n---\n\n# Deploy OpenASE\n\nUse the bundled scripts.\n")),
					"media_type":     "text/markdown; charset=utf-8",
				},
				{
					"path":           "scripts/redeploy.sh",
					"content_base64": base64.StdEncoding.EncodeToString([]byte("#!/usr/bin/env bash\necho deploy\n")),
					"media_type":     "text/x-shellscript; charset=utf-8",
					"is_executable":  true,
				},
				{
					"path":           "references/runbook.md",
					"content_base64": base64.StdEncoding.EncodeToString([]byte("# Runbook\n\n1. Validate\n")),
					"media_type":     "text/markdown; charset=utf-8",
				},
			},
		},
		http.StatusCreated,
		&importResp,
	)
	if importResp.Skill.Name != "deploy-openase" || importResp.Skill.CreatedBy != "user:cli" || len(importResp.Files) != 3 {
		t.Fatalf("unexpected import response: %+v", importResp)
	}
	scriptFile := findSkillFileResponse(t, importResp.Files, "scripts/redeploy.sh")
	if !scriptFile.IsExecutable || scriptFile.Content != "#!/usr/bin/env bash\necho deploy\n" {
		t.Fatalf("unexpected script file response: %+v", scriptFile)
	}

	filesResp := skillFilesResponse{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/skills/%s/files", importResp.Skill.ID),
		nil,
		http.StatusOK,
		&filesResp,
	)
	if len(filesResp.Files) != 3 {
		t.Fatalf("expected 3 skill files, got %+v", filesResp)
	}
	referenceFile := findSkillFileResponse(t, filesResp.Files, "references/runbook.md")
	if referenceFile.Content != "# Runbook\n\n1. Validate\n" {
		t.Fatalf("unexpected reference file response: %+v", referenceFile)
	}
	entrypointFile := findSkillFileResponse(t, filesResp.Files, "SKILL.md")
	if !strings.Contains(entrypointFile.Content, "name: \"deploy-openase\"") {
		t.Fatalf("unexpected entrypoint file response: %+v", entrypointFile)
	}

	updateResp := skillDetailResponse{}
	updatedEntrypoint := "# Deploy OpenASE\n\nUse the updated release flow.\n"
	executeJSON(
		t,
		server,
		http.MethodPut,
		fmt.Sprintf("/api/v1/skills/%s", importResp.Skill.ID),
		map[string]any{
			"description": "Safely redeploy OpenASE with the updated release flow.",
			"content":     updatedEntrypoint,
			"files": []map[string]any{
				{
					"path":           "SKILL.md",
					"content_base64": base64.StdEncoding.EncodeToString([]byte("placeholder")),
					"media_type":     "text/markdown; charset=utf-8",
				},
				{
					"path":           "scripts/release.sh",
					"content_base64": base64.StdEncoding.EncodeToString([]byte("#!/usr/bin/env bash\necho release\n")),
					"media_type":     "text/x-shellscript; charset=utf-8",
					"is_executable":  true,
				},
				{
					"path":           "assets/logo.txt",
					"content_base64": base64.StdEncoding.EncodeToString([]byte("openase\n")),
					"media_type":     "text/plain; charset=utf-8",
				},
			},
		},
		http.StatusOK,
		&updateResp,
	)
	if updateResp.Skill.CurrentVersion != 2 || len(updateResp.Files) != 3 {
		t.Fatalf("unexpected update response: %+v", updateResp)
	}
	updatedScript := findSkillFileResponse(t, updateResp.Files, "scripts/release.sh")
	if !updatedScript.IsExecutable || updatedScript.Content != "#!/usr/bin/env bash\necho release\n" {
		t.Fatalf("unexpected updated script file response: %+v", updatedScript)
	}
	if strings.Contains(findSkillFileResponse(t, updateResp.Files, "SKILL.md").Content, "placeholder") {
		t.Fatalf("expected update to normalize entrypoint content, got %+v", updateResp.Files)
	}

	filesAfterUpdate := skillFilesResponse{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/skills/%s/files", importResp.Skill.ID),
		nil,
		http.StatusOK,
		&filesAfterUpdate,
	)
	if len(filesAfterUpdate.Files) != 3 {
		t.Fatalf("expected 3 updated skill files, got %+v", filesAfterUpdate)
	}
	if containsSkillFileResponse(filesAfterUpdate.Files, "scripts/redeploy.sh") {
		t.Fatalf("expected renamed script path to be removed, got %+v", filesAfterUpdate)
	}
	if containsSkillFileResponse(filesAfterUpdate.Files, "references/runbook.md") {
		t.Fatalf("expected deleted directory content to be removed, got %+v", filesAfterUpdate)
	}
	updatedEntrypointFile := findSkillFileResponse(t, filesAfterUpdate.Files, "SKILL.md")
	if !strings.Contains(updatedEntrypointFile.Content, "Use the updated release flow.") {
		t.Fatalf("unexpected updated entrypoint file response: %+v", updatedEntrypointFile)
	}
}

func containsSkillName(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}

	return false
}

func containsSkillFileResponse(items []skillFileResponse, want string) bool {
	for _, item := range items {
		if item.Path == want {
			return true
		}
	}

	return false
}

func TestBuiltinRoleLibraryRoute(t *testing.T) {
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
	)

	resp := struct {
		Roles []builtinRoleResponse `json:"roles"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/roles/builtin",
		nil,
		http.StatusOK,
		&resp,
	)
	if len(resp.Roles) != 17 {
		t.Fatalf("expected 17 builtin roles, got %+v", resp.Roles)
	}
	if resp.Roles[0].HarnessPath == "" || resp.Roles[0].Content == "" || resp.Roles[0].WorkflowContent == "" {
		t.Fatalf("expected role payload to include harness path and content, got %+v", resp.Roles[0])
	}
	if resp.Roles[0].Slug != "dispatcher" {
		t.Fatalf("expected dispatcher to be included in builtin role payload, got %+v", resp.Roles[0])
	}
	findBuiltinRoleResponse(t, resp.Roles, "harness-optimizer")
	findBuiltinRoleResponse(t, resp.Roles, "env-provisioner")
}

func TestBuiltinRoleDetailRoute(t *testing.T) {
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
	)

	resp := builtinRoleDetailResponse{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/roles/builtin/fullstack-developer",
		nil,
		http.StatusOK,
		&resp,
	)
	if resp.Role.Slug != "fullstack-developer" {
		t.Fatalf("expected fullstack-developer role, got %+v", resp.Role)
	}
	if resp.Role.Content == "" || resp.Role.WorkflowContent == "" {
		t.Fatalf("expected role detail to include workflow content, got %+v", resp.Role)
	}
	if resp.Role.Content != resp.Role.WorkflowContent {
		t.Fatalf("expected workflow content alias to match content, got %+v", resp.Role)
	}
}

func TestBuiltinRoleDetailRouteRejectsMissingRole(t *testing.T) {
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
	)

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/roles/builtin/missing-role", "")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "ROLE_TEMPLATE_NOT_FOUND") {
		t.Fatalf("expected ROLE_TEMPLATE_NOT_FOUND, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSkillBindRouteRejectsMissingSkill(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := workflowSvc.Close(); closeErr != nil {
			t.Errorf("close workflow service: %v", closeErr)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		newTicketStatusService(client),
		nil,
		nil,
		workflowSvc,
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
	createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)
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
	attachPrimaryProjectRepoCheckout(ctx, t, client, project.ID, localMachine.ID, repoRoot)
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
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
	agent, err := client.Agent.Create().
		SetProviderID(provider.ID).
		SetProjectID(project.ID).
		SetName("codex-coding").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	createdWorkflow, err := workflowSvc.Create(ctx, workflowservice.CreateInput{
		ProjectID:           project.ID,
		AgentID:             agent.ID,
		Name:                "Coding Workflow",
		Type:                "coding",
		HarnessContent:      "# Coding\n",
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
		t.Fatalf("create workflow: %v", err)
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/workflows/%s/skills/bind", createdWorkflow.ID),
		`{"skills":["missing-skill"]}`,
	)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing skill, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func writeSkillFixture(t *testing.T, repoRoot string, name string, content string) {
	t.Helper()
	path := filepath.Join(repoRoot, ".openase", "skills", name, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create skill fixture parent: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		title := parseSkillTitle(content)
		if title == "" {
			title = name
		}
		content = fmt.Sprintf("---\nname: %q\ndescription: %q\n---\n\n%s\n", name, title, strings.TrimSpace(content))
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write skill fixture: %v", err)
	}
}

func writeWorkspaceSkill(t *testing.T, workspaceRoot string, adapterDir string, name string, content string) {
	t.Helper()
	path := filepath.Join(workspaceRoot, adapterDir, "skills", name, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create workspace skill parent: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(content), "---") {
		title := parseSkillTitle(content)
		if title == "" {
			title = name
		}
		content = fmt.Sprintf("---\nname: %q\ndescription: %q\n---\n\n%s\n", name, title, strings.TrimSpace(content))
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write workspace skill: %v", err)
	}
}

func findSkillResponse(t *testing.T, items []skillResponse, name string) skillResponse {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("expected to find skill %s in %+v", name, items)
	return skillResponse{}
}

func findSkillFileResponse(t *testing.T, items []skillFileResponse, path string) skillFileResponse {
	t.Helper()

	for _, item := range items {
		if item.Path == path {
			return item
		}
	}
	t.Fatalf("expected to find skill file %s in %+v", path, items)
	return skillFileResponse{}
}

func findBuiltinRoleResponse(t *testing.T, items []builtinRoleResponse, slug string) {
	t.Helper()
	for _, item := range items {
		if item.Slug == slug {
			return
		}
	}
	t.Fatalf("role %s not found in %+v", slug, items)
}

func parseSkillTitle(content string) string {
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return ""
}
