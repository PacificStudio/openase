import type { MockState } from './constants'
import { nowIso } from './constants'
import { asNumber, asString } from './helpers'

export function createProjectUpdateThreadRecord(
  state: MockState,
  projectId: string,
  input: Record<string, unknown>,
) {
  const createdAt = nowIso
  const body = asString(input.body)?.trim() ?? ''
  const status = parseProjectUpdateStatus(asString(input.status))
  const title = asString(input.title)?.trim() || summarizeProjectUpdateTitle(body)

  return {
    id: `update-thread-${++state.counters.projectUpdateThread}`,
    project_id: projectId,
    status,
    title,
    body_markdown: body,
    created_by: asString(input.created_by) ?? 'playwright',
    created_at: createdAt,
    updated_at: createdAt,
    edited_at: null,
    edit_count: 0,
    last_edited_by: null,
    is_deleted: false,
    deleted_at: null,
    deleted_by: null,
    last_activity_at: createdAt,
    comment_count: 0,
    comments: [],
  }
}

export function updateProjectUpdateThreadRecord(
  thread: Record<string, unknown>,
  input: Record<string, unknown>,
) {
  const updatedAt = nowIso
  thread.status = parseProjectUpdateStatus(asString(input.status))
  thread.title =
    asString(input.title)?.trim() || summarizeProjectUpdateTitle(asString(input.body) ?? '')
  thread.body_markdown = asString(input.body)?.trim() ?? ''
  thread.updated_at = updatedAt
  thread.edited_at = updatedAt
  thread.edit_count = (asNumber(thread.edit_count) ?? 0) + 1
  thread.last_edited_by = asString(input.edited_by) ?? 'playwright'
  thread.last_activity_at = updatedAt
}

export function deleteProjectUpdateThreadRecord(thread: Record<string, unknown>) {
  const deletedAt = nowIso
  thread.is_deleted = true
  thread.deleted_at = deletedAt
  thread.deleted_by = 'playwright'
  thread.updated_at = deletedAt
  thread.last_activity_at = deletedAt
}

export function createProjectUpdateCommentRecord(
  state: MockState,
  threadId: string,
  input: Record<string, unknown>,
) {
  const createdAt = nowIso
  return {
    id: `update-comment-${++state.counters.projectUpdateComment}`,
    thread_id: threadId,
    body_markdown: asString(input.body)?.trim() ?? '',
    created_by: asString(input.created_by) ?? 'playwright',
    created_at: createdAt,
    updated_at: createdAt,
    edited_at: null,
    edit_count: 0,
    last_edited_by: null,
    is_deleted: false,
    deleted_at: null,
    deleted_by: null,
  }
}

export function updateProjectUpdateCommentRecord(
  comment: Record<string, unknown>,
  input: Record<string, unknown>,
) {
  const updatedAt = nowIso
  comment.body_markdown = asString(input.body)?.trim() ?? ''
  comment.updated_at = updatedAt
  comment.edited_at = updatedAt
  comment.edit_count = (asNumber(comment.edit_count) ?? 0) + 1
  comment.last_edited_by = asString(input.edited_by) ?? 'playwright'
}

export function deleteProjectUpdateCommentRecord(comment: Record<string, unknown>) {
  const deletedAt = nowIso
  comment.is_deleted = true
  comment.deleted_at = deletedAt
  comment.deleted_by = 'playwright'
  comment.updated_at = deletedAt
}

export function readProjectUpdateComments(thread: Record<string, unknown>) {
  return Array.isArray(thread.comments) ? [...thread.comments] : []
}

export function summarizeProjectUpdateTitle(body: string) {
  const firstLine = body.split('\n')[0]?.trim() ?? ''
  if (firstLine.length > 0) {
    return firstLine.slice(0, 72)
  }
  return 'Update'
}

export function parseProjectUpdateStatus(raw: string | null | undefined) {
  return raw === 'at_risk' || raw === 'off_track' ? raw : 'on_track'
}
