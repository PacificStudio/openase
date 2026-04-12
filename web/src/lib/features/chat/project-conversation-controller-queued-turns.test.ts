import { waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

const {
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  interruptProjectConversationTurn,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
} = vi.hoisted(() => ({
  closeProjectConversationRuntime: vi.fn(),
  createProjectConversation: vi.fn(),
  executeProjectConversationActionProposal: vi.fn(),
  getProjectConversation: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  interruptProjectConversationTurn: vi.fn(),
  listProjectConversationEntries: vi.fn(),
  listProjectConversations: vi.fn(),
  respondProjectConversationInterrupt: vi.fn(),
  startProjectConversationTurn: vi.fn(),
}))

const { watchProjectConversationMux } = vi.hoisted(() => ({
  watchProjectConversationMux: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  interruptProjectConversationTurn,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversationMuxStream: vi.fn(),
}))

vi.mock('./project-conversation-event-bus', () => ({
  watchProjectConversationMux,
}))

import { createProjectConversationController } from './project-conversation-controller.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'
import {
  createWorkspaceDiff,
  resolvedMuxSubscription,
  startedTurnResponse,
} from './project-conversation-controller.test-helpers'

describe('createProjectConversationController queued turn dispatch', () => {
  afterEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('auto-dispatches a queued turn when its background tab becomes idle', async () => {
    const streamHandlers = new Map<
      string,
      { onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void }
    >()

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
    getProjectConversationWorkspaceDiff.mockImplementation(async (conversationId: string) =>
      createWorkspaceDiff(conversationId),
    )
    watchProjectConversationMux.mockImplementation((params) => {
      streamHandlers.set(params.conversationId, params)
      return resolvedMuxSubscription()
    })
    startProjectConversationTurn
      .mockResolvedValueOnce(startedTurnResponse('conversation-1'))
      .mockResolvedValueOnce(startedTurnResponse('conversation-2'))
      .mockResolvedValueOnce(startedTurnResponse('conversation-1'))

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('First question')
    expect(controller.enqueueTurn('Queued follow-up')).toBe(true)

    controller.createTab()
    const secondTabId = controller.tabs.find((tab) => tab.conversationId === '')?.id ?? ''
    controller.selectTab(secondTabId)
    await controller.sendTurn('Second tab stays foreground')

    expect(controller.conversationId).toBe('conversation-2')

    streamHandlers.get('conversation-1')?.onEvent({
      kind: 'turn_done',
      payload: {},
    })

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenNthCalledWith(3, 'conversation-1', {
        message: 'Queued follow-up',
        focus: undefined,
      })
    })
    expect(
      controller.tabs.find((tab) => tab.conversationId === 'conversation-1')?.queuedTurns,
    ).toHaveLength(0)
    expect(controller.conversationId).toBe('conversation-2')
  })

  it('flushes queued turns only in the tab that owns them after a background ready transition', async () => {
    const streamHandlers = new Map<
      string,
      {
        onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void
        onReconnect?: () => void
        onRetrying?: () => void
      }
    >()

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
    getProjectConversationWorkspaceDiff.mockImplementation(async (conversationId: string) =>
      createWorkspaceDiff(conversationId),
    )
    watchProjectConversationMux.mockImplementation((params) => {
      streamHandlers.set(params.conversationId, params)
      return resolvedMuxSubscription()
    })
    startProjectConversationTurn
      .mockResolvedValueOnce(startedTurnResponse('conversation-1'))
      .mockResolvedValueOnce(startedTurnResponse('conversation-2'))
      .mockResolvedValueOnce(startedTurnResponse('conversation-1'))
    getProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:06:00Z',
        runtimePrincipal: {
          runtimeState: 'ready',
        },
      },
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Background tab in flight')
    expect(controller.enqueueTurn('Queued only for tab A')).toBe(true)

    controller.createTab()
    const secondTabId = controller.tabs.find((tab) => tab.conversationId === '')?.id ?? ''
    controller.selectTab(secondTabId)
    await controller.sendTurn('Foreground tab B')

    streamHandlers.get('conversation-1')?.onRetrying?.()
    streamHandlers.get('conversation-1')?.onReconnect?.()

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenNthCalledWith(3, 'conversation-1', {
        message: 'Queued only for tab A',
        focus: undefined,
      })
    })
    expect(
      startProjectConversationTurn.mock.calls.filter(
        ([conversationId, payload]) =>
          conversationId === 'conversation-2' && payload.message === 'Queued only for tab A',
      ),
    ).toHaveLength(0)
    expect(controller.conversationId).toBe('conversation-2')
  })
})
