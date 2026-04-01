import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { Project, ProjectUpdatePayload } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import { ProjectUpdatesPage } from '..'

const {
  connectEventStream,
  createProjectUpdateComment,
  createProjectUpdateThread,
  deleteProjectUpdateComment,
  deleteProjectUpdateThread,
  listProjectUpdates,
  updateProjectUpdateThread,
  updateProjectUpdateComment,
} = vi.hoisted(() => ({
  connectEventStream: vi.fn(),
  createProjectUpdateComment: vi.fn(),
  createProjectUpdateThread: vi.fn(),
  deleteProjectUpdateComment: vi.fn(),
  deleteProjectUpdateThread: vi.fn(),
  listProjectUpdates: vi.fn(),
  updateProjectUpdateThread: vi.fn(),
  updateProjectUpdateComment: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createProjectUpdateComment,
  createProjectUpdateThread,
  deleteProjectUpdateComment,
  deleteProjectUpdateThread,
  listProjectUpdates,
  updateProjectUpdateThread,
  updateProjectUpdateComment,
}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'OpenASE',
  slug: 'openase',
  description: 'Autonomous software engineering',
  status: 'active',
  default_agent_provider_id: null,
  accessible_machine_ids: [],
  max_concurrent_agents: 4,
}

type StreamEventHandler = (frame: { event: string; data: string }) => void

const orderedUpdatesFixture: ProjectUpdatePayload = {
  threads: [
    {
      id: 'thread-late',
      project_id: 'project-1',
      status: 'at_risk',
      title: 'Migration watch',
      body_markdown: 'Database cleanup is running late.',
      created_by: 'user:ops',
      created_at: '2026-04-01T09:30:00Z',
      updated_at: '2026-04-01T09:35:00Z',
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
    {
      id: 'thread-early',
      project_id: 'project-1',
      status: 'on_track',
      title: 'Sprint 2 rollout',
      body_markdown: 'Launch is green.',
      created_by: 'user:codex',
      created_at: '2026-04-01T08:00:00Z',
      updated_at: '2026-04-01T08:15:00Z',
      edited_at: null,
      edit_count: 0,
      last_edited_by: null,
      is_deleted: false,
      deleted_at: null,
      deleted_by: null,
      last_activity_at: '2026-04-01T08:15:00Z',
      comment_count: 0,
      comments: [],
    },
  ],
}

describe('ProjectUpdatesPage', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    vi.stubGlobal(
      'confirm',
      vi.fn(() => true),
    )
    appStore.currentProject = projectFixture
    connectEventStream.mockReturnValue(() => {})
  })

  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.unstubAllGlobals()
  })

  it('renders status badges in last-activity order and posts a new update', async () => {
    listProjectUpdates
      .mockResolvedValueOnce({
        threads: [orderedUpdatesFixture.threads[1], orderedUpdatesFixture.threads[0]],
      })
      .mockResolvedValueOnce({
        threads: [
          {
            id: 'thread-new',
            project_id: 'project-1',
            status: 'off_track',
            title: 'Hotfix hold',
            body_markdown: 'Release paused pending rollback validation.',
            created_by: 'user:ops',
            created_at: '2026-04-01T10:31:00Z',
            updated_at: '2026-04-01T10:31:00Z',
            edited_at: null,
            edit_count: 0,
            last_edited_by: null,
            is_deleted: false,
            deleted_at: null,
            deleted_by: null,
            last_activity_at: '2026-04-01T10:31:00Z',
            comment_count: 0,
            comments: [],
          },
          ...orderedUpdatesFixture.threads,
        ],
      })
    createProjectUpdateThread.mockResolvedValue({ thread: { id: 'thread-new' } })

    const { findByText, getByPlaceholderText, getAllByRole, getByRole } = render(ProjectUpdatesPage)

    expect(await findByText('Migration watch')).toBeTruthy()
    expect(await findByText('At risk', { selector: 'span' })).toBeTruthy()
    expect(await findByText('On track', { selector: 'span' })).toBeTruthy()
    expect(
      getAllByRole('heading', { level: 2 })
        .map((node) => node.textContent)
        .filter((text) => text !== 'Post a project update'),
    ).toEqual(['Migration watch', 'Sprint 2 rollout'])

    await fireEvent.change(getByRole('combobox', { name: 'New update status' }), {
      target: { value: 'off_track' },
    })
    await fireEvent.input(getByPlaceholderText('Sprint 2 rollout'), {
      target: { value: 'Hotfix hold' },
    })
    await fireEvent.input(
      getByPlaceholderText('Summarize the latest delivery signal, risks, and next checkpoint.'),
      {
        target: { value: 'Release paused pending rollback validation.' },
      },
    )
    await fireEvent.click(getByRole('button', { name: 'Post update' }))

    expect(createProjectUpdateThread).toHaveBeenCalledWith('project-1', {
      status: 'off_track',
      title: 'Hotfix hold',
      body: 'Release paused pending rollback validation.',
    })
    expect(await findByText('Update posted.')).toBeTruthy()
    expect(await findByText('Hotfix hold')).toBeTruthy()
    expect(await findByText('Off track', { selector: 'span' })).toBeTruthy()
  })

  it('creates, edits, deletes comments and refreshes from the activity stream', async () => {
    connectEventStream.mockImplementation(
      (_url: string, handlers: { onEvent?: StreamEventHandler }) => {
        return () => {}
      },
    )

    listProjectUpdates
      .mockResolvedValueOnce({
        threads: [
          {
            id: 'thread-1',
            project_id: 'project-1',
            status: 'on_track',
            title: 'Sprint 2 rollout',
            body_markdown: 'Launch is green.',
            created_by: 'user:codex',
            created_at: '2026-04-01T08:00:00Z',
            updated_at: '2026-04-01T08:15:00Z',
            edited_at: null,
            edit_count: 0,
            last_edited_by: null,
            is_deleted: false,
            deleted_at: null,
            deleted_by: null,
            last_activity_at: '2026-04-01T08:15:00Z',
            comment_count: 0,
            comments: [],
          },
        ],
      })
      .mockResolvedValueOnce({
        threads: [
          {
            id: 'thread-1',
            project_id: 'project-1',
            status: 'on_track',
            title: 'Sprint 2 rollout',
            body_markdown: 'Launch is green.',
            created_by: 'user:codex',
            created_at: '2026-04-01T08:00:00Z',
            updated_at: '2026-04-01T08:15:00Z',
            edited_at: null,
            edit_count: 0,
            last_edited_by: null,
            is_deleted: false,
            deleted_at: null,
            deleted_by: null,
            last_activity_at: '2026-04-01T10:32:00Z',
            comment_count: 1,
            comments: [
              {
                id: 'comment-1',
                thread_id: 'thread-1',
                body_markdown: 'Need one more canary.',
                created_by: 'user:ops',
                created_at: '2026-04-01T10:32:00Z',
                updated_at: '2026-04-01T10:32:00Z',
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
      })
      .mockResolvedValueOnce({
        threads: [
          {
            id: 'thread-1',
            project_id: 'project-1',
            status: 'on_track',
            title: 'Sprint 2 rollout',
            body_markdown: 'Launch is green.',
            created_by: 'user:codex',
            created_at: '2026-04-01T08:00:00Z',
            updated_at: '2026-04-01T08:15:00Z',
            edited_at: null,
            edit_count: 0,
            last_edited_by: null,
            is_deleted: false,
            deleted_at: null,
            deleted_by: null,
            last_activity_at: '2026-04-01T10:33:00Z',
            comment_count: 1,
            comments: [
              {
                id: 'comment-1',
                thread_id: 'thread-1',
                body_markdown: 'Need one more canary before noon.',
                created_by: 'user:ops',
                created_at: '2026-04-01T10:32:00Z',
                updated_at: '2026-04-01T10:33:00Z',
                edited_at: '2026-04-01T10:33:00Z',
                edit_count: 1,
                last_edited_by: 'user:ops',
                is_deleted: false,
                deleted_at: null,
                deleted_by: null,
              },
            ],
          },
        ],
      })
      .mockResolvedValueOnce({
        threads: [
          {
            id: 'thread-1',
            project_id: 'project-1',
            status: 'on_track',
            title: 'Sprint 2 rollout',
            body_markdown: 'Launch is green.',
            created_by: 'user:codex',
            created_at: '2026-04-01T08:00:00Z',
            updated_at: '2026-04-01T08:15:00Z',
            edited_at: null,
            edit_count: 0,
            last_edited_by: null,
            is_deleted: false,
            deleted_at: null,
            deleted_by: null,
            last_activity_at: '2026-04-01T10:34:00Z',
            comment_count: 1,
            comments: [
              {
                id: 'comment-1',
                thread_id: 'thread-1',
                body_markdown: 'Need one more canary before noon.',
                created_by: 'user:ops',
                created_at: '2026-04-01T10:32:00Z',
                updated_at: '2026-04-01T10:34:00Z',
                edited_at: '2026-04-01T10:34:00Z',
                edit_count: 2,
                last_edited_by: 'user:ops',
                is_deleted: true,
                deleted_at: '2026-04-01T10:34:00Z',
                deleted_by: 'user:ops',
              },
            ],
          },
        ],
      })
      .mockResolvedValueOnce({
        threads: [
          {
            id: 'thread-1',
            project_id: 'project-1',
            status: 'off_track',
            title: 'Sprint 2 rollout',
            body_markdown: 'A new project update event arrived.',
            created_by: 'user:codex',
            created_at: '2026-04-01T08:00:00Z',
            updated_at: '2026-04-01T10:35:00Z',
            edited_at: '2026-04-01T10:35:00Z',
            edit_count: 1,
            last_edited_by: 'user:codex',
            is_deleted: false,
            deleted_at: null,
            deleted_by: null,
            last_activity_at: '2026-04-01T10:35:00Z',
            comment_count: 1,
            comments: [
              {
                id: 'comment-1',
                thread_id: 'thread-1',
                body_markdown: 'Need one more canary before noon.',
                created_by: 'user:ops',
                created_at: '2026-04-01T10:32:00Z',
                updated_at: '2026-04-01T10:34:00Z',
                edited_at: '2026-04-01T10:34:00Z',
                edit_count: 2,
                last_edited_by: 'user:ops',
                is_deleted: true,
                deleted_at: '2026-04-01T10:34:00Z',
                deleted_by: 'user:ops',
              },
            ],
          },
        ],
      })

    createProjectUpdateComment.mockResolvedValue({ comment: { id: 'comment-1' } })
    updateProjectUpdateComment.mockResolvedValue({ comment: { id: 'comment-1' } })
    deleteProjectUpdateComment.mockResolvedValue({ deleted_comment_id: 'comment-1' })

    const { findByText, findByPlaceholderText, findByRole, getByLabelText, getByText } =
      render(ProjectUpdatesPage)

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
    if (streamHandlers?.onEvent) {
      streamHandlers.onEvent({
        event: 'message',
        data: JSON.stringify({ type: 'project_update_thread.status_changed' }),
      })
    }

    await waitFor(() => {
      expect(listProjectUpdates).toHaveBeenCalledTimes(5)
    })
    expect(await findByText('A new project update event arrived.')).toBeTruthy()
    expect(await findByText('Off track', { selector: 'span' })).toBeTruthy()
  })
})
