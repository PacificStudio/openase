package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/labstack/echo/v4"
)

const agentClaimsContextKey = "agent_platform_claims"

type rawAgentCreateTicketRequest struct {
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	StatusID       *string  `json:"status_id"`
	Priority       *string  `json:"priority"`
	Type           *string  `json:"type"`
	WorkflowID     *string  `json:"workflow_id"`
	ParentTicketID *string  `json:"parent_ticket_id"`
	ExternalRef    *string  `json:"external_ref"`
	BudgetUSD      *float64 `json:"budget_usd"`
}

type rawAgentUpdateTicketRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	ExternalRef *string `json:"external_ref"`
}

type rawAgentProjectPatchRequest struct {
	Description *string `json:"description"`
}

func (s *Server) registerAgentPlatformRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/tickets", s.handleAgentListTickets)
	api.POST("/projects/:projectId/tickets", s.handleAgentCreateTicket)
	api.PATCH("/tickets/:ticketId", s.handleAgentUpdateOwnTicket)
	api.PATCH("/projects/:projectId", s.handleAgentUpdateProject)
	api.POST("/projects/:projectId/repos", s.handleAgentCreateProjectRepo)
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

	input, err := parseCreateTicketRequest(projectID, rawCreateTicketRequest{
		Title:          raw.Title,
		Description:    raw.Description,
		StatusID:       raw.StatusID,
		Priority:       raw.Priority,
		Type:           raw.Type,
		WorkflowID:     raw.WorkflowID,
		ParentTicketID: raw.ParentTicketID,
		ExternalRef:    raw.ExternalRef,
		BudgetUSD:      raw.BudgetUSD,
	})
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

	return c.JSON(http.StatusCreated, map[string]any{
		"ticket": mapTicketResponse(item),
	})
}

func (s *Server) handleAgentUpdateOwnTicket(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeTicketsUpdateSelf)
	if !ok {
		return nil
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}
	if claims.TicketID != ticketID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_TICKET_FORBIDDEN", "agent token can only update its current ticket")
	}

	current, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
	}
	if current.ProjectID != claims.ProjectID {
		return writeAPIError(c, http.StatusForbidden, "AGENT_PROJECT_FORBIDDEN", "agent token cannot access another project")
	}

	var raw rawAgentUpdateTicketRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input := ticketservice.UpdateInput{TicketID: ticketID}
	if raw.Title != nil {
		title := strings.TrimSpace(*raw.Title)
		if title == "" {
			return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "title must not be empty")
		}
		input.Title = ticketservice.Some(title)
	}
	if raw.Description != nil {
		input.Description = ticketservice.Some(strings.TrimSpace(*raw.Description))
	}
	if raw.ExternalRef != nil {
		input.ExternalRef = ticketservice.Some(strings.TrimSpace(*raw.ExternalRef))
	}
	input.CreatedBy = ticketservice.Some(claims.CreatedBy())

	item, err := s.ticketService.Update(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	if err := s.publishTicketEvent(c.Request().Context(), ticketUpdatedEventType, item); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"ticket": mapTicketResponse(item),
	})
}

func (s *Server) handleAgentUpdateProject(c echo.Context) error {
	if s.catalog == nil {
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
	if raw.Description == nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", "description is required")
	}

	request := domain.ProjectInput{
		Name:                   current.Name,
		Slug:                   current.Slug,
		Description:            strings.TrimSpace(*raw.Description),
		Status:                 current.Status.String(),
		DefaultWorkflowID:      uuidToStringPointer(current.DefaultWorkflowID),
		DefaultAgentProviderID: uuidToStringPointer(current.DefaultAgentProviderID),
		MaxConcurrentAgents:    intPointer(current.MaxConcurrentAgents),
	}
	input, err := domain.ParseUpdateProject(projectID, current.OrganizationID, request)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}

	item, err := s.catalog.UpdateProject(c.Request().Context(), input)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"project": mapProjectResponse(item),
	})
}

func (s *Server) handleAgentCreateProjectRepo(c echo.Context) error {
	if s.catalog == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "catalog service unavailable")
	}

	claims, ok := requireAgentScope(c, agentplatform.ScopeProjectsAddRepo)
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

func requireAgentScope(c echo.Context, scope agentplatform.Scope) (agentplatform.Claims, bool) {
	claims, ok := c.Get(agentClaimsContextKey).(agentplatform.Claims)
	if !ok {
		_ = writeAPIError(c, http.StatusUnauthorized, "INVALID_AGENT_TOKEN", "agent claims missing from request context")
		return agentplatform.Claims{}, false
	}
	if !claims.HasScope(scope) {
		_ = writeAPIError(c, http.StatusForbidden, "AGENT_SCOPE_FORBIDDEN", "agent token is missing required scope "+string(scope))
		return agentplatform.Claims{}, false
	}
	return claims, true
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
	case errors.Is(err, agentplatform.ErrAgentNotFound):
		return writeAPIError(c, http.StatusUnauthorized, "AGENT_NOT_FOUND", err.Error())
	case errors.Is(err, agentplatform.ErrProjectMismatch):
		return writeAPIError(c, http.StatusUnauthorized, "AGENT_PROJECT_MISMATCH", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
