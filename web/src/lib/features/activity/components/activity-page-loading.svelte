<script lang="ts">
  import { Skeleton } from '$ui/skeleton'
</script>

<div class="space-y-6">
  <div>
    <Skeleton class="mb-3 h-3.5 w-12" />
    <div class="space-y-1">
      {#each { length: 6 } as _, i}
        <div class="flex items-start gap-3 px-3 py-2.5">
          <Skeleton class="mt-0.5 size-6 shrink-0 rounded-full" />
          <div class="min-w-0 flex-1 space-y-1.5">
            <div class="flex items-center gap-2">
              <Skeleton class="h-4 w-20 rounded" />
              <Skeleton class="h-4 {i % 3 === 0 ? 'w-3/4' : i % 3 === 1 ? 'w-1/2' : 'w-2/3'}" />
            </div>
            <div class="flex items-center gap-1.5">
              <Skeleton class="h-3 w-16" />
              <Skeleton class="h-3 w-12" />
            </div>
          </div>
        </div>
      {/each}
    </div>
  </div>
</div>
