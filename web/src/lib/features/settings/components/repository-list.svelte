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
    materializingId = '',
    mirrorActionLabelByRepoId = {},
    onCreate,
    onOpenRepo,
    onDelete,
    onMaterialize,
    onConfigureMirror,
  }: {
    loading?: boolean
    repos?: ProjectRepoRecord[]
    selectedId?: string
    deletingId?: string
    materializingId?: string
    mirrorActionLabelByRepoId?: Record<string, string>
    onCreate?: () => void
    onOpenRepo?: (repo: ProjectRepoRecord) => void
    onDelete?: (repo: ProjectRepoRecord) => void
    onMaterialize?: (repo: ProjectRepoRecord) => void
    onConfigureMirror?: (repo: ProjectRepoRecord) => void
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
    <div class="text-muted-foreground py-8 text-center text-sm">Loading repositories…</div>
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
          materializing={materializingId === repo.id}
          mirrorActionLabel={mirrorActionLabelByRepoId[repo.id] ?? 'Set up mirror'}
          handleOpenRepo={() => onOpenRepo?.(repo)}
          onDelete={() => onDelete?.(repo)}
          onMaterialize={() => onMaterialize?.(repo)}
          onConfigureMirror={() => onConfigureMirror?.(repo)}
        />
      {/each}
    </div>
  {/if}
</div>
