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

func TestSummarizeConversationWorkspaceRepoHandlesUnbornHEAD(t *testing.T) {
	t.Parallel()

	repoPath, branch := createConversationUnbornGitRepo(t, map[string]string{"notes.txt": "note one\nnote two\n"})
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
	if summary.FilesChanged != 1 || summary.Added != 2 || summary.Removed != 0 {
		t.Fatalf("unexpected unborn HEAD totals: %+v", summary)
	}
	if len(summary.Files) != 1 || summary.Files[0].Path != "notes.txt" || summary.Files[0].Status != ProjectConversationWorkspaceFileStatusUntracked || summary.Files[0].Added != 2 || summary.Files[0].Removed != 0 {
		t.Fatalf("unexpected unborn HEAD file diff: %+v", summary.Files)
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

func createConversationUnbornGitRepo(t *testing.T, files map[string]string) (string, string) {
	t.Helper()

	repoPath := filepath.Join(t.TempDir(), "repo")
	runConversationGitCommand(t, "", "git", "init", repoPath)
	runConversationGitCommand(t, "", "git", "-C", repoPath, "symbolic-ref", "HEAD", "refs/heads/main")
	for path, content := range files {
		absolutePath := filepath.Join(repoPath, path)
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
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
