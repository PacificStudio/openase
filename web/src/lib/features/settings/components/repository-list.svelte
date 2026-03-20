<script lang="ts">
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'

  let {
    loading = false,
    repos = [],
    selectedId = '',
    onCreate,
    onSelect,
  }: {
    loading?: boolean
    repos?: ProjectRepoRecord[]
    selectedId?: string
    onCreate?: () => void
    onSelect?: (repo: ProjectRepoRecord) => void
  } = $props()
</script>

<div class="space-y-3">
  <div class="flex items-center justify-between gap-3">
    <div>
      <h3 class="text-foreground text-sm font-medium">Project repos</h3>
      <p class="text-muted-foreground text-xs">
        These repositories back ticket repo scopes and workspace preparation.
      </p>
    </div>
    <Button variant="outline" size="sm" onclick={onCreate}>New repo</Button>
  </div>

  {#if loading && repos.length === 0}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border px-4 py-10 text-center text-sm"
    >
      Loading repositories…
    </div>
  {:else if repos.length === 0}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border px-4 py-10 text-center text-sm"
    >
      No repositories configured yet. Add the first repository to seed repo scope defaults.
    </div>
  {:else}
    <div class="space-y-2">
      {#each repos as repo (repo.id)}
        <button
          type="button"
          class={cn(
            'border-border bg-card w-full rounded-xl border p-4 text-left transition-colors',
            repo.id === selectedId
              ? 'border-primary/50 ring-primary/20 bg-primary/5 ring-2'
              : 'hover:bg-muted/40',
          )}
          onclick={() => onSelect?.(repo)}
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span class="text-foreground truncate text-sm font-medium">{repo.name}</span>
                {#if repo.is_primary}
                  <Badge variant="secondary">Primary</Badge>
                {/if}
              </div>
              <p class="text-muted-foreground mt-1 truncate text-xs">{repo.repository_url}</p>
            </div>
            <Badge variant="outline">{repo.default_branch}</Badge>
          </div>

          <div class="text-muted-foreground mt-3 space-y-2 text-xs">
            <div>
              Clone path:
              <span class="text-foreground">{repo.clone_path || 'Auto workspace path'}</span>
            </div>

            {#if repo.labels.length === 0}
              <div>No labels</div>
            {:else}
              <div class="flex flex-wrap gap-1.5">
                {#each repo.labels as label (label)}
                  <Badge variant="outline">{label}</Badge>
                {/each}
              </div>
            {/if}
          </div>
        </button>
      {/each}
    </div>
  {/if}
</div>
