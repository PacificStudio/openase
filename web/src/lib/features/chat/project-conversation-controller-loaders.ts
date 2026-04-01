import { listProjectConversations, type ProjectConversation } from '$lib/api/chat'
import {
  clearProjectConversationSelection,
  isCurrentProjectConversationOperation,
  setProjectConversationIdleIfCurrent,
  type ProjectConversationControllerState,
} from './project-conversation-controller-helpers'
import { loadProjectConversation, restoreProjectConversation } from './project-conversation-runtime'
import {
  mapPersistedEntries,
  type ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-state'
import {
  readProjectConversationId,
  storeProjectConversationId,
} from './project-conversation-storage'

type ConversationLoaderInput = {
  state: ProjectConversationControllerState
  currentOperationId: number
  projectId: string
  providerId: string
  connectStream: (conversationId: string) => void
}

function loadConversationIntoState(input: ConversationLoaderInput, conversationId: string) {
  return loadProjectConversation({
    conversationId,
    mapEntries: mapPersistedEntries,
    setConversationId: (nextId) => {
      if (isCurrentProjectConversationOperation(input.state, input.currentOperationId)) {
        input.state.conversationId = nextId
        storeProjectConversationId(input.projectId, input.providerId, nextId)
      }
    },
    setEntries: (nextEntries) => {
      if (isCurrentProjectConversationOperation(input.state, input.currentOperationId)) {
        input.state.entries = nextEntries as ProjectConversationTranscriptEntry[]
      }
    },
    resetActiveAssistantEntry: () => {
      if (isCurrentProjectConversationOperation(input.state, input.currentOperationId)) {
        input.state.activeAssistantEntryId = ''
      }
    },
    connectStream: input.connectStream,
  })
}

export async function restoreProjectConversationControllerState(params: {
  state: ProjectConversationControllerState
  projectId: string
  providerId: string
  currentOperationId: number
  connectStream: (conversationId: string) => void
  setConversations: (conversations: ProjectConversation[]) => void
  onError?: (message: string) => void
}) {
  try {
    params.setConversations(
      await listProjectConversations({
        projectId: params.projectId,
        providerId: params.providerId,
      }).then((payload) => payload.conversations),
    )
    await restoreProjectConversation({
      projectId: params.projectId,
      providerId: params.providerId,
      readConversationId: readProjectConversationId,
      storeConversationId: storeProjectConversationId,
      loadConversation: (conversationId) =>
        loadConversationIntoState(
          {
            state: params.state,
            currentOperationId: params.currentOperationId,
            projectId: params.projectId,
            providerId: params.providerId,
            connectStream: params.connectStream,
          },
          conversationId,
        ),
      clearConversation: () => {
        if (isCurrentProjectConversationOperation(params.state, params.currentOperationId)) {
          clearProjectConversationSelection(params.state)
        }
      },
    })
    setProjectConversationIdleIfCurrent(params.state, params.currentOperationId)
  } catch (caughtError) {
    if (!isCurrentProjectConversationOperation(params.state, params.currentOperationId)) {
      return
    }
    params.state.phase = 'idle'
    params.onError?.(
      caughtError instanceof Error
        ? caughtError.message
        : 'Failed to restore project conversation.',
    )
  }
}

export async function selectProjectConversationControllerState(params: {
  state: ProjectConversationControllerState
  projectId: string
  providerId: string
  conversationId: string
  currentOperationId: number
  connectStream: (conversationId: string) => void
  onError?: (message: string) => void
}) {
  try {
    await loadConversationIntoState(
      {
        state: params.state,
        currentOperationId: params.currentOperationId,
        projectId: params.projectId,
        providerId: params.providerId,
        connectStream: params.connectStream,
      },
      params.conversationId,
    )
    setProjectConversationIdleIfCurrent(params.state, params.currentOperationId)
  } catch (caughtError) {
    if (!isCurrentProjectConversationOperation(params.state, params.currentOperationId)) {
      return
    }
    params.state.phase = 'idle'
    params.onError?.(
      caughtError instanceof Error ? caughtError.message : 'Failed to switch project conversation.',
    )
  }
}
