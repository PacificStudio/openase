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

import ProjectConversationPanel from './project-conversation-panel.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'
import { createWorkspaceDiff } from './project-conversation-panel.test-helpers'

function mockLiveMuxStream() {
  let handlers:
    | {
        signal?: AbortSignal
        onOpen?: () => void
        onFrame: (frame: {
          conversationId: string
          sentAt: string
          event: { kind: string; payload: Record<string, unknown> }
        }) => void
      }
    | undefined

  watchProjectConversationMuxStream.mockImplementation(async (_projectId, nextHandlers) => {
    handlers = nextHandlers
    nextHandlers.onOpen?.()
    await new Promise<void>((resolve) => {
      nextHandlers.signal?.addEventListener('abort', () => resolve(), { once: true })
    })
  })

  return {
    emit(conversationId: string, event: { kind: string; payload: Record<string, unknown> }) {
      handlers?.onFrame({
        conversationId,
        sentAt: '2026-04-01T10:00:00Z',
        event,
      })
    },
  }
}

async function waitForComposerReady(getByPlaceholderText: (text: string) => HTMLElement) {
  await waitFor(() => {
    expect(
      (getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement).disabled,
    ).toBe(false)
  })
}

describe('ProjectConversationPanel tab behavior', () => {
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

  it('keeps the composer enabled on an idle tab while another tab is waiting on input', async () => {
    const mux = mockLiveMuxStream()

    listProjectConversations.mockResolvedValue({ conversations: [] })
    createProjectConversation
      .mockResolvedValueOnce({
        conversation: {
          id: 'conversation-1',
          providerId: 'provider-1',
          lastActivityAt: '2026-04-01T10:00:00Z',
        },
      })
      .mockResolvedValueOnce({
        conversation: {
          id: 'conversation-2',
          providerId: 'provider-1',
          lastActivityAt: '2026-04-01T10:05:00Z',
        },
      })
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-2'))
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { getAllByRole, getByPlaceholderText, getByRole, findByText } = render(
      ProjectConversationPanel,
      {
        props: {
          context: { projectId: 'project-1' },
          providers: providerFixtures,
          defaultProviderId: 'provider-1',
          placeholder: 'Ask anything about this project…',
        },
      },
    )

    await waitForComposerReady(getByPlaceholderText)

    const prompt = getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement
    const sendButton = getByRole('button', { name: 'Send message' }) as HTMLButtonElement

    await fireEvent.input(prompt, { target: { value: 'Summarize the repo.' } })
    await fireEvent.click(sendButton)
    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
        message: 'Summarize the repo.',
        focus: undefined,
      })
    })

    mux.emit('conversation-1', {
      kind: 'interrupt_requested',
      payload: {
        interruptId: 'interrupt-1',
        provider: 'codex',
        kind: 'user_input',
        options: [],
        payload: {
          questions: [{ id: 'approval', question: 'Approve this change?' }],
        },
      },
    })

    await findByText('Input required')
    expect(prompt.disabled).toBe(true)

    await fireEvent.click(getByRole('button', { name: /New Tab/i }))

    const updatedPrompt = getByPlaceholderText(
      'Ask anything about this project…',
    ) as HTMLTextAreaElement
    const updatedSendButton = getByRole('button', { name: 'Send message' }) as HTMLButtonElement

    expect(updatedPrompt.disabled).toBe(false)

    await fireEvent.input(updatedPrompt, { target: { value: 'Parallel tab keeps working.' } })
    await fireEvent.click(updatedSendButton)

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenNthCalledWith(2, 'conversation-2', {
        message: 'Parallel tab keeps working.',
        focus: undefined,
      })
    })

    expect(getAllByRole('button', { name: /Close /i }).length).toBeGreaterThanOrEqual(1)
  })

  it('queues a follow-up message while waiting for the assistant reply and auto-sends it after turn completion', async () => {
    const mux = mockLiveMuxStream()

    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { findByText, getByPlaceholderText, getByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await waitForComposerReady(getByPlaceholderText)

    const prompt = getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement
    const sendButton = getByRole('button', { name: 'Send message' }) as HTMLButtonElement

    await fireEvent.input(prompt, { target: { value: 'What changed?' } })
    await fireEvent.click(sendButton)

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
        message: 'What changed?',
        focus: undefined,
      })
    })

    expect(prompt.disabled).toBe(false)

    await fireEvent.input(prompt, { target: { value: 'Draft the next request' } })
    expect(prompt.value).toBe('Draft the next request')
    expect(sendButton.disabled).toBe(false)

    await fireEvent.keyDown(prompt, { key: 'Enter' })
    expect(startProjectConversationTurn).toHaveBeenCalledTimes(1)
    await findByText('Queued')
    expect(prompt.value).toBe('')

    mux.emit('conversation-1', {
      kind: 'turn_done',
      payload: {},
    })

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenNthCalledWith(2, 'conversation-1', {
        message: 'Draft the next request',
        focus: undefined,
      })
    })
  })

  it('cancels a queued message before it is auto-sent', async () => {
    const mux = mockLiveMuxStream()

    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { findByText, getByPlaceholderText, getByRole, queryByText } = render(
      ProjectConversationPanel,
      {
        props: {
          context: { projectId: 'project-1' },
          providers: providerFixtures,
          defaultProviderId: 'provider-1',
          placeholder: 'Ask anything about this project…',
        },
      },
    )

    await waitForComposerReady(getByPlaceholderText)

    const prompt = getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement
    const sendButton = getByRole('button', { name: 'Send message' }) as HTMLButtonElement

    await fireEvent.input(prompt, { target: { value: 'First request' } })
    await fireEvent.click(sendButton)
    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
        message: 'First request',
        focus: undefined,
      })
    })

    await fireEvent.input(prompt, { target: { value: 'Queued request' } })
    await fireEvent.click(sendButton)
    await findByText('Queued')

    await fireEvent.click(getByRole('button', { name: 'Cancel queued message 1' }))
    expect(queryByText('Queued')).toBeNull()

    mux.emit('conversation-1', {
      kind: 'turn_done',
      payload: {},
    })

    await Promise.resolve()
    expect(startProjectConversationTurn).toHaveBeenCalledTimes(1)
  })
})
