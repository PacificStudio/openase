import { cleanup, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { AgentProvider } from '$lib/api/contracts'
import OrganizationProvidersSection from './organization-providers-section.svelte'

describe('OrganizationProvidersSection', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-01T12:00:00Z'))
  })

  afterEach(() => {
    cleanup()
    vi.useRealTimers()
  })

  it('renders provider rate limit details on the dashboard', () => {
    const { getByText } = render(OrganizationProvidersSection, {
      props: {
        providers: [makeProvider()],
        defaultProviderId: 'provider-1',
      },
    })

    expect(getByText('Primary')).toBeTruthy()
    expect(getByText('15.0% used')).toBeTruthy()
    expect(getByText('· 300m')).toBeTruthy()
    expect(getByText('· pro')).toBeTruthy()
    expect(getByText(/Resets/)).toBeTruthy()
    expect(getByText(/Updated 15m ago/)).toBeTruthy()
    expect(getByText('Default')).toBeTruthy()
  })
})

function makeProvider(): AgentProvider {
  return {
    id: 'provider-1',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Localhost',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'OpenAI Codex',
    adapter_type: 'codex-app-server',
    permission_profile: 'unrestricted',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-04-01T11:00:00Z',
    availability_reason: null,
    capabilities: {
      ephemeral_chat: {
        state: 'available',
        reason: null,
      },
    },
    cli_command: 'codex',
    cli_args: ['app-server', '--listen', 'stdio://'],
    auth_config: {},
    cli_rate_limit: {
      provider: 'codex',
      raw: { limitId: 'codex' },
      claude_code: null,
      codex: {
        limit_id: 'codex',
        limit_name: '',
        plan_type: 'pro',
        primary: {
          used_percent: 15,
          window_minutes: 300,
          resets_at: '2026-04-01T15:30:32Z',
        },
        secondary: null,
      },
      gemini: null,
    },
    cli_rate_limit_updated_at: '2026-04-01T11:45:00Z',
    model_name: 'gpt-5.4',
    model_temperature: 0,
    model_max_tokens: 32000,
    max_parallel_runs: 1,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
  }
}
