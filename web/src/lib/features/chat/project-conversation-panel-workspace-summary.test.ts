import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

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

import ProjectConversationPanel from './project-conversation-panel.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'
import { createWorkspaceDiff } from './project-conversation-panel.test-helpers'

describe('ProjectConversationPanel workspace summary', () => {
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

  it('renders compact workspace bar and expands to show repo details', async () => {
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Current conversation',
          lastActivityAt: '2026-04-01T10:00:00Z',
          providerId: 'provider-1',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(
      createWorkspaceDiff('conversation-1', true),
    )
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
    watchProjectConversation.mockResolvedValue(undefined)

    const { findByText, queryByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    // Compact bar shows summary
    await findByText('Workspace changes')
    await findByText('1 repo changed · +4 -1')

    // Details are hidden by default
    expect(queryByText('web/src/app.ts')).toBeNull()

    // Click to expand
    const bar = await findByText('Workspace changes')
    await fireEvent.click(bar.closest('button')!)

    // Now file list is visible (single repo omits repo header)
    await findByText('web/src/app.ts')
    await findByText('M')
  })
})
