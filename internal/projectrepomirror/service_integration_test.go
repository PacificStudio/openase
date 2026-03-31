package projectrepomirror

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entprojectrepomirror "github.com/BetterAndBetterII/openase/ent/projectrepomirror"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	githubauthrepo "github.com/BetterAndBetterII/openase/internal/repo/githubauth"
	githubauthservice "github.com/BetterAndBetterII/openase/internal/service/githubauth"
	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
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
		SetSSHUser("openase").
		SetSSHKeyPath("keys/builder").
		SetWorkspaceRoot(filepath.Join(t.TempDir(), "workspace")).
		ClearMirrorRoot().
		Save(ctx)
	if err != nil {
		t.Fatalf("update machine: %v", err)
	}
	sourceRepoPath, headCommit := createGitRepository(t)
	if _, err := client.ProjectRepo.UpdateOneID(projectRepo.ID).
		SetRepositoryURL(sourceRepoPath).
		SetDefaultBranch("master").
		Save(ctx); err != nil {
		t.Fatalf("update project repo remote: %v", err)
	}

	svc := NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	sshSession := &mirrorTestSSHSession{output: []byte(headCommit + "\n")}
	svc.ConfigureSSHPool(sshinfra.NewPool("/tmp/openase",
		sshinfra.WithDialer(&mirrorTestSSHDialer{clients: []sshinfra.Client{
			&mirrorTestSSHClient{session: sshSession},
		}}),
		sshinfra.WithReadFile(func(string) ([]byte, error) {
			return []byte("key"), nil
		}),
	))
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
	if prepared.HeadCommit == nil || *prepared.HeadCommit != headCommit {
		t.Fatalf("Prepare() head_commit = %v, want %q", prepared.HeadCommit, headCommit)
	}
	if !strings.Contains(sshSession.command, expected) {
		t.Fatalf("remote mirror command = %q, want path %q", sshSession.command, expected)
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

func TestServiceEnsureReadDoesNotPrepareMissingGitHubMirrorWithoutCredential(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	_, machine, projectRepo := createMirrorTestFixtures(ctx, t, client)
	if _, err := client.ProjectRepo.UpdateOneID(projectRepo.ID).
		SetRepositoryURL("https://github.com/GrandCX/private-repo.git").
		SetDefaultBranch("main").
		Save(ctx); err != nil {
		t.Fatalf("update project repo remote: %v", err)
	}

	svc := NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := svc.Ensure(ctx, EnsureInput{
		ProjectRepoID: projectRepo.ID,
		MachineID:     machine.ID,
		Operation:     EnsureOperationRead,
	}); !errors.Is(err, ErrMirrorNotReady) {
		t.Fatalf("Ensure(read) error = %v, want %v", err, ErrMirrorNotReady)
	}

	mirrorCount, err := client.ProjectRepoMirror.Query().
		Where(entprojectrepomirror.ProjectRepoIDEQ(projectRepo.ID)).
		Count(ctx)
	if err != nil {
		t.Fatalf("count mirrors after Ensure(read): %v", err)
	}
	if mirrorCount != 0 {
		t.Fatalf("Ensure(read) unexpectedly created %d mirrors", mirrorCount)
	}
}

func TestServicePrepareRemoteGitHubMirrorProjectsManagedCredential(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	project, machine, projectRepo := createMirrorTestFixtures(ctx, t, client)
	machine, err := client.Machine.UpdateOneID(machine.ID).
		SetName("builder").
		SetHost("10.0.0.12").
		SetSSHUser("openase").
		SetSSHKeyPath("keys/builder").
		SetWorkspaceRoot(filepath.Join(t.TempDir(), "workspace")).
		ClearMirrorRoot().
		Save(ctx)
	if err != nil {
		t.Fatalf("update machine: %v", err)
	}
	if _, err := client.ProjectRepo.UpdateOneID(projectRepo.ID).
		SetRepositoryURL("git@github.com:GrandCX/private-repo.git").
		SetDefaultBranch("master").
		Save(ctx); err != nil {
		t.Fatalf("update project repo remote: %v", err)
	}

	resolver := configureOrganizationGitHubCredential(ctx, t, client, project.OrganizationID, "ghu_project_repo_token")
	svc := NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.ConfigureGitHubCredentials(resolver)
	sshSession := &mirrorTestSSHSession{output: []byte("deadbeef\n")}
	svc.ConfigureSSHPool(sshinfra.NewPool("/tmp/openase",
		sshinfra.WithDialer(&mirrorTestSSHDialer{clients: []sshinfra.Client{
			&mirrorTestSSHClient{session: sshSession},
		}}),
		sshinfra.WithReadFile(func(string) ([]byte, error) {
			return []byte("key"), nil
		}),
	))

	prepared, err := svc.Prepare(ctx, PrepareInput{
		ProjectRepoID: projectRepo.ID,
		MachineID:     machine.ID,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if prepared.HeadCommit == nil || *prepared.HeadCommit != "deadbeef" {
		t.Fatalf("Prepare() head commit = %v", prepared.HeadCommit)
	}
	if !strings.Contains(sshSession.command, "export GH_TOKEN=") || !strings.Contains(sshSession.command, "ghu_project_repo_token") {
		t.Fatalf("expected GH_TOKEN projection in remote command, got %q", sshSession.command)
	}
	if !strings.Contains(sshSession.command, "unset SSH_AUTH_SOCK") {
		t.Fatalf("expected SSH_AUTH_SOCK to be cleared, got %q", sshSession.command)
	}
	if !strings.Contains(sshSession.command, "git_transport clone --branch") || !strings.Contains(sshSession.command, "https://github.com/grandcx/private-repo.git") {
		t.Fatalf("expected normalized GitHub HTTPS clone command, got %q", sshSession.command)
	}
	if !strings.Contains(sshSession.command, "git -c credential.helper=") {
		t.Fatalf("expected credential helper override, got %q", sshSession.command)
	}
	if strings.Contains(sshSession.command, "git@github.com:GrandCX/private-repo.git") {
		t.Fatalf("expected SSH GitHub URL to be normalized out of remote command, got %q", sshSession.command)
	}
}

func TestServiceRegisterExistingAcceptsEquivalentGitHubTransport(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()

	project, machine, projectRepo := createMirrorTestFixtures(ctx, t, client)
	repoPath, headCommit := createGitRepository(t)
	repository, err := git.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	if err := repository.DeleteRemote("origin"); err != nil {
		t.Fatalf("delete origin remote: %v", err)
	}
	if _, err := repository.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{"git@github.com:GrandCX/private-repo.git"},
	}); err != nil {
		t.Fatalf("create ssh origin remote: %v", err)
	}

	if _, err := client.ProjectRepo.UpdateOneID(projectRepo.ID).
		SetRepositoryURL("ssh://git@github.com/GrandCX/private-repo.git").
		SetDefaultBranch("master").
		Save(ctx); err != nil {
		t.Fatalf("update project repo remote: %v", err)
	}

	svc := NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.ConfigureGitHubCredentials(configureOrganizationGitHubCredential(ctx, t, client, project.OrganizationID, "ghu_registered_repo_token"))

	registered, err := svc.RegisterExisting(ctx, RegisterExistingInput{
		ProjectRepoID: projectRepo.ID,
		MachineID:     machine.ID,
		LocalPath:     repoPath,
	})
	if err != nil {
		t.Fatalf("RegisterExisting() error = %v", err)
	}
	if registered.HeadCommit == nil || *registered.HeadCommit != headCommit {
		t.Fatalf("RegisterExisting() head commit = %v, want %s", registered.HeadCommit, headCommit)
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

type mirrorTestSSHDialer struct {
	clients []sshinfra.Client
	calls   int
}

func (d *mirrorTestSSHDialer) DialContext(context.Context, sshinfra.DialConfig) (sshinfra.Client, error) {
	if d.calls >= len(d.clients) {
		return nil, errors.New("unexpected dial")
	}
	client := d.clients[d.calls]
	d.calls++
	return client, nil
}

type mirrorTestSSHClient struct {
	session sshinfra.Session
}

func (c *mirrorTestSSHClient) NewSession() (sshinfra.Session, error) {
	if c.session == nil {
		c.session = &mirrorTestSSHSession{}
	}
	return c.session, nil
}

func (c *mirrorTestSSHClient) SendRequest(string, bool, []byte) (bool, []byte, error) {
	return true, nil, nil
}

func (c *mirrorTestSSHClient) Close() error {
	return nil
}

type mirrorTestSSHSession struct {
	output  []byte
	err     error
	command string
}

func (s *mirrorTestSSHSession) CombinedOutput(cmd string) ([]byte, error) {
	s.command = cmd
	return s.output, s.err
}

func (s *mirrorTestSSHSession) StdinPipe() (io.WriteCloser, error) {
	return nil, errors.New("not implemented")
}

func (s *mirrorTestSSHSession) StdoutPipe() (io.Reader, error) {
	return nil, errors.New("not implemented")
}

func (s *mirrorTestSSHSession) StderrPipe() (io.Reader, error) {
	return nil, errors.New("not implemented")
}

func (s *mirrorTestSSHSession) Start(string) error {
	return errors.New("not implemented")
}

func (s *mirrorTestSSHSession) Signal(string) error {
	return errors.New("not implemented")
}

func (s *mirrorTestSSHSession) Wait() error {
	return nil
}

func (s *mirrorTestSSHSession) Close() error {
	return nil
}

func configureOrganizationGitHubCredential(ctx context.Context, t *testing.T, client *ent.Client, organizationID uuid.UUID, token string) githubauthservice.TokenResolver {
	t.Helper()

	service, err := githubauthservice.New(githubauthrepo.NewEntRepository(client), nil, "postgres://openase:test@localhost/openase")
	if err != nil {
		t.Fatalf("githubauthservice.New() error = %v", err)
	}
	sealed, err := service.SealToken(token, githubauthdomain.SourceManualPaste)
	if err != nil {
		t.Fatalf("SealToken() error = %v", err)
	}
	if _, err := client.Organization.UpdateOneID(organizationID).
		SetGithubOutboundCredential(&sealed).
		Save(ctx); err != nil {
		t.Fatalf("save organization GitHub credential: %v", err)
	}
	return service
}

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}
