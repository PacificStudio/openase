import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const {
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  interruptAgent,
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
  interruptAgent: vi.fn(),
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
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
  watchProjectConversationMuxStream,
}))

vi.mock('$lib/api/openase', () => ({
  interruptAgent,
}))

import ProjectConversationPanel from './project-conversation-panel.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'

function mockLiveMuxStream() {
  watchProjectConversationMuxStream.mockImplementation(async (_projectId, handlers) => {
    handlers.onOpen?.()
    await new Promise<void>((resolve) => {
      handlers.signal?.addEventListener('abort', () => resolve(), { once: true })
    })
  })
}

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
    mockLiveMuxStream()
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
    await fireEvent.input(prompt, { target: { value: 'Help me figure out what to change here.' } })
    const sendButton = getByRole('button', { name: 'Send message' })
    await waitFor(() => {
      expect((sendButton as HTMLButtonElement).disabled).toBe(false)
    })
    await fireEvent.click(sendButton)

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
        message: 'Help me figure out what to change here.',
        focus: undefined,
      })
    })
  })

  it('offers a focused ticket agent interrupt action without conflating it with project runtime close', async () => {
    listProjectConversations.mockResolvedValue({ conversations: [] })
    mockLiveMuxStream()
    interruptAgent.mockResolvedValue({ agent: { id: 'agent-1', name: 'Backend Engineer' } })
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    const { getByRole, queryByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        focus: {
          kind: 'ticket',
          projectId: 'project-1',
          ticketId: 'ticket-1',
          ticketIdentifier: 'ASE-57',
          ticketTitle: 'Implement interrupt control',
          ticketStatus: 'In Progress',
          ticketAssignedAgent: {
            id: 'agent-1',
            name: 'Backend Engineer',
            runtimeControlState: 'active',
          },
          ticketCurrentRun: {
            id: 'run-1',
            status: 'executing',
          },
        },
      },
    })

    expect(queryByRole('button', { name: 'Close Runtime' })).toBeNull()

    await fireEvent.click(getByRole('button', { name: 'Interrupt Agent' }))

    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalledWith(
        'Interrupt "Backend Engineer"? This stops the current agent run. Use Close Runtime separately if you want to stop Project AI itself.',
      )
      expect(interruptAgent).toHaveBeenCalledWith('agent-1')
    })

    confirmSpy.mockRestore()
  })
})
