package httpapi

import (
	"context"
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
)

type hrAdvisorSummaryResponse struct {
	OpenTickets            int      `json:"open_tickets"`
	CodingTickets          int      `json:"coding_tickets"`
	FailingTickets         int      `json:"failing_tickets"`
	BlockedTickets         int      `json:"blocked_tickets"`
	ActiveAgents           int      `json:"active_agents"`
	WorkflowCount          int      `json:"workflow_count"`
	RecentActivityCount    int      `json:"recent_activity_count"`
	ActiveWorkflowFamilies []string `json:"active_workflow_families"`
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
	RoleSlug                string   `json:"role_slug"`
	RoleName                string   `json:"role_name"`
	WorkflowType            string   `json:"workflow_type"`
	WorkflowFamily          string   `json:"workflow_family"`
	Summary                 string   `json:"summary"`
	HarnessPath             string   `json:"harness_path"`
	Priority                string   `json:"priority"`
	Reason                  string   `json:"reason"`
	Evidence                []string `json:"evidence"`
	SuggestedHeadcount      int      `json:"suggested_headcount"`
	SuggestedWorkflowName   string   `json:"suggested_workflow_name"`
	SuggestedWorkflowType   string   `json:"suggested_workflow_type"`
	SuggestedWorkflowFamily string   `json:"suggested_workflow_family"`
	ActivationReady         bool     `json:"activation_ready"`
	ActiveWorkflowName      *string  `json:"active_workflow_name,omitempty"`
}

type hrAdvisorActivationResponse struct {
	ProjectID       string                                   `json:"project_id"`
	RoleSlug        string                                   `json:"role_slug"`
	Agent           agentResponse                            `json:"agent"`
	Workflow        workflowResponse                         `json:"workflow"`
	BootstrapTicket hrAdvisorBootstrapTicketActivationResult `json:"bootstrap_ticket"`
}

type hrAdvisorBootstrapTicketActivationResult struct {
	Requested bool                             `json:"requested"`
	Status    string                           `json:"status"`
	Message   string                           `json:"message"`
	Ticket    *hrAdvisorActivationTicketResult `json:"ticket,omitempty"`
}

type hrAdvisorActivationTicketResult struct {
	ID         string  `json:"id"`
	Identifier string  `json:"identifier"`
	Title      string  `json:"title"`
	StatusID   string  `json:"status_id"`
	StatusName string  `json:"status_name"`
	WorkflowID *string `json:"workflow_id,omitempty"`
}

func (s *Server) registerHRAdvisorRoutes(api *echo.Group) {
	api.GET("/projects/:projectId/hr-advisor", s.handleGetHRAdvisor)
	api.POST("/projects/:projectId/hr-advisor/activate", s.handleActivateHRRecommendation)
}

func (s *Server) handleGetHRAdvisor(c echo.Context) error {
	if s.catalog.Empty() || s.ticketService == nil || s.workflowService == nil {
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

	workflowLabels := make(map[uuid.UUID]string, len(workflows))
	statusNamesByID := make(map[uuid.UUID]string)
	statusStagesByID := make(map[uuid.UUID]string)
	if s.ticketStatusService != nil {
		statuses, err := s.ticketStatusService.List(ctx, projectID)
		if err != nil {
			return writeTicketStatusError(c, err)
		}
		for _, statusItem := range statuses.Statuses {
			statusNamesByID[statusItem.ID] = statusItem.Name
			statusStagesByID[statusItem.ID] = statusItem.Stage
		}
	}
	activeRoleWorkflows := make(map[string]string)
	workflowDetails := make(map[uuid.UUID]workflowservice.WorkflowDetail, len(workflows))
	for _, workflowItem := range workflows {
		workflowLabels[workflowItem.ID] = workflowItem.Type.String()
		detail, err := s.workflowService.Get(ctx, workflowItem.ID)
		if err != nil {
			return writeWorkflowError(c, err)
		}
		workflowDetails[workflowItem.ID] = detail
		roleSlug := strings.TrimSpace(workflowItem.RoleSlug)
		if roleSlug != "" && workflowItem.IsActive {
			activeRoleWorkflows[roleSlug] = workflowItem.Name
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
		workflowLabel := ""
		if ticketItem.WorkflowID != nil {
			workflowLabel = workflowLabels[*ticketItem.WorkflowID]
		}

		snapshot.Tickets = append(snapshot.Tickets, hrdomain.TicketContext{
			Identifier:        ticketItem.Identifier,
			Type:              string(ticketItem.Type),
			StatusName:        ticketItem.StatusName,
			StatusStage:       statusStagesByID[ticketItem.StatusID],
			WorkflowTypeLabel: workflowLabel,
			HasActiveRun:      ticketItem.CurrentRunID != nil,
			ConsecutiveErrors: ticketItem.ConsecutiveErrors,
			RetryPaused:       ticketItem.RetryPaused,
		})
	}

	for _, workflowItem := range workflows {
		detail := workflowDetails[workflowItem.ID]
		snapshot.Workflows = append(snapshot.Workflows, hrdomain.WorkflowContext{
			Name:           workflowItem.Name,
			TypeLabel:      workflowItem.Type.String(),
			RoleSlug:       strings.TrimSpace(workflowItem.RoleSlug),
			IsActive:       workflowItem.IsActive,
			HarnessPath:    workflowItem.HarnessPath,
			HarnessContent: detail.HarnessContent,
			PickupStatuses: statusBindingsFromIDs(workflowItem.PickupStatusIDs, statusNamesByID, statusStagesByID),
			FinishStatuses: statusBindingsFromIDs(workflowItem.FinishStatusIDs, statusNamesByID, statusStagesByID),
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
		workflowType := recommendation.SuggestedWorkflowTypeLabel
		workflowFamily := recommendation.SuggestedWorkflowFamily
		summary := ""
		harnessPath := ""
		if ok {
			roleName = roleTemplate.Name
			workflowType = roleTemplate.WorkflowType
			summary = roleTemplate.Summary
			harnessPath = roleTemplate.HarnessPath
			workflowFamily = string(workflowservice.ClassifyWorkflow(workflowservice.WorkflowClassificationInput{
				RoleSlug:       roleTemplate.Slug,
				TypeLabel:      workflowservice.MustParseTypeLabel(roleTemplate.WorkflowType),
				WorkflowName:   roleTemplate.Name,
				HarnessPath:    roleTemplate.HarnessPath,
				HarnessContent: roleTemplate.Content,
			}).Family)
		}

		activeWorkflowName, isActive := activeRoleWorkflows[recommendation.RoleSlug]
		var activeWorkflowNamePtr *string
		if isActive {
			activeWorkflowNamePtr = &activeWorkflowName
		}

		recommendations = append(recommendations, hrAdvisorRecommendationResponse{
			RoleSlug:                recommendation.RoleSlug,
			RoleName:                roleName,
			WorkflowType:            workflowType,
			WorkflowFamily:          workflowFamily,
			Summary:                 summary,
			HarnessPath:             harnessPath,
			Priority:                recommendation.Priority,
			Reason:                  recommendation.Reason,
			Evidence:                cloneStringSlice(recommendation.Evidence),
			SuggestedHeadcount:      recommendation.SuggestedHeadcount,
			SuggestedWorkflowName:   recommendation.SuggestedWorkflowName,
			SuggestedWorkflowType:   recommendation.SuggestedWorkflowTypeLabel,
			SuggestedWorkflowFamily: recommendation.SuggestedWorkflowFamily,
			ActivationReady:         !isActive,
			ActiveWorkflowName:      activeWorkflowNamePtr,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"project_id": projectID.String(),
		"summary": hrAdvisorSummaryResponse{
			OpenTickets:            analysis.Summary.OpenTickets,
			CodingTickets:          analysis.Summary.CodingTickets,
			FailingTickets:         analysis.Summary.FailingTickets,
			BlockedTickets:         analysis.Summary.BlockedTickets,
			ActiveAgents:           analysis.Summary.ActiveAgents,
			WorkflowCount:          analysis.Summary.WorkflowCount,
			RecentActivityCount:    analysis.Summary.RecentActivityCount,
			ActiveWorkflowFamilies: cloneStringSlice(analysis.Summary.ActiveWorkflowFamilies),
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
	if s.catalog.Empty() || s.workflowService == nil || s.ticketStatusService == nil {
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
		hrAdvisorWorkflowAdapter{service: s.workflowService},
		hrAdvisorStatusAdapter{service: s.ticketStatusService},
		hrAdvisorTicketAdapter{service: s.ticketService},
	)
	result, err := activationService.Activate(c.Request().Context(), input)
	if err != nil {
		return writeHRAdvisorActivationError(c, err)
	}

	response := hrAdvisorActivationResponse{
		ProjectID: result.ProjectID.String(),
		RoleSlug:  result.RoleSlug,
		Agent:     mapAgentResponse(result.Agent),
		Workflow:  mapHRAdvisorActivationWorkflowResponse(result.Workflow),
		BootstrapTicket: hrAdvisorBootstrapTicketActivationResult{
			Requested: result.BootstrapTicket.Requested,
			Status:    result.BootstrapTicket.Status,
			Message:   result.BootstrapTicket.Message,
		},
	}
	if result.BootstrapTicket.Ticket != nil {
		ticketResponse := mapHRAdvisorActivationTicketResult(*result.BootstrapTicket.Ticket)
		response.BootstrapTicket.Ticket = &ticketResponse
	}

	return c.JSON(http.StatusCreated, response)
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
		errors.Is(err, ticketstatus.ErrStatusNotFound),
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

type hrAdvisorWorkflowAdapter struct {
	service *workflowservice.Service
}

func (a hrAdvisorWorkflowAdapter) List(ctx context.Context, projectID uuid.UUID) ([]hrservice.ActivationWorkflow, error) {
	items, err := a.service.List(ctx, projectID)
	if err != nil {
		return nil, err
	}

	response := make([]hrservice.ActivationWorkflow, 0, len(items))
	for _, item := range items {
		response = append(response, hrservice.ActivationWorkflow{
			ID:                    item.ID,
			ProjectID:             item.ProjectID,
			AgentID:               item.AgentID,
			Name:                  item.Name,
			Type:                  item.Type.String(),
			RoleSlug:              item.RoleSlug,
			RoleName:              item.RoleName,
			RoleDescription:       item.RoleDescription,
			PlatformAccessAllowed: append([]string(nil), item.PlatformAccessAllowed...),
			HarnessPath:           item.HarnessPath,
			MaxConcurrent:         item.MaxConcurrent,
			MaxRetryAttempts:      item.MaxRetryAttempts,
			TimeoutMinutes:        item.TimeoutMinutes,
			StallTimeoutMinutes:   item.StallTimeoutMinutes,
			Version:               item.Version,
			IsActive:              item.IsActive,
			PickupStatusIDs:       append([]uuid.UUID(nil), item.PickupStatusIDs...),
			FinishStatusIDs:       append([]uuid.UUID(nil), item.FinishStatusIDs...),
		})
	}

	return response, nil
}

func (a hrAdvisorWorkflowAdapter) Create(
	ctx context.Context,
	input hrservice.ActivateWorkflowInput,
) (hrservice.ActivationWorkflow, error) {
	pickupStatusIDs, err := workflowservice.ParseStatusBindingSet("pickup_status_ids", input.PickupStatusIDs)
	if err != nil {
		return hrservice.ActivationWorkflow{}, err
	}
	finishStatusIDs, err := workflowservice.ParseStatusBindingSet("finish_status_ids", input.FinishStatusIDs)
	if err != nil {
		return hrservice.ActivationWorkflow{}, err
	}

	harnessPath := input.HarnessPath
	workflowType, err := parseWorkflowTypeLabel(input.Type)
	if err != nil {
		return hrservice.ActivationWorkflow{}, err
	}
	item, err := a.service.Create(ctx, workflowservice.CreateInput{
		ProjectID:             input.ProjectID,
		AgentID:               input.AgentID,
		Name:                  input.Name,
		Type:                  workflowType,
		RoleSlug:              input.RoleSlug,
		RoleName:              input.RoleName,
		RoleDescription:       input.RoleDescription,
		PlatformAccessAllowed: append([]string(nil), input.PlatformAccessAllowed...),
		SkillNames:            append([]string(nil), input.SkillNames...),
		HarnessPath:           &harnessPath,
		HarnessContent:        input.HarnessContent,
		MaxConcurrent:         input.MaxConcurrent,
		MaxRetryAttempts:      input.MaxRetryAttempts,
		TimeoutMinutes:        input.TimeoutMinutes,
		StallTimeoutMinutes:   input.StallTimeoutMinutes,
		IsActive:              input.IsActive,
		PickupStatusIDs:       pickupStatusIDs,
		FinishStatusIDs:       finishStatusIDs,
	})
	if err != nil {
		return hrservice.ActivationWorkflow{}, err
	}

	return hrservice.ActivationWorkflow{
		ID:                    item.ID,
		ProjectID:             item.ProjectID,
		AgentID:               item.AgentID,
		Name:                  item.Name,
		Type:                  item.Type.String(),
		RoleSlug:              item.RoleSlug,
		RoleName:              item.RoleName,
		RoleDescription:       item.RoleDescription,
		PlatformAccessAllowed: append([]string(nil), item.PlatformAccessAllowed...),
		HarnessPath:           item.HarnessPath,
		HarnessContent:        item.HarnessContent,
		MaxConcurrent:         item.MaxConcurrent,
		MaxRetryAttempts:      item.MaxRetryAttempts,
		TimeoutMinutes:        item.TimeoutMinutes,
		StallTimeoutMinutes:   item.StallTimeoutMinutes,
		Version:               item.Version,
		IsActive:              item.IsActive,
		PickupStatusIDs:       append([]uuid.UUID(nil), item.PickupStatusIDs...),
		FinishStatusIDs:       append([]uuid.UUID(nil), item.FinishStatusIDs...),
	}, nil
}

type hrAdvisorStatusAdapter struct {
	service *ticketstatus.Service
}

func (a hrAdvisorStatusAdapter) List(ctx context.Context, projectID uuid.UUID) ([]hrservice.ActivationStatus, error) {
	result, err := a.service.List(ctx, projectID)
	if err != nil {
		return nil, err
	}

	statuses := make([]hrservice.ActivationStatus, 0, len(result.Statuses))
	for _, item := range result.Statuses {
		statuses = append(statuses, hrservice.ActivationStatus{
			ID:    item.ID,
			Name:  item.Name,
			Stage: item.Stage,
		})
	}

	return statuses, nil
}

type hrAdvisorTicketAdapter struct {
	service *ticketservice.Service
}

func (a hrAdvisorTicketAdapter) Create(
	ctx context.Context,
	input hrservice.CreateActivationTicketInput,
) (hrservice.ActivationTicket, error) {
	priority, err := parseTicketPriority(input.Priority)
	if err != nil {
		return hrservice.ActivationTicket{}, err
	}
	ticketType, err := parseTicketType(input.Type)
	if err != nil {
		return hrservice.ActivationTicket{}, err
	}

	item, err := a.service.Create(ctx, ticketservice.CreateInput{
		ProjectID:   input.ProjectID,
		Title:       input.Title,
		Description: input.Description,
		StatusID:    input.StatusID,
		Priority:    &priority,
		Type:        ticketType,
		WorkflowID:  input.WorkflowID,
		CreatedBy:   input.CreatedBy,
	})
	if err != nil {
		return hrservice.ActivationTicket{}, err
	}

	return hrservice.ActivationTicket{
		ID:          item.ID,
		ProjectID:   item.ProjectID,
		Identifier:  item.Identifier,
		Title:       item.Title,
		StatusID:    item.StatusID,
		StatusName:  item.StatusName,
		WorkflowID:  item.WorkflowID,
		CreatedBy:   item.CreatedBy,
		Priority:    item.Priority.String(),
		Type:        item.Type.String(),
		Description: item.Description,
	}, nil
}

func mapHRAdvisorActivationTicketResult(item hrservice.ActivationTicket) hrAdvisorActivationTicketResult {
	var workflowID *string
	if item.WorkflowID != nil {
		value := item.WorkflowID.String()
		workflowID = &value
	}

	return hrAdvisorActivationTicketResult{
		ID:         item.ID.String(),
		Identifier: item.Identifier,
		Title:      item.Title,
		StatusID:   item.StatusID.String(),
		StatusName: item.StatusName,
		WorkflowID: workflowID,
	}
}

func mapHRAdvisorActivationWorkflowResponse(item hrservice.ActivationWorkflow) workflowResponse {
	var agentID *string
	if item.AgentID != nil {
		value := item.AgentID.String()
		agentID = &value
	}

	harnessContent := item.HarnessContent
	typeLabel, err := workflowservice.ParseTypeLabel(item.Type)
	if err != nil {
		typeLabel = workflowservice.MustParseTypeLabel("unknown")
	}
	classification := workflowservice.ClassifyWorkflow(workflowservice.WorkflowClassificationInput{
		TypeLabel:      typeLabel,
		WorkflowName:   item.Name,
		HarnessPath:    item.HarnessPath,
		HarnessContent: item.HarnessContent,
	})
	return workflowResponse{
		ID:                    item.ID.String(),
		ProjectID:             item.ProjectID.String(),
		AgentID:               agentID,
		Name:                  item.Name,
		Type:                  item.Type,
		RoleSlug:              item.RoleSlug,
		RoleName:              item.RoleName,
		RoleDescription:       item.RoleDescription,
		PlatformAccessAllowed: append([]string(nil), item.PlatformAccessAllowed...),
		WorkflowFamily:        string(classification.Family),
		Classification:        mapClassificationResponse(classification),
		HarnessPath:           item.HarnessPath,
		HarnessContent:        &harnessContent,
		Hooks:                 map[string]any{},
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

func statusBindingsFromIDs(
	statusIDs []uuid.UUID,
	statusNamesByID map[uuid.UUID]string,
	statusStagesByID map[uuid.UUID]string,
) []hrdomain.StatusBindingContext {
	if len(statusIDs) == 0 || len(statusNamesByID) == 0 {
		return nil
	}

	bindings := make([]hrdomain.StatusBindingContext, 0, len(statusIDs))
	for _, statusID := range statusIDs {
		name := strings.TrimSpace(statusNamesByID[statusID])
		if name == "" {
			continue
		}
		bindings = append(bindings, hrdomain.StatusBindingContext{
			Name:  name,
			Stage: strings.TrimSpace(statusStagesByID[statusID]),
		})
	}
	return bindings
}
