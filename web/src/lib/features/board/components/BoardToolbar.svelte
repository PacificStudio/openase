<script lang="ts">
  import { FolderKanban } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card'
  import { streamBadgeClass } from '$lib/features/workspace'
  import type { StreamConnectionState } from '$lib/api/sse'

  let {
    projectName = '',
    laneCount = 0,
    ticketCount = 0,
    busy = false,
    error = '',
    streamState = 'idle',
  }: {
    projectName?: string
    laneCount?: number
    ticketCount?: number
    busy?: boolean
    error?: string
    streamState?: StreamConnectionState
  } = $props()
</script>

<CardHeader class="border-border/70 bg-muted/20 border-b">
  <div class="flex flex-wrap items-center justify-between gap-4">
    <div>
      <CardTitle class="flex items-center gap-2">
        <FolderKanban class="size-4" />
        <span>Board</span>
      </CardTitle>
      <CardDescription>
        {projectName
          ? `${projectName} routes tickets across custom statuses.`
          : 'Select a project to load the board.'}
      </CardDescription>
    </div>

    <div class="flex flex-wrap gap-2">
      <Badge variant="outline">{laneCount} lanes</Badge>
      <Badge variant="outline">{ticketCount} tickets</Badge>
      <Badge class={streamBadgeClass(streamState)}>{streamState}</Badge>
      {#if busy}
        <Badge variant="outline">loading</Badge>
      {/if}
    </div>
  </div>

  {#if error}
    <div
      class="text-destructive border-destructive/25 bg-destructive/10 mt-4 rounded-2xl border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {/if}
</CardHeader>
