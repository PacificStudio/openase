package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	projectupdatedomain "github.com/BetterAndBetterII/openase/internal/domain/projectupdate"
	projectupdateservice "github.com/BetterAndBetterII/openase/internal/projectupdate"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const agentClaimsContextKey = "agent_platform_claims"

func (s *Server) registerAgentPlatformRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/tickets", s.handleAgentListTickets)
	api.GET("/projects/:projectId/workflows", s.handleAgentListProjectWorkflows)
	api.GET("/projects/:projectId/updates", s.handleAgentListProjectUpdates)
	api.POST("/projects/:projectId/tickets", s.handleAgentCreateTicket)
	api.PATCH("/projects/:projectId/tickets/:ticketId", s.handleAgentUpdateProjectTicket)
	api.POST("/projects/:projectId/updates", s.handleAgentCreateProjectUpdateThread)
	api.GET("/tickets/:ticketId", s.handleAgentGetOwnTicket)
	api.PATCH("/tickets/:ticketId", s.handleAgentUpdateOwnTicket)
	api.GET("/tickets/:ticketId/comments", s.handleAgentListOwnTicketComments)
	api.POST("/tickets/:ticketId/comments", s.handleAgentCreateOwnTicketComment)
	api.PATCH("/tickets/:ticketId/comments/:commentId", s.handleAgentUpdateOwnTicketComment)
	api.POST("/tickets/:ticketId/usage", s.handleAgentReportUsage)
	api.PATCH("/projects/:projectId", s.handleAgentUpdateProject)
	api.PATCH("/projects/:projectId/updates/:threadId", s.handleAgentUpdateProjectUpdateThread)
	api.DELETE("/projects/:projectId/updates/:threadId", s.handleAgentDeleteProjectUpdateThread)
	api.POST("/projects/:projectId/updates/:threadId/comments", s.handleAgentCreateProjectUpdateComment)
	api.PATCH("/projects/:projectId/updates/:threadId/comments/:commentId", s.handleAgentUpdateProjectUpdateComment)
	api.DELETE("/projects/:projectId/updates/:threadId/comments/:commentId", s.handleAgentDeleteProjectUpdateComment)
	api.POST("/projects/:projectId/repos", s.handleAgentCreateProjectRepo)
	s.registerExpandedAgentPlatformRoutes(api)
}

func (s *Server) authenticateAgentToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if s.agentPlatform == nil {
			return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "agent platform service unavailable")
		}

		rawToken, err := agentplatform.ParseBearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
		if err != nil {
			return writeAPIError(c, http.StatusUnauthorized, "INVALID_AGENT_TOKEN", err.Error())
		}

		claims, err := s.agentPlatform.Authenticate(c.Request().Context(), rawToken)
		if err != nil {
			return writeAgentPlatformError(c, err)
		}

		c.Set(agentClaimsContextKey, claims)
		return next(c)
	}
}

func (s *Server) handleAgentListTickets(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	claims, ok := requireAgentAnyScope(c, agentplatform.ScopeTicketsList, agentplatform.ScopeWorkflowsList)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	return s.handleListTickets(c)
}

func (s *Server) handleAgentCreateTicket(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeTicketsCreate)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	var raw rawAgentCreateTicketRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseAgentCreateTicketRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	input.CreatedBy = claims.CreatedBy()

	item, err := s.ticketService.Create(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	if err := s.publishTicketEvent(c.Request().Context(), ticketCreatedEventType, item); err != nil {
		return writeTicketError(c, err)
	}
	if item.Parent != nil {
		if err := s.publishTicketUpdatesByID(c.Request().Context(), item.Parent.ID); err != nil {
			return writeTicketError(c, err)
		}
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"ticket": mapTicketResponse(item),
	})
}

func (s *Server) handleAgentListProjectWorkflows(c echo.Context) error {
	if s.workflowService == nil {
		return writeWorkflowError(c, workflowservice.ErrUnavailable)
	}

	claims, ok := requireAgentAnyScope(c, agentplatform.ScopeTicketsList, agentplatform.ScopeWorkflowsList)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	items, err := s.workflowService.List(c.Request().Context(), projectID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"workflows": mapWorkflowResponses(items),
	})
}

func (s *Server) handleAgentListProjectUpdates(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeTicketsList)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	pageInput, err := parseListProjectUpdatesPageRequest(projectID, projectupdatedomain.ListThreadsPageRequest{
		Limit:  c.QueryParam("limit"),
		Before: c.QueryParam("before"),
	})
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_UPDATES_PAGE", err.Error())
	}

	page, err := s.projectUpdateService.ListThreadPage(c.Request().Context(), pageInput)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"threads":     mapProjectUpdateThreadResponses(page.Threads),
		"next_cursor": page.NextCursor,
		"has_more":    page.HasMore,
	})
}

func (s *Server) handleAgentGetOwnTicket(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	claims, current, ok := s.requireAgentOwnTicket(c, agentplatform.ScopeTicketsList)
	if !ok {
		return nil
	}
	if current.ProjectID != claims.ProjectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"ticket": mapTicketResponse(current),
	})
}

func (s *Server) handleAgentUpdateOwnTicket(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	claims, current, ok := s.requireAgentOwnTicket(c, agentplatform.ScopeTicketsUpdateSelf)
	if !ok {
		return nil
	}
	if current.ProjectID != claims.ProjectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	return s.handleAgentTicketUpdate(c, claims, current)
}

func (s *Server) handleAgentUpdateProjectTicket(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeTicketsUpdate)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}
	current, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
	}
	if current.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	return s.handleAgentTicketUpdate(c, claims, current)
}

func (s *Server) handleAgentTicketUpdate(c echo.Context, claims agentplatform.Claims, current ticketservice.Ticket) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	var raw rawAgentUpdateTicketRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	resolveStatusID := func(ctx context.Context, projectID uuid.UUID, statusName string) (uuid.UUID, error) {
		if s.ticketStatusService == nil {
			return uuid.UUID{}, ticketstatus.ErrUnavailable
		}
		return s.ticketStatusService.ResolveStatusIDByName(ctx, projectID, statusName)
	}
	input, err := parseAgentUpdateTicketRequest(c.Request().Context(), current.ProjectID, current.ID, claims.CreatedBy(), raw, resolveStatusID)
	if err != nil {
		var statusNameErr agentStatusNameResolutionError
		if errors.As(err, &statusNameErr) || errors.Is(err, ticketstatus.ErrUnavailable) {
			return writeTicketStatusError(c, err)
		}
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	var previous ticketservice.Ticket
	if input.ParentTicketID.Set {
		previous = current
	}

	item, err := s.ticketService.Update(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	eventTypes := ticketMutationEventTypes(input, item)
	if err := s.publishTicketEvents(c.Request().Context(), eventTypes, item); err != nil {
		return writeTicketError(c, err)
	}
	if input.ParentTicketID.Set {
		if err := s.publishParentRelationshipUpdates(c.Request().Context(), previous, item); err != nil {
			return writeTicketError(c, err)
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"ticket": mapTicketResponse(item),
	})
}

func (s *Server) handleAgentReportUsage(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	claims, current, ok := s.requireAgentOwnTicket(c, agentplatform.ScopeTicketsReportUsage)
	if !ok {
		return nil
	}
	if current.ProjectID != claims.ProjectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	var raw rawAgentReportUsageRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	result, err := s.ticketService.RecordUsage(c.Request().Context(), ticketservice.RecordUsageInput{
		AgentID:  claims.AgentID,
		TicketID: current.ID,
		Usage:    parseAgentReportUsageRequest(raw),
	}, s.metrics)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	if err := s.publishTicketEvent(c.Request().Context(), ticketUpdatedEventType, result.Ticket); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"ticket":          mapTicketResponse(result.Ticket),
		"applied":         result.Applied,
		"budget_exceeded": result.BudgetExceeded,
	})
}

func (s *Server) handleAgentListOwnTicketComments(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	claims, current, ok := s.requireAgentOwnTicket(c, agentplatform.ScopeTicketsUpdateSelf)
	if !ok {
		return nil
	}
	if current.ProjectID != claims.ProjectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	comments, err := s.ticketService.ListComments(c.Request().Context(), current.ID)
	if err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"comments": mapTicketCommentResponses(comments),
	})
}

func (s *Server) handleAgentCreateOwnTicketComment(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	claims, current, ok := s.requireAgentOwnTicket(c, agentplatform.ScopeTicketsUpdateSelf)
	if !ok {
		return nil
	}
	if current.ProjectID != claims.ProjectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	var raw rawAgentTicketCommentRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseAgentCreateTicketCommentRequest(current.ID, claims.CreatedBy(), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	comment, err := s.ticketService.AddComment(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	commentResponse := mapTicketCommentResponse(comment)
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: current.ProjectID,
		TicketID:  &current.ID,
		EventType: activityevent.TypeTicketCommentCreated,
		Message:   "Added comment to " + current.Identifier,
		Metadata:  ticketCommentMetadata(commentResponse),
	}); err != nil {
		return writeTicketError(c, err)
	}
	if err := s.publishTicketUpdatedByID(c.Request().Context(), current.ID); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"comment": commentResponse,
	})
}

func (s *Server) handleAgentUpdateOwnTicketComment(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	claims, current, ok := s.requireAgentOwnTicket(c, agentplatform.ScopeTicketsUpdateSelf)
	if !ok {
		return nil
	}
	if current.ProjectID != claims.ProjectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	commentID, err := parseCommentID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_COMMENT_ID", err.Error())
	}

	var raw rawAgentTicketCommentRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseAgentUpdateTicketCommentRequest(current.ID, commentID, claims.CreatedBy(), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	comment, err := s.ticketService.UpdateComment(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	commentResponse := mapTicketCommentResponse(comment)
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: current.ProjectID,
		TicketID:  &current.ID,
		EventType: activityevent.TypeTicketCommentEdited,
		Message:   "Edited comment on " + current.Identifier,
		Metadata:  ticketCommentMetadata(commentResponse),
	}); err != nil {
		return writeTicketError(c, err)
	}
	if err := s.publishTicketUpdatedByID(c.Request().Context(), current.ID); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"comment": commentResponse,
	})
}

func (s *Server) handleAgentUpdateProject(c echo.Context) error {
	if s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeProjectsUpdate)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	current, err := s.catalog.GetProject(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	var raw rawAgentProjectPatchRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}
	input, err := parseAgentProjectPatchRequest(projectID, current, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.catalog.UpdateProject(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}
	if err := s.emitActivities(c.Request().Context(), buildProjectPatchActivityInputs(current, item)...); err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"project": mapProjectResponse(item),
	})
}

func (s *Server) handleAgentCreateProjectUpdateThread(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeProjectsUpdate)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	var raw rawAgentCreateProjectUpdateThreadRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseAgentCreateProjectUpdateThreadRequest(projectID, claims.CreatedBy(), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.projectUpdateService.AddThread(c.Request().Context(), input)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"thread": mapProjectUpdateThreadResponse(item),
	})
}

func (s *Server) handleAgentUpdateProjectUpdateThread(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeProjectsUpdate)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}

	var raw rawAgentUpdateProjectUpdateThreadRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseAgentUpdateProjectUpdateThreadRequest(projectID, threadID, claims.CreatedBy(), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.projectUpdateService.UpdateThread(c.Request().Context(), input)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"thread": mapProjectUpdateThreadResponse(item),
	})
}

func (s *Server) handleAgentDeleteProjectUpdateThread(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeProjectsUpdate)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}

	result, err := s.projectUpdateService.RemoveThread(c.Request().Context(), projectID, threadID)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"deleted_thread_id": result.DeletedThreadID.String(),
	})
}

func (s *Server) handleAgentCreateProjectUpdateComment(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeProjectsUpdate)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}

	var raw rawAgentCreateProjectUpdateCommentRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseAgentCreateProjectUpdateCommentRequest(projectID, threadID, claims.CreatedBy(), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.projectUpdateService.AddComment(c.Request().Context(), input)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"comment": mapProjectUpdateCommentResponse(item),
	})
}

func (s *Server) handleAgentUpdateProjectUpdateComment(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeProjectsUpdate)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}
	commentID, err := parseUUIDPathParam(c, "commentId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_COMMENT_ID", err.Error())
	}

	var raw rawAgentUpdateProjectUpdateCommentRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseAgentUpdateProjectUpdateCommentRequest(projectID, threadID, commentID, claims.CreatedBy(), raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.projectUpdateService.UpdateComment(c.Request().Context(), input)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"comment": mapProjectUpdateCommentResponse(item),
	})
}

func (s *Server) handleAgentDeleteProjectUpdateComment(c echo.Context) error {
	if s.projectUpdateService == nil {
		return writeProjectUpdateError(c, projectupdateservice.ErrUnavailable)
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeProjectsUpdate)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	threadID, err := parseUUIDPathParam(c, "threadId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_THREAD_ID", err.Error())
	}
	commentID, err := parseUUIDPathParam(c, "commentId")
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_COMMENT_ID", err.Error())
	}

	result, err := s.projectUpdateService.RemoveComment(c.Request().Context(), projectID, threadID, commentID)
	if err != nil {
		return writeProjectUpdateError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"deleted_comment_id": result.DeletedCommentID.String(),
	})
}

func (s *Server) handleAgentCreateProjectRepo(c echo.Context) error {
	if s.catalog.Empty() {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}

	claims, ok := requireAgentAnyScope(c, agentplatform.ScopeProjectsAddRepo, agentplatform.ScopeReposCreate)
	if !ok {
		return nil
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	if claims.ProjectID != projectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	var request domain.ProjectRepoInput
	if err := decodeJSON(c, &request); err != nil {
		return err
	}

	input, err := domain.ParseCreateProjectRepo(projectID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.CreateProjectRepo(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"repo": mapProjectRepoResponse(item),
	})
}

func (s *Server) requireAgentOwnTicket(c echo.Context, scope agentplatform.Scope) (agentplatform.Claims, ticketservice.Ticket, bool) {
	claims, ok := requireAgentScope(c, scope)
	if !ok {
		return agentplatform.Claims{}, ticketservice.Ticket{}, false
	}
	if !claims.IsTicketAgent() {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_PRINCIPAL_KIND_FORBIDDEN", "project conversation principals cannot access ticket-runtime-only endpoints")
		return agentplatform.Claims{}, ticketservice.Ticket{}, false
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		_ = writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
		return agentplatform.Claims{}, ticketservice.Ticket{}, false
	}
	if claims.TicketID != ticketID {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_TICKET_FORBIDDEN", "agent token can only access its current ticket")
		return agentplatform.Claims{}, ticketservice.Ticket{}, false
	}

	current, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		_ = writeTicketError(c, err)
		return agentplatform.Claims{}, ticketservice.Ticket{}, false
	}

	return claims, current, true
}

func requireAgentScope(c echo.Context, scope agentplatform.Scope) (agentplatform.Claims, bool) {
	return requireAgentAnyScope(c, scope)
}

func requireAgentAnyScope(c echo.Context, scopes ...agentplatform.Scope) (agentplatform.Claims, bool) {
	claims, ok := c.Get(agentClaimsContextKey).(agentplatform.Claims)
	if !ok {
		_ = writeAPIError(c, http.StatusUnauthorized, "INVALID_AGENT_TOKEN", "agent claims missing from request context")
		return agentplatform.Claims{}, false
	}
	for _, scope := range scopes {
		if claims.HasScope(scope) {
			return claims, true
		}
	}
	if len(scopes) == 0 {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_SCOPE_FORBIDDEN", "agent token is missing required scope")
		return agentplatform.Claims{}, false
	}
	required := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		required = append(required, string(scope))
	}
	if len(required) == 1 {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_SCOPE_FORBIDDEN", "agent token is missing required scope "+required[0])
		return agentplatform.Claims{}, false
	}
	_ = writeAPIError(c, http.StatusForbidden, "AGENT_SCOPE_FORBIDDEN", "agent token is missing required scope one of "+strings.Join(required, ", "))
	return agentplatform.Claims{}, false
}

func writeAgentPlatformError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, agentplatform.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, agentplatform.ErrInvalidToken):
		return writeAPIError(c, http.StatusUnauthorized, "INVALID_AGENT_TOKEN", err.Error())
	case errors.Is(err, agentplatform.ErrTokenNotFound):
		return writeAPIError(c, http.StatusUnauthorized, "AGENT_TOKEN_NOT_FOUND", err.Error())
	case errors.Is(err, agentplatform.ErrTokenExpired):
		return writeAPIError(c, http.StatusUnauthorized, "AGENT_TOKEN_EXPIRED", err.Error())
	case errors.Is(err, agentplatform.ErrInvalidScope):
		return writeAPIError(c, http.StatusForbidden, "AGENT_SCOPE_INVALID", err.Error())
	case errors.Is(err, agentplatform.ErrInvalidPrincipal):
		return writeAPIError(c, http.StatusUnauthorized, "AGENT_PRINCIPAL_INVALID", err.Error())
	case errors.Is(err, agentplatform.ErrAgentNotFound):
		return writeAPIError(c, http.StatusUnauthorized, "AGENT_NOT_FOUND", err.Error())
	case errors.Is(err, agentplatform.ErrPrincipalNotFound):
		return writeAPIError(c, http.StatusUnauthorized, "AGENT_PRINCIPAL_NOT_FOUND", err.Error())
	case errors.Is(err, agentplatform.ErrProjectMismatch):
		return writeAPIError(c, http.StatusUnauthorized, "AGENT_PROJECT_MISMATCH", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
