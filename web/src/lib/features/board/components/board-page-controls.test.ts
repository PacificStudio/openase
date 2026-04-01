import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type {
  ActivityPayload,
  AgentPayload,
  Project,
  Ticket,
  TicketPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import { TicketsPage } from '$lib/features/tickets'
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

const agentsFixture: AgentPayload = { agents: [] }
const activityFixture: ActivityPayload = { events: [] }

describe('TicketsPage board controls', () => {
  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    ticketViewStore.setMode('board')
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('shows tickets in their status columns and updates status and priority from the board card controls', async () => {
    appStore.currentProject = projectFixture

    const currentTickets: Ticket[] = cloneValue([
      { ...ticketsFixture.tickets[0], id: 'ticket-1', status_name: 'Todo', priority: 'high' },
      {
        ...ticketsFixture.tickets[0],
        id: 'ticket-2',
        identifier: 'ASE-201',
        title: 'Triage incoming board work',
        status_id: 'status-3',
        status_name: 'Inbox',
        priority: 'low',
      },
      {
        ...ticketsFixture.tickets[0],
        id: 'ticket-3',
        identifier: 'ASE-203',
        title: 'Ship workflow runner polish',
        status_id: 'status-2',
        status_name: 'Doing',
        priority: 'medium',
      },
    ])

    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockImplementation(async () => ({ tickets: cloneValue(currentTickets) }))
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockImplementation(
      async (ticketId: string, patch: { status_id?: string; priority?: string }) => {
        const ticket = currentTickets.find((item) => item.id === ticketId)
        if (!ticket) throw new Error(`unknown ticket ${ticketId}`)
        if (patch.status_id) {
          const status = statusesFixture.statuses.find((item) => item.id === patch.status_id)
          if (!status) throw new Error(`unknown status ${patch.status_id}`)
          ticket.status_id = status.id
          ticket.status_name = status.name
        }
        if (patch.priority) ticket.priority = patch.priority as Ticket['priority']
        return { ticket: cloneValue(ticket) }
      },
    )
    connectEventStream.mockReturnValue(() => {})

    const { findByRole, findByText, getByRole } = render(TicketsPage)

    expect(await findByText('ASE-201')).toBeTruthy()
    expect(
      within(await findByRole('list', { name: 'Inbox tickets' })).getByText('ASE-201'),
    ).toBeTruthy()
    expect(
      within(await findByRole('list', { name: 'Todo tickets' })).getByText('ASE-202'),
    ).toBeTruthy()
    expect(
      within(await findByRole('list', { name: 'Doing tickets' })).getByText('ASE-203'),
    ).toBeTruthy()

    await fireEvent.click(
      within(ticketCardFor(getByRole('list', { name: 'Todo tickets' }), 'ASE-202')).getByLabelText(
        'Change status',
      ),
    )
    await fireEvent.click(await findByRole('button', { name: 'Doing' }))

    await waitFor(() => {
      expect(updateTicket).toHaveBeenNthCalledWith(1, 'ticket-1', { status_id: 'status-2' })
      expect(within(getByRole('list', { name: 'Doing tickets' })).getByText('ASE-202')).toBeTruthy()
      expect(within(getByRole('list', { name: 'Todo tickets' })).queryByText('ASE-202')).toBeNull()
    })

    await fireEvent.click(
      within(ticketCardFor(getByRole('list', { name: 'Doing tickets' }), 'ASE-202')).getByLabelText(
        'Change priority',
      ),
    )
    await fireEvent.click(await findByRole('button', { name: /Urgent/i }))

    await waitFor(() => {
      expect(updateTicket).toHaveBeenNthCalledWith(2, 'ticket-1', { priority: 'urgent' })
      expect(
        within(
          ticketCardFor(getByRole('list', { name: 'Doing tickets' }), 'ASE-202'),
        ).getByLabelText('Priority: urgent'),
      ).toBeTruthy()
    })
  })
})

function cloneValue<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function ticketCardFor(container: HTMLElement, identifier: string) {
  const card = within(container).getByText(identifier).closest('button')
  if (!card) throw new Error(`ticket card not found for ${identifier}`)
  return card as HTMLButtonElement
}
