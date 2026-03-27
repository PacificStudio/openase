package chat

import (
	"context"
	"strings"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func TestParseStartInputRequiresTicketForTicketDetail(t *testing.T) {
	_, err := ParseStartInput(RawStartInput{
		Message: "Why did it fail?",
		Source:  string(SourceTicketDetail),
		Context: RawChatContext{
			ProjectID: stringPointer("550e8400-e29b-41d4-a716-446655440000"),
		},
	})
	if err == nil || err.Error() != "context.ticket_id is required for source ticket_detail" {
		t.Fatalf("expected missing ticket_id error, got %v", err)
	}
}

func TestMapClaudeEventPromotesActionProposalJSON(t *testing.T) {
	events := mapClaudeEvent(SessionID("session-1"), DefaultMaxTurns, provider.ClaudeCodeEvent{
		Kind: provider.ClaudeCodeEventKindAssistant,
		Message: []byte("{\n" +
			"  \"role\":\"assistant\",\n" +
			"  \"content\":[\n" +
			"    {\n" +
			"      \"type\":\"text\",\n" +
			"      \"text\":\"```json\\n{\\\"type\\\":\\\"action_proposal\\\",\\\"summary\\\":\\\"Create 2 child tickets\\\",\\\"actions\\\":[{\\\"method\\\":\\\"POST\\\",\\\"path\\\":\\\"/api/v1/projects/p/tickets\\\",\\\"body\\\":{\\\"title\\\":\\\"Child\\\"}}]}\\n```\"\n" +
			"    }\n" +
			"  ]\n" +
			"}"),
	})
	if len(events) != 1 {
		t.Fatalf("expected one mapped event, got %d", len(events))
	}

	payload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected action proposal payload map, got %#v", events[0].Payload)
	}
	if payload["type"] != "action_proposal" || payload["summary"] != "Create 2 child tickets" {
		t.Fatalf("unexpected action proposal payload: %#v", payload)
	}
}

func TestParseActionProposalTextAcceptsCodeFenceWithWhitespace(t *testing.T) {
	payload, ok := parseActionProposalText("```json \n {\"type\":\"action_proposal\",\"actions\":[]} \n```")
	if !ok {
		t.Fatalf("expected action proposal to parse")
	}
	if payload["type"] != "action_proposal" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestBuildSystemPromptGuidesHarnessEditorReplies(t *testing.T) {
	workflowID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	service := NewService(nil, nil, nil, nil, harnessWorkflowReader{
		detail: workflowservice.WorkflowDetail{
			Workflow: workflowservice.Workflow{
				Name: "Coding Workflow",
				Type: "coding",
			},
			HarnessContent: "---\ntype: coding\n---\n\nWrite code.\n",
		},
	}, "")

	prompt, err := service.buildSystemPrompt(
		context.Background(),
		StartInput{
			Source: SourceHarnessEditor,
			Context: Context{
				ProjectID:  uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
				WorkflowID: &workflowID,
			},
		},
		catalogdomain.Project{Name: "OpenASE"},
	)
	if err != nil {
		t.Fatalf("build system prompt: %v", err)
	}
	if !containsAll(prompt,
		"Harness 编辑器回复要求",
		"完整 Harness 必须放在一个 ```markdown 代码块中",
		"普通 Harness 建议不要输出 action_proposal",
	) {
		t.Fatalf("expected harness-editor response instructions in prompt, got %q", prompt)
	}
}

func TestBuildBaseArgsAddsModelFlagWhenMissing(t *testing.T) {
	args := buildBaseArgs([]string{"chat"}, "claude-sonnet-4-5")
	if len(args) != 3 {
		t.Fatalf("expected model flag to be appended, got %#v", args)
	}
	if args[1] != "--model" || args[2] != "claude-sonnet-4-5" {
		t.Fatalf("expected model flag args, got %#v", args)
	}
}

func TestBuildCodexArgsDoesNotAppendModelFlag(t *testing.T) {
	args := buildCodexArgs([]string{"app-server", "--listen", "stdio://"})
	if len(args) != 3 {
		t.Fatalf("expected codex args to remain unchanged, got %#v", args)
	}
	if strings.Contains(strings.Join(args, " "), "--model") {
		t.Fatalf("expected codex args without model flag, got %#v", args)
	}
}

type harnessWorkflowReader struct {
	detail workflowservice.WorkflowDetail
}

func (r harnessWorkflowReader) Get(context.Context, uuid.UUID) (workflowservice.WorkflowDetail, error) {
	return r.detail, nil
}

func stringPointer(value string) *string {
	return &value
}

func containsAll(value string, expected ...string) bool {
	for _, item := range expected {
		if !strings.Contains(value, item) {
			return false
		}
	}
	return true
}
