package ticket

import (
	domain "github.com/BetterAndBetterII/openase/internal/domain/ticket"
	"github.com/google/uuid"
)

var (
	ErrProjectNotFound       = domain.ErrProjectNotFound
	ErrProjectRepoNotFound   = domain.ErrProjectRepoNotFound
	ErrRepoScopeRequired     = domain.ErrRepoScopeRequired
	ErrTicketNotFound        = domain.ErrTicketNotFound
	ErrTicketConflict        = domain.ErrTicketConflict
	ErrStatusNotFound        = domain.ErrStatusNotFound
	ErrWorkflowNotFound      = domain.ErrWorkflowNotFound
	ErrStatusNotAllowed      = domain.ErrStatusNotAllowed
	ErrParentTicketNotFound  = domain.ErrParentTicketNotFound
	ErrTargetMachineNotFound = domain.ErrTargetMachineNotFound
	ErrDependencyNotFound    = domain.ErrDependencyNotFound
	ErrDependencyConflict    = domain.ErrDependencyConflict
	ErrCommentNotFound       = domain.ErrCommentNotFound
	ErrExternalLinkNotFound  = domain.ErrExternalLinkNotFound
	ErrExternalLinkConflict  = domain.ErrExternalLinkConflict
	ErrInvalidDependency     = domain.ErrInvalidDependency
	ErrRetryResumeConflict   = domain.ErrRetryResumeConflict
)

type Optional[T any] = domain.Optional[T]

func Some[T any](value T) Optional[T] {
	return Optional[T]{Set: true, Value: value}
}

type TicketReference = domain.TicketReference
type Dependency = domain.Dependency
type ExternalLink = domain.ExternalLink
type Comment = domain.Comment
type CommentRevision = domain.CommentRevision
type Ticket = domain.Ticket
type ListInput = domain.ListInput
type CreateInput = domain.CreateInput
type CreateRepoScopeInput = domain.CreateRepoScopeInput
type UpdateInput = domain.UpdateInput
type UpdateResult = domain.UpdateResult
type DeferredLifecycleHook = domain.DeferredLifecycleHook
type AddDependencyInput = domain.AddDependencyInput
type AddExternalLinkInput = domain.AddExternalLinkInput
type ResumeRetryInput = domain.ResumeRetryInput
type DeleteDependencyResult = domain.DeleteDependencyResult
type DeleteExternalLinkResult = domain.DeleteExternalLinkResult
type AddCommentInput = domain.AddCommentInput
type UpdateCommentInput = domain.UpdateCommentInput
type DeleteCommentResult = domain.DeleteCommentResult
type RecordActivityEventInput = domain.RecordActivityEventInput
type RecordUsageInput = domain.RecordUsageInput
type AppliedUsage = domain.AppliedUsage
type RecordUsageResult = domain.RecordUsageResult
type UsageMetricsAgent = domain.UsageMetricsAgent
type PersistedUsageResult = domain.PersistedUsageResult

type LifecycleHookRuntimeData = domain.LifecycleHookRuntimeData
type HookWorkspace = domain.HookWorkspace

func cloneUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
