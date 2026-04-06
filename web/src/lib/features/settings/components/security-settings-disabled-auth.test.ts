import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import { appStore } from '$lib/stores/app.svelte'
import SecuritySettings from './security-settings.svelte'
import { currentProject, disabledSecurity } from './security-settings.test-helpers'

const {
  deleteGitHubOutboundCredential,
  enableOIDC,
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
  getSecuritySettings: vi.fn(),
  importGitHubOutboundCredentialFromGHCLI: vi.fn(),
  revokeAllOtherAuthSessions: vi.fn(),
  revokeAuthSession: vi.fn(),
  retestGitHubOutboundCredential: vi.fn(),
  saveOIDCDraft: vi.fn(),
  saveGitHubOutboundCredential: vi.fn(),
  testOIDCDraft: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  deleteGitHubOutboundCredential,
  enableOIDC,
  getSecuritySettings,
  importGitHubOutboundCredentialFromGHCLI,
  revokeAllOtherAuthSessions,
  revokeAuthSession,
  retestGitHubOutboundCredential,
  saveOIDCDraft,
  saveGitHubOutboundCredential,
  testOIDCDraft,
}))

describe('Security settings disabled auth setup', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    appStore.currentProject = null
    vi.clearAllMocks()
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

  it('keeps disabled mode active while saving testing and enabling OIDC explicitly', async () => {
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
