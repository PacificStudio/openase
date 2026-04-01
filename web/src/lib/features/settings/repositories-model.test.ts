import { describe, expect, it } from 'vitest'

import { githubRepositoryToMutationInput, projectRepoToDraft } from './repositories-model'

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

  it('maps GitHub repositories into repository mutation input defaults', () => {
    expect(
      githubRepositoryToMutationInput({
        id: 42,
        name: 'backend',
        full_name: 'acme/backend',
        owner: 'acme',
        default_branch: 'develop',
        visibility: 'private',
        private: true,
        html_url: 'https://github.com/acme/backend',
        clone_url: 'https://github.com/acme/backend.git',
      }),
    ).toEqual({
      name: 'backend',
      repository_url: 'https://github.com/acme/backend.git',
      default_branch: 'develop',
      workspace_dirname: null,
      labels: [],
    })
  })
})
