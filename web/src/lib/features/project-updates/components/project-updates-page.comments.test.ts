import { fireEvent, render, waitFor } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'

import {
  connectEventStream,
  createProjectUpdateComment,
  deleteProjectUpdateComment,
  listProjectUpdates,
  makeCommentRecord,
  makeThreadRecord,
  setupProjectUpdatesPageTest,
  type StreamEventHandler,
  updateProjectUpdateComment,
} from './project-updates-page.test-support'
import { ProjectUpdatesPage } from '..'

describe('ProjectUpdatesPage comments and streaming', () => {
  setupProjectUpdatesPageTest()

  it('creates, edits, deletes comments and refreshes from the activity stream', async () => {
    listProjectUpdates
      .mockResolvedValueOnce({ threads: [makeThreadRecord()] })
      .mockResolvedValueOnce({
        threads: [
          makeThreadRecord({
            last_activity_at: '2026-04-01T10:32:00Z',
            comment_count: 1,
            comments: [makeCommentRecord()],
          }),
        ],
      })
      .mockResolvedValueOnce({
        threads: [
          makeThreadRecord({
            last_activity_at: '2026-04-01T10:33:00Z',
            comment_count: 1,
            comments: [
              makeCommentRecord({
                body_markdown: 'Need one more canary before noon.',
                updated_at: '2026-04-01T10:33:00Z',
                edited_at: '2026-04-01T10:33:00Z',
                edit_count: 1,
              }),
            ],
          }),
        ],
      })
      .mockResolvedValueOnce({
        threads: [
          makeThreadRecord({
            last_activity_at: '2026-04-01T10:34:00Z',
            comment_count: 1,
            comments: [
              makeCommentRecord({
                body_markdown: 'Need one more canary before noon.',
                updated_at: '2026-04-01T10:34:00Z',
                edited_at: '2026-04-01T10:34:00Z',
                edit_count: 2,
                is_deleted: true,
                deleted_at: '2026-04-01T10:34:00Z',
                deleted_by: 'user:ops',
              }),
            ],
          }),
        ],
      })
      .mockResolvedValueOnce({
        threads: [
          makeThreadRecord({
            status: 'off_track',
            body_markdown: 'A new project update event arrived.',
            updated_at: '2026-04-01T10:35:00Z',
            edited_at: '2026-04-01T10:35:00Z',
            edit_count: 1,
            last_activity_at: '2026-04-01T10:35:00Z',
            comment_count: 1,
            comments: [
              makeCommentRecord({
                body_markdown: 'Need one more canary before noon.',
                updated_at: '2026-04-01T10:34:00Z',
                edited_at: '2026-04-01T10:34:00Z',
                edit_count: 2,
                is_deleted: true,
                deleted_at: '2026-04-01T10:34:00Z',
                deleted_by: 'user:ops',
              }),
            ],
          }),
        ],
      })

    createProjectUpdateComment.mockResolvedValue({ comment: { id: 'comment-1' } })
    updateProjectUpdateComment.mockResolvedValue({ comment: { id: 'comment-1' } })
    deleteProjectUpdateComment.mockResolvedValue({ deleted_comment_id: 'comment-1' })

    const { findByText, findByRole, getByLabelText, getByText } = render(ProjectUpdatesPage)

    await fireEvent.input(await findByRole('textbox', { name: 'Reply to Sprint 2 rollout' }), {
      target: { value: 'Need one more canary.' },
    })
    await fireEvent.click(await findByRole('button', { name: 'Add comment' }))

    expect(createProjectUpdateComment).toHaveBeenCalledWith('project-1', 'thread-1', {
      body: 'Need one more canary.',
    })
    expect(await findByText('Comment added.')).toBeTruthy()
    expect(await findByText('Need one more canary.')).toBeTruthy()

    await fireEvent.click(getByLabelText('Edit comment comment-1'))
    await fireEvent.input(await findByRole('textbox', { name: 'Edit comment comment-1' }), {
      target: { value: 'Need one more canary before noon.' },
    })
    await fireEvent.click(getByText('Save'))

    expect(updateProjectUpdateComment).toHaveBeenCalledWith('project-1', 'thread-1', 'comment-1', {
      body: 'Need one more canary before noon.',
    })
    expect(await findByText('Comment edited.')).toBeTruthy()
    expect(await findByText('Need one more canary before noon.')).toBeTruthy()

    await fireEvent.click(getByLabelText('Delete comment comment-1'))

    expect(deleteProjectUpdateComment).toHaveBeenCalledWith('project-1', 'thread-1', 'comment-1')
    expect(await findByText('Comment deleted.')).toBeTruthy()
    expect(await findByText('This comment was deleted.')).toBeTruthy()

    const streamHandlers = connectEventStream.mock.calls[0]?.[1] as
      | { onEvent?: StreamEventHandler }
      | undefined
    streamHandlers?.onEvent?.({
      event: 'message',
      data: JSON.stringify({ type: 'project_update_thread.status_changed' }),
    })

    await waitFor(() => {
      expect(listProjectUpdates).toHaveBeenCalledTimes(5)
    })
    expect(await findByText('A new project update event arrived.')).toBeTruthy()
    expect(await findByText('Off track', { selector: 'span' })).toBeTruthy()
  })
})
