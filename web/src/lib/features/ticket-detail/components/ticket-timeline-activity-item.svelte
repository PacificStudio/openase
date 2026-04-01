<script lang="ts">
  import AlertTriangle from '@lucide/svelte/icons/alert-triangle'
  import Bot from '@lucide/svelte/icons/bot'
  import CircleCheck from '@lucide/svelte/icons/circle-check'
  import GitPullRequest from '@lucide/svelte/icons/git-pull-request'
  import Play from '@lucide/svelte/icons/play'
  import Settings from '@lucide/svelte/icons/settings'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { activityEventLabel, activityEventTone } from '$lib/features/activity'
  import type { TicketActivityTimelineItem } from '../types'

  let { item }: { item: TicketActivityTimelineItem } = $props()
  const activityStyle = $derived.by(() => activityPresentation(item.eventType))
  const statusChips = $derived(extractStatusChips(item.metadata))
  const linkEntries = $derived(extractLinks(item.metadata))

  const hiddenKeys = new Set([
    'event_type',
    'stream',
    'run_id',
    'current_run_id',
    'agent_id',
    'agent_run_id',
    'ticket_id',
    'workflow_id',
    'provider_id',
  ])

  const statusKeys = new Set(['runtime_phase', 'runtime_control_state', 'status', 'state', 'phase'])

  function activityPresentation(eventType: string) {
    if (eventType.startsWith('pr.')) {
      return { icon: GitPullRequest, className: 'text-green-500' }
    }
    if (eventType.startsWith('agent.')) {
      return { icon: Bot, className: 'text-blue-500' }
    }
    if (eventType === 'hook.started') {
      return { icon: Play, className: 'text-amber-500' }
    }
    switch (activityEventTone(eventType)) {
      case 'success':
        return { icon: CircleCheck, className: 'text-emerald-500' }
      case 'warning':
        return { icon: Play, className: 'text-amber-500' }
      case 'danger':
        return { icon: AlertTriangle, className: 'text-red-500' }
      case 'info':
        return { icon: Settings, className: 'text-sky-500' }
      default:
        return { icon: Settings, className: 'text-muted-foreground' }
    }
  }

  function extractStatusChips(metadata: Record<string, unknown>) {
    return Object.entries(metadata)
      .filter(
        ([key, value]) =>
          statusKeys.has(key) && typeof value === 'string' && value.trim().length > 0,
      )
      .map(([key, value]) => ({ key, label: String(value).replace(/[_-]+/g, ' ') }))
  }

  function extractLinks(metadata: Record<string, unknown>) {
    return Object.entries(metadata)
      .filter(
        ([key, value]) =>
          !hiddenKeys.has(key) &&
          !statusKeys.has(key) &&
          typeof value === 'string' &&
          /^https?:\/\//.test(value),
      )
      .slice(0, 2)
      .map(([key, value]) => ({
        label: key.replace(/[_-]+/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
        url: String(value),
      }))
  }

  function humanizeEventLabel(value: string) {
    return activityEventLabel(value)
  }
</script>

<div
  class="bg-muted/30 border-border relative z-10 mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full border"
>
  <activityStyle.icon class={cn('size-3', activityStyle.className)} />
</div>
<div class="min-w-0 flex-1">
  <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5 py-1 text-xs">
    <span class="text-foreground font-medium">
      {humanizeEventLabel(item.title || item.eventType)}
    </span>
    {#each statusChips as chip (chip.key)}
      <span
        class="bg-muted text-muted-foreground rounded-full px-1.5 py-0.5 text-[10px] font-medium"
      >
        {chip.label}
      </span>
    {/each}
    {#each linkEntries as link (link.url)}
      <a
        class="text-primary text-[10px] hover:underline"
        href={link.url}
        target="_blank"
        rel="noreferrer"
      >
        {link.label}
      </a>
    {/each}
    <span class="text-muted-foreground">{formatRelativeTime(item.createdAt)}</span>
  </div>
</div>
