import { waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

const {
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  interruptProjectConversationTurn,
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
  interruptProjectConversationTurn: vi.fn(),
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
  interruptProjectConversationTurn,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversationMuxStream: vi.fn(),
}))

vi.mock('./project-conversation-event-bus', () => ({
  watchProjectConversationMux,
}))

import { createProjectConversationController } from './project-conversation-controller.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'
import {
  createWorkspaceDiff,
  deferredPromise,
  resolvedMuxSubscription,
  startedTurnResponse,
} from './project-conversation-controller.test-helpers'

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
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

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

  it('stops an active turn and returns the tab to idle after interrupted terminal events', async () => {
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
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())
    interruptProjectConversationTurn.mockResolvedValue(undefined)

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Stop after the partial reply')
    expect(controller.phase).toBe('awaiting_reply')

    streamHandlers?.onEvent({
      kind: 'message',
      payload: {
        type: 'text',
        content: 'Partial assistant reply',
      },
    })
    expect(controller.entries).toMatchObject([
      { kind: 'text', role: 'user', content: 'Stop after the partial reply' },
      { kind: 'text', role: 'assistant', content: 'Partial assistant reply' },
    ])

    await controller.stopTurn()

    expect(interruptProjectConversationTurn).toHaveBeenCalledWith('conversation-1')
    expect(controller.phase).toBe('stopping_turn')

    streamHandlers?.onEvent({
      kind: 'interrupted',
      payload: {
        conversationId: 'conversation-1',
        turnId: 'turn-1',
        message: 'Turn stopped by user.',
        reason: 'stopped_by_user',
      },
    })
    expect(controller.phase).toBe('idle')
    expect(
      controller.entries.some(
        (entry) => entry.kind === 'text' && entry.content === 'Partial assistant reply',
      ),
    ).toBe(true)

    streamHandlers?.onEvent({
      kind: 'session',
      payload: {
        conversationId: 'conversation-1',
        runtimeState: 'ready',
        providerStatus: 'ready',
        providerActiveFlags: [],
      },
    })
    expect(controller.phase).toBe('idle')

    await controller.sendTurn('Continue after stop')
    expect(startProjectConversationTurn).toHaveBeenLastCalledWith('conversation-1', {
      message: 'Continue after stop',
      focus: undefined,
    })
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
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

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
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

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
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

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
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

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
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

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
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

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
          onRetrying?: () => void
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
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())
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

    streamHandlers?.onRetrying?.()
    expect(controller.phase).toBe('connecting_stream')

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
})
