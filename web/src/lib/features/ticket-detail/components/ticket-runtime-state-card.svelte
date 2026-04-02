<script lang="ts">
  import { Button } from '$ui/button'
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { TicketDetail, TicketPickupDiagnosis } from '../types'
  import { LoaderCircle } from '@lucide/svelte'

  let {
    ticket,
    resumingRetry = false,
    onResumeRetry,
  }: {
    ticket: TicketDetail
    resumingRetry?: boolean
    onResumeRetry?: () => Promise<void> | void
  } = $props()

  type RuntimeTone = 'neutral' | 'info' | 'success' | 'warning' | 'danger'
  type RuntimeSummary = {
    label: string
    tone: RuntimeTone
    message: string
    timestamp?: string
    nextActionHint?: string
    countdownLine?: string
    detailItems: Array<{ label: string; value: string }>
  }

  let nowMs = $state(Date.now())

  const retryTarget = $derived(ticket.pickupDiagnosis?.retry.nextRetryAt ?? ticket.nextRetryAt)
  const runtimeSummary = $derived.by(() => summarizeRuntime(ticket, nowMs))
  const isAnimating = $derived(runtimeSummary.tone === 'info' || runtimeSummary.tone === 'success')
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

  function summarizeRuntime(ticket: TicketDetail, now: number): RuntimeSummary {
    if (ticket.pickupDiagnosis) {
      return summarizeDiagnosis(ticket.pickupDiagnosis, ticket, now)
    }

    return summarizeLegacy(ticket)
  }

  function summarizeDiagnosis(
    diagnosis: TicketPickupDiagnosis,
    ticket: TicketDetail,
    now: number,
  ): RuntimeSummary {
    const detailItems: RuntimeSummary['detailItems'] = []

    if (diagnosis.workflow) {
      detailItems.push({
        label: 'Workflow',
        value: `${diagnosis.workflow.name} · ${diagnosis.workflow.isActive ? 'Active' : 'Inactive'}`,
      })
    }

    if (diagnosis.agent) {
      detailItems.push({
        label: 'Agent',
        value: `${diagnosis.agent.name} · ${humanizeControlState(diagnosis.agent.runtimeControlState)}`,
      })
    } else if (diagnosis.primaryReasonCode === 'workflow_missing_agent') {
      detailItems.push({ label: 'Agent', value: 'No agent is bound to the workflow.' })
    }

    if (diagnosis.provider) {
      detailItems.push({
        label: 'Provider',
        value: `${diagnosis.provider.name} · ${humanizeProviderState(diagnosis.provider)}`,
      })
    }

    if (diagnosis.blockedBy.length > 0) {
      detailItems.push({
        label: 'Dependencies',
        value: diagnosis.blockedBy.map((item) => `${item.identifier} ${item.title}`).join(', '),
      })
    }

    const capacityLine = buildCapacityLine(diagnosis)
    if (capacityLine) {
      detailItems.push({ label: 'Capacity', value: capacityLine })
    }

    const retryTimestamp = diagnosis.retry.nextRetryAt
    const countdownLine = retryTimestamp ? formatRetryCountdown(retryTimestamp, now) : undefined
    if (diagnosis.retry.retryPaused && diagnosis.retry.pauseReason) {
      detailItems.push({
        label: 'Retry',
        value: humanizePauseReason(diagnosis.retry.pauseReason),
      })
    }

    return {
      label: diagnosisLabel(diagnosis.state),
      tone: diagnosisTone(diagnosis.state, diagnosis.primaryReasonCode),
      message: diagnosis.primaryReasonMessage,
      timestamp:
        diagnosis.state === 'running'
          ? ticket.startedAt
          : diagnosis.state === 'completed'
            ? ticket.completedAt
            : undefined,
      nextActionHint: diagnosis.nextActionHint,
      countdownLine,
      detailItems,
    }
  }

  function summarizeLegacy(ticket: TicketDetail): RuntimeSummary {
    if (ticket.completedAt) {
      return {
        label: 'Completed',
        tone: 'success',
        message: 'Execution finished successfully.',
        timestamp: ticket.completedAt,
        detailItems: [],
      }
    }

    if (ticket.retryPaused && ticket.pauseReason === 'repeated_stalls') {
      return {
        label: 'Blocked',
        tone: 'warning',
        message: 'Paused after repeated stalls — manual retry required.',
        detailItems: [{ label: 'Retry', value: 'Manual retry required after repeated stalls.' }],
      }
    }

    if (ticket.retryPaused) {
      return {
        label: 'Waiting',
        tone: 'warning',
        message: 'Waiting for retry conditions to change.',
        detailItems: [],
      }
    }

    if (ticket.assignedAgent?.runtimeControlState === 'paused') {
      return {
        label: 'Unavailable',
        tone: 'warning',
        message: 'Assigned agent is paused.',
        detailItems: [{ label: 'Agent', value: `${ticket.assignedAgent.name} · Paused` }],
      }
    }

    switch (ticket.assignedAgent?.runtimePhase) {
      case 'failed':
        return {
          label: 'Failed',
          tone: 'danger',
          message: 'Latest attempt failed — needs attention.',
          detailItems: [],
        }
      case 'launching':
        return {
          label: 'Launching',
          tone: 'info',
          message: 'Agent is spinning up the runtime.',
          detailItems: [],
        }
      case 'ready':
      case 'executing':
        return {
          label: 'Running',
          tone: 'success',
          message: 'Agent runtime is live.',
          timestamp: ticket.startedAt,
          detailItems: [],
        }
      default:
        return {
          label: ticket.assignedAgent ? 'Assigned' : 'Waiting',
          tone: 'neutral',
          message: ticket.assignedAgent
            ? 'Agent bound, no active runtime.'
            : 'No agent runtime attached yet.',
          detailItems: [],
        }
    }
  }

  function diagnosisLabel(state: TicketPickupDiagnosis['state']) {
    switch (state) {
      case 'runnable':
        return 'Runnable'
      case 'waiting':
        return 'Waiting'
      case 'blocked':
        return 'Blocked'
      case 'running':
        return 'Running'
      case 'completed':
        return 'Completed'
      default:
        return 'Unavailable'
    }
  }

  function diagnosisTone(
    state: TicketPickupDiagnosis['state'],
    reasonCode: TicketPickupDiagnosis['primaryReasonCode'],
  ): RuntimeTone {
    if (state === 'running' || state === 'completed' || state === 'runnable') {
      return state === 'running' ? 'success' : state === 'completed' ? 'success' : 'info'
    }
    if (state === 'waiting') return 'info'
    if (
      reasonCode === 'retry_paused_repeated_stalls' ||
      reasonCode === 'retry_paused_budget' ||
      reasonCode === 'retry_paused_user' ||
      reasonCode === 'blocked_dependency'
    ) {
      return 'warning'
    }
    return 'danger'
  }

  function dotClass(tone: RuntimeTone) {
    switch (tone) {
      case 'info':
        return 'bg-sky-400'
      case 'success':
        return 'bg-emerald-400'
      case 'warning':
        return 'bg-amber-400'
      case 'danger':
        return 'bg-red-400'
      default:
        return 'bg-muted-foreground/50'
    }
  }

  function labelClass(tone: RuntimeTone) {
    switch (tone) {
      case 'info':
        return 'text-sky-400'
      case 'success':
        return 'text-emerald-400'
      case 'warning':
        return 'text-amber-400'
      case 'danger':
        return 'text-red-400'
      default:
        return 'text-muted-foreground'
    }
  }

  function humanizeControlState(value: string) {
    switch (value) {
      case 'pause_requested':
        return 'Pause requested'
      case 'paused':
        return 'Paused'
      default:
        return 'Active'
    }
  }

  function humanizeProviderState(provider: NonNullable<TicketPickupDiagnosis['provider']>) {
    const state =
      provider.availabilityState === 'available'
        ? 'Available'
        : provider.availabilityState === 'stale'
          ? 'Stale health'
          : provider.availabilityState === 'unknown'
            ? 'Unknown health'
            : 'Unavailable'
    if (!provider.availabilityReason) return state
    return `${state} (${humanizeAvailabilityReason(provider.availabilityReason)})`
  }

  function humanizeAvailabilityReason(reason: string) {
    switch (reason) {
      case 'machine_offline':
        return 'machine offline'
      case 'machine_degraded':
        return 'machine degraded'
      case 'machine_maintenance':
        return 'machine maintenance'
      case 'l4_snapshot_missing':
        return 'health not probed'
      case 'stale_l4_snapshot':
        return 'stale health snapshot'
      case 'cli_missing':
        return 'CLI missing'
      case 'not_logged_in':
        return 'not logged in'
      case 'not_ready':
        return 'CLI not ready'
      case 'config_incomplete':
        return 'config incomplete'
      default:
        return reason.replaceAll('_', ' ')
    }
  }

  function humanizePauseReason(reason: string) {
    switch (reason) {
      case 'repeated_stalls':
        return 'Manual retry required after repeated stalls.'
      case 'budget_exhausted':
        return 'Retries are paused because the budget is exhausted.'
      case 'user_paused':
        return 'Retries are paused manually.'
      default:
        return reason.replaceAll('_', ' ')
    }
  }

  function buildCapacityLine(diagnosis: TicketPickupDiagnosis) {
    const entries: string[] = []
    if (
      diagnosis.capacity.workflow.limited &&
      diagnosis.capacity.workflow.activeRuns >= diagnosis.capacity.workflow.capacity
    ) {
      entries.push(
        `Workflow ${diagnosis.capacity.workflow.activeRuns}/${diagnosis.capacity.workflow.capacity}`,
      )
    }
    if (
      diagnosis.capacity.project.limited &&
      diagnosis.capacity.project.activeRuns >= diagnosis.capacity.project.capacity
    ) {
      entries.push(
        `Project ${diagnosis.capacity.project.activeRuns}/${diagnosis.capacity.project.capacity}`,
      )
    }
    if (
      diagnosis.capacity.provider.limited &&
      diagnosis.capacity.provider.activeRuns >= diagnosis.capacity.provider.capacity
    ) {
      entries.push(
        `Provider ${diagnosis.capacity.provider.activeRuns}/${diagnosis.capacity.provider.capacity}`,
      )
    }
    if (
      diagnosis.capacity.status.limited &&
      diagnosis.capacity.status.capacity !== undefined &&
      diagnosis.capacity.status.activeRuns >= diagnosis.capacity.status.capacity
    ) {
      entries.push(
        `Status ${diagnosis.capacity.status.activeRuns}/${diagnosis.capacity.status.capacity}`,
      )
    }
    return entries.join(' · ')
  }

  function formatRetryCountdown(value: string, now: number) {
    const targetMs = new Date(value).getTime()
    if (!Number.isFinite(targetMs)) return undefined

    const absolute = formatUTC(value)
    const remainingMs = Math.max(0, targetMs - now)
    if (remainingMs === 0) {
      return `Retry window elapsed (at ${absolute})`
    }

    return `Retrying in ${formatRemaining(remainingMs)} (at ${absolute})`
  }

  function formatRemaining(value: number) {
    const totalSeconds = Math.max(0, Math.floor(value / 1000))
    const hours = Math.floor(totalSeconds / 3600)
    const minutes = Math.floor((totalSeconds % 3600) / 60)
    const seconds = totalSeconds % 60

    if (hours > 0) {
      return `${hours}h ${String(minutes).padStart(2, '0')}m`
    }
    if (minutes > 0) {
      return `${minutes}m ${String(seconds).padStart(2, '0')}s`
    }
    return `${seconds}s`
  }

  function formatUTC(value: string) {
    const date = new Date(value)
    const year = date.getUTCFullYear()
    const month = String(date.getUTCMonth() + 1).padStart(2, '0')
    const day = String(date.getUTCDate()).padStart(2, '0')
    const hours = String(date.getUTCHours()).padStart(2, '0')
    const minutes = String(date.getUTCMinutes()).padStart(2, '0')
    const seconds = String(date.getUTCSeconds()).padStart(2, '0')
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds} UTC`
  }
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
            dotClass(runtimeSummary.tone),
          )}
        ></span>
      {/if}
      {#if runtimeSummary.tone === 'info'}
        <LoaderCircle class="size-4 animate-spin text-sky-400" />
      {:else}
        <span class={cn('relative size-2 rounded-full', dotClass(runtimeSummary.tone))}></span>
      {/if}
    </div>

    <div class="min-w-0 flex-1 space-y-1">
      <div class="flex items-baseline gap-2">
        <span class={cn('text-xs font-semibold', labelClass(runtimeSummary.tone))}>
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
</section>
