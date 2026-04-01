import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { summarizeAgentProviderRateLimit, summarizeProviderRateLimit } from './rate-limit'

describe('summarizeProviderRateLimit', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-01T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders codex live rate limit percentages without over-scaling', () => {
    const summary = summarizeProviderRateLimit({
      adapterType: 'codex-app-server',
      modelName: 'gpt-5.4',
      cliRateLimitUpdatedAt: '2026-04-01T11:45:00Z',
      cliRateLimit: {
        provider: 'codex',
        raw: {},
        codex: {
          limitId: 'codex',
          planType: 'pro',
          primary: {
            usedPercent: 15,
            windowMinutes: 300,
            resetsAt: '2026-04-01T15:30:32Z',
          },
          secondary: null,
        },
      },
    })

    expect(summary).not.toBeNull()
    expect(summary?.headline).toBe('15.0% used · 300m window · pro')
    expect(summary?.updatedLabel).toContain('Updated 15m ago')
    expect(summary?.updatedLabel).toContain('2026')
  })

  it('still supports legacy fractional codex values', () => {
    const summary = summarizeProviderRateLimit({
      adapterType: 'codex-app-server',
      modelName: 'gpt-5.4',
      cliRateLimitUpdatedAt: '2026-04-01T11:45:00Z',
      cliRateLimit: {
        provider: 'codex',
        raw: {},
        codex: {
          limitId: 'codex',
          planType: 'pro',
          primary: {
            usedPercent: 0.42,
            windowMinutes: 300,
            resetsAt: '2026-04-01T15:30:32Z',
          },
          secondary: null,
        },
      },
    })

    expect(summary?.headline).toBe('42.0% used · 300m window · pro')
  })

  it('maps API provider payloads for dashboard display', () => {
    const summary = summarizeAgentProviderRateLimit({
      adapter_type: 'claude-code-cli',
      model_name: 'claude-sonnet-4',
      cli_rate_limit_updated_at: '2026-04-01T11:45:00Z',
      cli_rate_limit: {
        provider: 'claude_code',
        raw: { status: 'allowed' },
        claude_code: {
          status: 'allowed',
          rate_limit_type: 'five_hour',
          resets_at: '2026-04-01T15:30:32Z',
          overage_status: 'rejected',
          overage_disabled_reason: '',
          is_using_overage: false,
        },
        codex: null,
        gemini: null,
      },
    })

    expect(summary?.headline).toBe('allowed · five_hour')
    expect(summary?.detail).toContain('Resets')
    expect(summary?.updatedLabel).toContain('Updated 15m ago')
  })
})
