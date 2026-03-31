package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entskill "github.com/BetterAndBetterII/openase/ent/skill"
	entskillversion "github.com/BetterAndBetterII/openase/ent/skillversion"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	entworkflowskillbinding "github.com/BetterAndBetterII/openase/ent/workflowskillbinding"
	entworkflowversion "github.com/BetterAndBetterII/openase/ent/workflowversion"
	"github.com/google/uuid"
)

type RuntimeSnapshot struct {
	Workflow RuntimeWorkflowSnapshot
	Skills   []RuntimeSkillSnapshot
}

type RuntimeWorkflowSnapshot struct {
	WorkflowID uuid.UUID
	VersionID  uuid.UUID
	Version    int
	Path       string
	Content    string
}

type RuntimeSkillSnapshot struct {
	SkillID    uuid.UUID
	Name       string
	VersionID  uuid.UUID
	Version    int
	Content    string
	IsRequired bool
}

type MaterializeRuntimeSnapshotInput struct {
	WorkspaceRoot string
	AdapterType   string
	Snapshot      RuntimeSnapshot
}

type ResolveRecordedRuntimeSnapshotInput struct {
	WorkflowID        uuid.UUID
	WorkflowVersionID *uuid.UUID
	SkillVersionIDs   []uuid.UUID
}

type MaterializedRuntimeSnapshot struct {
	HarnessPath       string
	SkillsDir         string
	WorkflowVersionID uuid.UUID
	SkillVersionIDs   []uuid.UUID
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

func (s *Service) MaterializeRuntimeSnapshot(input MaterializeRuntimeSnapshotInput) (MaterializedRuntimeSnapshot, error) {
	if s == nil || s.client == nil {
		return MaterializedRuntimeSnapshot{}, ErrUnavailable
	}
	if input.Snapshot.Workflow.VersionID == uuid.Nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("%w: workflow snapshot version_id must not be empty", ErrHarnessInvalid)
	}

	target, err := resolveSkillTarget(input.WorkspaceRoot, input.AdapterType)
	if err != nil {
		return MaterializedRuntimeSnapshot{}, err
	}
	if err := os.RemoveAll(target.skillsDir.String()); err != nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("reset agent skill directory: %w", err)
	}
	if err := os.MkdirAll(target.skillsDir.String(), 0o750); err != nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("create agent skill directory: %w", err)
	}

	harnessPath, err := ResolveHarnessTargetForRuntime(target.workspace.String(), input.Snapshot.Workflow.Path)
	if err != nil {
		return MaterializedRuntimeSnapshot{}, err
	}
	if err := os.MkdirAll(filepath.Dir(harnessPath), 0o750); err != nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("create runtime harness directory: %w", err)
	}
	if err := os.WriteFile(harnessPath, []byte(input.Snapshot.Workflow.Content), 0o600); err != nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("write runtime harness snapshot: %w", err)
	}

	skillVersionIDs := make([]uuid.UUID, 0, len(input.Snapshot.Skills))
	for _, skill := range input.Snapshot.Skills {
		if err := writeProjectedSkill(target.skillsDir.String(), skill.Name, skill.Content); err != nil {
			return MaterializedRuntimeSnapshot{}, fmt.Errorf("materialize skill %s: %w", skill.Name, err)
		}
		skillVersionIDs = append(skillVersionIDs, skill.VersionID)
	}
	if err := writeWorkspaceOpenASEWrapper(target.workspace.String()); err != nil {
		return MaterializedRuntimeSnapshot{}, fmt.Errorf("sync openase wrapper: %w", err)
	}

	return MaterializedRuntimeSnapshot{
		HarnessPath:       harnessPath,
		SkillsDir:         target.skillsDir.String(),
		WorkflowVersionID: input.Snapshot.Workflow.VersionID,
		SkillVersionIDs:   skillVersionIDs,
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

func ResolveHarnessTargetForRuntime(workspaceRoot string, harnessPath string) (string, error) {
	trimmedWorkspaceRoot := strings.TrimSpace(workspaceRoot)
	if trimmedWorkspaceRoot == "" {
		return "", fmt.Errorf("%w: workspace_root must not be empty", ErrHarnessInvalid)
	}
	absoluteWorkspaceRoot, err := filepath.Abs(trimmedWorkspaceRoot)
	if err != nil {
		return "", fmt.Errorf("%w: resolve workspace root: %s", ErrHarnessInvalid, err)
	}

	normalizedHarnessPath, err := normalizeHarnessPath(harnessPath)
	if err != nil {
		return "", err
	}

	return filepath.Join(absoluteWorkspaceRoot, filepath.FromSlash(normalizedHarnessPath)), nil
}

func runtimeSkillNames(skills []RuntimeSkillSnapshot) []string {
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		names = append(names, skill.Name)
	}
	sort.Strings(names)
	return names
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
		snapshots = append(snapshots, RuntimeSkillSnapshot{
			SkillID:    skillItem.ID,
			Name:       skillItem.Name,
			VersionID:  versionItem.ID,
			Version:    versionItem.Version,
			Content:    versionItem.ContentMarkdown,
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
			snapshots = append(snapshots, RuntimeSkillSnapshot{
				SkillID:   platformSkill.ID,
				Name:      platformSkill.Name,
				VersionID: versionItem.ID,
				Version:   versionItem.Version,
				Content:   versionItem.ContentMarkdown,
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

		snapshots = append(snapshots, RuntimeSkillSnapshot{
			SkillID:   versionItem.Edges.Skill.ID,
			Name:      versionItem.Edges.Skill.Name,
			VersionID: versionItem.ID,
			Version:   versionItem.Version,
			Content:   versionItem.ContentMarkdown,
		})
	}

	sort.SliceStable(snapshots, func(i int, j int) bool {
		return snapshots[i].Name < snapshots[j].Name
	})
	return snapshots, nil
}
