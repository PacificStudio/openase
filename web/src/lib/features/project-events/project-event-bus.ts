import { connectEventStream, type SSEFrame, type StreamConnectionState } from '$lib/api/sse'
import type { ProjectReconnectRecovery } from './project-reconnect-recovery'

export type ProjectEventEnvelope = {
  topic: string
  type: string
  payload: unknown
  publishedAt: string
}

export const projectDashboardRefreshTopic = 'project.dashboard.events'
export const projectDashboardRefreshType = 'project.dashboard.refresh'

export type ProjectDashboardRefreshSection =
  | 'project'
  | 'agents'
  | 'tickets'
  | 'activity'
  | 'hr_advisor'
  | 'organization_summary'

type ProjectEventListener = (event: ProjectEventEnvelope) => void
type ProjectEventStateListener = (state: StreamConnectionState) => void
type ProjectEventReconnectListener = () => void
type ProjectEventReconnectRecoveryListener = (recovery: ProjectReconnectRecovery) => void

type ProjectEventSubscriptionOptions = {
  onReconnect?: ProjectEventReconnectListener
  onReconnectRecovery?: ProjectEventReconnectRecoveryListener
}

type Runtime = {
  projectId: string
  retainers: number
  state: StreamConnectionState
  hasConnected: boolean
  reconnectSequence: number
  disconnect: (() => void) | null
  eventListeners: Set<ProjectEventListener>
  reconnectListeners: Set<ProjectEventReconnectListener>
  reconnectRecoveryListeners: Set<ProjectEventReconnectRecoveryListener>
  stateListeners: Set<ProjectEventStateListener>
}

const runtimes = new Map<string, Runtime>()

export function retainProjectEventBus(
  projectId: string,
  options: { onStateChange?: ProjectEventStateListener } = {},
) {
  const runtime = getRuntime(projectId)
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

export function subscribeProjectEvents(
  projectId: string,
  listener: ProjectEventListener,
  options: ProjectEventSubscriptionOptions = {},
) {
  const runtime = getRuntime(projectId)
  runtime.eventListeners.add(listener)
  if (options.onReconnect) {
    runtime.reconnectListeners.add(options.onReconnect)
  }
  if (options.onReconnectRecovery) {
    runtime.reconnectRecoveryListeners.add(options.onReconnectRecovery)
  }

  return () => {
    runtime.eventListeners.delete(listener)
    if (options.onReconnect) {
      runtime.reconnectListeners.delete(options.onReconnect)
    }
    if (options.onReconnectRecovery) {
      runtime.reconnectRecoveryListeners.delete(options.onReconnectRecovery)
    }
    cleanupRuntime(runtime)
  }
}

export function subscribeProjectEventBusState(
  projectId: string,
  listener: ProjectEventStateListener,
) {
  const runtime = getRuntime(projectId)
  runtime.stateListeners.add(listener)
  listener(runtime.state)

  return () => {
    runtime.stateListeners.delete(listener)
    cleanupRuntime(runtime)
  }
}

export function isProjectUpdateEvent(event: Pick<ProjectEventEnvelope, 'type' | 'topic'>) {
  return event.topic === 'activity.events' && event.type.startsWith('project_update_')
}

export const isTicketRunProjectEvent = (event: Pick<ProjectEventEnvelope, 'topic'>) =>
  event.topic === 'ticket.run.events'

export const isProjectDashboardRefreshEvent = (
  event: Pick<ProjectEventEnvelope, 'topic' | 'type'>,
) => event.topic === projectDashboardRefreshTopic && event.type === projectDashboardRefreshType

export function readProjectDashboardRefreshSections(
  event: Pick<ProjectEventEnvelope, 'payload'>,
): ProjectDashboardRefreshSection[] {
  const sections = readNestedArray(event.payload, ['dirty_sections'])
  if (!sections) {
    return []
  }

  const allowed = new Set<ProjectDashboardRefreshSection>([
    'project',
    'agents',
    'tickets',
    'activity',
    'hr_advisor',
    'organization_summary',
  ])
  const deduped = new Set<ProjectDashboardRefreshSection>()
  for (const section of sections) {
    if (allowed.has(section as ProjectDashboardRefreshSection)) {
      deduped.add(section as ProjectDashboardRefreshSection)
    }
  }
  return [...deduped]
}

export function projectEventAffectsTicketDetailReferences(
  event: Pick<ProjectEventEnvelope, 'topic' | 'type' | 'payload'>,
  ticketId: string,
) {
  if (event.topic === 'ticket.events')
    return readNestedString(event.payload, ['ticket', 'id']) !== ticketId
  if (event.topic !== 'activity.events') return false

  const eventType = readNestedString(event.payload, ['event', 'event_type']) ?? event.type
  return (
    eventType.startsWith('ticket_status_') ||
    eventType.startsWith('project_repo_') ||
    eventType === 'ticket.created'
  )
}

export function projectEventReferencesTicket(
  event: Pick<ProjectEventEnvelope, 'topic' | 'payload'>,
  ticketId: string,
) {
  switch (event.topic) {
    case 'ticket.events':
      return readNestedString(event.payload, ['ticket', 'id']) === ticketId
    case 'agent.events':
      return readNestedString(event.payload, ['agent', 'current_ticket_id']) === ticketId
    case 'activity.events':
      return readNestedString(event.payload, ['event', 'ticket_id']) === ticketId
    case 'ticket.run.events':
      return (
        readNestedString(event.payload, ['run', 'ticket_id']) === ticketId ||
        readNestedString(event.payload, ['entry', 'ticket_id']) === ticketId ||
        readNestedString(event.payload, ['ticket_id']) === ticketId
      )
    default:
      return false
  }
}

export function toProjectEventFrame(event: ProjectEventEnvelope): SSEFrame {
  return {
    event: event.type,
    data: JSON.stringify(event.payload),
  }
}

function getRuntime(projectId: string): Runtime {
  const existing = runtimes.get(projectId)
  if (existing) {
    return existing
  }

  const created: Runtime = {
    projectId,
    retainers: 0,
    state: 'idle',
    hasConnected: false,
    reconnectSequence: 0,
    disconnect: null,
    eventListeners: new Set(),
    reconnectListeners: new Set(),
    reconnectRecoveryListeners: new Set(),
    stateListeners: new Set(),
  }
  runtimes.set(projectId, created)
  return created
}

function ensureRuntimeConnection(runtime: Runtime) {
  if (runtime.disconnect) {
    return
  }

  const { projectId } = runtime
  runtime.disconnect = connectEventStream(`/api/v1/projects/${projectId}/events/stream`, {
    onEvent: (frame) => {
      const event = parseProjectEventEnvelope(frame)
      if (!event) {
        return
      }

      for (const listener of [...runtime.eventListeners]) {
        listener(event)
      }
    },
    onStateChange: (state) => {
      const reconnected = state === 'live' && runtime.hasConnected
      if (state === 'live') {
        runtime.hasConnected = true
      } else if (state === 'idle') {
        runtime.hasConnected = false
        runtime.reconnectSequence = 0
      }
      setRuntimeState(runtime, state)
      if (reconnected) {
        runtime.reconnectSequence += 1
        const recovery = { sequence: runtime.reconnectSequence }
        for (const listener of [...runtime.reconnectListeners]) {
          listener()
        }
        for (const listener of [...runtime.reconnectRecoveryListeners]) {
          listener(recovery)
        }
      }
    },
    onError: (error) => {
      console.error('Project event bus error:', error)
    },
  })
}

function cleanupRuntime(runtime: Runtime) {
  if (
    runtime.retainers === 0 &&
    runtime.disconnect === null &&
    runtime.eventListeners.size === 0 &&
    runtime.stateListeners.size === 0
  )
    runtimes.delete(runtime.projectId)
}

function setRuntimeState(runtime: Runtime, state: StreamConnectionState) {
  runtime.state = state
  for (const listener of [...runtime.stateListeners]) {
    listener(state)
  }
}

function parseProjectEventEnvelope(frame: Pick<SSEFrame, 'data'>): ProjectEventEnvelope | null {
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

function readNestedString(value: unknown, path: string[]) {
  let current: unknown = value
  for (const key of path) {
    if (!current || typeof current !== 'object' || !(key in current)) {
      return null
    }
    current = (current as Record<string, unknown>)[key]
  }

  return typeof current === 'string' ? current : null
}

function readNestedArray(value: unknown, path: string[]) {
  let current: unknown = value
  for (const key of path) {
    if (!current || typeof current !== 'object' || !(key in current)) {
      return null
    }
    current = (current as Record<string, unknown>)[key]
  }

  return Array.isArray(current)
    ? current.filter((item): item is string => typeof item === 'string')
    : null
}
