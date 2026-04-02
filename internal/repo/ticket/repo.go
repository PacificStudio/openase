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
		Where(entticket.ProjectIDEQ(input.ProjectID)).
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
		SetPriority(toEntTicketPriority(input.Priority)).
		SetType(toEntTicketType(input.Type)).
		SetCreatedBy(resolveCreatedBy(input.CreatedBy)).
		SetBudgetUsd(input.BudgetUSD).
		SetRetryToken(NewRetryToken())

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
	targetMachineChanged := false
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
			hookName, err := releasedRunHookForStatusChange(ctx, tx, current.WorkflowID, input.StatusID.Value)
			if err != nil {
				return UpdateResult{}, err
			}
			releasedHookName = hookName
		}
		builder.SetStatusID(input.StatusID.Value)
	}
	if input.Priority.Set {
		builder.SetPriority(toEntTicketPriority(input.Priority.Value))
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
	if statusChanged {
		builder.ClearCurrentRunID()
	}
	if statusChanged || targetMachineChanged {
		ResetRetryBaseline(builder, current)
	}

	if _, err := builder.Save(ctx); err != nil {
		return UpdateResult{}, mapTicketWriteError("update ticket", err)
	}
	if statusChanged || targetMachineChanged {
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

func releasedRunHookForStatusChange(
	ctx context.Context,
	tx *ent.Tx,
	workflowID *uuid.UUID,
	statusID uuid.UUID,
) (string, error) {
	if workflowID != nil {
		allowed, err := isWorkflowFinishStatus(ctx, tx, *workflowID, statusID)
		if err != nil {
			return "", err
		}
		if allowed {
			return "on_done", nil
		}
	}

	return "on_cancel", nil
}

func isWorkflowFinishStatus(ctx context.Context, tx *ent.Tx, workflowID uuid.UUID, statusID uuid.UUID) (bool, error) {
	workflowItem, err := tx.Workflow.Query().
		Where(entworkflow.IDEQ(workflowID)).
		WithFinishStatuses().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, ErrWorkflowNotFound
		}
		return false, fmt.Errorf("load workflow finish statuses: %w", err)
	}

	for _, finishStatus := range workflowItem.Edges.FinishStatuses {
		if finishStatus.ID == statusID {
			return true, nil
		}
	}
	return false, nil
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

	return LifecycleHookRuntimeData{
		TicketID:         ticketItem.ID,
		ProjectID:        ticketItem.ProjectID,
		AgentID:          runItem.AgentID,
		TicketIdentifier: ticketItem.Identifier,
		AgentName:        runItem.Edges.Agent.Name,
		WorkflowType:     string(workflowItem.Type),
		Attempt:          ticketItem.AttemptCount + 1,
		WorkspaceRoot:    workspaceRoot,
		Hooks:            cloneAnyMap(workflowItem.Hooks),
		Machine:          mapTicketHookMachine(machineItem),
		Workspaces:       repos,
	}, nil
}

func mapTicketHookMachine(item *ent.Machine) catalogdomain.Machine {
	if item == nil {
		return catalogdomain.Machine{}
	}

	return catalogdomain.Machine{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		Name:           item.Name,
		Host:           item.Host,
		Port:           item.Port,
		SSHUser:        cloneOptionalText(item.SSHUser),
		SSHKeyPath:     cloneOptionalText(item.SSHKeyPath),
		Status:         catalogdomain.MachineStatus(item.Status),
		WorkspaceRoot:  cloneOptionalText(item.WorkspaceRoot),
		AgentCLIPath:   cloneOptionalText(item.AgentCliPath),
		EnvVars:        slices.Clone(item.EnvVars),
		Resources:      cloneMap(item.Resources),
	}
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

func cloneOptionalText(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
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
