package workflow

import "testing"

func TestParseType(t *testing.T) {
	got, err := ParseType(" Refine-Harness ")
	if err != nil {
		t.Fatalf("ParseType() error = %v", err)
	}
	if got != TypeLabel("Refine-Harness") {
		t.Fatalf("ParseType() = %q", got)
	}
	if got.String() != "Refine-Harness" {
		t.Fatalf("Type.String() = %q", got.String())
	}
	if got.NormalizedKey() != "refineharness" {
		t.Fatalf("Type.NormalizedKey() = %q", got.NormalizedKey())
	}

	if _, err := ParseType("  "); err == nil {
		t.Fatal("expected empty workflow type to fail")
	}
	if _, err := ParseType("line1\nline2"); err == nil {
		t.Fatal("expected control characters to fail")
	}
}
