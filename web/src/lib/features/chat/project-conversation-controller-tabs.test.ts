import { waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

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
  seedProjectConversationTabsStorage,
  startedTurnResponse,
} from './project-conversation-controller.test-helpers'

async function createControllerWithRestoredTabs() {
  seedProjectConversationTabsStorage(
    [
      { conversationId: 'conversation-1', providerId: 'provider-1' },
      { conversationId: 'conversation-2', providerId: 'provider-1' },
    ],
    0,
  )

  listProjectConversations.mockResolvedValue({
    conversations: [
      {
        id: 'conversation-1',
        rollingSummary: 'First conversation',
        lastActivityAt: '2026-04-01T10:00:00Z',
        providerId: 'provider-1',
      },
      {
        id: 'conversation-2',
        rollingSummary: 'Second conversation',
        lastActivityAt: '2026-04-01T10:05:00Z',
        providerId: 'provider-1',
      },
    ],
  })
  getProjectConversationWorkspaceDiff.mockImplementation(async (conversationId: string) =>
    createWorkspaceDiff(conversationId),
  )
  listProjectConversationEntries.mockImplementation(async (conversationId: string) => ({
    entries: [
      {
        id: `entry-${conversationId}`,
        conversationId,
        turnId: `turn-${conversationId}`,
        seq: 1,
        kind: 'user_message',
        payload: {
          content:
            conversationId === 'conversation-1'
              ? 'Loaded first conversation'
              : 'Loaded second conversation',
        },
        createdAt: '2026-04-01T10:00:00Z',
      },
    ],
  }))
  watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())

  const controller = createProjectConversationController({
    getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
    getProjectId: () => 'project-1',
  })
  controller.syncProviders(providerFixtures, 'provider-1')
  await controller.restore()

  const firstTabId =
    controller.tabs.find((tab) => tab.conversationId === 'conversation-1')?.id ?? ''
  const secondTabId =
    controller.tabs.find((tab) => tab.conversationId === 'conversation-2')?.id ?? ''

  return { controller, firstTabId, secondTabId }
}

describe('createProjectConversationController', () => {
  afterEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('does not crash when a conversation arrives without lastActivityAt', async () => {
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        updatedAt: '2026-04-01T10:00:00Z',
        createdAt: '2026-04-01T09:00:00Z',
      },
    })
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await expect(
      controller.sendTurn('Still works without activity timestamp'),
    ).resolves.toBeUndefined()

    expect(startProjectConversationTurn).toHaveBeenCalledWith('conversation-1', {
      message: 'Still works without activity timestamp',
      focus: undefined,
    })
    expect(controller.tabs[0]?.conversationId).toBe('conversation-1')
    expect(controller.phase).toBe('awaiting_reply')
  })

  it('blocks follow-up sends only for the same tab while an interrupt is pending', async () => {
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
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-2'))
    watchProjectConversationMux.mockImplementation((params) => {
      streamHandlers.set(params.conversationId, params)
      return resolvedMuxSubscription()
    })
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())
    respondProjectConversationInterrupt.mockResolvedValue({
      interrupt: {},
    })

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Need approval')
    streamHandlers.get('conversation-1')?.onEvent({
      kind: 'interrupt_requested',
      payload: {
        interruptId: 'interrupt-1',
        provider: 'codex',
        kind: 'user_input',
        options: [],
        payload: {
          questions: [
            {
              id: 'approval',
              question: 'Approve the next step?',
            },
          ],
        },
      },
    })

    expect(controller.phase).toBe('awaiting_interrupt')
    expect(controller.hasPendingInterrupt).toBe(true)
    expect(controller.inputDisabled).toBe(true)
    expect(controller.sendDisabled).toBe(true)

    await controller.sendTurn('This should stay blocked')
    expect(startProjectConversationTurn).toHaveBeenCalledTimes(1)

    controller.createTab()
    const secondTabId = controller.tabs.find((tab) => tab.conversationId === '')?.id ?? ''
    controller.selectTab(secondTabId)
    expect(controller.inputDisabled).toBe(false)
    expect(controller.sendDisabled).toBe(false)

    await controller.sendTurn('Independent tab keeps working')

    expect(startProjectConversationTurn).toHaveBeenNthCalledWith(2, 'conversation-2', {
      message: 'Independent tab keeps working',
      focus: undefined,
    })

    const firstTabId =
      controller.tabs.find((tab) => tab.conversationId === 'conversation-1')?.id ?? ''
    controller.selectTab(firstTabId)
    await controller.respondInterrupt({
      interruptId: 'interrupt-1',
      answer: {
        approval: {
          answers: ['yes'],
        },
      },
    })

    expect(respondProjectConversationInterrupt).toHaveBeenCalledWith(
      'conversation-1',
      'interrupt-1',
      {
        answer: {
          approval: {
            answers: ['yes'],
          },
        },
        decision: undefined,
      },
    )
  })

  it('restores persisted tabs, drops missing ones, and keeps the selected tab active', async () => {
    seedProjectConversationTabsStorage(
      [
        { conversationId: 'conversation-2', providerId: 'provider-1' },
        { conversationId: 'missing-conversation', providerId: 'provider-1' },
        { conversationId: 'conversation-1', providerId: 'provider-1' },
      ],
      2,
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
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.restore()

    expect(controller.tabs).toHaveLength(2)
    expect(controller.tabs.map((tab) => tab.conversationId)).toEqual([
      'conversation-2',
      'conversation-1',
    ])
    expect(controller.tabs.every((tab) => tab.restored)).toBe(true)
    expect(controller.tabs[0]?.needsHydration).toBe(true)
    expect(controller.tabs[1]?.needsHydration).toBe(false)
    expect(controller.conversationId).toBe('conversation-1')
    expect(controller.entries).toMatchObject([{ kind: 'text', content: 'Current conversation' }])
    expect(listProjectConversationEntries).toHaveBeenCalledTimes(1)
    expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledTimes(1)
  })

  it('closes only the selected tab and keeps the remaining tab active', async () => {
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
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('First tab')
    controller.createTab()
    const secondTabId = controller.tabs.find((tab) => tab.conversationId === '')?.id ?? ''
    controller.selectTab(secondTabId)
    await controller.sendTurn('Second tab')

    const firstTabId =
      controller.tabs.find((tab) => tab.conversationId === 'conversation-1')?.id ?? ''
    controller.closeTab(firstTabId)

    expect(controller.tabs).toHaveLength(1)
    expect(controller.conversationId).toBe('conversation-2')
    expect(controller.entries).toMatchObject([{ kind: 'text', content: 'Second tab' }])
  })

  it('hydrates the fallback tab immediately when closing the active restored tab', async () => {
    const { controller, firstTabId } = await createControllerWithRestoredTabs()

    expect(controller.conversationId).toBe('conversation-1')
    expect(controller.entries).toMatchObject([
      { kind: 'text', content: 'Loaded first conversation' },
    ])
    expect(
      controller.tabs.find((tab) => tab.conversationId === 'conversation-2')?.needsHydration,
    ).toBe(true)

    controller.closeTab(firstTabId)

    await waitFor(() => {
      expect(controller.conversationId).toBe('conversation-2')
      expect(controller.entries).toMatchObject([
        { kind: 'text', content: 'Loaded second conversation' },
      ])
    })
    expect(controller.tabs).toHaveLength(1)
    expect(controller.tabs[0]?.needsHydration).toBe(false)
    expect(listProjectConversationEntries).toHaveBeenCalledWith('conversation-2')
  })

  it('restores the same tab state for auto-selected and manually selected tabs', async () => {
    const manual = await createControllerWithRestoredTabs()
    const manualBaselineCalls = listProjectConversationEntries.mock.calls.length

    manual.controller.selectTab(manual.secondTabId)

    await waitFor(() => {
      expect(manual.controller.conversationId).toBe('conversation-2')
      expect(manual.controller.entries).toMatchObject([
        { kind: 'text', content: 'Loaded second conversation' },
      ])
    })
    const manualTab = manual.controller.tabs.find((tab) => tab.conversationId === 'conversation-2')
    expect(manualTab?.needsHydration).toBe(false)
    expect(listProjectConversationEntries.mock.calls.length - manualBaselineCalls).toBe(1)

    vi.clearAllMocks()
    window.localStorage.clear()

    const autoSelected = await createControllerWithRestoredTabs()
    const autoBaselineCalls = listProjectConversationEntries.mock.calls.length

    autoSelected.controller.closeTab(autoSelected.firstTabId)

    await waitFor(() => {
      expect(autoSelected.controller.conversationId).toBe('conversation-2')
      expect(autoSelected.controller.entries).toMatchObject([
        { kind: 'text', content: 'Loaded second conversation' },
      ])
    })
    const autoTab = autoSelected.controller.tabs.find(
      (tab) => tab.conversationId === 'conversation-2',
    )
    expect(autoTab?.needsHydration).toBe(false)
    expect(listProjectConversationEntries.mock.calls.length - autoBaselineCalls).toBe(1)
    expect(autoSelected.controller.entries).toEqual(manual.controller.entries)
  })

  it('retargets a blank tab when switching provider before the conversation starts', async () => {
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-2',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    expect(controller.tabs).toHaveLength(1)
    expect(controller.providerId).toBe('provider-1')

    await controller.selectProvider('provider-2')

    expect(controller.tabs).toHaveLength(1)
    expect(controller.providerId).toBe('provider-2')
    expect(controller.tabs[0]?.providerId).toBe('provider-2')

    await controller.sendTurn('Use Claude instead')

    expect(createProjectConversation).toHaveBeenCalledWith({
      providerId: 'provider-2',
      projectId: 'project-1',
    })
  })

  it('opens a new blank tab when switching provider after a conversation has started', async () => {
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
          providerId: 'provider-2',
          lastActivityAt: '2026-04-01T10:05:00Z',
        },
      })
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-2'))
    watchProjectConversationMux.mockReturnValue(resolvedMuxSubscription())
    startProjectConversationTurn.mockResolvedValue(startedTurnResponse())

    const controller = createProjectConversationController({
      getProjectContext: () => ({ projectId: 'project-1', projectName: 'Project 1' }),
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('First provider')
    expect(controller.phase).toBe('awaiting_reply')

    const streamHandler = watchProjectConversationMux.mock.calls[0]?.[0]?.onEvent as
      | ((event: { kind: string; payload: Record<string, unknown> }) => void)
      | undefined
    streamHandler?.({
      kind: 'turn_done',
      payload: {},
    })

    expect(controller.phase).toBe('idle')
    await controller.selectProvider('provider-2')

    expect(controller.tabs).toHaveLength(2)
    expect(controller.providerId).toBe('provider-2')
    expect(controller.tabs.map((tab) => tab.providerId)).toEqual(['provider-1', 'provider-2'])
    expect(controller.conversationId).toBe('')

    await controller.sendTurn('Second provider')

    expect(createProjectConversation).toHaveBeenNthCalledWith(2, {
      providerId: 'provider-2',
      projectId: 'project-1',
    })
    expect(controller.conversationId).toBe('conversation-2')
  })
})
