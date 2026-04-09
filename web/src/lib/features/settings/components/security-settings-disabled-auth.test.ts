import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import SecuritySettings from './security-settings.svelte'
import { currentOrg, currentProject, disabledSecurity } from './security-settings.test-helpers'

const { getSecuritySettings } = vi.hoisted(() => ({
  getSecuritySettings: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  getSecuritySettings,
}))

describe('Security settings disabled auth', () => {
  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('does not render the OIDC configuration form in disabled auth mode', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: disabledSecurity() })

    const { findByText, queryByLabelText } = render(SecuritySettings)

    expect(await findByText('GitHub outbound credentials')).toBeTruthy()
    expect(queryByLabelText('Issuer URL')).toBeNull()
  })
})
