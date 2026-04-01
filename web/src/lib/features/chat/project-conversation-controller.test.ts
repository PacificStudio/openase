import { afterEach, describe, expect, it, vi } from 'vitest'

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

import { createProjectConversationController } from './project-conversation-controller.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'

function deferredPromise<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

describe('createProjectConversationController', () => {
  afterEach(() => {
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('does not wait for the background stream before submitting the first turn', async () => {
    const stream = deferredPromise<void>()

    createProjectConversation.mockResolvedValue({
      conversation: { id: 'conversation-1' },
    })
    watchProjectConversation.mockReturnValue(stream.promise)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Summarize this project.')

    expect(createProjectConversation).toHaveBeenCalledWith({
      providerId: 'provider-1',
      projectId: 'project-1',
    })
    expect(watchProjectConversation).toHaveBeenCalledWith(
      'conversation-1',
      expect.objectContaining({
        signal: expect.any(AbortSignal),
        onEvent: expect.any(Function),
      }),
    )
    expect(startProjectConversationTurn).toHaveBeenCalledWith(
      'conversation-1',
      'Summarize this project.',
    )
    expect(controller.phase).toBe('awaiting_reply')
    expect(controller.entries).toMatchObject([
      { kind: 'text', role: 'user', content: 'Summarize this project.' },
    ])

    stream.resolve()
  })

  it('locks duplicate sends during first-turn bootstrap', async () => {
    const create = deferredPromise<{ conversation: { id: string } }>()

    createProjectConversation.mockReturnValue(create.promise)
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    const firstSend = controller.sendTurn('First question')
    expect(controller.phase).toBe('creating_conversation')
    expect(controller.inputDisabled).toBe(true)

    await controller.sendTurn('Second question')

    expect(createProjectConversation).toHaveBeenCalledTimes(1)
    expect(startProjectConversationTurn).not.toHaveBeenCalled()

    create.resolve({ conversation: { id: 'conversation-1' } })
    await firstSend

    expect(startProjectConversationTurn).toHaveBeenCalledTimes(1)
    expect(controller.entries.filter((entry) => entry.kind === 'text')).toMatchObject([
      { role: 'user', content: 'First question' },
    ])
  })

  it('blocks follow-up sends while an interrupt is pending and resumes cleanly after a response', async () => {
    let streamHandlers:
      | {
          onEvent: (event: { kind: string; payload: Record<string, unknown> }) => void
        }
      | undefined

    createProjectConversation.mockResolvedValue({
      conversation: { id: 'conversation-1' },
    })
    watchProjectConversation.mockImplementation(async (_conversationId, handlers) => {
      streamHandlers = handlers
    })
    startProjectConversationTurn.mockImplementation(async () => {
      streamHandlers?.onEvent({
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
      return { turn: { id: 'turn-1', turn_index: 1, status: 'started' } }
    })
    respondProjectConversationInterrupt.mockResolvedValue({
      interrupt: {},
    })

    const controller = createProjectConversationController({
      getProjectId: () => 'project-1',
    })
    controller.syncProviders(providerFixtures, 'provider-1')

    await controller.sendTurn('Need approval')

    expect(controller.phase).toBe('awaiting_interrupt')
    expect(controller.hasPendingInterrupt).toBe(true)
    expect(controller.inputDisabled).toBe(true)

    await controller.sendTurn('This should stay blocked')
    expect(startProjectConversationTurn).toHaveBeenCalledTimes(1)

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
    expect(controller.phase).toBe('awaiting_reply')
  })
})
