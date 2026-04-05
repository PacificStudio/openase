package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	executable "github.com/BetterAndBetterII/openase/internal/infra/executable"
	otelinfra "github.com/BetterAndBetterII/openase/internal/infra/otel"
	machinechannelservice "github.com/BetterAndBetterII/openase/internal/machinechannel"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	machinechannelrepo "github.com/BetterAndBetterII/openase/internal/repo/machinechannel"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestMachineConnectWebsocketRegistersMachineSession(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	machineID := createReverseWebsocketMachine(t, client)
	service := machinechannelservice.NewService(machinechannelrepo.NewEntRepository(client))
	issued, err := service.IssueToken(ctx, domain.IssueInput{MachineID: machineID, TTL: time.Hour})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
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
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	conn := dialMachineWebsocket(t, httpServer.URL)
	defer func() { _ = conn.Close() }()

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
		ToolInventory: []domain.ToolInfo{
			{Name: "claude_code", Installed: false, Ready: false, AuthStatus: "unknown", AuthMode: "unknown"},
			{Name: "codex", Installed: true, Ready: true, AuthStatus: "logged_in", AuthMode: "login"},
			{Name: "gemini", Installed: true, Ready: true, AuthStatus: "unknown", AuthMode: "unknown"},
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
	registered, err := domain.DecodePayload[domain.Registered](registeredEnvelope)
	if err != nil {
		t.Fatalf("decode registered payload: %v", err)
	}
	if registered.SessionID == "" || registered.MachineID != machineID.String() {
		t.Fatalf("unexpected registered payload: %+v", registered)
	}

	machineItem, err := client.Machine.Get(ctx, machineID)
	if err != nil {
		t.Fatalf("reload machine: %v", err)
	}
	if !machineItem.DaemonRegistered || machineItem.DaemonSessionID == "" {
		t.Fatalf("expected daemon registration fields to be stored, got %+v", machineItem)
	}
	if machineItem.Status != entmachine.StatusOnline {
		t.Fatalf("expected reverse websocket registration to mark machine online, got %+v", machineItem)
	}
	if machineItem.Resources["transport"] != "ws_reverse" {
		t.Fatalf("expected resources transport ws_reverse, got %+v", machineItem.Resources)
	}
}

func TestMachineConnectWebsocketRejectsInvalidToken(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	machineID := createReverseWebsocketMachine(t, client)
	service := machinechannelservice.NewService(machinechannelrepo.NewEntRepository(client))

	server := NewServer(
		config.ServerConfig{Port: 40023},
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
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	conn := dialMachineWebsocket(t, httpServer.URL)
	defer func() { _ = conn.Close() }()

	if err := writeMachineEnvelope(conn, domain.MessageTypeHello, "", domain.Hello{
		AgentVersion: "test-daemon",
		Hostname:     "reverse-builder",
	}); err != nil {
		t.Fatalf("write hello: %v", err)
	}
	if err := writeMachineEnvelope(conn, domain.MessageTypeAuthenticate, "", domain.Authenticate{
		Token:         "ase_machine_invalid",
		MachineID:     machineID.String(),
		TransportMode: "ws_reverse",
		SystemInfo:    domain.SystemInfo{Hostname: "reverse-builder", OS: "linux", Arch: "amd64"},
	}); err != nil {
		t.Fatalf("write authenticate: %v", err)
	}

	envelope, err := readMachineEnvelope(conn)
	if err != nil {
		t.Fatalf("read error envelope: %v", err)
	}
	if envelope.Type != domain.MessageTypeError {
		t.Fatalf("expected error envelope, got %+v", envelope)
	}
	payload, err := domain.DecodePayload[domain.ErrorPayload](envelope)
	if err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Code != "token_invalid" {
		t.Fatalf("expected token_invalid failure, got %+v", payload)
	}

	machineItem, err := client.Machine.Get(ctx, machineID)
	if err != nil {
		t.Fatalf("reload machine: %v", err)
	}
	if machineItem.DaemonRegistered {
		t.Fatalf("expected invalid auth to keep machine unregistered, got %+v", machineItem)
	}
}

func TestMachineConnectWebsocketPublishesActivityAndMetrics(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	projectID, machineID := bindReverseWebsocketMachineToProject(t, client)
	service := machinechannelservice.NewService(machinechannelrepo.NewEntRepository(client))
	issued, err := service.IssueToken(ctx, domain.IssueInput{MachineID: machineID, TTL: time.Hour})
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	metricsProvider, err := otelinfra.NewMetricsProvider(context.Background(), otelinfra.MetricsConfig{
		ServiceName: "openase",
		Prometheus:  true,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewMetricsProvider returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := metricsProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(ticketrepo.NewEntRepository(client)),
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
		WithMachineChannel(service, machinechannelservice.NewSessionRegistry(machinechannelservice.DefaultHeartbeatTimeout)),
		WithMetricsProvider(metricsProvider),
		WithMetricsHandler(metricsProvider.PrometheusHandler()),
	)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	conn1 := dialMachineWebsocket(t, httpServer.URL)
	defer func() { _ = conn1.Close() }()
	authenticateMachineWebsocket(t, conn1, issued.Token, machineID)
	if _, err := readMachineEnvelope(conn1); err != nil {
		t.Fatalf("read first registered envelope: %v", err)
	}

	conn2 := dialMachineWebsocket(t, httpServer.URL)
	defer func() { _ = conn2.Close() }()
	authenticateMachineWebsocket(t, conn2, issued.Token, machineID)
	registeredEnvelope, err := readMachineEnvelope(conn2)
	if err != nil {
		t.Fatalf("read second registered envelope: %v", err)
	}
	if registeredEnvelope.Type != domain.MessageTypeRegistered {
		t.Fatalf("expected registered envelope, got %+v", registeredEnvelope)
	}

	metricsBody := scrapeMetrics(t, server.Handler())
	for _, expected := range []string{
		`openase_machine_channel_active_sessions{transport_mode="ws_reverse"} 1`,
		`openase_machine_channel_websocket_reconnect_total{transport_mode="ws_reverse"} 1`,
		`openase_machine_channel_events_total{event="registered",transport_mode="ws_reverse"} 2`,
		`openase_machine_channel_events_total{event="reconnected",transport_mode="ws_reverse"} 1`,
	} {
		if !strings.Contains(metricsBody, expected) {
			t.Fatalf("expected metrics to contain %q, got %q", expected, metricsBody)
		}
	}

	if err := writeMachineEnvelope(conn2, domain.MessageTypeGoodbye, "", domain.Goodbye{Reason: "test complete"}); err != nil {
		t.Fatalf("write goodbye: %v", err)
	}
	activities := waitForProjectActivityEvents(ctx, t, client, projectID, 3)
	if len(activities) != 3 {
		t.Fatalf("expected three machine activity events, got %+v", activities)
	}
	eventTypes := map[string]map[string]any{}
	for _, item := range activities {
		eventTypes[item.EventType] = item.Metadata
	}
	for _, expectedType := range []string{"machine.connected", "machine.reconnected", "machine.disconnected"} {
		if _, ok := eventTypes[expectedType]; !ok {
			t.Fatalf("expected activity event %s, got %+v", expectedType, activities)
		}
	}
	if eventTypes["machine.reconnected"]["transport_mode"] != "ws_reverse" {
		t.Fatalf("expected transport_mode in reconnect activity, got %+v", eventTypes["machine.reconnected"])
	}
}

func TestMachineConnectWebsocketAuthFailurePublishesActivityAndMetric(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	projectID, machineID := bindReverseWebsocketMachineToProject(t, client)
	service := machinechannelservice.NewService(machinechannelrepo.NewEntRepository(client))

	metricsProvider, err := otelinfra.NewMetricsProvider(context.Background(), otelinfra.MetricsConfig{
		ServiceName: "openase",
		Prometheus:  true,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewMetricsProvider returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := metricsProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(ticketrepo.NewEntRepository(client)),
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
		WithMachineChannel(service, machinechannelservice.NewSessionRegistry(machinechannelservice.DefaultHeartbeatTimeout)),
		WithMetricsProvider(metricsProvider),
		WithMetricsHandler(metricsProvider.PrometheusHandler()),
	)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	conn := dialMachineWebsocket(t, httpServer.URL)
	defer func() { _ = conn.Close() }()
	authenticateMachineWebsocket(t, conn, "ase_machine_invalid", machineID)

	envelope, err := readMachineEnvelope(conn)
	if err != nil {
		t.Fatalf("read error envelope: %v", err)
	}
	if envelope.Type != domain.MessageTypeError {
		t.Fatalf("expected error envelope, got %+v", envelope)
	}

	activities := waitForProjectActivityEvents(ctx, t, client, projectID, 1)
	if len(activities) != 1 || activities[0].EventType != "machine.daemon_auth_failed" {
		t.Fatalf("expected machine.daemon_auth_failed activity, got %+v", activities)
	}
	if activities[0].Metadata["failure_code"] != "token_invalid" {
		t.Fatalf("expected token_invalid failure code, got %+v", activities[0].Metadata)
	}

	metricsBody := scrapeMetrics(t, server.Handler())
	if !strings.Contains(metricsBody, `openase_machine_channel_events_total{event="auth_failed",transport_mode="ws_reverse"} 1`) {
		t.Fatalf("expected auth_failed metric, got %q", metricsBody)
	}
}

func createReverseWebsocketMachine(t *testing.T, client *ent.Client) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Reverse Org").
		SetSlug("reverse-org").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	machineItem, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("reverse-builder").
		SetHost("reverse-builder.example.com").
		SetConnectionMode(entmachine.ConnectionModeWsReverse).
		SetStatus(entmachine.StatusOffline).
		SetResources(map[string]any{}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create reverse websocket machine: %v", err)
	}
	return machineItem.ID
}

func bindReverseWebsocketMachineToProject(t *testing.T, client *ent.Client) (uuid.UUID, uuid.UUID) {
	t.Helper()

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Reverse Org").
		SetSlug("reverse-org-project").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Reverse Project").
		SetSlug("reverse-project").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("reverse-builder").
		SetHost("reverse-builder.example.com").
		SetConnectionMode(entmachine.ConnectionModeWsReverse).
		SetStatus(entmachine.StatusOffline).
		SetResources(map[string]any{}).
		SetSSHUser("openase").
		SetSSHKeyPath("/tmp/reverse.pem").
		Save(ctx)
	if err != nil {
		t.Fatalf("create reverse websocket machine: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machine.ID).
		SetName("Reverse Provider").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if _, err := client.Agent.Create().
		SetProviderID(providerItem.ID).
		SetProjectID(project.ID).
		SetName("reverse-agent").
		Save(ctx); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	return project.ID, machine.ID
}

func waitForProjectActivityEvents(ctx context.Context, t *testing.T, client *ent.Client, projectID uuid.UUID, want int) []*ent.ActivityEvent {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		activities, err := client.ActivityEvent.Query().
			Where(entactivityevent.ProjectIDEQ(projectID)).
			Order(ent.Asc(entactivityevent.FieldCreatedAt)).
			All(ctx)
		if err != nil {
			t.Fatalf("list activity events: %v", err)
		}
		if len(activities) >= want {
			return activities
		}
		if time.Now().After(deadline) {
			return activities
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func authenticateMachineWebsocket(t *testing.T, conn *websocket.Conn, token string, machineID uuid.UUID) {
	t.Helper()

	if err := writeMachineEnvelope(conn, domain.MessageTypeHello, "", domain.Hello{
		AgentVersion: "test-daemon",
		Hostname:     "reverse-builder",
	}); err != nil {
		t.Fatalf("write hello: %v", err)
	}
	if err := writeMachineEnvelope(conn, domain.MessageTypeAuthenticate, "", domain.Authenticate{
		Token:         token,
		MachineID:     machineID.String(),
		TransportMode: "ws_reverse",
		SystemInfo:    domain.SystemInfo{Hostname: "reverse-builder", OS: "linux", Arch: "amd64"},
	}); err != nil {
		t.Fatalf("write authenticate: %v", err)
	}
}

func scrapeMetrics(t *testing.T, handler http.Handler) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/metrics", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected metrics route to return 200, got %d", rec.Code)
	}
	return rec.Body.String()
}

func dialMachineWebsocket(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/api/v1/machines/connect"
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if response != nil && response.Body != nil {
		defer func() {
			_ = response.Body.Close()
		}()
	}
	if err != nil {
		t.Fatalf("dial websocket %s: %v", wsURL, err)
	}
	return conn
}
