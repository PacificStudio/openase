import {
  getTicketDetail,
  listProjectRepos,
  listStatuses,
  listTickets,
  listWorkflows,
} from '$lib/api/openase'
import type {
  ProjectRepoPayload,
  StatusPayload,
  TicketDetailPayload,
  TicketTimelineItemRecord,
  TicketPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import type {
  HookExecution,
  TicketDetail,
  TicketTimelineItem,
  TicketReferenceOption,
  TicketRepoOption,
  TicketStatusOption,
} from './types'

export type TicketDetailContext = {
  ticket: TicketDetail
  timeline: TicketTimelineItem[]
  hooks: HookExecution[]
  statuses: TicketStatusOption[]
  dependencyCandidates: TicketReferenceOption[]
  repoOptions: TicketRepoOption[]
}

export async function fetchTicketDetailContext(projectId: string, ticketId: string) {
  const payloads = await Promise.all([
    getTicketDetail(projectId, ticketId),
    listStatuses(projectId),
    listWorkflows(projectId),
    listProjectRepos(projectId),
    listTickets(projectId),
  ])

  return buildTicketDetailContext(...payloads, ticketId)
}

export function buildTicketDetailContext(
  detailPayload: TicketDetailPayload,
  statusPayload: StatusPayload,
  workflowPayload: WorkflowListPayload,
  repoPayload: ProjectRepoPayload,
  ticketPayload: TicketPayload,
  currentTicketId: string,
): TicketDetailContext {
  const statusMap = new Map(statusPayload.statuses.map((status) => [status.id, status]))
  const workflowMap = new Map(workflowPayload.workflows.map((workflow) => [workflow.id, workflow]))
  const detailTicket = detailPayload.ticket
  const status = statusMap.get(detailTicket.status_id)
  const workflow = detailTicket.workflow_id ? workflowMap.get(detailTicket.workflow_id) : null

  return {
    statuses: statusPayload.statuses
      .slice()
      .sort((left, right) => left.position - right.position)
      .map((item) => ({
        id: item.id,
        name: item.name,
        color: item.color || '#94a3b8',
      })),
    dependencyCandidates: ticketPayload.tickets
      .filter((item) => item.id !== currentTicketId)
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
    ticket: {
      id: detailTicket.id,
      identifier: detailTicket.identifier,
      title: detailTicket.title,
      description: detailTicket.description,
      status: {
        id: detailTicket.status_id,
        name: detailTicket.status_name,
        color: status?.color ?? '#94a3b8',
      },
      priority: normalizePriority(detailTicket.priority),
      type: normalizeType(detailTicket.type),
      workflow: workflow
        ? {
            id: workflow.id,
            name: workflow.name,
            type: workflow.type,
          }
        : undefined,
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
        prUrl: scope.pull_request_url ?? undefined,
        prStatus: scope.pr_status ?? undefined,
        ciStatus: scope.ci_status ?? undefined,
        isPrimaryScope: scope.is_primary_scope,
      })),
      attemptCount: detailTicket.attempt_count,
      costAmount: detailTicket.cost_amount,
      budgetUsd: detailTicket.budget_usd,
      dependencies: detailTicket.dependencies.map((dependency) => ({
        id: dependency.id,
        targetId: dependency.target.id,
        identifier: dependency.target.identifier,
        title: dependency.target.title,
        relation: dependency.type,
      })),
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

  return 'medium'
}

function normalizeType(type: string): TicketDetail['type'] {
  if (type === 'feature' || type === 'bugfix' || type === 'refactor' || type === 'chore') {
    return type
  }

  return 'feature'
}

function inferHookStatus(eventType: string, message: string): HookExecution['status'] {
  if (eventType === 'hook.failed') return 'fail'
  if (eventType === 'hook.started') return 'running'
  if (message.toLowerCase().includes('timeout')) return 'timeout'
  return 'pass'
}

function parseTimelineItem(raw: TicketTimelineItemRecord): TicketTimelineItem | null {
  if (!raw.id || !raw.ticket_id || !raw.created_at || !raw.updated_at) {
    return null
  }

  const shared = {
    id: raw.id,
    ticketId: raw.ticket_id,
    actor: {
      name: normalizeActorName(raw.actor_name ?? ''),
      type: raw.actor_type ?? 'unknown',
    },
    createdAt: raw.created_at,
    updatedAt: raw.updated_at,
    editedAt: raw.edited_at ?? undefined,
    isCollapsible: raw.is_collapsible ?? false,
    isDeleted: raw.is_deleted ?? false,
  }

  if (raw.item_type === 'description') {
    return {
      ...shared,
      kind: 'description',
      title: raw.title ?? '',
      bodyMarkdown: raw.body_markdown ?? '',
      identifier: stringMetadata(raw.metadata, 'identifier'),
    }
  }

  if (raw.item_type === 'comment') {
    const commentId = parseTimelineScopedId(raw.id, 'comment:')
    if (!commentId) {
      return null
    }

    return {
      ...shared,
      kind: 'comment',
      commentId,
      bodyMarkdown: raw.body_markdown ?? '',
      editCount: numericMetadata(raw.metadata, 'edit_count') ?? 0,
      revisionCount: numericMetadata(raw.metadata, 'revision_count') ?? 1,
      lastEditedBy: stringMetadata(raw.metadata, 'last_edited_by'),
    }
  }

  if (raw.item_type === 'activity') {
    return {
      ...shared,
      kind: 'activity',
      eventType: stringMetadata(raw.metadata, 'event_type') ?? raw.title ?? '',
      title: raw.title ?? '',
      bodyText: raw.body_text ?? '',
      metadata: cloneMetadata(raw.metadata),
    }
  }

  return null
}

function parseTimelineScopedId(id: string, prefix: string) {
  return id.startsWith(prefix) ? id.slice(prefix.length) : null
}

function stringMetadata(metadata: Record<string, unknown> | undefined, key: string) {
  const value = metadata?.[key]
  return typeof value === 'string' && value.trim() ? value : undefined
}

function numericMetadata(metadata: Record<string, unknown> | undefined, key: string) {
  const value = metadata?.[key]
  return typeof value === 'number' ? value : undefined
}

function cloneMetadata(metadata: Record<string, unknown> | undefined) {
  return metadata ? { ...metadata } : {}
}

function normalizeActorName(value: string) {
  const normalized = value.trim()
  if (!normalized) return 'Unknown'
  return normalized.includes(':') ? (normalized.split(':').at(-1) ?? normalized) : normalized
}
