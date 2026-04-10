import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const {
  createProjectConversation,
  getProjectConversationWorkspaceDiff,
  interruptProjectConversationTurn,
  listProjectConversations,
  startProjectConversationTurn,
  watchProjectConversation,
  watchProjectConversationMuxStream,
} = vi.hoisted(() => ({
  createProjectConversation: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  interruptProjectConversationTurn: vi.fn(),
  listProjectConversations: vi.fn(),
  startProjectConversationTurn: vi.fn(),
  watchProjectConversation: vi.fn(),
  watchProjectConversationMuxStream: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeProjectConversationRuntime: vi.fn(),
  createProjectConversation,
  executeProjectConversationActionProposal: vi.fn(),
  getProjectConversation: vi.fn(),
  getProjectConversationWorkspaceDiff,
  interruptProjectConversationTurn,
  listProjectConversationEntries: vi.fn(),
  listProjectConversations,
  respondProjectConversationInterrupt: vi.fn(),
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

describe('ProjectConversationPanel stop action', () => {
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

  it('shows Stop while a turn is streaming and preserves partial output after stopping', async () => {
    const mux = mockLiveMuxStream()

    listProjectConversations.mockResolvedValue({ conversations: [] })
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-stop-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(
      createWorkspaceDiff('conversation-stop-1'),
    )
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
      conversation: {
        id: 'conversation-stop-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    interruptProjectConversationTurn.mockResolvedValue(undefined)

    const { findByRole, findByText, getByPlaceholderText, getByRole, queryByRole } = render(
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

    await fireEvent.input(getByPlaceholderText('Ask anything about this project…'), {
      target: { value: 'Stop this reply when it starts' },
    })
    await fireEvent.click(getByRole('button', { name: 'Send message' }))

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-stop-1', {
        message: 'Stop this reply when it starts',
        focus: undefined,
      })
    })

    const stopButton = await findByRole('button', { name: 'Stop reply' })
    expect(stopButton).toBeTruthy()

    mux.emit('conversation-stop-1', {
      kind: 'message',
      payload: {
        type: 'text',
        content: 'Partial streamed reply.',
      },
    })
    expect(await findByText('Partial streamed reply.')).toBeTruthy()

    await fireEvent.click(stopButton)
    expect(interruptProjectConversationTurn).toHaveBeenCalledWith('conversation-stop-1')
    expect(await findByText('Stopping the current reply…')).toBeTruthy()

    mux.emit('conversation-stop-1', {
      kind: 'message',
      payload: {
        type: 'turn_interrupted',
        raw: {
          message: 'Turn stopped by user.',
          reason: 'stopped_by_user',
        },
      },
    })
    mux.emit('conversation-stop-1', {
      kind: 'interrupted',
      payload: {
        conversationId: 'conversation-stop-1',
        turnId: 'turn-1',
        message: 'Turn stopped by user.',
        reason: 'stopped_by_user',
      },
    })
    mux.emit('conversation-stop-1', {
      kind: 'session',
      payload: {
        conversationId: 'conversation-stop-1',
        runtimeState: 'ready',
        providerStatus: 'ready',
        providerActiveFlags: [],
      },
    })

    expect(await findByText('Turn stopped')).toBeTruthy()
    expect(await findByText('Partial streamed reply.')).toBeTruthy()
    await waitFor(() => {
      expect(queryByRole('button', { name: 'Stop reply' })).toBeNull()
    })
  })
})
