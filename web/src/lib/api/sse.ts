export type StreamConnectionState = 'idle' | 'connecting' | 'live' | 'retrying'

export type SSEFrame = {
  event: string
  data: string
}

type StreamOptions = {
  onEvent: (frame: SSEFrame) => void
  onStateChange?: (state: StreamConnectionState) => void
  onError?: (error: unknown) => void
  retryDelayMs?: number
  activityTimeoutMs?: number
}

export const defaultRetryDelayMs = 2000
export const defaultActivityTimeoutMs = 10000

class StreamInactivityError extends Error {
  constructor(timeoutMs: number) {
    super(`stream went idle for ${timeoutMs}ms`)
    this.name = 'StreamInactivityError'
  }
}

export function connectEventStream(url: string, options: StreamOptions): () => void {
  let active = true
  let controller: AbortController | null = null
  const retryDelayMs = options.retryDelayMs ?? defaultRetryDelayMs
  const activityTimeoutMs = options.activityTimeoutMs ?? defaultActivityTimeoutMs

  async function run() {
    let firstAttempt = true

    while (active) {
      options.onStateChange?.(firstAttempt ? 'connecting' : 'retrying')
      controller = new AbortController()

      const failure = await openStream(url, controller.signal, options, activityTimeoutMs)
      if (failure === 'aborted') {
        return
      }
      if (failure) {
        options.onError?.(failure)
      }

      if (!active) {
        return
      }

      firstAttempt = false
      await wait(retryDelayMs)
    }
  }

  void run()

  return () => {
    active = false
    controller?.abort()
    options.onStateChange?.('idle')
  }
}

async function openStream(
  url: string,
  signal: AbortSignal,
  options: StreamOptions,
  activityTimeoutMs: number,
): Promise<unknown | 'aborted'> {
  try {
    const response = await fetch(url, {
      headers: { accept: 'text/event-stream' },
      signal,
    })
    if (!response.ok) {
      throw new Error(`stream request failed with status ${response.status}`)
    }
    if (!response.body) {
      throw new Error('stream response body is unavailable')
    }

    options.onStateChange?.('live')
    await consumeEventStream(response.body, options.onEvent, { activityTimeoutMs })
    return null
  } catch (error) {
    if (isAbortError(error)) {
      return 'aborted'
    }
    return error
  }
}

export async function consumeEventStream(
  stream: ReadableStream<Uint8Array>,
  onEvent: (frame: SSEFrame) => void,
  options: { activityTimeoutMs?: number } = {},
) {
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  const activityTimeoutMs = options.activityTimeoutMs ?? 0

  try {
    for (;;) {
      const { value, done } = await readWithActivityTimeout(reader, activityTimeoutMs)
      if (done) {
        buffer += decoder.decode()
        break
      }

      buffer += decoder.decode(value, { stream: true })
      buffer = emitBufferedFrames(buffer, onEvent)
    }
  } catch (error) {
    await reader.cancel().catch(() => {})
    throw error
  } finally {
    reader.releaseLock()
  }

  emitBufferedFrames(`${buffer}\n\n`, onEvent)
}

function emitBufferedFrames(buffer: string, onEvent: (frame: SSEFrame) => void): string {
  let cursor = 0

  for (;;) {
    const frameEnd = buffer.indexOf('\n\n', cursor)
    if (frameEnd === -1) {
      return buffer.slice(cursor)
    }

    const frame = parseFrame(buffer.slice(cursor, frameEnd))
    if (frame) {
      onEvent(frame)
    }

    cursor = frameEnd + 2
  }
}

function parseFrame(chunk: string): SSEFrame | null {
  const lines = chunk.split(/\r?\n/)
  let event = 'message'
  const dataLines: string[] = []

  for (const line of lines) {
    if (line === '' || line.startsWith(':')) {
      continue
    }

    if (line.startsWith('event:')) {
      event = line.slice('event:'.length).trim() || 'message'
      continue
    }

    if (line.startsWith('data:')) {
      dataLines.push(line.slice('data:'.length).trimStart())
    }
  }

  if (dataLines.length === 0) {
    return null
  }

  return {
    event,
    data: dataLines.join('\n'),
  }
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}

async function readWithActivityTimeout(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  timeoutMs: number,
) {
  if (timeoutMs <= 0) {
    return reader.read()
  }

  let timeoutId = 0
  try {
    return await Promise.race([
      reader.read(),
      new Promise<ReadableStreamReadResult<Uint8Array>>((_, reject) => {
        timeoutId = window.setTimeout(() => {
          reject(new StreamInactivityError(timeoutMs))
        }, timeoutMs)
      }),
    ])
  } finally {
    if (timeoutId !== 0) {
      window.clearTimeout(timeoutId)
    }
  }
}

function wait(durationMs: number) {
  return new Promise<void>((resolve) => {
    window.setTimeout(resolve, durationMs)
  })
}
