import {
  type ProjectConversationStreamEvent,
  watchProjectConversationMuxStream,
} from '$lib/api/chat'
import type { StreamConnectionState } from '$lib/api/sse'

type ProjectConversationEventSubscriber = {
  conversationId: string
  onEvent: (event: ProjectConversationStreamEvent) => void
  onReconnect?: () => void
  onConnected?: () => void
}

type ProjectConversationEventBusRuntime = {
  projectId: string
  state: StreamConnectionState
  disconnect: (() => void) | null
  subscribers: Map<number, ProjectConversationEventSubscriber>
  latestSessionByConversationId: Map<string, ProjectConversationStreamEvent>
  nextSubscriberId: number
  hasConnected: boolean
}

const runtimes = new Map<string, ProjectConversationEventBusRuntime>()

export function watchProjectConversationMux(params: {
  projectId: string
  conversationId: string
  signal?: AbortSignal
  onEvent: (event: ProjectConversationStreamEvent) => void
  onReconnect?: () => void
}) {
  let resolveConnected!: () => void
  const connected = new Promise<void>((resolve) => {
    resolveConnected = resolve
  })

  const projectId = params.projectId.trim()
  const conversationId = params.conversationId.trim()
  if (!projectId) {
    return {
      stream: Promise.reject(new Error('project conversation mux project id is required')),
      connected,
    }
  }
  if (!conversationId) {
    return {
      stream: Promise.reject(new Error('project conversation mux conversation id is required')),
      connected,
    }
  }

  const stream = new Promise<void>((resolve) => {
    if (params.signal?.aborted) {
      resolveConnected()
      resolve()
      return
    }

    const runtime = getRuntime(projectId)
    const subscriberId = runtime.nextSubscriberId
    runtime.nextSubscriberId += 1
    runtime.subscribers.set(subscriberId, {
      conversationId,
      onEvent: params.onEvent,
      onReconnect: params.onReconnect,
      onConnected: resolveConnected,
    })
    ensureRuntimeConnection(runtime)
    if (runtime.state === 'live') {
      resolveConnected()
    }

    const cachedSession = runtime.latestSessionByConversationId.get(conversationId)
    if (cachedSession) {
      params.onEvent(cachedSession)
    }

    const cleanup = () => {
      runtime.subscribers.delete(subscriberId)
      resolveConnected()
      cleanupRuntime(runtime)
      resolve()
    }

    params.signal?.addEventListener('abort', cleanup, { once: true })
  })

  return { stream, connected }
}

function getRuntime(projectId: string) {
  const existing = runtimes.get(projectId)
  if (existing) {
    return existing
  }

  const created: ProjectConversationEventBusRuntime = {
    projectId,
    state: 'idle',
    disconnect: null,
    subscribers: new Map(),
    latestSessionByConversationId: new Map(),
    nextSubscriberId: 1,
    hasConnected: false,
  }
  runtimes.set(projectId, created)
  return created
}

function ensureRuntimeConnection(runtime: ProjectConversationEventBusRuntime) {
  if (runtime.disconnect) {
    return
  }

  const controller = new AbortController()
  runtime.disconnect = () => {
    controller.abort()
    runtime.state = 'idle'
  }

  void runRuntimeConnection(runtime, controller.signal)
}

function cleanupRuntime(runtime: ProjectConversationEventBusRuntime) {
  if (runtime.subscribers.size > 0) {
    return
  }
  runtime.disconnect?.()
  runtime.disconnect = null
  runtime.state = 'idle'
  runtime.hasConnected = false
  runtime.latestSessionByConversationId.clear()
  runtimes.delete(runtime.projectId)
}

async function runRuntimeConnection(
  runtime: ProjectConversationEventBusRuntime,
  signal: AbortSignal,
) {
  let firstAttempt = true

  while (!signal.aborted) {
    runtime.state = firstAttempt ? 'connecting' : 'retrying'

    try {
      await watchProjectConversationMuxStream(runtime.projectId, {
        signal,
        onOpen: () => {
          const reconnected = runtime.hasConnected
          runtime.state = 'live'
          runtime.hasConnected = true
          for (const subscriber of runtime.subscribers.values()) {
            subscriber.onConnected?.()
          }
          if (reconnected) {
            for (const subscriber of runtime.subscribers.values()) {
              subscriber.onReconnect?.()
            }
          }
        },
        onFrame: (frame) => {
          if (frame.event.kind === 'session') {
            runtime.latestSessionByConversationId.set(frame.conversationId, frame.event)
          }

          for (const subscriber of runtime.subscribers.values()) {
            if (subscriber.conversationId === frame.conversationId) {
              subscriber.onEvent(frame.event)
            }
          }
        },
      })
    } catch (error) {
      if (isAbortError(error)) {
        return
      }
      console.error('Project conversation mux bus error:', error)
    }

    if (signal.aborted) {
      return
    }

    firstAttempt = false
    await waitForRetry(signal, 2000)
  }
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}

function waitForRetry(signal: AbortSignal, delayMs: number) {
  return new Promise<void>((resolve) => {
    if (signal.aborted) {
      resolve()
      return
    }

    const timeoutId = window.setTimeout(() => {
      signal.removeEventListener('abort', onAbort)
      resolve()
    }, delayMs)

    const onAbort = () => {
      window.clearTimeout(timeoutId)
      resolve()
    }

    signal.addEventListener('abort', onAbort, { once: true })
  })
}
