package workspace

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
)

func TestRemoteManagerPrepareBuildsCloneAndCheckoutCommands(t *testing.T) {
	session := &remoteTestSession{}
	client := &remoteTestClient{session: session}
	dialer := &remoteTestDialer{client: client}
	pool := sshinfra.NewPool("/tmp/openase", sshinfra.WithDialer(dialer), sshinfra.WithReadFile(func(string) ([]byte, error) {
		return []byte("key"), nil
	}))

	manager := NewRemoteManager(pool)
	request := SetupRequest{
		WorkspaceRoot:    "/srv/openase/workspaces",
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		TicketIdentifier: "ASE-104",
		BranchName:       "agent/ASE-104",
		Repos: []RepoRequest{
			{
				Name:             "backend",
				RepositoryURL:    "git@github.com:acme/backend.git",
				DefaultBranch:    "main",
				WorkspaceDirname: "backend",
				BranchName:       "agent/ASE-104",
			},
		},
	}

	workspaceItem, err := manager.Prepare(context.Background(), remoteTestMachine(), request)
	if err != nil {
		t.Fatalf("prepare remote workspace: %v", err)
	}

	if workspaceItem.Path != "/srv/openase/workspaces/acme/payments/ASE-104" {
		t.Fatalf("expected workspace path, got %q", workspaceItem.Path)
	}
	if !strings.Contains(session.command, "'git' 'clone' '--branch' 'main' '--single-branch' 'git@github.com:acme/backend.git' '/srv/openase/workspaces/acme/payments/ASE-104/backend'") {
		t.Fatalf("expected clone command, got %q", session.command)
	}
	if !strings.Contains(session.command, "git -C '/srv/openase/workspaces/acme/payments/ASE-104/backend' checkout -B 'agent/ASE-104' 'origin/main'") {
		t.Fatalf("expected checkout command, got %q", session.command)
	}
}

func TestBuildPrepareWorkspaceCommandUsesRepositoryURLAsOrigin(t *testing.T) {
	request := SetupRequest{
		WorkspaceRoot:    "/srv/openase/workspaces",
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		TicketIdentifier: "ASE-104",
		BranchName:       "agent/ASE-104",
		Repos: []RepoRequest{
			{
				Name:             "backend",
				RepositoryURL:    "git@github.com:acme/backend.git",
				DefaultBranch:    "main",
				WorkspaceDirname: "backend",
				BranchName:       "agent/ASE-104",
			},
		},
	}

	command, err := buildPrepareWorkspaceCommand(request)
	if err != nil {
		t.Fatalf("buildPrepareWorkspaceCommand() error = %v", err)
	}
	if !strings.Contains(command, "'git' 'clone' '--branch' 'main' '--single-branch' 'git@github.com:acme/backend.git'") {
		t.Fatalf("expected repository clone command, got %q", command)
	}
	if !strings.Contains(command, "if [ \"$actual_origin\" != 'git@github.com:acme/backend.git' ]; then") {
		t.Fatalf("expected origin verification against repository URL, got %q", command)
	}
}

func TestBuildPrepareWorkspaceCommandAddsHTTPSAuthForGitHubRepos(t *testing.T) {
	request := SetupRequest{
		WorkspaceRoot:    "/srv/openase/workspaces",
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		TicketIdentifier: "ASE-105",
		BranchName:       "agent/ASE-105",
		Repos: []RepoRequest{
			{
				Name:          "backend",
				RepositoryURL: "https://github.com/acme/backend.git",
				DefaultBranch: "main",
				BranchName:    "agent/ASE-105",
				HTTPBasicAuth: &HTTPBasicAuthRequest{
					Username: "x-access-token",
					Password: "ghu_test",
				},
			},
		},
	}

	command, err := buildPrepareWorkspaceCommand(request)
	if err != nil {
		t.Fatalf("buildPrepareWorkspaceCommand() error = %v", err)
	}
	if !strings.Contains(command, "'-c' 'http.https://github.com/.extraheader=AUTHORIZATION: basic ") {
		t.Fatalf("expected GitHub auth extraheader, got %q", command)
	}
	if !strings.Contains(command, "'-c' 'credential.helper=' '-C'") {
		t.Fatalf("expected disabled credential helper for authenticated fetch, got %q", command)
	}
}

func TestPrepareWithCommandRunnerClassifiesRepositoryAuthFailures(t *testing.T) {
	request := SetupRequest{
		WorkspaceRoot:    "/srv/openase/workspaces",
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		TicketIdentifier: "ASE-106",
	}

	_, err := PrepareWithCommandRunner(remoteFailingRunner{
		output: "fatal: Authentication failed for 'https://github.com/acme/backend.git/'",
		err:    errors.New("exit status 128"),
	}, request)
	if err == nil {
		t.Fatal("PrepareWithCommandRunner() error = nil, want repo auth failure")
	}
	var prepareErr *PrepareError
	if !errors.As(err, &prepareErr) {
		t.Fatalf("PrepareWithCommandRunner() error = %T, want *PrepareError", err)
	}
	if prepareErr.Stage != PrepareFailureStageRepoAuth {
		t.Fatalf("PrepareWithCommandRunner() stage = %q, want %q", prepareErr.Stage, PrepareFailureStageRepoAuth)
	}
}

func TestPrepareWithCommandRunnerClassifiesWorkspaceRootFailures(t *testing.T) {
	request := SetupRequest{
		WorkspaceRoot:    "/srv/openase/workspaces",
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		TicketIdentifier: "ASE-107",
	}

	_, err := PrepareWithCommandRunner(remoteFailingRunner{
		output: "mkdir: cannot create directory '/srv/openase/workspaces/acme': Permission denied",
		err:    errors.New("exit status 1"),
	}, request)
	if err == nil {
		t.Fatal("PrepareWithCommandRunner() error = nil, want workspace root failure")
	}
	var prepareErr *PrepareError
	if !errors.As(err, &prepareErr) {
		t.Fatalf("PrepareWithCommandRunner() error = %T, want *PrepareError", err)
	}
	if prepareErr.Stage != PrepareFailureStageWorkspaceRoot {
		t.Fatalf("PrepareWithCommandRunner() stage = %q, want %q", prepareErr.Stage, PrepareFailureStageWorkspaceRoot)
	}
}

type remoteTestDialer struct {
	client sshinfra.Client
}

func (d *remoteTestDialer) DialContext(context.Context, sshinfra.DialConfig) (sshinfra.Client, error) {
	return d.client, nil
}

type remoteTestClient struct {
	session sshinfra.Session
}

func (c *remoteTestClient) NewSession() (sshinfra.Session, error) {
	return c.session, nil
}

func (c *remoteTestClient) SendRequest(string, bool, []byte) (bool, []byte, error) {
	return true, nil, nil
}

func (c *remoteTestClient) Close() error {
	return nil
}

type remoteTestSession struct {
	command string
}

func (s *remoteTestSession) CombinedOutput(cmd string) ([]byte, error) {
	s.command = cmd
	return nil, nil
}

func (s *remoteTestSession) StdinPipe() (io.WriteCloser, error) { return nil, nil }

func (s *remoteTestSession) StdoutPipe() (io.Reader, error) { return strings.NewReader(""), nil }

func (s *remoteTestSession) StderrPipe() (io.Reader, error) { return strings.NewReader(""), nil }

func (s *remoteTestSession) Start(string) error { return nil }

func (s *remoteTestSession) Signal(string) error { return nil }

func (s *remoteTestSession) Wait() error { return nil }

func (s *remoteTestSession) Close() error { return nil }

func remoteTestMachine() domain.Machine {
	sshUser := "openase"
	keyPath := "keys/gpu-01.pem"
	workspaceRoot := "/srv/openase/workspaces"
	return domain.Machine{
		Name:          "gpu-01",
		Host:          "10.0.1.10",
		Port:          22,
		SSHUser:       &sshUser,
		SSHKeyPath:    &keyPath,
		WorkspaceRoot: &workspaceRoot,
	}
}

type remoteFailingRunner struct {
	output string
	err    error
}

func (r remoteFailingRunner) CombinedOutput(string) ([]byte, error) {
	return []byte(r.output), r.err
}
