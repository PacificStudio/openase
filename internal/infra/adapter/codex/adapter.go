package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

const defaultShutdownTimeout = 2 * time.Second
const defaultToolInputAnswer = "This is a non-interactive session. Operator input is unavailable."

var codexAdapterComponent = logging.DeclareComponent("codex-adapter")

type EventType string

const (
	EventTypeToolCallRequested  EventType = "tool_call_requested"
	EventTypeApprovalRequested  EventType = "approval_requested"
	EventTypeUserInputRequested EventType = "user_input_requested"
	EventTypeItemStarted        EventType = "item_started"
	// #nosec G101 -- event type identifier, not a credential.
	EventTypeTokenUsageUpdated EventType = "token_usage_updated"
	EventTypeRateLimitUpdated  EventType = "rate_limit_updated"
	EventTypeOutputProduced    EventType = "output_produced"
	EventTypeThreadStarted     EventType = "thread_started"
	EventTypeThreadCompacted   EventType = "thread_compacted"
	EventTypeTurnPlanUpdated   EventType = "turn_plan_updated"
	EventTypeTurnDiffUpdated   EventType = "turn_diff_updated"
	EventTypeReasoningUpdated  EventType = "reasoning_updated"
	EventTypeTurnStarted       EventType = "turn_started"
	EventTypeTurnCompleted     EventType = "turn_completed"
	EventTypeTurnInterrupted   EventType = "turn_interrupted"
	EventTypeTurnFailed        EventType = "turn_failed"
	EventTypeThreadStatus      EventType = "thread_status"
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
	logger         *slog.Logger
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
	ResumeThreadID         string
	WorkingDirectory       string
	Model                  string
	ModelProvider          string
	ReasoningEffort        string
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
	Type         EventType
	ToolCall     *ToolCallRequest
	Approval     *ApprovalRequest
	UserInput    *UserInputRequest
	TokenUsage   *TokenUsageEvent
	RateLimit    *provider.CLIRateLimit
	Item         *ItemEvent
	Output       *OutputEvent
	Turn         *TurnEvent
	ThreadStatus *ThreadStatusEvent
	Thread       *ThreadEvent
	Compaction   *ThreadCompactionEvent
	Plan         *TurnPlanEvent
	Diff         *TurnDiffEvent
	Reasoning    *ReasoningEvent
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

type ApprovalRequestKind string

const (
	ApprovalRequestKindCommandExecution ApprovalRequestKind = "command_execution"
	ApprovalRequestKindFileChange       ApprovalRequestKind = "file_change"
)

type ApprovalRequest struct {
	RequestID RequestID
	ThreadID  string
	TurnID    string
	Kind      ApprovalRequestKind
	Options   []ApprovalOption
	Payload   map[string]any
}

type UserInputRequest struct {
	RequestID RequestID
	ThreadID  string
	TurnID    string
	Payload   map[string]any
}

type ApprovalOption struct {
	ID          string
	Label       string
	RawDecision string
}

type TurnEvent struct {
	ThreadID string
	TurnID   string
	Status   string
	Error    *TurnError
}

type ThreadStatusEvent struct {
	ThreadID    string
	Status      string
	ActiveFlags []string
}

type ThreadEvent struct {
	ThreadID    string
	Status      string
	ActiveFlags []string
}

type ThreadCompactionEvent struct {
	ThreadID string
	TurnID   string
}

type TurnPlanStep struct {
	Step   string
	Status string
}

type TurnPlanEvent struct {
	ThreadID    string
	TurnID      string
	Explanation *string
	Plan        []TurnPlanStep
}

type TurnDiffEvent struct {
	ThreadID string
	TurnID   string
	Diff     string
}

type ReasoningKind string

const (
	ReasoningKindSummaryPart ReasoningKind = "summary_part_added"
	ReasoningKindSummaryText ReasoningKind = "summary_text_delta"
	ReasoningKindText        ReasoningKind = "text_delta"
)

type ReasoningEvent struct {
	ThreadID     string
	TurnID       string
	ItemID       string
	Kind         ReasoningKind
	Delta        string
	SummaryIndex *int
	ContentIndex *int
}

type TurnError struct {
	Message           string
	AdditionalDetails string
}

type TokenUsageEvent struct {
	ThreadID               string
	TurnID                 string
	TotalInputTokens       int64
	TotalOutputTokens      int64
	TotalCachedInputTokens int64
	TotalReasoningTokens   int64
	LastInputTokens        int64
	LastOutputTokens       int64
	LastCachedInputTokens  int64
	LastReasoningTokens    int64
	TotalTokens            int64
	LastTokens             int64
	ModelContextWindow     *int64
}

type OutputEvent struct {
	ThreadID string
	TurnID   string
	ItemID   string
	Stream   string
	Command  string
	Text     string
	Phase    string
	Snapshot bool
}

type ItemEvent struct {
	ThreadID string
	TurnID   string
	ItemID   string
	ItemType string
	Phase    string
	Command  string
	Text     string
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

	threadID     string
	statusMu     sync.RWMutex
	threadStatus *ThreadStatusEvent
	logger       *slog.Logger

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

	return &Adapter{
		processManager: options.ProcessManager,
		logger:         logging.WithComponent(nil, codexAdapterComponent),
	}, nil
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

	session := newSessionWithLogger(process, a.logger)

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

	if resumeThreadID := strings.TrimSpace(request.Thread.ResumeThreadID); resumeThreadID != "" {
		resumeParams, err := newWireThreadResumeParams(resumeThreadID, request.Thread)
		if err != nil {
			_ = session.stopWithTimeout()
			return nil, err
		}
		var threadResponse wireThreadResumeResponse
		if err := session.call(ctx, methodThreadResume, resumeParams, &threadResponse); err != nil {
			_ = session.stopWithTimeout()
			return nil, fmt.Errorf("resume codex thread: %w", err)
		}
		if strings.TrimSpace(threadResponse.Thread.ID) == "" {
			_ = session.stopWithTimeout()
			return nil, fmt.Errorf("codex thread/resume response missing thread id")
		}
		session.threadID = threadResponse.Thread.ID
		session.setThreadStatus(threadStatusEventFromWire(session.threadID, threadResponse.Thread.Status))
	} else {
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
		session.setThreadStatus(threadStatusEventFromWire(session.threadID, threadResponse.Thread.Status))
	}
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

func (s *Session) ThreadStatus() *ThreadStatusEvent {
	if s == nil {
		return nil
	}
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()
	if s.threadStatus == nil {
		return nil
	}
	copied := *s.threadStatus
	copied.ActiveFlags = append([]string(nil), s.threadStatus.ActiveFlags...)
	return &copied
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
	return newSessionWithLogger(process, nil)
}

func newSessionWithLogger(process provider.AgentCLIProcess, logger *slog.Logger) *Session {
	return &Session{
		process: process,
		encoder: json.NewEncoder(process.Stdin()),
		pending: map[string]chan callResult{},
		events:  make(chan Event, 32),
		done:    make(chan struct{}),
		logger:  logging.WithComponent(logger, codexAdapterComponent),
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
	if trimmed := strings.TrimSpace(params.ReasoningEffort); trimmed != "" {
		wire.ReasoningEffort = &trimmed
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
		select {
		case response := <-responseCh:
			return decodeCallResult(method, response, out)
		default:
			return s.sessionError()
		}
	case response := <-responseCh:
		return decodeCallResult(method, response, out)
	}
}

func decodeCallResult(method string, response callResult, out any) error {
	if response.err != nil {
		return &RPCError{
			Method:  method,
			Code:    response.err.Code,
			Message: response.err.Message,
		}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(response.result, out); err != nil {
		return fmt.Errorf("decode codex %s response: %w", method, err)
	}
	return nil
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

			s.logger.Error("decode codex json-rpc message failed", "error", err)
			s.shutdown(fmt.Errorf("decode codex json-rpc message: %w", err))
			return
		}
		select {
		case <-s.done:
			return
		default:
		}
		if err := message.validate(); err != nil {
			s.logger.Error("validate codex json-rpc message failed", "method", message.Method, "error", err)
			s.shutdown(fmt.Errorf("validate codex json-rpc message: %w", err))
			return
		}
		if err := s.handleMessage(message); err != nil {
			s.shutdown(err)
			return
		}
	}
}

func IsThreadNotFoundError(err error) bool {
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		return false
	}
	return rpcErr.Code == -32600 && strings.Contains(strings.ToLower(strings.TrimSpace(rpcErr.Message)), "thread not found")
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
		payload, _ := decodeJSONMap(message.Params)
		request := ApprovalRequest{
			RequestID: requestID,
			ThreadID:  readStringMap(payload, "threadId"),
			TurnID:    readStringMap(payload, "turnId"),
			Kind:      ApprovalRequestKindCommandExecution,
			Options:   approvalOptionsForMethod(message.Method),
			Payload:   payload,
		}
		s.emit(Event{
			Type:     EventTypeApprovalRequested,
			Approval: &request,
		})
		if !s.autoApproveRequests {
			return nil
		}
		return s.RespondApproval(context.Background(), request, "approve_for_session")
	case methodExecApproval, methodPatchApproval:
		payload, _ := decodeJSONMap(message.Params)
		request := ApprovalRequest{
			RequestID: requestID,
			ThreadID:  readStringMap(payload, "threadId"),
			TurnID:    readStringMap(payload, "turnId"),
			Kind:      ApprovalRequestKindCommandExecution,
			Options:   approvalOptionsForMethod(message.Method),
			Payload:   payload,
		}
		s.emit(Event{
			Type:     EventTypeApprovalRequested,
			Approval: &request,
		})
		if !s.autoApproveRequests {
			return nil
		}
		return s.RespondApproval(context.Background(), request, "approve_for_session")
	case methodFileApproval:
		payload, _ := decodeJSONMap(message.Params)
		request := ApprovalRequest{
			RequestID: requestID,
			ThreadID:  readStringMap(payload, "threadId"),
			TurnID:    readStringMap(payload, "turnId"),
			Kind:      ApprovalRequestKindFileChange,
			Options:   approvalOptionsForMethod(message.Method),
			Payload:   payload,
		}
		s.emit(Event{
			Type:     EventTypeApprovalRequested,
			Approval: &request,
		})
		if !s.autoApproveRequests {
			return nil
		}
		return s.RespondApproval(context.Background(), request, "approve_for_session")
	case methodRequestUserInput:
		payload, _ := decodeJSONMap(message.Params)
		request := UserInputRequest{
			RequestID: requestID,
			ThreadID:  readStringMap(payload, "threadId"),
			TurnID:    readStringMap(payload, "turnId"),
			Payload:   payload,
		}
		s.emit(Event{
			Type:      EventTypeUserInputRequested,
			UserInput: &request,
		})
		if !s.autoApproveRequests {
			return nil
		}
		return s.RespondUserInput(context.Background(), request, defaultToolRequestUserInputAnswers(payload))
	default:
		s.logger.Warn("unsupported codex server request", "method", message.Method)
		return s.respondWithError(requestID, jsonRPCMethodNotFound, fmt.Sprintf("unsupported codex server request %q", message.Method))
	}
}

func (s *Session) handleNotification(message jsonRPCMessage) error {
	switch message.Method {
	case methodThreadStarted:
		var notification wireThreadStartedNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex thread started notification: %w", err)
		}
		if strings.TrimSpace(notification.Thread.ID) != "" {
			s.threadID = strings.TrimSpace(notification.Thread.ID)
		}
		threadEvent := threadEventFromWire(notification.Thread)
		s.setThreadStatus(threadStatusEventFromThreadEvent(threadEvent))
		if threadEvent != nil {
			s.emit(Event{
				Type:   EventTypeThreadStarted,
				Thread: threadEvent,
			})
		}
		return nil
	case methodThreadStatusChanged:
		var notification wireThreadStatusChangedNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex thread status notification: %w", err)
		}
		event := threadStatusEventFromWire(notification.ThreadID, &notification.Status)
		s.setThreadStatus(event)
		if event != nil {
			s.emit(Event{
				Type:         EventTypeThreadStatus,
				ThreadStatus: event,
			})
		}
		return nil
	case methodThreadCompacted:
		var notification wireContextCompactedNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex thread compacted notification: %w", err)
		}
		s.emit(Event{
			Type: EventTypeThreadCompacted,
			Compaction: &ThreadCompactionEvent{
				ThreadID: strings.TrimSpace(notification.ThreadID),
				TurnID:   strings.TrimSpace(notification.TurnID),
			},
		})
		return nil
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
	case methodTurnPlanUpdated:
		var notification wireTurnPlanUpdatedNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex turn plan notification: %w", err)
		}
		plan := make([]TurnPlanStep, 0, len(notification.Plan))
		for _, item := range notification.Plan {
			plan = append(plan, TurnPlanStep{
				Step:   strings.TrimSpace(item.Step),
				Status: strings.TrimSpace(item.Status),
			})
		}
		s.emit(Event{
			Type: EventTypeTurnPlanUpdated,
			Plan: &TurnPlanEvent{
				ThreadID:    strings.TrimSpace(notification.ThreadID),
				TurnID:      strings.TrimSpace(notification.TurnID),
				Explanation: notification.Explanation,
				Plan:        plan,
			},
		})
		return nil
	case methodTurnDiffUpdated:
		var notification wireTurnDiffUpdatedNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex turn diff notification: %w", err)
		}
		s.emit(Event{
			Type: EventTypeTurnDiffUpdated,
			Diff: &TurnDiffEvent{
				ThreadID: strings.TrimSpace(notification.ThreadID),
				TurnID:   strings.TrimSpace(notification.TurnID),
				Diff:     notification.Diff,
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
	case methodReasoningSummaryPart:
		var notification wireReasoningSummaryPartAddedNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex reasoning summary part notification: %w", err)
		}
		summaryIndex := notification.SummaryIndex
		s.emit(Event{
			Type: EventTypeReasoningUpdated,
			Reasoning: &ReasoningEvent{
				ThreadID:     strings.TrimSpace(notification.ThreadID),
				TurnID:       strings.TrimSpace(notification.TurnID),
				ItemID:       strings.TrimSpace(notification.ItemID),
				Kind:         ReasoningKindSummaryPart,
				SummaryIndex: &summaryIndex,
			},
		})
		return nil
	case methodReasoningSummaryText:
		var notification wireReasoningSummaryTextDeltaNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex reasoning summary text notification: %w", err)
		}
		summaryIndex := notification.SummaryIndex
		s.emit(Event{
			Type: EventTypeReasoningUpdated,
			Reasoning: &ReasoningEvent{
				ThreadID:     strings.TrimSpace(notification.ThreadID),
				TurnID:       strings.TrimSpace(notification.TurnID),
				ItemID:       strings.TrimSpace(notification.ItemID),
				Kind:         ReasoningKindSummaryText,
				Delta:        notification.Delta,
				SummaryIndex: &summaryIndex,
			},
		})
		return nil
	case methodReasoningText:
		var notification wireReasoningTextDeltaNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex reasoning text notification: %w", err)
		}
		contentIndex := notification.ContentIndex
		s.emit(Event{
			Type: EventTypeReasoningUpdated,
			Reasoning: &ReasoningEvent{
				ThreadID:     strings.TrimSpace(notification.ThreadID),
				TurnID:       strings.TrimSpace(notification.TurnID),
				ItemID:       strings.TrimSpace(notification.ItemID),
				Kind:         ReasoningKindText,
				Delta:        notification.Delta,
				ContentIndex: &contentIndex,
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
				ThreadID:               notification.ThreadID,
				TurnID:                 notification.TurnID,
				TotalInputTokens:       notification.TokenUsage.Total.InputTokens,
				TotalOutputTokens:      notification.TokenUsage.Total.OutputTokens,
				TotalCachedInputTokens: notification.TokenUsage.Total.CachedInputTokens,
				TotalReasoningTokens:   notification.TokenUsage.Total.ReasoningOutputTokens,
				LastInputTokens:        notification.TokenUsage.Last.InputTokens,
				LastOutputTokens:       notification.TokenUsage.Last.OutputTokens,
				LastCachedInputTokens:  notification.TokenUsage.Last.CachedInputTokens,
				LastReasoningTokens:    notification.TokenUsage.Last.ReasoningOutputTokens,
				TotalTokens:            notification.TokenUsage.Total.TotalTokens,
				LastTokens:             notification.TokenUsage.Last.TotalTokens,
				ModelContextWindow:     notification.TokenUsage.ModelContextWindow,
			},
		})

		return nil
	case methodAccountRateLimitsUpdated:
		var notification wireAccountRateLimitsUpdatedNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex account rate limits notification: %w", err)
		}

		rateLimit, err := provider.ParseCodexRateLimit(notification.RateLimits)
		if err != nil {
			return fmt.Errorf("parse codex account rate limits notification: %w", err)
		}
		if rateLimit == nil {
			return nil
		}

		s.emit(Event{
			Type:      EventTypeRateLimitUpdated,
			RateLimit: rateLimit,
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
	case methodItemStarted:
		var notification wireItemCompletedNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex item started notification: %w", err)
		}
		itemEvent, ok := itemEventFromWire(notification)
		if !ok {
			return nil
		}
		s.emit(Event{
			Type: EventTypeItemStarted,
			Item: itemEvent,
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
				Command:  strings.TrimSpace(notification.Command),
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
	case methodTurnCancelled:
		var notification wireErrorNotification
		if err := decodeParams(message.Params, &notification); err != nil {
			return fmt.Errorf("decode codex interrupted notification: %w", err)
		}
		s.emit(Event{
			Type: EventTypeTurnInterrupted,
			Turn: &TurnEvent{
				ThreadID: notification.ThreadID,
				TurnID:   notification.TurnID,
				Status:   "interrupted",
				Error: &TurnError{
					Message:           notification.Error.Message,
					AdditionalDetails: notification.Error.AdditionalDetails,
				},
			},
		})
		return nil
	case methodTurnError, methodTurnFailed:
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

func (s *Session) setThreadStatus(status *ThreadStatusEvent) {
	if s == nil {
		return
	}
	s.statusMu.Lock()
	defer s.statusMu.Unlock()
	if status == nil {
		s.threadStatus = nil
		return
	}
	copied := *status
	copied.ActiveFlags = append([]string(nil), status.ActiveFlags...)
	s.threadStatus = &copied
}

func threadStatusEventFromWire(threadID string, status *wireThreadStatus) *ThreadStatusEvent {
	if strings.TrimSpace(threadID) == "" || status == nil {
		return nil
	}
	return &ThreadStatusEvent{
		ThreadID:    strings.TrimSpace(threadID),
		Status:      strings.TrimSpace(status.Type),
		ActiveFlags: append([]string(nil), status.ActiveFlags...),
	}
}

func threadEventFromWire(thread wireThread) *ThreadEvent {
	threadID := strings.TrimSpace(thread.ID)
	if threadID == "" {
		return nil
	}
	result := &ThreadEvent{ThreadID: threadID}
	if thread.Status != nil {
		result.Status = strings.TrimSpace(thread.Status.Type)
		result.ActiveFlags = append([]string(nil), thread.Status.ActiveFlags...)
	}
	return result
}

func threadStatusEventFromThreadEvent(event *ThreadEvent) *ThreadStatusEvent {
	if event == nil {
		return nil
	}
	return &ThreadStatusEvent{
		ThreadID:    event.ThreadID,
		Status:      event.Status,
		ActiveFlags: append([]string(nil), event.ActiveFlags...),
	}
}

func newWireThreadResumeParams(threadID string, params ThreadStartParams) (wireThreadResumeParams, error) {
	startParams, err := newWireThreadStartParams(params)
	if err != nil {
		return wireThreadResumeParams{}, err
	}
	return wireThreadResumeParams{
		ThreadID:              strings.TrimSpace(threadID),
		wireThreadStartParams: startParams,
	}, nil
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
			Command:  strings.TrimSpace(optionalStringValue(notification.Item.Command)),
			Text:     *notification.Item.AggregatedOutput,
			Snapshot: true,
		}, true
	default:
		return nil, false
	}
}

func itemEventFromWire(notification wireItemCompletedNotification) (*ItemEvent, bool) {
	itemID := strings.TrimSpace(notification.Item.ID)
	itemType := strings.TrimSpace(notification.Item.Type)
	if itemID == "" || itemType == "" {
		return nil, false
	}

	event := &ItemEvent{
		ThreadID: strings.TrimSpace(notification.ThreadID),
		TurnID:   strings.TrimSpace(notification.TurnID),
		ItemID:   itemID,
		ItemType: itemType,
		Phase:    strings.TrimSpace(notification.Item.Phase),
		Command:  strings.TrimSpace(optionalStringValue(notification.Item.Command)),
		Text:     strings.TrimSpace(notification.Item.Text),
	}
	return event, true
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
		if err != nil {
			s.logger.Error("codex session shutting down with error", "error", err)
		}
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

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *Session) respondApproval(requestID RequestID, decision string) error {
	if !s.autoApproveRequests {
		return s.respondWithError(requestID, jsonRPCMethodNotFound, "interactive approval is not supported in orchestrated sessions")
	}
	return s.respond(requestID, map[string]any{"decision": decision})
}

func (s *Session) RespondApproval(_ context.Context, request ApprovalRequest, decision string) error {
	if s == nil {
		return fmt.Errorf("session must not be nil")
	}
	rawDecision := strings.TrimSpace(decision)
	for _, option := range request.Options {
		if option.ID == rawDecision || option.RawDecision == rawDecision {
			rawDecision = option.RawDecision
			break
		}
	}
	if rawDecision == "" {
		return fmt.Errorf("approval decision must not be empty")
	}
	return s.respond(request.RequestID, map[string]any{"decision": rawDecision})
}

func (s *Session) RespondUserInput(_ context.Context, request UserInputRequest, answers map[string]any) error {
	if s == nil {
		return fmt.Errorf("session must not be nil")
	}
	if len(answers) == 0 {
		return s.respondWithError(request.RequestID, jsonRPCMethodNotFound, "requestUserInput requires at least one question")
	}
	return s.respond(request.RequestID, map[string]any{"answers": answers})
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

func defaultToolRequestUserInputAnswers(payload map[string]any) map[string]any {
	questions, _ := payload["questions"].([]any)
	answers := make(map[string]any, len(questions))
	for _, rawQuestion := range questions {
		question, ok := rawQuestion.(map[string]any)
		if !ok {
			continue
		}
		id := readStringMap(question, "id")
		if id == "" {
			continue
		}
		answer := defaultToolInputAnswer
		options := readOptionLabels(question["options"])
		if label, ok := approvalOptionLabel(options); ok {
			answer = label
		}
		answers[id] = map[string]any{"answers": []string{answer}}
	}
	return answers
}

func approvalOptionsForMethod(method string) []ApprovalOption {
	switch method {
	case methodExecApproval, methodPatchApproval:
		return []ApprovalOption{
			{ID: "approve_once", Label: "Approve Once", RawDecision: "approved_once"},
			{ID: "approve_for_session", Label: "Approve this Session", RawDecision: "approved_for_session"},
			{ID: "deny", Label: "Deny", RawDecision: "denied"},
		}
	default:
		return []ApprovalOption{
			{ID: "approve_once", Label: "Approve Once", RawDecision: "acceptOnce"},
			{ID: "approve_for_session", Label: "Approve this Session", RawDecision: "acceptForSession"},
			{ID: "deny", Label: "Deny", RawDecision: "deny"},
		}
	}
}

func decodeJSONMap(raw json.RawMessage) (map[string]any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func readStringMap(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

func readOptionLabels(raw any) []struct {
	Label string `json:"label"`
} {
	items, _ := raw.([]any)
	options := make([]struct {
		Label string `json:"label"`
	}, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		options = append(options, struct {
			Label string `json:"label"`
		}{Label: readStringMap(object, "label")})
	}
	return options
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
