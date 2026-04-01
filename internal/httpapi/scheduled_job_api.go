package httpapi

import (
	"errors"
	"net/http"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	"github.com/labstack/echo/v4"
)

type scheduledJobTicketTemplateResponse struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status,omitempty"`
	Priority    string  `json:"priority"`
	Type        string  `json:"type"`
	CreatedBy   string  `json:"created_by"`
	BudgetUSD   float64 `json:"budget_usd,omitempty"`
}

type scheduledJobResponse struct {
	ID             string                             `json:"id"`
	ProjectID      string                             `json:"project_id"`
	Name           string                             `json:"name"`
	CronExpression string                             `json:"cron_expression"`
	TicketTemplate scheduledJobTicketTemplateResponse `json:"ticket_template"`
	IsEnabled      bool                               `json:"is_enabled"`
	LastRunAt      *string                            `json:"last_run_at,omitempty"`
	NextRunAt      *string                            `json:"next_run_at,omitempty"`
}

func (s *Server) registerScheduledJobRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/scheduled-jobs", s.handleListScheduledJobs)
	api.POST("/projects/:projectId/scheduled-jobs", s.handleCreateScheduledJob)
	api.PATCH("/scheduled-jobs/:jobId", s.handleUpdateScheduledJob)
	api.DELETE("/scheduled-jobs/:jobId", s.handleDeleteScheduledJob)
	api.POST("/scheduled-jobs/:jobId/trigger", s.handleTriggerScheduledJob)
}

func (s *Server) handleListScheduledJobs(c echo.Context) error {
	if s.scheduledJobService == nil {
		return writeScheduledJobError(c, scheduledjobservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	items, err := s.scheduledJobService.List(c.Request().Context(), projectID)
	if err != nil {
		return writeScheduledJobError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"scheduled_jobs": mapScheduledJobResponses(items),
	})
}

func (s *Server) handleCreateScheduledJob(c echo.Context) error {
	if s.scheduledJobService == nil {
		return writeScheduledJobError(c, scheduledjobservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw rawCreateScheduledJobRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseCreateScheduledJobRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.scheduledJobService.Create(c.Request().Context(), input)
	if err != nil {
		return writeScheduledJobError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: item.ProjectID,
		EventType: activityevent.TypeScheduledJobCreated,
		Message:   "Created scheduled job " + item.Name,
		Metadata: map[string]any{
			"job_id":          item.ID.String(),
			"job_name":        item.Name,
			"cron_expression": item.CronExpression,
			"is_enabled":      item.IsEnabled,
			"changed_fields":  []string{"scheduled_job"},
		},
	}); err != nil {
		return writeScheduledJobError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"scheduled_job": mapScheduledJobResponse(item),
	})
}

func (s *Server) handleUpdateScheduledJob(c echo.Context) error {
	if s.scheduledJobService == nil {
		return writeScheduledJobError(c, scheduledjobservice.ErrUnavailable)
	}

	jobID, err := parseUUIDPathParamValue(c, "jobId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_JOB_ID", err.Error())
	}

	var raw rawUpdateScheduledJobRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateScheduledJobRequest(jobID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	current, err := s.scheduledJobService.Get(c.Request().Context(), jobID)
	if err != nil {
		return writeScheduledJobError(c, err)
	}

	item, err := s.scheduledJobService.Update(c.Request().Context(), input)
	if err != nil {
		return writeScheduledJobError(c, err)
	}
	activityInputs := make([]activitysvc.RecordInput, 0, 2)
	if raw.Name != nil || raw.CronExpression != nil || raw.TicketTemplate != nil {
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			EventType: activityevent.TypeScheduledJobUpdated,
			Message:   "Updated scheduled job " + item.Name,
			Metadata: map[string]any{
				"job_id":          item.ID.String(),
				"job_name":        item.Name,
				"cron_expression": item.CronExpression,
				"is_enabled":      item.IsEnabled,
				"changed_fields":  scheduledJobChangedFields(raw),
			},
		})
	}
	if raw.IsEnabled != nil && current.IsEnabled != item.IsEnabled {
		eventType := activityevent.TypeScheduledJobDisabled
		message := "Disabled scheduled job " + item.Name
		if item.IsEnabled {
			eventType = activityevent.TypeScheduledJobEnabled
			message = "Enabled scheduled job " + item.Name
		}
		activityInputs = append(activityInputs, activitysvc.RecordInput{
			ProjectID: item.ProjectID,
			EventType: eventType,
			Message:   message,
			Metadata: map[string]any{
				"job_id":         item.ID.String(),
				"job_name":       item.Name,
				"is_enabled":     item.IsEnabled,
				"changed_fields": []string{"is_enabled"},
			},
		})
	}
	if err := s.emitActivities(c.Request().Context(), activityInputs...); err != nil {
		return writeScheduledJobError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"scheduled_job": mapScheduledJobResponse(item),
	})
}

func (s *Server) handleDeleteScheduledJob(c echo.Context) error {
	if s.scheduledJobService == nil {
		return writeScheduledJobError(c, scheduledjobservice.ErrUnavailable)
	}

	jobID, err := parseUUIDPathParamValue(c, "jobId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_JOB_ID", err.Error())
	}
	current, err := s.scheduledJobService.Get(c.Request().Context(), jobID)
	if err != nil {
		return writeScheduledJobError(c, err)
	}

	result, err := s.scheduledJobService.Delete(c.Request().Context(), jobID)
	if err != nil {
		return writeScheduledJobError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: current.ProjectID,
		EventType: activityevent.TypeScheduledJobDeleted,
		Message:   "Deleted scheduled job " + current.Name,
		Metadata: map[string]any{
			"job_id":         current.ID.String(),
			"job_name":       current.Name,
			"changed_fields": []string{"scheduled_job"},
		},
	}); err != nil {
		return writeScheduledJobError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

func (s *Server) handleTriggerScheduledJob(c echo.Context) error {
	if s.scheduledJobService == nil {
		return writeScheduledJobError(c, scheduledjobservice.ErrUnavailable)
	}

	jobID, err := parseUUIDPathParamValue(c, "jobId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_JOB_ID", err.Error())
	}

	result, err := s.scheduledJobService.Trigger(c.Request().Context(), jobID)
	if err != nil {
		return writeScheduledJobError(c, err)
	}
	if err := s.publishTicketEvent(c.Request().Context(), ticketCreatedEventType, result.Ticket); err != nil {
		return writeScheduledJobError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: result.Job.ProjectID,
		TicketID:  &result.Ticket.ID,
		EventType: activityevent.TypeScheduledJobTriggered,
		Message:   "Triggered scheduled job " + result.Job.Name,
		Metadata: map[string]any{
			"job_id":            result.Job.ID.String(),
			"job_name":          result.Job.Name,
			"ticket_id":         result.Ticket.ID.String(),
			"ticket_identifier": result.Ticket.Identifier,
			"changed_fields":    []string{"trigger"},
		},
	}); err != nil {
		return writeScheduledJobError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"scheduled_job": mapScheduledJobResponse(result.Job),
		"ticket":        mapTicketResponse(result.Ticket),
	})
}

func scheduledJobChangedFields(raw rawUpdateScheduledJobRequest) []string {
	fields := make([]string, 0, 3)
	if raw.Name != nil {
		fields = append(fields, "name")
	}
	if raw.CronExpression != nil {
		fields = append(fields, "cron_expression")
	}
	if raw.TicketTemplate != nil {
		fields = append(fields, "ticket_template")
	}
	return fields
}

func writeScheduledJobError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, scheduledjobservice.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, scheduledjobservice.ErrProjectNotFound):
		return writeAPIError(c, http.StatusNotFound, "PROJECT_NOT_FOUND", err.Error())
	case errors.Is(err, scheduledjobservice.ErrWorkflowNotFound):
		return writeAPIError(c, http.StatusNotFound, "WORKFLOW_NOT_FOUND", err.Error())
	case errors.Is(err, scheduledjobservice.ErrScheduledJobNotFound):
		return writeAPIError(c, http.StatusNotFound, "SCHEDULED_JOB_NOT_FOUND", err.Error())
	case errors.Is(err, scheduledjobservice.ErrScheduledJobConflict):
		return writeAPIError(c, http.StatusConflict, "SCHEDULED_JOB_CONFLICT", err.Error())
	case errors.Is(err, scheduledjobservice.ErrStatusNotFound):
		return writeAPIError(c, http.StatusBadRequest, "STATUS_NOT_FOUND", err.Error())
	case errors.Is(err, scheduledjobservice.ErrInvalidCronExpression):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_CRON_EXPRESSION", err.Error())
	case errors.Is(err, scheduledjobservice.ErrInvalidTicketTemplate):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_TEMPLATE", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func mapScheduledJobResponses(items []scheduledjobservice.ScheduledJob) []scheduledJobResponse {
	response := make([]scheduledJobResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapScheduledJobResponse(item))
	}

	return response
}

func mapScheduledJobResponse(item scheduledjobservice.ScheduledJob) scheduledJobResponse {
	response := scheduledJobResponse{
		ID:             item.ID.String(),
		ProjectID:      item.ProjectID.String(),
		Name:           item.Name,
		CronExpression: item.CronExpression,
		TicketTemplate: scheduledJobTicketTemplateResponse{
			Title:       item.TicketTemplate.Title,
			Description: item.TicketTemplate.Description,
			Status:      item.TicketTemplate.Status,
			Priority:    string(item.TicketTemplate.Priority),
			Type:        string(item.TicketTemplate.Type),
			CreatedBy:   item.TicketTemplate.CreatedBy,
			BudgetUSD:   item.TicketTemplate.BudgetUSD,
		},
		IsEnabled: item.IsEnabled,
	}
	if item.LastRunAt != nil {
		lastRunAt := item.LastRunAt.UTC().Format(time.RFC3339)
		response.LastRunAt = &lastRunAt
	}
	if item.NextRunAt != nil {
		nextRunAt := item.NextRunAt.UTC().Format(time.RFC3339)
		response.NextRunAt = &nextRunAt
	}

	return response
}
