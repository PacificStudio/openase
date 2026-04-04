package workflow

import (
	"context"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/google/uuid"
)

type ProjectValidationRepository interface {
	EnsureProjectExists(ctx context.Context, projectID uuid.UUID) error
	EnsureAgentBelongsToProject(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID) error
	EnsureStatusBindingsBelongToProject(ctx context.Context, projectID uuid.UUID, statusIDs []uuid.UUID) error
	EnsureWorkflowNameAvailable(ctx context.Context, projectID uuid.UUID, name string, excludeWorkflowID uuid.UUID) error
	EnsurePickupStatusBindingsAvailable(ctx context.Context, projectID uuid.UUID, statusIDs []uuid.UUID, excludeWorkflowID uuid.UUID) error
	EnsureHarnessPathAvailable(ctx context.Context, projectID uuid.UUID, harnessPath string, excludeWorkflowID uuid.UUID) error
	StatusNames(ctx context.Context, statusIDs []uuid.UUID) ([]string, error)
}

type WorkflowRepository interface {
	List(ctx context.Context, projectID uuid.UUID) ([]domain.Workflow, error)
	Get(ctx context.Context, workflowID uuid.UUID) (domain.Workflow, error)
	Create(ctx context.Context, workflow domain.Workflow, harnessContent string, createdBy string) (domain.Workflow, error)
	Update(ctx context.Context, workflow domain.Workflow) (domain.Workflow, error)
	ImpactAnalysis(ctx context.Context, workflowID uuid.UUID) (domain.WorkflowImpactAnalysis, error)
	ReplaceReferences(ctx context.Context, input domain.ReplaceWorkflowReferencesInput) (domain.ReplaceWorkflowReferencesResult, error)
	Delete(ctx context.Context, workflowID uuid.UUID) (domain.Workflow, error)
}

type WorkflowVersionRepository interface {
	CurrentWorkflowVersion(ctx context.Context, workflowID uuid.UUID) (domain.WorkflowVersionRecord, error)
	RecordedWorkflowVersion(ctx context.Context, workflowID uuid.UUID, workflowVersionID *uuid.UUID) (domain.WorkflowVersionRecord, error)
	ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID) ([]domain.VersionSummary, error)
	PublishWorkflowVersion(ctx context.Context, workflowID uuid.UUID, content string, createdBy string) (domain.Workflow, error)
}

type SkillRepository interface {
	EnsureBuiltinSkills(ctx context.Context, projectID uuid.UUID, now time.Time, bundles []domain.SkillBundle) error
	ListSkills(ctx context.Context, projectID uuid.UUID) ([]domain.Skill, error)
	Skill(ctx context.Context, skillID uuid.UUID) (domain.SkillRecord, error)
	SkillInProject(ctx context.Context, projectID uuid.UUID, skillID uuid.UUID) (domain.SkillRecord, error)
	SkillByName(ctx context.Context, projectID uuid.UUID, name string) (domain.SkillRecord, error)
	SkillDetail(ctx context.Context, skillID uuid.UUID) (domain.SkillDetail, error)
	CreateSkillBundle(
		ctx context.Context,
		input domain.CreateSkillBundleInput,
		bundle domain.SkillBundle,
		enabled bool,
		createdBy string,
		now time.Time,
	) (domain.SkillDetail, error)
	UpdateSkillBundle(ctx context.Context, skillID uuid.UUID, bundle domain.SkillBundle, updatedAt time.Time) (domain.SkillDetail, error)
	DeleteSkill(ctx context.Context, skillID uuid.UUID, deletedAt time.Time) error
	SetSkillEnabled(ctx context.Context, skillID uuid.UUID, enabled bool, updatedAt time.Time) (domain.SkillDetail, error)
}

type SkillVersionRepository interface {
	CurrentSkillVersion(ctx context.Context, skillID uuid.UUID, requiredVersionID *uuid.UUID) (domain.SkillVersionRecord, error)
	ListSkillVersions(ctx context.Context, skillID uuid.UUID) ([]domain.VersionSummary, error)
	SkillVersionFiles(ctx context.Context, versionID uuid.UUID) ([]domain.SkillBundleFile, error)
}

type WorkflowSkillBindingRepository interface {
	ListWorkflowBoundSkillNames(ctx context.Context, workflowID uuid.UUID, enabledOnly bool) ([]string, error)
	ResolveInjectedSkillNames(ctx context.Context, projectID uuid.UUID, workflowID *uuid.UUID) ([]string, error)
	ApplyWorkflowSkillBindings(ctx context.Context, workflowID uuid.UUID, skillIDs []uuid.UUID, bind bool, content string, createdBy string) (domain.Workflow, error)
}

type WorkflowRuntimeSnapshotReader interface {
	ResolveRuntimeSnapshot(ctx context.Context, workflowID uuid.UUID) (domain.RuntimeSnapshot, error)
	ResolveRecordedRuntimeSnapshot(ctx context.Context, input domain.ResolveRecordedRuntimeSnapshotInput) (domain.RuntimeSnapshot, error)
}

type HarnessTemplateDataBuilder interface {
	BuildHarnessTemplateData(ctx context.Context, input domain.BuildHarnessTemplateDataInput) (domain.HarnessTemplateData, error)
}
