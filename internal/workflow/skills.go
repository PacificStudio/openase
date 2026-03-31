package workflow

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/builtin"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

var (
	ErrSkillInvalid  = errors.New("skill is invalid")
	ErrSkillNotFound = errors.New("skill not found")

	skillNamePattern = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*$`)
)

type Skill struct {
	ID             uuid.UUID              `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Path           string                 `json:"path"`
	IsBuiltin      bool                   `json:"is_builtin"`
	IsEnabled      bool                   `json:"is_enabled"`
	CreatedBy      string                 `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
	BoundWorkflows []SkillWorkflowBinding `json:"bound_workflows"`
}

type SkillWorkflowBinding struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	HarnessPath string    `json:"harness_path"`
}

type skillDocument struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type SkillDetail struct {
	Skill
	Content string `json:"content"`
}

type CreateSkillInput struct {
	ProjectID   uuid.UUID
	Name        string
	Content     string
	Description string
	CreatedBy   string
	Enabled     *bool
}

type UpdateSkillInput struct {
	SkillID     uuid.UUID
	Content     string
	Description string
}

type UpdateSkillBindingsInput struct {
	SkillID     uuid.UUID
	WorkflowIDs []uuid.UUID
}

type RefreshSkillsInput struct {
	ProjectID     uuid.UUID
	WorkspaceRoot string
	AdapterType   string
	WorkflowID    *uuid.UUID
}

type RefreshSkillsResult struct {
	SkillsDir      string   `json:"skills_dir"`
	InjectedSkills []string `json:"injected_skills"`
}

type HarvestSkillsInput struct {
	ProjectID     uuid.UUID
	WorkspaceRoot string
	AdapterType   string
	WorkflowID    *uuid.UUID
	CreatedBy     string
}

type HarvestSkillsResult struct {
	SkillsDir       string   `json:"skills_dir"`
	HarvestedSkills []string `json:"harvested_skills"`
	UpdatedSkills   []string `json:"updated_skills"`
}

type UpdateWorkflowSkillsInput struct {
	WorkflowID uuid.UUID
	Skills     []string
}

type resolvedSkillTarget struct {
	workspace provider.AbsolutePath
	adapter   entagentprovider.AdapterType
	skillsDir provider.AbsolutePath
}

type RuntimeSkillTarget struct {
	SkillsDir string
}

func (s *Service) ListSkills(ctx context.Context, projectID uuid.UUID) ([]Skill, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, projectID); err != nil {
		return nil, err
	}
	storage, err := s.storageForProject(ctx, projectID, workflowStorageUsageRead)
	if err != nil {
		return nil, err
	}

	index, err := s.syncSkillIndex(storage.skillRoot, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	workflows, err := s.client.Workflow.Query().
		Where(entworkflow.ProjectIDEQ(projectID)).
		Order(ent.Asc(entworkflow.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflows for skills: %w", err)
	}

	byName := make(map[string]*Skill, len(index))
	for name, entry := range index {
		description, err := s.readSkillDescription(storage, name)
		if err != nil {
			return nil, err
		}
		byName[name] = &Skill{
			ID:          entry.ID,
			Name:        name,
			Description: description,
			Path:        skillContentRelativePath(name),
			IsBuiltin:   builtin.IsBuiltinSkill(name),
			IsEnabled:   entry.IsEnabled,
			CreatedBy:   entry.CreatedBy,
			CreatedAt:   entry.CreatedAt.UTC(),
		}
	}

	for _, workflowItem := range workflows {
		content, err := storage.registry.Read(workflowItem.HarnessPath)
		if err != nil {
			return nil, fmt.Errorf("read workflow harness for skills: %w", err)
		}

		skillNames, err := ParseHarnessSkills(content)
		if err != nil {
			return nil, err
		}

		for _, name := range skillNames {
			skillItem, ok := byName[name]
			if !ok {
				description, descErr := s.readSkillDescription(storage, name)
				if descErr != nil && !errors.Is(descErr, fs.ErrNotExist) {
					return nil, descErr
				}
				entry, hasEntry := index[name]
				if !hasEntry {
					entry = skillIndexEntry{
						ID:        uuid.New(),
						Name:      name,
						IsEnabled: false,
						CreatedBy: defaultSkillCreator(name),
						CreatedAt: time.Now().UTC(),
					}
				}
				skillItem = &Skill{
					ID:          entry.ID,
					Name:        name,
					Description: description,
					Path:        skillContentRelativePath(name),
					IsBuiltin:   builtin.IsBuiltinSkill(name),
					IsEnabled:   entry.IsEnabled,
					CreatedBy:   entry.CreatedBy,
					CreatedAt:   entry.CreatedAt.UTC(),
				}
				byName[name] = skillItem
			}
			skillItem.BoundWorkflows = append(skillItem.BoundWorkflows, SkillWorkflowBinding{
				ID:          workflowItem.ID,
				Name:        workflowItem.Name,
				HarnessPath: workflowItem.HarnessPath,
			})
		}
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]Skill, 0, len(names))
	for _, name := range names {
		item := *byName[name]
		sort.Slice(item.BoundWorkflows, func(i int, j int) bool {
			return item.BoundWorkflows[i].Name < item.BoundWorkflows[j].Name
		})
		items = append(items, item)
	}

	return items, nil
}

func (s *Service) readSkillDescription(storage *projectStorage, name string) (string, error) {
	if skill, ok := builtin.SkillByName(name); ok && strings.TrimSpace(skill.Description) != "" {
		return skill.Description, nil
	}

	contentPath := filepath.Join(s.skillDirectoryPath(storage, name), "SKILL.md")
	//nolint:gosec // contentPath comes from validated skill metadata rooted in trusted directories
	data, err := os.ReadFile(contentPath)
	if err != nil {
		return "", fmt.Errorf("read skill %s metadata: %w", name, err)
	}

	document, body, err := parseSkillDocument(string(data))
	if err != nil {
		return "", fmt.Errorf("read skill %s metadata: %w", name, err)
	}

	title := parseSkillTitle(body)
	if title != "" {
		return title, nil
	}

	return strings.TrimSpace(document.Description), nil
}

func parseSkillTitle(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}

	return ""
}

type resolvedSkillRecord struct {
	projectID uuid.UUID
	storage   *projectStorage
	entry     skillIndexEntry
}

func (s *Service) GetSkill(ctx context.Context, skillID uuid.UUID) (SkillDetail, error) {
	record, err := s.resolveSkillRecord(ctx, skillID, workflowStorageUsageRead)
	if err != nil {
		return SkillDetail{}, err
	}
	return s.buildSkillDetail(ctx, record)
}

func (s *Service) CreateSkill(ctx context.Context, input CreateSkillInput) (SkillDetail, error) {
	if s.client == nil {
		return SkillDetail{}, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return SkillDetail{}, err
	}
	names, err := normalizeSkillNames([]string{input.Name})
	if err != nil {
		return SkillDetail{}, err
	}

	storage, err := s.storageForProject(ctx, input.ProjectID, workflowStorageUsageWrite)
	if err != nil {
		return SkillDetail{}, err
	}
	index, err := s.syncSkillIndex(storage.skillRoot, time.Now().UTC())
	if err != nil {
		return SkillDetail{}, err
	}
	name := names[0]
	if _, exists := index[name]; exists {
		return SkillDetail{}, fmt.Errorf("%w: skill %q already exists", ErrSkillInvalid, name)
	}

	content, err := ensureSkillContent(name, input.Content, input.Description)
	if err != nil {
		return SkillDetail{}, err
	}
	skillDir := s.skillDirectoryPath(storage, name)
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		return SkillDetail{}, fmt.Errorf("create skill directory %s: %w", skillDir, err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		return SkillDetail{}, fmt.Errorf("write skill %s: %w", name, err)
	}
	if err := validateSkillDirectory(skillDir); err != nil {
		_ = os.RemoveAll(skillDir)
		return SkillDetail{}, err
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	entry := skillIndexEntry{
		ID:        uuid.New(),
		Name:      name,
		IsEnabled: enabled,
		CreatedBy: strings.TrimSpace(input.CreatedBy),
		CreatedAt: time.Now().UTC(),
	}
	if entry.CreatedBy == "" {
		entry.CreatedBy = "user:manual"
	}
	index[name] = entry
	if err := saveSkillIndex(storage.skillRoot, index); err != nil {
		return SkillDetail{}, err
	}

	return s.buildSkillDetail(ctx, resolvedSkillRecord{
		projectID: input.ProjectID,
		storage:   storage,
		entry:     entry,
	})
}

func (s *Service) UpdateSkill(ctx context.Context, input UpdateSkillInput) (SkillDetail, error) {
	record, err := s.resolveSkillRecord(ctx, input.SkillID, workflowStorageUsageWrite)
	if err != nil {
		return SkillDetail{}, err
	}

	content, err := ensureSkillContent(record.entry.Name, input.Content, input.Description)
	if err != nil {
		return SkillDetail{}, err
	}
	skillDir := s.skillDirectoryPath(record.storage, record.entry.Name)
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		return SkillDetail{}, fmt.Errorf("create skill directory %s: %w", skillDir, err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		return SkillDetail{}, fmt.Errorf("write skill %s: %w", record.entry.Name, err)
	}
	if err := validateSkillDirectory(skillDir); err != nil {
		return SkillDetail{}, err
	}

	return s.buildSkillDetail(ctx, record)
}

func (s *Service) DeleteSkill(ctx context.Context, skillID uuid.UUID) error {
	record, err := s.resolveSkillRecord(ctx, skillID, workflowStorageUsageWrite)
	if err != nil {
		return err
	}
	workflows, err := s.client.Workflow.Query().
		Where(entworkflow.ProjectIDEQ(record.projectID)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list workflows for skill delete: %w", err)
	}
	for _, workflowItem := range workflows {
		if _, err := s.UnbindSkills(ctx, UpdateWorkflowSkillsInput{
			WorkflowID: workflowItem.ID,
			Skills:     []string{record.entry.Name},
		}); err != nil && !errors.Is(err, ErrUnavailable) {
			return err
		}
	}

	if err := os.RemoveAll(s.skillDirectoryPath(record.storage, record.entry.Name)); err != nil {
		return fmt.Errorf("delete skill %s: %w", record.entry.Name, err)
	}
	index, err := s.syncSkillIndex(record.storage.skillRoot, time.Now().UTC())
	if err != nil {
		return err
	}
	delete(index, record.entry.Name)
	return saveSkillIndex(record.storage.skillRoot, index)
}

func (s *Service) EnableSkill(ctx context.Context, skillID uuid.UUID) (SkillDetail, error) {
	return s.setSkillEnabled(ctx, skillID, true)
}

func (s *Service) DisableSkill(ctx context.Context, skillID uuid.UUID) (SkillDetail, error) {
	return s.setSkillEnabled(ctx, skillID, false)
}

func (s *Service) setSkillEnabled(ctx context.Context, skillID uuid.UUID, enabled bool) (SkillDetail, error) {
	record, err := s.resolveSkillRecord(ctx, skillID, workflowStorageUsageWrite)
	if err != nil {
		return SkillDetail{}, err
	}
	index, err := s.syncSkillIndex(record.storage.skillRoot, time.Now().UTC())
	if err != nil {
		return SkillDetail{}, err
	}
	entry := index[record.entry.Name]
	entry.IsEnabled = enabled
	index[record.entry.Name] = entry
	if err := saveSkillIndex(record.storage.skillRoot, index); err != nil {
		return SkillDetail{}, err
	}
	record.entry = entry
	return s.buildSkillDetail(ctx, record)
}

func (s *Service) BindSkill(ctx context.Context, input UpdateSkillBindingsInput) (SkillDetail, error) {
	record, err := s.resolveSkillRecord(ctx, input.SkillID, workflowStorageUsageWrite)
	if err != nil {
		return SkillDetail{}, err
	}
	workflowIDs, err := normalizeWorkflowIDs(input.WorkflowIDs)
	if err != nil {
		return SkillDetail{}, err
	}
	for _, workflowID := range workflowIDs {
		if _, err := s.BindSkills(ctx, UpdateWorkflowSkillsInput{
			WorkflowID: workflowID,
			Skills:     []string{record.entry.Name},
		}); err != nil {
			return SkillDetail{}, err
		}
	}
	return s.buildSkillDetail(ctx, record)
}

func (s *Service) UnbindSkill(ctx context.Context, input UpdateSkillBindingsInput) (SkillDetail, error) {
	record, err := s.resolveSkillRecord(ctx, input.SkillID, workflowStorageUsageWrite)
	if err != nil {
		return SkillDetail{}, err
	}
	workflowIDs, err := normalizeWorkflowIDs(input.WorkflowIDs)
	if err != nil {
		return SkillDetail{}, err
	}
	for _, workflowID := range workflowIDs {
		if _, err := s.UnbindSkills(ctx, UpdateWorkflowSkillsInput{
			WorkflowID: workflowID,
			Skills:     []string{record.entry.Name},
		}); err != nil {
			return SkillDetail{}, err
		}
	}
	return s.buildSkillDetail(ctx, record)
}

func normalizeWorkflowIDs(items []uuid.UUID) ([]uuid.UUID, error) {
	unique := make([]uuid.UUID, 0, len(items))
	seen := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		if item == uuid.Nil {
			return nil, fmt.Errorf("%w: workflow id must not be empty", ErrSkillInvalid)
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		unique = append(unique, item)
	}
	if len(unique) == 0 {
		return nil, fmt.Errorf("%w: workflow ids must not be empty", ErrSkillInvalid)
	}
	return unique, nil
}

func (s *Service) resolveSkillRecord(
	ctx context.Context,
	skillID uuid.UUID,
	usage workflowStorageUsage,
) (resolvedSkillRecord, error) {
	if s.client == nil {
		return resolvedSkillRecord{}, ErrUnavailable
	}
	projectIDs, err := s.client.Project.Query().IDs(ctx)
	if err != nil {
		return resolvedSkillRecord{}, fmt.Errorf("list projects for skill lookup: %w", err)
	}
	for _, projectID := range projectIDs {
		storage, err := s.storageForProject(ctx, projectID, usage)
		if err != nil {
			return resolvedSkillRecord{}, err
		}
		index, err := s.syncSkillIndex(storage.skillRoot, time.Now().UTC())
		if err != nil {
			return resolvedSkillRecord{}, err
		}
		for _, entry := range index {
			if entry.ID == skillID {
				return resolvedSkillRecord{
					projectID: projectID,
					storage:   storage,
					entry:     entry,
				}, nil
			}
		}
	}
	return resolvedSkillRecord{}, ErrSkillNotFound
}

func (s *Service) buildSkillDetail(ctx context.Context, record resolvedSkillRecord) (SkillDetail, error) {
	items, err := s.ListSkills(ctx, record.projectID)
	if err != nil {
		return SkillDetail{}, err
	}
	for _, item := range items {
		if item.ID != record.entry.ID {
			continue
		}
		contentPath := filepath.Join(filepath.Clean(s.skillDirectoryPath(record.storage, record.entry.Name)), "SKILL.md")
		//nolint:gosec // contentPath is derived from the validated project skill root and indexed skill name.
		data, err := os.ReadFile(contentPath)
		if err != nil {
			return SkillDetail{}, fmt.Errorf("read skill %s content: %w", record.entry.Name, err)
		}
		return SkillDetail{
			Skill:   item,
			Content: string(data),
		}, nil
	}
	return SkillDetail{}, ErrSkillNotFound
}

func (s *Service) resolveInjectedSkillNames(
	ctx context.Context,
	storage *projectStorage,
	index map[string]skillIndexEntry,
	workflowID *uuid.UUID,
) ([]string, error) {
	if workflowID == nil {
		names := make([]string, 0, len(index))
		for name, entry := range index {
			if entry.IsEnabled {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		return names, nil
	}

	workflowItem, err := s.client.Workflow.Get(ctx, *workflowID)
	if err != nil {
		return nil, s.mapWorkflowReadError("get workflow for skill refresh", err)
	}
	content, err := storage.registry.Read(workflowItem.HarnessPath)
	if err != nil {
		return nil, fmt.Errorf("read workflow harness for skill refresh: %w", err)
	}
	boundNames, err := ParseHarnessSkills(content)
	if err != nil {
		return nil, err
	}

	selected := make([]string, 0, len(boundNames)+1)
	seen := make(map[string]struct{}, len(boundNames)+1)
	for _, name := range boundNames {
		entry, ok := index[name]
		if !ok || !entry.IsEnabled {
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		selected = append(selected, name)
	}
	if entry, ok := index["openase-platform"]; ok && entry.IsEnabled {
		if _, duplicate := seen["openase-platform"]; !duplicate {
			selected = append(selected, "openase-platform")
		}
	}
	sort.Strings(selected)
	return selected, nil
}

func (s *Service) RefreshSkills(ctx context.Context, input RefreshSkillsInput) (RefreshSkillsResult, error) {
	if s.client == nil {
		return RefreshSkillsResult{}, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return RefreshSkillsResult{}, err
	}
	storage, err := s.storageForProject(ctx, input.ProjectID, workflowStorageUsageWrite)
	if err != nil {
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

	skillRoot := resolveWorkspaceSkillRoot(storage, input.WorkspaceRoot)
	index, err := s.syncSkillIndex(skillRoot, time.Now().UTC())
	if err != nil {
		return RefreshSkillsResult{}, err
	}
	skillNames, err := s.resolveInjectedSkillNames(ctx, storage, index, input.WorkflowID)
	if err != nil {
		return RefreshSkillsResult{}, err
	}

	for _, name := range skillNames {
		src := filepath.Join(skillRoot, name)
		dst := filepath.Join(target.skillsDir.String(), name)
		if err := replaceDirectory(src, dst); err != nil {
			return RefreshSkillsResult{}, fmt.Errorf("refresh skill %s: %w", name, err)
		}
	}
	if err := syncProjectWrapperToWorkspace(storage.repoRoot, target.workspace.String()); err != nil {
		return RefreshSkillsResult{}, fmt.Errorf("sync openase wrapper: %w", err)
	}

	return RefreshSkillsResult{
		SkillsDir:      target.skillsDir.String(),
		InjectedSkills: skillNames,
	}, nil
}

func (s *Service) HarvestSkills(ctx context.Context, input HarvestSkillsInput) (HarvestSkillsResult, error) {
	if s.client == nil {
		return HarvestSkillsResult{}, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return HarvestSkillsResult{}, err
	}
	storage, err := s.storageForProject(ctx, input.ProjectID, workflowStorageUsageWrite)
	if err != nil {
		return HarvestSkillsResult{}, err
	}
	skillRoot := resolveWorkspaceSkillRoot(storage, input.WorkspaceRoot)
	index, err := s.syncSkillIndex(skillRoot, time.Now().UTC())
	if err != nil {
		return HarvestSkillsResult{}, err
	}

	target, err := resolveSkillTarget(input.WorkspaceRoot, input.AdapterType)
	if err != nil {
		return HarvestSkillsResult{}, err
	}

	workspaceSkillNames, err := listSkillNames(target.skillsDir.String())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return HarvestSkillsResult{SkillsDir: target.skillsDir.String()}, nil
		}
		return HarvestSkillsResult{}, err
	}

	result := HarvestSkillsResult{SkillsDir: target.skillsDir.String()}
	for _, name := range workspaceSkillNames {
		src := filepath.Join(target.skillsDir.String(), name)
		dst := filepath.Join(skillRoot, name)

		srcFingerprint, err := directoryFingerprint(src)
		if err != nil {
			return HarvestSkillsResult{}, fmt.Errorf("fingerprint harvested skill %s: %w", name, err)
		}

		_, statErr := os.Stat(dst)
		if errors.Is(statErr, fs.ErrNotExist) {
			if err := replaceDirectory(src, dst); err != nil {
				return HarvestSkillsResult{}, fmt.Errorf("harvest skill %s: %w", name, err)
			}
			entry := skillIndexEntry{
				ID:        uuid.New(),
				Name:      name,
				IsEnabled: true,
				CreatedBy: strings.TrimSpace(input.CreatedBy),
				CreatedAt: time.Now().UTC(),
			}
			if entry.CreatedBy == "" {
				entry.CreatedBy = "agent:auto-harvest"
			}
			index[name] = entry
			result.HarvestedSkills = append(result.HarvestedSkills, name)
			continue
		}
		if statErr != nil {
			return HarvestSkillsResult{}, fmt.Errorf("stat project skill %s: %w", name, statErr)
		}

		dstFingerprint, err := directoryFingerprint(dst)
		if err != nil {
			return HarvestSkillsResult{}, fmt.Errorf("fingerprint project skill %s: %w", name, err)
		}
		if srcFingerprint == dstFingerprint {
			continue
		}

		if err := replaceDirectory(src, dst); err != nil {
			return HarvestSkillsResult{}, fmt.Errorf("update harvested skill %s: %w", name, err)
		}
		result.UpdatedSkills = append(result.UpdatedSkills, name)
	}

	sort.Strings(result.HarvestedSkills)
	sort.Strings(result.UpdatedSkills)
	if err := saveSkillIndex(skillRoot, index); err != nil {
		return HarvestSkillsResult{}, err
	}
	if input.WorkflowID != nil && len(result.HarvestedSkills) > 0 {
		if _, err := s.BindSkills(ctx, UpdateWorkflowSkillsInput{
			WorkflowID: *input.WorkflowID,
			Skills:     append([]string(nil), result.HarvestedSkills...),
		}); err != nil {
			return HarvestSkillsResult{}, err
		}
	}

	return result, nil
}

func (s *Service) BindSkills(ctx context.Context, input UpdateWorkflowSkillsInput) (HarnessDocument, error) {
	return s.updateWorkflowSkills(ctx, input, func(current []string, incoming []string) []string {
		next := append([]string(nil), current...)
		for _, name := range incoming {
			if !slicesContainsString(next, name) {
				next = append(next, name)
			}
		}
		return next
	}, true)
}

func (s *Service) UnbindSkills(ctx context.Context, input UpdateWorkflowSkillsInput) (HarnessDocument, error) {
	return s.updateWorkflowSkills(ctx, input, func(current []string, incoming []string) []string {
		next := make([]string, 0, len(current))
		for _, name := range current {
			if slicesContainsString(incoming, name) {
				continue
			}
			next = append(next, name)
		}
		return next
	}, false)
}

func (s *Service) updateWorkflowSkills(
	ctx context.Context,
	input UpdateWorkflowSkillsInput,
	mutate func([]string, []string) []string,
	requireExisting bool,
) (HarnessDocument, error) {
	if s.client == nil {
		return HarnessDocument{}, ErrUnavailable
	}

	skillNames, err := normalizeSkillNames(input.Skills)
	if err != nil {
		return HarnessDocument{}, err
	}
	if len(skillNames) == 0 {
		return HarnessDocument{}, fmt.Errorf("%w: skills must not be empty", ErrSkillInvalid)
	}

	workflowItem, err := s.client.Workflow.Get(ctx, input.WorkflowID)
	if err != nil {
		return HarnessDocument{}, s.mapWorkflowReadError("get workflow for skills update", err)
	}
	storage, err := s.storageForProject(ctx, workflowItem.ProjectID, workflowStorageUsageWrite)
	if err != nil {
		return HarnessDocument{}, err
	}

	if requireExisting {
		for _, name := range skillNames {
			if err := s.ensureSkillExists(storage, name); err != nil {
				return HarnessDocument{}, err
			}
		}
	}

	current, err := storage.registry.Read(workflowItem.HarnessPath)
	if err != nil {
		return HarnessDocument{}, fmt.Errorf("read workflow harness for skills update: %w", err)
	}

	currentSkills, err := ParseHarnessSkills(current)
	if err != nil {
		return HarnessDocument{}, err
	}

	nextSkills := mutate(currentSkills, skillNames)
	nextContent, err := setHarnessSkills(current, nextSkills)
	if err != nil {
		return HarnessDocument{}, err
	}
	if nextContent == current {
		return HarnessDocument{
			WorkflowID: workflowItem.ID,
			Path:       workflowItem.HarnessPath,
			Content:    current,
			Version:    workflowItem.Version,
		}, nil
	}

	return s.UpdateHarness(ctx, UpdateHarnessInput{
		WorkflowID: input.WorkflowID,
		Content:    nextContent,
	})
}

func ParseHarnessSkills(content string) ([]string, error) {
	frontmatter, _, err := extractHarnessFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrHarnessInvalid, err)
	}

	var document struct {
		Skills []string `yaml:"skills"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrHarnessInvalid, err)
	}

	return normalizeSkillNames(document.Skills)
}

func setHarnessSkills(content string, skills []string) (string, error) {
	frontmatter, body, err := extractHarnessFrontmatter(content)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrHarnessInvalid, err)
	}

	normalizedSkills, err := normalizeSkillNames(skills)
	if err != nil {
		return "", err
	}

	var document yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return "", fmt.Errorf("%w: %s", ErrHarnessInvalid, err)
	}

	root := &document
	if document.Kind == yaml.DocumentNode {
		if len(document.Content) != 1 {
			return "", fmt.Errorf("%w: harness frontmatter must contain a single YAML document", ErrHarnessInvalid)
		}
		root = document.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return "", fmt.Errorf("%w: harness frontmatter must be a YAML mapping", ErrHarnessInvalid)
	}

	skillsNode := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, name := range normalizedSkills {
		skillsNode.Content = append(skillsNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: name,
		})
	}

	index := findYAMLMappingValueIndex(root, "skills")
	switch {
	case len(normalizedSkills) == 0 && index >= 0:
		root.Content = append(root.Content[:index-1], root.Content[index+1:]...)
	case len(normalizedSkills) > 0 && index >= 0:
		root.Content[index] = skillsNode
	case len(normalizedSkills) > 0:
		root.Content = append(root.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "skills",
		}, skillsNode)
	}

	marshaled, err := yaml.Marshal(root)
	if err != nil {
		return "", fmt.Errorf("%w: marshal harness skills: %s", ErrHarnessInvalid, err)
	}

	return buildHarnessContent(string(marshaled), body), nil
}

func findYAMLMappingValueIndex(root *yaml.Node, key string) int {
	for index := 0; index+1 < len(root.Content); index += 2 {
		if root.Content[index].Value == key {
			return index + 1
		}
	}
	return -1
}

func buildHarnessContent(frontmatter string, body string) string {
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString(strings.TrimSpace(normalizeHarnessNewlines(frontmatter)))
	builder.WriteString("\n---\n")
	if body != "" {
		builder.WriteString(normalizeHarnessNewlines(body))
	}
	return builder.String()
}

func resolveSkillTarget(workspaceRoot string, rawAdapterType string) (resolvedSkillTarget, error) {
	trimmedWorkspaceRoot := strings.TrimSpace(workspaceRoot)
	if trimmedWorkspaceRoot == "" {
		return resolvedSkillTarget{}, fmt.Errorf("%w: workspace_root must not be empty", ErrSkillInvalid)
	}
	absoluteWorkspaceRoot, err := filepath.Abs(trimmedWorkspaceRoot)
	if err != nil {
		return resolvedSkillTarget{}, fmt.Errorf("%w: resolve workspace root: %s", ErrSkillInvalid, err)
	}
	workspace, err := provider.ParseAbsolutePath(absoluteWorkspaceRoot)
	if err != nil {
		return resolvedSkillTarget{}, fmt.Errorf("%w: %s", ErrSkillInvalid, err)
	}
	if err := os.MkdirAll(workspace.String(), 0o750); err != nil {
		return resolvedSkillTarget{}, fmt.Errorf("%w: create workspace root: %s", ErrSkillInvalid, err)
	}

	adapterType := entagentprovider.AdapterType(strings.ToLower(strings.TrimSpace(rawAdapterType)))
	if err := entagentprovider.AdapterTypeValidator(adapterType); err != nil {
		return resolvedSkillTarget{}, fmt.Errorf("%w: adapter_type must be one of claude-code-cli, codex-app-server, gemini-cli, custom", ErrSkillInvalid)
	}

	var skillsDir string
	switch adapterType {
	case entagentprovider.AdapterTypeClaudeCodeCli:
		skillsDir = filepath.Join(workspace.String(), ".claude", "skills")
	case entagentprovider.AdapterTypeCodexAppServer:
		skillsDir = filepath.Join(workspace.String(), ".codex", "skills")
	default:
		skillsDir = filepath.Join(workspace.String(), ".agent", "skills")
	}

	return resolvedSkillTarget{
		workspace: workspace,
		adapter:   adapterType,
		skillsDir: provider.MustParseAbsolutePath(filepath.Clean(skillsDir)),
	}, nil
}

func ResolveSkillTargetForRuntime(workspaceRoot string, rawAdapterType string) (RuntimeSkillTarget, error) {
	target, err := resolveSkillTarget(workspaceRoot, rawAdapterType)
	if err != nil {
		return RuntimeSkillTarget{}, err
	}
	return RuntimeSkillTarget{SkillsDir: target.skillsDir.String()}, nil
}

func normalizeSkillNames(raw []string) ([]string, error) {
	normalized := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: skill name must not be empty", ErrSkillInvalid)
		}
		if !skillNamePattern.MatchString(trimmed) {
			return nil, fmt.Errorf("%w: skill name %q must match %s", ErrSkillInvalid, trimmed, skillNamePattern.String())
		}
		if !slicesContainsString(normalized, trimmed) {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized, nil
}

func listSkillNames(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("list skills in %s: %w", root, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, err := normalizeSkillNames([]string{name}); err != nil {
			continue
		}
		if err := validateSkillDirectory(filepath.Join(root, name)); err != nil {
			continue
		}
		names = append(names, name)
	}

	sort.Strings(names)
	return names, nil
}

func validateSkillDirectory(dir string) error {
	contentPath := filepath.Join(dir, "SKILL.md")
	//nolint:gosec // contentPath is resolved from validated skill sources
	content, err := os.ReadFile(contentPath)
	if err != nil {
		return fmt.Errorf("%w: missing SKILL.md in %s", ErrSkillInvalid, dir)
	}
	document, _, err := parseSkillDocument(string(content))
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSkillInvalid, err)
	}
	if document.Name != filepath.Base(dir) {
		return fmt.Errorf("%w: skill frontmatter name %q must match directory %q", ErrSkillInvalid, document.Name, filepath.Base(dir))
	}
	return nil
}

func parseSkillDocument(content string) (skillDocument, string, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return skillDocument{}, "", fmt.Errorf("skill must begin with YAML frontmatter delimited by ---")
	}

	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) != "---" {
			continue
		}

		frontmatter := strings.Join(lines[1:index], "\n")
		body := strings.Join(lines[index+1:], "\n")
		if strings.TrimSpace(frontmatter) == "" {
			return skillDocument{}, "", fmt.Errorf("skill YAML frontmatter must not be empty")
		}

		var document skillDocument
		if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
			return skillDocument{}, "", fmt.Errorf("parse skill YAML frontmatter: %w", err)
		}
		document.Name = strings.TrimSpace(document.Name)
		document.Description = strings.TrimSpace(document.Description)
		if _, err := normalizeSkillNames([]string{document.Name}); err != nil {
			return skillDocument{}, "", fmt.Errorf("skill YAML frontmatter name is invalid: %w", err)
		}
		if document.Description == "" {
			return skillDocument{}, "", fmt.Errorf("skill YAML frontmatter description must not be empty")
		}
		if strings.TrimSpace(body) == "" {
			return skillDocument{}, "", fmt.Errorf("skill body must not be empty")
		}
		return document, body, nil
	}

	return skillDocument{}, "", fmt.Errorf("skill YAML frontmatter is missing the closing --- delimiter")
}

func (s *Service) ensureSkillExists(storage *projectStorage, name string) error {
	if err := validateSkillDirectory(s.skillDirectoryPath(storage, name)); err != nil {
		return fmt.Errorf("%w: %s", ErrSkillNotFound, name)
	}
	return nil
}

func (s *Service) skillDirectoryPath(storage *projectStorage, name string) string {
	return filepath.Join(storage.skillRoot, name)
}

func skillContentRelativePath(name string) string {
	return filepath.ToSlash(filepath.Join(".openase", "skills", name, "SKILL.md"))
}

func replaceDirectory(src string, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("remove %s before replace: %w", dst, err)
	}
	return copyDirectory(src, dst)
}

func copyDirectory(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source directory %s: %w", src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source %s must be a directory", src)
	}

	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk source directory %s: %w", src, walkErr)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not supported in skill directories: %s", path)
		}

		relative, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("resolve relative skill path for %s: %w", path, err)
		}
		targetPath := filepath.Join(dst, relative)

		entryInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read file info for %s: %w", path, err)
		}

		if entry.IsDir() {
			return os.MkdirAll(targetPath, entryInfo.Mode().Perm())
		}

		//nolint:gosec // path comes from walking the validated source skill directory
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read skill file %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
			return fmt.Errorf("create skill file parent %s: %w", targetPath, err)
		}
		if err := ensureCopyTargetWithinRoot(dst, targetPath); err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, content, entryInfo.Mode().Perm()); err != nil { //nolint:gosec // target path is validated to remain within the destination root
			return fmt.Errorf("write skill file %s: %w", targetPath, err)
		}
		return nil
	})
}

func ensureCopyTargetWithinRoot(root string, targetPath string) error {
	relative, err := filepath.Rel(root, targetPath)
	if err != nil {
		return fmt.Errorf("resolve skill copy target %s: %w", targetPath, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("skill copy target escapes root %s: %s", root, targetPath)
	}

	return nil
}

func directoryFingerprint(root string) (string, error) {
	hash := sha256.New()
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relative == "." {
			relative = ""
		}

		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}

		hash.Write([]byte(relative))
		hash.Write([]byte{0})
		hash.Write([]byte(entryInfo.Mode().String()))
		hash.Write([]byte{0})

		if entry.IsDir() {
			return nil
		}

		//nolint:gosec // path comes from walking the validated fingerprint root
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fileHash := sha256.Sum256(content)
		hash.Write(fileHash[:])
		hash.Write([]byte{0})
		return nil
	}); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
