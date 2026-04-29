package machinechannel

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestDaemonToolInventoryUsesScopedAgentCLIPaths(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	claudePath := writeExecutable(t, filepath.Join(tmpDir, "claude"))
	codexPath := writeExecutable(t, filepath.Join(tmpDir, "codex"))
	geminiPath := writeExecutable(t, filepath.Join(tmpDir, "gemini"))
	daemon := NewDaemon(nil)
	daemon.lookPath = func(command string) (string, error) {
		t.Fatalf("unexpected lookPath command %q", command)
		return "", nil
	}

	tools := daemon.toolInventory(domain.DaemonConfig{
		AgentCLIPath: "/usr/local/bin/legacy",
		AgentCLIPaths: map[string]string{
			"claude-code-cli":  claudePath,
			"codex-app-server": codexPath,
			"gemini-cli":       geminiPath,
		},
	})
	if len(tools) != 3 {
		t.Fatalf("toolInventory() len = %d, want 3", len(tools))
	}
	for _, tool := range tools {
		if !tool.Installed || !tool.Ready {
			t.Fatalf("toolInventory() entry = %+v, want installed+ready", tool)
		}
	}
}

func TestDaemonToolInventoryFallsBackToLegacyPathOnlyWhenNoScopedPaths(t *testing.T) {
	t.Parallel()

	legacyPath := writeExecutable(t, filepath.Join(t.TempDir(), "legacy"))
	daemon := NewDaemon(nil)
	daemon.lookPath = func(command string) (string, error) {
		if command != "claude" && command != "gemini" {
			t.Fatalf("unexpected lookPath command %q", command)
		}
		return command, nil
	}

	tools := daemon.toolInventory(domain.DaemonConfig{AgentCLIPath: legacyPath})
	if !tools[0].Installed || !tools[1].Installed || !tools[2].Installed {
		t.Fatalf("toolInventory() legacy fallback = %+v", tools)
	}
}

func writeExecutable(t *testing.T, path string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
	// #nosec G302 -- test wrapper must be executable inside the temp workspace.
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod executable %s: %v", path, err)
	}
	return path
}

func TestDaemonLocalRelayBridgesAPIRequestsOverMachineWebsocket(t *testing.T) {
	machineID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/machines/connect" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()

		if _, err := readJSONEnvelope(conn); err != nil {
			t.Errorf("read hello: %v", err)
			return
		}
		authEnvelope, err := readJSONEnvelope(conn)
		if err != nil {
			t.Errorf("read authenticate: %v", err)
			return
		}
		authenticate, err := domain.DecodePayload[domain.Authenticate](authEnvelope)
		if err != nil {
			t.Errorf("decode authenticate: %v", err)
			return
		}
		if authenticate.MachineID != machineID.String() {
			t.Errorf("authenticate machine_id = %q", authenticate.MachineID)
			return
		}
		if err := writeJSONEnvelope(conn, domain.Envelope{
			Version: domain.ProtocolVersion,
			Type:    domain.MessageTypeRegistered,
			Payload: mustMarshalJSON(domain.Registered{MachineID: machineID.String(), SessionID: "session-1", HeartbeatIntervalSeconds: 60}),
		}); err != nil {
			t.Errorf("write registered: %v", err)
			return
		}

		for {
			envelope, err := readJSONEnvelope(conn)
			if err != nil {
				return
			}
			if envelope.Type != domain.MessageTypeAPIRequest {
				continue
			}
			request, err := domain.DecodePayload[domain.APIRelayRequest](envelope)
			if err != nil {
				t.Errorf("decode api request: %v", err)
				return
			}
			if err := writeJSONEnvelope(conn, domain.Envelope{
				Version:   domain.ProtocolVersion,
				Type:      domain.MessageTypeAPIResponse,
				SessionID: "session-1",
				Payload: mustMarshalJSON(domain.APIRelayResponse{
					RequestID:  request.RequestID,
					StatusCode: http.StatusCreated,
					Status:     "201 Created",
					Headers:    map[string][]string{"Content-Type": {"application/json"}},
					Body:       []byte(`{"ok":true}`),
				}),
			}); err != nil {
				t.Errorf("write api response: %v", err)
			}
			return
		}
	}))
	defer server.Close()

	relayAddress := freeLocalRelayAddress(t)
	t.Setenv(domain.EnvMachineLocalRelayAddress, relayAddress)

	daemon := NewDaemon(nil)
	errCh := make(chan error, 1)
	go func() {
		errCh <- daemon.Run(ctx, domain.DaemonConfig{
			MachineID:         machineID,
			Token:             "ase_machine_test",
			ControlPlaneURL:   server.URL,
			HeartbeatInterval: time.Minute,
			ReconnectBackoff:  10 * time.Millisecond,
		})
	}()

	relayURL := waitForLocalRelay(t, "http://"+relayAddress)
	body, err := json.Marshal(domain.LocalRelayRequest{Method: http.MethodGet, URL: "https://control.example.com/api/v1/healthz"})
	if err != nil {
		t.Fatalf("marshal local relay request: %v", err)
	}
	var relayResponse domain.LocalRelayResponse
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Post(relayURL+"/__openase_cli_relay", "application/json", bytes.NewReader(body))
		if err == nil {
			func() {
				defer func() { _ = response.Body.Close() }()
				relayResponse = domain.LocalRelayResponse{}
				_ = json.NewDecoder(response.Body).Decode(&relayResponse)
			}()
			if relayResponse.StatusCode == http.StatusCreated {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if relayResponse.StatusCode != http.StatusCreated {
		t.Fatalf("relay status code = %d, want %d (error=%q)", relayResponse.StatusCode, http.StatusCreated, relayResponse.Error)
	}
	if string(relayResponse.Body) != `{"ok":true}` {
		t.Fatalf("relay body = %s", string(relayResponse.Body))
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("daemon.Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for daemon shutdown")
	}
}

func freeLocalRelayAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for relay address: %v", err)
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().String()
}

func waitForLocalRelay(t *testing.T, relayURL string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(relayURL + "/healthz")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return relayURL
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for local relay %s", relayURL)
	return ""
}
