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
	entprojectrepomirror "github.com/BetterAndBetterII/openase/ent/projectrepomirror"
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
		SetIsPrimary(true).
		Save(ctx); err != nil {
		t.Fatalf("create primary project repo: %v", err)
	}
}

func createPrimaryProjectRepoWithMirror(
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
		SetIsPrimary(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create primary project repo: %v", err)
	}

	createReadyProjectRepoMirror(ctx, t, client, projectRepo.ID, machineID, repoRoot)
}

func createReadyProjectRepoMirror(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	projectRepoID uuid.UUID,
	machineID uuid.UUID,
	repoRoot string,
) {
	t.Helper()

	if _, err := client.ProjectRepoMirror.Create().
		SetProjectRepoID(projectRepoID).
		SetMachineID(machineID).
		SetLocalPath(repoRoot).
		SetState(entprojectrepomirror.StateReady).
		Save(ctx); err != nil {
		t.Fatalf("create ready project repo mirror: %v", err)
	}
}

func createReadyPrimaryProjectRepoMirror(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	projectID uuid.UUID,
	machineID uuid.UUID,
	repoRoot string,
) {
	t.Helper()

	projectRepo, err := client.ProjectRepo.Query().
		Where(
			entprojectrepo.ProjectIDEQ(projectID),
			entprojectrepo.IsPrimary(true),
		).
		Only(ctx)
	if err != nil {
		t.Fatalf("query primary project repo: %v", err)
	}

	createReadyProjectRepoMirror(ctx, t, client, projectRepo.ID, machineID, repoRoot)
}
