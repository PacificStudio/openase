package httpapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type appContextResponse struct {
	Organizations []organizationResponse  `json:"organizations"`
	Projects      []projectResponse       `json:"projects"`
	Providers     []agentProviderResponse `json:"providers"`
	AgentCount    int                     `json:"agent_count"`
}

func (s *Server) registerAppContextRoutes(api *echo.Group) {
	api.GET("/app-context", s.handleGetAppContext)
}

func (s *Server) handleGetAppContext(c echo.Context) error {
	organizations, err := s.catalog.ListOrganizations(c.Request().Context())
	if err != nil {
		return writeCatalogError(c, err)
	}

	response := appContextResponse{
		Organizations: mapOrganizationResponses(organizations),
		Projects:      []projectResponse{},
		Providers:     []agentProviderResponse{},
		AgentCount:    0,
	}

	orgID, err := parseOptionalUUIDQueryParam("org_id", c.QueryParam("org_id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ORG_ID", err.Error())
	}
	if orgID == nil {
		return c.JSON(http.StatusOK, response)
	}

	projects, err := s.catalog.ListProjects(c.Request().Context(), *orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}
	providers, err := s.catalog.ListAgentProviders(c.Request().Context(), *orgID)
	if err != nil {
		return writeCatalogError(c, err)
	}
	response.Projects = mapProjectResponses(projects)
	response.Providers = mapAgentProviderResponses(providers)

	projectID, err := parseOptionalUUIDQueryParam("project_id", c.QueryParam("project_id"))
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if projectID == nil {
		return c.JSON(http.StatusOK, response)
	}

	agents, err := s.catalog.ListAgents(c.Request().Context(), *projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}
	response.AgentCount = len(agents)

	return c.JSON(http.StatusOK, response)
}
