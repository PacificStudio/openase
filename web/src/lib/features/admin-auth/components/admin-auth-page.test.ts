import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { SecurityAuthSettings } from '$lib/api/contracts'
import { authStore } from '$lib/stores/auth.svelte'
import AdminAuthPage from './admin-auth-page.svelte'

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

function disabledAdminAuthFixture(): SecurityAuthSettings {
  return {
    active_mode: 'disabled',
    configured_mode: 'disabled',
    issuer_url: '',
    local_principal: 'local_instance_admin:default',
    mode_summary:
      'OIDC is inactive. Browser access on this machine goes through local bootstrap links until you enable OIDC, and the saved OIDC draft remains available for rollout.',
    recommended_mode:
      'Use local bootstrap for personal or recovery access, and enable OIDC when you need managed multi-user browser login.',
    public_exposure_risk: 'local_only',
    warnings: [
      'OIDC is inactive on a loopback-bound instance. Use local bootstrap links for browser access, or enable OIDC before sharing the instance.',
    ],
    next_steps: [
      'Create a local bootstrap link for administrators who still need browser access on this machine.',
      'Save draft OIDC settings, test discovery, then enable OIDC only when you are ready for managed multi-user browser login.',
      'If an OIDC rollout locks you out, run `openase auth break-glass disable-oidc` locally before creating a fresh bootstrap link.',
    ],
    config_path: 'db:instance_auth_configs',
    bootstrap_state: {
      status: 'configured',
      admin_emails: ['admin@example.com'],
      summary:
        '1 bootstrap admin email(s) will receive instance_admin on first successful OIDC login.',
    },
    session_policy: {
      session_ttl: '8h0m0s',
      session_idle_ttl: '30m0s',
    },
    last_validation: {
      status: 'ok',
      message:
        'OIDC discovery succeeded. Saving this draft still keeps the active mode unchanged until you explicitly enable OIDC.',
      checked_at: '2026-04-07T04:12:00Z',
      issuer_url: 'https://idp.example.com',
      authorization_endpoint: 'https://idp.example.com/authorize',
      token_endpoint: 'https://idp.example.com/token',
      redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
      warnings: [],
    },
    oidc_draft: {
      issuer_url: 'https://idp.example.com',
      client_id: 'openase',
      client_secret_configured: true,
      redirect_mode: 'fixed',
      fixed_redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
      scopes: ['openid', 'profile', 'email', 'groups'],
      allowed_email_domains: ['example.com'],
      bootstrap_admin_emails: ['admin@example.com'],
    },
    docs: [
      {
        title: 'Mode selection guide',
        href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/human-auth-oidc-rbac.md',
        summary:
          'Plan local bootstrap access, OIDC rollout, and instance_admin bootstrap coverage.',
      },
      {
        title: 'Dual-mode contract',
        href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/iam-dual-mode-contract.md',
        summary:
          'Read the access-control contract, YAML import behavior, and local recovery paths.',
      },
      {
        title: 'IAM rollout checklist',
        href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/iam-admin-console-rollout.md',
        summary:
          'Roll out IAM with validation checks plus a documented break-glass recovery procedure.',
      },
    ],
  }
}

describe('Admin auth page', () => {
  afterEach(() => {
    cleanup()
    authStore.clear()
    vi.clearAllMocks()
  })

  it('renders the instance auth posture and persisted diagnostics', async () => {
    getAdminAuth.mockResolvedValue({ auth: disabledAdminAuthFixture() })

    const { findByText, findByLabelText } = render(AdminAuthPage)

    expect(await findByText('Instance scope')).toBeTruthy()
    expect(await findByText('Last validation diagnostics')).toBeTruthy()
    expect(await findByText('Draft, validation, and activation')).toBeTruthy()
    expect(await findByLabelText('Issuer URL')).toBeTruthy()
    expect(await findByText('8h0m0s')).toBeTruthy()
    expect(await findByText('Source of truth')).toBeTruthy()
    expect(await findByText('db:instance_auth_configs')).toBeTruthy()
  })

  it('saves, tests, enables, and disables instance auth explicitly', async () => {
    getAdminAuth.mockResolvedValue({ auth: disabledAdminAuthFixture() })
    saveAdminOIDCDraft.mockResolvedValue({ auth: disabledAdminAuthFixture() })
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
        ...disabledAdminAuthFixture(),
        configured_mode: 'oidc',
      },
    })
    disableAdminAuth.mockResolvedValue({
      transition: {
        status: 'disabled',
        message: 'OIDC is inactive again.',
        restart_required: false,
        next_steps: ['Keep the saved OIDC draft'],
      },
      auth: disabledAdminAuthFixture(),
    })

    const { findAllByText, findByLabelText, findByRole, findByText } = render(AdminAuthPage)

    await fireEvent.input(await findByLabelText('Issuer URL'), {
      target: { value: 'https://idp.example.com' },
    })
    await fireEvent.input(await findByLabelText('Client ID'), {
      target: { value: 'openase' },
    })
    await fireEvent.click(await findByRole('button', { name: 'Save draft' }))

    await waitFor(() => {
      expect(saveAdminOIDCDraft).toHaveBeenCalledWith({
        issuer_url: 'https://idp.example.com',
        client_id: 'openase',
        client_secret: '',
        redirect_mode: 'fixed',
        fixed_redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
        scopes: ['openid', 'profile', 'email', 'groups'],
        allowed_email_domains: ['example.com'],
        bootstrap_admin_emails: ['admin@example.com'],
      })
    })

    await fireEvent.click(await findByRole('button', { name: 'Validate draft' }))
    await waitFor(() => {
      expect(testAdminOIDCDraft).toHaveBeenCalledTimes(1)
    })
    expect(await findByText('OIDC discovery succeeded.')).toBeTruthy()

    await fireEvent.click(await findByRole('button', { name: 'Activate OIDC' }))
    await waitFor(() => {
      expect(enableAdminOIDC).toHaveBeenCalledTimes(1)
    })
    expect((await findAllByText('Configured: oidc')).length).toBeGreaterThan(0)

    await fireEvent.click(await findByRole('button', { name: 'Keep local bootstrap' }))
    await waitFor(() => {
      expect(disableAdminAuth).toHaveBeenCalledTimes(1)
    })
    expect(await findByText('OIDC is inactive again.')).toBeTruthy()
  })
})
