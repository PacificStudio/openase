package hradvisor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/builtin"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	hrdomain "github.com/BetterAndBetterII/openase/internal/domain/hradvisor"
	"github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

var (
	ErrActivationUnavailable         = errors.New("hr advisor activation unavailable")
	ErrActivationRoleNotFound        = errors.New("hr advisor role template not found")
	ErrActivationWorkflowExists      = errors.New("hr advisor role workflow already exists")
	ErrActivationProviderUnavailable = errors.New("hr advisor activation requires an available agent provider")
	ErrActivationStatusNotFound      = errors.New("hr advisor activation status is not configured")
)

type activationCatalog interface {
	GetProject(ctx context.Context, id uuid.UUID) (catalogdomain.Project, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (catalogdomain.Organization, error)
	ListAgentProviders(ctx context.Context, organizationID uuid.UUID) ([]catalogdomain.AgentProvider, error)
	CreateAgent(ctx context.Context, input catalogdomain.CreateAgent) (catalogdomain.Agent, error)
	DeleteAgent(ctx context.Context, id uuid.UUID) (catalogdomain.Agent, error)
}

type activationWorkflows interface {
	List(ctx context.Context, projectID uuid.UUID) ([]workflow.Workflow, error)
	Create(ctx context.Context, input workflow.CreateInput) (workflow.WorkflowDetail, error)
}

type activationStatuses interface {
	List(ctx context.Context, projectID uuid.UUID) (ticketstatus.ListResult, error)
}

type activationTickets interface {
	Create(ctx context.Context, input ticket.CreateInput) (ticket.Ticket, error)
}

type ActivationService struct {
	catalog   activationCatalog
	workflows activationWorkflows
	statuses  activationStatuses
	tickets   activationTickets
}

type ActivationResult struct {
	ProjectID       uuid.UUID
	RoleSlug        string
	Agent           catalogdomain.Agent
	Workflow        workflow.WorkflowDetail
	BootstrapTicket BootstrapTicketResult
}

type BootstrapTicketResult struct {
	Requested bool
	Status    string
	Message   string
	Ticket    *ticket.Ticket
}

func NewActivationService(
	catalog activationCatalog,
	workflows activationWorkflows,
	statuses activationStatuses,
	tickets activationTickets,
) *ActivationService {
	return &ActivationService{
		catalog:   catalog,
		workflows: workflows,
		statuses:  statuses,
		tickets:   tickets,
	}
}

func (s *ActivationService) Activate(
	ctx context.Context,
	input hrdomain.ActivateRecommendationInput,
) (ActivationResult, error) {
	if s.catalog == nil || s.workflows == nil || s.statuses == nil {
		return ActivationResult{}, ErrActivationUnavailable
	}

	project, err := s.catalog.GetProject(ctx, input.ProjectID)
	if err != nil {
		return ActivationResult{}, err
	}

	roleTemplate, ok := builtin.RoleBySlug(input.RoleSlug)
	if !ok {
		return ActivationResult{}, fmt.Errorf("%w: %s", ErrActivationRoleNotFound, input.RoleSlug)
	}

	template, err := hrdomain.ParseActivationTemplate(
		roleTemplate.Slug,
		roleTemplate.HarnessPath,
		roleTemplate.Content,
		roleTemplate.Summary,
	)
	if err != nil {
		return ActivationResult{}, err
	}

	existingWorkflows, err := s.workflows.List(ctx, input.ProjectID)
	if err != nil {
		return ActivationResult{}, err
	}
	if activationWorkflowExists(existingWorkflows, template.HarnessPath) {
		return ActivationResult{}, fmt.Errorf("%w: %s", ErrActivationWorkflowExists, template.HarnessPath)
	}

	statusList, err := s.statuses.List(ctx, input.ProjectID)
	if err != nil {
		return ActivationResult{}, err
	}
	pickupStatusID, finishStatusID, err := resolveActivationStatusIDs(statusList, template)
	if err != nil {
		return ActivationResult{}, err
	}

	org, err := s.catalog.GetOrganization(ctx, project.OrganizationID)
	if err != nil {
		return ActivationResult{}, err
	}
	providers, err := s.catalog.ListAgentProviders(ctx, project.OrganizationID)
	if err != nil {
		return ActivationResult{}, err
	}
	provider, err := selectActivationProvider(project, org, providers)
	if err != nil {
		return ActivationResult{}, err
	}

	createdAgent, err := s.catalog.CreateAgent(ctx, catalogdomain.CreateAgent{
		ProjectID:           input.ProjectID,
		ProviderID:          provider.ID,
		Name:                activationAgentName(template),
		RuntimeControlState: catalogdomain.DefaultAgentRuntimeControlState,
	})
	if err != nil {
		return ActivationResult{}, err
	}

	createdWorkflow, err := s.workflows.Create(ctx, workflow.CreateInput{
		ProjectID:           input.ProjectID,
		AgentID:             createdAgent.ID,
		Name:                template.WorkflowName,
		Type:                normalizeActivationWorkflowType(template.WorkflowType),
		HarnessPath:         stringPointer(template.HarnessPath),
		HarnessContent:      template.HarnessContent,
		MaxConcurrent:       1,
		MaxRetryAttempts:    1,
		TimeoutMinutes:      30,
		StallTimeoutMinutes: 5,
		IsActive:            true,
		PickupStatusIDs:     mustParseStatusBindingSet(pickupStatusID),
		FinishStatusIDs:     mustParseStatusBindingSet(finishStatusID),
	})
	if err != nil {
		if _, rollbackErr := s.catalog.DeleteAgent(ctx, createdAgent.ID); rollbackErr != nil {
			return ActivationResult{}, fmt.Errorf("%w (agent rollback failed: %v)", err, rollbackErr)
		}
		return ActivationResult{}, err
	}

	result := ActivationResult{
		ProjectID: input.ProjectID,
		RoleSlug:  input.RoleSlug,
		Agent:     createdAgent,
		Workflow:  createdWorkflow,
		BootstrapTicket: BootstrapTicketResult{
			Requested: input.CreateBootstrapTicket,
			Status:    "not_requested",
			Message:   "bootstrap ticket not requested",
		},
	}

	if !input.CreateBootstrapTicket {
		return result, nil
	}

	if s.tickets == nil {
		result.BootstrapTicket.Status = "failed"
		result.BootstrapTicket.Message = "ticket service unavailable for bootstrap ticket creation"
		return result, nil
	}

	draft := activationBootstrapTicketDraft(template)
	createdTicket, err := s.tickets.Create(ctx, ticket.CreateInput{
		ProjectID:   input.ProjectID,
		Title:       draft.Title,
		Description: draft.Description,
		StatusID:    &pickupStatusID,
		Priority:    draft.Priority,
		Type:        draft.Type,
		WorkflowID:  &createdWorkflow.ID,
		CreatedBy:   "system:hr-advisor",
	})
	if err != nil {
		result.BootstrapTicket.Status = "failed"
		result.BootstrapTicket.Message = err.Error()
		return result, nil
	}

	result.BootstrapTicket.Status = "created"
	result.BootstrapTicket.Message = "bootstrap ticket created"
	result.BootstrapTicket.Ticket = &createdTicket
	return result, nil
}

type activationBootstrapDraft struct {
	Title       string
	Description string
	Priority    entticket.Priority
	Type        entticket.Type
}

func activationBootstrapTicketDraft(template hrdomain.ActivationTemplate) activationBootstrapDraft {
	title := fmt.Sprintf("Bootstrap %s workflow", template.WorkflowName)
	description := fmt.Sprintf(
		"The HR Advisor activated the %s role.\n\nStart the new workflow with the smallest concrete task that proves the role is wired correctly.\n\nRole summary: %s",
		template.WorkflowName,
		strings.TrimSpace(template.Summary),
	)

	switch template.RoleSlug {
	case "qa-engineer":
		title = "Bootstrap QA regression coverage"
		description = "The HR Advisor activated the QA Engineer role.\n\nReview the current implementation backlog and create or update the highest-value automated regression coverage first."
	case "technical-writer":
		title = "Bootstrap documentation update workflow"
		description = "The HR Advisor activated the Technical Writer role.\n\nIdentify the most outdated shipped behavior and update the user-facing or engineering documentation for it."
	}

	return activationBootstrapDraft{
		Title:       title,
		Description: description,
		Priority:    entticket.PriorityMedium,
		Type:        entticket.TypeChore,
	}
}

func activationWorkflowExists(items []workflow.Workflow, harnessPath string) bool {
	normalizedPath := strings.TrimSpace(harnessPath)
	for _, item := range items {
		if strings.TrimSpace(item.HarnessPath) == normalizedPath {
			return true
		}
	}

	return false
}

func resolveActivationStatusIDs(
	statusList ticketstatus.ListResult,
	template hrdomain.ActivationTemplate,
) (uuid.UUID, uuid.UUID, error) {
	statusesByName := make(map[string]uuid.UUID, len(statusList.Statuses))
	for _, statusItem := range statusList.Statuses {
		statusesByName[strings.TrimSpace(statusItem.Name)] = statusItem.ID
	}

	pickupStatusID, ok := statusesByName[template.PickupStatusName]
	if !ok {
		return uuid.Nil, uuid.Nil, fmt.Errorf("%w: pickup status %q", ErrActivationStatusNotFound, template.PickupStatusName)
	}
	finishStatusID, ok := statusesByName[template.FinishStatusName]
	if !ok {
		return uuid.Nil, uuid.Nil, fmt.Errorf("%w: finish status %q", ErrActivationStatusNotFound, template.FinishStatusName)
	}

	return pickupStatusID, finishStatusID, nil
}

func selectActivationProvider(
	project catalogdomain.Project,
	org catalogdomain.Organization,
	providers []catalogdomain.AgentProvider,
) (catalogdomain.AgentProvider, error) {
	availableByID := make(map[uuid.UUID]catalogdomain.AgentProvider)
	for _, item := range providers {
		if item.Available {
			availableByID[item.ID] = item
		}
	}

	if project.DefaultAgentProviderID != nil {
		if item, ok := availableByID[*project.DefaultAgentProviderID]; ok {
			return item, nil
		}
	}
	if org.DefaultAgentProviderID != nil {
		if item, ok := availableByID[*org.DefaultAgentProviderID]; ok {
			return item, nil
		}
	}

	if item, ok := preferredActivationProvider(providers); ok {
		return item, nil
	}

	return catalogdomain.AgentProvider{}, fmt.Errorf(
		"%w: project %s has no available provider",
		ErrActivationProviderUnavailable,
		project.ID,
	)
}

func preferredActivationProvider(items []catalogdomain.AgentProvider) (catalogdomain.AgentProvider, bool) {
	preferred := []struct {
		name        string
		adapterType catalogdomain.AgentProviderAdapterType
	}{
		{name: "OpenAI Codex", adapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer},
		{name: "Claude Code", adapterType: catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI},
		{name: "Gemini CLI", adapterType: catalogdomain.AgentProviderAdapterTypeGeminiCLI},
	}
	for _, candidate := range preferred {
		for _, item := range items {
			if item.Available && item.Name == candidate.name && item.AdapterType == candidate.adapterType {
				return item, true
			}
		}
	}

	sorted := make([]catalogdomain.AgentProvider, 0, len(items))
	for _, item := range items {
		if item.Available {
			sorted = append(sorted, item)
		}
	}
	sort.Slice(sorted, func(i int, j int) bool {
		if sorted[i].Name == sorted[j].Name {
			return sorted[i].ID.String() < sorted[j].ID.String()
		}
		return sorted[i].Name < sorted[j].Name
	})
	if len(sorted) == 0 {
		return catalogdomain.AgentProvider{}, false
	}

	return sorted[0], true
}

func activationAgentName(template hrdomain.ActivationTemplate) string {
	return fmt.Sprintf("%s Agent", template.WorkflowName)
}

func normalizeActivationWorkflowType(raw string) entworkflow.Type {
	return entworkflow.Type(strings.TrimSpace(strings.ToLower(raw)))
}

func mustParseStatusBindingSet(id uuid.UUID) workflow.StatusBindingSet {
	set, err := workflow.ParseStatusBindingSet("status_ids", []uuid.UUID{id})
	if err != nil {
		panic(err)
	}
	return set
}

func stringPointer(value string) *string {
	return &value
}
