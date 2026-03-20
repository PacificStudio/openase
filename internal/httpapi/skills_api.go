package httpapi

import (
	"net/http"

	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/labstack/echo/v4"
)

type skillResponse struct {
	Name           string                         `json:"name"`
	Description    string                         `json:"description"`
	Path           string                         `json:"path"`
	IsBuiltin      bool                           `json:"is_builtin"`
	BoundWorkflows []skillWorkflowBindingResponse `json:"bound_workflows"`
}

type skillWorkflowBindingResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	HarnessPath string `json:"harness_path"`
}

type skillSyncResponse struct {
	SkillsDir       string   `json:"skills_dir"`
	InjectedSkills  []string `json:"injected_skills,omitempty"`
	HarvestedSkills []string `json:"harvested_skills,omitempty"`
	UpdatedSkills   []string `json:"updated_skills,omitempty"`
}

func (s *Server) registerSkillRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/skills", s.handleListSkills)
	api.POST("/projects/:projectId/skills/refresh", s.handleRefreshSkills)
	api.POST("/projects/:projectId/skills/harvest", s.handleHarvestSkills)
	api.POST("/workflows/:workflowId/skills/bind", s.handleBindWorkflowSkills)
	api.POST("/workflows/:workflowId/skills/unbind", s.handleUnbindWorkflowSkills)
}

func (s *Server) handleListSkills(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	items, err := s.workflowService.ListSkills(c.Request().Context(), projectID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"skills": mapSkillResponses(items),
	})
}

func (s *Server) handleRefreshSkills(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw rawSkillSyncRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseRefreshSkillsRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	result, err := s.workflowService.RefreshSkills(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, skillSyncResponse{
		SkillsDir:      result.SkillsDir,
		InjectedSkills: result.InjectedSkills,
	})
}

func (s *Server) handleHarvestSkills(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw rawSkillSyncRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseHarvestSkillsRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	result, err := s.workflowService.HarvestSkills(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, skillSyncResponse{
		SkillsDir:       result.SkillsDir,
		HarvestedSkills: result.HarvestedSkills,
		UpdatedSkills:   result.UpdatedSkills,
	})
}

func (s *Server) handleBindWorkflowSkills(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	var raw rawUpdateWorkflowSkillsRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateWorkflowSkillsRequest(workflowID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	document, err := s.workflowService.BindSkills(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"harness": mapHarnessResponse(document),
	})
}

func (s *Server) handleUnbindWorkflowSkills(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	var raw rawUpdateWorkflowSkillsRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateWorkflowSkillsRequest(workflowID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	document, err := s.workflowService.UnbindSkills(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"harness": mapHarnessResponse(document),
	})
}

func mapSkillResponses(items []workflowservice.Skill) []skillResponse {
	response := make([]skillResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapSkillResponse(item))
	}
	return response
}

func mapSkillResponse(item workflowservice.Skill) skillResponse {
	return skillResponse{
		Name:           item.Name,
		Description:    item.Description,
		Path:           item.Path,
		IsBuiltin:      item.IsBuiltin,
		BoundWorkflows: mapSkillWorkflowBindings(item.BoundWorkflows),
	}
}

func mapSkillWorkflowBindings(items []workflowservice.SkillWorkflowBinding) []skillWorkflowBindingResponse {
	response := make([]skillWorkflowBindingResponse, 0, len(items))
	for _, item := range items {
		response = append(response, skillWorkflowBindingResponse{
			ID:          item.ID.String(),
			Name:        item.Name,
			HarnessPath: item.HarnessPath,
		})
	}
	return response
}
