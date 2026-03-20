package codex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const jsonRPCVersion = "2.0"

const (
	methodInitialize        = "initialize"
	methodInitialized       = "initialized"
	methodThreadStart       = "thread/start"
	methodTurnStart         = "turn/start"
	methodToolCall          = "item/tool/call"
	methodTurnStarted       = "turn/started"
	methodTurnCompleted     = "turn/completed"
	methodTurnError         = "error"
	jsonRPCMethodNotFound   = -32601
	defaultClientName       = "openase"
	defaultClientVersion    = "dev"
	textInputType           = "text"
	toolCallTextOutputType  = "inputText"
	toolCallImageOutputType = "inputImage"
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
	JSONRPC string          `json:"jsonrpc"`
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

func (m jsonRPCMessage) validate() error {
	if strings.TrimSpace(m.JSONRPC) != jsonRPCVersion {
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
	ClientInfo wireClientInfo `json:"clientInfo"`
}

type wireClientInfo struct {
	Name    string  `json:"name"`
	Title   *string `json:"title"`
	Version string  `json:"version"`
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
	Ephemeral              *bool   `json:"ephemeral,omitempty"`
	ExperimentalRawEvents  bool    `json:"experimentalRawEvents"`
	PersistExtendedHistory bool    `json:"persistExtendedHistory"`
}

type wireThreadStartResponse struct {
	Thread wireThread `json:"thread"`
}

type wireThread struct {
	ID string `json:"id"`
}

type wireTurnStartParams struct {
	ThreadID string          `json:"threadId"`
	Input    []wireUserInput `json:"input"`
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

type wireErrorNotification struct {
	Error     wireTurnError `json:"error"`
	WillRetry bool          `json:"willRetry"`
	ThreadID  string        `json:"threadId"`
	TurnID    string        `json:"turnId"`
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
