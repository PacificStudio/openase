import { fireEvent, render, waitFor } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'

import {
  listProjectUpdates,
  makeProjectUpdatesPayload,
  makeThreadRecord,
  setupProjectUpdatesPageTest,
  subscribeProjectEvents,
  type StreamEventHandler,
} from './project-updates-page.test-support'
import { ProjectUpdatesPage } from '..'

describe('ProjectUpdatesPage pagination', () => {
  setupProjectUpdatesPageTest()

  it('loads the first page, appends older threads, and refreshes page one without duplicates', async () => {
    const newest = makeThreadRecord({
      id: 'thread-newest',
      title: 'Newest thread',
      body_markdown: 'Newest thread',
      last_activity_at: '2026-04-01T10:02:00Z',
    })
    const middle = makeThreadRecord({
      id: 'thread-middle',
      title: 'Middle thread',
      body_markdown: 'Middle thread',
      last_activity_at: '2026-04-01T10:01:00Z',
    })
    const older = makeThreadRecord({
      id: 'thread-older',
      title: 'Older thread',
      body_markdown: 'Older thread',
      last_activity_at: '2026-04-01T09:55:00Z',
    })
    const olderPromoted = makeThreadRecord({
      id: 'thread-older',
      title: 'Older thread',
      last_activity_at: '2026-04-01T10:03:00Z',
      body_markdown: 'An older thread just became active again.',
      comment_count: 1,
    })

    listProjectUpdates
      .mockResolvedValueOnce(
        makeProjectUpdatesPayload({
          threads: [newest, middle],
          has_more: true,
          next_cursor: 'cursor-page-1',
        }),
      )
      .mockResolvedValueOnce(
        makeProjectUpdatesPayload({
          threads: [older],
          has_more: false,
          next_cursor: '',
        }),
      )
      .mockResolvedValueOnce(
        makeProjectUpdatesPayload({
          threads: [olderPromoted, newest],
          has_more: true,
          next_cursor: 'cursor-page-1-refresh',
        }),
      )

    const view = render(ProjectUpdatesPage)

    expect(await view.findByText('Newest thread')).toBeTruthy()
    expect(listProjectUpdates).toHaveBeenNthCalledWith(1, 'project-1', { limit: 10 })

    await fireEvent.click(await view.findByRole('button', { name: 'Load more' }))

    expect(listProjectUpdates).toHaveBeenNthCalledWith(2, 'project-1', {
      limit: 10,
      before: 'cursor-page-1',
    })
    expect(await view.findByText('Older thread')).toBeTruthy()
    expect(view.queryByRole('button', { name: 'Load more' })).toBeNull()

    const onEvent = subscribeProjectEvents.mock.calls.at(-1)?.[1] as StreamEventHandler | undefined
    onEvent?.({
      topic: 'activity.events',
      type: 'project_update_comment.created',
      payload: null,
      publishedAt: '2026-04-01T10:03:00Z',
    })

    await waitFor(() => {
      expect(listProjectUpdates).toHaveBeenCalledTimes(3)
    })

    expect(await view.findByText('An older thread just became active again.')).toBeTruthy()

    const text = view.container.textContent ?? ''
    expect(text.indexOf('An older thread just became active again.')).toBeGreaterThanOrEqual(0)
    expect(text.indexOf('An older thread just became active again.')).toBeLessThan(
      text.indexOf('Newest thread'),
    )
    expect(text.indexOf('Newest thread')).toBeLessThan(text.indexOf('Middle thread'))
  })
})
