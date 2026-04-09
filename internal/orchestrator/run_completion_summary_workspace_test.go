package orchestrator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func TestCaptureRunCompletionWorkspaceRepoHandlesUnbornHead(t *testing.T) {
	t.Parallel()

	workspaceRoot := t.TempDir()
	repoPath := filepath.Join(workspaceRoot, "repo")
	runCompletionGit(t, workspaceRoot, "init", "repo")

	if err := os.WriteFile(filepath.Join(repoPath, "tracked.txt"), []byte("alpha\nbeta\n"), 0o600); err != nil {
		t.Fatalf("write tracked file: %v", err)
	}
	runCompletionGit(t, repoPath, "add", "tracked.txt")
	if err := os.WriteFile(filepath.Join(repoPath, "notes.txt"), []byte("gamma\n"), 0o600); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}

	coordinator := newRuntimeCompletionSummaryCoordinator(nil, nil, nil, nil, nil, nil, nil, nil, 0)
	repoSummary, err := coordinator.captureRunCompletionWorkspaceRepo(
		context.Background(),
		catalogdomain.Machine{Name: catalogdomain.LocalMachineName, Host: catalogdomain.LocalMachineHost},
		workspaceRoot,
		&ent.TicketRepoWorkspace{RepoPath: repoPath},
	)
	if err != nil {
		t.Fatalf("capture workspace repo: %v", err)
	}

	if !repoSummary.Dirty {
		t.Fatalf("expected repo to be dirty, got %+v", repoSummary)
	}
	if repoSummary.Branch == "" {
		t.Fatalf("expected unborn repo branch name to be captured, got %+v", repoSummary)
	}
	if repoSummary.Path != "repo" {
		t.Fatalf("expected relative repo path, got %q", repoSummary.Path)
	}
	if repoSummary.FilesChanged != 2 {
		t.Fatalf("expected two changed files, got %+v", repoSummary)
	}
	if repoSummary.Added != 3 || repoSummary.Removed != 0 {
		t.Fatalf("expected added/removed counts from unborn repo diff, got %+v", repoSummary)
	}

	filesByPath := make(map[string]runCompletionFileDiff, len(repoSummary.Files))
	for _, file := range repoSummary.Files {
		filesByPath[file.Path] = file
	}
	if filesByPath["tracked.txt"].Status != "added" || filesByPath["tracked.txt"].Added != 2 {
		t.Fatalf("expected staged file stats to survive unborn HEAD, got %+v", filesByPath["tracked.txt"])
	}
	if filesByPath["notes.txt"].Status != "untracked" || filesByPath["notes.txt"].Added != 1 {
		t.Fatalf("expected untracked file stats, got %+v", filesByPath["notes.txt"])
	}
}

func runCompletionGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	//nolint:gosec // Test helper intentionally invokes local git with controlled arguments.
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run git %v in %s: %v\n%s", args, dir, err, string(output))
	}
}
