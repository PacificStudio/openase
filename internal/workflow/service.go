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
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entprojectrepomirror "github.com/BetterAndBetterII/openase/ent/projectrepomirror"
	entskill "github.com/BetterAndBetterII/openase/ent/skill"
	entskillversion "github.com/BetterAndBetterII/openase/ent/skillversion"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	entworkflowskillbinding "github.com/BetterAndBetterII/openase/ent/workflowskillbinding"
	entworkflowversion "github.com/BetterAndBetterII/openase/ent/workflowversion"
	"github.com/BetterAndBetterII/openase/internal/builtin"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	projectrepomirrorsvc "github.com/BetterAndBetterII/openase/internal/projectrepomirror"
	git "github.com/go-git/go-git/v5"
	"github.com/google/uuid"
)

var (
	ErrUnavailable              = errors.New("workflow service unavailable")
	ErrProjectNotFound          = errors.New("project not found")
	ErrRepositoryMirrorNotReady = errors.New("workflow repository mirror is not ready")
	ErrWorkflowNotFound         = errors.New("workflow not found")
	ErrStatusNotFound           = errors.New("workflow status not found in project")
	ErrAgentNotFound            = errors.New("workflow agent not found in project")
	ErrWorkflowConflict         = errors.New("workflow conflict")
	ErrWorkflowInUse            = errors.New("workflow is still referenced by project or tickets")
	ErrHarnessInvalid           = errors.New("workflow harness is invalid")
	ErrHookConfigInvalid        = errors.New("workflow hook config is invalid")
	ErrWorkflowHookBlocked      = errors.New("workflow hook blocked the lifecycle operation")
)

var nonAlphaNumericPattern = regexp.MustCompile(`[^a-z0-9]+`)

type WorkflowRepositoryPrerequisiteKind string

const (
	WorkflowRepositoryPrerequisiteKindReady WorkflowRepositoryPrerequisiteKind = "ready"
)

type WorkflowRepositoryPrerequisiteAction string

const (
	WorkflowRepositoryPrerequisiteActionNone WorkflowRepositoryPrerequisiteAction = "none"
)

type WorkflowRepositoryPrerequisite struct {
	Kind      WorkflowRepositoryPrerequisiteKind
	RepoCount int
	Action    WorkflowRepositoryPrerequisiteAction
}

func (p WorkflowRepositoryPrerequisite) Ready() bool {
	return p.Kind == WorkflowRepositoryPrerequisiteKindReady
}

type WorkflowRepositoryPrerequisiteError struct {
	prerequisite WorkflowRepositoryPrerequisite
	cause        error
	message      string
}

func (e *WorkflowRepositoryPrerequisiteError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func (e *WorkflowRepositoryPrerequisiteError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *WorkflowRepositoryPrerequisiteError) Prerequisite() WorkflowRepositoryPrerequisite {
	if e == nil {
		return WorkflowRepositoryPrerequisite{}
	}
	return e.prerequisite
}

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
		if ent.IsNotFound(err) {
			seeded, seedErr := s.seedLegacyWorkflowVersion(ctx, workflowID)
			if seedErr == nil {
				return seeded, nil
			}
			return nil, seedErr
		}
		return nil, s.mapWorkflowReadError("get workflow version", err)
	}
	return item, nil
}

func (s *Service) seedLegacyWorkflowVersion(ctx context.Context, workflowID uuid.UUID) (*ent.WorkflowVersion, error) {
	if s == nil || s.client == nil {
		return nil, ErrUnavailable
	}
	s.workflowVersionMu.Lock()
	defer s.workflowVersionMu.Unlock()

	if existing, err := s.client.WorkflowVersion.Query().
		Where(entworkflowversion.WorkflowIDEQ(workflowID)).
		Order(ent.Desc(entworkflowversion.FieldVersion)).
		First(ctx); err == nil {
		return existing, nil
	}

	workflowItem, err := s.client.Workflow.Get(ctx, workflowID)
	if err != nil {
		return nil, s.mapWorkflowReadError("get workflow for legacy version import", err)
	}

	repoRoot := strings.TrimSpace(s.repoRoot)
	if repoRoot == "" {
		if resolvedRoot, resolveErr := s.projectStorageRoot(workflowItem.ProjectID); resolveErr == nil {
			repoRoot = strings.TrimSpace(resolvedRoot)
		}
	}
	if repoRoot == "" {
		return nil, ErrWorkflowNotFound
	}

	//nolint:gosec // legacy import only reads a workflow-owned harness path from the resolved repo root.
	contentBytes, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(workflowItem.HarnessPath)))
	if err != nil {
		return nil, s.mapWorkflowReadError("read legacy workflow harness", err)
	}
	content, err := sanitizeHarnessContent(string(contentBytes))
	if err != nil {
		return nil, err
	}

	versionNumber := workflowItem.Version
	if versionNumber <= 0 {
		versionNumber = 1
	}
	now := time.Now().UTC()
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("start workflow legacy version tx: %w", err)
	}
	defer rollback(tx)

	versionItem, err := tx.WorkflowVersion.Create().
		SetWorkflowID(workflowItem.ID).
		SetVersion(versionNumber).
		SetContentMarkdown(content).
		SetContentHash(contentHash(content)).
		SetCreatedBy("system:legacy-import").
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return s.client.WorkflowVersion.Query().
				Where(entworkflowversion.WorkflowIDEQ(workflowItem.ID)).
				Order(ent.Desc(entworkflowversion.FieldVersion)).
				First(ctx)
		}
		return nil, fmt.Errorf("create legacy workflow version: %w", err)
	}

	if workflowItem.CurrentVersionID == nil || *workflowItem.CurrentVersionID == uuid.Nil {
		if _, err := tx.Workflow.UpdateOneID(workflowItem.ID).
			SetCurrentVersionID(versionItem.ID).
			Save(ctx); err != nil {
			return nil, fmt.Errorf("set legacy workflow current version: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit workflow legacy version tx: %w", err)
	}

	return versionItem, nil
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
		description, err := skillDescriptionFromContent(content)
		if err != nil {
			rollback(tx)
			return err
		}

		skillItem, err := tx.Skill.Create().
			SetProjectID(projectID).
			SetName(template.Name).
			SetDescription(description).
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

		versionItem, err := tx.SkillVersion.Create().
			SetSkillID(skillItem.ID).
			SetVersion(1).
			SetContentMarkdown(content).
			SetContentHash(contentHash(content)).
			SetCreatedBy("builtin:openase").
			SetCreatedAt(now).
			Save(ctx)
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
	mirrors           *projectrepomirrorsvc.Service
	storageMu         sync.Mutex
	storages          map[uuid.UUID]*projectStorage
	builtinSkillsMu   sync.Mutex
	workflowVersionMu sync.Mutex
}

type workflowStorageUsage string

const (
	workflowStorageUsageRead  workflowStorageUsage = "read"
	workflowStorageUsageWrite workflowStorageUsage = "write"
)

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

func (s *Service) ConfigureMirrorService(service *projectrepomirrorsvc.Service) {
	if s == nil {
		return
	}
	s.mirrors = service
}

func newProjectStorage(projectID uuid.UUID, repoRoot string, service *Service) (*projectStorage, error) {
	harnessRoot := filepath.Join(repoRoot, ".openase", "harnesses")
	skillRoot := filepath.Join(repoRoot, ".openase", "skills")
	if err := ensureGitRepositoryExists(repoRoot); err != nil {
		return nil, fmt.Errorf("initialize project storage git repository: %w", err)
	}
	if err := ensureProjectBuiltinAssets(repoRoot); err != nil {
		return nil, fmt.Errorf("materialize built-in project assets: %w", err)
	}
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

func ensureGitRepositoryExists(repoRoot string) error {
	if _, err := git.PlainOpen(repoRoot); err == nil {
		return nil
	} else if err != git.ErrRepositoryNotExists {
		return err
	}

	if _, err := git.PlainInit(repoRoot, false); err != nil {
		return err
	}
	return nil
}

func (s *projectStorage) Close() error {
	if s == nil || s.registry == nil {
		return nil
	}

	return s.registry.Close()
}

func (s *Service) storageForProject(ctx context.Context, projectID uuid.UUID, usage workflowStorageUsage) (*projectStorage, error) {
	if err := s.ensureProjectExists(ctx, projectID); err != nil {
		return nil, err
	}
	repoRoot, err := s.projectStorageRoot(projectID)
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

func (s *Service) GetRepositoryPrerequisite(ctx context.Context, projectID uuid.UUID) (WorkflowRepositoryPrerequisite, error) {
	if s.client == nil {
		return WorkflowRepositoryPrerequisite{}, ErrUnavailable
	}
	prerequisite, _, err := s.loadRepositoryPrerequisite(ctx, projectID)
	return prerequisite, err
}

func (s *Service) ensureProjectStorageMirror(ctx context.Context, projectID uuid.UUID, usage workflowStorageUsage) error {
	_ = ctx
	_ = projectID
	_ = usage
	return nil
}

func (s *Service) loadRepositoryPrerequisite(ctx context.Context, projectID uuid.UUID) (WorkflowRepositoryPrerequisite, string, error) {
	if err := s.ensureProjectExists(ctx, projectID); err != nil {
		return WorkflowRepositoryPrerequisite{}, "", err
	}

	repos, err := s.client.ProjectRepo.Query().
		Where(entprojectrepo.ProjectID(projectID)).
		All(ctx)
	if err != nil {
		return WorkflowRepositoryPrerequisite{}, "", fmt.Errorf("list project repos for workflow prerequisite: %w", err)
	}
	repoRoot, err := s.projectStorageRoot(projectID)
	if err != nil {
		return WorkflowRepositoryPrerequisite{}, "", err
	}

	return WorkflowRepositoryPrerequisite{
		Kind:      WorkflowRepositoryPrerequisiteKindReady,
		RepoCount: len(repos),
		Action:    WorkflowRepositoryPrerequisiteActionNone,
	}, repoRoot, nil
}

func (s *Service) projectStorageRoot(projectID uuid.UUID) (string, error) {
	if projectID == uuid.Nil {
		return "", fmt.Errorf("project id must not be empty")
	}
	root := filepath.Join(s.repoRoot, ".openase-projects", projectID.String())
	if err := os.MkdirAll(root, 0o750); err != nil {
		return "", fmt.Errorf("create project storage root: %w", err)
	}
	return root, nil
}

func ResolveReadyMirrorRepoRoot(mirrors []*ent.ProjectRepoMirror) (string, error) {
	if len(mirrors) == 0 {
		return "", errors.New("no ready ProjectRepoMirror.local_path is available")
	}

	ordered := append([]*ent.ProjectRepoMirror(nil), mirrors...)
	slices.SortStableFunc(ordered, compareMirrorLocality)

	var lastErr error
	for _, mirror := range ordered {
		if mirror == nil {
			continue
		}
		localPath := strings.TrimSpace(mirror.LocalPath)
		if localPath == "" {
			continue
		}

		repoRoot, err := DetectRepoRoot(filepath.Clean(localPath))
		if err == nil {
			return repoRoot, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", lastErr
	}

	return "", errors.New("no ready ProjectRepoMirror.local_path is available")
}

func compareMirrorLocality(left *ent.ProjectRepoMirror, right *ent.ProjectRepoMirror) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}

	leftLocal := projectRepoMirrorIsLocal(left)
	rightLocal := projectRepoMirrorIsLocal(right)
	switch {
	case leftLocal && !rightLocal:
		return -1
	case !leftLocal && rightLocal:
		return 1
	default:
		return strings.Compare(strings.TrimSpace(left.LocalPath), strings.TrimSpace(right.LocalPath))
	}
}

func projectRepoMirrorIsLocal(mirror *ent.ProjectRepoMirror) bool {
	if mirror == nil || mirror.Edges.Machine == nil {
		return false
	}

	return mirror.Edges.Machine.Name == catalogdomain.LocalMachineName ||
		mirror.Edges.Machine.Host == catalogdomain.LocalMachineHost
}

func selectRepresentativeWorkflowMirror(mirrors []*ent.ProjectRepoMirror) *ent.ProjectRepoMirror {
	if len(mirrors) == 0 {
		return nil
	}

	ordered := append([]*ent.ProjectRepoMirror(nil), mirrors...)
	slices.SortStableFunc(ordered, compareWorkflowMirrorPriority)

	for _, mirror := range ordered {
		if mirror != nil {
			return mirror
		}
	}
	return nil
}

func selectWorkflowStorageMirror(mirrors []*ent.ProjectRepoMirror) *ent.ProjectRepoMirror {
	if len(mirrors) == 0 {
		return nil
	}

	ordered := append([]*ent.ProjectRepoMirror(nil), mirrors...)
	slices.SortStableFunc(ordered, compareWorkflowMirrorPriority)
	for _, mirror := range ordered {
		if mirror == nil || !projectRepoMirrorIsLocal(mirror) {
			continue
		}
		return mirror
	}
	return nil
}

func compareWorkflowMirrorPriority(left *ent.ProjectRepoMirror, right *ent.ProjectRepoMirror) int {
	leftPriority := workflowMirrorPriority(left)
	rightPriority := workflowMirrorPriority(right)
	switch {
	case leftPriority < rightPriority:
		return -1
	case leftPriority > rightPriority:
		return 1
	default:
		return compareMirrorLocality(left, right)
	}
}

func workflowMirrorPriority(mirror *ent.ProjectRepoMirror) int {
	if mirror == nil {
		return 99
	}

	switch mirror.State {
	case entprojectrepomirror.StateError, entprojectrepomirror.StateStale:
		return 0
	case entprojectrepomirror.StateSyncing, entprojectrepomirror.StateProvisioning:
		return 1
	case entprojectrepomirror.StateDeleting:
		return 2
	case entprojectrepomirror.StateMissing:
		return 3
	case entprojectrepomirror.StateReady:
		return 4
	default:
		return 5
	}
}

func toWorkflowMirrorState(value entprojectrepomirror.State) catalogdomain.ProjectRepoMirrorState {
	switch value {
	case entprojectrepomirror.StateMissing:
		return catalogdomain.ProjectRepoMirrorStateMissing
	case entprojectrepomirror.StateProvisioning:
		return catalogdomain.ProjectRepoMirrorStateProvisioning
	case entprojectrepomirror.StateReady:
		return catalogdomain.ProjectRepoMirrorStateReady
	case entprojectrepomirror.StateStale:
		return catalogdomain.ProjectRepoMirrorStateStale
	case entprojectrepomirror.StateSyncing:
		return catalogdomain.ProjectRepoMirrorStateSyncing
	case entprojectrepomirror.StateError:
		return catalogdomain.ProjectRepoMirrorStateError
	case entprojectrepomirror.StateDeleting:
		return catalogdomain.ProjectRepoMirrorStateDeleting
	default:
		return catalogdomain.ProjectRepoMirrorStateMissing
	}
}

func stringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
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

	if err := tx.Project.Update().
		Where(entproject.DefaultWorkflowIDEQ(current.ID)).
		ClearDefaultWorkflowID().
		Exec(ctx); err != nil {
		return Workflow{}, fmt.Errorf("clear project default workflow reference: %w", err)
	}

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

func (s *Service) handleHarnessReload(event harnessReloadEvent) {
	if s.client == nil {
		return
	}

	ctx := context.Background()
	storage, err := s.storageForProject(ctx, event.ProjectID, workflowStorageUsageWrite)
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
		if err := s.runWorkflowHooks(ctx, item.ProjectID, parsedHooks, workflowHookOnReload, workflowHookRuntime{
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

func (s *Service) hookExecutorForProject(ctx context.Context, projectID uuid.UUID) *workflowHookExecutor {
	if s == nil {
		return nil
	}
	logger := s.logger
	if logger == nil {
		logger = slog.Default()
	}
	repoRoot := strings.TrimSpace(s.repoRoot)
	if projectID != uuid.Nil {
		if resolvedRoot, err := s.projectStorageRoot(projectID); err == nil && strings.TrimSpace(resolvedRoot) != "" {
			repoRoot = resolvedRoot
		}
	}
	if repoRoot == "" {
		return nil
	}
	return newWorkflowHookExecutor(repoRoot, logger)
}

func mapWorkflowMirrorError(err error) error {
	switch {
	case errors.Is(err, projectrepomirrorsvc.ErrMirrorNotReady):
		return ErrRepositoryMirrorNotReady
	case errors.Is(err, projectrepomirrorsvc.ErrMirrorSyncFailed):
		return fmt.Errorf("ensure workflow mirror freshness: %w", err)
	case errors.Is(err, projectrepomirrorsvc.ErrInvalidInput),
		errors.Is(err, projectrepomirrorsvc.ErrNotFound),
		errors.Is(err, projectrepomirrorsvc.ErrInvalidTransition):
		return fmt.Errorf("ensure workflow mirror freshness: %w", err)
	default:
		return err
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
