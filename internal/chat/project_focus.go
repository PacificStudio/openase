package chat

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type RawProjectConversationFocus struct {
	Kind               string   `json:"kind"`
	WorkflowID         *string  `json:"workflow_id"`
	WorkflowName       *string  `json:"workflow_name"`
	WorkflowType       *string  `json:"workflow_type"`
	HarnessPath        *string  `json:"harness_path"`
	IsActive           *bool    `json:"is_active"`
	SelectedArea       *string  `json:"selected_area"`
	HasDirtyDraft      *bool    `json:"has_dirty_draft"`
	SkillID            *string  `json:"skill_id"`
	SkillName          *string  `json:"skill_name"`
	SelectedFilePath   *string  `json:"selected_file_path"`
	BoundWorkflowNames []string `json:"bound_workflow_names"`
	TicketID           *string  `json:"ticket_id"`
	TicketIdentifier   *string  `json:"ticket_identifier"`
	TicketTitle        *string  `json:"ticket_title"`
	TicketStatus       *string  `json:"ticket_status"`
	MachineID          *string  `json:"machine_id"`
	MachineName        *string  `json:"machine_name"`
	MachineHost        *string  `json:"machine_host"`
	MachineStatus      *string  `json:"machine_status"`
	HealthSummary      *string  `json:"health_summary"`
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
	ID           uuid.UUID
	Identifier   string
	Title        string
	Status       string
	SelectedArea string
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
				ID:           ticketID,
				Identifier:   identifier,
				Title:        title,
				Status:       status,
				SelectedArea: trimOptionalFocusString(raw.SelectedArea),
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
