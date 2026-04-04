package ticket

import domain "github.com/BetterAndBetterII/openase/internal/domain/ticket"
import "github.com/google/uuid"

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
type ArchivedListRawInput = domain.ArchivedListRawInput
type ArchivedListInput = domain.ArchivedListInput
type ArchivedListResult = domain.ArchivedListResult
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
type PickupDiagnosis = domain.PickupDiagnosis
type PickupDiagnosisState = domain.PickupDiagnosisState
type PickupDiagnosisReasonCode = domain.PickupDiagnosisReasonCode
type PickupDiagnosisReasonSeverity = domain.PickupDiagnosisReasonSeverity
type PickupDiagnosisReason = domain.PickupDiagnosisReason
type PickupDiagnosisWorkflow = domain.PickupDiagnosisWorkflow
type PickupDiagnosisAgent = domain.PickupDiagnosisAgent
type PickupDiagnosisProvider = domain.PickupDiagnosisProvider
type PickupDiagnosisRetry = domain.PickupDiagnosisRetry
type PickupDiagnosisCapacity = domain.PickupDiagnosisCapacity
type PickupDiagnosisCapacityBucket = domain.PickupDiagnosisCapacityBucket
type PickupDiagnosisStatusCapacity = domain.PickupDiagnosisStatusCapacity
type PickupDiagnosisBlockedTicket = domain.PickupDiagnosisBlockedTicket

type LifecycleHookRuntimeData = domain.LifecycleHookRuntimeData
type HookWorkspace = domain.HookWorkspace

func ParseArchivedListInput(projectID uuid.UUID, raw ArchivedListRawInput) (ArchivedListInput, error) {
	return domain.ParseArchivedListInput(projectID, raw)
}
