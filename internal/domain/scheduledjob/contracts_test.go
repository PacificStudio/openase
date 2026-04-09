package scheduledjob

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestScheduledJobContracts(t *testing.T) {
	projectID := uuid.New()
	workflowID := uuid.New()
	statusID := uuid.New()
	lastRunAt := time.Unix(1700000000, 0).UTC()
	nextRunAt := lastRunAt.Add(30 * time.Minute)

	job := Job{
		ID:             uuid.New(),
		ProjectID:      projectID,
		Name:           "nightly sync",
		CronExpression: "0 * * * *",
		TicketTemplate: map[string]any{"title": "Sync drift"},
		IsEnabled:      true,
		LastRunAt:      &lastRunAt,
		NextRunAt:      &nextRunAt,
		WorkflowID:     &workflowID,
	}
	if job.ProjectID != projectID {
		t.Fatalf("Job.ProjectID = %s, want %s", job.ProjectID, projectID)
	}
	if got := job.TicketTemplate["title"]; got != "Sync drift" {
		t.Fatalf("Job.TicketTemplate[title] = %v", got)
	}
	if job.WorkflowID == nil || *job.WorkflowID != workflowID {
		t.Fatalf("Job.WorkflowID = %v, want %s", job.WorkflowID, workflowID)
	}
	if job.LastRunAt == nil || !job.LastRunAt.Equal(lastRunAt) {
		t.Fatalf("Job.LastRunAt = %v, want %v", job.LastRunAt, lastRunAt)
	}
	if job.NextRunAt == nil || !job.NextRunAt.Equal(nextRunAt) {
		t.Fatalf("Job.NextRunAt = %v, want %v", job.NextRunAt, nextRunAt)
	}

	workflow := Workflow{
		ID:             workflowID,
		ProjectID:      projectID,
		Name:           "Ops workflow",
		Type:           "Fullstack Developer",
		PickupStatuses: []WorkflowStatus{{ID: statusID, Name: "Todo"}},
	}
	if workflow.ProjectID != projectID {
		t.Fatalf("Workflow.ProjectID = %s, want %s", workflow.ProjectID, projectID)
	}
	if len(workflow.PickupStatuses) != 1 || workflow.PickupStatuses[0].ID != statusID {
		t.Fatalf("Workflow.PickupStatuses = %#v", workflow.PickupStatuses)
	}

	if !errors.Is(ErrProjectNotFound, ErrProjectNotFound) {
		t.Fatal("ErrProjectNotFound should be stable for errors.Is")
	}
	if ErrWorkflowNotFound.Error() == "" {
		t.Fatal("ErrWorkflowNotFound should have a message")
	}
	if ErrScheduledJobNotFound.Error() == "" {
		t.Fatal("ErrScheduledJobNotFound should have a message")
	}
	if ErrScheduledJobConflict.Error() == "" {
		t.Fatal("ErrScheduledJobConflict should have a message")
	}
	if ErrStatusNotFound.Error() == "" {
		t.Fatal("ErrStatusNotFound should have a message")
	}
}
