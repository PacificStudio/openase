package activityevent

import (
	"io"
	"log/slog"
	"testing"
)

func TestCatalogHelpers(t *testing.T) {
	if got := TypeAgentReady.String(); got != "agent.ready" {
		t.Fatalf("Type.String() = %q", got)
	}
	if !TypeHookFailed.IsHook() {
		t.Fatal("expected hook event to be recognized")
	}
	if TypeAgentReady.IsHook() {
		t.Fatal("expected non-hook event to stay non-hook")
	}
	if !TypeTicketCommentEdited.IsTicketComment() {
		t.Fatal("expected ticket comment event to be recognized")
	}
	if TypeAgentReady.IsTicketComment() {
		t.Fatal("expected non-ticket-comment event to stay non-ticket-comment")
	}

	catalog := Catalog()
	if len(catalog) == 0 {
		t.Fatal("expected canonical catalog entries")
	}
	if MustParseType("ticket.created") != TypeTicketCreated {
		t.Fatal("expected ticket.created to remain canonical")
	}
	if _, err := ParseRawType("project.created"); err != nil {
		t.Fatalf("expected project.created to be supported: %v", err)
	}
	if _, err := ParseRawType("ticket_comment.created"); err != nil {
		t.Fatalf("expected ticket_comment.created to be supported: %v", err)
	}

	catalog[0].Label = "mutated"
	refreshed := Catalog()
	if refreshed[0].Label == "mutated" {
		t.Fatal("expected Catalog() to return a defensive copy")
	}
}

func TestParseRawType(t *testing.T) {
	eventType, err := ParseRawType(" agent.ready ")
	if err != nil {
		t.Fatalf("ParseRawType() error = %v", err)
	}
	if eventType != TypeAgentReady {
		t.Fatalf("ParseRawType() = %q", eventType)
	}

	if _, err := ParseRawType("agent.heartbeat"); err == nil {
		t.Fatal("expected unsupported activity event type to fail")
	}
}

func TestMustParseType(t *testing.T) {
	if got := MustParseType("hook.failed"); got != TypeHookFailed {
		t.Fatalf("MustParseType() = %q", got)
	}
}

func TestMustParseTypePanicsForUnsupportedValue(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected MustParseType to panic for unsupported activity event type")
		}
	}()

	_ = MustParseType("agent.heartbeat")
}

func TestParseStoredType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	eventType, raw := ParseStoredType("hook.failed", logger)
	if eventType != TypeHookFailed || raw != "" {
		t.Fatalf("ParseStoredType(valid) = (%q, %q)", eventType, raw)
	}

	previousDefault := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() {
		slog.SetDefault(previousDefault)
	})

	eventType, raw = ParseStoredType("agent.heartbeat", nil)
	if eventType != TypeUnknown || raw != "agent.heartbeat" {
		t.Fatalf("ParseStoredType(unknown) = (%q, %q)", eventType, raw)
	}
}
