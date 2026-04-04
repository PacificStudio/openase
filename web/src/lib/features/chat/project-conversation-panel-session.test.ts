import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const {
  createProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversations,
  startProjectConversationTurn,
  watchProjectConversation,
} = vi.hoisted(() => ({
  createProjectConversation: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  listProjectConversations: vi.fn(),
  startProjectConversationTurn: vi.fn(),
  watchProjectConversation: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeProjectConversationRuntime: vi.fn(),
  createProjectConversation,
  executeProjectConversationActionProposal: vi.fn(),
  getProjectConversation: vi.fn(),
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries: vi.fn(),
  listProjectConversations,
  respondProjectConversationInterrupt: vi.fn(),
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
      tabs: [{ conversationId: 'conversation-live-1', providerId: 'provider-1', draft: '' }],
      activeTabIndex: 0,
    }),
  )

describe('ProjectConversationPanel session status', () => {
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

  it('renders Claude session status from live session events', async () => {
    const streamHandlers = new Map<
      string,
      { onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void }
    >()

    listProjectConversations.mockResolvedValue({ conversations: [] })
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-claude-1',
        providerId: 'provider-2',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(
      createWorkspaceDiff('conversation-claude-1'),
    )
    watchProjectConversation.mockImplementation(async (conversationId, handlers) => {
      streamHandlers.set(conversationId, handlers)
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { getByPlaceholderText, getByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-2',
        placeholder: 'Ask anything about this project…',
      },
    })

    await fireEvent.input(getByPlaceholderText('Ask anything about this project…'), {
      target: { value: 'Inspect this Claude conversation' },
    })
    await fireEvent.click(getByRole('button', { name: 'Send message' }))

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-claude-1', {
        message: 'Inspect this Claude conversation',
        focus: undefined,
      })
    })

    streamHandlers.get('conversation-claude-1')?.onEvent({
      kind: 'session',
      payload: {
        conversationId: 'conversation-claude-1',
        runtimeState: 'ready',
        providerAnchorKind: 'session',
        providerAnchorId: 'claude-session-42',
        providerTurnSupported: false,
        providerStatus: 'requires_action',
        providerActiveFlags: ['requires_action'],
      },
    })

    expect(watchProjectConversation).toHaveBeenCalledWith(
      'conversation-claude-1',
      expect.any(Object),
    )
  })

  it('renders Codex thread status from live session events', async () => {
    const streamHandlers = new Map<
      string,
      { onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void }
    >()

    listProjectConversations.mockResolvedValue({ conversations: [] })
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-codex-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(
      createWorkspaceDiff('conversation-codex-1'),
    )
    watchProjectConversation.mockImplementation(async (conversationId, handlers) => {
      streamHandlers.set(conversationId, handlers)
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { getByPlaceholderText, getByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await fireEvent.input(getByPlaceholderText('Ask anything about this project…'), {
      target: { value: 'Inspect this Codex conversation' },
    })
    await fireEvent.click(getByRole('button', { name: 'Send message' }))

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-codex-1', {
        message: 'Inspect this Codex conversation',
        focus: undefined,
      })
    })

    streamHandlers.get('conversation-codex-1')?.onEvent({
      kind: 'session',
      payload: {
        conversationId: 'conversation-codex-1',
        runtimeState: 'ready',
        providerAnchorKind: 'thread',
        providerAnchorId: 'thread-99',
        providerTurnId: 'turn-99',
        providerTurnSupported: true,
        providerStatus: 'waitingOnApproval',
        providerActiveFlags: ['waitingOnApproval'],
      },
    })

    expect(watchProjectConversation).toHaveBeenCalledWith(
      'conversation-codex-1',
      expect.any(Object),
    )
  })

  it('renders live assistant text chunks from the project conversation stream', async () => {
    const streamHandlers = new Map<
      string,
      { onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void }
    >()

    seedConversationStorage()
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-live-1',
          rollingSummary: 'Live conversation',
          lastActivityAt: '2026-04-01T10:00:00Z',
          providerId: 'provider-1',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(
      createWorkspaceDiff('conversation-live-1'),
    )
    const { listProjectConversationEntries } = await import('$lib/api/chat')
    vi.mocked(listProjectConversationEntries).mockResolvedValue({
      entries: [
        {
          id: 'entry-user-1',
          conversationId: 'conversation-live-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'user_message',
          payload: { content: 'Existing prompt' },
          createdAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    watchProjectConversation.mockImplementation(async (conversationId, handlers) => {
      streamHandlers.set(conversationId, handlers)
    })

    const { findByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await findByText('Existing prompt')
    await waitFor(() => {
      expect(streamHandlers.has('conversation-live-1')).toBe(true)
    })

    streamHandlers.get('conversation-live-1')?.onEvent({
      kind: 'message',
      payload: {
        type: 'text',
        content: 'First streamed reply chunk.',
      },
    })

    expect(await findByText('First streamed reply chunk.')).toBeTruthy()
  })
})
