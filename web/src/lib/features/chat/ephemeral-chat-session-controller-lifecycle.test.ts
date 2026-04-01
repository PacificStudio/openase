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

describe('createEphemeralChatSessionController lifecycle', () => {
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
})
