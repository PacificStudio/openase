import { describe, expect, it } from 'vitest'

import type {
  ProjectRepoPayload,
  StatusPayload,
  TicketDetailPayload,
  TicketPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import { buildTicketDetailContext } from './context'

const detailPayloadFixture: TicketDetailPayload = {
  assigned_agent: {
    id: 'agent-1',
    name: 'todo-app-coding-01',
    provider: 'codex-cloud',
    runtime_control_state: 'active',
    runtime_phase: 'executing',
  },
  ticket: {
    id: 'ticket-1',
    project_id: 'project-1',
    identifier: 'ASE-1',
    title: 'Implement ticket detail agent binding',
    description: '',
    status_id: 'status-1',
    status_name: 'Todo',
    priority: 'high',
    type: 'bugfix',
    workflow_id: 'workflow-1',
    current_run_id: 'run-1',
    target_machine_id: null,
    created_by: 'user:test',
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
    created_at: '2026-03-27T12:00:00Z',
  },
  repo_scopes: [],
  comments: [],
  activity: [],
  hook_history: [],
}

const statusPayloadFixture: StatusPayload = {
  stages: [],
  statuses: [
    {
      id: 'status-1',
      project_id: 'project-1',
      stage_id: null,
      stage: null,
      name: 'Todo',
      color: '#2563eb',
      icon: '',
      is_default: true,
      description: '',
      position: 1,
    },
  ],
  stage_groups: [],
}

const workflowPayloadFixture: WorkflowListPayload = {
  workflows: [
    {
      id: 'workflow-1',
      project_id: 'project-1',
      agent_id: 'agent-1',
      name: 'Todo App Coding Workflow',
      type: 'coding',
      harness_path: '.openase/harnesses/todo.md',
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
  ],
}

const repoPayloadFixture: ProjectRepoPayload = {
  repos: [],
}

const ticketPayloadFixture: TicketPayload = {
  tickets: [
    {
      id: 'ticket-1',
      project_id: 'project-1',
      identifier: 'ASE-1',
      title: 'Implement ticket detail agent binding',
      description: '',
      status_id: 'status-1',
      status_name: 'Todo',
      priority: 'high',
      type: 'bugfix',
      workflow_id: 'workflow-1',
      current_run_id: 'run-1',
      target_machine_id: null,
      created_by: 'user:test',
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
      created_at: '2026-03-27T12:00:00Z',
    },
  ],
}

describe('buildTicketDetailContext', () => {
  it('maps assigned agent details from the explicit ticket detail payload', () => {
    const detail = buildTicketDetailContext(
      detailPayloadFixture,
      statusPayloadFixture,
      workflowPayloadFixture,
      repoPayloadFixture,
      ticketPayloadFixture,
      'ticket-1',
    )

    expect(detail.ticket.assignedAgent).toEqual({
      id: 'agent-1',
      name: 'todo-app-coding-01',
      provider: 'codex-cloud',
      runtimeControlState: 'active',
      runtimePhase: 'executing',
    })
  })
})
