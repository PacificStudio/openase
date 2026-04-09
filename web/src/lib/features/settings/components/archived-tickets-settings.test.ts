import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import ArchivedTicketsSettings from './archived-tickets-settings.svelte'

const { listArchivedTickets, updateTicket } = vi.hoisted(() => ({
  listArchivedTickets: vi.fn(),
  updateTicket: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/api/openase', () => ({
  listArchivedTickets,
  updateTicket,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

function seedProject() {
  appStore.currentProject = {
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
}

function buildArchivedTicket(
  overrides: Partial<{
    id: string
    identifier: string
    title: string
    completed_at: string | null
    created_at: string
  }> = {},
) {
  return {
    id: 'ticket-1',
    project_id: 'project-1',
    identifier: 'ASE-1',
    title: 'Archived ticket 1',
    description: '',
    status_id: 'status-archived',
    status_name: 'Archived',
    archived: true,
    priority: 'medium',
    type: 'feature',
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
    completed_at: '2026-04-02T10:00:00Z',
    next_retry_at: null,
    retry_paused: false,
    pause_reason: '',
    created_at: '2026-04-01T10:00:00Z',
    ...overrides,
  }
}

describe('Archived tickets settings', () => {
  beforeEach(() => {
    seedProject()
  })

  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('loads archived tickets page by page', async () => {
    listArchivedTickets
      .mockResolvedValueOnce({
        tickets: [buildArchivedTicket()],
        total: 21,
        page: 1,
        per_page: 20,
      })
      .mockResolvedValueOnce({
        tickets: [
          buildArchivedTicket({
            id: 'ticket-21',
            identifier: 'ASE-21',
            title: 'Archived ticket 21',
            completed_at: '2026-04-02T11:00:00Z',
            created_at: '2026-04-01T11:00:00Z',
          }),
        ],
        total: 21,
        page: 2,
        per_page: 20,
      })

    const { findByRole, findByText } = render(ArchivedTicketsSettings)

    expect(await findByText('Archived ticket 1')).toBeTruthy()
    expect(listArchivedTickets).toHaveBeenCalledWith('project-1', { page: 1, per_page: 20 })

    await fireEvent.click(await findByRole('button', { name: 'Next' }))

    await waitFor(() =>
      expect(listArchivedTickets).toHaveBeenLastCalledWith('project-1', { page: 2, per_page: 20 }),
    )
    expect(await findByText('Archived ticket 21')).toBeTruthy()
  })

  it('restores selected tickets and refreshes the current page', async () => {
    listArchivedTickets
      .mockResolvedValueOnce({
        tickets: [buildArchivedTicket({ title: 'Restore me' })],
        total: 1,
        page: 1,
        per_page: 20,
      })
      .mockResolvedValueOnce({
        tickets: [],
        total: 0,
        page: 1,
        per_page: 20,
      })
    updateTicket.mockResolvedValue({
      ticket: {
        id: 'ticket-1',
      },
    })

    const { findByRole, findByText } = render(ArchivedTicketsSettings)

    await fireEvent.click(await findByRole('button', { name: /ASE-1 Restore me/i }))
    await fireEvent.click(await findByRole('button', { name: 'Restore 1 ticket' }))

    await waitFor(() => expect(updateTicket).toHaveBeenCalledWith('ticket-1', { archived: false }))
    await waitFor(() =>
      expect(listArchivedTickets).toHaveBeenLastCalledWith('project-1', { page: 1, per_page: 20 }),
    )
    expect(toastStore.success).toHaveBeenCalledWith('1 ticket restored.')
    expect(await findByText('No archived tickets')).toBeTruthy()
  })
})
