import { afterEach, describe, expect, it, vi } from 'vitest'
import { waitFor } from '@testing-library/svelte'

const {
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
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
  getProjectConversationWorkspaceDiff: vi.fn(),
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
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
}))

import { createProjectConversationController } from './project-conversation-controller.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'

function createWorkspaceDiff(conversationId: string) {
  return {
    workspaceDiff: {
      conversationId,
      workspacePath: `/tmp/${conversationId}`,
      dirty: false,
      reposChanged: 0,
      filesChanged: 0,
      added: 0,
      removed: 0,
      repos: [],
    },
  }
}

function deferredPromise<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

function seedProjectConversationTabsStorage(
  tabs: Array<{
    conversationId: string
    providerId: string
    draft?: string
  }>,
  activeTabIndex: number,
) {
  window.localStorage.setItem(
    'openase.project-conversation.project-1',
    JSON.stringify({
      tabs: tabs.map((tab) => ({
        conversationId: tab.conversationId,
        providerId: tab.providerId,
        draft: tab.draft ?? '',
      })),
      activeTabIndex,
    }),
  )
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
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
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
    expect(controller.inputDisabled).toBe(false)
    expect(controller.sendDisabled).toBe(true)
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
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
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
    expect(controller.inputDisabled).toBe(false)
    expect(controller.sendDisabled).toBe(true)

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

  it('queues a follow-up turn while busy and sends it once the tab is idle again', async () => {
    const streamHandlers = new Map<
      string,
      { onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void }
    >()

    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockImplementation(async (conversationId, handlers) => {
      streamHandlers.set(conversationId, handlers)
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('First question')

    expect(controller.phase).toBe('awaiting_reply')
    expect(controller.canQueueTurn).toBe(true)
    expect(controller.enqueueTurn('Second question')).toBe(true)
    expect(controller.queuedTurns).toMatchObject([{ message: 'Second question' }])

    streamHandlers.get('conversation-1')?.onEvent({
      kind: 'turn_done',
      payload: {},
    })

    expect(controller.phase).toBe('idle')
    expect(controller.canQueueTurn).toBe(true)
    await controller.sendNextQueuedTurn()

    expect(startProjectConversationTurn).toHaveBeenNthCalledWith(2, 'conversation-1', {
      message: 'Second question',
      focus: undefined,
    })
    expect(controller.queuedTurns).toHaveLength(0)
  })

  it('reconciles a completed turn when the stream closes after progress but before turn_done arrives', async () => {
    const stream = deferredPromise<void>()
    let streamHandlers:
      | { onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void }
      | undefined

    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockImplementation(async (_conversationId, handlers) => {
      streamHandlers = handlers
      await stream.promise
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })
    getProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:01:00Z',
        runtimePrincipal: {
          runtimeState: 'ready',
        },
      },
    })
    listProjectConversationEntries.mockResolvedValue({
      entries: [
        {
          id: 'entry-1',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'user_message',
          payload: { content: 'Finish this turn' },
          createdAt: '2026-04-01T10:00:00Z',
        },
        {
          id: 'entry-2',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 2,
          kind: 'assistant_text_delta',
          payload: { content: 'Recovered reply' },
          createdAt: '2026-04-01T10:01:00Z',
        },
      ],
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Finish this turn')
    streamHandlers?.onEvent({
      kind: 'message',
      payload: {
        type: 'text',
        content: 'Partial reply',
      },
    })

    stream.resolve()

    await waitFor(() => {
      expect(controller.phase).toBe('idle')
    })

    expect(getProjectConversation).toHaveBeenCalledWith('conversation-1')
    expect(listProjectConversationEntries).toHaveBeenCalledWith('conversation-1')
    expect(
      controller.entries.some(
        (entry) =>
          entry.kind === 'text' &&
          entry.role === 'assistant' &&
          entry.content === 'Recovered reply',
      ),
    ).toBe(true)
  })

  it('does not crash when a conversation arrives without lastActivityAt', async () => {
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        updatedAt: '2026-04-01T10:00:00Z',
        createdAt: '2026-04-01T09:00:00Z',
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

    await expect(
      controller.sendTurn('Still works without activity timestamp'),
    ).resolves.toBeUndefined()

    expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
      message: 'Still works without activity timestamp',
      focus: undefined,
    })
    expect(controller.tabs[0]?.conversationId).toBe('conversation-1')
    expect(controller.phase).toBe('awaiting_reply')
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
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-2'))
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
    expect(controller.sendDisabled).toBe(true)

    await controller.sendTurn('This should stay blocked')
    expect(startProjectConversationTurn).toHaveBeenCalledTimes(1)

    controller.createTab()
    const secondTabId = controller.tabs.find((tab) => tab.conversationId === '')?.id ?? ''
    controller.selectTab(secondTabId)
    expect(controller.inputDisabled).toBe(false)
    expect(controller.sendDisabled).toBe(false)

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

  it('restores persisted tabs, drops missing ones, and keeps the selected tab active', async () => {
    seedProjectConversationTabsStorage(
      [
        { conversationId: 'conversation-2', providerId: 'provider-1' },
        { conversationId: 'missing-conversation', providerId: 'provider-1' },
        { conversationId: 'conversation-1', providerId: 'provider-1' },
      ],
      2,
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
    getProjectConversationWorkspaceDiff.mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
    listProjectConversationEntries.mockImplementation(async (conversationId: string) => ({
      entries:
        conversationId === 'conversation-1'
          ? [
              {
                id: 'entry-1',
                conversationId: 'conversation-1',
                turnId: 'turn-1',
                seq: 1,
                kind: 'user_message',
                payload: { content: 'Current conversation' },
                createdAt: '2026-04-01T10:00:00Z',
              },
            ]
          : [],
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
    expect(listProjectConversationEntries).toHaveBeenCalledTimes(2)
    expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledTimes(2)
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
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-2'))
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

  it('retargets a blank tab when switching provider before the conversation starts', async () => {
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-2',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    expect(controller.tabs).toHaveLength(1)
    expect(controller.providerId).toBe('provider-1')

    await controller.selectProvider('provider-2')

    expect(controller.tabs).toHaveLength(1)
    expect(controller.providerId).toBe('provider-2')
    expect(controller.tabs[0]?.providerId).toBe('provider-2')

    await controller.sendTurn('Use Claude instead')

    expect(createProjectConversation).toHaveBeenCalledWith({
      providerId: 'provider-2',
      projectId: 'project-1',
    })
  })

  it('opens a new blank tab when switching provider after a conversation has started', async () => {
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
          providerId: 'provider-2',
          lastActivityAt: '2026-04-01T10:05:00Z',
        },
      })
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-2'))
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('First provider')
    expect(controller.phase).toBe('awaiting_reply')

    const streamHandler = watchProjectConversation.mock.calls[0]?.[1]?.onEvent as
      | ((event: { kind: string; payload: Record<string, unknown> }) => void)
      | undefined
    streamHandler?.({
      kind: 'turn_done',
      payload: {},
    })

    expect(controller.phase).toBe('idle')
    await controller.selectProvider('provider-2')

    expect(controller.tabs).toHaveLength(2)
    expect(controller.providerId).toBe('provider-2')
    expect(controller.tabs.map((tab) => tab.providerId)).toEqual(['provider-1', 'provider-2'])
    expect(controller.conversationId).toBe('')

    await controller.sendTurn('Second provider')

    expect(createProjectConversation).toHaveBeenNthCalledWith(2, {
      providerId: 'provider-2',
      projectId: 'project-1',
    })
    expect(controller.conversationId).toBe('conversation-2')
  })
})
