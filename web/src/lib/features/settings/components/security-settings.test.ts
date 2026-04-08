import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import SecuritySettings from './security-settings.svelte'
import {
  configuredSecurity,
  configuredSecurityWithNullPermissions,
  currentOrg,
  currentProject,
} from './security-settings.test-helpers'

const {
  createProjectScopedSecret,
  deleteGitHubOutboundCredential,
  deleteProjectScopedSecret,
  disableProjectScopedSecret,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  listProjectScopedSecrets,
  rotateProjectScopedSecret,
  retestGitHubOutboundCredential,
  saveGitHubOutboundCredential,
} = vi.hoisted(() => ({
  createProjectScopedSecret: vi.fn(),
  deleteGitHubOutboundCredential: vi.fn(),
  deleteProjectScopedSecret: vi.fn(),
  disableProjectScopedSecret: vi.fn(),
  getSecuritySettings: vi.fn(),
  importGitHubOutboundCredentialFromGHCLI: vi.fn(),
  listProjectScopedSecrets: vi.fn(),
  rotateProjectScopedSecret: vi.fn(),
  retestGitHubOutboundCredential: vi.fn(),
  saveGitHubOutboundCredential: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createProjectScopedSecret,
  deleteGitHubOutboundCredential,
  deleteProjectScopedSecret,
  disableProjectScopedSecret,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  listProjectScopedSecrets,
  rotateProjectScopedSecret,
  retestGitHubOutboundCredential,
  saveGitHubOutboundCredential,
}))

describe('Security settings', () => {
  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders the migration panel before project-owned security controls', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    listProjectScopedSecrets.mockResolvedValue({
      secrets: [
        {
          id: 'secret-org',
          organization_id: 'org-1',
          project_id: null,
          scope: 'organization',
          name: 'OPENAI_API_KEY',
          kind: 'opaque',
          description: 'Inherited model key',
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
            value_preview: 'sk-live...1234',
          },
        },
      ],
    })

    const { findByRole, findByText } = render(SecuritySettings)

    expect(await findByText('Migration note')).toBeTruthy()
    expect(await findByText('Instance auth and directory')).toBeTruthy()
    expect(await findByText('Org members, invites, and roles')).toBeTruthy()
    expect(await findByText('Project access stays here')).toBeTruthy()
    expect(await findByRole('link', { name: 'Open /admin/auth' })).toBeTruthy()
    expect(await findByRole('link', { name: 'Open org admin' })).toBeTruthy()
    expect(await findByText('Scoped secrets')).toBeTruthy()
    expect(await findByText('Inherited organization defaults')).toBeTruthy()
    expect(await findByText('GitHub outbound credentials')).toBeTruthy()
    expect(await findByText('OPENASE_AGENT_TOKEN')).toBeTruthy()
  })

  it('saves a project override token from the security surface', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    listProjectScopedSecrets.mockResolvedValue({ secrets: [] })
    saveGitHubOutboundCredential.mockResolvedValue({ security: configuredSecurity() })

    const { findByPlaceholderText, findAllByRole } = render(SecuritySettings)

    const input = await findByPlaceholderText('ghu_xxx or github_pat_xxx')
    await fireEvent.input(input, { target: { value: 'ghu_project_override' } })

    const saveButtons = await findAllByRole('button', { name: 'Save' })
    await fireEvent.click(saveButtons[0])

    await waitFor(() => {
      expect(saveGitHubOutboundCredential).toHaveBeenCalledWith(appStore.currentProject?.id, {
        scope: 'project',
        token: 'ghu_project_override',
      })
    })
  })

  it('imports, retests, and deletes credentials through scoped actions', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    listProjectScopedSecrets.mockResolvedValue({ secrets: [] })
    importGitHubOutboundCredentialFromGHCLI.mockResolvedValue({ security: configuredSecurity() })
    retestGitHubOutboundCredential.mockResolvedValue({ security: configuredSecurity() })
    deleteGitHubOutboundCredential.mockResolvedValue({ security: configuredSecurity() })

    const { findAllByText, findAllByTitle } = render(SecuritySettings)

    const importButtons = await findAllByText('Import from gh')
    await fireEvent.click(importButtons[0])
    await waitFor(() => {
      expect(importGitHubOutboundCredentialFromGHCLI).toHaveBeenCalledWith(
        appStore.currentProject?.id,
        { scope: 'organization' },
      )
    })

    const retestButtons = await findAllByTitle('Retest')
    await fireEvent.click(retestButtons[0])
    await waitFor(() => {
      expect(retestGitHubOutboundCredential).toHaveBeenCalledWith(appStore.currentProject?.id, {
        scope: 'organization',
      })
    })

    const deleteButtons = await findAllByTitle('Delete')
    await fireEvent.click(deleteButtons[0])
    await waitFor(() => {
      expect(deleteGitHubOutboundCredential).toHaveBeenCalledWith(
        appStore.currentProject?.id,
        'organization',
      )
    })
  })

  it('normalizes null GitHub probe permissions so the page does not crash', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({
      security: configuredSecurityWithNullPermissions() as never,
    })
    listProjectScopedSecrets.mockResolvedValue({ secrets: [] })

    const { findByText } = render(SecuritySettings)

    expect(await findByText('GitHub outbound credentials')).toBeTruthy()
    expect(await findByText('No scopes reported')).toBeTruthy()
  })
})
