<script lang="ts">
  import { Activity } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
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

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <div class="flex items-center justify-between gap-3">
      <div>
        <CardTitle class="flex items-center gap-2">
          <Activity class="size-4" />
          <span>Activity feed</span>
        </CardTitle>
        <CardDescription>Recent events for {selectedAgentName.toLowerCase()}.</CardDescription>
      </div>
      <Badge class={streamBadgeClass(activityStreamState)}>{activityStreamState}</Badge>
    </div>
  </CardHeader>

  <CardContent class="space-y-3">
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
  </CardContent>
</Card>
