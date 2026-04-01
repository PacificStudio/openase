<script lang="ts">
  import { tick } from 'svelte'
  import type { StreamConnectionState } from '$lib/api/sse'
  import { Button } from '$ui/button'
  import { Badge } from '$ui/badge'
  import { formatRelativeTime } from '$lib/utils'
  import ChatMarkdownContent from '$lib/features/chat/chat-markdown-content.svelte'
  import type { TicketRun, TicketRunTranscriptBlock } from '../types'

  const autoFollowThresholdPx = 80
  const outputPreviewMaxLines = 14
  const outputPreviewMinChars = 720

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

  function statusLabel(run: TicketRun) {
    if (run.status === 'completed') return 'Completed'
    if (run.status === 'failed') return 'Failed'
    if (run.status === 'stalled') return 'Stalled'
    if ((run.currentStepStatus ?? '').toLowerCase().includes('approval')) return 'Awaiting Approval'
    if ((run.currentStepStatus ?? '').toLowerCase().includes('input')) return 'Waiting Input'
    if (run.status === 'launching') return 'Launching'
    return 'Running'
  }

  function statusTone(run: TicketRun) {
    switch (run.status) {
      case 'completed':
        return 'border-emerald-500/20 bg-emerald-500/10 text-emerald-700'
      case 'failed':
      case 'stalled':
        return 'border-red-500/20 bg-red-500/10 text-red-700'
      case 'ready':
      case 'executing':
        return 'border-sky-500/20 bg-sky-500/10 text-sky-700'
      default:
        return 'border-amber-500/20 bg-amber-500/10 text-amber-700'
    }
  }

  function blockLabel(block: TicketRunTranscriptBlock) {
    switch (block.kind) {
      case 'assistant_message':
        return 'Assistant'
      case 'terminal_output':
        return 'Terminal'
      case 'tool_call':
        return 'Tool'
      case 'step':
        return 'Step'
      case 'phase':
        return 'Phase'
      case 'interrupt':
        return 'Interrupt'
      case 'result':
        return 'Result'
    }
  }

  function blockCardClass(block: TicketRunTranscriptBlock) {
    switch (block.kind) {
      case 'assistant_message':
        return 'border-border bg-background'
      case 'terminal_output':
        return 'border-slate-400/20 bg-slate-950 text-slate-50'
      case 'tool_call':
        return 'border-sky-500/20 bg-sky-500/5'
      case 'interrupt':
        return 'border-amber-400/30 bg-amber-50/80'
      case 'result':
        switch (block.outcome) {
          case 'completed':
            return 'border-emerald-500/20 bg-emerald-500/5'
          case 'failed':
            return 'border-red-500/20 bg-red-500/5'
          default:
            return 'border-amber-500/20 bg-amber-500/5'
        }
      default:
        return 'border-border bg-muted/25'
    }
  }

  function blockMeta(block: TicketRunTranscriptBlock) {
    switch (block.kind) {
      case 'phase':
        return block.phase
      case 'step':
        return block.stepStatus
      case 'tool_call':
        return block.toolName
      case 'interrupt':
        return block.interruptKind.replace(/_/g, ' ')
      case 'result':
        return block.outcome
      default:
        return ''
    }
  }

  function blockTimestamp(block: TicketRunTranscriptBlock) {
    if (
      block.kind === 'phase' ||
      block.kind === 'step' ||
      block.kind === 'tool_call' ||
      block.kind === 'interrupt'
    ) {
      return formatRelativeTime(block.at)
    }
    return ''
  }

  function blockMutedTextClass(block: TicketRunTranscriptBlock) {
    return block.kind === 'terminal_output' ? 'text-slate-400' : 'text-muted-foreground'
  }

  function blockLabelClass(block: TicketRunTranscriptBlock) {
    return block.kind === 'terminal_output'
      ? 'font-medium uppercase tracking-[0.14em] text-slate-300'
      : 'text-muted-foreground font-medium uppercase tracking-[0.14em]'
  }

  function connectionLabel() {
    if (recovering) return 'Recovering transcript'
    switch (streamState) {
      case 'connecting':
        return 'Connecting'
      case 'retrying':
        return 'Reconnecting'
      case 'live':
        return liveSelected ? 'Live stream' : 'Connected'
      default:
        return ''
    }
  }

  function connectionTone() {
    if (recovering || streamState === 'retrying') {
      return 'border-amber-400/30 bg-amber-500/10 text-amber-700'
    }
    if (streamState === 'live') {
      return 'border-sky-500/30 bg-sky-500/10 text-sky-700'
    }
    return 'border-border bg-muted/40 text-muted-foreground'
  }

  function outputLineCount(text: string) {
    return text.split('\n').length
  }

  function isOutputExpandable(block: TicketRunTranscriptBlock) {
    return (
      block.kind === 'terminal_output' &&
      (block.text.length >= outputPreviewMinChars ||
        outputLineCount(block.text) > outputPreviewMaxLines)
    )
  }

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

  function interruptString(payload: Record<string, unknown>, ...keys: string[]) {
    for (const key of keys) {
      const value = payload[key]
      if (typeof value === 'string' && value.trim()) {
        return value
      }
    }
    return ''
  }

  function interruptQuestion(block: Extract<TicketRunTranscriptBlock, { kind: 'interrupt' }>) {
    const questions = block.payload.questions
    if (!Array.isArray(questions) || questions.length === 0) {
      return ''
    }
    const first = questions[0]
    if (!first || typeof first !== 'object') {
      return ''
    }
    const value = (first as Record<string, unknown>).question
    return typeof value === 'string' ? value : ''
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
        return `${tail.id}:${tail.toolName}:${tail.summary ?? ''}`
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
  <div class="bg-background/95 border-border sticky top-0 z-10 border-b px-5 py-4 backdrop-blur">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <div class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
          Run Detail
        </div>
        <h3 class="text-sm font-semibold">
          {#if run}
            Attempt {run.attemptNumber} transcript
          {:else}
            Run transcript
          {/if}
        </h3>
        <p class="text-muted-foreground text-xs">
          {#if run}
            {run.currentStepSummary ||
              run.currentStepStatus ||
              `Started ${formatRelativeTime(run.createdAt)}`}
          {:else}
            Select a run to inspect its execution transcript.
          {/if}
        </p>
      </div>

      <div class="flex flex-wrap items-center justify-end gap-2">
        {#if connectionLabel()}
          <Badge variant="outline" class={`h-6 px-2.5 text-[11px] ${connectionTone()}`}>
            {connectionLabel()}
          </Badge>
        {/if}
        {#if liveSelected && showJumpToLive}
          <Button size="sm" variant="outline" class="h-8 text-xs" onclick={() => void jumpToLive()}>
            Jump to live
          </Button>
        {/if}
        {#if run}
          {#if latestRunId === run.id}
            <Badge variant="outline" class="h-6 px-2.5 text-[11px]">Live</Badge>
          {/if}
          <Badge variant="outline" class={`h-6 px-2.5 text-[11px] ${statusTone(run)}`}>
            {statusLabel(run)}
          </Badge>
        {/if}
      </div>
    </div>

    {#if run}
      <div class="mt-3 flex flex-wrap items-center gap-2 text-[11px]">
        <Badge variant="outline" class="h-5 px-2 text-[10px]">{run.agentName}</Badge>
        <Badge variant="outline" class="h-5 px-2 text-[10px]">{run.provider}</Badge>
        <span class="text-muted-foreground">Started {formatRelativeTime(run.createdAt)}</span>
        {#if run.completedAt}
          <span class="text-muted-foreground">Completed {formatRelativeTime(run.completedAt)}</span>
        {:else if run.lastHeartbeatAt}
          <span class="text-muted-foreground"
            >Updated {formatRelativeTime(run.lastHeartbeatAt)}</span
          >
        {/if}
      </div>
    {/if}

    {#if canResumeRetry && onResumeRetry}
      <div class="mt-3">
        <Button
          size="sm"
          variant="outline"
          class="h-8 text-xs"
          disabled={resumingRetry}
          onclick={() => void onResumeRetry()}
        >
          {resumingRetry ? 'Continuing...' : 'Continue Retry'}
        </Button>
      </div>
    {/if}
  </div>

  <div
    bind:this={scrollViewport}
    class="min-h-0 flex-1 overflow-y-auto px-5 py-4"
    onscroll={handleScroll}
  >
    {#if loading}
      <div class="text-muted-foreground mb-3 text-xs">Loading transcript…</div>
    {/if}

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

            {#if block.kind === 'assistant_message'}
              <div class="prose prose-sm prose-neutral max-w-none break-words">
                <ChatMarkdownContent source={block.text} />
              </div>
            {:else if block.kind === 'terminal_output'}
              <div class="space-y-3">
                <div
                  class={`overflow-x-auto rounded-xl border border-white/10 bg-black/20 px-3 py-3 font-mono text-xs leading-5 whitespace-pre-wrap ${
                    isOutputExpandable(block) && !isExpanded(block.id)
                      ? 'max-h-72 overflow-y-hidden'
                      : ''
                  }`}
                >
                  {block.text}
                </div>
                {#if isOutputExpandable(block)}
                  <Button
                    size="sm"
                    variant="outline"
                    class="h-8 border-white/15 bg-white/5 text-xs text-slate-100 hover:bg-white/10"
                    onclick={() => void toggleOutputExpansion(block.id)}
                  >
                    {isExpanded(block.id) ? 'Collapse output' : 'Expand output'}
                  </Button>
                {/if}
              </div>
            {:else if block.kind === 'tool_call'}
              <div class="space-y-2">
                <p class="text-foreground text-sm font-medium">{block.toolName}</p>
                {#if block.summary}
                  <p class="text-muted-foreground text-sm">{block.summary}</p>
                {/if}
              </div>
            {:else if block.kind === 'interrupt'}
              <div class="space-y-3 text-sm">
                <p class="font-medium text-amber-950">{block.title}</p>
                <p class="text-amber-900">{block.summary}</p>

                {#if interruptString(block.payload, 'command')}
                  <div class="rounded-xl border border-amber-300/70 bg-white/80 px-3 py-2">
                    <div
                      class="mb-1 text-[10px] font-semibold tracking-[0.14em] text-amber-700 uppercase"
                    >
                      command
                    </div>
                    <pre
                      class="font-mono text-xs leading-5 whitespace-pre-wrap text-amber-950">{interruptString(
                        block.payload,
                        'command',
                      )}</pre>
                  </div>
                {/if}

                {#if interruptString(block.payload, 'file', 'path', 'target')}
                  <div class="rounded-xl border border-amber-300/70 bg-white/80 px-3 py-2">
                    <div
                      class="mb-1 text-[10px] font-semibold tracking-[0.14em] text-amber-700 uppercase"
                    >
                      target
                    </div>
                    <div class="font-mono text-xs leading-5 text-amber-950">
                      {interruptString(block.payload, 'file', 'path', 'target')}
                    </div>
                  </div>
                {/if}

                {#if interruptQuestion(block)}
                  <div
                    class="rounded-xl border border-amber-300/70 bg-white/80 px-3 py-2 text-amber-950"
                  >
                    {interruptQuestion(block)}
                  </div>
                {/if}

                {#if block.options.length > 0}
                  <div class="flex flex-wrap gap-2">
                    {#each block.options as option}
                      <Badge variant="outline" class="border-amber-300 bg-white text-amber-900">
                        {option.label}
                      </Badge>
                    {/each}
                  </div>
                {/if}
              </div>
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
