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
  TicketPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import type {
  HookExecution,
  TicketActivity,
  TicketComment,
  TicketDetail,
  TicketReferenceOption,
  TicketRepoOption,
  TicketStatusOption,
} from './types'

export type TicketDetailContext = {
  ticket: TicketDetail
  comments: TicketComment[]
  hooks: HookExecution[]
  activities: TicketActivity[]
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
    comments: detailPayload.comments.map((comment) => ({
      id: comment.id,
      ticketId: comment.ticket_id,
      body: comment.body,
      createdBy: comment.created_by,
      createdAt: comment.created_at,
      updatedAt: comment.updated_at ?? undefined,
    })),
    hooks: detailPayload.hook_history.map((entry) => ({
      id: entry.id,
      hookName: entry.event_type,
      status: inferHookStatus(entry.event_type, entry.message),
      output: entry.message,
      timestamp: entry.created_at,
    })),
    activities: detailPayload.activity.map((entry) => ({
      id: entry.id,
      type: normalizeActivityType(entry.event_type),
      message: entry.message,
      timestamp: entry.created_at,
      agentName: agentNameFromMetadata(entry.metadata),
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
  const normalized = `${eventType} ${message}`.toLowerCase()
  if (normalized.includes('fail') || normalized.includes('error')) return 'fail'
  if (normalized.includes('running') || normalized.includes('start')) return 'running'
  if (normalized.includes('timeout')) return 'timeout'
  return 'pass'
}

function normalizeActivityType(eventType: string) {
  if (eventType === 'status_changed') return 'status_change'
  if (eventType === 'agent_started') return 'started'
  if (eventType === 'agent_completed') return 'completed'
  if (eventType === 'comment_added') return 'comment'
  return eventType
}

function agentNameFromMetadata(metadata: Record<string, unknown>) {
  const value = metadata.agent_name
  return typeof value === 'string' ? value : undefined
}
