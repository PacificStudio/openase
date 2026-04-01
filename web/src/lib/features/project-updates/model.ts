import type { ProjectUpdateCommentRecord, ProjectUpdateThreadRecord } from '$lib/api/contracts'
import type { ProjectUpdateComment, ProjectUpdateStatus, ProjectUpdateThread } from './types'

const validStatuses = new Set<ProjectUpdateStatus>(['on_track', 'at_risk', 'off_track'])

export function parseProjectUpdateThreads(
  rawThreads: ProjectUpdateThreadRecord[],
): ProjectUpdateThread[] {
  return rawThreads
    .map(parseProjectUpdateThread)
    .filter((thread): thread is ProjectUpdateThread => thread !== null)
    .sort((left, right) => right.lastActivityAt.localeCompare(left.lastActivityAt))
}

function parseProjectUpdateThread(raw: ProjectUpdateThreadRecord): ProjectUpdateThread | null {
  const status = parseProjectUpdateStatus(raw.status)
  if (!status || !raw.id || !raw.project_id || !raw.title || !raw.created_at || !raw.updated_at) {
    return null
  }

  return {
    id: raw.id,
    projectId: raw.project_id,
    status,
    title: raw.title,
    bodyMarkdown: raw.body_markdown ?? '',
    createdBy: raw.created_by ?? 'system',
    createdAt: raw.created_at,
    updatedAt: raw.updated_at,
    editedAt: raw.edited_at ?? undefined,
    editCount: raw.edit_count ?? 0,
    lastEditedBy: raw.last_edited_by ?? undefined,
    isDeleted: Boolean(raw.is_deleted),
    deletedAt: raw.deleted_at ?? undefined,
    deletedBy: raw.deleted_by ?? undefined,
    lastActivityAt: raw.last_activity_at ?? raw.updated_at,
    commentCount: raw.comment_count ?? 0,
    comments: parseProjectUpdateComments(raw.comments ?? []),
  }
}

function parseProjectUpdateComments(
  rawComments: ProjectUpdateCommentRecord[],
): ProjectUpdateComment[] {
  const comments: ProjectUpdateComment[] = []

  for (const raw of rawComments) {
    if (!raw.id || !raw.thread_id || !raw.created_at || !raw.updated_at) {
      continue
    }

    comments.push({
      id: raw.id,
      threadId: raw.thread_id,
      bodyMarkdown: raw.body_markdown ?? '',
      createdBy: raw.created_by ?? 'system',
      createdAt: raw.created_at,
      updatedAt: raw.updated_at,
      editedAt: raw.edited_at ?? undefined,
      editCount: raw.edit_count ?? 0,
      lastEditedBy: raw.last_edited_by ?? undefined,
      isDeleted: Boolean(raw.is_deleted),
      deletedAt: raw.deleted_at ?? undefined,
      deletedBy: raw.deleted_by ?? undefined,
    })
  }

  return comments
}

function parseProjectUpdateStatus(raw: string | null | undefined): ProjectUpdateStatus | null {
  return raw && validStatuses.has(raw as ProjectUpdateStatus) ? (raw as ProjectUpdateStatus) : null
}
