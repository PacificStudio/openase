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
			Kind:             "ticket",
			TicketID:         testStringPointer("550e8400-e29b-41d4-a716-446655440000"),
			TicketIdentifier: testStringPointer("T-123"),
			TicketTitle:      testStringPointer("Investigate CI failure"),
			TicketStatus:     testStringPointer("In Review"),
			SelectedArea:     testStringPointer("detail"),
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
	if request.Focus.Ticket.Identifier != "T-123" || request.Focus.Ticket.SelectedArea != "detail" {
		t.Fatalf("unexpected ticket focus = %#v", request.Focus.Ticket)
	}
}

func testStringPointer(value string) *string {
	return &value
}
