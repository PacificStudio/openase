<script lang="ts">
  import {
    getSettingsSectionCapability,
    capabilityStateClasses,
    capabilityStateLabel,
  } from '$lib/features/capabilities'
  import RepositoriesList from './repository-list.svelte'
  import RepositoryEditorSheet from './repository-editor-sheet.svelte'
  import RepositoryMirrorDialog from './repository-mirror-dialog.svelte'
  import RepositoryReadinessBanner from './repository-readiness-banner.svelte'
  import { createRepositoriesSettingsState } from './repositories-settings-state.svelte'

  const repositoriesCapability = getSettingsSectionCapability('repositories')
  const state = createRepositoriesSettingsState()

  function openPrimaryMirror() {
    if (state.selectedPrimaryRepo) {
      state.openMirrorDialog(state.selectedPrimaryRepo)
    }
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

  {#if state.ui.repos.length > 0}
    <RepositoryReadinessBanner
      readiness={state.primaryReadiness}
      mirrorActionLabel={state.primaryMirrorActionLabel}
      onOpenPrimaryMirror={state.primaryReadiness.kind === 'primary_mirror_not_ready'
        ? openPrimaryMirror
        : undefined}
    />
  {/if}

  <RepositoriesList
    loading={state.ui.loading}
    repos={state.ui.repos}
    selectedId={state.ui.selectedId}
    deletingId={state.ui.deletingId}
    materializingId={state.ui.materializingId}
    mirrorActionLabelByRepoId={state.mirrorActionLabelByRepoId}
    onCreate={() => state.startCreate()}
    onOpenRepo={(repo) => state.openRepo(repo)}
    onDelete={(repo) => void state.deleteFromList(repo)}
    onMaterialize={(repo) => state.openMirrorDialog(repo)}
  />

  <RepositoryEditorSheet
    bind:open={state.ui.editorOpen}
    mode={state.ui.mode}
    selectedRepo={state.selectedRepo}
    draft={state.ui.draft}
    reposCount={state.ui.repos.length}
    saving={state.ui.saving}
    onDraftChange={(field, value) => state.updateField(field, value)}
    onSave={() => void state.save()}
  />

  <RepositoryMirrorDialog
    bind:open={state.ui.mirrorDialogOpen}
    repo={state.selectedMirrorRepo}
    draft={state.ui.mirrorDraft}
    machines={state.ui.machines}
    saving={state.ui.materializingId !== ''}
    errorMessage={state.ui.mirrorErrorMessage}
    title={state.selectedMirrorContext.dialogTitle}
    submitLabel={state.selectedMirrorContext.submitLabel}
    onDraftChange={(field, value) => state.updateMirrorField(field, value)}
    onSubmit={() => void state.materializeMirror()}
  />
</div>
