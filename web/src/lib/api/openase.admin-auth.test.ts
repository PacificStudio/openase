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

import { disableAdminAuth, getAdminAuth, getSecuritySettings, testAdminOIDCDraft } from './openase'

function disabledBootstrapAuthFixture() {
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
    warnings: [],
    next_steps: [
      'You can keep disabled mode for local single-user use with no extra IAM overhead.',
      'Save draft OIDC settings, test discovery, then enable OIDC only when you are ready for multi-user browser login.',
    ],
    config_path: 'db:instance_auth_configs',
    bootstrap_state: {
      status: 'disabled',
      admin_emails: [],
      summary: 'OIDC bootstrap is inactive while disabled mode stays configured.',
    },
    session_policy: {
      session_ttl: '8h0m0s',
      session_idle_ttl: '30m0s',
    },
    last_validation: {
      status: 'not_tested',
      message: 'No OIDC validation has been run yet for this disabled instance.',
      checked_at: null,
      issuer_url: '',
      authorization_endpoint: '',
      token_endpoint: '',
      redirect_url: '',
      warnings: null,
    },
    oidc_draft: {
      issuer_url: '',
      client_id: '',
      client_secret_configured: false,
      redirect_mode: 'auto',
      fixed_redirect_url: '',
      scopes: ['openid', 'profile', 'email'],
      allowed_email_domains: null,
      bootstrap_admin_emails: null,
    },
    docs: [],
  }
}

describe('admin auth API helpers', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    patch.mockReset()
    put.mockReset()
    del.mockReset()
  })

  it('normalizes null auth arrays from the admin auth endpoint', async () => {
    get.mockResolvedValue({
      auth: disabledBootstrapAuthFixture(),
    })

    const payload = await getAdminAuth()

    expect(payload.auth.oidc_draft.allowed_email_domains).toEqual([])
    expect(payload.auth.oidc_draft.bootstrap_admin_emails).toEqual([])
    expect(payload.auth.last_validation.warnings).toEqual([])
  })

  it('normalizes null auth arrays from disabled-mode transitions', async () => {
    post.mockResolvedValue({
      transition: {
        status: 'disabled',
        message: 'Disabled mode is configured again.',
        restart_required: false,
        next_steps: ['Keep the saved OIDC draft'],
      },
      auth: disabledBootstrapAuthFixture(),
    })

    const payload = await disableAdminAuth()

    expect(payload.auth.oidc_draft.allowed_email_domains).toEqual([])
    expect(payload.auth.oidc_draft.bootstrap_admin_emails).toEqual([])
    expect(payload.auth.last_validation.warnings).toEqual([])
  })

  it('normalizes null auth arrays from project security settings', async () => {
    get.mockResolvedValue({
      security: {
        auth: disabledBootstrapAuthFixture(),
      },
    })

    const payload = await getSecuritySettings('project-123')

    expect(payload.security.auth.oidc_draft.allowed_email_domains).toEqual([])
    expect(payload.security.auth.oidc_draft.bootstrap_admin_emails).toEqual([])
    expect(payload.security.auth.last_validation.warnings).toEqual([])
  })

  it('normalizes nullable warnings from OIDC draft validation responses', async () => {
    post.mockResolvedValue({
      status: 'not_tested',
      message: 'Validation has not run yet.',
      issuer_url: '',
      authorization_endpoint: '',
      token_endpoint: '',
      redirect_url: '',
      warnings: null,
    })

    const payload = await testAdminOIDCDraft({
      issuer_url: '',
      client_id: '',
      client_secret: '',
      redirect_mode: 'auto',
      fixed_redirect_url: '',
      scopes: ['openid', 'profile', 'email'],
      allowed_email_domains: [],
      bootstrap_admin_emails: [],
      session_ttl: '0s',
      session_idle_ttl: '0s',
    })

    expect(payload.warnings).toEqual([])
  })
})
