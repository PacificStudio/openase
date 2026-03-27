import { describe, expect, it, vi } from 'vitest'

const { listProjectRepos } = vi.hoisted(() => ({
  listProjectRepos: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  listProjectRepos,
}))

import { loadWorkflowRepositoryPrerequisite } from './data'

describe('workflow repository prerequisite', () => {
  it('requires a primary repository when the project has no repos', async () => {
    listProjectRepos.mockResolvedValue({
      repos: [],
    })

    await expect(loadWorkflowRepositoryPrerequisite('project-1')).resolves.toEqual({
      kind: 'missing_primary_repo',
      repoCount: 0,
    })
  })

  it('accepts a project once a primary repository exists', async () => {
    listProjectRepos.mockResolvedValue({
      repos: [
        {
          id: 'repo-1',
          name: 'openase',
          is_primary: true,
        },
      ],
    })

    await expect(loadWorkflowRepositoryPrerequisite('project-1')).resolves.toEqual({
      kind: 'ready',
      primaryRepoId: 'repo-1',
      primaryRepoName: 'openase',
      repoCount: 1,
    })
  })
})
