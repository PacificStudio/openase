<script lang="ts">
  import { formatCount } from '$lib/utils'
  import { Button } from '$ui/button'
  import type { TicketDetail, TicketRun } from '../types'

  type LoadRunsAction = () => Promise<void> | void

  let {
    ticket,
    runs = [],
    runsLoaded = false,
    loadingRuns = false,
    runsError = '',
    onLoadRuns,
  }: {
    ticket: Pick<TicketDetail, 'id' | 'costTokensTotal'>
    runs?: TicketRun[]
    runsLoaded?: boolean
    loadingRuns?: boolean
    runsError?: string
    onLoadRuns?: LoadRunsAction
  } = $props()

  const usageMetrics = [
    { key: 'input', label: 'Input' },
    { key: 'output', label: 'Output' },
    { key: 'cachedInput', label: 'Cached Input' },
    { key: 'cacheCreation', label: 'Cache Creation' },
    { key: 'reasoning', label: 'Reasoning' },
    { key: 'prompt', label: 'Prompt' },
    { key: 'candidate', label: 'Candidate' },
    { key: 'tool', label: 'Tool' },
  ] as const

  let expanded = $state(false)
  let previousTicketId = ''

  $effect(() => {
    if (ticket.id === previousTicketId) {
      return
    }

    previousTicketId = ticket.id
    expanded = false
  })

  async function toggleBreakdown() {
    expanded = !expanded
    if (expanded && !runsLoaded && !loadingRuns) {
      await onLoadRuns?.()
    }
  }

  async function retryLoad() {
    await onLoadRuns?.()
  }
</script>

<div class="text-muted-foreground">Total Tokens</div>
<div class="flex items-center gap-2">
  <span class="text-foreground">{formatCount(ticket.costTokensTotal)}</span>
  <Button
    type="button"
    variant="outline"
    size="sm"
    class="h-6 px-2 text-[11px]"
    aria-expanded={expanded}
    onclick={() => void toggleBreakdown()}
  >
    Breakdown
  </Button>
</div>

{#if expanded}
  <div class="col-span-2 mt-1">
    <div class="border-border/60 bg-muted/10 rounded-md border p-3">
      {#if loadingRuns}
        <p class="text-muted-foreground text-xs">Loading run usage breakdown…</p>
      {:else if runsError}
        <div class="space-y-2">
          <p class="text-destructive text-xs">{runsError}</p>
          <Button
            type="button"
            variant="outline"
            size="sm"
            class="h-6 px-2 text-[11px]"
            onclick={() => void retryLoad()}
          >
            Retry
          </Button>
        </div>
      {:else if runsLoaded && runs.length === 0}
        <p class="text-muted-foreground text-xs">No run usage yet.</p>
      {:else if runsLoaded}
        <div class="space-y-3">
          {#each runs as run (run.id)}
            <article class="border-border/50 bg-background/80 rounded-md border p-2.5">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="text-foreground truncate text-xs font-medium">
                    {run.adapterType || 'unknown-adapter'}
                  </div>
                  <div class="text-foreground/80 truncate text-[11px]">
                    {run.modelName || 'unknown-model'}
                  </div>
                  <div class="text-muted-foreground truncate text-[10px]">
                    Attempt #{run.attemptNumber} · {run.provider}
                  </div>
                </div>
                <div class="text-right">
                  <div class="text-foreground text-xs font-medium">
                    {formatCount(run.usage.total)}
                  </div>
                  <div class="text-muted-foreground text-[10px]">total</div>
                </div>
              </div>

              <dl class="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 text-[11px] sm:grid-cols-4">
                {#each usageMetrics as metric (metric.key)}
                  <div>
                    <dt class="text-muted-foreground">{metric.label}</dt>
                    <dd class="text-foreground">{formatCount(run.usage[metric.key])}</dd>
                  </div>
                {/each}
              </dl>
            </article>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{/if}
