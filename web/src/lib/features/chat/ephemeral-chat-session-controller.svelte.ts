import { ApiError } from '$lib/api/client'
import {
  closeChatSession,
  streamChatTurn,
  type ChatSource,
  type ChatStreamEvent,
} from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import { listEphemeralChatProviders, pickDefaultEphemeralChatProvider } from './provider-options'
import {
  isAbortError,
  mapChatPayloadToTranscriptEntry,
  type EphemeralChatRole,
  type EphemeralChatTranscriptEntry,
} from './transcript'

type EphemeralChatContext = {
  projectId: string
  workflowId?: string
  ticketId?: string
}

type CreateEphemeralChatSessionControllerInput = {
  source: ChatSource
  onError?: (message: string) => void
}

type CloseSessionOptions = {
  clearEntries: boolean
  suppressError?: boolean
}

export function createEphemeralChatSessionController(
  input: CreateEphemeralChatSessionControllerInput,
) {
  let providers = $state<AgentProvider[]>([])
  let providerId = $state('')
  let pending = $state(false)
  let sessionId = $state('')
  let entries = $state<EphemeralChatTranscriptEntry[]>([])

  let entryCounter = 0
  let requestId = 0
  let abortController: AbortController | null = null

  function appendEntry(role: EphemeralChatRole, content: string) {
    entryCounter += 1
    entries = [...entries, { id: `entry-${entryCounter}`, role, content }]
  }

  function handleStreamEvent(event: ChatStreamEvent) {
    if (event.kind === 'done') {
      sessionId = event.payload.sessionId
      pending = false
      return
    }

    if (event.kind === 'error') {
      reportError(event.payload.message)
      pending = false
      return
    }

    const entry = mapChatPayloadToTranscriptEntry(event.payload)
    appendEntry(entry.role, entry.content)
  }

  function reportError(message: string) {
    input.onError?.(message)
  }

  async function closeActiveSession(options: CloseSessionOptions) {
    const activeSessionId = sessionId
    const closeRequestId = ++requestId

    abortController?.abort()
    abortController = null
    pending = false
    sessionId = ''

    if (options.clearEntries) {
      entries = []
    }

    if (!activeSessionId) {
      return
    }

    try {
      await closeChatSession(activeSessionId)
    } catch (caughtError) {
      if (options.suppressError || closeRequestId !== requestId) {
        return
      }
      reportError(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to close chat session.',
      )
    }
  }

  return {
    get providers() {
      return providers
    },
    get providerId() {
      return providerId
    },
    get selectedProvider() {
      return providers.find((provider) => provider.id === providerId) ?? null
    },
    get pending() {
      return pending
    },
    get sessionId() {
      return sessionId
    },
    get entries() {
      return entries
    },
    syncProviders(nextProviders: AgentProvider[], defaultProviderId: string | null | undefined) {
      providers = listEphemeralChatProviders(nextProviders)

      if (
        providerId &&
        providers.some((provider) => provider.id === providerId && provider.available)
      ) {
        return
      }

      const nextProviderId = pickDefaultEphemeralChatProvider(providers, defaultProviderId)
      if (providerId && providerId !== nextProviderId) {
        void closeActiveSession({ clearEntries: true, suppressError: true })
      }
      providerId = nextProviderId
    },
    async sendTurn(inputContext: { message: string; context: EphemeralChatContext }) {
      const message = inputContext.message.trim()
      if (!message || !inputContext.context.projectId || !providerId || pending) {
        return
      }

      appendEntry('user', message)
      pending = true

      const controller = new AbortController()
      const activeRequestId = ++requestId
      abortController = controller

      try {
        await streamChatTurn(
          {
            message,
            source: input.source,
            providerId,
            sessionId: sessionId || undefined,
            context: inputContext.context,
          },
          {
            signal: controller.signal,
            onEvent: (event) => {
              if (activeRequestId !== requestId) {
                return
              }
              handleStreamEvent(event)
            },
          },
        )
      } catch (caughtError) {
        if (activeRequestId === requestId && !isAbortError(caughtError)) {
          reportError(
            caughtError instanceof ApiError ? caughtError.detail : 'Ephemeral chat request failed.',
          )
        }
      } finally {
        if (activeRequestId === requestId && abortController === controller) {
          abortController = null
          pending = false
        }
      }
    },
    async resetConversation() {
      await closeActiveSession({ clearEntries: true })
    },
    async selectProvider(nextProviderId: string) {
      if (nextProviderId === providerId) {
        return
      }

      providerId = nextProviderId
      await closeActiveSession({ clearEntries: true })
    },
    async dispose() {
      await closeActiveSession({ clearEntries: false, suppressError: true })
    },
  }
}
