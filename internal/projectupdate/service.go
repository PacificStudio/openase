package projectupdate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	entdb "github.com/BetterAndBetterII/openase/ent"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entprojectupdatecomment "github.com/BetterAndBetterII/openase/ent/projectupdatecomment"
	entprojectupdatecommentrevision "github.com/BetterAndBetterII/openase/ent/projectupdatecommentrevision"
	entprojectupdatethread "github.com/BetterAndBetterII/openase/ent/projectupdatethread"
	entprojectupdatethreadrevision "github.com/BetterAndBetterII/openase/ent/projectupdatethreadrevision"
	"github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	projectupdatedomain "github.com/BetterAndBetterII/openase/internal/domain/projectupdate"
	"github.com/google/uuid"
)

const defaultCreatedBy = "user:api"

var (
	ErrUnavailable     = errors.New("project update service unavailable")
	ErrProjectNotFound = errors.New("project not found")
	ErrThreadNotFound  = errors.New("project update thread not found")
	ErrCommentNotFound = errors.New("project update comment not found")
)

type Status string

const (
	StatusOnTrack  Status = "on_track"
	StatusAtRisk   Status = "at_risk"
	StatusOffTrack Status = "off_track"
)

type Thread struct {
	ID             uuid.UUID  `json:"id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	Status         Status     `json:"status"`
	Title          string     `json:"title"`
	BodyMarkdown   string     `json:"body_markdown"`
	CreatedBy      string     `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
	EditCount      int        `json:"edit_count"`
	LastEditedBy   *string    `json:"last_edited_by,omitempty"`
	IsDeleted      bool       `json:"is_deleted"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	DeletedBy      *string    `json:"deleted_by,omitempty"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	CommentCount   int        `json:"comment_count"`
	Comments       []Comment  `json:"comments"`
}

type ThreadRevision struct {
	ID             uuid.UUID `json:"id"`
	ThreadID       uuid.UUID `json:"thread_id"`
	RevisionNumber int       `json:"revision_number"`
	Status         Status    `json:"status"`
	Title          string    `json:"title"`
	BodyMarkdown   string    `json:"body_markdown"`
	EditedBy       string    `json:"edited_by"`
	EditedAt       time.Time `json:"edited_at"`
	EditReason     *string   `json:"edit_reason,omitempty"`
}

type Comment struct {
	ID           uuid.UUID  `json:"id"`
	ThreadID     uuid.UUID  `json:"thread_id"`
	BodyMarkdown string     `json:"body_markdown"`
	CreatedBy    string     `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	EditedAt     *time.Time `json:"edited_at,omitempty"`
	EditCount    int        `json:"edit_count"`
	LastEditedBy *string    `json:"last_edited_by,omitempty"`
	IsDeleted    bool       `json:"is_deleted"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	DeletedBy    *string    `json:"deleted_by,omitempty"`
}

type CommentRevision struct {
	ID             uuid.UUID `json:"id"`
	CommentID      uuid.UUID `json:"comment_id"`
	RevisionNumber int       `json:"revision_number"`
	BodyMarkdown   string    `json:"body_markdown"`
	EditedBy       string    `json:"edited_by"`
	EditedAt       time.Time `json:"edited_at"`
	EditReason     *string   `json:"edit_reason,omitempty"`
}

type AddThreadInput struct {
	ProjectID uuid.UUID
	Status    Status
	Title     string
	Body      string
	CreatedBy string
}

type UpdateThreadInput struct {
	ProjectID  uuid.UUID
	ThreadID   uuid.UUID
	Status     Status
	Title      string
	Body       string
	EditedBy   string
	EditReason string
}

type DeleteThreadResult struct {
	DeletedThreadID uuid.UUID `json:"deleted_thread_id"`
}

type AddCommentInput struct {
	ProjectID uuid.UUID
	ThreadID  uuid.UUID
	Body      string
	CreatedBy string
}

type UpdateCommentInput struct {
	ProjectID  uuid.UUID
	ThreadID   uuid.UUID
	CommentID  uuid.UUID
	Body       string
	EditedBy   string
	EditReason string
}

type DeleteCommentResult struct {
	DeletedCommentID uuid.UUID `json:"deleted_comment_id"`
}

type ThreadPage struct {
	Threads    []Thread `json:"threads"`
	NextCursor string   `json:"next_cursor,omitempty"`
	HasMore    bool     `json:"has_more"`
}

type Service struct {
	client          *entdb.Client
	activityEmitter *activity.Emitter
}

func NewService(client *entdb.Client, activityEmitter *activity.Emitter) *Service {
	return &Service{client: client, activityEmitter: activityEmitter}
}

func (s *Service) ListThreads(ctx context.Context, projectID uuid.UUID) ([]Thread, error) {
	page, err := s.listThreadPage(ctx, projectID, nil, 0)
	if err != nil {
		return nil, err
	}
	return page.Threads, nil
}

func (s *Service) ListThreadPage(
	ctx context.Context,
	input projectupdatedomain.ListThreadsPage,
) (ThreadPage, error) {
	return s.listThreadPage(ctx, input.ProjectID, input.Before, input.Limit)
}

func (s *Service) listThreadPage(
	ctx context.Context,
	projectID uuid.UUID,
	before *projectupdatedomain.ThreadCursor,
	limit int,
) (ThreadPage, error) {
	if s.client == nil {
		return ThreadPage{}, ErrUnavailable
	}
	if err := s.ensureProjectExists(ctx, projectID); err != nil {
		return ThreadPage{}, err
	}

	query := s.client.ProjectUpdateThread.Query().
		Where(entprojectupdatethread.ProjectIDEQ(projectID)).
		Order(entdb.Desc(entprojectupdatethread.FieldLastActivityAt), entdb.Desc(entprojectupdatethread.FieldID)).
		WithComments(func(query *entdb.ProjectUpdateCommentQuery) {
			query.Order(entdb.Asc(entprojectupdatecomment.FieldCreatedAt), entdb.Asc(entprojectupdatecomment.FieldID))
		})
	if before != nil {
		query.Where(
			entprojectupdatethread.Or(
				entprojectupdatethread.LastActivityAtLT(before.LastActivityAt),
				entprojectupdatethread.And(
					entprojectupdatethread.LastActivityAtEQ(before.LastActivityAt),
					entprojectupdatethread.IDLT(before.ID),
				),
			),
		)
	}
	if limit > 0 {
		query.Limit(limit + 1)
	}

	items, err := query.All(ctx)
	if err != nil {
		return ThreadPage{}, fmt.Errorf("list project update threads: %w", err)
	}

	hasMore := false
	if limit > 0 && len(items) > limit {
		hasMore = true
		items = items[:limit]
	}

	threads := make([]Thread, 0, len(items))
	for _, item := range items {
		threads = append(threads, mapThread(item))
	}

	nextCursor := ""
	if hasMore && len(items) > 0 {
		nextCursor = threadCursorForEntity(items[len(items)-1]).String()
	}

	return ThreadPage{
		Threads:    threads,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *Service) ListThreadRevisions(ctx context.Context, projectID, threadID uuid.UUID) ([]ThreadRevision, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}

	thread, err := s.getThread(ctx, projectID, threadID)
	if err != nil {
		return nil, err
	}

	items, err := s.client.ProjectUpdateThreadRevision.Query().
		Where(entprojectupdatethreadrevision.ThreadIDEQ(thread.ID)).
		Order(entdb.Asc(entprojectupdatethreadrevision.FieldRevisionNumber), entdb.Asc(entprojectupdatethreadrevision.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project update thread revisions: %w", err)
	}
	if len(items) == 0 {
		return []ThreadRevision{s.syntheticInitialThreadRevision(thread)}, nil
	}

	revisions := make([]ThreadRevision, 0, len(items))
	for _, item := range items {
		revisions = append(revisions, mapThreadRevision(item))
	}
	return revisions, nil
}

func (s *Service) AddThread(ctx context.Context, input AddThreadInput) (Thread, error) {
	if s.client == nil {
		return Thread{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Thread{}, fmt.Errorf("start add project update thread tx: %w", err)
	}
	defer rollback(tx)

	if err := s.ensureProjectExistsTx(ctx, tx, input.ProjectID); err != nil {
		return Thread{}, err
	}

	now := time.Now().UTC()
	createdBy := resolveCreatedBy(input.CreatedBy)
	body := strings.TrimSpace(input.Body)
	title := resolveThreadTitle(input.Title, body)
	item, err := tx.ProjectUpdateThread.Create().
		SetProjectID(input.ProjectID).
		SetStatus(mapStatus(input.Status)).
		SetTitle(title).
		SetBodyMarkdown(body).
		SetCreatedBy(createdBy).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetLastActivityAt(now).
		Save(ctx)
	if err != nil {
		return Thread{}, fmt.Errorf("create project update thread: %w", err)
	}
	if err := s.appendThreadRevisionTx(ctx, tx, item.ID, 1, item.Status, item.Title, item.BodyMarkdown, createdBy, now, ""); err != nil {
		return Thread{}, err
	}
	if err := tx.Commit(); err != nil {
		return Thread{}, fmt.Errorf("commit add project update thread tx: %w", err)
	}
	if err := s.emitThreadActivity(ctx, activityevent.TypeProjectUpdateThreadCreated, item, nil); err != nil {
		return Thread{}, err
	}

	return mapThread(item), nil
}

func (s *Service) UpdateThread(ctx context.Context, input UpdateThreadInput) (Thread, error) {
	if s.client == nil {
		return Thread{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Thread{}, fmt.Errorf("start update project update thread tx: %w", err)
	}
	defer rollback(tx)

	existing, err := tx.ProjectUpdateThread.Query().
		Where(
			entprojectupdatethread.IDEQ(input.ThreadID),
			entprojectupdatethread.ProjectIDEQ(input.ProjectID),
			entprojectupdatethread.IsDeleted(false),
		).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return Thread{}, ErrThreadNotFound
		}
		return Thread{}, fmt.Errorf("get project update thread for update: %w", err)
	}

	now := time.Now().UTC()
	revisionNumber, err := s.ensureInitialThreadRevisionTx(ctx, tx, existing)
	if err != nil {
		return Thread{}, err
	}
	editor := resolveCreatedBy(input.EditedBy)
	nextStatus := mapStatus(input.Status)
	revisionNumber++
	body := strings.TrimSpace(input.Body)
	title := resolveThreadTitle(input.Title, body)

	item, err := tx.ProjectUpdateThread.UpdateOneID(existing.ID).
		SetStatus(nextStatus).
		SetTitle(title).
		SetBodyMarkdown(body).
		SetUpdatedAt(now).
		SetEditedAt(now).
		SetEditCount(revisionNumber - 1).
		SetLastEditedBy(editor).
		SetLastActivityAt(now).
		Save(ctx)
	if err != nil {
		return Thread{}, fmt.Errorf("update project update thread: %w", err)
	}
	if err := s.appendThreadRevisionTx(ctx, tx, item.ID, revisionNumber, item.Status, item.Title, item.BodyMarkdown, editor, now, input.EditReason); err != nil {
		return Thread{}, err
	}
	if err := tx.Commit(); err != nil {
		return Thread{}, fmt.Errorf("commit update project update thread tx: %w", err)
	}
	if err := s.emitThreadActivity(ctx, threadEventType(existing.Status, item.Status), item, existing); err != nil {
		return Thread{}, err
	}

	return mapThread(item), nil
}

func (s *Service) RemoveThread(ctx context.Context, projectID, threadID uuid.UUID) (DeleteThreadResult, error) {
	if s.client == nil {
		return DeleteThreadResult{}, ErrUnavailable
	}

	now := time.Now().UTC()
	item, err := s.client.ProjectUpdateThread.Update().
		Where(
			entprojectupdatethread.IDEQ(threadID),
			entprojectupdatethread.ProjectIDEQ(projectID),
			entprojectupdatethread.IsDeleted(false),
		).
		SetIsDeleted(true).
		SetDeletedAt(now).
		SetDeletedBy(defaultCreatedBy).
		SetUpdatedAt(now).
		SetLastActivityAt(now).
		Save(ctx)
	if err != nil {
		return DeleteThreadResult{}, fmt.Errorf("soft delete project update thread: %w", err)
	}
	if item == 0 {
		return DeleteThreadResult{}, ErrThreadNotFound
	}

	deleted, err := s.getThread(ctx, projectID, threadID)
	if err != nil {
		return DeleteThreadResult{}, err
	}
	if err := s.emitThreadActivity(ctx, activityevent.TypeProjectUpdateThreadDeleted, deleted, nil); err != nil {
		return DeleteThreadResult{}, err
	}

	return DeleteThreadResult{DeletedThreadID: threadID}, nil
}

func (s *Service) AddComment(ctx context.Context, input AddCommentInput) (Comment, error) {
	if s.client == nil {
		return Comment{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Comment{}, fmt.Errorf("start add project update comment tx: %w", err)
	}
	defer rollback(tx)

	thread, err := tx.ProjectUpdateThread.Query().
		Where(
			entprojectupdatethread.IDEQ(input.ThreadID),
			entprojectupdatethread.ProjectIDEQ(input.ProjectID),
			entprojectupdatethread.IsDeleted(false),
		).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return Comment{}, ErrThreadNotFound
		}
		return Comment{}, fmt.Errorf("get project update thread for comment create: %w", err)
	}

	now := time.Now().UTC()
	createdBy := resolveCreatedBy(input.CreatedBy)
	item, err := tx.ProjectUpdateComment.Create().
		SetThreadID(thread.ID).
		SetBodyMarkdown(strings.TrimSpace(input.Body)).
		SetCreatedBy(createdBy).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		return Comment{}, fmt.Errorf("create project update comment: %w", err)
	}
	if err := s.appendCommentRevisionTx(ctx, tx, item.ID, 1, item.BodyMarkdown, createdBy, now, ""); err != nil {
		return Comment{}, err
	}
	if _, err := tx.ProjectUpdateThread.UpdateOneID(thread.ID).
		SetLastActivityAt(now).
		AddCommentCount(1).
		Save(ctx); err != nil {
		return Comment{}, fmt.Errorf("update project update thread activity after comment create: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("commit add project update comment tx: %w", err)
	}
	if err := s.emitCommentActivity(ctx, activityevent.TypeProjectUpdateCommentCreated, input.ProjectID, thread, item); err != nil {
		return Comment{}, err
	}

	return mapComment(item), nil
}

func (s *Service) UpdateComment(ctx context.Context, input UpdateCommentInput) (Comment, error) {
	if s.client == nil {
		return Comment{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return Comment{}, fmt.Errorf("start update project update comment tx: %w", err)
	}
	defer rollback(tx)

	thread, comment, err := s.getThreadAndCommentTx(ctx, tx, input.ProjectID, input.ThreadID, input.CommentID, false)
	if err != nil {
		return Comment{}, err
	}

	now := time.Now().UTC()
	revisionNumber, err := s.ensureInitialCommentRevisionTx(ctx, tx, comment)
	if err != nil {
		return Comment{}, err
	}
	editor := resolveCreatedBy(input.EditedBy)
	revisionNumber++

	item, err := tx.ProjectUpdateComment.UpdateOneID(comment.ID).
		SetBodyMarkdown(strings.TrimSpace(input.Body)).
		SetUpdatedAt(now).
		SetEditedAt(now).
		SetEditCount(revisionNumber - 1).
		SetLastEditedBy(editor).
		Save(ctx)
	if err != nil {
		return Comment{}, fmt.Errorf("update project update comment: %w", err)
	}
	if err := s.appendCommentRevisionTx(ctx, tx, item.ID, revisionNumber, item.BodyMarkdown, editor, now, input.EditReason); err != nil {
		return Comment{}, err
	}
	if _, err := tx.ProjectUpdateThread.UpdateOneID(thread.ID).
		SetLastActivityAt(now).
		Save(ctx); err != nil {
		return Comment{}, fmt.Errorf("update project update thread activity after comment edit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Comment{}, fmt.Errorf("commit update project update comment tx: %w", err)
	}
	if err := s.emitCommentActivity(ctx, activityevent.TypeProjectUpdateCommentEdited, input.ProjectID, thread, item); err != nil {
		return Comment{}, err
	}

	return mapComment(item), nil
}

func (s *Service) RemoveComment(ctx context.Context, projectID, threadID, commentID uuid.UUID) (DeleteCommentResult, error) {
	if s.client == nil {
		return DeleteCommentResult{}, ErrUnavailable
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return DeleteCommentResult{}, fmt.Errorf("start delete project update comment tx: %w", err)
	}
	defer rollback(tx)

	thread, comment, err := s.getThreadAndCommentTx(ctx, tx, projectID, threadID, commentID, true)
	if err != nil {
		return DeleteCommentResult{}, err
	}

	now := time.Now().UTC()
	if _, err := tx.ProjectUpdateComment.UpdateOneID(comment.ID).
		SetIsDeleted(true).
		SetDeletedAt(now).
		SetDeletedBy(defaultCreatedBy).
		SetUpdatedAt(now).
		Save(ctx); err != nil {
		return DeleteCommentResult{}, fmt.Errorf("soft delete project update comment: %w", err)
	}
	threadUpdate := tx.ProjectUpdateThread.UpdateOneID(thread.ID).SetLastActivityAt(now)
	if thread.CommentCount > 0 {
		threadUpdate.SetCommentCount(thread.CommentCount - 1)
	}
	if _, err := threadUpdate.Save(ctx); err != nil {
		return DeleteCommentResult{}, fmt.Errorf("update project update thread activity after comment delete: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return DeleteCommentResult{}, fmt.Errorf("commit delete project update comment tx: %w", err)
	}
	if err := s.emitCommentActivity(ctx, activityevent.TypeProjectUpdateCommentDeleted, projectID, thread, comment); err != nil {
		return DeleteCommentResult{}, err
	}

	return DeleteCommentResult{DeletedCommentID: commentID}, nil
}

func (s *Service) ListCommentRevisions(ctx context.Context, projectID, threadID, commentID uuid.UUID) ([]CommentRevision, error) {
	if s.client == nil {
		return nil, ErrUnavailable
	}

	thread, err := s.getThread(ctx, projectID, threadID)
	if err != nil {
		return nil, err
	}
	comment, err := s.client.ProjectUpdateComment.Query().
		Where(
			entprojectupdatecomment.IDEQ(commentID),
			entprojectupdatecomment.ThreadIDEQ(thread.ID),
		).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("get project update comment for revisions: %w", err)
	}

	items, err := s.client.ProjectUpdateCommentRevision.Query().
		Where(entprojectupdatecommentrevision.CommentIDEQ(comment.ID)).
		Order(entdb.Asc(entprojectupdatecommentrevision.FieldRevisionNumber), entdb.Asc(entprojectupdatecommentrevision.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project update comment revisions: %w", err)
	}
	if len(items) == 0 {
		return []CommentRevision{s.syntheticInitialCommentRevision(comment)}, nil
	}

	revisions := make([]CommentRevision, 0, len(items))
	for _, item := range items {
		revisions = append(revisions, mapCommentRevision(item))
	}
	return revisions, nil
}

func (s *Service) ensureProjectExists(ctx context.Context, projectID uuid.UUID) error {
	exists, err := s.client.Project.Query().Where(entproject.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}
	return nil
}

func (s *Service) ensureProjectExistsTx(ctx context.Context, tx *entdb.Tx, projectID uuid.UUID) error {
	exists, err := tx.Project.Query().Where(entproject.ID(projectID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check project existence: %w", err)
	}
	if !exists {
		return ErrProjectNotFound
	}
	return nil
}

func (s *Service) getThread(ctx context.Context, projectID, threadID uuid.UUID) (*entdb.ProjectUpdateThread, error) {
	item, err := s.client.ProjectUpdateThread.Query().
		Where(
			entprojectupdatethread.IDEQ(threadID),
			entprojectupdatethread.ProjectIDEQ(projectID),
		).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, ErrThreadNotFound
		}
		return nil, fmt.Errorf("get project update thread: %w", err)
	}
	return item, nil
}

func (s *Service) getThreadAndCommentTx(
	ctx context.Context,
	tx *entdb.Tx,
	projectID, threadID, commentID uuid.UUID,
	allowDeletedThread bool,
) (*entdb.ProjectUpdateThread, *entdb.ProjectUpdateComment, error) {
	threadQuery := tx.ProjectUpdateThread.Query().Where(
		entprojectupdatethread.IDEQ(threadID),
		entprojectupdatethread.ProjectIDEQ(projectID),
	)
	if !allowDeletedThread {
		threadQuery.Where(entprojectupdatethread.IsDeleted(false))
	}
	thread, err := threadQuery.Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, nil, ErrThreadNotFound
		}
		return nil, nil, fmt.Errorf("get project update thread for comment: %w", err)
	}

	comment, err := tx.ProjectUpdateComment.Query().
		Where(
			entprojectupdatecomment.IDEQ(commentID),
			entprojectupdatecomment.ThreadIDEQ(thread.ID),
			entprojectupdatecomment.IsDeleted(false),
		).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, nil, ErrCommentNotFound
		}
		return nil, nil, fmt.Errorf("get project update comment: %w", err)
	}
	return thread, comment, nil
}

func (s *Service) ensureInitialThreadRevisionTx(ctx context.Context, tx *entdb.Tx, thread *entdb.ProjectUpdateThread) (int, error) {
	latest, err := tx.ProjectUpdateThreadRevision.Query().
		Where(entprojectupdatethreadrevision.ThreadIDEQ(thread.ID)).
		Order(entdb.Desc(entprojectupdatethreadrevision.FieldRevisionNumber), entdb.Desc(entprojectupdatethreadrevision.FieldID)).
		First(ctx)
	switch {
	case err == nil:
		return latest.RevisionNumber, nil
	case !entdb.IsNotFound(err):
		return 0, fmt.Errorf("load latest project update thread revision: %w", err)
	}

	if err := s.appendThreadRevisionTx(ctx, tx, thread.ID, 1, thread.Status, thread.Title, thread.BodyMarkdown, thread.CreatedBy, thread.CreatedAt, ""); err != nil {
		return 0, err
	}
	return 1, nil
}

func (s *Service) appendThreadRevisionTx(
	ctx context.Context,
	tx *entdb.Tx,
	threadID uuid.UUID,
	revisionNumber int,
	status entprojectupdatethread.Status,
	title string,
	bodyMarkdown string,
	editedBy string,
	editedAt time.Time,
	editReason string,
) error {
	create := tx.ProjectUpdateThreadRevision.Create().
		SetThreadID(threadID).
		SetRevisionNumber(revisionNumber).
		SetStatus(mapThreadRevisionStatus(status)).
		SetTitle(title).
		SetBodyMarkdown(bodyMarkdown).
		SetEditedBy(resolveCreatedBy(editedBy)).
		SetEditedAt(editedAt)
	if trimmed := strings.TrimSpace(editReason); trimmed != "" {
		create.SetEditReason(trimmed)
	}
	if _, err := create.Save(ctx); err != nil {
		return fmt.Errorf("create project update thread revision: %w", err)
	}
	return nil
}

func (s *Service) ensureInitialCommentRevisionTx(ctx context.Context, tx *entdb.Tx, comment *entdb.ProjectUpdateComment) (int, error) {
	latest, err := tx.ProjectUpdateCommentRevision.Query().
		Where(entprojectupdatecommentrevision.CommentIDEQ(comment.ID)).
		Order(entdb.Desc(entprojectupdatecommentrevision.FieldRevisionNumber), entdb.Desc(entprojectupdatecommentrevision.FieldID)).
		First(ctx)
	switch {
	case err == nil:
		return latest.RevisionNumber, nil
	case !entdb.IsNotFound(err):
		return 0, fmt.Errorf("load latest project update comment revision: %w", err)
	}

	if err := s.appendCommentRevisionTx(ctx, tx, comment.ID, 1, comment.BodyMarkdown, comment.CreatedBy, comment.CreatedAt, ""); err != nil {
		return 0, err
	}
	return 1, nil
}

func (s *Service) appendCommentRevisionTx(
	ctx context.Context,
	tx *entdb.Tx,
	commentID uuid.UUID,
	revisionNumber int,
	bodyMarkdown string,
	editedBy string,
	editedAt time.Time,
	editReason string,
) error {
	create := tx.ProjectUpdateCommentRevision.Create().
		SetCommentID(commentID).
		SetRevisionNumber(revisionNumber).
		SetBodyMarkdown(bodyMarkdown).
		SetEditedBy(resolveCreatedBy(editedBy)).
		SetEditedAt(editedAt)
	if trimmed := strings.TrimSpace(editReason); trimmed != "" {
		create.SetEditReason(trimmed)
	}
	if _, err := create.Save(ctx); err != nil {
		return fmt.Errorf("create project update comment revision: %w", err)
	}
	return nil
}

func (s *Service) syntheticInitialThreadRevision(thread *entdb.ProjectUpdateThread) ThreadRevision {
	return ThreadRevision{
		ID:             uuid.Nil,
		ThreadID:       thread.ID,
		RevisionNumber: 1,
		Status:         Status(thread.Status),
		Title:          thread.Title,
		BodyMarkdown:   thread.BodyMarkdown,
		EditedBy:       thread.CreatedBy,
		EditedAt:       thread.CreatedAt,
	}
}

func (s *Service) syntheticInitialCommentRevision(comment *entdb.ProjectUpdateComment) CommentRevision {
	return CommentRevision{
		ID:             uuid.Nil,
		CommentID:      comment.ID,
		RevisionNumber: 1,
		BodyMarkdown:   comment.BodyMarkdown,
		EditedBy:       comment.CreatedBy,
		EditedAt:       comment.CreatedAt,
	}
}

func (s *Service) emitThreadActivity(
	ctx context.Context,
	eventType activityevent.Type,
	thread *entdb.ProjectUpdateThread,
	previous *entdb.ProjectUpdateThread,
) error {
	if s.activityEmitter == nil || thread == nil {
		return nil
	}
	metadata := map[string]any{
		"thread_id":      thread.ID.String(),
		"thread_title":   thread.Title,
		"thread_status":  thread.Status.String(),
		"comment_count":  thread.CommentCount,
		"changed_fields": []string{"updates"},
	}
	if previous != nil && previous.Status != thread.Status {
		metadata["from_status"] = previous.Status.String()
		metadata["to_status"] = thread.Status.String()
		metadata["changed_fields"] = []string{"status", "updates"}
	}
	message := buildThreadActivityMessage(eventType, thread)
	if _, err := s.activityEmitter.Emit(ctx, activity.RecordInput{
		ProjectID: thread.ProjectID,
		EventType: eventType,
		Message:   message,
		Metadata:  metadata,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("emit project update thread activity: %w", err)
	}
	return nil
}

func (s *Service) emitCommentActivity(
	ctx context.Context,
	eventType activityevent.Type,
	projectID uuid.UUID,
	thread *entdb.ProjectUpdateThread,
	comment *entdb.ProjectUpdateComment,
) error {
	if s.activityEmitter == nil || thread == nil || comment == nil {
		return nil
	}
	if _, err := s.activityEmitter.Emit(ctx, activity.RecordInput{
		ProjectID: projectID,
		EventType: eventType,
		Message:   buildCommentActivityMessage(eventType, thread),
		Metadata: map[string]any{
			"thread_id":      thread.ID.String(),
			"thread_title":   thread.Title,
			"thread_status":  thread.Status.String(),
			"comment_id":     comment.ID.String(),
			"changed_fields": []string{"comments", "updates"},
		},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("emit project update comment activity: %w", err)
	}
	return nil
}

func buildThreadActivityMessage(eventType activityevent.Type, thread *entdb.ProjectUpdateThread) string {
	switch eventType {
	case activityevent.TypeProjectUpdateThreadCreated:
		return fmt.Sprintf("Posted project update: %s", thread.Title)
	case activityevent.TypeProjectUpdateThreadDeleted:
		return fmt.Sprintf("Deleted project update: %s", thread.Title)
	case activityevent.TypeProjectUpdateThreadStatusChanged:
		return fmt.Sprintf("Changed project update status to %s: %s", humanizeStatus(Status(thread.Status)), thread.Title)
	default:
		return fmt.Sprintf("Updated project update: %s", thread.Title)
	}
}

func buildCommentActivityMessage(eventType activityevent.Type, thread *entdb.ProjectUpdateThread) string {
	switch eventType {
	case activityevent.TypeProjectUpdateCommentCreated:
		return fmt.Sprintf("Commented on project update: %s", thread.Title)
	case activityevent.TypeProjectUpdateCommentDeleted:
		return fmt.Sprintf("Deleted a project update comment: %s", thread.Title)
	default:
		return fmt.Sprintf("Edited a project update comment: %s", thread.Title)
	}
}

func threadEventType(from, to entprojectupdatethread.Status) activityevent.Type {
	if from != to {
		return activityevent.TypeProjectUpdateThreadStatusChanged
	}
	return activityevent.TypeProjectUpdateThreadEdited
}

func resolveCreatedBy(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultCreatedBy
	}
	return trimmed
}

func mapStatus(status Status) entprojectupdatethread.Status {
	switch status {
	case StatusAtRisk:
		return entprojectupdatethread.StatusAtRisk
	case StatusOffTrack:
		return entprojectupdatethread.StatusOffTrack
	default:
		return entprojectupdatethread.StatusOnTrack
	}
}

func mapThreadRevisionStatus(status entprojectupdatethread.Status) entprojectupdatethreadrevision.Status {
	switch status {
	case entprojectupdatethread.StatusAtRisk:
		return entprojectupdatethreadrevision.StatusAtRisk
	case entprojectupdatethread.StatusOffTrack:
		return entprojectupdatethreadrevision.StatusOffTrack
	default:
		return entprojectupdatethreadrevision.StatusOnTrack
	}
}

func humanizeStatus(status Status) string {
	switch status {
	case StatusAtRisk:
		return "At risk"
	case StatusOffTrack:
		return "Off track"
	default:
		return "On track"
	}
}

func resolveThreadTitle(rawTitle, body string) string {
	title := strings.TrimSpace(rawTitle)
	if title != "" {
		return title
	}
	return deriveThreadTitleFromBody(body)
}

func deriveThreadTitleFromBody(body string) string {
	const maxTitleRunes = 100

	normalized := strings.Join(strings.Fields(strings.TrimSpace(body)), " ")
	if normalized == "" {
		return ""
	}

	runes := []rune(normalized)
	if len(runes) <= maxTitleRunes {
		return normalized
	}

	truncated := runes[:maxTitleRunes]
	for idx := len(truncated) - 1; idx >= 0; idx-- {
		if unicode.IsSpace(truncated[idx]) {
			return strings.TrimSpace(string(truncated[:idx]))
		}
	}

	return string(truncated)
}

func mapThread(item *entdb.ProjectUpdateThread) Thread {
	comments := make([]Comment, 0, len(item.Edges.Comments))
	for _, comment := range item.Edges.Comments {
		comments = append(comments, mapComment(comment))
	}
	return Thread{
		ID:             item.ID,
		ProjectID:      item.ProjectID,
		Status:         Status(item.Status),
		Title:          item.Title,
		BodyMarkdown:   item.BodyMarkdown,
		CreatedBy:      item.CreatedBy,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
		EditedAt:       item.EditedAt,
		EditCount:      item.EditCount,
		LastEditedBy:   item.LastEditedBy,
		IsDeleted:      item.IsDeleted,
		DeletedAt:      item.DeletedAt,
		DeletedBy:      item.DeletedBy,
		LastActivityAt: item.LastActivityAt,
		CommentCount:   item.CommentCount,
		Comments:       comments,
	}
}

func mapThreadRevision(item *entdb.ProjectUpdateThreadRevision) ThreadRevision {
	return ThreadRevision{
		ID:             item.ID,
		ThreadID:       item.ThreadID,
		RevisionNumber: item.RevisionNumber,
		Status:         Status(item.Status),
		Title:          item.Title,
		BodyMarkdown:   item.BodyMarkdown,
		EditedBy:       item.EditedBy,
		EditedAt:       item.EditedAt,
		EditReason:     item.EditReason,
	}
}

func mapComment(item *entdb.ProjectUpdateComment) Comment {
	return Comment{
		ID:           item.ID,
		ThreadID:     item.ThreadID,
		BodyMarkdown: item.BodyMarkdown,
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

func mapCommentRevision(item *entdb.ProjectUpdateCommentRevision) CommentRevision {
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

func threadCursorForEntity(item *entdb.ProjectUpdateThread) projectupdatedomain.ThreadCursor {
	return projectupdatedomain.ThreadCursor{
		LastActivityAt: item.LastActivityAt.UTC(),
		ID:             item.ID,
	}
}

func rollback(tx *entdb.Tx) {
	_ = tx.Rollback()
}
