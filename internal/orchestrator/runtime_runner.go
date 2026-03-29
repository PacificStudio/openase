package orchestrator

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/BetterAndBetterII/openase/internal/infra/adapter/codex"
	"github.com/BetterAndBetterII/openase/internal/provider"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

const (
	defaultRuntimeMaxTurns = 10
	continuationRetryDelay = time.Second
)

type tokenUsageHighWater struct {
	inputTokens  int64
	outputTokens int64
}

func (l *RuntimeLauncher) startReadyExecutions(ctx context.Context) error {
	if l == nil || l.client == nil {
		return nil
	}

	assignments, err := l.listAssignments(ctx,
		entticket.CurrentRunIDNotNil(),
		entticket.HasCurrentRunWith(
			entagentrun.HasAgentWith(
				entagent.RuntimeControlStateEQ(entagent.RuntimeControlStateActive),
			),
			entagentrun.StatusEQ(entagentrun.StatusReady),
		),
	)
	if err != nil {
		return fmt.Errorf("list ready agents awaiting execution: %w", err)
	}

	for _, assignment := range assignments {
		if l.loadSession(assignment.run.ID) == nil {
			continue
		}
		if !l.beginExecution(assignment.run.ID) {
			continue
		}
		executingCount, err := l.client.AgentRun.Update().
			Where(
				entagentrun.IDEQ(assignment.run.ID),
				entagentrun.StatusEQ(entagentrun.StatusReady),
			).
			SetStatus(entagentrun.StatusExecuting).
			Save(ctx)
		if err != nil {
			l.finishExecution(assignment.run.ID)
			return fmt.Errorf("mark run %s executing: %w", assignment.run.ID, err)
		}
		if executingCount == 0 {
			l.finishExecution(assignment.run.ID)
			continue
		}

		//nolint:gosec // runtime executions intentionally continue asynchronously after the launcher tick claims the run.
		go l.runReadyExecution(ctx, assignment.run.ID)
	}

	return nil
}

func (l *RuntimeLauncher) runReadyExecution(ctx context.Context, runID uuid.UUID) {
	defer l.finishExecution(runID)
	session := l.loadSession(runID)
	if session == nil {
		return
	}

	highWater := tokenUsageHighWater{}
	lastError := ""
	var lastState runtimeExecutionState
	for turnNumber := 1; turnNumber <= defaultRuntimeMaxTurns; turnNumber++ {
		state, prompt, err := l.loadExecutionState(ctx, runID, turnNumber, lastError)
		if err != nil {
			l.logger.Error("load execution state", "run_id", runID, "error", err)
			stopSession(context.Background(), session)
			l.deleteSession(runID)
			return
		}
		lastState = state

		if err := l.markTicketStarted(ctx, state.ticket.ID); err != nil {
			l.logger.Warn("mark ticket started", "ticket_id", state.ticket.ID, "error", err)
		}

		turn, err := session.StartTurn(ctx, codex.TurnConfig{}, prompt)
		if err != nil {
			l.handleExecutionFailure(ctx, state.run.ID, state.agent.ID, state.ticket.ID, err)
			return
		}
		if err := l.recordAgentStep(ctx, state.agent.ProjectID, state.agent.ID, state.ticket.ID, state.run.ID, "planning", fmt.Sprintf("Started turn %d.", turnNumber), nil); err != nil {
			l.logger.Warn("record agent planning step", "run_id", state.run.ID, "error", err)
		}

		if err := l.consumeTurn(ctx, state.agent.ProjectID, state.agent.ID, state.run.ID, state.ticket.ID, session, turn.TurnID, &highWater); err != nil {
			lastError = err.Error()
			l.handleExecutionFailure(ctx, state.run.ID, state.agent.ID, state.ticket.ID, err)
			return
		}

		reloaded, err := l.reloadExecutionTicket(ctx, state.ticket.ID)
		if err != nil {
			l.logger.Error("reload execution ticket", "agent_id", state.agent.ID, "ticket_id", state.ticket.ID, "error", err)
			stopSession(context.Background(), session)
			l.deleteSession(runID)
			return
		}

		if !shouldContinueExecution(reloaded, state.run.ID) {
			if err := l.finishResolvedExecution(ctx, state.run.ID, state.agent.ID, reloaded); err != nil {
				l.handleExecutionFailure(ctx, state.run.ID, state.agent.ID, reloaded.ID, err)
			}
			return
		}
	}

	if err := l.scheduleContinuation(ctx, lastState.run.ID, lastState.agent.ID, lastState.ticket.ID); err != nil {
		l.logger.Error("schedule continuation", "run_id", lastState.run.ID, "agent_id", lastState.agent.ID, "error", err)
	}
}

type runtimeExecutionState struct {
	agent         *ent.Agent
	run           *ent.AgentRun
	ticket        *ent.Ticket
	launchContext runtimeLaunchContext
}

func (l *RuntimeLauncher) loadExecutionState(ctx context.Context, runID uuid.UUID, turnNumber int, lastError string) (runtimeExecutionState, string, error) {
	assignment, err := l.loadAssignmentByRun(ctx, runID)
	if err != nil {
		return runtimeExecutionState{}, "", err
	}
	if assignment.agent == nil || assignment.ticket == nil || assignment.run == nil {
		return runtimeExecutionState{}, "", fmt.Errorf("run %s no longer has an active assignment", runID)
	}

	launchContext, err := l.loadLaunchContext(ctx, assignment.agent.ID, assignment.ticket.ID)
	if err != nil {
		return runtimeExecutionState{}, "", err
	}

	machine, remote, err := l.resolveLaunchMachine(ctx, launchContext)
	if err != nil {
		return runtimeExecutionState{}, "", err
	}

	workspace, err := buildWorkspacePath(launchContext, machine, remote)
	if err != nil {
		return runtimeExecutionState{}, "", err
	}
	if workspace == "" {
		return runtimeExecutionState{}, "", fmt.Errorf("run %s workspace path must not be empty", runID)
	}

	var prompt string
	if turnNumber == 1 {
		prompt, err = l.buildDeveloperInstructions(ctx, launchContext, machine, workspace)
		if err != nil {
			return runtimeExecutionState{}, "", err
		}
	} else {
		prompt = buildContinuationPrompt(launchContext.ticket, turnNumber, defaultRuntimeMaxTurns, lastError)
	}

	return runtimeExecutionState{
		agent:         launchContext.agent,
		run:           assignment.run,
		ticket:        launchContext.ticket,
		launchContext: launchContext,
	}, prompt, nil
}

func buildContinuationPrompt(ticket *ent.Ticket, turnNumber int, maxTurns int, lastError string) string {
	var builder strings.Builder
	builder.WriteString("Continuation guidance:\n\n")
	builder.WriteString("- The previous Codex turn completed, but the ticket is still active.\n")
	builder.WriteString(fmt.Sprintf("- This is continuation turn #%d of %d.\n", turnNumber, maxTurns))
	builder.WriteString("- Resume from the current workspace and thread context.\n")
	builder.WriteString("- Do not restate the original task before acting.\n")
	if ticket != nil {
		builder.WriteString(fmt.Sprintf("- Continue working on ticket %s: %s.\n", ticket.Identifier, ticket.Title))
	}
	if trimmed := strings.TrimSpace(lastError); trimmed != "" {
		builder.WriteString(fmt.Sprintf("- Address the latest blocker or failure if it is still relevant: %s\n", trimmed))
	}
	return strings.TrimSpace(builder.String())
}

func (l *RuntimeLauncher) consumeTurn(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	runID uuid.UUID,
	ticketID uuid.UUID,
	session *codex.Session,
	turnID string,
	highWater *tokenUsageHighWater,
) error {
	outputItemsWithDelta := map[string]struct{}{}

	for {
		event, ok := <-session.Events()
		if !ok {
			return fmt.Errorf("codex session closed before turn %s completed", turnID)
		}

		if err := l.touchHeartbeat(ctx, runID); err != nil {
			l.logger.Warn("update agent heartbeat", "run_id", runID, "error", err)
		}

		switch event.Type {
		case codex.EventTypeToolCallRequested:
			if event.ToolCall == nil {
				continue
			}
			if err := l.recordAgentToolCall(ctx, projectID, agentID, ticketID, runID, event.ToolCall); err != nil {
				return err
			}
			if err := session.RespondToolCall(ctx, *event.ToolCall, codex.ToolCallResult{
				Success: false,
				ContentItems: []codex.ToolCallContentItem{
					{
						Type: codex.ToolCallContentTypeText,
						Text: "OpenASE orchestrated Codex sessions do not expose dynamic tools.",
					},
				},
			}); err != nil {
				return fmt.Errorf("respond tool call for turn %s: %w", turnID, err)
			}
		case codex.EventTypeTokenUsageUpdated:
			if event.TokenUsage == nil {
				continue
			}
			if event.TokenUsage.TurnID != "" && event.TokenUsage.TurnID != turnID {
				continue
			}
			if err := l.recordTokenUsage(ctx, agentID, ticketID, event.TokenUsage, highWater); err != nil {
				return err
			}
		case codex.EventTypeOutputProduced:
			if event.Output == nil {
				continue
			}
			if event.Output.TurnID != "" && event.Output.TurnID != turnID {
				continue
			}
			itemID := strings.TrimSpace(event.Output.ItemID)
			if itemID != "" && !event.Output.Snapshot {
				outputItemsWithDelta[itemID] = struct{}{}
			}
			if itemID != "" && event.Output.Snapshot {
				if _, ok := outputItemsWithDelta[itemID]; ok {
					continue
				}
			}
			if err := l.recordAgentOutput(ctx, projectID, agentID, ticketID, runID, event.Output); err != nil {
				return err
			}
		case codex.EventTypeTurnFailed:
			if event.Turn == nil || event.Turn.TurnID != turnID {
				continue
			}
			if event.Turn.Error == nil {
				return fmt.Errorf("codex turn %s failed", turnID)
			}
			return fmt.Errorf("codex turn %s failed: %s", turnID, strings.TrimSpace(event.Turn.Error.Message))
		case codex.EventTypeTurnCompleted:
			if event.Turn == nil || event.Turn.TurnID != turnID {
				continue
			}
			return nil
		}
	}
}

func (l *RuntimeLauncher) recordAgentOutput(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	output *codex.OutputEvent,
) error {
	if l == nil || output == nil {
		return nil
	}

	text := strings.TrimSpace(output.Text)
	if text == "" {
		return nil
	}

	metadata := map[string]any{
		"provider": "codex",
		"run_id":   runID.String(),
	}
	if itemID := strings.TrimSpace(output.ItemID); itemID != "" {
		metadata["item_id"] = itemID
	}
	if turnID := strings.TrimSpace(output.TurnID); turnID != "" {
		metadata["turn_id"] = turnID
	}
	if phase := strings.TrimSpace(output.Phase); phase != "" {
		metadata["phase"] = phase
	}
	if output.Snapshot {
		metadata["snapshot"] = true
	}

	traceItem, err := publishAgentTraceEvent(ctx, l.client, l.events, agentTraceEventInput{
		ProjectID:   projectID,
		AgentID:     agentID,
		TicketID:    ticketID,
		AgentRunID:  runID,
		Provider:    "codex",
		Kind:        agentTraceKindForOutput(output),
		Stream:      output.Stream,
		Text:        text,
		Payload:     metadata,
		EventType:   agentOutputType,
		PublishedAt: l.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("record agent output for run %s: %w", runID, err)
	}
	if stepStatus, stepSummary, ok := agentStepFromOutput(output, text); ok {
		traceID := traceItem.ID
		if err := l.recordAgentStep(ctx, projectID, agentID, ticketID, runID, stepStatus, stepSummary, &traceID); err != nil {
			return fmt.Errorf("record agent step for run %s: %w", runID, err)
		}
	}

	return nil
}

func (l *RuntimeLauncher) recordAgentToolCall(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	request *codex.ToolCallRequest,
) error {
	if l == nil || request == nil {
		return nil
	}

	toolName := strings.TrimSpace(request.Tool)
	metadata := map[string]any{
		"provider": "codex",
		"run_id":   runID.String(),
		"tool":     toolName,
	}
	if callID := strings.TrimSpace(request.CallID); callID != "" {
		metadata["call_id"] = callID
	}
	if turnID := strings.TrimSpace(request.TurnID); turnID != "" {
		metadata["turn_id"] = turnID
	}
	if threadID := strings.TrimSpace(request.ThreadID); threadID != "" {
		metadata["thread_id"] = threadID
	}

	traceItem, err := publishAgentTraceEvent(ctx, l.client, l.events, agentTraceEventInput{
		ProjectID:   projectID,
		AgentID:     agentID,
		TicketID:    ticketID,
		AgentRunID:  runID,
		Provider:    "codex",
		Kind:        catalogdomain.AgentTraceKindToolCallStarted,
		Stream:      "tool",
		Text:        toolName,
		Payload:     metadata,
		PublishedAt: l.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("record tool call trace for run %s: %w", runID, err)
	}
	traceID := traceItem.ID
	if err := l.recordAgentStep(ctx, projectID, agentID, ticketID, runID, "running_tool", toolCallStepSummary(toolName), &traceID); err != nil {
		return fmt.Errorf("record tool call step for run %s: %w", runID, err)
	}

	return nil
}

func (l *RuntimeLauncher) recordAgentStep(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	stepStatus string,
	summary string,
	sourceTraceEventID *uuid.UUID,
) error {
	if l == nil {
		return nil
	}

	if err := publishAgentStepEvent(ctx, l.client, l.events, projectID, agentID, ticketID, runID, stepStatus, summary, sourceTraceEventID, l.now().UTC()); err != nil {
		return err
	}
	return nil
}

func agentTraceKindForOutput(output *codex.OutputEvent) string {
	if output == nil {
		return catalogdomain.AgentTraceKindAssistantDelta
	}

	switch output.Stream {
	case "command":
		if output.Snapshot {
			return catalogdomain.AgentTraceKindCommandSnapshot
		}
		return catalogdomain.AgentTraceKindCommandDelta
	default:
		if output.Snapshot {
			return catalogdomain.AgentTraceKindAssistantSnapshot
		}
		return catalogdomain.AgentTraceKindAssistantDelta
	}
}

func agentStepFromOutput(output *codex.OutputEvent, text string) (string, string, bool) {
	if output == nil {
		return "", "", false
	}
	if phase := strings.TrimSpace(output.Phase); phase != "" {
		return phase, summarizeAgentStepText(text), true
	}
	switch strings.TrimSpace(output.Stream) {
	case "command":
		return "running_command", summarizeAgentStepText(text), true
	case "assistant":
		return "responding", summarizeAgentStepText(text), true
	default:
		return "", "", false
	}
}

func summarizeAgentStepText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	line := strings.Split(trimmed, "\n")[0]
	if len(line) <= 140 {
		return line
	}
	return strings.TrimSpace(line[:140]) + "..."
}

func toolCallStepSummary(toolName string) string {
	if strings.TrimSpace(toolName) == "" {
		return "Running provider tool call."
	}
	return "Running provider tool " + strconv.Quote(toolName) + "."
}

func (l *RuntimeLauncher) touchHeartbeat(ctx context.Context, runID uuid.UUID) error {
	_, err := l.client.AgentRun.Update().
		Where(
			entagentrun.IDEQ(runID),
			entagentrun.StatusEQ(entagentrun.StatusExecuting),
		).
		SetLastHeartbeatAt(l.now().UTC()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("touch heartbeat for run %s: %w", runID, err)
	}
	return nil
}

func (l *RuntimeLauncher) recordTokenUsage(
	ctx context.Context,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	usage *codex.TokenUsageEvent,
	highWater *tokenUsageHighWater,
) error {
	if l == nil || l.tickets == nil || usage == nil || highWater == nil {
		return nil
	}

	inputDelta := usage.TotalInputTokens - highWater.inputTokens
	outputDelta := usage.TotalOutputTokens - highWater.outputTokens
	if inputDelta < 0 {
		inputDelta = 0
	}
	if outputDelta < 0 {
		outputDelta = 0
	}

	if inputDelta == 0 && outputDelta == 0 {
		return nil
	}

	highWater.inputTokens = maxInt64(highWater.inputTokens, usage.TotalInputTokens)
	highWater.outputTokens = maxInt64(highWater.outputTokens, usage.TotalOutputTokens)

	result, err := l.tickets.RecordUsage(ctx, ticketservice.RecordUsageInput{
		AgentID:  agentID,
		TicketID: ticketID,
		Usage: ticketing.RawUsageDelta{
			InputTokens:  int64Pointer(inputDelta),
			OutputTokens: int64Pointer(outputDelta),
		},
	}, provider.NewNoopMetricsProvider())
	if err != nil {
		return fmt.Errorf("record token usage for ticket %s: %w", ticketID, err)
	}
	if result.BudgetExceeded {
		return fmt.Errorf("ticket %s exceeded its budget during execution", ticketID)
	}

	return nil
}

func (l *RuntimeLauncher) reloadExecutionTicket(ctx context.Context, ticketID uuid.UUID) (*ent.Ticket, error) {
	return l.client.Ticket.Query().
		Where(entticket.IDEQ(ticketID)).
		WithCurrentRun().
		WithWorkflow(func(query *ent.WorkflowQuery) {
			query.WithPickupStatuses()
			query.WithFinishStatuses()
		}).
		WithStatus().
		Only(ctx)
}

func shouldContinueExecution(ticket *ent.Ticket, runID uuid.UUID) bool {
	if ticket == nil || ticket.WorkflowID == nil || ticket.CurrentRunID == nil || ticket.Edges.CurrentRun == nil {
		return false
	}
	if ticket.Edges.CurrentRun.ID != runID {
		return false
	}
	return ticket.Edges.Workflow != nil &&
		slices.Contains(ticketStatusIDs(ticket.Edges.Workflow.Edges.PickupStatuses), ticket.StatusID) &&
		!ticket.RetryPaused
}

func (l *RuntimeLauncher) finishResolvedExecution(ctx context.Context, runID uuid.UUID, agentID uuid.UUID, ticket *ent.Ticket) error {
	stopSession(context.Background(), l.loadSession(runID))
	l.deleteSession(runID)

	if ticket != nil && ticket.WorkflowID != nil && (ticket.Edges.Workflow == nil || len(ticket.Edges.Workflow.Edges.FinishStatuses) == 0) {
		reloadedTicket, err := l.reloadExecutionTicket(ctx, ticket.ID)
		if err != nil {
			return fmt.Errorf("reload ticket %s for completion: %w", ticket.ID, err)
		}
		ticket = reloadedTicket
	}

	tx, err := l.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start finish execution tx: %w", err)
	}
	defer rollback(tx)

	now := l.now().UTC()
	finishStatusID, err := resolveWorkflowFinishStatus(ticket)
	if err != nil {
		return err
	}

	ticketUpdate := tx.Ticket.UpdateOneID(ticket.ID)
	if ticket.CurrentRunID != nil {
		ticketUpdate.ClearCurrentRunID()
	}
	if ticket.StatusID != finishStatusID {
		ticketUpdate.SetStatusID(finishStatusID)
	}
	if ticket.CompletedAt == nil {
		ticketUpdate.SetCompletedAt(now)
	}
	if _, err := ticketUpdate.Save(ctx); err != nil {
		return fmt.Errorf("update ticket %s after execution: %w", ticket.ID, err)
	}

	agentUpdate := tx.Agent.Update().
		Where(entagent.IDEQ(agentID)).
		SetRuntimeControlState(entagent.RuntimeControlStateActive)
	agentUpdate.AddTotalTicketsCompleted(1)
	if _, err := agentUpdate.Save(ctx); err != nil {
		return fmt.Errorf("update agent %s after execution: %w", agentID, err)
	}

	if _, err := clearRuntimeStateOne(
		tx.AgentRun.UpdateOneID(runID).
			SetStatus(entagentrun.StatusCompleted),
	).Save(ctx); err != nil {
		return fmt.Errorf("update run %s after execution: %w", runID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit finish execution tx: %w", err)
	}

	agentItem, err := loadAgentLifecycleState(ctx, l.client, agentID, &runID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentTerminatedType,
		agentItem,
		lifecycleMessage(agentTerminatedType, agentItem.agent.Name),
		runtimeEventMetadataForState(agentItem),
		now,
	)
	return nil
}

func resolveWorkflowFinishStatus(ticket *ent.Ticket) (uuid.UUID, error) {
	if ticket == nil {
		return uuid.UUID{}, fmt.Errorf("ticket missing workflow for completion")
	}
	if ticket.Edges.Workflow == nil {
		return uuid.UUID{}, fmt.Errorf("ticket %s missing workflow for completion", ticket.ID)
	}

	finishStatusIDs := ticketStatusIDs(ticket.Edges.Workflow.Edges.FinishStatuses)
	switch len(finishStatusIDs) {
	case 0:
		return uuid.UUID{}, fmt.Errorf("workflow %s has no finish statuses configured", ticket.Edges.Workflow.ID)
	case 1:
		return finishStatusIDs[0], nil
	default:
		if slices.Contains(finishStatusIDs, ticket.StatusID) {
			return ticket.StatusID, nil
		}
		return uuid.UUID{}, fmt.Errorf(
			"workflow %s requires an explicit finish status selection from the configured finish set",
			ticket.Edges.Workflow.ID,
		)
	}
}

func (l *RuntimeLauncher) handleExecutionFailure(ctx context.Context, runID uuid.UUID, agentID uuid.UUID, ticketID uuid.UUID, failure error) {
	stopSession(context.Background(), l.loadSession(runID))
	l.deleteSession(runID)

	now := l.now().UTC()
	if _, err := l.client.AgentRun.Update().
		Where(entagentrun.IDEQ(runID)).
		SetStatus(entagentrun.StatusErrored).
		SetLastError(strings.TrimSpace(failure.Error())).
		Save(ctx); err == nil {
		if failedAgent, err := loadAgentLifecycleState(ctx, l.client, agentID, &runID); err == nil {
			l.publishLifecycleEvent(
				ctx,
				agentFailedType,
				failedAgent,
				lifecycleMessage(agentFailedType, failedAgent.agent.Name),
				runtimeEventMetadataForState(failedAgent),
				now,
			)
		}
	}

	retrySvc := NewRetryService(l.client, l.logger)
	retrySvc.now = l.now
	if _, err := retrySvc.MarkAttemptFailed(ctx, ticketID); err != nil {
		l.logger.Error("mark execution failed retry", "ticket_id", ticketID, "agent_id", agentID, "error", err)
	}
}

func (l *RuntimeLauncher) scheduleContinuation(ctx context.Context, runID uuid.UUID, agentID uuid.UUID, ticketID uuid.UUID) error {
	session := l.loadSession(runID)
	stopSession(context.Background(), session)
	l.deleteSession(runID)

	tx, err := l.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start continuation tx: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.Ticket.Update().
		Where(
			entticket.IDEQ(ticketID),
			entticket.CurrentRunIDEQ(runID),
		).
		ClearCurrentRunID().
		SetNextRetryAt(l.now().UTC().Add(continuationRetryDelay)).
		SetRetryPaused(false).
		Save(ctx); err != nil {
		return fmt.Errorf("schedule ticket continuation: %w", err)
	}

	if _, err := tx.Agent.UpdateOneID(agentID).
		SetRuntimeControlState(entagent.RuntimeControlStateActive).
		Save(ctx); err != nil {
		return fmt.Errorf("reset agent %s after continuation limit: %w", agentID, err)
	}

	if _, err := clearRuntimeStateOne(
		tx.AgentRun.UpdateOneID(runID).
			SetStatus(entagentrun.StatusTerminated),
	).Save(ctx); err != nil {
		return fmt.Errorf("reset run %s after continuation limit: %w", runID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit continuation tx: %w", err)
	}

	reloaded, err := loadAgentLifecycleState(ctx, l.client, agentID, &runID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentTerminatedType,
		reloaded,
		lifecycleMessage(agentTerminatedType, reloaded.agent.Name),
		runtimeEventMetadataForState(reloaded),
		l.now().UTC(),
	)
	return nil
}

func (l *RuntimeLauncher) markTicketStarted(ctx context.Context, ticketID uuid.UUID) error {
	_, err := l.client.Ticket.Update().
		Where(
			entticket.IDEQ(ticketID),
			entticket.StartedAtIsNil(),
		).
		SetStartedAt(l.now().UTC()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark ticket %s started: %w", ticketID, err)
	}
	return nil
}

func int64Pointer(value int64) *int64 {
	return &value
}

func maxInt64(left int64, right int64) int64 {
	if right > left {
		return right
	}
	return left
}
