<script lang="ts">
  import type { StatusPayload } from '$lib/api/contracts'

  let { stages }: { stages: StatusPayload['stages'] } = $props()

  function hasUnlimitedCapacity(maxActiveRuns: number | null | undefined) {
    return maxActiveRuns == null
  }
</script>

<div class="space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-4">
  <div>
    <h3 class="text-sm font-semibold text-slate-900">Stage Concurrency</h3>
    <p class="mt-1 text-sm text-slate-600">
      Shared stage semaphores apply across every workflow that picks up from statuses inside the
      same stage.
    </p>
  </div>

  <div class="space-y-2">
    {#each stages as stage}
      <div
        class="flex items-center justify-between rounded-lg border border-slate-200 bg-white px-3 py-2"
      >
        <div>
          <p class="text-sm font-medium text-slate-900">{stage.name}</p>
          <p class="text-xs text-slate-500">
            {#if hasUnlimitedCapacity(stage.max_active_runs)}
              {stage.active_runs} active now, unlimited capacity
            {:else}
              {stage.active_runs} active now, capacity {stage.max_active_runs}
            {/if}
          </p>
        </div>
        <div class="text-sm font-medium text-slate-700">
          {#if hasUnlimitedCapacity(stage.max_active_runs)}
            {stage.active_runs}
          {:else}
            {stage.active_runs}/{stage.max_active_runs}
          {/if}
        </div>
      </div>
    {/each}
  </div>
</div>
