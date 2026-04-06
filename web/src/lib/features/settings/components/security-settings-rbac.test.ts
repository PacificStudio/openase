import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import { authStore } from '$lib/stores/auth.svelte'
import SecuritySettings from './security-settings.svelte'
import {
  createdOrganizationUserBinding,
  configuredSecurity,
  configuredSessionGovernance,
  currentOrg,
  currentProject,
  hydrateOidcAuth,
  mockEffectivePermissionsByScope,
  organizationGroupBinding,
} from './security-settings.test-helpers'

const { getSecuritySettings, getSessionGovernance } = vi.hoisted(() => ({
  getSecuritySettings: vi.fn(),
  getSessionGovernance: vi.fn(),
}))

const {
  createOrganizationRoleBinding,
  getEffectivePermissions,
  getInstanceUserDetail,
  listInstanceRoleBindings,
  listInstanceUsers,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
} = vi.hoisted(() => ({
  createOrganizationRoleBinding: vi.fn(),
  getEffectivePermissions: vi.fn(),
  getInstanceUserDetail: vi.fn(),
  listInstanceRoleBindings: vi.fn(),
  listInstanceUsers: vi.fn(),
  listOrganizationRoleBindings: vi.fn(),
  listProjectRoleBindings: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  getSecuritySettings,
  getSessionGovernance,
}))

vi.mock('$lib/api/auth', () => ({
  createOrganizationRoleBinding,
  getEffectivePermissions,
  getInstanceUserDetail,
  listInstanceRoleBindings,
  listInstanceUsers,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
  normalizeReturnTo: vi.fn((value?: string | null) => value?.trim() || '/'),
}))

describe('Security settings RBAC', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
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
    expect(
      await findByText(
        /Agent scopes are related runtime token capabilities, but they are not reused as human permissions\./,
      ),
    ).toBeTruthy()

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
})
