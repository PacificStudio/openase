import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import LoginGatePage from './login-gate-page.svelte'

vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
}))

describe('Login auth gate', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('shows only the OIDC entrypoint when capabilities allow OIDC, even if auth_mode is stale', () => {
    const { getByText, queryByText, queryByLabelText } = render(LoginGatePage, {
      props: {
        data: {
          returnTo: '/orgs',
          authSession: {
            authMode: 'disabled',
            loginRequired: true,
            authenticated: false,
            principalKind: 'anonymous',
            authCapabilities: {
              availableAuthMethods: ['oidc'],
              currentAuthMethod: 'oidc',
            },
            authConfigured: true,
            sessionGovernanceAvailable: false,
            canManageAuth: false,
            issuerURL: 'https://idp.example.com',
            roles: [],
            permissions: [],
          },
        },
      },
    })

    expect(getByText('OIDC sign-in')).toBeTruthy()
    expect(getByText('Continue with OIDC')).toBeTruthy()
    expect(queryByText('Local bootstrap authorization')).toBeNull()
    expect(queryByLabelText('Paste local bootstrap URL')).toBeNull()
  })

  it('shows only the local bootstrap entrypoint when capabilities require the bootstrap link path', () => {
    const { getByText, getByLabelText, queryByText } = render(LoginGatePage, {
      props: {
        data: {
          returnTo: '/admin/auth',
          authSession: {
            authMode: 'oidc',
            loginRequired: true,
            authenticated: false,
            principalKind: 'anonymous',
            authCapabilities: {
              availableAuthMethods: ['local_bootstrap_link'],
              currentAuthMethod: 'local_bootstrap_link',
            },
            authConfigured: false,
            sessionGovernanceAvailable: false,
            canManageAuth: false,
            issuerURL: '',
            roles: [],
            permissions: [],
          },
        },
      },
    })

    expect(getByText('Local bootstrap authorization')).toBeTruthy()
    expect(getByText(/openase auth bootstrap create-link --return-to \/admin\/auth/)).toBeTruthy()
    expect(getByLabelText('Paste local bootstrap URL')).toBeTruthy()
    expect(queryByText('Continue with OIDC')).toBeNull()
  })
})
