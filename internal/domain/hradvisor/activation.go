package hradvisor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
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
	RoleSlug         string
	WorkflowName     string
	WorkflowType     string
	HarnessPath      string
	HarnessContent   string
	PickupStatusName string
	FinishStatusName string
	Summary          string
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

func ParseActivationTemplate(roleSlug string, harnessPath string, harnessContent string, summary string) (ActivationTemplate, error) {
	frontmatter, err := extractActivationFrontmatter(harnessContent)
	if err != nil {
		return ActivationTemplate{}, err
	}

	var document struct {
		Workflow struct {
			Name string `yaml:"name"`
			Type string `yaml:"type"`
			Role string `yaml:"role"`
		} `yaml:"workflow"`
		Status struct {
			Pickup string `yaml:"pickup"`
			Finish string `yaml:"finish"`
		} `yaml:"status"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return ActivationTemplate{}, fmt.Errorf("parse harness frontmatter: %w", err)
	}

	normalizedRoleSlug := strings.TrimSpace(roleSlug)
	workflowRoleSlug := strings.TrimSpace(document.Workflow.Role)
	if workflowRoleSlug == "" {
		workflowRoleSlug = normalizedRoleSlug
	}
	if workflowRoleSlug != normalizedRoleSlug {
		return ActivationTemplate{}, fmt.Errorf("workflow role %q does not match requested role %q", workflowRoleSlug, normalizedRoleSlug)
	}

	template := ActivationTemplate{
		RoleSlug:         workflowRoleSlug,
		WorkflowName:     strings.TrimSpace(document.Workflow.Name),
		WorkflowType:     strings.TrimSpace(document.Workflow.Type),
		HarnessPath:      strings.TrimSpace(harnessPath),
		HarnessContent:   harnessContent,
		PickupStatusName: strings.TrimSpace(document.Status.Pickup),
		FinishStatusName: strings.TrimSpace(document.Status.Finish),
		Summary:          strings.TrimSpace(summary),
	}
	if template.WorkflowName == "" {
		return ActivationTemplate{}, fmt.Errorf("workflow.name must not be empty")
	}
	if template.WorkflowType == "" {
		return ActivationTemplate{}, fmt.Errorf("workflow.type must not be empty")
	}
	if template.HarnessPath == "" {
		return ActivationTemplate{}, fmt.Errorf("harness_path must not be empty")
	}
	if template.HarnessContent == "" {
		return ActivationTemplate{}, fmt.Errorf("harness_content must not be empty")
	}
	if template.PickupStatusName == "" {
		return ActivationTemplate{}, fmt.Errorf("status.pickup must not be empty")
	}
	if template.FinishStatusName == "" {
		return ActivationTemplate{}, fmt.Errorf("status.finish must not be empty")
	}

	return template, nil
}

func extractActivationFrontmatter(content string) (string, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", fmt.Errorf("harness frontmatter must start with ---")
	}

	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) != "---" {
			continue
		}
		frontmatter := strings.Join(lines[1:index], "\n")
		if strings.TrimSpace(frontmatter) == "" {
			return "", fmt.Errorf("harness frontmatter must not be empty")
		}
		return frontmatter, nil
	}

	return "", fmt.Errorf("harness frontmatter closing delimiter not found")
}
