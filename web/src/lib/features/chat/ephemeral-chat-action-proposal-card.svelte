<script lang="ts">
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { ChevronRight, LoaderCircle, Check, X, ShieldAlert } from '@lucide/svelte'
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

  let expandedActions = $state(new Set<number>())

  function toggleAction(index: number) {
    const next = new Set(expandedActions)
    if (next.has(index)) {
      next.delete(index)
    } else {
      next.add(index)
    }
    expandedActions = next
  }

  const statusConfig = $derived(getStatusConfig(entry.status))

  function getStatusConfig(status: EphemeralChatActionProposalEntry['status']) {
    switch (status) {
      case 'pending':
        return { label: 'Pending', dot: 'bg-sky-400', text: 'text-sky-400' }
      case 'executing':
        return { label: 'Executing', dot: 'bg-amber-400', text: 'text-amber-400' }
      case 'confirmed':
        return { label: 'Executed', dot: 'bg-emerald-400', text: 'text-emerald-400' }
      case 'cancelled':
        return { label: 'Cancelled', dot: 'bg-muted-foreground/40', text: 'text-muted-foreground' }
    }
  }

  function methodColor(method: string) {
    switch (method.toUpperCase()) {
      case 'GET':
        return 'text-emerald-500'
      case 'POST':
        return 'text-sky-500'
      case 'PUT':
      case 'PATCH':
        return 'text-amber-500'
      case 'DELETE':
        return 'text-red-500'
      default:
        return 'text-muted-foreground'
    }
  }
</script>

<div class="border-border/50 bg-muted/10 overflow-hidden rounded-lg border">
  <!-- Header -->
  <div class="flex items-center gap-2 px-3 py-2">
    <ShieldAlert class="text-muted-foreground/70 size-3.5 shrink-0" />
    <span class="text-foreground min-w-0 flex-1 truncate text-xs font-medium">
      {entry.proposal.summary ?? 'Platform action'}
    </span>
    <div class="flex items-center gap-1.5">
      {#if entry.status === 'executing'}
        <LoaderCircle class="size-3 animate-spin text-amber-400" />
      {:else}
        <span class={cn('size-1.5 rounded-full', statusConfig.dot)}></span>
      {/if}
      <span class={cn('text-[10px]', statusConfig.text)}>{statusConfig.label}</span>
    </div>
  </div>

  <!-- Actions -->
  <div class="border-border/30 border-t">
    {#each entry.proposal.actions as action, index}
      <div class={cn(index > 0 && 'border-border/20 border-t')}>
        <button
          type="button"
          class="hover:bg-muted/30 flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs transition-colors"
          onclick={() => toggleAction(index)}
        >
          <ChevronRight
            class={cn(
              'text-muted-foreground size-3 shrink-0 transition-transform duration-150',
              expandedActions.has(index) && 'rotate-90',
            )}
          />
          <span
            class={cn('shrink-0 font-mono text-[11px] font-semibold', methodColor(action.method))}
          >
            {action.method}
          </span>
          <code class="text-foreground/70 min-w-0 flex-1 truncate font-mono text-[11px]">
            {action.path}
          </code>
        </button>

        {#if expandedActions.has(index) && action.body}
          <div class="border-border/20 border-t px-3 py-2">
            <pre
              class="bg-muted/40 max-h-48 overflow-auto rounded-md px-2.5 py-2 font-mono text-[11px] leading-5 whitespace-pre-wrap">{JSON.stringify(
                action.body,
                null,
                2,
              )}</pre>
          </div>
        {/if}
      </div>
    {/each}
  </div>

  <!-- Actions / Status footer -->
  {#if entry.status === 'pending'}
    <div class="border-border/30 flex gap-2 border-t px-3 py-2">
      <Button
        size="sm"
        class="h-7 gap-1.5 px-3 text-[11px]"
        onclick={() => void onConfirm?.(entry.id)}
      >
        <Check class="size-3" />
        Confirm
      </Button>
      <Button
        variant="ghost"
        size="sm"
        class="h-7 gap-1.5 px-3 text-[11px]"
        onclick={() => onCancel?.(entry.id)}
      >
        <X class="size-3" />
        Cancel
      </Button>
    </div>
  {:else if entry.status === 'cancelled'}
    <div class="border-border/30 text-muted-foreground border-t px-3 py-2 text-[11px]">
      No API calls were executed.
    </div>
  {/if}

  <!-- Results -->
  {#if entry.results.length > 0}
    <div class="border-border/30 space-y-1 border-t px-3 py-2">
      {#each entry.results as result}
        <div class="flex items-start gap-2 text-[11px] leading-relaxed">
          {#if result.ok}
            <Check class="mt-0.5 size-3 shrink-0 text-emerald-500" />
          {:else}
            <X class="mt-0.5 size-3 shrink-0 text-red-500" />
          {/if}
          <div class="min-w-0">
            <span class="text-foreground">{result.summary}</span>
            {#if result.detail}
              <span class="text-muted-foreground"> — {result.detail}</span>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
