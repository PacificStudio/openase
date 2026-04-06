import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import TicketCard from './ticket-card.svelte'
import type { BoardStatusOption, BoardTicket } from '../types'

const statuses: BoardStatusOption[] = [
  {
    id: 'status-todo',
    name: 'Todo',
    color: '#94a3b8',
    stage: 'unstarted',
    position: 1,
    maxActiveRuns: null,
  },
  {
    id: 'status-doing',
    name: 'Doing',
    color: '#f59e0b',
    stage: 'started',
    position: 2,
    maxActiveRuns: null,
  },
]

function buildTicket(overrides: Partial<BoardTicket> = {}): BoardTicket {
  return {
    id: 'ticket-1',
    statusId: 'status-todo',
    statusName: 'Todo',
    statusColor: '#94a3b8',
    stage: 'unstarted',
    identifier: 'ASE-42',
    title: 'Implement context menu',
    priority: 'medium',
    archived: false,
    workflowType: 'coding',
    updatedAt: '2026-04-01T12:00:00Z',
    ...overrides,
  }
}

afterEach(() => {
  cleanup()
})

describe('TicketCard context menu', () => {
  it('opens context menu on right click without triggering card click', async () => {
    const onclick = vi.fn()
    const ticket = buildTicket()
    const { getByText, findByText } = render(TicketCard, {
      ticket,
      statuses,
      onclick,
    })

    const card = getByText('ASE-42').closest('button')!
    await fireEvent.contextMenu(card)

    expect(onclick).not.toHaveBeenCalled()
    expect(await findByText('Open details')).toBeTruthy()
    expect(await findByText('Archive')).toBeTruthy()
    expect(await findByText('Copy ticket ID')).toBeTruthy()
    expect(await findByText('Change status')).toBeTruthy()
    expect(await findByText('Change priority')).toBeTruthy()
  })

  it('opens context menu via the ellipsis action button', async () => {
    const onclick = vi.fn()
    const ticket = buildTicket()
    const { getByLabelText, findByText } = render(TicketCard, {
      ticket,
      statuses,
      onclick,
    })

    const actionButton = getByLabelText('Ticket actions')
    await fireEvent.click(actionButton)

    expect(onclick).not.toHaveBeenCalled()
    expect(await findByText('Open details')).toBeTruthy()
  })

  it('triggers Open details which calls onclick with the ticket', async () => {
    const onclick = vi.fn()
    const ticket = buildTicket()
    const { getByText, findByText } = render(TicketCard, {
      ticket,
      statuses,
      onclick,
    })

    const card = getByText('ASE-42').closest('button')!
    await fireEvent.contextMenu(card)

    const openDetails = await findByText('Open details')
    await fireEvent.click(openDetails)

    expect(onclick).toHaveBeenCalledWith(ticket)
  })

  it('triggers Archive which calls onArchiveTicket with the ticket id', async () => {
    const onArchiveTicket = vi.fn()
    const ticket = buildTicket()
    const { getByText, findByText } = render(TicketCard, {
      ticket,
      statuses,
      onArchiveTicket,
    })

    const card = getByText('ASE-42').closest('button')!
    await fireEvent.contextMenu(card)

    const archiveItem = await findByText('Archive')
    await fireEvent.click(archiveItem)

    expect(onArchiveTicket).toHaveBeenCalledWith('ticket-1')
  })

  it('renders with pending move state without errors', async () => {
    const onArchiveTicket = vi.fn()
    const ticket = buildTicket({ isMoving: true })
    const { getByText } = render(TicketCard, {
      ticket,
      statuses,
      onArchiveTicket,
      isPendingMove: true,
    })

    // Card renders in pending state and the disabled button prevents
    // direct interaction; the ellipsis trigger and archive are disabled
    expect(getByText('ASE-42')).toBeTruthy()
    expect(getByText('Moving…')).toBeTruthy()
  })

  it('opens context menu via keyboard Shift+F10', async () => {
    const ticket = buildTicket()
    const { getByText, findByText } = render(TicketCard, {
      ticket,
      statuses,
    })

    const card = getByText('ASE-42').closest('button')!
    card.focus()
    await fireEvent.keyDown(card, { key: 'F10', shiftKey: true })

    expect(await findByText('Open details')).toBeTruthy()
    expect(await findByText('Archive')).toBeTruthy()
  })

  it('does not open context menu while dragging', async () => {
    const ticket = buildTicket()
    const { getByText, queryByText } = render(TicketCard, {
      ticket,
      statuses,
      isDragging: true,
    })

    const card = getByText('ASE-42').closest('button')!
    await fireEvent.contextMenu(card)

    // Menu should not appear when dragging
    await new Promise((resolve) => setTimeout(resolve, 50))
    expect(queryByText('Open details')).toBeNull()
  })

  it('existing card click still works when context menu is not open', async () => {
    const onclick = vi.fn()
    const ticket = buildTicket()
    const { getByText } = render(TicketCard, {
      ticket,
      statuses,
      onclick,
    })

    const card = getByText('ASE-42').closest('button')!
    await fireEvent.click(card)

    expect(onclick).toHaveBeenCalledWith(ticket)
  })
})
