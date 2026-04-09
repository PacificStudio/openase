package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	}, request, nil)
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
	}, request, nil)
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

func TestPrepareWithCommandRunnerLogsRepoPreparePhases(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
	request := SetupRequest{
		WorkspaceRoot:    "/srv/openase/workspaces",
		OrganizationSlug: "acme",
		ProjectSlug:      "payments",
		TicketIdentifier: "ASE-146",
		Observability: PrepareObservability{
			MachineID: "machine-1",
			RunID:     "run-1",
			TicketID:  "ticket-1",
		},
		Repos: []RepoRequest{{
			Name:             "backend",
			RepositoryURL:    "git@github.com:acme/backend.git",
			DefaultBranch:    "main",
			WorkspaceDirname: "backend",
			BranchName:       "agent/ASE-146",
		}},
	}
	workspacePath, err := TicketWorkspacePath(
		request.WorkspaceRoot,
		request.OrganizationSlug,
		request.ProjectSlug,
		request.TicketIdentifier,
	)
	if err != nil {
		t.Fatalf("TicketWorkspacePath() error = %v", err)
	}
	repoPath := RepoPath(workspacePath, request.Repos[0].WorkspaceDirname, request.Repos[0].Name)
	output := strings.Join([]string{
		remotePreparePhaseLineForTest(request.Observability, repoPath, "repo_prepare_begin", 0, ""),
		remotePreparePhaseLineForTest(request.Observability, repoPath, "clone_or_open", 12, "clone"),
		remotePreparePhaseLineForTest(request.Observability, repoPath, "fetch", 8, ""),
		remotePreparePhaseLineForTest(request.Observability, repoPath, "checkout_reset", 5, ""),
		remotePreparePhaseLineForTest(request.Observability, repoPath, "repo_prepare_done", 25, ""),
	}, "\n")

	if _, err := PrepareWithCommandRunner(remoteFailingRunner{output: output}, request, logger); err != nil {
		t.Fatalf("PrepareWithCommandRunner() error = %v", err)
	}

	logOutput := logBuffer.String()
	for _, needle := range []string{
		"machine_id=machine-1",
		"run_id=run-1",
		"ticket_id=ticket-1",
		"repo_name=backend",
		"repo_path=" + repoPath,
		"phase=repo_prepare_begin",
		"phase=clone_or_open",
		"phase_result=clone",
		"phase=fetch",
		"phase=checkout_reset",
		"phase=repo_prepare_done",
	} {
		if !strings.Contains(logOutput, needle) {
			t.Fatalf("expected log output to contain %q, got %q", needle, logOutput)
		}
	}
}

func remotePreparePhaseLineForTest(
	observability PrepareObservability,
	repoPath string,
	phase string,
	durationMS int64,
	phaseResult string,
) string {
	return fmt.Sprintf(
		"%s%s|%s|%s|%s|%s|%s|%d|%s",
		remotePreparePhasePrefix,
		strings.TrimSpace(observability.MachineID),
		strings.TrimSpace(observability.RunID),
		strings.TrimSpace(observability.TicketID),
		"backend",
		strings.TrimSpace(repoPath),
		strings.TrimSpace(phase),
		durationMS,
		strings.TrimSpace(phaseResult),
	)
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
