import { formatRelativeTime } from '$lib/utils'
import { projectUpdatesT } from './i18n'

export function isProjectUpdateEdited(createdAt: string, updatedAt: string, editedAt?: string) {
  return Boolean(editedAt) || updatedAt !== createdAt
}

export function projectUpdateEditedLabel(createdAt: string, updatedAt: string, editedAt?: string) {
  const effective = editedAt ?? updatedAt
  return isProjectUpdateEdited(createdAt, updatedAt, editedAt)
    ? projectUpdatesT('projectUpdates.editedLabel', { time: formatRelativeTime(effective) })
    : ''
}
