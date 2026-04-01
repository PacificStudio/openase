<script lang="ts">
  import ChevronDown from '@lucide/svelte/icons/chevron-down'
  import ChevronUp from '@lucide/svelte/icons/chevron-up'
  import RotateCcw from '@lucide/svelte/icons/rotate-ccw'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
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
      <div class="border-border flex items-center justify-between gap-3 border-b px-4 py-3">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2 text-sm">
            <span class="font-medium">Attempt {attemptNumber}</span>
            <Badge variant="outline" class={`h-5 px-2 text-[10px] ${badgeClass(tone)}`}>
              {summary}
            </Badge>
          </div>
          {#if latestItem}
            <div class="text-muted-foreground mt-1 flex flex-wrap items-center gap-2 text-[11px]">
              <span>{items.length} event{items.length === 1 ? '' : 's'}</span>
              <span>{formatRelativeTime(latestItem.createdAt)}</span>
            </div>
          {/if}
        </div>

        <Button
          size="icon-xs"
          variant="ghost"
          aria-label={collapsed
            ? `Expand attempt ${attemptNumber}`
            : `Collapse attempt ${attemptNumber}`}
          onclick={() => onToggle?.()}
        >
          {#if collapsed}
            <ChevronDown class="size-3.5" />
          {:else}
            <ChevronUp class="size-3.5" />
          {/if}
        </Button>
      </div>

      <div class="px-4 py-4">
        {#if collapsed}
          <p class="text-muted-foreground text-sm italic">Attempt collapsed.</p>
        {:else}
          <div class="space-y-4">
            {#each items as item (item.id)}
              <div class="flex gap-4">
                <TicketTimelineActivityItem {item} />
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </article>
  </div>
</div>
