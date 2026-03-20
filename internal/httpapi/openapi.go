package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/labstack/echo/v4"
)

type OpenAPIErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type OpenAPIOrganization struct {
	ID                     string  `json:"id"`
	Name                   string  `json:"name"`
	Slug                   string  `json:"slug"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id,omitempty"`
}

type OpenAPIProject struct {
	ID                     string  `json:"id"`
	OrganizationID         string  `json:"organization_id"`
	Name                   string  `json:"name"`
	Slug                   string  `json:"slug"`
	Description            string  `json:"description"`
	Status                 string  `json:"status"`
	DefaultWorkflowID      *string `json:"default_workflow_id,omitempty"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id,omitempty"`
	MaxConcurrentAgents    int     `json:"max_concurrent_agents"`
}

type OpenAPIProjectRepo struct {
	ID            string   `json:"id"`
	ProjectID     string   `json:"project_id"`
	Name          string   `json:"name"`
	RepositoryURL string   `json:"repository_url"`
	DefaultBranch string   `json:"default_branch"`
	ClonePath     *string  `json:"clone_path,omitempty"`
	IsPrimary     bool     `json:"is_primary"`
	Labels        []string `json:"labels,omitempty"`
}

type OpenAPIAgentProvider struct {
	ID                 string         `json:"id"`
	OrganizationID     string         `json:"organization_id"`
	Name               string         `json:"name"`
	AdapterType        string         `json:"adapter_type"`
	CliCommand         string         `json:"cli_command"`
	CliArgs            []string       `json:"cli_args"`
	AuthConfig         map[string]any `json:"auth_config"`
	ModelName          string         `json:"model_name"`
	ModelTemperature   float64        `json:"model_temperature"`
	ModelMaxTokens     int            `json:"model_max_tokens"`
	CostPerInputToken  float64        `json:"cost_per_input_token"`
	CostPerOutputToken float64        `json:"cost_per_output_token"`
}

type OpenAPIAgent struct {
	ID                    string   `json:"id"`
	ProviderID            string   `json:"provider_id"`
	ProjectID             string   `json:"project_id"`
	Name                  string   `json:"name"`
	Status                string   `json:"status"`
	CurrentTicketID       *string  `json:"current_ticket_id,omitempty"`
	SessionID             string   `json:"session_id"`
	WorkspacePath         string   `json:"workspace_path"`
	Capabilities          []string `json:"capabilities"`
	TotalTokensUsed       int64    `json:"total_tokens_used"`
	TotalTicketsCompleted int      `json:"total_tickets_completed"`
	LastHeartbeatAt       *string  `json:"last_heartbeat_at,omitempty"`
}

type OpenAPIActivityEvent struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"project_id"`
	TicketID  *string        `json:"ticket_id,omitempty"`
	AgentID   *string        `json:"agent_id,omitempty"`
	EventType string         `json:"event_type"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"created_at"`
}

type OpenAPITicketReference struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	StatusID   string `json:"status_id"`
	StatusName string `json:"status_name"`
}

type OpenAPITicketDependency struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Target OpenAPITicketReference `json:"target"`
}

type OpenAPITicket struct {
	ID                string                    `json:"id"`
	ProjectID         string                    `json:"project_id"`
	Identifier        string                    `json:"identifier"`
	Title             string                    `json:"title"`
	Description       string                    `json:"description"`
	StatusID          string                    `json:"status_id"`
	StatusName        string                    `json:"status_name"`
	Priority          string                    `json:"priority"`
	Type              string                    `json:"type"`
	WorkflowID        *string                   `json:"workflow_id,omitempty"`
	CreatedBy         string                    `json:"created_by"`
	Parent            *OpenAPITicketReference   `json:"parent,omitempty"`
	Children          []OpenAPITicketReference  `json:"children"`
	Dependencies      []OpenAPITicketDependency `json:"dependencies"`
	ExternalRef       string                    `json:"external_ref"`
	BudgetUSD         float64                   `json:"budget_usd"`
	CostAmount        float64                   `json:"cost_amount"`
	AttemptCount      int                       `json:"attempt_count"`
	ConsecutiveErrors int                       `json:"consecutive_errors"`
	NextRetryAt       *string                   `json:"next_retry_at,omitempty"`
	RetryPaused       bool                      `json:"retry_paused"`
	PauseReason       string                    `json:"pause_reason,omitempty"`
	CreatedAt         string                    `json:"created_at"`
}

type OpenAPITicketRepoScopeDetail struct {
	ID             string              `json:"id"`
	TicketID       string              `json:"ticket_id"`
	RepoID         string              `json:"repo_id"`
	Repo           *OpenAPIProjectRepo `json:"repo,omitempty"`
	BranchName     string              `json:"branch_name"`
	PullRequestURL *string             `json:"pull_request_url,omitempty"`
	PrStatus       string              `json:"pr_status"`
	CiStatus       string              `json:"ci_status"`
	IsPrimaryScope bool                `json:"is_primary_scope"`
}

type OpenAPITicketStatus struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
	Position    int    `json:"position"`
	IsDefault   bool   `json:"is_default"`
	Description string `json:"description"`
}

type OpenAPIWorkflow struct {
	ID                  string         `json:"id"`
	ProjectID           string         `json:"project_id"`
	Name                string         `json:"name"`
	Type                string         `json:"type"`
	HarnessPath         string         `json:"harness_path"`
	HarnessContent      *string        `json:"harness_content,omitempty"`
	Hooks               map[string]any `json:"hooks"`
	MaxConcurrent       int            `json:"max_concurrent"`
	MaxRetryAttempts    int            `json:"max_retry_attempts"`
	TimeoutMinutes      int            `json:"timeout_minutes"`
	StallTimeoutMinutes int            `json:"stall_timeout_minutes"`
	Version             int            `json:"version"`
	IsActive            bool           `json:"is_active"`
	PickupStatusID      string         `json:"pickup_status_id"`
	FinishStatusID      *string        `json:"finish_status_id,omitempty"`
}

type OpenAPIHarnessDocument struct {
	WorkflowID string `json:"workflow_id"`
	Path       string `json:"path"`
	Content    string `json:"content"`
	Version    int    `json:"version"`
}

type OpenAPIScheduledJobTicketTemplate struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status,omitempty"`
	Priority    string  `json:"priority"`
	Type        string  `json:"type"`
	CreatedBy   string  `json:"created_by"`
	BudgetUSD   float64 `json:"budget_usd,omitempty"`
}

type OpenAPIScheduledJob struct {
	ID             string                            `json:"id"`
	ProjectID      string                            `json:"project_id"`
	Name           string                            `json:"name"`
	CronExpression string                            `json:"cron_expression"`
	WorkflowID     string                            `json:"workflow_id"`
	TicketTemplate OpenAPIScheduledJobTicketTemplate `json:"ticket_template"`
	IsEnabled      bool                              `json:"is_enabled"`
	LastRunAt      *string                           `json:"last_run_at,omitempty"`
	NextRunAt      *string                           `json:"next_run_at,omitempty"`
}

type OpenAPIValidationIssue struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
}

type OpenAPIHarnessValidationResponse struct {
	Valid  bool                     `json:"valid"`
	Issues []OpenAPIValidationIssue `json:"issues"`
}

type OpenAPISkillWorkflowBinding struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	HarnessPath string `json:"harness_path"`
}

type OpenAPISkill struct {
	Name           string                        `json:"name"`
	Description    string                        `json:"description"`
	Path           string                        `json:"path"`
	IsBuiltin      bool                          `json:"is_builtin"`
	BoundWorkflows []OpenAPISkillWorkflowBinding `json:"bound_workflows"`
}

type OpenAPIBuiltinRole struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	WorkflowType string `json:"workflow_type"`
	Summary      string `json:"summary"`
	HarnessPath  string `json:"harness_path"`
	Content      string `json:"content"`
}

type OpenAPIHRAdvisorSummary struct {
	OpenTickets         int      `json:"open_tickets"`
	CodingTickets       int      `json:"coding_tickets"`
	FailingTickets      int      `json:"failing_tickets"`
	BlockedTickets      int      `json:"blocked_tickets"`
	ActiveAgents        int      `json:"active_agents"`
	WorkflowCount       int      `json:"workflow_count"`
	RecentActivityCount int      `json:"recent_activity_count"`
	ActiveWorkflowTypes []string `json:"active_workflow_types"`
}

type OpenAPIHRAdvisorStaffing struct {
	Developers int `json:"developers"`
	QA         int `json:"qa"`
	Docs       int `json:"docs"`
	Security   int `json:"security"`
	Product    int `json:"product"`
	Research   int `json:"research"`
}

type OpenAPIHRAdvisorRecommendation struct {
	RoleSlug              string   `json:"role_slug"`
	RoleName              string   `json:"role_name"`
	WorkflowType          string   `json:"workflow_type"`
	Summary               string   `json:"summary"`
	HarnessPath           string   `json:"harness_path"`
	Priority              string   `json:"priority"`
	Reason                string   `json:"reason"`
	Evidence              []string `json:"evidence"`
	SuggestedHeadcount    int      `json:"suggested_headcount"`
	SuggestedWorkflowName string   `json:"suggested_workflow_name"`
	ActivationReady       bool     `json:"activation_ready"`
	ActiveWorkflowName    *string  `json:"active_workflow_name,omitempty"`
}

type OpenAPIOrganizationsResponse struct {
	Organizations []OpenAPIOrganization `json:"organizations"`
}

type OpenAPIOrganizationResponse struct {
	Organization OpenAPIOrganization `json:"organization"`
}

type OpenAPIProjectsResponse struct {
	Projects []OpenAPIProject `json:"projects"`
}

type OpenAPIProjectResponse struct {
	Project OpenAPIProject `json:"project"`
}

type OpenAPIAgentProvidersResponse struct {
	Providers []OpenAPIAgentProvider `json:"providers"`
}

type OpenAPIAgentProviderResponse struct {
	Provider OpenAPIAgentProvider `json:"provider"`
}

type OpenAPIAgentsResponse struct {
	Agents []OpenAPIAgent `json:"agents"`
}

type OpenAPIActivityEventsResponse struct {
	Events []OpenAPIActivityEvent `json:"events"`
}

type OpenAPITicketStatusesResponse struct {
	Statuses []OpenAPITicketStatus `json:"statuses"`
}

type OpenAPITicketsResponse struct {
	Tickets []OpenAPITicket `json:"tickets"`
}

type OpenAPITicketResponse struct {
	Ticket OpenAPITicket `json:"ticket"`
}

type OpenAPIWorkflowsResponse struct {
	Workflows []OpenAPIWorkflow `json:"workflows"`
}

type OpenAPIWorkflowResponse struct {
	Workflow OpenAPIWorkflow `json:"workflow"`
}

type OpenAPIScheduledJobsResponse struct {
	ScheduledJobs []OpenAPIScheduledJob `json:"scheduled_jobs"`
}

type OpenAPIScheduledJobResponse struct {
	ScheduledJob OpenAPIScheduledJob `json:"scheduled_job"`
}

type OpenAPIScheduledJobTriggerResponse struct {
	ScheduledJob OpenAPIScheduledJob `json:"scheduled_job"`
	Ticket       OpenAPITicket       `json:"ticket"`
}

type OpenAPIHarnessResponse struct {
	Harness OpenAPIHarnessDocument `json:"harness"`
}

type OpenAPISkillsResponse struct {
	Skills []OpenAPISkill `json:"skills"`
}

type OpenAPIRolesResponse struct {
	Roles []OpenAPIBuiltinRole `json:"roles"`
}

type OpenAPIHRAdvisorResponse struct {
	ProjectID       string                           `json:"project_id"`
	Summary         OpenAPIHRAdvisorSummary          `json:"summary"`
	Staffing        OpenAPIHRAdvisorStaffing         `json:"staffing"`
	Recommendations []OpenAPIHRAdvisorRecommendation `json:"recommendations"`
}

type OpenAPITicketDetailResponse struct {
	Ticket      OpenAPITicket                  `json:"ticket"`
	RepoScopes  []OpenAPITicketRepoScopeDetail `json:"repo_scopes"`
	Activity    []OpenAPIActivityEvent         `json:"activity"`
	HookHistory []OpenAPIActivityEvent         `json:"hook_history"`
}

type OpenAPICreateOrganizationRequest domain.OrganizationInput
type OpenAPIUpdateOrganizationRequest organizationPatchRequest
type OpenAPICreateAgentProviderRequest domain.AgentProviderInput
type OpenAPIUpdateAgentProviderRequest agentProviderPatchRequest
type OpenAPICreateProjectRequest domain.ProjectInput
type OpenAPIUpdateProjectRequest projectPatchRequest
type OpenAPICreateWorkflowRequest rawCreateWorkflowRequest
type OpenAPIUpdateWorkflowRequest rawUpdateWorkflowRequest
type OpenAPIUpdateHarnessRequest rawUpdateHarnessRequest
type OpenAPIValidateHarnessRequest rawValidateHarnessRequest
type OpenAPICreateScheduledJobRequest rawCreateScheduledJobRequest
type OpenAPIUpdateScheduledJobRequest rawUpdateScheduledJobRequest
type OpenAPIUpdateWorkflowSkillsRequest rawUpdateWorkflowSkillsRequest
type OpenAPIUpdateTicketRequest rawUpdateTicketRequest

func BuildOpenAPIDocument() (*openapi3.T, error) {
	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       "OpenASE Frontend Contract API",
			Version:     "0.1.0",
			Description: "OpenAPI contract for the current OpenASE web control-plane surface.",
		},
		Paths: openapi3.NewPaths(),
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{},
		},
		Tags: openapi3.Tags{
			{Name: "catalog"},
			{Name: "tickets"},
			{Name: "workflows"},
			{Name: "scheduled-jobs"},
			{Name: "skills"},
			{Name: "streams"},
			{Name: "hr-advisor"},
		},
	}

	builder := openAPISpecBuilder{doc: doc}
	if err := builder.addCatalogOperations(); err != nil {
		return nil, err
	}
	if err := builder.addWorkflowOperations(); err != nil {
		return nil, err
	}
	if err := builder.addScheduledJobOperations(); err != nil {
		return nil, err
	}
	if err := builder.addTicketOperations(); err != nil {
		return nil, err
	}
	if err := builder.addStreamOperations(); err != nil {
		return nil, err
	}

	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("validate openapi document: %w", err)
	}

	return doc, nil
}

func BuildOpenAPIJSON() ([]byte, error) {
	doc, err := BuildOpenAPIDocument()
	if err != nil {
		return nil, err
	}

	payload, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal openapi document: %w", err)
	}

	payload = append(payload, '\n')
	return payload, nil
}

func (s *Server) handleOpenAPI(c echo.Context) error {
	payload, err := BuildOpenAPIJSON()
	if err != nil {
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	return c.Blob(http.StatusOK, echo.MIMEApplicationJSON, payload)
}

type openAPISpecBuilder struct {
	doc *openapi3.T
}

func (b openAPISpecBuilder) addCatalogOperations() error {
	orgsGet, err := b.jsonOperation(
		"listOrganizations",
		"List organizations",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIOrganizationsResponse{},
		nil,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/orgs", http.MethodGet, orgsGet)

	orgsPost, err := b.jsonOperation(
		"createOrganization",
		"Create an organization",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPIOrganizationResponse{},
		OpenAPICreateOrganizationRequest{},
		http.StatusBadRequest,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/orgs", http.MethodPost, orgsPost)

	orgProjectsGet, err := b.jsonOperation(
		"listProjects",
		"List projects for an organization",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	orgProjectsGet.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}/projects", http.MethodGet, orgProjectsGet)

	orgProjectsPost, err := b.jsonOperation(
		"createProject",
		"Create a project",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPIProjectResponse{},
		OpenAPICreateProjectRequest{},
		http.StatusBadRequest,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	orgProjectsPost.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}/projects", http.MethodPost, orgProjectsPost)

	providersGet, err := b.jsonOperation(
		"listAgentProviders",
		"List agent providers for an organization",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentProvidersResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	providersGet.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}/providers", http.MethodGet, providersGet)

	providersPost, err := b.jsonOperation(
		"createAgentProvider",
		"Create an agent provider",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPIAgentProviderResponse{},
		OpenAPICreateAgentProviderRequest{},
		http.StatusBadRequest,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	providersPost.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}/providers", http.MethodPost, providersPost)

	orgPatch, err := b.jsonOperation(
		"updateOrganization",
		"Update an organization",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIOrganizationResponse{},
		OpenAPIUpdateOrganizationRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	orgPatch.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}", http.MethodPatch, orgPatch)

	projectGet, err := b.jsonOperation(
		"getProject",
		"Get a project",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}", http.MethodGet, projectGet)

	projectPatch, err := b.jsonOperation(
		"updateProject",
		"Update a project",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectResponse{},
		OpenAPIUpdateProjectRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectPatch.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}", http.MethodPatch, projectPatch)

	projectDelete, err := b.jsonOperation(
		"archiveProject",
		"Archive a project",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectDelete.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}", http.MethodDelete, projectDelete)

	providerPatch, err := b.jsonOperation(
		"updateAgentProvider",
		"Update an agent provider",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentProviderResponse{},
		OpenAPIUpdateAgentProviderRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	providerPatch.AddParameter(uuidPathParameter("providerId", "Agent provider ID."))
	b.doc.AddOperation("/api/v1/providers/{providerId}", http.MethodPatch, providerPatch)

	statusesGet, err := b.jsonOperation(
		"listTicketStatuses",
		"List ticket statuses",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPITicketStatusesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	statusesGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/statuses", http.MethodGet, statusesGet)

	agentsGet, err := b.jsonOperation(
		"listAgents",
		"List agents",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/agents", http.MethodGet, agentsGet)

	activityGet, err := b.jsonOperation(
		"listActivityEvents",
		"List activity events",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIActivityEventsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	activityGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	activityGet.AddParameter(uuidQueryParameter("agent_id", "Filter activity by agent ID."))
	activityGet.AddParameter(uuidQueryParameter("ticket_id", "Filter activity by ticket ID."))
	activityGet.AddParameter(intQueryParameter("limit", "Limit the number of returned activity events."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/activity", http.MethodGet, activityGet)

	rolesGet, err := b.jsonOperation(
		"listBuiltinRoles",
		"List builtin workflow role templates",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIRolesResponse{},
		nil,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/roles/builtin", http.MethodGet, rolesGet)

	hrAdvisorGet, err := b.jsonOperation(
		"getHRAdvisor",
		"Get HR advisor recommendations",
		[]string{"hr-advisor"},
		http.StatusOK,
		OpenAPIHRAdvisorResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	hrAdvisorGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/hr-advisor", http.MethodGet, hrAdvisorGet)

	return nil
}

func (b openAPISpecBuilder) addWorkflowOperations() error {
	workflowsGet, err := b.jsonOperation(
		"listWorkflows",
		"List workflows",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIWorkflowsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	workflowsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/workflows", http.MethodGet, workflowsGet)

	workflowsPost, err := b.jsonOperation(
		"createWorkflow",
		"Create a workflow",
		[]string{"workflows"},
		http.StatusCreated,
		OpenAPIWorkflowResponse{},
		OpenAPICreateWorkflowRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	workflowsPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/workflows", http.MethodPost, workflowsPost)

	workflowGet, err := b.jsonOperation(
		"getWorkflow",
		"Get a workflow",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIWorkflowResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	workflowGet.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}", http.MethodGet, workflowGet)

	workflowPatch, err := b.jsonOperation(
		"updateWorkflow",
		"Update a workflow",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIWorkflowResponse{},
		OpenAPIUpdateWorkflowRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	workflowPatch.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}", http.MethodPatch, workflowPatch)

	workflowDelete, err := b.jsonOperation(
		"deleteWorkflow",
		"Delete a workflow",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIWorkflowResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	workflowDelete.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}", http.MethodDelete, workflowDelete)

	harnessGet, err := b.jsonOperation(
		"getWorkflowHarness",
		"Get workflow harness content",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIHarnessResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	harnessGet.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}/harness", http.MethodGet, harnessGet)

	harnessPut, err := b.jsonOperation(
		"updateWorkflowHarness",
		"Update workflow harness content",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIHarnessResponse{},
		OpenAPIUpdateHarnessRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	harnessPut.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}/harness", http.MethodPut, harnessPut)

	harnessValidate, err := b.jsonOperation(
		"validateHarness",
		"Validate workflow harness content",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIHarnessValidationResponse{},
		OpenAPIValidateHarnessRequest{},
		http.StatusBadRequest,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/harness/validate", http.MethodPost, harnessValidate)

	skillsGet, err := b.jsonOperation(
		"listSkills",
		"List workflow skills",
		[]string{"skills"},
		http.StatusOK,
		OpenAPISkillsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	skillsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/skills", http.MethodGet, skillsGet)

	bindSkills, err := b.jsonOperation(
		"bindWorkflowSkills",
		"Bind skills to a workflow harness",
		[]string{"skills"},
		http.StatusOK,
		OpenAPIHarnessResponse{},
		OpenAPIUpdateWorkflowSkillsRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	bindSkills.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}/skills/bind", http.MethodPost, bindSkills)

	unbindSkills, err := b.jsonOperation(
		"unbindWorkflowSkills",
		"Unbind skills from a workflow harness",
		[]string{"skills"},
		http.StatusOK,
		OpenAPIHarnessResponse{},
		OpenAPIUpdateWorkflowSkillsRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	unbindSkills.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}/skills/unbind", http.MethodPost, unbindSkills)

	return nil
}

func (b openAPISpecBuilder) addScheduledJobOperations() error {
	scheduledJobsGet, err := b.jsonOperation(
		"listScheduledJobs",
		"List scheduled jobs for a project",
		[]string{"scheduled-jobs"},
		http.StatusOK,
		OpenAPIScheduledJobsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	scheduledJobsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/scheduled-jobs", http.MethodGet, scheduledJobsGet)

	scheduledJobsPost, err := b.jsonOperation(
		"createScheduledJob",
		"Create a scheduled job",
		[]string{"scheduled-jobs"},
		http.StatusCreated,
		OpenAPIScheduledJobResponse{},
		OpenAPICreateScheduledJobRequest{},
		http.StatusBadRequest,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	scheduledJobsPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/scheduled-jobs", http.MethodPost, scheduledJobsPost)

	scheduledJobPatch, err := b.jsonOperation(
		"updateScheduledJob",
		"Update a scheduled job",
		[]string{"scheduled-jobs"},
		http.StatusOK,
		OpenAPIScheduledJobResponse{},
		OpenAPIUpdateScheduledJobRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	scheduledJobPatch.AddParameter(uuidPathParameter("jobId", "Scheduled job ID."))
	b.doc.AddOperation("/api/v1/scheduled-jobs/{jobId}", http.MethodPatch, scheduledJobPatch)

	scheduledJobDelete, err := b.jsonOperation(
		"deleteScheduledJob",
		"Delete a scheduled job",
		[]string{"scheduled-jobs"},
		http.StatusOK,
		scheduledjobservice.DeleteResult{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	scheduledJobDelete.AddParameter(uuidPathParameter("jobId", "Scheduled job ID."))
	b.doc.AddOperation("/api/v1/scheduled-jobs/{jobId}", http.MethodDelete, scheduledJobDelete)

	scheduledJobTrigger, err := b.jsonOperation(
		"triggerScheduledJob",
		"Trigger a scheduled job once",
		[]string{"scheduled-jobs"},
		http.StatusOK,
		OpenAPIScheduledJobTriggerResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	scheduledJobTrigger.AddParameter(uuidPathParameter("jobId", "Scheduled job ID."))
	b.doc.AddOperation("/api/v1/scheduled-jobs/{jobId}/trigger", http.MethodPost, scheduledJobTrigger)

	return nil
}

func (b openAPISpecBuilder) addTicketOperations() error {
	ticketsGet, err := b.jsonOperation(
		"listTickets",
		"List tickets",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	ticketsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	ticketsGet.AddParameter(csvQueryParameter("status_name", "Filter tickets by status names."))
	ticketsGet.AddParameter(csvQueryParameter("priority", "Filter tickets by priorities."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/tickets", http.MethodGet, ticketsGet)

	ticketPatch, err := b.jsonOperation(
		"updateTicket",
		"Update a ticket",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketResponse{},
		OpenAPIUpdateTicketRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	ticketPatch.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}", http.MethodPatch, ticketPatch)

	ticketDetailGet, err := b.jsonOperation(
		"getTicketDetail",
		"Get ticket detail",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketDetailResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	ticketDetailGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	ticketDetailGet.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/tickets/{ticketId}/detail", http.MethodGet, ticketDetailGet)

	return nil
}

func (b openAPISpecBuilder) addStreamOperations() error {
	globalStream, err := b.streamOperation(
		"streamEvents",
		"Stream global platform events",
		[]string{"streams"},
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/events/stream", http.MethodGet, globalStream)

	projectStreams := []struct {
		path        string
		operationID string
		summary     string
	}{
		{path: "/api/v1/projects/{projectId}/tickets/stream", operationID: "streamProjectTickets", summary: "Stream project ticket events"},
		{path: "/api/v1/projects/{projectId}/agents/stream", operationID: "streamProjectAgents", summary: "Stream project agent events"},
		{path: "/api/v1/projects/{projectId}/activity/stream", operationID: "streamProjectActivity", summary: "Stream project activity events"},
		{path: "/api/v1/projects/{projectId}/hooks/stream", operationID: "streamProjectHooks", summary: "Stream project hook events"},
	}
	for _, item := range projectStreams {
		op, err := b.streamOperation(item.operationID, item.summary, []string{"streams"}, http.StatusBadRequest, http.StatusInternalServerError)
		if err != nil {
			return err
		}
		op.AddParameter(uuidPathParameter("projectId", "Project ID."))
		b.doc.AddOperation(item.path, http.MethodGet, op)
	}

	return nil
}

func (b openAPISpecBuilder) jsonOperation(
	operationID string,
	summary string,
	tags []string,
	successCode int,
	successBody any,
	requestBody any,
	errorCodes ...int,
) (*openapi3.Operation, error) {
	op := openapi3.NewOperation()
	op.OperationID = operationID
	op.Summary = summary
	op.Tags = append([]string(nil), tags...)
	op.Responses = openapi3.NewResponsesWithCapacity(1 + len(errorCodes))

	successResponse, err := b.jsonResponse(summary, successBody)
	if err != nil {
		return nil, err
	}
	op.AddResponse(successCode, successResponse)

	if requestBody != nil {
		bodyRef, err := b.schemaRef(requestBody)
		if err != nil {
			return nil, err
		}
		op.RequestBody = &openapi3.RequestBodyRef{
			Value: openapi3.NewRequestBody().
				WithDescription(summary + " request body.").
				WithJSONSchemaRef(bodyRef).
				WithRequired(true),
		}
	}

	for _, code := range errorCodes {
		errorResponse, err := b.errorResponse(code)
		if err != nil {
			return nil, err
		}
		op.AddResponse(code, errorResponse)
	}

	return op, nil
}

func (b openAPISpecBuilder) streamOperation(
	operationID string,
	summary string,
	tags []string,
	errorCodes ...int,
) (*openapi3.Operation, error) {
	op := openapi3.NewOperation()
	op.OperationID = operationID
	op.Summary = summary
	op.Tags = append([]string(nil), tags...)
	op.Responses = openapi3.NewResponsesWithCapacity(1 + len(errorCodes))
	op.AddResponse(http.StatusOK, openapi3.NewResponse().
		WithDescription("Server-sent events stream.").
		WithContent(openapi3.Content{
			"text/event-stream": &openapi3.MediaType{
				Schema: openapi3.NewStringSchema().NewRef(),
			},
		}),
	)
	for _, code := range errorCodes {
		errorResponse, err := b.errorResponse(code)
		if err != nil {
			return nil, err
		}
		op.AddResponse(code, errorResponse)
	}
	return op, nil
}

func (b openAPISpecBuilder) jsonResponse(summary string, body any) (*openapi3.Response, error) {
	schemaRef, err := b.schemaRef(body)
	if err != nil {
		return nil, err
	}

	return openapi3.NewResponse().
		WithDescription(summary + " response.").
		WithContent(openapi3.Content{
			"application/json": &openapi3.MediaType{Schema: schemaRef},
		}), nil
}

func (b openAPISpecBuilder) errorResponse(statusCode int) (*openapi3.Response, error) {
	schemaRef, err := b.schemaRef(OpenAPIErrorResponse{})
	if err != nil {
		return nil, err
	}

	return openapi3.NewResponse().
		WithDescription(http.StatusText(statusCode) + " response.").
		WithContent(openapi3.Content{
			"application/json": &openapi3.MediaType{Schema: schemaRef},
		}), nil
}

func (b openAPISpecBuilder) schemaRef(value any) (*openapi3.SchemaRef, error) {
	ref, err := openapi3gen.NewSchemaRefForValue(
		value,
		b.doc.Components.Schemas,
		openapi3gen.UseAllExportedFields(),
	)
	if err != nil {
		return nil, fmt.Errorf("generate schema for %T: %w", value, err)
	}

	return ref, nil
}

func uuidPathParameter(name string, description string) *openapi3.Parameter {
	return openapi3.NewPathParameter(name).
		WithDescription(description).
		WithRequired(true).
		WithSchema(openapi3.NewUUIDSchema())
}

func uuidQueryParameter(name string, description string) *openapi3.Parameter {
	return openapi3.NewQueryParameter(name).
		WithDescription(description).
		WithSchema(openapi3.NewUUIDSchema())
}

func intQueryParameter(name string, description string) *openapi3.Parameter {
	return openapi3.NewQueryParameter(name).
		WithDescription(description).
		WithSchema(openapi3.NewIntegerSchema())
}

func csvQueryParameter(name string, description string) *openapi3.Parameter {
	return openapi3.NewQueryParameter(name).
		WithDescription(description + " Accepts a comma-separated string.").
		WithSchema(openapi3.NewStringSchema())
}
