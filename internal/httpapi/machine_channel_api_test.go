package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	machinechannelservice "github.com/BetterAndBetterII/openase/internal/machinechannel"
	machinechannelrepo "github.com/BetterAndBetterII/openase/internal/repo/machinechannel"
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
