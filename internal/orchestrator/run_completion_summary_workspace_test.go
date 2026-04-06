package orchestrator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func TestCaptureRunCompletionWorkspaceRepoHandlesUnbornHEAD(t *testing.T) {
	t.Parallel()

	repoPath, branch := createUnbornWorkspaceGitRepo(t, map[string]string{"notes.txt": "note one\nnote two\n"})
	coordinator := newRuntimeCompletionSummaryCoordinator(nil, nil, nil, nil, nil, nil, nil, nil, 0)

	summary, err := coordinator.captureRunCompletionWorkspaceRepo(
		context.Background(),
		catalogdomain.Machine{Host: catalogdomain.LocalMachineHost},
		filepath.Dir(repoPath),
		&ent.TicketRepoWorkspace{RepoPath: repoPath},
	)
	if err != nil {
		t.Fatalf("captureRunCompletionWorkspaceRepo() error = %v", err)
	}
	if !summary.Dirty || summary.Branch != branch {
		t.Fatalf("unexpected repo summary header: %+v", summary)
	}
	if summary.Path != filepath.Base(repoPath) || summary.Name != filepath.Base(repoPath) {
		t.Fatalf("unexpected repo identity: %+v", summary)
	}
	if summary.FilesChanged != 1 || summary.Added != 2 || summary.Removed != 0 {
		t.Fatalf("unexpected unborn HEAD totals: %+v", summary)
	}
	if len(summary.Files) != 1 || summary.Files[0].Path != "notes.txt" || summary.Files[0].Status != "untracked" || summary.Files[0].Added != 2 || summary.Files[0].Removed != 0 {
		t.Fatalf("unexpected unborn HEAD file diff: %+v", summary.Files)
	}
}

func TestCaptureRunCompletionWorkspaceRepoMissingWorkspaceStillSkipped(t *testing.T) {
	t.Parallel()

	repoPath := filepath.Join(t.TempDir(), "missing")
	coordinator := newRuntimeCompletionSummaryCoordinator(nil, nil, nil, nil, nil, nil, nil, nil, 0)

	summary, err := coordinator.captureRunCompletionWorkspaceRepo(
		context.Background(),
		catalogdomain.Machine{Host: catalogdomain.LocalMachineHost},
		filepath.Dir(repoPath),
		&ent.TicketRepoWorkspace{RepoPath: repoPath},
	)
	if err != nil {
		t.Fatalf("captureRunCompletionWorkspaceRepo() error = %v", err)
	}
	if summary.Dirty || summary.Branch != "" || len(summary.Files) != 0 || summary.FilesChanged != 0 || summary.Added != 0 || summary.Removed != 0 {
		t.Fatalf("expected missing workspace to be skipped, got %+v", summary)
	}
}

func createUnbornWorkspaceGitRepo(t *testing.T, files map[string]string) (string, string) {
	t.Helper()

	repoPath := filepath.Join(t.TempDir(), "repo")
	runWorkspaceGitCommand(t, "", "git", "init", repoPath)
	runWorkspaceGitCommand(t, "", "git", "-C", repoPath, "symbolic-ref", "HEAD", "refs/heads/main")
	for path, content := range files {
		absolutePath := filepath.Join(repoPath, path)
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	branch := strings.TrimSpace(runWorkspaceGitCommand(t, "", "git", "-C", repoPath, "symbolic-ref", "-q", "--short", "HEAD"))
	return repoPath, branch
}

func runWorkspaceGitCommand(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...) // #nosec G204 -- test helper executes fixed commands.
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, string(output))
	}
	return string(output)
}
