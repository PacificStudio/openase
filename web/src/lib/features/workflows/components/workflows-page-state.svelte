<script lang="ts">
  import { Skeleton } from '$ui/skeleton'

  let {
    loading = false,
    loadError = '',
  }: {
    loading?: boolean
    loadError?: string
  } = $props()
</script>

{#if loading}
  <div class="border-border/60 bg-card/60 flex min-h-0 flex-1 overflow-hidden rounded-xl border">
    <!-- Skeleton workflow list sidebar -->
    <div class="border-border w-52 shrink-0 border-r">
      <div class="px-3 py-2">
        <Skeleton class="h-3.5 w-16" />
      </div>
      <div class="space-y-px">
        {#each { length: 5 } as _, i}
          <div class="border-border border-b px-3 py-3">
            <div class="flex items-center gap-2">
              <Skeleton class="size-2 shrink-0 rounded-full" />
              <Skeleton class="h-3.5 {i % 2 === 0 ? 'w-24' : 'w-20'}" />
              <Skeleton class="ml-auto h-3.5 w-10 rounded-md" />
            </div>
            <div class="mt-2 flex items-center gap-1">
              <Skeleton class="h-3 w-12" />
              <Skeleton class="h-3 w-3" />
              <Skeleton class="h-3 w-10" />
            </div>
            <Skeleton class="mt-2 h-2.5 w-14" />
          </div>
        {/each}
      </div>
    </div>

    <!-- Skeleton editor area -->
    <div class="flex flex-1 flex-col overflow-hidden">
      <!-- Skeleton toolbar -->
      <div class="border-border flex items-center gap-2 border-b px-3 py-2">
        <Skeleton class="size-8 rounded-md" />
        <Skeleton class="h-3.5 w-28" />
        <Skeleton class="h-4 w-12 rounded-full" />
        <div class="border-border mx-1 h-4 w-px border-l"></div>
        <div class="flex flex-1 items-center gap-1.5">
          <Skeleton class="h-5 w-16 rounded-full" />
          <Skeleton class="h-5 w-14 rounded-full" />
          <Skeleton class="h-5 w-18 rounded-full" />
        </div>
        <div class="ml-auto flex items-center gap-1.5">
          <Skeleton class="h-8 w-12 rounded-md" />
          <Skeleton class="h-8 w-16 rounded-md" />
          <Skeleton class="h-8 w-14 rounded-md" />
          <Skeleton class="size-8 rounded-md" />
        </div>
      </div>

      <!-- Skeleton code editor content -->
      <div class="flex-1 space-y-2.5 p-4">
        <Skeleton class="h-3.5 w-[30%]" />
        <Skeleton class="h-3.5 w-[65%]" />
        <Skeleton class="h-3.5 w-[45%]" />
        <Skeleton class="h-3.5 w-[80%]" />
        <Skeleton class="h-3.5 w-[35%]" />
        <div class="h-2"></div>
        <Skeleton class="h-3.5 w-[50%]" />
        <Skeleton class="h-3.5 w-[70%]" />
        <Skeleton class="h-3.5 w-[55%]" />
        <Skeleton class="h-3.5 w-[40%]" />
        <Skeleton class="h-3.5 w-[75%]" />
        <div class="h-2"></div>
        <Skeleton class="h-3.5 w-[60%]" />
        <Skeleton class="h-3.5 w-[85%]" />
        <Skeleton class="h-3.5 w-[42%]" />
        <Skeleton class="h-3.5 w-[68%]" />
      </div>
    </div>
  </div>
{:else if loadError}
  <div
    class="border-destructive/40 bg-destructive/10 text-destructive m-4 rounded-md border px-4 py-3 text-sm"
  >
    {loadError}
  </div>
{/if}
