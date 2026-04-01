import { describe, expect, it } from 'vitest'

import { parseGitHubTokenState } from './github'

describe('parseGitHubTokenState', () => {
  it('parses the nested security settings payload used by onboarding', () => {
    const parsed = parseGitHubTokenState({
      security: {
        github: {
          effective: {
            configured: true,
            probe: {
              state: 'valid',
              login: 'octocat',
            },
          },
        },
      },
    })

    expect(parsed).toEqual({
      hasToken: true,
      probeStatus: 'valid',
      login: 'octocat',
      confirmed: true,
    })
  })

  it('falls back to an empty unknown state when the payload is missing', () => {
    expect(parseGitHubTokenState(null)).toEqual({
      hasToken: false,
      probeStatus: 'unknown',
      login: '',
      confirmed: false,
    })
  })
})
