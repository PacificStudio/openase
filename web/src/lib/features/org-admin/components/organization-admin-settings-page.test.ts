import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import OrganizationAdminSettingsPage from './organization-admin-settings-page.svelte'

const {
  createOrganizationScopedSecret,
  deleteOrganizationScopedSecret,
  disableOrganizationScopedSecret,
  listOrganizationScopedSecrets,
  rotateOrganizationScopedSecret,
} = vi.hoisted(() => ({
  createOrganizationScopedSecret: vi.fn(),
  deleteOrganizationScopedSecret: vi.fn(),
  disableOrganizationScopedSecret: vi.fn(),
  listOrganizationScopedSecrets: vi.fn(),
  rotateOrganizationScopedSecret: vi.fn(),
}))

vi.mock('$lib/api/openase', async () => {
  const actual = await vi.importActual<typeof import('$lib/api/openase')>('$lib/api/openase')
  return {
    ...actual,
    createOrganizationScopedSecret,
    deleteOrganizationScopedSecret,
    disableOrganizationScopedSecret,
    listOrganizationScopedSecrets,
    rotateOrganizationScopedSecret,
  }
})

describe('OrganizationAdminSettingsPage', () => {
  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.providers = []
    vi.clearAllMocks()
  })

  it('renders the organization secret inventory alongside provider settings', async () => {
    appStore.currentOrg = {
      id: 'org-1',
      name: 'Acme',
      slug: 'acme',
      default_agent_provider_id: '',
      status: 'active',
    }
    listOrganizationScopedSecrets.mockResolvedValue({
      secrets: [
        {
          id: 'secret-1',
          organization_id: 'org-1',
          project_id: null,
          scope: 'organization',
          name: 'GH_TOKEN',
          kind: 'opaque',
          description: 'Shared automation token',
          disabled: false,
          disabled_at: null,
          created_at: '2026-04-08T12:00:00Z',
          updated_at: '2026-04-08T12:00:00Z',
          usage_count: 1,
          usage_scopes: ['organization'],
          encryption: {
            algorithm: 'aes-256-gcm',
            key_id: 'database-dsn-sha256:v1',
            key_source: 'database_dsn_sha256',
            rotated_at: '2026-04-08T12:00:00Z',
            value_preview: 'ghp_xxx...1234',
          },
        },
      ],
    })

    const view = render(OrganizationAdminSettingsPage, { organizationId: 'org-1' })

    expect(await view.findByText('Organization settings')).toBeTruthy()
    expect(await view.findByText('Organization secrets')).toBeTruthy()
    expect(await view.findByText('GH_TOKEN')).toBeTruthy()
    expect(listOrganizationScopedSecrets).toHaveBeenCalledWith('org-1')
  })
})
