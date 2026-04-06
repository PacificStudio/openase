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
  disabledSecurity,
  effectivePermissionsMock,
  hydrateOidcAuth,
} from './security-settings.test-helpers'

const {
  deleteGitHubOutboundCredential,
  enableOIDC,
  getSessionGovernance,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  revokeAllOtherAuthSessions,
  revokeAuthSession,
  retestGitHubOutboundCredential,
  saveOIDCDraft,
  saveGitHubOutboundCredential,
  testOIDCDraft,
} = vi.hoisted(() => ({
  deleteGitHubOutboundCredential: vi.fn(),
  enableOIDC: vi.fn(),
  getSessionGovernance: vi.fn(),
  getSecuritySettings: vi.fn(),
  importGitHubOutboundCredentialFromGHCLI: vi.fn(),
  revokeAllOtherAuthSessions: vi.fn(),
  revokeAuthSession: vi.fn(),
  retestGitHubOutboundCredential: vi.fn(),
  saveOIDCDraft: vi.fn(),
  saveGitHubOutboundCredential: vi.fn(),
  testOIDCDraft: vi.fn(),
}))

const {
  createInstanceRoleBinding,
  createOrganizationRoleBinding,
  createProjectRoleBinding,
  inviteOrganizationMember,
  deleteInstanceRoleBinding,
  deleteOrganizationRoleBinding,
  deleteProjectRoleBinding,
  getInstanceUserDetail,
  getEffectivePermissions,
  listOrganizationMemberships,
  listInstanceRoleBindings,
  listInstanceUsers,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
  logoutHumanSession,
  cancelOrganizationInvitation,
  resendOrganizationInvitation,
  transferOrganizationOwnership,
  updateOrganizationMembership,
} = vi.hoisted(() => ({
  createInstanceRoleBinding: vi.fn(),
  createOrganizationRoleBinding: vi.fn(),
  createProjectRoleBinding: vi.fn(),
  inviteOrganizationMember: vi.fn(),
  deleteInstanceRoleBinding: vi.fn(),
  deleteOrganizationRoleBinding: vi.fn(),
  deleteProjectRoleBinding: vi.fn(),
  getInstanceUserDetail: vi.fn(),
  getEffectivePermissions: vi.fn(),
  listOrganizationMemberships: vi.fn(),
  listInstanceRoleBindings: vi.fn(),
  listInstanceUsers: vi.fn(),
  listOrganizationRoleBindings: vi.fn(),
  listProjectRoleBindings: vi.fn(),
  logoutHumanSession: vi.fn(),
  cancelOrganizationInvitation: vi.fn(),
  resendOrganizationInvitation: vi.fn(),
  transferOrganizationOwnership: vi.fn(),
  updateOrganizationMembership: vi.fn(),
}))

const { goto, invalidateAll } = vi.hoisted(() => ({
  goto: vi.fn(),
  invalidateAll: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('$app/navigation', () => ({
  goto,
  invalidateAll,
}))

vi.mock('$lib/api/openase', () => ({
  deleteGitHubOutboundCredential,
  enableOIDC,
  getSessionGovernance,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  revokeAllOtherAuthSessions,
  revokeAuthSession,
  retestGitHubOutboundCredential,
  saveOIDCDraft,
  saveGitHubOutboundCredential,
  testOIDCDraft,
}))

vi.mock('$lib/api/auth', () => ({
  createInstanceRoleBinding,
  createOrganizationRoleBinding,
  createProjectRoleBinding,
  inviteOrganizationMember,
  deleteInstanceRoleBinding,
  deleteOrganizationRoleBinding,
  deleteProjectRoleBinding,
  getInstanceUserDetail,
  getEffectivePermissions,
  listOrganizationMemberships,
  listInstanceUsers,
  listInstanceRoleBindings,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
  logoutHumanSession,
  normalizeReturnTo: vi.fn((value?: string | null) => value?.trim() || '/'),
  cancelOrganizationInvitation,
  resendOrganizationInvitation,
  transferOrganizationOwnership,
  updateOrganizationMembership,
}))

describe('Security settings', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
    invalidateAll.mockClear()
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

  it('filters role picker options by scope, including instance bindings', async () => {
    hydrateOidcAuth()
    appStore.currentOrg = currentOrg()
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: configuredSecurity() })
    getSessionGovernance.mockResolvedValue({
      authMode: 'oidc',
      currentSessionID: 'session-current',
      sessions: [],
      auditEvents: [],
      stepUp: {
        status: 'reserved',
        summary: 'Reserved for future high-risk actions.',
        supportedMethods: [],
      },
    })
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
    listOrganizationMemberships.mockResolvedValue([])
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

  it('renders disabled-mode auth setup guidance and rollout links', async () => {
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: disabledSecurity() })

    const { findByText, findByLabelText } = render(SecuritySettings)

    expect(await findByText('Auth setup')).toBeTruthy()
    expect(await findByText('Enable OIDC')).toBeTruthy()
    expect(await findByText('Mode switch help')).toBeTruthy()
    expect(await findByLabelText('Issuer URL')).toBeTruthy()
    expect(await findByText('Disabled mode keeps this local admin available.')).toBeTruthy()
  })

  it('keeps disabled mode active while saving/testing/enabling OIDC explicitly', async () => {
    appStore.currentProject = currentProject()
    getSecuritySettings.mockResolvedValue({ security: disabledSecurity() })
    saveOIDCDraft.mockResolvedValue({ security: disabledSecurity() })
    testOIDCDraft.mockResolvedValue({
      status: 'ok',
      message: 'OIDC discovery succeeded.',
      issuer_url: 'https://idp.example.com',
      authorization_endpoint: 'https://idp.example.com/auth',
      token_endpoint: 'https://idp.example.com/token',
      redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
      warnings: [],
    })
    enableOIDC.mockResolvedValue({
      activation: {
        status: 'configured',
        message: 'OIDC is configured and will activate after restart.',
        restart_required: true,
        next_steps: ['Restart OpenASE', 'Complete the first OIDC login'],
      },
      security: {
        ...disabledSecurity(),
        auth: {
          ...disabledSecurity().auth,
          configured_mode: 'oidc',
        },
      },
    })

    const { findByLabelText, findByRole, findByText } = render(SecuritySettings)

    await fireEvent.input(await findByLabelText('Issuer URL'), {
      target: { value: 'https://idp.example.com' },
    })
    await fireEvent.input(await findByLabelText('Client ID'), {
      target: { value: 'openase' },
    })
    await fireEvent.input(await findByLabelText('Redirect URL'), {
      target: { value: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback' },
    })

    await fireEvent.click(await findByRole('button', { name: 'Save draft' }))
    await waitFor(() => {
      expect(saveOIDCDraft).toHaveBeenCalledWith(currentProject().id, {
        issuer_url: 'https://idp.example.com',
        client_id: 'openase',
        client_secret: '',
        redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
        scopes: ['openid', 'profile', 'email', 'groups'],
        allowed_email_domains: ['example.com'],
        bootstrap_admin_emails: ['admin@example.com'],
      })
    })

    await fireEvent.click(await findByRole('button', { name: 'Test configuration' }))
    await waitFor(() => {
      expect(testOIDCDraft).toHaveBeenCalledTimes(1)
    })
    expect(await findByText('OIDC discovery succeeded.')).toBeTruthy()

    await fireEvent.click(await findByRole('button', { name: 'Enable OIDC' }))
    await waitFor(() => {
      expect(enableOIDC).toHaveBeenCalledTimes(1)
    })
    expect(await findByText('Restart required')).toBeTruthy()
    expect(await findByText('Configured: oidc')).toBeTruthy()
  })
})
