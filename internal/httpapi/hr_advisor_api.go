package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/builtin"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	hrdomain "github.com/BetterAndBetterII/openase/internal/domain/hradvisor"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	hrservice "github.com/BetterAndBetterII/openase/internal/service/hradvisor"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.yaml.in/yaml/v3"
)

type hrAdvisorSummaryResponse struct {
	OpenTickets         int      `json:"open_tickets"`
	CodingTickets       int      `json:"coding_tickets"`
	FailingTickets      int      `json:"failing_tickets"`
	BlockedTickets      int      `json:"blocked_tickets"`
	ActiveAgents        int      `json:"active_agents"`
	WorkflowCount       int      `json:"workflow_count"`
	RecentActivityCount int      `json:"recent_activity_count"`
	ActiveWorkflowTypes []string `json:"active_workflow_types"`
}

type hrAdvisorStaffingResponse struct {
	Developers int `json:"developers"`
	QA         int `json:"qa"`
	Docs       int `json:"docs"`
	Security   int `json:"security"`
	Product    int `json:"product"`
	Research   int `json:"research"`
}

type hrAdvisorRecommendationResponse struct {
	RoleSlug              string   `json:"role_slug"`
	RoleName              string   `json:"role_name"`
	WorkflowType          string   `json:"workflow_type"`
	Summary               string   `json:"summary"`
	HarnessPath           string   `json:"harness_path"`
	Priority              string   `json:"priority"`
	Reason                string   `json:"reason"`
	Evidence              []string `json:"evidence"`
	SuggestedHeadcount    int      `json:"suggested_headcount"`
	SuggestedWorkflowName string   `json:"suggested_workflow_name"`
	ActivationReady       bool     `json:"activation_ready"`
	ActiveWorkflowName    *string  `json:"active_workflow_name,omitempty"`
}

type hrAdvisorActivationResponse struct {
	ProjectID       string                                   `json:"project_id"`
	RoleSlug        string                                   `json:"role_slug"`
	Agent           agentResponse                            `json:"agent"`
	Workflow        workflowResponse                         `json:"workflow"`
	BootstrapTicket hrAdvisorBootstrapTicketActivationResult `json:"bootstrap_ticket"`
}

type hrAdvisorBootstrapTicketActivationResult struct {
	Requested bool            `json:"requested"`
	Status    string          `json:"status"`
	Message   string          `json:"message"`
	Ticket    *ticketResponse `json:"ticket,omitempty"`
}

func (s *Server) registerHRAdvisorRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/hr-advisor", s.handleGetHRAdvisor)
	api.POST("/projects/:projectId/hr-advisor/activate", s.handleActivateHRRecommendation)
}

func (s *Server) handleGetHRAdvisor(c echo.Context) error {
	if s.catalog == nil || s.ticketService == nil || s.workflowService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "hr advisor is unavailable")
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	ctx := c.Request().Context()
	project, err := s.catalog.GetProject(ctx, projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	tickets, err := s.ticketService.List(ctx, ticketservice.ListInput{ProjectID: projectID})
	if err != nil {
		return writeTicketError(c, err)
	}

	workflows, err := s.workflowService.List(ctx, projectID)
	if err != nil {
		return writeWorkflowError(c, err)
	}

	agents, err := s.catalog.ListAgents(ctx, projectID)
	if err != nil {
		return writeCatalogError(c, err)
	}

	activityInput, err := catalogdomain.ParseListActivityEvents(projectID, catalogdomain.ActivityEventListInput{Limit: "40"})
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
	activityItems, err := s.catalog.ListActivityEvents(ctx, activityInput)
	if err != nil {
		return writeCatalogError(c, err)
	}

	workflowTypes := make(map[uuid.UUID]string, len(workflows))
	statusNamesByID := make(map[uuid.UUID]string)
	if s.ticketStatusService != nil {
		statuses, err := s.ticketStatusService.List(ctx, projectID)
		if err != nil {
			return writeTicketStatusError(c, err)
		}
		for _, statusItem := range statuses.Statuses {
			statusNamesByID[statusItem.ID] = statusItem.Name
		}
	}
	activeRoleWorkflows := make(map[string]string)
	workflowRoleSlugs := make(map[uuid.UUID]string)
	for _, workflowItem := range workflows {
		workflowTypes[workflowItem.ID] = string(workflowItem.Type)
		if !workflowItem.IsActive {
			continue
		}

		detail, err := s.workflowService.Get(ctx, workflowItem.ID)
		if err != nil {
			return writeWorkflowError(c, err)
		}
		if roleSlug := parseHarnessRoleSlug(detail.HarnessContent); roleSlug != "" {
			activeRoleWorkflows[roleSlug] = workflowItem.Name
			workflowRoleSlugs[workflowItem.ID] = roleSlug
		}
	}

	snapshot := hrdomain.Snapshot{
		Project: hrdomain.ProjectContext{
			Name:                project.Name,
			Description:         project.Description,
			Status:              string(project.Status),
			MaxConcurrentAgents: project.MaxConcurrentAgents,
		},
		Tickets:             make([]hrdomain.TicketContext, 0, len(tickets)),
		Workflows:           make([]hrdomain.WorkflowContext, 0, len(workflows)),
		Agents:              make([]hrdomain.AgentContext, 0, len(agents)),
		RecentActivityCount: len(activityItems),
		RecentTrends:        parseHRActivityTrends(activityItems),
		ActiveRoleSlugs:     make([]string, 0, len(activeRoleWorkflows)),
	}

	for roleSlug := range activeRoleWorkflows {
		snapshot.ActiveRoleSlugs = append(snapshot.ActiveRoleSlugs, roleSlug)
	}

	for _, ticketItem := range tickets {
		workflowType := ""
		if ticketItem.WorkflowID != nil {
			workflowType = workflowTypes[*ticketItem.WorkflowID]
		}

		snapshot.Tickets = append(snapshot.Tickets, hrdomain.TicketContext{
			Identifier:        ticketItem.Identifier,
			Type:              string(ticketItem.Type),
			StatusName:        ticketItem.StatusName,
			WorkflowType:      workflowType,
			HasActiveRun:      ticketItem.CurrentRunID != nil,
			ConsecutiveErrors: ticketItem.ConsecutiveErrors,
			RetryPaused:       ticketItem.RetryPaused,
		})
	}

	for _, workflowItem := range workflows {
		snapshot.Workflows = append(snapshot.Workflows, hrdomain.WorkflowContext{
			Name:              workflowItem.Name,
			Type:              string(workflowItem.Type),
			RoleSlug:          workflowRoleSlugs[workflowItem.ID],
			IsActive:          workflowItem.IsActive,
			PickupStatusNames: statusNamesFromIDs(workflowItem.PickupStatusIDs, statusNamesByID),
			FinishStatusNames: statusNamesFromIDs(workflowItem.FinishStatusIDs, statusNamesByID),
		})
	}

	for _, agentItem := range agents {
		status := string(catalogdomain.DefaultAgentStatus)
		if agentItem.Runtime != nil {
			status = string(agentItem.Runtime.Status)
		}
		snapshot.Agents = append(snapshot.Agents, hrdomain.AgentContext{
			Status: status,
		})
	}

	analysis := hrservice.Analyze(snapshot)
	recommendations := make([]hrAdvisorRecommendationResponse, 0, len(analysis.Recommendations))
	for _, recommendation := range analysis.Recommendations {
		roleTemplate, ok := builtin.RoleBySlug(recommendation.RoleSlug)
		roleName := recommendation.RoleSlug
		workflowType := "custom"
		summary := ""
		harnessPath := ""
		if ok {
			roleName = roleTemplate.Name
			workflowType = roleTemplate.WorkflowType
			summary = roleTemplate.Summary
			harnessPath = roleTemplate.HarnessPath
		}

		activeWorkflowName, isActive := activeRoleWorkflows[recommendation.RoleSlug]
		var activeWorkflowNamePtr *string
		if isActive {
			activeWorkflowNamePtr = &activeWorkflowName
		}

		recommendations = append(recommendations, hrAdvisorRecommendationResponse{
			RoleSlug:              recommendation.RoleSlug,
			RoleName:              roleName,
			WorkflowType:          workflowType,
			Summary:               summary,
			HarnessPath:           harnessPath,
			Priority:              recommendation.Priority,
			Reason:                recommendation.Reason,
			Evidence:              cloneStringSlice(recommendation.Evidence),
			SuggestedHeadcount:    recommendation.SuggestedHeadcount,
			SuggestedWorkflowName: recommendation.SuggestedWorkflowName,
			ActivationReady:       !isActive,
			ActiveWorkflowName:    activeWorkflowNamePtr,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"project_id": projectID.String(),
		"summary": hrAdvisorSummaryResponse{
			OpenTickets:         analysis.Summary.OpenTickets,
			CodingTickets:       analysis.Summary.CodingTickets,
			FailingTickets:      analysis.Summary.FailingTickets,
			BlockedTickets:      analysis.Summary.BlockedTickets,
			ActiveAgents:        analysis.Summary.ActiveAgents,
			WorkflowCount:       analysis.Summary.WorkflowCount,
			RecentActivityCount: analysis.Summary.RecentActivityCount,
			ActiveWorkflowTypes: cloneStringSlice(analysis.Summary.ActiveWorkflowTypes),
		},
		"staffing": hrAdvisorStaffingResponse{
			Developers: analysis.Staffing.Developers,
			QA:         analysis.Staffing.QA,
			Docs:       analysis.Staffing.Docs,
			Security:   analysis.Staffing.Security,
			Product:    analysis.Staffing.Product,
			Research:   analysis.Staffing.Research,
		},
		"recommendations": recommendations,
	})
}

func (s *Server) handleActivateHRRecommendation(c echo.Context) error {
	if s.catalog == nil || s.workflowService == nil || s.ticketStatusService == nil {
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "hr advisor activation is unavailable")
	}

	projectID, err := parseProjectID(c)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_PROJECT_ID", err.Error())
	}

	var raw hrdomain.ActivateRecommendationRequest
	if err := decodeJSON(c, &raw); err != nil {
		return err
	}

	input, err := hrdomain.ParseActivateRecommendation(projectID, raw)
	if err != nil {
		return writeAPIError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}

	activationService := hrservice.NewActivationService(
		s.catalog,
		s.workflowService,
		s.ticketStatusService,
		s.ticketService,
	)
	result, err := activationService.Activate(c.Request().Context(), input)
	if err != nil {
		return writeHRAdvisorActivationError(c, err)
	}

	response := hrAdvisorActivationResponse{
		ProjectID: result.ProjectID.String(),
		RoleSlug:  result.RoleSlug,
		Agent:     mapAgentResponse(result.Agent),
		Workflow:  mapWorkflowDetailResponse(result.Workflow),
		BootstrapTicket: hrAdvisorBootstrapTicketActivationResult{
			Requested: result.BootstrapTicket.Requested,
			Status:    result.BootstrapTicket.Status,
			Message:   result.BootstrapTicket.Message,
		},
	}
	if result.BootstrapTicket.Ticket != nil {
		ticketResponse := mapTicketResponse(*result.BootstrapTicket.Ticket)
		response.BootstrapTicket.Ticket = &ticketResponse
	}

	return c.JSON(http.StatusCreated, response)
}

func parseHarnessRoleSlug(content string) string {
	frontmatter, err := extractHarnessFrontmatter(content)
	if err != nil {
		return ""
	}

	var document struct {
		Workflow struct {
			Role string `yaml:"role"`
		} `yaml:"workflow"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return ""
	}

	return strings.TrimSpace(document.Workflow.Role)
}

func extractHarnessFrontmatter(content string) (string, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", workflowservice.ErrHarnessInvalid
	}

	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) != "---" {
			continue
		}
		frontmatter := strings.Join(lines[1:index], "\n")
		if strings.TrimSpace(frontmatter) == "" {
			return "", workflowservice.ErrHarnessInvalid
		}
		return frontmatter, nil
	}

	return "", workflowservice.ErrHarnessInvalid
}

func writeHRAdvisorActivationError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, hrservice.ErrActivationUnavailable):
		return writeAPIError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	case errors.Is(err, hrservice.ErrActivationRoleNotFound):
		return writeAPIError(c, http.StatusNotFound, "ROLE_TEMPLATE_NOT_FOUND", err.Error())
	case errors.Is(err, hrservice.ErrActivationWorkflowExists):
		return writeAPIError(c, http.StatusConflict, "HR_ROLE_ALREADY_ACTIVE", err.Error())
	case errors.Is(err, hrservice.ErrActivationProviderUnavailable):
		return writeAPIError(c, http.StatusConflict, "AGENT_PROVIDER_UNAVAILABLE", err.Error())
	case errors.Is(err, hrservice.ErrActivationStatusNotFound):
		return writeAPIError(c, http.StatusConflict, "HR_STATUS_NOT_CONFIGURED", err.Error())
	case errors.Is(err, catalogservice.ErrInvalidInput),
		errors.Is(err, catalogservice.ErrNotFound),
		errors.Is(err, catalogservice.ErrConflict),
		errors.Is(err, catalogservice.ErrMachineProbeFailed),
		errors.Is(err, catalogservice.ErrMachineTestingUnavailable):
		return writeCatalogError(c, err)
	case errors.Is(err, workflowservice.ErrUnavailable),
		errors.Is(err, workflowservice.ErrProjectNotFound),
		errors.Is(err, workflowservice.ErrPrimaryRepoUnavailable),
		errors.Is(err, workflowservice.ErrWorkflowNotFound),
		errors.Is(err, workflowservice.ErrStatusNotFound),
		errors.Is(err, workflowservice.ErrAgentNotFound),
		errors.Is(err, workflowservice.ErrWorkflowConflict),
		errors.Is(err, workflowservice.ErrHarnessInvalid),
		errors.Is(err, workflowservice.ErrHookConfigInvalid),
		errors.Is(err, workflowservice.ErrWorkflowHookBlocked):
		return writeWorkflowError(c, err)
	case errors.Is(err, ticketstatus.ErrUnavailable),
		errors.Is(err, ticketstatus.ErrProjectNotFound),
		errors.Is(err, ticketstatus.ErrStageNotFound),
		errors.Is(err, ticketstatus.ErrStatusNotFound),
		errors.Is(err, ticketstatus.ErrDuplicateStageKey),
		errors.Is(err, ticketstatus.ErrDuplicateStatusName),
		errors.Is(err, ticketstatus.ErrCannotDeleteLastStatus),
		errors.Is(err, ticketstatus.ErrDefaultStatusRequired):
		return writeTicketStatusError(c, err)
	case errors.Is(err, ticketservice.ErrUnavailable),
		errors.Is(err, ticketservice.ErrProjectNotFound),
		errors.Is(err, ticketservice.ErrStatusNotFound):
		return writeTicketError(c, err)
	default:
		return writeAPIError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func statusNamesFromIDs(statusIDs []uuid.UUID, statusNamesByID map[uuid.UUID]string) []string {
	if len(statusIDs) == 0 || len(statusNamesByID) == 0 {
		return nil
	}

	names := make([]string, 0, len(statusIDs))
	for _, statusID := range statusIDs {
		if name := strings.TrimSpace(statusNamesByID[statusID]); name != "" {
			names = append(names, name)
		}
	}
	return names
}
