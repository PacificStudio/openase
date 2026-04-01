import { ApiError } from '$lib/api/client'
import {
  createGitHubRepository,
  createProjectRepo,
  listGitHubNamespaces,
  listGitHubRepositories,
} from '$lib/api/openase'
import { toastStore } from '$lib/stores/toast.svelte'
import {
  createEmptyGitHubRepositoryCreateDraft,
  githubRepositoryToMutationInput,
  type GitHubRepositoryCreateDraft,
  type GitHubRepositoryRecord,
} from '../repositories-model'
import {
  reloadReposAfterMutation,
  type RepositoryReloadAction,
} from './repositories-settings-feedback'
import type { RepositoriesSettingsUI } from './repositories-settings-ui'

type GitHubActionOptions = {
  ui: RepositoriesSettingsUI
  getProjectId: () => string | undefined
  reloadRepos: (projectId: string) => Promise<void>
  setSelectedRepoId: (repoId: string) => void
  closeEditor: () => void
}

export function createRepositoriesGitHubActions(options: GitHubActionOptions) {
  const { ui, getProjectId, reloadRepos, setSelectedRepoId, closeEditor } = options
  const repositoriesCache = new Map<
    string,
    { repositories: GitHubRepositoryRecord[]; nextCursor: string }
  >()
  let repositoriesRequestSequence = 0

  function buildRepositoriesCacheKey(projectId: string, query: string, cursor: string) {
    return `${projectId}::${query.trim().toLowerCase()}::${cursor.trim()}`
  }

  function reset() {
    ui.githubRepoQuery = ''
    ui.githubRepos = []
    ui.githubReposLoading = false
    ui.githubReposLoadingMore = false
    ui.githubReposNextCursor = ''
    ui.githubRepoError = ''
    ui.githubBindingRepoFullName = ''
    ui.githubNamespaces = []
    ui.githubNamespacesLoading = false
    ui.githubCreateDraft = createEmptyGitHubRepositoryCreateDraft()
    ui.githubCreating = false
    repositoriesCache.clear()
  }

  async function loadNamespaces() {
    const projectId = getProjectId()
    if (!projectId) {
      return
    }

    ui.githubNamespacesLoading = true
    try {
      const payload = await listGitHubNamespaces(projectId)
      ui.githubNamespaces = payload.namespaces
      if (!ui.githubCreateDraft.owner.trim() && payload.namespaces.length > 0) {
        ui.githubCreateDraft = {
          ...ui.githubCreateDraft,
          owner: payload.namespaces[0].login,
        }
      }
    } catch (error) {
      ui.githubNamespaces = []
      reportApiError(error, 'Failed to load GitHub namespaces.')
    } finally {
      ui.githubNamespacesLoading = false
    }
  }

  async function loadRepositories(opts?: { append?: boolean; cursor?: string }) {
    const projectId = getProjectId()
    if (!projectId) {
      return
    }

    const append = opts?.append ?? false
    const cursor = opts?.cursor?.trim() ?? ''
    const cacheKey = buildRepositoriesCacheKey(projectId, ui.githubRepoQuery, cursor)
    const cached = repositoriesCache.get(cacheKey)
    if (cached) {
      ui.githubRepoError = ''
      ui.githubRepos = append ? [...ui.githubRepos, ...cached.repositories] : cached.repositories
      ui.githubReposNextCursor = cached.nextCursor
      ui.githubReposLoading = false
      ui.githubReposLoadingMore = false
      return
    }

    if (append) {
      ui.githubReposLoadingMore = true
    } else {
      ui.githubReposLoading = true
      ui.githubRepoError = ''
    }
    const requestSequence = ++repositoriesRequestSequence

    try {
      const payload = await listGitHubRepositories(projectId, {
        query: ui.githubRepoQuery,
        cursor: cursor || undefined,
      })
      if (requestSequence !== repositoriesRequestSequence) {
        return
      }
      repositoriesCache.set(cacheKey, {
        repositories: payload.repositories,
        nextCursor: payload.next_cursor,
      })
      ui.githubRepos = append ? [...ui.githubRepos, ...payload.repositories] : payload.repositories
      ui.githubReposNextCursor = payload.next_cursor
    } catch (error) {
      if (requestSequence !== repositoriesRequestSequence) {
        return
      }
      if (!append) {
        ui.githubRepos = []
      }
      ui.githubReposNextCursor = ''
      ui.githubRepoError =
        error instanceof ApiError ? error.detail : 'Failed to load GitHub repositories.'
    } finally {
      if (requestSequence === repositoriesRequestSequence) {
        ui.githubReposLoading = false
        ui.githubReposLoadingMore = false
      }
    }
  }

  async function bindRepository(repo: GitHubRepositoryRecord) {
    const projectId = getProjectId()
    if (!projectId) {
      toastStore.error('Project context is unavailable.')
      return
    }

    ui.githubBindingRepoFullName = repo.full_name
    try {
      const payload = await createProjectRepo(projectId, githubRepositoryToMutationInput(repo))
      setSelectedRepoId(payload.repo.id)
      closeEditor()
      await reportMutation(projectId, 'created', 'Repository created.')
    } catch (error) {
      reportApiError(error, 'Failed to bind GitHub repository.')
    } finally {
      ui.githubBindingRepoFullName = ''
    }
  }

  async function createAndBindRepository() {
    const projectId = getProjectId()
    if (!projectId) {
      toastStore.error('Project context is unavailable.')
      return
    }

    const owner = ui.githubCreateDraft.owner.trim()
    const name = ui.githubCreateDraft.name.trim()
    if (!owner) {
      toastStore.error('Select a GitHub namespace before creating a repository.')
      return
    }
    if (!name) {
      toastStore.error('Repository name is required.')
      return
    }

    ui.githubCreating = true
    try {
      const created = await createGitHubRepository(projectId, {
        owner,
        name,
        description: ui.githubCreateDraft.description.trim(),
        visibility: ui.githubCreateDraft.visibility,
        auto_init: true,
      })
      const payload = await createProjectRepo(
        projectId,
        githubRepositoryToMutationInput(created.repository),
      )
      repositoriesCache.clear()
      setSelectedRepoId(payload.repo.id)
      closeEditor()
      await reportMutation(projectId, 'created', 'Repository created.')
    } catch (error) {
      reportApiError(error, 'Failed to create GitHub repository.')
    } finally {
      ui.githubCreating = false
    }
  }

  function updateCreateDraft(field: keyof GitHubRepositoryCreateDraft, value: string) {
    ui.githubCreateDraft = { ...ui.githubCreateDraft, [field]: value }
  }

  return {
    reset,
    loadNamespaces,
    loadRepositories,
    bindRepository,
    createAndBindRepository,
    updateCreateDraft,
  }

  async function reportMutation(
    projectId: string,
    action: Exclude<RepositoryReloadAction, 'deleted' | 'mirror_updated'>,
    message: string,
  ) {
    await reloadReposAfterMutation(() => reloadRepos(projectId), action, message)
  }
}

function reportApiError(error: unknown, fallback: string) {
  toastStore.error(error instanceof ApiError ? error.detail : fallback)
}
