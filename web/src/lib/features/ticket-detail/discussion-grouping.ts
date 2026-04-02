import { formatRelativeTime } from '$lib/utils'
import type { TicketActivityTimelineItem, TicketTimelineItem } from './types'

/**
 * A non-activity item rendered on its own (description, comment).
 */
export type StandaloneTimelineItem = {
  type: 'standalone'
  item: TicketTimelineItem
}

/**
 * A group of consecutive activity events collapsed into a single row.
 */
export type ActivityGroup = {
  type: 'activity_group'
  id: string
  items: TicketActivityTimelineItem[]
  summary: string
  detail: string
  timeRange: string
  hasFailed: boolean
}

export type DiscussionDisplayItem = StandaloneTimelineItem | ActivityGroup

function buildActivityGroupSummary(items: TicketActivityTimelineItem[]): {
  summary: string
  detail: string
  hasFailed: boolean
} {
  const types = new Map<string, number>()
  let hasFailed = false

  for (const item of items) {
    const label = humanizeEventType(item.eventType)
    types.set(label, (types.get(label) ?? 0) + 1)
    if (item.eventType.includes('failed') || item.eventType.includes('error')) {
      hasFailed = true
    }
  }

  if (items.length === 1) {
    return {
      summary: humanizeEventType(items[0].eventType),
      detail: items[0].bodyText || '',
      hasFailed,
    }
  }

  // Build a compact summary like "3× claimed, 3× launching, 2× failed"
  const parts: string[] = []
  for (const [label, count] of types) {
    parts.push(count > 1 ? `${count}× ${label}` : label)
  }

  // Derive a headline from the overall pattern
  const runIds = new Set(
    items
      .map((i) => (i.metadata.run_id as string) || (i.metadata.current_run_id as string) || '')
      .filter(Boolean),
  )
  const runCount = runIds.size
  const summary =
    runCount > 1
      ? `${items.length} agent events across ${runCount} runs`
      : `${items.length} agent events`

  return { summary, detail: parts.join(', '), hasFailed }
}

function humanizeEventType(eventType: string): string {
  const label = eventType.replace(/^(agent|ticket|pr|hook)\./, '')
  return label.replace(/[_-]+/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}

function timeRange(items: TicketActivityTimelineItem[]): string {
  const first = items[0]
  const last = items[items.length - 1]
  if (first === last) return formatRelativeTime(first.createdAt)
  return `${formatRelativeTime(first.createdAt)} – ${formatRelativeTime(last.createdAt)}`
}

/**
 * Groups consecutive activity events into collapsible blocks,
 * keeping description and comment items standalone.
 */
export function groupDiscussionTimeline(timeline: TicketTimelineItem[]): DiscussionDisplayItem[] {
  const items: DiscussionDisplayItem[] = []
  let activityBatch: TicketActivityTimelineItem[] = []

  function flushBatch() {
    if (activityBatch.length === 0) return
    const { summary, detail, hasFailed } = buildActivityGroupSummary(activityBatch)
    items.push({
      type: 'activity_group',
      id: `ag-${activityBatch[0].id}`,
      items: activityBatch,
      summary,
      detail,
      timeRange: timeRange(activityBatch),
      hasFailed,
    })
    activityBatch = []
  }

  for (const entry of timeline) {
    if (entry.kind === 'activity') {
      activityBatch.push(entry)
    } else {
      flushBatch()
      items.push({ type: 'standalone', item: entry })
    }
  }

  flushBatch()
  return items
}
