package httpapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
)

func createTestGitRepo(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	repository, err := git.PlainInit(repoRoot, false)
	if err != nil {
		t.Fatalf("git init repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("httpapi repo\n"), 0o600); err != nil {
		t.Fatalf("write repo seed file: %v", err)
	}
	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("load repo worktree: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("git add repo seed file: %v", err)
	}
	if _, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@openai.com",
			When:  time.Date(2026, 3, 31, 3, 0, 0, 0, time.UTC),
		},
	}); err != nil {
		t.Fatalf("git commit repo seed file: %v", err)
	}

	return repoRoot
}

func createPrimaryProjectRepo(ctx context.Context, t *testing.T, client *ent.Client, projectID uuid.UUID, repoRoot string) {
	t.Helper()

	if _, err := client.ProjectRepo.Create().
		SetProjectID(projectID).
		SetName(filepath.Base(repoRoot)).
		SetRepositoryURL(fmt.Sprintf("https://github.com/acme/%s.git", filepath.Base(repoRoot))).
		SetDefaultBranch("main").
		SetWorkspaceDirname(filepath.Base(repoRoot)).
		Save(ctx); err != nil {
		t.Fatalf("create project repo: %v", err)
	}
}

func createPrimaryProjectRepoForCheckout(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	projectID uuid.UUID,
	machineID uuid.UUID,
	repoRoot string,
) {
	t.Helper()

	projectRepo, err := client.ProjectRepo.Create().
		SetProjectID(projectID).
		SetName(filepath.Base(repoRoot)).
		SetRepositoryURL(fmt.Sprintf("https://github.com/acme/%s.git", filepath.Base(repoRoot))).
		SetDefaultBranch("main").
		SetWorkspaceDirname(filepath.Base(repoRoot)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}

	attachProjectRepoCheckout(ctx, t, client, projectRepo.ID, machineID, repoRoot)
}

func attachProjectRepoCheckout(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	projectRepoID uuid.UUID,
	machineID uuid.UUID,
	repoRoot string,
) {
	t.Helper()
	_ = ctx
	_ = client
	_ = projectRepoID
	_ = machineID
	_ = repoRoot
}

func attachPrimaryProjectRepoCheckout(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	projectID uuid.UUID,
	machineID uuid.UUID,
	repoRoot string,
) {
	t.Helper()

	projectRepo, err := client.ProjectRepo.Query().
		Where(entprojectrepo.ProjectIDEQ(projectID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("query project repo: %v", err)
	}

	attachProjectRepoCheckout(ctx, t, client, projectRepo.ID, machineID, repoRoot)
}
