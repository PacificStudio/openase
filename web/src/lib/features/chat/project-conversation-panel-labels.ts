import type { ProjectConversation } from '$lib/api/chat'
import type { ProjectConversationPhase } from './project-conversation-controller-helpers'
import type {
  ProjectConversationTextEntry,
  ProjectConversationTranscriptEntry,
} from './project-conversation-transcript-types'
import { chatT } from './i18n'

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

function findFirstUserMessage(entries: ProjectConversationTranscriptEntry[]) {
  return entries.find(
    (entry): entry is ProjectConversationTextEntry =>
      entry.kind === 'text' && entry.role === 'user' && entry.content.trim().length > 0,
  )
}

function conversationTitle(conversation?: ProjectConversation) {
  return (conversation?.title ?? '').trim()
}

function summaryText(conversation?: ProjectConversation) {
  return (conversation?.rollingSummary ?? '').trim()
}

function titleFromTranscript(entries: ProjectConversationTranscriptEntry[]) {
  return findFirstUserMessage(entries)?.content?.trim() ?? ''
}

export function getProjectConversationDisplayTitle(conversation?: ProjectConversation) {
  return conversationTitle(conversation)
}

export function getProjectConversationSummary(conversation?: ProjectConversation) {
  return summaryText(conversation)
}

export function getProjectConversationTitleFromTranscript(
  entries: ProjectConversationTranscriptEntry[],
) {
  return titleFromTranscript(entries)
}

function findTranscriptFallbackTitle(entries: ProjectConversationTranscriptEntry[]) {
  return titleFromTranscript(entries)
}

function conversationTimestampLabel(conversation?: ProjectConversation) {
  const timestamp = new Date(conversation?.lastActivityAt ?? '')
  if (Number.isNaN(timestamp.getTime())) {
    return chatT('chat.conversationTabLabel')
  }

  return `Conversation · ${timestamp.toLocaleString([], {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })}`
}

export function formatProjectConversationLabel(
  tab: Pick<ProjectConversationTabView, 'conversationId' | 'entries' | 'draft'>,
  conversations: ProjectConversation[],
) {
  const conversation = conversations.find((item) => item.id === tab.conversationId)
  const title = conversationTitle(conversation)
  if (title) {
    return truncateLabel(title)
  }

  const transcriptTitle = findTranscriptFallbackTitle(tab.entries)
  if (transcriptTitle) {
    return truncateLabel(transcriptTitle)
  }

  const draft = tab.draft.trim()
  if (draft) {
    return truncateLabel(draft)
  }

  if (!tab.conversationId) {
    return chatT('chat.newTab')
  }

  return conversationTimestampLabel(conversation)
}

export function formatProjectConversationTabStatus(
  tab: Pick<ProjectConversationTabView, 'pending' | 'hasPendingInterrupt' | 'restored'>,
) {
  if (tab.hasPendingInterrupt) {
    return chatT('chat.tabStatusInputRequired')
  }
  if (tab.pending) {
    return chatT('chat.tabStatusRunning')
  }
  if (tab.restored) {
    return chatT('chat.tabStatusRestored')
  }
  return ''
}

export function getProjectConversationStatusMessage(
  currentPhase: ProjectConversationPhase,
  hasPendingInterrupt: boolean,
): string | null {
  if (hasPendingInterrupt) {
    return chatT('chat.statusMessageInputRequired')
  }

  switch (currentPhase) {
    case 'restoring':
      return chatT('chat.statusMessageRestoring')
    case 'creating_conversation':
      return chatT('chat.statusMessageCreatingConversation')
    case 'connecting_stream':
      return chatT('chat.statusMessageConnectingStream')
    case 'submitting_turn':
      return chatT('chat.statusMessageSubmittingTurn')
    case 'awaiting_reply':
      return chatT('chat.statusMessageWaitingReply')
    case 'stopping_turn':
      return chatT('chat.statusMessageStoppingTurn')
    case 'resetting':
      return chatT('chat.statusMessageResetting')
    default:
      return null
  }
}
