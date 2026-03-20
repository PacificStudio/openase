import type { ActivityEvent, StreamEnvelope } from '$lib/features/workspace'

export function extractRelatedTicketId(payload: unknown): string | null {
  const source = readRecord(payload)
  if (!source) {
    return null
  }

  const nestedTicket = readRecord(source.ticket)
  if (nestedTicket) {
    return readString(nestedTicket, 'id') ?? readNullableString(nestedTicket, 'ticket_id') ?? null
  }

  return readNullableString(source, 'ticket_id') ?? readString(source, 'ticketId') ?? null
}

export function isHookEvent(item: ActivityEvent) {
  if (item.event_type.toLowerCase().includes('hook')) {
    return true
  }

  return ['hook', 'hook_name', 'hook_stage', 'hook_result', 'hook_outcome'].some(
    (key) => key in item.metadata,
  )
}

export function shouldReloadTicket(envelope: StreamEnvelope, ticketId: string) {
  return extractRelatedTicketId(envelope.payload) === ticketId
}

function readRecord(value: unknown) {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null
}

function readString(source: Record<string, unknown>, key: string) {
  const value = source[key]
  return typeof value === 'string' && value.trim() ? value : undefined
}

function readNullableString(source: Record<string, unknown>, key: string) {
  const value = source[key]
  if (value === null) {
    return null
  }

  return typeof value === 'string' ? value : undefined
}
