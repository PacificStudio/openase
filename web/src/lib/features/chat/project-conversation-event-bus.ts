import {
  parseRawProjectConversationMuxFrame,
  type ProjectConversationStreamEvent,
} from '$lib/api/chat'
import { connectEventStream, type StreamConnectionState } from '$lib/api/sse'

type ProjectConversationEventSubscriber = {
  conversationId: string
  onEvent: (event: ProjectConversationStreamEvent) => void
  onReconnect?: () => void
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
  const projectId = params.projectId.trim()
  const conversationId = params.conversationId.trim()
  if (!projectId) {
    return Promise.reject(new Error('project conversation mux project id is required'))
  }
  if (!conversationId) {
    return Promise.reject(new Error('project conversation mux conversation id is required'))
  }

  return new Promise<void>((resolve) => {
    if (params.signal?.aborted) {
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
    })
    ensureRuntimeConnection(runtime)

    const cachedSession = runtime.latestSessionByConversationId.get(conversationId)
    if (cachedSession) {
      params.onEvent(cachedSession)
    }

    const cleanup = () => {
      runtime.subscribers.delete(subscriberId)
      cleanupRuntime(runtime)
      resolve()
    }

    params.signal?.addEventListener('abort', cleanup, { once: true })
  })
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

  runtime.disconnect = connectEventStream(
    `/api/v1/chat/projects/${encodeURIComponent(runtime.projectId)}/conversations/stream`,
    {
      onEvent: (frame) => {
        const parsed = parseRawProjectConversationMuxFrame(frame)
        if (!parsed.ok) {
          console.error('Project conversation mux parse error:', parsed.error)
          return
        }

        if (parsed.value.event.kind === 'session') {
          runtime.latestSessionByConversationId.set(parsed.value.conversationId, parsed.value.event)
        }

        for (const subscriber of runtime.subscribers.values()) {
          if (subscriber.conversationId === parsed.value.conversationId) {
            subscriber.onEvent(parsed.value.event)
          }
        }
      },
      onStateChange: (state) => {
        const previousState = runtime.state
        runtime.state = state
        if (state === 'live') {
          const reconnected = runtime.hasConnected && previousState !== 'live'
          runtime.hasConnected = true
          if (reconnected) {
            for (const subscriber of runtime.subscribers.values()) {
              subscriber.onReconnect?.()
            }
          }
        }
      },
      onError: (error) => {
        console.error('Project conversation mux bus error:', error)
      },
    },
  )
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
