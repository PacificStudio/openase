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
