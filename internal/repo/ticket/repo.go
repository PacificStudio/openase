package ticket

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	"github.com/BetterAndBetterII/openase/ent/project"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketcomment "github.com/BetterAndBetterII/openase/ent/ticketcomment"
	entticketcommentrevision "github.com/BetterAndBetterII/openase/ent/ticketcommentrevision"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketexternallink "github.com/BetterAndBetterII/openase/ent/ticketexternallink"
	entticketreposcope "github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/ticket"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	workflowdomain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/google/uuid"
)

const (
	defaultCreatedBy        = "user:api"
	defaultIdentifierPrefix = "ASE"
)

var errUnavailable = errors.New("ticket repository unavailable")

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) RecordActivityEvent(
	ctx context.Context,
	input RecordActivityEventInput,
) (catalogdomain.ActivityEvent, error) {
	if r == nil || r.client == nil {
		return catalogdomain.ActivityEvent{}, errUnavailable
	}
	if input.ProjectID == uuid.Nil {
		return catalogdomain.ActivityEvent{}, fmt.Errorf("activity event project id must not be empty")
	}

	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	builder := r.client.ActivityEvent.Create().
		SetProjectID(input.ProjectID).
		SetEventType(input.EventType.String()).
		SetMessage(strings.TrimSpace(input.Message)).
		SetMetadata(cloneAnyMap(input.Metadata)).
		SetCreatedAt(createdAt.UTC())
	if input.TicketID != nil {
		builder.SetTicketID(*input.TicketID)
	}
	if input.AgentID != nil {
		builder.SetAgentID(*input.AgentID)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return catalogdomain.ActivityEvent{}, fmt.Errorf("record activity event: %w", err)
	}

	return catalogdomain.ActivityEvent{
		ID:        item.ID,
		ProjectID: item.ProjectID,
		TicketID:  item.TicketID,
		AgentID:   item.AgentID,
		EventType: input.EventType,
		Message:   item.Message,
		Metadata:  cloneAnyMap(item.Metadata),
		CreatedAt: item.CreatedAt.UTC(),
	}, nil
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

// List returns tickets in a project ordered for UI consumption.
func (r *EntRepository) List(ctx context.Context, input ListInput) ([]Ticket, error) {
	if r.client == nil {
		return nil, errUnavailable
	}
	if err := r.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return nil, err
	}

	query := r.client.Ticket.Query().
		Where(entticket.ProjectIDEQ(input.ProjectID), entticket.Archived(false)).
		Order(ent.Asc(entticket.FieldCreatedAt), ent.Asc(entticket.FieldIdentifier)).
		WithStatus().
		WithParent(func(query *ent.TicketQuery) {
			query.WithStatus()
		}).
		WithOutgoingDependencies(func(query *ent.TicketDependencyQuery) {
			query.Order(ent.Asc(entticketdependency.FieldType), ent.Asc(entticketdependency.FieldTargetTicketID)).
				WithTargetTicket(func(ticketQuery *ent.TicketQuery) {
					ticketQuery.WithStatus()
				})
		}).
		WithIncomingDependencies(func(query *ent.TicketDependencyQuery) {
			query.Where(entticketdependency.TypeEQ(entticketdependency.TypeBlocks)).
				Order(ent.Asc(entticketdependency.FieldSourceTicketID)).
				WithSourceTicket(func(ticketQuery *ent.TicketQuery) {
					ticketQuery.WithStatus()
				})
		}).
		WithExternalLinks(func(query *ent.TicketExternalLinkQuery) {
			query.Order(ent.Asc(entticketexternallink.FieldCreatedAt), ent.Asc(entticketexternallink.FieldID))
		})

	if len(input.StatusNames) > 0 {
		query = query.Where(entticket.HasStatusWith(entticketstatus.NameIn(input.StatusNames...)))
	}
	if len(input.Priorities) > 0 {
		query = query.Where(entticket.PriorityIn(toEntTicketPriorities(input.Priorities)...))
	}
	if input.Limit > 0 {
		query = query.Limit(input.Limit)
	}

	items, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}

	tickets := make([]Ticket, 0, len(items))
	for _, item := range items {
		tickets = append(tickets, mapTicket(item))
	}

	return tickets, nil
}

func (r *EntRepository) ListArchived(ctx context.Context, input ArchivedListInput) (ArchivedListResult, error) {
	if r.client == nil {
		return ArchivedListResult{}, errUnavailable
	}
	if err := r.ensureProjectExists(ctx, input.ProjectID); err != nil {
		return ArchivedListResult{}, err
	}

	total, err := r.client.Ticket.Query().
		Where(
			entticket.ProjectIDEQ(input.ProjectID),
			entticket.Archived(true),
		).
		Count(ctx)
	if err != nil {
		return ArchivedListResult{}, fmt.Errorf("count archived tickets: %w", err)
	}

	offset := (input.Page - 1) * input.PerPage
	items, err := r.client.Ticket.Query().
		Where(
			entticket.ProjectIDEQ(input.ProjectID),
			entticket.Archived(true),
		).
		Order(ent.Asc(entticket.FieldCreatedAt), ent.Asc(entticket.FieldIdentifier)).
		Limit(input.PerPage).
		Offset(offset).
		WithStatus().
		All(ctx)
	if err != nil {
		return ArchivedListResult{}, fmt.Errorf("list archived tickets: %w", err)
	}

	tickets := make([]Ticket, 0, len(items))
	for _, item := range items {
		tickets = append(tickets, mapTicket(item))
	}

	return ArchivedListResult{
		Tickets: tickets,
		Total:   total,
		Page:    input.Page,
		PerPage: input.PerPage,
	}, nil
}

// Get loads a single ticket with its related status, parent, children, and dependencies.
func (r *EntRepository) Get(ctx context.Context, ticketID uuid.UUID) (Ticket, error) {
	if r.client == nil {
		return Ticket{}, errUnavailable
	}

	item, err := r.client.Ticket.Query().
		Where(entticket.ID(ticketID)).
		WithStatus().
		WithParent(func(query *ent.TicketQuery) {
			query.WithStatus()
		}).
		WithChildren(func(query *ent.TicketQuery) {
			query.Order(ent.Asc(entticket.FieldCreatedAt), ent.Asc(entticket.FieldIdentifier)).WithStatus()
		}).
		WithOutgoingDependencies(func(query *ent.TicketDependencyQuery) {
			query.Order(ent.Asc(entticketdependency.FieldType), ent.Asc(entticketdependency.FieldTargetTicketID)).
				WithTargetTicket(func(ticketQuery *ent.TicketQuery) {
					ticketQuery.WithStatus()
				})
		}).
		WithIncomingDependencies(func(query *ent.TicketDependencyQuery) {
			query.Where(entticketdependency.TypeEQ(entticketdependency.TypeBlocks)).
				Order(ent.Asc(entticketdependency.FieldSourceTicketID)).
				WithSourceTicket(func(ticketQuery *ent.TicketQuery) {
					ticketQuery.WithStatus()
				})
		}).
		WithExternalLinks(func(query *ent.TicketExternalLinkQuery) {
			query.Order(ent.Asc(entticketexternallink.FieldCreatedAt), ent.Asc(entticketexternallink.FieldID))
		}).
		Only(ctx)
	if err != nil {
		return Ticket{}, mapTicketReadError("get ticket", err)
	}

	return mapTicket(item), nil
}

// Create persists a new ticket and applies project defaults.
func (r *EntRepository) Create(ctx context.Context, input CreateInput) (Ticket, error) {
	if r.client == nil {
		return Ticket{}, errUnavailable
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return Ticket{}, fmt.Errorf("start ticket create tx: %w", err)
	}
	defer rollback(tx)

	if err := ensureProjectExistsTx(ctx, tx, input.ProjectID); err != nil {
		return Ticket{}, err
	}

	statusID, err := resolveCreateStatusID(ctx, tx, input.ProjectID, input.StatusID)
	if err != nil {
		return Ticket{}, err
	}
	if input.WorkflowID != nil {
		if err := ensureWorkflowBelongsToProject(ctx, tx, input.ProjectID, *input.WorkflowID); err != nil {
			return Ticket{}, err
		}
	}
	if input.TargetMachineID != nil {
		if err := ensureTargetMachineBelongsToProjectOrganization(ctx, tx, input.ProjectID, *input.TargetMachineID); err != nil {
			return Ticket{}, err
		}
	}
	if input.ParentTicketID != nil {
		if err := ensureTicketBelongsToProject(ctx, tx, input.ProjectID, *input.ParentTicketID, ErrParentTicketNotFound); err != nil {
			return Ticket{}, err
		}
	}

	identifier, err := nextTicketIdentifier(ctx, tx, input.ProjectID)
	if err != nil {
		return Ticket{}, err
	}

	builder := tx.Ticket.Create().
		SetProjectID(input.ProjectID).
		SetIdentifier(identifier).
		SetTitle(input.Title).
		SetDescription(input.Description).
		SetStatusID(statusID).
		SetArchived(input.Archived).
		SetType(toEntTicketType(input.Type)).
		SetCreatedBy(resolveCreatedBy(input.CreatedBy)).
		SetBudgetUsd(input.BudgetUSD).
		SetRetryToken(NewRetryToken())
	if input.Priority != nil {
		builder.SetPriority(toEntTicketPriority(*input.Priority))
	}

	if input.WorkflowID != nil {
		builder.SetWorkflowID(*input.WorkflowID)
	}
	if input.TargetMachineID != nil {
		builder.SetTargetMachineID(*input.TargetMachineID)
	}
	if input.ParentTicketID != nil {
		builder.SetParentTicketID(*input.ParentTicketID)
	}
	if input.ExternalRef != "" {
		builder.SetExternalRef(input.ExternalRef)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return Ticket{}, mapTicketWriteError("create ticket", err)
	}

	if input.ParentTicketID != nil {
		if _, err := ensureSubIssueDependency(ctx, tx, created.ID, *input.ParentTicketID); err != nil {
			return Ticket{}, err
		}
	}
	if err := createTicketRepoScopes(ctx, tx, created.ProjectID, created.ID, input.RepoScopes); err != nil {
		return Ticket{}, err
	}

	if err := tx.Commit(); err != nil {
		return Ticket{}, fmt.Errorf("commit ticket create tx: %w", err)
	}

	return r.Get(ctx, created.ID)
}

func createTicketRepoScopes(
	ctx context.Context,
	tx *ent.Tx,
	projectID uuid.UUID,
	ticketID uuid.UUID,
	requested []CreateRepoScopeInput,
) error {
	projectRepos, err := tx.ProjectRepo.Query().
		Where(entprojectrepo.ProjectID(projectID)).
		Order(entprojectrepo.ByName(), entprojectrepo.ByID()).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list project repos for ticket create: %w", err)
	}

	repoByID := make(map[uuid.UUID]*ent.ProjectRepo, len(projectRepos))
	for _, repo := range projectRepos {
		repoByID[repo.ID] = repo
	}

	if len(requested) == 0 {
		if len(projectRepos) <= 1 {
			if len(projectRepos) == 0 {
				return nil
			}
			requested = []CreateRepoScopeInput{{RepoID: projectRepos[0].ID}}
		} else {
			return ErrRepoScopeRequired
		}
	}

	seenRepoIDs := make(map[uuid.UUID]struct{}, len(requested))
	for _, scope := range requested {
		repo := repoByID[scope.RepoID]
		if repo == nil {
			return ErrProjectRepoNotFound
		}
		if _, duplicate := seenRepoIDs[scope.RepoID]; duplicate {
			return fmt.Errorf("repo_scopes must not contain duplicate repo_id values")
		}
		seenRepoIDs[scope.RepoID] = struct{}{}

		branchName := strings.TrimSpace(repo.DefaultBranch)
		if scope.BranchName != nil {
			branchName = strings.TrimSpace(*scope.BranchName)
		}
		if branchName == "" {
			branchName = "main"
		}

		if _, err := tx.TicketRepoScope.Create().
			SetTicketID(ticketID).
			SetRepoID(scope.RepoID).
			SetBranchName(branchName).
			SetPrStatus(entticketreposcope.PrStatusNone).
			SetCiStatus(entticketreposcope.CiStatusPending).
			Save(ctx); err != nil {
			return mapTicketWriteError("create ticket repo scope", err)
		}
	}

	return nil
}

// Update applies a partial update to an existing ticket.
func (r *EntRepository) Update(ctx context.Context, input UpdateInput) (UpdateResult, error) {
	if r.client == nil {
		return UpdateResult{}, errUnavailable
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("start ticket update tx: %w", err)
	}
	defer rollback(tx)

	current, err := tx.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		return UpdateResult{}, mapTicketReadError("get ticket for update", err)
	}

	builder := tx.Ticket.UpdateOneID(current.ID)
	statusChanged := false
	archivedChanged := false
	targetMachineChanged := false
	statusChangeDisposition := ticketing.StatusChangeRunDispositionRetain
	releasedRunID := current.CurrentRunID
	releasedWorkflowID := current.WorkflowID
	var releasedHookName string

	if input.Title.Set {
		builder.SetTitle(input.Title.Value)
	}
	if input.Description.Set {
		builder.SetDescription(input.Description.Value)
	}
	if input.StatusID.Set {
		if err := ensureStatusBelongsToProject(ctx, tx, current.ProjectID, input.StatusID.Value); err != nil {
			return UpdateResult{}, err
		}
		if input.RestrictStatusToWorkflowFinishSet && current.WorkflowID != nil {
			if err := ensureStatusAllowedByWorkflowFinishSet(ctx, tx, *current.WorkflowID, input.StatusID.Value); err != nil {
				return UpdateResult{}, err
			}
		}
		statusChanged = input.StatusID.Value != current.StatusID
		if statusChanged {
			statusChangeDisposition, err = classifyStatusChangeRunDisposition(ctx, tx, current, input.StatusID.Value)
			if err != nil {
				return UpdateResult{}, err
			}
			switch statusChangeDisposition {
			case ticketing.StatusChangeRunDispositionDone:
				releasedHookName = "on_done"
			case ticketing.StatusChangeRunDispositionCancel:
				releasedHookName = "on_cancel"
			}
		}
		builder.SetStatusID(input.StatusID.Value)
	}
	if input.Archived.Set {
		archivedChanged = input.Archived.Value != current.Archived
		builder.SetArchived(input.Archived.Value)
		if archivedChanged && input.Archived.Value {
			releasedHookName = "on_cancel"
		}
	}
	if input.Priority.Set {
		if input.Priority.Value == nil {
			builder.ClearPriority()
		} else {
			builder.SetPriority(toEntTicketPriority(*input.Priority.Value))
		}
	}
	if input.Type.Set {
		builder.SetType(toEntTicketType(input.Type.Value))
	}
	if input.WorkflowID.Set {
		if input.WorkflowID.Value == nil {
			builder.ClearWorkflowID()
		} else {
			if err := ensureWorkflowBelongsToProject(ctx, tx, current.ProjectID, *input.WorkflowID.Value); err != nil {
				return UpdateResult{}, err
			}
			builder.SetWorkflowID(*input.WorkflowID.Value)
		}
	}
	if input.TargetMachineID.Set {
		if input.TargetMachineID.Value == nil {
			builder.ClearTargetMachineID()
		} else {
			if err := ensureTargetMachineBelongsToProjectOrganization(ctx, tx, current.ProjectID, *input.TargetMachineID.Value); err != nil {
				return UpdateResult{}, err
			}
			builder.SetTargetMachineID(*input.TargetMachineID.Value)
		}
		targetMachineChanged = !optionalUUIDPointerEqual(current.TargetMachineID, input.TargetMachineID.Value)
		if targetMachineChanged {
			releasedHookName = "on_cancel"
			builder.ClearCurrentRunID()
		}
	}
	if input.CreatedBy.Set {
		builder.SetCreatedBy(resolveCreatedBy(input.CreatedBy.Value))
	}
	if input.ExternalRef.Set {
		if strings.TrimSpace(input.ExternalRef.Value) == "" {
			builder.ClearExternalRef()
		} else {
			builder.SetExternalRef(strings.TrimSpace(input.ExternalRef.Value))
		}
	}
	if input.BudgetUSD.Set {
		builder.SetBudgetUsd(input.BudgetUSD.Value)
		reconcileBudgetPauseState(builder, current, input.BudgetUSD.Value)
	}
	if input.ParentTicketID.Set {
		if input.ParentTicketID.Value == nil {
			builder.ClearParentTicketID()
		} else {
			if *input.ParentTicketID.Value == current.ID {
				return UpdateResult{}, ErrInvalidDependency
			}
			if err := ensureTicketBelongsToProject(ctx, tx, current.ProjectID, *input.ParentTicketID.Value, ErrParentTicketNotFound); err != nil {
				return UpdateResult{}, err
			}
			if err := ensureParentDoesNotCreateCycle(ctx, tx, current.ID, *input.ParentTicketID.Value); err != nil {
				return UpdateResult{}, err
			}
			builder.SetParentTicketID(*input.ParentTicketID.Value)
		}
	}
	if statusChangeDisposition != ticketing.StatusChangeRunDispositionRetain || (archivedChanged && input.Archived.Value) {
		builder.ClearCurrentRunID()
	}
	if statusChangeDisposition != ticketing.StatusChangeRunDispositionRetain || targetMachineChanged || (archivedChanged && input.Archived.Value) {
		ResetRetryBaseline(builder, current)
	}

	if _, err := builder.Save(ctx); err != nil {
		return UpdateResult{}, mapTicketWriteError("update ticket", err)
	}
	if statusChangeDisposition != ticketing.StatusChangeRunDispositionRetain || targetMachineChanged || (archivedChanged && input.Archived.Value) {
		if err := releaseTicketAgentClaim(ctx, tx, current, entagentrun.StatusTerminated); err != nil {
			return UpdateResult{}, err
		}
	}

	if input.ParentTicketID.Set {
		if err := syncSubIssueDependencies(ctx, tx, current.ID, input.ParentTicketID.Value); err != nil {
			return UpdateResult{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return UpdateResult{}, fmt.Errorf("commit ticket update tx: %w", err)
	}

	updated, err := r.Get(ctx, current.ID)
	if err != nil {
		return UpdateResult{}, err
	}

	result := UpdateResult{Ticket: updated}
	if releasedRunID != nil && releasedHookName != "" {
		result.DeferredHook = &DeferredLifecycleHook{
			RunID:      *releasedRunID,
			WorkflowID: releasedWorkflowID,
			HookName:   releasedHookName,
		}
	}

	return result, nil
}

// ResumeRetry clears a repeated-stall retry pause and makes the ticket schedulable again.
func (r *EntRepository) ResumeRetry(ctx context.Context, input ResumeRetryInput) (Ticket, error) {
	if r.client == nil {
		return Ticket{}, errUnavailable
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return Ticket{}, fmt.Errorf("start resume retry tx: %w", err)
	}
	defer rollback(tx)

	current, err := tx.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		if ent.IsNotFound(err) {
			return Ticket{}, ErrTicketNotFound
		}
		return Ticket{}, fmt.Errorf("load ticket for resume retry: %w", err)
	}

	if !current.RetryPaused || current.PauseReason != ticketing.PauseReasonRepeatedStalls.String() {
		return Ticket{}, ErrRetryResumeConflict
	}

	update := tx.Ticket.UpdateOneID(current.ID).
		SetRetryToken(NewRetryToken()).
		SetRetryPaused(false).
		ClearPauseReason().
		ClearNextRetryAt()
	if current.StallCount != 0 {
		update.SetStallCount(0)
	}

	if _, err := update.Save(ctx); err != nil {
		return Ticket{}, mapTicketWriteError("resume ticket retry", err)
	}

	if err := tx.Commit(); err != nil {
		return Ticket{}, fmt.Errorf("commit resume retry tx: %w", err)
	}

	return r.Get(ctx, current.ID)
}

// AddDependency creates a dependency edge between two tickets.
func (r *EntRepository) AddDependency(ctx context.Context, input AddDependencyInput) (Dependency, error) {
	if r.client == nil {
		return Dependency{}, errUnavailable
	}
	if input.TicketID == input.TargetTicketID {
		return Dependency{}, ErrInvalidDependency
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return Dependency{}, fmt.Errorf("start add ticket dependency tx: %w", err)
	}
	defer rollback(tx)

	source, err := tx.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		return Dependency{}, mapTicketReadError("get source ticket", err)
	}
	if err := ensureTicketBelongsToProject(ctx, tx, source.ProjectID, input.TargetTicketID, ErrTicketNotFound); err != nil {
		return Dependency{}, err
	}

	var dependency *ent.TicketDependency
	if input.Type == DependencyTypeSubIssue {
		if err := ensureParentDoesNotCreateCycle(ctx, tx, source.ID, input.TargetTicketID); err != nil {
			return Dependency{}, err
		}
		if _, err := tx.Ticket.UpdateOneID(source.ID).SetParentTicketID(input.TargetTicketID).Save(ctx); err != nil {
			return Dependency{}, mapTicketWriteError("set ticket parent", err)
		}
		dependency, err = ensureSubIssueDependency(ctx, tx, source.ID, input.TargetTicketID)
		if err != nil {
			return Dependency{}, err
		}
	} else {
		dependency, err = tx.TicketDependency.Create().
			SetSourceTicketID(source.ID).
			SetTargetTicketID(input.TargetTicketID).
			SetType(toEntDependencyType(input.Type)).
			Save(ctx)
		if err != nil {
			return Dependency{}, mapTicketWriteError("create ticket dependency", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Dependency{}, fmt.Errorf("commit add ticket dependency tx: %w", err)
	}

	dependency, err = r.client.TicketDependency.Query().
		Where(entticketdependency.ID(dependency.ID)).
		WithTargetTicket(func(query *ent.TicketQuery) {
			query.WithStatus()
		}).
		Only(ctx)
	if err != nil {
		return Dependency{}, fmt.Errorf("reload ticket dependency: %w", err)
	}

	return mapDependency(dependency), nil
}

// RemoveDependency deletes a dependency edge from a ticket.
func (r *EntRepository) RemoveDependency(ctx context.Context, ticketID uuid.UUID, dependencyID uuid.UUID) (DeleteDependencyResult, error) {
	if r.client == nil {
		return DeleteDependencyResult{}, errUnavailable
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return DeleteDependencyResult{}, fmt.Errorf("start delete ticket dependency tx: %w", err)
	}
	defer rollback(tx)

	dependency, err := tx.TicketDependency.Query().
		Where(
			entticketdependency.ID(dependencyID),
			entticketdependency.Or(
				entticketdependency.SourceTicketIDEQ(ticketID),
				entticketdependency.TargetTicketIDEQ(ticketID),
			),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return DeleteDependencyResult{}, ErrDependencyNotFound
		}
		return DeleteDependencyResult{}, fmt.Errorf("get ticket dependency for delete: %w", err)
	}
	if dependency.Type == entticketdependency.TypeSubIssue && dependency.SourceTicketID != ticketID {
		return DeleteDependencyResult{}, ErrDependencyNotFound
	}

	if dependency.Type == entticketdependency.TypeSubIssue {
		source, sourceErr := tx.Ticket.Get(ctx, ticketID)
		if sourceErr != nil {
			return DeleteDependencyResult{}, mapTicketReadError("get ticket for dependency delete", sourceErr)
		}
		if source.ParentTicketID != nil && *source.ParentTicketID == dependency.TargetTicketID {
			if _, err := tx.Ticket.UpdateOneID(ticketID).ClearParentTicketID().Save(ctx); err != nil {
				return DeleteDependencyResult{}, mapTicketWriteError("clear ticket parent", err)
			}
		}
	}

	if err := tx.TicketDependency.DeleteOneID(dependencyID).Exec(ctx); err != nil {
		return DeleteDependencyResult{}, mapTicketWriteError("delete ticket dependency", err)
	}
	if err := tx.Commit(); err != nil {
		return DeleteDependencyResult{}, fmt.Errorf("commit delete ticket dependency tx: %w", err)
	}

	return DeleteDependencyResult{DeletedDependencyID: dependencyID}, nil
}

// AddExternalLink creates a new external issue or PR association for a ticket.
func (r *EntRepository) AddExternalLink(ctx context.Context, input AddExternalLinkInput) (ExternalLink, error) {
	if r.client == nil {
		return ExternalLink{}, errUnavailable
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return ExternalLink{}, fmt.Errorf("start add ticket external link tx: %w", err)
	}
	defer rollback(tx)

	source, err := tx.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		return ExternalLink{}, mapTicketReadError("get ticket for external link create", err)
	}

	builder := tx.TicketExternalLink.Create().
		SetTicketID(source.ID).
		SetLinkType(toEntExternalLinkType(input.LinkType)).
		SetURL(input.URL).
		SetExternalID(input.ExternalID).
		SetRelation(toEntExternalLinkRelation(input.Relation))
	if input.Title != "" {
		builder.SetTitle(input.Title)
	}
	if input.Status != "" {
		builder.SetStatus(input.Status)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return ExternalLink{}, mapTicketWriteError("create ticket external link", err)
	}

	if strings.TrimSpace(source.ExternalRef) == "" {
		if _, err := tx.Ticket.UpdateOneID(source.ID).SetExternalRef(input.ExternalID).Save(ctx); err != nil {
			return ExternalLink{}, mapTicketWriteError("set ticket external_ref", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return ExternalLink{}, fmt.Errorf("commit add ticket external link tx: %w", err)
	}

	return mapExternalLink(created), nil
}

// ListComments returns user discussion comments ordered oldest-first for stable thread rendering.
func (r *EntRepository) ListComments(ctx context.Context, ticketID uuid.UUID) ([]Comment, error) {
	if r.client == nil {
		return nil, errUnavailable
	}
	if _, err := r.client.Ticket.Get(ctx, ticketID); err != nil {
		return nil, mapTicketReadError("get ticket for comment list", err)
	}

	items, err := r.client.TicketComment.Query().
		Where(entticketcomment.TicketIDEQ(ticketID)).
		Order(ent.Asc(entticketcomment.FieldCreatedAt), ent.Asc(entticketcomment.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket comments: %w", err)
	}

	comments := make([]Comment, 0, len(items))
	for _, item := range items {
		comments = append(comments, mapComment(item))
	}

	return comments, nil
}

// ListCommentRevisions returns immutable comment history oldest-first.
func (r *EntRepository) ListCommentRevisions(ctx context.Context, ticketID uuid.UUID, commentID uuid.UUID) ([]CommentRevision, error) {
	if r.client == nil {
		return nil, errUnavailable
	}

	comment, err := r.client.TicketComment.Query().
		Where(
			entticketcomment.IDEQ(commentID),
			entticketcomment.TicketIDEQ(ticketID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("get ticket comment for revisions: %w", err)
	}

	revisions, err := r.client.TicketCommentRevision.Query().
		Where(entticketcommentrevision.CommentIDEQ(comment.ID)).
		Order(ent.Asc(entticketcommentrevision.FieldRevisionNumber), ent.Asc(entticketcommentrevision.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ticket comment revisions: %w", err)
	}
	if len(revisions) == 0 {
		return []CommentRevision{syntheticInitialRevision(comment)}, nil
	}

	items := make([]CommentRevision, 0, len(revisions))
	for _, item := range revisions {
		items = append(items, mapCommentRevision(item))
	}

	return items, nil
}

// AddComment creates a new user discussion comment on a ticket.
func (r *EntRepository) AddComment(ctx context.Context, input AddCommentInput) (Comment, error) {
	if r.client == nil {
		return Comment{}, errUnavailable
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return Comment{}, fmt.Errorf("start add ticket comment tx: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.Ticket.Get(ctx, input.TicketID); err != nil {
		return Comment{}, mapTicketReadError("get ticket for comment create", err)
	}

	now := timeNowUTC()
	createdBy := resolveCreatedBy(input.CreatedBy)
	item, err := tx.TicketComment.Create().
		SetTicketID(input.TicketID).
		SetBody(input.Body).
		SetCreatedBy(createdBy).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return Comment{}, mapTicketWriteError("create ticket comment", err)
	}
	if err := appendCommentRevisionTx(ctx, tx, item.ID, 1, item.Body, createdBy, now, ""); err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("commit add ticket comment tx: %w", err)
	}

	return mapComment(item), nil
}

// UpdateComment updates the markdown body of an existing ticket discussion comment.
func (r *EntRepository) UpdateComment(ctx context.Context, input UpdateCommentInput) (Comment, error) {
	if r.client == nil {
		return Comment{}, errUnavailable
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return Comment{}, fmt.Errorf("start update ticket comment tx: %w", err)
	}
	defer rollback(tx)

	existing, err := tx.TicketComment.Query().
		Where(
			entticketcomment.IDEQ(input.CommentID),
			entticketcomment.TicketIDEQ(input.TicketID),
			entticketcomment.IsDeleted(false),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return Comment{}, ErrCommentNotFound
		}
		return Comment{}, fmt.Errorf("get ticket comment for update: %w", err)
	}

	now := timeNowUTC()
	revisionNumber, err := ensureInitialRevisionTx(ctx, tx, existing)
	if err != nil {
		return Comment{}, err
	}
	editor := resolveCreatedBy(input.EditedBy)
	revisionNumber++

	item, err := tx.TicketComment.UpdateOneID(existing.ID).
		SetBody(input.Body).
		SetUpdatedAt(now).
		SetEditedAt(now).
		SetEditCount(revisionNumber - 1).
		SetLastEditedBy(editor).
		Save(ctx)
	if err != nil {
		return Comment{}, mapTicketWriteError("update ticket comment", err)
	}
	if err := appendCommentRevisionTx(ctx, tx, existing.ID, revisionNumber, input.Body, editor, now, input.EditReason); err != nil {
		return Comment{}, err
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("commit update ticket comment tx: %w", err)
	}

	return mapComment(item), nil
}

// RemoveExternalLink deletes an external issue or PR association from a ticket.
func (r *EntRepository) RemoveExternalLink(ctx context.Context, ticketID uuid.UUID, externalLinkID uuid.UUID) (DeleteExternalLinkResult, error) {
	if r.client == nil {
		return DeleteExternalLinkResult{}, errUnavailable
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return DeleteExternalLinkResult{}, fmt.Errorf("start delete ticket external link tx: %w", err)
	}
	defer rollback(tx)

	link, err := tx.TicketExternalLink.Query().
		Where(
			entticketexternallink.ID(externalLinkID),
			entticketexternallink.TicketIDEQ(ticketID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return DeleteExternalLinkResult{}, ErrExternalLinkNotFound
		}
		return DeleteExternalLinkResult{}, fmt.Errorf("get ticket external link for delete: %w", err)
	}

	source, err := tx.Ticket.Get(ctx, ticketID)
	if err != nil {
		return DeleteExternalLinkResult{}, mapTicketReadError("get ticket for external link delete", err)
	}

	if err := tx.TicketExternalLink.DeleteOneID(externalLinkID).Exec(ctx); err != nil {
		return DeleteExternalLinkResult{}, mapTicketWriteError("delete ticket external link", err)
	}

	if strings.TrimSpace(source.ExternalRef) == link.ExternalID {
		replacement, replacementErr := tx.TicketExternalLink.Query().
			Where(entticketexternallink.TicketIDEQ(ticketID)).
			Order(ent.Asc(entticketexternallink.FieldCreatedAt), ent.Asc(entticketexternallink.FieldID)).
			First(ctx)
		switch {
		case ent.IsNotFound(replacementErr):
			if _, err := tx.Ticket.UpdateOneID(ticketID).ClearExternalRef().Save(ctx); err != nil {
				return DeleteExternalLinkResult{}, mapTicketWriteError("clear ticket external_ref", err)
			}
		case replacementErr != nil:
			return DeleteExternalLinkResult{}, fmt.Errorf("select replacement external link: %w", replacementErr)
		default:
			if _, err := tx.Ticket.UpdateOneID(ticketID).SetExternalRef(replacement.ExternalID).Save(ctx); err != nil {
				return DeleteExternalLinkResult{}, mapTicketWriteError("replace ticket external_ref", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return DeleteExternalLinkResult{}, fmt.Errorf("commit delete ticket external link tx: %w", err)
	}

	return DeleteExternalLinkResult{DeletedExternalLinkID: externalLinkID}, nil
}

// RemoveComment deletes a user discussion comment from a ticket.
func (r *EntRepository) RemoveComment(ctx context.Context, ticketID uuid.UUID, commentID uuid.UUID) (DeleteCommentResult, error) {
	if r.client == nil {
		return DeleteCommentResult{}, errUnavailable
	}

	now := timeNowUTC()
	deleted, err := r.client.TicketComment.Update().
		Where(
			entticketcomment.IDEQ(commentID),
			entticketcomment.TicketIDEQ(ticketID),
			entticketcomment.IsDeleted(false),
		).
		SetIsDeleted(true).
		SetDeletedAt(now).
		SetDeletedBy(defaultCreatedBy).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return DeleteCommentResult{}, fmt.Errorf("soft delete ticket comment: %w", err)
	}
	if deleted == 0 {
		return DeleteCommentResult{}, ErrCommentNotFound
	}

	return DeleteCommentResult{DeletedCommentID: commentID}, nil
}

func ensureInitialRevisionTx(ctx context.Context, tx *ent.Tx, comment *ent.TicketComment) (int, error) {
	latest, err := tx.TicketCommentRevision.Query().
		Where(entticketcommentrevision.CommentIDEQ(comment.ID)).
		Order(ent.Desc(entticketcommentrevision.FieldRevisionNumber), ent.Desc(entticketcommentrevision.FieldID)).
		First(ctx)
	switch {
	case err == nil:
		return latest.RevisionNumber, nil
	case !ent.IsNotFound(err):
		return 0, fmt.Errorf("load latest ticket comment revision: %w", err)
	}

	if err := appendCommentRevisionTx(ctx, tx, comment.ID, 1, comment.Body, comment.CreatedBy, comment.CreatedAt, ""); err != nil {
		return 0, err
	}

	return 1, nil
}

func appendCommentRevisionTx(
	ctx context.Context,
	tx *ent.Tx,
	commentID uuid.UUID,
	revisionNumber int,
	bodyMarkdown string,
	editedBy string,
	editedAt time.Time,
	editReason string,
) error {
	create := tx.TicketCommentRevision.Create().
		SetCommentID(commentID).
		SetRevisionNumber(revisionNumber).
		SetBodyMarkdown(bodyMarkdown).
		SetEditedBy(resolveCreatedBy(editedBy)).
		SetEditedAt(editedAt)
	if trimmed := strings.TrimSpace(editReason); trimmed != "" {
		create.SetEditReason(trimmed)
	}
	if _, err := create.Save(ctx); err != nil {
		return mapTicketWriteError("create ticket comment revision", err)
	}

	return nil
}

func syntheticInitialRevision(comment *ent.TicketComment) CommentRevision {
	return CommentRevision{
		ID:             uuid.Nil,
		CommentID:      comment.ID,
		RevisionNumber: 1,
		BodyMarkdown:   comment.Body,
		EditedBy:       comment.CreatedBy,
		EditedAt:       comment.CreatedAt,
	}
}

func (r *EntRepository) ensureProjectExists(ctx context.Context, projectID uuid.UUID) error {
	exists, err := r.client.Project.Query().Where(project.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}

	return nil
}

func ensureProjectExistsTx(ctx context.Context, tx *ent.Tx, projectID uuid.UUID) error {
	exists, err := tx.Project.Query().Where(project.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}

	return nil
}

func resolveCreateStatusID(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, inputStatusID *uuid.UUID) (uuid.UUID, error) {
	if inputStatusID != nil {
		if err := ensureStatusBelongsToProject(ctx, tx, projectID, *inputStatusID); err != nil {
			return uuid.UUID{}, err
		}
		return *inputStatusID, nil
	}

	defaultStatus, err := tx.TicketStatus.Query().
		Where(
			entticketstatus.ProjectIDEQ(projectID),
			entticketstatus.IsDefault(true),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return uuid.UUID{}, ErrStatusNotFound
		}
		return uuid.UUID{}, fmt.Errorf("get default project ticket status: %w", err)
	}

	return defaultStatus.ID, nil
}

func mapTicketReadError(action string, err error) error {
	if ent.IsNotFound(err) {
		return ErrTicketNotFound
	}

	return fmt.Errorf("%s: %w", action, err)
}

func mapTicketWriteError(action string, err error) error {
	switch {
	case ent.IsConstraintError(err):
		switch message := strings.ToLower(err.Error()); {
		case strings.Contains(message, "ticketdependency_source_ticket_id_target_ticket_id_type"):
			return ErrDependencyConflict
		case strings.Contains(message, "ticket_external_links_ticket_id_external_id"),
			strings.Contains(message, "ticketexternallink_ticket_id_external_id"),
			(strings.Contains(message, "ticket_external_links") && strings.Contains(message, "external_id")):
			return ErrExternalLinkConflict
		case strings.Contains(message, "ticket_project_id_identifier"),
			strings.Contains(message, "ticket_identifier"):
			return ErrTicketConflict
		default:
			return fmt.Errorf("%s: %w", action, err)
		}
	case ent.IsNotFound(err):
		return ErrTicketNotFound
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

func ensureStatusBelongsToProject(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, statusID uuid.UUID) error {
	exists, err := tx.TicketStatus.Query().
		Where(
			entticketstatus.ID(statusID),
			entticketstatus.ProjectIDEQ(projectID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check ticket status existence: %w", err)
	}
	if !exists {
		return ErrStatusNotFound
	}

	return nil
}

func ensureWorkflowBelongsToProject(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, workflowID uuid.UUID) error {
	exists, err := tx.Workflow.Query().
		Where(
			entworkflow.ID(workflowID),
			entworkflow.ProjectIDEQ(projectID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check workflow existence: %w", err)
	}
	if !exists {
		return ErrWorkflowNotFound
	}

	return nil
}

func ensureStatusAllowedByWorkflowFinishSet(ctx context.Context, tx *ent.Tx, workflowID uuid.UUID, statusID uuid.UUID) error {
	workflowItem, err := tx.Workflow.Query().
		Where(entworkflow.IDEQ(workflowID)).
		WithFinishStatuses().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ErrWorkflowNotFound
		}
		return fmt.Errorf("load workflow finish statuses: %w", err)
	}

	for _, finishStatus := range workflowItem.Edges.FinishStatuses {
		if finishStatus.ID == statusID {
			return nil
		}
	}
	return ErrStatusNotAllowed
}

func classifyStatusChangeRunDisposition(
	ctx context.Context,
	tx *ent.Tx,
	current *ent.Ticket,
	nextStatusID uuid.UUID,
) (ticketing.StatusChangeRunDisposition, error) {
	if current == nil || current.CurrentRunID == nil {
		return ticketing.StatusChangeRunDispositionRetain, nil
	}
	if current.WorkflowID == nil {
		return ticketing.StatusChangeRunDispositionCancel, nil
	}

	workflowItem, err := tx.Workflow.Query().
		Where(entworkflow.IDEQ(*current.WorkflowID)).
		WithPickupStatuses().
		WithFinishStatuses().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", ErrWorkflowNotFound
		}
		return "", fmt.Errorf("load workflow status ownership: %w", err)
	}

	pickupStatusIDs := make([]uuid.UUID, 0, len(workflowItem.Edges.PickupStatuses))
	for _, status := range workflowItem.Edges.PickupStatuses {
		pickupStatusIDs = append(pickupStatusIDs, status.ID)
	}
	finishStatusIDs := make([]uuid.UUID, 0, len(workflowItem.Edges.FinishStatuses))
	for _, status := range workflowItem.Edges.FinishStatuses {
		finishStatusIDs = append(finishStatusIDs, status.ID)
	}

	return ticketing.ClassifyStatusChangeRunDisposition(
		true,
		nextStatusID,
		pickupStatusIDs,
		finishStatusIDs,
	), nil
}

func ensureTicketBelongsToProject(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, ticketID uuid.UUID, notFound error) error {
	exists, err := tx.Ticket.Query().
		Where(
			entticket.ID(ticketID),
			entticket.ProjectIDEQ(projectID),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check ticket existence: %w", err)
	}
	if !exists {
		return notFound
	}

	return nil
}

func ensureTargetMachineBelongsToProjectOrganization(ctx context.Context, tx *ent.Tx, projectID uuid.UUID, machineID uuid.UUID) error {
	exists, err := tx.Machine.Query().
		Where(
			entmachine.IDEQ(machineID),
			entmachine.HasOrganizationWith(entorganization.HasProjectsWith(project.ID(projectID))),
		).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("check target machine existence: %w", err)
	}
	if !exists {
		return ErrTargetMachineNotFound
	}

	return nil
}

func releaseTicketAgentClaim(ctx context.Context, tx *ent.Tx, ticketItem *ent.Ticket, runStatus entagentrun.Status) error {
	if ticketItem == nil {
		return nil
	}

	var runItem *ent.AgentRun
	if ticketItem.CurrentRunID != nil {
		currentRun, err := tx.AgentRun.Get(ctx, *ticketItem.CurrentRunID)
		if err != nil {
			return fmt.Errorf("load current agent run: %w", err)
		}
		runItem = currentRun

		runUpdate := tx.AgentRun.UpdateOneID(currentRun.ID).
			SetStatus(runStatus).
			SetTerminalAt(timeNowUTC()).
			ClearSessionID().
			ClearRuntimeStartedAt().
			ClearLastHeartbeatAt()
		if runStatus != entagentrun.StatusErrored {
			runUpdate.SetLastError("")
		}
		if _, err := runUpdate.Save(ctx); err != nil {
			return fmt.Errorf("finalize current agent run: %w", err)
		}
	}

	if runItem != nil {
		if _, err := tx.Agent.UpdateOneID(runItem.AgentID).
			SetRuntimeControlState(entagent.RuntimeControlStateActive).
			Save(ctx); err != nil {
			return fmt.Errorf("reset current run agent runtime control state: %w", err)
		}
	}

	return nil
}

// InstallRetryTokenHooks keeps retry token semantics consistent for direct ent mutations.
func InstallRetryTokenHooks(client *ent.Client) {
	if client == nil {
		return
	}

	client.Ticket.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			ticketMutation, ok := mutation.(*ent.TicketMutation)
			if !ok {
				return next.Mutate(ctx, mutation)
			}

			ensureTicketCreateRetryToken(ticketMutation)
			normalizeTicketStatusTransition(ticketMutation)

			return next.Mutate(ctx, mutation)
		})
	})
}

// ScheduleRetryOne rotates the retry token and records a delayed retry intent.
func ScheduleRetryOne(update *ent.TicketUpdateOne, nextRetryAt time.Time, pauseReason string) *ent.TicketUpdateOne {
	if update == nil {
		return nil
	}

	update.SetRetryToken(NewRetryToken()).
		SetNextRetryAt(nextRetryAt).
		SetRetryPaused(pauseReason != "")
	if pauseReason == "" {
		return update.ClearPauseReason()
	}

	return update.SetPauseReason(pauseReason)
}

// ScheduleRetry rotates the retry token and records a delayed retry intent.
func ScheduleRetry(update *ent.TicketUpdate, nextRetryAt time.Time, pauseReason string) *ent.TicketUpdate {
	if update == nil {
		return nil
	}

	update.
		SetRetryToken(NewRetryToken()).
		SetNextRetryAt(nextRetryAt).
		SetRetryPaused(pauseReason != "")
	if pauseReason == "" {
		return update.ClearPauseReason()
	}

	return update.SetPauseReason(pauseReason)
}

// ResetRetryBaseline clears active retry-cycle state after a healthy/manual-forward transition
// and rotates the retry token so stale delayed retries are discarded.
// attempt_count stays cumulative per the PRD, while current failure streak state is normalized.
func ResetRetryBaseline(update *ent.TicketUpdateOne, current *ent.Ticket) *ent.TicketUpdateOne {
	if update == nil || current == nil {
		return update
	}

	update.SetRetryToken(NewRetryToken())
	if current.ConsecutiveErrors != 0 {
		update.SetConsecutiveErrors(0)
	}
	if current.StallCount != 0 {
		update.SetStallCount(0)
	}
	if current.NextRetryAt != nil {
		update.ClearNextRetryAt()
	}
	if current.RetryPaused {
		update.SetRetryPaused(false)
	}
	if current.PauseReason != "" {
		update.ClearPauseReason()
	}

	return update
}

func ensureTicketCreateRetryToken(mutation *ent.TicketMutation) {
	if mutation == nil || !mutation.Op().Is(ent.OpCreate) {
		return
	}
	if _, ok := mutation.RetryToken(); ok {
		return
	}

	mutation.SetRetryToken(NewRetryToken())
}

func normalizeTicketStatusTransition(mutation *ent.TicketMutation) {
	if mutation == nil || !mutation.Op().Is(ent.OpUpdate|ent.OpUpdateOne) {
		return
	}
	if _, ok := mutation.StatusID(); !ok {
		return
	}
	if _, ok := mutation.RetryToken(); !ok {
		mutation.SetRetryToken(NewRetryToken())
	}

	mutation.SetConsecutiveErrors(0)
	mutation.ClearNextRetryAt()
	mutation.SetRetryPaused(false)
	mutation.ClearPauseReason()
}

func ensureParentDoesNotCreateCycle(ctx context.Context, tx *ent.Tx, ticketID uuid.UUID, parentTicketID uuid.UUID) error {
	if ticketID == parentTicketID {
		return ErrInvalidDependency
	}

	seen := map[uuid.UUID]struct{}{ticketID: {}}
	currentID := parentTicketID
	for currentID != uuid.Nil {
		if _, ok := seen[currentID]; ok {
			return ErrInvalidDependency
		}
		seen[currentID] = struct{}{}

		current, err := tx.Ticket.Get(ctx, currentID)
		if err != nil {
			if ent.IsNotFound(err) {
				return ErrParentTicketNotFound
			}
			return fmt.Errorf("load ticket parent chain: %w", err)
		}
		if current.ParentTicketID == nil {
			return nil
		}
		currentID = *current.ParentTicketID
	}

	return nil
}

func syncSubIssueDependencies(ctx context.Context, tx *ent.Tx, ticketID uuid.UUID, parentTicketID *uuid.UUID) error {
	existing, err := tx.TicketDependency.Query().
		Where(
			entticketdependency.SourceTicketIDEQ(ticketID),
			entticketdependency.TypeEQ(entticketdependency.TypeSubIssue),
		).
		All(ctx)
	if err != nil {
		return fmt.Errorf("query sub-issue dependencies: %w", err)
	}

	keepID := uuid.Nil
	if parentTicketID != nil {
		for _, dependency := range existing {
			if dependency.TargetTicketID == *parentTicketID {
				keepID = dependency.ID
				break
			}
		}
	}

	for _, dependency := range existing {
		if dependency.ID == keepID {
			continue
		}
		if err := tx.TicketDependency.DeleteOneID(dependency.ID).Exec(ctx); err != nil {
			return fmt.Errorf("delete stale sub-issue dependency: %w", err)
		}
	}

	if parentTicketID == nil || keepID != uuid.Nil {
		return nil
	}

	_, err = tx.TicketDependency.Create().
		SetSourceTicketID(ticketID).
		SetTargetTicketID(*parentTicketID).
		SetType(entticketdependency.TypeSubIssue).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("create sub-issue dependency: %w", err)
	}

	return nil
}

func ensureSubIssueDependency(ctx context.Context, tx *ent.Tx, sourceTicketID uuid.UUID, targetTicketID uuid.UUID) (*ent.TicketDependency, error) {
	if err := syncSubIssueDependencies(ctx, tx, sourceTicketID, &targetTicketID); err != nil {
		return nil, err
	}

	dependency, err := tx.TicketDependency.Query().
		Where(
			entticketdependency.SourceTicketIDEQ(sourceTicketID),
			entticketdependency.TargetTicketIDEQ(targetTicketID),
			entticketdependency.TypeEQ(entticketdependency.TypeSubIssue),
		).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("reload sub-issue dependency: %w", err)
	}

	return dependency, nil
}

func nextTicketIdentifier(ctx context.Context, tx *ent.Tx, projectID uuid.UUID) (string, error) {
	items, err := tx.Ticket.Query().
		Where(entticket.ProjectIDEQ(projectID)).
		Select(entticket.FieldIdentifier).
		All(ctx)
	if err != nil {
		return "", fmt.Errorf("list project identifiers: %w", err)
	}

	maxValue := 0
	for _, item := range items {
		value, ok := parseIdentifierSequence(item.Identifier)
		if ok && value > maxValue {
			maxValue = value
		}
	}

	return fmt.Sprintf("%s-%d", defaultIdentifierPrefix, maxValue+1), nil
}

func parseIdentifierSequence(identifier string) (int, bool) {
	if !strings.HasPrefix(identifier, defaultIdentifierPrefix+"-") {
		return 0, false
	}

	value, err := strconv.Atoi(strings.TrimPrefix(identifier, defaultIdentifierPrefix+"-"))
	if err != nil || value < 1 {
		return 0, false
	}

	return value, true
}

func resolveCreatedBy(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return defaultCreatedBy
	}

	return strings.TrimSpace(raw)
}

func toEntTicketPriority(priority Priority) entticket.Priority {
	return entticket.Priority(priority.String())
}

func toEntTicketPriorities(priorities []Priority) []entticket.Priority {
	items := make([]entticket.Priority, 0, len(priorities))
	for _, priority := range priorities {
		items = append(items, toEntTicketPriority(priority))
	}
	return items
}

func toEntTicketType(ticketType Type) entticket.Type {
	return entticket.Type(ticketType.String())
}

func toEntDependencyType(dependencyType DependencyType) entticketdependency.Type {
	return entticketdependency.Type(dependencyType.String())
}

func toEntExternalLinkType(linkType ExternalLinkType) entticketexternallink.LinkType {
	return entticketexternallink.LinkType(linkType.String())
}

func toEntExternalLinkRelation(relation ExternalLinkRelation) entticketexternallink.Relation {
	return entticketexternallink.Relation(relation.String())
}

func optionalUUIDPointerEqual(left *uuid.UUID, right *uuid.UUID) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func mapTicket(item *ent.Ticket) Ticket {
	result := Ticket{
		ID:                   item.ID,
		ProjectID:            item.ProjectID,
		Identifier:           item.Identifier,
		Title:                item.Title,
		Description:          item.Description,
		StatusID:             item.StatusID,
		Priority:             Priority(item.Priority),
		Archived:             item.Archived,
		Type:                 Type(item.Type),
		WorkflowID:           item.WorkflowID,
		CurrentRunID:         item.CurrentRunID,
		TargetMachineID:      item.TargetMachineID,
		CreatedBy:            item.CreatedBy,
		Children:             []TicketReference{},
		Dependencies:         []Dependency{},
		IncomingDependencies: []Dependency{},
		ExternalLinks:        []ExternalLink{},
		ExternalRef:          item.ExternalRef,
		BudgetUSD:            item.BudgetUsd,
		CostTokensInput:      item.CostTokensInput,
		CostTokensOutput:     item.CostTokensOutput,
		CostAmount:           item.CostAmount,
		AttemptCount:         item.AttemptCount,
		ConsecutiveErrors:    item.ConsecutiveErrors,
		StartedAt:            item.StartedAt,
		CompletedAt:          item.CompletedAt,
		NextRetryAt:          item.NextRetryAt,
		RetryPaused:          item.RetryPaused,
		PauseReason:          item.PauseReason,
		CreatedAt:            item.CreatedAt,
	}

	if item.Edges.Status != nil {
		result.StatusName = item.Edges.Status.Name
	}
	if item.Edges.Parent != nil {
		parent := mapTicketReference(item.Edges.Parent)
		result.Parent = &parent
	}
	for _, child := range item.Edges.Children {
		result.Children = append(result.Children, mapTicketReference(child))
	}
	for _, dependency := range item.Edges.OutgoingDependencies {
		result.Dependencies = append(result.Dependencies, mapDependency(dependency))
	}
	for _, dependency := range item.Edges.IncomingDependencies {
		result.IncomingDependencies = append(result.IncomingDependencies, mapIncomingDependency(dependency))
	}
	for _, externalLink := range item.Edges.ExternalLinks {
		result.ExternalLinks = append(result.ExternalLinks, mapExternalLink(externalLink))
	}

	return result
}

func mapDependency(item *ent.TicketDependency) Dependency {
	dependency := Dependency{
		ID:   item.ID,
		Type: DependencyType(item.Type),
	}
	if item.Edges.TargetTicket != nil {
		dependency.Target = mapTicketReference(item.Edges.TargetTicket)
	}

	return dependency
}

func mapIncomingDependency(item *ent.TicketDependency) Dependency {
	dependency := Dependency{
		ID:   item.ID,
		Type: DependencyType(item.Type),
	}
	if item.Edges.SourceTicket != nil {
		dependency.Target = mapTicketReference(item.Edges.SourceTicket)
	}

	return dependency
}

func mapExternalLink(item *ent.TicketExternalLink) ExternalLink {
	return ExternalLink{
		ID:         item.ID,
		LinkType:   ExternalLinkType(item.LinkType),
		URL:        item.URL,
		ExternalID: item.ExternalID,
		Title:      item.Title,
		Status:     item.Status,
		Relation:   ExternalLinkRelation(item.Relation),
		CreatedAt:  item.CreatedAt,
	}
}

func mapComment(item *ent.TicketComment) Comment {
	return Comment{
		ID:           item.ID,
		TicketID:     item.TicketID,
		BodyMarkdown: item.Body,
		CreatedBy:    item.CreatedBy,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
		EditedAt:     item.EditedAt,
		EditCount:    item.EditCount,
		LastEditedBy: item.LastEditedBy,
		IsDeleted:    item.IsDeleted,
		DeletedAt:    item.DeletedAt,
		DeletedBy:    item.DeletedBy,
	}
}

func mapCommentRevision(item *ent.TicketCommentRevision) CommentRevision {
	return CommentRevision{
		ID:             item.ID,
		CommentID:      item.CommentID,
		RevisionNumber: item.RevisionNumber,
		BodyMarkdown:   item.BodyMarkdown,
		EditedBy:       item.EditedBy,
		EditedAt:       item.EditedAt,
		EditReason:     item.EditReason,
	}
}

func mapTicketReference(item *ent.Ticket) TicketReference {
	reference := TicketReference{
		ID:         item.ID,
		Identifier: item.Identifier,
		Title:      item.Title,
		StatusID:   item.StatusID,
	}
	if item.Edges.Status != nil {
		reference.StatusName = item.Edges.Status.Name
	}

	return reference
}

func rollback(tx *ent.Tx) {
	if tx == nil {
		return
	}
	_ = tx.Rollback()
}

func reconcileBudgetPauseState(builder *ent.TicketUpdateOne, current *ent.Ticket, budgetUSD float64) {
	if builder == nil || current == nil {
		return
	}

	if ticketing.ShouldPauseForBudget(current.CostAmount, budgetUSD) {
		if !current.RetryPaused || current.PauseReason == "" || current.PauseReason == ticketing.PauseReasonBudgetExhausted.String() {
			builder.SetRetryPaused(true).
				SetPauseReason(ticketing.PauseReasonBudgetExhausted.String())
		}
		return
	}

	if current.PauseReason == ticketing.PauseReasonBudgetExhausted.String() {
		if current.RetryPaused {
			builder.SetRetryPaused(false)
		}
		builder.ClearPauseReason()
	}
}

func (r *EntRepository) LoadLifecycleHookRuntimeData(
	ctx context.Context,
	ticketID uuid.UUID,
	runID uuid.UUID,
	workflowID *uuid.UUID,
) (LifecycleHookRuntimeData, error) {
	runItem, err := r.client.AgentRun.Query().
		Where(entagentrun.IDEQ(runID)).
		WithAgent(func(query *ent.AgentQuery) {
			query.WithProvider()
		}).
		Only(ctx)
	if err != nil {
		return LifecycleHookRuntimeData{}, fmt.Errorf("load ticket hook run %s: %w", runID, err)
	}
	if runItem.Edges.Agent == nil {
		return LifecycleHookRuntimeData{}, fmt.Errorf("ticket hook run %s is missing agent", runID)
	}
	if runItem.Edges.Agent.Edges.Provider == nil {
		return LifecycleHookRuntimeData{}, fmt.Errorf("ticket hook run %s agent is missing provider", runID)
	}

	ticketItem, err := r.client.Ticket.Get(ctx, ticketID)
	if err != nil {
		return LifecycleHookRuntimeData{}, mapTicketReadError("load ticket for lifecycle hook", err)
	}

	resolvedWorkflowID := ticketItem.WorkflowID
	if workflowID != nil {
		resolvedWorkflowID = workflowID
	}
	if resolvedWorkflowID == nil {
		return LifecycleHookRuntimeData{TicketID: ticketItem.ID}, nil
	}

	workflowItem, err := r.client.Workflow.Query().
		Where(entworkflow.IDEQ(*resolvedWorkflowID)).
		WithCurrentVersion().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return LifecycleHookRuntimeData{TicketID: ticketItem.ID}, nil
		}
		return LifecycleHookRuntimeData{}, fmt.Errorf("load workflow %s for lifecycle hook: %w", *resolvedWorkflowID, err)
	}

	workspaces, err := r.client.TicketRepoWorkspace.Query().
		Where(entticketrepoworkspace.AgentRunIDEQ(runID)).
		Order(ent.Asc(entticketrepoworkspace.FieldRepoPath)).
		WithRepo(func(query *ent.ProjectRepoQuery) {
			query.Order(entprojectrepo.ByName())
		}).
		All(ctx)
	if err != nil {
		return LifecycleHookRuntimeData{}, fmt.Errorf("load ticket repo workspaces for run %s: %w", runID, err)
	}

	machineItem, err := r.client.Machine.Get(ctx, runItem.Edges.Agent.Edges.Provider.MachineID)
	if err != nil {
		return LifecycleHookRuntimeData{}, fmt.Errorf("load machine for ticket hook run %s: %w", runID, err)
	}

	repos := make([]domain.HookWorkspace, 0, len(workspaces))
	for _, workspace := range workspaces {
		repoName := strings.TrimSpace(workspace.RepoPath)
		if workspace.Edges.Repo != nil && strings.TrimSpace(workspace.Edges.Repo.Name) != "" {
			repoName = strings.TrimSpace(workspace.Edges.Repo.Name)
		}
		repos = append(repos, domain.HookWorkspace{
			RepoName: repoName,
			RepoPath: strings.TrimSpace(workspace.RepoPath),
		})
	}

	workspaceRoot := ""
	if len(workspaces) > 0 {
		workspaceRoot = strings.TrimSpace(workspaces[0].WorkspaceRoot)
	}

	typeLabel, parseErr := workflowdomain.ParseTypeLabel(workflowItem.Type)
	if parseErr != nil {
		typeLabel = workflowdomain.MustParseTypeLabel("unknown")
	}
	harnessContent := ""
	if workflowItem.Edges.CurrentVersion != nil {
		harnessContent = workflowItem.Edges.CurrentVersion.ContentMarkdown
	}
	workflowFamily := workflowdomain.ClassifyWorkflow(workflowdomain.WorkflowClassificationInput{
		TypeLabel:      typeLabel,
		WorkflowName:   workflowItem.Name,
		HarnessPath:    workflowItem.HarnessPath,
		HarnessContent: harnessContent,
	}).Family

	return LifecycleHookRuntimeData{
		TicketID:              ticketItem.ID,
		ProjectID:             ticketItem.ProjectID,
		AgentID:               runItem.AgentID,
		TicketIdentifier:      ticketItem.Identifier,
		AgentName:             runItem.Edges.Agent.Name,
		WorkflowType:          workflowItem.Type,
		WorkflowFamily:        string(workflowFamily),
		PlatformAccessAllowed: append([]string(nil), workflowItem.PlatformAccessAllowed...),
		Attempt:               ticketItem.AttemptCount + 1,
		WorkspaceRoot:         workspaceRoot,
		Hooks:                 cloneAnyMap(workflowItem.Hooks),
		Machine:               mapTicketHookMachine(machineItem),
		Workspaces:            repos,
	}, nil
}

func mapTicketHookMachine(item *ent.Machine) catalogdomain.Machine {
	if item == nil {
		return catalogdomain.Machine{}
	}

	return catalogdomain.Machine{
		ID:                 item.ID,
		OrganizationID:     item.OrganizationID,
		Name:               item.Name,
		Host:               item.Host,
		Port:               item.Port,
		SSHUser:            cloneOptionalText(item.SSHUser),
		SSHKeyPath:         cloneOptionalText(item.SSHKeyPath),
		Status:             catalogdomain.MachineStatus(item.Status),
		ConnectionMode:     mapStoredTicketMachineConnectionMode(item),
		AdvertisedEndpoint: cloneOptionalText(item.AdvertisedEndpoint),
		WorkspaceRoot:      cloneOptionalText(item.WorkspaceRoot),
		AgentCLIPath:       cloneOptionalText(item.AgentCliPath),
		EnvVars:            slices.Clone(item.EnvVars),
		Resources:          cloneMap(item.Resources),
		DaemonStatus: catalogdomain.MachineDaemonStatus{
			Registered:       item.DaemonRegistered,
			LastRegisteredAt: cloneOptionalTime(item.DaemonLastRegisteredAt),
			CurrentSessionID: cloneOptionalText(item.DaemonSessionID),
			SessionState:     catalogdomain.MachineTransportSessionState(item.DaemonSessionState),
		},
		ChannelCredential: catalogdomain.MachineChannelCredential{
			Kind:          catalogdomain.MachineChannelCredentialKind(item.ChannelCredentialKind),
			TokenID:       cloneOptionalText(item.ChannelTokenID),
			CertificateID: cloneOptionalText(item.ChannelCertificateID),
		},
	}
}

func mapStoredTicketMachineConnectionMode(item *ent.Machine) catalogdomain.MachineConnectionMode {
	if item == nil {
		return ""
	}
	if item.Host == catalogdomain.LocalMachineHost || item.Name == catalogdomain.LocalMachineName {
		return catalogdomain.MachineConnectionModeLocal
	}
	mode, err := catalogdomain.ParseStoredMachineConnectionMode(string(item.ConnectionMode), item.Host)
	if err != nil {
		return catalogdomain.MachineConnectionModeWSListener
	}
	return mode
}

func (r *EntRepository) RecordUsage(
	ctx context.Context,
	input RecordUsageInput,
	usageDelta ticketing.UsageDelta,
) (PersistedUsageResult, error) {
	if r.client == nil {
		return PersistedUsageResult{}, errUnavailable
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return PersistedUsageResult{}, fmt.Errorf("start ticket usage tx: %w", err)
	}
	defer rollback(tx)

	ticketItem, err := tx.Ticket.Get(ctx, input.TicketID)
	if err != nil {
		return PersistedUsageResult{}, mapTicketReadError("get ticket for usage", err)
	}

	agentItem, err := tx.Agent.Query().
		Where(entagent.IDEQ(input.AgentID)).
		WithProvider().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return PersistedUsageResult{}, fmt.Errorf("agent %s not found", input.AgentID)
		}
		return PersistedUsageResult{}, fmt.Errorf("get agent for usage: %w", err)
	}
	if agentItem.ProjectID != ticketItem.ProjectID {
		return PersistedUsageResult{}, fmt.Errorf("agent %s does not belong to ticket project %s", agentItem.ID, ticketItem.ProjectID)
	}
	if agentItem.Edges.Provider == nil {
		return PersistedUsageResult{}, fmt.Errorf("agent provider must be loaded for usage accounting")
	}

	pricingConfig := catalogdomain.ResolveAgentProviderPricingConfig(
		catalogdomain.AgentProviderAdapterType(agentItem.Edges.Provider.AdapterType),
		agentItem.Edges.Provider.ModelName,
		agentItem.Edges.Provider.CostPerInputToken,
		agentItem.Edges.Provider.CostPerOutputToken,
		agentItem.Edges.Provider.PricingConfig,
	)
	resolvedCost, err := usageDelta.ResolveCost(pricingConfig)
	if err != nil {
		return PersistedUsageResult{}, err
	}

	nextCostAmount := ticketItem.CostAmount + resolvedCost.AmountUSD
	update := tx.Ticket.UpdateOneID(ticketItem.ID).
		AddCostTokensInput(usageDelta.InputTokens).
		AddCostTokensOutput(usageDelta.OutputTokens).
		AddCostAmount(resolvedCost.AmountUSD)

	if ticketing.ShouldPauseForBudget(nextCostAmount, ticketItem.BudgetUsd) &&
		(!ticketItem.RetryPaused || ticketItem.PauseReason == "" || ticketItem.PauseReason == ticketing.PauseReasonBudgetExhausted.String()) {
		update.SetRetryPaused(true).
			SetPauseReason(ticketing.PauseReasonBudgetExhausted.String())
	}

	if _, err := update.Save(ctx); err != nil {
		return PersistedUsageResult{}, mapTicketWriteError("update ticket usage", err)
	}

	if usageDelta.TotalTokens() > 0 {
		if _, err := tx.Agent.UpdateOneID(agentItem.ID).
			AddTotalTokensUsed(usageDelta.TotalTokens()).
			Save(ctx); err != nil {
			return PersistedUsageResult{}, fmt.Errorf("update agent usage counters: %w", err)
		}
	}

	var agentRunID string
	if input.RunID != nil {
		agentRunID = input.RunID.String()
		runItem, err := tx.AgentRun.Get(ctx, *input.RunID)
		if err != nil {
			if ent.IsNotFound(err) {
				return PersistedUsageResult{}, fmt.Errorf("agent run %s not found", *input.RunID)
			}
			return PersistedUsageResult{}, fmt.Errorf("get agent run for usage: %w", err)
		}
		if runItem.AgentID != agentItem.ID {
			return PersistedUsageResult{}, fmt.Errorf("agent run %s does not belong to agent %s", runItem.ID, agentItem.ID)
		}
		if runItem.TicketID != ticketItem.ID {
			return PersistedUsageResult{}, fmt.Errorf("agent run %s does not belong to ticket %s", runItem.ID, ticketItem.ID)
		}

		if _, err := tx.AgentRun.UpdateOneID(runItem.ID).
			AddInputTokens(usageDelta.InputTokens).
			AddOutputTokens(usageDelta.OutputTokens).
			AddCachedInputTokens(usageDelta.CachedInputTokens).
			AddCacheCreationInputTokens(usageDelta.CacheCreationInputTokens).
			AddReasoningTokens(usageDelta.ReasoningTokens).
			AddPromptTokens(usageDelta.PromptTokens).
			AddCandidateTokens(usageDelta.CandidateTokens).
			AddToolTokens(usageDelta.ToolTokens).
			AddTotalTokens(usageDelta.TotalTokens()).
			Save(ctx); err != nil {
			return PersistedUsageResult{}, fmt.Errorf("update agent run usage counters: %w", err)
		}
	}

	if _, err := tx.ActivityEvent.Create().
		SetProjectID(ticketItem.ProjectID).
		SetTicketID(ticketItem.ID).
		SetAgentID(agentItem.ID).
		SetEventType(ticketing.CostRecordedEventType).
		SetMessage("").
		SetMetadata(map[string]any{
			"input_tokens":  usageDelta.InputTokens,
			"output_tokens": usageDelta.OutputTokens,
			"total_tokens":  usageDelta.TotalTokens(),
			"cost_usd":      resolvedCost.AmountUSD,
			"cost_source":   resolvedCost.Source.String(),
			"agent_run_id":  agentRunID,
		}).
		SetCreatedAt(timeNowUTC()).
		Save(ctx); err != nil {
		return PersistedUsageResult{}, fmt.Errorf("create ticket cost event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return PersistedUsageResult{}, fmt.Errorf("commit ticket usage tx: %w", err)
	}

	ticketAfter, err := r.Get(ctx, ticketItem.ID)
	if err != nil {
		return PersistedUsageResult{}, err
	}

	return PersistedUsageResult{
		Result: RecordUsageResult{
			Ticket: ticketAfter,
			Applied: AppliedUsage{
				InputTokens:  usageDelta.InputTokens,
				OutputTokens: usageDelta.OutputTokens,
				CostUSD:      resolvedCost.AmountUSD,
				CostSource:   resolvedCost.Source.String(),
			},
			BudgetExceeded: ticketing.ShouldPauseForBudget(ticketAfter.CostAmount, ticketAfter.BudgetUSD),
		},
		MetricsAgent: UsageMetricsAgent{
			ProviderName: agentItem.Edges.Provider.Name,
			ModelName:    agentItem.Edges.Provider.ModelName,
		},
		ProjectID: ticketItem.ProjectID,
	}, nil
}

type pickupDiagnosisBuildContext struct {
	now                time.Time
	ticket             *ent.Ticket
	project            *ent.Project
	matchingWorkflows  []*ent.Workflow
	activeWorkflow     *ent.Workflow
	agent              *ent.Agent
	provider           *ent.AgentProvider
	machine            *ent.Machine
	providerState      *catalogdomain.AgentProvider
	blockedBy          []domain.PickupDiagnosisBlockedTicket
	workflowActiveRuns int
	projectActiveRuns  int
	providerActiveRuns int
	statusActiveRuns   int
}

func (r *EntRepository) GetPickupDiagnosis(ctx context.Context, ticketID uuid.UUID) (domain.PickupDiagnosis, error) {
	if r == nil || r.client == nil {
		return newPickupDiagnosis(
			domain.PickupDiagnosisStateUnavailable,
			domain.PickupDiagnosisReasonSchedulerUnavailable,
			"Scheduler state is unavailable.",
			"Retry once the ticket service is available again.",
			nil,
		), errUnavailable
	}

	ctxData, err := r.loadPickupDiagnosisContext(ctx, ticketID)
	if err != nil {
		return domain.PickupDiagnosis{}, err
	}

	return buildPickupDiagnosis(ctxData), nil
}

func (r *EntRepository) loadPickupDiagnosisContext(ctx context.Context, ticketID uuid.UUID) (pickupDiagnosisBuildContext, error) {
	now := timeNowUTC()
	ticketItem, err := r.client.Ticket.Query().
		Where(entticket.ID(ticketID)).
		WithStatus().
		Only(ctx)
	if err != nil {
		return pickupDiagnosisBuildContext{}, mapTicketReadError("get pickup diagnosis ticket", err)
	}

	projectItem, err := r.client.Project.Get(ctx, ticketItem.ProjectID)
	if err != nil {
		return pickupDiagnosisBuildContext{}, fmt.Errorf("get pickup diagnosis project: %w", err)
	}

	matchingWorkflows, err := r.client.Workflow.Query().
		Where(
			entworkflow.ProjectIDEQ(ticketItem.ProjectID),
			entworkflow.HasPickupStatusesWith(entticketstatus.IDEQ(ticketItem.StatusID)),
		).
		Order(ent.Asc(entworkflow.FieldName)).
		All(ctx)
	if err != nil {
		return pickupDiagnosisBuildContext{}, fmt.Errorf("list pickup diagnosis workflows: %w", err)
	}

	blockedBy, err := r.loadBlockedByTickets(ctx, ticketItem.ID)
	if err != nil {
		return pickupDiagnosisBuildContext{}, fmt.Errorf("list pickup diagnosis blockers: %w", err)
	}

	buildCtx := pickupDiagnosisBuildContext{
		now:               now,
		ticket:            ticketItem,
		project:           projectItem,
		matchingWorkflows: matchingWorkflows,
		blockedBy:         blockedBy,
	}
	buildCtx.activeWorkflow = firstActiveWorkflow(matchingWorkflows)
	if buildCtx.activeWorkflow == nil {
		return buildCtx, nil
	}

	agentItem, err := r.loadWorkflowDiagnosisAgent(ctx, buildCtx.activeWorkflow)
	if err != nil {
		return pickupDiagnosisBuildContext{}, err
	}
	buildCtx.agent = agentItem
	if agentItem == nil {
		return buildCtx, nil
	}

	providerItem, machineItem, providerState, err := r.loadWorkflowDiagnosisProvider(
		ctx,
		projectItem.OrganizationID,
		agentItem,
		now,
	)
	if err != nil {
		return pickupDiagnosisBuildContext{}, err
	}
	buildCtx.provider = providerItem
	buildCtx.machine = machineItem
	buildCtx.providerState = providerState

	buildCtx.workflowActiveRuns, buildCtx.projectActiveRuns, buildCtx.providerActiveRuns, buildCtx.statusActiveRuns, err = r.loadPickupDiagnosisActiveRuns(
		ctx,
		buildCtx.activeWorkflow,
		projectItem,
		providerItem,
		ticketItem.StatusID,
	)
	if err != nil {
		return pickupDiagnosisBuildContext{}, err
	}

	return buildCtx, nil
}

func (r *EntRepository) loadBlockedByTickets(ctx context.Context, ticketID uuid.UUID) ([]domain.PickupDiagnosisBlockedTicket, error) {
	dependencies, err := r.client.TicketDependency.Query().
		Where(
			entticketdependency.TargetTicketIDEQ(ticketID),
			entticketdependency.TypeEQ(entticketdependency.TypeBlocks),
		).
		WithSourceTicket(func(query *ent.TicketQuery) {
			query.WithStatus()
		}).
		Order(ent.Asc(entticketdependency.FieldSourceTicketID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	blockedBy := make([]domain.PickupDiagnosisBlockedTicket, 0, len(dependencies))
	for _, dependency := range dependencies {
		sourceTicket := dependency.Edges.SourceTicket
		if sourceTicket == nil || pickupDiagnosisDependencyResolved(sourceTicket) {
			continue
		}
		blockedBy = append(blockedBy, domain.PickupDiagnosisBlockedTicket{
			ID:         sourceTicket.ID,
			Identifier: sourceTicket.Identifier,
			Title:      sourceTicket.Title,
			StatusID:   sourceTicket.StatusID,
			StatusName: sourceTicket.Edges.Status.Name,
		})
	}

	return blockedBy, nil
}

func firstActiveWorkflow(workflows []*ent.Workflow) *ent.Workflow {
	for _, workflowItem := range workflows {
		if workflowItem.IsActive {
			return workflowItem
		}
	}
	return nil
}

func (r *EntRepository) loadWorkflowDiagnosisAgent(ctx context.Context, workflowItem *ent.Workflow) (*ent.Agent, error) {
	if workflowItem == nil || workflowItem.AgentID == nil {
		return nil, nil
	}

	agentItem, err := r.client.Agent.Query().
		Where(
			entagent.IDEQ(*workflowItem.AgentID),
			entagent.ProjectIDEQ(workflowItem.ProjectID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("get pickup diagnosis agent: %w", err)
	}

	return agentItem, nil
}

func (r *EntRepository) loadWorkflowDiagnosisProvider(
	ctx context.Context,
	organizationID uuid.UUID,
	agentItem *ent.Agent,
	now time.Time,
) (*ent.AgentProvider, *ent.Machine, *catalogdomain.AgentProvider, error) {
	if agentItem == nil {
		return nil, nil, nil, nil
	}

	providerItem, err := r.client.AgentProvider.Get(ctx, agentItem.ProviderID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("get pickup diagnosis provider: %w", err)
	}

	machineItem, err := r.client.Machine.Query().
		Where(
			entmachine.OrganizationIDEQ(organizationID),
			entmachine.IDEQ(providerItem.MachineID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return providerItem, nil, nil, nil
		}
		return nil, nil, nil, fmt.Errorf("get pickup diagnosis machine: %w", err)
	}

	state := catalogdomain.DeriveAgentProviderAvailability(catalogdomain.AgentProvider{
		ID:                   providerItem.ID,
		OrganizationID:       providerItem.OrganizationID,
		MachineID:            providerItem.MachineID,
		MachineName:          machineItem.Name,
		MachineHost:          machineItem.Host,
		MachineStatus:        catalogdomain.MachineStatus(machineItem.Status),
		MachineSSHUser:       optionalStringPointer(machineItem.SSHUser),
		MachineWorkspaceRoot: optionalStringPointer(machineItem.WorkspaceRoot),
		MachineAgentCLIPath:  optionalStringPointer(machineItem.AgentCliPath),
		MachineResources:     cloneAnyMap(machineItem.Resources),
		Name:                 providerItem.Name,
		AdapterType:          catalogdomain.AgentProviderAdapterType(providerItem.AdapterType),
		CliCommand:           providerItem.CliCommand,
		CliArgs:              append([]string(nil), providerItem.CliArgs...),
		AuthConfig:           cloneAnyMap(providerItem.AuthConfig),
		ModelName:            providerItem.ModelName,
		ModelTemperature:     providerItem.ModelTemperature,
		ModelMaxTokens:       providerItem.ModelMaxTokens,
		MaxParallelRuns:      providerItem.MaxParallelRuns,
		CostPerInputToken:    providerItem.CostPerInputToken,
		CostPerOutputToken:   providerItem.CostPerOutputToken,
	}, now)

	return providerItem, machineItem, &state, nil
}

func (r *EntRepository) loadPickupDiagnosisActiveRuns(
	ctx context.Context,
	workflowItem *ent.Workflow,
	projectItem *ent.Project,
	providerItem *ent.AgentProvider,
	statusID uuid.UUID,
) (int, int, int, int, error) {
	workflowActiveRuns := 0
	if workflowItem != nil {
		count, err := r.client.Ticket.Query().
			Where(
				entticket.WorkflowIDEQ(workflowItem.ID),
				entticket.CurrentRunIDNotNil(),
			).
			Count(ctx)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("count pickup diagnosis workflow active runs: %w", err)
		}
		workflowActiveRuns = count
	}

	projectActiveRuns := 0
	if projectItem != nil {
		count, err := r.client.Ticket.Query().
			Where(
				entticket.ProjectIDEQ(projectItem.ID),
				entticket.CurrentRunIDNotNil(),
			).
			Count(ctx)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("count pickup diagnosis project active runs: %w", err)
		}
		projectActiveRuns = count
	}

	providerActiveRuns := 0
	if providerItem != nil {
		count, err := r.client.AgentRun.Query().
			Where(
				entagentrun.ProviderIDEQ(providerItem.ID),
				entagentrun.HasCurrentForTicket(),
			).
			Count(ctx)
		if err != nil {
			return 0, 0, 0, 0, fmt.Errorf("count pickup diagnosis provider active runs: %w", err)
		}
		providerActiveRuns = count
	}

	statusActiveRuns, err := r.client.Ticket.Query().
		Where(
			entticket.StatusIDEQ(statusID),
			entticket.CurrentRunIDNotNil(),
		).
		Count(ctx)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("count pickup diagnosis status active runs: %w", err)
	}

	return workflowActiveRuns, projectActiveRuns, providerActiveRuns, statusActiveRuns, nil
}

func buildPickupDiagnosis(ctx pickupDiagnosisBuildContext) domain.PickupDiagnosis {
	diagnosis := newPickupDiagnosis(
		domain.PickupDiagnosisStateRunnable,
		domain.PickupDiagnosisReasonReadyForPickup,
		"Ticket is ready for pickup.",
		"Wait for the scheduler to claim the ticket.",
		nil,
	)
	diagnosis.Workflow = mapPickupDiagnosisWorkflow(ctx.activeWorkflow, ctx.matchingWorkflows)
	diagnosis.Agent = mapPickupDiagnosisAgent(ctx.agent)
	diagnosis.Provider = mapPickupDiagnosisProvider(ctx.provider, ctx.machine, ctx.providerState)
	diagnosis.Retry = domain.PickupDiagnosisRetry{
		AttemptCount: ctx.ticket.AttemptCount,
		RetryPaused:  ctx.ticket.RetryPaused,
		PauseReason:  ctx.ticket.PauseReason,
		NextRetryAt:  cloneTime(ctx.ticket.NextRetryAt),
	}
	diagnosis.Capacity = buildPickupDiagnosisCapacity(ctx)
	diagnosis.BlockedBy = append(diagnosis.BlockedBy, ctx.blockedBy...)

	if ctx.ticket.Archived {
		return newPickupDiagnosis(
			domain.PickupDiagnosisStateCompleted,
			domain.PickupDiagnosisReasonArchived,
			"Ticket is archived.",
			"Unarchive the ticket before pickup can resume.",
			diagnosisSummary(ctx, domain.PickupDiagnosisReasonArchived, domain.PickupDiagnosisReasonSeverityInfo, "Archived tickets are excluded from pickup."),
		)
	}

	statusStage := ticketing.StatusStage(ctx.ticket.Edges.Status.Stage)
	if ctx.ticket.CompletedAt != nil || (statusStage.IsValid() && statusStage.IsTerminal()) {
		return newPickupDiagnosis(
			domain.PickupDiagnosisStateCompleted,
			domain.PickupDiagnosisReasonCompleted,
			"Ticket is already in a terminal state.",
			"No pickup is needed.",
			diagnosisSummary(ctx, domain.PickupDiagnosisReasonCompleted, domain.PickupDiagnosisReasonSeverityInfo, "Ticket is already completed or otherwise terminal."),
		)
	}

	if ctx.ticket.CurrentRunID != nil {
		diagnosis.State = domain.PickupDiagnosisStateRunning
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRunningCurrentRun
		diagnosis.PrimaryReasonMessage = "Ticket already has an active run."
		diagnosis.NextActionHint = "Wait for the current run to finish or inspect the active runtime."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRunningCurrentRun, domain.PickupDiagnosisReasonSeverityInfo, "Current run is still attached to the ticket.")
		return diagnosis
	}

	if ctx.ticket.RetryPaused {
		diagnosis.State = domain.PickupDiagnosisStateBlocked
		switch ticketing.PauseReason(ctx.ticket.PauseReason) {
		case ticketing.PauseReasonRepeatedStalls:
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRetryPausedRepeatedStalls
			diagnosis.PrimaryReasonMessage = "Retries are paused after repeated stalls."
			diagnosis.NextActionHint = "Review the last failed attempt, then continue retry when ready."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRetryPausedRepeatedStalls, domain.PickupDiagnosisReasonSeverityWarning, "Manual retry is required after repeated stalls.")
		case ticketing.PauseReasonBudgetExhausted:
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRetryPausedBudget
			diagnosis.PrimaryReasonMessage = "Retries are paused because the ticket budget is exhausted."
			diagnosis.NextActionHint = "Increase the budget or reduce runtime cost before retrying."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRetryPausedBudget, domain.PickupDiagnosisReasonSeverityWarning, "Budget exhaustion paused further retries.")
		case ticketing.PauseReasonUserInterrupted:
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRetryPausedInterrupted
			diagnosis.PrimaryReasonMessage = "Retries are paused because the current run was interrupted."
			diagnosis.NextActionHint = "Resume retries when you want the agent to pick the ticket up again."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRetryPausedInterrupted, domain.PickupDiagnosisReasonSeverityWarning, "The last active run was interrupted by an operator.")
		default:
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRetryPausedUser
			diagnosis.PrimaryReasonMessage = "Retries are paused manually."
			diagnosis.NextActionHint = "Resume retries when you want the ticket to become schedulable again."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRetryPausedUser, domain.PickupDiagnosisReasonSeverityWarning, "Retries stay paused until they are resumed manually.")
		}
		return diagnosis
	}

	if ctx.ticket.NextRetryAt != nil && ctx.ticket.NextRetryAt.After(ctx.now) {
		nextRetry := ctx.ticket.NextRetryAt.UTC().Format(time.RFC3339)
		diagnosis.State = domain.PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonRetryBackoff
		diagnosis.PrimaryReasonMessage = "Waiting for retry backoff to expire."
		diagnosis.NextActionHint = "The ticket will become schedulable automatically after the retry window expires."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonRetryBackoff, domain.PickupDiagnosisReasonSeverityInfo, "Next retry is scheduled for "+nextRetry+".")
		return diagnosis
	}

	if len(ctx.blockedBy) > 0 {
		diagnosis.State = domain.PickupDiagnosisStateBlocked
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonBlockedDependency
		diagnosis.PrimaryReasonMessage = "Waiting for blocking tickets to finish."
		diagnosis.NextActionHint = "Resolve the blocking tickets or move them to a terminal status."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonBlockedDependency, domain.PickupDiagnosisReasonSeverityWarning, blockedDependencyMessage(ctx.blockedBy))
		return diagnosis
	}

	if len(ctx.matchingWorkflows) == 0 {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonNoMatchingActiveWorkflow
		diagnosis.PrimaryReasonMessage = "No workflow picks up the ticket's current status."
		diagnosis.NextActionHint = "Add an active workflow for this status or move the ticket into a pickup status."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonNoMatchingActiveWorkflow, domain.PickupDiagnosisReasonSeverityError, "No workflow in this project picks up status "+ctx.ticket.Edges.Status.Name+".")
		return diagnosis
	}

	if ctx.activeWorkflow == nil {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.Workflow = mapPickupDiagnosisWorkflow(ctx.matchingWorkflows[0], ctx.matchingWorkflows)
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonWorkflowInactive
		diagnosis.PrimaryReasonMessage = "Matching workflow is inactive."
		diagnosis.NextActionHint = "Reactivate the workflow or move the ticket into a status handled by an active workflow."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonWorkflowInactive, domain.PickupDiagnosisReasonSeverityError, "Workflow "+ctx.matchingWorkflows[0].Name+" matches the current status but is inactive.")
		return diagnosis
	}

	if ctx.activeWorkflow.AgentID == nil {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonWorkflowMissingAgent
		diagnosis.PrimaryReasonMessage = "Workflow has no bound agent."
		diagnosis.NextActionHint = "Bind an agent to the workflow before expecting pickup."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonWorkflowMissingAgent, domain.PickupDiagnosisReasonSeverityError, "Workflow "+ctx.activeWorkflow.Name+" does not have a bound agent.")
		return diagnosis
	}

	if ctx.agent == nil {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonAgentMissing
		diagnosis.PrimaryReasonMessage = "Workflow agent is missing."
		diagnosis.NextActionHint = "Rebind the workflow to an existing agent."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonAgentMissing, domain.PickupDiagnosisReasonSeverityError, "The workflow's bound agent record could not be found.")
		return diagnosis
	}

	switch ctx.agent.RuntimeControlState {
	case entagent.RuntimeControlStateInterruptRequested:
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonAgentInterruptRequested
		diagnosis.PrimaryReasonMessage = "Agent interrupt has been requested."
		diagnosis.NextActionHint = "Wait for the active runtime to stop before retrying or reassigning work."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonAgentInterruptRequested, domain.PickupDiagnosisReasonSeverityWarning, "Agent "+ctx.agent.Name+" is being interrupted.")
		return diagnosis
	case entagent.RuntimeControlStatePauseRequested:
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonAgentPauseRequested
		diagnosis.PrimaryReasonMessage = "Agent pause has been requested."
		diagnosis.NextActionHint = "Wait for the runtime to settle or resume the agent once it reaches paused."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonAgentPauseRequested, domain.PickupDiagnosisReasonSeverityWarning, "Agent "+ctx.agent.Name+" is transitioning toward a paused state.")
		return diagnosis
	case entagent.RuntimeControlStatePaused:
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonAgentPaused
		diagnosis.PrimaryReasonMessage = "Agent is paused."
		diagnosis.NextActionHint = "Resume the agent to allow pickup."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonAgentPaused, domain.PickupDiagnosisReasonSeverityWarning, "Agent "+ctx.agent.Name+" is paused.")
		return diagnosis
	}

	if ctx.provider == nil {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderMissing
		diagnosis.PrimaryReasonMessage = "Agent provider is missing."
		diagnosis.NextActionHint = "Reconnect the agent to a valid provider."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderMissing, domain.PickupDiagnosisReasonSeverityError, "The agent's provider record could not be found.")
		return diagnosis
	}

	if ctx.machine == nil {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonMachineMissing
		diagnosis.PrimaryReasonMessage = "Provider machine is missing."
		diagnosis.NextActionHint = "Reconnect the provider to an existing machine."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonMachineMissing, domain.PickupDiagnosisReasonSeverityError, "Provider "+ctx.provider.Name+" is bound to a machine that could not be found.")
		return diagnosis
	}

	if ctx.machine.Status != entmachine.StatusOnline {
		diagnosis.State = domain.PickupDiagnosisStateUnavailable
		if ctx.machine.Status == entmachine.StatusOffline {
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonMachineOffline
			diagnosis.PrimaryReasonMessage = "Provider machine is offline."
			diagnosis.NextActionHint = "Bring the machine back online before retrying pickup."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonMachineOffline, domain.PickupDiagnosisReasonSeverityError, "Machine "+ctx.machine.Name+" is offline.")
		} else {
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderUnavailable
			diagnosis.PrimaryReasonMessage = "Provider machine is not available."
			diagnosis.NextActionHint = "Recover the machine before expecting pickup."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderUnavailable, domain.PickupDiagnosisReasonSeverityError, "Machine "+ctx.machine.Name+" is "+strings.ToLower(string(ctx.machine.Status))+".")
		}
		return diagnosis
	}

	if ctx.providerState != nil {
		switch ctx.providerState.AvailabilityState {
		case catalogdomain.AgentProviderAvailabilityStateUnknown:
			diagnosis.State = domain.PickupDiagnosisStateUnavailable
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderUnknown
			diagnosis.PrimaryReasonMessage = "Provider availability is unknown."
			diagnosis.NextActionHint = "Refresh machine health so the provider can be probed again."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderUnknown, domain.PickupDiagnosisReasonSeverityWarning, providerAvailabilityMessage(ctx.providerState.AvailabilityReason))
			return diagnosis
		case catalogdomain.AgentProviderAvailabilityStateStale:
			diagnosis.State = domain.PickupDiagnosisStateUnavailable
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderStale
			diagnosis.PrimaryReasonMessage = "Provider health information is stale."
			diagnosis.NextActionHint = "Refresh provider health to confirm availability."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderStale, domain.PickupDiagnosisReasonSeverityWarning, providerAvailabilityMessage(ctx.providerState.AvailabilityReason))
			return diagnosis
		case catalogdomain.AgentProviderAvailabilityStateUnavailable:
			diagnosis.State = domain.PickupDiagnosisStateUnavailable
			diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderUnavailable
			diagnosis.PrimaryReasonMessage = "Provider is unavailable."
			diagnosis.NextActionHint = "Fix the provider health issue before expecting pickup."
			diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderUnavailable, domain.PickupDiagnosisReasonSeverityError, providerAvailabilityMessage(ctx.providerState.AvailabilityReason))
			return diagnosis
		}
	}

	if ctx.activeWorkflow.MaxConcurrent > 0 && ctx.workflowActiveRuns >= ctx.activeWorkflow.MaxConcurrent {
		diagnosis.State = domain.PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonWorkflowConcurrencyFull
		diagnosis.PrimaryReasonMessage = "Workflow concurrency is full."
		diagnosis.NextActionHint = "Wait for an active run in this workflow to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonWorkflowConcurrencyFull, domain.PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Workflow %s is using %d of %d allowed runs.", ctx.activeWorkflow.Name, ctx.workflowActiveRuns, ctx.activeWorkflow.MaxConcurrent))
		return diagnosis
	}

	if ctx.project.MaxConcurrentAgents > 0 && ctx.projectActiveRuns >= ctx.project.MaxConcurrentAgents {
		diagnosis.State = domain.PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProjectConcurrencyFull
		diagnosis.PrimaryReasonMessage = "Project concurrency is full."
		diagnosis.NextActionHint = "Wait for another active run in this project to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProjectConcurrencyFull, domain.PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Project is using %d of %d allowed runs.", ctx.projectActiveRuns, ctx.project.MaxConcurrentAgents))
		return diagnosis
	}

	if ctx.provider.MaxParallelRuns > 0 && ctx.providerActiveRuns >= ctx.provider.MaxParallelRuns {
		diagnosis.State = domain.PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonProviderConcurrencyFull
		diagnosis.PrimaryReasonMessage = "Provider concurrency is full."
		diagnosis.NextActionHint = "Wait for another run on this provider to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonProviderConcurrencyFull, domain.PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Provider %s is using %d of %d allowed runs.", ctx.provider.Name, ctx.providerActiveRuns, ctx.provider.MaxParallelRuns))
		return diagnosis
	}

	if ctx.ticket.Edges.Status.MaxActiveRuns != nil && ctx.statusActiveRuns >= *ctx.ticket.Edges.Status.MaxActiveRuns {
		diagnosis.State = domain.PickupDiagnosisStateWaiting
		diagnosis.PrimaryReasonCode = domain.PickupDiagnosisReasonStatusCapacityFull
		diagnosis.PrimaryReasonMessage = "Status capacity is full."
		diagnosis.NextActionHint = "Wait for another active run in this status to finish."
		diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonStatusCapacityFull, domain.PickupDiagnosisReasonSeverityInfo, fmt.Sprintf("Status %s is using %d of %d allowed runs.", ctx.ticket.Edges.Status.Name, ctx.statusActiveRuns, *ctx.ticket.Edges.Status.MaxActiveRuns))
		return diagnosis
	}

	diagnosis.Reasons = diagnosisSummary(ctx, domain.PickupDiagnosisReasonReadyForPickup, domain.PickupDiagnosisReasonSeverityInfo, "The scheduler can claim this ticket on the next tick.")
	return diagnosis
}

func newPickupDiagnosis(
	state domain.PickupDiagnosisState,
	code domain.PickupDiagnosisReasonCode,
	message string,
	hint string,
	reasons []domain.PickupDiagnosisReason,
) domain.PickupDiagnosis {
	return domain.PickupDiagnosis{
		State:                state,
		PrimaryReasonCode:    code,
		PrimaryReasonMessage: message,
		NextActionHint:       hint,
		Reasons:              append([]domain.PickupDiagnosisReason{}, reasons...),
		BlockedBy:            []domain.PickupDiagnosisBlockedTicket{},
	}
}

func diagnosisSummary(
	ctx pickupDiagnosisBuildContext,
	code domain.PickupDiagnosisReasonCode,
	severity domain.PickupDiagnosisReasonSeverity,
	message string,
) []domain.PickupDiagnosisReason {
	reasons := []domain.PickupDiagnosisReason{{
		Code:     code,
		Message:  message,
		Severity: severity,
	}}

	if len(ctx.blockedBy) > 1 && code == domain.PickupDiagnosisReasonBlockedDependency {
		for _, blocker := range ctx.blockedBy {
			reasons = append(reasons, domain.PickupDiagnosisReason{
				Code:     domain.PickupDiagnosisReasonBlockedDependency,
				Message:  blocker.Identifier + " " + blocker.Title,
				Severity: domain.PickupDiagnosisReasonSeverityWarning,
			})
		}
	}

	return reasons
}

func blockedDependencyMessage(blockers []domain.PickupDiagnosisBlockedTicket) string {
	if len(blockers) == 0 {
		return "Blocking dependency is unresolved."
	}

	parts := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		parts = append(parts, blocker.Identifier+" "+blocker.Title)
	}
	return "Blocked by " + strings.Join(parts, ", ") + "."
}

func mapPickupDiagnosisWorkflow(activeWorkflow *ent.Workflow, matchingWorkflows []*ent.Workflow) *domain.PickupDiagnosisWorkflow {
	if activeWorkflow != nil {
		return &domain.PickupDiagnosisWorkflow{
			ID:                activeWorkflow.ID,
			Name:              activeWorkflow.Name,
			IsActive:          activeWorkflow.IsActive,
			PickupStatusMatch: true,
		}
	}
	if len(matchingWorkflows) == 0 {
		return nil
	}
	workflowItem := matchingWorkflows[0]
	return &domain.PickupDiagnosisWorkflow{
		ID:                workflowItem.ID,
		Name:              workflowItem.Name,
		IsActive:          workflowItem.IsActive,
		PickupStatusMatch: true,
	}
}

func mapPickupDiagnosisAgent(agentItem *ent.Agent) *domain.PickupDiagnosisAgent {
	if agentItem == nil {
		return nil
	}
	return &domain.PickupDiagnosisAgent{
		ID:                  agentItem.ID,
		Name:                agentItem.Name,
		RuntimeControlState: catalogdomain.AgentRuntimeControlState(agentItem.RuntimeControlState),
	}
}

func mapPickupDiagnosisProvider(
	providerItem *ent.AgentProvider,
	machineItem *ent.Machine,
	providerState *catalogdomain.AgentProvider,
) *domain.PickupDiagnosisProvider {
	if providerItem == nil {
		return nil
	}

	response := &domain.PickupDiagnosisProvider{
		ID:   providerItem.ID,
		Name: providerItem.Name,
	}
	if machineItem != nil {
		response.MachineID = machineItem.ID
		response.MachineName = machineItem.Name
		response.MachineStatus = catalogdomain.MachineStatus(machineItem.Status)
	}
	if providerState != nil {
		response.AvailabilityState = providerState.AvailabilityState
		response.AvailabilityReason = cloneString(providerState.AvailabilityReason)
	}
	return response
}

func buildPickupDiagnosisCapacity(ctx pickupDiagnosisBuildContext) domain.PickupDiagnosisCapacity {
	capacity := domain.PickupDiagnosisCapacity{}
	if ctx.activeWorkflow != nil {
		capacity.Workflow = domain.PickupDiagnosisCapacityBucket{
			Limited:    ctx.activeWorkflow.MaxConcurrent > 0,
			ActiveRuns: ctx.workflowActiveRuns,
			Capacity:   ctx.activeWorkflow.MaxConcurrent,
		}
	}
	if ctx.project != nil {
		capacity.Project = domain.PickupDiagnosisCapacityBucket{
			Limited:    ctx.project.MaxConcurrentAgents > 0,
			ActiveRuns: ctx.projectActiveRuns,
			Capacity:   ctx.project.MaxConcurrentAgents,
		}
	}
	if ctx.provider != nil {
		capacity.Provider = domain.PickupDiagnosisCapacityBucket{
			Limited:    ctx.provider.MaxParallelRuns > 0,
			ActiveRuns: ctx.providerActiveRuns,
			Capacity:   ctx.provider.MaxParallelRuns,
		}
	}
	if ctx.ticket != nil && ctx.ticket.Edges.Status != nil {
		capacity.Status = domain.PickupDiagnosisStatusCapacity{
			Limited:    ctx.ticket.Edges.Status.MaxActiveRuns != nil,
			ActiveRuns: ctx.statusActiveRuns,
			Capacity:   cloneInt(ctx.ticket.Edges.Status.MaxActiveRuns),
		}
	}
	return capacity
}

func providerAvailabilityMessage(reason *string) string {
	switch strings.TrimSpace(stringValue(reason)) {
	case "machine_offline":
		return "Provider machine is offline."
	case "machine_degraded":
		return "Provider machine is degraded."
	case "machine_maintenance":
		return "Provider machine is in maintenance mode."
	case "l4_snapshot_missing":
		return "Provider health has not been probed yet."
	case "stale_l4_snapshot":
		return "Provider health snapshot is stale."
	case "cli_missing":
		return "Provider CLI is missing on the machine."
	case "not_logged_in":
		return "Provider CLI is not authenticated."
	case "not_ready":
		return "Provider CLI is not ready."
	case "config_incomplete":
		return "Provider launch configuration is incomplete."
	case "unsupported_adapter":
		return "Provider adapter is not supported by health checks."
	default:
		return "Provider health is blocking pickup."
	}
}

func pickupDiagnosisDependencyResolved(ticketItem *ent.Ticket) bool {
	if ticketItem == nil {
		return false
	}
	if ticketItem.CompletedAt != nil {
		return true
	}
	if ticketItem.Edges.Status == nil {
		return false
	}

	stage := ticketing.StatusStage(ticketItem.Edges.Status.Stage)
	return stage.IsValid() && stage.IsTerminal()
}

func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := strings.TrimSpace(*value)
	return &copied
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func cloneOptionalText(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}

func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}
