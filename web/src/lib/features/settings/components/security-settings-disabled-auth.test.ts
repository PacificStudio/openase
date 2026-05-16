import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import { appStore } from '$lib/stores/app.svelte'
import SecuritySettings from './security-settings.svelte'
import { currentOrg, currentProject, disabledSecurity } from './security-settings.test-helpers'

const {
  getSecuritySettings,
  listProjectScopedSecrets,
  listProjectUserAPIKeys,
  listScopedSecretBindings,
  listScopedSecrets,
  listTickets,
  listWorkflows,
} = vi.hoisted(() => ({
  getSecuritySettings: vi.fn(),
  listProjectScopedSecrets: vi.fn(),
  listProjectUserAPIKeys: vi.fn(),
  listScopedSecretBindings: vi.fn(),
  listScopedSecrets: vi.fn(),
  listTickets: vi.fn(),
  listWorkflows: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  getSecuritySettings,
  listProjectScopedSecrets,
  listProjectUserAPIKeys,
  listScopedSecretBindings,
  listScopedSecrets,
  listTickets,
  listWorkflows,
}))

describe('Security settings disabled auth', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('does not render the OIDC configuration form in disabled auth mode', async () => {
    authStore.hydrate({ authenticated: false, loginRequired: false })
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: disabledSecurity() })
    listProjectScopedSecrets.mockResolvedValue({ secrets: [] })
    listScopedSecrets.mockResolvedValue({ secrets: [] })
    listScopedSecretBindings.mockResolvedValue({ bindings: [] })
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue({ workflows: [] })
    listProjectUserAPIKeys.mockResolvedValue({ api_keys: [] })

    const { queryByLabelText } = render(SecuritySettings)

    expect(queryByLabelText('Issuer URL')).toBeNull()
  })
})
