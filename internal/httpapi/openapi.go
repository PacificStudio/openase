package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

type OpenAPIWorkspaceDashboardMetrics struct {
	OrganizationCount int     `json:"organization_count"`
	ProjectCount      int     `json:"project_count"`
	ProviderCount     int     `json:"provider_count"`
	RunningAgents     int     `json:"running_agents"`
	ActiveTickets     int     `json:"active_tickets"`
	TodayCost         float64 `json:"today_cost"`
	TotalTokens       int64   `json:"total_tokens"`
}

type OpenAPIWorkspaceOrganizationSummary struct {
	OrganizationID string  `json:"organization_id"`
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	ProjectCount   int     `json:"project_count"`
	ProviderCount  int     `json:"provider_count"`
	RunningAgents  int     `json:"running_agents"`
	ActiveTickets  int     `json:"active_tickets"`
	TodayCost      float64 `json:"today_cost"`
	TotalTokens    int64   `json:"total_tokens"`
}

type OpenAPIOrganizationDashboardMetrics struct {
	OrganizationID     string  `json:"organization_id"`
	ProjectCount       int     `json:"project_count"`
	ActiveProjectCount int     `json:"active_project_count"`
	ProviderCount      int     `json:"provider_count"`
	RunningAgents      int     `json:"running_agents"`
	ActiveTickets      int     `json:"active_tickets"`
	TodayCost          float64 `json:"today_cost"`
	TotalTokens        int64   `json:"total_tokens"`
}

type OpenAPIOrganizationProjectSummary struct {
	ProjectID      string  `json:"project_id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Status         string  `json:"status"`
	RunningAgents  int     `json:"running_agents"`
	ActiveTickets  int     `json:"active_tickets"`
	TodayCost      float64 `json:"today_cost"`
	TotalTokens    int64   `json:"total_tokens"`
	LastActivityAt *string `json:"last_activity_at,omitempty"`
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
	MirrorRoot      *string        `json:"mirror_root,omitempty"`
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
	ID               string   `json:"id"`
	ProjectID        string   `json:"project_id"`
	Name             string   `json:"name"`
	RepositoryURL    string   `json:"repository_url"`
	DefaultBranch    string   `json:"default_branch"`
	WorkspaceDirname string   `json:"workspace_dirname"`
	IsPrimary        bool     `json:"is_primary"`
	Labels           []string `json:"labels,omitempty"`
	MirrorCount      *int     `json:"mirror_count,omitempty"`
	MirrorState      *string  `json:"mirror_state,omitempty"`
	MirrorMachineID  *string  `json:"mirror_machine_id,omitempty"`
	LastSyncedAt     *string  `json:"last_synced_at,omitempty"`
	LastVerifiedAt   *string  `json:"last_verified_at,omitempty"`
	LastError        *string  `json:"last_error,omitempty"`
}

type OpenAPIProjectRepoMirror struct {
	ID             string  `json:"id"`
	ProjectID      string  `json:"project_id"`
	ProjectRepoID  string  `json:"project_repo_id"`
	MachineID      string  `json:"machine_id"`
	LocalPath      string  `json:"local_path"`
	State          string  `json:"state"`
	HeadCommit     *string `json:"head_commit,omitempty"`
	LastSyncedAt   *string `json:"last_synced_at,omitempty"`
	LastVerifiedAt *string `json:"last_verified_at,omitempty"`
	LastError      *string `json:"last_error,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
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
	MaxParallelRuns       int            `json:"max_parallel_runs"`
	CostPerInputToken     float64        `json:"cost_per_input_token"`
	CostPerOutputToken    float64        `json:"cost_per_output_token"`
}

type OpenAPIAgentProviderModelOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Recommended bool   `json:"recommended"`
	Preview     bool   `json:"preview"`
}

type OpenAPIAgentProviderModelCatalogEntry struct {
	AdapterType string                            `json:"adapter_type"`
	Options     []OpenAPIAgentProviderModelOption `json:"options"`
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
	ActiveRunCount       int     `json:"active_run_count"`
	CurrentRunID         *string `json:"current_run_id,omitempty"`
	Status               string  `json:"status"`
	CurrentTicketID      *string `json:"current_ticket_id,omitempty"`
	SessionID            string  `json:"session_id"`
	RuntimePhase         string  `json:"runtime_phase"`
	RuntimeStartedAt     *string `json:"runtime_started_at,omitempty"`
	LastError            string  `json:"last_error"`
	LastHeartbeatAt      *string `json:"last_heartbeat_at,omitempty"`
	CurrentStepStatus    *string `json:"current_step_status,omitempty"`
	CurrentStepSummary   *string `json:"current_step_summary,omitempty"`
	CurrentStepChangedAt *string `json:"current_step_changed_at,omitempty"`
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
	ID         string  `json:"id"`
	ProjectID  string  `json:"project_id"`
	AgentID    string  `json:"agent_id"`
	TicketID   *string `json:"ticket_id,omitempty"`
	AgentRunID string  `json:"agent_run_id"`
	Stream     string  `json:"stream"`
	Output     string  `json:"output"`
	CreatedAt  string  `json:"created_at"`
}

type OpenAPIAgentStepEntry struct {
	ID                 string  `json:"id"`
	ProjectID          string  `json:"project_id"`
	AgentID            string  `json:"agent_id"`
	TicketID           *string `json:"ticket_id,omitempty"`
	AgentRunID         string  `json:"agent_run_id"`
	StepStatus         string  `json:"step_status"`
	Summary            string  `json:"summary"`
	SourceTraceEventID *string `json:"source_trace_event_id,omitempty"`
	CreatedAt          string  `json:"created_at"`
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
	ID           string  `json:"id"`
	TicketID     string  `json:"ticket_id"`
	Body         string  `json:"body,omitempty"`
	BodyMarkdown string  `json:"body_markdown"`
	CreatedBy    string  `json:"created_by"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    *string `json:"updated_at,omitempty"`
	EditedAt     *string `json:"edited_at,omitempty"`
	EditCount    int     `json:"edit_count"`
	LastEditedBy *string `json:"last_edited_by,omitempty"`
	IsDeleted    bool    `json:"is_deleted"`
	DeletedAt    *string `json:"deleted_at,omitempty"`
	DeletedBy    *string `json:"deleted_by,omitempty"`
}

type OpenAPITicketCommentRevision struct {
	ID             string  `json:"id"`
	CommentID      string  `json:"comment_id"`
	RevisionNumber int     `json:"revision_number"`
	BodyMarkdown   string  `json:"body_markdown"`
	EditedBy       string  `json:"edited_by"`
	EditedAt       string  `json:"edited_at"`
	EditReason     *string `json:"edit_reason,omitempty"`
}

type OpenAPITicketTimelineItem struct {
	ID            string         `json:"id"`
	TicketID      string         `json:"ticket_id"`
	ItemType      string         `json:"item_type"`
	ActorName     string         `json:"actor_name"`
	ActorType     string         `json:"actor_type"`
	Title         *string        `json:"title,omitempty"`
	BodyMarkdown  *string        `json:"body_markdown,omitempty"`
	BodyText      *string        `json:"body_text,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	EditedAt      *string        `json:"edited_at,omitempty"`
	IsCollapsible bool           `json:"is_collapsible"`
	IsDeleted     bool           `json:"is_deleted"`
	Metadata      map[string]any `json:"metadata"`
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

type OpenAPITicketAssignedAgent struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Provider            string  `json:"provider"`
	RuntimeControlState string  `json:"runtime_control_state,omitempty"`
	RuntimePhase        *string `json:"runtime_phase,omitempty"`
}

type OpenAPIChatContext struct {
	ProjectID  string  `json:"project_id"`
	WorkflowID *string `json:"workflow_id,omitempty"`
	TicketID   *string `json:"ticket_id,omitempty"`
}

type OpenAPIChatStartRequest struct {
	Message    string             `json:"message"`
	Source     string             `json:"source"`
	ProviderID *string            `json:"provider_id,omitempty"`
	Context    OpenAPIChatContext `json:"context"`
	SessionID  *string            `json:"session_id,omitempty"`
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

type OpenAPIWorkflowRepositoryPrerequisite struct {
	Kind            string  `json:"kind"`
	RepoCount       int     `json:"repo_count"`
	PrimaryRepoID   *string `json:"primary_repo_id,omitempty"`
	PrimaryRepoName string  `json:"primary_repo_name,omitempty"`
	MirrorCount     int     `json:"mirror_count"`
	MirrorState     *string `json:"mirror_state,omitempty"`
	MirrorMachineID *string `json:"mirror_machine_id,omitempty"`
	MirrorLastError *string `json:"mirror_last_error,omitempty"`
	Action          string  `json:"action"`
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
	ID             string                        `json:"id"`
	Name           string                        `json:"name"`
	Description    string                        `json:"description"`
	Path           string                        `json:"path"`
	IsBuiltin      bool                          `json:"is_builtin"`
	IsEnabled      bool                          `json:"is_enabled"`
	CreatedBy      string                        `json:"created_by"`
	CreatedAt      string                        `json:"created_at"`
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

type OpenAPIWorkspaceSummaryResponse struct {
	Workspace     OpenAPIWorkspaceDashboardMetrics      `json:"workspace"`
	Organizations []OpenAPIWorkspaceOrganizationSummary `json:"organizations"`
}

type OpenAPIProjectsResponse struct {
	Projects []OpenAPIProject `json:"projects"`
}

type OpenAPIProjectResponse struct {
	Project OpenAPIProject `json:"project"`
}

type OpenAPIOrganizationSummaryResponse struct {
	Organization OpenAPIOrganizationDashboardMetrics `json:"organization"`
	Projects     []OpenAPIOrganizationProjectSummary `json:"projects"`
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

type OpenAPIMachineHealthRefreshResponse struct {
	Machine OpenAPIMachine `json:"machine"`
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

type OpenAPIAgentProviderModelCatalogResponse struct {
	AdapterModelOptions []OpenAPIAgentProviderModelCatalogEntry `json:"adapter_model_options"`
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

type OpenAPIAgentStepEntriesResponse struct {
	Entries []OpenAPIAgentStepEntry `json:"entries"`
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

type OpenAPITicketCommentsResponse struct {
	Comments []OpenAPITicketComment `json:"comments"`
}

type OpenAPITicketCommentRevisionsResponse struct {
	Revisions []OpenAPITicketCommentRevision `json:"revisions"`
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

type OpenAPIProjectRepoMirrorsResponse struct {
	Mirrors []OpenAPIProjectRepoMirror `json:"mirrors"`
}

type OpenAPIProjectRepoMirrorResponse struct {
	Mirror OpenAPIProjectRepoMirror `json:"mirror"`
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

type OpenAPIWorkflowRepositoryPrerequisiteResponse struct {
	Prerequisite OpenAPIWorkflowRepositoryPrerequisite `json:"prerequisite"`
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

type OpenAPIIssueConnectorFilters struct {
	Labels        []string `json:"labels"`
	ExcludeLabels []string `json:"exclude_labels"`
	States        []string `json:"states"`
	Authors       []string `json:"authors"`
}

type OpenAPIIssueConnectorConfig struct {
	Type                    string                       `json:"type"`
	BaseURL                 string                       `json:"base_url"`
	ProjectRef              string                       `json:"project_ref"`
	PollInterval            string                       `json:"poll_interval"`
	SyncDirection           string                       `json:"sync_direction"`
	Filters                 OpenAPIIssueConnectorFilters `json:"filters"`
	StatusMapping           map[string]string            `json:"status_mapping"`
	AutoWorkflow            string                       `json:"auto_workflow"`
	AuthTokenConfigured     bool                         `json:"auth_token_configured"`
	WebhookSecretConfigured bool                         `json:"webhook_secret_configured"`
}

type OpenAPIIssueConnectorStats struct {
	TotalSynced int `json:"total_synced"`
	Synced24h   int `json:"synced24h"`
	FailedCount int `json:"failed_count"`
}

type OpenAPIIssueConnector struct {
	ID         string                      `json:"id"`
	ProjectID  string                      `json:"project_id"`
	Type       string                      `json:"type"`
	Name       string                      `json:"name"`
	Status     string                      `json:"status"`
	Config     OpenAPIIssueConnectorConfig `json:"config"`
	LastSyncAt *string                     `json:"last_sync_at,omitempty"`
	LastError  string                      `json:"last_error"`
	Stats      OpenAPIIssueConnectorStats  `json:"stats"`
}

type OpenAPIIssueConnectorsResponse struct {
	Connectors []OpenAPIIssueConnector `json:"connectors"`
}

type OpenAPIIssueConnectorResponse struct {
	Connector OpenAPIIssueConnector `json:"connector"`
}

type OpenAPIIssueConnectorDeleteResponse struct {
	DeletedConnectorID string `json:"deleted_connector_id"`
}

type OpenAPIIssueConnectorTestResult struct {
	Healthy   bool   `json:"healthy"`
	CheckedAt string `json:"checked_at"`
	Message   string `json:"message"`
}

type OpenAPIIssueConnectorTestResponse struct {
	Result OpenAPIIssueConnectorTestResult `json:"result"`
}

type OpenAPIIssueConnectorSyncReport struct {
	ConnectorsScanned int `json:"connectors_scanned"`
	ConnectorsSynced  int `json:"connectors_synced"`
	ConnectorsFailed  int `json:"connectors_failed"`
	IssuesSynced      int `json:"issues_synced"`
}

type OpenAPIIssueConnectorSyncResponse struct {
	Connector OpenAPIIssueConnector           `json:"connector"`
	Report    OpenAPIIssueConnectorSyncReport `json:"report"`
}

type OpenAPIIssueConnectorStatsDetail struct {
	ConnectorID string                     `json:"connector_id"`
	Status      string                     `json:"status"`
	LastSyncAt  *string                    `json:"last_sync_at,omitempty"`
	LastError   string                     `json:"last_error"`
	Stats       OpenAPIIssueConnectorStats `json:"stats"`
}

type OpenAPIIssueConnectorStatsResponse struct {
	Stats OpenAPIIssueConnectorStatsDetail `json:"stats"`
}

type OpenAPIHarnessResponse struct {
	Harness OpenAPIHarnessDocument `json:"harness"`
}

type OpenAPISkillsResponse struct {
	Skills []OpenAPISkill `json:"skills"`
}

type OpenAPISkillDetailResponse struct {
	Skill   OpenAPISkill `json:"skill"`
	Content string       `json:"content"`
}

type OpenAPIDeleteSkillResponse struct {
	DeletedSkillID string `json:"deleted_skill_id"`
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

type OpenAPIActivateHRRecommendationRequest struct {
	RoleSlug              string `json:"role_slug"`
	CreateBootstrapTicket *bool  `json:"create_bootstrap_ticket,omitempty"`
}

type OpenAPIHRAdvisorActivationResponse struct {
	ProjectID       string                                `json:"project_id"`
	RoleSlug        string                                `json:"role_slug"`
	Agent           OpenAPIAgent                          `json:"agent"`
	Workflow        OpenAPIWorkflow                       `json:"workflow"`
	BootstrapTicket OpenAPIHRAdvisorBootstrapTicketResult `json:"bootstrap_ticket"`
}

type OpenAPIHRAdvisorBootstrapTicketResult struct {
	Requested bool           `json:"requested"`
	Status    string         `json:"status"`
	Message   string         `json:"message"`
	Ticket    *OpenAPITicket `json:"ticket,omitempty"`
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

type OpenAPIGitHubTokenProbe struct {
	State       string   `json:"state"`
	Configured  bool     `json:"configured"`
	Valid       bool     `json:"valid"`
	Permissions []string `json:"permissions"`
	RepoAccess  string   `json:"repo_access"`
	CheckedAt   *string  `json:"checked_at,omitempty"`
	LastError   string   `json:"last_error,omitempty"`
}

type OpenAPIGitHubOutboundCredential struct {
	Scope        string                  `json:"scope,omitempty"`
	Source       string                  `json:"source,omitempty"`
	TokenPreview string                  `json:"token_preview,omitempty"`
	Probe        OpenAPIGitHubTokenProbe `json:"probe"`
}

type OpenAPISecuritySettings struct {
	ProjectID     string                              `json:"project_id"`
	AgentTokens   OpenAPISecurityAgentTokens          `json:"agent_tokens"`
	GitHub        OpenAPIGitHubOutboundCredential     `json:"github"`
	Webhooks      OpenAPISecurityWebhooks             `json:"webhooks"`
	SecretHygiene OpenAPISecuritySecretHygiene        `json:"secret_hygiene"`
	Deferred      []OpenAPISecurityDeferredCapability `json:"deferred"`
}

type OpenAPISecuritySettingsResponse struct {
	Security OpenAPISecuritySettings `json:"security"`
}

type OpenAPITicketDetailResponse struct {
	AssignedAgent *OpenAPITicketAssignedAgent    `json:"assigned_agent,omitempty"`
	Ticket        OpenAPITicket                  `json:"ticket"`
	RepoScopes    []OpenAPITicketRepoScopeDetail `json:"repo_scopes"`
	Comments      []OpenAPITicketComment         `json:"comments"`
	Timeline      []OpenAPITicketTimelineItem    `json:"timeline"`
	Activity      []OpenAPIActivityEvent         `json:"activity"`
	HookHistory   []OpenAPIActivityEvent         `json:"hook_history"`
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
type OpenAPIMaterializeProjectRepoMirrorRequest projectRepoMirrorMaterializeRequest
type OpenAPIProjectRepoMirrorMachineRequest projectRepoMirrorMachineRequest
type OpenAPICreateTicketRepoScopeRequest catalogdomain.TicketRepoScopeInput
type OpenAPIUpdateTicketRepoScopeRequest ticketRepoScopePatchRequest
type OpenAPICreateAgentRequest catalogdomain.AgentInput
type OpenAPICreateWorkflowRequest rawCreateWorkflowRequest
type OpenAPIUpdateWorkflowRequest rawUpdateWorkflowRequest
type OpenAPIUpdateHarnessRequest rawUpdateHarnessRequest
type OpenAPIValidateHarnessRequest rawValidateHarnessRequest
type OpenAPICreateScheduledJobRequest rawCreateScheduledJobRequest
type OpenAPIUpdateScheduledJobRequest rawUpdateScheduledJobRequest
type OpenAPICreateIssueConnectorFilters struct {
	Labels        []string `json:"labels"`
	ExcludeLabels []string `json:"exclude_labels"`
	States        []string `json:"states"`
	Authors       []string `json:"authors"`
}

type OpenAPICreateIssueConnectorConfig struct {
	Type          string                             `json:"type"`
	BaseURL       string                             `json:"base_url"`
	AuthToken     string                             `json:"auth_token"`
	ProjectRef    string                             `json:"project_ref"`
	PollInterval  string                             `json:"poll_interval"`
	SyncDirection string                             `json:"sync_direction"`
	Filters       OpenAPICreateIssueConnectorFilters `json:"filters"`
	StatusMapping map[string]string                  `json:"status_mapping"`
	WebhookSecret string                             `json:"webhook_secret"`
	AutoWorkflow  string                             `json:"auto_workflow"`
}

type OpenAPICreateIssueConnectorRequest struct {
	Type   string                            `json:"type"`
	Name   string                            `json:"name"`
	Status string                            `json:"status"`
	Config OpenAPICreateIssueConnectorConfig `json:"config"`
}

type OpenAPIUpdateIssueConnectorConfig struct {
	BaseURL       *string                             `json:"base_url,omitempty"`
	AuthToken     *string                             `json:"auth_token,omitempty"`
	ProjectRef    *string                             `json:"project_ref,omitempty"`
	PollInterval  *string                             `json:"poll_interval,omitempty"`
	SyncDirection *string                             `json:"sync_direction,omitempty"`
	Filters       *OpenAPICreateIssueConnectorFilters `json:"filters,omitempty"`
	StatusMapping map[string]string                   `json:"status_mapping,omitempty"`
	WebhookSecret *string                             `json:"webhook_secret,omitempty"`
	AutoWorkflow  *string                             `json:"auto_workflow,omitempty"`
}

type OpenAPIUpdateIssueConnectorRequest struct {
	Name   *string                            `json:"name,omitempty"`
	Status *string                            `json:"status,omitempty"`
	Config *OpenAPIUpdateIssueConnectorConfig `json:"config,omitempty"`
}
type OpenAPIUpdateWorkflowSkillsRequest rawUpdateWorkflowSkillsRequest
type OpenAPISkillSyncRequest rawSkillSyncRequest
type OpenAPICreateSkillRequest rawCreateSkillRequest
type OpenAPIUpdateSkillRequest rawUpdateSkillRequest
type OpenAPIUpdateSkillBindingsRequest rawUpdateSkillBindingsRequest
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

var (
	openAPIOrganizationRequestDescriptions = map[string]string{
		"name":                      "Human-readable organization name.",
		"slug":                      "Stable URL-safe organization slug.",
		"default_agent_provider_id": "Optional default agent provider ID for the organization.",
	}
	openAPIChannelRequestDescriptions = map[string]string{
		"name":       "Human-readable notification channel name.",
		"type":       "Notification channel type, such as slack or webhook.",
		"config":     "Channel-specific configuration object submitted for this notification channel.",
		"is_enabled": "Whether the channel is enabled for delivery.",
	}
	openAPIMachineRequestDescriptions = map[string]string{
		"name":           "Human-readable machine name.",
		"host":           "Hostname or address used to reach the machine.",
		"port":           "SSH port used to connect to the machine.",
		"ssh_user":       "SSH username used for machine access.",
		"ssh_key_path":   "Path to the SSH private key used for machine access.",
		"description":    "Human-readable machine description.",
		"labels":         "Labels attached to the machine for operator reference.",
		"status":         "Machine lifecycle status value.",
		"workspace_root": "Filesystem root directory where ticket workspaces are created on the machine.",
		"mirror_root":    "Filesystem root directory where repository mirrors are stored on the machine.",
		"agent_cli_path": "Absolute path to the agent CLI executable on the machine.",
		"env_vars":       "Environment variable entries exported when work runs on the machine.",
	}
	openAPIProjectRequestDescriptions = map[string]string{
		"name":                      "Human-readable project name.",
		"slug":                      "Stable URL-safe project slug.",
		"description":               "Human-readable project description.",
		"status":                    "Current project lifecycle status name.",
		"default_workflow_id":       "Optional default workflow ID for newly created tickets in the project.",
		"default_agent_provider_id": "Optional default agent provider ID for the project.",
		"accessible_machine_ids":    "Machine IDs that the project is allowed to use.",
		"max_concurrent_agents":     "Maximum number of agents that may run concurrently in the project.",
	}
	openAPIProviderRequestDescriptions = map[string]string{
		"name":                  "Human-readable provider name.",
		"machine_id":            "Machine ID where this provider runs.",
		"adapter_type":          "Adapter type used to launch and communicate with the provider.",
		"cli_command":           "CLI command used to launch the provider.",
		"cli_args":              "Additional CLI arguments passed to the provider command.",
		"auth_config":           "Provider-specific authentication configuration object.",
		"model_name":            "Model name configured for the provider.",
		"model_temperature":     "Sampling temperature configured for the provider model.",
		"model_max_tokens":      "Maximum number of output tokens allowed for the provider model.",
		"max_parallel_runs":     "Maximum number of concurrent runs allowed for the provider.",
		"cost_per_input_token":  "Estimated USD cost per input token.",
		"cost_per_output_token": "Estimated USD cost per output token.",
	}
	openAPIRepoRequestDescriptions = map[string]string{
		"name":              "Human-readable repository name within the project.",
		"repository_url":    "Remote Git repository URL.",
		"default_branch":    "Default branch name used for mirrors and workspaces.",
		"workspace_dirname": "Directory name used for this repository inside a ticket workspace.",
		"is_primary":        "Whether this repository is the primary project repository.",
		"labels":            "Labels attached to the repository for workflow selection and filtering.",
	}
	openAPIRepoMirrorMaterializeDescriptions = map[string]string{
		"machine_id": "Machine ID where the repository mirror should be prepared or registered.",
		"local_path": "Absolute filesystem path where the mirror exists or should be created.",
		"mode":       "Mirror materialization mode, such as preparing a new mirror or registering an existing checkout.",
	}
	openAPIRepoMirrorMachineDescriptions = map[string]string{
		"machine_id": "Machine ID whose mirror should be verified or synchronized.",
	}
	openAPIAgentRequestDescriptions = map[string]string{
		"name":        "Human-readable agent name.",
		"provider_id": "Agent provider ID used to run the agent.",
	}
	openAPIWorkflowRequestDescriptions = map[string]string{
		"name":                  "Human-readable workflow name.",
		"type":                  "Workflow type, such as coding, test, doc, security, deploy, refine-harness, or custom.",
		"agent_id":              "Agent ID assigned to execute this workflow.",
		"pickup_status_ids":     "Ticket status IDs that allow the workflow to pick up tickets.",
		"finish_status_ids":     "Ticket status IDs that mark workflow completion.",
		"is_active":             "Whether the workflow is active and eligible to pick up work.",
		"max_concurrent":        "Maximum number of concurrent runs allowed for the workflow.",
		"timeout_minutes":       "Hard execution timeout for workflow runs, in minutes.",
		"stall_timeout_minutes": "Timeout for detecting stalled workflow runs, in minutes.",
		"max_retry_attempts":    "Maximum retry attempts before the workflow run fails permanently.",
		"harness_path":          "Repository path where the workflow harness file is stored.",
		"harness_content":       "Initial harness content written when creating the workflow.",
		"hooks":                 "Workflow hook configuration keyed by lifecycle phase.",
	}
	openAPIHarnessContentDescriptions = map[string]string{
		"content": "Harness content to write or validate.",
	}
	openAPIScheduledJobDescriptions = map[string]string{
		"name":            "Human-readable scheduled job name.",
		"cron_expression": "Cron expression that controls when the job triggers.",
		"workflow_id":     "Workflow ID executed when the scheduled job triggers.",
		"ticket_template": "Ticket template used to create a ticket for each scheduled run.",
		"is_enabled":      "Whether the scheduled job is enabled.",
	}
	openAPITicketRequestDescriptions = map[string]string{
		"title":            "Human-readable ticket title.",
		"description":      "Ticket description or problem statement.",
		"status_id":        "Optional ticket status ID to assign explicitly.",
		"priority":         "Ticket priority value.",
		"type":             "Ticket type value.",
		"workflow_id":      "Optional workflow ID that should handle the ticket.",
		"parent_ticket_id": "Optional parent ticket ID for hierarchical ticket relationships.",
		"external_ref":     "Optional external reference string associated with the ticket.",
		"created_by":       "Actor identifier recorded as the creator of the ticket.",
		"budget_usd":       "Optional budget limit for the ticket in USD.",
	}
	openAPITicketCommentRequestDescriptions = map[string]string{
		"body":       "Markdown body content for the ticket comment.",
		"created_by": "Actor identifier recorded as the creator of the comment.",
	}
	openAPITicketCommentPatchDescriptions = map[string]string{
		"body":        "Updated markdown body content for the ticket comment.",
		"edited_by":   "Actor identifier recorded as the editor of the comment.",
		"edit_reason": "Reason recorded for editing the comment.",
	}
	openAPIDependencyRequestDescriptions = map[string]string{
		"type":             "Dependency relationship type.",
		"target_ticket_id": "Target ticket ID referenced by the dependency.",
	}
	openAPIExternalLinkRequestDescriptions = map[string]string{
		"type":        "External link type.",
		"url":         "URL of the external resource.",
		"external_id": "External system identifier for the linked resource.",
		"title":       "Optional title for the external resource.",
		"status":      "Optional external status value.",
		"relation":    "Relationship between the ticket and the external resource.",
	}
	openAPIStageRequestDescriptions = map[string]string{
		"key":             "Stable machine-readable key for the ticket stage.",
		"name":            "Human-readable stage name.",
		"position":        "Zero-based display order of the stage.",
		"max_active_runs": "Maximum number of active runs allowed in this stage.",
		"description":     "Human-readable stage description.",
	}
	openAPIStatusRequestDescriptions = map[string]string{
		"stage_id":    "Optional stage ID that owns the status.",
		"name":        "Human-readable status name.",
		"color":       "Display color for the status.",
		"icon":        "Display icon identifier for the status.",
		"position":    "Zero-based display order of the status.",
		"is_default":  "Whether this status should become the default status.",
		"description": "Human-readable status description.",
	}
	openAPINotificationRuleDescriptions = map[string]string{
		"name":       "Human-readable notification rule name.",
		"event_type": "Event type that triggers the notification rule.",
		"channel_id": "Notification channel ID used by the rule.",
		"template":   "Notification template content rendered when the rule fires.",
		"filter":     "Optional filter configuration applied before the rule sends notifications.",
		"is_enabled": "Whether the notification rule is enabled.",
	}
	openAPIRepoScopeCreateDescriptions = map[string]string{
		"repo_id":          "Repository ID attached to the ticket scope.",
		"branch_name":      "Branch name associated with the scoped repository checkout.",
		"pull_request_url": "Pull request URL associated with the repository scope.",
		"pr_status":        "Pull request status associated with the repository scope.",
		"ci_status":        "Continuous integration status associated with the repository scope.",
		"is_primary_scope": "Whether this scope is the primary repository scope for the ticket.",
	}
	openAPIRepoScopePatchDescriptions = map[string]string{
		"branch_name":      "Branch name associated with the scoped repository checkout.",
		"pull_request_url": "Pull request URL associated with the repository scope.",
		"pr_status":        "Pull request status associated with the repository scope.",
		"ci_status":        "Continuous integration status associated with the repository scope.",
		"is_primary_scope": "Whether this scope is the primary repository scope for the ticket.",
	}
	openAPIHRAdvisorActivateDescriptions = map[string]string{
		"role_slug":               "HR advisor role slug to activate for the project.",
		"create_bootstrap_ticket": "Whether activation should create a bootstrap ticket immediately.",
	}
	openAPIIssueConnectorDescriptions = map[string]string{
		"type":                          "Connector implementation type. The shipped Settings surface currently targets the GitHub issue connector.",
		"name":                          "Human-readable connector name shown in Settings.",
		"status":                        "Runtime connector state. Supported values are active, paused, and error.",
		"config":                        "Connector runtime configuration.",
		"config.type":                   "Connector implementation type and must match the top-level type on create.",
		"config.base_url":               "Optional API base URL override for self-hosted or test endpoints.",
		"config.auth_token":             "Connector-scoped auth token. Leave blank to fall back to project GitHub credentials when supported.",
		"config.project_ref":            "External repository or project reference, for example owner/repo.",
		"config.poll_interval":          "Pull sync interval encoded as a Go duration string such as 5m.",
		"config.sync_direction":         "Sync policy. Supported values are bidirectional, pull_only, and push_only.",
		"config.filters":                "Optional connector-side issue filters.",
		"config.filters.labels":         "Labels required for an issue to be imported.",
		"config.filters.exclude_labels": "Labels that should exclude an issue from import.",
		"config.filters.states":         "Allowed upstream issue states.",
		"config.filters.authors":        "Allowed upstream issue authors.",
		"config.status_mapping":         "Map of upstream statuses to internal project ticket statuses.",
		"config.webhook_secret":         "Optional shared secret used to validate inbound webhook deliveries.",
		"config.auto_workflow":          "Optional workflow binding associated with imported tickets.",
	}
	openAPIChatRequestDescriptions = map[string]string{
		"message":             "User message content for the chat turn.",
		"provider_id":         "Optional provider ID used to run this chat session.",
		"session_id":          "Optional existing chat session ID to resume.",
		"source":              "Source identifier for the chat request, such as web or cli.",
		"context":             "Optional project, ticket, or workflow context attached to the chat turn.",
		"context.project_id":  "Project ID supplied to ground the chat request.",
		"context.ticket_id":   "Optional ticket ID supplied to ground the chat request.",
		"context.workflow_id": "Optional workflow ID supplied to ground the chat request.",
	}
	openAPISkillBindingDescriptions = map[string]string{
		"skills": "Skill names included in this workflow skill binding request.",
	}
	openAPISkillCreateDescriptions = map[string]string{
		"name":        "Project-unique skill directory name.",
		"content":     "Skill markdown content. Frontmatter is optional on input and will be normalized on write.",
		"description": "Optional description used when the input content does not declare one.",
		"created_by":  "Optional creator descriptor such as user:gary or agent:codex-01 via ASE-42.",
		"is_enabled":  "Whether the new skill should be enabled for runtime injection immediately.",
	}
	openAPISkillUpdateDescriptions = map[string]string{
		"content":     "Replacement skill markdown content. Frontmatter is optional on input and will be normalized on write.",
		"description": "Optional description override used when the input content does not declare one.",
	}
	openAPISkillSyncDescriptions = map[string]string{
		"workspace_root": "Workspace repository root that owns the agent skill directory.",
		"adapter_type":   "Agent adapter type used to derive the runtime skill directory.",
		"workflow_id":    "Optional workflow ID used to project only the currently bound enabled skills.",
	}
	openAPISkillBindingTargetDescriptions = map[string]string{
		"workflow_ids": "Workflow IDs that should bind or unbind this skill.",
		"harness_ids":  "Alias of workflow_ids kept for PRD terminology compatibility.",
	}
	openAPIRequestBodyDescriptions = map[string]map[string]string{
		"POST /api/v1/orgs":                                                           openAPIOrganizationRequestDescriptions,
		"PATCH /api/v1/orgs/{orgId}":                                                  openAPIOrganizationRequestDescriptions,
		"POST /api/v1/orgs/{orgId}/channels":                                          openAPIChannelRequestDescriptions,
		"PATCH /api/v1/channels/{channelId}":                                          openAPIChannelRequestDescriptions,
		"POST /api/v1/orgs/{orgId}/machines":                                          openAPIMachineRequestDescriptions,
		"PATCH /api/v1/machines/{machineId}":                                          openAPIMachineRequestDescriptions,
		"POST /api/v1/orgs/{orgId}/projects":                                          openAPIProjectRequestDescriptions,
		"PATCH /api/v1/projects/{projectId}":                                          openAPIProjectRequestDescriptions,
		"POST /api/v1/orgs/{orgId}/providers":                                         openAPIProviderRequestDescriptions,
		"PATCH /api/v1/providers/{providerId}":                                        openAPIProviderRequestDescriptions,
		"POST /api/v1/projects/{projectId}/repos":                                     openAPIRepoRequestDescriptions,
		"PATCH /api/v1/projects/{projectId}/repos/{repoId}":                           openAPIRepoRequestDescriptions,
		"POST /api/v1/projects/{projectId}/repos/{repoId}/mirrors":                    openAPIRepoMirrorMaterializeDescriptions,
		"POST /api/v1/projects/{projectId}/repos/{repoId}/mirrors/verify":             openAPIRepoMirrorMachineDescriptions,
		"POST /api/v1/projects/{projectId}/repos/{repoId}/mirrors/sync":               openAPIRepoMirrorMachineDescriptions,
		"POST /api/v1/projects/{projectId}/agents":                                    openAPIAgentRequestDescriptions,
		"POST /api/v1/projects/{projectId}/workflows":                                 openAPIWorkflowRequestDescriptions,
		"PATCH /api/v1/workflows/{workflowId}":                                        mergeRequestFieldDescriptions(openAPIWorkflowRequestDescriptions, map[string]string{"harness_content": ""}),
		"PUT /api/v1/workflows/{workflowId}/harness":                                  openAPIHarnessContentDescriptions,
		"POST /api/v1/harness/validate":                                               openAPIHarnessContentDescriptions,
		"POST /api/v1/projects/{projectId}/scheduled-jobs":                            openAPIScheduledJobDescriptions,
		"PATCH /api/v1/scheduled-jobs/{jobId}":                                        openAPIScheduledJobDescriptions,
		"POST /api/v1/projects/{projectId}/tickets":                                   openAPITicketRequestDescriptions,
		"PATCH /api/v1/tickets/{ticketId}":                                            openAPITicketRequestDescriptions,
		"POST /api/v1/tickets/{ticketId}/comments":                                    openAPITicketCommentRequestDescriptions,
		"PATCH /api/v1/tickets/{ticketId}/comments/{commentId}":                       openAPITicketCommentPatchDescriptions,
		"POST /api/v1/tickets/{ticketId}/dependencies":                                openAPIDependencyRequestDescriptions,
		"POST /api/v1/tickets/{ticketId}/external-links":                              openAPIExternalLinkRequestDescriptions,
		"POST /api/v1/projects/{projectId}/stages":                                    openAPIStageRequestDescriptions,
		"PATCH /api/v1/stages/{stageId}":                                              openAPIStageRequestDescriptions,
		"POST /api/v1/projects/{projectId}/statuses":                                  openAPIStatusRequestDescriptions,
		"PATCH /api/v1/statuses/{statusId}":                                           openAPIStatusRequestDescriptions,
		"POST /api/v1/projects/{projectId}/notification-rules":                        openAPINotificationRuleDescriptions,
		"PATCH /api/v1/notification-rules/{ruleId}":                                   openAPINotificationRuleDescriptions,
		"POST /api/v1/projects/{projectId}/connectors":                                openAPIIssueConnectorDescriptions,
		"PATCH /api/v1/connectors/{connectorId}":                                      openAPIIssueConnectorDescriptions,
		"POST /api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes":            openAPIRepoScopeCreateDescriptions,
		"PATCH /api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes/{scopeId}": openAPIRepoScopePatchDescriptions,
		"POST /api/v1/projects/{projectId}/hr-advisor/activate":                       openAPIHRAdvisorActivateDescriptions,
		"POST /api/v1/chat":                                                           openAPIChatRequestDescriptions,
		"POST /api/v1/projects/{projectId}/skills":                                    openAPISkillCreateDescriptions,
		"POST /api/v1/projects/{projectId}/skills/refresh":                            openAPISkillSyncDescriptions,
		"POST /api/v1/projects/{projectId}/skills/harvest":                            openAPISkillSyncDescriptions,
		"PUT /api/v1/skills/{skillId}":                                                openAPISkillUpdateDescriptions,
		"POST /api/v1/skills/{skillId}/bind":                                          openAPISkillBindingTargetDescriptions,
		"POST /api/v1/skills/{skillId}/unbind":                                        openAPISkillBindingTargetDescriptions,
		"POST /api/v1/workflows/{workflowId}/skills/bind":                             openAPISkillBindingDescriptions,
		"POST /api/v1/workflows/{workflowId}/skills/unbind":                           openAPISkillBindingDescriptions,
	}
)

func mergeRequestFieldDescriptions(maps ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, item := range maps {
		for key, value := range item {
			if strings.TrimSpace(value) == "" {
				continue
			}
			merged[key] = value
		}
	}
	return merged
}

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
			{Name: "connectors"},
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
	if err := builder.addIssueConnectorOperations(); err != nil {
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
	applyOpenAPIRequestBodyDescriptions(doc)

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
	workspaceSummaryGet, err := b.jsonOperation(
		"getWorkspaceSummary",
		"Get workspace dashboard summary metrics",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIWorkspaceSummaryResponse{},
		nil,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/workspace/summary", http.MethodGet, workspaceSummaryGet)

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

	orgSummaryGet, err := b.jsonOperation(
		"getOrganizationSummary",
		"Get organization dashboard summary metrics",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIOrganizationSummaryResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	orgSummaryGet.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	b.doc.AddOperation("/api/v1/orgs/{orgId}/summary", http.MethodGet, orgSummaryGet)

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

	providerModelOptionsGet, err := b.jsonOperation(
		"listProviderModelOptions",
		"List builtin provider model options",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentProviderModelCatalogResponse{},
		nil,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/provider-model-options", http.MethodGet, providerModelOptionsGet)

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

	machineHealthRefresh, err := b.jsonOperation(
		"refreshMachineHealth",
		"Refresh multi-level machine health checks",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIMachineHealthRefreshResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	machineHealthRefresh.AddParameter(uuidPathParameter("machineId", "Machine ID."))
	b.doc.AddOperation("/api/v1/machines/{machineId}/refresh-health", http.MethodPost, machineHealthRefresh)

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

	projectRepoMirrorsGet, err := b.jsonOperation(
		"listProjectRepoMirrors",
		"List project repository mirrors",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectRepoMirrorsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectRepoMirrorsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectRepoMirrorsGet.AddParameter(uuidPathParameter("repoId", "Repository ID."))
	projectRepoMirrorsGet.AddParameter(uuidQueryParameter("machine_id", "Optional machine filter."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/repos/{repoId}/mirrors", http.MethodGet, projectRepoMirrorsGet)

	projectRepoMirrorsPost, err := b.jsonOperation(
		"materializeProjectRepoMirror",
		"Register or prepare a project repository mirror",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPIProjectRepoMirrorResponse{},
		OpenAPIMaterializeProjectRepoMirrorRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectRepoMirrorsPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectRepoMirrorsPost.AddParameter(uuidPathParameter("repoId", "Repository ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/repos/{repoId}/mirrors", http.MethodPost, projectRepoMirrorsPost)

	projectRepoMirrorVerifyPost, err := b.jsonOperation(
		"verifyProjectRepoMirror",
		"Verify a project repository mirror",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectRepoMirrorResponse{},
		OpenAPIProjectRepoMirrorMachineRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectRepoMirrorVerifyPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectRepoMirrorVerifyPost.AddParameter(uuidPathParameter("repoId", "Repository ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/repos/{repoId}/mirrors/verify", http.MethodPost, projectRepoMirrorVerifyPost)

	projectRepoMirrorSyncPost, err := b.jsonOperation(
		"syncProjectRepoMirror",
		"Sync a project repository mirror",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectRepoMirrorResponse{},
		OpenAPIProjectRepoMirrorMachineRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectRepoMirrorSyncPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectRepoMirrorSyncPost.AddParameter(uuidPathParameter("repoId", "Repository ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/repos/{repoId}/mirrors/sync", http.MethodPost, projectRepoMirrorSyncPost)

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

	agentStepsGet, err := b.jsonOperation(
		"listAgentSteps",
		"List agent step entries",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentStepEntriesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentStepsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	agentStepsGet.AddParameter(uuidPathParameter("agentId", "Agent ID."))
	agentStepsGet.AddParameter(uuidQueryParameter("ticket_id", "Filter steps by ticket ID."))
	agentStepsGet.AddParameter(intQueryParameter("limit", "Limit the number of returned step entries."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/agents/{agentId}/steps", http.MethodGet, agentStepsGet)

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

	hrAdvisorActivate, err := b.jsonOperation(
		"activateHRRecommendation",
		"Activate an HR advisor recommendation",
		[]string{"hr-advisor"},
		http.StatusCreated,
		OpenAPIHRAdvisorActivationResponse{},
		OpenAPIActivateHRRecommendationRequest{},
		http.StatusBadRequest,
		http.StatusConflict,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	hrAdvisorActivate.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/hr-advisor/activate", http.MethodPost, hrAdvisorActivate)

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

	workflowPrerequisiteGet, err := b.jsonOperation(
		"getWorkflowRepositoryPrerequisite",
		"Get workflow repository prerequisite",
		[]string{"workflow"},
		http.StatusOK,
		OpenAPIWorkflowRepositoryPrerequisiteResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	workflowPrerequisiteGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/workflows/prerequisite", http.MethodGet, workflowPrerequisiteGet)

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

	skillsCreate, err := b.jsonOperation(
		"createSkill",
		"Create a skill in the project library",
		[]string{"skills"},
		http.StatusCreated,
		OpenAPISkillDetailResponse{},
		OpenAPICreateSkillRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	skillsCreate.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/skills", http.MethodPost, skillsCreate)

	refreshSkills, err := b.jsonOperation(
		"refreshSkills",
		"Refresh a workspace skill directory from the project skill library",
		[]string{"skills"},
		http.StatusOK,
		skillSyncResponse{},
		OpenAPISkillSyncRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	refreshSkills.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/skills/refresh", http.MethodPost, refreshSkills)

	harvestSkills, err := b.jsonOperation(
		"harvestSkills",
		"Harvest workspace-authored skills back into the project skill library",
		[]string{"skills"},
		http.StatusOK,
		skillSyncResponse{},
		OpenAPISkillSyncRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	harvestSkills.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/skills/harvest", http.MethodPost, harvestSkills)

	skillGet, err := b.jsonOperation(
		"getSkill",
		"Get skill detail",
		[]string{"skills"},
		http.StatusOK,
		OpenAPISkillDetailResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	skillGet.AddParameter(uuidPathParameter("skillId", "Skill ID."))
	b.doc.AddOperation("/api/v1/skills/{skillId}", http.MethodGet, skillGet)

	skillUpdate, err := b.jsonOperation(
		"updateSkill",
		"Update skill content",
		[]string{"skills"},
		http.StatusOK,
		OpenAPISkillDetailResponse{},
		OpenAPIUpdateSkillRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	skillUpdate.AddParameter(uuidPathParameter("skillId", "Skill ID."))
	b.doc.AddOperation("/api/v1/skills/{skillId}", http.MethodPut, skillUpdate)

	skillDelete, err := b.jsonOperation(
		"deleteSkill",
		"Delete a skill and unbind it from all workflows",
		[]string{"skills"},
		http.StatusOK,
		OpenAPIDeleteSkillResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	skillDelete.AddParameter(uuidPathParameter("skillId", "Skill ID."))
	b.doc.AddOperation("/api/v1/skills/{skillId}", http.MethodDelete, skillDelete)

	enableSkill, err := b.jsonOperation(
		"enableSkill",
		"Enable a skill for runtime injection",
		[]string{"skills"},
		http.StatusOK,
		OpenAPISkillDetailResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	enableSkill.AddParameter(uuidPathParameter("skillId", "Skill ID."))
	b.doc.AddOperation("/api/v1/skills/{skillId}/enable", http.MethodPost, enableSkill)

	disableSkill, err := b.jsonOperation(
		"disableSkill",
		"Disable a skill without deleting its files",
		[]string{"skills"},
		http.StatusOK,
		OpenAPISkillDetailResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	disableSkill.AddParameter(uuidPathParameter("skillId", "Skill ID."))
	b.doc.AddOperation("/api/v1/skills/{skillId}/disable", http.MethodPost, disableSkill)

	bindSkill, err := b.jsonOperation(
		"bindSkill",
		"Bind a skill to one or more workflow harnesses",
		[]string{"skills"},
		http.StatusOK,
		OpenAPISkillDetailResponse{},
		OpenAPIUpdateSkillBindingsRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	bindSkill.AddParameter(uuidPathParameter("skillId", "Skill ID."))
	b.doc.AddOperation("/api/v1/skills/{skillId}/bind", http.MethodPost, bindSkill)

	unbindSkill, err := b.jsonOperation(
		"unbindSkill",
		"Unbind a skill from one or more workflow harnesses",
		[]string{"skills"},
		http.StatusOK,
		OpenAPISkillDetailResponse{},
		OpenAPIUpdateSkillBindingsRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	unbindSkill.AddParameter(uuidPathParameter("skillId", "Skill ID."))
	b.doc.AddOperation("/api/v1/skills/{skillId}/unbind", http.MethodPost, unbindSkill)

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

func (b openAPISpecBuilder) addIssueConnectorOperations() error {
	connectorsGet, err := b.jsonOperation(
		"listIssueConnectors",
		"List project issue connectors",
		[]string{"connectors"},
		http.StatusOK,
		OpenAPIIssueConnectorsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	connectorsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/connectors", http.MethodGet, connectorsGet)

	connectorsPost, err := b.jsonOperation(
		"createIssueConnector",
		"Create a project issue connector",
		[]string{"connectors"},
		http.StatusCreated,
		OpenAPIIssueConnectorResponse{},
		OpenAPICreateIssueConnectorRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	connectorsPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/connectors", http.MethodPost, connectorsPost)

	connectorPatch, err := b.jsonOperation(
		"updateIssueConnector",
		"Update an issue connector",
		[]string{"connectors"},
		http.StatusOK,
		OpenAPIIssueConnectorResponse{},
		OpenAPIUpdateIssueConnectorRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	connectorPatch.AddParameter(uuidPathParameter("connectorId", "Connector ID."))
	b.doc.AddOperation("/api/v1/connectors/{connectorId}", http.MethodPatch, connectorPatch)

	connectorDelete, err := b.jsonOperation(
		"deleteIssueConnector",
		"Delete an issue connector",
		[]string{"connectors"},
		http.StatusOK,
		OpenAPIIssueConnectorDeleteResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	connectorDelete.AddParameter(uuidPathParameter("connectorId", "Connector ID."))
	b.doc.AddOperation("/api/v1/connectors/{connectorId}", http.MethodDelete, connectorDelete)

	connectorTest, err := b.jsonOperation(
		"testIssueConnector",
		"Run a connector health check",
		[]string{"connectors"},
		http.StatusOK,
		OpenAPIIssueConnectorTestResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	connectorTest.AddParameter(uuidPathParameter("connectorId", "Connector ID."))
	b.doc.AddOperation("/api/v1/connectors/{connectorId}/test", http.MethodPost, connectorTest)

	connectorSync, err := b.jsonOperation(
		"syncIssueConnector",
		"Trigger a manual connector pull sync",
		[]string{"connectors"},
		http.StatusOK,
		OpenAPIIssueConnectorSyncResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	connectorSync.AddParameter(uuidPathParameter("connectorId", "Connector ID."))
	b.doc.AddOperation("/api/v1/connectors/{connectorId}/sync", http.MethodPost, connectorSync)

	connectorStats, err := b.jsonOperation(
		"getIssueConnectorStats",
		"Get connector runtime stats and last sync details",
		[]string{"connectors"},
		http.StatusOK,
		OpenAPIIssueConnectorStatsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	connectorStats.AddParameter(uuidPathParameter("connectorId", "Connector ID."))
	b.doc.AddOperation("/api/v1/connectors/{connectorId}/stats", http.MethodGet, connectorStats)

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

	commentsGet, err := b.jsonOperation(
		"listTicketComments",
		"List ticket comments",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketCommentsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	commentsGet.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}/comments", http.MethodGet, commentsGet)

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

	commentRevisionsGet, err := b.jsonOperation(
		"listTicketCommentRevisions",
		"List ticket comment revisions",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketCommentRevisionsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	commentRevisionsGet.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	commentRevisionsGet.AddParameter(uuidPathParameter("commentId", "Comment ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}/comments/{commentId}/revisions", http.MethodGet, commentRevisionsGet)

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
		WithDescription("Ephemeral chat session ID.").
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

	agentStepStream, err := b.streamOperation(
		"streamAgentSteps",
		"Stream agent step entries",
		[]string{"streams"},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentStepStream.AddParameter(uuidPathParameter("projectId", "Project ID."))
	agentStepStream.AddParameter(uuidPathParameter("agentId", "Agent ID."))
	agentStepStream.AddParameter(uuidQueryParameter("ticket_id", "Filter streamed steps by ticket ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/agents/{agentId}/steps/stream", http.MethodGet, agentStepStream)

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

func applyOpenAPIRequestBodyDescriptions(doc *openapi3.T) {
	if doc == nil || doc.Paths == nil {
		return
	}
	for path, pathItem := range doc.Paths.Map() {
		if pathItem == nil {
			continue
		}
		for method, operation := range pathItem.Operations() {
			if operation == nil || operation.RequestBody == nil || operation.RequestBody.Value == nil {
				continue
			}
			mediaType := operation.RequestBody.Value.Content.Get("application/json")
			if mediaType == nil || mediaType.Schema == nil {
				continue
			}
			descriptions, ok := openAPIRequestBodyDescriptions[strings.ToUpper(method)+" "+path]
			if !ok {
				continue
			}
			applyRequestFieldDescriptions(mediaType.Schema, "", descriptions)
		}
	}
}

func applyRequestFieldDescriptions(schemaRef *openapi3.SchemaRef, prefix string, descriptions map[string]string) {
	if schemaRef == nil || schemaRef.Value == nil {
		return
	}
	schema := schemaRef.Value
	for name, property := range schema.Properties {
		if property == nil || property.Value == nil {
			continue
		}
		clonedProperty, err := cloneSchemaRef(property)
		if err != nil {
			continue
		}
		schema.Properties[name] = clonedProperty
		fieldPath := name
		if prefix != "" {
			fieldPath = prefix + "." + name
		}
		if description, ok := descriptions[fieldPath]; ok {
			clonedProperty.Value.Description = description
		}
		applyRequestFieldDescriptions(clonedProperty, fieldPath, descriptions)
	}
}

func cloneSchemaRef(schemaRef *openapi3.SchemaRef) (*openapi3.SchemaRef, error) {
	if schemaRef == nil {
		return nil, nil
	}
	payload, err := json.Marshal(schemaRef)
	if err != nil {
		return nil, err
	}
	var cloned openapi3.SchemaRef
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return nil, err
	}
	return &cloned, nil
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
