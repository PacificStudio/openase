import {
  createProjectConversation,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  type ProjectConversation,
} from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import {
  confirmProjectConversationActionProposal,
  resetProjectConversationRuntime,
} from './project-conversation-actions'
import {
  appendProjectConversationText,
  beginProjectConversationOperation,
  clearProjectConversationSelection,
  connectProjectConversationStream,
  invalidateProjectConversationStream,
  projectConversationHasPendingInterrupt,
  type ProjectConversationControllerState,
} from './project-conversation-controller-helpers'
import {
  restoreProjectConversationControllerState,
  selectProjectConversationControllerState,
} from './project-conversation-controller-loaders'
import { storeProjectConversationId } from './project-conversation-storage'
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
  let conversations = $state<ProjectConversation[]>([])
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
    get conversations() {
      return conversations
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
      const projectId = input.getProjectId()
      if (!projectId || !providerId || state.phase !== 'idle') {
        return
      }

      await restoreProjectConversationControllerState({
        state,
        projectId,
        providerId,
        currentOperationId: beginProjectConversationOperation(state, 'restoring'),
        connectStream,
        setConversations: (nextConversations) => {
          conversations = nextConversations
        },
        onError: input.onError,
      })
    },
    async selectProvider(nextProviderId: string) {
      if (!nextProviderId || providerId === nextProviderId || state.phase !== 'idle') {
        return
      }
      invalidateProjectConversationStream(state)
      providerId = nextProviderId
      conversations = []
      clearProjectConversationSelection(state)
    },
    async selectConversation(nextConversationId: string) {
      const projectId = input.getProjectId()
      if (!projectId || state.phase !== 'idle') {
        return
      }

      invalidateProjectConversationStream(state)
      state.activeAssistantEntryId = ''

      if (!nextConversationId) {
        clearProjectConversationSelection(state)
        return
      }

      await selectProjectConversationControllerState({
        state,
        projectId,
        providerId,
        conversationId: nextConversationId,
        currentOperationId: beginProjectConversationOperation(state, 'restoring'),
        connectStream,
        onError: input.onError,
      })
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
          if (currentOperationId !== state.operationId) {
            return
          }
          state.conversationId = createPayload.conversation.id
          conversations = [createPayload.conversation, ...conversations]
          storeProjectConversationId(projectId, providerId, state.conversationId)
          state.phase = 'connecting_stream'
          connectStream(state.conversationId)
        } else if (!state.abortController) {
          state.phase = 'connecting_stream'
          connectStream(state.conversationId)
        }

        if (currentOperationId !== state.operationId) {
          return
        }

        appendProjectConversationText(state, 'user', trimmed)
        state.activeAssistantEntryId = ''
        state.phase = 'submitting_turn'
        await startProjectConversationTurn(state.conversationId, trimmed)
        if (currentOperationId !== state.operationId) {
          return
        }
        if (projectConversationHasPendingInterrupt(state.entries)) {
          state.phase = 'awaiting_interrupt'
        } else if (state.phase === 'submitting_turn') {
          state.phase = 'awaiting_reply'
        }
      } catch (caughtError) {
        if (currentOperationId !== state.operationId) {
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
      if (currentOperationId !== state.operationId) {
        return
      }
      clearProjectConversationSelection(state)
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
