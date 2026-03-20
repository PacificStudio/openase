import type { StreamConnectionState } from '$lib/api/sse'
import type { ActivityEvent, Agent, Ticket } from './types'

export function chooseAgentSelection(items: Agent[], preferredAgentId: string) {
  if (preferredAgentId && items.some((item) => item.id === preferredAgentId)) {
    return preferredAgentId
  }

  return (
    items.find((item) => item.status === 'running')?.id ??
    items.find((item) => item.status === 'claimed')?.id ??
    items[0]?.id ??
    ''
  )
}

export function dedupeActivityEvents(items: ActivityEvent[]) {
  const seen = new Set<string>()
  return items.filter((item) => {
    if (seen.has(item.id)) {
      return false
    }
    seen.add(item.id)
    return true
  })
}

export function hasAutomationSignal(
  agents: Agent[],
  tickets: Ticket[],
  activityEvents: ActivityEvent[],
) {
  return (
    agents.some(
      (item) =>
        item.total_tickets_completed > 0 ||
        item.status === 'running' ||
        item.status === 'claimed' ||
        Boolean(item.current_ticket_id),
    ) ||
    tickets.some((item) => item.attempt_count > 0 || item.cost_amount > 0) ||
    activityEvents.some((item) => Boolean(item.agent_id) || Boolean(item.ticket_id))
  )
}

export function heartbeatAgeSeconds(timestamp: string | null | undefined, heartbeatNow: number) {
  if (!timestamp) {
    return null
  }

  const parsed = Date.parse(timestamp)
  if (Number.isNaN(parsed)) {
    return null
  }

  return Math.max(0, Math.floor((heartbeatNow - parsed) / 1000))
}

export function heartbeatTone(timestamp: string | null | undefined, heartbeatNow: number) {
  const ageSeconds = heartbeatAgeSeconds(timestamp, heartbeatNow)
  if (ageSeconds === null) {
    return 'stalled'
  }
  if (ageSeconds <= 60) {
    return 'healthy'
  }
  if (ageSeconds <= 180) {
    return 'warning'
  }
  return 'stalled'
}

export function heartbeatLabel(timestamp: string | null | undefined, heartbeatNow: number) {
  if (!timestamp) {
    return 'No heartbeat'
  }

  const ageSeconds = heartbeatAgeSeconds(timestamp, heartbeatNow)
  if (ageSeconds === null) {
    return 'Invalid heartbeat'
  }
  if (ageSeconds < 60) {
    return `${ageSeconds}s ago`
  }

  return `${Math.floor(ageSeconds / 60)}m ago`
}

export function heartbeatBadgeClass(timestamp: string | null | undefined, heartbeatNow: number) {
  switch (heartbeatTone(timestamp, heartbeatNow)) {
    case 'healthy':
      return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700'
    case 'warning':
      return 'border-amber-500/25 bg-amber-500/10 text-amber-700'
    default:
      return 'border-rose-500/25 bg-rose-500/10 text-rose-700'
  }
}

export function stalledAgentCount(agents: Agent[], heartbeatNow: number) {
  return agents.filter((item) => heartbeatTone(item.last_heartbeat_at, heartbeatNow) === 'stalled')
    .length
}

export function streamBadgeClass(state: StreamConnectionState) {
  switch (state) {
    case 'live':
      return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700'
    case 'connecting':
      return 'border-sky-500/25 bg-sky-500/10 text-sky-700'
    case 'retrying':
      return 'border-amber-500/25 bg-amber-500/10 text-amber-700'
    default:
      return 'border-border/80 bg-background text-muted-foreground'
  }
}

export function formatTimestamp(value: string) {
  const parsed = Date.parse(value)
  if (Number.isNaN(parsed)) {
    return value
  }

  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    month: 'short',
    day: 'numeric',
  }).format(parsed)
}
