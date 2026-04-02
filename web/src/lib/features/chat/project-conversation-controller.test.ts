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

describe('createProjectConversationController', () => {
  afterEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('creates a new conversation for the active tab and submits the first turn without waiting for the stream', async () => {
    const stream = deferredPromise<void>()

    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    watchProjectConversation.mockReturnValue(stream.promise)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Summarize this project.')

    expect(createProjectConversation).toHaveBeenCalledWith({
      providerId: 'provider-1',
      projectId: 'project-1',
    })
    expect(watchProjectConversation).toHaveBeenCalledWith(
      'conversation-1',
      expect.objectContaining({
        signal: expect.any(AbortSignal),
        onEvent: expect.any(Function),
      }),
    )
    expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
      message: 'Summarize this project.',
      focus: undefined,
    })
    expect(controller.phase).toBe('awaiting_reply')
    expect(controller.entries).toMatchObject([
      { kind: 'text', role: 'user', content: 'Summarize this project.' },
    ])
    expect(controller.tabs).toHaveLength(1)
    expect(controller.tabs[0]?.conversationId).toBe('conversation-1')

    stream.resolve()
  })

  it('passes per-turn focus metadata through to the project conversation turn request', async () => {
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('帮我看看这里要怎么改', {
      kind: 'ticket',
      projectId: 'project-1',
      ticketId: 'ticket-1',
      ticketIdentifier: 'T-123',
      ticketTitle: 'Investigate CI failure',
      ticketStatus: 'In Review',
      selectedArea: 'detail',
    })

    expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
      message: '帮我看看这里要怎么改',
      focus: {
        kind: 'ticket',
        projectId: 'project-1',
        ticketId: 'ticket-1',
        ticketIdentifier: 'T-123',
        ticketTitle: 'Investigate CI failure',
        ticketStatus: 'In Review',
        selectedArea: 'detail',
      },
    })
  })

  it('blocks duplicate sends while the active tab is creating its first conversation', async () => {
    const create = deferredPromise<{
      conversation: { id: string; providerId: string; lastActivityAt: string }
    }>()

    createProjectConversation.mockReturnValue(create.promise)
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    const firstSend = controller.sendTurn('First question')
    expect(controller.phase).toBe('creating_conversation')
    expect(controller.inputDisabled).toBe(true)

    await controller.sendTurn('Second question')

    expect(createProjectConversation).toHaveBeenCalledTimes(1)
    expect(startProjectConversationTurn).not.toHaveBeenCalled()

    create.resolve({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    await firstSend

    expect(startProjectConversationTurn).toHaveBeenCalledTimes(1)
    expect(controller.entries.filter((entry) => entry.kind === 'text')).toMatchObject([
      { role: 'user', content: 'First question' },
    ])
  })

  it('blocks follow-up sends only for the same tab while an interrupt is pending', async () => {
    const streamHandlers = new Map<
      string,
      { onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void }
    >()

    createProjectConversation
      .mockResolvedValueOnce({
        conversation: {
          id: 'conversation-1',
          providerId: 'provider-1',
          lastActivityAt: '2026-04-01T10:00:00Z',
        },
      })
      .mockResolvedValueOnce({
        conversation: {
          id: 'conversation-2',
          providerId: 'provider-1',
          lastActivityAt: '2026-04-01T10:05:00Z',
        },
      })
    watchProjectConversation.mockImplementation(async (conversationId, handlers) => {
      streamHandlers.set(conversationId, handlers)
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })
    respondProjectConversationInterrupt.mockResolvedValue({
      interrupt: {},
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Need approval')
    streamHandlers.get('conversation-1')?.onEvent({
      kind: 'interrupt_requested',
      payload: {
        interruptId: 'interrupt-1',
        provider: 'codex',
        kind: 'user_input',
        options: [],
        payload: {
          questions: [
            {
              id: 'approval',
              question: 'Approve the next step?',
            },
          ],
        },
      },
    })

    expect(controller.phase).toBe('awaiting_interrupt')
    expect(controller.hasPendingInterrupt).toBe(true)
    expect(controller.inputDisabled).toBe(true)

    await controller.sendTurn('This should stay blocked')
    expect(startProjectConversationTurn).toHaveBeenCalledTimes(1)

    controller.createTab()
    const secondTabId = controller.tabs.find((tab) => tab.conversationId === '')?.id ?? ''
    controller.selectTab(secondTabId)
    expect(controller.inputDisabled).toBe(false)

    await controller.sendTurn('Independent tab keeps working')

    expect(startProjectConversationTurn).toHaveBeenNthCalledWith(2, 'conversation-2', {
      message: 'Independent tab keeps working',
      focus: undefined,
    })

    const firstTabId =
      controller.tabs.find((tab) => tab.conversationId === 'conversation-1')?.id ?? ''
    controller.selectTab(firstTabId)
    await controller.respondInterrupt({
      interruptId: 'interrupt-1',
      answer: {
        approval: {
          answers: ['yes'],
        },
      },
    })

    expect(respondProjectConversationInterrupt).toHaveBeenCalledWith(
      'conversation-1',
      'interrupt-1',
      {
        answer: {
          approval: {
            answers: ['yes'],
          },
        },
        decision: undefined,
      },
    )
  })

  it('restores multiple tabs from storage, preserves the selected tab, and drops missing conversations', async () => {
    window.localStorage.setItem(
      'openase.project-conversation.project-1.provider-1',
      JSON.stringify({
        conversationIds: ['conversation-2', 'missing-conversation', 'conversation-1'],
        activeConversationId: 'conversation-1',
      }),
    )

    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Current conversation',
          lastActivityAt: '2026-04-01T10:00:00Z',
          providerId: 'provider-1',
        },
        {
          id: 'conversation-2',
          rollingSummary: 'Older discussion',
          lastActivityAt: '2026-03-31T09:00:00Z',
          providerId: 'provider-1',
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

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.restore()

    expect(controller.tabs).toHaveLength(2)
    expect(controller.tabs.map((tab) => tab.conversationId)).toEqual([
      'conversation-2',
      'conversation-1',
    ])
    expect(controller.tabs.every((tab) => tab.restored)).toBe(true)
    expect(controller.conversationId).toBe('conversation-1')
    expect(controller.entries).toMatchObject([{ kind: 'text', content: 'Current conversation' }])

    const secondTabId =
      controller.tabs.find((tab) => tab.conversationId === 'conversation-2')?.id ?? ''
    controller.selectTab(secondTabId)

    expect(controller.conversationId).toBe('conversation-2')
    expect(controller.entries).toMatchObject([{ kind: 'text', content: 'Continue the older plan' }])
    expect(listProjectConversationEntries).toHaveBeenCalledTimes(2)
  })

  it('closes only the selected tab and keeps the remaining tab active', async () => {
    createProjectConversation
      .mockResolvedValueOnce({
        conversation: {
          id: 'conversation-1',
          providerId: 'provider-1',
          lastActivityAt: '2026-04-01T10:00:00Z',
        },
      })
      .mockResolvedValueOnce({
        conversation: {
          id: 'conversation-2',
          providerId: 'provider-1',
          lastActivityAt: '2026-04-01T10:05:00Z',
        },
      })
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('First tab')
    controller.createTab()
    const secondTabId = controller.tabs.find((tab) => tab.conversationId === '')?.id ?? ''
    controller.selectTab(secondTabId)
    await controller.sendTurn('Second tab')

    const firstTabId =
      controller.tabs.find((tab) => tab.conversationId === 'conversation-1')?.id ?? ''
    controller.closeTab(firstTabId)

    expect(controller.tabs).toHaveLength(1)
    expect(controller.conversationId).toBe('conversation-2')
    expect(controller.entries).toMatchObject([{ kind: 'text', content: 'Second tab' }])
  })
})
