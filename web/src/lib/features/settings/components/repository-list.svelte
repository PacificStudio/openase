<script lang="ts">
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
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
  } = $props()
</script>

<Card.Root>
  <Card.Header class="flex-row items-center justify-between gap-3">
    <div>
      <Card.Title>Project repositories</Card.Title>
      <Card.Description>
        Repositories backing ticket repo scopes and workspace preparation.
      </Card.Description>
    </div>
    <Button size="sm" onclick={onCreate}>
      <Plus class="size-3.5" />
      New repo
    </Button>
  </Card.Header>

  <Card.Content class="space-y-3">
    {#if loading && repos.length === 0}
      <div class="text-muted-foreground py-8 text-center text-sm">
        Loading repositories…
      </div>
    {:else if repos.length === 0}
      <div class="text-muted-foreground py-8 text-center text-sm">
        No repositories configured yet. Add the first repository to get started.
      </div>
    {:else}
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
        />
      {/each}
    {/if}
  </Card.Content>
</Card.Root>
