/* eslint-disable max-lines */
import { afterEach, describe, expect, it, vi } from 'vitest'

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
  watchProjectConversationMuxStream,
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
  watchProjectConversationMuxStream: vi.fn(),
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
  watchProjectConversationMuxStream,
}))

import { createProjectConversationController } from './project-conversation-controller.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'
import { formatProjectConversationLabel } from './project-conversation-panel-labels'

function createWorkspaceDiff(conversationId: string, dirty = true) {
  return {
    workspaceDiff: {
      conversationId,
      workspacePath: `/tmp/${conversationId}`,
      dirty,
      reposChanged: dirty ? 1 : 0,
      filesChanged: dirty ? 1 : 0,
      added: dirty ? 3 : 0,
      removed: dirty ? 1 : 0,
      repos: dirty
        ? [
            {
              name: 'openase',
              path: 'openase',
              branch: 'agent/conv-1',
              dirty: true,
              filesChanged: 1,
              added: 3,
              removed: 1,
              files: [
                {
                  path: 'README.md',
                  status: 'modified',
                  added: 3,
                  removed: 1,
                },
              ],
            },
          ]
        : [],
    },
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

function mockLiveMuxStream(
  onFrame?: (handlers: {
    onFrame: (frame: {
      conversationId: string
      sentAt: string
      event: { kind: string; payload: Record<string, unknown> }
    }) => void
  }) => void,
) {
  watchProjectConversationMuxStream.mockImplementation(async (_projectId, handlers) => {
    handlers.onOpen?.()
    onFrame?.(handlers)
    await new Promise<void>((resolve) => {
      handlers.signal?.addEventListener('abort', () => resolve(), { once: true })
    })
  })
}

describe('createProjectConversationController restore flows', () => {
  afterEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('refreshes workspace diff on turn completion and preserves it after runtime reset', async () => {
    const muxHandlers: Array<{
      onFrame: (frame: {
        conversationId: string
        sentAt: string
        event: { kind: string; payload: Record<string, unknown> }
      }) => void
    }> = []

    createProjectConversation.mockResolvedValue({
      conversation: { id: 'conversation-1', title: '', providerId: 'provider-1' },
    })
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
    mockLiveMuxStream((handlers) => {
      muxHandlers.push(handlers)
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turnIndex: 1, status: 'started' },
      conversation: {
        id: 'conversation-1',
        projectId: 'project-1',
        userId: 'user-1',
        source: 'project_sidebar',
        providerId: 'provider-1',
        title: 'Race test',
        providerActiveFlags: [],
        status: 'active',
        rollingSummary: '',
        lastActivityAt: '2026-04-01T10:00:00Z',
        createdAt: '2026-04-01T10:00:00Z',
        updatedAt: '2026-04-01T10:00:00Z',
      },
    })
    closeProjectConversationRuntime.mockResolvedValue(undefined)

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Race test')
    muxHandlers[0]?.onFrame({
      conversationId: 'conversation-1',
      sentAt: '2026-04-01T10:00:01Z',
      event: {
        kind: 'turn_done',
        payload: {
          conversationId: 'conversation-1',
          turnId: 'turn-1',
        },
      },
    })
    await Promise.resolve()

    await controller.resetConversation()

    muxHandlers[0]?.onFrame({
      conversationId: 'conversation-1',
      sentAt: '2026-04-01T10:00:02Z',
      event: {
        kind: 'message',
        payload: {
          type: 'text',
          content: 'late assistant reply',
        },
      },
    })

    expect(closeProjectConversationRuntime).toHaveBeenCalledWith('conversation-1')
    expect(controller.phase).toBe('idle')
    expect(controller.conversationId).toBe('conversation-1')
    expect(
      controller.entries.some(
        (entry) => entry.kind === 'text' && entry.role === 'user' && entry.content === 'Race test',
      ),
    ).toBe(true)
    expect(
      controller.entries.some(
        (entry) => entry.kind === 'text' && entry.content === 'late assistant reply',
      ),
    ).toBe(false)
    expect(controller.workspaceDiff?.dirty).toBe(true)
    expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledTimes(2)

    controller.dispose()
  })

  it('switches conversations, reloads the matching transcript, and continues the selected session', async () => {
    seedProjectConversationTabsStorage(
      [{ conversationId: 'conversation-1', providerId: 'provider-1' }],
      0,
    )

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
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1', false))
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-2'))
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
    mockLiveMuxStream()
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-3', turnIndex: 2, status: 'started' },
      conversation: {
        id: 'conversation-2',
        projectId: 'project-1',
        userId: 'user-1',
        source: 'project_sidebar',
        providerId: 'provider-1',
        title: 'Older discussion',
        providerActiveFlags: [],
        status: 'active',
        rollingSummary: 'Older discussion',
        lastActivityAt: '2026-03-31T09:00:00Z',
        createdAt: '2026-03-31T09:00:00Z',
        updatedAt: '2026-03-31T09:00:00Z',
      },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
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
    expect(watchProjectConversationMuxStream).toHaveBeenCalledWith(
      'project-1',
      expect.objectContaining({
        signal: expect.any(AbortSignal),
        onFrame: expect.any(Function),
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
    expect(controller.workspaceDiff?.conversationId).toBe('conversation-2')
    expect(getProjectConversationWorkspaceDiff).toHaveBeenLastCalledWith('conversation-2')

    await controller.sendTurn('Follow up on the older plan')

    expect(createProjectConversation).not.toHaveBeenCalled()
    expect(startProjectConversationTurn).toHaveBeenLastCalledWith('conversation-2', {
      message: 'Follow up on the older plan',
      focus: undefined,
    })
    expect(controller.entries).toMatchObject([
      { kind: 'text', role: 'user', content: 'Continue the older plan' },
      { kind: 'text', role: 'user', content: 'Follow up on the older plan' },
    ])

    controller.dispose()
  })

  it('keeps the restored tab label anchored to the stable title when session summaries change', async () => {
    const muxHandlers: Array<{
      onFrame: (frame: {
        conversationId: string
        sentAt: string
        event: { kind: string; payload: Record<string, unknown> }
      }) => void
    }> = []

    seedProjectConversationTabsStorage(
      [{ conversationId: 'conversation-1', providerId: 'provider-1' }],
      0,
    )

    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          title: 'Keep the first title stable',
          rollingSummary: 'Initial recovery summary',
          providerId: 'provider-1',
          lastActivityAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(
      createWorkspaceDiff('conversation-1', false),
    )
    listProjectConversationEntries.mockResolvedValue({
      entries: [
        {
          id: 'entry-1',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'user_message',
          payload: { content: 'Keep the first title stable' },
          createdAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    watchProjectConversationMuxStream.mockImplementation(async (_projectId, handlers) => {
      handlers.onOpen?.()
      muxHandlers.push(handlers)
      await new Promise(() => {})
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.restore()

    expect(formatProjectConversationLabel(controller.tabs[0], controller.conversations)).toBe(
      'Keep the first title stable',
    )

    muxHandlers[0]?.onFrame({
      conversationId: 'conversation-1',
      sentAt: '2026-04-01T10:00:01Z',
      event: {
        kind: 'session',
        payload: {
          conversationId: 'conversation-1',
          runtimeState: 'executing',
          title: 'Keep the first title stable',
          rollingSummary: 'A newer summary should not replace the title',
          providerActiveFlags: [],
        },
      },
    })
    await Promise.resolve()

    expect(formatProjectConversationLabel(controller.tabs[0], controller.conversations)).toBe(
      'Keep the first title stable',
    )
  })

  it('does not stay in restoring when workspace diff loading hangs', async () => {
    seedProjectConversationTabsStorage(
      [{ conversationId: 'conversation-1', providerId: 'provider-1' }],
      0,
    )

    let resolveWorkspaceDiff: ((value: ReturnType<typeof createWorkspaceDiff>) => void) | null =
      null

    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Current conversation',
          providerId: 'provider-1',
          lastActivityAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    listProjectConversationEntries.mockResolvedValue({
      entries: [
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
    })
    getProjectConversationWorkspaceDiff.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveWorkspaceDiff = resolve
        }),
    )
    mockLiveMuxStream()

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.restore()

    expect(controller.phase).toBe('idle')
    expect(controller.conversationId).toBe('conversation-1')
    expect(controller.entries).toMatchObject([
      { kind: 'text', role: 'user', content: 'Current conversation' },
    ])
    expect(controller.workspaceDiffLoading).toBe(true)

    expect(resolveWorkspaceDiff).not.toBeNull()

    controller.dispose()
  })
})
