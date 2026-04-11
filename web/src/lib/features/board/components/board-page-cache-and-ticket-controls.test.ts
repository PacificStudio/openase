import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type {
  ActivityPayload,
  AgentPayload,
  StatusPayload,
  Ticket,
  TicketPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import { TicketsPage, resetTicketBoardToolbarStoreForTests } from '$lib/features/tickets'
import { resetProjectBoardCacheForTests } from '$lib/features/tickets'
import { appStore } from '$lib/stores/app.svelte'
import { ticketViewStore } from '$lib/stores/ticket-view.svelte'

const {
  listActivity,
  listAgents,
  listStatuses,
  listTickets,
  listWorkflows,
  toastStore,
  updateStatus,
  updateTicket,
  connectEventStream,
} = vi.hoisted(() => ({
  listActivity: vi.fn(),
  listAgents: vi.fn(),
  listStatuses: vi.fn(),
  listTickets: vi.fn(),
  listWorkflows: vi.fn(),
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
    clear: vi.fn(),
  },
  updateStatus: vi.fn(),
  updateTicket: vi.fn(),
  connectEventStream: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  listActivity,
  listAgents,
  listStatuses,
  listTickets,
  listWorkflows,
  updateStatus,
  updateTicket,
}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))
import { markProjectBoardCacheDirty } from '$lib/features/tickets'
import {
  activityFixture,
  agentsFixture,
  cloneValue,
  createDeferred,
  projectFixture,
  showEmptyColumns,
  statusesFixture,
  ticketCardFor,
  ticketsFixture,
  workflowsFixture,
} from './board-page-controls.test-helpers'

describe('TicketsPage board controls', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    ticketViewStore.setMode('board')
  })

  afterEach(async () => {
    cleanup()
    await vi.runOnlyPendingTimersAsync()
    vi.useRealTimers()
    resetProjectBoardCacheForTests()
    resetTicketBoardToolbarStoreForTests()
    appStore.currentProject = null
    ticketViewStore.setMode('board')
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('reuses the cached board snapshot when remounting the tickets page in the same project', async () => {
    appStore.currentProject = projectFixture

    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockResolvedValue(cloneValue(ticketsFixture))
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)

    const firstRender = render(TicketsPage)
    expect(await firstRender.findByText('ASE-202')).toBeTruthy()

    expect(listStatuses).toHaveBeenCalledTimes(1)
    expect(listTickets).toHaveBeenCalledTimes(1)
    expect(listWorkflows).toHaveBeenCalledTimes(1)
    expect(listAgents).toHaveBeenCalledTimes(1)
    expect(listActivity).toHaveBeenCalledTimes(1)

    firstRender.unmount()

    const secondRender = render(TicketsPage)
    expect(await secondRender.findByText('ASE-202')).toBeTruthy()

    expect(listStatuses).toHaveBeenCalledTimes(1)
    expect(listTickets).toHaveBeenCalledTimes(1)
    expect(listWorkflows).toHaveBeenCalledTimes(1)
    expect(listAgents).toHaveBeenCalledTimes(1)
    expect(listActivity).toHaveBeenCalledTimes(1)
  })

  it('shows the cached board immediately and refreshes in the background once cache is dirty', async () => {
    appStore.currentProject = projectFixture

    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockResolvedValue(cloneValue(ticketsFixture))
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)

    const firstRender = render(TicketsPage)
    expect(await firstRender.findByText('ASE-202')).toBeTruthy()
    firstRender.unmount()

    markProjectBoardCacheDirty(projectFixture.id)

    const deferredStatuses = createDeferred<StatusPayload>()
    const deferredTickets = createDeferred<TicketPayload>()
    const deferredWorkflows = createDeferred<WorkflowListPayload>()
    const deferredAgents = createDeferred<AgentPayload>()
    const deferredActivity = createDeferred<ActivityPayload>()

    listStatuses.mockImplementationOnce(() => deferredStatuses.promise)
    listTickets.mockImplementationOnce(() => deferredTickets.promise)
    listWorkflows.mockImplementationOnce(() => deferredWorkflows.promise)
    listAgents.mockImplementationOnce(() => deferredAgents.promise)
    listActivity.mockImplementationOnce(() => deferredActivity.promise)

    const secondRender = render(TicketsPage)
    expect(await secondRender.findByText('ASE-202')).toBeTruthy()

    expect(listStatuses).toHaveBeenCalledTimes(2)
    expect(listTickets).toHaveBeenCalledTimes(2)
    expect(listWorkflows).toHaveBeenCalledTimes(2)
    expect(listAgents).toHaveBeenCalledTimes(2)
    expect(listActivity).toHaveBeenCalledTimes(2)

    deferredStatuses.resolve(statusesFixture)
    deferredTickets.resolve(cloneValue(ticketsFixture))
    deferredWorkflows.resolve(workflowsFixture)
    deferredAgents.resolve(agentsFixture)
    deferredActivity.resolve(activityFixture)

    await waitFor(() => {
      expect(secondRender.getByText('ASE-202')).toBeTruthy()
    })
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
    await showEmptyColumns(findByRole)

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
