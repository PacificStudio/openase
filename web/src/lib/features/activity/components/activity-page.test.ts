import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ActivityPayload, Project, TicketPayload } from '$lib/api/contracts'
import type { ProjectEventEnvelope } from '$lib/features/project-events'
import { appStore } from '$lib/stores/app.svelte'
import ActivityPage from './activity-page.svelte'
import { markProjectActivityCacheDirty, resetProjectActivityCacheForTests } from '../activity-cache'

const { listActivity, listTickets, subscribeProjectEvents } = vi.hoisted(() => ({
  listActivity: vi.fn(),
  listTickets: vi.fn(),
  subscribeProjectEvents: vi.fn(),
}))

let projectEventListener: ((event: ProjectEventEnvelope) => void) | null = null

vi.mock('$lib/api/openase', () => ({
  listActivity,
  listTickets,
}))

vi.mock('$lib/features/project-events', async () => {
  const actual = await vi.importActual<typeof import('$lib/features/project-events')>(
    '$lib/features/project-events',
  )
  return {
    ...actual,
    subscribeProjectEvents,
  }
})

const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'OpenASE',
  slug: 'openase',
  description: '',
  status: 'active',
  default_agent_provider_id: null,
  accessible_machine_ids: [],
  max_concurrent_agents: 4,
}

const ticketPayload: TicketPayload = {
  tickets: [
    {
      id: 'ticket-1',
      identifier: 'ASE-101',
      project_id: 'project-1',
      title: 'Fix dashboard refresh',
      description: '',
      status_id: 'status-1',
      status_name: 'Todo',
      priority: 'high',
      type: 'task',
      archived: false,
      workflow_id: null,
      current_run_id: null,
      target_machine_id: null,
      created_by: 'user:test',
      parent: null,
      children: [],
      dependencies: [],
      external_links: [],
      pull_request_urls: [],
      external_ref: '',
      budget_usd: 0,
      cost_tokens_input: 0,
      cost_tokens_output: 0,
      cost_tokens_total: 0,
      cost_amount: 0,
      attempt_count: 0,
      consecutive_errors: 0,
      started_at: null,
      completed_at: null,
      next_retry_at: null,
      retry_paused: false,
      pause_reason: '',
      created_at: '2026-04-02T09:00:00Z',
    },
  ],
}

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

function activityEvent(
  id: string,
  message: string,
  createdAt: string,
  overrides: Partial<ActivityPayload['events'][number]> = {},
): ActivityPayload['events'][number] {
  return {
    id,
    project_id: 'project-1',
    ticket_id: 'ticket-1',
    agent_id: 'agent-1',
    event_type: 'ticket.updated',
    message,
    metadata: { agent_name: 'Coding Agent' },
    created_at: createdAt,
    ...overrides,
  }
}

function activityPayload(
  events: ActivityPayload['events'],
  pagination: { next_cursor?: string; has_more?: boolean } = {},
): ActivityPayload {
  return {
    events,
    next_cursor: pagination.next_cursor ?? '',
    has_more: pagination.has_more ?? false,
  }
}

describe('ActivityPage', () => {
  beforeEach(() => {
    appStore.currentProject = projectFixture
    projectEventListener = null
    subscribeProjectEvents.mockImplementation(
      (_: string, listener: (event: ProjectEventEnvelope) => void) => {
        projectEventListener = listener
        return () => {
          if (projectEventListener === listener) {
            projectEventListener = null
          }
        }
      },
    )
    listTickets.mockResolvedValue(ticketPayload)
    listActivity.mockResolvedValue(
      activityPayload([
        activityEvent('activity-1', 'Updated ticket ASE-101', '2026-04-02T10:00:00Z'),
      ]),
    )
  })

  afterEach(() => {
    cleanup()
    resetProjectActivityCacheForTests()
    appStore.currentProject = null
    projectEventListener = null
    vi.clearAllMocks()
  })

  it('loads the first activity page with pagination state and reuses the cache on remount', async () => {
    const firstRender = render(ActivityPage)
    expect(await firstRender.findByText('Updated ticket ASE-101')).toBeTruthy()

    expect(listActivity).toHaveBeenCalledWith(projectFixture.id, { limit: 40 })
    expect(listTickets).toHaveBeenCalledTimes(1)

    firstRender.unmount()

    const secondRender = render(ActivityPage)
    expect(await secondRender.findByText('Updated ticket ASE-101')).toBeTruthy()

    expect(listActivity).toHaveBeenCalledTimes(1)
    expect(listTickets).toHaveBeenCalledTimes(1)
  })

  it('shows cached activity immediately and refreshes in the background when the cache is dirty', async () => {
    const firstRender = render(ActivityPage)
    expect(await firstRender.findByText('Updated ticket ASE-101')).toBeTruthy()
    firstRender.unmount()

    markProjectActivityCacheDirty(projectFixture.id)

    const deferredActivity = createDeferred<ActivityPayload>()
    const deferredTickets = createDeferred<TicketPayload>()
    listActivity.mockImplementationOnce(() => deferredActivity.promise)
    listTickets.mockImplementationOnce(() => deferredTickets.promise)

    const secondRender = render(ActivityPage)
    expect(await secondRender.findByText('Updated ticket ASE-101')).toBeTruthy()

    expect(listActivity).toHaveBeenCalledTimes(2)
    expect(listTickets).toHaveBeenCalledTimes(2)

    deferredActivity.resolve(
      activityPayload([
        activityEvent('activity-1', 'Updated ticket ASE-101', '2026-04-02T10:00:00Z'),
      ]),
    )
    deferredTickets.resolve(ticketPayload)

    await waitFor(() => {
      expect(secondRender.getByText('Updated ticket ASE-101')).toBeTruthy()
    })
  })

  it('appends older events when Load more is clicked and hides the control when there is no more history', async () => {
    listActivity
      .mockResolvedValueOnce(
        activityPayload(
          [
            activityEvent('activity-3', 'Newest page item', '2026-04-02T10:02:00Z'),
            activityEvent('activity-2', 'Second page item', '2026-04-02T10:01:00Z'),
          ],
          { next_cursor: 'cursor-page-1', has_more: true },
        ),
      )
      .mockResolvedValueOnce(
        activityPayload([activityEvent('activity-1', 'Older page item', '2026-04-02T10:00:00Z')], {
          next_cursor: '',
          has_more: false,
        }),
      )

    const view = render(ActivityPage)
    expect(await view.findByText('Newest page item')).toBeTruthy()

    const loadMoreButton = view.getByRole('button', { name: 'Load more' })
    await fireEvent.click(loadMoreButton)

    await waitFor(() => {
      expect(view.getByText('Older page item')).toBeTruthy()
    })

    expect(listActivity).toHaveBeenNthCalledWith(1, projectFixture.id, { limit: 40 })
    expect(listActivity).toHaveBeenNthCalledWith(2, projectFixture.id, {
      limit: 40,
      before: 'cursor-page-1',
    })
    expect(view.queryByRole('button', { name: 'Load more' })).toBeNull()
  })

  it('dedupes and preserves older history when realtime refreshes reload the first page', async () => {
    listActivity
      .mockResolvedValueOnce(
        activityPayload(
          [
            activityEvent('activity-2', 'Existing latest item', '2026-04-02T10:01:00Z'),
            activityEvent('activity-1', 'Existing oldest loaded item', '2026-04-02T10:00:00Z'),
          ],
          { next_cursor: 'cursor-page-1', has_more: true },
        ),
      )
      .mockResolvedValueOnce(
        activityPayload(
          [activityEvent('activity-0', 'Loaded older history', '2026-04-02T09:59:00Z')],
          { next_cursor: '', has_more: false },
        ),
      )
      .mockResolvedValueOnce(
        activityPayload(
          [
            activityEvent('activity-3', 'Realtime fresh item', '2026-04-02T10:02:00Z'),
            activityEvent('activity-2', 'Existing latest item', '2026-04-02T10:01:00Z'),
          ],
          { next_cursor: 'cursor-refreshed-head', has_more: true },
        ),
      )

    const view = render(ActivityPage)
    expect(await view.findByText('Existing latest item')).toBeTruthy()

    await fireEvent.click(view.getByRole('button', { name: 'Load more' }))
    expect(await view.findByText('Loaded older history')).toBeTruthy()

    projectEventListener?.({
      topic: 'project.activity.events',
      type: 'activity.created',
      payload: {},
      publishedAt: '2026-04-02T10:02:30Z',
    } as ProjectEventEnvelope)

    await waitFor(() => {
      expect(view.getByText('Realtime fresh item')).toBeTruthy()
    })

    expect(view.getAllByText('Existing latest item')).toHaveLength(1)
    expect(view.getByText('Loaded older history')).toBeTruthy()

    const text = view.container.textContent ?? ''
    expect(text.indexOf('Realtime fresh item')).toBeLessThan(text.indexOf('Existing latest item'))
    expect(text.indexOf('Existing latest item')).toBeLessThan(text.indexOf('Loaded older history'))
  })

  it('keeps search filtering working across paginated history', async () => {
    listActivity
      .mockResolvedValueOnce(
        activityPayload(
          [
            activityEvent('activity-2', 'Fresh rollout update', '2026-04-02T10:01:00Z'),
            activityEvent('activity-1', 'Deploy finished', '2026-04-02T10:00:00Z'),
          ],
          { next_cursor: 'cursor-page-1', has_more: true },
        ),
      )
      .mockResolvedValueOnce(
        activityPayload(
          [activityEvent('activity-0', 'Cursor pagination landed', '2026-04-02T09:59:00Z')],
          { next_cursor: '', has_more: false },
        ),
      )

    const view = render(ActivityPage)
    expect(await view.findByText('Fresh rollout update')).toBeTruthy()

    await fireEvent.click(view.getByRole('button', { name: 'Load more' }))
    expect(await view.findByText('Cursor pagination landed')).toBeTruthy()

    await fireEvent.input(view.getByPlaceholderText('Search events...'), {
      target: { value: 'cursor pagination' },
    })

    await waitFor(() => {
      expect(view.getByText('Cursor pagination landed')).toBeTruthy()
    })
    expect(view.queryByText('Fresh rollout update')).toBeNull()
  })
})
