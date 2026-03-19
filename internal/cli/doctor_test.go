package cli

import "testing"

func TestNewRootCommandIncludesDoctor(t *testing.T) {
	root := NewRootCommand("dev")

	command, _, err := root.Find([]string{"doctor"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected doctor command")
	}
	if command.Use != "doctor" {
		t.Fatalf("expected doctor command, got %q", command.Use)
	}
}
