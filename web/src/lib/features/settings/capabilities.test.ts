import { describe, expect, it } from 'vitest'

import {
  capabilityCatalog,
  getSettingsSectionCapability,
  settingsCapabilityBySection,
} from '$lib/features/capabilities'
import { settingsSections } from './types'

describe('settings capability catalog', () => {
  it('classifies every shipped settings section', () => {
    expect(new Set(Object.keys(settingsCapabilityBySection))).toEqual(new Set(settingsSections))
  })

  it('routes statuses through the settings-scoped capability entry', () => {
    expect(getSettingsSectionCapability('statuses')).toBe(capabilityCatalog.statusesSettings)
  })

  it('marks workflow settings as available with shipped lifecycle management copy', () => {
    const workflowCapability = getSettingsSectionCapability('workflows')

    expect(workflowCapability.state).toBe('available')
    expect(workflowCapability.summary.toLowerCase()).not.toContain('placeholder')
  })

  it('marks security settings as available with explicit deferred scope copy', () => {
    const securityCapability = getSettingsSectionCapability('security')

    expect(securityCapability.state).toBe('available')
    expect(securityCapability.summary.toLowerCase()).not.toContain('placeholder')
    expect(securityCapability.summary.toLowerCase()).toContain('deferred')
  })
})
