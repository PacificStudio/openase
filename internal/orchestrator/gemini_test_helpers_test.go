package orchestrator

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func ensureGeminiCLIProbePath(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("LookPath(node) returned error: %v", err)
	}

	geminiPath := filepath.Join(tempDir, "gemini")
	if err := os.Symlink(nodePath, geminiPath); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}

	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
