package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/labstack/echo/v4"
)

type ticketReferenceResponse struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	StatusID   string `json:"status_id"`
	StatusName string `json:"status_name"`
}

type ticketDependencyResponse struct {
	ID     string                  `json:"id"`
	Type   string                  `json:"type"`
	Target ticketReferenceResponse `json:"target"`
}

type ticketResponse struct {
	ID                string                     `json:"id"`
	ProjectID         string                     `json:"project_id"`
	Identifier        string                     `json:"identifier"`
	Title             string                     `json:"title"`
	Description       string                     `json:"description"`
	StatusID          string                     `json:"status_id"`
	StatusName        string                     `json:"status_name"`
	Priority          string                     `json:"priority"`
	Type              string                     `json:"type"`
	WorkflowID        *string                    `json:"workflow_id,omitempty"`
	CreatedBy         string                     `json:"created_by"`
	Parent            *ticketReferenceResponse   `json:"parent,omitempty"`
	Children          []ticketReferenceResponse  `json:"children"`
	Dependencies      []ticketDependencyResponse `json:"dependencies"`
	ExternalRef       string                     `json:"external_ref"`
	BudgetUSD         float64                    `json:"budget_usd"`
	CostAmount        float64                    `json:"cost_amount"`
	AttemptCount      int                        `json:"attempt_count"`
	ConsecutiveErrors int                        `json:"consecutive_errors"`
	NextRetryAt       *string                    `json:"next_retry_at,omitempty"`
	RetryPaused       bool                       `json:"retry_paused"`
	PauseReason       string                     `json:"pause_reason,omitempty"`
	CreatedAt         string                     `json:"created_at"`
}

type ticketRepoScopeDetailResponse struct {
	ID             string               `json:"id"`
	TicketID       string               `json:"ticket_id"`
	RepoID         string               `json:"repo_id"`
	Repo           *projectRepoResponse `json:"repo,omitempty"`
	BranchName     string               `json:"branch_name"`
	PullRequestURL *string              `json:"pull_request_url,omitempty"`
	PrStatus       string               `json:"pr_status"`
	CiStatus       string               `json:"ci_status"`
	IsPrimaryScope bool                 `json:"is_primary_scope"`
}

func (s *Server) registerTicketRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/tickets", s.handleListTickets)
	api.POST("/projects/:projectId/tickets", s.handleCreateTicket)
	api.GET("/projects/:projectId/tickets/:ticketId/detail", s.handleGetTicketDetail)
	api.GET("/tickets/:ticketId", s.handleGetTicket)
	api.PATCH("/tickets/:ticketId", s.handleUpdateTicket)
	api.POST("/tickets/:ticketId/dependencies", s.handleAddTicketDependency)
	api.DELETE("/tickets/:ticketId/dependencies/:dependencyId", s.handleDeleteTicketDependency)
}

func (s *Server) handleListTickets(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	parsedPriorities := make([]entticket.Priority, 0, len(parseCSVQueryValues(c, "priority")))
	for _, raw := range parseCSVQueryValues(c, "priority") {
		priority, parseErr := parseTicketPriority(raw)
		if parseErr != nil {
			return writeAPIError(c, http.StatusBadRequest, "INVALID_PRIORITY", parseErr.Error())
		}
		parsedPriorities = append(parsedPriorities, priority)
	}

	input := ticketservice.ListInput{
		ProjectID:   projectID,
		StatusNames: parseCSVQueryValues(c, "status_name"),
		Priorities:  make([]entticket.Priority, 0, len(parsedPriorities)),
		Limit:       0,
	}
	input.Priorities = append(input.Priorities, parsedPriorities...)

	items, err := s.ticketService.List(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"tickets": mapTicketResponses(items),
	})
}

func (s *Server) handleCreateTicket(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw rawCreateTicketRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseCreateTicketRequest(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

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

func (s *Server) handleGetTicket(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}

	item, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"ticket": mapTicketResponse(item),
	})
}

func (s *Server) handleGetTicketDetail(c echo.Context) error {
	if s.ticketService == nil || s.catalog == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "ticket detail service unavailable")
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}
	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}

	item, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
	}
	if item.ProjectID != projectID {
		return writeTicketError(c, ticketservice.ErrTicketNotFound)
	}

	projectRepos, err := s.catalog.ListProjectRepos(c.Request().Context(), projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}
	repoScopes, err := s.catalog.ListTicketRepoScopes(c.Request().Context(), projectID, ticketID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	activityInput, err := domain.ParseListActivityEvents(projectID, domain.ActivityEventListInput{
		TicketID: ticketID.String(),
		Limit:    "100",
	})
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	activityItems, err := s.catalog.ListActivityEvents(c.Request().Context(), activityInput)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"ticket":       mapTicketResponse(item),
		"repo_scopes":  mapTicketRepoScopeDetailResponses(repoScopes, indexProjectRepoResponses(projectRepos)),
		"activity":     mapActivityEventResponses(activityItems),
		"hook_history": mapActivityEventResponses(filterHookActivityEvents(activityItems)),
	})
}

func (s *Server) handleUpdateTicket(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}

	var raw rawUpdateTicketRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateTicketRequest(ticketID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	item, err := s.ticketService.Update(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	eventType := ticketUpdatedEventType
	if input.StatusID.Set {
		eventType = ticketStatusEventType
	}
	if err := s.publishTicketEvent(c.Request().Context(), eventType, item); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"ticket": mapTicketResponse(item),
	})
}

func (s *Server) handleAddTicketDependency(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}

	var raw rawAddDependencyRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseAddDependencyRequest(ticketID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	dependency, err := s.ticketService.AddDependency(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"dependency": mapTicketDependencyResponse(dependency),
	})
}

func (s *Server) handleDeleteTicketDependency(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}
	dependencyID, err := parseDependencyID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_DEPENDENCY_ID", err.Error())
	}

	result, err := s.ticketService.RemoveDependency(c.Request().Context(), ticketID, dependencyID)
	if err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

func writeTicketError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, ticketservice.ErrUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, ticketservice.ErrProjectNotFound):
		return writeAPIError(c, http.StatusNotFound, "PROJECT_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrTicketNotFound):
		return writeAPIError(c, http.StatusNotFound, "TICKET_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrDependencyNotFound):
		return writeAPIError(c, http.StatusNotFound, "DEPENDENCY_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrStatusNotFound):
		return writeAPIError(c, http.StatusBadRequest, "STATUS_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrWorkflowNotFound):
		return writeAPIError(c, http.StatusBadRequest, "WORKFLOW_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrParentTicketNotFound):
		return writeAPIError(c, http.StatusBadRequest, "PARENT_TICKET_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrDependencyConflict):
		return writeAPIError(c, http.StatusConflict, "DEPENDENCY_CONFLICT", err.Error())
	case errors.Is(err, ticketservice.ErrInvalidDependency):
		return writeAPIError(c, http.StatusBadRequest, "INVALID_DEPENDENCY", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func mapTicketRepoScopeDetailResponses(
	items []domain.TicketRepoScope,
	reposByID map[string]projectRepoResponse,
) []ticketRepoScopeDetailResponse {
	response := make([]ticketRepoScopeDetailResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapTicketRepoScopeDetailResponse(item, reposByID[item.RepoID.String()]))
	}

	return response
}

func mapTicketRepoScopeDetailResponse(
	item domain.TicketRepoScope,
	repo projectRepoResponse,
) ticketRepoScopeDetailResponse {
	var repoResponse *projectRepoResponse
	if repo.ID != "" {
		copied := repo
		repoResponse = &copied
	}

	return ticketRepoScopeDetailResponse{
		ID:             item.ID.String(),
		TicketID:       item.TicketID.String(),
		RepoID:         item.RepoID.String(),
		Repo:           repoResponse,
		BranchName:     item.BranchName,
		PullRequestURL: item.PullRequestURL,
		PrStatus:       item.PrStatus.String(),
		CiStatus:       item.CiStatus.String(),
		IsPrimaryScope: item.IsPrimaryScope,
	}
}

func indexProjectRepoResponses(items []domain.ProjectRepo) map[string]projectRepoResponse {
	index := make(map[string]projectRepoResponse, len(items))
	for _, item := range items {
		response := mapProjectRepoResponse(item)
		index[response.ID] = response
	}

	return index
}

func filterHookActivityEvents(items []domain.ActivityEvent) []domain.ActivityEvent {
	filtered := make([]domain.ActivityEvent, 0, len(items))
	for _, item := range items {
		if !isHookActivityEvent(item) {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered
}

func isHookActivityEvent(item domain.ActivityEvent) bool {
	if strings.Contains(strings.ToLower(item.EventType), "hook") {
		return true
	}

	for _, key := range []string{"hook", "hook_name", "hook_stage", "hook_result", "hook_outcome"} {
		if _, ok := item.Metadata[key]; ok {
			return true
		}
	}

	return false
}

func mapTicketResponses(items []ticketservice.Ticket) []ticketResponse {
	response := make([]ticketResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapTicketResponse(item))
	}

	return response
}

func mapTicketResponse(item ticketservice.Ticket) ticketResponse {
	response := ticketResponse{
		ID:                item.ID.String(),
		ProjectID:         item.ProjectID.String(),
		Identifier:        item.Identifier,
		Title:             item.Title,
		Description:       item.Description,
		StatusID:          item.StatusID.String(),
		StatusName:        item.StatusName,
		Priority:          string(item.Priority),
		Type:              string(item.Type),
		CreatedBy:         item.CreatedBy,
		Children:          []ticketReferenceResponse{},
		Dependencies:      []ticketDependencyResponse{},
		ExternalRef:       item.ExternalRef,
		BudgetUSD:         item.BudgetUSD,
		CostAmount:        item.CostAmount,
		AttemptCount:      item.AttemptCount,
		ConsecutiveErrors: item.ConsecutiveErrors,
		RetryPaused:       item.RetryPaused,
		PauseReason:       item.PauseReason,
		CreatedAt:         item.CreatedAt.UTC().Format(time.RFC3339),
	}
	if item.WorkflowID != nil {
		workflowID := item.WorkflowID.String()
		response.WorkflowID = &workflowID
	}
	if item.NextRetryAt != nil {
		nextRetryAt := item.NextRetryAt.UTC().Format(time.RFC3339)
		response.NextRetryAt = &nextRetryAt
	}
	if item.Parent != nil {
		parent := mapTicketReferenceResponse(*item.Parent)
		response.Parent = &parent
	}
	for _, child := range item.Children {
		response.Children = append(response.Children, mapTicketReferenceResponse(child))
	}
	for _, dependency := range item.Dependencies {
		response.Dependencies = append(response.Dependencies, mapTicketDependencyResponse(dependency))
	}

	return response
}

func mapTicketDependencyResponse(item ticketservice.Dependency) ticketDependencyResponse {
	return ticketDependencyResponse{
		ID:     item.ID.String(),
		Type:   mapDependencyType(string(item.Type)),
		Target: mapTicketReferenceResponse(item.Target),
	}
}

func mapTicketReferenceResponse(item ticketservice.TicketReference) ticketReferenceResponse {
	return ticketReferenceResponse{
		ID:         item.ID.String(),
		Identifier: item.Identifier,
		Title:      item.Title,
		StatusID:   item.StatusID.String(),
		StatusName: item.StatusName,
	}
}

func mapDependencyType(value string) string {
	switch value {
	case "sub-issue":
		return "sub_issue"
	default:
		return value
	}
}
