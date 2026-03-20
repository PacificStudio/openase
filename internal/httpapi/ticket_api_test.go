package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
)

func TestTicketRoutesCRUDAndDependencies(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketservice.NewService(client),
		ticketstatus.NewService(client),
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
			"external_ref":     "BetterAndBetterII/openase#6",
			"budget_usd":       1.25,
			"parent_ticket_id": "",
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
