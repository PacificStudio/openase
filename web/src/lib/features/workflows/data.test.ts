import { describe, expect, it, vi } from 'vitest'

const { getWorkflowRepositoryPrerequisite } = vi.hoisted(() => ({
  getWorkflowRepositoryPrerequisite: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  getWorkflowRepositoryPrerequisite,
}))

import { loadWorkflowRepositoryPrerequisite } from './data'

describe('workflow repository prerequisite', () => {
  it('returns repo counts without primary semantics when the project has no repos', async () => {
    getWorkflowRepositoryPrerequisite.mockResolvedValue({
      prerequisite: {
        kind: 'ready',
        repo_count: 0,
        action: 'none',
      },
    })

    await expect(loadWorkflowRepositoryPrerequisite('project-1')).resolves.toEqual({
      kind: 'ready',
      repoCount: 0,
      action: 'none',
    })
  })

  it('returns repo counts for multi-repo projects without mirror-specific payload', async () => {
    getWorkflowRepositoryPrerequisite.mockResolvedValue({
      prerequisite: {
        kind: 'ready',
        repo_count: 2,
        action: 'none',
      },
    })

    await expect(loadWorkflowRepositoryPrerequisite('project-1')).resolves.toEqual({
      kind: 'ready',
      repoCount: 2,
      action: 'none',
    })
  })
})
