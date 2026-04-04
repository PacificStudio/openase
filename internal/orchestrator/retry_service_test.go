package orchestrator

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

func TestRetryServiceMarkAttemptFailedSchedulesExponentialBackoffAndReleasesClaim(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 0, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-401").
		SetTitle("Retry failed run").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflow.ID).
		SetAttemptCount(1).
		SetConsecutiveErrors(1).
		SetStallCount(2).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	originalRetryToken := ticketItem.RetryToken
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

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
	if got := backlogStageActiveRuns(ctx, t, client, fixture.projectID); got != 0 {
		t.Fatalf("expected retry release to drop backlog stage occupancy to 0, got %d", got)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID != nil {
		t.Fatalf("expected retry to clear current run, got %+v", ticketAfter.CurrentRunID)
	}
	if ticketAfter.AttemptCount != 2 || ticketAfter.ConsecutiveErrors != 2 {
		t.Fatalf("unexpected ticket counters after retry: %+v", ticketAfter)
	}
	if ticketAfter.StallCount != 0 {
		t.Fatalf("expected non-stall failure retry to reset stall count, got %d", ticketAfter.StallCount)
	}
	if ticketAfter.NextRetryAt == nil || !ticketAfter.NextRetryAt.Equal(wantNextRetryAt) {
		t.Fatalf("expected next retry at %s, got %+v", wantNextRetryAt, ticketAfter.NextRetryAt)
	}
	if ticketAfter.RetryToken == "" || ticketAfter.RetryToken == originalRetryToken {
		t.Fatalf("expected failed attempt to rotate retry token, got %q", ticketAfter.RetryToken)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.RuntimeControlState != "active" {
		t.Fatalf("expected agent control active after retry, got %+v", agentAfter)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusErrored {
		t.Fatalf("expected run marked errored, got %+v", runAfter)
	}
}

func TestRetryServiceMarkAttemptFailedPausesWhenBudgetIsExhausted(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 30, 0, 0, time.UTC)
	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-02", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-402").
		SetTitle("Pause exhausted budget").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflow.ID).
		SetBudgetUsd(5).
		SetCostAmount(5).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	originalRetryToken := ticketItem.RetryToken
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

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
	if ticketAfter.CurrentRunID != nil {
		t.Fatalf("expected budget pause to clear current run, got %+v", ticketAfter.CurrentRunID)
	}
	if ticketAfter.PauseReason != ticketing.PauseReasonBudgetExhausted.String() {
		t.Fatalf("expected pause reason %q, got %q", ticketing.PauseReasonBudgetExhausted, ticketAfter.PauseReason)
	}
	if ticketAfter.RetryToken == "" || ticketAfter.RetryToken == originalRetryToken {
		t.Fatalf("expected paused retry to rotate retry token, got %q", ticketAfter.RetryToken)
	}
	if got := backlogStageActiveRuns(ctx, t, client, fixture.projectID); got != 0 {
		t.Fatalf("expected paused retry release to drop backlog stage occupancy to 0, got %d", got)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusErrored {
		t.Fatalf("expected budget exhausted run errored, got %+v", runAfter)
	}
}

func TestRetryServiceLogsStructuredRetryDecision(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 16, 0, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "coding-log-01", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-499").
		SetTitle("Retry log coverage").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflow.ID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	_ = mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	var logBuffer bytes.Buffer
	retryService := NewRetryService(client, slog.New(slog.NewTextHandler(&logBuffer, nil)))
	retryService.now = func() time.Time {
		return now
	}

	if _, err := retryService.MarkAttemptFailed(ctx, ticketItem.ID); err != nil {
		t.Fatalf("mark attempt failed: %v", err)
	}

	logOutput := logBuffer.String()
	for _, want := range []string{
		"ticket retry scheduled",
		"operation=schedule_retry",
		"ticket_id=" + ticketItem.ID.String(),
		"project_id=" + fixture.projectID.String(),
		"workflow_id=" + workflow.ID.String(),
		"backoff_seconds=10",
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("expected retry log to contain %q, got %s", want, logOutput)
		}
	}
}

func TestRetryServiceMarkAttemptFailedUsesConsecutiveErrorsForBackoff(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 16, 30, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-05", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-406").
		SetTitle("Retry backoff follows current failure streak").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflow.ID).
		SetAttemptCount(8).
		SetConsecutiveErrors(0).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	_ = mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	retryService := NewRetryService(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	retryService.now = func() time.Time {
		return now
	}

	result, err := retryService.MarkAttemptFailed(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("mark attempt failed: %v", err)
	}

	wantNextRetryAt := now.Add(10 * time.Second)
	if !result.NextRetryAt.Equal(wantNextRetryAt) {
		t.Fatalf("expected next retry at %s, got %s", wantNextRetryAt, result.NextRetryAt)
	}
	if result.AttemptCount != 9 || result.ConsecutiveErrors != 1 {
		t.Fatalf("unexpected retry result counters: %+v", result)
	}
}

func TestSchedulerRunTickSkipsRetryPausedTickets(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "coding-03", 0)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

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
	if readyTicketAfter.CurrentRunID == nil {
		t.Fatalf("expected ready ticket to be dispatched, got %+v", readyTicketAfter)
	}
}

func TestSchedulerTryDispatchDropsStaleRetryCandidateWhenTokenRotates(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 14, 30, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "coding-04", 0)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-405").
		SetTitle("Stale retry candidate").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	staleRetryToken := uuid.NewString()
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetNextRetryAt(now.Add(-time.Second)).
		SetRetryToken(staleRetryToken).
		Save(ctx); err != nil {
		t.Fatalf("seed stale retry token: %v", err)
	}
	candidate, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload stale candidate snapshot: %v", err)
	}
	freshRetryToken := uuid.NewString()
	if _, err := client.Ticket.UpdateOneID(ticketItem.ID).
		SetRetryToken(freshRetryToken).
		Save(ctx); err != nil {
		t.Fatalf("rotate live retry token: %v", err)
	}

	workflowForDispatch, err := client.Workflow.Query().
		Where(entworkflow.IDEQ(workflow.ID)).
		WithProject().
		WithPickupStatuses().
		Only(ctx)
	if err != nil {
		t.Fatalf("load workflow dispatch edges: %v", err)
	}

	scheduler := newTestScheduler(client, now)
	dispatched, _, err := scheduler.tryDispatch(ctx, workflowForDispatch, candidate, now)
	if err != nil {
		t.Fatalf("tryDispatch(stale retry token): %v", err)
	}
	if dispatched {
		t.Fatal("expected stale retry candidate to be dropped")
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket after stale retry dispatch: %v", err)
	}
	if ticketAfter.CurrentRunID != nil {
		t.Fatalf("expected stale retry candidate to remain unclaimed, got %+v", ticketAfter.CurrentRunID)
	}
	runCount, err := client.AgentRun.Query().Where(entagentrun.TicketIDEQ(ticketItem.ID)).Count(ctx)
	if err != nil {
		t.Fatalf("count agent runs after stale retry dispatch: %v", err)
	}
	if runCount != 0 {
		t.Fatalf("expected no agent run for stale retry candidate, got %d", runCount)
	}
}
