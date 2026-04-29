package projectpreset

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	"github.com/BetterAndBetterII/openase/internal/builtin"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	presetdomain "github.com/BetterAndBetterII/openase/internal/domain/projectpreset"
	"github.com/BetterAndBetterII/openase/internal/repo/enttx"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

var ErrUnavailable = errors.New("project preset service unavailable")

type Service struct {
	client              *ent.Client
	ticketService       *ticketservice.Service
	ticketStatusService *ticketstatus.Service
	workflowService     *workflowservice.Service
}

func NewService(
	client *ent.Client,
	ticketSvc *ticketservice.Service,
	statusSvc *ticketstatus.Service,
	workflowSvc *workflowservice.Service,
) *Service {
	return &Service{
		client:              client,
		ticketService:       ticketSvc,
		ticketStatusService: statusSvc,
		workflowService:     workflowSvc,
	}
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID) (presetdomain.Catalog, error) {
	if err := s.ensureAvailable(); err != nil {
		return presetdomain.Catalog{}, err
	}
	if err := s.ensureProjectExists(ctx, projectID); err != nil {
		return presetdomain.Catalog{}, err
	}
	activeTicketCount, err := s.countActiveTickets(ctx, projectID)
	if err != nil {
		return presetdomain.Catalog{}, err
	}
	return presetdomain.Catalog{
		ActiveTicketCount: activeTicketCount,
		CanApply:          activeTicketCount == 0,
		Presets:           builtin.ProjectPresets(),
	}, nil
}

func (s *Service) Apply(ctx context.Context, input presetdomain.ApplyInput) (presetdomain.ApplyResult, error) {
	if err := s.ensureAvailable(); err != nil {
		return presetdomain.ApplyResult{}, err
	}
	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return presetdomain.ApplyResult{}, err
	}
	preset, ok := builtin.ProjectPresetByKey(strings.TrimSpace(input.PresetKey))
	if !ok {
		return presetdomain.ApplyResult{}, presetdomain.ErrPresetNotFound
	}
	activeTicketCount, err := s.countActiveTickets(ctx, input.ProjectID)
	if err != nil {
		return presetdomain.ApplyResult{}, err
	}
	if activeTicketCount != 0 {
		return presetdomain.ApplyResult{}, fmt.Errorf("%w: project has %d active tickets", presetdomain.ErrActiveTicketsPresent, activeTicketCount)
	}
	agentNamesByID, bindingsByKey, err := s.resolveAgentBindings(ctx, input.ProjectID, preset, input.AgentBindings)
	if err != nil {
		return presetdomain.ApplyResult{}, err
	}

	txCtx, session, err := enttx.Begin(ctx, s.client)
	if err != nil {
		return presetdomain.ApplyResult{}, err
	}
	defer session.Rollback()

	currentStatuses, err := s.ticketStatusService.List(txCtx, input.ProjectID)
	if err != nil {
		return presetdomain.ApplyResult{}, err
	}
	currentStatusesByName := makeStatusMap(currentStatuses.Statuses)
	appliedStatuses := make([]presetdomain.AppliedStatus, 0, len(preset.Statuses))
	for index, statusPreset := range preset.Statuses {
		existing, exists := currentStatusesByName[strings.ToLower(statusPreset.Name)]
		if exists {
			updated, updateErr := s.ticketStatusService.Update(txCtx, ticketstatus.UpdateInput{
				StatusID:      existing.ID,
				Name:          ticketstatus.Some(statusPreset.Name),
				Stage:         ticketstatus.Some(statusPreset.Stage),
				Color:         ticketstatus.Some(statusPreset.Color),
				Icon:          ticketstatus.Some(statusPreset.Icon),
				Position:      ticketstatus.Some(index),
				MaxActiveRuns: ticketstatus.Some(statusPreset.MaxActiveRuns),
				IsDefault:     ticketstatus.Some(statusPreset.Default),
				Description:   ticketstatus.Some(statusPreset.Description),
			})
			if updateErr != nil {
				return presetdomain.ApplyResult{}, updateErr
			}
			appliedStatuses = append(appliedStatuses, presetdomain.AppliedStatus{ID: updated.ID, Name: updated.Name, Action: "updated"})
			continue
		}
		created, createErr := s.ticketStatusService.Create(txCtx, ticketstatus.CreateInput{
			ProjectID:     input.ProjectID,
			Name:          statusPreset.Name,
			Stage:         statusPreset.Stage,
			Color:         statusPreset.Color,
			Icon:          statusPreset.Icon,
			Position:      ticketstatus.Some(index),
			MaxActiveRuns: statusPreset.MaxActiveRuns,
			IsDefault:     statusPreset.Default,
			Description:   statusPreset.Description,
		})
		if createErr != nil {
			return presetdomain.ApplyResult{}, createErr
		}
		appliedStatuses = append(appliedStatuses, presetdomain.AppliedStatus{ID: created.ID, Name: created.Name, Action: "created"})
	}

	updatedStatuses, err := s.ticketStatusService.List(txCtx, input.ProjectID)
	if err != nil {
		return presetdomain.ApplyResult{}, err
	}
	statusIDsByName := make(map[string]uuid.UUID, len(updatedStatuses.Statuses))
	for _, item := range updatedStatuses.Statuses {
		statusIDsByName[strings.ToLower(item.Name)] = item.ID
	}

	currentWorkflows, err := s.workflowService.List(txCtx, input.ProjectID)
	if err != nil {
		return presetdomain.ApplyResult{}, err
	}
	currentWorkflowsByName := makeWorkflowMap(currentWorkflows)
	appliedWorkflows := make([]presetdomain.AppliedWorkflow, 0, len(preset.Workflows))
	appliedBy := normalizeAppliedBy(input.AppliedBy)
	for _, workflowPreset := range preset.Workflows {
		agentID := bindingsByKey[workflowPreset.Key]
		pickupStatusIDs, finishStatusIDs, resolveErr := resolveWorkflowStatusBindings(workflowPreset, statusIDsByName)
		if resolveErr != nil {
			return presetdomain.ApplyResult{}, resolveErr
		}
		harnessContent := strings.TrimSpace(workflowPreset.HarnessContent)
		if harnessContent == "" {
			harnessContent = workflowservice.DefaultHarnessContent(
				workflowPreset.Name,
				workflowPreset.Type,
				workflowPreset.PickupStatusNames,
				workflowPreset.FinishStatusNames,
			)
		}
		if existing, exists := currentWorkflowsByName[strings.ToLower(workflowPreset.Name)]; exists {
			if roleConflict(existing.RoleSlug, workflowPreset.RoleSlug) {
				return presetdomain.ApplyResult{}, fmt.Errorf(
					"%w: workflow %q already uses role slug %q",
					presetdomain.ErrWorkflowRoleConflict,
					existing.Name,
					existing.RoleSlug,
				)
			}
			updateInput := workflowservice.UpdateInput{
				WorkflowID:            existing.ID,
				AgentID:               workflowservice.Some(agentID),
				Name:                  workflowservice.Some(workflowPreset.Name),
				Type:                  workflowservice.Some(workflowPreset.Type),
				RoleName:              workflowservice.Some(workflowPreset.RoleName),
				RoleDescription:       workflowservice.Some(workflowPreset.RoleDescription),
				PlatformAccessAllowed: workflowservice.Some(workflowPreset.PlatformAccessAllowed),
				EditedBy:              appliedBy,
				MaxConcurrent:         workflowservice.Some(workflowPreset.MaxConcurrent),
				MaxRetryAttempts:      workflowservice.Some(workflowPreset.MaxRetryAttempts),
				TimeoutMinutes:        workflowservice.Some(workflowPreset.TimeoutMinutes),
				StallTimeoutMinutes:   workflowservice.Some(workflowPreset.StallTimeoutMinutes),
				IsActive:              workflowservice.Some(true),
				PickupStatusIDs:       workflowservice.Some(workflowservice.MustStatusBindingSet(pickupStatusIDs...)),
				FinishStatusIDs:       workflowservice.Some(workflowservice.MustStatusBindingSet(finishStatusIDs...)),
			}
			if workflowPreset.HarnessPath != nil {
				updateInput.HarnessPath = workflowservice.Some(*workflowPreset.HarnessPath)
			}
			detail, updateErr := s.workflowService.Update(txCtx, updateInput)
			if updateErr != nil {
				return presetdomain.ApplyResult{}, updateErr
			}
			if _, updateHarnessErr := s.workflowService.UpdateHarness(txCtx, workflowservice.UpdateHarnessInput{
				WorkflowID: detail.ID,
				Content:    harnessContent,
				EditedBy:   appliedBy,
			}); updateHarnessErr != nil {
				return presetdomain.ApplyResult{}, updateHarnessErr
			}
			appliedWorkflows = append(appliedWorkflows, presetdomain.AppliedWorkflow{
				ID:          detail.ID,
				Key:         workflowPreset.Key,
				Name:        detail.Name,
				AgentID:     agentID,
				AgentName:   agentNamesByID[agentID],
				Action:      "updated",
				HarnessPath: detail.HarnessPath,
			})
			continue
		}
		detail, createErr := s.workflowService.Create(txCtx, workflowservice.CreateInput{
			ProjectID:             input.ProjectID,
			AgentID:               agentID,
			Name:                  workflowPreset.Name,
			Type:                  workflowPreset.Type,
			RoleSlug:              workflowPreset.RoleSlug,
			RoleName:              workflowPreset.RoleName,
			RoleDescription:       workflowPreset.RoleDescription,
			PlatformAccessAllowed: workflowPreset.PlatformAccessAllowed,
			SkillNames:            workflowPreset.SkillNames,
			CreatedBy:             appliedBy,
			HarnessPath:           workflowPreset.HarnessPath,
			HarnessContent:        workflowPreset.HarnessContent,
			MaxConcurrent:         workflowPreset.MaxConcurrent,
			MaxRetryAttempts:      workflowPreset.MaxRetryAttempts,
			TimeoutMinutes:        workflowPreset.TimeoutMinutes,
			StallTimeoutMinutes:   workflowPreset.StallTimeoutMinutes,
			IsActive:              true,
			PickupStatusIDs:       workflowservice.MustStatusBindingSet(pickupStatusIDs...),
			FinishStatusIDs:       workflowservice.MustStatusBindingSet(finishStatusIDs...),
		})
		if createErr != nil {
			return presetdomain.ApplyResult{}, createErr
		}
		appliedWorkflows = append(appliedWorkflows, presetdomain.AppliedWorkflow{
			ID:          detail.ID,
			Key:         workflowPreset.Key,
			Name:        detail.Name,
			AgentID:     agentID,
			AgentName:   agentNamesByID[agentID],
			Action:      "created",
			HarnessPath: detail.HarnessPath,
		})
	}

	if err := session.Commit(); err != nil {
		return presetdomain.ApplyResult{}, fmt.Errorf("commit project preset apply tx: %w", err)
	}
	return presetdomain.ApplyResult{
		Preset:            preset,
		ActiveTicketCount: 0,
		Statuses:          appliedStatuses,
		Workflows:         appliedWorkflows,
	}, nil
}

func (s *Service) ensureAvailable() error {
	if s == nil || s.client == nil || s.ticketService == nil || s.ticketStatusService == nil || s.workflowService == nil {
		return ErrUnavailable
	}
	return nil
}

func (s *Service) ensureProjectExists(ctx context.Context, projectID uuid.UUID) error {
	exists, err := s.client.Project.Query().Where(entproject.IDEQ(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ticketstatus.ErrProjectNotFound
	}
	return nil
}

func (s *Service) countActiveTickets(ctx context.Context, projectID uuid.UUID) (int, error) {
	items, err := s.ticketService.List(ctx, ticketservice.ListInput{ProjectID: projectID})
	if err != nil {
		return 0, err
	}
	count := 0
	for _, item := range items {
		if !catalogdomain.IsTerminalTicketStatusStage(item.StatusStage) {
			count++
		}
	}
	return count, nil
}

func (s *Service) resolveAgentBindings(
	ctx context.Context,
	projectID uuid.UUID,
	preset presetdomain.Preset,
	bindings []presetdomain.WorkflowAgentBinding,
) (map[uuid.UUID]string, map[string]uuid.UUID, error) {
	agents, err := s.client.Agent.Query().
		Where(
			entagent.ProjectIDEQ(projectID),
			entagent.RuntimeControlStateNEQ(entagent.RuntimeControlStateRetired),
		).
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list project agents: %w", err)
	}
	agentNamesByID := make(map[uuid.UUID]string, len(agents))
	for _, item := range agents {
		agentNamesByID[item.ID] = item.Name
	}
	bindingByKey := make(map[string]uuid.UUID, len(bindings))
	for _, binding := range bindings {
		key := strings.ToLower(strings.TrimSpace(binding.WorkflowKey))
		if key == "" {
			return nil, nil, fmt.Errorf("%w: workflow_key must not be empty", presetdomain.ErrAgentBindingRequired)
		}
		if _, exists := bindingByKey[key]; exists {
			return nil, nil, fmt.Errorf("%w: duplicate workflow_key %q", presetdomain.ErrAgentBindingInvalid, binding.WorkflowKey)
		}
		if _, exists := agentNamesByID[binding.AgentID]; !exists {
			return nil, nil, fmt.Errorf("%w: agent %s is not available in this project", presetdomain.ErrAgentBindingInvalid, binding.AgentID)
		}
		bindingByKey[key] = binding.AgentID
	}
	for _, workflowPreset := range preset.Workflows {
		key := strings.ToLower(workflowPreset.Key)
		if _, ok := bindingByKey[key]; !ok {
			return nil, nil, fmt.Errorf("%w: workflow %q requires an agent selection", presetdomain.ErrAgentBindingRequired, workflowPreset.Name)
		}
	}
	for key := range bindingByKey {
		if !presetHasWorkflowKey(preset, key) {
			return nil, nil, fmt.Errorf("%w: workflow_key %q is not defined by preset %q", presetdomain.ErrAgentBindingInvalid, key, preset.Meta.Key)
		}
	}
	return agentNamesByID, bindingByKey, nil
}

func presetHasWorkflowKey(preset presetdomain.Preset, key string) bool {
	for _, item := range preset.Workflows {
		if strings.EqualFold(item.Key, key) {
			return true
		}
	}
	return false
}

func makeStatusMap(items []ticketstatus.Status) map[string]ticketstatus.Status {
	result := make(map[string]ticketstatus.Status, len(items))
	for _, item := range items {
		result[strings.ToLower(item.Name)] = item
	}
	return result
}

func makeWorkflowMap(items []workflowservice.Workflow) map[string]workflowservice.Workflow {
	result := make(map[string]workflowservice.Workflow, len(items))
	for _, item := range items {
		result[strings.ToLower(item.Name)] = item
	}
	return result
}

func resolveWorkflowStatusBindings(
	workflowPreset presetdomain.Workflow,
	statusIDsByName map[string]uuid.UUID,
) ([]uuid.UUID, []uuid.UUID, error) {
	pickup := make([]uuid.UUID, 0, len(workflowPreset.PickupStatusNames))
	for _, name := range workflowPreset.PickupStatusNames {
		id, ok := statusIDsByName[strings.ToLower(name)]
		if !ok {
			return nil, nil, fmt.Errorf("preset workflow %q references unknown pickup status %q", workflowPreset.Name, name)
		}
		pickup = append(pickup, id)
	}
	finish := make([]uuid.UUID, 0, len(workflowPreset.FinishStatusNames))
	for _, name := range workflowPreset.FinishStatusNames {
		id, ok := statusIDsByName[strings.ToLower(name)]
		if !ok {
			return nil, nil, fmt.Errorf("preset workflow %q references unknown finish status %q", workflowPreset.Name, name)
		}
		finish = append(finish, id)
	}
	return pickup, finish, nil
}

func normalizeAppliedBy(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "user:api"
	}
	return trimmed
}

func roleConflict(existingRoleSlug, presetRoleSlug string) bool {
	presetRoleSlug = strings.TrimSpace(presetRoleSlug)
	if presetRoleSlug == "" {
		return false
	}
	existingRoleSlug = strings.TrimSpace(existingRoleSlug)
	if existingRoleSlug == "" {
		return true
	}
	return existingRoleSlug != presetRoleSlug
}
