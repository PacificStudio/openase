<script lang="ts">
  import { Button } from '$ui/button'
  import { Sheet, SheetContent, SheetHeader, SheetTitle } from '$ui/sheet'
  import * as Tabs from '$ui/tabs'
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import RepositoryEditorForm from './repository-editor-form.svelte'
  import RepositoryGitHubBrowser from './repository-github-browser.svelte'
  import RepositoryGitHubCreate from './repository-github-create.svelte'
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
    <SheetHeader class="border-border border-b px-6 py-4 text-left">
      <div class="flex items-center justify-between gap-4 pr-10">
        <SheetTitle class="text-base">
          {mode === 'create' ? 'Add repository' : (selectedRepo?.name ?? 'Edit repository')}
        </SheetTitle>
        {#if mode === 'edit'}
          <Button size="sm" onclick={onSave} disabled={saving} data-testid="repository-save-button">
            {saving ? 'Saving…' : 'Save changes'}
          </Button>
        {/if}
      </div>
    </SheetHeader>

    {#if mode === 'create'}
      <div class="flex flex-1 flex-col overflow-hidden">
        <Tabs.Root value="github" class="flex flex-1 flex-col overflow-hidden">
          <div class="border-border border-b px-6 pt-2">
            <Tabs.List>
              <Tabs.Trigger value="github">GitHub</Tabs.Trigger>
              <Tabs.Trigger value="create">Create repo</Tabs.Trigger>
              <Tabs.Trigger value="manual">Manual</Tabs.Trigger>
            </Tabs.List>
          </div>

          <Tabs.Content value="github" class="flex-1 overflow-y-auto px-6 py-4">
            <RepositoryGitHubBrowser
              repos={githubRepos}
              query={githubRepoQuery}
              loading={githubReposLoading}
              loadingMore={githubReposLoadingMore}
              nextCursor={githubReposNextCursor}
              error={githubRepoError}
              bindingRepoFullName={githubBindingRepoFullName}
              onQueryChange={onGitHubRepoQueryChange}
              onSearch={onGitHubRepoSearch}
              onLoadMore={onGitHubRepoLoadMore}
              onBind={onBindGitHubRepo}
            />
          </Tabs.Content>

          <Tabs.Content value="create" class="flex-1 overflow-y-auto px-6 py-4">
            <RepositoryGitHubCreate
              namespaces={githubNamespaces}
              draft={githubCreateDraft}
              loadingNamespaces={githubNamespacesLoading}
              creating={githubCreating}
              onDraftChange={onGitHubCreateDraftChange}
              onCreate={onCreateGitHubRepoAndBind}
            />
          </Tabs.Content>

          <Tabs.Content value="manual" class="flex-1 overflow-y-auto px-6 py-4">
            <RepositoryEditorForm {draft} {onDraftChange} compact />
            <div class="mt-4">
              <Button onclick={onSave} disabled={saving} data-testid="repository-save-button">
                {saving ? 'Creating…' : 'Create repository'}
              </Button>
            </div>
          </Tabs.Content>
        </Tabs.Root>
      </div>
    {:else}
      <div class="flex-1 overflow-y-auto px-6 py-5">
        <RepositoryEditorForm {draft} {onDraftChange} />
      </div>
    {/if}
  </SheetContent>
</Sheet>
