package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	githubrepodomain "github.com/BetterAndBetterII/openase/internal/domain/githubrepo"
	notificationdomain "github.com/BetterAndBetterII/openase/internal/domain/notification"
	"github.com/BetterAndBetterII/openase/internal/domain/pricing"
	workflowdomain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
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
	ID                             string   `json:"id"`
	OrganizationID                 string   `json:"organization_id"`
	Name                           string   `json:"name"`
	Slug                           string   `json:"slug"`
	Description                    string   `json:"description"`
	Status                         string   `json:"status"`
	DefaultAgentProviderID         *string  `json:"default_agent_provider_id,omitempty"`
	AccessibleMachineIDs           []string `json:"accessible_machine_ids,omitempty"`
	MaxConcurrentAgents            int      `json:"max_concurrent_agents"`
	AgentRunSummaryPrompt          *string  `json:"agent_run_summary_prompt,omitempty"`
	EffectiveAgentRunSummaryPrompt string   `json:"effective_agent_run_summary_prompt"`
	AgentRunSummaryPromptSource    string   `json:"agent_run_summary_prompt_source"`
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
	ID                    string                          `json:"id"`
	OrganizationID        string                          `json:"organization_id"`
	Name                  string                          `json:"name"`
	Host                  string                          `json:"host"`
	Port                  int                             `json:"port"`
	ReachabilityMode      string                          `json:"reachability_mode"`
	ExecutionMode         string                          `json:"execution_mode"`
	ExecutionCapabilities []string                        `json:"execution_capabilities,omitempty"`
	SSHHelperEnabled      bool                            `json:"ssh_helper_enabled"`
	SSHHelperRequired     bool                            `json:"ssh_helper_required"`
	ConnectionMode        string                          `json:"connection_mode"`
	TransportCapabilities []string                        `json:"transport_capabilities,omitempty"`
	SSHUser               *string                         `json:"ssh_user,omitempty"`
	SSHKeyPath            *string                         `json:"ssh_key_path,omitempty"`
	AdvertisedEndpoint    *string                         `json:"advertised_endpoint,omitempty"`
	DaemonStatus          OpenAPIMachineDaemonStatus      `json:"daemon_status"`
	DetectedOS            string                          `json:"detected_os"`
	DetectedArch          string                          `json:"detected_arch"`
	DetectionStatus       string                          `json:"detection_status"`
	DetectionMessage      string                          `json:"detection_message"`
	ChannelCredential     OpenAPIMachineChannelCredential `json:"channel_credential"`
	Description           string                          `json:"description"`
	Labels                []string                        `json:"labels,omitempty"`
	Status                string                          `json:"status"`
	WorkspaceRoot         *string                         `json:"workspace_root,omitempty"`
	AgentCLIPath          *string                         `json:"agent_cli_path,omitempty"`
	EnvVars               []string                        `json:"env_vars,omitempty"`
	LastHeartbeatAt       *string                         `json:"last_heartbeat_at,omitempty"`
	Resources             map[string]any                  `json:"resources"`
}

type OpenAPIMachineDaemonStatus struct {
	Registered       bool    `json:"registered"`
	LastRegisteredAt *string `json:"last_registered_at,omitempty"`
	CurrentSessionID *string `json:"current_session_id,omitempty"`
	SessionState     string  `json:"session_state"`
}

type OpenAPIMachineChannelCredential struct {
	Kind          string  `json:"kind"`
	TokenID       *string `json:"token_id,omitempty"`
	CertificateID *string `json:"certificate_id,omitempty"`
}

type OpenAPIMachineProbe struct {
	CheckedAt        string         `json:"checked_at"`
	Transport        string         `json:"transport"`
	Output           string         `json:"output"`
	Resources        map[string]any `json:"resources"`
	DetectedOS       string         `json:"detected_os"`
	DetectedArch     string         `json:"detected_arch"`
	DetectionStatus  string         `json:"detection_status"`
	DetectionMessage string         `json:"detection_message"`
}

type OpenAPIProjectRepo struct {
	ID               string   `json:"id"`
	ProjectID        string   `json:"project_id"`
	Name             string   `json:"name"`
	RepositoryURL    string   `json:"repository_url"`
	DefaultBranch    string   `json:"default_branch"`
	WorkspaceDirname string   `json:"workspace_dirname"`
	Labels           []string `json:"labels,omitempty"`
}

type OpenAPIAgentProvider struct {
	ID                    string                             `json:"id"`
	OrganizationID        string                             `json:"organization_id"`
	MachineID             string                             `json:"machine_id"`
	MachineName           string                             `json:"machine_name"`
	MachineHost           string                             `json:"machine_host"`
	MachineStatus         string                             `json:"machine_status"`
	MachineSSHUser        *string                            `json:"machine_ssh_user,omitempty"`
	MachineWorkspaceRoot  *string                            `json:"machine_workspace_root,omitempty"`
	Name                  string                             `json:"name"`
	AdapterType           string                             `json:"adapter_type"`
	PermissionProfile     string                             `json:"permission_profile"`
	AvailabilityState     string                             `json:"availability_state"`
	Available             bool                               `json:"available"`
	AvailabilityCheckedAt *string                            `json:"availability_checked_at,omitempty"`
	AvailabilityReason    *string                            `json:"availability_reason,omitempty"`
	Capabilities          OpenAPIAgentProviderCapabilities   `json:"capabilities"`
	CliCommand            string                             `json:"cli_command"`
	CliArgs               []string                           `json:"cli_args"`
	AuthConfig            map[string]any                     `json:"auth_config"`
	CLIRateLimit          *OpenAPIAgentProviderCLIRateLimit  `json:"cli_rate_limit,omitempty"`
	CLIRateLimitUpdatedAt *string                            `json:"cli_rate_limit_updated_at,omitempty"`
	ModelName             string                             `json:"model_name"`
	ModelTemperature      float64                            `json:"model_temperature"`
	ModelMaxTokens        int                                `json:"model_max_tokens"`
	MaxParallelRuns       int                                `json:"max_parallel_runs"`
	CostPerInputToken     float64                            `json:"cost_per_input_token"`
	CostPerOutputToken    float64                            `json:"cost_per_output_token"`
	PricingConfig         pricing.ProviderModelPricingConfig `json:"pricing_config"`
}

type OpenAPIAgentProviderCapabilities struct {
	EphemeralChat OpenAPIAgentProviderCapability `json:"ephemeral_chat"`
	HarnessAI     OpenAPIAgentProviderCapability `json:"harness_ai"`
	SkillAI       OpenAPIAgentProviderCapability `json:"skill_ai"`
}

type OpenAPIAgentProviderCapability struct {
	State  string  `json:"state"`
	Reason *string `json:"reason,omitempty"`
}

type OpenAPIAgentProviderCLIRateLimit struct {
	Provider   string                                   `json:"provider"`
	ClaudeCode *OpenAPIAgentProviderClaudeCodeRateLimit `json:"claude_code,omitempty"`
	Codex      *OpenAPIAgentProviderCodexRateLimit      `json:"codex,omitempty"`
	Gemini     *OpenAPIAgentProviderGeminiRateLimit     `json:"gemini,omitempty"`
	Raw        map[string]any                           `json:"raw,omitempty"`
}

type OpenAPIAgentProviderClaudeCodeRateLimit struct {
	Status                string   `json:"status,omitempty"`
	RateLimitType         string   `json:"rate_limit_type,omitempty"`
	ResetsAt              *string  `json:"resets_at,omitempty"`
	Utilization           *float64 `json:"utilization,omitempty"`
	SurpassedThreshold    *float64 `json:"surpassed_threshold,omitempty"`
	OverageStatus         string   `json:"overage_status,omitempty"`
	OverageDisabledReason string   `json:"overage_disabled_reason,omitempty"`
	IsUsingOverage        *bool    `json:"is_using_overage,omitempty"`
}

type OpenAPIAgentProviderCodexRateLimit struct {
	LimitID   string                                    `json:"limit_id,omitempty"`
	LimitName string                                    `json:"limit_name,omitempty"`
	Primary   *OpenAPIAgentProviderCodexRateLimitWindow `json:"primary,omitempty"`
	Secondary *OpenAPIAgentProviderCodexRateLimitWindow `json:"secondary,omitempty"`
	PlanType  string                                    `json:"plan_type,omitempty"`
}

type OpenAPIAgentProviderCodexRateLimitWindow struct {
	UsedPercent   *float64 `json:"used_percent,omitempty"`
	WindowMinutes int64    `json:"window_minutes,omitempty"`
	ResetsAt      *string  `json:"resets_at,omitempty"`
}

type OpenAPIAgentProviderGeminiRateLimit struct {
	AuthType  string                                      `json:"auth_type,omitempty"`
	Remaining *int64                                      `json:"remaining,omitempty"`
	Limit     *int64                                      `json:"limit,omitempty"`
	ResetTime *string                                     `json:"reset_time,omitempty"`
	Buckets   []OpenAPIAgentProviderGeminiRateLimitBucket `json:"buckets,omitempty"`
}

type OpenAPIAgentProviderGeminiRateLimitBucket struct {
	ModelID           string   `json:"model_id,omitempty"`
	TokenType         string   `json:"token_type,omitempty"`
	RemainingAmount   string   `json:"remaining_amount,omitempty"`
	RemainingFraction *float64 `json:"remaining_fraction,omitempty"`
	ResetTime         *string  `json:"reset_time,omitempty"`
}

type OpenAPIAgentProviderModelOption struct {
	ID            string                              `json:"id"`
	Label         string                              `json:"label"`
	Description   string                              `json:"description"`
	Recommended   bool                                `json:"recommended"`
	Preview       bool                                `json:"preview"`
	PricingConfig *pricing.ProviderModelPricingConfig `json:"pricing_config,omitempty"`
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
	ID                string   `json:"id"`
	AgentID           string   `json:"agent_id"`
	WorkflowID        string   `json:"workflow_id"`
	WorkflowVersionID *string  `json:"workflow_version_id,omitempty"`
	TicketID          string   `json:"ticket_id"`
	ProviderID        string   `json:"provider_id"`
	SkillVersionIDs   []string `json:"skill_version_ids"`
	Status            string   `json:"status"`
	SessionID         string   `json:"session_id"`
	RuntimeStartedAt  *string  `json:"runtime_started_at,omitempty"`
	TerminalAt        *string  `json:"terminal_at,omitempty"`
	LastError         string   `json:"last_error"`
	LastHeartbeatAt   *string  `json:"last_heartbeat_at,omitempty"`
	CreatedAt         string   `json:"created_at"`
}

type OpenAPIAgentRuntimeControlResponse struct {
	Agent       OpenAPIAgent `json:"agent"`
	Transition  string       `json:"transition"`
	RequestedAt string       `json:"requested_at"`
}

type OpenAPIWorkflowTicketReference workflowdomain.WorkflowTicketReference
type OpenAPIWorkflowScheduledJobReference workflowdomain.WorkflowScheduledJobReference
type OpenAPIWorkflowAgentRunReference workflowdomain.WorkflowAgentRunReference
type OpenAPIWorkflowReplaceableReferences workflowdomain.WorkflowReplaceableReferences
type OpenAPIWorkflowBlockingReferences workflowdomain.WorkflowBlockingReferences
type OpenAPIWorkflowImpactSummary workflowdomain.WorkflowImpactSummary
type OpenAPIWorkflowImpactAnalysis workflowdomain.WorkflowImpactAnalysis

type OpenAPIWorkflowImpactResponse struct {
	Impact OpenAPIWorkflowImpactAnalysis `json:"impact"`
}

type OpenAPIWorkflowReplaceReferencesResult struct {
	WorkflowID            string                                 `json:"workflow_id"`
	ReplacementWorkflowID string                                 `json:"replacement_workflow_id"`
	TicketCount           int                                    `json:"ticket_count"`
	ScheduledJobCount     int                                    `json:"scheduled_job_count"`
	Tickets               []OpenAPIWorkflowTicketReference       `json:"tickets"`
	ScheduledJobs         []OpenAPIWorkflowScheduledJobReference `json:"scheduled_jobs"`
}

type OpenAPIWorkflowReplaceReferencesResponse struct {
	Result OpenAPIWorkflowReplaceReferencesResult `json:"result"`
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

type OpenAPIProjectConversationContext struct {
	ProjectID string `json:"project_id"`
}

type OpenAPIProjectConversationCreateRequest struct {
	Source     string                            `json:"source"`
	ProviderID string                            `json:"provider_id"`
	Context    OpenAPIProjectConversationContext `json:"context"`
}

type OpenAPIProjectConversationTurnRequest struct {
	Message string                               `json:"message"`
	Focus   *OpenAPIProjectConversationTurnFocus `json:"focus,omitempty"`
}

type OpenAPIProjectConversationTurnFocus struct {
	Kind                 string                                         `json:"kind"`
	WorkflowID           *string                                        `json:"workflow_id,omitempty"`
	WorkflowName         *string                                        `json:"workflow_name,omitempty"`
	WorkflowType         *string                                        `json:"workflow_type,omitempty"`
	HarnessPath          *string                                        `json:"harness_path,omitempty"`
	IsActive             *bool                                          `json:"is_active,omitempty"`
	SelectedArea         *string                                        `json:"selected_area,omitempty"`
	HasDirtyDraft        *bool                                          `json:"has_dirty_draft,omitempty"`
	SkillID              *string                                        `json:"skill_id,omitempty"`
	SkillName            *string                                        `json:"skill_name,omitempty"`
	SelectedFilePath     *string                                        `json:"selected_file_path,omitempty"`
	BoundWorkflowNames   []string                                       `json:"bound_workflow_names,omitempty"`
	TicketID             *string                                        `json:"ticket_id,omitempty"`
	TicketIdentifier     *string                                        `json:"ticket_identifier,omitempty"`
	TicketTitle          *string                                        `json:"ticket_title,omitempty"`
	TicketDescription    *string                                        `json:"ticket_description,omitempty"`
	TicketStatus         *string                                        `json:"ticket_status,omitempty"`
	TicketPriority       *string                                        `json:"ticket_priority,omitempty"`
	TicketAttemptCount   *int                                           `json:"ticket_attempt_count,omitempty"`
	TicketRetryPaused    *bool                                          `json:"ticket_retry_paused,omitempty"`
	TicketPauseReason    *string                                        `json:"ticket_pause_reason,omitempty"`
	TicketDependencies   []OpenAPIProjectConversationTicketDependency   `json:"ticket_dependencies,omitempty"`
	TicketRepoScopes     []OpenAPIProjectConversationTicketRepoScope    `json:"ticket_repo_scopes,omitempty"`
	TicketRecentActivity []OpenAPIProjectConversationTicketActivity     `json:"ticket_recent_activity,omitempty"`
	TicketHookHistory    []OpenAPIProjectConversationTicketHook         `json:"ticket_hook_history,omitempty"`
	TicketAssignedAgent  *OpenAPIProjectConversationTicketAssignedAgent `json:"ticket_assigned_agent,omitempty"`
	TicketCurrentRun     *OpenAPIProjectConversationTicketRun           `json:"ticket_current_run,omitempty"`
	TicketTargetMachine  *OpenAPIProjectConversationTicketTargetMachine `json:"ticket_target_machine,omitempty"`
	MachineID            *string                                        `json:"machine_id,omitempty"`
	MachineName          *string                                        `json:"machine_name,omitempty"`
	MachineHost          *string                                        `json:"machine_host,omitempty"`
	MachineStatus        *string                                        `json:"machine_status,omitempty"`
	HealthSummary        *string                                        `json:"health_summary,omitempty"`
}

type OpenAPIProjectConversationTicketDependency struct {
	Identifier *string `json:"identifier,omitempty"`
	Title      *string `json:"title,omitempty"`
	Relation   *string `json:"relation,omitempty"`
	Status     *string `json:"status,omitempty"`
}

type OpenAPIProjectConversationTicketRepoScope struct {
	RepoID         *string `json:"repo_id,omitempty"`
	RepoName       *string `json:"repo_name,omitempty"`
	BranchName     *string `json:"branch_name,omitempty"`
	PullRequestURL *string `json:"pull_request_url,omitempty"`
}

type OpenAPIProjectConversationTicketActivity struct {
	EventType *string `json:"event_type,omitempty"`
	Message   *string `json:"message,omitempty"`
	CreatedAt *string `json:"created_at,omitempty"`
}

type OpenAPIProjectConversationTicketHook struct {
	HookName  *string `json:"hook_name,omitempty"`
	Status    *string `json:"status,omitempty"`
	Output    *string `json:"output,omitempty"`
	Timestamp *string `json:"timestamp,omitempty"`
}

type OpenAPIProjectConversationTicketAssignedAgent struct {
	ID                  *string `json:"id,omitempty"`
	Name                *string `json:"name,omitempty"`
	Provider            *string `json:"provider,omitempty"`
	RuntimeControlState *string `json:"runtime_control_state,omitempty"`
	RuntimePhase        *string `json:"runtime_phase,omitempty"`
}

type OpenAPIProjectConversationTicketRun struct {
	ID                 *string `json:"id,omitempty"`
	AttemptNumber      *int    `json:"attempt_number,omitempty"`
	Status             *string `json:"status,omitempty"`
	CurrentStepStatus  *string `json:"current_step_status,omitempty"`
	CurrentStepSummary *string `json:"current_step_summary,omitempty"`
	LastError          *string `json:"last_error,omitempty"`
}

type OpenAPIProjectConversationTicketTargetMachine struct {
	ID   *string `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
	Host *string `json:"host,omitempty"`
}

type OpenAPIProjectConversationInterruptResponseRequest struct {
	Decision *string        `json:"decision,omitempty"`
	Answer   map[string]any `json:"answer,omitempty"`
}

type OpenAPIProjectConversation struct {
	ID             string `json:"id"`
	ProjectID      string `json:"project_id"`
	UserID         string `json:"user_id"`
	Source         string `json:"source"`
	ProviderID     string `json:"provider_id"`
	Status         string `json:"status"`
	RollingSummary string `json:"rolling_summary"`
	LastActivityAt string `json:"last_activity_at"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type OpenAPIProjectConversationResponse struct {
	Conversation OpenAPIProjectConversation `json:"conversation"`
}

type OpenAPIProjectConversationListResponse struct {
	Conversations []OpenAPIProjectConversation `json:"conversations"`
}

type OpenAPIProjectConversationEntry struct {
	ID             string         `json:"id"`
	ConversationID string         `json:"conversation_id"`
	TurnID         *string        `json:"turn_id,omitempty"`
	Seq            int            `json:"seq"`
	Kind           string         `json:"kind"`
	Payload        map[string]any `json:"payload"`
	CreatedAt      string         `json:"created_at"`
}

type OpenAPIProjectConversationEntriesResponse struct {
	Entries []OpenAPIProjectConversationEntry `json:"entries"`
}

type OpenAPIProjectConversationWorkspaceDiffFile struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

type OpenAPIProjectConversationWorkspaceDiffRepo struct {
	Name         string                                        `json:"name"`
	Path         string                                        `json:"path"`
	Branch       string                                        `json:"branch"`
	Dirty        bool                                          `json:"dirty"`
	FilesChanged int                                           `json:"files_changed"`
	Added        int                                           `json:"added"`
	Removed      int                                           `json:"removed"`
	Files        []OpenAPIProjectConversationWorkspaceDiffFile `json:"files"`
}

type OpenAPIProjectConversationWorkspaceDiff struct {
	ConversationID string                                        `json:"conversation_id"`
	WorkspacePath  string                                        `json:"workspace_path"`
	Dirty          bool                                          `json:"dirty"`
	ReposChanged   int                                           `json:"repos_changed"`
	FilesChanged   int                                           `json:"files_changed"`
	Added          int                                           `json:"added"`
	Removed        int                                           `json:"removed"`
	Repos          []OpenAPIProjectConversationWorkspaceDiffRepo `json:"repos"`
}

type OpenAPIProjectConversationWorkspaceDiffResponse struct {
	WorkspaceDiff OpenAPIProjectConversationWorkspaceDiff `json:"workspace_diff"`
}

type OpenAPIProjectConversationTurn struct {
	ID        string `json:"id"`
	TurnIndex int    `json:"turn_index"`
	Status    string `json:"status"`
}

type OpenAPIProjectConversationTurnResponse struct {
	Turn OpenAPIProjectConversationTurn `json:"turn"`
}

type OpenAPIProjectConversationInterruptResponse struct {
	ID                string         `json:"id"`
	ConversationID    string         `json:"conversation_id"`
	TurnID            string         `json:"turn_id"`
	ProviderRequestID string         `json:"provider_request_id"`
	Kind              string         `json:"kind"`
	Payload           map[string]any `json:"payload"`
	Status            string         `json:"status"`
	Decision          *string        `json:"decision,omitempty"`
	ResolvedAt        *string        `json:"resolved_at,omitempty"`
}

type OpenAPIProjectConversationInterruptEnvelope struct {
	Interrupt OpenAPIProjectConversationInterruptResponse `json:"interrupt"`
}

type OpenAPIProjectConversationActionProposalExecutionResponse struct {
	ResultEntry OpenAPIProjectConversationEntry `json:"result_entry"`
	Results     []map[string]any                `json:"results"`
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

type OpenAPITicketRun struct {
	ID                 string                             `json:"id"`
	AttemptNumber      int                                `json:"attempt_number"`
	AgentID            string                             `json:"agent_id"`
	AgentName          string                             `json:"agent_name"`
	Provider           string                             `json:"provider"`
	Status             string                             `json:"status"`
	CurrentStepStatus  *string                            `json:"current_step_status,omitempty"`
	CurrentStepSummary *string                            `json:"current_step_summary,omitempty"`
	CreatedAt          string                             `json:"created_at"`
	RuntimeStartedAt   *string                            `json:"runtime_started_at,omitempty"`
	LastHeartbeatAt    *string                            `json:"last_heartbeat_at,omitempty"`
	TerminalAt         *string                            `json:"terminal_at,omitempty"`
	CompletedAt        *string                            `json:"completed_at,omitempty"`
	LastError          *string                            `json:"last_error,omitempty"`
	CompletionSummary  *OpenAPITicketRunCompletionSummary `json:"completion_summary,omitempty"`
}

type OpenAPITicketRunCompletionSummary struct {
	Status      string         `json:"status"`
	Markdown    *string        `json:"markdown,omitempty"`
	JSON        map[string]any `json:"json,omitempty"`
	GeneratedAt *string        `json:"generated_at,omitempty"`
	Error       *string        `json:"error,omitempty"`
}

type OpenAPITicketRunTraceEntry struct {
	ID         string         `json:"id"`
	AgentRunID string         `json:"agent_run_id"`
	Sequence   int64          `json:"sequence"`
	Provider   string         `json:"provider"`
	Kind       string         `json:"kind"`
	Stream     string         `json:"stream"`
	Output     string         `json:"output"`
	Payload    map[string]any `json:"payload"`
	CreatedAt  string         `json:"created_at"`
}

type OpenAPITicketRunStepEntry struct {
	ID                 string  `json:"id"`
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

type OpenAPIProjectUpdateComment struct {
	ID           string  `json:"id"`
	ThreadID     string  `json:"thread_id"`
	BodyMarkdown string  `json:"body_markdown"`
	CreatedBy    string  `json:"created_by"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	EditedAt     *string `json:"edited_at,omitempty"`
	EditCount    int     `json:"edit_count"`
	LastEditedBy *string `json:"last_edited_by,omitempty"`
	IsDeleted    bool    `json:"is_deleted"`
	DeletedAt    *string `json:"deleted_at,omitempty"`
	DeletedBy    *string `json:"deleted_by,omitempty"`
}

type OpenAPIProjectUpdateThread struct {
	ID             string                        `json:"id"`
	ProjectID      string                        `json:"project_id"`
	Status         string                        `json:"status"`
	Title          string                        `json:"title"`
	BodyMarkdown   string                        `json:"body_markdown"`
	CreatedBy      string                        `json:"created_by"`
	CreatedAt      string                        `json:"created_at"`
	UpdatedAt      string                        `json:"updated_at"`
	EditedAt       *string                       `json:"edited_at,omitempty"`
	EditCount      int                           `json:"edit_count"`
	LastEditedBy   *string                       `json:"last_edited_by,omitempty"`
	IsDeleted      bool                          `json:"is_deleted"`
	DeletedAt      *string                       `json:"deleted_at,omitempty"`
	DeletedBy      *string                       `json:"deleted_by,omitempty"`
	LastActivityAt string                        `json:"last_activity_at"`
	CommentCount   int                           `json:"comment_count"`
	Comments       []OpenAPIProjectUpdateComment `json:"comments"`
}

type OpenAPIProjectUpdateThreadRevision struct {
	ID             string  `json:"id"`
	ThreadID       string  `json:"thread_id"`
	RevisionNumber int     `json:"revision_number"`
	Status         string  `json:"status"`
	Title          string  `json:"title"`
	BodyMarkdown   string  `json:"body_markdown"`
	EditedBy       string  `json:"edited_by"`
	EditedAt       string  `json:"edited_at"`
	EditReason     *string `json:"edit_reason,omitempty"`
}

type OpenAPIProjectUpdateCommentRevision struct {
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
	Archived          bool                        `json:"archived"`
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
	ID                  string              `json:"id"`
	TicketID            string              `json:"ticket_id"`
	RepoID              string              `json:"repo_id"`
	Repo                *OpenAPIProjectRepo `json:"repo,omitempty"`
	BranchName          string              `json:"branch_name"`
	DefaultBranch       string              `json:"default_branch"`
	EffectiveBranchName string              `json:"effective_branch_name"`
	BranchSource        string              `json:"branch_source"`
	PullRequestURL      *string             `json:"pull_request_url,omitempty"`
}

type OpenAPITicketRepoScope struct {
	ID             string  `json:"id"`
	TicketID       string  `json:"ticket_id"`
	RepoID         string  `json:"repo_id"`
	BranchName     string  `json:"branch_name"`
	PullRequestURL *string `json:"pull_request_url,omitempty"`
}

type OpenAPITicketAssignedAgent struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Provider            string  `json:"provider"`
	RuntimeControlState string  `json:"runtime_control_state,omitempty"`
	RuntimePhase        *string `json:"runtime_phase,omitempty"`
}

type OpenAPITicketPickupDiagnosisReason struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type OpenAPITicketPickupDiagnosisWorkflow struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	IsActive          bool   `json:"is_active"`
	PickupStatusMatch bool   `json:"pickup_status_match"`
}

type OpenAPITicketPickupDiagnosisAgent struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	RuntimeControlState string `json:"runtime_control_state"`
}

type OpenAPITicketPickupDiagnosisProvider struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	MachineID          string  `json:"machine_id"`
	MachineName        string  `json:"machine_name"`
	MachineStatus      string  `json:"machine_status"`
	AvailabilityState  string  `json:"availability_state"`
	AvailabilityReason *string `json:"availability_reason,omitempty"`
}

type OpenAPITicketPickupDiagnosisRetry struct {
	AttemptCount int     `json:"attempt_count"`
	RetryPaused  bool    `json:"retry_paused"`
	PauseReason  string  `json:"pause_reason,omitempty"`
	NextRetryAt  *string `json:"next_retry_at,omitempty"`
}

type OpenAPITicketPickupDiagnosisCapacityBucket struct {
	Limited    bool `json:"limited"`
	ActiveRuns int  `json:"active_runs"`
	Capacity   int  `json:"capacity"`
}

type OpenAPITicketPickupDiagnosisStatusCapacity struct {
	Limited    bool `json:"limited"`
	ActiveRuns int  `json:"active_runs"`
	Capacity   *int `json:"capacity"`
}

type OpenAPITicketPickupDiagnosisBlockedTicket struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	StatusID   string `json:"status_id"`
	StatusName string `json:"status_name"`
}

type OpenAPITicketPickupDiagnosisCapacity struct {
	Workflow OpenAPITicketPickupDiagnosisCapacityBucket `json:"workflow"`
	Project  OpenAPITicketPickupDiagnosisCapacityBucket `json:"project"`
	Provider OpenAPITicketPickupDiagnosisCapacityBucket `json:"provider"`
	Status   OpenAPITicketPickupDiagnosisStatusCapacity `json:"status"`
}

type OpenAPITicketPickupDiagnosis struct {
	State                string                                      `json:"state"`
	PrimaryReasonCode    string                                      `json:"primary_reason_code"`
	PrimaryReasonMessage string                                      `json:"primary_reason_message"`
	NextActionHint       string                                      `json:"next_action_hint,omitempty"`
	Reasons              []OpenAPITicketPickupDiagnosisReason        `json:"reasons"`
	Workflow             *OpenAPITicketPickupDiagnosisWorkflow       `json:"workflow,omitempty"`
	Agent                *OpenAPITicketPickupDiagnosisAgent          `json:"agent,omitempty"`
	Provider             *OpenAPITicketPickupDiagnosisProvider       `json:"provider,omitempty"`
	Retry                OpenAPITicketPickupDiagnosisRetry           `json:"retry"`
	Capacity             OpenAPITicketPickupDiagnosisCapacity        `json:"capacity"`
	BlockedBy            []OpenAPITicketPickupDiagnosisBlockedTicket `json:"blocked_by"`
}

type OpenAPIChatContext struct {
	ProjectID    string  `json:"project_id"`
	WorkflowID   *string `json:"workflow_id,omitempty"`
	TicketID     *string `json:"ticket_id,omitempty"`
	HarnessDraft *string `json:"harness_draft,omitempty"`
}

type OpenAPIChatStartRequest struct {
	Message    string             `json:"message"`
	Source     string             `json:"source"`
	ProviderID *string            `json:"provider_id,omitempty"`
	Context    OpenAPIChatContext `json:"context"`
	SessionID  *string            `json:"session_id,omitempty"`
}

type OpenAPITicketStatus struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	Name          string `json:"name"`
	Stage         string `json:"stage"`
	Color         string `json:"color"`
	Icon          string `json:"icon"`
	Position      int    `json:"position"`
	ActiveRuns    int    `json:"active_runs"`
	MaxActiveRuns *int   `json:"max_active_runs,omitempty"`
	IsDefault     bool   `json:"is_default"`
	Description   string `json:"description"`
}

type OpenAPIWorkflow struct {
	ID                  string                        `json:"id"`
	ProjectID           string                        `json:"project_id"`
	AgentID             *string                       `json:"agent_id,omitempty"`
	Name                string                        `json:"name"`
	Type                string                        `json:"type"`
	WorkflowFamily      string                        `json:"workflow_family"`
	Classification      OpenAPIWorkflowClassification `json:"workflow_classification"`
	HarnessPath         string                        `json:"harness_path"`
	HarnessContent      *string                       `json:"harness_content,omitempty"`
	Hooks               map[string]any                `json:"hooks"`
	MaxConcurrent       int                           `json:"max_concurrent"`
	MaxRetryAttempts    int                           `json:"max_retry_attempts"`
	TimeoutMinutes      int                           `json:"timeout_minutes"`
	StallTimeoutMinutes int                           `json:"stall_timeout_minutes"`
	Version             int                           `json:"version"`
	IsActive            bool                          `json:"is_active"`
	PickupStatusIDs     []string                      `json:"pickup_status_ids"`
	FinishStatusIDs     []string                      `json:"finish_status_ids"`
}

type OpenAPIWorkflowClassification struct {
	Family     string   `json:"family"`
	Confidence float64  `json:"confidence"`
	Reasons    []string `json:"reasons"`
}

type OpenAPIHarnessDocument struct {
	WorkflowID string `json:"workflow_id"`
	Path       string `json:"path"`
	Content    string `json:"content"`
	Version    int    `json:"version"`
}

type OpenAPIVersionSummary struct {
	ID        string `json:"id"`
	Version   int    `json:"version"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
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
	CurrentVersion int                           `json:"current_version"`
	IsBuiltin      bool                          `json:"is_builtin"`
	IsEnabled      bool                          `json:"is_enabled"`
	CreatedBy      string                        `json:"created_by"`
	CreatedAt      string                        `json:"created_at"`
	BoundWorkflows []OpenAPISkillWorkflowBinding `json:"bound_workflows"`
}

type OpenAPIBuiltinRole struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	WorkflowType    string `json:"workflow_type"`
	WorkflowFamily  string `json:"workflow_family"`
	Summary         string `json:"summary"`
	HarnessPath     string `json:"harness_path"`
	Content         string `json:"content"`
	WorkflowContent string `json:"workflow_content"`
}

type OpenAPIHRAdvisorSummary struct {
	OpenTickets            int      `json:"open_tickets"`
	CodingTickets          int      `json:"coding_tickets"`
	FailingTickets         int      `json:"failing_tickets"`
	BlockedTickets         int      `json:"blocked_tickets"`
	ActiveAgents           int      `json:"active_agents"`
	WorkflowCount          int      `json:"workflow_count"`
	RecentActivityCount    int      `json:"recent_activity_count"`
	ActiveWorkflowFamilies []string `json:"active_workflow_families"`
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
	RoleSlug                string   `json:"role_slug"`
	RoleName                string   `json:"role_name"`
	WorkflowType            string   `json:"workflow_type"`
	WorkflowFamily          string   `json:"workflow_family"`
	Summary                 string   `json:"summary"`
	HarnessPath             string   `json:"harness_path"`
	Priority                string   `json:"priority"`
	Reason                  string   `json:"reason"`
	Evidence                []string `json:"evidence"`
	SuggestedHeadcount      int      `json:"suggested_headcount"`
	SuggestedWorkflowName   string   `json:"suggested_workflow_name"`
	SuggestedWorkflowType   string   `json:"suggested_workflow_type"`
	SuggestedWorkflowFamily string   `json:"suggested_workflow_family"`
	ActivationReady         bool     `json:"activation_ready"`
	ActiveWorkflowName      *string  `json:"active_workflow_name,omitempty"`
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

type OpenAPIOrganizationTokenUsageDay struct {
	Date              string `json:"date"`
	InputTokens       int64  `json:"input_tokens"`
	OutputTokens      int64  `json:"output_tokens"`
	CachedInputTokens int64  `json:"cached_input_tokens"`
	ReasoningTokens   int64  `json:"reasoning_tokens"`
	TotalTokens       int64  `json:"total_tokens"`
	FinalizedRunCount int    `json:"finalized_run_count"`
}

type OpenAPIOrganizationTokenUsagePeakDay struct {
	Date        string `json:"date"`
	TotalTokens int64  `json:"total_tokens"`
}

type OpenAPIOrganizationTokenUsageSummary struct {
	TotalTokens    int64                                 `json:"total_tokens"`
	AvgDailyTokens int64                                 `json:"avg_daily_tokens"`
	PeakDay        *OpenAPIOrganizationTokenUsagePeakDay `json:"peak_day,omitempty"`
}

type OpenAPIOrganizationTokenUsageResponse struct {
	Days    []OpenAPIOrganizationTokenUsageDay   `json:"days"`
	Summary OpenAPIOrganizationTokenUsageSummary `json:"summary"`
}

type OpenAPIProjectTokenUsageDay = OpenAPIOrganizationTokenUsageDay
type OpenAPIProjectTokenUsagePeakDay = OpenAPIOrganizationTokenUsagePeakDay
type OpenAPIProjectTokenUsageSummary = OpenAPIOrganizationTokenUsageSummary
type OpenAPIProjectTokenUsageResponse = OpenAPIOrganizationTokenUsageResponse

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

type OpenAPITicketRunsResponse struct {
	Runs []OpenAPITicketRun `json:"runs"`
}

type OpenAPITicketRunResponse struct {
	Run          OpenAPITicketRun             `json:"run"`
	TraceEntries []OpenAPITicketRunTraceEntry `json:"trace_entries"`
	StepEntries  []OpenAPITicketRunStepEntry  `json:"step_entries"`
}

type OpenAPIActivityEventsResponse struct {
	Events []OpenAPIActivityEvent `json:"events"`
}

type OpenAPITicketStatusesResponse struct {
	Statuses []OpenAPITicketStatus `json:"statuses"`
}

type OpenAPITicketStatusResponse struct {
	Status OpenAPITicketStatus `json:"status"`
}

type OpenAPITicketStatusDeleteResponse struct {
	DeletedStatusID     string `json:"deleted_status_id"`
	ReplacementStatusID string `json:"replacement_status_id"`
}

type OpenAPITicketsResponse struct {
	Tickets []OpenAPITicket `json:"tickets"`
}

type OpenAPIArchivedTicketsResponse struct {
	Tickets []OpenAPITicket `json:"tickets"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	PerPage int             `json:"per_page"`
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

type OpenAPITicketWorkspaceResetResponse struct {
	Reset bool `json:"reset"`
}

type OpenAPITicketCommentRevisionsResponse struct {
	Revisions []OpenAPITicketCommentRevision `json:"revisions"`
}

type OpenAPITicketCommentDeleteResponse struct {
	DeletedCommentID string `json:"deleted_comment_id"`
}

type OpenAPIProjectUpdateThreadsResponse struct {
	Threads []OpenAPIProjectUpdateThread `json:"threads"`
}

type OpenAPIProjectUpdateThreadResponse struct {
	Thread OpenAPIProjectUpdateThread `json:"thread"`
}

type OpenAPIProjectUpdateThreadRevisionsResponse struct {
	Revisions []OpenAPIProjectUpdateThreadRevision `json:"revisions"`
}

type OpenAPIProjectUpdateThreadDeleteResponse struct {
	DeletedThreadID string `json:"deleted_thread_id"`
}

type OpenAPIProjectUpdateCommentResponse struct {
	Comment OpenAPIProjectUpdateComment `json:"comment"`
}

type OpenAPIProjectUpdateCommentRevisionsResponse struct {
	Revisions []OpenAPIProjectUpdateCommentRevision `json:"revisions"`
}

type OpenAPIProjectUpdateCommentDeleteResponse struct {
	DeletedCommentID string `json:"deleted_comment_id"`
}

type OpenAPIProjectReposResponse struct {
	Repos []OpenAPIProjectRepo `json:"repos"`
}

type OpenAPIProjectRepoResponse struct {
	Repo OpenAPIProjectRepo `json:"repo"`
}

type OpenAPIGitHubRepositoryNamespace struct {
	Login string `json:"login"`
	Kind  string `json:"kind"`
}

type OpenAPIGitHubRepository struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Owner         string `json:"owner"`
	DefaultBranch string `json:"default_branch"`
	Visibility    string `json:"visibility"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
}

type OpenAPIGitHubRepositoryNamespacesResponse struct {
	Namespaces []OpenAPIGitHubRepositoryNamespace `json:"namespaces"`
}

type OpenAPIGitHubRepositoriesResponse struct {
	Repositories []OpenAPIGitHubRepository `json:"repositories"`
	NextCursor   string                    `json:"next_cursor,omitempty"`
}

type OpenAPIGitHubRepositoryResponse struct {
	Repository OpenAPIGitHubRepository `json:"repository"`
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

type OpenAPIWorkflowHistoryResponse struct {
	History []OpenAPIVersionSummary `json:"history"`
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

type OpenAPISkillDetailResponse struct {
	Skill   OpenAPISkill            `json:"skill"`
	Content string                  `json:"content"`
	Files   []OpenAPISkillFile      `json:"files,omitempty"`
	History []OpenAPIVersionSummary `json:"history"`
}

type OpenAPISkillFile struct {
	Path          string `json:"path"`
	FileKind      string `json:"file_kind"`
	MediaType     string `json:"media_type"`
	Encoding      string `json:"encoding"`
	IsExecutable  bool   `json:"is_executable"`
	SizeBytes     int64  `json:"size_bytes"`
	SHA256        string `json:"sha256"`
	Content       string `json:"content,omitempty"`
	ContentBase64 string `json:"content_base64,omitempty"`
}

type OpenAPISkillHistoryResponse struct {
	History []OpenAPIVersionSummary `json:"history"`
}

type OpenAPISkillFilesResponse struct {
	Files []OpenAPISkillFile `json:"files"`
}

type OpenAPIDeleteSkillResponse struct {
	DeletedSkillID string `json:"deleted_skill_id"`
}

type OpenAPISkillRefinementRequest rawSkillRefinementRequest

type OpenAPIRolesResponse struct {
	Roles []OpenAPIBuiltinRole `json:"roles"`
}

type OpenAPIRoleResponse struct {
	Role OpenAPIBuiltinRole `json:"role"`
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
	ConnectorEndpoint string `json:"connector_endpoint"`
}

type OpenAPISecuritySecretHygiene struct {
	NotificationChannelConfigsRedacted bool `json:"notification_channel_configs_redacted"`
}

type OpenAPISecurityApprovalPolicies struct {
	Status     string `json:"status"`
	RulesCount int    `json:"rules_count"`
	Summary    string `json:"summary"`
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

type OpenAPIGitHubCredentialSlot struct {
	Scope        string                  `json:"scope,omitempty"`
	Configured   bool                    `json:"configured"`
	Source       string                  `json:"source,omitempty"`
	TokenPreview string                  `json:"token_preview,omitempty"`
	Probe        OpenAPIGitHubTokenProbe `json:"probe"`
}

type OpenAPIGitHubOutboundCredential struct {
	Effective       OpenAPIGitHubCredentialSlot `json:"effective"`
	Organization    OpenAPIGitHubCredentialSlot `json:"organization"`
	ProjectOverride OpenAPIGitHubCredentialSlot `json:"project_override"`
}

type OpenAPISecuritySettings struct {
	ProjectID        string                              `json:"project_id"`
	AgentTokens      OpenAPISecurityAgentTokens          `json:"agent_tokens"`
	GitHub           OpenAPIGitHubOutboundCredential     `json:"github"`
	Webhooks         OpenAPISecurityWebhooks             `json:"webhooks"`
	SecretHygiene    OpenAPISecuritySecretHygiene        `json:"secret_hygiene"`
	ApprovalPolicies OpenAPISecurityApprovalPolicies     `json:"approval_policies"`
	Deferred         []OpenAPISecurityDeferredCapability `json:"deferred"`
}

type OpenAPISecuritySettingsResponse struct {
	Security OpenAPISecuritySettings `json:"security"`
}

type OpenAPIAuthSessionUser struct {
	ID           string `json:"id"`
	PrimaryEmail string `json:"primary_email"`
	DisplayName  string `json:"display_name"`
	AvatarURL    string `json:"avatar_url,omitempty"`
}

type OpenAPIAuthSessionResponse struct {
	AuthMode      string                  `json:"auth_mode"`
	Authenticated bool                    `json:"authenticated"`
	IssuerURL     string                  `json:"issuer_url,omitempty"`
	User          *OpenAPIAuthSessionUser `json:"user,omitempty"`
	CSRFToken     string                  `json:"csrf_token,omitempty"`
	Roles         []string                `json:"roles,omitempty"`
	Permissions   []string                `json:"permissions,omitempty"`
}

type OpenAPIHumanScope struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type OpenAPIHumanGroupMembership struct {
	GroupKey  string `json:"group_key"`
	GroupName string `json:"group_name"`
	Issuer    string `json:"issuer"`
}

type OpenAPIAuthPermissionsResponse struct {
	User        OpenAPIAuthSessionUser        `json:"user"`
	Scope       OpenAPIHumanScope             `json:"scope"`
	Roles       []string                      `json:"roles"`
	Permissions []string                      `json:"permissions"`
	Groups      []OpenAPIHumanGroupMembership `json:"groups"`
}

type OpenAPIRoleBinding struct {
	ID          string  `json:"id"`
	ScopeKind   string  `json:"scope_kind"`
	ScopeID     string  `json:"scope_id"`
	SubjectKind string  `json:"subject_kind"`
	SubjectKey  string  `json:"subject_key"`
	RoleKey     string  `json:"role_key"`
	GrantedBy   string  `json:"granted_by"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type OpenAPIRoleBindingsResponse struct {
	RoleBindings []OpenAPIRoleBinding `json:"role_bindings"`
}

type OpenAPIRoleBindingResponse struct {
	RoleBinding OpenAPIRoleBinding `json:"role_binding"`
}

type OpenAPICreateRoleBindingRequest struct {
	SubjectKind string  `json:"subject_kind"`
	SubjectKey  string  `json:"subject_key"`
	RoleKey     string  `json:"role_key"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

type OpenAPITicketDetailResponse struct {
	AssignedAgent   *OpenAPITicketAssignedAgent    `json:"assigned_agent,omitempty"`
	PickupDiagnosis OpenAPITicketPickupDiagnosis   `json:"pickup_diagnosis"`
	Ticket          OpenAPITicket                  `json:"ticket"`
	RepoScopes      []OpenAPITicketRepoScopeDetail `json:"repo_scopes"`
	Comments        []OpenAPITicketComment         `json:"comments"`
	Timeline        []OpenAPITicketTimelineItem    `json:"timeline"`
	Activity        []OpenAPIActivityEvent         `json:"activity"`
	HookHistory     []OpenAPIActivityEvent         `json:"hook_history"`
}

type OpenAPICreateOrganizationRequest catalogdomain.OrganizationInput
type OpenAPIUpdateOrganizationRequest organizationPatchRequest
type OpenAPICreateAgentProviderRequest catalogdomain.AgentProviderInput
type OpenAPIUpdateAgentProviderRequest agentProviderPatchRequest
type OpenAPIUpdateAgentRequest agentPatchRequest
type OpenAPICreateProjectRequest catalogdomain.ProjectInput
type OpenAPIUpdateProjectRequest projectPatchRequest
type OpenAPICreateMachineRequest catalogdomain.MachineInput
type OpenAPIUpdateMachineRequest machinePatchRequest
type OpenAPICreateProjectRepoRequest catalogdomain.ProjectRepoInput
type OpenAPIUpdateProjectRepoRequest projectRepoPatchRequest
type OpenAPICreateGitHubRepositoryRequest githubrepodomain.CreateRepositoryRequest
type OpenAPISaveGitHubOutboundCredentialRequest rawSaveGitHubOutboundCredentialRequest
type OpenAPIGitHubCredentialScopeRequest rawGitHubCredentialScopeRequest
type OpenAPICreateTicketRepoScopeRequest catalogdomain.TicketRepoScopeInput
type OpenAPIUpdateTicketRepoScopeRequest ticketRepoScopePatchRequest
type OpenAPICreateAgentRequest catalogdomain.AgentInput
type OpenAPICreateWorkflowRequest rawCreateWorkflowRequest
type OpenAPIUpdateWorkflowRequest rawUpdateWorkflowRequest
type OpenAPIRetireWorkflowRequest rawRetireWorkflowRequest
type OpenAPIReplaceWorkflowReferencesRequest rawReplaceWorkflowReferencesRequest
type OpenAPIUpdateHarnessRequest rawUpdateHarnessRequest
type OpenAPIValidateHarnessRequest rawValidateHarnessRequest
type OpenAPICreateScheduledJobRequest rawCreateScheduledJobRequest
type OpenAPIUpdateScheduledJobRequest rawUpdateScheduledJobRequest
type OpenAPIUpdateWorkflowSkillsRequest rawUpdateWorkflowSkillsRequest
type OpenAPISkillSyncRequest rawSkillSyncRequest
type OpenAPICreateSkillRequest rawCreateSkillRequest
type OpenAPISkillBundleFileRequest rawSkillBundleFileRequest
type OpenAPIUpdateSkillRequest rawUpdateSkillRequest
type OpenAPIUpdateSkillBindingsRequest rawUpdateSkillBindingsRequest
type OpenAPICreateTicketRequest rawCreateTicketRequest
type OpenAPIUpdateTicketRequest rawUpdateTicketRequest
type OpenAPICreateTicketCommentRequest rawCreateTicketCommentRequest
type OpenAPIUpdateTicketCommentRequest rawUpdateTicketCommentRequest
type OpenAPICreateProjectUpdateThreadRequest rawCreateProjectUpdateThreadRequest
type OpenAPIUpdateProjectUpdateThreadRequest rawUpdateProjectUpdateThreadRequest
type OpenAPICreateProjectUpdateCommentRequest rawCreateProjectUpdateCommentRequest
type OpenAPIUpdateProjectUpdateCommentRequest rawUpdateProjectUpdateCommentRequest
type OpenAPIAddTicketDependencyRequest rawAddDependencyRequest
type OpenAPICreateTicketExternalLinkRequest rawAddExternalLinkRequest
type OpenAPICreateTicketStatusRequest struct {
	Name          string `json:"name"`
	Stage         string `json:"stage"`
	Color         string `json:"color"`
	Icon          string `json:"icon"`
	Position      *int   `json:"position"`
	MaxActiveRuns *int   `json:"max_active_runs"`
	IsDefault     bool   `json:"is_default"`
	Description   string `json:"description"`
}

type OpenAPIUpdateTicketStatusRequest struct {
	Name          *string `json:"name"`
	Stage         *string `json:"stage"`
	Color         *string `json:"color"`
	Icon          *string `json:"icon"`
	Position      *int    `json:"position"`
	MaxActiveRuns *int    `json:"max_active_runs"`
	IsDefault     *bool   `json:"is_default"`
	Description   *string `json:"description"`
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
		"name":                              "Human-readable machine name.",
		"host":                              "Hostname or address used to reach the machine.",
		"port":                              "Transport-specific port used to connect to the machine.",
		"reachability_mode":                 "Reachability topology for the machine: local, direct_connect, or reverse_connect.",
		"execution_mode":                    "Execution path currently used by this record: local_process or websocket. Older records may still surface as ssh_compat until they are migrated.",
		"execution_capabilities":            "Runtime execution capabilities derived from the actually implemented path for this machine record.",
		"ssh_helper_enabled":                "Whether SSH helper credentials are configured for bootstrap or diagnostics.",
		"ssh_helper_required":               "Whether this record still reflects legacy ssh_compat state that should be migrated to websocket.",
		"connection_mode":                   "Legacy compatibility field derived from reachability_mode and execution_mode. New clients should prefer the separated fields.",
		"transport_capabilities":            "Legacy compatibility alias for execution_capabilities.",
		"ssh_user":                          "SSH helper username used for bootstrap, diagnostics, or emergency repair access.",
		"ssh_key_path":                      "Path to the SSH private key used for SSH helper bootstrap, diagnostics, or emergency repair access.",
		"advertised_endpoint":               "Direct-connect websocket endpoint advertised by the machine when execution_mode is websocket and reachability_mode is direct_connect.",
		"daemon_status":                     "Daemon registration and session metadata for websocket-capable machine transports.",
		"daemon_status.registered":          "Whether the machine daemon currently has an active registration with the control plane.",
		"daemon_status.last_registered_at":  "RFC3339 timestamp for the daemon's last successful registration heartbeat.",
		"daemon_status.current_session_id":  "Current daemon transport session identifier, when one is active.",
		"daemon_status.session_state":       "Current machine transport session state reported for the daemon connection.",
		"detected_os":                       "Detected operating system reported for the machine.",
		"detected_arch":                     "Detected CPU architecture reported for the machine.",
		"detection_status":                  "Status of machine OS and architecture detection.",
		"detection_message":                 "User-facing explanation of the current machine detection result and any manual follow-up needed.",
		"channel_credential":                "Machine channel credential reference reserved for transport registration, kept separate from runtime agent tokens.",
		"channel_credential.kind":           "Credential kind reserved for machine transport registration, such as none, token, or certificate.",
		"channel_credential.token_id":       "Opaque token identifier reserved for machine channel registration, distinct from runtime agent tokens.",
		"channel_credential.certificate_id": "Opaque certificate identifier reserved for machine channel registration.",
		"description":                       "Human-readable machine description.",
		"labels":                            "Labels attached to the machine for operator reference.",
		"status":                            "Machine lifecycle status value.",
		"workspace_root":                    "Filesystem root directory where ticket workspaces are created on the machine.",
		"agent_cli_path":                    "Absolute path to the agent CLI executable on the machine.",
		"env_vars":                          "Environment variable entries exported when work runs on the machine.",
	}
	openAPIProjectRequestDescriptions = map[string]string{
		"name":                      "Human-readable project name.",
		"slug":                      "Stable URL-safe project slug.",
		"description":               "Human-readable project description.",
		"status":                    "Current project lifecycle status name.",
		"default_agent_provider_id": "Optional default agent provider ID for the project.",
		"accessible_machine_ids":    "Machine IDs that the project is allowed to use.",
		"max_concurrent_agents":     "Maximum number of agents that may run concurrently in the project.",
		"agent_run_summary_prompt":  "Optional project-level prompt override for asynchronous terminal run summaries. Leave blank to use the built-in default prompt.",
	}
	openAPIProviderRequestDescriptions = map[string]string{
		"name":                  "Human-readable provider name.",
		"machine_id":            "Machine ID where this provider runs.",
		"adapter_type":          "Adapter type used to launch and communicate with the provider.",
		"permission_profile":    "Managed permission profile used to render adapter-specific approval and sandbox options.",
		"cli_command":           "CLI command used to launch the provider.",
		"cli_args":              "Additional CLI arguments passed to the provider command after OpenASE applies adapter-managed launch settings.",
		"auth_config":           "Provider-specific authentication configuration object.",
		"model_name":            "Model name configured for the provider.",
		"model_temperature":     "Sampling temperature configured for the provider model.",
		"model_max_tokens":      "Maximum number of output tokens allowed for the provider model.",
		"max_parallel_runs":     "Maximum number of concurrent runs allowed for the provider.",
		"cost_per_input_token":  "Estimated USD cost per input token.",
		"cost_per_output_token": "Estimated USD cost per output token.",
		"pricing_config":        "Structured pricing configuration, including official defaults, cache-aware rates, and tiered pricing metadata.",
	}
	openAPIRepoRequestDescriptions = map[string]string{
		"name":              "Human-readable repository name within the project.",
		"repository_url":    "Remote Git repository URL.",
		"default_branch":    "Repository base branch used when OpenASE creates a new ticket work branch.",
		"workspace_dirname": "Directory name used for this repository inside a ticket workspace.",
		"labels":            "Labels attached to the repository for workflow selection and filtering.",
	}
	openAPIGitHubRepositoryDescriptions = map[string]string{
		"owner":       "GitHub user or organization namespace that owns the repository.",
		"name":        "Repository name to create inside the selected GitHub namespace.",
		"description": "Optional GitHub repository description.",
		"visibility":  "GitHub repository visibility. Supported values are private and public.",
		"auto_init":   "Whether GitHub should initialize the repository with a default branch so OpenASE can bind it immediately.",
	}
	// #nosec G101 -- "token" is an OpenAPI field name/description, not a credential literal.
	openAPIGitHubCredentialDescriptions = map[string]string{
		"scope": "Credential scope to mutate. Supported values are organization and project.",
		"token": "GitHub token value copied into platform-managed secret storage.",
	}
	openAPIAgentRequestDescriptions = map[string]string{
		"name":        "Human-readable agent name.",
		"provider_id": "Agent provider ID used to run the agent.",
	}
	openAPIWorkflowRequestDescriptions = map[string]string{
		"name":                    "Human-readable workflow name.",
		"type":                    "Workflow type, such as coding, test, doc, security, deploy, refine-harness, or custom.",
		"agent_id":                "Agent ID assigned to execute this workflow.",
		"role_slug":               "Stable workflow role slug stored in structured workflow metadata.",
		"role_name":               "Human-readable workflow role name stored in structured workflow metadata.",
		"role_description":        "Structured workflow role summary shown in runtime and editor context.",
		"platform_access_allowed": "Allowed OpenASE platform API scopes for agents running this workflow.",
		"skill_names":             "Skill names to bind to the workflow when it is created.",
		"created_by":              "Optional creator descriptor recorded on the initial workflow harness version.",
		"edited_by":               "Optional editor descriptor recorded on subsequent workflow harness versions.",
		"pickup_status_ids":       "Ticket status IDs that allow the workflow to pick up tickets.",
		"finish_status_ids":       "Ticket status IDs that mark workflow completion.",
		"is_active":               "Whether the workflow is active and eligible to pick up work.",
		"max_concurrent":          "Maximum number of concurrent runs allowed for the workflow.",
		"timeout_minutes":         "Hard execution timeout for workflow runs, in minutes.",
		"stall_timeout_minutes":   "Timeout for detecting stalled workflow runs, in minutes.",
		"max_retry_attempts":      "Maximum retry attempts before the workflow run fails permanently.",
		"harness_path":            "Logical harness path tracked by the control plane for this workflow.",
		"harness_content":         "Initial pure Markdown or Gonja harness body written into the versioned control-plane workflow record.",
		"hooks":                   "Workflow hook configuration keyed by lifecycle phase.",
	}
	openAPIHarnessContentDescriptions = map[string]string{
		"content":   "Harness content to write or validate.",
		"edited_by": "Optional editor descriptor recorded on the published workflow harness version.",
	}
	openAPIScheduledJobDescriptions = map[string]string{
		"name":            "Human-readable scheduled job name.",
		"cron_expression": "Cron expression that controls when the job triggers.",
		"ticket_template": "Ticket template used to create a ticket for each scheduled run.",
		"is_enabled":      "Whether the scheduled job is enabled.",
	}
	openAPITicketRequestDescriptions = map[string]string{
		"title":                     "Human-readable ticket title.",
		"description":               "Ticket description or problem statement.",
		"status_id":                 "Optional ticket status ID to assign explicitly.",
		"archived":                  "Whether the ticket is archived and excluded from active board and pickup views.",
		"priority":                  "Ticket priority value.",
		"type":                      "Ticket type value.",
		"workflow_id":               "Optional workflow ID that should handle the ticket.",
		"repo_scopes":               "Optional repository scopes attached at ticket creation time. Multi-repo projects must supply explicit repo scopes; single-repo projects auto-select the only repo when omitted.",
		"repo_scopes[].repo_id":     "Repository ID attached to the ticket scope.",
		"repo_scopes[].branch_name": "Optional work-branch override for the scoped repository. When omitted or blank, OpenASE uses the generated ticket branch.",
		"parent_ticket_id":          "Optional parent ticket ID for hierarchical ticket relationships.",
		"external_ref":              "Optional external reference string associated with the ticket.",
		"created_by":                "Actor identifier recorded as the creator of the ticket.",
		"budget_usd":                "Optional budget limit for the ticket in USD.",
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
	openAPIProjectUpdateThreadRequestDescriptions = map[string]string{
		"status":     "Current delivery status for the update thread. Supported values are on_track, at_risk, and off_track.",
		"title":      "Human-readable project update title.",
		"body":       "Markdown body content for the project update thread.",
		"created_by": "Actor identifier recorded as the creator of the update thread.",
	}
	openAPIProjectUpdateThreadPatchDescriptions = map[string]string{
		"status":      "Updated delivery status for the update thread. Supported values are on_track, at_risk, and off_track.",
		"title":       "Updated human-readable project update title.",
		"body":        "Updated markdown body content for the project update thread.",
		"edited_by":   "Actor identifier recorded as the editor of the update thread.",
		"edit_reason": "Reason recorded for editing the update thread.",
	}
	openAPIProjectUpdateCommentRequestDescriptions = map[string]string{
		"body":       "Markdown body content for the project update comment.",
		"created_by": "Actor identifier recorded as the creator of the update comment.",
	}
	openAPIProjectUpdateCommentPatchDescriptions = map[string]string{
		"body":        "Updated markdown body content for the project update comment.",
		"edited_by":   "Actor identifier recorded as the editor of the update comment.",
		"edit_reason": "Reason recorded for editing the update comment.",
	}
	openAPIDependencyRequestDescriptions = map[string]string{
		"type":             "Dependency relationship type. Supported values: blocks, blocked_by, sub_issue.",
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
	openAPIStatusRequestDescriptions = map[string]string{
		"name":            "Human-readable status name.",
		"stage":           "Lifecycle stage for the status. One of backlog, unstarted, started, completed, or canceled.",
		"color":           "Display color for the status.",
		"icon":            "Display icon identifier for the status.",
		"position":        "Zero-based display order of the status.",
		"max_active_runs": "Maximum number of active runs allowed in this status.",
		"is_default":      "Whether this status should become the default status.",
		"description":     "Human-readable status description.",
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
		"branch_name":      "Optional work-branch override for the scoped repository. Leave blank to use the generated ticket branch.",
		"pull_request_url": "Pull request URL associated with the repository scope.",
	}
	openAPIRepoScopePatchDescriptions = map[string]string{
		"branch_name":      "Optional work-branch override for the scoped repository. Send an empty string to clear the override and use the generated ticket branch.",
		"pull_request_url": "Pull request URL associated with the repository scope.",
	}
	openAPIHRAdvisorActivateDescriptions = map[string]string{
		"role_slug":               "HR advisor role slug to activate for the project.",
		"create_bootstrap_ticket": "Whether activation should create a bootstrap ticket immediately.",
	}
	openAPIChatRequestDescriptions = map[string]string{
		"message":               "User message content for the chat turn.",
		"provider_id":           "Optional provider ID used to run this chat session.",
		"session_id":            "Optional existing chat session ID to resume.",
		"source":                "Source identifier for the chat request, such as web or cli.",
		"context":               "Optional project, ticket, or workflow context attached to the chat turn.",
		"context.project_id":    "Project ID supplied to ground the chat request.",
		"context.harness_draft": "Optional unsaved harness draft from the editor so the AI can reason about the current in-memory workflow text.",
		"context.ticket_id":     "Optional ticket ID supplied to ground the chat request.",
		"context.workflow_id":   "Optional workflow ID supplied to ground the chat request.",
	}
	openAPIProjectConversationCreateDescriptions = map[string]string{
		"source":             "Project conversation source and currently must be project_sidebar.",
		"provider_id":        "Provider ID used to run this persistent project conversation.",
		"context":            "Project conversation context.",
		"context.project_id": "Project ID that owns the conversation workspace and transcript.",
	}
	openAPIProjectConversationTurnDescriptions = map[string]string{
		"message":                                           "User message content appended as the next project conversation turn.",
		"focus":                                             "Optional per-turn focus context describing the currently selected workflow, skill, ticket, or machine surface.",
		"focus.kind":                                        "Focused surface kind. Supported values are workflow, skill, ticket, and machine.",
		"focus.workflow_id":                                 "Workflow ID currently in focus.",
		"focus.workflow_name":                               "Workflow name currently in focus.",
		"focus.workflow_type":                               "Workflow type currently in focus.",
		"focus.harness_path":                                "Harness path for the focused workflow.",
		"focus.is_active":                                   "Whether the focused workflow is currently active.",
		"focus.selected_area":                               "UI sub-area currently in focus, such as harness, detail, or health.",
		"focus.has_dirty_draft":                             "Whether the focused workflow or skill surface currently has unsaved draft edits.",
		"focus.skill_id":                                    "Skill ID currently in focus.",
		"focus.skill_name":                                  "Skill name currently in focus.",
		"focus.selected_file_path":                          "Selected bundle file path for the focused skill surface.",
		"focus.bound_workflow_names":                        "Workflow names currently bound to the focused skill.",
		"focus.ticket_id":                                   "Ticket ID currently in focus.",
		"focus.ticket_identifier":                           "Human-readable ticket identifier currently in focus.",
		"focus.ticket_title":                                "Ticket title currently in focus.",
		"focus.ticket_description":                          "Ticket description currently in focus.",
		"focus.ticket_status":                               "Ticket status currently in focus.",
		"focus.ticket_priority":                             "Ticket priority currently in focus.",
		"focus.ticket_attempt_count":                        "Current attempt count for the focused ticket.",
		"focus.ticket_retry_paused":                         "Whether retries are currently paused for the focused ticket.",
		"focus.ticket_pause_reason":                         "Reason retries are paused for the focused ticket.",
		"focus.ticket_dependencies":                         "Dependency summary for the focused ticket.",
		"focus.ticket_dependencies[].identifier":            "Human-readable identifier for a related dependency ticket.",
		"focus.ticket_dependencies[].title":                 "Title of a related dependency ticket.",
		"focus.ticket_dependencies[].relation":              "Dependency relation such as blocks or blocked_by.",
		"focus.ticket_dependencies[].status":                "Current status of the related dependency ticket.",
		"focus.ticket_repo_scopes":                          "Repo scopes and PR references for the focused ticket.",
		"focus.ticket_repo_scopes[].repo_id":                "Repository ID included in the focused ticket scope.",
		"focus.ticket_repo_scopes[].repo_name":              "Repository name included in the focused ticket scope.",
		"focus.ticket_repo_scopes[].branch_name":            "Branch name associated with the focused ticket scope.",
		"focus.ticket_repo_scopes[].pull_request_url":       "Pull request URL associated with the focused ticket scope.",
		"focus.ticket_recent_activity":                      "Recent ticket-scoped activity for the focused ticket.",
		"focus.ticket_recent_activity[].event_type":         "Activity event type for a recent focused ticket event.",
		"focus.ticket_recent_activity[].message":            "Human-readable summary for a recent focused ticket event.",
		"focus.ticket_recent_activity[].created_at":         "Creation timestamp for a recent focused ticket event.",
		"focus.ticket_hook_history":                         "Recent hook execution history for the focused ticket.",
		"focus.ticket_hook_history[].hook_name":             "Hook name for a recent focused ticket hook execution.",
		"focus.ticket_hook_history[].status":                "Execution status for a recent focused ticket hook.",
		"focus.ticket_hook_history[].output":                "Captured output summary for a recent focused ticket hook.",
		"focus.ticket_hook_history[].timestamp":             "Execution timestamp for a recent focused ticket hook.",
		"focus.ticket_assigned_agent":                       "Assigned agent summary for the focused ticket.",
		"focus.ticket_assigned_agent.id":                    "Assigned agent ID for the focused ticket.",
		"focus.ticket_assigned_agent.name":                  "Assigned agent name for the focused ticket.",
		"focus.ticket_assigned_agent.provider":              "Assigned agent provider name for the focused ticket.",
		"focus.ticket_assigned_agent.runtime_control_state": "Assigned agent runtime control state for the focused ticket.",
		"focus.ticket_assigned_agent.runtime_phase":         "Assigned agent runtime phase for the focused ticket.",
		"focus.ticket_current_run":                          "Current run summary for the focused ticket.",
		"focus.ticket_current_run.id":                       "Current run ID for the focused ticket.",
		"focus.ticket_current_run.attempt_number":           "Attempt number for the current focused ticket run.",
		"focus.ticket_current_run.status":                   "Current run status for the focused ticket.",
		"focus.ticket_current_run.current_step_status":      "Current step status for the focused ticket run.",
		"focus.ticket_current_run.current_step_summary":     "Current step summary for the focused ticket run.",
		"focus.ticket_current_run.last_error":               "Last error observed on the focused ticket run.",
		"focus.ticket_target_machine":                       "Target machine summary for the focused ticket.",
		"focus.ticket_target_machine.id":                    "Target machine ID for the focused ticket.",
		"focus.ticket_target_machine.name":                  "Target machine name for the focused ticket.",
		"focus.ticket_target_machine.host":                  "Target machine host for the focused ticket.",
		"focus.machine_id":                                  "Machine ID currently in focus.",
		"focus.machine_name":                                "Machine name currently in focus.",
		"focus.machine_host":                                "Machine host currently in focus.",
		"focus.machine_status":                              "Machine runtime status currently in focus.",
		"focus.health_summary":                              "Compact health or resource summary for the focused machine.",
	}
	openAPIProjectConversationInterruptResponseDescriptions = map[string]string{
		"decision": "Provider-native interrupt decision identifier such as approve_once.",
		"answer":   "Structured answer payload for requestUserInput interrupts.",
	}
	openAPIRoleBindingRequestDescriptions = map[string]string{
		"subject_kind": "Binding subject kind. Supported values are user and group.",
		"subject_key":  "Stable user identifier/email or synchronized OIDC group key that receives the role.",
		"role_key":     "Builtin OpenASE role key to grant on the selected scope.",
		"expires_at":   "Optional RFC3339 timestamp after which the binding automatically expires.",
	}
	openAPISkillBindingDescriptions = map[string]string{
		"skills": "Skill names included in this workflow skill binding request.",
	}
	openAPISkillCreateDescriptions = map[string]string{
		"name":        "Project-unique skill name in the control plane.",
		"content":     "Skill markdown content. Frontmatter is optional on input and will be normalized on write.",
		"description": "Optional description used when the input content does not declare one.",
		"created_by":  "Optional creator descriptor such as user:gary or agent:codex-01 via ASE-42.",
		"is_enabled":  "Whether the new skill should be enabled for runtime injection immediately.",
	}
	openAPISkillUpdateDescriptions = map[string]string{
		"content":                "Replacement skill markdown content. Frontmatter is optional on input and will be normalized on write.",
		"description":            "Optional description override used when the input content does not declare one.",
		"files":                  "Optional replacement skill bundle files. When present, the request publishes a new bundle version from the supplied file list.",
		"files[].path":           "Bundle-relative file path using forward slashes.",
		"files[].content_base64": "Base64-encoded file bytes for this bundle entry.",
		"files[].media_type":     "Optional media type persisted with the file entry.",
		"files[].is_executable":  "Whether the projected file should be marked executable at runtime.",
	}
	openAPISkillSyncDescriptions = map[string]string{
		"workspace_root": "Workspace repository root that owns the agent skill directory.",
		"adapter_type":   "Agent adapter type used to derive the runtime skill directory.",
		"workflow_id":    "Optional workflow ID used to project only the currently bound enabled skills.",
	}
	openAPISkillRefinementDescriptions = map[string]string{
		"project_id":             "Project ID that owns the skill draft and provider selection.",
		"message":                "Requested improvement goal that Codex should fix and verify against the current draft bundle.",
		"provider_id":            "Optional provider ID. Phase 1 supports Codex-backed refinement only.",
		"files":                  "Current draft skill bundle files from the editor.",
		"files[].path":           "Bundle-relative file path using forward slashes.",
		"files[].content_base64": "Base64-encoded file bytes for this draft bundle entry.",
		"files[].media_type":     "Optional media type persisted with the file entry.",
		"files[].is_executable":  "Whether the projected file should be marked executable at runtime.",
	}
	openAPISkillBindingTargetDescriptions = map[string]string{
		"workflow_ids": "Workflow IDs that should bind or unbind this skill.",
	}
	openAPIRequestBodyDescriptions = map[string]map[string]string{
		"POST /api/v1/orgs":                                                                            openAPIOrganizationRequestDescriptions,
		"PATCH /api/v1/orgs/{orgId}":                                                                   openAPIOrganizationRequestDescriptions,
		"POST /api/v1/orgs/{orgId}/channels":                                                           openAPIChannelRequestDescriptions,
		"PATCH /api/v1/channels/{channelId}":                                                           openAPIChannelRequestDescriptions,
		"POST /api/v1/orgs/{orgId}/machines":                                                           openAPIMachineRequestDescriptions,
		"PATCH /api/v1/machines/{machineId}":                                                           openAPIMachineRequestDescriptions,
		"POST /api/v1/orgs/{orgId}/projects":                                                           openAPIProjectRequestDescriptions,
		"PATCH /api/v1/projects/{projectId}":                                                           openAPIProjectRequestDescriptions,
		"POST /api/v1/orgs/{orgId}/providers":                                                          openAPIProviderRequestDescriptions,
		"PATCH /api/v1/providers/{providerId}":                                                         openAPIProviderRequestDescriptions,
		"POST /api/v1/projects/{projectId}/repos":                                                      openAPIRepoRequestDescriptions,
		"PATCH /api/v1/projects/{projectId}/repos/{repoId}":                                            openAPIRepoRequestDescriptions,
		"POST /api/v1/projects/{projectId}/github/repos":                                               openAPIGitHubRepositoryDescriptions,
		"PUT /api/v1/projects/{projectId}/security-settings/github-outbound-credential":                openAPIGitHubCredentialDescriptions,
		"POST /api/v1/projects/{projectId}/security-settings/github-outbound-credential/import-gh-cli": openAPIGitHubCredentialDescriptions,
		"POST /api/v1/projects/{projectId}/security-settings/github-outbound-credential/retest":        openAPIGitHubCredentialDescriptions,
		"POST /api/v1/projects/{projectId}/agents":                                                     openAPIAgentRequestDescriptions,
		"PATCH /api/v1/agents/{agentId}":                                                               openAPIAgentRequestDescriptions,
		"POST /api/v1/projects/{projectId}/workflows":                                                  openAPIWorkflowRequestDescriptions,
		"PATCH /api/v1/workflows/{workflowId}":                                                         mergeRequestFieldDescriptions(openAPIWorkflowRequestDescriptions, map[string]string{"harness_content": ""}),
		"POST /api/v1/workflows/{workflowId}/retire":                                                   map[string]string{"edited_by": openAPIWorkflowRequestDescriptions["edited_by"]},
		"POST /api/v1/workflows/{workflowId}/replace-references":                                       map[string]string{"replacement_workflow_id": "Workflow ID that should receive replaceable scheduled job and active ticket references.", "edited_by": openAPIWorkflowRequestDescriptions["edited_by"]},
		"PUT /api/v1/workflows/{workflowId}/harness":                                                   openAPIHarnessContentDescriptions,
		"POST /api/v1/harness/validate":                                                                openAPIHarnessContentDescriptions,
		"POST /api/v1/projects/{projectId}/scheduled-jobs":                                             openAPIScheduledJobDescriptions,
		"PATCH /api/v1/scheduled-jobs/{jobId}":                                                         openAPIScheduledJobDescriptions,
		"POST /api/v1/projects/{projectId}/tickets":                                                    openAPITicketRequestDescriptions,
		"PATCH /api/v1/tickets/{ticketId}":                                                             openAPITicketRequestDescriptions,
		"POST /api/v1/tickets/{ticketId}/comments":                                                     openAPITicketCommentRequestDescriptions,
		"PATCH /api/v1/tickets/{ticketId}/comments/{commentId}":                                        openAPITicketCommentPatchDescriptions,
		"POST /api/v1/projects/{projectId}/updates":                                                    openAPIProjectUpdateThreadRequestDescriptions,
		"PATCH /api/v1/projects/{projectId}/updates/{threadId}":                                        openAPIProjectUpdateThreadPatchDescriptions,
		"POST /api/v1/projects/{projectId}/updates/{threadId}/comments":                                openAPIProjectUpdateCommentRequestDescriptions,
		"PATCH /api/v1/projects/{projectId}/updates/{threadId}/comments/{commentId}":                   openAPIProjectUpdateCommentPatchDescriptions,
		"POST /api/v1/tickets/{ticketId}/dependencies":                                                 openAPIDependencyRequestDescriptions,
		"POST /api/v1/tickets/{ticketId}/external-links":                                               openAPIExternalLinkRequestDescriptions,
		"POST /api/v1/projects/{projectId}/statuses":                                                   openAPIStatusRequestDescriptions,
		"PATCH /api/v1/statuses/{statusId}":                                                            openAPIStatusRequestDescriptions,
		"POST /api/v1/projects/{projectId}/notification-rules":                                         openAPINotificationRuleDescriptions,
		"PATCH /api/v1/notification-rules/{ruleId}":                                                    openAPINotificationRuleDescriptions,
		"POST /api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes":                             openAPIRepoScopeCreateDescriptions,
		"PATCH /api/v1/projects/{projectId}/tickets/{ticketId}/repo-scopes/{scopeId}":                  openAPIRepoScopePatchDescriptions,
		"POST /api/v1/projects/{projectId}/hr-advisor/activate":                                        openAPIHRAdvisorActivateDescriptions,
		"POST /api/v1/chat":                                                                            openAPIChatRequestDescriptions,
		"POST /api/v1/chat/conversations":                                                              openAPIProjectConversationCreateDescriptions,
		"POST /api/v1/chat/conversations/{conversationId}/turns":                                       openAPIProjectConversationTurnDescriptions,
		"POST /api/v1/chat/conversations/{conversationId}/interrupts/{interruptId}/respond":            openAPIProjectConversationInterruptResponseDescriptions,
		"POST /api/v1/organizations/{orgId}/role-bindings":                                             openAPIRoleBindingRequestDescriptions,
		"POST /api/v1/projects/{projectId}/role-bindings":                                              openAPIRoleBindingRequestDescriptions,
		"POST /api/v1/projects/{projectId}/skills":                                                     openAPISkillCreateDescriptions,
		"POST /api/v1/projects/{projectId}/skills/refresh":                                             openAPISkillSyncDescriptions,
		"PUT /api/v1/skills/{skillId}":                                                                 openAPISkillUpdateDescriptions,
		"POST /api/v1/skills/{skillId}/bind":                                                           openAPISkillBindingTargetDescriptions,
		"POST /api/v1/skills/{skillId}/unbind":                                                         openAPISkillBindingTargetDescriptions,
		"POST /api/v1/skills/{skillId}/refinement-runs":                                                openAPISkillRefinementDescriptions,
		"POST /api/v1/workflows/{workflowId}/skills/bind":                                              openAPISkillBindingDescriptions,
		"POST /api/v1/workflows/{workflowId}/skills/unbind":                                            openAPISkillBindingDescriptions,
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
			Version:     "0.1.1",
			Description: "OpenAPI contract for the current OpenASE web control-plane surface.",
		},
		Paths: openapi3.NewPaths(),
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{},
		},
		Tags: openapi3.Tags{
			{Name: "system"},
			{Name: "auth"},
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
	if err := builder.addAuthOperations(); err != nil {
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

func (b openAPISpecBuilder) addAuthOperations() error {
	oidcStart := openapi3.NewOperation()
	oidcStart.OperationID = "startOIDCLogin"
	oidcStart.Summary = "Start the browser OIDC login flow"
	oidcStart.Tags = []string{"auth"}
	oidcStart.Responses = openapi3.NewResponsesWithCapacity(3)
	oidcStart.AddResponse(http.StatusFound, openapi3.NewResponse().WithDescription("OIDC login redirect response."))
	for _, code := range []int{http.StatusNotFound, http.StatusBadGateway} {
		errorResponse, err := b.errorResponse(code)
		if err != nil {
			return err
		}
		oidcStart.AddResponse(code, errorResponse)
	}
	oidcStart.AddParameter(openapi3.NewQueryParameter("return_to").
		WithDescription("Optional same-origin path to redirect back to after login completes.").
		WithSchema(openapi3.NewStringSchema()),
	)
	b.doc.AddOperation("/api/v1/auth/oidc/start", http.MethodGet, oidcStart)

	oidcCallback := openapi3.NewOperation()
	oidcCallback.OperationID = "handleOIDCCallback"
	oidcCallback.Summary = "Complete the browser OIDC login callback"
	oidcCallback.Tags = []string{"auth"}
	oidcCallback.Responses = openapi3.NewResponsesWithCapacity(4)
	oidcCallback.AddResponse(http.StatusFound, openapi3.NewResponse().WithDescription("OIDC callback redirect response."))
	for _, code := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound} {
		errorResponse, err := b.errorResponse(code)
		if err != nil {
			return err
		}
		oidcCallback.AddResponse(code, errorResponse)
	}
	oidcCallback.AddParameter(openapi3.NewQueryParameter("code").
		WithDescription("Authorization code returned by the upstream OIDC provider.").
		WithRequired(true).
		WithSchema(openapi3.NewStringSchema()),
	)
	oidcCallback.AddParameter(openapi3.NewQueryParameter("state").
		WithDescription("Opaque OIDC state value that must match the login flow cookie.").
		WithRequired(true).
		WithSchema(openapi3.NewStringSchema()),
	)
	b.doc.AddOperation("/api/v1/auth/oidc/callback", http.MethodGet, oidcCallback)

	authSession, err := b.jsonOperation(
		"getAuthSession",
		"Get the current browser human-auth session",
		[]string{"auth"},
		http.StatusOK,
		OpenAPIAuthSessionResponse{},
		nil,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/auth/session", http.MethodGet, authSession)

	authLogout := openapi3.NewOperation()
	authLogout.OperationID = "logoutHumanSession"
	authLogout.Summary = "Revoke the current browser human-auth session"
	authLogout.Tags = []string{"auth"}
	authLogout.Responses = openapi3.NewResponsesWithCapacity(2)
	authLogout.AddResponse(http.StatusNoContent, openapi3.NewResponse().WithDescription("Browser session revoked."))
	errorResponse, err := b.errorResponse(http.StatusForbidden)
	if err != nil {
		return err
	}
	authLogout.AddResponse(http.StatusForbidden, errorResponse)
	b.doc.AddOperation("/api/v1/auth/logout", http.MethodPost, authLogout)

	myPermissions, err := b.jsonOperation(
		"getMyEffectivePermissions",
		"Get effective OpenASE roles and permissions for the authenticated human",
		[]string{"auth"},
		http.StatusOK,
		OpenAPIAuthPermissionsResponse{},
		nil,
		http.StatusUnauthorized,
		http.StatusForbidden,
	)
	if err != nil {
		return err
	}
	myPermissions.AddParameter(uuidQueryParameter("project_id", "Optional project scope to evaluate."))
	myPermissions.AddParameter(uuidQueryParameter("org_id", "Optional organization scope to evaluate."))
	b.doc.AddOperation("/api/v1/auth/me/permissions", http.MethodGet, myPermissions)

	orgRoleBindings, err := b.jsonOperation(
		"listOrganizationRoleBindings",
		"List organization-scoped role bindings",
		[]string{"auth"},
		http.StatusOK,
		OpenAPIRoleBindingsResponse{},
		nil,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	orgRoleBindings.AddParameter(uuidPathParameter("orgId", "Organization ID that owns the role bindings."))
	b.doc.AddOperation("/api/v1/organizations/{orgId}/role-bindings", http.MethodGet, orgRoleBindings)

	createOrgRoleBinding, err := b.jsonOperation(
		"createOrganizationRoleBinding",
		"Create an organization-scoped role binding",
		[]string{"auth"},
		http.StatusCreated,
		OpenAPIRoleBindingResponse{},
		OpenAPICreateRoleBindingRequest{},
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
	)
	if err != nil {
		return err
	}
	createOrgRoleBinding.AddParameter(uuidPathParameter("orgId", "Organization ID that owns the role bindings."))
	b.doc.AddOperation("/api/v1/organizations/{orgId}/role-bindings", http.MethodPost, createOrgRoleBinding)

	deleteOrgRoleBinding := openapi3.NewOperation()
	deleteOrgRoleBinding.OperationID = "deleteOrganizationRoleBinding"
	deleteOrgRoleBinding.Summary = "Delete an organization-scoped role binding"
	deleteOrgRoleBinding.Tags = []string{"auth"}
	deleteOrgRoleBinding.Responses = openapi3.NewResponsesWithCapacity(4)
	deleteOrgRoleBinding.AddResponse(http.StatusNoContent, openapi3.NewResponse().WithDescription("Role binding deleted."))
	for _, code := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden} {
		errorResponse, err := b.errorResponse(code)
		if err != nil {
			return err
		}
		deleteOrgRoleBinding.AddResponse(code, errorResponse)
	}
	deleteOrgRoleBinding.AddParameter(uuidPathParameter("orgId", "Organization ID that owns the role bindings."))
	deleteOrgRoleBinding.AddParameter(uuidPathParameter("bindingId", "Role binding ID to delete."))
	b.doc.AddOperation("/api/v1/organizations/{orgId}/role-bindings/{bindingId}", http.MethodDelete, deleteOrgRoleBinding)

	projectRoleBindings, err := b.jsonOperation(
		"listProjectRoleBindings",
		"List project-scoped role bindings",
		[]string{"auth"},
		http.StatusOK,
		OpenAPIRoleBindingsResponse{},
		nil,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectRoleBindings.AddParameter(uuidPathParameter("projectId", "Project ID that owns the role bindings."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/role-bindings", http.MethodGet, projectRoleBindings)

	createProjectRoleBinding, err := b.jsonOperation(
		"createProjectRoleBinding",
		"Create a project-scoped role binding",
		[]string{"auth"},
		http.StatusCreated,
		OpenAPIRoleBindingResponse{},
		OpenAPICreateRoleBindingRequest{},
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
	)
	if err != nil {
		return err
	}
	createProjectRoleBinding.AddParameter(uuidPathParameter("projectId", "Project ID that owns the role bindings."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/role-bindings", http.MethodPost, createProjectRoleBinding)

	deleteProjectRoleBinding := openapi3.NewOperation()
	deleteProjectRoleBinding.OperationID = "deleteProjectRoleBinding"
	deleteProjectRoleBinding.Summary = "Delete a project-scoped role binding"
	deleteProjectRoleBinding.Tags = []string{"auth"}
	deleteProjectRoleBinding.Responses = openapi3.NewResponsesWithCapacity(4)
	deleteProjectRoleBinding.AddResponse(http.StatusNoContent, openapi3.NewResponse().WithDescription("Role binding deleted."))
	for _, code := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden} {
		errorResponse, err := b.errorResponse(code)
		if err != nil {
			return err
		}
		deleteProjectRoleBinding.AddResponse(code, errorResponse)
	}
	deleteProjectRoleBinding.AddParameter(uuidPathParameter("projectId", "Project ID that owns the role bindings."))
	deleteProjectRoleBinding.AddParameter(uuidPathParameter("bindingId", "Role binding ID to delete."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/role-bindings/{bindingId}", http.MethodDelete, deleteProjectRoleBinding)

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

	orgTokenUsageGet, err := b.jsonOperation(
		"getOrganizationTokenUsage",
		"Get organization daily token usage for a UTC date range",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIOrganizationTokenUsageResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	orgTokenUsageGet.AddParameter(uuidPathParameter("orgId", "Organization ID."))
	orgTokenUsageGet.AddParameter(openapi3.NewQueryParameter("from").
		WithDescription("UTC start date in YYYY-MM-DD format. Defaults to the last 30 days when omitted with to.").
		WithSchema(openapi3.NewStringSchema()))
	orgTokenUsageGet.AddParameter(openapi3.NewQueryParameter("to").
		WithDescription("UTC end date in YYYY-MM-DD format. Defaults to today when omitted with from.").
		WithSchema(openapi3.NewStringSchema()))
	b.doc.AddOperation("/api/v1/orgs/{orgId}/token-usage", http.MethodGet, orgTokenUsageGet)

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

	providerGet, err := b.jsonOperation(
		"getAgentProvider",
		"Get an agent provider",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentProviderResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	providerGet.AddParameter(uuidPathParameter("providerId", "Agent provider ID."))
	b.doc.AddOperation("/api/v1/providers/{providerId}", http.MethodGet, providerGet)

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

	projectTokenUsageGet, err := b.jsonOperation(
		"getProjectTokenUsage",
		"Get project daily token usage for a UTC date range",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectTokenUsageResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectTokenUsageGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectTokenUsageGet.AddParameter(openapi3.NewQueryParameter("from").
		WithDescription("UTC start date in YYYY-MM-DD format. Defaults to the last 30 days when omitted with to.").
		WithSchema(openapi3.NewStringSchema()))
	projectTokenUsageGet.AddParameter(openapi3.NewQueryParameter("to").
		WithDescription("UTC end date in YYYY-MM-DD format. Defaults to today when omitted with from.").
		WithSchema(openapi3.NewStringSchema()))
	b.doc.AddOperation("/api/v1/projects/{projectId}/token-usage", http.MethodGet, projectTokenUsageGet)

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

	githubNamespacesGet, err := b.jsonOperation(
		"listGitHubNamespaces",
		"List GitHub namespaces available to the project's effective credential",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIGitHubRepositoryNamespacesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusPreconditionFailed,
		http.StatusForbidden,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	githubNamespacesGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/github/namespaces", http.MethodGet, githubNamespacesGet)

	githubReposGet, err := b.jsonOperation(
		"listGitHubRepositories",
		"List GitHub repositories visible to the project's effective credential",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIGitHubRepositoriesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusPreconditionFailed,
		http.StatusForbidden,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	githubReposGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	githubReposGet.AddParameter(openapi3.NewQueryParameter("query").
		WithDescription("Optional case-insensitive search query matched against repo name, owner, and full name.").
		WithSchema(openapi3.NewStringSchema()))
	githubReposGet.AddParameter(openapi3.NewQueryParameter("cursor").
		WithDescription("Pagination cursor returned by the previous GitHub repository response.").
		WithSchema(openapi3.NewStringSchema()))
	b.doc.AddOperation("/api/v1/projects/{projectId}/github/repos", http.MethodGet, githubReposGet)

	githubReposPost, err := b.jsonOperation(
		"createGitHubRepository",
		"Create a GitHub repository using the project's effective credential",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPIGitHubRepositoryResponse{},
		OpenAPICreateGitHubRepositoryRequest{},
		http.StatusBadRequest,
		http.StatusPreconditionFailed,
		http.StatusForbidden,
		http.StatusConflict,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	githubReposPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/github/repos", http.MethodPost, githubReposPost)

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

	agentPatch, err := b.jsonOperation(
		"updateAgent",
		"Update an agent definition",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIAgentResponse{},
		OpenAPIUpdateAgentRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	agentPatch.AddParameter(uuidPathParameter("agentId", "Agent ID."))
	b.doc.AddOperation("/api/v1/agents/{agentId}", http.MethodPatch, agentPatch)

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

	agentRetire, err := b.jsonOperation(
		"retireAgent",
		"Retire an agent definition",
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
	agentRetire.AddParameter(uuidPathParameter("agentId", "Agent ID."))
	b.doc.AddOperation("/api/v1/agents/{agentId}/retire", http.MethodPost, agentRetire)

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

	projectUpdatesGet, err := b.jsonOperation(
		"listProjectUpdates",
		"List curated project update threads",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectUpdateThreadsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectUpdatesGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/updates", http.MethodGet, projectUpdatesGet)

	projectUpdatesPost, err := b.jsonOperation(
		"createProjectUpdateThread",
		"Create a curated project update thread",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPIProjectUpdateThreadResponse{},
		OpenAPICreateProjectUpdateThreadRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectUpdatesPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/updates", http.MethodPost, projectUpdatesPost)

	projectUpdatePatch, err := b.jsonOperation(
		"updateProjectUpdateThread",
		"Update a curated project update thread",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectUpdateThreadResponse{},
		OpenAPIUpdateProjectUpdateThreadRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectUpdatePatch.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectUpdatePatch.AddParameter(uuidPathParameter("threadId", "Project update thread ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/updates/{threadId}", http.MethodPatch, projectUpdatePatch)

	projectUpdateDelete, err := b.jsonOperation(
		"deleteProjectUpdateThread",
		"Soft-delete a curated project update thread",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectUpdateThreadDeleteResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectUpdateDelete.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectUpdateDelete.AddParameter(uuidPathParameter("threadId", "Project update thread ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/updates/{threadId}", http.MethodDelete, projectUpdateDelete)

	projectUpdateRevisionsGet, err := b.jsonOperation(
		"listProjectUpdateThreadRevisions",
		"List revision history for a project update thread",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectUpdateThreadRevisionsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectUpdateRevisionsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectUpdateRevisionsGet.AddParameter(uuidPathParameter("threadId", "Project update thread ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/updates/{threadId}/revisions", http.MethodGet, projectUpdateRevisionsGet)

	projectUpdateCommentPost, err := b.jsonOperation(
		"createProjectUpdateComment",
		"Create a comment on a project update thread",
		[]string{"catalog"},
		http.StatusCreated,
		OpenAPIProjectUpdateCommentResponse{},
		OpenAPICreateProjectUpdateCommentRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectUpdateCommentPost.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectUpdateCommentPost.AddParameter(uuidPathParameter("threadId", "Project update thread ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/updates/{threadId}/comments", http.MethodPost, projectUpdateCommentPost)

	projectUpdateCommentPatch, err := b.jsonOperation(
		"updateProjectUpdateComment",
		"Update a comment on a project update thread",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectUpdateCommentResponse{},
		OpenAPIUpdateProjectUpdateCommentRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectUpdateCommentPatch.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectUpdateCommentPatch.AddParameter(uuidPathParameter("threadId", "Project update thread ID."))
	projectUpdateCommentPatch.AddParameter(uuidPathParameter("commentId", "Project update comment ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/updates/{threadId}/comments/{commentId}", http.MethodPatch, projectUpdateCommentPatch)

	projectUpdateCommentDelete, err := b.jsonOperation(
		"deleteProjectUpdateComment",
		"Soft-delete a comment on a project update thread",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectUpdateCommentDeleteResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectUpdateCommentDelete.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectUpdateCommentDelete.AddParameter(uuidPathParameter("threadId", "Project update thread ID."))
	projectUpdateCommentDelete.AddParameter(uuidPathParameter("commentId", "Project update comment ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/updates/{threadId}/comments/{commentId}", http.MethodDelete, projectUpdateCommentDelete)

	projectUpdateCommentRevisionsGet, err := b.jsonOperation(
		"listProjectUpdateCommentRevisions",
		"List revision history for a project update comment",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIProjectUpdateCommentRevisionsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectUpdateCommentRevisionsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	projectUpdateCommentRevisionsGet.AddParameter(uuidPathParameter("threadId", "Project update thread ID."))
	projectUpdateCommentRevisionsGet.AddParameter(uuidPathParameter("commentId", "Project update comment ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/updates/{threadId}/comments/{commentId}/revisions", http.MethodGet, projectUpdateCommentRevisionsGet)

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

	roleGet, err := b.jsonOperation(
		"getBuiltinRole",
		"Get a builtin workflow role template",
		[]string{"catalog"},
		http.StatusOK,
		OpenAPIRoleResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	roleGet.AddParameter(
		openapi3.NewPathParameter("roleSlug").
			WithDescription("Builtin role slug.").
			WithRequired(true).
			WithSchema(openapi3.NewStringSchema()),
	)
	b.doc.AddOperation("/api/v1/roles/builtin/{roleSlug}", http.MethodGet, roleGet)

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

	workflowImpact, err := b.jsonOperation(
		"getWorkflowImpact",
		"Get workflow delete impact analysis",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIWorkflowImpactResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	workflowImpact.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}/impact", http.MethodGet, workflowImpact)

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

	workflowRetire, err := b.jsonOperation(
		"retireWorkflow",
		"Retire a workflow by deactivating it",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIWorkflowResponse{},
		OpenAPIRetireWorkflowRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	workflowRetire.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}/retire", http.MethodPost, workflowRetire)

	workflowReplaceReferences, err := b.jsonOperation(
		"replaceWorkflowReferences",
		"Replace workflow references in scheduled jobs and active tickets",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIWorkflowReplaceReferencesResponse{},
		OpenAPIReplaceWorkflowReferencesRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	workflowReplaceReferences.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}/replace-references", http.MethodPost, workflowReplaceReferences)

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

	harnessHistoryGet, err := b.jsonOperation(
		"getWorkflowHarnessHistory",
		"List published workflow harness versions",
		[]string{"workflows"},
		http.StatusOK,
		OpenAPIWorkflowHistoryResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	harnessHistoryGet.AddParameter(uuidPathParameter("workflowId", "Workflow ID."))
	b.doc.AddOperation("/api/v1/workflows/{workflowId}/harness/history", http.MethodGet, harnessHistoryGet)

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

	skillFilesGet, err := b.jsonOperation(
		"getSkillFiles",
		"Get the current published skill bundle files",
		[]string{"skills"},
		http.StatusOK,
		OpenAPISkillFilesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	skillFilesGet.AddParameter(uuidPathParameter("skillId", "Skill ID."))
	b.doc.AddOperation("/api/v1/skills/{skillId}/files", http.MethodGet, skillFilesGet)

	skillHistoryGet, err := b.jsonOperation(
		"getSkillHistory",
		"List published skill versions",
		[]string{"skills"},
		http.StatusOK,
		OpenAPISkillHistoryResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	skillHistoryGet.AddParameter(uuidPathParameter("skillId", "Skill ID."))
	b.doc.AddOperation("/api/v1/skills/{skillId}/history", http.MethodGet, skillHistoryGet)

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

	refinementRunStart, err := b.streamOperation(
		"startSkillRefinement",
		"Start a Codex-backed skill fix-and-verify refinement run",
		[]string{"skills"},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	refinementBodyRef, err := b.schemaRef(OpenAPISkillRefinementRequest{})
	if err != nil {
		return err
	}
	refinementRunStart.RequestBody = &openapi3.RequestBodyRef{
		Value: openapi3.NewRequestBody().
			WithDescription("Skill fix-and-verify refinement request body.").
			WithJSONSchemaRef(refinementBodyRef).
			WithRequired(true),
	}
	refinementRunStart.AddParameter(uuidPathParameter("skillId", "Skill ID."))
	b.doc.AddOperation("/api/v1/skills/{skillId}/refinement-runs", http.MethodPost, refinementRunStart)

	refinementRunDelete := openapi3.NewOperation()
	refinementRunDelete.OperationID = "closeSkillRefinementRun"
	refinementRunDelete.Summary = "Close a skill refinement run and clean up its temporary workspace"
	refinementRunDelete.Tags = []string{"skills"}
	refinementRunDelete.Responses = openapi3.NewResponsesWithCapacity(3)
	refinementRunDelete.AddResponse(http.StatusNoContent, openapi3.NewResponse().WithDescription("Skill refinement run closed."))
	for _, code := range []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError} {
		errorResponse, err := b.errorResponse(code)
		if err != nil {
			return err
		}
		refinementRunDelete.AddResponse(code, errorResponse)
	}
	refinementRunDelete.AddParameter(openapi3.NewPathParameter("sessionId").
		WithDescription("Skill refinement session ID.").
		WithRequired(true).
		WithSchema(openapi3.NewStringSchema()),
	)
	b.doc.AddOperation("/api/v1/skills/refinement-runs/{sessionId}", http.MethodDelete, refinementRunDelete)

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

	archivedTicketsGet, err := b.jsonOperation(
		"listArchivedTickets",
		"List archived tickets",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPIArchivedTicketsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	archivedTicketsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	archivedTicketsGet.AddParameter(intQueryParameter("page", "Archived tickets page number."))
	archivedTicketsGet.AddParameter(intQueryParameter("per_page", "Archived tickets page size."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/tickets/archived", http.MethodGet, archivedTicketsGet)

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

	ticketRetryResume, err := b.jsonOperation(
		"resumeTicketRetry",
		"Resume a stalled ticket retry",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	ticketRetryResume.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}/retry/resume", http.MethodPost, ticketRetryResume)

	ticketWorkspaceReset, err := b.jsonOperation(
		"resetTicketWorkspace",
		"Reset a preserved ticket workspace",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketWorkspaceResetResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	ticketWorkspaceReset.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/tickets/{ticketId}/workspace/reset", http.MethodPost, ticketWorkspaceReset)

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

	ticketRunsGet, err := b.jsonOperation(
		"listTicketRuns",
		"List ticket runs",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketRunsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	ticketRunsGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	ticketRunsGet.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/tickets/{ticketId}/runs", http.MethodGet, ticketRunsGet)

	ticketRunGet, err := b.jsonOperation(
		"getTicketRun",
		"Get ticket run transcript data",
		[]string{"tickets"},
		http.StatusOK,
		OpenAPITicketRunResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	ticketRunGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	ticketRunGet.AddParameter(uuidPathParameter("ticketId", "Ticket ID."))
	ticketRunGet.AddParameter(uuidPathParameter("runId", "Run ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/tickets/{ticketId}/runs/{runId}", http.MethodGet, ticketRunGet)

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
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	securityGet.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/security-settings", http.MethodGet, securityGet)

	securityPut, err := b.jsonOperation(
		"saveGitHubOutboundCredential",
		"Save a platform-managed GitHub outbound credential",
		[]string{"security-settings"},
		http.StatusOK,
		OpenAPISecuritySettingsResponse{},
		OpenAPISaveGitHubOutboundCredentialRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	securityPut.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/security-settings/github-outbound-credential", http.MethodPut, securityPut)

	securityImport, err := b.jsonOperation(
		"importGitHubOutboundCredentialFromGHCLI",
		"Import the current gh auth token into platform-managed GitHub credential storage",
		[]string{"security-settings"},
		http.StatusOK,
		OpenAPISecuritySettingsResponse{},
		OpenAPIGitHubCredentialScopeRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	securityImport.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/security-settings/github-outbound-credential/import-gh-cli", http.MethodPost, securityImport)

	securityRetest, err := b.jsonOperation(
		"retestGitHubOutboundCredential",
		"Retest a stored platform-managed GitHub outbound credential",
		[]string{"security-settings"},
		http.StatusOK,
		OpenAPISecuritySettingsResponse{},
		OpenAPIGitHubCredentialScopeRequest{},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	securityRetest.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/security-settings/github-outbound-credential/retest", http.MethodPost, securityRetest)

	securityDelete, err := b.jsonOperation(
		"deleteGitHubOutboundCredential",
		"Delete a stored platform-managed GitHub outbound credential",
		[]string{"security-settings"},
		http.StatusOK,
		OpenAPISecuritySettingsResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	securityDelete.AddParameter(uuidPathParameter("projectId", "Project ID."))
	securityDelete.AddParameter(openapi3.NewQueryParameter("scope").
		WithDescription("Credential scope to delete. Supported values are organization and project.").
		WithRequired(true).
		WithSchema(openapi3.NewStringSchema()))
	b.doc.AddOperation("/api/v1/projects/{projectId}/security-settings/github-outbound-credential", http.MethodDelete, securityDelete)

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

	projectConversationCreate, err := b.jsonOperation(
		"createProjectConversation",
		"Create a project conversation",
		[]string{"chat"},
		http.StatusCreated,
		OpenAPIProjectConversationResponse{},
		OpenAPIProjectConversationCreateRequest{},
		http.StatusBadRequest,
		http.StatusConflict,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	b.doc.AddOperation("/api/v1/chat/conversations", http.MethodPost, projectConversationCreate)

	projectConversationList, err := b.jsonOperation(
		"listProjectConversations",
		"List project conversations",
		[]string{"chat"},
		http.StatusOK,
		OpenAPIProjectConversationListResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectConversationList.AddParameter(uuidQueryParameter("project_id", "Project ID that scopes the conversation list."))
	projectConversationList.AddParameter(openapi3.NewQueryParameter("provider_id").
		WithDescription("Optional provider ID used to filter conversations.").
		WithSchema(openapi3.NewStringSchema().WithFormat("uuid")),
	)
	b.doc.AddOperation("/api/v1/chat/conversations", http.MethodGet, projectConversationList)

	projectConversationGet, err := b.jsonOperation(
		"getProjectConversation",
		"Get a project conversation",
		[]string{"chat"},
		http.StatusOK,
		OpenAPIProjectConversationResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectConversationGet.AddParameter(uuidPathParameter("conversationId", "Stable OpenASE conversation ID."))
	b.doc.AddOperation("/api/v1/chat/conversations/{conversationId}", http.MethodGet, projectConversationGet)

	projectConversationEntries, err := b.jsonOperation(
		"listProjectConversationEntries",
		"List project conversation transcript entries",
		[]string{"chat"},
		http.StatusOK,
		OpenAPIProjectConversationEntriesResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectConversationEntries.AddParameter(uuidPathParameter("conversationId", "Stable OpenASE conversation ID."))
	b.doc.AddOperation("/api/v1/chat/conversations/{conversationId}/entries", http.MethodGet, projectConversationEntries)

	projectConversationWorkspaceDiff, err := b.jsonOperation(
		"getProjectConversationWorkspaceDiff",
		"Get project conversation workspace diff summary",
		[]string{"chat"},
		http.StatusOK,
		OpenAPIProjectConversationWorkspaceDiffResponse{},
		nil,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectConversationWorkspaceDiff.AddParameter(uuidPathParameter("conversationId", "Stable OpenASE conversation ID."))
	b.doc.AddOperation("/api/v1/chat/conversations/{conversationId}/workspace-diff", http.MethodGet, projectConversationWorkspaceDiff)

	projectConversationTurn, err := b.jsonOperation(
		"startProjectConversationTurn",
		"Start a project conversation turn",
		[]string{"chat"},
		http.StatusAccepted,
		OpenAPIProjectConversationTurnResponse{},
		OpenAPIProjectConversationTurnRequest{},
		http.StatusBadRequest,
		http.StatusConflict,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectConversationTurn.AddParameter(uuidPathParameter("conversationId", "Stable OpenASE conversation ID."))
	b.doc.AddOperation("/api/v1/chat/conversations/{conversationId}/turns", http.MethodPost, projectConversationTurn)

	projectConversationMuxStream, err := b.streamOperation(
		"streamProjectConversationsMux",
		"Watch multiplexed project conversation events for one project",
		[]string{"chat"},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectConversationMuxStream.AddParameter(uuidPathParameter("projectId", "Stable OpenASE project ID."))
	b.doc.AddOperation("/api/v1/chat/projects/{projectId}/conversations/stream", http.MethodGet, projectConversationMuxStream)

	projectConversationStream, err := b.streamOperation(
		"streamProjectConversation",
		"Watch project conversation events",
		[]string{"chat"},
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectConversationStream.AddParameter(uuidPathParameter("conversationId", "Stable OpenASE conversation ID."))
	b.doc.AddOperation("/api/v1/chat/conversations/{conversationId}/stream", http.MethodGet, projectConversationStream)

	projectConversationInterrupt, err := b.jsonOperation(
		"respondProjectConversationInterrupt",
		"Respond to a project conversation interrupt",
		[]string{"chat"},
		http.StatusOK,
		OpenAPIProjectConversationInterruptEnvelope{},
		OpenAPIProjectConversationInterruptResponseRequest{},
		http.StatusBadRequest,
		http.StatusConflict,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectConversationInterrupt.AddParameter(uuidPathParameter("conversationId", "Stable OpenASE conversation ID."))
	projectConversationInterrupt.AddParameter(uuidPathParameter("interruptId", "Stable OpenASE interrupt ID."))
	b.doc.AddOperation("/api/v1/chat/conversations/{conversationId}/interrupts/{interruptId}/respond", http.MethodPost, projectConversationInterrupt)

	projectConversationRuntimeDelete := openapi3.NewOperation()
	projectConversationRuntimeDelete.OperationID = "closeProjectConversationRuntime"
	projectConversationRuntimeDelete.Summary = "Close a project conversation live runtime"
	projectConversationRuntimeDelete.Tags = []string{"chat"}
	projectConversationRuntimeDelete.Responses = openapi3.NewResponsesWithCapacity(6)
	projectConversationRuntimeDelete.AddResponse(http.StatusNoContent, openapi3.NewResponse().WithDescription("Project conversation live runtime closed."))
	for _, code := range []int{
		http.StatusBadRequest,
		http.StatusConflict,
		http.StatusNotFound,
		http.StatusServiceUnavailable,
		http.StatusInternalServerError,
	} {
		errorResponse, err := b.errorResponse(code)
		if err != nil {
			return err
		}
		projectConversationRuntimeDelete.AddResponse(code, errorResponse)
	}
	projectConversationRuntimeDelete.AddParameter(uuidPathParameter("conversationId", "Stable OpenASE conversation ID."))
	b.doc.AddOperation("/api/v1/chat/conversations/{conversationId}/runtime", http.MethodDelete, projectConversationRuntimeDelete)

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

	projectEventStream, err := b.streamOperation(
		"streamProjectEvents",
		"Stream the canonical passive project event bus",
		[]string{"streams"},
		http.StatusBadRequest,
		http.StatusInternalServerError,
	)
	if err != nil {
		return err
	}
	projectEventStream.AddParameter(uuidPathParameter("projectId", "Project ID."))
	b.doc.AddOperation("/api/v1/projects/{projectId}/events/stream", http.MethodGet, projectEventStream)

	orgStreams := []struct {
		path        string
		operationID string
		summary     string
	}{
		{path: "/api/v1/orgs/{orgId}/machines/stream", operationID: "streamOrganizationMachines", summary: "Stream organization machine events"},
		{path: "/api/v1/orgs/{orgId}/providers/stream", operationID: "streamOrganizationProviders", summary: "Stream organization provider events"},
	}
	for _, item := range orgStreams {
		op, err := b.streamOperation(item.operationID, item.summary, []string{"streams"}, http.StatusBadRequest, http.StatusInternalServerError)
		if err != nil {
			return err
		}
		op.AddParameter(uuidPathParameter("orgId", "Organization ID."))
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
	if schema.Items != nil {
		itemPrefix := prefix + "[]"
		if prefix == "" {
			itemPrefix = "[]"
		}
		applyRequestFieldDescriptions(schema.Items, itemPrefix, descriptions)
	}
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
