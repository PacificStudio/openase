package machinetransport

import (
	"context"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

func TestWebsocketListenerTransportProbeAndReachability(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(NewWebsocketListenerHandler(ListenerHandlerOptions{}))
	defer server.Close()

	machine := testListenerMachine(websocketURL(server.URL), "")
	transport := websocketTransport{mode: domain.MachineConnectionModeWSListener}

	probe, err := transport.Probe(context.Background(), machine)
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}
	if probe.Transport != domain.MachineConnectionModeWSListener.String() {
		t.Fatalf("Probe().Transport = %q", probe.Transport)
	}
	if strings.TrimSpace(probe.Output) == "" {
		t.Fatal("Probe().Output must not be empty")
	}
	if got := probe.Resources["advertised_endpoint"]; got != advertisedEndpointString(machine) {
		t.Fatalf("Probe().Resources[advertised_endpoint] = %v", got)
	}

	collector := NewMonitorCollector(NewResolver(nil, nil), nil)
	reachability, err := collector.CollectReachability(context.Background(), machine)
	if err != nil {
		t.Fatalf("CollectReachability() error = %v", err)
	}
	if !reachability.Reachable {
		t.Fatalf("CollectReachability().Reachable = false, want true")
	}
	if reachability.Transport != domain.MachineConnectionModeWSListener.String() {
		t.Fatalf("CollectReachability().Transport = %q", reachability.Transport)
	}
}

func TestWebsocketListenerTransportPrepareWorkspaceAndSyncArtifacts(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(NewWebsocketListenerHandler(ListenerHandlerOptions{}))
	defer server.Close()

	machine := testListenerMachine(websocketURL(server.URL), "")
	transport := websocketTransport{mode: domain.MachineConnectionModeWSListener}

	workspaceRoot := t.TempDir()
	request, err := workspaceinfra.ParseSetupRequest(workspaceinfra.SetupInput{
		WorkspaceRoot:    workspaceRoot,
		OrganizationSlug: "acme",
		ProjectSlug:      "listener-test",
		AgentName:        "agent",
		TicketIdentifier: "ase-10",
	})
	if err != nil {
		t.Fatalf("ParseSetupRequest() error = %v", err)
	}

	workspaceItem, err := transport.PrepareWorkspace(context.Background(), machine, request)
	if err != nil {
		t.Fatalf("PrepareWorkspace() error = %v", err)
	}
	if _, err := os.Stat(workspaceItem.Path); err != nil {
		t.Fatalf("prepared workspace %s missing: %v", workspaceItem.Path, err)
	}

	localRoot := t.TempDir()
	targetRoot := filepath.Join(workspaceRoot, "artifact-target")
	if err := os.MkdirAll(filepath.Join(localRoot, "subdir"), 0o750); err != nil {
		t.Fatalf("MkdirAll(localRoot) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(localRoot, "subdir", "hello.txt"), []byte("listener-sync"), 0o600); err != nil {
		t.Fatalf("WriteFile(local artifact) error = %v", err)
	}

	if err := transport.SyncArtifacts(context.Background(), machine, SyncArtifactsRequest{
		LocalRoot:  localRoot,
		TargetRoot: targetRoot,
		Paths:      []string{"subdir/hello.txt"},
	}); err != nil {
		t.Fatalf("SyncArtifacts() error = %v", err)
	}

	// #nosec G304 -- synced artifact path is derived from test-controlled temp directories.
	content, err := os.ReadFile(filepath.Join(targetRoot, "subdir", "hello.txt"))
	if err != nil {
		t.Fatalf("ReadFile(synced artifact) error = %v", err)
	}
	if string(content) != "listener-sync" {
		t.Fatalf("synced artifact = %q, want listener-sync", string(content))
	}
}

func TestWebsocketListenerTransportStartProcess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(NewWebsocketListenerHandler(ListenerHandlerOptions{}))
	defer server.Close()

	machine := testListenerMachine(websocketURL(server.URL), "")
	transport := websocketTransport{mode: domain.MachineConnectionModeWSListener}

	for attempt := range 20 {
		spec, err := provider.NewAgentCLIProcessSpec(
			provider.MustParseAgentCLICommand("sh"),
			[]string{"-lc", "printf listener-process"},
			nil,
			nil,
		)
		if err != nil {
			t.Fatalf("NewAgentCLIProcessSpec() error = %v", err)
		}

		process, err := transport.StartProcess(context.Background(), machine, spec)
		if err != nil {
			t.Fatalf("StartProcess() error = %v", err)
		}

		stdout, err := io.ReadAll(process.Stdout())
		if err != nil {
			t.Fatalf("ReadAll(process.Stdout()) error = %v", err)
		}
		if err := process.Wait(); err != nil {
			t.Fatalf("process.Wait() error = %v", err)
		}
		if !strings.Contains(string(stdout), "listener-process") {
			t.Fatalf("attempt %d process stdout = %q", attempt+1, string(stdout))
		}
	}
}

func TestWebsocketListenerTransportDialErrorClassification(t *testing.T) {
	t.Parallel()

	tokenServer := httptest.NewServer(NewWebsocketListenerHandler(ListenerHandlerOptions{BearerToken: "listener-secret"}))
	defer tokenServer.Close()

	machine := testListenerMachine(websocketURL(tokenServer.URL), "wrong-secret")
	transport := websocketTransport{mode: domain.MachineConnectionModeWSListener}
	if _, err := transport.OpenCommandSession(context.Background(), machine); err == nil || !strings.Contains(err.Error(), "authentication failed") {
		t.Fatalf("OpenCommandSession(auth failure) error = %v", err)
	}

	tlsServer := httptest.NewTLSServer(NewWebsocketListenerHandler(ListenerHandlerOptions{}))
	defer tlsServer.Close()
	if _, err := transport.OpenCommandSession(context.Background(), testListenerMachine(websocketURL(tlsServer.URL), "")); err == nil || !strings.Contains(err.Error(), "TLS verification failed") {
		t.Fatalf("OpenCommandSession(tls failure) error = %v", err)
	}
}

func testListenerMachine(endpoint string, token string) domain.Machine {
	machine := domain.Machine{
		ID:                 uuid.New(),
		Name:               "listener-01",
		Host:               "listener.internal",
		ConnectionMode:     domain.MachineConnectionModeWSListener,
		AdvertisedEndpoint: stringPtr(endpoint),
		ChannelCredential:  domain.MachineChannelCredential{Kind: domain.MachineChannelCredentialKindNone},
	}
	if token != "" {
		machine.ChannelCredential = domain.MachineChannelCredential{
			Kind:    domain.MachineChannelCredentialKindToken,
			TokenID: stringPtr(token),
		}
	}
	return machine
}

func advertisedEndpointString(machine domain.Machine) string {
	if machine.AdvertisedEndpoint == nil {
		return ""
	}
	return *machine.AdvertisedEndpoint
}

func websocketURL(raw string) string {
	switch {
	case strings.HasPrefix(raw, "https://"):
		return "wss://" + strings.TrimPrefix(raw, "https://")
	case strings.HasPrefix(raw, "http://"):
		return "ws://" + strings.TrimPrefix(raw, "http://")
	default:
		return raw
	}
}

func stringPtr(value string) *string {
	return &value
}
