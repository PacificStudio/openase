<script lang="ts">
  import { cn, formatBytes, formatRelativeTime } from '$lib/utils'
  import type { MemorySnapshot } from '../types'

  let {
    memory,
    class: className = '',
  }: {
    memory: MemorySnapshot | null
    class?: string
  } = $props()

  function usagePercent(snapshot: MemorySnapshot | null): number {
    if (!snapshot || snapshot.sys_bytes <= 0) return 0
    return Math.max(0, Math.min(100, (snapshot.heap_inuse_bytes / snapshot.sys_bytes) * 100))
  }
</script>

<div class={cn('border-border bg-card rounded-md border', className)}>
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <h3 class="text-foreground text-sm font-medium">Process Memory</h3>
    <span class="text-muted-foreground text-xs">
      {memory ? formatRelativeTime(memory.observed_at) : 'Waiting for sample'}
    </span>
  </div>

  {#if memory}
    {@const liveHeapPercent = usagePercent(memory)}
    <div class="space-y-4 p-4">
      <div>
        <div class="text-muted-foreground text-xs">Heap in use / reserved from OS</div>
        <div class="mt-2 flex items-end justify-between gap-4">
          <div>
            <p class="text-foreground text-xl font-semibold">
              {formatBytes(memory.heap_inuse_bytes)}
            </p>
            <p class="text-muted-foreground text-xs">of {formatBytes(memory.sys_bytes)} reserved</p>
          </div>
          <div class="text-right">
            <p class="text-foreground text-sm font-medium">{liveHeapPercent.toFixed(1)}%</p>
            <p class="text-muted-foreground text-xs">live heap pressure</p>
          </div>
        </div>
        <div class="bg-muted mt-3 h-2 rounded-full">
          <div
            class="h-full rounded-full bg-emerald-500 transition-[width]"
            style={`width: ${liveHeapPercent}%`}
          ></div>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <div class="bg-muted/40 rounded-md px-3 py-2">
          <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">Heap Idle</div>
          <div class="text-foreground mt-1 text-sm font-medium">
            {formatBytes(memory.heap_idle_bytes)}
          </div>
        </div>
        <div class="bg-muted/40 rounded-md px-3 py-2">
          <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">Next GC</div>
          <div class="text-foreground mt-1 text-sm font-medium">
            {formatBytes(memory.next_gc_bytes)}
          </div>
        </div>
        <div class="bg-muted/40 rounded-md px-3 py-2">
          <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
            Total Alloc
          </div>
          <div class="text-foreground mt-1 text-sm font-medium">
            {formatBytes(memory.total_alloc_bytes)}
          </div>
        </div>
        <div class="bg-muted/40 rounded-md px-3 py-2">
          <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">GC Cycles</div>
          <div class="text-foreground mt-1 text-sm font-medium">
            {memory.gc_cycles.toLocaleString()}
          </div>
        </div>
      </div>

      <div class="text-muted-foreground flex items-center justify-between text-xs">
        <span>{memory.goroutines.toLocaleString()} goroutines</span>
        <span>{formatBytes(memory.stack_inuse_bytes)} stack in use</span>
      </div>
    </div>
  {:else}
    <div class="text-muted-foreground px-4 py-8 text-center text-xs">
      No process memory sample available yet.
    </div>
  {/if}
</div>
