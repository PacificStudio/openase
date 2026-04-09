import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, patch, put, del } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}))

vi.mock('./client', () => ({
  api: {
    get,
    post,
    patch,
    put,
    delete: del,
  },
}))

import { getSessionGovernance } from './openase'

describe('session governance helpers', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('normalizes auth session governance responses', async () => {
    get.mockResolvedValue({
      auth_mode: 'oidc',
      current_session_id: 'session-current',
      sessions: [
        {
          id: 'session-current',
          current: true,
          device: {
            kind: 'desktop',
            os: 'Linux',
            browser: 'Firefox',
            label: 'Firefox on Linux',
          },
          created_at: '2026-04-09T10:00:00Z',
          last_active_at: '2026-04-09T10:30:00Z',
          expires_at: '2026-04-09T18:00:00Z',
          idle_expires_at: '2026-04-09T11:00:00Z',
        },
      ],
      audit_events: [
        {
          id: 'event-1',
          event_type: 'session.revoked',
          actor_id: 'user-1',
          message: 'Session revoked.',
          created_at: '2026-04-09T10:45:00Z',
        },
      ],
      step_up: {
        status: 'reserved',
        summary: 'Reserved for future high-risk actions.',
        supported_methods: [],
      },
    })

    const payload = await getSessionGovernance()

    expect(payload.authMode).toBe('oidc')
    expect(payload.currentSessionID).toBe('session-current')
    expect(payload.sessions[0]?.createdAt).toBe('2026-04-09T10:00:00Z')
    expect(payload.auditEvents[0]?.eventType).toBe('session.revoked')
    expect(payload.stepUp.summary).toBe('Reserved for future high-risk actions.')
  })
})
