package orchestrator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

// Claude Code stream protocol notes:
//
// 1. The CLI transport that OpenASE reads is line-delimited JSON produced by
//    Claude Code in `--input-format stream-json --output-format stream-json`
//    mode. The top-level event kinds are parsed in
//    internal/infra/adapter/claudecode/adapter.go.
//
// 2. The reference bundle under
//    references/claude-code-source/claude-code-2.1.88/cli.js.map exposes the
//    SDK-side schema names for several events:
//      - SDKAssistantMessageSchema
//      - SDKRateLimitEventSchema
//      - SDKTaskStartedMessageSchema
//      - SDKTaskProgressMessageSchema
//      - SDKTaskNotificationMessageSchema
//      - SDKSessionStateChangedMessageSchema
//
// 3. Those SDK schemas model task/session events as `type:"system"` with a
//    `subtype`. The CLI bridge that OpenASE consumes flattens some of them into
//    top-level event kinds like `task_started` and `task_progress`. We keep the
//    parse layer explicit about this difference so future readers can tell which
//    fields come from Claude's documented schemas and which are observed bridge
//    extensions.

type claudeProtocolEnvelope struct {
	EventKind        provider.ClaudeCodeEventKind
	EventUUID        string
	SessionID        string
	ParentToolUseID  string
	TurnID           string
	RawPayload       map[string]any
	ObservedVariants map[string]any
}

type claudeProtocolContentBlockKind string

const (
	claudeProtocolContentBlockText          claudeProtocolContentBlockKind = "text"
	claudeProtocolContentBlockToolUse       claudeProtocolContentBlockKind = "tool_use"
	claudeProtocolContentBlockServerToolUse claudeProtocolContentBlockKind = "server_tool_use"
	claudeProtocolContentBlockMCPToolUse    claudeProtocolContentBlockKind = "mcp_tool_use"
	claudeProtocolContentBlockToolResult    claudeProtocolContentBlockKind = "tool_result"
)

type claudeProtocolContentBlock struct {
	Kind      claudeProtocolContentBlockKind
	ID        string
	Name      string
	Input     map[string]any
	Text      string
	ToolUseID string
	Content   any
	IsError   bool
}

type claudeProtocolAssistantMessage struct {
	Envelope          claudeProtocolEnvelope
	Blocks            []claudeProtocolContentBlock
	ParentToolUseLink string
}

type claudeProtocolUserMessage struct {
	Envelope claudeProtocolEnvelope
	Blocks   []claudeProtocolContentBlock
}

type claudeProtocolTaskStarted struct {
	Envelope     claudeProtocolEnvelope
	TaskID       string
	ToolUseID    string
	Description  string
	TaskType     string
	WorkflowName string
	Prompt       string

	// Observed bridge-only compatibility fields seen in local runs/tests.
	Status  string
	Message string
}

type claudeProtocolUsageSummary struct {
	TotalTokens int
	ToolUses    int
	DurationMS  int
}

type claudeProtocolTaskProgress struct {
	Envelope         claudeProtocolEnvelope
	TaskID           string
	ToolUseID        string
	Description      string
	Usage            *claudeProtocolUsageSummary
	LastToolName     string
	Summary          string
	WorkflowProgress []any

	// Observed bridge-only compatibility fields seen in local runs/tests.
	Stream   string
	Command  string
	Text     string
	Snapshot *bool
}

type claudeProtocolTaskNotification struct {
	Envelope   claudeProtocolEnvelope
	TaskID     string
	ToolUseID  string
	Status     string
	OutputFile string
	Summary    string
	Usage      *claudeProtocolUsageSummary
}

type claudeProtocolSessionStateChanged struct {
	Envelope    claudeProtocolEnvelope
	State       string
	Detail      string
	ActiveFlags []string
}

type claudeDerivedToolKind string

const (
	claudeDerivedToolKindUnknown        claudeDerivedToolKind = "unknown"
	claudeDerivedToolKindOpenASECommand claudeDerivedToolKind = "openase_command_tool"
	claudeDerivedToolKindClaudeBash     claudeDerivedToolKind = "claude_bash_tool"
)

type claudeDerivedToolUseSemantics struct {
	Kind    claudeDerivedToolKind
	Command string
}

var claudeCommandToolKinds = map[string]claudeDerivedToolKind{
	"functions.exec_command": claudeDerivedToolKindOpenASECommand,
	"exec_command":           claudeDerivedToolKindOpenASECommand,
	"bash":                   claudeDerivedToolKindClaudeBash,
}

func claudeProtocolEnvelopeFromEvent(event provider.ClaudeCodeEvent) claudeProtocolEnvelope {
	payload := claudeRawPayload(event)
	return claudeProtocolEnvelope{
		EventKind:       event.Kind,
		EventUUID:       claudeEventUUID(event),
		SessionID:       claudeEventSessionID(event, payload),
		ParentToolUseID: claudeParentToolUseID(event, payload),
		TurnID:          claudeReadString(payload, "turn_id", "turnId"),
		RawPayload:      payload,
	}
}

func parseClaudeProtocolAssistantMessage(event provider.ClaudeCodeEvent) (claudeProtocolAssistantMessage, bool) {
	blocks, ok := parseClaudeProtocolMessageBlocks(event.Message)
	if !ok {
		return claudeProtocolAssistantMessage{}, false
	}
	envelope := claudeProtocolEnvelopeFromEvent(event)
	return claudeProtocolAssistantMessage{
		Envelope:          envelope,
		Blocks:            blocks,
		ParentToolUseLink: envelope.ParentToolUseID,
	}, true
}

func parseClaudeProtocolUserMessage(event provider.ClaudeCodeEvent) (claudeProtocolUserMessage, bool) {
	blocks, ok := parseClaudeProtocolMessageBlocks(event.Message)
	if !ok {
		return claudeProtocolUserMessage{}, false
	}
	return claudeProtocolUserMessage{
		Envelope: claudeProtocolEnvelopeFromEvent(event),
		Blocks:   blocks,
	}, true
}

func parseClaudeProtocolTaskStarted(event provider.ClaudeCodeEvent) (claudeProtocolTaskStarted, bool) {
	payload := claudeRawPayload(event)
	if len(payload) == 0 {
		return claudeProtocolTaskStarted{}, false
	}
	return claudeProtocolTaskStarted{
		Envelope:     claudeProtocolEnvelopeFromEvent(event),
		TaskID:       claudeReadString(payload, "task_id"),
		ToolUseID:    claudeReadString(payload, "tool_use_id"),
		Description:  claudeReadString(payload, "description"),
		TaskType:     claudeReadString(payload, "task_type"),
		WorkflowName: claudeReadString(payload, "workflow_name"),
		Prompt:       claudeReadString(payload, "prompt"),
		Status:       claudeReadString(payload, "status"),
		Message:      claudeReadString(payload, "message"),
	}, true
}

func parseClaudeProtocolTaskProgress(event provider.ClaudeCodeEvent) (claudeProtocolTaskProgress, bool) {
	payload := claudeRawPayload(event)
	if len(payload) == 0 {
		return claudeProtocolTaskProgress{}, false
	}
	return claudeProtocolTaskProgress{
		Envelope:         claudeProtocolEnvelopeFromEvent(event),
		TaskID:           claudeReadString(payload, "task_id"),
		ToolUseID:        firstClaudeNonEmptyString(claudeReadString(payload, "tool_use_id"), claudeReadString(payload, "parent_tool_use_id")),
		Description:      claudeReadString(payload, "description"),
		Usage:            parseClaudeProtocolUsageSummary(payload["usage"]),
		LastToolName:     claudeReadString(payload, "last_tool_name"),
		Summary:          claudeReadString(payload, "summary"),
		WorkflowProgress: claudeReadSlice(payload, "workflow_progress"),
		Stream:           claudeReadString(payload, "stream"),
		Command:          claudeReadString(payload, "command"),
		Text:             claudeReadString(payload, "text"),
		Snapshot:         claudeReadBoolPointer(payload, "snapshot"),
	}, true
}

func parseClaudeProtocolTaskNotification(event provider.ClaudeCodeEvent) (claudeProtocolTaskNotification, bool) {
	payload := claudeRawPayload(event)
	if len(payload) == 0 {
		return claudeProtocolTaskNotification{}, false
	}
	return claudeProtocolTaskNotification{
		Envelope:   claudeProtocolEnvelopeFromEvent(event),
		TaskID:     claudeReadString(payload, "task_id"),
		ToolUseID:  claudeReadString(payload, "tool_use_id"),
		Status:     claudeReadString(payload, "status"),
		OutputFile: claudeReadString(payload, "output_file"),
		Summary:    claudeReadString(payload, "summary"),
		Usage:      parseClaudeProtocolUsageSummary(payload["usage"]),
	}, true
}

func parseClaudeProtocolSessionStateChanged(event provider.ClaudeCodeEvent) (claudeProtocolSessionStateChanged, bool) {
	stateObject := decodeClaudeSessionStateObject(event)
	if len(stateObject) == 0 {
		return claudeProtocolSessionStateChanged{}, false
	}
	state := firstClaudeNonEmptyString(
		claudeReadString(stateObject, "state"),
		claudeReadString(stateObject, "session_state"),
		claudeReadString(stateObject, "status"),
	)
	if state == "" && strings.TrimSpace(event.Subtype) == "requires_action" {
		state = "requires_action"
	}
	if state == "" {
		return claudeProtocolSessionStateChanged{}, false
	}
	activeFlags := claudeReadStringSlice(stateObject, "active_flags")
	if len(activeFlags) == 0 {
		activeFlags = claudeReadStringSlice(stateObject, "activeFlags")
	}
	if len(activeFlags) == 0 && state == "requires_action" {
		activeFlags = []string{"requires_action"}
	}
	return claudeProtocolSessionStateChanged{
		Envelope:    claudeProtocolEnvelopeFromEvent(event),
		State:       state,
		Detail:      firstClaudeNonEmptyString(claudeReadString(stateObject, "detail"), claudeReadString(stateObject, "message"), claudeReadString(stateObject, "reason"), claudeReadString(claudeMap(stateObject["requires_action"]), "type")),
		ActiveFlags: activeFlags,
	}, true
}

func parseClaudeProtocolMessageBlocks(raw json.RawMessage) ([]claudeProtocolContentBlock, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var envelope struct {
		Content []map[string]any `json:"content"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, false
	}
	if len(envelope.Content) == 0 {
		return nil, false
	}
	blocks := make([]claudeProtocolContentBlock, 0, len(envelope.Content))
	for _, rawBlock := range envelope.Content {
		kind := claudeProtocolContentBlockKind(strings.TrimSpace(claudeReadString(rawBlock, "type")))
		if kind == "" {
			continue
		}
		block := claudeProtocolContentBlock{
			Kind:      kind,
			ID:        claudeReadString(rawBlock, "id"),
			Name:      claudeReadString(rawBlock, "name"),
			Input:     cloneClaudeMap(claudeMap(rawBlock["input"])),
			Text:      claudeReadString(rawBlock, "text"),
			ToolUseID: claudeReadString(rawBlock, "tool_use_id"),
			Content:   rawBlock["content"],
			IsError:   claudeReadBool(rawBlock, "is_error"),
		}
		blocks = append(blocks, block)
	}
	if len(blocks) == 0 {
		return nil, false
	}
	return blocks, true
}

func deriveClaudeToolUseSemantics(toolName string, input map[string]any) claudeDerivedToolUseSemantics {
	normalized := strings.ToLower(strings.TrimSpace(toolName))
	toolKind, ok := claudeCommandToolKinds[normalized]
	if !ok {
		return claudeDerivedToolUseSemantics{Kind: claudeDerivedToolKindUnknown}
	}
	command := claudeReadString(input, "cmd", "command")
	if command == "" {
		return claudeDerivedToolUseSemantics{Kind: toolKind}
	}
	return claudeDerivedToolUseSemantics{
		Kind:    toolKind,
		Command: command,
	}
}

func claudeTaskStartedTracePayload(event claudeProtocolTaskStarted) map[string]any {
	payload := cloneClaudeMap(event.Envelope.RawPayload)
	if payload == nil {
		payload = map[string]any{}
	}
	if event.TaskID != "" {
		payload["task_id"] = event.TaskID
	}
	if event.ToolUseID != "" {
		payload["tool_use_id"] = event.ToolUseID
	}
	if event.Description != "" {
		payload["description"] = event.Description
	}
	if event.TaskType != "" {
		payload["task_type"] = event.TaskType
	}
	if event.WorkflowName != "" {
		payload["workflow_name"] = event.WorkflowName
	}
	if event.Prompt != "" {
		payload["prompt"] = event.Prompt
	}
	return payload
}

func claudeTaskProgressTracePayload(event claudeProtocolTaskProgress) map[string]any {
	payload := cloneClaudeMap(event.Envelope.RawPayload)
	if payload == nil {
		payload = map[string]any{}
	}
	if event.TaskID != "" {
		payload["task_id"] = event.TaskID
	}
	if event.ToolUseID != "" {
		payload["tool_use_id"] = event.ToolUseID
	}
	if event.Description != "" {
		payload["description"] = event.Description
	}
	if event.LastToolName != "" {
		payload["last_tool_name"] = event.LastToolName
	}
	if event.Summary != "" {
		payload["summary"] = event.Summary
	}
	if event.Usage != nil {
		payload["usage"] = map[string]any{
			"total_tokens": event.Usage.TotalTokens,
			"tool_uses":    event.Usage.ToolUses,
			"duration_ms":  event.Usage.DurationMS,
		}
	}
	return payload
}

func claudeTaskNotificationTracePayload(event claudeProtocolTaskNotification) map[string]any {
	payload := cloneClaudeMap(event.Envelope.RawPayload)
	if payload == nil {
		payload = map[string]any{}
	}
	if event.TaskID != "" {
		payload["task_id"] = event.TaskID
	}
	if event.ToolUseID != "" {
		payload["tool_use_id"] = event.ToolUseID
	}
	if event.Status != "" {
		payload["status"] = event.Status
	}
	if event.OutputFile != "" {
		payload["output_file"] = event.OutputFile
	}
	if event.Summary != "" {
		payload["summary"] = event.Summary
	}
	if event.Usage != nil {
		payload["usage"] = map[string]any{
			"total_tokens": event.Usage.TotalTokens,
			"tool_uses":    event.Usage.ToolUses,
			"duration_ms":  event.Usage.DurationMS,
		}
	}
	return payload
}

func claudeSessionStateTracePayload(event claudeProtocolSessionStateChanged) map[string]any {
	payload := cloneClaudeMap(event.Envelope.RawPayload)
	if payload == nil {
		payload = map[string]any{}
	}
	payload["status"] = event.State
	if event.Detail != "" {
		payload["detail"] = event.Detail
	}
	if len(event.ActiveFlags) > 0 {
		payload["active_flags"] = append([]string(nil), event.ActiveFlags...)
	}
	return payload
}

func claudeProtocolTaskStatusText(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if text := claudeReadString(payload, "description", "summary", "message", "text", "detail", "reason"); text != "" {
		return text
	}
	stream := claudeReadString(payload, "stream")
	phase := claudeReadString(payload, "phase")
	switch {
	case stream != "" && phase != "":
		return stream + " / " + phase
	case stream != "":
		return stream
	case phase != "":
		return phase
	}
	if status := claudeReadString(payload, "status", "state", "session_state"); status != "" {
		return "Status: " + status
	}
	return ""
}

func claudeEventUUID(event provider.ClaudeCodeEvent) string {
	return strings.TrimSpace(event.UUID)
}

func claudeParentToolUseID(event provider.ClaudeCodeEvent, payload map[string]any) string {
	return firstClaudeNonEmptyString(strings.TrimSpace(event.ParentToolUseID), claudeReadString(payload, "parent_tool_use_id"))
}

func claudeEventSessionID(event provider.ClaudeCodeEvent, payload map[string]any) string {
	return firstClaudeNonEmptyString(strings.TrimSpace(event.SessionID), claudeReadString(payload, "session_id", "thread_id"))
}

func claudeRawPayload(event provider.ClaudeCodeEvent) map[string]any {
	payload := claudeMap(decodeClaudeRaw(event.Raw))
	if payload == nil {
		payload = map[string]any{}
	}
	if sessionID := strings.TrimSpace(event.SessionID); sessionID != "" {
		if _, exists := payload["session_id"]; !exists {
			payload["session_id"] = sessionID
		}
	}
	if eventUUID := claudeEventUUID(event); eventUUID != "" {
		if _, exists := payload["uuid"]; !exists {
			payload["uuid"] = eventUUID
		}
	}
	if parentToolUseID := strings.TrimSpace(event.ParentToolUseID); parentToolUseID != "" {
		if _, exists := payload["parent_tool_use_id"]; !exists {
			payload["parent_tool_use_id"] = parentToolUseID
		}
	}
	return payload
}

func parseClaudeProtocolUsageSummary(value any) *claudeProtocolUsageSummary {
	record := claudeMap(value)
	if record == nil {
		return nil
	}
	return &claudeProtocolUsageSummary{
		TotalTokens: claudeReadInt(record, "total_tokens"),
		ToolUses:    claudeReadInt(record, "tool_uses"),
		DurationMS:  claudeReadInt(record, "duration_ms"),
	}
}

func claudeReadString(record map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := record[key]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		if trimmed := strings.TrimSpace(text); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func claudeReadInt(record map[string]any, key string) int {
	value, ok := record[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	default:
		return 0
	}
}

func claudeReadBool(record map[string]any, key string) bool {
	value, ok := record[key]
	if !ok {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
}

func claudeReadBoolPointer(record map[string]any, key string) *bool {
	value, ok := record[key]
	if !ok {
		return nil
	}
	typed, ok := value.(bool)
	if !ok {
		return nil
	}
	return &typed
}

func claudeReadSlice(record map[string]any, key string) []any {
	value, ok := record[key]
	if !ok {
		return nil
	}
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return nil
	}
	return append([]any(nil), items...)
}

func claudeReadStringSlice(record map[string]any, key string) []string {
	raw, ok := record[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				continue
			}
			if trimmed := strings.TrimSpace(text); trimmed != "" {
				values = append(values, trimmed)
			}
		}
		return values
	default:
		return nil
	}
}

func firstClaudeNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func decodeClaudeSessionStateObject(event provider.ClaudeCodeEvent) map[string]any {
	if decoded := claudeMap(decodeClaudeRaw(event.Event)); decoded != nil {
		return decoded
	}
	if decoded := claudeMap(decodeClaudeRaw(event.Data)); decoded != nil {
		return decoded
	}
	raw := claudeRawPayload(event)
	if raw == nil {
		return map[string]any{}
	}
	if nested := claudeMap(raw["event"]); nested != nil {
		return nested
	}
	if nested := claudeMap(raw["data"]); nested != nil {
		return nested
	}
	return raw
}

func decodeClaudeRaw(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	return decoded
}

func claudeMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	record, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return record
}

func cloneClaudeMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func claudeTurnFailure(event provider.ClaudeCodeEvent) (string, string) {
	message := strings.TrimSpace(event.Result)
	rawPayload := decodeClaudeRaw(event.Raw)
	additionalDetails := ""
	if rawPayload != nil {
		if encoded, err := json.Marshal(rawPayload); err == nil {
			additionalDetails = string(encoded)
		}
	}
	if message != "" {
		return message, additionalDetails
	}

	subtype := strings.TrimSpace(event.Subtype)
	if subtype != "" {
		summary := fmt.Sprintf("Claude Code reported an empty %s result.", subtype)
		if subtype == "error" {
			summary = "Claude Code reported an empty error result."
		}
		return summary, additionalDetails
	}
	if additionalDetails != "" {
		return "Claude Code reported an empty result error.", additionalDetails
	}
	return "Claude Code reported an empty result error.", ""
}
