package chat

import "testing"

func TestParseGeminiCLIStreamEventParsesToolUseAndResult(t *testing.T) {
	toolUseEvent, err := parseGeminiCLIStreamEvent([]byte(`{
		"type":"tool_use",
		"timestamp":"2026-04-03T06:00:00Z",
		"tool_name":"run_shell_command",
		"tool_id":"tool-1",
		"parameters":{"command":"pwd","dir_path":"/tmp/openase"}
	}`))
	if err != nil {
		t.Fatalf("parseGeminiCLIStreamEvent(tool_use) error = %v", err)
	}
	toolUse, ok := toolUseEvent.(geminiCLIToolUseEvent)
	if !ok {
		t.Fatalf("tool_use event = %#v, want geminiCLIToolUseEvent", toolUseEvent)
	}
	if toolUse.ToolName != "run_shell_command" || toolUse.ToolID != "tool-1" {
		t.Fatalf("unexpected tool_use event: %#v", toolUse)
	}
	if geminiCLIToolCommand(toolUse) != "pwd" {
		t.Fatalf("geminiCLIToolCommand() = %q, want pwd", geminiCLIToolCommand(toolUse))
	}
	if deriveGeminiCLIToolSemantic(toolUse.ToolName) != geminiCLIToolSemanticCommand {
		t.Fatalf("deriveGeminiCLIToolSemantic(%q) = %q, want command", toolUse.ToolName, deriveGeminiCLIToolSemantic(toolUse.ToolName))
	}

	resultEvent, err := parseGeminiCLIStreamEvent([]byte(`{
		"type":"tool_result",
		"timestamp":"2026-04-03T06:00:01Z",
		"tool_id":"tool-1",
		"status":"success",
		"output":"pwd\n/tmp/openase"
	}`))
	if err != nil {
		t.Fatalf("parseGeminiCLIStreamEvent(tool_result) error = %v", err)
	}
	result, ok := resultEvent.(geminiCLIToolResultEvent)
	if !ok {
		t.Fatalf("tool_result event = %#v, want geminiCLIToolResultEvent", resultEvent)
	}
	if geminiCLIToolResultMessage(result) != "pwd\n/tmp/openase" {
		t.Fatalf("geminiCLIToolResultMessage() = %q, want command output", geminiCLIToolResultMessage(result))
	}
}

func TestParseGeminiCLIStreamEventParsesResultStats(t *testing.T) {
	event, err := parseGeminiCLIStreamEvent([]byte(`{
		"type":"result",
		"timestamp":"2026-04-03T06:00:02Z",
		"status":"success",
		"stats":{
			"total_tokens":155,
			"input_tokens":120,
			"output_tokens":35,
			"cached":5,
			"input":115,
			"duration_ms":900,
			"tool_calls":2,
			"models":{
				"gemini-2.5-pro":{
					"total_tokens":155,
					"input_tokens":120,
					"output_tokens":35,
					"cached":5,
					"input":115
				}
			}
		}
	}`))
	if err != nil {
		t.Fatalf("parseGeminiCLIStreamEvent(result) error = %v", err)
	}
	result, ok := event.(geminiCLIResultEvent)
	if !ok {
		t.Fatalf("result event = %#v, want geminiCLIResultEvent", event)
	}
	if result.Status != "success" || result.Stats == nil {
		t.Fatalf("unexpected result event: %#v", result)
	}
	if result.Stats.ToolCalls != 2 || result.Stats.DurationMS != 900 {
		t.Fatalf("unexpected stream stats: %#v", result.Stats)
	}
	if result.Stats.Models["gemini-2.5-pro"].OutputTokens != 35 {
		t.Fatalf("unexpected model stats: %#v", result.Stats.Models)
	}
}

func TestDeriveGeminiCLIToolSemanticUsesExplicitAllowlist(t *testing.T) {
	cases := map[string]geminiCLIToolSemantic{
		"run_shell_command": geminiCLIToolSemanticCommand,
		"replace":           geminiCLIToolSemanticEdit,
		"write_file":        geminiCLIToolSemanticEdit,
		"read_file":         geminiCLIToolSemanticRead,
		"ask_user":          geminiCLIToolSemanticQuestion,
		"google_web_search": geminiCLIToolSemanticSearch,
		"made_up_tool":      geminiCLIToolSemanticUnknown,
	}
	for toolName, want := range cases {
		if got := deriveGeminiCLIToolSemantic(toolName); got != want {
			t.Fatalf("deriveGeminiCLIToolSemantic(%q) = %q, want %q", toolName, got, want)
		}
	}
}
