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
