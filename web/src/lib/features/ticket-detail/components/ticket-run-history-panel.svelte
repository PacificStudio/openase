<!-- eslint-disable max-lines -->
<script lang="ts">
  import { tick } from 'svelte'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { ChevronRight, Layers } from '@lucide/svelte'
  import type { StreamConnectionState } from '$lib/api/sse'
  import TicketRunTranscriptDiffCard from './ticket-run-transcript-diff-card.svelte'
  import TicketRunTranscriptInterruptCard from './ticket-run-transcript-interrupt-card.svelte'
  import TicketRunTranscriptOutputBlock from './ticket-run-transcript-output-block.svelte'
  import TicketRunTranscriptStatusCard from './ticket-run-transcript-status-card.svelte'
  import TicketRunTranscriptToolCallCard from './ticket-run-transcript-tool-call-card.svelte'
  import {
    blockCardClass,
    blockLabel,
    blockLabelClass,
    connectionLabel,
    connectionTone,
  } from './ticket-run-transcript-view'
  import { groupRunTranscriptBlocks, type NoiseGroup } from '../run-transcript-grouping'
  import type { TicketDetail, TicketRun, TicketRunTranscriptBlock } from '../types'

  const autoFollowThresholdPx = 80

  let {
    ticket,
    runs = [],
    currentRun = null,
    blocks = [],
    loadingRunId = null,
    runStreamState = 'idle',
    recoveringRunTranscript = false,
    resumingRetry = false,
    onSelectRun,
    onResumeRetry,
  }: {
    ticket: TicketDetail
    runs?: TicketRun[]
    currentRun?: TicketRun | null
    blocks?: TicketRunTranscriptBlock[]
    loadingRunId?: string | null
    runStreamState?: StreamConnectionState
    recoveringRunTranscript?: boolean
    resumingRetry?: boolean
    onSelectRun?: (runId: string) => Promise<void> | void
    onResumeRetry?: () => Promise<void> | void
  } = $props()

  let sectionEl = $state<HTMLElement | null>(null)
  let expandedOutputIds = $state<string[]>([])
  let expandedNoiseGroups = $state(new Set<string>())
  let autoFollow = $state(true)
  let showJumpToLive = $state(false)
  let previousSelectionKey = ''
  let previousRenderKey = ''

  /** Walk up the DOM to find the nearest scrollable ancestor. */
  function getScrollViewport(): HTMLElement | null {
    let el = sectionEl?.parentElement ?? null
    while (el) {
      const style = getComputedStyle(el)
      if (
        (style.overflowY === 'auto' || style.overflowY === 'scroll') &&
        el.scrollHeight > el.clientHeight
      ) {
        return el
      }
      el = el.parentElement
    }
    return null
  }

  const latestRun = $derived(runs[0] ?? null)
  const chronologicalRuns = $derived([...runs].reverse())
  const liveSelected = $derived(currentRun != null && latestRun?.id === currentRun.id)
  const displayItems = $derived(groupRunTranscriptBlocks(blocks))

  const FINISHED_STATUSES = new Set(['completed', 'failed', 'stalled'])

  const canResumeRetry = $derived(
    ticket.retryPaused &&
      ticket.pauseReason === 'repeated_stalls' &&
      latestRun?.id === currentRun?.id,
  )

  function statusLabel(run: TicketRun) {
    if (run.status === 'completed') return 'Completed'
    if (run.status === 'failed') return 'Failed'
    if (run.status === 'stalled') return 'Stalled'
    if ((run.currentStepStatus ?? '').toLowerCase().includes('approval')) return 'Awaiting Approval'
    if ((run.currentStepStatus ?? '').toLowerCase().includes('input')) return 'Waiting Input'
    if (run.status === 'launching') return 'Launching'
    return 'Running'
  }

  function statusClass(run: TicketRun) {
    switch (run.status) {
      case 'completed':
        return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-600'
      case 'failed':
      case 'stalled':
        return 'border-red-500/30 bg-red-500/10 text-red-600'
      case 'ready':
      case 'executing':
        return 'border-sky-500/30 bg-sky-500/10 text-sky-600'
      default:
        return 'border-amber-500/30 bg-amber-500/10 text-amber-600'
    }
  }

  function summaryLine(run: TicketRun) {
    return (
      run.currentStepSummary ||
      run.currentStepStatus ||
      (run.completedAt
        ? `Completed ${formatRelativeTime(run.completedAt)}`
        : run.lastHeartbeatAt
          ? `Updated ${formatRelativeTime(run.lastHeartbeatAt)}`
          : `Started ${formatRelativeTime(run.createdAt)}`)
    )
  }

  function isOutputExpanded(blockId: string) {
    return expandedOutputIds.includes(blockId)
  }

  async function toggleOutputExpansion(blockId: string) {
    const viewport = getScrollViewport()
    const bottomOffset = viewport
      ? Math.max(viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop, 0)
      : 0

    expandedOutputIds = isOutputExpanded(blockId)
      ? expandedOutputIds.filter((item) => item !== blockId)
      : [...expandedOutputIds, blockId]

    await tick()

    if (!viewport) return
    if (liveSelected && autoFollow) {
      jumpToBottom()
      return
    }
    viewport.scrollTop = Math.max(viewport.scrollHeight - viewport.clientHeight - bottomOffset, 0)
  }

  function toggleNoiseGroup(groupId: string) {
    const next = new Set(expandedNoiseGroups)
    if (next.has(groupId)) {
      next.delete(groupId)
    } else {
      next.add(groupId)
    }
    expandedNoiseGroups = next
  }

  function selectRun(runId: string) {
    if (currentRun?.id === runId) return
    expandedOutputIds = []
    expandedNoiseGroups = new Set()
    void onSelectRun?.(runId)
  }

  function jumpToBottom() {
    const viewport = getScrollViewport()
    if (!viewport) return
    viewport.scrollTop = viewport.scrollHeight
  }

  async function jumpToLive() {
    autoFollow = true
    showJumpToLive = false
    await tick()
    jumpToBottom()
  }

  function isNearBottom() {
    const viewport = getScrollViewport()
    if (!viewport) return true
    return (
      viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop <= autoFollowThresholdPx
    )
  }

  function handleScroll() {
    if (!liveSelected) {
      showJumpToLive = false
      return
    }
    autoFollow = isNearBottom()
    showJumpToLive = !autoFollow
  }

  function renderKeyForBlocks(blocks: TicketRunTranscriptBlock[]) {
    const tail = blocks.at(-1)
    if (!tail) return 'empty'
    switch (tail.kind) {
      case 'assistant_message':
      case 'terminal_output':
        return `${tail.id}:${tail.text.length}:${tail.streaming}`
      case 'tool_call':
        return `${tail.id}:${tail.toolName}:${JSON.stringify(tail.arguments ?? null)}`
      case 'task_status':
        return `${tail.id}:${tail.statusType}:${tail.detail ?? ''}`
      case 'diff':
        return `${tail.id}:${tail.diff.file}:${tail.diff.hunks.length}`
      case 'interrupt':
        return `${tail.id}:${tail.summary}:${tail.options.length}`
      case 'phase':
      case 'step':
      case 'result':
        return `${tail.id}:${tail.summary}`
    }
  }

  /** Attach scroll handler to the scrollable ancestor. */
  $effect(() => {
    if (!sectionEl) return
    const viewport = getScrollViewport()
    if (!viewport) return

    viewport.addEventListener('scroll', handleScroll)
    return () => viewport.removeEventListener('scroll', handleScroll)
  })

  $effect(() => {
    const selectionKey = `${currentRun?.id ?? 'none'}:${latestRun?.id ?? 'none'}`
    if (selectionKey === previousSelectionKey) return
    previousSelectionKey = selectionKey
    autoFollow = liveSelected
    showJumpToLive = false
    expandedOutputIds = []
  })

  $effect(() => {
    const shouldStickToBottom = liveSelected && autoFollow && isNearBottom()
    const nextRenderKey = `${currentRun?.id ?? 'none'}:${blocks.length}:${renderKeyForBlocks(blocks)}:${runStreamState}:${recoveringRunTranscript}`
    if (nextRenderKey === previousRenderKey) return
    previousRenderKey = nextRenderKey
    void tick().then(() => {
      if (shouldStickToBottom) {
        jumpToBottom()
      } else if (liveSelected) {
        showJumpToLive = !isNearBottom()
      }
    })
  })
</script>

{#if runs.length === 0}
  <section class="px-4 py-6">
    <p class="text-muted-foreground text-xs">No runs yet.</p>
  </section>
{:else}
  <section bind:this={sectionEl}>
    {#each chronologicalRuns as run (run.id)}
      {@const selected = currentRun?.id === run.id}
      {@const live = latestRun?.id === run.id && !FINISHED_STATUSES.has(run.status)}
      {@const loading = loadingRunId === run.id}

      <!-- Attempt header -->
      <button
        type="button"
        class={cn(
          'border-border flex w-full items-center gap-2 border-b px-4 py-1.5 text-left text-xs transition',
          selected ? 'bg-muted sticky top-0 z-10' : 'hover:bg-muted/50',
        )}
        aria-label={`View Attempt ${run.attemptNumber}`}
        aria-pressed={selected}
        onclick={() => selectRun(run.id)}
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
          class={cn('h-4 shrink-0 px-1.5 py-0 text-[9px]', statusClass(run))}
        >
          {statusLabel(run)}
        </Badge>
        {#if live}
          <span class="size-1.5 shrink-0 rounded-full bg-green-400"></span>
        {/if}
        <span class="text-muted-foreground min-w-0 flex-1 truncate">{summaryLine(run)}</span>
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

      <!-- Expanded transcript content -->
      {#if selected}
        <div class="px-4 py-3">
          {#if loading}
            <p class="text-muted-foreground text-xs">Loading transcript…</p>
          {:else if displayItems.length === 0}
            <p class="text-muted-foreground text-xs">Waiting for transcript events…</p>
          {:else}
            <div class="space-y-2">
              {#each displayItems as item (item.type === 'content' ? item.block.id : item.id)}
                {#if item.type === 'noise_group'}
                  <!-- Collapsible noise group -->
                  {@const group = item as NoiseGroup}
                  {@const isExpanded = expandedNoiseGroups.has(group.id)}
                  <div class="border-border/50 bg-muted/10 rounded-md border">
                    <button
                      type="button"
                      class="hover:bg-muted/30 flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-[11px] transition-colors"
                      onclick={() => toggleNoiseGroup(group.id)}
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
                            <span class="text-[10px] font-medium tracking-wider uppercase"
                              >{blockLabel(b)}</span
                            >
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
                  <!-- Content block -->
                  {@const block = item.block}
                  {#if block.kind === 'assistant_message'}
                    <div class="prose prose-sm prose-neutral max-w-none break-words">
                      <TicketRunTranscriptOutputBlock
                        {block}
                        expanded={isOutputExpanded(block.id)}
                        onToggle={() => toggleOutputExpansion(block.id)}
                      />
                    </div>
                  {:else if block.kind === 'tool_call'}
                    <TicketRunTranscriptToolCallCard {block} />
                  {:else if block.kind === 'terminal_output'}
                    <TicketRunTranscriptOutputBlock
                      {block}
                      expanded={isOutputExpanded(block.id)}
                      onToggle={() => toggleOutputExpansion(block.id)}
                    />
                  {:else if block.kind === 'task_status'}
                    <TicketRunTranscriptStatusCard {block} />
                  {:else if block.kind === 'diff'}
                    <TicketRunTranscriptDiffCard {block} />
                  {:else if block.kind === 'interrupt'}
                    <article class={cn('rounded-md border px-3 py-2', blockCardClass(block))}>
                      <TicketRunTranscriptInterruptCard {block} />
                    </article>
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
                    <article
                      class={cn('rounded-md border px-3 py-2 text-xs', blockCardClass(block))}
                    >
                      <span class={blockLabelClass(block)}>{blockLabel(block)}</span>
                      <span class="ml-2">{block.summary}</span>
                    </article>
                  {/if}
                {/if}
              {/each}
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
                {resumingRetry ? 'Continuing…' : 'Continue Retry'}
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
              onclick={() => void jumpToLive()}
            >
              Jump to live
            </Button>
          </div>
        {/if}
      {/if}
    {/each}
  </section>
{/if}
