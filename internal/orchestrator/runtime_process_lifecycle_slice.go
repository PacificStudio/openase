package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	ticketingdomain "github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	infrahook "github.com/BetterAndBetterII/openase/internal/infra/hook"
	machinetransport "github.com/BetterAndBetterII/openase/internal/infra/machinetransport"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
)

func (s runtimeProcessLifecycleSlice) runLaunch(ctx context.Context, assignment runtimeAssignment) {
	l := s.launcher
	if l == nil || assignment.run == nil {
		return
	}
	defer l.finishLaunch(assignment.run.ID)

	err := s.launchAgent(ctx, assignment)
	if err == nil {
		return
	}

	logAttrs := []any{
		"agent_id", assignment.agent.ID,
		"run_id", assignment.run.ID,
		"ticket_id", assignment.ticket.ID,
		"error", err,
	}
	if details := runtimeLaunchFailureDetails(err); details != nil {
		if details.stage != "" {
			logAttrs = append(logAttrs, "failure_stage", string(details.stage))
		}
		if details.machineID != uuid.Nil {
			logAttrs = append(logAttrs, "machine_id", details.machineID.String())
		}
		if strings.TrimSpace(details.transportMode) != "" {
			logAttrs = append(logAttrs, "transport_mode", details.transportMode)
		}
		if strings.TrimSpace(details.workspaceRoot) != "" {
			logAttrs = append(logAttrs, "workspace_root", details.workspaceRoot)
		}
	}
	l.logger.Error("launch current run", logAttrs...)
	if assignment.agent == nil || assignment.run == nil || assignment.ticket == nil {
		return
	}

	failureCtx, failureCancel := l.launchContext(ctx, defaultLaunchCleanupTimeout)
	defer failureCancel()
	if markErr := s.markLaunchFailed(failureCtx, assignment.agent.ID, assignment.ticket.ID, assignment.run.ID, err); markErr != nil {
		l.logger.Error("mark launch failed", "agent_id", assignment.agent.ID, "ticket_id", assignment.ticket.ID, "run_id", assignment.run.ID, "error", markErr)
	}
}

func (s runtimeProcessLifecycleSlice) launchAgent(ctx context.Context, assignment runtimeAssignment) error {
	l := s.launcher
	if assignment.agent == nil || assignment.run == nil || assignment.ticket == nil {
		return nil
	}

	now := l.now().UTC()
	launchingCount, err := l.client.AgentRun.Update().
		Where(
			entagentrun.IDEQ(assignment.run.ID),
			entagentrun.StatusIn(entagentrun.StatusLaunching, entagentrun.StatusTerminated),
		).
		SetStatus(entagentrun.StatusLaunching).
		SetLastError("").
		ClearSessionID().
		ClearRuntimeStartedAt().
		ClearLastHeartbeatAt().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark run %s launching: %w", assignment.run.ID, err)
	}
	if launchingCount == 0 {
		return nil
	}

	launchingAgent, err := loadAgentLifecycleState(ctx, l.client, assignment.agent.ID, &assignment.run.ID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentLaunchingType,
		launchingAgent,
		lifecycleMessage(agentLaunchingType, launchingAgent.agent.Name),
		runtimeEventMetadataForState(launchingAgent),
		now,
	)

	session, launchErr := s.startRuntimeSessionWithTimeout(ctx, assignment)
	if launchErr != nil {
		return launchErr
	}

	l.storeSession(assignment.run.ID, session)

	readyAt := l.now().UTC()
	readyCount, err := l.client.AgentRun.Update().
		Where(
			entagentrun.IDEQ(assignment.run.ID),
			entagentrun.StatusEQ(entagentrun.StatusLaunching),
		).
		SetStatus(entagentrun.StatusReady).
		SetRuntimeStartedAt(readyAt).
		SetLastHeartbeatAt(readyAt).
		SetLastError("").
		Save(ctx)
	if err == nil {
		if sessionID, ok := session.SessionID(); ok {
			readyCount, err = l.client.AgentRun.Update().
				Where(
					entagentrun.IDEQ(assignment.run.ID),
					entagentrun.StatusEQ(entagentrun.StatusReady),
				).
				SetSessionID(sessionID).
				Save(ctx)
		}
	}
	if err != nil {
		l.deleteSession(assignment.run.ID)
		stopSession(context.Background(), session)
		return fmt.Errorf("mark run %s ready: %w", assignment.run.ID, err)
	}
	if readyCount == 0 {
		l.deleteSession(assignment.run.ID)
		stopSession(context.Background(), session)
		return nil
	}
	sessionID, _ := session.SessionID()
	l.runtime.markReady(
		assignment.run.ID,
		assignment.agent.ID,
		assignment.ticket.ID,
		assignment.run.WorkflowID,
		sessionID,
		readyAt,
	)

	readyAgent, err := loadAgentLifecycleState(ctx, l.client, assignment.agent.ID, &assignment.run.ID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentReadyType,
		readyAgent,
		lifecycleMessage(agentReadyType, readyAgent.agent.Name),
		runtimeEventMetadataForState(readyAgent),
		readyAt,
	)
	l.publishLifecycleEvent(
		ctx,
		agentHeartbeatType,
		readyAgent,
		lifecycleMessage(agentHeartbeatType, readyAgent.agent.Name),
		runtimeEventMetadataForState(readyAgent),
		readyAt,
	)

	return nil
}

func (s runtimeProcessLifecycleSlice) startRuntimeSessionWithTimeout(ctx context.Context, assignment runtimeAssignment) (agentSession, error) {
	return s.startRuntimeSession(ctx, assignment)
}

func (s runtimeProcessLifecycleSlice) startAgentSessionWithTimeout(
	ctx context.Context,
	start func(context.Context) (agentSession, error),
) (agentSession, error) {
	l := s.launcher
	timeout := l.agentSessionStartTimeout
	if timeout <= 0 {
		return start(ctx)
	}

	startCtx, cancel := context.WithCancel(ctx)

	type startResult struct {
		session agentSession
		err     error
	}

	resultCh := make(chan startResult)
	//nolint:gosec // session start timeout cleanup needs a detached stop context to reclaim late sessions safely.
	go func() {
		session, err := start(startCtx)
		select {
		case resultCh <- startResult{session: session, err: err}:
		case <-startCtx.Done():
			stopCtx, stopCancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer stopCancel()
			stopSession(stopCtx, session)
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-resultCh:
		return result.session, result.err
	case <-timer.C:
		cancel()
		return nil, fmt.Errorf("start runtime agent session timed out after %s", timeout)
	case <-ctx.Done():
		cancel()
		return nil, ctx.Err()
	}
}

func (s runtimeProcessLifecycleSlice) markLaunchFailed(ctx context.Context, agentID uuid.UUID, ticketID uuid.UUID, runID uuid.UUID, launchErr error) error {
	l := s.launcher
	now := l.now().UTC()
	count, err := l.client.AgentRun.Update().
		Where(
			entagentrun.IDEQ(runID),
			entagentrun.StatusEQ(entagentrun.StatusLaunching),
		).
		SetStatus(entagentrun.StatusErrored).
		SetTerminalAt(now).
		SetLastError(strings.TrimSpace(launchErr.Error())).
		ClearSessionID().
		ClearRuntimeStartedAt().
		ClearLastHeartbeatAt().
		Save(ctx)
	if err != nil {
		return fmt.Errorf("mark run %s failed: %w", runID, err)
	}
	if count == 0 {
		return nil
	}
	s.recordLaunchFailureMetric(launchErr)
	if err := catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, runID, now); err != nil {
		return err
	}
	l.tickets.RunLifecycleHookBestEffort(ctx, ticketservice.RunLifecycleHookInput{
		TicketID: ticketID,
		RunID:    runID,
		HookName: infrahook.TicketHookOnError,
	})

	retrySvc := NewRetryService(l.client, l.logger, l.events)
	retrySvc.now = l.now
	if _, err := retrySvc.MarkAttemptFailed(ctx, ticketID); err != nil {
		return fmt.Errorf("release failed launch claim for ticket %s: %w", ticketID, err)
	}

	failedAgent, err := loadAgentLifecycleState(ctx, l.client, agentID, &runID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentFailedType,
		failedAgent,
		lifecycleMessage(agentFailedType, failedAgent.agent.Name),
		mergeRuntimeFailureMetadata(runtimeEventMetadataForState(failedAgent), launchErr),
		now,
	)
	l.prepareRunCompletionSummaryBestEffort(ctx, runID)
	l.scheduleRunCompletionSummary(runID)
	return nil
}

func (s runtimeProcessLifecycleSlice) recordLaunchFailureMetric(launchErr error) {
	l := s.launcher
	if l == nil || l.metrics == nil {
		return
	}
	stage := string(runtimeLaunchStageProcessStart)
	transportMode := ""
	if details := runtimeLaunchFailureDetails(launchErr); details != nil {
		if details.stage != "" {
			stage = string(details.stage)
		}
		transportMode = strings.TrimSpace(details.transportMode)
	}
	l.metrics.Counter("openase.runtime.launch_failures_total", provider.Tags{
		"failure_stage":  stage,
		"transport_mode": transportMode,
	}).Add(1)
}

func (s runtimeProcessLifecycleSlice) pauseAgent(ctx context.Context, assignment runtimeAssignment) error {
	l := s.launcher
	if assignment.agent == nil || assignment.run == nil {
		return nil
	}

	session := l.loadSession(assignment.run.ID)
	if session != nil {
		stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		stopErr := session.Stop(stopCtx)
		cancel()
		if stopErr != nil {
			return fmt.Errorf("stop runtime session for run %s: %w", assignment.run.ID, stopErr)
		}
		l.deleteSession(assignment.run.ID)
	}

	pausedAt := l.now().UTC()
	pausedCount, err := clearRuntimeState(
		l.client.AgentRun.Update().
			Where(
				entagentrun.IDEQ(assignment.run.ID),
				entagentrun.StatusIn(entagentrun.StatusLaunching, entagentrun.StatusReady, entagentrun.StatusExecuting),
			).
			SetStatus(entagentrun.StatusTerminated).
			SetTerminalAt(pausedAt),
	).Save(ctx)
	if err != nil {
		return fmt.Errorf("mark agent %s paused: %w", assignment.agent.ID, err)
	}
	if pausedCount == 0 {
		return nil
	}
	if err := catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, assignment.run.ID, pausedAt); err != nil {
		return err
	}

	if _, err := l.client.Agent.UpdateOneID(assignment.agent.ID).
		SetRuntimeControlState(entagent.RuntimeControlStatePaused).
		Save(ctx); err != nil {
		return fmt.Errorf("mark agent %s control paused: %w", assignment.agent.ID, err)
	}

	pausedAgent, err := loadAgentLifecycleState(ctx, l.client, assignment.agent.ID, &assignment.run.ID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentPausedType,
		pausedAgent,
		lifecycleMessage(agentPausedType, pausedAgent.agent.Name),
		runtimeEventMetadataForState(pausedAgent),
		pausedAt,
	)
	return nil
}

func (s runtimeProcessLifecycleSlice) interruptAgent(ctx context.Context, assignment runtimeAssignment) error {
	l := s.launcher
	if assignment.agent == nil || assignment.run == nil || assignment.ticket == nil {
		return nil
	}

	session := l.loadSession(assignment.run.ID)
	if session != nil {
		stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		stopErr := session.Stop(stopCtx)
		cancel()
		if stopErr != nil {
			return fmt.Errorf("interrupt runtime session for run %s: %w", assignment.run.ID, stopErr)
		}
		l.deleteSession(assignment.run.ID)
	}
	l.runtime.delete(assignment.run.ID)

	interruptedAt := l.now().UTC()
	tx, err := l.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start interrupt tx for run %s: %w", assignment.run.ID, err)
	}
	defer rollback(tx)

	runCount, err := clearRuntimeState(
		tx.AgentRun.Update().
			Where(
				entagentrun.IDEQ(assignment.run.ID),
				entagentrun.StatusIn(entagentrun.StatusLaunching, entagentrun.StatusReady, entagentrun.StatusExecuting),
			).
			SetStatus(entagentrun.StatusInterrupted).
			SetTerminalAt(interruptedAt),
	).Save(ctx)
	if err != nil {
		return fmt.Errorf("mark agent %s interrupted: %w", assignment.agent.ID, err)
	}

	ticketUpdate := ticketrepo.ResetRetryBaseline(tx.Ticket.UpdateOneID(assignment.ticket.ID), assignment.ticket)
	if assignment.ticket.CurrentRunID != nil && *assignment.ticket.CurrentRunID == assignment.run.ID {
		ticketUpdate.ClearCurrentRunID()
	}
	ticketUpdate.
		SetRetryPaused(true).
		SetPauseReason(ticketingdomain.PauseReasonUserInterrupted.String())
	ticketUpdate.ClearNextRetryAt()
	if _, err := ticketUpdate.Save(ctx); err != nil {
		return fmt.Errorf("pause ticket %s after interrupt: %w", assignment.ticket.ID, err)
	}

	if _, err := tx.Agent.UpdateOneID(assignment.agent.ID).
		SetRuntimeControlState(entagent.RuntimeControlStateActive).
		Save(ctx); err != nil {
		return fmt.Errorf("reset agent %s control after interrupt: %w", assignment.agent.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit interrupt tx for run %s: %w", assignment.run.ID, err)
	}
	if runCount == 0 {
		return nil
	}
	if err := catalogrepo.MaterializeAgentRunDailyUsage(ctx, l.client, assignment.run.ID, interruptedAt); err != nil {
		return err
	}
	if err := l.eventSlice().recordAgentStep(
		ctx,
		assignment.agent.ProjectID,
		assignment.agent.ID,
		assignment.ticket.ID,
		assignment.run.ID,
		"interrupted",
		"Interrupted by operator request.",
		nil,
	); err != nil {
		return fmt.Errorf("record interrupt step for run %s: %w", assignment.run.ID, err)
	}

	interruptedAgent, err := loadAgentLifecycleState(ctx, l.client, assignment.agent.ID, &assignment.run.ID)
	if err != nil {
		return err
	}
	l.publishLifecycleEvent(
		ctx,
		agentInterruptedType,
		interruptedAgent,
		lifecycleMessage(agentInterruptedType, interruptedAgent.agent.Name),
		runtimeEventMetadataForState(interruptedAgent),
		interruptedAt,
	)
	l.prepareRunCompletionSummaryBestEffort(ctx, assignment.run.ID)
	l.scheduleRunCompletionSummary(assignment.run.ID)
	return nil
}

func (s runtimeProcessLifecycleSlice) startRuntimeSession(ctx context.Context, assignment runtimeAssignment) (agentSession, error) {
	l := s.launcher
	launchContext, err := l.selectionSlice().loadLaunchContext(ctx, assignment.agent.ID, assignment.ticket.ID)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(catalogdomain.Machine{}, "", runtimeLaunchStageContext, err)
	}

	machine, remote, err := l.selectionSlice().resolveLaunchMachine(ctx, launchContext)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(catalogdomain.Machine{}, "", runtimeLaunchStageResolveMachine, err)
	}

	session, err := s.startRuntimeSessionOnMachine(ctx, assignment, launchContext, machine, remote)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s runtimeProcessLifecycleSlice) startRuntimeSessionOnMachine(
	ctx context.Context,
	assignment runtimeAssignment,
	launchContext runtimeLaunchContext,
	machine catalogdomain.Machine,
	remote bool,
) (agentSession, error) {
	l := s.launcher
	workspaceRoot := ""
	if remote && machine.WorkspaceRoot != nil {
		workspaceRoot = strings.TrimSpace(*machine.WorkspaceRoot)
	}
	if remote && machine.ConnectionMode == catalogdomain.MachineConnectionModeSSH {
		return nil, wrapRuntimeLaunchFailure(
			machine,
			workspaceRoot,
			runtimeLaunchStageTransportResolve,
			fmt.Errorf("ssh runtime execution is no longer supported for machine %s; migrate the machine to websocket execution and use SSH only for bootstrap or diagnostics", machine.Name),
		)
	}

	command, err := provider.ParseAgentCLICommand(launchContext.agent.Edges.Provider.CliCommand)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageProcessStart, fmt.Errorf("parse agent cli command: %w", err))
	}
	environment, err := l.buildRuntimeAgentEnvironment(
		ctx,
		machine.EnvVars,
		launchContext.project.ID,
		launchContext.agent.Edges.Provider.AuthConfig,
		&assignment.ticket.ID,
		launchContext.ticket.WorkflowID,
		&assignment.agent.ID,
	)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageContext, err)
	}
	platformAccess, err := l.workspaceSlice().buildAgentPlatformAccess(ctx, launchContext)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageContext, err)
	}
	environment = append(environment, platformAccess.environment...)
	runtimeSecrets, err := l.workspaceSlice().buildRuntimeSecretEnvironment(ctx, launchContext)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageContext, err)
	}
	environment = append(environment, runtimeSecrets...)
	githubEnvironment, err := l.workspaceSlice().buildGitHubOutboundEnvironment(ctx, launchContext.project.ID, environment)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageContext, err)
	}
	environment = append(environment, githubEnvironment...)
	environment, err = buildMachineOpenASEEnvironment(machine, remote, environment)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageContext, err)
	}
	if requiresMachineCodexReady(command, environment) {
		if ready, reason, ok := machineCodexReady(machine.Resources); ok && !ready {
			return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, runtimeLaunchStageAgentCLIPreflight, fmt.Errorf("machine %s codex environment not ready: %s", machine.Name, reason))
		}
	}

	workspaceItem, err := l.prepareTicketWorkspace(ctx, assignment.run.ID, launchContext, machine, remote)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceRoot, classifyRuntimeLaunchWorkspaceStage(err), err)
	}
	if err := l.tickets.RunLifecycleHook(ctx, ticketservice.RunLifecycleHookInput{
		TicketID: assignment.ticket.ID,
		RunID:    assignment.run.ID,
		HookName: infrahook.TicketHookOnClaim,
		Blocking: true,
	}); err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workspaceItem.Path, runtimeLaunchStageHookOnClaim, fmt.Errorf("run ticket on_claim hooks: %w", err))
	}

	workingDirectoryValue := resolveAgentWorkingDirectory(launchContext, workspaceItem)
	var runtimeSnapshot workflowservice.RuntimeSnapshot
	if l.workflow != nil && launchContext.ticket.WorkflowID != nil {
		runtimeSnapshot, err = l.materializeRuntimeSnapshot(
			ctx,
			assignment.run.ID,
			*launchContext.ticket.WorkflowID,
			machine,
			workingDirectoryValue,
			string(launchContext.agent.Edges.Provider.AdapterType),
			remote,
		)
		if err != nil {
			return nil, wrapRuntimeLaunchFailure(machine, workspaceItem.Path, runtimeLaunchStageRuntimeSnapshot, fmt.Errorf("materialize runtime snapshot: %w", err))
		}
	}
	preflightCommand := strings.TrimSpace(launchContext.agent.Edges.Provider.CliCommand)
	if resolved := catalogdomain.ResolveMachineAgentCLIPath(
		machine,
		catalogdomain.AgentProviderAdapterType(launchContext.agent.Edges.Provider.AdapterType),
	); resolved != nil {
		preflightCommand = *resolved
	}
	if err := l.runRemoteRuntimePreflight(ctx, machine, remote, workingDirectoryValue, preflightCommand, environment); err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectoryValue, classifyRuntimeLaunchPreflightStage(err), err)
	}
	workingDirectory, err := provider.ParseAbsolutePath(workingDirectoryValue)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectoryValue, runtimeLaunchStageWorkspaceRoot, fmt.Errorf("parse agent workspace path: %w", err))
	}
	developerInstructions, err := l.workspaceSlice().buildDeveloperInstructions(
		ctx,
		launchContext,
		machine,
		workingDirectory.String(),
		runtimeSnapshot,
		platformAccess.contract,
	)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageBuildInstructions, err)
	}
	if err := l.tickets.RunLifecycleHook(ctx, ticketservice.RunLifecycleHookInput{
		TicketID: assignment.ticket.ID,
		RunID:    assignment.run.ID,
		HookName: infrahook.TicketHookOnStart,
		Blocking: true,
	}); err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageHookOnStart, fmt.Errorf("run ticket on_start hooks: %w", err))
	}

	processManager := l.processManager
	if l.transports != nil {
		transport, transportErr := l.transports.Resolve(machine)
		if transportErr != nil {
			return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageTransportResolve, transportErr)
		}
		processManager = machinetransport.NewProcessManager(transport, machine)
	}

	processSpec, err := provider.NewAgentCLIProcessSpec(
		command,
		launchContext.agent.Edges.Provider.CliArgs,
		&workingDirectory,
		environment,
	)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageProcessStart, fmt.Errorf("build agent process spec: %w", err))
	}

	adapter, err := l.adapters.adapterFor(launchContext.agent.Edges.Provider.AdapterType)
	if err != nil {
		return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageProcessStart, err)
	}

	return s.startAgentSessionWithTimeout(ctx, func(sessionCtx context.Context) (agentSession, error) {
		session, err := adapter.Start(sessionCtx, agentSessionStartSpec{
			Process:               processSpec,
			ProcessManager:        processManager,
			WorkingDirectory:      workingDirectory.String(),
			Model:                 launchContext.agent.Edges.Provider.ModelName,
			ReasoningEffort:       catalogdomain.ParseStoredAgentProviderReasoningEffort(launchContext.agent.Edges.Provider.ReasoningEffort),
			PermissionProfile:     catalogdomain.AgentProviderPermissionProfile(launchContext.agent.Edges.Provider.PermissionProfile),
			DeveloperInstructions: developerInstructions,
			TurnTitle:             fmt.Sprintf("%s: %s", launchContext.ticket.Identifier, launchContext.ticket.Title),
		})
		if err != nil {
			return nil, wrapRuntimeLaunchFailure(machine, workingDirectory.String(), runtimeLaunchStageProcessStart, err)
		}
		return session, nil
	})
}
