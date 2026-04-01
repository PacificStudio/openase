import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const {
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
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
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
}))

import ProjectConversationPanel from './project-conversation-panel.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'

describe('ProjectConversationPanel', () => {
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

  it('disables input and send controls while an interrupt response is pending', async () => {
    let streamHandlers:
      | {
          onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void
        }
      | undefined

    listProjectConversations.mockResolvedValue({ conversations: [] })
    createProjectConversation.mockResolvedValue({
      conversation: { id: 'conversation-1' },
    })
    watchProjectConversation.mockImplementation(async (_conversationId, handlers) => {
      streamHandlers = handlers
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { getByPlaceholderText, getByRole, findByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    const prompt = getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement
    const sendButton = getByRole('button', { name: 'Send message' }) as HTMLButtonElement

    await fireEvent.input(prompt, { target: { value: 'Summarize the repo.' } })
    await fireEvent.keyDown(prompt, { key: 'Enter' })
    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith(
        'conversation-1',
        'Summarize the repo.',
      )
      expect(streamHandlers).toBeDefined()
    })

    streamHandlers?.onEvent({
      kind: 'interrupt_requested',
      payload: {
        interruptId: 'interrupt-1',
        provider: 'codex',
        kind: 'user_input',
        options: [],
        payload: {
          questions: [
            {
              id: 'approval',
              question: 'Approve this change?',
            },
          ],
        },
      },
    })

    await waitFor(() => {
      expect(prompt.disabled).toBe(true)
      expect(sendButton.disabled).toBe(true)
    })
    expect(await findByText('User input required')).toBeTruthy()
  })

  it('renders a conversation selector and loads the chosen history', async () => {
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Most recent summary',
          lastActivityAt: '2026-04-01T10:00:00Z',
          providerId: 'provider-1',
        },
        {
          id: 'conversation-2',
          rollingSummary: 'Older discussion',
          lastActivityAt: '2026-03-31T09:00:00Z',
          providerId: 'provider-1',
        },
      ],
    })
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

    const { findByLabelText, findByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
      },
    })

    const selector = (await findByLabelText('Conversation')) as HTMLSelectElement
    await findByText('Current conversation')
    expect(selector.value).toBe('conversation-1')

    await fireEvent.change(selector, { target: { value: 'conversation-2' } })

    await findByText('Continue the older plan')
    expect(listProjectConversationEntries).toHaveBeenLastCalledWith('conversation-2')
    expect(watchProjectConversation).toHaveBeenLastCalledWith(
      'conversation-2',
      expect.objectContaining({
        signal: expect.any(AbortSignal),
        onEvent: expect.any(Function),
      }),
    )
  })
})
