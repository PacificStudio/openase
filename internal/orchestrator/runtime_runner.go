package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

const (
	defaultRuntimeMaxTurns = 10
	continuationRetryDelay = time.Second
)

type tokenUsageHighWater struct {
	inputTokens              int64
	outputTokens             int64
	cachedInputTokens        int64
	cacheCreationInputTokens int64
	reasoningTokens          int64
	promptTokens             int64
	candidateTokens          int64
	toolTokens               int64
}

type turnSessionClosedError struct {
	turnID string
	cause  error
}

func (e *turnSessionClosedError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return fmt.Sprintf("agent session closed before turn %s completed: %v", e.turnID, e.cause)
	}
	return fmt.Sprintf("agent session closed before turn %s completed", e.turnID)
}

func (e *turnSessionClosedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func isCleanTurnSessionClose(err error) bool {
	var closedErr *turnSessionClosedError
	return errors.As(err, &closedErr) && closedErr != nil && closedErr.cause == nil
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

		executingAt := l.now().UTC()
		executingAgent, err := loadAgentLifecycleState(ctx, l.client, assignment.agent.ID, &assignment.run.ID)
		if err != nil {
			l.finishExecution(assignment.run.ID)
			return err
		}
		l.publishLifecycleEvent(
			ctx,
			agentExecutingType,
			executingAgent,
			lifecycleMessage(agentExecutingType, executingAgent.agent.Name),
			runtimeEventMetadataForState(executingAgent),
			executingAt,
		)

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
		if l.loadSession(runID) != session {
			return
		}
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

		turn, err := session.SendPrompt(ctx, prompt)
		if err != nil {
			reloadedTicket, reloadErr := l.reloadExecutionTicket(ctx, state.ticket.ID)
			if reloadErr == nil && classifyRuntimeTicket(reloadedTicket, state.run.ID, state.run.WorkflowID) != runtimeTicketActive {
				return
			}
			l.handleExecutionFailure(ctx, state.run.ID, state.agent.ID, state.ticket.ID, err)
			return
		}
		l.runtime.recordTurnStart(state.run.ID, turnNumber, l.now().UTC())
		if err := l.recordAgentStep(ctx, state.agent.ProjectID, state.agent.ID, state.ticket.ID, state.run.ID, "planning", fmt.Sprintf("Started turn %d.", turnNumber), nil); err != nil {
			l.logger.Warn("record agent planning step", "run_id", state.run.ID, "error", err)
		}

		if err := l.consumeTurn(
			ctx,
			state.agent.ProjectID,
			state.agent.ID,
			state.run.ID,
			state.ticket.ID,
			state.launchContext.agent.Edges.Provider.AdapterType,
			state.launchContext.agent.Edges.Provider.ID,
			session,
			turn.TurnID,
			&highWater,
		); err != nil {
			if isCleanTurnSessionClose(err) {
				message := ""
				var closedErr *turnSessionClosedError
				if errors.As(err, &closedErr) && closedErr != nil && closedErr.cause != nil {
					message = closedErr.Error()
				}
				l.runtime.recordRuntimeFact(state.run.ID, runtimeFactSessionExited, l.now().UTC(), message)
				return
			}
			l.handleExecutionFailure(ctx, state.run.ID, state.agent.ID, state.ticket.ID, err)
			return
		}
		if l.loadSession(runID) != session {
			return
		}
		if err := l.tickets.RunLifecycleHook(ctx, ticketservice.RunLifecycleHookInput{
			TicketID: state.ticket.ID,
			RunID:    state.run.ID,
			HookName: infrahook.TicketHookOnComplete,
			Blocking: true,
		}); err != nil {
			l.handleExecutionFailure(ctx, state.run.ID, state.agent.ID, state.ticket.ID, fmt.Errorf("run ticket on_complete hooks: %w", err))
			return
		}

		reloaded, err := l.reloadExecutionTicket(ctx, state.ticket.ID)
		if err != nil {
			l.logger.Error("reload execution ticket", "agent_id", state.agent.ID, "ticket_id", state.ticket.ID, "error", err)
			stopSession(context.Background(), session)
			l.deleteSession(runID)
			return
		}

		if classifyRuntimeTicket(reloaded, state.run.ID, state.run.WorkflowID) != runtimeTicketActive {
			if err := l.releaseExecutionOwnership(ctx, state.run.ID, state.agent.ID, reloaded); err != nil {
				l.handleExecutionFailure(ctx, state.run.ID, state.agent.ID, reloaded.ID, err)
			}
			return
		}
	}

	reloaded, err := l.reloadExecutionTicket(ctx, lastState.ticket.ID)
	if err != nil {
		l.handleExecutionFailure(ctx, lastState.run.ID, lastState.agent.ID, lastState.ticket.ID, err)
		return
	}
	if classifyRuntimeTicket(reloaded, lastState.run.ID, lastState.run.WorkflowID) != runtimeTicketActive {
		if err := l.releaseExecutionOwnership(ctx, lastState.run.ID, lastState.agent.ID, reloaded); err != nil {
			l.handleExecutionFailure(ctx, lastState.run.ID, lastState.agent.ID, reloaded.ID, err)
		}
		return
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

	runtimeSnapshot, snapshotErr := l.loadRecordedRuntimeSnapshot(ctx, assignment.run.ID)
	if snapshotErr != nil {
		return runtimeExecutionState{}, "", snapshotErr
	}
	developerInstructions, err := l.buildDeveloperInstructions(
		ctx,
		launchContext,
		machine,
		workspace,
		runtimeSnapshot,
		l.ticketRuntimePlatformContract(launchContext, agentplatform.DefaultScopesForPrincipalKind(agentplatform.PrincipalKindTicketAgent)),
	)
	if err != nil {
		return runtimeExecutionState{}, "", err
	}

	prompt := strings.TrimSpace(developerInstructions)
	if turnNumber > 1 {
		prompt = composeRuntimeTurnPrompt(
			prompt,
			buildContinuationPrompt(launchContext.ticket, turnNumber, defaultRuntimeMaxTurns, lastError),
		)
	}

	return runtimeExecutionState{
		agent:         launchContext.agent,
		run:           assignment.run,
		ticket:        launchContext.ticket,
		launchContext: launchContext,
	}, prompt, nil
}

func composeRuntimeTurnPrompt(basePrompt string, continuationPrompt string) string {
	basePrompt = strings.TrimSpace(basePrompt)
	continuationPrompt = strings.TrimSpace(continuationPrompt)

	switch {
	case basePrompt == "":
		return continuationPrompt
	case continuationPrompt == "":
		return basePrompt
	default:
		return basePrompt + "\n\n" + continuationPrompt
	}
}

func buildContinuationPrompt(ticket *ent.Ticket, turnNumber int, maxTurns int, lastError string) string {
	var builder strings.Builder
	builder.WriteString("Continuation guidance:\n\n")
	builder.WriteString("- The previous orchestrated turn completed, but the ticket is still active.\n")
	_, _ = fmt.Fprintf(&builder, "- This is continuation turn #%d of %d.\n", turnNumber, maxTurns)
	builder.WriteString("- Resume from the current workspace and thread context.\n")
	builder.WriteString("- Do not restate the original task before acting.\n")
	if ticket != nil {
		_, _ = fmt.Fprintf(&builder, "- Continue working on ticket %s: %s.\n", ticket.Identifier, ticket.Title)
	}
	if trimmed := strings.TrimSpace(lastError); trimmed != "" {
		_, _ = fmt.Fprintf(&builder, "- Address the latest blocker or failure if it is still relevant: %s\n", trimmed)
	}
	return strings.TrimSpace(builder.String())
}

func (l *RuntimeLauncher) consumeTurn(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	runID uuid.UUID,
	ticketID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	providerID uuid.UUID,
	session agentSession,
	turnID string,
	highWater *tokenUsageHighWater,
) error {
	outputAccumulator := agentOutputAccumulator{}
	recordedOutputFingerprints := map[string]string{}

	recordBufferedOutput := func(output *agentOutputEvent) error {
		if output == nil {
			return nil
		}
		persisted := outputForPersistence(output)
		if persisted == nil {
			return nil
		}
		fingerprintKey, fingerprintValue := agentOutputPersistenceFingerprint(persisted)
		if fingerprintKey != "" {
			if previous, ok := recordedOutputFingerprints[fingerprintKey]; ok && previous == fingerprintValue {
				return nil
			}
		}
		if err := l.recordAgentOutput(ctx, projectID, agentID, ticketID, runID, adapterType, persisted); err != nil {
			return err
		}
		if fingerprintKey != "" {
			recordedOutputFingerprints[fingerprintKey] = fingerprintValue
		}
		return nil
	}

	flushBufferedOutputs := func() error {
		for _, output := range outputAccumulator.flush() {
			if err := recordBufferedOutput(output); err != nil {
				return err
			}
		}
		return nil
	}

	for {
		event, ok := <-session.Events()
		if !ok {
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			logAgentSessionClosed(l.logger, runID, turnID, adapterType, session)
			if sessionErr := session.Err(); sessionErr != nil {
				return &turnSessionClosedError{turnID: turnID, cause: sessionErr}
			}
			return &turnSessionClosedError{turnID: turnID}
		}
		observedAt := l.now().UTC()
		if event.ObservedAt != nil {
			observedAt = event.ObservedAt.UTC()
		}
		l.runtime.recordCodexEvent(runID, string(event.Type), observedAt)

		if err := l.persistRuntimeSessionID(ctx, runID, session); err != nil {
			l.logger.Warn("persist runtime session id", "run_id", runID, "error", err)
		}

		if err := l.touchHeartbeat(ctx, runID); err != nil {
			l.logger.Warn("update agent heartbeat", "run_id", runID, "error", err)
		}

		switch event.Type {
		case agentEventTypeToolCallRequested:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.ToolCall == nil {
				continue
			}
			if err := l.recordAgentToolCall(ctx, projectID, agentID, ticketID, runID, adapterType, event.ToolCall); err != nil {
				return err
			}
		case agentEventTypeApprovalRequested:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.Approval == nil {
				continue
			}
			if !turnMatches(turnID, event.Approval.TurnID) {
				continue
			}
			if err := l.recordAgentApprovalRequest(ctx, projectID, agentID, ticketID, runID, adapterType, event.Approval); err != nil {
				return err
			}
		case agentEventTypeUserInputRequested:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.UserInput == nil {
				continue
			}
			if !turnMatches(turnID, event.UserInput.TurnID) {
				continue
			}
			if err := l.recordAgentUserInputRequest(ctx, projectID, agentID, ticketID, runID, adapterType, event.UserInput); err != nil {
				return err
			}
		case agentEventTypeTokenUsageUpdated:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.TokenUsage == nil {
				continue
			}
			if !turnMatches(turnID, event.TokenUsage.TurnID) {
				continue
			}
			if err := l.recordTokenUsage(ctx, agentID, runID, ticketID, event.TokenUsage, highWater); err != nil {
				return err
			}
		case agentEventTypeRateLimitUpdated:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.RateLimit == nil {
				continue
			}
			if err := l.recordProviderRateLimit(ctx, providerID, event.RateLimit, observedAt); err != nil {
				return err
			}
		case agentEventTypeThreadStatus:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.Thread == nil {
				continue
			}
			if err := l.recordAgentThreadStatus(ctx, projectID, agentID, ticketID, runID, adapterType, event.Thread); err != nil {
				return err
			}
		case agentEventTypeTurnDiffUpdated:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.Diff == nil {
				continue
			}
			if !turnMatches(turnID, event.Diff.TurnID) {
				continue
			}
			if err := l.recordAgentTurnDiff(ctx, projectID, agentID, ticketID, runID, adapterType, event.Diff); err != nil {
				return err
			}
		case agentEventTypeReasoningUpdated:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.Reasoning == nil {
				continue
			}
			if !turnMatches(turnID, event.Reasoning.TurnID) {
				continue
			}
			if err := l.recordAgentReasoning(ctx, projectID, agentID, ticketID, runID, adapterType, event.Reasoning); err != nil {
				return err
			}
		case agentEventTypeOutputProduced:
			if event.Output == nil {
				continue
			}
			if !turnMatches(turnID, event.Output.TurnID) {
				continue
			}
			for _, output := range outputAccumulator.push(event.Output) {
				if err := recordBufferedOutput(output); err != nil {
					return err
				}
			}
		case agentEventTypeTaskStatus:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.TaskStatus == nil {
				continue
			}
			if !turnMatches(turnID, event.TaskStatus.TurnID) {
				continue
			}
			if err := l.recordAgentTaskStatus(ctx, projectID, agentID, ticketID, runID, adapterType, event.TaskStatus); err != nil {
				return err
			}
		case agentEventTypeTurnFailed:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.Turn == nil || !turnMatches(turnID, event.Turn.TurnID) {
				continue
			}
			if event.Turn.Error == nil {
				return fmt.Errorf("%s turn %s failed", runtimeProviderName(adapterType), turnID)
			}
			return fmt.Errorf("%s turn %s failed: %s", runtimeProviderName(adapterType), turnID, strings.TrimSpace(event.Turn.Error.Message))
		case agentEventTypeTurnCompleted:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.Turn == nil || !turnMatches(turnID, event.Turn.TurnID) {
				continue
			}
			return nil
		}
	}
}

func logAgentSessionClosed(logger *slog.Logger, runID uuid.UUID, turnID string, adapterType entagentprovider.AdapterType, session agentSession) {
	if logger == nil {
		return
	}

	diagnostic := session.Diagnostic()
	attrs := []any{
		"run_id", runID,
		"turn_id", strings.TrimSpace(turnID),
	}
	if diagnostic.PID != 0 {
		attrs = append(attrs, "provider_pid", diagnostic.PID)
	}
	if trimmed := strings.TrimSpace(diagnostic.SessionID); trimmed != "" {
		attrs = append(attrs, "provider_session_id", trimmed)
	}
	if trimmed := strings.TrimSpace(diagnostic.Error); trimmed != "" {
		attrs = append(attrs, "provider_session_error", trimmed)
	}
	if trimmed := strings.TrimSpace(diagnostic.Stderr); trimmed != "" {
		attrs = append(attrs, "provider_stderr", trimmed)
	}
	attrs = append(attrs, "adapter_type", string(adapterType))

	logger.Error("agent session closed before turn completed", attrs...)
}

func (l *RuntimeLauncher) recordAgentOutput(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	output *agentOutputEvent,
) error {
	if l == nil || output == nil {
		return nil
	}

	text := strings.TrimSpace(output.Text)
	if text == "" {
		return nil
	}

	metadata := map[string]any{
		"provider": runtimeProviderName(adapterType),
		"run_id":   runID.String(),
	}
	if itemID := strings.TrimSpace(output.ItemID); itemID != "" {
		metadata["item_id"] = itemID
	}
	if turnID := strings.TrimSpace(output.TurnID); turnID != "" {
		metadata["turn_id"] = turnID
	}
	if command := strings.TrimSpace(output.Command); command != "" {
		metadata["command"] = command
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
		Provider:    runtimeProviderName(adapterType),
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

func (l *RuntimeLauncher) recordAgentTaskStatus(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	status *agentTaskStatusEvent,
) error {
	if l == nil || status == nil {
		return nil
	}

	traceKind := strings.TrimSpace(status.StatusType)
	if traceKind == "" {
		return nil
	}

	payload := cloneAgentTracePayload(status.Payload)
	if payload == nil {
		payload = map[string]any{}
	}
	payload["provider"] = runtimeProviderName(adapterType)
	payload["run_id"] = runID.String()
	if threadID := strings.TrimSpace(status.ThreadID); threadID != "" {
		payload["thread_id"] = threadID
	}
	if turnID := strings.TrimSpace(status.TurnID); turnID != "" {
		payload["turn_id"] = turnID
	}
	if itemID := strings.TrimSpace(status.ItemID); itemID != "" {
		payload["item_id"] = itemID
	}

	if _, err := publishAgentTraceEvent(ctx, l.client, l.events, agentTraceEventInput{
		ProjectID:   projectID,
		AgentID:     agentID,
		TicketID:    ticketID,
		AgentRunID:  runID,
		Provider:    runtimeProviderName(adapterType),
		Kind:        traceKind,
		Stream:      "task",
		Text:        strings.TrimSpace(status.Text),
		Payload:     payload,
		PublishedAt: l.now().UTC(),
	}); err != nil {
		return fmt.Errorf("record task status trace for run %s: %w", runID, err)
	}

	return nil
}

func (l *RuntimeLauncher) recordAgentToolCall(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	request *agentToolCallRequest,
) error {
	if l == nil || request == nil {
		return nil
	}

	toolName := strings.TrimSpace(request.Tool)
	metadata := map[string]any{
		"provider": runtimeProviderName(adapterType),
		"run_id":   runID.String(),
		"tool":     toolName,
	}
	if decodedArguments := decodeRawJSON(request.Arguments); decodedArguments != nil {
		metadata["arguments"] = decodedArguments
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
		Provider:    runtimeProviderName(adapterType),
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

func (l *RuntimeLauncher) recordAgentThreadStatus(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	status *agentThreadStatusEvent,
) error {
	if l == nil || status == nil {
		return nil
	}

	metadata := map[string]any{
		"provider": runtimeProviderName(adapterType),
		"run_id":   runID.String(),
		"status":   strings.TrimSpace(status.Status),
	}
	if threadID := strings.TrimSpace(status.ThreadID); threadID != "" {
		metadata["thread_id"] = threadID
	}
	if len(status.ActiveFlags) > 0 {
		metadata["active_flags"] = append([]string(nil), status.ActiveFlags...)
	}

	_, err := publishAgentTraceEvent(ctx, l.client, l.events, agentTraceEventInput{
		ProjectID:   projectID,
		AgentID:     agentID,
		TicketID:    ticketID,
		AgentRunID:  runID,
		Provider:    runtimeProviderName(adapterType),
		Kind:        catalogdomain.AgentTraceKindThreadStatus,
		Stream:      "system",
		Text:        threadStatusTraceSummary(status),
		Payload:     metadata,
		PublishedAt: l.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("record thread status trace for run %s: %w", runID, err)
	}

	return nil
}

func (l *RuntimeLauncher) recordAgentTurnDiff(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	diff *agentTurnDiffEvent,
) error {
	if l == nil || diff == nil {
		return nil
	}

	diffText := strings.TrimSpace(diff.Diff)
	if diffText == "" {
		return nil
	}

	metadata := map[string]any{
		"provider": runtimeProviderName(adapterType),
		"run_id":   runID.String(),
	}
	if threadID := strings.TrimSpace(diff.ThreadID); threadID != "" {
		metadata["thread_id"] = threadID
	}
	if turnID := strings.TrimSpace(diff.TurnID); turnID != "" {
		metadata["turn_id"] = turnID
	}

	_, err := publishAgentTraceEvent(ctx, l.client, l.events, agentTraceEventInput{
		ProjectID:   projectID,
		AgentID:     agentID,
		TicketID:    ticketID,
		AgentRunID:  runID,
		Provider:    runtimeProviderName(adapterType),
		Kind:        catalogdomain.AgentTraceKindTurnDiffUpdated,
		Stream:      "diff",
		Text:        diffText,
		Payload:     metadata,
		PublishedAt: l.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("record diff trace for run %s: %w", runID, err)
	}

	return nil
}

func (l *RuntimeLauncher) recordAgentReasoning(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	reasoning *agentReasoningEvent,
) error {
	if l == nil || reasoning == nil {
		return nil
	}

	metadata := map[string]any{
		"provider": runtimeProviderName(adapterType),
		"run_id":   runID.String(),
		"kind":     strings.TrimSpace(reasoning.Kind),
	}
	if threadID := strings.TrimSpace(reasoning.ThreadID); threadID != "" {
		metadata["thread_id"] = threadID
	}
	if turnID := strings.TrimSpace(reasoning.TurnID); turnID != "" {
		metadata["turn_id"] = turnID
	}
	if itemID := strings.TrimSpace(reasoning.ItemID); itemID != "" {
		metadata["item_id"] = itemID
	}
	if reasoning.SummaryIndex != nil {
		metadata["summary_index"] = *reasoning.SummaryIndex
	}
	if reasoning.ContentIndex != nil {
		metadata["content_index"] = *reasoning.ContentIndex
	}

	_, err := publishAgentTraceEvent(ctx, l.client, l.events, agentTraceEventInput{
		ProjectID:   projectID,
		AgentID:     agentID,
		TicketID:    ticketID,
		AgentRunID:  runID,
		Provider:    runtimeProviderName(adapterType),
		Kind:        catalogdomain.AgentTraceKindReasoningUpdated,
		Stream:      "reasoning",
		Text:        reasoningTraceSummary(reasoning),
		Payload:     metadata,
		PublishedAt: l.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("record reasoning trace for run %s: %w", runID, err)
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

func (l *RuntimeLauncher) recordAgentApprovalRequest(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	request *agentApprovalRequest,
) error {
	if l == nil || request == nil {
		return nil
	}

	metadata := map[string]any{
		"provider": runtimeProviderName(adapterType),
		"run_id":   runID.String(),
		"kind":     strings.TrimSpace(request.Kind),
		"payload":  cloneAgentTracePayload(request.Payload),
	}
	if requestID := strings.TrimSpace(request.RequestID); requestID != "" {
		metadata["request_id"] = requestID
	}
	if threadID := strings.TrimSpace(request.ThreadID); threadID != "" {
		metadata["thread_id"] = threadID
	}
	if turnID := strings.TrimSpace(request.TurnID); turnID != "" {
		metadata["turn_id"] = turnID
	}
	if options := mapAgentApprovalOptions(request.Options); len(options) > 0 {
		metadata["options"] = options
	}

	traceItem, err := publishAgentTraceEvent(ctx, l.client, l.events, agentTraceEventInput{
		ProjectID:   projectID,
		AgentID:     agentID,
		TicketID:    ticketID,
		AgentRunID:  runID,
		Provider:    runtimeProviderName(adapterType),
		Kind:        catalogdomain.AgentTraceKindApprovalRequested,
		Stream:      "interrupt",
		Text:        approvalRequestTraceSummary(request),
		Payload:     metadata,
		PublishedAt: l.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("record approval request trace for run %s: %w", runID, err)
	}
	traceID := traceItem.ID
	if err := l.recordAgentStep(
		ctx,
		projectID,
		agentID,
		ticketID,
		runID,
		approvalRequestStepStatus(request),
		approvalRequestStepSummary(request),
		&traceID,
	); err != nil {
		return fmt.Errorf("record approval request step for run %s: %w", runID, err)
	}

	return nil
}

func (l *RuntimeLauncher) recordAgentUserInputRequest(
	ctx context.Context,
	projectID uuid.UUID,
	agentID uuid.UUID,
	ticketID uuid.UUID,
	runID uuid.UUID,
	adapterType entagentprovider.AdapterType,
	request *agentUserInputRequest,
) error {
	if l == nil || request == nil {
		return nil
	}

	metadata := map[string]any{
		"provider": runtimeProviderName(adapterType),
		"run_id":   runID.String(),
		"payload":  cloneAgentTracePayload(request.Payload),
	}
	if requestID := strings.TrimSpace(request.RequestID); requestID != "" {
		metadata["request_id"] = requestID
	}
	if threadID := strings.TrimSpace(request.ThreadID); threadID != "" {
		metadata["thread_id"] = threadID
	}
	if turnID := strings.TrimSpace(request.TurnID); turnID != "" {
		metadata["turn_id"] = turnID
	}

	traceItem, err := publishAgentTraceEvent(ctx, l.client, l.events, agentTraceEventInput{
		ProjectID:   projectID,
		AgentID:     agentID,
		TicketID:    ticketID,
		AgentRunID:  runID,
		Provider:    runtimeProviderName(adapterType),
		Kind:        catalogdomain.AgentTraceKindUserInputRequested,
		Stream:      "interrupt",
		Text:        userInputRequestTraceSummary(request),
		Payload:     metadata,
		PublishedAt: l.now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("record user input trace for run %s: %w", runID, err)
	}
	traceID := traceItem.ID
	if err := l.recordAgentStep(
		ctx,
		projectID,
		agentID,
		ticketID,
		runID,
		"awaiting_input",
		userInputRequestStepSummary(request),
		&traceID,
	); err != nil {
		return fmt.Errorf("record user input step for run %s: %w", runID, err)
	}

	return nil
}

func agentTraceKindForOutput(output *agentOutputEvent) string {
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

func approvalRequestStepStatus(request *agentApprovalRequest) string {
	switch strings.TrimSpace(request.Kind) {
	case "file_change":
		return "awaiting_file_approval"
	default:
		return "awaiting_command_approval"
	}
}

func approvalRequestStepSummary(request *agentApprovalRequest) string {
	switch strings.TrimSpace(request.Kind) {
	case "file_change":
		if target := trimmedInterruptString(request.Payload, "file", "path", "target"); target != "" {
			return fmt.Sprintf("Waiting for file change approval on %s.", target)
		}
		return "Waiting for file change approval."
	default:
		if command := trimmedInterruptString(request.Payload, "command"); command != "" {
			return fmt.Sprintf("Waiting for command approval to run %q.", command)
		}
		return "Waiting for command approval."
	}
}

func approvalRequestTraceSummary(request *agentApprovalRequest) string {
	summary := approvalRequestStepSummary(request)
	return strings.TrimSuffix(summary, ".")
}

func userInputRequestStepSummary(request *agentUserInputRequest) string {
	if prompt := firstInterruptQuestion(request.Payload); prompt != "" {
		return fmt.Sprintf("Waiting for user input: %s", prompt)
	}
	return "Waiting for user input."
}

func userInputRequestTraceSummary(request *agentUserInputRequest) string {
	summary := userInputRequestStepSummary(request)
	return strings.TrimSuffix(summary, ".")
}

func trimmedInterruptString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		text, ok := value.(string)
		if !ok {
			continue
		}
		if trimmed := strings.TrimSpace(text); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstInterruptQuestion(payload map[string]any) string {
	rawQuestions, ok := payload["questions"]
	if !ok {
		return ""
	}
	questions, ok := rawQuestions.([]any)
	if !ok || len(questions) == 0 {
		return ""
	}
	first, ok := questions[0].(map[string]any)
	if !ok {
		return ""
	}
	question, ok := first["question"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(question)
}

func mapAgentApprovalOptions(options []agentApprovalOption) []map[string]any {
	if len(options) == 0 {
		return nil
	}

	mapped := make([]map[string]any, 0, len(options))
	for _, option := range options {
		item := map[string]any{}
		if id := strings.TrimSpace(option.ID); id != "" {
			item["id"] = id
		}
		if label := strings.TrimSpace(option.Label); label != "" {
			item["label"] = label
		}
		if rawDecision := strings.TrimSpace(option.RawDecision); rawDecision != "" {
			item["raw_decision"] = rawDecision
		}
		if len(item) > 0 {
			mapped = append(mapped, item)
		}
	}
	return mapped
}

func cloneAgentTracePayload(payload map[string]any) map[string]any {
	if len(payload) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}

type agentOutputAccumulator struct {
	pending *agentOutputEvent
}

func (a *agentOutputAccumulator) push(output *agentOutputEvent) []*agentOutputEvent {
	if output == nil {
		return nil
	}

	current := cloneAgentOutputEvent(output)
	if current == nil {
		return nil
	}
	if !shouldAggregateAgentOutput(current) {
		return a.replacePending(current)
	}
	if a.pending == nil {
		a.pending = current
		if shouldFlushAggregatedOutput(a.pending) {
			return a.flush()
		}
		return nil
	}
	if !sameAggregatedOutputKey(a.pending, current) {
		return a.replacePending(current)
	}

	a.pending.Text = mergeAggregatedOutputText(a.pending.Text, current.Text, current.Snapshot)
	a.pending.Snapshot = a.pending.Snapshot || current.Snapshot
	if strings.TrimSpace(current.Command) != "" {
		a.pending.Command = current.Command
	}
	if strings.TrimSpace(current.Phase) != "" {
		a.pending.Phase = current.Phase
	}

	if shouldFlushAggregatedOutput(a.pending) {
		return a.flush()
	}
	return nil
}

func (a *agentOutputAccumulator) replacePending(current *agentOutputEvent) []*agentOutputEvent {
	flushed := a.flush()
	if shouldAggregateAgentOutput(current) {
		a.pending = current
		if shouldFlushAggregatedOutput(a.pending) {
			return append(flushed, a.flush()...)
		}
		return flushed
	}
	return append(flushed, current)
}

func (a *agentOutputAccumulator) flush() []*agentOutputEvent {
	if a == nil || a.pending == nil {
		return nil
	}
	flushed := []*agentOutputEvent{a.pending}
	a.pending = nil
	return flushed
}

func cloneAgentOutputEvent(output *agentOutputEvent) *agentOutputEvent {
	if output == nil {
		return nil
	}
	cloned := *output
	return &cloned
}

func outputForPersistence(output *agentOutputEvent) *agentOutputEvent {
	if !shouldPersistAgentOutput(output) {
		return nil
	}
	cloned := cloneAgentOutputEvent(output)
	if cloned == nil {
		return nil
	}
	if shouldAggregateAgentOutput(cloned) {
		cloned.Snapshot = true
	}
	return cloned
}

func shouldPersistAgentOutput(output *agentOutputEvent) bool {
	if output == nil {
		return false
	}
	switch strings.TrimSpace(output.Stream) {
	case "assistant", "command":
		return strings.TrimSpace(output.Text) != ""
	default:
		return false
	}
}

func shouldAggregateAgentOutput(output *agentOutputEvent) bool {
	if output == nil {
		return false
	}
	if strings.TrimSpace(output.ItemID) == "" {
		return false
	}
	switch strings.TrimSpace(output.Stream) {
	case "assistant", "command":
		return true
	default:
		return false
	}
}

func agentOutputPersistenceFingerprint(output *agentOutputEvent) (string, string) {
	if output == nil {
		return "", ""
	}
	itemID := strings.TrimSpace(output.ItemID)
	if itemID == "" {
		return "", ""
	}
	key := strings.Join([]string{
		strings.TrimSpace(output.Stream),
		itemID,
		strings.TrimSpace(output.TurnID),
		strings.TrimSpace(output.Command),
	}, "\x00")
	value := strings.Join([]string{
		strconv.FormatBool(output.Snapshot),
		strings.TrimSpace(output.Phase),
		output.Text,
	}, "\x00")
	return key, value
}

func sameAggregatedOutputKey(left *agentOutputEvent, right *agentOutputEvent) bool {
	if left == nil || right == nil {
		return false
	}
	return strings.TrimSpace(left.Stream) == strings.TrimSpace(right.Stream) &&
		strings.TrimSpace(left.ItemID) == strings.TrimSpace(right.ItemID) &&
		strings.TrimSpace(left.TurnID) == strings.TrimSpace(right.TurnID) &&
		strings.TrimSpace(left.Command) == strings.TrimSpace(right.Command) &&
		strings.TrimSpace(left.Phase) == strings.TrimSpace(right.Phase)
}

func mergeAggregatedOutputText(existing string, next string, snapshot bool) string {
	if !snapshot {
		return existing + next
	}
	switch {
	case existing == "":
		return next
	case strings.HasPrefix(next, existing):
		return next
	default:
		return existing + next
	}
}

func shouldFlushAggregatedOutput(output *agentOutputEvent) bool {
	if output == nil {
		return false
	}
	if output.Snapshot {
		return true
	}
	text := output.Text
	if strings.TrimSpace(text) == "" {
		return false
	}
	if strings.Contains(text, "\n") {
		return true
	}
	switch strings.TrimSpace(output.Stream) {
	case "assistant":
		return len(text) >= 192 || strings.ContainsAny(text, ".。！？!?")
	case "command":
		return len(text) >= 256
	default:
		return false
	}
}

func agentStepFromOutput(output *agentOutputEvent, text string) (string, string, bool) {
	if output == nil {
		return "", "", false
	}
	if phase := strings.TrimSpace(output.Phase); phase != "" {
		return phase, summarizeAgentStepText(text), true
	}
	switch strings.TrimSpace(output.Stream) {
	case "command":
		if command := strings.TrimSpace(output.Command); command != "" {
			return "running_command", command, true
		}
		return "running_command", summarizeAgentStepText(text), true
	default:
		return "", "", false
	}
}

func summarizeAgentStepText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	line := strings.ToValidUTF8(strings.Split(trimmed, "\n")[0], "")
	if len(line) <= 140 {
		return line
	}
	truncated := truncateUTF8Bytes(line, 140)
	if truncated == "" {
		return "..."
	}
	return strings.TrimSpace(truncated) + "..."
}

func decodeRawJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err == nil {
		return decoded
	}

	text := strings.TrimSpace(string(raw))
	if text == "" {
		return nil
	}
	return text
}

func threadStatusTraceSummary(status *agentThreadStatusEvent) string {
	if status == nil {
		return ""
	}

	parts := []string{}
	if trimmed := strings.TrimSpace(status.Status); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if len(status.ActiveFlags) > 0 {
		parts = append(parts, strings.Join(status.ActiveFlags, ", "))
	}
	return strings.Join(parts, " · ")
}

func reasoningTraceSummary(reasoning *agentReasoningEvent) string {
	if reasoning == nil {
		return ""
	}
	if delta := strings.TrimSpace(reasoning.Delta); delta != "" {
		return delta
	}
	return strings.TrimSpace(reasoning.Kind)
}

func toolCallStepSummary(toolName string) string {
	if strings.TrimSpace(toolName) == "" {
		return "Running provider tool call."
	}
	return "Running provider tool " + strconv.Quote(toolName) + "."
}

func turnMatches(expected string, actual string) bool {
	trimmedExpected := strings.TrimSpace(expected)
	trimmedActual := strings.TrimSpace(actual)
	if trimmedExpected == "" || trimmedActual == "" {
		return true
	}
	return trimmedExpected == trimmedActual
}

func (l *RuntimeLauncher) persistRuntimeSessionID(ctx context.Context, runID uuid.UUID, session agentSession) error {
	if l == nil || l.client == nil || session == nil {
		return nil
	}
	sessionID, ok := session.SessionID()
	if !ok || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	_, err := l.client.AgentRun.Update().
		Where(
			entagentrun.IDEQ(runID),
			entagentrun.SessionIDIsNil(),
		).
		SetSessionID(sessionID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("persist session id for run %s: %w", runID, err)
	}
	return nil
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
	runID uuid.UUID,
	ticketID uuid.UUID,
	usage *agentTokenUsageEvent,
	highWater *tokenUsageHighWater,
) error {
	if l == nil || l.tickets == nil || usage == nil || highWater == nil {
		return nil
	}

	inputDelta := usage.TotalInputTokens - highWater.inputTokens
	outputDelta := usage.TotalOutputTokens - highWater.outputTokens
	cachedInputDelta := usage.TotalCachedInputTokens - highWater.cachedInputTokens
	cacheCreationDelta := usage.TotalCacheCreationInputTokens - highWater.cacheCreationInputTokens
	reasoningDelta := usage.TotalReasoningTokens - highWater.reasoningTokens
	promptDelta := usage.TotalPromptTokens - highWater.promptTokens
	candidateDelta := usage.TotalCandidateTokens - highWater.candidateTokens
	toolDelta := usage.TotalToolTokens - highWater.toolTokens
	if inputDelta < 0 {
		inputDelta = 0
	}
	if outputDelta < 0 {
		outputDelta = 0
	}
	if cachedInputDelta < 0 {
		cachedInputDelta = 0
	}
	if cacheCreationDelta < 0 {
		cacheCreationDelta = 0
	}
	if reasoningDelta < 0 {
		reasoningDelta = 0
	}
	if promptDelta < 0 {
		promptDelta = 0
	}
	if candidateDelta < 0 {
		candidateDelta = 0
	}
	if toolDelta < 0 {
		toolDelta = 0
	}

	if inputDelta == 0 &&
		outputDelta == 0 &&
		cachedInputDelta == 0 &&
		cacheCreationDelta == 0 &&
		reasoningDelta == 0 &&
		promptDelta == 0 &&
		candidateDelta == 0 &&
		toolDelta == 0 {
		return nil
	}

	highWater.inputTokens = maxInt64(highWater.inputTokens, usage.TotalInputTokens)
	highWater.outputTokens = maxInt64(highWater.outputTokens, usage.TotalOutputTokens)
	highWater.cachedInputTokens = maxInt64(highWater.cachedInputTokens, usage.TotalCachedInputTokens)
	highWater.cacheCreationInputTokens = maxInt64(highWater.cacheCreationInputTokens, usage.TotalCacheCreationInputTokens)
	highWater.reasoningTokens = maxInt64(highWater.reasoningTokens, usage.TotalReasoningTokens)
	highWater.promptTokens = maxInt64(highWater.promptTokens, usage.TotalPromptTokens)
	highWater.candidateTokens = maxInt64(highWater.candidateTokens, usage.TotalCandidateTokens)
	highWater.toolTokens = maxInt64(highWater.toolTokens, usage.TotalToolTokens)

	result, err := l.tickets.RecordUsage(ctx, ticketservice.RecordUsageInput{
		AgentID:  agentID,
		TicketID: ticketID,
		RunID:    &runID,
		Usage: ticketing.RawUsageDelta{
			InputTokens:              int64Pointer(inputDelta),
			OutputTokens:             int64Pointer(outputDelta),
			CachedInputTokens:        int64Pointer(cachedInputDelta),
			CacheCreationInputTokens: int64Pointer(cacheCreationDelta),
			ReasoningTokens:          int64Pointer(reasoningDelta),
			PromptTokens:             int64Pointer(promptDelta),
			CandidateTokens:          int64Pointer(candidateDelta),
			ToolTokens:               int64Pointer(toolDelta),
			ModelContextWindow:       cloneInt64Pointer(usage.ModelContextWindow),
			CostUSD:                  cloneCostUSD(usage.CostUSD),
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

func (l *RuntimeLauncher) recordProviderRateLimit(
	ctx context.Context,
	providerID uuid.UUID,
	rateLimit *provider.CLIRateLimit,
	observedAt time.Time,
) error {
	if l == nil || l.client == nil || rateLimit == nil || providerID == uuid.Nil {
		return nil
	}

	currentProvider, err := l.client.AgentProvider.Get(ctx, providerID)
	if err != nil {
		return fmt.Errorf("load provider %s before rate limit update: %w", providerID, err)
	}

	payload, err := marshalProviderRateLimit(rateLimit)
	if err != nil {
		return fmt.Errorf("marshal provider rate limit for provider %s: %w", providerID, err)
	}
	if len(payload) == 0 {
		return nil
	}
	snapshotChanged := !reflect.DeepEqual(currentProvider.CliRateLimit, payload)

	updatedProvider, err := l.client.AgentProvider.UpdateOneID(providerID).
		SetCliRateLimit(payload).
		SetCliRateLimitUpdatedAt(observedAt.UTC()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("persist provider rate limit for provider %s: %w", providerID, err)
	}
	if !snapshotChanged {
		return nil
	}

	projectIDs, err := providerActivityProjectIDs(ctx, l.client, updatedProvider.OrganizationID, updatedProvider.ID)
	if err != nil {
		return fmt.Errorf("list provider rate limit activity projects for provider %s: %w", providerID, err)
	}

	emitter := activitysvc.NewEmitter(activitysvc.EntRecorder{Client: l.client}, l.events)
	for _, projectID := range projectIDs {
		if _, err := emitter.Emit(ctx, activitysvc.RecordInput{
			ProjectID: projectID,
			EventType: activityevent.TypeProviderRateLimitUpdated,
			Message:   fmt.Sprintf("Updated provider rate limit snapshot for %s", updatedProvider.Name),
			Metadata: map[string]any{
				"provider_id":           updatedProvider.ID.String(),
				"provider_name":         updatedProvider.Name,
				"machine_id":            updatedProvider.MachineID.String(),
				"rate_limit":            cloneLifecycleMetadata(payload),
				"rate_limit_updated_at": observedAt.UTC().Format(time.RFC3339),
				"changed_fields":        []string{"cli_rate_limit"},
			},
			CreatedAt: observedAt.UTC(),
		}); err != nil {
			return fmt.Errorf("emit provider rate limit activity for provider %s: %w", providerID, err)
		}
	}

	return nil
}

func marshalProviderRateLimit(rateLimit *provider.CLIRateLimit) (map[string]any, error) {
	if rateLimit == nil {
		return nil, nil
	}

	payload, err := json.Marshal(rateLimit)
	if err != nil {
		return nil, err
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}

	return decoded, nil
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
	if ticket == nil || ticket.Edges.CurrentRun == nil {
		return false
	}
	runWorkflowID := ticket.Edges.CurrentRun.WorkflowID
	if runWorkflowID == uuid.Nil && ticket.WorkflowID != nil {
		runWorkflowID = *ticket.WorkflowID
	}
	return classifyRuntimeTicket(ticket, runID, runWorkflowID) == runtimeTicketActive
}

func (l *RuntimeLauncher) releaseExecutionOwnership(ctx context.Context, runID uuid.UUID, agentID uuid.UUID, ticket *ent.Ticket) error {
	if ticket == nil {
		return fmt.Errorf("ticket missing for execution release")
	}

	stopSession(context.Background(), l.loadSession(runID))
	l.deleteSession(runID)
	l.runtime.delete(runID)

	tx, err := l.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start execution release tx: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.Ticket.Update().
		Where(
			entticket.IDEQ(ticket.ID),
			entticket.CurrentRunIDEQ(runID),
		).
		ClearCurrentRunID().
		Save(ctx); err != nil {
		return fmt.Errorf("release ticket %s execution ownership: %w", ticket.ID, err)
	}

	if _, err := tx.Agent.Update().
		Where(entagent.IDEQ(agentID)).
		SetRuntimeControlState(entagent.RuntimeControlStateActive).
		Save(ctx); err != nil {
		return fmt.Errorf("reset agent %s after execution release: %w", agentID, err)
	}

	if _, err := clearRuntimeStateOne(
		tx.AgentRun.UpdateOneID(runID).
			SetStatus(entagentrun.StatusTerminated).
			SetTerminalAt(l.now().UTC()),
	).Save(ctx); err != nil {
		return fmt.Errorf("reset run %s after execution release: %w", runID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit execution release tx: %w", err)
	}
	if err := catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, runID, l.now().UTC()); err != nil {
		return err
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
	l.prepareRunCompletionSummaryBestEffort(ctx, runID)
	l.scheduleRunCompletionSummary(runID)
	if ticketReachedWorkflowFinish(ticket) {
		l.cleanupRunWorkspacesBestEffort(ctx, runID, "execution release")
	}
	return nil
}

func (l *RuntimeLauncher) finishResolvedExecution(ctx context.Context, runID uuid.UUID, agentID uuid.UUID, ticket *ent.Ticket) error {
	stopSession(context.Background(), l.loadSession(runID))
	l.deleteSession(runID)
	l.runtime.delete(runID)

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

	ticketUpdate := ticketrepo.ResetRetryBaseline(tx.Ticket.UpdateOneID(ticket.ID), ticket)
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
			SetStatus(entagentrun.StatusCompleted).
			SetTerminalAt(now),
	).Save(ctx); err != nil {
		return fmt.Errorf("update run %s after execution: %w", runID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit finish execution tx: %w", err)
	}
	if err := catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, runID, now); err != nil {
		return err
	}
	l.tickets.RunLifecycleHookBestEffort(ctx, ticketservice.RunLifecycleHookInput{
		TicketID: ticket.ID,
		RunID:    runID,
		HookName: infrahook.TicketHookOnDone,
	})

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
	l.prepareRunCompletionSummaryBestEffort(ctx, runID)
	l.scheduleRunCompletionSummary(runID)
	l.cleanupRunWorkspacesBestEffort(ctx, runID, "execution finished")
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
	l.logger.Error(
		"runtime execution failed",
		"run_id", runID,
		"agent_id", agentID,
		"ticket_id", ticketID,
		"failure_stage", string(runtimeExecutionStageProcessStreaming),
		"error", failure,
	)
	stopSession(context.Background(), l.loadSession(runID))
	l.deleteSession(runID)
	l.runtime.delete(runID)

	suppressFailure, err := l.shouldSuppressExecutionFailure(ctx, runID, ticketID)
	if err != nil {
		l.logger.Warn("check execution failure suppression", "run_id", runID, "ticket_id", ticketID, "error", err)
	} else if suppressFailure {
		return
	}

	now := l.now().UTC()
	if _, err := l.client.AgentRun.Update().
		Where(entagentrun.IDEQ(runID)).
		SetStatus(entagentrun.StatusErrored).
		SetTerminalAt(now).
		SetLastError(strings.TrimSpace(failure.Error())).
		Save(ctx); err == nil {
		_ = catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, runID, now)
		l.tickets.RunLifecycleHookBestEffort(ctx, ticketservice.RunLifecycleHookInput{
			TicketID: ticketID,
			RunID:    runID,
			HookName: infrahook.TicketHookOnError,
		})
		if failedAgent, err := loadAgentLifecycleState(ctx, l.client, agentID, &runID); err == nil {
			l.publishLifecycleEvent(
				ctx,
				agentFailedType,
				failedAgent,
				lifecycleMessage(agentFailedType, failedAgent.agent.Name),
				mergeRuntimeFailureMetadata(runtimeEventMetadataForState(failedAgent), &runtimeLaunchFailure{stage: runtimeExecutionStageProcessStreaming, cause: failure}),
				now,
			)
		}
	}
	l.prepareRunCompletionSummaryBestEffort(ctx, runID)
	l.scheduleRunCompletionSummary(runID)

	retrySvc := NewRetryService(l.client, l.logger)
	retrySvc.now = l.now
	if _, err := retrySvc.MarkAttemptFailed(ctx, ticketID); err != nil {
		l.logger.Error("mark execution failed retry", "ticket_id", ticketID, "agent_id", agentID, "error", err)
	}
}

func (l *RuntimeLauncher) shouldSuppressExecutionFailure(ctx context.Context, runID uuid.UUID, ticketID uuid.UUID) (bool, error) {
	if l == nil || l.client == nil {
		return false, fmt.Errorf("runtime launcher unavailable")
	}

	runItem, err := l.client.AgentRun.Get(ctx, runID)
	if err != nil {
		return false, fmt.Errorf("load run %s for failure suppression: %w", runID, err)
	}
	if runItem.Status == entagentrun.StatusCompleted ||
		runItem.Status == entagentrun.StatusErrored ||
		runItem.Status == entagentrun.StatusTerminated {
		return true, nil
	}

	ticketItem, err := l.reloadExecutionTicket(ctx, ticketID)
	if err != nil {
		return false, fmt.Errorf("reload ticket %s for failure suppression: %w", ticketID, err)
	}
	return ticketReachedWorkflowFinish(ticketItem), nil
}

func ticketReachedWorkflowFinish(ticket *ent.Ticket) bool {
	if ticket == nil {
		return false
	}
	if ticket.CompletedAt != nil {
		return true
	}
	if ticket.Edges.Workflow == nil {
		return false
	}
	return slices.Contains(ticketStatusIDs(ticket.Edges.Workflow.Edges.FinishStatuses), ticket.StatusID)
}

func (l *RuntimeLauncher) scheduleContinuation(ctx context.Context, runID uuid.UUID, agentID uuid.UUID, ticketID uuid.UUID) error {
	session := l.loadSession(runID)
	stopSession(context.Background(), session)
	l.deleteSession(runID)
	l.runtime.delete(runID)

	tx, err := l.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start continuation tx: %w", err)
	}
	defer rollback(tx)

	if _, err := ticketrepo.ScheduleRetry(
		tx.Ticket.Update().
			Where(
				entticket.IDEQ(ticketID),
				entticket.CurrentRunIDEQ(runID),
			).
			ClearCurrentRunID().
			SetStallCount(0),
		l.now().UTC().Add(continuationRetryDelay),
		"",
	).
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
			SetStatus(entagentrun.StatusTerminated).
			SetTerminalAt(l.now().UTC()),
	).Save(ctx); err != nil {
		return fmt.Errorf("reset run %s after continuation limit: %w", runID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit continuation tx: %w", err)
	}
	if err := catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, runID, l.now().UTC()); err != nil {
		return err
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

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func maxInt64(left int64, right int64) int64 {
	if right > left {
		return right
	}
	return left
}
