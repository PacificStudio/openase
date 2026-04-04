import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import type {
  GitHubRepositoryNamespaceRecord,
  GitHubRepositoryRecord,
  ProjectRepoRecord,
} from '$lib/api/contracts'
import StepRepo from './step-repo.svelte'

const {
  createGitHubRepository,
  createProjectRepo,
  listGitHubNamespaces,
  listGitHubRepositories,
  listProjectRepos,
} = vi.hoisted(() => ({
  createGitHubRepository: vi.fn(),
  createProjectRepo: vi.fn(),
  listGitHubNamespaces: vi.fn(),
  listGitHubRepositories: vi.fn(),
  listProjectRepos: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/api/openase', () => ({
  createGitHubRepository,
  createProjectRepo,
  listGitHubNamespaces,
  listGitHubRepositories,
  listProjectRepos,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('StepRepo', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
  })

  beforeEach(() => {
    listGitHubNamespaces.mockResolvedValue({
      namespaces: [makeNamespace({ login: 'octo-org', kind: 'organization' })],
    })
    listGitHubRepositories.mockResolvedValue({
      repositories: [makeGitHubRepo()],
      next_cursor: '',
    })
    listProjectRepos.mockResolvedValue({ repos: [makeProjectRepo()] })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('loads namespaces when entering create mode from an empty onboarding state', async () => {
    const { getByText } = render(StepRepo, {
      props: {
        projectId: 'project-1',
        initialState: {
          repos: [],
          namespaces: [],
        },
        onComplete: vi.fn(),
      },
    })

    await fireEvent.click(getByText('创建新仓库'))

    await waitFor(() => {
      expect(listGitHubNamespaces).toHaveBeenCalledWith('project-1')
    })
  })

  it('prefetches browseable repositories when entering link mode and still supports query search', async () => {
    const repoA = makeGitHubRepo({
      name: 'alpha-app',
      full_name: 'octo-org/alpha-app',
      clone_url: 'https://github.com/octo-org/alpha-app.git',
    })
    const repoB = makeGitHubRepo({
      name: 'beta-service',
      full_name: 'octo-org/beta-service',
      clone_url: 'https://github.com/octo-org/beta-service.git',
      default_branch: 'develop',
    })

    listGitHubRepositories
      .mockResolvedValueOnce({
        repositories: [repoA],
        next_cursor: '',
      })
      .mockResolvedValueOnce({
        repositories: [repoB],
        next_cursor: '',
      })

    const { getByText, getByPlaceholderText, findByText } = render(StepRepo, {
      props: {
        projectId: 'project-1',
        initialState: {
          repos: [],
          namespaces: [],
        },
        onComplete: vi.fn(),
      },
    })

    await fireEvent.click(getByText('关联已有仓库'))

    await waitFor(() => {
      expect(listGitHubRepositories).toHaveBeenCalledWith('project-1', {
        query: undefined,
      })
    })
    expect(await findByText('octo-org/alpha-app')).toBeTruthy()

    const searchInput = getByPlaceholderText('搜索仓库名称，或直接浏览最近可访问仓库...')
    await fireEvent.input(searchInput, { target: { value: 'beta' } })
    await fireEvent.keyDown(searchInput, { key: 'Enter' })

    await waitFor(() => {
      expect(listGitHubRepositories).toHaveBeenLastCalledWith('project-1', {
        query: 'beta',
      })
    })
    expect(await findByText('octo-org/beta-service')).toBeTruthy()
  })
})

function makeNamespace(
  overrides: Partial<GitHubRepositoryNamespaceRecord> = {},
): GitHubRepositoryNamespaceRecord {
  return {
    login: 'octocat',
    kind: 'user',
    ...overrides,
  }
}

function makeGitHubRepo(overrides: Partial<GitHubRepositoryRecord> = {}): GitHubRepositoryRecord {
  return {
    id: 1,
    name: 'openase',
    full_name: 'octo-org/openase',
    owner: 'octo-org',
    default_branch: 'main',
    visibility: 'private',
    private: true,
    html_url: 'https://github.com/octo-org/openase',
    clone_url: 'https://github.com/octo-org/openase.git',
    ...overrides,
  }
}

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
