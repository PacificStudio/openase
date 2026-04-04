package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaterializeDraftSkillBundleProjectsIntoCodexSkillDirectory(t *testing.T) {
	workspaceRoot := t.TempDir()

	bundle, err := BuildSkillBundle("deploy-openase", []SkillBundleFileInput{
		{
			Path:    "SKILL.md",
			Content: []byte("---\nname: deploy-openase\ndescription: Safely redeploy OpenASE\n---\n\n# Deploy\n\nUse the bundled script.\n"),
		},
		{
			Path:         "scripts/redeploy.sh",
			Content:      []byte("#!/usr/bin/env bash\necho deploy\n"),
			IsExecutable: true,
		},
	})
	if err != nil {
		t.Fatalf("BuildSkillBundle() error = %v", err)
	}

	projection, err := MaterializeDraftSkillBundle(DraftSkillProjectionInput{
		WorkspaceRoot: workspaceRoot,
		AdapterType:   "codex-app-server",
		Bundle:        bundle,
	})
	if err != nil {
		t.Fatalf("MaterializeDraftSkillBundle() error = %v", err)
	}

	if projection.SkillsDir != filepath.Join(workspaceRoot, ".codex", "skills") {
		t.Fatalf("SkillsDir = %q", projection.SkillsDir)
	}
	if _, err := os.Stat(filepath.Join(projection.SkillDir, "SKILL.md")); err != nil {
		t.Fatalf("stat projected SKILL.md: %v", err)
	}
	scriptPath := filepath.Join(projection.SkillDir, "scripts", "redeploy.sh")
	scriptInfo, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("stat projected script: %v", err)
	}
	if scriptInfo.Mode().Perm() != 0o700 {
		t.Fatalf("projected script mode = %v, want 0700", scriptInfo.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(workspaceRoot, ".openase", "bin", "openase")); err != nil {
		t.Fatalf("stat openase wrapper: %v", err)
	}

	loaded, err := LoadSkillBundleFromDirectory("deploy-openase", projection.SkillDir)
	if err != nil {
		t.Fatalf("LoadSkillBundleFromDirectory() error = %v", err)
	}
	if loaded.BundleHash != bundle.BundleHash {
		t.Fatalf("loaded bundle hash = %q, want %q", loaded.BundleHash, bundle.BundleHash)
	}
	if len(loaded.Files) != len(bundle.Files) {
		t.Fatalf("loaded files = %d, want %d", len(loaded.Files), len(bundle.Files))
	}
}
