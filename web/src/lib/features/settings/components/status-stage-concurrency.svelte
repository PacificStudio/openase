<script lang="ts">
  import type { StatusPayload } from '$lib/api/contracts'
  import * as Card from '$ui/card'

  let { stages }: { stages: StatusPayload['stages'] } = $props()

  function hasUnlimitedCapacity(maxActiveRuns: number | null | undefined) {
    return maxActiveRuns == null
  }
</script>

<Card.Root class="gap-4">
  <Card.Header class="gap-1">
    <Card.Title>Stage Concurrency</Card.Title>
    <Card.Description>
      Shared stage semaphores apply across every workflow that picks up from statuses inside the
      same stage.
    </Card.Description>
  </Card.Header>

  <Card.Content class="space-y-2">
    {#each stages as stage}
      <div
        class="bg-muted/40 border-border/70 flex items-center justify-between rounded-xl border px-3 py-3"
      >
        <div class="min-w-0">
          <p class="text-foreground text-sm font-medium">{stage.name}</p>
          <p class="text-muted-foreground mt-1 text-xs">
            {#if hasUnlimitedCapacity(stage.max_active_runs)}
              {stage.active_runs} active now, unlimited capacity
            {:else}
              {stage.active_runs} active now, capacity {stage.max_active_runs}
            {/if}
          </p>
        </div>
        <div
          class="bg-background text-foreground border-border/70 shrink-0 rounded-full border px-2.5 py-1 text-sm font-medium"
        >
          {#if hasUnlimitedCapacity(stage.max_active_runs)}
            {stage.active_runs}
          {:else}
            {stage.active_runs}/{stage.max_active_runs}
          {/if}
        </div>
      </div>
    {/each}
  </Card.Content>
</Card.Root>
