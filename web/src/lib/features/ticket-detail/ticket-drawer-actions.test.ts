import { beforeEach, describe, expect, it, vi } from 'vitest'

import { buildLiveContext } from './drawer-state.test-fixtures'
import { createTicketDrawerActions } from './ticket-drawer-actions'

const { updateTicket } = vi.hoisted(() => ({
  updateTicket: vi.fn(),
}))

const { resetTicketWorkspace } = vi.hoisted(() => ({
  resetTicketWorkspace: vi.fn(),
}))

const { runTicketDrawerMutation } = vi.hoisted(() => ({
  runTicketDrawerMutation: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createTicketRepoScope: vi.fn(),
  deleteTicketRepoScope: vi.fn(),
  resetTicketWorkspace,
  updateTicket,
  updateTicketRepoScope: vi.fn(),
}))

vi.mock('./drawer-mutation', () => ({
  runTicketDrawerMutation,
}))

describe('createTicketDrawerActions.handleArchive', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('routes archive mutations to the archived flag', async () => {
    const ticket = buildLiveContext().ticket
    const setMutationError = vi.fn()
    updateTicket.mockResolvedValue({ ticket: { id: ticket.id } })
    runTicketDrawerMutation.mockImplementationOnce(async (input) => {
      expect(input.optimisticUpdate(ticket)).toMatchObject({
        archived: true,
        status: ticket.status,
      })
      await input.mutate()
      return true
    })

    const actions = createTicketDrawerActions({
      getProjectId: () => 'project-1',
      getTicketId: () => ticket.id,
      drawerState: {
        ticket,
        statuses: [],
        setMutationError,
      } as never,
      buildDrawerMutation: () => ({
        ticket,
        projectId: 'project-1',
        ticketId: ticket.id,
        load: vi.fn().mockResolvedValue(undefined),
        applyTicket: vi.fn(),
        clearMessages: vi.fn(),
        setError: vi.fn(),
        setNotice: vi.fn(),
      }),
    })

    await actions.handleArchive()

    expect(runTicketDrawerMutation).toHaveBeenCalledTimes(1)
    expect(updateTicket).toHaveBeenCalledWith(ticket.id, { archived: true })
    expect(setMutationError).not.toHaveBeenCalled()
  })
})

describe('createTicketDrawerActions.handleResetWorkspace', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('routes workspace reset through the shared drawer mutation pipeline', async () => {
    const ticket = {
      ...buildLiveContext().ticket,
      currentRunId: undefined,
    }
    resetTicketWorkspace.mockResolvedValue({ reset: true })
    runTicketDrawerMutation.mockImplementationOnce(async (input) => {
      expect(input.optimisticUpdate(ticket)).toBe(ticket)
      await input.mutate()
      return true
    })

    const actions = createTicketDrawerActions({
      getProjectId: () => 'project-1',
      getTicketId: () => ticket.id,
      drawerState: {
        ticket,
        statuses: [],
        resettingWorkspace: false,
        setMutationError: vi.fn(),
      } as never,
      buildDrawerMutation: () => ({
        ticket,
        projectId: 'project-1',
        ticketId: ticket.id,
        load: vi.fn().mockResolvedValue(undefined),
        applyTicket: vi.fn(),
        clearMessages: vi.fn(),
        setError: vi.fn(),
        setNotice: vi.fn(),
      }),
    })

    await actions.handleResetWorkspace()

    expect(runTicketDrawerMutation).toHaveBeenCalledTimes(1)
    expect(resetTicketWorkspace).toHaveBeenCalledWith(ticket.id)
  })
})
