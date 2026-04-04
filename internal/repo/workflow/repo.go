package workflow

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entprojectupdatecomment "github.com/BetterAndBetterII/openase/ent/projectupdatecomment"
	entprojectupdatethread "github.com/BetterAndBetterII/openase/ent/projectupdatethread"
	entskill "github.com/BetterAndBetterII/openase/ent/skill"
	entskillblob "github.com/BetterAndBetterII/openase/ent/skillblob"
	entskillversion "github.com/BetterAndBetterII/openase/ent/skillversion"
	entskillversionfile "github.com/BetterAndBetterII/openase/ent/skillversionfile"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	entworkflowskillbinding "github.com/BetterAndBetterII/openase/ent/workflowskillbinding"
	entworkflowversion "github.com/BetterAndBetterII/openase/ent/workflowversion"
	"github.com/BetterAndBetterII/openase/internal/builtin"
	ticketingdomain "github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/BetterAndBetterII/openase/internal/types/pgarray"
	"github.com/google/uuid"
)

type EntRepository struct {
	client *ent.Client
}

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*$`)

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) EnsureProjectExists(ctx context.Context, projectID uuid.UUID) error {
	exists, err := r.client.Project.Query().Where(entproject.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return domain.ErrProjectNotFound
	}
	return nil
}

func (r *EntRepository) EnsureAgentBelongsToProject(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID) error {
	exists, err := r.client.Agent.Query().
		Where(
			entagent.ProjectIDEQ(projectID),
			entagent.IDEQ(agentID),
			entagent.RuntimeControlStateNEQ(entagent.RuntimeControlStateRetired),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check workflow agent existence: %w", err)
	}
	if !exists {
		return domain.ErrAgentNotFound
	}
	return nil
}

func (r *EntRepository) EnsureStatusBindingsBelongToProject(ctx context.Context, projectID uuid.UUID, statusIDs []uuid.UUID) error {
	count, err := r.client.TicketStatus.Query().
		Where(
			entticketstatus.IDIn(statusIDs...),
			entticketstatus.ProjectIDEQ(projectID),
		).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("check workflow status existence: %w", err)
	}
	if count != len(statusIDs) {
		return domain.ErrStatusNotFound
	}
	return nil
}

func (r *EntRepository) EnsurePickupStatusBindingsAvailable(
	ctx context.Context,
	projectID uuid.UUID,
	statusIDs []uuid.UUID,
	excludeWorkflowID uuid.UUID,
) error {
	query := r.client.Workflow.Query().
		Where(
			entworkflow.ProjectIDEQ(projectID),
			entworkflow.HasPickupStatusesWith(entticketstatus.IDIn(statusIDs...)),
		).
		WithPickupStatuses(func(statusQuery *ent.TicketStatusQuery) {
			statusQuery.Where(entticketstatus.IDIn(statusIDs...)).
				Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		Order(ent.Asc(entworkflow.FieldName))
	if excludeWorkflowID != uuid.Nil {
		query = query.Where(entworkflow.IDNEQ(excludeWorkflowID))
	}

	items, err := query.All(ctx)
	if err != nil {
		return fmt.Errorf("check workflow pickup status uniqueness: %w", err)
	}
	if len(items) == 0 {
		return nil
	}

	conflicts := make([]string, 0)
	seen := make(map[uuid.UUID]struct{}, len(statusIDs))
	for _, item := range items {
		for _, status := range item.Edges.PickupStatuses {
			if _, ok := seen[status.ID]; ok {
				continue
			}
			seen[status.ID] = struct{}{}
			conflicts = append(conflicts, fmt.Sprintf("%q is already used by workflow %q", status.Name, item.Name))
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("%w: pickup status %s", domain.ErrPickupStatusConflict, strings.Join(conflicts, ", "))
}

func (r *EntRepository) EnsureWorkflowNameAvailable(
	ctx context.Context,
	projectID uuid.UUID,
	name string,
	excludeWorkflowID uuid.UUID,
) error {
	query := r.client.Workflow.Query().Where(
		entworkflow.ProjectIDEQ(projectID),
		entworkflow.NameEQ(strings.TrimSpace(name)),
	)
	if excludeWorkflowID != uuid.Nil {
		query = query.Where(entworkflow.IDNEQ(excludeWorkflowID))
	}
	exists, err := query.Exist(ctx)
	if err != nil {
		return fmt.Errorf("check workflow name uniqueness: %w", err)
	}
	if exists {
		return fmt.Errorf("%w: workflow name %q is already used in this project", domain.ErrWorkflowNameConflict, strings.TrimSpace(name))
	}
	return nil
}

func (r *EntRepository) EnsureHarnessPathAvailable(
	ctx context.Context,
	projectID uuid.UUID,
	harnessPath string,
	excludeWorkflowID uuid.UUID,
) error {
	query := r.client.Workflow.Query().Where(
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
		return fmt.Errorf("%w: harness_path %q is already used by another workflow", domain.ErrWorkflowHarnessPathConflict, harnessPath)
	}
	return nil
}

func (r *EntRepository) StatusNames(ctx context.Context, statusIDs []uuid.UUID) ([]string, error) {
	items, err := r.client.TicketStatus.Query().
		Where(entticketstatus.IDIn(statusIDs...)).
		Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("get workflow statuses: %w", err)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return names, nil
}

func (r *EntRepository) List(ctx context.Context, projectID uuid.UUID) ([]domain.Workflow, error) {
	if err := r.ensureProjectWorkflowsMigrated(ctx, projectID); err != nil {
		return nil, err
	}

	items, err := r.client.Workflow.Query().
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

	workflows := make([]domain.Workflow, 0, len(items))
	for _, item := range items {
		workflows = append(workflows, mapWorkflow(item))
	}
	return workflows, nil
}

func (r *EntRepository) Get(ctx context.Context, workflowID uuid.UUID) (domain.Workflow, error) {
	if err := r.ensureWorkflowMigrated(ctx, workflowID); err != nil {
		return domain.Workflow{}, err
	}

	item, err := r.client.Workflow.Query().
		Where(entworkflow.IDEQ(workflowID)).
		WithPickupStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		WithFinishStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		Only(ctx)
	if err != nil {
		return domain.Workflow{}, mapWorkflowReadError("get workflow", err)
	}
	return mapWorkflow(item), nil
}

func (r *EntRepository) Create(ctx context.Context, workflow domain.Workflow, harnessContent string, createdBy string) (domain.Workflow, error) {
	workflowID := workflow.ID
	if workflowID == uuid.Nil {
		workflowID = uuid.New()
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Workflow{}, fmt.Errorf("start workflow create tx: %w", err)
	}
	defer rollback(tx)

	_, err = tx.Workflow.Create().
		SetID(workflowID).
		SetProjectID(workflow.ProjectID).
		SetName(workflow.Name).
		SetType(workflow.Type.String()).
		SetRoleSlug(strings.TrimSpace(workflow.RoleSlug)).
		SetRoleName(strings.TrimSpace(workflow.RoleName)).
		SetRoleDescription(strings.TrimSpace(workflow.RoleDescription)).
		SetPlatformAccessAllowed(pgarray.StringArray(copyStrings(workflow.PlatformAccessAllowed))).
		SetHarnessPath(workflow.HarnessPath).
		SetHooks(copyHooks(workflow.Hooks)).
		SetMaxConcurrent(workflow.MaxConcurrent).
		SetMaxRetryAttempts(workflow.MaxRetryAttempts).
		SetTimeoutMinutes(workflow.TimeoutMinutes).
		SetStallTimeoutMinutes(workflow.StallTimeoutMinutes).
		SetVersion(1).
		SetIsActive(workflow.IsActive).
		AddPickupStatusIDs(workflow.PickupStatusIDs...).
		AddFinishStatusIDs(workflow.FinishStatusIDs...).
		Save(ctx)
	if err != nil {
		return domain.Workflow{}, mapWorkflowWriteError("create workflow", err)
	}
	if workflow.AgentID != nil {
		_, err = tx.Workflow.UpdateOneID(workflowID).SetAgentID(*workflow.AgentID).Save(ctx)
		if err != nil {
			return domain.Workflow{}, mapWorkflowWriteError("set workflow agent", err)
		}
	}

	versionItem, err := r.createWorkflowVersionSnapshot(
		ctx,
		tx,
		workflow,
		1,
		harnessContent,
		resolveCreatedBy(createdBy),
	)
	if err != nil {
		return domain.Workflow{}, mapWorkflowWriteError("create workflow version", err)
	}

	if _, err := tx.Workflow.UpdateOneID(workflowID).
		SetCurrentVersionID(versionItem.ID).
		Save(ctx); err != nil {
		return domain.Workflow{}, mapWorkflowWriteError("set workflow current version", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.Workflow{}, fmt.Errorf("commit workflow create tx: %w", err)
	}
	return r.Get(ctx, workflowID)
}

func (r *EntRepository) Update(ctx context.Context, workflow domain.Workflow) (domain.Workflow, error) {
	builder := r.client.Workflow.UpdateOneID(workflow.ID).
		SetName(workflow.Name).
		SetType(workflow.Type.String()).
		SetRoleSlug(strings.TrimSpace(workflow.RoleSlug)).
		SetRoleName(strings.TrimSpace(workflow.RoleName)).
		SetRoleDescription(strings.TrimSpace(workflow.RoleDescription)).
		SetPlatformAccessAllowed(pgarray.StringArray(copyStrings(workflow.PlatformAccessAllowed))).
		SetHarnessPath(workflow.HarnessPath).
		SetHooks(copyHooks(workflow.Hooks)).
		SetMaxConcurrent(workflow.MaxConcurrent).
		SetMaxRetryAttempts(workflow.MaxRetryAttempts).
		SetTimeoutMinutes(workflow.TimeoutMinutes).
		SetStallTimeoutMinutes(workflow.StallTimeoutMinutes).
		SetIsActive(workflow.IsActive).
		ClearPickupStatuses().
		AddPickupStatusIDs(workflow.PickupStatusIDs...).
		ClearFinishStatuses().
		AddFinishStatusIDs(workflow.FinishStatusIDs...)

	if workflow.AgentID == nil {
		builder.ClearAgent()
	} else {
		builder.SetAgentID(*workflow.AgentID)
	}

	if _, err := builder.Save(ctx); err != nil {
		return domain.Workflow{}, mapWorkflowWriteError("update workflow", err)
	}
	return r.Get(ctx, workflow.ID)
}

func (r *EntRepository) Delete(ctx context.Context, workflowID uuid.UUID) (domain.Workflow, error) {
	current, err := r.Get(ctx, workflowID)
	if err != nil {
		return domain.Workflow{}, err
	}
	impact, err := r.ImpactAnalysis(ctx, workflowID)
	if err != nil {
		return domain.Workflow{}, err
	}
	if conflictErr := workflowDeleteConflictError(impact); conflictErr != nil {
		return domain.Workflow{}, &domain.WorkflowImpactConflict{Err: conflictErr, Impact: impact}
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Workflow{}, fmt.Errorf("start workflow delete tx: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.WorkflowSkillBinding.Delete().
		Where(entworkflowskillbinding.WorkflowIDEQ(workflowID)).
		Exec(ctx); err != nil {
		return domain.Workflow{}, fmt.Errorf("delete workflow skill bindings: %w", err)
	}
	if _, err := tx.WorkflowVersion.Delete().
		Where(entworkflowversion.WorkflowIDEQ(workflowID)).
		Exec(ctx); err != nil {
		return domain.Workflow{}, fmt.Errorf("delete workflow versions: %w", err)
	}
	if err := tx.Workflow.DeleteOneID(workflowID).Exec(ctx); err != nil {
		return domain.Workflow{}, mapWorkflowWriteError("delete workflow", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Workflow{}, fmt.Errorf("commit workflow delete tx: %w", err)
	}
	return current, nil
}

func (r *EntRepository) CurrentWorkflowVersion(ctx context.Context, workflowID uuid.UUID) (domain.WorkflowVersionRecord, error) {
	if err := r.ensureWorkflowMigrated(ctx, workflowID); err != nil {
		return domain.WorkflowVersionRecord{}, err
	}

	item, err := r.client.WorkflowVersion.Query().
		Where(entworkflowversion.WorkflowIDEQ(workflowID)).
		Order(ent.Desc(entworkflowversion.FieldVersion)).
		First(ctx)
	if err != nil {
		return domain.WorkflowVersionRecord{}, mapWorkflowReadError("get workflow version", err)
	}
	return mapWorkflowVersion(item), nil
}

func (r *EntRepository) RecordedWorkflowVersion(
	ctx context.Context,
	workflowID uuid.UUID,
	workflowVersionID *uuid.UUID,
) (domain.WorkflowVersionRecord, error) {
	if err := r.ensureWorkflowMigrated(ctx, workflowID); err != nil {
		return domain.WorkflowVersionRecord{}, err
	}

	if workflowVersionID == nil || *workflowVersionID == uuid.Nil {
		return r.CurrentWorkflowVersion(ctx, workflowID)
	}
	item, err := r.client.WorkflowVersion.Query().
		Where(
			entworkflowversion.IDEQ(*workflowVersionID),
			entworkflowversion.WorkflowIDEQ(workflowID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.WorkflowVersionRecord{}, fmt.Errorf("%w: recorded workflow version %s not found for workflow %s", domain.ErrWorkflowNotFound, *workflowVersionID, workflowID)
		}
		return domain.WorkflowVersionRecord{}, fmt.Errorf("get recorded workflow version: %w", err)
	}
	return mapWorkflowVersion(item), nil
}

func (r *EntRepository) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID) ([]domain.VersionSummary, error) {
	if err := r.ensureWorkflowMigrated(ctx, workflowID); err != nil {
		return nil, err
	}

	if _, err := r.CurrentWorkflowVersion(ctx, workflowID); err != nil {
		return nil, err
	}

	items, err := r.client.WorkflowVersion.Query().
		Where(entworkflowversion.WorkflowIDEQ(workflowID)).
		Order(ent.Desc(entworkflowversion.FieldVersion)).
		All(ctx)
	if err != nil {
		return nil, mapWorkflowReadError("list workflow versions", err)
	}

	result := make([]domain.VersionSummary, 0, len(items))
	for _, item := range items {
		result = append(result, domain.VersionSummary{
			ID:        item.ID,
			Version:   item.Version,
			CreatedBy: item.CreatedBy,
			CreatedAt: item.CreatedAt.UTC(),
		})
	}
	return result, nil
}

func (r *EntRepository) PublishWorkflowVersion(ctx context.Context, workflowID uuid.UUID, content string, createdBy string) (domain.Workflow, error) {
	current, err := r.Get(ctx, workflowID)
	if err != nil {
		return domain.Workflow{}, err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Workflow{}, fmt.Errorf("start workflow publish tx: %w", err)
	}
	defer rollback(tx)

	versionItem, err := r.createWorkflowVersionSnapshot(
		ctx,
		tx,
		current,
		current.Version+1,
		content,
		resolveCreatedBy(createdBy),
	)
	if err != nil {
		return domain.Workflow{}, mapWorkflowWriteError("create workflow version", err)
	}

	if _, err := tx.Workflow.UpdateOneID(workflowID).
		SetVersion(current.Version + 1).
		SetCurrentVersionID(versionItem.ID).
		Save(ctx); err != nil {
		return domain.Workflow{}, mapWorkflowWriteError("update workflow current version", err)
	}

	if err := tx.Commit(); err != nil {
		return domain.Workflow{}, fmt.Errorf("commit workflow publish tx: %w", err)
	}
	return r.Get(ctx, workflowID)
}

func (r *EntRepository) ListWorkflowBoundSkillNames(ctx context.Context, workflowID uuid.UUID, enabledOnly bool) ([]string, error) {
	bindings, err := r.client.WorkflowSkillBinding.Query().
		Where(entworkflowskillbinding.WorkflowIDEQ(workflowID)).
		WithSkill().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflow skill bindings: %w", err)
	}

	names := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		if binding.Edges.Skill == nil || binding.Edges.Skill.ArchivedAt != nil {
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

func (r *EntRepository) EnsureBuiltinSkills(ctx context.Context, projectID uuid.UUID, now time.Time, bundles []domain.SkillBundle) error {
	existing, err := r.client.Skill.Query().
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

		var bundle domain.SkillBundle
		found := false
		for _, candidate := range bundles {
			if candidate.Name == template.Name {
				bundle = candidate
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("builtin skill bundle %s missing", template.Name)
		}

		tx, err := r.client.Tx(ctx)
		if err != nil {
			return fmt.Errorf("start builtin skill tx: %w", err)
		}
		defer rollback(tx)

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
			if ent.IsConstraintError(err) {
				rollback(tx)
				continue
			}
			return fmt.Errorf("create builtin skill %s: %w", template.Name, err)
		}

		versionItem, err := r.storeSkillBundleVersion(ctx, tx, skillItem.ID, 1, bundle, "builtin:openase", now)
		if err != nil {
			return fmt.Errorf("create builtin skill version %s: %w", template.Name, err)
		}

		if _, err := tx.Skill.UpdateOneID(skillItem.ID).
			SetCurrentVersionID(versionItem.ID).
			Save(ctx); err != nil {
			return fmt.Errorf("update builtin skill current version %s: %w", template.Name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit builtin skill %s: %w", template.Name, err)
		}
	}
	return nil
}

func (r *EntRepository) ListSkills(ctx context.Context, projectID uuid.UUID) ([]domain.Skill, error) {
	items, err := r.client.Skill.Query().
		Where(
			entskill.ProjectIDEQ(projectID),
			entskill.ArchivedAtIsNil(),
		).
		Order(ent.Asc(entskill.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}

	bindings, err := r.client.WorkflowSkillBinding.Query().
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

	versionBySkillID := make(map[uuid.UUID]int, len(items))
	skillIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		skillIDs = append(skillIDs, item.ID)
	}
	if len(skillIDs) > 0 {
		versions, versionErr := r.client.SkillVersion.Query().
			Where(entskillversion.SkillIDIn(skillIDs...)).
			Order(ent.Asc(entskillversion.FieldSkillID), ent.Desc(entskillversion.FieldVersion)).
			All(ctx)
		if versionErr != nil {
			return nil, fmt.Errorf("list skill versions: %w", versionErr)
		}
		for _, versionItem := range versions {
			if _, exists := versionBySkillID[versionItem.SkillID]; exists {
				continue
			}
			versionBySkillID[versionItem.SkillID] = versionItem.Version
		}
	}

	bindingsBySkillID := make(map[uuid.UUID][]domain.SkillWorkflowBinding, len(items))
	for _, binding := range bindings {
		if binding.Edges.Skill == nil || binding.Edges.Workflow == nil {
			continue
		}
		bindingsBySkillID[binding.Edges.Skill.ID] = append(bindingsBySkillID[binding.Edges.Skill.ID], domain.SkillWorkflowBinding{
			ID:          binding.Edges.Workflow.ID,
			Name:        binding.Edges.Workflow.Name,
			HarnessPath: binding.Edges.Workflow.HarnessPath,
		})
	}

	result := make([]domain.Skill, 0, len(items))
	for _, item := range items {
		workflowBindings := bindingsBySkillID[item.ID]
		sort.Slice(workflowBindings, func(i int, j int) bool {
			return workflowBindings[i].Name < workflowBindings[j].Name
		})
		result = append(result, domain.Skill{
			ID:             item.ID,
			Name:           item.Name,
			Description:    item.Description,
			Path:           skillContentRelativePath(item.Name),
			CurrentVersion: versionBySkillID[item.ID],
			IsBuiltin:      item.IsBuiltin,
			IsEnabled:      item.IsEnabled,
			CreatedBy:      item.CreatedBy,
			CreatedAt:      item.CreatedAt.UTC(),
			BoundWorkflows: workflowBindings,
		})
	}
	return result, nil
}

func (r *EntRepository) Skill(ctx context.Context, skillID uuid.UUID) (domain.SkillRecord, error) {
	item, err := r.client.Skill.Query().
		Where(
			entskill.IDEQ(skillID),
			entskill.ArchivedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.SkillRecord{}, domain.ErrSkillNotFound
		}
		return domain.SkillRecord{}, fmt.Errorf("get skill: %w", err)
	}
	return mapSkillRecord(item), nil
}

func (r *EntRepository) SkillInProject(ctx context.Context, projectID uuid.UUID, skillID uuid.UUID) (domain.SkillRecord, error) {
	item, err := r.client.Skill.Query().
		Where(
			entskill.IDEQ(skillID),
			entskill.ProjectIDEQ(projectID),
			entskill.ArchivedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.SkillRecord{}, domain.ErrSkillNotFound
		}
		return domain.SkillRecord{}, fmt.Errorf("get project skill: %w", err)
	}
	return mapSkillRecord(item), nil
}

func (r *EntRepository) SkillByName(ctx context.Context, projectID uuid.UUID, name string) (domain.SkillRecord, error) {
	item, err := r.client.Skill.Query().
		Where(
			entskill.ProjectIDEQ(projectID),
			entskill.NameEQ(name),
			entskill.ArchivedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.SkillRecord{}, domain.ErrSkillNotFound
		}
		return domain.SkillRecord{}, fmt.Errorf("get skill by name: %w", err)
	}
	return mapSkillRecord(item), nil
}

func (r *EntRepository) CurrentSkillVersion(ctx context.Context, skillID uuid.UUID, requiredVersionID *uuid.UUID) (domain.SkillVersionRecord, error) {
	query := r.client.SkillVersion.Query()
	if requiredVersionID != nil && *requiredVersionID != uuid.Nil {
		item, err := query.Where(entskillversion.IDEQ(*requiredVersionID)).Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return domain.SkillVersionRecord{}, domain.ErrSkillNotFound
			}
			return domain.SkillVersionRecord{}, fmt.Errorf("get required skill version: %w", err)
		}
		return mapSkillVersion(item), nil
	}

	item, err := query.
		Where(entskillversion.SkillIDEQ(skillID)).
		Order(ent.Desc(entskillversion.FieldVersion)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.SkillVersionRecord{}, domain.ErrSkillNotFound
		}
		return domain.SkillVersionRecord{}, fmt.Errorf("get current skill version: %w", err)
	}
	return mapSkillVersion(item), nil
}

func (r *EntRepository) ListSkillVersions(ctx context.Context, skillID uuid.UUID) ([]domain.VersionSummary, error) {
	if _, err := r.Skill(ctx, skillID); err != nil {
		return nil, err
	}
	items, err := r.client.SkillVersion.Query().
		Where(entskillversion.SkillIDEQ(skillID)).
		Order(ent.Desc(entskillversion.FieldVersion)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list skill versions: %w", err)
	}
	result := make([]domain.VersionSummary, 0, len(items))
	for _, item := range items {
		result = append(result, domain.VersionSummary{
			ID:        item.ID,
			Version:   item.Version,
			CreatedBy: item.CreatedBy,
			CreatedAt: item.CreatedAt.UTC(),
		})
	}
	return result, nil
}

func (r *EntRepository) SkillVersionFiles(ctx context.Context, versionID uuid.UUID) ([]domain.SkillBundleFile, error) {
	items, err := r.client.SkillVersionFile.Query().
		Where(entskillversionfile.SkillVersionIDEQ(versionID)).
		WithContentBlob().
		Order(ent.Asc(entskillversionfile.FieldPath)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list skill version files: %w", err)
	}

	files := make([]domain.SkillBundleFile, 0, len(items))
	for _, item := range items {
		if item.Edges.ContentBlob == nil {
			return nil, fmt.Errorf("list skill version files: content blob missing for %s", item.Path)
		}
		files = append(files, domain.SkillBundleFile{
			Path:         item.Path,
			FileKind:     item.FileKind.String(),
			MediaType:    item.MediaType,
			Encoding:     item.Encoding.String(),
			IsExecutable: item.IsExecutable,
			SizeBytes:    item.SizeBytes,
			SHA256:       item.Sha256,
			Content:      append([]byte(nil), item.Edges.ContentBlob.ContentBytes...),
		})
	}
	return files, nil
}

func (r *EntRepository) SkillDetail(ctx context.Context, skillID uuid.UUID) (domain.SkillDetail, error) {
	record, err := r.Skill(ctx, skillID)
	if err != nil {
		return domain.SkillDetail{}, err
	}
	items, err := r.ListSkills(ctx, record.ProjectID)
	if err != nil {
		return domain.SkillDetail{}, err
	}
	for _, item := range items {
		if item.ID != record.ID {
			continue
		}
		versionItem, err := r.CurrentSkillVersion(ctx, record.ID, nil)
		if err != nil {
			return domain.SkillDetail{}, err
		}
		files, err := r.SkillVersionFiles(ctx, versionItem.ID)
		if err != nil {
			return domain.SkillDetail{}, err
		}
		history, err := r.ListSkillVersions(ctx, record.ID)
		if err != nil {
			return domain.SkillDetail{}, err
		}
		return domain.SkillDetail{
			Skill:      item,
			Content:    versionItem.ContentMarkdown,
			BundleHash: versionItem.BundleHash,
			FileCount:  versionItem.FileCount,
			Files:      files,
			History:    history,
		}, nil
	}
	return domain.SkillDetail{}, domain.ErrSkillNotFound
}

func (r *EntRepository) CreateSkillBundle(
	ctx context.Context,
	input domain.CreateSkillBundleInput,
	bundle domain.SkillBundle,
	enabled bool,
	createdBy string,
	now time.Time,
) (domain.SkillDetail, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.SkillDetail{}, fmt.Errorf("start skill create tx: %w", err)
	}
	defer rollback(tx)

	skillItem, err := tx.Skill.Create().
		SetProjectID(input.ProjectID).
		SetName(bundle.Name).
		SetDescription(bundle.Description).
		SetIsBuiltin(false).
		SetIsEnabled(enabled).
		SetCreatedBy(createdBy).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return domain.SkillDetail{}, fmt.Errorf("create skill: %w", err)
	}

	versionItem, err := r.storeSkillBundleVersion(ctx, tx, skillItem.ID, 1, bundle, createdBy, now)
	if err != nil {
		return domain.SkillDetail{}, err
	}

	if _, err := tx.Skill.UpdateOneID(skillItem.ID).
		SetCurrentVersionID(versionItem.ID).
		Save(ctx); err != nil {
		return domain.SkillDetail{}, fmt.Errorf("set skill current version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.SkillDetail{}, fmt.Errorf("commit skill create tx: %w", err)
	}
	return r.SkillDetail(ctx, skillItem.ID)
}

func (r *EntRepository) UpdateSkillBundle(
	ctx context.Context,
	skillID uuid.UUID,
	bundle domain.SkillBundle,
	updatedAt time.Time,
) (domain.SkillDetail, error) {
	record, err := r.Skill(ctx, skillID)
	if err != nil {
		return domain.SkillDetail{}, err
	}
	currentVersion, err := r.CurrentSkillVersion(ctx, skillID, nil)
	if err != nil {
		return domain.SkillDetail{}, err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.SkillDetail{}, fmt.Errorf("start skill update tx: %w", err)
	}
	defer rollback(tx)

	versionItem, err := r.storeSkillBundleVersion(ctx, tx, skillID, currentVersion.Version+1, bundle, record.CreatedBy, time.Time{})
	if err != nil {
		return domain.SkillDetail{}, err
	}

	if _, err := tx.Skill.UpdateOneID(skillID).
		SetDescription(bundle.Description).
		SetCurrentVersionID(versionItem.ID).
		SetUpdatedAt(updatedAt).
		Save(ctx); err != nil {
		return domain.SkillDetail{}, fmt.Errorf("update skill metadata: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.SkillDetail{}, fmt.Errorf("commit skill update tx: %w", err)
	}
	return r.SkillDetail(ctx, skillID)
}

func (r *EntRepository) DeleteSkill(ctx context.Context, skillID uuid.UUID, deletedAt time.Time) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start skill delete tx: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.WorkflowSkillBinding.Delete().
		Where(entworkflowskillbinding.SkillIDEQ(skillID)).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete skill bindings: %w", err)
	}
	if _, err := tx.Skill.UpdateOneID(skillID).
		SetArchivedAt(deletedAt).
		SetIsEnabled(false).
		SetUpdatedAt(deletedAt).
		Save(ctx); err != nil {
		return fmt.Errorf("archive skill: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit skill delete tx: %w", err)
	}
	return nil
}

func (r *EntRepository) SetSkillEnabled(ctx context.Context, skillID uuid.UUID, enabled bool, updatedAt time.Time) (domain.SkillDetail, error) {
	if _, err := r.client.Skill.UpdateOneID(skillID).
		SetIsEnabled(enabled).
		SetUpdatedAt(updatedAt).
		Save(ctx); err != nil {
		return domain.SkillDetail{}, fmt.Errorf("update skill enabled state: %w", err)
	}
	return r.SkillDetail(ctx, skillID)
}

func (r *EntRepository) ResolveInjectedSkillNames(ctx context.Context, projectID uuid.UUID, workflowID *uuid.UUID) ([]string, error) {
	if workflowID == nil {
		items, err := r.client.Skill.Query().
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

	names, err := r.ListWorkflowBoundSkillNames(ctx, *workflowID, true)
	if err != nil {
		return nil, err
	}
	platformSkill, err := r.SkillByName(ctx, projectID, "openase-platform")
	if err != nil && !errors.Is(err, domain.ErrSkillNotFound) {
		return nil, err
	}
	if err == nil && platformSkill.IsEnabled && !slicesContainsString(names, platformSkill.Name) {
		names = append(names, platformSkill.Name)
		sort.Strings(names)
	}
	return names, nil
}

func (r *EntRepository) ApplyWorkflowSkillBindings(
	ctx context.Context,
	workflowID uuid.UUID,
	skillIDs []uuid.UUID,
	bind bool,
	content string,
	createdBy string,
) (domain.Workflow, error) {
	current, err := r.Get(ctx, workflowID)
	if err != nil {
		return domain.Workflow{}, err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Workflow{}, fmt.Errorf("start workflow skill binding tx: %w", err)
	}
	defer rollback(tx)

	for _, skillID := range skillIDs {
		if bind {
			if _, err := tx.WorkflowSkillBinding.Create().
				SetWorkflowID(workflowID).
				SetSkillID(skillID).
				Save(ctx); err != nil {
				return domain.Workflow{}, fmt.Errorf("create workflow skill binding: %w", err)
			}
			continue
		}
		if _, err := tx.WorkflowSkillBinding.Delete().
			Where(
				entworkflowskillbinding.WorkflowIDEQ(workflowID),
				entworkflowskillbinding.SkillIDEQ(skillID),
			).
			Exec(ctx); err != nil {
			return domain.Workflow{}, fmt.Errorf("delete workflow skill binding: %w", err)
		}
	}

	versionItem, err := r.createWorkflowVersionSnapshot(
		ctx,
		tx,
		current,
		current.Version+1,
		content,
		resolveCreatedBy(createdBy),
	)
	if err != nil {
		return domain.Workflow{}, mapWorkflowWriteError("create workflow version", err)
	}

	if _, err := tx.Workflow.UpdateOneID(workflowID).
		SetVersion(current.Version + 1).
		SetCurrentVersionID(versionItem.ID).
		Save(ctx); err != nil {
		return domain.Workflow{}, mapWorkflowWriteError("update workflow current version", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Workflow{}, fmt.Errorf("commit workflow skill binding tx: %w", err)
	}
	return r.Get(ctx, workflowID)
}

func (r *EntRepository) ResolveRuntimeSnapshot(ctx context.Context, workflowID uuid.UUID) (domain.RuntimeSnapshot, error) {
	if err := r.ensureWorkflowMigrated(ctx, workflowID); err != nil {
		return domain.RuntimeSnapshot{}, err
	}

	workflowItem, err := r.Get(ctx, workflowID)
	if err != nil {
		return domain.RuntimeSnapshot{}, err
	}
	workflowVersion, err := r.CurrentWorkflowVersion(ctx, workflowID)
	if err != nil {
		return domain.RuntimeSnapshot{}, err
	}
	skills, err := r.runtimeSkillSnapshots(ctx, workflowItem.ProjectID, workflowID)
	if err != nil {
		return domain.RuntimeSnapshot{}, err
	}

	projectedContent, err := projectHarnessContent(workflowVersion.ContentMarkdown)
	if err != nil {
		return domain.RuntimeSnapshot{}, err
	}

	return domain.RuntimeSnapshot{
		Workflow: domain.RuntimeWorkflowSnapshot{
			WorkflowID:            workflowItem.ID,
			VersionID:             workflowVersion.ID,
			Version:               workflowVersion.Version,
			Path:                  workflowVersion.HarnessPath,
			Content:               projectedContent,
			Name:                  workflowVersion.Name,
			Type:                  workflowVersion.Type,
			RoleSlug:              workflowVersion.RoleSlug,
			RoleName:              workflowVersion.RoleName,
			RoleDescription:       workflowVersion.RoleDescription,
			PickupStatusIDs:       append([]uuid.UUID(nil), workflowVersion.PickupStatusIDs...),
			FinishStatusIDs:       append([]uuid.UUID(nil), workflowVersion.FinishStatusIDs...),
			PlatformAccessAllowed: copyStrings(workflowVersion.PlatformAccessAllowed),
		},
		Skills: skills,
	}, nil
}

func (r *EntRepository) ResolveRecordedRuntimeSnapshot(
	ctx context.Context,
	input domain.ResolveRecordedRuntimeSnapshotInput,
) (domain.RuntimeSnapshot, error) {
	if err := r.ensureWorkflowMigrated(ctx, input.WorkflowID); err != nil {
		return domain.RuntimeSnapshot{}, err
	}

	workflowItem, err := r.Get(ctx, input.WorkflowID)
	if err != nil {
		return domain.RuntimeSnapshot{}, err
	}
	workflowVersion, err := r.RecordedWorkflowVersion(ctx, input.WorkflowID, input.WorkflowVersionID)
	if err != nil {
		return domain.RuntimeSnapshot{}, err
	}

	var skills []domain.RuntimeSkillSnapshot
	if len(input.SkillVersionIDs) == 0 {
		skills, err = r.runtimeSkillSnapshots(ctx, workflowItem.ProjectID, input.WorkflowID)
	} else {
		skills, err = r.recordedSkillSnapshots(ctx, workflowItem.ProjectID, input.SkillVersionIDs)
	}
	if err != nil {
		return domain.RuntimeSnapshot{}, err
	}

	projectedContent, err := projectHarnessContent(workflowVersion.ContentMarkdown)
	if err != nil {
		return domain.RuntimeSnapshot{}, err
	}

	return domain.RuntimeSnapshot{
		Workflow: domain.RuntimeWorkflowSnapshot{
			WorkflowID:            workflowItem.ID,
			VersionID:             workflowVersion.ID,
			Version:               workflowVersion.Version,
			Path:                  workflowVersion.HarnessPath,
			Content:               projectedContent,
			Name:                  workflowVersion.Name,
			Type:                  workflowVersion.Type,
			RoleSlug:              workflowVersion.RoleSlug,
			RoleName:              workflowVersion.RoleName,
			RoleDescription:       workflowVersion.RoleDescription,
			PickupStatusIDs:       append([]uuid.UUID(nil), workflowVersion.PickupStatusIDs...),
			FinishStatusIDs:       append([]uuid.UUID(nil), workflowVersion.FinishStatusIDs...),
			PlatformAccessAllowed: copyStrings(workflowVersion.PlatformAccessAllowed),
		},
		Skills: skills,
	}, nil
}

func (r *EntRepository) BuildHarnessTemplateData(
	ctx context.Context,
	input domain.BuildHarnessTemplateDataInput,
) (domain.HarnessTemplateData, error) {
	if err := r.ensureWorkflowMigrated(ctx, input.WorkflowID); err != nil {
		return domain.HarnessTemplateData{}, err
	}

	workflowItem, err := r.client.Workflow.Query().
		Where(entworkflow.IDEQ(input.WorkflowID)).
		WithProject().
		WithPickupStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		WithFinishStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		Only(ctx)
	if err != nil {
		return domain.HarnessTemplateData{}, mapWorkflowReadError("get workflow for harness render", err)
	}

	ticketItem, err := r.client.Ticket.Query().
		Where(entticket.IDEQ(input.TicketID)).
		WithStatus().
		WithParent().
		WithExternalLinks().
		WithRepoScopes(func(query *ent.TicketRepoScopeQuery) {
			query.WithRepo()
		}).
		WithOutgoingDependencies(func(query *ent.TicketDependencyQuery) {
			query.WithTargetTicket(func(target *ent.TicketQuery) {
				target.WithStatus()
			})
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.HarnessTemplateData{}, domain.ErrWorkflowNotFound
		}
		return domain.HarnessTemplateData{}, fmt.Errorf("get ticket for harness render: %w", err)
	}
	if ticketItem.ProjectID != workflowItem.ProjectID {
		return domain.HarnessTemplateData{}, fmt.Errorf("workflow and ticket belong to different projects")
	}

	projectItem, err := r.client.Project.Query().
		Where(entproject.IDEQ(workflowItem.ProjectID)).
		WithRepos(func(query *ent.ProjectRepoQuery) {
			query.Order(ent.Asc(entprojectrepo.FieldName))
		}).
		WithWorkflows(func(query *ent.WorkflowQuery) {
			query.
				Order(ent.Asc(entworkflow.FieldName)).
				WithPickupStatuses(func(statusQuery *ent.TicketStatusQuery) {
					statusQuery.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
				}).
				WithFinishStatuses(func(statusQuery *ent.TicketStatusQuery) {
					statusQuery.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
				})
		}).
		WithStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.HarnessTemplateData{}, domain.ErrProjectNotFound
		}
		return domain.HarnessTemplateData{}, fmt.Errorf("get project for harness render: %w", err)
	}

	projectUpdates, err := r.listHarnessProjectUpdates(ctx, projectItem.ID)
	if err != nil {
		return domain.HarnessTemplateData{}, err
	}

	agentData := domain.HarnessAgentData{}
	if input.AgentID != nil {
		agentItem, agentErr := r.client.Agent.Query().
			Where(
				entagent.IDEQ(*input.AgentID),
				entagent.ProjectIDEQ(workflowItem.ProjectID),
			).
			WithProvider().
			Only(ctx)
		if agentErr != nil {
			if ent.IsNotFound(agentErr) {
				return domain.HarnessTemplateData{}, fmt.Errorf("agent not found for workflow project")
			}
			return domain.HarnessTemplateData{}, fmt.Errorf("get agent for harness render: %w", agentErr)
		}
		agentData = mapHarnessAgent(agentItem)
	}

	attemptCount := normalizeAttemptCount(ticketItem.AttemptCount)
	maxAttempts := max(workflowItem.MaxRetryAttempts, 0)
	workspace := strings.TrimSpace(input.Workspace)
	renderTime := input.Timestamp.UTC()
	if renderTime.IsZero() {
		renderTime = time.Now().UTC()
	}

	scopedRepos, repoBranchByID := mapHarnessScopedRepos(ticketItem.Identifier, ticketItem.Edges.RepoScopes, workspace)
	allRepos := mapHarnessAllRepos(ticketItem.Identifier, projectItem.Edges.Repos, repoBranchByID, workspace)
	projectWorkflows, err := r.mapHarnessProjectWorkflows(ctx, projectItem.Edges.Workflows)
	if err != nil {
		return domain.HarnessTemplateData{}, err
	}

	return domain.HarnessTemplateData{
		Ticket: domain.HarnessTicketData{
			ID:               ticketItem.ID.String(),
			Identifier:       ticketItem.Identifier,
			Title:            ticketItem.Title,
			Description:      ticketItem.Description,
			Status:           edgeTicketStatusName(ticketItem.Edges.Status),
			Priority:         ticketItem.Priority.String(),
			Type:             ticketItem.Type.String(),
			CreatedBy:        ticketItem.CreatedBy,
			CreatedAt:        ticketItem.CreatedAt.UTC().Format(time.RFC3339),
			AttemptCount:     attemptCount,
			MaxAttempts:      maxAttempts,
			BudgetUSD:        ticketItem.BudgetUsd,
			ExternalRef:      ticketItem.ExternalRef,
			ParentIdentifier: parentIdentifier(ticketItem),
			URL:              strings.TrimSpace(input.TicketURL),
			Links:            mapHarnessTicketLinks(ticketItem.Edges.ExternalLinks),
			Dependencies:     mapHarnessDependencies(ticketItem.Edges.OutgoingDependencies),
		},
		Project: domain.HarnessProjectData{
			ID:          projectItem.ID.String(),
			Name:        projectItem.Name,
			Slug:        projectItem.Slug,
			Description: projectItem.Description,
			Status:      projectItem.Status,
			Workflows:   projectWorkflows,
			Statuses:    mapHarnessProjectStatuses(projectItem.Edges.Statuses),
			Machines:    mapHarnessProjectMachines(input.Machine, input.AccessibleMachines),
			Updates:     projectUpdates,
		},
		Repos:              scopedRepos,
		AllRepos:           allRepos,
		Agent:              agentData,
		Machine:            cloneHarnessMachine(input.Machine),
		AccessibleMachines: cloneAccessibleMachines(input.AccessibleMachines),
		Attempt:            attemptCount,
		MaxAttempts:        maxAttempts,
		Workspace:          workspace,
		Timestamp:          renderTime.Format(time.RFC3339),
		OpenASEVersion:     strings.TrimSpace(input.OpenASEVersion),
		Workflow: domain.HarnessWorkflowData{
			Name:         workflowItem.Name,
			Type:         workflowItem.Type,
			RoleName:     workflowRoleName(mapWorkflow(workflowItem)),
			PickupStatus: joinStatusNames(workflowItem.Edges.PickupStatuses),
			FinishStatus: joinStatusNames(workflowItem.Edges.FinishStatuses),
		},
		Platform: normalizePlatformData(input.Platform, workflowItem.ProjectID, ticketItem.ID),
	}, nil
}

func (r *EntRepository) storeSkillBundleVersion(
	ctx context.Context,
	tx *ent.Tx,
	skillID uuid.UUID,
	version int,
	bundle domain.SkillBundle,
	createdBy string,
	createdAt time.Time,
) (*ent.SkillVersion, error) {
	versionCreate := tx.SkillVersion.Create().
		SetSkillID(skillID).
		SetVersion(version).
		SetContentMarkdown(bundle.EntrypointBody).
		SetContentHash(bundle.EntrypointSHA256).
		SetBundleHash(bundle.BundleHash).
		SetManifestJSON(bundle.Manifest).
		SetSizeBytes(bundle.SizeBytes).
		SetFileCount(bundle.FileCount).
		SetCreatedBy(createdBy)
	if !createdAt.IsZero() {
		versionCreate.SetCreatedAt(createdAt)
	}
	versionItem, err := versionCreate.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create skill bundle version: %w", err)
	}

	for _, file := range bundle.Files {
		blobID, err := r.ensureSkillBlob(ctx, tx, file)
		if err != nil {
			return nil, err
		}
		fileCreate := tx.SkillVersionFile.Create().
			SetSkillVersionID(versionItem.ID).
			SetContentBlobID(blobID).
			SetPath(file.Path).
			SetFileKind(entskillversionfile.FileKind(file.FileKind)).
			SetMediaType(file.MediaType).
			SetEncoding(entskillversionfile.Encoding(file.Encoding)).
			SetIsExecutable(file.IsExecutable).
			SetSizeBytes(file.SizeBytes).
			SetSha256(file.SHA256)
		if !createdAt.IsZero() {
			fileCreate.SetCreatedAt(createdAt)
		}
		if _, err := fileCreate.Save(ctx); err != nil {
			return nil, fmt.Errorf("create skill version file %s: %w", file.Path, err)
		}
	}
	return versionItem, nil
}

func (r *EntRepository) ensureSkillBlob(ctx context.Context, tx *ent.Tx, file domain.SkillBundleFile) (uuid.UUID, error) {
	existing, err := tx.SkillBlob.Query().
		Where(entskillblob.Sha256EQ(file.SHA256)).
		Only(ctx)
	if err == nil {
		return existing.ID, nil
	}
	if err != nil && !ent.IsNotFound(err) {
		return uuid.Nil, fmt.Errorf("query skill blob %s: %w", file.Path, err)
	}

	blobItem, err := tx.SkillBlob.Create().
		SetSha256(file.SHA256).
		SetSizeBytes(file.SizeBytes).
		SetCompression(entskillblob.CompressionNone).
		SetContentBytes(file.Content).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			existing, retryErr := tx.SkillBlob.Query().
				Where(entskillblob.Sha256EQ(file.SHA256)).
				Only(ctx)
			if retryErr != nil {
				return uuid.Nil, fmt.Errorf("reload conflicted skill blob %s: %w", file.Path, retryErr)
			}
			return existing.ID, nil
		}
		return uuid.Nil, fmt.Errorf("create skill blob %s: %w", file.Path, err)
	}
	return blobItem.ID, nil
}

func (r *EntRepository) runtimeSkillSnapshots(
	ctx context.Context,
	projectID uuid.UUID,
	workflowID uuid.UUID,
) ([]domain.RuntimeSkillSnapshot, error) {
	bindings, err := r.client.WorkflowSkillBinding.Query().
		Where(entworkflowskillbinding.WorkflowIDEQ(workflowID)).
		Order(ent.Asc(entworkflowskillbinding.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list runtime workflow skill bindings: %w", err)
	}

	snapshots := make([]domain.RuntimeSkillSnapshot, 0, len(bindings)+1)
	seen := map[uuid.UUID]struct{}{}
	for _, binding := range bindings {
		skillItem, err := r.Skill(ctx, binding.SkillID)
		if err != nil {
			if errors.Is(err, domain.ErrSkillNotFound) {
				continue
			}
			return nil, err
		}
		if skillItem.ProjectID != projectID || !skillItem.IsEnabled || skillItem.ArchivedAt != nil {
			continue
		}

		versionItem, err := r.CurrentSkillVersion(ctx, skillItem.ID, binding.RequiredVersionID)
		if err != nil {
			return nil, err
		}
		files, err := r.runtimeSkillFiles(ctx, versionItem.ID)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, domain.RuntimeSkillSnapshot{
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

	platformSkill, err := r.SkillByName(ctx, projectID, "openase-platform")
	if err != nil && !errors.Is(err, domain.ErrSkillNotFound) {
		return nil, err
	}
	if err == nil && platformSkill.IsEnabled && platformSkill.ArchivedAt == nil {
		if _, ok := seen[platformSkill.ID]; !ok {
			versionItem, versionErr := r.CurrentSkillVersion(ctx, platformSkill.ID, nil)
			if versionErr != nil {
				return nil, versionErr
			}
			files, filesErr := r.runtimeSkillFiles(ctx, versionItem.ID)
			if filesErr != nil {
				return nil, filesErr
			}
			snapshots = append(snapshots, domain.RuntimeSkillSnapshot{
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

func (r *EntRepository) recordedSkillSnapshots(
	ctx context.Context,
	projectID uuid.UUID,
	versionIDs []uuid.UUID,
) ([]domain.RuntimeSkillSnapshot, error) {
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

	snapshots := make([]domain.RuntimeSkillSnapshot, 0, len(ordered))
	for _, versionID := range ordered {
		versionItem, err := r.client.SkillVersion.Query().
			Where(entskillversion.IDEQ(versionID)).
			WithSkill().
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return nil, fmt.Errorf("%w: recorded skill version %s not found", domain.ErrSkillNotFound, versionID)
			}
			return nil, fmt.Errorf("get recorded skill version: %w", err)
		}
		if versionItem.Edges.Skill == nil || versionItem.Edges.Skill.ProjectID != projectID {
			return nil, fmt.Errorf("%w: recorded skill version %s is outside project %s", domain.ErrSkillInvalid, versionID, projectID)
		}
		files, err := r.runtimeSkillFiles(ctx, versionItem.ID)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, domain.RuntimeSkillSnapshot{
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

func (r *EntRepository) runtimeSkillFiles(ctx context.Context, versionID uuid.UUID) ([]domain.RuntimeSkillFileSnapshot, error) {
	files, err := r.SkillVersionFiles(ctx, versionID)
	if err != nil {
		return nil, err
	}
	snapshots := make([]domain.RuntimeSkillFileSnapshot, 0, len(files))
	for _, file := range files {
		snapshots = append(snapshots, domain.RuntimeSkillFileSnapshot{
			Path:         file.Path,
			Content:      append([]byte(nil), file.Content...),
			IsExecutable: file.IsExecutable,
		})
	}
	return snapshots, nil
}

func mapWorkflow(item *ent.Workflow) domain.Workflow {
	return domain.Workflow{
		ID:                    item.ID,
		ProjectID:             item.ProjectID,
		AgentID:               item.AgentID,
		Name:                  item.Name,
		Type:                  domain.Type(item.Type),
		RoleSlug:              strings.TrimSpace(item.RoleSlug),
		RoleName:              strings.TrimSpace(item.RoleName),
		RoleDescription:       strings.TrimSpace(item.RoleDescription),
		PlatformAccessAllowed: copyStrings(item.PlatformAccessAllowed),
		HarnessPath:           item.HarnessPath,
		Hooks:                 copyHooks(item.Hooks),
		MaxConcurrent:         item.MaxConcurrent,
		MaxRetryAttempts:      item.MaxRetryAttempts,
		TimeoutMinutes:        item.TimeoutMinutes,
		StallTimeoutMinutes:   item.StallTimeoutMinutes,
		Version:               item.Version,
		IsActive:              item.IsActive,
		PickupStatusIDs:       statusIDsFromEdges(item.Edges.PickupStatuses),
		FinishStatusIDs:       statusIDsFromEdges(item.Edges.FinishStatuses),
	}
}

func mapWorkflowVersion(item *ent.WorkflowVersion) domain.WorkflowVersionRecord {
	return domain.WorkflowVersionRecord{
		ID:                    item.ID,
		WorkflowID:            item.WorkflowID,
		Version:               item.Version,
		ContentMarkdown:       item.ContentMarkdown,
		Name:                  item.Name,
		Type:                  domain.Type(item.Type),
		RoleSlug:              strings.TrimSpace(item.RoleSlug),
		RoleName:              strings.TrimSpace(item.RoleName),
		RoleDescription:       strings.TrimSpace(item.RoleDescription),
		PickupStatusIDs:       parseWorkflowVersionStatusIDs(item.PickupStatusIds),
		FinishStatusIDs:       parseWorkflowVersionStatusIDs(item.FinishStatusIds),
		HarnessPath:           item.HarnessPath,
		Hooks:                 copyHooks(item.Hooks),
		PlatformAccessAllowed: copyStrings(item.PlatformAccessAllowed),
		MaxConcurrent:         item.MaxConcurrent,
		MaxRetryAttempts:      item.MaxRetryAttempts,
		TimeoutMinutes:        item.TimeoutMinutes,
		StallTimeoutMinutes:   item.StallTimeoutMinutes,
		IsActive:              item.IsActive,
		ContentHash:           item.ContentHash,
		CreatedBy:             item.CreatedBy,
		CreatedAt:             item.CreatedAt.UTC(),
	}
}

func mapSkillRecord(item *ent.Skill) domain.SkillRecord {
	var currentVersionID *uuid.UUID
	if item.CurrentVersionID != nil {
		cloned := *item.CurrentVersionID
		currentVersionID = &cloned
	}
	return domain.SkillRecord{
		ID:               item.ID,
		ProjectID:        item.ProjectID,
		Name:             item.Name,
		Description:      item.Description,
		CurrentVersionID: currentVersionID,
		IsBuiltin:        item.IsBuiltin,
		IsEnabled:        item.IsEnabled,
		CreatedBy:        item.CreatedBy,
		CreatedAt:        item.CreatedAt.UTC(),
		UpdatedAt:        item.UpdatedAt.UTC(),
		ArchivedAt:       cloneTime(item.ArchivedAt),
	}
}

func mapSkillVersion(item *ent.SkillVersion) domain.SkillVersionRecord {
	return domain.SkillVersionRecord{
		ID:              item.ID,
		SkillID:         item.SkillID,
		Version:         item.Version,
		CreatedBy:       item.CreatedBy,
		CreatedAt:       item.CreatedAt.UTC(),
		ContentMarkdown: item.ContentMarkdown,
		BundleHash:      strings.TrimSpace(item.BundleHash),
		FileCount:       item.FileCount,
	}
}

func mapWorkflowReadError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return domain.ErrWorkflowNotFound
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func mapWorkflowWriteError(action string, err error) error {
	var impactConflict *domain.WorkflowImpactConflict
	switch {
	case ent.IsNotFound(err):
		return domain.ErrWorkflowNotFound
	case ent.IsConstraintError(err):
		return fmt.Errorf("%w: workflow name already exists in this project", domain.ErrWorkflowNameConflict)
	case errors.Is(err, domain.ErrWorkflowReplacementRequired):
		return err
	case errors.Is(err, domain.ErrWorkflowActiveAgentRuns):
		return err
	case errors.Is(err, domain.ErrWorkflowHistoricalAgentRuns):
		return err
	case errors.Is(err, domain.ErrWorkflowReplacementInvalid):
		return err
	case errors.Is(err, domain.ErrWorkflowReplacementNotFound):
		return err
	case errors.Is(err, domain.ErrWorkflowReplacementProjectMismatch):
		return err
	case errors.Is(err, domain.ErrWorkflowReplacementInactive):
		return err
	case errors.As(err, &impactConflict):
		return err
	case strings.Contains(strings.ToLower(err.Error()), "tickets"):
		return fmt.Errorf("%w: tickets still reference this workflow", domain.ErrWorkflowReferencedByTickets)
	case strings.Contains(strings.ToLower(err.Error()), "scheduled_jobs"):
		return fmt.Errorf("%w: scheduled jobs still reference this workflow", domain.ErrWorkflowReferencedByScheduledJobs)
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func workflowDeleteConflictError(impact domain.WorkflowImpactAnalysis) error {
	switch {
	case impact.Summary.ActiveAgentRunCount > 0:
		return domain.ErrWorkflowActiveAgentRuns
	case impact.Summary.HistoricalAgentRunCount > 0:
		return domain.ErrWorkflowHistoricalAgentRuns
	case impact.Summary.ReplaceableReferenceCount > 0:
		return domain.ErrWorkflowReplacementRequired
	case impact.Summary.BlockingReferenceCount > 0:
		return domain.ErrWorkflowInUse
	default:
		return nil
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

func rollback(tx *ent.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum[:])
}

func skillContentRelativePath(name string) string {
	return filepath.ToSlash(filepath.Join(".openase", "skills", name, "SKILL.md"))
}

func slicesContainsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func projectHarnessContent(content string) (string, error) {
	return normalizeHarnessNewlines(content), nil
}

func normalizeSkillNames(raw []string) ([]string, error) {
	normalized := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: skill name must not be empty", domain.ErrSkillInvalid)
		}
		if !skillNamePattern.MatchString(trimmed) {
			return nil, fmt.Errorf("%w: skill name %q must match %s", domain.ErrSkillInvalid, trimmed, skillNamePattern.String())
		}
		if !slicesContainsString(normalized, trimmed) {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized, nil
}

func (r *EntRepository) projectedWorkflowHarness(ctx context.Context, workflowItem domain.Workflow) (string, error) {
	version, err := r.CurrentWorkflowVersion(ctx, workflowItem.ID)
	if err != nil {
		return "", err
	}
	return projectHarnessContent(version.ContentMarkdown)
}

func mapHarnessScopedRepos(ticketIdentifier string, scopes []*ent.TicketRepoScope, workspace string) ([]domain.HarnessRepoData, map[uuid.UUID]string) {
	repos := make([]domain.HarnessRepoData, 0, len(scopes))
	branches := make(map[uuid.UUID]string, len(scopes))
	for _, scope := range scopes {
		repo := scope.Edges.Repo
		if repo == nil {
			continue
		}
		effectiveBranchName := ticketingdomain.ResolveRepoWorkBranch(ticketIdentifier, scope.BranchName)
		branches[repo.ID] = effectiveBranchName
		repos = append(repos, domain.HarnessRepoData{
			Name:          repo.Name,
			URL:           repo.RepositoryURL,
			Path:          resolveRepoPath(repo.WorkspaceDirname, workspace, repo.Name),
			Branch:        effectiveBranchName,
			DefaultBranch: repo.DefaultBranch,
			Labels:        append([]string(nil), repo.Labels...),
		})
	}
	return repos, branches
}

func mapHarnessAllRepos(ticketIdentifier string, repos []*ent.ProjectRepo, repoBranchByID map[uuid.UUID]string, workspace string) []domain.HarnessRepoData {
	items := make([]domain.HarnessRepoData, 0, len(repos))
	for _, repo := range repos {
		branch := repoBranchByID[repo.ID]
		if branch == "" {
			branch = ticketingdomain.DefaultRepoWorkBranch(ticketIdentifier)
		}
		items = append(items, domain.HarnessRepoData{
			Name:          repo.Name,
			URL:           repo.RepositoryURL,
			Path:          resolveRepoPath(repo.WorkspaceDirname, workspace, repo.Name),
			Branch:        branch,
			DefaultBranch: repo.DefaultBranch,
			Labels:        append([]string(nil), repo.Labels...),
		})
	}
	return items
}

func (r *EntRepository) mapHarnessProjectWorkflows(
	ctx context.Context,
	workflows []*ent.Workflow,
) ([]domain.HarnessProjectWorkflowData, error) {
	items := make([]domain.HarnessProjectWorkflowData, 0, len(workflows))
	workflowIDs := make([]uuid.UUID, 0, len(workflows))
	for _, workflowItem := range workflows {
		if workflowItem == nil || !workflowItem.IsActive {
			continue
		}
		workflowIDs = append(workflowIDs, workflowItem.ID)
	}

	activeCountByWorkflow := make(map[uuid.UUID]int, len(workflowIDs))
	if len(workflowIDs) > 0 {
		activeTickets, err := r.client.Ticket.Query().
			Where(
				entticket.WorkflowIDIn(workflowIDs...),
				entticket.CurrentRunIDNotNil(),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list active workflow tickets for harness render: %w", err)
		}
		for _, ticketItem := range activeTickets {
			if ticketItem.WorkflowID == nil {
				continue
			}
			activeCountByWorkflow[*ticketItem.WorkflowID]++
		}
	}

	for _, workflowItem := range workflows {
		if workflowItem == nil || !workflowItem.IsActive {
			continue
		}
		skills, err := r.ListWorkflowBoundSkillNames(ctx, workflowItem.ID, false)
		if err != nil {
			return nil, fmt.Errorf("load workflow skills for project context: %w", err)
		}
		recentTickets, err := r.listHarnessWorkflowRecentTickets(ctx, workflowItem.ID, 5)
		if err != nil {
			return nil, err
		}
		items = append(items, domain.HarnessProjectWorkflowData{
			Name:            workflowItem.Name,
			Type:            workflowItem.Type,
			RoleName:        workflowRoleName(mapWorkflow(workflowItem)),
			RoleDescription: strings.TrimSpace(workflowItem.RoleDescription),
			PickupStatus:    joinStatusNames(workflowItem.Edges.PickupStatuses),
			FinishStatus:    joinStatusNames(workflowItem.Edges.FinishStatuses),
			PickupStatuses:  mapHarnessProjectStatuses(workflowItem.Edges.PickupStatuses),
			FinishStatuses:  mapHarnessProjectStatuses(workflowItem.Edges.FinishStatuses),
			HarnessPath:     workflowItem.HarnessPath,
			HarnessContent:  "",
			Skills:          skills,
			MaxConcurrent:   workflowItem.MaxConcurrent,
			CurrentActive:   activeCountByWorkflow[workflowItem.ID],
			RecentTickets:   recentTickets,
		})
	}
	return items, nil
}

func joinStatusNames(statuses []*ent.TicketStatus) string {
	names := make([]string, 0, len(statuses))
	for _, status := range statuses {
		names = append(names, status.Name)
	}
	return strings.Join(names, ", ")
}

func (r *EntRepository) listHarnessWorkflowRecentTickets(
	ctx context.Context,
	workflowID uuid.UUID,
	limit int,
) ([]domain.HarnessProjectWorkflowTicketData, error) {
	query := r.client.Ticket.Query().
		Where(entticket.WorkflowIDEQ(workflowID)).
		Order(ent.Desc(entticket.FieldCreatedAt)).
		WithStatus()
	if limit > 0 {
		query = query.Limit(limit)
	}

	items, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflow history for harness render: %w", err)
	}

	tickets := make([]domain.HarnessProjectWorkflowTicketData, 0, len(items))
	for _, item := range items {
		tickets = append(tickets, mapHarnessProjectWorkflowTicket(item))
	}
	return tickets, nil
}

func (r *EntRepository) listHarnessProjectUpdates(
	ctx context.Context,
	projectID uuid.UUID,
) ([]domain.HarnessProjectUpdateThreadData, error) {
	items, err := r.client.ProjectUpdateThread.Query().
		Where(
			entprojectupdatethread.ProjectIDEQ(projectID),
			entprojectupdatethread.IsDeleted(false),
		).
		Order(ent.Desc(entprojectupdatethread.FieldLastActivityAt), ent.Desc(entprojectupdatethread.FieldID)).
		WithComments(func(query *ent.ProjectUpdateCommentQuery) {
			query.Where(entprojectupdatecomment.IsDeleted(false))
			query.Order(ent.Asc(entprojectupdatecomment.FieldCreatedAt), ent.Asc(entprojectupdatecomment.FieldID))
		}).
		Limit(10).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project updates for harness render: %w", err)
	}

	result := make([]domain.HarnessProjectUpdateThreadData, 0, len(items))
	for _, item := range items {
		comments := make([]domain.HarnessProjectUpdateCommentData, 0, len(item.Edges.Comments))
		for _, comment := range item.Edges.Comments {
			if comment == nil {
				continue
			}
			comments = append(comments, domain.HarnessProjectUpdateCommentData{
				ID:           comment.ID.String(),
				BodyMarkdown: comment.BodyMarkdown,
				CreatedBy:    comment.CreatedBy,
				CreatedAt:    comment.CreatedAt.UTC().Format(time.RFC3339),
				UpdatedAt:    comment.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}
		result = append(result, domain.HarnessProjectUpdateThreadData{
			ID:             item.ID.String(),
			Status:         item.Status.String(),
			Title:          item.Title,
			BodyMarkdown:   item.BodyMarkdown,
			CreatedBy:      item.CreatedBy,
			CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
			LastActivityAt: item.LastActivityAt.UTC().Format(time.RFC3339),
			CommentCount:   item.CommentCount,
			Comments:       comments,
		})
	}
	return result, nil
}

func mapHarnessProjectStatuses(statuses []*ent.TicketStatus) []domain.HarnessProjectStatusData {
	items := make([]domain.HarnessProjectStatusData, 0, len(statuses))
	for _, status := range statuses {
		if status == nil {
			continue
		}
		items = append(items, domain.HarnessProjectStatusData{
			ID:    status.ID.String(),
			Name:  status.Name,
			Stage: status.Stage.String(),
			Color: status.Color,
		})
	}
	return items
}

func mapHarnessAgent(item *ent.Agent) domain.HarnessAgentData {
	if item == nil {
		return domain.HarnessAgentData{}
	}
	providerName := ""
	adapterType := ""
	modelName := ""
	if item.Edges.Provider != nil {
		providerName = item.Edges.Provider.Name
		adapterType = item.Edges.Provider.AdapterType.String()
		modelName = item.Edges.Provider.ModelName
	}
	return domain.HarnessAgentData{
		ID:                    item.ID.String(),
		Name:                  item.Name,
		Provider:              providerName,
		AdapterType:           adapterType,
		Model:                 modelName,
		TotalTicketsCompleted: item.TotalTicketsCompleted,
	}
}

func mapHarnessTicketLinks(links []*ent.TicketExternalLink) []domain.HarnessTicketLinkData {
	items := make([]domain.HarnessTicketLinkData, 0, len(links))
	for _, link := range links {
		items = append(items, domain.HarnessTicketLinkData{
			Type:     link.LinkType.String(),
			URL:      link.URL,
			Title:    link.Title,
			Status:   link.Status,
			Relation: link.Relation.String(),
		})
	}
	return items
}

func mapHarnessDependencies(dependencies []*ent.TicketDependency) []domain.HarnessTicketDependencyData {
	items := make([]domain.HarnessTicketDependencyData, 0, len(dependencies))
	for _, dependency := range dependencies {
		target := dependency.Edges.TargetTicket
		if target == nil {
			continue
		}
		items = append(items, domain.HarnessTicketDependencyData{
			Identifier: target.Identifier,
			Title:      target.Title,
			Type:       normalizeDependencyType(dependency.Type),
			Status:     edgeTicketStatusName(target.Edges.Status),
		})
	}
	return items
}

func edgeTicketStatusName(status *ent.TicketStatus) string {
	if status == nil {
		return ""
	}
	return status.Name
}

func parentIdentifier(ticketItem *ent.Ticket) string {
	if ticketItem == nil || ticketItem.Edges.Parent == nil {
		return ""
	}
	return ticketItem.Edges.Parent.Identifier
}

func normalizeDependencyType(value entticketdependency.Type) string {
	return strings.ReplaceAll(value.String(), "-", "_")
}

func mapHarnessProjectWorkflowTicket(item *ent.Ticket) domain.HarnessProjectWorkflowTicketData {
	return domain.HarnessProjectWorkflowTicketData{
		Identifier:        item.Identifier,
		Title:             item.Title,
		Status:            edgeTicketStatusName(item.Edges.Status),
		Priority:          item.Priority.String(),
		Type:              item.Type.String(),
		AttemptCount:      normalizeAttemptCount(item.AttemptCount),
		ConsecutiveErrors: item.ConsecutiveErrors,
		RetryPaused:       item.RetryPaused,
		PauseReason:       item.PauseReason,
		CreatedAt:         item.CreatedAt.UTC().Format(time.RFC3339),
		StartedAt:         formatOptionalTime(item.StartedAt),
		CompletedAt:       formatOptionalTime(item.CompletedAt),
	}
}

func normalizeAttemptCount(raw int) int {
	if raw < 1 {
		return 1
	}
	return raw
}

func normalizePlatformData(input domain.HarnessPlatformData, projectID uuid.UUID, ticketID uuid.UUID) domain.HarnessPlatformData {
	platform := input
	if strings.TrimSpace(platform.ProjectID) == "" {
		platform.ProjectID = projectID.String()
	}
	if strings.TrimSpace(platform.TicketID) == "" {
		platform.TicketID = ticketID.String()
	}
	return platform
}

func resolveRepoPath(workspaceDirname string, workspace string, repoName string) string {
	if trimmed := strings.TrimSpace(workspaceDirname); trimmed != "" {
		return trimmed
	}
	if strings.TrimSpace(workspace) == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(workspace, repoName))
}

func workflowRoleName(item domain.Workflow) string {
	if trimmed := strings.TrimSpace(item.RoleName); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(item.Name)
}

func cloneHarnessMachine(machine domain.HarnessMachineData) domain.HarnessMachineData {
	machine.Labels = append([]string(nil), machine.Labels...)
	machine.Resources = cloneAnyMap(machine.Resources)
	return machine
}

func cloneAccessibleMachines(machines []domain.HarnessAccessibleMachineData) []domain.HarnessAccessibleMachineData {
	cloned := make([]domain.HarnessAccessibleMachineData, 0, len(machines))
	for _, machine := range machines {
		machine.Labels = append([]string(nil), machine.Labels...)
		machine.Resources = cloneAnyMap(machine.Resources)
		cloned = append(cloned, machine)
	}
	return cloned
}

func mapHarnessProjectMachines(
	selected domain.HarnessMachineData,
	accessible []domain.HarnessAccessibleMachineData,
) []domain.HarnessProjectMachineData {
	items := make([]domain.HarnessProjectMachineData, 0, len(accessible)+1)
	seen := make(map[string]struct{}, len(accessible)+1)
	add := func(name string, host string, description string, labels []string, status string, resources map[string]any) {
		key := strings.TrimSpace(name) + "|" + strings.TrimSpace(host)
		if key == "|" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		items = append(items, domain.HarnessProjectMachineData{
			Name:        name,
			Host:        host,
			Description: description,
			Labels:      append([]string(nil), labels...),
			Status:      status,
			Resources:   cloneAnyMap(resources),
		})
	}

	add(selected.Name, selected.Host, selected.Description, selected.Labels, "current", selected.Resources)
	for _, machine := range accessible {
		add(machine.Name, machine.Host, machine.Description, machine.Labels, "accessible", machine.Resources)
	}
	sort.Slice(items, func(i int, j int) bool {
		if compared := strings.Compare(items[i].Name, items[j].Name); compared != 0 {
			return compared < 0
		}
		return strings.Compare(items[i].Host, items[j].Host) < 0
	})
	return items
}

func cloneAnyMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}
	return cloned
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func resolveCreatedBy(raw string) string {
	createdBy := strings.TrimSpace(raw)
	if createdBy == "" {
		return "system:workflow-service"
	}
	return createdBy
}
