package workflow

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	"github.com/google/uuid"
)

var (
	ErrUnavailable         = errors.New("workflow service unavailable")
	ErrProjectNotFound     = errors.New("project not found")
	ErrWorkflowNotFound    = errors.New("workflow not found")
	ErrStatusNotFound      = errors.New("workflow status not found in project")
	ErrWorkflowConflict    = errors.New("workflow conflict")
	ErrWorkflowInUse       = errors.New("workflow is still referenced by project or tickets")
	ErrHarnessInvalid      = errors.New("workflow harness is invalid")
	ErrHookConfigInvalid   = errors.New("workflow hook config is invalid")
	ErrWorkflowHookBlocked = errors.New("workflow hook blocked the lifecycle operation")
)

var nonAlphaNumericPattern = regexp.MustCompile(`[^a-z0-9]+`)

type Optional[T any] struct {
	Set   bool
	Value T
}

func Some[T any](value T) Optional[T] {
	return Optional[T]{Set: true, Value: value}
}

type Workflow struct {
	ID                  uuid.UUID        `json:"id"`
	ProjectID           uuid.UUID        `json:"project_id"`
	Name                string           `json:"name"`
	Type                entworkflow.Type `json:"type"`
	HarnessPath         string           `json:"harness_path"`
	Hooks               map[string]any   `json:"hooks"`
	MaxConcurrent       int              `json:"max_concurrent"`
	MaxRetryAttempts    int              `json:"max_retry_attempts"`
	TimeoutMinutes      int              `json:"timeout_minutes"`
	StallTimeoutMinutes int              `json:"stall_timeout_minutes"`
	Version             int              `json:"version"`
	IsActive            bool             `json:"is_active"`
	PickupStatusID      uuid.UUID        `json:"pickup_status_id"`
	FinishStatusID      *uuid.UUID       `json:"finish_status_id"`
}

type WorkflowDetail struct {
	Workflow
	HarnessContent string `json:"harness_content"`
}

type HarnessDocument struct {
	WorkflowID uuid.UUID `json:"workflow_id"`
	Path       string    `json:"path"`
	Content    string    `json:"content"`
	Version    int       `json:"version"`
}

type CreateInput struct {
	ProjectID           uuid.UUID
	Name                string
	Type                entworkflow.Type
	HarnessPath         *string
	HarnessContent      string
	Hooks               map[string]any
	MaxConcurrent       int
	MaxRetryAttempts    int
	TimeoutMinutes      int
	StallTimeoutMinutes int
	IsActive            bool
	PickupStatusID      uuid.UUID
	FinishStatusID      *uuid.UUID
}

type UpdateInput struct {
	WorkflowID          uuid.UUID
	Name                Optional[string]
	Type                Optional[entworkflow.Type]
	HarnessPath         Optional[string]
	Hooks               Optional[map[string]any]
	MaxConcurrent       Optional[int]
	MaxRetryAttempts    Optional[int]
	TimeoutMinutes      Optional[int]
	StallTimeoutMinutes Optional[int]
	IsActive            Optional[bool]
	PickupStatusID      Optional[uuid.UUID]
	FinishStatusID      Optional[*uuid.UUID]
}

type UpdateHarnessInput struct {
	WorkflowID uuid.UUID
	Content    string
}

type Service struct {
	client       *ent.Client
	logger       *slog.Logger
	repoRoot     string
	harnessRoot  string
	skillRoot    string
	registry     *harnessRegistry
	hookExecutor *workflowHookExecutor
}

func NewService(client *ent.Client, logger *slog.Logger, repoRoot string) (*Service, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	if repoRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("resolve current working directory: %w", err)
		}
		repoRoot, err = detectRepoRoot(cwd)
		if err != nil {
			return nil, err
		}
	}

	harnessRoot := filepath.Join(repoRoot, ".openase", "harnesses")
	skillRoot := filepath.Join(repoRoot, ".openase", "skills")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create skill root: %w", err)
	}
	service := &Service{
		client:       client,
		logger:       logger.With("component", "workflow-service"),
		repoRoot:     repoRoot,
		harnessRoot:  harnessRoot,
		skillRoot:    skillRoot,
		hookExecutor: newWorkflowHookExecutor(repoRoot, logger),
	}

	registry, err := newHarnessRegistry(harnessRoot, service.logger, service.handleHarnessReload)
	if err != nil {
		return nil, err
	}
	service.registry = registry

	return service, nil
}

func (s *Service) Close() error {
	if s == nil || s.registry == nil {
		return nil
	}

	return s.registry.Close()
}

func validateConfiguredHooks(raw map[string]any) (workflowHooksConfig, error) {
	parsedWorkflowHooks, err := parseWorkflowHooks(raw)
	if err != nil {
		return workflowHooksConfig{}, err
	}
	if _, err := infrahook.ParseTicketHooks(raw); err != nil {
		return workflowHooksConfig{}, mapTicketHookConfigError(err)
	}

	return parsedWorkflowHooks, nil
}

func mapTicketHookConfigError(err error) error {
	if err == nil {
		return nil
	}

	message := err.Error()
	if errors.Is(err, infrahook.ErrConfigInvalid) {
		prefix := infrahook.ErrConfigInvalid.Error() + ": "
		message = strings.TrimPrefix(message, prefix)
	}

	return fmt.Errorf("%w: %s", ErrHookConfigInvalid, message)
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]Workflow, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, projectID); err != nil {
		return nil, err
	}

	items, err := s.client.Workflow.Query().
		Where(entworkflow.ProjectIDEQ(projectID)).
		Order(ent.Asc(entworkflow.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}

	workflows := make([]Workflow, 0, len(items))
	for _, item := range items {
		workflows = append(workflows, mapWorkflow(item))
	}

	return workflows, nil
}

func (s *Service) Get(ctx context.Context, workflowID uuid.UUID) (WorkflowDetail, error) {
	if s.client == nil {
		return WorkflowDetail{}, ErrUnavailable
	}

	item, err := s.client.Workflow.Get(ctx, workflowID)
	if err != nil {
		return WorkflowDetail{}, s.mapWorkflowReadError("get workflow", err)
	}

	content, err := s.registry.Read(item.HarnessPath)
	if err != nil {
		return WorkflowDetail{}, fmt.Errorf("read workflow harness: %w", err)
	}

	return mapWorkflowDetail(item, content), nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (WorkflowDetail, error) {
	if s.client == nil {
		return WorkflowDetail{}, ErrUnavailable
	}

	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return WorkflowDetail{}, err
	}
	if err := s.ensureStatusBelongsToProject(ctx, input.ProjectID, input.PickupStatusID); err != nil {
		return WorkflowDetail{}, err
	}
	if input.FinishStatusID != nil {
		if err := s.ensureStatusBelongsToProject(ctx, input.ProjectID, *input.FinishStatusID); err != nil {
			return WorkflowDetail{}, err
		}
	}

	harnessPath, err := s.resolveCreateHarnessPath(ctx, input.ProjectID, input.Name, input.HarnessPath)
	if err != nil {
		return WorkflowDetail{}, err
	}
	if err := s.ensureHarnessPathAvailable(ctx, harnessPath, uuid.Nil); err != nil {
		return WorkflowDetail{}, err
	}
	parsedHooks, err := validateConfiguredHooks(input.Hooks)
	if err != nil {
		return WorkflowDetail{}, err
	}

	harnessContent, err := s.resolveHarnessContent(ctx, input.Name, input.Type, input.PickupStatusID, input.FinishStatusID, input.HarnessContent)
	if err != nil {
		return WorkflowDetail{}, err
	}
	if err := s.registry.Write(harnessPath, harnessContent); err != nil {
		return WorkflowDetail{}, err
	}

	workflowID := uuid.New()
	if input.IsActive {
		if err := s.runWorkflowHooks(ctx, parsedHooks, workflowHookOnActivate, workflowHookRuntime{
			ProjectID:       input.ProjectID,
			WorkflowID:      workflowID,
			WorkflowName:    input.Name,
			WorkflowVersion: 1,
		}); err != nil {
			_ = s.registry.Delete(harnessPath)
			return WorkflowDetail{}, err
		}
	}

	builder := s.client.Workflow.Create().
		SetID(workflowID).
		SetProjectID(input.ProjectID).
		SetName(input.Name).
		SetType(input.Type).
		SetHarnessPath(harnessPath).
		SetHooks(copyHooks(input.Hooks)).
		SetMaxConcurrent(input.MaxConcurrent).
		SetMaxRetryAttempts(input.MaxRetryAttempts).
		SetTimeoutMinutes(input.TimeoutMinutes).
		SetStallTimeoutMinutes(input.StallTimeoutMinutes).
		SetIsActive(input.IsActive).
		SetPickupStatusID(input.PickupStatusID)
	if input.FinishStatusID != nil {
		builder.SetFinishStatusID(*input.FinishStatusID)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		_ = s.registry.Delete(harnessPath)
		return WorkflowDetail{}, s.mapWorkflowWriteError("create workflow", err)
	}

	return mapWorkflowDetail(item, harnessContent), nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (WorkflowDetail, error) {
	if s.client == nil {
		return WorkflowDetail{}, ErrUnavailable
	}

	current, err := s.client.Workflow.Get(ctx, input.WorkflowID)
	if err != nil {
		return WorkflowDetail{}, s.mapWorkflowReadError("get workflow for update", err)
	}

	projectID := current.ProjectID
	nextName := current.Name
	if input.Name.Set {
		nextName = input.Name.Value
	}

	nextHarnessPath := current.HarnessPath
	if input.HarnessPath.Set {
		nextHarnessPath, err = normalizeHarnessPath(input.HarnessPath.Value)
		if err != nil {
			return WorkflowDetail{}, err
		}
	}
	if !input.HarnessPath.Set && current.HarnessPath == "" {
		nextHarnessPath = defaultHarnessPath(projectID, nextName)
	}
	if nextHarnessPath != current.HarnessPath {
		if err := s.ensureHarnessPathAvailable(ctx, nextHarnessPath, current.ID); err != nil {
			return WorkflowDetail{}, err
		}
	}

	nextPickupStatusID := current.PickupStatusID
	if input.PickupStatusID.Set {
		nextPickupStatusID = input.PickupStatusID.Value
	}
	if err := s.ensureStatusBelongsToProject(ctx, projectID, nextPickupStatusID); err != nil {
		return WorkflowDetail{}, err
	}

	nextFinishStatusID := current.FinishStatusID
	if input.FinishStatusID.Set {
		nextFinishStatusID = input.FinishStatusID.Value
	}
	if nextFinishStatusID != nil {
		if err := s.ensureStatusBelongsToProject(ctx, projectID, *nextFinishStatusID); err != nil {
			return WorkflowDetail{}, err
		}
	}

	nextHooksRaw := current.Hooks
	if input.Hooks.Set {
		nextHooksRaw = input.Hooks.Value
	}
	parsedHooks, err := validateConfiguredHooks(nextHooksRaw)
	if err != nil {
		return WorkflowDetail{}, err
	}

	nextIsActive := current.IsActive
	if input.IsActive.Set {
		nextIsActive = input.IsActive.Value
	}

	if nextHarnessPath != current.HarnessPath {
		content, readErr := s.registry.Read(current.HarnessPath)
		if readErr != nil {
			return WorkflowDetail{}, fmt.Errorf("read workflow harness before move: %w", readErr)
		}
		if err := s.registry.Write(nextHarnessPath, content); err != nil {
			return WorkflowDetail{}, err
		}
	}

	if !current.IsActive && nextIsActive {
		if err := s.runWorkflowHooks(ctx, parsedHooks, workflowHookOnActivate, workflowHookRuntime{
			ProjectID:       current.ProjectID,
			WorkflowID:      current.ID,
			WorkflowName:    nextName,
			WorkflowVersion: current.Version,
		}); err != nil {
			if nextHarnessPath != current.HarnessPath {
				_ = s.registry.Delete(nextHarnessPath)
			}
			return WorkflowDetail{}, err
		}
	}

	builder := s.client.Workflow.UpdateOneID(current.ID)
	if input.Name.Set {
		builder.SetName(input.Name.Value)
	}
	if input.Type.Set {
		builder.SetType(input.Type.Value)
	}
	if nextHarnessPath != current.HarnessPath {
		builder.SetHarnessPath(nextHarnessPath)
	}
	if input.Hooks.Set {
		builder.SetHooks(copyHooks(input.Hooks.Value))
	}
	if input.MaxConcurrent.Set {
		builder.SetMaxConcurrent(input.MaxConcurrent.Value)
	}
	if input.MaxRetryAttempts.Set {
		builder.SetMaxRetryAttempts(input.MaxRetryAttempts.Value)
	}
	if input.TimeoutMinutes.Set {
		builder.SetTimeoutMinutes(input.TimeoutMinutes.Value)
	}
	if input.StallTimeoutMinutes.Set {
		builder.SetStallTimeoutMinutes(input.StallTimeoutMinutes.Value)
	}
	if input.IsActive.Set {
		builder.SetIsActive(input.IsActive.Value)
	}
	if input.PickupStatusID.Set {
		builder.SetPickupStatusID(input.PickupStatusID.Value)
	}
	if input.FinishStatusID.Set {
		if input.FinishStatusID.Value == nil {
			builder.ClearFinishStatusID()
		} else {
			builder.SetFinishStatusID(*input.FinishStatusID.Value)
		}
	}

	item, err := builder.Save(ctx)
	if err != nil {
		if nextHarnessPath != current.HarnessPath {
			_ = s.registry.Delete(nextHarnessPath)
		}
		return WorkflowDetail{}, s.mapWorkflowWriteError("update workflow", err)
	}

	if nextHarnessPath != current.HarnessPath {
		if err := s.registry.Delete(current.HarnessPath); err != nil {
			s.logger.Error("delete old workflow harness after move", "error", err, "path", current.HarnessPath)
		}
	}

	content, err := s.registry.Read(item.HarnessPath)
	if err != nil {
		return WorkflowDetail{}, fmt.Errorf("read workflow harness after update: %w", err)
	}

	return mapWorkflowDetail(item, content), nil
}

func (s *Service) Delete(ctx context.Context, workflowID uuid.UUID) (Workflow, error) {
	if s.client == nil {
		return Workflow{}, ErrUnavailable
	}

	current, err := s.client.Workflow.Get(ctx, workflowID)
	if err != nil {
		return Workflow{}, s.mapWorkflowReadError("get workflow for delete", err)
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Workflow{}, fmt.Errorf("start workflow delete tx: %w", err)
	}
	defer rollback(tx)

	if err := tx.Project.Update().
		Where(entproject.DefaultWorkflowIDEQ(current.ID)).
		ClearDefaultWorkflowID().
		Exec(ctx); err != nil {
		return Workflow{}, fmt.Errorf("clear project default workflow reference: %w", err)
	}

	if err := tx.Workflow.DeleteOneID(current.ID).Exec(ctx); err != nil {
		return Workflow{}, s.mapWorkflowWriteError("delete workflow", err)
	}

	if err := tx.Commit(); err != nil {
		return Workflow{}, fmt.Errorf("commit workflow delete tx: %w", err)
	}

	if err := s.registry.Delete(current.HarnessPath); err != nil {
		return Workflow{}, err
	}

	return mapWorkflow(current), nil
}

func (s *Service) GetHarness(ctx context.Context, workflowID uuid.UUID) (HarnessDocument, error) {
	if s.client == nil {
		return HarnessDocument{}, ErrUnavailable
	}

	item, err := s.client.Workflow.Get(ctx, workflowID)
	if err != nil {
		return HarnessDocument{}, s.mapWorkflowReadError("get workflow for harness", err)
	}

	content, err := s.registry.Read(item.HarnessPath)
	if err != nil {
		return HarnessDocument{}, fmt.Errorf("read workflow harness: %w", err)
	}

	return HarnessDocument{
		WorkflowID: item.ID,
		Path:       item.HarnessPath,
		Content:    content,
		Version:    item.Version,
	}, nil
}

func (s *Service) UpdateHarness(ctx context.Context, input UpdateHarnessInput) (HarnessDocument, error) {
	if s.client == nil {
		return HarnessDocument{}, ErrUnavailable
	}
	if err := validateHarnessForSave(input.Content); err != nil {
		return HarnessDocument{}, err
	}

	item, err := s.client.Workflow.Get(ctx, input.WorkflowID)
	if err != nil {
		return HarnessDocument{}, s.mapWorkflowReadError("get workflow for harness update", err)
	}
	parsedHooks, err := validateConfiguredHooks(item.Hooks)
	if err != nil {
		return HarnessDocument{}, err
	}

	previousContent, err := s.registry.Read(item.HarnessPath)
	if err != nil {
		return HarnessDocument{}, fmt.Errorf("read workflow harness before update: %w", err)
	}

	if err := s.registry.Write(item.HarnessPath, input.Content); err != nil {
		return HarnessDocument{}, err
	}

	if item.IsActive {
		if err := s.runWorkflowHooks(ctx, parsedHooks, workflowHookOnReload, workflowHookRuntime{
			ProjectID:       item.ProjectID,
			WorkflowID:      item.ID,
			WorkflowName:    item.Name,
			WorkflowVersion: item.Version + 1,
		}); err != nil {
			if restoreErr := s.registry.Write(item.HarnessPath, previousContent); restoreErr != nil {
				s.logger.Error("restore workflow harness after reload hook failure", "error", restoreErr, "workflow_id", item.ID, "path", item.HarnessPath)
			}
			return HarnessDocument{}, err
		}
	}

	updated, err := s.client.Workflow.UpdateOneID(item.ID).
		SetVersion(item.Version + 1).
		Save(ctx)
	if err != nil {
		if restoreErr := s.registry.Write(item.HarnessPath, previousContent); restoreErr != nil {
			s.logger.Error("restore workflow harness after version update failure", "error", restoreErr, "workflow_id", item.ID, "path", item.HarnessPath)
		}
		return HarnessDocument{}, s.mapWorkflowWriteError("update workflow harness version", err)
	}

	return HarnessDocument{
		WorkflowID: updated.ID,
		Path:       updated.HarnessPath,
		Content:    input.Content,
		Version:    updated.Version,
	}, nil
}

func (s *Service) ensureProjectExists(ctx context.Context, projectID uuid.UUID) error {
	exists, err := s.client.Project.Query().Where(entproject.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}

	return nil
}

func (s *Service) ensureStatusBelongsToProject(ctx context.Context, projectID uuid.UUID, statusID uuid.UUID) error {
	exists, err := s.client.TicketStatus.Query().
		Where(
			entticketstatus.ID(statusID),
			entticketstatus.ProjectIDEQ(projectID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check workflow status existence: %w", err)
	}
	if !exists {
		return ErrStatusNotFound
	}

	return nil
}

func (s *Service) resolveCreateHarnessPath(ctx context.Context, projectID uuid.UUID, name string, rawPath *string) (string, error) {
	if rawPath != nil {
		return normalizeHarnessPath(*rawPath)
	}

	return defaultHarnessPath(projectID, name), nil
}

func (s *Service) ensureHarnessPathAvailable(ctx context.Context, harnessPath string, excludeWorkflowID uuid.UUID) error {
	query := s.client.Workflow.Query().Where(entworkflow.HarnessPathEQ(harnessPath))
	if excludeWorkflowID != uuid.Nil {
		query = query.Where(entworkflow.IDNEQ(excludeWorkflowID))
	}
	exists, err := query.Exist(ctx)
	if err != nil {
		return fmt.Errorf("check workflow harness path uniqueness: %w", err)
	}
	if exists || s.registry.Exists(harnessPath) {
		return ErrWorkflowConflict
	}

	return nil
}

func (s *Service) resolveHarnessContent(
	ctx context.Context,
	name string,
	workflowType entworkflow.Type,
	pickupStatusID uuid.UUID,
	finishStatusID *uuid.UUID,
	rawContent string,
) (string, error) {
	if strings.TrimSpace(rawContent) != "" {
		if err := validateHarnessForSave(rawContent); err != nil {
			return "", err
		}
		return rawContent, nil
	}

	pickupStatus, err := s.client.TicketStatus.Get(ctx, pickupStatusID)
	if err != nil {
		return "", s.mapWorkflowReadError("get pickup status for harness template", err)
	}

	finishStatusName := ""
	if finishStatusID != nil {
		finishStatus, finishErr := s.client.TicketStatus.Get(ctx, *finishStatusID)
		if finishErr != nil {
			return "", s.mapWorkflowReadError("get finish status for harness template", finishErr)
		}
		finishStatusName = finishStatus.Name
	}

	content := defaultHarnessContent(name, workflowType, pickupStatus.Name, finishStatusName)
	if err := validateHarnessForSave(content); err != nil {
		return "", err
	}

	return content, nil
}

func (s *Service) handleHarnessReload(event harnessReloadEvent) {
	if s.client == nil {
		return
	}

	ctx := context.Background()
	item, err := s.client.Workflow.Query().Where(entworkflow.HarnessPathEQ(event.RelativePath)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return
		}

		s.logger.Error("reload harness workflow lookup", "error", err, "path", event.RelativePath)
		return
	}

	parsedHooks, err := validateConfiguredHooks(item.Hooks)
	if err != nil {
		s.logger.Error("parse workflow hooks for reload", "error", err, "workflow_id", item.ID, "path", event.RelativePath)
		return
	}

	if item.IsActive {
		if err := s.runWorkflowHooks(ctx, parsedHooks, workflowHookOnReload, workflowHookRuntime{
			ProjectID:       item.ProjectID,
			WorkflowID:      item.ID,
			WorkflowName:    item.Name,
			WorkflowVersion: item.Version + 1,
		}); err != nil {
			s.logger.Error("workflow reload hook blocked harness reload", "error", err, "workflow_id", item.ID, "path", event.RelativePath)
			if event.PreviousContent != "" {
				if restoreErr := s.registry.Write(event.RelativePath, event.PreviousContent); restoreErr != nil {
					s.logger.Error("restore workflow harness after blocked reload", "error", restoreErr, "workflow_id", item.ID, "path", event.RelativePath)
				}
			}
			return
		}
	}

	if _, err := s.client.Workflow.UpdateOneID(item.ID).SetVersion(item.Version + 1).Save(ctx); err != nil {
		s.logger.Error("bump workflow version after harness reload", "error", err, "workflow_id", item.ID, "path", event.RelativePath)
		if event.PreviousContent != "" {
			if restoreErr := s.registry.Write(event.RelativePath, event.PreviousContent); restoreErr != nil {
				s.logger.Error("restore workflow harness after failed version bump", "error", restoreErr, "workflow_id", item.ID, "path", event.RelativePath)
			}
		}
		return
	}

	s.logger.Info("workflow harness reloaded", "workflow_id", item.ID, "path", event.RelativePath)
}

func (s *Service) runWorkflowHooks(ctx context.Context, hooks workflowHooksConfig, hookName workflowHookName, runtime workflowHookRuntime) error {
	if s == nil || s.hookExecutor == nil {
		return nil
	}

	switch hookName {
	case workflowHookOnActivate:
		return s.hookExecutor.RunAll(ctx, hookName, hooks.OnActivate, runtime)
	case workflowHookOnReload:
		return s.hookExecutor.RunAll(ctx, hookName, hooks.OnReload, runtime)
	default:
		return nil
	}
}

func (s *Service) mapWorkflowReadError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return ErrWorkflowNotFound
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func (s *Service) mapWorkflowWriteError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return ErrWorkflowNotFound
	case ent.IsConstraintError(err):
		return ErrWorkflowConflict
	case strings.Contains(strings.ToLower(err.Error()), "tickets"):
		return ErrWorkflowInUse
	case strings.Contains(strings.ToLower(err.Error()), "scheduled_jobs"):
		return ErrWorkflowInUse
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func mapWorkflow(item *ent.Workflow) Workflow {
	return Workflow{
		ID:                  item.ID,
		ProjectID:           item.ProjectID,
		Name:                item.Name,
		Type:                item.Type,
		HarnessPath:         item.HarnessPath,
		Hooks:               copyHooks(item.Hooks),
		MaxConcurrent:       item.MaxConcurrent,
		MaxRetryAttempts:    item.MaxRetryAttempts,
		TimeoutMinutes:      item.TimeoutMinutes,
		StallTimeoutMinutes: item.StallTimeoutMinutes,
		Version:             item.Version,
		IsActive:            item.IsActive,
		PickupStatusID:      item.PickupStatusID,
		FinishStatusID:      item.FinishStatusID,
	}
}

func mapWorkflowDetail(item *ent.Workflow, harnessContent string) WorkflowDetail {
	return WorkflowDetail{
		Workflow:       mapWorkflow(item),
		HarnessContent: harnessContent,
	}
}

func copyHooks(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}

	copied := make(map[string]any, len(source))
	for key, value := range source {
		copied[key] = value
	}

	return copied
}

func defaultHarnessPath(projectID uuid.UUID, workflowName string) string {
	slug := slugify(workflowName)
	if slug == "" {
		slug = "workflow"
	}

	return filepath.ToSlash(filepath.Join(".openase", "harnesses", projectID.String(), slug+".md"))
}

func normalizeHarnessPath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: harness_path must not be empty", ErrHarnessInvalid)
	}
	if filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("%w: harness_path must be relative to the repo root", ErrHarnessInvalid)
	}

	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", fmt.Errorf("%w: harness_path must stay within the repo root", ErrHarnessInvalid)
	}
	if !strings.HasPrefix(cleaned, ".openase/harnesses/") {
		return "", fmt.Errorf("%w: harness_path must stay under .openase/harnesses/", ErrHarnessInvalid)
	}
	if strings.HasSuffix(cleaned, "/") || strings.HasSuffix(cleaned, ".") {
		return "", fmt.Errorf("%w: harness_path must point to a file", ErrHarnessInvalid)
	}

	return cleaned, nil
}

func defaultHarnessContent(name string, workflowType entworkflow.Type, pickupStatusName string, finishStatusName string) string {
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("workflow:\n")
	builder.WriteString(fmt.Sprintf("  name: %q\n", name))
	builder.WriteString(fmt.Sprintf("  type: %q\n", workflowType.String()))
	builder.WriteString("status:\n")
	builder.WriteString(fmt.Sprintf("  pickup: %q\n", pickupStatusName))
	if finishStatusName != "" {
		builder.WriteString(fmt.Sprintf("  finish: %q\n", finishStatusName))
	}
	builder.WriteString("---\n\n")
	builder.WriteString("# ")
	builder.WriteString(name)
	builder.WriteString("\n\n")
	builder.WriteString("Describe the role, constraints, and expected outputs for this workflow.\n")

	return builder.String()
}

func slugify(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	trimmed = nonAlphaNumericPattern.ReplaceAllString(trimmed, "-")
	return strings.Trim(trimmed, "-")
}

func detectRepoRoot(start string) (string, error) {
	current := start
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("inspect repository root: %w", err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("%w: could not find git repository root from %s", ErrHarnessInvalid, start)
		}
		current = parent
	}
}

func rollback(tx *ent.Tx) {
	if tx == nil {
		return
	}

	_ = tx.Rollback()
}
