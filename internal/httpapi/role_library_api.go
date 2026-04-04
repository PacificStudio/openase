package httpapi

import (
	"net/http"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/builtin"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/labstack/echo/v4"
)

type builtinRoleResponse struct {
	Slug                  string   `json:"slug"`
	Name                  string   `json:"name"`
	WorkflowType          string   `json:"workflow_type"`
	WorkflowFamily        string   `json:"workflow_family"`
	Summary               string   `json:"summary"`
	HarnessPath           string   `json:"harness_path"`
	Content               string   `json:"content"`
	WorkflowContent       string   `json:"workflow_content"`
	PickupStatusNames     []string `json:"pickup_status_names"`
	FinishStatusNames     []string `json:"finish_status_names"`
	SkillNames            []string `json:"skill_names"`
	PlatformAccessAllowed []string `json:"platform_access_allowed"`
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
	classification := workflowservice.ClassifyWorkflow(workflowservice.WorkflowClassificationInput{
		RoleSlug:       item.Slug,
		TypeLabel:      workflowservice.MustParseTypeLabel(item.WorkflowType),
		WorkflowName:   item.Name,
		HarnessPath:    item.HarnessPath,
		HarnessContent: item.Content,
	})
	return builtinRoleResponse{
		Slug:                  item.Slug,
		Name:                  item.Name,
		WorkflowType:          item.WorkflowType,
		WorkflowFamily:        string(classification.Family),
		Summary:               item.Summary,
		HarnessPath:           item.HarnessPath,
		Content:               item.Content,
		WorkflowContent:       item.Content,
		PickupStatusNames:     append([]string(nil), item.PickupStatusNames...),
		FinishStatusNames:     append([]string(nil), item.FinishStatusNames...),
		SkillNames:            append([]string(nil), item.SkillNames...),
		PlatformAccessAllowed: append([]string(nil), item.PlatformAccessAllowed...),
	}
}

func mapBuiltinRoleResponses(items []builtin.RoleTemplate) []builtinRoleResponse {
	response := make([]builtinRoleResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapBuiltinRoleResponse(item))
	}

	return response
}
