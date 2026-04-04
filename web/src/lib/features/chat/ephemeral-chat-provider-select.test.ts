import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import type { AgentProvider } from '$lib/api/contracts'
import EphemeralChatProviderSelect from './ephemeral-chat-provider-select.svelte'

const providerFixtures: AgentProvider[] = [
  {
    id: 'provider-1',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Localhost',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Codex',
    adapter_type: 'codex-app-server',
    permission_profile: 'unrestricted',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    capabilities: {
      ephemeral_chat: {
        state: 'available',
        reason: null,
      },
    },
    cli_command: 'codex',
    cli_args: [],
    auth_config: {},
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'gpt-5.4',
    model_temperature: 0,
    model_max_tokens: 4096,
    max_parallel_runs: 2,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
  },
  {
    id: 'provider-2',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Localhost',
    machine_host: '127.0.0.1',
    machine_status: 'offline',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Gemini',
    adapter_type: 'gemini-cli',
    permission_profile: 'unrestricted',
    availability_state: 'unavailable',
    available: false,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: 'machine_offline',
    capabilities: {
      ephemeral_chat: {
        state: 'unavailable',
        reason: 'machine_offline',
      },
    },
    cli_command: 'gemini',
    cli_args: [],
    auth_config: {},
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'gemini-2.5-pro',
    model_temperature: 0,
    model_max_tokens: 4096,
    max_parallel_runs: 2,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
  },
]

describe('EphemeralChatProviderSelect', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
  })

  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(async () => {
    cleanup()
    await vi.runOnlyPendingTimersAsync()
    vi.useRealTimers()
  })

  it('shows provider contract details and keeps unavailable providers visible but disabled', async () => {
    const { getByLabelText, getByText } = render(EphemeralChatProviderSelect, {
      props: {
        providers: providerFixtures,
        providerId: 'provider-1',
      },
    })

    expect(getByText('gpt-5.4')).toBeTruthy()

    const trigger = getByLabelText('Chat model')
    await fireEvent.pointerDown(trigger)
    await fireEvent.keyDown(trigger, { key: 'ArrowDown' })
    await vi.runOnlyPendingTimersAsync()

    expect(getByText('Codex · codex-app-server')).toBeTruthy()
    expect(getByText('Ready')).toBeTruthy()
    expect(getByText('Gemini · gemini-cli')).toBeTruthy()
    expect(getByText('gemini-2.5-pro')).toBeTruthy()
    expect(getByText('Unavailable')).toBeTruthy()
    expect(getByText('Host machine is offline.')).toBeTruthy()

    const unavailableOption = getByText('Gemini · gemini-cli').closest('[data-slot="select-item"]')
    expect(unavailableOption?.hasAttribute('data-disabled')).toBe(true)
  })
})
