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
})
