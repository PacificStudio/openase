import { describe, expect, it } from 'vitest'

import { resolveSettingsSectionHash, settingsSections } from './types'

describe('settings sections', () => {
  it('does not expose the removed access tab', () => {
    expect(settingsSections).not.toContain('access')
  })

  it('redirects legacy access hashes to security', () => {
    expect(resolveSettingsSectionHash('#access', 'general')).toBe('security')
    expect(resolveSettingsSectionHash('access', 'general')).toBe('security')
  })

  it('keeps current sections and invalid hashes stable', () => {
    expect(resolveSettingsSectionHash('#security', 'general')).toBe('security')
    expect(resolveSettingsSectionHash('#unknown', 'general')).toBe('general')
  })
})
