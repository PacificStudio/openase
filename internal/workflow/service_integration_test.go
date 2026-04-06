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
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	workflowrepo "github.com/BetterAndBetterII/openase/internal/repo/workflow"
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

	hooksRoot, err := workspaceinfra.ProjectHooksPath(repoRoot, fixture.projectID.String())
	if err != nil {
		t.Fatalf("ProjectHooksPath() error = %v", err)
	}
	activateMarkerPath := filepath.Join(hooksRoot, "activate.marker")
	reloadMarkerPath := filepath.Join(hooksRoot, "reload.marker")
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
		Type:                TypeCoding,
		HarnessContent:      "# Coding\n",
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
	if boundDoc.Version != 2 {
		t.Fatalf("BindSkills() version = %d, want 2", boundDoc.Version)
	}
	if got := mustReadWorkflowFile(t, reloadMarkerPath); got != "on_reload:2" {
		t.Fatalf("reload marker after BindSkills() = %q", got)
	}
	if boundDoc.Content != created.HarnessContent || strings.Contains(boundDoc.Content, "skills:") {
		t.Fatalf("BindSkills() should not rewrite harness content, got %q", boundDoc.Content)
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
	if got := mustReadWorkflowFile(t, reloadMarkerPath); got != "on_reload:3" {
		t.Fatalf("reload marker after UnbindSkills() = %q", got)
	}
	if unboundDoc.Content != created.HarnessContent || strings.Contains(unboundDoc.Content, "skills:") {
		t.Fatalf("UnbindSkills() should not rewrite harness content, got %q", unboundDoc.Content)
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

	updatedHarnessContent := "# Updated by API\n"
	updatedHarnessDoc, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: created.ID,
		Content:    updatedHarnessContent,
	})
	if err != nil {
		t.Fatalf("UpdateHarness() error = %v", err)
	}
	if updatedHarnessDoc.Version != 4 {
		t.Fatalf("UpdateHarness() = %+v", updatedHarnessDoc)
	}
	if got := mustReadWorkflowFile(t, reloadMarkerPath); got != "on_reload:4" {
		t.Fatalf("reload marker after UpdateHarness() = %q", got)
	}
	if updatedHarnessDoc.Content != updatedHarnessContent {
		t.Fatalf("UpdateHarness() content = %q, want %q", updatedHarnessDoc.Content, updatedHarnessContent)
	}
	currentVersion, err := service.currentWorkflowVersion(ctx, created.ID)
	if err != nil {
		t.Fatalf("currentWorkflowVersion() error = %v", err)
	}
	if strings.Contains(currentVersion.ContentMarkdown, "skill-two") || strings.Contains(currentVersion.ContentMarkdown, "skills:") {
		t.Fatalf("stored workflow version still carries projected bindings: %q", currentVersion.ContentMarkdown)
	}

	deleted, err := service.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted.ID != created.ID {
		t.Fatalf("Delete() = %+v", deleted)
	}
	if _, err := service.Get(ctx, created.ID); !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("Get() after delete error = %v, want %v", err, ErrWorkflowNotFound)
	}

	if err := service.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestWorkflowServiceShellQuotesRuntimeInterpolationInHooks(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	hooksRoot, err := workspaceinfra.ProjectHooksPath(repoRoot, fixture.projectID.String())
	if err != nil {
		t.Fatalf("ProjectHooksPath() error = %v", err)
	}
	safeMarkerPath := filepath.Join(hooksRoot, "safe.marker")
	pwnedPath := filepath.Join(hooksRoot, "pwned")
	maliciousWorkflowName := "foo; touch pwned #\nline two 'quoted' $(boom)"

	workflowHooks := map[string]any{
		"workflow_hooks": map[string]any{
			"on_activate": []map[string]any{{
				"cmd": "printf '%s' {{ workflow.name }} > safe.marker",
			}},
		},
	}

	created, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                maliciousWorkflowName,
		Type:                TypeCoding,
		HarnessContent:      "# Coding\n",
		Hooks:               workflowHooks,
		MaxConcurrent:       1,
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
	if created.Name != maliciousWorkflowName {
		t.Fatalf("Create() name = %q, want %q", created.Name, maliciousWorkflowName)
	}
	if got := mustReadWorkflowFile(t, safeMarkerPath); got != maliciousWorkflowName {
		t.Fatalf("safe marker = %q, want %q", got, maliciousWorkflowName)
	}
	if _, err := os.Stat(pwnedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %q to be absent, got err=%v", pwnedPath, err)
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
		Type:                TypeCoding,
		HarnessContent:      "# Snapshot v1\n",
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
	staleHarnessPath := filepath.Join(codexRoot, filepath.FromSlash(created.HarnessPath))
	if err := os.MkdirAll(filepath.Dir(staleHarnessPath), 0o750); err != nil {
		t.Fatalf("mkdir stale harness path: %v", err)
	}
	if err := os.WriteFile(staleHarnessPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("seed stale harness snapshot: %v", err)
	}
	if _, err := service.MaterializeRuntimeSnapshot(MaterializeRuntimeSnapshotInput{
		WorkspaceRoot: codexRoot,
		AdapterType:   string(entagentprovider.AdapterTypeCodexAppServer),
		Snapshot:      snapshotV1,
	}); err != nil {
		t.Fatalf("MaterializeRuntimeSnapshot(codex) error = %v", err)
	}
	if _, err := os.Stat(staleHarnessPath); !os.IsNotExist(err) {
		t.Fatalf("expected runtime harness snapshot to stay absent, stat err=%v", err)
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
		Content:    "# Snapshot v2\n",
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

func TestSkillBundleStorageRefreshAndRuntimeSnapshots(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	createdWorkflow, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Bundle Workflow",
		Type:                TypeCoding,
		HarnessContent:      "# Bundle Workflow\n",
		MaxConcurrent:       1,
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

	skillV1, err := service.CreateSkillBundle(ctx, CreateSkillBundleInput{
		ProjectID: fixture.projectID,
		Name:      "deploy-openase",
		Files: []SkillBundleFileInput{
			{
				Path: "SKILL.md",
				Content: []byte(strings.TrimSpace(`
---
name: "deploy-openase"
description: "Safely redeploy OpenASE"
---

# Deploy OpenASE

Use the bundled scripts to redeploy the local service.
`) + "\n"),
				MediaType: "text/markdown; charset=utf-8",
			},
			{
				Path:         "scripts/redeploy.sh",
				Content:      []byte("#!/usr/bin/env bash\necho v1\n"),
				IsExecutable: true,
				MediaType:    "text/x-shellscript; charset=utf-8",
			},
			{
				Path:      "references/runbook.md",
				Content:   []byte("# Runbook\n\nStep v1\n"),
				MediaType: "text/markdown; charset=utf-8",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateSkillBundle() error = %v", err)
	}
	if skillV1.FileCount != 3 || len(skillV1.Files) != 3 || skillV1.BundleHash == "" {
		t.Fatalf("CreateSkillBundle() = %+v", skillV1)
	}
	if got := findSkillBundleFile(t, skillV1.Files, "scripts/redeploy.sh"); !got.IsExecutable || string(got.Content) != "#!/usr/bin/env bash\necho v1\n" {
		t.Fatalf("unexpected stored script file = %+v", got)
	}

	if _, err := service.BindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: createdWorkflow.ID,
		Skills:     []string{"deploy-openase"},
	}); err != nil {
		t.Fatalf("BindSkills() error = %v", err)
	}

	workspaceRoot := t.TempDir()
	refreshResult, err := service.RefreshSkills(ctx, RefreshSkillsInput{
		ProjectID:     fixture.projectID,
		WorkspaceRoot: workspaceRoot,
		AdapterType:   string(entagentprovider.AdapterTypeCodexAppServer),
		WorkflowID:    &createdWorkflow.ID,
	})
	if err != nil {
		t.Fatalf("RefreshSkills() error = %v", err)
	}
	if !slices.Contains(refreshResult.InjectedSkills, "deploy-openase") || !slices.Contains(refreshResult.InjectedSkills, "openase-platform") {
		t.Fatalf("RefreshSkills() = %+v", refreshResult)
	}

	scriptPath := filepath.Join(workspaceRoot, ".codex", "skills", "deploy-openase", "scripts", "redeploy.sh")
	scriptInfo, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("stat projected script: %v", err)
	}
	if scriptInfo.Mode()&0o111 == 0 {
		t.Fatalf("expected projected script to be executable, mode=%v", scriptInfo.Mode())
	}
	// #nosec G304 -- scriptPath is built from the test-controlled temporary workspace root.
	if projectedScript, err := os.ReadFile(scriptPath); err != nil || string(projectedScript) != "#!/usr/bin/env bash\necho v1\n" {
		t.Fatalf("projected script = %q, err=%v", string(projectedScript), err)
	}
	projectedReferencePath := filepath.Join(
		workspaceRoot,
		".codex",
		"skills",
		"deploy-openase",
		"references",
		"runbook.md",
	)
	// #nosec G304 -- projectedReferencePath is built from the test-controlled temporary workspace root.
	if projectedReference, err := os.ReadFile(projectedReferencePath); err != nil || string(projectedReference) != "# Runbook\n\nStep v1\n" {
		t.Fatalf("projected reference = %q, err=%v", string(projectedReference), err)
	}

	snapshotV1, err := service.ResolveRuntimeSnapshot(ctx, createdWorkflow.ID)
	if err != nil {
		t.Fatalf("ResolveRuntimeSnapshot(v1) error = %v", err)
	}
	runtimeSkillV1 := findRuntimeSkillSnapshot(t, snapshotV1.Skills, "deploy-openase")
	if len(runtimeSkillV1.Files) != 3 {
		t.Fatalf("expected runtime bundle files, got %+v", runtimeSkillV1)
	}
	if got := findRuntimeSkillFile(t, runtimeSkillV1.Files, "scripts/redeploy.sh"); !got.IsExecutable || string(got.Content) != "#!/usr/bin/env bash\necho v1\n" {
		t.Fatalf("unexpected runtime script file = %+v", got)
	}

	updatedSkill, err := service.UpdateSkillBundle(ctx, UpdateSkillBundleInput{
		SkillID: skillV1.ID,
		Files: []SkillBundleFileInput{
			{
				Path: "SKILL.md",
				Content: []byte(strings.TrimSpace(`
---
name: "deploy-openase"
description: "Safely redeploy OpenASE"
---

# Deploy OpenASE

Use the bundled scripts to redeploy the local service.

Version 2.
`) + "\n"),
				MediaType: "text/markdown; charset=utf-8",
			},
			{
				Path:         "scripts/redeploy.sh",
				Content:      []byte("#!/usr/bin/env bash\necho v2\n"),
				IsExecutable: true,
				MediaType:    "text/x-shellscript; charset=utf-8",
			},
			{
				Path:      "references/runbook.md",
				Content:   []byte("# Runbook\n\nStep v2\n"),
				MediaType: "text/markdown; charset=utf-8",
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSkillBundle() error = %v", err)
	}
	if updatedSkill.CurrentVersion != 2 || updatedSkill.FileCount != 3 {
		t.Fatalf("UpdateSkillBundle() = %+v", updatedSkill)
	}

	snapshotV2, err := service.ResolveRuntimeSnapshot(ctx, createdWorkflow.ID)
	if err != nil {
		t.Fatalf("ResolveRuntimeSnapshot(v2) error = %v", err)
	}
	runtimeSkillV2 := findRuntimeSkillSnapshot(t, snapshotV2.Skills, "deploy-openase")
	if runtimeSkillV2.VersionID == runtimeSkillV1.VersionID {
		t.Fatalf("expected skill version to advance, got %+v", snapshotV2.Skills)
	}
	if got := findRuntimeSkillFile(t, runtimeSkillV2.Files, "scripts/redeploy.sh"); string(got.Content) != "#!/usr/bin/env bash\necho v2\n" {
		t.Fatalf("unexpected v2 runtime script = %+v", got)
	}

	recordedV1, err := service.ResolveRecordedRuntimeSnapshot(ctx, ResolveRecordedRuntimeSnapshotInput{
		WorkflowID:        createdWorkflow.ID,
		WorkflowVersionID: &snapshotV1.Workflow.VersionID,
		SkillVersionIDs:   skillVersionIDs(snapshotV1.Skills),
	})
	if err != nil {
		t.Fatalf("ResolveRecordedRuntimeSnapshot(v1) error = %v", err)
	}
	recordedSkillV1 := findRuntimeSkillSnapshot(t, recordedV1.Skills, "deploy-openase")
	if got := findRuntimeSkillFile(t, recordedSkillV1.Files, "scripts/redeploy.sh"); string(got.Content) != "#!/usr/bin/env bash\necho v1\n" {
		t.Fatalf("expected recorded v1 runtime script, got %+v", got)
	}

	recordedRoot := t.TempDir()
	if _, err := service.MaterializeRuntimeSnapshot(MaterializeRuntimeSnapshotInput{
		WorkspaceRoot: recordedRoot,
		AdapterType:   string(entagentprovider.AdapterTypeCodexAppServer),
		Snapshot:      recordedV1,
	}); err != nil {
		t.Fatalf("MaterializeRuntimeSnapshot(recorded v1) error = %v", err)
	}
	recordedScriptPath := filepath.Join(recordedRoot, ".codex", "skills", "deploy-openase", "scripts", "redeploy.sh")
	recordedInfo, err := os.Stat(recordedScriptPath)
	if err != nil {
		t.Fatalf("stat recorded projected script: %v", err)
	}
	if recordedInfo.Mode()&0o111 == 0 {
		t.Fatalf("expected recorded projected script to stay executable, mode=%v", recordedInfo.Mode())
	}
	// #nosec G304 -- recordedScriptPath is built from the test-controlled temporary workspace root.
	if recordedScript, err := os.ReadFile(recordedScriptPath); err != nil || string(recordedScript) != "#!/usr/bin/env bash\necho v1\n" {
		t.Fatalf("recorded projected script = %q, err=%v", string(recordedScript), err)
	}
}

func TestBuiltinOpenASEPlatformSkillProjectsWorkpadScriptAcrossRuntimes(t *testing.T) {
	ctx := context.Background()
	client := openWorkflowTestEntClient(t)
	repoRoot := createWorkflowTestGitRepo(t)
	service := newWorkflowTestService(t, client, repoRoot)
	fixture := seedWorkflowServiceFixture(ctx, t, client, repoRoot)

	skills, err := service.ListSkills(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}
	platformSkill := findSkillByName(skills, "openase-platform")
	if platformSkill == nil {
		t.Fatalf("expected openase-platform in %+v", skills)
	}
	platformDetail, err := service.GetSkill(ctx, platformSkill.ID)
	if err != nil {
		t.Fatalf("GetSkill(openase-platform) error = %v", err)
	}
	if got := findSkillBundleFile(t, platformDetail.Files, "scripts/upsert_workpad.sh"); !got.IsExecutable {
		t.Fatalf("expected builtin workpad helper to be executable, got %+v", got)
	}

	tests := []struct {
		name        string
		adapterType string
		scriptPath  string
	}{
		{
			name:        "gemini",
			adapterType: "gemini-cli",
			scriptPath:  ".gemini/skills/openase-platform/scripts/upsert_workpad.sh",
		},
		{
			name:        "agent",
			adapterType: "custom",
			scriptPath:  ".agent/skills/openase-platform/scripts/upsert_workpad.sh",
		},
		{
			name:        "codex",
			adapterType: "codex-app-server",
			scriptPath:  ".codex/skills/openase-platform/scripts/upsert_workpad.sh",
		},
		{
			name:        "claude",
			adapterType: "claude-code-cli",
			scriptPath:  ".claude/skills/openase-platform/scripts/upsert_workpad.sh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspaceRoot := t.TempDir()
			refreshResult, err := service.RefreshSkills(ctx, RefreshSkillsInput{
				ProjectID:     fixture.projectID,
				WorkspaceRoot: workspaceRoot,
				AdapterType:   tt.adapterType,
			})
			if err != nil {
				t.Fatalf("RefreshSkills(%s) error = %v", tt.adapterType, err)
			}
			if !slices.Contains(refreshResult.InjectedSkills, "openase-platform") {
				t.Fatalf("RefreshSkills(%s) = %+v", tt.adapterType, refreshResult)
			}

			scriptPath := filepath.Join(workspaceRoot, filepath.FromSlash(tt.scriptPath))
			info, err := os.Stat(scriptPath)
			if err != nil {
				t.Fatalf("stat projected workpad script: %v", err)
			}
			if info.Mode()&0o111 == 0 {
				t.Fatalf("expected projected script to be executable, mode=%v", info.Mode())
			}
		})
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
		Type:                TypeCoding,
		HarnessContent:      "# Coding\n",
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
		Save(ctx); err != nil {
		t.Fatalf("create ready project repo: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, readyProject.ID)
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
		Type:                TypeCoding,
		HarnessContent:      "# Coding\n",
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
	if strings.Contains(harnessDoc.Content, "skills:") {
		t.Fatalf("harness content unexpectedly contains skill frontmatter: %q", harnessDoc.Content)
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
		Type:                TypeCoding,
		HarnessContent:      "# Coverage\n",
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
	projectControlRoot, err := service.ProjectControlRoot(fixture.projectID)
	if err != nil {
		t.Fatalf("ProjectControlRoot() error = %v", err)
	}
	wantProjectControlRoot, err := workspaceinfra.ProjectStatePath(repoRoot, fixture.projectID.String())
	if err != nil {
		t.Fatalf("ProjectStatePath() error = %v", err)
	}
	if projectControlRoot != wantProjectControlRoot {
		t.Fatalf("ProjectControlRoot() = %q, want %q", projectControlRoot, wantProjectControlRoot)
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
		Content:    "# ok\n",
	}); !errors.Is(err, ErrWorkflowNotFound) {
		t.Fatalf("UpdateHarness() missing workflow error = %v, want %v", err, ErrWorkflowNotFound)
	}
	if _, err := service.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: uuid.New(),
		Content:    "{{",
	}); !errors.Is(err, ErrHarnessInvalid) {
		t.Fatalf("UpdateHarness() invalid content error = %v, want %v", err, ErrHarnessInvalid)
	}

	if _, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             uuid.New(),
		Name:                "Bad Agent",
		Type:                TypeCoding,
		HarnessContent:      "# Coding\n",
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
		Type:                TypeCoding,
		HarnessContent:      "# Coding\n",
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
		Type:                TypeCoding,
		HarnessContent:      "# Coding\n",
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
		Type:                TypeCoding,
		HarnessPath:         &badHarnessPath,
		HarnessContent:      "# Coding\n",
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
		Type:                TypeCoding,
		HarnessContent:      "# Coding\n",
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
		Type:                TypeCoding,
		HarnessContent:      "# Coding\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Backlog"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	}); !errors.Is(err, ErrWorkflowNameConflict) {
		t.Fatalf("Create() duplicate name error = %v, want %v", err, ErrWorkflowNameConflict)
	}

	duplicateHarnessPath := created.HarnessPath
	if _, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Duplicate Harness Path",
		Type:                TypeCoding,
		HarnessPath:         &duplicateHarnessPath,
		HarnessContent:      "# Coding\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Backlog"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	}); !errors.Is(err, ErrWorkflowHarnessPathConflict) {
		t.Fatalf("Create() duplicate harness path error = %v, want %v", err, ErrWorkflowHarnessPathConflict)
	}

	parallelCreated, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Parallel Workflow",
		Type:                TypeCoding,
		HarnessContent:      "# Parallel\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Backlog"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	})
	if err != nil {
		t.Fatalf("Create() parallel baseline error = %v", err)
	}

	if _, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Conflicting Pickup Workflow",
		Type:                TypeCoding,
		HarnessContent:      "# Conflicting\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Done"]),
	}); !errors.Is(err, ErrPickupStatusConflict) {
		t.Fatalf("Create() duplicate pickup status error = %v, want %v", err, ErrPickupStatusConflict)
	}

	if _, err := service.Update(ctx, UpdateInput{
		WorkflowID:      parallelCreated.ID,
		PickupStatusIDs: Some(MustStatusBindingSet(fixture.statusIDs["Todo"])),
	}); !errors.Is(err, ErrPickupStatusConflict) {
		t.Fatalf("Update() duplicate pickup status error = %v, want %v", err, ErrPickupStatusConflict)
	}

	if _, err := service.Update(ctx, UpdateInput{
		WorkflowID:      created.ID,
		PickupStatusIDs: Some(MustStatusBindingSet(fixture.statusIDs["Todo"])),
	}); err != nil {
		t.Fatalf("Update() self pickup status reuse error = %v", err)
	}

	if _, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Overlapping Status Workflow",
		Type:                TypeCoding,
		HarnessContent:      "# Overlapping\n",
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            false,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["In Review"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["In Review"]),
	}); !errors.Is(err, ErrWorkflowStatusBindingOverlap) {
		t.Fatalf("Create() overlapping status bindings error = %v, want %v", err, ErrWorkflowStatusBindingOverlap)
	}

	if _, err := service.Update(ctx, UpdateInput{
		WorkflowID:      created.ID,
		FinishStatusIDs: Some(MustStatusBindingSet(fixture.statusIDs["Todo"])),
	}); !errors.Is(err, ErrWorkflowStatusBindingOverlap) {
		t.Fatalf("Update() overlapping status bindings error = %v, want %v", err, ErrWorkflowStatusBindingOverlap)
	}

	if _, err := service.Update(ctx, UpdateInput{
		WorkflowID:  created.ID,
		HarnessPath: Some("../escape.md"),
	}); !errors.Is(err, ErrHarnessInvalid) {
		t.Fatalf("Update() invalid harness path error = %v, want %v", err, ErrHarnessInvalid)
	}

	if _, err := service.Update(ctx, UpdateInput{
		WorkflowID: parallelCreated.ID,
		Name:       Some(created.Name),
	}); !errors.Is(err, ErrWorkflowNameConflict) {
		t.Fatalf("Update() duplicate name error = %v, want %v", err, ErrWorkflowNameConflict)
	}

	if _, err := service.Update(ctx, UpdateInput{
		WorkflowID:  parallelCreated.ID,
		HarnessPath: Some(created.HarnessPath),
	}); !errors.Is(err, ErrWorkflowHarnessPathConflict) {
		t.Fatalf("Update() duplicate harness path error = %v, want %v", err, ErrWorkflowHarnessPathConflict)
	}

	if got := service.mapWorkflowReadError("get workflow", errors.New("boom")); got == nil || !errors.Is(got, errors.New("boom")) {
		if got == nil || got.Error() != "get workflow: boom" {
			t.Fatalf("mapWorkflowReadError() = %v", got)
		}
	}
	if got := service.mapWorkflowWriteError("delete workflow", errors.New("tickets still reference workflow")); got == nil || got.Error() != "delete workflow: tickets still reference workflow" {
		t.Fatalf("mapWorkflowWriteError(tickets) = %v", got)
	}
	if got := service.mapWorkflowWriteError("delete workflow", errors.New("scheduled_jobs still reference workflow")); got == nil || got.Error() != "delete workflow: scheduled_jobs still reference workflow" {
		t.Fatalf("mapWorkflowWriteError(scheduled_jobs) = %v", got)
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
		Type:                TypeCoding,
		HarnessContent:      "# Initial\n",
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
		Content:    "{{",
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
		Content:    "# Write failure\n",
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

	if _, err := service.CreateSkill(ctx, CreateSkillInput{
		ProjectID: fixture.projectID,
		Name:      "blocked-bind",
		Content:   "# Blocked Bind\nbody\n",
	}); err != nil {
		t.Fatalf("CreateSkill(blocked-bind) error = %v", err)
	}
	if _, err := service.BindSkills(ctx, UpdateWorkflowSkillsInput{
		WorkflowID: created.ID,
		Skills:     []string{"blocked-bind"},
	}); !errors.Is(err, ErrWorkflowHookBlocked) {
		t.Fatalf("BindSkills(blocked) error = %v, want %v", err, ErrWorkflowHookBlocked)
	}

	currentVersion, err = service.currentWorkflowVersion(ctx, created.ID)
	if err != nil {
		t.Fatalf("currentWorkflowVersion() after blocked bind error = %v", err)
	}
	if currentVersion.Version != 1 || !strings.Contains(currentVersion.ContentMarkdown, "# Initial") {
		t.Fatalf("current workflow version changed after blocked bind: %+v", currentVersion)
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

	templateContent := `Implement product changes end to end.

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
WorkflowBindings {% for wf in project.workflows %}{{ wf.role_name }}=pickup[{% for status in wf.pickup_statuses %}{{ status.name }}:{{ status.stage }}|{% endfor %}]finish[{% for status in wf.finish_statuses %}{{ status.name }}:{{ status.stage }}|{% endfor %}];{% endfor %}
WorkflowHistory {% for wf in project.workflows %}{{ wf.role_name }}={% for recent in wf.recent_tickets %}{{ recent.identifier }}:{{ recent.status }}:{{ recent.retry_paused }}:{{ recent.consecutive_errors }}|{% endfor %};{% endfor %}
ProjectStatuses {% for status in project.statuses %}{{ status.name }}:{{ status.stage }}:{{ status.color }}|{% endfor %}
ProjectMachines {% for machine in project.machines %}{{ machine.name }}:{{ machine.status }}:{{ machine.resources.transport | default("none") }}:{{ machine.labels | join(",") }}|{% endfor %}
ProjectUpdates {% for update in project.updates %}{{ update.status }}:{{ update.title }}:{{ update.created_by }}:{% for comment in update.comments %}{{ comment.created_by }}={{ comment.body_markdown }}|{% endfor %};{% endfor %}
Platform {{ platform.api_url }} {{ platform.project_id }} {{ platform.ticket_id }}
Timestamp {{ timestamp }} Version {{ openase_version }} URL {{ ticket.url }}
{% if attempt > 1 %}retry{% endif %}
`

	codingWorkflow, err := service.Create(ctx, CreateInput{
		ProjectID:           fixture.projectID,
		AgentID:             fixture.agentID,
		Name:                "Coding Workflow",
		Type:                TypeCoding,
		RoleSlug:            "fullstack-developer",
		RoleName:            "fullstack-developer",
		RoleDescription:     "Implement product changes end to end.",
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
		Type:                TypeCustom,
		RoleSlug:            "dispatcher",
		RoleName:            "dispatcher",
		RoleDescription:     "Evaluate backlog tickets and route them to the right workflow.",
		HarnessContent:      "Evaluate backlog tickets and route them to the right workflow.\n",
		Hooks:               map[string]any{},
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      5,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Backlog"]),
		FinishStatusIDs:     MustStatusBindingSet(fixture.statusIDs["Todo"]),
	}); err != nil {
		t.Fatalf("create dispatcher workflow: %v", err)
	}

	primaryRepo, err := client.ProjectRepo.Query().
		Where(entprojectrepo.ProjectIDEQ(fixture.projectID)).
		Order(entprojectrepo.ByID()).
		Only(ctx)
	if err != nil {
		t.Fatalf("load project repo: %v", err)
	}
	if _, err := client.ProjectRepo.UpdateOneID(primaryRepo.ID).
		SetName("backend").
		SetDefaultBranch("main").
		SetWorkspaceDirname("backend").
		Save(ctx); err != nil {
		t.Fatalf("normalize project repo metadata: %v", err)
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
		SetExternalRef("PacificStudio/openase#20").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(primaryRepo.ID).
		SetBranchName("agent/codex/ASE-42").
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
	projectUpdateService := projectupdateservice.NewService(client, nil)
	thread, err := projectUpdateService.AddThread(ctx, projectupdateservice.AddThreadInput{
		ProjectID: fixture.projectID,
		Status:    projectupdateservice.StatusAtRisk,
		Title:     "GPU queue saturation",
		Body:      "The A100 queue is drifting upward.",
		CreatedBy: "agent:dispatcher-01",
	})
	if err != nil {
		t.Fatalf("create project update thread: %v", err)
	}
	if _, err := projectUpdateService.AddComment(ctx, projectupdateservice.AddCommentInput{
		ProjectID: fixture.projectID,
		ThreadID:  thread.ID,
		Body:      "Routing lower-priority runs away from the saturated pool.",
		CreatedBy: "agent:dispatcher-01",
	}); err != nil {
		t.Fatalf("create project update comment: %v", err)
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
			Name:        "gpu-01",
			Host:        "10.0.1.10",
			Description: "NVIDIA A100 x4",
			Labels:      []string{"gpu", "a100"},
			Resources: map[string]any{
				"transport": "ssh",
			},
			WorkspaceRoot: "/workspaces",
		},
		AccessibleMachines: []HarnessAccessibleMachineData{
			{
				Name:        "storage",
				Host:        "10.0.1.20",
				Description: "Artifact storage",
				Labels:      []string{"storage", "nfs"},
				Resources: map[string]any{
					"transport": "ssh",
				},
				SSHUser: "openase",
			},
			{
				Name:        "gpu-01",
				Host:        "10.0.1.10",
				Description: "duplicate current machine",
				Labels:      []string{"gpu", "a100"},
				Resources: map[string]any{
					"transport": "ssh",
				},
				SSHUser: "openase",
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
	if dispatcherContext == nil || len(dispatcherContext.PickupStatuses) != 1 || dispatcherContext.PickupStatuses[0].Name != "Backlog" || dispatcherContext.PickupStatuses[0].Stage != "backlog" {
		t.Fatalf("dispatcher pickup statuses = %+v", dispatcherContext)
	}
	if dispatcherContext == nil || len(dispatcherContext.FinishStatuses) != 1 || dispatcherContext.FinishStatuses[0].Name != "Todo" || dispatcherContext.FinishStatuses[0].Stage != "unstarted" {
		t.Fatalf("dispatcher finish statuses = %+v", dispatcherContext)
	}

	if len(data.Project.Statuses) == 0 || data.Project.Statuses[0].Color == "" || data.Project.Statuses[0].Stage == "" {
		t.Fatalf("project statuses = %+v", data.Project.Statuses)
	}
	if len(data.Project.Machines) != 2 || data.Project.Machines[0].Name != "gpu-01" || data.Project.Machines[0].Status != "current" || data.Project.Machines[1].Name != "storage" {
		t.Fatalf("project machines = %+v", data.Project.Machines)
	}
	if len(data.Project.Updates) != 1 || data.Project.Updates[0].Title != "GPU queue saturation" || len(data.Project.Updates[0].Comments) != 1 {
		t.Fatalf("project updates = %+v", data.Project.Updates)
	}
	if data.Project.Machines[0].Resources["transport"] != "ssh" || data.Project.Machines[1].Resources["transport"] != "ssh" {
		t.Fatalf("project machine resources = %+v", data.Project.Machines)
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
		"WorkflowBindings fullstack-developer=pickup[Todo:unstarted|]finish[Done:completed|];dispatcher=pickup[Backlog:backlog|]finish[Todo:unstarted|];",
		"WorkflowHistory fullstack-developer=ASE-42:Todo:False:0|ASE-41:In Review:True:2|ASE-40:Todo:False:0|;dispatcher=;",
		"ProjectStatuses Backlog:backlog:#6B7280|Todo:unstarted:#3B82F6|In Progress:started:#F59E0B|In Review:started:#8B5CF6|Done:completed:#10B981|Cancelled:canceled:#4B5563|",
		"ProjectMachines gpu-01:current:ssh:gpu,a100|storage:accessible:ssh:storage,nfs|",
		"ProjectUpdates at_risk:GPU queue saturation:agent:dispatcher-01:agent:dispatcher-01=Routing lower-priority runs away from the saturated pool.|;",
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
		Save(ctx); err != nil {
		t.Fatalf("create project repo: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
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

	service, err := NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
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

func findRuntimeSkillFile(t *testing.T, items []RuntimeSkillFileSnapshot, path string) RuntimeSkillFileSnapshot {
	t.Helper()

	for _, item := range items {
		if item.Path == path {
			return item
		}
	}
	t.Fatalf("runtime skill file %s not found in %+v", path, items)
	return RuntimeSkillFileSnapshot{}
}

func findSkillBundleFile(t *testing.T, items []SkillBundleFile, path string) SkillBundleFile {
	t.Helper()

	for _, item := range items {
		if item.Path == path {
			return item
		}
	}
	t.Fatalf("skill bundle file %s not found in %+v", path, items)
	return SkillBundleFile{}
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
