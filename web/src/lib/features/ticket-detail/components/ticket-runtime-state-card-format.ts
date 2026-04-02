import type { TicketPickupDiagnosis } from '../pickup-diagnosis'
import type { RuntimeSummary, RuntimeTone } from './ticket-runtime-state-card-view'

export function diagnosisLabel(state: TicketPickupDiagnosis['state']) {
  switch (state) {
    case 'runnable':
      return 'Runnable'
    case 'waiting':
      return 'Waiting'
    case 'blocked':
      return 'Blocked'
    case 'running':
      return 'Running'
    case 'completed':
      return 'Completed'
    default:
      return 'Unavailable'
  }
}

export function diagnosisTone(
  state: TicketPickupDiagnosis['state'],
  reasonCode: TicketPickupDiagnosis['primaryReasonCode'],
): RuntimeTone {
  if (state === 'running' || state === 'completed' || state === 'runnable') {
    return state === 'running' ? 'success' : state === 'completed' ? 'success' : 'info'
  }
  if (state === 'waiting') return 'info'
  if (
    reasonCode === 'retry_paused_repeated_stalls' ||
    reasonCode === 'retry_paused_budget' ||
    reasonCode === 'retry_paused_user' ||
    reasonCode === 'blocked_dependency'
  ) {
    return 'warning'
  }
  return 'danger'
}

export function humanizeControlState(value: string) {
  switch (value) {
    case 'pause_requested':
      return 'Pause requested'
    case 'paused':
      return 'Paused'
    default:
      return 'Active'
  }
}

export function humanizeProviderState(provider: NonNullable<TicketPickupDiagnosis['provider']>) {
  const state =
    provider.availabilityState === 'available'
      ? 'Available'
      : provider.availabilityState === 'stale'
        ? 'Stale health'
        : provider.availabilityState === 'unknown'
          ? 'Unknown health'
          : 'Unavailable'
  if (!provider.availabilityReason) return state
  return `${state} (${humanizeAvailabilityReason(provider.availabilityReason)})`
}

export function humanizePauseReason(reason: string) {
  switch (reason) {
    case 'repeated_stalls':
      return 'Manual retry required after repeated stalls.'
    case 'budget_exhausted':
      return 'Retries are paused because the budget is exhausted.'
    case 'user_paused':
      return 'Retries are paused manually.'
    default:
      return reason.replaceAll('_', ' ')
  }
}

export function buildCapacityLine(diagnosis: TicketPickupDiagnosis) {
  const entries: string[] = []
  pushCapacityLine(entries, 'Workflow', diagnosis.capacity.workflow)
  pushCapacityLine(entries, 'Project', diagnosis.capacity.project)
  pushCapacityLine(entries, 'Provider', diagnosis.capacity.provider)
  pushCapacityLine(entries, 'Status', diagnosis.capacity.status)
  return entries.join(' · ')
}

export function buildLegacyRepeatedStallDetail(): RuntimeSummary['detailItems'] {
  return [{ label: 'Retry', value: 'Manual retry required after repeated stalls.' }]
}

export function formatRetryCountdown(value: string, now: number) {
  const targetMs = new Date(value).getTime()
  if (!Number.isFinite(targetMs)) return undefined

  const absolute = formatUTC(value)
  const remainingMs = Math.max(0, targetMs - now)
  if (remainingMs === 0) {
    return `Retry window elapsed (at ${absolute})`
  }

  return `Retrying in ${formatRemaining(remainingMs)} (at ${absolute})`
}

function humanizeAvailabilityReason(reason: string) {
  switch (reason) {
    case 'machine_offline':
      return 'machine offline'
    case 'machine_degraded':
      return 'machine degraded'
    case 'machine_maintenance':
      return 'machine maintenance'
    case 'l4_snapshot_missing':
      return 'health not probed'
    case 'stale_l4_snapshot':
      return 'stale health snapshot'
    case 'cli_missing':
      return 'CLI missing'
    case 'not_logged_in':
      return 'not logged in'
    case 'not_ready':
      return 'CLI not ready'
    case 'config_incomplete':
      return 'config incomplete'
    default:
      return reason.replaceAll('_', ' ')
  }
}

function pushCapacityLine(
  entries: string[],
  label: string,
  item: { limited: boolean; activeRuns: number; capacity?: number },
) {
  if (!item.limited || item.capacity === undefined || item.activeRuns < item.capacity) {
    return
  }
  entries.push(`${label} ${item.activeRuns}/${item.capacity}`)
}

function formatRemaining(value: number) {
  const totalSeconds = Math.max(0, Math.floor(value / 1000))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (hours > 0) {
    return `${hours}h ${String(minutes).padStart(2, '0')}m`
  }
  if (minutes > 0) {
    return `${minutes}m ${String(seconds).padStart(2, '0')}s`
  }
  return `${seconds}s`
}

function formatUTC(value: string) {
  const date = new Date(value)
  const year = date.getUTCFullYear()
  const month = String(date.getUTCMonth() + 1).padStart(2, '0')
  const day = String(date.getUTCDate()).padStart(2, '0')
  const hours = String(date.getUTCHours()).padStart(2, '0')
  const minutes = String(date.getUTCMinutes()).padStart(2, '0')
  const seconds = String(date.getUTCSeconds()).padStart(2, '0')
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds} UTC`
}
