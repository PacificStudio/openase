<script lang="ts">
  import { Badge } from '$ui/badge'
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { MachineItem } from '../types'

  let {
    machines,
    selectedId = '',
    emptyMessage = 'No machines match the current filter.',
    onSelect,
  }: {
    machines: MachineItem[]
    selectedId?: string
    emptyMessage?: string
    onSelect?: (machineId: string) => void
  } = $props()

  const statusColors: Record<string, string> = {
    online: 'bg-emerald-500',
    degraded: 'bg-amber-500',
    offline: 'bg-rose-500',
    maintenance: 'bg-slate-500',
  }

  function statusColor(status: string) {
    return statusColors[status] ?? 'bg-slate-500'
  }

  function transportOf(machine: MachineItem) {
    const transport = machine.resources.transport
    return typeof transport === 'string' && transport.trim() ? transport : null
  }
</script>

<div class="space-y-2">
  {#if machines.length === 0}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border border-dashed px-4 py-8 text-center text-sm"
    >
      {emptyMessage}
    </div>
  {:else}
    {#each machines as machine (machine.id)}
      <button
        type="button"
        class={cn(
          'border-border bg-card w-full rounded-xl border p-4 text-left transition-colors',
          machine.id === selectedId
            ? 'border-primary bg-primary/5 ring-primary/30 ring-1'
            : 'hover:bg-muted/40',
        )}
        onclick={() => onSelect?.(machine.id)}
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <span class={cn('size-2 rounded-full', statusColor(machine.status))}></span>
              <span class="text-foreground truncate font-medium">{machine.name}</span>
              {#if machine.host === 'local'}
                <Badge variant="secondary" class="text-[10px]">local</Badge>
              {/if}
            </div>
            <p class="text-muted-foreground mt-1 truncate font-mono text-xs">{machine.host}</p>
          </div>
          <Badge variant="outline" class="capitalize">{machine.status}</Badge>
        </div>

        <div class="mt-3 flex flex-wrap gap-1.5">
          {#if machine.labels.length === 0}
            <span class="text-muted-foreground text-xs">No labels</span>
          {:else}
            {#each machine.labels as label (label)}
              <Badge variant="secondary" class="text-[10px]">{label}</Badge>
            {/each}
          {/if}
        </div>

        <div class="text-muted-foreground mt-3 flex items-center justify-between text-xs">
          <span>{transportOf(machine) ?? 'No transport yet'}</span>
          <span>
            {#if machine.last_heartbeat_at}
              {formatRelativeTime(machine.last_heartbeat_at)}
            {:else}
              never checked
            {/if}
          </span>
        </div>
      </button>
    {/each}
  {/if}
</div>
