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

func TestWatchProjectHelpMentionsStreamingSemantics(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"watch", "project"})
	if err != nil {
		t.Fatalf("Find(watch project) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected watch project command")
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
		"single stream entrypoint",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestStreamEventsHelpMentionsOperatorObservation(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"stream", "events"})
	if err != nil {
		t.Fatalf("Find(stream events) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected stream events command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"first-class stream entrypoint",
		"Machine and provider lifecycle updates flow through the global event stream",
		"openase stream events",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestMachineRefreshHealthHelpMentionsHealthRefreshSemantics(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"machine", "refresh-health"})
	if err != nil {
		t.Fatalf("Find(machine refresh-health) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected machine refresh-health command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"re-runs the machine health collector",
		"provider availability can be observed from refreshed data",
		"openase machine refresh-health",
		"machineId must be UUID values",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestProviderGetHelpMentionsAvailabilitySemantics(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"provider", "get"})
	if err != nil {
		t.Fatalf("Find(provider get) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected provider get command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"availability_state",
		"backing machine metadata",
		"openase provider get",
		"providerId must be UUID values",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestTypedTicketCommentCommandExposesPrimitiveSubcommandsOnly(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"ticket", "comment"})
	if err != nil {
		t.Fatalf("Find(ticket comment) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected ticket comment command")
	}
	names := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		names = append(names, child.Name())
	}
	if strings.Join(names, ",") != "create,delete,list,revisions,update" {
		t.Fatalf("comment subcommands = %v, want [create delete list revisions update]", names)
	}
}

func TestStatusListHelpMentionsStatusBoardSemantics(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"status", "list"})
	if err != nil {
		t.Fatalf("Find(status list) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected status list command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"ordered status board",
		"concurrency limits",
		"projectId must be UUID values",
		"openase status list",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestStatusCommandExposesCRUDAndResetSubcommands(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"status"})
	if err != nil {
		t.Fatalf("Find(status) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected status command")
	}

	names := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		names = append(names, child.Name())
	}
	if strings.Join(names, ",") != "create,delete,list,reset,update" {
		t.Fatalf("status subcommands = %v, want [create delete list reset update]", names)
	}
}

func TestChatConversationCommandExposesLifecycleSubcommands(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"chat", "conversation"})
	if err != nil {
		t.Fatalf("Find(chat conversation) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected chat conversation command")
	}

	names := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		names = append(names, child.Name())
	}
	if strings.Join(names, ",") != "close-runtime,create,entries,get,list,respond-interrupt,turn,watch,workspace-diff" {
		t.Fatalf("chat conversation subcommands = %v", names)
	}
}

func TestRepoCommandExposesRepoGitHubAndScopeSubcommands(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"repo"})
	if err != nil {
		t.Fatalf("Find(repo) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected repo command")
	}

	names := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		names = append(names, child.Name())
	}
	if strings.Join(names, ",") != "create,delete,github,list,scope,update" {
		t.Fatalf("repo subcommands = %v", names)
	}
}

func TestChannelTestHelpMentionsDeliveryVerification(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"channel", "test"})
	if err != nil {
		t.Fatalf("Find(channel test) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected channel test command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"verify the destination accepts deliveries",
		"channelId must be UUID values",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestNotificationRuleCommandExposesEventTypesAndCRUDSubcommands(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"notification-rule"})
	if err != nil {
		t.Fatalf("Find(notification-rule) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected notification-rule command")
	}

	names := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		names = append(names, child.Name())
	}
	if strings.Join(names, ",") != "create,delete,list,update" {
		t.Fatalf("notification-rule subcommands = %v", names)
	}
}
