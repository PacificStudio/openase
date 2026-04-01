<script lang="ts">
  import { Badge } from '$ui/badge'
  import { formatRelativeTime } from '$lib/utils'
  import type { TicketDetail } from '../types'

  let { ticket }: { ticket: TicketDetail } = $props()

  type RuntimeTone = 'neutral' | 'info' | 'success' | 'warning' | 'danger'

  const runtimeSummary = $derived.by(() => summarizeRuntime(ticket))
  const phaseLabel = $derived(humanizeStatusValue(ticket.assignedAgent?.runtimePhase))
  const controlLabel = $derived(humanizeStatusValue(ticket.assignedAgent?.runtimeControlState))

  function summarizeRuntime(ticket: TicketDetail) {
    if (ticket.completedAt) {
      return {
        label: 'Completed',
        tone: 'success' as RuntimeTone,
        description: 'This ticket has completed execution.',
      }
    }

    if (ticket.assignedAgent?.runtimeControlState === 'paused') {
      return {
        label: 'Paused',
        tone: 'warning' as RuntimeTone,
        description: 'Execution is paused and waiting to be resumed.',
      }
    }

    switch (ticket.assignedAgent?.runtimePhase) {
      case 'failed':
        return {
          label: 'Failed',
          tone: 'danger' as RuntimeTone,
          description: 'The latest runtime attempt failed and needs attention.',
        }
      case 'launching':
        return {
          label: 'Running',
          tone: 'info' as RuntimeTone,
          description: 'The agent has the ticket and is still launching the runtime.',
        }
      case 'ready':
      case 'executing':
        return {
          label: 'Running',
          tone: 'success' as RuntimeTone,
          description: 'The agent runtime is live and can execute work for this ticket.',
        }
      default:
        if (ticket.assignedAgent) {
          return {
            label: 'Assigned',
            tone: 'neutral' as RuntimeTone,
            description: 'An agent is bound to this ticket, but no active runtime is reported.',
          }
        }
        return {
          label: 'Waiting',
          tone: 'neutral' as RuntimeTone,
          description: 'No active agent runtime is attached to this ticket yet.',
        }
    }
  }

  function badgeClass(tone: RuntimeTone) {
    switch (tone) {
      case 'info':
        return 'border-sky-500/30 bg-sky-500/10 text-sky-600'
      case 'success':
        return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-600'
      case 'warning':
        return 'border-amber-500/30 bg-amber-500/10 text-amber-600'
      case 'danger':
        return 'border-red-500/30 bg-red-500/10 text-red-600'
      default:
        return ''
    }
  }

  function humanizeStatusValue(value?: string | null) {
    if (!value) return ''
    return value
      .trim()
      .replace(/[_-]+/g, ' ')
      .replace(/\b\w/g, (char) => char.toUpperCase())
  }
</script>

<section class="space-y-3">
  <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
    Current State
  </span>

  <div class="border-border bg-muted/20 space-y-3 rounded-xl border px-3 py-3">
    <div class="flex flex-wrap items-center gap-2">
      <Badge
        variant="outline"
        class={`h-5 px-2 text-[10px] font-medium ${badgeClass(runtimeSummary.tone)}`}
      >
        State {runtimeSummary.label}
      </Badge>
      {#if phaseLabel}
        <Badge variant="outline" class="h-5 px-2 text-[10px]">Phase {phaseLabel}</Badge>
      {/if}
      {#if controlLabel && ticket.assignedAgent?.runtimeControlState !== 'active'}
        <Badge variant="outline" class="h-5 px-2 text-[10px]">Control {controlLabel}</Badge>
      {/if}
    </div>

    <p class="text-muted-foreground text-xs leading-relaxed">{runtimeSummary.description}</p>

    <div class="grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-2 text-xs">
      <span class="text-muted-foreground">Agent</span>
      <span class="text-foreground break-words">
        {ticket.assignedAgent ? ticket.assignedAgent.name : 'Unassigned'}
      </span>

      {#if ticket.assignedAgent}
        <span class="text-muted-foreground">Provider</span>
        <span class="text-foreground break-words">{ticket.assignedAgent.provider}</span>
      {/if}

      {#if ticket.startedAt}
        <span class="text-muted-foreground">Started</span>
        <span class="text-foreground">{formatRelativeTime(ticket.startedAt)}</span>
      {/if}

      {#if ticket.completedAt}
        <span class="text-muted-foreground">Completed</span>
        <span class="text-foreground">{formatRelativeTime(ticket.completedAt)}</span>
      {/if}
    </div>
  </div>
</section>
