package scheduledjob

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entscheduledjob "github.com/BetterAndBetterII/openase/ent/scheduledjob"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	domain "github.com/BetterAndBetterII/openase/internal/domain/scheduledjob"
	"github.com/google/uuid"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) EnsureProjectExists(ctx context.Context, projectID uuid.UUID) error {
	exists, err := r.client.Project.Query().
		Where(entproject.IDEQ(projectID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project: %w", err)
	}
	if !exists {
		return domain.ErrProjectNotFound
	}

	return nil
}

func (r *EntRepository) List(ctx context.Context, projectID uuid.UUID) ([]domain.Job, error) {
	items, err := r.client.ScheduledJob.Query().
		Where(entscheduledjob.ProjectIDEQ(projectID)).
		Order(ent.Asc(entscheduledjob.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list scheduled jobs: %w", err)
	}

	jobs := make([]domain.Job, 0, len(items))
	for _, item := range items {
		jobs = append(jobs, mapJob(item))
	}
	return jobs, nil
}

func (r *EntRepository) Get(ctx context.Context, jobID uuid.UUID) (domain.Job, error) {
	item, err := r.client.ScheduledJob.Get(ctx, jobID)
	if err != nil {
		return domain.Job{}, mapReadError("get scheduled job", err)
	}
	return mapJob(item), nil
}

func (r *EntRepository) Create(ctx context.Context, job domain.Job) (domain.Job, error) {
	builder := r.client.ScheduledJob.Create().
		SetProjectID(job.ProjectID).
		SetName(strings.TrimSpace(job.Name)).
		SetCronExpression(strings.TrimSpace(job.CronExpression)).
		SetTicketTemplate(cloneTemplate(job.TicketTemplate)).
		SetIsEnabled(job.IsEnabled)

	if job.NextRunAt != nil {
		builder.SetNextRunAt(job.NextRunAt.UTC())
	}
	if job.LastRunAt != nil {
		builder.SetLastRunAt(job.LastRunAt.UTC())
	}
	if job.WorkflowID != nil {
		builder.SetWorkflowID(*job.WorkflowID)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Job{}, mapWriteError("create scheduled job", err)
	}
	return mapJob(item), nil
}

func (r *EntRepository) Update(ctx context.Context, job domain.Job) (domain.Job, error) {
	builder := r.client.ScheduledJob.UpdateOneID(job.ID).
		SetName(strings.TrimSpace(job.Name)).
		SetCronExpression(strings.TrimSpace(job.CronExpression)).
		SetTicketTemplate(cloneTemplate(job.TicketTemplate)).
		SetIsEnabled(job.IsEnabled)

	if job.NextRunAt == nil {
		builder.ClearNextRunAt()
	} else {
		builder.SetNextRunAt(job.NextRunAt.UTC())
	}
	if job.LastRunAt == nil {
		builder.ClearLastRunAt()
	} else {
		builder.SetLastRunAt(job.LastRunAt.UTC())
	}
	if job.WorkflowID == nil {
		builder.ClearWorkflow()
	} else {
		builder.SetWorkflowID(*job.WorkflowID)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return domain.Job{}, mapWriteError("update scheduled job", err)
	}
	return mapJob(item), nil
}

func (r *EntRepository) Delete(ctx context.Context, jobID uuid.UUID) error {
	if err := r.client.ScheduledJob.DeleteOneID(jobID).Exec(ctx); err != nil {
		return mapWriteError("delete scheduled job", err)
	}
	return nil
}

func (r *EntRepository) ListDue(ctx context.Context, now time.Time) ([]domain.Job, error) {
	items, err := r.client.ScheduledJob.Query().
		Where(
			entscheduledjob.IsEnabled(true),
			entscheduledjob.NextRunAtNotNil(),
			entscheduledjob.NextRunAtLTE(now),
			entscheduledjob.HasProjectWith(entproject.StatusEQ("In Progress")),
		).
		Order(ent.Asc(entscheduledjob.FieldNextRunAt), ent.Asc(entscheduledjob.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list due scheduled jobs: %w", err)
	}

	jobs := make([]domain.Job, 0, len(items))
	for _, item := range items {
		jobs = append(jobs, mapJob(item))
	}
	return jobs, nil
}

func (r *EntRepository) LoadWorkflow(ctx context.Context, projectID uuid.UUID, workflowID uuid.UUID) (domain.Workflow, error) {
	item, err := r.client.Workflow.Query().
		Where(
			entworkflow.IDEQ(workflowID),
			entworkflow.ProjectIDEQ(projectID),
		).
		WithPickupStatuses(func(query *ent.TicketStatusQuery) {
			query.Order(ent.Asc(entticketstatus.FieldPosition), ent.Asc(entticketstatus.FieldName))
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Workflow{}, domain.ErrWorkflowNotFound
		}
		return domain.Workflow{}, fmt.Errorf("get workflow: %w", err)
	}

	pickupStatuses := make([]domain.WorkflowStatus, 0, len(item.Edges.PickupStatuses))
	for _, status := range item.Edges.PickupStatuses {
		if status == nil {
			continue
		}
		pickupStatuses = append(pickupStatuses, domain.WorkflowStatus{
			ID:   status.ID,
			Name: status.Name,
		})
	}

	return domain.Workflow{
		ID:             item.ID,
		ProjectID:      item.ProjectID,
		Name:           item.Name,
		Type:           item.Type,
		PickupStatuses: pickupStatuses,
	}, nil
}

func (r *EntRepository) ResolveStatusIDByName(ctx context.Context, projectID uuid.UUID, statusName string) (uuid.UUID, error) {
	item, err := r.client.TicketStatus.Query().
		Where(
			entticketstatus.ProjectIDEQ(projectID),
			entticketstatus.NameEQ(strings.TrimSpace(statusName)),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.UUID{}, domain.ErrStatusNotFound
		}
		return uuid.UUID{}, fmt.Errorf("get ticket status: %w", err)
	}
	return item.ID, nil
}

func mapJob(item *ent.ScheduledJob) domain.Job {
	return domain.Job{
		ID:             item.ID,
		ProjectID:      item.ProjectID,
		Name:           item.Name,
		CronExpression: item.CronExpression,
		TicketTemplate: cloneTemplate(item.TicketTemplate),
		IsEnabled:      item.IsEnabled,
		LastRunAt:      cloneTime(item.LastRunAt),
		NextRunAt:      cloneTime(item.NextRunAt),
		WorkflowID:     cloneUUID(item.WorkflowID),
	}
}

func cloneTemplate(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func cloneUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func mapReadError(action string, err error) error {
	if ent.IsNotFound(err) {
		return domain.ErrScheduledJobNotFound
	}
	return fmt.Errorf("%s: %w", action, err)
}

func mapWriteError(action string, err error) error {
	switch {
	case ent.IsNotFound(err):
		return domain.ErrScheduledJobNotFound
	case ent.IsConstraintError(err):
		return domain.ErrScheduledJobConflict
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}
