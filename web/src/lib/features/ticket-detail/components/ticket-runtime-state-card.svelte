<script lang="ts">
  import { Button } from '$ui/button'
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { TicketDetail } from '../types'
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

  const runtimeSummary = $derived.by(() => summarizeRuntime(ticket))

  function summarizeRuntime(ticket: TicketDetail) {
    if (ticket.completedAt) {
      return {
        label: 'Completed',
        tone: 'success' as RuntimeTone,
        description: 'Execution finished successfully.',
      }
    }

    if (ticket.retryPaused && ticket.pauseReason === 'repeated_stalls') {
      return {
        label: 'Stalled',
        tone: 'warning' as RuntimeTone,
        description: 'Paused after repeated stalls — manual retry required.',
      }
    }

    if (ticket.retryPaused) {
      return {
        label: 'Paused',
        tone: 'warning' as RuntimeTone,
        description: 'Waiting for retry conditions to change.',
      }
    }

    if (ticket.assignedAgent?.runtimeControlState === 'paused') {
      return {
        label: 'Paused',
        tone: 'warning' as RuntimeTone,
        description: 'Waiting to be resumed.',
      }
    }

    switch (ticket.assignedAgent?.runtimePhase) {
      case 'failed':
        return {
          label: 'Failed',
          tone: 'danger' as RuntimeTone,
          description: 'Latest attempt failed — needs attention.',
        }
      case 'launching':
        return {
          label: 'Launching',
          tone: 'info' as RuntimeTone,
          description: 'Agent is spinning up the runtime.',
        }
      case 'ready':
      case 'executing':
        return {
          label: 'Running',
          tone: 'success' as RuntimeTone,
          description: 'Agent runtime is live.',
        }
      default:
        if (ticket.assignedAgent) {
          return {
            label: 'Assigned',
            tone: 'neutral' as RuntimeTone,
            description: 'Agent bound, no active runtime.',
          }
        }
        return {
          label: 'Waiting',
          tone: 'neutral' as RuntimeTone,
          description: 'No agent runtime attached yet.',
        }
    }
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

  const isAnimating = $derived(
    runtimeSummary.tone === 'info' || runtimeSummary.tone === 'success',
  )
</script>

<section class="space-y-2.5">
  <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
    Runtime
  </span>

  <div class="flex items-start gap-3">
    <div class="relative mt-0.5 flex size-5 shrink-0 items-center justify-center">
      {#if isAnimating}
        <span
          class={cn('absolute inset-0 rounded-full opacity-30 animate-ping', dotClass(runtimeSummary.tone))}
        ></span>
      {/if}
      {#if runtimeSummary.tone === 'info'}
        <LoaderCircle class="text-sky-400 size-4 animate-spin" />
      {:else}
        <span class={cn('relative size-2 rounded-full', dotClass(runtimeSummary.tone))}></span>
      {/if}
    </div>

    <div class="min-w-0 flex-1 space-y-0.5">
      <div class="flex items-baseline gap-2">
        <span class={cn('text-xs font-semibold', labelClass(runtimeSummary.tone))}>
          {runtimeSummary.label}
        </span>
        {#if ticket.startedAt && !ticket.completedAt}
          <span class="text-muted-foreground text-[10px]">
            {formatRelativeTime(ticket.startedAt)}
          </span>
        {/if}
        {#if ticket.completedAt}
          <span class="text-muted-foreground text-[10px]">
            {formatRelativeTime(ticket.completedAt)}
          </span>
        {/if}
      </div>
      <p class="text-muted-foreground text-[11px] leading-normal">
        {runtimeSummary.description}
      </p>
    </div>
  </div>

  {#if ticket.retryPaused && ticket.pauseReason === 'repeated_stalls' && onResumeRetry}
    <Button
      size="sm"
      variant="outline"
      class="h-7 w-full text-[11px]"
      disabled={resumingRetry}
      onclick={() => void onResumeRetry()}
    >
      {resumingRetry ? 'Continuing...' : 'Continue Retry'}
    </Button>
  {/if}
</section>
