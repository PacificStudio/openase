import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import { authStore } from '$lib/stores/auth.svelte'
import SecuritySettings from './security-settings.svelte'
import {
  configuredSecurity,
  configuredSessionGovernance,
  currentOrg,
  currentProject,
  hydrateOidcAuth,
  mockEffectivePermissionsByScope,
} from './security-settings.test-helpers'

const { getSecuritySettings, getSessionGovernance } = vi.hoisted(() => ({
  getSecuritySettings: vi.fn(),
  getSessionGovernance: vi.fn(),
}))

const {
  getEffectivePermissions,
  getInstanceUserDetail,
  listInstanceRoleBindings,
  listInstanceUsers,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
  transitionInstanceUserStatus,
} = vi.hoisted(() => ({
  getEffectivePermissions: vi.fn(),
  getInstanceUserDetail: vi.fn(),
  listInstanceRoleBindings: vi.fn(),
  listInstanceUsers: vi.fn(),
  listOrganizationRoleBindings: vi.fn(),
  listProjectRoleBindings: vi.fn(),
  transitionInstanceUserStatus: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  getSecuritySettings,
  getSessionGovernance,
}))

vi.mock('$lib/api/auth', () => ({
  getEffectivePermissions,
  getInstanceUserDetail,
  listInstanceRoleBindings,
  listInstanceUsers,
  listOrganizationRoleBindings,
  listProjectRoleBindings,
  transitionInstanceUserStatus,
  normalizeReturnTo: vi.fn((value?: string | null) => value?.trim() || '/'),
}))

function seedUserDirectory() {
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
}

describe('Security settings user directory', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders the user directory and identity governance details', async () => {
    seedUserDirectory()

    const { findByText } = render(SecuritySettings)

    expect(await findByText('User directory and deprovision')).toBeTruthy()
    expect(await findByText('Bob Reviewer')).toBeTruthy()
    expect(await findByText('Identity governance detail')).toBeTruthy()
    expect(await findByText('subject: subject-bob')).toBeTruthy()
    expect(await findByText('OIDC group cache')).toBeTruthy()
    expect(await findByText('No synchronized groups for this user.')).toBeTruthy()
    expect(await findByText('Lifecycle controls')).toBeTruthy()
  })

  it('disables a cached user from the directory with an audit reason', async () => {
    seedUserDirectory()
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
