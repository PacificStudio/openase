package httpapi

import (
	"testing"

	chat "github.com/BetterAndBetterII/openase/internal/chat"
	chatdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
)

func TestParseProjectConversationTurnRequestPreservesFocus(t *testing.T) {
	t.Parallel()

	request, err := parseProjectConversationTurnRequest(rawConversationTurnRequest{
		Message: "Help me understand what should change here.",
		Focus: &chat.RawProjectConversationFocus{
			Kind:               "ticket",
			TicketID:           testStringPointer("550e8400-e29b-41d4-a716-446655440000"),
			TicketIdentifier:   testStringPointer("T-123"),
			TicketTitle:        testStringPointer("Investigate CI failure"),
			TicketDescription:  testStringPointer("The ticket drawer should open Project AI."),
			TicketStatus:       testStringPointer("In Review"),
			TicketPriority:     testStringPointer("high"),
			TicketAttemptCount: testIntPointer(3),
			TicketRetryPaused:  testBoolPointer(true),
			TicketPauseReason:  testStringPointer("Repeated hook failures"),
			TicketDependencies: []chat.RawProjectConversationTicketDependency{
				{
					Identifier: testStringPointer("ASE-100"),
					Title:      testStringPointer("Primary blocker"),
				},
			},
			TicketRepoScopes: []chat.RawProjectConversationTicketRepoScope{
				{
					RepoID:         testStringPointer("repo-1"),
					RepoName:       testStringPointer("openase"),
					BranchName:     testStringPointer("feat/openase-470-project-ai"),
					PullRequestURL: testStringPointer("https://github.com/PacificStudio/openase/pull/999"),
				},
			},
			TicketRecentActivity: []chat.RawProjectConversationTicketActivity{
				{
					EventType: testStringPointer("ticket.retry_paused"),
					Message:   testStringPointer("Paused retries after repeated failures."),
					CreatedAt: testStringPointer("2026-04-02T08:10:00Z"),
				},
			},
			TicketHookHistory: []chat.RawProjectConversationTicketHook{
				{
					HookName:  testStringPointer("ticket.on_complete"),
					Status:    testStringPointer("fail"),
					Output:    testStringPointer("go test ./... failed"),
					Timestamp: testStringPointer("2026-04-02T08:15:00Z"),
				},
			},
			TicketAssignedAgent: &chat.RawProjectConversationTicketAssignedAgent{
				Name:     testStringPointer("todo-app-coding-01"),
				Provider: testStringPointer("codex-cloud"),
			},
			TicketCurrentRun: &chat.RawProjectConversationTicketRun{
				ID:     testStringPointer("run-1"),
				Status: testStringPointer("failed"),
			},
			TicketTargetMachine: &chat.RawProjectConversationTicketTargetMachine{
				ID: testStringPointer("machine-1"),
			},
			SelectedArea: testStringPointer("detail"),
		},
	})
	if err != nil {
		t.Fatalf("parseProjectConversationTurnRequest() error = %v", err)
	}
	if request.Message != "Help me understand what should change here." {
		t.Fatalf("message = %q", request.Message)
	}
	if request.Focus == nil || request.Focus.Ticket == nil {
		t.Fatalf("expected ticket focus, got %#v", request.Focus)
	}
	if request.Focus.Ticket.Identifier != "T-123" ||
		request.Focus.Ticket.Description != "The ticket drawer should open Project AI." ||
		request.Focus.Ticket.AttemptCount != 3 ||
		!request.Focus.Ticket.RetryPaused ||
		request.Focus.Ticket.SelectedArea != "detail" ||
		len(request.Focus.Ticket.Dependencies) != 1 ||
		len(request.Focus.Ticket.RepoScopes) != 1 ||
		len(request.Focus.Ticket.RecentActivity) != 1 ||
		len(request.Focus.Ticket.HookHistory) != 1 ||
		request.Focus.Ticket.AssignedAgent == nil ||
		request.Focus.Ticket.CurrentRun == nil ||
		request.Focus.Ticket.TargetMachine == nil {
		t.Fatalf("unexpected ticket focus = %#v", request.Focus.Ticket)
	}
}

func TestParseCreateProjectConversationTerminalSessionRequest(t *testing.T) {
	t.Parallel()

	repoPath := " backend "
	cwdPath := " src "
	cols := 100
	rows := 35
	request, err := parseCreateProjectConversationTerminalSessionRequest(
		rawCreateProjectConversationTerminalSessionRequest{
			Mode:     " shell ",
			RepoPath: &repoPath,
			CWDPath:  &cwdPath,
			Cols:     &cols,
			Rows:     &rows,
		},
	)
	if err != nil {
		t.Fatalf("parseCreateProjectConversationTerminalSessionRequest() error = %v", err)
	}
	if request.Terminal.Mode != chatdomain.TerminalModeShell ||
		request.Terminal.RepoPath == nil ||
		*request.Terminal.RepoPath != "backend" ||
		request.Terminal.CWDPath == nil ||
		*request.Terminal.CWDPath != "src" ||
		request.Terminal.Cols != 100 ||
		request.Terminal.Rows != 35 {
		t.Fatalf("unexpected parsed request = %+v", request.Terminal)
	}
}

func TestParseCreateProjectConversationTerminalSessionRequestRejectsUnsupportedMode(t *testing.T) {
	t.Parallel()

	_, err := parseCreateProjectConversationTerminalSessionRequest(
		rawCreateProjectConversationTerminalSessionRequest{Mode: "tmux"},
	)
	if err == nil {
		t.Fatal("expected unsupported mode error")
	}
}

func TestParseProjectConversationTurnRequestParsesWorkspaceDraftContext(t *testing.T) {
	t.Parallel()

	request, err := parseProjectConversationTurnRequest(rawConversationTurnRequest{
		Message: "Review my unsaved workspace draft.",
		Focus: &chat.RawProjectConversationFocus{
			Kind:              "workspace_file",
			ConversationID:    testStringPointer("550e8400-e29b-41d4-a716-446655440000"),
			WorkspaceRepoPath: testStringPointer("openase"),
			WorkspaceFilePath: testStringPointer("web/src/lib/app.ts"),
			SelectedArea:      testStringPointer("edit"),
			HasDirtyDraft:     testBoolPointer(true),
		},
		WorkspaceFileDraft: &rawProjectConversationWorkspaceFileDraftContext{
			RepoPath:   "openase",
			Path:       "web/src/lib/app.ts",
			Content:    "console.log('draft')\n",
			Encoding:   "utf-8",
			LineEnding: "lf",
		},
	})
	if err != nil {
		t.Fatalf("parseProjectConversationTurnRequest() error = %v", err)
	}
	if request.Focus == nil || request.Focus.Workspace == nil {
		t.Fatalf("expected workspace focus, got %#v", request.Focus)
	}
	if request.WorkspaceFileDraft == nil {
		t.Fatal("expected workspace file draft context")
	}
	if request.WorkspaceFileDraft.RepoPath.String() != "openase" ||
		request.WorkspaceFileDraft.Path.String() != "web/src/lib/app.ts" ||
		request.WorkspaceFileDraft.Content.String() != "console.log('draft')\n" ||
		request.WorkspaceFileDraft.Encoding.String() != "utf-8" ||
		request.WorkspaceFileDraft.LineEnding.String() != "lf" {
		t.Fatalf("unexpected workspace draft context = %#v", request.WorkspaceFileDraft)
	}
}

func TestParseUpdateProjectConversationWorkspaceFileRequest(t *testing.T) {
	t.Parallel()

	request, err := parseUpdateProjectConversationWorkspaceFileRequest(
		rawUpdateProjectConversationWorkspaceFileRequest{
			RepoPath:     "openase",
			Path:         "web/src/lib/app.ts",
			BaseRevision: "rev-1",
			Content:      "console.log('saved')\n",
			Encoding:     "utf-8",
			LineEnding:   "crlf",
		},
	)
	if err != nil {
		t.Fatalf("parseUpdateProjectConversationWorkspaceFileRequest() error = %v", err)
	}
	if request.File.RepoPath.String() != "openase" ||
		request.File.Path.String() != "web/src/lib/app.ts" ||
		request.File.BaseRevision.String() != "rev-1" ||
		request.File.Content.String() != "console.log('saved')\n" ||
		request.File.Encoding.String() != "utf-8" ||
		request.File.LineEnding.String() != "crlf" {
		t.Fatalf("unexpected parsed workspace file save request = %#v", request.File)
	}
}

func testStringPointer(value string) *string {
	return &value
}

func testBoolPointer(value bool) *bool {
	return &value
}

func testIntPointer(value int) *int {
	return &value
}
