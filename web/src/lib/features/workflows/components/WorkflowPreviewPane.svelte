<script lang="ts">
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import type { TicketStatus, Workflow } from '$lib/features/workspace'

  let {
    selectedWorkflow = null,
    statuses = [],
  }: {
    selectedWorkflow?: Workflow | null
    statuses?: TicketStatus[]
  } = $props()

  function statusName(statusId?: string | null) {
    if (!statusId) {
      return 'None'
    }

    return statuses.find((status) => status.id === statusId)?.name ?? 'Unknown'
  }
</script>

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <CardTitle>Preview pane</CardTitle>
    <CardDescription>
      Keep the current workflow contract visible while editing harness content and runtime limits.
    </CardDescription>
  </CardHeader>

  <CardContent class="space-y-4">
    {#if selectedWorkflow}
      <div class="flex flex-wrap items-center gap-2">
        <Badge variant="secondary">{selectedWorkflow.type}</Badge>
        <Badge variant="outline">v{selectedWorkflow.version}</Badge>
        <Badge variant={selectedWorkflow.is_active ? 'secondary' : 'outline'}>
          {selectedWorkflow.is_active ? 'active' : 'paused'}
        </Badge>
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
          <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Pickup lane</p>
          <p class="mt-2 text-sm font-semibold">{statusName(selectedWorkflow.pickup_status_id)}</p>
        </div>
        <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
          <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Finish lane</p>
          <p class="mt-2 text-sm font-semibold">{statusName(selectedWorkflow.finish_status_id)}</p>
        </div>
      </div>

      <div class="grid gap-3 sm:grid-cols-3">
        <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
          <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Concurrency</p>
          <p class="mt-2 text-lg font-semibold">{selectedWorkflow.max_concurrent}</p>
        </div>
        <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
          <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Retries</p>
          <p class="mt-2 text-lg font-semibold">{selectedWorkflow.max_retry_attempts}</p>
        </div>
        <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
          <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Timeout</p>
          <p class="mt-2 text-lg font-semibold">{selectedWorkflow.timeout_minutes}m</p>
        </div>
      </div>

      <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
        <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Harness path</p>
        <p class="mt-2 font-mono text-xs">{selectedWorkflow.harness_path}</p>
      </div>
    {:else}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
      >
        Select a workflow to preview its lane bindings and execution settings.
      </div>
    {/if}
  </CardContent>
</Card>
