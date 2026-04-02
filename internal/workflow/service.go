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
	"strings"
	"sync"
	"time"

	"github.com/BetterAndBetterII/openase/internal/builtin"
	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	"github.com/google/uuid"
)

var (
	ErrUnavailable         = errors.New("workflow service unavailable")
	ErrProjectNotFound     = domain.ErrProjectNotFound
	ErrWorkflowNotFound    = domain.ErrWorkflowNotFound
	ErrStatusNotFound      = domain.ErrStatusNotFound
	ErrAgentNotFound       = domain.ErrAgentNotFound
	ErrWorkflowConflict    = domain.ErrWorkflowConflict
	ErrWorkflowInUse       = domain.ErrWorkflowInUse
	ErrHarnessInvalid      = errors.New("workflow harness is invalid")
	ErrHookConfigInvalid   = errors.New("workflow hook config is invalid")
	ErrWorkflowHookBlocked = errors.New("workflow hook blocked the lifecycle operation")
)

var nonAlphaNumericPattern = regexp.MustCompile(`[^a-z0-9]+`)

type Optional[T any] = domain.Optional[T]

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

func (s *Service) currentWorkflowVersion(ctx context.Context, workflowID uuid.UUID) (domain.WorkflowVersionRecord, error) {
	if s == nil || s.repo == nil {
		return domain.WorkflowVersionRecord{}, ErrUnavailable
	}
	return s.repo.CurrentWorkflowVersion(ctx, workflowID)
}

func (s *Service) projectedWorkflowHarness(ctx context.Context, workflowItem Workflow) (string, error) {
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
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	return s.repo.ListWorkflowVersions(ctx, workflowID)
}

func (s *Service) listWorkflowBoundSkillNames(ctx context.Context, workflowID uuid.UUID, enabledOnly bool) ([]string, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	return s.repo.ListWorkflowBoundSkillNames(ctx, workflowID, enabledOnly)
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

func (s *Service) builtinBundles() ([]domain.SkillBundle, error) {
	bundles := make([]domain.SkillBundle, 0, len(builtin.Skills()))
	for _, template := range builtin.Skills() {
		content, err := ensureSkillContent(template.Name, template.Content, template.Description)
		if err != nil {
			return nil, err
		}
		bundle, err := parseSkillBundle(template.Name, []SkillBundleFileInput{{
			Path:      "SKILL.md",
			Content:   []byte(content),
			MediaType: "text/markdown; charset=utf-8",
		}})
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, bundle)
	}
	return bundles, nil
}

func (s *Service) ensureBuiltinSkills(ctx context.Context, projectID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return ErrUnavailable
	}
	s.builtinSkillsMu.Lock()
	defer s.builtinSkillsMu.Unlock()

	bundles, err := s.builtinBundles()
	if err != nil {
		return err
	}
	return s.repo.EnsureBuiltinSkills(ctx, projectID, time.Now().UTC(), bundles)
}

func (s *Service) skillByName(ctx context.Context, projectID uuid.UUID, name string) (domain.SkillRecord, error) {
	if s == nil || s.repo == nil {
		return domain.SkillRecord{}, ErrUnavailable
	}
	return s.repo.SkillByName(ctx, projectID, name)
}

func (s *Service) currentSkillVersion(ctx context.Context, skillID uuid.UUID, requiredVersionID *uuid.UUID) (domain.SkillVersionRecord, error) {
	if s == nil || s.repo == nil {
		return domain.SkillVersionRecord{}, ErrUnavailable
	}
	return s.repo.CurrentSkillVersion(ctx, skillID, requiredVersionID)
}

type resolvedSkillRecord struct {
	skill domain.SkillRecord
}

func (s *Service) listSkillsPersistent(ctx context.Context, projectID uuid.UUID) ([]Skill, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, projectID); err != nil {
		return nil, err
	}
	if err := s.ensureBuiltinSkills(ctx, projectID); err != nil {
		return nil, err
	}
	return s.repo.ListSkills(ctx, projectID)
}

func (s *Service) getSkillPersistent(ctx context.Context, skillID uuid.UUID) (SkillDetail, error) {
	if s == nil || s.repo == nil {
		return SkillDetail{}, ErrUnavailable
	}
	return s.repo.SkillDetail(ctx, skillID)
}

func (s *Service) listSkillVersionsPersistent(ctx context.Context, skillID uuid.UUID) ([]VersionSummary, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	return s.repo.ListSkillVersions(ctx, skillID)
}

func (s *Service) createSkillBundlePersistent(ctx context.Context, input CreateSkillBundleInput) (SkillDetail, error) {
	if s == nil || s.repo == nil {
		return SkillDetail{}, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return SkillDetail{}, err
	}
	if err := s.ensureBuiltinSkills(ctx, input.ProjectID); err != nil {
		return SkillDetail{}, err
	}
	bundle, err := parseSkillBundle(input.Name, input.Files)
	if err != nil {
		return SkillDetail{}, err
	}
	if _, err := s.skillByName(ctx, input.ProjectID, bundle.Name); err == nil {
		return SkillDetail{}, fmt.Errorf("%w: skill %q already exists", ErrSkillInvalid, bundle.Name)
	} else if !errors.Is(err, ErrSkillNotFound) {
		return SkillDetail{}, err
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	createdBy := strings.TrimSpace(input.CreatedBy)
	if createdBy == "" {
		createdBy = "user:manual"
	}
	return s.repo.CreateSkillBundle(ctx, input, bundle, enabled, createdBy, time.Now().UTC())
}

func (s *Service) updateSkillBundlePersistent(ctx context.Context, input UpdateSkillBundleInput) (SkillDetail, error) {
	record, err := s.resolveSkillRecordPersistent(ctx, input.SkillID)
	if err != nil {
		return SkillDetail{}, err
	}

	files := append([]SkillBundleFileInput(nil), input.Files...)
	if input.ReplaceEntry {
		normalizedContent, err := ensureSkillContent(record.skill.Name, input.Content, input.Description)
		if err != nil {
			return SkillDetail{}, err
		}
		replaced := false
		for index := range files {
			if strings.TrimSpace(files[index].Path) != "SKILL.md" {
				continue
			}
			files[index].Content = []byte(normalizedContent)
			files[index].MediaType = "text/markdown; charset=utf-8"
			replaced = true
			break
		}
		if !replaced {
			files = append(files, SkillBundleFileInput{
				Path:      "SKILL.md",
				Content:   []byte(normalizedContent),
				MediaType: "text/markdown; charset=utf-8",
			})
		}
	}

	bundle, err := parseSkillBundle(record.skill.Name, files)
	if err != nil {
		return SkillDetail{}, err
	}
	return s.repo.UpdateSkillBundle(ctx, record.skill.ID, bundle, time.Now().UTC())
}

func (s *Service) deleteSkillPersistent(ctx context.Context, skillID uuid.UUID) error {
	record, err := s.resolveSkillRecordPersistent(ctx, skillID)
	if err != nil {
		return err
	}
	return s.repo.DeleteSkill(ctx, record.skill.ID, time.Now().UTC())
}

func (s *Service) setSkillEnabledPersistent(ctx context.Context, skillID uuid.UUID, enabled bool) (SkillDetail, error) {
	record, err := s.resolveSkillRecordPersistent(ctx, skillID)
	if err != nil {
		return SkillDetail{}, err
	}
	return s.repo.SetSkillEnabled(ctx, record.skill.ID, enabled, time.Now().UTC())
}

func (s *Service) resolveSkillRecordForWorkflowBindingsPersistent(
	ctx context.Context,
	input UpdateSkillBindingsInput,
) (resolvedSkillRecord, []uuid.UUID, error) {
	workflowIDs, err := normalizeWorkflowIDs(input.WorkflowIDs)
	if err != nil {
		return resolvedSkillRecord{}, nil, err
	}
	if s == nil || s.repo == nil {
		return resolvedSkillRecord{}, nil, ErrUnavailable
	}

	var projectID uuid.UUID
	for index, workflowID := range workflowIDs {
		workflowItem, err := s.repo.Get(ctx, workflowID)
		if err != nil {
			return resolvedSkillRecord{}, nil, err
		}
		if index == 0 {
			projectID = workflowItem.ProjectID
			continue
		}
		if workflowItem.ProjectID != projectID {
			return resolvedSkillRecord{}, nil, fmt.Errorf("%w: workflow ids must belong to the same project", ErrSkillInvalid)
		}
	}

	record, err := s.resolveSkillRecordInProjectPersistent(ctx, projectID, input.SkillID)
	if err != nil {
		return resolvedSkillRecord{}, nil, err
	}
	return record, workflowIDs, nil
}

func (s *Service) resolveSkillRecordPersistent(
	ctx context.Context,
	skillID uuid.UUID,
) (resolvedSkillRecord, error) {
	if s == nil || s.repo == nil {
		return resolvedSkillRecord{}, ErrUnavailable
	}
	item, err := s.repo.Skill(ctx, skillID)
	if err != nil {
		return resolvedSkillRecord{}, err
	}
	return resolvedSkillRecord{skill: item}, nil
}

func (s *Service) resolveSkillRecordInProjectPersistent(
	ctx context.Context,
	projectID uuid.UUID,
	skillID uuid.UUID,
) (resolvedSkillRecord, error) {
	if s == nil || s.repo == nil {
		return resolvedSkillRecord{}, ErrUnavailable
	}
	item, err := s.repo.SkillInProject(ctx, projectID, skillID)
	if err != nil {
		return resolvedSkillRecord{}, err
	}
	return resolvedSkillRecord{skill: item}, nil
}

func (s *Service) buildSkillDetailPersistent(ctx context.Context, record resolvedSkillRecord) (SkillDetail, error) {
	if record.skill.ID == uuid.Nil {
		return SkillDetail{}, ErrSkillNotFound
	}
	return s.repo.SkillDetail(ctx, record.skill.ID)
}

func (s *Service) skillVersionFilesPersistent(ctx context.Context, versionID uuid.UUID) ([]SkillBundleFile, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	return s.repo.SkillVersionFiles(ctx, versionID)
}

func (s *Service) resolveInjectedSkillNamesPersistent(
	ctx context.Context,
	projectID uuid.UUID,
	workflowID *uuid.UUID,
) ([]string, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	return s.repo.ResolveInjectedSkillNames(ctx, projectID, workflowID)
}

func (s *Service) refreshSkillsPersistent(ctx context.Context, input RefreshSkillsInput) (RefreshSkillsResult, error) {
	if s == nil || s.repo == nil {
		return RefreshSkillsResult{}, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return RefreshSkillsResult{}, err
	}
	if err := s.ensureBuiltinSkills(ctx, input.ProjectID); err != nil {
		return RefreshSkillsResult{}, err
	}

	target, err := resolveSkillTarget(input.WorkspaceRoot, input.AdapterType)
	if err != nil {
		return RefreshSkillsResult{}, err
	}
	if err := os.RemoveAll(target.skillsDir.String()); err != nil {
		return RefreshSkillsResult{}, fmt.Errorf("reset agent skill directory: %w", err)
	}
	if err := os.MkdirAll(target.skillsDir.String(), 0o750); err != nil {
		return RefreshSkillsResult{}, fmt.Errorf("create agent skill directory: %w", err)
	}

	skillNames, err := s.resolveInjectedSkillNamesPersistent(ctx, input.ProjectID, input.WorkflowID)
	if err != nil {
		return RefreshSkillsResult{}, err
	}

	for _, name := range skillNames {
		skillItem, err := s.skillByName(ctx, input.ProjectID, name)
		if err != nil {
			return RefreshSkillsResult{}, err
		}
		versionItem, err := s.currentSkillVersion(ctx, skillItem.ID, nil)
		if err != nil {
			return RefreshSkillsResult{}, err
		}
		files, err := s.skillVersionFilesPersistent(ctx, versionItem.ID)
		if err != nil {
			return RefreshSkillsResult{}, err
		}
		if err := writeProjectedSkillBundle(target.skillsDir.String(), name, files, versionItem.ContentMarkdown); err != nil {
			return RefreshSkillsResult{}, fmt.Errorf("refresh skill %s: %w", name, err)
		}
	}
	if err := writeWorkspaceOpenASEWrapper(target.workspace.String()); err != nil {
		return RefreshSkillsResult{}, fmt.Errorf("sync openase wrapper: %w", err)
	}

	return RefreshSkillsResult{
		SkillsDir:      target.skillsDir.String(),
		InjectedSkills: skillNames,
	}, nil
}

func (s *Service) updateWorkflowSkillsPersistent(
	ctx context.Context,
	input UpdateWorkflowSkillsInput,
	bind bool,
) (HarnessDocument, error) {
	if s == nil || s.repo == nil {
		return HarnessDocument{}, ErrUnavailable
	}

	skillNames, err := normalizeSkillNames(input.Skills)
	if err != nil {
		return HarnessDocument{}, err
	}
	if len(skillNames) == 0 {
		return HarnessDocument{}, fmt.Errorf("%w: skills must not be empty", ErrSkillInvalid)
	}

	workflowItem, err := s.repo.Get(ctx, input.WorkflowID)
	if err != nil {
		return HarnessDocument{}, err
	}
	if err := s.ensureBuiltinSkills(ctx, workflowItem.ProjectID); err != nil {
		return HarnessDocument{}, err
	}
	previousVersion, err := s.currentWorkflowVersion(ctx, workflowItem.ID)
	if err != nil {
		return HarnessDocument{}, err
	}

	existingBindings, err := s.listWorkflowBoundSkillNames(ctx, workflowItem.ID, false)
	if err != nil {
		return HarnessDocument{}, err
	}
	existingByName := make(map[string]struct{}, len(existingBindings))
	for _, name := range existingBindings {
		existingByName[name] = struct{}{}
	}

	pendingSkillIDs := make([]uuid.UUID, 0, len(skillNames))
	for _, name := range skillNames {
		skillItem, err := s.skillByName(ctx, workflowItem.ProjectID, name)
		if err != nil {
			if bind {
				return HarnessDocument{}, err
			}
			continue
		}
		_, alreadyBound := existingByName[skillItem.Name]
		if bind && alreadyBound {
			continue
		}
		if !bind && !alreadyBound {
			continue
		}
		pendingSkillIDs = append(pendingSkillIDs, skillItem.ID)
	}

	if len(pendingSkillIDs) == 0 {
		return s.GetHarness(ctx, workflowItem.ID)
	}

	if workflowItem.IsActive {
		parsedHooks, err := validateConfiguredHooks(workflowItem.Hooks)
		if err != nil {
			return HarnessDocument{}, err
		}
		if err := s.runWorkflowHooks(ctx, workflowItem.ProjectID, parsedHooks, workflowHookOnReload, workflowHookRuntime{
			ProjectID:       workflowItem.ProjectID,
			WorkflowID:      workflowItem.ID,
			WorkflowName:    workflowItem.Name,
			WorkflowVersion: workflowItem.Version + 1,
		}); err != nil {
			return HarnessDocument{}, err
		}
	}

	if _, err := s.repo.ApplyWorkflowSkillBindings(ctx, workflowItem.ID, pendingSkillIDs, bind, previousVersion.ContentMarkdown); err != nil {
		return HarnessDocument{}, err
	}
	return s.GetHarness(ctx, workflowItem.ID)
}

type Workflow = domain.Workflow

type WorkflowDetail = domain.WorkflowDetail

type VersionSummary = domain.VersionSummary

type HarnessDocument = domain.HarnessDocument

type CreateInput = domain.CreateInput

type UpdateInput = domain.UpdateInput

type UpdateHarnessInput = domain.UpdateHarnessInput

type Service struct {
	repo            Repository
	logger          *slog.Logger
	repoRoot        string
	builtinSkillsMu sync.Mutex
}

func NewService(repo Repository, logger *slog.Logger, repoRoot string) (*Service, error) {
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
		repo:     repo,
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

func (s *Service) logHookConfigValidationFailure(operation string, projectID uuid.UUID, workflowID *uuid.UUID, err error) {
	if err == nil {
		return
	}

	attrs := []any{
		"operation", operation,
		"project_id", projectID,
		"error", err,
	}
	if workflowID != nil {
		attrs = append(attrs, "workflow_id", *workflowID)
	}
	s.logger.Warn("workflow hook configuration invalid", attrs...)
}

func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]Workflow, error) {
	if s == nil || s.repo == nil {
		return nil, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, projectID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, projectID)
}

func (s *Service) Get(ctx context.Context, workflowID uuid.UUID) (WorkflowDetail, error) {
	if s == nil || s.repo == nil {
		return WorkflowDetail{}, ErrUnavailable
	}

	item, err := s.repo.Get(ctx, workflowID)
	if err != nil {
		return WorkflowDetail{}, err
	}
	content, err := s.projectedWorkflowHarness(ctx, item)
	if err != nil {
		return WorkflowDetail{}, err
	}

	return WorkflowDetail{
		Workflow:       item,
		HarnessContent: content,
	}, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (WorkflowDetail, error) {
	if s == nil || s.repo == nil {
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
		s.logHookConfigValidationFailure("create_workflow", input.ProjectID, nil, err)
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

	item, err := s.repo.Create(ctx, Workflow{
		ID:                  workflowID,
		ProjectID:           input.ProjectID,
		AgentID:             &input.AgentID,
		Name:                input.Name,
		Type:                input.Type,
		HarnessPath:         harnessPath,
		Hooks:               copyHooks(input.Hooks),
		MaxConcurrent:       input.MaxConcurrent,
		MaxRetryAttempts:    input.MaxRetryAttempts,
		TimeoutMinutes:      input.TimeoutMinutes,
		StallTimeoutMinutes: input.StallTimeoutMinutes,
		Version:             1,
		IsActive:            input.IsActive,
		PickupStatusIDs:     append([]uuid.UUID(nil), pickupStatusIDs.IDs()...),
		FinishStatusIDs:     append([]uuid.UUID(nil), finishStatusIDs.IDs()...),
	}, sanitizedHarnessContent)
	if err != nil {
		return WorkflowDetail{}, s.mapWorkflowWriteError("create workflow", err)
	}

	projectedContent, err := s.projectedWorkflowHarness(ctx, item)
	if err != nil {
		return WorkflowDetail{}, err
	}

	return WorkflowDetail{
		Workflow:       item,
		HarnessContent: projectedContent,
	}, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (WorkflowDetail, error) {
	if s == nil || s.repo == nil {
		return WorkflowDetail{}, ErrUnavailable
	}

	current, err := s.repo.Get(ctx, input.WorkflowID)
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
		if err := s.ensureHarnessPathAvailable(ctx, projectID, nextHarnessPath, current.ID); err != nil {
			return WorkflowDetail{}, err
		}
	}

	nextPickupStatusIDs := MustStatusBindingSet(current.PickupStatusIDs...)
	if input.PickupStatusIDs.Set {
		nextPickupStatusIDs = input.PickupStatusIDs.Value
	}
	if err := s.ensureStatusBindingsBelongToProject(ctx, projectID, nextPickupStatusIDs); err != nil {
		return WorkflowDetail{}, err
	}

	nextFinishStatusIDs := MustStatusBindingSet(current.FinishStatusIDs...)
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
		s.logHookConfigValidationFailure("update_workflow", projectID, &current.ID, err)
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

	next := current
	next.AgentID = nextAgentID
	next.Name = nextName
	next.HarnessPath = nextHarnessPath
	next.PickupStatusIDs = append([]uuid.UUID(nil), nextPickupStatusIDs.IDs()...)
	next.FinishStatusIDs = append([]uuid.UUID(nil), nextFinishStatusIDs.IDs()...)
	next.Hooks = copyHooks(nextHooksRaw)
	next.IsActive = nextIsActive
	if input.Type.Set {
		next.Type = input.Type.Value
	}
	if input.MaxConcurrent.Set {
		next.MaxConcurrent = input.MaxConcurrent.Value
	}
	if input.MaxRetryAttempts.Set {
		next.MaxRetryAttempts = input.MaxRetryAttempts.Value
	}
	if input.TimeoutMinutes.Set {
		next.TimeoutMinutes = input.TimeoutMinutes.Value
	}
	if input.StallTimeoutMinutes.Set {
		next.StallTimeoutMinutes = input.StallTimeoutMinutes.Value
	}

	item, err := s.repo.Update(ctx, next)
	if err != nil {
		return WorkflowDetail{}, s.mapWorkflowWriteError("update workflow", err)
	}

	content, err := s.projectedWorkflowHarness(ctx, item)
	if err != nil {
		return WorkflowDetail{}, err
	}

	return WorkflowDetail{
		Workflow:       item,
		HarnessContent: content,
	}, nil
}

func (s *Service) Delete(ctx context.Context, workflowID uuid.UUID) (Workflow, error) {
	if s == nil || s.repo == nil {
		return Workflow{}, ErrUnavailable
	}
	return s.repo.Delete(ctx, workflowID)
}

func (s *Service) GetHarness(ctx context.Context, workflowID uuid.UUID) (HarnessDocument, error) {
	if s == nil || s.repo == nil {
		return HarnessDocument{}, ErrUnavailable
	}

	item, err := s.repo.Get(ctx, workflowID)
	if err != nil {
		return HarnessDocument{}, err
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
	if s == nil || s.repo == nil {
		return HarnessDocument{}, ErrUnavailable
	}
	if err := validateHarnessForSave(input.Content); err != nil {
		return HarnessDocument{}, err
	}

	item, err := s.repo.Get(ctx, input.WorkflowID)
	if err != nil {
		return HarnessDocument{}, err
	}
	parsedHooks, err := validateConfiguredHooks(item.Hooks)
	if err != nil {
		s.logHookConfigValidationFailure("update_workflow_harness", item.ProjectID, &item.ID, err)
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

	updated, err := s.repo.PublishWorkflowVersion(ctx, item.ID, sanitizedContent)
	if err != nil {
		return HarnessDocument{}, s.mapWorkflowWriteError("update workflow harness", err)
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
	if s == nil || s.repo == nil {
		return ErrUnavailable
	}
	return s.repo.EnsureProjectExists(ctx, projectID)
}

func (s *Service) ensureStatusBindingsBelongToProject(ctx context.Context, projectID uuid.UUID, statusIDs StatusBindingSet) error {
	if s == nil || s.repo == nil {
		return ErrUnavailable
	}
	return s.repo.EnsureStatusBindingsBelongToProject(ctx, projectID, statusIDs.IDs())
}

func (s *Service) ensureAgentBelongsToProject(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return ErrUnavailable
	}
	return s.repo.EnsureAgentBelongsToProject(ctx, projectID, agentID)
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
	if s == nil || s.repo == nil {
		return ErrUnavailable
	}
	return s.repo.EnsureHarnessPathAvailable(ctx, projectID, harnessPath, excludeWorkflowID)
}

func (s *Service) resolveHarnessContent(
	ctx context.Context,
	name string,
	workflowType Type,
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

	pickupStatuses, err := s.repo.StatusNames(ctx, pickupStatusIDs.IDs())
	if err != nil {
		return "", err
	}
	finishStatuses, err := s.repo.StatusNames(ctx, finishStatusIDs.IDs())
	if err != nil {
		return "", err
	}

	content := defaultHarnessContent(
		name,
		workflowType,
		pickupStatuses,
		finishStatuses,
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
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrWorkflowNotFound):
		return ErrWorkflowNotFound
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func (s *Service) mapWorkflowWriteError(action string, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrWorkflowNotFound):
		return ErrWorkflowNotFound
	case errors.Is(err, ErrWorkflowConflict):
		return ErrWorkflowConflict
	case errors.Is(err, ErrWorkflowInUse):
		return ErrWorkflowInUse
	case strings.Contains(strings.ToLower(err.Error()), "constraint"):
		return ErrWorkflowConflict
	case strings.Contains(strings.ToLower(err.Error()), "tickets"):
		return ErrWorkflowInUse
	case strings.Contains(strings.ToLower(err.Error()), "scheduled_jobs"):
		return ErrWorkflowInUse
	default:
		return fmt.Errorf("%s: %w", action, err)
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

func defaultHarnessContent(name string, workflowType Type, pickupStatusNames []string, finishStatusNames []string) string {
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
	if s == nil || s.repo == nil {
		return RuntimeSnapshot{}, ErrUnavailable
	}

	workflowItem, err := s.repo.Get(ctx, workflowID)
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	if err := s.ensureBuiltinSkills(ctx, workflowItem.ProjectID); err != nil {
		return RuntimeSnapshot{}, err
	}
	return s.repo.ResolveRuntimeSnapshot(ctx, workflowID)
}

func (s *Service) ResolveRecordedRuntimeSnapshot(ctx context.Context, input ResolveRecordedRuntimeSnapshotInput) (RuntimeSnapshot, error) {
	if s == nil || s.repo == nil {
		return RuntimeSnapshot{}, ErrUnavailable
	}

	workflowItem, err := s.repo.Get(ctx, input.WorkflowID)
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	if err := s.ensureBuiltinSkills(ctx, workflowItem.ProjectID); err != nil {
		return RuntimeSnapshot{}, err
	}
	return s.repo.ResolveRecordedRuntimeSnapshot(ctx, input)
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

func (s *Service) BuildHarnessTemplateData(ctx context.Context, input BuildHarnessTemplateDataInput) (HarnessTemplateData, error) {
	if s == nil || s.repo == nil {
		return HarnessTemplateData{}, ErrUnavailable
	}
	workflowItem, err := s.repo.Get(ctx, input.WorkflowID)
	if err != nil {
		return HarnessTemplateData{}, err
	}
	if err := s.ensureBuiltinSkills(ctx, workflowItem.ProjectID); err != nil {
		return HarnessTemplateData{}, err
	}
	data, err := s.repo.BuildHarnessTemplateData(ctx, input)
	if err != nil {
		return HarnessTemplateData{}, err
	}
	return HarnessTemplateData(data), nil
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
