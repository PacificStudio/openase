import { connectEventStream, type SSEFrame, type StreamConnectionState } from '$lib/api/sse'

export type OrganizationEventScope = 'machines' | 'providers'

export type OrganizationEventEnvelope = {
  topic: string
  type: string
  payload: unknown
  publishedAt: string
}

type OrganizationEventListener = (event: OrganizationEventEnvelope) => void
type OrganizationEventStateListener = (state: StreamConnectionState) => void

type Runtime = {
  orgId: string
  scope: OrganizationEventScope
  retainers: number
  state: StreamConnectionState
  disconnect: (() => void) | null
  eventListeners: Set<OrganizationEventListener>
  stateListeners: Set<OrganizationEventStateListener>
}

const runtimes = new Map<string, Runtime>()

export function retainOrganizationEventBus(
  orgId: string,
  scope: OrganizationEventScope,
  options: { onStateChange?: OrganizationEventStateListener } = {},
) {
  const runtime = getRuntime(orgId, scope)
  runtime.retainers += 1

  if (options.onStateChange) {
    runtime.stateListeners.add(options.onStateChange)
    options.onStateChange(runtime.state)
  }

  ensureRuntimeConnection(runtime)

  return () => {
    if (options.onStateChange) {
      runtime.stateListeners.delete(options.onStateChange)
    }
    runtime.retainers = Math.max(0, runtime.retainers - 1)
    if (runtime.retainers === 0) {
      runtime.disconnect?.()
      runtime.disconnect = null
      setRuntimeState(runtime, 'idle')
    }
    cleanupRuntime(runtime)
  }
}

export function subscribeOrganizationEvents(
  orgId: string,
  scope: OrganizationEventScope,
  listener: OrganizationEventListener,
) {
  const runtime = getRuntime(orgId, scope)
  runtime.eventListeners.add(listener)

  return () => {
    runtime.eventListeners.delete(listener)
    cleanupRuntime(runtime)
  }
}

export function subscribeOrganizationMachineEvents(
  orgId: string,
  listener: OrganizationEventListener,
) {
  return subscribeOrganizationEvents(orgId, 'machines', listener)
}

export function resetOrganizationEventBusForTests() {
  for (const runtime of runtimes.values()) {
    runtime.disconnect?.()
  }
  runtimes.clear()
}

function getRuntime(orgId: string, scope: OrganizationEventScope) {
  const key = buildRuntimeKey(orgId, scope)
  const existing = runtimes.get(key)
  if (existing) {
    return existing
  }

  const created: Runtime = {
    orgId,
    scope,
    retainers: 0,
    state: 'idle',
    disconnect: null,
    eventListeners: new Set(),
    stateListeners: new Set(),
  }
  runtimes.set(key, created)
  return created
}

function ensureRuntimeConnection(runtime: Runtime) {
  if (runtime.disconnect) {
    return
  }

  runtime.disconnect = connectEventStream(
    organizationEventStreamPath(runtime.orgId, runtime.scope),
    {
      onEvent: (frame) => {
        const event = parseOrganizationEventEnvelope(frame)
        if (!event) {
          return
        }
        for (const listener of [...runtime.eventListeners]) {
          listener(event)
        }
      },
      onStateChange: (state) => {
        setRuntimeState(runtime, state)
      },
      onError: (error) => {
        console.error(`Organization ${runtime.scope} event bus error:`, error)
      },
    },
  )
}

function organizationEventStreamPath(orgId: string, scope: OrganizationEventScope) {
  switch (scope) {
    case 'machines':
      return `/api/v1/orgs/${orgId}/machines/stream`
    case 'providers':
      return `/api/v1/orgs/${orgId}/providers/stream`
  }
}

function cleanupRuntime(runtime: Runtime) {
  if (
    runtime.retainers === 0 &&
    runtime.disconnect === null &&
    runtime.eventListeners.size === 0 &&
    runtime.stateListeners.size === 0
  ) {
    runtimes.delete(buildRuntimeKey(runtime.orgId, runtime.scope))
  }
}

function setRuntimeState(runtime: Runtime, state: StreamConnectionState) {
  runtime.state = state
  for (const listener of [...runtime.stateListeners]) {
    listener(state)
  }
}

function parseOrganizationEventEnvelope(
  frame: Pick<SSEFrame, 'data'>,
): OrganizationEventEnvelope | null {
  try {
    const raw = JSON.parse(frame.data) as Record<string, unknown>
    const topic = typeof raw.topic === 'string' ? raw.topic : ''
    const type = typeof raw.type === 'string' ? raw.type : ''
    if (!topic || !type) {
      return null
    }

    return {
      topic,
      type,
      payload: raw.payload ?? null,
      publishedAt: typeof raw.published_at === 'string' ? raw.published_at : '',
    }
  } catch {
    return null
  }
}

function buildRuntimeKey(orgId: string, scope: OrganizationEventScope) {
  return `${orgId}:${scope}`
}
