package httpapi

import (
	"errors"
	"net/http"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerTicketStatusRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/statuses", s.handleListTicketStatuses)
	api.POST("/projects/:projectId/statuses", s.handleCreateTicketStatus)
	api.POST("/projects/:projectId/statuses/reset", s.handleResetTicketStatuses)
	api.PATCH("/statuses/:statusId", s.handleUpdateTicketStatus)
	api.DELETE("/statuses/:statusId", s.handleDeleteTicketStatus)
}

func (s *Server) handleListTicketStatuses(c echo.Context) error {
	service := s.ticketStatusService
	if service == nil {
		return writeTicketStatusError(c, ticketstatus.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	result, err := service.List(c.Request().Context(), projectID)
	if err != nil {
		return writeTicketStatusError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

func (s *Server) handleCreateTicketStatus(c echo.Context) error {
	service := s.ticketStatusService
	if service == nil {
		return writeTicketStatusError(c, ticketstatus.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw rawCreateTicketStatusRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}
	input, err := parseCreateTicketStatusRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	status, err := service.Create(c.Request().Context(), input)
	if err != nil {
		return writeTicketStatusError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: status.ProjectID,
		EventType: activityevent.TypeTicketStatusCreated,
		Message:   "Created ticket status " + status.Name,
		Metadata: map[string]any{
			"status_id":       status.ID.String(),
			"status_name":     status.Name,
			"stage":           status.Stage,
			"position":        status.Position,
			"max_active_runs": status.MaxActiveRuns,
			"is_default":      status.IsDefault,
			"changed_fields":  []string{"status"},
		},
	}); err != nil {
		return writeTicketStatusError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{"status": status})
}

func (s *Server) handleUpdateTicketStatus(c echo.Context) error {
	service := s.ticketStatusService
	if service == nil {
		return writeTicketStatusError(c, ticketstatus.ErrUnavailable)
	}

	statusID, err := parseStatusID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_STATUS_ID", err.Error())
	}

	var raw rawUpdateTicketStatusRequest
	if err := c.Bind(&raw); err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
	}
	input, err := parseUpdateTicketStatusRequest(statusID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	current, err := service.Get(c.Request().Context(), statusID)
	if err != nil {
		return writeTicketStatusError(c, err)
	}

	status, err := service.Update(c.Request().Context(), input)
	if err != nil {
		return writeTicketStatusError(c, err)
	}
	activityInputs := make([]activitysvc.RecordInput, 0, 3)
	if raw.Position != nil {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: status.ProjectID,
			EventType: activityevent.TypeTicketStatusReordered,
			Message:   "Reordered ticket status " + status.Name,
			Metadata: map[string]any{
				"status_id":      status.ID.String(),
				"status_name":    status.Name,
				"from_position":  current.Position,
				"to_position":    status.Position,
				"changed_fields": []string{"position"},
			},
		})
	}
	if raw.MaxActiveRuns.Set {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: status.ProjectID,
			EventType: activityevent.TypeTicketStatusConcurrencyChanged,
			Message:   "Changed ticket status concurrency for " + status.Name,
			Metadata: map[string]any{
				"status_id":            status.ID.String(),
				"status_name":          status.Name,
				"from_max_active_runs": current.MaxActiveRuns,
				"to_max_active_runs":   status.MaxActiveRuns,
				"changed_fields":       []string{"max_active_runs"},
			},
		})
	}
	if raw.Name != nil || raw.Stage != nil || raw.Color != nil || raw.Icon != nil || raw.IsDefault != nil || raw.Description != nil {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: status.ProjectID,
			EventType: activityevent.TypeTicketStatusUpdated,
			Message:   "Updated ticket status " + status.Name,
			Metadata: map[string]any{
				"status_id":      status.ID.String(),
				"status_name":    status.Name,
				"stage":          status.Stage,
				"position":       status.Position,
				"is_default":     status.IsDefault,
				"changed_fields": ticketStatusChangedFields(raw),
			},
		})
	}
	if err := s.emitActivities(c.Request().Context(), activityInputs...); err != nil {
		return writeTicketStatusError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"status": status})
}

func (s *Server) handleDeleteTicketStatus(c echo.Context) error {
	service := s.ticketStatusService
	if service == nil {
		return writeTicketStatusError(c, ticketstatus.ErrUnavailable)
	}

	statusID, err := parseStatusID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_STATUS_ID", err.Error())
	}
	current, err := service.Get(c.Request().Context(), statusID)
	if err != nil {
		return writeTicketStatusError(c, err)
	}

	result, err := service.Delete(c.Request().Context(), statusID)
	if err != nil {
		return writeTicketStatusError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: current.ProjectID,
		EventType: activityevent.TypeTicketStatusDeleted,
		Message:   "Deleted ticket status " + current.Name,
		Metadata: map[string]any{
			"status_id":             current.ID.String(),
			"status_name":           current.Name,
			"replacement_status_id": result.ReplacementStatusID.String(),
			"changed_fields":        []string{"status"},
		},
	}); err != nil {
		return writeTicketStatusError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

func (s *Server) handleResetTicketStatuses(c echo.Context) error {
	service := s.ticketStatusService
	if service == nil {
		return writeTicketStatusError(c, ticketstatus.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	_, err = service.ResetToDefaultTemplate(c.Request().Context(), projectID)
	if err != nil {
		return writeTicketStatusError(c, err)
	}

	result, err := service.List(c.Request().Context(), projectID)
	if err != nil {
		return writeTicketStatusError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: projectID,
		EventType: activityevent.TypeTicketStatusReset,
		Message:   "Reset ticket statuses to the default template",
		Metadata: map[string]any{
			"status_count":   len(result.Statuses),
			"status_names":   mapStatusNames(result.Statuses),
			"changed_fields": []string{"status_template"},
		},
	}); err != nil {
		return writeTicketStatusError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

func writeTicketStatusError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, ticketstatus.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, ticketstatus.ErrProjectNotFound):
		return writeAPIError(c, http.StatusNotFound, "PROJECT_NOT_FOUND", err.Error())
	case errors.Is(err, ticketstatus.ErrStatusNotFound):
		return writeAPIError(c, http.StatusNotFound, "STATUS_NOT_FOUND", err.Error())
	case errors.Is(err, ticketstatus.ErrDuplicateStatusName):
		return writeAPIError(c, http.StatusConflict, "STATUS_NAME_CONFLICT", err.Error())
	case errors.Is(err, ticketstatus.ErrDefaultStatusStage):
		return writeAPIError(c, http.StatusConflict, "DEFAULT_STATUS_STAGE_INVALID", err.Error())
	case errors.Is(err, ticketstatus.ErrCannotDeleteLastStatus):
		return writeAPIError(c, http.StatusConflict, "LAST_STATUS_DELETE_FORBIDDEN", err.Error())
	case errors.Is(err, ticketstatus.ErrDefaultStatusRequired):
		return writeAPIError(c, http.StatusConflict, "DEFAULT_STATUS_REQUIRED", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func writeAPIError(c echo.Context, statusCode int, code string, message string) error {
	logAPIBoundaryError(c, statusCode, code, message)
	return c.JSON(statusCode, map[string]string{
		"code":    code,
		"message": message,
	})
}

func writeAPIErrorWithDetails(c echo.Context, statusCode int, code string, message string, details any) error {
	logAPIBoundaryError(c, statusCode, code, message)
	payload := map[string]any{
		"code":    code,
		"message": message,
	}
	if details != nil {
		payload["details"] = details
	}
	return c.JSON(statusCode, payload)
}
