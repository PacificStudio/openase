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

	catalog := Catalog()
	if len(catalog) == 0 {
		t.Fatal("expected canonical catalog entries")
	}
	if catalog[0].EventType != TypeTicketCreated {
		t.Fatalf("unexpected first catalog entry: %+v", catalog[0])
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
