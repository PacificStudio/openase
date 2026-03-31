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
    capabilities: {
      ephemeral_chat: {
        state: 'available',
        reason: null,
      },
    },
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
    capabilities: {
      ephemeral_chat: {
        state: 'available',
        reason: null,
      },
    },
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
      getSource: () => 'harness_editor',
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
    expect(
      controller.entries.filter((entry) => entry.kind === 'text').map((entry) => entry.content),
    ).toEqual([
      'Help me tighten this harness.',
      'Session budget: 1/10 turns used, 9 remaining. Spend unavailable for this provider; the chat budget cap remains $2.00.',
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
        kind: 'session',
        payload: {
          sessionId: 'session-1',
        },
      })
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
      getSource: () => 'harness_editor',
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

  it('closes a first-turn session before the stream completes when reset is requested', async () => {
    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: {
          sessionId: 'session-1',
        },
      })

      await new Promise<void>((resolve) => {
        handlers.signal?.addEventListener(
          'abort',
          () => {
            resolve()
          },
          { once: true },
        )
      })
    })
    closeChatSession.mockResolvedValue(undefined)

    const controller = createEphemeralChatSessionController({
      getSource: () => 'harness_editor',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    const sendTurn = controller.sendTurn({
      message: 'Start a new session.',
      context: {
        projectId: 'project-1',
        workflowId: 'workflow-1',
      },
    })

    expect(controller.sessionId).toBe('session-1')

    await controller.resetConversation()
    await sendTurn

    expect(closeChatSession).toHaveBeenCalledWith('session-1')
    expect(controller.sessionId).toBe('')
    expect(controller.entries).toEqual([])
    expect(controller.pending).toBe(false)
  })

  it('records provider-reported spend in the usage summary after each completed turn', async () => {
    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'done',
        payload: {
          sessionId: 'session-1',
          turnsUsed: 2,
          costUSD: 0.37,
        },
      })
    })

    const controller = createEphemeralChatSessionController({
      getSource: () => 'project_sidebar',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn({
      message: 'Summarize the project state.',
      context: {
        projectId: 'project-1',
      },
    })

    expect(
      controller.entries.filter((entry) => entry.kind === 'text').map((entry) => entry.content),
    ).toEqual([
      'Summarize the project state.',
      'Project conversation: 2 turns so far. Current spend $0.37.',
    ])
  })

  it('groups streamed assistant text into one mutable transcript entry', async () => {
    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'text',
          content: '## Summary\n\n',
        },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'text',
          content: '- Item one\n',
        },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'text',
          content: '- Item two',
        },
      })
      handlers.onEvent({
        kind: 'done',
        payload: {
          sessionId: 'session-1',
          turnsUsed: 1,
        },
      })
    })

    const controller = createEphemeralChatSessionController({
      getSource: () => 'project_sidebar',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn({
      message: 'Summarize the project state.',
      context: {
        projectId: 'project-1',
      },
    })

    const assistantEntries = controller.entries.filter(
      (entry) => entry.kind === 'text' && entry.role === 'assistant',
    )
    expect(assistantEntries).toHaveLength(1)
    expect(assistantEntries[0]).toMatchObject({
      content: '## Summary\n\n- Item one\n- Item two',
      streaming: false,
    })
  })

  it('falls back from an unsupported default provider to the first available chat-capable provider', () => {
    const controller = createEphemeralChatSessionController({
      getSource: () => 'project_sidebar',
    })

    controller.syncProviders(
      [
        {
          ...providerFixtures[0],
          id: 'provider-custom',
          name: 'Custom',
          adapter_type: 'custom',
          capabilities: {
            ephemeral_chat: {
              state: 'unsupported',
              reason: 'unsupported_adapter',
            },
          },
        },
        providerFixtures[1],
      ],
      'provider-custom',
    )

    expect(controller.providerId).toBe('provider-2')
  })
})
