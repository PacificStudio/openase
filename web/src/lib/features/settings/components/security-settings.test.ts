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
import {
  scopedSecretBindings,
  scopedSecrets,
  ticketCatalog,
  workflowCatalog,
} from './security-settings-secret-binding-test-helpers'

const {
  createScopedSecretBinding,
  deleteScopedSecretBinding,
  deleteGitHubOutboundCredential,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  listScopedSecretBindings,
  listScopedSecrets,
  listTickets,
  listWorkflows,
  retestGitHubOutboundCredential,
  saveGitHubOutboundCredential,
} = vi.hoisted(() => ({
  createScopedSecretBinding: vi.fn(),
  deleteScopedSecretBinding: vi.fn(),
  deleteGitHubOutboundCredential: vi.fn(),
  getSecuritySettings: vi.fn(),
  importGitHubOutboundCredentialFromGHCLI: vi.fn(),
  listScopedSecretBindings: vi.fn(),
  listScopedSecrets: vi.fn(),
  listTickets: vi.fn(),
  listWorkflows: vi.fn(),
  retestGitHubOutboundCredential: vi.fn(),
  saveGitHubOutboundCredential: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createScopedSecretBinding,
  deleteScopedSecretBinding,
  deleteGitHubOutboundCredential,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  listScopedSecretBindings,
  listScopedSecrets,
  listTickets,
  listWorkflows,
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

  function mockSecretBindingCatalog() {
    listScopedSecrets.mockResolvedValue({ secrets: scopedSecrets() })
    listScopedSecretBindings.mockResolvedValue({ bindings: scopedSecretBindings() })
    listWorkflows.mockResolvedValue({ workflows: workflowCatalog() })
    listTickets.mockResolvedValue({ tickets: ticketCatalog() })
  }

  it('renders the migration panel before project-owned security controls', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()

    const { findByRole, findByText } = render(SecuritySettings)

    expect(await findByText('Migration note')).toBeTruthy()
    expect(await findByText('Instance auth and directory')).toBeTruthy()
    expect(await findByText('Org members, invites, and roles')).toBeTruthy()
    expect(await findByText('Project access stays here')).toBeTruthy()
    expect(await findByRole('link', { name: 'Open /admin/auth' })).toBeTruthy()
    expect(await findByRole('link', { name: 'Open org admin' })).toBeTruthy()
    expect(await findByText('GitHub outbound credentials')).toBeTruthy()
    expect(await findByText('Runtime secret bindings')).toBeTruthy()
    expect(
      await findByText('Fullstack Developer Workflow', { selector: 'div.text-sm.font-medium' }),
    ).toBeTruthy()
    expect(await findByText('OPENASE_AGENT_TOKEN')).toBeTruthy()
  })

  it('saves a project override token from the security surface', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
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
    mockSecretBindingCatalog()
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
    mockSecretBindingCatalog()

    const { findByText } = render(SecuritySettings)

    expect(await findByText('GitHub outbound credentials')).toBeTruthy()
    expect(await findByText('No scopes reported')).toBeTruthy()
  })

  it('creates a workflow secret binding from the security surface', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
    listScopedSecretBindings.mockResolvedValue({ bindings: [] })
    createScopedSecretBinding.mockResolvedValue({
      binding: scopedSecretBindings()[0],
    })

    const { findByLabelText, findByRole } = render(SecuritySettings)

    await fireEvent.input(await findByLabelText('Binding key'), {
      target: { value: 'openai_api_key' },
    })
    await fireEvent.change(await findByLabelText('Workflow target'), {
      target: { value: 'workflow-fullstack' },
    })
    await fireEvent.change(await findByLabelText('Secret'), {
      target: { value: 'secret-project-openai' },
    })
    await fireEvent.click(await findByRole('button', { name: 'Create binding' }))

    await waitFor(() => {
      expect(createScopedSecretBinding).toHaveBeenCalledWith(appStore.currentProject?.id, {
        secret_id: 'secret-project-openai',
        scope: 'workflow',
        scope_resource_id: 'workflow-fullstack',
        binding_key: 'openai_api_key',
      })
    })
  })

  it('deletes a runtime secret binding from the security surface', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
    deleteScopedSecretBinding.mockResolvedValue({})

    const { findByTitle } = render(SecuritySettings)

    await fireEvent.click(await findByTitle('Delete binding'))

    await waitFor(() => {
      expect(deleteScopedSecretBinding).toHaveBeenCalledWith(
        appStore.currentProject?.id,
        'binding-1',
      )
    })
  })
})
