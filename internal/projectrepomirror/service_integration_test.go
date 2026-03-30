package projectrepomirror

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	_ "github.com/lib/pq"
)

func TestServicePrepareMarkStaleVerifyAndDelete(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	project, machine, projectRepo := createMirrorTestFixtures(ctx, t, client)
	machine, err := client.Machine.UpdateOneID(machine.ID).
		SetMirrorRoot(filepath.Join(t.TempDir(), "mirrors")).
		Save(ctx)
	if err != nil {
		t.Fatalf("set machine mirror_root: %v", err)
	}
	sourceRepoPath, headCommit := createGitRepository(t)
	if _, err := client.ProjectRepo.UpdateOneID(projectRepo.ID).
		SetRepositoryURL(sourceRepoPath).
		SetDefaultBranch("master").
		Save(ctx); err != nil {
		t.Fatalf("update project repo remote: %v", err)
	}

	svc := NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	base := time.Date(2026, 3, 29, 15, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return base }

	prepared, err := svc.Prepare(ctx, PrepareInput{
		ProjectRepoID: projectRepo.ID,
		MachineID:     machine.ID,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if prepared.ProjectID != project.ID || prepared.State != domain.ProjectRepoMirrorStateReady {
		t.Fatalf("Prepare() = %+v", prepared)
	}
	expectedMirrorPath := filepath.Join(machine.MirrorRoot, "acme", "openase", "backend")
	if prepared.LocalPath != expectedMirrorPath {
		t.Fatalf("prepared local_path = %q, want %q", prepared.LocalPath, expectedMirrorPath)
	}
	if prepared.HeadCommit == nil || *prepared.HeadCommit != headCommit {
		t.Fatalf("prepared head commit = %v, want %s", prepared.HeadCommit, headCommit)
	}

	if err := svc.MarkStaleMirrors(ctx, time.Hour); err != nil {
		t.Fatalf("MarkStaleMirrors() unexpected error = %v", err)
	}

	svc.now = func() time.Time { return base.Add(2 * time.Hour) }
	if err := svc.MarkStaleMirrors(ctx, time.Hour); err != nil {
		t.Fatalf("MarkStaleMirrors() error = %v", err)
	}

	listed, err := svc.List(ctx, ListFilter{
		ProjectID:     project.ID,
		ProjectRepoID: projectRepo.ID,
		MachineID:     &machine.ID,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 || listed[0].State != domain.ProjectRepoMirrorStateStale {
		t.Fatalf("List() mirrors = %+v", listed)
	}

	verified, err := svc.Verify(ctx, VerifyInput{
		ProjectRepoID: projectRepo.ID,
		MachineID:     machine.ID,
	})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if verified.State != domain.ProjectRepoMirrorStateReady || verified.LastVerifiedAt == nil || !verified.LastVerifiedAt.Equal(base.Add(2*time.Hour)) {
		t.Fatalf("Verify() = %+v", verified)
	}

	deleted, err := svc.Delete(ctx, DeleteInput{
		ProjectRepoID: projectRepo.ID,
		MachineID:     machine.ID,
	})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted.State != domain.ProjectRepoMirrorStateMissing {
		t.Fatalf("Delete() = %+v", deleted)
	}
	if _, err := os.Stat(expectedMirrorPath); !os.IsNotExist(err) {
		t.Fatalf("mirror path still exists after delete: %v", err)
	}
}

func TestServiceRegisterExisting(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	project, machine, projectRepo := createMirrorTestFixtures(ctx, t, client)
	sourceRepoPath, headCommit := createGitRepository(t)
	if _, err := client.ProjectRepo.UpdateOneID(projectRepo.ID).
		SetRepositoryURL(sourceRepoPath).
		SetDefaultBranch("master").
		Save(ctx); err != nil {
		t.Fatalf("update project repo remote: %v", err)
	}

	svc := NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	base := time.Date(2026, 3, 29, 16, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return base }

	registered, err := svc.RegisterExisting(ctx, RegisterExistingInput{
		ProjectRepoID: projectRepo.ID,
		MachineID:     machine.ID,
		LocalPath:     sourceRepoPath,
	})
	if err != nil {
		t.Fatalf("RegisterExisting() error = %v", err)
	}
	if registered.ProjectID != project.ID || registered.State != domain.ProjectRepoMirrorStateReady {
		t.Fatalf("RegisterExisting() = %+v", registered)
	}
	if registered.HeadCommit == nil || *registered.HeadCommit != headCommit {
		t.Fatalf("registered head commit = %v, want %s", registered.HeadCommit, headCommit)
	}
}

func TestServicePrepareDerivesRemoteMirrorPathFromWorkspaceRoot(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	_, machine, projectRepo := createMirrorTestFixtures(ctx, t, client)
	machine, err := client.Machine.UpdateOneID(machine.ID).
		SetName("builder").
		SetHost("10.0.0.12").
		SetWorkspaceRoot(filepath.Join(t.TempDir(), "workspace")).
		ClearMirrorRoot().
		Save(ctx)
	if err != nil {
		t.Fatalf("update machine: %v", err)
	}
	sourceRepoPath, _ := createGitRepository(t)
	if _, err := client.ProjectRepo.UpdateOneID(projectRepo.ID).
		SetRepositoryURL(sourceRepoPath).
		SetDefaultBranch("master").
		Save(ctx); err != nil {
		t.Fatalf("update project repo remote: %v", err)
	}

	svc := NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	prepared, err := svc.Prepare(ctx, PrepareInput{
		ProjectRepoID: projectRepo.ID,
		MachineID:     machine.ID,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	expected := filepath.Join(filepath.Dir(machine.WorkspaceRoot), "mirrors", "acme", "openase", "backend")
	if prepared.LocalPath != expected {
		t.Fatalf("Prepare() local_path = %q, want %q", prepared.LocalPath, expected)
	}
}

func TestServiceRegisterExistingRequiresLocalPath(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	_, machine, projectRepo := createMirrorTestFixtures(ctx, t, client)
	svc := NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := svc.RegisterExisting(ctx, RegisterExistingInput{
		ProjectRepoID: projectRepo.ID,
		MachineID:     machine.ID,
	}); err == nil || err.Error() != "project repo mirror input is invalid: local_path must not be empty" {
		t.Fatalf("RegisterExisting() missing local_path error = %v", err)
	}
}

func createMirrorTestFixtures(ctx context.Context, t *testing.T, client *ent.Client) (*ent.Project, *ent.Machine, *ent.ProjectRepo) {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Acme").
		SetSlug("acme").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("127.0.0.1").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	projectRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("backend").
		SetRepositoryURL("https://example.invalid/backend.git").
		SetDefaultBranch("master").
		SetWorkspaceDirname("backend").
		SetIsPrimary(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}
	return project, machine, projectRepo
}

func createGitRepository(t *testing.T) (string, string) {
	t.Helper()

	repoPath := filepath.Join(t.TempDir(), "remote")
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		t.Fatalf("create git repo dir: %v", err)
	}

	repository, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("mirror test\n"), 0o600); err != nil {
		t.Fatalf("write git file: %v", err)
	}

	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("git worktree: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	hash, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@openai.com",
			When:  time.Date(2026, 3, 29, 14, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if _, err := repository.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{repoPath},
	}); err != nil {
		t.Fatalf("git create remote: %v", err)
	}

	return repoPath, hash.String()
}

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	port := freePort(t)
	dataDir := t.TempDir()
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(port).
			Username("postgres").
			Password("postgres").
			Database("openase").
			RuntimePath(filepath.Join(dataDir, "runtime")).
			BinariesPath(filepath.Join(dataDir, "binaries")).
			DataPath(filepath.Join(dataDir, "data")),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Stop(); err != nil {
			t.Errorf("stop embedded postgres: %v", err)
		}
	})

	dsn := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/openase?sslmode=disable", port)
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open ent client: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client: %v", err)
		}
	})

	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return client
}

func freePort(t *testing.T) uint32 {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate free port: %v", err)
	}

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCP address, got %T", listener.Addr())
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	if tcpAddr.Port < 0 || tcpAddr.Port > math.MaxUint16 {
		t.Fatalf("expected TCP port in uint16 range, got %d", tcpAddr.Port)
	}
	return uint32(tcpAddr.Port) //nolint:gosec // validated above to fit the TCP port range
}
