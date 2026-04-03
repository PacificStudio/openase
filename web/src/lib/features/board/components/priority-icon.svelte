<script lang="ts">
  import { cn } from '$lib/utils'
  import { formatBoardPriorityLabel, type BoardPriority } from '../priority'

  let {
    priority,
    class: className = '',
  }: {
    priority: BoardPriority
    class?: string
  } = $props()

  const active = 'var(--color-muted-foreground)'
  const inactiveOpacity = 0.2
</script>

<svg
  viewBox="0 0 16 16"
  fill="none"
  xmlns="http://www.w3.org/2000/svg"
  class={cn('size-4 shrink-0', className)}
  aria-label={`Priority: ${formatBoardPriorityLabel(priority).toLowerCase()}`}
>
  {#if priority === ''}
    <rect x="2.5" y="7" width="11" height="2" rx="1" fill={active} />
  {:else if priority === 'urgent'}
    <!-- Exclamation mark -->
    <rect x="6.75" y="3" width="2.5" height="6.5" rx="1.25" fill={active} />
    <circle cx="8" cy="12.25" r="1.25" fill={active} />
  {:else}
    <!-- Signal bars: 3 bars, increasing height -->
    {@const levels = priority === 'high' ? 3 : priority === 'medium' ? 2 : 1}
    <rect
      x="2.5"
      y="10"
      width="2.5"
      height="4"
      rx="0.5"
      fill={active}
      opacity={levels >= 1 ? 1 : inactiveOpacity}
    />
    <rect
      x="6.75"
      y="7"
      width="2.5"
      height="7"
      rx="0.5"
      fill={active}
      opacity={levels >= 2 ? 1 : inactiveOpacity}
    />
    <rect
      x="11"
      y="4"
      width="2.5"
      height="10"
      rx="0.5"
      fill={active}
      opacity={levels >= 3 ? 1 : inactiveOpacity}
    />
  {/if}
</svg>
