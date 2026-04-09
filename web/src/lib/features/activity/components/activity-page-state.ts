import type { ActivityPayload, TicketPayload } from '$lib/api/contracts'
import type { ActivityEntry } from '../types'

export const activityPageSize = 40

export type ActivitySnapshotState = {
  entries: ActivityEntry[]
  nextCursor: string
  hasMore: boolean
}

export function mapActivityEntries(activityPayload: ActivityPayload, ticketPayload: TicketPayload) {
  const ticketIdentifiers = new Map(
    ticketPayload.tickets.map((ticket) => [ticket.id, ticket.identifier]),
  )

  return activityPayload.events.map((event) => ({
    id: event.id,
    eventType: event.event_type,
    message: event.message,
    timestamp: event.created_at,
    ticketIdentifier: event.ticket_id
      ? (ticketIdentifiers.get(event.ticket_id) ?? event.ticket_id)
      : undefined,
    agentName: agentNameFromMetadata(event.metadata),
    metadata: event.metadata,
  }))
}

export function mergeActivityEntries(
  currentEntries: ActivityEntry[],
  incomingEntries: ActivityEntry[],
) {
  const merged = new Map(currentEntries.map((entry) => [entry.id, entry]))
  for (const entry of incomingEntries) {
    merged.set(entry.id, entry)
  }
  return [...merged.values()].sort(compareActivityEntries)
}

function compareActivityEntries(left: ActivityEntry, right: ActivityEntry) {
  if (left.timestamp !== right.timestamp) {
    return right.timestamp.localeCompare(left.timestamp)
  }
  return right.id.localeCompare(left.id)
}

function agentNameFromMetadata(metadata: Record<string, unknown>) {
  const value = metadata.agent_name
  return typeof value === 'string' ? value : undefined
}
