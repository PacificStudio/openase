package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticketcomment "github.com/BetterAndBetterII/openase/ent/ticketcomment"
	entticketcommentrevision "github.com/BetterAndBetterII/openase/ent/ticketcommentrevision"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entticketexternallink "github.com/BetterAndBetterII/openase/ent/ticketexternallink"
	entticketrepoworkspace "github.com/BetterAndBetterII/openase/ent/ticketrepoworkspace"
	"github.com/BetterAndBetterII/openase/internal/config"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	"github.com/BetterAndBetterII/openase/internal/orchestrator"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	ticketstatusrepo "github.com/BetterAndBetterII/openase/internal/repo/ticketstatus"
	workflowrepo "github.com/BetterAndBetterII/openase/internal/repo/workflow"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func TestTicketRoutesCRUDAndDependencies(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	targetMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("gpu-01").
		SetHost("10.0.1.10").
		SetPort(22).
		SetSSHUser("openase").
		SetSSHKeyPath("/tmp/gpu-01.pem").
		SetDescription("GPU worker").
		Save(ctx)
	if err != nil {
		t.Fatalf("create target machine: %v", err)
	}

	statusSvc := newTicketStatusService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")
	doneID := findStatusIDByName(t, statuses, "Done")

	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("coding-workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(backlogID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	parentCreateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{
			"title":       "Implement ticket API",
			"description": "cover create/list/detail/update",
			"priority":    "high",
			"type":        "epic",
			"workflow_id": workflowItem.ID.String(),
			"created_by":  "user:gary",
			"budget_usd":  3.5,
		},
		http.StatusCreated,
		&parentCreateResp,
	)
	if parentCreateResp.Ticket.Identifier != "ASE-1" {
		t.Fatalf("expected first identifier ASE-1, got %+v", parentCreateResp.Ticket)
	}
	if parentCreateResp.Ticket.StatusName != "Backlog" || parentCreateResp.Ticket.CreatedBy != "user:gary" {
		t.Fatalf("unexpected parent create response: %+v", parentCreateResp.Ticket)
	}
	if parentCreateResp.Ticket.TargetMachineID != nil {
		t.Fatalf("expected manual machine dispatch hint to stay unset, got %+v", parentCreateResp.Ticket.TargetMachineID)
	}

	legacyCreateRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		fmt.Sprintf(`{"title":"Legacy machine dispatch","target_machine_id":"%s"}`, targetMachine.ID),
	)
	if legacyCreateRec.Code != http.StatusBadRequest ||
		!strings.Contains(legacyCreateRec.Body.String(), "invalid JSON body") ||
		!strings.Contains(legacyCreateRec.Body.String(), "target_machine_id") {
		t.Fatalf("expected legacy create target_machine_id to be rejected, got %d: %s", legacyCreateRec.Code, legacyCreateRec.Body.String())
	}

	childCreateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{
			"title":            "Implement dependency routes",
			"description":      "child ticket",
			"parent_ticket_id": parentCreateResp.Ticket.ID,
			"type":             "feature",
		},
		http.StatusCreated,
		&childCreateResp,
	)
	if childCreateResp.Ticket.Identifier != "ASE-2" || childCreateResp.Ticket.CreatedBy != "user:api" {
		t.Fatalf("unexpected child create response: %+v", childCreateResp.Ticket)
	}
	if childCreateResp.Ticket.Parent == nil || childCreateResp.Ticket.Parent.ID != parentCreateResp.Ticket.ID {
		t.Fatalf("expected child to point at parent, got %+v", childCreateResp.Ticket)
	}
	if len(childCreateResp.Ticket.Dependencies) != 1 || childCreateResp.Ticket.Dependencies[0].Type != "sub_issue" {
		t.Fatalf("expected child create to add sub_issue dependency, got %+v", childCreateResp.Ticket.Dependencies)
	}

	listResp := struct {
		Tickets []ticketResponse `json:"tickets"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/tickets?priority=high", project.ID),
		nil,
		http.StatusOK,
		&listResp,
	)
	if len(listResp.Tickets) != 1 || listResp.Tickets[0].ID != parentCreateResp.Ticket.ID {
		t.Fatalf("expected priority filter to return only parent, got %+v", listResp.Tickets)
	}

	parentDetailResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", parentCreateResp.Ticket.ID),
		nil,
		http.StatusOK,
		&parentDetailResp,
	)
	if len(parentDetailResp.Ticket.Children) != 1 || parentDetailResp.Ticket.Children[0].ID != childCreateResp.Ticket.ID {
		t.Fatalf("expected parent detail to expose child, got %+v", parentDetailResp.Ticket)
	}

	childUpdateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", childCreateResp.Ticket.ID),
		map[string]any{
			"title":            "Implement dependency HTTP routes",
			"priority":         "low",
			"status_id":        doneID.String(),
			"external_ref":     "PacificStudio/openase#6",
			"budget_usd":       1.25,
			"parent_ticket_id": "",
		},
		http.StatusOK,
		&childUpdateResp,
	)
	if childUpdateResp.Ticket.Parent != nil || len(childUpdateResp.Ticket.Dependencies) != 0 {
		t.Fatalf("expected patch to clear sub_issue link, got %+v", childUpdateResp.Ticket)
	}
	if childUpdateResp.Ticket.StatusID != doneID.String() || childUpdateResp.Ticket.ExternalRef != "PacificStudio/openase#6" {
		t.Fatalf("unexpected child patch response: %+v", childUpdateResp.Ticket)
	}
	if childUpdateResp.Ticket.TargetMachineID != nil {
		t.Fatalf("expected patch to clear target machine binding, got %+v", childUpdateResp.Ticket.TargetMachineID)
	}

	legacyPatchRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", childCreateResp.Ticket.ID),
		fmt.Sprintf(`{"target_machine_id":"%s"}`, targetMachine.ID),
	)
	if legacyPatchRec.Code != http.StatusBadRequest ||
		!strings.Contains(legacyPatchRec.Body.String(), "invalid JSON body") ||
		!strings.Contains(legacyPatchRec.Body.String(), "target_machine_id") {
		t.Fatalf("expected legacy patch target_machine_id to be rejected, got %d: %s", legacyPatchRec.Code, legacyPatchRec.Body.String())
	}

	peerCreateResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{"title": "Peer ticket"},
		http.StatusCreated,
		&peerCreateResp,
	)

	blockDependencyResp := struct {
		Dependency ticketDependencyResponse `json:"dependency"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/dependencies", parentCreateResp.Ticket.ID),
		map[string]any{
			"target_ticket_id": peerCreateResp.Ticket.ID,
			"type":             "blocks",
		},
		http.StatusCreated,
		&blockDependencyResp,
	)
	if blockDependencyResp.Dependency.Type != "blocks" || blockDependencyResp.Dependency.Target.ID != peerCreateResp.Ticket.ID {
		t.Fatalf("unexpected blocks dependency response: %+v", blockDependencyResp.Dependency)
	}

	parentAfterBlocksResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", parentCreateResp.Ticket.ID),
		nil,
		http.StatusOK,
		&parentAfterBlocksResp,
	)
	if len(parentAfterBlocksResp.Ticket.Dependencies) != 1 || parentAfterBlocksResp.Ticket.Dependencies[0].Type != "blocks" {
		t.Fatalf("expected parent detail to expose blocks dependency, got %+v", parentAfterBlocksResp.Ticket.Dependencies)
	}

	peerAfterBlocksResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", peerCreateResp.Ticket.ID),
		nil,
		http.StatusOK,
		&peerAfterBlocksResp,
	)
	if len(peerAfterBlocksResp.Ticket.Dependencies) != 1 || peerAfterBlocksResp.Ticket.Dependencies[0].Type != "blocked_by" || peerAfterBlocksResp.Ticket.Dependencies[0].Target.ID != parentCreateResp.Ticket.ID {
		t.Fatalf("expected peer detail to expose blocked_by relationship, got %+v", peerAfterBlocksResp.Ticket.Dependencies)
	}

	deleteBlocksResp := ticketservice.DeleteDependencyResult{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/tickets/%s/dependencies/%s", peerCreateResp.Ticket.ID, blockDependencyResp.Dependency.ID),
		nil,
		http.StatusOK,
		&deleteBlocksResp,
	)
	if deleteBlocksResp.DeletedDependencyID.String() != blockDependencyResp.Dependency.ID {
		t.Fatalf("unexpected inbound blocks delete response: %+v", deleteBlocksResp)
	}

	blockedByResp := struct {
		Dependency ticketDependencyResponse `json:"dependency"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/dependencies", peerCreateResp.Ticket.ID),
		map[string]any{
			"target_ticket_id": parentCreateResp.Ticket.ID,
			"type":             "blocked_by",
		},
		http.StatusCreated,
		&blockedByResp,
	)
	if blockedByResp.Dependency.Type != "blocked_by" || blockedByResp.Dependency.Target.ID != parentCreateResp.Ticket.ID {
		t.Fatalf("unexpected blocked_by dependency response: %+v", blockedByResp.Dependency)
	}

	commentCreateResp := struct {
		Comment ticketCommentResponse `json:"comment"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/comments", parentCreateResp.Ticket.ID),
		map[string]any{
			"body":       "Needs a second pass on the API response shape.",
			"created_by": "user:reviewer",
		},
		http.StatusCreated,
		&commentCreateResp,
	)
	if commentCreateResp.Comment.CreatedBy != "user:reviewer" || commentCreateResp.Comment.BodyMarkdown == "" {
		t.Fatalf("unexpected comment create response: %+v", commentCreateResp.Comment)
	}

	commentUpdateResp := struct {
		Comment ticketCommentResponse `json:"comment"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf(
			"/api/v1/tickets/%s/comments/%s",
			parentCreateResp.Ticket.ID,
			commentCreateResp.Comment.ID,
		),
		map[string]any{
			"body": "Needs a second pass on the API response shape and markdown support.",
		},
		http.StatusOK,
		&commentUpdateResp,
	)
	if !strings.Contains(commentUpdateResp.Comment.BodyMarkdown, "markdown support") {
		t.Fatalf("unexpected comment update response: %+v", commentUpdateResp.Comment)
	}

	commentDeleteResp := struct {
		DeletedCommentID string `json:"deleted_comment_id"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf(
			"/api/v1/tickets/%s/comments/%s",
			parentCreateResp.Ticket.ID,
			commentCreateResp.Comment.ID,
		),
		nil,
		http.StatusOK,
		&commentDeleteResp,
	)
	if commentDeleteResp.DeletedCommentID != commentCreateResp.Comment.ID {
		t.Fatalf("unexpected comment delete response: %+v", commentDeleteResp)
	}

	subIssueDependencyResp := struct {
		Dependency ticketDependencyResponse `json:"dependency"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/dependencies", peerCreateResp.Ticket.ID),
		map[string]any{
			"target_ticket_id": parentCreateResp.Ticket.ID,
			"type":             "sub_issue",
		},
		http.StatusCreated,
		&subIssueDependencyResp,
	)
	if subIssueDependencyResp.Dependency.Type != "sub_issue" {
		t.Fatalf("unexpected sub_issue dependency response: %+v", subIssueDependencyResp.Dependency)
	}

	peerAfterSubIssueResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", peerCreateResp.Ticket.ID),
		nil,
		http.StatusOK,
		&peerAfterSubIssueResp,
	)
	if peerAfterSubIssueResp.Ticket.Parent == nil || peerAfterSubIssueResp.Ticket.Parent.ID != parentCreateResp.Ticket.ID {
		t.Fatalf("expected peer parent to be synced from sub_issue dependency, got %+v", peerAfterSubIssueResp.Ticket)
	}

	deleteResp := ticketservice.DeleteDependencyResult{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/tickets/%s/dependencies/%s", peerCreateResp.Ticket.ID, subIssueDependencyResp.Dependency.ID),
		nil,
		http.StatusOK,
		&deleteResp,
	)
	if deleteResp.DeletedDependencyID.String() != subIssueDependencyResp.Dependency.ID {
		t.Fatalf("unexpected dependency delete response: %+v", deleteResp)
	}

	peerAfterDeleteResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", peerCreateResp.Ticket.ID),
		nil,
		http.StatusOK,
		&peerAfterDeleteResp,
	)
	if peerAfterDeleteResp.Ticket.Parent != nil {
		t.Fatalf("expected sub_issue delete to clear parent, got %+v", peerAfterDeleteResp.Ticket)
	}
}

func TestTicketRoutesExposeStartedAndCompletedTimestamps(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-started-at").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE StartedAt").
		SetSlug("openase-started-at").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statusSvc := newTicketStatusService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")

	startedAt := time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(15 * time.Minute)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Expose started timestamps").
		SetStatusID(backlogID).
		SetCreatedBy("user:test").
		SetStartedAt(startedAt).
		SetCompletedAt(completedAt).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	responseBody := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		nil,
		http.StatusOK,
		&responseBody,
	)
	if responseBody.Ticket.StartedAt == nil || *responseBody.Ticket.StartedAt != startedAt.Format(time.RFC3339) {
		t.Fatalf("expected started_at %s, got %+v", startedAt.Format(time.RFC3339), responseBody.Ticket.StartedAt)
	}
	if responseBody.Ticket.CompletedAt == nil || *responseBody.Ticket.CompletedAt != completedAt.Format(time.RFC3339) {
		t.Fatalf("expected completed_at %s, got %+v", completedAt.Format(time.RFC3339), responseBody.Ticket.CompletedAt)
	}
}

func TestTicketListExposesBlockedByRelationships(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40031},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-ticket-list-blocked").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-ticket-list-blocked").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statusSvc := newTicketStatusService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doingID := findStatusIDByName(t, statuses, "In Review")

	blocker, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Blocking prerequisite").
		SetStatusID(doingID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create blocker ticket: %v", err)
	}
	target, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-2").
		SetTitle("Blocked target").
		SetStatusID(todoID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create target ticket: %v", err)
	}
	if _, err := client.TicketDependency.Create().
		SetSourceTicketID(blocker.ID).
		SetTargetTicketID(target.ID).
		SetType(entticketdependency.TypeBlocks).
		Save(ctx); err != nil {
		t.Fatalf("create dependency: %v", err)
	}

	responseBody := struct {
		Tickets []ticketResponse `json:"tickets"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		nil,
		http.StatusOK,
		&responseBody,
	)

	var targetTicket *ticketResponse
	for index := range responseBody.Tickets {
		if responseBody.Tickets[index].ID == target.ID.String() {
			targetTicket = &responseBody.Tickets[index]
			break
		}
	}
	if targetTicket == nil {
		t.Fatalf("target ticket missing from list response: %+v", responseBody.Tickets)
	}
	if len(targetTicket.Dependencies) != 1 || targetTicket.Dependencies[0].Type != "blocked_by" || targetTicket.Dependencies[0].Target.ID != blocker.ID.String() {
		t.Fatalf("expected blocked_by dependency in ticket list response, got %+v", targetTicket.Dependencies)
	}
}

func TestListArchivedTicketsSupportsPagination(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40032},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-archived-pagination").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Archived").
		SetSlug("openase-archived-pagination").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statusSvc := newTicketStatusService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")
	cancelledID := findStatusIDByName(t, statuses, "Cancelled")
	archivedStatus, err := statusSvc.Create(ctx, ticketstatus.CreateInput{
		ProjectID: project.ID,
		Name:      "Archived",
		Stage:     ticketing.StatusStageCanceled,
		Color:     "#374151",
		Position:  ticketstatus.Some(len(statuses)),
	})
	if err != nil {
		t.Fatalf("create archived status: %v", err)
	}

	for index := range 5 {
		builder := client.Ticket.Create().
			SetProjectID(project.ID).
			SetIdentifier(fmt.Sprintf("ASE-%d", index+1)).
			SetTitle(fmt.Sprintf("Archived ticket %d", index+1)).
			SetStatusID(archivedStatus.ID).
			SetArchived(true).
			SetCreatedBy("user:test").
			SetCreatedAt(time.Date(2026, 4, 1, 10, index, 0, 0, time.UTC))
		completedAt := time.Date(2026, 4, 2, 10, index, 0, 0, time.UTC)
		builder.SetCompletedAt(completedAt)
		if _, err := builder.Save(ctx); err != nil {
			t.Fatalf("create archived ticket %d: %v", index+1, err)
		}
	}
	if _, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-6").
		SetTitle("Active ticket").
		SetStatusID(backlogID).
		SetCreatedBy("user:test").
		SetCreatedAt(time.Date(2026, 4, 1, 10, 6, 0, 0, time.UTC)).
		Save(ctx); err != nil {
		t.Fatalf("create active ticket: %v", err)
	}
	if _, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-7").
		SetTitle("Cancelled ticket is not archived").
		SetStatusID(cancelledID).
		SetCreatedBy("user:test").
		SetCreatedAt(time.Date(2026, 4, 1, 10, 7, 0, 0, time.UTC)).
		Save(ctx); err != nil {
		t.Fatalf("create cancelled ticket: %v", err)
	}

	responseBody := archivedTicketsResponse{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/tickets/archived?page=2&per_page=2", project.ID),
		nil,
		http.StatusOK,
		&responseBody,
	)

	if responseBody.Total != 5 {
		t.Fatalf("expected total 5 archived tickets, got %+v", responseBody)
	}
	if responseBody.Page != 2 || responseBody.PerPage != 2 {
		t.Fatalf("expected pagination metadata page=2 per_page=2, got %+v", responseBody)
	}
	if len(responseBody.Tickets) != 2 {
		t.Fatalf("expected 2 archived tickets on page 2, got %+v", responseBody.Tickets)
	}
	if responseBody.Tickets[0].Identifier != "ASE-3" || responseBody.Tickets[1].Identifier != "ASE-4" {
		t.Fatalf("expected second page tickets ASE-3 and ASE-4, got %+v", responseBody.Tickets)
	}
	for _, ticket := range responseBody.Tickets {
		if ticket.StatusName != "Archived" {
			t.Fatalf("expected only archived tickets, got %+v", responseBody.Tickets)
		}
	}
}

func TestListArchivedTicketsRejectsInvalidPagination(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40033},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-archived-invalid").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Archived Invalid").
		SetSlug("openase-archived-invalid").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	recorder := performJSONRequest(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/tickets/archived?page=0&per_page=2", project.ID),
		"",
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid archived pagination, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "page must be greater than zero") {
		t.Fatalf("expected page validation error, got %s", recorder.Body.String())
	}
}

func TestTicketRoutesExternalLinks(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40027},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-external-links").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-external-links").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Implement ticket external links").
		SetStatusID(backlogID).
		SetCreatedBy("user:codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	firstLinkResp := struct {
		ExternalLink ticketExternalLinkResponse `json:"external_link"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/external-links", ticketItem.ID),
		map[string]any{
			"type":        "github_issue",
			"url":         "https://github.com/PacificStudio/openase/issues/99",
			"external_id": "PacificStudio/openase#99",
			"title":       "F57: TicketExternalLink",
			"status":      "open",
			"relation":    "related",
		},
		http.StatusCreated,
		&firstLinkResp,
	)
	if firstLinkResp.ExternalLink.Type != "github_issue" || firstLinkResp.ExternalLink.ExternalID != "PacificStudio/openase#99" {
		t.Fatalf("unexpected first external link response: %+v", firstLinkResp.ExternalLink)
	}

	secondLinkResp := struct {
		ExternalLink ticketExternalLinkResponse `json:"external_link"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/external-links", ticketItem.ID),
		map[string]any{
			"type":        "github_issue",
			"url":         "https://github.com/PacificStudio/openase/issues/6",
			"external_id": "PacificStudio/openase#6",
			"title":       "F06: Ticket CRUD + 依赖关系",
			"status":      "open",
			"relation":    "caused_by",
		},
		http.StatusCreated,
		&secondLinkResp,
	)
	if secondLinkResp.ExternalLink.Relation != "caused_by" {
		t.Fatalf("unexpected second external link response: %+v", secondLinkResp.ExternalLink)
	}

	duplicateRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/external-links", ticketItem.ID),
		`{"type":"github_issue","url":"https://github.com/PacificStudio/openase/issues/99","external_id":"PacificStudio/openase#99"}`,
	)
	if duplicateRec.Code != http.StatusConflict || !strings.Contains(duplicateRec.Body.String(), "EXTERNAL_LINK_CONFLICT") {
		t.Fatalf("expected duplicate external link conflict, got %d: %s", duplicateRec.Code, duplicateRec.Body.String())
	}

	getResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		nil,
		http.StatusOK,
		&getResp,
	)
	if getResp.Ticket.ExternalRef != "PacificStudio/openase#99" {
		t.Fatalf("expected first external link to seed external_ref, got %+v", getResp.Ticket)
	}
	if len(getResp.Ticket.ExternalLinks) != 2 {
		t.Fatalf("expected ticket get response to include two external links, got %+v", getResp.Ticket.ExternalLinks)
	}

	deleteFirstResp := ticketservice.DeleteExternalLinkResult{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/tickets/%s/external-links/%s", ticketItem.ID, firstLinkResp.ExternalLink.ID),
		nil,
		http.StatusOK,
		&deleteFirstResp,
	)
	if deleteFirstResp.DeletedExternalLinkID.String() != firstLinkResp.ExternalLink.ID {
		t.Fatalf("unexpected delete first external link response: %+v", deleteFirstResp)
	}

	afterFirstDeleteResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		nil,
		http.StatusOK,
		&afterFirstDeleteResp,
	)
	if afterFirstDeleteResp.Ticket.ExternalRef != "PacificStudio/openase#6" {
		t.Fatalf("expected external_ref to fall back to remaining link, got %+v", afterFirstDeleteResp.Ticket)
	}
	if len(afterFirstDeleteResp.Ticket.ExternalLinks) != 1 || afterFirstDeleteResp.Ticket.ExternalLinks[0].ID != secondLinkResp.ExternalLink.ID {
		t.Fatalf("expected only second external link to remain, got %+v", afterFirstDeleteResp.Ticket.ExternalLinks)
	}

	deleteSecondResp := ticketservice.DeleteExternalLinkResult{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/tickets/%s/external-links/%s", ticketItem.ID, secondLinkResp.ExternalLink.ID),
		nil,
		http.StatusOK,
		&deleteSecondResp,
	)
	if deleteSecondResp.DeletedExternalLinkID.String() != secondLinkResp.ExternalLink.ID {
		t.Fatalf("unexpected delete second external link response: %+v", deleteSecondResp)
	}

	afterSecondDeleteResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		nil,
		http.StatusOK,
		&afterSecondDeleteResp,
	)
	if afterSecondDeleteResp.Ticket.ExternalRef != "" || len(afterSecondDeleteResp.Ticket.ExternalLinks) != 0 {
		t.Fatalf("expected all external links cleared after second delete, got %+v", afterSecondDeleteResp.Ticket)
	}
}

func TestTicketMutationRoutesPublishAffectedTicketRefreshEvents(t *testing.T) {
	client := openTestEntClient(t)
	bus := eventinfra.NewChannelBus()
	server := NewServer(
		config.ServerConfig{Port: 40027},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better Ticket Events").
		SetSlug("better-and-better-ticket-events").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Ticket Events").
		SetSlug("openase-ticket-events").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID); err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}

	stream := subscribeTopicEvents(t, bus, ticketEventsTopic)

	parentResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{"title": "Parent ticket"},
		http.StatusCreated,
		&parentResp,
	)
	assertStringSet(t, readTicketEventTicketIDs(t, stream, 1), parentResp.Ticket.ID)

	childResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{
			"title":            "Child ticket",
			"parent_ticket_id": parentResp.Ticket.ID,
		},
		http.StatusCreated,
		&childResp,
	)
	assertStringSet(t, readTicketEventTicketIDs(t, stream, 2), childResp.Ticket.ID, parentResp.Ticket.ID)

	secondParentResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{"title": "Second parent"},
		http.StatusCreated,
		&secondParentResp,
	)
	assertStringSet(t, readTicketEventTicketIDs(t, stream, 1), secondParentResp.Ticket.ID)

	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", childResp.Ticket.ID),
		map[string]any{"parent_ticket_id": secondParentResp.Ticket.ID},
		http.StatusOK,
		nil,
	)
	assertStringSet(
		t,
		readTicketEventTicketIDs(t, stream, 3),
		childResp.Ticket.ID,
		parentResp.Ticket.ID,
		secondParentResp.Ticket.ID,
	)

	peerResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{"title": "Peer ticket"},
		http.StatusCreated,
		&peerResp,
	)
	assertStringSet(t, readTicketEventTicketIDs(t, stream, 1), peerResp.Ticket.ID)

	dependencyResp := struct {
		Dependency ticketDependencyResponse `json:"dependency"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/dependencies", secondParentResp.Ticket.ID),
		map[string]any{
			"target_ticket_id": peerResp.Ticket.ID,
			"type":             "blocks",
		},
		http.StatusCreated,
		&dependencyResp,
	)
	assertStringSet(
		t,
		readTicketEventTicketIDs(t, stream, 2),
		secondParentResp.Ticket.ID,
		peerResp.Ticket.ID,
	)

	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/tickets/%s/dependencies/%s", peerResp.Ticket.ID, dependencyResp.Dependency.ID),
		nil,
		http.StatusOK,
		nil,
	)
	assertStringSet(
		t,
		readTicketEventTicketIDs(t, stream, 2),
		secondParentResp.Ticket.ID,
		peerResp.Ticket.ID,
	)

	linkResp := struct {
		ExternalLink ticketExternalLinkResponse `json:"external_link"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/external-links", secondParentResp.Ticket.ID),
		map[string]any{
			"type":        "github_issue",
			"url":         "https://github.com/acme/openase/issues/7",
			"external_id": "acme/openase#7",
		},
		http.StatusCreated,
		&linkResp,
	)
	assertStringSet(t, readTicketEventTicketIDs(t, stream, 1), secondParentResp.Ticket.ID)

	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/tickets/%s/external-links/%s", secondParentResp.Ticket.ID, linkResp.ExternalLink.ID),
		nil,
		http.StatusOK,
		nil,
	)
	assertStringSet(t, readTicketEventTicketIDs(t, stream, 1), secondParentResp.Ticket.ID)
}

func TestListTicketsRouteReturnsEmptyArrayForNewProject(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	rec := performJSONRequest(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected ticket list 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"tickets":[]`) {
		t.Fatalf("expected empty tickets array in payload, got %s", rec.Body.String())
	}

	var payload struct {
		Tickets []ticketResponse `json:"tickets"`
	}
	decodeResponse(t, rec, &payload)
	if payload.Tickets == nil || len(payload.Tickets) != 0 {
		t.Fatalf("expected non-nil empty tickets slice, got %+v", payload.Tickets)
	}
}

func TestTicketRoutesErrorMappingsAndInvalidPayloads(t *testing.T) {
	serverWithoutService := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	validProjectID := uuid.NewString()
	validTicketID := uuid.NewString()
	validCommentID := uuid.NewString()
	validDependencyID := uuid.NewString()
	validExternalLinkID := uuid.NewString()

	for _, tc := range []struct {
		name   string
		method string
		target string
		body   string
	}{
		{name: "list tickets unavailable", method: http.MethodGet, target: "/api/v1/projects/" + validProjectID + "/tickets"},
		{name: "create ticket unavailable", method: http.MethodPost, target: "/api/v1/projects/" + validProjectID + "/tickets", body: `{"title":"x"}`},
		{name: "get ticket unavailable", method: http.MethodGet, target: "/api/v1/tickets/" + validTicketID},
		{name: "update ticket unavailable", method: http.MethodPatch, target: "/api/v1/tickets/" + validTicketID, body: `{"title":"x"}`},
		{name: "create comment unavailable", method: http.MethodPost, target: "/api/v1/tickets/" + validTicketID + "/comments", body: `{"body":"x"}`},
		{name: "update comment unavailable", method: http.MethodPatch, target: "/api/v1/tickets/" + validTicketID + "/comments/" + validCommentID, body: `{"body":"x"}`},
		{name: "delete comment unavailable", method: http.MethodDelete, target: "/api/v1/tickets/" + validTicketID + "/comments/" + validCommentID},
		{name: "add dependency unavailable", method: http.MethodPost, target: "/api/v1/tickets/" + validTicketID + "/dependencies", body: `{"target_ticket_id":"` + validTicketID + `","type":"blocks"}`},
		{name: "delete dependency unavailable", method: http.MethodDelete, target: "/api/v1/tickets/" + validTicketID + "/dependencies/" + validDependencyID},
		{name: "add external link unavailable", method: http.MethodPost, target: "/api/v1/tickets/" + validTicketID + "/external-links", body: `{"type":"github_issue","url":"https://example.com","external_id":"repo#1"}`},
		{name: "delete external link unavailable", method: http.MethodDelete, target: "/api/v1/tickets/" + validTicketID + "/external-links/" + validExternalLinkID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := performJSONRequest(t, serverWithoutService, tc.method, tc.target, tc.body)
			if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "SERVICE_UNAVAILABLE") {
				t.Fatalf("expected unavailable response, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}

	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)
	detailServer := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-ticket-errors").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Ticket Errors").
		SetSlug("openase-ticket-errors").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Ticket errors").
		SetStatusID(backlogID).
		SetCreatedBy("user:codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	comment, err := client.TicketComment.Create().
		SetTicketID(ticketItem.ID).
		SetBody("first comment").
		SetCreatedBy("user:codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	dependencyTarget, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-2").
		SetTitle("Ticket dependency target").
		SetStatusID(backlogID).
		SetCreatedBy("user:codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create dependency target: %v", err)
	}
	dependency, err := client.TicketDependency.Create().
		SetSourceTicketID(ticketItem.ID).
		SetTargetTicketID(dependencyTarget.ID).
		SetType(entticketdependency.TypeBlocks).
		Save(ctx)
	if err != nil {
		t.Fatalf("create dependency: %v", err)
	}
	externalLink, err := client.TicketExternalLink.Create().
		SetTicketID(ticketItem.ID).
		SetLinkType(entticketexternallink.LinkTypeGithubIssue).
		SetURL("https://github.com/PacificStudio/openase/issues/1").
		SetExternalID("PacificStudio/openase#1").
		SetRelation(entticketexternallink.RelationRelated).
		Save(ctx)
	if err != nil {
		t.Fatalf("create external link: %v", err)
	}

	for _, tc := range []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
		server     *Server
	}{
		{name: "list invalid project", method: http.MethodGet, target: "/api/v1/projects/not-a-uuid/tickets", wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID", server: server},
		{name: "list invalid priority", method: http.MethodGet, target: "/api/v1/projects/" + project.ID.String() + "/tickets?priority=unknown", wantStatus: http.StatusBadRequest, wantBody: "INVALID_PRIORITY", server: server},
		{name: "create invalid project", method: http.MethodPost, target: "/api/v1/projects/not-a-uuid/tickets", body: `{"title":"x"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID", server: server},
		{name: "create invalid json", method: http.MethodPost, target: "/api/v1/projects/" + project.ID.String() + "/tickets", body: `{"title":"x","extra":true}`, wantStatus: http.StatusBadRequest, wantBody: "invalid JSON body", server: server},
		{name: "get invalid ticket", method: http.MethodGet, target: "/api/v1/tickets/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "INVALID_TICKET_ID", server: server},
		{name: "get missing ticket", method: http.MethodGet, target: "/api/v1/tickets/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "TICKET_NOT_FOUND", server: server},
		{name: "detail unavailable", method: http.MethodGet, target: "/api/v1/projects/" + project.ID.String() + "/tickets/" + ticketItem.ID.String() + "/detail", wantStatus: http.StatusServiceUnavailable, wantBody: "SERVICE_UNAVAILABLE", server: detailServer},
		{name: "update invalid ticket", method: http.MethodPatch, target: "/api/v1/tickets/not-a-uuid", body: `{"title":"x"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_TICKET_ID", server: server},
		{name: "update invalid json", method: http.MethodPatch, target: "/api/v1/tickets/" + ticketItem.ID.String(), body: `{"title":"x","extra":true}`, wantStatus: http.StatusBadRequest, wantBody: "invalid JSON body", server: server},
		{name: "create comment invalid ticket", method: http.MethodPost, target: "/api/v1/tickets/not-a-uuid/comments", body: `{"body":"x","created_by":"user:codex"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_TICKET_ID", server: server},
		{name: "create comment invalid json", method: http.MethodPost, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/comments", body: `{"body":"x","created_by":"user:codex","extra":true}`, wantStatus: http.StatusBadRequest, wantBody: "invalid JSON body", server: server},
		{name: "create comment missing ticket", method: http.MethodPost, target: "/api/v1/tickets/" + uuid.NewString() + "/comments", body: `{"body":"x","created_by":"user:codex"}`, wantStatus: http.StatusNotFound, wantBody: "TICKET_NOT_FOUND", server: server},
		{name: "update comment invalid comment", method: http.MethodPatch, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/comments/not-a-uuid", body: `{"body":"x"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_COMMENT_ID", server: server},
		{name: "update comment missing", method: http.MethodPatch, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/comments/" + uuid.NewString(), body: `{"body":"x"}`, wantStatus: http.StatusNotFound, wantBody: "COMMENT_NOT_FOUND", server: server},
		{name: "delete comment invalid comment", method: http.MethodDelete, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/comments/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "INVALID_COMMENT_ID", server: server},
		{name: "delete comment missing", method: http.MethodDelete, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/comments/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "COMMENT_NOT_FOUND", server: server},
		{name: "add dependency invalid ticket", method: http.MethodPost, target: "/api/v1/tickets/not-a-uuid/dependencies", body: `{"target_ticket_id":"` + dependencyTarget.ID.String() + `","type":"blocks"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_TICKET_ID", server: server},
		{name: "add dependency invalid request", method: http.MethodPost, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/dependencies", body: `{"target_ticket_id":"not-a-uuid","type":"blocks"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST", server: server},
		{name: "delete dependency invalid dependency", method: http.MethodDelete, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/dependencies/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "INVALID_DEPENDENCY_ID", server: server},
		{name: "delete dependency missing", method: http.MethodDelete, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/dependencies/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "DEPENDENCY_NOT_FOUND", server: server},
		{name: "add external link invalid ticket", method: http.MethodPost, target: "/api/v1/tickets/not-a-uuid/external-links", body: `{"type":"github_issue","url":"https://example.com","external_id":"repo#1"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_TICKET_ID", server: server},
		{name: "add external link invalid json", method: http.MethodPost, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/external-links", body: `{"type":"github_issue","url":"https://example.com","external_id":"repo#1","extra":true}`, wantStatus: http.StatusBadRequest, wantBody: "invalid JSON body", server: server},
		{name: "delete external link invalid id", method: http.MethodDelete, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/external-links/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "INVALID_EXTERNAL_LINK_ID", server: server},
		{name: "delete external link missing", method: http.MethodDelete, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/external-links/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "EXTERNAL_LINK_NOT_FOUND", server: server},
		{name: "delete dependency existing sanity", method: http.MethodDelete, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/dependencies/" + dependency.ID.String(), wantStatus: http.StatusOK, wantBody: dependency.ID.String(), server: server},
		{name: "delete external link existing sanity", method: http.MethodDelete, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/external-links/" + externalLink.ID.String(), wantStatus: http.StatusOK, wantBody: externalLink.ID.String(), server: server},
		{name: "delete comment existing sanity", method: http.MethodDelete, target: "/api/v1/tickets/" + ticketItem.ID.String() + "/comments/" + comment.ID.String(), wantStatus: http.StatusOK, wantBody: comment.ID.String(), server: server},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := performJSONRequest(t, tc.server, tc.method, tc.target, tc.body)
			if rec.Code != tc.wantStatus || !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("%s %s = %d %s", tc.method, tc.target, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestTicketRoutesCreateFirstTicketPerProjectAfterWorkflowCreate(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := createTestGitRepo(t)

	workflowSvc, err := workflowservice.NewService(workflowrepo.NewEntRepository(client), slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
	if err != nil {
		t.Fatalf("create workflow service: %v", err)
	}
	t.Cleanup(func() {
		if err := workflowSvc.Close(); err != nil {
			t.Errorf("close workflow service: %v", err)
		}
	})

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		workflowSvc,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-two-projects").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	for index := range 2 {
		project, err := client.Project.Create().
			SetOrganizationID(org.ID).
			SetName(fmt.Sprintf("OpenASE %d", index+1)).
			SetSlug(fmt.Sprintf("openase-%d", index+1)).
			Save(ctx)
		if err != nil {
			t.Fatalf("create project %d: %v", index+1, err)
		}
		createPrimaryProjectRepo(ctx, t, client, project.ID, repoRoot)
		localMachine, err := client.Machine.Create().
			SetOrganizationID(org.ID).
			SetName(fmt.Sprintf("local-%d", index+1)).
			SetHost("local").
			SetPort(22).
			SetStatus("online").
			Save(ctx)
		if err != nil {
			t.Fatalf("create local machine %d: %v", index+1, err)
		}
		attachPrimaryProjectRepoCheckout(ctx, t, client, project.ID, localMachine.ID, repoRoot)

		statuses := struct {
			Statuses []ticketstatus.Status `json:"statuses"`
		}{}
		executeJSON(
			t,
			server,
			http.MethodPost,
			fmt.Sprintf("/api/v1/projects/%s/statuses/reset", project.ID),
			nil,
			http.StatusOK,
			&statuses,
		)
		todoID := findStatusIDByName(t, statuses.Statuses, "Todo")
		doneID := findStatusIDByName(t, statuses.Statuses, "Done")
		provider, err := client.AgentProvider.Create().
			SetOrganizationID(org.ID).
			SetMachineID(localMachine.ID).
			SetName(fmt.Sprintf("Codex %d", index+1)).
			SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
			SetCliCommand("codex").
			SetModelName("gpt-5.4").
			Save(ctx)
		if err != nil {
			t.Fatalf("create provider %d: %v", index+1, err)
		}
		agent, err := client.Agent.Create().
			SetProviderID(provider.ID).
			SetProjectID(project.ID).
			SetName(fmt.Sprintf("codex-%d", index+1)).
			Save(ctx)
		if err != nil {
			t.Fatalf("create agent %d: %v", index+1, err)
		}

		workflowResp := struct {
			Workflow workflowResponse `json:"workflow"`
		}{}
		executeJSON(
			t,
			server,
			http.MethodPost,
			fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
			map[string]any{
				"agent_id":          agent.ID.String(),
				"name":              fmt.Sprintf("Coding Workflow %d", index+1),
				"type":              "coding",
				"pickup_status_ids": []string{todoID.String()},
				"finish_status_ids": []string{doneID.String()},
				"harness_content":   "# Coding\n",
			},
			http.StatusCreated,
			&workflowResp,
		)

		createResp := struct {
			Ticket ticketResponse `json:"ticket"`
		}{}
		executeJSON(
			t,
			server,
			http.MethodPost,
			fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
			map[string]any{
				"title":       fmt.Sprintf("Ticket %d", index+1),
				"priority":    "high",
				"workflow_id": workflowResp.Workflow.ID,
				"created_by":  "user:blackbox",
			},
			http.StatusCreated,
			&createResp,
		)

		if createResp.Ticket.Identifier != "ASE-1" {
			t.Fatalf("expected first ticket in project %d to use ASE-1, got %+v", index+1, createResp.Ticket)
		}
		if createResp.Ticket.WorkflowID == nil || *createResp.Ticket.WorkflowID != workflowResp.Workflow.ID {
			t.Fatalf("expected created ticket to keep workflow reference for project %d, got %+v", index+1, createResp.Ticket)
		}
	}
}

func TestTicketDetailRouteIncludesRepoScopesAndTicketActivity(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40026},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")
	doneID := findStatusIDByName(t, statuses, "Done")
	detailBaseTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-9").
		SetTitle("Build ticket detail page").
		SetDescription("Expose PR status, activity, and hook history in one place.").
		SetStatusID(backlogID).
		SetCreatedBy("user:codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	backendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("backend").
		SetRepositoryURL("https://github.com/acme/backend.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create backend repo: %v", err)
	}
	frontendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("frontend").
		SetRepositoryURL("https://github.com/acme/frontend.git").
		SetDefaultBranch("develop").
		Save(ctx)
	if err != nil {
		t.Fatalf("create frontend repo: %v", err)
	}

	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(frontendRepo.ID).
		SetBranchName("agent/codex/ASE-9").
		SetPullRequestURL("https://github.com/acme/frontend/pull/9").
		SetPrStatus("open").
		SetCiStatus("pending").
		Save(ctx); err != nil {
		t.Fatalf("create frontend repo scope: %v", err)
	}
	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(backendRepo.ID).
		SetBranchName("main").
		SetPrStatus("approved").
		SetCiStatus("passing").
		Save(ctx); err != nil {
		t.Fatalf("create backend repo scope: %v", err)
	}

	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetEventType("ticket_comment.created").
		SetMessage("Looks good.\n\n- Please include PR links.").
		SetMetadata(map[string]any{"comment_author": "user:reviewer"}).
		SetCreatedAt(detailBaseTime.Add(2 * time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("create comment event: %v", err)
	}
	prOpenedEvent, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetEventType("pr.opened").
		SetMessage("Opened frontend PR #9").
		SetMetadata(map[string]any{"stream": "stdout"}).
		SetCreatedAt(detailBaseTime.Add(3 * time.Minute)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create activity event: %v", err)
	}
	hookFailedEvent, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetEventType("hook.failed").
		SetMessage("on_complete failed for run-tests.sh").
		SetMetadata(map[string]any{"hook_name": "on_complete", "command": "run-tests.sh"}).
		SetCreatedAt(detailBaseTime.Add(4 * time.Minute)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create hook event: %v", err)
	}
	if _, err := client.TicketExternalLink.Create().
		SetTicketID(ticketItem.ID).
		SetLinkType("github_issue").
		SetURL("https://github.com/acme/frontend/issues/9").
		SetExternalID("acme/frontend#9").
		SetTitle("Add ticket drawer PR metadata").
		SetStatus("open").
		SetRelation("related").
		Save(ctx); err != nil {
		t.Fatalf("create ticket external link: %v", err)
	}
	commentItem, err := client.TicketComment.Create().
		SetTicketID(ticketItem.ID).
		SetBody("Please split runtime hooks from discussion comments, then add revisions.").
		SetCreatedBy("user:product").
		SetCreatedAt(detailBaseTime.Add(time.Minute)).
		SetUpdatedAt(detailBaseTime.Add(5 * time.Minute)).
		SetEditedAt(detailBaseTime.Add(5 * time.Minute)).
		SetEditCount(1).
		SetLastEditedBy("agent:codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket comment: %v", err)
	}
	if _, err := client.TicketCommentRevision.Create().
		SetCommentID(commentItem.ID).
		SetRevisionNumber(1).
		SetBodyMarkdown("Please split runtime hooks from discussion comments.").
		SetEditedBy("user:product").
		SetEditedAt(detailBaseTime.Add(time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("create first ticket comment revision: %v", err)
	}
	if _, err := client.TicketCommentRevision.Create().
		SetCommentID(commentItem.ID).
		SetRevisionNumber(2).
		SetBodyMarkdown("Please split runtime hooks from discussion comments, then add revisions.").
		SetEditedBy("agent:codex").
		SetEditedAt(detailBaseTime.Add(5 * time.Minute)).
		SetEditReason("clarified scope").
		Save(ctx); err != nil {
		t.Fatalf("create second ticket comment revision: %v", err)
	}
	agentProvider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("codex-cloud").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent provider: %v", err)
	}
	assignedAgent, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(agentProvider.ID).
		SetName("todo-app-coding-01").
		SetRuntimeControlState(entagent.RuntimeControlStateActive).
		Save(ctx)
	if err != nil {
		t.Fatalf("create assigned agent: %v", err)
	}
	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetAgentID(assignedAgent.ID).
		SetName("Todo App Coding Workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(backlogID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	runtimeStartedAt := time.Now().UTC().Truncate(time.Second)
	runItem, err := client.AgentRun.Create().
		SetAgentID(assignedAgent.ID).
		SetWorkflowID(workflowItem.ID).
		SetTicketID(ticketItem.ID).
		SetProviderID(agentProvider.ID).
		SetStatus(entagentrun.StatusExecuting).
		SetRuntimeStartedAt(runtimeStartedAt).
		SetLastHeartbeatAt(runtimeStartedAt).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetCurrentRunID(runItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("attach current run to ticket: %v", err)
	}

	var payload struct {
		AssignedAgent   *ticketAssignedAgentResponse    `json:"assigned_agent"`
		PickupDiagnosis ticketPickupDiagnosisResponse   `json:"pickup_diagnosis"`
		Ticket          ticketResponse                  `json:"ticket"`
		RepoScopes      []ticketRepoScopeDetailResponse `json:"repo_scopes"`
		Comments        []ticketCommentResponse         `json:"comments"`
		Timeline        []ticketTimelineItemResponse    `json:"timeline"`
		Activity        []activityEventResponse         `json:"activity"`
		HookHistory     []activityEventResponse         `json:"hook_history"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/tickets/%s/detail", project.ID, ticketItem.ID),
		nil,
		http.StatusOK,
		&payload,
	)

	if payload.Ticket.ID != ticketItem.ID.String() || payload.Ticket.Identifier != "ASE-9" {
		t.Fatalf("unexpected ticket payload: %+v", payload.Ticket)
	}
	if payload.AssignedAgent == nil {
		t.Fatalf("expected ticket detail to include assigned agent")
	}
	if payload.AssignedAgent.ID != assignedAgent.ID.String() || payload.AssignedAgent.Name != "todo-app-coding-01" {
		t.Fatalf("unexpected assigned agent payload: %+v", payload.AssignedAgent)
	}
	if payload.AssignedAgent.Provider != "codex-cloud" || payload.AssignedAgent.RuntimeControlState != "active" {
		t.Fatalf("expected provider-backed assigned agent details, got %+v", payload.AssignedAgent)
	}
	if payload.AssignedAgent.RuntimePhase == nil || *payload.AssignedAgent.RuntimePhase != "executing" {
		t.Fatalf("expected assigned agent runtime phase executing, got %+v", payload.AssignedAgent)
	}
	if payload.PickupDiagnosis.PrimaryReasonCode != "running_current_run" || payload.PickupDiagnosis.State != "running" {
		t.Fatalf("expected pickup diagnosis to expose running current run, got %+v", payload.PickupDiagnosis)
	}
	if len(payload.Ticket.ExternalLinks) != 1 || payload.Ticket.ExternalLinks[0].ExternalID != "acme/frontend#9" {
		t.Fatalf("expected ticket detail to include external links, got %+v", payload.Ticket.ExternalLinks)
	}
	if len(payload.RepoScopes) != 2 {
		t.Fatalf("expected repo scopes with repo metadata, got %+v", payload.RepoScopes)
	}
	repoScopesByName := map[string]ticketRepoScopeDetailResponse{}
	for _, scope := range payload.RepoScopes {
		if scope.Repo == nil {
			t.Fatalf("expected repo scope metadata, got %+v", payload.RepoScopes)
		}
		repoScopesByName[scope.Repo.Name] = scope
	}
	frontendScope, ok := repoScopesByName["frontend"]
	if !ok || frontendScope.PullRequestURL == nil || *frontendScope.PullRequestURL != "https://github.com/acme/frontend/pull/9" {
		t.Fatalf("expected frontend pull request URL, got %+v", payload.RepoScopes)
	}
	if frontendScope.DefaultBranch != "develop" || frontendScope.EffectiveBranchName != "agent/codex/ASE-9" || frontendScope.BranchSource != "override" {
		t.Fatalf("expected frontend repo scope branch metadata, got %+v", frontendScope)
	}
	backendScope, ok := repoScopesByName["backend"]
	if !ok || backendScope.DefaultBranch != "main" || backendScope.EffectiveBranchName != "main" || backendScope.BranchSource != "override" {
		t.Fatalf("expected backend repo scope branch metadata, got %+v", backendScope)
	}
	if len(payload.Comments) != 1 || payload.Comments[0].CreatedBy != "user:product" || payload.Comments[0].EditCount != 1 || payload.Comments[0].EditedAt == nil {
		t.Fatalf("expected ticket detail to include comments, got %+v", payload.Comments)
	}
	if len(payload.Timeline) != 4 || payload.Timeline[0].ItemType != "description" || payload.Timeline[1].ItemType != "comment" || payload.Timeline[2].ItemType != "activity" || payload.Timeline[3].ItemType != "activity" {
		t.Fatalf("expected ticket detail timeline projection, got %+v", payload.Timeline)
	}
	if payload.Timeline[0].ID != "description:"+ticketItem.ID.String() || payload.Timeline[0].Title == nil || *payload.Timeline[0].Title != ticketItem.Title {
		t.Fatalf("expected description timeline root to use ticket title and stable id, got %+v", payload.Timeline[0])
	}
	if payload.Timeline[1].ID != "comment:"+commentItem.ID.String() || payload.Timeline[2].ID != "activity:"+prOpenedEvent.ID.String() || payload.Timeline[3].ID != "activity:"+hookFailedEvent.ID.String() {
		t.Fatalf("expected mixed timeline items to use stable ids and created_at ordering, got %+v", payload.Timeline)
	}
	if payload.Timeline[1].Metadata["revision_count"] != float64(2) && payload.Timeline[1].Metadata["revision_count"] != 2 {
		t.Fatalf("expected comment timeline metadata to include revision count, got %+v", payload.Timeline[1].Metadata)
	}
	if len(payload.Activity) != 2 {
		t.Fatalf("expected two ticket activity events, got %+v", payload.Activity)
	}
	if len(payload.HookHistory) != 1 || payload.HookHistory[0].EventType != "hook.failed" {
		t.Fatalf("expected hook history to filter hook-tagged events, got %+v", payload.HookHistory)
	}
}

func TestBuildTicketTimelineKeepsDescriptionRootAndNormalizesActors(t *testing.T) {
	projectID := uuid.MustParse("00000000-0000-0000-0000-000000000111")
	ticketID := uuid.MustParse("00000000-0000-0000-0000-000000000222")
	activityEarlierID := uuid.MustParse("00000000-0000-0000-0000-000000000301")
	activitySameTimeID := uuid.MustParse("00000000-0000-0000-0000-000000000302")
	commentSameTimeID := uuid.MustParse("00000000-0000-0000-0000-000000000401")
	commentLaterID := uuid.MustParse("00000000-0000-0000-0000-000000000402")
	baseTime := time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC)

	ticketItem := ticketservice.Ticket{
		ID:          ticketID,
		ProjectID:   projectID,
		Identifier:  "ASE-333",
		Title:       "Build the ticket detail timeline projector",
		Description: "Project the root ticket description as the first timeline item.",
		CreatedBy:   "system_proxy:dispatcher",
		CreatedAt:   baseTime,
	}
	comments := []ticketservice.Comment{
		{
			ID:           commentLaterID,
			TicketID:     ticketID,
			BodyMarkdown: "Later discussion entry.",
			CreatedBy:    "user:zoe",
			CreatedAt:    baseTime.Add(3 * time.Minute),
			UpdatedAt:    baseTime.Add(3 * time.Minute),
		},
		{
			ID:           commentSameTimeID,
			TicketID:     ticketID,
			BodyMarkdown: "Same timestamp as an activity entry.",
			CreatedBy:    "user:alice",
			CreatedAt:    baseTime.Add(2 * time.Minute),
			UpdatedAt:    baseTime.Add(2 * time.Minute),
		},
	}
	activity := []catalogdomain.ActivityEvent{
		{
			ID:        activitySameTimeID,
			ProjectID: projectID,
			TicketID:  &ticketID,
			EventType: activityevent.TypeTicketStatusChanged,
			Message:   "Moved to In Progress.",
			Metadata:  map[string]any{"actor_name": "dispatcher"},
			CreatedAt: baseTime.Add(2 * time.Minute),
		},
		{
			ID:        activityEarlierID,
			ProjectID: projectID,
			TicketID:  &ticketID,
			EventType: activityevent.TypePROpened,
			Message:   "Opened PR #333.",
			Metadata:  map[string]any{"agent_name": "codex"},
			CreatedAt: baseTime.Add(time.Minute),
		},
	}

	timeline := buildTicketTimeline(ticketItem, comments, activity)
	if len(timeline) != 5 {
		t.Fatalf("expected five timeline items, got %+v", timeline)
	}
	if timeline[0].ID != "description:"+ticketID.String() || timeline[0].Title == nil || *timeline[0].Title != ticketItem.Title {
		t.Fatalf("expected root description timeline item, got %+v", timeline[0])
	}
	if timeline[0].ActorType != "system" || timeline[0].ActorName != "dispatcher" {
		t.Fatalf("expected system_proxy actor to normalize to system, got %+v", timeline[0])
	}

	wantIDs := []string{
		"description:" + ticketID.String(),
		"activity:" + activityEarlierID.String(),
		"activity:" + activitySameTimeID.String(),
		"comment:" + commentSameTimeID.String(),
		"comment:" + commentLaterID.String(),
	}
	for i, wantID := range wantIDs {
		if timeline[i].ID != wantID {
			t.Fatalf("timeline[%d] id = %q, want %q; full timeline=%+v", i, timeline[i].ID, wantID, timeline)
		}
	}
}

func TestTicketCommentRoutesCreateUpdateDelete(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40026},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-12").
		SetTitle("Add ticket comments").
		SetStatusID(backlogID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	var createPayload struct {
		Comment ticketCommentResponse `json:"comment"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/comments", ticketItem.ID),
		map[string]any{
			"body":       "First comment",
			"created_by": "user:reviewer",
		},
		http.StatusCreated,
		&createPayload,
	)

	if createPayload.Comment.BodyMarkdown != "First comment" || createPayload.Comment.CreatedBy != "user:reviewer" || createPayload.Comment.EditCount != 0 {
		t.Fatalf("unexpected created comment payload: %+v", createPayload.Comment)
	}

	var listPayload struct {
		Comments []ticketCommentResponse `json:"comments"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s/comments", ticketItem.ID),
		nil,
		http.StatusOK,
		&listPayload,
	)

	if len(listPayload.Comments) != 1 || listPayload.Comments[0].ID != createPayload.Comment.ID || listPayload.Comments[0].BodyMarkdown != "First comment" {
		t.Fatalf("unexpected comment list payload: %+v", listPayload.Comments)
	}

	var updatePayload struct {
		Comment ticketCommentResponse `json:"comment"`
	}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s/comments/%s", ticketItem.ID, createPayload.Comment.ID),
		map[string]any{
			"body":        "Updated comment body",
			"edited_by":   "agent:codex",
			"edit_reason": "clarified scope",
		},
		http.StatusOK,
		&updatePayload,
	)

	if updatePayload.Comment.BodyMarkdown != "Updated comment body" || updatePayload.Comment.UpdatedAt == nil || updatePayload.Comment.EditedAt == nil || updatePayload.Comment.EditCount != 1 || updatePayload.Comment.LastEditedBy == nil || *updatePayload.Comment.LastEditedBy != "agent:codex" {
		t.Fatalf("unexpected updated comment payload: %+v", updatePayload.Comment)
	}

	var revisionsPayload struct {
		Revisions []ticketCommentRevisionResponse `json:"revisions"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/tickets/%s/comments/%s/revisions", ticketItem.ID, createPayload.Comment.ID),
		nil,
		http.StatusOK,
		&revisionsPayload,
	)

	if len(revisionsPayload.Revisions) != 2 || revisionsPayload.Revisions[0].RevisionNumber != 1 || revisionsPayload.Revisions[0].BodyMarkdown != "First comment" || revisionsPayload.Revisions[1].RevisionNumber != 2 || revisionsPayload.Revisions[1].BodyMarkdown != "Updated comment body" {
		t.Fatalf("unexpected revisions payload: %+v", revisionsPayload.Revisions)
	}

	var deletePayload struct {
		DeletedCommentID string `json:"deleted_comment_id"`
	}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/tickets/%s/comments/%s", ticketItem.ID, createPayload.Comment.ID),
		nil,
		http.StatusOK,
		&deletePayload,
	)

	if deletePayload.DeletedCommentID != createPayload.Comment.ID {
		t.Fatalf("unexpected deleted comment payload: %+v", deletePayload)
	}

	remaining, err := client.TicketComment.Query().
		Where(entticketcomment.TicketIDEQ(ticketItem.ID)).
		Count(ctx)
	if err != nil {
		t.Fatalf("count remaining comments: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected comment row to be soft deleted, remaining=%d", remaining)
	}
	deletedComment, err := client.TicketComment.Get(ctx, uuid.MustParse(createPayload.Comment.ID))
	if err != nil {
		t.Fatalf("get deleted comment: %v", err)
	}
	if !deletedComment.IsDeleted || deletedComment.DeletedAt == nil {
		t.Fatalf("expected comment to be soft deleted, got %+v", deletedComment)
	}
	revisionCount, err := client.TicketCommentRevision.Query().
		Where(entticketcommentrevision.CommentIDEQ(deletedComment.ID)).
		Count(ctx)
	if err != nil {
		t.Fatalf("count comment revisions: %v", err)
	}
	if revisionCount != 2 {
		t.Fatalf("expected revisions to survive delete, got %d", revisionCount)
	}
}

func TestTicketRouteStatusChangeRetainsPickupAssignmentAndReleasesOnFinish(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40024},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("coding-workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	assignedAgent, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(provider.ID).
		SetName("coding-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Implement pickup/finish state transitions").
		SetStatusID(todoID).
		SetWorkflowID(workflowItem.ID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runStartedAt := time.Now().UTC().Truncate(time.Second)
	runItem, err := client.AgentRun.Create().
		SetAgentID(assignedAgent.ID).
		SetWorkflowID(workflowItem.ID).
		SetTicketID(ticketItem.ID).
		SetProviderID(provider.ID).
		SetStatus(entagentrun.StatusReady).
		SetSessionID("session-ready").
		SetRuntimeStartedAt(runStartedAt).
		SetLastHeartbeatAt(runStartedAt).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetCurrentRunID(runItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("attach current run to ticket: %v", err)
	}

	statusesBeforeResp := ticketstatus.ListResult{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID),
		nil,
		http.StatusOK,
		&statusesBeforeResp,
	)
	todoStatusBefore := ticketstatus.Status{}
	for _, status := range statusesBeforeResp.Statuses {
		if status.Name == "Todo" {
			todoStatusBefore = status
			break
		}
	}
	if todoStatusBefore.ID == uuid.Nil || todoStatusBefore.ActiveRuns != 1 {
		t.Fatalf("expected Todo status active_runs=1 before status transition, got %+v", todoStatusBefore)
	}

	titleOnlyResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		map[string]any{"title": "Implement ticket pickup/finish transitions"},
		http.StatusOK,
		&titleOnlyResp,
	)

	ticketAfterTitleOnly, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after title update: %v", err)
	}
	if ticketAfterTitleOnly.CurrentRunID == nil || *ticketAfterTitleOnly.CurrentRunID != runItem.ID {
		t.Fatalf("expected non-status update to keep current run, got %+v", ticketAfterTitleOnly.CurrentRunID)
	}
	if titleOnlyResp.Ticket.CurrentRunID == nil || *titleOnlyResp.Ticket.CurrentRunID != runItem.ID.String() {
		t.Fatalf("expected patch response to expose current_run_id %s, got %+v", runItem.ID, titleOnlyResp.Ticket.CurrentRunID)
	}
	runAfterTitleOnly, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run after title update: %v", err)
	}
	if runAfterTitleOnly.Status != entagentrun.StatusReady {
		t.Fatalf("expected non-status update to keep run ready, got %+v", runAfterTitleOnly)
	}
	if _, err := client.Workflow.UpdateOneID(workflowItem.ID).
		AddPickupStatusIDs(backlogID).
		Save(ctx); err != nil {
		t.Fatalf("add backlog pickup status: %v", err)
	}

	pickupStatusResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		map[string]any{"status_id": backlogID.String()},
		http.StatusOK,
		&pickupStatusResp,
	)

	ticketAfterPickupStatusChange, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after pickup status update: %v", err)
	}
	if ticketAfterPickupStatusChange.StatusID != backlogID {
		t.Fatalf("expected ticket status %s, got %s", backlogID, ticketAfterPickupStatusChange.StatusID)
	}
	if ticketAfterPickupStatusChange.CurrentRunID == nil || *ticketAfterPickupStatusChange.CurrentRunID != runItem.ID {
		t.Fatalf("expected pickup status update to keep current run, got %+v", ticketAfterPickupStatusChange.CurrentRunID)
	}
	if pickupStatusResp.Ticket.CurrentRunID == nil || *pickupStatusResp.Ticket.CurrentRunID != runItem.ID.String() {
		t.Fatalf("expected pickup status response to keep current_run_id %s, got %+v", runItem.ID, pickupStatusResp.Ticket.CurrentRunID)
	}
	runAfterPickupStatusChange, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run after pickup status update: %v", err)
	}
	if runAfterPickupStatusChange.Status != entagentrun.StatusReady ||
		runAfterPickupStatusChange.SessionID != "session-ready" ||
		runAfterPickupStatusChange.RuntimeStartedAt == nil ||
		runAfterPickupStatusChange.LastHeartbeatAt == nil {
		t.Fatalf("expected pickup status update to preserve run state, got %+v", runAfterPickupStatusChange)
	}

	statusResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		map[string]any{"status_id": doneID.String()},
		http.StatusOK,
		&statusResp,
	)

	ticketAfterStatusChange, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after status update: %v", err)
	}
	if ticketAfterStatusChange.StatusID != doneID {
		t.Fatalf("expected ticket status %s, got %s", doneID, ticketAfterStatusChange.StatusID)
	}
	if ticketAfterStatusChange.CurrentRunID != nil {
		t.Fatalf("expected status update to clear current run, got %+v", ticketAfterStatusChange.CurrentRunID)
	}
	if statusResp.Ticket.CurrentRunID != nil {
		t.Fatalf("expected status patch response to clear current_run_id, got %+v", statusResp.Ticket.CurrentRunID)
	}
	runAfterStatusChange, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run after status update: %v", err)
	}
	if runAfterStatusChange.Status != entagentrun.StatusTerminated ||
		runAfterStatusChange.SessionID != "" ||
		runAfterStatusChange.RuntimeStartedAt != nil ||
		runAfterStatusChange.LastHeartbeatAt != nil {
		t.Fatalf("expected status update to finalize agent run, got %+v", runAfterStatusChange)
	}
	agentAfterStatusChange, err := client.Agent.Get(ctx, assignedAgent.ID)
	if err != nil {
		t.Fatalf("reload agent after status update: %v", err)
	}
	if agentAfterStatusChange.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("expected status update to reset agent control state, got %+v", agentAfterStatusChange)
	}

	statusesAfterResp := ticketstatus.ListResult{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID),
		nil,
		http.StatusOK,
		&statusesAfterResp,
	)
	todoStatusAfter := ticketstatus.Status{}
	backlogStatusAfter := ticketstatus.Status{}
	for _, status := range statusesAfterResp.Statuses {
		switch status.Name {
		case "Todo":
			todoStatusAfter = status
		case "Backlog":
			backlogStatusAfter = status
		}
	}
	if todoStatusAfter.ID == uuid.Nil || todoStatusAfter.ActiveRuns != 0 {
		t.Fatalf("expected Todo status active_runs=0 after status transition, got %+v", todoStatusAfter)
	}
	if backlogStatusAfter.ID == uuid.Nil || backlogStatusAfter.ActiveRuns != 0 {
		t.Fatalf("expected Backlog status active_runs=0 after finish transition, got %+v", backlogStatusAfter)
	}
}

func TestTicketRouteArchivingClearsAssignmentAndReleasesAgent(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40024},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(ticketrepo.NewEntRepository(client)),
		ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client)),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-archived").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local").
		SetHost("local").
		SetPort(22).
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-archived").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	statusSvc := ticketstatus.NewService(ticketstatusrepo.NewEntRepository(client))
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("coding-workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	assignedAgent, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(provider.ID).
		SetName("coding-archiver").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Archive ticket from board").
		SetStatusID(todoID).
		SetWorkflowID(workflowItem.ID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runStartedAt := time.Now().UTC().Truncate(time.Second)
	runItem, err := client.AgentRun.Create().
		SetAgentID(assignedAgent.ID).
		SetWorkflowID(workflowItem.ID).
		SetTicketID(ticketItem.ID).
		SetProviderID(provider.ID).
		SetStatus(entagentrun.StatusReady).
		SetSessionID("session-archived").
		SetRuntimeStartedAt(runStartedAt).
		SetLastHeartbeatAt(runStartedAt).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetCurrentRunID(runItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind current run: %v", err)
	}

	statusResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		map[string]any{"archived": true},
		http.StatusOK,
		&statusResp,
	)

	ticketAfterStatusChange, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after status update: %v", err)
	}
	if !ticketAfterStatusChange.Archived {
		t.Fatalf("expected ticket archived=true after patch, got %+v", ticketAfterStatusChange)
	}
	if ticketAfterStatusChange.CurrentRunID != nil {
		t.Fatalf("expected archived flag update to clear current run, got %+v", ticketAfterStatusChange.CurrentRunID)
	}
	if statusResp.Ticket.CurrentRunID != nil || !statusResp.Ticket.Archived {
		t.Fatalf("unexpected archive patch response: %+v", statusResp.Ticket)
	}
	runAfterStatusChange, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run after status update: %v", err)
	}
	if runAfterStatusChange.Status != entagentrun.StatusTerminated ||
		runAfterStatusChange.SessionID != "" ||
		runAfterStatusChange.RuntimeStartedAt != nil ||
		runAfterStatusChange.LastHeartbeatAt != nil {
		t.Fatalf("expected archived flag update to finalize agent run, got %+v", runAfterStatusChange)
	}
	agentAfterStatusChange, err := client.Agent.Get(ctx, assignedAgent.ID)
	if err != nil {
		t.Fatalf("reload agent after status update: %v", err)
	}
	if agentAfterStatusChange.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("expected archived flag update to reset agent control state, got %+v", agentAfterStatusChange)
	}
}

func TestTicketRoutesPublishSSEEvents(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40025},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	doneID := findStatusIDByName(t, statuses, "Done")

	createResponse, cancelCreate := openSSERequest(t, testServer.URL+fmt.Sprintf("/api/v1/projects/%s/events/stream", project.ID))
	t.Cleanup(func() {
		if err := createResponse.Body.Close(); err != nil {
			t.Errorf("close create response body: %v", err)
		}
	})
	createPayload := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/tickets", project.ID),
		map[string]any{
			"title":       "Implement board realtime updates",
			"description": "publish ticket.created when a ticket is added",
		},
		http.StatusCreated,
		&createPayload,
	)
	createBody := readSSEBody(t, createResponse, cancelCreate)
	if !strings.Contains(createBody, "event: ticket.created\n") {
		t.Fatalf("expected ticket.created frame, got %q", createBody)
	}
	if !strings.Contains(createBody, createPayload.Ticket.Identifier) {
		t.Fatalf("expected created ticket identifier in SSE payload, got %q", createBody)
	}

	updateResponse, cancelUpdate := openSSERequest(t, testServer.URL+fmt.Sprintf("/api/v1/projects/%s/events/stream", project.ID))
	t.Cleanup(func() {
		if err := updateResponse.Body.Close(); err != nil {
			t.Errorf("close update response body: %v", err)
		}
	})
	updatePayload := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", createPayload.Ticket.ID),
		map[string]any{"status_id": doneID.String()},
		http.StatusOK,
		&updatePayload,
	)
	updateBody := readSSEBody(t, updateResponse, cancelUpdate)
	if !strings.Contains(updateBody, "event: ticket.status_changed\n") {
		t.Fatalf("expected ticket.status_changed frame, got %q", updateBody)
	}
	if !strings.Contains(updateBody, doneID.String()) {
		t.Fatalf("expected updated status id in SSE payload, got %q", updateBody)
	}
}

func TestTicketBudgetUpdatesSyncBudgetExhaustedPauseState(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Adjust retry budget").
		SetStatusID(todoID).
		SetBudgetUsd(5).
		SetCostAmount(5).
		SetRetryPaused(true).
		SetPauseReason(ticketing.PauseReasonBudgetExhausted.String()).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	increaseResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		map[string]any{"budget_usd": 8.0},
		http.StatusOK,
		&increaseResp,
	)

	if increaseResp.Ticket.BudgetUSD != 8 || increaseResp.Ticket.CostAmount != 5 {
		t.Fatalf("unexpected ticket budget fields after increase: %+v", increaseResp.Ticket)
	}
	if increaseResp.Ticket.RetryPaused || increaseResp.Ticket.PauseReason != "" {
		t.Fatalf("expected budget increase to resume retry, got %+v", increaseResp.Ticket)
	}

	ticketAfterIncrease, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after increase: %v", err)
	}
	if ticketAfterIncrease.RetryPaused || ticketAfterIncrease.PauseReason != "" {
		t.Fatalf("expected budget increase to clear budget pause, got %+v", ticketAfterIncrease)
	}

	decreaseResp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/tickets/%s", ticketItem.ID),
		map[string]any{"budget_usd": 4.0},
		http.StatusOK,
		&decreaseResp,
	)

	if !decreaseResp.Ticket.RetryPaused || decreaseResp.Ticket.PauseReason != ticketing.PauseReasonBudgetExhausted.String() {
		t.Fatalf("expected lowered budget to pause retry again, got %+v", decreaseResp.Ticket)
	}

	ticketAfterDecrease, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after decrease: %v", err)
	}
	if !ticketAfterDecrease.RetryPaused || ticketAfterDecrease.PauseReason != ticketing.PauseReasonBudgetExhausted.String() {
		t.Fatalf("expected lowered budget to persist budget pause, got %+v", ticketAfterDecrease)
	}
}

func TestHandleResumeTicketRetryClearsRepeatedStallPause(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Resume stalled retry").
		SetStatusID(todoID).
		SetRetryPaused(true).
		SetPauseReason(ticketing.PauseReasonRepeatedStalls.String()).
		SetStallCount(20).
		SetNextRetryAt(time.Now().UTC().Add(time.Minute)).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	resp := struct {
		Ticket ticketResponse `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/retry/resume", ticketItem.ID),
		nil,
		http.StatusOK,
		&resp,
	)

	if resp.Ticket.RetryPaused || resp.Ticket.PauseReason != "" || resp.Ticket.NextRetryAt != nil {
		t.Fatalf("expected resumed retry response, got %+v", resp.Ticket)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.RetryPaused || ticketAfter.PauseReason != "" || ticketAfter.StallCount != 0 || ticketAfter.NextRetryAt != nil {
		t.Fatalf("expected repeated stall pause to clear, got %+v", ticketAfter)
	}

	activityItems, err := client.ActivityEvent.Query().All(ctx)
	if err != nil {
		t.Fatalf("query activity: %v", err)
	}
	if len(activityItems) != 1 || activityItems[0].EventType != activityevent.TypeTicketRetryResumed.String() {
		t.Fatalf("expected retry resumed activity, got %+v", activityItems)
	}
}

func TestHandleResumeTicketRetryRejectsNonStalledPause(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")

	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Budget paused retry").
		SetStatusID(todoID).
		SetRetryPaused(true).
		SetPauseReason(ticketing.PauseReasonBudgetExhausted.String()).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/retry/resume", ticketItem.ID),
		"",
	)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "RETRY_RESUME_CONFLICT") {
		t.Fatalf("expected retry resume conflict, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleResetTicketWorkspaceCleansWorkspaceAndPublishesUpdate(t *testing.T) {
	client := openTestEntClient(t)
	bus := eventinfra.NewChannelBus()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
		WithTicketWorkspaceResetter(
			orchestrator.NewTicketWorkspaceResetService(
				client,
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				nil,
			),
		),
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local-devbox").
		SetHost(catalogdomain.LocalMachineHost).
		SetPort(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(providerItem.ID).
		SetName("coder").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("coding-workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Preserve dirty worktree").
		SetStatusID(todoID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem, err := client.AgentRun.Create().
		SetTicketID(ticketItem.ID).
		SetWorkflowID(workflowItem.ID).
		SetAgentID(agentItem.ID).
		SetProviderID(providerItem.ID).
		SetStatus(entagentrun.StatusCompleted).
		Save(ctx)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	workspaceRoot := filepath.Join(t.TempDir(), "workspace-root")
	repoPath := filepath.Join(workspaceRoot, "repo")
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		t.Fatalf("mkdir repo path: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "DIRTY.txt"), []byte("dirty\n"), 0o600); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}
	repoItem, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("openase").
		SetRepositoryURL("https://example.com/openase.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project repo: %v", err)
	}
	workspaceItem, err := client.TicketRepoWorkspace.Create().
		SetTicketID(ticketItem.ID).
		SetAgentRunID(runItem.ID).
		SetRepoID(repoItem.ID).
		SetWorkspaceRoot(workspaceRoot).
		SetRepoPath(repoPath).
		SetBranchName("scratch").
		SetState(entticketrepoworkspace.StateReady).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workspace row: %v", err)
	}

	stream := subscribeTopicEvents(t, bus, ticketEventsTopic)
	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/workspace/reset", ticketItem.ID),
		"",
	)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"reset":true`) {
		t.Fatalf("expected workspace reset response, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, err := os.Stat(workspaceRoot); !os.IsNotExist(err) {
		t.Fatalf("expected workspace root removed, got err=%v", err)
	}

	workspaceAfter, err := client.TicketRepoWorkspace.Get(ctx, workspaceItem.ID)
	if err != nil {
		t.Fatalf("reload workspace row: %v", err)
	}
	if workspaceAfter.State != entticketrepoworkspace.StateCleaned || workspaceAfter.CleanedAt == nil {
		t.Fatalf("expected cleaned workspace row, got %+v", workspaceAfter)
	}

	assertStringSet(t, readTicketEventTicketIDs(t, stream, 1), ticketItem.ID.String())
}

func TestHandleResetTicketWorkspaceRejectsActiveRun(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		newTicketService(client),
		newTicketStatusService(client),
		nil,
		nil,
		nil,
		WithTicketWorkspaceResetter(
			orchestrator.NewTicketWorkspaceResetService(
				client,
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				nil,
			),
		),
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	machine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName("local-devbox").
		SetHost(catalogdomain.LocalMachineHost).
		SetPort(0).
		Save(ctx)
	if err != nil {
		t.Fatalf("create machine: %v", err)
	}
	providerItem, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(machine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	agentItem, err := client.Agent.Create().
		SetProjectID(project.ID).
		SetProviderID(providerItem.ID).
		SetName("coder").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	doneID := findStatusIDByName(t, statuses, "Done")
	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("coding-workflow").
		SetType("coding").
		SetHarnessPath("roles/coding.md").
		AddPickupStatusIDs(todoID).
		AddFinishStatusIDs(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-1").
		SetTitle("Still running").
		SetStatusID(todoID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem, err := client.AgentRun.Create().
		SetTicketID(ticketItem.ID).
		SetWorkflowID(workflowItem.ID).
		SetAgentID(agentItem.ID).
		SetProviderID(providerItem.ID).
		SetStatus(entagentrun.StatusExecuting).
		Save(ctx)
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).SetCurrentRunID(runItem.ID).Save(ctx); err != nil {
		t.Fatalf("attach current run: %v", err)
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/tickets/%s/workspace/reset", ticketItem.ID),
		"",
	)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "WORKSPACE_RESET_CONFLICT") {
		t.Fatalf("expected workspace reset conflict, got %d: %s", rec.Code, rec.Body.String())
	}
}
