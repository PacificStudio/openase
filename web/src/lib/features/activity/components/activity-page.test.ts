import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ActivityPayload, Project, TicketPayload } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import ActivityPage from './activity-page.svelte'
import { markProjectActivityCacheDirty, resetProjectActivityCacheForTests } from '../activity-cache'

const { listActivity, listTickets, subscribeProjectEvents } = vi.hoisted(() => ({
  listActivity: vi.fn(),
  listTickets: vi.fn(),
  subscribeProjectEvents: vi.fn(),
}))

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

const activityFixture: ActivityPayload = {
  events: [
    {
      id: 'activity-1',
      project_id: 'project-1',
      ticket_id: 'ticket-1',
      agent_id: 'agent-1',
      event_type: 'ticket.updated',
      message: 'Updated ticket ASE-101',
      metadata: { agent_name: 'Coding Agent' },
      created_at: '2026-04-02T10:00:00Z',
    },
  ],
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
      external_ref: '',
      budget_usd: 0,
      cost_tokens_input: 0,
      cost_tokens_output: 0,
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

describe('ActivityPage cache behavior', () => {
  beforeEach(() => {
    appStore.currentProject = projectFixture
    subscribeProjectEvents.mockReturnValue(() => {})
    listActivity.mockResolvedValue(activityFixture)
    listTickets.mockResolvedValue(ticketPayload)
  })

  afterEach(() => {
    cleanup()
    resetProjectActivityCacheForTests()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('reuses the cached activity snapshot when remounting the page in the same project', async () => {
    const firstRender = render(ActivityPage)
    expect(await firstRender.findByText('Updated ticket ASE-101')).toBeTruthy()

    expect(listActivity).toHaveBeenCalledTimes(1)
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

    deferredActivity.resolve(activityFixture)
    deferredTickets.resolve(ticketPayload)

    await waitFor(() => {
      expect(secondRender.getByText('Updated ticket ASE-101')).toBeTruthy()
    })
  })
})
