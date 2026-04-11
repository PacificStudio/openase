import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { ApiError } from '$lib/api/client'

const {
  closeProjectConversationRuntime,
  createProjectConversation,
  deleteProjectConversation,
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
  deleteProjectConversation: vi.fn(),
  getProjectConversation: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  listProjectConversationEntries: vi.fn(),
  listProjectConversations: vi.fn(),
  respondProjectConversationInterrupt: vi.fn(),
  startProjectConversationTurn: vi.fn(),
  watchProjectConversation: vi.fn(),
}))

const { interruptAgent } = vi.hoisted(() => ({
  interruptAgent: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

vi.mock('$lib/api/chat', () => ({
  closeProjectConversationRuntime,
  createProjectConversation,
  deleteProjectConversation,
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
  watchProjectConversationMuxStream: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  interruptAgent,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
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

function primeRestoredConversation() {
  seedConversationStorage()
  listProjectConversations.mockResolvedValue({
    conversations: [
      {
        id: 'conversation-1',
        projectId: 'project-1',
        userId: 'user-1',
        source: 'project_sidebar',
        providerId: 'provider-1',
        title: 'Current conversation',
        rollingSummary: 'Filtered transcript thread',
        status: 'idle',
        lastActivityAt: '2026-04-01T10:00:00Z',
        createdAt: '2026-04-01T09:55:00Z',
        updatedAt: '2026-04-01T10:00:00Z',
      },
    ],
  })
  listProjectConversationEntries.mockResolvedValue({
    entries: [],
  })
  getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
  watchProjectConversation.mockResolvedValue(undefined)
}

describe('ProjectConversationPanel actions', () => {
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

  it('does not render unsupported persisted transcript entries', async () => {
    seedConversationStorage()
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Filtered transcript thread',
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
          kind: 'unsupported_structured_entry',
          payload: {
            summary: 'Create 1 child ticket',
            items: [{ name: 'child-ticket' }],
          },
          createdAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockResolvedValue(undefined)

    const { queryByText, queryByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await waitFor(() => {
      expect(queryByText('Create 1 child ticket')).toBeNull()
    })
    expect(queryByRole('button', { name: 'Confirm' })).toBeNull()
    expect(queryByRole('button', { name: 'Cancel' })).toBeNull()
  })

  it('deletes a conversation from history and leaves a usable blank tab', async () => {
    primeRestoredConversation()
    deleteProjectConversation.mockResolvedValue(undefined)
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    const { getByRole, queryByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await waitFor(() => {
      expect(getByRole('tab', { name: /^Current conversation Restored$/ })).toBeTruthy()
    })

    await fireEvent.click(getByRole('button', { name: 'Conversation history' }))
    await fireEvent.click(getByRole('button', { name: /Delete Current conversation/i }))

    await waitFor(() =>
      expect(deleteProjectConversation).toHaveBeenCalledWith('conversation-1', { force: false }),
    )
    await waitFor(() => {
      expect(getByRole('tab', { name: /^New tab$/ })).toBeTruthy()
    })
    await waitFor(() => {
      expect(getByRole('button', { name: 'Conversation history' }).hasAttribute('disabled')).toBe(
        true,
      )
    })
    expect(queryByRole('tab', { name: /^Current conversation Restored$/ })).toBeNull()
    expect(toastStore.success).toHaveBeenCalledWith('Project AI conversation deleted.')
    expect(
      JSON.parse(window.localStorage.getItem('openase.project-conversation.global') ?? '{}'),
    ).toMatchObject({
      activeTabIndex: 0,
      tabs: [
        {
          projectId: 'project-1',
          conversationId: '',
          providerId: 'provider-1',
          draft: '',
        },
      ],
    })
  })

  it('asks for a second confirmation before force deleting a dirty workspace conversation', async () => {
    primeRestoredConversation()
    deleteProjectConversation
      .mockRejectedValueOnce(new ApiError(409, 'Workspace has uncommitted changes.'))
      .mockResolvedValueOnce(undefined)
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    const { getByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await waitFor(() => {
      expect(getByRole('tab', { name: /^Current conversation Restored$/ })).toBeTruthy()
    })

    await fireEvent.click(getByRole('button', { name: 'Conversation history' }))
    await fireEvent.click(getByRole('button', { name: /Delete Current conversation/i }))

    await waitFor(() => {
      expect(deleteProjectConversation).toHaveBeenNthCalledWith(1, 'conversation-1', {
        force: false,
      })
    })
    await waitFor(() => {
      expect(deleteProjectConversation).toHaveBeenNthCalledWith(2, 'conversation-1', {
        force: true,
      })
    })
    expect(confirmSpy).toHaveBeenCalledTimes(2)
    expect(confirmSpy.mock.calls[1]?.[0]).toContain('Workspace has uncommitted changes.')
    expect(toastStore.success).toHaveBeenCalledWith('Project AI conversation deleted.')
  })
})
