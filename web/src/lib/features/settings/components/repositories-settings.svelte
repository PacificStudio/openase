<script lang="ts">
  import { Separator } from '$ui/separator'
  import RepositoriesList from './repository-list.svelte'
  import RepositoryEditorSheet from './repository-editor-sheet.svelte'
  import RepositoryMirrorDialog from './repository-mirror-dialog.svelte'
  import { createRepositoriesSettingsState } from './repositories-settings-state.svelte'

  const state = createRepositoriesSettingsState()
</script>

<div class="max-w-2xl space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Repositories</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Manage project repository bindings and mirror configuration.
    </p>
  </div>

  <Separator />

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
    onMaterialize={(repo) => void state.runMirrorAction(repo)}
    onConfigureMirror={(repo) => state.openMirrorDialog(repo)}
  />
</div>

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
