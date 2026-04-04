package hradvisor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/builtin"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	hrdomain "github.com/BetterAndBetterII/openase/internal/domain/hradvisor"
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

type ActivationStatus struct {
	ID    uuid.UUID
	Name  string
	Stage string
}

type ActivationWorkflow struct {
	ID                    uuid.UUID
	ProjectID             uuid.UUID
	AgentID               *uuid.UUID
	Name                  string
	Type                  string
	RoleSlug              string
	RoleName              string
	RoleDescription       string
	PlatformAccessAllowed []string
	HarnessPath           string
	HarnessContent        string
	MaxConcurrent         int
	MaxRetryAttempts      int
	TimeoutMinutes        int
	StallTimeoutMinutes   int
	Version               int
	IsActive              bool
	PickupStatusIDs       []uuid.UUID
	FinishStatusIDs       []uuid.UUID
}

type ActivateWorkflowInput struct {
	ProjectID             uuid.UUID
	AgentID               uuid.UUID
	Name                  string
	Type                  string
	RoleSlug              string
	RoleName              string
	RoleDescription       string
	PlatformAccessAllowed []string
	SkillNames            []string
	HarnessPath           string
	HarnessContent        string
	MaxConcurrent         int
	MaxRetryAttempts      int
	TimeoutMinutes        int
	StallTimeoutMinutes   int
	IsActive              bool
	PickupStatusIDs       []uuid.UUID
	FinishStatusIDs       []uuid.UUID
}

type ActivationTicket struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Identifier  string
	Title       string
	StatusID    uuid.UUID
	StatusName  string
	WorkflowID  *uuid.UUID
	CreatedBy   string
	Priority    string
	Type        string
	Description string
}

type CreateActivationTicketInput struct {
	ProjectID   uuid.UUID
	Title       string
	Description string
	StatusID    *uuid.UUID
	Priority    string
	Type        string
	WorkflowID  *uuid.UUID
	CreatedBy   string
}

type activationWorkflows interface {
	List(ctx context.Context, projectID uuid.UUID) ([]ActivationWorkflow, error)
	Create(ctx context.Context, input ActivateWorkflowInput) (ActivationWorkflow, error)
}

type activationStatuses interface {
	List(ctx context.Context, projectID uuid.UUID) ([]ActivationStatus, error)
}

type activationTickets interface {
	Create(ctx context.Context, input CreateActivationTicketInput) (ActivationTicket, error)
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
	Workflow        ActivationWorkflow
	BootstrapTicket BootstrapTicketResult
}

type BootstrapTicketResult struct {
	Requested bool
	Status    string
	Message   string
	Ticket    *ActivationTicket
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

	template, err := hrdomain.NormalizeActivationTemplate(
		hrdomain.ActivationTemplate{
			RoleSlug:              roleTemplate.Slug,
			WorkflowName:          roleTemplate.Name,
			WorkflowType:          roleTemplate.WorkflowType,
			HarnessPath:           roleTemplate.HarnessPath,
			HarnessContent:        roleTemplate.Content,
			PickupStatusNames:     roleTemplate.PickupStatusNames,
			FinishStatusNames:     roleTemplate.FinishStatusNames,
			Summary:               roleTemplate.Summary,
			PlatformAccessAllowed: roleTemplate.PlatformAccessAllowed,
			SkillNames:            roleTemplate.SkillNames,
		},
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

	statuses, err := s.statuses.List(ctx, input.ProjectID)
	if err != nil {
		return ActivationResult{}, err
	}
	pickupStatusIDs, finishStatusIDs, err := resolveActivationStatusIDs(statuses, existingWorkflows, template)
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

	createdWorkflow, err := s.workflows.Create(ctx, ActivateWorkflowInput{
		ProjectID:             input.ProjectID,
		AgentID:               createdAgent.ID,
		Name:                  template.WorkflowName,
		Type:                  strings.TrimSpace(template.WorkflowType),
		RoleSlug:              template.RoleSlug,
		RoleName:              template.WorkflowName,
		RoleDescription:       template.Summary,
		PlatformAccessAllowed: append([]string(nil), template.PlatformAccessAllowed...),
		SkillNames:            append([]string(nil), template.SkillNames...),
		HarnessPath:           template.HarnessPath,
		HarnessContent:        template.HarnessContent,
		MaxConcurrent:         1,
		MaxRetryAttempts:      1,
		TimeoutMinutes:        30,
		StallTimeoutMinutes:   5,
		IsActive:              true,
		PickupStatusIDs:       pickupStatusIDs,
		FinishStatusIDs:       finishStatusIDs,
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
	createdTicket, err := s.tickets.Create(ctx, CreateActivationTicketInput{
		ProjectID:   input.ProjectID,
		Title:       draft.Title,
		Description: draft.Description,
		StatusID:    &pickupStatusIDs[0],
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
	Priority    string
	Type        string
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
		Priority:    "medium",
		Type:        "chore",
	}
}

func activationWorkflowExists(items []ActivationWorkflow, harnessPath string) bool {
	normalizedPath := strings.TrimSpace(harnessPath)
	for _, item := range items {
		if strings.TrimSpace(item.HarnessPath) == normalizedPath {
			return true
		}
	}

	return false
}

func resolveActivationStatusIDs(
	statuses []ActivationStatus,
	existingWorkflows []ActivationWorkflow,
	template hrdomain.ActivationTemplate,
) ([]uuid.UUID, []uuid.UUID, error) {
	pickupStatusIDs, err := resolveActivationStatusBinding(statuses, template, "pickup", template.PickupStatusNames)
	if err != nil {
		return nil, nil, err
	}

	if template.RoleSlug == "dispatcher" {
		finishStatusIDs, err := resolveDispatcherFinishStatusIDs(statuses, existingWorkflows, pickupStatusIDs)
		if err != nil {
			return nil, nil, err
		}
		return pickupStatusIDs, finishStatusIDs, nil
	}

	finishStatusIDs, err := resolveActivationStatusBinding(statuses, template, "finish", template.FinishStatusNames)
	if err != nil {
		return nil, nil, err
	}

	return pickupStatusIDs, finishStatusIDs, nil
}

func resolveActivationStatusBinding(
	statuses []ActivationStatus,
	template hrdomain.ActivationTemplate,
	binding string,
	statusNames []string,
) ([]uuid.UUID, error) {
	statusesByName := make(map[string]ActivationStatus, len(statuses))
	statusesByStage := make(map[string][]ActivationStatus)
	for _, statusItem := range statuses {
		nameKey := normalizeActivationStatusKey(statusItem.Name)
		if nameKey != "" {
			statusesByName[nameKey] = statusItem
		}
		stageKey := normalizeActivationStatusKey(statusItem.Stage)
		if stageKey != "" {
			statusesByStage[stageKey] = append(statusesByStage[stageKey], statusItem)
		}
	}

	resolved := make([]uuid.UUID, 0, len(statusNames))
	seen := make(map[uuid.UUID]struct{}, len(statusNames))
	for _, rawName := range statusNames {
		statusItem, err := resolveActivationStatus(statusesByName, statusesByStage, template, binding, rawName)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[statusItem.ID]; ok {
			continue
		}
		seen[statusItem.ID] = struct{}{}
		resolved = append(resolved, statusItem.ID)
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("%w: %s binding requires at least one configured status", ErrActivationStatusNotFound, binding)
	}
	return resolved, nil
}

func resolveActivationStatus(
	statusesByName map[string]ActivationStatus,
	statusesByStage map[string][]ActivationStatus,
	template hrdomain.ActivationTemplate,
	binding string,
	rawName string,
) (ActivationStatus, error) {
	name := strings.TrimSpace(rawName)
	if item, ok := statusesByName[normalizeActivationStatusKey(name)]; ok {
		return item, nil
	}

	if template.RoleSlug == "dispatcher" && binding == "pickup" {
		candidates := statusesByStage["backlog"]
		if len(candidates) == 1 {
			return candidates[0], nil
		}
		if len(candidates) > 1 {
			names := make([]string, 0, len(candidates))
			for _, candidate := range candidates {
				names = append(names, candidate.Name)
			}
			sort.Strings(names)
			return ActivationStatus{}, fmt.Errorf(
				"%w: dispatcher %s status %q is ambiguous; configure exactly one backlog-stage status or restore the literal name. candidates=%s",
				ErrActivationStatusNotFound,
				binding,
				name,
				strings.Join(names, ", "),
			)
		}
		return ActivationStatus{}, fmt.Errorf(
			"%w: dispatcher %s status %q requires a configured status with stage \"backlog\"",
			ErrActivationStatusNotFound,
			binding,
			name,
		)
	}

	return ActivationStatus{}, fmt.Errorf("%w: %s status %q", ErrActivationStatusNotFound, binding, name)
}

func resolveDispatcherFinishStatusIDs(
	statuses []ActivationStatus,
	existingWorkflows []ActivationWorkflow,
	pickupStatusIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	statusByID := make(map[uuid.UUID]ActivationStatus, len(statuses))
	for _, statusItem := range statuses {
		statusByID[statusItem.ID] = statusItem
	}

	pickupSet := make(map[uuid.UUID]struct{}, len(pickupStatusIDs))
	for _, statusID := range pickupStatusIDs {
		pickupSet[statusID] = struct{}{}
	}

	if ids := collectDispatcherFinishStatusIDsFromWorkflows(existingWorkflows, statusByID, pickupSet); len(ids) > 0 {
		return ids, nil
	}
	if ids := collectDispatcherFinishStatusIDsFromStatuses(statuses, pickupSet, "unstarted"); len(ids) > 0 {
		return ids, nil
	}
	if ids := collectDispatcherFinishStatusIDsFromStatuses(statuses, pickupSet, "started"); len(ids) > 0 {
		return ids, nil
	}

	return nil, fmt.Errorf(
		"%w: dispatcher finish binding requires at least one configured non-backlog work status",
		ErrActivationStatusNotFound,
	)
}

func collectDispatcherFinishStatusIDsFromWorkflows(
	workflows []ActivationWorkflow,
	statusByID map[uuid.UUID]ActivationStatus,
	pickupSet map[uuid.UUID]struct{},
) []uuid.UUID {
	unstarted := make([]uuid.UUID, 0)
	started := make([]uuid.UUID, 0)
	seen := make(map[uuid.UUID]struct{})

	for _, workflow := range workflows {
		if !workflow.IsActive || strings.EqualFold(strings.TrimSpace(workflow.RoleSlug), "dispatcher") {
			continue
		}
		for _, statusID := range workflow.PickupStatusIDs {
			statusItem, ok := statusByID[statusID]
			if !ok {
				continue
			}
			if _, blocked := pickupSet[statusID]; blocked {
				continue
			}
			if _, alreadySeen := seen[statusID]; alreadySeen {
				continue
			}
			switch normalizeActivationStatusKey(statusItem.Stage) {
			case "unstarted":
				unstarted = append(unstarted, statusID)
				seen[statusID] = struct{}{}
			case "started":
				started = append(started, statusID)
				seen[statusID] = struct{}{}
			}
		}
	}

	if len(unstarted) > 0 {
		return unstarted
	}
	return started
}

func collectDispatcherFinishStatusIDsFromStatuses(
	statuses []ActivationStatus,
	pickupSet map[uuid.UUID]struct{},
	stage string,
) []uuid.UUID {
	normalizedStage := normalizeActivationStatusKey(stage)
	result := make([]uuid.UUID, 0)
	for _, statusItem := range statuses {
		if normalizeActivationStatusKey(statusItem.Stage) != normalizedStage {
			continue
		}
		if _, blocked := pickupSet[statusItem.ID]; blocked {
			continue
		}
		result = append(result, statusItem.ID)
	}
	return result
}

func normalizeActivationStatusKey(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
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
