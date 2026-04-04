package catalog

import (
	"fmt"
	"regexp"
	"strings"

	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
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
	DefaultAgentProviderID *uuid.UUID
	AccessibleMachineIDs   []uuid.UUID
	MaxConcurrentAgents    int
	AgentRunSummaryPrompt  string
}

type ProjectRepo struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	Name             string
	RepositoryURL    string
	DefaultBranch    string
	WorkspaceDirname string
	Labels           []string
}

type TicketRepoScope struct {
	ID             uuid.UUID
	TicketID       uuid.UUID
	RepoID         uuid.UUID
	BranchName     string
	PullRequestURL *string
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
	DefaultAgentProviderID *string  `json:"default_agent_provider_id"`
	AccessibleMachineIDs   []string `json:"accessible_machine_ids"`
	MaxConcurrentAgents    *int     `json:"max_concurrent_agents"`
	AgentRunSummaryPrompt  *string  `json:"agent_run_summary_prompt"`
}

type ProjectRepoInput struct {
	Name             string   `json:"name"`
	RepositoryURL    string   `json:"repository_url"`
	DefaultBranch    string   `json:"default_branch"`
	WorkspaceDirname *string  `json:"workspace_dirname"`
	Labels           []string `json:"labels"`
}

type TicketRepoScopeInput struct {
	RepoID         string  `json:"repo_id"`
	BranchName     *string `json:"branch_name"`
	PullRequestURL *string `json:"pull_request_url"`
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
	DefaultAgentProviderID *uuid.UUID
	AccessibleMachineIDs   []uuid.UUID
	MaxConcurrentAgents    int
	AgentRunSummaryPrompt  string
}

type UpdateProject struct {
	ID                     uuid.UUID
	OrganizationID         uuid.UUID
	Name                   string
	Slug                   string
	Description            string
	Status                 ProjectStatus
	DefaultAgentProviderID *uuid.UUID
	AccessibleMachineIDs   []uuid.UUID
	MaxConcurrentAgents    int
	AgentRunSummaryPrompt  string
}

type CreateProjectRepo struct {
	ProjectID        uuid.UUID
	Name             string
	RepositoryURL    string
	DefaultBranch    string
	WorkspaceDirname string
	Labels           []string
}

type UpdateProjectRepo struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	Name             string
	RepositoryURL    string
	DefaultBranch    string
	WorkspaceDirname string
	Labels           []string
}

type CreateTicketRepoScope struct {
	ProjectID      uuid.UUID
	TicketID       uuid.UUID
	RepoID         uuid.UUID
	BranchName     *string
	PullRequestURL *string
}

type UpdateTicketRepoScope struct {
	ID             uuid.UUID
	ProjectID      uuid.UUID
	TicketID       uuid.UUID
	RepoID         uuid.UUID
	BranchName     *string
	BranchNameSet  bool
	PullRequestURL *string
	PullRequestSet bool
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
		DefaultAgentProviderID: defaultAgentProviderID,
		AccessibleMachineIDs:   accessibleMachineIDs,
		MaxConcurrentAgents:    maxConcurrentAgents,
		AgentRunSummaryPrompt:  strings.TrimSpace(derefString(raw.AgentRunSummaryPrompt)),
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
		DefaultAgentProviderID: input.DefaultAgentProviderID,
		AccessibleMachineIDs:   input.AccessibleMachineIDs,
		MaxConcurrentAgents:    input.MaxConcurrentAgents,
		AgentRunSummaryPrompt:  input.AgentRunSummaryPrompt,
	}, nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func ParseCreateProjectRepo(projectID uuid.UUID, raw ProjectRepoInput) (CreateProjectRepo, error) {
	name, err := parseName("name", raw.Name)
	if err != nil {
		return CreateProjectRepo{}, err
	}

	repositoryURL, err := parseProjectRepoRepositoryURL("repository_url", raw.RepositoryURL)
	if err != nil {
		return CreateProjectRepo{}, err
	}

	defaultBranch := parseDefaultBranch(raw.DefaultBranch)

	workspaceDirname, err := parseWorkspaceDirname(name, raw.WorkspaceDirname)
	if err != nil {
		return CreateProjectRepo{}, err
	}

	labels, err := parseLabels(raw.Labels)
	if err != nil {
		return CreateProjectRepo{}, err
	}

	return CreateProjectRepo{
		ProjectID:        projectID,
		Name:             name,
		RepositoryURL:    repositoryURL,
		DefaultBranch:    defaultBranch,
		WorkspaceDirname: workspaceDirname,
		Labels:           labels,
	}, nil
}

func ParseUpdateProjectRepo(id uuid.UUID, projectID uuid.UUID, raw ProjectRepoInput) (UpdateProjectRepo, error) {
	input, err := ParseCreateProjectRepo(projectID, raw)
	if err != nil {
		return UpdateProjectRepo{}, err
	}

	return UpdateProjectRepo{
		ID:               id,
		ProjectID:        input.ProjectID,
		Name:             input.Name,
		RepositoryURL:    input.RepositoryURL,
		DefaultBranch:    input.DefaultBranch,
		WorkspaceDirname: input.WorkspaceDirname,
		Labels:           input.Labels,
	}, nil
}

func parseProjectRepoRepositoryURL(fieldName string, raw string) (string, error) {
	repositoryURL, err := parseTrimmedRequired(fieldName, raw)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(repositoryURL)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "git@github.com:") || strings.HasPrefix(lower, "ssh://git@github.com/") {
		return "", fmt.Errorf("%s must use https://github.com/...git; ssh GitHub URLs are not allowed", fieldName)
	}

	if normalized, ok := githubauthdomain.NormalizeGitHubRepositoryURL(trimmed); ok {
		return normalized, nil
	}

	return trimmed, nil
}

func ParseCreateTicketRepoScope(projectID uuid.UUID, ticketID uuid.UUID, raw TicketRepoScopeInput) (CreateTicketRepoScope, error) {
	repoID, err := parseRequiredUUID("repo_id", raw.RepoID)
	if err != nil {
		return CreateTicketRepoScope{}, err
	}

	return CreateTicketRepoScope{
		ProjectID:      projectID,
		TicketID:       ticketID,
		RepoID:         repoID,
		BranchName:     parseOptionalText(raw.BranchName),
		PullRequestURL: parseOptionalText(raw.PullRequestURL),
	}, nil
}

func ParseUpdateTicketRepoScope(id uuid.UUID, projectID uuid.UUID, ticketID uuid.UUID, raw TicketRepoScopeInput) (UpdateTicketRepoScope, error) {
	input, err := ParseCreateTicketRepoScope(projectID, ticketID, raw)
	if err != nil {
		return UpdateTicketRepoScope{}, err
	}

	return UpdateTicketRepoScope{
		ID:             id,
		ProjectID:      input.ProjectID,
		TicketID:       input.TicketID,
		RepoID:         input.RepoID,
		BranchName:     input.BranchName,
		BranchNameSet:  raw.BranchName != nil,
		PullRequestURL: input.PullRequestURL,
		PullRequestSet: raw.PullRequestURL != nil,
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

func parseDefaultBranch(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "main"
	}

	return strings.TrimSpace(raw)
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

func parseWorkspaceDirname(repoName string, raw *string) (string, error) {
	if raw == nil {
		return repoName, nil
	}

	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return repoName, nil
	}
	if strings.HasPrefix(trimmed, "/") {
		return "", fmt.Errorf("workspace_dirname must be relative")
	}
	if strings.Contains(trimmed, "\\") {
		return "", fmt.Errorf("workspace_dirname must use forward slashes")
	}
	if strings.Contains(trimmed, "..") {
		return "", fmt.Errorf("workspace_dirname must stay inside the workspace")
	}

	cleaned := strings.TrimPrefix(trimmed, "./")
	if cleaned == "" {
		return "", fmt.Errorf("workspace_dirname must not be empty")
	}
	if strings.Contains(cleaned, " ") {
		return "", fmt.Errorf("workspace_dirname must not contain spaces")
	}

	return cleaned, nil
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
	if raw == "" {
		return DefaultProjectStatus, nil
	}

	status := ProjectStatus(raw)
	if !status.IsValid() {
		return "", fmt.Errorf("status must be one of Backlog, Planned, In Progress, Completed, Canceled, Archived")
	}

	return status, nil
}

func parseMaxConcurrentAgents(raw *int) (int, error) {
	if raw == nil {
		return 0, nil
	}
	if *raw < 0 {
		return 0, fmt.Errorf("max_concurrent_agents must be greater than or equal to zero")
	}

	return *raw, nil
}
