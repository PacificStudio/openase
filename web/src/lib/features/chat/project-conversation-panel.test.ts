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

  it('restores only the active persisted tab with a restored badge', async () => {
    window.localStorage.setItem(
      'openase.project-conversation.project-1.provider-1',
      JSON.stringify({
        conversationIds: ['conversation-2', 'conversation-1'],
        activeConversationId: 'conversation-1',
      }),
    )

    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Current conversation',
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
    getProjectConversationWorkspaceDiff.mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
    listProjectConversationEntries.mockImplementation(async (conversationId: string) => ({
      entries:
        conversationId === 'conversation-1'
          ? [
              {
                id: 'entry-1',
                conversationId: 'conversation-1',
                turnId: 'turn-1',
                seq: 1,
                kind: 'user_message',
                payload: { content: 'Current conversation' },
                createdAt: '2026-04-01T10:00:00Z',
              },
            ]
          : [],
    }))
    watchProjectConversation.mockResolvedValue(undefined)

    const { findAllByText, getAllByRole, getByRole, queryByRole } = render(
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

    expect((await findAllByText('Current conversation')).length).toBeGreaterThanOrEqual(1)
    expect((await findAllByText('Restored')).length).toBeGreaterThanOrEqual(1)
    expect(getByRole('tab', { name: /^Current conversation Restored$/ })).toBeTruthy()
    expect(queryByRole('tab', { name: /^Older discussion Restored$/ })).toBeNull()
    expect(getAllByRole('tab', { name: /Restored$/ }).length).toBe(1)
  })

  it('isolates unsent drafts per tab', async () => {
    listProjectConversations.mockResolvedValue({ conversations: [] })

    const { getAllByRole, getByPlaceholderText, getByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    const prompt = getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement
    await fireEvent.input(prompt, { target: { value: 'First draft' } })

    await fireEvent.click(getByRole('button', { name: /New Tab/i }))
    const secondPrompt = getByPlaceholderText(
      'Ask anything about this project…',
    ) as HTMLTextAreaElement
    expect(secondPrompt.value).toBe('')

    await fireEvent.input(secondPrompt, { target: { value: 'Second draft' } })
    const tabs = getAllByRole('tab')

    await fireEvent.click(tabs[0] as HTMLElement)
    expect(
      (getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement).value,
    ).toBe('First draft')

    await fireEvent.click(tabs[1] as HTMLElement)
    expect(
      (getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement).value,
    ).toBe('Second draft')
  })

  it('only injects initialPrompt when the active tab draft is empty', async () => {
    listProjectConversations.mockResolvedValue({ conversations: [] })

    const { getByPlaceholderText, rerender } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
        initialPrompt: 'Seed prompt',
      },
    })

    const prompt = getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement
    await waitFor(() => {
      expect(prompt.value).toBe('Seed prompt')
    })

    await fireEvent.input(prompt, { target: { value: 'Keep my draft' } })
    await rerender({
      context: { projectId: 'project-1' },
      providers: providerFixtures,
      defaultProviderId: 'provider-1',
      placeholder: 'Ask anything about this project…',
      initialPrompt: 'Replacement prompt',
    })
    expect(prompt.value).toBe('Keep my draft')

    await fireEvent.input(prompt, { target: { value: '' } })
    await rerender({
      context: { projectId: 'project-1' },
      providers: providerFixtures,
      defaultProviderId: 'provider-1',
      placeholder: 'Ask anything about this project…',
      initialPrompt: 'Fresh prompt',
    })
    await waitFor(() => {
      expect(prompt.value).toBe('Fresh prompt')
    })
  })

  it('closes one tab without affecting the other tab', async () => {
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
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-4', turn_index: 1, status: 'started' },
    })

    const { getByPlaceholderText, getByRole, queryByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    const prompt = getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement
    await fireEvent.input(prompt, { target: { value: 'First tab' } })
    await fireEvent.click(getByRole('button', { name: 'Send message' }))
    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
        message: 'First tab',
        focus: undefined,
      })
    })

    await fireEvent.click(getByRole('button', { name: /New Tab/i }))
    const secondPrompt = getByPlaceholderText(
      'Ask anything about this project…',
    ) as HTMLTextAreaElement
    await fireEvent.input(secondPrompt, { target: { value: 'Second tab' } })
    await fireEvent.click(getByRole('button', { name: 'Send message' }))
    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-2', {
        message: 'Second tab',
        focus: undefined,
      })
    })

    await fireEvent.click(getByRole('button', { name: 'Close First tab' }))

    expect(queryByRole('tab', { name: /^First tab$/ })).toBeNull()
    expect(getByRole('tab', { name: /^Second tab$/ })).toBeTruthy()
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
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-claude-1'))
    watchProjectConversation.mockImplementation(async (conversationId, handlers) => {
      streamHandlers.set(conversationId, handlers)
    })
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { findByText, getByPlaceholderText, getByRole } = render(ProjectConversationPanel, {
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

    expect(await findByText('Claude session · requires_action')).toBeTruthy()
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
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-codex-1'))
    watchProjectConversation.mockImplementation(async (conversationId, handlers) => {
      streamHandlers.set(conversationId, handlers)
    })
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

    expect(await findByText('Codex thread · waitingOnApproval')).toBeTruthy()
  })
})
