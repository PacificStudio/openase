import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import { InstanceAdminPage } from '$lib/features/admin'
import {
  configuredSessionGovernance,
  hydrateOidcAuth,
  mockEffectivePermissionsByScope,
} from './security-settings-human-auth.fixtures'

vi.hoisted(() => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation(() => ({
      matches: false,
      media: '',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
})

const { getAdminSecuritySettings, getSessionGovernance } = vi.hoisted(() => ({
  getAdminSecuritySettings: vi.fn(),
  getSessionGovernance: vi.fn(),
}))

const {
  adminRevokeUserAuthSessions,
  createInstanceRoleBinding,
  deleteInstanceRoleBinding,
  getEffectivePermissions,
  getInstanceUserDetail,
  listInstanceRoleBindings,
  listInstanceUsers,
  logoutHumanSession,
  transitionInstanceUserStatus,
} = vi.hoisted(() => ({
  adminRevokeUserAuthSessions: vi.fn(),
  createInstanceRoleBinding: vi.fn(),
  deleteInstanceRoleBinding: vi.fn(),
  getEffectivePermissions: vi.fn(),
  getInstanceUserDetail: vi.fn(),
  listInstanceRoleBindings: vi.fn(),
  listInstanceUsers: vi.fn(),
  logoutHumanSession: vi.fn(),
  transitionInstanceUserStatus: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  getAdminSecuritySettings,
  getSessionGovernance,
  revokeAllOtherAuthSessions: vi.fn(),
  revokeAuthSession: vi.fn(),
}))

vi.mock('$lib/api/auth', () => ({
  adminRevokeUserAuthSessions,
  createInstanceRoleBinding,
  deleteInstanceRoleBinding,
  getEffectivePermissions,
  getInstanceUserDetail,
  listInstanceRoleBindings,
  listInstanceUsers,
  logoutHumanSession,
  normalizeReturnTo: vi.fn((value?: string | null) => value?.trim() || '/'),
  transitionInstanceUserStatus,
}))

function seedInstanceAdmin() {
  hydrateOidcAuth()
  getAdminSecuritySettings.mockResolvedValue({
    settings: {
      auth: {
        active_mode: 'oidc',
        configured_mode: 'oidc',
        issuer_url: 'https://idp.example.com',
        local_principal: 'local_instance_admin:default',
        mode_summary: 'OIDC is active.',
        recommended_mode: 'oidc',
        public_exposure_risk: 'medium',
        warnings: [],
        next_steps: [],
        config_path: '~/.openase/config.yaml',
        bootstrap_state: {
          status: 'configured',
          admin_emails: ['admin@example.com'],
          summary: '1 bootstrap admin email(s) configured.',
        },
        oidc_draft: {
          issuer_url: 'https://idp.example.com',
          client_id: 'openase',
          client_secret_configured: true,
          redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
          scopes: ['openid', 'profile', 'email', 'groups'],
          allowed_email_domains: ['example.com'],
          bootstrap_admin_emails: ['admin@example.com'],
        },
        docs: [],
      },
      approval_policies: {
        status: 'reserved',
        rules_count: 0,
        summary: 'Reserved for future high-risk actions.',
      },
    },
  })
  getEffectivePermissions.mockImplementation(mockEffectivePermissionsByScope)
  listInstanceRoleBindings.mockResolvedValue([])
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

describe('Instance admin user directory', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    vi.clearAllMocks()
  })

  it('renders the instance user directory and identity detail', async () => {
    seedInstanceAdmin()

    const { findByText } = render(InstanceAdminPage)

    expect(await findByText('Instance Admin')).toBeTruthy()
    expect(await findByText('User directory and deprovision')).toBeTruthy()
    expect(await findByText('Bob Reviewer')).toBeTruthy()
    expect(await findByText('Identity governance detail')).toBeTruthy()
    expect(await findByText('subject: subject-bob')).toBeTruthy()
  })

  it('disables a cached user from the instance directory with an audit reason', async () => {
    seedInstanceAdmin()
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

    const { findByPlaceholderText, findByRole } = render(InstanceAdminPage)
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
