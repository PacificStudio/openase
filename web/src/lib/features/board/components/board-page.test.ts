import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type {
  ActivityPayload,
  AgentPayload,
  Project,
  TicketPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import { TicketsPage, resetTicketBoardToolbarStoreForTests } from '$lib/features/tickets'
import { orderedStatusPayloadFixture } from '$lib/features/board/test-fixtures'
import { appStore } from '$lib/stores/app.svelte'
import { ticketViewStore } from '$lib/stores/ticket-view.svelte'

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
  default_agent_provider_id: null,
  accessible_machine_ids: [],
  max_concurrent_agents: 4,
}

const statusesFixture = orderedStatusPayloadFixture

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
    appStore.closeRightPanel()
    ticketViewStore.setMode('board')
    resetTicketBoardToolbarStoreForTests()
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('renders board filters and switches into list view', async () => {
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
    expect(await findByRole('img', { name: 'Blocked' })).toBeTruthy()
    expect(await findByRole('button', { name: 'Agent' })).toBeTruthy()
    expect(queryByRole('table')).toBeNull()

    await fireEvent.click(await findByRole('button', { name: 'List view' }))

    expect(await findByRole('table')).toBeTruthy()
    expect(await findByText('Updated')).toBeTruthy()
    expect(await findByText('Blocked')).toBeTruthy()
  })

  it('wires the column create-ticket action to the new ticket dialog', async () => {
    appStore.currentProject = projectFixture

    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockResolvedValue(ticketsFixture)
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)

    await fireEvent.click(await findByRole('button', { name: 'Create ticket in Todo' }))

    expect(appStore.newTicketDialogOpen).toBe(true)
    expect(appStore.newTicketDefaultStatusId).toBe('status-1')
  })

  it('restores persisted toolbar controls after remounting the tickets page', async () => {
    appStore.currentProject = projectFixture

    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockResolvedValue(ticketsFixture)
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    connectEventStream.mockReturnValue(() => {})

    const firstRender = render(TicketsPage)

    const searchInput = (await firstRender.findByPlaceholderText(
      'Search tickets...',
    )) as HTMLInputElement
    await fireEvent.input(searchInput, { target: { value: 'runtime data' } })
    await fireEvent.click(await firstRender.findByRole('button', { name: 'Hide empty' }))

    expect(await firstRender.findByRole('list', { name: 'Doing tickets' })).toBeTruthy()

    firstRender.unmount()
    resetTicketBoardToolbarStoreForTests()

    const secondRender = render(TicketsPage)
    const restoredSearchInput = (await secondRender.findByPlaceholderText(
      'Search tickets...',
    )) as HTMLInputElement

    expect(restoredSearchInput.value).toBe('runtime data')
    expect(await secondRender.findByRole('list', { name: 'Doing tickets' })).toBeTruthy()
  })
})
