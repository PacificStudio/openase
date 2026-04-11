package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type platformHTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type platformCommandDeps struct {
	httpClient platformHTTPDoer
}

type rawPlatformContext struct {
	apiURL    string
	token     string
	projectID string
	ticketID  string
	scopes    []string
}

type platformContext struct {
	apiURL    string
	token     string
	projectID string
	ticketID  string
	scopes    []string
}

type ticketCommandOptions struct {
	rawPlatformContext
}

type projectCommandOptions struct {
	rawPlatformContext
}

type ticketListInput struct {
	projectID   string
	statusNames []string
	priorities  []string
}

type ticketCreateInput struct {
	projectID         string
	title             string
	description       string
	statusID          string
	priority          string
	typeName          string
	workflowID        string
	parentTicketID    string
	externalRef       string
	budgetUSD         float64
	archived          bool
	statusIDSet       bool
	workflowIDSet     bool
	parentTicketIDSet bool
	budgetUSDSet      bool
	archivedSet       bool
}

type ticketUpdateInput struct {
	ticketID          string
	title             string
	description       string
	externalRef       string
	statusID          string
	statusName        string
	priority          string
	typeName          string
	workflowID        string
	parentTicketID    string
	budgetUSD         float64
	archived          bool
	titleSet          bool
	descriptionSet    bool
	externalRefSet    bool
	statusIDSet       bool
	statusNameSet     bool
	prioritySet       bool
	typeSet           bool
	workflowIDSet     bool
	parentTicketIDSet bool
	budgetUSDSet      bool
	archivedSet       bool
}

type ticketReportUsageInput struct {
	ticketID        string
	inputTokens     *int64
	outputTokens    *int64
	costUSD         *float64
	inputTokensSet  bool
	outputTokensSet bool
	costUSDSet      bool
}

type ticketCommentListInput struct {
	ticketID string
}

type ticketCommentCreateInput struct {
	ticketID string
	body     string
}

type ticketCommentUpdateInput struct {
	ticketID  string
	commentID string
	body      string
}

type projectUpdateInput struct {
	projectID                  string
	name                       string
	slug                       string
	description                string
	status                     string
	defaultAgentProviderID     string
	projectAIPlatformAccess    []string
	accessibleMachineIDs       []string
	maxConcurrentAgents        *int
	agentRunSummaryPrompt      string
	nameSet                    bool
	slugSet                    bool
	descriptionSet             bool
	statusSet                  bool
	defaultAgentProviderSet    bool
	projectAIPlatformAccessSet bool
	accessibleMachineIDsSet    bool
	maxConcurrentAgentsSet     bool
	agentRunSummaryPromptSet   bool
}

type projectAddRepoInput struct {
	projectID        string
	name             string
	repositoryURL    string
	defaultBranch    string
	workspaceDirname string
	labels           []string
}

type platformClient struct {
	httpClient platformHTTPDoer
}

type platformHelpSpec struct {
	projectScope bool
	ticketScope  bool
	examples     []string
	notes        []string
}

func newAgentPlatformTicketCommandWithDeps(deps platformCommandDeps) *cobra.Command {
	options := &ticketCommandOptions{}
	client := platformClient(deps)

	command := &cobra.Command{
		Use:     "ticket",
		Short:   "Operate on OpenASE tickets through the agent platform API.",
		Long:    buildPlatformCommandHelp("Operate on OpenASE tickets through the agent platform API.", platformHelpSpec{projectScope: true, ticketScope: true}),
		Example: "openase ticket list\nopenase ticket update --description \"Blocked on flaky test\"",
	}

	applyCLICommandFlagNormalization(command)
	bindPlatformFlags(command.PersistentFlags(), &options.rawPlatformContext)
	command.AddCommand(newTicketListCommand(options, client))
	command.AddCommand(newTicketCreateCommand(options, client))
	command.AddCommand(newTicketCommentCommand(options, client))
	command.AddCommand(newTicketReportUsageCommand(options, client))
	command.AddCommand(newTicketUpdateCommand(options, client))

	return command
}

func newAgentPlatformProjectCommandWithDeps(deps platformCommandDeps) *cobra.Command {
	options := &projectCommandOptions{}
	client := platformClient(deps)

	command := &cobra.Command{
		Use:     "project",
		Short:   "Operate on OpenASE projects through the agent platform API.",
		Long:    buildPlatformCommandHelp("Operate on OpenASE projects through the agent platform API.", platformHelpSpec{projectScope: true}),
		Example: "openase project update --description \"Validation project\"\nopenase project add-repo --name backend --url https://github.com/acme/backend.git",
	}

	applyCLICommandFlagNormalization(command)
	bindPlatformFlags(command.PersistentFlags(), &options.rawPlatformContext)
	command.AddCommand(newProjectUpdateCommand(options, client))
	command.AddCommand(newProjectAddRepoCommand(options, client))

	return command
}

func newTicketCommentCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	command := &cobra.Command{
		Use:     "comment",
		Short:   "Operate on comments for the current ticket.",
		Long:    buildPlatformCommandHelp("Operate on comments for the current ticket.", platformHelpSpec{ticketScope: true}),
		Example: "openase ticket comment list\nopenase ticket comment create --body \"Progress\\n- reproduced issue\"",
	}

	command.AddCommand(newTicketCommentListCommand(options, client))
	command.AddCommand(newTicketCommentCreateCommand(options, client))
	command.AddCommand(newTicketCommentUpdateCommand(options, client))

	return command
}

func bindPlatformFlags(flags *pflag.FlagSet, raw *rawPlatformContext) {
	flags.StringVar(&raw.apiURL, "api-url", "", "Platform API base URL override. Defaults to OPENASE_API_URL.")
	flags.StringVar(&raw.token, "token", "", "Agent token override. Defaults to OPENASE_AGENT_TOKEN.")
	flags.StringVar(&raw.projectID, "project-id", "", "Project ID override. Defaults to OPENASE_PROJECT_ID.")
	flags.StringVar(&raw.ticketID, "ticket-id", "", "Ticket ID override. Defaults to OPENASE_TICKET_ID.")
}

func newTicketListCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	var statusNames []string
	var priorities []string

	spec := openAPICommandSpec{
		Method: http.MethodGet,
		Path:   "/api/v1/projects/{projectId}/tickets",
	}
	command := &cobra.Command{
		Use:   "list",
		Short: "List tickets in the current project.",
		Long: buildPlatformCommandHelp("List tickets in the current project.", platformHelpSpec{
			projectScope: true,
			examples: []string{
				"openase ticket list",
				"openase ticket list --status-name Todo --priority high",
			},
			notes: []string{
				"The target project defaults to --project-id and then OPENASE_PROJECT_ID.",
			},
		}),
		Example: "openase ticket list\nopenase ticket list --status-name Todo --priority high",
		RunE: func(cmd *cobra.Command, _ []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			input, err := platform.parseTicketListInput(ticketListInput{
				projectID:   options.projectID,
				statusNames: statusNames,
				priorities:  priorities,
			})
			if err != nil {
				return err
			}

			body, err := client.listTickets(cmd.Context(), platform, input)
			if err != nil {
				return err
			}
			return writePrettyJSON(cmd.OutOrStdout(), body)
		},
	}

	command.Flags().StringSliceVar(&statusNames, "status-name", nil, "Filter by one or more status names.")
	command.Flags().StringSliceVar(&priorities, "priority", nil, "Filter by one or more priorities.")

	return markCLICommandAPICoverageSpec(command, spec)
}

func newTicketCommentListCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	spec := openAPICommandSpec{
		Method: http.MethodGet,
		Path:   "/api/v1/tickets/{ticketId}/comments",
	}
	command := &cobra.Command{
		Use:   "list [ticket-id]",
		Short: "List comments for the current ticket.",
		Long: buildPlatformCommandHelp("List comments for the current ticket.", platformHelpSpec{
			ticketScope: true,
			examples: []string{
				"openase ticket comment list",
				"openase ticket comment list $OPENASE_TICKET_ID",
			},
			notes: []string{
				"If [ticket-id] is omitted, the command falls back to --ticket-id and then OPENASE_TICKET_ID.",
				"ticket-id is expected to be a UUID. Human-readable identifiers such as ASE-2 are not accepted.",
			},
		}),
		Example: "openase ticket comment list\nopenase ticket comment list $OPENASE_TICKET_ID",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			input, err := platform.parseTicketCommentListInput(ticketCommentListInput{
				ticketID: firstNonEmpty(firstArg(args), options.ticketID),
			})
			if err != nil {
				return err
			}

			body, err := client.listTicketComments(cmd.Context(), platform, input)
			if err != nil {
				return err
			}
			return writePrettyJSON(cmd.OutOrStdout(), body)
		},
	}

	return markCLICommandAPICoverageSpec(command, spec)
}

func newTicketCommentCreateCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	var body string
	var bodyFile string

	spec := openAPICommandSpec{
		Method: http.MethodPost,
		Path:   "/api/v1/tickets/{ticketId}/comments",
	}
	command := &cobra.Command{
		Use:   "create [ticket-id]",
		Short: "Create a comment on the current ticket.",
		Long: buildPlatformCommandHelp("Create a comment on the current ticket.", platformHelpSpec{
			ticketScope: true,
			examples: []string{
				"openase ticket comment create --body \"Need infra help\"",
				"openase ticket comment create $OPENASE_TICKET_ID --body-file /tmp/comment.md",
			},
			notes: []string{
				"Exactly one of --body or --body-file should be used to supply markdown content.",
			},
		}),
		Example: "openase ticket comment create --body \"Need infra help\"\nopenase ticket comment create $OPENASE_TICKET_ID --body-file /tmp/comment.md",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			commentBody, err := resolveCommentBody(body, bodyFile)
			if err != nil {
				return err
			}
			input, err := platform.parseTicketCommentCreateInput(ticketCommentCreateInput{
				ticketID: firstNonEmpty(firstArg(args), options.ticketID),
				body:     commentBody,
			})
			if err != nil {
				return err
			}

			responseBody, err := client.createTicketComment(cmd.Context(), platform, input)
			if err != nil {
				return err
			}
			return writePrettyJSON(cmd.OutOrStdout(), responseBody)
		},
	}

	command.Flags().StringVar(&body, "body", "", "Comment markdown body.")
	command.Flags().StringVar(&bodyFile, "body-file", "", "Read comment markdown from a file. Use '-' for stdin.")
	annotateCLICommandBodyFlag(command, "body", "body")
	annotateCLICommandBodyFlag(command, "body-file", "body")

	return markCLICommandAPICoverageSpec(command, spec)
}

func newTicketCommentUpdateCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	var body string
	var bodyFile string

	spec := openAPICommandSpec{
		Method: http.MethodPatch,
		Path:   "/api/v1/tickets/{ticketId}/comments/{commentId}",
	}
	command := &cobra.Command{
		Use:   "update [ticket-id] [comment-id]",
		Short: "Update a comment on the current ticket.",
		Long: buildPlatformCommandHelp("Update a comment on the current ticket.", platformHelpSpec{
			ticketScope: true,
			examples: []string{
				"openase ticket comment update $OPENASE_COMMENT_ID --body \"Updated progress\"",
				"openase ticket comment update $OPENASE_TICKET_ID $OPENASE_COMMENT_ID --body-file /tmp/comment.md",
			},
			notes: []string{
				"If two positional arguments are provided, the first is treated as ticket-id and the second as comment-id.",
				"If one positional argument is provided, it is treated as comment-id and ticket-id falls back to --ticket-id and then OPENASE_TICKET_ID.",
				"Both ticket-id and comment-id are expected to be UUIDs. Human-readable identifiers such as ASE-2 are not accepted.",
				"Exactly one of --body or --body-file should be used to supply markdown content.",
			},
		}),
		Example: "openase ticket comment update $OPENASE_COMMENT_ID --body \"Updated progress\"\nopenase ticket comment update $OPENASE_TICKET_ID $OPENASE_COMMENT_ID --body-file /tmp/comment.md",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			commentBody, err := resolveCommentBody(body, bodyFile)
			if err != nil {
				return err
			}
			input, err := platform.parseTicketCommentUpdateInput(ticketCommentUpdateInput{
				ticketID:  firstNonEmpty(ticketCommentUpdateTicketIDArg(args), options.ticketID, platform.ticketID),
				commentID: ticketCommentUpdateCommentIDArg(args),
				body:      commentBody,
			})
			if err != nil {
				return err
			}

			responseBody, err := client.updateTicketComment(cmd.Context(), platform, input)
			if err != nil {
				return err
			}
			return writePrettyJSON(cmd.OutOrStdout(), responseBody)
		},
	}

	command.Flags().StringVar(&body, "body", "", "Comment markdown body.")
	command.Flags().StringVar(&bodyFile, "body-file", "", "Read comment markdown from a file. Use '-' for stdin.")
	annotateCLICommandBodyFlag(command, "body", "body")
	annotateCLICommandBodyFlag(command, "body-file", "body")

	return markCLICommandAPICoverageSpec(
		markCLICommandIgnoredBodyFields(command, "edit_reason"),
		spec,
	)
}

func newTicketCreateCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	var title string
	var description string
	var statusID string
	var priority string
	var typeName string
	var workflowID string
	var parentTicketID string
	var externalRef string
	var budgetUSD float64
	var archived bool

	spec := openAPICommandSpec{
		Method: http.MethodPost,
		Path:   "/api/v1/projects/{projectId}/tickets",
	}
	command := &cobra.Command{
		Use:   "create",
		Short: "Create a ticket in the current project.",
		Long: buildPlatformCommandHelp("Create a ticket in the current project.", platformHelpSpec{
			projectScope: true,
			examples: []string{
				"openase ticket create --title \"Follow-up\" --description \"Split flaky test investigation\"",
				"openase ticket create --title \"Follow-up\" --workflow-id $OPENASE_WORKFLOW_ID --priority high",
			},
		}),
		Example: "openase ticket create --title \"Follow-up\" --description \"Split flaky test investigation\"\nopenase ticket create --title \"Follow-up\" --workflow-id $OPENASE_WORKFLOW_ID --priority high",
		RunE: func(cmd *cobra.Command, _ []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			input, err := platform.parseTicketCreateInput(ticketCreateInput{
				projectID:         options.projectID,
				title:             title,
				description:       description,
				statusID:          statusID,
				priority:          priority,
				typeName:          typeName,
				workflowID:        workflowID,
				parentTicketID:    parentTicketID,
				externalRef:       externalRef,
				budgetUSD:         budgetUSD,
				archived:          archived,
				statusIDSet:       cmd.Flags().Changed("status-id"),
				workflowIDSet:     cmd.Flags().Changed("workflow-id"),
				parentTicketIDSet: cmd.Flags().Changed("parent-ticket-id"),
				budgetUSDSet:      cmd.Flags().Changed("budget-usd"),
				archivedSet:       cmd.Flags().Changed("archived"),
			})
			if err != nil {
				return err
			}

			body, err := client.createTicket(cmd.Context(), platform, input)
			if err != nil {
				return err
			}
			return writePrettyJSON(cmd.OutOrStdout(), body)
		},
	}

	command.Flags().StringVar(&title, "title", "", "Ticket title.")
	command.Flags().StringVar(&description, "description", "", "Ticket description.")
	command.Flags().StringVar(&statusID, "status-id", "", "Ticket status ID override.")
	command.Flags().StringVar(&priority, "priority", "", "Ticket priority override.")
	command.Flags().StringVar(&typeName, "type", "", "Ticket type override.")
	command.Flags().StringVar(&workflowID, "workflow-id", "", "Workflow ID override.")
	command.Flags().StringVar(&parentTicketID, "parent-ticket-id", "", "Parent ticket ID override.")
	command.Flags().StringVar(&externalRef, "external-ref", "", "External reference, for example PacificStudio/openase#39.")
	command.Flags().Float64Var(&budgetUSD, "budget-usd", 0, "Optional ticket budget in USD.")
	command.Flags().BoolVar(&archived, "archived", false, "Create the ticket in archived state.")
	annotateCLICommandBodyFlag(command, "title", "title")
	annotateCLICommandBodyFlag(command, "description", "description")
	annotateCLICommandBodyFlag(command, "status-id", "status_id")
	annotateCLICommandBodyFlag(command, "priority", "priority")
	annotateCLICommandBodyFlag(command, "type", "type")
	annotateCLICommandBodyFlag(command, "workflow-id", "workflow_id")
	annotateCLICommandBodyFlag(command, "parent-ticket-id", "parent_ticket_id")
	annotateCLICommandBodyFlag(command, "external-ref", "external_ref")
	annotateCLICommandBodyFlag(command, "budget-usd", "budget_usd")
	annotateCLICommandBodyFlag(command, "archived", "archived")
	_ = command.MarkFlagRequired("title")

	return markCLICommandAPICoverageSpec(
		markCLICommandAllowedExtraBodyFields(
			command,
			"archived",
		),
		spec,
	)
}

func newTicketUpdateCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	var title string
	var description string
	var externalRef string
	var statusID string
	var statusName string
	var priority string
	var typeName string
	var workflowID string
	var parentTicketID string
	var budgetUSD float64
	var archived bool

	spec := openAPICommandSpec{
		Method: http.MethodPatch,
		Path:   "/api/v1/tickets/{ticketId}",
	}
	command := &cobra.Command{
		Use:   "update [ticket-id]",
		Short: "Update the current ticket or a specific ticket ID.",
		Long: buildPlatformCommandHelp("Update the current ticket or a specific ticket ID.", platformHelpSpec{
			ticketScope: true,
			examples: []string{
				"openase ticket update --description \"Blocked on remote CI\"",
				"openase ticket update $OPENASE_TICKET_ID --status-name Done",
				"openase ticket update $OPENASE_TICKET_ID --priority high --workflow-id $OPENASE_WORKFLOW_ID",
			},
			notes: []string{
				"If [ticket-id] is omitted, the command falls back to --ticket-id and then OPENASE_TICKET_ID.",
				"At least one update field must be provided.",
				"--status / --status-name and --status-id are mutually exclusive.",
				"--workflow-id, --parent-ticket-id, and --priority accept an empty string to clear the current value.",
				"ticket-id is expected to be a UUID. Human-readable identifiers such as ASE-2 are not accepted.",
			},
		}),
		Example: "openase ticket update --description \"Blocked on remote CI\"\nopenase ticket update $OPENASE_TICKET_ID --status-name Done\nopenase ticket update $OPENASE_TICKET_ID --priority high --workflow-id $OPENASE_WORKFLOW_ID",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			input, err := platform.parseTicketUpdateInput(ticketUpdateInput{
				ticketID:          firstNonEmpty(firstArg(args), options.ticketID),
				title:             title,
				description:       description,
				externalRef:       externalRef,
				statusID:          statusID,
				statusName:        statusName,
				priority:          priority,
				typeName:          typeName,
				workflowID:        workflowID,
				parentTicketID:    parentTicketID,
				budgetUSD:         budgetUSD,
				archived:          archived,
				titleSet:          cmd.Flags().Changed("title"),
				descriptionSet:    cmd.Flags().Changed("description"),
				externalRefSet:    cmd.Flags().Changed("external-ref"),
				statusIDSet:       cmd.Flags().Changed("status-id"),
				statusNameSet:     cmd.Flags().Changed("status") || cmd.Flags().Changed("status-name"),
				prioritySet:       cmd.Flags().Changed("priority"),
				typeSet:           cmd.Flags().Changed("type"),
				workflowIDSet:     cmd.Flags().Changed("workflow-id"),
				parentTicketIDSet: cmd.Flags().Changed("parent-ticket-id"),
				budgetUSDSet:      cmd.Flags().Changed("budget-usd"),
				archivedSet:       cmd.Flags().Changed("archived"),
			})
			if err != nil {
				return err
			}

			body, err := client.updateTicket(cmd.Context(), platform, input)
			if err != nil {
				return err
			}
			return writePrettyJSON(cmd.OutOrStdout(), body)
		},
	}

	command.Flags().StringVar(&title, "title", "", "Updated ticket title.")
	command.Flags().StringVar(&description, "description", "", "Updated ticket description.")
	command.Flags().StringVar(&externalRef, "external-ref", "", "Updated external reference.")
	command.Flags().StringVar(&statusName, "status", "", "Updated ticket status name.")
	command.Flags().StringVar(&statusName, "status-name", "", "Updated ticket status name.")
	command.Flags().StringVar(&statusID, "status-id", "", "Updated ticket status ID.")
	command.Flags().StringVar(&priority, "priority", "", "Updated ticket priority. Use an empty string to clear the current priority.")
	command.Flags().StringVar(&typeName, "type", "", "Updated ticket type.")
	command.Flags().StringVar(&workflowID, "workflow-id", "", "Updated workflow ID. Use an empty string to clear the current workflow.")
	command.Flags().StringVar(&parentTicketID, "parent-ticket-id", "", "Updated parent ticket ID. Use an empty string to clear the current parent.")
	command.Flags().Float64Var(&budgetUSD, "budget-usd", 0, "Updated ticket budget in USD.")
	command.Flags().BoolVar(&archived, "archived", false, "Archive or unarchive the ticket.")
	annotateCLICommandBodyFlag(command, "title", "title")
	annotateCLICommandBodyFlag(command, "description", "description")
	annotateCLICommandBodyFlag(command, "external-ref", "external_ref")
	annotateCLICommandBodyFlag(command, "status", "status_name")
	annotateCLICommandBodyFlag(command, "status-name", "status_name")
	annotateCLICommandBodyFlag(command, "status-id", "status_id")
	annotateCLICommandBodyFlag(command, "priority", "priority")
	annotateCLICommandBodyFlag(command, "type", "type")
	annotateCLICommandBodyFlag(command, "workflow-id", "workflow_id")
	annotateCLICommandBodyFlag(command, "parent-ticket-id", "parent_ticket_id")
	annotateCLICommandBodyFlag(command, "budget-usd", "budget_usd")
	annotateCLICommandBodyFlag(command, "archived", "archived")

	return markCLICommandAPICoverageSpec(
		markCLICommandAllowedExtraBodyFields(
			command,
			"archived",
			"status_name",
		),
		spec,
	)
}

func newTicketReportUsageCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	var inputTokens int64
	var outputTokens int64
	var costUSD float64

	command := &cobra.Command{
		Use:   "report-usage [ticket-id]",
		Short: "Report token and cost usage for the current ticket.",
		Long: buildPlatformCommandHelp("Report token and cost usage for the current ticket.", platformHelpSpec{
			ticketScope: true,
			examples: []string{
				"openase ticket report-usage --input-tokens 1200 --output-tokens 340 --cost-usd 0.0215",
			},
			notes: []string{
				"This command records usage deltas for the ticket. It does not replace previously reported totals.",
				"If [ticket-id] is omitted, the command falls back to --ticket-id and then OPENASE_TICKET_ID.",
				"At least one of --input-tokens, --output-tokens, or --cost-usd must be set.",
			},
		}),
		Example: "openase ticket report-usage --input-tokens 1200 --output-tokens 340 --cost-usd 0.0215",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}
			inputTokensSet := cmd.Flags().Changed("input-tokens")
			outputTokensSet := cmd.Flags().Changed("output-tokens")
			costUSDSet := cmd.Flags().Changed("cost-usd")

			input, err := platform.parseTicketReportUsageInput(ticketReportUsageInput{
				ticketID:        firstNonEmpty(firstArg(args), options.ticketID),
				inputTokens:     int64PointerWhen(inputTokensSet, inputTokens),
				outputTokens:    int64PointerWhen(outputTokensSet, outputTokens),
				costUSD:         float64PointerWhen(costUSDSet, costUSD),
				inputTokensSet:  inputTokensSet,
				outputTokensSet: outputTokensSet,
				costUSDSet:      costUSDSet,
			})
			if err != nil {
				return err
			}

			body, err := client.reportTicketUsage(cmd.Context(), platform, input)
			if err != nil {
				return err
			}
			return writePrettyJSON(cmd.OutOrStdout(), body)
		},
	}

	command.Flags().Int64Var(&inputTokens, "input-tokens", 0, "Prompt/input token delta to record.")
	command.Flags().Int64Var(&outputTokens, "output-tokens", 0, "Completion/output token delta to record.")
	command.Flags().Float64Var(&costUSD, "cost-usd", 0, "Explicit USD cost delta to record.")
	annotateCLICommandBodyFlag(command, "input-tokens", "input_tokens")
	annotateCLICommandBodyFlag(command, "output-tokens", "output_tokens")
	annotateCLICommandBodyFlag(command, "cost-usd", "cost_usd")

	return markCLICommandExpectedBodyFields(command, "input_tokens", "output_tokens", "cost_usd")
}

func newProjectUpdateCommand(options *projectCommandOptions, client platformClient) *cobra.Command {
	var name string
	var slug string
	var description string
	var status string
	var defaultAgentProviderID string
	var projectAIPlatformAccess []string
	var accessibleMachineIDs []string
	var maxConcurrentAgents int
	var agentRunSummaryPrompt string

	spec := openAPICommandSpec{
		Method: http.MethodPatch,
		Path:   "/api/v1/projects/{projectId}",
	}
	command := &cobra.Command{
		Use:   "update",
		Short: "Update the current project.",
		Long: buildPlatformCommandHelp("Update the current project.", platformHelpSpec{
			projectScope: true,
			examples: []string{
				"openase project update --description \"Todo App validation project\"",
				"openase project update --name \"OpenASE Automation\" --slug openase-automation --status \"In Progress\"",
				"openase project update --accessible-machine-ids $OPENASE_MACHINE_ID --max-concurrent-agents 4",
			},
			notes: []string{
				"At least one update field must be provided.",
				"Flags map to the fields accepted by PATCH /api/v1/projects/{projectId}.",
			},
		}),
		Example: "openase project update --description \"Todo App validation project\"\nopenase project update --name \"OpenASE Automation\" --slug openase-automation --status \"In Progress\"",
		RunE: func(cmd *cobra.Command, _ []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			input, err := platform.parseProjectUpdateInput(projectUpdateInput{
				projectID:                  options.projectID,
				name:                       name,
				slug:                       slug,
				description:                description,
				status:                     status,
				defaultAgentProviderID:     defaultAgentProviderID,
				projectAIPlatformAccess:    projectAIPlatformAccess,
				accessibleMachineIDs:       accessibleMachineIDs,
				maxConcurrentAgents:        intPointerWhen(cmd.Flags().Changed("max-concurrent-agents"), maxConcurrentAgents),
				agentRunSummaryPrompt:      agentRunSummaryPrompt,
				nameSet:                    cmd.Flags().Changed("name"),
				slugSet:                    cmd.Flags().Changed("slug"),
				descriptionSet:             cmd.Flags().Changed("description"),
				statusSet:                  cmd.Flags().Changed("status"),
				defaultAgentProviderSet:    cmd.Flags().Changed("default-agent-provider-id"),
				projectAIPlatformAccessSet: cmd.Flags().Changed("project-ai-platform-access-allowed"),
				accessibleMachineIDsSet:    cmd.Flags().Changed("accessible-machine-ids"),
				maxConcurrentAgentsSet:     cmd.Flags().Changed("max-concurrent-agents"),
				agentRunSummaryPromptSet:   cmd.Flags().Changed("agent-run-summary-prompt"),
			})
			if err != nil {
				return err
			}

			body, err := client.updateProject(cmd.Context(), platform, input)
			if err != nil {
				return err
			}
			return writePrettyJSON(cmd.OutOrStdout(), body)
		},
	}

	command.Flags().StringVar(&name, "name", "", "Updated project name.")
	command.Flags().StringVar(&slug, "slug", "", "Updated project slug.")
	command.Flags().StringVar(&description, "description", "", "Updated project description.")
	command.Flags().StringVar(&status, "status", "", "Updated project status.")
	command.Flags().StringVar(&defaultAgentProviderID, "default-agent-provider-id", "", "Updated default agent provider ID. Set to an empty string to clear it.")
	command.Flags().StringSliceVar(&projectAIPlatformAccess, "project-ai-platform-access-allowed", nil, "Updated allowed Project AI platform scopes. Repeat or comma-separate values; pass an empty value to clear.")
	command.Flags().StringSliceVar(&accessibleMachineIDs, "accessible-machine-ids", nil, "Updated accessible machine IDs. Repeat or comma-separate values; pass an empty value to clear.")
	command.Flags().IntVar(&maxConcurrentAgents, "max-concurrent-agents", 0, "Updated maximum concurrent agents.")
	command.Flags().StringVar(&agentRunSummaryPrompt, "agent-run-summary-prompt", "", "Updated agent run summary prompt. Set to an empty string to clear it.")
	annotateCLICommandBodyFlag(command, "name", "name")
	annotateCLICommandBodyFlag(command, "slug", "slug")
	annotateCLICommandBodyFlag(command, "description", "description")
	annotateCLICommandBodyFlag(command, "status", "status")
	annotateCLICommandBodyFlag(command, "default-agent-provider-id", "default_agent_provider_id")
	annotateCLICommandBodyFlag(command, "project-ai-platform-access-allowed", "project_ai_platform_access_allowed")
	annotateCLICommandBodyFlag(command, "accessible-machine-ids", "accessible_machine_ids")
	annotateCLICommandBodyFlag(command, "max-concurrent-agents", "max_concurrent_agents")
	annotateCLICommandBodyFlag(command, "agent-run-summary-prompt", "agent_run_summary_prompt")

	return markCLICommandAPICoverageSpec(command, spec)
}

func newProjectAddRepoCommand(options *projectCommandOptions, client platformClient) *cobra.Command {
	var name string
	var repositoryURL string
	var defaultBranch string
	var workspaceDirname string
	var labels []string

	spec := openAPICommandSpec{
		Method: http.MethodPost,
		Path:   "/api/v1/projects/{projectId}/repos",
	}
	command := &cobra.Command{
		Use:   "add-repo",
		Short: "Register a repository in the current project.",
		Long: buildPlatformCommandHelp("Register a repository in the current project.", platformHelpSpec{
			projectScope: true,
			examples: []string{
				"openase project add-repo --name backend --url https://github.com/acme/backend.git --default-branch main",
			},
			notes: []string{
				"The target project defaults to --project-id and then OPENASE_PROJECT_ID.",
			},
		}),
		Example: "openase project add-repo --name backend --url https://github.com/acme/backend.git --default-branch main",
		RunE: func(cmd *cobra.Command, _ []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			input, err := platform.parseProjectAddRepoInput(projectAddRepoInput{
				projectID:        options.projectID,
				name:             name,
				repositoryURL:    repositoryURL,
				defaultBranch:    defaultBranch,
				workspaceDirname: workspaceDirname,
				labels:           labels,
			})
			if err != nil {
				return err
			}

			body, err := client.addProjectRepo(cmd.Context(), platform, input)
			if err != nil {
				return err
			}
			return writePrettyJSON(cmd.OutOrStdout(), body)
		},
	}

	command.Flags().StringVar(&name, "name", "", "Repository name.")
	command.Flags().StringVar(&repositoryURL, "url", "", "Repository URL.")
	command.Flags().StringVar(&defaultBranch, "default-branch", "main", "Default branch name.")
	command.Flags().StringVar(&workspaceDirname, "workspace-dirname", "", "Workspace directory name override for this repository.")
	command.Flags().StringSliceVar(&labels, "label", nil, "Repository label. Repeat for multiple labels.")
	annotateCLICommandBodyFlag(command, "name", "name")
	annotateCLICommandBodyFlag(command, "url", "repository_url")
	annotateCLICommandBodyFlag(command, "default-branch", "default_branch")
	annotateCLICommandBodyFlag(command, "workspace-dirname", "workspace_dirname")
	annotateCLICommandBodyFlag(command, "label", "labels")
	_ = command.MarkFlagRequired("name")
	_ = command.MarkFlagRequired("url")

	return markCLICommandAPICoverageSpec(command, spec)
}

func (options *ticketCommandOptions) resolve() (platformContext, error) {
	return options.rawPlatformContext.resolve()
}

func (options *projectCommandOptions) resolve() (platformContext, error) {
	return options.rawPlatformContext.resolve()
}

func buildPlatformCommandHelp(summary string, spec platformHelpSpec) string {
	lines := []string{
		summary,
		"This command calls the agent platform API. It defaults to --api-url or OPENASE_API_URL, and to --token or OPENASE_AGENT_TOKEN for authentication.",
	}
	if spec.projectScope {
		lines = append(lines, "Project scope defaults to --project-id and then OPENASE_PROJECT_ID.")
	}
	if spec.ticketScope {
		lines = append(lines, "Ticket scope defaults to the positional [ticket-id] when provided, then --ticket-id, then OPENASE_TICKET_ID.")
		lines = append(lines, "Positional ticket-id values are expected to be UUIDs. Human-readable identifiers such as ASE-2 are not accepted.")
	}
	lines = append(lines, spec.notes...)
	if len(spec.examples) > 0 {
		lines = append(lines, "Examples:\n  "+strings.Join(spec.examples, "\n  "))
	}
	return strings.Join(lines, "\n\n")
}

func (raw rawPlatformContext) resolve() (platformContext, error) {
	apiURL := strings.TrimRight(strings.TrimSpace(firstNonEmpty(raw.apiURL, os.Getenv("OPENASE_API_URL"))), "/")
	if apiURL == "" {
		return platformContext{}, fmt.Errorf("platform api url is required via --api-url or OPENASE_API_URL")
	}
	if _, err := url.ParseRequestURI(apiURL); err != nil {
		return platformContext{}, fmt.Errorf("parse platform api url: %w", err)
	}

	token := strings.TrimSpace(firstNonEmpty(raw.token, os.Getenv("OPENASE_AGENT_TOKEN")))
	if token == "" {
		return platformContext{}, fmt.Errorf("agent token is required via --token or OPENASE_AGENT_TOKEN")
	}

	return platformContext{
		apiURL:    apiURL,
		token:     token,
		projectID: strings.TrimSpace(firstNonEmpty(raw.projectID, os.Getenv("OPENASE_PROJECT_ID"))),
		ticketID:  strings.TrimSpace(firstNonEmpty(raw.ticketID, os.Getenv("OPENASE_TICKET_ID"))),
		scopes:    parsePlatformScopeList(firstNonEmpty(strings.Join(raw.scopes, ","), os.Getenv("OPENASE_AGENT_SCOPES"))),
	}, nil
}

func parsePlatformScopeList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	items := strings.Split(raw, ",")
	scopes := make([]string, 0, len(items))
	for _, item := range items {
		scope := strings.TrimSpace(item)
		if scope == "" || slicesContains(scopes, scope) {
			continue
		}
		scopes = append(scopes, scope)
	}
	return scopes
}

func slicesContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func (platform platformContext) hasScope(scope string) bool {
	return slicesContains(platform.scopes, strings.TrimSpace(scope))
}

func (platform platformContext) parseTicketListInput(raw ticketListInput) (ticketListInput, error) {
	projectID := strings.TrimSpace(firstNonEmpty(raw.projectID, platform.projectID))
	if projectID == "" {
		return ticketListInput{}, fmt.Errorf("project id is required via --project-id or OPENASE_PROJECT_ID")
	}

	input := ticketListInput{
		projectID:   projectID,
		statusNames: make([]string, 0, len(raw.statusNames)),
		priorities:  make([]string, 0, len(raw.priorities)),
	}
	for _, item := range raw.statusNames {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			input.statusNames = append(input.statusNames, trimmed)
		}
	}
	for _, item := range raw.priorities {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			input.priorities = append(input.priorities, trimmed)
		}
	}

	return input, nil
}

func (platform platformContext) parseTicketCreateInput(raw ticketCreateInput) (ticketCreateInput, error) {
	projectID := strings.TrimSpace(firstNonEmpty(raw.projectID, platform.projectID))
	if projectID == "" {
		return ticketCreateInput{}, fmt.Errorf("project id is required via --project-id or OPENASE_PROJECT_ID")
	}

	title := strings.TrimSpace(raw.title)
	if title == "" {
		return ticketCreateInput{}, fmt.Errorf("title must not be empty")
	}
	if raw.budgetUSDSet && raw.budgetUSD < 0 {
		return ticketCreateInput{}, fmt.Errorf("budget-usd must be greater than or equal to zero")
	}

	return ticketCreateInput{
		projectID:         projectID,
		title:             title,
		description:       strings.TrimSpace(raw.description),
		statusID:          strings.TrimSpace(raw.statusID),
		priority:          strings.TrimSpace(raw.priority),
		typeName:          strings.TrimSpace(raw.typeName),
		workflowID:        strings.TrimSpace(raw.workflowID),
		parentTicketID:    strings.TrimSpace(raw.parentTicketID),
		externalRef:       strings.TrimSpace(raw.externalRef),
		budgetUSD:         raw.budgetUSD,
		archived:          raw.archived,
		statusIDSet:       raw.statusIDSet,
		workflowIDSet:     raw.workflowIDSet,
		parentTicketIDSet: raw.parentTicketIDSet,
		budgetUSDSet:      raw.budgetUSDSet,
		archivedSet:       raw.archivedSet,
	}, nil
}

func (platform platformContext) parseTicketUpdateInput(raw ticketUpdateInput) (ticketUpdateInput, error) {
	ticketID := strings.TrimSpace(firstNonEmpty(raw.ticketID, platform.ticketID))
	if ticketID == "" {
		return ticketUpdateInput{}, fmt.Errorf("ticket id is required via positional argument, --ticket-id, or OPENASE_TICKET_ID")
	}
	if !raw.titleSet &&
		!raw.descriptionSet &&
		!raw.externalRefSet &&
		!raw.statusIDSet &&
		!raw.statusNameSet &&
		!raw.prioritySet &&
		!raw.typeSet &&
		!raw.workflowIDSet &&
		!raw.parentTicketIDSet &&
		!raw.budgetUSDSet &&
		!raw.archivedSet {
		return ticketUpdateInput{}, fmt.Errorf(
			"at least one of --title, --description, --external-ref, --status, --status-name, --status-id, --priority, --type, --workflow-id, --parent-ticket-id, --budget-usd, or --archived must be set",
		)
	}
	if raw.titleSet && strings.TrimSpace(raw.title) == "" {
		return ticketUpdateInput{}, fmt.Errorf("title must not be empty")
	}
	if raw.statusIDSet && raw.statusNameSet {
		return ticketUpdateInput{}, fmt.Errorf("status-id and status name cannot be provided together")
	}
	if raw.statusIDSet && strings.TrimSpace(raw.statusID) == "" {
		return ticketUpdateInput{}, fmt.Errorf("status-id must not be empty")
	}
	if raw.statusNameSet && strings.TrimSpace(raw.statusName) == "" {
		return ticketUpdateInput{}, fmt.Errorf("status name must not be empty")
	}
	if raw.typeSet && strings.TrimSpace(raw.typeName) == "" {
		return ticketUpdateInput{}, fmt.Errorf("type must not be empty")
	}
	if raw.budgetUSDSet && raw.budgetUSD < 0 {
		return ticketUpdateInput{}, fmt.Errorf("budget-usd must be greater than or equal to zero")
	}

	return ticketUpdateInput{
		ticketID:          ticketID,
		title:             strings.TrimSpace(raw.title),
		description:       strings.TrimSpace(raw.description),
		externalRef:       strings.TrimSpace(raw.externalRef),
		statusID:          strings.TrimSpace(raw.statusID),
		statusName:        strings.TrimSpace(raw.statusName),
		priority:          strings.TrimSpace(raw.priority),
		typeName:          strings.TrimSpace(raw.typeName),
		workflowID:        strings.TrimSpace(raw.workflowID),
		parentTicketID:    strings.TrimSpace(raw.parentTicketID),
		budgetUSD:         raw.budgetUSD,
		archived:          raw.archived,
		titleSet:          raw.titleSet,
		descriptionSet:    raw.descriptionSet,
		externalRefSet:    raw.externalRefSet,
		statusIDSet:       raw.statusIDSet,
		statusNameSet:     raw.statusNameSet,
		prioritySet:       raw.prioritySet,
		typeSet:           raw.typeSet,
		workflowIDSet:     raw.workflowIDSet,
		parentTicketIDSet: raw.parentTicketIDSet,
		budgetUSDSet:      raw.budgetUSDSet,
		archivedSet:       raw.archivedSet,
	}, nil
}

func (platform platformContext) parseTicketReportUsageInput(raw ticketReportUsageInput) (ticketReportUsageInput, error) {
	ticketID := strings.TrimSpace(firstNonEmpty(raw.ticketID, platform.ticketID))
	if ticketID == "" {
		return ticketReportUsageInput{}, fmt.Errorf("ticket id is required via positional argument, --ticket-id, or OPENASE_TICKET_ID")
	}
	if !raw.inputTokensSet && !raw.outputTokensSet && !raw.costUSDSet {
		return ticketReportUsageInput{}, fmt.Errorf("at least one of --input-tokens, --output-tokens, or --cost-usd must be set")
	}
	if raw.inputTokens != nil && *raw.inputTokens < 0 {
		return ticketReportUsageInput{}, fmt.Errorf("input-tokens must be greater than or equal to zero")
	}
	if raw.outputTokens != nil && *raw.outputTokens < 0 {
		return ticketReportUsageInput{}, fmt.Errorf("output-tokens must be greater than or equal to zero")
	}
	if raw.costUSD != nil && *raw.costUSD < 0 {
		return ticketReportUsageInput{}, fmt.Errorf("cost-usd must be greater than or equal to zero")
	}

	return ticketReportUsageInput{
		ticketID:        ticketID,
		inputTokens:     raw.inputTokens,
		outputTokens:    raw.outputTokens,
		costUSD:         raw.costUSD,
		inputTokensSet:  raw.inputTokensSet,
		outputTokensSet: raw.outputTokensSet,
		costUSDSet:      raw.costUSDSet,
	}, nil
}

func (platform platformContext) parseTicketCommentListInput(raw ticketCommentListInput) (ticketCommentListInput, error) {
	ticketID := strings.TrimSpace(firstNonEmpty(raw.ticketID, platform.ticketID))
	if ticketID == "" {
		return ticketCommentListInput{}, fmt.Errorf("ticket id is required via positional argument, --ticket-id, or OPENASE_TICKET_ID")
	}

	return ticketCommentListInput{ticketID: ticketID}, nil
}

func (platform platformContext) parseTicketCommentCreateInput(raw ticketCommentCreateInput) (ticketCommentCreateInput, error) {
	ticketID := strings.TrimSpace(firstNonEmpty(raw.ticketID, platform.ticketID))
	if ticketID == "" {
		return ticketCommentCreateInput{}, fmt.Errorf("ticket id is required via positional argument, --ticket-id, or OPENASE_TICKET_ID")
	}
	body := strings.TrimSpace(raw.body)
	if body == "" {
		return ticketCommentCreateInput{}, fmt.Errorf("comment body must not be empty")
	}

	return ticketCommentCreateInput{ticketID: ticketID, body: body}, nil
}

func (platform platformContext) parseTicketCommentUpdateInput(raw ticketCommentUpdateInput) (ticketCommentUpdateInput, error) {
	ticketID := strings.TrimSpace(firstNonEmpty(raw.ticketID, platform.ticketID))
	if ticketID == "" {
		return ticketCommentUpdateInput{}, fmt.Errorf("ticket id is required via --ticket-id or OPENASE_TICKET_ID")
	}
	commentID := strings.TrimSpace(raw.commentID)
	if commentID == "" {
		return ticketCommentUpdateInput{}, fmt.Errorf("comment id must not be empty")
	}
	body := strings.TrimSpace(raw.body)
	if body == "" {
		return ticketCommentUpdateInput{}, fmt.Errorf("comment body must not be empty")
	}

	return ticketCommentUpdateInput{
		ticketID:  ticketID,
		commentID: commentID,
		body:      body,
	}, nil
}

func (platform platformContext) parseProjectUpdateInput(raw projectUpdateInput) (projectUpdateInput, error) {
	projectID := strings.TrimSpace(firstNonEmpty(raw.projectID, platform.projectID))
	if projectID == "" {
		return projectUpdateInput{}, fmt.Errorf("project id is required via --project-id or OPENASE_PROJECT_ID")
	}
	if !raw.nameSet &&
		!raw.slugSet &&
		!raw.descriptionSet &&
		!raw.statusSet &&
		!raw.defaultAgentProviderSet &&
		!raw.projectAIPlatformAccessSet &&
		!raw.accessibleMachineIDsSet &&
		!raw.maxConcurrentAgentsSet &&
		!raw.agentRunSummaryPromptSet {
		return projectUpdateInput{}, fmt.Errorf("at least one of --name, --slug, --description, --status, --default-agent-provider-id, --project-ai-platform-access-allowed, --accessible-machine-ids, --max-concurrent-agents, or --agent-run-summary-prompt must be set")
	}

	input := projectUpdateInput{
		projectID:                  projectID,
		nameSet:                    raw.nameSet,
		slugSet:                    raw.slugSet,
		descriptionSet:             raw.descriptionSet,
		statusSet:                  raw.statusSet,
		defaultAgentProviderSet:    raw.defaultAgentProviderSet,
		projectAIPlatformAccessSet: raw.projectAIPlatformAccessSet,
		accessibleMachineIDsSet:    raw.accessibleMachineIDsSet,
		maxConcurrentAgentsSet:     raw.maxConcurrentAgentsSet,
		agentRunSummaryPromptSet:   raw.agentRunSummaryPromptSet,
	}
	if raw.nameSet {
		input.name = strings.TrimSpace(raw.name)
	}
	if raw.slugSet {
		input.slug = strings.TrimSpace(raw.slug)
	}
	if raw.descriptionSet {
		input.description = strings.TrimSpace(raw.description)
	}
	if raw.statusSet {
		input.status = strings.TrimSpace(raw.status)
	}
	if raw.defaultAgentProviderSet {
		input.defaultAgentProviderID = strings.TrimSpace(raw.defaultAgentProviderID)
	}
	if raw.projectAIPlatformAccessSet {
		input.projectAIPlatformAccess = make([]string, 0, len(raw.projectAIPlatformAccess))
		for _, item := range raw.projectAIPlatformAccess {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				input.projectAIPlatformAccess = append(input.projectAIPlatformAccess, trimmed)
			}
		}
	}
	if raw.accessibleMachineIDsSet {
		input.accessibleMachineIDs = make([]string, 0, len(raw.accessibleMachineIDs))
		for _, item := range raw.accessibleMachineIDs {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				input.accessibleMachineIDs = append(input.accessibleMachineIDs, trimmed)
			}
		}
	}
	if raw.maxConcurrentAgentsSet {
		input.maxConcurrentAgents = raw.maxConcurrentAgents
	}
	if raw.agentRunSummaryPromptSet {
		input.agentRunSummaryPrompt = strings.TrimSpace(raw.agentRunSummaryPrompt)
	}

	return projectUpdateInput{
		projectID:                  input.projectID,
		name:                       input.name,
		slug:                       input.slug,
		description:                input.description,
		status:                     input.status,
		defaultAgentProviderID:     input.defaultAgentProviderID,
		projectAIPlatformAccess:    input.projectAIPlatformAccess,
		accessibleMachineIDs:       input.accessibleMachineIDs,
		maxConcurrentAgents:        input.maxConcurrentAgents,
		agentRunSummaryPrompt:      input.agentRunSummaryPrompt,
		nameSet:                    input.nameSet,
		slugSet:                    input.slugSet,
		descriptionSet:             input.descriptionSet,
		statusSet:                  input.statusSet,
		defaultAgentProviderSet:    input.defaultAgentProviderSet,
		projectAIPlatformAccessSet: input.projectAIPlatformAccessSet,
		accessibleMachineIDsSet:    input.accessibleMachineIDsSet,
		maxConcurrentAgentsSet:     input.maxConcurrentAgentsSet,
		agentRunSummaryPromptSet:   input.agentRunSummaryPromptSet,
	}, nil
}

func (platform platformContext) parseProjectAddRepoInput(raw projectAddRepoInput) (projectAddRepoInput, error) {
	projectID := strings.TrimSpace(firstNonEmpty(raw.projectID, platform.projectID))
	if projectID == "" {
		return projectAddRepoInput{}, fmt.Errorf("project id is required via --project-id or OPENASE_PROJECT_ID")
	}

	name := strings.TrimSpace(raw.name)
	if name == "" {
		return projectAddRepoInput{}, fmt.Errorf("name must not be empty")
	}
	repositoryURL := strings.TrimSpace(raw.repositoryURL)
	if repositoryURL == "" {
		return projectAddRepoInput{}, fmt.Errorf("url must not be empty")
	}

	input := projectAddRepoInput{
		projectID:        projectID,
		name:             name,
		repositoryURL:    repositoryURL,
		defaultBranch:    strings.TrimSpace(raw.defaultBranch),
		workspaceDirname: strings.TrimSpace(raw.workspaceDirname),
		labels:           make([]string, 0, len(raw.labels)),
	}
	if input.defaultBranch == "" {
		input.defaultBranch = "main"
	}
	for _, item := range raw.labels {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			input.labels = append(input.labels, trimmed)
		}
	}

	return input, nil
}

func (client platformClient) listTickets(ctx context.Context, platform platformContext, input ticketListInput) ([]byte, error) {
	values := url.Values{}
	if len(input.statusNames) > 0 {
		values.Set("status_name", strings.Join(input.statusNames, ","))
	}
	if len(input.priorities) > 0 {
		values.Set("priority", strings.Join(input.priorities, ","))
	}

	path := "/projects/" + url.PathEscape(input.projectID) + "/tickets"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}

	return client.doJSON(ctx, platform, http.MethodGet, path, nil)
}

func (client platformClient) createTicket(ctx context.Context, platform platformContext, input ticketCreateInput) ([]byte, error) {
	payload := map[string]any{
		"title": input.title,
	}
	if input.description != "" {
		payload["description"] = input.description
	}
	if input.statusIDSet {
		payload["status_id"] = input.statusID
	}
	if input.priority != "" {
		payload["priority"] = input.priority
	}
	if input.typeName != "" {
		payload["type"] = input.typeName
	}
	if input.workflowIDSet {
		payload["workflow_id"] = input.workflowID
	}
	if input.parentTicketIDSet {
		payload["parent_ticket_id"] = input.parentTicketID
	}
	if input.externalRef != "" {
		payload["external_ref"] = input.externalRef
	}
	if input.budgetUSDSet {
		payload["budget_usd"] = input.budgetUSD
	}
	if input.archivedSet {
		payload["archived"] = input.archived
	}

	return client.doJSON(ctx, platform, http.MethodPost, "/projects/"+url.PathEscape(input.projectID)+"/tickets", payload)
}

func (client platformClient) listTicketComments(ctx context.Context, platform platformContext, input ticketCommentListInput) ([]byte, error) {
	return client.doJSON(ctx, platform, http.MethodGet, "/tickets/"+url.PathEscape(input.ticketID)+"/comments", nil)
}

func (client platformClient) createTicketComment(ctx context.Context, platform platformContext, input ticketCommentCreateInput) ([]byte, error) {
	return client.doJSON(ctx, platform, http.MethodPost, "/tickets/"+url.PathEscape(input.ticketID)+"/comments", map[string]any{
		"body": input.body,
	})
}

func (client platformClient) updateTicketComment(ctx context.Context, platform platformContext, input ticketCommentUpdateInput) ([]byte, error) {
	return client.doJSON(
		ctx,
		platform,
		http.MethodPatch,
		"/tickets/"+url.PathEscape(input.ticketID)+"/comments/"+url.PathEscape(input.commentID),
		map[string]any{"body": input.body},
	)
}

func (client platformClient) updateTicket(ctx context.Context, platform platformContext, input ticketUpdateInput) ([]byte, error) {
	payload := map[string]any{}
	if input.titleSet {
		payload["title"] = input.title
	}
	if input.descriptionSet {
		payload["description"] = input.description
	}
	if input.externalRefSet {
		payload["external_ref"] = input.externalRef
	}
	if input.statusIDSet {
		payload["status_id"] = input.statusID
	}
	if input.statusNameSet {
		payload["status_name"] = input.statusName
	}
	if input.prioritySet {
		payload["priority"] = input.priority
	}
	if input.typeSet {
		payload["type"] = input.typeName
	}
	if input.workflowIDSet {
		payload["workflow_id"] = input.workflowID
	}
	if input.parentTicketIDSet {
		payload["parent_ticket_id"] = input.parentTicketID
	}
	if input.budgetUSDSet {
		payload["budget_usd"] = input.budgetUSD
	}
	if input.archivedSet {
		payload["archived"] = input.archived
	}

	return client.doJSON(ctx, platform, http.MethodPatch, "/tickets/"+url.PathEscape(input.ticketID), payload)
}

func (client platformClient) reportTicketUsage(ctx context.Context, platform platformContext, input ticketReportUsageInput) ([]byte, error) {
	payload := map[string]any{}
	if input.inputTokensSet {
		payload["input_tokens"] = *input.inputTokens
	}
	if input.outputTokensSet {
		payload["output_tokens"] = *input.outputTokens
	}
	if input.costUSDSet {
		payload["cost_usd"] = *input.costUSD
	}

	return client.doJSON(ctx, platform, http.MethodPost, "/tickets/"+url.PathEscape(input.ticketID)+"/usage", payload)
}

func (client platformClient) updateProject(ctx context.Context, platform platformContext, input projectUpdateInput) ([]byte, error) {
	payload := map[string]any{}
	if input.nameSet {
		payload["name"] = input.name
	}
	if input.slugSet {
		payload["slug"] = input.slug
	}
	if input.descriptionSet {
		payload["description"] = input.description
	}
	if input.statusSet {
		payload["status"] = input.status
	}
	if input.defaultAgentProviderSet {
		payload["default_agent_provider_id"] = input.defaultAgentProviderID
	}
	if input.projectAIPlatformAccessSet {
		payload["project_ai_platform_access_allowed"] = input.projectAIPlatformAccess
	}
	if input.accessibleMachineIDsSet {
		payload["accessible_machine_ids"] = input.accessibleMachineIDs
	}
	if input.maxConcurrentAgentsSet {
		if input.maxConcurrentAgents != nil {
			payload["max_concurrent_agents"] = *input.maxConcurrentAgents
		}
	}
	if input.agentRunSummaryPromptSet {
		payload["agent_run_summary_prompt"] = input.agentRunSummaryPrompt
	}

	return client.doJSON(ctx, platform, http.MethodPatch, "/projects/"+url.PathEscape(input.projectID), payload)
}

func (client platformClient) addProjectRepo(ctx context.Context, platform platformContext, input projectAddRepoInput) ([]byte, error) {
	payload := map[string]any{
		"name":           input.name,
		"repository_url": input.repositoryURL,
		"default_branch": input.defaultBranch,
	}
	if input.workspaceDirname != "" {
		payload["workspace_dirname"] = input.workspaceDirname
	}
	if len(input.labels) > 0 {
		payload["labels"] = input.labels
	}

	return client.doJSON(ctx, platform, http.MethodPost, "/projects/"+url.PathEscape(input.projectID)+"/repos", payload)
}

func (client platformClient) doJSON(ctx context.Context, platform platformContext, method string, path string, payload any) ([]byte, error) {
	var bodyReader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal %s %s payload: %w", method, path, err)
		}
		bodyReader = bytes.NewReader(body)
	}

	request, err := http.NewRequestWithContext(ctx, method, platform.apiURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build %s %s request: %w", method, path, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+platform.token)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s %s response: %w", method, path, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%s %s returned %s: %s", method, path, response.Status, extractPlatformErrorMessage(body))
	}

	return body, nil
}

func extractPlatformErrorMessage(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return strings.TrimSpace(string(body))
	}

	for _, key := range []string{"message", "error"} {
		value, ok := payload[key]
		if !ok {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" {
			return text
		}
	}

	return strings.TrimSpace(string(body))
}

func writePrettyJSON(out io.Writer, body []byte) error {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err != nil {
		if len(body) == 0 || body[len(body)-1] == '\n' {
			_, writeErr := out.Write(body)
			return writeErr
		}
		_, writeErr := fmt.Fprintf(out, "%s\n", body)
		return writeErr
	}

	pretty.WriteByte('\n')
	_, err := out.Write(pretty.Bytes())
	return err
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func ticketCommentUpdateTicketIDArg(args []string) string {
	if len(args) == 2 {
		return strings.TrimSpace(args[0])
	}
	return ""
}

func ticketCommentUpdateCommentIDArg(args []string) string {
	switch len(args) {
	case 0:
		return ""
	case 1:
		return strings.TrimSpace(args[0])
	default:
		return strings.TrimSpace(args[1])
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func int64PointerWhen(enabled bool, value int64) *int64 {
	if !enabled {
		return nil
	}

	return &value
}

func intPointerWhen(enabled bool, value int) *int {
	if !enabled {
		return nil
	}

	return &value
}

func float64PointerWhen(enabled bool, value float64) *float64 {
	if !enabled {
		return nil
	}

	return &value
}

func resolveCommentBody(body string, bodyFile string) (string, error) {
	if strings.TrimSpace(body) != "" && strings.TrimSpace(bodyFile) != "" {
		return "", fmt.Errorf("body and body-file cannot be used together")
	}
	if strings.TrimSpace(body) != "" {
		return strings.TrimSpace(body), nil
	}

	file := strings.TrimSpace(bodyFile)
	if file == "" {
		return "", fmt.Errorf("either --body or --body-file is required")
	}
	if file == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read comment body from stdin: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return "", fmt.Errorf("read comment body from %s: %w", file, err)
	}
	return strings.TrimSpace(string(data)), nil
}
