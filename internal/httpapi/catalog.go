package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var errAPIResponseCommitted = errors.New("api response already committed")

type organizationResponse struct {
	ID                     string  `json:"id"`
	Name                   string  `json:"name"`
	Slug                   string  `json:"slug"`
	Status                 string  `json:"status"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id,omitempty"`
}

type projectResponse struct {
	ID                     string   `json:"id"`
	OrganizationID         string   `json:"organization_id"`
	Name                   string   `json:"name"`
	Slug                   string   `json:"slug"`
	Description            string   `json:"description"`
	Status                 string   `json:"status"`
	DefaultWorkflowID      *string  `json:"default_workflow_id,omitempty"`
	DefaultAgentProviderID *string  `json:"default_agent_provider_id,omitempty"`
	AccessibleMachineIDs   []string `json:"accessible_machine_ids,omitempty"`
	MaxConcurrentAgents    int      `json:"max_concurrent_agents"`
}

type machineResponse struct {
	ID              string         `json:"id"`
	OrganizationID  string         `json:"organization_id"`
	Name            string         `json:"name"`
	Host            string         `json:"host"`
	Port            int            `json:"port"`
	SSHUser         *string        `json:"ssh_user,omitempty"`
	SSHKeyPath      *string        `json:"ssh_key_path,omitempty"`
	Description     string         `json:"description"`
	Labels          []string       `json:"labels"`
	Status          string         `json:"status"`
	WorkspaceRoot   *string        `json:"workspace_root,omitempty"`
	AgentCLIPath    *string        `json:"agent_cli_path,omitempty"`
	EnvVars         []string       `json:"env_vars"`
	LastHeartbeatAt *string        `json:"last_heartbeat_at,omitempty"`
	Resources       map[string]any `json:"resources"`
}

type machineProbeResponse struct {
	CheckedAt string         `json:"checked_at"`
	Transport string         `json:"transport"`
	Output    string         `json:"output"`
	Resources map[string]any `json:"resources"`
}

type projectRepoResponse struct {
	ID               string   `json:"id"`
	ProjectID        string   `json:"project_id"`
	Name             string   `json:"name"`
	RepositoryURL    string   `json:"repository_url"`
	DefaultBranch    string   `json:"default_branch"`
	WorkspaceDirname string   `json:"workspace_dirname"`
	IsPrimary        bool     `json:"is_primary"`
	Labels           []string `json:"labels"`
}

type ticketRepoScopeResponse struct {
	ID             string  `json:"id"`
	TicketID       string  `json:"ticket_id"`
	RepoID         string  `json:"repo_id"`
	BranchName     string  `json:"branch_name"`
	PullRequestURL *string `json:"pull_request_url,omitempty"`
	PrStatus       string  `json:"pr_status"`
	CiStatus       string  `json:"ci_status"`
	IsPrimaryScope bool    `json:"is_primary_scope"`
}

type organizationPatchRequest struct {
	Name                   *string `json:"name"`
	Slug                   *string `json:"slug"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id"`
}

type projectPatchRequest struct {
	Name                   *string   `json:"name"`
	Slug                   *string   `json:"slug"`
	Description            *string   `json:"description"`
	Status                 *string   `json:"status"`
	DefaultWorkflowID      *string   `json:"default_workflow_id"`
	DefaultAgentProviderID *string   `json:"default_agent_provider_id"`
	AccessibleMachineIDs   *[]string `json:"accessible_machine_ids"`
	MaxConcurrentAgents    *int      `json:"max_concurrent_agents"`
}

type machinePatchRequest struct {
	Name          *string   `json:"name"`
	Host          *string   `json:"host"`
	Port          *int      `json:"port"`
	SSHUser       *string   `json:"ssh_user"`
	SSHKeyPath    *string   `json:"ssh_key_path"`
	Description   *string   `json:"description"`
	Labels        *[]string `json:"labels"`
	Status        *string   `json:"status"`
	WorkspaceRoot *string   `json:"workspace_root"`
	AgentCLIPath  *string   `json:"agent_cli_path"`
	EnvVars       *[]string `json:"env_vars"`
}

type projectRepoPatchRequest struct {
	Name             *string   `json:"name"`
	RepositoryURL    *string   `json:"repository_url"`
	DefaultBranch    *string   `json:"default_branch"`
	WorkspaceDirname *string   `json:"workspace_dirname"`
	IsPrimary        *bool     `json:"is_primary"`
	Labels           *[]string `json:"labels"`
}

type ticketRepoScopePatchRequest struct {
	BranchName     *string `json:"branch_name"`
	PullRequestURL *string `json:"pull_request_url"`
	PrStatus       *string `json:"pr_status"`
	CiStatus       *string `json:"ci_status"`
	IsPrimaryScope *bool   `json:"is_primary_scope"`
}

func (s *Server) registerCatalogRoutes(api *echo.Group) {
	api.GET("/app-context", s.handleGetAppContext)
	api.GET("/orgs", s.listOrganizations)
	api.POST("/orgs", s.createOrganization)
	api.GET("/orgs/:orgId", s.getOrganization)
	api.PATCH("/orgs/:orgId", s.patchOrganization)
	api.DELETE("/orgs/:orgId", s.archiveOrganization)
	api.GET("/orgs/:orgId/projects", s.listProjects)
	api.POST("/orgs/:orgId/projects", s.createProject)
	api.GET("/orgs/:orgId/machines", s.listMachines)
	api.POST("/orgs/:orgId/machines", s.createMachine)
	api.GET("/orgs/:orgId/providers", s.listAgentProviders)
	api.POST("/orgs/:orgId/providers", s.createAgentProvider)
	api.GET("/machines/:machineId", s.getMachine)
	api.PATCH("/machines/:machineId", s.patchMachine)
	api.DELETE("/machines/:machineId", s.deleteMachine)
	api.POST("/machines/:machineId/test", s.testMachine)
	api.GET("/machines/:machineId/resources", s.getMachineResources)
	api.GET("/projects/:projectId", s.getProject)
	api.PATCH("/projects/:projectId", s.patchProject)
	api.DELETE("/projects/:projectId", s.archiveProject)
	api.GET("/projects/:projectId/repos", s.listProjectRepos)
	api.POST("/projects/:projectId/repos", s.createProjectRepo)
	api.PATCH("/projects/:projectId/repos/:repoId", s.patchProjectRepo)
	api.DELETE("/projects/:projectId/repos/:repoId", s.deleteProjectRepo)
	api.GET("/projects/:projectId/tickets/:ticketId/repo-scopes", s.listTicketRepoScopes)
	api.POST("/projects/:projectId/tickets/:ticketId/repo-scopes", s.createTicketRepoScope)
	api.PATCH("/projects/:projectId/tickets/:ticketId/repo-scopes/:scopeId", s.patchTicketRepoScope)
	api.DELETE("/projects/:projectId/tickets/:ticketId/repo-scopes/:scopeId", s.deleteTicketRepoScope)
	api.GET("/projects/:projectId/agents", s.listAgents)
	api.GET("/projects/:projectId/agent-runs", s.listAgentRuns)
	api.GET("/projects/:projectId/activity", s.listActivityEvents)
	api.GET("/projects/:projectId/agents/:agentId/output", s.listAgentOutput)
	api.GET("/projects/:projectId/agents/:agentId/steps", s.listAgentSteps)
	api.POST("/projects/:projectId/agents", s.createAgent)
	api.PATCH("/providers/:providerId", s.patchAgentProvider)
	api.GET("/agents/:agentId", s.getAgent)
	api.POST("/agents/:agentId/pause", s.pauseAgent)
	api.POST("/agents/:agentId/resume", s.resumeAgent)
	api.DELETE("/agents/:agentId", s.deleteAgent)
}

func (s *Server) listOrganizations(c echo.Context) error {
	items, err := s.catalog.ListOrganizations(c.Request().Context())
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"organizations": mapOrganizationResponses(items),
	})
}

func (s *Server) createOrganization(c echo.Context) error {
	var request domain.OrganizationInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateOrganization(request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateOrganization(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"organization": mapOrganizationResponse(item),
	})
}

func (s *Server) getOrganization(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	item, err := s.catalog.GetOrganization(c.Request().Context(), orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"organization": mapOrganizationResponse(item),
	})
}

func (s *Server) patchOrganization(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetOrganization(c.Request().Context(), orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch organizationPatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	request := domain.OrganizationInput{
		Name:                   current.Name,
		Slug:                   current.Slug,
		DefaultAgentProviderID: uuidToStringPointer(current.DefaultAgentProviderID),
	}
	if patch.Name != nil {
		request.Name = *patch.Name
	}
	if patch.Slug != nil {
		request.Slug = *patch.Slug
	}
	if patch.DefaultAgentProviderID != nil {
		request.DefaultAgentProviderID = patch.DefaultAgentProviderID
	}

	input, err := domain.ParseUpdateOrganization(orgID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateOrganization(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"organization": mapOrganizationResponse(item),
	})
}

func (s *Server) archiveOrganization(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	item, err := s.catalog.ArchiveOrganization(c.Request().Context(), orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"organization": mapOrganizationResponse(item),
	})
}

func (s *Server) listProjects(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListProjects(c.Request().Context(), orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"projects": mapProjectResponses(items),
	})
}

func (s *Server) listMachines(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListMachines(c.Request().Context(), orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machines": mapMachineResponses(items),
	})
}

func (s *Server) createProject(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	var request domain.ProjectInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateProject(orgID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateProject(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"project": mapProjectResponse(item),
	})
}

func (s *Server) createMachine(c echo.Context) error {
	orgID, err := parseUUIDPathParam(c, "orgId")
	if err != nil {
		return err
	}

	var request domain.MachineInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateMachine(orgID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateMachine(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"machine": mapMachineResponse(item),
	})
}

func (s *Server) getMachine(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	item, err := s.catalog.GetMachine(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine": mapMachineResponse(item),
	})
}

func (s *Server) patchMachine(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetMachine(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch machinePatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	request := domain.MachineInput{
		Name:          current.Name,
		Host:          current.Host,
		Port:          intPointer(current.Port),
		SSHUser:       current.SSHUser,
		SSHKeyPath:    current.SSHKeyPath,
		Description:   current.Description,
		Labels:        cloneStringSlice(current.Labels),
		Status:        current.Status.String(),
		WorkspaceRoot: current.WorkspaceRoot,
		AgentCLIPath:  current.AgentCLIPath,
		EnvVars:       cloneStringSlice(current.EnvVars),
	}
	if patch.Name != nil {
		request.Name = *patch.Name
	}
	if patch.Host != nil {
		request.Host = *patch.Host
	}
	if patch.Port != nil {
		request.Port = patch.Port
	}
	if patch.SSHUser != nil {
		request.SSHUser = patch.SSHUser
	}
	if patch.SSHKeyPath != nil {
		request.SSHKeyPath = patch.SSHKeyPath
	}
	if patch.Description != nil {
		request.Description = *patch.Description
	}
	if patch.Labels != nil {
		request.Labels = *patch.Labels
	}
	if patch.Status != nil {
		request.Status = *patch.Status
	}
	if patch.WorkspaceRoot != nil {
		request.WorkspaceRoot = patch.WorkspaceRoot
	}
	if patch.AgentCLIPath != nil {
		request.AgentCLIPath = patch.AgentCLIPath
	}
	if patch.EnvVars != nil {
		request.EnvVars = *patch.EnvVars
	}

	input, err := domain.ParseUpdateMachine(machineID, current.OrganizationID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateMachine(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine": mapMachineResponse(item),
	})
}

func (s *Server) deleteMachine(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	item, err := s.catalog.DeleteMachine(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine": mapMachineResponse(item),
	})
}

func (s *Server) testMachine(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	item, probe, err := s.catalog.TestMachineConnection(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine": mapMachineResponse(item),
		"probe":   mapMachineProbeResponse(probe),
	})
}

func (s *Server) getMachineResources(c echo.Context) error {
	machineID, err := parseUUIDPathParam(c, "machineId")
	if err != nil {
		return err
	}

	item, err := s.catalog.GetMachine(c.Request().Context(), machineID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"machine_id":               item.ID.String(),
		"status":                   item.Status.String(),
		"last_heartbeat_at":        timeToStringPointer(item.LastHeartbeatAt),
		"resources":                cloneMap(item.Resources),
		"environment_provisioning": domain.PlanMachineEnvironmentProvisioning(item),
	})
}

func (s *Server) getProject(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	item, err := s.catalog.GetProject(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"project": mapProjectResponse(item),
	})
}

func (s *Server) patchProject(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetProject(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch projectPatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	request := domain.ProjectInput{
		Name:                   current.Name,
		Slug:                   current.Slug,
		Description:            current.Description,
		Status:                 current.Status.String(),
		DefaultWorkflowID:      uuidToStringPointer(current.DefaultWorkflowID),
		DefaultAgentProviderID: uuidToStringPointer(current.DefaultAgentProviderID),
		AccessibleMachineIDs:   uuidSliceToStrings(current.AccessibleMachineIDs),
		MaxConcurrentAgents:    intPointer(current.MaxConcurrentAgents),
	}
	if patch.Name != nil {
		request.Name = *patch.Name
	}
	if patch.Slug != nil {
		request.Slug = *patch.Slug
	}
	if patch.Description != nil {
		request.Description = *patch.Description
	}
	if patch.Status != nil {
		request.Status = *patch.Status
	}
	if patch.DefaultWorkflowID != nil {
		request.DefaultWorkflowID = patch.DefaultWorkflowID
	}
	if patch.DefaultAgentProviderID != nil {
		request.DefaultAgentProviderID = patch.DefaultAgentProviderID
	}
	if patch.AccessibleMachineIDs != nil {
		request.AccessibleMachineIDs = cloneStringSlice(*patch.AccessibleMachineIDs)
	}
	if patch.MaxConcurrentAgents != nil {
		request.MaxConcurrentAgents = patch.MaxConcurrentAgents
	}

	input, err := domain.ParseUpdateProject(projectID, current.OrganizationID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateProject(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"project": mapProjectResponse(item),
	})
}

func (s *Server) archiveProject(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	item, err := s.catalog.ArchiveProject(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"project": mapProjectResponse(item),
	})
}

func (s *Server) listProjectRepos(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListProjectRepos(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repos": mapProjectRepoResponses(items),
	})
}

func (s *Server) createProjectRepo(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}

	var request domain.ProjectRepoInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateProjectRepo(projectID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateProjectRepo(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"repo": mapProjectRepoResponse(item),
	})
}

func (s *Server) patchProjectRepo(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	repoID, err := parseUUIDPathParam(c, "repoId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetProjectRepo(c.Request().Context(), projectID, repoID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch projectRepoPatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	request := domain.ProjectRepoInput{
		Name:             current.Name,
		RepositoryURL:    current.RepositoryURL,
		DefaultBranch:    current.DefaultBranch,
		WorkspaceDirname: stringPointer(current.WorkspaceDirname),
		IsPrimary:        boolPointer(current.IsPrimary),
		Labels:           append([]string(nil), current.Labels...),
	}
	if patch.Name != nil {
		request.Name = *patch.Name
	}
	if patch.RepositoryURL != nil {
		request.RepositoryURL = *patch.RepositoryURL
	}
	if patch.DefaultBranch != nil {
		request.DefaultBranch = *patch.DefaultBranch
	}
	if patch.WorkspaceDirname != nil {
		request.WorkspaceDirname = patch.WorkspaceDirname
	}
	if patch.IsPrimary != nil {
		request.IsPrimary = patch.IsPrimary
	}
	if patch.Labels != nil {
		request.Labels = *patch.Labels
	}

	input, err := domain.ParseUpdateProjectRepo(repoID, projectID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateProjectRepo(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repo": mapProjectRepoResponse(item),
	})
}

func (s *Server) deleteProjectRepo(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	repoID, err := parseUUIDPathParam(c, "repoId")
	if err != nil {
		return err
	}

	item, err := s.catalog.DeleteProjectRepo(c.Request().Context(), projectID, repoID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repo": mapProjectRepoResponse(item),
	})
}

func (s *Server) listTicketRepoScopes(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	ticketID, err := parseUUIDPathParam(c, "ticketId")
	if err != nil {
		return err
	}

	items, err := s.catalog.ListTicketRepoScopes(c.Request().Context(), projectID, ticketID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repo_scopes": mapTicketRepoScopeResponses(items),
	})
}

func (s *Server) createTicketRepoScope(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	ticketID, err := parseUUIDPathParam(c, "ticketId")
	if err != nil {
		return err
	}

	var request domain.TicketRepoScopeInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateTicketRepoScope(projectID, ticketID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateTicketRepoScope(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"repo_scope": mapTicketRepoScopeResponse(item),
	})
}

func (s *Server) patchTicketRepoScope(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	ticketID, err := parseUUIDPathParam(c, "ticketId")
	if err != nil {
		return err
	}
	scopeID, err := parseUUIDPathParam(c, "scopeId")
	if err != nil {
		return err
	}

	current, err := s.catalog.GetTicketRepoScope(c.Request().Context(), projectID, ticketID, scopeID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var patch ticketRepoScopePatchRequest
	if err := decodeJSON(c, &patch); err != nil {
		return err
	}

	request := domain.TicketRepoScopeInput{
		RepoID:         current.RepoID.String(),
		BranchName:     stringPointer(current.BranchName),
		PullRequestURL: current.PullRequestURL,
		PrStatus:       current.PrStatus.String(),
		CiStatus:       current.CiStatus.String(),
		IsPrimaryScope: boolPointer(current.IsPrimaryScope),
	}
	if patch.BranchName != nil {
		request.BranchName = patch.BranchName
	}
	if patch.PullRequestURL != nil {
		request.PullRequestURL = patch.PullRequestURL
	}
	if patch.PrStatus != nil {
		request.PrStatus = *patch.PrStatus
	}
	if patch.CiStatus != nil {
		request.CiStatus = *patch.CiStatus
	}
	if patch.IsPrimaryScope != nil {
		request.IsPrimaryScope = patch.IsPrimaryScope
	}

	input, err := domain.ParseUpdateTicketRepoScope(scopeID, projectID, ticketID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateTicketRepoScope(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repo_scope": mapTicketRepoScopeResponse(item),
	})
}

func (s *Server) deleteTicketRepoScope(c echo.Context) error {
	projectID, err := parseUUIDPathParam(c, "projectId")
	if err != nil {
		return err
	}
	ticketID, err := parseUUIDPathParam(c, "ticketId")
	if err != nil {
		return err
	}
	scopeID, err := parseUUIDPathParam(c, "scopeId")
	if err != nil {
		return err
	}

	item, err := s.catalog.DeleteTicketRepoScope(c.Request().Context(), projectID, ticketID, scopeID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"repo_scope": mapTicketRepoScopeResponse(item),
	})
}

func decodeJSON(c echo.Context, target any) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if writeErr := c.JSON(http.StatusBadRequest, errorResponse(fmt.Sprintf("invalid JSON body: %v", err))); writeErr != nil {
			return writeErr
		}
		return errAPIResponseCommitted
	}
	if decoder.More() {
		if writeErr := c.JSON(http.StatusBadRequest, errorResponse("invalid JSON body: multiple JSON values are not allowed")); writeErr != nil {
			return writeErr
		}
		return errAPIResponseCommitted
	}

	return nil
}

func parseUUIDPathParam(c echo.Context, name string) (uuid.UUID, error) {
	parsed, err := parseUUIDPathParamValue(c, name)
	if err != nil {
		if writeErr := c.JSON(http.StatusBadRequest, errorResponse(err.Error())); writeErr != nil {
			return uuid.UUID{}, writeErr
		}
		return uuid.UUID{}, errAPIResponseCommitted
	}

	return parsed, nil
}

func parseUUIDPathParamValue(c echo.Context, name string) (uuid.UUID, error) {
	raw := strings.TrimSpace(c.Param(name))
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%s must be a valid UUID", name)
	}

	return parsed, nil
}

func writeCatalogError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, catalogservice.ErrInvalidInput):
		return c.JSON(http.StatusBadRequest, errorResponse(catalogErrorMessage(err)))
	case errors.Is(err, catalogservice.ErrNotFound):
		return c.JSON(http.StatusNotFound, errorResponse("resource not found"))
	case errors.Is(err, catalogservice.ErrConflict):
		return c.JSON(http.StatusConflict, errorResponse("resource conflict"))
	case errors.Is(err, catalogservice.ErrMachineProbeFailed):
		return c.JSON(http.StatusBadGateway, errorResponse(catalogErrorMessage(err)))
	case errors.Is(err, catalogservice.ErrMachineTestingUnavailable):
		return c.JSON(http.StatusServiceUnavailable, errorResponse(catalogErrorMessage(err)))
	default:
		return c.JSON(http.StatusInternalServerError, errorResponse("internal server error"))
	}
}

func errorResponse(message string) map[string]string {
	return map[string]string{"error": message}
}

func catalogErrorMessage(err error) string {
	prefix := catalogservice.ErrInvalidInput.Error() + ": "
	if strings.HasPrefix(err.Error(), prefix) {
		return strings.TrimPrefix(err.Error(), prefix)
	}

	return err.Error()
}

func catalogConflictMessage(err error) string {
	prefix := catalogservice.ErrConflict.Error() + ": "
	if strings.HasPrefix(err.Error(), prefix) {
		return strings.TrimPrefix(err.Error(), prefix)
	}

	return err.Error()
}

func mapOrganizationResponses(items []domain.Organization) []organizationResponse {
	response := make([]organizationResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapOrganizationResponse(item))
	}

	return response
}

func mapOrganizationResponse(item domain.Organization) organizationResponse {
	return organizationResponse{
		ID:                     item.ID.String(),
		Name:                   item.Name,
		Slug:                   item.Slug,
		Status:                 string(item.Status),
		DefaultAgentProviderID: uuidToStringPointer(item.DefaultAgentProviderID),
	}
}

func mapProjectResponses(items []domain.Project) []projectResponse {
	response := make([]projectResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapProjectResponse(item))
	}

	return response
}

func mapMachineResponses(items []domain.Machine) []machineResponse {
	response := make([]machineResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapMachineResponse(item))
	}

	return response
}

func mapProjectResponse(item domain.Project) projectResponse {
	return projectResponse{
		ID:                     item.ID.String(),
		OrganizationID:         item.OrganizationID.String(),
		Name:                   item.Name,
		Slug:                   item.Slug,
		Description:            item.Description,
		Status:                 item.Status.String(),
		DefaultWorkflowID:      uuidToStringPointer(item.DefaultWorkflowID),
		DefaultAgentProviderID: uuidToStringPointer(item.DefaultAgentProviderID),
		AccessibleMachineIDs:   uuidSliceToStrings(item.AccessibleMachineIDs),
		MaxConcurrentAgents:    item.MaxConcurrentAgents,
	}
}

func mapMachineResponse(item domain.Machine) machineResponse {
	return machineResponse{
		ID:              item.ID.String(),
		OrganizationID:  item.OrganizationID.String(),
		Name:            item.Name,
		Host:            item.Host,
		Port:            item.Port,
		SSHUser:         item.SSHUser,
		SSHKeyPath:      item.SSHKeyPath,
		Description:     item.Description,
		Labels:          cloneStringSlice(item.Labels),
		Status:          item.Status.String(),
		WorkspaceRoot:   item.WorkspaceRoot,
		AgentCLIPath:    item.AgentCLIPath,
		EnvVars:         cloneStringSlice(item.EnvVars),
		LastHeartbeatAt: timeToStringPointer(item.LastHeartbeatAt),
		Resources:       cloneMap(item.Resources),
	}
}

func mapMachineProbeResponse(item domain.MachineProbe) machineProbeResponse {
	return machineProbeResponse{
		CheckedAt: item.CheckedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Transport: item.Transport,
		Output:    item.Output,
		Resources: cloneMap(item.Resources),
	}
}

func mapProjectRepoResponses(items []domain.ProjectRepo) []projectRepoResponse {
	response := make([]projectRepoResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapProjectRepoResponse(item))
	}

	return response
}

func mapProjectRepoResponse(item domain.ProjectRepo) projectRepoResponse {
	return projectRepoResponse{
		ID:               item.ID.String(),
		ProjectID:        item.ProjectID.String(),
		Name:             item.Name,
		RepositoryURL:    item.RepositoryURL,
		DefaultBranch:    item.DefaultBranch,
		WorkspaceDirname: item.WorkspaceDirname,
		IsPrimary:        item.IsPrimary,
		Labels:           cloneStringSlice(item.Labels),
	}
}

func mapTicketRepoScopeResponses(items []domain.TicketRepoScope) []ticketRepoScopeResponse {
	response := make([]ticketRepoScopeResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapTicketRepoScopeResponse(item))
	}

	return response
}

func mapTicketRepoScopeResponse(item domain.TicketRepoScope) ticketRepoScopeResponse {
	return ticketRepoScopeResponse{
		ID:             item.ID.String(),
		TicketID:       item.TicketID.String(),
		RepoID:         item.RepoID.String(),
		BranchName:     item.BranchName,
		PullRequestURL: item.PullRequestURL,
		PrStatus:       item.PrStatus.String(),
		CiStatus:       item.CiStatus.String(),
		IsPrimaryScope: item.IsPrimaryScope,
	}
}

func uuidToStringPointer(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}

	text := value.String()
	return &text
}

func uuidSliceToStrings(values []uuid.UUID) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, value.String())
	}
	return items
}

func intPointer(value int) *int {
	copied := value
	return &copied
}

func stringPointer(value string) *string {
	copied := value
	return &copied
}

func boolPointer(value bool) *bool {
	copied := value
	return &copied
}
