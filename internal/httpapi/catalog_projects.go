package httpapi

import (
	"net/http"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerProjectRoutes(api *echo.Group) {
	api.GET("/orgs/:orgId/summary", s.handleGetOrganizationSummary)
	api.GET("/orgs/:orgId/projects", s.listProjects)
	api.POST("/orgs/:orgId/projects", s.createProject)
	api.GET("/projects/:projectId", s.getProject)
	api.PATCH("/projects/:projectId", s.patchProject)
	api.DELETE("/projects/:projectId", s.archiveProject)
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

	input, err := parseProjectPatchRequest(projectID, current, patch)
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
