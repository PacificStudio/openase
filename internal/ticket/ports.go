package ticket

import (
	"context"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/ticket"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

type ActivityRepository interface {
	RecordActivityEvent(ctx context.Context, input domain.RecordActivityEventInput) (catalogdomain.ActivityEvent, error)
}

type QueryRepository interface {
	List(ctx context.Context, input domain.ListInput) ([]domain.Ticket, error)
	ListArchived(ctx context.Context, input domain.ArchivedListInput) (domain.ArchivedListResult, error)
	Get(ctx context.Context, ticketID uuid.UUID) (domain.Ticket, error)
}

type CommandRepository interface {
	Create(ctx context.Context, input domain.CreateInput) (domain.Ticket, error)
	Update(ctx context.Context, input domain.UpdateInput) (domain.UpdateResult, error)
	ResumeRetry(ctx context.Context, input domain.ResumeRetryInput) (domain.Ticket, error)
}

type LinkRepository interface {
	AddDependency(ctx context.Context, input domain.AddDependencyInput) (domain.Dependency, error)
	RemoveDependency(ctx context.Context, ticketID uuid.UUID, dependencyID uuid.UUID) (domain.DeleteDependencyResult, error)
	AddExternalLink(ctx context.Context, input domain.AddExternalLinkInput) (domain.ExternalLink, error)
	RemoveExternalLink(ctx context.Context, ticketID uuid.UUID, externalLinkID uuid.UUID) (domain.DeleteExternalLinkResult, error)
}

type CommentRepository interface {
	ListComments(ctx context.Context, ticketID uuid.UUID) ([]domain.Comment, error)
	ListCommentRevisions(ctx context.Context, ticketID uuid.UUID, commentID uuid.UUID) ([]domain.CommentRevision, error)
	AddComment(ctx context.Context, input domain.AddCommentInput) (domain.Comment, error)
	UpdateComment(ctx context.Context, input domain.UpdateCommentInput) (domain.Comment, error)
	RemoveComment(ctx context.Context, ticketID uuid.UUID, commentID uuid.UUID) (domain.DeleteCommentResult, error)
}

type UsageRepository interface {
	RecordUsage(ctx context.Context, input domain.RecordUsageInput, usageDelta ticketing.UsageDelta) (domain.PersistedUsageResult, error)
}

type RuntimeRepository interface {
	GetPickupDiagnosis(ctx context.Context, ticketID uuid.UUID) (domain.PickupDiagnosis, error)
	LoadLifecycleHookRuntimeData(ctx context.Context, ticketID uuid.UUID, runID uuid.UUID, workflowID *uuid.UUID) (domain.LifecycleHookRuntimeData, error)
}

type Repository interface {
	ActivityRepository
	QueryRepository
	CommandRepository
	LinkRepository
	CommentRepository
	UsageRepository
	RuntimeRepository
}

type Dependencies struct {
	Activity ActivityRepository
	Query    QueryRepository
	Command  CommandRepository
	Link     LinkRepository
	Comment  CommentRepository
	Usage    UsageRepository
	Runtime  RuntimeRepository
}
