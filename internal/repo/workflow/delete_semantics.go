package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entscheduledjob "github.com/BetterAndBetterII/openase/ent/scheduledjob"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/google/uuid"
)

func (r *EntRepository) ImpactAnalysis(ctx context.Context, workflowID uuid.UUID) (domain.WorkflowImpactAnalysis, error) {
	workflowItem, err := r.Get(ctx, workflowID)
	if err != nil {
		return domain.WorkflowImpactAnalysis{}, err
	}

	tickets, err := r.listReplaceableWorkflowTickets(ctx, workflowID)
	if err != nil {
		return domain.WorkflowImpactAnalysis{}, err
	}
	scheduledJobs, err := r.listWorkflowScheduledJobs(ctx, workflowID)
	if err != nil {
		return domain.WorkflowImpactAnalysis{}, err
	}
	activeRuns, err := r.listWorkflowAgentRuns(ctx, workflowID, true)
	if err != nil {
		return domain.WorkflowImpactAnalysis{}, err
	}
	historicalRuns, err := r.listWorkflowAgentRuns(ctx, workflowID, false)
	if err != nil {
		return domain.WorkflowImpactAnalysis{}, err
	}

	summary := domain.WorkflowImpactSummary{
		TicketCount:               len(tickets),
		ScheduledJobCount:         len(scheduledJobs),
		ActiveAgentRunCount:       len(activeRuns),
		HistoricalAgentRunCount:   len(historicalRuns),
		ReplaceableReferenceCount: len(tickets) + len(scheduledJobs),
		BlockingReferenceCount:    len(activeRuns) + len(historicalRuns),
	}

	return domain.WorkflowImpactAnalysis{
		WorkflowID:           workflowItem.ID,
		CanRetire:            workflowItem.IsActive,
		CanReplaceReferences: summary.ReplaceableReferenceCount > 0,
		CanPurge:             summary.ReplaceableReferenceCount == 0 && summary.BlockingReferenceCount == 0,
		Summary:              summary,
		ReplaceableReferences: domain.WorkflowReplaceableReferences{
			Tickets:       tickets,
			ScheduledJobs: scheduledJobs,
		},
		BlockingReferences: domain.WorkflowBlockingReferences{
			ActiveAgentRuns:     activeRuns,
			HistoricalAgentRuns: historicalRuns,
		},
	}, nil
}

func (r *EntRepository) ReplaceReferences(ctx context.Context, input domain.ReplaceWorkflowReferencesInput) (domain.ReplaceWorkflowReferencesResult, error) {
	if input.WorkflowID == uuid.Nil {
		return domain.ReplaceWorkflowReferencesResult{}, fmt.Errorf("%w: workflow id must not be empty", domain.ErrWorkflowReplacementInvalid)
	}
	if input.ReplacementWorkflowID == uuid.Nil {
		return domain.ReplaceWorkflowReferencesResult{}, fmt.Errorf("%w: replacement workflow id must not be empty", domain.ErrWorkflowReplacementInvalid)
	}
	if input.WorkflowID == input.ReplacementWorkflowID {
		return domain.ReplaceWorkflowReferencesResult{}, fmt.Errorf("%w: replacement workflow must differ from the source workflow", domain.ErrWorkflowReplacementInvalid)
	}

	source, err := r.Get(ctx, input.WorkflowID)
	if err != nil {
		return domain.ReplaceWorkflowReferencesResult{}, err
	}
	replacement, err := r.Get(ctx, input.ReplacementWorkflowID)
	if err != nil {
		if ent.IsNotFound(err) || errors.Is(err, domain.ErrWorkflowNotFound) {
			return domain.ReplaceWorkflowReferencesResult{}, fmt.Errorf("%w: %s", domain.ErrWorkflowReplacementNotFound, input.ReplacementWorkflowID)
		}
		return domain.ReplaceWorkflowReferencesResult{}, err
	}
	if replacement.ProjectID != source.ProjectID {
		return domain.ReplaceWorkflowReferencesResult{}, fmt.Errorf("%w: workflow %s belongs to project %s, expected %s", domain.ErrWorkflowReplacementProjectMismatch, replacement.ID, replacement.ProjectID, source.ProjectID)
	}
	if !replacement.IsActive {
		return domain.ReplaceWorkflowReferencesResult{}, fmt.Errorf("%w: workflow %s is inactive", domain.ErrWorkflowReplacementInactive, replacement.ID)
	}

	tickets, err := r.listReplaceableWorkflowTickets(ctx, source.ID)
	if err != nil {
		return domain.ReplaceWorkflowReferencesResult{}, err
	}
	scheduledJobs, err := r.listWorkflowScheduledJobs(ctx, source.ID)
	if err != nil {
		return domain.ReplaceWorkflowReferencesResult{}, err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.ReplaceWorkflowReferencesResult{}, fmt.Errorf("start workflow replace references tx: %w", err)
	}
	defer rollback(tx)

	if len(tickets) > 0 {
		ticketIDs := make([]uuid.UUID, 0, len(tickets))
		for _, ticketRef := range tickets {
			ticketIDs = append(ticketIDs, ticketRef.ID)
		}
		if _, err := tx.Ticket.Update().
			Where(entticket.IDIn(ticketIDs...)).
			SetWorkflowID(replacement.ID).
			Save(ctx); err != nil {
			return domain.ReplaceWorkflowReferencesResult{}, fmt.Errorf("replace workflow ticket references: %w", err)
		}
	}

	if len(scheduledJobs) > 0 {
		jobIDs := make([]uuid.UUID, 0, len(scheduledJobs))
		for _, jobRef := range scheduledJobs {
			jobIDs = append(jobIDs, jobRef.ID)
		}
		if _, err := tx.ScheduledJob.Update().
			Where(entscheduledjob.IDIn(jobIDs...)).
			SetWorkflowID(replacement.ID).
			Save(ctx); err != nil {
			return domain.ReplaceWorkflowReferencesResult{}, fmt.Errorf("replace workflow scheduled job references: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.ReplaceWorkflowReferencesResult{}, fmt.Errorf("commit workflow replace references tx: %w", err)
	}

	return domain.ReplaceWorkflowReferencesResult{
		WorkflowID:            source.ID,
		ReplacementWorkflowID: replacement.ID,
		TicketCount:           len(tickets),
		ScheduledJobCount:     len(scheduledJobs),
		Tickets:               tickets,
		ScheduledJobs:         scheduledJobs,
	}, nil
}

func (r *EntRepository) listReplaceableWorkflowTickets(ctx context.Context, workflowID uuid.UUID) ([]domain.WorkflowTicketReference, error) {
	items, err := r.client.Ticket.Query().
		Where(
			entticket.WorkflowIDEQ(workflowID),
			entticket.Archived(false),
			entticket.HasStatusWith(
				entticketstatus.StageNotIn(entticketstatus.StageCompleted, entticketstatus.StageCanceled),
			),
		).
		WithStatus().
		Order(ent.Asc(entticket.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflow tickets: %w", err)
	}

	result := make([]domain.WorkflowTicketReference, 0, len(items))
	for _, item := range items {
		result = append(result, mapWorkflowTicketReference(item))
	}
	return result, nil
}

func (r *EntRepository) listWorkflowScheduledJobs(ctx context.Context, workflowID uuid.UUID) ([]domain.WorkflowScheduledJobReference, error) {
	items, err := r.client.ScheduledJob.Query().
		Where(entscheduledjob.WorkflowIDEQ(workflowID)).
		Order(ent.Asc(entscheduledjob.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflow scheduled jobs: %w", err)
	}

	result := make([]domain.WorkflowScheduledJobReference, 0, len(items))
	for _, item := range items {
		result = append(result, domain.WorkflowScheduledJobReference{
			ID:        item.ID,
			Name:      item.Name,
			IsEnabled: item.IsEnabled,
		})
	}
	return result, nil
}

func (r *EntRepository) listWorkflowAgentRuns(ctx context.Context, workflowID uuid.UUID, active bool) ([]domain.WorkflowAgentRunReference, error) {
	query := r.client.AgentRun.Query().
		Where(entagentrun.WorkflowIDEQ(workflowID)).
		WithTicket()
	if active {
		query = query.Where(entagentrun.TerminalAtIsNil())
	} else {
		query = query.Where(entagentrun.TerminalAtNotNil())
	}
	items, err := query.Order(ent.Desc(entagentrun.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list workflow agent runs: %w", err)
	}

	result := make([]domain.WorkflowAgentRunReference, 0, len(items))
	for _, item := range items {
		result = append(result, mapWorkflowAgentRunReference(item))
	}
	return result, nil
}

func mapWorkflowTicketReference(item *ent.Ticket) domain.WorkflowTicketReference {
	ref := domain.WorkflowTicketReference{
		ID:         item.ID,
		Identifier: item.Identifier,
		Title:      item.Title,
		StatusID:   item.StatusID,
	}
	if item.CurrentRunID != nil {
		cloned := *item.CurrentRunID
		ref.CurrentRunID = &cloned
	}
	if item.Edges.Status != nil {
		ref.StatusName = item.Edges.Status.Name
	}
	return ref
}

func mapWorkflowAgentRunReference(item *ent.AgentRun) domain.WorkflowAgentRunReference {
	ref := domain.WorkflowAgentRunReference{
		ID:        item.ID,
		TicketID:  item.TicketID,
		Status:    item.Status.String(),
		CreatedAt: item.CreatedAt.UTC(),
	}
	if item.Edges.Ticket != nil {
		ref.TicketIdentifier = item.Edges.Ticket.Identifier
		ref.TicketTitle = item.Edges.Ticket.Title
	}
	return ref
}
