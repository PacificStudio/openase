package scheduledjob

import (
	"context"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/scheduledjob"
	"github.com/google/uuid"
)

type Repository interface {
	EnsureProjectExists(ctx context.Context, projectID uuid.UUID) error
	List(ctx context.Context, projectID uuid.UUID) ([]domain.Job, error)
	Get(ctx context.Context, jobID uuid.UUID) (domain.Job, error)
	Create(ctx context.Context, job domain.Job) (domain.Job, error)
	Update(ctx context.Context, job domain.Job) (domain.Job, error)
	Delete(ctx context.Context, jobID uuid.UUID) error
	ListDue(ctx context.Context, now time.Time) ([]domain.Job, error)
	LoadWorkflow(ctx context.Context, projectID uuid.UUID, workflowID uuid.UUID) (domain.Workflow, error)
	ResolveStatusIDByName(ctx context.Context, projectID uuid.UUID, statusName string) (uuid.UUID, error)
}
