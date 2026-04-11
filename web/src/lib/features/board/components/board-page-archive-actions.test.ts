import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { StatusPayload, Ticket } from '$lib/api/contracts'
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
import {
  activityFixture,
  agentsFixture,
  cloneValue,
  openColumnActionMenu,
  projectFixture,
  showEmptyColumns,
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

  it('archives all non-archived tickets in a column via the archived flag', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses: StatusPayload = cloneValue({
      statuses: [
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted' as const,
          color: '#2563eb',
          icon: '',
          is_default: true,
          description: '',
          position: 1,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-doing',
          project_id: 'project-1',
          name: 'Doing',
          stage: 'started' as const,
          color: '#f59e0b',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-cancelled',
          project_id: 'project-1',
          name: 'Cancelled',
          stage: 'canceled' as const,
          color: '#4b5563',
          icon: '',
          is_default: false,
          description: '',
          position: 3,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    const currentTickets: Ticket[] = cloneValue([
      {
        ...ticketsFixture.tickets[0],
        id: 'ticket-1',
        identifier: 'ASE-202',
        title: 'Wire board page to runtime data',
        status_id: 'status-todo',
        status_name: 'Todo',
        archived: false,
      },
      {
        ...ticketsFixture.tickets[0],
        id: 'ticket-2',
        identifier: 'ASE-203',
        title: 'Ship workflow runner polish',
        status_id: 'status-todo',
        status_name: 'Todo',
        archived: false,
      },
      {
        ...ticketsFixture.tickets[0],
        id: 'ticket-3',
        identifier: 'ASE-204',
        title: 'Keep existing archived ticket',
        status_id: 'status-cancelled',
        status_name: 'Cancelled',
        archived: true,
      },
    ])

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockImplementation(async () => ({ tickets: cloneValue(currentTickets) }))
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateStatus.mockResolvedValue({ status: currentStatuses.statuses[0] })
    updateTicket.mockImplementation(async (ticketId: string, patch: { archived?: boolean }) => {
      const ticket = currentTickets.find((item) => item.id === ticketId)
      if (!ticket || typeof patch.archived !== 'boolean') {
        throw new Error(`unknown ticket update ${ticketId}`)
      }
      ticket.archived = patch.archived
      return { ticket: cloneValue(ticket) }
    })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Archive all' }))

    await waitFor(() => {
      expect(updateTicket).toHaveBeenNthCalledWith(1, 'ticket-1', { archived: true })
      expect(updateTicket).toHaveBeenNthCalledWith(2, 'ticket-2', { archived: true })
      expect(toastStore.success).toHaveBeenCalledWith('2 tickets archived.')
    })

    expect(
      within(await findByRole('list', { name: 'Todo tickets' })).queryByText('ASE-202'),
    ).toBeNull()
    expect(
      within(await findByRole('list', { name: 'Todo tickets' })).queryByText('ASE-203'),
    ).toBeNull()
  })

  it('does not call archive when a column only contains archived tickets', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses: StatusPayload = cloneValue({
      statuses: [
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted' as const,
          color: '#2563eb',
          icon: '',
          is_default: true,
          description: '',
          position: 1,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-cancelled',
          project_id: 'project-1',
          name: 'Cancelled',
          stage: 'canceled' as const,
          color: '#4b5563',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({
      tickets: [
        {
          ...ticketsFixture.tickets[0],
          id: 'ticket-1',
          identifier: 'ASE-202',
          status_id: 'status-cancelled',
          status_name: 'Cancelled',
          archived: true,
        },
      ],
    })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateStatus.mockResolvedValue({ status: currentStatuses.statuses[0] })
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Archive all' }))

    expect(updateTicket).not.toHaveBeenCalled()
    expect(toastStore.error).not.toHaveBeenCalled()
  })
})
