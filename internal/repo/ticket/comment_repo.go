package ticket

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entticketcomment "github.com/BetterAndBetterII/openase/ent/ticketcomment"
	entticketcommentrevision "github.com/BetterAndBetterII/openase/ent/ticketcommentrevision"
	"github.com/google/uuid"
)

func (r *CommentRepository) ListComments(ctx context.Context, ticketID uuid.UUID) ([]Comment, error) {
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
func (r *CommentRepository) ListCommentRevisions(ctx context.Context, ticketID uuid.UUID, commentID uuid.UUID) ([]CommentRevision, error) {
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
func (r *CommentRepository) AddComment(ctx context.Context, input AddCommentInput) (Comment, error) {
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
func (r *CommentRepository) UpdateComment(ctx context.Context, input UpdateCommentInput) (Comment, error) {
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

// RemoveComment deletes a user discussion comment from a ticket.
func (r *CommentRepository) RemoveComment(ctx context.Context, ticketID uuid.UUID, commentID uuid.UUID) (DeleteCommentResult, error) {
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
