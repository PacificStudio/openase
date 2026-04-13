<script lang="ts">
  import { cn, formatBytes, formatRelativeTime } from '$lib/utils'
  import { i18nStore } from '$lib/i18n/store.svelte'
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
    <h3 class="text-foreground text-sm font-medium">
      {i18nStore.t('dashboard.memorySnapshot.labels.heading')}
    </h3>
    <span class="text-muted-foreground text-xs">
      {memory
        ? formatRelativeTime(memory.observed_at)
        : i18nStore.t('dashboard.memorySnapshot.messages.waiting')}
    </span>
  </div>

  {#if memory}
    {@const liveHeapPercent = usagePercent(memory)}
    <div class="space-y-4 p-4">
      <div>
        <div class="text-muted-foreground text-xs">
          {i18nStore.t('dashboard.memorySnapshot.labels.heapUsage')}
        </div>
        <div class="mt-2 flex items-end justify-between gap-4">
          <div>
            <p class="text-foreground text-xl font-semibold">
              {formatBytes(memory.heap_inuse_bytes)}
            </p>
            <p class="text-muted-foreground text-xs">
              {i18nStore.t('dashboard.memorySnapshot.labels.reserved', {
                amount: formatBytes(memory.sys_bytes),
              })}
            </p>
          </div>
          <div class="text-right">
            <p class="text-foreground text-sm font-medium">{liveHeapPercent.toFixed(1)}%</p>
            <p class="text-muted-foreground text-xs">
              {i18nStore.t('dashboard.memorySnapshot.labels.liveHeap')}
            </p>
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
          <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
            {i18nStore.t('dashboard.memorySnapshot.labels.heapIdle')}
          </div>
          <div class="text-foreground mt-1 text-sm font-medium">
            {formatBytes(memory.heap_idle_bytes)}
          </div>
        </div>
        <div class="bg-muted/40 rounded-md px-3 py-2">
          <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
            {i18nStore.t('dashboard.memorySnapshot.labels.nextGc')}
          </div>
          <div class="text-foreground mt-1 text-sm font-medium">
            {formatBytes(memory.next_gc_bytes)}
          </div>
        </div>
        <div class="bg-muted/40 rounded-md px-3 py-2">
          <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
            {i18nStore.t('dashboard.memorySnapshot.labels.totalAlloc')}
          </div>
          <div class="text-foreground mt-1 text-sm font-medium">
            {formatBytes(memory.total_alloc_bytes)}
          </div>
        </div>
        <div class="bg-muted/40 rounded-md px-3 py-2">
          <div class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
            {i18nStore.t('dashboard.memorySnapshot.labels.gcCycles')}
          </div>
          <div class="text-foreground mt-1 text-sm font-medium">
            {memory.gc_cycles.toLocaleString()}
          </div>
        </div>
      </div>

      <div class="text-muted-foreground flex items-center justify-between text-xs">
        <span>
          {memory.goroutines.toLocaleString()}{' '}
          {i18nStore.t('dashboard.memorySnapshot.labels.goroutines')}
        </span>
        <span>
          {formatBytes(memory.stack_inuse_bytes)}{' '}
          {i18nStore.t('dashboard.memorySnapshot.labels.stackInUse')}
        </span>
      </div>
    </div>
  {:else}
    <div class="text-muted-foreground px-4 py-8 text-center text-xs">
      {i18nStore.t('dashboard.memorySnapshot.messages.noSample')}
    </div>
  {/if}
</div>
