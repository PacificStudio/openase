package machinetransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinechanneldomain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/testutil/containerharness"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	websocketListenerContainerHelperEnv = "OPENASE_TEST_WS_LISTENER_HELPER"
	websocketListenerContainerPortEnv   = "OPENASE_TEST_WS_LISTENER_PORT"
	websocketListenerContainerPort      = 19852
)

func TestUnifiedWebsocketRuntimeContractSuite(t *testing.T) {
	t.Run("listener", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(NewWebsocketListenerHandler(ListenerHandlerOptions{}))
		defer server.Close()

		machine := testListenerMachine(websocketURL(server.URL), "")
		runRuntimeContractSuite(t, machine, func(ctx context.Context) (*runtimeProtocolClient, func(error), error) {
			return dialWebsocketRuntimeClient(ctx, machine)
		})
	})

	t.Run("reverse", func(t *testing.T) {
		t.Parallel()

		machineID := uuid.New()
		sessionRegistry := NewReverseRuntimeRelayRegistry()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer func() { _ = conn.Close() }()

			helloEnvelope, err := readMachineEnvelopeForTest(conn)
			if err != nil || helloEnvelope.Type != machinechanneldomain.MessageTypeHello {
				return
			}
			authenticateEnvelope, err := readMachineEnvelopeForTest(conn)
			if err != nil || authenticateEnvelope.Type != machinechanneldomain.MessageTypeAuthenticate {
				return
			}

			sessionID := uuid.NewString()
			sessionRegistry.Register(machineID, sessionID, func(ctx context.Context, envelope runtimecontract.Envelope) error {
				return writeMachineEnvelopeForTest(conn, machinechanneldomain.MessageTypeRuntime, sessionID, envelope)
			})
			defer sessionRegistry.Remove(sessionID)

			if err := writeMachineEnvelopeForTest(conn, machinechanneldomain.MessageTypeRegistered, sessionID, machinechanneldomain.Registered{
				MachineID:                machineID.String(),
				SessionID:                sessionID,
				HeartbeatIntervalSeconds: 1,
				HeartbeatTimeoutSeconds:  5,
			}); err != nil {
				return
			}

			for {
				envelope, err := readMachineEnvelopeForTest(conn)
				if err != nil {
					return
				}
				switch envelope.Type {
				case machinechanneldomain.MessageTypeHeartbeat:
					continue
				case machinechanneldomain.MessageTypeGoodbye:
					return
				case machinechanneldomain.MessageTypeRuntime:
					runtimeEnvelope, err := runtimeEnvelopeFromMachineEnvelopeForTest(envelope)
					if err != nil {
						return
					}
					if err := sessionRegistry.Deliver(sessionID, runtimeEnvelope); err != nil {
						return
					}
				default:
					return
				}
			}
		}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		errCh := make(chan error, 1)
		go func() {
			errCh <- runReverseRuntimeParticipant(ctx, server.URL, machineID)
		}()

		waitForReverseRuntimeRegistration(t, sessionRegistry, machineID)

		machine := catalogdomain.Machine{
			ID:             machineID,
			Name:           "reverse-contract",
			Host:           "reverse.internal",
			ConnectionMode: catalogdomain.MachineConnectionModeWSReverse,
			DaemonStatus: catalogdomain.MachineDaemonStatus{
				Registered:   true,
				SessionState: catalogdomain.MachineTransportSessionStateConnected,
			},
		}
		runRuntimeContractSuite(t, machine, func(ctx context.Context) (*runtimeProtocolClient, func(error), error) {
			client, err := sessionRegistry.client(machineID)
			return client, func(error) {}, err
		})

		cancel()
		if err := <-errCh; err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, net.ErrClosed) {
			t.Fatalf("reverse runtime participant error = %v", err)
		}
	})
}

func TestWebsocketListenerRuntimeContainerE2E(t *testing.T) {
	containerharness.RequireContainerSuite(t)

	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve current executable: %v", err)
	}

	hostPort := containerharness.FreeTCPPort(t)
	project := containerharness.NewProject(t, containerharness.Options{
		ProjectName: "ase41-listener-" + strings.ToLower(strings.ReplaceAll(uuid.NewString(), "-", "")),
		Env: map[string]string{
			"OPENASE_TEST_WS_LISTENER_BINARY":    filepath.Clean(testBinary),
			"OPENASE_TEST_WS_LISTENER_HOST_PORT": fmt.Sprintf("%d", hostPort),
			"OPENASE_TEST_WS_LISTENER_PORT":      fmt.Sprintf("%d", websocketListenerContainerPort),
			"OPENASE_TEST_TMP_ROOT":              filepath.Clean(os.TempDir()),
			"OPENASE_TEST_UID":                   fmt.Sprintf("%d", os.Getuid()),
			"OPENASE_TEST_GID":                   fmt.Sprintf("%d", os.Getgid()),
		},
	})
	project.Up(t, nil, "ws-listener")
	project.WriteLogs(t, "listener-compose.log", nil, "ws-listener")

	machine := testListenerMachine(fmt.Sprintf("ws://127.0.0.1:%d", hostPort), "")
	waitForContainerListenerRuntime(t, machine)
	runRuntimeContractSuite(t, machine, func(ctx context.Context) (*runtimeProtocolClient, func(error), error) {
		return dialWebsocketRuntimeClient(ctx, machine)
	})
}

func TestWebsocketListenerRuntimeContainerHelper(t *testing.T) {
	if os.Getenv(websocketListenerContainerHelperEnv) != "1" {
		t.Skip("helper process only")
	}

	port := strings.TrimSpace(os.Getenv(websocketListenerContainerPortEnv))
	if port == "" {
		t.Fatal("listener helper port is required")
	}

	server := &http.Server{
		Addr:              "0.0.0.0:" + port,
		Handler:           NewWebsocketListenerHandler(ListenerHandlerOptions{}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("listener helper server failed: %v", err)
	}
}

func runRuntimeContractSuite(
	t *testing.T,
	machine catalogdomain.Machine,
	openClient func(context.Context) (*runtimeProtocolClient, func(error), error),
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, closeClient, err := openClient(ctx)
	if err != nil {
		t.Fatalf("open runtime client: %v", err)
	}
	defer closeClient(nil)

	if err := client.ensureHello(ctx); err != nil {
		t.Fatalf("ensureHello() error = %v", err)
	}

	probeEnvelope, err := client.request(ctx, runtimecontract.OperationProbe, nil)
	if err != nil {
		t.Fatalf("probe request error = %v", err)
	}
	probe, err := runtimecontract.DecodePayload[runtimecontract.ProbeResponse](probeEnvelope)
	if err != nil {
		t.Fatalf("decode probe response: %v", err)
	}
	augmentedProbe := augmentRuntimeProbe(machine, probe)
	if strings.TrimSpace(augmentedProbe.Output) == "" {
		t.Fatal("probe output must not be empty")
	}

	workspaceRoot := t.TempDir()
	binDir := t.TempDir()
	writeRuntimePreflightWrapper(t, workspaceRoot)
	writeFakeOpenASEBinary(t, binDir)

	if _, err := client.request(ctx, runtimecontract.OperationPreflight, runtimecontract.PreflightRequest{
		WorkingDirectory: workspaceRoot,
		AgentCommand:     "/bin/sh",
		Environment:      []string{"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH")},
	}); err != nil {
		t.Fatalf("preflight request error = %v", err)
	}

	prepareInput, err := workspaceinfra.ParseSetupRequest(workspaceinfra.SetupInput{
		WorkspaceRoot:    t.TempDir(),
		OrganizationSlug: "acme",
		ProjectSlug:      "contract",
		AgentName:        "runtime",
		TicketIdentifier: "ase-36",
	})
	if err != nil {
		t.Fatalf("ParseSetupRequest() error = %v", err)
	}
	prepareEnvelope, err := client.request(ctx, runtimecontract.OperationWorkspacePrepare, runtimeWorkspacePrepareRequest(prepareInput))
	if err != nil {
		t.Fatalf("workspace prepare request error = %v", err)
	}
	workspaceResponse, err := runtimecontract.DecodePayload[runtimecontract.WorkspacePrepareResponse](prepareEnvelope)
	if err != nil {
		t.Fatalf("decode workspace response: %v", err)
	}
	if _, err := os.Stat(workspaceResponse.Path); err != nil {
		t.Fatalf("prepared workspace missing: %v", err)
	}

	if _, err := client.request(ctx, runtimecontract.OperationWorkspaceReset, runtimecontract.WorkspaceResetRequest{
		Path: workspaceResponse.Path,
	}); err != nil {
		t.Fatalf("workspace reset request error = %v", err)
	}
	if _, err := os.Stat(workspaceResponse.Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected workspace reset to remove %s, got %v", workspaceResponse.Path, err)
	}

	localRoot := t.TempDir()
	targetRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(localRoot, "subdir"), 0o750); err != nil {
		t.Fatalf("create local artifact dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localRoot, "subdir", "hello.txt"), []byte("runtime-contract"), 0o600); err != nil {
		t.Fatalf("write local artifact: %v", err)
	}
	entries, err := buildArtifactSyncEntries(SyncArtifactsRequest{
		LocalRoot: localRoot,
		Paths:     []string{"subdir/hello.txt"},
	})
	if err != nil {
		t.Fatalf("buildArtifactSyncEntries() error = %v", err)
	}
	if _, err := client.request(ctx, runtimecontract.OperationArtifactSync, runtimecontract.ArtifactSyncRequest{
		TargetRoot: targetRoot,
		Entries:    entries,
	}); err != nil {
		t.Fatalf("artifact sync request error = %v", err)
	}
	// #nosec G304 -- synced artifact path is derived from test-controlled temp directories.
	content, err := os.ReadFile(filepath.Join(targetRoot, "subdir", "hello.txt"))
	if err != nil {
		t.Fatalf("read synced artifact: %v", err)
	}
	if string(content) != "runtime-contract" {
		t.Fatalf("synced artifact = %q, want runtime-contract", string(content))
	}

	commandSession := newRuntimeCommandSession(ctx, client, func(error) {})
	output, err := commandSession.CombinedOutput("printf contract-command")
	if err != nil {
		t.Fatalf("command CombinedOutput() error = %v", err)
	}
	if string(output) != "contract-command" {
		t.Fatalf("command output = %q, want contract-command", string(output))
	}

	processEnvelope, err := client.request(ctx, runtimecontract.OperationProcessStart, runtimecontract.ProcessStartRequest{
		Command: "/bin/sh",
		Args:    []string{"-lc", "trap 'exit 0' INT; printf process-ready; sleep 30"},
	})
	if err != nil {
		t.Fatalf("process start request error = %v", err)
	}
	processResponse, err := runtimecontract.DecodePayload[runtimecontract.SessionResponse](processEnvelope)
	if err != nil {
		t.Fatalf("decode process response: %v", err)
	}
	processSession := newRuntimeManagedClientSession(ctx, client, func(error) {})
	processSession.setSessionID(processResponse.SessionID)
	client.registerSession(processResponse.SessionID, processSession)
	stdout, err := processSession.StdoutPipe()
	if err != nil {
		t.Fatalf("process StdoutPipe() error = %v", err)
	}
	buffer := make([]byte, len("process-ready"))
	if _, err := io.ReadFull(stdout, buffer); err != nil {
		t.Fatalf("read process stdout: %v", err)
	}
	if string(buffer) != "process-ready" {
		t.Fatalf("process stdout = %q, want process-ready", string(buffer))
	}
	statusEnvelope, err := client.request(
		ctx,
		runtimecontract.OperationProcessStatus,
		runtimecontract.ProcessStatusRequest(processResponse),
	)
	if err != nil {
		t.Fatalf("process status request error = %v", err)
	}
	status, err := runtimecontract.DecodePayload[runtimecontract.ProcessStatusResponse](statusEnvelope)
	if err != nil {
		t.Fatalf("decode process status: %v", err)
	}
	if !status.Running {
		t.Fatalf("expected process to be running, got %+v", status)
	}
	if err := processSession.Signal("INT"); err != nil {
		t.Fatalf("process Signal(INT) error = %v", err)
	}
	if err := processSession.Wait(); err != nil {
		t.Fatalf("process Wait() error = %v", err)
	}
}

func waitForReverseRuntimeRegistration(t *testing.T, registry *ReverseRuntimeRelayRegistry, machineID uuid.UUID) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := registry.client(machineID); err == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("reverse runtime session for machine %s did not register", machineID)
}

func runReverseRuntimeParticipant(ctx context.Context, serverURL string, machineID uuid.UUID) error {
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	defer func() { _ = conn.Close() }()
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	if err := writeMachineEnvelopeForTest(conn, machinechanneldomain.MessageTypeHello, "", machinechanneldomain.Hello{
		AgentVersion: "runtime-contract-test",
		Hostname:     "reverse-contract",
	}); err != nil {
		return err
	}
	if err := writeMachineEnvelopeForTest(conn, machinechanneldomain.MessageTypeAuthenticate, "", machinechanneldomain.Authenticate{
		Token:         machinechanneldomain.TokenPrefix + "reverse-contract",
		MachineID:     machineID.String(),
		TransportMode: "ws_reverse",
		SystemInfo: machinechanneldomain.SystemInfo{
			Hostname:          "reverse-contract",
			OS:                "linux",
			Arch:              "amd64",
			OpenASEBinaryPath: "/usr/local/bin/openase",
			AgentCLIPath:      "/bin/sh",
		},
	}); err != nil {
		return err
	}

	registeredEnvelope, err := readMachineEnvelopeForTest(conn)
	if err != nil {
		return err
	}
	registered, err := machinechanneldomain.DecodePayload[machinechanneldomain.Registered](registeredEnvelope)
	if err != nil {
		return err
	}

	var writeMu sync.Mutex
	server := NewDaemonRuntimeProtocolServer(func(ctx context.Context, envelope runtimecontract.Envelope) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeMachineEnvelopeForTest(conn, machinechanneldomain.MessageTypeRuntime, registered.SessionID, envelope)
	})
	defer server.Close()

	for {
		select {
		case <-ctx.Done():
			writeMu.Lock()
			_ = writeMachineEnvelopeForTest(conn, machinechanneldomain.MessageTypeGoodbye, registered.SessionID, machinechanneldomain.Goodbye{Reason: "shutdown"})
			writeMu.Unlock()
			return nil
		default:
		}

		envelope, err := readMachineEnvelopeForTest(conn)
		if err != nil {
			return err
		}
		switch envelope.Type {
		case machinechanneldomain.MessageTypeRuntime:
			runtimeEnvelope, err := runtimeEnvelopeFromMachineEnvelopeForTest(envelope)
			if err != nil {
				return err
			}
			if err := server.HandleEnvelope(ctx, runtimeEnvelope); err != nil {
				return err
			}
		case machinechanneldomain.MessageTypeGoodbye:
			return nil
		}
	}
}

func readMachineEnvelopeForTest(conn *websocket.Conn) (machinechanneldomain.Envelope, error) {
	_, payload, err := conn.ReadMessage()
	if err != nil {
		return machinechanneldomain.Envelope{}, err
	}
	return machinechanneldomain.ParseEnvelope(payload)
}

func writeMachineEnvelopeForTest(conn *websocket.Conn, messageType machinechanneldomain.MessageType, sessionID string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return conn.WriteJSON(machinechanneldomain.Envelope{
		Version:   machinechanneldomain.ProtocolVersion,
		Type:      messageType,
		SessionID: strings.TrimSpace(sessionID),
		Payload:   body,
	})
}

func runtimeEnvelopeFromMachineEnvelopeForTest(envelope machinechanneldomain.Envelope) (runtimecontract.Envelope, error) {
	var runtimeEnvelope runtimecontract.Envelope
	if err := json.Unmarshal(envelope.Payload, &runtimeEnvelope); err != nil {
		return runtimecontract.Envelope{}, err
	}
	return runtimeEnvelope, nil
}

func waitForContainerListenerRuntime(t *testing.T, machine catalogdomain.Machine) {
	t.Helper()

	transport := websocketTransport{mode: catalogdomain.MachineConnectionModeWSListener}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := transport.Probe(ctx, machine)
		cancel()
		if err == nil {
			return
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("listener container runtime for %s did not become reachable: %v", machine.Name, lastErr)
}
