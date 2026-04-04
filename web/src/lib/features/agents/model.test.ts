import { describe, expect, it } from 'vitest'

import type { AgentProvider } from '$lib/api/contracts'
import { buildAgentRows, buildAgentRunRows, buildProviderCards } from './model'
import {
  agentFixture,
  agentRunsFixture,
  providerFixture,
  ticketsFixture,
  workflowFixture,
} from './model.test-fixtures'

describe('agents model', () => {
  it('treats agent.runtime as an aggregate summary when multiple runs are active', () => {
    const rows = buildAgentRows([providerFixture], ticketsFixture, [agentFixture])

    expect(rows).toHaveLength(1)
    expect(rows[0]).toMatchObject({
      id: 'agent-1',
      activeRunCount: 2,
      currentTicket: undefined,
      runtimePhase: 'executing',
      status: 'running',
    })
  })

  it('builds agent run rows as first-class runtime records', () => {
    const rows = buildAgentRunRows(
      [providerFixture],
      ticketsFixture,
      [workflowFixture],
      [agentFixture],
      agentRunsFixture,
    )

    expect(rows).toHaveLength(2)
    expect(rows[0]).toMatchObject({
      id: 'run-2',
      agentName: 'Codex Worker',
      workflowName: 'Coding',
      providerName: 'Codex',
      ticket: {
        id: 'ticket-2',
        identifier: 'ASE-102',
      },
      status: 'executing',
    })
    expect(rows[1]?.ticket.identifier).toBe('ASE-101')
  })

  it('maps provider rate limit snapshots', () => {
    const providerWithRateLimit: AgentProvider = {
      ...providerFixture,
      adapter_type: 'claude-code-cli',
      cli_rate_limit: {
        provider: 'claude_code',
        raw: { status: 'allowed' },
        claude_code: {
          status: 'allowed',
          rate_limit_type: 'five_hour',
          resets_at: '2026-04-01T12:00:00Z',
          overage_status: 'rejected',
          overage_disabled_reason: 'org_level_disabled',
          is_using_overage: false,
        },
        codex: null,
        gemini: null,
      },
      cli_rate_limit_updated_at: '2026-04-01T11:45:00Z',
    }

    const cards = buildProviderCards([providerWithRateLimit], [agentFixture], null)

    expect(cards[0]?.cliRateLimit?.claudeCode?.rateLimitType).toBe('five_hour')
    expect(cards[0]?.cliRateLimitUpdatedAt).toBe('2026-04-01T11:45:00Z')
  })

  it('maps codex and gemini provider rate limit snapshots', () => {
    const codexProvider: AgentProvider = {
      ...providerFixture,
      adapter_type: 'codex-app-server',
      cli_rate_limit: {
        provider: 'codex',
        raw: { limit_id: 'codex' },
        codex: {
          limit_id: 'codex',
          limit_name: '',
          plan_type: 'pro',
          primary: {
            used_percent: 0.42,
            window_minutes: 300,
            resets_at: '2026-04-01T12:00:00Z',
          },
          secondary: null,
        },
        claude_code: null,
        gemini: null,
      },
      cli_rate_limit_updated_at: '2026-04-01T11:45:00Z',
    }
    const geminiProvider: AgentProvider = {
      ...providerFixture,
      id: 'provider-gemini',
      adapter_type: 'gemini-cli',
      model_name: 'gemini-2.5-pro',
      cli_rate_limit: {
        provider: 'gemini',
        raw: { authType: 'oauth-personal' },
        gemini: {
          auth_type: 'oauth-personal',
          remaining: 3,
          limit: 10,
          reset_time: '2026-04-02T10:02:55Z',
          buckets: [
            {
              model_id: 'gemini-2.5-pro',
              token_type: 'REQUESTS',
              remaining_amount: '',
              remaining_fraction: 0.3,
              reset_time: '2026-04-02T10:02:55Z',
            },
          ],
        },
        claude_code: null,
        codex: null,
      },
      cli_rate_limit_updated_at: '2026-04-01T11:46:00Z',
    }

    const cards = buildProviderCards([codexProvider, geminiProvider], [agentFixture], null)

    expect(cards[0]?.cliRateLimit?.codex?.primary?.usedPercent).toBe(0.42)
    expect(cards[1]?.cliRateLimit?.gemini?.remaining).toBe(3)
    expect(cards[1]?.cliRateLimit?.gemini?.buckets?.[0]?.modelId).toBe('gemini-2.5-pro')
  })
})
