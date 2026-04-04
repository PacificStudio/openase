package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func TestTicketStatusRoutesCRUDAndReset(t *testing.T) {
	client := openTicketStatusAPIEntClient(t)
	server := newTicketStatusTestServer(client)

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

	resetResp := ticketstatus.ListResult{}
	executeJSON(t, server, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/statuses/reset", project.ID), nil, http.StatusOK, &resetResp)
	if len(resetResp.Statuses) != 6 {
		t.Fatalf("expected 6 default statuses after reset, got %d", len(resetResp.Statuses))
	}
	if resetResp.Statuses[0].Name != "Backlog" || !resetResp.Statuses[0].IsDefault {
		t.Fatalf("expected Backlog to be first default status, got %+v", resetResp.Statuses[0])
	}

	createResp := struct {
		Status ticketstatus.Status `json:"status"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID),
		map[string]any{
			"name":            "QA",
			"stage":           "started",
			"color":           "#FF00AA",
			"max_active_runs": 1,
			"description":     "quality gate",
		},
		http.StatusCreated,
		&createResp,
	)
	if createResp.Status.Name != "QA" || createResp.Status.Stage != "started" || createResp.Status.MaxActiveRuns == nil || *createResp.Status.MaxActiveRuns != 1 {
		t.Fatalf("expected created status to carry max_active_runs, got %+v", createResp.Status)
	}

	updateResp := struct {
		Status ticketstatus.Status `json:"status"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/statuses/%s", createResp.Status.ID),
		map[string]any{
			"name":            "Ready for QA",
			"stage":           "completed",
			"icon":            "shield-check",
			"is_default":      false,
			"position":        9,
			"max_active_runs": nil,
			"description":     "review before merge",
			"color":           "#00AAFF",
		},
		http.StatusOK,
		&updateResp,
	)
	if updateResp.Status.Name != "Ready for QA" || updateResp.Status.Stage != "completed" || updateResp.Status.IsDefault || updateResp.Status.MaxActiveRuns != nil {
		t.Fatalf("expected updated status to change stage and clear max_active_runs, got %+v", updateResp.Status)
	}

	workflowWithDeletedStatus, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("qa-workflow").
		SetType("test").
		SetHarnessPath("roles/qa.md").
		AddPickupStatusIDs(updateResp.Status.ID).
		AddFinishStatusIDs(updateResp.Status.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow for delete rebind: %v", err)
	}
	ticketWithDeletedStatus, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-5").
		SetTitle("qa gate").
		SetStatusID(updateResp.Status.ID).
		SetWorkflowID(workflowWithDeletedStatus.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket for delete rebind: %v", err)
	}

	deleteResp := ticketstatus.DeleteResult{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/statuses/%s", updateResp.Status.ID),
		nil,
		http.StatusOK,
		&deleteResp,
	)
	if deleteResp.DeletedStatusID != updateResp.Status.ID {
		t.Fatalf("expected deleted status id %s, got %+v", updateResp.Status.ID, deleteResp)
	}

	ticketAfterDelete, err := client.Ticket.Get(ctx, ticketWithDeletedStatus.ID)
	if err != nil {
		t.Fatalf("load ticket after delete: %v", err)
	}
	if ticketAfterDelete.StatusID != deleteResp.ReplacementStatusID {
		t.Fatalf("expected ticket status to move to %s, got %s", deleteResp.ReplacementStatusID, ticketAfterDelete.StatusID)
	}
	workflowAfterDelete, err := client.Workflow.Query().
		Where(entworkflow.IDEQ(workflowWithDeletedStatus.ID)).
		WithPickupStatuses().
		WithFinishStatuses().
		Only(ctx)
	if err != nil {
		t.Fatalf("load workflow after delete: %v", err)
	}
	if len(workflowAfterDelete.Edges.PickupStatuses) != 1 || workflowAfterDelete.Edges.PickupStatuses[0].ID != deleteResp.ReplacementStatusID {
		t.Fatalf("expected pickup refs to move to %s, got %+v", deleteResp.ReplacementStatusID, workflowAfterDelete.Edges.PickupStatuses)
	}
	if len(workflowAfterDelete.Edges.FinishStatuses) != 1 || workflowAfterDelete.Edges.FinishStatuses[0].ID != deleteResp.ReplacementStatusID {
		t.Fatalf("expected finish refs to move to %s, got %+v", deleteResp.ReplacementStatusID, workflowAfterDelete.Edges.FinishStatuses)
	}

	listResp := ticketstatus.ListResult{}
	executeJSON(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID), nil, http.StatusOK, &listResp)
	if len(listResp.Statuses) != 6 {
		t.Fatalf("expected list to stay on default status template after delete, got %+v", listResp.Statuses)
	}
}

func TestTicketStatusRoutesRejectStageEndpoints(t *testing.T) {
	client := openTicketStatusAPIEntClient(t)
	server := newTicketStatusTestServer(client)

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/"+uuid.NewString()+"/stages", "")
	if strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("expected removed stage route to stop returning JSON API content, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTicketStatusRoutesValidateRequests(t *testing.T) {
	client := openTicketStatusAPIEntClient(t)
	server := newTicketStatusTestServer(client)

	rec := performJSONRequest(t, server, http.MethodPost, "/api/v1/projects/not-a-uuid/statuses", `{"name":"QA","color":"#fff"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid project id to return 400, got %d", rec.Code)
	}

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-validators").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-validators").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	badBody := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID),
		`{"name":"QA","color":"#ffffff","max_active_runs":0}`,
	)
	if badBody.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid max_active_runs to return 400, got %d: %s", badBody.Code, badBody.Body.String())
	}

	invalidStageBody := performJSONRequest(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID),
		`{"name":"QA","stage":"done","color":"#ffffff"}`,
	)
	if invalidStageBody.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid stage to return 400, got %d: %s", invalidStageBody.Code, invalidStageBody.Body.String())
	}
}

func newTicketStatusTestServer(client *ent.Client) *Server {
	return NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)
}

func openTicketStatusAPIEntClient(t *testing.T) *ent.Client {
	t.Helper()
	return testPostgres.NewIsolatedEntClient(t)
}
