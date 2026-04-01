<script lang="ts">
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import { Plus } from '@lucide/svelte'
  import RepositoryRowCard from './repository-row-card.svelte'

  let {
    loading = false,
    repos = [],
    selectedId = '',
    deletingId = '',
    onCreate,
    onOpenRepo,
    onDelete,
  }: {
    loading?: boolean
    repos?: ProjectRepoRecord[]
    selectedId?: string
    deletingId?: string
    onCreate?: () => void
    onOpenRepo?: (repo: ProjectRepoRecord) => void
    onDelete?: (repo: ProjectRepoRecord) => void
  } = $props()
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-3">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Project repositories</h3>
      <p class="text-muted-foreground mt-0.5 text-xs">
        Repositories backing ticket repo scopes and workspace preparation.
      </p>
    </div>
    <Button size="sm" onclick={onCreate}>
      <Plus class="mr-1.5 size-3.5" />
      New repo
    </Button>
  </div>

  {#if loading && repos.length === 0}
    <div class="space-y-2">
      {#each { length: 2 } as _}
        <div class="border-border bg-card flex items-center gap-3 rounded-xl border px-4 py-3">
          <div class="bg-muted size-8 shrink-0 animate-pulse rounded-lg"></div>
          <div class="min-w-0 flex-1 space-y-1.5">
            <div class="bg-muted h-4 w-36 animate-pulse rounded"></div>
            <div class="bg-muted h-3 w-56 animate-pulse rounded"></div>
          </div>
          <div class="flex shrink-0 items-center gap-1">
            <div class="bg-muted size-7 animate-pulse rounded"></div>
            <div class="bg-muted size-7 animate-pulse rounded"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else if repos.length === 0}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border border-dashed px-4 py-10 text-center text-sm"
    >
      No repositories configured yet. Add the first repository to get started.
    </div>
  {:else}
    <div class="space-y-2">
      {#each repos as repo (repo.id)}
        <RepositoryRowCard
          {repo}
          selected={repo.id === selectedId}
          deleting={deletingId === repo.id}
          handleOpenRepo={() => onOpenRepo?.(repo)}
          onDelete={() => onDelete?.(repo)}
        />
      {/each}
    </div>
  {/if}
</div>
