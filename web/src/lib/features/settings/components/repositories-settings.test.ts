import { cleanup, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import RepositoryGitHubBrowser from './repository-github-browser.svelte'
import RepositoriesSettingsStateTestHost from './repositories-settings-state-test-host.svelte'

const {
  createGitHubRepository,
  createProjectRepo,
  deleteProjectRepo,
  listGitHubNamespaces,
  listGitHubRepositories,
  listProjectRepos,
  updateProjectRepo,
} = vi.hoisted(() => ({
  createGitHubRepository: vi.fn(),
  createProjectRepo: vi.fn(),
  deleteProjectRepo: vi.fn(),
  listGitHubNamespaces: vi.fn(),
  listGitHubRepositories: vi.fn(),
  listProjectRepos: vi.fn(),
  updateProjectRepo: vi.fn(),
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
  deleteProjectRepo,
  listGitHubNamespaces,
  listGitHubRepositories,
  listProjectRepos,
  updateProjectRepo,
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

function buildRepoResponse(query: string) {
  const label = query.trim() || 'default'
  return {
    repositories: [
      {
        id: label.length + 1,
        name: `${label}-repo`,
        full_name: `acme/${label}-repo`,
        owner: 'acme',
        default_branch: 'main',
        visibility: 'private',
        private: true,
        html_url: `https://github.com/acme/${label}-repo`,
        clone_url: `https://github.com/acme/${label}-repo.git`,
      },
    ],
    next_cursor: '',
  }
}

async function flushPromises() {
  await Promise.resolve()
  await Promise.resolve()
}

describe('Repositories settings', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    seedProject()
    listProjectRepos.mockResolvedValue({ repos: [] })
    listGitHubNamespaces.mockResolvedValue({
      namespaces: [{ login: 'acme', kind: 'organization' }],
    })
    listGitHubRepositories.mockImplementation(async (_projectId, opts) =>
      buildRepoResponse(opts?.query ?? ''),
    )
  })

  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('debounces automatic GitHub repository search and reuses cached results for repeated queries', async () => {
    let state: ReturnType<
      typeof import('./repositories-settings-state.svelte').createRepositoriesSettingsState
    > | null = null

    render(RepositoriesSettingsStateTestHost, {
      props: {
        onReady: (nextState) => {
          state = nextState
        },
      },
    })
    await flushPromises()
    expect(state).toBeTruthy()

    state!.startCreate()
    await flushPromises()

    expect(listGitHubNamespaces).toHaveBeenCalledTimes(1)
    expect(listGitHubRepositories).toHaveBeenCalledTimes(1)
    expect(listGitHubRepositories).toHaveBeenCalledWith('project-1', {
      query: '',
      cursor: undefined,
    })

    state!.updateGitHubRepoQuery('back')
    await vi.advanceTimersByTimeAsync(150)
    expect(listGitHubRepositories).toHaveBeenCalledTimes(1)

    state!.updateGitHubRepoQuery('backend')
    await vi.advanceTimersByTimeAsync(299)
    expect(listGitHubRepositories).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(1)
    await flushPromises()
    expect(listGitHubRepositories).toHaveBeenCalledTimes(2)
    expect(listGitHubRepositories).toHaveBeenLastCalledWith('project-1', {
      query: 'backend',
      cursor: undefined,
    })

    state!.updateGitHubRepoQuery('frontend')
    await vi.advanceTimersByTimeAsync(300)
    await flushPromises()
    expect(listGitHubRepositories).toHaveBeenCalledTimes(3)
    expect(listGitHubRepositories).toHaveBeenLastCalledWith('project-1', {
      query: 'frontend',
      cursor: undefined,
    })

    state!.updateGitHubRepoQuery('backend')
    await vi.advanceTimersByTimeAsync(300)
    await flushPromises()
    expect(listGitHubRepositories).toHaveBeenCalledTimes(3)
  })

  it('renders the GitHub browser without a manual search button', () => {
    const { queryByRole } = render(RepositoryGitHubBrowser, {
      props: {
        repos: [],
        query: '',
        loading: false,
      },
    })

    expect(queryByRole('button', { name: 'Search' })).toBeNull()
  })
})
