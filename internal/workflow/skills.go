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
	entskill "github.com/BetterAndBetterII/openase/ent/skill"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	entworkflowskillbinding "github.com/BetterAndBetterII/openase/ent/workflowskillbinding"
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
	if err := s.ensureBuiltinSkills(ctx, projectID); err != nil {
		return nil, err
	}

	items, err := s.client.Skill.Query().
		Where(
			entskill.ProjectIDEQ(projectID),
			entskill.ArchivedAtIsNil(),
		).
		Order(ent.Asc(entskill.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}

	bindings, err := s.client.WorkflowSkillBinding.Query().
		Where(
			entworkflowskillbinding.HasWorkflowWith(entworkflow.ProjectIDEQ(projectID)),
			entworkflowskillbinding.HasSkillWith(
				entskill.ProjectIDEQ(projectID),
				entskill.ArchivedAtIsNil(),
			),
		).
		WithWorkflow().
		WithSkill().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflow skill bindings: %w", err)
	}

	bindingsBySkillID := make(map[uuid.UUID][]SkillWorkflowBinding, len(items))
	for _, binding := range bindings {
		if binding.Edges.Skill == nil || binding.Edges.Workflow == nil {
			continue
		}
		bindingsBySkillID[binding.Edges.Skill.ID] = append(bindingsBySkillID[binding.Edges.Skill.ID], SkillWorkflowBinding{
			ID:          binding.Edges.Workflow.ID,
			Name:        binding.Edges.Workflow.Name,
			HarnessPath: binding.Edges.Workflow.HarnessPath,
		})
	}

	result := make([]Skill, 0, len(items))
	for _, item := range items {
		workflowBindings := bindingsBySkillID[item.ID]
		sort.Slice(workflowBindings, func(i int, j int) bool {
			return workflowBindings[i].Name < workflowBindings[j].Name
		})
		result = append(result, Skill{
			ID:             item.ID,
			Name:           item.Name,
			Description:    item.Description,
			Path:           skillContentRelativePath(item.Name),
			IsBuiltin:      item.IsBuiltin,
			IsEnabled:      item.IsEnabled,
			CreatedBy:      item.CreatedBy,
			CreatedAt:      item.CreatedAt.UTC(),
			BoundWorkflows: workflowBindings,
		})
	}

	return result, nil
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
	skill *ent.Skill
}

func (s *Service) GetSkill(ctx context.Context, skillID uuid.UUID) (SkillDetail, error) {
	record, err := s.resolveSkillRecord(ctx, skillID)
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
	if err := s.ensureBuiltinSkills(ctx, input.ProjectID); err != nil {
		return SkillDetail{}, err
	}
	names, err := normalizeSkillNames([]string{input.Name})
	if err != nil {
		return SkillDetail{}, err
	}
	name := names[0]
	if _, err := s.skillByName(ctx, input.ProjectID, name); err == nil {
		return SkillDetail{}, fmt.Errorf("%w: skill %q already exists", ErrSkillInvalid, name)
	} else if !errors.Is(err, ErrSkillNotFound) {
		return SkillDetail{}, err
	}

	content, err := ensureSkillContent(name, input.Content, input.Description)
	if err != nil {
		return SkillDetail{}, err
	}
	description, err := skillDescriptionFromContent(content)
	if err != nil {
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
	now := time.Now().UTC()

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return SkillDetail{}, fmt.Errorf("start skill create tx: %w", err)
	}
	defer rollback(tx)

	skillItem, err := tx.Skill.Create().
		SetProjectID(input.ProjectID).
		SetName(name).
		SetDescription(description).
		SetIsBuiltin(false).
		SetIsEnabled(enabled).
		SetCreatedBy(createdBy).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return SkillDetail{}, fmt.Errorf("create skill: %w", err)
	}

	versionItem, err := tx.SkillVersion.Create().
		SetSkillID(skillItem.ID).
		SetVersion(1).
		SetContentMarkdown(content).
		SetContentHash(contentHash(content)).
		SetCreatedBy(createdBy).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return SkillDetail{}, fmt.Errorf("create skill version: %w", err)
	}

	skillItem, err = tx.Skill.UpdateOneID(skillItem.ID).
		SetCurrentVersionID(versionItem.ID).
		Save(ctx)
	if err != nil {
		return SkillDetail{}, fmt.Errorf("set skill current version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return SkillDetail{}, fmt.Errorf("commit skill create tx: %w", err)
	}

	return s.buildSkillDetail(ctx, resolvedSkillRecord{skill: skillItem})
}

func (s *Service) UpdateSkill(ctx context.Context, input UpdateSkillInput) (SkillDetail, error) {
	record, err := s.resolveSkillRecord(ctx, input.SkillID)
	if err != nil {
		return SkillDetail{}, err
	}

	content, err := ensureSkillContent(record.skill.Name, input.Content, input.Description)
	if err != nil {
		return SkillDetail{}, err
	}
	description, err := skillDescriptionFromContent(content)
	if err != nil {
		return SkillDetail{}, err
	}
	currentVersion, err := s.currentSkillVersion(ctx, record.skill.ID, nil)
	if err != nil {
		return SkillDetail{}, err
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return SkillDetail{}, fmt.Errorf("start skill update tx: %w", err)
	}
	defer rollback(tx)

	versionItem, err := tx.SkillVersion.Create().
		SetSkillID(record.skill.ID).
		SetVersion(currentVersion.Version + 1).
		SetContentMarkdown(content).
		SetContentHash(contentHash(content)).
		SetCreatedBy(record.skill.CreatedBy).
		Save(ctx)
	if err != nil {
		return SkillDetail{}, fmt.Errorf("create updated skill version: %w", err)
	}

	updatedSkill, err := tx.Skill.UpdateOneID(record.skill.ID).
		SetDescription(description).
		SetCurrentVersionID(versionItem.ID).
		SetUpdatedAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return SkillDetail{}, fmt.Errorf("update skill metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return SkillDetail{}, fmt.Errorf("commit skill update tx: %w", err)
	}

	return s.buildSkillDetail(ctx, resolvedSkillRecord{skill: updatedSkill})
}

func (s *Service) DeleteSkill(ctx context.Context, skillID uuid.UUID) error {
	record, err := s.resolveSkillRecord(ctx, skillID)
	if err != nil {
		return err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start skill delete tx: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.WorkflowSkillBinding.Delete().
		Where(entworkflowskillbinding.SkillIDEQ(record.skill.ID)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete skill bindings: %w", err)
	}

	if _, err := tx.Skill.UpdateOneID(record.skill.ID).
		SetArchivedAt(time.Now().UTC()).
		SetIsEnabled(false).
		SetUpdatedAt(time.Now().UTC()).
		Save(ctx); err != nil {
		return fmt.Errorf("archive skill: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit skill delete tx: %w", err)
	}
	return nil
}

func (s *Service) EnableSkill(ctx context.Context, skillID uuid.UUID) (SkillDetail, error) {
	return s.setSkillEnabled(ctx, skillID, true)
}

func (s *Service) DisableSkill(ctx context.Context, skillID uuid.UUID) (SkillDetail, error) {
	return s.setSkillEnabled(ctx, skillID, false)
}

func (s *Service) setSkillEnabled(ctx context.Context, skillID uuid.UUID, enabled bool) (SkillDetail, error) {
	record, err := s.resolveSkillRecord(ctx, skillID)
	if err != nil {
		return SkillDetail{}, err
	}
	updatedSkill, err := s.client.Skill.UpdateOneID(record.skill.ID).
		SetIsEnabled(enabled).
		SetUpdatedAt(time.Now().UTC()).
		Save(ctx)
	if err != nil {
		return SkillDetail{}, fmt.Errorf("update skill enabled state: %w", err)
	}
	record.skill = updatedSkill
	return s.buildSkillDetail(ctx, record)
}

func (s *Service) BindSkill(ctx context.Context, input UpdateSkillBindingsInput) (SkillDetail, error) {
	record, workflowIDs, err := s.resolveSkillRecordForWorkflowBindings(ctx, input)
	if err != nil {
		return SkillDetail{}, err
	}
	for _, workflowID := range workflowIDs {
		if _, err := s.updateWorkflowSkills(ctx, UpdateWorkflowSkillsInput{
			WorkflowID: workflowID,
			Skills:     []string{record.skill.Name},
		}, true); err != nil {
			return SkillDetail{}, err
		}
	}
	return s.buildSkillDetail(ctx, record)
}

func (s *Service) UnbindSkill(ctx context.Context, input UpdateSkillBindingsInput) (SkillDetail, error) {
	record, workflowIDs, err := s.resolveSkillRecordForWorkflowBindings(ctx, input)
	if err != nil {
		return SkillDetail{}, err
	}
	for _, workflowID := range workflowIDs {
		if _, err := s.updateWorkflowSkills(ctx, UpdateWorkflowSkillsInput{
			WorkflowID: workflowID,
			Skills:     []string{record.skill.Name},
		}, false); err != nil {
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

func (s *Service) resolveSkillRecordForWorkflowBindings(
	ctx context.Context,
	input UpdateSkillBindingsInput,
) (resolvedSkillRecord, []uuid.UUID, error) {
	workflowIDs, err := normalizeWorkflowIDs(input.WorkflowIDs)
	if err != nil {
		return resolvedSkillRecord{}, nil, err
	}
	if s.client == nil {
		return resolvedSkillRecord{}, nil, ErrUnavailable
	}

	workflowItems, err := s.client.Workflow.Query().
		Where(entworkflow.IDIn(workflowIDs...)).
		All(ctx)
	if err != nil {
		return resolvedSkillRecord{}, nil, fmt.Errorf("list workflows for skill binding: %w", err)
	}
	if len(workflowItems) != len(workflowIDs) {
		return resolvedSkillRecord{}, nil, ErrWorkflowNotFound
	}

	projectID := workflowItems[0].ProjectID
	for _, workflowItem := range workflowItems[1:] {
		if workflowItem.ProjectID != projectID {
			return resolvedSkillRecord{}, nil, fmt.Errorf("%w: workflow ids must belong to the same project", ErrSkillInvalid)
		}
	}

	record, err := s.resolveSkillRecordInProject(ctx, projectID, input.SkillID)
	if err != nil {
		return resolvedSkillRecord{}, nil, err
	}
	return record, workflowIDs, nil
}

func (s *Service) resolveSkillRecord(
	ctx context.Context,
	skillID uuid.UUID,
) (resolvedSkillRecord, error) {
	if s.client == nil {
		return resolvedSkillRecord{}, ErrUnavailable
	}
	item, err := s.client.Skill.Query().
		Where(
			entskill.IDEQ(skillID),
			entskill.ArchivedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return resolvedSkillRecord{}, ErrSkillNotFound
		}
		return resolvedSkillRecord{}, fmt.Errorf("get skill: %w", err)
	}
	return resolvedSkillRecord{skill: item}, nil
}

func (s *Service) resolveSkillRecordInProject(
	ctx context.Context,
	projectID uuid.UUID,
	skillID uuid.UUID,
) (resolvedSkillRecord, error) {
	item, err := s.client.Skill.Query().
		Where(
			entskill.IDEQ(skillID),
			entskill.ProjectIDEQ(projectID),
			entskill.ArchivedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return resolvedSkillRecord{}, ErrSkillNotFound
		}
		return resolvedSkillRecord{}, fmt.Errorf("get project skill: %w", err)
	}
	return resolvedSkillRecord{skill: item}, nil
}

func (s *Service) buildSkillDetail(ctx context.Context, record resolvedSkillRecord) (SkillDetail, error) {
	if record.skill == nil {
		return SkillDetail{}, ErrSkillNotFound
	}
	items, err := s.ListSkills(ctx, record.skill.ProjectID)
	if err != nil {
		return SkillDetail{}, err
	}
	for _, item := range items {
		if item.ID != record.skill.ID {
			continue
		}
		versionItem, err := s.currentSkillVersion(ctx, record.skill.ID, nil)
		if err != nil {
			return SkillDetail{}, err
		}
		return SkillDetail{Skill: item, Content: versionItem.ContentMarkdown}, nil
	}
	return SkillDetail{}, ErrSkillNotFound
}

func (s *Service) resolveInjectedSkillNames(
	ctx context.Context,
	projectID uuid.UUID,
	workflowID *uuid.UUID,
) ([]string, error) {
	if workflowID == nil {
		items, err := s.client.Skill.Query().
			Where(
				entskill.ProjectIDEQ(projectID),
				entskill.ArchivedAtIsNil(),
				entskill.IsEnabled(true),
			).
			Order(ent.Asc(entskill.FieldName)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list enabled skills: %w", err)
		}
		names := make([]string, 0, len(items))
		for _, item := range items {
			names = append(names, item.Name)
		}
		sort.Strings(names)
		return names, nil
	}

	names, err := s.listWorkflowBoundSkillNames(ctx, *workflowID, true)
	if err != nil {
		return nil, err
	}
	platformSkill, err := s.skillByName(ctx, projectID, "openase-platform")
	if err != nil && !errors.Is(err, ErrSkillNotFound) {
		return nil, err
	}
	if err == nil && platformSkill.IsEnabled && !slicesContainsString(names, platformSkill.Name) {
		names = append(names, platformSkill.Name)
		sort.Strings(names)
	}
	return names, nil
}

func (s *Service) RefreshSkills(ctx context.Context, input RefreshSkillsInput) (RefreshSkillsResult, error) {
	if s.client == nil {
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

	skillNames, err := s.resolveInjectedSkillNames(ctx, input.ProjectID, input.WorkflowID)
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
		if err := writeProjectedSkill(target.skillsDir.String(), name, versionItem.ContentMarkdown); err != nil {
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

func (s *Service) HarvestSkills(ctx context.Context, input HarvestSkillsInput) (HarvestSkillsResult, error) {
	if s.client == nil {
		return HarvestSkillsResult{}, ErrUnavailable
	}
	return HarvestSkillsResult{}, fmt.Errorf("%w: harvest is deprecated; use create/update skill APIs", ErrSkillInvalid)
}

func (s *Service) BindSkills(ctx context.Context, input UpdateWorkflowSkillsInput) (HarnessDocument, error) {
	return s.updateWorkflowSkills(ctx, input, true)
}

func (s *Service) UnbindSkills(ctx context.Context, input UpdateWorkflowSkillsInput) (HarnessDocument, error) {
	return s.updateWorkflowSkills(ctx, input, false)
}

func (s *Service) updateWorkflowSkills(
	ctx context.Context,
	input UpdateWorkflowSkillsInput,
	bind bool,
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
	if err := s.ensureBuiltinSkills(ctx, workflowItem.ProjectID); err != nil {
		return HarnessDocument{}, err
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return HarnessDocument{}, fmt.Errorf("start workflow skill binding tx: %w", err)
	}
	defer rollback(tx)

	for _, name := range skillNames {
		skillItem, err := s.skillByName(ctx, workflowItem.ProjectID, name)
		if err != nil {
			if bind {
				return HarnessDocument{}, err
			}
			continue
		}

		if bind {
			exists, err := tx.WorkflowSkillBinding.Query().
				Where(
					entworkflowskillbinding.WorkflowIDEQ(workflowItem.ID),
					entworkflowskillbinding.SkillIDEQ(skillItem.ID),
				).
				Exist(ctx)
			if err != nil {
				return HarnessDocument{}, fmt.Errorf("check workflow skill binding: %w", err)
			}
			if exists {
				continue
			}
			if _, err := tx.WorkflowSkillBinding.Create().
				SetWorkflowID(workflowItem.ID).
				SetSkillID(skillItem.ID).
				Save(ctx); err != nil {
				return HarnessDocument{}, fmt.Errorf("create workflow skill binding: %w", err)
			}
			continue
		}

		if _, err := tx.WorkflowSkillBinding.Delete().
			Where(
				entworkflowskillbinding.WorkflowIDEQ(workflowItem.ID),
				entworkflowskillbinding.SkillIDEQ(skillItem.ID),
			).
			Exec(ctx); err != nil {
			return HarnessDocument{}, fmt.Errorf("delete workflow skill binding: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return HarnessDocument{}, fmt.Errorf("commit workflow skill binding tx: %w", err)
	}

	return s.GetHarness(ctx, workflowItem.ID)
}

func writeProjectedSkill(skillsDir string, name string, content string) error {
	skillDir := filepath.Join(skillsDir, name)
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		return fmt.Errorf("create projected skill directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		return fmt.Errorf("write projected skill: %w", err)
	}
	return nil
}

func writeWorkspaceOpenASEWrapper(workspaceRoot string) error {
	dst := filepath.Join(workspaceRoot, ".openase", "bin", "openase")
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return fmt.Errorf("create workspace openase wrapper directory: %w", err)
	}
	if err := os.WriteFile(dst, []byte(workflowOpenASECLIWrapperScript()), 0o600); err != nil {
		return fmt.Errorf("write workspace openase wrapper: %w", err)
	}
	if err := os.Chmod(dst, 0o700); err != nil { //nolint:gosec // dst stays under the prepared workspace root.
		return fmt.Errorf("chmod workspace openase wrapper: %w", err)
	}
	return nil
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
