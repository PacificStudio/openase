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
	roles := builtin.Roles()
	response := make([]builtinRoleResponse, 0, len(roles))
	for _, item := range roles {
		response = append(response, builtinRoleResponse{
			Slug:         item.Slug,
			Name:         item.Name,
			WorkflowType: item.WorkflowType,
			Summary:      item.Summary,
			HarnessPath:  item.HarnessPath,
			Content:      item.Content,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"roles": response,
	})
}
