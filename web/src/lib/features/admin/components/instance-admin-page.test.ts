import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import InstanceAdminPage from './instance-admin-page.svelte'

const {
  adminRevokeAuthSession,
  adminRevokeUserAuthSessions,
  getEffectivePermissions,
  getInstanceUserDetail,
  listInstanceUsers,
  logoutHumanSession,
  transitionInstanceUserStatus,
} = vi.hoisted(() => ({
  adminRevokeAuthSession: vi.fn(),
  adminRevokeUserAuthSessions: vi.fn(),
  getEffectivePermissions: vi.fn(),
  getInstanceUserDetail: vi.fn(),
  listInstanceUsers: vi.fn(),
  logoutHumanSession: vi.fn(),
  transitionInstanceUserStatus: vi.fn(),
}))

const { getSessionGovernance, revokeAllOtherAuthSessions, revokeAuthSession } = vi.hoisted(() => ({
  getSessionGovernance: vi.fn(),
  revokeAllOtherAuthSessions: vi.fn(),
  revokeAuthSession: vi.fn(),
}))

vi.mock('$lib/api/auth', () => ({
  adminRevokeAuthSession,
  adminRevokeUserAuthSessions,
  getEffectivePermissions,
  getInstanceUserDetail,
  listInstanceUsers,
  logoutHumanSession,
  normalizeReturnTo: vi.fn((value?: string | null) => value?.trim() || '/'),
  transitionInstanceUserStatus,
}))

vi.mock('$lib/api/openase', () => ({
  getSessionGovernance,
  revokeAllOtherAuthSessions,
  revokeAuthSession,
}))

function seedOidcAdmin() {
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
    permissions: ['security.read', 'security.manage'],
  })

  getEffectivePermissions.mockResolvedValue({
    user: {
      id: 'user-1',
      primary_email: 'alice@example.com',
      display_name: 'Alice Control Plane',
    },
    scope: { kind: 'instance', id: '' },
    roles: ['instance_admin'],
    permissions: ['security.read', 'security.manage'],
    groups: [],
  })
  getSessionGovernance.mockResolvedValue({
    authMode: 'oidc',
    currentSessionID: 'session-current',
    sessions: [
      {
        id: 'session-current',
        current: true,
        device: { kind: 'desktop', os: 'Linux', browser: 'Firefox', label: 'Firefox on Linux' },
        ipSummary: '198.51.100.0/24',
        createdAt: '2026-04-04T10:00:00Z',
        lastActiveAt: '2026-04-04T10:30:00Z',
        expiresAt: '2026-04-04T18:00:00Z',
        idleExpiresAt: '2026-04-04T11:00:00Z',
      },
    ],
    auditEvents: [],
    stepUp: {
      status: 'reserved',
      summary: 'Reserved for future high-risk actions.',
      supportedMethods: [],
    },
  })
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
    identities: [],
    groups: [],
    activeSessions: [],
    activeSessionCount: 0,
    latestStatusAudit: undefined,
    recentAuditEvents: [],
  })
}

describe('Instance admin page', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    vi.clearAllMocks()
  })

  it('shows minimal disabled-mode diagnostics without multi-user governance semantics', async () => {
    authStore.hydrate({
      authMode: 'disabled',
      authenticated: false,
      roles: [],
      permissions: [],
    })

    const { findAllByText, findByText } = render(InstanceAdminPage)

    expect(await findByText('Disabled-mode diagnostics')).toBeTruthy()
    expect((await findAllByText('local_instance_admin:default')).length).toBeGreaterThan(0)
    expect(await findByText('Break-glass and recovery')).toBeTruthy()
  })

  it('loads current-session context and the governable user directory for oidc admins', async () => {
    seedOidcAdmin()

    const { findByText } = render(InstanceAdminPage)

    expect(await findByText('Current session boundary')).toBeTruthy()
    expect(await findByText('Session governance')).toBeTruthy()
    expect(await findByText('User directory and deprovision')).toBeTruthy()
    expect(await findByText('Bob Reviewer')).toBeTruthy()
  })
})
