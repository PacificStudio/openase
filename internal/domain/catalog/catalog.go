package catalog

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Organization struct {
	ID                     uuid.UUID
	Name                   string
	Slug                   string
	Status                 OrganizationStatus
	DefaultAgentProviderID *uuid.UUID
}

type Project struct {
	ID                     uuid.UUID
	OrganizationID         uuid.UUID
	Name                   string
	Slug                   string
	Description            string
	Status                 ProjectStatus
	DefaultWorkflowID      *uuid.UUID
	DefaultAgentProviderID *uuid.UUID
	AccessibleMachineIDs   []uuid.UUID
	MaxConcurrentAgents    int
}

type ProjectRepo struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	Name          string
	RepositoryURL string
	DefaultBranch string
	ClonePath     *string
	IsPrimary     bool
	Labels        []string
}

type TicketRepoScope struct {
	ID             uuid.UUID
	TicketID       uuid.UUID
	RepoID         uuid.UUID
	BranchName     string
	PullRequestURL *string
	PrStatus       TicketRepoScopePRStatus
	CiStatus       TicketRepoScopeCIStatus
	IsPrimaryScope bool
}

type OrganizationInput struct {
	Name                   string  `json:"name"`
	Slug                   string  `json:"slug"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id"`
}

type ProjectInput struct {
	Name                   string   `json:"name"`
	Slug                   string   `json:"slug"`
	Description            string   `json:"description"`
	Status                 string   `json:"status"`
	DefaultWorkflowID      *string  `json:"default_workflow_id"`
	DefaultAgentProviderID *string  `json:"default_agent_provider_id"`
	AccessibleMachineIDs   []string `json:"accessible_machine_ids"`
	MaxConcurrentAgents    *int     `json:"max_concurrent_agents"`
}

type ProjectRepoInput struct {
	Name          string   `json:"name"`
	RepositoryURL string   `json:"repository_url"`
	DefaultBranch string   `json:"default_branch"`
	ClonePath     *string  `json:"clone_path"`
	IsPrimary     *bool    `json:"is_primary"`
	Labels        []string `json:"labels"`
}

type TicketRepoScopeInput struct {
	RepoID         string  `json:"repo_id"`
	BranchName     *string `json:"branch_name"`
	PullRequestURL *string `json:"pull_request_url"`
	PrStatus       string  `json:"pr_status"`
	CiStatus       string  `json:"ci_status"`
	IsPrimaryScope *bool   `json:"is_primary_scope"`
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
	Status                 ProjectStatus
	DefaultWorkflowID      *uuid.UUID
	DefaultAgentProviderID *uuid.UUID
	AccessibleMachineIDs   []uuid.UUID
	MaxConcurrentAgents    int
}

type UpdateProject struct {
	ID                     uuid.UUID
	OrganizationID         uuid.UUID
	Name                   string
	Slug                   string
	Description            string
	Status                 ProjectStatus
	DefaultWorkflowID      *uuid.UUID
	DefaultAgentProviderID *uuid.UUID
	AccessibleMachineIDs   []uuid.UUID
	MaxConcurrentAgents    int
}

type CreateProjectRepo struct {
	ProjectID        uuid.UUID
	Name             string
	RepositoryURL    string
	DefaultBranch    string
	ClonePath        *string
	RequestedPrimary *bool
	Labels           []string
}

type UpdateProjectRepo struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	Name          string
	RepositoryURL string
	DefaultBranch string
	ClonePath     *string
	IsPrimary     bool
	Labels        []string
}

type CreateTicketRepoScope struct {
	ProjectID        uuid.UUID
	TicketID         uuid.UUID
	RepoID           uuid.UUID
	BranchName       *string
	PullRequestURL   *string
	PrStatus         TicketRepoScopePRStatus
	CiStatus         TicketRepoScopeCIStatus
	RequestedPrimary *bool
}

type UpdateTicketRepoScope struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	TicketID       uuid.UUID
	RepoID         uuid.UUID
	BranchName     *string
	PullRequestURL *string
	PrStatus       TicketRepoScopePRStatus
	CiStatus       TicketRepoScopeCIStatus
	IsPrimaryScope bool
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
	accessibleMachineIDs, err := parseUUIDList("accessible_machine_ids", raw.AccessibleMachineIDs)
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
		AccessibleMachineIDs:   accessibleMachineIDs,
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
		AccessibleMachineIDs:   input.AccessibleMachineIDs,
		MaxConcurrentAgents:    input.MaxConcurrentAgents,
	}, nil
}

func ParseCreateProjectRepo(projectID uuid.UUID, raw ProjectRepoInput) (CreateProjectRepo, error) {
	name, err := parseName("name", raw.Name)
	if err != nil {
		return CreateProjectRepo{}, err
	}

	repositoryURL, err := parseTrimmedRequired("repository_url", raw.RepositoryURL)
	if err != nil {
		return CreateProjectRepo{}, err
	}

	defaultBranch, err := parseDefaultBranch(raw.DefaultBranch)
	if err != nil {
		return CreateProjectRepo{}, err
	}

	clonePath := parseOptionalText(raw.ClonePath)

	labels, err := parseLabels(raw.Labels)
	if err != nil {
		return CreateProjectRepo{}, err
	}

	return CreateProjectRepo{
		ProjectID:        projectID,
		Name:             name,
		RepositoryURL:    repositoryURL,
		DefaultBranch:    defaultBranch,
		ClonePath:        clonePath,
		RequestedPrimary: raw.IsPrimary,
		Labels:           labels,
	}, nil
}

func ParseUpdateProjectRepo(id uuid.UUID, projectID uuid.UUID, raw ProjectRepoInput) (UpdateProjectRepo, error) {
	input, err := ParseCreateProjectRepo(projectID, raw)
	if err != nil {
		return UpdateProjectRepo{}, err
	}

	isPrimary := false
	if input.RequestedPrimary != nil {
		isPrimary = *input.RequestedPrimary
	}

	return UpdateProjectRepo{
		ID:            id,
		ProjectID:     input.ProjectID,
		Name:          input.Name,
		RepositoryURL: input.RepositoryURL,
		DefaultBranch: input.DefaultBranch,
		ClonePath:     input.ClonePath,
		IsPrimary:     isPrimary,
		Labels:        input.Labels,
	}, nil
}

func ParseCreateTicketRepoScope(projectID uuid.UUID, ticketID uuid.UUID, raw TicketRepoScopeInput) (CreateTicketRepoScope, error) {
	repoID, err := parseRequiredUUID("repo_id", raw.RepoID)
	if err != nil {
		return CreateTicketRepoScope{}, err
	}

	prStatus, err := parseTicketRepoScopePrStatus(raw.PrStatus)
	if err != nil {
		return CreateTicketRepoScope{}, err
	}

	ciStatus, err := parseTicketRepoScopeCiStatus(raw.CiStatus)
	if err != nil {
		return CreateTicketRepoScope{}, err
	}

	return CreateTicketRepoScope{
		ProjectID:        projectID,
		TicketID:         ticketID,
		RepoID:           repoID,
		BranchName:       parseOptionalText(raw.BranchName),
		PullRequestURL:   parseOptionalText(raw.PullRequestURL),
		PrStatus:         prStatus,
		CiStatus:         ciStatus,
		RequestedPrimary: raw.IsPrimaryScope,
	}, nil
}

func ParseUpdateTicketRepoScope(id uuid.UUID, projectID uuid.UUID, ticketID uuid.UUID, raw TicketRepoScopeInput) (UpdateTicketRepoScope, error) {
	input, err := ParseCreateTicketRepoScope(projectID, ticketID, raw)
	if err != nil {
		return UpdateTicketRepoScope{}, err
	}

	isPrimaryScope := false
	if input.RequestedPrimary != nil {
		isPrimaryScope = *input.RequestedPrimary
	}

	return UpdateTicketRepoScope{
		ID:             id,
		ProjectID:      input.ProjectID,
		TicketID:       input.TicketID,
		RepoID:         input.RepoID,
		BranchName:     input.BranchName,
		PullRequestURL: input.PullRequestURL,
		PrStatus:       input.PrStatus,
		CiStatus:       input.CiStatus,
		IsPrimaryScope: isPrimaryScope,
	}, nil
}

func parseName(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}

	return trimmed, nil
}

func parseTrimmedRequired(fieldName string, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}

	return trimmed, nil
}

func parseDefaultBranch(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "main", nil
	}

	return parseTrimmedRequired("default_branch", raw)
}

func parseOptionalText(raw *string) *string {
	if raw == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func parseLabels(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	labels := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for index, label := range raw {
		trimmed := strings.TrimSpace(label)
		if trimmed == "" {
			return nil, fmt.Errorf("labels[%d] must not be empty", index)
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		labels = append(labels, trimmed)
	}

	return labels, nil
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

func parseUUIDList(fieldName string, raw []string) ([]uuid.UUID, error) {
	parsed := make([]uuid.UUID, 0, len(raw))
	seen := make(map[uuid.UUID]struct{}, len(raw))
	for index, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, fmt.Errorf("%s[%d] must not be empty", fieldName, index)
		}
		value, err := uuid.Parse(trimmed)
		if err != nil {
			return nil, fmt.Errorf("%s[%d] must be a valid UUID", fieldName, index)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		parsed = append(parsed, value)
	}
	return parsed, nil
}

func parseProjectStatus(raw string) (ProjectStatus, error) {
	if strings.TrimSpace(raw) == "" {
		return DefaultProjectStatus, nil
	}

	status := ProjectStatus(strings.ToLower(strings.TrimSpace(raw)))
	if !status.IsValid() {
		return "", fmt.Errorf("status must be one of planning, active, paused, archived")
	}

	return status, nil
}

func parseMaxConcurrentAgents(raw *int) (int, error) {
	if raw == nil {
		return DefaultProjectMaxConcurrentAgents, nil
	}
	if *raw < 1 {
		return 0, fmt.Errorf("max_concurrent_agents must be greater than zero")
	}

	return *raw, nil
}

func parseTicketRepoScopePrStatus(raw string) (TicketRepoScopePRStatus, error) {
	if strings.TrimSpace(raw) == "" {
		return DefaultTicketRepoScopePRStatus, nil
	}

	status := TicketRepoScopePRStatus(strings.ToLower(strings.TrimSpace(raw)))
	if !status.IsValid() {
		return "", fmt.Errorf("pr_status must be one of none, open, changes_requested, approved, merged, closed")
	}

	return status, nil
}

func parseTicketRepoScopeCiStatus(raw string) (TicketRepoScopeCIStatus, error) {
	if strings.TrimSpace(raw) == "" {
		return DefaultTicketRepoScopeCIStatus, nil
	}

	status := TicketRepoScopeCIStatus(strings.ToLower(strings.TrimSpace(raw)))
	if !status.IsValid() {
		return "", fmt.Errorf("ci_status must be one of pending, passing, failing")
	}

	return status, nil
}
