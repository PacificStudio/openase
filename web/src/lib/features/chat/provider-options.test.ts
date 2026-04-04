import { describe, expect, it } from 'vitest'

import type { AgentProvider } from '$lib/api/contracts'
import { listProviderCapabilityProviders, pickDefaultProviderCapability } from './provider-options'

const providers: AgentProvider[] = [
  {
    id: 'provider-custom',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Localhost',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Custom',
    adapter_type: 'custom',
    permission_profile: 'unrestricted',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    capabilities: {
      ephemeral_chat: { state: 'unsupported', reason: 'unsupported_adapter' },
      harness_ai: { state: 'unsupported', reason: 'unsupported_adapter' },
      skill_ai: { state: 'unsupported', reason: 'skill_ai_requires_codex' },
    },
    cli_command: 'custom-chat',
    cli_args: [],
    auth_config: {},
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'manual',
    model_temperature: 0,
    model_max_tokens: 4096,
    max_parallel_runs: 2,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
  },
  {
    id: 'provider-claude',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Localhost',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Claude',
    adapter_type: 'claude-code-cli',
    permission_profile: 'unrestricted',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    capabilities: {
      ephemeral_chat: { state: 'available', reason: null },
      harness_ai: { state: 'available', reason: null },
      skill_ai: { state: 'unsupported', reason: 'skill_ai_requires_codex' },
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
    id: 'provider-codex-remote',
    organization_id: 'org-1',
    machine_id: 'machine-2',
    machine_name: 'builder-01',
    machine_host: '10.0.0.24',
    machine_status: 'online',
    machine_ssh_user: 'openase',
    machine_workspace_root: '/srv/workspace',
    name: 'Codex Remote',
    adapter_type: 'codex-app-server',
    permission_profile: 'unrestricted',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    capabilities: {
      ephemeral_chat: { state: 'available', reason: null },
      harness_ai: { state: 'unsupported', reason: 'remote_machine_not_supported' },
      skill_ai: { state: 'unsupported', reason: 'remote_machine_not_supported' },
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
    id: 'provider-codex-local',
    organization_id: 'org-1',
    machine_id: 'machine-3',
    machine_name: 'local',
    machine_host: 'local',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Codex Local',
    adapter_type: 'codex-app-server',
    permission_profile: 'unrestricted',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    capabilities: {
      ephemeral_chat: { state: 'available', reason: null },
      harness_ai: { state: 'available', reason: null },
      skill_ai: { state: 'available', reason: null },
    },
    cli_command: 'claude',
    cli_args: [],
    auth_config: {},
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'claude-sonnet-4',
    model_temperature: 0,
    model_max_tokens: 4096,
    max_parallel_runs: 2,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
  },
]

describe('provider-options', () => {
  it('filters providers per surface and falls back to the first available match', () => {
    const ephemeralChatProviders = listProviderCapabilityProviders(providers, 'ephemeral_chat')
    expect(ephemeralChatProviders.map((provider) => provider.id)).toEqual([
      'provider-claude',
      'provider-codex-remote',
      'provider-codex-local',
    ])

    const harnessProviders = listProviderCapabilityProviders(providers, 'harness_ai')
    expect(harnessProviders.map((provider) => provider.id)).toEqual([
      'provider-claude',
      'provider-codex-local',
    ])

    const skillProviders = listProviderCapabilityProviders(providers, 'skill_ai')
    expect(skillProviders.map((provider) => provider.id)).toEqual(['provider-codex-local'])

    expect(
      pickDefaultProviderCapability(harnessProviders, 'provider-codex-remote', 'harness_ai'),
    ).toBe('provider-claude')
    expect(pickDefaultProviderCapability(skillProviders, 'provider-claude', 'skill_ai')).toBe(
      'provider-codex-local',
    )
  })
})
