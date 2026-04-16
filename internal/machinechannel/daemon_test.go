package machinechannel

import (
	"os"
	"path/filepath"
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
)

func TestDaemonToolInventoryUsesScopedAgentCLIPaths(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	claudePath := writeExecutable(t, filepath.Join(tmpDir, "claude"))
	codexPath := writeExecutable(t, filepath.Join(tmpDir, "codex"))
	geminiPath := writeExecutable(t, filepath.Join(tmpDir, "gemini"))
	daemon := NewDaemon(nil)
	daemon.lookPath = func(command string) (string, error) {
		t.Fatalf("unexpected lookPath command %q", command)
		return "", nil
	}

	tools := daemon.toolInventory(domain.DaemonConfig{
		AgentCLIPath: "/usr/local/bin/legacy",
		AgentCLIPaths: map[string]string{
			"claude-code-cli":  claudePath,
			"codex-app-server": codexPath,
			"gemini-cli":       geminiPath,
		},
	})
	if len(tools) != 3 {
		t.Fatalf("toolInventory() len = %d, want 3", len(tools))
	}
	for _, tool := range tools {
		if !tool.Installed || !tool.Ready {
			t.Fatalf("toolInventory() entry = %+v, want installed+ready", tool)
		}
	}
}

func TestDaemonToolInventoryFallsBackToLegacyPathOnlyWhenNoScopedPaths(t *testing.T) {
	t.Parallel()

	legacyPath := writeExecutable(t, filepath.Join(t.TempDir(), "legacy"))
	daemon := NewDaemon(nil)
	daemon.lookPath = func(command string) (string, error) {
		if command != "claude" && command != "gemini" {
			t.Fatalf("unexpected lookPath command %q", command)
		}
		return command, nil
	}

	tools := daemon.toolInventory(domain.DaemonConfig{AgentCLIPath: legacyPath})
	if !tools[0].Installed || !tools[1].Installed || !tools[2].Installed {
		t.Fatalf("toolInventory() legacy fallback = %+v", tools)
	}
}

func writeExecutable(t *testing.T, path string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
	return path
}
