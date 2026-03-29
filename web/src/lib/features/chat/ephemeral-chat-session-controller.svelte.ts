import { ApiError } from '$lib/api/client'
import {
  closeChatSession,
  streamChatTurn,
  type ChatSource,
  type ChatStreamEvent,
} from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import { executeActionProposal, type ChatActionExecutionResult } from './action-proposal-executor'
import { listEphemeralChatProviders, pickDefaultEphemeralChatProvider } from './provider-options'
import { formatEphemeralChatUsageSummary } from './session-policy'
import {
  createTextTranscriptEntry,
  isAbortError,
  isActionProposalEntry,
  isTextTranscriptEntry,
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
  getSource: () => ChatSource
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
  let activeAssistantEntryId = ''

  function appendEntry(
    role: EphemeralChatRole,
    content: string,
    options?: { streaming?: boolean },
  ) {
    entryCounter += 1
    entries = [
      ...entries,
      createTextTranscriptEntry(`entry-${entryCounter}`, role, content, options),
    ]
  }

  function updateEntry(
    entryId: string,
    update: (entry: EphemeralChatTranscriptEntry) => EphemeralChatTranscriptEntry,
  ) {
    entries = entries.map((entry) => (entry.id === entryId ? update(entry) : entry))
  }

  function appendAssistantText(content: string) {
    if (!activeAssistantEntryId) {
      entryCounter += 1
      activeAssistantEntryId = `entry-${entryCounter}`
      entries = [
        ...entries,
        createTextTranscriptEntry(activeAssistantEntryId, 'assistant', content, {
          streaming: true,
        }),
      ]
      return
    }

    updateEntry(activeAssistantEntryId, (entry) => {
      if (!isTextTranscriptEntry(entry) || entry.role !== 'assistant') {
        return entry
      }

      return {
        ...entry,
        content: `${entry.content}${content}`,
        streaming: true,
      }
    })
  }

  function finalizeAssistantTextEntry() {
    if (!activeAssistantEntryId) {
      return
    }

    const entryId = activeAssistantEntryId
    activeAssistantEntryId = ''
    updateEntry(entryId, (entry) => {
      if (!isTextTranscriptEntry(entry) || entry.role !== 'assistant') {
        return entry
      }

      return {
        ...entry,
        streaming: false,
      }
    })
  }

  function appendMappedEntry(event: Extract<ChatStreamEvent, { kind: 'message' }>) {
    entryCounter += 1
    entries = [...entries, mapChatPayloadToTranscriptEntry(`entry-${entryCounter}`, event.payload)]
  }

  function handleStreamEvent(event: ChatStreamEvent) {
    if (event.kind === 'session') {
      sessionId = event.payload.sessionId
      return
    }

    if (event.kind === 'done') {
      sessionId = event.payload.sessionId
      finalizeAssistantTextEntry()
      appendEntry('system', formatEphemeralChatUsageSummary(input.getSource(), event.payload))
      pending = false
      return
    }

    if (event.kind === 'error') {
      finalizeAssistantTextEntry()
      reportError(event.payload.message)
      pending = false
      return
    }

    const messageEvent = event as Extract<ChatStreamEvent, { kind: 'message' }>
    const payload = messageEvent.payload
    if (payload.type === 'text') {
      appendAssistantText((payload as Extract<typeof payload, { type: 'text' }>).content)
      return
    }

    appendMappedEntry(messageEvent)
  }

  function reportError(message: string) {
    input.onError?.(message)
  }

  async function closeActiveSession(options: CloseSessionOptions) {
    const activeSessionId = sessionId
    const closeRequestId = ++requestId

    abortController?.abort()
    abortController = null
    activeAssistantEntryId = ''
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

  function updateActionProposalEntry(
    entryId: string,
    nextState: {
      status: 'executing' | 'confirmed' | 'cancelled'
      results: ChatActionExecutionResult[]
    },
  ) {
    entries = entries.map((entry) => {
      if (!isActionProposalEntry(entry) || entry.id !== entryId) {
        return entry
      }

      return {
        ...entry,
        status: nextState.status,
        results: nextState.results,
      }
    })
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
    get hasPendingActionProposal() {
      return entries.some((entry) => isActionProposalEntry(entry) && entry.status === 'pending')
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
      activeAssistantEntryId = ''
      pending = true

      const controller = new AbortController()
      const activeRequestId = ++requestId
      abortController = controller

      try {
        await streamChatTurn(
          {
            message,
            source: input.getSource(),
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
    async confirmActionProposal(entryId: string) {
      const entry = entries.find((item) => item.id === entryId)
      if (!entry || !isActionProposalEntry(entry) || entry.status !== 'pending') {
        return
      }

      updateActionProposalEntry(entryId, {
        status: 'executing',
        results: [],
      })

      const results = await executeActionProposal(entry.proposal)
      updateActionProposalEntry(entryId, {
        status: 'confirmed',
        results,
      })
      appendEntry('system', summarizeExecutionResults(results))
    },
    cancelActionProposal(entryId: string) {
      const entry = entries.find((item) => item.id === entryId)
      if (!entry || !isActionProposalEntry(entry) || entry.status !== 'pending') {
        return
      }

      updateActionProposalEntry(entryId, {
        status: 'cancelled',
        results: [],
      })
      appendEntry('system', 'Cancelled the proposed platform actions.')
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

function summarizeExecutionResults(results: ChatActionExecutionResult[]) {
  if (results.length === 0) {
    return 'No platform actions were proposed.'
  }

  const successCount = results.filter((result) => result.ok).length
  if (successCount === results.length) {
    return `Executed ${successCount} proposed platform action${successCount === 1 ? '' : 's'}.`
  }

  if (successCount === 0) {
    return `All ${results.length} proposed platform actions failed.`
  }

  return `Executed ${successCount} of ${results.length} proposed platform actions successfully.`
}
