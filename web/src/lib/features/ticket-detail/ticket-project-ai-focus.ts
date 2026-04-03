import type { ProjectAIFocus } from '$lib/features/chat'
import type { HookExecution, TicketDetail, TicketRun, TicketTimelineItem } from './types'

export function buildTicketProjectAIFocus(args: {
  ticket: TicketDetail
  projectId: string
  timeline: TicketTimelineItem[]
  hooks: HookExecution[]
  currentRun: TicketRun | null
  selectedArea: 'detail' | 'comments' | 'runs'
}): ProjectAIFocus | null {
  const { ticket, projectId, timeline, hooks, currentRun, selectedArea } = args
  if (!projectId || !ticket.id) {
    return null
  }

  return {
    kind: 'ticket',
    projectId,
    ticketId: ticket.id,
    ticketIdentifier: ticket.identifier,
    ticketTitle: ticket.title,
    ticketDescription: ticket.description,
    ticketStatus: ticket.status.name,
    ticketPriority: ticket.priority,
    ticketAttemptCount: ticket.attemptCount,
    ticketRetryPaused: ticket.retryPaused,
    ticketPauseReason: ticket.pauseReason,
    ticketDependencies: ticket.dependencies.map((dependency) => ({
      identifier: dependency.identifier,
      title: dependency.title,
      relation: dependency.relation,
      status: dependency.stage,
    })),
    ticketRepoScopes: ticket.repoScopes.map((scope) => ({
      repoId: scope.repoId,
      repoName: scope.repoName,
      branchName: scope.branchName,
      pullRequestUrl: scope.prUrl,
    })),
    ticketRecentActivity: timeline
      .filter((item) => item.kind === 'activity')
      .slice(-12)
      .map((item) => ({
        eventType: item.eventType,
        message: item.bodyText || item.title,
        createdAt: item.createdAt,
      })),
    ticketHookHistory: hooks.slice(-12).map((hook) => ({
      hookName: hook.hookName,
      status: hook.status,
      output: hook.output,
      timestamp: hook.timestamp,
    })),
    ticketAssignedAgent: ticket.assignedAgent
      ? {
          id: ticket.assignedAgent.id,
          name: ticket.assignedAgent.name,
          provider: ticket.assignedAgent.provider,
          runtimeControlState: ticket.assignedAgent.runtimeControlState,
          runtimePhase: ticket.assignedAgent.runtimePhase,
        }
      : undefined,
    ticketCurrentRun: currentRun
      ? {
          id: currentRun.id,
          attemptNumber: currentRun.attemptNumber,
          status: currentRun.status,
          currentStepStatus: currentRun.currentStepStatus,
          currentStepSummary: currentRun.currentStepSummary,
          lastError: currentRun.lastError,
        }
      : undefined,
    ticketTargetMachine: ticket.targetMachineId
      ? {
          id: ticket.targetMachineId,
        }
      : undefined,
    selectedArea,
  }
}
