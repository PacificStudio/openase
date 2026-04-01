<script lang="ts">
  import { ChevronDown, ChevronUp } from '@lucide/svelte'
  import type { HarnessValidationIssue } from '$lib/api/contracts'

  let {
    validationIssues,
  }: {
    validationIssues: HarnessValidationIssue[]
  } = $props()

  let issuesExpanded = $state(false)
</script>

<div class="border-border border-t">
  <button
    type="button"
    class="flex w-full items-center gap-2 px-3 py-1.5 text-xs text-amber-400 hover:bg-amber-500/5"
    onclick={() => (issuesExpanded = !issuesExpanded)}
  >
    {#if issuesExpanded}
      <ChevronDown class="size-3" />
    {:else}
      <ChevronUp class="size-3" />
    {/if}
    <span class="font-medium"
      >{validationIssues.length} validation issue{validationIssues.length > 1 ? 's' : ''}</span
    >
    {#if !issuesExpanded}
      <span class="text-muted-foreground truncate">— {validationIssues[0].message}</span>
    {/if}
  </button>

  {#if issuesExpanded}
    <div
      class="max-h-32 space-y-1 overflow-y-auto border-t border-amber-500/20 bg-amber-500/5 px-3 py-2 text-xs text-amber-200"
    >
      {#each validationIssues as issue, index (index)}
        <div>
          {issue.level?.toUpperCase() ?? 'ISSUE'}: {issue.message}
          {#if issue.line}
            at line {issue.line}
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
