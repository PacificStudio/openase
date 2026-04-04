import { cleanup, render, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'

import BoardColumn from './board-column.svelte'
import type { BoardColumn as BoardColumnModel, BoardStatusOption, BoardTicket } from '../types'

const statuses: BoardStatusOption[] = [
  {
    id: 'todo',
    name: 'Todo',
    color: '#94a3b8',
    stage: 'unstarted',
    position: 1,
    maxActiveRuns: null,
  },
]

afterEach(() => {
  cleanup()
})

describe('BoardColumn', () => {
  it('renders the inline add-ticket button above the empty-state message for empty columns', () => {
    const column = buildColumn([])
    const { getByRole } = render(BoardColumn, { column, statuses })

    const list = getByRole('list', { name: 'Todo tickets' })
    const addButton = within(list).getByLabelText('Add ticket to Todo')

    expect(list.firstElementChild).toBe(addButton)
    expect(list.lastElementChild).not.toBe(addButton)
  })

  it('renders the inline add-ticket button after the last ticket for populated columns', () => {
    const column = buildColumn([buildTicket()])
    const { getByRole, getByText } = render(BoardColumn, { column, statuses })

    const list = getByRole('list', { name: 'Todo tickets' })
    const addButton = within(list).getByLabelText('Add ticket to Todo')
    const ticketCard = getByText('Wire board page to runtime data').closest('button')

    expect(ticketCard).toBeTruthy()
    expect(list.lastElementChild).toBe(addButton)
    expect(list.firstElementChild).toBe(ticketCard)
  })
})

function buildColumn(tickets: BoardTicket[]): BoardColumnModel {
  return {
    id: 'todo',
    name: 'Todo',
    color: '#94a3b8',
    tickets,
  }
}

function buildTicket(): BoardTicket {
  return {
    id: 'ticket-1',
    statusId: 'todo',
    statusName: 'Todo',
    statusColor: '#94a3b8',
    stage: 'unstarted',
    identifier: 'ASE-202',
    title: 'Wire board page to runtime data',
    priority: 'high',
    archived: false,
    workflowType: 'coding',
    updatedAt: '2026-04-01T12:00:00Z',
  }
}
