import { afterEach, describe, expect, it, vi } from 'vitest'

const { listProjectConversations } = vi.hoisted(() => ({
  listProjectConversations: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  listProjectConversations,
}))

import {
  fetchCrossProjectConversations,
  mapPersistedToRestoredTabs,
} from './project-conversation-controller-restore-helpers'
import type { ProjectConversationTabState } from './project-conversation-controller-state'
import type { PersistedProjectConversationTab } from './project-conversation-storage'

function createTabState(
  providerId = '',
  projectId = '',
  projectName = '',
): ProjectConversationTabState {
  return {
    id: `tab-${providerId || 'blank'}-${projectId || 'global'}`,
    providerId,
    projectId,
    projectName,
    conversationId: '',
    entries: [],
    queuedTurns: [],
    draft: '',
    phase: 'idle',
    restored: false,
    needsHydration: false,
  } as unknown as ProjectConversationTabState
}

describe('project conversation restore helpers', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('tracks failed project fetches without dropping the successful conversations', async () => {
    listProjectConversations.mockImplementation(async ({ projectId }: { projectId: string }) => {
      if (projectId === 'project-2') {
        throw new Error('temporary failure')
      }
      return {
        conversations: [
          {
            id: 'conversation-1',
            projectId,
            providerId: 'provider-1',
            lastActivityAt: '2026-04-01T10:00:00Z',
          },
        ],
      }
    })

    const persistedTabs: PersistedProjectConversationTab[] = [
      {
        projectId: 'project-2',
        projectName: 'Project 2',
        conversationId: 'conversation-2',
        providerId: 'provider-2',
        draft: 'keep me',
      },
    ]

    const result = await fetchCrossProjectConversations('project-1', persistedTabs)

    expect(result.allConversations).toHaveLength(1)
    expect(result.allConversations[0]?.id).toBe('conversation-1')
    expect(result.currentProjectConversationIds.has('conversation-1')).toBe(true)
    expect(result.failedProjectIds.has('project-2')).toBe(true)
  })

  it('keeps persisted tabs hydration-ready when their project fetch fails', () => {
    const persistedTabs: PersistedProjectConversationTab[] = [
      {
        projectId: 'project-2',
        projectName: 'Project 2',
        conversationId: 'conversation-2',
        providerId: 'provider-2',
        draft: 'recover me',
      },
    ]

    const restoredTabs = mapPersistedToRestoredTabs(
      persistedTabs,
      new Map(),
      new Set(['project-2']),
      (providerId, _restored, projectId, projectName) =>
        createTabState(providerId, projectId, projectName),
      'provider-fallback',
    )

    expect(restoredTabs).toHaveLength(1)
    expect(restoredTabs[0]?.conversationId).toBe('conversation-2')
    expect(restoredTabs[0]?.restored).toBe(true)
    expect(restoredTabs[0]?.tab.conversationId).toBe('conversation-2')
    expect(restoredTabs[0]?.tab.providerId).toBe('provider-2')
    expect(restoredTabs[0]?.tab.needsHydration).toBe(true)
    expect(restoredTabs[0]?.tab.draft).toBe('recover me')
  })
})
