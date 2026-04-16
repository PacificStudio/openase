package machinetransport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestPrepareRemoteOpenASEEnvironmentRepairsMissingBinary(t *testing.T) {
	t.Parallel()

	remoteHome := "/home/agentuser"
	existing := map[string]bool{}
	commandExecutor := &fakeRuntimePreflightCommandExecutor{
		home:     remoteHome,
		existing: existing,
	}
	artifactSync := &fakeRuntimePreflightArtifactSync{
		onSync: func(request SyncArtifactsRequest) {
			existing[filepath.ToSlash(filepath.Join(request.TargetRoot, request.Paths[0]))] = true
		},
	}

	environment, err := PrepareRemoteOpenASEEnvironment(context.Background(), commandExecutor, artifactSync, runtimePreflightTestMachine(""), []string{"PATH=/usr/bin"})
	if err != nil {
		t.Fatalf("PrepareRemoteOpenASEEnvironment() error = %v", err)
	}
	wantBin := filepath.ToSlash(filepath.Join(remoteHome, defaultRemoteOpenASEBinRelativePath))
	if got, ok := provider.LookupEnvironmentValue(environment, "OPENASE_REAL_BIN"); !ok || got != wantBin {
		t.Fatalf("OPENASE_REAL_BIN = %q, %v, want %q", got, ok, wantBin)
	}
	if artifactSync.calls != 1 {
		t.Fatalf("artifact sync calls = %d, want 1", artifactSync.calls)
	}
	if !existing[wantBin] {
		t.Fatalf("expected repaired binary at %s", wantBin)
	}
	if got := commandExecutor.homeLookups; got == 0 {
		t.Fatal("expected remote home lookup")
	}
}

func TestRunRemoteRuntimePreflightOverWebsocketListener(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(NewWebsocketListenerHandler(ListenerHandlerOptions{}))
	defer server.Close()

	workspaceRoot := t.TempDir()
	binDir := t.TempDir()
	writeRuntimePreflightWrapper(t, workspaceRoot)
	writeFakeOpenASEBinary(t, binDir)

	transport := websocketTransport{mode: domain.MachineConnectionModeWSListener}
	machine := runtimePreflightTestMachine(websocketURL(server.URL))
	err := RunRemoteRuntimePreflight(context.Background(), transport, machine, RuntimePreflightSpec{
		WorkingDirectory: workspaceRoot,
		AgentCommand:     "/bin/sh",
		Environment:      []string{"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH")},
	})
	if err != nil {
		t.Fatalf("RunRemoteRuntimePreflight() error = %v", err)
	}
}

func TestRunRemoteRuntimePreflightReportsOpenASEFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(NewWebsocketListenerHandler(ListenerHandlerOptions{}))
	defer server.Close()

	workspaceRoot := t.TempDir()
	writeRuntimePreflightWrapper(t, workspaceRoot)

	transport := websocketTransport{mode: domain.MachineConnectionModeWSListener}
	machine := runtimePreflightTestMachine(websocketURL(server.URL))
	err := RunRemoteRuntimePreflight(context.Background(), transport, machine, RuntimePreflightSpec{
		WorkingDirectory: workspaceRoot,
		AgentCommand:     "/bin/sh",
		Environment:      []string{"PATH=/nonexistent", "OPENASE_REAL_BIN="},
	})
	if err == nil {
		t.Fatal("RunRemoteRuntimePreflight() error = nil, want openase failure")
	}
	var preflightErr *RuntimePreflightError
	if !errors.As(err, &preflightErr) {
		t.Fatalf("RunRemoteRuntimePreflight() error = %T, want *RuntimePreflightError", err)
	}
	if preflightErr.Stage != RuntimePreflightStageOpenASE {
		t.Fatalf("RunRemoteRuntimePreflight() stage = %q, want %q", preflightErr.Stage, RuntimePreflightStageOpenASE)
	}
	if !strings.Contains(preflightErr.Error(), "workspace openase wrapper could not resolve a runnable openase binary") {
		t.Fatalf("RunRemoteRuntimePreflight() error = %q", preflightErr.Error())
	}
}

func TestRunRemoteRuntimePreflightReportsOpenASEFailureForFastExits(t *testing.T) {
	server := httptest.NewServer(NewWebsocketListenerHandler(ListenerHandlerOptions{}))
	defer server.Close()

	transport := websocketTransport{mode: domain.MachineConnectionModeWSListener}
	machine := runtimePreflightTestMachine(websocketURL(server.URL))

	for i := 0; i < 50; i++ {
		workspaceRoot := t.TempDir()
		writeRuntimePreflightWrapper(t, workspaceRoot)

		err := RunRemoteRuntimePreflight(context.Background(), transport, machine, RuntimePreflightSpec{
			WorkingDirectory: workspaceRoot,
			AgentCommand:     "/bin/sh",
			Environment:      []string{"PATH=/nonexistent", "OPENASE_REAL_BIN="},
		})
		if err == nil {
			t.Fatalf("iteration %d: RunRemoteRuntimePreflight() error = nil, want openase failure", i)
		}
		var preflightErr *RuntimePreflightError
		if !errors.As(err, &preflightErr) {
			t.Fatalf("iteration %d: RunRemoteRuntimePreflight() error = %T, want *RuntimePreflightError", i, err)
		}
		if preflightErr.Stage != RuntimePreflightStageOpenASE {
			t.Fatalf("iteration %d: RunRemoteRuntimePreflight() stage = %q, want %q", i, preflightErr.Stage, RuntimePreflightStageOpenASE)
		}
	}
}

func runtimePreflightTestMachine(endpoint string) domain.Machine {
	return domain.Machine{
		ID:                 uuid.New(),
		Name:               "listener-preflight",
		Host:               "listener.internal",
		ConnectionMode:     domain.MachineConnectionModeWSListener,
		AdvertisedEndpoint: stringPtr(endpoint),
		ChannelCredential:  domain.MachineChannelCredential{Kind: domain.MachineChannelCredentialKindNone},
	}
}

func writeRuntimePreflightWrapper(t *testing.T, workspaceRoot string) {
	t.Helper()

	wrapperPath := filepath.Join(workspaceRoot, ".openase", "bin", "openase")
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o750); err != nil {
		t.Fatalf("MkdirAll(wrapper) error = %v", err)
	}
	content := `#!/bin/sh
set -eu

if [ -n "${OPENASE_REAL_BIN:-}" ]; then
  exec "$OPENASE_REAL_BIN" "$@"
fi
if command -v openase >/dev/null 2>&1; then
  exec "$(command -v openase)" "$@"
fi
echo "missing openase" >&2
exit 1
`
	if err := os.WriteFile(wrapperPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(wrapper) error = %v", err)
	}
	// #nosec G302 -- test wrapper must be executable in the temp workspace.
	if err := os.Chmod(wrapperPath, 0o700); err != nil {
		t.Fatalf("WriteFile(wrapper) error = %v", err)
	}
}

func writeFakeOpenASEBinary(t *testing.T, binDir string) {
	t.Helper()

	fakeBinaryPath := filepath.Join(binDir, "openase")
	content := `#!/bin/sh
set -eu
if [ "${1:-}" = "version" ]; then
  exit 0
fi
exit 0
`
	if err := os.WriteFile(fakeBinaryPath, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(fake openase) error = %v", err)
	}
	// #nosec G302 -- test binary must be executable in the temp workspace.
	if err := os.Chmod(fakeBinaryPath, 0o700); err != nil {
		t.Fatalf("WriteFile(fake openase) error = %v", err)
	}
}

type fakeRuntimePreflightArtifactSync struct {
	calls  int
	onSync func(SyncArtifactsRequest)
}

func (f *fakeRuntimePreflightArtifactSync) SyncArtifacts(_ context.Context, _ domain.Machine, request SyncArtifactsRequest) error {
	f.calls++
	if f.onSync != nil {
		f.onSync(request)
	}
	return nil
}

type fakeRuntimePreflightCommandExecutor struct {
	home        string
	existing    map[string]bool
	homeLookups int
}

func (f *fakeRuntimePreflightCommandExecutor) OpenCommandSession(context.Context, domain.Machine) (CommandSession, error) {
	return &fakeRuntimePreflightCommandSession{executor: f}, nil
}

type fakeRuntimePreflightCommandSession struct {
	executor *fakeRuntimePreflightCommandExecutor
}

func (s *fakeRuntimePreflightCommandSession) CombinedOutput(cmd string) ([]byte, error) {
	if strings.Contains(cmd, `printf %s "${HOME:-}"`) {
		s.executor.homeLookups++
		return []byte(s.executor.home), nil
	}
	if strings.Contains(cmd, "test -x ") {
		for path, ok := range s.executor.existing {
			if ok && strings.Contains(cmd, sshinfra.ShellQuote(path)) {
				return []byte("ok"), nil
			}
		}
		return nil, fmt.Errorf("missing executable")
	}
	return nil, fmt.Errorf("unexpected command %s", cmd)
}

func (s *fakeRuntimePreflightCommandSession) StdinPipe() (io.WriteCloser, error) {
	return nil, fmt.Errorf("not supported")
}
func (s *fakeRuntimePreflightCommandSession) StdoutPipe() (io.Reader, error) {
	return strings.NewReader(""), nil
}
func (s *fakeRuntimePreflightCommandSession) StderrPipe() (io.Reader, error) {
	return strings.NewReader(""), nil
}
func (s *fakeRuntimePreflightCommandSession) Start(string) error  { return fmt.Errorf("not supported") }
func (s *fakeRuntimePreflightCommandSession) Signal(string) error { return nil }
func (s *fakeRuntimePreflightCommandSession) Wait() error         { return nil }
func (s *fakeRuntimePreflightCommandSession) Close() error        { return nil }
