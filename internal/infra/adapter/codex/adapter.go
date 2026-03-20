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

type EventType string

const (
	EventTypeToolCallRequested EventType = "tool_call_requested"
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
	Ephemeral              *bool
	ExperimentalRawEvents  bool
	PersistExtendedHistory bool
}

type TurnStartResult struct {
	TurnID string
}

type Event struct {
	Type     EventType
	ToolCall *ToolCallRequest
	Turn     *TurnEvent
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

func (s *Session) SendPrompt(ctx context.Context, prompt string) (TurnStartResult, error) {
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
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}

	s.stopRequested.Store(true)
	if err := s.process.Stop(ctx); err != nil {
		return err
	}

	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
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
	}
}

func newWireThreadStartParams(params ThreadStartParams) (wireThreadStartParams, error) {
	wire := wireThreadStartParams{
		ExperimentalRawEvents:  params.ExperimentalRawEvents,
		PersistExtendedHistory: params.PersistExtendedHistory,
		Ephemeral:              params.Ephemeral,
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
	decoder := json.NewDecoder(s.process.Stdout())
	for {
		var message jsonRPCMessage
		if err := decoder.Decode(&message); err != nil {
			if errors.Is(err, io.EOF) {
				s.shutdown(s.sessionError())
				return
			}

			s.shutdown(fmt.Errorf("decode codex json-rpc message: %w", err))
			return
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
	case methodTurnError:
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

	s.shutdown(s.sessionError())
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
		close(s.events)
	})
}

func (s *Session) sessionError() error {
	s.doneMu.RLock()
	defer s.doneMu.RUnlock()

	return s.doneErr
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	return s.Stop(ctx)
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
