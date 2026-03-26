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
	"github.com/BetterAndBetterII/openase/internal/config"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
)

func TestTicketRoutesCRUDAndDependencies(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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

	statusSvc := ticketstatus.NewService(client)
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
		SetPickupStatusID(backlogID).
		SetFinishStatusID(doneID).
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
			"title":             "Implement ticket API",
			"description":       "cover create/list/detail/update",
			"priority":          "high",
			"type":              "epic",
			"workflow_id":       workflowItem.ID.String(),
			"target_machine_id": targetMachine.ID.String(),
			"created_by":        "user:gary",
			"budget_usd":        3.5,
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
	if parentCreateResp.Ticket.TargetMachineID == nil || *parentCreateResp.Ticket.TargetMachineID != targetMachine.ID.String() {
		t.Fatalf("expected target machine binding %s, got %+v", targetMachine.ID, parentCreateResp.Ticket.TargetMachineID)
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
			"title":             "Implement dependency HTTP routes",
			"priority":          "low",
			"status_id":         doneID.String(),
			"external_ref":      "BetterAndBetterII/openase#6",
			"budget_usd":        1.25,
			"target_machine_id": "",
			"parent_ticket_id":  "",
		},
		http.StatusOK,
		&childUpdateResp,
	)
	if childUpdateResp.Ticket.Parent != nil || len(childUpdateResp.Ticket.Dependencies) != 0 {
		t.Fatalf("expected patch to clear sub_issue link, got %+v", childUpdateResp.Ticket)
	}
	if childUpdateResp.Ticket.StatusID != doneID.String() || childUpdateResp.Ticket.ExternalRef != "BetterAndBetterII/openase#6" {
		t.Fatalf("unexpected child patch response: %+v", childUpdateResp.Ticket)
	}
	if childUpdateResp.Ticket.TargetMachineID != nil {
		t.Fatalf("expected patch to clear target machine binding, got %+v", childUpdateResp.Ticket.TargetMachineID)
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
	if commentCreateResp.Comment.CreatedBy != "user:reviewer" || commentCreateResp.Comment.Body == "" {
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
	if !strings.Contains(commentUpdateResp.Comment.Body, "markdown support") {
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
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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

	statusSvc := ticketstatus.NewService(client)
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

func TestTicketRoutesExternalLinks(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40027},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
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
			"url":         "https://github.com/BetterAndBetterII/openase/issues/99",
			"external_id": "BetterAndBetterII/openase#99",
			"title":       "F57: TicketExternalLink",
			"status":      "open",
			"relation":    "related",
		},
		http.StatusCreated,
		&firstLinkResp,
	)
	if firstLinkResp.ExternalLink.Type != "github_issue" || firstLinkResp.ExternalLink.ExternalID != "BetterAndBetterII/openase#99" {
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
			"url":         "https://github.com/BetterAndBetterII/openase/issues/6",
			"external_id": "BetterAndBetterII/openase#6",
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
		`{"type":"github_issue","url":"https://github.com/BetterAndBetterII/openase/issues/99","external_id":"BetterAndBetterII/openase#99"}`,
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
	if getResp.Ticket.ExternalRef != "BetterAndBetterII/openase#99" {
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
	if afterFirstDeleteResp.Ticket.ExternalRef != "BetterAndBetterII/openase#6" {
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

func TestListTicketsRouteReturnsEmptyArrayForNewProject(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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

func TestTicketRoutesCreateFirstTicketPerProjectAfterWorkflowCreate(t *testing.T) {
	client := openTestEntClient(t)
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o750); err != nil {
		t.Fatalf("create git marker: %v", err)
	}

	workflowSvc, err := workflowservice.NewService(client, slog.New(slog.NewTextHandler(io.Discard, nil)), repoRoot)
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
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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

		workflowResp := struct {
			Workflow workflowResponse `json:"workflow"`
		}{}
		executeJSON(
			t,
			server,
			http.MethodPost,
			fmt.Sprintf("/api/v1/projects/%s/workflows", project.ID),
			map[string]any{
				"name":             fmt.Sprintf("Coding Workflow %d", index+1),
				"type":             "coding",
				"pickup_status_id": todoID.String(),
				"finish_status_id": doneID.String(),
				"harness_content":  "---\nworkflow:\n  role: coding\n---\n\n# Coding\n",
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
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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
	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")

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
		SetIsPrimary(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create backend repo: %v", err)
	}
	frontendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("frontend").
		SetRepositoryURL("https://github.com/acme/frontend.git").
		SetDefaultBranch("develop").
		SetIsPrimary(false).
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
		SetIsPrimaryScope(true).
		Save(ctx); err != nil {
		t.Fatalf("create frontend repo scope: %v", err)
	}
	if _, err := client.TicketRepoScope.Create().
		SetTicketID(ticketItem.ID).
		SetRepoID(backendRepo.ID).
		SetBranchName("main").
		SetPrStatus("approved").
		SetCiStatus("passing").
		SetIsPrimaryScope(false).
		Save(ctx); err != nil {
		t.Fatalf("create backend repo scope: %v", err)
	}

	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetEventType("comment_added").
		SetMessage("Looks good.\n\n- Please include PR links.").
		SetMetadata(map[string]any{"comment_author": "user:reviewer"}).
		Save(ctx); err != nil {
		t.Fatalf("create comment event: %v", err)
	}
	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetEventType("agent.output").
		SetMessage("Opened frontend PR #9").
		SetMetadata(map[string]any{"stream": "stdout"}).
		Save(ctx); err != nil {
		t.Fatalf("create activity event: %v", err)
	}
	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetTicketID(ticketItem.ID).
		SetEventType("hook.failed").
		SetMessage("on_complete failed for run-tests.sh").
		SetMetadata(map[string]any{"hook_name": "on_complete", "command": "run-tests.sh"}).
		Save(ctx); err != nil {
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
	if _, err := client.TicketComment.Create().
		SetTicketID(ticketItem.ID).
		SetBody("Please split runtime hooks from discussion comments.").
		SetCreatedBy("user:product").
		Save(ctx); err != nil {
		t.Fatalf("create ticket comment: %v", err)
	}

	var payload struct {
		Ticket      ticketResponse                  `json:"ticket"`
		RepoScopes  []ticketRepoScopeDetailResponse `json:"repo_scopes"`
		Comments    []ticketCommentResponse         `json:"comments"`
		Activity    []activityEventResponse         `json:"activity"`
		HookHistory []activityEventResponse         `json:"hook_history"`
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
	if len(payload.Ticket.ExternalLinks) != 1 || payload.Ticket.ExternalLinks[0].ExternalID != "acme/frontend#9" {
		t.Fatalf("expected ticket detail to include external links, got %+v", payload.Ticket.ExternalLinks)
	}
	if len(payload.RepoScopes) != 2 || payload.RepoScopes[0].Repo == nil || payload.RepoScopes[0].Repo.Name != "frontend" {
		t.Fatalf("expected repo scopes with repo metadata, got %+v", payload.RepoScopes)
	}
	if payload.RepoScopes[0].PullRequestURL == nil || *payload.RepoScopes[0].PullRequestURL != "https://github.com/acme/frontend/pull/9" {
		t.Fatalf("expected frontend pull request URL, got %+v", payload.RepoScopes[0])
	}
	if len(payload.Comments) != 1 || payload.Comments[0].CreatedBy != "user:product" {
		t.Fatalf("expected ticket detail to include comments, got %+v", payload.Comments)
	}
	if len(payload.Activity) != 2 {
		t.Fatalf("expected two ticket activity events, got %+v", payload.Activity)
	}
	if len(payload.HookHistory) != 1 || payload.HookHistory[0].EventType != "hook.failed" {
		t.Fatalf("expected hook history to filter hook-tagged events, got %+v", payload.HookHistory)
	}
}

func TestTicketCommentRoutesCreateUpdateDelete(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40026},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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
	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
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

	if createPayload.Comment.Body != "First comment" || createPayload.Comment.CreatedBy != "user:reviewer" {
		t.Fatalf("unexpected created comment payload: %+v", createPayload.Comment)
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
			"body": "Updated comment body",
		},
		http.StatusOK,
		&updatePayload,
	)

	if updatePayload.Comment.Body != "Updated comment body" || updatePayload.Comment.UpdatedAt == nil {
		t.Fatalf("unexpected updated comment payload: %+v", updatePayload.Comment)
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
	if remaining != 0 {
		t.Fatalf("expected comment row to be deleted, remaining=%d", remaining)
	}
}

func TestTicketRouteStatusChangeClearsAssignmentAndReleasesAgent(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40024},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
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
		SetPickupStatusID(todoID).
		SetFinishStatusID(doneID).
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
}

func TestTicketRoutesPublishSSEEvents(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40025},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	doneID := findStatusIDByName(t, statuses, "Done")

	createResponse, cancelCreate := openSSERequest(t, testServer.URL+fmt.Sprintf("/api/v1/projects/%s/tickets/stream", project.ID))
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

	updateResponse, cancelUpdate := openSSERequest(t, testServer.URL+fmt.Sprintf("/api/v1/projects/%s/tickets/stream", project.ID))
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
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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

	statuses, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID)
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
