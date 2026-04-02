package scheduledjob

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	scheduledjobrepo "github.com/BetterAndBetterII/openase/internal/repo/scheduledjob"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

func TestScheduledJobServiceLifecycleTriggerAndRunDue(t *testing.T) {
	ctx := context.Background()
	client := openScheduledJobTestEntClient(t)
	fixture := seedScheduledJobFixture(ctx, t, client)
	ticketSvc := newTicketService(client)
	service := NewService(scheduledjobrepo.NewEntRepository(client), ticketSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	now := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)
	service.SetNowFunc(func() time.Time { return now })

	listBeforeCreate, err := service.List(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("List() before create error = %v", err)
	}
	if len(listBeforeCreate) != 0 {
		t.Fatalf("List() before create = %+v, want empty", listBeforeCreate)
	}

	created, err := service.Create(ctx, CreateInput{
		ProjectID:      fixture.projectID,
		Name:           "weekly-security-scan",
		CronExpression: "0 9 * * 1",
		TicketTemplate: TicketTemplate{
			Title:       "Weekly security scan - {{ date }}",
			Description: "Audit all repos",
			Status:      "Backlog",
			Priority:    ticketservice.PriorityHigh,
			Type:        ticketservice.TypeFeature,
			CreatedBy:   "system:scheduled-job",
			BudgetUSD:   12.5,
		},
		IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.NextRunAt == nil || created.NextRunAt.Format(time.RFC3339) != "2026-03-23T09:00:00Z" {
		t.Fatalf("Create() next run = %+v", created.NextRunAt)
	}

	if _, err := service.Create(ctx, CreateInput{
		ProjectID:      fixture.projectID,
		Name:           created.Name,
		CronExpression: "0 10 * * 1",
		TicketTemplate: TicketTemplate{Title: "Duplicate", Status: "Todo"},
		IsEnabled:      true,
	}); !errors.Is(err, ErrScheduledJobConflict) {
		t.Fatalf("Create() duplicate error = %v, want %v", err, ErrScheduledJobConflict)
	}

	listAfterCreate, err := service.List(ctx, fixture.projectID)
	if err != nil {
		t.Fatalf("List() after create error = %v", err)
	}
	if len(listAfterCreate) != 1 || listAfterCreate[0].ID != created.ID {
		t.Fatalf("List() after create = %+v", listAfterCreate)
	}

	disabled, err := service.Update(ctx, UpdateInput{
		JobID:     created.ID,
		Name:      Some("paused-weekly-security-scan"),
		IsEnabled: Some(false),
	})
	if err != nil {
		t.Fatalf("Update() disable error = %v", err)
	}
	if disabled.Name != "paused-weekly-security-scan" || disabled.IsEnabled || disabled.NextRunAt != nil {
		t.Fatalf("Update() disable = %+v", disabled)
	}

	enabled, err := service.Update(ctx, UpdateInput{
		JobID:          created.ID,
		Name:           Some("weekly-security-scan"),
		CronExpression: Some("0 8 * * 1"),
		TicketTemplate: Some(TicketTemplate{
			Title:       "Weekly security scan - {{ date }}",
			Description: "Audit all repos and infra",
			Status:      "Backlog",
			Priority:    ticketservice.PriorityUrgent,
			Type:        ticketservice.TypeBugfix,
			CreatedBy:   "system:scheduled-job",
			BudgetUSD:   21.0,
		}),
		IsEnabled: Some(true),
	})
	if err != nil {
		t.Fatalf("Update() enable error = %v", err)
	}
	if !enabled.IsEnabled || enabled.NextRunAt == nil {
		t.Fatalf("Update() enable = %+v", enabled)
	}

	triggerResult, err := service.Trigger(ctx, created.ID)
	if err != nil {
		t.Fatalf("Trigger() error = %v", err)
	}
	if triggerResult.Ticket.Title != "Weekly security scan - 2026-03-20" {
		t.Fatalf("Trigger() ticket title = %q", triggerResult.Ticket.Title)
	}
	if triggerResult.Ticket.StatusID != fixture.statusIDs["Backlog"] || triggerResult.Ticket.WorkflowID != nil {
		t.Fatalf("Trigger() ticket = %+v", triggerResult.Ticket)
	}
	if triggerResult.Ticket.CreatedBy != "system:scheduled-job" || triggerResult.Ticket.BudgetUSD != 21.0 {
		t.Fatalf("Trigger() ticket metadata = %+v", triggerResult.Ticket)
	}
	if triggerResult.Job.LastRunAt == nil || !triggerResult.Job.LastRunAt.Equal(now) {
		t.Fatalf("Trigger() job last run = %+v", triggerResult.Job.LastRunAt)
	}
	if triggerResult.Job.NextRunAt == nil || !triggerResult.Job.NextRunAt.Equal(*enabled.NextRunAt) {
		t.Fatalf("Trigger() job next run = %+v, want preserve %+v", triggerResult.Job.NextRunAt, enabled.NextRunAt)
	}

	if count, err := client.Ticket.Query().Where(entticket.ProjectIDEQ(fixture.projectID)).Count(ctx); err != nil || count != 1 {
		t.Fatalf("ticket count after trigger = %d, %v", count, err)
	}

	brokenDueJob, err := client.ScheduledJob.Create().
		SetProjectID(fixture.projectID).
		SetName("broken-due-job").
		SetCronExpression("0 9 * * 1").
		SetWorkflowID(fixture.workflowID).
		SetTicketTemplate(map[string]any{}).
		SetIsEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed broken due job: %v", err)
	}
	validDueJob, err := service.Create(ctx, CreateInput{
		ProjectID:      fixture.projectID,
		Name:           "todo-triage",
		CronExpression: "*/30 * * * *",
		TicketTemplate: TicketTemplate{
			Title:       "Todo triage - {{ time }}",
			Description: "Check inbox",
			Status:      "Todo",
			Priority:    ticketservice.PriorityMedium,
			Type:        ticketservice.TypeChore,
			CreatedBy:   "system:triage",
		},
		IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() valid due job error = %v", err)
	}
	dueAt := now.Add(-time.Minute)
	if _, err := client.ScheduledJob.UpdateOneID(brokenDueJob.ID).SetNextRunAt(dueAt).Save(ctx); err != nil {
		t.Fatalf("force broken due next run: %v", err)
	}
	if _, err := client.ScheduledJob.UpdateOneID(validDueJob.ID).SetNextRunAt(dueAt).Save(ctx); err != nil {
		t.Fatalf("force valid due next run: %v", err)
	}

	report, err := service.RunDue(ctx)
	if err != nil {
		t.Fatalf("RunDue() error = %v", err)
	}
	if report.JobsScanned != 2 || report.TicketsCreated != 1 {
		t.Fatalf("RunDue() = %+v, want scanned=2 created=1", report)
	}

	ticketsAfterRunDue, err := client.Ticket.Query().
		Where(entticket.ProjectIDEQ(fixture.projectID)).
		Order(ent.Asc(entticket.FieldIdentifier)).
		All(ctx)
	if err != nil {
		t.Fatalf("load tickets after RunDue: %v", err)
	}
	if len(ticketsAfterRunDue) != 2 {
		t.Fatalf("tickets after RunDue = %d, want 2", len(ticketsAfterRunDue))
	}
	if ticketsAfterRunDue[1].StatusID != fixture.statusIDs["Todo"] || ticketsAfterRunDue[1].CreatedBy != "system:triage" {
		t.Fatalf("RunDue() created ticket = %+v", ticketsAfterRunDue[1])
	}

	validDueAfter, err := client.ScheduledJob.Get(ctx, validDueJob.ID)
	if err != nil {
		t.Fatalf("reload valid due job: %v", err)
	}
	if validDueAfter.LastRunAt == nil || !validDueAfter.LastRunAt.Equal(now) || validDueAfter.NextRunAt == nil || !validDueAfter.NextRunAt.After(now) {
		t.Fatalf("valid due job after RunDue = %+v", validDueAfter)
	}
	brokenDueAfter, err := client.ScheduledJob.Get(ctx, brokenDueJob.ID)
	if err != nil {
		t.Fatalf("reload broken due job: %v", err)
	}
	if brokenDueAfter.LastRunAt != nil {
		t.Fatalf("broken due job should not advance, got %+v", brokenDueAfter)
	}

	deleteResult, err := service.Delete(ctx, brokenDueJob.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleteResult.DeletedJobID != brokenDueJob.ID {
		t.Fatalf("Delete() = %+v", deleteResult)
	}
}

func TestScheduledJobServiceValidationAndErrorPaths(t *testing.T) {
	ctx := context.Background()

	if _, err := (*Service)(nil).List(ctx, uuid.New()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil List() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := (*Service)(nil).Create(ctx, CreateInput{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil Create() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := (*Service)(nil).Update(ctx, UpdateInput{}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil Update() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := (*Service)(nil).Delete(ctx, uuid.New()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil Delete() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := (*Service)(nil).Trigger(ctx, uuid.New()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil Trigger() error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := (*Service)(nil).RunDue(ctx); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil RunDue() error = %v, want %v", err, ErrUnavailable)
	}

	client := openScheduledJobTestEntClient(t)
	fixture := seedScheduledJobFixture(ctx, t, client)
	service := NewService(scheduledjobrepo.NewEntRepository(client), newTicketService(client), slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.SetNowFunc(func() time.Time { return time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC) })

	if _, err := service.List(ctx, uuid.New()); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("List() missing project error = %v, want %v", err, ErrProjectNotFound)
	}
	if _, err := service.Create(ctx, CreateInput{
		ProjectID:      uuid.New(),
		Name:           "missing-project",
		CronExpression: "0 9 * * 1",
		TicketTemplate: TicketTemplate{Title: "Task", Status: "Todo"},
	}); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("Create() missing project error = %v, want %v", err, ErrProjectNotFound)
	}
	if _, err := service.Create(ctx, CreateInput{
		ProjectID:      fixture.projectID,
		Name:           "missing-status",
		CronExpression: "0 9 * * 1",
		TicketTemplate: TicketTemplate{Title: "Task"},
	}); !errors.Is(err, ErrInvalidTicketTemplate) {
		t.Fatalf("Create() missing status error = %v, want %v", err, ErrInvalidTicketTemplate)
	}
	if _, err := service.Create(ctx, CreateInput{
		ProjectID:      fixture.projectID,
		Name:           "bad-cron",
		CronExpression: "bad cron",
		TicketTemplate: TicketTemplate{Title: "Task", Status: "Todo"},
	}); !errors.Is(err, ErrInvalidCronExpression) {
		t.Fatalf("Create() bad cron error = %v, want %v", err, ErrInvalidCronExpression)
	}

	job, err := service.Create(ctx, CreateInput{
		ProjectID:      fixture.projectID,
		Name:           "status-check",
		CronExpression: "0 9 * * 1",
		TicketTemplate: TicketTemplate{
			Title:     "Status check",
			Status:    "Missing",
			Priority:  ticketservice.PriorityLow,
			Type:      ticketservice.TypeFeature,
			CreatedBy: "system:test",
		},
		IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() status-check error = %v", err)
	}

	if _, err := service.Update(ctx, UpdateInput{JobID: uuid.New(), Name: Some("missing")}); !errors.Is(err, ErrScheduledJobNotFound) {
		t.Fatalf("Update() missing job error = %v, want %v", err, ErrScheduledJobNotFound)
	}
	if _, err := service.Update(ctx, UpdateInput{
		JobID: job.ID,
		TicketTemplate: Some(TicketTemplate{
			Title: "Status check",
		}),
	}); !errors.Is(err, ErrInvalidTicketTemplate) {
		t.Fatalf("Update() missing status error = %v, want %v", err, ErrInvalidTicketTemplate)
	}
	if _, err := service.Update(ctx, UpdateInput{JobID: job.ID, CronExpression: Some("bad cron")}); !errors.Is(err, ErrInvalidCronExpression) {
		t.Fatalf("Update() bad cron error = %v, want %v", err, ErrInvalidCronExpression)
	}
	if _, err := service.Trigger(ctx, uuid.New()); !errors.Is(err, ErrScheduledJobNotFound) {
		t.Fatalf("Trigger() missing job error = %v, want %v", err, ErrScheduledJobNotFound)
	}
	if _, err := service.Trigger(ctx, job.ID); !errors.Is(err, ErrStatusNotFound) {
		t.Fatalf("Trigger() missing status error = %v, want %v", err, ErrStatusNotFound)
	}
	if _, err := service.Delete(ctx, uuid.New()); !errors.Is(err, ErrScheduledJobNotFound) {
		t.Fatalf("Delete() missing job error = %v, want %v", err, ErrScheduledJobNotFound)
	}
}

type scheduledJobFixture struct {
	projectID     uuid.UUID
	workflowID    uuid.UUID
	workflowAltID uuid.UUID
	statusIDs     map[string]uuid.UUID
}

func seedScheduledJobFixture(ctx context.Context, t *testing.T, client *ent.Client) scheduledJobFixture {
	t.Helper()

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
		SetStatus("In Progress").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	statuses, err := newTicketStatusService(client).ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset statuses: %v", err)
	}

	statusIDs := make(map[string]uuid.UUID, len(statuses))
	for _, status := range statuses {
		statusIDs[status.Name] = status.ID
	}

	workflowItem, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("Security Workflow").
		SetType(entworkflow.TypeSecurity).
		SetHarnessPath(".openase/harnesses/security.md").
		AddPickupStatusIDs(statusIDs["Todo"]).
		AddFinishStatusIDs(statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create primary workflow: %v", err)
	}
	workflowAlt, err := client.Workflow.Create().
		SetProjectID(project.ID).
		SetName("Backlog Workflow").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/backlog.md").
		AddPickupStatusIDs(statusIDs["Backlog"]).
		AddFinishStatusIDs(statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create alternate workflow: %v", err)
	}

	return scheduledJobFixture{
		projectID:     project.ID,
		workflowID:    workflowItem.ID,
		workflowAltID: workflowAlt.ID,
		statusIDs:     statusIDs,
	}
}

func openScheduledJobTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	return testPostgres.NewIsolatedEntClient(t)
}
