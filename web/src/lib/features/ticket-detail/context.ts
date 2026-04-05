import { getTicketDetail, listProjectRepos, listStatuses, listTickets } from '$lib/api/openase'
import type {
  ProjectRepoPayload,
  StatusPayload,
  TicketDetailPayload,
  TicketPayload,
} from '$lib/api/contracts'
import type {
  HookExecution,
  TicketDetail,
  TicketTimelineItem,
  TicketReferenceOption,
  TicketRepoOption,
  TicketStatusOption,
} from './types'
import { inferHookStatus, parseTimelineItem } from './context-timeline'
import { mapTicketPickupDiagnosis } from './pickup-diagnosis-context'

export type TicketDetailContext = {
  ticket: TicketDetail
  timeline: TicketTimelineItem[]
  hooks: HookExecution[]
  statuses: TicketStatusOption[]
  dependencyCandidates: TicketReferenceOption[]
  repoOptions: TicketRepoOption[]
}

export type TicketDetailLiveContext = Pick<TicketDetailContext, 'ticket' | 'timeline' | 'hooks'>

export type TicketDetailProjectReferenceData = {
  statusLookup: Array<{
    id: string
    stage: string
    color: string
  }>
  statuses: TicketStatusOption[]
  dependencyCandidatesByTicketId: TicketReferenceOption[]
  repoOptions: TicketRepoOption[]
}

export async function fetchTicketDetailContext(projectId: string, ticketId: string) {
  const [detailPayload, referenceData] = await Promise.all([
    getTicketDetail(projectId, ticketId),
    fetchTicketDetailProjectReferenceData(projectId),
  ])

  return buildTicketDetailContext(detailPayload, referenceData, ticketId)
}

export function buildTicketDetailContext(
  detailPayload: TicketDetailPayload,
  referenceData: TicketDetailProjectReferenceData,
  currentTicketId: string,
): TicketDetailContext {
  return {
    ...buildTicketDetailLiveContext(detailPayload, referenceData.statusLookup),
    ...selectTicketDetailReferenceData(referenceData, currentTicketId),
  }
}

export async function fetchTicketDetailProjectReferenceData(projectId: string) {
  const payloads = await Promise.all([
    listStatuses(projectId),
    listProjectRepos(projectId),
    listTickets(projectId),
  ])

  return buildTicketDetailProjectReferenceData(...payloads)
}

export async function fetchTicketDetailLiveContext(
  projectId: string,
  ticketId: string,
  referenceData: TicketDetailProjectReferenceData,
) {
  const detailPayload = await getTicketDetail(projectId, ticketId)
  return buildTicketDetailLiveContext(detailPayload, referenceData.statusLookup)
}

export function buildTicketDetailProjectReferenceData(
  statusPayload: StatusPayload,
  repoPayload: ProjectRepoPayload,
  ticketPayload: TicketPayload,
): TicketDetailProjectReferenceData {
  return {
    statusLookup: statusPayload.statuses.map((item) => ({
      id: item.id,
      stage: item.stage,
      color: item.color || '#94a3b8',
    })),
    statuses: statusPayload.statuses
      .slice()
      .sort((left, right) => left.position - right.position)
      .map((item) => ({
        id: item.id,
        name: item.name,
        color: item.color || '#94a3b8',
        stage: item.stage,
      })),
    dependencyCandidatesByTicketId: ticketPayload.tickets
      .map((item) => ({
        id: item.id,
        identifier: item.identifier,
        title: item.title,
      }))
      .sort((left, right) => left.identifier.localeCompare(right.identifier)),
    repoOptions: repoPayload.repos
      .map((item) => ({
        id: item.id,
        name: item.name,
        defaultBranch: item.default_branch,
      }))
      .sort((left, right) => left.name.localeCompare(right.name)),
  }
}

export function selectTicketDetailReferenceData(
  referenceData: TicketDetailProjectReferenceData,
  currentTicketId: string,
) {
  return {
    statuses: referenceData.statuses,
    dependencyCandidates: referenceData.dependencyCandidatesByTicketId.filter(
      (item) => item.id !== currentTicketId,
    ),
    repoOptions: referenceData.repoOptions,
  }
}

export function buildTicketDetailLiveContext(
  detailPayload: TicketDetailPayload,
  referenceStatuses: TicketDetailProjectReferenceData['statusLookup'],
): TicketDetailLiveContext {
  const statusMap = new Map(referenceStatuses.map((status) => [status.id, status]))
  const detailTicket = detailPayload.ticket
  const status = statusMap.get(detailTicket.status_id)

  return {
    ticket: {
      id: detailTicket.id,
      identifier: detailTicket.identifier,
      title: detailTicket.title,
      description: detailTicket.description,
      archived: detailTicket.archived,
      status: {
        id: detailTicket.status_id,
        name: detailTicket.status_name,
        color: status?.color ?? '#94a3b8',
      },
      priority: normalizePriority(detailTicket.priority),
      type: normalizeType(detailTicket.type),
      assignedAgent: detailPayload.assigned_agent
        ? {
            id: detailPayload.assigned_agent.id,
            name: detailPayload.assigned_agent.name,
            provider: detailPayload.assigned_agent.provider,
            runtimeControlState: detailPayload.assigned_agent.runtime_control_state,
            runtimePhase: detailPayload.assigned_agent.runtime_phase ?? undefined,
          }
        : undefined,
      repoScopes: detailPayload.repo_scopes.map((scope) => ({
        id: scope.id,
        repoId: scope.repo_id,
        repoName: scope.repo?.name ?? 'Detached repository',
        branchName: scope.branch_name,
        defaultBranch: scope.default_branch,
        effectiveBranchName: scope.effective_branch_name,
        branchSource: scope.branch_source === 'override' ? 'override' : 'generated',
        prUrl: scope.pull_request_url ?? undefined,
      })),
      attemptCount: detailTicket.attempt_count,
      consecutiveErrors: detailTicket.consecutive_errors,
      retryPaused: detailTicket.retry_paused,
      pauseReason: detailTicket.pause_reason || undefined,
      currentRunId: detailTicket.current_run_id ?? undefined,
      targetMachineId: detailTicket.target_machine_id ?? undefined,
      nextRetryAt: detailTicket.next_retry_at ?? undefined,
      costTokensInput: detailTicket.cost_tokens_input,
      costTokensOutput: detailTicket.cost_tokens_output,
      costTokensTotal:
        detailTicket.cost_tokens_total ??
        (detailTicket.cost_tokens_input ?? 0) + (detailTicket.cost_tokens_output ?? 0),
      costAmount: detailTicket.cost_amount,
      budgetUsd: detailTicket.budget_usd,
      pickupDiagnosis: mapTicketPickupDiagnosis(detailPayload.pickup_diagnosis),
      dependencies: detailTicket.dependencies.map((dependency) => {
        const targetStatus = statusMap.get(dependency.target.status_id)!
        const stage = targetStatus.stage as TicketDetail['dependencies'][number]['stage']
        return {
          id: dependency.id,
          targetId: dependency.target.id,
          identifier: dependency.target.identifier,
          title: dependency.target.title,
          relation: normalizeDependencyRelation(dependency.type),
          stage,
        }
      }),
      externalLinks: detailTicket.external_links.map((link) => ({
        id: link.id,
        type: link.type,
        url: link.url,
        externalId: link.external_id,
        title: link.title ?? undefined,
        status: link.status ?? undefined,
        relation: link.relation,
      })),
      children: detailTicket.children.map((child) => ({
        id: child.id,
        identifier: child.identifier,
        title: child.title,
        status: child.status_name,
      })),
      createdBy: detailTicket.created_by,
      createdAt: detailTicket.created_at,
      updatedAt: detailTicket.created_at,
      startedAt: detailTicket.started_at ?? undefined,
      completedAt: detailTicket.completed_at ?? undefined,
    },
    timeline: detailPayload.timeline
      .map(parseTimelineItem)
      .filter((item): item is TicketTimelineItem => item !== null),
    hooks: detailPayload.hook_history.map((entry) => ({
      id: entry.id,
      hookName: entry.event_type,
      status: inferHookStatus(entry.event_type, entry.message),
      output: entry.message,
      timestamp: entry.created_at,
    })),
  }
}

export function frameReferencesTicket(frameData: string, currentTicketId: string) {
  try {
    const envelope = JSON.parse(frameData) as {
      payload?: {
        ticket?: { id?: string }
        ticket_id?: string
        event?: { ticket_id?: string }
      }
    }

    return (
      envelope.payload?.ticket?.id === currentTicketId ||
      envelope.payload?.ticket_id === currentTicketId ||
      envelope.payload?.event?.ticket_id === currentTicketId
    )
  } catch {
    return false
  }
}

function normalizePriority(priority: string): TicketDetail['priority'] {
  if (priority === 'urgent' || priority === 'high' || priority === 'medium' || priority === 'low') {
    return priority
  }

  return ''
}

function normalizeType(type: string): TicketDetail['type'] {
  if (type === 'feature' || type === 'bugfix' || type === 'refactor' || type === 'chore') {
    return type
  }

  return 'feature'
}

function normalizeDependencyRelation(
  relation: string,
): TicketDetail['dependencies'][number]['relation'] {
  if (relation === 'blocks' || relation === 'blocked_by' || relation === 'sub_issue') {
    return relation
  }

  return 'blocks'
}
