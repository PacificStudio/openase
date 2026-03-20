package ssh

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestProcessManagerStartsRemoteCommand(t *testing.T) {
	session := &fakeSession{waitCh: make(chan error, 1)}
	client := &fakeClient{session: session}
	dialer := &fakeDialer{clients: []Client{client}}
	pool := NewPool("/tmp/openase", WithDialer(dialer), WithReadFile(func(string) ([]byte, error) {
		return []byte("key"), nil
	}))

	manager := NewProcessManager(pool, testRemoteMachine())
	workingDirectory := provider.MustParseAbsolutePath("/srv/openase/workspaces/ASE-104")
	spec, err := provider.NewAgentCLIProcessSpec(
		provider.MustParseAgentCLICommand("/usr/local/bin/codex"),
		[]string{"serve", "--stdio"},
		&workingDirectory,
		[]string{"OPENASE_MODE=remote"},
	)
	if err != nil {
		t.Fatalf("build process spec: %v", err)
	}

	process, err := manager.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("start process: %v", err)
	}

	if !strings.Contains(session.startedCommand, "cd '/srv/openase/workspaces/ASE-104'") {
		t.Fatalf("expected working directory in command, got %q", session.startedCommand)
	}
	if !strings.Contains(session.startedCommand, "env 'OPENASE_MODE=remote' '/usr/local/bin/codex' 'serve' '--stdio'") {
		t.Fatalf("expected env and command in remote shell, got %q", session.startedCommand)
	}

	session.waitCh <- nil
	if err := process.Wait(); err != nil {
		t.Fatalf("wait returned error: %v", err)
	}
}

func TestRemoteProcessStopSignalsInterrupt(t *testing.T) {
	session := &fakeSession{waitCh: make(chan error, 1)}
	client := &fakeClient{session: session}
	dialer := &fakeDialer{clients: []Client{client}}
	pool := NewPool("/tmp/openase", WithDialer(dialer), WithReadFile(func(string) ([]byte, error) {
		return []byte("key"), nil
	}))

	manager := NewProcessManager(pool, testRemoteMachine())
	spec, err := provider.NewAgentCLIProcessSpec(provider.MustParseAgentCLICommand("codex"), nil, nil, nil)
	if err != nil {
		t.Fatalf("build process spec: %v", err)
	}

	process, err := manager.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("start process: %v", err)
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		session.waitCh <- nil
	}()

	if err := process.Stop(context.Background()); err != nil {
		t.Fatalf("stop returned error: %v", err)
	}
	if session.signal != sshInterruptSignal {
		t.Fatalf("expected interrupt signal %q, got %q", sshInterruptSignal, session.signal)
	}
}

func TestRemoteProcessStopReturnsWaitErrorWhenContextCloses(t *testing.T) {
	session := &fakeSession{
		waitCh:   make(chan error, 1),
		closeErr: errors.New("close failed"),
	}
	client := &fakeClient{session: session}
	dialer := &fakeDialer{clients: []Client{client}}
	pool := NewPool("/tmp/openase", WithDialer(dialer), WithReadFile(func(string) ([]byte, error) {
		return []byte("key"), nil
	}))

	manager := NewProcessManager(pool, testRemoteMachine())
	spec, err := provider.NewAgentCLIProcessSpec(provider.MustParseAgentCLICommand("codex"), nil, nil, nil)
	if err != nil {
		t.Fatalf("build process spec: %v", err)
	}

	process, err := manager.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("start process: %v", err)
	}

	stopCtx, cancel := context.WithCancel(context.Background())
	cancel()

	wantErr := errors.New("remote process exited with status 130")
	go func() {
		time.Sleep(10 * time.Millisecond)
		session.waitCh <- wantErr
	}()

	if err := process.Stop(stopCtx); !errors.Is(err, wantErr) {
		t.Fatalf("expected wait error %v, got %v", wantErr, err)
	}
}
