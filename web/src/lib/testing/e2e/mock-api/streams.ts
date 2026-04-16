import { encoder, nowIso } from './constants'
import { asString, findById } from './helpers'
import {
  projectConversationMuxStreamControllers,
  projectConversationStreamControllers,
  queuedProjectConversationFrames,
  queuedProjectConversationMuxFrames,
} from './stream-state'
import { getMockState } from './store'

export function streamResponse() {
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      controller.enqueue(encoder.encode(': openase-e2e\n\n'))
    },
    cancel() {},
  })

  return new Response(stream, {
    headers: {
      'content-type': 'text/event-stream',
      'cache-control': 'no-store',
      connection: 'keep-alive',
    },
  })
}

export function projectConversationStreamResponse(conversationId: string) {
  let sink: ReadableStreamDefaultController<Uint8Array> | null = null
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      sink = controller
      controller.enqueue(encoder.encode(': openase-e2e\n\n'))
      let sinks = projectConversationStreamControllers.get(conversationId)
      if (!sinks) {
        sinks = new Set()
        projectConversationStreamControllers.set(conversationId, sinks)
      }
      sinks.add(controller)

      const queuedFrames = queuedProjectConversationFrames.get(conversationId) ?? []
      for (const frame of queuedFrames) {
        controller.enqueue(encoder.encode(frame))
      }
      queuedProjectConversationFrames.delete(conversationId)
    },
    cancel() {
      if (!sink) {
        return
      }
      const sinks = projectConversationStreamControllers.get(conversationId)
      if (!sinks) {
        return
      }
      sinks.delete(sink)
      if (sinks.size === 0) {
        projectConversationStreamControllers.delete(conversationId)
      }
    },
  })

  return new Response(stream, {
    headers: {
      'content-type': 'text/event-stream',
      'cache-control': 'no-store',
      connection: 'keep-alive',
    },
  })
}

export function projectConversationMuxStreamResponse(projectId: string) {
  let sink: ReadableStreamDefaultController<Uint8Array> | null = null
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      sink = controller
      controller.enqueue(encoder.encode(': openase-e2e\n\n'))
      let sinks = projectConversationMuxStreamControllers.get(projectId)
      if (!sinks) {
        sinks = new Set()
        projectConversationMuxStreamControllers.set(projectId, sinks)
      }
      sinks.add(controller)

      const queuedFrames = queuedProjectConversationMuxFrames.get(projectId) ?? []
      for (const frame of queuedFrames) {
        controller.enqueue(encoder.encode(frame))
      }
      queuedProjectConversationMuxFrames.delete(projectId)
    },
    cancel() {
      if (!sink) {
        return
      }
      const sinks = projectConversationMuxStreamControllers.get(projectId)
      if (!sinks) {
        return
      }
      sinks.delete(sink)
      if (sinks.size === 0) {
        projectConversationMuxStreamControllers.delete(projectId)
      }
    },
  })

  return new Response(stream, {
    headers: {
      'content-type': 'text/event-stream',
      'cache-control': 'no-store',
      connection: 'keep-alive',
    },
  })
}

export function queueOrBroadcastProjectConversationFrame(conversationId: string, frame: string) {
  const sinks = projectConversationStreamControllers.get(conversationId)
  if (!sinks || sinks.size === 0) {
    const queued = queuedProjectConversationFrames.get(conversationId) ?? []
    queued.push(frame)
    queuedProjectConversationFrames.set(conversationId, queued)
    return
  }

  for (const sink of sinks) {
    try {
      sink.enqueue(encoder.encode(frame))
    } catch {
      sinks.delete(sink)
    }
  }

  if (sinks.size === 0) {
    projectConversationStreamControllers.delete(conversationId)
  }
}

export function queueOrBroadcastProjectConversationMuxFrame(projectId: string, frame: string) {
  const sinks = projectConversationMuxStreamControllers.get(projectId)
  if (!sinks || sinks.size === 0) {
    const queued = queuedProjectConversationMuxFrames.get(projectId) ?? []
    queued.push(frame)
    queuedProjectConversationMuxFrames.set(projectId, queued)
    return
  }

  for (const sink of sinks) {
    try {
      sink.enqueue(encoder.encode(frame))
    } catch {
      sinks.delete(sink)
    }
  }

  if (sinks.size === 0) {
    projectConversationMuxStreamControllers.delete(projectId)
  }
}

export function queueOrBroadcastProjectConversationEvent(
  conversationId: string,
  event: string,
  payload: Record<string, unknown>,
  sentAt: string,
) {
  queueOrBroadcastProjectConversationFrame(conversationId, encodeSSEFrame(event, payload))

  const conversation = findById(getMockState().projectConversations, conversationId)
  const projectId = asString(conversation?.project_id)
  if (!projectId) {
    return
  }

  queueOrBroadcastProjectConversationMuxFrame(
    projectId,
    encodeSSEFrame(event, {
      conversation_id: conversationId,
      sent_at: sentAt,
      payload,
    }),
  )
}

export function encodeSSEFrame(event: string, payload: Record<string, unknown>) {
  return `event: ${event}\ndata: ${JSON.stringify(payload)}\n\n`
}

export function shiftedIso(offsetMinutes: number) {
  return new Date(Date.parse(nowIso) + offsetMinutes * 60_000).toISOString()
}

export function nextProjectConversationSeq(conversationId: string) {
  return (
    getMockState().projectConversationEntries.filter(
      (entry) => entry.conversation_id === conversationId,
    ).length + 1
  )
}
