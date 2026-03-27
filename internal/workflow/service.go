package workflow

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	"github.com/google/uuid"
)

var (
	ErrUnavailable            = errors.New("workflow service unavailable")
	ErrProjectNotFound        = errors.New("project not found")
	ErrPrimaryRepoUnavailable = errors.New("project primary repo is unavailable")
	ErrWorkflowNotFound       = errors.New("workflow not found")
	ErrStatusNotFound         = errors.New("workflow status not found in project")
	ErrAgentNotFound          = errors.New("workflow agent not found in project")
	ErrWorkflowConflict       = errors.New("workflow conflict")
	ErrWorkflowInUse          = errors.New("workflow is still referenced by project or tickets")
	ErrHarnessInvalid         = errors.New("workflow harness is invalid")
	ErrHookConfigInvalid      = errors.New("workflow hook config is invalid")
	ErrWorkflowHookBlocked    = errors.New("workflow hook blocked the lifecycle operation")
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
	AgentID             *uuid.UUID       `json:"agent_id"`
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
	PickupStatusIDs     []uuid.UUID      `json:"pickup_status_ids"`
	FinishStatusIDs     []uuid.UUID      `json:"finish_status_ids"`
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
	AgentID             uuid.UUID
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
	PickupStatusIDs     StatusBindingSet
	FinishStatusIDs     StatusBindingSet
}

type UpdateInput struct {
	WorkflowID          uuid.UUID
	AgentID             Optional[uuid.UUID]
	Name                Optional[string]
	Type                Optional[entworkflow.Type]
	HarnessPath         Optional[string]
	Hooks               Optional[map[string]any]
	MaxConcurrent       Optional[int]
	MaxRetryAttempts    Optional[int]
	TimeoutMinutes      Optional[int]
	StallTimeoutMinutes Optional[int]
	IsActive            Optional[bool]
	PickupStatusIDs     Optional[StatusBindingSet]
	FinishStatusIDs     Optional[StatusBindingSet]
}

type UpdateHarnessInput struct {
	WorkflowID uuid.UUID
	Content    string
}

type Service struct {
	client    *ent.Client
	logger    *slog.Logger
	repoRoot  string
	storageMu sync.Mutex
	storages  map[uuid.UUID]*projectStorage
}

type projectStorage struct {
	projectID    uuid.UUID
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
		repoRoot, err = DetectRepoRoot(cwd)
		if err != nil {
			return nil, err
		}
	}

	service := &Service{
		client:   client,
		logger:   logger.With("component", "workflow-service"),
		repoRoot: repoRoot,
		storages: map[uuid.UUID]*projectStorage{},
	}

	return service, nil
}

func newProjectStorage(projectID uuid.UUID, repoRoot string, service *Service) (*projectStorage, error) {
	harnessRoot := filepath.Join(repoRoot, ".openase", "harnesses")
	skillRoot := filepath.Join(repoRoot, ".openase", "skills")
	if err := os.MkdirAll(skillRoot, 0o750); err != nil {
		return nil, fmt.Errorf("create skill root: %w", err)
	}

	storage := &projectStorage{
		projectID:    projectID,
		repoRoot:     repoRoot,
		harnessRoot:  harnessRoot,
		skillRoot:    skillRoot,
		hookExecutor: newWorkflowHookExecutor(repoRoot, service.logger),
	}

	registry, err := newHarnessRegistry(harnessRoot, service.logger, func(event harnessReloadEvent) {
		event.ProjectID = projectID
		service.handleHarnessReload(event)
	})
	if err != nil {
		return nil, err
	}
	storage.registry = registry

	return storage, nil
}

func (s *projectStorage) Close() error {
	if s == nil || s.registry == nil {
		return nil
	}

	return s.registry.Close()
}

func (s *Service) storageForProject(ctx context.Context, projectID uuid.UUID) (*projectStorage, error) {
	repoRoot, err := s.resolvePrimaryRepoRoot(ctx, projectID)
	if err != nil {
		return nil, err
	}

	s.storageMu.Lock()
	defer s.storageMu.Unlock()

	if existing, ok := s.storages[projectID]; ok && existing.repoRoot == repoRoot {
		return existing, nil
	}

	storage, err := newProjectStorage(projectID, repoRoot, s)
	if err != nil {
		return nil, err
	}

	if existing := s.storages[projectID]; existing != nil {
		if closeErr := existing.Close(); closeErr != nil {
			s.logger.Error("close replaced project storage", "error", closeErr, "project_id", projectID)
		}
	}
	s.storages[projectID] = storage

	return storage, nil
}

func (s *Service) resolvePrimaryRepoRoot(ctx context.Context, projectID uuid.UUID) (string, error) {
	repoItem, err := s.client.ProjectRepo.Query().
		Where(
			entprojectrepo.ProjectID(projectID),
			entprojectrepo.IsPrimary(true),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", ErrPrimaryRepoUnavailable
		}
		return "", fmt.Errorf("get primary project repo: %w", err)
	}

	for _, candidate := range []string{repoItem.ClonePath, repoItem.RepositoryURL} {
		repoRoot, ok, candidateErr := resolveLocalProjectRepoRoot(candidate)
		if candidateErr != nil {
			return "", fmt.Errorf("%w: %s", ErrPrimaryRepoUnavailable, candidateErr)
		}
		if ok {
			return repoRoot, nil
		}
	}

	return "", fmt.Errorf(
		"%w: primary repo %q must expose a local repository path via clone_path or repository_url",
		ErrPrimaryRepoUnavailable,
		repoItem.Name,
	)
}

func resolveLocalProjectRepoRoot(raw string) (string, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false, nil
	}

	if filepath.IsAbs(trimmed) {
		repoRoot, err := DetectRepoRoot(filepath.Clean(trimmed))
		if err != nil {
			return "", false, err
		}
		return repoRoot, true, nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false, fmt.Errorf("parse project repo location %q: %w", trimmed, err)
	}
	if parsed.Scheme != "file" {
		return "", false, nil
	}

	repoPath, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return "", false, fmt.Errorf("decode project repo file URI %q: %w", trimmed, err)
	}
	if repoPath == "" {
		return "", false, fmt.Errorf("project repo file URI %q must include a path", trimmed)
	}

	repoRoot, err := DetectRepoRoot(filepath.Clean(filepath.FromSlash(repoPath)))
	if err != nil {
		return "", false, err
	}
	return repoRoot, true, nil
}

func (s *Service) Close() error {
	if s == nil {
		return nil
	}

	s.storageMu.Lock()
	storages := make([]*projectStorage, 0, len(s.storages))
	for _, storage := range s.storages {
		storages = append(storages, storage)
	}
	s.storages = map[uuid.UUID]*projectStorage{}
	s.storageMu.Unlock()

	var closeErr error
	for _, storage := range storages {
		if err := storage.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}

	return closeErr
}

func (s *Service) RepoRoot() string {
	if s == nil {
		return ""
	}

	return s.repoRoot
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
		WithPickupStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		WithFinishStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
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

	item, err := s.client.Workflow.Query().
		Where(entworkflow.IDEQ(workflowID)).
		WithPickupStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		WithFinishStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		Only(ctx)
	if err != nil {
		return WorkflowDetail{}, s.mapWorkflowReadError("get workflow", err)
	}
	storage, err := s.storageForProject(ctx, item.ProjectID)
	if err != nil {
		return WorkflowDetail{}, err
	}

	content, err := storage.registry.Read(item.HarnessPath)
	if err != nil {
		return WorkflowDetail{}, fmt.Errorf("read workflow harness: %w", err)
	}

	return mapWorkflowDetail(item, content), nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (WorkflowDetail, error) {
	if s.client == nil {
		return WorkflowDetail{}, ErrUnavailable
	}

	pickupStatusIDs := input.PickupStatusIDs
	finishStatusIDs := input.FinishStatusIDs

	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return WorkflowDetail{}, err
	}
	if err := s.ensureAgentBelongsToProject(ctx, input.ProjectID, input.AgentID); err != nil {
		return WorkflowDetail{}, err
	}
	if err := s.ensureStatusBindingsBelongToProject(ctx, input.ProjectID, pickupStatusIDs); err != nil {
		return WorkflowDetail{}, err
	}
	if err := s.ensureStatusBindingsBelongToProject(ctx, input.ProjectID, finishStatusIDs); err != nil {
		return WorkflowDetail{}, err
	}
	storage, err := s.storageForProject(ctx, input.ProjectID)
	if err != nil {
		return WorkflowDetail{}, err
	}

	harnessPath, err := s.resolveCreateHarnessPath(input.Name, input.HarnessPath)
	if err != nil {
		return WorkflowDetail{}, err
	}
	if err := s.ensureHarnessPathAvailable(ctx, input.ProjectID, storage, harnessPath, uuid.Nil); err != nil {
		return WorkflowDetail{}, err
	}
	parsedHooks, err := validateConfiguredHooks(input.Hooks)
	if err != nil {
		return WorkflowDetail{}, err
	}

	harnessContent, err := s.resolveHarnessContent(ctx, input.Name, input.Type, pickupStatusIDs, finishStatusIDs, input.HarnessContent)
	if err != nil {
		return WorkflowDetail{}, err
	}
	if err := storage.registry.Write(harnessPath, harnessContent); err != nil {
		return WorkflowDetail{}, err
	}

	workflowID := uuid.New()
	if input.IsActive {
		if err := s.runWorkflowHooks(ctx, storage, parsedHooks, workflowHookOnActivate, workflowHookRuntime{
			ProjectID:       input.ProjectID,
			WorkflowID:      workflowID,
			WorkflowName:    input.Name,
			WorkflowVersion: 1,
		}); err != nil {
			_ = storage.registry.Delete(harnessPath)
			return WorkflowDetail{}, err
		}
	}

	builder := s.client.Workflow.Create().
		SetID(workflowID).
		SetProjectID(input.ProjectID).
		SetAgentID(input.AgentID).
		SetName(input.Name).
		SetType(input.Type).
		SetHarnessPath(harnessPath).
		SetHooks(copyHooks(input.Hooks)).
		SetMaxConcurrent(input.MaxConcurrent).
		SetMaxRetryAttempts(input.MaxRetryAttempts).
		SetTimeoutMinutes(input.TimeoutMinutes).
		SetStallTimeoutMinutes(input.StallTimeoutMinutes).
		SetIsActive(input.IsActive).
		AddPickupStatusIDs(pickupStatusIDs.IDs()...).
		AddFinishStatusIDs(finishStatusIDs.IDs()...)

	item, err := builder.Save(ctx)
	if err != nil {
		_ = storage.registry.Delete(harnessPath)
		return WorkflowDetail{}, s.mapWorkflowWriteError("create workflow", err)
	}

	item, err = s.getWorkflowWithStatusBindings(ctx, item.ID)
	if err != nil {
		_ = storage.registry.Delete(harnessPath)
		return WorkflowDetail{}, err
	}

	return mapWorkflowDetail(item, harnessContent), nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (WorkflowDetail, error) {
	if s.client == nil {
		return WorkflowDetail{}, ErrUnavailable
	}

	current, err := s.getWorkflowWithStatusBindings(ctx, input.WorkflowID)
	if err != nil {
		return WorkflowDetail{}, s.mapWorkflowReadError("get workflow for update", err)
	}
	storage, err := s.storageForProject(ctx, current.ProjectID)
	if err != nil {
		return WorkflowDetail{}, err
	}

	projectID := current.ProjectID
	nextAgentID := current.AgentID
	if input.AgentID.Set {
		nextAgentID = &input.AgentID.Value
	}
	if nextAgentID == nil {
		return WorkflowDetail{}, ErrAgentNotFound
	}
	if err := s.ensureAgentBelongsToProject(ctx, projectID, *nextAgentID); err != nil {
		return WorkflowDetail{}, err
	}
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
		nextHarnessPath = defaultHarnessPath(nextName)
	}
	if nextHarnessPath != current.HarnessPath {
		if err := s.ensureHarnessPathAvailable(ctx, projectID, storage, nextHarnessPath, current.ID); err != nil {
			return WorkflowDetail{}, err
		}
	}

	nextPickupStatusIDs := MustStatusBindingSet(statusIDsFromEdges(current.Edges.PickupStatuses)...)
	if input.PickupStatusIDs.Set {
		nextPickupStatusIDs = input.PickupStatusIDs.Value
	}
	if err := s.ensureStatusBindingsBelongToProject(ctx, projectID, nextPickupStatusIDs); err != nil {
		return WorkflowDetail{}, err
	}

	nextFinishStatusIDs := MustStatusBindingSet(statusIDsFromEdges(current.Edges.FinishStatuses)...)
	if input.FinishStatusIDs.Set {
		nextFinishStatusIDs = input.FinishStatusIDs.Value
	}
	if err := s.ensureStatusBindingsBelongToProject(ctx, projectID, nextFinishStatusIDs); err != nil {
		return WorkflowDetail{}, err
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
		content, readErr := storage.registry.Read(current.HarnessPath)
		if readErr != nil {
			return WorkflowDetail{}, fmt.Errorf("read workflow harness before move: %w", readErr)
		}
		if err := storage.registry.Write(nextHarnessPath, content); err != nil {
			return WorkflowDetail{}, err
		}
	}

	if !current.IsActive && nextIsActive {
		if err := s.runWorkflowHooks(ctx, storage, parsedHooks, workflowHookOnActivate, workflowHookRuntime{
			ProjectID:       current.ProjectID,
			WorkflowID:      current.ID,
			WorkflowName:    nextName,
			WorkflowVersion: current.Version,
		}); err != nil {
			if nextHarnessPath != current.HarnessPath {
				_ = storage.registry.Delete(nextHarnessPath)
			}
			return WorkflowDetail{}, err
		}
	}

	builder := s.client.Workflow.UpdateOneID(current.ID)
	if input.AgentID.Set {
		builder.SetAgentID(input.AgentID.Value)
	}
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
	if input.PickupStatusIDs.Set {
		builder.ClearPickupStatuses()
		builder.AddPickupStatusIDs(nextPickupStatusIDs.IDs()...)
	}
	if input.FinishStatusIDs.Set {
		builder.ClearFinishStatuses()
		builder.AddFinishStatusIDs(nextFinishStatusIDs.IDs()...)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		if nextHarnessPath != current.HarnessPath {
			_ = storage.registry.Delete(nextHarnessPath)
		}
		return WorkflowDetail{}, s.mapWorkflowWriteError("update workflow", err)
	}

	if nextHarnessPath != current.HarnessPath {
		if err := storage.registry.Delete(current.HarnessPath); err != nil {
			s.logger.Error("delete old workflow harness after move", "error", err, "path", current.HarnessPath)
		}
	}

	item, err = s.getWorkflowWithStatusBindings(ctx, item.ID)
	if err != nil {
		return WorkflowDetail{}, err
	}

	content, err := storage.registry.Read(item.HarnessPath)
	if err != nil {
		return WorkflowDetail{}, fmt.Errorf("read workflow harness after update: %w", err)
	}

	return mapWorkflowDetail(item, content), nil
}

func (s *Service) Delete(ctx context.Context, workflowID uuid.UUID) (Workflow, error) {
	if s.client == nil {
		return Workflow{}, ErrUnavailable
	}

	current, err := s.getWorkflowWithStatusBindings(ctx, workflowID)
	if err != nil {
		return Workflow{}, s.mapWorkflowReadError("get workflow for delete", err)
	}
	storage, err := s.storageForProject(ctx, current.ProjectID)
	if err != nil {
		return Workflow{}, err
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

	if err := storage.registry.Delete(current.HarnessPath); err != nil {
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
	storage, err := s.storageForProject(ctx, item.ProjectID)
	if err != nil {
		return HarnessDocument{}, err
	}

	content, err := storage.registry.Read(item.HarnessPath)
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
	storage, err := s.storageForProject(ctx, item.ProjectID)
	if err != nil {
		return HarnessDocument{}, err
	}
	parsedHooks, err := validateConfiguredHooks(item.Hooks)
	if err != nil {
		return HarnessDocument{}, err
	}

	previousContent, err := storage.registry.Read(item.HarnessPath)
	if err != nil {
		return HarnessDocument{}, fmt.Errorf("read workflow harness before update: %w", err)
	}

	if err := storage.registry.Write(item.HarnessPath, input.Content); err != nil {
		return HarnessDocument{}, err
	}

	if item.IsActive {
		if err := s.runWorkflowHooks(ctx, storage, parsedHooks, workflowHookOnReload, workflowHookRuntime{
			ProjectID:       item.ProjectID,
			WorkflowID:      item.ID,
			WorkflowName:    item.Name,
			WorkflowVersion: item.Version + 1,
		}); err != nil {
			if restoreErr := storage.registry.Write(item.HarnessPath, previousContent); restoreErr != nil {
				s.logger.Error("restore workflow harness after reload hook failure", "error", restoreErr, "workflow_id", item.ID, "path", item.HarnessPath)
			}
			return HarnessDocument{}, err
		}
	}

	updated, err := s.client.Workflow.UpdateOneID(item.ID).
		SetVersion(item.Version + 1).
		Save(ctx)
	if err != nil {
		if restoreErr := storage.registry.Write(item.HarnessPath, previousContent); restoreErr != nil {
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

func (s *Service) ensureStatusBindingsBelongToProject(ctx context.Context, projectID uuid.UUID, statusIDs StatusBindingSet) error {
	exists, err := s.client.TicketStatus.Query().
		Where(
			entticketstatus.IDIn(statusIDs.IDs()...),
			entticketstatus.ProjectIDEQ(projectID),
		).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check workflow status existence: %w", err)
	}
	if exists != statusIDs.Len() {
		return ErrStatusNotFound
	}

	return nil
}

func (s *Service) ensureAgentBelongsToProject(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID) error {
	exists, err := s.client.Agent.Query().
		Where(
			entagent.ProjectIDEQ(projectID),
			entagent.IDEQ(agentID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check workflow agent existence: %w", err)
	}
	if !exists {
		return ErrAgentNotFound
	}

	return nil
}

func (s *Service) resolveCreateHarnessPath(name string, rawPath *string) (string, error) {
	if rawPath != nil {
		return normalizeHarnessPath(*rawPath)
	}

	return defaultHarnessPath(name), nil
}

func (s *Service) ensureHarnessPathAvailable(
	ctx context.Context,
	projectID uuid.UUID,
	storage *projectStorage,
	harnessPath string,
	excludeWorkflowID uuid.UUID,
) error {
	query := s.client.Workflow.Query().Where(
		entworkflow.ProjectIDEQ(projectID),
		entworkflow.HarnessPathEQ(harnessPath),
	)
	if excludeWorkflowID != uuid.Nil {
		query = query.Where(entworkflow.IDNEQ(excludeWorkflowID))
	}
	exists, err := query.Exist(ctx)
	if err != nil {
		return fmt.Errorf("check workflow harness path uniqueness: %w", err)
	}
	if exists || storage.registry.Exists(harnessPath) {
		return ErrWorkflowConflict
	}

	return nil
}

func (s *Service) resolveHarnessContent(
	ctx context.Context,
	name string,
	workflowType entworkflow.Type,
	pickupStatusIDs StatusBindingSet,
	finishStatusIDs StatusBindingSet,
	rawContent string,
) (string, error) {
	if strings.TrimSpace(rawContent) != "" {
		if err := validateHarnessForSave(rawContent); err != nil {
			return "", err
		}
		return rawContent, nil
	}

	pickupStatuses, err := s.client.TicketStatus.Query().
		Where(entticketstatus.IDIn(pickupStatusIDs.IDs()...)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return "", s.mapWorkflowReadError("get pickup statuses for harness template", err)
	}
	finishStatuses, err := s.client.TicketStatus.Query().
		Where(entticketstatus.IDIn(finishStatusIDs.IDs()...)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return "", s.mapWorkflowReadError("get finish statuses for harness template", err)
	}

	content := defaultHarnessContent(
		name,
		workflowType,
		statusNamesFromEdges(pickupStatuses),
		statusNamesFromEdges(finishStatuses),
	)
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
	storage, err := s.storageForProject(ctx, event.ProjectID)
	if err != nil {
		s.logger.Error("resolve project storage for harness reload", "error", err, "project_id", event.ProjectID, "path", event.RelativePath)
		return
	}

	item, err := s.client.Workflow.Query().
		Where(
			entworkflow.ProjectIDEQ(event.ProjectID),
			entworkflow.HarnessPathEQ(event.RelativePath),
		).
		Only(ctx)
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
		if err := s.runWorkflowHooks(ctx, storage, parsedHooks, workflowHookOnReload, workflowHookRuntime{
			ProjectID:       item.ProjectID,
			WorkflowID:      item.ID,
			WorkflowName:    item.Name,
			WorkflowVersion: item.Version + 1,
		}); err != nil {
			s.logger.Error("workflow reload hook blocked harness reload", "error", err, "workflow_id", item.ID, "path", event.RelativePath)
			if event.PreviousContent != "" {
				if restoreErr := storage.registry.Write(event.RelativePath, event.PreviousContent); restoreErr != nil {
					s.logger.Error("restore workflow harness after blocked reload", "error", restoreErr, "workflow_id", item.ID, "path", event.RelativePath)
				}
			}
			return
		}
	}

	if _, err := s.client.Workflow.UpdateOneID(item.ID).SetVersion(item.Version + 1).Save(ctx); err != nil {
		s.logger.Error("bump workflow version after harness reload", "error", err, "workflow_id", item.ID, "path", event.RelativePath)
		if event.PreviousContent != "" {
			if restoreErr := storage.registry.Write(event.RelativePath, event.PreviousContent); restoreErr != nil {
				s.logger.Error("restore workflow harness after failed version bump", "error", restoreErr, "workflow_id", item.ID, "path", event.RelativePath)
			}
		}
		return
	}

	s.logger.Info("workflow harness reloaded", "workflow_id", item.ID, "path", event.RelativePath)
}

func (s *Service) runWorkflowHooks(
	ctx context.Context,
	storage *projectStorage,
	hooks workflowHooksConfig,
	hookName workflowHookName,
	runtime workflowHookRuntime,
) error {
	if s == nil || storage == nil || storage.hookExecutor == nil {
		return nil
	}

	switch hookName {
	case workflowHookOnActivate:
		return storage.hookExecutor.RunAll(ctx, hookName, hooks.OnActivate, runtime)
	case workflowHookOnReload:
		return storage.hookExecutor.RunAll(ctx, hookName, hooks.OnReload, runtime)
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

func (s *Service) getWorkflowWithStatusBindings(ctx context.Context, workflowID uuid.UUID) (*ent.Workflow, error) {
	item, err := s.client.Workflow.Query().
		Where(entworkflow.IDEQ(workflowID)).
		WithPickupStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		WithFinishStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		Only(ctx)
	if err != nil {
		return nil, s.mapWorkflowReadError("get workflow with status bindings", err)
	}

	return item, nil
}

func mapWorkflow(item *ent.Workflow) Workflow {
	return Workflow{
		ID:                  item.ID,
		ProjectID:           item.ProjectID,
		AgentID:             item.AgentID,
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
		PickupStatusIDs:     statusIDsFromEdges(item.Edges.PickupStatuses),
		FinishStatusIDs:     statusIDsFromEdges(item.Edges.FinishStatuses),
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

func statusIDsFromEdges(statuses []*ent.TicketStatus) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(statuses))
	for _, status := range statuses {
		ids = append(ids, status.ID)
	}
	return ids
}

func statusNamesFromEdges(statuses []*ent.TicketStatus) []string {
	names := make([]string, 0, len(statuses))
	for _, status := range statuses {
		names = append(names, status.Name)
	}
	return names
}

func defaultHarnessPath(workflowName string) string {
	slug := slugify(workflowName)
	if slug == "" {
		slug = "workflow"
	}

	return filepath.ToSlash(filepath.Join(".openase", "harnesses", slug+".md"))
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

func defaultHarnessContent(name string, workflowType entworkflow.Type, pickupStatusNames []string, finishStatusNames []string) string {
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("workflow:\n")
	_, _ = fmt.Fprintf(&builder, "  name: %q\n", name)
	_, _ = fmt.Fprintf(&builder, "  type: %q\n", workflowType.String())
	builder.WriteString("---\n\n")
	builder.WriteString("# ")
	builder.WriteString(name)
	builder.WriteString("\n\n")
	if len(pickupStatusNames) > 0 {
		builder.WriteString("Pickup statuses: ")
		builder.WriteString(strings.Join(pickupStatusNames, ", "))
		builder.WriteString("\n")
	}
	if len(finishStatusNames) > 0 {
		builder.WriteString("Finish statuses: ")
		builder.WriteString(strings.Join(finishStatusNames, ", "))
		builder.WriteString("\n")
	}
	if len(pickupStatusNames) > 0 || len(finishStatusNames) > 0 {
		builder.WriteString("\n")
	}
	builder.WriteString("Describe the role, constraints, and expected outputs for this workflow.\n")

	return builder.String()
}

func slugify(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	trimmed = nonAlphaNumericPattern.ReplaceAllString(trimmed, "-")
	return strings.Trim(trimmed, "-")
}

func DetectRepoRoot(start string) (string, error) {
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
