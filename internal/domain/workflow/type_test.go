package workflow

import (
	"strings"
	"testing"
)

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
	if _, err := ParseTypeLabel(strings.Repeat("a", maxTypeLabelRunes+1)); err == nil {
		t.Fatal("expected too-long workflow type to fail")
	}
}

func TestMustParseTypeLabelPanicsOnInvalidInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected MustParseTypeLabel to panic")
		}
	}()

	_ = MustParseTypeLabel("")
}

func TestNormalizeSemanticKeyKeepsNonControlMarks(t *testing.T) {
	label := MustParseTypeLabel(" A\u0301 + B ")
	if got := label.NormalizedKey(); got != "a\u0301b" {
		t.Fatalf("NormalizedKey() = %q", got)
	}
}
