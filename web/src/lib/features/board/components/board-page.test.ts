import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type {
  ActivityPayload,
  AgentPayload,
  Project,
  StatusPayload,
  TicketPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import { TicketsPage } from '$lib/features/tickets'

const {
  listActivity,
  listAgents,
  listStatuses,
  listTickets,
  listWorkflows,
  updateTicket,
  connectEventStream,
} = vi.hoisted(() => ({
  listActivity: vi.fn(),
  listAgents: vi.fn(),
  listStatuses: vi.fn(),
  listTickets: vi.fn(),
  listWorkflows: vi.fn(),
  updateTicket: vi.fn(),
  connectEventStream: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  listActivity,
  listAgents,
  listStatuses,
  listTickets,
  listWorkflows,
  updateTicket,
}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'OpenASE',
  slug: 'openase',
  description: '',
  status: 'active',
  default_workflow_id: null,
  default_agent_provider_id: null,
  accessible_machine_ids: [],
  max_concurrent_agents: 4,
}

const statusesFixture: StatusPayload = {
  stages: [],
  stage_groups: [],
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
}

const ticketsFixture: TicketPayload = {
  tickets: [
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
      current_run_id: null,
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
  ],
}

const workflowsFixture: WorkflowListPayload = {
  workflows: [
    {
      id: 'workflow-1',
      project_id: 'project-1',
      agent_id: 'agent-1',
      name: 'Coding',
      type: 'coding',
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
  ],
}

const agentsFixture: AgentPayload = {
  agents: [
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
  ],
}

const activityFixture: ActivityPayload = {
  events: [
    {
      id: 'activity-1',
      project_id: 'project-1',
      ticket_id: 'ticket-1',
      agent_id: 'agent-1',
      event_type: 'agent_started',
      message: 'Agent started work.',
      metadata: {
        agent_name: 'Codex Worker',
      },
      created_at: '2026-03-22T09:30:00Z',
    },
  ],
}

describe('TicketsPage', () => {
  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders mapped agent metadata, exposes the agent filter, and switches into list view', async () => {
    appStore.currentProject = projectFixture

    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockResolvedValue(ticketsFixture)
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole, findByText, queryByRole } = render(TicketsPage)

    expect(await findByText('ASE-202')).toBeTruthy()
    expect(await findByText('Codex Worker')).toBeTruthy()
    expect(await findByRole('button', { name: 'Agent' })).toBeTruthy()
    expect(queryByRole('table')).toBeNull()

    await fireEvent.click(await findByRole('button', { name: 'List view' }))

    expect(await findByRole('table')).toBeTruthy()
    expect(await findByText('Updated')).toBeTruthy()
  })
})
