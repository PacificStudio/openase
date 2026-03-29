package httpapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entprojectrepomirror "github.com/BetterAndBetterII/openase/ent/projectrepomirror"
	"github.com/google/uuid"
)

func createTestGitRepo(t *testing.T) string {
	t.Helper()

	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("create git marker: %v", err)
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
