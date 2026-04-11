package cli

import (
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func newRootTicketCommand() *cobra.Command {
	command := newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: http.DefaultClient})
	command.Short = "Operate on tickets through OpenASE."
	command.Long = strings.TrimSpace(command.Long +
		"\n\nAll ticket subcommands use the agent-platform wrapper semantics so agent workspaces can rely on `--status-name` / `--status_name` and `--body-file` / `--body_file` consistently.")

	command.AddCommand(newAgentPlatformTicketArchivedCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformTicketGetCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformTicketDetailCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformTicketRetryResumeCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformTicketDependencyCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformTicketExternalLinkCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformTicketRunCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	return command
}

func newRootProjectCommand() *cobra.Command {
	command := newAgentPlatformProjectCommandWithDeps(platformCommandDeps{httpClient: http.DefaultClient})
	command.Short = "Operate on projects through OpenASE."
	command.Long = strings.TrimSpace(command.Long +
		"\n\nAll project subcommands use the agent-platform wrapper semantics in agent workspaces." +
		"\n\nUse `openase project current` to inspect the active project from `OPENASE_PROJECT_ID`, `openase project updates ...` for curated project updates, or `openase machine list --project-id $OPENASE_PROJECT_ID` to move from project context into machine inspection without raw API fallback.")

	command.AddCommand(newAgentPlatformProjectCurrentCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformProjectUpdatesCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformProjectListCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformProjectGetCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformProjectCreateCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	command.AddCommand(newAgentPlatformProjectDeleteCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient}))
	return command
}

func newRootStatusCommand() *cobra.Command {
	command := newAgentPlatformStatusCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient})
	command.Short = "Operate on ticket statuses through OpenASE."
	command.Long = strings.TrimSpace(command.Long +
		"\n\nAll status subcommands use the agent-platform wrapper semantics so Project AI runtimes can rely on OPENASE_API_URL, OPENASE_AGENT_TOKEN, and OPENASE_PROJECT_ID consistently.")
	return command
}

func newRootWorkflowCommand() *cobra.Command {
	command := newAgentPlatformWorkflowCommandWithDeps(apiCommandDeps{httpClient: http.DefaultClient})
	command.Short = "Operate on workflows through OpenASE."
	command.Long = strings.TrimSpace(command.Long +
		"\n\nAll workflow and workflow harness subcommands use the agent-platform wrapper semantics so Project AI runtimes keep platform-aware routing, authentication, and project defaults aligned with `openase ticket` and `openase project`.")
	return command
}
