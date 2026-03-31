<script lang="ts">
  import { Separator } from '$ui/separator'
  import RepositoriesList from './repository-list.svelte'
  import RepositoryEditorSheet from './repository-editor-sheet.svelte'
  import { createRepositoriesSettingsState } from './repositories-settings-state.svelte'

  const state = createRepositoriesSettingsState()
</script>

<div class="max-w-2xl space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Repositories</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Manage project repository bindings used for direct remote checkout and ticket workspaces.
    </p>
  </div>

  <Separator />

  <RepositoriesList
    loading={state.ui.loading}
    repos={state.ui.repos}
    selectedId={state.ui.selectedId}
    deletingId={state.ui.deletingId}
    onCreate={() => state.startCreate()}
    onOpenRepo={(repo) => state.openRepo(repo)}
    onDelete={(repo) => void state.deleteFromList(repo)}
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
