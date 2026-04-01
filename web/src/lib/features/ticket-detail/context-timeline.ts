import type { HookExecution, TicketTimelineItem } from './types'
import type { TicketTimelineItemRecord } from '$lib/api/contracts'

export function inferHookStatus(eventType: string, message: string): HookExecution['status'] {
  if (eventType === 'hook.failed') return 'fail'
  if (eventType === 'hook.started') return 'running'
  if (message.toLowerCase().includes('timeout')) return 'timeout'
  return 'pass'
}

export function parseTimelineItem(raw: TicketTimelineItemRecord): TicketTimelineItem | null {
  if (!raw.id || !raw.ticket_id || !raw.created_at || !raw.updated_at) return null

  const shared = {
    id: raw.id,
    ticketId: raw.ticket_id,
    actor: {
      name: normalizeActorName(raw.actor_name ?? ''),
      type: raw.actor_type ?? 'unknown',
    },
    createdAt: raw.created_at,
    updatedAt: raw.updated_at,
    editedAt: raw.edited_at ?? undefined,
    isCollapsible: raw.is_collapsible ?? false,
    isDeleted: raw.is_deleted ?? false,
  }

  if (raw.item_type === 'description') {
    return {
      ...shared,
      kind: 'description',
      title: raw.title ?? '',
      bodyMarkdown: raw.body_markdown ?? '',
      identifier: stringMetadata(raw.metadata, 'identifier'),
    }
  }

  if (raw.item_type === 'comment') {
    const commentId = parseTimelineScopedId(raw.id, 'comment:')
    if (!commentId) return null

    return {
      ...shared,
      kind: 'comment',
      commentId,
      bodyMarkdown: raw.body_markdown ?? '',
      editCount: numericMetadata(raw.metadata, 'edit_count') ?? 0,
      revisionCount: numericMetadata(raw.metadata, 'revision_count') ?? 1,
      lastEditedBy: stringMetadata(raw.metadata, 'last_edited_by'),
    }
  }

  if (raw.item_type === 'activity') {
    return {
      ...shared,
      kind: 'activity',
      eventType: stringMetadata(raw.metadata, 'event_type') ?? raw.title ?? '',
      title: raw.title ?? '',
      bodyText: raw.body_text ?? '',
      metadata: cloneMetadata(raw.metadata),
    }
  }

  return null
}

function parseTimelineScopedId(id: string, prefix: string) {
  return id.startsWith(prefix) ? id.slice(prefix.length) : null
}

function stringMetadata(metadata: Record<string, unknown> | undefined, key: string) {
  const value = metadata?.[key]
  return typeof value === 'string' && value.trim() ? value : undefined
}

function numericMetadata(metadata: Record<string, unknown> | undefined, key: string) {
  const value = metadata?.[key]
  return typeof value === 'number' ? value : undefined
}

function cloneMetadata(metadata: Record<string, unknown> | undefined) {
  return metadata ? { ...metadata } : {}
}

function normalizeActorName(value: string) {
  const normalized = value.trim()
  if (!normalized) return 'Unknown'
  return normalized.includes(':') ? (normalized.split(':').at(-1) ?? normalized) : normalized
}
