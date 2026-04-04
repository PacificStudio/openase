package orchestrator

import (
	"context"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
)

func (l *RuntimeLauncher) ensureSessionRegistry() *runtimeSessionRegistry {
	if l == nil {
		return nil
	}
	if l.sessions == nil {
		l.sessions = newRuntimeSessionRegistry()
	}
	return l.sessions
}

func (l *RuntimeLauncher) ensureLaunchTracker() *runtimeRunTracker {
	if l == nil {
		return nil
	}
	if l.launches == nil {
		l.launches = newRuntimeRunTracker()
	}
	return l.launches
}

func (l *RuntimeLauncher) ensureExecutionTracker() *runtimeRunTracker {
	if l == nil {
		return nil
	}
	if l.executions == nil {
		l.executions = newRuntimeRunTracker()
	}
	return l.executions
}

func (l *RuntimeLauncher) ensureWorkspaceProvisioner() *runtimeWorkspaceProvisioner {
	if l == nil {
		return nil
	}
	now := l.now
	if now == nil {
		now = time.Now
	}
	if l.workspaces == nil {
		l.workspaces = newRuntimeWorkspaceProvisioner(l.client, l.logger, l.sshPool, now)
	}
	l.workspaces.client = l.client
	l.workspaces.logger = l.logger.With("component", "runtime-workspace-provisioner")
	l.workspaces.sshPool = l.sshPool
	l.workspaces.transports = l.transports
	l.workspaces.githubAuth = l.githubAuth
	l.workspaces.now = now
	return l.workspaces
}

func (l *RuntimeLauncher) ensureCompletionSummaryCoordinator() *runtimeCompletionSummaryCoordinator {
	if l == nil {
		return nil
	}
	now := l.now
	if now == nil {
		now = time.Now
	}
	if l.completionSummaries == nil {
		l.completionSummaries = newRuntimeCompletionSummaryCoordinator(
			l.client,
			l.logger,
			l.events,
			l.adapters,
			l.processManager,
			l.sshPool,
			l.workflow,
			now,
			defaultCompletionSummaryTimeout,
		)
	}
	l.completionSummaries.client = l.client
	l.completionSummaries.logger = l.logger.With("component", "runtime-completion-summary")
	l.completionSummaries.adapters = l.adapters
	l.completionSummaries.processManager = l.processManager
	l.completionSummaries.sshPool = l.sshPool
	l.completionSummaries.transports = l.transports
	l.completionSummaries.workflow = l.workflow
	l.completionSummaries.now = now
	return l.completionSummaries
}

func (l *RuntimeLauncher) storeSession(runID uuid.UUID, session agentSession) {
	if registry := l.ensureSessionRegistry(); registry != nil {
		registry.store(runID, session)
	}
}

func (l *RuntimeLauncher) loadSession(runID uuid.UUID) agentSession {
	registry := l.ensureSessionRegistry()
	if registry == nil {
		return nil
	}
	return registry.load(runID)
}

func (l *RuntimeLauncher) deleteSession(runID uuid.UUID) {
	if registry := l.ensureSessionRegistry(); registry != nil {
		registry.delete(runID)
	}
}

func (l *RuntimeLauncher) drainSessions() map[uuid.UUID]agentSession {
	registry := l.ensureSessionRegistry()
	if registry == nil {
		return nil
	}
	return registry.drain()
}

func (l *RuntimeLauncher) sessionRunIDs() []uuid.UUID {
	registry := l.ensureSessionRegistry()
	if registry == nil {
		return nil
	}
	return registry.runIDs()
}

func (l *RuntimeLauncher) beginExecution(runID uuid.UUID) bool {
	tracker := l.ensureExecutionTracker()
	if tracker == nil {
		return false
	}
	return tracker.begin(runID)
}

func (l *RuntimeLauncher) finishExecution(runID uuid.UUID) {
	if tracker := l.ensureExecutionTracker(); tracker != nil {
		tracker.finish(runID)
	}
}

func (l *RuntimeLauncher) executionActive(runID uuid.UUID) bool {
	tracker := l.ensureExecutionTracker()
	if tracker == nil {
		return false
	}
	return tracker.active(runID)
}

func (l *RuntimeLauncher) executionRunIDs() []uuid.UUID {
	tracker := l.ensureExecutionTracker()
	if tracker == nil {
		return nil
	}
	return tracker.list()
}

func (l *RuntimeLauncher) beginLaunch(runID uuid.UUID) bool {
	tracker := l.ensureLaunchTracker()
	if tracker == nil {
		return false
	}
	return tracker.begin(runID)
}

func (l *RuntimeLauncher) finishLaunch(runID uuid.UUID) {
	if tracker := l.ensureLaunchTracker(); tracker != nil {
		tracker.finish(runID)
	}
}

func (l *RuntimeLauncher) reconcileRunCompletionSummaries(ctx context.Context) error {
	coordinator := l.ensureCompletionSummaryCoordinator()
	if coordinator == nil {
		return nil
	}
	return coordinator.reconcileRunCompletionSummaries(ctx)
}

func (l *RuntimeLauncher) prepareRunCompletionSummaryBestEffort(ctx context.Context, runID uuid.UUID) {
	if coordinator := l.ensureCompletionSummaryCoordinator(); coordinator != nil {
		coordinator.prepareRunCompletionSummaryBestEffort(ctx, runID)
	}
}

func (l *RuntimeLauncher) scheduleRunCompletionSummary(runID uuid.UUID) {
	if coordinator := l.ensureCompletionSummaryCoordinator(); coordinator != nil {
		coordinator.scheduleRunCompletionSummary(runID)
	}
}

func (l *RuntimeLauncher) prepareTicketWorkspace(
	ctx context.Context,
	runID uuid.UUID,
	launchContext runtimeLaunchContext,
	machine catalogdomain.Machine,
	remote bool,
) (workspaceinfra.Workspace, error) {
	provisioner := l.ensureWorkspaceProvisioner()
	if provisioner == nil {
		return workspaceinfra.Workspace{}, nil
	}
	return provisioner.prepareTicketWorkspace(ctx, runID, launchContext, machine, remote)
}

func (l *RuntimeLauncher) applyGitHubWorkspaceAuth(
	ctx context.Context,
	projectID uuid.UUID,
	request workspaceinfra.SetupRequest,
) (workspaceinfra.SetupRequest, error) {
	provisioner := l.ensureWorkspaceProvisioner()
	if provisioner == nil {
		return request, nil
	}
	return provisioner.applyGitHubWorkspaceAuth(ctx, projectID, request)
}

func (l *RuntimeLauncher) cleanupRunWorkspacesBestEffort(ctx context.Context, runID uuid.UUID, reason string) {
	if provisioner := l.ensureWorkspaceProvisioner(); provisioner != nil {
		provisioner.cleanupRunWorkspacesBestEffort(ctx, runID, reason)
	}
}
