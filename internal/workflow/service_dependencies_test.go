package workflow

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/google/uuid"
)

type stubProjectValidationRepository struct {
	ensureProjectExists                 func(context.Context, uuid.UUID) error
	ensureAgentBelongsToProject         func(context.Context, uuid.UUID, uuid.UUID) error
	ensureStatusBindingsBelongToProject func(context.Context, uuid.UUID, []uuid.UUID) error
	ensureWorkflowNameAvailable         func(context.Context, uuid.UUID, string, uuid.UUID) error
	ensurePickupStatusBindingsAvailable func(context.Context, uuid.UUID, []uuid.UUID, uuid.UUID) error
	ensureHarnessPathAvailable          func(context.Context, uuid.UUID, string, uuid.UUID) error
	statusNames                         func(context.Context, []uuid.UUID) ([]string, error)
}

func (s stubProjectValidationRepository) EnsureProjectExists(ctx context.Context, projectID uuid.UUID) error {
	if s.ensureProjectExists != nil {
		return s.ensureProjectExists(ctx, projectID)
	}
	return nil
}

func (s stubProjectValidationRepository) EnsureAgentBelongsToProject(ctx context.Context, projectID uuid.UUID, agentID uuid.UUID) error {
	if s.ensureAgentBelongsToProject != nil {
		return s.ensureAgentBelongsToProject(ctx, projectID, agentID)
	}
	return nil
}

func (s stubProjectValidationRepository) EnsureStatusBindingsBelongToProject(ctx context.Context, projectID uuid.UUID, statusIDs []uuid.UUID) error {
	if s.ensureStatusBindingsBelongToProject != nil {
		return s.ensureStatusBindingsBelongToProject(ctx, projectID, statusIDs)
	}
	return nil
}

func (s stubProjectValidationRepository) EnsureWorkflowNameAvailable(
	ctx context.Context,
	projectID uuid.UUID,
	name string,
	excludeWorkflowID uuid.UUID,
) error {
	if s.ensureWorkflowNameAvailable != nil {
		return s.ensureWorkflowNameAvailable(ctx, projectID, name, excludeWorkflowID)
	}
	return nil
}

func (s stubProjectValidationRepository) EnsurePickupStatusBindingsAvailable(
	ctx context.Context,
	projectID uuid.UUID,
	statusIDs []uuid.UUID,
	excludeWorkflowID uuid.UUID,
) error {
	if s.ensurePickupStatusBindingsAvailable != nil {
		return s.ensurePickupStatusBindingsAvailable(ctx, projectID, statusIDs, excludeWorkflowID)
	}
	return nil
}

func (s stubProjectValidationRepository) EnsureHarnessPathAvailable(
	ctx context.Context,
	projectID uuid.UUID,
	harnessPath string,
	excludeWorkflowID uuid.UUID,
) error {
	if s.ensureHarnessPathAvailable != nil {
		return s.ensureHarnessPathAvailable(ctx, projectID, harnessPath, excludeWorkflowID)
	}
	return nil
}

func (s stubProjectValidationRepository) StatusNames(ctx context.Context, statusIDs []uuid.UUID) ([]string, error) {
	if s.statusNames != nil {
		return s.statusNames(ctx, statusIDs)
	}
	return nil, nil
}

type stubWorkflowRepository struct {
	list              func(context.Context, uuid.UUID) ([]domain.Workflow, error)
	get               func(context.Context, uuid.UUID) (domain.Workflow, error)
	create            func(context.Context, domain.Workflow, string, string) (domain.Workflow, error)
	update            func(context.Context, domain.Workflow) (domain.Workflow, error)
	impactAnalysis    func(context.Context, uuid.UUID) (domain.WorkflowImpactAnalysis, error)
	replaceReferences func(context.Context, domain.ReplaceWorkflowReferencesInput) (domain.ReplaceWorkflowReferencesResult, error)
	delete            func(context.Context, uuid.UUID) (domain.Workflow, error)
}

func (s stubWorkflowRepository) List(ctx context.Context, projectID uuid.UUID) ([]domain.Workflow, error) {
	if s.list != nil {
		return s.list(ctx, projectID)
	}
	return nil, nil
}

func (s stubWorkflowRepository) Get(ctx context.Context, workflowID uuid.UUID) (domain.Workflow, error) {
	if s.get != nil {
		return s.get(ctx, workflowID)
	}
	return domain.Workflow{}, nil
}

func (s stubWorkflowRepository) Create(ctx context.Context, workflow domain.Workflow, harnessContent string, createdBy string) (domain.Workflow, error) {
	if s.create != nil {
		return s.create(ctx, workflow, harnessContent, createdBy)
	}
	return domain.Workflow{}, nil
}

func (s stubWorkflowRepository) Update(ctx context.Context, workflow domain.Workflow) (domain.Workflow, error) {
	if s.update != nil {
		return s.update(ctx, workflow)
	}
	return domain.Workflow{}, nil
}

func (s stubWorkflowRepository) ImpactAnalysis(ctx context.Context, workflowID uuid.UUID) (domain.WorkflowImpactAnalysis, error) {
	if s.impactAnalysis != nil {
		return s.impactAnalysis(ctx, workflowID)
	}
	return domain.WorkflowImpactAnalysis{}, nil
}

func (s stubWorkflowRepository) ReplaceReferences(
	ctx context.Context,
	input domain.ReplaceWorkflowReferencesInput,
) (domain.ReplaceWorkflowReferencesResult, error) {
	if s.replaceReferences != nil {
		return s.replaceReferences(ctx, input)
	}
	return domain.ReplaceWorkflowReferencesResult{}, nil
}

func (s stubWorkflowRepository) Delete(ctx context.Context, workflowID uuid.UUID) (domain.Workflow, error) {
	if s.delete != nil {
		return s.delete(ctx, workflowID)
	}
	return domain.Workflow{}, nil
}

type stubWorkflowVersionRepository struct {
	current  func(context.Context, uuid.UUID) (domain.WorkflowVersionRecord, error)
	recorded func(context.Context, uuid.UUID, *uuid.UUID) (domain.WorkflowVersionRecord, error)
	list     func(context.Context, uuid.UUID) ([]domain.VersionSummary, error)
	publish  func(context.Context, uuid.UUID, string, string) (domain.Workflow, error)
}

func (s stubWorkflowVersionRepository) CurrentWorkflowVersion(ctx context.Context, workflowID uuid.UUID) (domain.WorkflowVersionRecord, error) {
	if s.current != nil {
		return s.current(ctx, workflowID)
	}
	return domain.WorkflowVersionRecord{}, nil
}

func (s stubWorkflowVersionRepository) RecordedWorkflowVersion(
	ctx context.Context,
	workflowID uuid.UUID,
	workflowVersionID *uuid.UUID,
) (domain.WorkflowVersionRecord, error) {
	if s.recorded != nil {
		return s.recorded(ctx, workflowID, workflowVersionID)
	}
	return domain.WorkflowVersionRecord{}, nil
}

func (s stubWorkflowVersionRepository) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID) ([]domain.VersionSummary, error) {
	if s.list != nil {
		return s.list(ctx, workflowID)
	}
	return nil, nil
}

func (s stubWorkflowVersionRepository) PublishWorkflowVersion(
	ctx context.Context,
	workflowID uuid.UUID,
	content string,
	createdBy string,
) (domain.Workflow, error) {
	if s.publish != nil {
		return s.publish(ctx, workflowID, content, createdBy)
	}
	return domain.Workflow{}, nil
}

type stubSkillRepository struct {
	ensureBuiltinSkills func(context.Context, uuid.UUID, time.Time, []domain.SkillBundle) error
	listSkills          func(context.Context, uuid.UUID) ([]domain.Skill, error)
	skill               func(context.Context, uuid.UUID) (domain.SkillRecord, error)
	skillInProject      func(context.Context, uuid.UUID, uuid.UUID) (domain.SkillRecord, error)
	skillByName         func(context.Context, uuid.UUID, string) (domain.SkillRecord, error)
	skillDetail         func(context.Context, uuid.UUID) (domain.SkillDetail, error)
	createBundle        func(context.Context, domain.CreateSkillBundleInput, domain.SkillBundle, bool, string, time.Time) (domain.SkillDetail, error)
	updateBundle        func(context.Context, uuid.UUID, domain.SkillBundle, time.Time) (domain.SkillDetail, error)
	deleteSkill         func(context.Context, uuid.UUID, time.Time) error
	setSkillEnabled     func(context.Context, uuid.UUID, bool, time.Time) (domain.SkillDetail, error)
}

func (s stubSkillRepository) EnsureBuiltinSkills(ctx context.Context, projectID uuid.UUID, now time.Time, bundles []domain.SkillBundle) error {
	if s.ensureBuiltinSkills != nil {
		return s.ensureBuiltinSkills(ctx, projectID, now, bundles)
	}
	return nil
}

func (s stubSkillRepository) ListSkills(ctx context.Context, projectID uuid.UUID) ([]domain.Skill, error) {
	if s.listSkills != nil {
		return s.listSkills(ctx, projectID)
	}
	return nil, nil
}

func (s stubSkillRepository) Skill(ctx context.Context, skillID uuid.UUID) (domain.SkillRecord, error) {
	if s.skill != nil {
		return s.skill(ctx, skillID)
	}
	return domain.SkillRecord{}, nil
}

func (s stubSkillRepository) SkillInProject(ctx context.Context, projectID uuid.UUID, skillID uuid.UUID) (domain.SkillRecord, error) {
	if s.skillInProject != nil {
		return s.skillInProject(ctx, projectID, skillID)
	}
	return domain.SkillRecord{}, nil
}

func (s stubSkillRepository) SkillByName(ctx context.Context, projectID uuid.UUID, name string) (domain.SkillRecord, error) {
	if s.skillByName != nil {
		return s.skillByName(ctx, projectID, name)
	}
	return domain.SkillRecord{}, nil
}

func (s stubSkillRepository) SkillDetail(ctx context.Context, skillID uuid.UUID) (domain.SkillDetail, error) {
	if s.skillDetail != nil {
		return s.skillDetail(ctx, skillID)
	}
	return domain.SkillDetail{}, nil
}

func (s stubSkillRepository) CreateSkillBundle(
	ctx context.Context,
	input domain.CreateSkillBundleInput,
	bundle domain.SkillBundle,
	enabled bool,
	createdBy string,
	now time.Time,
) (domain.SkillDetail, error) {
	if s.createBundle != nil {
		return s.createBundle(ctx, input, bundle, enabled, createdBy, now)
	}
	return domain.SkillDetail{}, nil
}

func (s stubSkillRepository) UpdateSkillBundle(
	ctx context.Context,
	skillID uuid.UUID,
	bundle domain.SkillBundle,
	updatedAt time.Time,
) (domain.SkillDetail, error) {
	if s.updateBundle != nil {
		return s.updateBundle(ctx, skillID, bundle, updatedAt)
	}
	return domain.SkillDetail{}, nil
}

func (s stubSkillRepository) DeleteSkill(ctx context.Context, skillID uuid.UUID, deletedAt time.Time) error {
	if s.deleteSkill != nil {
		return s.deleteSkill(ctx, skillID, deletedAt)
	}
	return nil
}

func (s stubSkillRepository) SetSkillEnabled(ctx context.Context, skillID uuid.UUID, enabled bool, updatedAt time.Time) (domain.SkillDetail, error) {
	if s.setSkillEnabled != nil {
		return s.setSkillEnabled(ctx, skillID, enabled, updatedAt)
	}
	return domain.SkillDetail{}, nil
}

type stubRuntimeSnapshotReader struct {
	resolve         func(context.Context, uuid.UUID) (domain.RuntimeSnapshot, error)
	resolveRecorded func(context.Context, domain.ResolveRecordedRuntimeSnapshotInput) (domain.RuntimeSnapshot, error)
}

func (s stubRuntimeSnapshotReader) ResolveRuntimeSnapshot(ctx context.Context, workflowID uuid.UUID) (domain.RuntimeSnapshot, error) {
	if s.resolve != nil {
		return s.resolve(ctx, workflowID)
	}
	return domain.RuntimeSnapshot{}, nil
}

func (s stubRuntimeSnapshotReader) ResolveRecordedRuntimeSnapshot(
	ctx context.Context,
	input domain.ResolveRecordedRuntimeSnapshotInput,
) (domain.RuntimeSnapshot, error) {
	if s.resolveRecorded != nil {
		return s.resolveRecorded(ctx, input)
	}
	return domain.RuntimeSnapshot{}, nil
}

type stubHarnessTemplateDataBuilder struct {
	build func(context.Context, domain.BuildHarnessTemplateDataInput) (domain.HarnessTemplateData, error)
}

func (s stubHarnessTemplateDataBuilder) BuildHarnessTemplateData(
	ctx context.Context,
	input domain.BuildHarnessTemplateDataInput,
) (domain.HarnessTemplateData, error) {
	if s.build != nil {
		return s.build(ctx, input)
	}
	return domain.HarnessTemplateData{}, nil
}

func newServiceWithTestDependencies(t *testing.T, deps ServiceDependencies) *Service {
	t.Helper()

	service, err := NewServiceWithDependencies(
		deps,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		t.TempDir(),
	)
	if err != nil {
		t.Fatalf("NewServiceWithDependencies() error = %v", err)
	}
	return service
}

func TestServiceListWorkflowVersionsUsesWorkflowVersionSlice(t *testing.T) {
	workflowID := uuid.New()
	want := []domain.VersionSummary{{ID: uuid.New(), Version: 3, CreatedBy: "tester"}}
	service := newServiceWithTestDependencies(t, ServiceDependencies{
		WorkflowVersions: stubWorkflowVersionRepository{
			list: func(_ context.Context, gotWorkflowID uuid.UUID) ([]domain.VersionSummary, error) {
				if gotWorkflowID != workflowID {
					t.Fatalf("workflow id = %s, want %s", gotWorkflowID, workflowID)
				}
				return want, nil
			},
		},
	})

	got, err := service.ListWorkflowVersions(context.Background(), workflowID)
	if err != nil {
		t.Fatalf("ListWorkflowVersions() error = %v", err)
	}
	if len(got) != 1 || got[0].Version != want[0].Version || got[0].CreatedBy != want[0].CreatedBy {
		t.Fatalf("ListWorkflowVersions() = %+v", got)
	}
}

func TestServiceListSkillsUsesSkillSlices(t *testing.T) {
	projectID := uuid.New()
	want := []domain.Skill{{ID: uuid.New(), Name: "skill-one"}}
	service := newServiceWithTestDependencies(t, ServiceDependencies{
		Validators: stubProjectValidationRepository{
			ensureProjectExists: func(_ context.Context, gotProjectID uuid.UUID) error {
				if gotProjectID != projectID {
					t.Fatalf("project id = %s, want %s", gotProjectID, projectID)
				}
				return nil
			},
		},
		Skills: stubSkillRepository{
			ensureBuiltinSkills: func(_ context.Context, gotProjectID uuid.UUID, _ time.Time, bundles []domain.SkillBundle) error {
				if gotProjectID != projectID {
					t.Fatalf("builtin project id = %s, want %s", gotProjectID, projectID)
				}
				if len(bundles) == 0 {
					t.Fatal("expected builtin skill bundles")
				}
				return nil
			},
			listSkills: func(_ context.Context, gotProjectID uuid.UUID) ([]domain.Skill, error) {
				if gotProjectID != projectID {
					t.Fatalf("list project id = %s, want %s", gotProjectID, projectID)
				}
				return want, nil
			},
		},
	})

	got, err := service.ListSkills(context.Background(), projectID)
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "skill-one" {
		t.Fatalf("ListSkills() = %+v", got)
	}
}

func TestServiceResolveRuntimeSnapshotUsesRuntimeSlices(t *testing.T) {
	workflowID := uuid.New()
	projectID := uuid.New()
	want := domain.RuntimeSnapshot{
		Workflow: domain.RuntimeWorkflowSnapshot{WorkflowID: workflowID, Version: 2},
	}
	service := newServiceWithTestDependencies(t, ServiceDependencies{
		Workflows: stubWorkflowRepository{
			get: func(_ context.Context, gotWorkflowID uuid.UUID) (domain.Workflow, error) {
				if gotWorkflowID != workflowID {
					t.Fatalf("workflow id = %s, want %s", gotWorkflowID, workflowID)
				}
				return domain.Workflow{ID: workflowID, ProjectID: projectID}, nil
			},
		},
		Skills: stubSkillRepository{
			ensureBuiltinSkills: func(_ context.Context, gotProjectID uuid.UUID, _ time.Time, bundles []domain.SkillBundle) error {
				if gotProjectID != projectID {
					t.Fatalf("builtin project id = %s, want %s", gotProjectID, projectID)
				}
				if len(bundles) == 0 {
					t.Fatal("expected builtin skill bundles")
				}
				return nil
			},
		},
		RuntimeSnapshots: stubRuntimeSnapshotReader{
			resolve: func(_ context.Context, gotWorkflowID uuid.UUID) (domain.RuntimeSnapshot, error) {
				if gotWorkflowID != workflowID {
					t.Fatalf("runtime workflow id = %s, want %s", gotWorkflowID, workflowID)
				}
				return want, nil
			},
		},
	})

	got, err := service.ResolveRuntimeSnapshot(context.Background(), workflowID)
	if err != nil {
		t.Fatalf("ResolveRuntimeSnapshot() error = %v", err)
	}
	if got.Workflow.WorkflowID != want.Workflow.WorkflowID || got.Workflow.Version != want.Workflow.Version {
		t.Fatalf("ResolveRuntimeSnapshot() = %+v", got)
	}
}

func TestServiceBuildHarnessTemplateDataUsesTemplateSlices(t *testing.T) {
	workflowID := uuid.New()
	projectID := uuid.New()
	ticketID := uuid.New()
	want := domain.HarnessTemplateData{
		Ticket:   domain.HarnessTicketData{ID: ticketID.String(), Identifier: "ASE-9"},
		Workflow: domain.HarnessWorkflowData{Name: "coding"},
	}
	service := newServiceWithTestDependencies(t, ServiceDependencies{
		Workflows: stubWorkflowRepository{
			get: func(_ context.Context, gotWorkflowID uuid.UUID) (domain.Workflow, error) {
				if gotWorkflowID != workflowID {
					t.Fatalf("workflow id = %s, want %s", gotWorkflowID, workflowID)
				}
				return domain.Workflow{ID: workflowID, ProjectID: projectID}, nil
			},
		},
		Skills: stubSkillRepository{
			ensureBuiltinSkills: func(_ context.Context, gotProjectID uuid.UUID, _ time.Time, bundles []domain.SkillBundle) error {
				if gotProjectID != projectID {
					t.Fatalf("builtin project id = %s, want %s", gotProjectID, projectID)
				}
				if len(bundles) == 0 {
					t.Fatal("expected builtin skill bundles")
				}
				return nil
			},
		},
		HarnessTemplates: stubHarnessTemplateDataBuilder{
			build: func(_ context.Context, input domain.BuildHarnessTemplateDataInput) (domain.HarnessTemplateData, error) {
				if input.WorkflowID != workflowID {
					t.Fatalf("builder workflow id = %s, want %s", input.WorkflowID, workflowID)
				}
				if input.TicketID != ticketID {
					t.Fatalf("builder ticket id = %s, want %s", input.TicketID, ticketID)
				}
				return want, nil
			},
		},
	})

	got, err := service.BuildHarnessTemplateData(context.Background(), BuildHarnessTemplateDataInput{
		WorkflowID: workflowID,
		TicketID:   ticketID,
	})
	if err != nil {
		t.Fatalf("BuildHarnessTemplateData() error = %v", err)
	}
	if got.Ticket.Identifier != want.Ticket.Identifier || got.Workflow.Name != want.Workflow.Name {
		t.Fatalf("BuildHarnessTemplateData() = %+v", got)
	}
}
