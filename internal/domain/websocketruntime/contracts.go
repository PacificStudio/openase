package websocketruntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const ProtocolVersion = 1

var (
	ErrUnexpectedMessage  = errors.New("websocket runtime message is unexpected")
	ErrUnsupportedVersion = errors.New("websocket runtime version is unsupported")
)

type MessageType string

const (
	MessageTypeHello    MessageType = "hello"
	MessageTypeHelloAck MessageType = "hello_ack"
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeEvent    MessageType = "event"
)

func (t MessageType) IsValid() bool {
	switch t {
	case MessageTypeHello, MessageTypeHelloAck, MessageTypeRequest, MessageTypeResponse, MessageTypeEvent:
		return true
	default:
		return false
	}
}

type Operation string

const (
	OperationProbe            Operation = "probe"
	OperationPreflight        Operation = "preflight"
	OperationWorkspacePrepare Operation = "workspace_prepare"
	OperationWorkspaceReset   Operation = "workspace_reset"
	OperationArtifactSync     Operation = "artifact_sync"
	OperationCommandOpen      Operation = "command_open"
	OperationSessionInput     Operation = "session_input"
	OperationSessionSignal    Operation = "session_signal"
	OperationSessionClose     Operation = "session_close"
	OperationProcessStart     Operation = "process_start"
	OperationProcessStatus    Operation = "process_status"
	OperationSessionOutput    Operation = "session_output"
	OperationSessionExit      Operation = "session_exit"
)

func (o Operation) IsValid() bool {
	switch o {
	case OperationProbe,
		OperationPreflight,
		OperationWorkspacePrepare,
		OperationWorkspaceReset,
		OperationArtifactSync,
		OperationCommandOpen,
		OperationSessionInput,
		OperationSessionSignal,
		OperationSessionClose,
		OperationProcessStart,
		OperationProcessStatus,
		OperationSessionOutput,
		OperationSessionExit:
		return true
	default:
		return false
	}
}

type ErrorClass string

const (
	ErrorClassAuth             ErrorClass = "auth"
	ErrorClassMisconfiguration ErrorClass = "misconfiguration"
	ErrorClassTransient        ErrorClass = "transient"
	ErrorClassUnsupported      ErrorClass = "unsupported"
	ErrorClassInternal         ErrorClass = "internal"
)

func (c ErrorClass) IsValid() bool {
	switch c {
	case ErrorClassAuth,
		ErrorClassMisconfiguration,
		ErrorClassTransient,
		ErrorClassUnsupported,
		ErrorClassInternal:
		return true
	default:
		return false
	}
}

type ErrorCode string

const (
	ErrorCodeInvalidRequest       ErrorCode = "invalid_request"
	ErrorCodeProtocolVersion      ErrorCode = "protocol_version"
	ErrorCodeWorkspace            ErrorCode = "workspace"
	ErrorCodeArtifactSync         ErrorCode = "artifact_sync"
	ErrorCodePreflight            ErrorCode = "preflight"
	ErrorCodeSessionNotFound      ErrorCode = "session_not_found"
	ErrorCodeProcessStart         ErrorCode = "process_start"
	ErrorCodeProcessSignal        ErrorCode = "process_signal"
	ErrorCodeTransportUnavailable ErrorCode = "transport_unavailable"
	ErrorCodeUnauthorized         ErrorCode = "unauthorized"
	ErrorCodeUnsupported          ErrorCode = "unsupported"
	ErrorCodeInternal             ErrorCode = "internal"
)

type ErrorPayload struct {
	Code      ErrorCode      `json:"code"`
	Class     ErrorClass     `json:"class"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

type Envelope struct {
	Version   int             `json:"version"`
	Type      MessageType     `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Operation Operation       `json:"operation,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Error     *ErrorPayload   `json:"error,omitempty"`
}

func ParseEnvelope(raw []byte) (Envelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Envelope{}, fmt.Errorf("parse websocket runtime envelope: %w", err)
	}
	if envelope.Version != ProtocolVersion {
		return Envelope{}, fmt.Errorf("%w: expected %d got %d", ErrUnsupportedVersion, ProtocolVersion, envelope.Version)
	}
	if !envelope.Type.IsValid() {
		return Envelope{}, fmt.Errorf("%w: %s", ErrUnexpectedMessage, envelope.Type)
	}
	if envelope.Operation != "" && !envelope.Operation.IsValid() {
		return Envelope{}, fmt.Errorf("%w: %s", ErrUnexpectedMessage, envelope.Operation)
	}
	if envelope.Error != nil {
		if !envelope.Error.Class.IsValid() {
			return Envelope{}, fmt.Errorf("%w: invalid error class %q", ErrUnexpectedMessage, envelope.Error.Class)
		}
		envelope.Error.Message = strings.TrimSpace(envelope.Error.Message)
	}
	envelope.RequestID = strings.TrimSpace(envelope.RequestID)
	return envelope, nil
}

func DecodePayload[T any](envelope Envelope) (T, error) {
	var zero T
	if len(envelope.Payload) == 0 {
		return zero, nil
	}
	var payload T
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return zero, fmt.Errorf("decode websocket runtime payload for %s: %w", envelope.Operation, err)
	}
	return payload, nil
}

type Hello struct {
	SupportedVersions []int       `json:"supported_versions,omitempty"`
	Capabilities      []Operation `json:"capabilities,omitempty"`
}

type HelloAck struct {
	SelectedVersion int         `json:"selected_version"`
	Capabilities    []Operation `json:"capabilities,omitempty"`
}

type ProbeResponse struct {
	CheckedAt       string         `json:"checked_at"`
	Output          string         `json:"output"`
	Resources       map[string]any `json:"resources,omitempty"`
	DetectedOS      string         `json:"detected_os,omitempty"`
	DetectedArch    string         `json:"detected_arch,omitempty"`
	DetectionStatus string         `json:"detection_status,omitempty"`
}

type PreflightRequest struct {
	WorkingDirectory string   `json:"working_directory"`
	AgentCommand     string   `json:"agent_command"`
	Environment      []string `json:"environment,omitempty"`
}

type WorkspaceRepoAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type WorkspaceRepo struct {
	Name             string             `json:"name"`
	RepositoryURL    string             `json:"repository_url"`
	DefaultBranch    string             `json:"default_branch,omitempty"`
	WorkspaceDirname *string            `json:"workspace_dirname,omitempty"`
	BranchName       *string            `json:"branch_name,omitempty"`
	HTTPBasicAuth    *WorkspaceRepoAuth `json:"http_basic_auth,omitempty"`
}

type WorkspacePrepareRequest struct {
	WorkspaceRoot    string          `json:"workspace_root"`
	OrganizationSlug string          `json:"organization_slug"`
	ProjectSlug      string          `json:"project_slug"`
	AgentName        string          `json:"agent_name,omitempty"`
	TicketIdentifier string          `json:"ticket_identifier"`
	MachineID        string          `json:"machine_id,omitempty"`
	RunID            string          `json:"run_id,omitempty"`
	TicketID         string          `json:"ticket_id,omitempty"`
	Repos            []WorkspaceRepo `json:"repos,omitempty"`
}

type PreparedRepo struct {
	Name             string `json:"name"`
	RepositoryURL    string `json:"repository_url"`
	DefaultBranch    string `json:"default_branch"`
	BranchName       string `json:"branch_name"`
	WorkspaceDirname string `json:"workspace_dirname"`
	HeadCommit       string `json:"head_commit"`
	Path             string `json:"path"`
}

type WorkspacePrepareResponse struct {
	Path       string         `json:"path"`
	BranchName string         `json:"branch_name"`
	Repos      []PreparedRepo `json:"repos,omitempty"`
}

type WorkspaceResetRequest struct {
	Path string `json:"path"`
}

type ArtifactEntryKind string

const (
	ArtifactEntryKindFile ArtifactEntryKind = "file"
	ArtifactEntryKindDir  ArtifactEntryKind = "dir"
)

type ArtifactEntry struct {
	Path          string            `json:"path"`
	Kind          ArtifactEntryKind `json:"kind"`
	Mode          int64             `json:"mode,omitempty"`
	ContentBase64 string            `json:"content_base64,omitempty"`
}

type ArtifactSyncRequest struct {
	TargetRoot  string          `json:"target_root"`
	RemovePaths []string        `json:"remove_paths,omitempty"`
	Entries     []ArtifactEntry `json:"entries,omitempty"`
}

type CommandOpenRequest struct {
	Command string `json:"command"`
}

type ProcessStartRequest struct {
	Command          string   `json:"command"`
	Args             []string `json:"args,omitempty"`
	WorkingDirectory string   `json:"working_directory,omitempty"`
	Environment      []string `json:"environment,omitempty"`
}

type SessionResponse struct {
	SessionID string `json:"session_id"`
}

type SessionInputRequest struct {
	SessionID  string `json:"session_id"`
	DataBase64 string `json:"data_base64,omitempty"`
	CloseStdin bool   `json:"close_stdin,omitempty"`
}

type SessionSignalRequest struct {
	SessionID string `json:"session_id"`
	Signal    string `json:"signal,omitempty"`
}

type SessionCloseRequest struct {
	SessionID string `json:"session_id"`
}

type ProcessStatusRequest struct {
	SessionID string `json:"session_id"`
}

type ProcessStatusResponse struct {
	SessionID string `json:"session_id"`
	Running   bool   `json:"running"`
	ExitCode  *int   `json:"exit_code,omitempty"`
}

type SessionOutputEvent struct {
	SessionID  string `json:"session_id"`
	Stream     string `json:"stream"`
	DataBase64 string `json:"data_base64,omitempty"`
}

type SessionExitEvent struct {
	SessionID string `json:"session_id"`
	ExitCode  int    `json:"exit_code"`
}
