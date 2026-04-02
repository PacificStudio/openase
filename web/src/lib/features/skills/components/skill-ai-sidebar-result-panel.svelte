<script lang="ts">
  import type { SkillRefinementResultPayload } from '$lib/api/skill-refinement'
  import type { DiffPreview, SkillSuggestion } from '$lib/features/skills/assistant'
  import { Sparkles } from '@lucide/svelte'
  import SkillSuggestionCard from './skill-suggestion-card.svelte'

  let {
    result,
    suggestion,
    preview,
    dismissed,
    selectedSuggestionPath,
    suggestionAlreadyApplied,
    onSelectPath,
    onApply,
    onDismiss,
  }: {
    result: SkillRefinementResultPayload | null
    suggestion: SkillSuggestion | null
    preview: DiffPreview | null
    dismissed: boolean
    selectedSuggestionPath: string
    suggestionAlreadyApplied: boolean
    onSelectPath?: (path: string) => void
    onApply?: () => void
    onDismiss?: () => void
  } = $props()
</script>

{#if result}
  <div class="space-y-3">
    {#if result.status === 'verified' && suggestion && preview && !dismissed}
      <SkillSuggestionCard
        {suggestion}
        selectedPath={selectedSuggestionPath}
        {preview}
        {suggestionAlreadyApplied}
        {onSelectPath}
        {onApply}
        {onDismiss}
      />
    {/if}

    {#if result.failureReason}
      <div class="rounded-lg border border-rose-500/30 bg-rose-500/8 p-3">
        <p class="text-[11px] font-medium tracking-[0.18em] text-rose-200 uppercase">Failure</p>
        <p class="mt-2 text-xs leading-5 whitespace-pre-wrap text-rose-50">
          {result.failureReason}
        </p>
      </div>
    {/if}

    {#if result.transcriptSummary}
      <div class="rounded-lg border border-white/8 bg-white/4 p-3">
        <div class="flex items-center gap-2">
          <Sparkles class="size-3.5 text-sky-200" />
          <p class="text-[11px] font-medium tracking-[0.18em] uppercase">Transcript Summary</p>
        </div>
        <p class="mt-2 text-xs leading-5 whitespace-pre-wrap">{result.transcriptSummary}</p>
      </div>
    {/if}

    {#if result.commandOutputSummary}
      <div class="rounded-lg border border-white/8 bg-black/20 p-3">
        <p class="text-muted-foreground text-[11px] font-medium tracking-[0.18em] uppercase">
          Verification Output
        </p>
        <pre class="mt-2 font-mono text-[11px] leading-5 break-words whitespace-pre-wrap">
{result.commandOutputSummary}</pre>
      </div>
    {/if}
  </div>
{/if}
