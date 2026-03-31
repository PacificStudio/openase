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
        action: 'bind_primary_repo',
      },
    })

    await expect(loadWorkflowRepositoryPrerequisite('project-1')).resolves.toEqual({
      kind: 'missing_primary_repo',
      repoCount: 0,
      action: 'bind_primary_repo',
    })
  })

  it('treats unknown prerequisite kinds as missing primary repo', async () => {
    getWorkflowRepositoryPrerequisite.mockResolvedValue({
      prerequisite: {
        kind: 'unexpected',
        repo_count: 1,
        primary_repo_id: 'repo-1',
        primary_repo_name: 'openase',
        action: 'none',
      },
    })

    await expect(loadWorkflowRepositoryPrerequisite('project-1')).resolves.toEqual({
      kind: 'missing_primary_repo',
      repoCount: 1,
      action: 'bind_primary_repo',
    })
  })

  it('accepts a project once a primary repo is bound', async () => {
    getWorkflowRepositoryPrerequisite.mockResolvedValue({
      prerequisite: {
        kind: 'ready',
        repo_count: 1,
        primary_repo_id: 'repo-1',
        primary_repo_name: 'openase',
        action: 'none',
      },
    })

    await expect(loadWorkflowRepositoryPrerequisite('project-1')).resolves.toEqual({
      kind: 'ready',
      repoCount: 1,
      primaryRepoId: 'repo-1',
      primaryRepoName: 'openase',
      action: 'none',
    })
  })
})
