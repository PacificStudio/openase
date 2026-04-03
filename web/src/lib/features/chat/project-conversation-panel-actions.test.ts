import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
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

describe('ProjectConversationPanel action proposals', () => {
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

  it('renders action proposals as interactive cards instead of raw JSON and confirms them', async () => {
    window.localStorage.setItem(
      'openase.project-conversation.project-1.provider-1',
      JSON.stringify({
        conversationIds: ['conversation-1'],
        activeConversationId: 'conversation-1',
      }),
    )

    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Action proposal thread',
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
            actions: [
              {
                method: 'POST',
                path: '/api/v1/projects/project-1/tickets',
                body: {
                  title: 'Implement child ticket',
                  priority: 'high',
                },
              },
            ],
          },
          createdAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockResolvedValue(undefined)
    executeProjectConversationActionProposal.mockResolvedValue({
      results: [
        {
          ok: true,
          summary: 'POST /api/v1/projects/project-1/tickets succeeded.',
        },
      ],
    })

    const { findByText, getByRole, getByText, queryByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await findByText('Create 1 child ticket')
    expect(queryByText(/action_proposal/i)).toBeNull()
    expect(getByRole('button', { name: 'Confirm' })).toBeTruthy()
    expect(getByRole('button', { name: 'Cancel' })).toBeTruthy()

    await fireEvent.click(getByRole('button', { name: 'Confirm' }))

    await waitFor(() => {
      expect(executeProjectConversationActionProposal).toHaveBeenCalledWith(
        'conversation-1',
        'entry-proposal',
      )
    })
    expect(await findByText('Executed')).toBeTruthy()
    expect(await findByText('POST /api/v1/projects/project-1/tickets succeeded.')).toBeTruthy()
  })

  it('lets users cancel a pending action proposal without executing it', async () => {
    window.localStorage.setItem(
      'openase.project-conversation.project-1.provider-1',
      JSON.stringify({
        conversationIds: ['conversation-1'],
        activeConversationId: 'conversation-1',
      }),
    )

    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Action proposal thread',
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
            actions: [
              {
                method: 'POST',
                path: '/api/v1/projects/project-1/tickets',
                body: {
                  title: 'Implement child ticket',
                },
              },
            ],
          },
          createdAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockResolvedValue(undefined)

    const { findByText, getByRole, queryByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await findByText('Create 1 child ticket')
    await fireEvent.click(getByRole('button', { name: 'Cancel' }))

    expect(executeProjectConversationActionProposal).not.toHaveBeenCalled()
    expect(await findByText('Cancelled')).toBeTruthy()
    expect(await findByText('No API calls were executed.')).toBeTruthy()
    expect(queryByRole('button', { name: 'Confirm' })).toBeNull()
    expect(queryByRole('button', { name: 'Cancel' })).toBeNull()
  })

  it('renders platform command proposals as readable command cards and confirms them', async () => {
    window.localStorage.setItem(
      'openase.project-conversation.project-1.provider-1',
      JSON.stringify({
        conversationIds: ['conversation-1'],
        activeConversationId: 'conversation-1',
      }),
    )

    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Command proposal thread',
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
            type: 'platform_command_proposal',
            summary: 'Update project and ticket',
            commands: [
              {
                command: 'project_update.create',
                args: {
                  project: 'CDN',
                  content: 'Shift the project to a backend-only control plane.',
                },
              },
              {
                command: 'ticket.update',
                args: {
                  ticket: 'ASE-1',
                  status: 'Todo',
                },
              },
            ],
          },
          createdAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockResolvedValue(undefined)
    executeProjectConversationActionProposal.mockResolvedValue({
      results: [
        {
          ok: true,
          summary: 'Created project update in CDN.',
        },
        {
          ok: true,
          summary: 'Updated ticket ASE-1.',
        },
      ],
    })

    const { findByText, getByRole, getByText, queryByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await findByText('Update project and ticket')
    expect(queryByText('/api/v1/projects/project-1/tickets')).toBeNull()

    await fireEvent.click(getByText('project_update.create'))
    expect(await findByText(/"project": "CDN"/)).toBeTruthy()

    await fireEvent.click(getByRole('button', { name: 'Confirm' }))

    await waitFor(() => {
      expect(executeProjectConversationActionProposal).toHaveBeenCalledWith(
        'conversation-1',
        'entry-proposal',
      )
    })
    expect(await findByText('Created project update in CDN.')).toBeTruthy()
    expect(await findByText('Updated ticket ASE-1.')).toBeTruthy()
  })
})
