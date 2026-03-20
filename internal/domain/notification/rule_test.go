package notification

import (
	"testing"

	"github.com/google/uuid"
)

func TestRuleMatchesSupportsFlatAndNestedFields(t *testing.T) {
	t.Parallel()

	rule := Rule{
		Filter: map[string]any{
			"priority":           "high",
			"ticket.status_name": "In Progress",
		},
	}

	context := map[string]any{
		"priority": "high",
		"ticket": map[string]any{
			"status_name": "In Progress",
		},
	}

	if !rule.Matches(context) {
		t.Fatalf("expected rule filter to match context")
	}
}

func TestRuleRenderMessageUsesDefaultTemplate(t *testing.T) {
	t.Parallel()

	rule := Rule{
		EventType: RuleEventTypeTicketCreated,
	}

	message, err := rule.RenderMessage(map[string]any{
		"ticket": map[string]any{
			"identifier":  "ASE-69",
			"title":       "Build notification rules",
			"status_name": "Todo",
			"priority":    "high",
		},
		"new_status": "Todo",
	})
	if err != nil {
		t.Fatalf("RenderMessage() error = %v", err)
	}

	if message.Title != "Ticket created: ASE-69" {
		t.Fatalf("RenderMessage() title = %q, want %q", message.Title, "Ticket created: ASE-69")
	}
	if message.Body == "" {
		t.Fatalf("RenderMessage() expected non-empty body")
	}
}

func TestParseCreateRuleRejectsInvalidChannelID(t *testing.T) {
	t.Parallel()

	_, err := ParseCreateRule(uuid.New(), RuleInput{
		Name:      "Ticket Created",
		EventType: RuleEventTypeTicketCreated.String(),
		ChannelID: "not-a-uuid",
	})
	if err == nil {
		t.Fatal("expected ParseCreateRule() to reject invalid channel id")
	}
}
