import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ProjectEventEnvelope } from '$lib/features/project-events'
import OrgDashboard from './org-dashboard.svelte'
import {
  activityPayload,
  agentPayload,
  hrAdvisorPayload,
  organizationSummaryPayload,
  seedOrgDashboardStore,
  systemDashboardPayload,
  ticketPayload,
} from './org-dashboard.test-fixtures'

const projectEventListeners = new Set<(event: ProjectEventEnvelope) => void>()

const {
  createProjectUpdateComment,
  createProjectUpdateThread,
  deleteProjectUpdateComment,
  deleteProjectUpdateThread,
  getHRAdvisor,
  getProjectTokenUsage,
  getSystemDashboard,
  listActivity,
  listAgents,
  listProjectUpdates,
  listTickets,
  updateProject,
  updateProjectUpdateComment,
  updateProjectUpdateThread,
} = vi.hoisted(() => ({
  createProjectUpdateComment: vi.fn(),
  createProjectUpdateThread: vi.fn(),
  deleteProjectUpdateComment: vi.fn(),
  deleteProjectUpdateThread: vi.fn(),
  getHRAdvisor: vi.fn(),
  getProjectTokenUsage: vi.fn(),
  getSystemDashboard: vi.fn(),
  listActivity: vi.fn(),
  listAgents: vi.fn(),
  listProjectUpdates: vi.fn(),
  listTickets: vi.fn(),
  updateProject: vi.fn(),
  updateProjectUpdateComment: vi.fn(),
  updateProjectUpdateThread: vi.fn(),
}))

const { loadOrganizationDashboardSummary } = vi.hoisted(() => ({
  loadOrganizationDashboardSummary: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createProjectUpdateComment,
  createProjectUpdateThread,
  deleteProjectUpdateComment,
  deleteProjectUpdateThread,
  getHRAdvisor,
  getProjectTokenUsage,
  getSystemDashboard,
  listActivity,
  listAgents,
  listProjectUpdates,
  listTickets,
  updateProject,
  updateProjectUpdateComment,
  updateProjectUpdateThread,
}))

vi.mock('../organization-summary', () => ({
  loadOrganizationDashboardSummary,
}))

vi.mock('../model', async () => {
  const actual = await vi.importActual<typeof import('../model')>('../model')
  return {
    ...actual,
    shouldShowProjectOnboarding: vi.fn(() => false),
  }
})

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/features/onboarding', () => ({
  markProjectOnboardingCompleted: vi.fn(),
  readProjectOnboardingCompletion: vi.fn(() => true),
  OnboardingPanel: class {},
}))

vi.mock('$lib/features/project-events', async () => {
  const actual = await vi.importActual<typeof import('$lib/features/project-events')>(
    '$lib/features/project-events',
  )
  return {
    ...actual,
    subscribeProjectEvents: vi.fn((_: string, listener: (event: ProjectEventEnvelope) => void) => {
      projectEventListeners.add(listener)
      return () => {
        projectEventListeners.delete(listener)
      }
    }),
  }
})

describe('OrgDashboard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-02T12:00:00Z'))
    seedOrgDashboardStore()

    listAgents.mockResolvedValue(agentPayload)
    listTickets.mockResolvedValue(ticketPayload)
    listActivity.mockResolvedValue(activityPayload)
    getSystemDashboard.mockResolvedValue(systemDashboardPayload)
    getHRAdvisor.mockResolvedValue(hrAdvisorPayload)
    getProjectTokenUsage.mockResolvedValue({
      days: [
        {
          date: '2026-04-02',
          input_tokens: 90,
          output_tokens: 30,
          cached_input_tokens: 10,
          reasoning_tokens: 4,
          total_tokens: 120,
          finalized_run_count: 2,
        },
      ],
      summary: {
        total_tokens: 120,
        avg_daily_tokens: 4,
        peak_day: {
          date: '2026-04-02',
          total_tokens: 120,
        },
      },
    })
    listProjectUpdates.mockResolvedValue({ threads: [], has_more: false, next_cursor: '' })
    loadOrganizationDashboardSummary.mockResolvedValue(organizationSummaryPayload)
  })

  afterEach(() => {
    cleanup()
    projectEventListeners.clear()
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  it('loads once, stops 1s polling, and only refreshes memory on the slow interval', async () => {
    render(OrgDashboard)

    await waitFor(() => {
      expect(listAgents).toHaveBeenCalledTimes(1)
      expect(listTickets).toHaveBeenCalledTimes(1)
      expect(listActivity).toHaveBeenCalledTimes(1)
      expect(getSystemDashboard).toHaveBeenCalledTimes(1)
      expect(getHRAdvisor).toHaveBeenCalledTimes(1)
      expect(getProjectTokenUsage).toHaveBeenCalledTimes(1)
      expect(loadOrganizationDashboardSummary).toHaveBeenCalledTimes(1)
      expect(listProjectUpdates).toHaveBeenCalledTimes(1)
    })

    await vi.advanceTimersByTimeAsync(3000)

    expect(listAgents).toHaveBeenCalledTimes(1)
    expect(listTickets).toHaveBeenCalledTimes(1)
    expect(listActivity).toHaveBeenCalledTimes(1)
    expect(getSystemDashboard).toHaveBeenCalledTimes(1)
    expect(getHRAdvisor).toHaveBeenCalledTimes(1)
    expect(getProjectTokenUsage).toHaveBeenCalledTimes(1)
    expect(loadOrganizationDashboardSummary).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(10_000)

    await waitFor(() => {
      expect(getSystemDashboard).toHaveBeenCalledTimes(2)
    })

    expect(listAgents).toHaveBeenCalledTimes(1)
    expect(listTickets).toHaveBeenCalledTimes(1)
    expect(listActivity).toHaveBeenCalledTimes(1)
    expect(getHRAdvisor).toHaveBeenCalledTimes(1)
    expect(getProjectTokenUsage).toHaveBeenCalledTimes(1)
    expect(loadOrganizationDashboardSummary).toHaveBeenCalledTimes(1)
  })

  it('refreshes only the dirty dashboard slices when the project bus emits a coalesced refresh event', async () => {
    render(OrgDashboard)

    await waitFor(() => {
      expect(listAgents).toHaveBeenCalledTimes(1)
      expect(listTickets).toHaveBeenCalledTimes(1)
      expect(listActivity).toHaveBeenCalledTimes(1)
      expect(getSystemDashboard).toHaveBeenCalledTimes(1)
      expect(getHRAdvisor).toHaveBeenCalledTimes(1)
      expect(getProjectTokenUsage).toHaveBeenCalledTimes(1)
      expect(loadOrganizationDashboardSummary).toHaveBeenCalledTimes(1)
    })

    for (const listener of [...projectEventListeners]) {
      listener({
        topic: 'project.dashboard.events',
        type: 'project.dashboard.refresh',
        payload: {
          project_id: 'project-1',
          dirty_sections: ['agents', 'tickets'],
        },
        publishedAt: '2026-04-02T10:00:01Z',
      })
    }

    await waitFor(() => {
      expect(listAgents).toHaveBeenCalledTimes(2)
      expect(listTickets).toHaveBeenCalledTimes(2)
    })

    expect(listActivity).toHaveBeenCalledTimes(1)
    expect(getSystemDashboard).toHaveBeenCalledTimes(1)
    expect(getHRAdvisor).toHaveBeenCalledTimes(1)
    expect(getProjectTokenUsage).toHaveBeenCalledTimes(1)
    expect(loadOrganizationDashboardSummary).toHaveBeenCalledTimes(1)
  })

  it('loads project token usage on the dashboard and refreshes the selected window', async () => {
    const view = render(OrgDashboard)

    await waitFor(() => {
      expect(getProjectTokenUsage).toHaveBeenCalledTimes(1)
    })

    expect(getProjectTokenUsage.mock.calls[0]?.[0]).toBe('project-1')
    expect(getProjectTokenUsage.mock.calls[0]?.[1]).toEqual({
      from: '2026-03-04',
      to: '2026-04-02',
    })
    expect(view.getByText('Token Usage')).toBeTruthy()

    await fireEvent.click(view.getByRole('button', { name: '7d' }))

    await waitFor(() => {
      expect(getProjectTokenUsage).toHaveBeenCalledTimes(2)
    })

    expect(getProjectTokenUsage.mock.calls[1]?.[1]).toEqual({
      from: '2026-03-27',
      to: '2026-04-02',
    })
  })
})
