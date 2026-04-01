<script lang="ts">
  import { Badge } from '$ui/badge'
  import { formatRelativeTime } from '$lib/utils'
  import TicketRunTranscriptPanel from './ticket-run-transcript-panel.svelte'
  import type { TicketDetail, TicketRun, TicketRunTranscriptBlock } from '../types'

  let {
    ticket,
    runs = [],
    currentRun = null,
    blocks = [],
    loadingRunId = null,
    resumingRetry = false,
    onSelectRun,
    onResumeRetry,
  }: {
    ticket: TicketDetail
    runs?: TicketRun[]
    currentRun?: TicketRun | null
    blocks?: TicketRunTranscriptBlock[]
    loadingRunId?: string | null
    resumingRetry?: boolean
    onSelectRun?: (runId: string) => Promise<void> | void
    onResumeRetry?: () => Promise<void> | void
  } = $props()

  const latestRun = $derived(runs[0] ?? null)

  function statusLabel(run: TicketRun) {
    if (run.status === 'completed') return 'Completed'
    if (run.status === 'failed') return 'Failed'
    if (run.status === 'stalled') return 'Stalled'
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
</script>

{#if runs.length === 0}
  <section class="flex min-h-0 flex-1 items-center justify-center px-6 py-10">
    <div class="text-muted-foreground max-w-sm rounded-2xl border border-dashed px-5 py-6 text-sm">
      No ticket runs yet. When execution starts, each attempt will appear here as its own
      transcript.
    </div>
  </section>
{:else}
  <section class="flex min-h-0 flex-1 flex-col overflow-hidden md:flex-row">
    <aside
      class="bg-muted/15 border-border flex w-full shrink-0 flex-col border-b md:w-80 md:border-r md:border-b-0"
    >
      <div class="border-border border-b px-5 py-4">
        <div class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
          Run Rail
        </div>
        <h3 class="mt-1 text-sm font-semibold">Attempts</h3>
        <p class="text-muted-foreground mt-1 text-xs">
          Latest run stays marked as live while older attempts remain directly browsable.
        </p>
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto px-3 py-3">
        <div class="space-y-2">
          {#each runs as run (run.id)}
            {@const selected = currentRun?.id === run.id}
            {@const live = latestRun?.id === run.id}
            <button
              type="button"
              class={`w-full rounded-2xl border px-4 py-3 text-left transition ${
                selected
                  ? 'border-foreground/15 bg-background shadow-sm'
                  : 'border-border bg-background/50 hover:bg-background'
              }`}
              aria-label={`View Attempt ${run.attemptNumber}`}
              aria-pressed={selected}
              onclick={() => void onSelectRun?.(run.id)}
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="text-sm font-medium">Attempt {run.attemptNumber}</span>
                    {#if live}
                      <Badge variant="outline" class="h-5 px-2 text-[10px]">Live</Badge>
                    {/if}
                    {#if selected}
                      <Badge variant="outline" class="h-5 px-2 text-[10px]">Selected</Badge>
                    {/if}
                  </div>
                  <p class="text-muted-foreground mt-1 truncate text-xs">{run.agentName}</p>
                </div>
                <Badge
                  variant="outline"
                  class={`h-5 shrink-0 px-2 text-[10px] ${statusClass(run)}`}
                >
                  {statusLabel(run)}
                </Badge>
              </div>

              <div class="mt-3 flex flex-wrap items-center gap-2 text-[11px]">
                <Badge variant="outline" class="h-5 px-2 text-[10px]">{run.provider}</Badge>
                <span class="text-muted-foreground">{summaryLine(run)}</span>
              </div>

              {#if loadingRunId === run.id}
                <div class="text-muted-foreground mt-2 text-[11px]">Loading transcript…</div>
              {/if}
            </button>
          {/each}
        </div>
      </div>
    </aside>

    <TicketRunTranscriptPanel
      run={currentRun}
      {blocks}
      latestRunId={latestRun?.id ?? null}
      loading={loadingRunId === currentRun?.id}
      canResumeRetry={ticket.retryPaused &&
        ticket.pauseReason === 'repeated_stalls' &&
        latestRun?.id === currentRun?.id}
      {resumingRetry}
      {onResumeRetry}
    />
  </section>
{/if}
