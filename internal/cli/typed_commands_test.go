package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestTypedTicketUpdateHelpClarifiesUUIDSemantics(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"ticket", "update"})
	if err != nil {
		t.Fatalf("Find(ticket update) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected ticket update command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"ticketId must be UUID values",
		"Human-readable identifiers such as ASE-2 are not accepted",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}
