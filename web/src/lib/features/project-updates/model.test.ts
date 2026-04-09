import { describe, expect, it } from 'vitest'

import type { ProjectUpdatePayload } from '$lib/api/contracts'
import {
  mergeProjectUpdateThreads,
  parseProjectUpdatePage,
  parseProjectUpdateThreads,
} from './model'

describe('project update model', () => {
  it('sorts threads by last activity descending while preserving deleted placeholders', () => {
    const payload: ProjectUpdatePayload = {
      has_more: false,
      next_cursor: '',
      threads: [
        {
          id: 'thread-older',
          project_id: 'project-1',
          status: 'off_track',
          title: 'Older',
          body_markdown: 'Older body',
          created_by: 'user:alice',
          created_at: '2026-04-01T08:00:00Z',
          updated_at: '2026-04-01T08:30:00Z',
          edited_at: null,
          edit_count: 0,
          last_edited_by: null,
          is_deleted: true,
          deleted_at: '2026-04-01T08:45:00Z',
          deleted_by: 'user:alice',
          last_activity_at: '2026-04-01T09:00:00Z',
          comment_count: 0,
          comments: [],
        },
        {
          id: 'thread-newer',
          project_id: 'project-1',
          status: 'on_track',
          title: 'Newer',
          body_markdown: 'Newer body',
          created_by: 'user:bob',
          created_at: '2026-04-01T10:00:00Z',
          updated_at: '2026-04-01T10:05:00Z',
          edited_at: null,
          edit_count: 0,
          last_edited_by: null,
          is_deleted: false,
          deleted_at: null,
          deleted_by: null,
          last_activity_at: '2026-04-01T10:06:00Z',
          comment_count: 1,
          comments: [
            {
              id: 'comment-1',
              thread_id: 'thread-newer',
              body_markdown: 'LGTM',
              created_by: 'user:carol',
              created_at: '2026-04-01T10:06:00Z',
              updated_at: '2026-04-01T10:06:00Z',
              edited_at: null,
              edit_count: 0,
              last_edited_by: null,
              is_deleted: false,
              deleted_at: null,
              deleted_by: null,
            },
          ],
        },
      ],
    }

    const threads = parseProjectUpdateThreads(payload.threads)

    expect(threads.map((thread) => thread.id)).toEqual(['thread-newer', 'thread-older'])
    expect(threads[0]?.comments[0]?.bodyMarkdown).toBe('LGTM')
    expect(threads[1]?.isDeleted).toBe(true)
  })

  it('parses pagination metadata and deduplicates merged threads with the preferred copy', () => {
    const firstPage = parseProjectUpdatePage({
      threads: [
        {
          id: 'thread-b',
          project_id: 'project-1',
          status: 'on_track',
          title: 'Thread B',
          body_markdown: 'B',
          created_by: 'user:bob',
          created_at: '2026-04-01T10:00:00Z',
          updated_at: '2026-04-01T10:00:00Z',
          edited_at: null,
          edit_count: 0,
          last_edited_by: null,
          is_deleted: false,
          deleted_at: null,
          deleted_by: null,
          last_activity_at: '2026-04-01T10:00:00Z',
          comment_count: 0,
          comments: [],
        },
      ],
      has_more: true,
      next_cursor: 'cursor-1',
    })

    const merged = mergeProjectUpdateThreads(
      [
        {
          ...firstPage.threads[0],
          lastActivityAt: '2026-04-01T10:02:00Z',
          bodyMarkdown: 'fresh',
        },
      ],
      [
        {
          ...firstPage.threads[0],
          lastActivityAt: '2026-04-01T09:58:00Z',
          bodyMarkdown: 'stale',
        },
        {
          ...firstPage.threads[0],
          id: 'thread-a',
          title: 'Thread A',
          bodyMarkdown: 'older',
          lastActivityAt: '2026-04-01T09:57:00Z',
        },
      ],
    )

    expect(firstPage.hasMore).toBe(true)
    expect(firstPage.nextCursor).toBe('cursor-1')
    expect(merged).toHaveLength(2)
    expect(merged[0]?.bodyMarkdown).toBe('fresh')
    expect(merged.map((thread) => thread.id)).toEqual(['thread-b', 'thread-a'])
  })
})
