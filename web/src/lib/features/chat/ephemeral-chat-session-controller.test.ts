import { afterEach, describe, expect, it, vi } from 'vitest'

const { closeChatSession, streamChatTurn } = vi.hoisted(() => ({
  closeChatSession: vi.fn(),
  streamChatTurn: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeChatSession,
  streamChatTurn,
}))

import type { AgentProvider } from '$lib/api/contracts'
import { createEphemeralChatSessionController } from './ephemeral-chat-session-controller.svelte'

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
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    cli_command: 'codex',
    cli_args: [],
    auth_config: {},
    model_name: 'gpt-5.4',
    model_temperature: 0,
    model_max_tokens: 4096,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
  },
  {
    id: 'provider-2',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Localhost',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Claude',
    adapter_type: 'claude-code-cli',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    cli_command: 'claude',
    cli_args: [],
    auth_config: {},
    model_name: 'claude-sonnet-4',
    model_temperature: 0,
    model_max_tokens: 4096,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
  },
]

describe('createEphemeralChatSessionController', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('closes the previous session and clears transcript when switching providers', async () => {
    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'done',
        payload: {
          sessionId: 'session-1',
          turnsUsed: 1,
          turnsRemaining: 9,
        },
      })
    })
    closeChatSession.mockResolvedValue(undefined)

    const controller = createEphemeralChatSessionController({
      source: 'harness_editor',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn({
      message: 'Help me tighten this harness.',
      context: {
        projectId: 'project-1',
        workflowId: 'workflow-1',
      },
    })

    expect(controller.providerId).toBe('provider-1')
    expect(controller.sessionId).toBe('session-1')
    expect(controller.entries.map((entry) => entry.content)).toEqual([
      'Help me tighten this harness.',
    ])

    await controller.selectProvider('provider-2')

    expect(closeChatSession).toHaveBeenCalledWith('session-1')
    expect(controller.providerId).toBe('provider-2')
    expect(controller.sessionId).toBe('')
    expect(controller.entries).toEqual([])
  })

  it('closes the active session when the embedding panel is hidden', async () => {
    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'done',
        payload: {
          sessionId: 'session-1',
          turnsUsed: 1,
          turnsRemaining: 9,
        },
      })
    })
    closeChatSession.mockResolvedValue(undefined)

    const controller = createEphemeralChatSessionController({
      source: 'harness_editor',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn({
      message: 'Explain this workflow.',
      context: {
        projectId: 'project-1',
        workflowId: 'workflow-1',
      },
    })

    await controller.dispose()

    expect(closeChatSession).toHaveBeenCalledWith('session-1')
    expect(controller.sessionId).toBe('')
  })
})
