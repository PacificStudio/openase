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

describe('createProjectConversationController restore live flows', () => {
  afterEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('restores the latest project conversation when no tabs were persisted', async () => {
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
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    mockLiveMuxStream()

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.restore()

    expect(controller.conversationId).toBe('conversation-1')
    expect(controller.entries).toMatchObject([
      { kind: 'text', role: 'user', content: 'Current conversation' },
    ])
    expect(controller.tabs).toHaveLength(1)
    expect(controller.tabs[0]?.restored).toBe(true)
    expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledWith('conversation-1')

    controller.dispose()
  })

  it('appends live assistant text to a restored conversation controller state', async () => {
    seedProjectConversationTabsStorage(
      [{ conversationId: 'conversation-1', providerId: 'provider-1' }],
      0,
    )

    let streamHandlers:
      | {
          onFrame: (frame: {
            conversationId: string
            sentAt: string
            event: { kind: string; payload: Record<string, unknown> }
          }) => void
        }
      | undefined

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
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
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
    mockLiveMuxStream((handlers) => {
      streamHandlers = handlers
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.restore()

    streamHandlers?.onFrame({
      conversationId: 'conversation-1',
      sentAt: '2026-04-01T10:05:00Z',
      event: {
        kind: 'message',
        payload: {
          type: 'text',
          content: 'First streamed reply chunk.',
        },
      },
    })

    expect(
      controller.entries.some(
        (entry) =>
          entry.kind === 'text' &&
          entry.role === 'assistant' &&
          entry.content === 'First streamed reply chunk.',
      ),
    ).toBe(true)

    controller.dispose()
  })
})
