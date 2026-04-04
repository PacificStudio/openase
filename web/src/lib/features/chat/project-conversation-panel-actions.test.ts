import { cleanup, render } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const {
  closeProjectConversationRuntime,
  createProjectConversation,
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
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
}))

import ProjectConversationPanel from './project-conversation-panel.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'
import { createWorkspaceDiff } from './project-conversation-panel.test-helpers'

const seedConversationStorage = () =>
  window.localStorage.setItem(
    'openase.project-conversation.project-1',
    JSON.stringify({
      tabs: [{ conversationId: 'conversation-1', providerId: 'provider-1', draft: '' }],
      activeTabIndex: 0,
    }),
  )

describe('ProjectConversationPanel legacy proposal rendering', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('renders legacy action proposal entries as assistant text without confirm controls', async () => {
    seedConversationStorage()
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Legacy proposal thread',
          lastActivityAt: '2026-04-01T10:00:00Z',
          providerId: 'provider-1',
        },
      ],
    })
    listProjectConversationEntries.mockResolvedValue({
      entries: [
        {
          id: 'entry-proposal',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'action_proposal',
          payload: {
            summary: 'Create 1 child ticket',
            actions: [{ method: 'POST', path: '/api/v1/projects/project-1/tickets' }],
          },
          createdAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockResolvedValue(undefined)

    const { findByText, queryByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    expect(await findByText('Create 1 child ticket')).toBeTruthy()
    expect(queryByRole('button', { name: 'Confirm' })).toBeNull()
    expect(queryByRole('button', { name: 'Cancel' })).toBeNull()
  })
})
