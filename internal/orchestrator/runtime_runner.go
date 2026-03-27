package orchestrator

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
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
		if l.loadSession(assignment.agent.ID) == nil {
			continue
		}
		if !l.beginExecution(assignment.agent.ID) {
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
			l.finishExecution(assignment.agent.ID)
			return fmt.Errorf("mark run %s executing: %w", assignment.run.ID, err)
		}
		if executingCount == 0 {
			l.finishExecution(assignment.agent.ID)
			continue
		}

		//nolint:gosec // runtime executions intentionally continue asynchronously after the launcher tick claims the run.
		go l.runReadyExecution(ctx, assignment.agent.ID)
	}

	return nil
}

func (l *RuntimeLauncher) runReadyExecution(ctx context.Context, agentID uuid.UUID) {
	defer l.finishExecution(agentID)
	session := l.loadSession(agentID)
	if session == nil {
		return
	}

	highWater := tokenUsageHighWater{}
	lastError := ""
	for turnNumber := 1; turnNumber <= defaultRuntimeMaxTurns; turnNumber++ {
		state, prompt, err := l.loadExecutionState(ctx, agentID, turnNumber, lastError)
		if err != nil {
			l.logger.Error("load execution state", "agent_id", agentID, "error", err)
			stopSession(context.Background(), session)
			l.deleteSession(agentID)
			return
		}

		if err := l.markTicketStarted(ctx, state.ticket.ID); err != nil {
			l.logger.Warn("mark ticket started", "ticket_id", state.ticket.ID, "error", err)
		}

		turn, err := session.StartTurn(ctx, codex.TurnConfig{}, prompt)
		if err != nil {
			l.handleExecutionFailure(ctx, state.agent.ID, state.ticket.ID, err)
			return
		}

		if err := l.consumeTurn(ctx, state.agent.ID, state.run.ID, state.ticket.ID, session, turn.TurnID, &highWater); err != nil {
			lastError = err.Error()
			l.handleExecutionFailure(ctx, state.agent.ID, state.ticket.ID, err)
			return
		}

		reloaded, err := l.reloadExecutionTicket(ctx, state.ticket.ID)
		if err != nil {
			l.logger.Error("reload execution ticket", "agent_id", state.agent.ID, "ticket_id", state.ticket.ID, "error", err)
			stopSession(context.Background(), session)
			l.deleteSession(agentID)
			return
		}

		if !shouldContinueExecution(reloaded, state.agent.ID) {
			if err := l.finishResolvedExecution(ctx, state.agent.ID, reloaded); err != nil {
				l.handleExecutionFailure(ctx, state.agent.ID, reloaded.ID, err)
			}
			return
		}
	}

	if err := l.scheduleContinuation(ctx, agentID); err != nil {
		l.logger.Error("schedule continuation", "agent_id", agentID, "error", err)
	}
}

type runtimeExecutionState struct {
	agent         *ent.Agent
	run           *ent.AgentRun
	ticket        *ent.Ticket
	launchContext runtimeLaunchContext
}

func (l *RuntimeLauncher) loadExecutionState(ctx context.Context, agentID uuid.UUID, turnNumber int, lastError string) (runtimeExecutionState, string, error) {
	assignment, err := l.loadAssignmentByAgent(ctx, agentID)
	if err != nil {
		return runtimeExecutionState{}, "", err
	}
	if assignment.agent == nil || assignment.ticket == nil || assignment.run == nil {
		return runtimeExecutionState{}, "", fmt.Errorf("agent %s no longer has an active run", agentID)
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
		return runtimeExecutionState{}, "", fmt.Errorf("agent %s workspace path must not be empty", agentID)
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
	agentID uuid.UUID,
	runID uuid.UUID,
	ticketID uuid.UUID,
	session *codex.Session,
	turnID string,
	highWater *tokenUsageHighWater,
) error {
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

func shouldContinueExecution(ticket *ent.Ticket, agentID uuid.UUID) bool {
	if ticket == nil || ticket.WorkflowID == nil || ticket.CurrentRunID == nil || ticket.Edges.CurrentRun == nil {
		return false
	}
	if ticket.Edges.CurrentRun.AgentID != agentID {
		return false
	}
	return ticket.Edges.Workflow != nil &&
		slices.Contains(ticketStatusIDs(ticket.Edges.Workflow.Edges.PickupStatuses), ticket.StatusID) &&
		!ticket.RetryPaused
}

func (l *RuntimeLauncher) finishResolvedExecution(ctx context.Context, agentID uuid.UUID, ticket *ent.Ticket) error {
	stopSession(context.Background(), l.loadSession(agentID))
	l.deleteSession(agentID)

	if ticket != nil && ticket.WorkflowID != nil && (ticket.Edges.Workflow == nil || len(ticket.Edges.Workflow.Edges.FinishStatuses) == 0) {
		reloadedTicket, err := l.reloadExecutionTicket(ctx, ticket.ID)
		if err != nil {
			return fmt.Errorf("reload ticket %s for completion: %w", ticket.ID, err)
		}
		ticket = reloadedTicket
	}

	assignment, err := l.loadAssignmentByAgent(ctx, agentID)
	if err != nil {
		return err
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

	if assignment.run != nil {
		if _, err := clearRuntimeStateOne(
			tx.AgentRun.UpdateOneID(assignment.run.ID).
				SetStatus(entagentrun.StatusCompleted),
		).Save(ctx); err != nil {
			return fmt.Errorf("update run %s after execution: %w", assignment.run.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit finish execution tx: %w", err)
	}

	agentItem, err := loadAgentLifecycleState(ctx, l.client, agentID)
	if err != nil {
		return err
	}
	return publishAgentLifecycleEvent(
		ctx,
		l.client,
		l.events,
		agentTerminatedType,
		agentItem,
		lifecycleMessage(agentTerminatedType, agentItem.agent.Name),
		runtimeEventMetadataForState(agentItem),
		now,
	)
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

func (l *RuntimeLauncher) handleExecutionFailure(ctx context.Context, agentID uuid.UUID, ticketID uuid.UUID, failure error) {
	stopSession(context.Background(), l.loadSession(agentID))
	l.deleteSession(agentID)

	now := l.now().UTC()
	assignment, assignmentErr := l.loadAssignmentByAgent(ctx, agentID)
	if assignmentErr != nil {
		l.logger.Error("load failed assignment", "ticket_id", ticketID, "agent_id", agentID, "error", assignmentErr)
	} else if assignment.run != nil {
		if _, err := l.client.AgentRun.Update().
			Where(entagentrun.IDEQ(assignment.run.ID)).
			SetStatus(entagentrun.StatusErrored).
			SetLastError(strings.TrimSpace(failure.Error())).
			Save(ctx); err == nil {
			if failedAgent, err := loadAgentLifecycleState(ctx, l.client, agentID); err == nil {
				_ = publishAgentLifecycleEvent(
					ctx,
					l.client,
					l.events,
					agentFailedType,
					failedAgent,
					lifecycleMessage(agentFailedType, failedAgent.agent.Name),
					runtimeEventMetadataForState(failedAgent),
					now,
				)
			}
		}
	}

	retrySvc := NewRetryService(l.client, l.logger)
	retrySvc.now = l.now
	if _, err := retrySvc.MarkAttemptFailed(ctx, ticketID); err != nil {
		l.logger.Error("mark execution failed retry", "ticket_id", ticketID, "agent_id", agentID, "error", err)
	}
}

func (l *RuntimeLauncher) scheduleContinuation(ctx context.Context, agentID uuid.UUID) error {
	session := l.loadSession(agentID)
	stopSession(context.Background(), session)
	l.deleteSession(agentID)

	assignment, err := l.loadAssignmentByAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("load agent %s for continuation: %w", agentID, err)
	}
	if assignment.agent == nil || assignment.ticket == nil || assignment.run == nil {
		return nil
	}

	tx, err := l.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start continuation tx: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.Ticket.Update().
		Where(
			entticket.IDEQ(assignment.ticket.ID),
			entticket.HasCurrentRunWith(entagentrun.AgentIDEQ(agentID)),
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
		tx.AgentRun.UpdateOneID(assignment.run.ID).
			SetStatus(entagentrun.StatusTerminated),
	).Save(ctx); err != nil {
		return fmt.Errorf("reset run %s after continuation limit: %w", assignment.run.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit continuation tx: %w", err)
	}

	reloaded, err := loadAgentLifecycleState(ctx, l.client, agentID)
	if err != nil {
		return err
	}
	return publishAgentLifecycleEvent(
		ctx,
		l.client,
		l.events,
		agentTerminatedType,
		reloaded,
		lifecycleMessage(agentTerminatedType, reloaded.agent.Name),
		runtimeEventMetadataForState(reloaded),
		l.now().UTC(),
	)
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
