import { connectEventStream, type SSEFrame, type StreamConnectionState } from '$lib/api/sse'
import {
  dedupeActivityEvents,
  parseActivityEvent,
  parseStreamEnvelope,
  toErrorMessage,
} from '$lib/features/workspace'
import { loadTicketDetailData } from './api'
import { isHookEvent, shouldReloadTicket } from './stream'
import type { TicketDetailPayload } from './types'

type StartOptions = {
  projectId: string
  ticketId: string
}

type LoadOptions = {
  silent?: boolean
}

export function createTicketDetailStore() {
  let projectId = $state('')
  let ticketId = $state('')
  let loading = $state(true)
  let refreshing = $state(false)
  let errorMessage = $state('')
  let project = $state<Awaited<ReturnType<typeof loadTicketDetailData>>['project']>(null)
  let detail = $state<TicketDetailPayload | null>(null)
  let ticketStreamState = $state<StreamConnectionState>('idle')
  let activityStreamState = $state<StreamConnectionState>('idle')
  let hookStreamState = $state<StreamConnectionState>('idle')
  let loadInFlight = false
  let reloadQueued = false
  let streamCleanup: (() => void) | null = null

  async function start(options: StartOptions) {
    projectId = options.projectId
    ticketId = options.ticketId
    await load()
    connect()
  }

  function destroy() {
    streamCleanup?.()
    streamCleanup = null
  }

  async function load(options: LoadOptions = {}) {
    if (loadInFlight) {
      reloadQueued = true
      return
    }

    if (!projectId || !ticketId) {
      loading = false
      errorMessage = 'Missing project or ticket identifier in the URL.'
      return
    }

    loadInFlight = true
    if (options.silent) {
      refreshing = true
    } else {
      loading = true
    }
    errorMessage = ''

    try {
      const next = await loadTicketDetailData(projectId, ticketId)
      project = next.project
      detail = next.detail
    } catch (error) {
      errorMessage = toErrorMessage(error)
    } finally {
      const queuedReload = reloadQueued
      loadInFlight = false
      reloadQueued = false
      loading = false
      refreshing = false

      if (queuedReload) {
        void load({ silent: true })
      }
    }
  }

  function connect() {
    destroy()

    if (!projectId || !ticketId) {
      return
    }

    const closeTicketStream = connectEventStream(`/api/v1/projects/${projectId}/tickets/stream`, {
      onEvent: handleTicketFrame,
      onStateChange: (state) => {
        ticketStreamState = state
      },
      onError: (error) => {
        errorMessage = toErrorMessage(error)
      },
    })
    const closeActivityStream = connectEventStream(
      `/api/v1/projects/${projectId}/activity/stream`,
      {
        onEvent: handleActivityFrame,
        onStateChange: (state) => {
          activityStreamState = state
        },
        onError: (error) => {
          errorMessage = toErrorMessage(error)
        },
      },
    )
    const closeHookStream = connectEventStream(`/api/v1/projects/${projectId}/hooks/stream`, {
      onEvent: handleHookFrame,
      onStateChange: (state) => {
        hookStreamState = state
      },
      onError: (error) => {
        errorMessage = toErrorMessage(error)
      },
    })

    streamCleanup = () => {
      closeTicketStream()
      closeActivityStream()
      closeHookStream()
    }
  }

  function handleTicketFrame(frame: SSEFrame) {
    const envelope = parseStreamEnvelope(frame)
    if (!envelope || !shouldReloadTicket(envelope, ticketId)) {
      return
    }

    void load({ silent: true })
  }

  function handleActivityFrame(frame: SSEFrame) {
    const envelope = parseStreamEnvelope(frame)
    if (!envelope || !detail) {
      return
    }

    const activityEvent = parseActivityEvent(envelope.payload, envelope.published_at)
    if (!activityEvent || activityEvent.ticket_id !== ticketId) {
      return
    }

    detail = {
      ...detail,
      activity: dedupeActivityEvents([activityEvent, ...detail.activity]).slice(0, 100),
      hook_history: isHookEvent(activityEvent)
        ? dedupeActivityEvents([activityEvent, ...detail.hook_history]).slice(0, 50)
        : detail.hook_history,
    }
  }

  function handleHookFrame(frame: SSEFrame) {
    const envelope = parseStreamEnvelope(frame)
    if (!envelope || !shouldReloadTicket(envelope, ticketId)) {
      return
    }

    void load({ silent: true })
  }

  return {
    get projectId() {
      return projectId
    },
    get ticketId() {
      return ticketId
    },
    get loading() {
      return loading
    },
    get refreshing() {
      return refreshing
    },
    get errorMessage() {
      return errorMessage
    },
    get project() {
      return project
    },
    get detail() {
      return detail
    },
    get ticketStreamState() {
      return ticketStreamState
    },
    get activityStreamState() {
      return activityStreamState
    },
    get hookStreamState() {
      return hookStreamState
    },
    start,
    destroy,
    load,
  }
}
