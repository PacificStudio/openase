package httpapi

import (
	"net/http"

	"github.com/BetterAndBetterII/openase/internal/builtin"
	"github.com/labstack/echo/v4"
)

type builtinRoleResponse struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	WorkflowType string `json:"workflow_type"`
	Summary      string `json:"summary"`
	HarnessPath  string `json:"harness_path"`
	Content      string `json:"content"`
}

func (s *Server) registerRoleLibraryRoutes(api *echo.Group) {
	api.GET("/roles/builtin", s.handleListBuiltinRoles)
}

func (s *Server) handleListBuiltinRoles(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"roles": mapBuiltinRoleResponses(builtin.Roles()),
	})
}

func mapBuiltinRoleResponses(items []builtin.RoleTemplate) []builtinRoleResponse {
	response := make([]builtinRoleResponse, 0, len(items))
	for _, item := range items {
		response = append(response, builtinRoleResponse{
			Slug:         item.Slug,
			Name:         item.Name,
			WorkflowType: item.WorkflowType,
			Summary:      item.Summary,
			HarnessPath:  item.HarnessPath,
			Content:      item.Content,
		})
	}

	return response
}
