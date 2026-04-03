<script lang="ts">
  import { ChatMarkdownContent } from '$lib/features/chat'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { ChevronRight, Sparkles } from '@lucide/svelte'
  import {
    completionSummaryClass,
    completionSummaryLabel,
  } from './ticket-run-history-panel-view-model'
  import type { TicketRun } from '../types'

  let { run }: { run: TicketRun } = $props()

  let summaryExpanded = $state(true)
  let summaryJustCompleted = $state(false)
  let prevSummaryStatus = ''

  $effect(() => {
    const status = run.completionSummary?.status ?? ''
    if (status === 'completed' && prevSummaryStatus && prevSummaryStatus !== 'completed') {
      summaryJustCompleted = true
      summaryExpanded = true
    }
    prevSummaryStatus = status
  })
</script>

<div
  class={cn(
    'border-border/50 rounded-md border',
    completionSummaryClass(run),
    summaryJustCompleted && 'border-glow-enter',
  )}
  onanimationend={() => (summaryJustCompleted = false)}
>
  <button
    type="button"
    class="hover:bg-muted/30 flex w-full items-center gap-1.5 px-2.5 py-1.5 text-left text-[11px] transition-colors"
    onclick={() => (summaryExpanded = !summaryExpanded)}
  >
    <ChevronRight
      class={cn(
        'text-muted-foreground size-3 shrink-0 transition-transform duration-150',
        summaryExpanded && 'rotate-90',
      )}
    />
    <Sparkles
      class={cn('size-3 shrink-0 text-amber-400', summaryJustCompleted && 'sparkle-enter')}
    />
    <span class="font-medium">{completionSummaryLabel(run)}</span>
    {#if run.completionSummary?.generatedAt}
      <span class="text-muted-foreground/60 text-[10px]">
        {formatRelativeTime(run.completionSummary.generatedAt)}
      </span>
    {/if}
  </button>

  {#if summaryExpanded}
    <div class={cn('border-border/50 border-t px-3 py-2', summaryJustCompleted && 'summary-enter')}>
      {#if run.completionSummary?.status === 'completed' && run.completionSummary.markdown}
        <ChatMarkdownContent source={run.completionSummary.markdown} class="text-xs" />
      {:else if run.completionSummary?.status === 'failed'}
        <p class="text-muted-foreground text-xs">
          {run.completionSummary.error || 'Post-run summary generation failed.'}
        </p>
      {:else}
        <p class="text-muted-foreground text-xs">Generating post-run summary...</p>
      {/if}
    </div>
  {/if}
</div>

<style>
  @keyframes summary-appear {
    0% {
      opacity: 0;
      transform: translateY(-4px);
    }
    100% {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes sparkle-pop {
    0% {
      transform: scale(1) rotate(0deg);
    }
    30% {
      transform: scale(1.6) rotate(-12deg);
    }
    60% {
      transform: scale(0.9) rotate(6deg);
    }
    100% {
      transform: scale(1) rotate(0deg);
    }
  }

  @keyframes border-glow {
    0% {
      border-color: rgb(251 191 36 / 0.5);
      box-shadow: 0 0 8px rgb(251 191 36 / 0.15);
    }
    100% {
      border-color: var(--color-border);
      box-shadow: none;
    }
  }

  .summary-enter {
    animation: summary-appear 0.35s ease-out both;
  }

  .sparkle-enter {
    animation: sparkle-pop 0.5s ease-out 0.1s both;
  }

  .border-glow-enter {
    animation: border-glow 1.2s ease-out both;
  }
</style>
