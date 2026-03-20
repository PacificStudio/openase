package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/config"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	scheduledjobservice "github.com/BetterAndBetterII/openase/internal/scheduledjob"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
)

func TestScheduledJobRoutesCRUDAndTrigger(t *testing.T) {
	client := openTestEntClient(t)
	ticketSvc := ticketservice.NewService(client)
	scheduledJobSvc := scheduledjobservice.NewService(client, ticketSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	now := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)
	scheduledJobSvc.SetNowFunc(func() time.Time { return now })

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		ticketSvc,
		ticketstatus.NewService(client),
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

	statusSvc := ticketstatus.NewService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset ticket statuses: %v", err)
	}
	todoID := findStatusIDByName(t, statuses, "Todo")
	backlogID := findStatusIDByName(t, statuses, "Backlog")
	doneID := findStatusIDByName(t, statuses, "Done")

	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("Security Workflow").
		SetType(entworkflow.TypeSecurity).
		SetHarnessPath(".openase/harnesses/security.md").
		SetPickupStatusID(todoID).
		SetFinishStatusID(doneID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

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
			"workflow_id":     workflowItem.ID.String(),
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
