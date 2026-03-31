<script lang="ts">
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { cn } from '$lib/utils'
  import { Ellipsis, Pencil, Trash2, RefreshCcw } from '@lucide/svelte'
  import {
    projectRepoMirrorProjection,
    repositoryMirrorStateLabel,
    repositoryMirrorToneClasses,
  } from '../repositories-readiness'

  let {
    repo,
    selected = false,
    deleting = false,
    materializing = false,
    mirrorActionLabel = 'Set up mirror',
    handleOpenRepo,
    onDelete,
    onMaterialize,
  }: {
    repo: ProjectRepoRecord
    selected?: boolean
    deleting?: boolean
    materializing?: boolean
    mirrorActionLabel?: string
    handleOpenRepo?: () => void
    onDelete?: () => void
    onMaterialize?: () => void
  } = $props()

  let confirmDeleteOpen = $state(false)
  const mirror = $derived(projectRepoMirrorProjection(repo))

  const metaParts = $derived.by(() => {
    const parts: string[] = [repo.default_branch]
    if (mirror.mirrorCount > 0) {
      parts.push(`${mirror.mirrorCount} mirror${mirror.mirrorCount !== 1 ? 's' : ''}`)
    }
    if (mirror.lastSyncedAt) {
      parts.push(`synced ${formatRelativeTime(mirror.lastSyncedAt)}`)
    }
    return parts
  })
</script>

<article
  data-testid={`repository-card-${repo.id}`}
  class={cn(
    'border-border bg-card hover:bg-muted/20 rounded-lg border p-4 transition-colors',
    selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
  )}
>
  <div class="flex items-start gap-3">
    <button
      type="button"
      class="min-w-0 flex-1 text-left"
      onclick={() => handleOpenRepo?.()}
      data-testid={`repository-open-${repo.id}`}
    >
      <div class="flex flex-wrap items-center gap-2">
        <h3 class="text-foreground truncate text-sm font-semibold">{repo.name}</h3>
        {#if repo.is_primary}
          <Badge variant="secondary" class="text-[10px]">Primary</Badge>
        {/if}
        <span
          class={`inline-flex items-center rounded-full border px-1.5 py-0.5 text-[10px] font-medium ${repositoryMirrorToneClasses(mirror.mirrorState)}`}
        >
          {repositoryMirrorStateLabel(mirror.mirrorState)}
        </span>
      </div>

      <p class="text-muted-foreground mt-1 truncate text-xs">{repo.repository_url}</p>

      <p class="text-muted-foreground mt-1.5 text-xs">
        {metaParts.join(' \u00b7 ')}
      </p>

      {#if repo.labels.length > 0}
        <div class="mt-2 flex flex-wrap gap-1">
          {#each repo.labels as label (label)}
            <Badge variant="outline" class="text-[10px]">{label}</Badge>
          {/each}
        </div>
      {/if}

      {#if mirror.lastError}
        <p class="text-destructive mt-2 text-xs">{mirror.lastError}</p>
      {/if}
    </button>

    <div class="flex shrink-0 items-center gap-1.5">
      <Button
        size="sm"
        variant="ghost"
        class="h-8 w-8 p-0"
        onclick={(event) => {
          event.stopPropagation()
          handleOpenRepo?.()
        }}
      >
        <Pencil class="size-3.5" />
        <span class="sr-only">Edit</span>
      </Button>

      <DropdownMenu.Root>
        <DropdownMenu.Trigger>
          {#snippet child({ props })}
            <Button
              size="sm"
              variant="ghost"
              class="h-8 w-8 p-0"
              {...props}
              onclick={(event) => event.stopPropagation()}
            >
              <Ellipsis class="size-3.5" />
              <span class="sr-only">More actions</span>
            </Button>
          {/snippet}
        </DropdownMenu.Trigger>
        <DropdownMenu.Content align="end" class="w-48">
          {#if mirror.action !== 'none'}
            <DropdownMenu.Item
              disabled={materializing || mirror.action === 'wait_for_mirror'}
              onclick={() => onMaterialize?.()}
            >
              <RefreshCcw class="mr-2 size-3.5" />
              {materializing
                ? 'Updating mirror\u2026'
                : mirror.action === 'wait_for_mirror'
                  ? 'Mirror busy'
                  : mirrorActionLabel}
            </DropdownMenu.Item>
          {/if}
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
        {deleting ? 'Deleting\u2026' : 'Delete repository'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
