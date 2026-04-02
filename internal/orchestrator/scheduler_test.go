package orchestrator

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	domaincatalog "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	"github.com/google/uuid"
)

func TestSchedulerRunTickMatchesWorkflowPickupAcrossStatuses(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	codingWorkflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(3).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create coding workflow: %v", err)
	}
	reviewWorkflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Review").
		SetType(entworkflow.TypeTest).
		SetHarnessPath(".openase/harnesses/review.md").
		SetMaxConcurrent(3).
		AddPickupStatusIDs(fixture.statusIDs["In Review"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create review workflow: %v", err)
	}

	codingAgent := fixture.createAgent(ctx, t, "coding-01", 0)
	reviewAgent := fixture.createAgent(ctx, t, "review-01", 1)
	fixture.createAgent(ctx, t, "general-01", 10)
	if _, err := client.Workflow.UpdateOneID(codingWorkflow.ID).SetAgentID(codingAgent.ID).Save(ctx); err != nil {
		t.Fatalf("bind coding workflow agent: %v", err)
	}
	if _, err := client.Workflow.UpdateOneID(reviewWorkflow.ID).SetAgentID(reviewAgent.ID).Save(ctx); err != nil {
		t.Fatalf("bind review workflow agent: %v", err)
	}

	codingTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-101").
		SetTitle("Implement scheduler").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		SetCreatedAt(now.Add(-2 * time.Hour)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create coding ticket: %v", err)
	}
	reviewTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-102").
		SetTitle("Review scheduler").
		SetStatusID(fixture.statusIDs["In Review"]).
		SetPriority(entticket.PriorityMedium).
		SetCreatedBy("user:test").
		SetCreatedAt(now.Add(-time.Hour)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create review ticket: %v", err)
	}
	if _, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-103").
		SetTitle("Future retry").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityUrgent).
		SetCreatedBy("user:test").
		SetNextRetryAt(now.Add(30 * time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("create future retry ticket: %v", err)
	}

	scheduler := newTestScheduler(client, now)
	report, err := scheduler.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}

	if report.WorkflowsScanned != 2 || report.CandidatesScanned != 2 || report.TicketsDispatched != 2 {
		t.Fatalf("unexpected tick report: %+v", report)
	}

	codingTicketAfter, err := client.Ticket.Get(ctx, codingTicket.ID)
	if err != nil {
		t.Fatalf("reload coding ticket: %v", err)
	}
	if codingTicketAfter.WorkflowID == nil || *codingTicketAfter.WorkflowID != codingWorkflow.ID {
		t.Fatalf("expected coding workflow claim %s, got %+v", codingWorkflow.ID, codingTicketAfter.WorkflowID)
	}
	if codingTicketAfter.CurrentRunID == nil {
		t.Fatalf("expected coding current run claim, got %+v", codingTicketAfter)
	}
	codingRunAfter, err := client.AgentRun.Get(ctx, *codingTicketAfter.CurrentRunID)
	if err != nil {
		t.Fatalf("reload coding run: %v", err)
	}
	if codingRunAfter.AgentID != codingAgent.ID {
		t.Fatalf("expected coding run agent %s, got %s", codingAgent.ID, codingRunAfter.AgentID)
	}

	reviewTicketAfter, err := client.Ticket.Get(ctx, reviewTicket.ID)
	if err != nil {
		t.Fatalf("reload review ticket: %v", err)
	}
	if reviewTicketAfter.WorkflowID == nil || *reviewTicketAfter.WorkflowID != reviewWorkflow.ID {
		t.Fatalf("expected review workflow claim %s, got %+v", reviewWorkflow.ID, reviewTicketAfter.WorkflowID)
	}
	if reviewTicketAfter.CurrentRunID == nil {
		t.Fatalf("expected review current run claim, got %+v", reviewTicketAfter)
	}
	reviewRunAfter, err := client.AgentRun.Get(ctx, *reviewTicketAfter.CurrentRunID)
	if err != nil {
		t.Fatalf("reload review run: %v", err)
	}
	if reviewRunAfter.AgentID != reviewAgent.ID {
		t.Fatalf("expected review run agent %s, got %s", reviewAgent.ID, reviewRunAfter.AgentID)
	}
}

func TestSchedulerRunTickDispatchesActiveWorkflowOutsideInProgressProjects(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 10, 5, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	if _, err := client.Project.UpdateOneID(fixture.projectID).SetStatus("Planned").Save(ctx); err != nil {
		t.Fatalf("set project status planned: %v", err)
	}

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Backlog Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Backlog"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflowItem.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-110").
		SetTitle("Dispatch from planned project").
		SetStatusID(fixture.statusIDs["Backlog"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	report, err := newTestScheduler(client, now).RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 1 {
		t.Fatalf("expected planned project workflow to dispatch, got %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID == nil || ticketAfter.WorkflowID == nil || *ticketAfter.WorkflowID != workflowItem.ID {
		t.Fatalf("expected ticket to be claimed by workflow, got %+v", ticketAfter)
	}
}

func TestSchedulerRunTickUsesMatchedPickupStatusForStatusCapacityInMultiPickupWorkflow(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 10, 30, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	if _, err := client.TicketStatus.UpdateOneID(fixture.statusIDs["In Review"]).SetMaxActiveRuns(1).Save(ctx); err != nil {
		t.Fatalf("limit review status: %v", err)
	}

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(3).
		AddPickupStatusIDs(fixture.statusIDs["Todo"], fixture.statusIDs["In Review"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflowItem.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}
	reviewWorkflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Review Capacity Holder").
		SetType(entworkflow.TypeTest).
		SetHarnessPath(".openase/harnesses/review.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["In Review"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create review workflow: %v", err)
	}
	reviewAgent := fixture.createAgent(ctx, t, "review-01", 1)
	if _, err := client.Workflow.UpdateOneID(reviewWorkflow.ID).SetAgentID(reviewAgent.ID).Save(ctx); err != nil {
		t.Fatalf("bind review workflow agent: %v", err)
	}

	reviewTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-104").
		SetTitle("Review slot occupied").
		SetStatusID(fixture.statusIDs["In Review"]).
		SetWorkflowID(reviewWorkflow.ID).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create review ticket: %v", err)
	}
	_ = mustCreateCurrentRun(ctx, t, client, reviewAgent, reviewWorkflow.ID, reviewTicket.ID, entagentrun.StatusExecuting, now)

	todoTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-105").
		SetTitle("Todo slot should still dispatch").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create todo ticket: %v", err)
	}

	report, err := newTestScheduler(client, now).RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 1 {
		t.Fatalf("expected todo pickup to dispatch despite review status being full, got %+v", report)
	}

	todoAfter, err := client.Ticket.Get(ctx, todoTicket.ID)
	if err != nil {
		t.Fatalf("reload todo ticket: %v", err)
	}
	if todoAfter.CurrentRunID == nil || todoAfter.WorkflowID == nil || *todoAfter.WorkflowID != workflowItem.ID {
		t.Fatalf("expected todo ticket to be claimed by multi-pickup workflow, got %+v", todoAfter)
	}
}

func TestSchedulerRunTickSkipsBlockedTickets(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 11, 0, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	if _, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx); err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	if _, err := client.Workflow.Update().
		Where(entworkflow.ProjectIDEQ(fixture.projectID), entworkflow.NameEQ("Coding")).
		SetAgentID(agentItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	blocker, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-201").
		SetTitle("Blocked prerequisite").
		SetStatusID(fixture.statusIDs["In Progress"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create blocker ticket: %v", err)
	}
	target, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-202").
		SetTitle("Blocked target").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create target ticket: %v", err)
	}
	if _, err := client.TicketDependency.Create().
		SetSourceTicketID(blocker.ID).
		SetTargetTicketID(target.ID).
		SetType(entticketdependency.TypeBlocks).
		Save(ctx); err != nil {
		t.Fatalf("create dependency: %v", err)
	}

	scheduler := newTestScheduler(client, now)
	report, err := scheduler.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}

	if report.TicketsDispatched != 0 || report.TicketsSkipped[skipReasonBlocked] != 1 {
		t.Fatalf("expected blocked skip, got %+v", report)
	}

	targetAfter, err := client.Ticket.Get(ctx, target.ID)
	if err != nil {
		t.Fatalf("reload target ticket: %v", err)
	}
	if targetAfter.CurrentRunID != nil || targetAfter.WorkflowID != nil {
		t.Fatalf("expected blocked ticket to stay unclaimed, got %+v", targetAfter)
	}
}

func TestSchedulerRunTickHonorsConcurrencyLimits(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	busyAgent := fixture.createAgent(ctx, t, "busy-01", 0)
	idleAgent := fixture.createAgent(ctx, t, "idle-01", 1)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(busyAgent.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	runningTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-301").
		SetTitle("Already running").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		SetWorkflowID(workflow.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create running ticket: %v", err)
	}
	mustCreateCurrentRun(ctx, t, client, busyAgent, workflow.ID, runningTicket.ID, entagentrun.StatusExecuting, now)

	target, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-302").
		SetTitle("Waiting for capacity").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityUrgent).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create target ticket: %v", err)
	}

	scheduler := newTestScheduler(client, now)
	report, err := scheduler.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}

	if report.TicketsDispatched != 0 || report.TicketsSkipped[skipReasonMaxConcurrency] != 1 {
		t.Fatalf("expected max_concurrency skip, got %+v", report)
	}

	targetAfter, err := client.Ticket.Get(ctx, target.ID)
	if err != nil {
		t.Fatalf("reload target ticket: %v", err)
	}
	if targetAfter.CurrentRunID != nil || targetAfter.WorkflowID != nil {
		t.Fatalf("expected ticket to remain unclaimed, got %+v", targetAfter)
	}

	idleAgentAfter, err := client.Agent.Get(ctx, idleAgent.ID)
	if err != nil {
		t.Fatalf("reload idle agent: %v", err)
	}
	if idleAgentAfter.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("expected idle agent unchanged, got %+v", idleAgentAfter)
	}
	if activeRunCount, err := client.AgentRun.Query().Where(
		entagentrun.AgentIDEQ(idleAgent.ID),
		entagentrun.HasCurrentForTicket(),
	).Count(ctx); err != nil {
		t.Fatalf("count active current runs for idle agent: %v", err)
	} else if activeRunCount != 0 {
		t.Fatalf("expected idle agent to remain idle, got %d current runs", activeRunCount)
	}
}

func TestSchedulerRunTickAllowsConcurrentClaimsForSingleAgentDefinition(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 12, 15, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

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

	agentItem := fixture.createAgent(ctx, t, "parallel-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	runningTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-302A").
		SetTitle("Already running on shared agent").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		SetWorkflowID(workflow.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create running ticket: %v", err)
	}
	mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, runningTicket.ID, entagentrun.StatusExecuting, now)

	target, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-302B").
		SetTitle("Concurrent claim on same agent definition").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityUrgent).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create target ticket: %v", err)
	}

	scheduler := newTestScheduler(client, now)
	report, err := scheduler.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 1 {
		t.Fatalf("expected second ticket to dispatch onto the same agent definition, got %+v", report)
	}

	targetAfter, err := client.Ticket.Get(ctx, target.ID)
	if err != nil {
		t.Fatalf("reload target ticket: %v", err)
	}
	if targetAfter.CurrentRunID == nil {
		t.Fatalf("expected target ticket to be claimed, got %+v", targetAfter)
	}

	runAfter, err := client.AgentRun.Get(ctx, *targetAfter.CurrentRunID)
	if err != nil {
		t.Fatalf("reload target run: %v", err)
	}
	if runAfter.AgentID != agentItem.ID {
		t.Fatalf("expected shared agent %s, got %s", agentItem.ID, runAfter.AgentID)
	}

	activeRunCount, err := client.AgentRun.Query().
		Where(
			entagentrun.AgentIDEQ(agentItem.ID),
			entagentrun.HasCurrentForTicket(),
		).
		Count(ctx)
	if err != nil {
		t.Fatalf("count active current runs: %v", err)
	}
	if activeRunCount != 2 {
		t.Fatalf("expected same agent definition to hold 2 concurrent current runs, got %d", activeRunCount)
	}
}

func TestSchedulerRunTickSkipsBusyProviderWhenParallelRunLimitReached(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 12, 20, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).SetMaxParallelRuns(1).Save(ctx); err != nil {
		t.Fatalf("set provider max parallel runs: %v", err)
	}

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

	agentItem := fixture.createAgent(ctx, t, "provider-bound-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	runningTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-302C").
		SetTitle("Already running against provider").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		SetWorkflowID(workflow.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create running ticket: %v", err)
	}
	mustCreateCurrentRun(ctx, t, client, agentItem, workflow.ID, runningTicket.ID, entagentrun.StatusExecuting, now)

	target, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-302D").
		SetTitle("Blocked by provider semaphore").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityUrgent).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create target ticket: %v", err)
	}

	report, err := newTestScheduler(client, now).RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 0 || report.TicketsSkipped[skipReasonProviderBusy] != 1 {
		t.Fatalf("expected provider_busy skip, got %+v", report)
	}

	targetAfter, err := client.Ticket.Get(ctx, target.ID)
	if err != nil {
		t.Fatalf("reload target ticket: %v", err)
	}
	if targetAfter.CurrentRunID != nil || targetAfter.WorkflowID != nil {
		t.Fatalf("expected ticket to remain unclaimed, got %+v", targetAfter)
	}
}

func TestSchedulerRunTickTreatsZeroConcurrencyLimitsAsUnlimited(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 12, 25, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	if _, err := client.Project.UpdateOneID(fixture.projectID).SetMaxConcurrentAgents(0).Save(ctx); err != nil {
		t.Fatalf("set project max concurrent agents: %v", err)
	}
	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).SetMaxParallelRuns(0).Save(ctx); err != nil {
		t.Fatalf("set provider max parallel runs: %v", err)
	}

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(0).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "provider-bound-unlimited", 0)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	for i, identifier := range []string{"ASE-302U1", "ASE-302U2", "ASE-302U3"} {
		if _, err := client.Ticket.Create().
			SetProjectID(fixture.projectID).
			SetIdentifier(identifier).
			SetTitle("Unlimited slot test").
			SetStatusID(fixture.statusIDs["Todo"]).
			SetPriority(entticket.PriorityUrgent).
			SetCreatedBy("user:test").
			SetCreatedAt(now.Add(time.Duration(i) * time.Minute)).
			Save(ctx); err != nil {
			t.Fatalf("create ticket %s: %v", identifier, err)
		}
	}

	report, err := newTestScheduler(client, now).RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 3 {
		t.Fatalf("expected all tickets to dispatch with unlimited limits, got %+v", report)
	}
	if report.TicketsSkipped[skipReasonMaxConcurrency] != 0 || report.TicketsSkipped[skipReasonProviderBusy] != 0 {
		t.Fatalf("expected no concurrency-limit skips, got %+v", report)
	}

	activeRunCount, err := client.Ticket.Query().
		Where(
			entticket.ProjectIDEQ(fixture.projectID),
			entticket.WorkflowIDEQ(workflow.ID),
			entticket.CurrentRunIDNotNil(),
		).
		Count(ctx)
	if err != nil {
		t.Fatalf("count active claimed tickets: %v", err)
	}
	if activeRunCount != 3 {
		t.Fatalf("expected 3 active claimed tickets, got %d", activeRunCount)
	}
}

func TestSchedulerRunTickPublishesClaimedLifecycleAndClearsRuntimeState(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 12, 30, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	bus := eventinfra.NewChannelBus()
	stream, err := bus.Subscribe(ctx, agentLifecycleTopic)
	if err != nil {
		t.Fatalf("subscribe agent lifecycle topic: %v", err)
	}

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-01").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-303").
		SetTitle("Fresh claim").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	scheduler := NewScheduler(client, slog.New(slog.NewTextHandler(io.Discard, nil)), bus)
	scheduler.now = func() time.Time {
		return now
	}

	report, err := scheduler.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 1 {
		t.Fatalf("expected one dispatch, got %+v", report)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("expected active runtime control state, got %+v", agentAfter)
	}

	event := waitForSchedulerEvent(t, stream, agentClaimedType)
	if event.Type != agentClaimedType {
		t.Fatalf("expected agent.claimed event, got %s", event.Type)
	}

	var payload agentLifecycleEnvelope
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("decode event payload: %v", err)
	}
	if payload.Agent.ID != agentItem.ID.String() ||
		payload.Agent.Status != "claimed" ||
		payload.Agent.RuntimePhase != "launching" ||
		payload.Agent.SessionID != "" ||
		payload.Agent.CurrentRunID == nil ||
		payload.Agent.CurrentTicketID == nil {
		t.Fatalf("unexpected lifecycle payload: %+v", payload.Agent)
	}
	if ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID); err != nil {
		t.Fatalf("reload ticket: %v", err)
	} else if ticketAfter.WorkflowID == nil || *ticketAfter.WorkflowID != workflow.ID {
		t.Fatalf("expected ticket workflow %s, got %+v", workflow.ID, ticketAfter.WorkflowID)
	} else if ticketAfter.CurrentRunID == nil {
		t.Fatalf("expected ticket current run after dispatch, got %+v", ticketAfter)
	} else if ticketAfter.TargetMachineID == nil {
		t.Fatalf("expected ticket machine binding, got %+v", ticketAfter)
	} else if runAfter, err := client.AgentRun.Get(ctx, *ticketAfter.CurrentRunID); err != nil {
		t.Fatalf("reload agent run: %v", err)
	} else if runAfter.AgentID != agentItem.ID {
		t.Fatalf("expected current run agent %s, got %s", agentItem.ID, runAfter.AgentID)
	} else if runAfter.Status != entagentrun.StatusLaunching || runAfter.SessionID != "" || runAfter.RuntimeStartedAt != nil || runAfter.LastHeartbeatAt != nil || runAfter.LastError != "" {
		t.Fatalf("expected clean launching run after dispatch, got %+v", runAfter)
	}
}

func TestSchedulerRunTickResolvesExecutionMachineFromBoundProvider(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 13, 0, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	repo := catalogrepo.NewEntRepository(client)
	remoteMachineInput, err := domaincatalog.ParseCreateMachine(fixture.orgID, domaincatalog.MachineInput{
		Name:          "gpu-01",
		Host:          "10.0.1.10",
		SSHUser:       stringPointer("openase"),
		SSHKeyPath:    stringPointer("/tmp/gpu-01.pem"),
		Labels:        []string{"gpu", "a100"},
		WorkspaceRoot: stringPointer("/srv/openase/workspaces"),
		Status:        entmachine.StatusOnline.String(),
	})
	if err != nil {
		t.Fatalf("parse remote machine: %v", err)
	}
	remoteMachine, err := repo.CreateMachine(ctx, remoteMachineInput)
	if err != nil {
		t.Fatalf("create remote machine: %v", err)
	}
	if _, err := client.Machine.UpdateOneID(remoteMachine.ID).
		SetResources(map[string]any{
			"monitor": map[string]any{
				"l4": codexL4Snapshot(now.Add(-5*time.Minute), domaincatalog.MachineAgentAuthStatusLoggedIn, true),
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("seed remote machine l4 snapshot: %v", err)
	}

	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(remoteMachine.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider machine: %v", err)
	}
	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Training").
		SetType(entworkflow.TypeCustom).
		SetHarnessPath(".openase/harnesses/training.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "trainer-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-401").
		SetTitle("Run GPU experiment").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	report, err := newTestScheduler(client, now).RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 1 {
		t.Fatalf("expected one dispatch, got %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.TargetMachineID == nil || *ticketAfter.TargetMachineID != remoteMachine.ID {
		t.Fatalf("expected ticket target machine %s, got %+v", remoteMachine.ID, ticketAfter.TargetMachineID)
	}
	if ticketAfter.CurrentRunID == nil {
		t.Fatalf("expected current run for claimed ticket, got %+v", ticketAfter)
	}
	if ticketAfter.WorkflowID == nil || *ticketAfter.WorkflowID != workflow.ID {
		t.Fatalf("expected workflow %s, got %+v", workflow.ID, ticketAfter.WorkflowID)
	}
	runAfter, err := client.AgentRun.Get(ctx, *ticketAfter.CurrentRunID)
	if err != nil {
		t.Fatalf("reload claimed run: %v", err)
	}
	if runAfter.AgentID != agentItem.ID {
		t.Fatalf("expected claimed run agent %s, got %s", agentItem.ID, runAfter.AgentID)
	}
}

func TestSchedulerRunTickIgnoresTicketTargetMachineAndUsesProviderBinding(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 13, 30, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	repo := catalogrepo.NewEntRepository(client)
	explicitMachineInput, err := domaincatalog.ParseCreateMachine(fixture.orgID, domaincatalog.MachineInput{
		Name:          "cpu-01",
		Host:          "10.0.2.10",
		SSHUser:       stringPointer("openase"),
		SSHKeyPath:    stringPointer("/tmp/cpu-01.pem"),
		Labels:        []string{"cpu"},
		WorkspaceRoot: stringPointer("/srv/openase/workspaces"),
		Status:        entmachine.StatusOnline.String(),
	})
	if err != nil {
		t.Fatalf("parse explicit machine: %v", err)
	}
	explicitMachine, err := repo.CreateMachine(ctx, explicitMachineInput)
	if err != nil {
		t.Fatalf("create explicit machine: %v", err)
	}

	if _, err := client.AgentProvider.UpdateOneID(fixture.providerID).
		SetMachineID(fixture.localMachineID).
		Save(ctx); err != nil {
		t.Fatalf("bind provider machine: %v", err)
	}

	if _, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx); err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	if _, err := client.Workflow.Update().
		Where(entworkflow.ProjectIDEQ(fixture.projectID), entworkflow.NameEQ("Coding")).
		SetAgentID(agentItem.ID).
		Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-402").
		SetTitle("Force CPU worker").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		SetTargetMachineID(explicitMachine.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	report, err := newTestScheduler(client, now).RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 1 {
		t.Fatalf("expected one dispatch, got %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.TargetMachineID == nil || *ticketAfter.TargetMachineID != fixture.localMachineID {
		t.Fatalf("expected provider-bound machine %s, got %+v", fixture.localMachineID, ticketAfter.TargetMachineID)
	}
}

func TestSchedulerRunTickSkipsUnavailableProvider(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 13, 45, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	if _, err := client.Machine.UpdateOneID(fixture.localMachineID).
		SetResources(map[string]any{
			"transport":    "local",
			"last_success": true,
			"monitor": map[string]any{
				"l4": codexL4Snapshot(now.Add(-5*time.Minute), domaincatalog.MachineAgentAuthStatusNotLoggedIn, false),
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("set unavailable provider snapshot: %v", err)
	}

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-405").
		SetTitle("Unavailable provider should not dispatch").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	report, err := newTestScheduler(client, now).RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 0 || report.TicketsSkipped[skipReasonProviderUnavailable] != 1 {
		t.Fatalf("expected provider_unavailable skip, got %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID != nil || ticketAfter.WorkflowID != nil {
		t.Fatalf("expected ticket to remain unclaimed, got %+v", ticketAfter)
	}
}

func TestSchedulerRunTickSkipsStaleProvider(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 13, 50, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	if _, err := client.Machine.UpdateOneID(fixture.localMachineID).
		SetResources(map[string]any{
			"transport":    "local",
			"last_success": true,
			"monitor": map[string]any{
				"l4": codexL4Snapshot(now.Add(-domaincatalog.ProviderAvailabilityStaleAfter-time.Minute), domaincatalog.MachineAgentAuthStatusLoggedIn, true),
			},
		}).
		Save(ctx); err != nil {
		t.Fatalf("set stale provider snapshot: %v", err)
	}

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-406").
		SetTitle("Stale provider should not dispatch").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	report, err := newTestScheduler(client, now).RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 0 || report.TicketsSkipped[skipReasonProviderStale] != 1 {
		t.Fatalf("expected provider_stale skip, got %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID != nil || ticketAfter.WorkflowID != nil {
		t.Fatalf("expected ticket to remain unclaimed, got %+v", ticketAfter)
	}
}

func TestSchedulerRunTickSkipsWorkflowWhenBoundAgentIsPaused(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	if _, err := client.Agent.UpdateOneID(agentItem.ID).
		SetRuntimeControlState(entagent.RuntimeControlStatePaused).
		Save(ctx); err != nil {
		t.Fatalf("pause agent: %v", err)
	}

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetAgentID(agentItem.ID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-403").
		SetTitle("Wait for bound agent").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	report, err := newTestScheduler(client, now).RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 0 || report.TicketsSkipped[skipReasonNoAgent] != 1 {
		t.Fatalf("expected bound paused agent to skip dispatch, got %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID != nil || ticketAfter.WorkflowID != nil {
		t.Fatalf("expected ticket to remain unclaimed when bound agent is paused, got %+v", ticketAfter)
	}
	if workflowAfter, err := client.Workflow.Get(ctx, workflow.ID); err != nil {
		t.Fatalf("reload workflow: %v", err)
	} else if workflowAfter.AgentID == nil || *workflowAfter.AgentID != agentItem.ID {
		t.Fatalf("expected workflow to stay bound to %s, got %+v", agentItem.ID, workflowAfter.AgentID)
	}
}

func TestSchedulerRunTickSkipsWorkflowWhenBoundAgentIsMissing(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 14, 30, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	agentItem := fixture.createAgent(ctx, t, "coding-01", 0)
	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetAgentID(agentItem.ID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	if _, err := client.Workflow.UpdateOneID(workflow.ID).ClearAgentID().Save(ctx); err != nil {
		t.Fatalf("clear workflow agent binding: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-404").
		SetTitle("Bound agent record removed").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	report, err := newTestScheduler(client, now).RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}
	if report.TicketsDispatched != 0 || report.TicketsSkipped[skipReasonNoAgent] != 1 {
		t.Fatalf("expected missing bound agent to skip dispatch, got %+v", report)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.CurrentRunID != nil || ticketAfter.WorkflowID != nil {
		t.Fatalf("expected ticket to remain unclaimed when bound agent is missing, got %+v", ticketAfter)
	}
	if workflowAfter, err := client.Workflow.Get(ctx, workflow.ID); err != nil {
		t.Fatalf("reload workflow: %v", err)
	} else if workflowAfter.AgentID != nil {
		t.Fatalf("expected workflow binding to be missing, got %+v", workflowAfter.AgentID)
	}
}

func waitForSchedulerEvent(t *testing.T, stream <-chan provider.Event, want provider.EventType) provider.Event {
	t.Helper()

	select {
	case event := <-stream:
		return event
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", want)
		return provider.Event{}
	}
}

type projectFixture struct {
	client         *ent.Client
	orgID          uuid.UUID
	projectID      uuid.UUID
	providerID     uuid.UUID
	localMachineID uuid.UUID
	statusIDs      map[string]uuid.UUID
}

func TestSchedulerRunTickCreatesDueScheduledJobTicketsBeforeDispatch(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Security").
		SetType(entworkflow.TypeSecurity).
		SetHarnessPath(".openase/harnesses/security.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "security-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	job, err := client.ScheduledJob.Create().
		SetProjectID(fixture.projectID).
		SetName("weekly-security-scan").
		SetCronExpression("0 9 * * 1").
		SetWorkflowID(workflow.ID).
		SetTicketTemplate(map[string]any{
			"title":      "Weekly security scan - {{ date }}",
			"status":     "Todo",
			"priority":   "high",
			"type":       "feature",
			"created_by": "system:scheduled-job",
		}).
		SetIsEnabled(true).
		SetNextRunAt(now.Add(-time.Minute)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create scheduled job: %v", err)
	}

	scheduler := newTestScheduler(client, now)
	report, err := scheduler.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}

	if report.ScheduledJobsScanned != 1 || report.ScheduledTicketsCreated != 1 {
		t.Fatalf("expected one due scheduled job to create one ticket, got %+v", report)
	}
	if report.WorkflowsScanned != 1 || report.CandidatesScanned != 1 || report.TicketsDispatched != 1 {
		t.Fatalf("expected created ticket to enter workflow dispatch in the same tick, got %+v", report)
	}

	createdTicket, err := client.Ticket.Query().
		Where(entticket.ProjectIDEQ(fixture.projectID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("load created ticket: %v", err)
	}
	if createdTicket.Title != "Weekly security scan - 2026-03-20" {
		t.Fatalf("expected rendered scheduled ticket title, got %+v", createdTicket)
	}
	if createdTicket.WorkflowID == nil || *createdTicket.WorkflowID != workflow.ID {
		t.Fatalf("expected created ticket to bind workflow %s, got %+v", workflow.ID, createdTicket.WorkflowID)
	}
	if createdTicket.CurrentRunID == nil {
		t.Fatalf("expected created ticket to be dispatched, got %+v", createdTicket)
	}
	createdRun, err := client.AgentRun.Get(ctx, *createdTicket.CurrentRunID)
	if err != nil {
		t.Fatalf("reload created run: %v", err)
	}
	if createdRun.AgentID != agentItem.ID {
		t.Fatalf("expected created run agent %s, got %s", agentItem.ID, createdRun.AgentID)
	}

	jobAfter, err := client.ScheduledJob.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("reload scheduled job: %v", err)
	}
	if jobAfter.LastRunAt == nil || !jobAfter.LastRunAt.Equal(now) {
		t.Fatalf("expected scheduled job last_run_at to update to %s, got %+v", now, jobAfter.LastRunAt)
	}
	if jobAfter.NextRunAt == nil || !jobAfter.NextRunAt.After(now) {
		t.Fatalf("expected scheduled job next_run_at to advance beyond %s, got %+v", now, jobAfter.NextRunAt)
	}
}

func TestSchedulerRunTickEnforcesSharedStatusCapacityAcrossWorkflows(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 12, 15, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	if _, err := client.TicketStatus.UpdateOneID(fixture.statusIDs["Backlog"]).SetMaxActiveRuns(1).Save(ctx); err != nil {
		t.Fatalf("set backlog status capacity: %v", err)
	}

	backlogWorkflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Backlog triage").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/backlog.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Backlog"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create backlog workflow: %v", err)
	}
	todoWorkflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Todo implementation").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/todo.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Backlog"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create todo workflow: %v", err)
	}

	backlogAgent := fixture.createAgent(ctx, t, "backlog-01", 0)
	todoAgent := fixture.createAgent(ctx, t, "todo-01", 0)
	if _, err := client.Workflow.UpdateOneID(backlogWorkflow.ID).SetAgentID(backlogAgent.ID).Save(ctx); err != nil {
		t.Fatalf("bind backlog workflow agent: %v", err)
	}
	if _, err := client.Workflow.UpdateOneID(todoWorkflow.ID).SetAgentID(todoAgent.ID).Save(ctx); err != nil {
		t.Fatalf("bind todo workflow agent: %v", err)
	}

	if _, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-351").
		SetTitle("Backlog semaphore contender").
		SetStatusID(fixture.statusIDs["Backlog"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx); err != nil {
		t.Fatalf("create backlog ticket: %v", err)
	}
	if _, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-352").
		SetTitle("Backlog semaphore contender 2").
		SetStatusID(fixture.statusIDs["Backlog"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx); err != nil {
		t.Fatalf("create second backlog ticket: %v", err)
	}

	scheduler := newTestScheduler(client, now)
	report, err := scheduler.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}

	if report.TicketsDispatched != 1 || report.TicketsSkipped[skipReasonStatusCapacity] != 2 {
		t.Fatalf("expected one dispatch and two shared status-capacity skips across workflows, got %+v", report)
	}
	if got := backlogStageActiveRuns(ctx, t, client, fixture.projectID); got != 1 {
		t.Fatalf("expected backlog status active runs 1, got %d", got)
	}
}

func TestSchedulerRunTickContinuesWhenOneDueScheduledJobFails(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	now := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)
	fixture := seedProjectFixtureAt(ctx, t, client, now)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Security").
		SetType(entworkflow.TypeSecurity).
		SetHarnessPath(".openase/harnesses/security.md").
		SetMaxConcurrent(2).
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "security-01", 0)
	if _, err := client.Workflow.UpdateOneID(workflow.ID).SetAgentID(agentItem.ID).Save(ctx); err != nil {
		t.Fatalf("bind workflow agent: %v", err)
	}

	if _, err := client.ScheduledJob.Create().
		SetProjectID(fixture.projectID).
		SetName("broken-weekly-security-scan").
		SetCronExpression("0 9 * * 1").
		SetWorkflowID(workflow.ID).
		SetTicketTemplate(map[string]any{}).
		SetIsEnabled(true).
		SetNextRunAt(now.Add(-2 * time.Minute)).
		Save(ctx); err != nil {
		t.Fatalf("create broken scheduled job: %v", err)
	}

	job, err := client.ScheduledJob.Create().
		SetProjectID(fixture.projectID).
		SetName("weekly-security-scan").
		SetCronExpression("0 9 * * 1").
		SetWorkflowID(workflow.ID).
		SetTicketTemplate(map[string]any{
			"title":      "Weekly security scan - {{ date }}",
			"status":     "Todo",
			"priority":   "high",
			"type":       "feature",
			"created_by": "system:scheduled-job",
		}).
		SetIsEnabled(true).
		SetNextRunAt(now.Add(-time.Minute)).
		Save(ctx)
	if err != nil {
		t.Fatalf("create scheduled job: %v", err)
	}

	scheduler := newTestScheduler(client, now)
	report, err := scheduler.RunTick(ctx)
	if err != nil {
		t.Fatalf("run tick: %v", err)
	}

	if report.ScheduledJobsScanned != 2 || report.ScheduledTicketsCreated != 1 {
		t.Fatalf("expected one of two due scheduled jobs to succeed, got %+v", report)
	}
	if report.WorkflowsScanned != 1 || report.CandidatesScanned != 1 || report.TicketsDispatched != 1 {
		t.Fatalf("expected valid scheduled job ticket to dispatch in same tick, got %+v", report)
	}

	createdTickets, err := client.Ticket.Query().
		Where(entticket.ProjectIDEQ(fixture.projectID)).
		All(ctx)
	if err != nil {
		t.Fatalf("load created tickets: %v", err)
	}
	if len(createdTickets) != 1 {
		t.Fatalf("expected exactly one created ticket, got %d", len(createdTickets))
	}
	if createdTickets[0].CurrentRunID == nil {
		t.Fatalf("expected created ticket to dispatch, got %+v", createdTickets[0])
	}
	createdRun, err := client.AgentRun.Get(ctx, *createdTickets[0].CurrentRunID)
	if err != nil {
		t.Fatalf("reload created scheduled run: %v", err)
	}
	if createdRun.AgentID != agentItem.ID {
		t.Fatalf("expected created scheduled run agent %s, got %s", agentItem.ID, createdRun.AgentID)
	}

	jobAfter, err := client.ScheduledJob.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("reload scheduled job: %v", err)
	}
	if jobAfter.LastRunAt == nil || !jobAfter.LastRunAt.Equal(now) {
		t.Fatalf("expected successful scheduled job last_run_at to update to %s, got %+v", now, jobAfter.LastRunAt)
	}
}

func TestSchedulerHelperCoverage(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	scheduler := newTestScheduler(client, time.Date(2026, 3, 27, 18, 0, 0, 0, time.UTC))

	agentItem := fixture.createAgent(ctx, t, "resolve-machine", 0)
	machine, providerItem, reason, err := scheduler.resolveExecutionMachine(ctx, fixture.orgID, agentItem, scheduler.now())
	if err != nil || reason != "" || machine == nil || machine.ID != fixture.localMachineID || providerItem == nil || providerItem.ID != fixture.providerID {
		t.Fatalf("resolveExecutionMachine(success) = %+v, %+v, %q, %v", machine, providerItem, reason, err)
	}

	if _, err := client.Machine.UpdateOneID(fixture.localMachineID).SetStatus(entmachine.StatusOffline).Save(ctx); err != nil {
		t.Fatalf("set machine offline: %v", err)
	}
	machine, providerItem, reason, err = scheduler.resolveExecutionMachine(ctx, fixture.orgID, agentItem, scheduler.now())
	if err != nil || reason != skipReasonNoMachine || machine != nil || providerItem != nil {
		t.Fatalf("resolveExecutionMachine(offline) = %+v, %+v, %q, %v", machine, providerItem, reason, err)
	}
	if _, err := client.Machine.UpdateOneID(fixture.localMachineID).SetStatus(entmachine.StatusOnline).Save(ctx); err != nil {
		t.Fatalf("restore machine online: %v", err)
	}

	otherOrg, err := client.Organization.Create().
		SetName("Other Org").
		SetSlug("other-org").
		Save(ctx)
	if err != nil {
		t.Fatalf("create other org: %v", err)
	}
	otherMachine, err := client.Machine.Create().
		SetOrganizationID(otherOrg.ID).
		SetName("other-machine").
		SetHost("10.0.0.9").
		SetPort(22).
		SetStatus(entmachine.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create other machine: %v", err)
	}
	otherProvider, err := client.AgentProvider.Create().
		SetOrganizationID(otherOrg.ID).
		SetMachineID(otherMachine.ID).
		SetName("Other Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create other provider: %v", err)
	}
	crossOrgAgent, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(otherProvider.ID).
		SetName("cross-org").
		Save(ctx)
	if err != nil {
		t.Fatalf("create cross org agent: %v", err)
	}
	machine, providerItem, reason, err = scheduler.resolveExecutionMachine(ctx, fixture.orgID, crossOrgAgent, scheduler.now())
	if err != nil || reason != skipReasonNoMachine || machine != nil || providerItem != nil {
		t.Fatalf("resolveExecutionMachine(org mismatch) = %+v, %+v, %q, %v", machine, providerItem, reason, err)
	}

	if !isDependencyResolved(&ent.Ticket{CompletedAt: timePointer(time.Now())}) {
		t.Fatal("isDependencyResolved(completedAt) expected true")
	}
	if !isDependencyResolved(&ent.Ticket{
		Edges: ent.TicketEdges{
			Status: &ent.TicketStatus{Stage: "completed"},
		},
	}) {
		t.Fatal("isDependencyResolved(completed stage) expected true")
	}
	if !isDependencyResolved(&ent.Ticket{
		Edges: ent.TicketEdges{
			Status: &ent.TicketStatus{Stage: "canceled"},
		},
	}) {
		t.Fatal("isDependencyResolved(canceled stage) expected true")
	}
	if isDependencyResolved(&ent.Ticket{StatusID: fixture.statusIDs["Todo"]}) {
		t.Fatal("isDependencyResolved(todo) expected false")
	}

	counts := map[string]int{"capacity": 1}
	mergeSkipCounts(counts, map[string]int{"capacity": 2, "blocked": 1})
	if counts["capacity"] != 3 || counts["blocked"] != 1 {
		t.Fatalf("mergeSkipCounts() = %+v", counts)
	}
	rollback(nil)
}

func seedProjectFixture(ctx context.Context, t *testing.T, client *ent.Client) projectFixture {
	t.Helper()
	return seedProjectFixtureAt(ctx, t, client, time.Now().UTC())
}

func seedProjectFixtureAt(ctx context.Context, t *testing.T, client *ent.Client, now time.Time) projectFixture {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	localMachine, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName(domaincatalog.LocalMachineName).
		SetHost(domaincatalog.LocalMachineHost).
		SetPort(22).
		SetDescription("Control-plane local execution host.").
		SetStatus(entmachine.StatusOnline).
		SetResources(map[string]any{
			"transport":    "local",
			"last_success": true,
			"monitor": map[string]any{
				"l4": codexL4Snapshot(now.Add(-5*time.Minute), domaincatalog.MachineAgentAuthStatusLoggedIn, true),
			},
		}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetStatus("In Progress").
		SetMaxConcurrentAgents(2).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetMachineID(localMachine.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		SetMaxParallelRuns(domaincatalog.DefaultAgentProviderMaxParallelRuns).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent provider: %v", err)
	}

	statusSvc := newTicketStatusService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset default statuses: %v", err)
	}

	statusIDs := make(map[string]uuid.UUID, len(statuses))
	for _, status := range statuses {
		statusIDs[status.Name] = status.ID
	}
	return projectFixture{
		client:         client,
		orgID:          org.ID,
		projectID:      project.ID,
		providerID:     provider.ID,
		localMachineID: localMachine.ID,
		statusIDs:      statusIDs,
	}
}

func (f projectFixture) createAgent(ctx context.Context, t *testing.T, name string, totalTicketsCompleted int) *ent.Agent {
	t.Helper()

	agentItem, err := f.client.Agent.Create().
		SetProjectID(f.projectID).
		SetProviderID(f.providerID).
		SetName(name).
		SetTotalTicketsCompleted(totalTicketsCompleted).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent %s: %v", name, err)
	}
	return agentItem
}

func stringPointer(value string) *string {
	return &value
}

func codexL4Snapshot(
	checkedAt time.Time,
	authStatus domaincatalog.MachineAgentAuthStatus,
	ready bool,
) map[string]any {
	return map[string]any{
		"checked_at": checkedAt.UTC().Format(time.RFC3339),
		"codex": map[string]any{
			"installed":   true,
			"auth_status": string(authStatus),
			"auth_mode":   string(domaincatalog.MachineAgentAuthModeLogin),
			"ready":       ready,
		},
	}
}

func newTestScheduler(client *ent.Client, now time.Time) *Scheduler {
	scheduler := NewScheduler(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	scheduler.now = func() time.Time {
		return now
	}
	return scheduler
}

func timePointer(value time.Time) *time.Time {
	return &value
}

func backlogStageActiveRuns(ctx context.Context, t *testing.T, client *ent.Client, projectID uuid.UUID) int {
	return statusActiveRuns(ctx, t, client, projectID, "Backlog")
}

func statusActiveRuns(ctx context.Context, t *testing.T, client *ent.Client, projectID uuid.UUID, statusName string) int {
	t.Helper()

	snapshots, err := listProjectStatusRuntimeSnapshots(ctx, client, projectID)
	if err != nil {
		t.Fatalf("list status runtime snapshots: %v", err)
	}
	for _, snapshot := range snapshots {
		if snapshot.Name == statusName {
			return snapshot.ActiveRuns
		}
	}
	t.Fatalf("%s status runtime snapshot not found", statusName)
	return 0
}

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	client := testPostgres.NewIsolatedEntClient(t)
	ticketrepo.InstallRetryTokenHooks(client)
	return client
}
