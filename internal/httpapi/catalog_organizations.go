package httpapi

import (
	"net/http"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerOrganizationRoutes(api *echo.Group) {
	api.GET("/orgs", s.listOrganizations)
	api.POST("/orgs", s.createOrganization)
	api.GET("/orgs/:orgId", s.getOrganization)
	api.PATCH("/orgs/:orgId", s.patchOrganization)
	api.DELETE("/orgs/:orgId", s.archiveOrganization)
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

	input, err := parseOrganizationPatchRequest(orgID, current, patch)
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
