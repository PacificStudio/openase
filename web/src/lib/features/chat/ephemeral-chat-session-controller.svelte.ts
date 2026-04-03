import { ApiError } from '$lib/api/client'
import { closeChatSession, streamChatTurn, type ChatSource } from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import {
  appendEphemeralChatEntry,
  clearEphemeralChatSessionState,
  finalizeEphemeralAssistantText,
  handleEphemeralChatStreamEvent,
  type EphemeralChatSessionState,
} from './ephemeral-chat-session-state'
import { describeTurnFailure } from './ephemeral-chat-turn-failure'
import {
  listProviderCapabilityProviders,
  pickDefaultProviderCapability,
  shouldKeepProviderCapability,
  type ProviderCapabilityName,
} from './provider-options'
import { isAbortError } from './transcript'

type EphemeralChatContext = {
  projectId: string
  workflowId?: string
  ticketId?: string
  harnessDraft?: string
  skillId?: string
  skillFilePath?: string
  skillFileDraft?: string
}

type CreateEphemeralChatSessionControllerInput = {
  getSource: () => ChatSource
  capability?: ProviderCapabilityName
  onError?: (message: string) => void
}

type CloseSessionOptions = { clearEntries: boolean; suppressError?: boolean }

export function createEphemeralChatSessionController(
  input: CreateEphemeralChatSessionControllerInput,
) {
  const capability = input.capability ?? 'ephemeral_chat'
  let providers = $state<AgentProvider[]>([])
  let providerId = $state('')
  const state = $state<EphemeralChatSessionState>({
    pending: false,
    sessionId: '',
    entries: [],
    entryCounter: 0,
    activeAssistantEntryId: '',
  })
  let requestId = 0
  let abortController: AbortController | null = null

  async function closeActiveSession(options: CloseSessionOptions) {
    const activeSessionId = state.sessionId
    const closeRequestId = ++requestId
    abortController?.abort()
    abortController = null
    clearEphemeralChatSessionState(state, { clearEntries: options.clearEntries })
    if (!activeSessionId) {
      return
    }
    try {
      await closeChatSession(activeSessionId)
    } catch (caughtError) {
      if (options.suppressError || closeRequestId !== requestId) {
        return
      }
      input.onError?.(
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
      return state.pending
    },
    get sessionId() {
      return state.sessionId
    },
    get entries() {
      return state.entries
    },
    syncProviders(nextProviders: AgentProvider[], defaultProviderId: string | null | undefined) {
      providers = listProviderCapabilityProviders(nextProviders, capability)
      if (shouldKeepProviderCapability(providers, providerId, capability)) {
        return
      }
      const nextProviderId = pickDefaultProviderCapability(providers, defaultProviderId, capability)
      if (providerId && providerId !== nextProviderId) {
        void closeActiveSession({ clearEntries: true, suppressError: true })
      }
      providerId = nextProviderId
    },
    async sendTurn(inputContext: { message: string; context: EphemeralChatContext }) {
      const message = inputContext.message.trim()
      if (!message || !inputContext.context.projectId || !providerId || state.pending) {
        return
      }
      appendEphemeralChatEntry(state, 'user', message)
      state.activeAssistantEntryId = ''
      state.pending = true
      const controller = new AbortController()
      const activeRequestId = ++requestId
      abortController = controller
      let streamStarted = false
      let partialReplyReceived = false
      try {
        await streamChatTurn(
          {
            message,
            source: input.getSource(),
            providerId,
            sessionId: state.sessionId || undefined,
            context: inputContext.context,
          },
          {
            signal: controller.signal,
            onEvent: (event) => {
              if (activeRequestId !== requestId) {
                return
              }
              streamStarted = true
              if (event.kind === 'message') {
                const payload = event.payload
                if (payload.type === 'text' && 'content' in payload && payload.content) {
                  partialReplyReceived = true
                }
              }
              handleEphemeralChatStreamEvent(state, event, {
                source: input.getSource(),
                onError: input.onError,
              })
            },
          },
        )
      } catch (caughtError) {
        if (activeRequestId === requestId && !isAbortError(caughtError)) {
          finalizeEphemeralAssistantText(state)
          const errorMessage = describeTurnFailure(caughtError, {
            streamStarted,
            partialReplyReceived,
          })
          console.error('Ephemeral chat turn failed', {
            source: input.getSource(),
            providerId,
            sessionId: state.sessionId,
            context: inputContext.context,
            streamStarted,
            partialReplyReceived,
            error: caughtError,
            errorMessage,
          })
          input.onError?.(errorMessage)
        }
      } finally {
        if (activeRequestId === requestId && abortController === controller) {
          abortController = null
          state.pending = false
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
