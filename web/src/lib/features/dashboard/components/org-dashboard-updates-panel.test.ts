import { fireEvent, render } from '@testing-library/svelte'
import { describe, expect, it, vi } from 'vitest'

import { parseProjectUpdateThreads } from '$lib/features/project-updates'
import OrgDashboardUpdatesPanel from './org-dashboard-updates-panel.svelte'

describe('OrgDashboardUpdatesPanel', () => {
  it('shows a load more button and calls the handler when older threads are available', async () => {
    const onLoadMoreThreads = vi.fn()
    const threads = parseProjectUpdateThreads([
      {
        id: 'thread-1',
        project_id: 'project-1',
        status: 'on_track',
        title: 'Newest thread',
        body_markdown: 'Top of the list',
        created_by: 'user:codex',
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
    ])

    const view = render(OrgDashboardUpdatesPanel, {
      props: {
        threads,
        loading: false,
        initialLoaded: true,
        creatingThread: false,
        loadError: '',
        hasMoreThreads: true,
        loadingMoreThreads: false,
        onLoadMoreThreads,
      },
    })

    await fireEvent.click(view.getByRole('button', { name: 'Load more' }))

    expect(onLoadMoreThreads).toHaveBeenCalledTimes(1)
  })
})
