import { describe, expect, it } from 'vitest'

import { buildMockProjectConversationReply } from './project-conversation-data'
import { createInitialState } from './initial-state'
import {
  buildTicketDetailPayload,
  buildTicketRunDetailPayload,
  buildTicketRunsPayload,
} from './ticket-runtime-data'

describe('ticket runtime payloads', () => {
  it('builds the seeded default ticket detail payload', () => {
    const state = createInitialState()
    const payload = buildTicketDetailPayload(state, 'ticket-1')
    const ticket = payload?.ticket as { identifier?: string } | undefined

    expect(ticket?.identifier).toBe('ASE-101')
    expect(payload?.pickup_diagnosis.state).toBe('blocked')
    expect(payload?.repo_scopes).toHaveLength(1)
    expect(payload?.hook_history).toHaveLength(1)
  })

  it('returns empty or null payloads for unknown runs and tickets', () => {
    const state = createInitialState()

    expect(buildTicketDetailPayload(state, 'missing-ticket')).toBeNull()
    expect(buildTicketRunsPayload(state, 'missing-ticket')).toEqual({ runs: [] })
    expect(buildTicketRunDetailPayload(state, 'ticket-1', 'missing-run')).toBeNull()
  })

  it('answers ticket-focused mock conversation prompts from ticket state', () => {
    const reply = buildMockProjectConversationReply(
      'Which repos does this ticket currently affect?',
      {
        identifier: 'ASE-101',
        status: 'In Progress',
        retryPaused: true,
        pauseReason: 'Repeated hook failures',
        repoScopes: [{ repo_name: 'openase', branch_name: 'refactor/ase-275-frontend-debt' }],
        hookHistory: [],
        currentRun: { status: 'failed', last_error: 'hook failed' },
      },
    )

    expect(reply).toContain('openase @ refactor/ase-275-frontend-debt')
  })
})
