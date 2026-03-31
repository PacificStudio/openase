<script lang="ts">
  import type { StatusPayload } from '$lib/api/contracts'
  import * as Card from '$ui/card'

  let { statuses }: { statuses: StatusPayload['statuses'] } = $props()

  function hasUnlimitedCapacity(maxActiveRuns: number | null | undefined) {
    return maxActiveRuns == null
  }
</script>

<Card.Root class="gap-4">
  <Card.Header class="gap-1">
    <Card.Title>Status Concurrency</Card.Title>
    <Card.Description>
      Status-level semaphores apply directly to the pickup status that a workflow matches.
    </Card.Description>
  </Card.Header>

  <Card.Content class="space-y-2">
    {#each statuses.filter((status) => status.max_active_runs != null) as status}
      <div
        class="bg-muted/40 border-border/70 flex items-center justify-between rounded-xl border px-3 py-3"
      >
        <div class="min-w-0">
          <p class="text-foreground text-sm font-medium">{status.name}</p>
          <p class="text-muted-foreground mt-1 text-xs">
            {#if hasUnlimitedCapacity(status.max_active_runs)}
              {status.active_runs} active now, unlimited capacity
            {:else}
              {status.active_runs} active now, capacity {status.max_active_runs}
            {/if}
          </p>
        </div>
        <div
          class="bg-background text-foreground border-border/70 shrink-0 rounded-full border px-2.5 py-1 text-sm font-medium"
        >
          {#if hasUnlimitedCapacity(status.max_active_runs)}
            {status.active_runs}
          {:else}
            {status.active_runs}/{status.max_active_runs}
          {/if}
        </div>
      </div>
    {/each}
  </Card.Content>
</Card.Root>
