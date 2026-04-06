package httpapi

import (
	"fmt"
	"strings"

	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

type rawCreateWorkflowRequest struct {
	AgentID               string         `json:"agent_id"`
	Name                  string         `json:"name"`
	Type                  string         `json:"type"`
	RoleSlug              *string        `json:"role_slug"`
	RoleName              *string        `json:"role_name"`
	RoleDescription       *string        `json:"role_description"`
	PlatformAccessAllowed []string       `json:"platform_access_allowed"`
	SkillNames            []string       `json:"skill_names"`
	HarnessPath           *string        `json:"harness_path"`
	HarnessContent        string         `json:"harness_content"`
	Hooks                 map[string]any `json:"hooks"`
	MaxConcurrent         *int           `json:"max_concurrent"`
	MaxRetryAttempts      *int           `json:"max_retry_attempts"`
	TimeoutMinutes        *int           `json:"timeout_minutes"`
	StallTimeoutMinutes   *int           `json:"stall_timeout_minutes"`
	IsActive              *bool          `json:"is_active"`
	PickupStatusIDs       []string       `json:"pickup_status_ids"`
	FinishStatusIDs       []string       `json:"finish_status_ids"`
}

type rawUpdateWorkflowRequest struct {
	AgentID               *string         `json:"agent_id"`
	Name                  *string         `json:"name"`
	Type                  *string         `json:"type"`
	RoleSlug              *string         `json:"role_slug"`
	RoleName              *string         `json:"role_name"`
	RoleDescription       *string         `json:"role_description"`
	PlatformAccessAllowed *[]string       `json:"platform_access_allowed"`
	HarnessPath           *string         `json:"harness_path"`
	Hooks                 *map[string]any `json:"hooks"`
	MaxConcurrent         *int            `json:"max_concurrent"`
	MaxRetryAttempts      *int            `json:"max_retry_attempts"`
	TimeoutMinutes        *int            `json:"timeout_minutes"`
	StallTimeoutMinutes   *int            `json:"stall_timeout_minutes"`
	IsActive              *bool           `json:"is_active"`
	PickupStatusIDs       *[]string       `json:"pickup_status_ids"`
	FinishStatusIDs       *[]string       `json:"finish_status_ids"`
}

type rawUpdateHarnessRequest struct {
	Content string `json:"content"`
}

type rawRetireWorkflowRequest struct{}

type rawReplaceWorkflowReferencesRequest struct {
	ReplacementWorkflowID string `json:"replacement_workflow_id"`
}

type rawValidateHarnessRequest struct {
	Content string `json:"content"`
}

func parseCreateWorkflowRequest(projectID uuid.UUID, auditActor string, raw rawCreateWorkflowRequest) (workflowservice.CreateInput, error) {
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return workflowservice.CreateInput{}, fmt.Errorf("name must not be empty")
	}

	workflowType, err := parseWorkflowTypeLabel(raw.Type)
	if err != nil {
		return workflowservice.CreateInput{}, err
	}
	agentID, err := parseUUIDString("agent_id", raw.AgentID)
	if err != nil {
		return workflowservice.CreateInput{}, err
	}

	pickupStatusIDs, err := parseStatusBindingSet("pickup_status_ids", raw.PickupStatusIDs)
	if err != nil {
		return workflowservice.CreateInput{}, err
	}

	finishStatusIDs, err := parseStatusBindingSet("finish_status_ids", raw.FinishStatusIDs)
	if err != nil {
		return workflowservice.CreateInput{}, err
	}

	maxConcurrent, err := parseConcurrencyLimit("max_concurrent", raw.MaxConcurrent)
	if err != nil {
		return workflowservice.CreateInput{}, err
	}
	maxRetryAttempts, err := parseMaxRetryAttempts(raw.MaxRetryAttempts, 3)
	if err != nil {
		return workflowservice.CreateInput{}, err
	}
	timeoutMinutes, err := parsePositiveInt("timeout_minutes", raw.TimeoutMinutes, 60)
	if err != nil {
		return workflowservice.CreateInput{}, err
	}
	stallTimeoutMinutes, err := parsePositiveInt("stall_timeout_minutes", raw.StallTimeoutMinutes, 5)
	if err != nil {
		return workflowservice.CreateInput{}, err
	}

	input := workflowservice.CreateInput{
		ProjectID:             projectID,
		AgentID:               agentID,
		Name:                  name,
		Type:                  workflowType,
		PlatformAccessAllowed: parseNormalizedStringList(raw.PlatformAccessAllowed),
		SkillNames:            parseNormalizedStringList(raw.SkillNames),
		HarnessContent:        raw.HarnessContent,
		Hooks:                 raw.Hooks,
		MaxConcurrent:         maxConcurrent,
		MaxRetryAttempts:      maxRetryAttempts,
		TimeoutMinutes:        timeoutMinutes,
		StallTimeoutMinutes:   stallTimeoutMinutes,
		IsActive:              true,
		PickupStatusIDs:       pickupStatusIDs,
		FinishStatusIDs:       finishStatusIDs,
	}
	if strings.TrimSpace(auditActor) != "" {
		input.CreatedBy = strings.TrimSpace(auditActor)
	}
	if raw.RoleSlug != nil {
		input.RoleSlug = strings.TrimSpace(*raw.RoleSlug)
	}
	if raw.RoleName != nil {
		input.RoleName = strings.TrimSpace(*raw.RoleName)
	}
	if raw.RoleDescription != nil {
		input.RoleDescription = strings.TrimSpace(*raw.RoleDescription)
	}
	if raw.HarnessPath != nil {
		path := strings.TrimSpace(*raw.HarnessPath)
		input.HarnessPath = &path
	}
	if raw.IsActive != nil {
		input.IsActive = *raw.IsActive
	}

	return input, nil
}

func parseUpdateWorkflowRequest(workflowID uuid.UUID, auditActor string, raw rawUpdateWorkflowRequest) (workflowservice.UpdateInput, error) {
	input := workflowservice.UpdateInput{WorkflowID: workflowID}
	if strings.TrimSpace(auditActor) != "" {
		input.EditedBy = strings.TrimSpace(auditActor)
	}

	if raw.AgentID != nil {
		agentID, err := parseUUIDString("agent_id", *raw.AgentID)
		if err != nil {
			return workflowservice.UpdateInput{}, err
		}
		input.AgentID = workflowservice.Some(agentID)
	}

	if raw.Name != nil {
		name := strings.TrimSpace(*raw.Name)
		if name == "" {
			return workflowservice.UpdateInput{}, fmt.Errorf("name must not be empty")
		}
		input.Name = workflowservice.Some(name)
	}

	if raw.Type != nil {
		workflowType, err := parseWorkflowTypeLabel(*raw.Type)
		if err != nil {
			return workflowservice.UpdateInput{}, err
		}
		input.Type = workflowservice.Some(workflowType)
	}
	if raw.RoleSlug != nil {
		return workflowservice.UpdateInput{}, fmt.Errorf("role_slug cannot be updated")
	}
	if raw.RoleName != nil {
		input.RoleName = workflowservice.Some(strings.TrimSpace(*raw.RoleName))
	}
	if raw.RoleDescription != nil {
		input.RoleDescription = workflowservice.Some(strings.TrimSpace(*raw.RoleDescription))
	}
	if raw.PlatformAccessAllowed != nil {
		input.PlatformAccessAllowed = workflowservice.Some(parseNormalizedStringList(*raw.PlatformAccessAllowed))
	}

	if raw.HarnessPath != nil {
		input.HarnessPath = workflowservice.Some(strings.TrimSpace(*raw.HarnessPath))
	}

	if raw.Hooks != nil {
		input.Hooks = workflowservice.Some(*raw.Hooks)
	}

	if raw.MaxConcurrent != nil {
		if *raw.MaxConcurrent < 0 {
			return workflowservice.UpdateInput{}, fmt.Errorf(
				"max_concurrent must be greater than or equal to zero",
			)
		}
		input.MaxConcurrent = workflowservice.Some(*raw.MaxConcurrent)
	}

	if raw.MaxRetryAttempts != nil {
		if *raw.MaxRetryAttempts < 0 {
			return workflowservice.UpdateInput{}, fmt.Errorf("max_retry_attempts must be greater than or equal to zero")
		}
		input.MaxRetryAttempts = workflowservice.Some(*raw.MaxRetryAttempts)
	}

	if raw.TimeoutMinutes != nil {
		if *raw.TimeoutMinutes < 1 {
			return workflowservice.UpdateInput{}, fmt.Errorf("timeout_minutes must be greater than zero")
		}
		input.TimeoutMinutes = workflowservice.Some(*raw.TimeoutMinutes)
	}

	if raw.StallTimeoutMinutes != nil {
		if *raw.StallTimeoutMinutes < 1 {
			return workflowservice.UpdateInput{}, fmt.Errorf("stall_timeout_minutes must be greater than zero")
		}
		input.StallTimeoutMinutes = workflowservice.Some(*raw.StallTimeoutMinutes)
	}

	if raw.IsActive != nil {
		input.IsActive = workflowservice.Some(*raw.IsActive)
	}

	if raw.PickupStatusIDs != nil {
		parsed, err := parseStatusBindingSet("pickup_status_ids", *raw.PickupStatusIDs)
		if err != nil {
			return workflowservice.UpdateInput{}, err
		}
		input.PickupStatusIDs = workflowservice.Some(parsed)
	}

	if raw.FinishStatusIDs != nil {
		parsed, err := parseStatusBindingSet("finish_status_ids", *raw.FinishStatusIDs)
		if err != nil {
			return workflowservice.UpdateInput{}, err
		}
		input.FinishStatusIDs = workflowservice.Some(parsed)
	}

	return input, nil
}

func parseNormalizedStringList(raw []string) []string {
	normalized := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func parseRetireWorkflowRequest(workflowID uuid.UUID, auditActor string, raw rawRetireWorkflowRequest) string {
	_ = workflowID
	_ = raw
	return strings.TrimSpace(auditActor)
}

func parseReplaceWorkflowReferencesRequest(
	workflowID uuid.UUID,
	auditActor string,
	raw rawReplaceWorkflowReferencesRequest,
) (workflowservice.ReplaceWorkflowReferencesInput, string, error) {
	replacementWorkflowID, err := parseUUIDString("replacement_workflow_id", raw.ReplacementWorkflowID)
	if err != nil {
		return workflowservice.ReplaceWorkflowReferencesInput{}, "", err
	}
	editedBy := strings.TrimSpace(auditActor)
	return workflowservice.ReplaceWorkflowReferencesInput{
		WorkflowID:            workflowID,
		ReplacementWorkflowID: replacementWorkflowID,
	}, editedBy, nil
}

func parseUpdateHarnessRequest(workflowID uuid.UUID, auditActor string, raw rawUpdateHarnessRequest) (workflowservice.UpdateHarnessInput, error) {
	if strings.TrimSpace(raw.Content) == "" {
		return workflowservice.UpdateHarnessInput{}, fmt.Errorf("content must not be empty")
	}

	input := workflowservice.UpdateHarnessInput{
		WorkflowID: workflowID,
		Content:    raw.Content,
	}
	if strings.TrimSpace(auditActor) != "" {
		input.EditedBy = strings.TrimSpace(auditActor)
	}
	return input, nil
}

func parseWorkflowTypeLabel(raw string) (workflowservice.TypeLabel, error) {
	return workflowservice.ParseTypeLabel(raw)
}

func parseUUIDString(fieldName string, raw string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%s must be a valid UUID", fieldName)
	}

	return parsed, nil
}

func parseOptionalUUIDString(fieldName string, raw *string) (*uuid.UUID, error) {
	if raw == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid UUID", fieldName)
	}

	return &parsed, nil
}

func parseStatusBindingSet(fieldName string, raw []string) (workflowservice.StatusBindingSet, error) {
	ids := make([]uuid.UUID, 0, len(raw))
	for _, item := range raw {
		parsed, err := parseUUIDString(fieldName, item)
		if err != nil {
			return workflowservice.StatusBindingSet{}, err
		}
		ids = append(ids, parsed)
	}

	return workflowservice.ParseStatusBindingSet(fieldName, ids)
}

func parsePositiveInt(fieldName string, raw *int, defaultValue int) (int, error) {
	if raw == nil {
		return defaultValue, nil
	}
	if *raw < 1 {
		return 0, fmt.Errorf("%s must be greater than zero", fieldName)
	}

	return *raw, nil
}

func parseConcurrencyLimit(fieldName string, raw *int) (int, error) {
	if raw == nil {
		return 0, nil
	}
	if *raw < 0 {
		return 0, fmt.Errorf("%s must be greater than or equal to zero", fieldName)
	}

	return *raw, nil
}

func parseMaxRetryAttempts(raw *int, defaultValue int) (int, error) {
	if raw == nil {
		return defaultValue, nil
	}
	if *raw < 0 {
		return 0, fmt.Errorf("max_retry_attempts must be greater than or equal to zero")
	}

	return *raw, nil
}
