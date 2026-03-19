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

type organizationResponse struct {
	ID                     string  `json:"id"`
	Name                   string  `json:"name"`
	Slug                   string  `json:"slug"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id,omitempty"`
}

type projectResponse struct {
	ID                     string  `json:"id"`
	OrganizationID         string  `json:"organization_id"`
	Name                   string  `json:"name"`
	Slug                   string  `json:"slug"`
	Description            string  `json:"description"`
	Status                 string  `json:"status"`
	DefaultWorkflowID      *string `json:"default_workflow_id,omitempty"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id,omitempty"`
	MaxConcurrentAgents    int     `json:"max_concurrent_agents"`
}

type organizationPatchRequest struct {
	Name                   *string `json:"name"`
	Slug                   *string `json:"slug"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id"`
}

type projectPatchRequest struct {
	Name                   *string `json:"name"`
	Slug                   *string `json:"slug"`
	Description            *string `json:"description"`
	Status                 *string `json:"status"`
	DefaultWorkflowID      *string `json:"default_workflow_id"`
	DefaultAgentProviderID *string `json:"default_agent_provider_id"`
	MaxConcurrentAgents    *int    `json:"max_concurrent_agents"`
}

func (s *Server) registerCatalogRoutes(api *echo.Group) {
	api.GET("/orgs", s.listOrganizations)
	api.POST("/orgs", s.createOrganization)
	api.GET("/orgs/:orgId", s.getOrganization)
	api.PATCH("/orgs/:orgId", s.patchOrganization)
	api.GET("/orgs/:orgId/projects", s.listProjects)
	api.POST("/orgs/:orgId/projects", s.createProject)
	api.GET("/projects/:projectId", s.getProject)
	api.PATCH("/projects/:projectId", s.patchProject)
	api.DELETE("/projects/:projectId", s.archiveProject)
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

	request := domain.ProjectInput{
		Name:                   current.Name,
		Slug:                   current.Slug,
		Description:            current.Description,
		Status:                 current.Status.String(),
		DefaultWorkflowID:      uuidToStringPointer(current.DefaultWorkflowID),
		DefaultAgentProviderID: uuidToStringPointer(current.DefaultAgentProviderID),
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

func decodeJSON(c echo.Context, target any) error {
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(fmt.Sprintf("invalid JSON body: %v", err)))
	}
	if decoder.More() {
		return c.JSON(http.StatusBadRequest, errorResponse("invalid JSON body: multiple JSON values are not allowed"))
	}

	return nil
}

func parseUUIDPathParam(c echo.Context, name string) (uuid.UUID, error) {
	raw := strings.TrimSpace(c.Param(name))
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, c.JSON(http.StatusBadRequest, errorResponse(fmt.Sprintf("%s must be a valid UUID", name)))
	}

	return parsed, nil
}

func writeCatalogError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, catalogservice.ErrNotFound):
		return c.JSON(http.StatusNotFound, errorResponse("resource not found"))
	case errors.Is(err, catalogservice.ErrConflict):
		return c.JSON(http.StatusConflict, errorResponse("resource conflict"))
	default:
		return c.JSON(http.StatusInternalServerError, errorResponse("internal server error"))
	}
}

func errorResponse(message string) map[string]string {
	return map[string]string{"error": message}
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
		MaxConcurrentAgents:    item.MaxConcurrentAgents,
	}
}

func uuidToStringPointer(value *uuid.UUID) *string {
	if value == nil {
		return nil
	}

	text := value.String()
	return &text
}

func intPointer(value int) *int {
	copied := value
	return &copied
}
