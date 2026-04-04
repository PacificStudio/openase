package chat

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type RawProjectConversationFocus struct {
	Kind                 string                                     `json:"kind"`
	WorkflowID           *string                                    `json:"workflow_id"`
	WorkflowName         *string                                    `json:"workflow_name"`
	WorkflowType         *string                                    `json:"workflow_type"`
	HarnessPath          *string                                    `json:"harness_path"`
	IsActive             *bool                                      `json:"is_active"`
	SelectedArea         *string                                    `json:"selected_area"`
	HasDirtyDraft        *bool                                      `json:"has_dirty_draft"`
	SkillID              *string                                    `json:"skill_id"`
	SkillName            *string                                    `json:"skill_name"`
	SelectedFilePath     *string                                    `json:"selected_file_path"`
	BoundWorkflowNames   []string                                   `json:"bound_workflow_names"`
	TicketID             *string                                    `json:"ticket_id"`
	TicketIdentifier     *string                                    `json:"ticket_identifier"`
	TicketTitle          *string                                    `json:"ticket_title"`
	TicketDescription    *string                                    `json:"ticket_description"`
	TicketStatus         *string                                    `json:"ticket_status"`
	TicketPriority       *string                                    `json:"ticket_priority"`
	TicketAttemptCount   *int                                       `json:"ticket_attempt_count"`
	TicketRetryPaused    *bool                                      `json:"ticket_retry_paused"`
	TicketPauseReason    *string                                    `json:"ticket_pause_reason"`
	TicketDependencies   []RawProjectConversationTicketDependency   `json:"ticket_dependencies"`
	TicketRepoScopes     []RawProjectConversationTicketRepoScope    `json:"ticket_repo_scopes"`
	TicketRecentActivity []RawProjectConversationTicketActivity     `json:"ticket_recent_activity"`
	TicketHookHistory    []RawProjectConversationTicketHook         `json:"ticket_hook_history"`
	TicketAssignedAgent  *RawProjectConversationTicketAssignedAgent `json:"ticket_assigned_agent"`
	TicketCurrentRun     *RawProjectConversationTicketRun           `json:"ticket_current_run"`
	TicketTargetMachine  *RawProjectConversationTicketTargetMachine `json:"ticket_target_machine"`
	MachineID            *string                                    `json:"machine_id"`
	MachineName          *string                                    `json:"machine_name"`
	MachineHost          *string                                    `json:"machine_host"`
	MachineStatus        *string                                    `json:"machine_status"`
	HealthSummary        *string                                    `json:"health_summary"`
}

type RawProjectConversationTicketDependency struct {
	Identifier *string `json:"identifier"`
	Title      *string `json:"title"`
	Relation   *string `json:"relation"`
	Status     *string `json:"status"`
}

type RawProjectConversationTicketRepoScope struct {
	RepoID         *string `json:"repo_id"`
	RepoName       *string `json:"repo_name"`
	BranchName     *string `json:"branch_name"`
	PullRequestURL *string `json:"pull_request_url"`
}

type RawProjectConversationTicketActivity struct {
	EventType *string `json:"event_type"`
	Message   *string `json:"message"`
	CreatedAt *string `json:"created_at"`
}

type RawProjectConversationTicketHook struct {
	HookName  *string `json:"hook_name"`
	Status    *string `json:"status"`
	Output    *string `json:"output"`
	Timestamp *string `json:"timestamp"`
}

type RawProjectConversationTicketAssignedAgent struct {
	ID                  *string `json:"id"`
	Name                *string `json:"name"`
	Provider            *string `json:"provider"`
	RuntimeControlState *string `json:"runtime_control_state"`
	RuntimePhase        *string `json:"runtime_phase"`
}

type RawProjectConversationTicketRun struct {
	ID                 *string `json:"id"`
	AttemptNumber      *int    `json:"attempt_number"`
	Status             *string `json:"status"`
	CurrentStepStatus  *string `json:"current_step_status"`
	CurrentStepSummary *string `json:"current_step_summary"`
	LastError          *string `json:"last_error"`
}

type RawProjectConversationTicketTargetMachine struct {
	ID   *string `json:"id"`
	Name *string `json:"name"`
	Host *string `json:"host"`
}

type ProjectConversationFocusKind string

const (
	ProjectConversationFocusWorkflow ProjectConversationFocusKind = "workflow"
	ProjectConversationFocusSkill    ProjectConversationFocusKind = "skill"
	ProjectConversationFocusTicket   ProjectConversationFocusKind = "ticket"
	ProjectConversationFocusMachine  ProjectConversationFocusKind = "machine"
)

type ProjectConversationFocus struct {
	Kind     ProjectConversationFocusKind
	Workflow *ProjectConversationWorkflowFocus
	Skill    *ProjectConversationSkillFocus
	Ticket   *ProjectConversationTicketFocus
	Machine  *ProjectConversationMachineFocus
}

type ProjectConversationWorkflowFocus struct {
	ID            uuid.UUID
	Name          string
	Type          string
	HarnessPath   string
	IsActive      bool
	SelectedArea  string
	HasDirtyDraft bool
}

type ProjectConversationSkillFocus struct {
	ID                 uuid.UUID
	Name               string
	SelectedFilePath   string
	BoundWorkflowNames []string
	HasDirtyDraft      bool
}

type ProjectConversationTicketFocus struct {
	ID             uuid.UUID
	Identifier     string
	Title          string
	Description    string
	Status         string
	Priority       string
	AttemptCount   int
	RetryPaused    bool
	PauseReason    string
	SelectedArea   string
	Dependencies   []ProjectConversationTicketDependency
	RepoScopes     []ProjectConversationTicketRepoScope
	RecentActivity []ProjectConversationTicketActivity
	HookHistory    []ProjectConversationTicketHook
	AssignedAgent  *ProjectConversationTicketAssignedAgent
	CurrentRun     *ProjectConversationTicketRun
	TargetMachine  *ProjectConversationTicketTargetMachine
}

type ProjectConversationTicketDependency struct {
	Identifier string
	Title      string
	Relation   string
	Status     string
}

type ProjectConversationTicketRepoScope struct {
	RepoID         string
	RepoName       string
	BranchName     string
	PullRequestURL string
}

type ProjectConversationTicketActivity struct {
	EventType string
	Message   string
	CreatedAt string
}

type ProjectConversationTicketHook struct {
	HookName  string
	Status    string
	Output    string
	Timestamp string
}

type ProjectConversationTicketAssignedAgent struct {
	ID                  string
	Name                string
	Provider            string
	RuntimeControlState string
	RuntimePhase        string
}

type ProjectConversationTicketRun struct {
	ID                 string
	AttemptNumber      int
	Status             string
	CurrentStepStatus  string
	CurrentStepSummary string
	LastError          string
}

type ProjectConversationTicketTargetMachine struct {
	ID   string
	Name string
	Host string
}

type ProjectConversationMachineFocus struct {
	ID            uuid.UUID
	Name          string
	Host          string
	Status        string
	SelectedArea  string
	HealthSummary string
}

func ParseProjectConversationFocus(raw *RawProjectConversationFocus) (*ProjectConversationFocus, error) {
	if raw == nil || strings.TrimSpace(raw.Kind) == "" {
		return nil, nil
	}

	switch ProjectConversationFocusKind(strings.TrimSpace(raw.Kind)) {
	case ProjectConversationFocusWorkflow:
		workflowID, err := parseRequiredFocusUUID("focus.workflow_id", raw.WorkflowID)
		if err != nil {
			return nil, err
		}
		name, err := parseRequiredFocusString("focus.workflow_name", raw.WorkflowName)
		if err != nil {
			return nil, err
		}
		workflowType, err := parseRequiredFocusString("focus.workflow_type", raw.WorkflowType)
		if err != nil {
			return nil, err
		}
		harnessPath, err := parseRequiredFocusString("focus.harness_path", raw.HarnessPath)
		if err != nil {
			return nil, err
		}
		isActive, err := parseRequiredFocusBool("focus.is_active", raw.IsActive)
		if err != nil {
			return nil, err
		}
		return &ProjectConversationFocus{
			Kind: ProjectConversationFocusWorkflow,
			Workflow: &ProjectConversationWorkflowFocus{
				ID:            workflowID,
				Name:          name,
				Type:          workflowType,
				HarnessPath:   harnessPath,
				IsActive:      isActive,
				SelectedArea:  trimOptionalFocusString(raw.SelectedArea),
				HasDirtyDraft: boolPointerValue(raw.HasDirtyDraft),
			},
		}, nil
	case ProjectConversationFocusSkill:
		skillID, err := parseRequiredFocusUUID("focus.skill_id", raw.SkillID)
		if err != nil {
			return nil, err
		}
		name, err := parseRequiredFocusString("focus.skill_name", raw.SkillName)
		if err != nil {
			return nil, err
		}
		selectedFilePath, err := parseRequiredFocusString("focus.selected_file_path", raw.SelectedFilePath)
		if err != nil {
			return nil, err
		}
		return &ProjectConversationFocus{
			Kind: ProjectConversationFocusSkill,
			Skill: &ProjectConversationSkillFocus{
				ID:                 skillID,
				Name:               name,
				SelectedFilePath:   selectedFilePath,
				BoundWorkflowNames: trimNonEmptyFocusStrings(raw.BoundWorkflowNames),
				HasDirtyDraft:      boolPointerValue(raw.HasDirtyDraft),
			},
		}, nil
	case ProjectConversationFocusTicket:
		ticketID, err := parseRequiredFocusUUID("focus.ticket_id", raw.TicketID)
		if err != nil {
			return nil, err
		}
		identifier, err := parseRequiredFocusString("focus.ticket_identifier", raw.TicketIdentifier)
		if err != nil {
			return nil, err
		}
		title, err := parseRequiredFocusString("focus.ticket_title", raw.TicketTitle)
		if err != nil {
			return nil, err
		}
		status, err := parseRequiredFocusString("focus.ticket_status", raw.TicketStatus)
		if err != nil {
			return nil, err
		}
		return &ProjectConversationFocus{
			Kind: ProjectConversationFocusTicket,
			Ticket: &ProjectConversationTicketFocus{
				ID:             ticketID,
				Identifier:     identifier,
				Title:          title,
				Description:    trimOptionalFocusString(raw.TicketDescription),
				Status:         status,
				Priority:       trimOptionalFocusString(raw.TicketPriority),
				AttemptCount:   intPointerValue(raw.TicketAttemptCount),
				RetryPaused:    boolPointerValue(raw.TicketRetryPaused),
				PauseReason:    trimOptionalFocusString(raw.TicketPauseReason),
				SelectedArea:   trimOptionalFocusString(raw.SelectedArea),
				Dependencies:   parseTicketFocusDependencies(raw.TicketDependencies),
				RepoScopes:     parseTicketFocusRepoScopes(raw.TicketRepoScopes),
				RecentActivity: parseTicketFocusActivity(raw.TicketRecentActivity),
				HookHistory:    parseTicketFocusHooks(raw.TicketHookHistory),
				AssignedAgent:  parseTicketFocusAssignedAgent(raw.TicketAssignedAgent),
				CurrentRun:     parseTicketFocusRun(raw.TicketCurrentRun),
				TargetMachine:  parseTicketFocusTargetMachine(raw.TicketTargetMachine),
			},
		}, nil
	case ProjectConversationFocusMachine:
		machineID, err := parseRequiredFocusUUID("focus.machine_id", raw.MachineID)
		if err != nil {
			return nil, err
		}
		name, err := parseRequiredFocusString("focus.machine_name", raw.MachineName)
		if err != nil {
			return nil, err
		}
		host, err := parseRequiredFocusString("focus.machine_host", raw.MachineHost)
		if err != nil {
			return nil, err
		}
		return &ProjectConversationFocus{
			Kind: ProjectConversationFocusMachine,
			Machine: &ProjectConversationMachineFocus{
				ID:            machineID,
				Name:          name,
				Host:          host,
				Status:        trimOptionalFocusString(raw.MachineStatus),
				SelectedArea:  trimOptionalFocusString(raw.SelectedArea),
				HealthSummary: trimOptionalFocusString(raw.HealthSummary),
			},
		}, nil
	default:
		return nil, fmt.Errorf("focus.kind must be one of workflow, skill, ticket, machine")
	}
}

func parseRequiredFocusUUID(field string, raw *string) (uuid.UUID, error) {
	value, err := parseRequiredFocusString(field, raw)
	if err != nil {
		return uuid.UUID{}, err
	}
	parsed, parseErr := uuid.Parse(value)
	if parseErr != nil {
		return uuid.UUID{}, fmt.Errorf("%s must be a valid UUID", field)
	}
	return parsed, nil
}

func parseRequiredFocusString(field string, raw *string) (string, error) {
	value := trimOptionalFocusString(raw)
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", field)
	}
	return value, nil
}

func parseRequiredFocusBool(field string, raw *bool) (bool, error) {
	if raw == nil {
		return false, fmt.Errorf("%s must not be empty", field)
	}
	return *raw, nil
}

func trimOptionalFocusString(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
}

func trimNonEmptyFocusStrings(raw []string) []string {
	items := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func boolPointerValue(raw *bool) bool {
	return raw != nil && *raw
}

func intPointerValue(raw *int) int {
	if raw == nil {
		return 0
	}
	return *raw
}

func parseTicketFocusDependencies(raw []RawProjectConversationTicketDependency) []ProjectConversationTicketDependency {
	items := make([]ProjectConversationTicketDependency, 0, len(raw))
	for _, item := range raw {
		identifier := trimOptionalFocusString(item.Identifier)
		title := trimOptionalFocusString(item.Title)
		if identifier == "" && title == "" {
			continue
		}
		items = append(items, ProjectConversationTicketDependency{
			Identifier: identifier,
			Title:      title,
			Relation:   trimOptionalFocusString(item.Relation),
			Status:     trimOptionalFocusString(item.Status),
		})
	}
	return items
}

func parseTicketFocusRepoScopes(raw []RawProjectConversationTicketRepoScope) []ProjectConversationTicketRepoScope {
	items := make([]ProjectConversationTicketRepoScope, 0, len(raw))
	for _, item := range raw {
		repoID := trimOptionalFocusString(item.RepoID)
		repoName := trimOptionalFocusString(item.RepoName)
		branchName := trimOptionalFocusString(item.BranchName)
		pullRequestURL := trimOptionalFocusString(item.PullRequestURL)
		if repoID == "" && repoName == "" && branchName == "" && pullRequestURL == "" {
			continue
		}
		items = append(items, ProjectConversationTicketRepoScope{
			RepoID:         repoID,
			RepoName:       repoName,
			BranchName:     branchName,
			PullRequestURL: pullRequestURL,
		})
	}
	return items
}

func parseTicketFocusActivity(raw []RawProjectConversationTicketActivity) []ProjectConversationTicketActivity {
	items := make([]ProjectConversationTicketActivity, 0, len(raw))
	for _, item := range raw {
		eventType := trimOptionalFocusString(item.EventType)
		message := trimOptionalFocusString(item.Message)
		createdAt := trimOptionalFocusString(item.CreatedAt)
		if eventType == "" && message == "" && createdAt == "" {
			continue
		}
		items = append(items, ProjectConversationTicketActivity{
			EventType: eventType,
			Message:   message,
			CreatedAt: createdAt,
		})
	}
	return items
}

func parseTicketFocusHooks(raw []RawProjectConversationTicketHook) []ProjectConversationTicketHook {
	items := make([]ProjectConversationTicketHook, 0, len(raw))
	for _, item := range raw {
		hookName := trimOptionalFocusString(item.HookName)
		status := trimOptionalFocusString(item.Status)
		output := trimOptionalFocusString(item.Output)
		timestamp := trimOptionalFocusString(item.Timestamp)
		if hookName == "" && status == "" && output == "" && timestamp == "" {
			continue
		}
		items = append(items, ProjectConversationTicketHook{
			HookName:  hookName,
			Status:    status,
			Output:    output,
			Timestamp: timestamp,
		})
	}
	return items
}

func parseTicketFocusAssignedAgent(raw *RawProjectConversationTicketAssignedAgent) *ProjectConversationTicketAssignedAgent {
	if raw == nil {
		return nil
	}
	item := &ProjectConversationTicketAssignedAgent{
		ID:                  trimOptionalFocusString(raw.ID),
		Name:                trimOptionalFocusString(raw.Name),
		Provider:            trimOptionalFocusString(raw.Provider),
		RuntimeControlState: trimOptionalFocusString(raw.RuntimeControlState),
		RuntimePhase:        trimOptionalFocusString(raw.RuntimePhase),
	}
	if item.ID == "" && item.Name == "" && item.Provider == "" && item.RuntimeControlState == "" && item.RuntimePhase == "" {
		return nil
	}
	return item
}

func parseTicketFocusRun(raw *RawProjectConversationTicketRun) *ProjectConversationTicketRun {
	if raw == nil {
		return nil
	}
	item := &ProjectConversationTicketRun{
		ID:                 trimOptionalFocusString(raw.ID),
		AttemptNumber:      intPointerValue(raw.AttemptNumber),
		Status:             trimOptionalFocusString(raw.Status),
		CurrentStepStatus:  trimOptionalFocusString(raw.CurrentStepStatus),
		CurrentStepSummary: trimOptionalFocusString(raw.CurrentStepSummary),
		LastError:          trimOptionalFocusString(raw.LastError),
	}
	if item.ID == "" && item.AttemptNumber == 0 && item.Status == "" && item.CurrentStepStatus == "" && item.CurrentStepSummary == "" && item.LastError == "" {
		return nil
	}
	return item
}

func parseTicketFocusTargetMachine(raw *RawProjectConversationTicketTargetMachine) *ProjectConversationTicketTargetMachine {
	if raw == nil {
		return nil
	}
	item := &ProjectConversationTicketTargetMachine{
		ID:   trimOptionalFocusString(raw.ID),
		Name: trimOptionalFocusString(raw.Name),
		Host: trimOptionalFocusString(raw.Host),
	}
	if item.ID == "" && item.Name == "" && item.Host == "" {
		return nil
	}
	return item
}
