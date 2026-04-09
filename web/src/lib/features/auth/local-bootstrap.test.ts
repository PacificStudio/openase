import { describe, expect, it } from 'vitest'

import {
  buildLocalBootstrapRedeemPath,
  parseLocalBootstrapAuthorizationBundle,
} from './local-bootstrap'

describe('local bootstrap authorization bundle parsing', () => {
  it('accepts a full URL from another origin and keeps the current auth-gate return target', () => {
    expect(
      parseLocalBootstrapAuthorizationBundle(
        'http://127.0.0.1:19836/local-bootstrap?request_id=req-1&code=code-1&nonce=nonce-1&return_to=%2Fignored',
        '/admin/auth',
      ),
    ).toEqual({
      requestID: 'req-1',
      code: 'code-1',
      nonce: 'nonce-1',
      returnTo: '/admin/auth',
    })
  })

  it('accepts CLI JSON output with escaped url separators', () => {
    expect(
      parseLocalBootstrapAuthorizationBundle(
        JSON.stringify({
          url: 'https://review.example.com/local-bootstrap?request_id=req-2\\u0026code=code-2\\u0026nonce=nonce-2',
        }),
        '/projects/project-1',
      ),
    ).toEqual({
      requestID: 'req-2',
      code: 'code-2',
      nonce: 'nonce-2',
      returnTo: '/projects/project-1',
    })
  })

  it('accepts text-mode CLI output', () => {
    expect(
      parseLocalBootstrapAuthorizationBundle(
        [
          'Open this URL in a browser before 2026-04-09T06:45:00Z:',
          'https://review.example.com/local-bootstrap?request_id=req-3&code=code-3&nonce=nonce-3',
        ].join('\n'),
        '/orgs/org-1',
      ),
    ).toEqual({
      requestID: 'req-3',
      code: 'code-3',
      nonce: 'nonce-3',
      returnTo: '/orgs/org-1',
    })
  })

  it('builds the local redeem route on the current origin', () => {
    expect(
      buildLocalBootstrapRedeemPath({
        requestID: 'req-4',
        code: 'code-4',
        nonce: 'nonce-4',
        returnTo: '/orgs/org-1/projects/project-1/settings?tab=auth',
      }),
    ).toBe(
      '/local-bootstrap?request_id=req-4&code=code-4&nonce=nonce-4&return_to=%2Forgs%2Forg-1%2Fprojects%2Fproject-1%2Fsettings%3Ftab%3Dauth',
    )
  })
})
