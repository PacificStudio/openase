import type { MockState } from './constants'
import {
  DEFAULT_AGENT_ID,
  DEFAULT_PROVIDER_ID,
  DEFAULT_REPO_ID,
  DEFAULT_STATUS_IDS,
  DEFAULT_TICKET_ID,
  DEFAULT_WORKFLOW_ID,
  LOCAL_MACHINE_ID,
  nowIso,
} from './constants'
import { asBoolean, asNumber, asString, clone, findById } from './helpers'

function buildPickupDiagnosis(ticketRecord: Record<string, unknown>) {
  if (ticketRecord.retry_paused) {
    return {
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
      blocked_by: Array.isArray(ticketRecord.dependencies)
        ? ticketRecord.dependencies
            .filter((item) => item.type === 'blocked_by')
            .map((item) => ({
              id: item.target.id,
              identifier: item.target.identifier,
              title: item.target.title,
              status_id: item.target.status_id,
              status_name: item.target.status_name,
            }))
        : [],
    }
  }

  if (ticketRecord.current_run_id) {
    return {
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
  }

  return {
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
}

function buildDefaultTicketRecord(ticket: Record<string, unknown>, ticketId: string) {
  return {
    ...clone(ticket),
    description:
      ticketId === DEFAULT_TICKET_ID
        ? 'Project AI needs the full ticket capsule so the drawer no longer depends on a separate ticket-detail assistant.'
        : (asString(ticket.description) ?? ''),
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
}

export function buildTicketDetailPayload(state: MockState, ticketId: string) {
  const ticket = findById(state.tickets, ticketId)
  if (!ticket) {
    return null
  }

  const repo = state.repos.find((item) => item.id === DEFAULT_REPO_ID)
  const ticketRecord = buildDefaultTicketRecord(ticket, ticketId)

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

  const detailDescription = asString(ticketRecord.description) ?? ''
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
    pickup_diagnosis: buildPickupDiagnosis(ticketRecord),
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
              message: 'go test ./internal/chat failed in internal/chat',
              metadata: {},
              created_at: '2026-04-02T08:15:00.000Z',
            },
          ]
        : [],
  }
}

export function buildTicketRunsPayload(_state: MockState, ticketId: string) {
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
