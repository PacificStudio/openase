<script lang="ts">
  import { ChatMarkdownContent } from '$lib/features/chat'
  import type { StreamConnectionState } from '$lib/api/sse'
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { ChevronRight, Layers } from '@lucide/svelte'
  import TicketRunErrorCard from './ticket-run-error-card.svelte'
  import TicketRunHistorySummaryCard from './ticket-run-history-summary-card.svelte'
  import TicketRunTranscriptDiffCard from './ticket-run-transcript-diff-card.svelte'
  import TicketRunTranscriptInterruptCard from './ticket-run-transcript-interrupt-card.svelte'
  import TicketRunTranscriptOutputBlock from './ticket-run-transcript-output-block.svelte'
  import TicketRunTranscriptStatusCard from './ticket-run-transcript-status-card.svelte'
  import TicketRunTranscriptToolCallCard from './ticket-run-transcript-tool-call-card.svelte'
  import {
    blockCardClass,
    blockLabel,
    blockLabelClass,
    blockTimestamp,
    connectionLabel,
    connectionTone,
  } from './ticket-run-transcript-view'
  import {
    ticketRunStatusClass,
    ticketRunStatusLabel,
    ticketRunSummaryLine,
  } from './ticket-run-history-panel-view-model'
  import { groupRunTranscriptBlocks, type NoiseGroup } from '../run-transcript-grouping'
  import type { TicketDetail, TicketRun, TicketRunTranscriptBlock } from '../types'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    run,
    selected = false,
    live = false,
    blocks = [],
    loading = false,
    hasOlderHistory = false,
    hiddenOlderCount = 0,
    loadingOlderHistory = false,
    runStreamState = 'idle',
    recoveringRunTranscript = false,
    liveSelected = false,
    expandedOutputIds = [],
    expandedNoiseGroups = new Set<string>(),
    canResumeRetry = false,
    resumingRetry = false,
    showJumpToLive = false,
    onSelectRun,
    onLoadOlderHistory,
    onToggleOutput,
    onToggleNoiseGroup,
    onResumeRetry,
    onJumpToLive,
  }: {
    ticket?: TicketDetail
    run: TicketRun
    selected?: boolean
    live?: boolean
    blocks?: TicketRunTranscriptBlock[]
    loading?: boolean
    hasOlderHistory?: boolean
    hiddenOlderCount?: number
    loadingOlderHistory?: boolean
    runStreamState?: StreamConnectionState
    recoveringRunTranscript?: boolean
    liveSelected?: boolean
    expandedOutputIds?: string[]
    expandedNoiseGroups?: Set<string>
    canResumeRetry?: boolean
    resumingRetry?: boolean
    showJumpToLive?: boolean
    onSelectRun?: (runId: string) => void
    onLoadOlderHistory?: (runId: string) => void | Promise<void>
    onToggleOutput?: (blockId: string) => void | Promise<void>
    onToggleNoiseGroup?: (groupId: string) => void
    onResumeRetry?: () => void | Promise<void>
    onJumpToLive?: () => void | Promise<void>
  } = $props()

  const displayItems = $derived(groupRunTranscriptBlocks(blocks))
  const isOutputExpanded = (blockId: string) => expandedOutputIds.includes(blockId)
  const t = i18nStore.t
</script>

<button
  type="button"
  class={cn(
    'border-border flex w-full items-center gap-2 border-b px-4 py-1.5 text-left text-xs transition',
    selected ? 'bg-muted sticky top-0 z-10' : 'hover:bg-muted/50',
  )}
  aria-label={t('ticketDetail.runHistoryAttempt.actions.viewAttempt', {
    attemptNumber: run.attemptNumber,
  })}
  aria-pressed={selected}
  onclick={() => onSelectRun?.(run.id)}
>
  <ChevronRight
    class={cn(
      'text-muted-foreground size-3 shrink-0 transition-transform duration-150',
      selected && 'rotate-90',
    )}
  />
  <span class="font-medium">#{run.attemptNumber}</span>
  <Badge
    variant="outline"
    class={cn('h-4 shrink-0 px-1.5 py-0 text-[9px]', ticketRunStatusClass(run))}
  >
    {ticketRunStatusLabel(run)}
  </Badge>
  {#if live}
    <span class="size-1.5 shrink-0 rounded-full bg-green-400"></span>
  {/if}
  <span
    class="text-muted-foreground min-w-0 flex-1 truncate hover:break-words hover:whitespace-normal"
    title={ticketRunSummaryLine(run)}>{ticketRunSummaryLine(run)}</span
  >
  <span class="text-muted-foreground shrink-0 text-[10px]">{run.provider}</span>
  {#if selected && connectionLabel(runStreamState, recoveringRunTranscript, liveSelected)}
    <Badge
      variant="outline"
      class={cn(
        'h-4 shrink-0 px-1.5 py-0 text-[9px]',
        connectionTone(runStreamState, recoveringRunTranscript),
      )}
    >
      {connectionLabel(runStreamState, recoveringRunTranscript, liveSelected)}
    </Badge>
  {/if}
</button>

{#if selected}
  <div class="px-4 py-3" data-run-content={run.id}>
    {#if loading}
      <p class="text-muted-foreground text-xs">
        {t('ticketDetail.runHistoryAttempt.loadingTranscript')}
      </p>
    {:else}
      <div class="space-y-3">
        <TicketRunErrorCard {run} />

        {#if run.completionSummary}
          <TicketRunHistorySummaryCard {run} />
        {/if}

        {#if hasOlderHistory}
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground w-full rounded-md border border-dashed px-3 py-2 text-left text-[11px] transition"
            disabled={loadingOlderHistory}
            onclick={() => void onLoadOlderHistory?.(run.id)}
          >
            {loadingOlderHistory
              ? t('ticketDetail.runHistoryAttempt.actions.loadingEarlierEvents')
              : t('ticketDetail.runHistoryAttempt.actions.hiddenEvents', {
                  count: hiddenOlderCount,
                })}
          </button>
        {/if}

        {#if displayItems.length === 0}
          <p class="text-muted-foreground text-xs">
            {t('ticketDetail.runHistoryAttempt.status.waiting')}
          </p>
        {:else}
          <div class="space-y-2">
            {#each displayItems as item (item.type === 'content' ? item.block.id : item.id)}
              {#if item.type === 'noise_group'}
                {@const group = item as NoiseGroup}
                {@const isExpanded = expandedNoiseGroups.has(group.id)}
                <div class="border-border/50 bg-muted/10 rounded-md border">
                  <button
                    type="button"
                    class="hover:bg-muted/30 flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-[11px] transition-colors"
                    onclick={() => onToggleNoiseGroup?.(group.id)}
                  >
                    <ChevronRight
                      class={cn(
                        'text-muted-foreground size-3 shrink-0 transition-transform duration-150',
                        isExpanded && 'rotate-90',
                      )}
                    />
                    <Layers class="text-muted-foreground/70 size-3 shrink-0" />
                    <span class="text-foreground min-w-0 flex-1 truncate font-medium">
                      {group.summary}
                    </span>
                    {#if group.detail}
                      <span class="text-muted-foreground/60 shrink-0 text-[10px]"
                        >{group.detail}</span
                      >
                    {/if}
                  </button>
                  {#if isExpanded}
                    <div class="border-border/30 space-y-1 border-t px-2.5 py-1.5 text-xs">
                      {#each group.blocks as b (b.id)}
                        <div class="text-muted-foreground flex items-center gap-2">
                          <span class="text-[10px] font-medium tracking-wider uppercase">
                            {blockLabel(b)}
                          </span>
                          <span class="truncate"
                            >{b.kind === 'tool_call'
                              ? b.toolName
                              : 'summary' in b
                                ? b.summary
                                : ''}</span
                          >
                        </div>
                      {/each}
                    </div>
                  {/if}
                </div>
              {:else}
                {@const block = item.block}
                {#if block.kind === 'assistant_message'}
                  <TicketRunTranscriptOutputBlock
                    {block}
                    expanded={isOutputExpanded(block.id)}
                    onToggle={() => onToggleOutput?.(block.id)}
                  />
                {:else if block.kind === 'tool_call'}
                  <TicketRunTranscriptToolCallCard {block} />
                {:else if block.kind === 'terminal_output'}
                  <TicketRunTranscriptOutputBlock
                    {block}
                    expanded={isOutputExpanded(block.id)}
                    onToggle={() => onToggleOutput?.(block.id)}
                  />
                {:else if block.kind === 'task_status'}
                  <TicketRunTranscriptStatusCard {block} />
                {:else if block.kind === 'diff'}
                  <TicketRunTranscriptDiffCard {block} />
                {:else if block.kind === 'interrupt'}
                  <article class={cn('rounded-md border px-3 py-2', blockCardClass(block))}>
                    <TicketRunTranscriptInterruptCard {block} />
                  </article>
                {:else if block.kind === 'step'}
                  {#if block.stepStatus === 'commentary'}
                    <ChatMarkdownContent source={block.summary} class="text-xs" />
                  {:else}
                    <div class="text-muted-foreground flex items-center gap-2 py-0.5 text-[11px]">
                      <span class="bg-muted rounded px-1.5 py-0.5 text-[10px] font-medium"
                        >{block.stepStatus}</span
                      >
                      {#if block.summary}
                        <span class="min-w-0 truncate">{block.summary}</span>
                      {/if}
                      <span class="text-muted-foreground/50 shrink-0 text-[10px]"
                        >{blockTimestamp(block)}</span
                      >
                    </div>
                  {/if}
                {:else if block.kind === 'result'}
                  <article
                    class={cn(
                      'flex items-center gap-2 rounded-md border px-3 py-2 text-xs',
                      blockCardClass(block),
                    )}
                  >
                    <span class="font-medium">{blockLabel(block)}</span>
                    <span>{block.summary}</span>
                  </article>
                {:else}
                  <article class={cn('rounded-md border px-3 py-2 text-xs', blockCardClass(block))}>
                    <span class={blockLabelClass(block)}>{blockLabel(block)}</span>
                    <span class="ml-2">{block.summary}</span>
                  </article>
                {/if}
              {/if}
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    {#if canResumeRetry && onResumeRetry}
      <div class="mt-2">
        <Button
          size="sm"
          variant="outline"
          class="h-6 px-2 text-[11px]"
          disabled={resumingRetry}
          onclick={() => void onResumeRetry()}
        >
          {resumingRetry
            ? t('ticketDetail.runHistoryAttempt.actions.continuing')
            : t('ticketDetail.runHistoryAttempt.actions.resumeRetry')}
        </Button>
      </div>
    {/if}
  </div>

  {#if liveSelected && showJumpToLive}
    <div
      class="border-border bg-background/95 sticky bottom-0 z-10 border-t px-4 py-1.5 backdrop-blur"
    >
      <Button
        size="sm"
        variant="outline"
        class="h-6 px-2 text-[11px]"
        onclick={() => void onJumpToLive?.()}
      >
        {t('ticketDetail.runHistoryAttempt.actions.jumpToLive')}
      </Button>
    </div>
  {/if}
{/if}
