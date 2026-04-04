<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Search } from '@lucide/svelte'
  import type { GitHubRepositoryRecord } from '../repositories-model'

  let {
    repos = [],
    query = '',
    loading = false,
    loadingMore = false,
    nextCursor = '',
    bindingRepoFullName = '',
    error = '',
    onQueryChange,
    onSearch,
    onLoadMore,
    onBind,
  }: {
    repos?: GitHubRepositoryRecord[]
    query?: string
    loading?: boolean
    loadingMore?: boolean
    nextCursor?: string
    bindingRepoFullName?: string
    error?: string
    onQueryChange?: (value: string) => void
    onSearch?: () => void
    onLoadMore?: () => void
    onBind?: (repo: GitHubRepositoryRecord) => void
  } = $props()
</script>

<div class="flex flex-col gap-3">
  <div
    class="border-input focus-within:ring-ring flex items-center gap-2 rounded-md border px-3 focus-within:ring-1"
  >
    <Search class="text-muted-foreground size-3.5 shrink-0" />
    <input
      type="text"
      value={query}
      placeholder="Search repositories…"
      class="placeholder:text-muted-foreground h-9 flex-1 bg-transparent text-sm outline-none"
      oninput={(event) => onQueryChange?.((event.currentTarget as HTMLInputElement).value)}
      onkeydown={(event) => {
        if (event.key === 'Enter' && !event.isComposing) {
          event.preventDefault()
          onSearch?.()
        }
      }}
    />
    {#if loading}
      <span class="text-muted-foreground shrink-0 text-xs">Searching…</span>
    {/if}
  </div>

  {#if error}
    <div class="text-destructive text-xs">{error}</div>
  {/if}

  {#if loading && repos.length === 0}
    <div class="text-muted-foreground py-6 text-center text-xs">Loading repositories…</div>
  {:else if repos.length === 0}
    <div class="text-muted-foreground py-6 text-center text-xs">
      No repositories matched. Try a different search.
    </div>
  {:else}
    <div
      class="border-border overflow-y-auto rounded-md border"
      style="max-height: calc(100vh - 220px)"
    >
      {#each repos as repo, index (repo.id)}
        <button
          type="button"
          class="hover:bg-muted/50 flex w-full items-center gap-3 px-3 py-2 text-left transition-colors disabled:opacity-50
            {index > 0 ? 'border-border/40 border-t' : ''}"
          disabled={bindingRepoFullName === repo.full_name}
          onclick={() => onBind?.(repo)}
        >
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="text-foreground truncate text-sm font-medium">{repo.full_name}</span>
              <Badge variant="outline" class="h-4 shrink-0 px-1.5 text-[10px] capitalize">
                {repo.visibility}
              </Badge>
              <span class="text-muted-foreground shrink-0 text-[10px]">{repo.default_branch}</span>
            </div>
          </div>
          <span class="text-muted-foreground shrink-0 text-xs">
            {bindingRepoFullName === repo.full_name ? 'Binding…' : 'Bind'}
          </span>
        </button>
      {/each}

      {#if nextCursor}
        <div class="border-border/40 border-t px-3 py-2">
          <Button
            variant="ghost"
            size="sm"
            class="h-7 w-full text-xs"
            onclick={onLoadMore}
            disabled={loadingMore}
          >
            {loadingMore ? 'Loading…' : 'Load more'}
          </Button>
        </div>
      {/if}
    </div>
  {/if}
</div>
