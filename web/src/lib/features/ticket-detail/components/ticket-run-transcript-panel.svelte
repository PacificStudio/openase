<script lang="ts">
  import { tick } from 'svelte'
  import type { StreamConnectionState } from '$lib/api/sse'
  import TicketRunTranscriptHeader from './ticket-run-transcript-header.svelte'
  import TicketRunTranscriptDiffCard from './ticket-run-transcript-diff-card.svelte'
  import TicketRunTranscriptInterruptCard from './ticket-run-transcript-interrupt-card.svelte'
  import TicketRunTranscriptOutputBlock from './ticket-run-transcript-output-block.svelte'
  import TicketRunTranscriptStatusCard from './ticket-run-transcript-status-card.svelte'
  import TicketRunTranscriptToolCallCard from './ticket-run-transcript-tool-call-card.svelte'
  import {
    blockCardClass,
    blockLabel,
    blockLabelClass,
    blockMeta,
    blockMutedTextClass,
    blockTimestamp,
  } from './ticket-run-transcript-view'
  import type { TicketRun, TicketRunTranscriptBlock } from '../types'

  const autoFollowThresholdPx = 80

  let {
    run = null,
    blocks = [],
    latestRunId = null,
    loading = false,
    streamState = 'idle',
    recovering = false,
    canResumeRetry = false,
    resumingRetry = false,
    onResumeRetry,
  }: {
    run?: TicketRun | null
    blocks?: TicketRunTranscriptBlock[]
    latestRunId?: string | null
    loading?: boolean
    streamState?: StreamConnectionState
    recovering?: boolean
    canResumeRetry?: boolean
    resumingRetry?: boolean
    onResumeRetry?: () => Promise<void> | void
  } = $props()

  let scrollViewport = $state<HTMLDivElement | null>(null)
  let expandedBlockIds = $state<string[]>([])
  let autoFollow = $state(true)
  let showJumpToLive = $state(false)
  let previousSelectionKey = ''
  let previousRenderKey = ''

  const liveSelected = $derived(run != null && latestRunId === run.id)

  function isExpanded(blockId: string) {
    return expandedBlockIds.includes(blockId)
  }

  async function toggleOutputExpansion(blockId: string) {
    const viewport = scrollViewport
    const bottomOffset = viewport
      ? Math.max(viewport.scrollHeight - viewport.clientHeight - viewport.scrollTop, 0)
      : 0

    expandedBlockIds = isExpanded(blockId)
      ? expandedBlockIds.filter((item) => item !== blockId)
      : [...expandedBlockIds, blockId]

    await tick()

    if (!viewport) {
      return
    }

    if (liveSelected && autoFollow) {
      jumpToBottom()
      return
    }

    viewport.scrollTop = Math.max(viewport.scrollHeight - viewport.clientHeight - bottomOffset, 0)
  }

  function jumpToBottom() {
    if (!scrollViewport) {
      return
    }
    scrollViewport.scrollTop = scrollViewport.scrollHeight
  }

  async function jumpToLive() {
    autoFollow = true
    showJumpToLive = false
    await tick()
    jumpToBottom()
  }

  function isNearBottom() {
    if (!scrollViewport) {
      return true
    }
    return (
      scrollViewport.scrollHeight - scrollViewport.clientHeight - scrollViewport.scrollTop <=
      autoFollowThresholdPx
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
    if (!tail) {
      return 'empty'
    }
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

  $effect(() => {
    const selectionKey = `${run?.id ?? 'none'}:${latestRunId ?? 'none'}`
    if (selectionKey === previousSelectionKey) {
      return
    }

    previousSelectionKey = selectionKey
    autoFollow = liveSelected
    showJumpToLive = false
    expandedBlockIds = []
  })

  $effect(() => {
    const shouldStickToBottom = liveSelected && autoFollow && isNearBottom()
    const nextRenderKey = `${run?.id ?? 'none'}:${blocks.length}:${renderKeyForBlocks(blocks)}:${streamState}:${recovering}`
    if (nextRenderKey === previousRenderKey) {
      return
    }

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

<section class="relative flex min-h-0 flex-1 flex-col overflow-hidden">
  <TicketRunTranscriptHeader
    {run}
    {latestRunId}
    {liveSelected}
    {showJumpToLive}
    {loading}
    {streamState}
    {recovering}
    {canResumeRetry}
    {resumingRetry}
    onJumpToLive={jumpToLive}
    {onResumeRetry}
  />

  <div
    bind:this={scrollViewport}
    class="min-h-0 flex-1 overflow-y-auto px-5 py-4"
    onscroll={handleScroll}
  >
    {#if run}
      <div class="space-y-3">
        {#each blocks as block (block.id)}
          <article class={`rounded-2xl border px-4 py-3 ${blockCardClass(block)}`}>
            <div class="mb-2 flex items-center justify-between gap-3 text-[11px]">
              <div class="flex min-w-0 items-center gap-2">
                <span class={blockLabelClass(block)}>{blockLabel(block)}</span>
                {#if blockMeta(block)}
                  <span class={blockMutedTextClass(block)}>{blockMeta(block)}</span>
                {/if}
              </div>
              <div class="flex items-center gap-2">
                {#if block.kind === 'assistant_message' || block.kind === 'terminal_output'}
                  {#if block.streaming}
                    <span class={blockMutedTextClass(block)}>Streaming</span>
                  {/if}
                {/if}
                {#if blockTimestamp(block)}
                  <span class={blockMutedTextClass(block)}>{blockTimestamp(block)}</span>
                {/if}
              </div>
            </div>

            {#if block.kind === 'assistant_message' || block.kind === 'terminal_output'}
              <TicketRunTranscriptOutputBlock
                {block}
                expanded={isExpanded(block.id)}
                onToggle={() => toggleOutputExpansion(block.id)}
              />
            {:else if block.kind === 'tool_call'}
              <TicketRunTranscriptToolCallCard {block} />
            {:else if block.kind === 'task_status'}
              <TicketRunTranscriptStatusCard {block} />
            {:else if block.kind === 'diff'}
              <TicketRunTranscriptDiffCard {block} />
            {:else if block.kind === 'interrupt'}
              <TicketRunTranscriptInterruptCard {block} />
            {:else}
              <p class="text-foreground text-sm">{block.summary}</p>
            {/if}
          </article>
        {/each}

        {#if blocks.length === 0}
          <div class="text-muted-foreground rounded-xl border border-dashed px-3 py-4 text-sm">
            Run created, waiting for transcript events.
          </div>
        {/if}
      </div>
    {:else}
      <div class="text-muted-foreground rounded-xl border border-dashed px-4 py-5 text-sm">
        No execution transcript yet.
      </div>
    {/if}
  </div>
</section>
