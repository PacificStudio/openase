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
  watchProjectConversationMuxStream.mockImplementation(async (_projectId, handlers) => {
    handlers.onOpen?.()
    await new Promise<void>((resolve) => {
      handlers.signal?.addEventListener('abort', () => resolve(), { once: true })
    })
  })
}

async function waitForComposerReady(getByPlaceholderText: (text: string) => HTMLElement) {
  await waitFor(() => {
    expect(
      (getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement).disabled,
    ).toBe(false)
  })
}

describe('ProjectConversationPanel tab close', () => {
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

  it('closes one tab without affecting the other tab', async () => {
    mockLiveMuxStream()
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
      turn: { id: 'turn-4', turn_index: 1, status: 'started' },
    })

    const { getByPlaceholderText, getByRole, queryByRole, findByRole } = render(
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
    await fireEvent.click(await findByRole('button', { name: 'Close anyway' }))

    expect(queryByRole('tab', { name: /^First tab$/ })).toBeNull()
    expect(getByRole('tab', { name: /^Second tab$/ })).toBeTruthy()
  })
})
