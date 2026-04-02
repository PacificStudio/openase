<div class="border-border/60 bg-card/60 mb-3 flex items-center gap-5 rounded-xl border px-4 py-3">
  <div class="bg-muted h-4 w-16 animate-pulse rounded"></div>
  <div class="bg-muted h-4 w-20 animate-pulse rounded"></div>
</div>

<div class="space-y-2">
  {#each { length: 3 } as _}
    <div class="border-border/60 bg-card/60 flex items-center gap-3 rounded-xl border px-4 py-3">
      <div class="bg-muted h-5 w-9 shrink-0 animate-pulse rounded-full"></div>
      <div class="min-w-0 flex-1 space-y-2">
        <div class="flex items-center gap-2">
          <div class="bg-muted h-4 w-32 animate-pulse rounded"></div>
          <div class="bg-muted h-3.5 w-24 animate-pulse rounded"></div>
        </div>
        <div class="flex items-center gap-3">
          <div class="bg-muted h-3 w-20 animate-pulse rounded"></div>
          <div class="bg-muted h-3 w-16 animate-pulse rounded"></div>
        </div>
      </div>
      <div class="flex shrink-0 items-center gap-1">
        <div class="bg-muted size-6 animate-pulse rounded"></div>
        <div class="bg-muted size-6 animate-pulse rounded"></div>
        <div class="bg-muted size-6 animate-pulse rounded"></div>
      </div>
    </div>
  {/each}
</div>
