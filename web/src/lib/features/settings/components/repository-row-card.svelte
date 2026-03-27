<script lang="ts">
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { cn } from '$lib/utils'
  import { Pencil, Trash2 } from '@lucide/svelte'

  let {
    repo,
    selected = false,
    deleting = false,
    onOpen,
    onDelete,
  }: {
    repo: ProjectRepoRecord
    selected?: boolean
    deleting?: boolean
    onOpen?: () => void
    onDelete?: () => void
  } = $props()

  let confirmDeleteOpen = $state(false)
</script>

<article
  class={cn(
    'border-border bg-card hover:bg-muted/20 rounded-2xl border p-4 transition-colors',
    selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
  )}
>
  <div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_auto] xl:items-center">
    <button type="button" class="min-w-0 text-left" onclick={onOpen}>
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <h3 class="text-foreground truncate text-base font-semibold">{repo.name}</h3>
            {#if repo.is_primary}
              <Badge variant="secondary">Primary</Badge>
            {/if}
          </div>
          <p class="text-muted-foreground mt-1 truncate text-sm">{repo.repository_url}</p>
        </div>
        <Badge variant="outline">{repo.default_branch}</Badge>
      </div>

      <div class="text-muted-foreground mt-4 flex flex-wrap items-center gap-3 text-xs">
        <span>
          Clone path:
          <span class="text-foreground">{repo.clone_path || 'Auto workspace path'}</span>
        </span>

        {#if repo.labels.length === 0}
          <span>No labels</span>
        {:else}
          <div class="flex flex-wrap gap-1.5">
            {#each repo.labels as label (label)}
              <Badge variant="outline">{label}</Badge>
            {/each}
          </div>
        {/if}
      </div>
    </button>

    <div class="flex flex-wrap items-center justify-end gap-2 xl:flex-col xl:items-stretch">
      <Button
        size="sm"
        class="gap-1.5"
        onclick={(event) => {
          event.stopPropagation()
          onOpen?.()
        }}
      >
        <Pencil class="size-3.5" />
        Edit
      </Button>
      <Button
        size="sm"
        variant="destructive"
        class="gap-1.5"
        onclick={(event) => {
          event.stopPropagation()
          confirmDeleteOpen = true
        }}
        disabled={deleting}
      >
        <Trash2 class="size-3.5" />
        {deleting ? 'Deleting…' : 'Delete'}
      </Button>
    </div>
  </div>
</article>

<Dialog.Root bind:open={confirmDeleteOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Delete repository?</Dialog.Title>
      <Dialog.Description>
        This removes {repo.name} from the project. Existing ticket repo scopes or workflow defaults that
        reference it may need to be updated.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="mt-6">
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button
        variant="destructive"
        disabled={deleting}
        onclick={() => {
          confirmDeleteOpen = false
          onDelete?.()
        }}
      >
        {deleting ? 'Deleting…' : 'Delete repository'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
