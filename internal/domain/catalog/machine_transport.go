package catalog

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type MachineConnectionMode string

const (
	MachineConnectionModeLocal      MachineConnectionMode = "local"
	MachineConnectionModeSSH        MachineConnectionMode = "ssh"
	MachineConnectionModeWSReverse  MachineConnectionMode = "ws_reverse"
	MachineConnectionModeWSListener MachineConnectionMode = "ws_listener"
)

func (m MachineConnectionMode) String() string {
	return string(m)
}

func (m MachineConnectionMode) IsValid() bool {
	switch m {
	case MachineConnectionModeLocal,
		MachineConnectionModeSSH,
		MachineConnectionModeWSReverse,
		MachineConnectionModeWSListener:
		return true
	default:
		return false
	}
}

type MachineTransportCapability string

const (
	MachineTransportCapabilityProbe            MachineTransportCapability = "probe"
	MachineTransportCapabilityWorkspacePrepare MachineTransportCapability = "workspace_prepare"
	MachineTransportCapabilityArtifactSync     MachineTransportCapability = "artifact_sync"
	MachineTransportCapabilityProcessStreaming MachineTransportCapability = "process_streaming"
)

func (c MachineTransportCapability) String() string {
	return string(c)
}

func (c MachineTransportCapability) IsValid() bool {
	switch c {
	case MachineTransportCapabilityProbe,
		MachineTransportCapabilityWorkspacePrepare,
		MachineTransportCapabilityArtifactSync,
		MachineTransportCapabilityProcessStreaming:
		return true
	default:
		return false
	}
}

type MachineDetectedOS string

const (
	MachineDetectedOSDarwin  MachineDetectedOS = "darwin"
	MachineDetectedOSLinux   MachineDetectedOS = "linux"
	MachineDetectedOSUnknown MachineDetectedOS = "unknown"
)

func (o MachineDetectedOS) String() string {
	return string(o)
}

func (o MachineDetectedOS) IsValid() bool {
	switch o {
	case MachineDetectedOSDarwin, MachineDetectedOSLinux, MachineDetectedOSUnknown:
		return true
	default:
		return false
	}
}

type MachineDetectedArch string

const (
	MachineDetectedArchAMD64   MachineDetectedArch = "amd64"
	MachineDetectedArchARM64   MachineDetectedArch = "arm64"
	MachineDetectedArchUnknown MachineDetectedArch = "unknown"
)

func (a MachineDetectedArch) String() string {
	return string(a)
}

func (a MachineDetectedArch) IsValid() bool {
	switch a {
	case MachineDetectedArchAMD64, MachineDetectedArchARM64, MachineDetectedArchUnknown:
		return true
	default:
		return false
	}
}

type MachineDetectionStatus string

const (
	MachineDetectionStatusPending  MachineDetectionStatus = "pending"
	MachineDetectionStatusOK       MachineDetectionStatus = "ok"
	MachineDetectionStatusDegraded MachineDetectionStatus = "degraded"
	MachineDetectionStatusUnknown  MachineDetectionStatus = "unknown"
)

func (s MachineDetectionStatus) String() string {
	return string(s)
}

func (s MachineDetectionStatus) IsValid() bool {
	switch s {
	case MachineDetectionStatusPending,
		MachineDetectionStatusOK,
		MachineDetectionStatusDegraded,
		MachineDetectionStatusUnknown:
		return true
	default:
		return false
	}
}

type MachineTransportSessionState string

const (
	MachineTransportSessionStateUnknown      MachineTransportSessionState = "unknown"
	MachineTransportSessionStateConnected    MachineTransportSessionState = "connected"
	MachineTransportSessionStateDisconnected MachineTransportSessionState = "disconnected"
	MachineTransportSessionStateUnavailable  MachineTransportSessionState = "unavailable"
)

func (s MachineTransportSessionState) String() string {
	return string(s)
}

func (s MachineTransportSessionState) IsValid() bool {
	switch s {
	case MachineTransportSessionStateUnknown,
		MachineTransportSessionStateConnected,
		MachineTransportSessionStateDisconnected,
		MachineTransportSessionStateUnavailable:
		return true
	default:
		return false
	}
}

type MachineDaemonStatus struct {
	Registered       bool
	LastRegisteredAt *time.Time
	CurrentSessionID *string
	SessionState     MachineTransportSessionState
}

type MachineDaemonStatusInput struct {
	Registered       *bool   `json:"registered"`
	LastRegisteredAt *string `json:"last_registered_at"`
	CurrentSessionID *string `json:"current_session_id"`
	SessionState     string  `json:"session_state"`
}

type MachineChannelCredentialKind string

const (
	MachineChannelCredentialKindNone        MachineChannelCredentialKind = "none"
	MachineChannelCredentialKindToken       MachineChannelCredentialKind = "token"
	MachineChannelCredentialKindCertificate MachineChannelCredentialKind = "certificate"
)

func (k MachineChannelCredentialKind) String() string {
	return string(k)
}

func (k MachineChannelCredentialKind) IsValid() bool {
	switch k {
	case MachineChannelCredentialKindNone,
		MachineChannelCredentialKindToken,
		MachineChannelCredentialKindCertificate:
		return true
	default:
		return false
	}
}

type MachineChannelCredential struct {
	Kind          MachineChannelCredentialKind
	TokenID       *string
	CertificateID *string
}

type MachineChannelCredentialInput struct {
	Kind          string  `json:"kind"`
	TokenID       *string `json:"token_id"`
	CertificateID *string `json:"certificate_id"`
}

func defaultMachineTransportCapabilities(mode MachineConnectionMode) []MachineTransportCapability {
	switch mode {
	case MachineConnectionModeLocal,
		MachineConnectionModeSSH,
		MachineConnectionModeWSReverse,
		MachineConnectionModeWSListener:
		return []MachineTransportCapability{
			MachineTransportCapabilityProbe,
			MachineTransportCapabilityWorkspacePrepare,
			MachineTransportCapabilityArtifactSync,
			MachineTransportCapabilityProcessStreaming,
		}
	default:
		return nil
	}
}

func parseMachineConnectionMode(raw string, host string) (MachineConnectionMode, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		if host == LocalMachineHost {
			return MachineConnectionModeLocal, nil
		}
		return MachineConnectionModeSSH, nil
	}

	mode := MachineConnectionMode(strings.ToLower(trimmed))
	if !mode.IsValid() {
		return "", fmt.Errorf("connection_mode must be one of local, ssh, ws_reverse, ws_listener")
	}

	return mode, nil
}

func ParseStoredMachineConnectionMode(raw string, host string) (MachineConnectionMode, error) {
	return parseMachineConnectionMode(raw, host)
}

func parseMachineTransportCapabilities(
	raw []string,
	mode MachineConnectionMode,
) ([]MachineTransportCapability, error) {
	if len(raw) == 0 {
		return defaultMachineTransportCapabilities(mode), nil
	}

	items := make([]MachineTransportCapability, 0, len(raw))
	seen := make(map[MachineTransportCapability]struct{}, len(raw))
	for index, item := range raw {
		capability := MachineTransportCapability(strings.ToLower(strings.TrimSpace(item)))
		if !capability.IsValid() {
			return nil, fmt.Errorf(
				"transport_capabilities[%d] must be one of probe, workspace_prepare, artifact_sync, process_streaming",
				index,
			)
		}
		if _, ok := seen[capability]; ok {
			continue
		}
		seen[capability] = struct{}{}
		items = append(items, capability)
	}

	return items, nil
}

func ParseStoredMachineTransportCapabilities(
	raw []string,
	mode MachineConnectionMode,
) ([]MachineTransportCapability, error) {
	return parseMachineTransportCapabilities(raw, mode)
}

func parseMachineAdvertisedEndpoint(raw *string, mode MachineConnectionMode) (*string, error) {
	endpoint := parseOptionalText(raw)
	if endpoint == nil {
		if mode == MachineConnectionModeWSListener {
			return nil, fmt.Errorf("advertised_endpoint must not be empty for ws_listener machines")
		}
		return nil, nil
	}

	parsed, err := url.Parse(*endpoint)
	if err != nil {
		return nil, fmt.Errorf("advertised_endpoint must be a valid ws:// or wss:// URL")
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return nil, fmt.Errorf("advertised_endpoint must use ws or wss scheme")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return nil, fmt.Errorf("advertised_endpoint must include a host")
	}

	normalized := parsed.String()
	return &normalized, nil
}

func parseMachineDetectedOS(raw string) (MachineDetectedOS, error) {
	if strings.TrimSpace(raw) == "" {
		return MachineDetectedOSUnknown, nil
	}

	value := MachineDetectedOS(strings.ToLower(strings.TrimSpace(raw)))
	if !value.IsValid() {
		return "", fmt.Errorf("detected_os must be one of darwin, linux, unknown")
	}

	return value, nil
}

func ParseStoredMachineDetectedOS(raw string) (MachineDetectedOS, error) {
	return parseMachineDetectedOS(raw)
}

func parseMachineDetectedArch(raw string) (MachineDetectedArch, error) {
	if strings.TrimSpace(raw) == "" {
		return MachineDetectedArchUnknown, nil
	}

	value := MachineDetectedArch(strings.ToLower(strings.TrimSpace(raw)))
	if !value.IsValid() {
		return "", fmt.Errorf("detected_arch must be one of amd64, arm64, unknown")
	}

	return value, nil
}

func ParseStoredMachineDetectedArch(raw string) (MachineDetectedArch, error) {
	return parseMachineDetectedArch(raw)
}

func parseMachineDetectionStatus(raw string) (MachineDetectionStatus, error) {
	if strings.TrimSpace(raw) == "" {
		return MachineDetectionStatusUnknown, nil
	}

	value := MachineDetectionStatus(strings.ToLower(strings.TrimSpace(raw)))
	if !value.IsValid() {
		return "", fmt.Errorf("detection_status must be one of pending, ok, degraded, unknown")
	}

	return value, nil
}

func ParseStoredMachineDetectionStatus(raw string) (MachineDetectionStatus, error) {
	return parseMachineDetectionStatus(raw)
}

func parseMachineDaemonStatus(raw MachineDaemonStatusInput) (MachineDaemonStatus, error) {
	registered := false
	if raw.Registered != nil {
		registered = *raw.Registered
	}

	lastRegisteredAt, err := parseOptionalRFC3339("daemon_status.last_registered_at", raw.LastRegisteredAt)
	if err != nil {
		return MachineDaemonStatus{}, err
	}

	sessionState := MachineTransportSessionStateUnknown
	if strings.TrimSpace(raw.SessionState) != "" {
		sessionState = MachineTransportSessionState(strings.ToLower(strings.TrimSpace(raw.SessionState)))
		if !sessionState.IsValid() {
			return MachineDaemonStatus{}, fmt.Errorf("daemon_status.session_state must be one of unknown, connected, disconnected, unavailable")
		}
	}

	return MachineDaemonStatus{
		Registered:       registered,
		LastRegisteredAt: lastRegisteredAt,
		CurrentSessionID: parseOptionalText(raw.CurrentSessionID),
		SessionState:     sessionState,
	}, nil
}

func parseMachineChannelCredential(raw *MachineChannelCredentialInput) (MachineChannelCredential, error) {
	if raw == nil {
		return MachineChannelCredential{Kind: MachineChannelCredentialKindNone}, nil
	}

	kind := MachineChannelCredentialKind(strings.ToLower(strings.TrimSpace(raw.Kind)))
	if kind == "" {
		kind = MachineChannelCredentialKindNone
	}
	if !kind.IsValid() {
		return MachineChannelCredential{}, fmt.Errorf("channel_credential.kind must be one of none, token, certificate")
	}

	tokenID := parseOptionalText(raw.TokenID)
	certificateID := parseOptionalText(raw.CertificateID)
	switch kind {
	case MachineChannelCredentialKindNone:
		tokenID = nil
		certificateID = nil
	case MachineChannelCredentialKindToken:
		if tokenID == nil {
			return MachineChannelCredential{}, fmt.Errorf("channel_credential.token_id must not be empty for token credentials")
		}
		certificateID = nil
	case MachineChannelCredentialKindCertificate:
		if certificateID == nil {
			return MachineChannelCredential{}, fmt.Errorf("channel_credential.certificate_id must not be empty for certificate credentials")
		}
		tokenID = nil
	}

	return MachineChannelCredential{
		Kind:          kind,
		TokenID:       tokenID,
		CertificateID: certificateID,
	}, nil
}

func ParseStoredMachineChannelCredentialKind(raw string) (MachineChannelCredentialKind, error) {
	kind := MachineChannelCredentialKind(strings.ToLower(strings.TrimSpace(raw)))
	if kind == "" {
		return MachineChannelCredentialKindNone, nil
	}
	if !kind.IsValid() {
		return "", fmt.Errorf("channel credential kind must be one of none, token, certificate")
	}
	return kind, nil
}

func ParseStoredMachineSessionState(raw string) (MachineTransportSessionState, error) {
	state := MachineTransportSessionState(strings.ToLower(strings.TrimSpace(raw)))
	if state == "" {
		return MachineTransportSessionStateUnknown, nil
	}
	if !state.IsValid() {
		return "", fmt.Errorf("machine session state must be one of unknown, connected, disconnected, unavailable")
	}
	return state, nil
}

func parseOptionalRFC3339(fieldName string, raw *string) (*time.Time, error) {
	trimmed := parseOptionalText(raw)
	if trimmed == nil {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, *trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid RFC3339 timestamp", fieldName)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func cloneMachineDaemonStatus(status MachineDaemonStatus) MachineDaemonStatus {
	return MachineDaemonStatus{
		Registered:       status.Registered,
		LastRegisteredAt: cloneTimePointer(status.LastRegisteredAt),
		CurrentSessionID: cloneStringPointer(status.CurrentSessionID),
		SessionState:     status.SessionState,
	}
}

func cloneMachineChannelCredential(credential MachineChannelCredential) MachineChannelCredential {
	return MachineChannelCredential{
		Kind:          credential.Kind,
		TokenID:       cloneStringPointer(credential.TokenID),
		CertificateID: cloneStringPointer(credential.CertificateID),
	}
}
