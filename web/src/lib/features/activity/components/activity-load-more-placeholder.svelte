<script lang="ts">
  import { Skeleton } from '$ui/skeleton'
</script>

<div
  class="border-border/60 bg-muted/20 animate-fade-in-up rounded-xl border border-dashed px-3 py-3"
  aria-live="polite"
>
  <div
    class="text-muted-foreground mb-3 flex items-center gap-2 text-[11px] font-medium tracking-[0.22em] uppercase"
  >
    <span class="bg-primary/70 animate-pulse-dot size-2 rounded-full"></span>
    Loading older activity
  </div>
  <div class="space-y-1">
    {#each { length: 3 } as _, i}
      <div
        class="animate-stagger flex items-start gap-3 rounded-md px-1 py-2"
        style="--stagger-index: {i}"
      >
        <Skeleton class="mt-0.5 size-6 shrink-0 rounded-full" />
        <div class="min-w-0 flex-1 space-y-1.5">
          <div class="flex items-center gap-2">
            <Skeleton class="h-4 w-20 rounded" />
            <Skeleton class="h-4 {i === 0 ? 'w-3/5' : i === 1 ? 'w-2/3' : 'w-1/2'} rounded" />
          </div>
          <div class="flex items-center gap-1.5">
            <Skeleton class="h-3 w-16 rounded" />
            <Skeleton class="h-3 w-12 rounded" />
          </div>
        </div>
      </div>
    {/each}
  </div>
</div>
