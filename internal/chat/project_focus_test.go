package chat

import "testing"

func TestParseProjectConversationFocusWorkflow(t *testing.T) {
	t.Parallel()

	focus, err := ParseProjectConversationFocus(&RawProjectConversationFocus{
		Kind:          "workflow",
		WorkflowID:    stringPointer("550e8400-e29b-41d4-a716-446655440000"),
		WorkflowName:  stringPointer("Backend Engineer"),
		WorkflowType:  stringPointer("coding"),
		HarnessPath:   stringPointer(".openase/harnesses/backend.md"),
		IsActive:      testBoolPointer(true),
		SelectedArea:  stringPointer("harness"),
		HasDirtyDraft: testBoolPointer(true),
	})
	if err != nil {
		t.Fatalf("ParseProjectConversationFocus() error = %v", err)
	}
	if focus == nil || focus.Workflow == nil {
		t.Fatalf("expected workflow focus, got %#v", focus)
	}
	if focus.Workflow.Name != "Backend Engineer" || !focus.Workflow.HasDirtyDraft {
		t.Fatalf("unexpected workflow focus = %#v", focus.Workflow)
	}
}

func TestParseProjectConversationFocusRejectsMissingRequiredField(t *testing.T) {
	t.Parallel()

	_, err := ParseProjectConversationFocus(&RawProjectConversationFocus{
		Kind:         "ticket",
		TicketID:     stringPointer("550e8400-e29b-41d4-a716-446655440000"),
		TicketStatus: stringPointer("In Review"),
	})
	if err == nil || err.Error() != "focus.ticket_identifier must not be empty" {
		t.Fatalf("unexpected error = %v", err)
	}
}

func TestParseProjectConversationFocusTicketPreservesExpandedPayload(t *testing.T) {
	t.Parallel()

	focus, err := ParseProjectConversationFocus(&RawProjectConversationFocus{
		Kind:               "ticket",
		TicketID:           stringPointer("550e8400-e29b-41d4-a716-446655440000"),
		TicketIdentifier:   stringPointer("ASE-470"),
		TicketTitle:        stringPointer("Replace Ticket AI"),
		TicketDescription:  stringPointer("Route the drawer entry through Project AI."),
		TicketStatus:       stringPointer("In Review"),
		TicketPriority:     stringPointer("high"),
		TicketAttemptCount: focusIntPointer(3),
		TicketRetryPaused:  testBoolPointer(true),
		TicketPauseReason:  stringPointer("Repeated hook failures"),
		SelectedArea:       stringPointer("hooks"),
		TicketDependencies: []RawProjectConversationTicketDependency{
			{
				Identifier: stringPointer("ASE-100"),
				Title:      stringPointer("Primary blocker"),
				Relation:   stringPointer("blocked_by"),
				Status:     stringPointer("started"),
			},
		},
		TicketRepoScopes: []RawProjectConversationTicketRepoScope{
			{
				RepoID:         stringPointer("repo-1"),
				RepoName:       stringPointer("openase"),
				BranchName:     stringPointer("feat/openase-470-project-ai"),
				PullRequestURL: stringPointer("https://github.com/PacificStudio/openase/pull/999"),
			},
		},
		TicketRecentActivity: []RawProjectConversationTicketActivity{
			{
				EventType: stringPointer("ticket.retry_paused"),
				Message:   stringPointer("Paused retries after repeated failures."),
				CreatedAt: stringPointer("2026-04-02T08:10:00Z"),
			},
		},
		TicketHookHistory: []RawProjectConversationTicketHook{
			{
				HookName:  stringPointer("ticket.on_complete"),
				Status:    stringPointer("fail"),
				Output:    stringPointer("go test ./... failed"),
				Timestamp: stringPointer("2026-04-02T08:15:00Z"),
			},
		},
		TicketAssignedAgent: &RawProjectConversationTicketAssignedAgent{
			ID:                  stringPointer("agent-1"),
			Name:                stringPointer("todo-app-coding-01"),
			Provider:            stringPointer("codex-cloud"),
			RuntimeControlState: stringPointer("active"),
			RuntimePhase:        stringPointer("executing"),
		},
		TicketCurrentRun: &RawProjectConversationTicketRun{
			ID:                 stringPointer("run-1"),
			AttemptNumber:      intPointer(3),
			Status:             stringPointer("failed"),
			CurrentStepStatus:  stringPointer("failed"),
			CurrentStepSummary: stringPointer("openase test ./internal/chat"),
			LastError:          stringPointer("ticket.on_complete hook failed"),
		},
		TicketTargetMachine: &RawProjectConversationTicketTargetMachine{
			ID:   stringPointer("machine-1"),
			Name: stringPointer("worker-a"),
			Host: stringPointer("10.0.0.15"),
		},
	})
	if err != nil {
		t.Fatalf("ParseProjectConversationFocus() error = %v", err)
	}
	if focus == nil || focus.Ticket == nil {
		t.Fatalf("expected ticket focus, got %#v", focus)
	}
	if focus.Ticket.Description != "Route the drawer entry through Project AI." ||
		focus.Ticket.AttemptCount != 3 ||
		!focus.Ticket.RetryPaused ||
		focus.Ticket.SelectedArea != "hooks" {
		t.Fatalf("unexpected expanded ticket focus = %#v", focus.Ticket)
	}
	if len(focus.Ticket.Dependencies) != 1 || focus.Ticket.Dependencies[0].Identifier != "ASE-100" {
		t.Fatalf("ticket dependencies = %#v", focus.Ticket.Dependencies)
	}
	if len(focus.Ticket.RepoScopes) != 1 || focus.Ticket.RepoScopes[0].PullRequestURL == "" {
		t.Fatalf("ticket repo scopes = %#v", focus.Ticket.RepoScopes)
	}
	if len(focus.Ticket.RecentActivity) != 1 || focus.Ticket.RecentActivity[0].EventType != "ticket.retry_paused" {
		t.Fatalf("ticket recent activity = %#v", focus.Ticket.RecentActivity)
	}
	if len(focus.Ticket.HookHistory) != 1 || focus.Ticket.HookHistory[0].HookName != "ticket.on_complete" {
		t.Fatalf("ticket hook history = %#v", focus.Ticket.HookHistory)
	}
	if focus.Ticket.AssignedAgent == nil || focus.Ticket.AssignedAgent.Name != "todo-app-coding-01" {
		t.Fatalf("ticket assigned agent = %#v", focus.Ticket.AssignedAgent)
	}
	if focus.Ticket.CurrentRun == nil || focus.Ticket.CurrentRun.ID != "run-1" {
		t.Fatalf("ticket current run = %#v", focus.Ticket.CurrentRun)
	}
	if focus.Ticket.TargetMachine == nil || focus.Ticket.TargetMachine.Name != "worker-a" {
		t.Fatalf("ticket target machine = %#v", focus.Ticket.TargetMachine)
	}
}

func TestParseProjectConversationFocusWorkspace(t *testing.T) {
	t.Parallel()

	focus, err := ParseProjectConversationFocus(&RawProjectConversationFocus{
		Kind:                          "workspace_file",
		ConversationID:                stringPointer("550e8400-e29b-41d4-a716-446655440000"),
		WorkspaceRepoPath:             stringPointer("openase"),
		WorkspaceFilePath:             stringPointer("web/src/lib/app.ts"),
		SelectedArea:                  stringPointer("selection"),
		HasDirtyDraft:                 testBoolPointer(true),
		WorkspaceSelectionFrom:        focusIntPointer(10),
		WorkspaceSelectionTo:          focusIntPointer(24),
		WorkspaceSelectionStartLine:   focusIntPointer(2),
		WorkspaceSelectionStartColumn: focusIntPointer(3),
		WorkspaceSelectionEndLine:     focusIntPointer(3),
		WorkspaceSelectionEndColumn:   focusIntPointer(5),
		WorkspaceSelectionText:        stringPointer("selected text"),
		WorkspaceWorkingSet: []RawProjectConversationWorkspaceWorkingSet{{
			FilePath:       stringPointer("README.md"),
			ContentExcerpt: stringPointer("line one"),
			Dirty:          testBoolPointer(false),
		}},
	})
	if err != nil {
		t.Fatalf("ParseProjectConversationFocus() error = %v", err)
	}
	if focus == nil || focus.Workspace == nil {
		t.Fatalf("expected workspace focus, got %#v", focus)
	}
	if focus.Workspace.RepoPath != "openase" ||
		focus.Workspace.FilePath != "web/src/lib/app.ts" ||
		focus.Workspace.SelectedArea != "selection" ||
		!focus.Workspace.HasDirtyDraft ||
		focus.Workspace.Selection == nil ||
		focus.Workspace.Selection.Text != "selected text" ||
		len(focus.Workspace.WorkingSet) != 1 {
		t.Fatalf("unexpected workspace focus = %#v", focus.Workspace)
	}
}

func testBoolPointer(value bool) *bool {
	return &value
}

func focusIntPointer(value int) *int {
	return &value
}
