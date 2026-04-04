import { formatRelativeTime } from '$lib/utils'
import type { TicketTimelineItem } from '../types'

export function isEditedTimelineItem(item: TicketTimelineItem) {
  return Boolean(item.editedAt) || item.updatedAt !== item.createdAt
}

export function getEditedTimelineLabel(item: TicketTimelineItem) {
  const editedAt = item.editedAt ?? item.updatedAt
  return editedAt && isEditedTimelineItem(item) ? `edited ${formatRelativeTime(editedAt)}` : null
}
