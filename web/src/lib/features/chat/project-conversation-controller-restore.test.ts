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

describe('createProjectConversationController restore flows', () => {
  afterEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('refreshes workspace diff on turn completion and preserves it after runtime reset', async () => {
    const streamHandlers: Array<{
      onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void
    }> = []

    createProjectConversation.mockResolvedValue({
      conversation: { id: 'conversation-1' },
    })
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockImplementation(async (_conversationId, handlers) => {
      streamHandlers.push(handlers)
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })
    closeProjectConversationRuntime.mockResolvedValue(undefined)

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Race test')
    streamHandlers[0]?.onEvent({
      kind: 'turn_done',
      payload: {
        conversationId: 'conversation-1',
        turnId: 'turn-1',
      },
    })
    await Promise.resolve()

    await controller.resetConversation()

    streamHandlers[0]?.onEvent({
      kind: 'message',
      payload: {
        type: 'text',
        content: 'late assistant reply',
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
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-3', turn_index: 2, status: 'started' },
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
    watchProjectConversation.mockResolvedValue(undefined)

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
  })
})
