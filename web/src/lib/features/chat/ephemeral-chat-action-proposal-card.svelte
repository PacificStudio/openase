<script lang="ts">
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { LoaderCircle } from '@lucide/svelte'
  import type { EphemeralChatActionProposalEntry } from './transcript'

  let {
    entry,
    onConfirm,
    onCancel,
  }: {
    entry: EphemeralChatActionProposalEntry
    onConfirm?: (entryId: string) => Promise<void> | void
    onCancel?: (entryId: string) => void
  } = $props()

  const statusLabel = $derived(getStatusLabel(entry.status))
  const statusClassName = $derived(getStatusClassName(entry.status))

  function getStatusLabel(status: EphemeralChatActionProposalEntry['status']) {
    switch (status) {
      case 'pending':
        return 'Pending confirmation'
      case 'executing':
        return 'Executing'
      case 'confirmed':
        return 'Executed'
      case 'cancelled':
        return 'Cancelled'
    }
  }

  function getStatusClassName(status: EphemeralChatActionProposalEntry['status']) {
    switch (status) {
      case 'pending':
        return 'border-sky-500/30 text-sky-700'
      case 'executing':
        return 'border-amber-500/30 text-amber-700'
      case 'confirmed':
        return 'border-emerald-500/30 text-emerald-700'
      case 'cancelled':
        return 'border-slate-500/30 text-slate-700'
    }
  }
</script>

<div class="rounded-2xl border border-sky-500/30 bg-sky-500/5 px-3 py-3 text-sm leading-6">
  <div class="mb-1 text-[10px] font-semibold tracking-[0.16em] uppercase opacity-70">assistant</div>

  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0">
      <div class="font-medium">
        {entry.proposal.summary ?? 'Proposed platform action'}
      </div>
      <p class="text-muted-foreground mt-1 text-xs leading-5">
        {entry.proposal.actions.length} action{entry.proposal.actions.length === 1 ? '' : 's'}
        require explicit confirmation before any platform write.
      </p>
    </div>
    <Badge variant="outline" class={statusClassName}>{statusLabel}</Badge>
  </div>

  <div class="mt-3 space-y-2">
    {#each entry.proposal.actions as action, index}
      <div class="bg-background/70 rounded-xl border border-sky-500/20 px-3 py-2">
        <div class="flex flex-wrap items-center gap-2 text-xs">
          <span class="text-muted-foreground">{index + 1}.</span>
          <span class="font-semibold">{action.method}</span>
          <code class="rounded bg-slate-950/5 px-1.5 py-0.5 text-[11px]">{action.path}</code>
        </div>
        {#if action.body}
          <pre
            class="mt-2 overflow-x-auto rounded-lg bg-slate-950 px-3 py-2 text-[11px] leading-5 text-slate-100">{JSON.stringify(
              action.body,
              null,
              2,
            )}</pre>
        {/if}
      </div>
    {/each}
  </div>

  {#if entry.status === 'pending'}
    <div class="mt-3 flex flex-wrap gap-2">
      <Button size="sm" onclick={() => void onConfirm?.(entry.id)}>Confirm</Button>
      <Button variant="outline" size="sm" onclick={() => onCancel?.(entry.id)}>Cancel</Button>
    </div>
  {:else if entry.status === 'executing'}
    <div
      class="mt-3 inline-flex items-center gap-2 rounded-lg border border-sky-500/20 px-3 py-2 text-xs"
    >
      <LoaderCircle class="size-4 animate-spin" />
      Executing the proposed platform actions...
    </div>
  {:else if entry.status === 'cancelled'}
    <div class="text-muted-foreground mt-3 text-xs">No platform API calls were executed.</div>
  {/if}

  {#if entry.results.length > 0}
    <div class="mt-3 space-y-2">
      {#each entry.results as result}
        <div
          class={cn(
            'rounded-xl border px-3 py-2 text-xs leading-5',
            result.ok
              ? 'border-emerald-500/30 bg-emerald-500/10'
              : 'border-red-500/30 bg-red-500/10',
          )}
        >
          <div class="font-medium">{result.summary}</div>
          {#if result.detail}
            <div class="mt-1 opacity-80">{result.detail}</div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
