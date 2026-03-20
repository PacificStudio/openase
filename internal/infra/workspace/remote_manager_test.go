package workspace

import (
	"context"
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
		TicketIdentifier: "ASE-104",
		BranchName:       "agent/codex-01/ASE-104",
		Repos: []RepoRequest{
			{
				Name:          "backend",
				RepositoryURL: "git@github.com:acme/backend.git",
				DefaultBranch: "main",
				ClonePath:     "backend",
				BranchName:    "agent/codex-01/ASE-104",
			},
		},
	}

	workspaceItem, err := manager.Prepare(context.Background(), remoteTestMachine(), request)
	if err != nil {
		t.Fatalf("prepare remote workspace: %v", err)
	}

	if workspaceItem.Path != "/srv/openase/workspaces/ASE-104" {
		t.Fatalf("expected workspace path, got %q", workspaceItem.Path)
	}
	if !strings.Contains(session.command, "git clone --branch 'main' --single-branch 'git@github.com:acme/backend.git' '/srv/openase/workspaces/ASE-104/backend'") {
		t.Fatalf("expected clone command, got %q", session.command)
	}
	if !strings.Contains(session.command, "git -C '/srv/openase/workspaces/ASE-104/backend' checkout -B 'agent/codex-01/ASE-104' 'origin/main'") {
		t.Fatalf("expected checkout command, got %q", session.command)
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
