package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseClaudeCodeMessageBlocksPreservesToolUseAndToolResultBlocks(t *testing.T) {
	blocks, ok := ParseClaudeCodeMessageBlocks(json.RawMessage(`{
		"role":"assistant",
		"content":[
			{"type":"text","text":"Inspecting"},
			{"type":"tool_use","id":"toolu_01","name":"functions.exec_command","input":{"cmd":"git status --short"}},
			{"type":"tool_result","tool_use_id":"toolu_01","content":[{"type":"text","text":"M README.md"}]}
		]
	}`))
	if !ok {
		t.Fatal("ParseClaudeCodeMessageBlocks returned false")
	}
	if len(blocks) != 3 {
		t.Fatalf("block count = %d, want 3", len(blocks))
	}
	if blocks[1].Kind != ClaudeCodeContentBlockKindToolUse || blocks[1].Name != "functions.exec_command" {
		t.Fatalf("tool use block = %+v", blocks[1])
	}
	if blocks[2].Kind != ClaudeCodeContentBlockKindToolResult || blocks[2].ToolUseID != "toolu_01" {
		t.Fatalf("tool result block = %+v", blocks[2])
	}
}

func TestExtractClaudeCodeToolResultTextJoinsStructuredTextContent(t *testing.T) {
	text := ExtractClaudeCodeToolResultText([]any{
		map[string]any{"type": "text", "text": "first"},
		map[string]any{"type": "text", "text": "second"},
	})
	if text != "first\n\nsecond" {
		t.Fatalf("tool result text = %q, want joined text blocks", text)
	}
}

func TestClaudeCodeTurnFailureFallsBackToSubtypeSummary(t *testing.T) {
	message, details := ClaudeCodeTurnFailure(ClaudeCodeEvent{
		Kind:    ClaudeCodeEventKindResult,
		Subtype: "error_during_execution",
		Raw:     json.RawMessage(`{"type":"result","subtype":"error_during_execution"}`),
	})
	if !strings.Contains(message, "error_during_execution") {
		t.Fatalf("message = %q, want subtype summary", message)
	}
	if !strings.Contains(details, `"subtype":"error_during_execution"`) {
		t.Fatalf("details = %q, want encoded raw payload", details)
	}
}
