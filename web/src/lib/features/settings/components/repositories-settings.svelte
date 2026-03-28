<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import {
    createProjectRepo,
    deleteProjectRepo,
    listProjectRepos,
    updateProjectRepo,
  } from '$lib/api/openase'
  import {
    getSettingsSectionCapability,
    capabilityStateClasses,
    capabilityStateLabel,
  } from '$lib/features/capabilities'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import {
    createEmptyRepositoryDraft,
    parseRepositoryDraft,
    projectRepoToDraft,
    sortProjectRepos,
    type RepositoryDraft,
    type RepositoryEditorMode,
  } from '../repositories-model'
  import RepositoriesList from './repository-list.svelte'
  import RepositoryEditorSheet from './repository-editor-sheet.svelte'

  const repositoriesCapability = getSettingsSectionCapability('repositories')
  let {} = $props()

  let repos = $state<ProjectRepoRecord[]>([])
  let loading = $state(false)
  let saving = $state(false)
  let deletingId = $state('')
  let editorOpen = $state(false)
  let selectedId = $state('')
  let mode = $state<RepositoryEditorMode>('create')
  let draft = $state<RepositoryDraft>(createEmptyRepositoryDraft())

  const selectedRepo = $derived(repos.find((repo) => repo.id === selectedId) ?? null)
  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      repos = []
      selectedId = ''
      editorOpen = false
      mode = 'create'
      draft = createEmptyRepositoryDraft()
      return
    }

    let cancelled = false
    const load = async () => {
      loading = true
      try {
        const payload = await listProjectRepos(projectId)
        if (cancelled) return
        syncLoadedRepos(payload.repos)
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  function syncLoadedRepos(nextRepos: ProjectRepoRecord[]) {
    const sortedRepos = sortProjectRepos(nextRepos)
    repos = sortedRepos

    if (sortedRepos.length === 0) {
      selectedId = ''
      editorOpen = false
      mode = 'create'
      draft = createEmptyRepositoryDraft({ isPrimary: true })
      return
    }

    if (selectedId && !sortedRepos.some((repo) => repo.id === selectedId)) {
      selectedId = ''
      editorOpen = false
      mode = 'create'
      draft = createEmptyRepositoryDraft({ isPrimary: false })
    }
  }

  function openRepo(repo: ProjectRepoRecord) {
    mode = 'edit'
    selectedId = repo.id
    draft = projectRepoToDraft(repo)
    editorOpen = true
  }

  function startCreate() {
    mode = 'create'
    selectedId = ''
    draft = createEmptyRepositoryDraft({ isPrimary: repos.length === 0 })
    editorOpen = true
  }

  async function reloadRepos(projectId: string) {
    const payload = await listProjectRepos(projectId)
    syncLoadedRepos(payload.repos)
  }

  function reloadFailureMessage(action: 'created' | 'updated' | 'deleted', detail?: string) {
    const message = `Repository ${action}, but reloading the repository list failed.`
    return detail ? `${message} ${detail}` : message
  }

  async function reloadReposAfterMutation(
    projectId: string,
    action: 'created' | 'updated' | 'deleted',
    successMessage: string,
  ) {
    try {
      await reloadRepos(projectId)
      toastStore.success(successMessage)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? reloadFailureMessage(action, caughtError.detail)
          : reloadFailureMessage(action),
      )
    }
  }

  async function handleSave() {
    const projectId = appStore.currentProject?.id
    const parsed = parseRepositoryDraft(draft)
    if (!projectId || !parsed.ok) {
      toastStore.error(parsed.ok ? 'Project context is unavailable.' : parsed.error)
      return
    }

    saving = true

    try {
      let successAction: 'created' | 'updated'

      if (mode === 'create') {
        const payload = await createProjectRepo(projectId, parsed.value)
        selectedId = payload.repo.id
        successAction = 'created'
      } else if (selectedRepo) {
        const payload = await updateProjectRepo(projectId, selectedRepo.id, parsed.value)
        selectedId = payload.repo.id
        successAction = 'updated'
      } else {
        return
      }

      await reloadReposAfterMutation(projectId, successAction, `Repository ${successAction}.`)
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save repository.',
      )
    } finally {
      saving = false
    }
  }

  async function handleDelete(targetRepo: ProjectRepoRecord | null = selectedRepo) {
    const projectId = appStore.currentProject?.id
    if (!projectId || !targetRepo) {
      return
    }

    deletingId = targetRepo.id

    try {
      await deleteProjectRepo(projectId, targetRepo.id)
      if (selectedId === targetRepo.id) {
        selectedId = ''
        editorOpen = false
      }
      await reloadReposAfterMutation(projectId, 'deleted', 'Repository deleted.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete repository.',
      )
    } finally {
      deletingId = ''
    }
  }

  async function handleDeleteRepo(repo: ProjectRepoRecord) {
    if (repo.id !== selectedId) {
      selectedId = repo.id
      mode = 'edit'
      draft = projectRepoToDraft(repo)
    }

    await handleDelete(repo)
  }

  function updateField(field: keyof RepositoryDraft, value: string | boolean) {
    draft = { ...draft, [field]: value }
  }
</script>

<div class="space-y-6">
  <div>
    <div class="flex items-center gap-2">
      <h2 class="text-foreground text-base font-semibold">Repositories</h2>
      <span
        class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(repositoriesCapability.state)}`}
      >
        {capabilityStateLabel(repositoriesCapability.state)}
      </span>
    </div>
    <p class="text-muted-foreground mt-1 max-w-3xl text-sm">{repositoriesCapability.summary}</p>
  </div>

  <RepositoriesList
    {loading}
    {repos}
    {selectedId}
    {deletingId}
    onCreate={startCreate}
    onOpenRepo={openRepo}
    onDelete={(repo) => void handleDeleteRepo(repo)}
  />

  <RepositoryEditorSheet
    bind:open={editorOpen}
    {mode}
    {selectedRepo}
    {draft}
    reposCount={repos.length}
    {saving}
    onDraftChange={updateField}
    onSave={() => void handleSave()}
  />
</div>
