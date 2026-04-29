package projectpreset

import (
	"errors"

	ticketingdomain "github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	workflowdomain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/google/uuid"
)

var (
	ErrPresetNotFound       = errors.New("project preset not found")
	ErrActiveTicketsPresent = errors.New("project preset requires zero active tickets")
	ErrAgentBindingRequired = errors.New("workflow agent binding is required")
	ErrAgentBindingInvalid  = errors.New("workflow agent binding is invalid")
	ErrWorkflowRoleConflict = errors.New("existing workflow role metadata conflicts with preset")
)

type Preset struct {
	Version   int        `json:"version"`
	Meta      PresetMeta `json:"preset"`
	Statuses  []Status   `json:"statuses"`
	Workflows []Workflow `json:"workflows"`
	ProjectAI ProjectAI  `json:"project_ai,omitempty"`
}

type PresetMeta struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SourcePath  string `json:"source_path,omitempty"`
}

type Status struct {
	Name          string                      `json:"name"`
	Stage         ticketingdomain.StatusStage `json:"stage"`
	Color         string                      `json:"color"`
	Icon          string                      `json:"icon,omitempty"`
	MaxActiveRuns *int                        `json:"max_active_runs,omitempty"`
	Default       bool                        `json:"default,omitempty"`
	Description   string                      `json:"description,omitempty"`
}

type Workflow struct {
	Key                   string              `json:"key"`
	Name                  string              `json:"name"`
	Type                  workflowdomain.Type `json:"type"`
	RoleSlug              string              `json:"role_slug,omitempty"`
	RoleName              string              `json:"role_name,omitempty"`
	RoleDescription       string              `json:"role_description,omitempty"`
	PlatformAccessAllowed []string            `json:"platform_access_allowed,omitempty"`
	SkillNames            []string            `json:"skill_names,omitempty"`
	HarnessPath           *string             `json:"harness_path,omitempty"`
	HarnessContent        string              `json:"harness_content,omitempty"`
	MaxConcurrent         int                 `json:"max_concurrent"`
	MaxRetryAttempts      int                 `json:"max_retry_attempts"`
	TimeoutMinutes        int                 `json:"timeout_minutes"`
	StallTimeoutMinutes   int                 `json:"stall_timeout_minutes"`
	PickupStatusNames     []string            `json:"pickup_statuses"`
	FinishStatusNames     []string            `json:"finish_statuses"`
}

type ProjectAI struct {
	SkillReferences []SkillReference `json:"skill_references,omitempty"`
}

type SkillReference struct {
	Skill string   `json:"skill"`
	Files []string `json:"files,omitempty"`
}

type WorkflowAgentBinding struct {
	WorkflowKey string    `json:"workflow_key"`
	AgentID     uuid.UUID `json:"agent_id"`
}

type ApplyInput struct {
	ProjectID     uuid.UUID              `json:"project_id"`
	PresetKey     string                 `json:"preset_key"`
	AppliedBy     string                 `json:"applied_by,omitempty"`
	AgentBindings []WorkflowAgentBinding `json:"workflow_agent_bindings"`
}

type Catalog struct {
	ActiveTicketCount int      `json:"active_ticket_count"`
	CanApply          bool     `json:"can_apply"`
	Presets           []Preset `json:"presets"`
}

type ApplyResult struct {
	Preset            Preset            `json:"preset"`
	ActiveTicketCount int               `json:"active_ticket_count"`
	Statuses          []AppliedStatus   `json:"statuses"`
	Workflows         []AppliedWorkflow `json:"workflows"`
}

type AppliedStatus struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Action string    `json:"action"`
}

type AppliedWorkflow struct {
	ID          uuid.UUID `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	AgentID     uuid.UUID `json:"agent_id"`
	AgentName   string    `json:"agent_name"`
	Action      string    `json:"action"`
	HarnessPath string    `json:"harness_path"`
}
