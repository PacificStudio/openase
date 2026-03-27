package httpapi

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
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

func createPrimaryProjectRepo(t *testing.T, ctx context.Context, client *ent.Client, projectID uuid.UUID, repoRoot string) {
	t.Helper()

	if _, err := client.ProjectRepo.Create().
		SetProjectID(projectID).
		SetName(filepath.Base(repoRoot)).
		SetRepositoryURL(repoRoot).
		SetDefaultBranch("main").
		SetIsPrimary(true).
		Save(ctx); err != nil {
		t.Fatalf("create primary project repo: %v", err)
	}
}
