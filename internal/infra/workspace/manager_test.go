package workspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestParseSetupRequestRejectsNonCanonicalBranchName(t *testing.T) {
	rawBranch := "feature/custom"
	_, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-33",
		Repos: []RepoInput{
			{
				Name:          "backend",
				RepositoryURL: "/tmp/backend.git",
				BranchName:    &rawBranch,
			},
		},
	})
	if err == nil {
		t.Fatal("expected parse to fail for non-canonical branch name")
	}
	if !strings.Contains(err.Error(), `must equal "agent/codex-01/ASE-33"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseSetupRequestAllowsEmptyRepos(t *testing.T) {
	request, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-33",
	})
	if err != nil {
		t.Fatalf("expected parse to allow empty repos: %v", err)
	}
	if len(request.Repos) != 0 {
		t.Fatalf("expected no repos, got %+v", request.Repos)
	}
	if request.BranchName != "agent/codex-01/ASE-33" {
		t.Fatalf("unexpected branch name %q", request.BranchName)
	}
}

func TestManagerPrepareCreatesJointWorkspaceWithFeatureBranch(t *testing.T) {
	backendRepoPath, _ := createRemoteRepo(t, "main", map[string]string{
		"README.md": "backend",
	})
	frontendRepoPath, _ := createRemoteRepo(t, "main", map[string]string{
		"package.json": "{}",
	})

	clonePath := "services/frontend"
	request, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-33",
		Repos: []RepoInput{
			{
				Name:          "backend",
				RepositoryURL: backendRepoPath,
				DefaultBranch: "main",
			},
			{
				Name:          "frontend",
				RepositoryURL: frontendRepoPath,
				DefaultBranch: "main",
				ClonePath:     &clonePath,
			},
		},
	})
	if err != nil {
		t.Fatalf("parse setup request: %v", err)
	}

	manager := NewManager()
	workspace, err := manager.Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}

	expectedWorkspacePath := filepath.Join(request.WorkspaceRoot, "acme", "payments", "ASE-33")
	if workspace.Path != expectedWorkspacePath {
		t.Fatalf("expected workspace path %s, got %s", expectedWorkspacePath, workspace.Path)
	}
	if workspace.BranchName != "agent/codex-01/ASE-33" {
		t.Fatalf("expected branch name agent/codex-01/ASE-33, got %s", workspace.BranchName)
	}

	backendClonePath := filepath.Join(workspace.Path, "backend")
	frontendClonePath := filepath.Join(workspace.Path, filepath.FromSlash(clonePath))

	assertFileExists(t, filepath.Join(backendClonePath, "README.md"))
	assertFileExists(t, filepath.Join(frontendClonePath, "package.json"))
	assertHeadBranch(t, backendClonePath, "agent/codex-01/ASE-33")
	assertHeadBranch(t, frontendClonePath, "agent/codex-01/ASE-33")
}

func TestManagerPrepareFetchesExistingClone(t *testing.T) {
	repositoryURL, initialHash := createRemoteRepo(t, "main", map[string]string{
		"README.md": "first",
	})

	request, err := ParseSetupRequest(SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		AgentName:        "codex-01",
		TicketIdentifier: "ASE-33",
		Repos: []RepoInput{
			{
				Name:          "backend",
				RepositoryURL: repositoryURL,
				DefaultBranch: "main",
			},
		},
	})
	if err != nil {
		t.Fatalf("parse setup request: %v", err)
	}

	manager := NewManager()
	workspace, err := manager.Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("prepare workspace first run: %v", err)
	}

	backendClonePath := filepath.Join(workspace.Path, "backend")
	assertRemoteBranchHash(t, backendClonePath, "main", initialHash)

	updatedHash := appendCommit(t, repositoryURL, "main", "README.md", "second")
	workspace, err = manager.Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("prepare workspace second run: %v", err)
	}

	assertHeadBranch(t, backendClonePath, "agent/codex-01/ASE-33")
	assertRemoteBranchHash(t, backendClonePath, "main", updatedHash)
}

func TestTicketWorkspacePathAndPattern(t *testing.T) {
	workspacePath, err := TicketWorkspacePath("/srv/openase/workspace", "acme", "payments", "ASE-42")
	if err != nil {
		t.Fatalf("derive workspace path: %v", err)
	}
	if workspacePath != filepath.Join("/srv/openase/workspace", "acme", "payments", "ASE-42") {
		t.Fatalf("unexpected workspace path %q", workspacePath)
	}

	pattern, err := TicketWorkspacePattern(LocalWorkspacePatternRoot, "acme", "payments")
	if err != nil {
		t.Fatalf("derive workspace pattern: %v", err)
	}
	if pattern != filepath.Join(LocalWorkspacePatternRoot, "acme", "payments", ticketPlaceholder) {
		t.Fatalf("unexpected workspace pattern %q", pattern)
	}
}

func createRemoteRepo(t *testing.T, defaultBranch string, files map[string]string) (string, plumbing.Hash) {
	t.Helper()

	repoPath := t.TempDir()
	repository, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("init repository: %v", err)
	}

	hash := commitFiles(t, repository, repoPath, files, "initial commit")
	setDefaultBranch(t, repository, defaultBranch, hash)

	return repoPath, hash
}

func appendCommit(t *testing.T, repoPath string, branch string, filePath string, content string) plumbing.Hash {
	t.Helper()

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}

	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("open worktree: %v", err)
	}

	branchRef := plumbing.NewBranchReferenceName(branch)
	head, err := repository.Head()
	if err != nil {
		t.Fatalf("load head: %v", err)
	}
	if head.Name() != branchRef {
		if err := worktree.Checkout(&git.CheckoutOptions{Branch: branchRef}); err != nil {
			t.Fatalf("checkout branch %s: %v", branch, err)
		}
	}

	absoluteFilePath := filepath.Join(repoPath, filePath)
	if err := os.MkdirAll(filepath.Dir(absoluteFilePath), 0o750); err != nil {
		t.Fatalf("create directories for %s: %v", filePath, err)
	}
	if err := os.WriteFile(absoluteFilePath, []byte(content), 0o600); err != nil {
		t.Fatalf("write file %s: %v", filePath, err)
	}

	if _, err := worktree.Add(filePath); err != nil {
		t.Fatalf("add file %s: %v", filePath, err)
	}

	hash, err := worktree.Commit("update commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@example.com",
			When:  time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("commit update: %v", err)
	}

	return hash
}

func commitFiles(t *testing.T, repository *git.Repository, repoPath string, files map[string]string, message string) plumbing.Hash {
	t.Helper()

	for relativePath, content := range files {
		absolutePath := filepath.Join(repoPath, relativePath)
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
			t.Fatalf("create directory for %s: %v", relativePath, err)
		}
		if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
			t.Fatalf("write file %s: %v", relativePath, err)
		}
	}

	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("open worktree: %v", err)
	}
	if err := worktree.AddGlob("."); err != nil {
		t.Fatalf("add files: %v", err)
	}

	hash, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@example.com",
			When:  time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("commit files: %v", err)
	}

	return hash
}

func setDefaultBranch(t *testing.T, repository *git.Repository, branch string, hash plumbing.Hash) {
	t.Helper()

	branchRef := plumbing.NewBranchReferenceName(branch)
	if err := repository.Storer.SetReference(plumbing.NewHashReference(branchRef, hash)); err != nil {
		t.Fatalf("set branch %s: %v", branch, err)
	}
	if err := repository.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, branchRef)); err != nil {
		t.Fatalf("set HEAD to %s: %v", branch, err)
	}

	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("open worktree: %v", err)
	}
	if err := worktree.Checkout(&git.CheckoutOptions{Branch: branchRef, Force: true}); err != nil {
		t.Fatalf("checkout branch %s: %v", branch, err)
	}
}

func assertFileExists(t *testing.T, filePath string) {
	t.Helper()

	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("expected %s to exist: %v", filePath, err)
	}
}

func assertHeadBranch(t *testing.T, repoPath string, expectedBranch string) {
	t.Helper()

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repository %s: %v", repoPath, err)
	}

	head, err := repository.Head()
	if err != nil {
		t.Fatalf("load head for %s: %v", repoPath, err)
	}
	if head.Name().Short() != expectedBranch {
		t.Fatalf("expected branch %s, got %s", expectedBranch, head.Name().Short())
	}
}

func assertRemoteBranchHash(t *testing.T, repoPath string, branch string, expectedHash plumbing.Hash) {
	t.Helper()

	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repository %s: %v", repoPath, err)
	}

	ref, err := repository.Reference(plumbing.NewRemoteReferenceName("origin", branch), true)
	if err != nil {
		t.Fatalf("load remote branch %s: %v", branch, err)
	}
	if ref.Hash() != expectedHash {
		t.Fatalf("expected remote branch hash %s, got %s", expectedHash, ref.Hash())
	}
}
