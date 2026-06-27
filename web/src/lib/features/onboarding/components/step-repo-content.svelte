<script lang="ts">
  import type {
    GitHubRepositoryNamespaceRecord,
    GitHubRepositoryRecord,
    ProjectRepoRecord,
  } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { Plus, Link, Loader2, CheckCircle2, GitBranch, FolderGit2, Search } from '@lucide/svelte'
  import { REPO_STEP_KEYS as KEYS, t as repoCopy, type StepRepoCopyKey } from './step-repo-copy'

  let {
    mode = $bindable<'choose' | 'create' | 'link'>(),
    creating = false,
    linking = false,
    searchingRepos = false,
    loadingMoreSearchResults = false,
    newRepoName = $bindable(''),
    newRepoVisibility = $bindable<'private' | 'public'>('private'),
    newRepoDefaultBranch = $bindable('main'),
    selectedNamespace = $bindable(''),
    repoSearchQuery = $bindable(''),
    searchResults = [],
    searchResultsNextCursor = '',
    repoSearchEmptyStateKey = KEYS.searchNoRepositories,
    linkRepoUrl = $bindable(''),
    linkRepoName = $bindable(''),
    linkRepoBranch = $bindable('main'),
    namespaces = [],
    repos = [],
    hasRepos = false,
    onEnterCreateMode,
    onEnterLinkMode,
    onSearchRepos,
    onLoadMoreSearchResults,
    onSelectSearchResult,
    onCreateRepo,
    onLinkRepo,
  }: {
    mode: 'choose' | 'create' | 'link'
    creating?: boolean
    linking?: boolean
    searchingRepos?: boolean
    loadingMoreSearchResults?: boolean
    newRepoName?: string
    newRepoVisibility?: 'private' | 'public'
    newRepoDefaultBranch?: string
    selectedNamespace?: string
    repoSearchQuery?: string
    searchResults?: GitHubRepositoryRecord[]
    searchResultsNextCursor?: string
    repoSearchEmptyStateKey?: StepRepoCopyKey
    linkRepoUrl?: string
    linkRepoName?: string
    linkRepoBranch?: string
    namespaces?: GitHubRepositoryNamespaceRecord[]
    repos?: ProjectRepoRecord[]
    hasRepos?: boolean
    onEnterCreateMode?: () => void
    onEnterLinkMode?: () => void
    onSearchRepos?: () => void | Promise<void>
    onLoadMoreSearchResults?: () => void | Promise<void>
    onSelectSearchResult?: (repo: GitHubRepositoryRecord) => void
    onCreateRepo?: () => void | Promise<void>
    onLinkRepo?: () => void | Promise<void>
  } = $props()
</script>

<div class="space-y-4">
  {#if hasRepos}
    <div class="space-y-2">
      {#each repos as repo (repo.id)}
        <div
          class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 dark:border-emerald-900/50 dark:bg-emerald-950/30"
        >
          <CheckCircle2 class="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
          <FolderGit2 class="text-muted-foreground size-4 shrink-0" />
          <div class="min-w-0 flex-1">
            <p class="text-foreground truncate text-sm font-medium">{repo.name}</p>
            <p class="text-muted-foreground truncate text-xs">{repo.repository_url}</p>
          </div>
          <span class="text-muted-foreground flex items-center gap-1 text-xs">
            <GitBranch class="size-3" />
            {repo.default_branch}
          </span>
        </div>
      {/each}
    </div>

    {#if mode === 'choose'}
      <Button variant="outline" size="sm" onclick={onEnterCreateMode}>
        <Plus class="mr-1.5 size-3.5" />
        {repoCopy(KEYS.actionsAddRepository)}
      </Button>
    {/if}
  {/if}

  {#if !hasRepos || mode !== 'choose'}
    {#if mode === 'choose'}
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <button
          type="button"
          class="border-border hover:border-primary/50 hover:bg-primary/5 flex items-start gap-3 rounded-lg border p-4 text-left transition-colors"
          onclick={onEnterCreateMode}
        >
          <div class="bg-primary/10 flex size-9 shrink-0 items-center justify-center rounded-lg">
            <Plus class="text-primary size-4" />
          </div>
          <div>
            <p class="text-foreground text-sm font-medium">{repoCopy(KEYS.createCardTitle)}</p>
            <p class="text-muted-foreground mt-0.5 text-xs">
              {repoCopy(KEYS.createCardDescription)}
            </p>
          </div>
        </button>

        <button
          type="button"
          class="border-border hover:border-primary/50 hover:bg-primary/5 flex items-start gap-3 rounded-lg border p-4 text-left transition-colors"
          onclick={onEnterLinkMode}
        >
          <div class="bg-primary/10 flex size-9 shrink-0 items-center justify-center rounded-lg">
            <Link class="text-primary size-4" />
          </div>
          <div>
            <p class="text-foreground text-sm font-medium">{repoCopy(KEYS.linkCardTitle)}</p>
            <p class="text-muted-foreground mt-0.5 text-xs">{repoCopy(KEYS.linkCardDescription)}</p>
          </div>
        </button>
      </div>
    {:else if mode === 'create'}
      <div class="space-y-3">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">{repoCopy(KEYS.formNamespace)}</p>
            <Select.Root
              type="single"
              value={selectedNamespace}
              onValueChange={(v) => {
                if (v) selectedNamespace = v
              }}
            >
              <Select.Trigger class="h-9 w-full text-sm">
                {selectedNamespace || repoCopy(KEYS.placeholderNamespace)}
              </Select.Trigger>
              <Select.Content>
                {#each namespaces as ns (ns.login)}
                  <Select.Item value={ns.login}>{ns.login}</Select.Item>
                {/each}
              </Select.Content>
            </Select.Root>
          </div>
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">
              {repoCopy(KEYS.formRepositoryName)}
            </p>
            <Input
              bind:value={newRepoName}
              placeholder={repoCopy(KEYS.placeholderRepositoryName)}
              class="h-9 text-sm"
            />
          </div>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">{repoCopy(KEYS.formVisibility)}</p>
            <Select.Root
              type="single"
              value={newRepoVisibility}
              onValueChange={(v) => {
                if (v) newRepoVisibility = v as 'private' | 'public'
              }}
            >
              <Select.Trigger class="h-9 w-full text-sm">
                {newRepoVisibility === 'private'
                  ? repoCopy(KEYS.visibilityPrivate)
                  : repoCopy(KEYS.visibilityPublic)}
              </Select.Trigger>
              <Select.Content>
                <Select.Item value="private">
                  {repoCopy(KEYS.visibilityPrivate)}
                </Select.Item>
                <Select.Item value="public">
                  {repoCopy(KEYS.visibilityPublic)}
                </Select.Item>
              </Select.Content>
            </Select.Root>
          </div>
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">
              {repoCopy(KEYS.formDefaultBranch)}
            </p>
            <Input
              bind:value={newRepoDefaultBranch}
              placeholder={repoCopy(KEYS.placeholderDefaultBranch)}
              class="h-9 text-sm"
            />
          </div>
        </div>
        <div class="flex items-center gap-2">
          <Button
            onclick={onCreateRepo}
            disabled={creating || !newRepoName.trim() || !selectedNamespace}
          >
            {#if creating}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              {repoCopy(KEYS.actionsCreating)}
            {:else}
              <Plus class="mr-1.5 size-3.5" />
              {repoCopy(KEYS.actionsCreateAndLink)}
            {/if}
          </Button>
          <Button variant="ghost" size="sm" onclick={() => (mode = 'choose')}>
            {repoCopy(KEYS.actionsBack)}
          </Button>
        </div>
      </div>
    {:else}
      <div class="space-y-3">
        <p class="text-muted-foreground text-xs leading-relaxed">
          {repoCopy(KEYS.searchTokenScope)}
        </p>
        <div>
          <p class="text-foreground mb-1 text-xs font-medium">
            {repoCopy(KEYS.searchHeading)}
          </p>
          <div
            class="border-input focus-within:ring-ring flex items-center gap-2 rounded-md border px-3 focus-within:ring-1"
          >
            <Search class="text-muted-foreground size-3.5 shrink-0" />
            <input
              type="text"
              bind:value={repoSearchQuery}
              placeholder={repoCopy(KEYS.searchPlaceholder)}
              class="placeholder:text-muted-foreground h-9 flex-1 bg-transparent text-sm outline-none"
              onkeydown={(e) => {
                if (e.key === 'Enter' && !e.isComposing) {
                  e.preventDefault()
                  void onSearchRepos?.()
                }
              }}
            />
            {#if searchingRepos}
              <span class="text-muted-foreground shrink-0 text-xs">
                {repoCopy(KEYS.searchSearching)}
              </span>
            {/if}
          </div>
        </div>

        {#if searchingRepos && searchResults.length === 0}
          <div class="text-muted-foreground py-4 text-center text-xs">
            {repoCopy(KEYS.searchSearching)}
          </div>
        {:else if searchResults.length === 0}
          <div class="flex flex-col items-center gap-1.5 py-4 text-center">
            <p class="text-muted-foreground text-xs">
              {repoCopy(repoSearchEmptyStateKey)}
            </p>
            <p class="text-muted-foreground/80 max-w-sm text-[11px] leading-relaxed">
              {repoCopy(KEYS.searchPermissionLimited)}
            </p>
          </div>
        {:else}
          <div class="border-border max-h-48 overflow-y-auto rounded-lg border">
            {#each searchResults as result (result.full_name)}
              <button
                type="button"
                class="hover:bg-muted w-full border-border/40 border-t px-3 py-2 text-left transition-colors first:border-t-0"
                onclick={() => onSelectSearchResult?.(result)}
              >
                <p class="text-foreground text-sm">{result.full_name}</p>
                <p class="text-muted-foreground text-xs">{result.visibility}</p>
              </button>
            {/each}
            {#if searchResultsNextCursor}
              <div class="border-border/40 border-t px-3 py-2">
                <Button
                  variant="ghost"
                  size="sm"
                  class="h-7 w-full text-xs"
                  onclick={() => onLoadMoreSearchResults?.()}
                  disabled={loadingMoreSearchResults}
                >
                  {loadingMoreSearchResults
                    ? repoCopy(KEYS.searchSearching)
                    : repoCopy(KEYS.searchLoadMore)}
                </Button>
              </div>
            {/if}
          </div>
        {/if}

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">
              {repoCopy(KEYS.formRepositoryName)}
            </p>
            <Input
              bind:value={linkRepoName}
              placeholder={repoCopy(KEYS.placeholderRepositoryName)}
              class="h-9 text-sm"
            />
          </div>
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">
              {repoCopy(KEYS.formDefaultBranch)}
            </p>
            <Input
              bind:value={linkRepoBranch}
              placeholder={repoCopy(KEYS.placeholderDefaultBranch)}
              class="h-9 text-sm"
            />
          </div>
        </div>
        <div>
          <p class="text-foreground mb-1 text-xs font-medium">{repoCopy(KEYS.formGitUrl)}</p>
          <Input
            bind:value={linkRepoUrl}
            placeholder={repoCopy(KEYS.placeholderGitUrl)}
            class="h-9 text-sm"
          />
        </div>

        <div class="flex items-center gap-2">
          <Button
            onclick={onLinkRepo}
            disabled={linking || !linkRepoUrl.trim() || !linkRepoName.trim()}
          >
            {#if linking}
              <Loader2 class="mr-1.5 size-3.5 animate-spin" />
              {repoCopy(KEYS.actionsLinking)}
            {:else}
              <Link class="mr-1.5 size-3.5" />
              {repoCopy(KEYS.actionsLinkRepository)}
            {/if}
          </Button>
          <Button variant="ghost" size="sm" onclick={() => (mode = 'choose')}>
            {repoCopy(KEYS.actionsBack)}
          </Button>
        </div>
      </div>
    {/if}
  {/if}
</div>
