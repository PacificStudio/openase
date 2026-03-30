package cli

import (
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
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRawAPICommand(cmd, deps, args[0], args[1], options, output, headers, fields, queryItems, inputPath)
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLIFlagNormalization(command.Flags())
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
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [ticketId]", Short: "Get a ticket.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}", PositionalParams: []string{"ticketId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [projectId]", Short: "Create a ticket.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/tickets", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [ticketId]", Short: "Update a ticket.", Method: http.MethodPatch, Path: "/api/v1/tickets/{ticketId}", PositionalParams: []string{"ticketId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "detail [projectId] [ticketId]", Short: "Get ticket detail.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets/{ticketId}/detail", PositionalParams: []string{"projectId", "ticketId"}}))
	command.AddCommand(newTicketCommentCommand())
	return command
}

func newTicketCommentCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "comment",
		Short: "Operate on ticket comments.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [ticketId]", Short: "List ticket comments.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}/comments", PositionalParams: []string{"ticketId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [ticketId]", Short: "Create a ticket comment.", Method: http.MethodPost, Path: "/api/v1/tickets/{ticketId}/comments", PositionalParams: []string{"ticketId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [ticketId] [commentId]", Short: "Update a ticket comment.", Method: http.MethodPatch, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}", PositionalParams: []string{"ticketId", "commentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [ticketId] [commentId]", Short: "Delete a ticket comment.", Method: http.MethodDelete, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}", PositionalParams: []string{"ticketId", "commentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "revisions [ticketId] [commentId]", Short: "List ticket comment revisions.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}/revisions", PositionalParams: []string{"ticketId", "commentId"}}))
	command.AddCommand(newTicketCommentWorkpadCommand())
	return command
}

func newProjectCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "project",
		Short: "Operate on projects through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [orgId]", Short: "List projects.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/projects", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [projectId]", Short: "Get a project.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [orgId]", Short: "Create a project.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/projects", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [projectId]", Short: "Update a project.", Method: http.MethodPatch, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [projectId]", Short: "Archive a project.", Method: http.MethodDelete, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}}))
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
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "prerequisite [projectId]", Short: "Get workflow repository prerequisites.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/workflows/prerequisite", PositionalParams: []string{"projectId"}}))

	harness := &cobra.Command{
		Use:   "harness",
		Short: "Operate on workflow harness content.",
	}
	harness.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [workflowId]", Short: "Get workflow harness content.", Method: http.MethodGet, Path: "/api/v1/workflows/{workflowId}/harness", PositionalParams: []string{"workflowId"}}))
	harness.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [workflowId]", Short: "Update workflow harness content.", Method: http.MethodPut, Path: "/api/v1/workflows/{workflowId}/harness", PositionalParams: []string{"workflowId"}}))
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

func newMachineCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "machine",
		Short: "Operate on machines through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [orgId]", Short: "List machines.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/machines", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "get [machineId]", Short: "Get a machine.", Method: http.MethodGet, Path: "/api/v1/machines/{machineId}", PositionalParams: []string{"machineId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "create [orgId]", Short: "Create a machine.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/machines", PositionalParams: []string{"orgId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "update [machineId]", Short: "Update a machine.", Method: http.MethodPatch, Path: "/api/v1/machines/{machineId}", PositionalParams: []string{"machineId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [machineId]", Short: "Delete a machine.", Method: http.MethodDelete, Path: "/api/v1/machines/{machineId}", PositionalParams: []string{"machineId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "resources [machineId]", Short: "Get machine resources.", Method: http.MethodGet, Path: "/api/v1/machines/{machineId}/resources", PositionalParams: []string{"machineId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "test [machineId]", Short: "Test a machine connection.", Method: http.MethodPost, Path: "/api/v1/machines/{machineId}/test", PositionalParams: []string{"machineId"}}))
	return command
}

func newProviderCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "provider",
		Short: "Operate on agent providers through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [orgId]", Short: "List providers.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/providers", PositionalParams: []string{"orgId"}}))
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
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "delete [agentId]", Short: "Delete an agent.", Method: http.MethodDelete, Path: "/api/v1/agents/{agentId}", PositionalParams: []string{"agentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "pause [agentId]", Short: "Pause an agent.", Method: http.MethodPost, Path: "/api/v1/agents/{agentId}/pause", PositionalParams: []string{"agentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "resume [agentId]", Short: "Resume an agent.", Method: http.MethodPost, Path: "/api/v1/agents/{agentId}/resume", PositionalParams: []string{"agentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "output [projectId] [agentId]", Short: "List agent output entries.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/output", PositionalParams: []string{"projectId", "agentId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "steps [projectId] [agentId]", Short: "List agent step entries.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/steps", PositionalParams: []string{"projectId", "agentId"}}))
	return command
}

func newSkillCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "skill",
		Short: "Operate on skills through the OpenASE API.",
	}
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "list [projectId]", Short: "List project skills.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/skills", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "bind [workflowId]", Short: "Bind skills to a workflow.", Method: http.MethodPost, Path: "/api/v1/workflows/{workflowId}/skills/bind", PositionalParams: []string{"workflowId"}}))
	command.AddCommand(newOpenAPIOperationCommand(openAPICommandSpec{Use: "unbind [workflowId]", Short: "Unbind skills from a workflow.", Method: http.MethodPost, Path: "/api/v1/workflows/{workflowId}/skills/unbind", PositionalParams: []string{"workflowId"}}))
	return command
}

func newWatchCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "watch",
		Short: "Operate on stream and watch endpoints.",
	}
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{Use: "events", Short: "Stream system events.", Method: http.MethodGet, Path: "/api/v1/events/stream"}))
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{Use: "tickets [projectId]", Short: "Stream project ticket events.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets/stream", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{Use: "agents [projectId]", Short: "Stream project agent events.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/stream", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{Use: "activity [projectId]", Short: "Stream project activity events.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/activity/stream", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{Use: "hooks [projectId]", Short: "Stream project hook events.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/hooks/stream", PositionalParams: []string{"projectId"}}))
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{Use: "output [projectId] [agentId]", Short: "Stream agent output.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/output/stream", PositionalParams: []string{"projectId", "agentId"}}))
	command.AddCommand(newOpenAPIStreamCommand(openAPICommandSpec{Use: "steps [projectId] [agentId]", Short: "Stream agent steps.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/steps/stream", PositionalParams: []string{"projectId", "agentId"}}))
	return command
}

func newOpenAPIOperationCommand(spec openAPICommandSpec) *cobra.Command {
	contract := mustOpenAPICommandContract(spec)
	deps := apiCommandDeps{httpClient: http.DefaultClient}
	command := &cobra.Command{
		Use:   spec.Use,
		Short: contract.summary,
		Args:  cobra.MaximumNArgs(len(spec.PositionalParams)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpenAPIOperationCommand(cmd, deps, contract, args)
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLIFlagNormalization(command.Flags())
	registerOpenAPICommandFlags(command.Flags(), contract)
	return command
}

func newOpenAPIStreamCommand(spec openAPICommandSpec) *cobra.Command {
	contract := mustOpenAPICommandContract(spec)
	deps := apiCommandDeps{httpClient: http.DefaultClient}
	command := &cobra.Command{
		Use:   spec.Use,
		Short: contract.summary,
		Args:  cobra.MaximumNArgs(len(spec.PositionalParams)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpenAPIStreamCommand(cmd, deps, contract, args)
		},
	}
	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLIFlagNormalization(command.Flags())
	registerOpenAPIStreamFlags(command.Flags(), contract)
	return command
}

func newTicketCommentWorkpadCommand() *cobra.Command {
	var apiOptions apiCommandOptions
	var output apiOutputOptions
	var body string
	var bodyFile string
	var createdBy string
	var editedBy string
	var editReason string

	command := &cobra.Command{
		Use:   "workpad [ticketId]",
		Short: "Upsert the `## Codex Workpad` comment for a ticket.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(body) == "" && strings.TrimSpace(bodyFile) == "" {
				return fmt.Errorf("one of --body or --body-file is required")
			}
			if strings.TrimSpace(body) != "" && strings.TrimSpace(bodyFile) != "" {
				return fmt.Errorf("only one of --body or --body-file can be set")
			}

			apiContext, err := apiOptions.resolve()
			if err != nil {
				return err
			}
			ticketID, err := resolveCommandPathValue(cmd, "ticketId", args, 0)
			if err != nil {
				return err
			}

			content, err := resolveWorkpadBody(body, bodyFile)
			if err != nil {
				return err
			}
			content = normalizeWorkpadBody(content)

			deps := apiCommandDeps{httpClient: http.DefaultClient}
			listResponse, err := apiContext.do(cmd.Context(), deps, apiRequest{
				Method: http.MethodGet,
				Path:   "tickets/" + urlPathEscape(ticketID) + "/comments",
			})
			if err != nil {
				return err
			}

			commentID, err := findCodexWorkpadCommentID(listResponse.Body)
			if err != nil {
				return err
			}

			var response apiResponse
			if commentID == "" {
				payload := map[string]any{"body": content}
				if strings.TrimSpace(createdBy) != "" {
					payload["created_by"] = strings.TrimSpace(createdBy)
				}
				bodyBytes, marshalErr := json.Marshal(payload)
				if marshalErr != nil {
					return fmt.Errorf("marshal workpad create payload: %w", marshalErr)
				}
				response, err = apiContext.do(cmd.Context(), deps, apiRequest{
					Method: http.MethodPost,
					Path:   "tickets/" + urlPathEscape(ticketID) + "/comments",
					Body:   bodyBytes,
				})
			} else {
				payload := map[string]any{"body": content}
				if strings.TrimSpace(editedBy) != "" {
					payload["edited_by"] = strings.TrimSpace(editedBy)
				}
				if strings.TrimSpace(editReason) != "" {
					payload["edit_reason"] = strings.TrimSpace(editReason)
				}
				bodyBytes, marshalErr := json.Marshal(payload)
				if marshalErr != nil {
					return fmt.Errorf("marshal workpad update payload: %w", marshalErr)
				}
				response, err = apiContext.do(cmd.Context(), deps, apiRequest{
					Method: http.MethodPatch,
					Path:   "tickets/" + urlPathEscape(ticketID) + "/comments/" + urlPathEscape(commentID),
					Body:   bodyBytes,
				})
			}
			if err != nil {
				return err
			}
			return writeAPIOutput(cmd.OutOrStdout(), response.Body, output)
		},
	}

	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLIFlagNormalization(command.Flags())
	bindAPICommandFlags(command.Flags(), &apiOptions)
	bindAPIOutputFlags(command.Flags(), &output)
	command.Flags().StringVar(&body, "body", "", "Workpad markdown body.")
	command.Flags().StringVar(&bodyFile, "body-file", "", "Read the workpad markdown body from a file. Use - for stdin.")
	command.Flags().StringVar(&createdBy, "created_by", "", "Override the comment creator when the workpad is created.")
	command.Flags().StringVar(&editedBy, "edited_by", "", "Override the workpad editor when the workpad is updated.")
	command.Flags().StringVar(&editReason, "edit_reason", "", "Optional reason for a workpad update.")
	return command
}

func runOpenAPIOperationCommand(cmd *cobra.Command, deps apiCommandDeps, contract openAPICommandContract, args []string) error {
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

func runOpenAPIStreamCommand(cmd *cobra.Command, deps apiCommandDeps, contract openAPICommandContract, args []string) error {
	apiContext, err := apiOptionsFromFlags(cmd.Flags()).resolve()
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
		registerFieldFlag(flags, field)
	}
	for _, field := range contract.queryParams {
		registerFieldFlag(flags, field)
	}
	for _, field := range contract.bodyFields {
		registerFieldFlag(flags, field)
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
		registerFieldFlag(flags, field)
	}
	for _, field := range contract.queryParams {
		registerFieldFlag(flags, field)
	}
	var apiOptions apiCommandOptions
	bindAPICommandFlags(flags, &apiOptions)
}

func registerFieldFlag(flags *pflag.FlagSet, field openAPIInputField) {
	switch field.Kind {
	case flagValueString:
		flags.String(field.Name, "", field.Description)
	case flagValueStringSlice:
		flags.StringSlice(field.Name, nil, field.Description)
	case flagValueInt64:
		flags.Int64(field.Name, 0, field.Description)
	case flagValueFloat64:
		flags.Float64(field.Name, 0, field.Description)
	case flagValueBool:
		flags.Bool(field.Name, false, field.Description)
	default:
		flags.String(field.Name, "", field.Description)
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
	switch field.Kind {
	case flagValueInt64:
		value, err := flags.GetInt64(field.Name)
		if err != nil {
			return "", err
		}
		if !flags.Changed(field.Name) {
			return "", nil
		}
		return fmt.Sprintf("%d", value), nil
	case flagValueFloat64:
		value, err := flags.GetFloat64(field.Name)
		if err != nil {
			return "", err
		}
		if !flags.Changed(field.Name) {
			return "", nil
		}
		return strings.TrimSpace(strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", value), "0"), ".")), nil
	case flagValueBool:
		value, err := flags.GetBool(field.Name)
		if err != nil {
			return "", err
		}
		if !flags.Changed(field.Name) {
			return "", nil
		}
		if value {
			return "true", nil
		}
		return "false", nil
	case flagValueStringSlice:
		values, err := flags.GetStringSlice(field.Name)
		if err != nil {
			return "", err
		}
		if !flags.Changed(field.Name) {
			return "", nil
		}
		return strings.Join(trimNonEmpty(values), ","), nil
	default:
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
	return flags.Changed(field.Name)
}

func readFieldValue(flags *pflag.FlagSet, field openAPIInputField) (any, error) {
	switch field.Kind {
	case flagValueInt64:
		return flags.GetInt64(field.Name)
	case flagValueFloat64:
		return flags.GetFloat64(field.Name)
	case flagValueBool:
		return flags.GetBool(field.Name)
	case flagValueStringSlice:
		values, err := flags.GetStringSlice(field.Name)
		if err != nil {
			return nil, err
		}
		return trimNonEmpty(values), nil
	default:
		value, err := flags.GetString(field.Name)
		if err != nil {
			return nil, err
		}
		return strings.TrimSpace(value), nil
	}
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
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read input file %s: %w", path, err)
	}
	return body, nil
}

func resolveWorkpadBody(body string, bodyFile string) (string, error) {
	if strings.TrimSpace(bodyFile) == "" {
		return strings.TrimSpace(body), nil
	}
	payload, err := readInputFile(strings.TrimSpace(bodyFile))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}

func normalizeWorkpadBody(body string) string {
	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "## Codex Workpad") {
		return trimmed
	}
	if trimmed == "" {
		return "## Codex Workpad"
	}
	return "## Codex Workpad\n\n" + trimmed
}

func findCodexWorkpadCommentID(body []byte) (string, error) {
	var payload struct {
		Comments []struct {
			ID           string `json:"id"`
			BodyMarkdown string `json:"body_markdown"`
			IsDeleted    bool   `json:"is_deleted"`
		} `json:"comments"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode ticket comments response: %w", err)
	}
	for _, comment := range payload.Comments {
		if comment.IsDeleted {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(comment.BodyMarkdown), "## Codex Workpad") {
			return comment.ID, nil
		}
	}
	return "", nil
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
			fields, required, err := openAPIFieldsFromSchema(mediaType.Schema.Value)
			if err != nil {
				return openAPICommandContract{}, fmt.Errorf("%s %s request body: %w", spec.Method, spec.Path, err)
			}
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

func openAPIFieldsFromSchema(schema *openapi3.Schema) ([]openAPIInputField, []string, error) {
	if schema == nil {
		return nil, nil, nil
	}
	if !schema.Type.Is("object") {
		return nil, nil, nil
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
	return fields, requiredNames, nil
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
		{Use: "get [ticketId]", Short: "Get a ticket.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}", PositionalParams: []string{"ticketId"}},
		{Use: "create [projectId]", Short: "Create a ticket.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/tickets", PositionalParams: []string{"projectId"}},
		{Use: "update [ticketId]", Short: "Update a ticket.", Method: http.MethodPatch, Path: "/api/v1/tickets/{ticketId}", PositionalParams: []string{"ticketId"}},
		{Use: "detail [projectId] [ticketId]", Short: "Get ticket detail.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets/{ticketId}/detail", PositionalParams: []string{"projectId", "ticketId"}},
		{Use: "list [ticketId]", Short: "List ticket comments.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}/comments", PositionalParams: []string{"ticketId"}},
		{Use: "create [ticketId]", Short: "Create a ticket comment.", Method: http.MethodPost, Path: "/api/v1/tickets/{ticketId}/comments", PositionalParams: []string{"ticketId"}},
		{Use: "update [ticketId] [commentId]", Short: "Update a ticket comment.", Method: http.MethodPatch, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}", PositionalParams: []string{"ticketId", "commentId"}},
		{Use: "delete [ticketId] [commentId]", Short: "Delete a ticket comment.", Method: http.MethodDelete, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}", PositionalParams: []string{"ticketId", "commentId"}},
		{Use: "revisions [ticketId] [commentId]", Short: "List ticket comment revisions.", Method: http.MethodGet, Path: "/api/v1/tickets/{ticketId}/comments/{commentId}/revisions", PositionalParams: []string{"ticketId", "commentId"}},
		{Use: "list [orgId]", Short: "List projects.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/projects", PositionalParams: []string{"orgId"}},
		{Use: "get [projectId]", Short: "Get a project.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}},
		{Use: "create [orgId]", Short: "Create a project.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/projects", PositionalParams: []string{"orgId"}},
		{Use: "update [projectId]", Short: "Update a project.", Method: http.MethodPatch, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}},
		{Use: "delete [projectId]", Short: "Archive a project.", Method: http.MethodDelete, Path: "/api/v1/projects/{projectId}", PositionalParams: []string{"projectId"}},
		{Use: "list [projectId]", Short: "List workflows.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/workflows", PositionalParams: []string{"projectId"}},
		{Use: "get [workflowId]", Short: "Get a workflow.", Method: http.MethodGet, Path: "/api/v1/workflows/{workflowId}", PositionalParams: []string{"workflowId"}},
		{Use: "create [projectId]", Short: "Create a workflow.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/workflows", PositionalParams: []string{"projectId"}},
		{Use: "update [workflowId]", Short: "Update a workflow.", Method: http.MethodPatch, Path: "/api/v1/workflows/{workflowId}", PositionalParams: []string{"workflowId"}},
		{Use: "delete [workflowId]", Short: "Delete a workflow.", Method: http.MethodDelete, Path: "/api/v1/workflows/{workflowId}", PositionalParams: []string{"workflowId"}},
		{Use: "prerequisite [projectId]", Short: "Get workflow repository prerequisites.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/workflows/prerequisite", PositionalParams: []string{"projectId"}},
		{Use: "get [workflowId]", Short: "Get workflow harness content.", Method: http.MethodGet, Path: "/api/v1/workflows/{workflowId}/harness", PositionalParams: []string{"workflowId"}},
		{Use: "update [workflowId]", Short: "Update workflow harness content.", Method: http.MethodPut, Path: "/api/v1/workflows/{workflowId}/harness", PositionalParams: []string{"workflowId"}},
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
		{Use: "list [orgId]", Short: "List providers.", Method: http.MethodGet, Path: "/api/v1/orgs/{orgId}/providers", PositionalParams: []string{"orgId"}},
		{Use: "create [orgId]", Short: "Create a provider.", Method: http.MethodPost, Path: "/api/v1/orgs/{orgId}/providers", PositionalParams: []string{"orgId"}},
		{Use: "update [providerId]", Short: "Update a provider.", Method: http.MethodPatch, Path: "/api/v1/providers/{providerId}", PositionalParams: []string{"providerId"}},
		{Use: "list [projectId]", Short: "List agents.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents", PositionalParams: []string{"projectId"}},
		{Use: "get [agentId]", Short: "Get an agent.", Method: http.MethodGet, Path: "/api/v1/agents/{agentId}", PositionalParams: []string{"agentId"}},
		{Use: "create [projectId]", Short: "Create an agent.", Method: http.MethodPost, Path: "/api/v1/projects/{projectId}/agents", PositionalParams: []string{"projectId"}},
		{Use: "delete [agentId]", Short: "Delete an agent.", Method: http.MethodDelete, Path: "/api/v1/agents/{agentId}", PositionalParams: []string{"agentId"}},
		{Use: "pause [agentId]", Short: "Pause an agent.", Method: http.MethodPost, Path: "/api/v1/agents/{agentId}/pause", PositionalParams: []string{"agentId"}},
		{Use: "resume [agentId]", Short: "Resume an agent.", Method: http.MethodPost, Path: "/api/v1/agents/{agentId}/resume", PositionalParams: []string{"agentId"}},
		{Use: "output [projectId] [agentId]", Short: "List agent output entries.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/output", PositionalParams: []string{"projectId", "agentId"}},
		{Use: "steps [projectId] [agentId]", Short: "List agent step entries.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/{agentId}/steps", PositionalParams: []string{"projectId", "agentId"}},
		{Use: "list [projectId]", Short: "List project skills.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/skills", PositionalParams: []string{"projectId"}},
		{Use: "bind [workflowId]", Short: "Bind skills to a workflow.", Method: http.MethodPost, Path: "/api/v1/workflows/{workflowId}/skills/bind", PositionalParams: []string{"workflowId"}},
		{Use: "unbind [workflowId]", Short: "Unbind skills from a workflow.", Method: http.MethodPost, Path: "/api/v1/workflows/{workflowId}/skills/unbind", PositionalParams: []string{"workflowId"}},
		{Use: "events", Short: "Stream system events.", Method: http.MethodGet, Path: "/api/v1/events/stream"},
		{Use: "tickets [projectId]", Short: "Stream project ticket events.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/tickets/stream", PositionalParams: []string{"projectId"}},
		{Use: "agents [projectId]", Short: "Stream project agent events.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/agents/stream", PositionalParams: []string{"projectId"}},
		{Use: "activity [projectId]", Short: "Stream project activity events.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/activity/stream", PositionalParams: []string{"projectId"}},
		{Use: "hooks [projectId]", Short: "Stream project hook events.", Method: http.MethodGet, Path: "/api/v1/projects/{projectId}/hooks/stream", PositionalParams: []string{"projectId"}},
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

func applyCLIFlagNormalization(flags *pflag.FlagSet) {
	flags.SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		return pflag.NormalizedName(strings.ReplaceAll(name, "-", "_"))
	})
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
