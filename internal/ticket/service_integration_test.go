package ticket

import (
	"context"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketexternallink "github.com/BetterAndBetterII/openase/ent/ticketexternallink"
	"github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
)

func TestTicketServiceCRUDDependenciesCommentsLinksAndRunRelease(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(t, ctx, client)
	service := NewService(client)

	parent, err := service.Create(ctx, CreateInput{
		ProjectID:       fixture.projectID,
		Title:           "Parent ticket",
		Description:     "raise backend coverage",
		Priority:        "high",
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
		Priority:       "low",
		Type:           "chore",
		CreatedBy:      " codex ",
		ParentTicketID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("Create(child) error = %v", err)
	}
	if child.Parent == nil || child.Parent.ID != parent.ID || child.CreatedBy != "codex" || len(child.Dependencies) != 1 || child.Dependencies[0].Type != entticketdependency.TypeSubIssue {
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
		Priorities:  []entticket.Priority{"low"},
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
		Type:           entticketdependency.TypeBlocks,
	})
	if err != nil {
		t.Fatalf("AddDependency(blocks) error = %v", err)
	}
	if blocksDependency.Target.ID != fixture.legacyTicketID || blocksDependency.Type != entticketdependency.TypeBlocks {
		t.Fatalf("AddDependency(blocks) = %+v", blocksDependency)
	}
	if _, err := service.AddDependency(ctx, AddDependencyInput{
		TicketID:       parent.ID,
		TargetTicketID: fixture.legacyTicketID,
		Type:           entticketdependency.TypeBlocks,
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
		LinkType:   entticketexternallink.LinkTypeGithubPr,
		URL:        "https://github.com/GrandCX/openase/pull/278",
		ExternalID: "GrandCX/openase#278",
		Title:      "coverage rollout",
		Status:     "open",
		Relation:   entticketexternallink.RelationResolves,
	})
	if err != nil {
		t.Fatalf("AddExternalLink(linkOne) error = %v", err)
	}
	if linkOne.Relation != entticketexternallink.RelationResolves {
		t.Fatalf("AddExternalLink(linkOne) = %+v", linkOne)
	}
	if _, err := service.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   parent.ID,
		LinkType:   entticketexternallink.LinkTypeGithubPr,
		URL:        "https://github.com/GrandCX/openase/pull/278",
		ExternalID: "GrandCX/openase#278",
		Relation:   entticketexternallink.RelationRelated,
	}); err != ErrExternalLinkConflict {
		t.Fatalf("AddExternalLink(duplicate) error = %v, want %v", err, ErrExternalLinkConflict)
	}
	linkTwo, err := service.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   parent.ID,
		LinkType:   entticketexternallink.LinkTypeJiraTicket,
		URL:        "https://jira.example.com/browse/ASE-278",
		ExternalID: "ASE-278-JIRA",
		Relation:   entticketexternallink.RelationRelated,
	})
	if err != nil {
		t.Fatalf("AddExternalLink(linkTwo) error = %v", err)
	}
	parentWithLinks, err := service.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Get(parent with links) error = %v", err)
	}
	if parentWithLinks.ExternalRef != "GrandCX/openase#278" || len(parentWithLinks.ExternalLinks) != 2 {
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
		TicketID:  parent.ID,
		CommentID: commentOne.ID,
		Body:      "updated first comment",
	})
	if err != nil {
		t.Fatalf("UpdateComment() error = %v", err)
	}
	if updatedCommentOne.Body != "updated first comment" {
		t.Fatalf("UpdateComment() = %+v", updatedCommentOne)
	}
	removeCommentResult, err := service.RemoveComment(ctx, parent.ID, commentTwo.ID)
	if err != nil {
		t.Fatalf("RemoveComment() error = %v", err)
	}
	if removeCommentResult.DeletedCommentID != commentTwo.ID {
		t.Fatalf("RemoveComment() = %+v", removeCommentResult)
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

	runID := seedTicketCurrentRun(t, ctx, client, fixture, parent.ID)
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
		Priorities:  []entticket.Priority{"high"},
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
	fixture := seedTicketServiceFixture(t, ctx, client)
	service := NewService(client)

	parent, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Parent ticket",
		StatusID:   &fixture.todoID,
		Priority:   "medium",
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
		Priority:       "low",
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
		Priority:  "medium",
		Type:      "feature",
	}); err != ErrStatusNotFound {
		t.Fatalf("Create(status mismatch) error = %v, want %v", err, ErrStatusNotFound)
	}
	if _, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Bad workflow",
		StatusID:   &fixture.todoID,
		Priority:   "medium",
		Type:       "feature",
		WorkflowID: &fixture.otherWorkflowID,
	}); err != ErrWorkflowNotFound {
		t.Fatalf("Create(workflow mismatch) error = %v, want %v", err, ErrWorkflowNotFound)
	}
	if _, err := service.Create(ctx, CreateInput{
		ProjectID:       fixture.projectID,
		Title:           "Bad machine",
		StatusID:        &fixture.todoID,
		Priority:        "medium",
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
		Type:           entticketdependency.TypeBlocks,
	}); err != ErrInvalidDependency {
		t.Fatalf("AddDependency(self) error = %v, want %v", err, ErrInvalidDependency)
	}
	if _, err := service.AddDependency(ctx, AddDependencyInput{
		TicketID:       parent.ID,
		TargetTicketID: fixture.otherProjectTodoID,
		Type:           entticketdependency.TypeBlocks,
	}); err != ErrTicketNotFound {
		t.Fatalf("AddDependency(other project) error = %v, want %v", err, ErrTicketNotFound)
	}
	if _, err := service.RemoveDependency(ctx, parent.ID, uuid.New()); err != ErrDependencyNotFound {
		t.Fatalf("RemoveDependency(missing) error = %v, want %v", err, ErrDependencyNotFound)
	}
	if _, err := service.RemoveDependency(ctx, parent.ID, child.Dependencies[0].ID); err != ErrDependencyNotFound {
		t.Fatalf("RemoveDependency(wrong source ticket) error = %v, want %v", err, ErrDependencyNotFound)
	}

	link, err := service.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   parent.ID,
		LinkType:   entticketexternallink.LinkTypeGithubIssue,
		URL:        "https://github.com/GrandCX/openase/issues/278",
		ExternalID: "GrandCX/openase#278",
		Relation:   entticketexternallink.RelationRelated,
	})
	if err != nil {
		t.Fatalf("AddExternalLink() error = %v", err)
	}
	if _, err := service.AddExternalLink(ctx, AddExternalLinkInput{
		TicketID:   uuid.New(),
		LinkType:   entticketexternallink.LinkTypeGithubIssue,
		URL:        "https://github.com/GrandCX/openase/issues/999",
		ExternalID: "GrandCX/openase#999",
		Relation:   entticketexternallink.RelationRelated,
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

	if _, err := service.RemoveExternalLink(ctx, parent.ID, link.ID); err != nil {
		t.Fatalf("RemoveExternalLink(existing) error = %v", err)
	}
	if _, err := service.RemoveComment(ctx, parent.ID, comment.ID); err != nil {
		t.Fatalf("RemoveComment(existing) error = %v", err)
	}
}

func TestTicketServiceUpdateClearsFieldsAndResyncsSubIssueDependencies(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(t, ctx, client)
	service := NewService(client)

	parentOne, err := service.Create(ctx, CreateInput{
		ProjectID: fixture.projectID,
		Title:     "Parent one",
		StatusID:  &fixture.todoID,
		Priority:  "high",
		Type:      "feature",
	})
	if err != nil {
		t.Fatalf("Create(parentOne) error = %v", err)
	}
	parentTwo, err := service.Create(ctx, CreateInput{
		ProjectID: fixture.projectID,
		Title:     "Parent two",
		StatusID:  &fixture.todoID,
		Priority:  "high",
		Type:      "feature",
	})
	if err != nil {
		t.Fatalf("Create(parentTwo) error = %v", err)
	}
	child, err := service.Create(ctx, CreateInput{
		ProjectID:       fixture.projectID,
		Title:           "Child ticket",
		StatusID:        &fixture.todoID,
		Priority:        "medium",
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

	runID := seedTicketCurrentRun(t, ctx, client, fixture, child.ID)
	updated, err := service.Update(ctx, UpdateInput{
		TicketID:        child.ID,
		WorkflowID:      Some((*uuid.UUID)(nil)),
		TargetMachineID: Some((*uuid.UUID)(nil)),
		CreatedBy:       Some(" reviewer "),
		ParentTicketID:  Some(&parentTwo.ID),
		ExternalRef:     Some("   "),
	})
	if err != nil {
		t.Fatalf("Update(reparent and clear fields) error = %v", err)
	}
	if updated.WorkflowID != nil || updated.TargetMachineID != nil || updated.Parent == nil || updated.Parent.ID != parentTwo.ID || updated.ExternalRef != "" || updated.CreatedBy != "reviewer" || updated.CurrentRunID != nil {
		t.Fatalf("Update(reparent and clear fields) = %+v", updated)
	}
	if len(updated.Dependencies) != 1 || updated.Dependencies[0].Type != entticketdependency.TypeSubIssue || updated.Dependencies[0].Target.ID != parentTwo.ID {
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
		Type:           entticketdependency.TypeSubIssue,
	})
	if err != nil {
		t.Fatalf("AddDependency(sub-issue) error = %v", err)
	}
	if addedSubIssue.Type != entticketdependency.TypeSubIssue || addedSubIssue.Target.ID != parentOne.ID {
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

func TestTicketServiceSyncRepoScopePRStatusTransitionsRetryAndFinish(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(t, ctx, client)
	service := NewService(client)

	backendRepo, err := client.ProjectRepo.Create().
		SetProjectID(fixture.projectID).
		SetName("backend").
		SetRepositoryURL("https://github.com/GrandCX/openase.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create backend repo: %v", err)
	}
	frontendRepo, err := client.ProjectRepo.Create().
		SetProjectID(fixture.projectID).
		SetName("frontend").
		SetRepositoryURL("https://github.com/GrandCX/openase-ui.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create frontend repo: %v", err)
	}

	retryTicket, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Retry from closed PR",
		StatusID:   &fixture.todoID,
		WorkflowID: &fixture.workflowID,
		Priority:   "high",
		Type:       "feature",
	})
	if err != nil {
		t.Fatalf("Create(retryTicket) error = %v", err)
	}
	retryRunID := seedTicketCurrentRun(t, ctx, client, fixture, retryTicket.ID)
	retryScope, err := client.TicketRepoScope.Create().
		SetTicketID(retryTicket.ID).
		SetRepoID(backendRepo.ID).
		SetBranchName("agent/codex/retry-ticket").
		SetPrStatus(ticketreposcope.PrStatusOpen).
		Save(ctx)
	if err != nil {
		t.Fatalf("create retry repo scope: %v", err)
	}

	retryResult, err := service.SyncRepoScopePRStatus(ctx, SyncRepoScopePRStatusInput{
		RepositoryURL:  backendRepo.RepositoryURL,
		BranchName:     "agent/codex/retry-ticket",
		PullRequestURL: "https://github.com/GrandCX/openase/pull/278",
		PRStatus:       ticketreposcope.PrStatusClosed,
	})
	if err != nil {
		t.Fatalf("SyncRepoScopePRStatus(retry) error = %v", err)
	}
	if !retryResult.Matched || retryResult.Outcome != RepoScopePRStatusSyncOutcomeRetried || retryResult.Ticket == nil {
		t.Fatalf("SyncRepoScopePRStatus(retry) = %+v", retryResult)
	}
	if retryResult.Ticket.CurrentRunID != nil || retryResult.Ticket.AttemptCount != 1 || retryResult.Ticket.ConsecutiveErrors != 1 || retryResult.Ticket.NextRetryAt == nil {
		t.Fatalf("retry ticket after sync = %+v", retryResult.Ticket)
	}
	retryScopeAfter, err := client.TicketRepoScope.Get(ctx, retryScope.ID)
	if err != nil {
		t.Fatalf("load retry scope: %v", err)
	}
	if retryScopeAfter.PrStatus != ticketreposcope.PrStatusClosed || retryScopeAfter.PullRequestURL != "https://github.com/GrandCX/openase/pull/278" {
		t.Fatalf("retry scope after sync = %+v", retryScopeAfter)
	}
	retryRunAfter, err := client.AgentRun.Get(ctx, retryRunID)
	if err != nil {
		t.Fatalf("load retry run: %v", err)
	}
	if retryRunAfter.Status != entagentrun.StatusErrored || retryRunAfter.SessionID != "" || retryRunAfter.LastError != "stuck" {
		t.Fatalf("retry run after sync = %+v", retryRunAfter)
	}
	retryAgentAfter, err := client.Agent.Get(ctx, fixture.agentID)
	if err != nil {
		t.Fatalf("load retry agent: %v", err)
	}
	if retryAgentAfter.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("retry agent after sync = %+v", retryAgentAfter)
	}

	finishTicket, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Finish from merged PRs",
		StatusID:   &fixture.todoID,
		WorkflowID: &fixture.workflowID,
		Priority:   "high",
		Type:       "feature",
	})
	if err != nil {
		t.Fatalf("Create(finishTicket) error = %v", err)
	}
	finishRunID := seedTicketCurrentRun(t, ctx, client, fixture, finishTicket.ID)
	finishScopeOne, err := client.TicketRepoScope.Create().
		SetTicketID(finishTicket.ID).
		SetRepoID(backendRepo.ID).
		SetBranchName("agent/codex/finish-ticket").
		SetPrStatus(ticketreposcope.PrStatusOpen).
		Save(ctx)
	if err != nil {
		t.Fatalf("create finish scope one: %v", err)
	}
	if _, err := client.TicketRepoScope.Create().
		SetTicketID(finishTicket.ID).
		SetRepoID(frontendRepo.ID).
		SetBranchName("agent/codex/finish-ticket").
		SetPrStatus(ticketreposcope.PrStatusMerged).
		Save(ctx); err != nil {
		t.Fatalf("create finish scope two: %v", err)
	}

	unfinishedResult, err := service.SyncRepoScopePRStatus(ctx, SyncRepoScopePRStatusInput{
		RepositoryFullName: "GrandCX/openase",
		BranchName:         "agent/codex/finish-ticket",
		PRStatus:           ticketreposcope.PrStatusOpen,
	})
	if err != nil {
		t.Fatalf("SyncRepoScopePRStatus(open) error = %v", err)
	}
	if !unfinishedResult.Matched || unfinishedResult.Outcome != RepoScopePRStatusSyncOutcomeNone || unfinishedResult.Ticket != nil {
		t.Fatalf("SyncRepoScopePRStatus(open) = %+v", unfinishedResult)
	}

	finishResult, err := service.SyncRepoScopePRStatus(ctx, SyncRepoScopePRStatusInput{
		RepositoryURL:  backendRepo.RepositoryURL,
		BranchName:     "agent/codex/finish-ticket",
		PullRequestURL: "https://github.com/GrandCX/openase/pull/279",
		PRStatus:       ticketreposcope.PrStatusMerged,
	})
	if err != nil {
		t.Fatalf("SyncRepoScopePRStatus(finish) error = %v", err)
	}
	if !finishResult.Matched || finishResult.Outcome != RepoScopePRStatusSyncOutcomeFinished || finishResult.Ticket == nil {
		t.Fatalf("SyncRepoScopePRStatus(finish) = %+v", finishResult)
	}
	if finishResult.Ticket.StatusID != fixture.doneID || finishResult.Ticket.CompletedAt == nil || finishResult.Ticket.CurrentRunID != nil {
		t.Fatalf("finish ticket after sync = %+v", finishResult.Ticket)
	}
	finishScopeAfter, err := client.TicketRepoScope.Get(ctx, finishScopeOne.ID)
	if err != nil {
		t.Fatalf("load finish scope one: %v", err)
	}
	if finishScopeAfter.PrStatus != ticketreposcope.PrStatusMerged || finishScopeAfter.PullRequestURL != "https://github.com/GrandCX/openase/pull/279" {
		t.Fatalf("finish scope after sync = %+v", finishScopeAfter)
	}
	finishRunAfter, err := client.AgentRun.Get(ctx, finishRunID)
	if err != nil {
		t.Fatalf("load finish run: %v", err)
	}
	if finishRunAfter.Status != entagentrun.StatusCompleted || finishRunAfter.SessionID != "" || finishRunAfter.LastError != "" {
		t.Fatalf("finish run after sync = %+v", finishRunAfter)
	}
	finishAgentAfter, err := client.Agent.Get(ctx, fixture.agentID)
	if err != nil {
		t.Fatalf("load finish agent: %v", err)
	}
	if finishAgentAfter.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("finish agent after sync = %+v", finishAgentAfter)
	}

	unmatched, err := service.SyncRepoScopePRStatus(ctx, SyncRepoScopePRStatusInput{
		RepositoryFullName: "GrandCX/unknown",
		BranchName:         "agent/codex/finish-ticket",
		PRStatus:           ticketreposcope.PrStatusMerged,
	})
	if err != nil {
		t.Fatalf("SyncRepoScopePRStatus(unmatched) error = %v", err)
	}
	if unmatched.Matched || unmatched.Outcome != RepoScopePRStatusSyncOutcomeNone || unmatched.Ticket != nil {
		t.Fatalf("SyncRepoScopePRStatus(unmatched) = %+v", unmatched)
	}
}

func TestTicketServiceSyncRepoScopePRStatusErrorBranches(t *testing.T) {
	client := openTestEntClient(t)
	ctx := context.Background()
	fixture := seedTicketServiceFixture(t, ctx, client)
	service := NewService(client)

	backendRepo, err := client.ProjectRepo.Create().
		SetProjectID(fixture.projectID).
		SetName("backend").
		SetRepositoryURL("https://github.com/GrandCX/openase.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create backend repo: %v", err)
	}
	backendAliasRepo, err := client.ProjectRepo.Create().
		SetProjectID(fixture.projectID).
		SetName("backend-alias").
		SetRepositoryURL("git@github.com:GrandCX/openase.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create backend alias repo: %v", err)
	}

	if _, err := service.SyncRepoScopePRStatus(ctx, SyncRepoScopePRStatusInput{
		RepositoryURL: backendRepo.RepositoryURL,
		BranchName:    "   ",
		PRStatus:      ticketreposcope.PrStatusMerged,
	}); err == nil || err.Error() != "branch name must not be empty" {
		t.Fatalf("SyncRepoScopePRStatus(empty branch) error = %v", err)
	}
	if _, err := service.SyncRepoScopePRStatus(ctx, SyncRepoScopePRStatusInput{
		RepositoryURL: "https://github.com/GrandCX",
		BranchName:    "agent/codex/invalid-repo",
		PRStatus:      ticketreposcope.PrStatusMerged,
	}); err == nil {
		t.Fatal("SyncRepoScopePRStatus(invalid repository) expected error")
	}

	ambiguousTicket, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Ambiguous repo scope match",
		StatusID:   &fixture.todoID,
		WorkflowID: &fixture.workflowID,
		Priority:   "high",
		Type:       "feature",
	})
	if err != nil {
		t.Fatalf("Create(ambiguousTicket) error = %v", err)
	}
	for _, repoID := range []uuid.UUID{backendRepo.ID, backendAliasRepo.ID} {
		if _, err := client.TicketRepoScope.Create().
			SetTicketID(ambiguousTicket.ID).
			SetRepoID(repoID).
			SetBranchName("agent/codex/ambiguous-ticket").
			SetPrStatus(ticketreposcope.PrStatusOpen).
			Save(ctx); err != nil {
			t.Fatalf("create ambiguous repo scope for %s: %v", repoID, err)
		}
	}
	if _, err := service.SyncRepoScopePRStatus(ctx, SyncRepoScopePRStatusInput{
		RepositoryFullName: "GrandCX/openase",
		BranchName:         "agent/codex/ambiguous-ticket",
		PRStatus:           ticketreposcope.PrStatusMerged,
	}); err == nil || err.Error() != `multiple ticket repo scopes matched repository "grandcx/openase" and branch "agent/codex/ambiguous-ticket"` {
		t.Fatalf("SyncRepoScopePRStatus(ambiguous) error = %v", err)
	}

	budgetRetryTicket, err := service.Create(ctx, CreateInput{
		ProjectID:  fixture.projectID,
		Title:      "Retry pauses for exhausted budget",
		StatusID:   &fixture.todoID,
		WorkflowID: &fixture.workflowID,
		Priority:   "high",
		Type:       "feature",
		BudgetUSD:  5,
	})
	if err != nil {
		t.Fatalf("Create(budgetRetryTicket) error = %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(budgetRetryTicket.ID).
		SetCostAmount(7).
		Save(ctx); err != nil {
		t.Fatalf("set exhausted budget cost: %v", err)
	}
	budgetRunID := seedTicketCurrentRun(t, ctx, client, fixture, budgetRetryTicket.ID)
	if _, err := client.TicketRepoScope.Create().
		SetTicketID(budgetRetryTicket.ID).
		SetRepoID(backendRepo.ID).
		SetBranchName("agent/codex/budget-retry").
		SetPrStatus(ticketreposcope.PrStatusOpen).
		Save(ctx); err != nil {
		t.Fatalf("create budget retry repo scope: %v", err)
	}

	budgetRetryResult, err := service.SyncRepoScopePRStatus(ctx, SyncRepoScopePRStatusInput{
		RepositoryURL: backendRepo.RepositoryURL,
		BranchName:    "agent/codex/budget-retry",
		PRStatus:      ticketreposcope.PrStatusClosed,
	})
	if err != nil {
		t.Fatalf("SyncRepoScopePRStatus(budget retry) error = %v", err)
	}
	if !budgetRetryResult.Matched || budgetRetryResult.Outcome != RepoScopePRStatusSyncOutcomeRetried || budgetRetryResult.Ticket == nil {
		t.Fatalf("SyncRepoScopePRStatus(budget retry) = %+v", budgetRetryResult)
	}
	if !budgetRetryResult.Ticket.RetryPaused || budgetRetryResult.Ticket.PauseReason != ticketing.PauseReasonBudgetExhausted.String() {
		t.Fatalf("budget retry ticket after sync = %+v", budgetRetryResult.Ticket)
	}
	budgetRunAfter, err := client.AgentRun.Get(ctx, budgetRunID)
	if err != nil {
		t.Fatalf("load budget retry run: %v", err)
	}
	if budgetRunAfter.Status != entagentrun.StatusErrored {
		t.Fatalf("budget retry run after sync = %+v", budgetRunAfter)
	}

	noWorkflowTicket, err := service.Create(ctx, CreateInput{
		ProjectID: fixture.projectID,
		Title:     "Missing workflow finish",
		StatusID:  &fixture.todoID,
		Priority:  "high",
		Type:      "feature",
	})
	if err != nil {
		t.Fatalf("Create(noWorkflowTicket) error = %v", err)
	}
	if _, err := client.TicketRepoScope.Create().
		SetTicketID(noWorkflowTicket.ID).
		SetRepoID(backendRepo.ID).
		SetBranchName("agent/codex/no-workflow-finish").
		SetPrStatus(ticketreposcope.PrStatusOpen).
		Save(ctx); err != nil {
		t.Fatalf("create missing workflow repo scope: %v", err)
	}
	if _, err := service.SyncRepoScopePRStatus(ctx, SyncRepoScopePRStatusInput{
		RepositoryURL: backendRepo.RepositoryURL,
		BranchName:    "agent/codex/no-workflow-finish",
		PRStatus:      ticketreposcope.PrStatusMerged,
	}); err == nil || err.Error() != "ticket "+noWorkflowTicket.ID.String()+" has no workflow to finish" {
		t.Fatalf("SyncRepoScopePRStatus(no workflow) error = %v", err)
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

func seedTicketServiceFixture(t *testing.T, ctx context.Context, client *ent.Client) ticketServiceFixture {
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

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	otherStatuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, otherProject.ID)
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

func seedTicketCurrentRun(t *testing.T, ctx context.Context, client *ent.Client, fixture ticketServiceFixture, ticketID uuid.UUID) uuid.UUID {
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
