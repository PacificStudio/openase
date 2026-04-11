import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { SecurityAuthSettings } from '$lib/api/contracts'
import SecuritySettingsHumanAuthSetupPanel from './security-settings-human-auth-setup-panel.svelte'
import { configuredSecurity, currentProject } from './security-settings.test-helpers'

const { enableOIDC, saveOIDCDraft, testOIDCDraft } = vi.hoisted(() => ({
  enableOIDC: vi.fn(),
  saveOIDCDraft: vi.fn(),
  testOIDCDraft: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  enableOIDC,
  saveOIDCDraft,
  testOIDCDraft,
}))

function authFixture(): SecurityAuthSettings {
  return configuredSecurity().auth
}

function expectedDraftPayload(overrides?: { session_ttl?: string; session_idle_ttl?: string }) {
  return {
    issuer_url: 'https://idp.example.com',
    client_id: 'openase',
    client_secret: '',
    redirect_mode: 'fixed',
    fixed_redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
    scopes: ['openid', 'profile', 'email', 'groups'],
    allowed_email_domains: ['example.com'],
    bootstrap_admin_emails: ['admin@example.com'],
    session_ttl: overrides?.session_ttl ?? '8h0m0s',
    session_idle_ttl: overrides?.session_idle_ttl ?? '30m0s',
  }
}

describe('Security settings human auth setup panel', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('shows session policy fields and submits them for save, test, and enable', async () => {
    const projectId = currentProject().id
    const security = configuredSecurity()

    saveOIDCDraft.mockResolvedValue({ security })
    testOIDCDraft.mockResolvedValue({
      status: 'ok',
      message: 'OIDC provider discovery succeeded.',
      issuer_url: 'https://idp.example.com',
      authorization_endpoint: 'https://idp.example.com/auth',
      token_endpoint: 'https://idp.example.com/token',
      redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
      warnings: [],
    })
    enableOIDC.mockResolvedValue({
      security,
      activation: {
        status: 'configured',
        message: 'OIDC is configured.',
        restart_required: false,
        next_steps: [],
      },
    })

    const { findAllByText, findByLabelText, findByRole, findByText } = render(
      SecuritySettingsHumanAuthSetupPanel,
      {
        auth: authFixture(),
        projectId,
      },
    )

    const sessionTTL = (await findByLabelText('Session TTL')) as HTMLInputElement
    const sessionIdleTTL = (await findByLabelText('Idle TTL')) as HTMLInputElement

    expect(sessionTTL.value).toBe('8h0m0s')
    expect(sessionIdleTTL.value).toBe('30m0s')
    expect(await findByText(/Absolute browser session lifetime/)).toBeTruthy()
    expect((await findAllByText(/0 or 0s means never expires/)).length).toBeGreaterThan(0)

    await fireEvent.input(sessionTTL, { target: { value: '1h' } })
    await fireEvent.input(sessionIdleTTL, { target: { value: '15m' } })

    await fireEvent.click(await findByRole('button', { name: 'Save draft' }))
    await waitFor(() => {
      expect(saveOIDCDraft).toHaveBeenCalledWith(
        projectId,
        expectedDraftPayload({ session_ttl: '1h', session_idle_ttl: '15m' }),
      )
    })

    await fireEvent.click(await findByRole('button', { name: 'Test configuration' }))
    await waitFor(() => {
      expect(testOIDCDraft).toHaveBeenCalledWith(
        projectId,
        expectedDraftPayload({ session_ttl: '1h', session_idle_ttl: '15m' }),
      )
    })

    await fireEvent.click(await findByRole('button', { name: 'Enable OIDC' }))
    await waitFor(() => {
      expect(enableOIDC).toHaveBeenCalledWith(
        projectId,
        expectedDraftPayload({ session_ttl: '1h', session_idle_ttl: '15m' }),
      )
    })
  })
})
