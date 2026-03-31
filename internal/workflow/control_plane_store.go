package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entskill "github.com/BetterAndBetterII/openase/ent/skill"
	entskillversion "github.com/BetterAndBetterII/openase/ent/skillversion"
	entworkflowskillbinding "github.com/BetterAndBetterII/openase/ent/workflowskillbinding"
	entworkflowversion "github.com/BetterAndBetterII/openase/ent/workflowversion"
	"github.com/BetterAndBetterII/openase/internal/builtin"
	"github.com/google/uuid"
)

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
		if resolvedRoot, resolveErr := s.resolvePrimaryRepoRoot(ctx, workflowItem.ProjectID); resolveErr == nil {
			repoRoot = strings.TrimSpace(resolvedRoot)
		}
	}
	if repoRoot == "" {
		return nil, ErrWorkflowNotFound
	}

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

func mapSkillBindingNames(bindings []*ent.WorkflowSkillBinding) []string {
	names := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		if binding.Edges.Skill == nil || binding.Edges.Skill.ArchivedAt != nil {
			continue
		}
		names = append(names, binding.Edges.Skill.Name)
	}
	sort.Strings(names)
	return names
}

func mapSkillDetail(item *ent.Skill, content string, workflows []SkillWorkflowBinding) SkillDetail {
	return SkillDetail{
		Skill: Skill{
			ID:             item.ID,
			Name:           item.Name,
			Description:    item.Description,
			Path:           skillContentRelativePath(item.Name),
			IsBuiltin:      item.IsBuiltin,
			IsEnabled:      item.IsEnabled,
			CreatedBy:      item.CreatedBy,
			CreatedAt:      item.CreatedAt.UTC(),
			BoundWorkflows: workflows,
		},
		Content: content,
	}
}
