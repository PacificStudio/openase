<script lang="ts">
  import StructuredDiffPreview from '$lib/features/chat/structured-diff-preview.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import type { DiffPreview, SkillSuggestion } from '../assistant'

  let {
    suggestion,
    preview,
    suggestionAlreadyApplied,
    onApply,
  }: {
    suggestion: SkillSuggestion
    preview: DiffPreview
    suggestionAlreadyApplied: boolean
    onApply?: () => void
  } = $props()
</script>

<div class="space-y-2 rounded-lg border border-sky-500/30 bg-sky-500/8 p-2">
  <div class="flex items-center justify-between gap-2">
    <div class="min-w-0">
      <p class="text-muted-foreground truncate text-[11px] leading-4">{suggestion.summary}</p>
      <p class="text-muted-foreground truncate text-[10px] leading-4">{suggestion.path}</p>
    </div>
    {#if suggestionAlreadyApplied}
      <Badge variant="outline" class="shrink-0 text-[10px]">Applied</Badge>
    {:else}
      <Button size="sm" class="h-6 shrink-0 px-2.5 text-[11px]" onclick={() => onApply?.()}>
        Apply
      </Button>
    {/if}
  </div>

  <StructuredDiffPreview {preview} />
</div>
