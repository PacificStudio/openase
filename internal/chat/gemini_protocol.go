package chat

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Gemini CLI stream-json protocol references:
// - references/gemini-cli/docs/cli/headless.md
// - references/gemini-cli/packages/core/src/output/types.ts
//
// Field provenance rules:
// - Types in this file model Gemini CLI transport fields exactly where the
//   reference exposes them.
// - Any OpenASE-specific interpretation lives in derived helper functions below.

type geminiCLIStreamEventType string

const (
	geminiCLIStreamEventTypeInit       geminiCLIStreamEventType = "init"
	geminiCLIStreamEventTypeMessage    geminiCLIStreamEventType = "message"
	geminiCLIStreamEventTypeToolUse    geminiCLIStreamEventType = "tool_use"
	geminiCLIStreamEventTypeToolResult geminiCLIStreamEventType = "tool_result"
	geminiCLIStreamEventTypeError      geminiCLIStreamEventType = "error"
	geminiCLIStreamEventTypeResult     geminiCLIStreamEventType = "result"
)

type geminiCLIStreamEvent interface {
	geminiCLIType() geminiCLIStreamEventType
}

type geminiCLIBaseEvent struct {
	Type      geminiCLIStreamEventType `json:"type"`
	Timestamp string                   `json:"timestamp"`
}

func (e geminiCLIBaseEvent) geminiCLIType() geminiCLIStreamEventType {
	return e.Type
}

type geminiCLIInitEvent struct {
	geminiCLIBaseEvent
	SessionID string `json:"session_id"`
	Model     string `json:"model"`
}

type geminiCLIMessageEvent struct {
	geminiCLIBaseEvent
	Role    string `json:"role"`
	Content string `json:"content"`
	Delta   bool   `json:"delta,omitempty"`
}

type geminiCLIToolUseEvent struct {
	geminiCLIBaseEvent
	ToolName   string         `json:"tool_name"`
	ToolID     string         `json:"tool_id"`
	Parameters map[string]any `json:"parameters"`
}

type geminiCLIToolError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type geminiCLIToolResultEvent struct {
	geminiCLIBaseEvent
	ToolID string              `json:"tool_id"`
	Status string              `json:"status"`
	Output string              `json:"output,omitempty"`
	Error  *geminiCLIToolError `json:"error,omitempty"`
}

type geminiCLIErrorEvent struct {
	geminiCLIBaseEvent
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type geminiCLIResultError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type geminiCLIModelStats struct {
	TotalTokens  int64 `json:"total_tokens"`
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	Cached       int64 `json:"cached"`
	Input        int64 `json:"input"`
}

type geminiCLIStreamStats struct {
	TotalTokens  int64                          `json:"total_tokens"`
	InputTokens  int64                          `json:"input_tokens"`
	OutputTokens int64                          `json:"output_tokens"`
	Cached       int64                          `json:"cached"`
	Input        int64                          `json:"input"`
	DurationMS   int64                          `json:"duration_ms"`
	ToolCalls    int64                          `json:"tool_calls"`
	Models       map[string]geminiCLIModelStats `json:"models"`
}

type geminiCLIResultEvent struct {
	geminiCLIBaseEvent
	Status string                `json:"status"`
	Error  *geminiCLIResultError `json:"error,omitempty"`
	Stats  *geminiCLIStreamStats `json:"stats,omitempty"`
}

func parseGeminiCLIStreamEvent(raw []byte) (geminiCLIStreamEvent, error) {
	var envelope struct {
		Type geminiCLIStreamEventType `json:"type"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("parse gemini stream event envelope: %w", err)
	}

	switch envelope.Type {
	case geminiCLIStreamEventTypeInit:
		var event geminiCLIInitEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, fmt.Errorf("parse gemini init event: %w", err)
		}
		return event, nil
	case geminiCLIStreamEventTypeMessage:
		var event geminiCLIMessageEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, fmt.Errorf("parse gemini message event: %w", err)
		}
		return event, nil
	case geminiCLIStreamEventTypeToolUse:
		var event geminiCLIToolUseEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, fmt.Errorf("parse gemini tool_use event: %w", err)
		}
		return event, nil
	case geminiCLIStreamEventTypeToolResult:
		var event geminiCLIToolResultEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, fmt.Errorf("parse gemini tool_result event: %w", err)
		}
		return event, nil
	case geminiCLIStreamEventTypeError:
		var event geminiCLIErrorEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, fmt.Errorf("parse gemini error event: %w", err)
		}
		return event, nil
	case geminiCLIStreamEventTypeResult:
		var event geminiCLIResultEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			return nil, fmt.Errorf("parse gemini result event: %w", err)
		}
		return event, nil
	default:
		return nil, fmt.Errorf("unsupported gemini stream event type %q", envelope.Type)
	}
}

type geminiCLIToolSemantic string

const (
	geminiCLIToolSemanticUnknown   geminiCLIToolSemantic = "unknown"
	geminiCLIToolSemanticCommand   geminiCLIToolSemantic = "command"
	geminiCLIToolSemanticRead      geminiCLIToolSemantic = "read"
	geminiCLIToolSemanticEdit      geminiCLIToolSemantic = "edit"
	geminiCLIToolSemanticQuestion  geminiCLIToolSemantic = "question"
	geminiCLIToolSemanticSearch    geminiCLIToolSemantic = "search"
	geminiCLIToolSemanticPlanning  geminiCLIToolSemantic = "planning"
	geminiCLIToolSemanticMemory    geminiCLIToolSemantic = "memory"
	geminiCLIToolSemanticLifecycle geminiCLIToolSemantic = "lifecycle"
)

func deriveGeminiCLIToolSemantic(toolName string) geminiCLIToolSemantic {
	switch strings.TrimSpace(toolName) {
	case "run_shell_command":
		return geminiCLIToolSemanticCommand
	case "read_file", "read_many_files", "list_directory":
		return geminiCLIToolSemanticRead
	case "write_file", "replace":
		return geminiCLIToolSemanticEdit
	case "glob", "grep_search", "google_web_search", "web_fetch":
		return geminiCLIToolSemanticSearch
	case "ask_user":
		return geminiCLIToolSemanticQuestion
	case "write_todos", "enter_plan_mode", "exit_plan_mode":
		return geminiCLIToolSemanticPlanning
	case "activate_skill", "get_internal_docs", "save_memory":
		return geminiCLIToolSemanticMemory
	case "complete_task":
		return geminiCLIToolSemanticLifecycle
	default:
		return geminiCLIToolSemanticUnknown
	}
}

func geminiCLIToolCommand(event geminiCLIToolUseEvent) string {
	if strings.TrimSpace(event.ToolName) != "run_shell_command" {
		return ""
	}
	command, _ := event.Parameters["command"].(string)
	return strings.TrimSpace(command)
}

func geminiCLIToolDisplayName(toolName string) string {
	return strings.TrimSpace(toolName)
}

func geminiCLIToolResultMessage(event geminiCLIToolResultEvent) string {
	if strings.TrimSpace(event.Output) != "" {
		return strings.TrimSpace(event.Output)
	}
	if event.Error != nil && strings.TrimSpace(event.Error.Message) != "" {
		return strings.TrimSpace(event.Error.Message)
	}
	return ""
}
