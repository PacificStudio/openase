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
})
