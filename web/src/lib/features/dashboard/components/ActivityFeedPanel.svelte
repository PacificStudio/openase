<script lang="ts">
  import { Activity } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import ScrollPane from '$lib/components/layout/ScrollPane.svelte'
  import SurfacePanel from '$lib/components/layout/SurfacePanel.svelte'
  import { formatTimestamp, streamBadgeClass, type ActivityEvent } from '$lib/features/workspace'
  import type { StreamConnectionState } from '$lib/api/sse'

  let {
    activityEvents = [],
    activityStreamState = 'idle',
    selectedAgentName = 'All agents',
  }: {
    activityEvents?: ActivityEvent[]
    activityStreamState?: StreamConnectionState
    selectedAgentName?: string
  } = $props()
</script>

<SurfacePanel class="h-full">
  {#snippet header()}
    <div class="flex items-center justify-between gap-3">
      <div>
        <div class="flex items-center gap-2 text-sm font-semibold">
          <Activity class="size-4" />
          <span>Activity feed</span>
        </div>
        <p class="text-muted-foreground mt-1 text-xs">
          Recent events for {selectedAgentName.toLowerCase()}.
        </p>
      </div>
      <Badge class={streamBadgeClass(activityStreamState)}>{activityStreamState}</Badge>
    </div>
  {/snippet}

  <ScrollPane class="max-h-[34rem] space-y-3 px-4 py-4">
    {#if activityEvents.length === 0}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
      >
        No activity events yet for the current selection.
      </div>
    {:else}
      {#each activityEvents as event}
        <div class="border-border/70 bg-background/60 rounded-3xl border px-4 py-4">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <p class="text-sm font-semibold">{event.event_type}</p>
            <Badge variant="outline">{formatTimestamp(event.created_at)}</Badge>
          </div>
          <p class="mt-2 text-sm leading-6">{event.message || 'No event message provided.'}</p>
        </div>
      {/each}
    {/if}
  </ScrollPane>
</SurfacePanel>
