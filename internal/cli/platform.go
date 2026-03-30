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
}

type platformContext struct {
	apiURL    string
	token     string
	projectID string
	ticketID  string
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
	projectID   string
	title       string
	description string
	priority    string
	typeName    string
	externalRef string
}

type ticketUpdateInput struct {
	ticketID       string
	title          string
	description    string
	externalRef    string
	statusID       string
	statusName     string
	titleSet       bool
	descriptionSet bool
	externalRefSet bool
	statusIDSet    bool
	statusNameSet  bool
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

type projectUpdateInput struct {
	projectID   string
	description string
}

type projectAddRepoInput struct {
	projectID     string
	name          string
	repositoryURL string
	defaultBranch string
	labels        []string
}

type platformClient struct {
	httpClient platformHTTPDoer
}

func newAgentPlatformTicketCommand() *cobra.Command {
	return newAgentPlatformTicketCommandWithDeps(platformCommandDeps{httpClient: http.DefaultClient})
}

func newAgentPlatformTicketCommandWithDeps(deps platformCommandDeps) *cobra.Command {
	options := &ticketCommandOptions{}
	client := platformClient(deps)

	command := &cobra.Command{
		Use:   "ticket",
		Short: "Operate on OpenASE tickets through the agent platform API.",
	}

	bindPlatformFlags(command.PersistentFlags(), &options.rawPlatformContext)
	command.AddCommand(newTicketListCommand(options, client))
	command.AddCommand(newTicketCreateCommand(options, client))
	command.AddCommand(newTicketReportUsageCommand(options, client))
	command.AddCommand(newTicketUpdateCommand(options, client))

	return command
}

func newAgentPlatformProjectCommand() *cobra.Command {
	return newAgentPlatformProjectCommandWithDeps(platformCommandDeps{httpClient: http.DefaultClient})
}

func newAgentPlatformProjectCommandWithDeps(deps platformCommandDeps) *cobra.Command {
	options := &projectCommandOptions{}
	client := platformClient(deps)

	command := &cobra.Command{
		Use:   "project",
		Short: "Operate on OpenASE projects through the agent platform API.",
	}

	bindPlatformFlags(command.PersistentFlags(), &options.rawPlatformContext)
	command.AddCommand(newProjectUpdateCommand(options, client))
	command.AddCommand(newProjectAddRepoCommand(options, client))

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

	command := &cobra.Command{
		Use:   "list",
		Short: "List tickets in the current project.",
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

	return command
}

func newTicketCreateCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	var title string
	var description string
	var priority string
	var typeName string
	var externalRef string

	command := &cobra.Command{
		Use:   "create",
		Short: "Create a ticket in the current project.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			input, err := platform.parseTicketCreateInput(ticketCreateInput{
				projectID:   options.projectID,
				title:       title,
				description: description,
				priority:    priority,
				typeName:    typeName,
				externalRef: externalRef,
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
	command.Flags().StringVar(&priority, "priority", "", "Ticket priority override.")
	command.Flags().StringVar(&typeName, "type", "", "Ticket type override.")
	command.Flags().StringVar(&externalRef, "external-ref", "", "External reference, for example BetterAndBetterII/openase#39.")
	_ = command.MarkFlagRequired("title")

	return command
}

func newTicketUpdateCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	var title string
	var description string
	var externalRef string
	var statusID string
	var statusName string

	command := &cobra.Command{
		Use:   "update [ticket-id]",
		Short: "Update the current ticket or a specific ticket ID.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			input, err := platform.parseTicketUpdateInput(ticketUpdateInput{
				ticketID:       firstNonEmpty(firstArg(args), options.ticketID),
				title:          title,
				description:    description,
				externalRef:    externalRef,
				statusID:       statusID,
				statusName:     statusName,
				titleSet:       cmd.Flags().Changed("title"),
				descriptionSet: cmd.Flags().Changed("description"),
				externalRefSet: cmd.Flags().Changed("external-ref"),
				statusIDSet:    cmd.Flags().Changed("status-id"),
				statusNameSet:  cmd.Flags().Changed("status") || cmd.Flags().Changed("status-name"),
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

	return command
}

func newTicketReportUsageCommand(options *ticketCommandOptions, client platformClient) *cobra.Command {
	var inputTokens int64
	var outputTokens int64
	var costUSD float64

	command := &cobra.Command{
		Use:   "report-usage [ticket-id]",
		Short: "Report token and cost usage for the current ticket.",
		Args:  cobra.MaximumNArgs(1),
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

	return command
}

func newProjectUpdateCommand(options *projectCommandOptions, client platformClient) *cobra.Command {
	var description string

	command := &cobra.Command{
		Use:   "update",
		Short: "Update the current project description.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			input, err := platform.parseProjectUpdateInput(projectUpdateInput{
				projectID:   options.projectID,
				description: description,
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

	command.Flags().StringVar(&description, "description", "", "Updated project description.")
	_ = command.MarkFlagRequired("description")

	return command
}

func newProjectAddRepoCommand(options *projectCommandOptions, client platformClient) *cobra.Command {
	var name string
	var repositoryURL string
	var defaultBranch string
	var labels []string

	command := &cobra.Command{
		Use:   "add-repo",
		Short: "Register a repository in the current project.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			platform, err := options.resolve()
			if err != nil {
				return err
			}

			input, err := platform.parseProjectAddRepoInput(projectAddRepoInput{
				projectID:     options.projectID,
				name:          name,
				repositoryURL: repositoryURL,
				defaultBranch: defaultBranch,
				labels:        labels,
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
	command.Flags().StringSliceVar(&labels, "label", nil, "Repository label. Repeat for multiple labels.")
	_ = command.MarkFlagRequired("name")
	_ = command.MarkFlagRequired("url")

	return command
}

func (options *ticketCommandOptions) resolve() (platformContext, error) {
	return options.rawPlatformContext.resolve()
}

func (options *projectCommandOptions) resolve() (platformContext, error) {
	return options.rawPlatformContext.resolve()
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
	}, nil
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

	return ticketCreateInput{
		projectID:   projectID,
		title:       title,
		description: strings.TrimSpace(raw.description),
		priority:    strings.TrimSpace(raw.priority),
		typeName:    strings.TrimSpace(raw.typeName),
		externalRef: strings.TrimSpace(raw.externalRef),
	}, nil
}

func (platform platformContext) parseTicketUpdateInput(raw ticketUpdateInput) (ticketUpdateInput, error) {
	ticketID := strings.TrimSpace(firstNonEmpty(raw.ticketID, platform.ticketID))
	if ticketID == "" {
		return ticketUpdateInput{}, fmt.Errorf("ticket id is required via positional argument, --ticket-id, or OPENASE_TICKET_ID")
	}
	if !raw.titleSet && !raw.descriptionSet && !raw.externalRefSet && !raw.statusIDSet && !raw.statusNameSet {
		return ticketUpdateInput{}, fmt.Errorf("at least one of --title, --description, --external-ref, --status, --status-name, or --status-id must be set")
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

	return ticketUpdateInput{
		ticketID:       ticketID,
		title:          strings.TrimSpace(raw.title),
		description:    strings.TrimSpace(raw.description),
		externalRef:    strings.TrimSpace(raw.externalRef),
		statusID:       strings.TrimSpace(raw.statusID),
		statusName:     strings.TrimSpace(raw.statusName),
		titleSet:       raw.titleSet,
		descriptionSet: raw.descriptionSet,
		externalRefSet: raw.externalRefSet,
		statusIDSet:    raw.statusIDSet,
		statusNameSet:  raw.statusNameSet,
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

func (platform platformContext) parseProjectUpdateInput(raw projectUpdateInput) (projectUpdateInput, error) {
	projectID := strings.TrimSpace(firstNonEmpty(raw.projectID, platform.projectID))
	if projectID == "" {
		return projectUpdateInput{}, fmt.Errorf("project id is required via --project-id or OPENASE_PROJECT_ID")
	}

	description := strings.TrimSpace(raw.description)
	if description == "" {
		return projectUpdateInput{}, fmt.Errorf("description must not be empty")
	}

	return projectUpdateInput{
		projectID:   projectID,
		description: description,
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
		projectID:     projectID,
		name:          name,
		repositoryURL: repositoryURL,
		defaultBranch: strings.TrimSpace(raw.defaultBranch),
		labels:        make([]string, 0, len(raw.labels)),
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
	if input.priority != "" {
		payload["priority"] = input.priority
	}
	if input.typeName != "" {
		payload["type"] = input.typeName
	}
	if input.externalRef != "" {
		payload["external_ref"] = input.externalRef
	}

	return client.doJSON(ctx, platform, http.MethodPost, "/projects/"+url.PathEscape(input.projectID)+"/tickets", payload)
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
	return client.doJSON(ctx, platform, http.MethodPatch, "/projects/"+url.PathEscape(input.projectID), map[string]any{
		"description": input.description,
	})
}

func (client platformClient) addProjectRepo(ctx context.Context, platform platformContext, input projectAddRepoInput) ([]byte, error) {
	payload := map[string]any{
		"name":           input.name,
		"repository_url": input.repositoryURL,
		"default_branch": input.defaultBranch,
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

func float64PointerWhen(enabled bool, value float64) *float64 {
	if !enabled {
		return nil
	}

	return &value
}
