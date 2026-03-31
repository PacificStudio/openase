package chatconversation

import "testing"

func TestParseSource(t *testing.T) {
	t.Parallel()

	source, err := ParseSource("  project_sidebar  ")
	if err != nil {
		t.Fatalf("ParseSource() error = %v", err)
	}
	if source != SourceProjectSidebar {
		t.Fatalf("ParseSource() = %q, want %q", source, SourceProjectSidebar)
	}
}

func TestParseSourceRejectsUnsupportedValues(t *testing.T) {
	t.Parallel()

	if _, err := ParseSource("ticket_detail"); err == nil {
		t.Fatal("ParseSource() error = nil, want error")
	}
}

func TestDomainErrorsAreStableSentinels(t *testing.T) {
	t.Parallel()

	if ErrNotFound == nil || ErrConflict == nil || ErrInvalidInput == nil {
		t.Fatal("expected domain error sentinels to be initialized")
	}
	if ErrNotFound == ErrConflict || ErrConflict == ErrInvalidInput || ErrNotFound == ErrInvalidInput {
		t.Fatal("expected distinct domain error sentinels")
	}
}
