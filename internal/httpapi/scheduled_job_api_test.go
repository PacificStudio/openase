package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	scheduledjobrepo "github.com/BetterAndBetterII/openase/internal/repo/scheduledjob"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	"github.com/google/uuid"
)

func TestScheduledJobRoutesErrorMappingsAndInvalidPayloads(t *testing.T) {
	client := openTestEntClient(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ticketSvc := newTicketService(client)
	scheduledJobSvc := scheduledjobservice.NewService(scheduledjobrepo.NewEntRepository(client), ticketSvc, logger)

	server := NewServer(
		config.ServerConfig{Port: 40027},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		ticketSvc,
		newTicketStatusService(client),
		nil,
		nil,
		nil,
		WithScheduledJobService(scheduledJobSvc),
	)
	unavailableServer := NewServer(
		config.ServerConfig{Port: 40028},
		config.GitHubConfig{},
		logger,
		eventinfra.NewChannelBus(),
		ticketSvc,
		newTicketStatusService(client),
		nil,
		nil,
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("OpenASE").
		SetSlug("openase-scheduled-job-errors").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Platform").
		SetSlug("platform-scheduled-job-errors").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}
	_ = findStatusIDByName(t, statuses, "Todo")

	rec := performJSONRequest(t, unavailableServer, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/scheduled-jobs", project.ID), "")
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "SERVICE_UNAVAILABLE") {
		t.Fatalf("list scheduled jobs unavailable = %d %s", rec.Code, rec.Body.String())
	}

	for _, testCase := range []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "list invalid project", method: http.MethodGet, target: "/api/v1/projects/not-a-uuid/scheduled-jobs", wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "create invalid project", method: http.MethodPost, target: "/api/v1/projects/not-a-uuid/scheduled-jobs", body: `{"name":"nightly","cron_expression":"0 1 * * *","ticket_template":{"title":"Nightly","description":"Run checks","status":"Todo","priority":"medium","type":"feature"}}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_PROJECT_ID"},
		{name: "create invalid payload", method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/scheduled-jobs", project.ID), body: `{"name":"","cron_expression":"0 1 * * *","ticket_template":{"title":"Nightly","description":"Run checks","status":"Todo","priority":"medium","type":"feature"}}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "create invalid cron", method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/scheduled-jobs", project.ID), body: `{"name":"nightly","cron_expression":"bad cron","ticket_template":{"title":"Nightly","description":"Run checks","status":"Todo","priority":"medium","type":"feature"}}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_CRON_EXPRESSION"},
		{name: "create missing status", method: http.MethodPost, target: fmt.Sprintf("/api/v1/projects/%s/scheduled-jobs", project.ID), body: `{"name":"nightly","cron_expression":"0 1 * * *","ticket_template":{"title":"Nightly","description":"Run checks","priority":"medium","type":"feature"}}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "update invalid job id", method: http.MethodPatch, target: "/api/v1/scheduled-jobs/not-a-uuid", body: `{"name":"renamed"}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_JOB_ID"},
		{name: "update invalid payload", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/scheduled-jobs/%s", uuid.New()), body: `{"name":"   "}`, wantStatus: http.StatusBadRequest, wantBody: "INVALID_REQUEST"},
		{name: "update missing job", method: http.MethodPatch, target: fmt.Sprintf("/api/v1/scheduled-jobs/%s", uuid.New()), body: `{"name":"renamed"}`, wantStatus: http.StatusNotFound, wantBody: "SCHEDULED_JOB_NOT_FOUND"},
		{name: "delete invalid job id", method: http.MethodDelete, target: "/api/v1/scheduled-jobs/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "INVALID_JOB_ID"},
		{name: "delete missing job", method: http.MethodDelete, target: fmt.Sprintf("/api/v1/scheduled-jobs/%s", uuid.New()), wantStatus: http.StatusNotFound, wantBody: "SCHEDULED_JOB_NOT_FOUND"},
		{name: "trigger invalid job id", method: http.MethodPost, target: "/api/v1/scheduled-jobs/not-a-uuid/trigger", wantStatus: http.StatusBadRequest, wantBody: "INVALID_JOB_ID"},
		{name: "trigger missing job", method: http.MethodPost, target: fmt.Sprintf("/api/v1/scheduled-jobs/%s/trigger", uuid.New()), wantStatus: http.StatusNotFound, wantBody: "SCHEDULED_JOB_NOT_FOUND"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequest(t, server, testCase.method, testCase.target, testCase.body)
			if rec.Code != testCase.wantStatus || !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("%s %s = %d %s, want %d containing %q", testCase.method, testCase.target, rec.Code, rec.Body.String(), testCase.wantStatus, testCase.wantBody)
			}
		})
	}
}

func TestScheduledJobRoutesCRUDAndTrigger(t *testing.T) {
	client := openTestEntClient(t)
	ticketSvc := newTicketService(client)
	scheduledJobSvc := scheduledjobservice.NewService(scheduledjobrepo.NewEntRepository(client), ticketSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	now := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)
	scheduledJobSvc.SetNowFunc(func() time.Time { return now })

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketSvc,
		newTicketStatusService(client),
		nil,
		nil,
		nil,
		WithScheduledJobService(scheduledJobSvc),
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

	statusSvc := newTicketStatusService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	backlogID := findStatusIDByName(t, statuses, "Backlog")

	createResp := struct {
		ScheduledJob scheduledJobResponse `json:"scheduled_job"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/projects/%s/scheduled-jobs", project.ID),
		map[string]any{
			"name":            "weekly-security-scan",
			"cron_expression": "0 9 * * 1",
			"ticket_template": map[string]any{
				"title":       "Weekly security scan - {{ date }}",
				"description": "Audit all repos",
				"status":      "Backlog",
				"priority":    "high",
				"type":        "feature",
			},
		},
		http.StatusCreated,
		&createResp,
	)
	if createResp.ScheduledJob.Name != "weekly-security-scan" {
		t.Fatalf("unexpected scheduled job create response: %+v", createResp.ScheduledJob)
	}
	if createResp.ScheduledJob.NextRunAt == nil || *createResp.ScheduledJob.NextRunAt != "2026-03-23T09:00:00Z" {
		t.Fatalf("expected next run at 2026-03-23T09:00:00Z, got %+v", createResp.ScheduledJob.NextRunAt)
	}

	listResp := struct {
		ScheduledJobs []scheduledJobResponse `json:"scheduled_jobs"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/scheduled-jobs", project.ID),
		nil,
		http.StatusOK,
		&listResp,
	)
	if len(listResp.ScheduledJobs) != 1 || listResp.ScheduledJobs[0].ID != createResp.ScheduledJob.ID {
		t.Fatalf("expected one scheduled job in list, got %+v", listResp.ScheduledJobs)
	}

	triggerResp := struct {
		ScheduledJob scheduledJobResponse `json:"scheduled_job"`
		Ticket       ticketResponse       `json:"ticket"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/scheduled-jobs/%s/trigger", createResp.ScheduledJob.ID),
		nil,
		http.StatusOK,
		&triggerResp,
	)
	if triggerResp.Ticket.Title != "Weekly security scan - 2026-03-20" {
		t.Fatalf("expected rendered ticket title, got %+v", triggerResp.Ticket)
	}
	if triggerResp.Ticket.StatusID != backlogID.String() || triggerResp.Ticket.StatusName != "Backlog" {
		t.Fatalf("expected trigger to honor template status Backlog %s, got %+v", backlogID, triggerResp.Ticket)
	}
	if triggerResp.Ticket.WorkflowID != nil {
		t.Fatalf("expected trigger to create an unbound ticket, got %+v", triggerResp.Ticket.WorkflowID)
	}
	if triggerResp.ScheduledJob.LastRunAt == nil || *triggerResp.ScheduledJob.LastRunAt != "2026-03-20T09:00:00Z" {
		t.Fatalf("expected trigger to set last_run_at, got %+v", triggerResp.ScheduledJob)
	}
	if triggerResp.ScheduledJob.NextRunAt == nil || *triggerResp.ScheduledJob.NextRunAt != "2026-03-23T09:00:00Z" {
		t.Fatalf("expected trigger to preserve future next_run_at, got %+v", triggerResp.ScheduledJob)
	}

	updateResp := struct {
		ScheduledJob scheduledJobResponse `json:"scheduled_job"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		fmt.Sprintf("/api/v1/scheduled-jobs/%s", createResp.ScheduledJob.ID),
		map[string]any{
			"name":       "paused-weekly-security-scan",
			"is_enabled": false,
		},
		http.StatusOK,
		&updateResp,
	)
	if updateResp.ScheduledJob.Name != "paused-weekly-security-scan" || updateResp.ScheduledJob.IsEnabled {
		t.Fatalf("expected patch to rename and disable job, got %+v", updateResp.ScheduledJob)
	}
	if updateResp.ScheduledJob.NextRunAt != nil {
		t.Fatalf("expected disabled job to clear next_run_at, got %+v", updateResp.ScheduledJob.NextRunAt)
	}

	deleteResp := struct {
		DeletedJobID string `json:"deleted_job_id"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodDelete,
		fmt.Sprintf("/api/v1/scheduled-jobs/%s", createResp.ScheduledJob.ID),
		nil,
		http.StatusOK,
		&deleteResp,
	)
	if deleteResp.DeletedJobID != createResp.ScheduledJob.ID {
		t.Fatalf("unexpected delete response: %+v", deleteResp)
	}

	listAfterDeleteResp := struct {
		ScheduledJobs []scheduledJobResponse `json:"scheduled_jobs"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/scheduled-jobs", project.ID),
		nil,
		http.StatusOK,
		&listAfterDeleteResp,
	)
	if len(listAfterDeleteResp.ScheduledJobs) != 0 {
		t.Fatalf("expected scheduled job list to be empty after delete, got %+v", listAfterDeleteResp.ScheduledJobs)
	}
}
