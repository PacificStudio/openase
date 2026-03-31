import type { TicketStatus } from '$lib/api/contracts'

export type StatusDraft = {
  name: string
  color: string
  isDefault: boolean
  maxActiveRuns: string
}

export type ParsedStatusDraft = {
  name: string
  color: string
  isDefault: boolean
  maxActiveRuns: number | null
}

export type EditableStatus = ParsedStatusDraft & {
  id: string
  position: number
  activeRuns: number
}

type ParseResult<T> = { ok: true; value: T } | { ok: false; error: string }

const HEX_COLOR_PATTERN = /^#[0-9a-f]{6}$/i

export function createEmptyStatusDraft(): StatusDraft {
  return {
    name: '',
    color: '#94a3b8',
    isDefault: false,
    maxActiveRuns: '',
  }
}

export function normalizeStatuses(statuses: TicketStatus[]): EditableStatus[] {
  return statuses
    .slice()
    .sort((left, right) => left.position - right.position)
    .map((status) => ({
      id: status.id,
      name: status.name,
      color: (status.color || '#94a3b8').toLowerCase(),
      isDefault: status.is_default,
      maxActiveRuns: typeof status.max_active_runs === 'number' ? status.max_active_runs : null,
      position: status.position,
      activeRuns: status.active_runs,
    }))
}

export function parseStatusDraft(raw: StatusDraft): ParseResult<ParsedStatusDraft> {
  const name = raw.name.trim()
  if (!name) {
    return { ok: false, error: 'Status name is required.' }
  }

  const color = raw.color.trim()
  if (!HEX_COLOR_PATTERN.test(color)) {
    return { ok: false, error: 'Status color must be a 6-digit hex value.' }
  }

  const maxActiveRuns = String(raw.maxActiveRuns ?? '').trim()
  if (!maxActiveRuns) {
    return {
      ok: true,
      value: {
        name,
        color: color.toLowerCase(),
        isDefault: raw.isDefault,
        maxActiveRuns: null,
      },
    }
  }

  const parsed = Number(maxActiveRuns)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    return { ok: false, error: 'Status concurrency must be a whole number greater than 0.' }
  }

  return {
    ok: true,
    value: {
      name,
      color: color.toLowerCase(),
      isDefault: raw.isDefault,
      maxActiveRuns: parsed,
    },
  }
}

export function moveStatus(
  statuses: EditableStatus[],
  statusId: string,
  direction: 'up' | 'down',
): EditableStatus[] {
  const currentIndex = statuses.findIndex((status) => status.id === statusId)
  if (currentIndex < 0) {
    return statuses
  }

  const targetIndex = direction === 'up' ? currentIndex - 1 : currentIndex + 1
  if (targetIndex < 0 || targetIndex >= statuses.length) {
    return statuses
  }

  const nextStatuses = statuses.slice()
  const [moved] = nextStatuses.splice(currentIndex, 1)
  nextStatuses.splice(targetIndex, 0, moved)

  return nextStatuses.map((status, index) => ({
    ...status,
    position: index,
  }))
}
