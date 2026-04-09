import { describe, expect, it, vi } from 'vitest'
import { sendNextQueuedProjectConversationTurn } from './project-conversation-controller-actions'
import {
  createProjectConversationTabState,
  type QueuedProjectTurn,
} from './project-conversation-controller-state'

function deferredPromise<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

function createQueuedTurn(id: string, message: string): QueuedProjectTurn {
  return {
    id,
    message,
    focus: null,
    createdAt: `2026-04-08T18:00:0${id}Z`,
  }
}

describe('sendNextQueuedProjectConversationTurn', () => {
  it('removes the in-flight queued turn immediately and keeps later turns queued through a busy retry', async () => {
    const firstSend = deferredPromise<boolean>()
    let busy = false
    const tab = createProjectConversationTabState(1, 'provider-1', false, 'project-1', 'Project 1')
    tab.queuedTurns = [
      createQueuedTurn('1', 'First queued message'),
      createQueuedTurn('2', 'Second queued message'),
    ]

    const sendTurnInTab = vi.fn(async (_tab, message: string) => {
      if (busy) {
        return false
      }
      if (message === 'First queued message') {
        busy = true
        const sent = await firstSend.promise
        busy = false
        return sent
      }
      return true
    })

    const firstDispatch = sendNextQueuedProjectConversationTurn({ tab, sendTurnInTab })
    await Promise.resolve()

    expect(sendTurnInTab).toHaveBeenNthCalledWith(1, tab, 'First queued message', null)
    expect(tab.queuedTurns).toMatchObject([{ id: '2', message: 'Second queued message' }])

    await expect(sendNextQueuedProjectConversationTurn({ tab, sendTurnInTab })).resolves.toBe(false)
    expect(sendTurnInTab).toHaveBeenNthCalledWith(2, tab, 'Second queued message', null)
    expect(tab.queuedTurns).toMatchObject([{ id: '2', message: 'Second queued message' }])

    firstSend.resolve(true)
    await expect(firstDispatch).resolves.toBe(true)

    await expect(sendNextQueuedProjectConversationTurn({ tab, sendTurnInTab })).resolves.toBe(true)
    expect(sendTurnInTab).toHaveBeenNthCalledWith(3, tab, 'Second queued message', null)
    expect(tab.queuedTurns).toHaveLength(0)
  })
})
