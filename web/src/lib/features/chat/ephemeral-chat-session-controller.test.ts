import { afterEach, describe, expect, it, vi } from 'vitest'

const { closeChatSession, streamChatTurn } = vi.hoisted(() => ({
  closeChatSession: vi.fn(),
  streamChatTurn: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeChatSession,
  streamChatTurn,
}))

import { createEphemeralChatSessionController } from './ephemeral-chat-session-controller.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'

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

  it('surfaces a transient stream interruption clearly and finalizes any partial assistant reply', async () => {
    const onError = vi.fn()
    streamChatTurn.mockImplementation(async (_request, handlers) => {
      handlers.onEvent({
        kind: 'session',
        payload: {
          sessionId: 'session-1',
        },
      })
      handlers.onEvent({
        kind: 'message',
        payload: {
          type: 'text',
          content: 'Partial reply',
        },
      })
      throw new TypeError('network error')
    })

    const controller = createEphemeralChatSessionController({
      getSource: () => 'project_sidebar',
      onError,
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn({
      message: 'What changed?',
      context: {
        projectId: 'project-1',
      },
    })

    expect(onError).toHaveBeenCalledWith(
      'The reply stopped mid-stream because the browser connection closed before OpenASE sent the final completion event. The partial reply was kept above. This usually means the OpenASE server restarted during the turn or the network connection reset. Retry the request.',
    )
    expect(controller.sessionId).toBe('session-1')
    expect(controller.pending).toBe(false)
    expect(
      controller.entries
        .filter((entry) => entry.kind === 'text')
        .map((entry) => ({
          role: entry.role,
          content: entry.content,
          streaming: entry.streaming,
        })),
    ).toEqual([
      {
        role: 'user',
        content: 'What changed?',
        streaming: false,
      },
      {
        role: 'assistant',
        content: 'Partial reply',
        streaming: false,
      },
    ])
  })
})
