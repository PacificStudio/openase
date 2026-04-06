import { afterEach, describe, expect, it, vi } from 'vitest'

const { closeChatSession, streamChatTurn } = vi.hoisted(() => ({
  closeChatSession: vi.fn(),
  streamChatTurn: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeChatSession,
  streamChatTurn,
  watchProjectConversationMuxStream: vi.fn(),
}))

import { createEphemeralChatSessionController } from './ephemeral-chat-session-controller.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'

describe('createEphemeralChatSessionController', () => {
  afterEach(() => {
    vi.clearAllMocks()
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
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
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
    expect(consoleError).toHaveBeenCalledWith(
      'Ephemeral chat turn failed',
      expect.objectContaining({
        source: 'project_sidebar',
        providerId: 'provider-1',
        sessionId: 'session-1',
        context: {
          projectId: 'project-1',
        },
        streamStarted: true,
        partialReplyReceived: true,
        error: expect.any(TypeError),
        errorMessage:
          'The reply stopped mid-stream because the browser connection closed before OpenASE sent the final completion event. The partial reply was kept above. This usually means the OpenASE server restarted during the turn or the network connection reset. Retry the request.',
      }),
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
