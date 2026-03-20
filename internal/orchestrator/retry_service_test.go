package orchestrator

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
)

func TestRetryServiceMarkAttemptFailedSchedulesExponentialBackoffAndReleasesClaim(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(t, ctx, client)
	now := time.Date(2026, 3, 20, 13, 0, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(t, ctx, "coding-01", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-401").
		SetTitle("Retry failed run").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflow.ID).
		SetAssignedAgentID(agentItem.ID).
		SetAttemptCount(1).
		SetConsecutiveErrors(1).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := client.Agent.UpdateOneID(agentItem.ID).
		SetStatus(entagent.StatusRunning).
		SetCurrentTicketID(ticketItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("mark agent running: %v", err)
	}

	retryService := NewRetryService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	retryService.now = func() time.Time {
		return now
	}

	result, err := retryService.MarkAttemptFailed(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("mark attempt failed: %v", err)
	}

	wantNextRetryAt := now.Add(20 * time.Second)
	if result.AttemptCount != 2 || result.ConsecutiveErrors != 2 {
		t.Fatalf("unexpected retry result counters: %+v", result)
	}
	if !result.NextRetryAt.Equal(wantNextRetryAt) {
		t.Fatalf("expected next retry at %s, got %s", wantNextRetryAt, result.NextRetryAt)
	}
	if result.RetryPaused {
		t.Fatalf("expected retry to stay active, got %+v", result)
	}
	if result.ReleasedAgentID == nil || *result.ReleasedAgentID != agentItem.ID {
		t.Fatalf("expected released agent %s, got %+v", agentItem.ID, result.ReleasedAgentID)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.AssignedAgentID != nil {
		t.Fatalf("expected retry to clear assignment, got %+v", ticketAfter.AssignedAgentID)
	}
	if ticketAfter.AttemptCount != 2 || ticketAfter.ConsecutiveErrors != 2 {
		t.Fatalf("unexpected ticket counters after retry: %+v", ticketAfter)
	}
	if ticketAfter.NextRetryAt == nil || !ticketAfter.NextRetryAt.Equal(wantNextRetryAt) {
		t.Fatalf("expected next retry at %s, got %+v", wantNextRetryAt, ticketAfter.NextRetryAt)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.Status != entagent.StatusIdle || agentAfter.CurrentTicketID != nil {
		t.Fatalf("expected agent release after retry, got %+v", agentAfter)
	}
}

func TestRetryServiceMarkAttemptFailedPausesWhenBudgetIsExhausted(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(t, ctx, client)
	now := time.Date(2026, 3, 20, 13, 30, 0, 0, time.UTC)

	agentItem := fixture.createAgent(t, ctx, "coding-02", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-402").
		SetTitle("Pause exhausted budget").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetAssignedAgentID(agentItem.ID).
		SetBudgetUsd(5).
		SetCostAmount(5).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := client.Agent.UpdateOneID(agentItem.ID).
		SetStatus(entagent.StatusRunning).
		SetCurrentTicketID(ticketItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("mark agent running: %v", err)
	}

	retryService := NewRetryService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	retryService.now = func() time.Time {
		return now
	}

	result, err := retryService.MarkAttemptFailed(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("mark attempt failed: %v", err)
	}

	if !result.RetryPaused || result.PauseReason != ticketing.PauseReasonBudgetExhausted {
		t.Fatalf("expected budget pause result, got %+v", result)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if !ticketAfter.RetryPaused {
		t.Fatal("expected ticket retry to be paused")
	}
	if ticketAfter.PauseReason != ticketing.PauseReasonBudgetExhausted.String() {
		t.Fatalf("expected pause reason %q, got %q", ticketing.PauseReasonBudgetExhausted, ticketAfter.PauseReason)
	}
}

func TestSchedulerRunTickSkipsRetryPausedTickets(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(t, ctx, client)
	now := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)

	if _, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx); err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	fixture.createAgent(t, ctx, "coding-03", 0)

	if _, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-403").
		SetTitle("Paused retry ticket").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetRetryPaused(true).
		SetPauseReason(ticketing.PauseReasonBudgetExhausted.String()).
		SetCreatedBy("user:test").
		Save(ctx); err != nil {
		t.Fatalf("create paused ticket: %v", err)
	}
	readyTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-404").
		SetTitle("Ready ticket").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ready ticket: %v", err)
	}

	scheduler := newTestScheduler(client, now)
	report, err := scheduler.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}

	if report.CandidatesScanned != 1 || report.TicketsDispatched != 1 {
		t.Fatalf("expected only the ready ticket to be considered, got %+v", report)
	}

	readyTicketAfter, err := client.Ticket.Get(ctx, readyTicket.ID)
	if err != nil {
		t.Fatalf("reload ready ticket: %v", err)
	}
	if readyTicketAfter.AssignedAgentID == nil {
		t.Fatalf("expected ready ticket to be dispatched, got %+v", readyTicketAfter)
	}
}
