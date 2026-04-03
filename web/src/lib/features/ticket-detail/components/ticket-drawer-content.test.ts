import { cleanup, fireEvent, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

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

  it('opens Project AI from the ticket drawer and sends the full ticket focus through project conversations', async () => {
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

    const { getByLabelText, getByPlaceholderText, getByRole } = renderTicketDrawerContent()

    await fireEvent.click(getByLabelText('AI 分析'))
    await waitFor(() => {
      expect(listProjectConversations).toHaveBeenCalled()
    })

    const prompt = getByPlaceholderText(
      'Ask about this ticket without restating the basics…',
    ) as HTMLTextAreaElement
    await fireEvent.input(prompt, { target: { value: 'Why is this ticket not running?' } })
    await fireEvent.click(getByRole('button', { name: 'Send message' }))

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

  it('keeps action proposal cards working from the ticket drawer entry point', async () => {
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
    executeProjectConversationActionProposal.mockResolvedValue({
      results: [{ ok: true, summary: 'POST /api/v1/projects/project-1/tickets succeeded.' }],
    })

    const { findByText, getByLabelText, getByRole } = renderTicketDrawerContent()

    await fireEvent.click(getByLabelText('AI 分析'))
    await findByText('Create a retry investigation child ticket')
    await fireEvent.click(getByRole('button', { name: 'Confirm' }))

    await waitFor(() => {
      expect(executeProjectConversationActionProposal).toHaveBeenCalledWith(
        'conversation-1',
        'entry-proposal',
      )
    })
    expect(await findByText('Executed')).toBeTruthy()
  })
})
