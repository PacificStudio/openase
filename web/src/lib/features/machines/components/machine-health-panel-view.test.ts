import { describe, expect, it } from 'vitest'

import { parseMachineSnapshot } from '../model'
import { buildAuditRows, buildLevelCards } from './machine-health-panel-view'

describe('machine health panel view', () => {
  it('treats GitHub CLI auth as observational and token probe as the readiness signal', () => {
    const snapshot = parseMachineSnapshot({
      checked_at: '2026-03-31T10:00:00Z',
      last_success: true,
      agent_dispatchable: true,
      monitor: {
        l5: {
          checked_at: '2026-03-31T10:00:00Z',
        },
      },
      agent_environment: {},
      full_audit: {
        checked_at: '2026-03-31T10:00:00Z',
        git: {
          installed: true,
        },
        gh_cli: {
          installed: true,
          auth_status: 'logged_in',
        },
        github_token_probe: {
          checked_at: '2026-03-31T10:00:00Z',
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
          npm_reachable: true,
        },
      },
    })

    expect(snapshot).not.toBeNull()

    const levelCards = buildLevelCards(snapshot!)
    expect(levelCards.find((card) => card.id === 'l5')?.value).toBe(
      'Git, GitHub CLI observation, token probe, and network audit captured',
    )

    const auditRows = buildAuditRows(snapshot!)

    const ghCliRow = auditRows.find((row) => row.kind === 'gh-cli')
    expect(ghCliRow).toMatchObject({
      kind: 'gh-cli',
      label: 'GitHub CLI',
      installed: 'yes',
      authStatus: 'logged_in',
    })

    const tokenRow = auditRows.find((row) => row.kind === 'token-probe')
    expect(tokenRow).toMatchObject({
      kind: 'token-probe',
      label: 'GitHub Token',
      state: 'valid',
      permissions: ['repo', 'read:org'],
      detail: null,
    })
  })
})
