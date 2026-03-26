import { describe, expect, it } from 'vitest'

import type { ActivityEvent, Agent, Ticket, TicketStatus, Workflow } from '$lib/api/contracts'
import { buildBoardData } from './model'

const statusesFixture: TicketStatus[] = [
  {
    id: 'status-1',
    project_id: 'project-1',
    name: 'Todo',
    color: '#2563eb',
    icon: '',
    is_default: true,
    description: '',
    position: 1,
  },
]

const workflowsFixture: Workflow[] = [
  {
    id: 'workflow-1',
    project_id: 'project-1',
    name: 'Coding',
    type: 'coding',
    harness_path: '.openase/harnesses/coding.md',
    harness_content: null,
    hooks: {},
    required_machine_labels: [],
    max_concurrent: 1,
    max_retry_attempts: 0,
    timeout_minutes: 30,
    stall_timeout_minutes: 10,
    version: 1,
    is_active: true,
    pickup_status_id: 'status-1',
    finish_status_id: null,
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
    workflow_id: 'workflow-1',
    target_machine_id: null,
    created_by: 'codex',
    parent: null,
    children: [],
    dependencies: [],
    external_links: [],
    external_ref: '',
    budget_usd: 0,
    cost_tokens_input: 0,
    cost_tokens_output: 0,
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
    status: 'running',
    current_run_id: null,
    current_ticket_id: 'ticket-1',
    session_id: 'session-1',
    runtime_phase: 'ready',
    runtime_control_state: 'active',
    runtime_started_at: null,
    last_error: '',
    workspace_path: '/tmp/agent-1',
    total_tokens_used: 0,
    total_tickets_completed: 0,
    last_heartbeat_at: null,
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
    expect(board.columns).toHaveLength(1)
    expect(board.columns[0]?.tickets[0]).toMatchObject({
      id: 'ticket-1',
      workflowType: 'coding',
      agentName: 'Codex Worker',
      updatedAt: '2026-03-22T09:30:00Z',
    })
    expect('prCount' in (board.columns[0]?.tickets[0] ?? {})).toBe(false)
    expect('prStatus' in (board.columns[0]?.tickets[0] ?? {})).toBe(false)
  })
})
