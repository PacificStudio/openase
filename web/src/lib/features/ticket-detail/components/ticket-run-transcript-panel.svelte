<script lang="ts">
  import { Button } from '$ui/button'
  import { Badge } from '$ui/badge'
  import { formatRelativeTime } from '$lib/utils'
  import type { TicketRun, TicketRunTranscriptBlock } from '../types'

  let {
    run = null,
    blocks = [],
    latestRunId = null,
    loading = false,
    canResumeRetry = false,
    resumingRetry = false,
    onResumeRetry,
  }: {
    run?: TicketRun | null
    blocks?: TicketRunTranscriptBlock[]
    latestRunId?: string | null
    loading?: boolean
    canResumeRetry?: boolean
    resumingRetry?: boolean
    onResumeRetry?: () => Promise<void> | void
  } = $props()

  function statusLabel(run: TicketRun) {
    if (run.status === 'completed') return 'Completed'
    if (run.status === 'failed') return 'Failed'
    if (run.status === 'stalled') return 'Stalled'
    if ((run.currentStepStatus ?? '').toLowerCase().includes('input')) return 'Waiting Input'
    if (run.status === 'launching') return 'Launching'
    return 'Running'
  }

  function statusTone(run: TicketRun) {
    switch (run.status) {
      case 'completed':
        return 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600'
      case 'failed':
      case 'stalled':
        return 'border-red-500/20 bg-red-500/10 text-red-600'
      case 'ready':
      case 'executing':
        return 'border-sky-500/20 bg-sky-500/10 text-sky-600'
      default:
        return 'border-amber-500/20 bg-amber-500/10 text-amber-600'
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
      case 'result':
        return 'Result'
    }
  }

  function isStreamingBlock(
    block: TicketRunTranscriptBlock,
  ): block is Extract<TicketRunTranscriptBlock, { kind: 'assistant_message' | 'terminal_output' }> {
    return block.kind === 'assistant_message' || block.kind === 'terminal_output'
  }

  function blockCardClass(block: TicketRunTranscriptBlock) {
    if (block.kind !== 'result') {
      return 'bg-muted/25 border-border'
    }

    switch (block.outcome) {
      case 'completed':
        return 'border-emerald-500/20 bg-emerald-500/5'
      case 'failed':
        return 'border-red-500/20 bg-red-500/5'
      default:
        return 'border-amber-500/20 bg-amber-500/5'
    }
  }
</script>

<section class="flex min-h-0 flex-1 flex-col overflow-hidden">
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

      {#if run}
        <div class="flex flex-wrap items-center justify-end gap-2">
          {#if latestRunId === run.id}
            <Badge variant="outline" class="h-6 px-2.5 text-[11px]">Live</Badge>
          {/if}
          <Badge variant="outline" class={`h-6 px-2.5 text-[11px] ${statusTone(run)}`}>
            {statusLabel(run)}
          </Badge>
        </div>
      {/if}
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

  <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
    {#if loading}
      <div class="text-muted-foreground mb-3 text-xs">Loading transcript…</div>
    {/if}

    {#if run}
      <div class="space-y-3">
        {#each blocks as block (block.id)}
          <article class={`rounded-xl border px-3 py-3 ${blockCardClass(block)}`}>
            <div class="mb-2 flex items-center justify-between gap-2 text-[11px]">
              <span class="text-muted-foreground font-medium uppercase">{blockLabel(block)}</span>
              {#if block.kind === 'phase'}
                <span class="text-muted-foreground">{block.phase}</span>
              {:else if block.kind === 'step'}
                <span class="text-muted-foreground">{block.stepStatus}</span>
              {:else if block.kind === 'tool_call'}
                <span class="text-muted-foreground">{block.toolName}</span>
              {:else if block.kind === 'result'}
                <span class="text-muted-foreground">{block.outcome}</span>
              {:else if isStreamingBlock(block) && block.streaming}
                <span class="text-muted-foreground">Streaming</span>
              {/if}
            </div>

            {#if block.kind === 'assistant_message' || block.kind === 'terminal_output'}
              <pre
                class="text-foreground overflow-x-auto font-mono text-xs leading-5 break-words whitespace-pre-wrap">{block.text}</pre>
            {:else if block.kind === 'tool_call'}
              <p class="text-foreground text-sm">{block.summary || block.toolName}</p>
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
