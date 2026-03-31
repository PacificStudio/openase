<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { formatRelativeTime } from '$lib/utils'
  import { RefreshCw } from '@lucide/svelte'
  import type { MachineItem, MachineSnapshot } from '../types'

  let {
    machine,
    snapshot,
    loading = false,
    refreshing = false,
    onRefresh,
  }: {
    machine: MachineItem | null
    snapshot: MachineSnapshot | null
    loading?: boolean
    refreshing?: boolean
    onRefresh?: () => void
  } = $props()
</script>

<div class="flex items-center justify-between gap-3">
  <div>
    <h3 class="text-foreground text-sm font-semibold">Health snapshot</h3>
    <p class="text-muted-foreground mt-1 text-xs">
      {#if snapshot?.checkedAt}
        Snapshot collected {formatRelativeTime(snapshot.checkedAt)} and reflects detected machine state.
      {:else if machine?.last_heartbeat_at}
        Last heartbeat {formatRelativeTime(machine.last_heartbeat_at)}.
      {:else}
        No heartbeat has been recorded yet.
      {/if}
    </p>
  </div>
  <div class="flex items-center gap-2">
    {#if loading || refreshing}
      <Badge variant="outline">{refreshing ? 'Running checks…' : 'Refreshing…'}</Badge>
    {/if}
    <Button
      variant="outline"
      size="sm"
      class="gap-1.5"
      onclick={onRefresh}
      disabled={loading || refreshing}
    >
      <RefreshCw class="size-3.5" />
      {refreshing ? 'Running checks…' : 'Run checks'}
    </Button>
  </div>
</div>
