package workflow

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entprojectrepomirror "github.com/BetterAndBetterII/openase/ent/projectrepomirror"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	projectrepomirrorsvc "github.com/BetterAndBetterII/openase/internal/projectrepomirror"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/fsnotify/fsnotify"
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
	if got := mustReadWorkflowFile(t, filepath.Join(repoRoot, filepath.FromSlash(created.HarnessPath))); got != created.HarnessContent {
		t.Fatalf("created harness content = %q, want %q", got, created.HarnessContent)
	}

	if len(service.storages) != 1 {
		t.Fatalf("service.storages = %d, want 1", len(service.storages))
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

	mustWriteSkill(t, filepath.Join(repoRoot, ".openase", "skills", "skill-one"), "# Skill One\nbody")
	mustWriteSkill(t, filepath.Join(repoRoot, ".openase", "skills", "skill-two"), "# Skill Two\nbody")

	boundDoc, err := service.BindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: created.ID,
		Skills:     []string{"skill-one", "skill-two"},
	})
	if err != nil {
		t.Fatalf("BindSkills() error = %v", err)
	}
	if boundDoc.Version != 2 {
		t.Fatalf("BindSkills() version = %d, want 2", boundDoc.Version)
	}
	if got := mustReadWorkflowFile(t, reloadMarkerPath); got != "on_reload:2" {
		t.Fatalf("reload marker after BindSkills() = %q", got)
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
	refreshResult, err := service.RefreshSkills(ctx, RefreshSkillsInput{
		ProjectID:     fixture.projectID,
		WorkspaceRoot: workspaceRoot,
		AdapterType:   string(entagentprovider.AdapterTypeCodexAppServer),
	})
	if err != nil {
		t.Fatalf("RefreshSkills() error = %v", err)
	}
	if !slices.Contains(refreshResult.InjectedSkills, "skill-one") || !slices.Contains(refreshResult.InjectedSkills, "skill-two") {
		t.Fatalf("RefreshSkills() = %+v", refreshResult)
	}

	workspaceSkillsRoot := filepath.Join(workspaceRoot, ".codex", "skills")
	mustWriteSkill(t, filepath.Join(workspaceSkillsRoot, "skill-one"), "# Skill One Updated\nbody")
	mustWriteSkill(t, filepath.Join(workspaceSkillsRoot, "skill-three"), "# Skill Three\nbody")

	harvestResult, err := service.HarvestSkills(ctx, HarvestSkillsInput{
		ProjectID:     fixture.projectID,
		WorkspaceRoot: workspaceRoot,
		AdapterType:   string(entagentprovider.AdapterTypeCodexAppServer),
	})
	if err != nil {
		t.Fatalf("HarvestSkills() error = %v", err)
	}
	if !slices.Equal(harvestResult.HarvestedSkills, []string{"skill-three"}) || !slices.Equal(harvestResult.UpdatedSkills, []string{"skill-one"}) {
		t.Fatalf("HarvestSkills() = %+v", harvestResult)
	}

	unboundDoc, err := service.UnbindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: created.ID,
		Skills:     []string{"skill-two"},
	})
	if err != nil {
		t.Fatalf("UnbindSkills() error = %v", err)
	}
	if unboundDoc.Version != 3 {
		t.Fatalf("UnbindSkills() version = %d, want 3", unboundDoc.Version)
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
	if _, err := os.Stat(filepath.Join(repoRoot, ".openase", "harnesses", "coding-workflow.md")); !os.IsNotExist(err) {
		t.Fatalf("old harness path still exists, stat err = %v", err)
	}

	updatedHarnessContent := "---\nworkflow:\n  role: coding\nskills:\n  - skill-one\n---\n\n# Updated by API\n"
	updatedHarnessDoc, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: created.ID,
		Content:    updatedHarnessContent,
	})
	if err != nil {
		t.Fatalf("UpdateHarness() error = %v", err)
	}
	if updatedHarnessDoc.Version != 4 || updatedHarnessDoc.Content != updatedHarnessContent {
		t.Fatalf("UpdateHarness() = %+v", updatedHarnessDoc)
	}
	if got := mustReadWorkflowFile(t, reloadMarkerPath); got != "on_reload:4" {
		t.Fatalf("reload marker after UpdateHarness() = %q", got)
	}

	storage, err := service.storageForProject(ctx, fixture.projectID, workflowStorageUsageRead)
	if err != nil {
		t.Fatalf("storageForProject() error = %v", err)
	}
	externalContent := "---\nworkflow:\n  role: coding\nskills:\n  - skill-one\n---\n\n# Updated on disk\n"
	harnessAbsPath := storage.registry.absolutePath(updatedHarnessDoc.Path)
	if err := os.WriteFile(harnessAbsPath, []byte(externalContent), 0o600); err != nil {
		t.Fatalf("write external harness: %v", err)
	}
	storage.registry.handleEvent(fsnotify.Event{Name: harnessAbsPath, Op: fsnotify.Write})
	waitForWorkflowVersion(ctx, t, client, created.ID, 5)
	if got := mustReadWorkflowFile(t, reloadMarkerPath); got != "on_reload:5" {
		t.Fatalf("reload marker after external write = %q", got)
	}

	afterExternal, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() after external write error = %v", err)
	}
	if afterExternal.Version != 5 || afterExternal.HarnessContent != externalContent {
		t.Fatalf("Get() after external write = %+v", afterExternal)
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
	if _, err := os.Stat(harnessAbsPath); !os.IsNotExist(err) {
		t.Fatalf("deleted harness still exists, stat err = %v", err)
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

func TestWorkflowServiceUsesMirrorFreshnessPolicyForReadAndWritePaths(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	sourceRepoPath := createWorkflowSourceRepository(t)
	mirrorPath := filepath.Join(t.TempDir(), "mirror")
	service := newWorkflowTestService(t, client, mirrorPath)
	fixture := seedWorkflowServiceFixture(ctx, t, client, mirrorPath)

	projectRepo, err := client.ProjectRepo.Query().
		Where(entprojectrepo.ProjectIDEQ(fixture.projectID), entprojectrepo.IsPrimary(true)).
		Only(ctx)
	if err != nil {
		t.Fatalf("load primary repo: %v", err)
	}
	projectItem, err := client.Project.Get(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	localMachine, err := client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(projectItem.OrganizationID),
			entmachine.NameEQ(catalogdomain.LocalMachineName),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load local machine: %v", err)
	}
	if _, err := client.ProjectRepo.UpdateOneID(projectRepo.ID).
		SetRepositoryURL(sourceRepoPath).
		SetDefaultBranch("master").
		Save(ctx); err != nil {
		t.Fatalf("update primary repo remote: %v", err)
	}

	mirrorService := projectrepomirrorsvc.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.ConfigureMirrorService(mirrorService)
	if _, err := mirrorService.Prepare(ctx, projectrepomirrorsvc.PrepareInput{
		ProjectRepoID: projectRepo.ID,
		MachineID:     localMachine.ID,
		LocalPath:     mirrorPath,
	}); err != nil {
		t.Fatalf("prepare mirror: %v", err)
	}
	if _, err := client.ProjectRepoMirror.Update().
		Where(
			entprojectrepomirror.ProjectRepoIDEQ(projectRepo.ID),
			entprojectrepomirror.MachineIDEQ(localMachine.ID),
		).
		ClearLastSyncedAt().
		ClearLastVerifiedAt().
		SetState(entprojectrepomirror.StateReady).
		Save(ctx); err != nil {
		t.Fatalf("clear mirror timestamps: %v", err)
	}

	created, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Freshness Workflow",
		Type:                entworkflow.TypeCoding,
		HarnessContent:      "---\nworkflow:\n  role: coding\n---\n\n# Freshness\n",
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

	mirrorAfterCreate, err := client.ProjectRepoMirror.Query().
		Where(
			entprojectrepomirror.ProjectRepoIDEQ(projectRepo.ID),
			entprojectrepomirror.MachineIDEQ(localMachine.ID),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load mirror after create: %v", err)
	}
	if mirrorAfterCreate.LastSyncedAt == nil || mirrorAfterCreate.LastVerifiedAt == nil {
		t.Fatalf("expected create to sync mirror, got %+v", mirrorAfterCreate)
	}
	syncedAfterCreate := *mirrorAfterCreate.LastSyncedAt
	verifiedAfterCreate := *mirrorAfterCreate.LastVerifiedAt

	time.Sleep(20 * time.Millisecond)

	if _, err := service.GetHarness(ctx, created.ID); err != nil {
		t.Fatalf("GetHarness() error = %v", err)
	}

	mirrorAfterRead, err := client.ProjectRepoMirror.Query().
		Where(
			entprojectrepomirror.ProjectRepoIDEQ(projectRepo.ID),
			entprojectrepomirror.MachineIDEQ(localMachine.ID),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load mirror after read: %v", err)
	}
	if mirrorAfterRead.LastSyncedAt == nil || !mirrorAfterRead.LastSyncedAt.Equal(syncedAfterCreate) {
		t.Fatalf("expected read path to avoid sync, got %+v", mirrorAfterRead)
	}
	if mirrorAfterRead.LastVerifiedAt == nil || !mirrorAfterRead.LastVerifiedAt.After(verifiedAfterCreate) {
		t.Fatalf("expected read path to verify mirror, got %+v", mirrorAfterRead)
	}

	commitWorkflowSourceFile(t, sourceRepoPath, created.HarnessPath, created.HarnessContent)
	time.Sleep(20 * time.Millisecond)

	if _, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: created.ID,
		Content:    "---\nworkflow:\n  role: coding\n---\n\n# Freshness updated\n",
	}); err != nil {
		t.Fatalf("UpdateHarness() error = %v", err)
	}

	mirrorAfterWrite, err := client.ProjectRepoMirror.Query().
		Where(
			entprojectrepomirror.ProjectRepoIDEQ(projectRepo.ID),
			entprojectrepomirror.MachineIDEQ(localMachine.ID),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("load mirror after write: %v", err)
	}
	if mirrorAfterWrite.LastSyncedAt == nil || !mirrorAfterWrite.LastSyncedAt.After(syncedAfterCreate) {
		t.Fatalf("expected write path to sync mirror, got %+v", mirrorAfterWrite)
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

	mustWriteSkill(t, filepath.Join(repoRoot, ".openase", "skills", "skill-one"), "# Skill One\nbody")
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

	if err := os.RemoveAll(filepath.Join(repoRoot, ".openase", "skills", "skill-one")); err != nil {
		t.Fatalf("remove skill-one: %v", err)
	}
	skills, err := service.ListSkills(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("ListSkills() after removing bound skill error = %v", err)
	}
	skillOne := findSkillByName(skills, "skill-one")
	if skillOne == nil || skillOne.Description != "" || len(skillOne.BoundWorkflows) != 1 || skillOne.BoundWorkflows[0].ID != created.ID {
		t.Fatalf("ListSkills() after removing bound skill = %+v", skills)
	}

	harvestResult, err := service.HarvestSkills(ctx, HarvestSkillsInput{
		ProjectID:     fixture.projectID,
		WorkspaceRoot: t.TempDir(),
		AdapterType:   string(entagentprovider.AdapterTypeCodexAppServer),
	})
	if err != nil {
		t.Fatalf("HarvestSkills() empty workspace error = %v", err)
	}
	if harvestResult.SkillsDir == "" || len(harvestResult.HarvestedSkills) != 0 || len(harvestResult.UpdatedSkills) != 0 {
		t.Fatalf("HarvestSkills() empty workspace = %+v", harvestResult)
	}

	storage, err := service.storageForProject(ctx, fixture.projectID, workflowStorageUsageRead)
	if err != nil {
		t.Fatalf("storageForProject() error = %v", err)
	}
	harnessAbsPath := storage.registry.absolutePath(created.HarnessPath)
	previousContent := mustReadWorkflowFile(t, harnessAbsPath)
	blockedContent := strings.Replace(previousContent, "# Coverage", "# Blocked Reload", 1)
	if err := os.WriteFile(harnessAbsPath, []byte(blockedContent), 0o600); err != nil {
		t.Fatalf("write blocked harness content: %v", err)
	}
	storage.registry.handleEvent(fsnotify.Event{Name: harnessAbsPath, Op: fsnotify.Write})

	afterReload, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() after blocked reload error = %v", err)
	}
	if afterReload.Version != boundDoc.Version || afterReload.HarnessContent != previousContent {
		t.Fatalf("Get() after blocked reload = %+v, want version %d and restored content", afterReload, boundDoc.Version)
	}
	if got := mustReadWorkflowFile(t, harnessAbsPath); got != previousContent {
		t.Fatalf("harness file after blocked reload = %q, want %q", got, previousContent)
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
	if detected, err := ResolveReadyMirrorRepoRoot([]*ent.ProjectRepoMirror{{LocalPath: repoRoot}}); err != nil || detected != repoRoot {
		t.Fatalf("ResolveReadyMirrorRepoRoot(abs) = %q, %v", detected, err)
	}
	if _, err := ResolveReadyMirrorRepoRoot([]*ent.ProjectRepoMirror{{LocalPath: "https://example.com/repo.git"}}); err == nil {
		t.Fatal("ResolveReadyMirrorRepoRoot(https) expected error")
	}

	readyPrerequisite, err := service.GetRepositoryPrerequisite(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("GetRepositoryPrerequisite() ready error = %v", err)
	}
	if !readyPrerequisite.Ready() || readyPrerequisite.PrimaryRepoID == nil || readyPrerequisite.MirrorState == nil || *readyPrerequisite.MirrorState != catalogdomain.ProjectRepoMirrorStateReady {
		t.Fatalf("GetRepositoryPrerequisite() ready = %+v", readyPrerequisite)
	}

	missingPrerequisite, err := service.GetRepositoryPrerequisite(ctx, fixture.projectWithoutRepoID)
	if err != nil {
		t.Fatalf("GetRepositoryPrerequisite() missing repo error = %v", err)
	}
	if missingPrerequisite.Kind != WorkflowRepositoryPrerequisiteKindMissingPrimaryRepo || missingPrerequisite.Action != WorkflowRepositoryPrerequisiteActionBindPrimaryRepo {
		t.Fatalf("GetRepositoryPrerequisite() missing repo = %+v", missingPrerequisite)
	}

	projectItem, err := client.Project.Get(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("load project for no-mirror fixture: %v", err)
	}
	projectWithoutMirror, err := client.Project.Create().
		SetOrganizationID(projectItem.OrganizationID).
		SetName("Mirror Pending").
		SetSlug("mirror-pending").
		SetStatus("In Progress").
		Save(ctx)
	if err != nil {
		t.Fatalf("create projectWithoutMirror: %v", err)
	}
	if _, err := client.ProjectRepo.Create().
		SetProjectID(projectWithoutMirror.ID).
		SetName("pending-repo").
		SetRepositoryURL("https://github.com/acme/pending.git").
		SetDefaultBranch("main").
		SetWorkspaceDirname("pending-repo").
		SetIsPrimary(true).
		Save(ctx); err != nil {
		t.Fatalf("create primary repo without mirror: %v", err)
	}

	notReadyPrerequisite, err := service.GetRepositoryPrerequisite(ctx, projectWithoutMirror.ID)
	if err != nil {
		t.Fatalf("GetRepositoryPrerequisite() mirror pending error = %v", err)
	}
	if notReadyPrerequisite.Kind != WorkflowRepositoryPrerequisiteKindPrimaryMirrorNotReady ||
		notReadyPrerequisite.MirrorState == nil ||
		*notReadyPrerequisite.MirrorState != catalogdomain.ProjectRepoMirrorStateMissing ||
		notReadyPrerequisite.Action != WorkflowRepositoryPrerequisiteActionPrepareMirror {
		t.Fatalf("GetRepositoryPrerequisite() mirror pending = %+v", notReadyPrerequisite)
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

	if _, err := service.storageForProject(ctx, fixture.projectWithoutRepoID, workflowStorageUsageRead); !errors.Is(err, ErrPrimaryRepoRequired) {
		t.Fatalf("storageForProject() missing repo error = %v, want %v", err, ErrPrimaryRepoRequired)
	}
	if _, err := service.storageForProject(ctx, projectWithoutMirror.ID, workflowStorageUsageRead); !errors.Is(err, ErrPrimaryMirrorNotReady) {
		t.Fatalf("storageForProject() mirror pending error = %v, want %v", err, ErrPrimaryMirrorNotReady)
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

	storage, err := service.storageForProject(ctx, fixture.projectID, workflowStorageUsageRead)
	if err != nil {
		t.Fatalf("storageForProject() error = %v", err)
	}
	harnessAbsPath := storage.registry.absolutePath(created.HarnessPath)

	if err := os.Remove(harnessAbsPath); err != nil {
		t.Fatalf("remove harness file: %v", err)
	}
	if err := os.MkdirAll(harnessAbsPath, 0o750); err != nil {
		t.Fatalf("mkdir harness path: %v", err)
	}
	storage.registry.mu.Lock()
	delete(storage.registry.cache, created.HarnessPath)
	storage.registry.mu.Unlock()
	if _, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: created.ID,
		Content:    "---\nworkflow:\n  role: coding\n---\n\n# Updated\n",
	}); err == nil || !strings.Contains(err.Error(), "read workflow harness before update") {
		t.Fatalf("UpdateHarness(read failure) error = %v", err)
	}

	if err := os.RemoveAll(harnessAbsPath); err != nil {
		t.Fatalf("remove harness directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(harnessAbsPath), 0o750); err != nil {
		t.Fatalf("mkdir harness parent: %v", err)
	}
	if err := os.WriteFile(harnessAbsPath, []byte(created.HarnessContent), 0o600); err != nil {
		t.Fatalf("restore harness file: %v", err)
	}
	if _, err := storage.registry.Read(created.HarnessPath); err != nil {
		t.Fatalf("registry.Read() error = %v", err)
	}

	brokenRoot := filepath.Join(t.TempDir(), "registry-root-file")
	if err := os.WriteFile(brokenRoot, []byte("x"), 0o600); err != nil {
		t.Fatalf("write broken root file: %v", err)
	}
	storage.registry.rootDir = brokenRoot

	if _, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: created.ID,
		Content:    "---\nworkflow:\n  role: coding\n---\n\n# Write failure\n",
	}); err == nil || (!strings.Contains(err.Error(), "write harness file") && !strings.Contains(err.Error(), "create harness parent directory")) {
		t.Fatalf("UpdateHarness(write failure) error = %v", err)
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
	if codingContext == nil || codingContext.CurrentActive != 1 || !slices.Equal(codingContext.Skills, []string{"openase-platform", "commit"}) || len(codingContext.RecentTickets) != 3 {
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
	projectRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName(filepath.Base(repoRoot)).
		SetRepositoryURL("https://github.com/GrandCX/openase.git").
		SetDefaultBranch("main").
		SetWorkspaceDirname(filepath.Base(repoRoot)).
		SetIsPrimary(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create primary project repo: %v", err)
	}
	if _, err := client.ProjectRepoMirror.Create().
		SetProjectRepoID(projectRepo.ID).
		SetMachineID(machine.ID).
		SetLocalPath(repoRoot).
		SetState("ready").
		Save(ctx); err != nil {
		t.Fatalf("create primary project repo mirror: %v", err)
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

	port := freeWorkflowPort(t)
	dataDir := t.TempDir()
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(port).
			Username("postgres").
			Password("postgres").
			Database("openase").
			RuntimePath(filepath.Join(dataDir, "runtime")).
			BinariesPath(filepath.Join(dataDir, "binaries")).
			DataPath(filepath.Join(dataDir, "data")),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Stop(); err != nil {
			t.Errorf("stop embedded postgres: %v", err)
		}
	})

	dsn := "postgres://postgres:postgres@127.0.0.1:" + strconv.Itoa(int(port)) + "/openase?sslmode=disable"
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open ent client: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client: %v", err)
		}
	})
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return client
}

func createWorkflowTestGitRepo(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("create git marker: %v", err)
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

func freeWorkflowPort(t *testing.T) uint32 {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	parsed, err := strconv.ParseUint(strconv.Itoa(port), 10, 32)
	if err != nil {
		t.Fatalf("parse free port %d: %v", port, err)
	}

	return uint32(parsed)
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
