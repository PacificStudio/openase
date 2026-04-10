import { afterEach, describe, expect, it, vi } from 'vitest'

const {
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries,
} = vi.hoisted(() => ({
  getProjectConversation: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  listProjectConversationEntries: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries,
}))

import {
  handleTabStreamEvent,
  projectConversationTabPhaseFromRuntimeState,
  reconcileTabAfterReconnect,
  refreshTabWorkspaceDiff,
} from './project-conversation-controller-runtime-effects'
import { createProjectConversationTabState } from './project-conversation-controller-state'

function createWorkspaceDiff(conversationId: string, added: number) {
  return {
    workspaceDiff: {
      conversationId,
      workspacePath: `/tmp/${conversationId}`,
      dirty: added > 0,
      reposChanged: added > 0 ? 1 : 0,
      filesChanged: added > 0 ? 1 : 0,
      added,
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

describe('project conversation runtime effects', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('maps runtime and provider states to the expected tab phases', () => {
    expect(projectConversationTabPhaseFromRuntimeState('interrupted')).toBe('awaiting_interrupt')
    expect(projectConversationTabPhaseFromRuntimeState('executing')).toBe('awaiting_reply')
    expect(projectConversationTabPhaseFromRuntimeState('ready', 'waiting_on_approval')).toBe(
      'awaiting_interrupt',
    )
    expect(
      projectConversationTabPhaseFromRuntimeState('ready', undefined, ['waiting-on-user-input']),
    ).toBe('awaiting_interrupt')
    expect(projectConversationTabPhaseFromRuntimeState('ready', 'active')).toBe('awaiting_reply')
    expect(projectConversationTabPhaseFromRuntimeState('ready', undefined, ['running'])).toBe(
      'awaiting_reply',
    )
    expect(projectConversationTabPhaseFromRuntimeState('ready')).toBe('idle')
  })

  it('keeps the newest workspace diff when older refresh requests finish later', async () => {
    const tab = createProjectConversationTabState(1, 'provider-1')
    tab.conversationId = 'conversation-1'

    const first = deferredPromise<ReturnType<typeof createWorkspaceDiff>>()
    const second = deferredPromise<ReturnType<typeof createWorkspaceDiff>>()

    getProjectConversationWorkspaceDiff
      .mockImplementationOnce(() => first.promise)
      .mockImplementationOnce(() => second.promise)

    const firstRefresh = refreshTabWorkspaceDiff(tab, 'conversation-1')
    const secondRefresh = refreshTabWorkspaceDiff(tab, 'conversation-1')

    second.resolve(createWorkspaceDiff('conversation-1', 7))
    await secondRefresh

    expect(tab.workspaceDiff?.added).toBe(7)
    expect(tab.workspaceDiffLoading).toBe(false)

    first.resolve(createWorkspaceDiff('conversation-1', 1))
    await firstRefresh

    expect(tab.workspaceDiff?.added).toBe(7)
    expect(tab.workspaceDiffLoading).toBe(false)
  })

  it('marks inactive tabs unread and pending when live text arrives', () => {
    const tab = createProjectConversationTabState(1, 'provider-1')
    tab.conversationId = 'conversation-1'

    const conversations = {
      touchConversation: vi.fn(),
      applySessionPayload: vi.fn(),
      upsertConversation: vi.fn(),
    }

    handleTabStreamEvent(
      {
        conversations,
        isActiveTab: () => false,
        persistTabs: vi.fn(),
        touchTabs: vi.fn(),
        connectTabStream: vi.fn(),
      },
      tab,
      {
        kind: 'message',
        payload: {
          type: 'text',
          content: 'background chunk',
        },
      },
    )

    expect(tab.phase).toBe('awaiting_reply')
    expect(tab.needsHydration).toBe(true)
    expect(tab.unread).toBe(true)
    expect(conversations.touchConversation).toHaveBeenCalledWith('conversation-1')
  })

  it('keeps inactive tabs in hydration mode after reconnect when the runtime is still executing', async () => {
    const tab = createProjectConversationTabState(1, 'provider-1')
    tab.conversationId = 'conversation-1'
    tab.streamId = 2
    tab.phase = 'connecting_stream'

    const conversations = {
      touchConversation: vi.fn(),
      applySessionPayload: vi.fn(),
      upsertConversation: vi.fn(),
    }

    getProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:01:00Z',
        runtimePrincipal: {
          runtimeState: 'executing',
        },
      },
    })

    await reconcileTabAfterReconnect(
      {
        conversations,
        isActiveTab: () => false,
        persistTabs: vi.fn(),
        touchTabs: vi.fn(),
        connectTabStream: vi.fn(),
      },
      tab,
      'conversation-1',
      2,
    )

    expect(tab.phase).toBe('awaiting_reply')
    expect(tab.needsHydration).toBe(true)
    expect(tab.unread).toBe(true)
    expect(listProjectConversationEntries).not.toHaveBeenCalled()
    expect(conversations.upsertConversation).toHaveBeenCalled()
  })

  it('keeps active tabs in awaiting_interrupt when reconnect hydration still contains a pending interrupt', async () => {
    const tab = createProjectConversationTabState(1, 'provider-1')
    tab.conversationId = 'conversation-1'
    tab.streamId = 5
    tab.phase = 'connecting_stream'

    const conversations = {
      touchConversation: vi.fn(),
      applySessionPayload: vi.fn(),
      upsertConversation: vi.fn(),
    }

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
          id: 'entry-user-1',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'user_message',
          payload: { content: 'Need approval' },
          createdAt: '2026-04-01T10:00:00Z',
        },
        {
          id: 'entry-interrupt-1',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 2,
          kind: 'interrupt',
          payload: {
            interrupt_id: 'interrupt-1',
            provider: 'codex',
            kind: 'user_input',
            payload: {},
            options: [],
          },
          createdAt: '2026-04-01T10:00:01Z',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1', 4))

    await reconcileTabAfterReconnect(
      {
        conversations,
        isActiveTab: () => true,
        persistTabs: vi.fn(),
        touchTabs: vi.fn(),
        connectTabStream: vi.fn(),
      },
      tab,
      'conversation-1',
      5,
    )

    expect(tab.phase).toBe('awaiting_interrupt')
    expect(tab.needsHydration).toBe(false)
    expect(tab.unread).toBe(false)
    expect(listProjectConversationEntries).toHaveBeenCalledWith('conversation-1')
    expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledWith('conversation-1')
  })
})
