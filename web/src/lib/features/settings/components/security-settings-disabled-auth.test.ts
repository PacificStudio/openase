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

describe('Security settings disabled auth migration', () => {
  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders migration guidance instead of the legacy auth setup form', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: disabledSecurity() })

    const { findByRole, findByText, queryByLabelText } = render(SecuritySettings)

    expect(await findByText('Migration note')).toBeTruthy()
    expect(await findByText('Active: disabled')).toBeTruthy()
    expect(await findByRole('link', { name: 'Open /admin/auth' })).toBeTruthy()
    expect(await findByRole('link', { name: 'Open org admin' })).toBeTruthy()
    expect(await findByRole('link', { name: 'Open Settings -> Access' })).toBeTruthy()
    expect(await findByText('GitHub outbound credentials')).toBeTruthy()
    expect(queryByLabelText('Issuer URL')).toBeNull()
  })
})
