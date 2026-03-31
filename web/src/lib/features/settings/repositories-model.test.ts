import { describe, expect, it } from 'vitest'

import { projectRepoToDraft } from './repositories-model'

describe('repositories model', () => {
  it('tolerates repo payloads that omit labels', () => {
    const draft = projectRepoToDraft({
      id: 'repo-1',
      project_id: 'project-1',
      name: 'TodoApp',
      repository_url: 'https://github.com/BetterAndBetterII/TodoApp.git',
      default_branch: 'main',
    } as unknown as Parameters<typeof projectRepoToDraft>[0])

    expect(draft).toMatchObject({
      name: 'TodoApp',
      repositoryURL: 'https://github.com/BetterAndBetterII/TodoApp.git',
      defaultBranch: 'main',
      labels: '',
    })
  })
})
