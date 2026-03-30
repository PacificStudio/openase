package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

const defaultShutdownTimeout = 2 * time.Second
const defaultToolInputAnswer = "This is a non-interactive session. Operator input is unavailable."

type EventType string

const (
	EventTypeToolCallRequested EventType = "tool_call_requested"
	EventTypeTokenUsageUpdated EventType = "token_usage_updated"
	EventTypeOutputProduced    EventType = "output_produced"
	EventTypeTurnStarted       EventType = "turn_started"
	EventTypeTurnCompleted     EventType = "turn_completed"
	EventTypeTurnFailed        EventType = "turn_failed"
)

type ToolCallContentType string

const (
	ToolCallContentTypeText  ToolCallContentType = toolCallTextOutputType
	ToolCallContentTypeImage ToolCallContentType = toolCallImageOutputType
)

type AdapterOptions struct {
	ProcessManager provider.AgentCLIProcessManager
}

type Adapter struct {
	processManager provider.AgentCLIProcessManager
}

type StartRequest struct {
	Process    provider.AgentCLIProcessSpec
	Initialize InitializeParams
	Thread     ThreadStartParams
	Turn       TurnConfig
}

type InitializeParams struct {
	ClientName    string
	ClientVersion string
	ClientTitle   string
}

type ThreadStartParams struct {
	WorkingDirectory       string
	Model                  string
	ModelProvider          string
	ServiceName            string
	BaseInstructions       string
	DeveloperInstructions  string
	ApprovalPolicy         any
	Sandbox                any
	Ephemeral              *bool
	ExperimentalRawEvents  bool
	PersistExtendedHistory bool
}

type TurnConfig struct {
	WorkingDirectory string
	Title            string
	ApprovalPolicy   any
	SandboxPolicy    any
}

type TurnStartResult struct {
	TurnID string
}

type Event struct {
	Type       EventType
	ToolCall   *ToolCallRequest
	TokenUsage *TokenUsageEvent
	Output     *OutputEvent
	Turn       *TurnEvent
}

type ToolCallRequest struct {
	RequestID RequestID
	ThreadID  string
	TurnID    string
	CallID    string
	Tool      string
	Arguments json.RawMessage
}

type ToolCallResult struct {
	Success      bool
	ContentItems []ToolCallContentItem
}

type ToolCallContentItem struct {
	Type     ToolCallContentType
	Text     string
	ImageURL string
}

type TurnEvent struct {
	ThreadID string
	TurnID   string
	Status   string
	Error    *TurnError
}

type TurnError struct {
	Message           string
	AdditionalDetails string
}

type TokenUsageEvent struct {
	ThreadID           string
	TurnID             string
	TotalInputTokens   int64
	TotalOutputTokens  int64
	LastInputTokens    int64
	LastOutputTokens   int64
	TotalTokens        int64
	LastTokens         int64
	ModelContextWindow *int64
}

type OutputEvent struct {
	ThreadID string
	TurnID   string
	ItemID   string
	Stream   string
	Text     string
	Phase    string
	Snapshot bool
}

type Session struct {
	process provider.AgentCLIProcess
	encoder *json.Encoder

	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[string]chan callResult

	doneMu  sync.RWMutex
	doneErr error

	events chan Event
	done   chan struct{}

	nextID int64

	stopRequested atomic.Bool
	shutdownOnce  sync.Once

	stderrMu sync.Mutex
	stderr   bytes.Buffer

	threadID string

	autoApproveRequests         bool
	defaultTurnWorkingDirectory string
	defaultTurnTitle            string
	defaultApprovalPolicy       any
	defaultSandboxPolicy        any
}

type SessionDiagnostic struct {
	PID      int
	ThreadID string
	Error    string
	Stderr   string
}

type callResult struct {
	result json.RawMessage
	err    *jsonRPCError
}

func NewAdapter(options AdapterOptions) (*Adapter, error) {
	if options.ProcessManager == nil {
		return nil, fmt.Errorf("process manager must not be nil")
	}

	return &Adapter{processManager: options.ProcessManager}, nil
}

func (a *Adapter) Start(ctx context.Context, request StartRequest) (*Session, error) {
	if a == nil {
		return nil, fmt.Errorf("adapter must not be nil")
	}
	if ctx == nil {
		return nil, fmt.Errorf("context must not be nil")
	}
	if request.Process.Command == "" {
		return nil, fmt.Errorf("process command must not be empty")
	}

	process, err := a.processManager.Start(ctx, request.Process)
	if err != nil {
		return nil, fmt.Errorf("start codex app server: %w", err)
	}

	session := newSession(process)

	go session.captureStderr()
	go session.readLoop()
	go session.waitLoop()

	initializeParams := newWireInitializeParams(request.Initialize)
	if err := session.call(ctx, methodInitialize, initializeParams, &wireInitializeResponse{}); err != nil {
		_ = session.stopWithTimeout()
		return nil, fmt.Errorf("initialize codex app server: %w", err)
	}
	if err := session.notify(methodInitialized, nil); err != nil {
		_ = session.stopWithTimeout()
		return nil, fmt.Errorf("notify codex app server initialized: %w", err)
	}

	threadParams, err := newWireThreadStartParams(request.Thread)
	if err != nil {
		_ = session.stopWithTimeout()
		return nil, err
	}

	var threadResponse wireThreadStartResponse
	if err := session.call(ctx, methodThreadStart, threadParams, &threadResponse); err != nil {
		_ = session.stopWithTimeout()
		return nil, fmt.Errorf("start codex thread: %w", err)
	}
	if strings.TrimSpace(threadResponse.Thread.ID) == "" {
		_ = session.stopWithTimeout()
		return nil, fmt.Errorf("codex thread/start response missing thread id")
	}
	session.threadID = threadResponse.Thread.ID
	session.autoApproveRequests = approvalPolicyIsNever(request.Thread.ApprovalPolicy)
	session.defaultTurnWorkingDirectory = strings.TrimSpace(request.Turn.WorkingDirectory)
	session.defaultTurnTitle = strings.TrimSpace(request.Turn.Title)
	session.defaultApprovalPolicy = cloneJSONCompatibleValue(request.Turn.ApprovalPolicy)
	session.defaultSandboxPolicy = cloneJSONCompatibleValue(request.Turn.SandboxPolicy)

	return session, nil
}

func (s *Session) ThreadID() string {
	if s == nil {
		return ""
	}

	return s.threadID
}

func (s *Session) Events() <-chan Event {
	if s == nil {
		return nil
	}

	return s.events
}

func (s *Session) Err() error {
	if s == nil {
		return nil
	}

	return s.sessionError()
}

func (s *Session) Diagnostic() SessionDiagnostic {
	if s == nil {
		return SessionDiagnostic{}
	}

	diagnostic := SessionDiagnostic{
		ThreadID: s.threadID,
	}
	if s.process != nil {
		diagnostic.PID = s.process.PID()
	}
	if err := s.sessionError(); err != nil {
		diagnostic.Error = err.Error()
	}

	s.stderrMu.Lock()
	diagnostic.Stderr = strings.TrimSpace(s.stderr.String())
	s.stderrMu.Unlock()

	return diagnostic
}

func (s *Session) SendPrompt(ctx context.Context, prompt string) (TurnStartResult, error) {
	return s.StartTurn(ctx, TurnConfig{Title: s.defaultTurnTitle, WorkingDirectory: s.defaultTurnWorkingDirectory, ApprovalPolicy: s.defaultApprovalPolicy, SandboxPolicy: s.defaultSandboxPolicy}, prompt)
}

func (s *Session) StartTurn(ctx context.Context, config TurnConfig, prompt string) (TurnStartResult, error) {
	if s == nil {
		return TurnStartResult{}, fmt.Errorf("session must not be nil")
	}
	if ctx == nil {
		return TurnStartResult{}, fmt.Errorf("context must not be nil")
	}
	if strings.TrimSpace(s.threadID) == "" {
		return TurnStartResult{}, fmt.Errorf("session thread id must not be empty")
	}

	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "" {
		return TurnStartResult{}, fmt.Errorf("prompt must not be empty")
	}

	turnConfig := mergeTurnConfig(s, config)
	var response wireTurnStartResponse
	err := s.call(ctx, methodTurnStart, wireTurnStartParams{
		ThreadID: s.threadID,
		Input: []wireUserInput{
			{
				Type:         textInputType,
				Text:         trimmedPrompt,
				TextElements: []any{},
			},
		},
		CWD:            optionalAbsolutePath(turnConfig.WorkingDirectory),
		Title:          optionalString(turnConfig.Title),
		ApprovalPolicy: cloneJSONCompatibleValue(turnConfig.ApprovalPolicy),
		SandboxPolicy:  cloneJSONCompatibleValue(turnConfig.SandboxPolicy),
	}, &response)
	if err != nil {
		return TurnStartResult{}, fmt.Errorf("start codex turn: %w", err)
	}
	if strings.TrimSpace(response.Turn.ID) == "" {
		return TurnStartResult{}, fmt.Errorf("codex turn/start response missing turn id")
	}

	return TurnStartResult{TurnID: response.Turn.ID}, nil
}

func (s *Session) RespondToolCall(ctx context.Context, request ToolCallRequest, result ToolCallResult) error {
	if s == nil {
		return fmt.Errorf("session must not be nil")
	}
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}
	if request.RequestID.String() == "" {
		return fmt.Errorf("tool call request id must not be empty")
	}

	payload := wireToolCallResponse{
		Success:      result.Success,
		ContentItems: make([]wireToolCallContentItem, 0, len(result.ContentItems)),
	}
	for _, item := range result.ContentItems {
		switch item.Type {
		case ToolCallContentTypeText:
			payload.ContentItems = append(payload.ContentItems, wireToolCallContentItem{
				Type: toolCallTextOutputType,
				Text: item.Text,
			})
		case ToolCallContentTypeImage:
			payload.ContentItems = append(payload.ContentItems, wireToolCallContentItem{
				Type:     toolCallImageOutputType,
				ImageURL: item.ImageURL,
			})
		default:
			return fmt.Errorf("unsupported tool call content type %q", item.Type)
		}
	}

	return s.respond(request.RequestID, payload)
}

func (s *Session) Stop(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("session must not be nil")
	}
	stopCtx, cancel, err := normalizeStopContext(ctx)
	if err != nil {
		return err
	}
	defer cancel()

	s.stopRequested.Store(true)
	if err := s.process.Stop(stopCtx); err != nil {
		return err
	}

	select {
	case <-s.done:
		return nil
	case <-stopCtx.Done():
		return stopCtx.Err()
	}
}

func newSession(process provider.AgentCLIProcess) *Session {
	return &Session{
		process: process,
		encoder: json.NewEncoder(process.Stdin()),
		pending: map[string]chan callResult{},
		events:  make(chan Event, 32),
		done:    make(chan struct{}),
	}
}

func newWireInitializeParams(params InitializeParams) wireInitializeParams {
	name := strings.TrimSpace(params.ClientName)
	if name == "" {
		name = defaultClientName
	}
	version := strings.TrimSpace(params.ClientVersion)
	if version == "" {
		version = defaultClientVersion
	}

	var title *string
	if trimmed := strings.TrimSpace(params.ClientTitle); trimmed != "" {
		title = &trimmed
	}

	return wireInitializeParams{
		ClientInfo: wireClientInfo{
			Name:    name,
			Title:   title,
			Version: version,
		},
		Capabilities: &wireInitializeCapabilities{
			ExperimentalAPI: true,
		},
	}
}

func newWireThreadStartParams(params ThreadStartParams) (wireThreadStartParams, error) {
	wire := wireThreadStartParams{
		ExperimentalRawEvents:  params.ExperimentalRawEvents,
		PersistExtendedHistory: params.PersistExtendedHistory,
		Ephemeral:              params.Ephemeral,
		ApprovalPolicy:         cloneJSONCompatibleValue(params.ApprovalPolicy),
		Sandbox:                cloneJSONCompatibleValue(params.Sandbox),
	}

	if trimmed := strings.TrimSpace(params.Model); trimmed != "" {
		wire.Model = &trimmed
	}
	if trimmed := strings.TrimSpace(params.ModelProvider); trimmed != "" {
		wire.ModelProvider = &trimmed
	}
	if trimmed := strings.TrimSpace(params.WorkingDirectory); trimmed != "" {
		path, err := provider.ParseAbsolutePath(trimmed)
		if err != nil {
			return wireThreadStartParams{}, fmt.Errorf("parse codex thread cwd: %w", err)
		}
		clean := path.String()
		wire.CWD = &clean
	}
	if trimmed := strings.TrimSpace(params.ServiceName); trimmed != "" {
		wire.ServiceName = &trimmed
	}
	if trimmed := strings.TrimSpace(params.BaseInstructions); trimmed != "" {
		wire.BaseInstructions = &trimmed
	}
	if trimmed := strings.TrimSpace(params.DeveloperInstructions); trimmed != "" {
		wire.DeveloperInstructions = &trimmed
	}

	return wire, nil
}

func (s *Session) call(ctx context.Context, method string, params any, out any) error {
	requestID := newNumericRequestID(atomic.AddInt64(&s.nextID, 1))
	responseCh := make(chan callResult, 1)

	s.pendingMu.Lock()
	s.pending[requestID.String()] = responseCh
	s.pendingMu.Unlock()

	if err := s.send(jsonRPCMessage{
		JSONRPC: jsonRPCVersion,
		ID:      requestID.raw,
		Method:  method,
		Params:  mustMarshalJSON(params),
	}); err != nil {
		s.pendingMu.Lock()
		delete(s.pending, requestID.String())
		s.pendingMu.Unlock()
		return err
	}

	select {
	case <-ctx.Done():
		s.pendingMu.Lock()
		delete(s.pending, requestID.String())
		s.pendingMu.Unlock()
		return ctx.Err()
	case <-s.done:
		return s.sessionError()
	case response := <-responseCh:
		if response.err != nil {
			return fmt.Errorf("codex %s failed: %s (%d)", method, response.err.Message, response.err.Code)
		}
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(response.result, out); err != nil {
			return fmt.Errorf("decode codex %s response: %w", method, err)
		}
		return nil
	}
}

func (s *Session) notify(method string, params any) error {
	message := jsonRPCMessage{
		JSONRPC: jsonRPCVersion,
		Method:  method,
	}
	if params != nil {
		message.Params = mustMarshalJSON(params)
	}

	return s.send(message)
}

func (s *Session) respond(id RequestID, result any) error {
	return s.send(jsonRPCMessage{
		JSONRPC: jsonRPCVersion,
		ID:      id.raw,
		Result:  mustMarshalJSON(result),
	})
}

func (s *Session) respondWithError(id RequestID, code int, message string) error {
	return s.send(jsonRPCMessage{
		JSONRPC: jsonRPCVersion,
		ID:      id.raw,
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
		},
	})
}

func (s *Session) send(message jsonRPCMessage) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if err := s.encoder.Encode(message); err != nil {
		return fmt.Errorf("write codex json-rpc message: %w", err)
	}

	return nil
}

func (s *Session) readLoop() {
	defer close(s.events)

	decoder := json.NewDecoder(s.process.Stdout())
	for {
		var message jsonRPCMessage
		if err := decoder.Decode(&message); err != nil {
			if errors.Is(err, io.EOF) {
				s.shutdown(s.waitForExitCause())
				return
			}

			s.shutdown(fmt.Errorf("decode codex json-rpc message: %w", err))
			return
		}
		select {
		case <-s.done:
			return
		default:
		}
		if err := message.validate(); err != nil {
			s.shutdown(fmt.Errorf("validate codex json-rpc message: %w", err))
			return
		}
		if err := s.handleMessage(message); err != nil {
			s.shutdown(err)
			return
		}
	}
}

func (s *Session) handleMessage(message jsonRPCMessage) error {
	hasMethod := strings.TrimSpace(message.Method) != ""
	hasID := len(bytes.TrimSpace(message.ID)) > 0

	switch {
	case hasMethod && hasID:
		return s.handleServerRequest(message)
	case hasMethod:
		return s.handleNotification(message)
	default:
		return s.handleResponse(message)
	}
}

func (s *Session) handleServerRequest(message jsonRPCMessage) error {
	requestID, err := parseRequestID(message.ID)
	if err != nil {
		return err
	}

	switch message.Method {
	case methodToolCall:
		var params wireToolCallRequestParams
		if err := decodeParams(message.Params, &params); err != nil {
			return fmt.Errorf("decode codex tool call request: %w", err)
		}

		s.emit(Event{
			Type: EventTypeToolCallRequested,
			ToolCall: &ToolCallRequest{
				RequestID: requestID,
				ThreadID:  params.ThreadID,
				TurnID:    params.TurnID,
				CallID:    params.CallID,
				Tool:      params.Tool,
				Arguments: append(json.RawMessage(nil), params.Arguments...),
			},
		})

		return nil
	case methodCommandApproval:
		return s.respondApproval(requestID, "acceptForSession")
	case methodExecApproval, methodPatchApproval:
		return s.respondApproval(requestID, "approved_for_session")
	case methodFileApproval:
		return s.respondApproval(requestID, "acceptForSession")
	case methodRequestUserInput:
		return s.respondToolRequestUserInput(requestID, message.Params)
	default:
		return s.respondWithError(requestID, jsonRPCMethodNotFound, fmt.Sprintf("unsupported codex server request %q", message.Method))
	}
}

func (s *Session) handleNotification(message jsonRPCMessage) error {
	switch message.Method {
	case methodTurnStarted:
		var notification wireTurnNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex turn started notification: %w", err)
		}

		s.emit(Event{
			Type: EventTypeTurnStarted,
			Turn: &TurnEvent{
				ThreadID: notification.ThreadID,
				TurnID:   notification.Turn.ID,
				Status:   notification.Turn.Status,
				Error:    turnErrorFromWire(notification.Turn.Error),
			},
		})

		return nil
	case methodTurnCompleted:
		var notification wireTurnNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex turn completed notification: %w", err)
		}

		s.emit(Event{
			Type: EventTypeTurnCompleted,
			Turn: &TurnEvent{
				ThreadID: notification.ThreadID,
				TurnID:   notification.Turn.ID,
				Status:   notification.Turn.Status,
				Error:    turnErrorFromWire(notification.Turn.Error),
			},
		})

		return nil
	case methodTokenUsageUpdated:
		var notification wireThreadTokenUsageUpdatedNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex token usage notification: %w", err)
		}

		s.emit(Event{
			Type: EventTypeTokenUsageUpdated,
			TokenUsage: &TokenUsageEvent{
				ThreadID:           notification.ThreadID,
				TurnID:             notification.TurnID,
				TotalInputTokens:   notification.TokenUsage.Total.InputTokens,
				TotalOutputTokens:  notification.TokenUsage.Total.OutputTokens,
				LastInputTokens:    notification.TokenUsage.Last.InputTokens,
				LastOutputTokens:   notification.TokenUsage.Last.OutputTokens,
				TotalTokens:        notification.TokenUsage.Total.TotalTokens,
				LastTokens:         notification.TokenUsage.Last.TotalTokens,
				ModelContextWindow: notification.TokenUsage.ModelContextWindow,
			},
		})

		return nil
	case methodAgentMessageDelta:
		var notification wireAgentMessageDeltaNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex agent message delta notification: %w", err)
		}

		s.emit(Event{
			Type: EventTypeOutputProduced,
			Output: &OutputEvent{
				ThreadID: notification.ThreadID,
				TurnID:   notification.TurnID,
				ItemID:   notification.ItemID,
				Stream:   "assistant",
				Text:     notification.Delta,
			},
		})

		return nil
	case methodCommandOutput:
		var notification wireCommandExecutionOutputDeltaNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex command execution output notification: %w", err)
		}

		s.emit(Event{
			Type: EventTypeOutputProduced,
			Output: &OutputEvent{
				ThreadID: notification.ThreadID,
				TurnID:   notification.TurnID,
				ItemID:   notification.ItemID,
				Stream:   "command",
				Text:     notification.Delta,
			},
		})

		return nil
	case methodItemCompleted:
		var notification wireItemCompletedNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex item completed notification: %w", err)
		}

		outputEvent, ok := outputEventFromCompletedItem(notification)
		if !ok {
			return nil
		}
		s.emit(Event{
			Type:   EventTypeOutputProduced,
			Output: outputEvent,
		})

		return nil
	case methodTurnError, methodTurnFailed, methodTurnCancelled:
		var notification wireErrorNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex error notification: %w", err)
		}

		s.emit(Event{
			Type: EventTypeTurnFailed,
			Turn: &TurnEvent{
				ThreadID: notification.ThreadID,
				TurnID:   notification.TurnID,
				Status:   "failed",
				Error: &TurnError{
					Message:           notification.Error.Message,
					AdditionalDetails: notification.Error.AdditionalDetails,
				},
			},
		})

		return nil
	default:
		return nil
	}
}

func outputEventFromCompletedItem(notification wireItemCompletedNotification) (*OutputEvent, bool) {
	switch notification.Item.Type {
	case "agentMessage":
		if strings.TrimSpace(notification.Item.Text) == "" {
			return nil, false
		}

		return &OutputEvent{
			ThreadID: notification.ThreadID,
			TurnID:   notification.TurnID,
			ItemID:   notification.Item.ID,
			Stream:   "assistant",
			Text:     notification.Item.Text,
			Phase:    strings.TrimSpace(notification.Item.Phase),
			Snapshot: true,
		}, true
	case "commandExecution":
		if notification.Item.AggregatedOutput == nil || strings.TrimSpace(*notification.Item.AggregatedOutput) == "" {
			return nil, false
		}

		return &OutputEvent{
			ThreadID: notification.ThreadID,
			TurnID:   notification.TurnID,
			ItemID:   notification.Item.ID,
			Stream:   "command",
			Text:     *notification.Item.AggregatedOutput,
			Snapshot: true,
		}, true
	default:
		return nil, false
	}
}

func (s *Session) handleResponse(message jsonRPCMessage) error {
	requestID, err := parseRequestID(message.ID)
	if err != nil {
		return err
	}

	s.pendingMu.Lock()
	responseCh, ok := s.pending[requestID.String()]
	if ok {
		delete(s.pending, requestID.String())
	}
	s.pendingMu.Unlock()
	if !ok {
		return nil
	}

	responseCh <- callResult{
		result: append(json.RawMessage(nil), message.Result...),
		err:    message.Error,
	}

	return nil
}

func (s *Session) waitLoop() {
	err := s.process.Wait()
	if s.stopRequested.Load() {
		s.shutdown(nil)
		return
	}
	if err != nil {
		s.shutdown(fmt.Errorf("codex app server exited: %w%s", err, s.stderrSuffix()))
		return
	}
}

func (s *Session) captureStderr() {
	reader := s.process.Stderr()
	if reader == nil {
		return
	}

	var buffer [1024]byte
	for {
		count, err := reader.Read(buffer[:])
		if count > 0 {
			s.stderrMu.Lock()
			if s.stderr.Len() < 8192 {
				_, _ = s.stderr.Write(buffer[:count])
			}
			s.stderrMu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (s *Session) emit(event Event) {
	select {
	case s.events <- event:
	case <-s.done:
	}
}

func (s *Session) shutdown(err error) {
	s.shutdownOnce.Do(func() {
		s.doneMu.Lock()
		s.doneErr = err
		s.doneMu.Unlock()

		s.pendingMu.Lock()
		for key, responseCh := range s.pending {
			delete(s.pending, key)
			responseCh <- callResult{
				err: &jsonRPCError{
					Code:    -32000,
					Message: coalesceError(err, "codex session closed"),
				},
			}
		}
		s.pendingMu.Unlock()

		close(s.done)
	})
}

func (s *Session) sessionError() error {
	s.doneMu.RLock()
	defer s.doneMu.RUnlock()

	return s.doneErr
}

func (s *Session) waitForExitCause() error {
	if s == nil {
		return nil
	}
	if s.stopRequested.Load() {
		return nil
	}

	err := s.process.Wait()
	if s.stopRequested.Load() {
		return nil
	}
	if err != nil {
		return fmt.Errorf("codex app server exited: %w%s", err, s.stderrSuffix())
	}

	return nil
}

func (s *Session) stderrSuffix() string {
	s.stderrMu.Lock()
	defer s.stderrMu.Unlock()

	trimmed := strings.TrimSpace(s.stderr.String())
	if trimmed == "" {
		return ""
	}

	return ": " + trimmed
}

func (s *Session) stopWithTimeout() error {
	return s.Stop(context.Background())
}

func normalizeStopContext(ctx context.Context) (context.Context, context.CancelFunc, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("context must not be nil")
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}, nil
	}
	//nolint:gosec // Cancel ownership is intentionally transferred to the caller via the returned cancel func.
	stopCtx, cancel := context.WithTimeout(ctx, defaultShutdownTimeout)
	return stopCtx, cancel, nil
}

func turnErrorFromWire(value *wireTurnError) *TurnError {
	if value == nil {
		return nil
	}

	return &TurnError{
		Message:           value.Message,
		AdditionalDetails: value.AdditionalDetails,
	}
}

func mergeTurnConfig(session *Session, config TurnConfig) TurnConfig {
	merged := TurnConfig{
		WorkingDirectory: session.defaultTurnWorkingDirectory,
		Title:            session.defaultTurnTitle,
		ApprovalPolicy:   cloneJSONCompatibleValue(session.defaultApprovalPolicy),
		SandboxPolicy:    cloneJSONCompatibleValue(session.defaultSandboxPolicy),
	}
	if trimmed := strings.TrimSpace(config.WorkingDirectory); trimmed != "" {
		merged.WorkingDirectory = trimmed
	}
	if trimmed := strings.TrimSpace(config.Title); trimmed != "" {
		merged.Title = trimmed
	}
	if config.ApprovalPolicy != nil {
		merged.ApprovalPolicy = cloneJSONCompatibleValue(config.ApprovalPolicy)
	}
	if config.SandboxPolicy != nil {
		merged.SandboxPolicy = cloneJSONCompatibleValue(config.SandboxPolicy)
	}
	return merged
}

func (s *Session) respondApproval(requestID RequestID, decision string) error {
	if !s.autoApproveRequests {
		return s.respondWithError(requestID, jsonRPCMethodNotFound, "interactive approval is not supported in orchestrated sessions")
	}
	return s.respond(requestID, map[string]any{"decision": decision})
}

func (s *Session) respondToolRequestUserInput(requestID RequestID, raw json.RawMessage) error {
	var params struct {
		Questions []struct {
			ID      string `json:"id"`
			Options []struct {
				Label string `json:"label"`
			} `json:"options"`
		} `json:"questions"`
	}
	if err := decodeParams(raw, &params); err != nil {
		return fmt.Errorf("decode codex requestUserInput request: %w", err)
	}

	answers := make(map[string]map[string][]string, len(params.Questions))
	for _, question := range params.Questions {
		if strings.TrimSpace(question.ID) == "" {
			continue
		}
		answer := defaultToolInputAnswer
		if s.autoApproveRequests {
			if label, ok := approvalOptionLabel(question.Options); ok {
				answer = label
			}
		}
		answers[question.ID] = map[string][]string{"answers": []string{answer}}
	}
	if len(answers) == 0 {
		return s.respondWithError(requestID, jsonRPCMethodNotFound, "requestUserInput requires at least one question")
	}
	return s.respond(requestID, map[string]any{"answers": answers})
}

func approvalOptionLabel(options []struct {
	Label string `json:"label"`
}) (string, bool) {
	for _, option := range options {
		label := strings.TrimSpace(option.Label)
		switch label {
		case "Approve this Session", "Approve Once":
			return label, true
		}
	}
	for _, option := range options {
		label := strings.TrimSpace(option.Label)
		normalized := strings.ToLower(label)
		if strings.HasPrefix(normalized, "approve") || strings.HasPrefix(normalized, "allow") {
			return label, true
		}
	}
	return "", false
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalAbsolutePath(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	path, err := provider.ParseAbsolutePath(trimmed)
	if err != nil {
		return nil
	}
	clean := path.String()
	return &clean
}

func approvalPolicyIsNever(value any) bool {
	text, ok := value.(string)
	return ok && strings.EqualFold(strings.TrimSpace(text), "never")
}

func cloneJSONCompatibleValue(value any) any {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return value
	}
	return decoded
}

func decodeParams(raw json.RawMessage, out any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return fmt.Errorf("params must not be empty")
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return err
	}

	return nil
}

func mustMarshalJSON(value any) json.RawMessage {
	if value == nil {
		return nil
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return bytes
}

func coalesceError(err error, fallback string) string {
	if err == nil {
		return fallback
	}

	return err.Error()
}
