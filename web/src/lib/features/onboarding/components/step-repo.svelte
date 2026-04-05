<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import {
    createGitHubRepository,
    createProjectRepo,
    listGitHubNamespaces,
    listGitHubRepositories,
    listProjectRepos,
  } from '$lib/api/openase'
  import type {
    GitHubRepositoryNamespaceRecord,
    GitHubRepositoryRecord,
    ProjectRepoRecord,
  } from '$lib/api/contracts'
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { RepoState } from '../types'
  import StepRepoContent from './step-repo-content.svelte'

  let {
    projectId,
    initialState,
    onComplete,
  }: {
    projectId: string
    initialState: RepoState
    onComplete: (repos: ProjectRepoRecord[]) => void
  } = $props()

  let mode = $state<'choose' | 'create' | 'link'>('choose')
  let creating = $state(false)
  let linking = $state(false)
  let searchingRepos = $state(false)

  // Create new repo fields
  let newRepoName = $state('')
  let newRepoVisibility = $state<'private' | 'public'>('private')
  let newRepoDefaultBranch = $state('main')
  let selectedNamespace = $state('')

  // Link existing repo fields
  let repoSearchQuery = $state('')
  let searchResults = $state<GitHubRepositoryRecord[]>([])
  let linkRepoUrl = $state('')
  let linkRepoName = $state('')
  let linkRepoBranch = $state('main')

  let namespaces = $state<GitHubRepositoryNamespaceRecord[]>([
    ...untrack(() => initialState.namespaces),
  ])
  let repos = $state<ProjectRepoRecord[]>([...untrack(() => initialState.repos)])

  const hasRepos = $derived(repos.length > 0)

  $effect(() => {
    if (namespaces.length > 0 && !selectedNamespace) {
      selectedNamespace = namespaces[0]?.login ?? ''
    }
  })

  $effect(() => {
    if (repos.length === 0 && initialState.repos.length > 0) {
      repos = [...initialState.repos]
    }
    if (namespaces.length === 0 && initialState.namespaces.length > 0) {
      namespaces = [...initialState.namespaces]
    }
  })

  async function loadNamespaces() {
    try {
      const payload = await listGitHubNamespaces(projectId)
      namespaces = payload.namespaces
    } catch {
      // ignore
    }
  }

  async function loadBrowsableRepositories(query?: string) {
    searchingRepos = true
    try {
      const payload = await listGitHubRepositories(projectId, {
        query: query?.trim() || undefined,
      })
      searchResults = payload.repositories
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to search repositories.',
      )
    } finally {
      searchingRepos = false
    }
  }

  async function handleSearchRepos() {
    await loadBrowsableRepositories(repoSearchQuery)
  }

  function selectSearchResult(repo: GitHubRepositoryRecord) {
    linkRepoUrl = repo.clone_url || repo.html_url || ''
    linkRepoName = repo.name
    linkRepoBranch = repo.default_branch || 'main'
  }

  async function handleCreateRepo() {
    if (!newRepoName.trim() || !selectedNamespace) return
    creating = true
    try {
      const ghResult = await createGitHubRepository(projectId, {
        owner: selectedNamespace,
        name: newRepoName.trim(),
        visibility: newRepoVisibility,
        auto_init: true,
      })

      await createProjectRepo(projectId, {
        name: newRepoName.trim(),
        repository_url: ghResult.repository.clone_url || ghResult.repository.html_url,
        default_branch: newRepoDefaultBranch || 'main',
      })

      const updatedRepos = await listProjectRepos(projectId)
      repos = updatedRepos.repos
      toastStore.success(`Repository ${newRepoName} was created and linked to the project.`)
      onComplete(updatedRepos.repos)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create the repository.',
      )
    } finally {
      creating = false
    }
  }

  async function handleLinkRepo() {
    if (!linkRepoUrl.trim() || !linkRepoName.trim()) return
    linking = true
    try {
      await createProjectRepo(projectId, {
        name: linkRepoName.trim(),
        repository_url: linkRepoUrl.trim(),
        default_branch: linkRepoBranch || 'main',
      })

      const updatedRepos = await listProjectRepos(projectId)
      repos = updatedRepos.repos
      toastStore.success(`Repository ${linkRepoName} was linked to the project.`)
      onComplete(updatedRepos.repos)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to link the repository.',
      )
    } finally {
      linking = false
    }
  }

  function enterCreateMode() {
    mode = 'create'
    void loadNamespaces()
  }

  function enterLinkMode() {
    mode = 'link'
    void loadBrowsableRepositories()
  }
</script>

<StepRepoContent
  bind:mode
  bind:newRepoName
  bind:newRepoVisibility
  bind:newRepoDefaultBranch
  bind:selectedNamespace
  bind:repoSearchQuery
  bind:linkRepoUrl
  bind:linkRepoName
  bind:linkRepoBranch
  {creating}
  {linking}
  {searchingRepos}
  {searchResults}
  {namespaces}
  {repos}
  {hasRepos}
  onEnterCreateMode={enterCreateMode}
  onEnterLinkMode={enterLinkMode}
  onSearchRepos={handleSearchRepos}
  onSelectSearchResult={selectSearchResult}
  onCreateRepo={handleCreateRepo}
  onLinkRepo={handleLinkRepo}
/>
