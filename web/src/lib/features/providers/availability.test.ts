import { describe, expect, it } from 'vitest'

import {
  providerAvailabilityDescription,
  providerAvailabilityHeadline,
  providerIsDispatchReady,
} from './availability'

describe('provider availability messaging', () => {
  it('treats machine maintenance as manually blocking dispatch', () => {
    expect(providerIsDispatchReady('unavailable')).toBe(false)
    expect(providerAvailabilityHeadline('unavailable', 'machine_maintenance')).toBe(
      'Machine in maintenance',
    )
    expect(providerAvailabilityDescription('unavailable', 'machine_maintenance')).toContain(
      'manually in maintenance',
    )
  })
})
