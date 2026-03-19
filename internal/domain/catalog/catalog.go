package catalog

import (
	"fmt"
	"regexp"
	"strings"

	entproject "github.com/BetterAndBetterII/openase/ent/project"
	"github.com/google/uuid"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Organization struct {
	ID                     uuid.UUID
	Name                   string
	Slug                   string
	DefaultAgentProviderID *uuid.UUID
}

type Project struct {
	ID                     uuid.UUID
	OrganizationID         uuid.UUID
	Name                   string
	Slug                   string
	Description            string
	Status                 entproject.Status
	DefaultWorkflowID      *uuid.UUID
	DefaultAgentProviderID *uuid.UUID
	MaxConcurrentAgents    int
}

type OrganizationInput struct {
	Name                   string  `json:"name"`
	Slug                   string  `json:"slug"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id"`
}

type ProjectInput struct {
	Name                   string  `json:"name"`
	Slug                   string  `json:"slug"`
	Description            string  `json:"description"`
	Status                 string  `json:"status"`
	DefaultWorkflowID      *string `json:"default_workflow_id"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id"`
	MaxConcurrentAgents    *int    `json:"max_concurrent_agents"`
}

type CreateOrganization struct {
	Name                   string
	Slug                   string
	DefaultAgentProviderID *uuid.UUID
}

type UpdateOrganization struct {
	ID                     uuid.UUID
	Name                   string
	Slug                   string
	DefaultAgentProviderID *uuid.UUID
}

type CreateProject struct {
	OrganizationID         uuid.UUID
	Name                   string
	Slug                   string
	Description            string
	Status                 entproject.Status
	DefaultWorkflowID      *uuid.UUID
	DefaultAgentProviderID *uuid.UUID
	MaxConcurrentAgents    int
}

type UpdateProject struct {
	ID                     uuid.UUID
	OrganizationID         uuid.UUID
	Name                   string
	Slug                   string
	Description            string
	Status                 entproject.Status
	DefaultWorkflowID      *uuid.UUID
	DefaultAgentProviderID *uuid.UUID
	MaxConcurrentAgents    int
}

func ParseCreateOrganization(raw OrganizationInput) (CreateOrganization, error) {
	name, err := parseName("name", raw.Name)
	if err != nil {
		return CreateOrganization{}, err
	}

	slug, err := parseSlug(raw.Slug)
	if err != nil {
		return CreateOrganization{}, err
	}

	defaultAgentProviderID, err := parseOptionalUUID("default_agent_provider_id", raw.DefaultAgentProviderID)
	if err != nil {
		return CreateOrganization{}, err
	}

	return CreateOrganization{
		Name:                   name,
		Slug:                   slug,
		DefaultAgentProviderID: defaultAgentProviderID,
	}, nil
}

func ParseUpdateOrganization(id uuid.UUID, raw OrganizationInput) (UpdateOrganization, error) {
	input, err := ParseCreateOrganization(raw)
	if err != nil {
		return UpdateOrganization{}, err
	}

	return UpdateOrganization{
		ID:                     id,
		Name:                   input.Name,
		Slug:                   input.Slug,
		DefaultAgentProviderID: input.DefaultAgentProviderID,
	}, nil
}

func ParseCreateProject(organizationID uuid.UUID, raw ProjectInput) (CreateProject, error) {
	name, err := parseName("name", raw.Name)
	if err != nil {
		return CreateProject{}, err
	}

	slug, err := parseSlug(raw.Slug)
	if err != nil {
		return CreateProject{}, err
	}

	defaultWorkflowID, err := parseOptionalUUID("default_workflow_id", raw.DefaultWorkflowID)
	if err != nil {
		return CreateProject{}, err
	}

	defaultAgentProviderID, err := parseOptionalUUID("default_agent_provider_id", raw.DefaultAgentProviderID)
	if err != nil {
		return CreateProject{}, err
	}

	status, err := parseProjectStatus(raw.Status)
	if err != nil {
		return CreateProject{}, err
	}

	maxConcurrentAgents, err := parseMaxConcurrentAgents(raw.MaxConcurrentAgents)
	if err != nil {
		return CreateProject{}, err
	}

	return CreateProject{
		OrganizationID:         organizationID,
		Name:                   name,
		Slug:                   slug,
		Description:            strings.TrimSpace(raw.Description),
		Status:                 status,
		DefaultWorkflowID:      defaultWorkflowID,
		DefaultAgentProviderID: defaultAgentProviderID,
		MaxConcurrentAgents:    maxConcurrentAgents,
	}, nil
}

func ParseUpdateProject(id uuid.UUID, organizationID uuid.UUID, raw ProjectInput) (UpdateProject, error) {
	input, err := ParseCreateProject(organizationID, raw)
	if err != nil {
		return UpdateProject{}, err
	}

	return UpdateProject{
		ID:                     id,
		OrganizationID:         input.OrganizationID,
		Name:                   input.Name,
		Slug:                   input.Slug,
		Description:            input.Description,
		Status:                 input.Status,
		DefaultWorkflowID:      input.DefaultWorkflowID,
		DefaultAgentProviderID: input.DefaultAgentProviderID,
		MaxConcurrentAgents:    input.MaxConcurrentAgents,
	}, nil
}

func parseName(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}

	return trimmed, nil
}

func parseSlug(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if !slugPattern.MatchString(trimmed) {
		return "", fmt.Errorf("slug must match %q", slugPattern.String())
	}

	return trimmed, nil
}

func parseOptionalUUID(fieldName string, raw *string) (*uuid.UUID, error) {
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

func parseProjectStatus(raw string) (entproject.Status, error) {
	if strings.TrimSpace(raw) == "" {
		return entproject.DefaultStatus, nil
	}

	status := entproject.Status(strings.ToLower(strings.TrimSpace(raw)))
	if err := entproject.StatusValidator(status); err != nil {
		return "", fmt.Errorf("status must be one of planning, active, paused, archived")
	}

	return status, nil
}

func parseMaxConcurrentAgents(raw *int) (int, error) {
	if raw == nil {
		return entproject.DefaultMaxConcurrentAgents, nil
	}
	if *raw < 1 {
		return 0, fmt.Errorf("max_concurrent_agents must be greater than zero")
	}

	return *raw, nil
}
