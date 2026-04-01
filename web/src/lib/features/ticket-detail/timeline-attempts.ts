import { activityEventLabel } from '$lib/features/activity'
import type { TicketActivityTimelineItem, TicketTimelineItem } from './types'

export type TicketTimelineRenderBlock =
  | {
      kind: 'item'
      id: string
      item: TicketTimelineItem
    }
  | {
      kind: 'attempt'
      id: string
      attemptNumber: number
      runId: string
      items: TicketActivityTimelineItem[]
      summary: string
      tone: 'info' | 'success' | 'warning' | 'danger' | 'neutral'
      isLatest: boolean
    }

export function buildTimelineRenderBlocks(
  timeline: TicketTimelineItem[],
): TicketTimelineRenderBlock[] {
  const blocks: TicketTimelineRenderBlock[] = []
  const grouped: Array<{ runId: string; items: TicketActivityTimelineItem[] }> = []

  for (let index = 0; index < timeline.length; ) {
    const item = timeline[index]
    if (!isAttemptActivity(item)) {
      blocks.push({ kind: 'item', id: item.id, item })
      index += 1
      continue
    }

    const runId = activityRunID(item)
    const items: TicketActivityTimelineItem[] = [item]
    index += 1

    while (index < timeline.length) {
      const candidate = timeline[index]
      if (!isAttemptActivity(candidate) || activityRunID(candidate) !== runId) {
        break
      }
      items.push(candidate)
      index += 1
    }

    if (items.length < 2) {
      blocks.push({ kind: 'item', id: item.id, item })
      continue
    }

    grouped.push({ runId, items })
    blocks.push({
      kind: 'attempt',
      id: `attempt:${runId}`,
      attemptNumber: 0,
      runId,
      items,
      summary: '',
      tone: 'neutral',
      isLatest: false,
    })
  }

  if (grouped.length === 0) {
    return blocks
  }

  const groupedByRun = new Map(grouped.map((group, index) => [group.runId, { group, index }]))

  return blocks.map((block) => {
    if (block.kind !== 'attempt') {
      return block
    }
    const entry = groupedByRun.get(block.runId)
    if (!entry) return block

    const attemptNumber = entry.index + 1
    const lastItem = entry.group.items.at(-1) ?? entry.group.items[0]
    return {
      ...block,
      attemptNumber,
      summary: summarizeAttempt(lastItem),
      tone: summarizeAttemptTone(lastItem),
      isLatest: entry.index === grouped.length - 1,
    }
  })
}

function isAttemptActivity(item: TicketTimelineItem): item is TicketActivityTimelineItem {
  return (
    item.kind === 'activity' && item.eventType.startsWith('agent.') && activityRunID(item) !== ''
  )
}

function activityRunID(item: TicketActivityTimelineItem) {
  const currentRunID = stringMetadata(item.metadata, 'current_run_id')
  if (currentRunID) return currentRunID
  return stringMetadata(item.metadata, 'run_id')
}

function summarizeAttempt(item: TicketActivityTimelineItem) {
  const label = activityEventLabel(item.eventType)
  if (label.toLowerCase().startsWith('agent ')) {
    return capitalizeLabel(label.slice('Agent '.length))
  }
  return capitalizeLabel(label)
}

function summarizeAttemptTone(
  item: TicketActivityTimelineItem,
): 'info' | 'success' | 'warning' | 'danger' | 'neutral' {
  switch (item.eventType) {
    case 'agent.failed':
      return 'danger'
    case 'agent.ready':
    case 'agent.completed':
      return 'success'
    case 'agent.paused':
      return 'warning'
    case 'agent.claimed':
    case 'agent.launching':
      return 'info'
    default:
      return 'neutral'
  }
}

function stringMetadata(metadata: Record<string, unknown>, key: string) {
  const value = metadata[key]
  return typeof value === 'string' && value.trim().length > 0 ? value.trim() : ''
}

function capitalizeLabel(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return ''
  return trimmed.charAt(0).toUpperCase() + trimmed.slice(1)
}
