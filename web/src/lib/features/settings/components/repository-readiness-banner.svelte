<script lang="ts">
  import type { PrimaryRepositoryReadiness } from '../repositories-readiness'
  import { formatMirrorTimestamp, repositoryMirrorToneClasses } from '../repositories-readiness'

  let { readiness }: { readiness: PrimaryRepositoryReadiness } = $props()
</script>

<section
  class={`rounded-2xl border px-4 py-4 ${repositoryMirrorToneClasses(readiness.kind === 'missing_primary_repo' ? 'missing' : readiness.mirrorState)}`}
>
  <div class="flex flex-wrap items-start justify-between gap-3">
    <div class="space-y-1">
      <h3 class="text-sm font-semibold">
        {#if readiness.kind === 'missing_primary_repo'}
          Primary repository not configured
        {:else if readiness.kind === 'ready'}
          Primary mirror ready
        {:else}
          Primary mirror needs attention
        {/if}
      </h3>
      <p class="text-sm">
        {#if readiness.kind === 'missing_primary_repo'}
          This project has repository bindings, but none of them is marked primary. Mark one
          repository as primary before configuring workflows or harness files.
        {:else if readiness.action === 'prepare_mirror'}
          <span class="font-medium">{readiness.primaryRepoName}</span> is bound as the primary repository,
          but no mirror is ready yet. Prepare a mirror on the target machine before editing workflows
          or harness files.
        {:else if readiness.action === 'wait_for_mirror'}
          <span class="font-medium">{readiness.primaryRepoName}</span> is bound as the primary
          repository, but its mirror is currently
          <span class="font-medium">{readiness.mirrorState}</span>. Wait for the mirror lifecycle to
          finish before continuing.
        {:else if readiness.action === 'sync_mirror'}
          <span class="font-medium">{readiness.primaryRepoName}</span> is bound as the primary
          repository, but its mirror is
          <span class="font-medium">{readiness.mirrorState}</span>. Repair or resync the mirror
          before using it for workflows.
        {:else}
          <span class="font-medium">{readiness.primaryRepoName}</span> has a ready primary mirror for
          workflow and harness operations.
        {/if}
      </p>
    </div>

    {#if readiness.kind !== 'missing_primary_repo'}
      <dl class="grid gap-x-4 gap-y-2 text-xs sm:grid-cols-2">
        <div>
          <dt class="opacity-70">Mirror state</dt>
          <dd class="font-medium">{readiness.mirrorState}</dd>
        </div>
        <div>
          <dt class="opacity-70">Known mirrors</dt>
          <dd class="font-medium">{readiness.mirrorCount}</dd>
        </div>
        {#if readiness.mirrorMachineId}
          <div>
            <dt class="opacity-70">Target machine</dt>
            <dd class="font-medium break-all">{readiness.mirrorMachineId}</dd>
          </div>
        {/if}
        {#if formatMirrorTimestamp(readiness.lastSyncedAt)}
          <div>
            <dt class="opacity-70">Last synced</dt>
            <dd class="font-medium">{formatMirrorTimestamp(readiness.lastSyncedAt)}</dd>
          </div>
        {/if}
        {#if formatMirrorTimestamp(readiness.lastVerifiedAt)}
          <div>
            <dt class="opacity-70">Last verified</dt>
            <dd class="font-medium">{formatMirrorTimestamp(readiness.lastVerifiedAt)}</dd>
          </div>
        {/if}
      </dl>
    {/if}
  </div>

  {#if readiness.kind !== 'missing_primary_repo' && readiness.lastError}
    <div class="bg-background/70 mt-3 rounded-xl border border-current/20 px-3 py-2 text-sm">
      <p class="font-medium">Last mirror error</p>
      <p class="mt-1 break-words opacity-80">{readiness.lastError}</p>
    </div>
  {/if}
</section>
