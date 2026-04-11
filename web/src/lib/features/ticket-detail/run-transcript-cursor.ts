import type { TicketRunStepEntry, TicketRunTraceEntry, TicketRunTranscriptItem } from './types'

const STEP_KIND = 'step'
const TRACE_KIND = 'trace'

type TranscriptCursorParts = {
  createdAt: string
  kind: typeof STEP_KIND | typeof TRACE_KIND
  order: number
  id: string
}

export function buildTicketRunTraceCursor(entry: TicketRunTraceEntry): string {
  return buildCursor({
    createdAt: entry.createdAt,
    kind: TRACE_KIND,
    order: entry.sequence,
    id: entry.id,
  })
}

export function buildTicketRunStepCursor(entry: TicketRunStepEntry): string {
  return buildCursor({
    createdAt: entry.createdAt,
    kind: STEP_KIND,
    order: 0,
    id: entry.id,
  })
}

export function buildTicketRunTranscriptItemCursor(item: TicketRunTranscriptItem): string {
  return item.kind === TRACE_KIND
    ? buildTicketRunTraceCursor(item.traceEntry)
    : buildTicketRunStepCursor(item.stepEntry)
}

export function compareTicketRunTranscriptCursors(left: string, right: string): number {
  const leftParts = parseCursor(left)
  const rightParts = parseCursor(right)
  if (!leftParts || !rightParts) {
    if (!leftParts && !rightParts) {
      return left.localeCompare(right)
    }
    return leftParts ? 1 : -1
  }

  const timeDiff = Date.parse(leftParts.createdAt) - Date.parse(rightParts.createdAt)
  if (timeDiff !== 0) {
    return timeDiff
  }

  const kindDiff = kindRank(leftParts.kind) - kindRank(rightParts.kind)
  if (kindDiff !== 0) {
    return kindDiff
  }

  if (leftParts.order !== rightParts.order) {
    return leftParts.order - rightParts.order
  }

  return leftParts.id.localeCompare(rightParts.id)
}

export function maxTicketRunTranscriptCursor(
  left: string | undefined,
  right: string | undefined,
): string | undefined {
  if (!left) return right
  if (!right) return left
  return compareTicketRunTranscriptCursors(left, right) >= 0 ? left : right
}

export function normalizeTicketRunTranscriptCursor(
  raw: string | null | undefined,
): string | undefined {
  const parsed = parseCursor(raw)
  return parsed ? buildCursor(parsed) : undefined
}

function buildCursor(parts: TranscriptCursorParts): string {
  return `${parts.createdAt}|${parts.kind}|${parts.order}|${parts.id}`
}

function parseCursor(cursor: string | null | undefined): TranscriptCursorParts | null {
  const trimmed = cursor?.trim() ?? ''
  if (!trimmed) {
    return null
  }

  const parts = trimmed.split('|')
  if (parts.length !== 4) {
    return null
  }

  const [createdAt = '', kind = STEP_KIND, orderText = '0', id = ''] = parts
  if (!createdAt || Number.isNaN(Date.parse(createdAt))) {
    return null
  }
  if (kind !== STEP_KIND && kind !== TRACE_KIND) {
    return null
  }

  const order = Number.parseInt(orderText, 10)
  if (!Number.isInteger(order) || id.trim() === '') {
    return null
  }

  return {
    createdAt,
    kind,
    order,
    id,
  }
}

function kindRank(kind: TranscriptCursorParts['kind']): number {
  return kind === STEP_KIND ? 0 : 1
}
