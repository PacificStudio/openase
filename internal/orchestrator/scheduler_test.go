package orchestrator

import (
	"context"
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
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
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
	if _, err := client.Agent.UpdateOneID(busyAgent.ID).
		SetStatus(entagent.StatusClaimed).
		SetCurrentTicketID(runningTicket.ID).
		Save(ctx); err != nil {
		t.Fatalf("claim busy agent: %v", err)
	}

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

type projectFixture struct {
	client     *ent.Client
	orgID      uuid.UUID
	projectID  uuid.UUID
	providerID uuid.UUID
	statusIDs  map[string]uuid.UUID
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

func newTestScheduler(client *ent.Client, now time.Time) *Scheduler {
	scheduler := NewScheduler(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
