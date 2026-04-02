package workflow

import "testing"

func TestParseType(t *testing.T) {
	got, err := ParseType(" Refine-Harness ")
	if err != nil {
		t.Fatalf("ParseType() error = %v", err)
	}
	if got != TypeRefineHarness {
		t.Fatalf("ParseType() = %q", got)
	}
	if got.String() != "refine-harness" {
		t.Fatalf("Type.String() = %q", got.String())
	}

	if _, err := ParseType("review"); err == nil {
		t.Fatal("expected invalid workflow type to fail")
	}
}
