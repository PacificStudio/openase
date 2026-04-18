import { describe, expect, it } from 'vitest'

import { DEFAULT_STATUS_IDS, PROJECT_ID } from './constants'
import { createInitialState, seedBoardState } from './initial-state'

describe('mock api initial state', () => {
  it('seeds the core project data in one place', () => {
    const state = createInitialState()

    expect(state.organizations).toHaveLength(1)
    expect(state.projects).toHaveLength(1)
    expect(state.providers).toHaveLength(4)
    expect(state.workflows).toHaveLength(1)
    expect(state.securitySettingsByProjectId[PROJECT_ID]).toBeDefined()
  })

  it('rebuilds board fixtures from the requested status counts', () => {
    const state = createInitialState()

    seedBoardState(state, {
      [DEFAULT_STATUS_IDS.todo]: 2,
      [DEFAULT_STATUS_IDS.review]: 1,
      [DEFAULT_STATUS_IDS.done]: 0,
    })

    expect(state.tickets).toHaveLength(3)
    expect(state.activityEvents).toEqual([])
    expect(state.tickets.map((ticket) => ticket.status_id)).toEqual([
      DEFAULT_STATUS_IDS.todo,
      DEFAULT_STATUS_IDS.todo,
      DEFAULT_STATUS_IDS.review,
    ])
  })
})
