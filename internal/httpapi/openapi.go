package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	notificationdomain "github.com/BetterAndBetterII/openase/internal/domain/notification"
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
	Status                 string  `json:"status"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id,omitempty"`
}

type OpenAPIProject struct {
	ID                     string   `json:"id"`
	OrganizationID         string   `json:"organization_id"`
	Name                   string   `json:"name"`
	Slug                   string   `json:"slug"`
	Description            string   `json:"description"`
	Status                 string   `json:"status"`
	DefaultWorkflowID      *string  `json:"default_workflow_id,omitempty"`
	DefaultAgentProviderID *string  `json:"default_agent_provider_id,omitempty"`
	AccessibleMachineIDs   []string `json:"accessible_machine_ids,omitempty"`
	MaxConcurrentAgents    int      `json:"max_concurrent_agents"`
}

type OpenAPIMachine struct {
	ID              string         `json:"id"`
	OrganizationID  string         `json:"organization_id"`
	Name            string         `json:"name"`
	Host            string         `json:"host"`
	Port            int            `json:"port"`
	SSHUser         *string        `json:"ssh_user,omitempty"`
	SSHKeyPath      *string        `json:"ssh_key_path,omitempty"`
	Description     string         `json:"description"`
	Labels          []string       `json:"labels,omitempty"`
	Status          string         `json:"status"`
	WorkspaceRoot   *string        `json:"workspace_root,omitempty"`
	AgentCLIPath    *string        `json:"agent_cli_path,omitempty"`
	EnvVars         []string       `json:"env_vars,omitempty"`
	LastHeartbeatAt *string        `json:"last_heartbeat_at,omitempty"`
	Resources       map[string]any `json:"resources"`
}

type OpenAPIMachineProbe struct {
	CheckedAt string         `json:"checked_at"`
	Transport string         `json:"transport"`
	Output    string         `json:"output"`
	Resources map[string]any `json:"resources"`
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
	ID                    string         `json:"id"`
	OrganizationID        string         `json:"organization_id"`
	MachineID             string         `json:"machine_id"`
	MachineName           string         `json:"machine_name"`
	MachineHost           string         `json:"machine_host"`
	MachineStatus         string         `json:"machine_status"`
	MachineSSHUser        *string        `json:"machine_ssh_user,omitempty"`
	MachineWorkspaceRoot  *string        `json:"machine_workspace_root,omitempty"`
	Name                  string         `json:"name"`
	AdapterType           string         `json:"adapter_type"`
	AvailabilityState     string         `json:"availability_state"`
	Available             bool           `json:"available"`
	AvailabilityCheckedAt *string        `json:"availability_checked_at,omitempty"`
	AvailabilityReason    *string        `json:"availability_reason,omitempty"`
	CliCommand            string         `json:"cli_command"`
	CliArgs               []string       `json:"cli_args"`
	AuthConfig            map[string]any `json:"auth_config"`
	ModelName             string         `json:"model_name"`
	ModelTemperature      float64        `json:"model_temperature"`
	ModelMaxTokens        int            `json:"model_max_tokens"`
	CostPerInputToken     float64        `json:"cost_per_input_token"`
	CostPerOutputToken    float64        `json:"cost_per_output_token"`
}

type OpenAPIAgent struct {
	ID                    string               `json:"id"`
	ProviderID            string               `json:"provider_id"`
	ProjectID             string               `json:"project_id"`
	Name                  string               `json:"name"`
	RuntimeControlState   string               `json:"runtime_control_state"`
	TotalTokensUsed       int64                `json:"total_tokens_used"`
	TotalTicketsCompleted int                  `json:"total_tickets_completed"`
	Runtime               *OpenAPIAgentRuntime `json:"runtime,omitempty"`
}

type OpenAPIAgentRuntime struct {
	ActiveRunCount   int     `json:"active_run_count"`
	CurrentRunID     *string `json:"current_run_id,omitempty"`
	Status           string  `json:"status"`
	CurrentTicketID  *string `json:"current_ticket_id,omitempty"`
	SessionID        string  `json:"session_id"`
	RuntimePhase     string  `json:"runtime_phase"`
	RuntimeStartedAt *string `json:"runtime_started_at,omitempty"`
	LastError        string  `json:"last_error"`
	LastHeartbeatAt  *string `json:"last_heartbeat_at,omitempty"`
}

type OpenAPIAgentRun struct {
	ID               string  `json:"id"`
	AgentID          string  `json:"agent_id"`
	WorkflowID       string  `json:"workflow_id"`
	TicketID         string  `json:"ticket_id"`
	ProviderID       string  `json:"provider_id"`
	Status           string  `json:"status"`
	SessionID        string  `json:"session_id"`
	RuntimeStartedAt *string `json:"runtime_started_at,omitempty"`
	LastError        string  `json:"last_error"`
	LastHeartbeatAt  *string `json:"last_heartbeat_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
}

type OpenAPIAgentRuntimeControlResponse struct {
	Agent       OpenAPIAgent `json:"agent"`
	Transition  string       `json:"transition"`
	RequestedAt string       `json:"requested_at"`
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

type OpenAPIAgentOutputEntry struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"project_id"`
	AgentID   string  `json:"agent_id"`
	TicketID  *string `json:"ticket_id,omitempty"`
	Stream    string  `json:"stream"`
	Output    string  `json:"output"`
	CreatedAt string  `json:"created_at"`
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

type OpenAPITicketExternalLink struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	URL        string `json:"url"`
	ExternalID string `json:"external_id"`
	Title      string `json:"title,omitempty"`
	Status     string `json:"status,omitempty"`
	Relation   string `json:"relation"`
	CreatedAt  string `json:"created_at"`
}

type OpenAPITicketComment struct {
	ID        string  `json:"id"`
	TicketID  string  `json:"ticket_id"`
	Body      string  `json:"body"`
	CreatedBy string  `json:"created_by"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt *string `json:"updated_at,omitempty"`
}

type OpenAPITicket struct {
	ID                string                      `json:"id"`
	ProjectID         string                      `json:"project_id"`
	Identifier        string                      `json:"identifier"`
	Title             string                      `json:"title"`
	Description       string                      `json:"description"`
	StatusID          string                      `json:"status_id"`
	StatusName        string                      `json:"status_name"`
	Priority          string                      `json:"priority"`
	Type              string                      `json:"type"`
	WorkflowID        *string                     `json:"workflow_id,omitempty"`
	CurrentRunID      *string                     `json:"current_run_id,omitempty"`
	TargetMachineID   *string                     `json:"target_machine_id,omitempty"`
	CreatedBy         string                      `json:"created_by"`
	Parent            *OpenAPITicketReference     `json:"parent,omitempty"`
	Children          []OpenAPITicketReference    `json:"children"`
	Dependencies      []OpenAPITicketDependency   `json:"dependencies"`
	ExternalLinks     []OpenAPITicketExternalLink `json:"external_links"`
	ExternalRef       string                      `json:"external_ref"`
	BudgetUSD         float64                     `json:"budget_usd"`
	CostTokensInput   int64                       `json:"cost_tokens_input"`
	CostTokensOutput  int64                       `json:"cost_tokens_output"`
	CostAmount        float64                     `json:"cost_amount"`
	AttemptCount      int                         `json:"attempt_count"`
	ConsecutiveErrors int                         `json:"consecutive_errors"`
	StartedAt         *string                     `json:"started_at,omitempty"`
	CompletedAt       *string                     `json:"completed_at,omitempty"`
	NextRetryAt       *string                     `json:"next_retry_at,omitempty"`
	RetryPaused       bool                        `json:"retry_paused"`
	PauseReason       string                      `json:"pause_reason,omitempty"`
	CreatedAt         string                      `json:"created_at"`
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

type OpenAPITicketRepoScope struct {
	ID             string  `json:"id"`
	TicketID       string  `json:"ticket_id"`
	RepoID         string  `json:"repo_id"`
	BranchName     string  `json:"branch_name"`
	PullRequestURL *string `json:"pull_request_url,omitempty"`
	PrStatus       string  `json:"pr_status"`
	CiStatus       string  `json:"ci_status"`
	IsPrimaryScope bool    `json:"is_primary_scope"`
}

type OpenAPIChatContext struct {
	ProjectID  string  `json:"project_id"`
	WorkflowID *string `json:"workflow_id,omitempty"`
	TicketID   *string `json:"ticket_id,omitempty"`
}

type OpenAPIChatStartRequest struct {
	Message   string             `json:"message"`
	Source    string             `json:"source"`
	Context   OpenAPIChatContext `json:"context"`
	SessionID *string            `json:"session_id,omitempty"`
}

type OpenAPITicketStatus struct {
	ID          string              `json:"id"`
	ProjectID   string              `json:"project_id"`
	StageID     *string             `json:"stage_id,omitempty"`
	Stage       *OpenAPITicketStage `json:"stage,omitempty"`
	Name        string              `json:"name"`
	Color       string              `json:"color"`
	Icon        string              `json:"icon"`
	Position    int                 `json:"position"`
	IsDefault   bool                `json:"is_default"`
	Description string              `json:"description"`
}

type OpenAPITicketStage struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	Key           string `json:"key"`
	Name          string `json:"name"`
	Position      int    `json:"position"`
	ActiveRuns    int    `json:"active_runs"`
	MaxActiveRuns *int   `json:"max_active_runs,omitempty"`
	Description   string `json:"description"`
}

type OpenAPITicketStatusGroup struct {
	Stage    *OpenAPITicketStage   `json:"stage,omitempty"`
	Statuses []OpenAPITicketStatus `json:"statuses"`
}

type OpenAPIWorkflow struct {
	ID                  string         `json:"id"`
	ProjectID           string         `json:"project_id"`
	AgentID             *string        `json:"agent_id,omitempty"`
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
	PickupStatusIDs     []string       `json:"pickup_status_ids"`
	FinishStatusIDs     []string       `json:"finish_status_ids"`
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

type OpenAPIHarnessVariableMetadata struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

type OpenAPIHarnessVariableGroup struct {
	Name      string                           `json:"name"`
	Variables []OpenAPIHarnessVariableMetadata `json:"variables"`
}

type OpenAPIHarnessVariablesResponse struct {
	Groups []OpenAPIHarnessVariableGroup `json:"groups"`
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

type OpenAPINotificationChannel struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Config         map[string]any `json:"config"`
	IsEnabled      bool           `json:"is_enabled"`
	CreatedAt      string         `json:"created_at"`
}

type OpenAPINotificationRule struct {
	ID        string                     `json:"id"`
	ProjectID string                     `json:"project_id"`
	ChannelID string                     `json:"channel_id"`
	Name      string                     `json:"name"`
	EventType string                     `json:"event_type"`
	Filter    map[string]any             `json:"filter"`
	Template  string                     `json:"template"`
	IsEnabled bool                       `json:"is_enabled"`
	CreatedAt string                     `json:"created_at"`
	Channel   OpenAPINotificationChannel `json:"channel"`
}

type OpenAPINotificationRuleEventType struct {
	EventType       string `json:"event_type"`
	Label           string `json:"label"`
	DefaultTemplate string `json:"default_template"`
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

type OpenAPIMachinesResponse struct {
	Machines []OpenAPIMachine `json:"machines"`
}

type OpenAPIMachineResponse struct {
	Machine OpenAPIMachine `json:"machine"`
}

type OpenAPIMachineTestResponse struct {
	Machine OpenAPIMachine      `json:"machine"`
	Probe   OpenAPIMachineProbe `json:"probe"`
}

type OpenAPIMachineResourcesResponse struct {
	MachineID               string                                `json:"machine_id"`
	Status                  string                                `json:"status"`
	LastHeartbeatAt         *string                               `json:"last_heartbeat_at,omitempty"`
	Resources               map[string]any                        `json:"resources"`
	EnvironmentProvisioning OpenAPIMachineEnvironmentProvisioning `json:"environment_provisioning"`
}

type OpenAPIMachineEnvironmentProvisioning struct {
	Available         bool                             `json:"available"`
	Needed            bool                             `json:"needed"`
	Runnable          bool                             `json:"runnable"`
	RoleSlug          string                           `json:"role_slug"`
	RoleName          string                           `json:"role_name"`
	RequiredSkills    []string                         `json:"required_skills"`
	Summary           string                           `json:"summary"`
	Issues            []OpenAPIMachineEnvironmentIssue `json:"issues"`
	Notes             []string                         `json:"notes"`
	TicketTitle       string                           `json:"ticket_title"`
	TicketDescription string                           `json:"ticket_description"`
}

type OpenAPIMachineEnvironmentIssue struct {
	Code      string  `json:"code"`
	Source    string  `json:"source"`
	Title     string  `json:"title"`
	Detail    string  `json:"detail"`
	SkillName *string `json:"skill_name,omitempty"`
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

type OpenAPIAgentRunsResponse struct {
	AgentRuns []OpenAPIAgentRun `json:"agent_runs"`
}

type OpenAPIAgentResponse struct {
	Agent OpenAPIAgent `json:"agent"`
}

type OpenAPIAgentOutputEntriesResponse struct {
	Entries []OpenAPIAgentOutputEntry `json:"entries"`
}

type OpenAPIActivityEventsResponse struct {
	Events []OpenAPIActivityEvent `json:"events"`
}

type OpenAPITicketStatusesResponse struct {
	Stages      []OpenAPITicketStage       `json:"stages"`
	Statuses    []OpenAPITicketStatus      `json:"statuses"`
	StageGroups []OpenAPITicketStatusGroup `json:"stage_groups"`
}

type OpenAPITicketStatusResponse struct {
	Status OpenAPITicketStatus `json:"status"`
}

type OpenAPITicketStatusDeleteResponse struct {
	DeletedStatusID     string `json:"deleted_status_id"`
	ReplacementStatusID string `json:"replacement_status_id"`
}

type OpenAPITicketStagesResponse struct {
	Stages []OpenAPITicketStage `json:"stages"`
}

type OpenAPITicketStageResponse struct {
	Stage OpenAPITicketStage `json:"stage"`
}

type OpenAPITicketStageDeleteResponse struct {
	DeletedStageID   string `json:"deleted_stage_id"`
	DetachedStatuses int    `json:"detached_statuses"`
}

type OpenAPITicketsResponse struct {
	Tickets []OpenAPITicket `json:"tickets"`
}

type OpenAPITicketResponse struct {
	Ticket OpenAPITicket `json:"ticket"`
}

type OpenAPITicketExternalLinkResponse struct {
	ExternalLink OpenAPITicketExternalLink `json:"external_link"`
}

type OpenAPIDeleteTicketExternalLinkResponse struct {
	DeletedExternalLinkID string `json:"deleted_external_link_id"`
}

type OpenAPITicketDependencyResponse struct {
	Dependency OpenAPITicketDependency `json:"dependency"`
}

type OpenAPITicketDependencyDeleteResponse struct {
	DeletedDependencyID string `json:"deleted_dependency_id"`
}

type OpenAPITicketCommentResponse struct {
	Comment OpenAPITicketComment `json:"comment"`
}

type OpenAPITicketCommentDeleteResponse struct {
	DeletedCommentID string `json:"deleted_comment_id"`
}

type OpenAPIProjectReposResponse struct {
	Repos []OpenAPIProjectRepo `json:"repos"`
}

type OpenAPIProjectRepoResponse struct {
	Repo OpenAPIProjectRepo `json:"repo"`
}

type OpenAPITicketRepoScopesResponse struct {
	RepoScopes []OpenAPITicketRepoScope `json:"repo_scopes"`
}

type OpenAPITicketRepoScopeResponse struct {
	RepoScope OpenAPITicketRepoScope `json:"repo_scope"`
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

type OpenAPINotificationRuleEventTypesResponse struct {
	EventTypes []OpenAPINotificationRuleEventType `json:"event_types"`
}

type OpenAPINotificationChannelsResponse struct {
	Channels []OpenAPINotificationChannel `json:"channels"`
}

type OpenAPINotificationChannelResponse struct {
	Channel OpenAPINotificationChannel `json:"channel"`
}

type OpenAPINotificationChannelDeleteResponse struct {
	DeletedChannelID string `json:"deleted_channel_id"`
}

type OpenAPINotificationChannelTestResponse struct {
	Status string `json:"status"`
}

type OpenAPINotificationRulesResponse struct {
	Rules []OpenAPINotificationRule `json:"rules"`
}

type OpenAPINotificationRuleResponse struct {
	Rule OpenAPINotificationRule `json:"rule"`
}

type OpenAPINotificationRuleDeleteResponse struct {
	DeletedRuleID string `json:"deleted_rule_id"`
}

type OpenAPISecurityDeferredCapability struct {
	Key     string `json:"key"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

type OpenAPISecurityAgentTokens struct {
	Transport              string   `json:"transport"`
	EnvironmentVariable    string   `json:"environment_variable"`
	TokenPrefix            string   `json:"token_prefix"`
	DefaultScopes          []string `json:"default_scopes"`
	SupportedProjectScopes []string `json:"supported_project_scopes"`
}

type OpenAPISecurityWebhooks struct {
	LegacyGitHubEndpoint          string `json:"legacy_github_endpoint"`
	ConnectorEndpoint             string `json:"connector_endpoint"`
	LegacyGitHubSignatureRequired bool   `json:"legacy_github_signature_required"`
}

type OpenAPISecuritySecretHygiene struct {
	NotificationChannelConfigsRedacted bool `json:"notification_channel_configs_redacted"`
}

type OpenAPISecuritySettings struct {
	ProjectID     string                              `json:"project_id"`
	AgentTokens   OpenAPISecurityAgentTokens          `json:"agent_tokens"`
	Webhooks      OpenAPISecurityWebhooks             `json:"webhooks"`
	SecretHygiene OpenAPISecuritySecretHygiene        `json:"secret_hygiene"`
	Deferred      []OpenAPISecurityDeferredCapability `json:"deferred"`
}

type OpenAPISecuritySettingsResponse struct {
	Security OpenAPISecuritySettings `json:"security"`
}

type OpenAPITicketDetailResponse struct {
	Ticket      OpenAPITicket                  `json:"ticket"`
	RepoScopes  []OpenAPITicketRepoScopeDetail `json:"repo_scopes"`
	Comments    []OpenAPITicketComment         `json:"comments"`
	Activity    []OpenAPIActivityEvent         `json:"activity"`
	HookHistory []OpenAPIActivityEvent         `json:"hook_history"`
}

type OpenAPICreateOrganizationRequest catalogdomain.OrganizationInput
type OpenAPIUpdateOrganizationRequest organizationPatchRequest
type OpenAPICreateAgentProviderRequest catalogdomain.AgentProviderInput
type OpenAPIUpdateAgentProviderRequest agentProviderPatchRequest
type OpenAPICreateProjectRequest catalogdomain.ProjectInput
type OpenAPIUpdateProjectRequest projectPatchRequest
type OpenAPICreateMachineRequest catalogdomain.MachineInput
type OpenAPIUpdateMachineRequest machinePatchRequest
type OpenAPICreateProjectRepoRequest catalogdomain.ProjectRepoInput
type OpenAPIUpdateProjectRepoRequest projectRepoPatchRequest
type OpenAPICreateTicketRepoScopeRequest catalogdomain.TicketRepoScopeInput
type OpenAPIUpdateTicketRepoScopeRequest ticketRepoScopePatchRequest
type OpenAPICreateAgentRequest catalogdomain.AgentInput
type OpenAPICreateWorkflowRequest rawCreateWorkflowRequest
type OpenAPIUpdateWorkflowRequest rawUpdateWorkflowRequest
type OpenAPIUpdateHarnessRequest rawUpdateHarnessRequest
type OpenAPIValidateHarnessRequest rawValidateHarnessRequest
type OpenAPICreateScheduledJobRequest rawCreateScheduledJobRequest
type OpenAPIUpdateScheduledJobRequest rawUpdateScheduledJobRequest
type OpenAPIUpdateWorkflowSkillsRequest rawUpdateWorkflowSkillsRequest
type OpenAPICreateTicketRequest rawCreateTicketRequest
type OpenAPIUpdateTicketRequest rawUpdateTicketRequest
type OpenAPICreateTicketCommentRequest rawCreateTicketCommentRequest
type OpenAPIUpdateTicketCommentRequest rawUpdateTicketCommentRequest
type OpenAPIAddTicketDependencyRequest rawAddDependencyRequest
type OpenAPICreateTicketExternalLinkRequest rawAddExternalLinkRequest
type OpenAPICreateTicketStageRequest struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Position      *int   `json:"position"`
	MaxActiveRuns *int   `json:"max_active_runs"`
	Description   string `json:"description"`
}

type OpenAPIUpdateTicketStageRequest struct {
	Name          *string `json:"name"`
	Position      *int    `json:"position"`
	MaxActiveRuns *int    `json:"max_active_runs"`
	Description   *string `json:"description"`
}

type OpenAPICreateTicketStatusRequest struct {
	StageID     *string `json:"stage_id"`
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	Icon        string  `json:"icon"`
	Position    *int    `json:"position"`
	IsDefault   bool    `json:"is_default"`
	Description string  `json:"description"`
}

type OpenAPIUpdateTicketStatusRequest struct {
	StageID     *string `json:"stage_id"`
	Name        *string `json:"name"`
	Color       *string `json:"color"`
	Icon        *string `json:"icon"`
	Position    *int    `json:"position"`
	IsDefault   *bool   `json:"is_default"`
	Description *string `json:"description"`
}
type OpenAPICreateNotificationChannelRequest notificationdomain.ChannelInput
type OpenAPIUpdateNotificationChannelRequest notificationdomain.ChannelPatchInput
type OpenAPICreateNotificationRuleRequest notificationdomain.RuleInput
type OpenAPIUpdateNotificationRuleRequest notificationdomain.RulePatchInput

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
			{Name: "system"},
			{Name: "catalog"},
			{Name: "tickets"},
			{Name: "workflows"},
			{Name: "scheduled-jobs"},
			{Name: "skills"},
			{Name: "streams"},
			{Name: "hr-advisor"},
			{Name: "notifications"},
			{Name: "security-settings"},
		},
	}

	builder := openAPISpecBuilder{doc: doc}
	if err := builder.addSystemOperations(); err != nil {
		return nil, err
	}
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
	if err := builder.addNotificationOperations(); err != nil {
		return nil, err
	}
	if err := builder.addSecurityOperations(); err != nil {
		return nil, err
	}
	if err := builder.addChatOperations(); err != nil {
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

func (b openAPISpecBuilder) addSystemOperations() error {
	dashboardGet, err := b.jsonOperation(
		"getSystemDashboard",
		"Get process memory and runtime dashboard metrics",
		[]string{"system"},
		http.StatusOK,
		OpenAPISystemDashboardResponse{},
		nil,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/system/dashboard", http.MethodGet, dashboardGet)

	return nil
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

	machinesGet, err := b.jsonOperation(
		"listMachines",
		"List machines for an organization",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIMachinesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	machinesGet.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}/machines", http.MethodGet, machinesGet)

	machinesPost, err := b.jsonOperation(
		"createMachine",
		"Create a machine",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPIMachineResponse{},
		OpenAPICreateMachineRequest{},
		http.StatusBadRequest,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	machinesPost.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}/machines", http.MethodPost, machinesPost)

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

	machineGet, err := b.jsonOperation(
		"getMachine",
		"Get a machine",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIMachineResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	machineGet.AddParameter(uuidPathParameter("machineId", "Machine ID."))
	b.doc.AddOperation("/api/v1/machines/{machineId}", http.MethodGet, machineGet)

	machinePatch, err := b.jsonOperation(
		"updateMachine",
		"Update a machine",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIMachineResponse{},
		OpenAPIUpdateMachineRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	machinePatch.AddParameter(uuidPathParameter("machineId", "Machine ID."))
	b.doc.AddOperation("/api/v1/machines/{machineId}", http.MethodPatch, machinePatch)

	machineDelete, err := b.jsonOperation(
		"deleteMachine",
		"Delete a machine",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIMachineResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	machineDelete.AddParameter(uuidPathParameter("machineId", "Machine ID."))
	b.doc.AddOperation("/api/v1/machines/{machineId}", http.MethodDelete, machineDelete)

	machineTest, err := b.jsonOperation(
		"testMachineConnection",
		"Test a machine SSH connection",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIMachineTestResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	machineTest.AddParameter(uuidPathParameter("machineId", "Machine ID."))
	b.doc.AddOperation("/api/v1/machines/{machineId}/test", http.MethodPost, machineTest)

	machineResources, err := b.jsonOperation(
		"getMachineResources",
		"Get machine resources",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIMachineResourcesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	machineResources.AddParameter(uuidPathParameter("machineId", "Machine ID."))
	b.doc.AddOperation("/api/v1/machines/{machineId}/resources", http.MethodGet, machineResources)

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

	orgDelete, err := b.jsonOperation(
		"archiveOrganization",
		"Archive an organization and all its projects",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIOrganizationResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	orgDelete.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}", http.MethodDelete, orgDelete)

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

	projectReposGet, err := b.jsonOperation(
		"listProjectRepos",
		"List project repositories",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectReposResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectReposGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/repos", http.MethodGet, projectReposGet)

	projectReposPost, err := b.jsonOperation(
		"createProjectRepo",
		"Create a project repository",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPIProjectRepoResponse{},
		OpenAPICreateProjectRepoRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectReposPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/repos", http.MethodPost, projectReposPost)

	projectRepoPatch, err := b.jsonOperation(
		"updateProjectRepo",
		"Update a project repository",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectRepoResponse{},
		OpenAPIUpdateProjectRepoRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectRepoPatch.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectRepoPatch.AddParameter(uuidPathParameter("repoId", "Repository ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/repos/{repoId}", http.MethodPatch, projectRepoPatch)

	projectRepoDelete, err := b.jsonOperation(
		"deleteProjectRepo",
		"Delete a project repository",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectRepoResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectRepoDelete.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectRepoDelete.AddParameter(uuidPathParameter("repoId", "Repository ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/repos/{repoId}", http.MethodDelete, projectRepoDelete)

	repoScopesGet, err := b.jsonOperation(
		"listTicketRepoScopes",
		"List ticket repository scopes",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPITicketRepoScopesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	repoScopesGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	repoScopesGet.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes", http.MethodGet, repoScopesGet)

	repoScopesPost, err := b.jsonOperation(
		"createTicketRepoScope",
		"Create a ticket repository scope",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPITicketRepoScopeResponse{},
		OpenAPICreateTicketRepoScopeRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	repoScopesPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	repoScopesPost.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes", http.MethodPost, repoScopesPost)

	repoScopePatch, err := b.jsonOperation(
		"updateTicketRepoScope",
		"Update a ticket repository scope",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPITicketRepoScopeResponse{},
		OpenAPIUpdateTicketRepoScopeRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	repoScopePatch.AddParameter(uuidPathParameter("projectId", "Project ID."))
	repoScopePatch.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	repoScopePatch.AddParameter(uuidPathParameter("scopeId", "Repository scope ID."))
	b.doc.AddOperation(
		"/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes/{scopeId}",
		http.MethodPatch,
		repoScopePatch,
	)

	repoScopeDelete, err := b.jsonOperation(
		"deleteTicketRepoScope",
		"Delete a ticket repository scope",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPITicketRepoScopeResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	repoScopeDelete.AddParameter(uuidPathParameter("projectId", "Project ID."))
	repoScopeDelete.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	repoScopeDelete.AddParameter(uuidPathParameter("scopeId", "Repository scope ID."))
	b.doc.AddOperation(
		"/api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes/{scopeId}",
		http.MethodDelete,
		repoScopeDelete,
	)

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

	stagesGet, err := b.jsonOperation(
		"listTicketStages",
		"List ticket stages",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPITicketStagesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	stagesGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/stages", http.MethodGet, stagesGet)

	stagesPost, err := b.jsonOperation(
		"createTicketStage",
		"Create a ticket stage",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPITicketStageResponse{},
		OpenAPICreateTicketStageRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	stagesPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/stages", http.MethodPost, stagesPost)

	stagePatch, err := b.jsonOperation(
		"updateTicketStage",
		"Update a ticket stage",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPITicketStageResponse{},
		OpenAPIUpdateTicketStageRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	stagePatch.AddParameter(uuidPathParameter("stageId", "Ticket stage ID."))
	b.doc.AddOperation("/api/v1/stages/{stageId}", http.MethodPatch, stagePatch)

	stageDelete, err := b.jsonOperation(
		"deleteTicketStage",
		"Delete a ticket stage",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPITicketStageDeleteResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	stageDelete.AddParameter(uuidPathParameter("stageId", "Ticket stage ID."))
	b.doc.AddOperation("/api/v1/stages/{stageId}", http.MethodDelete, stageDelete)

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

	statusesPost, err := b.jsonOperation(
		"createTicketStatus",
		"Create a ticket status",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPITicketStatusResponse{},
		OpenAPICreateTicketStatusRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	statusesPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/statuses", http.MethodPost, statusesPost)

	statusesReset, err := b.jsonOperation(
		"resetTicketStatuses",
		"Reset project statuses to the default template",
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
	statusesReset.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/statuses/reset", http.MethodPost, statusesReset)

	statusPatch, err := b.jsonOperation(
		"updateTicketStatus",
		"Update a ticket status",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPITicketStatusResponse{},
		OpenAPIUpdateTicketStatusRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	statusPatch.AddParameter(uuidPathParameter("statusId", "Ticket status ID."))
	b.doc.AddOperation("/api/v1/statuses/{statusId}", http.MethodPatch, statusPatch)

	statusDelete, err := b.jsonOperation(
		"deleteTicketStatus",
		"Delete a ticket status",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPITicketStatusDeleteResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	statusDelete.AddParameter(uuidPathParameter("statusId", "Ticket status ID."))
	b.doc.AddOperation("/api/v1/statuses/{statusId}", http.MethodDelete, statusDelete)

	agentsGet, err := b.jsonOperation(
		"listAgents",
		"List agent definitions with aggregate runtime summaries",
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

	agentRunsGet, err := b.jsonOperation(
		"listAgentRuns",
		"List project agent runs",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentRunsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentRunsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/agent-runs", http.MethodGet, agentRunsGet)

	agentsPost, err := b.jsonOperation(
		"createAgent",
		"Create an agent definition",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPIAgentResponse{},
		OpenAPICreateAgentRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentsPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/agents", http.MethodPost, agentsPost)

	agentGet, err := b.jsonOperation(
		"getAgent",
		"Get an agent definition",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentGet.AddParameter(uuidPathParameter("agentId", "Agent ID."))
	b.doc.AddOperation("/api/v1/agents/{agentId}", http.MethodGet, agentGet)

	agentDelete, err := b.jsonOperation(
		"deleteAgent",
		"Delete an agent",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentDelete.AddParameter(uuidPathParameter("agentId", "Agent ID."))
	b.doc.AddOperation("/api/v1/agents/{agentId}", http.MethodDelete, agentDelete)

	agentPause, err := b.jsonOperation(
		"pauseAgent",
		"Pause an agent runtime",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentPause.AddParameter(uuidPathParameter("agentId", "Agent ID."))
	b.doc.AddOperation("/api/v1/agents/{agentId}/pause", http.MethodPost, agentPause)

	agentResume, err := b.jsonOperation(
		"resumeAgent",
		"Resume an agent runtime",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentResume.AddParameter(uuidPathParameter("agentId", "Agent ID."))
	b.doc.AddOperation("/api/v1/agents/{agentId}/resume", http.MethodPost, agentResume)

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

	agentOutputGet, err := b.jsonOperation(
		"listAgentOutput",
		"List agent output entries",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentOutputEntriesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentOutputGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	agentOutputGet.AddParameter(uuidPathParameter("agentId", "Agent ID."))
	agentOutputGet.AddParameter(uuidQueryParameter("ticket_id", "Filter output by ticket ID."))
	agentOutputGet.AddParameter(intQueryParameter("limit", "Limit the number of returned output entries."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/agents/{agentId}/output", http.MethodGet, agentOutputGet)

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

	harnessVariables, err := b.jsonOperation(
		"listHarnessVariables",
		"List the harness variable dictionary",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIHarnessVariablesResponse{},
		nil,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/harness/variables", http.MethodGet, harnessVariables)

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

	ticketsPost, err := b.jsonOperation(
		"createTicket",
		"Create a ticket",
		[]string{"tickets"},
		http.StatusCreated,
		OpenAPITicketResponse{},
		OpenAPICreateTicketRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	ticketsPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/tickets", http.MethodPost, ticketsPost)

	ticketGet, err := b.jsonOperation(
		"getTicket",
		"Get a ticket",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	ticketGet.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}", http.MethodGet, ticketGet)

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

	commentPost, err := b.jsonOperation(
		"createTicketComment",
		"Create a ticket comment",
		[]string{"tickets"},
		http.StatusCreated,
		OpenAPITicketCommentResponse{},
		OpenAPICreateTicketCommentRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	commentPost.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}/comments", http.MethodPost, commentPost)

	commentPatch, err := b.jsonOperation(
		"updateTicketComment",
		"Update a ticket comment",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketCommentResponse{},
		OpenAPIUpdateTicketCommentRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	commentPatch.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	commentPatch.AddParameter(uuidPathParameter("commentId", "Comment ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}/comments/{commentId}", http.MethodPatch, commentPatch)

	commentDelete, err := b.jsonOperation(
		"deleteTicketComment",
		"Delete a ticket comment",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketCommentDeleteResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	commentDelete.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	commentDelete.AddParameter(uuidPathParameter("commentId", "Comment ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}/comments/{commentId}", http.MethodDelete, commentDelete)

	dependencyPost, err := b.jsonOperation(
		"addTicketDependency",
		"Add a ticket dependency",
		[]string{"tickets"},
		http.StatusCreated,
		OpenAPITicketDependencyResponse{},
		OpenAPIAddTicketDependencyRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	dependencyPost.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}/dependencies", http.MethodPost, dependencyPost)

	dependencyDelete, err := b.jsonOperation(
		"deleteTicketDependency",
		"Delete a ticket dependency",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketDependencyDeleteResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	dependencyDelete.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	dependencyDelete.AddParameter(uuidPathParameter("dependencyId", "Dependency ID."))
	b.doc.AddOperation(
		"/api/v1/tickets/{ticketId}/dependencies/{dependencyId}",
		http.MethodDelete,
		dependencyDelete,
	)

	externalLinkPost, err := b.jsonOperation(
		"addTicketExternalLink",
		"Add an external link to a ticket",
		[]string{"tickets"},
		http.StatusCreated,
		OpenAPITicketExternalLinkResponse{},
		OpenAPICreateTicketExternalLinkRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	externalLinkPost.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}/external-links", http.MethodPost, externalLinkPost)

	externalLinkDelete, err := b.jsonOperation(
		"deleteTicketExternalLink",
		"Delete an external link from a ticket",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPIDeleteTicketExternalLinkResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	externalLinkDelete.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	externalLinkDelete.AddParameter(uuidPathParameter("externalLinkId", "External link ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}/external-links/{externalLinkId}", http.MethodDelete, externalLinkDelete)

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

func (b openAPISpecBuilder) addNotificationOperations() error {
	eventTypesGet, err := b.jsonOperation(
		"listNotificationRuleEventTypes",
		"List supported notification rule event types",
		[]string{"notifications"},
		http.StatusOK,
		OpenAPINotificationRuleEventTypesResponse{},
		nil,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/notification-event-types", http.MethodGet, eventTypesGet)

	channelsGet, err := b.jsonOperation(
		"listNotificationChannels",
		"List organization notification channels",
		[]string{"notifications"},
		http.StatusOK,
		OpenAPINotificationChannelsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	channelsGet.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}/channels", http.MethodGet, channelsGet)

	channelsPost, err := b.jsonOperation(
		"createNotificationChannel",
		"Create a notification channel",
		[]string{"notifications"},
		http.StatusCreated,
		OpenAPINotificationChannelResponse{},
		OpenAPICreateNotificationChannelRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	channelsPost.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}/channels", http.MethodPost, channelsPost)

	channelPatch, err := b.jsonOperation(
		"updateNotificationChannel",
		"Update a notification channel",
		[]string{"notifications"},
		http.StatusOK,
		OpenAPINotificationChannelResponse{},
		OpenAPIUpdateNotificationChannelRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	channelPatch.AddParameter(uuidPathParameter("channelId", "Notification channel ID."))
	b.doc.AddOperation("/api/v1/channels/{channelId}", http.MethodPatch, channelPatch)

	channelDelete, err := b.jsonOperation(
		"deleteNotificationChannel",
		"Delete a notification channel",
		[]string{"notifications"},
		http.StatusOK,
		OpenAPINotificationChannelDeleteResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	channelDelete.AddParameter(uuidPathParameter("channelId", "Notification channel ID."))
	b.doc.AddOperation("/api/v1/channels/{channelId}", http.MethodDelete, channelDelete)

	channelTest, err := b.jsonOperation(
		"testNotificationChannel",
		"Test a notification channel",
		[]string{"notifications"},
		http.StatusOK,
		OpenAPINotificationChannelTestResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	channelTest.AddParameter(uuidPathParameter("channelId", "Notification channel ID."))
	b.doc.AddOperation("/api/v1/channels/{channelId}/test", http.MethodPost, channelTest)

	rulesGet, err := b.jsonOperation(
		"listNotificationRules",
		"List project notification rules",
		[]string{"notifications"},
		http.StatusOK,
		OpenAPINotificationRulesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	rulesGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/notification-rules", http.MethodGet, rulesGet)

	rulesPost, err := b.jsonOperation(
		"createNotificationRule",
		"Create a notification rule",
		[]string{"notifications"},
		http.StatusCreated,
		OpenAPINotificationRuleResponse{},
		OpenAPICreateNotificationRuleRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	rulesPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/notification-rules", http.MethodPost, rulesPost)

	rulePatch, err := b.jsonOperation(
		"updateNotificationRule",
		"Update a notification rule",
		[]string{"notifications"},
		http.StatusOK,
		OpenAPINotificationRuleResponse{},
		OpenAPIUpdateNotificationRuleRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	rulePatch.AddParameter(uuidPathParameter("ruleId", "Notification rule ID."))
	b.doc.AddOperation("/api/v1/notification-rules/{ruleId}", http.MethodPatch, rulePatch)

	ruleDelete, err := b.jsonOperation(
		"deleteNotificationRule",
		"Delete a notification rule",
		[]string{"notifications"},
		http.StatusOK,
		OpenAPINotificationRuleDeleteResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	ruleDelete.AddParameter(uuidPathParameter("ruleId", "Notification rule ID."))
	b.doc.AddOperation("/api/v1/notification-rules/{ruleId}", http.MethodDelete, ruleDelete)

	return nil
}

func (b openAPISpecBuilder) addSecurityOperations() error {
	securityGet, err := b.jsonOperation(
		"getSecuritySettings",
		"Get project security settings posture",
		[]string{"security-settings"},
		http.StatusOK,
		OpenAPISecuritySettingsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	securityGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/security-settings", http.MethodGet, securityGet)

	return nil
}

func (b openAPISpecBuilder) addChatOperations() error {
	chatPost, err := b.streamOperation(
		"startEphemeralChat",
		"Start an ephemeral chat turn",
		[]string{"chat"},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	bodyRef, err := b.schemaRef(OpenAPIChatStartRequest{})
	if err != nil {
		return err
	}
	chatPost.RequestBody = &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().
			WithDescription("Start ephemeral chat request body.").
			WithJSONSchemaRef(bodyRef).
			WithRequired(true),
	}
	b.doc.AddOperation("/api/v1/chat", http.MethodPost, chatPost)

	chatDelete := openapi3.NewOperation()
	chatDelete.OperationID = "closeEphemeralChat"
	chatDelete.Summary = "Close an ephemeral chat session"
	chatDelete.Tags = []string{"chat"}
	chatDelete.Responses = openapi3.NewResponsesWithCapacity(3)
	chatDelete.AddResponse(http.StatusNoContent, openapi3.NewResponse().WithDescription("Chat session closed."))
	for _, code := range []int{http.StatusBadRequest, http.StatusInternalServerError} {
		errorResponse, err := b.errorResponse(code)
		if err != nil {
			return err
		}
		chatDelete.AddResponse(code, errorResponse)
	}
	chatDelete.AddParameter(openapi3.NewPathParameter("sessionId").
		WithDescription("Claude Code session ID.").
		WithRequired(true).
		WithSchema(openapi3.NewStringSchema()),
	)
	b.doc.AddOperation("/api/v1/chat/{sessionId}", http.MethodDelete, chatDelete)

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

	agentOutputStream, err := b.streamOperation(
		"streamAgentOutput",
		"Stream agent output entries",
		[]string{"streams"},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentOutputStream.AddParameter(uuidPathParameter("projectId", "Project ID."))
	agentOutputStream.AddParameter(uuidPathParameter("agentId", "Agent ID."))
	agentOutputStream.AddParameter(uuidQueryParameter("ticket_id", "Filter streamed output by ticket ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/agents/{agentId}/output/stream", http.MethodGet, agentOutputStream)

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
