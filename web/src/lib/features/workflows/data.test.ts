import { describe, expect, it, vi } from 'vitest'

const { getWorkflowRepositoryPrerequisite } = vi.hoisted(() => ({
  getWorkflowRepositoryPrerequisite: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  getWorkflowRepositoryPrerequisite,
}))

import { loadWorkflowRepositoryPrerequisite } from './data'

describe('workflow repository prerequisite', () => {
  it('requires a primary repository when the project has no repos', async () => {
    getWorkflowRepositoryPrerequisite.mockResolvedValue({
      prerequisite: {
        kind: 'missing_primary_repo',
        repo_count: 0,
        mirror_count: 0,
        action: 'bind_primary_repo',
      },
    })

    await expect(loadWorkflowRepositoryPrerequisite('project-1')).resolves.toEqual({
      kind: 'missing_primary_repo',
      repoCount: 0,
      action: 'bind_primary_repo',
    })
  })

  it('surfaces a bound primary repo whose mirror is not ready', async () => {
    getWorkflowRepositoryPrerequisite.mockResolvedValue({
      prerequisite: {
        kind: 'primary_mirror_not_ready',
        repo_count: 1,
        primary_repo_id: 'repo-1',
        primary_repo_name: 'openase',
        mirror_count: 1,
        mirror_state: 'error',
        mirror_machine_id: 'machine-1',
        mirror_last_error: 'sync failed',
        action: 'sync_primary_mirror',
      },
    })

    await expect(loadWorkflowRepositoryPrerequisite('project-1')).resolves.toEqual({
      kind: 'primary_mirror_not_ready',
      repoCount: 1,
      primaryRepoId: 'repo-1',
      primaryRepoName: 'openase',
      mirrorCount: 1,
      mirrorState: 'error',
      mirrorMachineId: 'machine-1',
      mirrorLastError: 'sync failed',
      action: 'sync_primary_mirror',
    })
  })

  it('accepts a project once a primary mirror is ready', async () => {
    getWorkflowRepositoryPrerequisite.mockResolvedValue({
      prerequisite: {
        kind: 'ready',
        repo_count: 1,
        primary_repo_id: 'repo-1',
        primary_repo_name: 'openase',
        mirror_count: 2,
        mirror_state: 'ready',
        action: 'none',
      },
    })

    await expect(loadWorkflowRepositoryPrerequisite('project-1')).resolves.toEqual({
      kind: 'ready',
      repoCount: 1,
      primaryRepoId: 'repo-1',
      primaryRepoName: 'openase',
      mirrorCount: 2,
      mirrorState: 'ready',
      action: 'none',
    })
  })
})
