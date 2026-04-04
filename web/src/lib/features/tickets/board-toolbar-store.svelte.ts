import type { BoardFilter } from '$lib/features/board'
import { parseBoardFilterPriority } from '$lib/features/board/public'

const storagePrefix = 'openase.ticket-board.toolbar'

export type TicketBoardToolbarState = {
  filter: BoardFilter
  hideEmpty: boolean
}

function defaultState(): TicketBoardToolbarState {
  return {
    filter: { search: '' },
    hideEmpty: true,
  }
}

function storageKey(projectId: string) {
  return `${storagePrefix}.${projectId}`
}

function normalizeProjectId(projectId: string | null | undefined) {
  return typeof projectId === 'string' ? projectId.trim() : ''
}

function normalizeChoice(value: unknown) {
  if (typeof value !== 'string') {
    return undefined
  }
  const trimmed = value.trim()
  return trimmed ? trimmed : undefined
}

function normalizeSearch(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function normalizePriority(value: unknown): BoardFilter['priority'] {
  if (typeof value !== 'string') {
    return undefined
  }
  return parseBoardFilterPriority(value)
}

function normalizeFilter(filter: BoardFilter | null | undefined): BoardFilter {
  if (!filter) {
    return { search: '' }
  }

  const normalized: BoardFilter = {
    search: normalizeSearch(filter.search),
  }

  const workflow = normalizeChoice(filter.workflow)
  if (workflow) {
    normalized.workflow = workflow
  }

  const agent = normalizeChoice(filter.agent)
  if (agent) {
    normalized.agent = agent
  }

  const priority = normalizePriority(filter.priority)
  if (priority) {
    normalized.priority = priority
  }

  if (filter.anomalyOnly === true) {
    normalized.anomalyOnly = true
  }

  return normalized
}

function sameFilter(left: BoardFilter, right: BoardFilter) {
  return (
    left.search === right.search &&
    left.workflow === right.workflow &&
    left.agent === right.agent &&
    left.priority === right.priority &&
    left.anomalyOnly === right.anomalyOnly
  )
}

function parseStoredState(raw: string | null): TicketBoardToolbarState {
  if (!raw) {
    return defaultState()
  }

  try {
    const parsed = JSON.parse(raw) as {
      filter?: BoardFilter
      hideEmpty?: unknown
    }
    return {
      filter: normalizeFilter(parsed.filter),
      hideEmpty: parsed.hideEmpty === false ? false : true,
    }
  } catch {
    return defaultState()
  }
}

export function readProjectTicketBoardToolbarState(projectId: string): TicketBoardToolbarState {
  if (typeof window === 'undefined') {
    return defaultState()
  }

  const normalizedProjectId = normalizeProjectId(projectId)
  if (!normalizedProjectId) {
    return defaultState()
  }

  try {
    return parseStoredState(window.localStorage.getItem(storageKey(normalizedProjectId)))
  } catch {
    return defaultState()
  }
}

class TicketBoardToolbarStore {
  projectId = $state<string | null>(null)
  filter = $state<BoardFilter>(defaultState().filter)
  hideEmpty = $state(defaultState().hideEmpty)

  activateProject(projectId: string | null | undefined) {
    const normalizedProjectId = normalizeProjectId(projectId)
    const nextProjectId = normalizedProjectId || null
    if (this.projectId === nextProjectId) {
      return
    }

    this.projectId = nextProjectId

    const nextState = nextProjectId
      ? readProjectTicketBoardToolbarState(nextProjectId)
      : defaultState()
    this.filter = nextState.filter
    this.hideEmpty = nextState.hideEmpty
  }

  setFilter(nextFilter: BoardFilter) {
    const normalizedFilter = normalizeFilter(nextFilter)
    if (sameFilter(this.filter, normalizedFilter)) {
      return
    }

    this.filter = normalizedFilter
    this.persist()
  }

  setHideEmpty(nextHideEmpty: boolean) {
    if (this.hideEmpty === nextHideEmpty) {
      return
    }

    this.hideEmpty = nextHideEmpty
    this.persist()
  }

  resetForTests() {
    this.projectId = null
    this.filter = defaultState().filter
    this.hideEmpty = defaultState().hideEmpty
  }

  private persist() {
    if (typeof window === 'undefined' || !this.projectId) {
      return
    }

    try {
      window.localStorage.setItem(
        storageKey(this.projectId),
        JSON.stringify({
          filter: normalizeFilter(this.filter),
          hideEmpty: this.hideEmpty,
        }),
      )
    } catch {
      // Ignore localStorage failures.
    }
  }
}

export const ticketBoardToolbarStore = new TicketBoardToolbarStore()

export function resetTicketBoardToolbarStoreForTests() {
  ticketBoardToolbarStore.resetForTests()
}
