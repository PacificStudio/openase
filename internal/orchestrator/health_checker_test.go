package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
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
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
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
		SetAssignedAgentID(agentItem.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := client.Agent.UpdateOneID(agentItem.ID).
		SetStatus(entagent.StatusRunning).
		SetCurrentTicketID(ticketItem.ID).
		SetLastHeartbeatAt(now.Add(-2 * time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("mark agent running: %v", err)
	}

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
	if ticketAfter.AssignedAgentID != nil {
		t.Fatalf("expected assigned agent to be cleared, got %+v", ticketAfter.AssignedAgentID)
	}
	if ticketAfter.StallCount != 1 {
		t.Fatalf("expected stall count 1, got %d", ticketAfter.StallCount)
	}
	if ticketAfter.NextRetryAt == nil || !ticketAfter.NextRetryAt.Equal(now.Add(stalledRetryDelay)) {
		t.Fatalf("expected next retry at %s, got %+v", now.Add(stalledRetryDelay), ticketAfter.NextRetryAt)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.Status != entagent.StatusIdle || agentAfter.CurrentTicketID != nil {
		t.Fatalf("expected agent returned to idle, got %+v", agentAfter)
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
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
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
		SetAssignedAgentID(agentItem.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := client.Agent.UpdateOneID(agentItem.ID).
		SetStatus(entagent.StatusClaimed).
		SetCurrentTicketID(ticketItem.ID).
		SetLastHeartbeatAt(now.Add(-30 * time.Second)).
		Save(ctx); err != nil {
		t.Fatalf("mark agent claimed: %v", err)
	}

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
	if ticketAfter.AssignedAgentID == nil || *ticketAfter.AssignedAgentID != agentItem.ID {
		t.Fatalf("expected assigned agent to stay %s, got %+v", agentItem.ID, ticketAfter.AssignedAgentID)
	}
	if ticketAfter.StallCount != 0 || ticketAfter.NextRetryAt != nil {
		t.Fatalf("expected healthy ticket unchanged, got %+v", ticketAfter)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.Status != entagent.StatusClaimed || agentAfter.CurrentTicketID == nil || *agentAfter.CurrentTicketID != ticketItem.ID {
		t.Fatalf("expected claimed agent unchanged, got %+v", agentAfter)
	}
}

func TestHealthCheckerTreatsMissingHeartbeatAsStalled(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 15, 0, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		SetStallTimeoutMinutes(1).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
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
		SetAssignedAgentID(agentItem.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	if _, err := client.Agent.UpdateOneID(agentItem.ID).
		SetStatus(entagent.StatusRunning).
		SetCurrentTicketID(ticketItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("mark agent running without heartbeat: %v", err)
	}

	checker := newTestHealthChecker(client, now)
	report, err := checker.Run(ctx)
	if err != nil {
		t.Fatalf("run health checker: %v", err)
	}

	if report.StalledClaims != 1 || report.AgentsReleased != 1 {
		t.Fatalf("expected missing heartbeat to stall, got %+v", report)
	}
}

func newTestHealthChecker(client *ent.Client, now time.Time) *HealthChecker {
	checker := NewHealthChecker(client, nil)
	checker.now = func() time.Time {
		return now
	}
	return checker
}
