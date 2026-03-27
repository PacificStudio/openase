import type { BadgeVariant } from '$ui/badge'

export type ProviderAvailabilityState = 'unknown' | 'available' | 'unavailable' | 'stale'

export function normalizeProviderAvailabilityState(
  raw: string | null | undefined,
): ProviderAvailabilityState {
  if (raw === 'unknown' || raw === 'available' || raw === 'unavailable' || raw === 'stale') {
    return raw
  }

  return 'unknown'
}

export function providerAvailabilityLabel(raw: string | null | undefined): string {
  switch (normalizeProviderAvailabilityState(raw)) {
    case 'available':
      return 'Ready'
    case 'unavailable':
      return 'Unavailable'
    case 'stale':
      return 'Stale'
    default:
      return 'Unknown'
  }
}

export function providerAvailabilityBadgeVariant(raw: string | null | undefined): BadgeVariant {
  switch (normalizeProviderAvailabilityState(raw)) {
    case 'available':
      return 'secondary'
    case 'unavailable':
      return 'destructive'
    case 'stale':
      return 'outline'
    default:
      return 'ghost'
  }
}

export function providerIsDispatchReady(raw: string | null | undefined): boolean {
  return normalizeProviderAvailabilityState(raw) === 'available'
}

export function providerAvailabilityHeadline(
  state: string | null | undefined,
  reason: string | null | undefined,
): string {
  switch (reason) {
    case 'machine_offline':
      return 'Host machine is offline.'
    case 'machine_degraded':
      return 'Host machine is degraded.'
    case 'machine_maintenance':
      return 'Host machine is in maintenance mode.'
    case 'l4_snapshot_missing':
      return 'No environment snapshot is available yet.'
    case 'stale_l4_snapshot':
      return 'Environment snapshot is stale.'
    case 'cli_missing':
      return 'Provider CLI is missing.'
    case 'not_logged_in':
      return 'Provider authentication is not ready.'
    case 'not_ready':
      return 'Provider is not launch-ready.'
    case 'config_incomplete':
      return 'Provider configuration is incomplete.'
    case 'unsupported_adapter':
      return 'Provider adapter is unsupported.'
    default:
      switch (normalizeProviderAvailabilityState(state)) {
        case 'available':
          return 'Provider is ready for dispatch.'
        case 'stale':
          return 'Provider health data is stale.'
        case 'unavailable':
          return 'Provider is not dispatch-ready.'
        default:
          return 'Provider health is still unknown.'
      }
  }
}

export function providerAvailabilityDescription(
  state: string | null | undefined,
  reason: string | null | undefined,
): string {
  switch (reason) {
    case 'machine_offline':
      return 'OpenASE cannot reach the bound machine, so this provider is blocked from scheduling.'
    case 'machine_degraded':
      return 'The bound machine is reachable but degraded, so provider scheduling is held back.'
    case 'machine_maintenance':
      return 'The bound machine is explicitly in maintenance mode and will not accept work.'
    case 'l4_snapshot_missing':
      return 'The machine has not produced a usable L4 agent environment snapshot for this provider yet.'
    case 'stale_l4_snapshot':
      return 'The last L4 agent environment snapshot is too old to trust for scheduling decisions.'
    case 'cli_missing':
      return 'The expected provider CLI is not installed or was not detected on the bound machine.'
    case 'not_logged_in':
      return 'The provider CLI exists, but authentication has not been completed or is no longer valid.'
    case 'not_ready':
      return 'The provider CLI was found, but its readiness probe still reports that it cannot launch work.'
    case 'config_incomplete':
      return 'Required launch configuration is missing, such as the CLI command or remote workspace root.'
    case 'unsupported_adapter':
      return 'OpenASE does not have an availability probe contract for this provider adapter type yet.'
    default:
      switch (normalizeProviderAvailabilityState(state)) {
        case 'available':
          return 'The latest machine and provider readiness checks passed, so this provider can take work.'
        case 'stale':
          return 'OpenASE has health data for this provider, but it is too old to trust until refreshed.'
        case 'unavailable':
          return 'OpenASE has recent health data for this provider and it is currently blocked from taking work.'
        default:
          return 'OpenASE does not yet have enough recent machine health data to judge this provider.'
      }
  }
}

export function providerAvailabilityCheckedAtText(raw: string | null | undefined): string | null {
  if (!raw) {
    return null
  }

  const value = new Date(raw)
  if (Number.isNaN(value.getTime())) {
    return null
  }

  const year = value.getUTCFullYear()
  const month = String(value.getUTCMonth() + 1).padStart(2, '0')
  const day = String(value.getUTCDate()).padStart(2, '0')
  const hours = String(value.getUTCHours()).padStart(2, '0')
  const minutes = String(value.getUTCMinutes()).padStart(2, '0')

  return `${year}-${month}-${day} ${hours}:${minutes} UTC`
}
