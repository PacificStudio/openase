import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import LoginGatePage from './login-gate-page.svelte'

const { goto } = vi.hoisted(() => ({
  goto: vi.fn(),
}))

vi.mock('$app/navigation', () => ({
  goto,
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
    expect(queryByLabelText('Paste local bootstrap bundle')).toBeNull()
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
    expect(getByLabelText('Paste local bootstrap bundle')).toBeTruthy()
    expect(getByText(/Accepted formats: the CLI JSON output/)).toBeTruthy()
    expect(queryByText('Continue with OIDC')).toBeNull()
  })

  it('rebuilds a current-origin redeem path from pasted CLI JSON', async () => {
    const { getByLabelText, getByRole } = render(LoginGatePage, {
      props: {
        data: {
          returnTo: '/orgs/org-1/projects/project-1/settings',
          authSession: {
            authMode: 'disabled',
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

    await fireEvent.input(getByLabelText('Paste local bootstrap bundle'), {
      target: {
        value: JSON.stringify({
          request_id: 'req-123',
          code: 'code-123',
          nonce: 'nonce-123',
          url: 'http://127.0.0.1:19836/local-bootstrap?request_id=req-123&code=code-123&nonce=nonce-123',
        }),
      },
    })
    await fireEvent.click(getByRole('button', { name: 'Continue with local bootstrap' }))

    await waitFor(() => {
      expect(goto).toHaveBeenCalledWith(
        '/local-bootstrap?request_id=req-123&code=code-123&nonce=nonce-123&return_to=%2Forgs%2Forg-1%2Fprojects%2Fproject-1%2Fsettings',
      )
    })
  })
})
