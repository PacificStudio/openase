package scheduledjob

import (
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/scheduledjob"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

func TestScheduledJobTemplateParsingHelpers(t *testing.T) {
	template, err := ParseRawTicketTemplate(map[string]any{
		"title":       " Daily Build ",
		"description": " Run the workflow ",
		"status":      " Todo ",
		"priority":    "high",
		"type":        "bugfix",
		"created_by":  " system:test ",
		"budget_usd":  4.5,
	})
	if err != nil {
		t.Fatalf("ParseRawTicketTemplate() error = %v", err)
	}
	if template.Title != "Daily Build" || template.Description != "Run the workflow" || template.Status != "Todo" || template.Priority != ticketservice.PriorityHigh || template.Type != ticketservice.TypeBugfix || template.CreatedBy != "system:test" || template.BudgetUSD != 4.5 {
		t.Fatalf("ParseRawTicketTemplate() = %+v", template)
	}

	raw := template.Raw()
	if raw["title"] != "Daily Build" || raw["budget_usd"] != 4.5 {
		t.Fatalf("TicketTemplate.Raw() = %+v", raw)
	}
	minimal, err := ParseRawTicketTemplate(map[string]any{"title": "Task"})
	if err != nil {
		t.Fatalf("ParseRawTicketTemplate(minimal) error = %v", err)
	}
	if minimal.Priority != ticketservice.DefaultPriority || minimal.Type != ticketservice.DefaultType || minimal.CreatedBy != defaultCreatedBy {
		t.Fatalf("ParseRawTicketTemplate(minimal) = %+v", minimal)
	}
	if got := minimal.Raw(); got["created_by"] != defaultCreatedBy {
		t.Fatalf("minimal.Raw() = %+v", got)
	}

	if _, err := ParseRawTicketTemplate(nil); err == nil {
		t.Fatal("ParseRawTicketTemplate(nil) expected error")
	}
	if _, err := ParseRawTicketTemplate(map[string]any{"title": "Task", "extra": true}); err == nil {
		t.Fatal("ParseRawTicketTemplate(extra field) expected error")
	}
	if _, err := ParseRawTicketTemplate(map[string]any{"title": 1}); err == nil {
		t.Fatal("ParseRawTicketTemplate(title type) expected error")
	}
	if _, err := ParseRawTicketTemplate(map[string]any{"title": "Task", "priority": "bad"}); err == nil {
		t.Fatal("ParseRawTicketTemplate(priority) expected error")
	}
	if _, err := ParseRawTicketTemplate(map[string]any{"title": "Task", "type": "bad"}); err == nil {
		t.Fatal("ParseRawTicketTemplate(type) expected error")
	}
	if _, err := ParseRawTicketTemplate(map[string]any{"title": "Task", "created_by": " "}); err == nil {
		t.Fatal("ParseRawTicketTemplate(created_by blank) expected error")
	}
	if _, err := ParseRawTicketTemplate(map[string]any{"title": "Task", "budget_usd": -1.0}); err == nil {
		t.Fatal("ParseRawTicketTemplate(budget negative) expected error")
	}

	if got, err := parseRequiredString(map[string]any{"title": " Title "}, "title"); err != nil || got != "Title" {
		t.Fatalf("parseRequiredString() = %q, %v", got, err)
	}
	if _, err := parseRequiredString(map[string]any{}, "title"); err == nil {
		t.Fatal("parseRequiredString(missing) expected error")
	}
	if _, err := parseOptionalString(map[string]any{"description": 1}, "description"); err == nil {
		t.Fatal("parseOptionalString(type) expected error")
	}
	if got, err := parseOptionalString(map[string]any{}, "description"); err != nil || got != "" {
		t.Fatalf("parseOptionalString(missing) = %q, %v", got, err)
	}
	if got, err := parseBudgetUSD(2.25); err != nil || got != 2.25 {
		t.Fatalf("parseBudgetUSD() = %v, %v", got, err)
	}
	if _, err := parseBudgetUSD("bad"); err == nil {
		t.Fatal("parseBudgetUSD(type) expected error")
	}
}

func TestScheduledJobCronAndTemplateHelpers(t *testing.T) {
	svc := NewService(nil, nil, nil)
	now := time.Date(2026, 3, 27, 12, 30, 0, 0, time.UTC)
	svc.SetNowFunc(func() time.Time { return now })

	schedule, err := svc.parseCron("0 * * * *")
	if err != nil {
		t.Fatalf("parseCron() error = %v", err)
	}
	if _, err := svc.parseCron("bad cron"); err == nil {
		t.Fatal("parseCron(bad) expected error")
	}

	currentNext := now.Add(10 * time.Minute)
	current := domain.Job{
		IsEnabled: true,
		NextRunAt: &currentNext,
	}
	if got := svc.nextRunAfterUpdate(current, schedule, false, false); got != nil {
		t.Fatalf("nextRunAfterUpdate(disabled) = %v", got)
	}
	if got := svc.nextRunAfterUpdate(current, schedule, true, false); got == nil || !got.Equal(currentNext.UTC()) {
		t.Fatalf("nextRunAfterUpdate(existing future) = %v", got)
	}
	if got := svc.nextRunAfterUpdate(current, schedule, true, true); got == nil || !got.After(now) {
		t.Fatalf("nextRunAfterUpdate(cron changed) = %v", got)
	}
	pastNext := now.Add(-time.Minute)
	current.NextRunAt = &pastNext
	if got := svc.nextRunAfterUpdate(current, schedule, true, false); got == nil || !got.After(now) {
		t.Fatalf("nextRunAfterUpdate(past next run) = %v", got)
	}

	workflowID := uuid.New()
	workflowItem := &domain.Workflow{
		ID:   workflowID,
		Name: "Daily Coding",
		Type: "coding",
		PickupStatuses: []domain.WorkflowStatus{{
			ID:   uuid.New(),
			Name: "Todo",
		}},
	}
	if statusID, err := resolveScheduledJobPickupStatus("", workflowItem); err != nil || statusID != workflowItem.PickupStatuses[0].ID {
		t.Fatalf("resolveScheduledJobPickupStatus(single) = %s, %v", statusID, err)
	}
	if statusID, err := resolveScheduledJobPickupStatus("Explicit", workflowItem); err != nil || statusID != uuid.Nil {
		t.Fatalf("resolveScheduledJobPickupStatus(explicit) = %s, %v", statusID, err)
	}
	workflowItem.PickupStatuses = []domain.WorkflowStatus{{ID: uuid.New()}, {ID: uuid.New()}}
	if _, err := resolveScheduledJobPickupStatus("", workflowItem); err == nil {
		t.Fatal("resolveScheduledJobPickupStatus(multi) expected error")
	}
	workflowItem.PickupStatuses = nil
	if _, err := resolveScheduledJobPickupStatus("", workflowItem); err == nil {
		t.Fatal("resolveScheduledJobPickupStatus(no statuses) expected error")
	}

	jobID := uuid.New()
	projectID := uuid.New()
	contextMap := scheduledJobTemplateContext(domain.Job{
		ID:             jobID,
		Name:           "Daily",
		CronExpression: "0 * * * *",
	}, &domain.Workflow{ID: workflowID, Name: "Daily Coding", Type: "coding"}, now)
	if contextMap["date"] != "2026-03-27" || contextMap["job"].(map[string]any)["id"] != jobID.String() || contextMap["workflow"].(map[string]any)["type"] != "coding" {
		t.Fatalf("scheduledJobTemplateContext() = %+v", contextMap)
	}

	rendered, err := renderScheduledJobTemplateField("{{ job.name }} on {{ date }}", contextMap)
	if err != nil || rendered != "Daily on 2026-03-27" {
		t.Fatalf("renderScheduledJobTemplateField() = %q, %v", rendered, err)
	}
	if got, err := renderScheduledJobTemplateField(" ", contextMap); err != nil || got != "" {
		t.Fatalf("renderScheduledJobTemplateField(blank) = %q, %v", got, err)
	}
	if _, err := renderScheduledJobTemplateField("{{", contextMap); err == nil {
		t.Fatal("renderScheduledJobTemplateField(parse error) expected error")
	}
	if _, err := renderScheduledJobTemplateField("{{ missing.value }}", contextMap); err == nil {
		t.Fatal("renderScheduledJobTemplateField(render error) expected error")
	}

	lastRunAt := now.Add(-time.Hour)
	nextRunAt := now.Add(time.Hour)
	mapped, err := mapScheduledJob(domain.Job{
		ID:             jobID,
		ProjectID:      projectID,
		Name:           "Daily",
		CronExpression: "0 * * * *",
		TicketTemplate: map[string]any{"title": "Task"},
		IsEnabled:      true,
		LastRunAt:      &lastRunAt,
		NextRunAt:      &nextRunAt,
	})
	if err != nil {
		t.Fatalf("mapScheduledJob() error = %v", err)
	}
	if mapped.Name != "Daily" || mapped.TicketTemplate.Title != "Task" || mapped.LastRunAt == nil || mapped.NextRunAt == nil {
		t.Fatalf("mapScheduledJob() = %+v", mapped)
	}
	if mapped.LastRunAt == &lastRunAt || mapped.NextRunAt == &nextRunAt {
		t.Fatal("mapScheduledJob() did not clone times")
	}
	if _, err := mapScheduledJob(domain.Job{TicketTemplate: map[string]any{"title": 1}}); err == nil {
		t.Fatal("mapScheduledJob(invalid template) expected error")
	}

	if got := cloneTime(nil); got != nil {
		t.Fatalf("cloneTime(nil) = %v", got)
	}
	if got := cloneTime(&nextRunAt); got == nil || !got.Equal(nextRunAt.UTC()) || got == &nextRunAt {
		t.Fatalf("cloneTime() = %v", got)
	}
}
