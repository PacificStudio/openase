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

describe('ProjectConversationPanel focus', () => {
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

  it('shows the current focus card and lets the user remove it for the next send', async () => {
    listProjectConversations.mockResolvedValue({ conversations: [] })
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { getByLabelText, getByPlaceholderText, getByRole, queryByText } = render(
      ProjectConversationPanel,
      {
        props: {
          context: { projectId: 'project-1' },
          providers: providerFixtures,
          defaultProviderId: 'provider-1',
          focus: {
            kind: 'workflow',
            projectId: 'project-1',
            workflowId: 'workflow-1',
            workflowName: 'Backend Engineer',
            workflowType: 'coding',
            harnessPath: '.openase/harnesses/backend.md',
            isActive: true,
            selectedArea: 'harness',
            hasDirtyDraft: true,
          },
          placeholder: 'Ask anything about this project…',
        },
      },
    )

    expect(queryByText('Workflow')).toBeTruthy()
    expect(queryByText('Backend Engineer / harness')).toBeTruthy()

    await fireEvent.click(getByLabelText('Remove focus for this send'))
    expect(queryByText('Workflow')).toBeNull()
    expect(queryByText('Backend Engineer / harness')).toBeNull()

    const prompt = getByPlaceholderText('Ask anything about this project…') as HTMLTextAreaElement
    await fireEvent.input(prompt, { target: { value: '帮我看看这里要怎么改' } })
    await fireEvent.click(getByRole('button', { name: 'Send message' }))

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
        message: '帮我看看这里要怎么改',
        focus: undefined,
      })
    })
  })
})
