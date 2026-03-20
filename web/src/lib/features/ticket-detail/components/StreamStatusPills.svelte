<script lang="ts">
  import { Activity, Cable, Waypoints } from '@lucide/svelte'
  import { streamBadgeClass } from '$lib/features/workspace'
  import type { StreamConnectionState } from '$lib/api/sse'

  let {
    ticketStreamState = 'idle',
    activityStreamState = 'idle',
    hookStreamState = 'idle',
  }: {
    ticketStreamState?: StreamConnectionState
    activityStreamState?: StreamConnectionState
    hookStreamState?: StreamConnectionState
  } = $props()
</script>

<div class="flex flex-wrap items-center gap-2">
  <span
    class={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${streamBadgeClass(ticketStreamState)}`}
  >
    <Cable class="mr-1.5 size-3.5" />
    Ticket {ticketStreamState}
  </span>
  <span
    class={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${streamBadgeClass(activityStreamState)}`}
  >
    <Activity class="mr-1.5 size-3.5" />
    Activity {activityStreamState}
  </span>
  <span
    class={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${streamBadgeClass(hookStreamState)}`}
  >
    <Waypoints class="mr-1.5 size-3.5" />
    Hooks {hookStreamState}
  </span>
</div>
