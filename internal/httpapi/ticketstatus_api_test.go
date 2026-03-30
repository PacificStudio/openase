package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

func TestTicketStatusRoutesCRUDAndReset(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
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

	resetResp := struct {
		Stages      []ticketstatus.Stage       `json:"stages"`
		Statuses    []ticketstatus.Status      `json:"statuses"`
		StageGroups []ticketstatus.StatusGroup `json:"stage_groups"`
	}{}
	executeJSON(t, server, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/statuses/reset", project.ID), nil, http.StatusOK, &resetResp)
	if len(resetResp.Statuses) != 6 {
		t.Fatalf("expected 6 default statuses after reset, got %d", len(resetResp.Statuses))
	}
	if len(resetResp.Stages) != 4 {
		t.Fatalf("expected 4 default stages after reset, got %d", len(resetResp.Stages))
	}
	if len(resetResp.StageGroups) != 4 {
		t.Fatalf("expected 4 default stage groups after reset, got %d", len(resetResp.StageGroups))
	}
	if resetResp.Statuses[0].Name != "Backlog" || !resetResp.Statuses[0].IsDefault {
		t.Fatalf("expected Backlog to be first default status, got %+v", resetResp.Statuses[0])
	}
	stageKeys := make([]string, 0, len(resetResp.Stages))
	for _, stage := range resetResp.Stages {
		stageKeys = append(stageKeys, stage.Key)
	}
	if strings.Join(stageKeys, ",") != "backlog,in_progress,review,done" {
		t.Fatalf("unexpected default stage order after reset: %v", stageKeys)
	}
	backlogStageID := findStageIDByKey(t, resetResp.Stages, "backlog")
	inProgressStageID := findStageIDByKey(t, resetResp.Stages, "in_progress")
	reviewStageID := findStageIDByKey(t, resetResp.Stages, "review")
	doneStageID := findStageIDByKey(t, resetResp.Stages, "done")
	if status := findStatusByName(t, resetResp.Statuses, "Backlog"); status.StageID == nil || *status.StageID != backlogStageID {
		t.Fatalf("expected Backlog status to map to backlog stage %s, got %+v", backlogStageID, status)
	}
	if status := findStatusByName(t, resetResp.Statuses, "Todo"); status.StageID == nil || *status.StageID != backlogStageID {
		t.Fatalf("expected Todo status to map to backlog stage %s, got %+v", backlogStageID, status)
	}
	if status := findStatusByName(t, resetResp.Statuses, "In Progress"); status.StageID == nil || *status.StageID != inProgressStageID {
		t.Fatalf("expected In Progress status to map to in_progress stage %s, got %+v", inProgressStageID, status)
	}
	if status := findStatusByName(t, resetResp.Statuses, "In Review"); status.StageID == nil || *status.StageID != reviewStageID {
		t.Fatalf("expected In Review status to map to review stage %s, got %+v", reviewStageID, status)
	}
	if status := findStatusByName(t, resetResp.Statuses, "Done"); status.StageID == nil || *status.StageID != doneStageID {
		t.Fatalf("expected Done status to map to done stage %s, got %+v", doneStageID, status)
	}
	if status := findStatusByName(t, resetResp.Statuses, "Cancelled"); status.StageID == nil || *status.StageID != doneStageID {
		t.Fatalf("expected Cancelled status to map to done stage %s, got %+v", doneStageID, status)
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
			"name":        "QA",
			"color":       "#FF00AA",
			"description": "quality gate",
		},
		http.StatusCreated,
		&createResp,
	)
	if createResp.Status.Name != "QA" {
		t.Fatalf("expected created status to be QA, got %+v", createResp.Status)
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
			"name":        "Ready for QA",
			"icon":        "shield-check",
			"is_default":  true,
			"position":    9,
			"description": "review before merge",
			"color":       "#00AAFF",
		},
		http.StatusOK,
		&updateResp,
	)
	if updateResp.Status.Name != "Ready for QA" || !updateResp.Status.IsDefault {
		t.Fatalf("expected updated status to become default, got %+v", updateResp.Status)
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
	if len(workflowAfterDelete.Edges.PickupStatuses) != 1 || workflowAfterDelete.Edges.PickupStatuses[0].ID != deleteResp.ReplacementStatusID ||
		len(workflowAfterDelete.Edges.FinishStatuses) != 1 || workflowAfterDelete.Edges.FinishStatuses[0].ID != deleteResp.ReplacementStatusID {
		t.Fatalf(
			"expected workflow refs to move to %s, got pickup=%v finish=%v",
			deleteResp.ReplacementStatusID,
			workflowAfterDelete.Edges.PickupStatuses,
			workflowAfterDelete.Edges.FinishStatuses,
		)
	}

	extraResp := struct {
		Status ticketstatus.Status `json:"status"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID),
		map[string]any{
			"name":       "Research",
			"color":      "#111111",
			"position":   12,
			"is_default": false,
		},
		http.StatusCreated,
		&extraResp,
	)

	workflowForReset, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("research-workflow").
		SetType("custom").
		SetHarnessPath("roles/research.md").
		AddPickupStatusIDs(extraResp.Status.ID).
		AddFinishStatusIDs(extraResp.Status.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow for reset rebind: %v", err)
	}
	ticketForReset, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-6").
		SetTitle("research").
		SetStatusID(extraResp.Status.ID).
		SetWorkflowID(workflowForReset.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket for reset rebind: %v", err)
	}

	resetAgainResp := struct {
		Stages      []ticketstatus.Stage       `json:"stages"`
		Statuses    []ticketstatus.Status      `json:"statuses"`
		StageGroups []ticketstatus.StatusGroup `json:"stage_groups"`
	}{}
	executeJSON(t, server, http.MethodPost, fmt.Sprintf("/api/v1/projects/%s/statuses/reset", project.ID), nil, http.StatusOK, &resetAgainResp)
	if len(resetAgainResp.Statuses) != 6 {
		t.Fatalf("expected reset to leave 6 statuses, got %d", len(resetAgainResp.Statuses))
	}
	if len(resetAgainResp.Stages) != 4 || len(resetAgainResp.StageGroups) != 4 {
		t.Fatalf("expected reset to restore 4 stages and 4 groups, got stages=%d groups=%d", len(resetAgainResp.Stages), len(resetAgainResp.StageGroups))
	}
	for _, status := range resetAgainResp.Statuses {
		if status.Name == "Research" {
			t.Fatalf("expected reset to remove Research status, got %+v", resetAgainResp.Statuses)
		}
	}

	listResp := struct {
		Stages      []ticketstatus.Stage       `json:"stages"`
		Statuses    []ticketstatus.Status      `json:"statuses"`
		StageGroups []ticketstatus.StatusGroup `json:"stage_groups"`
	}{}
	executeJSON(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID), nil, http.StatusOK, &listResp)
	names := make([]string, 0, len(listResp.Statuses))
	for _, status := range listResp.Statuses {
		names = append(names, status.Name)
	}
	if strings.Join(names, ",") != "Backlog,Todo,In Progress,In Review,Done,Cancelled" {
		t.Fatalf("unexpected status order after reset: %v", names)
	}
	if len(listResp.StageGroups) != 4 {
		t.Fatalf("expected 4 stage groups after reset list, got %d", len(listResp.StageGroups))
	}
	if got := len(listResp.StageGroups[0].Statuses); got != 2 {
		t.Fatalf("expected backlog stage group to contain 2 statuses, got %d", got)
	}
	if got := len(listResp.StageGroups[1].Statuses); got != 1 {
		t.Fatalf("expected in_progress stage group to contain 1 status, got %d", got)
	}
	if got := len(listResp.StageGroups[2].Statuses); got != 1 {
		t.Fatalf("expected review stage group to contain 1 status, got %d", got)
	}
	if got := len(listResp.StageGroups[3].Statuses); got != 2 {
		t.Fatalf("expected done stage group to contain 2 statuses, got %d", got)
	}

	backlogID := findStatusIDByName(t, listResp.Statuses, "Backlog")
	todoID := findStatusIDByName(t, listResp.Statuses, "Todo")
	doneID := findStatusIDByName(t, listResp.Statuses, "Done")

	ticketAfterReset, err := client.Ticket.Get(ctx, ticketForReset.ID)
	if err != nil {
		t.Fatalf("load ticket after reset: %v", err)
	}
	if ticketAfterReset.StatusID != backlogID {
		t.Fatalf("expected ticket reset status to move to Backlog %s, got %s", backlogID, ticketAfterReset.StatusID)
	}

	workflowAfterReset, err := client.Workflow.Query().
		Where(entworkflow.IDEQ(workflowForReset.ID)).
		WithPickupStatuses().
		WithFinishStatuses().
		Only(ctx)
	if err != nil {
		t.Fatalf("load workflow after reset: %v", err)
	}
	if len(workflowAfterReset.Edges.PickupStatuses) != 1 || workflowAfterReset.Edges.PickupStatuses[0].ID != todoID {
		t.Fatalf("expected workflow pickup to move to Todo %s, got %v", todoID, workflowAfterReset.Edges.PickupStatuses)
	}
	if len(workflowAfterReset.Edges.FinishStatuses) != 1 || workflowAfterReset.Edges.FinishStatuses[0].ID != doneID {
		t.Fatalf("expected workflow finish to move to Done %s, got %v", doneID, workflowAfterReset.Edges.FinishStatuses)
	}
}

func TestTicketStageRoutesCRUDAndStatusGrouping(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
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

	createStageResp := struct {
		Stage ticketstatus.Stage `json:"stage"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/stages", project.ID),
		map[string]any{
			"key":             "qa",
			"name":            "QA",
			"position":        2,
			"max_active_runs": 1,
			"description":     "quality gate",
		},
		http.StatusCreated,
		&createStageResp,
	)
	if createStageResp.Stage.Key != "qa" || createStageResp.Stage.MaxActiveRuns == nil || *createStageResp.Stage.MaxActiveRuns != 1 {
		t.Fatalf("expected created stage to persist key and max_active_runs, got %+v", createStageResp.Stage)
	}

	createStatusResp := struct {
		Status ticketstatus.Status `json:"status"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID),
		map[string]any{
			"name":       "QA Ready",
			"color":      "#FF00AA",
			"stage_id":   createStageResp.Stage.ID.String(),
			"is_default": true,
		},
		http.StatusCreated,
		&createStatusResp,
	)
	if createStatusResp.Status.StageID == nil || *createStatusResp.Status.StageID != createStageResp.Stage.ID {
		t.Fatalf("expected created status to attach to stage %s, got %+v", createStageResp.Stage.ID, createStatusResp.Status)
	}

	listResp := struct {
		Stages      []ticketstatus.Stage       `json:"stages"`
		Statuses    []ticketstatus.Status      `json:"statuses"`
		StageGroups []ticketstatus.StatusGroup `json:"stage_groups"`
	}{}
	executeJSON(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID), nil, http.StatusOK, &listResp)
	if len(listResp.Stages) != 1 || len(listResp.StageGroups) != 1 {
		t.Fatalf("expected one stage and one stage group, got stages=%d groups=%d", len(listResp.Stages), len(listResp.StageGroups))
	}
	if listResp.StageGroups[0].Stage == nil || listResp.StageGroups[0].Stage.ID != createStageResp.Stage.ID {
		t.Fatalf("expected grouped response to include created stage, got %+v", listResp.StageGroups)
	}
	if len(listResp.StageGroups[0].Statuses) != 1 || listResp.StageGroups[0].Statuses[0].Name != "QA Ready" {
		t.Fatalf("expected grouped response to contain QA Ready status, got %+v", listResp.StageGroups[0].Statuses)
	}

	updateStageResp := struct {
		Stage ticketstatus.Stage `json:"stage"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/stages/%s", createStageResp.Stage.ID),
		map[string]any{
			"name":            "QA Gate",
			"position":        5,
			"max_active_runs": nil,
			"description":     "merge gate",
		},
		http.StatusOK,
		&updateStageResp,
	)
	if updateStageResp.Stage.Name != "QA Gate" || updateStageResp.Stage.MaxActiveRuns != nil || updateStageResp.Stage.Position != 5 {
		t.Fatalf("expected updated stage to clear max_active_runs and rename, got %+v", updateStageResp.Stage)
	}

	deleteStageResp := ticketstatus.DeleteStageResult{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/stages/%s", createStageResp.Stage.ID),
		nil,
		http.StatusOK,
		&deleteStageResp,
	)
	if deleteStageResp.DeletedStageID != createStageResp.Stage.ID || deleteStageResp.DetachedStatuses != 1 {
		t.Fatalf("expected delete stage result to report one detached status, got %+v", deleteStageResp)
	}

	statusAfterDelete, err := client.TicketStatus.Get(ctx, createStatusResp.Status.ID)
	if err != nil {
		t.Fatalf("load status after stage delete: %v", err)
	}
	if statusAfterDelete.StageID != nil {
		t.Fatalf("expected stage delete to clear status stage_id, got %+v", statusAfterDelete)
	}

	listAfterDelete := struct {
		Stages      []ticketstatus.Stage       `json:"stages"`
		Statuses    []ticketstatus.Status      `json:"statuses"`
		StageGroups []ticketstatus.StatusGroup `json:"stage_groups"`
	}{}
	executeJSON(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID), nil, http.StatusOK, &listAfterDelete)
	if len(listAfterDelete.Stages) != 0 {
		t.Fatalf("expected no stages after delete, got %+v", listAfterDelete.Stages)
	}
	if len(listAfterDelete.StageGroups) != 1 || listAfterDelete.StageGroups[0].Stage != nil || len(listAfterDelete.StageGroups[0].Statuses) != 1 {
		t.Fatalf("expected deleted-stage status to move into ungrouped bucket, got %+v", listAfterDelete.StageGroups)
	}
	if listAfterDelete.Statuses[0].StageID != nil {
		t.Fatalf("expected listed status to be ungrouped after stage delete, got %+v", listAfterDelete.Statuses[0])
	}
}

func TestListTicketStatusesRouteReturnsEmptyArrayForNewProject(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
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

	rec := performJSONRequest(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/statuses", project.ID), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected ticket status list 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"statuses":[]`) {
		t.Fatalf("expected empty statuses array in payload, got %s", rec.Body.String())
	}

	var payload struct {
		Stages      []ticketstatus.Stage       `json:"stages"`
		Statuses    []ticketstatus.Status      `json:"statuses"`
		StageGroups []ticketstatus.StatusGroup `json:"stage_groups"`
	}
	decodeResponse(t, rec, &payload)
	if payload.Stages == nil || len(payload.Stages) != 0 {
		t.Fatalf("expected non-nil empty stages slice, got %+v", payload.Stages)
	}
	if payload.Statuses == nil || len(payload.Statuses) != 0 {
		t.Fatalf("expected non-nil empty statuses slice, got %+v", payload.Statuses)
	}
	if payload.StageGroups == nil || len(payload.StageGroups) != 0 {
		t.Fatalf("expected non-nil empty stage_groups slice, got %+v", payload.StageGroups)
	}
}

func TestTicketStageListRouteNullableFieldsAndErrorMappings(t *testing.T) {
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

	unavailableRec := performJSONRequest(t, serverWithoutService, http.MethodGet, "/api/v1/projects/"+uuid.NewString()+"/stages", "")
	if unavailableRec.Code != http.StatusServiceUnavailable || !strings.Contains(unavailableRec.Body.String(), "SERVICE_UNAVAILABLE") {
		t.Fatalf("expected unavailable stage list response, got %d: %s", unavailableRec.Code, unavailableRec.Body.String())
	}

	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		ticketstatus.NewService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-stage-list").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-stage-list").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := ticketstatus.NewService(client).ResetToDefaultTemplate(ctx, project.ID); err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}

	stagesRec := performJSONRequest(t, server, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/stages", project.ID), "")
	if stagesRec.Code != http.StatusOK || !strings.Contains(stagesRec.Body.String(), `"stages"`) {
		t.Fatalf("expected stage list payload, got %d: %s", stagesRec.Code, stagesRec.Body.String())
	}

	badProjectRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/not-a-uuid/stages", "")
	if badProjectRec.Code != http.StatusBadRequest || !strings.Contains(badProjectRec.Body.String(), "INVALID_PROJECT_ID") {
		t.Fatalf("expected invalid project id response, got %d: %s", badProjectRec.Code, badProjectRec.Body.String())
	}

	for _, tc := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "project", err: ticketstatus.ErrProjectNotFound, wantStatus: http.StatusNotFound, wantCode: "PROJECT_NOT_FOUND"},
		{name: "stage", err: ticketstatus.ErrStageNotFound, wantStatus: http.StatusNotFound, wantCode: "STAGE_NOT_FOUND"},
		{name: "status", err: ticketstatus.ErrStatusNotFound, wantStatus: http.StatusNotFound, wantCode: "STATUS_NOT_FOUND"},
		{name: "duplicate-stage", err: ticketstatus.ErrDuplicateStageKey, wantStatus: http.StatusConflict, wantCode: "STAGE_KEY_CONFLICT"},
		{name: "duplicate-status", err: ticketstatus.ErrDuplicateStatusName, wantStatus: http.StatusConflict, wantCode: "STATUS_NAME_CONFLICT"},
		{name: "last-status", err: ticketstatus.ErrCannotDeleteLastStatus, wantStatus: http.StatusConflict, wantCode: "LAST_STATUS_DELETE_FORBIDDEN"},
		{name: "default-required", err: ticketstatus.ErrDefaultStatusRequired, wantStatus: http.StatusConflict, wantCode: "DEFAULT_STATUS_REQUIRED"},
		{name: "internal", err: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			echoCtx := e.NewContext(req, rec)

			if err := writeTicketStatusError(echoCtx, tc.err); err != nil {
				t.Fatalf("writeTicketStatusError() error = %v", err)
			}
			if rec.Code != tc.wantStatus || !strings.Contains(rec.Body.String(), tc.wantCode) {
				t.Fatalf("writeTicketStatusError(%s) = %d %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}

	var stringField nullableStringField
	if err := json.Unmarshal([]byte(`"stage-id"`), &stringField); err != nil || !stringField.Set || stringField.Value == nil || *stringField.Value != "stage-id" {
		t.Fatalf("nullableStringField string = %+v, %v", stringField, err)
	}
	if err := json.Unmarshal([]byte(`null`), &stringField); err != nil || !stringField.Set || stringField.Value != nil {
		t.Fatalf("nullableStringField null = %+v, %v", stringField, err)
	}

	var intField nullableIntField
	if err := json.Unmarshal([]byte(`3`), &intField); err != nil || !intField.Set || intField.Value == nil || *intField.Value != 3 {
		t.Fatalf("nullableIntField int = %+v, %v", intField, err)
	}
	if err := json.Unmarshal([]byte(`null`), &intField); err != nil || !intField.Set || intField.Value != nil {
		t.Fatalf("nullableIntField null = %+v, %v", intField, err)
	}
	if err := json.Unmarshal([]byte(`"bad"`), &intField); err == nil {
		t.Fatal("nullableIntField invalid JSON expected error")
	}
}

func TestTicketStatusRoutesErrorMappingsAndInvalidPayloads(t *testing.T) {
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
	validStageID := uuid.NewString()
	validStatusID := uuid.NewString()

	for _, tc := range []struct {
		name   string
		method string
		target string
		body   string
	}{
		{name: "list stages unavailable", method: http.MethodGet, target: "/api/v1/projects/" + validProjectID + "/stages"},
		{name: "create stage unavailable", method: http.MethodPost, target: "/api/v1/projects/" + validProjectID + "/stages", body: `{"key":"qa","name":"QA"}`},
		{name: "update stage unavailable", method: http.MethodPatch, target: "/api/v1/stages/" + validStageID, body: `{"name":"QA"}`},
		{name: "delete stage unavailable", method: http.MethodDelete, target: "/api/v1/stages/" + validStageID},
		{name: "list statuses unavailable", method: http.MethodGet, target: "/api/v1/projects/" + validProjectID + "/statuses"},
		{name: "create status unavailable", method: http.MethodPost, target: "/api/v1/projects/" + validProjectID + "/statuses", body: `{"name":"QA","color":"#fff"}`},
		{name: "reset statuses unavailable", method: http.MethodPost, target: "/api/v1/projects/" + validProjectID + "/statuses/reset"},
		{name: "update status unavailable", method: http.MethodPatch, target: "/api/v1/statuses/" + validStatusID, body: `{"name":"QA"}`},
		{name: "delete status unavailable", method: http.MethodDelete, target: "/api/v1/statuses/" + validStatusID},
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
		nil,
		ticketstatus.NewService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-ticketstatus-errors").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE TicketStatus Errors").
		SetSlug("openase-ticketstatus-errors").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	stage, err := client.TicketStage.Create().
		SetProjectID(project.ID).
		SetKey("qa").
		SetName("QA").
		Save(ctx)
	if err != nil {
		t.Fatalf("create stage: %v", err)
	}
	status, err := client.TicketStatus.Create().
		SetProjectID(project.ID).
		SetName("QA Ready").
		SetColor("#FF00AA").
		SetStageID(stage.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create status: %v", err)
	}

	for _, tc := range []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "list stages invalid project", method: http.MethodGet, target: "/api/v1/projects/not-a-uuid/stages", wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "create stage invalid project", method: http.MethodPost, target: "/api/v1/projects/not-a-uuid/stages", body: `{"key":"qa","name":"QA"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "create stage invalid json", method: http.MethodPost, target: "/api/v1/projects/" + project.ID.String() + "/stages", body: `{"key":`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "create stage invalid request", method: http.MethodPost, target: "/api/v1/projects/" + project.ID.String() + "/stages", body: `{"key":"","name":"QA"}`, wantStatus: http.StatusBadRequest, wantBody: "key must not be empty"},
		{name: "update stage invalid stage", method: http.MethodPatch, target: "/api/v1/stages/not-a-uuid", body: `{"name":"QA"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_STAGE_ID"},
		{name: "update stage invalid json", method: http.MethodPatch, target: "/api/v1/stages/" + stage.ID.String(), body: `{"name":`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "update stage invalid request", method: http.MethodPatch, target: "/api/v1/stages/" + stage.ID.String(), body: `{"position":-1}`, wantStatus: http.StatusBadRequest, wantBody: "position must be greater than or equal to 0"},
		{name: "delete stage invalid stage", method: http.MethodDelete, target: "/api/v1/stages/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "INVALID_STAGE_ID"},
		{name: "list statuses invalid project", method: http.MethodGet, target: "/api/v1/projects/not-a-uuid/statuses", wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "create status invalid project", method: http.MethodPost, target: "/api/v1/projects/not-a-uuid/statuses", body: `{"name":"QA","color":"#fff"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "create status invalid json", method: http.MethodPost, target: "/api/v1/projects/" + project.ID.String() + "/statuses", body: `{"name":`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "create status invalid request", method: http.MethodPost, target: "/api/v1/projects/" + project.ID.String() + "/statuses", body: `{"name":"QA"}`, wantStatus: http.StatusBadRequest, wantBody: "color must not be empty"},
		{name: "update status invalid status", method: http.MethodPatch, target: "/api/v1/statuses/not-a-uuid", body: `{"name":"QA"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_STATUS_ID"},
		{name: "update status invalid json", method: http.MethodPatch, target: "/api/v1/statuses/" + status.ID.String(), body: `{"name":`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "update status invalid request", method: http.MethodPatch, target: "/api/v1/statuses/" + status.ID.String(), body: `{"stage_id":"not-a-uuid"}`, wantStatus: http.StatusBadRequest, wantBody: "stage_id must be a valid UUID"},
		{name: "delete status invalid status", method: http.MethodDelete, target: "/api/v1/statuses/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "INVALID_STATUS_ID"},
		{name: "reset statuses invalid project", method: http.MethodPost, target: "/api/v1/projects/not-a-uuid/statuses/reset", wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := performJSONRequest(t, server, tc.method, tc.target, tc.body)
			if rec.Code != tc.wantStatus || !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("%s %s = %d %s", tc.method, tc.target, rec.Code, rec.Body.String())
			}
		})
	}
}

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}

func executeJSON(t *testing.T, server *Server, method string, target string, body any, wantStatus int, out any) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != wantStatus {
		t.Fatalf("expected %s %s to return %d, got %d with body %s", method, target, wantStatus, rec.Code, rec.Body.String())
	}
	if out == nil {
		return
	}
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

func findStatusIDByName(t *testing.T, statuses []ticketstatus.Status, name string) uuid.UUID {
	t.Helper()

	for _, status := range statuses {
		if status.Name == name {
			return status.ID
		}
	}
	t.Fatalf("status %q not found in %+v", name, statuses)
	return uuid.UUID{}
}

func findStatusByName(t *testing.T, statuses []ticketstatus.Status, name string) ticketstatus.Status {
	t.Helper()

	for _, status := range statuses {
		if status.Name == name {
			return status
		}
	}
	t.Fatalf("status %q not found in %+v", name, statuses)
	return ticketstatus.Status{}
}

func findStageIDByKey(t *testing.T, stages []ticketstatus.Stage, key string) uuid.UUID {
	t.Helper()

	for _, stage := range stages {
		if stage.Key == key {
			return stage.ID
		}
	}
	t.Fatalf("stage %q not found in %+v", key, stages)
	return uuid.UUID{}
}
