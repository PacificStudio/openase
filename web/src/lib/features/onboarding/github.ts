import type { GitHubTokenState } from './types'

type RawObject = Record<string, unknown>

function asObject(value: unknown): RawObject | null {
  return value && typeof value === 'object' ? (value as RawObject) : null
}

function asString(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

export function parseGitHubTokenState(payload: unknown): GitHubTokenState {
  const root = asObject(payload)
  const security = asObject(root?.security)
  const github = asObject(security?.github)
  const effective = asObject(github?.effective)
  const probe = asObject(effective?.probe)

  const configured = Boolean(effective?.configured)
  const state = asString(probe?.state)
  const login = asString(probe?.login)
  const valid = state === 'valid'

  return {
    hasToken: configured,
    probeStatus:
      state === 'valid' || state === 'invalid' || state === 'testing' ? state : 'unknown',
    login,
    confirmed: valid,
  }
}
