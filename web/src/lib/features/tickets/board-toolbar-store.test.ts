import { afterEach, describe, expect, it } from 'vitest'

import {
  readProjectTicketBoardToolbarState,
  resetTicketBoardToolbarStoreForTests,
  ticketBoardToolbarStore,
} from './board-toolbar-store.svelte'

describe('ticketBoardToolbarStore', () => {
  afterEach(() => {
    resetTicketBoardToolbarStoreForTests()
    window.localStorage.clear()
  })

  it('persists board toolbar state per project', () => {
    ticketBoardToolbarStore.activateProject('project-1')
    ticketBoardToolbarStore.setFilter({
      search: 'runtime',
      workflow: 'coding',
      agent: 'Codex Worker',
      priority: 'high',
      anomalyOnly: true,
    })
    ticketBoardToolbarStore.setHideEmpty(false)

    expect(readProjectTicketBoardToolbarState('project-1')).toEqual({
      filter: {
        search: 'runtime',
        workflow: 'coding',
        agent: 'Codex Worker',
        priority: 'high',
        anomalyOnly: true,
      },
      hideEmpty: false,
    })

    ticketBoardToolbarStore.activateProject('project-2')

    expect(ticketBoardToolbarStore.filter).toEqual({ search: '' })
    expect(ticketBoardToolbarStore.hideEmpty).toBe(true)

    ticketBoardToolbarStore.setFilter({ search: 'infra' })

    expect(readProjectTicketBoardToolbarState('project-2')).toEqual({
      filter: { search: 'infra' },
      hideEmpty: true,
    })
    expect(readProjectTicketBoardToolbarState('project-1')).toEqual({
      filter: {
        search: 'runtime',
        workflow: 'coding',
        agent: 'Codex Worker',
        priority: 'high',
        anomalyOnly: true,
      },
      hideEmpty: false,
    })
  })

  it('parses malformed local storage into a safe default state', () => {
    window.localStorage.setItem(
      'openase.ticket-board.toolbar.project-1',
      JSON.stringify({
        filter: {
          search: 123,
          workflow: ' ',
          agent: ['Codex Worker'],
          priority: 'p1',
          anomalyOnly: 'yes',
        },
        hideEmpty: 'no',
      }),
    )

    expect(readProjectTicketBoardToolbarState('project-1')).toEqual({
      filter: { search: '' },
      hideEmpty: true,
    })
  })
})
