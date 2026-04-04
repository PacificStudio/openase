package orchestrator

import (
	"encoding/json"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestParseClaudeProtocolAssistantMessagePreservesTypedBlocks(t *testing.T) {
	event := provider.ClaudeCodeEvent{
		Kind:      provider.ClaudeCodeEventKindAssistant,
		SessionID: "claude-session-typed",
		UUID:      "assistant-evt-1",
		Message: json.RawMessage(`{
			"content": [
				{"type":"text","text":"Let me inspect the repository first."},
				{"type":"tool_use","id":"toolu_01","name":"functions.exec_command","input":{"cmd":"git status --short"}},
				{"type":"server_tool_use","id":"toolu_02","name":"mcp.search","input":{"q":"foo"}}
			]
		}`),
	}

	parsed, ok := parseClaudeProtocolAssistantMessage(event)
	if !ok {
		t.Fatal("parseClaudeProtocolAssistantMessage returned false")
	}
	if parsed.Envelope.EventUUID != "assistant-evt-1" || parsed.Envelope.SessionID != "claude-session-typed" {
		t.Fatalf("unexpected envelope: %+v", parsed.Envelope)
	}
	if len(parsed.Blocks) != 3 {
		t.Fatalf("parsed %d blocks, want 3", len(parsed.Blocks))
	}
	if parsed.Blocks[0].Kind != claudeProtocolContentBlockText || parsed.Blocks[0].Text != "Let me inspect the repository first." {
		t.Fatalf("unexpected text block: %+v", parsed.Blocks[0])
	}
	if parsed.Blocks[1].Kind != claudeProtocolContentBlockToolUse || parsed.Blocks[1].ID != "toolu_01" || parsed.Blocks[1].Name != "functions.exec_command" {
		t.Fatalf("unexpected tool_use block: %+v", parsed.Blocks[1])
	}
	if parsed.Blocks[2].Kind != claudeProtocolContentBlockServerToolUse || parsed.Blocks[2].Name != "mcp.search" {
		t.Fatalf("unexpected server_tool_use block: %+v", parsed.Blocks[2])
	}
}

func TestParseClaudeProtocolTaskProgressSeparatesReferenceAndObservedFields(t *testing.T) {
	event := provider.ClaudeCodeEvent{
		Kind: provider.ClaudeCodeEventKindTaskProgress,
		Raw: json.RawMessage(`{
			"type":"task_progress",
			"session_id":"claude-session-progress",
			"uuid":"progress-1",
			"task_id":"task-1",
			"tool_use_id":"toolu_progress",
			"description":"Running repository command",
			"usage":{"total_tokens":120,"tool_uses":1,"duration_ms":2500},
			"last_tool_name":"functions.exec_command",
			"summary":"Repository inspection",
			"stream":"command",
			"command":"git status --short",
			"text":"M README.md",
			"snapshot":true
		}`),
	}

	parsed, ok := parseClaudeProtocolTaskProgress(event)
	if !ok {
		t.Fatal("parseClaudeProtocolTaskProgress returned false")
	}
	if parsed.TaskID != "task-1" || parsed.ToolUseID != "toolu_progress" || parsed.Description != "Running repository command" {
		t.Fatalf("unexpected protocol task progress core fields: %+v", parsed)
	}
	if parsed.Usage == nil || parsed.Usage.TotalTokens != 120 || parsed.Usage.ToolUses != 1 || parsed.Usage.DurationMS != 2500 {
		t.Fatalf("unexpected usage summary: %+v", parsed.Usage)
	}
	if parsed.LastToolName != "functions.exec_command" || parsed.Summary != "Repository inspection" {
		t.Fatalf("unexpected task progress metadata: %+v", parsed)
	}
	if parsed.Stream != "command" || parsed.Command != "git status --short" || parsed.Text != "M README.md" {
		t.Fatalf("unexpected observed bridge fields: %+v", parsed)
	}
	if parsed.Snapshot == nil || !*parsed.Snapshot {
		t.Fatalf("unexpected snapshot field: %+v", parsed.Snapshot)
	}
}

func TestDeriveClaudeToolUseSemanticsUsesExplicitAllowlist(t *testing.T) {
	if semantics := deriveClaudeToolUseSemantics("functions.exec_command", map[string]any{"cmd": "git status --short"}); semantics.Kind != claudeDerivedToolKindOpenASECommand || semantics.Command != "git status --short" {
		t.Fatalf("unexpected OpenASE command semantics: %+v", semantics)
	}
	if semantics := deriveClaudeToolUseSemantics("Bash", map[string]any{"command": "pwd"}); semantics.Kind != claudeDerivedToolKindClaudeBash || semantics.Command != "pwd" {
		t.Fatalf("unexpected Bash semantics: %+v", semantics)
	}
	if semantics := deriveClaudeToolUseSemantics("shell_script_runner", map[string]any{"command": "pwd"}); semantics.Kind != claudeDerivedToolKindUnknown || semantics.Command != "" {
		t.Fatalf("expected non-allowlisted shell tool to remain unknown, got %+v", semantics)
	}
}

func TestClaudeRawPayloadKeepsEventUUIDAndParentToolUseIDSeparate(t *testing.T) {
	event := provider.ClaudeCodeEvent{
		Kind:            provider.ClaudeCodeEventKindTaskProgress,
		SessionID:       "claude-session-linkage",
		UUID:            "event-uuid-1",
		ParentToolUseID: "toolu_parent",
		Raw:             json.RawMessage(`{"type":"task_progress","text":"running"}`),
	}

	envelope := claudeProtocolEnvelopeFromEvent(event)
	if envelope.EventUUID != "event-uuid-1" {
		t.Fatalf("EventUUID = %q, want event-uuid-1", envelope.EventUUID)
	}
	if envelope.ParentToolUseID != "toolu_parent" {
		t.Fatalf("ParentToolUseID = %q, want toolu_parent", envelope.ParentToolUseID)
	}
	if got := claudeReadString(envelope.RawPayload, "uuid"); got != "event-uuid-1" {
		t.Fatalf("payload uuid = %q, want event-uuid-1", got)
	}
	if got := claudeReadString(envelope.RawPayload, "parent_tool_use_id"); got != "toolu_parent" {
		t.Fatalf("payload parent_tool_use_id = %q, want toolu_parent", got)
	}
}
