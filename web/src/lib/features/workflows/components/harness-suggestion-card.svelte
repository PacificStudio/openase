<script lang="ts">
  import type { DiffPreview, HarnessSuggestion } from '../assistant'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import HarnessDiffPreview from './harness-diff-preview.svelte'

  let {
    suggestion,
    preview,
    suggestionAlreadyApplied,
    onApply,
  }: {
    suggestion: HarnessSuggestion
    preview: DiffPreview
    suggestionAlreadyApplied: boolean
    onApply?: () => void
  } = $props()
</script>

<div class="space-y-3 rounded-2xl border border-sky-500/30 bg-sky-500/8 p-3">
  <div class="flex items-start justify-between gap-3">
    <div>
      <div class="text-sm font-medium">Suggested Harness Update</div>
      <p class="text-muted-foreground mt-1 text-xs leading-5">{suggestion.summary}</p>
    </div>
    {#if suggestionAlreadyApplied}
      <Badge variant="outline" class="text-[10px]">Applied</Badge>
    {/if}
  </div>

  <HarnessDiffPreview {preview} />

  <Button size="sm" class="w-full" onclick={() => onApply?.()} disabled={suggestionAlreadyApplied}>
    Apply to Editor
  </Button>
</div>
