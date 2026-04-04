package httpapi

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	machinechannelservice "github.com/BetterAndBetterII/openase/internal/machinechannel"
	chatrepo "github.com/BetterAndBetterII/openase/internal/repo/chatconversation"
	machinechannelrepo "github.com/BetterAndBetterII/openase/internal/repo/machinechannel"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestServerRunShutdownCutsActiveEventStream(t *testing.T) {
	server := NewServer(
		shutdownTestServerConfig(t),
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	baseURL, stopServer, errCh := startShutdownTestServer(t, server)
	response, err := http.Get(baseURL + "/api/v1/events/stream?topic=runtime.events")
	if err != nil {
		stopServer()
		t.Fatalf("open event stream: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		stopServer()
		t.Fatalf("read initial keepalive: %v", err)
	}
	if strings.TrimSpace(line) != ": keepalive" {
		stopServer()
		t.Fatalf("expected initial keepalive comment, got %q", line)
	}

	streamClosed := drainReader(reader)
	stopServer()

	if elapsed, err := waitForServerRunResult(errCh); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	} else if elapsed > 2*time.Second {
		t.Fatalf("Run() took %s after shutdown, want under 2s", elapsed)
	}
	waitForReaderDrain(t, streamClosed, "event stream")
}

func TestServerRunShutdownCutsProjectConversationMuxStream(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)

	org, err := client.Organization.Create().
		SetName("Shutdown Org").
		SetSlug("shutdown-org").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Shutdown Project").
		SetSlug("shutdown-project").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	repoStore := chatrepo.NewEntRepository(client)
	conversation, err := repoStore.CreateConversation(ctx, chatdomain.CreateConversation{
		ProjectID:  project.ID,
		UserID:     "user:shutdown",
		Source:     chatdomain.SourceProjectSidebar,
		ProviderID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	projectConversationService := chatservice.NewProjectConversationService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		repoStore,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	server := NewServer(
		shutdownTestServerConfig(t),
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithProjectConversationService(projectConversationService),
	)

	baseURL, stopServer, errCh := startShutdownTestServer(t, server)
	request, err := http.NewRequest(
		http.MethodGet,
		baseURL+"/api/v1/chat/projects/"+project.ID.String()+"/conversations/stream",
		nil,
	)
	if err != nil {
		stopServer()
		t.Fatalf("new mux stream request: %v", err)
	}
	request.Header.Set(chatUserHeader, "user:shutdown")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		stopServer()
		t.Fatalf("open mux stream: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	reader := bufio.NewReader(response.Body)
	frame := readProjectConversationSSEFrame(t, reader)
	if frame.Event != "session" || !strings.Contains(frame.Data, conversation.ID.String()) {
		stopServer()
		t.Fatalf("expected initial session frame for %s, got %+v", conversation.ID, frame)
	}

	streamClosed := drainReader(reader)
	stopServer()

	if elapsed, err := waitForServerRunResult(errCh); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	} else if elapsed > 2*time.Second {
		t.Fatalf("Run() took %s after shutdown, want under 2s", elapsed)
	}
	waitForReaderDrain(t, streamClosed, "project conversation mux stream")
}

func TestServerRunShutdownCutsMachineWebsocket(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	machineID := createReverseWebsocketMachine(t, client)
	service := machinechannelservice.NewService(machinechannelrepo.NewEntRepository(client))
	issued, err := service.IssueToken(ctx, domain.IssueInput{MachineID: machineID, TTL: time.Hour})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	server := NewServer(
		shutdownTestServerConfig(t),
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithMachineChannel(service, machinechannelservice.NewSessionRegistry(machinechannelservice.DefaultHeartbeatTimeout)),
	)

	baseURL, stopServer, errCh := startShutdownTestServer(t, server)
	conn := dialMachineWebsocket(t, baseURL)
	defer func() { _ = conn.Close() }()

	if err := writeMachineEnvelope(conn, domain.MessageTypeHello, "", domain.Hello{
		AgentVersion: "test-daemon",
		Hostname:     "shutdown-builder",
	}); err != nil {
		stopServer()
		t.Fatalf("write hello: %v", err)
	}
	if err := writeMachineEnvelope(conn, domain.MessageTypeAuthenticate, "", domain.Authenticate{
		Token:         issued.Token,
		MachineID:     machineID.String(),
		TransportMode: "ws_reverse",
		SystemInfo:    domain.SystemInfo{Hostname: "shutdown-builder", OS: "linux", Arch: "amd64"},
	}); err != nil {
		stopServer()
		t.Fatalf("write authenticate: %v", err)
	}

	registeredEnvelope, err := readMachineEnvelope(conn)
	if err != nil {
		stopServer()
		t.Fatalf("read registered envelope: %v", err)
	}
	if registeredEnvelope.Type != domain.MessageTypeRegistered {
		stopServer()
		t.Fatalf("expected registered envelope, got %+v", registeredEnvelope)
	}

	readErrCh := make(chan error, 1)
	go func() {
		_, _, err := conn.ReadMessage()
		readErrCh <- err
	}()

	stopServer()

	if elapsed, err := waitForServerRunResult(errCh); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	} else if elapsed > 2*time.Second {
		t.Fatalf("Run() took %s after shutdown, want under 2s", elapsed)
	}
	select {
	case err := <-readErrCh:
		if err == nil {
			t.Fatal("expected websocket read to fail after server shutdown")
		}
		var closeErr *websocket.CloseError
		if !errors.As(err, &closeErr) {
			t.Fatalf("expected websocket close error after shutdown, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for websocket shutdown")
	}
}

func shutdownTestServerConfig(t *testing.T) config.ServerConfig {
	t.Helper()

	return config.ServerConfig{
		Host:            "127.0.0.1",
		Port:            freeShutdownTestPort(t),
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		ShutdownTimeout: 300 * time.Millisecond,
	}
}

func startShutdownTestServer(t *testing.T, server *Server) (string, context.CancelFunc, <-chan error) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", server.cfg.Port)
	waitForServerReady(t, baseURL+"/healthz")
	return baseURL, cancel, errCh
}

func waitForServerReady(t *testing.T, url string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		//nolint:gosec // test helper polls the localhost server it just started
		response, err := http.Get(url)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for server readiness at %s", url)
}

func waitForServerRunResult(errCh <-chan error) (time.Duration, error) {
	startedAt := time.Now()
	select {
	case err := <-errCh:
		return time.Since(startedAt), err
	case <-time.After(5 * time.Second):
		return time.Since(startedAt), errors.New("timed out waiting for Server.Run")
	}
}

func drainReader(reader io.Reader) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, reader)
		errCh <- err
	}()
	return errCh
}

func waitForReaderDrain(t *testing.T, errCh <-chan error, streamName string) {
	t.Helper()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("%s read error = %v, want nil", streamName, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s to close", streamName)
	}
}

func freeShutdownTestPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free port: %v", err)
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().(*net.TCPAddr).Port
}
