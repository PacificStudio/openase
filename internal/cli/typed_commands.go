package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/BetterAndBetterII/openase/internal/httpapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type flagValueKind string

const (
	flagValueString      flagValueKind = "string"
	flagValueStringSlice flagValueKind = "string_slice"
	flagValueInt64       flagValueKind = "int64"
	flagValueFloat64     flagValueKind = "float64"
	flagValueBool        flagValueKind = "bool"
)

type openAPICommandSpec struct {
	Use              string   `json:"use"`
	Short            string   `json:"short"`
	Method           string   `json:"method"`
	Path             string   `json:"path"`
	PositionalParams []string `json:"positional_params,omitempty"`
	HelpNotes        []string `json:"help_notes,omitempty"`
	Example          string   `json:"example,omitempty"`
}

type openAPICommandContract struct {
	spec         openAPICommandSpec
	operationID  string
	summary      string
	pathParams   []openAPIInputField
	queryParams  []openAPIInputField
	bodyFields   []openAPIInputField
	hasBody      bool
	requiredBody []string
}

type openAPIInputField struct {
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Required    bool          `json:"required,omitempty"`
	Kind        flagValueKind `json:"kind"`
}

type openAPIContractSnapshot struct {
	OpenAPISHA256 string                        `json:"openapi_sha256"`
	Commands      []openAPIContractSnapshotItem `json:"commands"`
}

type openAPIContractSnapshotItem struct {
	Use              string              `json:"use"`
	Method           string              `json:"method"`
	Path             string              `json:"path"`
	OperationID      string              `json:"operation_id"`
	PositionalParams []string            `json:"positional_params,omitempty"`
	PathParams       []openAPIInputField `json:"path_params,omitempty"`
	QueryParams      []openAPIInputField `json:"query_params,omitempty"`
	BodyFields       []openAPIInputField `json:"body_fields,omitempty"`
}

type cliProjectResponseEnvelope struct {
	Project struct {
		ID             string `json:"id"`
		OrganizationID string `json:"organization_id"`
	} `json:"project"`
}

var (
	cliContractOnce sync.Once
	cliContractData map[string]openAPICommandContract
	cliContractErr  error
)

func newAPICommand() *cobra.Command {
	deps := apiCommandDeps{httpClient: http.DefaultClient}
	var options apiCommandOptions
	var output apiOutputOptions
	var headers []string
	var fields []string
	var queryItems []string
	var inputPath string
	command := &cobra.Command{
		Use:   "api METHOD PATH",
		Short: "Call the OpenASE HTTP API directly.",
		Long: strings.TrimSpace(`
Call the OpenASE HTTP API directly.

This is the raw passthrough CLI entrypoint. It accepts an HTTP method and path,
then forwards the request to the configured OpenASE API without inventing a
second resource model.

Use -f/--field to assemble a JSON body from key=value pairs, --query to append
query parameters, --header to add custom headers, and --input to send a raw
request body from a file or stdin. --input cannot be combined with body fields.
`),
		Example: strings.TrimSpace(`
  openase api GET /api/v1/tickets/$OPENASE_TICKET_ID
  openase api POST /api/v1/projects/$OPENASE_PROJECT_ID/tickets -f title="Follow-up" -f workflow_id="550e8400-e29b-41d4-a716-446655440000"
  openase api PATCH /api/v1/tickets/$OPENASE_TICKET_ID/comments/$COMMENT_ID --input payload.json
  openase api GET /api/v1/projects/$OPENASE_PROJECT_ID/tickets --query status_name=Todo --query priority=high
`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRawAPICommand(cmd, deps, args[0], args[1], options, output, headers, fields, queryItems, inputPath)
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	bindAPICommandFlags(command.Flags(), &options)
	bindAPIOutputFlags(command.Flags(), &output)
	command.Flags().StringSliceVarP(&fields, "field", "f", nil, "Add a JSON body field as key=value. Repeat for multiple fields.")
	command.Flags().StringSliceVar(&queryItems, "query", nil, "Add a query string field as key=value. Repeat for multiple fields.")
	command.Flags().StringSliceVar(&headers, "header", nil, "Add an HTTP header as Key: Value. Repeat for multiple headers.")
	command.Flags().StringVar(&inputPath, "input", "", "Read the raw request body from a file. Use - for stdin.")
	return command
}

func runRawAPICommand(
	cmd *cobra.Command,
	deps apiCommandDeps,
	method string,
	path string,
	options apiCommandOptions,
	output apiOutputOptions,
	headers []string,
	fields []string,
	queryItems []string,
	inputPath string,
) error {
	apiContext, err := options.resolve()
	if err != nil {
		return err
	}

	requestHeaders, err := parseHeaderPairs(headers)
	if err != nil {
		return err
	}
	body, err := buildRequestBody(fields, inputPath)
	if err != nil {
		return err
	}
	requestPath, err := buildRawRequestPath(path, queryItems)
	if err != nil {
		return err
	}

	response, err := apiContext.do(cmd.Context(), deps, apiRequest{
		Method:  strings.ToUpper(strings.TrimSpace(method)),
		Path:    requestPath,
		Body:    body,
		Headers: requestHeaders,
	})
	if err != nil {
		return err
	}

	return writeAPIOutput(cmd.OutOrStdout(), response.Body, output)
}

func newTicketCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "ticket",
		Short: "Operate on tickets through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [projectId]", Short: "List tickets.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets", PositionalParams: []string{"projectId"}}))
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
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [projectId]", Short: "Create a ticket.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/tickets", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [ticketId]", Short: "Update a ticket.", Method: http.MethodPatch, Path: "/api/v1/tickets/{ticketId}", PositionalParams: []string{"ticketId"}}))
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
	command.AddCommand(newTypedTicketCommentCommand())
	command.AddCommand(newTypedTicketDependencyCommand())
	command.AddCommand(newTypedTicketExternalLinkCommand())
	command.AddCommand(newTypedTicketRunCommand())
	return command
}

func newActivityCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "activity",
		Short: "Read project activity events through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "list [projectId]",
		Short:            "List project activity events.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/activity",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"Use this to inspect the project event timeline, including workflow edits, ticket transitions, and runtime activity.",
		},
		Example: "openase activity list $OPENASE_PROJECT_ID --json events",
	}))
	return command
}

func newStatusCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "status",
		Short: "Operate on ticket statuses through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
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
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "create [projectId]",
		Short:            "Create a ticket status.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/statuses",
		PositionalParams: []string{"projectId"},
		Example:          `openase status create $OPENASE_PROJECT_ID --name "QA" --stage started --color "#FF00AA" --description "Quality gate"`,
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "update [statusId]",
		Short:            "Update a ticket status.",
		Method:           http.MethodPatch,
		Path:             "/api/v1/statuses/{statusId}",
		PositionalParams: []string{"statusId"},
		Example:          `openase status update $OPENASE_STATUS_ID --name "Ready for QA" --stage completed --position 5`,
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "delete [statusId]",
		Short:            "Delete a ticket status.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/statuses/{statusId}",
		PositionalParams: []string{"statusId"},
		Example:          "openase status delete $OPENASE_STATUS_ID",
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "reset [projectId]",
		Short:            "Reset project statuses to the default template.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/statuses/reset",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"This replaces the project's status board with the built-in default template and should be treated as an administrative operation.",
		},
		Example: "openase status reset $OPENASE_PROJECT_ID",
	}))
	return command
}

func newChatCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "chat",
		Short: "Operate on chat sessions and project conversations through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:    "send",
		Short:  "Start an ephemeral chat request.",
		Method: http.MethodPost,
		Path:   "/api/v1/chat",
		HelpNotes: []string{
			"This is the raw ephemeral chat entrypoint. Use the conversation subcommands when you need persistent project conversation state instead of a one-shot session.",
		},
		Example: `openase chat send --message "Summarize the latest ticket activity"`,
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "close [sessionId]",
		Short:            "Close an ephemeral chat session.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/chat/{sessionId}",
		PositionalParams: []string{"sessionId"},
		Example:          "openase chat close $OPENASE_CHAT_SESSION_ID",
	}))

	conversation := &cobra.Command{
		Use:   "conversation",
		Short: "Operate on persistent project chat conversations.",
	}
	conversation.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:    "create",
		Short:  "Create a project conversation.",
		Method: http.MethodPost,
		Path:   "/api/v1/chat/conversations",
		HelpNotes: []string{
			"Create a persistent project-scoped conversation before sending turns when you need resumable AI collaboration state.",
		},
		Example: `openase chat conversation create --source project --provider-id $OPENASE_PROVIDER_ID --context.project-id $OPENASE_PROJECT_ID`,
	}))
	conversation.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:    "list",
		Short:  "List project conversations.",
		Method: http.MethodGet,
		Path:   "/api/v1/chat/conversations",
		HelpNotes: []string{
			"Filter by --project-id to scope the list to one project, and optionally add --provider-id when you are auditing provider-specific conversations.",
		},
		Example: "openase chat conversation list --project-id $OPENASE_PROJECT_ID",
	}))
	conversation.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "get [conversationId]",
		Short:            "Get a project conversation.",
		Method:           http.MethodGet,
		Path:             "/api/v1/chat/conversations/{conversationId}",
		PositionalParams: []string{"conversationId"},
	}))
	conversation.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "entries [conversationId]",
		Short:            "List project conversation transcript entries.",
		Method:           http.MethodGet,
		Path:             "/api/v1/chat/conversations/{conversationId}/entries",
		PositionalParams: []string{"conversationId"},
	}))
	conversation.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "workspace-diff [conversationId]",
		Short:            "Get project conversation workspace diff summary.",
		Method:           http.MethodGet,
		Path:             "/api/v1/chat/conversations/{conversationId}/workspace-diff",
		PositionalParams: []string{"conversationId"},
	}))
	conversation.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "turn [conversationId]",
		Short:            "Start a project conversation turn.",
		Method:           http.MethodPost,
		Path:             "/api/v1/chat/conversations/{conversationId}/turns",
		PositionalParams: []string{"conversationId"},
		Example:          `openase chat conversation turn $OPENASE_CONVERSATION_ID --message "Continue from the last interrupted step"`,
	}))
	conversation.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{
		Use:              "watch [conversationId]",
		Short:            "Watch project conversation events.",
		Method:           http.MethodGet,
		Path:             "/api/v1/chat/conversations/{conversationId}/stream",
		PositionalParams: []string{"conversationId"},
		Example:          "openase chat conversation watch $OPENASE_CONVERSATION_ID",
	}))
	conversation.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "respond-interrupt [conversationId] [interruptId]",
		Short:            "Respond to a project conversation interrupt.",
		Method:           http.MethodPost,
		Path:             "/api/v1/chat/conversations/{conversationId}/interrupts/{interruptId}/respond",
		PositionalParams: []string{"conversationId", "interruptId"},
	}))
	conversation.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "close-runtime [conversationId]",
		Short:            "Close a project conversation live runtime.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/chat/conversations/{conversationId}/runtime",
		PositionalParams: []string{"conversationId"},
	}))
	command.AddCommand(conversation)

	return command
}

func newTypedTicketCommentCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "comment",
		Short: "Operate on ticket comments.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [ticketId]", Short: "List ticket comments.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}/comments", PositionalParams: []string{"ticketId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [ticketId]", Short: "Create a ticket comment.", Method: http.MethodPost, Path: "/api/v1/tickets/{ticketId}/comments", PositionalParams: []string{"ticketId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "update [ticketId] [commentId]",
		Short:            "Update a ticket comment.",
		Method:           http.MethodPatch,
		Path:             "/api/v1/tickets/{ticketId}/comments/{commentId}",
		PositionalParams: []string{"ticketId", "commentId"},
		Example:          "openase ticket comment update $OPENASE_TICKET_ID $OPENASE_COMMENT_ID --body \"Updated progress\"",
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [ticketId] [commentId]", Short: "Delete a ticket comment.", Method: http.MethodDelete, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}", PositionalParams: []string{"ticketId", "commentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "revisions [ticketId] [commentId]", Short: "List ticket comment revisions.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}/revisions", PositionalParams: []string{"ticketId", "commentId"}}))
	return command
}

func newTypedTicketDependencyCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "dependency",
		Short: "Operate on ticket dependency relationships.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
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
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "delete [ticketId] [dependencyId]",
		Short:            "Delete a ticket dependency.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/tickets/{ticketId}/dependencies/{dependencyId}",
		PositionalParams: []string{"ticketId", "dependencyId"},
	}))
	return command
}

func newTypedTicketExternalLinkCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "external-link",
		Short: "Operate on ticket external links.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "add [ticketId]",
		Short:            "Add a ticket external link.",
		Method:           http.MethodPost,
		Path:             "/api/v1/tickets/{ticketId}/external-links",
		PositionalParams: []string{"ticketId"},
		HelpNotes: []string{
			"Use this to attach upstream issue, incident, document, or pull request references to a ticket.",
		},
		Example: `openase ticket external-link add $OPENASE_TICKET_ID --title "PR 482" --url https://github.com/acme/repo/pull/482`,
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "delete [ticketId] [externalLinkId]",
		Short:            "Delete a ticket external link.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/tickets/{ticketId}/external-links/{externalLinkId}",
		PositionalParams: []string{"ticketId", "externalLinkId"},
	}))
	return command
}

func newTypedTicketRunCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "run",
		Short: "Inspect ticket run history.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "list [projectId] [ticketId]",
		Short:            "List ticket runs.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/tickets/{ticketId}/runs",
		PositionalParams: []string{"projectId", "ticketId"},
		HelpNotes: []string{
			"Use this to inspect execution history, retry chains, and current runtime state for one ticket.",
		},
		Example: "openase ticket run list $OPENASE_PROJECT_ID $OPENASE_TICKET_ID",
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "get [projectId] [ticketId] [runId]",
		Short:            "Get a ticket run.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/tickets/{ticketId}/runs/{runId}",
		PositionalParams: []string{"projectId", "ticketId", "runId"},
		HelpNotes: []string{
			"This returns the stored runtime snapshot for one run, including status, lifecycle timestamps, and retry metadata.",
		},
		Example: "openase ticket run get $OPENASE_PROJECT_ID $OPENASE_TICKET_ID $OPENASE_RUN_ID",
	}))
	return command
}

func newProjectCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "project",
		Short: "Operate on projects through the OpenASE API.",
	}
	command.AddCommand(newProjectCurrentCommand())
	command.AddCommand(newProjectUpdatesCommand())
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [orgId]", Short: "List projects.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/projects", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [projectId]", Short: "Get a project.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [orgId]", Short: "Create a project.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/projects", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [projectId]", Short: "Update a project.", Method: http.MethodPatch, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [projectId]", Short: "Archive a project.", Method: http.MethodDelete, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}}))
	return command
}

func newProjectUpdatesCommand() *cobra.Command {
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
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "list [projectId]",
		Short:            "List project update threads.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/updates",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"If [projectId] is omitted, the command falls back to --project-id and then OPENASE_PROJECT_ID.",
		},
		Example: "openase project updates list\nopenase project updates list $OPENASE_PROJECT_ID",
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "create [projectId]",
		Short:            "Create a project update thread.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/updates",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"If [projectId] is omitted, the command falls back to --project-id and then OPENASE_PROJECT_ID.",
		},
		Example: "openase project updates create --status on_track --title \"CLI parity\" --body \"Implemented runtime-safe project and machine commands.\"",
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "update [projectId] [threadId]",
		Short:            "Update a project update thread.",
		Method:           http.MethodPatch,
		Path:             "/api/v1/projects/{projectId}/updates/{threadId}",
		PositionalParams: []string{"projectId", "threadId"},
		HelpNotes: []string{
			"You can pass --thread-id together with OPENASE_PROJECT_ID when you only want to provide the thread identifier explicitly.",
		},
		Example: "openase project updates update --thread-id $OPENASE_THREAD_ID --status at_risk --title \"CLI parity\" --body \"Waiting on review.\"",
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "delete [projectId] [threadId]",
		Short:            "Delete a project update thread.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/projects/{projectId}/updates/{threadId}",
		PositionalParams: []string{"projectId", "threadId"},
		HelpNotes: []string{
			"You can pass --thread-id together with OPENASE_PROJECT_ID when you only want to provide the thread identifier explicitly.",
		},
		Example: "openase project updates delete --thread-id $OPENASE_THREAD_ID",
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "revisions [projectId] [threadId]",
		Short:            "List project update thread revisions.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/updates/{threadId}/revisions",
		PositionalParams: []string{"projectId", "threadId"},
		HelpNotes: []string{
			"You can pass --thread-id together with OPENASE_PROJECT_ID when you only want to provide the thread identifier explicitly.",
		},
		Example: "openase project updates revisions --thread-id $OPENASE_THREAD_ID",
	}))
	return command
}

func newRepoCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "repo",
		Short: "Operate on project repositories through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "list [projectId]",
		Short:            "List project repositories.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/repos",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"Use this to inspect repositories currently bound to a project before wiring workflows, repo scopes, or GitHub imports.",
		},
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "create [projectId]",
		Short:            "Create a project repository.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/repos",
		PositionalParams: []string{"projectId"},
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "update [projectId] [repoId]",
		Short:            "Update a project repository.",
		Method:           http.MethodPatch,
		Path:             "/api/v1/projects/{projectId}/repos/{repoId}",
		PositionalParams: []string{"projectId", "repoId"},
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "delete [projectId] [repoId]",
		Short:            "Delete a project repository.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/projects/{projectId}/repos/{repoId}",
		PositionalParams: []string{"projectId", "repoId"},
	}))

	github := &cobra.Command{
		Use:   "github",
		Short: "Operate on GitHub-backed repository discovery for a project.",
	}
	github.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "namespaces [projectId]",
		Short:            "List GitHub namespaces available to the project's effective credential.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/github/namespaces",
		PositionalParams: []string{"projectId"},
	}))
	github.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "list [projectId]",
		Short:            "List GitHub repositories visible to the project's effective credential.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/github/repos",
		PositionalParams: []string{"projectId"},
	}))
	github.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "create [projectId]",
		Short:            "Create a GitHub repository using the project's effective credential.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/github/repos",
		PositionalParams: []string{"projectId"},
	}))
	command.AddCommand(github)

	scope := &cobra.Command{
		Use:   "scope",
		Short: "Operate on ticket repository scopes.",
	}
	scope.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "list [projectId] [ticketId]",
		Short:            "List ticket repository scopes.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes",
		PositionalParams: []string{"projectId", "ticketId"},
	}))
	scope.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "create [projectId] [ticketId]",
		Short:            "Create a ticket repository scope.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes",
		PositionalParams: []string{"projectId", "ticketId"},
	}))
	scope.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "update [projectId] [ticketId] [scopeId]",
		Short:            "Update a ticket repository scope.",
		Method:           http.MethodPatch,
		Path:             "/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes/{scopeId}",
		PositionalParams: []string{"projectId", "ticketId", "scopeId"},
	}))
	scope.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "delete [projectId] [ticketId] [scopeId]",
		Short:            "Delete a ticket repository scope.",
		Method:           http.MethodDelete,
		Path:             "/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes/{scopeId}",
		PositionalParams: []string{"projectId", "ticketId", "scopeId"},
	}))
	command.AddCommand(scope)

	return command
}

func newWorkflowCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "workflow",
		Short: "Operate on workflows through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [projectId]", Short: "List workflows.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/workflows", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [workflowId]", Short: "Get a workflow.", Method: http.MethodGet, Path: "/api/v1/workflows/{workflowId}", PositionalParams: []string{"workflowId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [projectId]", Short: "Create a workflow.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/workflows", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [workflowId]", Short: "Update a workflow.", Method: http.MethodPatch, Path: "/api/v1/workflows/{workflowId}", PositionalParams: []string{"workflowId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [workflowId]", Short: "Delete a workflow.", Method: http.MethodDelete, Path: "/api/v1/workflows/{workflowId}", PositionalParams: []string{"workflowId"}}))

	harness := &cobra.Command{
		Use:   "harness",
		Short: "Operate on workflow harness content.",
	}
	harness.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [workflowId]", Short: "Get workflow harness content.", Method: http.MethodGet, Path: "/api/v1/workflows/{workflowId}/harness", PositionalParams: []string{"workflowId"}}))
	harness.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "history [workflowId]",
		Short:            "List workflow harness revisions.",
		Method:           http.MethodGet,
		Path:             "/api/v1/workflows/{workflowId}/harness/history",
		PositionalParams: []string{"workflowId"},
		HelpNotes: []string{
			"Use this to audit harness edits and recover the exact stored revision sequence for one workflow.",
		},
		Example: "openase workflow harness history $OPENASE_WORKFLOW_ID",
	}))
	harness.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [workflowId]", Short: "Update workflow harness content.", Method: http.MethodPut, Path: "/api/v1/workflows/{workflowId}/harness", PositionalParams: []string{"workflowId"}}))
	harness.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:    "variables",
		Short:  "List harness variables.",
		Method: http.MethodGet,
		Path:   "/api/v1/harness/variables",
		HelpNotes: []string{
			"Use this to inspect the variable catalog available to workflow harness templates before editing or validating one.",
		},
		Example: "openase workflow harness variables",
	}))
	harness.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:    "validate",
		Short:  "Validate harness content.",
		Method: http.MethodPost,
		Path:   "/api/v1/harness/validate",
		HelpNotes: []string{
			"This validates harness markdown and structured references without mutating any stored workflow harness.",
		},
		Example: "openase workflow harness validate --input /tmp/harness.json",
	}))
	command.AddCommand(harness)
	return command
}

func newScheduledJobCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "scheduled-job",
		Short: "Operate on scheduled jobs through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [projectId]", Short: "List scheduled jobs.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/scheduled-jobs", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [projectId]", Short: "Create a scheduled job.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/scheduled-jobs", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [jobId]", Short: "Update a scheduled job.", Method: http.MethodPatch, Path: "/api/v1/scheduled-jobs/{jobId}", PositionalParams: []string{"jobId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [jobId]", Short: "Delete a scheduled job.", Method: http.MethodDelete, Path: "/api/v1/scheduled-jobs/{jobId}", PositionalParams: []string{"jobId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "trigger [jobId]", Short: "Trigger a scheduled job once.", Method: http.MethodPost, Path: "/api/v1/scheduled-jobs/{jobId}/trigger", PositionalParams: []string{"jobId"}}))
	return command
}

func newMachineCommand(options *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "machine",
		Short: "Operate on machines through the OpenASE API.",
		Long: strings.TrimSpace(`
Operate on machines through the OpenASE API.

Use this command group for machine CRUD, resource inspection, health refresh,
and machine-channel credential lifecycle operations. Machine IDs must be UUID
values. Reverse websocket daemon runtime entrypoints live under
` + "`openase machine-agent ...`" + `.
`),
	}
	command.AddCommand(newMachineListCommand())
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "get [machineId]",
		Short:            "Get a machine.",
		Method:           http.MethodGet,
		Path:             "/api/v1/machines/{machineId}",
		PositionalParams: []string{"machineId"},
		HelpNotes: []string{
			"This shows the current machine status, last heartbeat, workspace settings, and the latest stored resources for one machine.",
		},
		Example: strings.TrimSpace(`
  openase machine get $OPENASE_MACHINE_ID
  openase machine get 550e8400-e29b-41d4-a716-446655440000
`),
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [orgId]", Short: "Create a machine.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/machines", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [machineId]", Short: "Update a machine.", Method: http.MethodPatch, Path: "/api/v1/machines/{machineId}", PositionalParams: []string{"machineId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [machineId]", Short: "Delete a machine.", Method: http.MethodDelete, Path: "/api/v1/machines/{machineId}", PositionalParams: []string{"machineId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "resources [machineId]",
		Short:            "Get machine resources.",
		Method:           http.MethodGet,
		Path:             "/api/v1/machines/{machineId}/resources",
		PositionalParams: []string{"machineId"},
		HelpNotes: []string{
			"This returns the current stored resource snapshot. Run `openase machine refresh-health` first when you need a fresh probe before inspection.",
		},
		Example: strings.TrimSpace(`
  openase machine resources $OPENASE_MACHINE_ID
  openase machine resources 550e8400-e29b-41d4-a716-446655440000
`),
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "test [machineId]",
		Short:            "Test a machine connection.",
		Method:           http.MethodPost,
		Path:             "/api/v1/machines/{machineId}/test",
		PositionalParams: []string{"machineId"},
		HelpNotes: []string{
			"This runs an on-demand transport probe against the machine and returns the machine payload plus probe output for operator troubleshooting.",
		},
		Example: strings.TrimSpace(`
  openase machine test $OPENASE_MACHINE_ID
  openase machine test 550e8400-e29b-41d4-a716-446655440000
`),
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "refresh-health [machineId]",
		Short:            "Refresh machine health.",
		Method:           http.MethodPost,
		Path:             "/api/v1/machines/{machineId}/refresh-health",
		PositionalParams: []string{"machineId"},
		HelpNotes: []string{
			"This re-runs the machine health collector so machine status and provider availability can be observed from refreshed data before you inspect resources or providers.",
		},
		Example: strings.TrimSpace(`
  openase machine refresh-health $OPENASE_MACHINE_ID
  openase machine refresh-health 550e8400-e29b-41d4-a716-446655440000
`),
	}))
	command.AddCommand(newMachineStreamCommand())
	command.AddCommand(newMachineSSHBootstrapCommand(options))
	command.AddCommand(newMachineSSHDiagnosticsCommand())
	command.AddCommand(newMachineIssueChannelTokenCommand(options))
	command.AddCommand(newMachineRevokeChannelTokenCommand(options))
	return command
}

func newProjectCurrentCommand() *cobra.Command {
	var apiOptions apiCommandOptions
	var output apiOutputOptions
	var projectID string

	spec := openAPICommandSpec{
		Method: http.MethodGet,
		Path:   "/api/v1/projects/{projectId}",
	}
	command := &cobra.Command{
		Use:   "current",
		Short: "Get the current project.",
		Long: strings.TrimSpace(`
Get the current project.

This command resolves the target project from --project-id and then OPENASE_PROJECT_ID,
so project-scoped runtimes can inspect the active project without falling back to raw API calls.

The response includes organization_id, which can be used directly or bridged into
machine inspection with ` + "`openase machine list --project-id $OPENASE_PROJECT_ID`" + `.
`),
		Example: strings.TrimSpace(`
  openase project current
  openase project current --project-id $OPENASE_PROJECT_ID --json project.organization_id
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiContext, err := apiOptionsFromFlags(cmd.Flags()).resolveResource()
			if err != nil {
				return err
			}

			resolvedProjectID := strings.TrimSpace(firstNonEmpty(projectID, os.Getenv("OPENASE_PROJECT_ID")))
			if resolvedProjectID == "" {
				return fmt.Errorf("project id is required via --project-id or OPENASE_PROJECT_ID")
			}

			response, err := apiContext.do(cmd.Context(), apiCommandDeps{httpClient: http.DefaultClient}, apiRequest{
				Method: http.MethodGet,
				Path:   "projects/" + urlPathEscape(resolvedProjectID),
			})
			if err != nil {
				return err
			}
			return writeAPIOutput(cmd.OutOrStdout(), response.Body, outputOptionsFromFlags(cmd.Flags()))
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	bindAPICommandFlags(command.Flags(), &apiOptions)
	bindAPIOutputFlags(command.Flags(), &output)
	command.Flags().StringVar(&projectID, "project-id", "", "Project ID override. Defaults to OPENASE_PROJECT_ID.")
	return markCLICommandAPICoverageSpec(command, spec)
}

func newMachineListCommand() *cobra.Command {
	var apiOptions apiCommandOptions
	var output apiOutputOptions
	var orgID string
	var projectID string

	spec := openAPICommandSpec{
		Method: http.MethodGet,
		Path:   "/api/v1/orgs/{orgId}/machines",
	}
	command := &cobra.Command{
		Use:   "list [orgId]",
		Short: "List machines.",
		Long: strings.TrimSpace(`
List machines.

Use this to audit machine status, heartbeat freshness, workspace roots, and the latest cached
resource snapshot before scheduling work.

Positional [orgId] values must be UUIDs.

The command resolves organization scope in this order:
1. positional [orgId]
2. --org-id
3. OPENASE_ORG_ID
4. --project-id
5. OPENASE_PROJECT_ID

When project context is used, the CLI fetches the project first and derives organization_id automatically.
`),
		Example: strings.TrimSpace(`
  openase machine list $OPENASE_ORG_ID
  openase machine list --project-id $OPENASE_PROJECT_ID
  openase machine list 550e8400-e29b-41d4-a716-446655440000 --json machines
`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiContext, err := apiOptionsFromFlags(cmd.Flags()).resolveResource()
			if err != nil {
				return err
			}

			resolvedOrgID, err := resolveOrganizationIDForMachineCommand(
				cmd.Context(),
				apiCommandDeps{httpClient: http.DefaultClient},
				apiContext,
				firstNonEmpty(firstArg(args), orgID, os.Getenv("OPENASE_ORG_ID")),
				firstNonEmpty(projectID, os.Getenv("OPENASE_PROJECT_ID")),
			)
			if err != nil {
				return err
			}

			response, err := apiContext.do(cmd.Context(), apiCommandDeps{httpClient: http.DefaultClient}, apiRequest{
				Method: http.MethodGet,
				Path:   "orgs/" + urlPathEscape(resolvedOrgID) + "/machines",
			})
			if err != nil {
				return err
			}
			return writeAPIOutput(cmd.OutOrStdout(), response.Body, outputOptionsFromFlags(cmd.Flags()))
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	bindAPICommandFlags(command.Flags(), &apiOptions)
	bindAPIOutputFlags(command.Flags(), &output)
	command.Flags().StringVar(&orgID, "org-id", "", "Organization ID override. Defaults to OPENASE_ORG_ID.")
	command.Flags().StringVar(&projectID, "project-id", "", "Project ID fallback. Defaults to OPENASE_PROJECT_ID when organization scope is not set.")
	return markCLICommandAPICoverageSpec(command, spec)
}

func newMachineStreamCommand() *cobra.Command {
	var apiOptions apiCommandOptions
	var orgID string
	var projectID string

	spec := openAPICommandSpec{
		Method: http.MethodGet,
		Path:   "/api/v1/orgs/{orgId}/machines/stream",
	}
	command := &cobra.Command{
		Use:   "stream [orgId]",
		Short: "Stream organization machine events.",
		Long: strings.TrimSpace(`
Stream organization machine events.

This command opens the organization machine event stream and keeps the connection open until the
server closes it or you interrupt the process.

Positional [orgId] values must be UUIDs.

Organization scope resolves from [orgId], --org-id, OPENASE_ORG_ID, --project-id, and then
OPENASE_PROJECT_ID. When project context is used, the CLI derives organization_id from the project first.

Use Ctrl-C to stop the stream when running interactively.
`),
		Example: strings.TrimSpace(`
  openase machine stream $OPENASE_ORG_ID
  openase machine stream --project-id $OPENASE_PROJECT_ID
`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiContext, err := apiOptionsFromFlags(cmd.Flags()).resolveResource()
			if err != nil {
				return err
			}

			resolvedOrgID, err := resolveOrganizationIDForMachineCommand(
				cmd.Context(),
				apiCommandDeps{httpClient: http.DefaultClient},
				apiContext,
				firstNonEmpty(firstArg(args), orgID, os.Getenv("OPENASE_ORG_ID")),
				firstNonEmpty(projectID, os.Getenv("OPENASE_PROJECT_ID")),
			)
			if err != nil {
				return err
			}

			return apiContext.stream(cmd.Context(), apiCommandDeps{httpClient: http.DefaultClient}, apiStreamRequest{
				Method: http.MethodGet,
				Path:   "orgs/" + urlPathEscape(resolvedOrgID) + "/machines/stream",
			}, cmd.OutOrStdout())
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	bindAPICommandFlags(command.Flags(), &apiOptions)
	command.Flags().StringVar(&orgID, "org-id", "", "Organization ID override. Defaults to OPENASE_ORG_ID.")
	command.Flags().StringVar(&projectID, "project-id", "", "Project ID fallback. Defaults to OPENASE_PROJECT_ID when organization scope is not set.")
	return markCLICommandAPICoverageSpec(command, spec)
}

func resolveOrganizationIDForMachineCommand(
	ctx context.Context,
	deps apiCommandDeps,
	apiContext apiCommandContext,
	explicitOrgID string,
	projectID string,
) (string, error) {
	resolvedOrgID := strings.TrimSpace(explicitOrgID)
	if resolvedOrgID != "" {
		return resolvedOrgID, nil
	}

	resolvedProjectID := strings.TrimSpace(projectID)
	if resolvedProjectID == "" {
		return "", fmt.Errorf("organization id is required via positional argument, --org-id, OPENASE_ORG_ID, --project-id, or OPENASE_PROJECT_ID")
	}

	project, err := fetchCLIProject(ctx, deps, apiContext, resolvedProjectID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(project.Project.OrganizationID) == "" {
		return "", fmt.Errorf("project %s response did not include organization_id", resolvedProjectID)
	}
	return strings.TrimSpace(project.Project.OrganizationID), nil
}

func fetchCLIProject(ctx context.Context, deps apiCommandDeps, apiContext apiCommandContext, projectID string) (cliProjectResponseEnvelope, error) {
	response, err := apiContext.do(ctx, deps, apiRequest{
		Method: http.MethodGet,
		Path:   "projects/" + urlPathEscape(strings.TrimSpace(projectID)),
	})
	if err != nil {
		return cliProjectResponseEnvelope{}, err
	}

	var payload cliProjectResponseEnvelope
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return cliProjectResponseEnvelope{}, fmt.Errorf("decode project response: %w", err)
	}
	if strings.TrimSpace(payload.Project.ID) == "" {
		return cliProjectResponseEnvelope{}, fmt.Errorf("decode project response: missing project.id")
	}
	return payload, nil
}

func newProviderCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "provider",
		Short: "Operate on agent providers through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "list [orgId]",
		Short:            "List providers.",
		Method:           http.MethodGet,
		Path:             "/api/v1/orgs/{orgId}/providers",
		PositionalParams: []string{"orgId"},
		HelpNotes: []string{
			"Provider list responses include derived availability fields so operators can audit whether each provider is runnable on its machine without reading source code.",
		},
		Example: strings.TrimSpace(`
  openase provider list $OPENASE_ORG_ID
  openase provider list 550e8400-e29b-41d4-a716-446655440000 --json providers
`),
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "get [providerId]",
		Short:            "Get a provider.",
		Method:           http.MethodGet,
		Path:             "/api/v1/providers/{providerId}",
		PositionalParams: []string{"providerId"},
		HelpNotes: []string{
			"This shows one provider with derived availability_state, available, availability_reason, and backing machine metadata so provider health is directly inspectable from the CLI.",
		},
		Example: strings.TrimSpace(`
  openase provider get $OPENASE_PROVIDER_ID
  openase provider get 550e8400-e29b-41d4-a716-446655440000
`),
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [orgId]", Short: "Create a provider.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/providers", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [providerId]", Short: "Update a provider.", Method: http.MethodPatch, Path: "/api/v1/providers/{providerId}", PositionalParams: []string{"providerId"}}))
	return command
}

func newAgentCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "agent",
		Short: "Operate on agents through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [projectId]", Short: "List agents.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [agentId]", Short: "Get an agent.", Method: http.MethodGet, Path: "/api/v1/agents/{agentId}", PositionalParams: []string{"agentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [projectId]", Short: "Create an agent.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/agents", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [agentId]", Short: "Update an agent.", Method: http.MethodPatch, Path: "/api/v1/agents/{agentId}", PositionalParams: []string{"agentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [agentId]", Short: "Delete an agent.", Method: http.MethodDelete, Path: "/api/v1/agents/{agentId}", PositionalParams: []string{"agentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "pause [agentId]", Short: "Pause an agent.", Method: http.MethodPost, Path: "/api/v1/agents/{agentId}/pause", PositionalParams: []string{"agentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "resume [agentId]", Short: "Resume an agent.", Method: http.MethodPost, Path: "/api/v1/agents/{agentId}/resume", PositionalParams: []string{"agentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "output [projectId] [agentId]", Short: "List agent output entries.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/output", PositionalParams: []string{"projectId", "agentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "steps [projectId] [agentId]", Short: "List agent step entries.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/steps", PositionalParams: []string{"projectId", "agentId"}}))
	return command
}

func newChannelCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "channel",
		Short: "Operate on notification channels through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [orgId]", Short: "List notification channels.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/channels", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [orgId]", Short: "Create a notification channel.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/channels", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [channelId]", Short: "Update a notification channel.", Method: http.MethodPatch, Path: "/api/v1/channels/{channelId}", PositionalParams: []string{"channelId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [channelId]", Short: "Delete a notification channel.", Method: http.MethodDelete, Path: "/api/v1/channels/{channelId}", PositionalParams: []string{"channelId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "test [channelId]",
		Short:            "Test a notification channel.",
		Method:           http.MethodPost,
		Path:             "/api/v1/channels/{channelId}/test",
		PositionalParams: []string{"channelId"},
		HelpNotes: []string{
			"Use this after create or update to verify the destination accepts deliveries before you bind project notification rules to it.",
		},
	}))
	return command
}

func newNotificationRuleCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "notification-rule",
		Short: "Operate on notification rules through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [projectId]", Short: "List project notification rules.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/notification-rules", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newRawBodyOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "create [projectId]",
		Short:            "Create a notification rule.",
		Method:           http.MethodPost,
		Path:             "/api/v1/projects/{projectId}/notification-rules",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"Use repeated -f/--field entries or --input because this request body includes a field named template, which would otherwise collide with the CLI output templating flag.",
		},
	}))
	command.AddCommand(newRawBodyOpenAPIOperationCommand(openAPICommandSpec{
		Use:              "update [ruleId]",
		Short:            "Update a notification rule.",
		Method:           http.MethodPatch,
		Path:             "/api/v1/notification-rules/{ruleId}",
		PositionalParams: []string{"ruleId"},
		HelpNotes: []string{
			"Use repeated -f/--field entries or --input because this request body includes a field named template, which would otherwise collide with the CLI output templating flag.",
		},
	}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [ruleId]", Short: "Delete a notification rule.", Method: http.MethodDelete, Path: "/api/v1/notification-rules/{ruleId}", PositionalParams: []string{"ruleId"}}))
	return command
}

func newSkillCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "skill",
		Short: "Operate on skills through the OpenASE API.",
	}
	command.AddCommand(newSkillImportCommand())
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [projectId]", Short: "List project skills.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/skills", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [projectId]", Short: "Create a skill.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/skills", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "refresh [projectId]", Short: "Refresh workspace skills.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/skills/refresh", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [skillId]", Short: "Get a skill.", Method: http.MethodGet, Path: "/api/v1/skills/{skillId}", PositionalParams: []string{"skillId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [skillId]", Short: "Update a skill.", Method: http.MethodPut, Path: "/api/v1/skills/{skillId}", PositionalParams: []string{"skillId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [skillId]", Short: "Delete a skill.", Method: http.MethodDelete, Path: "/api/v1/skills/{skillId}", PositionalParams: []string{"skillId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "enable [skillId]", Short: "Enable a skill.", Method: http.MethodPost, Path: "/api/v1/skills/{skillId}/enable", PositionalParams: []string{"skillId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "disable [skillId]", Short: "Disable a skill.", Method: http.MethodPost, Path: "/api/v1/skills/{skillId}/disable", PositionalParams: []string{"skillId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "bind [skillId]", Short: "Bind a skill to workflows.", Method: http.MethodPost, Path: "/api/v1/skills/{skillId}/bind", PositionalParams: []string{"skillId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "unbind [skillId]", Short: "Unbind a skill from workflows.", Method: http.MethodPost, Path: "/api/v1/skills/{skillId}/unbind", PositionalParams: []string{"skillId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "bind-workflow [workflowId]", Short: "Bind skills to a workflow.", Method: http.MethodPost, Path: "/api/v1/workflows/{workflowId}/skills/bind", PositionalParams: []string{"workflowId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "unbind-workflow [workflowId]", Short: "Unbind skills from a workflow.", Method: http.MethodPost, Path: "/api/v1/workflows/{workflowId}/skills/unbind", PositionalParams: []string{"workflowId"}}))
	return command
}

func newWatchCommand() *cobra.Command {
	return newStreamNamespaceCommand("watch", "Operate on real-time watch endpoints.")
}

func newStreamCommand() *cobra.Command {
	return newStreamNamespaceCommand("stream", "Operate on real-time stream endpoints.")
}

func newStreamNamespaceCommand(use string, short string) *cobra.Command {
	command := &cobra.Command{
		Use:   use,
		Short: short,
	}
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{
		Use:    "events",
		Short:  "Stream system events.",
		Method: http.MethodGet,
		Path:   "/api/v1/events/stream",
		HelpNotes: []string{
			"Use this first-class stream entrypoint for operator observation. Machine and provider lifecycle updates flow through the global event stream until dedicated resource-specific streams exist.",
		},
		Example: "openase " + use + " events",
	}))
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{
		Use:              "project [projectId]",
		Short:            "Stream the passive project event bus.",
		Method:           http.MethodGet,
		Path:             "/api/v1/projects/{projectId}/events/stream",
		PositionalParams: []string{"projectId"},
		HelpNotes: []string{
			"Passive project UI state must converge on this single stream entrypoint; tickets, agents, activity, hooks, and ticket-run transcript updates are multiplexed here.",
		},
		Example: "openase " + use + " project $OPENASE_PROJECT_ID",
	}))
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{Use: "output [projectId] [agentId]", Short: "Stream agent output.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/output/stream", PositionalParams: []string{"projectId", "agentId"}, Example: "openase " + use + " output $OPENASE_PROJECT_ID $OPENASE_AGENT_ID"}))
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{Use: "steps [projectId] [agentId]", Short: "Stream agent steps.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/steps/stream", PositionalParams: []string{"projectId", "agentId"}, Example: "openase " + use + " steps $OPENASE_PROJECT_ID $OPENASE_AGENT_ID"}))
	return command
}

func newOpenAPIOperationCommand(spec openAPICommandSpec) *cobra.Command {
	contract := mustOpenAPICommandContract(spec)
	deps := apiCommandDeps{httpClient: http.DefaultClient}
	command := &cobra.Command{
		Use:     spec.Use,
		Short:   contract.summary,
		Long:    buildOpenAPIOperationHelp(spec, contract.summary),
		Example: strings.TrimSpace(spec.Example),
		Args:    cobra.MaximumNArgs(len(spec.PositionalParams)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpenAPIOperationCommand(cmd, deps, contract, args)
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	registerOpenAPICommandFlags(command.Flags(), contract)
	return markCLICommandAPICoverageSpec(command, spec)
}

func newRawBodyOpenAPIOperationCommand(spec openAPICommandSpec) *cobra.Command {
	contract := mustOpenAPICommandContract(spec)
	deps := apiCommandDeps{httpClient: http.DefaultClient}
	var fields []string
	var inputPath string
	command := &cobra.Command{
		Use:     spec.Use,
		Short:   contract.summary,
		Long:    buildOpenAPIOperationHelp(spec, contract.summary),
		Example: strings.TrimSpace(spec.Example),
		Args:    cobra.MaximumNArgs(len(spec.PositionalParams)),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiContext, err := apiOptionsFromFlags(cmd.Flags()).resolveResource()
			if err != nil {
				return err
			}
			requestPath, err := resolveOpenAPIRequestPath(cmd, contract, args)
			if err != nil {
				return err
			}
			body, err := buildRequestBody(fields, inputPath)
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
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	bindAPICommandFlags(command.Flags(), &apiCommandOptions{})
	bindAPIOutputFlags(command.Flags(), &apiOutputOptions{})
	command.Flags().StringSliceVarP(&fields, "field", "f", nil, "Add a JSON body field as key=value. Repeat for multiple fields.")
	command.Flags().StringVar(&inputPath, "input", "", "Read the raw request body from a file. Use - for stdin.")
	return markCLICommandAPICoverageSpec(markCLICommandRawBodyProxy(command), spec)
}

func buildOpenAPIOperationHelp(spec openAPICommandSpec, summary string) string {
	lines := []string{summary}
	if len(spec.PositionalParams) == 0 {
		lines = append(lines, spec.HelpNotes...)
		return strings.Join(lines, "\n\n")
	}

	uuidParams := make([]string, 0, len(spec.PositionalParams))
	for _, name := range spec.PositionalParams {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), "id") {
			uuidParams = append(uuidParams, name)
		}
	}
	if len(uuidParams) > 0 {
		lines = append(lines, fmt.Sprintf(
			"Positional parameter(s) %s must be UUID values unless the help text for that command says otherwise. Human-readable identifiers such as ASE-2 are not accepted for %s.",
			strings.Join(uuidParams, ", "),
			strings.Join(uuidParams, ", "),
		))
	}
	lines = append(lines, spec.HelpNotes...)
	return strings.Join(lines, "\n\n")
}

func newOpenAPIStreamCommand(spec openAPICommandSpec) *cobra.Command {
	contract := mustOpenAPICommandContract(spec)
	deps := apiCommandDeps{httpClient: http.DefaultClient}
	command := &cobra.Command{
		Use:     spec.Use,
		Short:   contract.summary,
		Long:    buildOpenAPIStreamHelp(spec, contract.summary),
		Example: firstNonEmpty(strings.TrimSpace(spec.Example), "openase watch "+spec.Use),
		Args:    cobra.MaximumNArgs(len(spec.PositionalParams)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpenAPIStreamCommand(cmd, deps, contract, args)
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	registerOpenAPIStreamFlags(command.Flags(), contract)
	return markCLICommandAPICoverageSpec(command, spec)
}

func buildOpenAPIStreamHelp(spec openAPICommandSpec, summary string) string {
	lines := []string{
		summary,
		"This command opens a streaming API endpoint and keeps the connection open until the server closes it or you interrupt the process.",
	}

	uuidParams := make([]string, 0, len(spec.PositionalParams))
	for _, name := range spec.PositionalParams {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(name)), "id") {
			uuidParams = append(uuidParams, name)
		}
	}
	if len(uuidParams) > 0 {
		lines = append(lines, fmt.Sprintf(
			"Positional parameter(s) %s must be UUID values unless the help text for that command says otherwise. Human-readable identifiers such as ASE-2 are not accepted for %s.",
			strings.Join(uuidParams, ", "),
			strings.Join(uuidParams, ", "),
		))
	}

	lines = append(lines, spec.HelpNotes...)
	lines = append(lines, "Use Ctrl-C to stop the stream when running interactively.")
	return strings.Join(lines, "\n\n")
}

func runOpenAPIOperationCommand(cmd *cobra.Command, deps apiCommandDeps, contract openAPICommandContract, args []string) error {
	apiContext, err := apiOptionsFromFlags(cmd.Flags()).resolveResource()
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

func runOpenAPIStreamCommand(cmd *cobra.Command, deps apiCommandDeps, contract openAPICommandContract, args []string) error {
	apiContext, err := apiOptionsFromFlags(cmd.Flags()).resolveResource()
	if err != nil {
		return err
	}
	requestPath, err := resolveOpenAPIRequestPath(cmd, contract, args)
	if err != nil {
		return err
	}
	return apiContext.stream(cmd.Context(), deps, apiStreamRequest{
		Method: contract.spec.Method,
		Path:   requestPath,
	}, cmd.OutOrStdout())
}

func registerOpenAPICommandFlags(flags *pflag.FlagSet, contract openAPICommandContract) {
	for _, field := range contract.pathParams {
		registerFieldFlag(flags, field, false)
	}
	for _, field := range contract.queryParams {
		registerFieldFlag(flags, field, false)
	}
	for _, field := range contract.bodyFields {
		registerFieldFlag(flags, field, true)
	}

	var apiOptions apiCommandOptions
	var output apiOutputOptions
	var input string
	bindAPICommandFlags(flags, &apiOptions)
	bindAPIOutputFlags(flags, &output)
	if contract.hasBody {
		flags.StringVar(&input, "input", "", "Read the raw JSON request body from a file. Use - for stdin.")
	}
}

func registerOpenAPIStreamFlags(flags *pflag.FlagSet, contract openAPICommandContract) {
	for _, field := range contract.pathParams {
		registerFieldFlag(flags, field, false)
	}
	for _, field := range contract.queryParams {
		registerFieldFlag(flags, field, false)
	}
	var apiOptions apiCommandOptions
	bindAPICommandFlags(flags, &apiOptions)
}

func registerFieldFlag(flags *pflag.FlagSet, field openAPIInputField, bodyField bool) {
	registerFieldFlagName(flags, field, field.Name)
	if bodyField {
		annotateCLIFlagBodyFields(flags.Lookup(field.Name), field.Name)
	}
	for _, alias := range fieldFlagAliases(field.Name) {
		registerFieldFlagName(flags, field, alias)
		if bodyField {
			annotateCLIFlagBodyFields(flags.Lookup(alias), field.Name)
		}
		_ = flags.MarkHidden(alias)
	}
}

func registerFieldFlagName(flags *pflag.FlagSet, field openAPIInputField, name string) {
	switch field.Kind {
	case flagValueString:
		flags.String(name, "", field.Description)
	case flagValueStringSlice:
		flags.StringSlice(name, nil, field.Description)
	case flagValueInt64:
		flags.Int64(name, 0, field.Description)
	case flagValueFloat64:
		flags.Float64(name, 0, field.Description)
	case flagValueBool:
		flags.Bool(name, false, field.Description)
	default:
		flags.String(name, "", field.Description)
	}
}

func resolveOpenAPIRequestPath(cmd *cobra.Command, contract openAPICommandContract, args []string) (string, error) {
	path := strings.TrimPrefix(contract.spec.Path, "/api/v1/")
	positionalIndex := make(map[string]int, len(contract.spec.PositionalParams))
	for index, name := range contract.spec.PositionalParams {
		positionalIndex[name] = index
	}

	for _, field := range contract.pathParams {
		index, ok := positionalIndex[field.Name]
		value, err := resolveCommandPathValue(cmd, field.Name, args, index)
		if err != nil {
			return "", err
		}
		if !ok {
			value, err = resolveCommandFlagValue(cmd, field)
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(value) == "" {
				value = strings.TrimSpace(os.Getenv(envVarForParameter(field.Name)))
			}
		}
		if strings.TrimSpace(value) == "" && field.Required {
			return "", fmt.Errorf("%s is required via positional argument, --%s, or %s", field.Name, field.Name, envVarForParameter(field.Name))
		}
		path = strings.ReplaceAll(path, "{"+field.Name+"}", urlPathEscape(strings.TrimSpace(value)))
	}

	queryParts := make([]string, 0, len(contract.queryParams))
	for _, field := range contract.queryParams {
		encoded, include, err := resolveCommandQueryValue(cmd, field)
		if err != nil {
			return "", err
		}
		if include {
			queryParts = append(queryParts, field.Name+"="+encoded)
		}
	}
	if len(queryParts) > 0 {
		path += "?" + strings.Join(queryParts, "&")
	}
	return path, nil
}

func resolveOpenAPIBody(cmd *cobra.Command, contract openAPICommandContract) ([]byte, error) {
	if !contract.hasBody {
		return nil, nil
	}

	inputPath, _ := cmd.Flags().GetString("input")
	bodyFieldsSet := 0
	for _, field := range contract.bodyFields {
		if isFieldSet(cmd.Flags(), field) {
			bodyFieldsSet++
		}
	}
	if strings.TrimSpace(inputPath) != "" {
		if bodyFieldsSet > 0 {
			return nil, fmt.Errorf("--input cannot be combined with body field flags")
		}
		return readInputFile(strings.TrimSpace(inputPath))
	}

	payload := map[string]any{}
	for _, field := range contract.bodyFields {
		if !isFieldSet(cmd.Flags(), field) {
			continue
		}
		value, err := readFieldValue(cmd.Flags(), field)
		if err != nil {
			return nil, err
		}
		payload[field.Name] = value
	}
	if len(payload) == 0 {
		return nil, nil
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request payload: %w", err)
	}
	return encoded, nil
}

func resolveCommandPathValue(cmd *cobra.Command, name string, args []string, index int) (string, error) {
	if index >= 0 && index < len(args) {
		if trimmed := strings.TrimSpace(args[index]); trimmed != "" {
			return trimmed, nil
		}
	}

	value, err := resolveCommandFlagValue(cmd, openAPIInputField{Name: name, Kind: flagValueString})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value), nil
	}

	if envValue := strings.TrimSpace(os.Getenv(envVarForParameter(name))); envValue != "" {
		return envValue, nil
	}
	return "", nil
}

func resolveCommandFlagValue(cmd *cobra.Command, field openAPIInputField) (string, error) {
	flags := cmd.Flags()
	names := append([]string{field.Name}, fieldFlagAliases(field.Name)...)
	switch field.Kind {
	case flagValueInt64:
		for _, name := range names {
			value, err := flags.GetInt64(name)
			if err != nil {
				return "", err
			}
			if flags.Changed(name) {
				return fmt.Sprintf("%d", value), nil
			}
		}
		return "", nil
	case flagValueFloat64:
		for _, name := range names {
			value, err := flags.GetFloat64(name)
			if err != nil {
				return "", err
			}
			if flags.Changed(name) {
				return strings.TrimSpace(strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", value), "0"), ".")), nil
			}
		}
		return "", nil
	case flagValueBool:
		for _, name := range names {
			value, err := flags.GetBool(name)
			if err != nil {
				return "", err
			}
			if !flags.Changed(name) {
				continue
			}
			if value {
				return "true", nil
			}
			return "false", nil
		}
		return "", nil
	case flagValueStringSlice:
		for _, name := range names {
			values, err := flags.GetStringSlice(name)
			if err != nil {
				return "", err
			}
			if flags.Changed(name) {
				return strings.Join(trimNonEmpty(values), ","), nil
			}
		}
		return "", nil
	default:
		for _, name := range names {
			value, err := flags.GetString(name)
			if err != nil {
				return "", err
			}
			if flags.Changed(name) {
				return strings.TrimSpace(value), nil
			}
		}
		value, err := flags.GetString(field.Name)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(value), nil
	}
}

func resolveCommandQueryValue(cmd *cobra.Command, field openAPIInputField) (string, bool, error) {
	if !isFieldSet(cmd.Flags(), field) {
		return "", false, nil
	}
	value, err := resolveCommandFlagValue(cmd, field)
	if err != nil {
		return "", false, err
	}
	return urlQueryEscape(value), true, nil
}

func isFieldSet(flags *pflag.FlagSet, field openAPIInputField) bool {
	for _, name := range append([]string{field.Name}, fieldFlagAliases(field.Name)...) {
		if flags.Changed(name) {
			return true
		}
	}
	return false
}

func readFieldValue(flags *pflag.FlagSet, field openAPIInputField) (any, error) {
	switch field.Kind {
	case flagValueInt64:
		for _, name := range append([]string{field.Name}, fieldFlagAliases(field.Name)...) {
			if flags.Changed(name) {
				return flags.GetInt64(name)
			}
		}
		return flags.GetInt64(field.Name)
	case flagValueFloat64:
		for _, name := range append([]string{field.Name}, fieldFlagAliases(field.Name)...) {
			if flags.Changed(name) {
				return flags.GetFloat64(name)
			}
		}
		return flags.GetFloat64(field.Name)
	case flagValueBool:
		for _, name := range append([]string{field.Name}, fieldFlagAliases(field.Name)...) {
			if flags.Changed(name) {
				return flags.GetBool(name)
			}
		}
		return flags.GetBool(field.Name)
	case flagValueStringSlice:
		for _, name := range append([]string{field.Name}, fieldFlagAliases(field.Name)...) {
			if !flags.Changed(name) {
				continue
			}
			values, err := flags.GetStringSlice(name)
			if err != nil {
				return nil, err
			}
			return trimNonEmpty(values), nil
		}
		values, err := flags.GetStringSlice(field.Name)
		if err != nil {
			return nil, err
		}
		return trimNonEmpty(values), nil
	default:
		for _, name := range append([]string{field.Name}, fieldFlagAliases(field.Name)...) {
			if !flags.Changed(name) {
				continue
			}
			value, err := flags.GetString(name)
			if err != nil {
				return nil, err
			}
			return strings.TrimSpace(value), nil
		}
		value, err := flags.GetString(field.Name)
		if err != nil {
			return nil, err
		}
		return strings.TrimSpace(value), nil
	}
}

func fieldFlagAliases(name string) []string {
	snake := snakeCaseParameterName(name)
	if snake == "" || snake == name {
		return nil
	}
	return []string{snake}
}

func snakeCaseParameterName(name string) string {
	var builder strings.Builder
	for index, r := range strings.TrimSpace(name) {
		switch {
		case r == '-' || r == '_':
			builder.WriteByte('_')
		case unicode.IsUpper(r):
			if index > 0 {
				builder.WriteByte('_')
			}
			builder.WriteRune(unicode.ToLower(r))
		default:
			builder.WriteRune(unicode.ToLower(r))
		}
	}
	return builder.String()
}

func parseHeaderPairs(values []string) (http.Header, error) {
	headers := make(http.Header)
	for _, item := range values {
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid header %q, want key:value", item)
		}
		headers.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
	return headers, nil
}

func buildRequestBody(fields []string, inputPath string) ([]byte, error) {
	if strings.TrimSpace(inputPath) != "" {
		if len(fields) > 0 {
			return nil, fmt.Errorf("--input cannot be combined with -f/--field")
		}
		return readInputFile(strings.TrimSpace(inputPath))
	}
	if len(fields) == 0 {
		return nil, nil
	}

	payload := make(map[string]any, len(fields))
	for _, item := range fields {
		key, value, err := parseKeyValue(item)
		if err != nil {
			return nil, err
		}
		payload[key] = coerceJSONValue(value)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request payload: %w", err)
	}
	return encoded, nil
}

func buildRawRequestPath(path string, queryItems []string) (string, error) {
	if len(queryItems) == 0 {
		return path, nil
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	queryParts := make([]string, 0, len(queryItems))
	for _, item := range queryItems {
		key, value, err := parseKeyValue(item)
		if err != nil {
			return "", err
		}
		queryParts = append(queryParts, urlQueryEscape(key)+"="+urlQueryEscape(value))
	}
	return path + separator + strings.Join(queryParts, "&"), nil
}

func parseKeyValue(value string) (string, string, error) {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return "", "", fmt.Errorf("invalid key=value pair %q", value)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func coerceJSONValue(value string) any {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err == nil {
		return decoded
	}
	return value
}

func readInputFile(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	// #nosec G304 -- CLI users explicitly choose the input file path.
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read input file %s: %w", path, err)
	}
	return body, nil
}

func mustOpenAPICommandContract(spec openAPICommandSpec) openAPICommandContract {
	contracts, err := loadOpenAPICommandContracts()
	if err != nil {
		panic(err)
	}
	contract, ok := contracts[contractKey(spec.Method, spec.Path)]
	if !ok {
		panic(fmt.Errorf("missing OpenAPI command contract for %s %s", spec.Method, spec.Path))
	}
	if contract.spec.Use != spec.Use {
		contract.spec.Use = spec.Use
	}
	if len(spec.PositionalParams) > 0 {
		contract.spec.PositionalParams = append([]string(nil), spec.PositionalParams...)
	}
	return contract
}

func loadOpenAPICommandContracts() (map[string]openAPICommandContract, error) {
	cliContractOnce.Do(func() {
		doc, err := httpapi.BuildOpenAPIDocument()
		if err != nil {
			cliContractErr = err
			return
		}
		contracts := make(map[string]openAPICommandContract)
		for _, spec := range allOpenAPICommandSpecs() {
			contract, buildErr := buildOpenAPICommandContract(doc, spec)
			if buildErr != nil {
				cliContractErr = buildErr
				return
			}
			contracts[contractKey(spec.Method, spec.Path)] = contract
		}
		cliContractData = contracts
	})
	return cliContractData, cliContractErr
}

func buildOpenAPICommandContract(doc *openapi3.T, spec openAPICommandSpec) (openAPICommandContract, error) {
	pathItem := doc.Paths.Find(spec.Path)
	if pathItem == nil {
		return openAPICommandContract{}, fmt.Errorf("OpenAPI path %s not found", spec.Path)
	}
	operation := pathItem.GetOperation(spec.Method)
	if operation == nil {
		return openAPICommandContract{}, fmt.Errorf("OpenAPI operation %s %s not found", spec.Method, spec.Path)
	}

	contract := openAPICommandContract{
		spec:        spec,
		operationID: operation.OperationID,
		summary:     firstNonEmpty(spec.Short, operation.Summary),
	}
	for _, parameter := range append(pathItem.Parameters, operation.Parameters...) {
		if parameter == nil || parameter.Value == nil {
			continue
		}
		field, err := openAPIFieldFromParameter(parameter.Value)
		if err != nil {
			return openAPICommandContract{}, fmt.Errorf("%s %s parameter %s: %w", spec.Method, spec.Path, parameter.Value.Name, err)
		}
		switch parameter.Value.In {
		case "path":
			contract.pathParams = append(contract.pathParams, field)
		case "query":
			contract.queryParams = append(contract.queryParams, field)
		}
	}

	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		mediaType := operation.RequestBody.Value.Content.Get("application/json")
		if mediaType != nil && mediaType.Schema != nil && mediaType.Schema.Value != nil {
			contract.hasBody = true
			fields, required := openAPIFieldsFromSchema(mediaType.Schema.Value)
			contract.bodyFields = fields
			contract.requiredBody = required
		}
	}

	sort.Slice(contract.pathParams, func(i, j int) bool { return contract.pathParams[i].Name < contract.pathParams[j].Name })
	sort.Slice(contract.queryParams, func(i, j int) bool { return contract.queryParams[i].Name < contract.queryParams[j].Name })
	sort.Slice(contract.bodyFields, func(i, j int) bool { return contract.bodyFields[i].Name < contract.bodyFields[j].Name })
	return contract, nil
}

func openAPIFieldFromParameter(parameter *openapi3.Parameter) (openAPIInputField, error) {
	kind, err := schemaKind(parameter.Schema)
	if err != nil {
		return openAPIInputField{}, err
	}
	if parameter.In == "query" && kind == flagValueString {
		kind = flagValueStringSlice
	}
	return openAPIInputField{
		Name:        parameter.Name,
		Description: parameter.Description,
		Required:    parameter.Required,
		Kind:        kind,
	}, nil
}

func openAPIFieldsFromSchema(schema *openapi3.Schema) ([]openAPIInputField, []string) {
	if schema == nil {
		return nil, nil
	}
	if !schema.Type.Is("object") {
		return nil, nil
	}
	required := make(map[string]struct{}, len(schema.Required))
	for _, name := range schema.Required {
		required[name] = struct{}{}
	}

	fields := make([]openAPIInputField, 0, len(schema.Properties))
	for name, property := range schema.Properties {
		if property == nil || property.Value == nil {
			continue
		}
		kind, err := schemaKind(property)
		if err != nil {
			continue
		}
		_, isRequired := required[name]
		fields = append(fields, openAPIInputField{
			Name:        name,
			Required:    isRequired,
			Kind:        kind,
			Description: property.Value.Description,
		})
	}

	requiredNames := make([]string, 0, len(schema.Required))
	requiredNames = append(requiredNames, schema.Required...)
	sort.Strings(requiredNames)
	return fields, requiredNames
}

func schemaKind(schema *openapi3.SchemaRef) (flagValueKind, error) {
	if schema == nil || schema.Value == nil {
		return flagValueString, nil
	}
	switch {
	case schema.Value.Type.Is("boolean"):
		return flagValueBool, nil
	case schema.Value.Type.Is("integer"):
		return flagValueInt64, nil
	case schema.Value.Type.Is("number"):
		return flagValueFloat64, nil
	case schema.Value.Type.Is("array"):
		if schema.Value.Items != nil && schema.Value.Items.Value != nil && schema.Value.Items.Value.Type.Is("string") {
			return flagValueStringSlice, nil
		}
		return "", fmt.Errorf("unsupported array schema")
	case schema.Value.Type.Is("string"):
		return flagValueString, nil
	case schema.Value.Type.Is("object"):
		return "", fmt.Errorf("unsupported object schema")
	default:
		return flagValueString, nil
	}
}

func allOpenAPICommandSpecs() []openAPICommandSpec {
	return []openAPICommandSpec{
		{Use: "list [projectId]", Short: "List tickets.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets", PositionalParams: []string{"projectId"}},
		{Use: "archived [projectId]", Short: "List archived tickets.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets/archived", PositionalParams: []string{"projectId"}},
		{Use: "get [ticketId]", Short: "Get a ticket.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}", PositionalParams: []string{"ticketId"}},
		{Use: "create [projectId]", Short: "Create a ticket.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/tickets", PositionalParams: []string{"projectId"}},
		{Use: "update [ticketId]", Short: "Update a ticket.", Method: http.MethodPatch, Path: "/api/v1/tickets/{ticketId}", PositionalParams: []string{"ticketId"}},
		{Use: "detail [projectId] [ticketId]", Short: "Get ticket detail.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets/{ticketId}/detail", PositionalParams: []string{"projectId", "ticketId"}},
		{Use: "retry-resume [ticketId]", Short: "Resume a ticket after a retryable failure.", Method: http.MethodPost, Path: "/api/v1/tickets/{ticketId}/retry/resume", PositionalParams: []string{"ticketId"}},
		{Use: "list [ticketId]", Short: "List ticket comments.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}/comments", PositionalParams: []string{"ticketId"}},
		{Use: "create [ticketId]", Short: "Create a ticket comment.", Method: http.MethodPost, Path: "/api/v1/tickets/{ticketId}/comments", PositionalParams: []string{"ticketId"}},
		{Use: "update [ticketId] [commentId]", Short: "Update a ticket comment.", Method: http.MethodPatch, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}", PositionalParams: []string{"ticketId", "commentId"}},
		{Use: "delete [ticketId] [commentId]", Short: "Delete a ticket comment.", Method: http.MethodDelete, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}", PositionalParams: []string{"ticketId", "commentId"}},
		{Use: "revisions [ticketId] [commentId]", Short: "List ticket comment revisions.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}/revisions", PositionalParams: []string{"ticketId", "commentId"}},
		{Use: "add [ticketId]", Short: "Add a ticket dependency.", Method: http.MethodPost, Path: "/api/v1/tickets/{ticketId}/dependencies", PositionalParams: []string{"ticketId"}},
		{Use: "delete [ticketId] [dependencyId]", Short: "Delete a ticket dependency.", Method: http.MethodDelete, Path: "/api/v1/tickets/{ticketId}/dependencies/{dependencyId}", PositionalParams: []string{"ticketId", "dependencyId"}},
		{Use: "add [ticketId]", Short: "Add a ticket external link.", Method: http.MethodPost, Path: "/api/v1/tickets/{ticketId}/external-links", PositionalParams: []string{"ticketId"}},
		{Use: "delete [ticketId] [externalLinkId]", Short: "Delete a ticket external link.", Method: http.MethodDelete, Path: "/api/v1/tickets/{ticketId}/external-links/{externalLinkId}", PositionalParams: []string{"ticketId", "externalLinkId"}},
		{Use: "list [projectId] [ticketId]", Short: "List ticket runs.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets/{ticketId}/runs", PositionalParams: []string{"projectId", "ticketId"}},
		{Use: "get [projectId] [ticketId] [runId]", Short: "Get a ticket run.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets/{ticketId}/runs/{runId}", PositionalParams: []string{"projectId", "ticketId", "runId"}},
		{Use: "list [projectId]", Short: "List ticket statuses.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/statuses", PositionalParams: []string{"projectId"}},
		{Use: "list [projectId]", Short: "List project activity events.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/activity", PositionalParams: []string{"projectId"}},
		{Use: "create [projectId]", Short: "Create a ticket status.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/statuses", PositionalParams: []string{"projectId"}},
		{Use: "update [statusId]", Short: "Update a ticket status.", Method: http.MethodPatch, Path: "/api/v1/statuses/{statusId}", PositionalParams: []string{"statusId"}},
		{Use: "delete [statusId]", Short: "Delete a ticket status.", Method: http.MethodDelete, Path: "/api/v1/statuses/{statusId}", PositionalParams: []string{"statusId"}},
		{Use: "reset [projectId]", Short: "Reset project statuses to the default template.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/statuses/reset", PositionalParams: []string{"projectId"}},
		{Use: "send", Short: "Start an ephemeral chat request.", Method: http.MethodPost, Path: "/api/v1/chat"},
		{Use: "close [sessionId]", Short: "Close an ephemeral chat session.", Method: http.MethodDelete, Path: "/api/v1/chat/{sessionId}", PositionalParams: []string{"sessionId"}},
		{Use: "create", Short: "Create a project conversation.", Method: http.MethodPost, Path: "/api/v1/chat/conversations"},
		{Use: "list", Short: "List project conversations.", Method: http.MethodGet, Path: "/api/v1/chat/conversations"},
		{Use: "get [conversationId]", Short: "Get a project conversation.", Method: http.MethodGet, Path: "/api/v1/chat/conversations/{conversationId}", PositionalParams: []string{"conversationId"}},
		{Use: "entries [conversationId]", Short: "List project conversation transcript entries.", Method: http.MethodGet, Path: "/api/v1/chat/conversations/{conversationId}/entries", PositionalParams: []string{"conversationId"}},
		{Use: "workspace-diff [conversationId]", Short: "Get project conversation workspace diff summary.", Method: http.MethodGet, Path: "/api/v1/chat/conversations/{conversationId}/workspace-diff", PositionalParams: []string{"conversationId"}},
		{Use: "turn [conversationId]", Short: "Start a project conversation turn.", Method: http.MethodPost, Path: "/api/v1/chat/conversations/{conversationId}/turns", PositionalParams: []string{"conversationId"}},
		{Use: "watch [conversationId]", Short: "Watch project conversation events.", Method: http.MethodGet, Path: "/api/v1/chat/conversations/{conversationId}/stream", PositionalParams: []string{"conversationId"}},
		{Use: "respond-interrupt [conversationId] [interruptId]", Short: "Respond to a project conversation interrupt.", Method: http.MethodPost, Path: "/api/v1/chat/conversations/{conversationId}/interrupts/{interruptId}/respond", PositionalParams: []string{"conversationId", "interruptId"}},
		{Use: "close-runtime [conversationId]", Short: "Close a project conversation live runtime.", Method: http.MethodDelete, Path: "/api/v1/chat/conversations/{conversationId}/runtime", PositionalParams: []string{"conversationId"}},
		{Use: "list [orgId]", Short: "List projects.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/projects", PositionalParams: []string{"orgId"}},
		{Use: "get [projectId]", Short: "Get a project.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}},
		{Use: "create [orgId]", Short: "Create a project.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/projects", PositionalParams: []string{"orgId"}},
		{Use: "update [projectId]", Short: "Update a project.", Method: http.MethodPatch, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}},
		{Use: "delete [projectId]", Short: "Archive a project.", Method: http.MethodDelete, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}},
		{Use: "list [projectId]", Short: "List project update threads.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/updates", PositionalParams: []string{"projectId"}},
		{Use: "create [projectId]", Short: "Create a project update thread.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/updates", PositionalParams: []string{"projectId"}},
		{Use: "update [projectId] [threadId]", Short: "Update a project update thread.", Method: http.MethodPatch, Path: "/api/v1/projects/{projectId}/updates/{threadId}", PositionalParams: []string{"projectId", "threadId"}},
		{Use: "delete [projectId] [threadId]", Short: "Delete a project update thread.", Method: http.MethodDelete, Path: "/api/v1/projects/{projectId}/updates/{threadId}", PositionalParams: []string{"projectId", "threadId"}},
		{Use: "revisions [projectId] [threadId]", Short: "List project update thread revisions.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/updates/{threadId}/revisions", PositionalParams: []string{"projectId", "threadId"}},
		{Use: "list [projectId]", Short: "List project repositories.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/repos", PositionalParams: []string{"projectId"}},
		{Use: "create [projectId]", Short: "Create a project repository.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/repos", PositionalParams: []string{"projectId"}},
		{Use: "update [projectId] [repoId]", Short: "Update a project repository.", Method: http.MethodPatch, Path: "/api/v1/projects/{projectId}/repos/{repoId}", PositionalParams: []string{"projectId", "repoId"}},
		{Use: "delete [projectId] [repoId]", Short: "Delete a project repository.", Method: http.MethodDelete, Path: "/api/v1/projects/{projectId}/repos/{repoId}", PositionalParams: []string{"projectId", "repoId"}},
		{Use: "namespaces [projectId]", Short: "List GitHub namespaces available to the project's effective credential.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/github/namespaces", PositionalParams: []string{"projectId"}},
		{Use: "list [projectId]", Short: "List GitHub repositories visible to the project's effective credential.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/github/repos", PositionalParams: []string{"projectId"}},
		{Use: "create [projectId]", Short: "Create a GitHub repository using the project's effective credential.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/github/repos", PositionalParams: []string{"projectId"}},
		{Use: "list [projectId] [ticketId]", Short: "List ticket repository scopes.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes", PositionalParams: []string{"projectId", "ticketId"}},
		{Use: "create [projectId] [ticketId]", Short: "Create a ticket repository scope.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes", PositionalParams: []string{"projectId", "ticketId"}},
		{Use: "update [projectId] [ticketId] [scopeId]", Short: "Update a ticket repository scope.", Method: http.MethodPatch, Path: "/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes/{scopeId}", PositionalParams: []string{"projectId", "ticketId", "scopeId"}},
		{Use: "delete [projectId] [ticketId] [scopeId]", Short: "Delete a ticket repository scope.", Method: http.MethodDelete, Path: "/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes/{scopeId}", PositionalParams: []string{"projectId", "ticketId", "scopeId"}},
		{Use: "list [projectId]", Short: "List workflows.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/workflows", PositionalParams: []string{"projectId"}},
		{Use: "get [workflowId]", Short: "Get a workflow.", Method: http.MethodGet, Path: "/api/v1/workflows/{workflowId}", PositionalParams: []string{"workflowId"}},
		{Use: "create [projectId]", Short: "Create a workflow.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/workflows", PositionalParams: []string{"projectId"}},
		{Use: "update [workflowId]", Short: "Update a workflow.", Method: http.MethodPatch, Path: "/api/v1/workflows/{workflowId}", PositionalParams: []string{"workflowId"}},
		{Use: "delete [workflowId]", Short: "Delete a workflow.", Method: http.MethodDelete, Path: "/api/v1/workflows/{workflowId}", PositionalParams: []string{"workflowId"}},
		{Use: "get [workflowId]", Short: "Get workflow harness content.", Method: http.MethodGet, Path: "/api/v1/workflows/{workflowId}/harness", PositionalParams: []string{"workflowId"}},
		{Use: "history [workflowId]", Short: "List workflow harness revisions.", Method: http.MethodGet, Path: "/api/v1/workflows/{workflowId}/harness/history", PositionalParams: []string{"workflowId"}},
		{Use: "update [workflowId]", Short: "Update workflow harness content.", Method: http.MethodPut, Path: "/api/v1/workflows/{workflowId}/harness", PositionalParams: []string{"workflowId"}},
		{Use: "variables", Short: "List harness variables.", Method: http.MethodGet, Path: "/api/v1/harness/variables"},
		{Use: "validate", Short: "Validate harness content.", Method: http.MethodPost, Path: "/api/v1/harness/validate"},
		{Use: "list [projectId]", Short: "List scheduled jobs.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/scheduled-jobs", PositionalParams: []string{"projectId"}},
		{Use: "create [projectId]", Short: "Create a scheduled job.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/scheduled-jobs", PositionalParams: []string{"projectId"}},
		{Use: "update [jobId]", Short: "Update a scheduled job.", Method: http.MethodPatch, Path: "/api/v1/scheduled-jobs/{jobId}", PositionalParams: []string{"jobId"}},
		{Use: "delete [jobId]", Short: "Delete a scheduled job.", Method: http.MethodDelete, Path: "/api/v1/scheduled-jobs/{jobId}", PositionalParams: []string{"jobId"}},
		{Use: "trigger [jobId]", Short: "Trigger a scheduled job once.", Method: http.MethodPost, Path: "/api/v1/scheduled-jobs/{jobId}/trigger", PositionalParams: []string{"jobId"}},
		{Use: "list [orgId]", Short: "List machines.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/machines", PositionalParams: []string{"orgId"}},
		{Use: "get [machineId]", Short: "Get a machine.", Method: http.MethodGet, Path: "/api/v1/machines/{machineId}", PositionalParams: []string{"machineId"}},
		{Use: "create [orgId]", Short: "Create a machine.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/machines", PositionalParams: []string{"orgId"}},
		{Use: "update [machineId]", Short: "Update a machine.", Method: http.MethodPatch, Path: "/api/v1/machines/{machineId}", PositionalParams: []string{"machineId"}},
		{Use: "delete [machineId]", Short: "Delete a machine.", Method: http.MethodDelete, Path: "/api/v1/machines/{machineId}", PositionalParams: []string{"machineId"}},
		{Use: "resources [machineId]", Short: "Get machine resources.", Method: http.MethodGet, Path: "/api/v1/machines/{machineId}/resources", PositionalParams: []string{"machineId"}},
		{Use: "test [machineId]", Short: "Test a machine connection.", Method: http.MethodPost, Path: "/api/v1/machines/{machineId}/test", PositionalParams: []string{"machineId"}},
		{Use: "refresh-health [machineId]", Short: "Refresh machine health.", Method: http.MethodPost, Path: "/api/v1/machines/{machineId}/refresh-health", PositionalParams: []string{"machineId"}},
		{Use: "list [orgId]", Short: "List providers.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/providers", PositionalParams: []string{"orgId"}},
		{Use: "get [providerId]", Short: "Get a provider.", Method: http.MethodGet, Path: "/api/v1/providers/{providerId}", PositionalParams: []string{"providerId"}},
		{Use: "create [orgId]", Short: "Create a provider.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/providers", PositionalParams: []string{"orgId"}},
		{Use: "update [providerId]", Short: "Update a provider.", Method: http.MethodPatch, Path: "/api/v1/providers/{providerId}", PositionalParams: []string{"providerId"}},
		{Use: "list [projectId]", Short: "List agents.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents", PositionalParams: []string{"projectId"}},
		{Use: "get [agentId]", Short: "Get an agent.", Method: http.MethodGet, Path: "/api/v1/agents/{agentId}", PositionalParams: []string{"agentId"}},
		{Use: "create [projectId]", Short: "Create an agent.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/agents", PositionalParams: []string{"projectId"}},
		{Use: "update [agentId]", Short: "Update an agent.", Method: http.MethodPatch, Path: "/api/v1/agents/{agentId}", PositionalParams: []string{"agentId"}},
		{Use: "delete [agentId]", Short: "Delete an agent.", Method: http.MethodDelete, Path: "/api/v1/agents/{agentId}", PositionalParams: []string{"agentId"}},
		{Use: "pause [agentId]", Short: "Pause an agent.", Method: http.MethodPost, Path: "/api/v1/agents/{agentId}/pause", PositionalParams: []string{"agentId"}},
		{Use: "resume [agentId]", Short: "Resume an agent.", Method: http.MethodPost, Path: "/api/v1/agents/{agentId}/resume", PositionalParams: []string{"agentId"}},
		{Use: "output [projectId] [agentId]", Short: "List agent output entries.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/output", PositionalParams: []string{"projectId", "agentId"}},
		{Use: "steps [projectId] [agentId]", Short: "List agent step entries.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/steps", PositionalParams: []string{"projectId", "agentId"}},
		{Use: "list [orgId]", Short: "List notification channels.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/channels", PositionalParams: []string{"orgId"}},
		{Use: "create [orgId]", Short: "Create a notification channel.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/channels", PositionalParams: []string{"orgId"}},
		{Use: "update [channelId]", Short: "Update a notification channel.", Method: http.MethodPatch, Path: "/api/v1/channels/{channelId}", PositionalParams: []string{"channelId"}},
		{Use: "delete [channelId]", Short: "Delete a notification channel.", Method: http.MethodDelete, Path: "/api/v1/channels/{channelId}", PositionalParams: []string{"channelId"}},
		{Use: "test [channelId]", Short: "Test a notification channel.", Method: http.MethodPost, Path: "/api/v1/channels/{channelId}/test", PositionalParams: []string{"channelId"}},
		{Use: "list [projectId]", Short: "List project notification rules.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/notification-rules", PositionalParams: []string{"projectId"}},
		{Use: "create [projectId]", Short: "Create a notification rule.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/notification-rules", PositionalParams: []string{"projectId"}},
		{Use: "update [ruleId]", Short: "Update a notification rule.", Method: http.MethodPatch, Path: "/api/v1/notification-rules/{ruleId}", PositionalParams: []string{"ruleId"}},
		{Use: "delete [ruleId]", Short: "Delete a notification rule.", Method: http.MethodDelete, Path: "/api/v1/notification-rules/{ruleId}", PositionalParams: []string{"ruleId"}},
		{Use: "list [projectId]", Short: "List project skills.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/skills", PositionalParams: []string{"projectId"}},
		{Use: "create [projectId]", Short: "Create a skill.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/skills", PositionalParams: []string{"projectId"}},
		{Use: "refresh [projectId]", Short: "Refresh workspace skills.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/skills/refresh", PositionalParams: []string{"projectId"}},
		{Use: "get [skillId]", Short: "Get a skill.", Method: http.MethodGet, Path: "/api/v1/skills/{skillId}", PositionalParams: []string{"skillId"}},
		{Use: "update [skillId]", Short: "Update a skill.", Method: http.MethodPut, Path: "/api/v1/skills/{skillId}", PositionalParams: []string{"skillId"}},
		{Use: "delete [skillId]", Short: "Delete a skill.", Method: http.MethodDelete, Path: "/api/v1/skills/{skillId}", PositionalParams: []string{"skillId"}},
		{Use: "enable [skillId]", Short: "Enable a skill.", Method: http.MethodPost, Path: "/api/v1/skills/{skillId}/enable", PositionalParams: []string{"skillId"}},
		{Use: "disable [skillId]", Short: "Disable a skill.", Method: http.MethodPost, Path: "/api/v1/skills/{skillId}/disable", PositionalParams: []string{"skillId"}},
		{Use: "bind [skillId]", Short: "Bind a skill to workflows.", Method: http.MethodPost, Path: "/api/v1/skills/{skillId}/bind", PositionalParams: []string{"skillId"}},
		{Use: "unbind [skillId]", Short: "Unbind a skill from workflows.", Method: http.MethodPost, Path: "/api/v1/skills/{skillId}/unbind", PositionalParams: []string{"skillId"}},
		{Use: "bind-workflow [workflowId]", Short: "Bind skills to a workflow.", Method: http.MethodPost, Path: "/api/v1/workflows/{workflowId}/skills/bind", PositionalParams: []string{"workflowId"}},
		{Use: "unbind-workflow [workflowId]", Short: "Unbind skills from a workflow.", Method: http.MethodPost, Path: "/api/v1/workflows/{workflowId}/skills/unbind", PositionalParams: []string{"workflowId"}},
		{Use: "events", Short: "Stream system events.", Method: http.MethodGet, Path: "/api/v1/events/stream"},
		{Use: "project [projectId]", Short: "Stream the passive project event bus.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/events/stream", PositionalParams: []string{"projectId"}},
		{Use: "output [projectId] [agentId]", Short: "Stream agent output.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/output/stream", PositionalParams: []string{"projectId", "agentId"}},
		{Use: "steps [projectId] [agentId]", Short: "Stream agent steps.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/steps/stream", PositionalParams: []string{"projectId", "agentId"}},
	}
}

func commandContractSnapshot() (openAPIContractSnapshot, error) {
	doc, err := httpapi.BuildOpenAPIDocument()
	if err != nil {
		return openAPIContractSnapshot{}, err
	}
	payload, err := json.Marshal(doc)
	if err != nil {
		return openAPIContractSnapshot{}, fmt.Errorf("marshal OpenAPI document: %w", err)
	}
	sum := sha256.Sum256(payload)
	contracts, err := loadOpenAPICommandContracts()
	if err != nil {
		return openAPIContractSnapshot{}, err
	}
	items := make([]openAPIContractSnapshotItem, 0, len(contracts))
	for _, spec := range allOpenAPICommandSpecs() {
		contract := contracts[contractKey(spec.Method, spec.Path)]
		items = append(items, openAPIContractSnapshotItem{
			Use:              spec.Use,
			Method:           spec.Method,
			Path:             spec.Path,
			OperationID:      contract.operationID,
			PositionalParams: append([]string(nil), spec.PositionalParams...),
			PathParams:       append([]openAPIInputField(nil), contract.pathParams...),
			QueryParams:      append([]openAPIInputField(nil), contract.queryParams...),
			BodyFields:       append([]openAPIInputField(nil), contract.bodyFields...),
		})
	}
	return openAPIContractSnapshot{
		OpenAPISHA256: hex.EncodeToString(sum[:]),
		Commands:      items,
	}, nil
}

func contractKey(method string, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}

func flagErrorWithNormalize(command *cobra.Command, err error) error {
	return err
}

func normalizeCLIFlagName(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	return pflag.NormalizedName(strings.ReplaceAll(name, "-", "_"))
}

func applyCLIFlagNormalization(flags *pflag.FlagSet) {
	flags.SetNormalizeFunc(normalizeCLIFlagName)
}

func applyCLICommandFlagNormalization(command *cobra.Command) {
	command.SetGlobalNormalizationFunc(normalizeCLIFlagName)
	applyCLIFlagNormalization(command.Flags())
	applyCLIFlagNormalization(command.PersistentFlags())
}

func apiOptionsFromFlags(flags *pflag.FlagSet) apiCommandOptions {
	apiURL, _ := flags.GetString("api_url")
	token, _ := flags.GetString("token")
	return apiCommandOptions{
		apiURL: apiURL,
		token:  token,
	}
}

func outputOptionsFromFlags(flags *pflag.FlagSet) apiOutputOptions {
	jqExpr, _ := flags.GetString("jq")
	jsonExpr, _ := flags.GetString("json")
	templateExpr, _ := flags.GetString("template")
	return apiOutputOptions{
		jqExpr:   jqExpr,
		jsonExpr: jsonExpr,
		template: templateExpr,
	}
}

func urlPathEscape(value string) string {
	return strings.ReplaceAll(urlQueryEscape(value), "+", "%20")
}

func urlQueryEscape(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(url.QueryEscape(strings.TrimSpace(value)), "%2C", ","), "%2F", "/")
}
