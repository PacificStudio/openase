package hradvisor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var activationRoleSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type ActivateRecommendationRequest struct {
	RoleSlug              string `json:"role_slug"`
	CreateBootstrapTicket *bool  `json:"create_bootstrap_ticket,omitempty"`
}

type ActivateRecommendationInput struct {
	ProjectID             uuid.UUID
	RoleSlug              string
	CreateBootstrapTicket bool
}

type ActivationTemplate struct {
	RoleSlug              string
	WorkflowName          string
	WorkflowType          string
	HarnessPath           string
	HarnessContent        string
	PickupStatusNames     []string
	FinishStatusNames     []string
	Summary               string
	PlatformAccessAllowed []string
	SkillNames            []string
}

func ParseActivateRecommendation(projectID uuid.UUID, raw ActivateRecommendationRequest) (ActivateRecommendationInput, error) {
	roleSlug := strings.TrimSpace(strings.ToLower(raw.RoleSlug))
	if roleSlug == "" {
		return ActivateRecommendationInput{}, fmt.Errorf("role_slug must not be empty")
	}
	if !activationRoleSlugPattern.MatchString(roleSlug) {
		return ActivateRecommendationInput{}, fmt.Errorf("role_slug must be a lowercase slug")
	}

	input := ActivateRecommendationInput{
		ProjectID: projectID,
		RoleSlug:  roleSlug,
	}
	if raw.CreateBootstrapTicket != nil {
		input.CreateBootstrapTicket = *raw.CreateBootstrapTicket
	}

	return input, nil
}

func NormalizeActivationTemplate(template ActivationTemplate) (ActivationTemplate, error) {
	template.RoleSlug = strings.TrimSpace(template.RoleSlug)
	template.WorkflowName = strings.TrimSpace(template.WorkflowName)
	template.WorkflowType = strings.TrimSpace(template.WorkflowType)
	template.HarnessPath = strings.TrimSpace(template.HarnessPath)
	template.HarnessContent = normalizeActivationHarnessBody(template.HarnessContent)
	template.Summary = strings.TrimSpace(template.Summary)
	template.PickupStatusNames = normalizeActivationStringList(template.PickupStatusNames)
	template.FinishStatusNames = normalizeActivationStringList(template.FinishStatusNames)
	template.PlatformAccessAllowed = normalizeActivationStringList(template.PlatformAccessAllowed)
	template.SkillNames = normalizeActivationStringList(template.SkillNames)

	if template.RoleSlug == "" {
		return ActivationTemplate{}, fmt.Errorf("role_slug must not be empty")
	}
	if template.WorkflowName == "" {
		return ActivationTemplate{}, fmt.Errorf("workflow_name must not be empty")
	}
	if template.WorkflowType == "" {
		return ActivationTemplate{}, fmt.Errorf("workflow_type must not be empty")
	}
	if template.HarnessPath == "" {
		return ActivationTemplate{}, fmt.Errorf("harness_path must not be empty")
	}
	if strings.TrimSpace(template.HarnessContent) == "" {
		return ActivationTemplate{}, fmt.Errorf("harness_content must not be empty")
	}
	if len(template.PickupStatusNames) == 0 {
		return ActivationTemplate{}, fmt.Errorf("pickup_status_names must not be empty")
	}
	if len(template.FinishStatusNames) == 0 {
		return ActivationTemplate{}, fmt.Errorf("finish_status_names must not be empty")
	}

	return template, nil
}

func normalizeActivationStringList(items []string) []string {
	normalized := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
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

func normalizeActivationHarnessBody(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.TrimSpace(strings.ReplaceAll(normalized, "\r", "\n"))
}
