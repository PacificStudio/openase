package ticket

import (
	"context"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entprojectrepo "github.com/BetterAndBetterII/openase/ent/projectrepo"
	entticketreposcope "github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

// Create persists a new ticket and applies project defaults.
func (r *CommandRepository) Create(ctx context.Context, input CreateInput) (Ticket, error) {
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

	return NewQueryRepository(r.client).Get(ctx, created.ID)
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
func (r *CommandRepository) Update(ctx context.Context, input UpdateInput) (UpdateResult, error) {
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

	updated, err := NewQueryRepository(r.client).Get(ctx, current.ID)
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
func (r *CommandRepository) ResumeRetry(ctx context.Context, input ResumeRetryInput) (Ticket, error) {
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

	return NewQueryRepository(r.client).Get(ctx, current.ID)
}
