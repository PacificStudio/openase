package projectupdate

import (
	"context"
	"errors"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
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
		Title:     "Sprint 2 rollout",
		Body:      "Initial rollout plan",
		CreatedBy: " user:codex ",
	})
	if err != nil {
		t.Fatalf("AddThread(threadOne) error = %v", err)
	}
	if threadOne.CreatedBy != "user:codex" || threadOne.CommentCount != 0 || threadOne.Status != StatusOnTrack {
		t.Fatalf("AddThread(threadOne) = %+v", threadOne)
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
		Title:      "Sprint 2 rollout",
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
