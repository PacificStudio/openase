package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
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

type ticketExternalLinkResponse struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	URL        string `json:"url"`
	ExternalID string `json:"external_id"`
	Title      string `json:"title,omitempty"`
	Status     string `json:"status,omitempty"`
	Relation   string `json:"relation"`
	CreatedAt  string `json:"created_at"`
}

type ticketCommentResponse struct {
	ID        string  `json:"id"`
	TicketID  string  `json:"ticket_id"`
	Body      string  `json:"body"`
	CreatedBy string  `json:"created_by"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt *string `json:"updated_at,omitempty"`
}

type ticketResponse struct {
	ID                string                       `json:"id"`
	ProjectID         string                       `json:"project_id"`
	Identifier        string                       `json:"identifier"`
	Title             string                       `json:"title"`
	Description       string                       `json:"description"`
	StatusID          string                       `json:"status_id"`
	StatusName        string                       `json:"status_name"`
	Priority          string                       `json:"priority"`
	Type              string                       `json:"type"`
	WorkflowID        *string                      `json:"workflow_id,omitempty"`
	CurrentRunID      *string                      `json:"current_run_id,omitempty"`
	TargetMachineID   *string                      `json:"target_machine_id,omitempty"`
	CreatedBy         string                       `json:"created_by"`
	Parent            *ticketReferenceResponse     `json:"parent,omitempty"`
	Children          []ticketReferenceResponse    `json:"children"`
	Dependencies      []ticketDependencyResponse   `json:"dependencies"`
	ExternalLinks     []ticketExternalLinkResponse `json:"external_links"`
	ExternalRef       string                       `json:"external_ref"`
	BudgetUSD         float64                      `json:"budget_usd"`
	CostTokensInput   int64                        `json:"cost_tokens_input"`
	CostTokensOutput  int64                        `json:"cost_tokens_output"`
	CostAmount        float64                      `json:"cost_amount"`
	AttemptCount      int                          `json:"attempt_count"`
	ConsecutiveErrors int                          `json:"consecutive_errors"`
	StartedAt         *string                      `json:"started_at,omitempty"`
	CompletedAt       *string                      `json:"completed_at,omitempty"`
	NextRetryAt       *string                      `json:"next_retry_at,omitempty"`
	RetryPaused       bool                         `json:"retry_paused"`
	PauseReason       string                       `json:"pause_reason,omitempty"`
	CreatedAt         string                       `json:"created_at"`
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

type ticketAssignedAgentResponse struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Provider            string  `json:"provider"`
	RuntimeControlState string  `json:"runtime_control_state,omitempty"`
	RuntimePhase        *string `json:"runtime_phase,omitempty"`
}

const ticketCommentEventType = "comment_added"

func (s *Server) registerTicketRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/tickets", s.handleListTickets)
	api.POST("/projects/:projectId/tickets", s.handleCreateTicket)
	api.GET("/projects/:projectId/tickets/:ticketId/detail", s.handleGetTicketDetail)
	api.GET("/tickets/:ticketId", s.handleGetTicket)
	api.PATCH("/tickets/:ticketId", s.handleUpdateTicket)
	api.POST("/tickets/:ticketId/comments", s.handleCreateTicketComment)
	api.PATCH("/tickets/:ticketId/comments/:commentId", s.handleUpdateTicketComment)
	api.DELETE("/tickets/:ticketId/comments/:commentId", s.handleDeleteTicketComment)
	api.POST("/tickets/:ticketId/dependencies", s.handleAddTicketDependency)
	api.DELETE("/tickets/:ticketId/dependencies/:dependencyId", s.handleDeleteTicketDependency)
	api.POST("/tickets/:ticketId/external-links", s.handleAddTicketExternalLink)
	api.DELETE("/tickets/:ticketId/external-links/:externalLinkId", s.handleDeleteTicketExternalLink)
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
	comments, err := s.ticketService.ListComments(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
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

	assignedAgent, err := s.loadTicketAssignedAgent(c.Request().Context(), item)
	if err != nil {
		return writeCatalogError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"assigned_agent": assignedAgent,
		"ticket":         mapTicketResponse(item),
		"repo_scopes":    mapTicketRepoScopeDetailResponses(repoScopes, indexProjectRepoResponses(projectRepos)),
		"comments":       mapTicketCommentResponses(comments),
		"activity":       mapActivityEventResponses(filterNonCommentActivityEvents(activityItems)),
		"hook_history":   mapActivityEventResponses(filterHookActivityEvents(filterNonCommentActivityEvents(activityItems))),
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

func (s *Server) handleCreateTicketComment(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}

	var raw rawCreateTicketCommentRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseCreateTicketCommentRequest(ticketID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	comment, err := s.ticketService.AddComment(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	if err := s.publishTicketUpdatedByID(c.Request().Context(), ticketID); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"comment": mapTicketCommentResponse(comment),
	})
}

func (s *Server) handleUpdateTicketComment(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}
	commentID, err := parseCommentID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_COMMENT_ID", err.Error())
	}

	var raw rawUpdateTicketCommentRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseUpdateTicketCommentRequest(ticketID, commentID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	comment, err := s.ticketService.UpdateComment(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	if err := s.publishTicketUpdatedByID(c.Request().Context(), ticketID); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"comment": mapTicketCommentResponse(comment),
	})
}

func (s *Server) handleDeleteTicketComment(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}
	commentID, err := parseCommentID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_COMMENT_ID", err.Error())
	}

	result, err := s.ticketService.RemoveComment(c.Request().Context(), ticketID, commentID)
	if err != nil {
		return writeTicketError(c, err)
	}
	if err := s.publishTicketUpdatedByID(c.Request().Context(), ticketID); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, result)
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

func (s *Server) handleAddTicketExternalLink(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}

	var raw rawAddExternalLinkRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := parseAddExternalLinkRequest(ticketID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	externalLink, err := s.ticketService.AddExternalLink(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"external_link": mapTicketExternalLinkResponse(externalLink),
	})
}

func (s *Server) handleDeleteTicketExternalLink(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}
	externalLinkID, err := parseExternalLinkID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_EXTERNAL_LINK_ID", err.Error())
	}

	result, err := s.ticketService.RemoveExternalLink(c.Request().Context(), ticketID, externalLinkID)
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
	case errors.Is(err, ticketservice.ErrTicketConflict):
		return writeAPIError(c, http.StatusConflict, "TICKET_CONFLICT", err.Error())
	case errors.Is(err, ticketservice.ErrCommentNotFound):
		return writeAPIError(c, http.StatusNotFound, "COMMENT_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrDependencyNotFound):
		return writeAPIError(c, http.StatusNotFound, "DEPENDENCY_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrExternalLinkNotFound):
		return writeAPIError(c, http.StatusNotFound, "EXTERNAL_LINK_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrStatusNotFound):
		return writeAPIError(c, http.StatusBadRequest, "STATUS_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrStatusNotAllowed):
		return writeAPIError(c, http.StatusBadRequest, "STATUS_NOT_ALLOWED", err.Error())
	case errors.Is(err, ticketservice.ErrWorkflowNotFound):
		return writeAPIError(c, http.StatusBadRequest, "WORKFLOW_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrTargetMachineNotFound):
		return writeAPIError(c, http.StatusBadRequest, "TARGET_MACHINE_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrParentTicketNotFound):
		return writeAPIError(c, http.StatusBadRequest, "PARENT_TICKET_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrDependencyConflict):
		return writeAPIError(c, http.StatusConflict, "DEPENDENCY_CONFLICT", err.Error())
	case errors.Is(err, ticketservice.ErrExternalLinkConflict):
		return writeAPIError(c, http.StatusConflict, "EXTERNAL_LINK_CONFLICT", err.Error())
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

func mapTicketCommentResponses(items []ticketservice.Comment) []ticketCommentResponse {
	response := make([]ticketCommentResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapTicketCommentResponse(item))
	}

	return response
}

func mapTicketCommentResponse(item ticketservice.Comment) ticketCommentResponse {
	var updatedAt *string
	if !item.UpdatedAt.IsZero() && !item.UpdatedAt.Equal(item.CreatedAt) {
		formatted := item.UpdatedAt.UTC().Format(time.RFC3339)
		updatedAt = &formatted
	}

	return ticketCommentResponse{
		ID:        item.ID.String(),
		TicketID:  item.TicketID.String(),
		Body:      item.Body,
		CreatedBy: item.CreatedBy,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: updatedAt,
	}
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

func filterNonCommentActivityEvents(items []domain.ActivityEvent) []domain.ActivityEvent {
	filtered := make([]domain.ActivityEvent, 0, len(items))
	for _, item := range items {
		if item.EventType == ticketCommentEventType {
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
		ExternalLinks:     []ticketExternalLinkResponse{},
		ExternalRef:       item.ExternalRef,
		BudgetUSD:         item.BudgetUSD,
		CostTokensInput:   item.CostTokensInput,
		CostTokensOutput:  item.CostTokensOutput,
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
	if item.CurrentRunID != nil {
		currentRunID := item.CurrentRunID.String()
		response.CurrentRunID = &currentRunID
	}
	if item.TargetMachineID != nil {
		targetMachineID := item.TargetMachineID.String()
		response.TargetMachineID = &targetMachineID
	}
	if item.NextRetryAt != nil {
		nextRetryAt := item.NextRetryAt.UTC().Format(time.RFC3339)
		response.NextRetryAt = &nextRetryAt
	}
	if item.StartedAt != nil {
		startedAt := item.StartedAt.UTC().Format(time.RFC3339)
		response.StartedAt = &startedAt
	}
	if item.CompletedAt != nil {
		completedAt := item.CompletedAt.UTC().Format(time.RFC3339)
		response.CompletedAt = &completedAt
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
	for _, externalLink := range item.ExternalLinks {
		response.ExternalLinks = append(response.ExternalLinks, mapTicketExternalLinkResponse(externalLink))
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

func mapTicketExternalLinkResponse(item ticketservice.ExternalLink) ticketExternalLinkResponse {
	return ticketExternalLinkResponse{
		ID:         item.ID.String(),
		Type:       item.LinkType.String(),
		URL:        item.URL,
		ExternalID: item.ExternalID,
		Title:      item.Title,
		Status:     item.Status,
		Relation:   item.Relation.String(),
		CreatedAt:  item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *Server) loadTicketAssignedAgent(ctx context.Context, item ticketservice.Ticket) (*ticketAssignedAgentResponse, error) {
	if item.CurrentRunID == nil {
		return nil, nil
	}

	runItem, err := s.catalog.GetAgentRun(ctx, *item.CurrentRunID)
	if err != nil {
		return nil, err
	}

	agentItem, err := s.catalog.GetAgent(ctx, runItem.AgentID)
	if err != nil {
		return nil, err
	}

	providerItem, err := s.catalog.GetAgentProvider(ctx, agentItem.ProviderID)
	if err != nil {
		return nil, err
	}

	return mapTicketAssignedAgentResponse(agentItem, providerItem), nil
}

func mapTicketAssignedAgentResponse(agentItem domain.Agent, providerItem domain.AgentProvider) *ticketAssignedAgentResponse {
	response := &ticketAssignedAgentResponse{
		ID:                  agentItem.ID.String(),
		Name:                agentItem.Name,
		Provider:            providerItem.Name,
		RuntimeControlState: agentItem.RuntimeControlState.String(),
	}
	if agentItem.Runtime != nil {
		runtimePhase := agentItem.Runtime.RuntimePhase.String()
		response.RuntimePhase = &runtimePhase
	}

	return response
}

func mapDependencyType(value string) string {
	switch value {
	case "sub-issue":
		return "sub_issue"
	default:
		return value
	}
}

func (s *Server) publishTicketUpdatedByID(ctx context.Context, ticketID uuid.UUID) error {
	item, err := s.ticketService.Get(ctx, ticketID)
	if err != nil {
		return err
	}

	return s.publishTicketEvent(ctx, ticketUpdatedEventType, item)
}
