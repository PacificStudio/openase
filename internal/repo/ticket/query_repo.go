package ticket

import (
	"context"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketexternallink "github.com/BetterAndBetterII/openase/ent/ticketexternallink"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	"github.com/google/uuid"
)

// List returns tickets in a project ordered for UI consumption.
func (r *QueryRepository) List(ctx context.Context, input ListInput) ([]Ticket, error) {
	if r.client == nil {
		return nil, errUnavailable
	}
	if err := ensureProjectExists(ctx, r.client, input.ProjectID); err != nil {
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
		}).
		WithRepoScopes()

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

func (r *QueryRepository) ListArchived(ctx context.Context, input ArchivedListInput) (ArchivedListResult, error) {
	if r.client == nil {
		return ArchivedListResult{}, errUnavailable
	}
	if err := ensureProjectExists(ctx, r.client, input.ProjectID); err != nil {
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

func (r *QueryRepository) Get(ctx context.Context, ticketID uuid.UUID) (Ticket, error) {
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
