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

const seedProjectConversationTabsStorage = (
  tabs: Array<{ conversationId: string; providerId: string; draft?: string }>,
  activeTabIndex: number,
) =>
  window.localStorage.setItem(
    'openase.project-conversation.project-1',
    JSON.stringify({
      tabs: tabs.map((tab) => ({
        conversationId: tab.conversationId,
        providerId: tab.providerId,
        draft: tab.draft ?? '',
      })),
      activeTabIndex,
    }),
  )

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

  it('restores all persisted tabs and keeps the selected one active', async () => {
    seedProjectConversationTabsStorage(
      [
        { conversationId: 'conversation-2', providerId: 'provider-1' },
        { conversationId: 'conversation-1', providerId: 'provider-1' },
      ],
      1,
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
    expect(getByRole('tab', { name: /^Older discussion Restored$/ })).toBeTruthy()
    expect(queryByRole('tab', { name: /^New tab$/ })).toBeNull()
    expect(getAllByRole('tab', { name: /Restored$/ }).length).toBe(2)
  })

  it('clears the restoring status message once the conversation is loaded', async () => {
    seedProjectConversationTabsStorage(
      [{ conversationId: 'conversation-1', providerId: 'provider-1' }],
      0,
    )

    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Current conversation',
          lastActivityAt: '2026-04-01T10:00:00Z',
          providerId: 'provider-1',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
    listProjectConversationEntries.mockResolvedValue({
      entries: [
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
    })
    watchProjectConversation.mockResolvedValue(undefined)

    const { findAllByText, queryByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await findAllByText('Current conversation')
    await waitFor(() => {
      expect(queryByText('Restoring this project conversation…')).toBeNull()
    })
  })

  it('isolates unsent drafts per tab', async () => {
    listProjectConversations.mockResolvedValue({ conversations: [] })

    const { getByPlaceholderText, getByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await waitFor(() => {
      expect((getByRole('button', { name: /New Tab/i }) as HTMLButtonElement).disabled).toBe(false)
    })

    const prompt = getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement
    await fireEvent.input(prompt, { target: { value: 'First draft' } })

    await fireEvent.click(getByRole('button', { name: /New Tab/i }))
    const secondPrompt = getByPlaceholderText(
      'Ask anything about this project…',
    ) as HTMLTextAreaElement
    expect(secondPrompt.value).toBe('')

    await fireEvent.input(secondPrompt, { target: { value: 'Second draft' } })
    await fireEvent.click(getByRole('tab', { name: /^First draft$/ }))
    expect(
      (getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement).value,
    ).toBe('First draft')

    await fireEvent.click(getByRole('tab', { name: /^Second draft$/ }))
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
})
