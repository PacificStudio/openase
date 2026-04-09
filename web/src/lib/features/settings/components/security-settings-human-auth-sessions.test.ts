import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import SecuritySettingsHumanAuthSessions from './security-settings-human-auth-sessions.svelte'

const { goto } = vi.hoisted(() => ({
  goto: vi.fn(),
}))

const { getSessionGovernance, revokeAllOtherAuthSessions, revokeAuthSession } = vi.hoisted(() => ({
  getSessionGovernance: vi.fn(),
  revokeAllOtherAuthSessions: vi.fn(),
  revokeAuthSession: vi.fn(),
}))

const { logoutHumanSession, normalizeReturnTo } = vi.hoisted(() => ({
  logoutHumanSession: vi.fn(),
  normalizeReturnTo: vi.fn((value?: string | null) => value?.trim() || '/'),
}))

vi.mock('$app/navigation', () => ({
  goto,
}))

vi.mock('$lib/api/auth', () => ({
  logoutHumanSession,
  normalizeReturnTo,
}))

vi.mock('$lib/api/openase', () => ({
  getSessionGovernance,
  revokeAllOtherAuthSessions,
  revokeAuthSession,
}))

describe('Security settings human auth sessions', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    vi.clearAllMocks()
  })

  it('revokes other sessions from the session governance panel', async () => {
    authStore.hydrate({
      authMode: 'oidc',
      loginRequired: true,
      authenticated: true,
      principalKind: 'human_session',
      authConfigured: true,
      sessionGovernanceAvailable: true,
      canManageAuth: true,
      csrfToken: 'csrf-token',
      user: {
        id: 'user-1',
        primaryEmail: 'alice@example.com',
        displayName: 'Alice Control Plane',
      },
      roles: ['instance_admin'],
      permissions: ['security_setting.update'],
    })

    getSessionGovernance.mockResolvedValue({
      authMode: 'oidc',
      currentSessionID: 'session-current',
      sessions: [
        {
          id: 'session-current',
          current: true,
          device: { kind: 'desktop', os: 'Linux', browser: 'Firefox', label: 'Firefox on Linux' },
          createdAt: '2026-04-04T10:00:00Z',
          lastActiveAt: '2026-04-04T10:30:00Z',
          expiresAt: '2026-04-04T18:00:00Z',
          idleExpiresAt: '2026-04-04T11:00:00Z',
        },
        {
          id: 'session-other',
          current: false,
          device: { kind: 'mobile', os: 'iOS', browser: 'Safari', label: 'Safari on iPhone' },
          createdAt: '2026-04-04T09:00:00Z',
          lastActiveAt: '2026-04-04T09:45:00Z',
          expiresAt: '2026-04-04T18:00:00Z',
          idleExpiresAt: '2026-04-04T10:15:00Z',
        },
      ],
      auditEvents: [],
      stepUp: {
        status: 'reserved',
        summary: 'Reserved for future high-risk actions.',
        supportedMethods: [],
      },
    })
    revokeAllOtherAuthSessions.mockResolvedValue({ revoked_count: 1 })

    const { findByRole, findByText } = render(SecuritySettingsHumanAuthSessions)

    expect(await findByText('Session governance')).toBeTruthy()
    expect(await findByText('Safari on iPhone')).toBeTruthy()

    const button = await findByRole('button', { name: 'Revoke other sessions' })
    await fireEvent.click(button)

    await waitFor(() => {
      expect(revokeAllOtherAuthSessions).toHaveBeenCalled()
    })
  })
})
