import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { AgentProvider } from '$lib/api/contracts'
import { toastStore } from '$lib/stores/toast.svelte'
import AgentDrawer from './agent-drawer.svelte'
import { makeAgent } from './agents-page.test-helpers'

const { deleteAgent, pauseAgent, resumeAgent, updateAgent } = vi.hoisted(() => ({
  deleteAgent: vi.fn(),
  pauseAgent: vi.fn(),
  resumeAgent: vi.fn(),
  updateAgent: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  deleteAgent,
  pauseAgent,
  resumeAgent,
  updateAgent,
}))

describe('AgentDrawer', () => {
  afterEach(() => {
    cleanup()
    toastStore.clear()
    vi.clearAllMocks()
  })

  it('renames an agent through the patch API', async () => {
    const agent = makeAgent({ name: 'todo-app-coding-real-01' })
    const onUpdated = vi.fn()
    updateAgent.mockResolvedValue({ agent: {} })

    const { findByLabelText, findByDisplayValue } = render(AgentDrawer, {
      open: true,
      agent,
      providers: [makeProvider()],
      onUpdated,
    })

    await fireEvent.click(await findByLabelText('Rename agent'))

    const input = await findByDisplayValue('todo-app-coding-real-01')
    await fireEvent.input(input, { target: { value: 'todo-app-coding-real-02' } })
    await fireEvent.keyDown(input, { key: 'Enter' })

    await waitFor(() => {
      expect(updateAgent).toHaveBeenCalledWith(agent.id, { name: 'todo-app-coding-real-02' })
      expect(onUpdated).toHaveBeenCalledTimes(1)
    })
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
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'gpt-5.4',
    model_temperature: 0,
    model_max_tokens: 32000,
    max_parallel_runs: 1,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
  }
}
