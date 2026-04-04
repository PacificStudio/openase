<script lang="ts">
  import type { StreamConnectionState } from '$lib/api/sse'
  import { Button } from '$ui/button'
  import { Badge } from '$ui/badge'
  import { formatRelativeTime } from '$lib/utils'
  import {
    connectionLabel,
    connectionTone,
    statusLabel,
    statusTone,
  } from './ticket-run-transcript-view'
  import type { TicketRun } from '../types'

  let {
    run = null,
    latestRunId = null,
    liveSelected = false,
    showJumpToLive = false,
    loading = false,
    streamState = 'idle',
    recovering = false,
    canResumeRetry = false,
    resumingRetry = false,
    onJumpToLive,
    onResumeRetry,
  }: {
    run?: TicketRun | null
    latestRunId?: string | null
    liveSelected?: boolean
    showJumpToLive?: boolean
    loading?: boolean
    streamState?: StreamConnectionState
    recovering?: boolean
    canResumeRetry?: boolean
    resumingRetry?: boolean
    onJumpToLive?: () => Promise<void> | void
    onResumeRetry?: () => Promise<void> | void
  } = $props()
</script>

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
      <p
        class="text-muted-foreground group/summary max-w-full truncate text-xs transition-all hover:break-words hover:whitespace-normal"
        title={run?.currentStepSummary || ''}
      >
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
      {#if connectionLabel(streamState, recovering, liveSelected)}
        <Badge
          variant="outline"
          class={`h-6 px-2.5 text-[11px] ${connectionTone(streamState, recovering)}`}
        >
          {connectionLabel(streamState, recovering, liveSelected)}
        </Badge>
      {/if}
      {#if liveSelected && showJumpToLive}
        <Button
          size="sm"
          variant="outline"
          class="h-8 text-xs"
          onclick={() => void onJumpToLive?.()}
        >
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
      {#if run.status === 'completed' && run.completedAt}
        <span class="text-muted-foreground">Completed {formatRelativeTime(run.completedAt)}</span>
      {:else if run.status === 'ended' && run.terminalAt}
        <span class="text-muted-foreground">Ended {formatRelativeTime(run.terminalAt)}</span>
      {:else if run.lastHeartbeatAt}
        <span class="text-muted-foreground">Updated {formatRelativeTime(run.lastHeartbeatAt)}</span>
      {/if}
    </div>
  {/if}

  {#if loading}
    <div class="text-muted-foreground mt-3 text-xs">Loading transcript…</div>
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
