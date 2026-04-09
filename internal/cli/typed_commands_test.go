package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootTicketUpdateHelpClarifiesPlatformUUIDSemantics(t *testing.T) {
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
		"OPENASE_TICKET_ID",
		"At least one update field must be provided.",
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

func TestProjectCurrentHelpMentionsProjectContextBridge(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"project", "current"})
	if err != nil {
		t.Fatalf("Find(project current) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected project current command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"OPENASE_PROJECT_ID",
		"organization_id",
		"openase machine list --project-id $OPENASE_PROJECT_ID",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestProjectUpdatesHelpMentionsRuntimeDefaults(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"project", "updates", "list"})
	if err != nil {
		t.Fatalf("Find(project updates list) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected project updates list command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"OPENASE_PROJECT_ID",
		"openase project updates list",
		"List project update threads",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestSecretBindingCreateHelpMentionsScopedRuntimeBindings(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"secret", "binding", "create"})
	if err != nil {
		t.Fatalf("Find(secret binding create) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected secret binding create command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"Binding scope is limited to workflow or ticket",
		"openase secret binding create",
		"projectId must be UUID values",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestMachineListHelpMentionsProjectAwareBridge(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"machine", "list"})
	if err != nil {
		t.Fatalf("Find(machine list) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected machine list command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"OPENASE_PROJECT_ID",
		"--project-id",
		"organization_id automatically",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestMachineStreamHelpMentionsProjectAwareBridge(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"machine", "stream"})
	if err != nil {
		t.Fatalf("Find(machine stream) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected machine stream command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"OPENASE_PROJECT_ID",
		"Use Ctrl-C to stop the stream",
		"openase machine stream --project-id $OPENASE_PROJECT_ID",
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
	if strings.Join(names, ",") != "create,list,update" {
		t.Fatalf("comment subcommands = %v, want [create list update]", names)
	}
}

func TestTypedTicketDependencyCommandExposesPrimitiveSubcommandsOnly(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"ticket", "dependency"})
	if err != nil {
		t.Fatalf("Find(ticket dependency) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected ticket dependency command")
	}
	names := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		names = append(names, child.Name())
	}
	if strings.Join(names, ",") != "add,delete" {
		t.Fatalf("dependency subcommands = %v, want [add delete]", names)
	}
}

func TestTypedTicketDependencyAddHelpMentionsBlockerSemantics(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"ticket", "dependency", "add"})
	if err != nil {
		t.Fatalf("Find(ticket dependency add) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected ticket dependency add command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"--type blocks or --type blocked_by",
		"API does not expose a patch operation",
		"--target-ticket-id",
		"ticketId must be UUID values",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestActivityListHelpMentionsTimelineSemantics(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"activity", "list"})
	if err != nil {
		t.Fatalf("Find(activity list) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected activity list command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"project event timeline",
		"runtime activity",
		"projectId must be UUID values",
		"openase activity list",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestTicketCommandExposesRunRetryAndExternalLinkSubcommands(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"ticket"})
	if err != nil {
		t.Fatalf("Find(ticket) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected ticket command")
	}

	names := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		names = append(names, child.Name())
	}
	if strings.Join(names, ",") != "archived,comment,create,dependency,detail,external-link,get,list,report-usage,retry-resume,run,update" {
		t.Fatalf("ticket subcommands = %v", names)
	}
}

func TestTicketRunCommandExposesListAndGetSubcommands(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"ticket", "run"})
	if err != nil {
		t.Fatalf("Find(ticket run) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected ticket run command")
	}

	names := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		names = append(names, child.Name())
	}
	if strings.Join(names, ",") != "get,list" {
		t.Fatalf("ticket run subcommands = %v", names)
	}
}

func TestTicketRetryResumeHelpMentionsRetryableFailureSemantics(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"ticket", "retry-resume"})
	if err != nil {
		t.Fatalf("Find(ticket retry-resume) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected ticket retry-resume command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"fresh retry attempt",
		"resumable retry state",
		"ticketId must be UUID values",
		"openase ticket retry-resume",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
	}
}

func TestWorkflowHarnessCommandExposesHistoryVariablesAndValidateSubcommands(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"workflow", "harness"})
	if err != nil {
		t.Fatalf("Find(workflow harness) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected workflow harness command")
	}

	names := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		names = append(names, child.Name())
	}
	if strings.Join(names, ",") != "get,history,update,validate,variables" {
		t.Fatalf("workflow harness subcommands = %v", names)
	}
}

func TestWorkflowHarnessVariablesHelpMentionsVariableCatalog(t *testing.T) {
	root := NewRootCommand("dev")
	command, _, err := root.Find([]string{"workflow", "harness", "variables"})
	if err != nil {
		t.Fatalf("Find(workflow harness variables) returned error: %v", err)
	}
	if command == nil {
		t.Fatal("expected workflow harness variables command")
	}

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	if err := command.Help(); err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"variable catalog",
		"before editing or validating",
		"openase workflow harness variables",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q, got %q", want, output)
		}
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
