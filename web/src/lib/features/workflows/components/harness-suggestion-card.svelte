<script lang="ts">
  import type { DiffPreview, HarnessSuggestion } from '../assistant'
  import { cn } from '$lib/utils'
  import { Check, X } from '@lucide/svelte'
  import HarnessDiffPreview from './harness-diff-preview.svelte'

  let {
    suggestion,
    preview,
    suggestionAlreadyApplied,
    onApply,
    onDismiss,
  }: {
    suggestion: HarnessSuggestion
    preview: DiffPreview
    suggestionAlreadyApplied: boolean
    onApply?: () => void
    onDismiss?: () => void
  } = $props()
</script>

<div class="space-y-1.5">
  <div class="flex items-center justify-between gap-2">
    <p class="text-muted-foreground min-w-0 truncate text-[11px]">{suggestion.summary}</p>
    {#if suggestionAlreadyApplied}
      <span class="text-muted-foreground shrink-0 text-[10px] italic">Applied</span>
    {:else}
      <div class="flex shrink-0 items-center gap-1">
        <button
          type="button"
          class={cn(
            'flex items-center gap-1 rounded-md px-2 py-0.5 text-[11px] font-medium transition-colors',
            'bg-emerald-600 text-white hover:bg-emerald-500',
          )}
          onclick={() => onApply?.()}
        >
          <Check class="size-3" />
          Apply
        </button>
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground rounded-md p-0.5 transition-colors"
          onclick={() => onDismiss?.()}
          aria-label="Dismiss suggestion"
        >
          <X class="size-3.5" />
        </button>
      </div>
    {/if}
  </div>

  <HarnessDiffPreview {preview} />
</div>
