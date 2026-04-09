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

func TestCaptureRunCompletionWorkspaceRepoHandlesUnbornHead(t *testing.T) {
	t.Parallel()

	repoPath, branch := createUnbornWorkspaceGitRepo(t)
	if err := os.WriteFile(filepath.Join(repoPath, "tracked.txt"), []byte("alpha\nbeta\n"), 0o600); err != nil {
		t.Fatalf("write tracked file: %v", err)
	}
	runWorkspaceGitCommand(t, "", "git", "-C", repoPath, "add", "tracked.txt")
	if err := os.WriteFile(filepath.Join(repoPath, "notes.txt"), []byte("gamma\n"), 0o600); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}

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
	if summary.FilesChanged != 2 || summary.Added != 3 || summary.Removed != 0 {
		t.Fatalf("unexpected unborn HEAD totals: %+v", summary)
	}

	filesByPath := make(map[string]runCompletionFileDiff, len(summary.Files))
	for _, file := range summary.Files {
		filesByPath[file.Path] = file
	}
	if filesByPath["tracked.txt"].Status != "added" || filesByPath["tracked.txt"].Added != 2 || filesByPath["tracked.txt"].Removed != 0 {
		t.Fatalf("unexpected tracked file diff: %+v", filesByPath["tracked.txt"])
	}
	if filesByPath["notes.txt"].Status != "untracked" || filesByPath["notes.txt"].Added != 1 || filesByPath["notes.txt"].Removed != 0 {
		t.Fatalf("unexpected untracked file diff: %+v", filesByPath["notes.txt"])
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

func createUnbornWorkspaceGitRepo(t *testing.T) (string, string) {
	t.Helper()

	repoPath := filepath.Join(t.TempDir(), "repo")
	runWorkspaceGitCommand(t, "", "git", "init", repoPath)
	runWorkspaceGitCommand(t, "", "git", "-C", repoPath, "symbolic-ref", "HEAD", "refs/heads/main")
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
