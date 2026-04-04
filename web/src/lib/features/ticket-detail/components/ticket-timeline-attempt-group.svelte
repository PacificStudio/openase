<script lang="ts">
  import ChevronDown from '@lucide/svelte/icons/chevron-down'
  import ChevronUp from '@lucide/svelte/icons/chevron-up'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'
  import { Badge } from '$ui/badge'
  import { formatRelativeTime } from '$lib/utils'
  import TicketTimelineActivityItem from './ticket-timeline-activity-item.svelte'
  import type { TicketActivityTimelineItem } from '../types'

  let {
    attemptNumber,
    summary,
    tone = 'neutral',
    items,
    collapsed = false,
    onToggle,
    showConnector = false,
  }: {
    attemptNumber: number
    summary: string
    tone?: 'info' | 'success' | 'warning' | 'danger' | 'neutral'
    items: TicketActivityTimelineItem[]
    collapsed?: boolean
    onToggle?: () => void
    showConnector?: boolean
  } = $props()

  const latestItem = $derived(items.at(-1) ?? items[0])

  function badgeClass(tone: 'info' | 'success' | 'warning' | 'danger' | 'neutral') {
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
</script>

<div class="relative flex gap-4 pb-6">
  {#if showConnector}
    <div class="bg-border absolute top-10 bottom-0 left-4 w-px"></div>
  {/if}

  <div
    class="bg-background border-border relative z-10 mt-1 flex size-8 shrink-0 items-center justify-center rounded-full border"
  >
    <RotateCcw class="text-foreground size-4" />
  </div>

  <div class="min-w-0 flex-1">
    <article class="border-border bg-background rounded-xl border shadow-sm">
      <button
        type="button"
        class="flex w-full items-center gap-2 px-4 py-2.5 text-left"
        aria-label={collapsed
          ? `Expand attempt ${attemptNumber}`
          : `Collapse attempt ${attemptNumber}`}
        onclick={() => onToggle?.()}
      >
        <span class="text-sm font-medium">Attempt {attemptNumber}</span>
        <Badge variant="outline" class={`h-5 px-2 text-[10px] ${badgeClass(tone)}`}>
          {summary}
        </Badge>
        {#if latestItem}
          <span class="text-muted-foreground text-[11px]">
            {items.length} event{items.length === 1 ? '' : 's'}
          </span>
          <span class="text-muted-foreground/50 text-[11px]">&middot;</span>
          <span class="text-muted-foreground text-[11px]">
            {formatRelativeTime(latestItem.createdAt)}
          </span>
        {/if}
        <span class="ml-auto shrink-0">
          {#if collapsed}
            <ChevronDown class="text-muted-foreground size-3.5" />
          {:else}
            <ChevronUp class="text-muted-foreground size-3.5" />
          {/if}
        </span>
      </button>

      {#if !collapsed}
        <div class="border-border border-t px-4 py-4">
          <div class="space-y-4">
            {#each items as item (item.id)}
              <div class="flex gap-4">
                <TicketTimelineActivityItem {item} />
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </article>
  </div>
</div>
