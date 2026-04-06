import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import { ProjectConversationPanel } from '$lib/features/chat'
import { appStore } from '$lib/stores/app.svelte'

const {
  closeChatSession,
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  streamChatTurn,
  watchProjectConversation,
  watchProjectConversationMuxStream,
} = vi.hoisted(() => ({
  closeChatSession: vi.fn(),
  closeProjectConversationRuntime: vi.fn(),
  createProjectConversation: vi.fn(),
  executeProjectConversationActionProposal: vi.fn(),
  getProjectConversation: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  listProjectConversationEntries: vi.fn(),
  listProjectConversations: vi.fn(),
  respondProjectConversationInterrupt: vi.fn(),
  startProjectConversationTurn: vi.fn(),
  streamChatTurn: vi.fn(),
  watchProjectConversation: vi.fn(),
  watchProjectConversationMuxStream: vi.fn(),
}))

const { listProviders } = vi.hoisted(() => ({
  listProviders: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeChatSession,
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  streamChatTurn,
  watchProjectConversation,
  watchProjectConversationMuxStream,
}))

vi.mock('$lib/api/openase', async () => {
  const actual = await vi.importActual<typeof import('$lib/api/openase')>('$lib/api/openase')
  return {
    ...actual,
    listProviders,
  }
})

import {
  createWorkspaceDiff,
  providerFixtures,
  renderTicketDrawerContent,
  resetTicketDrawerTestAppStore,
} from './ticket-drawer-content.test-helpers'

describe('TicketDrawerContent project AI integration', () => {
  function mockLiveMuxStream() {
    watchProjectConversationMuxStream.mockImplementation(async (_projectId, handlers) => {
      handlers.onOpen?.()
      await new Promise<void>((resolve) => {
        handlers.signal?.addEventListener('abort', () => resolve(), { once: true })
      })
    })
  }

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
    resetTicketDrawerTestAppStore()
  })

  function renderTicketFocusedProjectAI() {
    renderTicketDrawerContent()
    expect(appStore.projectAssistantFocus).toMatchObject({
      kind: 'ticket',
      projectId: 'project-1',
      ticketId: 'ticket-1',
      selectedArea: 'comments',
    })

    return render(ProjectConversationPanel, {
      props: {
        organizationId: 'org-1',
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        focus: appStore.projectAssistantFocus,
        placeholder: 'Ask about this ticket without restating the basics…',
      },
    })
  }

  it('routes the full ticket focus into ticket-focused Project AI conversations', async () => {
    mockLiveMuxStream()
    listProviders.mockResolvedValue({ providers: providerFixtures })
    listProjectConversations.mockResolvedValue({ conversations: [] })
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-02T09:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { getByPlaceholderText, getByRole } = renderTicketFocusedProjectAI()

    await waitFor(() => {
      expect(listProjectConversations).toHaveBeenCalled()
    })

    const prompt = getByPlaceholderText(
      'Ask about this ticket without restating the basics…',
    ) as HTMLTextAreaElement
    await fireEvent.input(prompt, { target: { value: 'Why is this ticket not running?' } })
    const sendButton = getByRole('button', { name: 'Send message' })
    await waitFor(() => {
      expect((sendButton as HTMLButtonElement).disabled).toBe(false)
    })
    await fireEvent.click(sendButton)

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith(
        'conversation-1',
        expect.objectContaining({
          message: 'Why is this ticket not running?',
          focus: expect.objectContaining({
            kind: 'ticket',
            projectId: 'project-1',
            ticketId: 'ticket-1',
            ticketIdentifier: 'ASE-470',
            ticketDescription:
              'Route ticket drawer AI through the durable project conversation runtime.',
            ticketPriority: 'high',
            ticketAttemptCount: 3,
            ticketRetryPaused: true,
            ticketPauseReason: 'Repeated hook failures',
            selectedArea: 'comments',
            ticketDependencies: [
              expect.objectContaining({
                identifier: 'ASE-100',
                relation: 'blocked_by',
              }),
            ],
            ticketRepoScopes: [
              expect.objectContaining({
                repoId: 'repo-1',
                pullRequestUrl: 'https://github.com/PacificStudio/openase/pull/999',
              }),
            ],
            ticketHookHistory: [
              expect.objectContaining({
                hookName: 'ticket.on_complete',
                status: 'fail',
              }),
            ],
            ticketCurrentRun: expect.objectContaining({
              id: 'run-1',
              status: 'failed',
              currentStepSummary: 'openase test ./internal/chat',
            }),
          }),
        }),
      )
    })

    expect(streamChatTurn).not.toHaveBeenCalled()
    expect(closeChatSession).not.toHaveBeenCalled()
  })

  it('keeps legacy action proposal summaries visible from the ticket-focused Project AI entry point', async () => {
    mockLiveMuxStream()
    window.localStorage.setItem(
      'openase.project-conversation.project-1.provider-1',
      JSON.stringify({
        conversationIds: ['conversation-1'],
        activeConversationId: 'conversation-1',
      }),
    )

    listProviders.mockResolvedValue({ providers: providerFixtures })
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'ticket-focused project ai',
          lastActivityAt: '2026-04-02T09:00:00Z',
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
            summary: 'Create a retry investigation child ticket',
            actions: [
              {
                method: 'POST',
                path: '/api/v1/projects/project-1/tickets',
                body: { title: 'Investigate repeated retry pause' },
              },
            ],
          },
          createdAt: '2026-04-02T09:00:00Z',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockResolvedValue(undefined)

    const { findByText, queryByRole } = renderTicketFocusedProjectAI()

    await findByText('Create a retry investigation child ticket')
    expect(queryByRole('button', { name: 'Confirm' })).toBeNull()
    expect(queryByRole('button', { name: 'Cancel' })).toBeNull()
    expect(executeProjectConversationActionProposal).not.toHaveBeenCalled()
  })
})
