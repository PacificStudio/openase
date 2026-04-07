import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { authStore } from '$lib/stores/auth.svelte'
import AdminAuthPage from './admin-auth-page.svelte'
import { disabledSecurity } from '$lib/features/settings/components/security-settings.test-helpers'

const { disableAdminAuth, enableAdminOIDC, getAdminAuth, saveAdminOIDCDraft, testAdminOIDCDraft } =
  vi.hoisted(() => ({
    disableAdminAuth: vi.fn(),
    enableAdminOIDC: vi.fn(),
    getAdminAuth: vi.fn(),
    saveAdminOIDCDraft: vi.fn(),
    testAdminOIDCDraft: vi.fn(),
  }))

vi.mock('$lib/api/openase', () => ({
  disableAdminAuth,
  enableAdminOIDC,
  getAdminAuth,
  saveAdminOIDCDraft,
  testAdminOIDCDraft,
}))

describe('Admin auth page', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    vi.clearAllMocks()
  })

  it('renders the instance auth posture and persisted diagnostics', async () => {
    getAdminAuth.mockResolvedValue({ auth: disabledSecurity().auth })

    const { findByText, findByLabelText } = render(AdminAuthPage)

    expect(await findByText('Instance scope')).toBeTruthy()
    expect(await findByText('Last validation diagnostics')).toBeTruthy()
    expect(await findByText('OIDC configuration')).toBeTruthy()
    expect(await findByLabelText('Issuer URL')).toBeTruthy()
    expect(await findByText('8h0m0s')).toBeTruthy()
  })

  it('saves, tests, enables, and disables instance auth explicitly', async () => {
    getAdminAuth.mockResolvedValue({ auth: disabledSecurity().auth })
    saveAdminOIDCDraft.mockResolvedValue({ auth: disabledSecurity().auth })
    testAdminOIDCDraft.mockResolvedValue({
      status: 'ok',
      message: 'OIDC discovery succeeded.',
      issuer_url: 'https://idp.example.com',
      authorization_endpoint: 'https://idp.example.com/auth',
      token_endpoint: 'https://idp.example.com/token',
      redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
      warnings: [],
    })
    enableAdminOIDC.mockResolvedValue({
      transition: {
        status: 'configured',
        message: 'OIDC is configured for the instance.',
        restart_required: true,
        next_steps: ['Restart OpenASE', 'Complete bootstrap login'],
      },
      auth: {
        ...disabledSecurity().auth,
        configured_mode: 'oidc',
      },
    })
    disableAdminAuth.mockResolvedValue({
      transition: {
        status: 'disabled',
        message: 'Disabled mode is configured again.',
        restart_required: false,
        next_steps: ['Keep the saved OIDC draft'],
      },
      auth: disabledSecurity().auth,
    })

    const { findByLabelText, findByRole, findByText } = render(AdminAuthPage)

    await fireEvent.input(await findByLabelText('Issuer URL'), {
      target: { value: 'https://idp.example.com' },
    })
    await fireEvent.input(await findByLabelText('Client ID'), {
      target: { value: 'openase' },
    })
    await fireEvent.click(await findByRole('button', { name: 'Save configuration' }))

    await waitFor(() => {
      expect(saveAdminOIDCDraft).toHaveBeenCalledWith({
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
      expect(testAdminOIDCDraft).toHaveBeenCalledTimes(1)
    })
    expect(await findByText('OIDC discovery succeeded.')).toBeTruthy()

    await fireEvent.click(await findByRole('button', { name: 'Enable OIDC' }))
    await waitFor(() => {
      expect(enableAdminOIDC).toHaveBeenCalledTimes(1)
    })
    expect(await findByText('Configured oidc')).toBeTruthy()

    await fireEvent.click(await findByRole('button', { name: 'Revert to disabled' }))
    await waitFor(() => {
      expect(disableAdminAuth).toHaveBeenCalledTimes(1)
    })
    expect(await findByText('Disabled mode is configured again.')).toBeTruthy()
  })
})
