<script lang="ts">
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { cn } from '$lib/utils'
  import { Ellipsis, Pencil, Trash2 } from '@lucide/svelte'

  let {
    repo,
    selected = false,
    deleting = false,
    handleOpenRepo,
    onDelete,
  }: {
    repo: ProjectRepoRecord
    selected?: boolean
    deleting?: boolean
    handleOpenRepo?: () => void
    onDelete?: () => void
  } = $props()

  let confirmDeleteOpen = $state(false)
</script>

<article
  data-testid={`repository-card-${repo.id}`}
  class={cn(
    'border-border/60 bg-card/60 flex items-center gap-3 rounded-xl border px-4 py-3',
    selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
  )}
>
  <button
    type="button"
    class="min-w-0 flex-1 text-left"
    onclick={() => handleOpenRepo?.()}
    data-testid={`repository-open-${repo.id}`}
  >
    <div class="flex flex-wrap items-center gap-2">
      <span class="text-foreground text-sm font-semibold hover:underline">{repo.name}</span>
    </div>
    <div class="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs">
      <span>{repo.default_branch}</span>
      <span class="truncate">{repo.repository_url}</span>
      {#if repo.labels.length > 0}
        {#each repo.labels as label (label)}
          <Badge variant="outline" class="text-[10px]">{label}</Badge>
        {/each}
      {/if}
    </div>
  </button>

  <!-- Actions -->
  <div class="flex shrink-0 items-center gap-1">
    <Button
      variant="ghost"
      size="icon-xs"
      title="Edit repository"
      onclick={(event) => {
        event.stopPropagation()
        handleOpenRepo?.()
      }}
    >
      <Pencil class="size-3.5" />
    </Button>

    <DropdownMenu.Root>
      <DropdownMenu.Trigger>
        {#snippet child({ props })}
          <Button
            variant="ghost"
            size="icon-xs"
            {...props}
            onclick={(event) => event.stopPropagation()}
          >
            <Ellipsis class="size-3.5" />
            <span class="sr-only">More actions</span>
          </Button>
        {/snippet}
      </DropdownMenu.Trigger>
      <DropdownMenu.Content align="end" class="w-48">
        <DropdownMenu.Item
          class="text-destructive focus:text-destructive"
          disabled={deleting}
          onclick={() => {
            confirmDeleteOpen = true
          }}
        >
          <Trash2 class="mr-2 size-3.5" />
          {deleting ? 'Deleting\u2026' : 'Delete'}
        </DropdownMenu.Item>
      </DropdownMenu.Content>
    </DropdownMenu.Root>
  </div>
</article>

<Dialog.Root bind:open={confirmDeleteOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Delete repository?</Dialog.Title>
      <Dialog.Description>
        This removes {repo.name} from the project. Existing ticket repo scopes that point to it may need
        to be updated.
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
        {deleting ? 'Deleting\u2026' : 'Delete repository'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
