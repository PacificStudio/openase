import { describe, expect, it } from 'vitest'

import type { ScheduledJob } from '$lib/api/contracts'
import type { RepoScopeOption } from '$lib/features/repo-scope-selection'
import type { WorkflowStatusOption } from '$lib/features/workflows'

import {
  emptyScheduledJobDraft,
  parseScheduledJobDraft,
  scheduledJobDraftFromRecord,
} from './workflow-scheduled-jobs'

const statuses: WorkflowStatusOption[] = [{ id: 'todo', name: 'Todo', stage: 'unstarted' }]

describe('workflow scheduled jobs repo scope parsing', () => {
  it('auto-selects the only repository for single-repo projects', () => {
    const repoOptions: RepoScopeOption[] = [
      { id: 'repo-1', label: 'backend', defaultBranch: 'main' },
    ]

    const parsed = parseScheduledJobDraft(
      {
        ...emptyScheduledJobDraft('todo', repoOptions),
        name: 'Nightly',
        cronExpression: '0 1 * * *',
        ticketTitle: 'Nightly checks',
      },
      statuses,
      repoOptions,
    )

    expect(parsed).toEqual({
      ok: true,
      value: {
        name: 'Nightly',
        cron_expression: '0 1 * * *',
        is_enabled: true,
        ticket_template: {
          priority: 'medium',
          repo_scopes: [{ repo_id: 'repo-1' }],
          status: 'Todo',
          title: 'Nightly checks',
          type: 'feature',
        },
      },
    })
  })

  it('requires an explicit repository scope for multi-repo projects', () => {
    const repoOptions: RepoScopeOption[] = [
      { id: 'repo-1', label: 'backend', defaultBranch: 'main' },
      { id: 'repo-2', label: 'frontend', defaultBranch: 'develop' },
    ]

    const parsed = parseScheduledJobDraft(
      {
        ...emptyScheduledJobDraft('todo', repoOptions),
        name: 'Nightly',
        cronExpression: '0 1 * * *',
        ticketTitle: 'Nightly checks',
      },
      statuses,
      repoOptions,
    )

    expect(parsed).toEqual({
      ok: false,
      error: 'Select at least one repository scope for tickets created by this scheduled job.',
    })
  })

  it('includes branch overrides for selected repositories', () => {
    const repoOptions: RepoScopeOption[] = [
      { id: 'repo-1', label: 'backend', defaultBranch: 'main' },
      { id: 'repo-2', label: 'frontend', defaultBranch: 'develop' },
    ]

    const parsed = parseScheduledJobDraft(
      {
        ...emptyScheduledJobDraft('todo', repoOptions),
        name: 'Nightly',
        cronExpression: '0 1 * * *',
        ticketTitle: 'Nightly checks',
        ticketRepoIds: ['repo-2', 'repo-1'],
        ticketRepoBranchOverrides: {
          'repo-2': 'release/train',
        },
      },
      statuses,
      repoOptions,
    )

    expect(parsed).toEqual({
      ok: true,
      value: {
        name: 'Nightly',
        cron_expression: '0 1 * * *',
        is_enabled: true,
        ticket_template: {
          priority: 'medium',
          repo_scopes: [{ repo_id: 'repo-1' }, { repo_id: 'repo-2', branch_name: 'release/train' }],
          status: 'Todo',
          title: 'Nightly checks',
          type: 'feature',
        },
      },
    })
  })

  it('hydrates repo scope draft state from the scheduled job record', () => {
    const job: ScheduledJob = {
      id: 'job-1',
      project_id: 'project-1',
      name: 'Nightly',
      cron_expression: '0 1 * * *',
      is_enabled: true,
      last_run_at: null,
      next_run_at: null,
      ticket_template: {
        title: 'Nightly checks',
        description: '',
        status: 'Todo',
        priority: 'medium',
        type: 'feature',
        created_by: '',
        budget_usd: 0,
        repo_scopes: [{ repo_id: 'repo-1' }, { repo_id: 'repo-2', branch_name: 'release/train' }],
      },
    }
    const repoOptions: RepoScopeOption[] = [
      { id: 'repo-1', label: 'backend', defaultBranch: 'main' },
      { id: 'repo-2', label: 'frontend', defaultBranch: 'develop' },
    ]

    expect(scheduledJobDraftFromRecord(job, statuses, repoOptions)).toMatchObject({
      ticketRepoIds: ['repo-1', 'repo-2'],
      ticketRepoBranchOverrides: {
        'repo-2': 'release/train',
      },
    })
  })
})
