package codex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/logging"
)

const jsonRPCVersion = "2.0"

var _ = logging.DeclareComponent("codex-protocol")

const (
	methodInitialize               = "initialize"
	methodInitialized              = "initialized"
	methodThreadStart              = "thread/start"
	methodThreadResume             = "thread/resume"
	methodThreadStarted            = "thread/started"
	methodThreadStatusChanged      = "thread/status/changed"
	methodThreadCompacted          = "thread/compacted"
	methodTurnStart                = "turn/start"
	methodToolCall                 = "item/tool/call"
	methodCommandApproval          = "item/commandExecution/requestApproval"
	methodExecApproval             = "execCommandApproval"
	methodPatchApproval            = "applyPatchApproval"
	methodFileApproval             = "item/fileChange/requestApproval"
	methodRequestUserInput         = "item/tool/requestUserInput"
	methodAgentMessageDelta        = "item/agentMessage/delta"
	methodItemCompleted            = "item/completed"
	methodCommandOutput            = "item/commandExecution/outputDelta"
	methodTurnStarted              = "turn/started"
	methodTurnCompleted            = "turn/completed"
	methodTurnDiffUpdated          = "turn/diff/updated"
	methodTurnPlanUpdated          = "turn/plan/updated"
	methodTurnFailed               = "turn/failed"
	methodTurnCancelled            = "turn/cancelled"
	methodReasoningSummaryPart     = "item/reasoning/summaryPartAdded"
	methodReasoningSummaryText     = "item/reasoning/summaryTextDelta"
	methodReasoningText            = "item/reasoning/textDelta"
	methodTokenUsageUpdated        = "thread/tokenUsage/updated"
	methodAccountRateLimitsUpdated = "account/rateLimits/updated"
	methodTurnError                = "error"
	jsonRPCMethodNotFound          = -32601
	defaultClientName              = "openase"
	defaultClientVersion           = "dev"
	textInputType                  = "text"
	toolCallTextOutputType         = "inputText"
	toolCallImageOutputType        = "inputImage"
)

type RequestID struct {
	raw json.RawMessage
	key string
}

func newNumericRequestID(value int64) RequestID {
	raw := json.RawMessage(strconv.AppendInt(nil, value, 10))

	return RequestID{
		raw: raw,
		key: string(raw),
	}
}

func parseRequestID(raw json.RawMessage) (RequestID, error) {
	compacted := bytes.TrimSpace(raw)
	if len(compacted) == 0 {
		return RequestID{}, fmt.Errorf("request id must not be empty")
	}

	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(compacted))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return RequestID{}, fmt.Errorf("decode request id: %w", err)
	}

	switch decoded.(type) {
	case string, json.Number:
	default:
		return RequestID{}, fmt.Errorf("request id must be a string or number")
	}

	return RequestID{
		raw: append(json.RawMessage(nil), compacted...),
		key: string(compacted),
	}, nil
}

func ParseRequestIDString(raw string) (RequestID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return RequestID{}, fmt.Errorf("request id must not be empty")
	}
	if strings.HasPrefix(trimmed, "\"") || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") || ('0' <= trimmed[0] && trimmed[0] <= '9') {
		return parseRequestID(json.RawMessage(trimmed))
	}
	encoded, err := json.Marshal(trimmed)
	if err != nil {
		return RequestID{}, err
	}
	return parseRequestID(encoded)
}

func (id RequestID) MarshalJSON() ([]byte, error) {
	if len(id.raw) == 0 {
		return nil, fmt.Errorf("request id must not be empty")
	}

	return append([]byte(nil), id.raw...), nil
}

func (id RequestID) String() string {
	return id.key
}

type jsonRPCMessage struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type RPCError struct {
	Method  string
	Code    int
	Message string
}

func (e *RPCError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("codex %s failed: %s (%d)", e.Method, e.Message, e.Code)
}

func (m jsonRPCMessage) validate() error {
	if trimmed := strings.TrimSpace(m.JSONRPC); trimmed != "" && trimmed != jsonRPCVersion {
		return fmt.Errorf("unsupported jsonrpc version %q", m.JSONRPC)
	}

	hasMethod := strings.TrimSpace(m.Method) != ""
	hasID := len(bytes.TrimSpace(m.ID)) > 0
	hasResult := len(bytes.TrimSpace(m.Result)) > 0
	hasError := m.Error != nil

	switch {
	case hasMethod && (hasResult || hasError):
		return fmt.Errorf("json-rpc message must not mix method with result or error")
	case !hasMethod && !hasID:
		return fmt.Errorf("json-rpc response must include an id")
	case !hasMethod && !hasResult && !hasError:
		return fmt.Errorf("json-rpc response must include either result or error")
	}

	return nil
}

type wireInitializeParams struct {
	ClientInfo   wireClientInfo              `json:"clientInfo"`
	Capabilities *wireInitializeCapabilities `json:"capabilities"`
}

type wireClientInfo struct {
	Name    string  `json:"name"`
	Title   *string `json:"title"`
	Version string  `json:"version"`
}

type wireInitializeCapabilities struct {
	ExperimentalAPI bool     `json:"experimentalApi"`
	OptOutMethods   []string `json:"optOutNotificationMethods,omitempty"`
}

type wireInitializeResponse struct {
	UserAgent      string `json:"userAgent"`
	PlatformFamily string `json:"platformFamily"`
	PlatformOS     string `json:"platformOs"`
}

type wireThreadStartParams struct {
	Model                  *string `json:"model,omitempty"`
	ModelProvider          *string `json:"modelProvider,omitempty"`
	CWD                    *string `json:"cwd,omitempty"`
	ServiceName            *string `json:"serviceName,omitempty"`
	BaseInstructions       *string `json:"baseInstructions,omitempty"`
	DeveloperInstructions  *string `json:"developerInstructions,omitempty"`
	ApprovalPolicy         any     `json:"approvalPolicy,omitempty"`
	Sandbox                any     `json:"sandbox,omitempty"`
	Ephemeral              *bool   `json:"ephemeral,omitempty"`
	ExperimentalRawEvents  bool    `json:"experimentalRawEvents"`
	PersistExtendedHistory bool    `json:"persistExtendedHistory"`
}

type wireThreadStartResponse struct {
	Thread wireThread `json:"thread"`
}

type wireThreadResumeParams struct {
	ThreadID string `json:"threadId"`
	wireThreadStartParams
}

type wireThreadResumeResponse struct {
	Thread wireThread `json:"thread"`
}

type wireThread struct {
	ID     string            `json:"id"`
	Status *wireThreadStatus `json:"status,omitempty"`
}

type wireThreadStatus struct {
	Type        string   `json:"type"`
	ActiveFlags []string `json:"activeFlags,omitempty"`
}

type wireThreadStatusChangedNotification struct {
	ThreadID string           `json:"threadId"`
	Status   wireThreadStatus `json:"status"`
}

type wireThreadStartedNotification struct {
	Thread wireThread `json:"thread"`
}

type wireContextCompactedNotification struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
}

type wireTurnStartParams struct {
	ThreadID       string          `json:"threadId"`
	Input          []wireUserInput `json:"input"`
	CWD            *string         `json:"cwd,omitempty"`
	Title          *string         `json:"title,omitempty"`
	ApprovalPolicy any             `json:"approvalPolicy,omitempty"`
	SandboxPolicy  any             `json:"sandboxPolicy,omitempty"`
}

type wireUserInput struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	TextElements []any  `json:"text_elements,omitempty"`
}

type wireTurnStartResponse struct {
	Turn wireTurn `json:"turn"`
}

type wireTurn struct {
	ID     string         `json:"id"`
	Status string         `json:"status"`
	Error  *wireTurnError `json:"error"`
}

type wireTurnError struct {
	Message           string `json:"message"`
	AdditionalDetails string `json:"additionalDetails"`
}

type wireToolCallRequestParams struct {
	ThreadID  string          `json:"threadId"`
	TurnID    string          `json:"turnId"`
	CallID    string          `json:"callId"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
}

type wireTurnNotification struct {
	ThreadID string   `json:"threadId"`
	Turn     wireTurn `json:"turn"`
}

type wireTurnPlanStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

type wireTurnPlanUpdatedNotification struct {
	ThreadID    string             `json:"threadId"`
	TurnID      string             `json:"turnId"`
	Explanation *string            `json:"explanation"`
	Plan        []wireTurnPlanStep `json:"plan"`
}

type wireTurnDiffUpdatedNotification struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	Diff     string `json:"diff"`
}

type wireErrorNotification struct {
	Error     wireTurnError `json:"error"`
	WillRetry bool          `json:"willRetry"`
	ThreadID  string        `json:"threadId"`
	TurnID    string        `json:"turnId"`
}

type wireThreadTokenUsageUpdatedNotification struct {
	ThreadID   string               `json:"threadId"`
	TurnID     string               `json:"turnId"`
	TokenUsage wireThreadTokenUsage `json:"tokenUsage"`
}

type wireAccountRateLimitsUpdatedNotification struct {
	RateLimits json.RawMessage `json:"rateLimits"`
}

type wireAgentMessageDeltaNotification struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
	Delta    string `json:"delta"`
}

type wireReasoningSummaryPartAddedNotification struct {
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
	ItemID       string `json:"itemId"`
	SummaryIndex int    `json:"summaryIndex"`
}

type wireReasoningSummaryTextDeltaNotification struct {
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
	ItemID       string `json:"itemId"`
	Delta        string `json:"delta"`
	SummaryIndex int    `json:"summaryIndex"`
}

type wireReasoningTextDeltaNotification struct {
	ThreadID     string `json:"threadId"`
	TurnID       string `json:"turnId"`
	ItemID       string `json:"itemId"`
	Delta        string `json:"delta"`
	ContentIndex int    `json:"contentIndex"`
}

type wireCommandExecutionOutputDeltaNotification struct {
	ThreadID string `json:"threadId"`
	TurnID   string `json:"turnId"`
	ItemID   string `json:"itemId"`
	Command  string `json:"command,omitempty"`
	Delta    string `json:"delta"`
}

type wireItemCompletedNotification struct {
	ThreadID string         `json:"threadId"`
	TurnID   string         `json:"turnId"`
	Item     wireThreadItem `json:"item"`
}

type wireThreadItem struct {
	ID               string  `json:"id"`
	Type             string  `json:"type"`
	Text             string  `json:"text,omitempty"`
	Phase            string  `json:"phase,omitempty"`
	Command          *string `json:"command,omitempty"`
	AggregatedOutput *string `json:"aggregatedOutput,omitempty"`
}

type wireThreadTokenUsage struct {
	Total              wireTokenUsageBreakdown `json:"total"`
	Last               wireTokenUsageBreakdown `json:"last"`
	ModelContextWindow *int64                  `json:"modelContextWindow,omitempty"`
}

type wireTokenUsageBreakdown struct {
	InputTokens           int64 `json:"inputTokens,omitempty"`
	CachedInputTokens     int64 `json:"cachedInputTokens,omitempty"`
	OutputTokens          int64 `json:"outputTokens,omitempty"`
	ReasoningOutputTokens int64 `json:"reasoningOutputTokens,omitempty"`
	TotalTokens           int64 `json:"totalTokens,omitempty"`
}

type wireToolCallResponse struct {
	ContentItems []wireToolCallContentItem `json:"contentItems"`
	Success      bool                      `json:"success"`
}

type wireToolCallContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}
