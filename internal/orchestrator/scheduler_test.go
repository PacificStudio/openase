package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	domaincatalog "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	"github.com/BetterAndBetterII/openase/internal/types/pgarray"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
)

func TestSchedulerRunTickMatchesWorkflowPickupAcrossStatuses(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)

	codingWorkflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(3).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
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
		SetPickupStatusID(fixture.statusIDs["In Review"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create review workflow: %v", err)
	}

	codingAgent := fixture.createAgent(ctx, t, "coding-01", 0)
	reviewAgent := fixture.createAgent(ctx, t, "review-01", 1)
	fixture.createAgent(ctx, t, "general-01", 10)

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
	if codingTicketAfter.AssignedAgentID == nil || *codingTicketAfter.AssignedAgentID != codingAgent.ID {
		t.Fatalf("expected coding agent claim %s, got %+v", codingAgent.ID, codingTicketAfter.AssignedAgentID)
	}

	reviewTicketAfter, err := client.Ticket.Get(ctx, reviewTicket.ID)
	if err != nil {
		t.Fatalf("reload review ticket: %v", err)
	}
	if reviewTicketAfter.WorkflowID == nil || *reviewTicketAfter.WorkflowID != reviewWorkflow.ID {
		t.Fatalf("expected review workflow claim %s, got %+v", reviewWorkflow.ID, reviewTicketAfter.WorkflowID)
	}
	if reviewTicketAfter.AssignedAgentID == nil || *reviewTicketAfter.AssignedAgentID != reviewAgent.ID {
		t.Fatalf("expected review agent claim %s, got %+v", reviewAgent.ID, reviewTicketAfter.AssignedAgentID)
	}
}

func TestSchedulerRunTickSkipsBlockedTickets(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 11, 0, 0, 0, time.UTC)

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
	fixture.createAgent(ctx, t, "coding-01", 0)

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
	if targetAfter.AssignedAgentID != nil || targetAfter.WorkflowID != nil {
		t.Fatalf("expected blocked ticket to stay unclaimed, got %+v", targetAfter)
	}
}

func TestSchedulerRunTickHonorsConcurrencyLimits(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetMaxConcurrent(1).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	busyAgent := fixture.createAgent(ctx, t, "busy-01", 0)
	idleAgent := fixture.createAgent(ctx, t, "idle-01", 1)

	runningTicket, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-301").
		SetTitle("Already running").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		SetWorkflowID(workflow.ID).
		SetAssignedAgentID(busyAgent.ID).
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
	if targetAfter.AssignedAgentID != nil || targetAfter.WorkflowID != nil {
		t.Fatalf("expected ticket to remain unclaimed, got %+v", targetAfter)
	}

	idleAgentAfter, err := client.Agent.Get(ctx, idleAgent.ID)
	if err != nil {
		t.Fatalf("reload idle agent: %v", err)
	}
	if idleAgentAfter.Status != entagent.StatusIdle || idleAgentAfter.CurrentTicketID != nil {
		t.Fatalf("expected idle agent unchanged, got %+v", idleAgentAfter)
	}
}

func TestSchedulerRunTickPublishesClaimedLifecycleAndClearsRuntimeState(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 12, 30, 0, 0, time.UTC)

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
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	staleHeartbeat := now.Add(-time.Hour)
	agentItem, err := client.Agent.Create().
		SetProjectID(fixture.projectID).
		SetProviderID(fixture.providerID).
		SetName("codex-01").
		SetStatus(entagent.StatusIdle).
		SetSessionID("stale-session").
		SetRuntimePhase(entagent.RuntimePhaseReady).
		SetRuntimeStartedAt(staleHeartbeat).
		SetLastError("stale error").
		SetLastHeartbeatAt(staleHeartbeat).
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent: %v", err)
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
	} else if ticketAfter.AssignedAgentID == nil || *ticketAfter.AssignedAgentID != agentItem.ID {
		t.Fatalf("expected ticket assigned agent %s, got %+v", agentItem.ID, ticketAfter.AssignedAgentID)
	} else if ticketAfter.WorkflowID == nil || *ticketAfter.WorkflowID != workflow.ID {
		t.Fatalf("expected ticket workflow %s, got %+v", workflow.ID, ticketAfter.WorkflowID)
	} else if ticketAfter.CurrentRunID == nil {
		t.Fatalf("expected ticket current run after dispatch, got %+v", ticketAfter)
	} else if ticketAfter.TargetMachineID == nil {
		t.Fatalf("expected ticket machine binding, got %+v", ticketAfter)
	} else if runAfter, err := client.AgentRun.Get(ctx, *ticketAfter.CurrentRunID); err != nil {
		t.Fatalf("reload agent run: %v", err)
	} else if runAfter.Status != entagentrun.StatusLaunching || runAfter.SessionID != "" || runAfter.RuntimeStartedAt != nil || runAfter.LastHeartbeatAt != nil || runAfter.LastError != "" {
		t.Fatalf("expected clean launching run after dispatch, got %+v", runAfter)
	}
}

func TestSchedulerRunTickMatchesRequiredMachineLabelsAndBindsTicket(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 0, 0, 0, time.UTC)

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

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Training").
		SetType(entworkflow.TypeCustom).
		SetHarnessPath(".openase/harnesses/training.md").
		SetRequiredMachineLabels(pgarray.StringArray{"gpu"}).
		SetMaxConcurrent(1).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "trainer-01", 0)
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
	if ticketAfter.AssignedAgentID == nil || *ticketAfter.AssignedAgentID != agentItem.ID {
		t.Fatalf("expected claimed agent %s, got %+v", agentItem.ID, ticketAfter.AssignedAgentID)
	}
	if ticketAfter.WorkflowID == nil || *ticketAfter.WorkflowID != workflow.ID {
		t.Fatalf("expected workflow %s, got %+v", workflow.ID, ticketAfter.WorkflowID)
	}
}

func TestSchedulerRunTickHonorsExplicitTargetMachineBinding(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 30, 0, 0, time.UTC)

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

	if _, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Coding").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		SetRequiredMachineLabels(pgarray.StringArray{"gpu"}).
		SetMaxConcurrent(1).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx); err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	fixture.createAgent(ctx, t, "coding-01", 0)

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
	if ticketAfter.TargetMachineID == nil || *ticketAfter.TargetMachineID != explicitMachine.ID {
		t.Fatalf("expected explicit target machine %s, got %+v", explicitMachine.ID, ticketAfter.TargetMachineID)
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
	client     *ent.Client
	orgID      uuid.UUID
	projectID  uuid.UUID
	providerID uuid.UUID
	statusIDs  map[string]uuid.UUID
}

func TestSchedulerRunTickCreatesDueScheduledJobTicketsBeforeDispatch(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Security").
		SetType(entworkflow.TypeSecurity).
		SetHarnessPath(".openase/harnesses/security.md").
		SetMaxConcurrent(2).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "security-01", 0)

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
	if createdTicket.AssignedAgentID == nil || *createdTicket.AssignedAgentID != agentItem.ID {
		t.Fatalf("expected created ticket to be dispatched to agent %s, got %+v", agentItem.ID, createdTicket.AssignedAgentID)
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

func TestSchedulerRunTickContinuesWhenOneDueScheduledJobFails(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 9, 0, 0, 0, time.UTC)

	workflow, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Security").
		SetType(entworkflow.TypeSecurity).
		SetHarnessPath(".openase/harnesses/security.md").
		SetMaxConcurrent(2).
		SetPickupStatusID(fixture.statusIDs["Todo"]).
		SetFinishStatusID(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "security-01", 0)

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
	if createdTickets[0].AssignedAgentID == nil || *createdTickets[0].AssignedAgentID != agentItem.ID {
		t.Fatalf("expected created ticket to dispatch to agent %s, got %+v", agentItem.ID, createdTickets[0].AssignedAgentID)
	}

	jobAfter, err := client.ScheduledJob.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("reload scheduled job: %v", err)
	}
	if jobAfter.LastRunAt == nil || !jobAfter.LastRunAt.Equal(now) {
		t.Fatalf("expected successful scheduled job last_run_at to update to %s, got %+v", now, jobAfter.LastRunAt)
	}
}

func seedProjectFixture(ctx context.Context, t *testing.T, client *ent.Client) projectFixture {
	t.Helper()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	if _, err := client.Machine.Create().
		SetOrganizationID(org.ID).
		SetName(domaincatalog.LocalMachineName).
		SetHost(domaincatalog.LocalMachineHost).
		SetPort(22).
		SetDescription("Control-plane local execution host.").
		SetStatus(entmachine.StatusOnline).
		SetResources(map[string]any{
			"transport":    "local",
			"last_success": true,
		}).
		Save(ctx); err != nil {
		t.Fatalf("create local machine: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetStatus(entproject.StatusActive).
		SetMaxConcurrentAgents(2).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	provider, err := client.AgentProvider.Create().
		SetOrganizationID(org.ID).
		SetName("Codex").
		SetAdapterType(entagentprovider.AdapterTypeCodexAppServer).
		SetCliCommand("codex").
		SetModelName("gpt-5.4").
		Save(ctx)
	if err != nil {
		t.Fatalf("create agent provider: %v", err)
	}

	statusSvc := ticketstatus.NewService(client)
	statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
	if err != nil {
		t.Fatalf("reset default statuses: %v", err)
	}

	statusIDs := make(map[string]uuid.UUID, len(statuses))
	for _, status := range statuses {
		statusIDs[status.Name] = status.ID
	}

	return projectFixture{
		client:     client,
		orgID:      org.ID,
		projectID:  project.ID,
		providerID: provider.ID,
		statusIDs:  statusIDs,
	}
}

func (f projectFixture) createAgent(ctx context.Context, t *testing.T, name string, totalTicketsCompleted int) *ent.Agent {
	t.Helper()

	agentItem, err := f.client.Agent.Create().
		SetProjectID(f.projectID).
		SetProviderID(f.providerID).
		SetName(name).
		SetStatus(entagent.StatusIdle).
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

func newTestScheduler(client *ent.Client, now time.Time) *Scheduler {
	scheduler := NewScheduler(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	scheduler.now = func() time.Time {
		return now
	}
	return scheduler
}

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	port := freePort(t)
	dataDir := t.TempDir()
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(port).
			Username("postgres").
			Password("postgres").
			Database("openase").
			RuntimePath(filepath.Join(dataDir, "runtime")).
			BinariesPath(filepath.Join(dataDir, "binaries")).
			DataPath(filepath.Join(dataDir, "data")),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Stop(); err != nil {
			t.Errorf("stop embedded postgres: %v", err)
		}
	})

	dsn := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/openase?sslmode=disable", port)
	client, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open ent client: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client: %v", err)
		}
	})

	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return client
}

func freePort(t *testing.T) uint32 {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate free port: %v", err)
	}

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCP address, got %T", listener.Addr())
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	if tcpAddr.Port < 0 || tcpAddr.Port > math.MaxUint16 {
		t.Fatalf("expected TCP port in uint16 range, got %d", tcpAddr.Port)
	}
	return uint32(tcpAddr.Port) //nolint:gosec // validated above to fit the TCP port range
}
