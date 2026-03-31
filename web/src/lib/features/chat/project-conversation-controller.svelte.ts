/* eslint-disable max-lines */
import { ApiError } from '$lib/api/client'
import {
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
  type ChatActionProposalPayload,
  type ChatDiffPayload,
  type ChatMessagePayload,
  type ProjectConversationEntry,
  type ProjectConversationStreamEvent,
} from '$lib/api/chat'
import type { AgentProvider } from '$lib/api/contracts'
import type { ChatActionExecutionResult } from './action-proposal-executor'
import {
  listEphemeralChatProviders,
  pickDefaultEphemeralChatProvider,
  shouldKeepEphemeralChatProvider,
} from './provider-options'

type ProjectConversationRole = 'user' | 'assistant' | 'system'

type ProjectConversationTextEntry = {
  id: string
  kind: 'text'
  role: ProjectConversationRole
  turnId?: string
  content: string
  streaming: boolean
}

type ProjectConversationActionProposalEntry = {
  id: string
  kind: 'action_proposal'
  role: 'assistant'
  proposal: ChatActionProposalPayload
  status: 'pending' | 'executing' | 'confirmed'
  results: ChatActionExecutionResult[]
}

type ProjectConversationDiffEntry = {
  id: string
  kind: 'diff'
  role: 'assistant'
  diff: ChatDiffPayload
}

type ProjectConversationInterruptEntry = {
  id: string
  kind: 'interrupt'
  role: 'system'
  interruptId: string
  provider: string
  interruptKind: string
  payload: Record<string, unknown>
  options: { id: string; label: string }[]
  status: 'pending' | 'resolved'
  decision?: string
}

export type ProjectConversationTranscriptEntry =
  | ProjectConversationTextEntry
  | ProjectConversationActionProposalEntry
  | ProjectConversationDiffEntry
  | ProjectConversationInterruptEntry

type CreateProjectConversationControllerInput = {
  getProjectId: () => string
  onError?: (message: string) => void
}

const projectConversationStoragePrefix = 'openase.project-conversation'

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

  function storageKey(projectId: string, currentProviderId: string) {
    return `${projectConversationStoragePrefix}.${projectId}.${currentProviderId}`
  }

  function appendTextEntry(
    role: ProjectConversationRole,
    content: string,
    options?: { turnId?: string; streaming?: boolean; entryId?: string },
  ) {
    entryCounter += 1
    entries = [
      ...entries,
      {
        id: options?.entryId ?? `entry-${entryCounter}`,
        kind: 'text',
        role,
        turnId: options?.turnId,
        content,
        streaming: options?.streaming ?? false,
      },
    ]
  }

  function finalizeAssistantEntry() {
    if (!activeAssistantEntryId) {
      return
    }
    entries = entries.map((entry) =>
      entry.kind === 'text' && entry.id === activeAssistantEntryId
        ? { ...entry, streaming: false }
        : entry,
    )
    activeAssistantEntryId = ''
  }

  function appendAssistantChunk(content: string, turnId?: string) {
    if (!activeAssistantEntryId) {
      entryCounter += 1
      activeAssistantEntryId = `entry-${entryCounter}`
      entries = [
        ...entries,
        {
          id: activeAssistantEntryId,
          kind: 'text',
          role: 'assistant',
          turnId,
          content,
          streaming: true,
        },
      ]
      return
    }

    entries = entries.map((entry) =>
      entry.kind === 'text' && entry.id === activeAssistantEntryId
        ? { ...entry, content: `${entry.content}${content}`, streaming: true }
        : entry,
    )
  }

  function storeConversationId(
    projectId: string,
    currentProviderId: string,
    nextConversationId: string,
  ) {
    if (typeof window === 'undefined') {
      return
    }
    try {
      window.localStorage.setItem(storageKey(projectId, currentProviderId), nextConversationId)
    } catch {
      // Ignore localStorage failures.
    }
  }

  function readStoredConversationId(projectId: string, currentProviderId: string) {
    if (typeof window === 'undefined') {
      return ''
    }
    try {
      return window.localStorage.getItem(storageKey(projectId, currentProviderId))?.trim() ?? ''
    } catch {
      return ''
    }
  }

  async function connectStream(nextConversationId: string) {
    abortController?.abort()
    const controller = new AbortController()
    abortController = controller
    const activeRequestId = ++requestId

    try {
      await watchProjectConversation(nextConversationId, {
        signal: controller.signal,
        onEvent: (event) => {
          if (activeRequestId !== requestId) {
            return
          }
          handleStreamEvent(event)
        },
      })
    } catch (caughtError) {
      if (activeRequestId !== requestId || isAbortError(caughtError)) {
        return
      }
      input.onError?.(
        caughtError instanceof ApiError
          ? caughtError.detail
          : 'Project conversation stream disconnected.',
      )
    }
  }

  async function loadConversation(nextConversationId: string) {
    const payload = await listProjectConversationEntries(nextConversationId)
    conversationId = nextConversationId
    entries = mapPersistedEntries(payload.entries)
    activeAssistantEntryId = ''
    await connectStream(nextConversationId)
  }

  async function restoreConversation(projectId: string, currentProviderId: string) {
    const storedConversationId = readStoredConversationId(projectId, currentProviderId)

    if (storedConversationId) {
      try {
        const conversation = await getProjectConversation(storedConversationId)
        if (conversation.conversation.providerId === currentProviderId) {
          await loadConversation(storedConversationId)
          return
        }
      } catch {
        // Fall through to list lookup.
      }
    }

    const listPayload = await listProjectConversations({
      projectId,
      providerId: currentProviderId,
    })
    const current = listPayload.conversations[0]
    if (!current) {
      conversationId = ''
      entries = []
      return
    }

    storeConversationId(projectId, currentProviderId, current.id)
    await loadConversation(current.id)
  }

  function handleStreamEvent(event: ProjectConversationStreamEvent) {
    if (event.kind === 'session') {
      conversationId = event.payload.conversationId
      storeConversationId(input.getProjectId(), providerId, conversationId)
      return
    }

    if (event.kind === 'message') {
      const payload = event.payload
      if (isTextPayload(payload)) {
        appendAssistantChunk(payload.content)
        return
      }

      finalizeAssistantEntry()
      if (isActionProposalPayload(payload)) {
        entries = [
          ...entries,
          {
            id: payload.entryId ?? `entry-${++entryCounter}`,
            kind: 'action_proposal',
            role: 'assistant',
            proposal: payload,
            status: 'pending',
            results: [],
          },
        ]
        return
      }

      if (isDiffPayload(payload)) {
        entries = [
          ...entries,
          {
            id: payload.entryId ?? `entry-${++entryCounter}`,
            kind: 'diff',
            role: 'assistant',
            diff: payload,
          },
        ]
        return
      }

      if (payload.type === 'action_result') {
        const resultPayload = (
          payload.raw as { payload?: { entry_id?: string; results?: unknown[] } }
        )?.payload
        const resultEntryId = resultPayload?.entry_id
        const results = (resultPayload?.results ?? []) as ChatActionExecutionResult[]
        if (resultEntryId) {
          entries = entries.map((entry) =>
            entry.kind === 'action_proposal' && entry.id === resultEntryId
              ? { ...entry, status: 'confirmed', results }
              : entry,
          )
        }
        return
      }

      return
    }

    if (event.kind === 'interrupt_requested') {
      finalizeAssistantEntry()
      pending = false
      entryCounter += 1
      entries = [
        ...entries,
        {
          id: `entry-${entryCounter}`,
          kind: 'interrupt',
          role: 'system',
          interruptId: event.payload.interruptId,
          provider: event.payload.provider,
          interruptKind: event.payload.kind,
          payload: event.payload.payload,
          options: event.payload.options,
          status: 'pending',
        },
      ]
      return
    }

    if (event.kind === 'interrupt_resolved') {
      entries = entries.map((entry) =>
        entry.kind === 'interrupt' && entry.interruptId === event.payload.interruptId
          ? { ...entry, status: 'resolved', decision: event.payload.decision }
          : entry,
      )
      pending = true
      return
    }

    if (event.kind === 'turn_done') {
      finalizeAssistantEntry()
      pending = false
      return
    }

    finalizeAssistantEntry()
    pending = false
    input.onError?.(event.payload.message)
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
      await restoreConversation(input.getProjectId(), providerId)
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
        storeConversationId(projectId, providerId, conversationId)
        await connectStream(conversationId)
      }

      appendTextEntry('user', trimmed)
      pending = true
      await startProjectConversationTurn(conversationId, trimmed)
    },
    async resetConversation() {
      abortController?.abort()
      if (conversationId) {
        await closeProjectConversationRuntime(conversationId).catch(() => {})
      }
      conversationId = ''
      entries = []
      activeAssistantEntryId = ''
      pending = false
    },
    async confirmActionProposal(entryId: string) {
      if (!conversationId) {
        return
      }
      entries = entries.map((entry) =>
        entry.kind === 'action_proposal' && entry.id === entryId
          ? { ...entry, status: 'executing' }
          : entry,
      )
      try {
        const payload = await executeProjectConversationActionProposal(conversationId, entryId)
        const results = payload.results as ChatActionExecutionResult[]
        entries = entries.map((entry) =>
          entry.kind === 'action_proposal' && entry.id === entryId
            ? { ...entry, status: 'confirmed', results }
            : entry,
        )
      } catch (caughtError) {
        input.onError?.(
          caughtError instanceof ApiError
            ? caughtError.detail
            : 'Failed to execute project conversation action proposal.',
        )
        entries = entries.map((entry) =>
          entry.kind === 'action_proposal' && entry.id === entryId
            ? { ...entry, status: 'pending' }
            : entry,
        )
      }
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

function mapPersistedEntries(
  entries: ProjectConversationEntry[],
): ProjectConversationTranscriptEntry[] {
  const transcript: ProjectConversationTranscriptEntry[] = []

  for (const entry of entries) {
    if (entry.kind === 'user_message') {
      transcript.push({
        id: entry.id,
        kind: 'text',
        role: 'user',
        turnId: entry.turnId,
        content: String(entry.payload.content ?? ''),
        streaming: false,
      })
      continue
    }

    if (entry.kind === 'assistant_text_delta') {
      const last = transcript.at(-1)
      if (
        last &&
        last.kind === 'text' &&
        last.role === 'assistant' &&
        last.turnId === entry.turnId
      ) {
        last.content = `${last.content}${String(entry.payload.content ?? '')}`
      } else {
        transcript.push({
          id: entry.id,
          kind: 'text',
          role: 'assistant',
          turnId: entry.turnId,
          content: String(entry.payload.content ?? ''),
          streaming: false,
        })
      }
      continue
    }

    if (entry.kind === 'action_proposal') {
      transcript.push({
        id: entry.id,
        kind: 'action_proposal',
        role: 'assistant',
        proposal: {
          ...(entry.payload as unknown as ChatActionProposalPayload),
          type: 'action_proposal',
          entryId: entry.id,
        },
        status: 'pending',
        results: [],
      })
      continue
    }

    if (entry.kind === 'diff') {
      transcript.push({
        id: entry.id,
        kind: 'diff',
        role: 'assistant',
        diff: {
          ...(entry.payload as unknown as ChatDiffPayload),
          type: 'diff',
          entryId: entry.id,
        },
      })
      continue
    }

    if (entry.kind === 'interrupt') {
      transcript.push({
        id: entry.id,
        kind: 'interrupt',
        role: 'system',
        interruptId: String(entry.payload.interrupt_id ?? ''),
        provider: String(entry.payload.provider ?? 'codex'),
        interruptKind: String(entry.payload.kind ?? ''),
        payload: (entry.payload.payload as Record<string, unknown>) ?? {},
        options: ((entry.payload.options as { id: string; label: string }[]) ?? []).map(
          (option) => ({
            id: option.id,
            label: option.label,
          }),
        ),
        status: 'pending',
      })
      continue
    }

    if (entry.kind === 'interrupt_resolution') {
      const interruptId = String(entry.payload.interrupt_id ?? '')
      const decision = entry.payload.decision ? String(entry.payload.decision) : undefined
      for (const transcriptEntry of transcript) {
        if (transcriptEntry.kind === 'interrupt' && transcriptEntry.interruptId === interruptId) {
          transcriptEntry.status = 'resolved'
          transcriptEntry.decision = decision
        }
      }
      continue
    }

    if (entry.kind === 'action_result') {
      const payload = entry.payload as {
        entry_id?: string
        results?: ChatActionExecutionResult[]
      }
      if (payload.entry_id) {
        for (const transcriptEntry of transcript) {
          if (
            transcriptEntry.kind === 'action_proposal' &&
            transcriptEntry.id === payload.entry_id
          ) {
            transcriptEntry.status = 'confirmed'
            transcriptEntry.results = payload.results ?? []
          }
        }
      }
    }
  }

  return transcript
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === 'AbortError'
}

function isTextPayload(
  payload: ChatMessagePayload,
): payload is Extract<ChatMessagePayload, { type: 'text' }> {
  return payload.type === 'text'
}

function isActionProposalPayload(
  payload: ChatMessagePayload,
): payload is ChatActionProposalPayload {
  return payload.type === 'action_proposal'
}

function isDiffPayload(payload: ChatMessagePayload): payload is ChatDiffPayload {
  return payload.type === 'diff'
}
