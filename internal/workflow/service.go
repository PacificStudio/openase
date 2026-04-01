package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entskill "github.com/BetterAndBetterII/openase/ent/skill"
	entskillversion "github.com/BetterAndBetterII/openase/ent/skillversion"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	entworkflowskillbinding "github.com/BetterAndBetterII/openase/ent/workflowskillbinding"
	entworkflowversion "github.com/BetterAndBetterII/openase/ent/workflowversion"
	"github.com/BetterAndBetterII/openase/internal/builtin"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	"github.com/google/uuid"
)

var (
	ErrUnavailable         = errors.New("workflow service unavailable")
	ErrProjectNotFound     = errors.New("project not found")
	ErrWorkflowNotFound    = errors.New("workflow not found")
	ErrStatusNotFound      = errors.New("workflow status not found in project")
	ErrAgentNotFound       = errors.New("workflow agent not found in project")
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

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func sanitizeHarnessContent(content string) (string, error) {
	if err := validateHarnessForSave(content); err != nil {
		return "", err
	}
	sanitized, err := setHarnessSkills(content, nil)
	if err != nil {
		return "", err
	}
	if err := validateHarnessForSave(sanitized); err != nil {
		return "", err
	}
	return sanitized, nil
}

func projectHarnessContent(content string, skillNames []string) (string, error) {
	projected, err := setHarnessSkills(content, skillNames)
	if err != nil {
		return "", err
	}
	if err := validateHarnessForSave(projected); err != nil {
		return "", err
	}
	return projected, nil
}

func (s *Service) currentWorkflowVersion(ctx context.Context, workflowID uuid.UUID) (*ent.WorkflowVersion, error) {
	item, err := s.client.WorkflowVersion.Query().
		Where(entworkflowversion.WorkflowIDEQ(workflowID)).
		Order(ent.Desc(entworkflowversion.FieldVersion)).
		First(ctx)
	if err != nil {
		return nil, s.mapWorkflowReadError("get workflow version", err)
	}
	return item, nil
}

func (s *Service) projectedWorkflowHarness(ctx context.Context, workflowItem *ent.Workflow) (string, error) {
	version, err := s.currentWorkflowVersion(ctx, workflowItem.ID)
	if err != nil {
		return "", err
	}
	boundSkills, err := s.listWorkflowBoundSkillNames(ctx, workflowItem.ID, false)
	if err != nil {
		return "", err
	}
	return projectHarnessContent(version.ContentMarkdown, boundSkills)
}

func (s *Service) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID) ([]VersionSummary, error) {
	if s == nil || s.client == nil {
		return nil, ErrUnavailable
	}

	if _, err := s.currentWorkflowVersion(ctx, workflowID); err != nil {
		return nil, err
	}

	items, err := s.client.WorkflowVersion.Query().
		Where(entworkflowversion.WorkflowIDEQ(workflowID)).
		Order(ent.Desc(entworkflowversion.FieldVersion)).
		All(ctx)
	if err != nil {
		return nil, s.mapWorkflowReadError("list workflow versions", err)
	}

	result := make([]VersionSummary, 0, len(items))
	for _, item := range items {
		result = append(result, VersionSummary{
			ID:        item.ID,
			Version:   item.Version,
			CreatedBy: item.CreatedBy,
			CreatedAt: item.CreatedAt.UTC(),
		})
	}
	return result, nil
}

func (s *Service) listWorkflowBoundSkillNames(ctx context.Context, workflowID uuid.UUID, enabledOnly bool) ([]string, error) {
	query := s.client.WorkflowSkillBinding.Query().
		Where(entworkflowskillbinding.WorkflowIDEQ(workflowID)).
		WithSkill()
	bindings, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflow skill bindings: %w", err)
	}

	names := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		if binding.Edges.Skill == nil {
			continue
		}
		if binding.Edges.Skill.ArchivedAt != nil {
			continue
		}
		if enabledOnly && !binding.Edges.Skill.IsEnabled {
			continue
		}
		names = append(names, binding.Edges.Skill.Name)
	}
	sort.Strings(names)
	return names, nil
}

func skillDescriptionFromContent(content string) (string, error) {
	document, body, err := parseSkillDocument(content)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrSkillInvalid, err)
	}
	title := parseSkillTitle(body)
	if strings.TrimSpace(title) != "" {
		return title, nil
	}
	description := strings.TrimSpace(document.Description)
	if description == "" {
		return "", fmt.Errorf("%w: description must not be empty", ErrSkillInvalid)
	}
	return description, nil
}

func (s *Service) ensureBuiltinSkills(ctx context.Context, projectID uuid.UUID) error {
	if s == nil || s.client == nil {
		return ErrUnavailable
	}
	s.builtinSkillsMu.Lock()
	defer s.builtinSkillsMu.Unlock()

	existing, err := s.client.Skill.Query().
		Where(entskill.ProjectIDEQ(projectID)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list existing skills: %w", err)
	}
	byName := make(map[string]*ent.Skill, len(existing))
	for _, item := range existing {
		byName[item.Name] = item
	}

	for _, template := range builtin.Skills() {
		if _, ok := byName[template.Name]; ok {
			continue
		}

		now := time.Now().UTC()
		tx, err := s.client.Tx(ctx)
		if err != nil {
			return fmt.Errorf("start builtin skill tx: %w", err)
		}

		content, err := ensureSkillContent(template.Name, template.Content, template.Description)
		if err != nil {
			rollback(tx)
			return err
		}
		bundle, err := parseSkillBundle(template.Name, []SkillBundleFileInput{
			{
				Path:      "SKILL.md",
				Content:   []byte(content),
				MediaType: "text/markdown; charset=utf-8",
			},
		})
		if err != nil {
			rollback(tx)
			return err
		}

		skillItem, err := tx.Skill.Create().
			SetProjectID(projectID).
			SetName(template.Name).
			SetDescription(bundle.Description).
			SetIsBuiltin(true).
			SetIsEnabled(true).
			SetCreatedBy("builtin:openase").
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		if err != nil {
			rollback(tx)
			if ent.IsConstraintError(err) {
				continue
			}
			return fmt.Errorf("create builtin skill %s: %w", template.Name, err)
		}

		versionItem, err := s.storeSkillBundleVersion(ctx, tx, skillItem.ID, 1, bundle, "builtin:openase", now)
		if err != nil {
			rollback(tx)
			return fmt.Errorf("create builtin skill version %s: %w", template.Name, err)
		}

		if _, err := tx.Skill.UpdateOneID(skillItem.ID).
			SetCurrentVersionID(versionItem.ID).
			Save(ctx); err != nil {
			rollback(tx)
			return fmt.Errorf("update builtin skill current version %s: %w", template.Name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit builtin skill %s: %w", template.Name, err)
		}
	}

	return nil
}

func (s *Service) skillByName(ctx context.Context, projectID uuid.UUID, name string) (*ent.Skill, error) {
	item, err := s.client.Skill.Query().
		Where(
			entskill.ProjectIDEQ(projectID),
			entskill.NameEQ(name),
			entskill.ArchivedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrSkillNotFound
		}
		return nil, fmt.Errorf("get skill by name: %w", err)
	}
	return item, nil
}

func (s *Service) currentSkillVersion(ctx context.Context, skillID uuid.UUID, requiredVersionID *uuid.UUID) (*ent.SkillVersion, error) {
	query := s.client.SkillVersion.Query()
	if requiredVersionID != nil && *requiredVersionID != uuid.Nil {
		item, err := query.Where(entskillversion.IDEQ(*requiredVersionID)).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, ErrSkillNotFound
			}
			return nil, fmt.Errorf("get required skill version: %w", err)
		}
		return item, nil
	}

	item, err := query.
		Where(entskillversion.SkillIDEQ(skillID)).
		Order(ent.Desc(entskillversion.FieldVersion)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrSkillNotFound
		}
		return nil, fmt.Errorf("get current skill version: %w", err)
	}
	return item, nil
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

type VersionSummary struct {
	ID        uuid.UUID `json:"id"`
	Version   int       `json:"version"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
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
	client            *ent.Client
	logger            *slog.Logger
	repoRoot          string
	builtinSkillsMu   sync.Mutex
	workflowVersionMu sync.Mutex
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
	}

	return service, nil
}

func (s *Service) Close() error {
	return nil
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
	content, err := s.projectedWorkflowHarness(ctx, item)
	if err != nil {
		return WorkflowDetail{}, err
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
	harnessPath, err := s.resolveCreateHarnessPath(input.Name, input.HarnessPath)
	if err != nil {
		return WorkflowDetail{}, err
	}
	if err := s.ensureHarnessPathAvailable(ctx, input.ProjectID, harnessPath, uuid.Nil); err != nil {
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
	sanitizedHarnessContent, err := sanitizeHarnessContent(harnessContent)
	if err != nil {
		return WorkflowDetail{}, err
	}

	workflowID := uuid.New()
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return WorkflowDetail{}, fmt.Errorf("start workflow create tx: %w", err)
	}
	defer rollback(tx)

	item, err := tx.Workflow.Create().
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
		SetVersion(1).
		SetIsActive(input.IsActive).
		AddPickupStatusIDs(pickupStatusIDs.IDs()...).
		AddFinishStatusIDs(finishStatusIDs.IDs()...).
		Save(ctx)
	if err != nil {
		return WorkflowDetail{}, s.mapWorkflowWriteError("create workflow", err)
	}

	versionItem, err := tx.WorkflowVersion.Create().
		SetWorkflowID(workflowID).
		SetVersion(1).
		SetContentMarkdown(sanitizedHarnessContent).
		SetContentHash(contentHash(sanitizedHarnessContent)).
		Save(ctx)
	if err != nil {
		return WorkflowDetail{}, s.mapWorkflowWriteError("create workflow version", err)
	}

	if _, err := tx.Workflow.UpdateOneID(workflowID).
		SetCurrentVersionID(versionItem.ID).
		Save(ctx); err != nil {
		return WorkflowDetail{}, s.mapWorkflowWriteError("set workflow current version", err)
	}

	if input.IsActive {
		if err := s.runWorkflowHooks(ctx, input.ProjectID, parsedHooks, workflowHookOnActivate, workflowHookRuntime{
			ProjectID:       input.ProjectID,
			WorkflowID:      workflowID,
			WorkflowName:    input.Name,
			WorkflowVersion: 1,
		}); err != nil {
			return WorkflowDetail{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return WorkflowDetail{}, fmt.Errorf("commit workflow create tx: %w", err)
	}

	item, err = s.getWorkflowWithStatusBindings(ctx, item.ID)
	if err != nil {
		return WorkflowDetail{}, err
	}

	projectedContent, err := s.projectedWorkflowHarness(ctx, item)
	if err != nil {
		return WorkflowDetail{}, err
	}

	return mapWorkflowDetail(item, projectedContent), nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (WorkflowDetail, error) {
	if s.client == nil {
		return WorkflowDetail{}, ErrUnavailable
	}

	current, err := s.getWorkflowWithStatusBindings(ctx, input.WorkflowID)
	if err != nil {
		return WorkflowDetail{}, s.mapWorkflowReadError("get workflow for update", err)
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
		if err := s.ensureHarnessPathAvailable(ctx, projectID, nextHarnessPath, current.ID); err != nil {
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

	if !current.IsActive && nextIsActive {
		if err := s.runWorkflowHooks(ctx, projectID, parsedHooks, workflowHookOnActivate, workflowHookRuntime{
			ProjectID:       current.ProjectID,
			WorkflowID:      current.ID,
			WorkflowName:    nextName,
			WorkflowVersion: current.Version,
		}); err != nil {
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
		return WorkflowDetail{}, s.mapWorkflowWriteError("update workflow", err)
	}

	item, err = s.getWorkflowWithStatusBindings(ctx, item.ID)
	if err != nil {
		return WorkflowDetail{}, err
	}

	content, err := s.projectedWorkflowHarness(ctx, item)
	if err != nil {
		return WorkflowDetail{}, err
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
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Workflow{}, fmt.Errorf("start workflow delete tx: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.WorkflowSkillBinding.Delete().
		Where(entworkflowskillbinding.WorkflowIDEQ(current.ID)).
		Exec(ctx); err != nil {
		return Workflow{}, fmt.Errorf("delete workflow skill bindings: %w", err)
	}

	if _, err := tx.WorkflowVersion.Delete().
		Where(entworkflowversion.WorkflowIDEQ(current.ID)).
		Exec(ctx); err != nil {
		return Workflow{}, fmt.Errorf("delete workflow versions: %w", err)
	}

	if err := tx.Workflow.DeleteOneID(current.ID).Exec(ctx); err != nil {
		return Workflow{}, s.mapWorkflowWriteError("delete workflow", err)
	}

	if err := tx.Commit(); err != nil {
		return Workflow{}, fmt.Errorf("commit workflow delete tx: %w", err)
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
	content, err := s.projectedWorkflowHarness(ctx, item)
	if err != nil {
		return HarnessDocument{}, err
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

	previousVersion, err := s.currentWorkflowVersion(ctx, item.ID)
	if err != nil {
		return HarnessDocument{}, err
	}
	sanitizedContent, err := sanitizeHarnessContent(input.Content)
	if err != nil {
		return HarnessDocument{}, err
	}
	if sanitizedContent == previousVersion.ContentMarkdown {
		content, err := s.projectedWorkflowHarness(ctx, item)
		if err != nil {
			return HarnessDocument{}, err
		}
		return HarnessDocument{
			WorkflowID: item.ID,
			Path:       item.HarnessPath,
			Content:    content,
			Version:    item.Version,
		}, nil
	}

	if item.IsActive {
		if err := s.runWorkflowHooks(ctx, item.ProjectID, parsedHooks, workflowHookOnReload, workflowHookRuntime{
			ProjectID:       item.ProjectID,
			WorkflowID:      item.ID,
			WorkflowName:    item.Name,
			WorkflowVersion: item.Version + 1,
		}); err != nil {
			return HarnessDocument{}, err
		}
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return HarnessDocument{}, fmt.Errorf("start workflow harness update tx: %w", err)
	}
	defer rollback(tx)

	versionItem, err := tx.WorkflowVersion.Create().
		SetWorkflowID(item.ID).
		SetVersion(item.Version + 1).
		SetContentMarkdown(sanitizedContent).
		SetContentHash(contentHash(sanitizedContent)).
		Save(ctx)
	if err != nil {
		return HarnessDocument{}, s.mapWorkflowWriteError("create workflow harness version", err)
	}

	updated, err := tx.Workflow.UpdateOneID(item.ID).
		SetVersion(item.Version + 1).
		SetCurrentVersionID(versionItem.ID).
		Save(ctx)
	if err != nil {
		return HarnessDocument{}, s.mapWorkflowWriteError("update workflow harness version", err)
	}
	if err := tx.Commit(); err != nil {
		return HarnessDocument{}, fmt.Errorf("commit workflow harness update tx: %w", err)
	}

	content, err := s.projectedWorkflowHarness(ctx, updated)
	if err != nil {
		return HarnessDocument{}, err
	}
	return HarnessDocument{
		WorkflowID: updated.ID,
		Path:       updated.HarnessPath,
		Content:    content,
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
	count, err := s.client.TicketStatus.Query().
		Where(
			entticketstatus.IDIn(statusIDs.IDs()...),
			entticketstatus.ProjectIDEQ(projectID),
		).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check workflow status existence: %w", err)
	}
	if count != statusIDs.Len() {
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
	if exists {
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

func (s *Service) runWorkflowHooks(
	ctx context.Context,
	projectID uuid.UUID,
	hooks workflowHooksConfig,
	hookName workflowHookName,
	runtime workflowHookRuntime,
) error {
	if s == nil {
		return nil
	}
	executor := s.hookExecutorForProject(ctx, projectID)
	if executor == nil {
		return nil
	}

	switch hookName {
	case workflowHookOnActivate:
		return executor.RunAll(ctx, hookName, hooks.OnActivate, runtime)
	case workflowHookOnReload:
		return executor.RunAll(ctx, hookName, hooks.OnReload, runtime)
	default:
		return nil
	}
}

func (s *Service) hookExecutorForProject(_ context.Context, _ uuid.UUID) *workflowHookExecutor {
	if s == nil {
		return nil
	}
	logger := s.logger
	if logger == nil {
		logger = slog.Default()
	}
	repoRoot := strings.TrimSpace(s.repoRoot)
	if repoRoot == "" {
		return nil
	}
	return newWorkflowHookExecutor(repoRoot, logger)
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

func (s *Service) ResolveRuntimeSnapshot(ctx context.Context, workflowID uuid.UUID) (RuntimeSnapshot, error) {
	if s == nil || s.client == nil {
		return RuntimeSnapshot{}, ErrUnavailable
	}

	workflowItem, err := s.client.Workflow.Query().
		Where(entworkflow.IDEQ(workflowID)).
		Only(ctx)
	if err != nil {
		return RuntimeSnapshot{}, s.mapWorkflowReadError("get workflow for runtime snapshot", err)
	}
	if err := s.ensureBuiltinSkills(ctx, workflowItem.ProjectID); err != nil {
		return RuntimeSnapshot{}, err
	}

	workflowVersion, err := s.runtimeWorkflowVersion(ctx, workflowItem)
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	skills, err := s.runtimeSkillSnapshots(ctx, workflowItem.ProjectID, workflowItem.ID)
	if err != nil {
		return RuntimeSnapshot{}, err
	}

	projectedContent, err := projectHarnessContent(workflowVersion.ContentMarkdown, runtimeSkillNames(skills))
	if err != nil {
		return RuntimeSnapshot{}, err
	}

	return RuntimeSnapshot{
		Workflow: RuntimeWorkflowSnapshot{
			WorkflowID: workflowItem.ID,
			VersionID:  workflowVersion.ID,
			Version:    workflowVersion.Version,
			Path:       workflowItem.HarnessPath,
			Content:    projectedContent,
		},
		Skills: skills,
	}, nil
}

func (s *Service) ResolveRecordedRuntimeSnapshot(ctx context.Context, input ResolveRecordedRuntimeSnapshotInput) (RuntimeSnapshot, error) {
	if s == nil || s.client == nil {
		return RuntimeSnapshot{}, ErrUnavailable
	}

	workflowItem, err := s.client.Workflow.Query().
		Where(entworkflow.IDEQ(input.WorkflowID)).
		Only(ctx)
	if err != nil {
		return RuntimeSnapshot{}, s.mapWorkflowReadError("get workflow for recorded runtime snapshot", err)
	}

	workflowVersion, err := s.recordedWorkflowVersion(ctx, workflowItem, input.WorkflowVersionID)
	if err != nil {
		return RuntimeSnapshot{}, err
	}

	var skills []RuntimeSkillSnapshot
	if len(input.SkillVersionIDs) == 0 {
		skills, err = s.runtimeSkillSnapshots(ctx, workflowItem.ProjectID, workflowItem.ID)
	} else {
		skills, err = s.recordedSkillSnapshots(ctx, workflowItem.ProjectID, input.SkillVersionIDs)
	}
	if err != nil {
		return RuntimeSnapshot{}, err
	}

	projectedContent, err := projectHarnessContent(workflowVersion.ContentMarkdown, runtimeSkillNames(skills))
	if err != nil {
		return RuntimeSnapshot{}, err
	}

	return RuntimeSnapshot{
		Workflow: RuntimeWorkflowSnapshot{
			WorkflowID: workflowItem.ID,
			VersionID:  workflowVersion.ID,
			Version:    workflowVersion.Version,
			Path:       workflowItem.HarnessPath,
			Content:    projectedContent,
		},
		Skills: skills,
	}, nil
}

func (s *Service) runtimeWorkflowVersion(ctx context.Context, workflowItem *ent.Workflow) (*ent.WorkflowVersion, error) {
	if workflowItem == nil {
		return nil, ErrWorkflowNotFound
	}

	if workflowItem.CurrentVersionID != nil && *workflowItem.CurrentVersionID != uuid.Nil {
		item, err := s.client.WorkflowVersion.Query().
			Where(
				entworkflowversion.IDEQ(*workflowItem.CurrentVersionID),
				entworkflowversion.WorkflowIDEQ(workflowItem.ID),
			).
			Only(ctx)
		if err == nil {
			return item, nil
		}
		if !ent.IsNotFound(err) {
			return nil, fmt.Errorf("get workflow current version: %w", err)
		}
	}

	item, err := s.client.WorkflowVersion.Query().
		Where(entworkflowversion.WorkflowIDEQ(workflowItem.ID)).
		Order(ent.Desc(entworkflowversion.FieldVersion)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: workflow %s has no published versions", ErrWorkflowNotFound, workflowItem.ID)
		}
		return nil, fmt.Errorf("get workflow runtime version: %w", err)
	}
	return item, nil
}

func (s *Service) recordedWorkflowVersion(
	ctx context.Context,
	workflowItem *ent.Workflow,
	workflowVersionID *uuid.UUID,
) (*ent.WorkflowVersion, error) {
	if workflowVersionID == nil || *workflowVersionID == uuid.Nil {
		return s.runtimeWorkflowVersion(ctx, workflowItem)
	}

	item, err := s.client.WorkflowVersion.Query().
		Where(
			entworkflowversion.IDEQ(*workflowVersionID),
			entworkflowversion.WorkflowIDEQ(workflowItem.ID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: recorded workflow version %s not found for workflow %s", ErrWorkflowNotFound, *workflowVersionID, workflowItem.ID)
		}
		return nil, fmt.Errorf("get recorded workflow version: %w", err)
	}
	return item, nil
}

func (s *Service) runtimeSkillSnapshots(
	ctx context.Context,
	projectID uuid.UUID,
	workflowID uuid.UUID,
) ([]RuntimeSkillSnapshot, error) {
	bindings, err := s.client.WorkflowSkillBinding.Query().
		Where(entworkflowskillbinding.WorkflowIDEQ(workflowID)).
		Order(ent.Asc(entworkflowskillbinding.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list runtime workflow skill bindings: %w", err)
	}

	snapshots := make([]RuntimeSkillSnapshot, 0, len(bindings)+1)
	seen := map[uuid.UUID]struct{}{}
	for _, binding := range bindings {
		skillItem, err := s.skillByID(ctx, binding.SkillID)
		if err != nil {
			if err == ErrSkillNotFound {
				continue
			}
			return nil, err
		}
		if skillItem.ProjectID != projectID || !skillItem.IsEnabled || skillItem.ArchivedAt != nil {
			continue
		}

		versionItem, err := s.runtimeSkillVersion(ctx, skillItem, binding.RequiredVersionID)
		if err != nil {
			return nil, err
		}
		files, err := s.runtimeSkillFiles(ctx, versionItem.ID)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, RuntimeSkillSnapshot{
			SkillID:    skillItem.ID,
			Name:       skillItem.Name,
			VersionID:  versionItem.ID,
			Version:    versionItem.Version,
			Content:    versionItem.ContentMarkdown,
			Files:      files,
			IsRequired: binding.RequiredVersionID != nil && *binding.RequiredVersionID != uuid.Nil,
		})
		seen[skillItem.ID] = struct{}{}
	}

	platformSkill, err := s.skillByName(ctx, projectID, "openase-platform")
	if err != nil && err != ErrSkillNotFound {
		return nil, err
	}
	if err == nil && platformSkill.IsEnabled && platformSkill.ArchivedAt == nil {
		if _, ok := seen[platformSkill.ID]; !ok {
			versionItem, versionErr := s.runtimeSkillVersion(ctx, platformSkill, nil)
			if versionErr != nil {
				return nil, versionErr
			}
			files, filesErr := s.runtimeSkillFiles(ctx, versionItem.ID)
			if filesErr != nil {
				return nil, filesErr
			}
			snapshots = append(snapshots, RuntimeSkillSnapshot{
				SkillID:   platformSkill.ID,
				Name:      platformSkill.Name,
				VersionID: versionItem.ID,
				Version:   versionItem.Version,
				Content:   versionItem.ContentMarkdown,
				Files:     files,
			})
		}
	}

	sort.SliceStable(snapshots, func(i int, j int) bool {
		return snapshots[i].Name < snapshots[j].Name
	})
	return snapshots, nil
}

func (s *Service) skillByID(ctx context.Context, skillID uuid.UUID) (*ent.Skill, error) {
	item, err := s.client.Skill.Query().
		Where(
			entskill.IDEQ(skillID),
			entskill.ArchivedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrSkillNotFound
		}
		return nil, fmt.Errorf("get skill by id: %w", err)
	}
	return item, nil
}

func (s *Service) runtimeSkillVersion(ctx context.Context, skillItem *ent.Skill, requiredVersionID *uuid.UUID) (*ent.SkillVersion, error) {
	if skillItem == nil {
		return nil, ErrSkillNotFound
	}

	if requiredVersionID != nil && *requiredVersionID != uuid.Nil {
		item, err := s.client.SkillVersion.Query().
			Where(
				entskillversion.IDEQ(*requiredVersionID),
				entskillversion.SkillIDEQ(skillItem.ID),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, fmt.Errorf("%w: required skill version %s not found for skill %s", ErrSkillNotFound, *requiredVersionID, skillItem.Name)
			}
			return nil, fmt.Errorf("get required skill version: %w", err)
		}
		return item, nil
	}

	if skillItem.CurrentVersionID != nil && *skillItem.CurrentVersionID != uuid.Nil {
		item, err := s.client.SkillVersion.Query().
			Where(
				entskillversion.IDEQ(*skillItem.CurrentVersionID),
				entskillversion.SkillIDEQ(skillItem.ID),
			).
			Only(ctx)
		if err == nil {
			return item, nil
		}
		if !ent.IsNotFound(err) {
			return nil, fmt.Errorf("get current skill version: %w", err)
		}
	}

	item, err := s.client.SkillVersion.Query().
		Where(entskillversion.SkillIDEQ(skillItem.ID)).
		Order(ent.Desc(entskillversion.FieldVersion)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: skill %s has no published versions", ErrSkillNotFound, skillItem.Name)
		}
		return nil, fmt.Errorf("get runtime skill version: %w", err)
	}
	return item, nil
}

func (s *Service) recordedSkillSnapshots(
	ctx context.Context,
	projectID uuid.UUID,
	versionIDs []uuid.UUID,
) ([]RuntimeSkillSnapshot, error) {
	ordered := make([]uuid.UUID, 0, len(versionIDs))
	seen := map[uuid.UUID]struct{}{}
	for _, id := range versionIDs {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}
	if len(ordered) == 0 {
		return nil, nil
	}

	snapshots := make([]RuntimeSkillSnapshot, 0, len(ordered))
	for _, versionID := range ordered {
		versionItem, err := s.client.SkillVersion.Query().
			Where(entskillversion.IDEQ(versionID)).
			WithSkill().
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, fmt.Errorf("%w: recorded skill version %s not found", ErrSkillNotFound, versionID)
			}
			return nil, fmt.Errorf("get recorded skill version: %w", err)
		}
		if versionItem.Edges.Skill == nil || versionItem.Edges.Skill.ProjectID != projectID {
			return nil, fmt.Errorf("%w: recorded skill version %s is outside project %s", ErrSkillInvalid, versionID, projectID)
		}
		files, err := s.runtimeSkillFiles(ctx, versionItem.ID)
		if err != nil {
			return nil, err
		}

		snapshots = append(snapshots, RuntimeSkillSnapshot{
			SkillID:   versionItem.Edges.Skill.ID,
			Name:      versionItem.Edges.Skill.Name,
			VersionID: versionItem.ID,
			Version:   versionItem.Version,
			Content:   versionItem.ContentMarkdown,
			Files:     files,
		})
	}

	sort.SliceStable(snapshots, func(i int, j int) bool {
		return snapshots[i].Name < snapshots[j].Name
	})
	return snapshots, nil
}

func (s *Service) runtimeSkillFiles(ctx context.Context, versionID uuid.UUID) ([]RuntimeSkillFileSnapshot, error) {
	files, err := s.skillVersionFiles(ctx, versionID)
	if err != nil {
		return nil, err
	}
	snapshots := make([]RuntimeSkillFileSnapshot, 0, len(files))
	for _, file := range files {
		snapshots = append(snapshots, RuntimeSkillFileSnapshot{
			Path:         file.Path,
			Content:      append([]byte(nil), file.Content...),
			IsExecutable: file.IsExecutable,
		})
	}
	return snapshots, nil
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
