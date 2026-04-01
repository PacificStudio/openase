import {
  createProjectConversation,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
} from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import {
  confirmProjectConversationActionProposal,
  resetProjectConversationRuntime,
} from './project-conversation-actions'
import {
  appendProjectConversationText,
  beginProjectConversationOperation,
  connectProjectConversationStream,
  invalidateProjectConversationStream,
  isCurrentProjectConversationOperation,
  projectConversationHasPendingInterrupt,
  setProjectConversationIdleIfCurrent,
  type ProjectConversationControllerState,
} from './project-conversation-controller-helpers'
import {
  mapPersistedEntries,
  type ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-state'
import {
  readProjectConversationId,
  storeProjectConversationId,
} from './project-conversation-storage'
import { loadProjectConversation, restoreProjectConversation } from './project-conversation-runtime'
import {
  listEphemeralChatProviders,
  pickDefaultEphemeralChatProvider,
  shouldKeepEphemeralChatProvider,
} from './provider-options'

type CreateProjectConversationControllerInput = {
  getProjectId: () => string
  onError?: (message: string) => void
}

export function createProjectConversationController(
  input: CreateProjectConversationControllerInput,
) {
  let providers = $state<AgentProvider[]>([])
  let providerId = $state('')
  const state = $state<ProjectConversationControllerState>({
    phase: 'idle',
    conversationId: '',
    entries: [],
    activeAssistantEntryId: '',
    abortController: null,
    entryCounter: 0,
    operationId: 0,
    streamId: 0,
  })

  function connectStream(nextConversationId: string) {
    connectProjectConversationStream({
      state,
      conversationId: nextConversationId,
      getProjectId: input.getProjectId,
      getProviderId: () => providerId,
      onError: input.onError,
    })
  }
  return {
    get providers() {
      return providers
    },
    get providerId() {
      return providerId
    },
    get phase() {
      return state.phase
    },
    get selectedProvider() {
      return providers.find((provider) => provider.id === providerId) ?? null
    },
    get busy() {
      return state.phase !== 'idle'
    },
    get pending() {
      return (
        state.phase === 'creating_conversation' ||
        state.phase === 'connecting_stream' ||
        state.phase === 'submitting_turn' ||
        state.phase === 'awaiting_reply'
      )
    },
    get conversationId() {
      return state.conversationId
    },
    get entries() {
      return state.entries
    },
    get hasPendingInterrupt() {
      return projectConversationHasPendingInterrupt(state.entries)
    },
    get inputDisabled() {
      return (
        !input.getProjectId() ||
        !providerId ||
        state.phase !== 'idle' ||
        projectConversationHasPendingInterrupt(state.entries)
      )
    },
    get providerSelectionDisabled() {
      return state.phase !== 'idle'
    },
    syncProviders(nextProviders: AgentProvider[], defaultProviderId: string | null | undefined) {
      providers = listEphemeralChatProviders(nextProviders)
      if (shouldKeepEphemeralChatProvider(providers, providerId)) {
        return
      }
      providerId = pickDefaultEphemeralChatProvider(providers, defaultProviderId)
    },
    async restore() {
      if (!input.getProjectId() || !providerId || state.phase !== 'idle') {
        return
      }
      const currentOperationId = beginProjectConversationOperation(state, 'restoring')
      try {
        await restoreProjectConversation({
          projectId: input.getProjectId(),
          providerId,
          readConversationId: readProjectConversationId,
          storeConversationId: storeProjectConversationId,
          loadConversation: async (nextConversationId) => {
            if (!isCurrentProjectConversationOperation(state, currentOperationId)) {
              return
            }
            await loadProjectConversation({
              conversationId: nextConversationId,
              mapEntries: mapPersistedEntries,
              setConversationId: (nextId) => {
                if (isCurrentProjectConversationOperation(state, currentOperationId)) {
                  state.conversationId = nextId
                }
              },
              setEntries: (nextEntries) => {
                if (isCurrentProjectConversationOperation(state, currentOperationId)) {
                  state.entries = nextEntries as ProjectConversationTranscriptEntry[]
                }
              },
              resetActiveAssistantEntry: () => {
                if (isCurrentProjectConversationOperation(state, currentOperationId)) {
                  state.activeAssistantEntryId = ''
                }
              },
              connectStream,
            })
          },
          clearConversation: () => {
            if (isCurrentProjectConversationOperation(state, currentOperationId)) {
              state.conversationId = ''
              state.entries = []
            }
          },
        })
        setProjectConversationIdleIfCurrent(state, currentOperationId)
      } catch (caughtError) {
        if (!isCurrentProjectConversationOperation(state, currentOperationId)) {
          return
        }
        state.phase = 'idle'
        input.onError?.(
          caughtError instanceof Error
            ? caughtError.message
            : 'Failed to restore project conversation.',
        )
      }
    },
    async selectProvider(nextProviderId: string) {
      if (!nextProviderId || providerId === nextProviderId || state.phase !== 'idle') {
        return
      }
      invalidateProjectConversationStream(state)
      providerId = nextProviderId
      state.conversationId = ''
      state.entries = []
      state.activeAssistantEntryId = ''
    },
    async sendTurn(message: string) {
      const trimmed = message.trim()
      const projectId = input.getProjectId()
      if (
        !trimmed ||
        !projectId ||
        !providerId ||
        state.phase !== 'idle' ||
        projectConversationHasPendingInterrupt(state.entries)
      ) {
        return
      }
      const currentOperationId = beginProjectConversationOperation(
        state,
        state.conversationId ? 'submitting_turn' : 'creating_conversation',
      )

      try {
        if (!state.conversationId) {
          const createPayload = await createProjectConversation({ providerId, projectId })
          if (!isCurrentProjectConversationOperation(state, currentOperationId)) {
            return
          }
          state.conversationId = createPayload.conversation.id
          storeProjectConversationId(projectId, providerId, state.conversationId)
          state.phase = 'connecting_stream'
          connectStream(state.conversationId)
        } else if (!state.abortController) {
          state.phase = 'connecting_stream'
          connectStream(state.conversationId)
        }

        if (!isCurrentProjectConversationOperation(state, currentOperationId)) {
          return
        }

        appendProjectConversationText(state, 'user', trimmed)
        state.activeAssistantEntryId = ''
        state.phase = 'submitting_turn'
        await startProjectConversationTurn(state.conversationId, trimmed)
        if (!isCurrentProjectConversationOperation(state, currentOperationId)) {
          return
        }
        if (projectConversationHasPendingInterrupt(state.entries)) {
          state.phase = 'awaiting_interrupt'
        } else if (state.phase === 'submitting_turn') {
          state.phase = 'awaiting_reply'
        }
      } catch (caughtError) {
        if (!isCurrentProjectConversationOperation(state, currentOperationId)) {
          return
        }
        state.phase = 'idle'
        input.onError?.(
          caughtError instanceof Error ? caughtError.message : 'Failed to send project message.',
        )
      }
    },
    async resetConversation() {
      const currentOperationId = beginProjectConversationOperation(state, 'resetting')
      const activeConversationId = state.conversationId
      invalidateProjectConversationStream(state)
      await resetProjectConversationRuntime(activeConversationId)
      if (!isCurrentProjectConversationOperation(state, currentOperationId)) {
        return
      }
      state.conversationId = ''
      state.entries = []
      state.activeAssistantEntryId = ''
      state.phase = 'idle'
    },
    async confirmActionProposal(entryId: string) {
      if (!state.conversationId) {
        return
      }
      await confirmProjectConversationActionProposal({
        conversationId: state.conversationId,
        entryId,
        entries: state.entries,
        setEntries: (nextEntries) => {
          state.entries = nextEntries
        },
        onError: input.onError,
      })
    },
    async respondInterrupt(inputValue: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) {
      if (!state.conversationId || state.phase !== 'awaiting_interrupt') {
        return
      }
      state.phase = 'awaiting_reply'
      try {
        await respondProjectConversationInterrupt(state.conversationId, inputValue.interruptId, {
          decision: inputValue.decision,
          answer: inputValue.answer,
        })
      } catch (caughtError) {
        state.phase = 'awaiting_interrupt'
        input.onError?.(
          caughtError instanceof Error ? caughtError.message : 'Failed to answer interrupt.',
        )
      }
    },
    dispose() {
      invalidateProjectConversationStream(state)
      state.phase = 'idle'
    },
  }
}
