package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func newRootTicketCommand() *cobra.Command {
	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: defaultCLIHTTPDoer()})
	command.Short = "Operate on tickets through OpenASE."
	command.Long = strings.TrimSpace(command.Long +
		"\n\nAll ticket subcommands use the agent-platform wrapper semantics so agent workspaces can rely on `--status-name` / `--status_name` and `--body-file` / `--body_file` consistently.")

	command.AddCommand(newAgentPlatformTicketArchivedCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformTicketGetCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformTicketDetailCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformTicketRetryResumeCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformTicketDependencyCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformTicketExternalLinkCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformTicketRunCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	return command
}

func newRootProjectCommand() *cobra.Command {
	command := newAgentPlatformProjectCommandWithDeps(platformCommandDeps{httpClient: defaultCLIHTTPDoer()})
	command.Short = "Operate on projects through OpenASE."
	command.Long = strings.TrimSpace(command.Long +
		"\n\nAll project subcommands use the agent-platform wrapper semantics in agent workspaces." +
		"\n\nUse `openase project current` to inspect the active project from `OPENASE_PROJECT_ID`, `openase project updates ...` for curated project updates, or `openase machine list --project-id $OPENASE_PROJECT_ID` to move from project context into machine inspection without raw API fallback.")

	command.AddCommand(newAgentPlatformProjectCurrentCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformProjectUpdatesCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformProjectListCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformProjectGetCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformProjectCreateCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	command.AddCommand(newAgentPlatformProjectDeleteCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()}))
	return command
}

func newRootStatusCommand() *cobra.Command {
	command := newAgentPlatformStatusCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()})
	command.Short = "Operate on ticket statuses through OpenASE."
	command.Long = strings.TrimSpace(command.Long +
		"\n\nAll status subcommands use the agent-platform wrapper semantics so Project AI runtimes can rely on OPENASE_API_URL, OPENASE_AGENT_TOKEN, and OPENASE_PROJECT_ID consistently.")
	return command
}

func newRootWorkflowCommand() *cobra.Command {
	command := newAgentPlatformWorkflowCommandWithDeps(apiCommandDeps{httpClient: defaultCLIHTTPDoer()})
	command.Short = "Operate on workflows through OpenASE."
	command.Long = strings.TrimSpace(command.Long +
		"\n\nAll workflow and workflow harness subcommands use the agent-platform wrapper semantics so Project AI runtimes keep platform-aware routing, authentication, and project defaults aligned with `openase ticket` and `openase project`.")
	return command
}
