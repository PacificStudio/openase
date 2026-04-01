<script lang="ts">
  import { Badge } from '$ui/badge'
  import { formatRelativeTime } from '$lib/utils'
  import type { TicketRun, TicketRunTranscriptBlock } from '../types'

  let {
    run = null,
    blocks = [],
  }: {
    run?: TicketRun | null
    blocks?: TicketRunTranscriptBlock[]
  } = $props()

  function statusTone(status: TicketRun['status']) {
    switch (status) {
      case 'completed':
        return 'bg-emerald-500/10 text-emerald-300 border-emerald-500/20'
      case 'failed':
      case 'stalled':
        return 'bg-red-500/10 text-red-300 border-red-500/20'
      case 'executing':
        return 'bg-blue-500/10 text-blue-300 border-blue-500/20'
      default:
        return 'bg-amber-500/10 text-amber-300 border-amber-500/20'
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
</script>

<section class="shrink-0 border-b px-6 py-5">
  <div class="flex items-start justify-between gap-4">
    <div class="space-y-1">
      <div class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">Runs</div>
      <h3 class="text-sm font-semibold">
        {#if run}
          Attempt {run.attemptNumber} · {run.agentName}
        {:else}
          Latest Run
        {/if}
      </h3>
      <p class="text-muted-foreground text-xs">
        {#if run}
          {run.currentStepSummary ||
            run.currentStepStatus ||
            `Started ${formatRelativeTime(run.createdAt)}`}
        {:else}
          No execution transcript yet.
        {/if}
      </p>
    </div>

    {#if run}
      <Badge variant="outline" class={`h-6 px-2.5 text-[11px] ${statusTone(run.status)}`}>
        {run.status}
      </Badge>
    {/if}
  </div>

  {#if run}
    <div class="mt-4 max-h-72 space-y-3 overflow-y-auto pr-1">
      {#each blocks as block (block.id)}
        <article class="bg-muted/25 border-border rounded-xl border px-3 py-3">
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
          {:else if block.kind === 'result'}
            <p class="text-foreground text-sm">{block.summary}</p>
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
  {/if}
</section>
