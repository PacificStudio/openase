<script lang="ts">
  import { Button } from '$ui/button'
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { TicketDetail } from '../types'
  import { LoaderCircle } from '@lucide/svelte'
  import {
    runtimeDotClass,
    runtimeLabelClass,
    summarizeRuntime,
  } from './ticket-runtime-state-card-view'

  let {
    ticket,
    resumingRetry = false,
    resettingWorkspace = false,
    onResumeRetry,
    onResetWorkspace,
  }: {
    ticket: TicketDetail
    resumingRetry?: boolean
    resettingWorkspace?: boolean
    onResumeRetry?: () => Promise<void> | void
    onResetWorkspace?: () => Promise<void> | void
  } = $props()

  let nowMs = $state(Date.now())

  const retryTarget = $derived(ticket.pickupDiagnosis?.retry.nextRetryAt ?? ticket.nextRetryAt)
  const runtimeSummary = $derived.by(() => summarizeRuntime(ticket, nowMs))
  const isAnimating = $derived(
    runtimeSummary.label === 'Running' || runtimeSummary.label === 'Launching',
  )
  const shouldShowResumeRetry = $derived.by(() => {
    if (!onResumeRetry) return false
    if (ticket.pickupDiagnosis) {
      return (
        ticket.pickupDiagnosis.primaryReasonCode === 'retry_paused_repeated_stalls' ||
        ticket.pickupDiagnosis.reasons.some(
          (reason) => reason.code === 'retry_paused_repeated_stalls',
        )
      )
    }
    return ticket.retryPaused && ticket.pauseReason === 'repeated_stalls'
  })
  const shouldShowResetWorkspace = $derived(!ticket.currentRunId && !!onResetWorkspace)

  $effect(() => {
    const target = retryTarget
    if (typeof window === 'undefined' || !target) {
      return
    }

    const deadline = new Date(target).getTime()
    if (!Number.isFinite(deadline) || deadline <= Date.now()) {
      nowMs = Date.now()
      return
    }

    nowMs = Date.now()
    const interval = window.setInterval(() => {
      nowMs = Date.now()
    }, 1000)

    return () => window.clearInterval(interval)
  })
</script>

<section class="space-y-3">
  <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
    Runtime
  </span>

  <div class="flex items-start gap-3">
    <div class="relative mt-0.5 flex size-5 shrink-0 items-center justify-center">
      {#if isAnimating}
        <span
          class={cn(
            'absolute inset-0 animate-ping rounded-full opacity-30',
            runtimeDotClass(runtimeSummary.tone),
          )}
        ></span>
      {/if}
      {#if runtimeSummary.tone === 'info'}
        <LoaderCircle class="size-4 animate-spin text-sky-400" />
      {:else}
        <span class={cn('relative size-2 rounded-full', runtimeDotClass(runtimeSummary.tone))}
        ></span>
      {/if}
    </div>

    <div class="min-w-0 flex-1 space-y-1">
      <div class="flex items-baseline gap-2">
        <span class={cn('text-xs font-semibold', runtimeLabelClass(runtimeSummary.tone))}>
          {runtimeSummary.label}
        </span>
        {#if runtimeSummary.timestamp}
          <span class="text-muted-foreground text-[10px]">
            {formatRelativeTime(runtimeSummary.timestamp)}
          </span>
        {/if}
      </div>

      <p class="text-foreground text-[11px] leading-normal">{runtimeSummary.message}</p>

      {#if runtimeSummary.countdownLine}
        <p class="text-muted-foreground text-[11px] leading-normal">
          {runtimeSummary.countdownLine}
        </p>
      {/if}

      {#if runtimeSummary.nextActionHint}
        <p class="text-muted-foreground text-[11px] leading-normal">
          {runtimeSummary.nextActionHint}
        </p>
      {/if}
    </div>
  </div>

  {#if runtimeSummary.detailItems.length > 0}
    <div class="border-border/60 space-y-2 rounded-md border p-2.5">
      {#each runtimeSummary.detailItems as item}
        <div class="grid grid-cols-[70px_1fr] gap-2 text-[11px]">
          <span class="text-muted-foreground">{item.label}</span>
          <span class="text-foreground break-words">{item.value}</span>
        </div>
      {/each}
    </div>
  {/if}

  {#if shouldShowResumeRetry}
    <Button
      size="sm"
      variant="outline"
      class="h-7 w-full text-[11px]"
      disabled={resumingRetry}
      onclick={() => void onResumeRetry?.()}
    >
      {resumingRetry ? 'Continuing...' : 'Continue Retry'}
    </Button>
  {/if}

  {#if shouldShowResetWorkspace}
    <Button
      size="sm"
      variant="outline"
      class="h-7 w-full text-[11px]"
      disabled={resettingWorkspace}
      onclick={() => void onResetWorkspace?.()}
    >
      {resettingWorkspace ? 'Resetting...' : 'Reset Workspace'}
    </Button>
  {/if}
</section>
