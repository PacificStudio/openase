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
    capabilityCatalog,
    capabilityStateClasses,
    capabilityStateLabel,
  } from '$lib/features/capabilities'
  import { appStore } from '$lib/stores/app.svelte'
  import {
    createEmptyRepositoryDraft,
    parseRepositoryDraft,
    projectRepoToDraft,
    sortProjectRepos,
    type RepositoryDraft,
    type RepositoryEditorMode,
  } from '../repositories-model'
  import RepositoriesList from './repository-list.svelte'
  import RepositoryEditor from './repository-editor.svelte'

  const repositoriesCapability = capabilityCatalog.repositoriesSettings

  let repos = $state<ProjectRepoRecord[]>([])
  let loading = $state(false)
  let saving = $state(false)
  let deleting = $state(false)
  let error = $state('')
  let feedback = $state('')
  let selectedId = $state('')
  let mode = $state<RepositoryEditorMode>('create')
  let draft = $state<RepositoryDraft>(createEmptyRepositoryDraft())

  const selectedRepo = $derived(repos.find((repo) => repo.id === selectedId) ?? null)
  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      repos = []
      selectedId = ''
      mode = 'create'
      draft = createEmptyRepositoryDraft()
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const payload = await listProjectRepos(projectId)
        if (cancelled) return
        applyLoadedRepos(payload.repos)
      } catch (caughtError) {
        if (cancelled) return
        error =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load repositories.'
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

  function applyLoadedRepos(nextRepos: ProjectRepoRecord[]) {
    repos = sortProjectRepos(nextRepos)

    if (repos.length === 0) {
      startCreate({ preserveFeedback: true })
      return
    }

    const nextRepo = repos.find((repo) => repo.id === selectedId) ?? repos[0]
    openRepo(nextRepo, { preserveFeedback: true })
  }

  function openRepo(repo: ProjectRepoRecord, options: { preserveFeedback?: boolean } = {}) {
    mode = 'edit'
    selectedId = repo.id
    draft = projectRepoToDraft(repo)
    error = ''
    if (!options.preserveFeedback) {
      feedback = ''
    }
  }

  function startCreate(options: { preserveFeedback?: boolean } = {}) {
    mode = 'create'
    selectedId = ''
    draft = createEmptyRepositoryDraft({ isPrimary: repos.length === 0 })
    error = ''
    if (!options.preserveFeedback) {
      feedback = ''
    }
  }

  function resetDraft() {
    if (mode === 'create') {
      draft = createEmptyRepositoryDraft({ isPrimary: repos.length === 0 })
    } else if (selectedRepo) {
      draft = projectRepoToDraft(selectedRepo)
    }

    error = ''
    feedback = ''
  }

  async function reloadRepos(projectId: string) {
    const payload = await listProjectRepos(projectId)
    applyLoadedRepos(payload.repos)
  }

  async function handleSave() {
    const projectId = appStore.currentProject?.id
    const parsed = parseRepositoryDraft(draft)
    if (!projectId || !parsed.ok) {
      error = parsed.ok ? 'Project context is unavailable.' : parsed.error
      feedback = ''
      return
    }

    saving = true
    error = ''
    feedback = ''

    try {
      if (mode === 'create') {
        const payload = await createProjectRepo(projectId, parsed.value)
        selectedId = payload.repo.id
        feedback = 'Repository created.'
      } else if (selectedRepo) {
        const payload = await updateProjectRepo(projectId, selectedRepo.id, parsed.value)
        selectedId = payload.repo.id
        feedback = 'Repository updated.'
      }

      await reloadRepos(projectId)
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to save repository.'
      feedback = ''
    } finally {
      saving = false
    }
  }

  async function handleDelete() {
    const projectId = appStore.currentProject?.id
    if (!projectId || !selectedRepo) {
      return
    }

    deleting = true
    error = ''
    feedback = ''

    try {
      await deleteProjectRepo(projectId, selectedRepo.id)
      selectedId = ''
      feedback = 'Repository deleted.'
      await reloadRepos(projectId)
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete repository.'
    } finally {
      deleting = false
    }
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

  <div class="grid gap-6 xl:grid-cols-[20rem_minmax(0,1fr)]">
    <RepositoriesList
      {loading}
      {repos}
      {selectedId}
      onCreate={() => startCreate()}
      onSelect={(repo) => openRepo(repo)}
    />

    <RepositoryEditor
      {mode}
      {selectedRepo}
      {draft}
      reposCount={repos.length}
      {loading}
      {saving}
      {deleting}
      {feedback}
      {error}
      onDraftChange={updateField}
      onSave={() => void handleSave()}
      onDelete={() => void handleDelete()}
      onReset={resetDraft}
    />
  </div>
</div>
