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
		"\n\nShared ticket mutations (`list`, `create`, `update`, `comment`, `report-usage`) use the agent-platform wrapper semantics so agent workspaces can rely on `--status-name` / `--status_name` and `--body-file` / `--body_file` consistently." +
		"\n\nExtended inspection and history subcommands (`archived`, `get`, `detail`, `retry-resume`, `dependency`, `external-link`, `run`) remain direct OpenAPI operations.")

	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "archived [projectId]",
		Short:            "List archived tickets.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/tickets/archived",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"Use this to audit tickets that left the active board without falling back to raw API calls.",
		},
		Example: "openase ticket archived $OPENASE_PROJECT_ID",
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [ticketId]", Short: "Get a ticket.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}", PositionalParams: []string{"ticketId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "detail [projectId] [ticketId]", Short: "Get ticket detail.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets/{ticketId}/detail", PositionalParams: []string{"projectId", "ticketId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "retry-resume [ticketId]",
		Short:            "Resume a ticket after a retryable failure.",
		Method:           http.MethodPost,
		Path:             "/api/v1/tickets/{ticketId}/retry/resume",
		PositionalParams: []string{"ticketId"},
		HelpNotes: []string{
			"This requests a fresh retry attempt for a ticket whose last run stopped in a resumable retry state.",
		},
		Example: "openase ticket retry-resume $OPENASE_TICKET_ID",
	}))
	command.AddCommand(newTypedTicketDependencyCommand())
	command.AddCommand(newTypedTicketExternalLinkCommand())
	command.AddCommand(newTypedTicketRunCommand())
	return command
}

func newRootProjectCommand() *cobra.Command {
	command := newAgentPlatformProjectCommandWithDeps(platformCommandDeps{httpClient: http.DefaultClient})
	command.Short = "Operate on projects through OpenASE."
	command.Long = strings.TrimSpace(command.Long +
		"\n\nShared project mutations (`update` and `add-repo`) use the agent-platform wrapper semantics in agent workspaces. The remaining subcommands (`current`, `updates`, `list`, `get`, `create`, `delete`) remain direct OpenAPI operations." +
		"\n\nUse `openase project current` to inspect the active project from `OPENASE_PROJECT_ID`, `openase project updates ...` for curated project updates, or `openase machine list --project-id $OPENASE_PROJECT_ID` to move from project context into machine inspection without raw API fallback.")

	command.AddCommand(newProjectCurrentCommand())
	command.AddCommand(newProjectUpdatesCommand())
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [orgId]", Short: "List projects.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/projects", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [projectId]", Short: "Get a project.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [orgId]", Short: "Create a project.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/projects", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [projectId]", Short: "Archive a project.", Method: http.MethodDelete, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}}))
	return command
}
