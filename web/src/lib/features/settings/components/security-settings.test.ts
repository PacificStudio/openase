import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import type { ScopedSecretRecord } from '$lib/api/contracts'
import { i18nStore } from '$lib/i18n/store.svelte'
import { appStore } from '$lib/stores/app.svelte'
import SecuritySettings from './security-settings.svelte'
import {
  configuredSecurity,
  configuredSecurityWithProjectGitHubOverride,
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
  createProjectScopedSecret,
  createScopedSecretBinding,
  deleteGitHubOutboundCredential,
  deleteProjectScopedSecret,
  deleteScopedSecretBinding,
  disableProjectScopedSecret,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  listProjectScopedSecrets,
  listScopedSecretBindings,
  listScopedSecrets,
  listTickets,
  listWorkflows,
  retestGitHubOutboundCredential,
  rotateProjectScopedSecret,
  saveGitHubOutboundCredential,
  updateProject,
} = vi.hoisted(() => ({
  createProjectScopedSecret: vi.fn(),
  createScopedSecretBinding: vi.fn(),
  deleteGitHubOutboundCredential: vi.fn(),
  deleteProjectScopedSecret: vi.fn(),
  deleteScopedSecretBinding: vi.fn(),
  disableProjectScopedSecret: vi.fn(),
  getSecuritySettings: vi.fn(),
  importGitHubOutboundCredentialFromGHCLI: vi.fn(),
  listProjectScopedSecrets: vi.fn(),
  listScopedSecretBindings: vi.fn(),
  listScopedSecrets: vi.fn(),
  listTickets: vi.fn(),
  listWorkflows: vi.fn(),
  retestGitHubOutboundCredential: vi.fn(),
  rotateProjectScopedSecret: vi.fn(),
  saveGitHubOutboundCredential: vi.fn(),
  updateProject: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createProjectScopedSecret,
  createScopedSecretBinding,
  deleteGitHubOutboundCredential,
  deleteProjectScopedSecret,
  deleteScopedSecretBinding,
  disableProjectScopedSecret,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  listProjectScopedSecrets,
  listScopedSecretBindings,
  listScopedSecrets,
  listTickets,
  listWorkflows,
  retestGitHubOutboundCredential,
  rotateProjectScopedSecret,
  saveGitHubOutboundCredential,
  updateProject,
}))

describe('Security settings', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
  })

  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.currentProject = null
    i18nStore.setLocale('en')
    vi.clearAllMocks()
  })

  function mockSecretBindingCatalog() {
    listScopedSecrets.mockResolvedValue({ secrets: scopedSecrets() })
    listScopedSecretBindings.mockResolvedValue({ bindings: scopedSecretBindings() })
    listWorkflows.mockResolvedValue({ workflows: workflowCatalog() })
    listTickets.mockResolvedValue({ tickets: ticketCatalog() })
  }

  function mockProjectScopedSecrets(secrets: ScopedSecretRecord[] = []) {
    listProjectScopedSecrets.mockResolvedValue({ secrets })
  }

  it('renders project-owned security controls', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
    mockProjectScopedSecrets([
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
    ])

    const { findByText } = render(SecuritySettings)

    expect(await findByText('Overview')).toBeTruthy()
    expect(await findByText('Inherited organization defaults')).toBeTruthy()
    expect(await findByText('GitHub outbound access')).toBeTruthy()
    expect(await findByText('Secret Bindings')).toBeTruthy()
    expect(await findByText('Project AI Platform')).toBeTruthy()
    expect(await findByText('Fullstack Developer Workflow')).toBeTruthy()
    expect(await findByText('OPENASE_AGENT_TOKEN')).toBeTruthy()
  })

  it('saves a project override token from the security surface', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
    mockProjectScopedSecrets()
    saveGitHubOutboundCredential.mockResolvedValue({ security: configuredSecurity() })

    const { findByPlaceholderText, findAllByRole } = render(SecuritySettings)

    const input = await findByPlaceholderText('Token')
    await fireEvent.input(input, { target: { value: 'ghu_project_override' } })

    const saveButtons = await findAllByRole('button', { name: 'Save' })
    await fireEvent.click(saveButtons[0])

    await waitFor(() => {
      expect(saveGitHubOutboundCredential).toHaveBeenCalledWith(appStore.currentProject?.id, {
        token: 'ghu_project_override',
      })
    })
  })

  it('saves project ai platform access from the security surface', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
    mockProjectScopedSecrets()
    updateProject.mockResolvedValue({
      project: {
        ...currentProject(),
        project_ai_platform_access_allowed: ['projects.update'],
      },
    })

    const { findAllByRole, findByText } = render(SecuritySettings)

    await fireEvent.click(await findByText('projects'))
    await fireEvent.click(await findByText('add_repo'))
    const saveButtons = await findAllByRole('button', { name: 'Save' })
    await fireEvent.click(saveButtons[saveButtons.length - 1]!)

    await waitFor(() => {
      expect(updateProject).toHaveBeenCalledWith(appStore.currentProject?.id, {
        project_ai_platform_access_allowed: [
          'project_updates.read',
          'project_updates.write',
          'projects.update',
        ],
      })
    })
    expect(appStore.currentProject?.project_ai_platform_access_allowed).toEqual(['projects.update'])
  })

  it('keeps project ai access explicitly empty after clearing all scopes', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
    mockProjectScopedSecrets()
    updateProject.mockResolvedValue({
      project: {
        ...currentProject(),
        project_ai_platform_access_allowed: [],
      },
    })

    const { findAllByRole, findByText } = render(SecuritySettings)

    await fireEvent.click(await findByText('projects'))
    await fireEvent.click(await findByText('update'))
    await fireEvent.click(await findByText('add_repo'))
    const saveButtons = await findAllByRole('button', { name: 'Save' })
    await fireEvent.click(saveButtons[saveButtons.length - 1]!)

    await waitFor(() => {
      expect(updateProject).toHaveBeenCalledWith(appStore.currentProject?.id, {
        project_ai_platform_access_allowed: ['project_updates.read', 'project_updates.write'],
      })
    })
    expect(appStore.currentProject?.project_ai_platform_access_allowed).toEqual([])
  })

  it('renders only backend-provided project ai scope groups', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
    mockProjectScopedSecrets()

    const { findByText, queryByText } = render(SecuritySettings)

    expect(await findByText('projects')).toBeTruthy()
    expect(queryByText('tickets')).toBeNull()
    expect(queryByText('update.self')).toBeNull()
  })

  it('imports, retests, and deletes credentials through scoped actions', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
    mockProjectScopedSecrets()
    const importedSecurity = configuredSecurityWithProjectGitHubOverride()
    importGitHubOutboundCredentialFromGHCLI.mockResolvedValue({ security: importedSecurity })
    retestGitHubOutboundCredential.mockResolvedValue({ security: importedSecurity })
    deleteGitHubOutboundCredential.mockResolvedValue({ security: configuredSecurity() })

    const { findAllByText, findAllByTitle } = render(SecuritySettings)

    const importButtons = await findAllByText('Import from gh')
    await fireEvent.click(importButtons[0])
    await waitFor(() => {
      expect(importGitHubOutboundCredentialFromGHCLI).toHaveBeenCalledWith(
        appStore.currentProject?.id,
      )
    })

    const retestButtons = await findAllByTitle('Retest')
    await fireEvent.click(retestButtons[0])
    await waitFor(() => {
      expect(retestGitHubOutboundCredential).toHaveBeenCalledWith(appStore.currentProject?.id)
    })

    const deleteButtons = await findAllByTitle('Delete')
    await fireEvent.click(deleteButtons[0])
    await waitFor(() => {
      expect(deleteGitHubOutboundCredential).toHaveBeenCalledWith(appStore.currentProject?.id)
    })
  })

  it('normalizes null GitHub probe permissions so the page does not crash', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({
      security: configuredSecurityWithNullPermissions() as never,
    })
    mockSecretBindingCatalog()
    mockProjectScopedSecrets()

    const { findByText, queryByText } = render(SecuritySettings)

    expect(await findByText('GitHub outbound access')).toBeTruthy()
    expect(await findByText('ghu_test...1234')).toBeTruthy()
    expect(queryByText('No scopes reported')).toBeNull()
  })

  it('renders natural Chinese security copy without exposing raw expressions', async () => {
    i18nStore.setLocale('zh')
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
    mockProjectScopedSecrets()

    const { container, findByText } = render(SecuritySettings)

    expect(await findByText('GitHub 出站访问')).toBeTruthy()
    expect(container.textContent).toContain(
      '待 OAuth 应用接线完成后，可通过 GitHub 设备验证码流程授权 CLI。',
    )
    expect(container.textContent).not.toContain('()=>')
    expect(container.textContent).not.toContain('m.t(')
    expect(container.textContent).not.toContain('github-device-flow')
  })

  it('creates a workflow secret binding from the security surface', async () => {
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    mockSecretBindingCatalog()
    listScopedSecrets.mockResolvedValue({ secrets: [scopedSecrets()[0]] })
    mockProjectScopedSecrets()
    listScopedSecretBindings.mockResolvedValue({ bindings: [] })
    createScopedSecretBinding.mockResolvedValue({
      binding: scopedSecretBindings()[0],
    })

    const { findAllByRole, findByLabelText, findByText } = render(SecuritySettings)

    const bindButtons = await findAllByRole('button', { name: 'Bind secret' })
    await fireEvent.click(bindButtons[bindButtons.length - 1])

    await fireEvent.input(await findByLabelText('Binding Key'), {
      target: { value: 'openai_api_key' },
    })
    const workflowTrigger = await findByText('Enter select workflow target...')
    workflowTrigger.focus()
    await fireEvent.keyDown(workflowTrigger, { key: 'ArrowDown' })
    await fireEvent.keyDown(workflowTrigger, { key: 'Enter' })
    const secretTrigger = await findByText('Enter select secret...')
    secretTrigger.focus()
    await fireEvent.keyDown(secretTrigger, { key: 'ArrowDown' })
    await fireEvent.keyDown(secretTrigger, { key: 'Enter' })
    const submitButtons = await findAllByRole('button', { name: 'Bind secret' })
    await fireEvent.click(submitButtons[submitButtons.length - 1])

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
    mockProjectScopedSecrets()
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
