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
	"github.com/BetterAndBetterII/openase/internal/builtin"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func TestSkillRoutesErrorMappingsAndInvalidPayloads(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
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
		ticketstatus.NewService(client),
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
		ticketstatus.NewService(client),
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
		{name: "harvest invalid project", server: server, method: http.MethodPost, target: "/api/v1/projects/not-a-uuid/skills/harvest", body: `{"workspace_root":"/tmp/ws","adapter_type":"claude-code-cli"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "harvest invalid payload", server: server, method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/skills/harvest", uuid.New()), body: `{"workspace_root":"/tmp/ws","adapter_type":"   "}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
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

func TestSkillRoutesRefreshHarvestBindAndUnbind(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
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
		ticketstatus.NewService(client),
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

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
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
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
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
	boundSkills, err := workflowservice.ParseHarnessSkills(bindResp.Harness.Content)
	if err != nil {
		t.Fatalf("parse bound harness skills: %v", err)
	}
	if len(boundSkills) != 2 || boundSkills[0] != "commit" || boundSkills[1] != "review-code" {
		t.Fatalf("unexpected bound skills: %#v", boundSkills)
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
	if _, err := os.Stat(filepath.Join(workspaceRoot, ".openase", "bin", "openase")); err != nil {
		t.Fatalf("expected openase wrapper in refreshed workspace: %v", err)
	}

	writeWorkspaceSkill(t, workspaceRoot, ".claude", "deploy-docker", "# Deploy Docker\n\nDeploy the app with Docker.\n")
	writeWorkspaceSkill(t, workspaceRoot, ".claude", "commit", "# Commit\n\nWrite a stricter conventional commit message.\n")

	harvestRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/skills/harvest", project.ID),
		fmt.Sprintf(`{"workspace_root":%q,"adapter_type":"claude-code-cli"}`, workspaceRoot),
	)
	if harvestRec.Code != http.StatusBadRequest ||
		!strings.Contains(harvestRec.Body.String(), "INVALID_SKILL") ||
		!strings.Contains(harvestRec.Body.String(), "harvest is deprecated") {
		t.Fatalf("expected harvest deprecation error, got %d: %s", harvestRec.Code, harvestRec.Body.String())
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
	unboundSkills, err := workflowservice.ParseHarnessSkills(unbindResp.Harness.Content)
	if err != nil {
		t.Fatalf("parse unbound harness skills: %v", err)
	}
	if len(unboundSkills) != 0 {
		t.Fatalf("expected all skills to be unbound, got %#v", unboundSkills)
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

func containsSkillName(items []string, want string) bool {
	for _, item := range items {
		if item == want {
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
	if resp.Roles[0].HarnessPath == "" || resp.Roles[0].Content == "" {
		t.Fatalf("expected role payload to include harness path and content, got %+v", resp.Roles[0])
	}
	if resp.Roles[0].Slug != "dispatcher" {
		t.Fatalf("expected dispatcher to be included in builtin role payload, got %+v", resp.Roles[0])
	}
	findBuiltinRoleResponse(t, resp.Roles, "harness-optimizer")
	findBuiltinRoleResponse(t, resp.Roles, "env-provisioner")
}

func TestSkillBindRouteRejectsMissingSkill(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
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
		ticketstatus.NewService(client),
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
	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
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
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
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
