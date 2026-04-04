package cli

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestBuildSkillImportPayloadPackagesBundleDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deploy-openase")
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o750); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "references"), 0o750); err != nil {
		t.Fatalf("mkdir references: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("---\nname: \"deploy-openase\"\ndescription: \"Safely redeploy OpenASE\"\n---\n\n# Deploy OpenASE\n"), 0o600); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	scriptPath := filepath.Join(root, "scripts", "redeploy.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\necho deploy\n"), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}
	// #nosec G302 -- test fixture must be executable so packaging preserves the executable bit.
	if err := os.Chmod(scriptPath, 0o500); err != nil {
		t.Fatalf("chmod script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "references", "runbook.md"), []byte("# Runbook\n"), 0o600); err != nil {
		t.Fatalf("write reference: %v", err)
	}

	payload, err := buildSkillImportPayload(root, "", "user:cli", enableFlagValue(true, false))
	if err != nil {
		t.Fatalf("buildSkillImportPayload() error = %v", err)
	}
	if payload.Name != "deploy-openase" || payload.CreatedBy != "user:cli" || payload.IsEnabled == nil || !*payload.IsEnabled {
		t.Fatalf("unexpected payload metadata: %+v", payload)
	}
	if paths := []string{payload.Files[0].Path, payload.Files[1].Path, payload.Files[2].Path}; !slices.Equal(paths, []string{"SKILL.md", "references/runbook.md", "scripts/redeploy.sh"}) {
		t.Fatalf("unexpected payload paths: %+v", payload.Files)
	}

	script := payload.Files[2]
	if !script.IsExecutable {
		t.Fatalf("expected packaged script to stay executable: %+v", script)
	}
	content, err := base64.StdEncoding.DecodeString(script.ContentBase64)
	if err != nil {
		t.Fatalf("decode script payload: %v", err)
	}
	if string(content) != "#!/usr/bin/env bash\necho deploy\n" {
		t.Fatalf("unexpected packaged script content = %q", string(content))
	}
}
