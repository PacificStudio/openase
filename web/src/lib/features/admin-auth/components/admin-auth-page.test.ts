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
      'Disabled mode keeps OpenASE in local single-user operation. The current user keeps local highest privilege without browser login or OIDC dependency.',
    recommended_mode:
      'Keep disabled mode for personal or local-only use. Move to OIDC + instance_admin when you need real multi-user browser access control.',
    public_exposure_risk: 'local_only',
    warnings: [
      'Disabled mode is appropriate for local-only or single-user use on a loopback-bound instance.',
    ],
    next_steps: [
      'You can keep disabled mode for local single-user use with no extra IAM overhead.',
      'Save draft OIDC settings, test discovery, then enable OIDC only when you are ready for multi-user browser login.',
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
      redirect_url: 'http://127.0.0.1:19836/api/v1/auth/oidc/callback',
      scopes: ['openid', 'profile', 'email', 'groups'],
      allowed_email_domains: ['example.com'],
      bootstrap_admin_emails: ['admin@example.com'],
    },
    docs: [
      {
        title: 'Mode selection guide',
        href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/human-auth-oidc-rbac.md',
        summary:
          'Choose between disabled mode and OIDC, including local-user and instance_admin guidance.',
      },
      {
        title: 'Dual-mode contract',
        href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/iam-dual-mode-contract.md',
        summary:
          'Read the long-term disabled versus OIDC contract and the explicit enable / rollback flow.',
      },
      {
        title: 'IAM rollout checklist',
        href: 'https://github.com/pacificstudio/openase/blob/main/docs/en/iam-admin-console-rollout.md',
        summary:
          'Roll out the full IAM console in stages with migration checks, rollback steps, and validation coverage.',
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

    expect(await findByText('Migration note')).toBeTruthy()
    expect(await findByText('Instance scope')).toBeTruthy()
    expect(await findByText('Last validation diagnostics')).toBeTruthy()
    expect(await findByText('OIDC configuration')).toBeTruthy()
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
        message: 'Disabled mode is configured again.',
        restart_required: false,
        next_steps: ['Keep the saved OIDC draft'],
      },
      auth: disabledAdminAuthFixture(),
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
