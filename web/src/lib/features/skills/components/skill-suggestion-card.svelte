<script lang="ts">
  import { StructuredDiffPreview } from '$lib/features/chat'
  import type { DiffPreview, SkillSuggestion } from '$lib/features/skills/assistant'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'

  let {
    suggestion,
    selectedPath,
    preview,
    suggestionAlreadyApplied,
    onSelectPath,
    onApply,
  }: {
    suggestion: SkillSuggestion
    selectedPath: string
    preview: DiffPreview
    suggestionAlreadyApplied: boolean
    onSelectPath?: (path: string) => void
    onApply?: () => void
  } = $props()
</script>

<div class="space-y-2 rounded-lg border border-sky-500/30 bg-sky-500/8 p-2">
  <div class="flex items-center justify-between gap-2">
    <div class="min-w-0">
      <p class="text-muted-foreground truncate text-[11px] leading-4">{suggestion.summary}</p>
      <p class="text-muted-foreground truncate text-[10px] leading-4">
        {suggestion.files.length} file{suggestion.files.length === 1 ? '' : 's'}
      </p>
    </div>
    {#if suggestionAlreadyApplied}
      <Badge variant="outline" class="shrink-0 text-[10px]">Applied</Badge>
    {:else}
      <Button size="sm" class="h-6 shrink-0 px-2.5 text-[11px]" onclick={() => onApply?.()}>
        Apply All
      </Button>
    {/if}
  </div>

  <div class="flex flex-wrap gap-1" data-testid="skill-suggestion-file-list">
    {#each suggestion.files as file (file.path)}
      <button
        type="button"
        class={`rounded-full border px-2 py-0.5 text-[11px] transition-colors ${
          file.path === selectedPath
            ? 'border-primary/40 bg-primary/10 text-foreground'
            : 'border-border text-muted-foreground hover:bg-muted'
        }`}
        onclick={() => onSelectPath?.(file.path)}
      >
        {file.path}
      </button>
    {/each}
  </div>

  <StructuredDiffPreview {preview} />
</div>
