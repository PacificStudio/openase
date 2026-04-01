<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
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

<section class="border-border bg-card/60 space-y-4 rounded-xl border p-4">
  <div class="space-y-1">
    <h3 class="text-foreground text-sm font-semibold">Bind from GitHub</h3>
    <p class="text-muted-foreground text-xs">
      Pick an existing repository from the current GitHub credential instead of typing clone URLs by
      hand.
    </p>
  </div>

  <div class="flex flex-col gap-2 sm:flex-row">
    <Input
      value={query}
      placeholder="Search owner or repo"
      oninput={(event) => onQueryChange?.((event.currentTarget as HTMLInputElement).value)}
      onkeydown={(event) => {
        if (event.key === 'Enter') {
          event.preventDefault()
          onSearch?.()
        }
      }}
    />
    <Button variant="outline" onclick={onSearch} disabled={loading}>Search</Button>
  </div>

  {#if error}
    <div class="text-destructive text-xs">{error}</div>
  {/if}

  {#if loading && repos.length === 0}
    <div class="text-muted-foreground py-6 text-sm">Loading GitHub repositories…</div>
  {:else if repos.length === 0}
    <div class="text-muted-foreground py-6 text-sm">
      No GitHub repositories matched the current search.
    </div>
  {:else}
    <div class="space-y-2">
      {#each repos as repo (repo.id)}
        <article class="border-border/60 flex items-center gap-3 rounded-lg border px-3 py-3">
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-foreground text-sm font-medium">{repo.full_name}</span>
              <Badge variant="outline" class="capitalize">{repo.visibility}</Badge>
              <span class="text-muted-foreground text-[11px] uppercase">{repo.default_branch}</span>
            </div>
            <div class="text-muted-foreground mt-1 truncate text-xs">{repo.clone_url}</div>
          </div>

          <Button
            size="sm"
            disabled={bindingRepoFullName === repo.full_name}
            onclick={() => onBind?.(repo)}
          >
            {bindingRepoFullName === repo.full_name ? 'Binding…' : 'Bind'}
          </Button>
        </article>
      {/each}
    </div>
  {/if}

  {#if nextCursor}
    <Button variant="outline" size="sm" onclick={onLoadMore} disabled={loadingMore}>
      {loadingMore ? 'Loading…' : 'Load more'}
    </Button>
  {/if}
</section>
