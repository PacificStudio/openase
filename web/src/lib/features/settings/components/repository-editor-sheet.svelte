<script lang="ts">
  import { Button } from '$ui/button'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import RepositoryEditor from './repository-editor.svelte'
  import type {
    GitHubRepositoryCreateDraft,
    GitHubRepositoryNamespace,
    GitHubRepositoryRecord,
    RepositoryDraft,
    RepositoryEditorMode,
  } from '../repositories-model'

  let {
    open = $bindable(false),
    mode,
    selectedRepo,
    draft,
    reposCount = 0,
    saving = false,
    githubRepos = [],
    githubRepoQuery = '',
    githubReposLoading = false,
    githubReposLoadingMore = false,
    githubReposNextCursor = '',
    githubRepoError = '',
    githubBindingRepoFullName = '',
    githubNamespaces = [],
    githubNamespacesLoading = false,
    githubCreateDraft,
    githubCreating = false,
    onDraftChange,
    onGitHubRepoQueryChange,
    onGitHubRepoSearch,
    onGitHubRepoLoadMore,
    onBindGitHubRepo,
    onGitHubCreateDraftChange,
    onCreateGitHubRepoAndBind,
    onSave,
  }: {
    open?: boolean
    mode: RepositoryEditorMode
    selectedRepo: ProjectRepoRecord | null
    draft: RepositoryDraft
    reposCount?: number
    saving?: boolean
    githubRepos?: GitHubRepositoryRecord[]
    githubRepoQuery?: string
    githubReposLoading?: boolean
    githubReposLoadingMore?: boolean
    githubReposNextCursor?: string
    githubRepoError?: string
    githubBindingRepoFullName?: string
    githubNamespaces?: GitHubRepositoryNamespace[]
    githubNamespacesLoading?: boolean
    githubCreateDraft: GitHubRepositoryCreateDraft
    githubCreating?: boolean
    onDraftChange?: (field: keyof RepositoryDraft, value: string | boolean) => void
    onGitHubRepoQueryChange?: (value: string) => void
    onGitHubRepoSearch?: () => void
    onGitHubRepoLoadMore?: () => void
    onBindGitHubRepo?: (repo: GitHubRepositoryRecord) => void
    onGitHubCreateDraftChange?: (field: keyof GitHubRepositoryCreateDraft, value: string) => void
    onCreateGitHubRepoAndBind?: () => void
    onSave?: () => void
  } = $props()
</script>

<Sheet bind:open>
  <SheetContent
    side="right"
    class="flex w-full flex-col gap-0 p-0 sm:max-w-xl"
    data-testid="repository-editor-sheet"
  >
    <SheetHeader class="border-border border-b px-6 py-5 text-left">
      <div class="flex items-start justify-between gap-4 pr-10">
        <div class="min-w-0 space-y-2">
          <SheetTitle>
            {mode === 'create' ? 'Add repository' : (selectedRepo?.name ?? 'Edit repository')}
          </SheetTitle>
          <SheetDescription>
            Configure repository metadata that ticket repo scopes, workflows, and workspace
            preparation consume.
          </SheetDescription>
        </div>

        <Button size="sm" onclick={onSave} disabled={saving} data-testid="repository-save-button">
          {saving ? 'Saving…' : mode === 'create' ? 'Create repository' : 'Save changes'}
        </Button>
      </div>
    </SheetHeader>

    <div class="flex-1 overflow-y-auto px-6 py-6">
      <RepositoryEditor
        {mode}
        {selectedRepo}
        {draft}
        {reposCount}
        {saving}
        {githubRepos}
        {githubRepoQuery}
        {githubReposLoading}
        {githubReposLoadingMore}
        {githubReposNextCursor}
        {githubRepoError}
        {githubBindingRepoFullName}
        {githubNamespaces}
        {githubNamespacesLoading}
        {githubCreateDraft}
        {githubCreating}
        {onDraftChange}
        {onGitHubRepoQueryChange}
        {onGitHubRepoSearch}
        {onGitHubRepoLoadMore}
        {onBindGitHubRepo}
        {onGitHubCreateDraftChange}
        {onCreateGitHubRepoAndBind}
      />
    </div>
  </SheetContent>
</Sheet>
