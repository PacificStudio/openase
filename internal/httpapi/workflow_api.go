package httpapi

import (
	"errors"
	"net/http"

	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/labstack/echo/v4"
)

type workflowResponse struct {
	ID                  string         `json:"id"`
	ProjectID           string         `json:"project_id"`
	AgentID             *string        `json:"agent_id,omitempty"`
	Name                string         `json:"name"`
	Type                string         `json:"type"`
	HarnessPath         string         `json:"harness_path"`
	HarnessContent      *string        `json:"harness_content,omitempty"`
	Hooks               map[string]any `json:"hooks"`
	MaxConcurrent       int            `json:"max_concurrent"`
	MaxRetryAttempts    int            `json:"max_retry_attempts"`
	TimeoutMinutes      int            `json:"timeout_minutes"`
	StallTimeoutMinutes int            `json:"stall_timeout_minutes"`
	Version             int            `json:"version"`
	IsActive            bool           `json:"is_active"`
	PickupStatusIDs     []string       `json:"pickup_status_ids"`
	FinishStatusIDs     []string       `json:"finish_status_ids"`
}

type harnessResponse struct {
	WorkflowID string `json:"workflow_id"`
	Path       string `json:"path"`
	Content    string `json:"content"`
	Version    int    `json:"version"`
}

type harnessValidationResponse struct {
	Valid  bool                              `json:"valid"`
	Issues []workflowservice.ValidationIssue `json:"issues"`
}

type harnessVariablesResponse struct {
	Groups []workflowservice.HarnessVariableGroup `json:"groups"`
}

func (s *Server) registerWorkflowRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/workflows", s.handleListWorkflows)
	api.POST("/projects/:projectId/workflows", s.handleCreateWorkflow)
	api.GET("/workflows/:workflowId", s.handleGetWorkflow)
	api.PATCH("/workflows/:workflowId", s.handleUpdateWorkflow)
	api.DELETE("/workflows/:workflowId", s.handleDeleteWorkflow)
	api.GET("/workflows/:workflowId/harness", s.handleGetWorkflowHarness)
	api.PUT("/workflows/:workflowId/harness", s.handleUpdateWorkflowHarness)
	api.GET("/harness/variables", s.handleListHarnessVariables)
	api.POST("/harness/validate", s.handleValidateHarness)
}

func (s *Server) handleListWorkflows(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	items, err := s.workflowService.List(c.Request().Context(), projectID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"workflows": mapWorkflowResponses(items),
	})
}

func (s *Server) handleCreateWorkflow(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw rawCreateWorkflowRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseCreateWorkflowRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.workflowService.Create(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"workflow": mapWorkflowDetailResponse(item),
	})
}

func (s *Server) handleGetWorkflow(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	item, err := s.workflowService.Get(c.Request().Context(), workflowID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"workflow": mapWorkflowDetailResponse(item),
	})
}

func (s *Server) handleUpdateWorkflow(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	var raw rawUpdateWorkflowRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateWorkflowRequest(workflowID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.workflowService.Update(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"workflow": mapWorkflowDetailResponse(item),
	})
}

func (s *Server) handleDeleteWorkflow(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	item, err := s.workflowService.Delete(c.Request().Context(), workflowID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"workflow": mapWorkflowResponse(item),
	})
}

func (s *Server) handleGetWorkflowHarness(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	document, err := s.workflowService.GetHarness(c.Request().Context(), workflowID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"harness": mapHarnessResponse(document),
	})
}

func (s *Server) handleUpdateWorkflowHarness(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	var raw rawUpdateHarnessRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateHarnessRequest(workflowID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	document, err := s.workflowService.UpdateHarness(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"harness": mapHarnessResponse(document),
	})
}

func (s *Server) handleValidateHarness(c echo.Context) error {
	var raw rawValidateHarnessRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	result := workflowservice.ValidateHarnessContent(raw.Content)
	return c.JSON(http.StatusOK, harnessValidationResponse{
		Valid:  result.Valid,
		Issues: result.Issues,
	})
}

func (s *Server) handleListHarnessVariables(c echo.Context) error {
	return c.JSON(http.StatusOK, harnessVariablesResponse{
		Groups: workflowservice.HarnessVariableDictionary(),
	})
}

func writeWorkflowError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, workflowservice.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, workflowservice.ErrProjectNotFound):
		return writeAPIError(c, http.StatusNotFound, "PROJECT_NOT_FOUND", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowNotFound):
		return writeAPIError(c, http.StatusNotFound, "WORKFLOW_NOT_FOUND", err.Error())
	case errors.Is(err, workflowservice.ErrStatusNotFound):
		return writeAPIError(c, http.StatusBadRequest, "STATUS_NOT_FOUND", err.Error())
	case errors.Is(err, workflowservice.ErrAgentNotFound):
		return writeAPIError(c, http.StatusBadRequest, "AGENT_NOT_FOUND", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowConflict):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_CONFLICT", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowInUse):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_IN_USE", err.Error())
	case errors.Is(err, workflowservice.ErrSkillNotFound):
		return writeAPIError(c, http.StatusNotFound, "SKILL_NOT_FOUND", err.Error())
	case errors.Is(err, workflowservice.ErrSkillInvalid):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_SKILL", err.Error())
	case errors.Is(err, workflowservice.ErrHarnessInvalid):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_HARNESS", err.Error())
	case errors.Is(err, workflowservice.ErrHookConfigInvalid):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_HOOKS", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowHookBlocked):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_HOOK_BLOCKED", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func mapWorkflowResponses(items []workflowservice.Workflow) []workflowResponse {
	response := make([]workflowResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapWorkflowResponse(item))
	}

	return response
}

func mapWorkflowResponse(item workflowservice.Workflow) workflowResponse {
	var agentID *string
	if item.AgentID != nil {
		value := item.AgentID.String()
		agentID = &value
	}

	return workflowResponse{
		ID:                  item.ID.String(),
		ProjectID:           item.ProjectID.String(),
		AgentID:             agentID,
		Name:                item.Name,
		Type:                item.Type.String(),
		HarnessPath:         item.HarnessPath,
		Hooks:               item.Hooks,
		MaxConcurrent:       item.MaxConcurrent,
		MaxRetryAttempts:    item.MaxRetryAttempts,
		TimeoutMinutes:      item.TimeoutMinutes,
		StallTimeoutMinutes: item.StallTimeoutMinutes,
		Version:             item.Version,
		IsActive:            item.IsActive,
		PickupStatusIDs:     uuidSliceToStrings(item.PickupStatusIDs),
		FinishStatusIDs:     uuidSliceToStrings(item.FinishStatusIDs),
	}
}

func mapWorkflowDetailResponse(item workflowservice.WorkflowDetail) workflowResponse {
	response := mapWorkflowResponse(item.Workflow)
	response.HarnessContent = stringPointer(item.HarnessContent)
	return response
}

func mapHarnessResponse(item workflowservice.HarnessDocument) harnessResponse {
	return harnessResponse{
		WorkflowID: item.WorkflowID.String(),
		Path:       item.Path,
		Content:    item.Content,
		Version:    item.Version,
	}
}
