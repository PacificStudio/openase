<script lang="ts">
  import { Badge } from '$ui/badge'
  import AlertTriangle from '@lucide/svelte/icons/alert-triangle'
  import Bot from '@lucide/svelte/icons/bot'
  import CircleCheck from '@lucide/svelte/icons/circle-check'
  import GitPullRequest from '@lucide/svelte/icons/git-pull-request'
  import Play from '@lucide/svelte/icons/play'
  import Settings from '@lucide/svelte/icons/settings'
  import { activityEventLabel, activityEventTone } from '$lib/features/activity'
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { TicketActivityTimelineItem } from '../types'

  let { item }: { item: TicketActivityTimelineItem } = $props()

  const activityStyle = $derived.by(() => activityPresentation(item.eventType))
  const contextEntries = $derived(extractContextEntries(item))
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
    'runtime_phase',
    'runtime_control_state',
    'status',
    'state',
    'phase',
    'target_machine_name',
  ])

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

  function extractContextEntries(item: TicketActivityTimelineItem) {
    const entries: Array<{ key: string; label: string; value: string }> = []

    if (item.actor.type === 'agent' && item.actor.name && item.actor.name !== 'Unknown') {
      entries.push({ key: 'actor', label: 'Agent', value: item.actor.name })
    } else if (item.actor.type !== 'system' && item.actor.name && item.actor.name !== 'Unknown') {
      entries.push({ key: 'actor', label: 'Source', value: item.actor.name })
    }

    const machineName = stringMetadata(item.metadata, 'target_machine_name')
    if (machineName) {
      entries.push({ key: 'machine', label: 'Machine', value: machineName })
    }

    const controlState = stringMetadata(item.metadata, 'runtime_control_state')
    if (controlState && controlState !== 'active') {
      entries.push({ key: 'control', label: 'Control', value: humanizeValue(controlState) })
    }

    const runID =
      stringMetadata(item.metadata, 'current_run_id') || stringMetadata(item.metadata, 'run_id')
    if (runID) {
      entries.push({ key: 'run', label: 'Run', value: shortenIdentifier(runID) })
    }

    return entries
  }

  function extractLinks(metadata: Record<string, unknown>) {
    return Object.entries(metadata)
      .filter(
        ([key, value]) =>
          !hiddenKeys.has(key) && typeof value === 'string' && /^https?:\/\//.test(value),
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

  function humanizeValue(value: string) {
    return value.replace(/[_-]+/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase())
  }

  function shortenIdentifier(value: string) {
    return value.length > 8 ? value.slice(0, 8) : value
  }

  function stringMetadata(metadata: Record<string, unknown>, key: string) {
    const value = metadata[key]
    return typeof value === 'string' && value.trim().length > 0 ? value.trim() : ''
  }
</script>

<div
  class="bg-muted/30 border-border relative z-10 mt-0.5 flex size-6 shrink-0 items-center justify-center rounded-full border"
>
  <activityStyle.icon class={cn('size-3', activityStyle.className)} />
</div>
<div class="min-w-0 flex-1">
  <div class="space-y-2 py-1">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <span class="text-foreground text-xs font-medium">
        {humanizeEventLabel(item.title || item.eventType)}
      </span>
      <span class="text-muted-foreground text-[11px]">{formatRelativeTime(item.createdAt)}</span>
    </div>

    {#if item.bodyText}
      <p class="text-muted-foreground text-xs leading-relaxed">{item.bodyText}</p>
    {/if}

    {#if contextEntries.length > 0 || linkEntries.length > 0}
      <div class="flex flex-wrap items-center gap-2 text-[10px]">
        {#each contextEntries as entry (entry.key)}
          <Badge variant="outline" class="h-5 px-2 text-[10px]">
            {entry.label}
            {entry.value}
          </Badge>
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
      </div>
    {/if}
  </div>
</div>
