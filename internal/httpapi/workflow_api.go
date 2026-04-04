package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/labstack/echo/v4"
)

type workflowResponse struct {
	ID                    string                 `json:"id"`
	ProjectID             string                 `json:"project_id"`
	AgentID               *string                `json:"agent_id,omitempty"`
	Name                  string                 `json:"name"`
	Type                  string                 `json:"type"`
	RoleSlug              string                 `json:"role_slug"`
	RoleName              string                 `json:"role_name"`
	RoleDescription       string                 `json:"role_description"`
	PlatformAccessAllowed []string               `json:"platform_access_allowed"`
	WorkflowFamily        string                 `json:"workflow_family"`
	Classification        classificationResponse `json:"workflow_classification"`
	HarnessPath           string                 `json:"harness_path"`
	HarnessContent        *string                `json:"harness_content,omitempty"`
	Hooks                 map[string]any         `json:"hooks"`
	MaxConcurrent         int                    `json:"max_concurrent"`
	MaxRetryAttempts      int                    `json:"max_retry_attempts"`
	TimeoutMinutes        int                    `json:"timeout_minutes"`
	StallTimeoutMinutes   int                    `json:"stall_timeout_minutes"`
	Version               int                    `json:"version"`
	IsActive              bool                   `json:"is_active"`
	PickupStatusIDs       []string               `json:"pickup_status_ids"`
	FinishStatusIDs       []string               `json:"finish_status_ids"`
}

type classificationResponse struct {
	Family     string   `json:"family"`
	Confidence float64  `json:"confidence"`
	Reasons    []string `json:"reasons"`
}

type harnessResponse struct {
	WorkflowID string `json:"workflow_id"`
	Path       string `json:"path"`
	Content    string `json:"content"`
	Version    int    `json:"version"`
}

type workflowVersionResponse struct {
	ID        string `json:"id"`
	Version   int    `json:"version"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type workflowHistoryResponse struct {
	History []workflowVersionResponse `json:"history"`
}

type harnessValidationResponse struct {
	Valid  bool                              `json:"valid"`
	Issues []workflowservice.ValidationIssue `json:"issues"`
}

type harnessVariablesResponse struct {
	Groups []workflowservice.HarnessVariableGroup `json:"groups"`
}

type workflowReplaceReferencesResponse struct {
	WorkflowID            string                                          `json:"workflow_id"`
	ReplacementWorkflowID string                                          `json:"replacement_workflow_id"`
	TicketCount           int                                             `json:"ticket_count"`
	ScheduledJobCount     int                                             `json:"scheduled_job_count"`
	Tickets               []workflowservice.WorkflowTicketReference       `json:"tickets"`
	ScheduledJobs         []workflowservice.WorkflowScheduledJobReference `json:"scheduled_jobs"`
}

func (s *Server) registerWorkflowRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/workflows", s.handleListWorkflows)
	api.POST("/projects/:projectId/workflows", s.handleCreateWorkflow)
	api.GET("/workflows/:workflowId", s.handleGetWorkflow)
	api.GET("/workflows/:workflowId/impact", s.handleGetWorkflowImpact)
	api.PATCH("/workflows/:workflowId", s.handleUpdateWorkflow)
	api.POST("/workflows/:workflowId/retire", s.handleRetireWorkflow)
	api.POST("/workflows/:workflowId/replace-references", s.handleReplaceWorkflowReferences)
	api.DELETE("/workflows/:workflowId", s.handleDeleteWorkflow)
	api.GET("/workflows/:workflowId/harness", s.handleGetWorkflowHarness)
	api.GET("/workflows/:workflowId/harness/history", s.handleGetWorkflowHarnessHistory)
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
	raw.CreatedBy = optionalActor(raw.CreatedBy, actorFromHumanPrincipal(c))

	input, err := parseCreateWorkflowRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.workflowService.Create(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		AgentID:   item.AgentID,
		EventType: activityevent.TypeWorkflowCreated,
		Message:   "Created workflow " + item.Name,
		Metadata: map[string]any{
			"workflow_id":       item.ID.String(),
			"workflow_name":     item.Name,
			"workflow_type":     item.Type.String(),
			"is_active":         item.IsActive,
			"pickup_status_ids": item.PickupStatusIDs,
			"finish_status_ids": item.FinishStatusIDs,
			"audit_actor":       input.CreatedBy,
			"changed_fields":    []string{"workflow"},
		},
	}); err != nil {
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
	raw.EditedBy = optionalActor(raw.EditedBy, actorFromHumanPrincipal(c))

	input, err := parseUpdateWorkflowRequest(workflowID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	current, err := s.workflowService.Get(c.Request().Context(), workflowID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	item, err := s.workflowService.Update(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}
	activityInputs := make([]activitysvc.RecordInput, 0, 8)
	if raw.IsActive != nil {
		eventType := activityevent.TypeWorkflowDeactivated
		message := "Deactivated workflow " + item.Name
		if *raw.IsActive {
			eventType = activityevent.TypeWorkflowActivated
			message = "Activated workflow " + item.Name
		}
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			AgentID:   item.AgentID,
			EventType: eventType,
			Message:   message,
			Metadata: map[string]any{
				"workflow_id":    item.ID.String(),
				"workflow_name":  item.Name,
				"is_active":      item.IsActive,
				"audit_actor":    input.EditedBy,
				"changed_fields": []string{"is_active"},
			},
		})
	}
	if raw.Hooks != nil {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			AgentID:   item.AgentID,
			EventType: activityevent.TypeWorkflowHooksUpdated,
			Message:   "Updated workflow hooks for " + item.Name,
			Metadata: map[string]any{
				"workflow_id":    item.ID.String(),
				"workflow_name":  item.Name,
				"audit_actor":    input.EditedBy,
				"changed_fields": []string{"hooks"},
			},
		})
	}
	if raw.AgentID != nil {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			AgentID:   item.AgentID,
			EventType: activityevent.TypeWorkflowAgentChanged,
			Message:   "Changed workflow agent for " + item.Name,
			Metadata: map[string]any{
				"workflow_id":    item.ID.String(),
				"workflow_name":  item.Name,
				"from_agent_id":  current.AgentID,
				"to_agent_id":    item.AgentID,
				"audit_actor":    input.EditedBy,
				"changed_fields": []string{"agent_id"},
			},
		})
	}
	if raw.PickupStatusIDs != nil {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			AgentID:   item.AgentID,
			EventType: activityevent.TypeWorkflowPickupStatusesChanged,
			Message:   "Updated workflow pickup statuses for " + item.Name,
			Metadata: map[string]any{
				"workflow_id":       item.ID.String(),
				"workflow_name":     item.Name,
				"pickup_status_ids": item.PickupStatusIDs,
				"audit_actor":       input.EditedBy,
				"changed_fields":    []string{"pickup_status_ids"},
			},
		})
	}
	if raw.FinishStatusIDs != nil {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			AgentID:   item.AgentID,
			EventType: activityevent.TypeWorkflowFinishStatusesChanged,
			Message:   "Updated workflow finish statuses for " + item.Name,
			Metadata: map[string]any{
				"workflow_id":       item.ID.String(),
				"workflow_name":     item.Name,
				"finish_status_ids": item.FinishStatusIDs,
				"audit_actor":       input.EditedBy,
				"changed_fields":    []string{"finish_status_ids"},
			},
		})
	}
	if raw.MaxConcurrent != nil {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			AgentID:   item.AgentID,
			EventType: activityevent.TypeWorkflowConcurrencyChanged,
			Message:   "Changed workflow concurrency for " + item.Name,
			Metadata: map[string]any{
				"workflow_id":    item.ID.String(),
				"workflow_name":  item.Name,
				"max_concurrent": item.MaxConcurrent,
				"audit_actor":    input.EditedBy,
				"changed_fields": []string{"max_concurrent"},
			},
		})
	}
	if raw.MaxRetryAttempts != nil {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			AgentID:   item.AgentID,
			EventType: activityevent.TypeWorkflowRetryPolicyChanged,
			Message:   "Changed workflow retry policy for " + item.Name,
			Metadata: map[string]any{
				"workflow_id":        item.ID.String(),
				"workflow_name":      item.Name,
				"max_retry_attempts": item.MaxRetryAttempts,
				"audit_actor":        input.EditedBy,
				"changed_fields":     []string{"max_retry_attempts"},
			},
		})
	}
	if raw.TimeoutMinutes != nil || raw.StallTimeoutMinutes != nil {
		metadata := workflowTimeoutMetadata(raw, mapWorkflowDetailResponse(item))
		metadata["audit_actor"] = input.EditedBy
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			AgentID:   item.AgentID,
			EventType: activityevent.TypeWorkflowTimeoutChanged,
			Message:   "Changed workflow timeouts for " + item.Name,
			Metadata:  metadata,
		})
	}
	if raw.Name != nil || raw.Type != nil || raw.HarnessPath != nil {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			AgentID:   item.AgentID,
			EventType: activityevent.TypeWorkflowUpdated,
			Message:   "Updated workflow " + item.Name,
			Metadata: map[string]any{
				"workflow_id":    item.ID.String(),
				"workflow_name":  item.Name,
				"workflow_type":  item.Type.String(),
				"harness_path":   item.HarnessPath,
				"audit_actor":    input.EditedBy,
				"changed_fields": workflowChangedFields(raw),
			},
		})
	}
	if err := s.emitActivities(c.Request().Context(), activityInputs...); err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"workflow": mapWorkflowDetailResponse(item),
	})
}

func (s *Server) handleGetWorkflowImpact(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	impact, err := s.workflowService.ImpactAnalysis(c.Request().Context(), workflowID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"impact": impact,
	})
}

func (s *Server) handleRetireWorkflow(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	var raw rawRetireWorkflowRequest
	if err := decodeJSON(c, &raw); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	raw.EditedBy = optionalActor(raw.EditedBy, actorFromHumanPrincipal(c))

	editedBy := parseRetireWorkflowRequest(workflowID, raw)

	item, err := s.workflowService.Retire(c.Request().Context(), workflowID, editedBy)
	if err != nil {
		return writeWorkflowError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		AgentID:   item.AgentID,
		EventType: activityevent.TypeWorkflowDeactivated,
		Message:   "Retired workflow " + item.Name,
		Metadata: map[string]any{
			"workflow_id":    item.ID.String(),
			"workflow_name":  item.Name,
			"is_active":      item.IsActive,
			"audit_actor":    editedBy,
			"changed_fields": []string{"is_active"},
		},
	}); err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"workflow": mapWorkflowDetailResponse(item),
	})
}

func (s *Server) handleReplaceWorkflowReferences(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	var raw rawReplaceWorkflowReferencesRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	raw.EditedBy = optionalActor(raw.EditedBy, actorFromHumanPrincipal(c))

	input, editedBy, err := parseReplaceWorkflowReferencesRequest(workflowID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	result, err := s.workflowService.ReplaceReferences(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}
	source, err := s.workflowService.Get(c.Request().Context(), workflowID)
	if err != nil {
		return writeWorkflowError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: source.ProjectID,
		AgentID:   source.AgentID,
		EventType: activityevent.TypeWorkflowUpdated,
		Message:   "Replaced workflow references for " + source.Name,
		Metadata: map[string]any{
			"workflow_id":             source.ID.String(),
			"workflow_name":           source.Name,
			"replacement_workflow_id": result.ReplacementWorkflowID.String(),
			"ticket_count":            result.TicketCount,
			"scheduled_job_count":     result.ScheduledJobCount,
			"audit_actor":             editedBy,
			"changed_fields":          []string{"workflow_references"},
		},
	}); err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"result": mapWorkflowReplaceReferencesResponse(result),
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
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		AgentID:   item.AgentID,
		EventType: activityevent.TypeWorkflowDeleted,
		Message:   "Deleted workflow " + item.Name,
		Metadata: map[string]any{
			"workflow_id":    item.ID.String(),
			"workflow_name":  item.Name,
			"changed_fields": []string{"workflow"},
		},
	}); err != nil {
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

func (s *Server) handleGetWorkflowHarnessHistory(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	workflowID, err := parseUUIDPathParamValue(c, "workflowId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_ID", err.Error())
	}

	items, err := s.workflowService.ListWorkflowVersions(c.Request().Context(), workflowID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, workflowHistoryResponse{
		History: mapWorkflowVersionResponses(items),
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
	raw.EditedBy = optionalActor(raw.EditedBy, actorFromHumanPrincipal(c))

	input, err := parseUpdateHarnessRequest(workflowID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	document, err := s.workflowService.UpdateHarness(c.Request().Context(), input)
	if err != nil {
		return writeWorkflowError(c, err)
	}
	workflowItem, err := s.workflowService.Get(c.Request().Context(), workflowID)
	if err != nil {
		return writeWorkflowError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: workflowItem.ProjectID,
		AgentID:   workflowItem.AgentID,
		EventType: activityevent.TypeWorkflowHarnessUpdated,
		Message:   "Updated workflow harness for " + workflowItem.Name,
		Metadata: map[string]any{
			"workflow_id":    workflowItem.ID.String(),
			"workflow_name":  workflowItem.Name,
			"harness_path":   document.Path,
			"version":        document.Version,
			"audit_actor":    input.EditedBy,
			"changed_fields": []string{"harness"},
		},
	}); err != nil {
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
	var impactConflict *workflowservice.WorkflowImpactConflict
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
	case errors.Is(err, workflowservice.ErrPickupStatusConflict):
		return writeAPIError(c, http.StatusConflict, "PICKUP_STATUS_CONFLICT", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowStatusBindingOverlap):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_STATUS_BINDING_OVERLAP", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowNameConflict):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_NAME_CONFLICT", normalizeWorkflowErrorMessage(err, workflowservice.ErrWorkflowNameConflict))
	case errors.Is(err, workflowservice.ErrWorkflowHarnessPathConflict):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_HARNESS_PATH_CONFLICT", normalizeWorkflowErrorMessage(err, workflowservice.ErrWorkflowHarnessPathConflict))
	case errors.Is(err, workflowservice.ErrWorkflowReferencedByTickets):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_REFERENCED_BY_TICKETS", normalizeWorkflowErrorMessage(err, workflowservice.ErrWorkflowReferencedByTickets))
	case errors.Is(err, workflowservice.ErrWorkflowReferencedByScheduledJobs):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_REFERENCED_BY_SCHEDULED_JOBS", normalizeWorkflowErrorMessage(err, workflowservice.ErrWorkflowReferencedByScheduledJobs))
	case errors.Is(err, workflowservice.ErrWorkflowConflict):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_CONFLICT", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowInUse):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_IN_USE", err.Error())
	case errors.As(err, &impactConflict):
		return writeAPIErrorWithDetails(
			c,
			http.StatusConflict,
			workflowConflictCode(err),
			normalizeWorkflowErrorMessage(err, impactConflict.Err),
			impactConflict.Impact,
		)
	case errors.Is(err, workflowservice.ErrWorkflowReplacementRequired):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_REPLACEMENT_REQUIRED", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowActiveAgentRuns):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_ACTIVE_AGENT_RUNS", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowHistoricalAgentRuns):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_HISTORICAL_AGENT_RUNS", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowReplacementInvalid):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_WORKFLOW_REPLACEMENT", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowReplacementNotFound):
		return writeAPIError(c, http.StatusNotFound, "REPLACEMENT_WORKFLOW_NOT_FOUND", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowReplacementProjectMismatch):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_REPLACEMENT_PROJECT_MISMATCH", err.Error())
	case errors.Is(err, workflowservice.ErrWorkflowReplacementInactive):
		return writeAPIError(c, http.StatusConflict, "WORKFLOW_REPLACEMENT_INACTIVE", err.Error())
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

func normalizeWorkflowErrorMessage(err error, sentinel error) string {
	if err == nil {
		return ""
	}
	prefix := sentinel.Error() + ": "
	if strings.HasPrefix(err.Error(), prefix) {
		return strings.TrimPrefix(err.Error(), prefix)
	}
	return err.Error()
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
	classification := workflowservice.ClassifyWorkflow(workflowservice.WorkflowClassificationInput{
		TypeLabel:    item.Type,
		WorkflowName: item.Name,
		HarnessPath:  item.HarnessPath,
	})

	return workflowResponse{
		ID:                    item.ID.String(),
		ProjectID:             item.ProjectID.String(),
		AgentID:               agentID,
		Name:                  item.Name,
		Type:                  item.Type.String(),
		RoleSlug:              item.RoleSlug,
		RoleName:              item.RoleName,
		RoleDescription:       item.RoleDescription,
		PlatformAccessAllowed: append([]string(nil), item.PlatformAccessAllowed...),
		WorkflowFamily:        string(classification.Family),
		Classification:        mapClassificationResponse(classification),
		HarnessPath:           item.HarnessPath,
		Hooks:                 item.Hooks,
		MaxConcurrent:         item.MaxConcurrent,
		MaxRetryAttempts:      item.MaxRetryAttempts,
		TimeoutMinutes:        item.TimeoutMinutes,
		StallTimeoutMinutes:   item.StallTimeoutMinutes,
		Version:               item.Version,
		IsActive:              item.IsActive,
		PickupStatusIDs:       uuidSliceToStrings(item.PickupStatusIDs),
		FinishStatusIDs:       uuidSliceToStrings(item.FinishStatusIDs),
	}
}

func mapWorkflowDetailResponse(item workflowservice.WorkflowDetail) workflowResponse {
	response := mapWorkflowResponse(item.Workflow)
	response.Classification = mapClassificationResponse(workflowservice.ClassifyWorkflow(workflowservice.WorkflowClassificationInput{
		RoleSlug:       item.RoleSlug,
		TypeLabel:      item.Type,
		WorkflowName:   item.Name,
		HarnessPath:    item.HarnessPath,
		HarnessContent: item.HarnessContent,
	}))
	response.WorkflowFamily = response.Classification.Family
	response.HarnessContent = stringPointer(item.HarnessContent)
	return response
}

func mapClassificationResponse(classification workflowservice.WorkflowClassification) classificationResponse {
	return classificationResponse{
		Family:     string(classification.Family),
		Confidence: classification.Confidence,
		Reasons:    cloneStringSlice(classification.Reasons),
	}
}

func mapHarnessResponse(item workflowservice.HarnessDocument) harnessResponse {
	return harnessResponse{
		WorkflowID: item.WorkflowID.String(),
		Path:       item.Path,
		Content:    item.Content,
		Version:    item.Version,
	}
}

func mapWorkflowVersionResponses(items []workflowservice.VersionSummary) []workflowVersionResponse {
	response := make([]workflowVersionResponse, 0, len(items))
	for _, item := range items {
		response = append(response, workflowVersionResponse{
			ID:        item.ID.String(),
			Version:   item.Version,
			CreatedBy: item.CreatedBy,
			CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return response
}

func mapWorkflowReplaceReferencesResponse(result workflowservice.ReplaceWorkflowReferencesResult) workflowReplaceReferencesResponse {
	return workflowReplaceReferencesResponse{
		WorkflowID:            result.WorkflowID.String(),
		ReplacementWorkflowID: result.ReplacementWorkflowID.String(),
		TicketCount:           result.TicketCount,
		ScheduledJobCount:     result.ScheduledJobCount,
		Tickets:               result.Tickets,
		ScheduledJobs:         result.ScheduledJobs,
	}
}

func workflowConflictCode(err error) string {
	switch {
	case errors.Is(err, workflowservice.ErrWorkflowActiveAgentRuns):
		return "WORKFLOW_ACTIVE_AGENT_RUNS"
	case errors.Is(err, workflowservice.ErrWorkflowHistoricalAgentRuns):
		return "WORKFLOW_HISTORICAL_AGENT_RUNS"
	case errors.Is(err, workflowservice.ErrWorkflowReplacementRequired):
		return "WORKFLOW_REPLACEMENT_REQUIRED"
	default:
		return "WORKFLOW_IN_USE"
	}
}
