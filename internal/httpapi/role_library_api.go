package httpapi

import (
	"net/http"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/builtin"
	"github.com/labstack/echo/v4"
)

type builtinRoleResponse struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	WorkflowType    string `json:"workflow_type"`
	Summary         string `json:"summary"`
	HarnessPath     string `json:"harness_path"`
	Content         string `json:"content"`
	WorkflowContent string `json:"workflow_content"`
}

type builtinRoleDetailResponse struct {
	Role builtinRoleResponse `json:"role"`
}

func (s *Server) registerRoleLibraryRoutes(api *echo.Group) {
	api.GET("/roles/builtin", s.handleListBuiltinRoles)
	api.GET("/roles/builtin/:roleSlug", s.handleGetBuiltinRole)
}

func (s *Server) handleListBuiltinRoles(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"roles": mapBuiltinRoleResponses(builtin.Roles()),
	})
}

func (s *Server) handleGetBuiltinRole(c echo.Context) error {
	roleSlug := strings.TrimSpace(c.Param("roleSlug"))
	if roleSlug == "" {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_ROLE_SLUG", "role slug must not be empty")
	}

	item, ok := builtin.RoleBySlug(roleSlug)
	if !ok {
		return writeAPIError(c, http.StatusNotFound, "ROLE_TEMPLATE_NOT_FOUND", "builtin role template not found")
	}

	return c.JSON(http.StatusOK, builtinRoleDetailResponse{Role: mapBuiltinRoleResponse(item)})
}

func mapBuiltinRoleResponse(item builtin.RoleTemplate) builtinRoleResponse {
	return builtinRoleResponse{
		Slug:            item.Slug,
		Name:            item.Name,
		WorkflowType:    item.WorkflowType,
		Summary:         item.Summary,
		HarnessPath:     item.HarnessPath,
		Content:         item.Content,
		WorkflowContent: item.Content,
	}
}

func mapBuiltinRoleResponses(items []builtin.RoleTemplate) []builtinRoleResponse {
	response := make([]builtinRoleResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapBuiltinRoleResponse(item))
	}

	return response
}
