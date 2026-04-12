package ticket

import (
	"context"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketexternallink "github.com/BetterAndBetterII/openase/ent/ticketexternallink"
	"github.com/google/uuid"
)

// AddDependency creates a dependency edge between two tickets.
func (r *LinkRepository) AddDependency(ctx context.Context, input AddDependencyInput) (Dependency, error) {
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
func (r *LinkRepository) RemoveDependency(ctx context.Context, ticketID uuid.UUID, dependencyID uuid.UUID) (DeleteDependencyResult, error) {
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
func (r *LinkRepository) AddExternalLink(ctx context.Context, input AddExternalLinkInput) (ExternalLink, error) {
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
		SetURL(input.URL).
		SetExternalID(input.ExternalID)
	if input.LinkType != "" {
		builder.SetLinkType(toEntExternalLinkType(input.LinkType))
	}
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

func (r *LinkRepository) RemoveExternalLink(ctx context.Context, ticketID uuid.UUID, externalLinkID uuid.UUID) (DeleteExternalLinkResult, error) {
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
