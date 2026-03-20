package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
)

func TestSkillRoutesRefreshHarvestBindAndUnbind(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("create git marker: %v", err)
	}

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

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")

	writeSkillFixture(t, repoRoot, "commit", "# Commit\n\nWrite a conventional commit message.\n")
	writeSkillFixture(t, repoRoot, "review-code", "# Review Code\n\nReview the patch before shipping.\n")

	createdWorkflow, err := workflowSvc.Create(ctx, workflowservice.CreateInput{
		ProjectID:           project.ID,
		Name:                "Coding Workflow",
		Type:                "coding",
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      60,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusID:      todoID,
		FinishStatusID:      &doneID,
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
	if len(boundSkills) != 2 || boundSkills[0] != "review-code" || boundSkills[1] != "commit" {
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
	if len(listResp.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %+v", listResp.Skills)
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

	workspaceRoot := t.TempDir()
	refreshResp := skillSyncResponse{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/skills/refresh", project.ID),
		map[string]any{
			"workspace_path": workspaceRoot,
			"adapter_type":   "claude-code-cli",
		},
		http.StatusOK,
		&refreshResp,
	)
	if len(refreshResp.InjectedSkills) != 2 {
		t.Fatalf("expected 2 injected skills, got %+v", refreshResp)
	}
	refreshedSkill, err := os.ReadFile(filepath.Join(workspaceRoot, ".claude", "skills", "review-code", "SKILL.md"))
	if err != nil {
		t.Fatalf("read refreshed skill: %v", err)
	}
	if string(refreshedSkill) == "" {
		t.Fatalf("expected refreshed skill content")
	}

	writeWorkspaceSkill(t, workspaceRoot, ".claude", "deploy-docker", "# Deploy Docker\n\nDeploy the app with Docker.\n")
	writeWorkspaceSkill(t, workspaceRoot, ".claude", "commit", "# Commit\n\nWrite a stricter conventional commit message.\n")

	harvestResp := skillSyncResponse{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/skills/harvest", project.ID),
		map[string]any{
			"workspace_path": workspaceRoot,
			"adapter_type":   "claude-code-cli",
		},
		http.StatusOK,
		&harvestResp,
	)
	if len(harvestResp.HarvestedSkills) != 1 || harvestResp.HarvestedSkills[0] != "deploy-docker" {
		t.Fatalf("expected deploy-docker to be harvested, got %+v", harvestResp)
	}
	if len(harvestResp.UpdatedSkills) != 1 || harvestResp.UpdatedSkills[0] != "commit" {
		t.Fatalf("expected commit to be updated, got %+v", harvestResp)
	}

	harvestedSkill, err := os.ReadFile(filepath.Join(repoRoot, ".openase", "skills", "deploy-docker", "SKILL.md"))
	if err != nil {
		t.Fatalf("read harvested skill: %v", err)
	}
	if string(harvestedSkill) == "" {
		t.Fatalf("expected harvested skill content")
	}
	updatedCommit, err := os.ReadFile(filepath.Join(repoRoot, ".openase", "skills", "commit", "SKILL.md"))
	if err != nil {
		t.Fatalf("read updated commit skill: %v", err)
	}
	if string(updatedCommit) != "# Commit\n\nWrite a stricter conventional commit message.\n" {
		t.Fatalf("unexpected updated commit skill content: %q", string(updatedCommit))
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
	if len(listAfterResp.Skills) != 3 {
		t.Fatalf("expected 3 skills after harvest, got %+v", listAfterResp.Skills)
	}
	for _, item := range listAfterResp.Skills {
		if len(item.BoundWorkflows) != 0 {
			t.Fatalf("expected %s to have no bound workflows, got %+v", item.Name, item.BoundWorkflows)
		}
	}
	deploySkill := findSkillResponse(t, listAfterResp.Skills, "deploy-docker")
	if deploySkill.IsBuiltin {
		t.Fatalf("expected harvested deploy-docker skill to be non built-in, got %+v", deploySkill)
	}
	if deploySkill.Description != "Deploy Docker" {
		t.Fatalf("expected deploy-docker description to come from SKILL.md title, got %+v", deploySkill)
	}
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
	if len(resp.Roles) != 15 {
		t.Fatalf("expected 15 builtin roles, got %+v", resp.Roles)
	}
	if resp.Roles[0].HarnessPath == "" || resp.Roles[0].Content == "" {
		t.Fatalf("expected role payload to include harness path and content, got %+v", resp.Roles[0])
	}
	if resp.Roles[0].Slug != "dispatcher" {
		t.Fatalf("expected dispatcher to be included in builtin role payload, got %+v", resp.Roles[0])
	}
}

func TestSkillBindRouteRejectsMissingSkill(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("create git marker: %v", err)
	}

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
	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")

	createdWorkflow, err := workflowSvc.Create(ctx, workflowservice.CreateInput{
		ProjectID:           project.ID,
		Name:                "Coding Workflow",
		Type:                "coding",
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      60,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusID:      todoID,
		FinishStatusID:      &doneID,
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create skill fixture parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write skill fixture: %v", err)
	}
}

func writeWorkspaceSkill(t *testing.T, workspaceRoot string, adapterDir string, name string, content string) {
	t.Helper()
	path := filepath.Join(workspaceRoot, adapterDir, "skills", name, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create workspace skill parent: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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
