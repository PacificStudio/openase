package machinetransport

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

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
