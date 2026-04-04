package machinechannel

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	TokenPrefix = "ase_machine_"

	EnvMachineID                = "OPENASE_MACHINE_ID"
	EnvMachineChannelToken      = "OPENASE_MACHINE_CHANNEL_TOKEN" // #nosec G101 -- environment variable key name, not a credential
	EnvMachineControlPlaneURL   = "OPENASE_MACHINE_CONTROL_PLANE_URL"
	EnvMachineHeartbeatInterval = "OPENASE_MACHINE_HEARTBEAT_INTERVAL"
)

var (
	ErrNotFound             = errors.New("machine channel record not found")
	ErrInvalidToken         = errors.New("machine channel token is invalid")
	ErrTokenExpired         = errors.New("machine channel token expired")
	ErrTokenRevoked         = errors.New("machine channel token is revoked")
	ErrUnexpectedMessage    = errors.New("machine channel message is unexpected")
	ErrUnsupportedVersion   = errors.New("machine channel version is unsupported")
	ErrConnectionMode       = errors.New("machine connection mode does not support reverse websocket registration")
	ErrMachineDisabled      = errors.New("machine is disabled for reverse websocket registration")
	ErrSessionReplaced      = errors.New("machine channel session was replaced")
	ErrHeartbeatTimedOut    = errors.New("machine channel heartbeat timed out")
	ErrAuthenticationFailed = errors.New("machine channel authentication failed")
)

type IssueInput struct {
	MachineID uuid.UUID
	TTL       time.Duration
}

type IssuedToken struct {
	Token     string
	TokenID   uuid.UUID
	MachineID uuid.UUID
	ExpiresAt time.Time
}

type Claims struct {
	TokenID   uuid.UUID
	MachineID uuid.UUID
	ExpiresAt time.Time
}

type DaemonConfig struct {
	MachineID         uuid.UUID
	Token             string
	ControlPlaneURL   string
	HeartbeatInterval time.Duration
	ReconnectBackoff  time.Duration
	OpenASEBinaryPath string
	AgentCLIPath      string
}

func ParseDaemonConfig(
	machineID string,
	token string,
	controlPlaneURL string,
	heartbeatInterval time.Duration,
	reconnectBackoff time.Duration,
	openaseBinaryPath string,
	agentCLIPath string,
) (DaemonConfig, error) {
	parsedMachineID, err := parseUUID(machineID)
	if err != nil {
		return DaemonConfig{}, err
	}
	parsedToken, err := ParseToken(token)
	if err != nil {
		return DaemonConfig{}, err
	}
	url := strings.TrimRight(strings.TrimSpace(controlPlaneURL), "/")
	if url == "" {
		return DaemonConfig{}, fmt.Errorf("control_plane_url must not be empty")
	}
	if heartbeatInterval <= 0 {
		return DaemonConfig{}, fmt.Errorf("heartbeat_interval must be greater than zero")
	}
	if reconnectBackoff <= 0 {
		return DaemonConfig{}, fmt.Errorf("reconnect_backoff must be greater than zero")
	}
	return DaemonConfig{
		MachineID:         parsedMachineID,
		Token:             parsedToken,
		ControlPlaneURL:   url,
		HeartbeatInterval: heartbeatInterval,
		ReconnectBackoff:  reconnectBackoff,
		OpenASEBinaryPath: strings.TrimSpace(openaseBinaryPath),
		AgentCLIPath:      strings.TrimSpace(agentCLIPath),
	}, nil
}

type MessageType string

const (
	MessageTypeHello         MessageType = "hello"
	MessageTypeAuthenticate  MessageType = "authenticate"
	MessageTypeRegistered    MessageType = "registered"
	MessageTypeHeartbeat     MessageType = "heartbeat"
	MessageTypeReconnect     MessageType = "reconnect"
	MessageTypeGoodbye       MessageType = "goodbye"
	MessageTypeCapabilities  MessageType = "capabilities"
	MessageTypeSystemInfo    MessageType = "system_info"
	MessageTypeToolInventory MessageType = "tool_inventory"
	MessageTypeError         MessageType = "error"
	MessageTypeRetryAfter    MessageType = "retry_after"
)

const ProtocolVersion = 1

type Envelope struct {
	Version   int             `json:"version"`
	Type      MessageType     `json:"type"`
	SessionID string          `json:"session_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type Hello struct {
	AgentVersion string `json:"agent_version"`
	Hostname     string `json:"hostname"`
}

type Authenticate struct {
	Token            string            `json:"token"`
	MachineID        string            `json:"machine_id"`
	TransportMode    string            `json:"transport_mode"`
	SystemInfo       SystemInfo        `json:"system_info"`
	Capabilities     []string          `json:"capabilities,omitempty"`
	ToolInventory    []ToolInfo        `json:"tool_inventory,omitempty"`
	ResourceSnapshot *ResourceSnapshot `json:"resource_snapshot,omitempty"`
}

type Registered struct {
	MachineID                string `json:"machine_id"`
	SessionID                string `json:"session_id"`
	HeartbeatIntervalSeconds int    `json:"heartbeat_interval_seconds"`
	HeartbeatTimeoutSeconds  int    `json:"heartbeat_timeout_seconds"`
	ReplacedPreviousSession  bool   `json:"replaced_previous_session"`
}

type Heartbeat struct {
	SentAt           string            `json:"sent_at"`
	SystemInfo       *SystemInfo       `json:"system_info,omitempty"`
	ToolInventory    []ToolInfo        `json:"tool_inventory,omitempty"`
	ResourceSnapshot *ResourceSnapshot `json:"resource_snapshot,omitempty"`
}

type Goodbye struct {
	Reason string `json:"reason"`
}

type ErrorPayload struct {
	Code              string `json:"code"`
	Message           string `json:"message"`
	RetryAfterSeconds int    `json:"retry_after_seconds,omitempty"`
}

type SystemInfo struct {
	Hostname          string `json:"hostname"`
	OS                string `json:"os"`
	Arch              string `json:"arch"`
	OpenASEBinaryPath string `json:"openase_binary_path,omitempty"`
	AgentCLIPath      string `json:"agent_cli_path,omitempty"`
}

type ToolInfo struct {
	Name       string `json:"name"`
	Installed  bool   `json:"installed"`
	Version    string `json:"version,omitempty"`
	AuthStatus string `json:"auth_status,omitempty"`
	AuthMode   string `json:"auth_mode,omitempty"`
	Ready      bool   `json:"ready"`
}

type GPUInfo struct {
	Index              int     `json:"index"`
	Name               string  `json:"name"`
	MemoryTotalGB      float64 `json:"memory_total_gb"`
	MemoryUsedGB       float64 `json:"memory_used_gb"`
	UtilizationPercent float64 `json:"utilization_percent"`
}

type GitAudit struct {
	Installed bool   `json:"installed"`
	UserName  string `json:"user_name,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
}

type GitHubCLIAudit struct {
	Installed  bool   `json:"installed"`
	AuthStatus string `json:"auth_status,omitempty"`
}

type GitHubTokenProbe struct {
	State       string   `json:"state,omitempty"`
	Configured  bool     `json:"configured"`
	Valid       bool     `json:"valid"`
	Permissions []string `json:"permissions,omitempty"`
	RepoAccess  string   `json:"repo_access,omitempty"`
	LastError   string   `json:"last_error,omitempty"`
}

type NetworkAudit struct {
	GitHubReachable bool `json:"github_reachable"`
	PyPIReachable   bool `json:"pypi_reachable"`
	NPMReachable    bool `json:"npm_reachable"`
}

type FullAudit struct {
	Git              GitAudit         `json:"git"`
	GitHubCLI        GitHubCLIAudit   `json:"gh_cli"`
	GitHubTokenProbe GitHubTokenProbe `json:"github_token_probe"`
	Network          NetworkAudit     `json:"network"`
}

type ResourceSnapshot struct {
	CollectedAt       string     `json:"collected_at"`
	CPUCores          int        `json:"cpu_cores"`
	CPUUsagePercent   float64    `json:"cpu_usage_percent"`
	MemoryTotalGB     float64    `json:"memory_total_gb"`
	MemoryUsedGB      float64    `json:"memory_used_gb"`
	MemoryAvailableGB float64    `json:"memory_available_gb"`
	DiskTotalGB       float64    `json:"disk_total_gb"`
	DiskAvailableGB   float64    `json:"disk_available_gb"`
	GPUs              []GPUInfo  `json:"gpus,omitempty"`
	FullAudit         *FullAudit `json:"full_audit,omitempty"`
}

func ParseToken(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.HasPrefix(trimmed, TokenPrefix) {
		return "", ErrInvalidToken
	}
	return trimmed, nil
}

func ParseEnvelope(raw []byte) (Envelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Envelope{}, fmt.Errorf("parse machine channel envelope: %w", err)
	}
	if envelope.Version != ProtocolVersion {
		return Envelope{}, fmt.Errorf("%w: expected %d got %d", ErrUnsupportedVersion, ProtocolVersion, envelope.Version)
	}
	switch envelope.Type {
	case MessageTypeHello,
		MessageTypeAuthenticate,
		MessageTypeRegistered,
		MessageTypeHeartbeat,
		MessageTypeReconnect,
		MessageTypeGoodbye,
		MessageTypeCapabilities,
		MessageTypeSystemInfo,
		MessageTypeToolInventory,
		MessageTypeError,
		MessageTypeRetryAfter:
	default:
		return Envelope{}, fmt.Errorf("%w: unsupported message type %q", ErrUnexpectedMessage, strings.TrimSpace(string(envelope.Type)))
	}
	return envelope, nil
}

func DecodePayload[T any](envelope Envelope) (T, error) {
	var payload T
	if len(envelope.Payload) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return payload, fmt.Errorf("decode %s payload: %w", envelope.Type, err)
	}
	return payload, nil
}

func parseUUID(raw string) (uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return uuid.UUID{}, fmt.Errorf("machine_id must not be empty")
	}
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("machine_id must be a valid UUID")
	}
	return parsed, nil
}
