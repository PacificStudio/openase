import { fireEvent, render } from '@testing-library/svelte'
import { describe, expect, it, vi } from 'vitest'

import { parseProjectUpdateThreads } from '$lib/features/project-updates/model'
import { makeThreadRecord } from '$lib/features/project-updates/components/project-updates-page.test-support'
import OrgDashboardUpdatesPanel from './org-dashboard-updates-panel.svelte'

describe('OrgDashboardUpdatesPanel', () => {
  it('shows a load more button and calls the handler when older threads are available', async () => {
    const onLoadMoreThreads = vi.fn()
    const threads = parseProjectUpdateThreads([
      makeThreadRecord({
        id: 'thread-1',
        title: 'Newest thread',
        body_markdown: 'Top of the list',
      }),
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
