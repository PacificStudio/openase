import { describe, expect, it } from 'vitest'

import { parseMachineSnapshot } from './model'

describe('machines model', () => {
  it('parses multi-level monitor, runtime, and audit snapshots', () => {
    const snapshot = parseMachineSnapshot({
      checked_at: '2026-03-30T14:24:24Z',
      last_success: true,
      agent_dispatchable: true,
      agent_environment_checked_at: '2026-03-30T14:24:24Z',
      cpu_usage_percent: 14.5,
      memory_available_gb: 43,
      gpu_dispatchable: false,
      monitor: {
        l1: {
          checked_at: '2026-03-30T14:24:24Z',
          reachable: true,
        },
        l4: {
          checked_at: '2026-03-30T14:24:24Z',
          agent_dispatchable: true,
        },
        l5: {
          checked_at: '2026-03-30T14:24:24Z',
        },
      },
      agent_environment: {
        claude_code: {
          installed: true,
          version: '2.1.87',
          auth_status: 'logged_in',
          auth_mode: 'login',
          ready: true,
        },
        codex: {
          installed: true,
          version: '0.117.0',
          auth_status: 'logged_in',
          auth_mode: 'login',
          ready: true,
        },
      },
      full_audit: {
        checked_at: '2026-03-30T14:24:24Z',
        git: {
          installed: true,
          user_name: 'Codex',
          user_email: 'codex@openai.com',
        },
        gh_cli: {
          installed: true,
          auth_status: 'logged_in',
        },
        github_token_probe: {
          checked_at: '2026-03-30T14:24:24Z',
          state: 'valid',
          configured: true,
          valid: true,
          permissions: ['repo', 'read:org'],
          repo_access: 'granted',
          last_error: '',
        },
        network: {
          github_reachable: true,
          pypi_reachable: true,
          npm_reachable: false,
        },
      },
    })

    expect(snapshot).not.toBeNull()
    expect(snapshot?.monitor.l4?.agentDispatchable).toBe(true)
    expect(snapshot?.agentEnvironment).toHaveLength(2)
    expect(snapshot?.agentEnvironment[0]).toMatchObject({
      name: 'claude_code',
      ready: true,
      authStatus: 'logged_in',
    })
    expect(snapshot?.fullAudit?.githubTokenProbe?.permissions).toEqual(['repo', 'read:org'])
    expect(snapshot?.fullAudit?.network?.npmReachable).toBe(false)
  })
})
