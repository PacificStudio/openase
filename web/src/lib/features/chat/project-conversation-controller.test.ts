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
}))

const { watchProjectConversationMux } = vi.hoisted(() => ({
  watchProjectConversationMux: vi.fn(),
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
}))

vi.mock('./project-conversation-event-bus', () => ({
  watchProjectConversationMux,
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

function resolvedMuxSubscription() {
  return {
    stream: Promise.resolve(),
    connected: Promise.resolve(),
  }
}

function seedProjectConversationTabsStorage(
  tabs: Array<{
    conversationId: string
    providerId: string
    draft?: string
    projectId?: string
  }>,
  activeTabIndex: number,
) {
  window.localStorage.setItem(
    'openase.project-conversation.global',
    JSON.stringify({
      tabs: tabs.map((tab) => ({
        projectId: tab.projectId ?? 'project-1',
        projectName: 'Project 1',
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

  it('creates a new conversation for the active tab and waits for the stream before the first turn', async () => {
    const stream = deferredPromise<void>()
    const connected = deferredPromise<void>()

    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversationMux.mockReturnValue({
      stream: stream.promise,
      connected: connected.promise,
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    const sendTurn = controller.sendTurn('Summarize this project.')
    await Promise.resolve()

    expect(createProjectConversation).toHaveBeenCalledWith({
      providerId: 'provider-1',
      projectId: 'project-1',
    })
    expect(watchProjectConversationMux).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: 'project-1',
        conversationId: 'conversation-1',
        signal: expect.any(AbortSignal),
        onEvent: expect.any(Function),
      }),
    )
    expect(startProjectConversationTurn).not.toHaveBeenCalled()

    connected.resolve()
    await sendTurn

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
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Help me figure out what to change here.', {
      kind: 'ticket',
      projectId: 'project-1',
      ticketId: 'ticket-1',
      ticketIdentifier: 'T-123',
      ticketTitle: 'Investigate CI failure',
      ticketStatus: 'In Review',
      selectedArea: 'detail',
    })

    expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
      message: 'Help me figure out what to change here.',
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

  it('keeps the active tab pending when session updates report an active provider', async () => {
    let streamHandlers:
      | {
          onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void
        }
      | undefined

    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversationMux.mockImplementation((params) => {
      streamHandlers = params
      return resolvedMuxSubscription()
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Keep thinking')
    expect(controller.phase).toBe('awaiting_reply')
    expect(controller.pending).toBe(true)

    streamHandlers?.onEvent({
      kind: 'session',
      payload: {
        conversationId: 'conversation-1',
        runtimeState: 'ready',
        providerStatus: 'active',
        providerActiveFlags: ['active'],
      },
    })

    expect(controller.phase).toBe('awaiting_reply')
    expect(controller.pending).toBe(true)
  })

  it('drops the active tab back to idle when the backend auto-releases the runtime after completion', async () => {
    let streamHandlers:
      | {
          onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void
        }
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
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
    watchProjectConversationMux.mockImplementation((params) => {
      streamHandlers = params
      return resolvedMuxSubscription()
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Finish this turn and release the runtime')
    expect(controller.phase).toBe('awaiting_reply')

    streamHandlers?.onEvent({
      kind: 'session',
      payload: {
        conversationId: 'conversation-1',
        runtimeState: 'inactive',
        providerAnchorKind: 'thread',
        providerAnchorId: 'thread-1',
        providerTurnId: 'turn-1',
        providerStatus: 'notLoaded',
        providerActiveFlags: [],
      },
    })
    streamHandlers?.onEvent({
      kind: 'turn_done',
      payload: {
        conversationId: 'conversation-1',
        turnId: 'turn-1',
      },
    })

    expect(controller.phase).toBe('idle')
    expect(controller.pending).toBe(false)
  })

  it('keeps a running tab pending after switching away and hydrating it again', async () => {
    const streamHandlers = new Map<
      string,
      {
        onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void
      }
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
      .mockResolvedValue(createWorkspaceDiff('conversation-1'))
      .mockResolvedValue(createWorkspaceDiff('conversation-2'))
    listProjectConversationEntries.mockResolvedValue({
      entries: [
        {
          id: 'entry-1',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'user_message',
          payload: { content: 'Keep thinking' },
          createdAt: '2026-04-01T10:00:00Z',
        },
        {
          id: 'entry-2',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 2,
          kind: 'assistant_text_delta',
          payload: { content: 'Hidden tab chunk.' },
          createdAt: '2026-04-01T10:00:01Z',
        },
      ],
    })
    watchProjectConversationMux.mockImplementation((params) => {
      streamHandlers.set(params.conversationId, params)
      return resolvedMuxSubscription()
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Keep thinking')
    const firstTabId = controller.activeTabId
    expect(controller.phase).toBe('awaiting_reply')

    controller.createTab()
    await controller.sendTurn('Work in parallel')
    expect(controller.activeTabId).not.toBe(firstTabId)

    streamHandlers.get('conversation-1')?.onEvent({
      kind: 'message',
      payload: {
        type: 'text',
        content: 'Hidden tab chunk.',
      },
    })

    const backgroundTab = controller.tabs.find((tab) => tab.id === firstTabId)
    expect(backgroundTab?.needsHydration).toBe(true)
    expect(backgroundTab?.phase).toBe('awaiting_reply')

    controller.selectTab(firstTabId)

    await waitFor(() => {
      expect(listProjectConversationEntries).toHaveBeenCalledWith('conversation-1')
    })

    expect(controller.activeTabId).toBe(firstTabId)
    expect(controller.phase).toBe('awaiting_reply')
    expect(controller.pending).toBe(true)
    expect(
      controller.entries.some(
        (entry) =>
          entry.kind === 'text' &&
          entry.role === 'assistant' &&
          entry.content.includes('Hidden tab chunk.'),
      ),
    ).toBe(true)
  })

  it('blocks duplicate sends while the active tab is creating its first conversation', async () => {
    const create = deferredPromise<{
      conversation: { id: string; providerId: string; lastActivityAt: string }
    }>()

    createProjectConversation.mockReturnValue(create.promise)
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
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
    watchProjectConversationMux.mockImplementation((params) => {
      streamHandlers.set(params.conversationId, params)
      return resolvedMuxSubscription()
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
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

  it('reconciles a completed turn when the mux stream reconnects after progress but before turn_done arrives', async () => {
    const stream = deferredPromise<void>()
    let streamHandlers:
      | {
          onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void
          onReconnect?: () => void
        }
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
    watchProjectConversationMux.mockImplementation((params) => {
      streamHandlers = params
      return {
        stream: stream.promise,
        connected: Promise.resolve(),
      }
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
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
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

    streamHandlers?.onReconnect?.()
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
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
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
    watchProjectConversationMux.mockImplementation((params) => {
      streamHandlers.set(params.conversationId, params)
      return resolvedMuxSubscription()
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })
    respondProjectConversationInterrupt.mockResolvedValue({
      interrupt: {},
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
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
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
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
    expect(controller.tabs[0]?.needsHydration).toBe(true)
    expect(controller.tabs[1]?.needsHydration).toBe(false)
    expect(controller.conversationId).toBe('conversation-1')
    expect(controller.entries).toMatchObject([{ kind: 'text', content: 'Current conversation' }])
    expect(listProjectConversationEntries).toHaveBeenCalledTimes(1)
    expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledTimes(1)
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
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
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
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
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
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('First provider')
    expect(controller.phase).toBe('awaiting_reply')

    const streamHandler = watchProjectConversationMux.mock.calls[0]?.[0]?.onEvent as
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
