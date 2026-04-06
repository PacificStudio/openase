import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import { appStore } from '$lib/stores/app.svelte'
import SecuritySettings from './security-settings.svelte'
import {
  configuredSecurity,
  configuredSecurityWithNullPermissions,
  currentOrg,
  currentProject,
} from './security-settings.test-helpers'

const {
  deleteGitHubOutboundCredential,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  retestGitHubOutboundCredential,
  saveGitHubOutboundCredential,
} = vi.hoisted(() => ({
  deleteGitHubOutboundCredential: vi.fn(),
  getSecuritySettings: vi.fn(),
  importGitHubOutboundCredentialFromGHCLI: vi.fn(),
  retestGitHubOutboundCredential: vi.fn(),
  saveGitHubOutboundCredential: vi.fn(),
}))

const {
  createInstanceRoleBinding,
  createOrganizationRoleBinding,
  createProjectRoleBinding,
  deleteInstanceRoleBinding,
  deleteOrganizationRoleBinding,
  deleteProjectRoleBinding,
  getEffectivePermissions,
  listInstanceRoleBindings,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
} = vi.hoisted(() => ({
  createInstanceRoleBinding: vi.fn(),
  createOrganizationRoleBinding: vi.fn(),
  createProjectRoleBinding: vi.fn(),
  deleteInstanceRoleBinding: vi.fn(),
  deleteOrganizationRoleBinding: vi.fn(),
  deleteProjectRoleBinding: vi.fn(),
  getEffectivePermissions: vi.fn(),
  listInstanceRoleBindings: vi.fn(),
  listOrganizationRoleBindings: vi.fn(),
  listProjectRoleBindings: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  deleteGitHubOutboundCredential,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  retestGitHubOutboundCredential,
  saveGitHubOutboundCredential,
}))

vi.mock('$lib/api/auth', () => ({
  createInstanceRoleBinding,
  createOrganizationRoleBinding,
  createProjectRoleBinding,
  deleteInstanceRoleBinding,
  deleteOrganizationRoleBinding,
  deleteProjectRoleBinding,
  getEffectivePermissions,
  listInstanceRoleBindings,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
}))

describe('Security settings', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders the GitHub control plane alongside runtime boundaries', async () => {
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })

    const { findByText } = render(SecuritySettings)

    expect(await findByText('GitHub outbound credentials')).toBeTruthy()
    expect(await findByText('Effective credential')).toBeTruthy()
    expect(await findByText('Organization default')).toBeTruthy()
    expect(await findByText('Project override')).toBeTruthy()
    expect(await findByText('User @octocat')).toBeTruthy()
    expect(await findByText('Device Flow')).toBeTruthy()
    expect(await findByText('OPENASE_AGENT_TOKEN')).toBeTruthy()
  })

  it('saves a project override token from the settings surface', async () => {
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
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
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
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
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({
      security: configuredSecurityWithNullPermissions() as never,
    })

    const { findByText } = render(SecuritySettings)

    expect(await findByText('GitHub outbound credentials')).toBeTruthy()
    expect(await findByText('No scopes reported')).toBeTruthy()
  })

  it('renders oidc principal state and creates an organization role binding', async () => {
    authStore.hydrate({
      authMode: 'oidc',
      authenticated: true,
      issuerURL: 'https://idp.example.com',
      csrfToken: 'csrf-token',
      user: {
        id: 'user-1',
        primaryEmail: 'alice@example.com',
        displayName: 'Alice Control Plane',
      },
      roles: ['instance_admin'],
      permissions: ['org.update'],
    })
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    getEffectivePermissions.mockImplementation(async ({ orgId, projectId }) => {
      if (!orgId && !projectId) {
        return {
          user: {
            id: 'user-1',
            primary_email: 'alice@example.com',
            display_name: 'Alice Control Plane',
          },
          scope: { kind: 'instance', id: '' },
          roles: ['instance_admin'],
          permissions: ['rbac.manage'],
          groups: [{ group_key: 'platform-admins', group_name: 'Platform Admins', issuer: 'oidc' }],
        }
      }
      if (orgId) {
        return {
          user: {
            id: 'user-1',
            primary_email: 'alice@example.com',
            display_name: 'Alice Control Plane',
          },
          scope: { kind: 'organization', id: orgId },
          roles: ['org_admin'],
          permissions: ['org.read', 'rbac.manage'],
          groups: [{ group_key: 'platform-admins', group_name: 'Platform Admins', issuer: 'oidc' }],
        }
      }
      return {
        user: {
          id: 'user-1',
          primary_email: 'alice@example.com',
          display_name: 'Alice Control Plane',
        },
        scope: { kind: 'project', id: projectId ?? '' },
        roles: ['project_admin'],
        permissions: ['project.read', 'rbac.manage'],
        groups: [{ group_key: 'platform-admins', group_name: 'Platform Admins', issuer: 'oidc' }],
      }
    })
    listInstanceRoleBindings.mockResolvedValue([])
    listOrganizationRoleBindings.mockResolvedValue([
      {
        id: 'binding-1',
        scopeKind: 'organization',
        scopeID: currentOrg().id,
        subjectKind: 'group',
        subjectKey: 'platform-admins',
        roleKey: 'org_admin',
        grantedBy: 'user:user-1',
        createdAt: '2026-04-04T09:00:00Z',
      },
    ])
    listProjectRoleBindings.mockResolvedValue([])
    createOrganizationRoleBinding.mockResolvedValue({
      id: 'binding-2',
      scopeKind: 'organization',
      scopeID: currentOrg().id,
      subjectKind: 'user',
      subjectKey: 'bob@example.com',
      roleKey: 'org_member',
      grantedBy: 'user:user-1',
      createdAt: '2026-04-04T10:00:00Z',
    })

    const { findAllByPlaceholderText, findByText } = render(SecuritySettings)

    expect(await findByText('Human access and RBAC')).toBeTruthy()
    expect(await findByText('Alice Control Plane')).toBeTruthy()
    expect(await findByText('alice@example.com')).toBeTruthy()
    expect(await findByText('Instance effective access')).toBeTruthy()
    expect(await findByText('Platform Admins')).toBeTruthy()
    expect(await findByText('org_admin')).toBeTruthy()
    expect(await findByText('project_admin')).toBeTruthy()
    expect(await findByText('Approval boundary')).toBeTruthy()
    expect(await findByText('Stored rules')).toBeTruthy()
    expect(await findByText('reserved')).toBeTruthy()

    const orgSectionTitle = await findByText('Organization RBAC')
    const orgSection = orgSectionTitle.closest('.border-border') as HTMLElement
    const subjectInputs = await findAllByPlaceholderText('user@example.com')
    const orgInput = subjectInputs.find((element) => orgSection.contains(element as Node))
    expect(orgInput).toBeTruthy()
    await fireEvent.input(orgInput as HTMLElement, { target: { value: 'bob@example.com' } })

    const addButton = within(orgSection).getByRole('button', { name: 'Add binding' })
    await fireEvent.click(addButton)

    await waitFor(() => {
      expect(createOrganizationRoleBinding).toHaveBeenCalledWith(currentOrg().id, {
        subject_kind: 'user',
        subject_key: 'bob@example.com',
        role_key: 'org_member',
        expires_at: undefined,
      })
    })
  })

  it('filters role picker options by scope, including instance bindings', async () => {
    authStore.hydrate({
      authMode: 'oidc',
      authenticated: true,
      issuerURL: 'https://idp.example.com',
      csrfToken: 'csrf-token',
      user: {
        id: 'user-1',
        primaryEmail: 'alice@example.com',
        displayName: 'Alice Control Plane',
      },
      roles: ['instance_admin'],
      permissions: ['org.update'],
    })
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    getEffectivePermissions.mockImplementation(async ({ orgId, projectId }) => ({
      user: {
        id: 'user-1',
        primary_email: 'alice@example.com',
        display_name: 'Alice Control Plane',
      },
      scope: { kind: orgId ? 'organization' : projectId ? 'project' : 'instance', id: orgId ?? projectId ?? '' },
      roles: ['instance_admin'],
      permissions: ['rbac.manage'],
      groups: [],
    }))
    listInstanceRoleBindings.mockResolvedValue([])
    listOrganizationRoleBindings.mockResolvedValue([])
    listProjectRoleBindings.mockResolvedValue([])

    const { findByText } = render(SecuritySettings)

    const instanceSection = (await findByText('Instance RBAC')).closest('.border-border') as HTMLElement
    const organizationSection = (await findByText('Organization RBAC')).closest('.border-border') as HTMLElement
    const projectSection = (await findByText('Project RBAC')).closest('.border-border') as HTMLElement

    const instanceRoleSelect = within(instanceSection).getAllByRole('combobox')[1] as HTMLSelectElement
    const organizationRoleSelect = within(organizationSection).getAllByRole('combobox')[1] as HTMLSelectElement
    const projectRoleSelect = within(projectSection).getAllByRole('combobox')[1] as HTMLSelectElement

    const instanceRoleOptions = Array.from(instanceRoleSelect.options).map((option) => option.value)
    const organizationRoleOptions = Array.from(organizationRoleSelect.options).map((option) => option.value)
    const projectRoleOptions = Array.from(projectRoleSelect.options).map((option) => option.value)

    expect(instanceRoleOptions).toEqual(['instance_admin'])
    expect(organizationRoleOptions).toEqual(['org_owner', 'org_admin', 'org_member'])
    expect(projectRoleOptions).toEqual([
      'project_admin',
      'project_operator',
      'project_reviewer',
      'project_member',
      'project_viewer',
    ])
  })
})
