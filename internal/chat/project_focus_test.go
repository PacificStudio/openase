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

func testBoolPointer(value bool) *bool {
	return &value
}
