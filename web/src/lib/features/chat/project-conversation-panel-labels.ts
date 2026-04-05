import type { ProjectConversation } from '$lib/api/chat'
import type { ProjectConversationPhase } from './project-conversation-controller-helpers'
import type {
  ProjectConversationTextEntry,
  ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-types'

export type ProjectConversationTabView = {
  id: string
  projectId: string
  projectName: string
  conversationId: string
  entries: ProjectConversationTranscriptEntry[]
  pending: boolean
  hasPendingInterrupt: boolean
  restored: boolean
  draft: string
}

const MAX_LABEL_LENGTH = 32

function truncateLabel(value: string) {
  return value.length > MAX_LABEL_LENGTH ? `${value.slice(0, MAX_LABEL_LENGTH)}…` : value
}

function findRecentUserMessage(entries: ProjectConversationTranscriptEntry[]) {
  return [...entries]
    .reverse()
    .find(
      (entry): entry is ProjectConversationTextEntry =>
        entry.kind === 'text' && entry.role === 'user' && entry.content.trim().length > 0,
    )
}

export function formatProjectConversationLabel(
  tab: Pick<ProjectConversationTabView, 'conversationId' | 'entries' | 'draft'>,
  conversations: ProjectConversation[],
) {
  const conversation = conversations.find((item) => item.id === tab.conversationId)
  const summary = (conversation?.rollingSummary ?? '').trim()
  if (summary) {
    return truncateLabel(summary)
  }

  const recentUserMessage = findRecentUserMessage(tab.entries)
  if (recentUserMessage?.content) {
    return truncateLabel(recentUserMessage.content.trim())
  }

  const draft = tab.draft.trim()
  if (draft) {
    return truncateLabel(draft)
  }

  if (!tab.conversationId) {
    return 'New tab'
  }

  const timestamp = new Date(conversation?.lastActivityAt ?? '')
  if (Number.isNaN(timestamp.getTime())) {
    return 'Conversation'
  }

  return `Conversation · ${timestamp.toLocaleString([], {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })}`
}

export function formatProjectConversationTabStatus(
  tab: Pick<ProjectConversationTabView, 'pending' | 'hasPendingInterrupt' | 'restored'>,
) {
  if (tab.hasPendingInterrupt) {
    return 'Input required'
  }
  if (tab.pending) {
    return 'Running'
  }
  if (tab.restored) {
    return 'Restored'
  }
  return ''
}

export function getProjectConversationStatusMessage(
  currentPhase: ProjectConversationPhase,
  hasPendingInterrupt: boolean,
): string | null {
  if (hasPendingInterrupt) {
    return 'Additional input is required before the conversation can continue.'
  }

  switch (currentPhase) {
    case 'restoring':
      return 'Restoring this project conversation…'
    case 'creating_conversation':
      return 'Creating a fresh project conversation…'
    case 'connecting_stream':
      return 'Connecting the live conversation stream…'
    case 'submitting_turn':
      return 'Sending your message…'
    case 'awaiting_reply':
      return 'Waiting for the assistant reply…'
    case 'resetting':
      return 'Resetting the current conversation…'
    default:
      return null
  }
}
