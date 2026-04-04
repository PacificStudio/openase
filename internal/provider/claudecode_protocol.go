package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ClaudeCodeContentBlockKind string

const (
	ClaudeCodeContentBlockKindText          ClaudeCodeContentBlockKind = "text"
	ClaudeCodeContentBlockKindToolUse       ClaudeCodeContentBlockKind = "tool_use"
	ClaudeCodeContentBlockKindServerToolUse ClaudeCodeContentBlockKind = "server_tool_use"
	ClaudeCodeContentBlockKindMCPToolUse    ClaudeCodeContentBlockKind = "mcp_tool_use"
	ClaudeCodeContentBlockKindToolResult    ClaudeCodeContentBlockKind = "tool_result"
)

type ClaudeCodeContentBlock struct {
	Kind      ClaudeCodeContentBlockKind
	ID        string
	Name      string
	Input     map[string]any
	Text      string
	ToolUseID string
	Content   any
	IsError   bool
}

type ClaudeCodeToolUseSemantics struct {
	Command string
}

var claudeCodeCommandToolKinds = map[string]struct{}{
	"functions.exec_command": {},
	"exec_command":           {},
	"bash":                   {},
}

func ParseClaudeCodeMessageBlocks(raw json.RawMessage) ([]ClaudeCodeContentBlock, bool) {
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

	blocks := make([]ClaudeCodeContentBlock, 0, len(envelope.Content))
	for _, rawBlock := range envelope.Content {
		kind := ClaudeCodeContentBlockKind(strings.TrimSpace(readClaudeCodeString(rawBlock, "type")))
		if kind == "" {
			continue
		}
		blocks = append(blocks, ClaudeCodeContentBlock{
			Kind:      kind,
			ID:        readClaudeCodeString(rawBlock, "id"),
			Name:      readClaudeCodeString(rawBlock, "name"),
			Input:     cloneClaudeCodeMap(asClaudeCodeMap(rawBlock["input"])),
			Text:      readClaudeCodeString(rawBlock, "text"),
			ToolUseID: readClaudeCodeString(rawBlock, "tool_use_id"),
			Content:   rawBlock["content"],
			IsError:   readClaudeCodeBool(rawBlock, "is_error"),
		})
	}
	if len(blocks) == 0 {
		return nil, false
	}
	return blocks, true
}

func DeriveClaudeCodeToolUseSemantics(toolName string, input map[string]any) ClaudeCodeToolUseSemantics {
	normalized := strings.ToLower(strings.TrimSpace(toolName))
	if _, ok := claudeCodeCommandToolKinds[normalized]; !ok {
		return ClaudeCodeToolUseSemantics{}
	}
	return ClaudeCodeToolUseSemantics{
		Command: readClaudeCodeString(input, "cmd", "command"),
	}
}

func ExtractClaudeCodeToolResultText(content any) string {
	switch typed := content.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			record := asClaudeCodeMap(item)
			if record == nil {
				text, ok := item.(string)
				if ok && strings.TrimSpace(text) != "" {
					items = append(items, strings.TrimSpace(text))
				}
				continue
			}
			if strings.TrimSpace(readClaudeCodeString(record, "type")) != "text" {
				continue
			}
			if trimmed := strings.TrimSpace(readClaudeCodeString(record, "text")); trimmed != "" {
				items = append(items, trimmed)
			}
		}
		return strings.TrimSpace(strings.Join(items, "\n\n"))
	case map[string]any:
		if strings.TrimSpace(readClaudeCodeString(typed, "type")) == "text" {
			return strings.TrimSpace(readClaudeCodeString(typed, "text"))
		}
	}
	return ""
}

func ClaudeCodeRawPayload(event ClaudeCodeEvent) map[string]any {
	payload := asClaudeCodeMap(decodeClaudeCodeRaw(event.Raw))
	if payload == nil {
		payload = map[string]any{}
	}
	if sessionID := strings.TrimSpace(event.SessionID); sessionID != "" {
		if _, exists := payload["session_id"]; !exists {
			payload["session_id"] = sessionID
		}
	}
	if uuid := strings.TrimSpace(event.UUID); uuid != "" {
		if _, exists := payload["uuid"]; !exists {
			payload["uuid"] = uuid
		}
	}
	if parent := strings.TrimSpace(event.ParentToolUseID); parent != "" {
		if _, exists := payload["parent_tool_use_id"]; !exists {
			payload["parent_tool_use_id"] = parent
		}
	}
	return payload
}

func ClaudeCodeEventTurnID(event ClaudeCodeEvent) string {
	payload := ClaudeCodeRawPayload(event)
	return firstClaudeCodeNonEmptyString(
		readClaudeCodeString(payload, "turn_id", "turnId"),
		readClaudeCodeString(asClaudeCodeMap(payload["event"]), "turn_id", "turnId"),
		readClaudeCodeString(asClaudeCodeMap(payload["data"]), "turn_id", "turnId"),
	)
}

func ClaudeCodeEventSessionID(event ClaudeCodeEvent) string {
	payload := ClaudeCodeRawPayload(event)
	return firstClaudeCodeNonEmptyString(strings.TrimSpace(event.SessionID), readClaudeCodeString(payload, "session_id", "thread_id"))
}

func ClaudeCodeEventUUID(event ClaudeCodeEvent) string {
	return strings.TrimSpace(event.UUID)
}

func ClaudeCodeTurnFailure(event ClaudeCodeEvent) (string, string) {
	message := strings.TrimSpace(event.Result)
	rawPayload := decodeClaudeCodeRaw(event.Raw)
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

func ClaudeCodeAssistantSnapshotContinues(previous string, next string) bool {
	trimmedPrevious := strings.TrimSpace(previous)
	trimmedNext := strings.TrimSpace(next)
	if trimmedPrevious == "" || trimmedNext == "" {
		return false
	}
	return strings.HasPrefix(trimmedNext, trimmedPrevious)
}

func decodeClaudeCodeRaw(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	return decoded
}

func asClaudeCodeMap(value any) map[string]any {
	record, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return record
}

func cloneClaudeCodeMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func readClaudeCodeString(record map[string]any, keys ...string) string {
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

func readClaudeCodeBool(record map[string]any, key string) bool {
	value, ok := record[key]
	if !ok {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
}

func firstClaudeCodeNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
