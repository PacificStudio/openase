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

  function metadataEntries(metadata: Record<string, unknown>) {
    return Object.entries(metadata)
      .filter(([key, value]) => key !== 'event_type' && key !== 'stream' && isScalarValue(value))
      .slice(0, 4)
      .map(([key, value]) => ({
        key,
        label: humanizeLabel(key),
        value: String(value),
        isUrl: typeof value === 'string' && /^https?:\/\//.test(value),
      }))
  }

  function isScalarValue(value: unknown): value is string | number | boolean {
    return typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean'
  }

  function humanizeEventLabel(value: string) {
    return activityEventLabel(value)
  }

  function humanizeLabel(value: string) {
    return value.replace(/[_-]+/g, ' ').replace(/\b\w/g, (char) => char.toUpperCase())
  }
</script>

<div
  class="bg-muted/30 border-border relative z-10 mt-1 flex size-8 shrink-0 items-center justify-center rounded-full border"
>
  <activityStyle.icon class={cn('size-4', activityStyle.className)} />
</div>
<div class="min-w-0 flex-1">
  <div class="border-border/70 bg-muted/15 rounded-xl border border-dashed px-4 py-3">
    <div class="flex flex-wrap items-center gap-2 text-xs">
      <span class="font-medium">{humanizeEventLabel(item.title || item.eventType)}</span>
      <span class="text-muted-foreground">{formatRelativeTime(item.createdAt)}</span>
    </div>
    <p class="text-muted-foreground mt-2 text-sm leading-6 whitespace-pre-wrap">
      {item.bodyText || humanizeEventLabel(item.eventType)}
    </p>
    {#if metadataEntries(item.metadata).length > 0}
      <div class="mt-3 flex flex-wrap gap-2">
        {#each metadataEntries(item.metadata) as entry (entry.key)}
          {#if entry.isUrl}
            <a
              class="bg-background text-muted-foreground hover:text-foreground rounded-full border px-2.5 py-1 text-[11px]"
              href={entry.value}
              target="_blank"
              rel="noreferrer"
            >
              {entry.label}
            </a>
          {:else}
            <span
              class="bg-background text-muted-foreground rounded-full border px-2.5 py-1 text-[11px]"
            >
              {entry.label}: {entry.value}
            </span>
          {/if}
        {/each}
      </div>
    {/if}
  </div>
</div>
