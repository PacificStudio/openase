package machinetransport

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinechanneldomain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
	"github.com/BetterAndBetterII/openase/internal/testutil/containerharness"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestWebsocketReverseRuntimeContainerE2E(t *testing.T) {
	containerharness.RequireContainerSuite(t)

	openaseBinary := containerharness.BuiltOpenASEBinary(t)
	machineID := uuid.New()
	sessionRegistry := NewReverseRuntimeRelayRegistry()
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		}),
	}

	// #nosec G102 -- the reverse-daemon container reaches this host listener through host-gateway.
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("listen for reverse control plane: %v", err)
	}
	serverPort := listener.Addr().(*net.TCPAddr).Port
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Serve(listener)
	}()
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		if err := <-serverErrCh; err != nil && err != http.ErrServerClosed {
			t.Errorf("reverse control plane server error: %v", err)
		}
	})

	project := containerharness.NewProject(t, containerharness.Options{
		ProjectName: "ase41-reverse-" + strings.ToLower(strings.ReplaceAll(uuid.NewString(), "-", "")),
		Env: map[string]string{
			"OPENASE_TEST_OPENASE_BINARY":        openaseBinary,
			"OPENASE_TEST_TMP_ROOT":              filepath.Clean(os.TempDir()),
			"OPENASE_TEST_UID":                   fmt.Sprintf("%d", os.Getuid()),
			"OPENASE_TEST_GID":                   fmt.Sprintf("%d", os.Getgid()),
			"OPENASE_MACHINE_ID":                 machineID.String(),
			"OPENASE_MACHINE_CHANNEL_TOKEN":      machinechanneldomain.TokenPrefix + "reverse-container",
			"OPENASE_MACHINE_CONTROL_PLANE_URL":  fmt.Sprintf("http://host.docker.internal:%d", serverPort),
			"OPENASE_MACHINE_HEARTBEAT_INTERVAL": "200ms",
		},
	})
	project.Up(t, nil, "reverse-daemon")
	project.WriteLogs(t, "reverse-daemon-compose.log", nil, "reverse-daemon")

	waitForReverseRuntimeRegistration(t, sessionRegistry, machineID)

	machine := catalogdomain.Machine{
		ID:             machineID,
		Name:           "reverse-container",
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
}
