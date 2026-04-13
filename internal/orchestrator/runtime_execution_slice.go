package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	"github.com/BetterAndBetterII/openase/internal/agentplatform"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/google/uuid"
)

func (s runtimeExecutionSlice) startReadyExecutions(ctx context.Context) error {
	l := s.launcher
	if l == nil || l.client == nil {
		return nil
	}

	assignments, err := l.selectionSlice().listAssignments(ctx,
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
		go s.runReadyExecution(ctx, assignment.run.ID)
	}

	return nil
}

func (s runtimeExecutionSlice) runReadyExecution(ctx context.Context, runID uuid.UUID) {
	l := s.launcher
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
		state, prompt, err := s.loadExecutionState(ctx, runID, turnNumber, lastError)
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
		if err := l.eventSlice().recordAgentStep(ctx, state.agent.ProjectID, state.agent.ID, state.ticket.ID, state.run.ID, "planning", fmt.Sprintf("Started turn %d.", turnNumber), nil); err != nil {
			l.logger.Warn("record agent planning step", "run_id", state.run.ID, "error", err)
		}

		if err := s.consumeTurn(
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

func (s runtimeExecutionSlice) loadExecutionState(ctx context.Context, runID uuid.UUID, turnNumber int, lastError string) (runtimeExecutionState, string, error) {
	l := s.launcher
	assignment, err := l.selectionSlice().loadAssignmentByRun(ctx, runID)
	if err != nil {
		return runtimeExecutionState{}, "", err
	}

	if assignment.agent == nil || assignment.ticket == nil || assignment.run == nil {
		return runtimeExecutionState{}, "", fmt.Errorf("run %s no longer has an active assignment", runID)
	}

	launchContext, err := l.selectionSlice().loadLaunchContext(ctx, assignment.agent.ID, assignment.ticket.ID)
	if err != nil {
		return runtimeExecutionState{}, "", err
	}

	machine, remote, err := l.selectionSlice().resolveLaunchMachine(ctx, launchContext)
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
	developerInstructions, err := l.workspaceSlice().buildDeveloperInstructions(
		ctx,
		launchContext,
		machine,
		workspace,
		runtimeSnapshot,
		l.workspaceSlice().ticketRuntimePlatformContract(launchContext, agentplatform.DefaultScopesForPrincipalKind(agentplatform.PrincipalKindTicketAgent)),
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

func (s runtimeExecutionSlice) consumeTurn(
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
	l := s.launcher
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
		if err := l.eventSlice().recordAgentOutput(ctx, projectID, agentID, ticketID, runID, adapterType, persisted); err != nil {
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

		if err := l.eventSlice().persistRuntimeSessionID(ctx, runID, session); err != nil {
			l.logger.Warn("persist runtime session id", "run_id", runID, "error", err)
		}

		if err := l.eventSlice().touchHeartbeat(ctx, runID); err != nil {
			l.logger.Warn("update agent heartbeat", "run_id", runID, "error", err)
		}
		if err := l.eventSlice().projectRuntimeEvent(ctx, runtimeEventProjectionInput{
			ProjectID:  projectID,
			AgentID:    agentID,
			TicketID:   ticketID,
			RunID:      runID,
			Provider:   runtimeProviderName(adapterType),
			ObservedAt: observedAt,
			Event:      event,
		}); err != nil {
			return fmt.Errorf("project runtime event for run %s: %w", runID, err)
		}

		switch event.Type {
		case agentEventTypeToolCallRequested:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.ToolCall == nil {
				continue
			}
			if err := l.eventSlice().recordAgentToolCall(ctx, projectID, agentID, ticketID, runID, adapterType, event.ToolCall); err != nil {
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
			if err := l.eventSlice().recordAgentApprovalRequest(ctx, projectID, agentID, ticketID, runID, adapterType, event.Approval); err != nil {
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
			if err := l.eventSlice().recordAgentUserInputRequest(ctx, projectID, agentID, ticketID, runID, adapterType, event.UserInput); err != nil {
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
			if err := l.eventSlice().recordTokenUsage(ctx, agentID, runID, ticketID, event.TokenUsage, highWater); err != nil {
				return err
			}
		case agentEventTypeRateLimitUpdated:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.RateLimit == nil {
				continue
			}
			if err := l.eventSlice().recordProviderRateLimit(ctx, providerID, event.RateLimit, observedAt); err != nil {
				return err
			}
		case agentEventTypeThreadStatus:
			if err := flushBufferedOutputs(); err != nil {
				return err
			}
			if event.Thread == nil {
				continue
			}
			if err := l.eventSlice().recordAgentThreadStatus(ctx, projectID, agentID, ticketID, runID, adapterType, event.Thread); err != nil {
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
			if err := l.eventSlice().recordAgentTurnDiff(ctx, projectID, agentID, ticketID, runID, adapterType, event.Diff); err != nil {
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
			if err := l.eventSlice().recordAgentReasoning(ctx, projectID, agentID, ticketID, runID, adapterType, event.Reasoning); err != nil {
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
			if err := l.eventSlice().recordAgentTaskStatus(ctx, projectID, agentID, ticketID, runID, adapterType, event.TaskStatus); err != nil {
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
