<script lang="ts">
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import * as Tooltip from '$ui/tooltip'
  import { cn } from '$lib/utils'
  import { Ellipsis, HardDrive, Pencil, Trash2, RefreshCcw } from '@lucide/svelte'
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
    onConfigureMirror,
  }: {
    repo: ProjectRepoRecord
    selected?: boolean
    deleting?: boolean
    materializing?: boolean
    mirrorActionLabel?: string
    handleOpenRepo?: () => void
    onDelete?: () => void
    onMaterialize?: () => void
    onConfigureMirror?: () => void
  } = $props()

  let confirmDeleteOpen = $state(false)
  const mirror = $derived(projectRepoMirrorProjection(repo))

  const mirrorTooltipLines = $derived.by(() => {
    const lines: string[] = []
    lines.push(repo.repository_url)
    lines.push(`Branch: ${repo.default_branch}`)
    if (mirror.mirrorCount > 0) {
      lines.push(`Mirrors: ${mirror.mirrorCount}`)
    }
    if (mirror.lastSyncedAt) {
      lines.push(`Synced: ${formatRelativeTime(mirror.lastSyncedAt)}`)
    }
    if (mirror.lastVerifiedAt) {
      lines.push(`Verified: ${formatRelativeTime(mirror.lastVerifiedAt)}`)
    }
    if (mirror.lastError) {
      lines.push(`Error: ${mirror.lastError}`)
    }
    return lines
  })
</script>

<article
  data-testid={`repository-card-${repo.id}`}
  class={cn(
    'border-border/60 bg-card/60 flex items-center gap-3 rounded-xl border px-4 py-3',
    selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
  )}
>
  <!-- Main info -->
  <button
    type="button"
    class="min-w-0 flex-1 text-left"
    onclick={() => handleOpenRepo?.()}
    data-testid={`repository-open-${repo.id}`}
  >
    <div class="flex flex-wrap items-center gap-2">
      <span class="text-foreground text-sm font-semibold hover:underline">{repo.name}</span>
      <Tooltip.Root>
        <Tooltip.Trigger>
          {#snippet child({ props })}
            <span
              {...props}
              class={cn(
                'inline-flex cursor-help items-center rounded-full border px-1.5 py-0.5 text-[10px] font-medium',
                repositoryMirrorToneClasses(mirror.mirrorState),
              )}
            >
              {repositoryMirrorStateLabel(mirror.mirrorState)}
            </span>
          {/snippet}
        </Tooltip.Trigger>
        <Tooltip.Content side="bottom" sideOffset={6} class="max-w-xs">
          <div class="space-y-0.5 text-xs">
            {#each mirrorTooltipLines as line}
              <div>{line}</div>
            {/each}
          </div>
        </Tooltip.Content>
      </Tooltip.Root>
    </div>
    <div class="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs">
      <span>{repo.default_branch}</span>
      {#if mirror.mirrorCount > 0}
        <span>{mirror.mirrorCount} mirror{mirror.mirrorCount !== 1 ? 's' : ''}</span>
      {/if}
      {#if mirror.lastSyncedAt}
        <span>synced {formatRelativeTime(mirror.lastSyncedAt)}</span>
      {/if}
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
        <DropdownMenu.Item onclick={() => onConfigureMirror?.()}>
          <HardDrive class="mr-2 size-3.5" />
          Configure mirror
        </DropdownMenu.Item>
        {#if mirror.action !== 'none' && mirror.action !== 'prepare_mirror'}
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
        <DropdownMenu.Separator />
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

{#if mirror.lastError}
  <div class="text-destructive -mt-1 px-4 text-xs">{mirror.lastError}</div>
{/if}

<Dialog.Root bind:open={confirmDeleteOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Delete repository?</Dialog.Title>
      <Dialog.Description>
        This removes {repo.name} from the project. Existing ticket repo scopes or mirror references that
        point to it may need to be updated.
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
