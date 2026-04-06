package ticket

import (
	"errors"
	"fmt"
	"strings"
	"time"

	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

var (
	ErrProjectNotFound       = errors.New("project not found")
	ErrProjectRepoNotFound   = errors.New("project repo not found")
	ErrRepoScopeRequired     = errors.New("explicit repo scope is required when a project has multiple repos")
	ErrTicketNotFound        = errors.New("ticket not found")
	ErrTicketConflict        = errors.New("ticket identifier already exists in project")
	ErrStatusNotFound        = errors.New("ticket status not found")
	ErrWorkflowNotFound      = errors.New("workflow not found")
	ErrStatusNotAllowed      = errors.New("ticket status is not allowed by the workflow finish set")
	ErrParentTicketNotFound  = errors.New("parent ticket not found")
	ErrTargetMachineNotFound = errors.New("target machine not found in project organization")
	ErrDependencyNotFound    = errors.New("ticket dependency not found")
	ErrDependencyConflict    = errors.New("ticket dependency already exists")
	ErrCommentNotFound       = errors.New("ticket comment not found")
	ErrExternalLinkNotFound  = errors.New("ticket external link not found")
	ErrExternalLinkConflict  = errors.New("ticket external link already exists")
	ErrInvalidDependency     = errors.New("invalid ticket dependency")
	ErrRetryResumeConflict   = errors.New("ticket retry is not paused for repeated stalls")
)

type Optional[T any] struct {
	Set   bool
	Value T
}

type Priority string

const (
	DefaultPriority Priority = PriorityMedium
	PriorityUrgent  Priority = "urgent"
	PriorityHigh    Priority = "high"
	PriorityMedium  Priority = "medium"
	PriorityLow     Priority = "low"
)

func ParsePriority(raw string) (Priority, error) {
	priority := Priority(strings.ToLower(strings.TrimSpace(raw)))
	switch priority {
	case PriorityUrgent, PriorityHigh, PriorityMedium, PriorityLow:
		return priority, nil
	default:
		return "", fmt.Errorf("priority must be one of urgent, high, medium, low")
	}
}

func (p Priority) String() string { return string(p) }

type Type string

const (
	DefaultType  Type = TypeFeature
	TypeFeature  Type = "feature"
	TypeBugfix   Type = "bugfix"
	TypeRefactor Type = "refactor"
	TypeChore    Type = "chore"
	TypeEpic     Type = "epic"
)

func ParseType(raw string) (Type, error) {
	ticketType := Type(strings.ToLower(strings.TrimSpace(raw)))
	switch ticketType {
	case TypeFeature, TypeBugfix, TypeRefactor, TypeChore, TypeEpic:
		return ticketType, nil
	default:
		return "", fmt.Errorf("type must be one of feature, bugfix, refactor, chore, epic")
	}
}

func (t Type) String() string { return string(t) }

type DependencyType string

const (
	DefaultDependencyType  DependencyType = DependencyTypeBlocks
	DependencyTypeBlocks   DependencyType = "blocks"
	DependencyTypeSubIssue DependencyType = "sub-issue"
)

func ParseDependencyType(raw string) (DependencyType, error) {
	dependencyType := DependencyType(strings.ToLower(strings.TrimSpace(raw)))
	switch dependencyType {
	case DependencyTypeBlocks, DependencyTypeSubIssue:
		return dependencyType, nil
	default:
		return "", fmt.Errorf("type must be one of blocks, blocked_by, sub_issue")
	}
}

func (t DependencyType) String() string { return string(t) }

type ExternalLinkType string

const (
	ExternalLinkTypeGithubIssue ExternalLinkType = "github_issue"
	ExternalLinkTypeGitlabIssue ExternalLinkType = "gitlab_issue"
	ExternalLinkTypeJiraTicket  ExternalLinkType = "jira_ticket"
	ExternalLinkTypeGithubPR    ExternalLinkType = "github_pr"
	ExternalLinkTypeGitlabMR    ExternalLinkType = "gitlab_mr"
	ExternalLinkTypeCustom      ExternalLinkType = "custom"
)

func ParseExternalLinkType(raw string) (ExternalLinkType, error) {
	linkType := ExternalLinkType(strings.ToLower(strings.TrimSpace(raw)))
	switch linkType {
	case ExternalLinkTypeGithubIssue, ExternalLinkTypeGitlabIssue, ExternalLinkTypeJiraTicket, ExternalLinkTypeGithubPR, ExternalLinkTypeGitlabMR, ExternalLinkTypeCustom:
		return linkType, nil
	default:
		return "", fmt.Errorf("type must be one of github_issue, gitlab_issue, jira_ticket, github_pr, gitlab_mr, custom")
	}
}

func (t ExternalLinkType) String() string { return string(t) }

type ExternalLinkRelation string

const (
	DefaultExternalLinkRelation  ExternalLinkRelation = ExternalLinkRelationRelated
	ExternalLinkRelationResolves ExternalLinkRelation = "resolves"
	ExternalLinkRelationRelated  ExternalLinkRelation = "related"
	ExternalLinkRelationCausedBy ExternalLinkRelation = "caused_by"
)

func ParseExternalLinkRelation(raw string) (ExternalLinkRelation, error) {
	relation := ExternalLinkRelation(strings.ToLower(strings.TrimSpace(raw)))
	switch relation {
	case ExternalLinkRelationResolves, ExternalLinkRelationRelated, ExternalLinkRelationCausedBy:
		return relation, nil
	default:
		return "", fmt.Errorf("relation must be one of resolves, related, caused_by")
	}
}

func (r ExternalLinkRelation) String() string { return string(r) }

type TicketReference struct {
	ID         uuid.UUID `json:"id"`
	Identifier string    `json:"identifier"`
	Title      string    `json:"title"`
	StatusID   uuid.UUID `json:"status_id"`
	StatusName string    `json:"status_name"`
}

type Dependency struct {
	ID     uuid.UUID       `json:"id"`
	Type   DependencyType  `json:"type"`
	Target TicketReference `json:"target"`
}

type ExternalLink struct {
	ID         uuid.UUID            `json:"id"`
	LinkType   ExternalLinkType     `json:"link_type"`
	URL        string               `json:"url"`
	ExternalID string               `json:"external_id"`
	Title      string               `json:"title,omitempty"`
	Status     string               `json:"status,omitempty"`
	Relation   ExternalLinkRelation `json:"relation"`
	CreatedAt  time.Time            `json:"created_at"`
}

type Comment struct {
	ID           uuid.UUID  `json:"id"`
	TicketID     uuid.UUID  `json:"ticket_id"`
	BodyMarkdown string     `json:"body_markdown"`
	CreatedBy    string     `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	EditedAt     *time.Time `json:"edited_at,omitempty"`
	EditCount    int        `json:"edit_count"`
	LastEditedBy *string    `json:"last_edited_by,omitempty"`
	IsDeleted    bool       `json:"is_deleted"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	DeletedBy    *string    `json:"deleted_by,omitempty"`
}

type CommentRevision struct {
	ID             uuid.UUID `json:"id"`
	CommentID      uuid.UUID `json:"comment_id"`
	RevisionNumber int       `json:"revision_number"`
	BodyMarkdown   string    `json:"body_markdown"`
	EditedBy       string    `json:"edited_by"`
	EditedAt       time.Time `json:"edited_at"`
	EditReason     *string   `json:"edit_reason,omitempty"`
}

type Ticket struct {
	ID                   uuid.UUID         `json:"id"`
	ProjectID            uuid.UUID         `json:"project_id"`
	Identifier           string            `json:"identifier"`
	Title                string            `json:"title"`
	Description          string            `json:"description"`
	StatusID             uuid.UUID         `json:"status_id"`
	StatusName           string            `json:"status_name"`
	Archived             bool              `json:"archived"`
	Priority             Priority          `json:"priority"`
	Type                 Type              `json:"type"`
	WorkflowID           *uuid.UUID        `json:"workflow_id,omitempty"`
	CurrentRunID         *uuid.UUID        `json:"current_run_id,omitempty"`
	TargetMachineID      *uuid.UUID        `json:"target_machine_id,omitempty"`
	CreatedBy            string            `json:"created_by"`
	Parent               *TicketReference  `json:"parent,omitempty"`
	Children             []TicketReference `json:"children"`
	Dependencies         []Dependency      `json:"dependencies"`
	IncomingDependencies []Dependency      `json:"incoming_dependencies"`
	ExternalLinks        []ExternalLink    `json:"external_links"`
	ExternalRef          string            `json:"external_ref"`
	BudgetUSD            float64           `json:"budget_usd"`
	CostTokensInput      int64             `json:"cost_tokens_input"`
	CostTokensOutput     int64             `json:"cost_tokens_output"`
	CostAmount           float64           `json:"cost_amount"`
	AttemptCount         int               `json:"attempt_count"`
	ConsecutiveErrors    int               `json:"consecutive_errors"`
	StartedAt            *time.Time        `json:"started_at,omitempty"`
	CompletedAt          *time.Time        `json:"completed_at,omitempty"`
	NextRetryAt          *time.Time        `json:"next_retry_at,omitempty"`
	RetryPaused          bool              `json:"retry_paused"`
	PauseReason          string            `json:"pause_reason,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
}

type ListInput struct {
	ProjectID   uuid.UUID
	StatusNames []string
	Priorities  []Priority
	Limit       int
}

type CreateInput struct {
	ProjectID       uuid.UUID
	Title           string
	Description     string
	StatusID        *uuid.UUID
	Archived        bool
	Priority        *Priority
	Type            Type
	WorkflowID      *uuid.UUID
	TargetMachineID *uuid.UUID
	CreatedBy       string
	ParentTicketID  *uuid.UUID
	ExternalRef     string
	BudgetUSD       float64
	RepoScopes      []CreateRepoScopeInput
}

type CreateRepoScopeInput struct {
	RepoID     uuid.UUID
	BranchName *string
}

type UpdateInput struct {
	TicketID                          uuid.UUID
	Title                             Optional[string]
	Description                       Optional[string]
	StatusID                          Optional[uuid.UUID]
	Archived                          Optional[bool]
	Priority                          Optional[*Priority]
	Type                              Optional[Type]
	WorkflowID                        Optional[*uuid.UUID]
	TargetMachineID                   Optional[*uuid.UUID]
	CreatedBy                         Optional[string]
	ParentTicketID                    Optional[*uuid.UUID]
	ExternalRef                       Optional[string]
	BudgetUSD                         Optional[float64]
	RestrictStatusToWorkflowFinishSet bool
}

type DeferredLifecycleHook struct {
	RunID      uuid.UUID
	WorkflowID *uuid.UUID
	HookName   string
}

type UpdateResult struct {
	Ticket       Ticket
	DeferredHook *DeferredLifecycleHook
}

type AddDependencyInput struct {
	TicketID       uuid.UUID
	TargetTicketID uuid.UUID
	Type           DependencyType
}

type AddExternalLinkInput struct {
	TicketID   uuid.UUID
	LinkType   ExternalLinkType
	URL        string
	ExternalID string
	Title      string
	Status     string
	Relation   ExternalLinkRelation
}

type ResumeRetryInput struct {
	TicketID uuid.UUID
}

type DeleteDependencyResult struct {
	DeletedDependencyID uuid.UUID `json:"deleted_dependency_id"`
}

type DeleteExternalLinkResult struct {
	DeletedExternalLinkID uuid.UUID `json:"deleted_external_link_id"`
}

type AddCommentInput struct {
	TicketID  uuid.UUID
	Body      string
	CreatedBy string
}

type UpdateCommentInput struct {
	TicketID   uuid.UUID
	CommentID  uuid.UUID
	Body       string
	EditedBy   string
	EditReason string
}

type DeleteCommentResult struct {
	DeletedCommentID uuid.UUID `json:"deleted_comment_id"`
}

type RecordActivityEventInput struct {
	ProjectID uuid.UUID
	TicketID  *uuid.UUID
	AgentID   *uuid.UUID
	EventType activityevent.Type
	Message   string
	Metadata  map[string]any
	CreatedAt time.Time
}

type RecordUsageInput struct {
	AgentID  uuid.UUID
	TicketID uuid.UUID
	RunID    *uuid.UUID
	Usage    ticketing.RawUsageDelta
}

type AppliedUsage struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	CostSource   string  `json:"cost_source"`
}

type RecordUsageResult struct {
	Ticket         Ticket       `json:"ticket"`
	Applied        AppliedUsage `json:"applied"`
	BudgetExceeded bool         `json:"budget_exceeded"`
}

type UsageMetricsAgent struct {
	ProviderName string
	ModelName    string
}

type PersistedUsageResult struct {
	Result       RecordUsageResult
	MetricsAgent UsageMetricsAgent
	ProjectID    uuid.UUID
}

type HookWorkspace struct {
	RepoName string
	RepoPath string
}

type LifecycleHookRuntimeData struct {
	TicketID              uuid.UUID
	ProjectID             uuid.UUID
	AgentID               uuid.UUID
	TicketIdentifier      string
	AgentName             string
	WorkflowType          string
	WorkflowFamily        string
	PlatformAccessAllowed []string
	Attempt               int
	WorkspaceRoot         string
	Hooks                 map[string]any
	Machine               catalogdomain.Machine
	Workspaces            []HookWorkspace
}

type PickupDiagnosisState string

const (
	PickupDiagnosisStateRunnable    PickupDiagnosisState = "runnable"
	PickupDiagnosisStateWaiting     PickupDiagnosisState = "waiting"
	PickupDiagnosisStateBlocked     PickupDiagnosisState = "blocked"
	PickupDiagnosisStateRunning     PickupDiagnosisState = "running"
	PickupDiagnosisStateCompleted   PickupDiagnosisState = "completed"
	PickupDiagnosisStateUnavailable PickupDiagnosisState = "unavailable"
)

type PickupDiagnosisReasonCode string

const (
	PickupDiagnosisReasonReadyForPickup            PickupDiagnosisReasonCode = "ready_for_pickup"
	PickupDiagnosisReasonArchived                  PickupDiagnosisReasonCode = "archived"
	PickupDiagnosisReasonCompleted                 PickupDiagnosisReasonCode = "completed"
	PickupDiagnosisReasonRunningCurrentRun         PickupDiagnosisReasonCode = "running_current_run"
	PickupDiagnosisReasonRetryBackoff              PickupDiagnosisReasonCode = "retry_backoff"
	PickupDiagnosisReasonRetryPausedRepeatedStalls PickupDiagnosisReasonCode = "retry_paused_repeated_stalls"
	PickupDiagnosisReasonRetryPausedBudget         PickupDiagnosisReasonCode = "retry_paused_budget"
	PickupDiagnosisReasonRetryPausedInterrupted    PickupDiagnosisReasonCode = "retry_paused_interrupted"
	PickupDiagnosisReasonRetryPausedUser           PickupDiagnosisReasonCode = "retry_paused_user"
	PickupDiagnosisReasonBlockedDependency         PickupDiagnosisReasonCode = "blocked_dependency"
	PickupDiagnosisReasonNoMatchingActiveWorkflow  PickupDiagnosisReasonCode = "no_matching_active_workflow"
	PickupDiagnosisReasonWorkflowInactive          PickupDiagnosisReasonCode = "workflow_inactive"
	PickupDiagnosisReasonWorkflowMissingAgent      PickupDiagnosisReasonCode = "workflow_missing_agent"
	PickupDiagnosisReasonAgentMissing              PickupDiagnosisReasonCode = "agent_missing"
	PickupDiagnosisReasonAgentInterruptRequested   PickupDiagnosisReasonCode = "agent_interrupt_requested"
	PickupDiagnosisReasonAgentPaused               PickupDiagnosisReasonCode = "agent_paused"
	PickupDiagnosisReasonAgentPauseRequested       PickupDiagnosisReasonCode = "agent_pause_requested"
	PickupDiagnosisReasonProviderMissing           PickupDiagnosisReasonCode = "provider_missing"
	PickupDiagnosisReasonMachineMissing            PickupDiagnosisReasonCode = "machine_missing"
	PickupDiagnosisReasonMachineOffline            PickupDiagnosisReasonCode = "machine_offline"
	PickupDiagnosisReasonProviderUnknown           PickupDiagnosisReasonCode = "provider_unknown"
	PickupDiagnosisReasonProviderStale             PickupDiagnosisReasonCode = "provider_stale"
	PickupDiagnosisReasonProviderUnavailable       PickupDiagnosisReasonCode = "provider_unavailable"
	PickupDiagnosisReasonWorkflowConcurrencyFull   PickupDiagnosisReasonCode = "workflow_concurrency_full"
	PickupDiagnosisReasonProjectConcurrencyFull    PickupDiagnosisReasonCode = "project_concurrency_full"
	PickupDiagnosisReasonProviderConcurrencyFull   PickupDiagnosisReasonCode = "provider_concurrency_full"
	PickupDiagnosisReasonStatusCapacityFull        PickupDiagnosisReasonCode = "status_capacity_full"
	PickupDiagnosisReasonSchedulerUnavailable      PickupDiagnosisReasonCode = "scheduler_unavailable"
)

type PickupDiagnosisReasonSeverity string

const (
	PickupDiagnosisReasonSeverityInfo    PickupDiagnosisReasonSeverity = "info"
	PickupDiagnosisReasonSeverityWarning PickupDiagnosisReasonSeverity = "warning"
	PickupDiagnosisReasonSeverityError   PickupDiagnosisReasonSeverity = "error"
)

type PickupDiagnosis struct {
	State                PickupDiagnosisState           `json:"state"`
	PrimaryReasonCode    PickupDiagnosisReasonCode      `json:"primary_reason_code"`
	PrimaryReasonMessage string                         `json:"primary_reason_message"`
	NextActionHint       string                         `json:"next_action_hint,omitempty"`
	Reasons              []PickupDiagnosisReason        `json:"reasons"`
	Workflow             *PickupDiagnosisWorkflow       `json:"workflow,omitempty"`
	Agent                *PickupDiagnosisAgent          `json:"agent,omitempty"`
	Provider             *PickupDiagnosisProvider       `json:"provider,omitempty"`
	Retry                PickupDiagnosisRetry           `json:"retry"`
	Capacity             PickupDiagnosisCapacity        `json:"capacity"`
	BlockedBy            []PickupDiagnosisBlockedTicket `json:"blocked_by"`
}

type PickupDiagnosisReason struct {
	Code     PickupDiagnosisReasonCode     `json:"code"`
	Message  string                        `json:"message"`
	Severity PickupDiagnosisReasonSeverity `json:"severity"`
}

type PickupDiagnosisWorkflow struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	IsActive          bool      `json:"is_active"`
	PickupStatusMatch bool      `json:"pickup_status_match"`
}

type PickupDiagnosisAgent struct {
	ID                  uuid.UUID                              `json:"id"`
	Name                string                                 `json:"name"`
	RuntimeControlState catalogdomain.AgentRuntimeControlState `json:"runtime_control_state"`
}

type PickupDiagnosisProvider struct {
	ID                 uuid.UUID                                    `json:"id"`
	Name               string                                       `json:"name"`
	MachineID          uuid.UUID                                    `json:"machine_id"`
	MachineName        string                                       `json:"machine_name"`
	MachineStatus      catalogdomain.MachineStatus                  `json:"machine_status"`
	AvailabilityState  catalogdomain.AgentProviderAvailabilityState `json:"availability_state"`
	AvailabilityReason *string                                      `json:"availability_reason,omitempty"`
}

type PickupDiagnosisRetry struct {
	AttemptCount int        `json:"attempt_count"`
	RetryPaused  bool       `json:"retry_paused"`
	PauseReason  string     `json:"pause_reason,omitempty"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`
}

type PickupDiagnosisCapacity struct {
	Workflow PickupDiagnosisCapacityBucket `json:"workflow"`
	Project  PickupDiagnosisCapacityBucket `json:"project"`
	Provider PickupDiagnosisCapacityBucket `json:"provider"`
	Status   PickupDiagnosisStatusCapacity `json:"status"`
}

type PickupDiagnosisCapacityBucket struct {
	Limited    bool `json:"limited"`
	ActiveRuns int  `json:"active_runs"`
	Capacity   int  `json:"capacity"`
}

type PickupDiagnosisStatusCapacity struct {
	Limited    bool `json:"limited"`
	ActiveRuns int  `json:"active_runs"`
	Capacity   *int `json:"capacity"`
}

type PickupDiagnosisBlockedTicket struct {
	ID         uuid.UUID `json:"id"`
	Identifier string    `json:"identifier"`
	Title      string    `json:"title"`
	StatusID   uuid.UUID `json:"status_id"`
	StatusName string    `json:"status_name"`
}
