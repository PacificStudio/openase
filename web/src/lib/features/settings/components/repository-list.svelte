<script lang="ts">
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import RepositoryRowCard from './repository-row-card.svelte'

  let {
    loading = false,
    repos = [],
    selectedId = '',
    deletingId = '',
    onCreate,
    onSelect,
    onDelete,
  }: {
    loading?: boolean
    repos?: ProjectRepoRecord[]
    selectedId?: string
    deletingId?: string
    onCreate?: () => void
    onSelect?: (repo: ProjectRepoRecord) => void
    onDelete?: (repo: ProjectRepoRecord) => void
  } = $props()
</script>

<div class="space-y-4">
  <div
    class="flex flex-col gap-3 rounded-2xl border border-dashed px-4 py-4 md:flex-row md:items-center md:justify-between"
  >
    <div>
      <h3 class="text-foreground text-sm font-medium">Project repos</h3>
      <p class="text-muted-foreground mt-1 text-sm">
        These repositories back ticket repo scopes and workspace preparation.
      </p>
    </div>
    <Button size="sm" onclick={onCreate}>New repo</Button>
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
    <div class="space-y-3">
      {#each repos as repo (repo.id)}
        <RepositoryRowCard
          {repo}
          selected={repo.id === selectedId}
          deleting={deletingId === repo.id}
          onOpen={() => onSelect?.(repo)}
          onDelete={() => onDelete?.(repo)}
        />
      {/each}
    </div>
  {/if}
</div>
