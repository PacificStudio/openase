import type { BadgeVariant } from '$ui/badge'
import { providersT } from './i18n'

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
      return providersT('providers.availability.label.ready')
    case 'unavailable':
      return providersT('providers.availability.label.unavailable')
    case 'stale':
      return providersT('providers.availability.label.stale')
    default:
      return providersT('providers.availability.label.unknown')
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
      return providersT('providers.availability.headline.machineOffline')
    case 'machine_degraded':
      return providersT('providers.availability.headline.machineDegraded')
    case 'machine_maintenance':
      return providersT('providers.availability.headline.machineMaintenance')
    case 'l4_snapshot_missing':
      return providersT('providers.availability.headline.snapshotMissing')
    case 'stale_l4_snapshot':
      return providersT('providers.availability.headline.snapshotStale')
    case 'cli_missing':
      return providersT('providers.availability.headline.cliMissing')
    case 'not_logged_in':
      return providersT('providers.availability.headline.notLoggedIn')
    case 'not_ready':
      return providersT('providers.availability.headline.notReady')
    case 'config_incomplete':
      return providersT('providers.availability.headline.configIncomplete')
    case 'unsupported_adapter':
      return providersT('providers.availability.headline.unsupportedAdapter')
    default:
      switch (normalizeProviderAvailabilityState(state)) {
        case 'available':
          return providersT('providers.availability.headline.ready')
        case 'stale':
          return providersT('providers.availability.headline.healthStale')
        case 'unavailable':
          return providersT('providers.availability.headline.dispatchBlocked')
        default:
          return providersT('providers.availability.headline.unknown')
      }
  }
}

export function providerAvailabilityDescription(
  state: string | null | undefined,
  reason: string | null | undefined,
): string {
  switch (reason) {
    case 'machine_offline':
      return providersT('providers.availability.description.machineOffline')
    case 'machine_degraded':
      return providersT('providers.availability.description.machineDegraded')
    case 'machine_maintenance':
      return providersT('providers.availability.description.machineMaintenance')
    case 'l4_snapshot_missing':
      return providersT('providers.availability.description.snapshotMissing')
    case 'stale_l4_snapshot':
      return providersT('providers.availability.description.snapshotStale')
    case 'cli_missing':
      return providersT('providers.availability.description.cliMissing')
    case 'not_logged_in':
      return providersT('providers.availability.description.notLoggedIn')
    case 'not_ready':
      return providersT('providers.availability.description.notReady')
    case 'config_incomplete':
      return providersT('providers.availability.description.configIncomplete')
    case 'unsupported_adapter':
      return providersT('providers.availability.description.unsupportedAdapter')
    default:
      switch (normalizeProviderAvailabilityState(state)) {
        case 'available':
          return providersT('providers.availability.description.ready')
        case 'stale':
          return providersT('providers.availability.description.healthStale')
        case 'unavailable':
          return providersT('providers.availability.description.unavailable')
        default:
          return providersT('providers.availability.description.unknown')
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
