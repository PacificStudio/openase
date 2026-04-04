package ticket

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func priorityPtr(priority Priority) *Priority {
	return &priority
}

func TestTicketServiceCRUDDependenciesCommentsLinksAndRunRelease(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	parent, err := service.Create(ctx, CreateInput{
		ProjectID:       fixture.projectID,
		Title:           "Parent ticket",
		Description:     "raise backend coverage",
		Priority:        priorityPtr(PriorityHigh),
		Type:            "feature",
		WorkflowID:      &fixture.workflowID,
		TargetMachineID: &fixture.workerOneID,
		BudgetUSD:       50,
	})
	if err != nil {
		t.Fatalf("Create(parent) error = %v", err)
	}
	if parent.Identifier != "ASE-8" || parent.CreatedBy != defaultCreatedBy || parent.StatusID != fixture.backlogID || parent.TargetMachineID == nil || *parent.TargetMachineID != fixture.workerOneID {
		t.Fatalf("Create(parent) = %+v", parent)
	}

	child, err := service.Create(ctx, CreateInput{
		ProjectID:      fixture.projectID,
		Title:          "Child ticket",
		Description:    "split follow-up work",
		StatusID:       &fixture.todoID,
		Priority:       priorityPtr(PriorityLow),
		Type:           "chore",
		CreatedBy:      " codex ",
		ParentTicketID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("Create(child) error = %v", err)
	}
	if child.Parent == nil || child.Parent.ID != parent.ID || child.CreatedBy != "codex" || len(child.Dependencies) != 1 || child.Dependencies[0].Type != DependencyTypeSubIssue {
		t.Fatalf("Create(child) = %+v", child)
	}

	parentAfterChild, err := service.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Get(parent) error = %v", err)
	}
	if len(parentAfterChild.Children) != 1 || parentAfterChild.Children[0].ID != child.ID {
		t.Fatalf("Get(parent) = %+v", parentAfterChild)
	}

	filtered, err := service.List(ctx, ListInput{
		ProjectID:   fixture.projectID,
		StatusNames: []string{"Todo"},
		Priorities:  []Priority{PriorityLow},
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("List(filtered) error = %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != child.ID {
		t.Fatalf("List(filtered) = %+v", filtered)
	}

	blocksDependency, err := service.AddDependency(ctx, AddDependencyInput{
		TicketID:       parent.ID,
		TargetTicketID: fixture.legacyTicketID,
		Type:           DependencyTypeBlocks,
	})
	if err != nil {
		t.Fatalf("AddDependency(blocks) error = %v", err)
	}
	if blocksDependency.Target.ID != fixture.legacyTicketID || blocksDependency.Type != DependencyTypeBlocks {
		t.Fatalf("AddDependency(blocks) = %+v", blocksDependency)
	}
	legacyBlockedByParent, err := service.Get(ctx, fixture.legacyTicketID)
	if err != nil {
		t.Fatalf("Get(legacy blocked by parent) error = %v", err)
	}
	if len(legacyBlockedByParent.IncomingDependencies) != 1 || legacyBlockedByParent.IncomingDependencies[0].ID != blocksDependency.ID || legacyBlockedByParent.IncomingDependencies[0].Target.ID != parent.ID {
		t.Fatalf("Get(legacy blocked by parent) = %+v", legacyBlockedByParent)
	}
	if _, err := service.AddDependency(ctx, AddDependencyInput{
		TicketID:       parent.ID,
		TargetTicketID: fixture.legacyTicketID,
		Type:           DependencyTypeBlocks,
	}); err != ErrDependencyConflict {
		t.Fatalf("AddDependency(duplicate) error = %v, want %v", err, ErrDependencyConflict)
	}

	removeBlocksResult, err := service.RemoveDependency(ctx, parent.ID, blocksDependency.ID)
	if err != nil {
		t.Fatalf("RemoveDependency(blocks) error = %v", err)
	}
	if removeBlocksResult.DeletedDependencyID != blocksDependency.ID {
		t.Fatalf("RemoveDependency(blocks) = %+v", removeBlocksResult)
	}

	blocksDependency, err = service.AddDependency(ctx, AddDependencyInput{
		TicketID:       parent.ID,
		TargetTicketID: fixture.legacyTicketID,
		Type:           DependencyTypeBlocks,
	})
	if err != nil {
		t.Fatalf("AddDependency(blocks re-add) error = %v", err)
	}
	removeInboundBlocksResult, err := service.RemoveDependency(ctx, fixture.legacyTicketID, blocksDependency.ID)
	if err != nil {
		t.Fatalf("RemoveDependency(blocks via target) error = %v", err)
	}
	if removeInboundBlocksResult.DeletedDependencyID != blocksDependency.ID {
		t.Fatalf("RemoveDependency(blocks via target) = %+v", removeInboundBlocksResult)
	}

	removeSubIssueResult, err := service.RemoveDependency(ctx, child.ID, child.Dependencies[0].ID)
	if err != nil {
		t.Fatalf("RemoveDependency(sub-issue) error = %v", err)
	}
	if removeSubIssueResult.DeletedDependencyID != child.Dependencies[0].ID {
		t.Fatalf("RemoveDependency(sub-issue) = %+v", removeSubIssueResult)
	}
	childAfterRemove, err := service.Get(ctx, child.ID)
	if err != nil {
		t.Fatalf("Get(child after dependency delete) error = %v", err)
	}
	if childAfterRemove.Parent != nil || len(childAfterRemove.Dependencies) != 0 {
		t.Fatalf("Get(child after dependency delete) = %+v", childAfterRemove)
	}

	linkOne, err := service.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   parent.ID,
		LinkType:   ExternalLinkTypeGithubPR,
		URL:        "https://github.com/PacificStudio/openase/pull/278",
		ExternalID: "PacificStudio/openase#278",
		Title:      "coverage rollout",
		Status:     "open",
		Relation:   ExternalLinkRelationResolves,
	})
	if err != nil {
		t.Fatalf("AddExternalLink(linkOne) error = %v", err)
	}
	if linkOne.Relation != ExternalLinkRelationResolves {
		t.Fatalf("AddExternalLink(linkOne) = %+v", linkOne)
	}
	if _, err := service.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   parent.ID,
		LinkType:   ExternalLinkTypeGithubPR,
		URL:        "https://github.com/PacificStudio/openase/pull/278",
		ExternalID: "PacificStudio/openase#278",
		Relation:   ExternalLinkRelationRelated,
	}); err != ErrExternalLinkConflict {
		t.Fatalf("AddExternalLink(duplicate) error = %v, want %v", err, ErrExternalLinkConflict)
	}
	linkTwo, err := service.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   parent.ID,
		LinkType:   ExternalLinkTypeJiraTicket,
		URL:        "https://jira.example.com/browse/ASE-278",
		ExternalID: "ASE-278-JIRA",
		Relation:   ExternalLinkRelationRelated,
	})
	if err != nil {
		t.Fatalf("AddExternalLink(linkTwo) error = %v", err)
	}
	parentWithLinks, err := service.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Get(parent with links) error = %v", err)
	}
	if parentWithLinks.ExternalRef != "PacificStudio/openase#278" || len(parentWithLinks.ExternalLinks) != 2 {
		t.Fatalf("Get(parent with links) = %+v", parentWithLinks)
	}

	removeLinkOne, err := service.RemoveExternalLink(ctx, parent.ID, linkOne.ID)
	if err != nil {
		t.Fatalf("RemoveExternalLink(linkOne) error = %v", err)
	}
	if removeLinkOne.DeletedExternalLinkID != linkOne.ID {
		t.Fatalf("RemoveExternalLink(linkOne) = %+v", removeLinkOne)
	}
	parentAfterLinkOne, err := service.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Get(parent after linkOne delete) error = %v", err)
	}
	if parentAfterLinkOne.ExternalRef != "ASE-278-JIRA" || len(parentAfterLinkOne.ExternalLinks) != 1 {
		t.Fatalf("Get(parent after linkOne delete) = %+v", parentAfterLinkOne)
	}

	removeLinkTwo, err := service.RemoveExternalLink(ctx, parent.ID, linkTwo.ID)
	if err != nil {
		t.Fatalf("RemoveExternalLink(linkTwo) error = %v", err)
	}
	if removeLinkTwo.DeletedExternalLinkID != linkTwo.ID {
		t.Fatalf("RemoveExternalLink(linkTwo) = %+v", removeLinkTwo)
	}
	parentAfterLinksRemoved, err := service.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Get(parent after all links deleted) error = %v", err)
	}
	if parentAfterLinksRemoved.ExternalRef != "" || len(parentAfterLinksRemoved.ExternalLinks) != 0 {
		t.Fatalf("Get(parent after all links deleted) = %+v", parentAfterLinksRemoved)
	}

	commentOne, err := service.AddComment(ctx, AddCommentInput{
		TicketID:  parent.ID,
		Body:      "first comment",
		CreatedBy: "",
	})
	if err != nil {
		t.Fatalf("AddComment(commentOne) error = %v", err)
	}
	commentTwo, err := service.AddComment(ctx, AddCommentInput{
		TicketID:  parent.ID,
		Body:      "second comment",
		CreatedBy: " reviewer ",
	})
	if err != nil {
		t.Fatalf("AddComment(commentTwo) error = %v", err)
	}
	comments, err := service.ListComments(ctx, parent.ID)
	if err != nil {
		t.Fatalf("ListComments() error = %v", err)
	}
	if len(comments) != 2 || comments[0].ID != commentOne.ID || comments[0].CreatedBy != defaultCreatedBy || comments[1].ID != commentTwo.ID || comments[1].CreatedBy != "reviewer" {
		t.Fatalf("ListComments() = %+v", comments)
	}
	updatedCommentOne, err := service.UpdateComment(ctx, UpdateCommentInput{
		TicketID:   parent.ID,
		CommentID:  commentOne.ID,
		Body:       "updated first comment",
		EditedBy:   "agent:codex",
		EditReason: "clarify scope",
	})
	if err != nil {
		t.Fatalf("UpdateComment() error = %v", err)
	}
	if updatedCommentOne.BodyMarkdown != "updated first comment" || updatedCommentOne.EditCount != 1 || updatedCommentOne.LastEditedBy == nil || *updatedCommentOne.LastEditedBy != "agent:codex" {
		t.Fatalf("UpdateComment() = %+v", updatedCommentOne)
	}
	revisions, err := service.ListCommentRevisions(ctx, parent.ID, commentOne.ID)
	if err != nil {
		t.Fatalf("ListCommentRevisions() error = %v", err)
	}
	if len(revisions) != 2 || revisions[0].RevisionNumber != 1 || revisions[0].BodyMarkdown != "first comment" || revisions[1].RevisionNumber != 2 || revisions[1].BodyMarkdown != "updated first comment" {
		t.Fatalf("ListCommentRevisions() = %+v", revisions)
	}
	removeCommentResult, err := service.RemoveComment(ctx, parent.ID, commentTwo.ID)
	if err != nil {
		t.Fatalf("RemoveComment() error = %v", err)
	}
	if removeCommentResult.DeletedCommentID != commentTwo.ID {
		t.Fatalf("RemoveComment() = %+v", removeCommentResult)
	}
	commentsAfterDelete, err := service.ListComments(ctx, parent.ID)
	if err != nil {
		t.Fatalf("ListComments(after delete) error = %v", err)
	}
	if len(commentsAfterDelete) != 2 || !commentsAfterDelete[1].IsDeleted || commentsAfterDelete[1].DeletedAt == nil {
		t.Fatalf("ListComments(after delete) = %+v", commentsAfterDelete)
	}

	if _, err := client.Ticket.UpdateOneID(parent.ID).SetCostAmount(75).Save(ctx); err != nil {
		t.Fatalf("seed ticket cost amount: %v", err)
	}
	pausedTicket, err := service.Update(ctx, UpdateInput{
		TicketID:  parent.ID,
		BudgetUSD: Some(50.0),
	})
	if err != nil {
		t.Fatalf("Update(budget pause) error = %v", err)
	}
	if !pausedTicket.RetryPaused || pausedTicket.PauseReason != ticketing.PauseReasonBudgetExhausted.String() {
		t.Fatalf("Update(budget pause) = %+v", pausedTicket)
	}
	resumedTicket, err := service.Update(ctx, UpdateInput{
		TicketID:  parent.ID,
		BudgetUSD: Some(120.0),
	})
	if err != nil {
		t.Fatalf("Update(budget resume) error = %v", err)
	}
	if resumedTicket.RetryPaused || resumedTicket.PauseReason != "" {
		t.Fatalf("Update(budget resume) = %+v", resumedTicket)
	}
	retryAt := time.Date(2026, 3, 27, 16, 0, 0, 0, time.UTC)
	seedRetryToken := uuid.NewString()
	if _, err := client.Ticket.UpdateOneID(parent.ID).
		SetAttemptCount(4).
		SetConsecutiveErrors(2).
		SetNextRetryAt(retryAt).
		SetRetryPaused(true).
		SetPauseReason(ticketing.PauseReasonBudgetExhausted.String()).
		SetRetryToken(seedRetryToken).
		Save(ctx); err != nil {
		t.Fatalf("seed retry baseline before manual status change: %v", err)
	}

	runID := seedTicketCurrentRun(ctx, t, client, fixture, parent.ID)
	if _, err := client.Ticket.UpdateOneID(parent.ID).SetStallCount(2).Save(ctx); err != nil {
		t.Fatalf("seed ticket stall count: %v", err)
	}
	updatedParent, err := service.Update(ctx, UpdateInput{
		TicketID:                          parent.ID,
		StatusID:                          Some(fixture.doneID),
		TargetMachineID:                   Some(&fixture.workerTwoID),
		RestrictStatusToWorkflowFinishSet: true,
	})
	if err != nil {
		t.Fatalf("Update(status transition) error = %v", err)
	}
	if updatedParent.CurrentRunID != nil || updatedParent.StatusID != fixture.doneID || updatedParent.TargetMachineID == nil || *updatedParent.TargetMachineID != fixture.workerTwoID {
		t.Fatalf("Update(status transition) = %+v", updatedParent)
	}
	updatedParentEntity, err := client.Ticket.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("reload updated parent: %v", err)
	}
	if updatedParentEntity.StallCount != 0 {
		t.Fatalf("expected status transition to reset stall count, got %+v", updatedParentEntity)
	}
	if updatedParent.AttemptCount != 4 || updatedParent.ConsecutiveErrors != 0 || updatedParent.NextRetryAt != nil || updatedParent.RetryPaused || updatedParent.PauseReason != "" {
		t.Fatalf("Update(status transition) should normalize retry baseline, got %+v", updatedParent)
	}
	parentAfter, err := client.Ticket.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("reload ticket after status transition: %v", err)
	}
	if parentAfter.RetryToken == "" || parentAfter.RetryToken == seedRetryToken {
		t.Fatalf("expected status transition to rotate retry token, got %q", parentAfter.RetryToken)
	}

	runAfterRelease, err := client.AgentRun.Get(ctx, runID)
	if err != nil {
		t.Fatalf("reload agent run after release: %v", err)
	}
	if runAfterRelease.Status != entagentrun.StatusTerminated || runAfterRelease.SessionID != "" || runAfterRelease.RuntimeStartedAt != nil || runAfterRelease.LastHeartbeatAt != nil || runAfterRelease.LastError != "" {
		t.Fatalf("agent run after release = %+v", runAfterRelease)
	}
	agentAfterRelease, err := client.Agent.Get(ctx, fixture.agentID)
	if err != nil {
		t.Fatalf("reload agent after release: %v", err)
	}
	if agentAfterRelease.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("agent runtime control state = %s, want active", agentAfterRelease.RuntimeControlState)
	}

	doneTickets, err := service.List(ctx, ListInput{
		ProjectID:   fixture.projectID,
		StatusNames: []string{"Done"},
		Priorities:  []Priority{PriorityHigh},
		Limit:       5,
	})
	if err != nil {
		t.Fatalf("List(done) error = %v", err)
	}
	if len(doneTickets) != 1 || doneTickets[0].ID != parent.ID {
		t.Fatalf("List(done) = %+v", doneTickets)
	}
}

func TestTicketServiceValidationAndNotFoundPaths(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	parent, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Parent ticket",
		StatusID:   &fixture.todoID,
		Priority:   priorityPtr(PriorityMedium),
		Type:       "feature",
		WorkflowID: &fixture.workflowID,
	})
	if err != nil {
		t.Fatalf("Create(parent) error = %v", err)
	}
	child, err := service.Create(ctx, CreateInput{
		ProjectID:      fixture.projectID,
		Title:          "Child ticket",
		StatusID:       &fixture.todoID,
		Priority:       priorityPtr(PriorityLow),
		Type:           "chore",
		ParentTicketID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("Create(child) error = %v", err)
	}

	if _, err := service.List(ctx, ListInput{ProjectID: uuid.New()}); err != ErrProjectNotFound {
		t.Fatalf("List(missing project) error = %v, want %v", err, ErrProjectNotFound)
	}
	if _, err := service.Get(ctx, uuid.New()); err != ErrTicketNotFound {
		t.Fatalf("Get(missing ticket) error = %v, want %v", err, ErrTicketNotFound)
	}

	if _, err := service.Create(ctx, CreateInput{
		ProjectID: fixture.projectID,
		Title:     "Bad status",
		StatusID:  &fixture.otherProjectTodoID,
		Priority:  priorityPtr(PriorityMedium),
		Type:      "feature",
	}); err != ErrStatusNotFound {
		t.Fatalf("Create(status mismatch) error = %v, want %v", err, ErrStatusNotFound)
	}
	if _, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Bad workflow",
		StatusID:   &fixture.todoID,
		Priority:   priorityPtr(PriorityMedium),
		Type:       "feature",
		WorkflowID: &fixture.otherWorkflowID,
	}); err != ErrWorkflowNotFound {
		t.Fatalf("Create(workflow mismatch) error = %v, want %v", err, ErrWorkflowNotFound)
	}
	if _, err := service.Create(ctx, CreateInput{
		ProjectID:       fixture.projectID,
		Title:           "Bad machine",
		StatusID:        &fixture.todoID,
		Priority:        priorityPtr(PriorityMedium),
		Type:            "feature",
		TargetMachineID: &fixture.foreignMachineID,
	}); err != ErrTargetMachineNotFound {
		t.Fatalf("Create(machine mismatch) error = %v, want %v", err, ErrTargetMachineNotFound)
	}

	if _, err := service.Update(ctx, UpdateInput{
		TicketID:                          parent.ID,
		StatusID:                          Some(fixture.todoID),
		RestrictStatusToWorkflowFinishSet: true,
	}); err != ErrStatusNotAllowed {
		t.Fatalf("Update(status restriction) error = %v, want %v", err, ErrStatusNotAllowed)
	}
	if _, err := service.Update(ctx, UpdateInput{
		TicketID:   parent.ID,
		WorkflowID: Some(&fixture.otherWorkflowID),
	}); err != ErrWorkflowNotFound {
		t.Fatalf("Update(workflow mismatch) error = %v, want %v", err, ErrWorkflowNotFound)
	}
	if _, err := service.Update(ctx, UpdateInput{
		TicketID:        parent.ID,
		TargetMachineID: Some(&fixture.foreignMachineID),
	}); err != ErrTargetMachineNotFound {
		t.Fatalf("Update(machine mismatch) error = %v, want %v", err, ErrTargetMachineNotFound)
	}
	if _, err := service.Update(ctx, UpdateInput{
		TicketID:       parent.ID,
		ParentTicketID: Some(&child.ID),
	}); err != ErrInvalidDependency {
		t.Fatalf("Update(cycle) error = %v, want %v", err, ErrInvalidDependency)
	}

	if _, err := service.AddDependency(ctx, AddDependencyInput{
		TicketID:       parent.ID,
		TargetTicketID: parent.ID,
		Type:           DependencyTypeBlocks,
	}); err != ErrInvalidDependency {
		t.Fatalf("AddDependency(self) error = %v, want %v", err, ErrInvalidDependency)
	}
	if _, err := service.AddDependency(ctx, AddDependencyInput{
		TicketID:       parent.ID,
		TargetTicketID: fixture.otherProjectTodoID,
		Type:           DependencyTypeBlocks,
	}); err != ErrTicketNotFound {
		t.Fatalf("AddDependency(other project) error = %v, want %v", err, ErrTicketNotFound)
	}
	if _, err := service.RemoveDependency(ctx, parent.ID, uuid.New()); err != ErrDependencyNotFound {
		t.Fatalf("RemoveDependency(missing) error = %v, want %v", err, ErrDependencyNotFound)
	}
	if _, err := service.RemoveDependency(ctx, parent.ID, child.Dependencies[0].ID); err != ErrDependencyNotFound {
		t.Fatalf("RemoveDependency(sub-issue via parent target) error = %v, want %v", err, ErrDependencyNotFound)
	}

	link, err := service.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   parent.ID,
		LinkType:   ExternalLinkTypeGithubIssue,
		URL:        "https://github.com/PacificStudio/openase/issues/278",
		ExternalID: "PacificStudio/openase#278",
		Relation:   ExternalLinkRelationRelated,
	})
	if err != nil {
		t.Fatalf("AddExternalLink() error = %v", err)
	}
	if _, err := service.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   uuid.New(),
		LinkType:   ExternalLinkTypeGithubIssue,
		URL:        "https://github.com/PacificStudio/openase/issues/999",
		ExternalID: "PacificStudio/openase#999",
		Relation:   ExternalLinkRelationRelated,
	}); err != ErrTicketNotFound {
		t.Fatalf("AddExternalLink(missing ticket) error = %v, want %v", err, ErrTicketNotFound)
	}
	if _, err := service.RemoveExternalLink(ctx, parent.ID, uuid.New()); err != ErrExternalLinkNotFound {
		t.Fatalf("RemoveExternalLink(missing) error = %v, want %v", err, ErrExternalLinkNotFound)
	}
	if _, err := service.RemoveExternalLink(ctx, child.ID, link.ID); err != ErrExternalLinkNotFound {
		t.Fatalf("RemoveExternalLink(wrong ticket) error = %v, want %v", err, ErrExternalLinkNotFound)
	}

	comment, err := service.AddComment(ctx, AddCommentInput{
		TicketID: parent.ID,
		Body:     "hello",
	})
	if err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}
	if _, err := service.AddComment(ctx, AddCommentInput{
		TicketID: uuid.New(),
		Body:     "missing ticket",
	}); err != ErrTicketNotFound {
		t.Fatalf("AddComment(missing ticket) error = %v, want %v", err, ErrTicketNotFound)
	}
	if _, err := service.UpdateComment(ctx, UpdateCommentInput{
		TicketID:  parent.ID,
		CommentID: uuid.New(),
		Body:      "nope",
	}); err != ErrCommentNotFound {
		t.Fatalf("UpdateComment(missing) error = %v, want %v", err, ErrCommentNotFound)
	}
	if _, err := service.UpdateComment(ctx, UpdateCommentInput{
		TicketID:  child.ID,
		CommentID: comment.ID,
		Body:      "wrong ticket",
	}); err != ErrCommentNotFound {
		t.Fatalf("UpdateComment(wrong ticket) error = %v, want %v", err, ErrCommentNotFound)
	}
	if _, err := service.RemoveComment(ctx, parent.ID, uuid.New()); err != ErrCommentNotFound {
		t.Fatalf("RemoveComment(missing) error = %v, want %v", err, ErrCommentNotFound)
	}
	if _, err := service.ListCommentRevisions(ctx, parent.ID, uuid.New()); err != ErrCommentNotFound {
		t.Fatalf("ListCommentRevisions(missing) error = %v, want %v", err, ErrCommentNotFound)
	}

	if _, err := service.RemoveExternalLink(ctx, parent.ID, link.ID); err != nil {
		t.Fatalf("RemoveExternalLink(existing) error = %v", err)
	}
	if _, err := service.RemoveComment(ctx, parent.ID, comment.ID); err != nil {
		t.Fatalf("RemoveComment(existing) error = %v", err)
	}
}

func TestTicketServiceRunsCancelHookWhenNonFinishStatusChangeReleasesCurrentRun(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	projectItem, err := client.Project.Get(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(projectItem.OrganizationID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(localMachine.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider to local machine: %v", err)
	}
	if _, err := client.Workflow.UpdateOneID(fixture.workflowID).
		SetHooks(map[string]any{
			"ticket_hooks": map[string]any{
				"on_cancel": []any{
					map[string]any{"cmd": `printf 'cancel\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("set workflow cancel hooks: %v", err)
	}

	ticketItem, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Cancel current run",
		StatusID:   &fixture.todoID,
		Priority:   priorityPtr(PriorityHigh),
		Type:       "feature",
		WorkflowID: &fixture.workflowID,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runID := seedTicketCurrentRun(ctx, t, client, fixture, ticketItem.ID)

	repoItem, err := client.ProjectRepo.Create().
		SetProjectID(fixture.projectID).
		SetName("backend").
		SetRepositoryURL("https://github.com/acme/backend.git").
		SetDefaultBranch("main").
		SetWorkspaceDirname("backend").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}
	workspaceRoot := t.TempDir()
	repoPath := filepath.Join(workspaceRoot, "backend")
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	if _, err := client.TicketRepoWorkspace.Create().
		SetTicketID(ticketItem.ID).
		SetAgentRunID(runID).
		SetRepoID(repoItem.ID).
		SetWorkspaceRoot(workspaceRoot).
		SetRepoPath(repoPath).
		SetBranchName("agent/planner/ASE-cancel").
		SetState(entticketrepoworkspace.StateReady).
		SetPreparedAt(time.Now().UTC()).
		Save(ctx); err != nil {
		t.Fatalf("create ticket repo workspace: %v", err)
	}

	updated, err := service.Update(ctx, UpdateInput{
		TicketID:                          ticketItem.ID,
		StatusID:                          Some(fixture.backlogID),
		RestrictStatusToWorkflowFinishSet: false,
	})
	if err != nil {
		t.Fatalf("update ticket status: %v", err)
	}
	if updated.CurrentRunID != nil {
		t.Fatalf("expected released current run, got %+v", updated.CurrentRunID)
	}

	//nolint:gosec // Test controls the temporary workspace path under t.TempDir-backed fixtures.
	raw, err := os.ReadFile(filepath.Join(workspaceRoot, "hook.log"))
	if err != nil {
		t.Fatalf("read cancel hook log: %v", err)
	}
	if string(raw) != "cancel\n" {
		t.Fatalf("unexpected cancel hook log %q", string(raw))
	}
}

func TestTicketServiceRunsDoneHookWhenFinishStatusChangeReleasesCurrentRun(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	projectItem, err := client.Project.Get(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(projectItem.OrganizationID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(localMachine.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider to local machine: %v", err)
	}
	if _, err := client.Workflow.UpdateOneID(fixture.workflowID).
		SetHooks(map[string]any{
			"ticket_hooks": map[string]any{
				"on_done": []any{
					map[string]any{"cmd": `printf 'done\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
				"on_cancel": []any{
					map[string]any{"cmd": `printf 'cancel\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("set workflow done hooks: %v", err)
	}

	ticketItem, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Finish current run",
		StatusID:   &fixture.todoID,
		Priority:   priorityPtr(PriorityHigh),
		Type:       "feature",
		WorkflowID: &fixture.workflowID,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runID := seedTicketCurrentRun(ctx, t, client, fixture, ticketItem.ID)

	repoItem, err := client.ProjectRepo.Create().
		SetProjectID(fixture.projectID).
		SetName("backend").
		SetRepositoryURL("https://github.com/acme/backend.git").
		SetDefaultBranch("main").
		SetWorkspaceDirname("backend").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}
	workspaceRoot := t.TempDir()
	repoPath := filepath.Join(workspaceRoot, "backend")
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	if _, err := client.TicketRepoWorkspace.Create().
		SetTicketID(ticketItem.ID).
		SetAgentRunID(runID).
		SetRepoID(repoItem.ID).
		SetWorkspaceRoot(workspaceRoot).
		SetRepoPath(repoPath).
		SetBranchName("agent/planner/ASE-done").
		SetState(entticketrepoworkspace.StateReady).
		SetPreparedAt(time.Now().UTC()).
		Save(ctx); err != nil {
		t.Fatalf("create ticket repo workspace: %v", err)
	}

	updated, err := service.Update(ctx, UpdateInput{
		TicketID:                          ticketItem.ID,
		StatusID:                          Some(fixture.doneID),
		RestrictStatusToWorkflowFinishSet: true,
	})
	if err != nil {
		t.Fatalf("update ticket status: %v", err)
	}
	if updated.CurrentRunID != nil {
		t.Fatalf("expected released current run, got %+v", updated.CurrentRunID)
	}

	//nolint:gosec // Test controls the temporary workspace path under t.TempDir-backed fixtures.
	raw, err := os.ReadFile(filepath.Join(workspaceRoot, "hook.log"))
	if err != nil {
		t.Fatalf("read done hook log: %v", err)
	}
	if string(raw) != "done\n" {
		t.Fatalf("unexpected done hook log %q", string(raw))
	}
}

func TestTicketServiceKeepsCurrentRunWhenStatusStaysWithinWorkflowPickupSet(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	projectItem, err := client.Project.Get(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(projectItem.OrganizationID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(localMachine.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider to local machine: %v", err)
	}
	if _, err := client.Workflow.UpdateOneID(fixture.workflowID).
		AddPickupStatusIDs(fixture.backlogID).
		SetHooks(map[string]any{
			"ticket_hooks": map[string]any{
				"on_done": []any{
					map[string]any{"cmd": `printf 'done\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
				"on_cancel": []any{
					map[string]any{"cmd": `printf 'cancel\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("configure workflow pickup hooks: %v", err)
	}

	ticketItem, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Keep current run within pickup set",
		StatusID:   &fixture.todoID,
		Priority:   priorityPtr(PriorityHigh),
		Type:       "feature",
		WorkflowID: &fixture.workflowID,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runID := seedTicketCurrentRun(ctx, t, client, fixture, ticketItem.ID)
	workspaceRoot := seedTicketHookWorkspace(ctx, t, client, fixture.projectID, ticketItem.ID, runID, "agent/planner/ASE-retain")

	updated, err := service.Update(ctx, UpdateInput{
		TicketID:                          ticketItem.ID,
		StatusID:                          Some(fixture.backlogID),
		RestrictStatusToWorkflowFinishSet: false,
	})
	if err != nil {
		t.Fatalf("update ticket status within pickup set: %v", err)
	}
	if updated.CurrentRunID == nil || *updated.CurrentRunID != runID {
		t.Fatalf("expected current run to remain attached, got %+v", updated.CurrentRunID)
	}
	if updated.StatusID != fixture.backlogID {
		t.Fatalf("expected status %s, got %s", fixture.backlogID, updated.StatusID)
	}

	runAfter, err := client.AgentRun.Get(ctx, runID)
	if err != nil {
		t.Fatalf("reload run after pickup status change: %v", err)
	}
	if runAfter.Status != entagentrun.StatusExecuting ||
		runAfter.SessionID != "sess-active" ||
		runAfter.RuntimeStartedAt == nil ||
		runAfter.LastHeartbeatAt == nil {
		t.Fatalf("expected executing run to stay intact, got %+v", runAfter)
	}
	agentAfter, err := client.Agent.Get(ctx, fixture.agentID)
	if err != nil {
		t.Fatalf("reload agent after pickup status change: %v", err)
	}
	if agentAfter.RuntimeControlState != entagent.RuntimeControlStatePaused {
		t.Fatalf("expected agent control state to remain unchanged, got %+v", agentAfter)
	}
	if _, err := os.Stat(filepath.Join(workspaceRoot, "hook.log")); !os.IsNotExist(err) {
		t.Fatalf("expected no lifecycle hook output, got err=%v", err)
	}
}

func TestTicketServiceRunsCancelHookWhenArchivingReleasesCurrentRun(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	projectItem, err := client.Project.Get(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(projectItem.OrganizationID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(localMachine.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider to local machine: %v", err)
	}
	if _, err := client.Workflow.UpdateOneID(fixture.workflowID).
		SetHooks(map[string]any{
			"ticket_hooks": map[string]any{
				"on_done": []any{
					map[string]any{"cmd": `printf 'done\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
				"on_cancel": []any{
					map[string]any{"cmd": `printf 'cancel\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("set workflow hooks: %v", err)
	}

	ticketItem, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Archive current run",
		StatusID:   &fixture.todoID,
		Priority:   priorityPtr(PriorityHigh),
		Type:       "feature",
		WorkflowID: &fixture.workflowID,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runID := seedTicketCurrentRun(ctx, t, client, fixture, ticketItem.ID)
	workspaceRoot := seedTicketHookWorkspace(ctx, t, client, fixture.projectID, ticketItem.ID, runID, "agent/planner/ASE-archived-cancel")

	updated, err := service.Update(ctx, UpdateInput{
		TicketID:                          ticketItem.ID,
		Archived:                          Some(true),
		RestrictStatusToWorkflowFinishSet: false,
	})
	if err != nil {
		t.Fatalf("update ticket status: %v", err)
	}
	if updated.CurrentRunID != nil || !updated.Archived {
		t.Fatalf("unexpected archived update result: %+v", updated)
	}

	//nolint:gosec // Test controls the temporary workspace path under t.TempDir-backed fixtures.
	raw, err := os.ReadFile(filepath.Join(workspaceRoot, "hook.log"))
	if err != nil {
		t.Fatalf("read cancel hook log: %v", err)
	}
	if string(raw) != "cancel\n" {
		t.Fatalf("unexpected archived cancel hook log %q", string(raw))
	}
}

func TestTicketServiceRunsDoneHookWhenArchivedFinishStatusReleasesCurrentRun(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	projectItem, err := client.Project.Get(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(projectItem.OrganizationID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(localMachine.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider to local machine: %v", err)
	}
	if _, err := client.Workflow.UpdateOneID(fixture.workflowID).
		SetHooks(map[string]any{
			"ticket_hooks": map[string]any{
				"on_done": []any{
					map[string]any{"cmd": `printf 'done\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
				"on_cancel": []any{
					map[string]any{"cmd": `printf 'cancel\n' >> "$OPENASE_WORKSPACE/hook.log"`},
				},
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("set workflow hooks: %v", err)
	}

	archivedStatus, err := newTicketStatusService(client).Create(ctx, ticketstatus.CreateInput{
		ProjectID: fixture.projectID,
		Name:      "Archived",
		Stage:     ticketing.StatusStageCanceled,
		Color:     "#4B5563",
	})
	if err != nil {
		t.Fatalf("create archived status: %v", err)
	}
	if _, err := client.Workflow.UpdateOneID(fixture.workflowID).
		AddFinishStatusIDs(archivedStatus.ID).
		Save(ctx); err != nil {
		t.Fatalf("add archived status to workflow finish set: %v", err)
	}

	ticketItem, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Archive finished run",
		StatusID:   &fixture.todoID,
		Priority:   priorityPtr(PriorityHigh),
		Type:       "feature",
		WorkflowID: &fixture.workflowID,
	})
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runID := seedTicketCurrentRun(ctx, t, client, fixture, ticketItem.ID)
	workspaceRoot := seedTicketHookWorkspace(ctx, t, client, fixture.projectID, ticketItem.ID, runID, "agent/planner/ASE-archived-done")

	updated, err := service.Update(ctx, UpdateInput{
		TicketID:                          ticketItem.ID,
		StatusID:                          Some(archivedStatus.ID),
		RestrictStatusToWorkflowFinishSet: true,
	})
	if err != nil {
		t.Fatalf("update ticket status: %v", err)
	}
	if updated.CurrentRunID != nil || updated.StatusID != archivedStatus.ID || updated.StatusName != "Archived" {
		t.Fatalf("unexpected archived update result: %+v", updated)
	}

	//nolint:gosec // Test controls the temporary workspace path under t.TempDir-backed fixtures.
	raw, err := os.ReadFile(filepath.Join(workspaceRoot, "hook.log"))
	if err != nil {
		t.Fatalf("read done hook log: %v", err)
	}
	if string(raw) != "done\n" {
		t.Fatalf("unexpected archived done hook log %q", string(raw))
	}
}

func TestTicketServiceUpdateClearsFieldsAndResyncsSubIssueDependencies(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(ctx, t, client)
	service := newTicketService(client)

	parentOne, err := service.Create(ctx, CreateInput{
		ProjectID: fixture.projectID,
		Title:     "Parent one",
		StatusID:  &fixture.todoID,
		Priority:  priorityPtr(PriorityHigh),
		Type:      "feature",
	})
	if err != nil {
		t.Fatalf("Create(parentOne) error = %v", err)
	}
	parentTwo, err := service.Create(ctx, CreateInput{
		ProjectID: fixture.projectID,
		Title:     "Parent two",
		StatusID:  &fixture.todoID,
		Priority:  priorityPtr(PriorityHigh),
		Type:      "feature",
	})
	if err != nil {
		t.Fatalf("Create(parentTwo) error = %v", err)
	}
	child, err := service.Create(ctx, CreateInput{
		ProjectID:       fixture.projectID,
		Title:           "Child ticket",
		StatusID:        &fixture.todoID,
		Priority:        priorityPtr(PriorityMedium),
		Type:            "chore",
		WorkflowID:      &fixture.workflowID,
		TargetMachineID: &fixture.workerOneID,
		CreatedBy:       " initial ",
		ParentTicketID:  &parentOne.ID,
		ExternalRef:     "GH-278",
	})
	if err != nil {
		t.Fatalf("Create(child) error = %v", err)
	}

	runID := seedTicketCurrentRun(ctx, t, client, fixture, child.ID)
	updated, err := service.Update(ctx, UpdateInput{
		TicketID:        child.ID,
		Priority:        Some((*Priority)(nil)),
		WorkflowID:      Some((*uuid.UUID)(nil)),
		TargetMachineID: Some((*uuid.UUID)(nil)),
		CreatedBy:       Some(" reviewer "),
		ParentTicketID:  Some(&parentTwo.ID),
		ExternalRef:     Some("   "),
	})
	if err != nil {
		t.Fatalf("Update(reparent and clear fields) error = %v", err)
	}
	if updated.WorkflowID != nil || updated.TargetMachineID != nil || updated.Parent == nil || updated.Parent.ID != parentTwo.ID || updated.ExternalRef != "" || updated.CreatedBy != "reviewer" || updated.CurrentRunID != nil || updated.Priority != "" {
		t.Fatalf("Update(reparent and clear fields) = %+v", updated)
	}
	if len(updated.Dependencies) != 1 || updated.Dependencies[0].Type != DependencyTypeSubIssue || updated.Dependencies[0].Target.ID != parentTwo.ID {
		t.Fatalf("Update(reparent and clear fields) dependencies = %+v", updated.Dependencies)
	}

	runAfterUpdate, err := client.AgentRun.Get(ctx, runID)
	if err != nil {
		t.Fatalf("load agent run after update: %v", err)
	}
	if runAfterUpdate.Status != entagentrun.StatusTerminated || runAfterUpdate.SessionID != "" || runAfterUpdate.RuntimeStartedAt != nil || runAfterUpdate.LastHeartbeatAt != nil || runAfterUpdate.LastError != "" {
		t.Fatalf("agent run after update = %+v", runAfterUpdate)
	}
	agentAfterUpdate, err := client.Agent.Get(ctx, fixture.agentID)
	if err != nil {
		t.Fatalf("load agent after update: %v", err)
	}
	if agentAfterUpdate.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("agent after update = %+v", agentAfterUpdate)
	}

	clearedParent, err := service.Update(ctx, UpdateInput{
		TicketID:       child.ID,
		ParentTicketID: Some((*uuid.UUID)(nil)),
	})
	if err != nil {
		t.Fatalf("Update(clear parent) error = %v", err)
	}
	if clearedParent.Parent != nil || len(clearedParent.Dependencies) != 0 {
		t.Fatalf("Update(clear parent) = %+v", clearedParent)
	}

	addedSubIssue, err := service.AddDependency(ctx, AddDependencyInput{
		TicketID:       child.ID,
		TargetTicketID: parentOne.ID,
		Type:           DependencyTypeSubIssue,
	})
	if err != nil {
		t.Fatalf("AddDependency(sub-issue) error = %v", err)
	}
	if addedSubIssue.Type != DependencyTypeSubIssue || addedSubIssue.Target.ID != parentOne.ID {
		t.Fatalf("AddDependency(sub-issue) = %+v", addedSubIssue)
	}

	reloadedChild, err := service.Get(ctx, child.ID)
	if err != nil {
		t.Fatalf("Get(child after AddDependency) error = %v", err)
	}
	if reloadedChild.Parent == nil || reloadedChild.Parent.ID != parentOne.ID || len(reloadedChild.Dependencies) != 1 {
		t.Fatalf("Get(child after AddDependency) = %+v", reloadedChild)
	}
}

type ticketServiceFixture struct {
	projectID          uuid.UUID
	otherProjectID     uuid.UUID
	backlogID          uuid.UUID
	todoID             uuid.UUID
	doneID             uuid.UUID
	otherProjectTodoID uuid.UUID
	workflowID         uuid.UUID
	otherWorkflowID    uuid.UUID
	workerOneID        uuid.UUID
	workerTwoID        uuid.UUID
	foreignMachineID   uuid.UUID
	providerID         uuid.UUID
	agentID            uuid.UUID
	legacyTicketID     uuid.UUID
}

func seedTicketServiceFixture(ctx context.Context, t *testing.T, client *ent.Client) ticketServiceFixture {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	otherOrg, err := client.Organization.Create().
		SetName("Other Org").
		SetSlug("other-org").
		Save(ctx)
	if err != nil {
		t.Fatalf("create other organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	otherProject, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Other").
		SetSlug("openase-other").
		Save(ctx)
	if err != nil {
		t.Fatalf("create other project: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	otherStatuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, otherProject.ID)
	if err != nil {
		t.Fatalf("reset other project statuses: %v", err)
	}

	workerOne, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("worker-one").
		SetHost("worker-one.internal").
		SetPort(22).
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker one: %v", err)
	}
	workerTwo, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("worker-two").
		SetHost("worker-two.internal").
		SetPort(22).
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker two: %v", err)
	}
	foreignMachine, err := client.Machine.Create().
		SetOrganizationID(otherOrg.ID).
		SetName("foreign-worker").
		SetHost("foreign-worker.internal").
		SetPort(22).
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create foreign worker: %v", err)
	}

	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(workerOne.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetCliArgs([]string{"run"}).
		SetAuthConfig(map[string]any{}).
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(providerItem.ID).
		SetName("planner").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	backlogID := findStatusIDByName(t, statuses, "Backlog")
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	otherTodoID := findStatusIDByName(t, otherStatuses, "Todo")

	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetAgentID(agentItem.ID).
		SetName("coding").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	otherWorkflowItem, err := client.Workflow.Create().
		SetProjectID(otherProject.ID).
		SetName("other").
		SetType("coding").
		SetHarnessPath("roles/other.md").
		AddPickupStatusIDs(otherTodoID).
		AddFinishStatusIDs(otherTodoID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create other workflow: %v", err)
	}

	legacyTicket, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-7").
		SetTitle("legacy ticket").
		SetStatusID(backlogID).
		SetPriority(entticket.PriorityMedium).
		SetType(entticket.TypeFeature).
		SetCreatedBy("seed").
		Save(ctx)
	if err != nil {
		t.Fatalf("create legacy ticket: %v", err)
	}

	return ticketServiceFixture{
		projectID:          project.ID,
		otherProjectID:     otherProject.ID,
		backlogID:          backlogID,
		todoID:             todoID,
		doneID:             doneID,
		otherProjectTodoID: otherTodoID,
		workflowID:         workflowItem.ID,
		otherWorkflowID:    otherWorkflowItem.ID,
		workerOneID:        workerOne.ID,
		workerTwoID:        workerTwo.ID,
		foreignMachineID:   foreignMachine.ID,
		providerID:         providerItem.ID,
		agentID:            agentItem.ID,
		legacyTicketID:     legacyTicket.ID,
	}
}

func seedTicketCurrentRun(ctx context.Context, t *testing.T, client *ent.Client, fixture ticketServiceFixture, ticketID uuid.UUID) uuid.UUID {
	t.Helper()

	now := time.Date(2026, 3, 27, 14, 0, 0, 0, time.UTC)
	if _, err := client.Agent.UpdateOneID(fixture.agentID).
		SetRuntimeControlState(entagent.RuntimeControlStatePaused).
		Save(ctx); err != nil {
		t.Fatalf("pause agent before current run seed: %v", err)
	}

	runItem, err := client.AgentRun.Create().
		SetAgentID(fixture.agentID).
		SetWorkflowID(fixture.workflowID).
		SetTicketID(ticketID).
		SetProviderID(fixture.providerID).
		SetStatus(entagentrun.StatusExecuting).
		SetSessionID("sess-active").
		SetRuntimeStartedAt(now).
		SetLastHeartbeatAt(now.Add(2 * time.Minute)).
		SetLastError("stuck").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketID).SetCurrentRunID(runItem.ID).Save(ctx); err != nil {
		t.Fatalf("set ticket current run: %v", err)
	}

	return runItem.ID
}

func seedTicketHookWorkspace(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	projectID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	branchName string,
) string {
	t.Helper()

	repoItem, err := client.ProjectRepo.Create().
		SetProjectID(projectID).
		SetName("backend").
		SetRepositoryURL("https://github.com/acme/backend.git").
		SetDefaultBranch("main").
		SetWorkspaceDirname("backend").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}

	workspaceRoot := t.TempDir()
	repoPath := filepath.Join(workspaceRoot, "backend")
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		t.Fatalf("create repo path: %v", err)
	}
	if _, err := client.TicketRepoWorkspace.Create().
		SetTicketID(ticketID).
		SetAgentRunID(runID).
		SetRepoID(repoItem.ID).
		SetWorkspaceRoot(workspaceRoot).
		SetRepoPath(repoPath).
		SetBranchName(branchName).
		SetState(entticketrepoworkspace.StateReady).
		SetPreparedAt(time.Now().UTC()).
		Save(ctx); err != nil {
		t.Fatalf("create ticket repo workspace: %v", err)
	}

	return workspaceRoot
}
