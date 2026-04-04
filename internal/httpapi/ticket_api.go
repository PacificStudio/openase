package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	ticketingdomain "github.com/BetterAndBetterII/openase/internal/domain/ticketing"
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
	ID           string  `json:"id"`
	TicketID     string  `json:"ticket_id"`
	Body         string  `json:"body,omitempty"`
	BodyMarkdown string  `json:"body_markdown"`
	CreatedBy    string  `json:"created_by"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    *string `json:"updated_at,omitempty"`
	EditedAt     *string `json:"edited_at,omitempty"`
	EditCount    int     `json:"edit_count"`
	LastEditedBy *string `json:"last_edited_by,omitempty"`
	IsDeleted    bool    `json:"is_deleted"`
	DeletedAt    *string `json:"deleted_at,omitempty"`
	DeletedBy    *string `json:"deleted_by,omitempty"`
}

type ticketCommentRevisionResponse struct {
	ID             string  `json:"id"`
	CommentID      string  `json:"comment_id"`
	RevisionNumber int     `json:"revision_number"`
	BodyMarkdown   string  `json:"body_markdown"`
	EditedBy       string  `json:"edited_by"`
	EditedAt       string  `json:"edited_at"`
	EditReason     *string `json:"edit_reason,omitempty"`
}

type ticketTimelineItemResponse struct {
	ID            string         `json:"id"`
	TicketID      string         `json:"ticket_id"`
	ItemType      string         `json:"item_type"`
	ActorName     string         `json:"actor_name"`
	ActorType     string         `json:"actor_type"`
	Title         *string        `json:"title,omitempty"`
	BodyMarkdown  *string        `json:"body_markdown,omitempty"`
	BodyText      *string        `json:"body_text,omitempty"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	EditedAt      *string        `json:"edited_at,omitempty"`
	IsCollapsible bool           `json:"is_collapsible"`
	IsDeleted     bool           `json:"is_deleted"`
	Metadata      map[string]any `json:"metadata"`
}

type ticketResponse struct {
	ID                string                       `json:"id"`
	ProjectID         string                       `json:"project_id"`
	Identifier        string                       `json:"identifier"`
	Title             string                       `json:"title"`
	Description       string                       `json:"description"`
	StatusID          string                       `json:"status_id"`
	StatusName        string                       `json:"status_name"`
	Archived          bool                         `json:"archived"`
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

type archivedTicketsResponse struct {
	Tickets []ticketResponse `json:"tickets"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
}

type ticketWorkspaceResetResponse struct {
	Reset bool `json:"reset"`
}

type workspaceResetConflict interface {
	WorkspaceResetConflict() bool
}

type ticketRepoScopeDetailResponse struct {
	ID                  string               `json:"id"`
	TicketID            string               `json:"ticket_id"`
	RepoID              string               `json:"repo_id"`
	Repo                *projectRepoResponse `json:"repo,omitempty"`
	BranchName          string               `json:"branch_name"`
	DefaultBranch       string               `json:"default_branch"`
	EffectiveBranchName string               `json:"effective_branch_name"`
	BranchSource        string               `json:"branch_source"`
	PullRequestURL      *string              `json:"pull_request_url,omitempty"`
}

type ticketAssignedAgentResponse struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Provider            string  `json:"provider"`
	RuntimeControlState string  `json:"runtime_control_state,omitempty"`
	RuntimePhase        *string `json:"runtime_phase,omitempty"`
}

type ticketPickupDiagnosisReasonResponse struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type ticketPickupDiagnosisWorkflowResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	IsActive          bool   `json:"is_active"`
	PickupStatusMatch bool   `json:"pickup_status_match"`
}

type ticketPickupDiagnosisAgentResponse struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	RuntimeControlState string `json:"runtime_control_state"`
}

type ticketPickupDiagnosisProviderResponse struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	MachineID          string  `json:"machine_id"`
	MachineName        string  `json:"machine_name"`
	MachineStatus      string  `json:"machine_status"`
	AvailabilityState  string  `json:"availability_state"`
	AvailabilityReason *string `json:"availability_reason,omitempty"`
}

type ticketPickupDiagnosisRetryResponse struct {
	AttemptCount int     `json:"attempt_count"`
	RetryPaused  bool    `json:"retry_paused"`
	PauseReason  string  `json:"pause_reason,omitempty"`
	NextRetryAt  *string `json:"next_retry_at,omitempty"`
}

type ticketPickupDiagnosisCapacityBucketResponse struct {
	Limited    bool `json:"limited"`
	ActiveRuns int  `json:"active_runs"`
	Capacity   int  `json:"capacity"`
}

type ticketPickupDiagnosisStatusCapacityResponse struct {
	Limited    bool `json:"limited"`
	ActiveRuns int  `json:"active_runs"`
	Capacity   *int `json:"capacity"`
}

type ticketPickupDiagnosisBlockedTicketResponse struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	StatusID   string `json:"status_id"`
	StatusName string `json:"status_name"`
}

type ticketPickupDiagnosisCapacityResponse struct {
	Workflow ticketPickupDiagnosisCapacityBucketResponse `json:"workflow"`
	Project  ticketPickupDiagnosisCapacityBucketResponse `json:"project"`
	Provider ticketPickupDiagnosisCapacityBucketResponse `json:"provider"`
	Status   ticketPickupDiagnosisStatusCapacityResponse `json:"status"`
}

type ticketPickupDiagnosisResponse struct {
	State                string                                       `json:"state"`
	PrimaryReasonCode    string                                       `json:"primary_reason_code"`
	PrimaryReasonMessage string                                       `json:"primary_reason_message"`
	NextActionHint       string                                       `json:"next_action_hint,omitempty"`
	Reasons              []ticketPickupDiagnosisReasonResponse        `json:"reasons"`
	Workflow             *ticketPickupDiagnosisWorkflowResponse       `json:"workflow,omitempty"`
	Agent                *ticketPickupDiagnosisAgentResponse          `json:"agent,omitempty"`
	Provider             *ticketPickupDiagnosisProviderResponse       `json:"provider,omitempty"`
	Retry                ticketPickupDiagnosisRetryResponse           `json:"retry"`
	Capacity             ticketPickupDiagnosisCapacityResponse        `json:"capacity"`
	BlockedBy            []ticketPickupDiagnosisBlockedTicketResponse `json:"blocked_by"`
}

func (s *Server) registerTicketRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/tickets", s.handleListTickets)
	api.GET("/projects/:projectId/tickets/archived", s.handleListArchivedTickets)
	api.POST("/projects/:projectId/tickets", s.handleCreateTicket)
	api.GET("/projects/:projectId/tickets/:ticketId/detail", s.handleGetTicketDetail)
	api.GET("/projects/:projectId/tickets/:ticketId/runs", s.handleListTicketRuns)
	api.GET("/projects/:projectId/tickets/:ticketId/runs/:runId", s.handleGetTicketRun)
	api.GET("/tickets/:ticketId", s.handleGetTicket)
	api.PATCH("/tickets/:ticketId", s.handleUpdateTicket)
	api.POST("/tickets/:ticketId/retry/resume", s.handleResumeTicketRetry)
	api.POST("/tickets/:ticketId/workspace/reset", s.handleResetTicketWorkspace)
	api.GET("/tickets/:ticketId/comments", s.handleListTicketComments)
	api.POST("/tickets/:ticketId/comments", s.handleCreateTicketComment)
	api.PATCH("/tickets/:ticketId/comments/:commentId", s.handleUpdateTicketComment)
	api.DELETE("/tickets/:ticketId/comments/:commentId", s.handleDeleteTicketComment)
	api.GET("/tickets/:ticketId/comments/:commentId/revisions", s.handleListTicketCommentRevisions)
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

	parsedPriorities := make([]ticketservice.Priority, 0, len(parseCSVQueryValues(c, "priority")))
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
		Priorities:  make([]ticketservice.Priority, 0, len(parsedPriorities)),
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

func (s *Server) handleListArchivedTickets(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	input, err := ticketservice.ParseArchivedListInput(projectID, ticketservice.ArchivedListRawInput{
		Page:    c.QueryParam("page"),
		PerPage: c.QueryParam("per_page"),
	})
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	result, err := s.ticketService.ListArchived(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, archivedTicketsResponse{
		Tickets: mapTicketResponses(result.Tickets),
		Total:   result.Total,
		Page:    result.Page,
		PerPage: result.PerPage,
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
	raw.CreatedBy = optionalActor(raw.CreatedBy, actorFromHumanPrincipal(c))

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
	if item.Parent != nil {
		if err := s.publishTicketUpdatesByID(c.Request().Context(), item.Parent.ID); err != nil {
			return writeTicketError(c, err)
		}
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
	if s.ticketService == nil || s.catalog.Empty() {
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
	activity := filterNonCommentActivityEvents(activityItems)

	assignedAgent, err := s.loadTicketAssignedAgent(c.Request().Context(), item)
	if err != nil {
		return writeCatalogError(c, err)
	}
	pickupDiagnosis, err := s.ticketService.GetPickupDiagnosis(c.Request().Context(), ticketID)
	if err != nil && !errors.Is(err, ticketservice.ErrUnavailable) {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"assigned_agent":   assignedAgent,
		"pickup_diagnosis": mapPickupDiagnosisResponse(pickupDiagnosis),
		"ticket":           mapTicketResponse(item),
		"repo_scopes":      mapTicketRepoScopeDetailResponses(item.Identifier, repoScopes, indexProjectRepoResponses(projectRepos)),
		"comments":         mapTicketCommentResponses(comments),
		"timeline":         buildTicketTimeline(item, comments, activity),
		"activity":         mapActivityEventResponses(activity),
		"hook_history":     mapActivityEventResponses(filterHookActivityEvents(activity)),
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
	raw.CreatedBy = optionalActor(raw.CreatedBy, actorFromHumanPrincipal(c))

	input, err := parseUpdateTicketRequest(ticketID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	var previous ticketservice.Ticket
	if input.ParentTicketID.Set {
		previous, err = s.ticketService.Get(c.Request().Context(), ticketID)
		if err != nil {
			return writeTicketError(c, err)
		}
	}

	item, err := s.ticketService.Update(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	eventType := ticketMutationEventType(input)
	if err := s.publishTicketEvent(c.Request().Context(), eventType, item); err != nil {
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

func (s *Server) handleResumeTicketRetry(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}

	item, err := s.ticketService.ResumeRetry(c.Request().Context(), ticketservice.ResumeRetryInput{
		TicketID: ticketID,
	})
	if err != nil {
		return writeTicketError(c, err)
	}
	if err := s.publishTicketEvent(c.Request().Context(), ticketRetryResumedType, item); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"ticket": mapTicketResponse(item),
	})
}

func (s *Server) handleResetTicketWorkspace(c echo.Context) error {
	if s.ticketService == nil || s.ticketWorkspaceResetter == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}
	if _, err := s.ticketService.Get(c.Request().Context(), ticketID); err != nil {
		return writeTicketError(c, err)
	}
	if err := s.ticketWorkspaceResetter.ResetTicketWorkspace(c.Request().Context(), ticketID); err != nil {
		var conflictErr workspaceResetConflict
		if errors.As(err, &conflictErr) && conflictErr.WorkspaceResetConflict() {
			return writeAPIError(c, http.StatusConflict, "WORKSPACE_RESET_CONFLICT", err.Error())
		}
		return writeTicketError(c, err)
	}
	if err := s.publishTicketUpdatedByID(c.Request().Context(), ticketID); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, ticketWorkspaceResetResponse{Reset: true})
}

func (s *Server) handleListTicketComments(c echo.Context) error {
	if s.ticketService == nil {
		return writeTicketError(c, ticketservice.ErrUnavailable)
	}

	ticketID, err := parseTicketID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_TICKET_ID", err.Error())
	}

	comments, err := s.ticketService.ListComments(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"comments": mapTicketCommentResponses(comments),
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
	raw.CreatedBy = optionalActor(raw.CreatedBy, actorFromHumanPrincipal(c))

	input, err := parseCreateTicketCommentRequest(ticketID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	comment, err := s.ticketService.AddComment(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	ticket, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
	}
	commentResponse := mapTicketCommentResponse(comment)
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: ticket.ProjectID,
		TicketID:  &ticket.ID,
		EventType: activityevent.TypeTicketCommentCreated,
		Message:   "Added comment to " + ticket.Identifier,
		Metadata:  ticketCommentMetadata(commentResponse),
	}); err != nil {
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
	raw.EditedBy = optionalActor(raw.EditedBy, actorFromHumanPrincipal(c))

	input, err := parseUpdateTicketCommentRequest(ticketID, commentID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	comment, err := s.ticketService.UpdateComment(c.Request().Context(), input)
	if err != nil {
		return writeTicketError(c, err)
	}
	ticket, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
	}
	commentResponse := mapTicketCommentResponse(comment)
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: ticket.ProjectID,
		TicketID:  &ticket.ID,
		EventType: activityevent.TypeTicketCommentEdited,
		Message:   "Edited comment on " + ticket.Identifier,
		Metadata:  ticketCommentMetadata(commentResponse),
	}); err != nil {
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
	ticket, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
	}
	if err := s.emitActivity(c.Request().Context(), activitysvc.RecordInput{
		ProjectID: ticket.ProjectID,
		TicketID:  &ticket.ID,
		EventType: activityevent.TypeTicketCommentDeleted,
		Message:   "Deleted comment on " + ticket.Identifier,
		Metadata: map[string]any{
			"comment_id":     result.DeletedCommentID.String(),
			"changed_fields": []string{"comment"},
		},
	}); err != nil {
		return writeTicketError(c, err)
	}
	if err := s.publishTicketUpdatedByID(c.Request().Context(), ticketID); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

func (s *Server) handleListTicketCommentRevisions(c echo.Context) error {
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

	revisions, err := s.ticketService.ListCommentRevisions(c.Request().Context(), ticketID, commentID)
	if err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"revisions": mapTicketCommentRevisionResponses(revisions),
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

	parsed, err := parseAddDependencyRequest(ticketID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	dependency, err := s.ticketService.AddDependency(c.Request().Context(), parsed.Input)
	if err != nil {
		return writeTicketError(c, err)
	}
	ticketItem, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
	}
	relationship, ok := findTicketDependencyResponse(ticketItem, dependency.ID)
	if !ok {
		return writeAPIError(
			c,
			http.StatusInternalServerError,
			"DEPENDENCY_VIEW_MISSING",
			"created dependency missing from current ticket relationship view",
		)
	}
	if err := s.publishTicketUpdatesByID(c.Request().Context(), ticketID, parsed.Input.TargetTicketID); err != nil {
		return writeTicketError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"dependency": relationship,
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
	current, err := s.ticketService.Get(c.Request().Context(), ticketID)
	if err != nil {
		return writeTicketError(c, err)
	}
	counterpartyID, _ := findRelatedDependencyTicketID(current, dependencyID)

	result, err := s.ticketService.RemoveDependency(c.Request().Context(), ticketID, dependencyID)
	if err != nil {
		return writeTicketError(c, err)
	}
	if err := s.publishTicketUpdatesByID(c.Request().Context(), ticketID, counterpartyID); err != nil {
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
	if err := s.publishTicketUpdatedByID(c.Request().Context(), ticketID); err != nil {
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
	if err := s.publishTicketUpdatedByID(c.Request().Context(), ticketID); err != nil {
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
	case errors.Is(err, ticketservice.ErrProjectRepoNotFound):
		return writeAPIError(c, http.StatusBadRequest, "PROJECT_REPO_NOT_FOUND", err.Error())
	case errors.Is(err, ticketservice.ErrRepoScopeRequired):
		return writeAPIError(c, http.StatusBadRequest, "REPO_SCOPE_REQUIRED", err.Error())
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
	case errors.Is(err, ticketservice.ErrRetryResumeConflict):
		return writeAPIError(c, http.StatusConflict, "RETRY_RESUME_CONFLICT", err.Error())
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func mapTicketRepoScopeDetailResponses(
	ticketIdentifier string,
	items []domain.TicketRepoScope,
	reposByID map[string]projectRepoResponse,
) []ticketRepoScopeDetailResponse {
	response := make([]ticketRepoScopeDetailResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapTicketRepoScopeDetailResponse(ticketIdentifier, item, reposByID[item.RepoID.String()]))
	}

	return response
}

func mapTicketRepoScopeDetailResponse(
	ticketIdentifier string,
	item domain.TicketRepoScope,
	repo projectRepoResponse,
) ticketRepoScopeDetailResponse {
	var repoResponse *projectRepoResponse
	if repo.ID != "" {
		copied := repo
		repoResponse = &copied
	}

	return ticketRepoScopeDetailResponse{
		ID:                  item.ID.String(),
		TicketID:            item.TicketID.String(),
		RepoID:              item.RepoID.String(),
		Repo:                repoResponse,
		BranchName:          item.BranchName,
		DefaultBranch:       repo.DefaultBranch,
		EffectiveBranchName: ticketingdomain.ResolveRepoWorkBranch(ticketIdentifier, item.BranchName),
		BranchSource:        string(ticketingdomain.RepoWorkBranchSourceForOverride(item.BranchName)),
		PullRequestURL:      item.PullRequestURL,
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
	return ticketCommentResponse{
		ID:           item.ID.String(),
		TicketID:     item.TicketID.String(),
		Body:         displayCommentBody(item),
		BodyMarkdown: displayCommentBody(item),
		CreatedBy:    item.CreatedBy,
		CreatedAt:    item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    optionalUpdatedAt(item.CreatedAt, item.UpdatedAt),
		EditedAt:     formatOptionalTime(item.EditedAt),
		EditCount:    item.EditCount,
		LastEditedBy: item.LastEditedBy,
		IsDeleted:    item.IsDeleted,
		DeletedAt:    formatOptionalTime(item.DeletedAt),
		DeletedBy:    item.DeletedBy,
	}
}

func mapTicketCommentRevisionResponses(items []ticketservice.CommentRevision) []ticketCommentRevisionResponse {
	response := make([]ticketCommentRevisionResponse, 0, len(items))
	for _, item := range items {
		response = append(response, mapTicketCommentRevisionResponse(item))
	}

	return response
}

func mapTicketCommentRevisionResponse(item ticketservice.CommentRevision) ticketCommentRevisionResponse {
	return ticketCommentRevisionResponse{
		ID:             item.ID.String(),
		CommentID:      item.CommentID.String(),
		RevisionNumber: item.RevisionNumber,
		BodyMarkdown:   item.BodyMarkdown,
		EditedBy:       item.EditedBy,
		EditedAt:       item.EditedAt.UTC().Format(time.RFC3339),
		EditReason:     item.EditReason,
	}
}

func buildTicketTimeline(
	item ticketservice.Ticket,
	comments []ticketservice.Comment,
	activity []domain.ActivityEvent,
) []ticketTimelineItemResponse {
	timeline := make([]ticketTimelineItemResponse, 0, 1+len(comments)+len(activity))
	timeline = append(timeline, buildTicketDescriptionTimelineItem(item))
	for _, comment := range comments {
		timeline = append(timeline, buildTicketCommentTimelineItem(comment))
	}
	for _, entry := range activity {
		timeline = append(timeline, buildTicketActivityTimelineItem(item.ID, entry))
	}

	if len(timeline) <= 2 {
		return timeline
	}

	head := timeline[0]
	rest := append([]ticketTimelineItemResponse(nil), timeline[1:]...)
	sort.SliceStable(rest, func(left, right int) bool {
		if rest[left].CreatedAt == rest[right].CreatedAt {
			return rest[left].ID < rest[right].ID
		}
		return rest[left].CreatedAt < rest[right].CreatedAt
	})

	return append([]ticketTimelineItemResponse{head}, rest...)
}

func buildTicketDescriptionTimelineItem(item ticketservice.Ticket) ticketTimelineItemResponse {
	actor := parseStoredActor(item.CreatedBy)
	title := item.Title
	bodyMarkdown := item.Description
	metadata := map[string]any{
		"identifier": item.Identifier,
	}

	return ticketTimelineItemResponse{
		ID:            fmt.Sprintf("description:%s", item.ID),
		TicketID:      item.ID.String(),
		ItemType:      "description",
		ActorName:     actor.Name,
		ActorType:     actor.Type,
		Title:         &title,
		BodyMarkdown:  &bodyMarkdown,
		CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
		IsCollapsible: false,
		IsDeleted:     false,
		Metadata:      metadata,
	}
}

func buildTicketCommentTimelineItem(item ticketservice.Comment) ticketTimelineItemResponse {
	actor := parseStoredActor(item.CreatedBy)
	metadata := map[string]any{
		"edit_count":     item.EditCount,
		"revision_count": item.EditCount + 1,
	}
	if item.LastEditedBy != nil {
		metadata["last_edited_by"] = *item.LastEditedBy
	}
	bodyMarkdown := displayCommentBody(item)

	return ticketTimelineItemResponse{
		ID:            fmt.Sprintf("comment:%s", item.ID),
		TicketID:      item.TicketID.String(),
		ItemType:      "comment",
		ActorName:     actor.Name,
		ActorType:     actor.Type,
		BodyMarkdown:  &bodyMarkdown,
		CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     item.UpdatedAt.UTC().Format(time.RFC3339),
		EditedAt:      formatOptionalTime(item.EditedAt),
		IsCollapsible: true,
		IsDeleted:     item.IsDeleted,
		Metadata:      metadata,
	}
}

func buildTicketActivityTimelineItem(ticketID uuid.UUID, item domain.ActivityEvent) ticketTimelineItemResponse {
	actorName, actorType := activityTimelineActor(item)
	title := item.EventType.String()
	bodyText := item.Message
	metadata := cloneMap(item.Metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["event_type"] = item.EventType.String()
	if item.UnknownEventTypeRaw != "" {
		metadata["unknown_event_type_raw"] = item.UnknownEventTypeRaw
	}

	return ticketTimelineItemResponse{
		ID:            fmt.Sprintf("activity:%s", item.ID),
		TicketID:      ticketID.String(),
		ItemType:      "activity",
		ActorName:     actorName,
		ActorType:     actorType,
		Title:         &title,
		BodyText:      &bodyText,
		CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
		IsCollapsible: true,
		IsDeleted:     false,
		Metadata:      metadata,
	}
}

func displayCommentBody(item ticketservice.Comment) string {
	if item.IsDeleted {
		return "Comment deleted"
	}

	return item.BodyMarkdown
}

func optionalUpdatedAt(createdAt time.Time, updatedAt time.Time) *string {
	if updatedAt.IsZero() || updatedAt.Equal(createdAt) {
		return nil
	}

	formatted := updatedAt.UTC().Format(time.RFC3339)
	return &formatted
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil || value.IsZero() {
		return nil
	}

	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

type storedActor struct {
	Name string
	Type string
	ID   string
}

func parseStoredActor(raw string) storedActor {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return storedActor{Name: "api", Type: "system", ID: "api"}
	}

	prefix, suffix, ok := strings.Cut(trimmed, ":")
	if !ok || strings.TrimSpace(suffix) == "" {
		return storedActor{Name: trimmed, Type: "user", ID: trimmed}
	}

	actorType := normalizeTimelineActorType(prefix)

	return storedActor{
		Name: suffix,
		Type: actorType,
		ID:   suffix,
	}
}

func normalizeTimelineActorType(raw string) string {
	switch strings.TrimSpace(raw) {
	case "agent":
		return "agent"
	case "system", "system_proxy":
		return "system"
	case "user":
		return "user"
	default:
		return "user"
	}
}

func activityTimelineActor(item domain.ActivityEvent) (string, string) {
	for _, key := range []string{"actor_name", "agent_name"} {
		if value, ok := item.Metadata[key].(string); ok && strings.TrimSpace(value) != "" {
			if key == "agent_name" || item.AgentID != nil {
				return value, "agent"
			}
			return value, "system"
		}
	}
	if item.AgentID != nil {
		return item.AgentID.String(), "agent"
	}

	return "System", "system"
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
		if item.EventType.IsTicketComment() {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered
}

func isHookActivityEvent(item domain.ActivityEvent) bool {
	switch item.EventType {
	case activityevent.TypeHookStarted, activityevent.TypeHookPassed, activityevent.TypeHookFailed:
		return true
	default:
		return false
	}
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
		Archived:          item.Archived,
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
	response.Dependencies = mapTicketDependencyResponses(item)
	for _, externalLink := range item.ExternalLinks {
		response.ExternalLinks = append(response.ExternalLinks, mapTicketExternalLinkResponse(externalLink))
	}

	return response
}

func mapTicketDependencyResponse(item ticketservice.Dependency) ticketDependencyResponse {
	return mapTicketDependencyResponseWithRelation(item, mapDependencyType(string(item.Type)))
}

func mapTicketDependencyResponseWithRelation(item ticketservice.Dependency, relation string) ticketDependencyResponse {
	return ticketDependencyResponse{
		ID:     item.ID.String(),
		Type:   relation,
		Target: mapTicketReferenceResponse(item.Target),
	}
}

func mapTicketDependencyResponses(item ticketservice.Ticket) []ticketDependencyResponse {
	responses := make([]ticketDependencyResponse, 0, len(item.Dependencies)+len(item.IncomingDependencies))
	for _, dependency := range item.IncomingDependencies {
		responses = append(responses, mapTicketDependencyResponseWithRelation(dependency, "blocked_by"))
	}
	for _, dependency := range item.Dependencies {
		responses = append(responses, mapTicketDependencyResponse(dependency))
	}

	return responses
}

func findTicketDependencyResponse(item ticketservice.Ticket, dependencyID uuid.UUID) (ticketDependencyResponse, bool) {
	for _, dependency := range item.IncomingDependencies {
		if dependency.ID == dependencyID {
			return mapTicketDependencyResponseWithRelation(dependency, "blocked_by"), true
		}
	}
	for _, dependency := range item.Dependencies {
		if dependency.ID == dependencyID {
			return mapTicketDependencyResponse(dependency), true
		}
	}

	return ticketDependencyResponse{}, false
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

func mapPickupDiagnosisResponse(item ticketservice.PickupDiagnosis) ticketPickupDiagnosisResponse {
	response := ticketPickupDiagnosisResponse{
		State:                string(item.State),
		PrimaryReasonCode:    string(item.PrimaryReasonCode),
		PrimaryReasonMessage: item.PrimaryReasonMessage,
		NextActionHint:       item.NextActionHint,
		Reasons:              []ticketPickupDiagnosisReasonResponse{},
		BlockedBy:            []ticketPickupDiagnosisBlockedTicketResponse{},
		Retry: ticketPickupDiagnosisRetryResponse{
			AttemptCount: item.Retry.AttemptCount,
			RetryPaused:  item.Retry.RetryPaused,
			PauseReason:  item.Retry.PauseReason,
		},
		Capacity: ticketPickupDiagnosisCapacityResponse{
			Workflow: ticketPickupDiagnosisCapacityBucketResponse{
				Limited:    item.Capacity.Workflow.Limited,
				ActiveRuns: item.Capacity.Workflow.ActiveRuns,
				Capacity:   item.Capacity.Workflow.Capacity,
			},
			Project: ticketPickupDiagnosisCapacityBucketResponse{
				Limited:    item.Capacity.Project.Limited,
				ActiveRuns: item.Capacity.Project.ActiveRuns,
				Capacity:   item.Capacity.Project.Capacity,
			},
			Provider: ticketPickupDiagnosisCapacityBucketResponse{
				Limited:    item.Capacity.Provider.Limited,
				ActiveRuns: item.Capacity.Provider.ActiveRuns,
				Capacity:   item.Capacity.Provider.Capacity,
			},
			Status: ticketPickupDiagnosisStatusCapacityResponse{
				Limited:    item.Capacity.Status.Limited,
				ActiveRuns: item.Capacity.Status.ActiveRuns,
				Capacity:   item.Capacity.Status.Capacity,
			},
		},
	}

	if item.Retry.NextRetryAt != nil {
		nextRetryAt := item.Retry.NextRetryAt.UTC().Format(time.RFC3339)
		response.Retry.NextRetryAt = &nextRetryAt
	}
	for _, reason := range item.Reasons {
		response.Reasons = append(response.Reasons, ticketPickupDiagnosisReasonResponse{
			Code:     string(reason.Code),
			Message:  reason.Message,
			Severity: string(reason.Severity),
		})
	}
	if item.Workflow != nil {
		response.Workflow = &ticketPickupDiagnosisWorkflowResponse{
			ID:                item.Workflow.ID.String(),
			Name:              item.Workflow.Name,
			IsActive:          item.Workflow.IsActive,
			PickupStatusMatch: item.Workflow.PickupStatusMatch,
		}
	}
	if item.Agent != nil {
		response.Agent = &ticketPickupDiagnosisAgentResponse{
			ID:                  item.Agent.ID.String(),
			Name:                item.Agent.Name,
			RuntimeControlState: item.Agent.RuntimeControlState.String(),
		}
	}
	if item.Provider != nil {
		response.Provider = &ticketPickupDiagnosisProviderResponse{
			ID:                item.Provider.ID.String(),
			Name:              item.Provider.Name,
			MachineID:         item.Provider.MachineID.String(),
			MachineName:       item.Provider.MachineName,
			MachineStatus:     item.Provider.MachineStatus.String(),
			AvailabilityState: item.Provider.AvailabilityState.String(),
		}
		if item.Provider.AvailabilityReason != nil {
			reason := *item.Provider.AvailabilityReason
			response.Provider.AvailabilityReason = &reason
		}
	}
	for _, blocker := range item.BlockedBy {
		response.BlockedBy = append(response.BlockedBy, ticketPickupDiagnosisBlockedTicketResponse{
			ID:         blocker.ID.String(),
			Identifier: blocker.Identifier,
			Title:      blocker.Title,
			StatusID:   blocker.StatusID.String(),
			StatusName: blocker.StatusName,
		})
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
	if s.ticketService == nil || ticketID == uuid.Nil {
		return nil
	}

	item, err := s.ticketService.Get(ctx, ticketID)
	if err != nil {
		return err
	}

	return s.publishTicketEvent(ctx, ticketUpdatedEventType, item)
}

func (s *Server) publishTicketUpdatesByID(ctx context.Context, ticketIDs ...uuid.UUID) error {
	seen := make(map[uuid.UUID]struct{}, len(ticketIDs))
	for _, ticketID := range ticketIDs {
		if ticketID == uuid.Nil {
			continue
		}
		if _, exists := seen[ticketID]; exists {
			continue
		}
		seen[ticketID] = struct{}{}
		if err := s.publishTicketUpdatedByID(ctx, ticketID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) publishParentRelationshipUpdates(
	ctx context.Context,
	previous ticketservice.Ticket,
	current ticketservice.Ticket,
) error {
	oldParentID := optionalTicketReferenceID(previous.Parent)
	newParentID := optionalTicketReferenceID(current.Parent)
	if oldParentID == newParentID {
		return nil
	}

	return s.publishTicketUpdatesByID(ctx, oldParentID, newParentID)
}

func optionalTicketReferenceID(reference *ticketservice.TicketReference) uuid.UUID {
	if reference == nil {
		return uuid.Nil
	}
	return reference.ID
}

func findRelatedDependencyTicketID(item ticketservice.Ticket, dependencyID uuid.UUID) (uuid.UUID, bool) {
	for _, dependency := range item.Dependencies {
		if dependency.ID == dependencyID {
			return dependency.Target.ID, true
		}
	}
	for _, dependency := range item.IncomingDependencies {
		if dependency.ID == dependencyID {
			return dependency.Target.ID, true
		}
	}

	return uuid.Nil, false
}
