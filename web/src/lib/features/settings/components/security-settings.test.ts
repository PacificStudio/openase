import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import { appStore } from '$lib/stores/app.svelte'
import SecuritySettings from './security-settings.svelte'
import {
  createdOrganizationUserBinding,
  configuredSecurity,
  configuredSessionGovernance,
  configuredSecurityWithNullPermissions,
  currentOrg,
  currentProject,
  effectivePermissionsMock,
  hydrateOidcAuth,
  mockEffectivePermissionsByScope,
  organizationGroupBinding,
} from './security-settings.test-helpers'

const {
  deleteGitHubOutboundCredential,
  getSecuritySettings,
  getSessionGovernance,
  importGitHubOutboundCredentialFromGHCLI,
  revokeAllOtherAuthSessions,
  revokeAuthSession,
  retestGitHubOutboundCredential,
  saveGitHubOutboundCredential,
} = vi.hoisted(() => ({
  deleteGitHubOutboundCredential: vi.fn(),
  getSecuritySettings: vi.fn(),
  getSessionGovernance: vi.fn(),
  importGitHubOutboundCredentialFromGHCLI: vi.fn(),
  revokeAllOtherAuthSessions: vi.fn(),
  revokeAuthSession: vi.fn(),
  retestGitHubOutboundCredential: vi.fn(),
  saveGitHubOutboundCredential: vi.fn(),
}))

const {
  createInstanceRoleBinding,
  createOrganizationRoleBinding,
  createProjectRoleBinding,
  adminRevokeUserAuthSessions,
  deleteInstanceRoleBinding,
  deleteOrganizationRoleBinding,
  deleteProjectRoleBinding,
  getInstanceUserDetail,
  getEffectivePermissions,
  listInstanceRoleBindings,
  listInstanceUsers,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
  logoutHumanSession,
  transitionInstanceUserStatus,
} = vi.hoisted(() => ({
  createInstanceRoleBinding: vi.fn(),
  createOrganizationRoleBinding: vi.fn(),
  createProjectRoleBinding: vi.fn(),
  adminRevokeUserAuthSessions: vi.fn(),
  deleteInstanceRoleBinding: vi.fn(),
  deleteOrganizationRoleBinding: vi.fn(),
  deleteProjectRoleBinding: vi.fn(),
  getInstanceUserDetail: vi.fn(),
  getEffectivePermissions: vi.fn(),
  listInstanceRoleBindings: vi.fn(),
  listInstanceUsers: vi.fn(),
  listOrganizationRoleBindings: vi.fn(),
  listProjectRoleBindings: vi.fn(),
  logoutHumanSession: vi.fn(),
  transitionInstanceUserStatus: vi.fn(),
}))

const { goto } = vi.hoisted(() => ({
  goto: vi.fn(),
}))

vi.mock('$app/navigation', () => ({
  goto,
}))

vi.mock('$lib/api/openase', () => ({
  deleteGitHubOutboundCredential,
  getSecuritySettings,
  getSessionGovernance,
  importGitHubOutboundCredentialFromGHCLI,
  revokeAllOtherAuthSessions,
  revokeAuthSession,
  retestGitHubOutboundCredential,
  saveGitHubOutboundCredential,
}))

vi.mock('$lib/api/auth', () => ({
  createInstanceRoleBinding,
  createOrganizationRoleBinding,
  createProjectRoleBinding,
  adminRevokeUserAuthSessions,
  deleteInstanceRoleBinding,
  deleteOrganizationRoleBinding,
  deleteProjectRoleBinding,
  getInstanceUserDetail,
  getEffectivePermissions,
  listInstanceUsers,
  listInstanceRoleBindings,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
  logoutHumanSession,
  transitionInstanceUserStatus,
  normalizeReturnTo: vi.fn((value?: string | null) => value?.trim() || '/'),
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
    hydrateOidcAuth()
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    getEffectivePermissions.mockImplementation(mockEffectivePermissionsByScope)
    listInstanceRoleBindings.mockResolvedValue([])
    listOrganizationRoleBindings.mockResolvedValue([organizationGroupBinding()])
    listProjectRoleBindings.mockResolvedValue([])
    getSessionGovernance.mockResolvedValue(configuredSessionGovernance())
    createOrganizationRoleBinding.mockResolvedValue(createdOrganizationUserBinding())
    listInstanceUsers.mockResolvedValue([
      {
        id: 'user-2',
        status: 'active',
        primaryEmail: 'bob@example.com',
        displayName: 'Bob Reviewer',
        avatarURL: '',
        lastLoginAt: '2026-04-05T10:00:00Z',
        createdAt: '2026-04-05T09:00:00Z',
        updatedAt: '2026-04-05T10:00:00Z',
        primaryIdentity: {
          id: 'identity-1',
          issuer: 'https://idp.example.com',
          subject: 'subject-bob',
          email: 'bob@example.com',
          emailVerified: true,
          lastSyncedAt: '2026-04-05T10:00:00Z',
        },
      },
    ])
    getInstanceUserDetail.mockResolvedValue({
      user: {
        id: 'user-2',
        status: 'active',
        primaryEmail: 'bob@example.com',
        displayName: 'Bob Reviewer',
        avatarURL: '',
        lastLoginAt: '2026-04-05T10:00:00Z',
        createdAt: '2026-04-05T09:00:00Z',
        updatedAt: '2026-04-05T10:00:00Z',
        primaryIdentity: {
          id: 'identity-1',
          issuer: 'https://idp.example.com',
          subject: 'subject-bob',
          email: 'bob@example.com',
          emailVerified: true,
          lastSyncedAt: '2026-04-05T10:00:00Z',
        },
      },
      identities: [
        {
          id: 'identity-1',
          issuer: 'https://idp.example.com',
          subject: 'subject-bob',
          email: 'bob@example.com',
          emailVerified: true,
          lastSyncedAt: '2026-04-05T10:00:00Z',
          claimsVersion: 4,
          rawClaimsJSON: '{}',
          createdAt: '2026-04-05T09:00:00Z',
          updatedAt: '2026-04-05T10:00:00Z',
        },
      ],
      groups: [],
      activeSessionCount: 1,
      latestStatusAudit: undefined,
      recentAuditEvents: [],
    })

    const { findAllByPlaceholderText, findAllByText, findByText } = render(SecuritySettings)

    expect(await findByText('Human access and RBAC')).toBeTruthy()
    expect(await findByText('Alice Control Plane')).toBeTruthy()
    expect(await findByText('alice@example.com')).toBeTruthy()
    expect(await findByText('Instance effective access')).toBeTruthy()
    expect(await findByText('Platform Admins')).toBeTruthy()
    expect(await findByText('org_admin')).toBeTruthy()
    expect(await findByText('project_admin')).toBeTruthy()
    expect(await findByText('Approval boundary')).toBeTruthy()
    expect(await findByText('Session governance')).toBeTruthy()
    expect(await findByText('User directory and deprovision')).toBeTruthy()
    expect(await findByText('Bob Reviewer')).toBeTruthy()
    expect((await findAllByText('Firefox on Linux')).length).toBeGreaterThan(0)
    expect(await findByText('Login succeeded')).toBeTruthy()
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
    hydrateOidcAuth()
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    getEffectivePermissions.mockImplementation(async ({ orgId, projectId }) =>
      effectivePermissionsMock(
        orgId ? 'organization' : projectId ? 'project' : 'instance',
        orgId ?? projectId ?? '',
      ),
    )
    listInstanceRoleBindings.mockResolvedValue([])
    listOrganizationRoleBindings.mockResolvedValue([])
    listProjectRoleBindings.mockResolvedValue([])
    listInstanceUsers.mockResolvedValue([])
    getInstanceUserDetail.mockResolvedValue({
      user: {
        id: '',
        status: 'active',
        primaryEmail: '',
        displayName: '',
        avatarURL: '',
        createdAt: '',
        updatedAt: '',
      },
      identities: [],
      groups: [],
      activeSessionCount: 0,
      recentAuditEvents: [],
    })

    const { findByText } = render(SecuritySettings)

    const instanceSection = (await findByText('Instance RBAC')).closest(
      '.border-border',
    ) as HTMLElement
    const organizationSection = (await findByText('Organization RBAC')).closest(
      '.border-border',
    ) as HTMLElement
    const projectSection = (await findByText('Project RBAC')).closest(
      '.border-border',
    ) as HTMLElement

    const instanceRoleSelect = within(instanceSection).getAllByRole(
      'combobox',
    )[1] as HTMLSelectElement
    const organizationRoleSelect = within(organizationSection).getAllByRole(
      'combobox',
    )[1] as HTMLSelectElement
    const projectRoleSelect = within(projectSection).getAllByRole(
      'combobox',
    )[1] as HTMLSelectElement

    const instanceRoleOptions = Array.from(instanceRoleSelect.options).map((option) => option.value)
    const organizationRoleOptions = Array.from(organizationRoleSelect.options).map(
      (option) => option.value,
    )
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

  it('disables a cached user from the directory with an audit reason', async () => {
    hydrateOidcAuth()
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    getEffectivePermissions.mockImplementation(mockEffectivePermissionsByScope)
    listInstanceRoleBindings.mockResolvedValue([])
    listOrganizationRoleBindings.mockResolvedValue([])
    listProjectRoleBindings.mockResolvedValue([])
    getSessionGovernance.mockResolvedValue(configuredSessionGovernance())
    listInstanceUsers.mockResolvedValue([
      {
        id: 'user-2',
        status: 'active',
        primaryEmail: 'bob@example.com',
        displayName: 'Bob Reviewer',
        avatarURL: '',
        lastLoginAt: '2026-04-05T10:00:00Z',
        createdAt: '2026-04-05T09:00:00Z',
        updatedAt: '2026-04-05T10:00:00Z',
      },
    ])
    getInstanceUserDetail.mockResolvedValue({
      user: {
        id: 'user-2',
        status: 'active',
        primaryEmail: 'bob@example.com',
        displayName: 'Bob Reviewer',
        avatarURL: '',
        lastLoginAt: '2026-04-05T10:00:00Z',
        createdAt: '2026-04-05T09:00:00Z',
        updatedAt: '2026-04-05T10:00:00Z',
      },
      identities: [],
      groups: [],
      activeSessionCount: 1,
      recentAuditEvents: [],
    })
    transitionInstanceUserStatus.mockResolvedValue({
      user: {
        id: 'user-2',
        status: 'disabled',
        primaryEmail: 'bob@example.com',
        displayName: 'Bob Reviewer',
        avatarURL: '',
        lastLoginAt: '2026-04-05T10:00:00Z',
        createdAt: '2026-04-05T09:00:00Z',
        updatedAt: '2026-04-06T10:00:00Z',
      },
      changed: true,
      revokedSessionCount: 1,
      latestStatusAudit: {
        status: 'disabled',
        reason: 'Left the organization',
        source: 'admin_manual',
        actorID: 'user:user-1',
        changedAt: '2026-04-06T10:00:00Z',
        revokedSessionCount: 1,
      },
    })

    const { findByPlaceholderText, findByRole } = render(SecuritySettings)
    const reasonInput = await findByPlaceholderText(
      'Document the lifecycle reason for audit and future review',
    )
    await fireEvent.input(reasonInput, { target: { value: 'Left the organization' } })

    const disableButton = await findByRole('button', { name: 'Disable and revoke sessions' })
    await fireEvent.click(disableButton)

    await waitFor(() => {
      expect(transitionInstanceUserStatus).toHaveBeenCalledWith('user-2', {
        status: 'disabled',
        reason: 'Left the organization',
      })
    })
  })
})
