package httpapi

import (
	"testing"

	chat "github.com/BetterAndBetterII/openase/internal/chat"
)

func TestParseProjectConversationTurnRequestPreservesFocus(t *testing.T) {
	t.Parallel()

	request, err := parseProjectConversationTurnRequest(rawConversationTurnRequest{
		Message: "帮我看看这里要怎么改",
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
	if request.Message != "帮我看看这里要怎么改" {
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

func testStringPointer(value string) *string {
	return &value
}

func testBoolPointer(value bool) *bool {
	return &value
}

func testIntPointer(value int) *int {
	return &value
}
