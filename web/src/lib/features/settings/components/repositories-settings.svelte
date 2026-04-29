<script lang="ts">
  import { Separator } from '$ui/separator'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import RepositoriesList from './repository-list.svelte'
  import RepositoryEditorSheet from './repository-editor-sheet.svelte'
  import { createRepositoriesSettingsState } from './repositories-settings-state.svelte'

  const state = createRepositoriesSettingsState()
</script>

<div class="space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">
      {i18nStore.t('settings.repositories.heading')}
    </h2>
    <p class="text-muted-foreground mt-1 text-sm">
      {i18nStore.t('settings.repositories.description')}
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
  saving={state.ui.saving}
  githubRepos={state.ui.githubRepos}
  githubRepoQuery={state.ui.githubRepoQuery}
  githubReposLoading={state.ui.githubReposLoading}
  githubReposLoadingMore={state.ui.githubReposLoadingMore}
  githubReposNextCursor={state.ui.githubReposNextCursor}
  githubRepoError={state.ui.githubRepoError}
  githubBindingRepoFullName={state.ui.githubBindingRepoFullName}
  githubNamespaces={state.ui.githubNamespaces}
  githubNamespacesLoading={state.ui.githubNamespacesLoading}
  githubCreateDraft={state.ui.githubCreateDraft}
  githubCreating={state.ui.githubCreating}
  onDraftChange={(field, value) => state.updateField(field, value)}
  onGitHubRepoQueryChange={(value) => state.updateGitHubRepoQuery(value)}
  onGitHubRepoSearch={() => void state.searchGitHubRepos()}
  onGitHubRepoLoadMore={() => void state.loadMoreGitHubRepos()}
  onBindGitHubRepo={(repo) => void state.bindGitHubRepo(repo)}
  onGitHubCreateDraftChange={(field, value) => state.updateGitHubCreateField(field, value)}
  onCreateGitHubRepoAndBind={() => void state.createGitHubRepoAndBind()}
  onSave={() => void state.save()}
/>
