<script lang="ts">
  import { formatRelativeTime } from '$lib/utils'
  import { Button } from '$ui/button'
  import { CircleCheck, CircleAlert, Info } from '@lucide/svelte'
  import type { PrimaryRepositoryReadiness } from '../repositories-readiness'

  let {
    readiness,
    mirrorActionLabel = 'Set up mirror',
    onOpenPrimaryMirror,
  }: {
    readiness: PrimaryRepositoryReadiness
    mirrorActionLabel?: string
    onOpenPrimaryMirror?: (() => void) | undefined
  } = $props()
</script>

{#if readiness.kind === 'missing_primary_repo'}
  <div
    class="border-amber-500/30 bg-amber-500/5 flex items-center gap-3 rounded-lg border px-4 py-3"
  >
    <Info class="text-amber-600 size-4 shrink-0" />
    <p class="text-foreground flex-1 text-sm">
      No primary repository configured. Mark one repository as primary before configuring workflows.
    </p>
  </div>
{:else if readiness.kind === 'ready'}
  <div
    class="border-border bg-muted/30 flex items-center gap-3 rounded-lg border px-4 py-3"
  >
    <CircleCheck class="text-emerald-500 size-4 shrink-0" />
    <span class="text-foreground text-sm font-medium">{readiness.primaryRepoName}</span>
    <span class="text-muted-foreground text-sm">
      Primary mirror ready
      {#if readiness.lastSyncedAt}
        &middot; synced {formatRelativeTime(readiness.lastSyncedAt)}
      {/if}
    </span>
  </div>
{:else}
  <div
    class="border-amber-500/30 bg-amber-500/5 space-y-3 rounded-lg border px-4 py-3"
  >
    <div class="flex items-center gap-3">
      <CircleAlert class="text-amber-600 size-4 shrink-0" />
      <div class="flex-1">
        <p class="text-foreground text-sm">
          <span class="font-medium">{readiness.primaryRepoName}</span>
          {#if readiness.action === 'prepare_mirror'}
            needs a mirror before workflows can run.
          {:else if readiness.action === 'wait_for_mirror'}
            mirror is {readiness.mirrorState}. Waiting for it to finish.
          {:else if readiness.action === 'sync_mirror'}
            mirror is {readiness.mirrorState}. Repair or resync to continue.
          {/if}
        </p>
      </div>
      {#if onOpenPrimaryMirror}
        <Button variant="outline" size="sm" onclick={onOpenPrimaryMirror}>
          {mirrorActionLabel}
        </Button>
      {/if}
    </div>

    {#if readiness.lastError}
      <div class="bg-background/60 rounded-md border border-current/10 px-3 py-2 text-xs">
        <p class="text-muted-foreground break-words">{readiness.lastError}</p>
      </div>
    {/if}
  </div>
{/if}
