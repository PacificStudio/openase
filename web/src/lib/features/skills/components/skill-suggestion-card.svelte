<script lang="ts">
  import { StructuredDiffPreview } from '$lib/features/chat'
  import type { DiffPreview, SkillSuggestion } from '$lib/features/skills/assistant'
  import { cn } from '$lib/utils'
  import { Check, X } from '@lucide/svelte'

  let {
    suggestion,
    selectedPath,
    preview,
    suggestionAlreadyApplied,
    onSelectPath,
    onApply,
    onDismiss,
  }: {
    suggestion: SkillSuggestion
    selectedPath: string
    preview: DiffPreview
    suggestionAlreadyApplied: boolean
    onSelectPath?: (path: string) => void
    onApply?: () => void
    onDismiss?: () => void
  } = $props()
</script>

<div class="space-y-1.5">
  <div class="flex items-center justify-between gap-2">
    <div class="min-w-0">
      <p class="text-muted-foreground truncate text-[11px]">{suggestion.summary}</p>
      <p class="text-muted-foreground/60 text-[10px]">
        {suggestion.files.length} file{suggestion.files.length === 1 ? '' : 's'}
      </p>
    </div>
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
          Apply All
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

  <div class="flex flex-wrap gap-1" data-testid="skill-suggestion-file-list">
    {#each suggestion.files as file (file.path)}
      <button
        type="button"
        class={cn(
          'rounded-full border px-2 py-0.5 text-[11px] transition-colors',
          file.path === selectedPath
            ? 'border-primary/40 bg-primary/10 text-foreground'
            : 'border-border text-muted-foreground hover:bg-muted',
        )}
        onclick={() => onSelectPath?.(file.path)}
      >
        {file.path}
      </button>
    {/each}
  </div>

  <StructuredDiffPreview {preview} />
</div>
