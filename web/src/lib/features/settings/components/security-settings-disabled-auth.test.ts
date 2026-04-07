import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import { appStore } from '$lib/stores/app.svelte'
import SecuritySettings from './security-settings.svelte'
import { currentOrg, currentProject, disabledSecurity } from './security-settings.test-helpers'

const { getSecuritySettings } = vi.hoisted(() => ({
  getSecuritySettings: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  getSecuritySettings,
}))

vi.mock('$lib/api/auth', () => ({
  createProjectRoleBinding: vi.fn(),
  deleteProjectRoleBinding: vi.fn(),
  getEffectivePermissions: vi.fn(),
  listProjectRoleBindings: vi.fn(),
}))

describe('Security settings disabled auth split', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('keeps project security local while pointing instance and org governance to new control planes', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: disabledSecurity() })

    const { findByText, queryByText } = render(SecuritySettings)

    expect(await findByText('Instance auth moved to `/admin`')).toBeTruthy()
    expect(await findByText('Org member governance moved to org admin')).toBeTruthy()
    expect(await findByText(/Disabled mode keeps project collaboration single-user\./)).toBeTruthy()
    expect(await findByText('GitHub outbound credentials')).toBeTruthy()
    expect(queryByText('Auth setup')).toBeNull()
  })
})
