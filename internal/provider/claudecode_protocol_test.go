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

func TestClaudeCodeTurnFailureUsesUserFacingSubtypeMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		subtype  string
		expected string
	}{
		{
			name:     "execution failure",
			subtype:  "error_during_execution",
			expected: "Claude Code failed while executing the task. Try again or check the logs for more details.",
		},
		{
			name:     "generic error",
			subtype:  "error",
			expected: "Claude Code failed before it returned a result. Try again or check the logs for more details.",
		},
		{
			name:     "unknown subtype falls back to generic guidance",
			subtype:  "backend_unreachable",
			expected: "Claude Code failed before it returned a result. Try again or check the logs for more details.",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			message, details := ClaudeCodeTurnFailure(ClaudeCodeEvent{
				Kind:    ClaudeCodeEventKindResult,
				Subtype: tc.subtype,
				Raw:     json.RawMessage(`{"type":"result","subtype":"` + tc.subtype + `"}`),
			})
			if message != tc.expected {
				t.Fatalf("message = %q, want %q", message, tc.expected)
			}
			if strings.Contains(message, tc.subtype) {
				t.Fatalf("message = %q, should hide internal subtype %q", message, tc.subtype)
			}
			if !strings.Contains(details, `"subtype":"`+tc.subtype+`"`) {
				t.Fatalf("details = %q, want encoded raw payload", details)
			}
		})
	}
}
