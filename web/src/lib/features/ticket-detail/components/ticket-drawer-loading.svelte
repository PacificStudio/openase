<script lang="ts">
  import { Skeleton } from '$ui/skeleton'
</script>

<div class="flex flex-col overflow-hidden">
  <div class="border-border flex shrink-0 items-center gap-3 border-b px-5 py-3">
    <div class="bg-muted h-5 w-5 animate-pulse rounded"></div>
    <div class="bg-muted h-5 w-16 animate-pulse rounded"></div>
    <div class="bg-muted h-4 w-48 animate-pulse rounded"></div>
    <div class="ml-auto flex items-center gap-2">
      <div class="bg-muted h-6 w-20 animate-pulse rounded-full"></div>
      <div class="bg-muted h-6 w-6 animate-pulse rounded"></div>
    </div>
  </div>

  <div class="flex flex-1 overflow-hidden">
    <div class="flex min-w-0 flex-1 flex-col gap-4 p-5">
      {#each { length: 4 } as _}
        <div class="flex gap-3">
          <div class="bg-muted size-7 shrink-0 animate-pulse rounded-full"></div>
          <div class="flex-1 space-y-2">
            <div class="flex items-center gap-2">
              <div class="bg-muted h-3 w-20 animate-pulse rounded"></div>
              <div class="bg-muted h-3 w-16 animate-pulse rounded"></div>
            </div>
            <div class="bg-muted h-16 w-full animate-pulse rounded-lg"></div>
          </div>
        </div>
      {/each}
    </div>

    <div class="border-border w-80 shrink-0 border-l">
      <div class="flex flex-col gap-5 px-5 py-5">
        <div class="bg-muted h-16 w-full animate-pulse rounded-lg"></div>
        <div class="bg-border h-px"></div>

        <div class="space-y-3">
          <Skeleton class="h-3 w-12" />
          {#each { length: 5 } as _}
            <div class="flex items-center gap-3">
              <Skeleton class="h-3 w-20" />
              <Skeleton class="h-3 w-24" />
            </div>
          {/each}
        </div>

        <div class="bg-border h-px"></div>

        <div class="space-y-2">
          <Skeleton class="h-3 w-16" />
          <Skeleton class="h-8 w-full rounded-md" />
        </div>

        <div class="bg-border h-px"></div>

        <div class="space-y-2">
          <Skeleton class="h-3 w-24" />
          <Skeleton class="h-8 w-full rounded-md" />
        </div>

        <div class="bg-border h-px"></div>

        <div class="space-y-2">
          <Skeleton class="h-3 w-20" />
          <Skeleton class="h-8 w-full rounded-md" />
        </div>
      </div>
    </div>
  </div>
</div>
