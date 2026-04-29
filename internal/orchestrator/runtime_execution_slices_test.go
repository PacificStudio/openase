package orchestrator

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	entactivityevent "github.com/BetterAndBetterII/openase/ent/activityevent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entagentstepevent "github.com/BetterAndBetterII/openase/ent/agentstepevent"
	entagenttraceevent "github.com/BetterAndBetterII/openase/ent/agenttraceevent"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
)

func TestRuntimeExecutionSliceStartReadyExecutionsPublishesExecutingLifecycle(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	executingAt := time.Date(2026, time.April, 3, 15, 4, 5, 0, time.UTC)

	bus := eventinfra.NewChannelBus()
	defer func() {
		if err := bus.Close(); err != nil {
			t.Fatalf("bus.Close() error = %v", err)
		}
	}()
	stream, err := bus.Subscribe(ctx, agentLifecycleTopic)
	if err != nil {
		t.Fatalf("Subscribe(agent lifecycle) error = %v", err)
	}

	workflowItem, err := client.Workflow.Create().
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
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-902-SLICE").
		SetTitle("Publish executing lifecycle from execution slice").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority("high").
		SetType("feature").
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "executing-slice-01", 0)
	runItem := mustCreateCurrentRun(
		ctx,
		t,
		client,
		agentItem,
		workflowItem.ID,
		ticketItem.ID,
		entagentrun.StatusReady,
		executingAt.Add(-15*time.Second),
	)

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, nil, nil, nil)
	launcher.now = func() time.Time { return executingAt }
	launcher.storeSession(runItem.ID, runtimeRunnerFailingSession{})

	if err := launcher.executionSlice().startReadyExecutions(ctx); err != nil {
		t.Fatalf("executionSlice.startReadyExecutions() error = %v", err)
	}

	executingEvent := waitForAgentLifecycleEvent(t, stream, agentExecutingType)
	payload := decodeLifecycleEnvelope(t, executingEvent.Payload)
	if payload.Agent.ID != agentItem.ID.String() {
		t.Fatalf("executing payload agent id = %q, want %q", payload.Agent.ID, agentItem.ID.String())
	}
	if payload.Agent.RuntimePhase != "executing" {
		t.Fatalf("executing payload runtime phase = %q, want executing", payload.Agent.RuntimePhase)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusExecuting {
		t.Fatalf("run status = %s, want executing", runAfter.Status)
	}

	activities, err := client.ActivityEvent.Query().
		Where(
			entactivityevent.AgentIDEQ(agentItem.ID),
			entactivityevent.EventTypeEQ(activityevent.TypeAgentExecuting.String()),
		).
		All(ctx)
	if err != nil {
		t.Fatalf("query executing activities: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected one executing activity, got %+v", activities)
	}
}

func TestRuntimeEventPersistenceSliceRecordAgentOutputPersistsTraceAndCommentaryStep(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, time.April, 4, 11, 30, 0, 0, time.UTC)

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Commentary").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-903-SLICE").
		SetTitle("Persist commentary output").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	agentItem := fixture.createAgent(ctx, t, "event-slice-01", 0)
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, nil, nil)
	launcher.now = func() time.Time { return now }

	err = launcher.eventSlice().recordAgentOutput(ctx, fixture.projectID, agentItem.ID, ticketItem.ID, runItem.ID, entagentprovider.AdapterTypeCodexAppServer, &agentOutputEvent{
		Stream:   "assistant",
		Text:     "Investigating runtime slice commentary state.",
		Phase:    "commentary",
		Snapshot: true,
		ItemID:   "assistant-item-1",
		TurnID:   "turn-1",
	})
	if err != nil {
		t.Fatalf("eventSlice.recordAgentOutput() error = %v", err)
	}

	traceItems, err := client.AgentTraceEvent.Query().
		Where(entagenttraceevent.AgentRunIDEQ(runItem.ID)).
		All(ctx)
	if err != nil {
		t.Fatalf("query trace events: %v", err)
	}
	if len(traceItems) != 1 {
		t.Fatalf("expected one trace event, got %+v", traceItems)
	}
	if traceItems[0].Kind != catalogdomain.AgentTraceKindAssistantSnapshot {
		t.Fatalf("trace kind = %q, want %q", traceItems[0].Kind, catalogdomain.AgentTraceKindAssistantSnapshot)
	}
	if traceItems[0].Payload["phase"] != "commentary" {
		t.Fatalf("trace payload = %+v", traceItems[0].Payload)
	}

	stepItems, err := client.AgentStepEvent.Query().
		Where(entagentstepevent.AgentRunIDEQ(runItem.ID)).
		All(ctx)
	if err != nil {
		t.Fatalf("query step events: %v", err)
	}
	if len(stepItems) != 1 {
		t.Fatalf("expected one step event, got %+v", stepItems)
	}
	if stepItems[0].StepStatus != "commentary" {
		t.Fatalf("step status = %q, want commentary", stepItems[0].StepStatus)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.CurrentStepStatus == nil || *runAfter.CurrentStepStatus != "commentary" {
		t.Fatalf("run current_step_status = %+v, want commentary", runAfter.CurrentStepStatus)
	}
}

func TestRuntimeRecoverySliceReconcileRuntimeFactsSchedulesContinuationAfterCleanSessionExit(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 24, 15, 0, 0, 0, time.UTC)

	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Runtime fact reconciliation").
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
		SetIdentifier("ASE-904-SLICE").
		SetTitle("Continue after clean session exit").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "runtime-fact-slice-01", 0)
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusExecuting, now)

	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, &runtimeFakeProcessManager{}, nil, nil)
	launcher.now = func() time.Time { return now }
	runtimeState := NewRuntimeStateStore()
	launcher.ConfigureRuntimeState(runtimeState)
	runtimeState.markReady(runItem.ID, agentItem.ID, ticketItem.ID, workflowItem.ID, "thread-runtime-slice-1", now)
	runtimeState.recordTurnStart(runItem.ID, 1, now)
	runtimeState.recordRuntimeFact(runItem.ID, runtimeFactSessionExited, now, "")

	if err := launcher.recoverySlice().reconcileRuntimeFacts(ctx); err != nil {
		t.Fatalf("recoverySlice.reconcileRuntimeFacts() error = %v", err)
	}

	ticketAfter, err := client.Ticket.Get(ctx, ticketItem.ID)
	if err != nil {
		t.Fatalf("reload ticket: %v", err)
	}
	if ticketAfter.NextRetryAt == nil || !ticketAfter.NextRetryAt.UTC().Equal(now.Add(continuationRetryDelay)) {
		t.Fatalf("expected continuation retry at %s, got %+v", now.Add(continuationRetryDelay), ticketAfter.NextRetryAt)
	}
	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusTerminated {
		t.Fatalf("expected terminated run after runtime fact continuation, got %+v", runAfter)
	}
}

func TestRuntimeRecoverySliceReconcileInterruptRequestsInterruptsActiveRun(t *testing.T) {
	ctx := context.Background()
	client := openTestEntClient(t)
	fixture := seedProjectFixture(ctx, t, client)
	now := time.Date(2026, 3, 20, 13, 0, 0, 0, time.UTC)

	bus := eventinfra.NewChannelBus()
	defer func() {
		if err := bus.Close(); err != nil {
			t.Fatalf("bus.Close() error = %v", err)
		}
	}()
	stream, err := bus.Subscribe(ctx, agentLifecycleTopic)
	if err != nil {
		t.Fatalf("subscribe agent lifecycle stream: %v", err)
	}
	workflowItem, err := client.Workflow.Create().
		SetProjectID(fixture.projectID).
		SetName("Interrupt runtime").
		SetType(entworkflow.TypeCoding).
		SetHarnessPath(".openase/harnesses/coding.md").
		AddPickupStatusIDs(fixture.statusIDs["Todo"]).
		AddFinishStatusIDs(fixture.statusIDs["Done"]).
		Save(ctx)
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	ticketItem, err := client.Ticket.Create().
		SetProjectID(fixture.projectID).
		SetIdentifier("ASE-905-SLICE").
		SetTitle("Interrupt Codex runtime via recovery slice").
		SetStatusID(fixture.statusIDs["Todo"]).
		SetWorkflowID(workflowItem.ID).
		SetPriority(entticket.PriorityHigh).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	agentItem := fixture.createAgent(ctx, t, "codex-interrupt-slice-01", 0)
	runItem := mustCreateCurrentRun(ctx, t, client, agentItem, workflowItem.ID, ticketItem.ID, entagentrun.StatusLaunching, time.Time{})

	manager := &runtimeFakeProcessManager{}
	launcher := NewRuntimeLauncher(client, slog.New(slog.NewTextHandler(io.Discard, nil)), bus, manager, nil, nil)
	launcher.now = func() time.Time { return now }
	t.Cleanup(func() {
		if err := launcher.Close(context.Background()); err != nil {
			t.Errorf("close launcher: %v", err)
		}
	})

	if err := launcher.RunTick(ctx); err != nil {
		t.Fatalf("run initial launcher tick: %v", err)
	}
	waitForAgentLifecycleEvent(t, stream, agentReadyType)

	if _, err := client.Agent.UpdateOneID(agentItem.ID).
		SetRuntimeControlState(entagent.RuntimeControlStateInterruptRequested).
		Save(ctx); err != nil {
		t.Fatalf("request interrupt: %v", err)
	}

	if err := launcher.recoverySlice().reconcileInterruptRequests(ctx); err != nil {
		t.Fatalf("recoverySlice.reconcileInterruptRequests() error = %v", err)
	}

	interruptedEvent := waitForAgentLifecycleEvent(t, stream, agentInterruptedType)
	payload := decodeLifecycleEnvelope(t, interruptedEvent.Payload)
	if payload.Agent.ID != agentItem.ID.String() || payload.Agent.Status != "interrupted" {
		t.Fatalf("unexpected interrupted event payload: %+v", payload.Agent)
	}

	agentAfter, err := client.Agent.Get(ctx, agentItem.ID)
	if err != nil {
		t.Fatalf("reload agent: %v", err)
	}
	if agentAfter.RuntimeControlState != entagent.RuntimeControlStateActive {
		t.Fatalf("expected active control state after interrupt, got %s", agentAfter.RuntimeControlState)
	}

	runAfter, err := client.AgentRun.Get(ctx, runItem.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if runAfter.Status != entagentrun.StatusInterrupted {
		t.Fatalf("expected interrupted run after interrupt, got %s", runAfter.Status)
	}
}
