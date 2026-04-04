package workflow

import (
	"time"

	"github.com/google/uuid"
)

type Optional[T any] struct {
	Set   bool
	Value T
}

type Workflow struct {
	ID                    uuid.UUID      `json:"id"`
	ProjectID             uuid.UUID      `json:"project_id"`
	AgentID               *uuid.UUID     `json:"agent_id"`
	Name                  string         `json:"name"`
	Type                  Type           `json:"type"`
	RoleSlug              string         `json:"role_slug"`
	RoleName              string         `json:"role_name"`
	RoleDescription       string         `json:"role_description"`
	PlatformAccessAllowed []string       `json:"platform_access_allowed"`
	HarnessPath           string         `json:"harness_path"`
	Hooks                 map[string]any `json:"hooks"`
	MaxConcurrent         int            `json:"max_concurrent"`
	MaxRetryAttempts      int            `json:"max_retry_attempts"`
	TimeoutMinutes        int            `json:"timeout_minutes"`
	StallTimeoutMinutes   int            `json:"stall_timeout_minutes"`
	Version               int            `json:"version"`
	IsActive              bool           `json:"is_active"`
	PickupStatusIDs       []uuid.UUID    `json:"pickup_status_ids"`
	FinishStatusIDs       []uuid.UUID    `json:"finish_status_ids"`
}

type WorkflowDetail struct {
	Workflow
	HarnessContent string `json:"harness_content"`
}

type WorkflowTicketReference struct {
	ID           uuid.UUID  `json:"id"`
	Identifier   string     `json:"identifier"`
	Title        string     `json:"title"`
	StatusID     uuid.UUID  `json:"status_id"`
	StatusName   string     `json:"status_name"`
	CurrentRunID *uuid.UUID `json:"current_run_id,omitempty"`
}

type WorkflowScheduledJobReference struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	IsEnabled bool      `json:"is_enabled"`
}

type WorkflowAgentRunReference struct {
	ID               uuid.UUID `json:"id"`
	TicketID         uuid.UUID `json:"ticket_id"`
	TicketIdentifier string    `json:"ticket_identifier"`
	TicketTitle      string    `json:"ticket_title"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

type WorkflowReplaceableReferences struct {
	Tickets       []WorkflowTicketReference       `json:"tickets"`
	ScheduledJobs []WorkflowScheduledJobReference `json:"scheduled_jobs"`
}

type WorkflowBlockingReferences struct {
	ActiveAgentRuns     []WorkflowAgentRunReference `json:"active_agent_runs"`
	HistoricalAgentRuns []WorkflowAgentRunReference `json:"historical_agent_runs"`
}

type WorkflowImpactSummary struct {
	TicketCount               int `json:"ticket_count"`
	ScheduledJobCount         int `json:"scheduled_job_count"`
	ActiveAgentRunCount       int `json:"active_agent_run_count"`
	HistoricalAgentRunCount   int `json:"historical_agent_run_count"`
	ReplaceableReferenceCount int `json:"replaceable_reference_count"`
	BlockingReferenceCount    int `json:"blocking_reference_count"`
}

type WorkflowImpactAnalysis struct {
	WorkflowID            uuid.UUID                     `json:"workflow_id"`
	CanRetire             bool                          `json:"can_retire"`
	CanReplaceReferences  bool                          `json:"can_replace_references"`
	CanPurge              bool                          `json:"can_purge"`
	Summary               WorkflowImpactSummary         `json:"summary"`
	ReplaceableReferences WorkflowReplaceableReferences `json:"replaceable_references"`
	BlockingReferences    WorkflowBlockingReferences    `json:"blocking_references"`
}

type ReplaceWorkflowReferencesInput struct {
	WorkflowID            uuid.UUID
	ReplacementWorkflowID uuid.UUID
}

type ReplaceWorkflowReferencesResult struct {
	WorkflowID            uuid.UUID                       `json:"workflow_id"`
	ReplacementWorkflowID uuid.UUID                       `json:"replacement_workflow_id"`
	TicketCount           int                             `json:"ticket_count"`
	ScheduledJobCount     int                             `json:"scheduled_job_count"`
	Tickets               []WorkflowTicketReference       `json:"tickets"`
	ScheduledJobs         []WorkflowScheduledJobReference `json:"scheduled_jobs"`
}

type WorkflowImpactConflict struct {
	Err    error
	Impact WorkflowImpactAnalysis
}

func (e *WorkflowImpactConflict) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *WorkflowImpactConflict) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type HarnessDocument struct {
	WorkflowID uuid.UUID `json:"workflow_id"`
	Path       string    `json:"path"`
	Content    string    `json:"content"`
	Version    int       `json:"version"`
}

type CreateInput struct {
	ProjectID             uuid.UUID
	AgentID               uuid.UUID
	Name                  string
	Type                  Type
	RoleSlug              string
	RoleName              string
	RoleDescription       string
	PlatformAccessAllowed []string
	SkillNames            []string
	CreatedBy             string
	HarnessPath           *string
	HarnessContent        string
	Hooks                 map[string]any
	MaxConcurrent         int
	MaxRetryAttempts      int
	TimeoutMinutes        int
	StallTimeoutMinutes   int
	IsActive              bool
	PickupStatusIDs       StatusBindingSet
	FinishStatusIDs       StatusBindingSet
}

type UpdateInput struct {
	WorkflowID            uuid.UUID
	AgentID               Optional[uuid.UUID]
	Name                  Optional[string]
	Type                  Optional[Type]
	RoleName              Optional[string]
	RoleDescription       Optional[string]
	PlatformAccessAllowed Optional[[]string]
	EditedBy              string
	HarnessPath           Optional[string]
	Hooks                 Optional[map[string]any]
	MaxConcurrent         Optional[int]
	MaxRetryAttempts      Optional[int]
	TimeoutMinutes        Optional[int]
	StallTimeoutMinutes   Optional[int]
	IsActive              Optional[bool]
	PickupStatusIDs       Optional[StatusBindingSet]
	FinishStatusIDs       Optional[StatusBindingSet]
}

type UpdateHarnessInput struct {
	WorkflowID uuid.UUID
	Content    string
	EditedBy   string
}

type VersionSummary struct {
	ID        uuid.UUID `json:"id"`
	Version   int       `json:"version"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type WorkflowVersionRecord struct {
	ID                    uuid.UUID
	WorkflowID            uuid.UUID
	Version               int
	ContentMarkdown       string
	Name                  string
	Type                  Type
	RoleSlug              string
	RoleName              string
	RoleDescription       string
	PickupStatusIDs       []uuid.UUID
	FinishStatusIDs       []uuid.UUID
	HarnessPath           string
	Hooks                 map[string]any
	PlatformAccessAllowed []string
	MaxConcurrent         int
	MaxRetryAttempts      int
	TimeoutMinutes        int
	StallTimeoutMinutes   int
	IsActive              bool
	ContentHash           string
	CreatedBy             string
	CreatedAt             time.Time
}

type SkillRecord struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	Name             string
	Description      string
	CurrentVersionID *uuid.UUID
	IsBuiltin        bool
	IsEnabled        bool
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ArchivedAt       *time.Time
}

type SkillVersionRecord struct {
	ID              uuid.UUID
	SkillID         uuid.UUID
	Version         int
	CreatedBy       string
	CreatedAt       time.Time
	ContentMarkdown string
	BundleHash      string
	FileCount       int
}

type SkillWorkflowBinding struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	HarnessPath string    `json:"harness_path"`
}

type Skill struct {
	ID             uuid.UUID              `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Path           string                 `json:"path"`
	CurrentVersion int                    `json:"current_version"`
	IsBuiltin      bool                   `json:"is_builtin"`
	IsEnabled      bool                   `json:"is_enabled"`
	CreatedBy      string                 `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
	BoundWorkflows []SkillWorkflowBinding `json:"bound_workflows"`
}

type SkillBundleFileInput struct {
	Path         string
	Content      []byte
	IsExecutable bool
	MediaType    string
}

type SkillBundleFile struct {
	Path         string `json:"path"`
	FileKind     string `json:"file_kind"`
	MediaType    string `json:"media_type"`
	Encoding     string `json:"encoding"`
	IsExecutable bool   `json:"is_executable"`
	SizeBytes    int64  `json:"size_bytes"`
	SHA256       string `json:"sha256"`
	Content      []byte `json:"-"`
}

type SkillBundle struct {
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Files            []SkillBundleFile `json:"files"`
	FileCount        int               `json:"file_count"`
	SizeBytes        int64             `json:"size_bytes"`
	BundleHash       string            `json:"bundle_hash"`
	Manifest         map[string]any    `json:"manifest"`
	EntrypointPath   string            `json:"entrypoint_path"`
	EntrypointSHA256 string            `json:"entrypoint_sha256"`
	EntrypointBody   string            `json:"entrypoint_body"`
}

type SkillDetail struct {
	Skill      `json:",inline"`
	Content    string            `json:"content"`
	BundleHash string            `json:"bundle_hash"`
	FileCount  int               `json:"file_count"`
	Files      []SkillBundleFile `json:"files"`
	History    []VersionSummary  `json:"history"`
}

type CreateSkillInput struct {
	ProjectID   uuid.UUID
	Name        string
	Content     string
	Description string
	CreatedBy   string
	Enabled     *bool
}

type UpdateSkillInput struct {
	SkillID     uuid.UUID
	Content     string
	Description string
}

type CreateSkillBundleInput struct {
	ProjectID uuid.UUID
	Name      string
	Files     []SkillBundleFileInput
	CreatedBy string
	Enabled   *bool
}

type UpdateSkillBundleInput struct {
	SkillID      uuid.UUID
	Files        []SkillBundleFileInput
	Content      string
	Description  string
	ReplaceEntry bool
}

type UpdateSkillBindingsInput struct {
	SkillID     uuid.UUID
	WorkflowIDs []uuid.UUID
}

type RefreshSkillsInput struct {
	ProjectID     uuid.UUID
	WorkspaceRoot string
	AdapterType   string
	WorkflowID    *uuid.UUID
}

type RefreshSkillsResult struct {
	SkillsDir      string   `json:"skills_dir"`
	InjectedSkills []string `json:"injected_skills"`
}

type UpdateWorkflowSkillsInput struct {
	WorkflowID uuid.UUID
	Skills     []string
}

type RuntimeSnapshot struct {
	Workflow RuntimeWorkflowSnapshot
	Skills   []RuntimeSkillSnapshot
}

type RuntimeWorkflowSnapshot struct {
	WorkflowID            uuid.UUID
	VersionID             uuid.UUID
	Version               int
	Path                  string
	Content               string
	Name                  string
	Type                  Type
	RoleSlug              string
	RoleName              string
	RoleDescription       string
	PickupStatusIDs       []uuid.UUID
	FinishStatusIDs       []uuid.UUID
	PlatformAccessAllowed []string
}

type RuntimeSkillSnapshot struct {
	SkillID    uuid.UUID
	Name       string
	VersionID  uuid.UUID
	Version    int
	Content    string
	Files      []RuntimeSkillFileSnapshot
	IsRequired bool
}

type RuntimeSkillFileSnapshot struct {
	Path         string
	Content      []byte
	IsExecutable bool
}

type MaterializeRuntimeSnapshotInput struct {
	WorkspaceRoot string
	AdapterType   string
	Snapshot      RuntimeSnapshot
}

type ResolveRecordedRuntimeSnapshotInput struct {
	WorkflowID        uuid.UUID
	WorkflowVersionID *uuid.UUID
	SkillVersionIDs   []uuid.UUID
}

type MaterializedRuntimeSnapshot struct {
	HarnessPath       string
	SkillsDir         string
	WorkflowVersionID uuid.UUID
	SkillVersionIDs   []uuid.UUID
}

type BuildHarnessTemplateDataInput struct {
	WorkflowID         uuid.UUID
	TicketID           uuid.UUID
	AgentID            *uuid.UUID
	Workspace          string
	Timestamp          time.Time
	OpenASEVersion     string
	TicketURL          string
	Platform           HarnessPlatformData
	Machine            HarnessMachineData
	AccessibleMachines []HarnessAccessibleMachineData
}

type HarnessTemplateData struct {
	Ticket             HarnessTicketData
	Project            HarnessProjectData
	Repos              []HarnessRepoData
	AllRepos           []HarnessRepoData
	Agent              HarnessAgentData
	Machine            HarnessMachineData
	AccessibleMachines []HarnessAccessibleMachineData
	Attempt            int
	MaxAttempts        int
	Workspace          string
	Timestamp          string
	OpenASEVersion     string
	Workflow           HarnessWorkflowData
	Platform           HarnessPlatformData
}

type HarnessTicketData struct {
	ID               string
	Identifier       string
	Title            string
	Description      string
	Status           string
	Priority         string
	Type             string
	CreatedBy        string
	CreatedAt        string
	AttemptCount     int
	MaxAttempts      int
	BudgetUSD        float64
	ExternalRef      string
	ParentIdentifier string
	URL              string
	Links            []HarnessTicketLinkData
	Dependencies     []HarnessTicketDependencyData
}

type HarnessTicketLinkData struct {
	Type     string
	URL      string
	Title    string
	Status   string
	Relation string
}

type HarnessTicketDependencyData struct {
	Identifier string
	Title      string
	Type       string
	Status     string
}

type HarnessProjectData struct {
	ID          string
	Name        string
	Slug        string
	Description string
	Status      string
	Workflows   []HarnessProjectWorkflowData
	Statuses    []HarnessProjectStatusData
	Machines    []HarnessProjectMachineData
	Updates     []HarnessProjectUpdateThreadData
}

type HarnessProjectWorkflowData struct {
	Name            string
	Type            string
	RoleName        string
	RoleDescription string
	PickupStatus    string
	FinishStatus    string
	PickupStatuses  []HarnessProjectStatusData
	FinishStatuses  []HarnessProjectStatusData
	HarnessPath     string
	HarnessContent  string
	Skills          []string
	MaxConcurrent   int
	CurrentActive   int
	RecentTickets   []HarnessProjectWorkflowTicketData
}

type HarnessProjectWorkflowTicketData struct {
	Identifier        string
	Title             string
	Status            string
	Priority          string
	Type              string
	AttemptCount      int
	ConsecutiveErrors int
	RetryPaused       bool
	PauseReason       string
	CreatedAt         string
	StartedAt         string
	CompletedAt       string
}

type HarnessProjectStatusData struct {
	ID    string
	Name  string
	Stage string
	Color string
}

type HarnessProjectMachineData struct {
	Name        string
	Host        string
	Description string
	Labels      []string
	Status      string
	Resources   map[string]any
}

type HarnessProjectUpdateThreadData struct {
	ID             string
	Status         string
	Title          string
	BodyMarkdown   string
	CreatedBy      string
	CreatedAt      string
	UpdatedAt      string
	LastActivityAt string
	CommentCount   int
	Comments       []HarnessProjectUpdateCommentData
}

type HarnessProjectUpdateCommentData struct {
	ID           string
	BodyMarkdown string
	CreatedBy    string
	CreatedAt    string
	UpdatedAt    string
}

type HarnessRepoData struct {
	Name          string
	URL           string
	Path          string
	Branch        string
	DefaultBranch string
	Labels        []string
}

type HarnessAgentData struct {
	ID                    string
	Name                  string
	Provider              string
	AdapterType           string
	Model                 string
	TotalTicketsCompleted int
}

type HarnessMachineData struct {
	Name          string
	Host          string
	Description   string
	Labels        []string
	Resources     map[string]any
	WorkspaceRoot string
}

type HarnessAccessibleMachineData struct {
	Name        string
	Host        string
	Description string
	Labels      []string
	Resources   map[string]any
	SSHUser     string
}

type HarnessWorkflowData struct {
	Name         string
	Type         string
	RoleName     string
	PickupStatus string
	FinishStatus string
}

type HarnessPlatformData struct {
	APIURL     string
	AgentToken string
	ProjectID  string
	TicketID   string
}

type HarnessVariableGroup struct {
	Name      string                    `json:"name"`
	Variables []HarnessVariableMetadata `json:"variables"`
}

type HarnessVariableMetadata struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}
