import { describe, expect, it } from 'vitest'

import type { ActivityEvent, Agent, Ticket, Workflow } from '$lib/api/contracts'
import { buildBoardData } from './model'
import { orderedStatusPayloadFixture } from './test-fixtures'

const statusesFixture = orderedStatusPayloadFixture

const workflowsFixture: Workflow[] = [
  {
    id: 'workflow-1',
    project_id: 'project-1',
    agent_id: 'agent-1',
    name: 'Coding',
    type: 'coding',
    workflow_family: 'coding',
    workflow_classification: {
      family: 'coding',
      confidence: 1,
      reasons: ['fixture'],
    },
    harness_path: '.openase/harnesses/coding.md',
    harness_content: null,
    hooks: {},
    max_concurrent: 1,
    max_retry_attempts: 0,
    timeout_minutes: 30,
    stall_timeout_minutes: 10,
    version: 1,
    is_active: true,
    pickup_status_ids: ['status-1'],
    finish_status_ids: ['status-2'],
  },
]

const ticketsFixture: Ticket[] = [
  {
    id: 'ticket-1',
    project_id: 'project-1',
    identifier: 'ASE-202',
    title: 'Wire board page to runtime data',
    description: '',
    status_id: 'status-1',
    status_name: 'Todo',
    priority: 'high',
    type: 'feature',
    archived: false,
    workflow_id: 'workflow-1',
    current_run_id: null,
    target_machine_id: null,
    created_by: 'codex',
    parent: null,
    children: [],
    dependencies: [
      {
        id: 'dep-1',
        type: 'blocked_by',
        target: {
          id: 'ticket-9',
          identifier: 'ASE-201',
          title: 'Unblock infra',
          status_id: 'status-2',
          status_name: 'Doing',
        },
      },
    ],
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
    created_at: '2026-03-21T12:00:00Z',
  },
]

const agentsFixture: Agent[] = [
  {
    id: 'agent-1',
    provider_id: 'provider-1',
    project_id: 'project-1',
    name: 'Codex Worker',
    runtime_control_state: 'active',
    runtime: {
      active_run_count: 1,
      status: 'running',
      current_run_id: null,
      current_ticket_id: 'ticket-1',
      session_id: 'session-1',
      runtime_phase: 'ready',
      runtime_started_at: null,
      last_error: '',
      last_heartbeat_at: null,
      current_step_status: null,
      current_step_summary: null,
      current_step_changed_at: null,
    },
    total_tokens_used: 0,
    total_tickets_completed: 0,
  },
]

const activityFixture: ActivityEvent[] = [
  {
    id: 'activity-1',
    project_id: 'project-1',
    ticket_id: 'ticket-1',
    agent_id: 'agent-1',
    event_type: 'agent_started',
    message: 'Agent started work.',
    metadata: {},
    created_at: '2026-03-22T09:30:00Z',
  },
]

describe('board model', () => {
  it('maps runtime agent names and exposes matching agent filter options', () => {
    const board = buildBoardData(
      statusesFixture,
      ticketsFixture,
      workflowsFixture,
      agentsFixture,
      activityFixture,
    )

    expect(board.workflowTypes).toEqual(['coding'])
    expect(board.agentOptions).toEqual(['Codex Worker'])
    expect(board.groups.map((group) => group.name)).toEqual(['Board'])
    expect(board.columns.map((column) => column.name)).toEqual(['Inbox', 'Todo', 'Doing'])
    expect(board.columns[2]?.wipInfo).toBe('1 / 1 active')
    expect(board.columns[1]?.tickets[0]).toMatchObject({
      id: 'ticket-1',
      workflowType: 'coding',
      agentName: 'Codex Worker',
      updatedAt: '2026-03-22T09:30:00Z',
      isBlocked: true,
    })
    expect('prCount' in (board.columns[1]?.tickets[0] ?? {})).toBe(false)
    expect('prStatus' in (board.columns[1]?.tickets[0] ?? {})).toBe(false)
  })

  it('preserves tickets with no explicit priority instead of defaulting them to medium', () => {
    const board = buildBoardData(
      statusesFixture,
      [{ ...ticketsFixture[0], id: 'ticket-2', priority: '' }],
      workflowsFixture,
      agentsFixture,
      activityFixture,
    )

    expect(board.columns[1]?.tickets[0]?.priority).toBe('')
  })

  it('orders board statuses by stage first and position within the stage', () => {
    const board = buildBoardData(
      {
        statuses: [
          {
            id: 'done',
            project_id: 'project-1',
            name: 'Done',
            stage: 'completed',
            color: '#10b981',
            icon: '',
            is_default: false,
            description: '',
            position: 0,
            active_runs: 0,
            max_active_runs: null,
          },
          {
            id: 'review',
            project_id: 'project-1',
            name: 'Review',
            stage: 'started',
            color: '#f59e0b',
            icon: '',
            is_default: false,
            description: '',
            position: 9,
            active_runs: 0,
            max_active_runs: null,
          },
          {
            id: 'backlog',
            project_id: 'project-1',
            name: 'Backlog',
            stage: 'backlog',
            color: '#64748b',
            icon: '',
            is_default: false,
            description: '',
            position: 5,
            active_runs: 0,
            max_active_runs: null,
          },
          {
            id: 'todo',
            project_id: 'project-1',
            name: 'Todo',
            stage: 'unstarted',
            color: '#2563eb',
            icon: '',
            is_default: true,
            description: '',
            position: 7,
            active_runs: 0,
            max_active_runs: null,
          },
          {
            id: 'doing',
            project_id: 'project-1',
            name: 'Doing',
            stage: 'started',
            color: '#f97316',
            icon: '',
            is_default: false,
            description: '',
            position: 2,
            active_runs: 0,
            max_active_runs: null,
          },
          {
            id: 'canceled',
            project_id: 'project-1',
            name: 'Canceled',
            stage: 'canceled',
            color: '#ef4444',
            icon: '',
            is_default: false,
            description: '',
            position: 1,
            active_runs: 0,
            max_active_runs: null,
          },
        ],
      },
      [],
      [],
      [],
      [],
    )

    expect(board.columns.map((column) => column.name)).toEqual([
      'Backlog',
      'Todo',
      'Doing',
      'Review',
      'Done',
      'Canceled',
    ])
    expect(board.statusOptions.map((status) => status.name)).toEqual([
      'Backlog',
      'Todo',
      'Doing',
      'Review',
      'Done',
      'Canceled',
    ])
  })
})
