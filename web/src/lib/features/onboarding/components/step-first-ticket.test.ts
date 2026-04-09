import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ProjectRepoRecord, TicketStatus } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import StepFirstTicket from './step-first-ticket.svelte'

const { createTicket, listProjectRepos } = vi.hoisted(() => ({
  createTicket: vi.fn(),
  listProjectRepos: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createTicket,
  listProjectRepos,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('StepFirstTicket', () => {
  beforeEach(() => {
    appStore.openRightPanel = vi.fn()
    listProjectRepos.mockResolvedValue({ repos: [makeProjectRepo()] })
    createTicket.mockResolvedValue({
      ticket: {
        id: 'ticket-1',
        identifier: 'ASE-1',
      },
    })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('creates the first ticket with the recommended pickup status instead of binding a workflow', async () => {
    const onComplete = vi.fn()
    const { getByText, queryByText } = render(StepFirstTicket, {
      props: {
        projectId: 'project-1',
        orgId: 'org-1',
        projectStatus: 'Planned',
        statuses: [
          makeStatus({ id: 'status-backlog', name: 'Backlog', stage: 'backlog' }),
          makeStatus({ id: 'status-todo', name: 'Todo', stage: 'unstarted' }),
          makeStatus({ id: 'status-done', name: 'Done', stage: 'completed' }),
        ],
        ticketCount: 0,
        onComplete,
      },
    })

    expect(queryByText('Workflow')).toBeNull()
    expect(
      getByText(
        'The ticket will enter "Backlog", and the orchestrator will automatically pick it up and assign it to an agent based on the status pickup rules.',
      ),
    ).toBeTruthy()

    await fireEvent.click(getByText('Create ticket'))

    await waitFor(() => {
      expect(createTicket).toHaveBeenCalledWith(
        'project-1',
        expect.objectContaining({
          title: 'Draft the initial product requirements',
          status_id: 'status-backlog',
        }),
      )
      expect(createTicket.mock.calls[0]?.[1]).not.toHaveProperty('workflow_id')
      expect(onComplete).toHaveBeenCalled()
      expect(appStore.openRightPanel).toHaveBeenCalledWith({ type: 'ticket', id: 'ticket-1' })
    })
  })
})

function makeProjectRepo(overrides: Partial<ProjectRepoRecord> = {}): ProjectRepoRecord {
  const repo: ProjectRepoRecord = {
    id: 'repo-1',
    project_id: 'project-1',
    name: 'openase',
    repository_url: 'https://github.com/octo-org/openase.git',
    default_branch: 'main',
    workspace_dirname: 'openase',
    labels: [],
    ...overrides,
  }
  repo.workspace_dirname = overrides.workspace_dirname ?? 'openase'
  return repo
}

function makeStatus(overrides: Partial<TicketStatus> = {}): TicketStatus {
  const status: TicketStatus = {
    id: 'status-1',
    project_id: 'project-1',
    name: 'Todo',
    stage: 'unstarted',
    color: '#3B82F6',
    icon: '',
    position: 1,
    is_default: false,
    active_runs: 0,
    max_active_runs: 0,
    description: '',
    ...overrides,
  }
  status.active_runs = overrides.active_runs ?? 0
  return status
}
