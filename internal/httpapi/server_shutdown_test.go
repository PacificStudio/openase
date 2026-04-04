package httpapi

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	machinechannelservice "github.com/BetterAndBetterII/openase/internal/machinechannel"
	machinechannelrepo "github.com/BetterAndBetterII/openase/internal/repo/machinechannel"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestServerRunShutdownClosesActiveProjectEventStream(t *testing.T) {
	port := reserveShutdownTestPort(t)
	server := NewServer(
		config.ServerConfig{
			Host:            "127.0.0.1",
			Port:            port,
			ReadTimeout:     time.Second,
			WriteTimeout:    time.Second,
			ShutdownTimeout: 500 * time.Millisecond,
		},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	baseURL, stopServer, runErrCh := startShutdownTestServer(t, server)

	response, err := http.Get(baseURL + "/api/v1/projects/" + uuid.NewString() + "/events/stream")
	if err != nil {
		t.Fatalf("open project event stream: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	reader := bufio.NewReader(response.Body)
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read initial keepalive line: %v", err)
	}
	if firstLine != ": keepalive\n" {
		t.Fatalf("expected initial keepalive line, got %q", firstLine)
	}
	secondLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read keepalive separator: %v", err)
	}
	if secondLine != "\n" {
		t.Fatalf("expected keepalive separator, got %q", secondLine)
	}

	bodyCh := make(chan string, 1)
	go func() {
		bytes, _ := io.ReadAll(reader)
		bodyCh <- firstLine + secondLine + string(bytes)
	}()

	stopServer()
	waitForServerRunResult(t, runErrCh)

	select {
	case body := <-bodyCh:
		if !strings.Contains(body, ": keepalive\n\n") {
			t.Fatalf("expected SSE body to contain initial keepalive, got %q", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server-side SSE shutdown")
	}
}

func TestServerRunShutdownClosesReverseWebsocketSessions(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	machineID := createReverseWebsocketMachine(t, client)
	service := machinechannelservice.NewService(machinechannelrepo.NewEntRepository(client))
	issued, err := service.IssueToken(ctx, domain.IssueInput{MachineID: machineID, TTL: time.Hour})
	if err != nil {
		t.Fatalf("issue machine channel token: %v", err)
	}

	registry := machinechannelservice.NewSessionRegistry(machinechannelservice.DefaultHeartbeatTimeout)
	server := NewServer(
		config.ServerConfig{
			Host:            "127.0.0.1",
			Port:            reserveShutdownTestPort(t),
			ReadTimeout:     time.Second,
			WriteTimeout:    time.Second,
			ShutdownTimeout: 500 * time.Millisecond,
		},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithMachineChannel(service, registry),
	)

	baseURL, stopServer, runErrCh := startShutdownTestServer(t, server)

	conn := dialMachineWebsocket(t, baseURL)
	defer func() {
		_ = conn.Close()
	}()

	if err := writeMachineEnvelope(conn, domain.MessageTypeHello, "", domain.Hello{
		AgentVersion: "test-daemon",
		Hostname:     "reverse-builder",
	}); err != nil {
		t.Fatalf("write hello: %v", err)
	}
	if err := writeMachineEnvelope(conn, domain.MessageTypeAuthenticate, "", domain.Authenticate{
		Token:         issued.Token,
		MachineID:     machineID.String(),
		TransportMode: "ws_reverse",
		SystemInfo: domain.SystemInfo{
			Hostname:          "reverse-builder",
			OS:                "linux",
			Arch:              "amd64",
			OpenASEBinaryPath: "/usr/local/bin/openase",
			AgentCLIPath:      "/usr/local/bin/codex",
		},
	}); err != nil {
		t.Fatalf("write authenticate: %v", err)
	}

	registeredEnvelope, err := readMachineEnvelope(conn)
	if err != nil {
		t.Fatalf("read registered envelope: %v", err)
	}
	if registeredEnvelope.Type != domain.MessageTypeRegistered {
		t.Fatalf("expected registered envelope, got %+v", registeredEnvelope)
	}

	stopServer()

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil || !websocket.IsCloseError(err, websocket.ClosePolicyViolation) {
		t.Fatalf("expected shutdown close frame from server, got %v", err)
	}

	waitForServerRunResult(t, runErrCh)

	machineItem, err := client.Machine.Get(ctx, machineID)
	if err != nil {
		t.Fatalf("reload machine after shutdown: %v", err)
	}
	if machineItem.Status != entmachine.StatusOffline {
		t.Fatalf("expected machine to be offline after shutdown, got %+v", machineItem)
	}
	if machineItem.DaemonRegistered {
		t.Fatalf("expected daemon registration to be cleared after shutdown, got %+v", machineItem)
	}
	if _, ok := registry.Snapshot(machineID); ok {
		t.Fatal("expected reverse websocket session registry to be drained on shutdown")
	}
}

func startShutdownTestServer(t *testing.T, server *Server) (string, context.CancelFunc, <-chan error) {
	t.Helper()

	//nolint:gosec // returned to the caller so the test can trigger shutdown
	runCtx, cancel := context.WithCancel(context.Background())
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- server.Run(runCtx)
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", server.cfg.Port)
	waitForShutdownTestServerReady(t, baseURL, runErrCh)
	return baseURL, cancel, runErrCh
}

func waitForShutdownTestServerReady(t *testing.T, baseURL string, runErrCh <-chan error) {
	t.Helper()

	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-runErrCh:
			t.Fatalf("server exited before readiness: %v", err)
		default:
		}

		response, err := client.Get(baseURL + "/healthz")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for server readiness at %s", baseURL)
}

func waitForServerRunResult(t *testing.T, runErrCh <-chan error) {
	t.Helper()

	select {
	case err := <-runErrCh:
		if err != nil {
			t.Fatalf("Server.Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Server.Run to return")
	}
}

func reserveShutdownTestPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve tcp port: %v", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	return listener.Addr().(*net.TCPAddr).Port
}
