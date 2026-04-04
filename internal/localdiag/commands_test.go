package localdiag

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectReportsReadyMissingAndVersionFailure(t *testing.T) {
	reports := Inspect(context.Background(), []CommandSpec{
		{ID: "git", Name: "Git", Command: "git"},
		{ID: "codex", Name: "OpenAI Codex", Command: "codex"},
		{ID: "claude", Name: "Claude Code", Command: "claude"},
	}, Options{
		LookPath: func(name string) (string, error) {
			switch name {
			case "git":
				return "/usr/bin/git", nil
			case "codex":
				return "/usr/local/bin/codex", nil
			default:
				return "", os.ErrNotExist
			}
		},
		RunCommand: func(_ context.Context, name string, _ ...string) (string, error) {
			switch filepath.Base(name) {
			case "git":
				return "git version 2.48.1\n", nil
			case "codex":
				return "", context.DeadlineExceeded
			default:
				return "", nil
			}
		},
	})

	if len(reports) != 3 {
		t.Fatalf("Inspect() returned %d reports", len(reports))
	}
	if reports[0].Status != StatusReady || reports[0].Version != "git version 2.48.1" {
		t.Fatalf("git report = %+v", reports[0])
	}
	if reports[1].Status != StatusVersionError || !strings.Contains(reports[1].Error, "context deadline exceeded") {
		t.Fatalf("codex report = %+v", reports[1])
	}
	if reports[2].Status != StatusMissing {
		t.Fatalf("claude report = %+v", reports[2])
	}
}

func TestSetupCommandSpecsIncludesBuiltinProviders(t *testing.T) {
	specs := SetupCommandSpecs()
	if len(specs) < 4 {
		t.Fatalf("SetupCommandSpecs() returned %d specs", len(specs))
	}
	if specs[0].Command != "git" {
		t.Fatalf("first spec = %+v", specs[0])
	}
}
