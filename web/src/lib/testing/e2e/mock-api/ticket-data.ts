import type { MockState } from './constants'
import {
  DEFAULT_AGENT_ID,
  DEFAULT_PROVIDER_ID,
  DEFAULT_REPO_ID,
  DEFAULT_STATUS_IDS,
  DEFAULT_TICKET_ID,
  DEFAULT_WORKFLOW_ID,
  LOCAL_MACHINE_ID,
  PROJECT_ID,
  nowIso,
} from './constants'
import { asBoolean, asNumber, asObject, asString, clone, findById } from './helpers'

export function createMockTicketRecord(input: {
  id: string
  identifier: string
  title: string
  description?: string
  statusId: string
  statusName: string
  workflowId: string
  createdAt?: string
}) {
  return {
    id: input.id,
    project_id: PROJECT_ID,
    identifier: input.identifier,
    title: input.title,
    description: input.description ?? '',
    status_id: input.statusId,
    status_name: input.statusName,
    priority: 'medium',
    type: 'feature',
    workflow_id: input.workflowId,
    current_run_id: null,
    target_machine_id: null,
    created_by: 'playwright',
    parent: null,
    children: [],
    dependencies: [],
    external_links: [],
    pull_request_urls: [],
    external_ref: '',
    budget_usd: 0,
    cost_tokens_input: 0,
    cost_tokens_output: 0,
    cost_tokens_total: 0,
    cost_amount: 0,
    attempt_count: 0,
    consecutive_errors: 0,
    started_at: null,
    completed_at: null,
    next_retry_at: null,
    retry_paused: false,
    pause_reason: '',
    created_at: input.createdAt ?? nowIso,
  }
}

export function createProjectUpdateThreadRecord(
  state: MockState,
  projectId: string,
  input: Record<string, unknown>,
) {
  const createdAt = nowIso
  const body = asString(input.body)?.trim() ?? ''
  const status = parseProjectUpdateStatus(asString(input.status))
  const title = asString(input.title)?.trim() || summarizeProjectUpdateTitle(body)

  return {
    id: `update-thread-${++state.counters.projectUpdateThread}`,
    project_id: projectId,
    status,
    title,
    body_markdown: body,
    created_by: asString(input.created_by) ?? 'playwright',
    created_at: createdAt,
    updated_at: createdAt,
    edited_at: null,
    edit_count: 0,
    last_edited_by: null,
    is_deleted: false,
    deleted_at: null,
    deleted_by: null,
    last_activity_at: createdAt,
    comment_count: 0,
    comments: [],
  }
}

export function updateProjectUpdateThreadRecord(
  thread: Record<string, unknown>,
  input: Record<string, unknown>,
) {
  const updatedAt = nowIso
  thread.status = parseProjectUpdateStatus(asString(input.status))
  thread.title =
    asString(input.title)?.trim() || summarizeProjectUpdateTitle(asString(input.body) ?? '')
  thread.body_markdown = asString(input.body)?.trim() ?? ''
  thread.updated_at = updatedAt
  thread.edited_at = updatedAt
  thread.edit_count = (asNumber(thread.edit_count) ?? 0) + 1
  thread.last_edited_by = asString(input.edited_by) ?? 'playwright'
  thread.last_activity_at = updatedAt
}

export function deleteProjectUpdateThreadRecord(thread: Record<string, unknown>) {
  const deletedAt = nowIso
  thread.is_deleted = true
  thread.deleted_at = deletedAt
  thread.deleted_by = 'playwright'
  thread.updated_at = deletedAt
  thread.last_activity_at = deletedAt
}

export function createProjectUpdateCommentRecord(
  state: MockState,
  threadId: string,
  input: Record<string, unknown>,
) {
  const createdAt = nowIso
  return {
    id: `update-comment-${++state.counters.projectUpdateComment}`,
    thread_id: threadId,
    body_markdown: asString(input.body)?.trim() ?? '',
    created_by: asString(input.created_by) ?? 'playwright',
    created_at: createdAt,
    updated_at: createdAt,
    edited_at: null,
    edit_count: 0,
    last_edited_by: null,
    is_deleted: false,
    deleted_at: null,
    deleted_by: null,
  }
}

export function updateProjectUpdateCommentRecord(
  comment: Record<string, unknown>,
  input: Record<string, unknown>,
) {
  const updatedAt = nowIso
  comment.body_markdown = asString(input.body)?.trim() ?? ''
  comment.updated_at = updatedAt
  comment.edited_at = updatedAt
  comment.edit_count = (asNumber(comment.edit_count) ?? 0) + 1
  comment.last_edited_by = asString(input.edited_by) ?? 'playwright'
}

export function deleteProjectUpdateCommentRecord(comment: Record<string, unknown>) {
  const deletedAt = nowIso
  comment.is_deleted = true
  comment.deleted_at = deletedAt
  comment.deleted_by = 'playwright'
  comment.updated_at = deletedAt
}

export function readProjectUpdateComments(thread: Record<string, unknown>) {
  return Array.isArray(thread.comments) ? [...thread.comments] : []
}

export function summarizeProjectUpdateTitle(body: string) {
  const firstLine = body.split('\n')[0]?.trim() ?? ''
  if (firstLine.length > 0) {
    return firstLine.slice(0, 72)
  }
  return 'Update'
}

export function parseProjectUpdateStatus(raw: string | null | undefined) {
  return raw === 'at_risk' || raw === 'off_track' ? raw : 'on_track'
}

export function buildTicketDetailPayload(state: MockState, ticketId: string) {
  const ticket = findById(state.tickets, ticketId)
  if (!ticket) {
    return null
  }

  const repo = state.repos.find((item) => item.id === DEFAULT_REPO_ID)
  const detailDescription =
    ticketId === DEFAULT_TICKET_ID
      ? 'Project AI needs the full ticket capsule so the drawer no longer depends on a separate ticket-detail assistant.'
      : (asString(ticket.description) ?? '')

  const ticketRecord = {
    ...clone(ticket),
    description: detailDescription,
    priority: ticketId === DEFAULT_TICKET_ID ? 'high' : (asString(ticket.priority) ?? 'medium'),
    current_run_id: ticketId === DEFAULT_TICKET_ID ? 'run-1' : null,
    attempt_count: ticketId === DEFAULT_TICKET_ID ? 3 : (asNumber(ticket.attempt_count) ?? 0),
    consecutive_errors:
      ticketId === DEFAULT_TICKET_ID ? 2 : (asNumber(ticket.consecutive_errors) ?? 0),
    retry_paused: ticketId === DEFAULT_TICKET_ID ? true : (asBoolean(ticket.retry_paused) ?? false),
    pause_reason:
      ticketId === DEFAULT_TICKET_ID
        ? 'Repeated hook failures'
        : (asString(ticket.pause_reason) ?? ''),
    next_retry_at: ticketId === DEFAULT_TICKET_ID ? null : (asString(ticket.next_retry_at) ?? null),
    dependencies:
      ticketId === DEFAULT_TICKET_ID
        ? [
            {
              id: 'dependency-1',
              type: 'blocked_by',
              target: {
                id: 'ticket-blocker',
                identifier: 'ASE-77',
                title: 'Stabilize project conversation restore',
                status_id: DEFAULT_STATUS_IDS.review,
                status_name: 'In Review',
              },
            },
          ]
        : [],
    children: [],
  }

  const pickupDiagnosis = ticketRecord.retry_paused
    ? {
        state: 'blocked',
        primary_reason_code: 'retry_paused_repeated_stalls',
        primary_reason_message: 'Retries are paused after repeated stalls.',
        next_action_hint: 'Review the last failed attempt, then continue retry when ready.',
        reasons: [
          {
            code: 'retry_paused_repeated_stalls',
            message: 'Manual retry is required after repeated stalls.',
            severity: 'warning',
          },
        ],
        workflow: {
          id: DEFAULT_WORKFLOW_ID,
          name: 'Coding Workflow',
          is_active: true,
          pickup_status_match: true,
        },
        agent: {
          id: DEFAULT_AGENT_ID,
          name: 'coding-main',
          runtime_control_state: 'active',
        },
        provider: {
          id: DEFAULT_PROVIDER_ID,
          name: 'codex-app-server',
          machine_id: LOCAL_MACHINE_ID,
          machine_name: 'local-dev',
          machine_status: 'online',
          availability_state: 'available',
          availability_reason: null,
        },
        retry: {
          attempt_count: ticketRecord.attempt_count,
          retry_paused: ticketRecord.retry_paused,
          pause_reason: 'repeated_stalls',
          next_retry_at: null,
        },
        capacity: {
          workflow: { limited: false, active_runs: 0, capacity: 0 },
          project: { limited: false, active_runs: 0, capacity: 0 },
          provider: { limited: false, active_runs: 0, capacity: 0 },
          status: { limited: false, active_runs: 0, capacity: null },
        },
        blocked_by: ticketRecord.dependencies
          .filter((item) => item.type === 'blocked_by')
          .map((item) => ({
            id: item.target.id,
            identifier: item.target.identifier,
            title: item.target.title,
            status_id: item.target.status_id,
            status_name: item.target.status_name,
          })),
      }
    : ticketRecord.current_run_id
      ? {
          state: 'running',
          primary_reason_code: 'running_current_run',
          primary_reason_message: 'Ticket already has an active run.',
          next_action_hint: 'Wait for the current run to finish or inspect the active runtime.',
          reasons: [
            {
              code: 'running_current_run',
              message: 'Current run is still attached to the ticket.',
              severity: 'info',
            },
          ],
          workflow: {
            id: DEFAULT_WORKFLOW_ID,
            name: 'Coding Workflow',
            is_active: true,
            pickup_status_match: true,
          },
          agent: {
            id: DEFAULT_AGENT_ID,
            name: 'coding-main',
            runtime_control_state: 'active',
          },
          provider: {
            id: DEFAULT_PROVIDER_ID,
            name: 'codex-app-server',
            machine_id: LOCAL_MACHINE_ID,
            machine_name: 'local-dev',
            machine_status: 'online',
            availability_state: 'available',
            availability_reason: null,
          },
          retry: {
            attempt_count: ticketRecord.attempt_count,
            retry_paused: false,
            pause_reason: '',
            next_retry_at: ticketRecord.next_retry_at,
          },
          capacity: {
            workflow: { limited: false, active_runs: 1, capacity: 0 },
            project: { limited: false, active_runs: 1, capacity: 0 },
            provider: { limited: false, active_runs: 1, capacity: 0 },
            status: { limited: false, active_runs: 1, capacity: null },
          },
          blocked_by: [],
        }
      : {
          state: 'runnable',
          primary_reason_code: 'ready_for_pickup',
          primary_reason_message: 'Ticket is ready for pickup.',
          next_action_hint: 'Wait for the scheduler to claim the ticket.',
          reasons: [
            {
              code: 'ready_for_pickup',
              message: 'The scheduler can claim this ticket on the next tick.',
              severity: 'info',
            },
          ],
          workflow: {
            id: DEFAULT_WORKFLOW_ID,
            name: 'Coding Workflow',
            is_active: true,
            pickup_status_match: true,
          },
          agent: {
            id: DEFAULT_AGENT_ID,
            name: 'coding-main',
            runtime_control_state: 'active',
          },
          provider: {
            id: DEFAULT_PROVIDER_ID,
            name: 'codex-app-server',
            machine_id: LOCAL_MACHINE_ID,
            machine_name: 'local-dev',
            machine_status: 'online',
            availability_state: 'available',
            availability_reason: null,
          },
          retry: {
            attempt_count: ticketRecord.attempt_count,
            retry_paused: false,
            pause_reason: '',
            next_retry_at: ticketRecord.next_retry_at,
          },
          capacity: {
            workflow: { limited: false, active_runs: 0, capacity: 0 },
            project: { limited: false, active_runs: 0, capacity: 0 },
            provider: { limited: false, active_runs: 0, capacity: 0 },
            status: { limited: false, active_runs: 0, capacity: null },
          },
          blocked_by: [],
        }

  const repoScopes =
    ticketId === DEFAULT_TICKET_ID && repo
      ? [
          {
            id: 'repo-scope-1',
            ticket_id: ticketId,
            repo_id: DEFAULT_REPO_ID,
            repo,
            branch_name: 'feat/openase-470-project-ai',
            pull_request_url: 'https://github.com/PacificStudio/openase/pull/999',
            created_at: nowIso,
          },
        ]
      : []

  const timeline = [
    {
      id: `description:${ticketId}`,
      ticket_id: ticketId,
      item_type: 'description',
      actor_name: 'playwright',
      actor_type: 'user',
      title: asString(ticket.title),
      body_markdown: detailDescription,
      body_text: null,
      created_at: asString(ticket.created_at) ?? nowIso,
      updated_at: asString(ticket.created_at) ?? nowIso,
      edited_at: null,
      is_collapsible: false,
      is_deleted: false,
      metadata: {
        identifier: asString(ticket.identifier),
      },
    },
    {
      id: `activity:${ticketId}:retry-paused`,
      ticket_id: ticketId,
      item_type: 'activity',
      actor_name: 'orchestrator',
      actor_type: 'system',
      title: 'ticket.retry_paused',
      body_markdown: null,
      body_text: 'Paused retries after repeated hook failures.',
      created_at: '2026-04-02T08:10:00.000Z',
      updated_at: '2026-04-02T08:10:00.000Z',
      edited_at: null,
      is_collapsible: true,
      is_deleted: false,
      metadata: {
        event_type: 'ticket.retry_paused',
      },
    },
  ]

  return {
    assigned_agent:
      ticketId === DEFAULT_TICKET_ID
        ? {
            id: DEFAULT_AGENT_ID,
            name: 'coding-main',
            provider: 'codex-app-server',
            runtime_control_state: 'active',
            runtime_phase: 'executing',
          }
        : null,
    pickup_diagnosis: pickupDiagnosis,
    ticket: ticketRecord,
    repo_scopes: repoScopes,
    comments: [],
    timeline,
    activity: [],
    hook_history:
      ticketId === DEFAULT_TICKET_ID
        ? [
            {
              id: 'hook-history-1',
              ticket_id: ticketId,
              event_type: 'ticket.on_complete',
              message: 'go test ./... failed in internal/chat',
              metadata: {},
              created_at: '2026-04-02T08:15:00.000Z',
            },
          ]
        : [],
  }
}

export function buildTicketRunsPayload(state: MockState, ticketId: string) {
  if (ticketId !== DEFAULT_TICKET_ID) {
    return { runs: [] }
  }
  return {
    runs: [
      {
        id: 'run-1',
        ticket_id: ticketId,
        attempt_number: 3,
        agent_id: DEFAULT_AGENT_ID,
        agent_name: 'coding-main',
        provider: 'codex-app-server',
        adapter_type: 'codex-app-server',
        model_name: 'gpt-5.4',
        usage: {
          total: 1540,
          input: 1200,
          output: 340,
          cached_input: 120,
          cache_creation: 45,
          reasoning: 80,
          prompt: 920,
          candidate: 260,
          tool: 30,
        },
        status: 'failed',
        current_step_status: 'failed',
        current_step_summary: 'openase test ./internal/chat',
        created_at: '2026-04-02T08:00:00.000Z',
        runtime_started_at: '2026-04-02T08:01:00.000Z',
        last_heartbeat_at: '2026-04-02T08:14:00.000Z',
        completed_at: '2026-04-02T08:15:00.000Z',
        last_error: 'ticket.on_complete hook failed',
      },
    ],
  }
}

export function buildTicketRunDetailPayload(state: MockState, ticketId: string, runId: string) {
  if (ticketId !== DEFAULT_TICKET_ID || runId !== 'run-1') {
    return null
  }
  return {
    run: buildTicketRunsPayload(state, ticketId).runs[0],
    trace_entries: [],
    step_entries: [
      {
        id: 'step-1',
        agent_run_id: 'run-1',
        step_status: 'failed',
        summary: 'openase test ./internal/chat',
        source_trace_event_id: null,
        created_at: '2026-04-02T08:15:00.000Z',
      },
    ],
  }
}

export function buildMockProjectConversationReply(
  message: string,
  ticketFocus: {
    identifier: string
    status: string
    retryPaused: boolean
    pauseReason: string
    repoScopes: unknown[]
    hookHistory: unknown[]
    currentRun: Record<string, unknown> | null
  } | null,
) {
  if (!ticketFocus) {
    return `Mock assistant reply for: ${message}`
  }

  const normalized = message.toLowerCase()
  if (normalized.includes('why is this ticket not running')) {
    const currentRunStatus = asString(ticketFocus.currentRun?.status) ?? 'unknown'
    const lastError = asString(ticketFocus.currentRun?.last_error) ?? 'unknown'
    return `${ticketFocus.identifier} is currently ${ticketFocus.status}. Retries are paused=${ticketFocus.retryPaused} because "${ticketFocus.pauseReason}". The latest run status was ${currentRunStatus} and the latest failure was "${lastError}".`
  }

  if (normalized.includes('which repos does this ticket currently affect')) {
    const scopes = ticketFocus.repoScopes
      .map((scope) => asObject(scope))
      .filter((scope): scope is Record<string, unknown> => scope !== null)
      .map((scope) =>
        [
          asString(scope.repo_name) ?? asString(scope.repo_id) ?? 'unknown-repo',
          asString(scope.branch_name),
        ]
          .filter(Boolean)
          .join(' @ '),
      )
      .filter(Boolean)
    return `${ticketFocus.identifier} currently affects ${scopes.join(', ')}.`
  }

  if (normalized.includes('what hook failed most recently')) {
    const latestHook = ticketFocus.hookHistory
      .map((hook) => asObject(hook))
      .filter((hook): hook is Record<string, unknown> => hook !== null)
      .at(-1)
    return latestHook
      ? `The latest hook was ${asString(latestHook.hook_name) ?? 'unknown'} and it reported "${asString(latestHook.output) ?? ''}".`
      : `No hook history is available for ${ticketFocus.identifier}.`
  }

  return `Mock assistant reply for ${ticketFocus.identifier}: ${message}`
}

export function buildMockProjectConversationWorkspaceDiff(conversationId: string) {
  return {
    conversation_id: conversationId,
    workspace_path: `/tmp/${conversationId}`,
    preparing: false,
    dirty: false,
    repos_changed: 0,
    files_changed: 0,
    added: 0,
    removed: 0,
    repos: [],
  }
}
