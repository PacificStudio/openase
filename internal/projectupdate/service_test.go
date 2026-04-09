package projectupdate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	projectupdatedomain "github.com/BetterAndBetterII/openase/internal/domain/projectupdate"
	"github.com/google/uuid"
)

func TestProjectUpdateServiceNilClientGuards(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil)
	ctx := context.Background()
	projectID := newUUID()
	threadID := newUUID()
	commentID := newUUID()

	if _, err := service.ListThreads(ctx, projectID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListThreads() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.ListThreadPage(ctx, projectupdatedomain.ListThreadsPage{
		ProjectID: projectID,
		Limit:     projectupdatedomain.DefaultThreadPageLimit,
	}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListThreadPage() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.AddThread(ctx, AddThreadInput{ProjectID: projectID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("AddThread() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.UpdateThread(ctx, UpdateThreadInput{ProjectID: projectID, ThreadID: threadID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("UpdateThread() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.RemoveThread(ctx, projectID, threadID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RemoveThread() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.ListThreadRevisions(ctx, projectID, threadID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListThreadRevisions() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.AddComment(ctx, AddCommentInput{ProjectID: projectID, ThreadID: threadID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("AddComment() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.UpdateComment(ctx, UpdateCommentInput{ProjectID: projectID, ThreadID: threadID, CommentID: commentID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("UpdateComment() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.RemoveComment(ctx, projectID, threadID, commentID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RemoveComment() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.ListCommentRevisions(ctx, projectID, threadID, commentID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListCommentRevisions() error = %v, want %v", err, ErrUnavailable)
	}
}

func TestProjectUpdateServiceCRUDAndOrdering(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID := seedProject(ctx, t, client)
	service := NewService(client, nil)

	threadOne, err := service.AddThread(ctx, AddThreadInput{
		ProjectID: projectID,
		Status:    StatusOnTrack,
		Body:      "Initial rollout plan",
		CreatedBy: " user:codex ",
	})
	if err != nil {
		t.Fatalf("AddThread(threadOne) error = %v", err)
	}
	if threadOne.CreatedBy != "user:codex" || threadOne.CommentCount != 0 || threadOne.Status != StatusOnTrack {
		t.Fatalf("AddThread(threadOne) = %+v", threadOne)
	}
	if threadOne.Title != "Initial rollout plan" {
		t.Fatalf("AddThread(threadOne) title = %q, want derived body title", threadOne.Title)
	}

	threadTwo, err := service.AddThread(ctx, AddThreadInput{
		ProjectID: projectID,
		Status:    StatusOffTrack,
		Title:     "Infra migration",
		Body:      "Waiting on upstream cleanup",
	})
	if err != nil {
		t.Fatalf("AddThread(threadTwo) error = %v", err)
	}

	updatedThreadOne, err := service.UpdateThread(ctx, UpdateThreadInput{
		ProjectID:  projectID,
		ThreadID:   threadOne.ID,
		Status:     StatusAtRisk,
		Body:       "Blocked by flaky deploy validation",
		EditedBy:   " user:reviewer ",
		EditReason: "status recalibration",
	})
	if err != nil {
		t.Fatalf("UpdateThread(threadOne) error = %v", err)
	}
	if updatedThreadOne.Status != StatusAtRisk || updatedThreadOne.EditCount != 1 || updatedThreadOne.LastEditedBy == nil || *updatedThreadOne.LastEditedBy != "user:reviewer" {
		t.Fatalf("UpdateThread(threadOne) = %+v", updatedThreadOne)
	}
	if updatedThreadOne.Title != "Blocked by flaky deploy validation" {
		t.Fatalf("UpdateThread(threadOne) title = %q, want derived body title", updatedThreadOne.Title)
	}

	commentOne, err := service.AddComment(ctx, AddCommentInput{
		ProjectID: projectID,
		ThreadID:  threadOne.ID,
		Body:      "Need a fresh canary run before Friday.",
		CreatedBy: " user:ops ",
	})
	if err != nil {
		t.Fatalf("AddComment(commentOne) error = %v", err)
	}

	commentTwo, err := service.AddComment(ctx, AddCommentInput{
		ProjectID: projectID,
		ThreadID:  threadOne.ID,
		Body:      "Provider limits were adjusted this morning.",
	})
	if err != nil {
		t.Fatalf("AddComment(commentTwo) error = %v", err)
	}

	updatedCommentOne, err := service.UpdateComment(ctx, UpdateCommentInput{
		ProjectID:  projectID,
		ThreadID:   threadOne.ID,
		CommentID:  commentOne.ID,
		Body:       "Need a fresh canary run before Friday noon.",
		EditedBy:   "user:ops",
		EditReason: "narrowed the deadline",
	})
	if err != nil {
		t.Fatalf("UpdateComment(commentOne) error = %v", err)
	}
	if updatedCommentOne.EditCount != 1 || updatedCommentOne.LastEditedBy == nil || *updatedCommentOne.LastEditedBy != "user:ops" {
		t.Fatalf("UpdateComment(commentOne) = %+v", updatedCommentOne)
	}

	if _, err := service.RemoveComment(ctx, projectID, threadOne.ID, commentTwo.ID); err != nil {
		t.Fatalf("RemoveComment(commentTwo) error = %v", err)
	}
	if _, err := service.RemoveThread(ctx, projectID, threadTwo.ID); err != nil {
		t.Fatalf("RemoveThread(threadTwo) error = %v", err)
	}

	threads, err := service.ListThreads(ctx, projectID)
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if len(threads) != 2 {
		t.Fatalf("ListThreads() len = %d, want 2", len(threads))
	}
	if threads[0].ID != threadTwo.ID || threads[1].ID != threadOne.ID {
		t.Fatalf("ListThreads() ordering = %+v, want latest activity first", threads)
	}
	if !threads[0].IsDeleted {
		t.Fatalf("ListThreads() deleted thread = %+v", threads[0])
	}
	if threads[0].Title != "Infra migration" || threads[0].BodyMarkdown != "Waiting on upstream cleanup" {
		t.Fatalf("ListThreads() explicit title/body thread = %+v", threads[0])
	}
	if threads[1].CommentCount != 1 || len(threads[1].Comments) != 2 {
		t.Fatalf("ListThreads() threadOne comments = %+v", threads[1])
	}
	if !threads[1].Comments[1].IsDeleted {
		t.Fatalf("ListThreads() deleted comment = %+v", threads[1].Comments[1])
	}

	threadRevisions, err := service.ListThreadRevisions(ctx, projectID, threadOne.ID)
	if err != nil {
		t.Fatalf("ListThreadRevisions() error = %v", err)
	}
	if len(threadRevisions) != 2 || threadRevisions[0].Status != StatusOnTrack || threadRevisions[1].Status != StatusAtRisk {
		t.Fatalf("ListThreadRevisions() = %+v", threadRevisions)
	}
	if threadRevisions[1].EditReason == nil || *threadRevisions[1].EditReason != "status recalibration" {
		t.Fatalf("ListThreadRevisions() edit reason = %+v", threadRevisions[1])
	}

	commentRevisions, err := service.ListCommentRevisions(ctx, projectID, threadOne.ID, commentOne.ID)
	if err != nil {
		t.Fatalf("ListCommentRevisions() error = %v", err)
	}
	if len(commentRevisions) != 2 || commentRevisions[0].BodyMarkdown != commentOne.BodyMarkdown || commentRevisions[1].BodyMarkdown != updatedCommentOne.BodyMarkdown {
		t.Fatalf("ListCommentRevisions() = %+v", commentRevisions)
	}
	if commentRevisions[1].EditReason == nil || *commentRevisions[1].EditReason != "narrowed the deadline" {
		t.Fatalf("ListCommentRevisions() edit reason = %+v", commentRevisions[1])
	}

	if _, err := service.AddComment(ctx, AddCommentInput{
		ProjectID: projectID,
		ThreadID:  threadTwo.ID,
		Body:      "Should not be accepted",
	}); !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("AddComment(deleted thread) error = %v, want %v", err, ErrThreadNotFound)
	}
}

func TestProjectUpdateServiceListThreadPageUsesStableCursorPagination(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	projectID := seedProject(ctx, t, client)
	service := NewService(client, nil)

	sharedActivityAt := time.Date(2026, 4, 1, 11, 30, 0, 0, time.UTC)
	olderActivityAt := sharedActivityAt.Add(-1 * time.Hour)

	threadA, err := service.AddThread(ctx, AddThreadInput{
		ProjectID: projectID,
		Status:    StatusOnTrack,
		Title:     "Thread A",
		Body:      "Alpha",
	})
	if err != nil {
		t.Fatalf("AddThread(threadA) error = %v", err)
	}
	threadB, err := service.AddThread(ctx, AddThreadInput{
		ProjectID: projectID,
		Status:    StatusAtRisk,
		Title:     "Thread B",
		Body:      "Beta",
	})
	if err != nil {
		t.Fatalf("AddThread(threadB) error = %v", err)
	}
	threadC, err := service.AddThread(ctx, AddThreadInput{
		ProjectID: projectID,
		Status:    StatusOffTrack,
		Title:     "Thread C",
		Body:      "Gamma",
	})
	if err != nil {
		t.Fatalf("AddThread(threadC) error = %v", err)
	}

	if _, err := client.ProjectUpdateThread.UpdateOneID(threadA.ID).
		SetLastActivityAt(sharedActivityAt).
		SetUpdatedAt(sharedActivityAt).
		Save(ctx); err != nil {
		t.Fatalf("set threadA activity: %v", err)
	}
	if _, err := client.ProjectUpdateThread.UpdateOneID(threadB.ID).
		SetLastActivityAt(sharedActivityAt).
		SetUpdatedAt(sharedActivityAt).
		Save(ctx); err != nil {
		t.Fatalf("set threadB activity: %v", err)
	}
	if _, err := client.ProjectUpdateThread.UpdateOneID(threadC.ID).
		SetLastActivityAt(olderActivityAt).
		SetUpdatedAt(olderActivityAt).
		Save(ctx); err != nil {
		t.Fatalf("set threadC activity: %v", err)
	}

	firstPage, err := service.ListThreadPage(ctx, projectupdatedomain.ListThreadsPage{
		ProjectID: projectID,
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("ListThreadPage(first) error = %v", err)
	}
	if !firstPage.HasMore || firstPage.NextCursor == "" {
		t.Fatalf("ListThreadPage(first) page metadata = %+v", firstPage)
	}
	if len(firstPage.Threads) != 2 {
		t.Fatalf("ListThreadPage(first) len = %d, want 2", len(firstPage.Threads))
	}
	if !firstPage.Threads[0].LastActivityAt.Equal(sharedActivityAt) || !firstPage.Threads[1].LastActivityAt.Equal(sharedActivityAt) {
		t.Fatalf("ListThreadPage(first) shared timestamps = %+v", firstPage.Threads)
	}
	if firstPage.Threads[0].ID.String() < firstPage.Threads[1].ID.String() {
		t.Fatalf("ListThreadPage(first) ids = %s, %s, want desc tie-break", firstPage.Threads[0].ID, firstPage.Threads[1].ID)
	}

	before, err := projectupdatedomain.ParseThreadCursor(firstPage.NextCursor)
	if err != nil {
		t.Fatalf("ParseThreadCursor(firstPage.NextCursor) error = %v", err)
	}
	if before.ID != firstPage.Threads[1].ID || !before.LastActivityAt.Equal(firstPage.Threads[1].LastActivityAt) {
		t.Fatalf("next cursor = %+v, want thread %+v", before, firstPage.Threads[1])
	}

	secondPage, err := service.ListThreadPage(ctx, projectupdatedomain.ListThreadsPage{
		ProjectID: projectID,
		Limit:     2,
		Before:    &before,
	})
	if err != nil {
		t.Fatalf("ListThreadPage(second) error = %v", err)
	}
	if secondPage.HasMore || secondPage.NextCursor != "" {
		t.Fatalf("ListThreadPage(second) page metadata = %+v", secondPage)
	}
	if len(secondPage.Threads) != 1 || secondPage.Threads[0].ID != threadC.ID {
		t.Fatalf("ListThreadPage(second) = %+v, want only threadC", secondPage.Threads)
	}
}

func TestDeriveThreadTitleFromBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "collapses whitespace",
			body: "  Ship room\n\nstatus update   is green  ",
			want: "Ship room status update is green",
		},
		{
			name: "truncates at word boundary",
			body: "This update body is intentionally longer than one hundred characters so the stored title stops at the last whole word",
			want: "This update body is intentionally longer than one hundred characters so the stored title stops at",
		},
		{
			name: "hard truncates long token",
			body: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			want: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := deriveThreadTitleFromBody(tc.body); got != tc.want {
				t.Fatalf("deriveThreadTitleFromBody() = %q, want %q", got, tc.want)
			}
		})
	}
}

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()
	return testPostgres.NewIsolatedEntClient(t)
}

func seedProject(ctx context.Context, t *testing.T, client *ent.Client) uuid.UUID {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Platform").
		SetSlug("platform").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return project.ID
}

func newUUID() uuid.UUID {
	return uuid.New()
}
