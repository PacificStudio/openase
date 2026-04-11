package cli

import (
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func newAgentPlatformOpenAPIOperationCommandWithDeps(spec openAPICommandSpec, deps apiCommandDeps) *cobra.Command {
	contract := mustOpenAPICommandContract(spec)
	command := &cobra.Command{
		Use:     spec.Use,
		Short:   contract.summary,
		Long:    buildOpenAPIOperationHelp(spec, contract.summary),
		Example: strings.TrimSpace(spec.Example),
		Args:    cobra.MaximumNArgs(len(spec.PositionalParams)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentPlatformOpenAPIOperationCommand(cmd, deps, contract, args)
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	registerOpenAPICommandFlags(command.Flags(), contract)
	return markCLICommandAPICoverageSpec(command, spec)
}

func newAgentPlatformNestedOpenAPIOperationCommandWithDeps(spec openAPICommandSpec, deps apiCommandDeps) *cobra.Command {
	contract := mustOpenAPICommandContract(spec)
	command := &cobra.Command{
		Use:     spec.Use,
		Short:   contract.summary,
		Long:    buildOpenAPIOperationHelp(spec, contract.summary),
		Example: strings.TrimSpace(spec.Example),
		Args:    cobra.MaximumNArgs(len(spec.PositionalParams)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentPlatformOpenAPIOperationCommand(cmd, deps, contract, args)
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	registerOpenAPIOperationFlags(command.Flags(), contract)
	return markCLICommandAPICoverageSpec(command, spec)
}

func newAgentPlatformTicketDependencyCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	command := &cobra.Command{
		Use:   "dependency",
		Short: "Operate on ticket dependency relationships.",
	}
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "add [ticketId]",
		Short:            "Add a ticket dependency.",
		Method:           http.MethodPost,
		Path:             "/api/v1/tickets/{ticketId}/dependencies",
		PositionalParams: []string{"ticketId"},
		HelpNotes: []string{
			"Use --type blocks or --type blocked_by to express blocker relationships. Existing dependencies are edited by deleting and recreating them because the API does not expose a patch operation.",
		},
		Example: strings.TrimSpace(`
  openase ticket dependency add $OPENASE_TICKET_ID --type blocked_by --target-ticket-id $BLOCKER_TICKET_ID
  openase ticket dependency add $OPENASE_TICKET_ID --type blocks --target-ticket-id $BLOCKED_TICKET_ID
`),
	}, deps))
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "delete [ticketId] [dependencyId]",
		Short:            "Delete a ticket dependency.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/tickets/{ticketId}/dependencies/{dependencyId}",
		PositionalParams: []string{"ticketId", "dependencyId"},
	}, deps))
	return command
}

func newAgentPlatformTicketArchivedCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	return newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "archived [projectId]",
		Short:            "List archived tickets.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/tickets/archived",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"Use this to audit tickets that left the active board without falling back to raw API calls.",
		},
		Example: "openase ticket archived $OPENASE_PROJECT_ID",
	}, deps)
}

func newAgentPlatformTicketGetCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	return newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "get [ticketId]",
		Short:            "Get a ticket.",
		Method:           http.MethodGet,
		Path:             "/api/v1/tickets/{ticketId}",
		PositionalParams: []string{"ticketId"},
	}, deps)
}

func newAgentPlatformTicketDetailCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	return newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "detail [projectId] [ticketId]",
		Short:            "Get ticket detail.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/tickets/{ticketId}/detail",
		PositionalParams: []string{"projectId", "ticketId"},
	}, deps)
}

func newAgentPlatformTicketRetryResumeCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	return newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "retry-resume [ticketId]",
		Short:            "Resume a ticket after a retryable failure.",
		Method:           http.MethodPost,
		Path:             "/api/v1/tickets/{ticketId}/retry/resume",
		PositionalParams: []string{"ticketId"},
		HelpNotes: []string{
			"This requests a fresh retry attempt for a ticket whose last run stopped in a resumable retry state.",
		},
		Example: "openase ticket retry-resume $OPENASE_TICKET_ID",
	}, deps)
}

func newAgentPlatformTicketExternalLinkCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	command := &cobra.Command{
		Use:   "external-link",
		Short: "Operate on ticket external links.",
	}
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "add [ticketId]",
		Short:            "Add a ticket external link.",
		Method:           http.MethodPost,
		Path:             "/api/v1/tickets/{ticketId}/external-links",
		PositionalParams: []string{"ticketId"},
		HelpNotes: []string{
			"Use this to attach upstream issue, incident, document, or pull request references to a ticket.",
		},
		Example: `openase ticket external-link add $OPENASE_TICKET_ID --title "PR 482" --url https://github.com/acme/repo/pull/482`,
	}, deps))
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "delete [ticketId] [externalLinkId]",
		Short:            "Delete a ticket external link.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/tickets/{ticketId}/external-links/{externalLinkId}",
		PositionalParams: []string{"ticketId", "externalLinkId"},
	}, deps))
	return command
}

func newAgentPlatformTicketRunCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	command := &cobra.Command{
		Use:   "run",
		Short: "Inspect ticket run history.",
	}
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "list [projectId] [ticketId]",
		Short:            "List ticket runs.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/tickets/{ticketId}/runs",
		PositionalParams: []string{"projectId", "ticketId"},
		HelpNotes: []string{
			"Use this to inspect execution history, retry chains, and current runtime state for one ticket.",
		},
		Example: "openase ticket run list $OPENASE_PROJECT_ID $OPENASE_TICKET_ID",
	}, deps))
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "get [projectId] [ticketId] [runId]",
		Short:            "Get a ticket run.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/tickets/{ticketId}/runs/{runId}",
		PositionalParams: []string{"projectId", "ticketId", "runId"},
		HelpNotes: []string{
			"This returns the stored runtime snapshot for one run, including status, lifecycle timestamps, and retry metadata.",
		},
		Example: "openase ticket run get $OPENASE_PROJECT_ID $OPENASE_TICKET_ID $OPENASE_RUN_ID",
	}, deps))
	return command
}

func newAgentPlatformProjectCurrentCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	return newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "current [projectId]",
		Short:            "Get the current project.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"This command resolves the target project from [projectId], --project-id, and then OPENASE_PROJECT_ID so project-scoped runtimes can inspect the active project without falling back to raw API calls.",
			"The response includes organization_id, which can be used directly or bridged into machine inspection with `openase machine list --project-id $OPENASE_PROJECT_ID`.",
		},
		Example: strings.TrimSpace(`
  openase project current
  openase project current --project-id $OPENASE_PROJECT_ID --json project.organization_id
`),
	}, deps)
}

func newAgentPlatformProjectListCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	return newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "list [orgId]",
		Short:            "List projects.",
		Method:           http.MethodGet,
		Path:             "/api/v1/orgs/{orgId}/projects",
		PositionalParams: []string{"orgId"},
	}, deps)
}

func newAgentPlatformProjectGetCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	return newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "get [projectId]",
		Short:            "Get a project.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}",
		PositionalParams: []string{"projectId"},
	}, deps)
}

func newAgentPlatformProjectCreateCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	return newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "create [orgId]",
		Short:            "Create a project.",
		Method:           http.MethodPost,
		Path:             "/api/v1/orgs/{orgId}/projects",
		PositionalParams: []string{"orgId"},
	}, deps)
}

func newAgentPlatformProjectDeleteCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	return newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "delete [projectId]",
		Short:            "Archive a project.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/projects/{projectId}",
		PositionalParams: []string{"projectId"},
	}, deps)
}

func newAgentPlatformProjectUpdatesCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	command := &cobra.Command{
		Use:   "updates",
		Short: "Operate on project update threads.",
		Long: strings.TrimSpace(`
Operate on project update threads.

These first-class commands expose the curated project updates surface without
falling back to raw ` + "`openase api`" + ` calls.

Project scope defaults to [projectId], then --project-id, then OPENASE_PROJECT_ID.
When a command also needs [threadId], you can pass --thread-id while letting
project scope fall back to OPENASE_PROJECT_ID.
`),
	}
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "list [projectId]",
		Short:            "List project update threads.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/updates",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"If [projectId] is omitted, the command falls back to --project-id and then OPENASE_PROJECT_ID.",
		},
		Example: "openase project updates list\nopenase project updates list $OPENASE_PROJECT_ID",
	}, deps))
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "create [projectId]",
		Short:            "Create a project update thread.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/updates",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"If [projectId] is omitted, the command falls back to --project-id and then OPENASE_PROJECT_ID.",
		},
		Example: "openase project updates create --status on_track --body \"Implemented runtime-safe project and machine commands.\"",
	}, deps))
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "update [projectId] [threadId]",
		Short:            "Update a project update thread.",
		Method:           http.MethodPatch,
		Path:             "/api/v1/projects/{projectId}/updates/{threadId}",
		PositionalParams: []string{"projectId", "threadId"},
		HelpNotes: []string{
			"You can pass --thread-id together with OPENASE_PROJECT_ID when you only want to provide the thread identifier explicitly.",
		},
		Example: "openase project updates update --thread-id $OPENASE_THREAD_ID --status at_risk --body \"Waiting on review.\"",
	}, deps))
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "delete [projectId] [threadId]",
		Short:            "Delete a project update thread.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/projects/{projectId}/updates/{threadId}",
		PositionalParams: []string{"projectId", "threadId"},
		HelpNotes: []string{
			"You can pass --thread-id together with OPENASE_PROJECT_ID when you only want to provide the thread identifier explicitly.",
		},
		Example: "openase project updates delete --thread-id $OPENASE_THREAD_ID",
	}, deps))
	command.AddCommand(newAgentPlatformNestedOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "revisions [projectId] [threadId]",
		Short:            "List project update thread revisions.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/updates/{threadId}/revisions",
		PositionalParams: []string{"projectId", "threadId"},
		HelpNotes: []string{
			"You can pass --thread-id together with OPENASE_PROJECT_ID when you only want to provide the thread identifier explicitly.",
		},
		Example: "openase project updates revisions --thread-id $OPENASE_THREAD_ID",
	}, deps))
	return command
}

func runAgentPlatformOpenAPIOperationCommand(cmd *cobra.Command, deps apiCommandDeps, contract openAPICommandContract, args []string) error {
	// Platform-aware wrappers must preserve `/api/v1/platform` when the runtime
	// injects it through OPENASE_API_URL.
	apiContext, err := apiOptionsFromFlags(cmd.Flags()).resolve()
	if err != nil {
		return err
	}
	requestPath, err := resolveOpenAPIRequestPath(cmd, contract, args)
	if err != nil {
		return err
	}
	body, err := resolveOpenAPIBody(cmd, contract)
	if err != nil {
		return err
	}
	response, err := apiContext.do(cmd.Context(), deps, apiRequest{
		Method: contract.spec.Method,
		Path:   requestPath,
		Body:   body,
	})
	if err != nil {
		return err
	}
	return writeAPIOutput(cmd.OutOrStdout(), response.Body, outputOptionsFromFlags(cmd.Flags()))
}

func newAgentPlatformStatusCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	command := &cobra.Command{
		Use:   "status",
		Short: "Operate on ticket statuses through the agent platform API.",
		Long: buildPlatformCommandHelp("Operate on ticket statuses through the agent platform API.", platformHelpSpec{
			projectScope: true,
			examples: []string{
				"openase status list",
				"openase status create --name \"QA\" --stage started --color \"#FF00AA\"",
				"openase status reset",
			},
			notes: []string{
				"Project-scoped status commands also accept a positional [projectId]; if omitted they fall back to --project-id and then OPENASE_PROJECT_ID.",
			},
		}),
	}
	command.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "list [projectId]",
		Short:            "List ticket statuses.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/statuses",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"Use this to inspect the ordered status board for a project, including default status selection, status stage, and concurrency limits.",
		},
		Example: strings.TrimSpace(`
  openase status list $OPENASE_PROJECT_ID
  openase status list 550e8400-e29b-41d4-a716-446655440000 --json statuses
`),
	}, deps))
	command.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "create [projectId]",
		Short:            "Create a ticket status.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/statuses",
		PositionalParams: []string{"projectId"},
		Example:          `openase status create $OPENASE_PROJECT_ID --name "QA" --stage started --color "#FF00AA" --description "Quality gate"`,
	}, deps))
	command.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "update [statusId]",
		Short:            "Update a ticket status.",
		Method:           http.MethodPatch,
		Path:             "/api/v1/statuses/{statusId}",
		PositionalParams: []string{"statusId"},
		Example:          `openase status update $OPENASE_STATUS_ID --name "Ready for QA" --stage completed --position 5`,
	}, deps))
	command.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "delete [statusId]",
		Short:            "Delete a ticket status.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/statuses/{statusId}",
		PositionalParams: []string{"statusId"},
		Example:          "openase status delete $OPENASE_STATUS_ID",
	}, deps))
	command.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "reset [projectId]",
		Short:            "Reset project statuses to the default template.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/statuses/reset",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"This replaces the project's status board with the built-in default template and should be treated as an administrative operation.",
		},
		Example: "openase status reset $OPENASE_PROJECT_ID",
	}, deps))
	return command
}

func newAgentPlatformWorkflowCommandWithDeps(deps apiCommandDeps) *cobra.Command {
	command := &cobra.Command{
		Use:   "workflow",
		Short: "Operate on workflows through the agent platform API.",
		Long: buildPlatformCommandHelp("Operate on workflows through the agent platform API.", platformHelpSpec{
			projectScope: true,
			examples: []string{
				"openase workflow list",
				"openase workflow get $OPENASE_WORKFLOW_ID",
				"openase workflow harness variables",
			},
			notes: []string{
				"Project-scoped workflow commands also accept a positional [projectId]; if omitted they fall back to --project-id and then OPENASE_PROJECT_ID.",
				"Workflow harness subcommands keep the same platform-aware API URL and token defaults in Project AI runtimes.",
			},
		}),
	}
	command.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "list [projectId]",
		Short:            "List workflows.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/workflows",
		PositionalParams: []string{"projectId"},
	}, deps))
	command.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "get [workflowId]",
		Short:            "Get a workflow.",
		Method:           http.MethodGet,
		Path:             "/api/v1/workflows/{workflowId}",
		PositionalParams: []string{"workflowId"},
	}, deps))
	command.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "create [projectId]",
		Short:            "Create a workflow.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/workflows",
		PositionalParams: []string{"projectId"},
	}, deps))
	command.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "update [workflowId]",
		Short:            "Update a workflow.",
		Method:           http.MethodPatch,
		Path:             "/api/v1/workflows/{workflowId}",
		PositionalParams: []string{"workflowId"},
	}, deps))
	command.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "delete [workflowId]",
		Short:            "Delete a workflow.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/workflows/{workflowId}",
		PositionalParams: []string{"workflowId"},
	}, deps))

	harness := &cobra.Command{
		Use:   "harness",
		Short: "Operate on workflow harness content.",
		Long: buildPlatformCommandHelp("Operate on workflow harness content.", platformHelpSpec{
			examples: []string{
				"openase workflow harness get $OPENASE_WORKFLOW_ID",
				"openase workflow harness history $OPENASE_WORKFLOW_ID",
				"openase workflow harness variables",
			},
			notes: []string{
				"These subcommands preserve the runtime-provided platform API URL and token so Project AI can call the agent platform harness routes directly.",
			},
		}),
	}
	harness.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "get [workflowId]",
		Short:            "Get workflow harness content.",
		Method:           http.MethodGet,
		Path:             "/api/v1/workflows/{workflowId}/harness",
		PositionalParams: []string{"workflowId"},
	}, deps))
	harness.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "history [workflowId]",
		Short:            "List workflow harness revisions.",
		Method:           http.MethodGet,
		Path:             "/api/v1/workflows/{workflowId}/harness/history",
		PositionalParams: []string{"workflowId"},
		HelpNotes: []string{
			"Use this to audit harness edits and recover the exact stored revision sequence for one workflow.",
		},
		Example: "openase workflow harness history $OPENASE_WORKFLOW_ID",
	}, deps))
	harness.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:              "update [workflowId]",
		Short:            "Update workflow harness content.",
		Method:           http.MethodPut,
		Path:             "/api/v1/workflows/{workflowId}/harness",
		PositionalParams: []string{"workflowId"},
	}, deps))
	harness.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:    "variables",
		Short:  "List harness variables.",
		Method: http.MethodGet,
		Path:   "/api/v1/harness/variables",
		HelpNotes: []string{
			"Use this to inspect the variable catalog available to workflow harness templates before editing or validating one.",
		},
		Example: "openase workflow harness variables",
	}, deps))
	harness.AddCommand(newAgentPlatformOpenAPIOperationCommandWithDeps(openAPICommandSpec{
		Use:    "validate",
		Short:  "Validate harness content.",
		Method: http.MethodPost,
		Path:   "/api/v1/harness/validate",
		HelpNotes: []string{
			"This validates harness markdown and structured references without mutating any stored workflow harness.",
		},
		Example: "openase workflow harness validate --input /tmp/harness.json",
	}, deps))
	command.AddCommand(harness)
	return command
}
