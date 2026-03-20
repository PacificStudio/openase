<script lang="ts">
  import { Activity, Clock3, Waypoints } from '@lucide/svelte'
  import ScrollPane from '$lib/components/layout/ScrollPane.svelte'
  import SurfacePanel from '$lib/components/layout/SurfacePanel.svelte'
  import MetadataViewer from '$lib/components/metadata-viewer.svelte'
  import { formatTimestamp, type ActivityEvent } from '$lib/features/workspace'

  let {
    title = 'Execution stream',
    description = '',
    variant = 'activity',
    events = [],
    emptyMessage = 'No events yet.',
  }: {
    title?: string
    description?: string
    variant?: 'activity' | 'hooks'
    events?: ActivityEvent[]
    emptyMessage?: string
  } = $props()
</script>

<SurfacePanel class="h-full">
  {#snippet header()}
    <div>
      <div class="flex items-center gap-2 text-sm font-semibold">
      {#if variant === 'hooks'}
        <Waypoints class="size-4" />
      {:else}
        <Activity class="size-4" />
      {/if}
      <span>{title}</span>
      </div>
      <p class="text-muted-foreground mt-1 text-xs leading-5">{description}</p>
    </div>
  {/snippet}

  <ScrollPane class="max-h-[30rem] space-y-3 px-4 py-4">
    {#if events.length === 0}
      <div
        class="border-border/80 bg-muted/30 text-muted-foreground rounded-3xl border border-dashed px-4 py-6 text-sm"
      >
        {emptyMessage}
      </div>
    {:else}
      {#each events as item}
        <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
          <div class="flex items-center justify-between gap-3">
            <p class="text-muted-foreground text-xs font-semibold tracking-[0.18em] uppercase">
              {item.event_type}
            </p>
            <span class="text-muted-foreground inline-flex items-center gap-1 text-xs">
              <Clock3 class="size-3" />
              {formatTimestamp(item.created_at)}
            </span>
          </div>
          <p class="text-foreground mt-3 text-sm leading-6">
            {item.message || 'No message payload.'}
          </p>
          <MetadataViewer metadata={item.metadata} />
        </div>
      {/each}
    {/if}
  </ScrollPane>
</SurfacePanel>
