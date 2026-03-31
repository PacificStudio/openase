import {
  createProjectConversation,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  type ProjectConversationStreamEvent,
} from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import {
  confirmProjectConversationActionProposal,
  resetProjectConversationRuntime,
} from './project-conversation-actions'
import {
  appendProjectConversationAssistantChunk,
  appendProjectConversationTextEntry,
  finalizeProjectConversationAssistantEntry,
  mapPersistedEntries,
  type ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-state'
import { handleProjectConversationStreamEvent } from './project-conversation-stream'
import {
  readProjectConversationId,
  storeProjectConversationId,
} from './project-conversation-storage'
import {
  loadProjectConversation,
  restoreProjectConversation,
  startProjectConversationStream,
} from './project-conversation-runtime'
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
  let pending = $state(false)
  let conversationId = $state('')
  let entries = $state<ProjectConversationTranscriptEntry[]>([])
  let activeAssistantEntryId = $state('')
  let abortController: AbortController | null = null
  let requestId = 0
  let entryCounter = 0

  function appendTextEntry(role: 'user' | 'assistant' | 'system', content: string) {
    entryCounter += 1
    const nextEntries = appendProjectConversationTextEntry(entries, role, content, {
      entryId: `entry-${entryCounter}`,
    })
    entries = nextEntries
  }

  function finalizeAssistantEntry() {
    entries = finalizeProjectConversationAssistantEntry(entries, activeAssistantEntryId)
    activeAssistantEntryId = ''
  }

  function appendAssistantChunk(content: string, turnId?: string) {
    if (!activeAssistantEntryId) {
      entryCounter += 1
    }
    const next = appendProjectConversationAssistantChunk({
      entries,
      activeAssistantEntryId,
      nextEntryId: activeAssistantEntryId || `entry-${entryCounter}`,
      content,
      turnId,
    })
    entries = next.entries
    activeAssistantEntryId = next.activeAssistantEntryId
  }

  async function connectStream(nextConversationId: string) {
    const started = startProjectConversationStream({
      conversationId: nextConversationId,
      abortController,
      requestId,
      onEvent: handleStreamEvent,
      onError: (message) => input.onError?.(message),
    })
    abortController = started.controller
    requestId = started.requestId
    await started.stream
  }
  function handleStreamEvent(event: ProjectConversationStreamEvent) {
    handleProjectConversationStreamEvent(event, {
      appendAssistantChunk,
      finalizeAssistantEntry,
      appendActionProposal: (entryId, payload) => {
        entries = [
          ...entries,
          {
            id: entryId ?? `entry-${++entryCounter}`,
            kind: 'action_proposal',
            role: 'assistant',
            proposal: payload as never,
            status: 'pending',
            results: [],
          },
        ]
      },
      appendDiff: (entryId, payload) => {
        entries = [
          ...entries,
          {
            id: entryId ?? `entry-${++entryCounter}`,
            kind: 'diff',
            role: 'assistant',
            diff: payload as never,
          },
        ]
      },
      confirmActionResult: (entryId, results) => {
        entries = entries.map((entry) =>
          entry.kind === 'action_proposal' && entry.id === entryId
            ? { ...entry, status: 'confirmed', results }
            : entry,
        )
      },
      appendInterrupt: (payload) => {
        entryCounter += 1
        entries = [
          ...entries,
          {
            id: `entry-${entryCounter}`,
            kind: 'interrupt',
            role: 'system',
            interruptId: payload.interruptId,
            provider: payload.provider,
            interruptKind: payload.kind,
            payload: payload.payload,
            options: payload.options,
            status: 'pending',
          },
        ]
      },
      resolveInterrupt: (interruptId, decision) => {
        entries = entries.map((entry) =>
          entry.kind === 'interrupt' && entry.interruptId === interruptId
            ? { ...entry, status: 'resolved', decision }
            : entry,
        )
      },
      setConversationId: (nextConversationId) => {
        conversationId = nextConversationId
        storeProjectConversationId(input.getProjectId(), providerId, conversationId)
      },
      setPending: (value) => (pending = value),
      onError: (message) => input.onError?.(message),
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
    get conversationId() {
      return conversationId
    },
    get entries() {
      return entries
    },
    syncProviders(nextProviders: AgentProvider[], defaultProviderId: string | null | undefined) {
      providers = listEphemeralChatProviders(nextProviders)
      if (shouldKeepEphemeralChatProvider(providers, providerId)) {
        return
      }
      providerId = pickDefaultEphemeralChatProvider(providers, defaultProviderId)
    },
    async restore() {
      if (!input.getProjectId() || !providerId) {
        return
      }
      await restoreProjectConversation({
        projectId: input.getProjectId(),
        providerId,
        readConversationId: readProjectConversationId,
        storeConversationId: storeProjectConversationId,
        loadConversation: async (nextConversationId) => {
          await loadProjectConversation({
            conversationId: nextConversationId,
            mapEntries: mapPersistedEntries,
            setConversationId: (nextId) => {
              conversationId = nextId
            },
            setEntries: (nextEntries) => {
              entries = nextEntries as ProjectConversationTranscriptEntry[]
            },
            resetActiveAssistantEntry: () => {
              activeAssistantEntryId = ''
            },
            connectStream,
          })
        },
        clearConversation: () => {
          conversationId = ''
          entries = []
        },
      })
    },
    async selectProvider(nextProviderId: string) {
      if (!nextProviderId || providerId === nextProviderId) {
        return
      }
      abortController?.abort()
      providerId = nextProviderId
      conversationId = ''
      entries = []
      pending = false
      activeAssistantEntryId = ''
    },
    async sendTurn(message: string) {
      const trimmed = message.trim()
      const projectId = input.getProjectId()
      if (!trimmed || !projectId || !providerId || pending) {
        return
      }
      if (!conversationId) {
        const createPayload = await createProjectConversation({ providerId, projectId })
        conversationId = createPayload.conversation.id
        storeProjectConversationId(projectId, providerId, conversationId)
        await connectStream(conversationId)
      }
      appendTextEntry('user', trimmed)
      pending = true
      await startProjectConversationTurn(conversationId, trimmed)
    },
    async resetConversation() {
      abortController?.abort()
      await resetProjectConversationRuntime(conversationId)
      conversationId = ''
      entries = []
      activeAssistantEntryId = ''
      pending = false
    },
    async confirmActionProposal(entryId: string) {
      if (!conversationId) {
        return
      }
      await confirmProjectConversationActionProposal({
        conversationId,
        entryId,
        entries,
        setEntries: (nextEntries) => {
          entries = nextEntries
        },
        onError: input.onError,
      })
    },
    async respondInterrupt(inputValue: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) {
      if (!conversationId) {
        return
      }
      await respondProjectConversationInterrupt(conversationId, inputValue.interruptId, {
        decision: inputValue.decision,
        answer: inputValue.answer,
      })
    },
    dispose() {
      abortController?.abort()
      abortController = null
    },
  }
}
