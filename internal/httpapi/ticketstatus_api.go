package httpapi

import (
	"errors"
	"net/http"

	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerTicketStatusRoutes() {
	s.echo.GET("/api/v1/projects/:projectId/statuses", s.handleListTicketStatuses)
	s.echo.POST("/api/v1/projects/:projectId/statuses", s.handleCreateTicketStatus)
	s.echo.POST("/api/v1/projects/:projectId/statuses/reset", s.handleResetTicketStatuses)
	s.echo.PATCH("/api/v1/statuses/:statusId", s.handleUpdateTicketStatus)
	s.echo.DELETE("/api/v1/statuses/:statusId", s.handleDeleteTicketStatus)
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

	status, err := service.Update(c.Request().Context(), input)
	if err != nil {
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

	result, err := service.Delete(c.Request().Context(), statusID)
	if err != nil {
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
	return c.JSON(statusCode, map[string]string{
		"code":    code,
		"message": message,
	})
}

func writeAPIErrorWithDetails(c echo.Context, statusCode int, code string, message string, details any) error {
	payload := map[string]any{
		"code":    code,
		"message": message,
	}
	if details != nil {
		payload["details"] = details
	}
	return c.JSON(statusCode, payload)
}
