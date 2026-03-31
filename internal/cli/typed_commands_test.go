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

func TestRawAPIHelpMentionsPassthroughInputs(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"api"})
	if err != nil {
		t.Fatalf("Find(api) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected api command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"raw passthrough CLI entrypoint",
		"--input cannot be combined with body fields",
		"openase api GET /api/v1/tickets/$OPENASE_TICKET_ID",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestWatchTicketsHelpMentionsStreamingSemantics(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"watch", "tickets"})
	if err != nil {
		t.Fatalf("Find(watch tickets) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected watch tickets command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"keeps the connection open",
		"Use Ctrl-C to stop the stream",
		"projectId must be UUID values",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestTypedTicketWorkpadHelpMentionsUpsertSemantics(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"ticket", "comment", "workpad"})
	if err != nil {
		t.Fatalf("Find(ticket comment workpad) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected ticket comment workpad command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"lists comments on the target ticket",
		"creates or updates that single comment",
		"Exactly one of --body or --body-file must be provided",
		"Human-readable identifiers such as ASE-2 are not",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}
