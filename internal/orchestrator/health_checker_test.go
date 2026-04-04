package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	"github.com/google/uuid"
)

func TestHealthCheckerReleasesStalledClaim(t *testing.T) {
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
		SetStallTimeoutMinutes(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetWorkflowID(workflow.ID).
		SetIdentifier("ASE-401").
		SetTitle("Recover stalled claim").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	originalRetryToken := ticketItem.RetryToken
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusExecuting, now.Add(-2*time.Minute))

	checker := newTestHealthChecker(client, now)
	report, err := checker.Run(ctx)
	if err != nil {
		t.Fatalf("run health checker: %v", err)
	}

	if report.ClaimsChecked != 1 || report.StalledClaims != 1 || report.AgentsReleased != 1 {
		t.Fatalf("unexpected health report: %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID != nil {
		t.Fatalf("expected current run to be cleared, got %+v", ticketAfter.CurrentRunID)
	}
	if ticketAfter.StallCount != 1 {
		t.Fatalf("expected stall count 1, got %d", ticketAfter.StallCount)
	}
	if ticketAfter.AttemptCount != 1 || ticketAfter.ConsecutiveErrors != 1 {
		t.Fatalf(
			"expected first stall to increment attempts/errors, got attempts=%d errors=%d",
			ticketAfter.AttemptCount,
			ticketAfter.ConsecutiveErrors,
		)
	}
	if ticketAfter.RetryPaused || ticketAfter.PauseReason != "" {
		t.Fatalf("expected first stall to keep retry active, got %+v", ticketAfter)
	}
	if ticketAfter.NextRetryAt == nil || !ticketAfter.NextRetryAt.Equal(now.Add(stalledRetryDelay)) {
		t.Fatalf("expected next retry at %s, got %+v", now.Add(stalledRetryDelay), ticketAfter.NextRetryAt)
	}
	if ticketAfter.RetryToken == "" || ticketAfter.RetryToken == originalRetryToken {
		t.Fatalf("expected stalled claim release to rotate retry token, got %q", ticketAfter.RetryToken)
	}
	if got := backlogStageActiveRuns(ctx, t, client, fixture.projectID); got != 0 {
		t.Fatalf("expected stalled release to drop backlog stage occupancy to 0, got %d", got)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.RuntimeControlState != "active" {
		t.Fatalf("expected agent control state active, got %+v", agentAfter)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusErrored {
		t.Fatalf("expected stalled run errored, got %+v", runAfter)
	}
}

func TestHealthCheckerKeepsSecondConsecutiveStallRetryActive(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 15, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		SetStallTimeoutMinutes(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-01b", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetWorkflowID(workflow.ID).
		SetIdentifier("ASE-401B").
		SetTitle("Recover second stalled claim").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		SetAttemptCount(1).
		SetConsecutiveErrors(1).
		SetStallCount(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusExecuting, now.Add(-2*time.Minute))

	checker := newTestHealthChecker(client, now)
	report, err := checker.Run(ctx)
	if err != nil {
		t.Fatalf("run health checker: %v", err)
	}
	if report.StalledClaims != 1 {
		t.Fatalf("expected second stall to be handled, got %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.StallCount != 2 {
		t.Fatalf("expected stall count 2, got %d", ticketAfter.StallCount)
	}
	if ticketAfter.AttemptCount != 2 || ticketAfter.ConsecutiveErrors != 2 {
		t.Fatalf(
			"expected second stall to increment attempts/errors, got attempts=%d errors=%d",
			ticketAfter.AttemptCount,
			ticketAfter.ConsecutiveErrors,
		)
	}
	if ticketAfter.RetryPaused || ticketAfter.PauseReason != "" {
		t.Fatalf("expected second stall to keep retry active, got %+v", ticketAfter)
	}
	if ticketAfter.NextRetryAt == nil || !ticketAfter.NextRetryAt.Equal(now.Add(stalledRetryDelay)) {
		t.Fatalf("expected next retry at %s, got %+v", now.Add(stalledRetryDelay), ticketAfter.NextRetryAt)
	}
}

func TestHealthCheckerPausesRetryAfterConfiguredConsecutiveStalls(t *testing.T) {
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
		SetStallTimeoutMinutes(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-01c", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetWorkflowID(workflow.ID).
		SetIdentifier("ASE-401C").
		SetTitle("Pause repeated stalls").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		SetAttemptCount(stalledRetryPauseThreshold - 1).
		SetConsecutiveErrors(stalledRetryPauseThreshold - 1).
		SetStallCount(stalledRetryPauseThreshold - 1).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusExecuting, now.Add(-2*time.Minute))

	checker := newTestHealthChecker(client, now)
	report, err := checker.Run(ctx)
	if err != nil {
		t.Fatalf("run health checker: %v", err)
	}
	if report.StalledClaims != 1 || report.AgentsReleased != 1 {
		t.Fatalf("expected paused threshold stall release, got %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.StallCount != stalledRetryPauseThreshold {
		t.Fatalf("expected stall count %d, got %d", stalledRetryPauseThreshold, ticketAfter.StallCount)
	}
	if ticketAfter.AttemptCount != stalledRetryPauseThreshold ||
		ticketAfter.ConsecutiveErrors != stalledRetryPauseThreshold {
		t.Fatalf(
			"expected paused stall to increment attempts/errors to threshold, got attempts=%d errors=%d",
			ticketAfter.AttemptCount,
			ticketAfter.ConsecutiveErrors,
		)
	}
	if !ticketAfter.RetryPaused {
		t.Fatalf("expected retries to be paused after threshold stall, got %+v", ticketAfter)
	}
	if ticketAfter.PauseReason != ticketing.PauseReasonRepeatedStalls.String() {
		t.Fatalf("expected repeated stall pause reason %q, got %q", ticketing.PauseReasonRepeatedStalls, ticketAfter.PauseReason)
	}
	if ticketAfter.NextRetryAt != nil {
		t.Fatalf("expected paused threshold stall to clear next retry, got %+v", ticketAfter.NextRetryAt)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusErrored {
		t.Fatalf("expected threshold stalled run errored, got %+v", runAfter)
	}
	if runAfter.LastError == "" || runAfter.LastError == "runtime stalled or heartbeat missing" {
		t.Fatalf("expected paused threshold stall to enrich last error, got %q", runAfter.LastError)
	}

	activityItems, err := client.ActivityEvent.Query().
		Where(
			entactivityevent.TicketIDEQ(ticketItem.ID),
			entactivityevent.EventTypeEQ(stalledRetryPauseEventType.String()),
		).
		All(ctx)
	if err != nil {
		t.Fatalf("list pause activity events: %v", err)
	}
	if len(activityItems) != 1 {
		t.Fatalf("expected one pause activity event, got %+v", activityItems)
	}
	if activityItems[0].Message == "" {
		t.Fatal("expected pause activity event message")
	}
	if activityItems[0].Metadata["pause_reason"] != ticketing.PauseReasonRepeatedStalls.String() {
		t.Fatalf("expected pause activity metadata, got %+v", activityItems[0].Metadata)
	}
}

func TestHealthCheckerLeavesHealthyClaimUntouched(t *testing.T) {
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
		SetStallTimeoutMinutes(5).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-02", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetWorkflowID(workflow.ID).
		SetIdentifier("ASE-402").
		SetTitle("Healthy claim").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityMedium).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusLaunching, now.Add(-30*time.Second))

	checker := newTestHealthChecker(client, now)
	report, err := checker.Run(ctx)
	if err != nil {
		t.Fatalf("run health checker: %v", err)
	}

	if report.ClaimsChecked != 1 || report.StalledClaims != 0 || report.AgentsReleased != 0 {
		t.Fatalf("unexpected health report: %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID == nil || *ticketAfter.CurrentRunID != runItem.ID {
		t.Fatalf("expected current run to stay %s, got %+v", runItem.ID, ticketAfter.CurrentRunID)
	}
	if ticketAfter.StallCount != 0 || ticketAfter.NextRetryAt != nil {
		t.Fatalf("expected healthy ticket unchanged, got %+v", ticketAfter)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.RuntimeControlState != "active" {
		t.Fatalf("expected claimed agent control state unchanged, got %+v", agentAfter)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusLaunching {
		t.Fatalf("expected run unchanged, got %+v", runAfter)
	}
}

func TestHealthCheckerLeavesFreshMissingHeartbeatLaunchUntouched(t *testing.T) {
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
		SetStallTimeoutMinutes(5).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-02b", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetWorkflowID(workflow.ID).
		SetIdentifier("ASE-402B").
		SetTitle("Fresh missing heartbeat launch").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityMedium).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	checker := newTestHealthChecker(client, now)
	report, err := checker.Run(ctx)
	if err != nil {
		t.Fatalf("run health checker: %v", err)
	}

	if report.ClaimsChecked != 1 || report.StalledClaims != 0 || report.AgentsReleased != 0 {
		t.Fatalf("unexpected health report: %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID == nil || *ticketAfter.CurrentRunID != runItem.ID {
		t.Fatalf("expected current run to stay %s, got %+v", runItem.ID, ticketAfter.CurrentRunID)
	}
}

func TestHealthCheckerTreatsMissingHeartbeatAsStalled(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Now().UTC().Add(2 * time.Minute)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		SetStallTimeoutMinutes(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-03", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetWorkflowID(workflow.ID).
		SetIdentifier("ASE-403").
		SetTitle("Missing heartbeat").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusExecuting, time.Time{})

	checker := newTestHealthChecker(client, now)
	report, err := checker.Run(ctx)
	if err != nil {
		t.Fatalf("run health checker: %v", err)
	}

	if report.StalledClaims != 1 || report.AgentsReleased != 1 {
		t.Fatalf("expected missing heartbeat to stall, got %+v", report)
	}
	if got := backlogStageActiveRuns(ctx, t, client, fixture.projectID); got != 0 {
		t.Fatalf("expected missing-heartbeat recovery to drop backlog stage occupancy to 0, got %d", got)
	}
}

func TestHealthCheckerSkipsRegistryManagedRunAndLeavesFallbackToLauncher(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		SetStallTimeoutMinutes(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-runtime-managed", 0)
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetWorkflowID(workflow.ID).
		SetIdentifier("ASE-404").
		SetTitle("Registry-managed runtime").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, ticketItem.ID, entagentrun.StatusExecuting, now.Add(-2*time.Minute))

	checker := newTestHealthChecker(client, now)
	runtimeState := NewRuntimeStateStore()
	checker.ConfigureRuntimeState(runtimeState)
	runtimeState.markReady(runItem.ID, agentItem.ID, ticketItem.ID, workflow.ID, "thread-runtime-1", now.Add(-2*time.Minute))
	runtimeState.recordCodexEvent(runItem.ID, string(codex.EventTypeOutputProduced), now.Add(-30*time.Second))

	report, err := checker.Run(ctx)
	if err != nil {
		t.Fatalf("run health checker: %v", err)
	}

	if report.ClaimsChecked != 1 || report.StalledClaims != 0 || report.AgentsReleased != 0 {
		t.Fatalf("expected registry-managed run to skip fallback release, got %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID == nil || *ticketAfter.CurrentRunID != runItem.ID {
		t.Fatalf("expected current run to remain attached, got %+v", ticketAfter.CurrentRunID)
	}
}

func newTestHealthChecker(client *ent.Client, now time.Time) *HealthChecker {
	checker := NewHealthChecker(client, nil)
	checker.now = func() time.Time {
		return now
	}
	return checker
}

func mustCreateCurrentRun(
	ctx context.Context,
	t *testing.T,
	client *ent.Client,
	agentItem *ent.Agent,
	workflowID uuid.UUID,
	ticketID uuid.UUID,
	status entagentrun.Status,
	lastHeartbeat time.Time,
) *ent.AgentRun {
	t.Helper()

	builder := client.AgentRun.Create().
		SetAgentID(agentItem.ID).
		SetWorkflowID(workflowID).
		SetTicketID(ticketID).
		SetProviderID(agentItem.ProviderID).
		SetStatus(status)
	if !lastHeartbeat.IsZero() {
		builder.SetLastHeartbeatAt(lastHeartbeat)
	}
	runItem, err := builder.Save(ctx)
	if err != nil {
		t.Fatalf("create agent run: %v", err)
	}
	if _, err := client.Ticket.UpdateOneID(ticketID).
		SetCurrentRunID(runItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("attach current run: %v", err)
	}
	return runItem
}
