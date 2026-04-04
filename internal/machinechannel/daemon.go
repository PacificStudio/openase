package machinechannel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	"github.com/gorilla/websocket"
)

type Daemon struct {
	dialer     *websocket.Dialer
	now        func() time.Time
	hostname   func() (string, error)
	executable func() (string, error)
	lookPath   func(string) (string, error)
	logger     *slog.Logger
}

func NewDaemon(logger *slog.Logger) *Daemon {
	if logger == nil {
		logger = slog.Default()
	}
	return &Daemon{
		dialer:     websocket.DefaultDialer,
		now:        time.Now,
		hostname:   os.Hostname,
		executable: os.Executable,
		lookPath:   exec.LookPath,
		logger:     logger.With("component", "machine-channel-daemon"),
	}
}

func (d *Daemon) Run(ctx context.Context, config domain.DaemonConfig) error {
	if d == nil || d.dialer == nil {
		return fmt.Errorf("machine daemon unavailable")
	}

	backoff := config.ReconnectBackoff
	if backoff <= 0 {
		backoff = DefaultReconnectBackoff
	}

	for {
		err := d.runSession(ctx, config)
		if err == nil || errors.Is(err, context.Canceled) {
			return nil
		}
		var connectionErr machineConnectionError
		if errors.As(err, &connectionErr) && connectionErr.Fatal {
			return err
		}
		d.logger.Warn(
			"machine reverse websocket session ended",
			"machine_id", config.MachineID.String(),
			"error", err,
			"reconnect_backoff", backoff.String(),
		)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
	}
}

func (d *Daemon) runSession(ctx context.Context, config domain.DaemonConfig) error {
	connectURL, err := machineConnectURL(config.ControlPlaneURL)
	if err != nil {
		return err
	}

	conn, _, err := d.dialer.DialContext(ctx, connectURL, nil)
	if err != nil {
		return fmt.Errorf("connect machine websocket: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	systemInfo := d.systemInfo(config)
	toolInventory := d.toolInventory(config)

	if err := writeJSONEnvelope(conn, domain.Envelope{
		Version: domain.ProtocolVersion,
		Type:    domain.MessageTypeHello,
		Payload: mustMarshalJSON(domain.Hello{
			AgentVersion: "openase-machine-agent",
			Hostname:     systemInfo.Hostname,
		}),
	}); err != nil {
		return fmt.Errorf("send hello: %w", err)
	}

	if err := writeJSONEnvelope(conn, domain.Envelope{
		Version: domain.ProtocolVersion,
		Type:    domain.MessageTypeAuthenticate,
		Payload: mustMarshalJSON(domain.Authenticate{
			Token:         config.Token,
			MachineID:     config.MachineID.String(),
			TransportMode: "ws_reverse",
			SystemInfo:    systemInfo,
			Capabilities: []string{
				"probe",
				"workspace_prepare",
				"artifact_sync",
				"process_streaming",
			},
			ToolInventory: toolInventory,
		}),
	}); err != nil {
		return fmt.Errorf("send authenticate: %w", err)
	}

	envelope, err := readJSONEnvelope(conn)
	if err != nil {
		return fmt.Errorf("read machine registration: %w", err)
	}
	if envelope.Type == domain.MessageTypeError {
		return parseMachineConnectionError(envelope)
	}
	if envelope.Type != domain.MessageTypeRegistered {
		return fmt.Errorf("expected registered message, got %q", envelope.Type)
	}
	registered, err := domain.DecodePayload[domain.Registered](envelope)
	if err != nil {
		return err
	}
	heartbeatInterval := config.HeartbeatInterval
	if registered.HeartbeatIntervalSeconds > 0 {
		heartbeatInterval = time.Duration(registered.HeartbeatIntervalSeconds) * time.Second
	}
	if heartbeatInterval <= 0 {
		heartbeatInterval = DefaultHeartbeatInterval
	}

	d.logger.Info(
		"machine reverse websocket registered",
		"machine_id", config.MachineID.String(),
		"session_id", registered.SessionID,
		"transport", "ws_reverse",
		"heartbeat_interval", heartbeatInterval.String(),
		"replaced_previous_session", registered.ReplacedPreviousSession,
	)

	readErrCh := make(chan error, 1)
	go func() {
		for {
			incoming, readErr := readJSONEnvelope(conn)
			if readErr != nil {
				readErrCh <- readErr
				return
			}
			if incoming.Type == domain.MessageTypeError {
				readErrCh <- parseMachineConnectionError(incoming)
				return
			}
		}
	}()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = writeJSONEnvelope(conn, domain.Envelope{
				Version:   domain.ProtocolVersion,
				Type:      domain.MessageTypeGoodbye,
				SessionID: registered.SessionID,
				Payload:   mustMarshalJSON(domain.Goodbye{Reason: "shutdown"}),
			})
			return nil
		case err := <-readErrCh:
			return err
		case now := <-ticker.C:
			if err := writeJSONEnvelope(conn, domain.Envelope{
				Version:   domain.ProtocolVersion,
				Type:      domain.MessageTypeHeartbeat,
				SessionID: registered.SessionID,
				Payload: mustMarshalJSON(domain.Heartbeat{
					SentAt:        now.UTC().Format(time.RFC3339),
					SystemInfo:    &systemInfo,
					ToolInventory: toolInventory,
				}),
			}); err != nil {
				return fmt.Errorf("send heartbeat: %w", err)
			}
		}
	}
}

func (d *Daemon) systemInfo(config domain.DaemonConfig) domain.SystemInfo {
	hostname := "unknown"
	if d != nil && d.hostname != nil {
		if value, err := d.hostname(); err == nil && strings.TrimSpace(value) != "" {
			hostname = strings.TrimSpace(value)
		}
	}

	openaseBinaryPath := strings.TrimSpace(config.OpenASEBinaryPath)
	if openaseBinaryPath == "" && d != nil && d.executable != nil {
		if value, err := d.executable(); err == nil && strings.TrimSpace(value) != "" {
			openaseBinaryPath = strings.TrimSpace(value)
		}
	}
	if openaseBinaryPath != "" {
		openaseBinaryPath = filepath.Clean(openaseBinaryPath)
	}

	return domain.SystemInfo{
		Hostname:          hostname,
		OS:                runtime.GOOS,
		Arch:              runtime.GOARCH,
		OpenASEBinaryPath: openaseBinaryPath,
		AgentCLIPath:      strings.TrimSpace(config.AgentCLIPath),
	}
}

func (d *Daemon) toolInventory(config domain.DaemonConfig) []domain.ToolInfo {
	type toolSpec struct {
		name    string
		command string
	}

	specs := []toolSpec{
		{name: "claude_code", command: "claude"},
		{name: "codex", command: firstNonEmptyString(strings.TrimSpace(config.AgentCLIPath), "codex")},
		{name: "gemini", command: "gemini"},
	}

	tools := make([]domain.ToolInfo, 0, len(specs))
	for _, spec := range specs {
		command := strings.TrimSpace(spec.command)
		info := domain.ToolInfo{
			Name:       spec.name,
			AuthStatus: "unknown",
			AuthMode:   "unknown",
		}
		if command == "" {
			tools = append(tools, info)
			continue
		}
		path, err := resolveCommandPath(d.lookPath, command)
		if err == nil {
			info.Installed = true
			info.Ready = true
			if spec.name == "codex" && strings.TrimSpace(path) != "" {
				info.Version = ""
			}
		}
		tools = append(tools, info)
	}
	return tools
}

func resolveCommandPath(lookPath func(string) (string, error), command string) (string, error) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", fmt.Errorf("command path must not be empty")
	}
	if strings.Contains(trimmed, string(os.PathSeparator)) {
		info, err := os.Stat(trimmed)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("%s is a directory", trimmed)
		}
		return filepath.Clean(trimmed), nil
	}
	if lookPath == nil {
		return "", fmt.Errorf("lookPath unavailable")
	}
	return lookPath(trimmed)
}

type machineConnectionError struct {
	Code       string
	Message    string
	RetryAfter time.Duration
	Fatal      bool
}

func (e machineConnectionError) Error() string {
	if strings.TrimSpace(e.Code) == "" {
		return strings.TrimSpace(e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, strings.TrimSpace(e.Message))
}

func parseMachineConnectionError(envelope domain.Envelope) error {
	payload, err := domain.DecodePayload[domain.ErrorPayload](envelope)
	if err != nil {
		return err
	}
	connectionErr := machineConnectionError{
		Code:       strings.TrimSpace(payload.Code),
		Message:    strings.TrimSpace(payload.Message),
		RetryAfter: time.Duration(payload.RetryAfterSeconds) * time.Second,
	}
	switch connectionErr.Code {
	case "token_invalid", "token_revoked", "token_expired", "machine_disabled", "mode_mismatch", "unsupported_version":
		connectionErr.Fatal = true
	}
	return connectionErr
}

func machineConnectURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("parse control plane url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("control plane url must include scheme and host")
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported control plane url scheme %q", parsed.Scheme)
	}

	pathValue := strings.TrimRight(strings.TrimSpace(parsed.Path), "/")
	switch {
	case pathValue == "":
		parsed.Path = "/api/v1/machines/connect"
	case strings.HasSuffix(pathValue, "/api/v1/platform"):
		parsed.Path = strings.TrimSuffix(pathValue, "/api/v1/platform") + "/api/v1/machines/connect"
	case strings.HasSuffix(pathValue, "/api/v1/machines/connect"):
		parsed.Path = pathValue
	case strings.HasSuffix(pathValue, "/api/v1"):
		parsed.Path = pathValue + "/machines/connect"
	default:
		parsed.Path = pathValue + "/api/v1/machines/connect"
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func readJSONEnvelope(conn *websocket.Conn) (domain.Envelope, error) {
	_, payload, err := conn.ReadMessage()
	if err != nil {
		return domain.Envelope{}, err
	}
	return domain.ParseEnvelope(payload)
}

func writeJSONEnvelope(conn *websocket.Conn, envelope domain.Envelope) error {
	return conn.WriteJSON(envelope)
}

func mustMarshalJSON(payload any) json.RawMessage {
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return body
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
