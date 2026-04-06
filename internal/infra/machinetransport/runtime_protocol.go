package machinetransport

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
	"github.com/BetterAndBetterII/openase/internal/infra/machineprobe"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

var _ = logging.DeclareComponent("machine-transport-runtime-protocol")

type runtimeEnvelopeSender func(context.Context, runtimecontract.Envelope) error

type runtimeProtocolError struct {
	payload runtimecontract.ErrorPayload
}

func (e runtimeProtocolError) Error() string {
	return strings.TrimSpace(e.payload.Message)
}

func (e runtimeProtocolError) Payload() runtimecontract.ErrorPayload {
	return e.payload
}

func marshalRuntimePayload(value any) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

type runtimeProtocolClient struct {
	send runtimeEnvelopeSender

	requestMu sync.Mutex
	requests  map[string]chan runtimecontract.Envelope

	sessionMu sync.Mutex
	sessions  map[string]*runtimeManagedClientSession
	pending   map[string][]runtimecontract.Envelope

	closeOnce sync.Once
	closed    chan struct{}
	closeErr  error

	helloMu   sync.Mutex
	helloDone bool
	helloErr  error
}

func newRuntimeProtocolClient(send runtimeEnvelopeSender) *runtimeProtocolClient {
	return &runtimeProtocolClient{
		send:     send,
		requests: map[string]chan runtimecontract.Envelope{},
		sessions: map[string]*runtimeManagedClientSession{},
		pending:  map[string][]runtimecontract.Envelope{},
		closed:   make(chan struct{}),
	}
}

func (c *runtimeProtocolClient) Close(err error) {
	c.closeOnce.Do(func() {
		if err == nil {
			err = io.EOF
		}
		c.closeErr = err
		close(c.closed)

		c.requestMu.Lock()
		requests := c.requests
		c.requests = map[string]chan runtimecontract.Envelope{}
		c.requestMu.Unlock()
		for _, ch := range requests {
			close(ch)
		}

		c.sessionMu.Lock()
		sessions := c.sessions
		c.sessions = map[string]*runtimeManagedClientSession{}
		c.pending = map[string][]runtimecontract.Envelope{}
		c.sessionMu.Unlock()
		for _, session := range sessions {
			session.finish(err)
		}
	})
}

func (c *runtimeProtocolClient) HandleEnvelope(envelope runtimecontract.Envelope) error {
	switch envelope.Type {
	case runtimecontract.MessageTypeHelloAck, runtimecontract.MessageTypeResponse:
		c.requestMu.Lock()
		ch := c.requests[envelope.RequestID]
		c.requestMu.Unlock()
		if ch == nil {
			return nil
		}
		select {
		case ch <- envelope:
		default:
		}
		return nil
	case runtimecontract.MessageTypeEvent:
		switch envelope.Operation {
		case runtimecontract.OperationSessionOutput:
			payload, err := runtimecontract.DecodePayload[runtimecontract.SessionOutputEvent](envelope)
			if err != nil {
				return err
			}
			return c.handleSessionEvent(payload.SessionID, envelope)
		case runtimecontract.OperationSessionExit:
			payload, err := runtimecontract.DecodePayload[runtimecontract.SessionExitEvent](envelope)
			if err != nil {
				return err
			}
			return c.handleSessionEvent(payload.SessionID, envelope)
		default:
			return fmt.Errorf("runtime client event %q is not supported", envelope.Operation)
		}
	default:
		return fmt.Errorf("runtime client message type %q is not supported", envelope.Type)
	}
}

func (c *runtimeProtocolClient) ensureHello(ctx context.Context) error {
	c.helloMu.Lock()
	if c.helloDone {
		err := c.helloErr
		c.helloMu.Unlock()
		return err
	}
	c.helloMu.Unlock()

	envelope, err := c.sendRequest(ctx, runtimecontract.MessageTypeHello, "", runtimecontract.Hello{
		SupportedVersions: []int{runtimecontract.ProtocolVersion},
		Capabilities: []runtimecontract.Operation{
			runtimecontract.OperationProbe,
			runtimecontract.OperationPreflight,
			runtimecontract.OperationWorkspacePrepare,
			runtimecontract.OperationWorkspaceReset,
			runtimecontract.OperationArtifactSync,
			runtimecontract.OperationCommandOpen,
			runtimecontract.OperationSessionInput,
			runtimecontract.OperationSessionSignal,
			runtimecontract.OperationSessionClose,
			runtimecontract.OperationProcessStart,
			runtimecontract.OperationProcessStatus,
		},
	})
	if err != nil {
		c.helloMu.Lock()
		c.helloDone = true
		c.helloErr = err
		c.helloMu.Unlock()
		return err
	}
	if envelope.Type != runtimecontract.MessageTypeHelloAck {
		err = fmt.Errorf("expected hello_ack, got %s", envelope.Type)
		c.helloMu.Lock()
		c.helloDone = true
		c.helloErr = err
		c.helloMu.Unlock()
		return err
	}

	c.helloMu.Lock()
	c.helloDone = true
	c.helloErr = nil
	c.helloMu.Unlock()
	return nil
}

func (c *runtimeProtocolClient) sendRequest(
	ctx context.Context,
	messageType runtimecontract.MessageType,
	operation runtimecontract.Operation,
	payload any,
) (runtimecontract.Envelope, error) {
	if ctx == nil {
		return runtimecontract.Envelope{}, fmt.Errorf("context must not be nil")
	}
	if c == nil || c.send == nil {
		return runtimecontract.Envelope{}, fmt.Errorf("runtime protocol client unavailable")
	}
	requestID := uuid.NewString()
	rawPayload, err := marshalRuntimePayload(payload)
	if err != nil {
		return runtimecontract.Envelope{}, err
	}

	responseCh := make(chan runtimecontract.Envelope, 1)
	c.requestMu.Lock()
	c.requests[requestID] = responseCh
	c.requestMu.Unlock()
	defer func() {
		c.requestMu.Lock()
		delete(c.requests, requestID)
		c.requestMu.Unlock()
	}()

	if err := c.send(ctx, runtimecontract.Envelope{
		Version:   runtimecontract.ProtocolVersion,
		Type:      messageType,
		RequestID: requestID,
		Operation: operation,
		Payload:   rawPayload,
	}); err != nil {
		return runtimecontract.Envelope{}, err
	}

	select {
	case <-ctx.Done():
		return runtimecontract.Envelope{}, ctx.Err()
	case <-c.closed:
		if c.closeErr != nil {
			return runtimecontract.Envelope{}, c.closeErr
		}
		return runtimecontract.Envelope{}, io.EOF
	case envelope, ok := <-responseCh:
		if !ok {
			if c.closeErr != nil {
				return runtimecontract.Envelope{}, c.closeErr
			}
			return runtimecontract.Envelope{}, io.EOF
		}
		if envelope.Error != nil {
			return runtimecontract.Envelope{}, runtimeProtocolError{payload: *envelope.Error}
		}
		return envelope, nil
	}
}

func (c *runtimeProtocolClient) request(ctx context.Context, operation runtimecontract.Operation, payload any) (runtimecontract.Envelope, error) {
	if err := c.ensureHello(ctx); err != nil {
		return runtimecontract.Envelope{}, err
	}
	return c.sendRequest(ctx, runtimecontract.MessageTypeRequest, operation, payload)
}

func (c *runtimeProtocolClient) registerSession(sessionID string, session *runtimeManagedClientSession) {
	c.sessionMu.Lock()
	pending := append([]runtimecontract.Envelope(nil), c.pending[sessionID]...)
	delete(c.pending, sessionID)
	c.sessions[sessionID] = session
	c.sessionMu.Unlock()
	for _, envelope := range pending {
		_ = c.dispatchSessionEvent(session, envelope)
	}
}

func (c *runtimeProtocolClient) handleSessionEvent(sessionID string, envelope runtimecontract.Envelope) error {
	c.sessionMu.Lock()
	session := c.sessions[sessionID]
	if session == nil {
		c.pending[sessionID] = append(c.pending[sessionID], envelope)
		c.sessionMu.Unlock()
		return nil
	}
	c.sessionMu.Unlock()
	return c.dispatchSessionEvent(session, envelope)
}

func (c *runtimeProtocolClient) dispatchSessionEvent(session *runtimeManagedClientSession, envelope runtimecontract.Envelope) error {
	if session == nil {
		return nil
	}
	switch envelope.Operation {
	case runtimecontract.OperationSessionOutput:
		payload, err := runtimecontract.DecodePayload[runtimecontract.SessionOutputEvent](envelope)
		if err != nil {
			return err
		}
		return session.handleOutput(payload)
	case runtimecontract.OperationSessionExit:
		payload, err := runtimecontract.DecodePayload[runtimecontract.SessionExitEvent](envelope)
		if err != nil {
			return err
		}
		c.sessionMu.Lock()
		delete(c.sessions, payload.SessionID)
		delete(c.pending, payload.SessionID)
		c.sessionMu.Unlock()
		session.handleExit(payload.ExitCode)
		return nil
	default:
		return fmt.Errorf("runtime client event %q is not supported", envelope.Operation)
	}
}

type runtimeProtocolServer struct {
	send runtimeEnvelopeSender

	sessionMu sync.Mutex
	sessions  map[string]*runtimeProcessSession
}

type DaemonRuntimeProtocolServer struct {
	server *runtimeProtocolServer
}

func NewDaemonRuntimeProtocolServer(send func(context.Context, runtimecontract.Envelope) error) *DaemonRuntimeProtocolServer {
	return &DaemonRuntimeProtocolServer{server: newRuntimeProtocolServer(send)}
}

func (s *DaemonRuntimeProtocolServer) HandleEnvelope(ctx context.Context, envelope runtimecontract.Envelope) error {
	if s == nil || s.server == nil {
		return fmt.Errorf("runtime protocol server unavailable")
	}
	return s.server.HandleEnvelope(ctx, envelope)
}

func (s *DaemonRuntimeProtocolServer) Close() {
	if s == nil || s.server == nil {
		return
	}
	s.server.Close()
}

func newRuntimeProtocolServer(send runtimeEnvelopeSender) *runtimeProtocolServer {
	return &runtimeProtocolServer{
		send:     send,
		sessions: map[string]*runtimeProcessSession{},
	}
}

func (s *runtimeProtocolServer) Close() {
	s.sessionMu.Lock()
	sessions := s.sessions
	s.sessions = map[string]*runtimeProcessSession{}
	s.sessionMu.Unlock()
	for _, session := range sessions {
		session.stop()
	}
}

func (s *runtimeProtocolServer) HandleEnvelope(ctx context.Context, envelope runtimecontract.Envelope) error {
	switch envelope.Type {
	case runtimecontract.MessageTypeHello:
		return s.handleHello(ctx, envelope)
	case runtimecontract.MessageTypeRequest:
		return s.handleRequest(ctx, envelope)
	default:
		return s.sendError(ctx, envelope, runtimecontract.ErrorPayload{
			Code:      runtimecontract.ErrorCodeInvalidRequest,
			Class:     runtimecontract.ErrorClassUnsupported,
			Message:   fmt.Sprintf("runtime message type %q is not supported", envelope.Type),
			Retryable: false,
		})
	}
}

func (s *runtimeProtocolServer) handleHello(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.Hello](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimecontract.ErrorPayload{
			Code:      runtimecontract.ErrorCodeProtocolVersion,
			Class:     runtimecontract.ErrorClassUnsupported,
			Message:   err.Error(),
			Retryable: false,
		})
	}
	supported := false
	for _, version := range payload.SupportedVersions {
		if version == runtimecontract.ProtocolVersion {
			supported = true
			break
		}
	}
	if !supported {
		return s.sendError(ctx, envelope, runtimecontract.ErrorPayload{
			Code:      runtimecontract.ErrorCodeProtocolVersion,
			Class:     runtimecontract.ErrorClassUnsupported,
			Message:   fmt.Sprintf("runtime protocol version %d is not supported", runtimecontract.ProtocolVersion),
			Retryable: false,
			Details: map[string]any{
				"selected_version": runtimecontract.ProtocolVersion,
			},
		})
	}
	return s.send(ctx, runtimecontract.Envelope{
		Version:   runtimecontract.ProtocolVersion,
		Type:      runtimecontract.MessageTypeHelloAck,
		RequestID: envelope.RequestID,
		Payload: mustRuntimePayload(runtimecontract.HelloAck{
			SelectedVersion: runtimecontract.ProtocolVersion,
			Capabilities: []runtimecontract.Operation{
				runtimecontract.OperationProbe,
				runtimecontract.OperationPreflight,
				runtimecontract.OperationWorkspacePrepare,
				runtimecontract.OperationWorkspaceReset,
				runtimecontract.OperationArtifactSync,
				runtimecontract.OperationCommandOpen,
				runtimecontract.OperationSessionInput,
				runtimecontract.OperationSessionSignal,
				runtimecontract.OperationSessionClose,
				runtimecontract.OperationProcessStart,
				runtimecontract.OperationProcessStatus,
			},
		}),
	})
}

func (s *runtimeProtocolServer) handleRequest(ctx context.Context, envelope runtimecontract.Envelope) error {
	switch envelope.Operation {
	case runtimecontract.OperationProbe:
		return s.handleProbe(ctx, envelope)
	case runtimecontract.OperationPreflight:
		return s.handlePreflight(ctx, envelope)
	case runtimecontract.OperationWorkspacePrepare:
		return s.handleWorkspacePrepare(ctx, envelope)
	case runtimecontract.OperationWorkspaceReset:
		return s.handleWorkspaceReset(ctx, envelope)
	case runtimecontract.OperationArtifactSync:
		return s.handleArtifactSync(ctx, envelope)
	case runtimecontract.OperationCommandOpen:
		return s.handleCommandOpen(ctx, envelope)
	case runtimecontract.OperationProcessStart:
		return s.handleProcessStart(ctx, envelope)
	case runtimecontract.OperationSessionInput:
		return s.handleSessionInput(ctx, envelope)
	case runtimecontract.OperationSessionSignal:
		return s.handleSessionSignal(ctx, envelope)
	case runtimecontract.OperationSessionClose:
		return s.handleSessionClose(ctx, envelope)
	case runtimecontract.OperationProcessStatus:
		return s.handleProcessStatus(ctx, envelope)
	default:
		return s.sendError(ctx, envelope, runtimecontract.ErrorPayload{
			Code:      runtimecontract.ErrorCodeUnsupported,
			Class:     runtimecontract.ErrorClassUnsupported,
			Message:   fmt.Sprintf("runtime operation %q is not supported", envelope.Operation),
			Retryable: false,
		})
	}
}

func (s *runtimeProtocolServer) handleProbe(ctx context.Context, envelope runtimecontract.Envelope) error {
	output, err := runRuntimeShellCommand(ctx, `sh -lc 'whoami && hostname && uname -srm'`, nil, "")
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInternal, runtimecontract.ErrorClassTransient, err, true, nil))
	}
	now := time.Now().UTC()
	detectedOS, detectedArch, detectionStatus := machineprobe.DetectPlatformFromProbeOutput(output)
	return s.sendResponse(ctx, envelope, runtimecontract.ProbeResponse{
		CheckedAt:       now.Format(time.RFC3339),
		Output:          strings.TrimSpace(output),
		Resources:       buildRuntimeProbeResources(now, output),
		DetectedOS:      detectedOS.String(),
		DetectedArch:    detectedArch.String(),
		DetectionStatus: detectionStatus.String(),
	})
}

func (s *runtimeProtocolServer) handlePreflight(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.PreflightRequest](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInvalidRequest, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	command := buildRemoteRuntimePreflightCommand(RuntimePreflightSpec{
		WorkingDirectory: payload.WorkingDirectory,
		AgentCommand:     payload.AgentCommand,
		Environment:      payload.Environment,
	})
	output, runErr := runRuntimeShellCommand(ctx, command, nil, "")
	if runErr != nil {
		if classified := parseRuntimePreflightFailure(output, runErr); classified != nil {
			details := map[string]any{}
			var preflightErr *RuntimePreflightError
			if errors.As(classified, &preflightErr) {
				details["stage"] = string(preflightErr.Stage)
			}
			return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodePreflight, runtimecontract.ErrorClassMisconfiguration, classified, false, details))
		}
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodePreflight, runtimecontract.ErrorClassTransient, fmt.Errorf("%w: %s", runErr, strings.TrimSpace(output)), true, nil))
	}
	return s.sendResponse(ctx, envelope, map[string]any{"ok": true})
}

func (s *runtimeProtocolServer) handleWorkspacePrepare(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.WorkspacePrepareRequest](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInvalidRequest, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	input := workspaceinfra.SetupInput{
		WorkspaceRoot:    payload.WorkspaceRoot,
		OrganizationSlug: payload.OrganizationSlug,
		ProjectSlug:      payload.ProjectSlug,
		AgentName:        payload.AgentName,
		TicketIdentifier: payload.TicketIdentifier,
		Repos:            make([]workspaceinfra.RepoInput, 0, len(payload.Repos)),
	}
	for _, repo := range payload.Repos {
		item := workspaceinfra.RepoInput{
			Name:             repo.Name,
			RepositoryURL:    repo.RepositoryURL,
			DefaultBranch:    repo.DefaultBranch,
			WorkspaceDirname: repo.WorkspaceDirname,
			BranchName:       repo.BranchName,
		}
		if repo.HTTPBasicAuth != nil {
			item.HTTPBasicAuth = &workspaceinfra.HTTPBasicAuthInput{
				Username: repo.HTTPBasicAuth.Username,
				Password: repo.HTTPBasicAuth.Password,
			}
		}
		input.Repos = append(input.Repos, item)
	}
	request, err := workspaceinfra.ParseSetupRequest(input)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeWorkspace, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	workspaceItem, err := workspaceinfra.NewManager().Prepare(ctx, request)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeWorkspace, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	response := runtimecontract.WorkspacePrepareResponse{
		Path:       workspaceItem.Path,
		BranchName: workspaceItem.BranchName,
		Repos:      make([]runtimecontract.PreparedRepo, 0, len(workspaceItem.Repos)),
	}
	for _, repo := range workspaceItem.Repos {
		response.Repos = append(response.Repos, runtimecontract.PreparedRepo{
			Name:             repo.Name,
			RepositoryURL:    repo.RepositoryURL,
			DefaultBranch:    repo.DefaultBranch,
			BranchName:       repo.BranchName,
			WorkspaceDirname: repo.WorkspaceDirname,
			HeadCommit:       repo.HeadCommit,
			Path:             repo.Path,
		})
	}
	return s.sendResponse(ctx, envelope, response)
}

func (s *runtimeProtocolServer) handleWorkspaceReset(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.WorkspaceResetRequest](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInvalidRequest, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	if err := removeLocalPath(payload.Path); err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeWorkspace, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	return s.sendResponse(ctx, envelope, map[string]any{"ok": true})
}

func (s *runtimeProtocolServer) handleArtifactSync(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.ArtifactSyncRequest](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInvalidRequest, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	if err := materializeArtifactEntries(payload.TargetRoot, payload.RemovePaths, payload.Entries); err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeArtifactSync, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	return s.sendResponse(ctx, envelope, map[string]any{"ok": true})
}

func (s *runtimeProtocolServer) handleCommandOpen(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.CommandOpenRequest](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInvalidRequest, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	sessionID, err := s.startSession(shellCommandSpec(payload.Command), "")
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeProcessStart, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	return s.sendResponse(ctx, envelope, runtimecontract.SessionResponse{SessionID: sessionID})
}

func (s *runtimeProtocolServer) handleProcessStart(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.ProcessStartRequest](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInvalidRequest, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	spec := processCommandSpec{
		command: payload.Command,
		args:    payload.Args,
		env:     payload.Environment,
		cwd:     payload.WorkingDirectory,
	}
	sessionID, err := s.startSession(spec, payload.WorkingDirectory)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeProcessStart, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	return s.sendResponse(ctx, envelope, runtimecontract.SessionResponse{SessionID: sessionID})
}

func (s *runtimeProtocolServer) handleSessionInput(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.SessionInputRequest](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInvalidRequest, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	session, ok := s.session(payload.SessionID)
	if !ok {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeSessionNotFound, runtimecontract.ErrorClassMisconfiguration, fmt.Errorf("runtime session %s is not active", payload.SessionID), false, nil))
	}
	if err := session.writeInput(payload); err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeSessionNotFound, runtimecontract.ErrorClassTransient, err, true, nil))
	}
	return s.sendResponse(ctx, envelope, map[string]any{"ok": true})
}

func (s *runtimeProtocolServer) handleSessionSignal(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.SessionSignalRequest](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInvalidRequest, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	session, ok := s.session(payload.SessionID)
	if !ok {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeSessionNotFound, runtimecontract.ErrorClassMisconfiguration, fmt.Errorf("runtime session %s is not active", payload.SessionID), false, nil))
	}
	if err := session.signal(payload.Signal); err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeProcessSignal, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	return s.sendResponse(ctx, envelope, map[string]any{"ok": true})
}

func (s *runtimeProtocolServer) handleSessionClose(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.SessionCloseRequest](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInvalidRequest, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	session, ok := s.session(payload.SessionID)
	if ok {
		session.stop()
	}
	return s.sendResponse(ctx, envelope, map[string]any{"ok": true})
}

func (s *runtimeProtocolServer) handleProcessStatus(ctx context.Context, envelope runtimecontract.Envelope) error {
	payload, err := runtimecontract.DecodePayload[runtimecontract.ProcessStatusRequest](envelope)
	if err != nil {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeInvalidRequest, runtimecontract.ErrorClassMisconfiguration, err, false, nil))
	}
	session, ok := s.session(payload.SessionID)
	if !ok {
		return s.sendError(ctx, envelope, runtimeErrorPayload(runtimecontract.ErrorCodeSessionNotFound, runtimecontract.ErrorClassMisconfiguration, fmt.Errorf("runtime session %s is not active", payload.SessionID), false, nil))
	}
	running, exitCode := session.status()
	return s.sendResponse(ctx, envelope, runtimecontract.ProcessStatusResponse{
		SessionID: payload.SessionID,
		Running:   running,
		ExitCode:  exitCode,
	})
}

func (s *runtimeProtocolServer) sendResponse(ctx context.Context, request runtimecontract.Envelope, payload any) error {
	return s.send(ctx, runtimecontract.Envelope{
		Version:   runtimecontract.ProtocolVersion,
		Type:      runtimecontract.MessageTypeResponse,
		RequestID: request.RequestID,
		Operation: request.Operation,
		Payload:   mustRuntimePayload(payload),
	})
}

func (s *runtimeProtocolServer) sendError(ctx context.Context, request runtimecontract.Envelope, payload runtimecontract.ErrorPayload) error {
	return s.send(ctx, runtimecontract.Envelope{
		Version:   runtimecontract.ProtocolVersion,
		Type:      runtimecontract.MessageTypeResponse,
		RequestID: request.RequestID,
		Operation: request.Operation,
		Error:     &payload,
	})
}

func (s *runtimeProtocolServer) session(sessionID string) (*runtimeProcessSession, bool) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	session, ok := s.sessions[strings.TrimSpace(sessionID)]
	return session, ok
}

func (s *runtimeProtocolServer) deleteSession(sessionID string) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	delete(s.sessions, strings.TrimSpace(sessionID))
}

type commandSpec interface {
	start(context.Context, string, runtimeEnvelopeSender) (*exec.Cmd, io.WriteCloser, error)
}

type shellCommandSpec string

func (s shellCommandSpec) start(ctx context.Context, sessionID string, send runtimeEnvelopeSender) (*exec.Cmd, io.WriteCloser, error) {
	command := strings.TrimSpace(string(s))
	if command == "" {
		return nil, nil, fmt.Errorf("remote command must not be empty")
	}
	shell, args := listenerShellCommand(command)
	cmd := exec.CommandContext(ctx, shell, args...) // #nosec G204 -- runtime contract intentionally runs orchestrator-provided shell commands.
	cmd.SysProcAttr = runtimeProcessSysProcAttr()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	cmd.Stdout = runtimeProcessStreamWriter{sessionID: sessionID, stream: "stdout", send: send}
	cmd.Stderr = runtimeProcessStreamWriter{sessionID: sessionID, stream: "stderr", send: send}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, stdin, nil
}

type processCommandSpec struct {
	command string
	args    []string
	env     []string
	cwd     string
}

func (s processCommandSpec) start(ctx context.Context, sessionID string, send runtimeEnvelopeSender) (*exec.Cmd, io.WriteCloser, error) {
	command := strings.TrimSpace(s.command)
	if command == "" {
		return nil, nil, fmt.Errorf("agent cli command must not be empty")
	}
	cmd := exec.CommandContext(ctx, command, append([]string(nil), s.args...)...) // #nosec G204 -- runtime contract intentionally runs orchestrator-provided processes.
	cmd.SysProcAttr = runtimeProcessSysProcAttr()
	if trimmed := strings.TrimSpace(s.cwd); trimmed != "" {
		cmd.Dir = trimmed
	}
	if len(s.env) > 0 {
		cmd.Env = append(os.Environ(), s.env...)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	cmd.Stdout = runtimeProcessStreamWriter{sessionID: sessionID, stream: "stdout", send: send}
	cmd.Stderr = runtimeProcessStreamWriter{sessionID: sessionID, stream: "stderr", send: send}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, stdin, nil
}

func (s *runtimeProtocolServer) startSession(spec commandSpec, workingDirectory string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	sessionID := uuid.NewString()
	cmd, stdin, err := spec.start(ctx, sessionID, s.send)
	if err != nil {
		cancel()
		return "", err
	}
	session := &runtimeProcessSession{
		id:       sessionID,
		cmd:      cmd,
		stdin:    stdin,
		cancel:   cancel,
		send:     s.send,
		onFinish: s.deleteSession,
	}
	s.sessionMu.Lock()
	s.sessions[sessionID] = session
	s.sessionMu.Unlock()
	go session.run()
	_ = workingDirectory
	return sessionID, nil
}

type runtimeProcessSession struct {
	id string

	cmd      *exec.Cmd
	stdin    io.WriteCloser
	cancel   context.CancelFunc
	send     runtimeEnvelopeSender
	onFinish func(string)

	done     chan struct{}
	doneOnce sync.Once

	statusMu sync.Mutex
	running  bool
	exitCode *int
}

func (s *runtimeProcessSession) run() {
	s.statusMu.Lock()
	s.running = true
	s.statusMu.Unlock()
	s.done = make(chan struct{})
	go s.wait()
}

func (s *runtimeProcessSession) wait() {
	exitCode := 0
	if err := s.cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr):
			exitCode = exitErr.ExitCode()
		case errors.Is(err, context.Canceled):
			exitCode = 130
		default:
			exitCode = 1
		}
	}
	s.statusMu.Lock()
	s.running = false
	s.exitCode = &exitCode
	s.statusMu.Unlock()
	_ = s.send(context.Background(), runtimecontract.Envelope{
		Version:   runtimecontract.ProtocolVersion,
		Type:      runtimecontract.MessageTypeEvent,
		Operation: runtimecontract.OperationSessionExit,
		Payload: mustRuntimePayload(runtimecontract.SessionExitEvent{
			SessionID: s.id,
			ExitCode:  exitCode,
		}),
	})
	s.doneOnce.Do(func() {
		close(s.done)
	})
	if s.onFinish != nil {
		s.onFinish(s.id)
	}
}

func (s *runtimeProcessSession) writeInput(payload runtimecontract.SessionInputRequest) error {
	if payload.CloseStdin {
		if s.stdin != nil {
			return s.stdin.Close()
		}
		return nil
	}
	if s.stdin == nil {
		return fmt.Errorf("session stdin is unavailable")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload.DataBase64))
	if err != nil {
		return fmt.Errorf("decode session input: %w", err)
	}
	_, err = s.stdin.Write(data)
	return err
}

func (s *runtimeProcessSession) signal(raw string) error {
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("runtime session process is not running")
	}
	pid := s.cmd.Process.Pid
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "INT":
		if runtime.GOOS == "windows" {
			return s.cmd.Process.Signal(os.Interrupt)
		}
		return syscall.Kill(-pid, syscall.SIGINT)
	case "TERM":
		if runtime.GOOS == "windows" {
			return s.cmd.Process.Kill()
		}
		return syscall.Kill(-pid, syscall.SIGTERM)
	case "KILL":
		if runtime.GOOS == "windows" {
			return s.cmd.Process.Signal(os.Kill)
		}
		return syscall.Kill(-pid, syscall.SIGKILL)
	default:
		return fmt.Errorf("unsupported remote signal %q", raw)
	}
}

func (s *runtimeProcessSession) stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		if runtime.GOOS == "windows" {
			_ = s.cmd.Process.Kill()
			return
		}
		_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
	}
}

func (s *runtimeProcessSession) status() (bool, *int) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	if s.exitCode == nil {
		return s.running, nil
	}
	value := *s.exitCode
	return s.running, &value
}

type runtimeProcessStreamWriter struct {
	sessionID string
	stream    string
	send      runtimeEnvelopeSender
}

func (w runtimeProcessStreamWriter) Write(p []byte) (int, error) {
	if len(p) == 0 || w.send == nil {
		return len(p), nil
	}
	_ = w.send(context.Background(), runtimecontract.Envelope{
		Version:   runtimecontract.ProtocolVersion,
		Type:      runtimecontract.MessageTypeEvent,
		Operation: runtimecontract.OperationSessionOutput,
		Payload: mustRuntimePayload(runtimecontract.SessionOutputEvent{
			SessionID:  w.sessionID,
			Stream:     w.stream,
			DataBase64: base64.StdEncoding.EncodeToString(append([]byte(nil), p...)),
		}),
	})
	return len(p), nil
}

func runRuntimeShellCommand(ctx context.Context, command string, environment []string, workingDirectory string) (string, error) {
	shell, args := listenerShellCommand(command)
	cmd := exec.CommandContext(ctx, shell, args...) // #nosec G204,G702 -- runtime probe/preflight intentionally executes contract commands.
	cmd.SysProcAttr = runtimeProcessSysProcAttr()
	if len(environment) > 0 {
		cmd.Env = append(os.Environ(), environment...)
	}
	if strings.TrimSpace(workingDirectory) != "" {
		cmd.Dir = strings.TrimSpace(workingDirectory)
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runtimeErrorPayload(
	code runtimecontract.ErrorCode,
	class runtimecontract.ErrorClass,
	err error,
	retryable bool,
	details map[string]any,
) runtimecontract.ErrorPayload {
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	return runtimecontract.ErrorPayload{
		Code:      code,
		Class:     class,
		Message:   message,
		Retryable: retryable,
		Details:   details,
	}
}

func buildRuntimeProbeResources(checkedAt time.Time, output string) map[string]any {
	detectedOS, detectedArch, detectionStatus := machineprobe.DetectPlatformFromProbeOutput(output)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	resources := map[string]any{
		"checked_at":       checkedAt.Format(time.RFC3339),
		"last_success":     true,
		"detected_os":      detectedOS.String(),
		"detected_arch":    detectedArch.String(),
		"detection_status": detectionStatus.String(),
	}
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		resources["remote_user"] = strings.TrimSpace(lines[0])
	}
	if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
		resources["remote_host"] = strings.TrimSpace(lines[1])
	}
	if len(lines) > 2 && strings.TrimSpace(lines[2]) != "" {
		resources["kernel"] = strings.TrimSpace(lines[2])
	}
	resources["platform_family"] = runtime.GOOS
	return resources
}

func runtimeProcessSysProcAttr() *syscall.SysProcAttr {
	if runtime.GOOS == "windows" {
		return nil
	}
	return &syscall.SysProcAttr{Setpgid: true}
}

func mustRuntimePayload(value any) json.RawMessage {
	payload, err := marshalRuntimePayload(value)
	if err != nil {
		panic(err)
	}
	return payload
}

func runtimeProcessRequestFromSpec(spec provider.AgentCLIProcessSpec) runtimecontract.ProcessStartRequest {
	request := runtimecontract.ProcessStartRequest{
		Command:     spec.Command.String(),
		Args:        append([]string(nil), spec.Args...),
		Environment: append([]string(nil), spec.Environment...),
	}
	if spec.WorkingDirectory != nil {
		request.WorkingDirectory = spec.WorkingDirectory.String()
	}
	return request
}

func runtimeWorkspacePrepareRequest(request workspaceinfra.SetupRequest) runtimecontract.WorkspacePrepareRequest {
	result := runtimecontract.WorkspacePrepareRequest{
		WorkspaceRoot:    request.WorkspaceRoot,
		OrganizationSlug: request.OrganizationSlug,
		ProjectSlug:      request.ProjectSlug,
		TicketIdentifier: request.TicketIdentifier,
		Repos:            make([]runtimecontract.WorkspaceRepo, 0, len(request.Repos)),
	}
	for _, repo := range request.Repos {
		item := runtimecontract.WorkspaceRepo{
			Name:             repo.Name,
			RepositoryURL:    repo.RepositoryURL,
			DefaultBranch:    repo.DefaultBranch,
			WorkspaceDirname: &repo.WorkspaceDirname,
			BranchName:       &repo.BranchName,
		}
		if repo.HTTPBasicAuth != nil {
			item.HTTPBasicAuth = &runtimecontract.WorkspaceRepoAuth{
				Username: repo.HTTPBasicAuth.Username,
				Password: repo.HTTPBasicAuth.Password,
			}
		}
		result.Repos = append(result.Repos, item)
	}
	return result
}

func runtimeWorkspaceResponse(response runtimecontract.WorkspacePrepareResponse) workspaceinfra.Workspace {
	workspaceItem := workspaceinfra.Workspace{
		Path:       response.Path,
		BranchName: response.BranchName,
		Repos:      make([]workspaceinfra.PreparedRepo, 0, len(response.Repos)),
	}
	for _, repo := range response.Repos {
		workspaceItem.Repos = append(workspaceItem.Repos, workspaceinfra.PreparedRepo{
			Name:             repo.Name,
			RepositoryURL:    repo.RepositoryURL,
			DefaultBranch:    repo.DefaultBranch,
			BranchName:       repo.BranchName,
			WorkspaceDirname: repo.WorkspaceDirname,
			HeadCommit:       repo.HeadCommit,
			Path:             repo.Path,
		})
	}
	return workspaceItem
}

func augmentRuntimeProbe(machine domain.Machine, probe runtimecontract.ProbeResponse) domain.MachineProbe {
	checkedAt := time.Now().UTC()
	if parsedCheckedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(probe.CheckedAt)); err == nil {
		checkedAt = parsedCheckedAt.UTC()
	}
	detectedOS, detectedArch, detectionStatus := parseRuntimeProbePlatform(probe)
	result := domain.MachineProbe{
		CheckedAt:       checkedAt,
		Transport:       machine.ConnectionMode.String(),
		Output:          strings.TrimSpace(probe.Output),
		DetectedOS:      detectedOS,
		DetectedArch:    detectedArch,
		DetectionStatus: detectionStatus,
		Resources:       map[string]any{},
	}
	for key, value := range probe.Resources {
		result.Resources[key] = value
	}
	result.Resources["transport"] = machine.ConnectionMode.String()
	result.Resources["detected_os"] = detectedOS.String()
	result.Resources["detected_arch"] = detectedArch.String()
	result.Resources["detection_status"] = detectionStatus.String()
	if machine.ConnectionMode == domain.MachineConnectionModeWSListener {
		result.Resources["advertised_endpoint"] = strings.TrimSpace(pointerString(machine.AdvertisedEndpoint))
		result.Resources["listener_session_mode"] = "runtime_contract_v1"
	}
	return result
}

func parseRuntimeProbePlatform(probe runtimecontract.ProbeResponse) (domain.MachineDetectedOS, domain.MachineDetectedArch, domain.MachineDetectionStatus) {
	rawDetectedOS := strings.TrimSpace(probe.DetectedOS)
	rawDetectedArch := strings.TrimSpace(probe.DetectedArch)
	rawDetectionStatus := strings.TrimSpace(probe.DetectionStatus)
	if rawDetectedOS == "" && rawDetectedArch == "" && rawDetectionStatus == "" {
		return machineprobe.DetectPlatformFromProbeOutput(probe.Output)
	}

	detectedOS, err := domain.ParseStoredMachineDetectedOS(rawDetectedOS)
	if err != nil {
		detectedOS = domain.MachineDetectedOSUnknown
	}
	detectedArch, err := domain.ParseStoredMachineDetectedArch(rawDetectedArch)
	if err != nil {
		detectedArch = domain.MachineDetectedArchUnknown
	}
	if rawDetectionStatus == "" {
		_, _, derivedStatus := machineprobe.NormalizePlatform(detectedOS.String(), detectedArch.String())
		return detectedOS, detectedArch, derivedStatus
	}
	detectionStatus, err := domain.ParseStoredMachineDetectionStatus(rawDetectionStatus)
	if err != nil {
		detectionStatus = domain.MachineDetectionStatusUnknown
	}
	return detectedOS, detectedArch, detectionStatus
}
