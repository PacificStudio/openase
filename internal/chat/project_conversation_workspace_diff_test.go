package chat

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func TestSummarizeConversationWorkspaceRepoHandlesUnbornHead(t *testing.T) {
	t.Parallel()

	repoPath, branch := createConversationUnbornGitRepo(t)
	trackedPath := filepath.Join(repoPath, "tracked.txt")
	if err := os.WriteFile(trackedPath, []byte("alpha\nbeta\n"), 0o600); err != nil {
		t.Fatalf("write tracked file: %v", err)
	}
	runConversationGitCommand(t, "", "git", "-C", repoPath, "add", "tracked.txt")
	if err := os.WriteFile(filepath.Join(repoPath, "notes.txt"), []byte("gamma\n"), 0o600); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}

	service := &ProjectConversationService{}
	summary, err := service.summarizeConversationWorkspaceRepo(
		context.Background(),
		catalogdomain.Machine{Host: catalogdomain.LocalMachineHost},
		projectConversationWorkspaceRepoLocation{name: "backend", repoPath: repoPath, relativePath: "backend"},
	)
	if err != nil {
		t.Fatalf("summarizeConversationWorkspaceRepo() error = %v", err)
	}
	if !summary.Dirty || summary.Branch != branch {
		t.Fatalf("unexpected repo summary header: %+v", summary)
	}
	if summary.Name != "backend" || summary.Path != "backend" {
		t.Fatalf("unexpected repo identity: %+v", summary)
	}
	if summary.FilesChanged != 2 || summary.Added != 3 || summary.Removed != 0 {
		t.Fatalf("unexpected unborn HEAD totals: %+v", summary)
	}

	filesByPath := make(map[string]ProjectConversationWorkspaceFileDiff, len(summary.Files))
	for _, file := range summary.Files {
		filesByPath[file.Path] = file
	}
	if filesByPath["tracked.txt"].Status != ProjectConversationWorkspaceFileStatusAdded ||
		!filesByPath["tracked.txt"].Staged ||
		filesByPath["tracked.txt"].Unstaged ||
		filesByPath["tracked.txt"].Added != 2 ||
		filesByPath["tracked.txt"].Removed != 0 {
		t.Fatalf("unexpected tracked file diff: %+v", filesByPath["tracked.txt"])
	}
	if filesByPath["notes.txt"].Status != ProjectConversationWorkspaceFileStatusUntracked ||
		filesByPath["notes.txt"].Staged ||
		!filesByPath["notes.txt"].Unstaged ||
		filesByPath["notes.txt"].Added != 1 ||
		filesByPath["notes.txt"].Removed != 0 {
		t.Fatalf("unexpected untracked file diff: %+v", filesByPath["notes.txt"])
	}
}

func TestSummarizeConversationWorkspaceRepoMissingGitDirectoryStillSkipped(t *testing.T) {
	t.Parallel()

	repoPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("not a repo\n"), 0o600); err != nil {
		t.Fatalf("write plain directory file: %v", err)
	}
	service := &ProjectConversationService{}

	summary, err := service.summarizeConversationWorkspaceRepo(
		context.Background(),
		catalogdomain.Machine{Host: catalogdomain.LocalMachineHost},
		projectConversationWorkspaceRepoLocation{name: "backend", repoPath: repoPath, relativePath: "backend"},
	)
	if err != nil {
		t.Fatalf("summarizeConversationWorkspaceRepo() error = %v", err)
	}
	if summary.Dirty || summary.Branch != "" || len(summary.Files) != 0 || summary.FilesChanged != 0 || summary.Added != 0 || summary.Removed != 0 {
		t.Fatalf("expected non-git directory to be skipped, got %+v", summary)
	}
}

func createConversationUnbornGitRepo(t *testing.T) (string, string) {
	t.Helper()

	repoPath := filepath.Join(t.TempDir(), "repo")
	runConversationGitCommand(t, "", "git", "init", repoPath)
	runConversationGitCommand(t, "", "git", "-C", repoPath, "symbolic-ref", "HEAD", "refs/heads/main")
	branch := strings.TrimSpace(runConversationGitCommand(t, "", "git", "-C", repoPath, "symbolic-ref", "-q", "--short", "HEAD"))
	return repoPath, branch
}

func runConversationGitCommand(t *testing.T, dir string, name string, args ...string) string {
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
