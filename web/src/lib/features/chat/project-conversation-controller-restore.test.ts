import { afterEach, describe, expect, it, vi } from 'vitest'

const {
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
} = vi.hoisted(() => ({
  closeProjectConversationRuntime: vi.fn(),
  createProjectConversation: vi.fn(),
  executeProjectConversationActionProposal: vi.fn(),
  getProjectConversation: vi.fn(),
  listProjectConversationEntries: vi.fn(),
  listProjectConversations: vi.fn(),
  respondProjectConversationInterrupt: vi.fn(),
  startProjectConversationTurn: vi.fn(),
  watchProjectConversation: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
}))

import { createProjectConversationController } from './project-conversation-controller.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'

function deferredPromise<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

describe('createProjectConversationController restore flows', () => {
  afterEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('ignores late stream events after reset clears the active runtime', async () => {
    const stream = deferredPromise<void>()
    let streamHandlers:
      | {
          onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void
        }
      | undefined

    createProjectConversation.mockResolvedValue({
      conversation: { id: 'conversation-1' },
    })
    watchProjectConversation.mockImplementation(async (_conversationId, handlers) => {
      streamHandlers = handlers
      return stream.promise
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })
    closeProjectConversationRuntime.mockResolvedValue(undefined)

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Race test')
    await controller.resetConversation()

    streamHandlers?.onEvent({
      kind: 'message',
      payload: {
        type: 'text',
        content: 'late assistant reply',
      },
    })
    streamHandlers?.onEvent({
      kind: 'turn_done',
      payload: {
        conversationId: 'conversation-1',
        turnId: 'turn-1',
      },
    })

    expect(closeProjectConversationRuntime).toHaveBeenCalledWith('conversation-1')
    expect(controller.phase).toBe('idle')
    expect(controller.conversationId).toBe('')
    expect(controller.entries).toEqual([])

    stream.resolve()
  })

  it('switches conversations, reloads the matching transcript, and continues the selected session', async () => {
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Current conversation',
          providerId: 'provider-1',
          lastActivityAt: '2026-04-01T10:00:00Z',
        },
        {
          id: 'conversation-2',
          rollingSummary: 'Older discussion',
          providerId: 'provider-1',
          lastActivityAt: '2026-03-31T09:00:00Z',
        },
      ],
    })
    listProjectConversationEntries.mockImplementation(async (conversationId: string) => ({
      entries:
        conversationId === 'conversation-2'
          ? [
              {
                id: 'entry-2',
                conversationId: 'conversation-2',
                turnId: 'turn-2',
                seq: 1,
                kind: 'user_message',
                payload: { content: 'Continue the older plan' },
                createdAt: '2026-03-31T09:00:00Z',
              },
            ]
          : [
              {
                id: 'entry-1',
                conversationId: 'conversation-1',
                turnId: 'turn-1',
                seq: 1,
                kind: 'user_message',
                payload: { content: 'Current conversation' },
                createdAt: '2026-04-01T10:00:00Z',
              },
            ],
    }))
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-3', turn_index: 2, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.restore()

    expect(controller.conversations.map((conversation) => conversation.id)).toEqual([
      'conversation-1',
      'conversation-2',
    ])
    expect(controller.conversationId).toBe('conversation-1')
    expect(controller.entries).toMatchObject([{ kind: 'text', content: 'Current conversation' }])

    await controller.selectConversation('conversation-2')

    expect(controller.conversationId).toBe('conversation-2')
    expect(listProjectConversationEntries).toHaveBeenLastCalledWith('conversation-2')
    expect(watchProjectConversation).toHaveBeenLastCalledWith(
      'conversation-2',
      expect.objectContaining({
        signal: expect.any(AbortSignal),
        onEvent: expect.any(Function),
      }),
    )
    expect(controller.entries).toMatchObject([
      { kind: 'text', role: 'user', content: 'Continue the older plan' },
    ])
    expect(
      controller.entries.some(
        (entry) => entry.kind === 'text' && entry.content === 'Current conversation',
      ),
    ).toBe(false)

    await controller.sendTurn('Follow up on the older plan')

    expect(createProjectConversation).not.toHaveBeenCalled()
    expect(startProjectConversationTurn).toHaveBeenLastCalledWith(
      'conversation-2',
      'Follow up on the older plan',
    )
    expect(controller.entries).toMatchObject([
      { kind: 'text', role: 'user', content: 'Continue the older plan' },
      { kind: 'text', role: 'user', content: 'Follow up on the older plan' },
    ])
  })
})
