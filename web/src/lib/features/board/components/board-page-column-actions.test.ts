import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { StatusPayload } from '$lib/api/contracts'
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
  assertColumnMoveState,
  cloneValue,
  listColumnNames,
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

  it('enables move left and move right only within the current stage', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses: StatusPayload = cloneValue({
      statuses: [
        {
          id: 'status-backlog',
          project_id: 'project-1',
          name: 'Inbox',
          stage: 'backlog' as const,
          color: '#64748b',
          icon: '',
          is_default: false,
          description: '',
          position: 0,
          active_runs: 0,
          max_active_runs: null,
        },
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
          id: 'status-triage',
          project_id: 'project-1',
          name: 'Triage',
          stage: 'unstarted' as const,
          color: '#0ea5e9',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
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
          position: 3,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-review',
          project_id: 'project-1',
          name: 'Review',
          stage: 'started' as const,
          color: '#f97316',
          icon: '',
          is_default: false,
          description: '',
          position: 4,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-done',
          project_id: 'project-1',
          name: 'Done',
          stage: 'completed' as const,
          color: '#10b981',
          icon: '',
          is_default: false,
          description: '',
          position: 5,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockResolvedValue({ status: currentStatuses.statuses[0] })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await assertColumnMoveState(findByRole, 'Inbox', {
      leftDisabled: true,
      rightDisabled: true,
    })
    await assertColumnMoveState(findByRole, 'Todo', {
      leftDisabled: true,
      rightDisabled: false,
    })
    await assertColumnMoveState(findByRole, 'Triage', {
      leftDisabled: false,
      rightDisabled: true,
    })
    await assertColumnMoveState(findByRole, 'Doing', {
      leftDisabled: true,
      rightDisabled: false,
    })
    await assertColumnMoveState(findByRole, 'Review', {
      leftDisabled: false,
      rightDisabled: true,
    })
    await assertColumnMoveState(findByRole, 'Done', {
      leftDisabled: true,
      rightDisabled: true,
    })
  })

  it('shows concurrency actions in the column menu and only enables clear when a limit exists', async () => {
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
          active_runs: 1,
          max_active_runs: 3,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockResolvedValue({ status: currentStatuses.statuses[0] })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Todo')
    expect(await findByRole('menuitem', { name: 'Set concurrency' })).toBeTruthy()
    expect(
      (await findByRole('menuitem', { name: 'Clear concurrency' })).hasAttribute('data-disabled'),
    ).toBe(true)
    await fireEvent.keyDown(document.body, { key: 'Escape' })

    await openColumnActionMenu(findByRole, 'Doing')
    expect(await findByRole('menuitem', { name: 'Set concurrency' })).toBeTruthy()
    expect(
      (await findByRole('menuitem', { name: 'Clear concurrency' })).hasAttribute('data-disabled'),
    ).toBe(false)
  })

  it('updates a status concurrency limit from the column menu prompt', async () => {
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
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockImplementation(
      async (statusId: string, patch: { max_active_runs?: number | null }) => {
        const status = currentStatuses.statuses.find((item) => item.id === statusId)
        if (!status) throw new Error(`unknown status ${statusId}`)
        status.max_active_runs = patch.max_active_runs ?? null
        return { status: cloneValue(status) }
      },
    )
    connectEventStream.mockReturnValue(() => {})
    const promptSpy = vi.spyOn(window, 'prompt').mockReturnValue('3')

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Set concurrency' }))

    await waitFor(() => {
      expect(promptSpy).toHaveBeenCalledWith(
        'Set concurrency limit for "Todo". Leave blank for Unlimited.',
        '',
      )
      expect(updateStatus).toHaveBeenCalledWith('status-todo', { max_active_runs: 3 })
    })
  })

  it('clears a status concurrency limit and ignores invalid prompt values', async () => {
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
          active_runs: 1,
          max_active_runs: 2,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockImplementation(
      async (statusId: string, patch: { max_active_runs?: number | null }) => {
        const status = currentStatuses.statuses.find((item) => item.id === statusId)
        if (!status) throw new Error(`unknown status ${statusId}`)
        status.max_active_runs = patch.max_active_runs ?? null
        return { status: cloneValue(status) }
      },
    )
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    const promptSpy = vi.spyOn(window, 'prompt').mockReturnValue('0')
    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Set concurrency' }))
    expect(updateStatus).not.toHaveBeenCalled()
    expect(promptSpy).toHaveBeenCalledWith(
      'Set concurrency limit for "Todo". Leave blank for Unlimited.',
      '2',
    )

    promptSpy.mockRestore()
    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Clear concurrency' }))

    await waitFor(() => {
      expect(updateStatus).toHaveBeenCalledWith('status-todo', { max_active_runs: null })
    })
  })

  it('moves a column right by swapping positions with the next status in the same stage', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses = cloneValue({
      statuses: [
        {
          id: 'status-backlog',
          project_id: 'project-1',
          name: 'Inbox',
          stage: 'backlog' as const,
          color: '#64748b',
          icon: '',
          is_default: false,
          description: '',
          position: 0,
          active_runs: 0,
          max_active_runs: null,
        },
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
          id: 'status-triage',
          project_id: 'project-1',
          name: 'Triage',
          stage: 'unstarted' as const,
          color: '#0ea5e9',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
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
          position: 3,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockImplementation(async (statusId: string, patch: { position?: number }) => {
      const status = currentStatuses.statuses.find((item) => item.id === statusId)
      if (!status || typeof patch.position !== 'number') {
        throw new Error(`unknown status update ${statusId}`)
      }
      status.position = patch.position
      return { status: cloneValue(status) }
    })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Move right' }))

    await waitFor(() => {
      expect(updateStatus).toHaveBeenNthCalledWith(1, 'status-todo', { position: 2 })
      expect(updateStatus).toHaveBeenNthCalledWith(2, 'status-triage', { position: 1 })
      expect(listColumnNames()).toEqual(['Inbox', 'Triage', 'Todo', 'Doing'])
    })
  })

  it('moves a column left by swapping positions with the previous status in the same stage', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses = cloneValue({
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
          id: 'status-triage',
          project_id: 'project-1',
          name: 'Triage',
          stage: 'unstarted' as const,
          color: '#0ea5e9',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-review',
          project_id: 'project-1',
          name: 'Review',
          stage: 'started' as const,
          color: '#f97316',
          icon: '',
          is_default: false,
          description: '',
          position: 3,
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
          position: 4,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockImplementation(async (statusId: string, patch: { position?: number }) => {
      const status = currentStatuses.statuses.find((item) => item.id === statusId)
      if (!status || typeof patch.position !== 'number') {
        throw new Error(`unknown status update ${statusId}`)
      }
      status.position = patch.position
      return { status: cloneValue(status) }
    })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Doing')
    await fireEvent.click(await findByRole('menuitem', { name: 'Move left' }))

    await waitFor(() => {
      expect(updateStatus).toHaveBeenNthCalledWith(1, 'status-doing', { position: 3 })
      expect(updateStatus).toHaveBeenNthCalledWith(2, 'status-review', { position: 4 })
      expect(listColumnNames()).toEqual(['Todo', 'Triage', 'Doing', 'Review'])
    })
  })
})
