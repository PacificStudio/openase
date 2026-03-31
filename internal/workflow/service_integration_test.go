package workflow

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entskill "github.com/BetterAndBetterII/openase/ent/skill"
	entskillversion "github.com/BetterAndBetterII/openase/ent/skillversion"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
)

func TestWorkflowServiceCRUDHarnessStorageSkillsAndReload(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	activateMarkerPath := filepath.Join(repoRoot, "activate.marker")
	reloadMarkerPath := filepath.Join(repoRoot, "reload.marker")
	workflowHooks := map[string]any{
		"workflow_hooks": map[string]any{
			"on_activate": []map[string]any{{
				"cmd": "printf '%s:%s' \"$OPENASE_WORKFLOW_NAME\" \"$OPENASE_WORKFLOW_VERSION\" > activate.marker",
			}},
			"on_reload": []map[string]any{{
				"cmd": "printf '%s:%s' \"$OPENASE_HOOK_NAME\" \"$OPENASE_WORKFLOW_VERSION\" > reload.marker",
			}},
		},
	}

	created, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Coding Workflow",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		Hooks:               workflowHooks,
		MaxConcurrent:       3,
		MaxRetryAttempts:    2,
		TimeoutMinutes:      45,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.HarnessPath != ".openase/harnesses/coding-workflow.md" || created.Version != 1 || !created.IsActive {
		t.Fatalf("Create() = %+v", created)
	}
	if got := mustReadWorkflowFile(t, activateMarkerPath); got != "Coding Workflow:1" {
		t.Fatalf("activate marker = %q", got)
	}

	listed, err := service.List(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("List() = %+v", listed)
	}

	got, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID || got.HarnessContent != created.HarnessContent {
		t.Fatalf("Get() = %+v", got)
	}

	harnessDoc, err := service.GetHarness(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetHarness() error = %v", err)
	}
	if harnessDoc.Path != created.HarnessPath || harnessDoc.Content != created.HarnessContent || harnessDoc.Version != 1 {
		t.Fatalf("GetHarness() = %+v", harnessDoc)
	}

	if _, err := service.CreateSkill(ctx, CreateSkillInput{
		ProjectID: fixture.projectID,
		Name:      "skill-one",
		Content:   "# Skill One\nbody\n",
	}); err != nil {
		t.Fatalf("CreateSkill(skill-one) error = %v", err)
	}
	if _, err := service.CreateSkill(ctx, CreateSkillInput{
		ProjectID: fixture.projectID,
		Name:      "skill-two",
		Content:   "# Skill Two\nbody\n",
	}); err != nil {
		t.Fatalf("CreateSkill(skill-two) error = %v", err)
	}

	boundDoc, err := service.BindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: created.ID,
		Skills:     []string{"skill-one", "skill-two"},
	})
	if err != nil {
		t.Fatalf("BindSkills() error = %v", err)
	}
	if boundDoc.Version != 1 {
		t.Fatalf("BindSkills() version = %d, want 1", boundDoc.Version)
	}
	if _, err := os.Stat(reloadMarkerPath); !os.IsNotExist(err) {
		t.Fatalf("expected bind to avoid harness reload, stat error = %v", err)
	}
	if skillNames, err := ParseHarnessSkills(boundDoc.Content); err != nil || !slices.Equal(skillNames, []string{"skill-one", "skill-two"}) {
		t.Fatalf("ParseHarnessSkills(boundDoc) = %+v, %v", skillNames, err)
	}

	skills, err := service.ListSkills(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}
	skillOne := findSkillByName(skills, "skill-one")
	skillTwo := findSkillByName(skills, "skill-two")
	if skillOne == nil || skillTwo == nil || len(skillOne.BoundWorkflows) != 1 || skillOne.BoundWorkflows[0].ID != created.ID || skillTwo.IsBuiltin {
		t.Fatalf("ListSkills() = %+v", skills)
	}

	workspaceRoot := t.TempDir()
	legacySkillDir := filepath.Join(workspaceRoot, ".codex", "skills", "legacy-skill")
	if err := os.MkdirAll(legacySkillDir, 0o750); err != nil {
		t.Fatalf("mkdir legacy skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacySkillDir, "SKILL.md"), []byte("# Legacy Skill\nbody\n"), 0o600); err != nil {
		t.Fatalf("write legacy skill file: %v", err)
	}
	refreshResult, err := service.RefreshSkills(ctx, RefreshSkillsInput{
		ProjectID:     fixture.projectID,
		WorkspaceRoot: workspaceRoot,
		AdapterType:   string(entagentprovider.AdapterTypeCodexAppServer),
		WorkflowID:    &created.ID,
	})
	if err != nil {
		t.Fatalf("RefreshSkills() error = %v", err)
	}
	if !slices.Equal(refreshResult.InjectedSkills, []string{"openase-platform", "skill-one", "skill-two"}) {
		t.Fatalf("RefreshSkills() = %+v", refreshResult)
	}

	workspaceSkillsRoot := filepath.Join(workspaceRoot, ".codex", "skills")
	if _, err := os.Stat(filepath.Join(workspaceSkillsRoot, "legacy-skill")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy workspace skill to be removed, stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspaceRoot, ".openase", "bin", "openase")); err != nil {
		t.Fatalf("expected workspace wrapper to exist: %v", err)
	}
	//nolint:gosec // test reads a controlled projected skill fixture in the temp workspace.
	if projected, err := os.ReadFile(filepath.Join(workspaceSkillsRoot, "skill-one", "SKILL.md")); err != nil || !strings.Contains(string(projected), "# Skill One") {
		t.Fatalf("projected skill-one = %q, err=%v", string(projected), err)
	}

	if _, err := service.HarvestSkills(ctx, HarvestSkillsInput{
		ProjectID:     fixture.projectID,
		WorkspaceRoot: workspaceRoot,
		AdapterType:   string(entagentprovider.AdapterTypeCodexAppServer),
	}); !errors.Is(err, ErrSkillInvalid) {
		t.Fatalf("HarvestSkills() error = %v, want %v", err, ErrSkillInvalid)
	}

	unboundDoc, err := service.UnbindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: created.ID,
		Skills:     []string{"skill-two"},
	})
	if err != nil {
		t.Fatalf("UnbindSkills() error = %v", err)
	}
	if unboundDoc.Version != 1 {
		t.Fatalf("UnbindSkills() version = %d, want 1", unboundDoc.Version)
	}
	if skillNames, err := ParseHarnessSkills(unboundDoc.Content); err != nil || !slices.Equal(skillNames, []string{"skill-one"}) {
		t.Fatalf("ParseHarnessSkills(unboundDoc) = %+v, %v", skillNames, err)
	}

	updated, err := service.Update(ctx, UpdateInput{
		WorkflowID:          created.ID,
		Name:                Some("Core Coding Workflow"),
		HarnessPath:         Some(".openase/harnesses/platform/core-coding.md"),
		MaxConcurrent:       Some(7),
		MaxRetryAttempts:    Some(4),
		TimeoutMinutes:      Some(90),
		StallTimeoutMinutes: Some(10),
		PickupStatusIDs:     Some(MustStatusBindingSet(fixture.statusIDs["Backlog"])),
		FinishStatusIDs:     Some(MustStatusBindingSet(fixture.statusIDs["Done"])),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "Core Coding Workflow" || updated.HarnessPath != ".openase/harnesses/platform/core-coding.md" || updated.MaxConcurrent != 7 {
		t.Fatalf("Update() = %+v", updated)
	}

	updatedHarnessContent := "---\nworkflow:\n  role: coding\nskills:\n  - skill-two\n---\n\n# Updated by API\n"
	updatedHarnessDoc, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: created.ID,
		Content:    updatedHarnessContent,
	})
	if err != nil {
		t.Fatalf("UpdateHarness() error = %v", err)
	}
	if updatedHarnessDoc.Version != 2 {
		t.Fatalf("UpdateHarness() = %+v", updatedHarnessDoc)
	}
	if got := mustReadWorkflowFile(t, reloadMarkerPath); got != "on_reload:2" {
		t.Fatalf("reload marker after UpdateHarness() = %q", got)
	}
	if skillNames, err := ParseHarnessSkills(updatedHarnessDoc.Content); err != nil || !slices.Equal(skillNames, []string{"skill-one"}) {
		t.Fatalf("projected harness skills after UpdateHarness() = %+v, %v", skillNames, err)
	}
	currentVersion, err := service.currentWorkflowVersion(ctx, created.ID)
	if err != nil {
		t.Fatalf("currentWorkflowVersion() error = %v", err)
	}
	if strings.Contains(currentVersion.ContentMarkdown, "skill-two") || strings.Contains(currentVersion.ContentMarkdown, "skills:") {
		t.Fatalf("stored workflow version still carries projected bindings: %q", currentVersion.ContentMarkdown)
	}

	if err := client.Project.UpdateOneID(fixture.projectID).SetDefaultWorkflowID(created.ID).Exec(ctx); err != nil {
		t.Fatalf("set default workflow: %v", err)
	}
	deleted, err := service.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted.ID != created.ID {
		t.Fatalf("Delete() = %+v", deleted)
	}
	projectAfterDelete, err := client.Project.Get(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("load project after delete: %v", err)
	}
	if projectAfterDelete.DefaultWorkflowID != nil {
		t.Fatalf("project default workflow = %v, want nil", projectAfterDelete.DefaultWorkflowID)
	}
	if _, err := service.Get(ctx, created.ID); !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("Get() after delete error = %v, want %v", err, ErrWorkflowNotFound)
	}

	if err := service.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if len(service.storages) != 0 {
		t.Fatalf("service.storages after Close() = %d, want 0", len(service.storages))
	}
}

func TestRuntimeSnapshotMaterializationAndRecordedResolution(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	created, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Runtime Snapshot Workflow",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Snapshot v1\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    2,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.CreateSkill(ctx, CreateSkillInput{
		ProjectID: fixture.projectID,
		Name:      "runtime-skill",
		Content:   "# Runtime Skill\nv1\n",
	}); err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}
	if _, err := service.BindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: created.ID,
		Skills:     []string{"runtime-skill"},
	}); err != nil {
		t.Fatalf("BindSkills() error = %v", err)
	}

	snapshotV1, err := service.ResolveRuntimeSnapshot(ctx, created.ID)
	if err != nil {
		t.Fatalf("ResolveRuntimeSnapshot(v1) error = %v", err)
	}
	if snapshotV1.Workflow.VersionID == uuid.Nil || len(snapshotV1.Skills) != 2 {
		t.Fatalf("ResolveRuntimeSnapshot(v1) = %+v", snapshotV1)
	}
	v1Skill := findRuntimeSkillSnapshot(t, snapshotV1.Skills, "runtime-skill")

	codexRoot := t.TempDir()
	materializedV1, err := service.MaterializeRuntimeSnapshot(MaterializeRuntimeSnapshotInput{
		WorkspaceRoot: codexRoot,
		AdapterType:   string(entagentprovider.AdapterTypeCodexAppServer),
		Snapshot:      snapshotV1,
	})
	if err != nil {
		t.Fatalf("MaterializeRuntimeSnapshot(codex) error = %v", err)
	}
	if _, err := os.Stat(materializedV1.HarnessPath); err != nil {
		t.Fatalf("expected runtime harness snapshot: %v", err)
	}
	if _, err := os.Stat(filepath.Join(codexRoot, ".codex", "skills", "runtime-skill", "SKILL.md")); err != nil {
		t.Fatalf("expected codex skill projection: %v", err)
	}

	if _, err := service.UpdateSkill(ctx, UpdateSkillInput{
		SkillID: findSkillIDByName(ctx, t, client, fixture.projectID, "runtime-skill"),
		Content: "# Runtime Skill\nv2\n",
	}); err != nil {
		t.Fatalf("UpdateSkill() error = %v", err)
	}
	if _, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: created.ID,
		Content:    "---\nworkflow:\n  role: coding\n---\n\n# Snapshot v2\n",
	}); err != nil {
		t.Fatalf("UpdateHarness() error = %v", err)
	}

	snapshotV2, err := service.ResolveRuntimeSnapshot(ctx, created.ID)
	if err != nil {
		t.Fatalf("ResolveRuntimeSnapshot(v2) error = %v", err)
	}
	if snapshotV2.Workflow.VersionID == snapshotV1.Workflow.VersionID {
		t.Fatalf("expected workflow version to advance, got %+v", snapshotV2.Workflow)
	}
	v2Skill := findRuntimeSkillSnapshot(t, snapshotV2.Skills, "runtime-skill")
	if v2Skill.VersionID == v1Skill.VersionID {
		t.Fatalf("expected skill version to advance, got %+v", snapshotV2.Skills)
	}

	recordedV1, err := service.ResolveRecordedRuntimeSnapshot(ctx, ResolveRecordedRuntimeSnapshotInput{
		WorkflowID:        created.ID,
		WorkflowVersionID: &snapshotV1.Workflow.VersionID,
		SkillVersionIDs:   skillVersionIDs(snapshotV1.Skills),
	})
	if err != nil {
		t.Fatalf("ResolveRecordedRuntimeSnapshot(v1) error = %v", err)
	}
	if recordedV1.Workflow.Content != snapshotV1.Workflow.Content {
		t.Fatalf("expected recorded snapshot to keep v1 workflow content, got %q want %q", recordedV1.Workflow.Content, snapshotV1.Workflow.Content)
	}

	claudeRoot := t.TempDir()
	if _, err := service.MaterializeRuntimeSnapshot(MaterializeRuntimeSnapshotInput{
		WorkspaceRoot: claudeRoot,
		AdapterType:   string(entagentprovider.AdapterTypeClaudeCodeCli),
		Snapshot:      snapshotV2,
	}); err != nil {
		t.Fatalf("MaterializeRuntimeSnapshot(claude) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(claudeRoot, ".claude", "skills", "runtime-skill", "SKILL.md")); err != nil {
		t.Fatalf("expected claude skill projection: %v", err)
	}
}

func TestWorkflowServiceSkillLifecycleCommits(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	createdWorkflow, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Coding Workflow",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      60,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	})
	if err != nil {
		t.Fatalf("Create() workflow error = %v", err)
	}

	createdSkill, err := service.CreateSkill(ctx, CreateSkillInput{
		ProjectID:   fixture.projectID,
		Name:        "deploy-docker",
		Content:     "# Deploy Docker\n\nRun the Docker deployment flow.\n",
		Description: "Deploy Docker",
	})
	if err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}
	if _, err := service.DisableSkill(ctx, createdSkill.ID); err != nil {
		t.Fatalf("DisableSkill() error = %v", err)
	}
	if _, err := service.EnableSkill(ctx, createdSkill.ID); err != nil {
		t.Fatalf("EnableSkill() error = %v", err)
	}
	if _, err := service.BindSkill(ctx, UpdateSkillBindingsInput{
		SkillID:     createdSkill.ID,
		WorkflowIDs: []uuid.UUID{createdWorkflow.ID},
	}); err != nil {
		t.Fatalf("BindSkill() error = %v", err)
	}
	if _, err := service.UpdateSkill(ctx, UpdateSkillInput{
		SkillID:     createdSkill.ID,
		Content:     "# Deploy Docker\n\nRun the hardened Docker deployment flow.\n",
		Description: "Deploy Docker",
	}); err != nil {
		t.Fatalf("UpdateSkill() error = %v", err)
	}
	if err := service.DeleteSkill(ctx, createdSkill.ID); err != nil {
		t.Fatalf("DeleteSkill() error = %v", err)
	}
	if _, err := service.GetSkill(ctx, createdSkill.ID); !errors.Is(err, ErrSkillNotFound) {
		t.Fatalf("GetSkill() after delete error = %v, want %v", err, ErrSkillNotFound)
	}
	archivedSkill, err := client.Skill.Get(ctx, createdSkill.ID)
	if err != nil {
		t.Fatalf("load archived skill: %v", err)
	}
	if archivedSkill.ArchivedAt == nil || archivedSkill.IsEnabled {
		t.Fatalf("archived skill = %+v", archivedSkill)
	}
	versionCount, err := client.SkillVersion.Query().Where(entskillversion.SkillIDEQ(createdSkill.ID)).Count(ctx)
	if err != nil {
		t.Fatalf("count skill versions: %v", err)
	}
	if versionCount != 2 {
		t.Fatalf("skill version count = %d, want 2", versionCount)
	}
}

func TestWorkflowServiceUnbindSkillIgnoresUnrelatedProjectRepoState(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-isolation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}

	projectWithoutRepoWorkspace, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Repo Workspace Pending").
		SetSlug("repo-workspace-pending").
		SetStatus("In Progress").
		Save(ctx)
	if err != nil {
		t.Fatalf("create projectWithoutRepoWorkspace: %v", err)
	}
	if _, err := client.ProjectRepo.Create().
		SetProjectID(projectWithoutRepoWorkspace.ID).
		SetName("todo-app").
		SetRepositoryURL("https://github.com/acme/todo-app.git").
		SetDefaultBranch("main").
		SetWorkspaceDirname("todo-app").
		SetIsPrimary(true).
		Save(ctx); err != nil {
		t.Fatalf("create project repo without workspace: %v", err)
	}

	readyProject, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-isolated").
		SetStatus("In Progress").
		Save(ctx)
	if err != nil {
		t.Fatalf("create readyProject: %v", err)
	}
	if _, err := client.ProjectRepo.Create().
		SetProjectID(readyProject.ID).
		SetName(filepath.Base(repoRoot)).
		SetRepositoryURL(repoRoot).
		SetDefaultBranch("main").
		SetWorkspaceDirname(filepath.Base(repoRoot)).
		SetIsPrimary(true).
		Save(ctx); err != nil {
		t.Fatalf("create ready project repo: %v", err)
	}

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, readyProject.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	statusIDs := make(map[string]uuid.UUID, len(statuses))
	for _, status := range statuses {
		statusIDs[status.Name] = status.ID
	}

	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(readyProject.ID).
		SetProviderID(providerItem.ID).
		SetName("codex-coding").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	createdWorkflow, err := service.Create(ctx, CreateInput{
		ProjectID:           readyProject.ID,
		AgentID:             agentItem.ID,
		Name:                "Coding Workflow",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      60,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     MustStatusBindingSet(statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(statusIDs["Done"]),
	})
	if err != nil {
		t.Fatalf("Create() workflow error = %v", err)
	}

	createdSkill, err := service.CreateSkill(ctx, CreateSkillInput{
		ProjectID:   readyProject.ID,
		Name:        "deploy-docker",
		Content:     "# Deploy Docker\n\nRun the Docker deployment flow.\n",
		Description: "Deploy Docker",
	})
	if err != nil {
		t.Fatalf("CreateSkill() error = %v", err)
	}
	if _, err := service.BindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: createdWorkflow.ID,
		Skills:     []string{createdSkill.Name},
	}); err != nil {
		t.Fatalf("BindSkills() error = %v", err)
	}

	if _, err := service.UnbindSkill(ctx, UpdateSkillBindingsInput{
		SkillID:     createdSkill.ID,
		WorkflowIDs: []uuid.UUID{createdWorkflow.ID},
	}); err != nil {
		t.Fatalf("UnbindSkill() error = %v", err)
	}

	harnessDoc, err := service.GetHarness(ctx, createdWorkflow.ID)
	if err != nil {
		t.Fatalf("GetHarness() error = %v", err)
	}
	skillNames, err := ParseHarnessSkills(harnessDoc.Content)
	if err != nil {
		t.Fatalf("ParseHarnessSkills() error = %v", err)
	}
	if len(skillNames) != 0 {
		t.Fatalf("ParseHarnessSkills() after unbind = %v, want empty", skillNames)
	}
}

func TestWorkflowServiceSkillAndReloadEdgeCases(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	created, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Coverage Workflow",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coverage\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       2,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := service.CreateSkill(ctx, CreateSkillInput{
		ProjectID: fixture.projectID,
		Name:      "skill-one",
		Content:   "# Skill One\nbody\n",
	}); err != nil {
		t.Fatalf("CreateSkill(skill-one) error = %v", err)
	}
	boundDoc, err := service.BindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: created.ID,
		Skills:     []string{"skill-one"},
	})
	if err != nil {
		t.Fatalf("BindSkills() error = %v", err)
	}
	if _, err := service.Update(ctx, UpdateInput{
		WorkflowID: created.ID,
		Hooks: Some(map[string]any{
			"workflow_hooks": map[string]any{
				"on_reload": []map[string]any{{
					"cmd":        "exit 9",
					"on_failure": "block",
				}},
			},
		}),
	}); err != nil {
		t.Fatalf("Update() hooks error = %v", err)
	}

	noopDoc, err := service.UnbindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: created.ID,
		Skills:     []string{"skill-missing"},
	})
	if err != nil {
		t.Fatalf("UnbindSkills(no-op) error = %v", err)
	}
	if noopDoc.Version != boundDoc.Version || noopDoc.Content != boundDoc.Content {
		t.Fatalf("UnbindSkills(no-op) = %+v, want version %d unchanged", noopDoc, boundDoc.Version)
	}

	if _, err := service.BindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: created.ID,
		Skills:     []string{"skill-missing"},
	}); !errors.Is(err, ErrSkillNotFound) {
		t.Fatalf("BindSkills(missing) error = %v, want %v", err, ErrSkillNotFound)
	}

	if _, err := service.HarvestSkills(ctx, HarvestSkillsInput{
		ProjectID:     fixture.projectID,
		WorkspaceRoot: t.TempDir(),
		AdapterType:   string(entagentprovider.AdapterTypeCodexAppServer),
	}); !errors.Is(err, ErrSkillInvalid) {
		t.Fatalf("HarvestSkills() error = %v, want %v", err, ErrSkillInvalid)
	}

	previousDoc, err := service.GetHarness(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetHarness() before blocked update error = %v", err)
	}
	if _, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: created.ID,
		Content:    strings.Replace(previousDoc.Content, "# Coverage", "# Blocked Reload", 1),
	}); !errors.Is(err, ErrWorkflowHookBlocked) {
		t.Fatalf("UpdateHarness(blocked) error = %v, want %v", err, ErrWorkflowHookBlocked)
	}
	afterReload, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() after blocked update error = %v", err)
	}
	if afterReload.Version != boundDoc.Version || afterReload.HarnessContent != previousDoc.Content {
		t.Fatalf("Get() after blocked update = %+v, want version %d and restored content", afterReload, boundDoc.Version)
	}
}

func findSkillByName(items []Skill, name string) *Skill {
	for i := range items {
		if items[i].Name == name {
			return &items[i]
		}
	}

	return nil
}

func TestWorkflowServiceErrorsAndRepoHelpers(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	if got := Some("value"); !got.Set || got.Value != "value" {
		t.Fatalf("Some() = %+v", got)
	}
	if service.RepoRoot() != repoRoot {
		t.Fatalf("RepoRoot() = %q, want %q", service.RepoRoot(), repoRoot)
	}

	childDir := filepath.Join(repoRoot, "nested", "child")
	if err := os.MkdirAll(childDir, 0o750); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	if detected, err := DetectRepoRoot(childDir); err != nil || detected != repoRoot {
		t.Fatalf("DetectRepoRoot() = %q, %v", detected, err)
	}
	readyPrerequisite, err := service.GetRepositoryPrerequisite(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("GetRepositoryPrerequisite() ready error = %v", err)
	}
	if !readyPrerequisite.Ready() || readyPrerequisite.Action != WorkflowRepositoryPrerequisiteActionNone {
		t.Fatalf("GetRepositoryPrerequisite() ready = %+v", readyPrerequisite)
	}

	missingPrerequisite, err := service.GetRepositoryPrerequisite(ctx, fixture.projectWithoutRepoID)
	if err != nil {
		t.Fatalf("GetRepositoryPrerequisite() missing repo error = %v", err)
	}
	if missingPrerequisite.Kind != WorkflowRepositoryPrerequisiteKindReady || missingPrerequisite.Action != WorkflowRepositoryPrerequisiteActionNone {
		t.Fatalf("GetRepositoryPrerequisite() repo-less project = %+v", missingPrerequisite)
	}

	if _, err := service.List(ctx, uuid.New()); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("List() missing project error = %v, want %v", err, ErrProjectNotFound)
	}
	if _, err := service.Get(ctx, uuid.New()); !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("Get() missing workflow error = %v, want %v", err, ErrWorkflowNotFound)
	}
	if _, err := service.GetHarness(ctx, uuid.New()); !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("GetHarness() missing workflow error = %v, want %v", err, ErrWorkflowNotFound)
	}
	if _, err := service.Update(ctx, UpdateInput{WorkflowID: uuid.New(), Name: Some("missing")}); !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("Update() missing workflow error = %v, want %v", err, ErrWorkflowNotFound)
	}
	if _, err := service.Delete(ctx, uuid.New()); !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("Delete() missing workflow error = %v, want %v", err, ErrWorkflowNotFound)
	}
	if _, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: uuid.New(),
		Content:    "---\nworkflow:\n  role: coding\n---\n\n# ok\n",
	}); !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("UpdateHarness() missing workflow error = %v, want %v", err, ErrWorkflowNotFound)
	}
	if _, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: uuid.New(),
		Content:    "---\nworkflow:\n  role: coding\n---\n\n{{",
	}); !errors.Is(err, ErrHarnessInvalid) {
		t.Fatalf("UpdateHarness() invalid content error = %v, want %v", err, ErrHarnessInvalid)
	}

	if _, err := service.storageForProject(ctx, fixture.projectWithoutRepoID, workflowStorageUsageRead); err != nil {
		t.Fatalf("storageForProject() repo-less project error = %v", err)
	}

	if _, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             uuid.New(),
		Name:                "Bad Agent",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	}); !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("Create() missing agent error = %v, want %v", err, ErrAgentNotFound)
	}

	if _, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Bad Status",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(uuid.New()),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	}); !errors.Is(err, ErrStatusNotFound) {
		t.Fatalf("Create() missing status error = %v, want %v", err, ErrStatusNotFound)
	}

	if _, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Bad Hooks",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		Hooks:               map[string]any{"workflow_hooks": "bad"},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	}); !errors.Is(err, ErrHookConfigInvalid) {
		t.Fatalf("Create() invalid hooks error = %v, want %v", err, ErrHookConfigInvalid)
	}

	badHarnessPath := "../escape.md"
	if _, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Bad Harness Path",
		Type:                entworkflow.TypeCoding,
		HarnessPath:         &badHarnessPath,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	}); !errors.Is(err, ErrHarnessInvalid) {
		t.Fatalf("Create() invalid harness path error = %v, want %v", err, ErrHarnessInvalid)
	}

	created, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Duplicate Workflow",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	})
	if err != nil {
		t.Fatalf("Create() duplicate baseline error = %v", err)
	}

	if _, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Duplicate Workflow",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	}); !errors.Is(err, ErrWorkflowConflict) {
		t.Fatalf("Create() duplicate name error = %v, want %v", err, ErrWorkflowConflict)
	}

	if _, err := service.Update(ctx, UpdateInput{
		WorkflowID:  created.ID,
		HarnessPath: Some("../escape.md"),
	}); !errors.Is(err, ErrHarnessInvalid) {
		t.Fatalf("Update() invalid harness path error = %v, want %v", err, ErrHarnessInvalid)
	}

	if got := service.mapWorkflowReadError("get workflow", errors.New("boom")); got == nil || !errors.Is(got, errors.New("boom")) {
		if got == nil || got.Error() != "get workflow: boom" {
			t.Fatalf("mapWorkflowReadError() = %v", got)
		}
	}
	if got := service.mapWorkflowWriteError("delete workflow", errors.New("tickets still reference workflow")); !errors.Is(got, ErrWorkflowInUse) {
		t.Fatalf("mapWorkflowWriteError(tickets) = %v, want %v", got, ErrWorkflowInUse)
	}
	if got := service.mapWorkflowWriteError("delete workflow", errors.New("scheduled_jobs still reference workflow")); !errors.Is(got, ErrWorkflowInUse) {
		t.Fatalf("mapWorkflowWriteError(scheduled_jobs) = %v, want %v", got, ErrWorkflowInUse)
	}

	if _, err := validateConfiguredHooks(map[string]any{"workflow_hooks": "bad"}); !errors.Is(err, ErrHookConfigInvalid) {
		t.Fatalf("validateConfiguredHooks() error = %v, want %v", err, ErrHookConfigInvalid)
	}

	rollback(nil)
}

func TestWorkflowServiceUpdateHarnessRegistryFailurePaths(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	created, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Harness Failure Workflow",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Initial\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: created.ID,
		Content:    "invalid harness",
	}); !errors.Is(err, ErrHarnessInvalid) {
		t.Fatalf("UpdateHarness(invalid) error = %v, want %v", err, ErrHarnessInvalid)
	}

	if _, err := service.Update(ctx, UpdateInput{
		WorkflowID: created.ID,
		IsActive:   Some(true),
		Hooks: Some(map[string]any{
			"workflow_hooks": map[string]any{
				"on_reload": []map[string]any{{
					"cmd":        "exit 7",
					"on_failure": "block",
				}},
			},
		}),
	}); err != nil {
		t.Fatalf("Update(active hooks) error = %v", err)
	}

	if _, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: created.ID,
		Content:    "---\nworkflow:\n  role: coding\n---\n\n# Write failure\n",
	}); !errors.Is(err, ErrWorkflowHookBlocked) {
		t.Fatalf("UpdateHarness(blocked) error = %v, want %v", err, ErrWorkflowHookBlocked)
	}

	currentVersion, err := service.currentWorkflowVersion(ctx, created.ID)
	if err != nil {
		t.Fatalf("currentWorkflowVersion() error = %v", err)
	}
	if !strings.Contains(currentVersion.ContentMarkdown, "# Initial") {
		t.Fatalf("current workflow version changed after blocked update: %q", currentVersion.ContentMarkdown)
	}
}

func TestWorkflowServiceBuildHarnessTemplateDataProjectContext(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	if _, err := client.Agent.UpdateOneID(fixture.agentID).
		SetTotalTicketsCompleted(47).
		Save(ctx); err != nil {
		t.Fatalf("seed completed ticket count: %v", err)
	}

	templateContent := `---
workflow:
  role_name: "fullstack-developer"
status:
  pickup: "Todo"
  finish: "Done"
skills:
  - openase-platform
  - commit
---
Implement product changes end to end.

Ticket {{ ticket.identifier }} {{ ticket.title | markdown_escape }}
Status {{ ticket.status }} parent={{ ticket.parent_identifier }} attempts={{ attempt }}/{{ max_attempts }}
Links {{ ticket.links | length }} {{ ticket.links[0].type }} {{ ticket.links[0].relation }}
Deps {% for dep in ticket.dependencies %}{{ dep.identifier }}:{{ dep.type }}:{{ dep.status }}{% endfor %}
Repos {% for repo in repos %}{{ repo.name }}@{{ repo.branch }} labels={{ repo.labels | join(",") }} path={{ repo.path }}{% endfor %}
All {{ all_repos | map(attribute="name") | join(",") }}
Agent {{ agent.name }} {{ agent.provider }} {{ agent.adapter_type }} {{ agent.model }} {{ agent.total_tickets_completed }}
Machine {{ machine.name }} {{ accessible_machines[0].ssh_user }}
Workflow {{ workflow.name }} {{ workflow.type }} {{ workflow.role_name }} {{ workflow.pickup_status }} {{ workflow.finish_status }}
ProjectWorkflows {% for wf in project.workflows %}{{ wf.role_name }}:{{ wf.pickup_status }}:{{ wf.current_active }}/{{ wf.max_concurrent }}:{{ wf.role_description }}|{% endfor %}
WorkflowArtifacts {% for wf in project.workflows %}{{ wf.role_name }}={{ wf.finish_status }}:{{ wf.harness_path }}:{{ wf.skills | join(",") }}|{% endfor %}
WorkflowHistory {% for wf in project.workflows %}{{ wf.role_name }}={% for recent in wf.recent_tickets %}{{ recent.identifier }}:{{ recent.status }}:{{ recent.retry_paused }}:{{ recent.consecutive_errors }}|{% endfor %};{% endfor %}
ProjectStatuses {{ project.statuses | map(attribute="name") | join(",") }} first={{ project.statuses[0].color }}
ProjectMachines {% for machine in project.machines %}{{ machine.name }}:{{ machine.status }}:{{ machine.labels | join(",") }}|{% endfor %}
Platform {{ platform.api_url }} {{ platform.project_id }} {{ platform.ticket_id }}
Timestamp {{ timestamp }} Version {{ openase_version }} URL {{ ticket.url }}
{% if attempt > 1 %}retry{% endif %}
`

	codingWorkflow, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Coding Workflow",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      templateContent,
		Hooks:               map[string]any{},
		MaxConcurrent:       3,
		MaxRetryAttempts:    3,
		TimeoutMinutes:      60,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	})
	if err != nil {
		t.Fatalf("create coding workflow: %v", err)
	}
	if _, err := service.BindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: codingWorkflow.ID,
		Skills:     []string{"openase-platform", "commit"},
	}); err != nil {
		t.Fatalf("bind default skills: %v", err)
	}
	if _, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Dispatcher Workflow",
		Type:                entworkflow.TypeCustom,
		HarnessContent:      "---\nworkflow:\n  role: dispatcher\nstatus:\n  pickup: \"Backlog\"\n  finish: \"Backlog\"\n---\n\nEvaluate backlog tickets and route them to the right workflow.\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      5,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Backlog"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Backlog"]),
	}); err != nil {
		t.Fatalf("create dispatcher workflow: %v", err)
	}

	primaryRepo, err := client.ProjectRepo.Query().
		Where(
			entprojectrepo.ProjectIDEQ(fixture.projectID),
			entprojectrepo.IsPrimary(true),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load primary repo: %v", err)
	}
	if _, err := client.ProjectRepo.UpdateOneID(primaryRepo.ID).
		SetName("backend").
		SetDefaultBranch("main").
		SetWorkspaceDirname("backend").
		Save(ctx); err != nil {
		t.Fatalf("normalize primary repo metadata: %v", err)
	}
	frontendRepo, err := client.ProjectRepo.Create().
		SetProjectID(fixture.projectID).
		SetName("frontend").
		SetRepositoryURL("https://github.com/acme/frontend").
		SetDefaultBranch("develop").
		SetWorkspaceDirname("frontend").
		Save(ctx)
	if err != nil {
		t.Fatalf("create frontend repo: %v", err)
	}

	activeTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-40").
		SetTitle("Implement auth boundary parsing").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityMedium).
		SetType(entticket.TypeFeature).
		SetCreatedBy("user:gary").
		SetWorkflowID(codingWorkflow.ID).
		SetCreatedAt(time.Date(2026, 3, 20, 7, 0, 0, 0, time.UTC)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create active workflow ticket: %v", err)
	}
	activeRun, err := client.AgentRun.Create().
		SetAgentID(fixture.agentID).
		SetWorkflowID(codingWorkflow.ID).
		SetTicketID(activeTicket.ID).
		SetProviderID(fixture.providerID).
		SetStatus(entagentrun.StatusExecuting).
		Save(ctx)
	if err != nil {
		t.Fatalf("create active workflow run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(activeTicket.ID).
		SetCurrentRunID(activeRun.ID).
		Save(ctx); err != nil {
		t.Fatalf("attach active workflow run: %v", err)
	}
	if _, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-41").
		SetTitle("Tighten auth retry guidance").
		SetStatusID(fixture.statusIDs["In Review"]).
		SetPriority(entticket.PriorityHigh).
		SetType(entticket.TypeBugfix).
		SetCreatedBy("user:gary").
		SetWorkflowID(codingWorkflow.ID).
		SetAttemptCount(3).
		SetConsecutiveErrors(2).
		SetRetryPaused(true).
		SetPauseReason("needs_human_review").
		SetStartedAt(time.Date(2026, 3, 20, 8, 0, 0, 0, time.UTC)).
		SetCreatedAt(time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)).
		Save(ctx); err != nil {
		t.Fatalf("create paused workflow history ticket: %v", err)
	}

	parentTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-30").
		SetTitle("Parent ticket").
		SetStatusID(fixture.statusIDs["Done"]).
		SetPriority(entticket.PriorityMedium).
		SetCreatedBy("user:gary").
		Save(ctx)
	if err != nil {
		t.Fatalf("create parent ticket: %v", err)
	}
	dependencyTarget, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-31").
		SetTitle("Dependency ticket").
		SetStatusID(fixture.statusIDs["Done"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:gary").
		Save(ctx)
	if err != nil {
		t.Fatalf("create dependency target: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-42").
		SetTitle("Escape * markdown").
		SetDescription("Render the harness body").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetType(entticket.TypeBugfix).
		SetCreatedBy("user:gary").
		SetParentTicketID(parentTicket.ID).
		SetWorkflowID(codingWorkflow.ID).
		SetAttemptCount(2).
		SetBudgetUsd(5.0).
		SetExternalRef("BetterAndBetterII/openase#20").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(primaryRepo.ID).
		SetBranchName("agent/codex/ASE-42").
		SetIsPrimaryScope(true).
		Save(ctx); err != nil {
		t.Fatalf("create primary ticket repo scope: %v", err)
	}
	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(frontendRepo.ID).
		SetBranchName("agent/codex/frontend-ASE-42").
		Save(ctx); err != nil {
		t.Fatalf("create secondary ticket repo scope: %v", err)
	}
	if _, err := client.TicketExternalLink.Create().
		SetTicketID(ticketItem.ID).
		SetLinkType("github_issue").
		SetURL("https://github.com/acme/backend/issues/42").
		SetExternalID("42").
		SetTitle("Login validation broken on Safari").
		SetStatus("open").
		SetRelation("resolves").
		Save(ctx); err != nil {
		t.Fatalf("create external link: %v", err)
	}
	if _, err := client.TicketDependency.Create().
		SetSourceTicketID(ticketItem.ID).
		SetTargetTicketID(dependencyTarget.ID).
		SetType(entticketdependency.TypeBlocks).
		Save(ctx); err != nil {
		t.Fatalf("create dependency: %v", err)
	}

	data, err := service.BuildHarnessTemplateData(ctx, BuildHarnessTemplateDataInput{
		WorkflowID:     codingWorkflow.ID,
		TicketID:       ticketItem.ID,
		AgentID:        &fixture.agentID,
		Workspace:      "/workspaces/ASE-42",
		Timestamp:      time.Date(2026, 3, 20, 10, 30, 0, 0, time.UTC),
		OpenASEVersion: "0.3.1",
		TicketURL:      "http://localhost:19836/tickets/ASE-42",
		Platform: HarnessPlatformData{
			APIURL:     "http://localhost:19836/api/v1",
			AgentToken: "ase_agent_token",
		},
		Machine: HarnessMachineData{
			Name:          "gpu-01",
			Host:          "10.0.1.10",
			Description:   "NVIDIA A100 x4",
			Labels:        []string{"gpu", "a100"},
			WorkspaceRoot: "/workspaces",
		},
		AccessibleMachines: []HarnessAccessibleMachineData{
			{
				Name:        "storage",
				Host:        "10.0.1.20",
				Description: "Artifact storage",
				Labels:      []string{"storage", "nfs"},
				SSHUser:     "openase",
			},
			{
				Name:        "gpu-01",
				Host:        "10.0.1.10",
				Description: "duplicate current machine",
				Labels:      []string{"gpu", "a100"},
				SSHUser:     "openase",
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildHarnessTemplateData() error = %v", err)
	}

	if data.Attempt != 2 || data.MaxAttempts != 3 || data.Timestamp != "2026-03-20T10:30:00Z" {
		t.Fatalf("harness attempt context = %+v", data)
	}
	if data.Ticket.ParentIdentifier != "ASE-30" || len(data.Ticket.Links) != 1 || len(data.Ticket.Dependencies) != 1 {
		t.Fatalf("ticket harness data = %+v", data.Ticket)
	}
	if data.Ticket.Links[0].Type != "github_issue" || data.Ticket.Links[0].Relation != "resolves" {
		t.Fatalf("ticket links = %+v", data.Ticket.Links)
	}
	if data.Ticket.Dependencies[0].Identifier != "ASE-31" || data.Ticket.Dependencies[0].Status != "Done" {
		t.Fatalf("ticket dependencies = %+v", data.Ticket.Dependencies)
	}
	if data.Agent.Name != "codex-coding" || data.Agent.Provider != "Codex" || data.Agent.Model != "gpt-5.4" || data.Agent.TotalTicketsCompleted != 47 {
		t.Fatalf("agent harness data = %+v", data.Agent)
	}
	if len(data.Repos) != 2 || len(data.AllRepos) != 2 {
		t.Fatalf("repo harness data = %+v %+v", data.Repos, data.AllRepos)
	}

	var backendRepoData, frontendRepoData *HarnessRepoData
	for index := range data.AllRepos {
		switch data.AllRepos[index].Name {
		case "backend":
			backendRepoData = &data.AllRepos[index]
		case "frontend":
			frontendRepoData = &data.AllRepos[index]
		}
	}
	if backendRepoData == nil || backendRepoData.Branch != "agent/codex/ASE-42" || backendRepoData.Path != "backend" {
		t.Fatalf("backend repo data = %+v", backendRepoData)
	}
	if frontendRepoData == nil || frontendRepoData.Branch != "agent/codex/frontend-ASE-42" || frontendRepoData.Path != "frontend" {
		t.Fatalf("frontend repo data = %+v", frontendRepoData)
	}

	if len(data.Project.Workflows) != 2 {
		t.Fatalf("project workflows = %+v", data.Project.Workflows)
	}
	var codingContext, dispatcherContext *HarnessProjectWorkflowData
	for index := range data.Project.Workflows {
		switch data.Project.Workflows[index].RoleName {
		case "fullstack-developer":
			codingContext = &data.Project.Workflows[index]
		case "dispatcher":
			dispatcherContext = &data.Project.Workflows[index]
		}
	}
	if codingContext == nil || codingContext.CurrentActive != 1 || !slices.Equal(codingContext.Skills, []string{"commit", "openase-platform"}) || len(codingContext.RecentTickets) != 3 {
		t.Fatalf("coding workflow context = %+v", codingContext)
	}
	if codingContext.RecentTickets[0].Identifier != "ASE-42" || codingContext.RecentTickets[1].Identifier != "ASE-41" || !codingContext.RecentTickets[1].RetryPaused || codingContext.RecentTickets[2].Identifier != "ASE-40" {
		t.Fatalf("coding workflow history = %+v", codingContext.RecentTickets)
	}
	if dispatcherContext == nil || dispatcherContext.CurrentActive != 0 || dispatcherContext.MaxConcurrent != 1 || len(dispatcherContext.RecentTickets) != 0 {
		t.Fatalf("dispatcher workflow context = %+v", dispatcherContext)
	}

	if len(data.Project.Statuses) == 0 || data.Project.Statuses[0].Color == "" {
		t.Fatalf("project statuses = %+v", data.Project.Statuses)
	}
	if len(data.Project.Machines) != 2 || data.Project.Machines[0].Name != "gpu-01" || data.Project.Machines[0].Status != "current" || data.Project.Machines[1].Name != "storage" {
		t.Fatalf("project machines = %+v", data.Project.Machines)
	}
	if data.Platform.ProjectID != fixture.projectID.String() || data.Platform.TicketID != ticketItem.ID.String() {
		t.Fatalf("platform data = %+v", data.Platform)
	}

	rendered, err := RenderHarnessBody(templateContent, data)
	if err != nil {
		t.Fatalf("RenderHarnessBody() error = %v", err)
	}
	for _, want := range []string{
		`Ticket ASE-42 Escape \* markdown`,
		"Status Todo parent=ASE-30 attempts=2/3",
		"Deps ASE-31:blocks:Done",
		"Agent codex-coding Codex codex-app-server gpt-5.4 47",
		"ProjectWorkflows fullstack-developer:Todo:1/3:Implement product changes end to end.|dispatcher:Backlog:0/1:Evaluate backlog tickets and route them to the right workflow.|",
		"WorkflowHistory fullstack-developer=ASE-42:Todo:False:0|ASE-41:In Review:True:2|ASE-40:Todo:False:0|;dispatcher=;",
		"ProjectMachines gpu-01:current:gpu,a100|storage:accessible:storage,nfs|",
		fmt.Sprintf("Platform http://localhost:19836/api/v1 %s %s", fixture.projectID, ticketItem.ID),
		"retry",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered harness to contain %q, got:\n%s", want, rendered)
		}
	}
}

type workflowServiceFixture struct {
	projectID            uuid.UUID
	projectWithoutRepoID uuid.UUID
	agentID              uuid.UUID
	providerID           uuid.UUID
	statusIDs            map[string]uuid.UUID
}

func seedWorkflowServiceFixture(ctx context.Context, t *testing.T, client *ent.Client, repoRoot string) workflowServiceFixture {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetStatus("In Progress").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	projectWithoutRepo, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Repo Less").
		SetSlug("repo-less").
		SetStatus("In Progress").
		Save(ctx)
	if err != nil {
		t.Fatalf("create projectWithoutRepo: %v", err)
	}
	if _, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName(filepath.Base(repoRoot)).
		SetRepositoryURL(repoRoot).
		SetDefaultBranch("main").
		SetWorkspaceDirname(filepath.Base(repoRoot)).
		SetIsPrimary(true).
		Save(ctx); err != nil {
		t.Fatalf("create primary project repo: %v", err)
	}

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	statusIDs := make(map[string]uuid.UUID, len(statuses))
	for _, status := range statuses {
		statusIDs[status.Name] = status.ID
	}

	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(providerItem.ID).
		SetName("codex-coding").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	return workflowServiceFixture{
		projectID:            project.ID,
		projectWithoutRepoID: projectWithoutRepo.ID,
		agentID:              agentItem.ID,
		providerID:           providerItem.ID,
		statusIDs:            statusIDs,
	}
}

func newWorkflowTestService(t *testing.T, client *ent.Client, repoRoot string) *Service {
	t.Helper()

	service, err := NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	t.Cleanup(func() {
		if err := service.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	return service
}

func openWorkflowTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}

func findSkillIDByName(ctx context.Context, t *testing.T, client *ent.Client, projectID uuid.UUID, name string) uuid.UUID {
	t.Helper()

	item, err := client.Skill.Query().
		Where(
			entskill.ProjectIDEQ(projectID),
			entskill.NameEQ(name),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("find skill %s: %v", name, err)
	}
	return item.ID
}

func findRuntimeSkillSnapshot(t *testing.T, items []RuntimeSkillSnapshot, name string) RuntimeSkillSnapshot {
	t.Helper()

	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("runtime skill %s not found in %+v", name, items)
	return RuntimeSkillSnapshot{}
}

func skillVersionIDs(items []RuntimeSkillSnapshot) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.VersionID)
	}
	return ids
}

func createWorkflowTestGitRepo(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	repository, err := git.PlainInit(repoRoot, false)
	if err != nil {
		t.Fatalf("git init repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("workflow repo\n"), 0o600); err != nil {
		t.Fatalf("write repo seed file: %v", err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("load repo worktree: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("git add repo seed file: %v", err)
	}
	if _, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@openai.com",
			When:  time.Date(2026, 3, 31, 3, 0, 0, 0, time.UTC),
		},
	}); err != nil {
		t.Fatalf("git commit repo seed file: %v", err)
	}

	return repoRoot
}

func createWorkflowSourceRepository(t *testing.T) string {
	t.Helper()

	repoPath := filepath.Join(t.TempDir(), "source")
	repository, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("git init source repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("workflow source\n"), 0o600); err != nil {
		t.Fatalf("write source repo file: %v", err)
	}

	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("load source repo worktree: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("git add source repo: %v", err)
	}
	if _, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@openai.com",
			When:  time.Date(2026, 3, 29, 18, 0, 0, 0, time.UTC),
		},
	}); err != nil {
		t.Fatalf("git commit source repo: %v", err)
	}
	if _, err := repository.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{repoPath},
	}); err != nil {
		t.Fatalf("create source repo origin: %v", err)
	}

	return repoPath
}

func commitWorkflowSourceFile(t *testing.T, repoPath string, relativePath string, content string) {
	t.Helper()

	absolutePath := filepath.Join(repoPath, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
		t.Fatalf("create source repo parent: %v", err)
	}
	if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
		t.Fatalf("write source repo file: %v", err)
	}

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open source repo: %v", err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("load source repo worktree: %v", err)
	}
	if _, err := worktree.Add(filepath.ToSlash(relativePath)); err != nil {
		t.Fatalf("git add source repo file: %v", err)
	}
	if _, err := worktree.Commit("update workflow source", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@openai.com",
			When:  time.Now().UTC(),
		},
	}); err != nil {
		t.Fatalf("git commit source repo file: %v", err)
	}
}
func mustReadWorkflowFile(t *testing.T, path string) string {
	t.Helper()

	//nolint:gosec // Tests only read files created in isolated temp/project directories.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}

	return string(data)
}

func readWorkflowGitCommitMessages(t *testing.T, repoRoot string) []string {
	t.Helper()

	repository, err := git.PlainOpen(repoRoot)
	if err != nil {
		t.Fatalf("open workflow git repo: %v", err)
	}
	head, err := repository.Head()
	if err != nil {
		t.Fatalf("load workflow git head: %v", err)
	}
	iterator, err := repository.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		t.Fatalf("read workflow git log: %v", err)
	}

	messages := []string{}
	if err := iterator.ForEach(func(commit *object.Commit) error {
		messages = append(messages, commit.Message)
		return nil
	}); err != nil {
		t.Fatalf("iterate workflow git log: %v", err)
	}
	return messages
}

func assertWorkflowGitCommitMessage(t *testing.T, messages []string, want string) {
	t.Helper()

	for _, message := range messages {
		if strings.TrimSpace(message) == want {
			return
		}
	}
	t.Fatalf("git commit message %q not found in %+v", want, messages)
}

func waitForWorkflowVersion(ctx context.Context, t *testing.T, client *ent.Client, workflowID uuid.UUID, wantVersion int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		item, err := client.Workflow.Get(ctx, workflowID)
		if err == nil && item.Version == wantVersion {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	item, err := client.Workflow.Get(ctx, workflowID)
	if err != nil {
		t.Fatalf("load workflow version: %v", err)
	}
	t.Fatalf("workflow version = %d, want %d", item.Version, wantVersion)
}

func waitForWorkflowFileContent(t *testing.T, path string, want string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		//nolint:gosec // Tests only read files created in isolated temp/project directories.
		data, err := os.ReadFile(path)
		if err == nil && string(data) == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	if got := mustReadWorkflowFile(t, path); got != want {
		t.Fatalf("workflow file %s = %q, want %q", path, got, want)
	}
}
