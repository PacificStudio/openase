<script lang="ts">
  import type { ProjectRepoRecord } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { cn } from '$lib/utils'
  import { Pencil, Trash2 } from '@lucide/svelte'
  import {
    formatMirrorTimestamp,
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
</script>

<article
  data-testid={`repository-card-${repo.id}`}
  class={cn(
    'border-border bg-card hover:bg-muted/20 rounded-2xl border p-4 transition-colors',
    selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
  )}
>
  <div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_auto] xl:items-center">
    <button
      type="button"
      class="min-w-0 text-left"
      onclick={() => handleOpenRepo?.()}
      data-testid={`repository-open-${repo.id}`}
    >
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <h3 class="text-foreground truncate text-base font-semibold">{repo.name}</h3>
            {#if repo.is_primary}
              <Badge variant="secondary">Primary</Badge>
            {/if}
            <span
              class={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${repositoryMirrorToneClasses(mirror.mirrorState)}`}
            >
              {repositoryMirrorStateLabel(mirror.mirrorState)}
            </span>
          </div>
          <p class="text-muted-foreground mt-1 truncate text-sm">{repo.repository_url}</p>
        </div>
        <Badge variant="outline">{repo.default_branch}</Badge>
      </div>

      <div class="text-muted-foreground mt-4 flex flex-wrap items-center gap-3 text-xs">
        <span>
          Mirrors:
          <span class="text-foreground">{mirror.mirrorCount}</span>
        </span>
        <span>
          Workspace dirname:
          <span class="text-foreground">{repo.workspace_dirname || repo.name}</span>
        </span>
        {#if mirror.mirrorMachineId}
          <span>
            Target machine:
            <span class="text-foreground">{mirror.mirrorMachineId}</span>
          </span>
        {/if}
        {#if formatMirrorTimestamp(mirror.lastSyncedAt)}
          <span>
            Last synced:
            <span class="text-foreground">{formatMirrorTimestamp(mirror.lastSyncedAt)}</span>
          </span>
        {/if}

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

      {#if mirror.lastError}
        <div
          class="border-destructive/20 bg-destructive/5 mt-3 rounded-xl border px-3 py-2 text-xs"
        >
          <p class="text-foreground font-medium">Last mirror error</p>
          <p class="text-muted-foreground mt-1 break-words">{mirror.lastError}</p>
        </div>
      {/if}
    </button>

    <div class="flex flex-wrap items-center justify-end gap-2 xl:flex-col xl:items-stretch">
      {#if mirror.action !== 'none'}
        <Button
          size="sm"
          variant="outline"
          class="gap-1.5"
          onclick={(event) => {
            event.stopPropagation()
            onMaterialize?.()
          }}
          disabled={materializing || mirror.action === 'wait_for_mirror'}
        >
          {materializing
            ? 'Updating mirror…'
            : mirror.action === 'wait_for_mirror'
              ? 'Mirror busy'
              : mirrorActionLabel}
        </Button>
      {/if}
      <Button
        size="sm"
        class="gap-1.5"
        onclick={(event) => {
          event.stopPropagation()
          handleOpenRepo?.()
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
