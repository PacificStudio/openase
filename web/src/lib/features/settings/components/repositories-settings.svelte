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
  import {
    derivePrimaryRepositoryReadiness,
    formatMirrorTimestamp,
    repositoryMirrorToneClasses,
  } from '../repositories-readiness'
  import RepositoriesList from './repository-list.svelte'
  import RepositoryEditorSheet from './repository-editor-sheet.svelte'

  const repositoriesCapability = getSettingsSectionCapability('repositories')

  let repos = $state<ProjectRepoRecord[]>([])
  let loading = $state(false)
  let saving = $state(false)
  let deletingId = $state('')
  let editorOpen = $state(false)
  let selectedId = $state('')
  let mode = $state<RepositoryEditorMode>('create')
  let draft = $state<RepositoryDraft>(createEmptyRepositoryDraft())

  const selectedRepo = $derived(repos.find((repo) => repo.id === selectedId) ?? null)
  const primaryReadiness = $derived(derivePrimaryRepositoryReadiness(repos))
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

  {#if repos.length > 0}
    <section
      class={`rounded-2xl border px-4 py-4 ${repositoryMirrorToneClasses(primaryReadiness.kind === 'missing_primary_repo' ? 'missing' : primaryReadiness.mirrorState)}`}
    >
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="space-y-1">
          <h3 class="text-sm font-semibold">
            {#if primaryReadiness.kind === 'missing_primary_repo'}
              Primary repository not configured
            {:else if primaryReadiness.kind === 'ready'}
              Primary mirror ready
            {:else}
              Primary mirror needs attention
            {/if}
          </h3>
          <p class="text-sm">
            {#if primaryReadiness.kind === 'missing_primary_repo'}
              This project has repository bindings, but none of them is marked primary. Mark one
              repository as primary before configuring workflows or harness files.
            {:else if primaryReadiness.action === 'prepare_mirror'}
              <span class="font-medium">{primaryReadiness.primaryRepoName}</span> is bound as the primary
              repository, but no mirror is ready yet. Prepare a mirror on the target machine before editing
              workflows or harness files.
            {:else if primaryReadiness.action === 'wait_for_mirror'}
              <span class="font-medium">{primaryReadiness.primaryRepoName}</span> is bound as the
              primary repository, but its mirror is currently
              <span class="font-medium">{primaryReadiness.mirrorState}</span>. Wait for the mirror
              lifecycle to finish before continuing.
            {:else if primaryReadiness.action === 'sync_mirror'}
              <span class="font-medium">{primaryReadiness.primaryRepoName}</span> is bound as the
              primary repository, but its mirror is
              <span class="font-medium">{primaryReadiness.mirrorState}</span>. Repair or resync the
              mirror before using it for workflows.
            {:else}
              <span class="font-medium">{primaryReadiness.primaryRepoName}</span> has a ready primary
              mirror for workflow and harness operations.
            {/if}
          </p>
        </div>

        {#if primaryReadiness.kind !== 'missing_primary_repo'}
          <dl class="grid gap-x-4 gap-y-2 text-xs sm:grid-cols-2">
            <div>
              <dt class="opacity-70">Mirror state</dt>
              <dd class="font-medium">{primaryReadiness.mirrorState}</dd>
            </div>
            <div>
              <dt class="opacity-70">Known mirrors</dt>
              <dd class="font-medium">{primaryReadiness.mirrorCount}</dd>
            </div>
            {#if primaryReadiness.mirrorMachineId}
              <div>
                <dt class="opacity-70">Target machine</dt>
                <dd class="font-medium break-all">{primaryReadiness.mirrorMachineId}</dd>
              </div>
            {/if}
            {#if formatMirrorTimestamp(primaryReadiness.lastSyncedAt)}
              <div>
                <dt class="opacity-70">Last synced</dt>
                <dd class="font-medium">{formatMirrorTimestamp(primaryReadiness.lastSyncedAt)}</dd>
              </div>
            {/if}
            {#if formatMirrorTimestamp(primaryReadiness.lastVerifiedAt)}
              <div>
                <dt class="opacity-70">Last verified</dt>
                <dd class="font-medium">
                  {formatMirrorTimestamp(primaryReadiness.lastVerifiedAt)}
                </dd>
              </div>
            {/if}
          </dl>
        {/if}
      </div>

      {#if primaryReadiness.kind !== 'missing_primary_repo' && primaryReadiness.lastError}
        <div class="bg-background/70 mt-3 rounded-xl border border-current/20 px-3 py-2 text-sm">
          <p class="font-medium">Last mirror error</p>
          <p class="mt-1 break-words opacity-80">{primaryReadiness.lastError}</p>
        </div>
      {/if}
    </section>
  {/if}

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
